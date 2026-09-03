package ai

import (
	"strings"
	"testing"
)

func TestNewFromEnvEmptyBaseReturnsNil(t *testing.T) {
	t.Parallel()
	suggester, err := NewFromEnv(envMap(nil))
	if err != nil || suggester != nil {
		t.Fatalf("suggester = %v, error = %v", suggester, err)
	}
}

func TestNewFromEnvDerivesLabels(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		baseURL string
		label   string
		want    string
	}{
		{name: "OpenRouter", baseURL: "https://openrouter.ai/api/v1", want: "Cloud (OpenRouter)"},
		{name: "local", baseURL: "http://mac-studio.local:11434/v1", want: "lokal"},
		{name: "explicit", baseURL: "https://openrouter.ai/api/v1", label: "Demo-Cloud", want: "Demo-Cloud"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			suggester, err := NewFromEnv(envMap(map[string]string{
				"AI_BASE_URL": test.baseURL, "AI_MODEL": "demo", "AI_PROVIDER_LABEL": test.label,
			}))
			if err != nil {
				t.Fatalf("NewFromEnv: %v", err)
			}
			if got := suggester.Label(); got != test.want {
				t.Fatalf("Label = %q, want %q", got, test.want)
			}
		})
	}
}

func TestNewFromEnvRejectsNaNConfidence(t *testing.T) {
	t.Parallel()
	_, err := NewFromEnv(envMap(map[string]string{
		"AI_BASE_URL": "https://openrouter.ai/api/v1",
		"AI_MODEL":    "demo", "AI_MIN_CONFIDENCE": "NaN",
	}))
	if err == nil {
		t.Fatal("NewFromEnv accepted NaN confidence")
	}
}

func TestPromptContainsEveryCatalogueKeyAndStableHash(t *testing.T) {
	t.Parallel()
	in := testInput()
	in.Houses = append(in.Houses, HouseHint{Slug: "haus-b", Name: "Haus B"})
	in.Categories = append(in.Categories, CategoryRule{Key: "abrechnung", Label: "Abrechnung"})
	in.Templates = append(in.Templates, TemplateHint{Key: "abrechnung-antwort", Category: "abrechnung"})
	messages, hashOne, err := buildPrompt(in)
	if err != nil {
		t.Fatalf("buildPrompt: %v", err)
	}
	_, hashTwo, err := buildPrompt(in)
	if err != nil {
		t.Fatalf("buildPrompt again: %v", err)
	}
	if hashOne != hashTwo || len(hashOne) != 64 {
		t.Fatalf("hashes = %q, %q", hashOne, hashTwo)
	}
	combined := messages[0].Content + messages[1].Content
	for _, want := range []string{"haus-a", "haus-b", "reparatur", "abrechnung", "antwort", "abrechnung-antwort"} {
		if !strings.Contains(combined, want) {
			t.Errorf("prompt does not contain %q", want)
		}
	}
	if !strings.Contains(combined, "WEG") || !strings.Contains(combined, "MRG") {
		t.Error("prompt does not establish Austrian WEG/MRG context")
	}
}

func envMap(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}
