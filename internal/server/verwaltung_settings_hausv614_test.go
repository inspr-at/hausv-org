package server

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/ai"
	"github.com/inspr-at/hausv-org/internal/store"
)

func TestEffectiveAIConfigUsesEnvironmentAndOverrides(t *testing.T) {
	const syntheticKey = "audit-synthetic-nonsecret"
	environment := map[string]string{
		"AI_BASE_URL":       "https://openrouter.ai/api/v1?source=demo",
		"AI_MODEL":          "cloud-model",
		"AI_TIMEOUT":        "12s",
		"AI_PROVIDER_LABEL": "Cloud-Dienst",
		"AI_API_KEY":        syntheticKey,
	}
	getenv := func(key string) string { return environment[key] }

	fromEnv := effectiveAIConfig(getenv, store.OrgSettings{})
	if fromEnv.Provider != "cloud" || fromEnv.Label != "Cloud-Dienst" || fromEnv.Host != "openrouter.ai" || fromEnv.Model != "cloud-model" || fromEnv.Timeout != "12s" {
		t.Fatalf("environment config = %#v", fromEnv)
	}
	if strings.Contains(fromEnv.Host, "source") {
		t.Fatalf("display host leaked URL query: %q", fromEnv.Host)
	}

	override := effectiveAIConfig(getenv, store.OrgSettings{AIProvider: "local", AIBaseURL: "http://localhost:11434/v1", AIModel: "local-model"})
	if override.Provider != "local" || override.Label != "Lokal (OpenAI-kompatibel)" || override.Host != "localhost:11434" || override.Model != "local-model" {
		t.Fatalf("override config = %#v", override)
	}
	resolved := aiSettingsGetenv(getenv, store.OrgSettings{AIBaseURL: override.BaseURL, AIModel: override.Model})
	if resolved("AI_BASE_URL") != "http://localhost:11434/v1" || resolved("AI_MODEL") != "local-model" {
		t.Fatalf("resolved override = %q / %q", resolved("AI_BASE_URL"), resolved("AI_MODEL"))
	}
	// HAUSV-772: localhost is not the operator origin, so the process key stops here.
	if resolved("AI_API_KEY") != "" {
		t.Fatal("foreign override inherited the process key")
	}
	if got := aiSettingsGetenv(getenv, store.OrgSettings{})("AI_API_KEY"); got != syntheticKey {
		t.Fatal("operator destination lost its process key")
	}
	sameOrigin := aiSettingsGetenv(getenv, store.OrgSettings{AIBaseURL: "https://openrouter.ai/api/v1/extra"})
	if sameOrigin("AI_API_KEY") != syntheticKey || sameOrigin("AI_BASE_URL") != "https://openrouter.ai/api/v1/extra" {
		t.Fatal("same-origin override lost the process key or the override URL")
	}
}

func TestValidateAIBaseURLRejectsCredentials(t *testing.T) {
	if err := validateAIBaseURL("https://user:pass@example.test/v1"); err == nil {
		t.Fatal("URL userinfo was accepted")
	}
	if err := validateAIBaseURL("https://example.test/v1#abschnitt"); err == nil {
		t.Fatal("URL fragment was accepted")
	}
	if err := validateAIBaseURL("ftp://example.test/v1"); err == nil {
		t.Fatal("non-http URL was accepted")
	}
	if err := validateAIBaseURL("http://8.8.8.8/v1"); err == nil {
		t.Fatal("public http URL was accepted")
	}
	if err := validateAIBaseURL("http://192.168.8.10/v1"); err != nil {
		t.Fatalf("LAN http URL rejected: %v", err)
	}
	if err := validateAIBaseURL("https://example.test/v1"); err != nil {
		t.Fatalf("valid URL rejected: %v", err)
	}
}

type settingsAITestSuggester struct {
	suggestion ai.TriageSuggestion
	err        error
}

func (s settingsAITestSuggester) Suggest(context.Context, ai.TriageInput) (ai.TriageSuggestion, error) {
	return s.suggestion, s.err
}

func (settingsAITestSuggester) Label() string { return "Test" }

func TestVerwaltungAITestUsesFakeAndNeverRendersAPIKey(t *testing.T) {
	a, _, settingsRepo := newInboxTestApp(t, roleAdmin)
	settings := store.DefaultOrgSettings("musterstadt")
	settings.AIProvider = "cloud"
	settings.AIBaseURL = "https://openrouter.ai/api/v1"
	settings.AIModel = "override-model"
	if err := settingsRepo.Save(context.Background(), settings); err != nil {
		t.Fatal(err)
	}
	const secret = "dummy-api-key-must-not-appear"
	t.Setenv("AI_API_KEY", secret)
	t.Setenv("AI_BASE_URL", "https://operator.example/v1")
	original := newSettingsAISuggester
	t.Cleanup(func() { newSettingsAISuggester = original })
	calls := 0
	newSettingsAISuggester = func(getenv func(string) string) (ai.TriageSuggester, error) {
		calls++
		// HAUSV-772: openrouter.ai is not the operator origin, so the factory
		// must not observe the process key. The page still must not render it.
		if getenv("AI_API_KEY") != "" || getenv("AI_MODEL") != "override-model" {
			return nil, errors.New("test resolver mismatch")
		}
		return settingsAITestSuggester{suggestion: ai.TriageSuggestion{Model: "resolved-model"}}, nil
	}

	response := authedFormRequest(t, a, "vera@example.com", "/demo/app/verwaltung/einstellungen/ki-test", url.Values{})
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "Verbindung ok") || !strings.Contains(response.Body.String(), "resolved-model") {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if calls != 1 {
		t.Fatalf("factory calls = %d, want 1", calls)
	}
	if strings.Contains(response.Body.String(), secret) {
		t.Fatal("HTML response contains AI_API_KEY")
	}
}

func TestVerwaltungSettingsRejectsAIURLWithUserinfo(t *testing.T) {
	a, _, _ := newInboxTestApp(t, roleAdmin)
	response := authedFormRequest(t, a, "vera@example.com", "/demo/app/verwaltung/einstellungen", url.Values{
		"threshold": {"90"}, "ai_provider": {"cloud"}, "ai_base_url": {"https://user:pass@example.test/v1"},
	})
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

// HAUSV-700: a valid answer below the confidence threshold proves the
// connection works; the test must say so instead of "fehlgeschlagen".
func TestVerwaltungAITestTreatsUncertainAnswerAsWorkingConnection(t *testing.T) {
	a, _, _ := newInboxTestApp(t, roleAdmin)
	t.Setenv("AI_MIN_CONFIDENCE", "0.75")
	original := newSettingsAISuggester
	t.Cleanup(func() { newSettingsAISuggester = original })
	newSettingsAISuggester = func(getenv func(string) string) (ai.TriageSuggester, error) {
		return settingsAITestSuggester{suggestion: ai.TriageSuggestion{Model: "unsure-model", Confidence: map[string]float64{"overall": 0.42}}, err: ai.ErrUncertain}, nil
	}
	response := authedFormRequest(t, a, "vera@example.com", "/demo/app/verwaltung/einstellungen/ki-test", url.Values{})
	body := aiTestResultLine(t, response.Body.String())
	for _, want := range []string{"Verbindung ok", "unsure-model", "Zuversicht 42 %", "Schwelle 75 %", `data-ok="true"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("status=%d result lacks %q: %s", response.Code, want, body)
		}
	}
	if strings.Contains(body, "fehlgeschlagen") || strings.Contains(body, "uncertain") {
		t.Fatalf("uncertain answer reported as failure: %s", body)
	}
}

// HAUSV-700: real failures stay failures, in German, without raw provider text.
func TestVerwaltungAITestExplainsRealFailuresInGerman(t *testing.T) {
	a, _, _ := newInboxTestApp(t, roleAdmin)
	original := newSettingsAISuggester
	t.Cleanup(func() { newSettingsAISuggester = original })
	newSettingsAISuggester = func(getenv func(string) string) (ai.TriageSuggester, error) {
		return settingsAITestSuggester{err: errors.New("ai: completion status 401 unauthorized: invalid api key sk-secret")}, nil
	}
	response := authedFormRequest(t, a, "vera@example.com", "/demo/app/verwaltung/einstellungen/ki-test", url.Values{})
	body := aiTestResultLine(t, response.Body.String())
	if !strings.Contains(body, "Verbindung fehlgeschlagen · Anmeldung beim Anbieter abgelehnt") || !strings.Contains(body, `data-ok="false"`) {
		t.Fatalf("status=%d result=%s", response.Code, body)
	}
	if strings.Contains(body, "sk-secret") || strings.Contains(body, "unauthorized") {
		t.Fatalf("raw provider text leaked: %s", body)
	}
}

// aiTestResultLine isolates the KI-Verbindung status line from the settings page.
func aiTestResultLine(t *testing.T, html string) string {
	t.Helper()
	start := strings.Index(html, `<p class="settings-result"`)
	if start < 0 {
		t.Fatalf("settings result line missing: %s", html)
	}
	end := strings.Index(html[start:], "</p>")
	if end < 0 {
		t.Fatalf("settings result line unterminated")
	}
	return html[start : start+end]
}
