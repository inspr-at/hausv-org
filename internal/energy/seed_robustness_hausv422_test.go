package energy

import (
	"math"
	"strings"
	"testing"
	"time"
)

// Seeding läuft genau einmal pro Zuhause. Was hier stillschweigend verloren
// geht, kommt nie wieder — deshalb muss jede unklare Angabe beim Start
// scheitern statt im Pilotbetrieb zu fehlen.

func applySeedHAUSV422(t *testing.T, raw string) error {
	t.Helper()
	store := NewMemoryStore()
	known := map[string]struct{}{"haus": {}}
	return ApplyProfileSeeds(store, raw, known, time.Now())
}

func TestSeedWithoutKindFailsLoudlyHAUSV422(t *testing.T) {
	err := applySeedHAUSV422(t, `[{"tenant_slug":"haus","assets":[{"name":"Sauna Keller","rated_power_kw":8}]}]`)
	if err == nil {
		t.Fatal("ein Verbraucher ohne Art muss den Start scheitern lassen, nicht verschwinden")
	}
	if !strings.Contains(err.Error(), "without kind") {
		t.Fatalf("die Meldung muss die Ursache benennen: %v", err)
	}
}

func TestSeedWithCollidingNamesFailsHAUSV422(t *testing.T) {
	err := applySeedHAUSV422(t, `[{"tenant_slug":"haus","assets":[
		{"kind":"sauna","name":"Sauna Keller"},
		{"kind":"sauna","name":"sauna-keller"}]}]`)
	if err == nil {
		t.Fatal("zwei Namen mit derselben ID würden sich überschreiben und müssen auffallen")
	}
}

func TestSeedWithUnusableNameFailsHAUSV422(t *testing.T) {
	err := applySeedHAUSV422(t, `[{"tenant_slug":"haus","assets":[{"kind":"sauna","name":"★"}]}]`)
	if err == nil {
		t.Fatal("ein Name ohne verwertbare Zeichen ergibt keine unterscheidbare ID")
	}
}

func TestSeedWithTwoNamedAssetsOfOneKindWorksHAUSV422(t *testing.T) {
	store := NewMemoryStore()
	if err := ApplyProfileSeeds(store, `[{"tenant_slug":"haus","assets":[
		{"kind":"sauna","name":"Sauna Keller","rated_power_kw":8,"flexibility":"shift"},
		{"kind":"sauna","name":"Infrarotkabine","rated_power_kw":2.5,"flexibility":"shift"}]}]`,
		map[string]struct{}{"haus": {}}, time.Now()); err != nil {
		t.Fatalf("gültiger Seed muss durchlaufen: %v", err)
	}
	assets, err := store.ListAssets("haus")
	if err != nil {
		t.Fatalf("Assets lesen: %v", err)
	}
	if len(assets) != 2 {
		t.Fatalf("erwartet wurden zwei eigenständige Verbraucher: %+v", assets)
	}
}

// NaN besteht jeden Vergleich: "value <= 0 || value > 1000" ist für NaN falsch.
// Ohne die ausdrückliche Prüfung stünde im Cockpit "NaN kW".
func TestNaNIsRejectedAsPowerHAUSV422(t *testing.T) {
	for _, value := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if ValidPowerKW(value, 1000) {
			t.Fatalf("%v darf keine gültige Leistung sein", value)
		}
		asset := NormalizeAsset(Asset{ID: "a1", Kind: "sauna", RatedPowerKW: &value}, time.Now())
		if asset.RatedPowerKW != nil {
			t.Fatalf("%v muss verworfen werden, gespeichert wurde %v", value, *asset.RatedPowerKW)
		}
		profile := NormalizeProfile(HomeProfile{TenantSlug: "haus", AgreedPowerKW: &value}, time.Now())
		if profile.AgreedPowerKW != nil {
			t.Fatalf("%v muss als Anschlussleistung verworfen werden", value)
		}
	}
}
