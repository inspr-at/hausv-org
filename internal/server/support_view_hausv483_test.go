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

func newSupportViewTestApp(t *testing.T, supportPermission bool) *app {
	t.Helper()
	permissions := []string{}
	if supportPermission {
		permissions = append(permissions, permissionSupportView)
	}
	a := newTestPortalApp(t, userProfile{
		Email: "admin@example.com", FirstName: "Ada", LastName: "Admin", Role: roleAdmin,
		Tenants: []string{"demo"}, Permissions: permissions, AuthMethods: defaultAuthMethods(),
	})
	a.profiles["resident@example.com"] = userProfile{
		Email: "resident@example.com", FirstName: "Rita", LastName: "Resident", Role: roleResident,
		Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods(), Status: "Aktiv",
	}
	return a
}

func supportViewPost(t *testing.T, a *app, token string, path string, values url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "http://hausv.org/demo"+path, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://hausv.org/demo")
	req.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	rr := httptest.NewRecorder()
	a.handler().ServeHTTP(rr, req)
	return rr
}

func startSupportViewForResident(t *testing.T, a *app) (string, int64) {
	t.Helper()
	parentExpiry := time.Now().Add(50 * time.Minute).Truncate(time.Second)
	oldToken, _, err := a.sessions.PutSession("admin@example.com", "demo", authMethodEmail, roleAdmin, parentExpiry)
	if err != nil {
		t.Fatal(err)
	}
	rr := supportViewPost(t, a, oldToken, "/app/support-view/start", url.Values{
		"tenant": {"demo"}, "target_email": {"resident@example.com"}, "target_role": {roleResident},
	})
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("start status = %d: %s", rr.Code, rr.Body.String())
	}
	cookie := sessionCookieFrom(t, rr)
	if _, ok := a.sessions.GetSession(oldToken); ok {
		t.Fatal("starting support view must revoke the ordinary session")
	}
	return cookie.Value, parentExpiry.Unix()
}

func TestSupportViewRequiresSeparatePermissionAndExactTargetRole(t *testing.T) {
	withoutPermission := newSupportViewTestApp(t, false)
	token, _, err := withoutPermission.sessions.PutSession("admin@example.com", "demo", authMethodEmail, roleAdmin, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	denied := supportViewPost(t, withoutPermission, token, "/app/support-view/start", url.Values{
		"tenant": {"demo"}, "target_email": {"resident@example.com"}, "target_role": {roleResident},
	})
	if denied.Code != http.StatusForbidden || len(denied.Result().Cookies()) != 0 {
		t.Fatalf("missing permission status=%d cookies=%v", denied.Code, denied.Result().Cookies())
	}

	withPermission := newSupportViewTestApp(t, true)
	for name, values := range map[string]url.Values{
		"role elevation": {"tenant": {"demo"}, "target_email": {"resident@example.com"}, "target_role": {roleAdmin}},
		"tenant tamper":  {"tenant": {"other"}, "target_email": {"resident@example.com"}, "target_role": {roleResident}},
		"unknown user":   {"tenant": {"demo"}, "target_email": {"outsider@example.com"}, "target_role": {roleResident}},
	} {
		t.Run(name, func(t *testing.T) {
			token, _, err := withPermission.sessions.PutSession("admin@example.com", "demo", authMethodEmail, roleAdmin, time.Now().Add(time.Hour))
			if err != nil {
				t.Fatal(err)
			}
			rr := supportViewPost(t, withPermission, token, "/app/support-view/start", values)
			if rr.Code != http.StatusForbidden || len(rr.Result().Cookies()) != 0 {
				t.Fatalf("status=%d cookies=%v body=%s", rr.Code, rr.Result().Cookies(), rr.Body.String())
			}
		})
	}
}

func TestSupportViewChooserIsVisibleOnlyWithExplicitPermission(t *testing.T) {
	withPermission := newSupportViewTestApp(t, true)
	page := authedRequest(t, withPermission, "admin@example.com", "/demo/app/settings/users")
	if page.Code != http.StatusOK {
		t.Fatalf("chooser page status=%d", page.Code)
	}
	for _, want := range []string{"data-support-view-start", "Rita Resident", `name="target_role" value="Bewohner"`, "15 Minuten · schreibgeschützt"} {
		if !strings.Contains(page.Body.String(), want) {
			t.Fatalf("chooser missing %q", want)
		}
	}

	withoutPermission := newSupportViewTestApp(t, false)
	page = authedRequest(t, withoutPermission, "admin@example.com", "/demo/app/settings/users")
	if page.Code != http.StatusOK || strings.Contains(page.Body.String(), "data-support-view-start") {
		t.Fatalf("chooser without permission status=%d visible=%v", page.Code, strings.Contains(page.Body.String(), "data-support-view-start"))
	}
}

func TestSupportViewUsesTargetPermissionsBlocksWritesAndExitsSafely(t *testing.T) {
	a := newSupportViewTestApp(t, true)
	token, parentExpiry := startSupportViewForResident(t, a)
	session, ok := a.sessions.GetSession(token)
	if !ok || session.Email != "admin@example.com" || session.Role != roleAdmin || session.SupportTargetEmail != "resident@example.com" || session.SupportTargetRole != roleResident {
		t.Fatalf("support session = %+v ok=%v", session, ok)
	}
	if session.ExpiresAt != parentExpiry || session.SupportExpiresAt > time.Now().Add(supportViewTTL+time.Second).Unix() {
		t.Fatalf("support expiry=%d parent=%d", session.SupportExpiresAt, session.ExpiresAt)
	}

	pageReq := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/app", nil)
	pageReq.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	page := httptest.NewRecorder()
	a.handler().ServeHTTP(page, pageReq)
	body := page.Body.String()
	for _, want := range []string{"Portal anzeigen als Rita Resident", roleResident, "Supportansicht beenden", "data-support-view-banner"} {
		if !strings.Contains(body, want) {
			t.Fatalf("support portal missing %q", want)
		}
	}
	if strings.Contains(body, ">Benutzer &amp; Rechte<") {
		t.Fatal("support portal leaked admin user-management navigation")
	}
	announcementsReq := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/app/announcements", nil)
	announcementsReq.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	announcementsResult := httptest.NewRecorder()
	a.handler().ServeHTTP(announcementsResult, announcementsReq)
	if announcementsResult.Code != http.StatusOK {
		t.Fatalf("support announcements status=%d", announcementsResult.Code)
	}
	reads, _ := store.BindAnnouncementReadRepository(a.announcementReadStore, testTenantRef("demo"))
	if !reads.LastSeen("resident@example.com").IsZero() {
		t.Fatal("support view changed the target user's unread state")
	}

	before := len(testRepositories(a, "demo").announcements.List())
	write := supportViewPost(t, a, token, "/app/announcements", url.Values{"title": {"Forbidden"}, "body": {"No write"}})
	if write.Code != http.StatusForbidden || len(testRepositories(a, "demo").announcements.List()) != before {
		t.Fatalf("support write status=%d announcements=%d", write.Code, len(testRepositories(a, "demo").announcements.List()))
	}

	exit := supportViewPost(t, a, token, "/app/support-view/end", nil)
	if exit.Code != http.StatusSeeOther {
		t.Fatalf("exit status=%d body=%s", exit.Code, exit.Body.String())
	}
	restoredCookie := sessionCookieFrom(t, exit)
	restored, ok := a.sessions.GetSession(restoredCookie.Value)
	if !ok || restored.Email != "admin@example.com" || restored.Role != roleAdmin || restored.SupportTargetEmail != "" || restored.ExpiresAt != parentExpiry {
		t.Fatalf("restored session=%+v ok=%v", restored, ok)
	}
	if _, ok := a.sessions.GetSession(token); ok {
		t.Fatal("support token remains valid after exit")
	}
	starts := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionSupportViewStart, Limit: 10})
	ends := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionSupportViewEnd, Limit: 10})
	if len(starts) != 1 || len(ends) != 1 || starts[0].ActorEmail != "admin@example.com" || starts[0].TargetID != "resident@example.com" || ends[0].Details["reason"] != "explicit" {
		t.Fatalf("support audit starts=%+v ends=%+v", starts, ends)
	}
}

func TestExpiredSupportViewRestoresAdminAndAuditsEndBeforeAnyPage(t *testing.T) {
	a := newSupportViewTestApp(t, true)
	now := time.Now().UTC().Truncate(time.Second)
	parentExpiry := now.Add(time.Hour)
	token, _, err := a.sessions.PutSupportView("admin@example.com", "demo", authMethodEmail, roleAdmin, "resident@example.com", roleResident, now.Add(-2*time.Minute), now.Add(-time.Minute), parentExpiry)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/app", nil)
	req.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	rr := httptest.NewRecorder()
	a.handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expired support status=%d body=%s", rr.Code, rr.Body.String())
	}
	restoredCookie := sessionCookieFrom(t, rr)
	restored, ok := a.sessions.GetSession(restoredCookie.Value)
	if !ok || restored.Email != "admin@example.com" || restored.Role != roleAdmin || restored.SupportTargetEmail != "" || restored.ExpiresAt != parentExpiry.Unix() {
		t.Fatalf("expired restored session=%+v ok=%v", restored, ok)
	}
	if _, ok := a.sessions.GetSession(token); ok {
		t.Fatal("expired support token remains usable")
	}
	ends := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionSupportViewEnd, Limit: 10})
	if len(ends) != 1 || ends[0].Details["reason"] != "expired" || ends[0].ActorEmail != "admin@example.com" {
		t.Fatalf("expired support audit=%+v", ends)
	}
}

func TestSupportViewCannotBeTransferredByURLOrManipulatedCookie(t *testing.T) {
	a := newSupportViewTestApp(t, true)
	token, _ := startSupportViewForResident(t, a)

	withoutCookie := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/app", nil)
	withoutCookieResult := httptest.NewRecorder()
	a.handler().ServeHTTP(withoutCookieResult, withoutCookie)
	if withoutCookieResult.Code != http.StatusSeeOther {
		t.Fatalf("copied URL status=%d", withoutCookieResult.Code)
	}

	last := token[len(token)-1]
	replacement := byte('A')
	if last == replacement {
		replacement = 'B'
	}
	forged := token[:len(token)-1] + string(replacement)
	forgedRequest := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/app", nil)
	forgedRequest.AddCookie(&http.Cookie{Name: "weg_session", Value: forged})
	forgedResult := httptest.NewRecorder()
	a.handler().ServeHTTP(forgedResult, forgedRequest)
	if forgedResult.Code != http.StatusSeeOther {
		t.Fatalf("manipulated cookie status=%d", forgedResult.Code)
	}
}

func TestLogoutEndsSupportViewAndAuditsTheRealAdmin(t *testing.T) {
	a := newSupportViewTestApp(t, true)
	token, _ := startSupportViewForResident(t, a)
	logout := supportViewPost(t, a, token, "/auth/logout", nil)
	if logout.Code != http.StatusSeeOther {
		t.Fatalf("logout status=%d", logout.Code)
	}
	if _, ok := a.sessions.GetSession(token); ok {
		t.Fatal("support token remains valid after logout")
	}
	ends := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionSupportViewEnd, Limit: 10})
	if len(ends) != 1 || ends[0].ActorEmail != "admin@example.com" || ends[0].ActorRole != roleAdmin || ends[0].TargetID != "resident@example.com" || ends[0].Details["reason"] != "logout" {
		t.Fatalf("logout support audit=%+v", ends)
	}
}
