package server

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func reorderAppHAUSV441(t *testing.T) *app {
	t.Helper()
	a := newTestPortalApp(t, userProfile{
		Email:       "owner@example.com",
		Role:        roleOwner,
		Tenants:     []string{"jhw22"},
		AuthMethods: defaultAuthMethods(),
	})
	steps := []url.Values{
		{"action": {"profile"}, "household_name": {"Zuhause Test"}, "home_type": {"house"}},
		{"action": {"assets"}, "assets": {"pv", "ev", "heat-pump"}},
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

func consumerIDsHAUSV441(t *testing.T, a *app) []string {
	t.Helper()
	assets, err := a.energyStore.ListAssets("jhw22")
	if err != nil {
		t.Fatalf("Assets laden: %v", err)
	}
	ids := []string{}
	for _, asset := range sortEnergyConsumers(assets) {
		switch asset.Kind {
		case "pv", "battery":
			continue
		}
		ids = append(ids, asset.ID)
	}
	return ids
}

// HAUSV-441: die per Drag oder Pfeiltasten gewählte Reihenfolge der
// Verbraucher wird in den Asset-Metadaten festgehalten und bestimmt die
// Rail- und Flow-Config-Reihenfolge dauerhaft.
func TestConsumerReorderPersistsPrioritiesHAUSV441(t *testing.T) {
	a := reorderAppHAUSV441(t)
	ids := consumerIDsHAUSV441(t, a)
	if len(ids) < 2 {
		t.Fatalf("erwartet mindestens zwei Verbraucher, waren %v", ids)
	}

	reversed := append([]string(nil), ids...)
	for left, right := 0, len(reversed)-1; left < right; left, right = left+1, right-1 {
		reversed[left], reversed[right] = reversed[right], reversed[left]
	}
	form := url.Values{"order": reversed}
	if response := authedFormRequest(t, a, "owner@example.com", "/app/energie/verbraucher/reihenfolge", form); response.Code != http.StatusNoContent {
		t.Fatalf("Reihenfolge speichern: status=%d body=%s", response.Code, response.Body.String())
	}

	after := consumerIDsHAUSV441(t, a)
	if strings.Join(after, ",") != strings.Join(reversed, ",") {
		t.Fatalf("Reihenfolge nicht übernommen: erwartet %v, war %v", reversed, after)
	}

	// Die Flow-Config für den Renderer muss die neue Reihenfolge tragen.
	assets, err := a.energyStore.ListAssets("jhw22")
	if err != nil {
		t.Fatalf("Assets laden: %v", err)
	}
	cfg := buildEnergyFlowConfig("haus", energyLiveView{}, assets, nil, nil, parkingLiveView{}, true)
	got := []string{}
	for _, consumer := range cfg.Consumers {
		got = append(got, consumer.ID)
	}
	if strings.Join(got, ",") != strings.Join(reversed, ",") {
		t.Fatalf("Flow-Config trägt die neue Reihenfolge nicht: erwartet %v, war %v", reversed, got)
	}
}

// Unvollständige oder fremde Listen dürfen nichts umsortieren.
func TestConsumerReorderRejectsPartialOrForeignOrdersHAUSV441(t *testing.T) {
	a := reorderAppHAUSV441(t)
	ids := consumerIDsHAUSV441(t, a)

	if response := authedFormRequest(t, a, "owner@example.com", "/app/energie/verbraucher/reihenfolge",
		url.Values{"order": ids[:1]}); response.Code != http.StatusBadRequest {
		t.Fatalf("Teilliste muss abgelehnt werden: status=%d", response.Code)
	}
	foreign := append([]string(nil), ids...)
	foreign[0] = "asset-fremd"
	if response := authedFormRequest(t, a, "owner@example.com", "/app/energie/verbraucher/reihenfolge",
		url.Values{"order": foreign}); response.Code != http.StatusBadRequest {
		t.Fatalf("fremde ID muss abgelehnt werden: status=%d", response.Code)
	}
	if after := consumerIDsHAUSV441(t, a); strings.Join(after, ",") != strings.Join(ids, ",") {
		t.Fatalf("abgelehnte Eingaben dürfen die Reihenfolge nicht ändern: %v vs %v", ids, after)
	}
}

// Ohne Energie-Verwaltungsrecht bleibt die Reihenfolge unantastbar.
func TestConsumerReorderRequiresManageEnergyHAUSV441(t *testing.T) {
	a := reorderAppHAUSV441(t)
	ids := consumerIDsHAUSV441(t, a)
	a.profiles["resident@example.com"] = userProfile{
		Email: "resident@example.com", Role: roleResident,
		Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods(),
	}
	if response := authedFormRequest(t, a, "resident@example.com", "/app/energie/verbraucher/reihenfolge",
		url.Values{"order": ids}); response.Code != http.StatusForbidden {
		t.Fatalf("Bewohner ohne Recht: status=%d", response.Code)
	}
}
