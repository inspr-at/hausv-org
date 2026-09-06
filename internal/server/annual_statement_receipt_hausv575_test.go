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

func TestAnnualStatementReceiptSuggestionRequiresExplicitConfirmationBeforeWriting(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}

	// Rendering is read-only. The period's catalogue is initialized explicitly
	// below before suggestion requests, which must not change it.
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
	if _, err := repositories.annualStatementPeriods.Save(storepkg.AnnualStatementPeriod{
		Year: 2026, StartsOn: "2026-01-01", EndsOn: "2026-12-31", UpdatedBy: "manager@example.com",
	}); err != nil {
		t.Fatalf("save period: %v", err)
	}
	seedAnnualStatementPeriodStructure(t, repositories, 2026)
	periodsBefore := repositories.annualStatementPeriods.List()
	costTypesBefore := repositories.annualStatementCostTypes.List()
	receiptsBefore := repositories.annualStatementReceipts.List()

	if got := authedFormRequest(t, a, "resident@example.com", "/demo/app/settings/annual-statement/receipts/suggest", url.Values{"document_id": {document.ID}}).Code; got != http.StatusForbidden {
		t.Fatalf("resident suggestion status = %d, want 403", got)
	}
	if got := authedFormRequest(t, a, "resident@example.com", "/demo/app/settings/annual-statement/receipts/confirm", url.Values{
		"document_id": {document.ID}, "amount_cents": {"384216"}, "invoice_date": {"2026-01-31"}, "cost_type_key": {"hausbetreuung"}, "year": {"2026"},
	}).Code; got != http.StatusForbidden {
		t.Fatalf("resident confirmation status = %d, want 403", got)
	}

	preview := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/receipts/suggest", url.Values{"document_id": {document.ID}, "year": {"2026"}})
	if preview.Code != http.StatusOK {
		t.Fatalf("suggestion status = %d", preview.Code)
	}
	for _, want := range []string{"3.842,16 €", "31.01.2026", "Hausbetreuung", "Vorschlag ausdrücklich bestätigen", `name="amount_cents" value="384216"`} {
		if !strings.Contains(preview.Body.String(), want) {
			t.Fatalf("suggestion page missing %q", want)
		}
	}
	assertAnnualStatementReceiptDidNotWrite(t, repositories, periodsBefore, costTypesBefore, receiptsBefore)

	tampered := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/receipts/confirm", url.Values{
		"document_id": {document.ID}, "amount_cents": {"1"}, "invoice_date": {"2026-01-31"}, "cost_type_key": {"hausbetreuung"}, "year": {"2026"},
	})
	if tampered.Code != http.StatusOK || !strings.Contains(tampered.Body.String(), "stimmt nicht mehr mit dem sicheren Vorschlag überein") || strings.Contains(tampered.Body.String(), "<strong>Vorschlag bestätigt.</strong>") {
		t.Fatalf("tampered confirmation did not fail closed: status=%d", tampered.Code)
	}
	assertAnnualStatementReceiptDidNotWrite(t, repositories, periodsBefore, costTypesBefore, receiptsBefore)

	confirmed := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/receipts/confirm", url.Values{
		"document_id": {document.ID}, "amount_cents": {"384216"}, "invoice_date": {"2026-01-31"}, "cost_type_key": {"hausbetreuung"}, "year": {"2026"},
	})
	if confirmed.Code != http.StatusOK || !strings.Contains(confirmed.Body.String(), "Vorschlag bestätigt und Beleg gespeichert.") || strings.Contains(confirmed.Body.String(), "Es wurde noch kein Beleg gespeichert") {
		t.Fatalf("explicit confirmation was not rendered: status=%d", confirmed.Code)
	}
	receipts := repositories.annualStatementReceipts.List()
	if len(receipts) != 1 || receipts[0].DocumentID != document.ID || receipts[0].PeriodYear != 2026 || receipts[0].AmountCents != 384216 || receipts[0].InvoiceDate != "2026-01-31" || receipts[0].CostTypeKey != "hausbetreuung" {
		t.Fatalf("confirmed receipt = %+v", receipts)
	}
	if suggester.calls != 3 {
		t.Fatalf("suggester calls = %d, want preview plus two trusted re-checks", suggester.calls)
	}
}

func TestAnnualStatementReceiptSuggestionUsesSelectedPeriodCatalog(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	repositories := testRepositories(a, "demo")
	for _, period := range []storepkg.AnnualStatementPeriod{
		{Year: 2026, StartsOn: "2026-01-01", EndsOn: "2026-12-31", UpdatedBy: "manager@example.com"},
		{Year: 2027, StartsOn: "2027-01-01", EndsOn: "2027-12-31", UpdatedBy: "manager@example.com"},
	} {
		if _, err := repositories.annualStatementPeriods.Save(period); err != nil {
			t.Fatal(err)
		}
	}
	seedAnnualStatementPeriodStructure(t, repositories, 2026)
	if _, err := repositories.annualStatementPeriods.SaveStructureCostType(2026, storepkg.AnnualStatementCostType{
		Key: "period_only", Name: "Nur 2026", Allocatable: true, AllocationKey: storepkg.AllocationKeyNutzwert, UpdatedBy: "manager@example.com",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := repositories.annualStatementCostTypes.Save(storepkg.AnnualStatementCostType{
		Key: "global_only", Name: "Nur global", Allocatable: true, AllocationKey: storepkg.AllocationKeyNutzwert, UpdatedBy: "manager@example.com",
	}); err != nil {
		t.Fatal(err)
	}
	document, err := repositories.documents.CreateGenerated(storepkg.DocumentRecord{
		Title: "Periodenbeleg", Category: documentCategoryBilling, Visibility: documentVisibilityManagerOnly, UploadedBy: "manager@example.com",
	}, "periode.pdf", "application/pdf", []byte("%PDF-1.4 fixture receipt"), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	a.annualStatementReceiptSuggester = &fixedAnnualStatementReceiptSuggester{suggestion: annualStatementReceiptSuggestion{
		AmountCents: 1200, InvoiceDate: "2026-06-01", CostTypeKey: "period_only", AmountCertain: true, DateCertain: true, CostTypeCertain: true,
	}}
	preview := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/receipts/suggest", url.Values{
		"document_id": {document.ID}, "year": {"2026"},
	})
	if preview.Code != http.StatusOK || !strings.Contains(preview.Body.String(), "Nur 2026") ||
		!strings.Contains(preview.Body.String(), `name="year" value="2026"`) || strings.Contains(preview.Body.String(), "Nur global") {
		t.Fatalf("selected-period suggestion was not preserved: status=%d", preview.Code)
	}

	a.annualStatementReceiptSuggester = &fixedAnnualStatementReceiptSuggester{suggestion: annualStatementReceiptSuggestion{
		AmountCents: 1200, InvoiceDate: "2026-06-01", CostTypeKey: "global_only", AmountCertain: true, DateCertain: true, CostTypeCertain: true,
	}}
	rejected := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/receipts/suggest", url.Values{
		"document_id": {document.ID}, "year": {"2026"},
	})
	if rejected.Code != http.StatusOK || !strings.Contains(rejected.Body.String(), "Keine verlässlichen Vorschläge erkannt") ||
		strings.Contains(rejected.Body.String(), `action="/demo/app/settings/annual-statement/receipts/confirm"`) {
		t.Fatalf("global-only suggestion did not fail closed for selected period: status=%d", rejected.Code)
	}
}

func seedAnnualStatementPeriodStructure(t *testing.T, repositories requestRepositories, year int) {
	t.Helper()
	if err := repositories.annualStatementCostTypes.EnsureDefaults("manager@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := repositories.annualStatementPeriods.EnsureStructure(year, repositories.annualStatementCostTypes.List(), repositories.units.List(), "manager@example.com"); err != nil {
		t.Fatal(err)
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
	if _, err := repositories.annualStatementPeriods.Save(storepkg.AnnualStatementPeriod{Year: 2026, StartsOn: "2026-01-01", EndsOn: "2026-12-31", UpdatedBy: "manager@example.com"}); err != nil {
		t.Fatal(err)
	}
	seedAnnualStatementPeriodStructure(t, repositories, 2026)
	periodsBefore := repositories.annualStatementPeriods.List()
	costTypesBefore := repositories.annualStatementCostTypes.List()
	receiptsBefore := repositories.annualStatementReceipts.List()

	response := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/receipts/suggest", url.Values{"document_id": {document.ID}, "year": {"2026"}})
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "Keine verlässlichen Vorschläge erkannt. Es wurde nichts übernommen.") {
		t.Fatalf("uncertain response did not fail closed: status=%d", response.Code)
	}
	if strings.Contains(response.Body.String(), `action="/demo/app/settings/annual-statement/receipts/confirm"`) || strings.Contains(response.Body.String(), "3.842,16 €") {
		t.Fatal("uncertain response exposed a confirmable suggestion")
	}
	assertAnnualStatementReceiptDidNotWrite(t, repositories, periodsBefore, costTypesBefore, receiptsBefore)
}

func TestAnnualStatementReceiptSuggestionIsClosedWithoutApprovedHost(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	page := authedRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement")
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "Automatische Erkennung derzeit geschlossen.") || !strings.Contains(page.Body.String(), "Es werden keine Belegdaten versendet.") {
		t.Fatalf("closed-host contract missing: status=%d", page.Code)
	}
}

func assertAnnualStatementReceiptDidNotWrite(t *testing.T, repositories requestRepositories, periodsBefore []storepkg.AnnualStatementPeriod, costTypesBefore []storepkg.AnnualStatementCostType, receiptsBefore []storepkg.AnnualStatementReceipt) {
	t.Helper()
	if got := repositories.annualStatementPeriods.List(); !reflect.DeepEqual(got, periodsBefore) {
		t.Fatalf("receipt suggestion changed periods: before=%+v after=%+v", periodsBefore, got)
	}
	if got := repositories.annualStatementCostTypes.List(); !reflect.DeepEqual(got, costTypesBefore) {
		t.Fatalf("receipt suggestion changed cost types: before=%+v after=%+v", costTypesBefore, got)
	}
	if got := repositories.annualStatementReceipts.List(); !reflect.DeepEqual(got, receiptsBefore) {
		t.Fatalf("receipt suggestion wrote receipts: before=%+v after=%+v", receiptsBefore, got)
	}
}
