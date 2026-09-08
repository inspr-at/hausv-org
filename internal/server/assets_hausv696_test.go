package server

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/version"
	"github.com/inspr-at/hausv-org/internal/web"
)

func TestAssetsCacheAndConditionalRequestsHAUSV696(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	handler := a.handler()
	for _, asset := range []string{"source-serif-4-semibold.woff2", "app.js", "portal-shell.css", "icons/lucide/car.svg", "feature-integrations.webp", "hausv-landing-hero.png"} {
		contents, err := web.Assets.ReadFile("assets/" + asset)
		if err != nil {
			t.Fatal(err)
		}
		etag := fmt.Sprintf(`"%x"`, sha256.Sum256(contents))
		for _, prefix := range []string{"", "/demo"} {
			for _, query := range []string{"", "?v=", "?v=old-build", "?v=" + version.AssetVersion()} {
				path := prefix + "/assets/" + asset + query
				t.Run(path, func(t *testing.T) {
					wantCache := "public, max-age=86400"
					if query == "?v="+version.AssetVersion() {
						wantCache = "public, max-age=31536000, immutable"
					}
					for _, method := range []string{http.MethodGet, http.MethodHead} {
						rr := httptest.NewRecorder()
						handler.ServeHTTP(rr, httptest.NewRequest(method, "http://hausv.org"+path, nil))
						if rr.Code != http.StatusOK || rr.Header().Get("Cache-Control") != wantCache || rr.Header().Get("ETag") != etag {
							t.Fatalf("%s: status=%d cache=%q etag=%q", method, rr.Code, rr.Header().Get("Cache-Control"), rr.Header().Get("ETag"))
						}
						if rr.Header().Get("Content-Type") == "" || (asset == "source-serif-4-semibold.woff2" && rr.Header().Get("Content-Type") != "font/woff2") {
							t.Fatalf("unexpected Content-Type: %q", rr.Header().Get("Content-Type"))
						}
						if method == http.MethodGet && rr.Body.String() != string(contents) {
							t.Fatal("asset bytes changed")
						}
						if method == http.MethodHead && rr.Body.Len() != 0 {
							t.Fatal("HEAD must not send a body")
						}
						request := httptest.NewRequest(method, "http://hausv.org"+path, nil)
						request.Header.Set("If-None-Match", etag)
						cached := httptest.NewRecorder()
						handler.ServeHTTP(cached, request)
						if cached.Code != http.StatusNotModified || cached.Body.Len() != 0 || cached.Header().Get("ETag") != etag || cached.Header().Get("Cache-Control") != wantCache {
							t.Fatalf("conditional %s: status=%d headers=%v", method, cached.Code, cached.Header())
						}
					}
				})
			}
		}
	}
	for _, path := range []string{"/assets/missing.woff2?v=" + version.AssetVersion(), "/assets/"} {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "http://hausv.org"+path, nil))
		if rr.Code != http.StatusNotFound || strings.Contains(rr.Header().Get("Cache-Control"), "immutable") || rr.Header().Get("ETag") != "" {
			t.Fatalf("missing asset must not be cached as immutable: status=%d headers=%v", rr.Code, rr.Header())
		}
	}
	request := httptest.NewRequest(http.MethodGet, "http://hausv.org/assets/source-serif-4-semibold.woff2", nil)
	request.Header.Set("Range", "bytes=0-15")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, request)
	if rr.Code != http.StatusPartialContent || rr.Body.Len() != 16 || !strings.HasPrefix(rr.Header().Get("Content-Range"), "bytes 0-15/") {
		t.Fatalf("range request: status=%d size=%d", rr.Code, rr.Body.Len())
	}
}

func TestTenantShellFontPreloadAndInboxCSPHAUSV696(t *testing.T) {
	a, _, _ := newInboxTestApp(t, roleManager)
	for _, path := range []string{"/demo/app", "/demo/app/verwaltung", "/demo/app/verwaltung/posteingang"} {
		page := authedRequest(t, a, "vera@example.com", path)
		if page.Code != http.StatusOK {
			t.Fatalf("%s: status=%d", path, page.Code)
		}
		body := page.Body.String()
		head, _, _ := strings.Cut(body, "</head>")
		fontURL := "/demo/assets/source-serif-4-semibold.woff2?v=" + version.AssetVersion()
		preload := `<link rel="preload" href="` + fontURL + `" as="font" type="font/woff2" crossorigin="anonymous">`
		if strings.Count(head, preload) != 1 || !strings.Contains(head, `src:url("`+fontURL+`")`) || strings.Count(body, "@font-face") != 1 {
			t.Errorf("%s: preload and unique font-face must use the same tenant URL", path)
		}
		csp := page.Header().Get("Content-Security-Policy")
		if !strings.Contains(csp, "font-src 'self'") {
			t.Errorf("%s: same-origin font must be allowed: %s", path, csp)
		}
		if strings.HasSuffix(path, "/posteingang") && !strings.Contains(csp, "script-src 'self' 'nonce-") {
			t.Error("inbox must retain its nonce CSP")
		}
	}
}
