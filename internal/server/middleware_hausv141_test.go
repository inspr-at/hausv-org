package server

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/markus-barta/hausv-org/internal/config"
)

// HAUSV-141: a panicking handler must yield a 500, not crash the request.
func TestRecoverMiddlewareTurnsPanicInto500(t *testing.T) {
	h := (&app{}).recoverAndLog(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		var m map[string]int
		m["boom"] = 1 // nil-map assignment: real panic
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/x", nil))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rr.Code)
	}
}

// The status wrapper must not alter a normal response.
func TestStatusWriterIsTransparent(t *testing.T) {
	h := (&app{}).recoverAndLog(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Test", "1")
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("hello"))
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/x", nil))
	if rr.Code != http.StatusTeapot || rr.Body.String() != "hello" || rr.Header().Get("X-Test") != "1" {
		t.Fatalf("wrapper altered response: code=%d body=%q hdr=%q", rr.Code, rr.Body.String(), rr.Header().Get("X-Test"))
	}
}

func TestRequestLogIncludesTenant(t *testing.T) {
	var logs bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	defer slog.SetDefault(old)

	a := &app{
		defaultTenant: "jhw22",
		tenants: map[string]config.TenantConfig{
			"jhw22": {Slug: "jhw22", Host: "jhw22.hausv.org"},
		},
	}
	h := a.recoverAndLog(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "https://jhw22.hausv.org/app", nil)
	h.ServeHTTP(httptest.NewRecorder(), req)
	if got := logs.String(); !strings.Contains(got, `"tenant":"jhw22"`) {
		t.Fatalf("request log has no tenant: %s", got)
	}
}
