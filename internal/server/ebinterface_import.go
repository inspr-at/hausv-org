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
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/integrations"
	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/web"
)

const (
	maxEBInterfaceImportBytes     = 4 << 20
	maxEBInterfaceImportFormBytes = maxEBInterfaceImportBytes + (256 << 10)
	ebInterfaceImportPreviewTTL   = 15 * time.Minute
	maxEBInterfaceImportPreviews  = 8
)

type ebInterfaceImportPreview struct {
	TenantSlug    string
	Filename      string
	FileDigest    string
	SourceVersion string
	CreatedAt     time.Time
	RawXML        []byte
	Invoice       integrations.Invoice
	Errors        []integrations.RecordError
	Fingerprint   string
}

type ebInterfaceImportPreviewView = web.EbInterfaceImportPreviewView

func (a *app) ebInterfaceImportPage(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if denyServiceProviderArea(w, ac.role) {
		return
	}
	resultMessage, resultOK := ebInterfaceImportResultMessage(r.URL.Query())
	var previewView *ebInterfaceImportPreviewView
	if token := strings.TrimSpace(r.URL.Query().Get("preview")); token != "" {
		if preview, found := a.ebInterfaceImportPreview(token, ac.tenant.Slug); found {
			view := a.ebInterfaceImportPreviewView(ac.tenantRef, token, preview)
			previewView = &view
		} else if resultMessage == "" {
			resultMessage = "Diese Vorschau ist abgelaufen. Bitte die XML-Datei erneut auswählen."
			resultOK = false
		}
	}
	a.renderSettingsComponent(w, r, ac.tenant.Slug, web.EbInterfaceImportPage(web.EbInterfaceImportPageData{
		Portal:                   a.settingsPortalContext(ac, "E-Rechnung einlesen", "documents"),
		EBInterfacePreview:       previewView,
		EBInterfaceImportMsg:     resultMessage,
		EBInterfaceImportOK:      resultOK,
		MaxEBInterfaceImportSize: formatBytes(maxEBInterfaceImportBytes),
	}))
}

func (a *app) previewEBInterfaceImport(w http.ResponseWriter, r *http.Request, ac authCtx) {
	r.Body = http.MaxBytesReader(w, r.Body, maxEBInterfaceImportFormBytes)
	if err := r.ParseMultipartForm(maxEBInterfaceImportBytes); err != nil {
		a.redirectEBInterfaceImport(w, r, "", "invalid")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	headers := r.MultipartForm.File["invoice_file"]
	if len(headers) != 1 || strings.ToLower(filepath.Ext(headers[0].Filename)) != ".xml" {
		a.redirectEBInterfaceImport(w, r, "", "invalid")
		return
	}
	file, err := headers[0].Open()
	if err != nil {
		a.redirectEBInterfaceImport(w, r, "", "invalid")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxEBInterfaceImportBytes+1))
	if err != nil || len(data) == 0 || len(data) > maxEBInterfaceImportBytes ||
		!strings.HasPrefix(strings.TrimSpace(string(data)), "<") {
		a.redirectEBInterfaceImport(w, r, "", "invalid")
		return
	}

	digestBytes := sha256.Sum256(data)
	digest := hex.EncodeToString(digestBytes[:])
	source := integrations.Source{
		TenantSlug: ac.tenant.Slug,
		Format:     integrations.FormatEBInterface,
		Filename:   filepath.Base(headers[0].Filename),
		Imported:   time.Now().UTC(),
	}
	parsed, err := (integrations.EBInterfaceAdapter{}).ParseInvoices(r.Context(), source, strings.NewReader(string(data)))
	if err != nil {
		a.redirectEBInterfaceImport(w, r, "", "invalid")
		return
	}
	preview := ebInterfaceImportPreview{
		TenantSlug:    ac.tenant.Slug,
		Filename:      filepath.Base(headers[0].Filename),
		FileDigest:    digest,
		SourceVersion: parsed.Report.Source.Version,
		CreatedAt:     time.Now().UTC(),
		RawXML:        append([]byte(nil), data...),
		Errors:        sanitizedEBInterfaceErrors(parsed.Report.Errors),
	}
	if len(parsed.Invoices) == 1 {
		preview.Invoice = parsed.Invoices[0]
	}
	preview.Fingerprint = ebInterfaceImportFingerprint(preview)
	token, err := a.storeEBInterfaceImportPreview(preview)
	if err != nil {
		logError("ebInterface preview token failed", err, "tenant", ac.tenant.Slug)
		a.redirectEBInterfaceImport(w, r, "", "error")
		return
	}
	a.redirectEBInterfaceImport(w, r, token, "")
}

func (a *app) storeEBInterfaceImport(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if err := r.ParseForm(); err != nil {
		a.redirectEBInterfaceImport(w, r, "", "invalid")
		return
	}
	token := strings.TrimSpace(r.FormValue("preview_token"))

	a.ebInterfaceImportMu.Lock()
	defer a.ebInterfaceImportMu.Unlock()
	a.cleanupEBInterfaceImportPreviewsLocked(time.Now().UTC())
	preview, found := a.ebInterfaceImportPreviews[token]
	if !found || preview.TenantSlug != normalizeSlug(ac.tenant.Slug) {
		a.redirectEBInterfaceImport(w, r, "", "expired")
		return
	}
	if a.ebInterfaceImportAlreadyStored(ac.tenantRef, preview.FileDigest) {
		delete(a.ebInterfaceImportPreviews, token)
		a.redirectEBInterfaceImport(w, r, "", "already")
		return
	}
	if len(preview.Errors) > 0 || preview.Invoice.InvoiceNumber == "" || len(preview.RawXML) == 0 {
		a.redirectEBInterfaceImport(w, r, token, "invalid")
		return
	}

	parsed, err := (integrations.EBInterfaceAdapter{}).ParseInvoices(r.Context(), integrations.Source{
		TenantSlug: ac.tenant.Slug,
		Format:     integrations.FormatEBInterface,
		Filename:   preview.Filename,
		Imported:   preview.CreatedAt,
	}, strings.NewReader(string(preview.RawXML)))
	if err != nil || len(parsed.Invoices) != 1 || parsed.Report.HasErrors() {
		delete(a.ebInterfaceImportPreviews, token)
		a.redirectEBInterfaceImport(w, r, "", "changed")
		return
	}
	verified := preview
	verified.Invoice = parsed.Invoices[0]
	verified.Errors = sanitizedEBInterfaceErrors(parsed.Report.Errors)
	verified.SourceVersion = parsed.Report.Source.Version
	if ebInterfaceImportFingerprint(verified) != preview.Fingerprint {
		delete(a.ebInterfaceImportPreviews, token)
		a.redirectEBInterfaceImport(w, r, "", "changed")
		return
	}
	created, err := storeEBInterfaceInvoiceDocument(a.documentStore, ac.tenantRef, verified.Invoice, ac.email, verified.RawXML, time.Now())
	if err != nil {
		logError("ebInterface document storage failed", err, "tenant", ac.tenant.Slug)
		a.redirectEBInterfaceImport(w, r, token, "error")
		return
	}

	ledgerErr := a.recordEBInterfaceImportLedger(ac.tenantRef, ac.email, verified)
	auditErr := a.appendEBInterfaceImportAudit(ac, verified, created)
	delete(a.ebInterfaceImportPreviews, token)
	if ledgerErr != nil || auditErr != nil {
		logError("ebInterface import record failed", fmt.Errorf("ledger: %v; audit: %v", ledgerErr, auditErr), "tenant", ac.tenant.Slug, "document_id", created.ID)
		a.redirectEBInterfaceImport(w, r, "", "recorded")
		return
	}
	http.Redirect(w, r, "/app/dokumente?doc=invoice-imported#document-"+url.PathEscape(created.ID), http.StatusSeeOther)
}

func (a *app) ebInterfaceImportPreviewView(tenant store.TenantRef, token string, preview ebInterfaceImportPreview) ebInterfaceImportPreviewView {
	invoice := preview.Invoice
	view := ebInterfaceImportPreviewView{
		Token:         token,
		Filename:      preview.Filename,
		SourceVersion: firstNonEmpty(preview.SourceVersion, "Nicht unterstützt"),
		CreatedAt:     formatLocalDateTime(preview.CreatedAt),
		InvoiceNumber: firstNonEmpty(invoice.InvoiceNumber, "—"),
		IssuerName:    firstNonEmpty(invoice.IssuerName, "—"),
		RecipientName: firstNonEmpty(invoice.RecipientName, "—"),
		Amount:        paymentImportAmountLabel(invoice.Amount),
		IssueDate:     ebInterfaceDateLabel(invoice.IssueDate),
		DueDate:       ebInterfaceDateLabel(invoice.DueDate),
		ServicePeriod: ebInterfaceServicePeriodLabel(invoice.ServicePeriodStart, invoice.ServicePeriodEnd),
		ErrorLabels:   ebInterfaceErrorLabels(preview.Errors),
		AlreadyStored: a.ebInterfaceImportAlreadyStored(tenant, preview.FileDigest),
	}
	view.CanStore = len(view.ErrorLabels) == 0 && !view.AlreadyStored && invoice.InvoiceNumber != "" && invoice.Amount.Cents > 0 && !invoice.IssueDate.IsZero()
	return view
}

func sanitizedEBInterfaceErrors(errors []integrations.RecordError) []integrations.RecordError {
	out := make([]integrations.RecordError, 0, len(errors))
	for _, recordErr := range errors {
		recordErr.RecordID = ""
		recordErr.Message = ebInterfaceErrorLabel(recordErr)
		out = append(out, recordErr)
	}
	return out
}

func ebInterfaceErrorLabels(errors []integrations.RecordError) []string {
	out := make([]string, 0, len(errors))
	for _, recordErr := range errors {
		out = append(out, ebInterfaceErrorLabel(recordErr))
	}
	return out
}

func ebInterfaceErrorLabel(recordErr integrations.RecordError) string {
	switch recordErr.Field {
	case "namespace":
		return "Dieses ebInterface-Profil wird noch nicht unterstützt."
	case "xml":
		return "Die XML-Struktur konnte nicht gelesen werden."
	case "invoice_number":
		return "Die Rechnungsnummer fehlt."
	case "amount":
		return "Der Bruttobetrag oder die Währung fehlt oder ist ungültig."
	case "issue_date":
		return "Das Rechnungsdatum fehlt oder ist ungültig."
	case "tenant_slug":
		return "Die Rechnung konnte keiner Liegenschaft zugeordnet werden."
	default:
		return "Die Rechnung erfüllt das unterstützte ebInterface-Profil nicht."
	}
}

func ebInterfaceImportFingerprint(preview ebInterfaceImportPreview) string {
	hash := sha256.New()
	invoice := preview.Invoice
	_, _ = fmt.Fprintf(hash, "%s\x1f%s\x1f%s\x1f%s\x1f%s\x1f%s\x1f%d\x1f%s\x1f%s\x1f%s\x1f%s\x1f%s",
		normalizeSlug(preview.TenantSlug),
		preview.FileDigest,
		preview.SourceVersion,
		invoice.InvoiceNumber,
		invoice.IssuerName,
		invoice.RecipientName,
		invoice.Amount.Cents,
		invoice.Amount.Currency,
		invoice.IssueDate.UTC().Format(time.RFC3339),
		invoice.DueDate.UTC().Format(time.RFC3339),
		invoice.ServicePeriodStart.UTC().Format(time.RFC3339),
		invoice.ServicePeriodEnd.UTC().Format(time.RFC3339),
	)
	for _, recordErr := range preview.Errors {
		_, _ = fmt.Fprintf(hash, "\x1e%s\x1f%s", recordErr.Field, recordErr.Message)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func ebInterfaceImportTargetID(fileDigest string) string {
	return string(integrations.FormatEBInterface) + ":" + strings.TrimSpace(fileDigest)
}

// ebInterfaceImportAlreadyStored is the dedup read on the TENANT lane; see
// paymentImportAlreadyApplied for why the read keys on tenant_id while the
// constraint keys on tenant_slug, and what happens to the one row where the two
// disagree.
func (a *app) ebInterfaceImportAlreadyStored(tenant store.TenantRef, fileDigest string) bool {
	fileDigest = strings.TrimSpace(fileDigest)
	if a != nil && a.tenantDB != nil && tenant.Valid() && fileDigest != "" {
		var exists bool
		err := a.tenantDB.For(tenant).QueryRow(
			`SELECT EXISTS(
				SELECT 1 FROM integration_imports
				WHERE tenant_id = $1 AND format = $2 AND file_digest = $3
			)`,
			tenant.ID, string(integrations.FormatEBInterface), fileDigest,
		).Scan(&exists)
		if err == nil && exists {
			return true
		}
		if err != nil {
			logError("ebInterface import ledger lookup failed", err, "tenant", tenant.Slug)
		}
	}
	return a != nil && a.auditStore != nil &&
		a.auditStore.HasTarget(tenant.Slug, auditActionIntegrationImport, ebInterfaceImportTargetID(fileDigest))
}

// recordEBInterfaceImportLedger writes the ledger row on the MAINTENANCE lane
// under store.HealOrphanReason, for the reason spelled out on the camt.053
// ledger: the natural conflict key can land on a row the previous release left
// without a tenant_id, and only the maintenance lane can reach that row.
func (a *app) recordEBInterfaceImportLedger(tenant store.TenantRef, actorEmail string, preview ebInterfaceImportPreview) error {
	if a == nil || a.tenantDB == nil {
		return nil
	}
	if !tenant.Valid() {
		return fmt.Errorf("ebInterface import ledger: tenant reference is not usable")
	}
	// DO NOTHING on everything EXCEPT tenant_id, for the reason spelled out on
	// the camt.053 ledger: the first application's timestamp and actor must
	// survive a re-upload, but a row left without an identity is a row the
	// tenant_id lookup above can never find, so the ledger would keep storing
	// the same invoice for ever.
	_, err := a.tenantDB.Unscoped(store.HealOrphanReason).Exec(
		`INSERT INTO integration_imports(
			tenant_id, tenant_slug, format, file_digest, source_version, applied_at, applied_by,
			assigned, changed, unclear, rejected
		) VALUES($1, $2, $3, $4, $5, $6, $7, 1, 1, 0, 0)
		ON CONFLICT(tenant_slug, format, file_digest) DO UPDATE SET
		  tenant_id=coalesce(integration_imports.tenant_id, excluded.tenant_id)`,
		tenant.ID,
		tenant.Slug,
		string(integrations.FormatEBInterface),
		strings.TrimSpace(preview.FileDigest),
		strings.TrimSpace(preview.SourceVersion),
		time.Now().UTC().Format(time.RFC3339),
		normalizeEmail(actorEmail),
	)
	return err
}

func (a *app) appendEBInterfaceImportAudit(ac authCtx, preview ebInterfaceImportPreview, created documentRecord) error {
	if a == nil || a.auditStore == nil {
		return fmt.Errorf("audit store not configured")
	}
	return a.auditStore.Append(auditEvent{
		TenantSlug: ac.tenant.Slug,
		ActorEmail: ac.email,
		ActorRole:  ac.role,
		Action:     auditActionIntegrationImport,
		TargetType: "integration",
		TargetID:   ebInterfaceImportTargetID(preview.FileDigest),
		Summary:    "E-Rechnung geschützt abgelegt",
		Details: map[string]string{
			"format":         string(integrations.FormatEBInterface),
			"source_version": strings.TrimSpace(preview.SourceVersion),
			"file_digest":    shortImportDigest(preview.FileDigest),
			"invoice_number": strings.TrimSpace(preview.Invoice.InvoiceNumber),
			"document_id":    created.ID,
			"visibility":     documentVisibilityLabel(created.Visibility),
		},
	})
}

func (a *app) storeEBInterfaceImportPreview(preview ebInterfaceImportPreview) (string, error) {
	var tokenBytes [18]byte
	if _, err := rand.Read(tokenBytes[:]); err != nil {
		return "", err
	}
	token := hex.EncodeToString(tokenBytes[:])
	a.ebInterfaceImportMu.Lock()
	defer a.ebInterfaceImportMu.Unlock()
	now := time.Now().UTC()
	a.cleanupEBInterfaceImportPreviewsLocked(now)
	if a.ebInterfaceImportPreviews == nil {
		a.ebInterfaceImportPreviews = map[string]ebInterfaceImportPreview{}
	}
	if len(a.ebInterfaceImportPreviews) >= maxEBInterfaceImportPreviews {
		oldestToken := ""
		var oldest time.Time
		for candidateToken, candidate := range a.ebInterfaceImportPreviews {
			if oldestToken == "" || candidate.CreatedAt.Before(oldest) {
				oldestToken = candidateToken
				oldest = candidate.CreatedAt
			}
		}
		delete(a.ebInterfaceImportPreviews, oldestToken)
	}
	a.ebInterfaceImportPreviews[token] = preview
	return token, nil
}

func (a *app) ebInterfaceImportPreview(token, tenantSlug string) (ebInterfaceImportPreview, bool) {
	a.ebInterfaceImportMu.Lock()
	defer a.ebInterfaceImportMu.Unlock()
	a.cleanupEBInterfaceImportPreviewsLocked(time.Now().UTC())
	preview, ok := a.ebInterfaceImportPreviews[token]
	if !ok || preview.TenantSlug != normalizeSlug(tenantSlug) {
		return ebInterfaceImportPreview{}, false
	}
	return preview, true
}

func (a *app) cleanupEBInterfaceImportPreviewsLocked(now time.Time) {
	for token, preview := range a.ebInterfaceImportPreviews {
		if preview.CreatedAt.IsZero() || now.Sub(preview.CreatedAt) > ebInterfaceImportPreviewTTL {
			delete(a.ebInterfaceImportPreviews, token)
		}
	}
}

func ebInterfaceDateLabel(value time.Time) string {
	if value.IsZero() {
		return "—"
	}
	return formatLocalDate(value)
}

func ebInterfaceServicePeriodLabel(start, end time.Time) string {
	switch {
	case !start.IsZero() && !end.IsZero():
		return formatLocalDate(start) + " – " + formatLocalDate(end)
	case !start.IsZero():
		return "ab " + formatLocalDate(start)
	case !end.IsZero():
		return "bis " + formatLocalDate(end)
	default:
		return "—"
	}
}

func ebInterfaceImportResultMessage(values url.Values) (string, bool) {
	switch values.Get("result") {
	case "already":
		return "Diese E-Rechnung wurde bereits abgelegt. Es wurde kein zweites Dokument erzeugt.", true
	case "changed":
		return "Die Vorschau konnte nicht unverändert bestätigt werden. Bitte die Datei erneut auswählen.", false
	case "expired":
		return "Diese Vorschau ist abgelaufen. Bitte die XML-Datei erneut auswählen.", false
	case "invalid":
		return "Bitte eine lesbare ebInterface-XML-Datei bis 4 MB auswählen.", false
	case "recorded":
		return "Die Rechnung wurde geschützt abgelegt, aber der technische Nachweis ist unvollständig. Bitte den Aktivitätsverlauf prüfen.", false
	case "error":
		return "Die E-Rechnung konnte nicht abgelegt werden. Bitte erneut versuchen.", false
	default:
		return "", false
	}
}

func (a *app) redirectEBInterfaceImport(w http.ResponseWriter, r *http.Request, preview, result string) {
	query := url.Values{}
	if preview != "" {
		query.Set("preview", preview)
	}
	if result != "" {
		query.Set("result", result)
	}
	target := "/app/dokumente/rechnungen/import"
	if encoded := query.Encode(); encoded != "" {
		target += "?" + encoded
	}
	if preview != "" {
		target += "#preview"
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}
