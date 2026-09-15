package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The vendored inspr-modules consent-gate is pinned byte for byte
// (HAUSV-753): these digests are the files of inspr-modules v0.14.0
// (packages/consent-gate). Re-copy and re-pin when the doctrine moves.
const (
	consentGateJSSHA256  = "7447e4e1ef8821e84a0db2d14c7506a873ebb3f960cc9da1549385ec6104d2f3"
	consentGateCSSSHA256 = "9610aac261f8df04978562f44bcbbc7fdf93305774daaf3788f616498dc9c262"
)

func TestVendoredConsentGateIsPinned(t *testing.T) {
	for name, want := range map[string]string{"consent-gate.js": consentGateJSSHA256, "consent-gate.css": consentGateCSSSHA256} {
		raw, err := os.ReadFile(filepath.Join("..", "web", "assets", name))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(raw)
		if got := hex.EncodeToString(sum[:]); got != want {
			t.Fatalf("%s drifted from the pinned inspr-modules bytes: %s != %s (re-copy from doctrine/packages/consent-gate and re-pin)", name, got, want)
		}
		if upstream, err := os.ReadFile(filepath.Join("..", "..", "doctrine", "packages", "consent-gate", name)); err == nil && string(upstream) != string(raw) {
			t.Fatalf("%s differs from doctrine/packages/consent-gate/%s", name, name)
		}
	}
}

// The Google Ads tag is demo-host configuration behind the consent gate
// (HAUSV-742, HAUSV-753): with GOOGLE_ADS_TAG_ID set, the public pages carry
// the manifest and the self-hosted gate script that injects Google's tag after
// "marketing" was granted; the served HTML never references Google's host, the
// CSP opens Google's hosts in exactly that case, and the conversion fires only
// on the "sent" view of /start, never on a plain visit.
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
	manifestOf := func(t *testing.T, body string) map[string]any {
		t.Helper()
		start := strings.Index(body, `<script type="application/json" id="consent-manifest">`)
		if start < 0 {
			t.Fatal("manifest script missing")
		}
		start += len(`<script type="application/json" id="consent-manifest">`)
		end := strings.Index(body[start:], "</script>")
		var m map[string]any
		if err := json.Unmarshal([]byte(body[start:start+end]), &m); err != nil {
			t.Fatalf("manifest is not JSON: %v", err)
		}
		return m
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
			mustNotContain(t, rr.Body.String(), path, "googletagmanager", "consent-gate", "consent-manifest", "data-consent-open", "Werbe-Cookies")
			if csp := rr.Header().Get("Content-Security-Policy"); strings.Contains(csp, "google") {
				t.Fatalf("CSP must not open Google hosts without a tag: %q", csp)
			}
		}
		mustContain(t, get(t, a, "/datenschutz").Body.String(), "privacy without a tag", "Es gibt keine Werbung, keine Analyse-Skripte")
	})

	t.Run("gated tag on the public pages, conversion only on the sent view", func(t *testing.T) {
		setEnv(t, "AW-18425188397", "AW-18425188397/sEB6CJKVjvEcEK2g6NFE")
		a, err := newApp()
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range []string{"/", "/impressum", "/datenschutz", "/start"} {
			rr := get(t, a, path)
			if rr.Code != http.StatusOK {
				t.Fatalf("%s: %d", path, rr.Code)
			}
			body := rr.Body.String()
			mustContain(t, body, path,
				// asset URLs are tenant-prefixed by the response rewriter (/demo/assets/...)
				`assets/consent-gate.js?v=`,
				`assets/consent-gate.css?v=`,
				`data-consent-manifest="#consent-manifest"`,
				`data-consent-open`,
				`--ic-surface: var(--panel)`,
			)
			mustNotContain(t, body, path, "googletagmanager", "google-ads.js", "consent.js", "gtag/js", "data-tag-id")
			m := manifestOf(t, body)
			if m["scope"] != "hausv.example" || m["cookieName"] != "hausv_consent" || m["language"] != "de" || m["privacyUrl"] != "/datenschutz" {
				t.Fatalf("manifest identity wrong: %v", m)
			}
			svc := m["services"].([]any)[0].(map[string]any)
			dest := svc["destination"].(map[string]any)
			if dest["tagId"] != "AW-18425188397" || dest["conversion"].(map[string]any)["sendTo"] != "AW-18425188397/sEB6CJKVjvEcEK2g6NFE" {
				t.Fatalf("destination wrong: %v", dest)
			}
			if keys := dest["consent"].([]any); len(keys) != 2 || keys[0] != "ad_storage" || keys[1] != "ad_user_data" {
				t.Fatalf("only measurement keys may be declared: %v", keys)
			}
			csp := rr.Header().Get("Content-Security-Policy")
			mustContain(t, csp, path+" CSP", "script-src 'self' https://www.googletagmanager.com", "connect-src 'self' https://www.google.com", "frame-ancestors 'none'")
			for _, directive := range strings.Split(csp, ";") {
				directive = strings.TrimSpace(directive)
				if strings.HasPrefix(directive, "script-src") && strings.Contains(directive, "'unsafe-inline'") {
					t.Fatalf("script-src must stay free of unsafe-inline: %q", csp)
				}
			}
		}

		mustContain(t, get(t, a, "/").Body.String(), "landing", `data-consent-fire=""`)
		mustContain(t, get(t, a, "/start").Body.String(), "plain /start", `data-consent-fire=""`)
		mustContain(t, get(t, a, "/start?sent=1").Body.String(), "/start?sent=1", `data-consent-fire="google-ads"`)

		privacy := get(t, a, "/datenschutz").Body.String()
		mustContain(t, privacy, "privacy with a tag", "Werbe-Cookies (Google Ads)", "hausv_consent", "Global-Privacy-Control", "§ 165 Abs 3 TKG 2021")
		mustNotContain(t, privacy, "privacy with a tag", "Es gibt keine Werbung, keine Analyse-Skripte")

		for _, name := range []string{"consent-gate.js", "consent-gate.css"} {
			if rr := get(t, a, "/assets/"+name); rr.Code != http.StatusOK {
				t.Fatalf("vendored %s not served: %d", name, rr.Code)
			}
		}
		script := get(t, a, "/assets/consent-gate.js").Body.String()
		if strings.Count(script, "googletagmanager.com/gtag/js") != 1 {
			t.Fatal("consent-gate.js must reference the tag loader exactly once, inside the gated loader")
		}
		for _, gone := range []string{"/assets/google-ads.js", "/assets/consent.js"} {
			if rr := get(t, a, gone); rr.Code == http.StatusOK {
				t.Fatalf("%s must be gone", gone)
			}
		}
	})

	t.Run("manifest omits the conversion without a configured lead conversion", func(t *testing.T) {
		setEnv(t, "AW-18425188397", "")
		a, err := newApp()
		if err != nil {
			t.Fatal(err)
		}
		m := manifestOf(t, get(t, a, "/start?sent=1").Body.String())
		dest := m["services"].([]any)[0].(map[string]any)["destination"].(map[string]any)
		if _, has := dest["conversion"]; has {
			t.Fatal("conversion must not be declared without GOOGLE_ADS_LEAD_CONVERSION")
		}
	})
}
