package server

import (
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
)

func TestAnnualApprovalLifecycle(t *testing.T) {
	a, repos, _ := newArchiveDemoApp(t)
	w := archiveDemoRequest(t, a, archiveDemoManager, http.MethodPost, "/app/settings/annual-statement/runs", url.Values{"year": {"2025"}})
	if w.Code != http.StatusSeeOther {
		t.Fatal(w.Code)
	}
	runs, err := repos.annualStatementRuns.List(2025)
	if err != nil || len(runs) != 1 {
		t.Fatal(runs, err)
	}
	draft := runs[0]
	route := "/app/settings/annual-statement/runs/" + draft.ID
	for _, suffix := range []string{"/archive", "/send"} {
		w = archiveDemoRequest(t, a, archiveDemoManager, http.MethodPost, route+suffix, nil)
		if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "freigeben") {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	w = archiveDemoRequest(t, a, archiveDemoManager, http.MethodPost, route+"/approve", nil)
	if w.Code != http.StatusSeeOther {
		t.Fatal(w.Code, w.Body.String())
	}
	final, _, err := repos.annualStatementRuns.Get(draft.ID)
	if err != nil || final.Approval == nil || final.Approval.ApprovedBy != archiveDemoManager || final.Approval.Role != store.RoleManager {
		t.Fatal(final.Approval, err)
	}
	if final.InputHash != draft.InputHash || !reflect.DeepEqual(final.Input, draft.Input) || !reflect.DeepEqual(final.Result, draft.Result) {
		t.Fatal("approval changed calculation")
	}
	again, changed, err := repos.annualStatementRuns.Approve(draft.ID, "other@example.com", store.RoleAdmin, time.Now().Add(time.Hour))
	if err != nil || changed || !reflect.DeepEqual(again, final) {
		t.Fatal("approval was overwritten", err)
	}
	pdf := archiveDemoRequest(t, a, archiveDemoManager, http.MethodGet, route+"/pdf", nil)
	if pdf.Code != 200 || strings.Contains(pdf.Body.String(), "Entwurf") || !strings.Contains(pdf.Body.String(), "Freigegeben:") {
		t.Fatal("final PDF", pdf.Code)
	}
}
