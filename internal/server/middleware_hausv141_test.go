package server

import (
	"bytes"
	"encoding/json"
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
	mux := http.NewServeMux()
	mux.HandleFunc("GET /app", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	h := a.recoverAndLog(mux)
	req := httptest.NewRequest(http.MethodGet, "https://jhw22.hausv.org/app", nil)
	h.ServeHTTP(httptest.NewRecorder(), req)
	got := logs.String()
	if !strings.Contains(got, `"tenant":"jhw22"`) {
		t.Fatalf("request log has no tenant: %s", got)
	}
	if !strings.Contains(got, `"route":"GET /app"`) {
		t.Fatalf("request log has no route pattern: %s", got)
	}
}

func TestRequestLogNeverStoresPathTokensOrQueryData(t *testing.T) {
	var logs bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	defer slog.SetDefault(old)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /calendar/{token}", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})
	mux.HandleFunc("GET /handover/{token}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /auth/verify", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	h := (&app{}).recoverAndLog(mux)

	secrets := []string{
		"calendar-secret-token",
		"handover-secret-token",
		"magic-secret-token",
		"person@example.com",
	}
	for _, target := range []string{
		"https://jhw22.hausv.org/calendar/calendar-secret-token.ics",
		"https://jhw22.hausv.org/handover/handover-secret-token",
		"https://jhw22.hausv.org/auth/verify?token=magic-secret-token&email=person@example.com",
	} {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, target, nil))
	}

	for _, secret := range secrets {
		if strings.Contains(logs.String(), secret) {
			t.Fatalf("request log leaked %q: %s", secret, logs.String())
		}
	}

	lines := strings.Split(strings.TrimSpace(logs.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("log lines = %d, want 3: %s", len(lines), logs.String())
	}
	wantRoutes := []string{
		"GET /calendar/{token}",
		"GET /handover/{token}",
		"GET /auth/verify",
	}
	for i, line := range lines {
		var event map[string]any
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("line %d is not JSON: %v", i, err)
		}
		if event["route"] != wantRoutes[i] {
			t.Fatalf("line %d route = %q, want %q", i, event["route"], wantRoutes[i])
		}
		if _, found := event["path"]; found {
			t.Fatalf("line %d retained a concrete path: %s", i, line)
		}
		if _, found := event["host"]; found {
			t.Fatalf("line %d retained a redundant host: %s", i, line)
		}
	}
}
