package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
)

func newPortalContextTestApp(t *testing.T) *app {
	t.Helper()
	a := newTestPortalApp(t, userProfile{
		Email:     "multi@example.com",
		FirstName: "Mara",
		LastName:  "Mehrhaus",
		Role:      roleOwner,
		Tenants:   []string{"demo", "haus-b"},
		TenantMemberships: map[string]tenantMembership{
			"demo":   {Role: roleOwner},
			"haus-b": {Role: roleRenter},
		},
		AuthMethods: defaultAuthMethods(),
	})
	a.tenants["haus-b"] = tenantConfig{Slug: "haus-b", Name: "Haus B", Address: "Nebenweg 2", MapLatitude: 47.0707, MapLongitude: 15.4395, MapZoom: 17}
	if err := testUnitRepository(t, a, "demo").SetUnits([]unit{
		{ID: "top-owner", Label: "Eigentum", OwnerEmails: []string{"multi@example.com"}},
		{ID: "top-renter", Label: "Miete", RenterEmails: []string{"multi@example.com"}},
	}); err != nil {
		t.Fatalf("SetTenantUnits: %v", err)
	}
	return a
}

func TestPortalContextListsActivatedDynamicHomeAfterSwitchingAway(t *testing.T) {
	const (
		email = "multi@example.com"
		slug  = "waldweg"
	)
	a := newPortalContextTestApp(t)
	now := time.Date(2026, time.August, 14, 20, 0, 0, 0, time.UTC)
	if _, err := a.homeReservations.Reserve(store.HomeReservation{
		Slug: slug, HouseholdName: "Waldweg 87", OwnerEmail: email, AuthorizationConfirmed: true,
	}, now); err != nil {
		t.Fatal(err)
	}
	if _, confirmed, err := a.homeReservations.Confirm(slug, email, now.Add(time.Minute)); err != nil || !confirmed {
		t.Fatalf("confirm confirmed=%v err=%v", confirmed, err)
	}
	if _, created, err := a.homePortals.Activate(slug, email, now.Add(2*time.Minute)); err != nil || !created {
		t.Fatalf("activate created=%v err=%v", created, err)
	}

	contexts := a.portalContextsFor(email, "demo", roleOwner)
	found := false
	for _, context := range contexts {
		if context.TenantSlug == slug && context.HouseName == "Waldweg 87" && context.Role == roleOwner {
			found = true
		}
	}
	if !found {
		t.Fatalf("dynamic home missing after switching away: %+v", contexts)
	}
}

func portalContextPost(t *testing.T, a *app, token string, values url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "http://hausv.org/demo/app/context", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://hausv.org/demo")
	req.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	rr := httptest.NewRecorder()
	a.handler().ServeHTTP(rr, req)
	return rr
}

func sessionCookieFrom(t *testing.T, rr *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, cookie := range rr.Result().Cookies() {
		if cookie.Name == "weg_session" {
			return cookie
		}
	}
	t.Fatal("response did not issue a session cookie")
	return nil
}

func TestPortalContextSwitchChangesTenantWithoutExtendingLogin(t *testing.T) {
	a := newPortalContextTestApp(t)
	expiresAt := time.Now().Add(40 * time.Minute).Truncate(time.Second)
	oldToken, _, err := a.sessions.PutSession("multi@example.com", "demo", authMethodEmail, roleOwner, expiresAt)
	if err != nil {
		t.Fatalf("PutSession: %v", err)
	}

	rr := portalContextPost(t, a, oldToken, url.Values{"tenant": {"haus-b"}, "role": {roleRenter}})
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("switch status = %d, want 303: %s", rr.Code, rr.Body.String())
	}
	if got, want := rr.Header().Get("Location"), "http://localhost:8080/haus-b/app"; got != want {
		t.Fatalf("location = %q, want %q", got, want)
	}
	newCookie := sessionCookieFrom(t, rr)
	context, ok := a.sessions.GetSession(newCookie.Value)
	if !ok || context.Email != "multi@example.com" || context.TenantSlug != "haus-b" || context.Role != roleRenter || context.ExpiresAt != expiresAt.Unix() {
		t.Fatalf("new context ok=%v context=%+v", ok, context)
	}
	if _, ok := a.sessions.GetSession(oldToken); ok {
		t.Fatal("the replaced session must be revoked")
	}

	req := httptest.NewRequest(http.MethodGet, "http://hausv.org/haus-b/app", nil)
	req.AddCookie(newCookie)
	page := httptest.NewRecorder()
	a.handler().ServeHTTP(page, req)
	targetTile := sidebarMapForTenant(a.tenants["haus-b"]).Tiles[0].URL
	sourceTile := sidebarMapForTenant(a.tenants["demo"]).Tiles[0].URL
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "Haus B") || !strings.Contains(page.Body.String(), roleRenter) || !strings.Contains(page.Body.String(), targetTile) {
		t.Fatalf("target portal status=%d body=%s", page.Code, page.Body.String())
	}
	if sourceTile != targetTile && strings.Contains(page.Body.String(), sourceTile) {
		t.Fatal("target portal must not keep the previous tenant map tiles")
	}
	if strings.Contains(page.Body.String(), "Standort nicht hinterlegt") {
		t.Fatal("target portal with coordinates must render its map instead of the fallback")
	}
	events := a.auditStore.List(auditFilter{TenantSlug: "haus-b", Action: auditActionContextSwitch, Limit: 10})
	if len(events) != 1 || events[0].ActorEmail != "multi@example.com" || events[0].ActorRole != roleRenter {
		t.Fatalf("context-switch audit = %+v", events)
	}
}

func TestPortalContextSwitchRejectsForeignTenantAndRoleElevation(t *testing.T) {
	a := newPortalContextTestApp(t)
	for _, values := range []url.Values{
		{"tenant": {"foreign"}, "role": {roleRenter}},
		{"tenant": {"haus-b"}, "role": {roleAdmin}},
	} {
		token, _, err := a.sessions.Put("multi@example.com", "demo", authMethodEmail, time.Hour)
		if err != nil {
			t.Fatalf("Put: %v", err)
		}
		rr := portalContextPost(t, a, token, values)
		if rr.Code != http.StatusForbidden {
			t.Fatalf("values=%v status=%d, want 403", values, rr.Code)
		}
		if _, ok := a.sessions.GetSession(token); !ok {
			t.Fatal("rejected switch must not revoke the valid session")
		}
		if len(rr.Result().Cookies()) != 0 {
			t.Fatal("rejected switch must not issue a cookie")
		}
	}
}

func TestUserCanChooseOwnAssignedRenterRoleButNotElevate(t *testing.T) {
	a := newPortalContextTestApp(t)
	token, _, err := a.sessions.Put("multi@example.com", "demo", authMethodEmail, time.Hour)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	rr := portalContextPost(t, a, token, url.Values{"tenant": {"demo"}, "role": {roleRenter}})
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("posture status = %d: %s", rr.Code, rr.Body.String())
	}
	newCookie := sessionCookieFrom(t, rr)
	req := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/app", nil)
	req.AddCookie(newCookie)
	_, role, tenantSlug, ok := a.currentUser(req)
	if !ok || role != roleRenter || tenantSlug != "demo" {
		t.Fatalf("current user ok=%v tenant=%q role=%q", ok, tenantSlug, role)
	}

	forged, _, err := a.sessions.PutSession("multi@example.com", "haus-b", authMethodEmail, roleAdmin, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("signed test session: %v", err)
	}
	forgedReq := httptest.NewRequest(http.MethodGet, "http://hausv.org/haus-b/app", nil)
	forgedReq.AddCookie(&http.Cookie{Name: "weg_session", Value: forged})
	if _, _, _, ok := a.currentUser(forgedReq); ok {
		t.Fatal("a signed but unassigned role must still be rejected server-side")
	}
}

func TestPortalContextRejectsUnassignedRenterRole(t *testing.T) {
	a := newPortalContextTestApp(t)
	if err := testUnitRepository(t, a, "demo").SetUnits([]unit{{ID: "top-owner", Label: "Eigentum", OwnerEmails: []string{"multi@example.com"}}}); err != nil {
		t.Fatalf("SetTenantUnits: %v", err)
	}
	token, _, err := a.sessions.Put("multi@example.com", "demo", authMethodEmail, time.Hour)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	rr := portalContextPost(t, a, token, url.Values{"tenant": {"demo"}, "role": {roleRenter}})
	if rr.Code != http.StatusForbidden {
		t.Fatalf("unassigned role status = %d, want 403", rr.Code)
	}
	if contexts := a.portalContextsFor("multi@example.com", "demo", roleOwner); len(contexts) != 2 {
		t.Fatalf("actual contexts = %+v, want demo owner and haus-b renter", contexts)
	}
}

func TestPortalContextSwitcherListsOnlyOwnedContexts(t *testing.T) {
	a := newPortalContextTestApp(t)
	a.tenants["foreign"] = tenantConfig{Slug: "foreign", Name: "Fremdes Haus", Address: "Fremdweg 9"}
	page := authedRequest(t, a, "multi@example.com", "/demo/app")
	if page.Code != http.StatusOK {
		t.Fatalf("portal status = %d", page.Code)
	}
	body := page.Body.String()
	for _, want := range []string{"Portal wechseln", "Musterweg 1", "Haus B", roleOwner, roleRenter, `action="/demo/app/context"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("switcher missing %q", want)
		}
	}
	if strings.Contains(body, "Fremdes Haus") {
		t.Fatal("switcher must not disclose an unassigned tenant")
	}
}

func TestPortalContextSwitcherOmitsUnavailableOwnedTenant(t *testing.T) {
	a := newPortalContextTestApp(t)
	profile := a.profiles["multi@example.com"]
	profile.Deactivated = true
	a.profiles["multi@example.com"] = profile

	if contexts := a.portalContextsFor("multi@example.com", "demo", roleOwner); len(contexts) != 0 {
		t.Fatalf("unavailable contexts = %+v, want none", contexts)
	}
}
