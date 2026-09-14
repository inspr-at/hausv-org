package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/config"
	"github.com/inspr-at/hausv-org/internal/store"
)

func TestRolePreviewAdminLifecycleIsReadOnlyHAUSV605(t *testing.T) {
	const email = "admin@example.com"
	a := newTestPortalApp(t, userProfile{Email: email, FirstName: "Ada", LastName: "Admin", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	adminCookie := rolePreviewTestSession(t, a, email, roleAdmin)

	start := rolePreviewTestRequest(t, a, http.MethodPost, "/demo/app/ansicht/start", url.Values{"role": {roleOwner}}, adminCookie)
	if start.Code != http.StatusSeeOther {
		t.Fatalf("start status = %d, want 303: %s", start.Code, start.Body.String())
	}
	previewCookie := rolePreviewResponseCookie(t, start)
	page := rolePreviewTestRequest(t, a, http.MethodGet, "/demo/app", nil, previewCookie)
	if page.Code != http.StatusOK {
		t.Fatalf("preview page status = %d: %s", page.Code, page.Body.String())
	}
	body := page.Body.String()
	for _, want := range []string{`data-preview-role="Eigentümer"`, "Vorschau: Eigentümer-Sicht", "Schreibgeschützt", "calm-main"} {
		if !strings.Contains(body, want) {
			t.Errorf("preview page missing %q", want)
		}
	}
	if hasUsers, hasGreeting := strings.Contains(body, `href="/demo/app/settings/users"`), strings.Contains(body, "Ada Admin."); hasUsers || hasGreeting {
		t.Fatalf("owner preview must not expose admin navigation or a personal greeting: users=%v greeting=%v", hasUsers, hasGreeting)
	}

	write := rolePreviewTestRequest(t, a, http.MethodPost, "/demo/app/anliegen", url.Values{"title": {"Darf nicht gespeichert werden"}}, previewCookie)
	if write.Code != http.StatusForbidden || !strings.Contains(write.Body.String(), rolePreviewReadOnlyMessage) {
		t.Fatalf("preview write status=%d body=%q", write.Code, write.Body.String())
	}

	end := rolePreviewTestRequest(t, a, http.MethodPost, "/demo/app/ansicht/ende", nil, previewCookie)
	if end.Code != http.StatusSeeOther {
		t.Fatalf("end status = %d, want 303: %s", end.Code, end.Body.String())
	}
	restoredCookie := rolePreviewResponseCookie(t, end)
	restored := rolePreviewTestRequest(t, a, http.MethodGet, "/demo/app", nil, restoredCookie)
	if restored.Code != http.StatusOK || strings.Contains(restored.Body.String(), `data-preview-role=`) || !strings.Contains(restored.Body.String(), `href="/demo/app/settings/users"`) {
		t.Fatalf("restored admin page is not normal: status=%d", restored.Code)
	}

	starts := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: store.AuditActionRolePreviewStart, Limit: 10})
	ends := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: store.AuditActionRolePreviewEnd, Limit: 10})
	if len(starts) != 1 || starts[0].Details["preview_role"] != roleOwner || starts[0].Details["tenant"] != "demo" || starts[0].Details["expires_at"] == "" {
		t.Fatalf("start audit = %+v", starts)
	}
	if len(ends) != 1 || ends[0].Details["reason"] != "explicit" {
		t.Fatalf("end audit = %+v", ends)
	}
}

func TestRolePreviewStartRequiresRealAdminHAUSV605(t *testing.T) {
	const email = "manager@example.com"
	a := newTestPortalApp(t, userProfile{Email: email, Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	cookie := rolePreviewTestSession(t, a, email, roleManager)
	response := rolePreviewTestRequest(t, a, http.MethodPost, "/demo/app/ansicht/start", url.Values{"role": {roleOwner}}, cookie)
	if response.Code != http.StatusForbidden {
		t.Fatalf("manager start status = %d, want 403", response.Code)
	}
	page := rolePreviewTestRequest(t, a, http.MethodGet, "/demo/app", nil, cookie)
	if strings.Contains(page.Body.String(), "role-preview-chooser") {
		t.Fatal("manager must not see the role preview chooser")
	}
}

func TestPortalIssueAssigneeUsesDisplayNameHAUSV605(t *testing.T) {
	const email = "vera@example.com"
	a := newTestPortalApp(t, userProfile{
		Email: email, FirstName: "Vera", LastName: "Verwalter", Role: roleAdmin,
		Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods(),
	})
	repository := testRequestRepositories(t, a, "demo").issues
	if _, err := repository.Create(store.ResidentIssue{
		AuthorEmail: "resident@example.com", AuthorName: "Resi Dent", Category: "Reparatur",
		Title: "Lift steckt wieder", Body: "Der Lift bleibt stehen.", LocationType: store.IssueLocationCommon,
		Status: store.IssueStatusNew, AssigneeEmail: email,
	}); err != nil {
		t.Fatalf("seed assigned issue: %v", err)
	}

	page := rolePreviewTestRequest(t, a, http.MethodGet, "/demo/app", nil, rolePreviewTestSession(t, a, email, roleAdmin))
	if page.Code != http.StatusOK {
		t.Fatalf("portal status = %d: %s", page.Code, page.Body.String())
	}
	if !strings.Contains(page.Body.String(), `<td title="vera@example.com">Vera Verwalter</td>`) {
		t.Fatalf("portal issue assignee did not use the profile display name: %s", page.Body.String())
	}
}

func TestRolePreviewExpiryRestoresSessionAndAuditsHAUSV605(t *testing.T) {
	const email = "admin@example.com"
	a := newTestPortalApp(t, userProfile{Email: email, Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	adminCookie := rolePreviewTestSession(t, a, email, roleAdmin)
	start := rolePreviewTestRequest(t, a, http.MethodPost, "/demo/app/ansicht/start", url.Values{"role": {roleResident}}, adminCookie)
	previewCookie := rolePreviewResponseCookie(t, start)
	current, ok := a.sessions.GetSession(previewCookie.Value)
	if !ok {
		t.Fatal("preview session should verify")
	}
	now := time.Now().UTC().Truncate(time.Second)
	expiredToken, _, err := a.sessions.PutRolePreview(current, roleResident, now.Add(-20*time.Minute), now.Add(-time.Minute))
	if err != nil {
		t.Fatalf("create expired preview: %v", err)
	}
	expiredCookie := &http.Cookie{Name: "weg_session", Value: expiredToken}
	response := rolePreviewTestRequest(t, a, http.MethodGet, "/demo/app", nil, expiredCookie)
	if response.Code != http.StatusSeeOther {
		t.Fatalf("expired preview status = %d, want 303", response.Code)
	}
	restoredCookie := rolePreviewResponseCookie(t, response)
	restored, ok := a.sessions.GetSession(restoredCookie.Value)
	if !ok || restored.Role != roleAdmin || restored.PreviewRole != "" {
		t.Fatalf("restored session = %+v ok=%v", restored, ok)
	}
	starts := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: store.AuditActionRolePreviewStart, Limit: 10})
	ends := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: store.AuditActionRolePreviewEnd, Limit: 10})
	if len(starts) != 1 || len(ends) != 1 || ends[0].Details["reason"] != rolePreviewEndExpired {
		t.Fatalf("expiry audits start=%+v end=%+v", starts, ends)
	}
}

func TestRolePreviewEndsWhenAdminMembershipChangesHAUSV605(t *testing.T) {
	const email = "admin@example.com"
	a := newTestPortalApp(t, userProfile{Email: email, Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	start := rolePreviewTestRequest(t, a, http.MethodPost, "/demo/app/ansicht/start", url.Values{"role": {roleOwner}}, rolePreviewTestSession(t, a, email, roleAdmin))
	previewCookie := rolePreviewResponseCookie(t, start)
	profile := a.profiles[email]
	profile.Role = roleManager
	a.profiles[email] = profile

	response := rolePreviewTestRequest(t, a, http.MethodGet, "/demo/app", nil, previewCookie)
	if response.Code != http.StatusSeeOther {
		t.Fatalf("changed actor context status = %d, want 303", response.Code)
	}
	restoredCookie := rolePreviewResponseCookie(t, response)
	restored, ok := a.sessions.GetSession(restoredCookie.Value)
	if !ok || restored.Role != roleManager || restored.PreviewRole != "" {
		t.Fatalf("restored changed context = %+v ok=%v", restored, ok)
	}
	ends := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: store.AuditActionRolePreviewEnd, Limit: 10})
	if len(ends) != 1 || ends[0].Details["reason"] != rolePreviewEndActorContextInvalid {
		t.Fatalf("actor invalidation audit = %+v", ends)
	}
}

func TestRolePreviewLogoutRecordsEndHAUSV605(t *testing.T) {
	const email = "admin@example.com"
	a := newTestPortalApp(t, userProfile{Email: email, Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	start := rolePreviewTestRequest(t, a, http.MethodPost, "/demo/app/ansicht/start", url.Values{"role": {roleOwner}}, rolePreviewTestSession(t, a, email, roleAdmin))
	previewCookie := rolePreviewResponseCookie(t, start)
	logout := rolePreviewTestRequest(t, a, http.MethodPost, "/demo/auth/logout", nil, previewCookie)
	if logout.Code != http.StatusSeeOther {
		t.Fatalf("logout status = %d", logout.Code)
	}
	ends := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: store.AuditActionRolePreviewEnd, Limit: 10})
	if len(ends) != 1 || ends[0].Details["reason"] != "logout" {
		t.Fatalf("logout audit = %+v", ends)
	}
}

func TestNavigationEntriesByRoleFamilyHAUSV606(t *testing.T) {
	tests := []struct {
		name string
		role string
		want []string
	}{
		{name: "Hausverwaltung", role: roleAdmin, want: []string{"/demo/app/verwaltung", "/demo/app/verwaltung/posteingang", "/demo/app/verwaltung/textbausteine", "/demo/app/verwaltung/rechte", "/demo/app/verwaltung/einstellungen", "/demo/app", "/demo/app/energie", "/demo/app/announcements", "/demo/app/events", "/demo/app/kontakte", "/demo/app/dokumente", "/demo/app/anliegen/board", "/demo/app/abstimmungen", "/demo/app/parking", "/demo/app/uebergaben", "/demo/app/settings/users", "/demo/app/audit", "/demo/app/settings", "/demo/app/hilfe"}},
		{name: "Eigentümer", role: roleOwner, want: []string{"/demo/app", "/demo/app/energie", "/demo/app/announcements", "/demo/app/events", "/demo/app/kontakte", "/demo/app/dokumente", "/demo/app/anliegen", "/demo/app/abstimmungen", "/demo/app/audit", "/demo/app/settings", "/demo/app/hilfe"}},
		{name: "Bewohner", role: roleResident, want: []string{"/demo/app", "/demo/app/announcements", "/demo/app/events", "/demo/app/kontakte", "/demo/app/dokumente", "/demo/app/anliegen", "/demo/app/abstimmungen", "/demo/app/audit", "/demo/app/settings", "/demo/app/hilfe"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			email := strings.ToLower(test.name) + "@example.com"
			a := newTestPortalApp(t, userProfile{Email: email, Role: test.role, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
			if test.role == roleAdmin {
				tenant := a.tenants["demo"]
				tenant.Organisation = "test-verwaltung"
				a.tenants["demo"] = tenant
				a.organisations = map[string]config.OrganisationConfig{"test-verwaltung": {Key: "test-verwaltung", Name: "Test-Verwaltung"}}
			}
			page := rolePreviewTestRequest(t, a, http.MethodGet, "/demo/app", nil, rolePreviewTestSession(t, a, email, test.role))
			if page.Code != http.StatusOK {
				t.Fatalf("portal status = %d", page.Code)
			}
			if got := rolePreviewSidebarNavigation(page.Body.String()); !slices.Equal(got, test.want) {
				t.Fatalf("navigation = %v, want %v", got, test.want)
			}
		})
	}
}

var rolePreviewHrefPattern = regexp.MustCompile(`href="([^"]+)"`)

func rolePreviewSidebarNavigation(body string) []string {
	start := strings.Index(body, `<nav class="nav"`)
	if start < 0 {
		return nil
	}
	end := strings.Index(body[start:], `</nav>`)
	if end < 0 {
		return nil
	}
	matches := rolePreviewHrefPattern.FindAllStringSubmatch(body[start:start+end], -1)
	result := make([]string, 0, len(matches))
	for _, match := range matches {
		// HAUSV-621 moved the house header card, whose map thumbnail links to
		// OpenStreetMap, inside the navigation. Navigation entries are in-app routes.
		if !strings.HasPrefix(match[1], "/") {
			continue
		}
		result = append(result, match[1])
	}
	return result
}

func rolePreviewTestSession(t *testing.T, a *app, email string, role string) *http.Cookie {
	t.Helper()
	token, _, err := a.sessions.PutSession(email, "demo", authMethodEmail, role, time.Now().Add(time.Hour).Truncate(time.Second))
	if err != nil {
		t.Fatalf("put session: %v", err)
	}
	return &http.Cookie{Name: "weg_session", Value: token}
}

func rolePreviewTestRequest(t *testing.T, a *app, method string, path string, values url.Values, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var body *strings.Reader
	if values == nil {
		body = strings.NewReader("")
	} else {
		body = strings.NewReader(values.Encode())
	}
	req := httptest.NewRequest(method, "http://hausv.org"+path, body)
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", "http://hausv.org/demo")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rr := httptest.NewRecorder()
	a.handler().ServeHTTP(rr, req)
	return rr
}

func rolePreviewResponseCookie(t *testing.T, response *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == "weg_session" && cookie.Value != "" {
			return cookie
		}
	}
	t.Fatalf("response has no live weg_session cookie: %v", response.Header()["Set-Cookie"])
	return nil
}
