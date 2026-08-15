package server

import (
	"net/http"
	"net/url"
	"testing"
)

// HAUSV-148: filing the same handover twice must produce exactly one document,
// not a duplicate on the second (retry) submit.
func TestHandoverFilingIsIdempotent(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email: "manager@example.com", Role: roleManager,
		Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods(),
	})
	h, err := testRepositories(a, "demo").handovers.Create(handoverRecord{
		ID: "hv-test-1", TenantSlug: "demo", Title: "Whg 1", HandoverType: "auszug", CreatedBy: "manager@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	file := func() int {
		rr := authedFormRequest(t, a, "manager@example.com", "/demo/app/uebergaben/file", url.Values{"id": {h.ID}})
		if rr.Code != http.StatusSeeOther {
			t.Fatalf("file status = %d", rr.Code)
		}
		return len(documentRepositoryForTest(a, "demo").List())
	}
	if got := file(); got != 1 {
		t.Fatalf("after first filing: %d documents, want 1", got)
	}
	if got := file(); got != 1 {
		t.Fatalf("after RE-filing: %d documents, want 1 (duplicate created)", got)
	}
}
