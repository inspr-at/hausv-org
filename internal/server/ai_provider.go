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

// aiUnavailableLabel is the customer-facing text when an organisation's own
// provider cannot be read or built. The environment provider is not a
// substitute: that would send the organisation's data somewhere else.
const aiUnavailableLabel = "KI derzeit nicht verfügbar"

// triageFor resolves the configured suggester for an organisation. With no
// organisation override, the boot provider is the operator destination. A
// read, destination or construction failure stays closed: AI for that
// organisation is unavailable and the environment provider is not used.
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
		logError("organisation ai settings unavailable", fmt.Errorf("settings repository missing"), "organisation", orgKey)
		return nil, aiUnavailableLabel
	}
	settings, err := repo.Get(ctx)
	if err != nil {
		logError("organisation ai settings unavailable", err, "organisation", orgKey)
		return nil, aiUnavailableLabel
	}
	if !organisationAIOverridden(settings) {
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
	if strings.TrimSpace(settings.AIBaseURL) != "" {
		if err := validateAIBaseURL(settings.AIBaseURL); err != nil {
			logError("organisation ai destination rejected", err, "organisation", orgKey)
			return rememberUnavailableAI(a, orgKey, fingerprint)
		}
	}
	configured, buildErr := newSettingsAISuggester(aiSettingsGetenv(os.Getenv, settings))
	if buildErr != nil || configured == nil {
		if buildErr == nil {
			buildErr = fmt.Errorf("ai provider unavailable")
		}
		logError("organisation ai provider unavailable", buildErr, "organisation", orgKey)
		return rememberUnavailableAI(a, orgKey, fingerprint)
	}
	entry := triageProviderCacheEntry{fingerprint: fingerprint, suggester: configured, label: effectiveAIConfig(os.Getenv, settings).Label}
	a.triageProviders[orgKey] = entry
	return entry.suggester, entry.label
}

func organisationAIOverridden(settings store.OrgSettings) bool {
	return strings.TrimSpace(settings.AIProvider) != "" || strings.TrimSpace(settings.AIBaseURL) != "" || strings.TrimSpace(settings.AIModel) != ""
}

// rememberUnavailableAI caches a closed failure. The lock is held by triageFor.
func rememberUnavailableAI(a *app, orgKey, fingerprint string) (ai.TriageSuggester, string) {
	a.triageProviders[orgKey] = triageProviderCacheEntry{fingerprint: fingerprint, suggester: nil, label: aiUnavailableLabel}
	return nil, aiUnavailableLabel
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
