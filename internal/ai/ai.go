// Package ai provides fail-closed AI suggestions for human-reviewed triage.
// No provider is installed when AI_BASE_URL is empty, and callers must treat a
// nil suggester or any provider error as unavailable rather than accepting
// guessed data. The OpenAI-compatible transport supports OpenRouter for the
// current demonstration and the same API on a local Mac Studio later.
package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	// ErrUnavailable indicates that no AI triage provider has been configured.
	ErrUnavailable = errors.New("ai: no triage provider configured")
	// ErrUncertain indicates a valid suggestion below the configured confidence
	// threshold. Suggest returns the suggestion alongside this error so a caller
	// may present it explicitly as uncertain for human review.
	ErrUncertain = errors.New("ai: triage suggestion is uncertain")
)

type HouseHint struct {
	Slug    string   `json:"slug"`
	Name    string   `json:"name"`
	Address string   `json:"address"`
	Units   []string `json:"units"`
}

type CategoryRule struct {
	Key             string `json:"key"`
	Label           string `json:"label"`
	Description     string `json:"description"`
	DefaultPriority string `json:"default_priority"`
}

type TemplateHint struct {
	Key      string `json:"key"`
	Category string `json:"category"`
	Title    string `json:"title"`
	Body     string `json:"body"`
}

type AssigneeHint struct {
	Key    string   `json:"key"`
	Name   string   `json:"name"`
	Houses []string `json:"houses"`
}

type TriageInput struct {
	Organisation      string
	AssignedHouseSlug string
	AssignedHouseName string
	AssignedUnit      string
	Source            string
	Subject           string
	Body              string
	FromName          string
	FromEmail         string
	FromPhone         string
	ReceivedAt        time.Time
	Houses            []HouseHint
	Categories        []CategoryRule
	Templates         []TemplateHint
	Assignees         []AssigneeHint
}

type TriageSuggestion struct {
	Category    string             `json:"category"`
	Priority    string             `json:"priority"`
	HouseSlug   string             `json:"house"`
	Unit        string             `json:"unit"`
	Assignee    string             `json:"assignee"`
	TemplateKey string             `json:"template_key"`
	Reply       string             `json:"reply"`
	Actions     []string           `json:"actions"`
	Confidence  map[string]float64 `json:"confidence"`
	Model       string             `json:"model"`
	PromptHash  string             `json:"prompt_hash"`
	Raw         json.RawMessage    `json:"raw"`
}

type TriageSuggester interface {
	Suggest(ctx context.Context, in TriageInput) (TriageSuggestion, error)
	Label() string
}

// NewFromEnv constructs an OpenAI-compatible suggester. An empty AI_BASE_URL
// deliberately returns (nil, nil): callers treat nil as AI being unavailable.
// The client refuses cross-origin redirects. AI_ENFORCE_DESTINATION_POLICY=1
// rejects bases that fail ValidateDestination.
func NewFromEnv(getenv func(string) string) (TriageSuggester, error) {
	if getenv == nil {
		return nil, errors.New("ai: getenv function is nil")
	}
	baseURL := strings.TrimSpace(getenv("AI_BASE_URL"))
	if baseURL == "" {
		return nil, nil
	}
	parsed, err := url.Parse(baseURL)
	scheme := ""
	if parsed != nil {
		scheme = strings.ToLower(parsed.Scheme)
	}
	if err != nil || scheme == "" || parsed.Host == "" || (scheme != "http" && scheme != "https") {
		return nil, fmt.Errorf("ai: invalid AI_BASE_URL")
	}
	model := strings.TrimSpace(getenv("AI_MODEL"))
	if model == "" {
		return nil, errors.New("ai: AI_MODEL is required when AI_BASE_URL is set")
	}
	timeout := 45 * time.Second
	if raw := strings.TrimSpace(getenv("AI_TIMEOUT")); raw != "" {
		timeout, err = time.ParseDuration(raw)
		if err != nil || timeout <= 0 {
			return nil, errors.New("ai: AI_TIMEOUT must be a positive duration")
		}
	}
	minConfidence := 0.6
	if raw := strings.TrimSpace(getenv("AI_MIN_CONFIDENCE")); raw != "" {
		minConfidence, err = strconv.ParseFloat(raw, 64)
		if err != nil || math.IsNaN(minConfidence) || minConfidence < 0 || minConfidence > 1 {
			return nil, errors.New("ai: AI_MIN_CONFIDENCE must be between 0 and 1")
		}
	}
	label := strings.TrimSpace(getenv("AI_PROVIDER_LABEL"))
	if label == "" {
		label = "lokal"
		if strings.Contains(strings.ToLower(parsed.Host), "openrouter") {
			label = "Cloud (OpenRouter)"
		}
	}
	// Organisation overrides opt in. The operator's own base URL stays on the
	// existing parser so a process-wide destination is not judged as a tenant
	// choice. Callers withhold AI_API_KEY themselves unless this URL is the
	// operator origin; this function sends whatever key it is given.
	if strings.TrimSpace(getenv("AI_ENFORCE_DESTINATION_POLICY")) == "1" {
		if err := ValidateDestination(baseURL); err != nil {
			return nil, fmt.Errorf("ai: organisation destination rejected: %w", err)
		}
	}
	return &openAICompatSuggester{
		baseURL:       strings.TrimRight(baseURL, "/"),
		apiKey:        strings.TrimSpace(getenv("AI_API_KEY")),
		model:         model,
		label:         label,
		appTitle:      strings.TrimSpace(getenv("AI_APP_TITLE")),
		httpReferer:   strings.TrimSpace(getenv("AI_HTTP_REFERER")),
		timeout:       timeout,
		minConfidence: minConfidence,
		client:        aiHTTPClient(),
		retryBackoff:  100 * time.Millisecond,
	}, nil
}
