package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/config"
	"github.com/inspr-at/hausv-org/internal/dbtest"
	"github.com/inspr-at/hausv-org/internal/store"
)

func rightsTestApp(t *testing.T) *app {
	t.Helper()
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	tenant := a.tenants["demo"]
	tenant.Organisation = "org"
	a.tenants["demo"] = tenant
	a.organisations = map[string]config.OrganisationConfig{"org": {Key: "org", Name: "Testverwaltung"}}
	database := dbtest.Open(t)
	a.capabilityRepo = func(org string) store.CapabilityRepository { return store.BindCapabilityRepository(database, org) }
	for email, role := range map[string]string{"manager@example.com": roleManager, "resident@example.com": roleResident} {
		a.profiles[email] = userProfile{Email: email, Role: role, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	}
	return a
}

func TestRechteEditingAndAuditHAUSV699(t *testing.T) {
	a := rightsTestApp(t)
	page := authedRequest(t, a, "admin@example.com", "/demo/app/verwaltung/rechte")
	for _, want := range []string{`data-rights-switch`, "Auf Standard zurücksetzen", "Betrifft 1 Benutzer dieser Organisation", `/assets/rights.js`, "nicht delegierbar"} {
		if !strings.Contains(page.Body.String(), want) {
			t.Errorf("admin page missing %s", want)
		}
	}
	readonly := authedRequest(t, a, "manager@example.com", "/demo/app/verwaltung/rechte")
	if readonly.Code != http.StatusOK || !strings.Contains(readonly.Body.String(), "Nur zur Ansicht") || strings.Contains(readonly.Body.String(), `data-rights-switch`) {
		t.Fatal("non-admin view changed")
	}
	form := url.Values{"operation": {"set"}, "family": {"bewohner"}, "area": {"documents"}, "capability": {"manage-documents"}, "allowed": {"1"}}
	if result := authedFormRequest(t, a, "manager@example.com", "/demo/app/verwaltung/rechte", form); result.Code != http.StatusForbidden {
		t.Fatalf("manager edited matrix: %d", result.Code)
	}
	if result := authedFormRequest(t, a, "admin@example.com", "/demo/app/verwaltung/rechte", form); result.Code != http.StatusSeeOther {
		t.Fatalf("save: %d %s", result.Code, result.Body.String())
	}
	actor := a.actorFor("resident@example.com", "demo", roleResident)
	if !can(actor, capabilityManageDocuments, resourceFor("demo")) {
		t.Fatal("matrix change did not affect server authorization")
	}
	page = authedRequest(t, a, "admin@example.com", "/demo/app/verwaltung/rechte")
	if !strings.Contains(page.Body.String(), "Abweichend vom Standard") {
		t.Fatal("missing override marker")
	}
	audit := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: store.AuditActionCapabilityOverride})
	if len(audit) != 1 || audit[0].Details["family"] != "bewohner" || audit[0].Details["area"] != "documents" || audit[0].Details["before"] != "false" || audit[0].Details["after"] != "true" {
		t.Fatalf("audit: %+v", audit)
	}
	history := authedRequest(t, a, "admin@example.com", "/demo/app/audit")
	if !strings.Contains(history.Body.String(), "Rollenrecht geändert") {
		t.Fatal("audit missing from history")
	}
	// Protected capabilities cannot be smuggled into a legitimate form.
	for _, capability := range []string{"manage-users", "platform-admin"} {
		form.Set("capability", capability)
		form.Set("area", "additional")
		if result := authedFormRequest(t, a, "admin@example.com", "/demo/app/verwaltung/rechte", form); result.Code != http.StatusBadRequest {
			t.Fatalf("protected %s accepted", capability)
		}
	}
	if result := authedFormRequest(t, a, "admin@example.com", "/demo/app/verwaltung/rechte", url.Values{"operation": {"reset"}}); result.Code != http.StatusBadRequest {
		t.Fatal("reset without confirmation accepted")
	}
	if result := authedFormRequest(t, a, "admin@example.com", "/demo/app/verwaltung/rechte", url.Values{"operation": {"reset"}, "confirm": {"yes"}}); result.Code != http.StatusSeeOther {
		t.Fatalf("reset: %d", result.Code)
	}
	actor = a.actorFor("resident@example.com", "demo", roleResident)
	if can(actor, capabilityManageDocuments, resourceFor("demo")) {
		t.Fatal("reset did not restore defaults")
	}
	if events := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: store.AuditActionCapabilityReset}); len(events) != 1 {
		t.Fatal("reset audit missing")
	}
}

func TestUserRightsProfilesAndProfileVisibilityHAUSV699(t *testing.T) {
	a := rightsTestApp(t)
	path := "/demo/app/settings/users/rights"
	form := url.Values{"operation": {"grant"}, "email": {"resident@example.com"}, "capability": {"manage-documents"}, "effect": {"grant"}, "tenant_slug": {"demo"}}
	if result := authedFormRequest(t, a, "manager@example.com", path, form); result.Code != http.StatusForbidden {
		t.Fatal("manager changed own rights")
	}
	for _, capability := range []string{"manage-users", "platform-admin"} {
		form.Set("capability", capability)
		if result := authedFormRequest(t, a, "admin@example.com", path, form); result.Code != http.StatusBadRequest {
			t.Fatal("protected user grant accepted")
		}
	}
	form.Set("capability", "manage-documents")
	foreign := a.tenants["demo"]
	foreign.Slug = "foreign"
	foreign.Organisation = "another-org"
	a.tenants["foreign"] = foreign
	target := a.profiles["resident@example.com"]
	target.Tenants = append(target.Tenants, "foreign")
	a.profiles[target.Email] = target
	form.Set("tenant_slug", "foreign")
	if result := authedFormRequest(t, a, "admin@example.com", path, form); result.Code != http.StatusBadRequest {
		t.Fatal("foreign tenant scope accepted")
	}
	form.Set("tenant_slug", "demo")
	if result := authedFormRequest(t, a, "admin@example.com", path, form); result.Code != http.StatusSeeOther {
		t.Fatalf("grant: %d %s", result.Code, result.Body.String())
	}
	if result := authedRequest(t, a, "resident@example.com", "/demo/app/dokumente"); result.Code != http.StatusOK {
		t.Fatalf("grantee documents: %d", result.Code)
	}
	actor := a.actorFor("resident@example.com", "demo", roleResident)
	if !can(actor, capabilityManageDocuments, resourceFor("demo")) {
		t.Fatal("personal grant not applied")
	}
	users := authedRequest(t, a, "admin@example.com", "/demo/app/settings/users")
	for _, want := range []string{"Eigene Rechte", "Profil anwenden", "Gewährt"} {
		if !strings.Contains(users.Body.String(), want) {
			t.Errorf("users page missing %s", want)
		}
	}
	profile := authedRequest(t, a, "resident@example.com", "/demo/app/settings/profile")
	if !strings.Contains(profile.Body.String(), "Meine Berechtigungen") || !regexp.MustCompile(`Dokumente und Übergaben verwalten</strong>\s*<span>Erlaubt</span>`).MatchString(profile.Body.String()) {
		t.Fatal("effective rights missing from own profile")
	}
	save := url.Values{"operation": {"save-profile"}, "email": {"resident@example.com"}, "tenant_slug": {"demo"}, "profile_name": {"Dokumente"}}
	if result := authedFormRequest(t, a, "admin@example.com", path, save); result.Code != http.StatusSeeOther {
		t.Fatalf("save profile: %d", result.Code)
	}
	apply := url.Values{"operation": {"apply-profile"}, "email": {"resident@example.com"}, "tenant_slug": {"demo"}, "profile": {"Standard"}}
	if result := authedFormRequest(t, a, "admin@example.com", path, apply); result.Code != http.StatusSeeOther {
		t.Fatal("apply Standard failed")
	}
	if can(a.actorFor("resident@example.com", "demo", roleResident), capabilityManageDocuments, resourceFor("demo")) {
		t.Fatal("Standard retained grant")
	}
	apply.Set("profile", "Dokumente")
	if result := authedFormRequest(t, a, "admin@example.com", path, apply); result.Code != http.StatusSeeOther {
		t.Fatal("apply named profile failed")
	}
	form.Set("effect", "deny")
	form.Set("tenant_slug", "")
	if result := authedFormRequest(t, a, "admin@example.com", path, form); result.Code != http.StatusSeeOther {
		t.Fatal("deny failed")
	}
	if can(a.actorFor("resident@example.com", "demo", roleResident), capabilityManageDocuments, resourceFor("demo")) {
		t.Fatal("global deny did not beat scoped grant")
	}
	if events := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: store.AuditActionUserCapability}); len(events) != 4 {
		t.Fatalf("user changes audit count = %d, want 4", len(events))
	}
	if events := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: store.AuditActionCapabilityProfile}); len(events) != 1 {
		t.Fatal("profile audit missing")
	}
	form.Set("email", "missing@example.com")
	if result := authedFormRequest(t, a, "admin@example.com", path, form); result.Code != http.StatusNotFound {
		t.Fatal("unknown user accepted")
	}
}

func TestConfiguredDeniesReachRoutesHAUSV699(t *testing.T) {
	a := rightsTestApp(t)
	if err := a.capabilityRepo("org").SetGrant(t.Context(), store.UserCapabilityGrant{Email: "manager@example.com", Capability: "manage-announcements", Effect: "deny", UpdatedBy: "admin@example.com"}); err != nil {
		t.Fatal(err)
	}
	response := authedFormRequest(t, a, "manager@example.com", "/demo/app/announcements", url.Values{"title": {"Test"}, "body": {"Inhalt"}})
	if response.Code != http.StatusForbidden {
		t.Fatalf("revoked manager could publish: %d", response.Code)
	}
	// Energy has historical assignment checks; explicit decisions must precede them.
	ac := authCtx{tenant: a.tenants["demo"], email: "manager@example.com", role: roleManager}
	if err := a.capabilityRepo("org").SetGrant(t.Context(), store.UserCapabilityGrant{Email: ac.email, Capability: "control-energy", Effect: "deny", UpdatedBy: "admin@example.com"}); err != nil {
		t.Fatal(err)
	}
	ac.policy = a.capabilityPolicy(t.Context(), ac.tenant, false)
	if a.canControlEnergy(ac) {
		t.Fatal("legacy energy role bypassed explicit deny")
	}
	preview := a.capabilityPolicy(t.Context(), ac.tenant, true)
	if len(preview.Rules.Grants) != 0 {
		t.Fatal("personal rights leaked into preview policy")
	}
}

func TestRechteOptimisticPostAndRequestGuardsHAUSV699(t *testing.T) {
	a := rightsTestApp(t)
	form := url.Values{"operation": {"set"}, "family": {"bewohner"}, "area": {"documents"}, "capability": {"manage-documents"}, "allowed": {"1"}}
	cookie := rolePreviewTestSession(t, a, "admin@example.com", roleAdmin)
	request := httptest.NewRequest(http.MethodPost, "http://hausv.org/demo/app/verwaltung/rechte", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "http://hausv.org")
	request.Header.Set("Accept", "application/json")
	request.AddCookie(cookie)
	result := httptest.NewRecorder()
	a.handler().ServeHTTP(result, request)
	var response struct{ Saved, Changed bool }
	if result.Code != http.StatusOK || json.Unmarshal(result.Body.Bytes(), &response) != nil || !response.Saved || !response.Changed {
		t.Fatalf("optimistic response: %d %s", result.Code, result.Body.String())
	}
	if rejected := authedFormRequestWithOrigin(t, a, "admin@example.com", "/demo/app/verwaltung/rechte", form, "https://foreign.example"); rejected.Code != http.StatusForbidden {
		t.Fatal("cross-origin change accepted")
	}
	start := rolePreviewTestRequest(t, a, http.MethodPost, "/demo/app/ansicht/start", url.Values{"role": {roleOwner}}, cookie)
	if start.Code != http.StatusSeeOther {
		t.Fatal("preview setup failed")
	}
	previewCookie := rolePreviewResponseCookie(t, start)
	if rejected := rolePreviewTestRequest(t, a, http.MethodPost, "/demo/app/verwaltung/rechte", form, previewCookie); rejected.Code != http.StatusForbidden {
		t.Fatal("preview edited matrix")
	}
	if rejected := rolePreviewTestRequest(t, a, http.MethodPost, "/demo/app/settings/users/rights", url.Values{"operation": {"grant"}, "email": {"resident@example.com"}, "capability": {"vote"}, "effect": {"grant"}}, previewCookie); rejected.Code != http.StatusForbidden {
		t.Fatal("preview edited personal rights")
	}
}

// HAUSV-699: switching off a delegable right of its own family must never lock
// the organisation administration out of the Verwaltung area and the rights page.
func TestRechteAdminKeepsVerwaltungAreaAfterSelfRestriction(t *testing.T) {
	a := rightsTestApp(t)
	form := url.Values{"operation": {"set"}, "family": {"verwaltung"}, "area": {"portfolio"}, "capability": {string(capabilityManageIssues)}}
	if result := authedFormRequest(t, a, "admin@example.com", "/demo/app/verwaltung/rechte", form); result.Code != http.StatusSeeOther {
		t.Fatalf("save: %d %s", result.Code, result.Body.String())
	}
	page := authedRequest(t, a, "admin@example.com", "/demo/app/verwaltung/rechte")
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), `data-rights-switch`) {
		t.Fatalf("admin lost the rights page after restricting the Verwaltung family: %d", page.Code)
	}
	// The Verwalter keeps the area through its non-delegable administration
	// right, but the switched-off Anliegen right is really gone.
	manager := a.actorFor("manager@example.com", "demo", roleManager)
	if can(manager, capabilityManageIssues, resourceFor("demo")) || !can(manager, capabilityManageUsers, resourceFor("demo")) {
		t.Fatal("family override did not apply to the Verwalter as expected")
	}
	if area := authedRequest(t, a, "manager@example.com", "/demo/app/verwaltung"); area.Code != http.StatusOK {
		t.Fatalf("Verwalter lost the Verwaltung area: %d", area.Code)
	}
	oversight := url.Values{"operation": {"set"}, "family": {"verwaltung"}, "area": {"portfolio"}, "capability": {string(capabilityOversight)}}
	if result := authedFormRequest(t, a, "admin@example.com", "/demo/app/verwaltung/rechte", oversight); result.Code == http.StatusSeeOther {
		t.Fatal("locked Oversight of the Verwaltung family must be rejected")
	}
}
