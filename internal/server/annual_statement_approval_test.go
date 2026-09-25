package server

import (
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/config"
	"github.com/inspr-at/hausv-org/internal/dbtest"
	"github.com/inspr-at/hausv-org/internal/store"
)

func TestAnnualManagementAddressWarningAndFrozenSnapshot(t *testing.T) {
	a, repos, _ := newArchiveDemoApp(t)
	tenant := a.tenants[archiveDemoTenant]
	tenant.Organisation = "musterstadt"
	a.tenants[archiveDemoTenant] = tenant
	a.organisations = map[string]config.OrganisationConfig{"musterstadt": {Key: "musterstadt", Name: "Hausverwaltung Musterstadt GmbH"}}
	database := dbtest.Open(t)
	a.orgSettings = func(key string) store.OrgSettingsRepository { return store.BindOrgSettingsRepository(database, key) }
	settings, err := a.orgSettings("musterstadt").Get(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	settings.ContactAddress = ""
	if err := a.orgSettings("musterstadt").Save(t.Context(), settings); err != nil {
		t.Fatal(err)
	}
	checkWarning := func(runID string, want bool) {
		t.Helper()
		page := archiveDemoRequest(t, a, archiveDemoManager, http.MethodGet, "/app/settings/annual-statement?year=2025&run="+url.QueryEscape(runID), nil)
		if page.Code != http.StatusOK || strings.Contains(page.Body.String(), "data-management-address-warning") != want {
			t.Fatalf("address warning=%v status=%d", want, page.Code)
		}
		if want && !strings.Contains(page.Body.String(), "/app/verwaltung/einstellungen#hausverwaltung") {
			t.Fatal("address warning has no settings link")
		}
	}
	create := func() store.AnnualStatementRun {
		t.Helper()
		response := archiveDemoRequest(t, a, archiveDemoManager, http.MethodPost, "/app/settings/annual-statement/runs", url.Values{"year": {"2025"}})
		if response.Code != http.StatusSeeOther {
			t.Fatal("create", response.Code)
		}
		runs, err := repos.annualStatementRuns.List(2025)
		if err != nil || len(runs) == 0 {
			t.Fatal("runs", err)
		}
		return runs[0]
	}
	checkWarning("", true)
	first := create()
	checkWarning(first.ID, true)
	settings.ContactAddress = "Musterstraße 12, 8010 Graz"
	if err := a.orgSettings("musterstadt").Save(t.Context(), settings); err != nil {
		t.Fatal(err)
	}
	checkWarning(first.ID, true)
	second := create()
	if second.Input.Presentation.ContactAddress != settings.ContactAddress {
		t.Fatal("new run did not snapshot organisation address")
	}
	checkWarning(second.ID, false)
	frozen, _, err := repos.annualStatementRuns.Get(first.ID)
	if err != nil || frozen.Input.Presentation.ContactAddress != "" {
		t.Fatal("settings edit changed stored address", err)
	}
	if _, _, err := repos.annualStatementRuns.Approve(first.ID, archiveDemoManager, store.RoleManager, time.Now()); err != nil {
		t.Fatal(err)
	}
	checkWarning(first.ID, false)
}

func TestAnnualApprovalLifecycle(t *testing.T) {
	a, repos, _ := newArchiveDemoApp(t)
	profile := a.profiles[archiveDemoManager]
	profile.FirstName, profile.LastName = "Vera", "Verwalter"
	a.profiles[archiveDemoManager] = profile
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

	structure, _ := repos.annualStatementPeriods.Structure(2025)
	changedLegal := structure.Legal
	changedLegal.Regime = "mrg_voll"
	if err := repos.annualStatementPeriods.SaveLegal(2025, changedLegal); err != nil {
		t.Fatal(err)
	}
	if response := archiveDemoRequest(t, a, archiveDemoManager, http.MethodPost, route+"/approve", nil); response.Code != http.StatusConflict {
		t.Fatal("stale legal snapshot approved", response.Code)
	}
	if err := repos.annualStatementPeriods.SaveLegal(2025, structure.Legal); err != nil {
		t.Fatal(err)
	}
	w = archiveDemoRequest(t, a, archiveDemoManager, http.MethodPost, route+"/approve", nil)
	if w.Code != http.StatusSeeOther {
		t.Fatal(w.Code, w.Body.String())
	}
	final, _, err := repos.annualStatementRuns.Get(draft.ID)
	if err != nil || final.Approval == nil || final.Approval.ApprovedBy != archiveDemoManager || final.Approval.Role != store.RoleManager || final.Approval.ApprovedName != "Vera Verwalter" {
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
	profile.FirstName = "Geändert"
	a.profiles[archiveDemoManager] = profile
	if err := repos.annualStatementPeriods.SaveLegal(2025, changedLegal); err != nil {
		t.Fatal(err)
	}
	frozen := archiveDemoRequest(t, a, archiveDemoManager, http.MethodGet, route+"/pdf", nil)
	if frozen.Body.String() != pdf.Body.String() {
		t.Fatal("approved PDF changed with legal settings")
	}
}

func TestAnnualApprovalViennaCalendarDate(t *testing.T) {
	for _, tc := range []struct{ at, want string }{
		{"2026-09-24T23:47:00Z", "25.09.2026"},
		{"2026-01-31T23:30:00Z", "01.02.2026"},
		{"2026-03-29T01:30:00Z", "29.03.2026"},
	} {
		at, _ := time.Parse(time.RFC3339, tc.at)
		if got := annualStatementViennaDate(at); got != tc.want {
			t.Errorf("%s: %s", tc.at, got)
		}
	}
}
