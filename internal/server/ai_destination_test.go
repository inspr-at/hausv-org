package server

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/ai"
	"github.com/inspr-at/hausv-org/internal/store"
)

// Converted from the architecture probe TestAuditAIOverrideReceivesSharedCredential.
// The probe reproduced the leak with a synthetic key and an in-memory transport.
// The regression asserts that the inherited credential does not reach the override.
func TestAuditAIOverrideReceivesSharedCredential(t *testing.T) {
	const key = "audit-synthetic-nonsecret"
	previous := http.DefaultClient
	t.Cleanup(func() { http.DefaultClient = previous })
	var auth, host string
	var calls int
	http.DefaultClient = &http.Client{Transport: aiAuditTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		auth = r.Header.Get("Authorization")
		host = r.URL.Host
		return &http.Response{StatusCode: http.StatusBadRequest, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("audit rejection")), Request: r}, nil
	})}

	settings := store.OrgSettings{AIProvider: "local", AIBaseURL: "https://audit.example.invalid/v1", AIModel: "audit-model"}
	if err := validateAIBaseURL(settings.AIBaseURL); err != nil {
		t.Fatal(err)
	}
	getter := func(name string) string {
		if name == "AI_API_KEY" {
			return key
		}
		return ""
	}
	provider, err := ai.NewFromEnv(aiSettingsGetenv(getter, settings))
	if err != nil || provider == nil {
		t.Fatalf("provider = %v, err = %v", provider, err)
	}
	_, _ = provider.Suggest(context.Background(), ai.TriageInput{Subject: "Synthetic audit"})
	if calls != 1 || host != "audit.example.invalid" {
		t.Fatalf("override was not called: calls=%d host=%q", calls, host)
	}
	if auth != "" || strings.Contains(auth, key) {
		t.Fatalf("inherited credential reached the override")
	}
}

func TestHAUSV772SameOriginOverrideKeepsProcessCredential(t *testing.T) {
	const key = "audit-synthetic-nonsecret"
	previous := http.DefaultClient
	t.Cleanup(func() { http.DefaultClient = previous })
	var auth, host string
	http.DefaultClient = &http.Client{Transport: aiAuditTransport(func(r *http.Request) (*http.Response, error) {
		auth = r.Header.Get("Authorization")
		host = r.URL.Host
		return &http.Response{StatusCode: http.StatusBadRequest, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("audit rejection")), Request: r}, nil
	})}

	settings := store.OrgSettings{AIProvider: "cloud", AIBaseURL: "https://OpenRouter.ai:443/other", AIModel: "audit-model"}
	getter := func(name string) string {
		switch name {
		case "AI_API_KEY":
			return key
		case "AI_BASE_URL":
			return "https://openrouter.ai/api/v1"
		case "AI_MODEL":
			return "env-model"
		default:
			return ""
		}
	}
	provider, err := ai.NewFromEnv(aiSettingsGetenv(getter, settings))
	if err != nil || provider == nil {
		t.Fatal(err)
	}
	_, _ = provider.Suggest(context.Background(), ai.TriageInput{Subject: "Synthetic audit"})
	if host != "OpenRouter.ai:443" || auth != "Bearer "+key {
		t.Fatalf("same-origin override host=%q authorization set=%v", host, auth == "Bearer "+key)
	}
}

func TestHAUSV772SaveAcceptsLANHTTPAndRejectsPublicHTTP(t *testing.T) {
	a, _, repo := newInboxTestApp(t, roleAdmin)
	const key = "audit-synthetic-nonsecret"
	t.Setenv("AI_API_KEY", key)

	accepted := authedFormRequest(t, a, "vera@example.com", "/demo/app/verwaltung/einstellungen", url.Values{
		"threshold": {"90"}, "ai_provider": {"local"}, "ai_base_url": {"http://192.168.8.10:11434/v1"}, "ai_model": {"local-model"},
	})
	if accepted.Code != http.StatusSeeOther {
		t.Fatalf("lan save status=%d body=%s", accepted.Code, accepted.Body.String())
	}
	saved, err := repo.Get(t.Context())
	if err != nil || saved.AIBaseURL != "http://192.168.8.10:11434/v1" {
		t.Fatalf("saved = %#v err=%v", saved, err)
	}
	events := a.auditStore.List(store.AuditFilter{TenantSlug: "demo", Action: store.AuditActionVerwaltungSettings})
	if len(events) != 1 {
		t.Fatalf("audit events = %d", len(events))
	}
	event := events[0]
	if event.Details["ai_origin_before"] != "Umgebung" || event.Details["ai_origin_after"] != "http://192.168.8.10:11434" {
		t.Fatalf("origins = %#v", event.Details)
	}
	if strings.Contains(event.Summary, key) || strings.Contains(event.Details["diff"], key) {
		t.Fatal("audit record contains the process key")
	}

	rejected := authedFormRequest(t, a, "vera@example.com", "/demo/app/verwaltung/einstellungen", url.Values{
		"threshold": {"90"}, "ai_provider": {"local"}, "ai_base_url": {"http://8.8.8.8/v1"}, "ai_model": {"local-model"},
	})
	if rejected.Code != http.StatusBadRequest || !strings.Contains(rejected.Body.String(), "nur im lokalen Netz") {
		t.Fatalf("public http status=%d body=%s", rejected.Code, rejected.Body.String())
	}
	saved, err = repo.Get(t.Context())
	if err != nil || saved.AIBaseURL != "http://192.168.8.10:11434/v1" {
		t.Fatalf("public http was stored: %#v err=%v", saved, err)
	}
	if got := len(a.auditStore.List(store.AuditFilter{TenantSlug: "demo", Action: store.AuditActionVerwaltungSettings})); got != 1 {
		t.Fatalf("rejected save wrote an audit event, count=%d", got)
	}
}

func TestHAUSV772UseTimeRejectsStoredPublicHTTP(t *testing.T) {
	a, _, repo := newInboxTestApp(t, roleAdmin)
	boot := hausv624Suggester{label: "Boot"}
	a.triage = boot
	settings := store.DefaultOrgSettings("musterstadt")
	settings.AIProvider = "local"
	settings.AIBaseURL = "http://example.com/v1"
	settings.AIModel = "stored-model"
	if err := repo.Save(t.Context(), settings); err != nil {
		t.Fatal(err)
	}
	calls := 0
	useHausv624Factory(t, func(func(string) string) (ai.TriageSuggester, error) {
		calls++
		return hausv624Suggester{label: "must not build"}, nil
	})

	got, label := a.triageFor(t.Context(), "musterstadt")
	if got != nil || label != aiUnavailableLabel || calls != 0 {
		t.Fatalf("use-time = %#v, %q; factory calls=%d", got, label, calls)
	}
	view, err := a.inboxCaseView(t.Context(), "musterstadt", store.IntakeItem{ID: "case-1", Subject: "Synthetisch", Body: "Prüfen.", ReceivedAt: time.Now()}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if view.ProviderLabel != aiUnavailableLabel || view.CanSuggest {
		t.Fatalf("case view label=%q canSuggest=%v", view.ProviderLabel, view.CanSuggest)
	}
}

func TestHAUSV772SettingsReadErrorFailsClosed(t *testing.T) {
	a, _, repo := newInboxTestApp(t, roleAdmin)
	boot := hausv624Suggester{label: "Boot"}
	a.triage = boot
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	a.orgSettings = func(string) store.OrgSettingsRepository {
		return failingOrgSettings{err: io.ErrUnexpectedEOF}
	}
	got, label := a.triageFor(t.Context(), "musterstadt")
	if got != nil || label != aiUnavailableLabel {
		t.Fatalf("read failure = %#v, %q", got, label)
	}
	if !strings.Contains(logs.String(), "organisation ai settings unavailable") || !strings.Contains(logs.String(), "musterstadt") {
		t.Fatalf("log = %s", logs.String())
	}

	a.orgSettings = func(orgKey string) store.OrgSettingsRepository { return repo }
	got, label = a.triageFor(t.Context(), "musterstadt")
	if got != boot || label != "Boot" {
		t.Fatalf("recovered provider = %#v, %q", got, label)
	}
}

func TestHAUSV772ConstructionErrorIsLoggedAndShown(t *testing.T) {
	a, _, repo := newInboxTestApp(t, roleAdmin)
	settings := store.DefaultOrgSettings("musterstadt")
	settings.AIProvider = "local"
	settings.AIBaseURL = "http://10.0.0.9:11434/v1"
	settings.AIModel = "local-model"
	if err := repo.Save(t.Context(), settings); err != nil {
		t.Fatal(err)
	}
	const key = "audit-synthetic-nonsecret"
	t.Setenv("AI_API_KEY", key)
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	useHausv624Factory(t, func(func(string) string) (ai.TriageSuggester, error) {
		return nil, io.ErrUnexpectedEOF
	})

	response := authedFormRequest(t, a, "vera@example.com", "/demo/app/verwaltung/einstellungen/ki-test", url.Values{})
	body := response.Body.String()
	if response.Code != http.StatusOK || !strings.Contains(body, "Verbindung fehlgeschlagen · "+aiUnavailableLabel) {
		t.Fatalf("status=%d body=%s", response.Code, body)
	}
	if strings.Contains(body, key) || strings.Contains(logs.String(), key) {
		t.Fatal("process key appeared in the page or the log")
	}
	if !strings.Contains(logs.String(), "organisation ai provider unavailable") {
		t.Fatalf("log = %s", logs.String())
	}
}

func TestHAUSV772SettingsPageNamesRejectedStoredDestination(t *testing.T) {
	a, _, repo := newInboxTestApp(t, roleAdmin)
	settings := store.DefaultOrgSettings("musterstadt")
	settings.AIProvider = "local"
	settings.AIBaseURL = "http://8.8.8.8/v1"
	settings.AIModel = "stored-model"
	if err := repo.Save(t.Context(), settings); err != nil {
		t.Fatal(err)
	}
	page := authedRequest(t, a, "vera@example.com", "/demo/app/verwaltung/einstellungen")
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), aiUnavailableLabel) {
		t.Fatalf("status=%d", page.Code)
	}
	calls := 0
	useHausv624Factory(t, func(func(string) string) (ai.TriageSuggester, error) {
		calls++
		return hausv624Suggester{label: "must not build"}, nil
	})
	tested := authedFormRequest(t, a, "vera@example.com", "/demo/app/verwaltung/einstellungen/ki-test", url.Values{})
	if tested.Code != http.StatusOK || !strings.Contains(tested.Body.String(), "Verbindung fehlgeschlagen · "+aiUnavailableLabel) || calls != 0 {
		t.Fatalf("status=%d calls=%d body=%s", tested.Code, calls, tested.Body.String())
	}
}

type aiAuditTransport func(*http.Request) (*http.Response, error)

func (f aiAuditTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type failingOrgSettings struct{ err error }

func (r failingOrgSettings) Get(context.Context) (store.OrgSettings, error) {
	return store.OrgSettings{}, r.err
}
func (r failingOrgSettings) Save(context.Context, store.OrgSettings) error { return r.err }
func (r failingOrgSettings) Increment(context.Context, string) error       { return r.err }
