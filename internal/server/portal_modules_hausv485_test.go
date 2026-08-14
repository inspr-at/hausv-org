package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPortalModuleStorePersistsAndBuildingMetaKeepsSelection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tenants.json")
	store, err := newTenantOverrideStore(path)
	if err != nil {
		t.Fatalf("newTenantOverrideStore: %v", err)
	}
	if err := store.SetDisabledModules("demo", []string{"events", "events", "unknown", "parking"}); err != nil {
		t.Fatalf("SetDisabledModules: %v", err)
	}
	if err := store.SetMeta("demo", tenantOverride{Name: "Haus", Address: "Weg 1"}); err != nil {
		t.Fatalf("SetMeta: %v", err)
	}

	reopened, err := newTenantOverrideStore(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	override, ok := reopened.Get("demo")
	if !ok || strings.Join(override.DisabledModules, ",") != "events,parking" {
		t.Fatalf("persisted override = %+v", override)
	}
}

func TestPortalModulesHideNavigationContentAndRoutes(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email: "admin@example.com", FirstName: "Ada", LastName: "Admin", Role: roleAdmin,
		Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods(),
	})
	token, _, err := a.sessions.Put("admin@example.com", "demo", authMethodEmail, time.Hour)
	if err != nil {
		t.Fatalf("session: %v", err)
	}
	cookie := &http.Cookie{Name: "weg_session", Value: token}

	settingsReq := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/app/settings/modules", nil)
	settingsReq.AddCookie(cookie)
	settings := httptest.NewRecorder()
	a.handler().ServeHTTP(settings, settingsReq)
	if settings.Code != http.StatusOK || !strings.Contains(settings.Body.String(), "Hausüberblick") || !strings.Contains(settings.Body.String(), "Immer aktiv") {
		t.Fatalf("module settings status=%d body=%s", settings.Code, settings.Body.String())
	}

	form := url.Values{"modules": {"announcements"}}
	postReq := httptest.NewRequest(http.MethodPost, "http://hausv.org/demo/app/settings/modules", strings.NewReader(form.Encode()))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.Header.Set("Origin", "http://hausv.org")
	postReq.AddCookie(cookie)
	post := httptest.NewRecorder()
	a.handler().ServeHTTP(post, postReq)
	if post.Code != http.StatusSeeOther {
		t.Fatalf("save status=%d body=%s", post.Code, post.Body.String())
	}
	flags := a.portalModulesFor("demo")
	if !flags.Announcements || flags.Events || flags.Parking || flags.Help {
		t.Fatalf("saved flags = %+v", flags)
	}
	audit := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionPortalModulesUpdate, Limit: 10})
	if len(audit) != 1 || audit[0].ActorEmail != "admin@example.com" {
		t.Fatalf("module audit = %+v", audit)
	}

	for _, path := range []string{"/demo/app/events", "/demo/app/parking", "/demo/app/hilfe", "/demo/app/settings/users"} {
		req := httptest.NewRequest(http.MethodGet, "http://hausv.org"+path, nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		a.handler().ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Errorf("GET %s status=%d, want 404", path, rr.Code)
		}
	}

	homeReq := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/app", nil)
	homeReq.AddCookie(cookie)
	home := httptest.NewRecorder()
	a.handler().ServeHTTP(home, homeReq)
	if home.Code != http.StatusOK {
		t.Fatalf("home status=%d", home.Code)
	}
	body := home.Body.String()
	hasEvents := strings.Contains(body, `href="/demo/app/events"`)
	hasParking := strings.Contains(body, `href="/demo/app/parking"`)
	hasSettings := strings.Contains(body, `href="/demo/app/settings"`)
	if hasEvents || hasParking || !hasSettings {
		t.Fatalf("home navigation does not reflect module settings: events=%v parking=%v settings=%v", hasEvents, hasParking, hasSettings)
	}
}

func TestPortalModuleForPathKeepsCoreSettingsAvailable(t *testing.T) {
	for _, path := range []string{"/app", "/app/settings", "/app/settings/building", "/app/settings/modules", "/app/settings/profile"} {
		if module, managed := portalModuleForPath(path); managed {
			t.Errorf("core path %s unexpectedly mapped to %s", path, module)
		}
	}
	if module, managed := portalModuleForPath("/app/settings/energy-data/export"); !managed || module != portalModuleEnergy {
		t.Fatalf("energy settings mapping = %q, %v", module, managed)
	}
}
