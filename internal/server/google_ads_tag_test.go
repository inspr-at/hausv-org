package server

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

// The Google Ads tag is demo-host configuration: present on the public pages
// only when GOOGLE_ADS_TAG_ID is set, with the CSP opened for Google's hosts
// in exactly that case, and the lead conversion only on the page a visitor
// reaches after submitting the landing form (/start).
func TestGoogleAdsTagIsOptionalAndScopedToPublicPages(t *testing.T) {
	setEnv := func(t *testing.T, tag, conversion string) {
		t.Helper()
		t.Setenv("DB_PATH", filepath.Join(t.TempDir(), "boot.db"))
		t.Setenv("PARKING_DATA_PATH", filepath.Join(t.TempDir(), "parking.json"))
		t.Setenv("BASE_URL", "https://hausv.example")
		t.Setenv("ROOT_DOMAIN", "hausv.example") // the demo deploy sets this to the public host
		t.Setenv("TRUSTED_PROXY_CIDRS", "127.0.0.1/32")
		t.Setenv("SESSION_KEY", strings.Repeat("ab", 32))
		t.Setenv("SMTP_HOST", "")
		t.Setenv("OIDC_ISSUER", "")
		t.Setenv("DEMO_LOGIN_ENABLED", "true")
		t.Setenv("DEMO_LOGIN_ACCESS_CODE", "musterstadt-2026")
		t.Setenv("GOOGLE_ADS_TAG_ID", tag)
		t.Setenv("GOOGLE_ADS_LEAD_CONVERSION", conversion)
	}
	get := func(t *testing.T, a *app, path string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "http://hausv.example"+path, nil)
		rr := httptest.NewRecorder()
		a.handler().ServeHTTP(rr, req)
		return rr
	}

	t.Run("off by default", func(t *testing.T) {
		setEnv(t, "", "")
		a, err := newApp()
		if err != nil {
			t.Fatal(err)
		}
		rr := get(t, a, "/")
		if rr.Code != http.StatusOK {
			t.Fatalf("landing: %d", rr.Code)
		}
		if strings.Contains(rr.Body.String(), "googletagmanager") {
			t.Fatal("landing must not load Google's tag without GOOGLE_ADS_TAG_ID")
		}
		if csp := rr.Header().Get("Content-Security-Policy"); strings.Contains(csp, "google") {
			t.Fatalf("CSP must not open Google hosts without a tag: %q", csp)
		}
	})

	t.Run("tag on the landing page, conversion only after the lead form", func(t *testing.T) {
		setEnv(t, "AW-18425188397", "AW-18425188397/sEB6CJKVjvEcEK2g6NFE")
		a, err := newApp()
		if err != nil {
			t.Fatal(err)
		}
		landing := get(t, a, "/")
		body := landing.Body.String()
		for _, want := range []string{
			`<script async src="https://www.googletagmanager.com/gtag/js?id=AW-18425188397"></script>`,
			// asset URLs are tenant-prefixed by the response rewriter (/demo/assets/...)
			`assets/google-ads.js?v=`,
			`data-tag-id="AW-18425188397"`,
		} {
			if !strings.Contains(body, want) {
				t.Fatalf("landing missing %q", want)
			}
		}
		if strings.Contains(body, "data-lead-conversion") {
			t.Fatal("landing must not fire the lead conversion")
		}
		csp := landing.Header().Get("Content-Security-Policy")
		for _, want := range []string{
			"script-src 'self' https://www.googletagmanager.com",
			"connect-src 'self' https://www.google.com",
			"frame-ancestors 'none'",
		} {
			if !strings.Contains(csp, want) {
				t.Fatalf("CSP missing %q: %q", want, csp)
			}
		}
		for _, directive := range strings.Split(csp, ";") {
			directive = strings.TrimSpace(directive)
			if strings.HasPrefix(directive, "script-src") && strings.Contains(directive, "'unsafe-inline'") {
				t.Fatalf("script-src must stay free of unsafe-inline: %q", csp)
			}
		}

		start := get(t, a, "/start")
		if start.Code != http.StatusOK {
			t.Fatalf("/start: %d", start.Code)
		}
		if !strings.Contains(start.Body.String(), `data-lead-conversion="AW-18425188397/sEB6CJKVjvEcEK2g6NFE"`) {
			t.Fatal("/start must carry the lead conversion")
		}

		asset := get(t, a, "/assets/google-ads.js")
		if asset.Code != http.StatusOK || !strings.Contains(asset.Body.String(), "gtag(\"config\", script.dataset.tagId)") {
			t.Fatalf("self-hosted bootstrap not served: %d", asset.Code)
		}
	})
}
