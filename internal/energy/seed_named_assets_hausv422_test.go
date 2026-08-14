package energy_test

import (
	"math"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/energy"
)

func seedTenantsHAUSV422() map[string]struct{} {
	return map[string]struct{}{"haus": {}}
}

// Die Kurzform steht in bestehenden Pilotkonfigurationen und muss unverändert
// weiter funktionieren — sonst bricht ein Neustart die laufenden Piloten.
func TestSeedStillAcceptsShortFormHAUSV422(t *testing.T) {
	store := energy.NewMemoryStore()
	raw := `[{"tenant_slug":"haus","household_name":"Haus","home_type":"house","assets":["ev","wallbox"]}]`
	if err := energy.ApplyProfileSeeds(store, raw, seedTenantsHAUSV422(), time.Now()); err != nil {
		t.Fatalf("Seed: %v", err)
	}
	assets, _ := store.ListAssets("haus")
	if len(assets) != 2 {
		t.Fatalf("erwartet zwei Assets, waren %d", len(assets))
	}
	for _, asset := range assets {
		if asset.ID != energy.StableAssetID("haus", asset.Kind) {
			t.Fatalf("Kurzform muss die abgeleitete ID behalten, war %q", asset.ID)
		}
	}
}

// Die Objektform trägt Name, Leistung und Flexibilität — genau die
// Eigenschaften, aus denen das Lastmanagement rechnet.
func TestSeedAcceptsNamedConsumersHAUSV422(t *testing.T) {
	store := energy.NewMemoryStore()
	raw := `[{"tenant_slug":"haus","household_name":"Haus","home_type":"house","assets":[
		"pv",
		{"kind":"sauna","name":"Sauna Keller","rated_power_kw":8,"flexibility":"shift"},
		{"kind":"sauna","name":"Infrarotkabine","rated_power_kw":2.5,"flexibility":"shift"}
	]}]`
	if err := energy.ApplyProfileSeeds(store, raw, seedTenantsHAUSV422(), time.Now()); err != nil {
		t.Fatalf("Seed: %v", err)
	}
	assets, _ := store.ListAssets("haus")
	if len(assets) != 3 {
		t.Fatalf("erwartet drei Assets, waren %d (%+v)", len(assets), assets)
	}

	byName := map[string]energy.Asset{}
	for _, asset := range assets {
		byName[asset.Name] = asset
	}
	sauna, ok := byName["Sauna Keller"]
	if !ok {
		t.Fatalf("benannter Verbraucher fehlt: %+v", assets)
	}
	if sauna.RatedPowerKW == nil || math.Abs(*sauna.RatedPowerKW-8) > 0.001 {
		t.Fatalf("Leistung ging verloren: %+v", sauna.RatedPowerKW)
	}
	if sauna.Flexibility != energy.FlexShift {
		t.Fatalf("Flexibilität = %q, erwartet shift", sauna.Flexibility)
	}
	if _, ok := byName["Infrarotkabine"]; !ok {
		t.Fatal("zwei benannte Verbraucher derselben Art müssen nebeneinander bestehen")
	}
}

// Seeds sind über Neustarts idempotent; benannte Verbraucher dürfen sich dabei
// nicht verdoppeln.
func TestSeedNamedConsumersAreIdempotentHAUSV422(t *testing.T) {
	store := energy.NewMemoryStore()
	raw := `[{"tenant_slug":"haus","household_name":"Haus","home_type":"house","assets":[
		{"kind":"other","name":"Werkstatt","rated_power_kw":4,"flexibility":"throttle"}
	]}]`
	for i := 0; i < 3; i++ {
		if err := energy.ApplyProfileSeeds(store, raw, seedTenantsHAUSV422(), time.Now()); err != nil {
			t.Fatalf("Seed-Durchlauf %d: %v", i+1, err)
		}
	}
	assets, _ := store.ListAssets("haus")
	if len(assets) != 1 {
		t.Fatalf("mehrfaches Seeden hat Verbraucher verdoppelt: %d (%+v)", len(assets), assets)
	}
}

func TestSeedRejectsMalformedAssetEntryHAUSV422(t *testing.T) {
	store := energy.NewMemoryStore()
	raw := `[{"tenant_slug":"haus","household_name":"Haus","home_type":"house","assets":[42]}]`
	if err := energy.ApplyProfileSeeds(store, raw, seedTenantsHAUSV422(), time.Now()); err == nil {
		t.Fatal("eine unbrauchbare Asset-Angabe muss den Seed ablehnen, nicht stillschweigend übergehen")
	}
}
