package server

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

// The Google Ads tag is demo-host configuration behind a consent gate
// (HAUSV-742): with GOOGLE_ADS_TAG_ID set, the public pages carry only the
// self-hosted consent script that injects Google's tag after "marketing" was
// granted; the served HTML never references Google's host, the CSP opens
// Google's hosts in exactly that case, and the lead conversion is configured
// only on the "sent" view of /start, never on a plain visit.
func TestGoogleAdsTagIsConsentGatedAndScopedToPublicPages(t *testing.T) {
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
	mustContain := func(t *testing.T, body, label string, wants ...string) {
		t.Helper()
		for _, want := range wants {
			if !strings.Contains(body, want) {
				t.Fatalf("%s missing %q", label, want)
			}
		}
	}
	mustNotContain := func(t *testing.T, body, label string, wants ...string) {
		t.Helper()
		for _, want := range wants {
			if strings.Contains(body, want) {
				t.Fatalf("%s must not contain %q", label, want)
			}
		}
	}

	t.Run("off by default", func(t *testing.T) {
		setEnv(t, "", "")
		a, err := newApp()
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range []string{"/", "/impressum", "/datenschutz", "/start"} {
			rr := get(t, a, path)
			if rr.Code != http.StatusOK {
				t.Fatalf("%s: %d", path, rr.Code)
			}
			mustNotContain(t, rr.Body.String(), path, "googletagmanager", "consent.js", "data-consent-open", "hv-consent", "Werbe-Cookies")
			if csp := rr.Header().Get("Content-Security-Policy"); strings.Contains(csp, "google") {
				t.Fatalf("CSP must not open Google hosts without a tag: %q", csp)
			}
		}
		privacy := get(t, a, "/datenschutz").Body.String()
		mustContain(t, privacy, "privacy without a tag", "Es gibt keine Werbung, keine Analyse-Skripte")
	})

	t.Run("gated tag on the public pages, conversion only on the sent view", func(t *testing.T) {
		setEnv(t, "AW-18425188397", "AW-18425188397/sEB6CJKVjvEcEK2g6NFE")
		a, err := newApp()
		if err != nil {
			t.Fatal(err)
		}
		// Every public page: the gate script with the tag id, a withdrawal
		// control, the bar styles — and no reference to Google's host in the
		// markup, because the tag only exists after consent.
		for _, path := range []string{"/", "/impressum", "/datenschutz", "/start"} {
			rr := get(t, a, path)
			if rr.Code != http.StatusOK {
				t.Fatalf("%s: %d", path, rr.Code)
			}
			body := rr.Body.String()
			mustContain(t, body, path,
				// asset URLs are tenant-prefixed by the response rewriter (/demo/assets/...)
				`assets/consent.js?v=`,
				`data-tag-id="AW-18425188397"`,
				`data-consent-open`,
				`.hv-consent-btn {`,
			)
			mustNotContain(t, body, path, "googletagmanager", "google-ads.js", "gtag/js")
			csp := rr.Header().Get("Content-Security-Policy")
			mustContain(t, csp, path+" CSP",
				"script-src 'self' https://www.googletagmanager.com",
				"connect-src 'self' https://www.google.com",
				"frame-ancestors 'none'",
			)
			for _, directive := range strings.Split(csp, ";") {
				directive = strings.TrimSpace(directive)
				if strings.HasPrefix(directive, "script-src") && strings.Contains(directive, "'unsafe-inline'") {
					t.Fatalf("script-src must stay free of unsafe-inline: %q", csp)
				}
			}
		}

		landing := get(t, a, "/").Body.String()
		mustNotContain(t, landing, "landing", "data-lead-conversion")

		start := get(t, a, "/start").Body.String()
		mustNotContain(t, start, "plain /start", "data-lead-conversion")
		sent := get(t, a, "/start?sent=1").Body.String()
		mustContain(t, sent, "/start?sent=1", `data-lead-conversion="AW-18425188397/sEB6CJKVjvEcEK2g6NFE"`)

		privacy := get(t, a, "/datenschutz").Body.String()
		mustContain(t, privacy, "privacy with a tag",
			"Werbe-Cookies (Google Ads)",
			"hausv_consent",
			"Global-Privacy-Control",
			"§ 165 Abs 3 TKG 2021",
		)
		mustNotContain(t, privacy, "privacy with a tag", "Es gibt keine Werbung, keine Analyse-Skripte")

		asset := get(t, a, "/assets/consent.js")
		if asset.Code != http.StatusOK {
			t.Fatalf("self-hosted consent script not served: %d", asset.Code)
		}
		script := asset.Body.String()
		mustContain(t, script, "consent.js",
			`gtag("consent", "default", { ad_storage: "denied"`,
			`cookie_domain: location.hostname`,
			`linker: { accept_incoming: true }`,
			`navigator.globalPrivacyControl === true`,
		)
		if strings.Count(script, "googletagmanager.com") != 1 {
			t.Fatal("consent.js must reference Google's host exactly once, inside the gated loader")
		}
		if rr := get(t, a, "/assets/google-ads.js"); rr.Code == http.StatusOK {
			t.Fatal("the ungated bootstrap must be gone")
		}
	})
}
