package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTemplExampleRouteIsOffByDefault(t *testing.T) {
	a := &app{}
	response := httptest.NewRecorder()
	a.routes().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "http://hausv.org/_templ/example", nil))

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
	if strings.Contains(response.Body.String(), "data-templ-example") {
		t.Fatal("templ example rendered while its switch was off")
	}
}

func TestTemplExampleRouteRendersWhenEnabled(t *testing.T) {
	a := &app{templExampleEnabled: true}
	response := httptest.NewRecorder()
	a.routes().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "http://hausv.org/_templ/example", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	body := response.Body.String()
	for _, want := range []string{
		"data-templ-example",
		`/assets/htmx/2.0.10/htmx.min.js?v=`,
		"templ ist bereit",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("example response does not contain %q", want)
		}
	}
	if got := response.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q", got)
	}
}

func TestVendoredHTMXAssetIsServed(t *testing.T) {
	a := &app{}
	response := httptest.NewRecorder()
	a.routes().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "http://hausv.org/assets/htmx/2.0.10/htmx.min.js?v=test", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), "htmx") {
		t.Fatal("vendored htmx response does not contain the library marker")
	}
}
