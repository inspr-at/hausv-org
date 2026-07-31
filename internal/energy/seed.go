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
	TenantSlug    string      `json:"tenant_slug"`
	HouseholdName string      `json:"household_name"`
	HomeType      string      `json:"home_type"`
	Assets        []SeedAsset `json:"assets,omitempty"`
	Complete      bool        `json:"complete,omitempty"`
}

// SeedAsset akzeptiert beide Schreibweisen: die Kurzform "ev" für eine Vorlage
// und die Objektform für einen benannten Verbraucher mit eigenen Eigenschaften.
// Die Kurzform bleibt gültig, weil bestehende Pilotkonfigurationen sie nutzen.
type SeedAsset struct {
	Kind         string   `json:"kind"`
	Name         string   `json:"name,omitempty"`
	RatedPowerKW *float64 `json:"rated_power_kw,omitempty"`
	Flexibility  string   `json:"flexibility,omitempty"`
}

func (item *SeedAsset) UnmarshalJSON(data []byte) error {
	var kind string
	if err := json.Unmarshal(data, &kind); err == nil {
		*item = SeedAsset{Kind: kind}
		return nil
	}
	// Alias, damit der eigene Unmarshaler nicht rekursiv aufgerufen wird.
	type plain SeedAsset
	var parsed plain
	if err := json.Unmarshal(data, &parsed); err != nil {
		return fmt.Errorf("asset must be a kind string or an object")
	}
	*item = SeedAsset(parsed)
	return nil
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
		for _, item := range seed.Assets {
			kind := normalizeToken(item.Kind, "")
			if kind == "" {
				continue
			}
			asset := Asset{
				ID:          StableAssetID(slug, kind),
				TenantSlug:  slug,
				Kind:        kind,
				Name:        AssetKindLabel(kind),
				Flexibility: seedAssetFlexibility(kind),
				Source:      "profile-seed",
				Confirmed:   true,
			}
			if name := strings.TrimSpace(item.Name); name != "" {
				asset.Name = name
				// Eigene, aber weiterhin deterministische ID: mehrere benannte
				// Verbraucher derselben Art dürfen sich nicht überschreiben,
				// und ein Neustart darf sie nicht verdoppeln.
				asset.ID = StableAssetID(slug, kind) + "-" + normalizeSlug(name)
			}
			if item.RatedPowerKW != nil {
				value := *item.RatedPowerKW
				asset.RatedPowerKW = &value
			}
			if flexibility := strings.TrimSpace(item.Flexibility); flexibility != "" {
				asset.Flexibility = flexibility
			}
			if err := storage.UpsertAsset(asset); err != nil {
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
