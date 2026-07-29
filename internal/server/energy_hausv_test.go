package server

import (
	"fmt"
	"math"
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
	for _, want := range []string{"Nur beobachten", "Womit möchten Sie beginnen? Mit Ihrem Zuhause."} {
		if !strings.Contains(page.Body.String(), want) {
			t.Fatalf("onboarding missing %q", want)
		}
	}
	response := authedFormRequest(t, a, "owner@example.com", "/app/zuhause/onboarding", url.Values{"action": {"understand"}})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("understand status = %d", response.Code)
	}
	profilePage := authedRequest(t, a, "owner@example.com", "/app/zuhause/onboarding")
	for _, want := range []string{
		`data-home-type-select`,
		`aria-describedby="home-type-explanation"`,
		"Ein einzelner Haushalt in einem Mehrparteienhaus.",
		"Ein Haushalt mit eigenem Gebäude.",
		"Mehrere Parteien und gemeinsam genutzte Anlagen.",
		"Rechte und „Nur beobachten“ bleiben unverändert.",
	} {
		if !strings.Contains(profilePage.Body.String(), want) {
			t.Fatalf("profile step missing home-type guidance %q", want)
		}
	}

	steps := []url.Values{
		{"action": {"profile"}, "household_name": {"Zuhause Test"}, "home_type": {"house"}},
		{"action": {"assets"}, "assets": {"pv", "ev", "heat-pump"}},
		{"action": {"mappings"}},
		{"action": {"finish"}},
	}
	for i, form := range steps {
		response := authedFormRequest(t, a, "owner@example.com", "/app/zuhause/onboarding", form)
		if response.Code != http.StatusSeeOther {
			t.Fatalf("step %d status = %d body=%s", i+2, response.Code, response.Body.String())
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
		Permissions: []string{permissionEnergyCaretaker, permissionEnergyControl},
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
		t.Fatalf("caretaker with legacy energy-control active status = %d", response.Code)
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

func TestOwnerCanGrantAndImmediatelyRevokeScopedEnergyAccessWithoutDelegatingHouseMode(t *testing.T) {
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
	for _, permission := range []string{permissionEnergyView, permissionEnergyConfigure} {
		if !helper.HasPermission(permission) {
			t.Fatalf("helper missing %q: %+v", permission, helper.Permissions)
		}
	}
	if helper.HasPermission(permissionEnergyControl) {
		t.Fatalf("technical helper received forbidden house-mode permission: %+v", helper.Permissions)
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
	if response.Code != http.StatusForbidden {
		t.Fatalf("technical helper active status = %d", response.Code)
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

func TestOwnerCanInviteEnergyCaretakerButResidentCannot(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email: "owner@example.com", Role: roleOwner, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods(),
	})
	mailer := &recordingMailer{}
	a.mailer = mailer
	response := authedFormRequest(t, a, "owner@example.com", "/app/energie/caretaker/invite", url.Values{
		"email": {"helper@example.com"}, "first_name": {"Technische"}, "last_name": {"Hilfe"},
		"scope": {"configure"},
	})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("invite status = %d body=%s", response.Code, response.Body.String())
	}
	helper, ok := a.inviteStore.Get("helper@example.com")
	if !ok || !helper.HasTenant("jhw22") || !helper.HasPermission(permissionEnergyView) ||
		!helper.HasPermission(permissionEnergyConfigure) || helper.HasPermission(permissionEnergyControl) {
		t.Fatalf("invited helper = %+v ok=%v", helper, ok)
	}
	if len(mailer.invites) != 1 || mailer.invites[0] != "helper@example.com" {
		t.Fatalf("invite mails = %+v", mailer.invites)
	}
	if !a.auditStore.HasTarget("jhw22", "energy.caretaker.invite", "helper@example.com") {
		t.Fatal("caretaker invitation missing audit evidence")
	}
	if _, err := a.inviteStore.Add(userProfile{
		Email: "known@example.com", Role: roleResident, Tenants: []string{"other-house"}, AuthMethods: defaultAuthMethods(),
	}); err != nil {
		t.Fatal(err)
	}
	response = authedFormRequest(t, a, "owner@example.com", "/app/energie/caretaker/invite", url.Values{
		"email": {"known@example.com"}, "scope": {"control"},
	})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("existing identity invite status = %d body=%s", response.Code, response.Body.String())
	}
	known, ok := a.inviteStore.Get("known@example.com")
	if !ok || !known.HasTenant("other-house") || !known.HasTenant("jhw22") ||
		!known.ForTenant("jhw22").HasPermission(permissionEnergyView) ||
		known.ForTenant("jhw22").HasPermission(permissionEnergyControl) {
		t.Fatalf("multi-house caretaker = %+v ok=%v", known, ok)
	}

	a.profiles["resident@example.com"] = userProfile{
		Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods(),
	}
	response = authedFormRequest(t, a, "resident@example.com", "/app/energie/caretaker/invite", url.Values{
		"email": {"attacker@example.com"},
	})
	if response.Code != http.StatusForbidden {
		t.Fatalf("resident invite status = %d", response.Code)
	}
	if _, ok := a.inviteStore.Get("attacker@example.com"); ok {
		t.Fatal("resident created caretaker invitation")
	}
}

func TestEnergyActionHandlersDoNotCrossTenantBoundary(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email: "owner@example.com", Role: roleOwner, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods(),
	})
	if err := a.energyStore.SaveProfile(energy.DefaultProfile("other-house", time.Now())); err != nil {
		t.Fatal(err)
	}
	foreignAsset := energy.StableAssetID("other-house", "pv")
	if err := a.energyStore.UpsertAsset(energy.Asset{
		ID: foreignAsset, TenantSlug: "other-house", Kind: "pv", Name: "Fremde PV", Confirmed: true,
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.energyStore.UpsertMaintenance(energy.MaintenancePlan{
		ID: "foreign-maintenance", TenantSlug: "other-house", AssetID: foreignAsset,
		Title: "Fremde Wartung", IntervalMonths: 12, NextDueAt: time.Now(), Active: true,
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.energyStore.UpsertMeasure(energy.Measure{
		ID: "foreign-measure", TenantSlug: "other-house", IssueID: "foreign-issue",
		RecommendationID: "measure", Title: "Fremde Maßnahme", Status: energy.MeasureRequested,
	}); err != nil {
		t.Fatal(err)
	}
	response := authedFormRequest(t, a, "owner@example.com", "/app/energie/maintenance/complete", url.Values{
		"id": {"foreign-maintenance"}, "completed_at": {time.Now().Format("2006-01-02")}, "evidence_note": {"Nope"},
	})
	if response.Code != http.StatusNotFound {
		t.Fatalf("foreign maintenance status = %d", response.Code)
	}
	response = authedFormRequest(t, a, "owner@example.com", "/app/energie/measure/update", url.Values{
		"id": {"foreign-measure"}, "status": {energy.MeasureCancelled},
	})
	if response.Code != http.StatusNotFound {
		t.Fatalf("foreign measure status = %d", response.Code)
	}
	foreignPlans, _ := a.energyStore.ListMaintenance("other-house")
	foreignMeasure, ok, _ := a.energyStore.GetMeasure("other-house", "foreign-measure")
	if len(foreignPlans) != 1 || foreignPlans[0].LastCompletedAt != nil || !ok || foreignMeasure.Status != energy.MeasureRequested {
		t.Fatalf("foreign state changed: plans=%+v measure=%+v ok=%v", foreignPlans, foreignMeasure, ok)
	}
}

func TestMaintenanceIsTenantScopedRecurringAndBecomesNextStep(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email: "owner@example.com", Role: roleOwner, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods(),
	})
	profile := energy.DefaultProfile("jhw22", time.Now())
	profile.OnboardingComplete = true
	if err := a.energyStore.SaveProfile(profile); err != nil {
		t.Fatal(err)
	}
	assetID := energy.StableAssetID("jhw22", "heat-pump")
	if err := a.energyStore.UpsertAsset(energy.Asset{
		ID: assetID, TenantSlug: "jhw22", Kind: "heat-pump", Name: "Wärmepumpe", Confirmed: true,
	}); err != nil {
		t.Fatal(err)
	}
	due := time.Now().Add(-24 * time.Hour).Format("2006-01-02")
	response := authedFormRequest(t, a, "owner@example.com", "/app/energie/maintenance", url.Values{
		"asset_id": {assetID}, "title": {"Wärmepumpe warten"}, "interval_months": {"12"},
		"next_due": {due}, "active": {"true"},
	})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("maintenance save status = %d body=%s", response.Code, response.Body.String())
	}
	plans, err := a.energyStore.ListMaintenance("jhw22")
	if err != nil || len(plans) != 1 || plans[0].ID == "" {
		t.Fatalf("plans = %+v err=%v", plans, err)
	}
	page := authedRequest(t, a, "owner@example.com", "/app/energie")
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "Wärmepumpe warten") ||
		!strings.Contains(page.Body.String(), "Jetzt fällig") {
		t.Fatalf("maintenance next step missing: status=%d", page.Code)
	}
	response = authedFormRequest(t, a, "owner@example.com", "/app/energie/measure", url.Values{
		"recommendation_id": {"maintenance-" + plans[0].ID}, "share": {"inventory"},
	})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("maintenance measure status = %d body=%s", response.Code, response.Body.String())
	}
	if measures, err := a.energyStore.ListMeasures("jhw22"); err != nil || len(measures) != 1 ||
		measures[0].RecommendationID != "maintenance-"+plans[0].ID {
		t.Fatalf("maintenance measures = %+v err=%v", measures, err)
	}
	completed := time.Now().Format("2006-01-02")
	response = authedFormRequest(t, a, "owner@example.com", "/app/energie/maintenance/complete", url.Values{
		"id": {plans[0].ID}, "completed_at": {completed}, "evidence_note": {"Servicebericht geprüft"},
	})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("maintenance complete status = %d body=%s", response.Code, response.Body.String())
	}
	updated, ok := findMaintenancePlan(a.energyStore, "jhw22", plans[0].ID)
	if !ok || updated.LastCompletedAt == nil || updated.NextDueAt.Before(time.Now().AddDate(0, 11, 0)) {
		t.Fatalf("updated maintenance = %+v ok=%v", updated, ok)
	}

	a.profiles["resident@example.com"] = userProfile{
		Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods(),
	}
	response = authedFormRequest(t, a, "resident@example.com", "/app/energie/maintenance", url.Values{
		"asset_id": {assetID}, "interval_months": {"1"}, "next_due": {due},
	})
	if response.Code != http.StatusForbidden {
		t.Fatalf("resident maintenance status = %d", response.Code)
	}
}

func TestTariffAssessmentKeepsImmutableRuleSnapshot(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email: "owner@example.com", Role: roleOwner, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods(),
	})
	profile := energy.DefaultProfile("jhw22", time.Now())
	profile.OnboardingComplete = true
	if err := a.energyStore.SaveProfile(profile); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := a.energyStore.PutInterval(energy.Interval{
		TenantSlug: "jhw22", StartsAt: now, Duration: 15 * time.Minute,
		AverageKW: 8.75, ImportKWh: 2.1875, Quality: energy.QualityMeasured, Source: "smart-meter",
	}); err != nil {
		t.Fatal(err)
	}
	response := authedFormRequest(t, a, "owner@example.com", "/app/energie/tariff/assessment", nil)
	if response.Code != http.StatusSeeOther {
		t.Fatalf("assessment status = %d body=%s", response.Code, response.Body.String())
	}
	items, err := a.energyStore.ListTariffAssessments("jhw22")
	if err != nil || len(items) != 1 || items[0].ID == "" || items[0].ProfileID == "" ||
		items[0].ProfileVersion == "" || items[0].PeakKW != 8.75 {
		t.Fatalf("assessments = %+v err=%v", items, err)
	}
	if !a.auditStore.HasTarget("jhw22", "energy.tariff.assessment", items[0].ID) {
		t.Fatal("tariff snapshot missing audit evidence")
	}
	page := authedRequest(t, a, "owner@example.com", "/app/energie")
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), items[0].ProfileVersion) ||
		!strings.Contains(page.Body.String(), "Festgehaltene Bewertungen") {
		t.Fatalf("tariff history missing: status=%d", page.Code)
	}
}

func TestCuratedEnergySpecialistAndMeasureStayClosedUntilExplicitPortalGate(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email: "owner@example.com", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods(),
	})
	if a.serviceAccessEnabled {
		t.Fatal("service portal gate must default closed")
	}
	profile := energy.DefaultProfile("jhw22", time.Now())
	profile.OnboardingComplete = true
	if err := a.energyStore.SaveProfile(profile); err != nil {
		t.Fatal(err)
	}
	mailer := &recordingMailer{}
	a.mailer = mailer
	response := authedFormRequest(t, a, "owner@example.com", "/app/kontakte", url.Values{
		"kind": {"Energie-Fachbetrieb"}, "name": {"Energiehilfe Graz"}, "email": {"energie@example.com"},
		"service_region": {"Graz und Umgebung"}, "qualification": {"Elektrotechnik"},
		"energy_capabilities": {"metering", "home-assistant"}, "active": {"true"},
	})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("energy contact status = %d body=%s", response.Code, response.Body.String())
	}
	contacts := a.contactStore.ListTenant("jhw22", false)
	if len(contacts) != 1 || contacts[0].Kind != "Energie-Fachbetrieb" ||
		len(contacts[0].EnergyCapabilities) != 2 {
		t.Fatalf("contacts = %+v", contacts)
	}
	if len(mailer.invites) != 0 {
		t.Fatalf("curated contact unexpectedly received portal invite: %+v", mailer.invites)
	}

	response = authedFormRequest(t, a, "owner@example.com", "/app/energie/measure", url.Values{
		"recommendation_id": {"inventory"}, "share": {"inventory", "measurements"},
	})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("measure create status = %d body=%s", response.Code, response.Body.String())
	}
	measures, err := a.energyStore.ListMeasures("jhw22")
	if err != nil || len(measures) != 1 {
		t.Fatalf("measures = %+v err=%v", measures, err)
	}
	response = authedFormRequest(t, a, "owner@example.com", "/app/energie/measure/update", url.Values{
		"id": {measures[0].ID}, "status": {energy.MeasureRequested}, "contact_id": {contacts[0].ID},
		"offer_note": {"Messkonzept angefragt"},
	})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("measure assign status = %d body=%s", response.Code, response.Body.String())
	}
	measure, ok, err := a.energyStore.GetMeasure("jhw22", measures[0].ID)
	if err != nil || !ok || measure.Status != energy.MeasureAssigned || measure.ContactID != contacts[0].ID {
		t.Fatalf("assigned measure = %+v ok=%v err=%v", measure, ok, err)
	}
	issue, ok := a.issueStore.Get("jhw22", measure.IssueID)
	if !ok || issue.AssigneeEmail != "" {
		t.Fatalf("closed gate granted issue access: %+v ok=%v", issue, ok)
	}

	before := time.Date(2026, 6, 15, 12, 0, 0, 0, time.Local)
	after := time.Date(2026, 7, 15, 12, 0, 0, 0, time.Local)
	for _, interval := range []energy.Interval{
		{TenantSlug: "jhw22", StartsAt: before, Duration: 15 * time.Minute, AverageKW: 9.2, Quality: energy.QualityMeasured, Source: "smart-meter"},
		{TenantSlug: "jhw22", StartsAt: after, Duration: 15 * time.Minute, AverageKW: 6.4, Quality: energy.QualityMeasured, Source: "smart-meter"},
	} {
		if err := a.energyStore.PutInterval(interval); err != nil {
			t.Fatal(err)
		}
	}
	response = authedFormRequest(t, a, "owner@example.com", "/app/energie/measure/update", url.Values{
		"id": {measure.ID}, "status": {energy.MeasureCompleted}, "contact_id": {contacts[0].ID},
		"offer_note": {"Angebot angenommen"}, "work_note": {"Lastmanagement eingerichtet"},
		"evidence_note": {"Smart-Meter-Zeiträume geprüft"},
		"before_from":   {"2026-06-01"}, "before_to": {"2026-06-30"},
		"after_from": {"2026-07-01"}, "after_to": {"2026-07-31"},
	})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("measure complete status = %d body=%s", response.Code, response.Body.String())
	}
	measure, ok, err = a.energyStore.GetMeasure("jhw22", measure.ID)
	if err != nil || !ok || measure.Status != energy.MeasureCompleted || measure.BeforePeakKW == nil ||
		measure.AfterPeakKW == nil || *measure.BeforePeakKW != 9.2 || *measure.AfterPeakKW != 6.4 {
		t.Fatalf("completed measure = %+v ok=%v err=%v", measure, ok, err)
	}
	if len(mailer.invites) != 0 {
		t.Fatalf("closed gate caused portal invitation: %+v", mailer.invites)
	}
	page := authedRequest(t, a, "owner@example.com", "/app/energie")
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "9,2 kW") ||
		!strings.Contains(page.Body.String(), "6,4 kW") || !strings.Contains(page.Body.String(), "Energiehilfe Graz") {
		t.Fatalf("measure context missing: status=%d", page.Code)
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
			{"entity_id":"sensor.grid_import_power","state":"2.4","attributes":{"friendly_name":"Netzbezug","device_class":"power","unit_of_measurement":"kW"},"last_updated":"2026-07-28T10:00:00Z"},
			{"entity_id":"sensor.grid_export_power","state":"0.4","attributes":{"friendly_name":"Netzeinspeisung","device_class":"power","unit_of_measurement":"kW"},"last_updated":"2026-07-28T10:00:00Z"},
			{"entity_id":"sensor.grid_import_energy","state":"42","attributes":{"friendly_name":"Netzbezug Energie","device_class":"energy","unit_of_measurement":"kWh"},"last_updated":"2026-07-28T10:00:00Z"},
			{"entity_id":"sensor.solaredge_keller_current_power","state":"3.1","attributes":{"friendly_name":"SolarEdge Keller Current Power","device_class":"power","unit_of_measurement":"kW"},"last_updated":"2026-07-28T10:00:00Z"},
			{"entity_id":"sensor.sonnenbatterie_state_battery_percentage_user","state":"78","attributes":{"friendly_name":"Sonnenbatterie State Battery Percentage User","device_class":"battery","unit_of_measurement":"%"},"last_updated":"2026-07-28T10:00:00Z"},
			{"entity_id":"sensor.sonnenbatterie_state_consumption_current","state":"1.8","attributes":{"friendly_name":"Sonnenbatterie State Consumption Current","device_class":"power","unit_of_measurement":"kW"},"last_updated":"2026-07-28T10:00:00Z"},
			{"entity_id":"sensor.battery_charge_power","state":"0.8","attributes":{"friendly_name":"Battery Charge Power","device_class":"power","unit_of_measurement":"kW"},"last_updated":"2026-07-28T10:00:00Z"},
			{"entity_id":"sensor.iphone_battery","state":"81","attributes":{"friendly_name":"iPhone Battery","device_class":"battery","unit_of_measurement":"%"},"last_updated":"2026-07-28T10:00:00Z"},
			{"entity_id":"sensor.pv_forecast_power","state":"4.4","attributes":{"friendly_name":"PV Forecast Power","device_class":"power","unit_of_measurement":"kW"},"last_updated":"2026-07-28T10:00:00Z"},
			{"entity_id":"sensor.kettle_power","state":"1.9","attributes":{"friendly_name":"Wasserkocher Leistung","device_class":"power","unit_of_measurement":"kW"},"last_updated":"2026-07-28T10:00:00Z"},
			{"entity_id":"number.battery_force_charge","state":"0","attributes":{"friendly_name":"Battery Force Charge","device_class":"battery","unit_of_measurement":"%"},"last_updated":"2026-07-28T10:00:00Z"},
			{"entity_id":"binary_sensor.lock_battery","state":"off","attributes":{"friendly_name":"Nuki Battery","device_class":"battery","unit_of_measurement":"%"},"last_updated":"2026-07-28T10:00:00Z"},
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
	if len(candidates.Recommended) != 5 {
		t.Fatalf("recommended = %+v", candidates.Recommended)
	}
	if len(candidates.Additional) != 2 {
		t.Fatalf("additional = %+v", candidates.Additional)
	}
	all := append(append([]energyCandidateView{}, candidates.Recommended...), candidates.Additional...)
	for _, candidate := range all {
		if strings.Contains(candidate.EntityID, "iphone") ||
			strings.Contains(candidate.EntityID, "forecast") ||
			strings.Contains(candidate.EntityID, "kettle") ||
			!strings.HasPrefix(candidate.EntityID, "sensor.") {
			t.Fatalf("unsafe discovery candidate = %+v", candidate)
		}
	}
	for _, candidate := range candidates.Recommended {
		if !candidate.Checked {
			t.Fatalf("recommended candidate is not selected by default: %+v", candidate)
		}
	}
}

func TestEnergyLiveViewCondensesManyReadingsIntoHouseFlow(t *testing.T) {
	metrics := []energyMetricView{
		{Metric: energy.MetricBatteryPower, Kind: "battery-charge", Label: "Batterieleistung", Value: "0 W", Detail: "Battery Charge Power", Numeric: 0, Unit: "W"},
		{Metric: energy.MetricBatteryPower, Kind: "battery-discharge", Label: "Batterieleistung", Value: "701 W", Detail: "Battery Discharge Power", Numeric: 701, Unit: "W"},
		{Metric: energy.MetricLoadPower, Kind: energy.MetricLoadPower, Label: "Hausverbrauch", Value: "7,19 kW", Detail: "Home Current Consumption", Numeric: 7192, Unit: "W"},
		{Metric: energy.MetricBatterySOC, Kind: energy.MetricBatterySOC, Label: "Batteriestand", Value: "50 %", Detail: "Sonnenbatterie Ladestand", Numeric: 50, Unit: "%"},
		{Metric: energy.MetricGridExportPower, Kind: energy.MetricGridExportPower, Label: "Einspeisung", Value: "0 W", Detail: "Grid Export Power", Numeric: 0, Unit: "W"},
		{Metric: energy.MetricGridImportEnergy, Kind: energy.MetricGridImportEnergy, Label: "Netzbezug gesamt", Value: "5.407 kWh", Detail: "Grid Import Energy", Numeric: 5407, Unit: "kWh"},
		{Metric: energy.MetricGridImportPower, Kind: energy.MetricGridImportPower, Label: "Netzbezug jetzt", Value: "50 W", Detail: "Grid Import Power", Numeric: 50, Unit: "W"},
		{Metric: energy.MetricPVPower, Kind: energy.MetricPVPower, Label: "PV-Leistung", Value: "6,35 kW", Detail: "SolarEdge Current Power", Numeric: 6345, Unit: "W"},
	}

	view := buildEnergyLiveView(metrics)
	if !view.HasMain || view.Main.Label != "Hausverbrauch" || view.Main.Value != "7,19 kW" {
		t.Fatalf("main = %+v", view.Main)
	}
	if !view.HasBattery || view.Battery.Value != "701 W" || view.Battery.Detail != "liefert Energie" {
		t.Fatalf("battery = %+v", view.Battery)
	}
	if !view.HasBatterySOC || view.BatterySOC.Value != "50 %" || view.BatteryFill != "50.0" {
		t.Fatalf("battery SOC = %+v fill=%q", view.BatterySOC, view.BatteryFill)
	}
	if len(view.Flows) != 3 {
		t.Fatalf("flows = %+v", view.Flows)
	}
	if !view.HasAdditional || view.AdditionalCount != 1 ||
		view.Additional[0].Metric != energy.MetricGridImportEnergy ||
		view.Additional[0].Detail != "" {
		t.Fatalf("additional = %+v", view.Additional)
	}
}

func TestEnergyReadingUsesHumanScaleAndUnitAwarePeakTone(t *testing.T) {
	for _, test := range []struct {
		value float64
		unit  string
		want  string
	}{
		{7192, "W", "7,19 kW"},
		{701, "W", "701 W"},
		{0.05, "kW", "0,05 kW"},
		{50, "%", "50 %"},
	} {
		if got := formatEnergyReading(test.value, test.unit); got != test.want {
			t.Fatalf("formatEnergyReading(%v, %q) = %q, want %q", test.value, test.unit, got, test.want)
		}
	}
	target := 8.0
	if got := energyMetricTone(energy.MetricGridImportPower, 7000, "W", &target); got != "warning" {
		t.Fatalf("unit-aware tone = %q", got)
	}
	if got := energyMetricTone(energy.MetricGridImportPower, 50, "W", &target); got != "" {
		t.Fatalf("50 W tone = %q", got)
	}
}

func TestEnergyHistoryBucketsForwardFillAndConvertPower(t *testing.T) {
	start := time.Date(2026, 7, 29, 8, 0, 0, 0, time.UTC)
	items := []homeassistant.HistoryState{
		{EntityID: "sensor.house", State: "1000", LastChanged: start},
		{EntityID: "sensor.house", State: "2400", LastChanged: start.Add(30 * time.Minute)},
	}
	values, present := energyHistoryBuckets(items, "W", start, start.Add(time.Hour), 5)
	want := []float64{1, 1, 2.4, 2.4, 2.4}
	for index := range want {
		if !present[index] || math.Abs(values[index]-want[index]) > 0.0001 {
			t.Fatalf("bucket %d = %v present=%v, want %v", index, values[index], present[index], want[index])
		}
	}
}

func TestEnergyChartSummarisesPeakWithoutPromise(t *testing.T) {
	start := time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)
	load := make([]float64, 97)
	pv := make([]float64, 97)
	grid := make([]float64, 97)
	battery := make([]float64, 97)
	present := make([]bool, 97)
	for index := range present {
		present[index] = true
		load[index] = 1
	}
	load[48] = 6
	pv[48] = 4
	battery[48] = 1
	summary, detail := energyChartSummary(start, load, present, pv, present, grid, present, battery, present, time.UTC)
	if !strings.Contains(summary, "12:00 Uhr") || !strings.Contains(summary, "6 kW") {
		t.Fatalf("summary = %q", summary)
	}
	if detail != "PV und Speicher deckten zu diesem Zeitpunkt den größten Teil." {
		t.Fatalf("detail = %q", detail)
	}
}

func TestEnergyChartUsesSymmetricFiveKWScaleWithHeadroom(t *testing.T) {
	for _, test := range []struct {
		peak float64
		want float64
	}{
		{0, 5},
		{4, 5},
		{5, 10},
		{8.9, 15},
		{10, 15},
		{15.1, 20},
	} {
		if got := energyChartRoundedLimit(test.peak); got != test.want {
			t.Fatalf("energyChartRoundedLimit(%v) = %v, want %v", test.peak, got, test.want)
		}
	}
	ticks := energyChartValueTicks(15)
	labels := make([]string, 0, len(ticks))
	for _, tick := range ticks {
		labels = append(labels, tick.Label)
	}
	if strings.Join(labels, ",") != "15 kW,10 kW,5 kW,0 kW,-5 kW,-10 kW,-15 kW" {
		t.Fatalf("ticks = %v", labels)
	}
	path := energyChartAreaPath(
		[]float64{1, 5, 10}, []bool{true, true, true}, -15, 15, 52, 788,
		energyChartValuePosition(0, -15, 15),
	)
	if !strings.HasSuffix(path, "Z") || !strings.Contains(path, "M52.0 110.0L52.0") {
		t.Fatalf("load area path = %q", path)
	}
}

func TestEnergyChartSamplesExplainSignedFlowsAndOmitMissingValues(t *testing.T) {
	start := time.Date(2026, 7, 29, 8, 0, 0, 0, time.UTC)
	end := start.Add(30 * time.Minute)
	samples := energyChartSamples(start, end, time.UTC, []energyChartData{
		{Key: "load", Label: "Hausverbrauch", Values: []float64{2, 3, 4}, Present: []bool{true, true, true}},
		{Key: "pv", Label: "PV-Erzeugung", Values: []float64{1, 2, 0}, Present: []bool{true, true, false}},
		{Key: "grid", Label: "Netz", Values: []float64{-1, 2, 3}, Present: []bool{true, true, true}},
		{Key: "battery", Label: "Speicher", Values: []float64{-0.5, 0.8, 0}, Present: []bool{true, true, true}},
	}, -15, 15)
	if len(samples) != 3 {
		t.Fatalf("samples = %d, want 3", len(samples))
	}
	if samples[0].Time != "08:00" || samples[1].Time != "08:15" || samples[0].Position == "" || samples[0].HitWidth == "" {
		t.Fatalf("sample geometry/time missing: %+v", samples[:2])
	}
	labels := map[string]string{}
	for _, value := range samples[0].Values {
		labels[value.Key] = value.Label
		if value.Position == "" || value.MobilePosition == "" {
			t.Fatalf("sample value position missing: %+v", value)
		}
	}
	if labels["grid"] != "Einspeisung" || labels["battery"] != "Speicher lädt" {
		t.Fatalf("signed flow labels = %+v", labels)
	}
	if got := len(samples[2].Values); got != 3 {
		t.Fatalf("missing PV value was not omitted: values=%+v", samples[2].Values)
	}
}

func TestEnergyMappingSlotsExplainRequiredAndDerivedValues(t *testing.T) {
	assets := []energy.Asset{
		{Kind: "pv", Confirmed: true},
		{Kind: "battery", Confirmed: true},
	}
	mappings := []energy.EntityMapping{
		{Metric: energy.MetricLoadPower, Confirmed: true},
		{Metric: energy.MetricGridImportPower, Confirmed: true},
		{Metric: energy.MetricPVPower, Confirmed: true},
		{Metric: energy.MetricBatteryPower, Confirmed: true},
	}
	slots := buildEnergyMappingSlots(assets, mappings)
	byKey := map[string]energyMappingSlotView{}
	for _, slot := range slots {
		byKey[slot.Key] = slot
	}
	if byKey["load"].Status != "Zugeordnet" || byKey["pv"].Status != "Zugeordnet" ||
		byKey["battery-soc"].Status != "Noch zuordnen" ||
		byKey["peaks"].Status != "Wird berechnet" {
		t.Fatalf("mapping slots = %+v", slots)
	}
	withoutAssets := buildEnergyMappingSlots(nil, nil)
	for _, slot := range withoutAssets {
		if (slot.Key == "pv" || slot.Key == "battery-power" || slot.Key == "battery-soc") &&
			slot.Status != "Derzeit nicht benötigt" {
			t.Fatalf("optional slot %q = %+v", slot.Key, slot)
		}
	}
}

func TestCurrentEnergyMetricsUsesLiveTimestampAndCorrectsLegacyConsumptionDisplay(t *testing.T) {
	updated := time.Now().UTC().Add(-time.Minute).Truncate(time.Second)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"entity_id":"sensor.sonnenbatterie_state_consumption_current","state":"7228","attributes":{"friendly_name":"Home Current Consumption","device_class":"power","unit_of_measurement":"W"},"last_updated":%q}`, updated.Format(time.RFC3339))
	}))
	t.Cleanup(server.Close)

	cfg := homeassistant.NewConfig(server.URL, "fixture", "", "", "")
	a := &app{}
	metrics, _, latest := a.currentEnergyMetrics(t.Context(), tenantConfig{Slug: "jhw22", HA: cfg}, []energy.EntityMapping{{
		TenantSlug: "jhw22", EntityID: "sensor.sonnenbatterie_state_consumption_current",
		Metric: energy.MetricBatteryPower, DisplayName: "Home Current Consumption", Unit: "W", Confirmed: true,
	}}, energy.DefaultProfile("jhw22", time.Now()))
	if len(metrics) != 1 || metrics[0].Metric != energy.MetricLoadPower || metrics[0].Value != "7,23 kW" {
		t.Fatalf("metrics = %+v", metrics)
	}
	if !latest.Equal(updated) {
		t.Fatalf("latest = %s, want %s", latest, updated)
	}
	if quality := energy.AssessQuality(time.Now(), latestEnergySeen(nil, nil, latest), 0, 0); quality.Status != energy.QualityMeasured {
		t.Fatalf("quality = %+v", quality)
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
	if err := a.saveManualEnergyMapping("jhw22", "sensor.grid_power", energy.MetricLoadPower, "Hausverbrauch korrigiert", "kW", ""); err != nil {
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
	if err := a.saveManualEnergyMapping("jhw22", "switch.wallbox", energy.MetricLoadPower, "Nope", "kW", ""); err == nil {
		t.Fatal("switch entity should be rejected")
	}
}

func TestEnergyMappingsLinkOnlySameHouseAssetsAndExplainCoverage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"entity_id":"sensor.grid_import_power","state":"2.4","attributes":{"friendly_name":"Netzbezug","device_class":"power","unit_of_measurement":"kW"},"last_updated":"2026-07-28T10:00:00Z"},
			{"entity_id":"sensor.pv_current_power","state":"3.1","attributes":{"friendly_name":"PV Leistung","device_class":"power","unit_of_measurement":"kW"},"last_updated":"2026-07-28T10:00:00Z"}
		]`))
	}))
	t.Cleanup(server.Close)
	a := newTestPortalApp(t, userProfile{
		Email:       "owner@example.com",
		Role:        roleOwner,
		Tenants:     []string{"jhw22"},
		AuthMethods: defaultAuthMethods(),
	})
	for _, kind := range []string{"pv", "ev", "heat-pump"} {
		if err := a.energyStore.UpsertAsset(energy.Asset{
			ID:          energy.StableAssetID("jhw22", kind),
			TenantSlug:  "jhw22",
			Kind:        kind,
			Name:        energy.AssetKindLabel(kind),
			Confirmed:   true,
			Flexibility: energy.FlexUnknown,
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := a.energyStore.UpsertAsset(energy.Asset{
		ID:         energy.StableAssetID("other-home", "pv"),
		TenantSlug: "other-home",
		Kind:       "pv",
		Name:       "Fremde PV",
		Confirmed:  true,
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.saveManualEnergyMapping(
		"jhw22", "sensor.foreign", energy.MetricPVPower, "Fremd", "kW",
		energy.StableAssetID("other-home", "pv"),
	); err == nil {
		t.Fatal("manual mapping accepted an asset from another house")
	}

	tenant := a.tenants["jhw22"]
	tenant.HA = homeassistant.NewConfig(server.URL, "fixture", "", "", "")
	if err := a.saveSelectedEnergyMappings(t.Context(), tenant, []string{
		"sensor.grid_import_power",
		"sensor.pv_current_power",
	}); err != nil {
		t.Fatalf("saveSelectedEnergyMappings: %v", err)
	}
	mappings, err := a.energyStore.ListMappings("jhw22")
	if err != nil {
		t.Fatal(err)
	}
	var pvMapping energy.EntityMapping
	for _, mapping := range mappings {
		if mapping.Metric == energy.MetricPVPower {
			pvMapping = mapping
		}
	}
	if pvMapping.AssetID != energy.StableAssetID("jhw22", "pv") {
		t.Fatalf("automatic PV asset link = %+v", pvMapping)
	}
	assets, _ := a.energyStore.ListAssets("jhw22")
	coverage, summary := buildEnergyCoverageViews(assets, mappings)
	if summary != "2 von 4 Bereichen gemessen" {
		t.Fatalf("coverage summary = %q, rows=%+v", summary, coverage)
	}
	status := map[string]string{}
	for _, row := range coverage {
		status[row.Label] = row.Status
	}
	if status["Hausanschluss"] != "Gemessen" || status["PV-Anlage"] != "Gemessen" ||
		status["E-Auto"] != "Nur erfasst" || status["Wärmepumpe"] != "Nur erfasst" {
		t.Fatalf("coverage = %+v", coverage)
	}
}

func TestEnergyOnboardingCanSkipConnectionWithoutDeletingMappings(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "owner@example.com",
		Role:        roleOwner,
		Tenants:     []string{"jhw22"},
		AuthMethods: defaultAuthMethods(),
	})
	profile := energy.DefaultProfile("jhw22", time.Now())
	profile.OnboardingStep = 4
	if err := a.energyStore.SaveProfile(profile); err != nil {
		t.Fatal(err)
	}
	if err := a.saveManualEnergyMapping("jhw22", "sensor.grid_power", energy.MetricGridImportPower, "Bestehender Netzbezug", "kW", ""); err != nil {
		t.Fatal(err)
	}
	response := authedFormRequest(t, a, "owner@example.com", "/app/zuhause/onboarding", url.Values{
		"action": {"skip-mappings"},
	})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
	mappings, err := a.energyStore.ListMappings("jhw22")
	if err != nil || len(mappings) != 1 || mappings[0].DisplayName != "Bestehender Netzbezug" {
		t.Fatalf("mappings after skip = %+v err=%v", mappings, err)
	}
}
