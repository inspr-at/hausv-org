package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/markus-barta/hausv-org/internal/energy"
	"github.com/markus-barta/hausv-org/internal/homeassistant"
)

func TestHomeOnboardingCompletesInObserveMode(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "owner@example.com",
		Role:        roleOwner,
		Tenants:     []string{"jhw22"},
		AuthMethods: defaultAuthMethods(),
	})

	page := authedRequest(t, a, "owner@example.com", "/app/zuhause/onboarding")
	if page.Code != http.StatusOK {
		t.Fatalf("GET onboarding status = %d", page.Code)
	}
	for _, want := range []string{"Nur beobachten", "Wir beginnen mit dem, was schon da ist."} {
		if !strings.Contains(page.Body.String(), want) {
			t.Fatalf("onboarding missing %q", want)
		}
	}

	steps := []url.Values{
		{"action": {"understand"}},
		{"action": {"profile"}, "household_name": {"Zuhause Test"}, "home_type": {"house"}},
		{"action": {"assets"}, "assets": {"pv", "ev", "heat-pump"}},
		{"action": {"mappings"}},
		{"action": {"finish"}},
	}
	for i, form := range steps {
		response := authedFormRequest(t, a, "owner@example.com", "/app/zuhause/onboarding", form)
		if response.Code != http.StatusSeeOther {
			t.Fatalf("step %d status = %d body=%s", i+1, response.Code, response.Body.String())
		}
	}
	profile, ok, err := a.energyStore.Profile("jhw22")
	if err != nil || !ok {
		t.Fatalf("Profile: ok=%v err=%v", ok, err)
	}
	if !profile.OnboardingComplete || profile.OperatingMode != energy.ModeObserve || profile.HouseholdName != "Zuhause Test" {
		t.Fatalf("profile = %+v", profile)
	}
	assets, err := a.energyStore.ListAssets("jhw22")
	if err != nil || len(assets) != 3 {
		t.Fatalf("assets = %+v err=%v", assets, err)
	}
	cockpit := authedRequest(t, a, "owner@example.com", "/app/energie")
	if cockpit.Code != http.StatusOK || !strings.Contains(cockpit.Body.String(), "drei Jahre") {
		t.Fatalf("cockpit pricing missing: status=%d", cockpit.Code)
	}
}

func TestEnergyModeRequiresOwnerAndExplicitConfirmation(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "resident@example.com",
		Role:        roleResident,
		Permissions: []string{permissionEnergyCaretaker},
		Tenants:     []string{"jhw22"},
		AuthMethods: defaultAuthMethods(),
	})
	profile := energy.DefaultProfile("jhw22", time.Now())
	profile.OnboardingComplete = true
	if err := a.energyStore.SaveProfile(profile); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}

	response := authedFormRequest(t, a, "resident@example.com", "/app/energie/mode", url.Values{
		"mode":              {"active"},
		"confirm":           {"yes"},
		"confirmation_text": {"AKTIVIEREN"},
	})
	if response.Code != http.StatusForbidden {
		t.Fatalf("caretaker active status = %d", response.Code)
	}

	a.profiles["owner@example.com"] = userProfile{
		Email:       "owner@example.com",
		Role:        roleOwner,
		Tenants:     []string{"jhw22"},
		AuthMethods: defaultAuthMethods(),
	}
	response = authedFormRequest(t, a, "owner@example.com", "/app/energie/mode", url.Values{
		"mode":              {"active"},
		"confirm":           {"yes"},
		"confirmation_text": {"wrong"},
	})
	if response.Code != http.StatusBadRequest {
		t.Fatalf("unconfirmed active status = %d", response.Code)
	}
	response = authedFormRequest(t, a, "owner@example.com", "/app/energie/mode", url.Values{
		"mode":              {"active"},
		"confirm":           {"yes"},
		"confirmation_text": {"AKTIVIEREN"},
	})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("confirmed active status = %d body=%s", response.Code, response.Body.String())
	}
	updated, _, _ := a.energyStore.Profile("jhw22")
	if updated.OperatingMode != energy.ModeActive {
		t.Fatalf("mode = %q", updated.OperatingMode)
	}
	if updated.AutomationStage != energy.StageShadow {
		t.Fatalf("automation stage = %q, want shadow", updated.AutomationStage)
	}
	response = authedFormRequest(t, a, "owner@example.com", "/app/energie/mode", url.Values{"mode": {"observe"}})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("return to observe status = %d", response.Code)
	}
	updated, _, _ = a.energyStore.Profile("jhw22")
	if updated.OperatingMode != energy.ModeObserve {
		t.Fatalf("mode after observe = %q", updated.OperatingMode)
	}
}

func TestOwnerCanGrantAndImmediatelyRevokeScopedEnergyAccess(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "owner@example.com",
		Role:        roleOwner,
		Tenants:     []string{"jhw22"},
		AuthMethods: defaultAuthMethods(),
	})
	added, err := a.inviteStore.Add(userProfile{
		Email:       "helper@example.com",
		FirstName:   "Technische",
		LastName:    "Hilfe",
		Role:        roleResident,
		Tenants:     []string{"jhw22"},
		AuthMethods: defaultAuthMethods(),
	})
	if err != nil || !added {
		t.Fatalf("Add helper: added=%v err=%v", added, err)
	}
	response := authedFormRequest(t, a, "owner@example.com", "/app/energie/caretaker", url.Values{
		"email": {"helper@example.com"},
		"scope": {"view", "configure", "control"},
	})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("grant status = %d body=%s", response.Code, response.Body.String())
	}
	helper := a.profileForTenant("helper@example.com", "jhw22")
	for _, permission := range []string{permissionEnergyView, permissionEnergyConfigure, permissionEnergyControl} {
		if !helper.HasPermission(permission) {
			t.Fatalf("helper missing %q: %+v", permission, helper.Permissions)
		}
	}
	profile := energy.DefaultProfile("jhw22", time.Now())
	profile.OnboardingComplete = true
	if err := a.energyStore.SaveProfile(profile); err != nil {
		t.Fatal(err)
	}
	response = authedFormRequest(t, a, "helper@example.com", "/app/energie/mode", url.Values{
		"mode":              {"active"},
		"confirm":           {"yes"},
		"confirmation_text": {"AKTIVIEREN"},
	})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("separately granted control status = %d", response.Code)
	}
	response = authedFormRequest(t, a, "owner@example.com", "/app/energie/caretaker", url.Values{
		"email": {"helper@example.com"},
	})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("revoke status = %d", response.Code)
	}
	helper = a.profileForTenant("helper@example.com", "jhw22")
	if helper.HasPermission(permissionEnergyView) || helper.HasPermission(permissionEnergyConfigure) || helper.HasPermission(permissionEnergyControl) {
		t.Fatalf("energy permissions not revoked: %+v", helper.Permissions)
	}
	response = authedFormRequest(t, a, "helper@example.com", "/app/energie/mode", url.Values{"mode": {"observe"}})
	if response.Code != http.StatusForbidden {
		t.Fatalf("revoked helper mode status = %d", response.Code)
	}
}

func TestEnergyCaretakerGrantRejectsForeignHouseMember(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "owner@example.com",
		Role:        roleOwner,
		Tenants:     []string{"jhw22"},
		AuthMethods: defaultAuthMethods(),
	})
	_, err := a.inviteStore.Add(userProfile{
		Email:       "foreign@example.com",
		Role:        roleResident,
		Tenants:     []string{"other-house"},
		AuthMethods: defaultAuthMethods(),
	})
	if err != nil {
		t.Fatal(err)
	}
	response := authedFormRequest(t, a, "owner@example.com", "/app/energie/caretaker", url.Values{
		"email": {"foreign@example.com"},
		"scope": {"control"},
	})
	if response.Code != http.StatusBadRequest {
		t.Fatalf("foreign grant status = %d", response.Code)
	}
}

func TestEnergyRecommendationCanBecomeDataSparseIssue(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "resident@example.com",
		FirstName:   "Max",
		LastName:    "Muster",
		Role:        roleResident,
		Tenants:     []string{"jhw22"},
		AuthMethods: defaultAuthMethods(),
	})
	profile := energy.DefaultProfile("jhw22", time.Now())
	profile.OnboardingComplete = true
	if err := a.energyStore.SaveProfile(profile); err != nil {
		t.Fatal(err)
	}
	response := authedFormRequest(t, a, "resident@example.com", "/app/energie/measure", url.Values{
		"recommendation_id": {"inventory"},
		"share":             {"inventory"},
	})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("measure status = %d body=%s", response.Code, response.Body.String())
	}
	issues := a.issueStore.ListTenant("jhw22")
	if len(issues) != 1 {
		t.Fatalf("issues = %+v", issues)
	}
	if !strings.Contains(issues[0].Body, "Bewusst freigegebene Daten: Anlageninventar") ||
		!strings.Contains(issues[0].Body, "Keine automatische Beauftragung") {
		t.Fatalf("measure body = %q", issues[0].Body)
	}
	if issues[0].AssigneeEmail != "" || issues[0].EstimateAmountCents != 0 {
		t.Fatalf("measure should not assign or price: %+v", issues[0])
	}
}

func TestSmartMeterImportEndpointIsIdempotentAndTenantScoped(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "owner@example.com",
		Role:        roleOwner,
		Tenants:     []string{"jhw22"},
		AuthMethods: defaultAuthMethods(),
	})
	profile := energy.DefaultProfile("jhw22", time.Now())
	profile.OnboardingComplete = true
	if err := a.energyStore.SaveProfile(profile); err != nil {
		t.Fatal(err)
	}
	csv := []byte("timestamp;import_kwh\n2026-07-01T00:00:00+02:00;0,42\n2026-07-01T00:15:00+02:00;0,38\n")
	for i := 0; i < 2; i++ {
		response := authedMultipartFileRequest(t, a, "owner@example.com", "/app/energie/smart-meter", nil, "smart_meter_file", "meter.csv", csv)
		if response.Code != http.StatusSeeOther {
			t.Fatalf("import %d status = %d body=%s", i, response.Code, response.Body.String())
		}
	}
	imports, err := a.energyStore.ListImports("jhw22")
	if err != nil || len(imports) != 1 {
		t.Fatalf("imports = %+v err=%v", imports, err)
	}
	intervals, err := a.energyStore.ListIntervals("jhw22", time.Time{}, time.Time{})
	if err != nil || len(intervals) != 2 {
		t.Fatalf("intervals = %+v err=%v", intervals, err)
	}
	foreign, err := a.energyStore.ListImports("other-house")
	if err != nil || len(foreign) != 0 {
		t.Fatalf("foreign imports = %+v err=%v", foreign, err)
	}
}

func TestEnergyDiscoveryOnlySuggestsMeasurementEntities(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/states" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"entity_id":"sensor.grid_power","state":"2.4","attributes":{"friendly_name":"Netzbezug","device_class":"power","unit_of_measurement":"kW"},"last_updated":"2026-07-28T10:00:00Z"},
			{"entity_id":"switch.wallbox","state":"off","attributes":{"friendly_name":"Wallbox"}},
			{"entity_id":"light.kitchen","state":"on","attributes":{"friendly_name":"Küche"}}
		]`))
	}))
	t.Cleanup(server.Close)
	cfg := homeassistant.NewConfig(server.URL, "fixture", "", "", "")
	candidates, err := discoverEnergyCandidates(t.Context(), cfg, nil)
	if err != nil {
		t.Fatalf("discoverEnergyCandidates: %v", err)
	}
	if len(candidates) != 1 || candidates[0].EntityID != "sensor.grid_power" {
		t.Fatalf("candidates = %+v", candidates)
	}
}

func TestManualEnergyMappingIsValidatedAndNotOverwrittenByDiscovery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"entity_id":"sensor.grid_power","state":"2.4","attributes":{"friendly_name":"Automatisch","device_class":"power","unit_of_measurement":"kW"},"last_updated":"2026-07-28T10:00:00Z"}
		]`))
	}))
	t.Cleanup(server.Close)
	a := newTestPortalApp(t, userProfile{
		Email:       "owner@example.com",
		Role:        roleOwner,
		Tenants:     []string{"jhw22"},
		AuthMethods: defaultAuthMethods(),
	})
	if err := a.saveManualEnergyMapping("jhw22", "sensor.grid_power", energy.MetricLoadPower, "Hausverbrauch korrigiert", "kW"); err != nil {
		t.Fatalf("saveManualEnergyMapping: %v", err)
	}
	tenant := a.tenants["jhw22"]
	tenant.HA = homeassistant.NewConfig(server.URL, "fixture", "", "", "")
	if err := a.saveSelectedEnergyMappings(t.Context(), tenant, []string{"sensor.grid_power"}); err != nil {
		t.Fatalf("saveSelectedEnergyMappings: %v", err)
	}
	mappings, err := a.energyStore.ListMappings("jhw22")
	if err != nil || len(mappings) != 1 {
		t.Fatalf("mappings = %+v err=%v", mappings, err)
	}
	if mappings[0].Metric != energy.MetricLoadPower || mappings[0].DisplayName != "Hausverbrauch korrigiert" {
		t.Fatalf("confirmed manual mapping overwritten: %+v", mappings[0])
	}
	if err := a.saveManualEnergyMapping("jhw22", "switch.wallbox", energy.MetricLoadPower, "Nope", "kW"); err == nil {
		t.Fatal("switch entity should be rejected")
	}
}
