package server

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/inspr-at/hausv-org/internal/ai"
	"github.com/inspr-at/hausv-org/internal/store"
)

// triageProviderCacheEntry contains no credentials. Its fingerprint identifies
// only the organisation's selected provider endpoint and model.
type triageProviderCacheEntry struct {
	fingerprint string
	suggester   ai.TriageSuggester
	label       string
}

// triageFor resolves the configured suggester for an organisation. The boot
// provider remains the single environment fallback and is never rebuilt.
func (a *app) triageFor(ctx context.Context, orgKey string) (ai.TriageSuggester, string) {
	if a == nil {
		return nil, ""
	}
	fallback, fallbackLabel := a.triage, triageSuggesterLabel(a.triage)
	orgKey = normalizeSlug(orgKey)
	if orgKey == "" || a.orgSettings == nil {
		return fallback, fallbackLabel
	}
	repo := a.orgSettings(orgKey)
	if repo == nil {
		return fallback, fallbackLabel
	}
	settings, err := repo.Get(ctx)
	if err != nil {
		logError("organisation ai settings unavailable", err, "organisation", orgKey)
		return fallback, fallbackLabel
	}
	if settings.AIProvider == "" && settings.AIBaseURL == "" && settings.AIModel == "" {
		return fallback, fallbackLabel
	}

	fingerprint := triageProviderFingerprint(settings)
	a.triageProvidersMu.Lock()
	defer a.triageProvidersMu.Unlock()
	if a.triageProviders == nil {
		a.triageProviders = map[string]triageProviderCacheEntry{}
	}
	if cached, ok := a.triageProviders[orgKey]; ok && cached.fingerprint == fingerprint {
		return cached.suggester, cached.label
	}

	configured, buildErr := newSettingsAISuggester(aiSettingsGetenv(os.Getenv, settings))
	if buildErr != nil || configured == nil {
		if buildErr == nil {
			buildErr = fmt.Errorf("ai provider unavailable")
		}
		logError("organisation ai provider unavailable; using environment provider", buildErr, "organisation", orgKey)
		a.triageProviders[orgKey] = triageProviderCacheEntry{fingerprint: fingerprint, suggester: fallback, label: fallbackLabel}
		return fallback, fallbackLabel
	}
	entry := triageProviderCacheEntry{fingerprint: fingerprint, suggester: configured, label: effectiveAIConfig(os.Getenv, settings).Label}
	a.triageProviders[orgKey] = entry
	return entry.suggester, entry.label
}

func (a *app) hasTriage(ctx context.Context, orgKey string) bool {
	suggester, _ := a.triageFor(ctx, orgKey)
	return suggester != nil
}

func triageProviderFingerprint(settings store.OrgSettings) string {
	return strings.Join([]string{
		strings.TrimSpace(settings.AIProvider),
		strings.TrimSpace(settings.AIBaseURL),
		strings.TrimSpace(settings.AIModel),
	}, "\x00")
}

func triageSuggesterLabel(suggester ai.TriageSuggester) string {
	if suggester == nil {
		return ""
	}
	return suggester.Label()
}
