package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
)

const (
	maxResponseBytes  = 1 << 20
	maxErrorBodyBytes = 1024
)

var confidenceKeys = [...]string{"category", "priority", "house", "unit", "assignee", "overall"}

type openAICompatSuggester struct {
	baseURL       string
	apiKey        string
	model         string
	label         string
	appTitle      string
	httpReferer   string
	timeout       time.Duration
	minConfidence float64
	client        *http.Client
	retryBackoff  time.Duration
}

type completionRequest struct {
	Model          string         `json:"model"`
	Messages       []chatMessage  `json:"messages"`
	Temperature    float64        `json:"temperature"`
	ResponseFormat map[string]any `json:"response_format"`
	MaxTokens      int            `json:"max_tokens"`
	// Reasoning is OpenRouter's unified knob. Reasoning models spend their
	// budget on hidden thinking first (measured 2026-09-03: gemini-3.8-flash
	// used 206 reasoning tokens for a one-line answer), so a small budget
	// returns truncated JSON. Only sent to OpenRouter; other OpenAI-compatible
	// servers may reject unknown fields.
	Reasoning map[string]any `json:"reasoning,omitempty"`
}

// completionMaxTokens leaves room for hidden reasoning plus the reply text and
// actions; the answer itself stays well under a thousand tokens.
const completionMaxTokens = 4000

type completionResponse struct {
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Message      struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// reasoningFor returns the OpenRouter reasoning setting for the base URL:
// low effort keeps the hidden thinking short so the JSON answer fits.
func reasoningFor(baseURL string) map[string]any {
	if strings.Contains(strings.ToLower(baseURL), "openrouter.ai") {
		return map[string]any{"effort": "low"}
	}
	return nil
}

type triageAnswer struct {
	Category    string             `json:"category"`
	Priority    string             `json:"priority"`
	House       string             `json:"house"`
	Unit        string             `json:"unit"`
	Assignee    string             `json:"assignee"`
	TemplateKey string             `json:"template_key"`
	Reply       string             `json:"reply"`
	Actions     []string           `json:"actions"`
	Confidence  map[string]float64 `json:"confidence"`
	Reasoning   string             `json:"reasoning"`
}

func (s *openAICompatSuggester) Label() string { return s.label }

func (s *openAICompatSuggester) Suggest(ctx context.Context, in TriageInput) (TriageSuggestion, error) {
	if s == nil || s.client == nil || s.baseURL == "" || s.model == "" {
		return TriageSuggestion{}, ErrUnavailable
	}
	messages, promptHash, err := buildPrompt(in)
	if err != nil {
		return TriageSuggestion{}, err
	}
	payload, err := json.Marshal(completionRequest{
		Model: s.model, Messages: messages, Temperature: 0.1,
		ResponseFormat: map[string]any{"type": "json_object"}, MaxTokens: completionMaxTokens,
		Reasoning: reasoningFor(s.baseURL),
	})
	if err != nil {
		return TriageSuggestion{}, fmt.Errorf("ai: encode completion request: %w", err)
	}
	requestCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	var responseBody []byte
	for attempt := 0; attempt < 2; attempt++ {
		responseBody, err = s.complete(requestCtx, payload)
		var statusErr *completionStatusError
		if err == nil || !errors.As(err, &statusErr) || !statusErr.retryable() || attempt == 1 {
			break
		}
		timer := time.NewTimer(s.retryBackoff)
		select {
		case <-requestCtx.Done():
			timer.Stop()
			return TriageSuggestion{}, fmt.Errorf("ai: completion retry: %w", requestCtx.Err())
		case <-timer.C:
		}
	}
	if err != nil {
		return TriageSuggestion{}, err
	}
	suggestion, err := parseSuggestion(responseBody, in, s.model, promptHash)
	if err != nil {
		return TriageSuggestion{}, err
	}
	if suggestion.Confidence["overall"] < s.minConfidence {
		return suggestion, ErrUncertain
	}
	return suggestion, nil
}

func (s *openAICompatSuggester) complete(ctx context.Context, payload []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("ai: create completion request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}
	if s.httpReferer != "" {
		req.Header.Set("HTTP-Referer", s.httpReferer)
	}
	if s.appTitle != "" {
		req.Header.Set("X-Title", s.appTitle)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ai: completion request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes+1))
		if len(body) > maxErrorBodyBytes {
			body = body[:maxErrorBodyBytes]
		}
		return nil, &completionStatusError{statusCode: resp.StatusCode, body: strings.TrimSpace(string(body))}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("ai: read completion response: %w", err)
	}
	if len(body) > maxResponseBytes {
		return nil, errors.New("ai: completion response is too large")
	}
	return body, nil
}

type completionStatusError struct {
	statusCode int
	body       string
}

func (e *completionStatusError) Error() string {
	if e.body == "" {
		return fmt.Sprintf("ai: completion returned HTTP %d", e.statusCode)
	}
	return fmt.Sprintf("ai: completion returned HTTP %d: %s", e.statusCode, e.body)
}

func (e *completionStatusError) retryable() bool {
	return e.statusCode == http.StatusTooManyRequests || (e.statusCode >= 500 && e.statusCode <= 599)
}

func parseSuggestion(responseBody []byte, in TriageInput, model, promptHash string) (TriageSuggestion, error) {
	var completion completionResponse
	if err := json.Unmarshal(responseBody, &completion); err != nil {
		return TriageSuggestion{}, fmt.Errorf("ai: decode completion response: %w", err)
	}
	if len(completion.Choices) == 0 {
		return TriageSuggestion{}, errors.New("ai: completion contained no choices")
	}
	if completion.Choices[0].FinishReason == "length" {
		return TriageSuggestion{}, errors.New("ai: completion truncated by the token budget")
	}
	raw := stripCodeFence(completion.Choices[0].Message.Content)
	if raw == "" {
		return TriageSuggestion{}, errors.New("ai: completion choice was empty")
	}
	var answer triageAnswer
	if err := json.Unmarshal([]byte(raw), &answer); err != nil {
		return TriageSuggestion{}, fmt.Errorf("ai: decode triage answer: %w", err)
	}
	// Models paraphrase the vocabulary's casing ("hoch", "Reparatur"); the
	// catalogue spelling is authoritative, so match case-insensitively and
	// store the canonical key.
	category, ok := canonicalCategory(in.Categories, answer.Category)
	if !ok {
		return TriageSuggestion{}, fmt.Errorf("ai: unknown category %q", answer.Category)
	}
	answer.Category = category
	priority, ok := canonicalPriority(answer.Priority)
	if !ok {
		return TriageSuggestion{}, fmt.Errorf("ai: unknown priority %q", answer.Priority)
	}
	answer.Priority = priority
	if answer.House != "" && !containsHouse(in.Houses, answer.House) {
		return TriageSuggestion{}, fmt.Errorf("ai: unknown house %q", answer.House)
	}
	if answer.Assignee != "" && !containsAssignee(in.Assignees, answer.Assignee) {
		return TriageSuggestion{}, fmt.Errorf("ai: unknown assignee %q", answer.Assignee)
	}
	if answer.TemplateKey != "" && !containsTemplate(in.Templates, answer.TemplateKey) {
		return TriageSuggestion{}, fmt.Errorf("ai: unknown template %q", answer.TemplateKey)
	}
	confidence := make(map[string]float64, len(confidenceKeys))
	for _, key := range confidenceKeys {
		confidence[key] = clampConfidence(answer.Confidence[key])
	}
	return TriageSuggestion{
		Category: answer.Category, Priority: answer.Priority, HouseSlug: answer.House,
		Unit: answer.Unit, Assignee: answer.Assignee, TemplateKey: answer.TemplateKey,
		Reply: answer.Reply, Actions: answer.Actions, Confidence: confidence,
		Model: model, PromptHash: promptHash, Raw: json.RawMessage(append([]byte(nil), raw...)),
	}, nil
}

func stripCodeFence(content string) string {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "```") {
		return content
	}
	firstLine := strings.IndexByte(content, '\n')
	if firstLine < 0 {
		return content
	}
	content = content[firstLine+1:]
	if end := strings.LastIndex(content, "```"); end >= 0 && strings.TrimSpace(content[end+3:]) == "" {
		content = content[:end]
	}
	return strings.TrimSpace(content)
}

func clampConfidence(value float64) float64 {
	if math.IsNaN(value) || value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func containsCategory(items []CategoryRule, value string) bool {
	for _, item := range items {
		if item.Key == value {
			return true
		}
	}
	return false
}

func containsHouse(items []HouseHint, value string) bool {
	for _, item := range items {
		if item.Slug == value {
			return true
		}
	}
	return false
}

func containsAssignee(items []AssigneeHint, value string) bool {
	for _, item := range items {
		if item.Key == value {
			return true
		}
	}
	return false
}

func containsTemplate(items []TemplateHint, value string) bool {
	for _, item := range items {
		if item.Key == value {
			return true
		}
	}
	return false
}

var priorities = []string{"Niedrig", "Mittel", "Hoch", "Dringend"}

func validPriority(value string) bool {
	_, ok := canonicalPriority(value)
	return ok
}

// canonicalPriority maps any casing of a known priority to its catalogue spelling.
func canonicalPriority(value string) (string, bool) {
	value = strings.TrimSpace(value)
	for _, known := range priorities {
		if strings.EqualFold(known, value) {
			return known, true
		}
	}
	return "", false
}

// canonicalCategory maps any casing of a known category key to the catalogue key.
func canonicalCategory(items []CategoryRule, value string) (string, bool) {
	value = strings.TrimSpace(value)
	for _, item := range items {
		if strings.EqualFold(item.Key, value) {
			return item.Key, true
		}
	}
	return "", false
}
