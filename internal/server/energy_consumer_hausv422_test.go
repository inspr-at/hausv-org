package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/markus-barta/hausv-org/internal/energy"
	"github.com/markus-barta/hausv-org/internal/homeassistant"
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

// Der Onboarding-Schritt wählt nur die Arten aus. Eine erfasste Nennleistung
// — etwa aus einem Pilot-Seed — darf er nicht mit der Vorbelegung überschreiben:
// aus 9 kW wurde sonst stillschweigend wieder 1 kW.
func TestPresetResaveKeepsRecordedPowerHAUSV422(t *testing.T) {
	a := consumerAppHAUSV422(t)
	rated := 9.0
	id := energy.StableAssetID("jhw22", "heat-pump")
	if err := a.energyStore.UpsertAsset(energy.Asset{
		ID: id, TenantSlug: "jhw22", Kind: "heat-pump", Name: "Wärmepumpe",
		RatedPowerKW: &rated, Flexibility: energy.FlexThrottle,
		Source: "profile-seed", Confirmed: true,
	}); err != nil {
		t.Fatalf("Seed-Asset speichern: %v", err)
	}

	if response := authedFormRequest(t, a, "owner@example.com", "/app/zuhause/onboarding",
		url.Values{"action": {"assets"}, "assets": {"pv", "sauna", "heat-pump"}}); response.Code != http.StatusSeeOther {
		t.Fatalf("Vorlagen erneut speichern: status=%d", response.Code)
	}

	assets, _ := a.energyStore.ListAssets("jhw22")
	for _, asset := range assets {
		if asset.ID != id {
			continue
		}
		if asset.RatedPowerKW == nil || *asset.RatedPowerKW != 9 {
			t.Fatalf("die erfasste Nennleistung ging verloren: %+v", asset)
		}
		if asset.Flexibility != energy.FlexThrottle {
			t.Fatalf("die erklärte Flexibilität ging verloren: %+v", asset)
		}
		return
	}
	t.Fatalf("die Vorlage wurde nicht gefunden: %+v", assets)
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

func TestPresetConsumerCanBeDeletedFromDialog(t *testing.T) {
	a := consumerAppHAUSV422(t)
	presetID := energy.StableAssetID("jhw22", "sauna")
	response := authedFormRequest(t, a, "owner@example.com", "/app/energie/verbraucher/entfernen",
		url.Values{"asset_id": {presetID}})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	assets, _ := a.energyStore.ListAssets("jhw22")
	for _, asset := range assets {
		if asset.ID == presetID {
			t.Fatal("Verbraucher-Vorlage wurde trotz bestätigtem Löschen behalten")
		}
	}
}

func TestConsumerMeasurementsUseDedicatedHomeAssistantSlots(t *testing.T) {
	ha := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		states := `[
			{"entity_id":"sensor.sauna_power","state":"7.2","attributes":{"friendly_name":"Sauna Leistung","device_class":"power","unit_of_measurement":"kW"}},
			{"entity_id":"sensor.sauna_energy","state":"42","attributes":{"friendly_name":"Sauna Energie","device_class":"energy","unit_of_measurement":"kWh"}},
			{"entity_id":"sensor.temperature","state":"22","attributes":{"friendly_name":"Temperatur","device_class":"temperature","unit_of_measurement":"°C"}}
		]`
		switch r.URL.Path {
		case "/api/states":
			_, _ = w.Write([]byte(states))
		case "/api/states/sensor.sauna_power":
			_, _ = w.Write([]byte(`{"entity_id":"sensor.sauna_power","state":"7.2","attributes":{"friendly_name":"Sauna Leistung","device_class":"power","unit_of_measurement":"kW"}}`))
		case "/api/states/sensor.sauna_energy":
			_, _ = w.Write([]byte(`{"entity_id":"sensor.sauna_energy","state":"42","attributes":{"friendly_name":"Sauna Energie","device_class":"energy","unit_of_measurement":"kWh"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(ha.Close)
	a := consumerAppHAUSV422(t)
	tenant := a.tenants["jhw22"]
	tenant.HA = homeassistant.NewConfig(ha.URL, "fixture", "", "", "")
	a.tenants["jhw22"] = tenant

	optionsResponse := authedRequest(t, a, "owner@example.com", "/app/energie/verbraucher/messwerte")
	if optionsResponse.Code != http.StatusOK {
		t.Fatalf("Messwertoptionen: status=%d body=%s", optionsResponse.Code, optionsResponse.Body.String())
	}
	var options energyConsumerMeasurementsResponse
	if err := json.Unmarshal(optionsResponse.Body.Bytes(), &options); err != nil {
		t.Fatalf("Messwertoptionen dekodieren: %v", err)
	}
	if options.Status != "ok" || len(options.Entities) != 2 {
		t.Fatalf("erwartet nur Leistungs-/Energie-Entities, war %+v", options)
	}

	assetID := energy.StableAssetID("jhw22", "sauna")
	response := authedFormRequest(t, a, "owner@example.com", "/app/energie/verbraucher", url.Values{
		"asset_id": {assetID}, "name": {"Sauna"}, "kind": {"sauna"}, "priority": {"1"},
		"icon": {"alarm-clock"}, "flexibility": {"shift"}, "measurements_present": {"1"},
		"consumer_power_entity": {"sensor.sauna_power"}, "consumer_energy_entity": {"sensor.sauna_energy"},
	})
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/app/energie?verbraucher=gespeichert" {
		t.Fatalf("Verbraucher speichern: status=%d location=%s", response.Code, response.Header().Get("Location"))
	}
	mappings, _ := a.energyStore.ListMappings("jhw22")
	seen := map[string]bool{}
	for _, mapping := range mappings {
		if mapping.AssetID != assetID || !mapping.Confirmed {
			continue
		}
		seen[mapping.Metric] = true
	}
	if !seen[energy.MetricConsumerPower] || !seen[energy.MetricConsumerEnergy] {
		t.Fatalf("beide Verbraucher-Messwerte müssen getrennt zugeordnet sein: %+v", mappings)
	}
	assets, _ := a.energyStore.ListAssets("jhw22")
	for _, asset := range assets {
		if asset.ID == assetID && asset.Metadata["icon"] != "alarm-clock" {
			t.Fatalf("Symbol aus vollständiger Lucide-Library nicht gespeichert: %+v", asset.Metadata)
		}
	}
	metrics, _, _ := a.currentEnergyMetrics(t.Context(), tenant, mappings, energy.HomeProfile{})
	cfg := buildEnergyFlowConfig("haus", energyLiveView{}, assets, mappings, metrics, parkingLiveView{}, true)
	for _, consumer := range cfg.Consumers {
		if consumer.ID == assetID {
			if consumer.KW != 7.2 || consumer.PowerEntity != "sensor.sauna_power" || consumer.EnergyEntity != "sensor.sauna_energy" {
				t.Fatalf("Verbraucher erhält nicht seine eigenen Live-Messwerte: %+v", consumer)
			}
			return
		}
	}
	t.Fatal("Sauna fehlt im Energiefluss")
}

func TestConsumerCannotStealWholeHomeMeasurement(t *testing.T) {
	a := consumerAppHAUSV422(t)
	assetID := energy.StableAssetID("jhw22", "sauna")
	if err := a.energyStore.UpsertMapping(energy.EntityMapping{
		TenantSlug: "jhw22", EntityID: "sensor.home_consumption", Metric: energy.MetricLoadPower,
		DisplayName: "Hausverbrauch", Unit: "kW", DeviceClass: "power", Confirmed: true,
	}); err != nil {
		t.Fatal(err)
	}
	response := authedFormRequest(t, a, "owner@example.com", "/app/energie/verbraucher", url.Values{
		"asset_id": {assetID}, "name": {"Sauna"}, "kind": {"sauna"}, "priority": {"1"},
		"icon": {"flame"}, "flexibility": {"shift"}, "measurements_present": {"1"},
		"consumer_power_entity": {"sensor.home_consumption"},
	})
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/app/energie?verbraucher=messwerte" {
		t.Fatalf("Hausmesswert wurde nicht geschützt: status=%d location=%s", response.Code, response.Header().Get("Location"))
	}
	mappings, _ := a.energyStore.ListMappings("jhw22")
	if len(mappings) != 1 || mappings[0].AssetID != "" || mappings[0].Metric != energy.MetricLoadPower {
		t.Fatalf("Hausmesswert wurde umgehängt: %+v", mappings)
	}
}

func TestConsumerCanBeEditedInPlaceWithLucideIconHAUSV446(t *testing.T) {
	a := consumerAppHAUSV422(t)
	assets, _ := a.energyStore.ListAssets("jhw22")
	var before energy.Asset
	for _, asset := range assets {
		if asset.Kind == "sauna" {
			before = asset
			break
		}
	}
	if before.ID == "" {
		t.Fatal("Sauna-Vorlage fehlt")
	}

	response := authedFormRequest(t, a, "owner@example.com", "/app/energie/verbraucher", url.Values{
		"asset_id": {before.ID}, "name": {"Werkstatt Sauna"}, "kind": {"sauna"},
		"priority": {"1"}, "icon": {"drill"}, "rated_power_kw": {"7,5"}, "flexibility": {"throttle"},
	})
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/app/energie?verbraucher=gespeichert" {
		t.Fatalf("Bearbeiten: status=%d location=%q body=%s", response.Code, response.Header().Get("Location"), response.Body.String())
	}

	assets, _ = a.energyStore.ListAssets("jhw22")
	for _, asset := range assets {
		if asset.ID != before.ID {
			continue
		}
		if asset.Name != "Werkstatt Sauna" || asset.Kind != "sauna" || asset.Source != before.Source {
			t.Fatalf("Identität oder Kerndaten gingen beim Bearbeiten verloren: vorher=%+v nachher=%+v", before, asset)
		}
		if asset.RatedPowerKW == nil || *asset.RatedPowerKW != 7.5 || asset.Flexibility != energy.FlexThrottle {
			t.Fatalf("technische Angaben wurden nicht gespeichert: %+v", asset)
		}
		if asset.Metadata["icon"] != "drill" || energyConsumerIcon(asset) != "drill" || asset.Metadata["priority"] != "1" {
			t.Fatalf("Lucide-Symbol oder Priorität wurde nicht gespeichert: %+v", asset.Metadata)
		}
		return
	}
	t.Fatal("bearbeiteter Verbraucher wurde nicht unter derselben ID gefunden")
}

func TestConsumerIconIsRestrictedToLocalLucideWhitelistHAUSV446(t *testing.T) {
	a := consumerAppHAUSV422(t)
	malicious := `https://example.invalid/icon.svg"><svg onload=alert(1)>`
	addConsumerHAUSV422(t, a, url.Values{
		"name": {"Nicht vertrauenswürdig"}, "kind": {"other"}, "priority": {"1"},
		"icon": {malicious}, "flexibility": {"unknown"},
	})

	assets, _ := a.energyStore.ListAssets("jhw22")
	for _, asset := range assets {
		if asset.Name != "Nicht vertrauenswürdig" {
			continue
		}
		if got := asset.Metadata["icon"]; got != "plug" {
			t.Fatalf("nicht freigegebenes Symbol wurde gespeichert: %q", got)
		}
		return
	}
	t.Fatal("angelegter Verbraucher fehlt")
}

func TestConsumerDialogReplacesDuplicateLowerManagementHAUSV446(t *testing.T) {
	a := consumerAppHAUSV422(t)
	body := authedRequest(t, a, "owner@example.com", "/app/energie").Body.String()
	for _, want := range []string{
		`id="energy-consumer-dialog"`, `data-consumer-dialog-title`, `Symbol auswählen`,
		`Symbole aus der lokal eingebundenen Lucide-Library.`, `name="priority"`,
		`name="icon_choice" value="car-front"`, `name="icon_choice" value="plug-zap"`,
		`data-consumer-measurement-fields`, `name="node_type" value="consumer"`,
		`data-consumer-delete`, `Ihr Energiesystem`, `Verbraucher verwalten Sie direkt oben`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("Verbraucher-Dialog fehlt %q", want)
		}
	}
	for _, obsolete := range []string{`id="anlagen"`, `Eigenen Verbraucher hinzufügen`, `class="energy-consumer-list"`} {
		if strings.Contains(body, obsolete) {
			t.Errorf("doppelte untere Verbraucherpflege ist noch vorhanden: %q", obsolete)
		}
	}
}

func TestParkingChargingUsesNormalEditableConsumerContract(t *testing.T) {
	a := consumerAppHAUSV422(t)
	assets, _ := a.energyStore.ListAssets("jhw22")
	charging := parkingLiveView{
		Available: true, Mode: "manual", ModeLabel: "Normalladen", PowerKW: 3.6,
		PowerEntity: "sensor.parking_power", EnergyEntity: "sensor.parking_energy",
	}
	cfg := buildEnergyFlowConfig("jhw22", energyLiveView{}, assets, nil, nil, charging, true)
	parkingID := energyFlowNodeID("jhw22", "parking")
	found := false
	for _, consumer := range cfg.Consumers {
		if consumer.ID != parkingID {
			continue
		}
		found = true
		if consumer.NodeType != "parking" || !consumer.Deletable || consumer.KW != 3.6 || len(consumer.Measurements) != 2 {
			t.Fatalf("Parkplatz verwendet nicht den normalen Verbraucher-Vertrag: %+v", consumer)
		}
	}
	if !found {
		t.Fatal("konfigurierter Parkplatz fehlt im Energiefluss")
	}
	response := authedFormRequest(t, a, "owner@example.com", "/app/energie/verbraucher/entfernen", url.Values{"asset_id": {parkingID}})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("Parkplatz löschen: status=%d", response.Code)
	}
	assets, _ = a.energyStore.ListAssets("jhw22")
	cfg = buildEnergyFlowConfig("jhw22", energyLiveView{}, assets, nil, nil, charging, true)
	for _, consumer := range cfg.Consumers {
		if consumer.ID == parkingID {
			t.Fatal("gelöschter Parkplatz bleibt im Energiefluss")
		}
	}
}

func TestStorageChargePowerIsExplicitlyConfigurableAndDisplayed(t *testing.T) {
	ha := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/states":
			_, _ = w.Write([]byte(`[
				{"entity_id":"sensor.storage_charge","state":"2400","attributes":{"friendly_name":"Speicher Ladeleistung","device_class":"power","unit_of_measurement":"W"}},
				{"entity_id":"sensor.storage_soc","state":"85","attributes":{"friendly_name":"Speicher Ladestand","device_class":"battery","unit_of_measurement":"%"}}
			]`))
		case "/api/states/sensor.storage_charge":
			_, _ = w.Write([]byte(`{"entity_id":"sensor.storage_charge","state":"2400","attributes":{"friendly_name":"Speicher Ladeleistung","device_class":"power","unit_of_measurement":"W"}}`))
		case "/api/states/sensor.storage_soc":
			_, _ = w.Write([]byte(`{"entity_id":"sensor.storage_soc","state":"85","attributes":{"friendly_name":"Speicher Ladestand","device_class":"battery","unit_of_measurement":"%"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(ha.Close)
	a := consumerAppHAUSV422(t)
	storageID := energy.StableAssetID("jhw22", "battery")
	if err := a.energyStore.UpsertAsset(energy.Asset{ID: storageID, TenantSlug: "jhw22", Kind: "battery", Name: "Speicher", Confirmed: true}); err != nil {
		t.Fatal(err)
	}
	tenant := a.tenants["jhw22"]
	tenant.HA = homeassistant.NewConfig(ha.URL, "fixture", "", "", "")
	a.tenants["jhw22"] = tenant
	response := authedFormRequest(t, a, "owner@example.com", "/app/energie/verbraucher", url.Values{
		"asset_id": {storageID}, "node_type": {"storage"}, "name": {"Hausspeicher"}, "icon": {"battery-charging"},
		"battery_charge_entity": {"sensor.storage_charge"}, "battery_soc_entity": {"sensor.storage_soc"},
	})
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/app/energie?verbraucher=gespeichert" {
		t.Fatalf("Speicher speichern: status=%d location=%s", response.Code, response.Header().Get("Location"))
	}
	mappings, _ := a.energyStore.ListMappings("jhw22")
	metrics, _, _ := a.currentEnergyMetrics(t.Context(), tenant, mappings, energy.HomeProfile{})
	live := buildEnergyLiveView(metrics)
	assets, _ := a.energyStore.ListAssets("jhw22")
	cfg := buildEnergyFlowConfig("jhw22", live, assets, mappings, metrics, parkingLiveView{}, true)
	if cfg.Storage == nil || cfg.Storage.Mode != "lädt" || cfg.Storage.Value != "2,4" || cfg.Storage.Label != "Hausspeicher" || cfg.Storage.Icon != "battery-charging" {
		t.Fatalf("konfigurierte Ladeleistung wird nicht angezeigt: %+v", cfg.Storage)
	}
	if len(cfg.Storage.Measurements) != 4 {
		t.Fatalf("Speicher braucht Netto-, Lade-, Entladeleistung und Ladestand: %+v", cfg.Storage.Measurements)
	}
}

func TestAllVisibleSystemFlowNodesExposeConfiguration(t *testing.T) {
	live := energyLiveView{
		HasMain: true, Main: energyMetricView{Numeric: 1200, Unit: "W"},
		HasGrid: true, Grid: energyMetricView{Metric: energy.MetricGridImportPower, Numeric: 300, Unit: "W"},
		HasBattery: true, Battery: energyMetricView{Numeric: 400, Unit: "W", Direction: "charging"},
		Flows: []energyMetricView{{Metric: energy.MetricPVPower, Numeric: 1500, Unit: "W"}},
	}
	cfg := buildEnergyFlowConfig("jhw22", live, nil, nil, nil, parkingLiveView{}, true)
	nodes := []*energyFlowNodeConfig{&cfg.Home, cfg.Grid, cfg.Storage}
	if len(cfg.Producers) != 1 {
		t.Fatalf("PV-Knoten fehlt: %+v", cfg.Producers)
	}
	nodes = append(nodes, &cfg.Producers[0])
	for _, node := range nodes {
		if node == nil || !node.Editable || node.ID == "" || node.NodeType == "" || len(node.Measurements) == 0 {
			t.Fatalf("sichtbarer Systemknoten ist noch statisch: %+v", node)
		}
	}
}
