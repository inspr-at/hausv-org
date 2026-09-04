package server

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/ai"
	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/web"
)

type hausv624Suggester struct{ label string }

func (s hausv624Suggester) Suggest(context.Context, ai.TriageInput) (ai.TriageSuggestion, error) {
	return ai.TriageSuggestion{}, nil
}

func (s hausv624Suggester) Label() string { return s.label }

func useHausv624Factory(t *testing.T, factory func(func(string) string) (ai.TriageSuggester, error)) {
	t.Helper()
	original := newSettingsAISuggester
	newSettingsAISuggester = factory
	t.Cleanup(func() { newSettingsAISuggester = original })
}

func TestHausv624TriageForUsesOrganisationOverride(t *testing.T) {
	a, _, repo := newInboxTestApp(t, roleAdmin)
	boot := hausv624Suggester{label: "Boot"}
	a.triage = boot
	settings := store.DefaultOrgSettings("musterstadt")
	settings.AIProvider = "cloud"
	settings.AIBaseURL = "https://provider.example/v1"
	settings.AIModel = "override-model"
	if err := repo.Save(t.Context(), settings); err != nil {
		t.Fatal(err)
	}
	var baseURL, model string
	configured := hausv624Suggester{label: "factory label"}
	useHausv624Factory(t, func(getenv func(string) string) (ai.TriageSuggester, error) {
		baseURL, model = getenv("AI_BASE_URL"), getenv("AI_MODEL")
		return configured, nil
	})

	got, label := a.triageFor(t.Context(), "musterstadt")
	if got != configured || label != "Cloud (OpenRouter)" {
		t.Fatalf("triageFor = %#v, %q", got, label)
	}
	if baseURL != settings.AIBaseURL || model != settings.AIModel {
		t.Fatalf("factory settings = %q / %q", baseURL, model)
	}
}

func TestHausv624TriageForUsesBootProviderWithoutOverrides(t *testing.T) {
	a, _, _ := newInboxTestApp(t, roleAdmin)
	boot := hausv624Suggester{label: "Boot"}
	a.triage = boot
	calls := 0
	useHausv624Factory(t, func(func(string) string) (ai.TriageSuggester, error) {
		calls++
		return nil, errors.New("must not be called")
	})

	got, label := a.triageFor(t.Context(), "musterstadt")
	if got != boot || label != "Boot" || calls != 0 {
		t.Fatalf("triageFor = %#v, %q; factory calls = %d", got, label, calls)
	}
}

func TestHausv624TriageForRefreshesChangedConfiguration(t *testing.T) {
	a, _, repo := newInboxTestApp(t, roleAdmin)
	settings := store.DefaultOrgSettings("musterstadt")
	settings.AIProvider = "local"
	settings.AIBaseURL = "http://local.example/v1"
	settings.AIModel = "first-model"
	if err := repo.Save(t.Context(), settings); err != nil {
		t.Fatal(err)
	}
	var calls []string
	useHausv624Factory(t, func(getenv func(string) string) (ai.TriageSuggester, error) {
		calls = append(calls, getenv("AI_BASE_URL")+"|"+getenv("AI_MODEL"))
		return hausv624Suggester{label: "configured"}, nil
	})

	a.triageFor(t.Context(), "musterstadt")
	a.triageFor(t.Context(), "musterstadt")
	settings.AIModel = "second-model"
	if err := repo.Save(t.Context(), settings); err != nil {
		t.Fatal(err)
	}
	a.triageFor(t.Context(), "musterstadt")
	if got, want := strings.Join(calls, ","), "http://local.example/v1|first-model,http://local.example/v1|second-model"; got != want {
		t.Fatalf("factory calls = %q, want %q", got, want)
	}
}

func TestHausv624TriageForFallsBackAfterFactoryError(t *testing.T) {
	a, _, repo := newInboxTestApp(t, roleAdmin)
	boot := hausv624Suggester{label: "Boot"}
	a.triage = boot
	settings := store.DefaultOrgSettings("musterstadt")
	settings.AIProvider = "local"
	settings.AIBaseURL = "http://local.example/v1"
	settings.AIModel = "broken-model"
	if err := repo.Save(t.Context(), settings); err != nil {
		t.Fatal(err)
	}
	useHausv624Factory(t, func(func(string) string) (ai.TriageSuggester, error) {
		return nil, errors.New("unavailable")
	})

	got, label := a.triageFor(t.Context(), "musterstadt")
	if got != boot || label != "Boot" {
		t.Fatalf("fallback = %#v, %q", got, label)
	}
}

func TestHausv624InboxRendersEffectiveAndStoredProviderWithoutAPIKey(t *testing.T) {
	a, _, repo := newInboxTestApp(t, roleAdmin)
	settings := store.DefaultOrgSettings("musterstadt")
	settings.AIProvider = "cloud"
	settings.AIBaseURL = "https://provider.example/v1"
	settings.AIModel = "cloud-model"
	if err := repo.Save(t.Context(), settings); err != nil {
		t.Fatal(err)
	}
	const apiKey = "dummy-ai-api-key-must-not-render"
	t.Setenv("AI_API_KEY", apiKey)
	useHausv624Factory(t, func(func(string) string) (ai.TriageSuggester, error) {
		return hausv624Suggester{label: "factory label"}, nil
	})

	page := authedRequest(t, a, "vera@example.com", "/demo/app/verwaltung/posteingang")
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "KI: Cloud (OpenRouter)") {
		t.Fatalf("inbox response = %d: %s", page.Code, page.Body.String())
	}
	if strings.Contains(page.Body.String(), apiKey) {
		t.Fatal("inbox HTML contains AI_API_KEY")
	}
	item := store.IntakeItem{ID: "case-1", Source: store.IntakeSourceEmail, Subject: "Vorschreibung", Body: "Bitte prüfen.", ReceivedAt: time.Now(), Suggestion: &store.IntakeSuggestion{Model: "stored-model", Provider: "Lokal (OpenAI-kompatibel)"}}
	caseView, err := a.inboxCaseView(t.Context(), "musterstadt", item, []web.InboxHouse{}, false)
	if err != nil {
		t.Fatal(err)
	}
	if caseView.ProviderLabel != "Lokal (OpenAI-kompatibel)" {
		t.Fatalf("stored provider label = %q", caseView.ProviderLabel)
	}
	var history bytes.Buffer
	if err := web.InboxTechnicalDetails(caseView).Render(t.Context(), &history); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(history.String(), "Anbieter: Lokal (OpenAI-kompatibel)") {
		t.Fatalf("history does not show stored provider: %s", history.String())
	}
}
