package server

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/markus-barta/hausv-org/internal/energy"
)

func consumerAppHAUSV422(t *testing.T) *app {
	t.Helper()
	a := newTestPortalApp(t, userProfile{
		Email:       "owner@example.com",
		Role:        roleOwner,
		Tenants:     []string{"jhw22"},
		AuthMethods: defaultAuthMethods(),
	})
	steps := []url.Values{
		{"action": {"profile"}, "household_name": {"Zuhause Test"}, "home_type": {"house"}},
		{"action": {"assets"}, "assets": {"pv", "sauna"}},
		{"action": {"mappings"}},
		{"action": {"finish"}},
	}
	for i, form := range steps {
		if response := authedFormRequest(t, a, "owner@example.com", "/app/zuhause/onboarding", form); response.Code != http.StatusSeeOther {
			t.Fatalf("Onboarding-Schritt %d: status=%d", i+1, response.Code)
		}
	}
	return a
}

func addConsumerHAUSV422(t *testing.T, a *app, form url.Values) {
	t.Helper()
	if response := authedFormRequest(t, a, "owner@example.com", "/app/energie/verbraucher", form); response.Code != http.StatusSeeOther {
		t.Fatalf("Verbraucher anlegen: status=%d body=%s", response.Code, response.Body.String())
	}
}

// Der Kern von HAUSV-422: zwei Verbraucher derselben Art, beide benannt, beide
// dauerhaft. Vorher überschrieb der zweite den ersten, weil die ID aus
// (Haus, Art) abgeleitet wurde.
func TestTwoConsumersOfSameKindCoexistHAUSV422(t *testing.T) {
	a := consumerAppHAUSV422(t)
	addConsumerHAUSV422(t, a, url.Values{
		"name": {"Sauna Keller"}, "kind": {"sauna"}, "rated_power_kw": {"8"}, "flexibility": {"shift"},
	})
	addConsumerHAUSV422(t, a, url.Values{
		"name": {"Infrarotkabine"}, "kind": {"sauna"}, "rated_power_kw": {"2,5"}, "flexibility": {"shift"},
	})

	assets, err := a.energyStore.ListAssets("jhw22")
	if err != nil {
		t.Fatalf("Assets laden: %v", err)
	}
	names := map[string]bool{}
	saunas := 0
	for _, asset := range assets {
		names[asset.Name] = true
		if asset.Kind == "sauna" {
			saunas++
		}
	}
	if saunas != 3 { // Vorlage aus dem Onboarding plus zwei freie
		t.Fatalf("erwartet drei Assets der Art sauna, waren %d (%+v)", saunas, assets)
	}
	if !names["Sauna Keller"] || !names["Infrarotkabine"] {
		t.Fatalf("beide Namen müssen erhalten bleiben, waren %v", names)
	}
}

// Der destruktive Abgleich der Vorlagen darf freie Verbraucher nicht anfassen.
// Er löscht über die abgeleitete ID, die ein freier Verbraucher nie trägt —
// dieser Test hält genau das fest, weil ein Umbau dort Daten kosten würde.
func TestPresetResaveKeepsCustomConsumersHAUSV422(t *testing.T) {
	a := consumerAppHAUSV422(t)
	addConsumerHAUSV422(t, a, url.Values{
		"name": {"Werkstatt"}, "kind": {"other"}, "rated_power_kw": {"4"}, "flexibility": {"throttle"},
	})

	// Sauna-Vorlage abwählen: der Abgleich löscht sie, der freie Verbraucher bleibt.
	if response := authedFormRequest(t, a, "owner@example.com", "/app/zuhause/onboarding",
		url.Values{"action": {"assets"}, "assets": {"pv"}}); response.Code != http.StatusSeeOther {
		t.Fatalf("Vorlagen erneut speichern: status=%d", response.Code)
	}

	assets, _ := a.energyStore.ListAssets("jhw22")
	found := false
	for _, asset := range assets {
		if asset.Name == "Werkstatt" {
			found = true
		}
		if asset.Kind == "sauna" {
			t.Fatalf("abgewählte Vorlage sauna wurde nicht entfernt: %+v", asset)
		}
	}
	if !found {
		t.Fatalf("der freie Verbraucher wurde vom Vorlagen-Abgleich gelöscht: %+v", assets)
	}
}

// Ein freier Verbraucher muss im Lastmanagement ankommen — sonst ist er
// Dekoration. Das ist der eigentliche Zweck im Sinne von Peak Shaving.
func TestCustomConsumerCountsTowardsPeakShavingHAUSV422(t *testing.T) {
	a := consumerAppHAUSV422(t)
	addConsumerHAUSV422(t, a, url.Values{
		"name": {"Sauna Keller"}, "kind": {"sauna"}, "rated_power_kw": {"8"}, "flexibility": {"shift"},
	})
	assets, _ := a.energyStore.ListAssets("jhw22")

	views := buildEnergyScenarioViews(energy.HomeProfile{TenantSlug: "jhw22"}, assets, scenarioIntervals(18))
	if len(views) == 0 {
		t.Fatal("erwartet wurde ein Szenario")
	}
	if !strings.Contains(views[0].Assumptions, "Sauna Keller") {
		t.Fatalf("der freie Verbraucher fehlt in den Annahmen: %q", views[0].Assumptions)
	}
}

// Ohne Leistung oder ohne gesetzte Flexibilität darf nichts versprochen werden.
func TestConsumerWithoutFlexibilityPromisesNothingHAUSV422(t *testing.T) {
	a := consumerAppHAUSV422(t)
	addConsumerHAUSV422(t, a, url.Values{
		"name": {"Serverschrank"}, "kind": {"other"}, "rated_power_kw": {"1,2"}, "flexibility": {"unknown"},
	})
	assets, _ := a.energyStore.ListAssets("jhw22")

	for _, asset := range assets {
		if asset.Name == "Serverschrank" && asset.Flexibility != energy.FlexUnknown {
			t.Fatalf("offene Flexibilität wurde stillschweigend gesetzt: %q", asset.Flexibility)
		}
	}
	views := buildEnergyScenarioViews(energy.HomeProfile{TenantSlug: "jhw22"}, assets, scenarioIntervals(18))
	if len(views) > 0 && strings.Contains(views[0].Assumptions, "Serverschrank") {
		t.Fatal("ein Verbraucher ohne Flexibilität darf nicht als Peak-Wirkung erscheinen")
	}
}

// Vorlagen gehören dem Onboarding; über diesen Weg dürfen sie nicht
// verschwinden, sonst legt der nächste Abgleich sie ohnehin wieder an.
func TestPresetCannotBeDeletedAsConsumerHAUSV422(t *testing.T) {
	a := consumerAppHAUSV422(t)
	presetID := energy.StableAssetID("jhw22", "pv")

	response := authedFormRequest(t, a, "owner@example.com", "/app/energie/verbraucher/entfernen",
		url.Values{"asset_id": {presetID}})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("status=%d", response.Code)
	}
	assets, _ := a.energyStore.ListAssets("jhw22")
	for _, asset := range assets {
		if asset.ID == presetID {
			return
		}
	}
	t.Fatal("die Vorlage wurde über den Verbraucher-Pfad gelöscht")
}
