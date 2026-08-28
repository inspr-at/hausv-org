package server

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/web"
)

// annualStatementReceiptSuggester is deliberately host-local. Production does
// not install an implementation until an Austria/EU-hosted inference service
// has been approved and wired. A nil implementation therefore fails closed and
// cannot accidentally send receipt data off-host.
type annualStatementReceiptSuggester interface {
	Suggest(context.Context, annualStatementReceiptInput) (annualStatementReceiptSuggestion, error)
}

type annualStatementReceiptInput struct {
	Filename    string
	ContentType string
	Data        []byte
}

type annualStatementReceiptSuggestion struct {
	AmountCents     int64
	InvoiceDate     string
	CostTypeKey     string
	AmountCertain   bool
	DateCertain     bool
	CostTypeCertain bool
}

func (a *app) suggestAnnualStatementReceipt(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if _, _, _, _, ok := a.buildingSettingsContext(w, ac); !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		a.renderAnnualStatementPage(w, r, ac, web.AnnualStatementReceiptSuggestionView{}, "Der Beleg konnte nicht sicher geprüft werden. Es wurde nichts übernommen.", false)
		return
	}
	view, status := a.trustedAnnualStatementReceiptSuggestion(r.Context(), ac, r.FormValue("document_id"))
	if status != annualStatementReceiptSuggestionReady {
		a.renderAnnualStatementPage(w, r, ac, web.AnnualStatementReceiptSuggestionView{}, annualStatementReceiptSuggestionStatusMessage(status), false)
		return
	}
	a.renderAnnualStatementPage(w, r, ac, view, "Vorschläge erkannt. Bitte mit dem Originalbeleg vergleichen und ausdrücklich bestätigen.", true)
}

func (a *app) confirmAnnualStatementReceiptSuggestion(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actorEmail, role, _, ok := a.buildingSettingsContext(w, ac)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		a.renderAnnualStatementPage(w, r, ac, web.AnnualStatementReceiptSuggestionView{}, "Die Bestätigung konnte nicht sicher geprüft werden. Es wurde nichts übernommen.", false)
		return
	}
	view, status := a.trustedAnnualStatementReceiptSuggestion(r.Context(), ac, r.FormValue("document_id"))
	if status != annualStatementReceiptSuggestionReady ||
		r.FormValue("amount_cents") != view.AmountCents ||
		r.FormValue("invoice_date") != view.InvoiceDateValue ||
		r.FormValue("cost_type_key") != view.CostTypeKey {
		a.renderAnnualStatementPage(w, r, ac, web.AnnualStatementReceiptSuggestionView{}, "Die Bestätigung stimmt nicht mehr mit dem sicheren Vorschlag überein. Es wurde nichts übernommen.", false)
		return
	}
	periodYear, err := strconv.Atoi(strings.TrimSpace(r.FormValue("year")))
	amountCents, amountErr := strconv.ParseInt(view.AmountCents, 10, 64)
	if err != nil || amountErr != nil || ac.repositories.annualStatementReceipts == nil {
		a.renderAnnualStatementPage(w, r, ac, web.AnnualStatementReceiptSuggestionView{}, "Der Beleg konnte nicht gespeichert werden. Es wurde nichts übernommen.", false)
		return
	}
	saved, err := ac.repositories.annualStatementReceipts.Create(store.AnnualStatementReceipt{
		DocumentID: view.DocumentID, PeriodYear: periodYear, CostTypeKey: view.CostTypeKey,
		AmountCents: amountCents, InvoiceDate: view.InvoiceDateValue, CreatedBy: actorEmail,
	})
	if err != nil {
		a.renderAnnualStatementPage(w, r, ac, web.AnnualStatementReceiptSuggestionView{}, "Der Beleg konnte nicht gespeichert werden. Es wurde nichts übernommen.", false)
		return
	}
	a.recordAnnualStatementReceiptAudit(tenant, actorEmail, role, auditActionAnnualReceiptCreate, "Beleg erfasst", saved)
	query := r.URL.Query()
	query.Set("year", strconv.Itoa(periodYear))
	r.URL.RawQuery = query.Encode()
	view.Confirmed = true
	a.renderAnnualStatementPage(w, r, ac, view, "Vorschlag bestätigt und Beleg gespeichert.", true)
}

func (a *app) createAnnualStatementReceipt(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actorEmail, role, _, ok := a.buildingSettingsContext(w, ac)
	if !ok {
		return
	}
	if ac.repositories.annualStatementReceipts == nil || ac.repositories.documents == nil {
		http.Redirect(w, r, "/app/settings/annual-statement?receipt=invalid", http.StatusSeeOther)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxDocumentFormBytes)
	if strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "multipart/form-data") {
		if err := r.ParseMultipartForm(maxDocumentBytes); err != nil {
			http.Redirect(w, r, "/app/settings/annual-statement?receipt=invalid", http.StatusSeeOther)
			return
		}
	} else if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/app/settings/annual-statement?receipt=invalid", http.StatusSeeOther)
		return
	}
	periodYear, err := strconv.Atoi(strings.TrimSpace(r.FormValue("year")))
	amountCents, amountOK := parseAnnualStatementReceiptAmount(r.FormValue("amount"))
	receipt := store.AnnualStatementReceipt{
		DocumentID: strings.TrimSpace(r.FormValue("document_id")), PeriodYear: periodYear,
		CostTypeKey: strings.TrimSpace(r.FormValue("cost_type_key")), AmountCents: amountCents,
		InvoiceDate: strings.TrimSpace(r.FormValue("invoice_date")), CreatedBy: actorEmail,
	}
	var files []*multipart.FileHeader
	if r.MultipartForm != nil {
		files = r.MultipartForm.File["receipt_file"]
	}
	var headerPresent bool
	if len(files) == 1 && files[0] != nil && strings.TrimSpace(files[0].Filename) != "" && files[0].Size > 0 {
		headerPresent = true
	} else if len(files) > 1 {
		http.Redirect(w, r, annualStatementReceiptRedirect(periodYear, "invalid"), http.StatusSeeOther)
		return
	}
	if err != nil || !amountOK || (receipt.DocumentID == "") == !headerPresent || !annualStatementReceiptFieldsReferenceExisting(receipt, ac.repositories) {
		http.Redirect(w, r, annualStatementReceiptRedirect(periodYear, "invalid"), http.StatusSeeOther)
		return
	}
	var uploadedDocument store.DocumentRecord
	if headerPresent {
		header := files[0]
		if !annualStatementReceiptUploadSupported(header) {
			http.Redirect(w, r, annualStatementReceiptRedirect(periodYear, "invalid"), http.StatusSeeOther)
			return
		}
		uploadedDocument, err = ac.repositories.documents.Create(store.DocumentRecord{
			TenantSlug: tenant.Slug, Title: store.SanitizeDocumentFilename(header.Filename), Category: documentCategoryBilling,
			Visibility: documentVisibilityManagerOnly, UploadedBy: actorEmail,
		}, uploadedFileFromHeader(header), time.Now())
		if err != nil {
			http.Redirect(w, r, annualStatementReceiptRedirect(periodYear, "invalid"), http.StatusSeeOther)
			return
		}
		receipt.DocumentID = uploadedDocument.ID
	}
	saved, err := ac.repositories.annualStatementReceipts.Create(receipt)
	if err != nil {
		logError("annual statement receipt create failed", err, "tenant", tenant.Slug)
		http.Redirect(w, r, annualStatementReceiptRedirect(periodYear, "invalid"), http.StatusSeeOther)
		return
	}
	if uploadedDocument.ID != "" {
		a.recordAudit(auditEvent{
			TenantSlug: tenant.Slug, ActorEmail: actorEmail, ActorRole: role,
			Action: auditActionDocumentUpload, TargetType: "document", TargetID: uploadedDocument.ID,
			Summary: "Dokument hochgeladen",
			Details: map[string]string{"title": uploadedDocument.Title, "category": uploadedDocument.Category, "visibility": documentVisibilityLabel(uploadedDocument.Visibility), "size": formatBytes(uploadedDocument.Size), "content_type": uploadedDocument.ContentType},
		})
	}
	a.recordAnnualStatementReceiptAudit(tenant, actorEmail, role, auditActionAnnualReceiptCreate, "Beleg erfasst", saved)
	http.Redirect(w, r, annualStatementReceiptRedirect(periodYear, "created"), http.StatusSeeOther)
}

func (a *app) updateAnnualStatementReceiptAmount(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actorEmail, role, _, ok := a.buildingSettingsContext(w, ac)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil || ac.repositories.annualStatementReceipts == nil {
		http.Redirect(w, r, "/app/settings/annual-statement?receipt=invalid", http.StatusSeeOther)
		return
	}
	year, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("year")))
	amountCents, amountOK := parseAnnualStatementReceiptAmount(r.FormValue("amount"))
	if rawCents := strings.TrimSpace(r.FormValue("amount_cents")); rawCents != "" {
		var err error
		amountCents, err = strconv.ParseInt(rawCents, 10, 64)
		amountOK = err == nil && amountCents > 0
	}
	if !amountOK {
		http.Redirect(w, r, annualStatementReceiptRedirect(year, "invalid"), http.StatusSeeOther)
		return
	}
	saved, err := ac.repositories.annualStatementReceipts.UpdateAmount(strings.TrimSpace(r.FormValue("id")), amountCents, actorEmail)
	if err != nil {
		http.Redirect(w, r, annualStatementReceiptRedirect(year, "invalid"), http.StatusSeeOther)
		return
	}
	a.recordAnnualStatementReceiptAudit(tenant, actorEmail, role, auditActionAnnualReceiptAmount, "Belegbetrag korrigiert", saved)
	http.Redirect(w, r, annualStatementReceiptRedirect(saved.PeriodYear, "amount-updated"), http.StatusSeeOther)
}

func (a *app) deleteAnnualStatementReceipt(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actorEmail, role, _, ok := a.buildingSettingsContext(w, ac)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil || ac.repositories.annualStatementReceipts == nil {
		http.Redirect(w, r, "/app/settings/annual-statement?receipt=invalid", http.StatusSeeOther)
		return
	}
	year, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("year")))
	receipt, found := ac.repositories.annualStatementReceipts.Get(strings.TrimSpace(r.FormValue("id")))
	if !found {
		http.Redirect(w, r, annualStatementReceiptRedirect(year, "invalid"), http.StatusSeeOther)
		return
	}
	removed, err := ac.repositories.annualStatementReceipts.Delete(receipt.ID)
	if err != nil || !removed {
		http.Redirect(w, r, annualStatementReceiptRedirect(year, "invalid"), http.StatusSeeOther)
		return
	}
	a.recordAnnualStatementReceiptAudit(tenant, actorEmail, role, auditActionAnnualReceiptDelete, "Belegzuordnung entfernt", receipt)
	http.Redirect(w, r, annualStatementReceiptRedirect(receipt.PeriodYear, "deleted"), http.StatusSeeOther)
}

func annualStatementReceiptFieldsReferenceExisting(receipt store.AnnualStatementReceipt, repositories requestRepositories) bool {
	if receipt.PeriodYear < 1 || receipt.PeriodYear > 9999 || receipt.AmountCents <= 0 || strings.TrimSpace(receipt.CostTypeKey) == "" {
		return false
	}
	date, err := time.Parse("2006-01-02", receipt.InvoiceDate)
	if err != nil || date.Format("2006-01-02") != receipt.InvoiceDate || repositories.annualStatementPeriods == nil || repositories.annualStatementCostTypes == nil {
		return false
	}
	periodFound := false
	for _, period := range repositories.annualStatementPeriods.List() {
		periodFound = periodFound || period.Year == receipt.PeriodYear
	}
	costTypeFound := false
	for _, costType := range repositories.annualStatementCostTypes.List() {
		costTypeFound = costTypeFound || costType.Key == strings.TrimSpace(receipt.CostTypeKey)
	}
	return periodFound && costTypeFound
}

func annualStatementReceiptUploadSupported(header *multipart.FileHeader) bool {
	file, err := header.Open()
	if err != nil {
		return false
	}
	defer file.Close()
	sniff := make([]byte, 512)
	n, err := file.Read(sniff)
	return (err == nil || err == io.EOF) && n > 0 && annualStatementReceiptContentTypeSupported(http.DetectContentType(sniff[:n]))
}

func parseAnnualStatementReceiptAmount(raw string) (int64, bool) {
	raw = strings.TrimSpace(strings.ReplaceAll(raw, ",", "."))
	parts := strings.Split(raw, ".")
	if len(parts) != 2 || parts[0] == "" || len(parts[1]) != 2 {
		return 0, false
	}
	euros, errEuros := strconv.ParseInt(parts[0], 10, 64)
	cents, errCents := strconv.ParseInt(parts[1], 10, 64)
	if errEuros != nil || errCents != nil || euros < 0 || cents < 0 || cents > 99 || euros > (int64(^uint64(0)>>1)-cents)/100 {
		return 0, false
	}
	amount := euros*100 + cents
	return amount, amount > 0
}

func formatAnnualStatementReceiptAmountValue(cents int64) string {
	return fmt.Sprintf("%d,%02d", cents/100, cents%100)
}

func annualStatementReceiptRedirect(year int, status string) string {
	path := "/app/settings/annual-statement?receipt=" + status
	if year > 0 {
		path += "&year=" + strconv.Itoa(year)
	}
	return path
}

func (a *app) recordAnnualStatementReceiptAudit(tenant tenantConfig, actorEmail, role, action, summary string, receipt store.AnnualStatementReceipt) {
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug, ActorEmail: actorEmail, ActorRole: role,
		Action: action, TargetType: "annual-statement-receipt", TargetID: receipt.ID, Summary: summary,
		Details: map[string]string{
			"document_id": receipt.DocumentID, "period_year": strconv.Itoa(receipt.PeriodYear), "cost_type_key": receipt.CostTypeKey,
			"amount_cents": strconv.FormatInt(receipt.AmountCents, 10), "invoice_date": receipt.InvoiceDate,
		},
	})
}

type annualStatementReceiptSuggestionStatus uint8

const (
	annualStatementReceiptSuggestionInvalid annualStatementReceiptSuggestionStatus = iota
	annualStatementReceiptSuggestionUnavailable
	annualStatementReceiptSuggestionUncertain
	annualStatementReceiptSuggestionReady
)

func (a *app) trustedAnnualStatementReceiptSuggestion(ctx context.Context, ac authCtx, documentID string) (web.AnnualStatementReceiptSuggestionView, annualStatementReceiptSuggestionStatus) {
	if a.annualStatementReceiptSuggester == nil {
		return web.AnnualStatementReceiptSuggestionView{}, annualStatementReceiptSuggestionUnavailable
	}
	if ac.repositories.documents == nil || ac.repositories.annualStatementCostTypes == nil {
		return web.AnnualStatementReceiptSuggestionView{}, annualStatementReceiptSuggestionInvalid
	}
	document, ok := ac.repositories.documents.Get(strings.TrimSpace(documentID))
	if !ok || !document.Current || !annualStatementReceiptContentTypeSupported(document.ContentType) {
		return web.AnnualStatementReceiptSuggestionView{}, annualStatementReceiptSuggestionInvalid
	}
	path, ok := ac.repositories.documents.FilePath(document)
	if !ok {
		return web.AnnualStatementReceiptSuggestionView{}, annualStatementReceiptSuggestionInvalid
	}
	file, err := os.Open(path)
	if err != nil {
		return web.AnnualStatementReceiptSuggestionView{}, annualStatementReceiptSuggestionInvalid
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, maxDocumentBytes+1))
	if err != nil || len(raw) == 0 || int64(len(raw)) > maxDocumentBytes {
		return web.AnnualStatementReceiptSuggestionView{}, annualStatementReceiptSuggestionInvalid
	}
	suggestion, err := a.annualStatementReceiptSuggester.Suggest(ctx, annualStatementReceiptInput{
		Filename: document.Filename, ContentType: document.ContentType, Data: raw,
	})
	if err != nil || !suggestion.AmountCertain || !suggestion.DateCertain || !suggestion.CostTypeCertain {
		return web.AnnualStatementReceiptSuggestionView{}, annualStatementReceiptSuggestionUncertain
	}
	invoiceDate, err := time.Parse("2006-01-02", strings.TrimSpace(suggestion.InvoiceDate))
	if err != nil || invoiceDate.Format("2006-01-02") != strings.TrimSpace(suggestion.InvoiceDate) || suggestion.AmountCents <= 0 {
		return web.AnnualStatementReceiptSuggestionView{}, annualStatementReceiptSuggestionUncertain
	}
	var selectedCostType store.AnnualStatementCostType
	for _, costType := range ac.repositories.annualStatementCostTypes.List() {
		if costType.Key == strings.TrimSpace(suggestion.CostTypeKey) {
			selectedCostType = costType
			break
		}
	}
	if selectedCostType.Key == "" {
		return web.AnnualStatementReceiptSuggestionView{}, annualStatementReceiptSuggestionUncertain
	}
	return web.AnnualStatementReceiptSuggestionView{
		DocumentID: document.ID,
		Amount:     formatAnnualStatementReceiptAmount(suggestion.AmountCents), AmountCents: strconv.FormatInt(suggestion.AmountCents, 10),
		InvoiceDate: invoiceDate.Format("02.01.2006"), InvoiceDateValue: invoiceDate.Format("2006-01-02"),
		CostTypeKey: selectedCostType.Key, CostTypeName: selectedCostType.Name,
	}, annualStatementReceiptSuggestionReady
}

func annualStatementReceiptContentTypeSupported(contentType string) bool {
	switch strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0])) {
	case "application/pdf", "image/jpeg", "image/png", "image/webp":
		return true
	default:
		return false
	}
}

func formatAnnualStatementReceiptAmount(cents int64) string {
	euros := cents / 100
	digits := strconv.FormatInt(euros, 10)
	for index := len(digits) - 3; index > 0; index -= 3 {
		digits = digits[:index] + "." + digits[index:]
	}
	return fmt.Sprintf("%s,%02d €", digits, cents%100)
}

func annualStatementReceiptSuggestionStatusMessage(status annualStatementReceiptSuggestionStatus) string {
	switch status {
	case annualStatementReceiptSuggestionUnavailable:
		return "Keine Belegdaten verarbeitet: Es ist kein freigegebener Inferenzdienst in Österreich oder der EU eingerichtet."
	case annualStatementReceiptSuggestionUncertain:
		return "Keine verlässlichen Vorschläge erkannt. Es wurde nichts übernommen."
	default:
		return "Der Beleg konnte nicht sicher geprüft werden. Es wurde nichts übernommen."
	}
}

func annualStatementReceiptMessage(status string) (string, bool) {
	switch strings.TrimSpace(status) {
	case "created":
		return "Beleg gespeichert und dem Originaldokument zugeordnet.", true
	case "amount-updated":
		return "Belegbetrag korrigiert. Das Originaldokument wurde nicht ersetzt.", true
	case "deleted":
		return "Belegzuordnung entfernt. Die Originaldatei bleibt in Dokumente.", true
	case "invalid":
		return "Der Beleg konnte nicht gespeichert werden. Bitte Dokument, Periode, Kostenart, Betrag und Datum prüfen.", false
	case "unavailable":
		return annualStatementReceiptSuggestionStatusMessage(annualStatementReceiptSuggestionUnavailable), false
	default:
		return "", false
	}
}
