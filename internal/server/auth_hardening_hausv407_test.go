package server

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/web"
)

func TestAuthRateLimiterAppliesChecksAtomicallyWithoutRetainingPII(t *testing.T) {
	now := time.Date(2026, 7, 30, 8, 0, 0, 0, time.UTC)
	limiter := newAuthRateLimiter(func() time.Time { return now })
	sourcePolicy := authRatePolicy{scope: "test-source", limit: 2, window: 10 * time.Minute}
	accountPolicy := authRatePolicy{scope: "test-account", limit: 2, window: 10 * time.Minute}
	checks := []authRateCheck{
		{policy: sourcePolicy, value: "203.0.113.4"},
		{policy: accountPolicy, value: "person@example.com"},
	}

	for i := 0; i < 2; i++ {
		if allowed, retry := limiter.Allow(checks...); !allowed || retry != 0 {
			t.Fatalf("attempt %d = allowed %v retry %s", i+1, allowed, retry)
		}
	}
	if allowed, retry := limiter.Allow(checks...); allowed || retry != 10*time.Minute {
		t.Fatalf("limited attempt = allowed %v retry %s", allowed, retry)
	}
	for key := range limiter.buckets {
		if strings.Contains(key, "person@example.com") || strings.Contains(key, "203.0.113.4") {
			t.Fatalf("limiter retained raw source/account value in %q", key)
		}
	}

	now = now.Add(10 * time.Minute)
	if allowed, retry := limiter.Allow(checks...); !allowed || retry != 0 {
		t.Fatalf("attempt after reset = allowed %v retry %s", allowed, retry)
	}
}

func TestTrustedProxyAllowlistHasSafeLocalAndPublicDefaults(t *testing.T) {
	local, err := parseTrustedProxyAllowlist("", false)
	if err != nil {
		t.Fatal(err)
	}
	if !local.contains(net.ParseIP("127.0.0.1")) || !local.contains(net.ParseIP("::1")) {
		t.Fatal("local default does not trust loopback")
	}
	if local.contains(net.ParseIP("172.18.0.4")) {
		t.Fatal("local default trusts an unrelated private peer")
	}
	if _, err := parseTrustedProxyAllowlist("", true); err == nil {
		t.Fatal("public app accepted an implicit proxy trust boundary")
	}
	if _, err := parseTrustedProxyAllowlist("0.0.0.0/0", true); err == nil {
		t.Fatal("public app accepted the complete IPv4 address space")
	}
}

func TestAuthRequestSourceOnlyTrustsSanitizedHeaderFromExplicitProxy(t *testing.T) {
	proxies, err := parseTrustedProxyAllowlist("172.18.0.4/32", true)
	if err != nil {
		t.Fatal(err)
	}
	a := &app{trustedProxies: proxies}

	proxied := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/", nil)
	proxied.RemoteAddr = "172.18.0.4:43122"
	proxied.Header.Set("X-Real-IP", "198.51.100.12")
	proxied.Header.Set("X-Forwarded-For", "192.0.2.99")
	if got := a.authRequestSource(proxied); got != "198.51.100.12" {
		t.Fatalf("proxied source = %q", got)
	}

	foreignPrivate := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/", nil)
	foreignPrivate.RemoteAddr = "172.18.0.5:43122"
	foreignPrivate.Header.Set("X-Real-IP", "198.51.100.12")
	foreignPrivate.Header.Set("X-Forwarded-For", "192.0.2.99")
	if got := a.authRequestSource(foreignPrivate); got != "172.18.0.5" {
		t.Fatalf("foreign private peer trusted forwarding header: %q", got)
	}

	direct := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/", nil)
	direct.RemoteAddr = "203.0.113.20:43122"
	direct.Header.Set("X-Real-IP", "198.51.100.12")
	direct.Header.Set("X-Forwarded-For", "192.0.2.99")
	if got := a.authRequestSource(direct); got != "203.0.113.20" {
		t.Fatalf("direct source trusted spoofable forwarding header: %q", got)
	}
}

func TestMagicLinkRequestDoesNotRevealWhetherAccountExists(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "owner@example.com",
		Role:        roleOwner,
		Tenants:     []string{"demo"},
		AuthMethods: defaultAuthMethods(),
	})
	mailer := &recordingMailer{}
	a.mailer = mailer

	known := requestMagicLink(t, a, "owner@example.com", "203.0.113.40:1234")
	unknown := requestMagicLink(t, a, "unknown@example.com", "203.0.113.41:1234")
	if known.Code != http.StatusSeeOther || unknown.Code != known.Code {
		t.Fatalf("known/unknown statuses = %d/%d", known.Code, unknown.Code)
	}
	if known.Header().Get("Location") != "/demo/?sent=1" ||
		unknown.Header().Get("Location") != known.Header().Get("Location") {
		t.Fatalf("known/unknown locations = %q/%q", known.Header().Get("Location"), unknown.Header().Get("Location"))
	}
	if known.Body.String() != unknown.Body.String() {
		t.Fatalf("known/unknown response bodies differ: %q / %q", known.Body.String(), unknown.Body.String())
	}
	magicLinks := waitForMagicLinks(t, mailer, 1)
	if magicLinks[0].To != "owner@example.com" {
		t.Fatalf("sent magic links = %+v", magicLinks)
	}
}

func TestMagicLinkRateLimitsAccountAndSourceWithGenericRetry(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "owner@example.com",
		Role:        roleOwner,
		Tenants:     []string{"demo"},
		AuthMethods: defaultAuthMethods(),
	})
	now := time.Date(2026, 7, 30, 8, 0, 0, 0, time.UTC)
	a.authLimiter = newAuthRateLimiter(func() time.Time { return now })
	a.mailer = &recordingMailer{}

	for i := 0; i < magicLinkAccountPolicy.limit; i++ {
		response := requestMagicLink(t, a, "unknown@example.com", "203.0.113.50:1234")
		if response.Code != http.StatusSeeOther {
			t.Fatalf("account attempt %d status = %d", i+1, response.Code)
		}
	}
	accountLimited := requestMagicLink(t, a, "unknown@example.com", "203.0.113.51:1234")
	assertGenericRateLimit(t, accountLimited, int(authRateWindow/time.Second))

	sourceApp := newTestPortalApp(t, userProfile{
		Email:       "owner@example.com",
		Role:        roleOwner,
		Tenants:     []string{"demo"},
		AuthMethods: defaultAuthMethods(),
	})
	sourceApp.authLimiter = newAuthRateLimiter(func() time.Time { return now })
	sourceApp.mailer = &recordingMailer{}
	for i := 0; i < magicLinkSourcePolicy.limit; i++ {
		response := requestMagicLink(
			t,
			sourceApp,
			"unknown"+strconv.Itoa(i)+"@example.com",
			"203.0.113.60:1234",
		)
		if response.Code != http.StatusSeeOther {
			t.Fatalf("source attempt %d status = %d", i+1, response.Code)
		}
	}
	sourceLimited := requestMagicLink(t, sourceApp, "last@example.com", "203.0.113.60:1234")
	assertGenericRateLimit(t, sourceLimited, int(authRateWindow/time.Second))
}

func TestMagicLinkAccountLimitDoesNotConsumeValidToken(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "owner@example.com",
		Role:        roleOwner,
		Tenants:     []string{"demo"},
		AuthMethods: defaultAuthMethods(),
	})
	now := time.Date(2026, 7, 30, 8, 0, 0, 0, time.UTC)
	a.authLimiter = newAuthRateLimiter(func() time.Time { return now })
	account := "demo|owner@example.com"
	for i := 0; i < loginCompletionAccountPolicy.limit; i++ {
		allowed, retry := a.authLimiter.Allow(authRateCheck{
			policy: loginCompletionAccountPolicy,
			value:  account,
		})
		if !allowed || retry != 0 {
			t.Fatalf("seed account bucket attempt %d = %v/%s", i+1, allowed, retry)
		}
	}
	a.tokens.Put("still-valid", "owner@example.com", "demo", 15*time.Minute)

	limited := httptest.NewRecorder()
	limitedRequest := httptest.NewRequest(
		http.MethodGet,
		"http://hausv.org/demo/auth/verify?token=still-valid",
		nil,
	)
	limitedRequest.RemoteAddr = "203.0.113.70:1234"
	a.handler().ServeHTTP(limited, limitedRequest)
	assertGenericRateLimit(t, limited, int(authRateWindow/time.Second))

	now = now.Add(authRateWindow)
	accepted := httptest.NewRecorder()
	acceptedRequest := httptest.NewRequest(
		http.MethodGet,
		"http://hausv.org/demo/auth/verify?token=still-valid",
		nil,
	)
	acceptedRequest.RemoteAddr = "203.0.113.70:1234"
	a.handler().ServeHTTP(accepted, acceptedRequest)
	if accepted.Code != http.StatusSeeOther || accepted.Header().Get("Location") != "/demo/app" {
		t.Fatalf("token after limit reset = %d location %q body %q", accepted.Code, accepted.Header().Get("Location"), accepted.Body.String())
	}
	if accepted.Header().Get("Set-Cookie") == "" {
		t.Fatal("accepted one-time token did not create a session")
	}
}

func TestSecurityHeadersPreventCachingAndLeaveHSTSToEdge(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "owner@example.com",
		Role:        roleOwner,
		Tenants:     []string{"demo"},
		AuthMethods: defaultAuthMethods(),
	})

	login := httptest.NewRecorder()
	a.handler().ServeHTTP(login, httptest.NewRequest(http.MethodGet, "https://hausv.org/demo/", nil))
	if login.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("login cache control = %q", login.Header().Get("Cache-Control"))
	}
	if values := login.Header().Values("Strict-Transport-Security"); len(values) != 0 {
		t.Fatalf("application must leave HSTS to Traefik, got %#v", values)
	}

	protected := httptest.NewRecorder()
	a.handler().ServeHTTP(protected, httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/app", nil))
	if protected.Code != http.StatusSeeOther || protected.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("protected response = %d cache %q", protected.Code, protected.Header().Get("Cache-Control"))
	}
	if protected.Header().Get("Strict-Transport-Security") != "" {
		t.Fatalf("plain HTTP emitted HSTS: %q", protected.Header().Get("Strict-Transport-Security"))
	}

	errorPage := httptest.NewRecorder()
	a.handler().ServeHTTP(errorPage, httptest.NewRequest(http.MethodPost, "http://hausv.org/demo/not-a-route", nil))
	if errorPage.Code < http.StatusBadRequest || errorPage.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("error response = %d cache %q", errorPage.Code, errorPage.Header().Get("Cache-Control"))
	}

	asset := httptest.NewRecorder()
	a.handler().ServeHTTP(asset, httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/assets/app.js", nil))
	if asset.Code != http.StatusOK || asset.Header().Get("Cache-Control") == "no-store" {
		t.Fatalf("asset response = %d cache %q", asset.Code, asset.Header().Get("Cache-Control"))
	}

	proxiedHTTPS := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/healthz", nil)
	proxiedHTTPS.RemoteAddr = "172.18.0.4:1234"
	proxiedHTTPS.Header.Set("X-Forwarded-Proto", "https")
	proxied := httptest.NewRecorder()
	a.handler().ServeHTTP(proxied, proxiedHTTPS)
	if values := proxied.Header().Values("Strict-Transport-Security"); len(values) != 0 {
		t.Fatalf("application emitted HSTS for a forwarded request: %#v", values)
	}
}

func TestProtectedAppContentAlwaysDisablesCaching(t *testing.T) {
	for _, disposition := range []string{
		"",
		`inline; filename="preview.pdf"`,
		`attachment; filename="export.csv"`,
	} {
		t.Run(disposition, func(t *testing.T) {
			a := &app{}
			handler := a.securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Cache-Control", "private, max-age=300")
				if disposition != "" {
					w.Header().Set("Content-Disposition", disposition)
				}
				w.WriteHeader(http.StatusOK)
			}))
			response := httptest.NewRecorder()
			handler.ServeHTTP(
				response,
				httptest.NewRequest(http.MethodGet, "http://hausv.org/app/documents/content", nil),
			)
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d", response.Code)
			}
			if got := response.Header().Get("Cache-Control"); got != "no-store" {
				t.Fatalf("Cache-Control = %q for Content-Disposition %q", got, disposition)
			}
			if got := response.Header().Get("Content-Disposition"); got != disposition {
				t.Fatalf("Content-Disposition = %q, want %q", got, disposition)
			}
		})
	}
}

func TestLogoutRevokesSessionAndAuthenticatedBackCacheIsRevalidated(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "owner@example.com",
		Role:        roleOwner,
		Tenants:     []string{"demo"},
		AuthMethods: defaultAuthMethods(),
	})
	token, _, err := a.sessions.Put("owner@example.com", "demo", authMethodEmail, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	logoutRequest := httptest.NewRequest(http.MethodPost, "http://hausv.org/demo/auth/logout", nil)
	logoutRequest.Header.Set("Origin", "http://hausv.org/demo")
	logoutRequest.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	logout := httptest.NewRecorder()
	a.handler().ServeHTTP(logout, logoutRequest)
	if logout.Code != http.StatusSeeOther || logout.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("logout = %d cache %q", logout.Code, logout.Header().Get("Cache-Control"))
	}

	backRequest := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/app", nil)
	backRequest.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	back := httptest.NewRecorder()
	a.handler().ServeHTTP(back, backRequest)
	if back.Code != http.StatusSeeOther || back.Header().Get("Location") != "/demo/" ||
		back.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("revoked session response = %d location %q cache %q", back.Code, back.Header().Get("Location"), back.Header().Get("Cache-Control"))
	}

	page := authedRequest(t, a, "owner@example.com", "/demo/app")
	if !strings.Contains(page.Body.String(), "data-authenticated-app") {
		t.Fatal("authenticated page is missing its back-cache marker")
	}
	script, err := web.Assets.ReadFile("assets/app.js")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(script), `event.persisted`) ||
		!strings.Contains(string(script), `window.location.reload()`) {
		t.Fatal("app script does not revalidate a restored authenticated page")
	}
}

func requestMagicLink(t *testing.T, a *app, email string, remoteAddr string) *httptest.ResponseRecorder {
	t.Helper()
	values := url.Values{"email": {email}}
	request := httptest.NewRequest(
		http.MethodPost,
		"http://hausv.org/demo/auth/request",
		strings.NewReader(values.Encode()),
	)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "http://hausv.org/demo")
	request.RemoteAddr = remoteAddr
	response := httptest.NewRecorder()
	a.handler().ServeHTTP(response, request)
	return response
}

func waitForMagicLinks(t *testing.T, mailer *recordingMailer, count int) []sentMagicLink {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		links := mailer.recordedMagicLinks()
		if len(links) >= count {
			return links
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d magic links; got %+v", count, mailer.recordedMagicLinks())
	return nil
}

func assertGenericRateLimit(t *testing.T, response *httptest.ResponseRecorder, retryAfter int) {
	t.Helper()
	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("rate limit status = %d body %q", response.Code, response.Body.String())
	}
	if response.Header().Get("Retry-After") != strconv.Itoa(retryAfter) {
		t.Fatalf("Retry-After = %q, want %d", response.Header().Get("Retry-After"), retryAfter)
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("rate limit cache control = %q", response.Header().Get("Cache-Control"))
	}
	// The guarantee this asserts is that the wording stays generic: a throttled
	// caller must not learn whether the account exists. POST /auth/request still
	// answers in plain text; GET /auth/verify is a page a person navigates to and
	// now renders the branded error page, which embeds this exact wording and
	// adds nothing account-specific.
	if !strings.Contains(response.Body.String(), genericAuthRateLimitMessage) {
		t.Fatalf("rate limit body = %q", response.Body.String())
	}
}
