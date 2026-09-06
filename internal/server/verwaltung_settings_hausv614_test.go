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
	environment := map[string]string{
		"AI_BASE_URL":       "https://openrouter.ai/api/v1?source=demo",
		"AI_MODEL":          "cloud-model",
		"AI_TIMEOUT":        "12s",
		"AI_PROVIDER_LABEL": "Cloud-Dienst",
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
}

func TestValidateAIBaseURLRejectsCredentials(t *testing.T) {
	if err := validateAIBaseURL("https://user:pass@example.test/v1"); err == nil {
		t.Fatal("URL userinfo was accepted")
	}
	if err := validateAIBaseURL("ftp://example.test/v1"); err == nil {
		t.Fatal("non-http URL was accepted")
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
	original := newSettingsAISuggester
	t.Cleanup(func() { newSettingsAISuggester = original })
	calls := 0
	newSettingsAISuggester = func(getenv func(string) string) (ai.TriageSuggester, error) {
		calls++
		if getenv("AI_API_KEY") != secret || getenv("AI_MODEL") != "override-model" {
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
