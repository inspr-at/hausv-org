package energy

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ProfileSeed is a non-secret, idempotent bootstrap description for a pilot
// home. It creates missing records only; completed onboarding and later user
// changes are never overwritten by process configuration.
type ProfileSeed struct {
	TenantSlug    string   `json:"tenant_slug"`
	HouseholdName string   `json:"household_name"`
	HomeType      string   `json:"home_type"`
	Assets        []string `json:"assets,omitempty"`
	Complete      bool     `json:"complete,omitempty"`
}

func ApplyProfileSeeds(storage Storage, raw string, knownTenants map[string]struct{}, now time.Time) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var seeds []ProfileSeed
	if err := json.Unmarshal([]byte(raw), &seeds); err != nil {
		return fmt.Errorf("invalid HOME_PROFILE_SEEDS_JSON")
	}
	seen := map[string]struct{}{}
	for _, seed := range seeds {
		slug := normalizeSlug(seed.TenantSlug)
		if slug == "" {
			return fmt.Errorf("home profile seed is missing tenant")
		}
		if _, ok := knownTenants[slug]; !ok {
			return fmt.Errorf("home profile seed references unknown tenant")
		}
		if _, duplicate := seen[slug]; duplicate {
			return fmt.Errorf("duplicate home profile seed for tenant %s", slug)
		}
		seen[slug] = struct{}{}
		if _, exists, err := storage.Profile(slug); err != nil {
			return err
		} else if exists {
			continue
		}
		profile := DefaultProfile(slug, now)
		profile.HouseholdName = strings.TrimSpace(seed.HouseholdName)
		profile.HomeType = seed.HomeType
		profile.OnboardingComplete = seed.Complete
		if seed.Complete {
			profile.OnboardingStep = 5
			started := now.UTC()
			profile.FreeStartedAt = &started
		}
		// The seed format intentionally has no operating-mode field. Bootstraps
		// cannot silently authorize control.
		profile.OperatingMode = ModeObserve
		profile.AutomationStage = StageObserve
		if err := storage.SaveProfile(profile); err != nil {
			return err
		}
		for _, rawKind := range seed.Assets {
			kind := normalizeToken(rawKind, "")
			if kind == "" {
				continue
			}
			if err := storage.UpsertAsset(Asset{
				ID:          "asset-" + kind,
				TenantSlug:  slug,
				Kind:        kind,
				Name:        AssetKindLabel(kind),
				Flexibility: seedAssetFlexibility(kind),
				Source:      "profile-seed",
				Confirmed:   true,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func seedAssetFlexibility(kind string) string {
	switch kind {
	case "ev", "wallbox", "hot-water":
		return FlexShift
	case "heat-pump", "battery", "air-conditioning":
		return FlexThrottle
	default:
		return FlexUnknown
	}
}
