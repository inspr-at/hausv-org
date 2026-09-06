package server

import (
	"github.com/inspr-at/hausv-org/internal/mail"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

// The demo bundle (HAUSV-609) runs on a public host where LOCAL_DEV_LOGIN is
// off by design. DEMO_LOGIN_ENABLED plus a shared access code renders the
// magic link inline; a wrong code renders nothing and creates no token.
func TestDemoLoginRendersDevLinkOnlyWithAccessCode(t *testing.T) {
	const email = "manager@example.com"
	a := newTestPortalApp(t, userProfile{
		Email: email, FirstName: "Vera", LastName: "Verwaltung", Role: roleManager,
		Tenants: []string{"demo"}, TenantMemberships: map[string]tenantMembership{"demo": {Role: roleManager}},
		AuthMethods: defaultAuthMethods(),
	})
	a.localDevLogin = false
	a.demoLogin = true
	a.demoLoginCode = "musterstadt-2026"
	if !a.emailLoginAvailable() {
		t.Fatalf("demo login must make e-mail login available without SMTP")
	}

	post := func(code string) *httptest.ResponseRecorder {
		values := url.Values{"email": {email}}
		if code != "" {
			values.Set("access_code", code)
		}
		req := httptest.NewRequest(http.MethodPost, "http://hausv.org/demo/auth/request", strings.NewReader(values.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", "http://hausv.org")
		rr := httptest.NewRecorder()
		a.handler().ServeHTTP(rr, req)
		return rr
	}

	right := post("musterstadt-2026")
	if right.Code != http.StatusOK || !strings.Contains(right.Body.String(), `class="dev-link"`) {
		t.Fatalf("right code: status=%d, dev link missing: %s", right.Code, right.Body.String())
	}
	wrong := post("falsch")
	if wrong.Code != http.StatusOK || strings.Contains(wrong.Body.String(), `class="dev-link"`) || !strings.Contains(wrong.Body.String(), "Der Zugangscode war falsch") {
		t.Fatalf("wrong code: status=%d body=%s", wrong.Code, wrong.Body.String())
	}
	missing := post("")
	if strings.Contains(missing.Body.String(), `class="dev-link"`) {
		t.Fatalf("missing code must not render the dev link")
	}

	home := httptest.NewRecorder()
	a.handler().ServeHTTP(home, httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/", nil))
	if !strings.Contains(home.Body.String(), `name="access_code"`) {
		t.Fatalf("login form must ask for the access code when demo login is enabled")
	}
}

// HAUSV-643: a file outbox makes the mailer "configured" for statement
// delivery, but nobody reads those files — the demo login must still show
// the direct link instead of stranding the person.
func TestDemoLoginRendersDevLinkWithOutboxMailer(t *testing.T) {
	const email = "manager@example.com"
	a := newTestPortalApp(t, userProfile{
		Email: email, FirstName: "Vera", LastName: "Verwaltung", Role: roleManager,
		Tenants: []string{"demo"}, TenantMemberships: map[string]tenantMembership{"demo": {Role: roleManager}},
		AuthMethods: defaultAuthMethods(),
	})
	a.localDevLogin = false
	a.demoLogin = true
	a.demoLoginCode = "musterstadt-2026"
	outbox := t.TempDir()
	a.mailer = mail.NewSMTP("", "587", "", "", "hausv <noreply@example.invalid>").WithOutbox(outbox)
	if !a.mailer.Configured() || a.mailer.Delivers() {
		t.Fatalf("outbox mailer must be configured but not delivering")
	}
	values := url.Values{"email": {email}, "access_code": {"musterstadt-2026"}}
	req := httptest.NewRequest(http.MethodPost, "http://hausv.org/demo/auth/request", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://hausv.org")
	rr := httptest.NewRecorder()
	a.handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `class="dev-link"`) {
		t.Fatalf("demo login with outbox mailer: status=%d, dev link missing", rr.Code)
	}
	entries, err := os.ReadDir(outbox)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("login link must not be written to the outbox, found %d files", len(entries))
	}
}
