package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSuggestHappyPathAndHeaders(t *testing.T) {
	t.Parallel()
	var received completionRequest
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/chat/completions" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.Header.Get("HTTP-Referer"); got != "https://hausv.example" {
			t.Errorf("HTTP-Referer = %q", got)
		}
		if got := r.Header.Get("X-Title"); got != "HAUSV Demo" {
			t.Errorf("X-Title = %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &received); err != nil {
			t.Errorf("decode request: %v", err)
		}
		writeCompletion(t, w, validAnswerJSON())
	})

	suggester := newTestSuggester(handler)
	suggester.apiKey = "test-key"
	suggester.httpReferer = "https://hausv.example"
	suggester.appTitle = "HAUSV Demo"
	suggestion, err := suggester.Suggest(context.Background(), testInput())
	if err != nil {
		t.Fatalf("Suggest: %v", err)
	}
	if suggestion.Category != "reparatur" || suggestion.Priority != "Hoch" || suggestion.HouseSlug != "haus-a" {
		t.Fatalf("unexpected suggestion: %+v", suggestion)
	}
	if suggestion.Model != "test-model" || len(suggestion.PromptHash) != 64 || len(suggestion.Raw) == 0 {
		t.Fatalf("missing provenance: %+v", suggestion)
	}
	if received.Model != "test-model" || received.Temperature != 0.1 || received.MaxTokens != completionMaxTokens {
		t.Fatalf("unexpected request: %+v", received)
	}
	if got := received.ResponseFormat["type"]; got != "json_object" {
		t.Fatalf("response format = %v", got)
	}
	if len(received.Messages) != 2 || received.Messages[0].Role != "system" || received.Messages[1].Role != "user" {
		t.Fatalf("messages = %+v", received.Messages)
	}
}

func TestSuggestAcceptsFencedJSON(t *testing.T) {
	t.Parallel()
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeCompletion(t, w, "```json\n"+validAnswerJSON()+"\n```")
	})

	suggestion, err := newTestSuggester(handler).Suggest(context.Background(), testInput())
	if err != nil {
		t.Fatalf("Suggest: %v", err)
	}
	if suggestion.Category != "reparatur" {
		t.Fatalf("category = %q", suggestion.Category)
	}
}

func TestSuggestRejectsUnknownCategory(t *testing.T) {
	t.Parallel()
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeCompletion(t, w, strings.Replace(validAnswerJSON(), `"category":"reparatur"`, `"category":"erfunden"`, 1))
	})

	suggestion, err := newTestSuggester(handler).Suggest(context.Background(), testInput())
	if err == nil || !strings.Contains(err.Error(), "unknown category") {
		t.Fatalf("error = %v", err)
	}
	if !reflect.DeepEqual(suggestion, TriageSuggestion{}) {
		t.Fatalf("malformed answer returned suggestion: %+v", suggestion)
	}
}

// OpenRouter models answer with the vocabulary in their own casing ("hoch",
// "Reparatur"); the live demo rejected every such answer. The catalogue
// spelling wins, and the suggestion carries it.
func TestSuggestNormalisesVocabularyCase(t *testing.T) {
	t.Parallel()
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		answer := strings.Replace(validAnswerJSON(), `"priority":"Hoch"`, `"priority":" hoch"`, 1)
		answer = strings.Replace(answer, `"category":"reparatur"`, `"category":"Reparatur"`, 1)
		writeCompletion(t, w, answer)
	})

	suggestion, err := newTestSuggester(handler).Suggest(context.Background(), testInput())
	if err != nil {
		t.Fatalf("Suggest: %v", err)
	}
	if suggestion.Priority != "Hoch" || suggestion.Category != "reparatur" {
		t.Fatalf("priority=%q category=%q; want catalogue spelling", suggestion.Priority, suggestion.Category)
	}
}

// Live finding 2026-09-03: OpenRouter's gemini-3.8-flash spends hidden
// reasoning tokens before the JSON answer; with a 1200-token budget every
// triage came back truncated. The request now carries a large budget and,
// for OpenRouter only, a low reasoning effort; a truncated answer is named
// as such instead of surfacing as a JSON decode error.
func TestSuggestRequestBudgetAndReasoning(t *testing.T) {
	t.Parallel()
	var body map[string]any
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		writeCompletion(t, w, validAnswerJSON())
	})

	if _, err := newTestSuggester(handler).Suggest(context.Background(), testInput()); err != nil {
		t.Fatalf("Suggest: %v", err)
	}
	if got := body["max_tokens"]; got != float64(completionMaxTokens) {
		t.Fatalf("max_tokens = %v, want %d", got, completionMaxTokens)
	}
	if _, ok := body["reasoning"]; ok {
		t.Fatalf("reasoning must not be sent to a generic OpenAI-compatible server: %v", body["reasoning"])
	}

	s := newTestSuggester(handler)
	s.baseURL = "https://openrouter.ai/api/v1"
	if _, err := s.Suggest(context.Background(), testInput()); err != nil {
		t.Fatalf("Suggest via OpenRouter: %v", err)
	}
	reasoning, _ := body["reasoning"].(map[string]any)
	if reasoning["effort"] != "low" {
		t.Fatalf("reasoning = %v, want effort low for OpenRouter", body["reasoning"])
	}
}

func TestSuggestNamesTruncatedCompletion(t *testing.T) {
	t.Parallel()
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{"finish_reason": "length", "message": map[string]string{"content": `{"category":"reparatur","priority":"Hoch","reply":"Sehr geehr`}}},
		})
	})

	_, err := newTestSuggester(handler).Suggest(context.Background(), testInput())
	if err == nil || !strings.Contains(err.Error(), "truncated") {
		t.Fatalf("error = %v, want a truncation error", err)
	}
}

func TestSuggestFillsMissingConfidenceWithZero(t *testing.T) {
	t.Parallel()
	answer := `{"category":"reparatur","priority":"Hoch","house":"haus-a","unit":"Top 7","assignee":"vera","template_key":"antwort","reply":"Danke.","actions":[],"confidence":{"category":1},"reasoning":"Die Meldung nennt einen Schaden."}`
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeCompletion(t, w, answer)
	})
	suggester := newTestSuggester(handler)
	suggester.minConfidence = 0

	suggestion, err := suggester.Suggest(context.Background(), testInput())
	if err != nil {
		t.Fatalf("Suggest: %v", err)
	}
	if len(suggestion.Confidence) != len(confidenceKeys) {
		t.Fatalf("confidence = %#v", suggestion.Confidence)
	}
	for _, key := range confidenceKeys {
		want := 0.0
		if key == "category" {
			want = 1
		}
		if suggestion.Confidence[key] != want {
			t.Errorf("confidence[%q] = %v, want %v", key, suggestion.Confidence[key], want)
		}
	}
}

func TestSuggestRetriesOnceAfterServerError(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			http.Error(w, "temporary", http.StatusInternalServerError)
			return
		}
		writeCompletion(t, w, validAnswerJSON())
	})
	suggester := newTestSuggester(handler)
	suggester.retryBackoff = time.Millisecond

	if _, err := suggester.Suggest(context.Background(), testInput()); err != nil {
		t.Fatalf("Suggest: %v", err)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("calls = %d, want 2", got)
	}
}

func TestSuggestTimeoutReturnsContextError(t *testing.T) {
	t.Parallel()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	})
	suggester := newTestSuggester(handler)
	suggester.timeout = 20 * time.Millisecond

	suggestion, err := suggester.Suggest(context.Background(), testInput())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context deadline", err)
	}
	if !reflect.DeepEqual(suggestion, TriageSuggestion{}) {
		t.Fatalf("timeout returned suggestion: %+v", suggestion)
	}
}

func TestSuggestReturnsUncertainSuggestion(t *testing.T) {
	t.Parallel()
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		answer := strings.Replace(validAnswerJSON(), `"overall":0.92`, `"overall":0.2`, 1)
		writeCompletion(t, w, answer)
	})

	suggestion, err := newTestSuggester(handler).Suggest(context.Background(), testInput())
	if !errors.Is(err, ErrUncertain) || suggestion.Category != "reparatur" {
		t.Fatalf("suggestion = %+v, error = %v", suggestion, err)
	}
}

func TestConfidenceIsClamped(t *testing.T) {
	t.Parallel()
	answer := strings.Replace(validAnswerJSON(), `"category":0.95`, `"category":7`, 1)
	answer = strings.Replace(answer, `"unit":0.80`, `"unit":-2`, 1)
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeCompletion(t, w, answer)
	})
	suggestion, err := newTestSuggester(handler).Suggest(context.Background(), testInput())
	if err != nil {
		t.Fatalf("Suggest: %v", err)
	}
	if suggestion.Confidence["category"] != 1 || suggestion.Confidence["unit"] != 0 {
		t.Fatalf("confidence = %#v", suggestion.Confidence)
	}
}

func newTestSuggester(handler http.Handler) *openAICompatSuggester {
	return &openAICompatSuggester{
		baseURL: "http://ai.test", model: "test-model", label: "Test",
		timeout: time.Second, minConfidence: 0.6,
		client: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)
			if err := req.Context().Err(); err != nil {
				return nil, err
			}
			return recorder.Result(), nil
		})},
		retryBackoff: time.Millisecond,
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func testInput() TriageInput {
	return TriageInput{
		Organisation: "Hausverwaltung Beispiel", Source: "email", Subject: "Wasser im Bad",
		Body: "Bei Top 7 tropft Wasser.", FromName: "Erika Beispiel",
		FromEmail: "erika@example.test", ReceivedAt: time.Date(2026, 9, 3, 10, 0, 0, 0, time.FixedZone("CEST", 2*60*60)),
		Houses:     []HouseHint{{Slug: "haus-a", Name: "Haus A", Address: "Gasse 1", Units: []string{"Top 7"}}},
		Categories: []CategoryRule{{Key: "reparatur", Label: "Reparatur", Description: "Schäden", DefaultPriority: "Mittel"}},
		Templates:  []TemplateHint{{Key: "antwort", Category: "reparatur", Title: "Antwort", Body: "Guten Tag {{Name}}"}},
		Assignees:  []AssigneeHint{{Key: "vera", Name: "Vera", Houses: []string{"haus-a"}}},
	}
}

func validAnswerJSON() string {
	return `{"category":"reparatur","priority":"Hoch","house":"haus-a","unit":"Top 7","assignee":"vera","template_key":"antwort","reply":"Guten Tag Erika Beispiel, danke für Ihre Meldung.","actions":["Schaden prüfen"],"confidence":{"category":0.95,"priority":0.90,"house":0.98,"unit":0.80,"assignee":0.85,"overall":0.92},"reasoning":"Die Meldung beschreibt einen konkreten Wasserschaden."}`
}

func writeCompletion(t *testing.T, w http.ResponseWriter, content string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"choices": []any{map[string]any{"message": map[string]string{"content": content}}},
	}); err != nil {
		t.Errorf("write response: %v", err)
	}
}

func TestHTTPErrorBodyIsTruncated(t *testing.T) {
	t.Parallel()
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprint(w, strings.Repeat("x", maxErrorBodyBytes+200))
	})
	_, err := newTestSuggester(handler).Suggest(context.Background(), testInput())
	if err == nil || len(err.Error()) > maxErrorBodyBytes+100 {
		t.Fatalf("error was not bounded: %v", err)
	}
}
