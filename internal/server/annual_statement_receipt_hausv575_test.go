package server

import (
	"context"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	storepkg "github.com/inspr-at/hausv-org/internal/store"
)

type fixedAnnualStatementReceiptSuggester struct {
	suggestion annualStatementReceiptSuggestion
	calls      int
}

func (s *fixedAnnualStatementReceiptSuggester) Suggest(_ context.Context, input annualStatementReceiptInput) (annualStatementReceiptSuggestion, error) {
	s.calls++
	if input.Filename == "" || !annualStatementReceiptContentTypeSupported(input.ContentType) || !strings.Contains(string(input.Data), "fixture receipt") {
		return annualStatementReceiptSuggestion{}, nil
	}
	return s.suggestion, nil
}

func TestAnnualStatementReceiptSuggestionRequiresExplicitConfirmationWithoutWriting(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}

	// The existing page initializes the HAUSV-574 catalogue before a document
	// can be selected. The suggestion requests below must not change it.
	if response := authedRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement"); response.Code != http.StatusOK {
		t.Fatalf("annual statement page status = %d", response.Code)
	}
	document, err := documentRepositoryForTest(a, "demo").CreateGenerated(storepkg.DocumentRecord{
		Title: "Fernwärme Jänner", Category: documentCategoryBilling, Visibility: documentVisibilityManagerOnly, UploadedBy: "manager@example.com",
	}, "fernwaerme-jaenner.pdf", "application/pdf", []byte("%PDF-1.4 fixture receipt"), time.Now())
	if err != nil {
		t.Fatalf("create receipt document: %v", err)
	}
	suggester := &fixedAnnualStatementReceiptSuggester{suggestion: annualStatementReceiptSuggestion{
		AmountCents: 384216, InvoiceDate: "2026-01-31", CostTypeKey: "hausbetreuung",
		AmountCertain: true, DateCertain: true, CostTypeCertain: true,
	}}
	a.annualStatementReceiptSuggester = suggester
	repositories := testRepositories(a, "demo")
	periodsBefore := repositories.annualStatementPeriods.List()
	costTypesBefore := repositories.annualStatementCostTypes.List()

	if got := authedFormRequest(t, a, "resident@example.com", "/demo/app/settings/annual-statement/receipts/suggest", url.Values{"document_id": {document.ID}}).Code; got != http.StatusForbidden {
		t.Fatalf("resident suggestion status = %d, want 403", got)
	}
	if got := authedFormRequest(t, a, "resident@example.com", "/demo/app/settings/annual-statement/receipts/confirm", url.Values{
		"document_id": {document.ID}, "amount_cents": {"384216"}, "invoice_date": {"2026-01-31"}, "cost_type_key": {"hausbetreuung"},
	}).Code; got != http.StatusForbidden {
		t.Fatalf("resident confirmation status = %d, want 403", got)
	}

	preview := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/receipts/suggest", url.Values{"document_id": {document.ID}})
	if preview.Code != http.StatusOK {
		t.Fatalf("suggestion status = %d", preview.Code)
	}
	for _, want := range []string{"3.842,16 €", "31.01.2026", "Hausbetreuung", "Vorschlag ausdrücklich bestätigen", `name="amount_cents" value="384216"`} {
		if !strings.Contains(preview.Body.String(), want) {
			t.Fatalf("suggestion page missing %q", want)
		}
	}
	assertAnnualStatementReceiptDidNotWrite(t, repositories, periodsBefore, costTypesBefore)

	tampered := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/receipts/confirm", url.Values{
		"document_id": {document.ID}, "amount_cents": {"1"}, "invoice_date": {"2026-01-31"}, "cost_type_key": {"hausbetreuung"},
	})
	if tampered.Code != http.StatusOK || !strings.Contains(tampered.Body.String(), "stimmt nicht mehr mit dem sicheren Vorschlag überein") || strings.Contains(tampered.Body.String(), "<strong>Vorschlag bestätigt.</strong>") {
		t.Fatalf("tampered confirmation did not fail closed: status=%d", tampered.Code)
	}
	assertAnnualStatementReceiptDidNotWrite(t, repositories, periodsBefore, costTypesBefore)

	confirmed := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/receipts/confirm", url.Values{
		"document_id": {document.ID}, "amount_cents": {"384216"}, "invoice_date": {"2026-01-31"}, "cost_type_key": {"hausbetreuung"},
	})
	if confirmed.Code != http.StatusOK || !strings.Contains(confirmed.Body.String(), "Vorschlag ausdrücklich bestätigt.") || !strings.Contains(confirmed.Body.String(), "Es wurde noch kein Beleg gespeichert") {
		t.Fatalf("explicit confirmation was not rendered: status=%d", confirmed.Code)
	}
	assertAnnualStatementReceiptDidNotWrite(t, repositories, periodsBefore, costTypesBefore)
	if suggester.calls != 3 {
		t.Fatalf("suggester calls = %d, want preview plus two trusted re-checks", suggester.calls)
	}
}

func TestAnnualStatementReceiptSuggestionAcceptsOnlyPDFAndScanInputs(t *testing.T) {
	for _, test := range []struct {
		contentType string
		want        bool
	}{
		{contentType: "application/pdf", want: true},
		{contentType: "image/jpeg", want: true},
		{contentType: "image/png", want: true},
		{contentType: "image/webp", want: true},
		{contentType: "application/xml", want: false},
		{contentType: "text/plain", want: false},
	} {
		if got := annualStatementReceiptContentTypeSupported(test.contentType); got != test.want {
			t.Errorf("content type %q supported = %t, want %t", test.contentType, got, test.want)
		}
	}
}

func TestAnnualStatementReceiptSuggestionFailsClosedWhenAnyFieldIsUncertain(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	_ = authedRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement")
	document, err := documentRepositoryForTest(a, "demo").CreateGenerated(storepkg.DocumentRecord{
		Title: "Unsicherer Beleg", Category: documentCategoryBilling, Visibility: documentVisibilityManagerOnly, UploadedBy: "manager@example.com",
	}, "unsicher.pdf", "application/pdf", []byte("%PDF-1.4 fixture receipt"), time.Now())
	if err != nil {
		t.Fatalf("create receipt document: %v", err)
	}
	a.annualStatementReceiptSuggester = &fixedAnnualStatementReceiptSuggester{suggestion: annualStatementReceiptSuggestion{
		AmountCents: 384216, InvoiceDate: "2026-01-31", CostTypeKey: "hausbetreuung",
		AmountCertain: true, DateCertain: true, CostTypeCertain: false,
	}}
	repositories := testRepositories(a, "demo")
	periodsBefore := repositories.annualStatementPeriods.List()
	costTypesBefore := repositories.annualStatementCostTypes.List()

	response := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/receipts/suggest", url.Values{"document_id": {document.ID}})
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "Keine verlässlichen Vorschläge erkannt. Es wurde nichts übernommen.") {
		t.Fatalf("uncertain response did not fail closed: status=%d", response.Code)
	}
	if strings.Contains(response.Body.String(), `action="/demo/app/settings/annual-statement/receipts/confirm"`) || strings.Contains(response.Body.String(), "3.842,16 €") {
		t.Fatal("uncertain response exposed a confirmable suggestion")
	}
	assertAnnualStatementReceiptDidNotWrite(t, repositories, periodsBefore, costTypesBefore)
}

func TestAnnualStatementReceiptSuggestionIsClosedWithoutApprovedHost(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	page := authedRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement")
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "Automatische Erkennung derzeit geschlossen.") || !strings.Contains(page.Body.String(), "Es werden keine Belegdaten versendet.") {
		t.Fatalf("closed-host contract missing: status=%d", page.Code)
	}
}

func assertAnnualStatementReceiptDidNotWrite(t *testing.T, repositories requestRepositories, periodsBefore []storepkg.AnnualStatementPeriod, costTypesBefore []storepkg.AnnualStatementCostType) {
	t.Helper()
	if got := repositories.annualStatementPeriods.List(); !reflect.DeepEqual(got, periodsBefore) {
		t.Fatalf("receipt suggestion changed periods: before=%+v after=%+v", periodsBefore, got)
	}
	if got := repositories.annualStatementCostTypes.List(); !reflect.DeepEqual(got, costTypesBefore) {
		t.Fatalf("receipt suggestion changed cost types: before=%+v after=%+v", costTypesBefore, got)
	}
}
