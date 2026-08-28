package server

import (
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	storepkg "github.com/inspr-at/hausv-org/internal/store"
)

func TestHAUSV576ManagerReceiptCRUDRetainsOriginalDocument(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	if response := authedRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement"); response.Code != http.StatusOK {
		t.Fatalf("page status = %d", response.Code)
	}
	repositories := testRepositories(a, "demo")
	if _, err := repositories.annualStatementPeriods.Save(storepkg.AnnualStatementPeriod{
		Year: 2026, StartsOn: "2026-01-01", EndsOn: "2026-12-31", UpdatedBy: "manager@example.com",
	}); err != nil {
		t.Fatalf("period: %v", err)
	}
	document, err := repositories.documents.CreateGenerated(storepkg.DocumentRecord{
		Title: "Wasserrechnung April", Category: documentCategoryBilling, Visibility: documentVisibilityManagerOnly, UploadedBy: "manager@example.com",
	}, "wasser-april.pdf", "application/pdf", []byte("%PDF-1.4 original receipt"), time.Now())
	if err != nil {
		t.Fatalf("document: %v", err)
	}
	originalPath, ok := repositories.documents.FilePath(document)
	if !ok {
		t.Fatal("original file path missing")
	}

	for path, values := range map[string]url.Values{
		"/demo/app/settings/annual-statement/receipts":               {"document_id": {document.ID}, "year": {"2026"}, "cost_type_key": {"grundsteuer"}, "amount": {"128,40"}, "invoice_date": {"2026-04-30"}},
		"/demo/app/settings/annual-statement/receipts/update-amount": {"id": {"missing"}, "year": {"2026"}, "amount": {"1,00"}},
		"/demo/app/settings/annual-statement/receipts/delete":        {"id": {"missing"}, "year": {"2026"}},
	} {
		if got := authedFormRequest(t, a, "resident@example.com", path, values).Code; got != http.StatusForbidden {
			t.Fatalf("resident %s status = %d, want 403", path, got)
		}
	}

	created := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/receipts", url.Values{
		"document_id": {document.ID}, "year": {"2026"}, "cost_type_key": {"grundsteuer"}, "amount": {"128,40"}, "invoice_date": {"2026-04-30"},
	})
	if created.Code != http.StatusSeeOther || !strings.Contains(created.Header().Get("Location"), "receipt=created") {
		t.Fatalf("create status=%d location=%q", created.Code, created.Header().Get("Location"))
	}
	receipts := repositories.annualStatementReceipts.ListByPeriod(2026)
	if len(receipts) != 1 || receipts[0].DocumentID != document.ID || receipts[0].AmountCents != 12840 {
		t.Fatalf("created receipts = %+v", receipts)
	}
	page := authedRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement?year=2026")
	for _, want := range []string{"Wasserrechnung April", "128,40 €", "30.04.2026", "Grundsteuer", "Betrag korrigieren", "Die Originaldatei bleibt in Dokumente."} {
		if !strings.Contains(page.Body.String(), want) {
			t.Fatalf("receipt page missing %q", want)
		}
	}

	updated := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/receipts/update-amount", url.Values{
		"id": {receipts[0].ID}, "year": {"2026"}, "amount": {"200,00"},
	})
	if updated.Code != http.StatusSeeOther {
		t.Fatalf("update status = %d", updated.Code)
	}
	corrected, found := repositories.annualStatementReceipts.Get(receipts[0].ID)
	if !found || corrected.AmountCents != 20000 || corrected.DocumentID != document.ID {
		t.Fatalf("corrected receipt = %+v found=%t", corrected, found)
	}

	deleted := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/receipts/delete", url.Values{
		"id": {receipts[0].ID}, "year": {"2026"},
	})
	if deleted.Code != http.StatusSeeOther || len(repositories.annualStatementReceipts.List()) != 0 {
		t.Fatalf("delete status=%d receipts=%+v", deleted.Code, repositories.annualStatementReceipts.List())
	}
	retained, found := repositories.documents.Get(document.ID)
	if !found || retained.ID != document.ID {
		t.Fatalf("original document removed: %+v found=%t", retained, found)
	}
	if _, err := os.Stat(originalPath); err != nil {
		t.Fatalf("original file removed: %v", err)
	}
	if got := len(a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionAnnualReceiptCreate, Limit: 20})); got != 1 {
		t.Fatalf("create audits = %d", got)
	}
	if got := len(a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionAnnualReceiptAmount, Limit: 20})); got != 1 {
		t.Fatalf("amount audits = %d", got)
	}
	if got := len(a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionAnnualReceiptDelete, Limit: 20})); got != 1 {
		t.Fatalf("delete audits = %d", got)
	}
}

func TestHAUSV576ReceiptUploadCreatesDocumentThenReceipt(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	_ = authedRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement")
	repositories := testRepositories(a, "demo")
	if _, err := repositories.annualStatementPeriods.Save(storepkg.AnnualStatementPeriod{Year: 2026, StartsOn: "2026-01-01", EndsOn: "2026-12-31", UpdatedBy: "manager@example.com"}); err != nil {
		t.Fatalf("period: %v", err)
	}
	documentsBefore := len(repositories.documents.List())
	response := authedMultipartFilesRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/receipts", map[string]string{
		"year": "2026", "cost_type_key": "grundsteuer", "amount": "77,15", "invoice_date": "2026-05-15",
	}, []multipartTestFile{{Field: "receipt_file", Filename: "grundsteuer-mai.pdf", Body: []byte("%PDF-1.4 uploaded receipt")}})
	if response.Code != http.StatusSeeOther || !strings.Contains(response.Header().Get("Location"), "receipt=created") {
		t.Fatalf("upload status=%d location=%q", response.Code, response.Header().Get("Location"))
	}
	receipts := repositories.annualStatementReceipts.List()
	if len(receipts) != 1 || receipts[0].AmountCents != 7715 {
		t.Fatalf("uploaded receipts = %+v", receipts)
	}
	if len(repositories.documents.List()) != documentsBefore+1 {
		t.Fatalf("documents count = %d, want %d", len(repositories.documents.List()), documentsBefore+1)
	}
	document, found := repositories.documents.Get(receipts[0].DocumentID)
	if !found || document.Title != "grundsteuer-mai.pdf" || document.Category != documentCategoryBilling || document.Visibility != documentVisibilityManagerOnly || !document.Current {
		t.Fatalf("uploaded document = %+v found=%t", document, found)
	}
	path, ok := repositories.documents.FilePath(document)
	if !ok {
		t.Fatal("uploaded document path missing")
	}
	if raw, err := os.ReadFile(path); err != nil || !strings.Contains(string(raw), "uploaded receipt") {
		t.Fatalf("uploaded original bytes missing: err=%v", err)
	}
}

func TestHAUSV576InvalidReceiptSubmitWritesNothingOrAudit(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	_ = authedRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement")
	repositories := testRepositories(a, "demo")
	if _, err := repositories.annualStatementPeriods.Save(storepkg.AnnualStatementPeriod{Year: 2026, StartsOn: "2026-01-01", EndsOn: "2026-12-31", UpdatedBy: "manager@example.com"}); err != nil {
		t.Fatalf("period: %v", err)
	}
	document, err := repositories.documents.CreateGenerated(storepkg.DocumentRecord{
		Title: "Valid original", Category: documentCategoryBilling, Visibility: documentVisibilityManagerOnly, UploadedBy: "manager@example.com",
	}, "valid.pdf", "application/pdf", []byte("%PDF-1.4 valid original"), time.Now())
	if err != nil {
		t.Fatalf("document: %v", err)
	}
	base := url.Values{"document_id": {document.ID}, "year": {"2026"}, "cost_type_key": {"grundsteuer"}, "amount": {"10,00"}, "invoice_date": {"2026-01-31"}}
	for field, invalid := range map[string]string{"year": "2025", "cost_type_key": "unknown", "amount": "0,00", "invoice_date": "2026-02-30", "document_id": "unknown"} {
		values := url.Values{}
		for key, entries := range base {
			values[key] = append([]string(nil), entries...)
		}
		values.Set(field, invalid)
		response := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/receipts", values)
		if response.Code != http.StatusSeeOther || !strings.Contains(response.Header().Get("Location"), "receipt=invalid") {
			t.Fatalf("invalid %s status=%d location=%q", field, response.Code, response.Header().Get("Location"))
		}
	}
	documentsBefore := len(repositories.documents.List())
	badUpload := authedMultipartFilesRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/receipts", map[string]string{
		"year": "2026", "cost_type_key": "grundsteuer", "amount": "10,00", "invoice_date": "2026-01-31",
	}, []multipartTestFile{{Field: "receipt_file", Filename: "not-a-receipt.txt", Body: []byte("plain text")}})
	if badUpload.Code != http.StatusSeeOther || !strings.Contains(badUpload.Header().Get("Location"), "receipt=invalid") {
		t.Fatalf("unsupported upload status=%d location=%q", badUpload.Code, badUpload.Header().Get("Location"))
	}
	if len(repositories.annualStatementReceipts.List()) != 0 || len(repositories.documents.List()) != documentsBefore {
		t.Fatalf("invalid submit wrote receipts=%+v documents=%d/%d", repositories.annualStatementReceipts.List(), len(repositories.documents.List()), documentsBefore)
	}
	if got := len(a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionAnnualReceiptCreate, Limit: 20})); got != 0 {
		t.Fatalf("invalid submit audits = %d", got)
	}
}

func TestHAUSV576SuggestDoesNotPersistAndConfirmPersistsExactlyOne(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	_ = authedRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement")
	repositories := testRepositories(a, "demo")
	if _, err := repositories.annualStatementPeriods.Save(storepkg.AnnualStatementPeriod{Year: 2026, StartsOn: "2026-01-01", EndsOn: "2026-12-31", UpdatedBy: "manager@example.com"}); err != nil {
		t.Fatalf("period: %v", err)
	}
	document, err := repositories.documents.CreateGenerated(storepkg.DocumentRecord{
		Title: "Vorgeschlagener Beleg", Category: documentCategoryBilling, Visibility: documentVisibilityManagerOnly, UploadedBy: "manager@example.com",
	}, "vorschlag.pdf", "application/pdf", []byte("%PDF-1.4 fixture receipt"), time.Now())
	if err != nil {
		t.Fatalf("document: %v", err)
	}
	a.annualStatementReceiptSuggester = &fixedAnnualStatementReceiptSuggester{suggestion: annualStatementReceiptSuggestion{
		AmountCents: 9900, InvoiceDate: "2026-06-01", CostTypeKey: "grundsteuer", AmountCertain: true, DateCertain: true, CostTypeCertain: true,
	}}
	preview := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/receipts/suggest", url.Values{"document_id": {document.ID}})
	if preview.Code != http.StatusOK || len(repositories.annualStatementReceipts.List()) != 0 {
		t.Fatalf("suggest status=%d receipts=%+v", preview.Code, repositories.annualStatementReceipts.List())
	}
	confirmed := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/receipts/confirm", url.Values{
		"document_id": {document.ID}, "year": {"2026"}, "amount_cents": {"9900"}, "invoice_date": {"2026-06-01"}, "cost_type_key": {"grundsteuer"},
	})
	receipts := repositories.annualStatementReceipts.List()
	if confirmed.Code != http.StatusOK || len(receipts) != 1 || receipts[0].DocumentID != document.ID || receipts[0].AmountCents != 9900 {
		t.Fatalf("confirm status=%d receipts=%+v", confirmed.Code, receipts)
	}
}
