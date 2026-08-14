package server

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/integrations"
)

const (
	maxCAMTImportBytes     = 2 << 20
	maxCAMTImportFormBytes = maxCAMTImportBytes + (256 << 10)
	camtImportPreviewTTL   = 15 * time.Minute
	maxCAMTImportPreviews  = 24
)

type camtImportPreview struct {
	TenantSlug     string
	Period         string
	Filename       string
	FileDigest     string
	SourceVersion  string
	CreatedAt      time.Time
	Payments       []integrations.Payment
	ParserErrors   []integrations.RecordError
	RowFingerprint string
}

type paymentImportUnitView struct {
	Label     string
	Reference string
}

type paymentImportRowView struct {
	Decision      string
	DecisionClass string
	Reference     string
	UnitLabel     string
	Status        string
	Amount        string
	Reason        string
}

type paymentImportPreviewView struct {
	Token          string
	Filename       string
	SourceVersion  string
	CreatedAt      string
	Assigned       int
	Unclear        int
	Rejected       int
	Rows           []paymentImportRowView
	CanApply       bool
	AlreadyApplied bool
	Changed        bool
}

func (a *app) paymentImportPage(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, _, _, _, ok := a.buildingSettingsContext(w, ac)
	if !ok {
		return
	}
	period := normalizePaymentImportPeriod(r.URL.Query().Get("period"))
	if period == "" {
		period = time.Now().In(time.Local).Format("2006-01")
	}
	units := a.unitStore.ListTenant(tenant.Slug)
	candidates, err := unitPaymentReferenceCandidates(tenant.Slug, period, units, nil)
	if err != nil {
		logError("payment import references failed", err, "tenant", tenant.Slug)
		http.Error(w, "Zahlungsreferenzen konnten nicht vorbereitet werden.", http.StatusInternalServerError)
		return
	}
	references := make([]paymentImportUnitView, 0, len(candidates))
	for _, candidate := range candidates {
		references = append(references, paymentImportUnitView{
			Label:     candidate.UnitLabel,
			Reference: candidate.Reference,
		})
	}

	resultMessage, resultOK := paymentImportResultMessage(r.URL.Query())
	var previewView *paymentImportPreviewView
	if token := strings.TrimSpace(r.URL.Query().Get("preview")); token != "" {
		if preview, found := a.paymentImportPreview(token, tenant.Slug); found {
			view := a.paymentImportPreviewView(token, preview, candidates)
			previewView = &view
		} else if resultMessage == "" {
			resultMessage = "Diese Vorschau ist abgelaufen. Bitte die Datei erneut auswählen."
			resultOK = false
		}
	}

	a.render(w, "paymentImport", a.withBase(ac, map[string]any{
		"Title":                   "Zahlungen aus Bankdatei",
		"ActivePage":              "settings",
		"PaymentImportPeriod":     period,
		"PaymentImportReferences": references,
		"HasPaymentImportUnits":   len(references) > 0,
		"PaymentImportPreview":    previewView,
		"PaymentImportMsg":        resultMessage,
		"PaymentImportOK":         resultOK,
		"MaxCAMTImportSize":       formatBytes(maxCAMTImportBytes),
	}))
}

func (a *app) previewPaymentImport(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, _, _, _, ok := a.buildingSettingsContext(w, ac)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxCAMTImportFormBytes)
	if err := r.ParseMultipartForm(maxCAMTImportBytes); err != nil {
		a.redirectPaymentImport(w, r, "", "", "invalid")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	period := normalizePaymentImportPeriod(r.FormValue("period"))
	if period == "" {
		a.redirectPaymentImport(w, r, "", "", "period")
		return
	}
	headers := r.MultipartForm.File["camt_file"]
	if len(headers) != 1 || strings.ToLower(filepath.Ext(headers[0].Filename)) != ".xml" {
		a.redirectPaymentImport(w, r, period, "", "invalid")
		return
	}
	file, err := headers[0].Open()
	if err != nil {
		a.redirectPaymentImport(w, r, period, "", "invalid")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxCAMTImportBytes+1))
	if err != nil || len(data) == 0 || len(data) > maxCAMTImportBytes ||
		!strings.HasPrefix(strings.TrimSpace(string(data)), "<") {
		a.redirectPaymentImport(w, r, period, "", "invalid")
		return
	}

	digestBytes := sha256.Sum256(data)
	fileDigest := hex.EncodeToString(digestBytes[:])
	source := integrations.Source{
		TenantSlug: tenant.Slug,
		Format:     integrations.FormatCAMT053,
		Filename:   filepath.Base(headers[0].Filename),
		Imported:   time.Now().UTC(),
	}
	parsed, err := (integrations.CAMT053Adapter{}).ParsePayments(r.Context(), source, strings.NewReader(string(data)))
	if err != nil {
		a.redirectPaymentImport(w, r, period, "", "invalid")
		return
	}
	payments := sanitizedImportPayments(parsed.Payments)
	parserErrors := sanitizedImportErrors(parsed.Report.Errors)
	candidates, err := unitPaymentReferenceCandidates(tenant.Slug, period, a.unitStore.ListTenant(tenant.Slug), nil)
	if err != nil {
		logError("payment import preview failed", err, "tenant", tenant.Slug)
		a.redirectPaymentImport(w, r, period, "", "error")
		return
	}
	report := reconcileImportedPaymentsWithUnitStatus(payments, candidates)
	preview := camtImportPreview{
		TenantSlug:     tenant.Slug,
		Period:         period,
		Filename:       filepath.Base(headers[0].Filename),
		FileDigest:     fileDigest,
		SourceVersion:  parsed.Report.Source.Version,
		CreatedAt:      time.Now().UTC(),
		Payments:       payments,
		ParserErrors:   parserErrors,
		RowFingerprint: paymentImportFingerprint(tenant.Slug, period, fileDigest, report, parserErrors),
	}
	token, err := a.storePaymentImportPreview(preview)
	if err != nil {
		logError("payment import preview token failed", err, "tenant", tenant.Slug)
		a.redirectPaymentImport(w, r, period, "", "error")
		return
	}
	a.redirectPaymentImport(w, r, period, token, "")
}

func (a *app) applyPaymentImport(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actorEmail, role, _, ok := a.buildingSettingsContext(w, ac)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		a.redirectPaymentImport(w, r, "", "", "invalid")
		return
	}
	token := strings.TrimSpace(r.FormValue("preview_token"))

	a.paymentImportMu.Lock()
	defer a.paymentImportMu.Unlock()
	a.cleanupPaymentImportPreviewsLocked(time.Now().UTC())
	preview, found := a.paymentImportPreviews[token]
	if !found || preview.TenantSlug != tenant.Slug {
		a.redirectPaymentImport(w, r, "", "", "expired")
		return
	}
	targetID := paymentImportTargetID(preview.FileDigest)
	if a.paymentImportAlreadyApplied(tenant.Slug, preview.FileDigest) {
		delete(a.paymentImportPreviews, token)
		a.redirectPaymentImport(w, r, preview.Period, "", "already")
		return
	}
	candidates, err := unitPaymentReferenceCandidates(tenant.Slug, preview.Period, a.unitStore.ListTenant(tenant.Slug), nil)
	if err != nil {
		logError("payment import apply references failed", err, "tenant", tenant.Slug)
		a.redirectPaymentImport(w, r, preview.Period, token, "error")
		return
	}
	report := reconcileImportedPaymentsWithUnitStatus(preview.Payments, candidates)
	fingerprint := paymentImportFingerprint(tenant.Slug, preview.Period, preview.FileDigest, report, preview.ParserErrors)
	if fingerprint != preview.RowFingerprint {
		delete(a.paymentImportPreviews, token)
		a.redirectPaymentImport(w, r, preview.Period, "", "changed")
		return
	}
	report, err = a.applyImportedPaymentsToUnitStatuses(
		preview.Payments,
		candidates,
		actorEmail,
		role,
		paymentImportAuditMeta{
			TargetID:      targetID,
			SourceVersion: preview.SourceVersion,
			FileDigest:    preview.FileDigest,
		},
	)
	if err != nil {
		logError("payment import apply failed", err, "tenant", tenant.Slug)
		a.redirectPaymentImport(w, r, preview.Period, token, "error")
		return
	}
	if err := a.recordPaymentImportLedger(tenant.Slug, actorEmail, preview, report); err != nil {
		logError("payment import ledger failed", err, "tenant", tenant.Slug)
		a.redirectPaymentImport(w, r, preview.Period, token, "error")
		return
	}
	delete(a.paymentImportPreviews, token)
	query := url.Values{
		"period":   {preview.Period},
		"result":   {"applied"},
		"assigned": {strconv.Itoa(report.Assigned)},
		"changed":  {strconv.Itoa(report.Changed)},
		"unclear":  {strconv.Itoa(report.Unclear)},
		"rejected": {strconv.Itoa(report.Rejected + len(preview.ParserErrors))},
	}
	http.Redirect(w, r, "/app/settings/payments/import?"+query.Encode(), http.StatusSeeOther)
}

func (a *app) paymentImportPreviewView(token string, preview camtImportPreview, candidates []unitPaymentReferenceCandidate) paymentImportPreviewView {
	report := reconcileImportedPaymentsWithUnitStatus(preview.Payments, candidates)
	fingerprint := paymentImportFingerprint(preview.TenantSlug, preview.Period, preview.FileDigest, report, preview.ParserErrors)
	view := paymentImportPreviewView{
		Token:         token,
		Filename:      preview.Filename,
		SourceVersion: firstNonEmpty(preview.SourceVersion, "Nicht unterstützt"),
		CreatedAt:     formatLocalDateTime(preview.CreatedAt),
		Assigned:      report.Assigned,
		Unclear:       report.Unclear,
		Rejected:      report.Rejected + len(preview.ParserErrors),
		Changed:       fingerprint != preview.RowFingerprint,
	}
	view.AlreadyApplied = a.paymentImportAlreadyApplied(preview.TenantSlug, preview.FileDigest)
	for _, row := range report.Rows {
		view.Rows = append(view.Rows, paymentImportRowViewFrom(row))
	}
	for _, recordErr := range preview.ParserErrors {
		view.Rows = append(view.Rows, paymentImportRowView{
			Decision:      "Abgelehnt",
			DecisionClass: "danger",
			Reference:     "—",
			UnitLabel:     "—",
			Amount:        "—",
			Reason:        paymentImportErrorLabel(recordErr),
		})
	}
	view.CanApply = view.Assigned > 0 && !view.AlreadyApplied && !view.Changed
	return view
}

func paymentImportRowViewFrom(row unitPaymentImportRow) paymentImportRowView {
	view := paymentImportRowView{
		Reference: firstNonEmpty(row.Reference, "—"),
		UnitLabel: firstNonEmpty(row.UnitLabel, "—"),
		Amount:    paymentImportAmountLabel(row.Amount),
		Reason:    row.Reason,
	}
	switch row.Decision {
	case unitPaymentImportAssigned:
		view.Decision = "Zuordnen"
		view.DecisionClass = "ok"
		view.Status = unitPaymentStatusLabel(row.Status)
	case unitPaymentImportUnclear:
		view.Decision = "Prüfen"
		view.DecisionClass = "warn"
	default:
		view.Decision = "Ablehnen"
		view.DecisionClass = "danger"
	}
	return view
}

func paymentImportAmountLabel(amount integrations.MoneyAmount) string {
	currency := strings.ToUpper(strings.TrimSpace(amount.Currency))
	if currency == "" {
		return "—"
	}
	if currency == "EUR" {
		return formatEUR(float64(amount.Cents) / 100)
	}
	return formatDecimal(float64(amount.Cents)/100, 2) + " " + currency
}

func paymentImportErrorLabel(recordErr integrations.RecordError) string {
	switch recordErr.Field {
	case "namespace":
		return "Dieses camt.053-Profil wird nicht unterstützt."
	case "credit_debit_indicator":
		return "Nur eingehende Zahlungen werden berücksichtigt."
	case "amount":
		return "Der Betrag ist ungültig."
	case "booking_date":
		return "Das Buchungsdatum fehlt oder ist ungültig."
	default:
		return "Der Datensatz erfüllt das unterstützte camt.053-Profil nicht."
	}
}

func sanitizedImportPayments(payments []integrations.Payment) []integrations.Payment {
	out := make([]integrations.Payment, 0, len(payments))
	for _, payment := range payments {
		payment.DebtorName = ""
		payment.DebtorIBAN = ""
		payment.RemittanceLines = nil
		out = append(out, payment)
	}
	return out
}

func sanitizedImportErrors(errors []integrations.RecordError) []integrations.RecordError {
	out := make([]integrations.RecordError, 0, len(errors))
	for _, recordErr := range errors {
		recordErr.RecordID = ""
		recordErr.Message = paymentImportErrorLabel(recordErr)
		out = append(out, recordErr)
	}
	return out
}

func paymentImportFingerprint(tenantSlug, period, fileDigest string, report unitPaymentImportReport, parserErrors []integrations.RecordError) string {
	hash := sha256.New()
	_, _ = fmt.Fprintf(hash, "%s\x1f%s\x1f%s\x1f", normalizeSlug(tenantSlug), period, fileDigest)
	for _, row := range report.Rows {
		_, _ = fmt.Fprintf(hash, "%s\x1f%s\x1f%s\x1f%s\x1f%s\x1f%d\x1f%s\x1e",
			row.Decision, row.Reference, row.UnitID, row.Status, row.Reason, row.Amount.Cents, row.Amount.Currency)
	}
	for _, recordErr := range parserErrors {
		_, _ = fmt.Fprintf(hash, "%s\x1f%s\x1e", recordErr.Field, recordErr.Message)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func paymentImportTargetID(fileDigest string) string {
	return string(integrations.FormatCAMT053) + ":" + strings.TrimSpace(fileDigest)
}

func (a *app) paymentImportAlreadyApplied(tenantSlug, fileDigest string) bool {
	tenantSlug = normalizeSlug(tenantSlug)
	fileDigest = strings.TrimSpace(fileDigest)
	if a != nil && a.db != nil && tenantSlug != "" && fileDigest != "" {
		var exists int
		err := a.db.QueryRow(
			`SELECT EXISTS(
				SELECT 1 FROM integration_imports
				WHERE tenant_slug = ? AND format = ? AND file_digest = ?
			)`,
			tenantSlug, string(integrations.FormatCAMT053), fileDigest,
		).Scan(&exists)
		if err == nil && exists == 1 {
			return true
		}
		if err != nil {
			logError("payment import ledger lookup failed", err, "tenant", tenantSlug)
		}
	}
	targetID := paymentImportTargetID(fileDigest)
	return a != nil && a.auditStore != nil && a.auditStore.HasTarget(tenantSlug, auditActionIntegrationImport, targetID)
}

func (a *app) recordPaymentImportLedger(tenantSlug, actorEmail string, preview camtImportPreview, report unitPaymentImportReport) error {
	if a == nil || a.db == nil {
		return nil
	}
	_, err := a.db.Exec(
		`INSERT INTO integration_imports(
			tenant_slug, format, file_digest, source_version, applied_at, applied_by,
			assigned, changed, unclear, rejected
		) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(tenant_slug, format, file_digest) DO NOTHING`,
		normalizeSlug(tenantSlug),
		string(integrations.FormatCAMT053),
		strings.TrimSpace(preview.FileDigest),
		strings.TrimSpace(preview.SourceVersion),
		time.Now().UTC().Format(time.RFC3339),
		normalizeEmail(actorEmail),
		report.Assigned,
		report.Changed,
		report.Unclear,
		report.Rejected+len(preview.ParserErrors),
	)
	return err
}

func normalizePaymentImportPeriod(raw string) string {
	raw = strings.TrimSpace(raw)
	parsed, err := time.Parse("2006-01", raw)
	if err != nil || parsed.Year() < 2000 || parsed.Year() > 2100 {
		return ""
	}
	return parsed.Format("2006-01")
}

func (a *app) storePaymentImportPreview(preview camtImportPreview) (string, error) {
	var tokenBytes [18]byte
	if _, err := rand.Read(tokenBytes[:]); err != nil {
		return "", err
	}
	token := hex.EncodeToString(tokenBytes[:])
	a.paymentImportMu.Lock()
	defer a.paymentImportMu.Unlock()
	now := time.Now().UTC()
	a.cleanupPaymentImportPreviewsLocked(now)
	if a.paymentImportPreviews == nil {
		a.paymentImportPreviews = map[string]camtImportPreview{}
	}
	if len(a.paymentImportPreviews) >= maxCAMTImportPreviews {
		oldestToken := ""
		var oldest time.Time
		for candidateToken, candidate := range a.paymentImportPreviews {
			if oldestToken == "" || candidate.CreatedAt.Before(oldest) {
				oldestToken = candidateToken
				oldest = candidate.CreatedAt
			}
		}
		delete(a.paymentImportPreviews, oldestToken)
	}
	a.paymentImportPreviews[token] = preview
	return token, nil
}

func (a *app) paymentImportPreview(token, tenantSlug string) (camtImportPreview, bool) {
	a.paymentImportMu.Lock()
	defer a.paymentImportMu.Unlock()
	a.cleanupPaymentImportPreviewsLocked(time.Now().UTC())
	preview, ok := a.paymentImportPreviews[token]
	if !ok || preview.TenantSlug != normalizeSlug(tenantSlug) {
		return camtImportPreview{}, false
	}
	return preview, true
}

func (a *app) cleanupPaymentImportPreviewsLocked(now time.Time) {
	for token, preview := range a.paymentImportPreviews {
		if preview.CreatedAt.IsZero() || now.Sub(preview.CreatedAt) > camtImportPreviewTTL {
			delete(a.paymentImportPreviews, token)
		}
	}
}

func paymentImportResultMessage(values url.Values) (string, bool) {
	switch values.Get("result") {
	case "applied":
		return fmt.Sprintf(
			"Import übernommen: %s eindeutige Treffer, %s Status geändert, %s zu prüfen, %s abgelehnt.",
			values.Get("assigned"), values.Get("changed"), values.Get("unclear"), values.Get("rejected"),
		), true
	case "already":
		return "Diese Bankdatei wurde bereits übernommen. Es wurden keine Status oder Audit-Einträge dupliziert.", true
	case "changed":
		return "Einheiten oder Referenzen haben sich seit der Vorschau geändert. Bitte eine neue Vorschau erstellen.", false
	case "expired":
		return "Diese Vorschau ist abgelaufen. Bitte die Datei erneut auswählen.", false
	case "period":
		return "Bitte einen gültigen Monat zwischen 2000 und 2100 wählen.", false
	case "invalid":
		return "Bitte eine gültige camt.053-XML-Datei bis 2 MB auswählen.", false
	case "error":
		return "Der Import konnte nicht verarbeitet werden. Es wurde nichts stillschweigend übernommen.", false
	default:
		return "", false
	}
}

func (a *app) redirectPaymentImport(w http.ResponseWriter, r *http.Request, period, preview, result string) {
	query := url.Values{}
	if period != "" {
		query.Set("period", period)
	}
	if preview != "" {
		query.Set("preview", preview)
	}
	if result != "" {
		query.Set("result", result)
	}
	target := "/app/settings/payments/import"
	if encoded := query.Encode(); encoded != "" {
		target += "?" + encoded
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}
