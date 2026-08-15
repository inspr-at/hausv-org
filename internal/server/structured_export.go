package server

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"mime"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/integrations"
	"github.com/inspr-at/hausv-org/internal/store"
)

const (
	structuredExportSourcePayments = "unit-payment-status"
	structuredExportSourceParking  = "parking-months"
	structuredExportPreviewTTL     = 15 * time.Minute
	maxStructuredExportPreviews    = 24
)

type structuredExportManifest struct {
	Schema                    string   `json:"schema"`
	Format                    string   `json:"format"`
	FormatVersion             string   `json:"format_version"`
	GeneratedAt               string   `json:"generated_at"`
	TenantSlug                string   `json:"tenant_slug"`
	SelectedSources           []string `json:"selected_sources"`
	RecordCount               int      `json:"record_count"`
	RejectedRecordCount       int      `json:"rejected_record_count"`
	CSVFile                   string   `json:"csv_file"`
	CSVSHA256                 string   `json:"csv_sha256"`
	TargetSystemCompatibility bool     `json:"target_system_compatibility"`
	Purpose                   string   `json:"purpose"`
}

type structuredExportPreview struct {
	TenantSlug      string
	ActorEmail      string
	CreatedAt       time.Time
	Sources         []string
	SourceCounts    map[string]int
	Accepted        int
	Rejected        int
	CSVChecksum     string
	PackageChecksum string
	Filename        string
	Package         []byte
}

type structuredExportSourceView struct {
	Value       string
	Title       string
	Description string
	Count       int
}

type structuredExportPreviewView struct {
	Token       string
	CreatedAt   string
	Sources     []structuredExportSourceView
	Accepted    int
	Rejected    int
	CSVChecksum string
	Filename    string
}

func (a *app) structuredExportPage(w http.ResponseWriter, r *http.Request, ac authCtx) {
	sources := a.structuredExportSourceViews(ac.repositories, ac)
	resultMessage, resultOK := structuredExportResultMessage(r.URL.Query().Get("result"))
	var previewView *structuredExportPreviewView
	if token := strings.TrimSpace(r.URL.Query().Get("preview")); token != "" {
		if preview, found := a.getStructuredExportPreview(token, ac.tenant.Slug, ac.email); found {
			view := structuredExportPreviewView{
				Token:       token,
				CreatedAt:   formatLocalDateTime(preview.CreatedAt),
				Sources:     selectedStructuredExportSourceViews(sources, preview.Sources, preview.SourceCounts),
				Accepted:    preview.Accepted,
				Rejected:    preview.Rejected,
				CSVChecksum: preview.CSVChecksum,
				Filename:    preview.Filename,
			}
			previewView = &view
		} else if resultMessage == "" {
			resultMessage = "Diese Vorschau ist abgelaufen. Bitte die Daten erneut auswählen."
			resultOK = false
		}
	}
	a.render(w, "structuredExport", a.withBase(ac, map[string]any{
		"Title":                   "Strukturierte Datenübergabe",
		"ActivePage":              "settings",
		"StructuredExportSources": sources,
		"StructuredExportPreview": previewView,
		"StructuredExportMsg":     resultMessage,
		"StructuredExportOK":      resultOK,
	}))
}

func (a *app) previewStructuredExport(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if err := r.ParseForm(); err != nil {
		a.redirectStructuredExport(w, r, "", "selection")
		return
	}
	sources, ok := normalizeStructuredExportSources(r.Form["source"], ac.can(capabilityPlatformAdmin))
	if !ok {
		a.redirectStructuredExport(w, r, "", "selection")
		return
	}
	records, sourceCounts := a.structuredExportRecords(ac.repositories, ac.tenant.Slug, sources)
	if len(records) == 0 {
		a.redirectStructuredExport(w, r, "", "empty")
		return
	}
	now := time.Now().UTC().Truncate(time.Second)
	pkg, manifest, filename, err := buildStructuredExportPackage(r.Context(), ac.tenant.Slug, sources, records, now)
	if err != nil {
		logError("structured export preview failed", err, "tenant", ac.tenant.Slug)
		a.redirectStructuredExport(w, r, "", "error")
		return
	}
	if manifest.RecordCount == 0 {
		a.redirectStructuredExport(w, r, "", "empty")
		return
	}
	token, err := randomToken(24)
	if err != nil {
		a.redirectStructuredExport(w, r, "", "error")
		return
	}
	packageDigest := sha256.Sum256(pkg)
	preview := structuredExportPreview{
		TenantSlug:      normalizeSlug(ac.tenant.Slug),
		ActorEmail:      normalizeEmail(ac.email),
		CreatedAt:       now,
		Sources:         append([]string(nil), sources...),
		SourceCounts:    sourceCounts,
		Accepted:        manifest.RecordCount,
		Rejected:        manifest.RejectedRecordCount,
		CSVChecksum:     manifest.CSVSHA256,
		PackageChecksum: hex.EncodeToString(packageDigest[:]),
		Filename:        filename,
		Package:         append([]byte(nil), pkg...),
	}
	a.storeStructuredExportPreview(token, preview)
	a.redirectStructuredExport(w, r, token, "")
}

func (a *app) downloadStructuredExport(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	token := strings.TrimSpace(r.FormValue("preview_token"))
	preview, found := a.takeStructuredExportPreview(token, ac.tenant.Slug, ac.email)
	if !found {
		http.Error(w, "Diese Vorschau ist abgelaufen.", http.StatusGone)
		return
	}
	if a.auditStore == nil {
		a.storeStructuredExportPreview(token, preview)
		http.Error(w, "Der Export konnte nicht protokolliert werden.", http.StatusInternalServerError)
		return
	}
	event := auditEvent{
		TenantSlug: ac.tenant.Slug,
		ActorEmail: ac.email,
		ActorRole:  ac.role,
		Action:     store.AuditActionIntegrationExport,
		TargetType: "integration",
		TargetID:   integrations.StructuredHandoffVersion,
		Summary:    "Strukturierte Rohdatenübergabe erstellt",
		Details: map[string]string{
			"format":         string(integrations.FormatManualCSV),
			"format_version": integrations.StructuredHandoffVersion,
			"sources":        strings.Join(preview.Sources, ","),
			"records":        strconv.Itoa(preview.Accepted),
			"rejected":       strconv.Itoa(preview.Rejected),
			"csv_checksum":   shortImportDigest(preview.CSVChecksum),
			"package_digest": shortImportDigest(preview.PackageChecksum),
		},
	}
	if err := a.auditStore.Append(event); err != nil {
		a.storeStructuredExportPreview(token, preview)
		logError("structured export audit failed", err, "tenant", ac.tenant.Slug)
		http.Error(w, "Der Export konnte nicht protokolliert werden.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": preview.Filename}))
	w.Header().Set("Content-Length", strconv.Itoa(len(preview.Package)))
	_, _ = w.Write(preview.Package)
}

func (a *app) structuredExportRecords(repositories requestRepositories, tenantSlug string, sources []string) ([]integrations.ExportRecord, map[string]int) {
	records := make([]integrations.ExportRecord, 0)
	counts := map[string]int{}
	for _, source := range sources {
		switch source {
		case structuredExportSourcePayments:
			units := map[string]unit{}
			for _, item := range repositories.units.List() {
				units[item.ID] = item
			}
			for _, item := range repositories.unitPayments.List() {
				label := item.UnitID
				if entry, ok := units[item.UnitID]; ok && strings.TrimSpace(entry.Label) != "" {
					label = entry.Label
				}
				records = append(records, integrations.ExportRecord{
					TenantSlug: tenantSlug,
					RecordID:   "unit-payment-status-" + item.UnitID,
					Kind:       structuredExportSourcePayments,
					Occurred:   item.UpdatedAt,
					UnitID:     item.UnitID,
					Fields: map[string]string{
						"status":            structuredExportPaymentStatus(item.Status),
						"source":            "building",
						"description":       "Zahlungsstatus " + label,
						"verification_note": "Neutrale Rohdatenübergabe ohne Zielsystem-Kompatibilitätszusage",
					},
				})
				counts[source]++
			}
		case structuredExportSourceParking:
			data := a.parkingStore.TenantData(tenantSlug)
			for _, month := range calculateParkingMonths(data, time.Now(), time.Local) {
				occurred, err := time.ParseInLocation("2006-01", month.Month, time.Local)
				if err != nil {
					continue
				}
				occurred = occurred.AddDate(0, 1, -1)
				status := "open"
				if month.Paid {
					status = "paid"
				} else if month.Overdue {
					status = "overdue"
				}
				verificationNote := "Neutrale Rohdatenübergabe ohne Zielsystem-Kompatibilitätszusage"
				if month.Partial {
					verificationNote += "; Teilmonat oder unvollständige Messdaten"
				}
				records = append(records, integrations.ExportRecord{
					TenantSlug: tenantSlug,
					RecordID:   "parking-month-" + month.Month,
					Kind:       structuredExportSourceParking,
					Occurred:   occurred,
					Reference:  month.PaymentReference,
					Amount:     integrations.MoneyAmount{Currency: "EUR", Cents: int64(math.Round(month.TotalCostValue * 100))},
					Fields: map[string]string{
						"period":            month.Month,
						"status":            status,
						"source":            "parking",
						"description":       "Parkplatz-Abrechnung " + month.MonthLabel,
						"verification_note": verificationNote,
					},
				})
				counts[source]++
			}
		}
	}
	sort.SliceStable(records, func(i, j int) bool {
		if records[i].Occurred.Equal(records[j].Occurred) {
			return records[i].RecordID < records[j].RecordID
		}
		return records[i].Occurred.Before(records[j].Occurred)
	})
	return records, counts
}

func buildStructuredExportPackage(ctx context.Context, tenantSlug string, sources []string, records []integrations.ExportRecord, generatedAt time.Time) ([]byte, structuredExportManifest, string, error) {
	var csvData bytes.Buffer
	report, err := (integrations.StructuredHandoffAdapter{}).WriteExportData(ctx, &csvData, records)
	if err != nil {
		return nil, structuredExportManifest{}, "", err
	}
	csvDigest := sha256.Sum256(csvData.Bytes())
	csvFilename := "hausv-raw-v0.csv"
	manifest := structuredExportManifest{
		Schema:                    "hausv.raw-export-manifest",
		Format:                    string(integrations.FormatManualCSV),
		FormatVersion:             integrations.StructuredHandoffVersion,
		GeneratedAt:               generatedAt.UTC().Format(time.RFC3339),
		TenantSlug:                normalizeSlug(tenantSlug),
		SelectedSources:           append([]string(nil), sources...),
		RecordCount:               report.Accepted,
		RejectedRecordCount:       report.Rejected,
		CSVFile:                   csvFilename,
		CSVSHA256:                 hex.EncodeToString(csvDigest[:]),
		TargetSystemCompatibility: false,
		Purpose:                   "Kontrollierte, neutrale Rohdatenübergabe; keine Buchungs- oder Steuerlogik.",
	}
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, structuredExportManifest{}, "", err
	}
	manifestData = append(manifestData, '\n')
	var packageData bytes.Buffer
	archive := zip.NewWriter(&packageData)
	for _, file := range []struct {
		name string
		data []byte
	}{
		{name: csvFilename, data: csvData.Bytes()},
		{name: "manifest.json", data: manifestData},
	} {
		header := &zip.FileHeader{Name: file.name, Method: zip.Deflate}
		header.SetModTime(generatedAt)
		entry, err := archive.CreateHeader(header)
		if err != nil {
			_ = archive.Close()
			return nil, structuredExportManifest{}, "", err
		}
		if _, err := entry.Write(file.data); err != nil {
			_ = archive.Close()
			return nil, structuredExportManifest{}, "", err
		}
	}
	if err := archive.Close(); err != nil {
		return nil, structuredExportManifest{}, "", err
	}
	filename := fmt.Sprintf("hausv-rohdaten-%s-%s.zip", normalizeSlug(tenantSlug), generatedAt.In(time.Local).Format("20060102-150405"))
	return packageData.Bytes(), manifest, filename, nil
}

func (a *app) structuredExportSourceViews(repositories requestRepositories, ac authCtx) []structuredExportSourceView {
	paymentCount := 0
	if repositories.unitPayments != nil {
		paymentCount = len(repositories.unitPayments.List())
	}
	sources := []structuredExportSourceView{{
		Value:       structuredExportSourcePayments,
		Title:       "Zahlungsstatus der Einheiten",
		Description: "Einheit, Status und letzter Änderungszeitpunkt",
		Count:       paymentCount,
	}}
	if ac.can(capabilityPlatformAdmin) {
		parkingCount := 0
		if a.parkingStore != nil {
			parkingCount = len(calculateParkingMonths(a.parkingStore.TenantData(ac.tenant.Slug), time.Now(), time.Local))
		}
		sources = append(sources, structuredExportSourceView{
			Value:       structuredExportSourceParking,
			Title:       "Parkplatz-Abrechnung",
			Description: "Monatswerte mit Betrag, Status und Referenz",
			Count:       parkingCount,
		})
	}
	return sources
}

func normalizeStructuredExportSources(raw []string, allowParking bool) ([]string, bool) {
	selected := map[string]bool{}
	for _, item := range raw {
		switch strings.TrimSpace(item) {
		case structuredExportSourcePayments:
			selected[structuredExportSourcePayments] = true
		case structuredExportSourceParking:
			if !allowParking {
				return nil, false
			}
			selected[structuredExportSourceParking] = true
		default:
			return nil, false
		}
	}
	out := make([]string, 0, 2)
	for _, source := range []string{structuredExportSourcePayments, structuredExportSourceParking} {
		if selected[source] {
			out = append(out, source)
		}
	}
	return out, len(out) > 0
}

func selectedStructuredExportSourceViews(all []structuredExportSourceView, selected []string, counts map[string]int) []structuredExportSourceView {
	wanted := map[string]bool{}
	for _, source := range selected {
		wanted[source] = true
	}
	out := make([]structuredExportSourceView, 0, len(selected))
	for _, source := range all {
		if wanted[source.Value] {
			source.Count = counts[source.Value]
			out = append(out, source)
		}
	}
	return out
}

func structuredExportPaymentStatus(status string) string {
	switch normalizeUnitPaymentStatus(status) {
	case unitPaymentStatusPaid:
		return "paid"
	case unitPaymentStatusPartial:
		return "partial"
	case unitPaymentStatusOverdue:
		return "overdue"
	default:
		return "open"
	}
}

func (a *app) storeStructuredExportPreview(token string, preview structuredExportPreview) {
	a.structuredExportMu.Lock()
	defer a.structuredExportMu.Unlock()
	if a.structuredExportPreviews == nil {
		a.structuredExportPreviews = map[string]structuredExportPreview{}
	}
	now := time.Now().UTC()
	for key, item := range a.structuredExportPreviews {
		if now.Sub(item.CreatedAt) > structuredExportPreviewTTL {
			delete(a.structuredExportPreviews, key)
		}
	}
	for len(a.structuredExportPreviews) >= maxStructuredExportPreviews {
		oldestKey := ""
		var oldest time.Time
		for key, item := range a.structuredExportPreviews {
			if oldestKey == "" || item.CreatedAt.Before(oldest) {
				oldestKey, oldest = key, item.CreatedAt
			}
		}
		delete(a.structuredExportPreviews, oldestKey)
	}
	a.structuredExportPreviews[token] = preview
}

func (a *app) getStructuredExportPreview(token, tenantSlug, actorEmail string) (structuredExportPreview, bool) {
	a.structuredExportMu.Lock()
	defer a.structuredExportMu.Unlock()
	preview, ok := a.structuredExportPreviews[token]
	if !ok || time.Since(preview.CreatedAt) > structuredExportPreviewTTL ||
		preview.TenantSlug != normalizeSlug(tenantSlug) || preview.ActorEmail != normalizeEmail(actorEmail) {
		return structuredExportPreview{}, false
	}
	preview.Package = append([]byte(nil), preview.Package...)
	preview.Sources = append([]string(nil), preview.Sources...)
	return preview, true
}

func (a *app) takeStructuredExportPreview(token, tenantSlug, actorEmail string) (structuredExportPreview, bool) {
	a.structuredExportMu.Lock()
	defer a.structuredExportMu.Unlock()
	preview, ok := a.structuredExportPreviews[token]
	if !ok || time.Since(preview.CreatedAt) > structuredExportPreviewTTL ||
		preview.TenantSlug != normalizeSlug(tenantSlug) || preview.ActorEmail != normalizeEmail(actorEmail) {
		return structuredExportPreview{}, false
	}
	delete(a.structuredExportPreviews, token)
	preview.Package = append([]byte(nil), preview.Package...)
	preview.Sources = append([]string(nil), preview.Sources...)
	return preview, true
}

func (a *app) redirectStructuredExport(w http.ResponseWriter, r *http.Request, token, result string) {
	query := url.Values{}
	if token != "" {
		query.Set("preview", token)
	}
	if result != "" {
		query.Set("result", result)
	}
	target := "/app/settings/data-export"
	if encoded := query.Encode(); encoded != "" {
		target += "?" + encoded
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func structuredExportResultMessage(result string) (string, bool) {
	switch result {
	case "selection":
		return "Bitte mindestens einen verfügbaren Datenbereich auswählen.", false
	case "empty":
		return "In der Auswahl sind derzeit keine exportierbaren Datensätze.", false
	case "error":
		return "Die Exportvorschau konnte nicht erstellt werden.", false
	default:
		return "", false
	}
}
