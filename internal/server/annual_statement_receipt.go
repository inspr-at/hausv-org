package server

import (
	"context"
	"fmt"
	"io"
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
	if _, _, _, _, ok := a.buildingSettingsContext(w, ac); !ok {
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
	view.Confirmed = true
	a.renderAnnualStatementPage(w, r, ac, view, "Vorschlag ausdrücklich bestätigt.", true)
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
	case "application/pdf", "image/jpeg", "image/jpg", "image/png", "image/webp":
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
	if strings.TrimSpace(status) == "unavailable" {
		return annualStatementReceiptSuggestionStatusMessage(annualStatementReceiptSuggestionUnavailable), false
	}
	return "", false
}
