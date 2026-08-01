package server

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/markus-barta/hausv-org/internal/config"
	"github.com/markus-barta/hausv-org/internal/energy"
	"github.com/markus-barta/hausv-org/internal/homeassistant"
	"github.com/markus-barta/hausv-org/internal/store"
)

func TestPublicHomeCopyFollowsPortalTypeNotEnergyProfile(t *testing.T) {
	tests := []struct {
		name           string
		portalType     string
		energyHomeType string
		headline       string
		lead           string
	}{
		{
			name:           "apartment portal despite community energy profile",
			portalType:     config.PortalTypeApartment,
			energyHomeType: energy.HomeCommunity,
			headline:       "Alles Wichtige für Ihre Wohnung.",
			lead:           "Aushänge, Termine, Dokumente, Anliegen und Energie – privat an einem Ort.",
		},
		{
			name:       "private house portal before energy onboarding",
			portalType: config.PortalTypeHouse,
			headline:   "Alles Wichtige für Ihr Zuhause.",
			lead:       "Termine, Dokumente, Aufgaben und Energie – privat an einem Ort.",
		},
		{
			name:           "jhw22 community despite penthouse energy profile",
			portalType:     config.PortalTypeCommunity,
			energyHomeType: energy.HomeApartment,
			headline:       "Alles Wichtige rund um unser Haus.",
			lead:           "Aushänge, Termine, Dokumente und Anliegen – privat an einem Ort.",
		},
		{
			name:           "legacy tenant defaults to community",
			energyHomeType: "unclaimed",
			headline:       "Alles Wichtige rund um unser Haus.",
			lead:           "Aushänge, Termine, Dokumente und Anliegen – privat an einem Ort.",
		},
		{
			name:       "invalid direct tenant stays neutral",
			portalType: "unknown",
			headline:   "Alles Wichtige an einem Ort.",
			lead:       "Aushänge, Termine, Dokumente und Anliegen – nur für eingeladene Personen.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newTestPortalApp(t, userProfile{
				Email: "owner@example.com", Role: roleOwner, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods(),
			})
			tenant := a.tenants["jhw22"]
			tenant.PortalType = tt.portalType
			a.tenants["jhw22"] = tenant
			switch tt.energyHomeType {
			case "":
				// No profile is the pre-onboarding state and must not affect
				// the independent portal classification.
			case "unclaimed":
				if err := a.energyStore.SaveProfile(energy.DefaultProfile("jhw22", time.Now())); err != nil {
					t.Fatal(err)
				}
			default:
				saveClaimedEnergyProfileHAUSV410(t, a, tt.energyHomeType)
			}

			got := a.publicHomeCopy("jhw22")
			if got.Headline != tt.headline || got.Lead != tt.lead {
				t.Fatalf("public copy = %+v, want headline %q and lead %q", got, tt.headline, tt.lead)
			}
		})
	}
}

func TestLocalDevLoginRerenderUsesHomeTypeAwareCopy(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email: "owner@example.com", Role: roleOwner, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods(),
	})
	tenant := a.tenants["jhw22"]
	tenant.PortalType = config.PortalTypeHouse
	a.tenants["jhw22"] = tenant
	saveClaimedEnergyProfileHAUSV410(t, a, energy.HomeHouse)
	a.localDevLogin = true

	form := url.Values{"email": {"owner@example.com"}}
	req := httptest.NewRequest(http.MethodPost, "http://jhw22.hausv.org/auth/request", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://jhw22.hausv.org")
	rr := httptest.NewRecorder()
	a.handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("local dev login status = %d body=%s", rr.Code, rr.Body.String())
	}
	for _, want := range []string{"Alles Wichtige für Ihr Zuhause.", "E-Mail prüfen", "Weiter zum Portal", "Andere Adresse verwenden"} {
		if !strings.Contains(rr.Body.String(), want) {
			t.Fatalf("local dev login page missing %q:\n%s", want, rr.Body.String())
		}
	}
	if got := strings.Count(rr.Body.String(), `action="/auth/request"`); got != 1 {
		t.Fatalf("prepared login renders %d competing request forms, want one collapsed retry form", got)
	}
	for _, forbidden := range []string{"Lokalen Testzugang", "lokale Mailversand", "Magic-Link", "SSO", "Zitadel"} {
		if strings.Contains(rr.Body.String(), forbidden) {
			t.Fatalf("prepared login page exposes implementation wording %q", forbidden)
		}
	}
}

func TestPrivacyCopyDistinguishesPrivateHomeAndCommunityEnergyContext(t *testing.T) {
	tests := []struct {
		name           string
		portalType     string
		energyHomeType string
		want           []string
		notWanted      string
	}{
		{
			name:           "private house",
			portalType:     config.PortalTypeHouse,
			energyHomeType: energy.HomeHouse,
			want: []string{
				"Bei einem privaten Zuhause entscheidet die Eigentümerin oder der Eigentümer",
				"Technische Vertrauenspersonen sehen oder konfigurieren Energie nur im sichtbar erteilten Umfang",
				"Smart-Meter-Originaldateien werden nach 30 Tagen",
				"normalisierte Viertelstundenwerte nach 13 Monaten",
				"festgehaltene Tarifbewertungen nach drei Jahren",
				"keine ausschließlich automatisierte Entscheidung",
				"Solange kein externer Auditor verfügbar ist",
				"Beginn des dreijährigen kostenlosen Nutzungszeitraums",
			},
			notWanted: "Bei einer Hausgemeinschaft entscheidet",
		},
		{
			name:           "community with apartment energy profile",
			portalType:     config.PortalTypeCommunity,
			energyHomeType: energy.HomeApartment,
			want: []string{
				"Bei einer Hausgemeinschaft entscheidet die Eigentümergemeinschaft",
				"Die datenschutzrechtliche Rolle wird deshalb je Zweck bestimmt",
				"Home-Assistant-Endpunkt und Zugangstoken bleiben in der verschlüsselten Host-Konfiguration",
				"Eigentümer und Hausadministration können unter",
				"Energiedaten &amp; Datenschutz",
			},
			notWanted: "Bei einem privaten Zuhause entscheidet",
		},
		{
			name:           "community after energy profile reset",
			portalType:     config.PortalTypeCommunity,
			energyHomeType: "unclaimed",
			want: []string{
				"Bei einer Hausgemeinschaft entscheidet die Eigentümergemeinschaft",
				"Beginn des kostenlosen Anspruchs bleibt bis zum Ende",
			},
			notWanted: "Bei einem privaten Zuhause entscheidet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newTestPortalApp(t, userProfile{
				Email: "manager@example.com", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods(),
			})
			tenant := a.tenants["jhw22"]
			tenant.PortalType = tt.portalType
			a.tenants["jhw22"] = tenant
			if tt.energyHomeType == "unclaimed" {
				if err := a.energyStore.SaveProfile(energy.DefaultProfile("jhw22", time.Now())); err != nil {
					t.Fatal(err)
				}
			} else {
				saveClaimedEnergyProfileHAUSV410(t, a, tt.energyHomeType)
			}

			rr := httptest.NewRecorder()
			a.handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "http://jhw22.hausv.org/datenschutz", nil))
			if rr.Code != http.StatusOK {
				t.Fatalf("privacy status = %d body=%s", rr.Code, rr.Body.String())
			}
			body := rr.Body.String()
			for _, want := range tt.want {
				if !strings.Contains(body, want) {
					t.Fatalf("privacy page missing %q:\n%s", want, body)
				}
			}
			if strings.Contains(body, tt.notWanted) {
				t.Fatalf("privacy page unexpectedly contains %q:\n%s", tt.notWanted, body)
			}
		})
	}
}

func TestEnergyDataPageRoleMatrix(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods(),
	})
	saveClaimedEnergyProfileHAUSV410(t, a, energy.HomeHouse)
	a.serviceAccessEnabled = true
	for email, profile := range map[string]userProfile{
		"manager@example.com": {
			Email: "manager@example.com", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods(),
		},
		"owner@example.com": {
			Email: "owner@example.com", Role: roleOwner, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods(),
		},
		"resident@example.com": {
			Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods(),
		},
		"renter@example.com": {
			Email: "renter@example.com", Role: roleRenter, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods(),
		},
		"board@example.com": {
			Email: "board@example.com", Role: roleBeirat, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods(),
		},
		"caretaker@example.com": {
			Email: "caretaker@example.com", Role: roleResident, Permissions: []string{
				permissionEnergyView, permissionEnergyConfigure, permissionEnergyCaretaker,
			}, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods(),
		},
		"service@example.com": {
			Email: "service@example.com", Role: roleServiceProvider, Permissions: []string{
				permissionEnergyView, permissionEnergyConfigure,
			}, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods(),
		},
	} {
		a.profiles[email] = profile
	}

	for _, tt := range []struct {
		email string
		want  int
	}{
		{email: "admin@example.com", want: http.StatusOK},
		{email: "manager@example.com", want: http.StatusOK},
		{email: "owner@example.com", want: http.StatusOK},
		{email: "resident@example.com", want: http.StatusForbidden},
		{email: "renter@example.com", want: http.StatusForbidden},
		{email: "board@example.com", want: http.StatusForbidden},
		{email: "caretaker@example.com", want: http.StatusForbidden},
		{email: "service@example.com", want: http.StatusForbidden},
	} {
		t.Run(tt.email, func(t *testing.T) {
			rr := authedRequest(t, a, tt.email, "/app/settings/energy-data")
			if rr.Code != tt.want {
				t.Fatalf("energy data page status = %d, want %d body=%s", rr.Code, tt.want, rr.Body.String())
			}
		})
	}
}

func TestEnergyDataPageRendersUnderstandableLifecycleAndSeparateDeleteChoices(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email: "owner@example.com", Role: roleOwner, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods(),
	})
	saveClaimedEnergyProfileHAUSV410(t, a, energy.HomeHouse)
	if _, err := a.energyStore.PutImport(energy.ImportRecord{
		ID: "import-page", TenantSlug: "jhw22", Filename: "mein-smart-meter.csv",
		SHA256: "page-sha", Format: "smart-meter-csv", Payload: []byte("page"), ImportedAt: time.Now(),
	}, nil); err != nil {
		t.Fatal(err)
	}

	rr := authedRequest(t, a, "owner@example.com", "/app/settings/energy-data")
	if rr.Code != http.StatusOK {
		t.Fatalf("energy data page status = %d body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	for _, want := range []string{
		"Energiedaten &amp; Datenschutz",
		"Home Assistant nur gelesen",
		"30 Tage",
		"13 Monate",
		"3 Jahre",
		"Energiedaten exportieren",
		"mein-smart-meter.csv",
		"Nur Messverlauf löschen",
		"MESSVERLAUF LÖSCHEN",
		"Ganzes Energieprofil löschen",
		"ENERGIEPROFIL LÖSCHEN",
		"Unabhängige Anliegen, Dokumente und Sicherheitsnachweise",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("energy data page missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, `action="/app/settings/energy-data/import/`) {
		t.Fatalf("page offers unsafe per-import deletion:\n%s", body)
	}
}

func TestEnergyExportIsTenantScopedSecretFreeAndRedactsOtherActors(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email: "owner@example.com", Role: roleOwner, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods(),
	})
	saveClaimedEnergyProfileHAUSV410(t, a, energy.HomeHouse)
	tenant := a.tenants["jhw22"]
	tenant.HA = homeassistant.NewConfig(
		"https://home-assistant-secret.invalid",
		"HOME-ASSISTANT-TOKEN-SENTINEL",
		"",
		"",
		"",
	)
	a.tenants["jhw22"] = tenant

	currentAssetID := "asset-current"
	if err := a.energyStore.UpsertAsset(energy.Asset{
		ID: currentAssetID, TenantSlug: "jhw22", Kind: "pv", Name: "Eigene PV",
		Confirmed: true, Metadata: map[string]string{
			"manufacturer":  "Sicherer Hersteller",
			"api_token":     "ASSET-TOKEN-SENTINEL",
			"api_key":       "ASSET-API-KEY-SENTINEL",
			"authorization": "ASSET-AUTHORIZATION-SENTINEL",
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.energyStore.UpsertMapping(energy.EntityMapping{
		ID: "mapping-current", TenantSlug: "jhw22", EntityID: "sensor.pv_power",
		AssetID: currentAssetID, Metric: energy.MetricPVPower, DisplayName: "PV Leistung", Unit: "kW", Confirmed: true,
	}); err != nil {
		t.Fatal(err)
	}
	currentInterval := energy.Interval{
		TenantSlug: "jhw22", StartsAt: time.Date(2026, 7, 29, 8, 0, 0, 0, time.UTC),
		Duration: 15 * time.Minute, ImportKWh: 0.5, AverageKW: 2, Quality: "measured", Source: "smart-meter",
	}
	if _, err := a.energyStore.PutImport(energy.ImportRecord{
		ID: "import-current", TenantSlug: "jhw22", Filename: "eigene-werte.csv",
		SHA256: "current-sha", Format: "smart-meter-csv",
		Payload: []byte("CURRENT_IMPORT_MARKER"), ImportedAt: time.Now(),
	}, []energy.Interval{currentInterval}); err != nil {
		t.Fatal(err)
	}

	if err := a.energyStore.SaveProfile(func() energy.HomeProfile {
		profile := energy.DefaultProfile("other-house", time.Now())
		profile.HouseholdName = "FOREIGN_HOME_MARKER"
		profile.HomeType = energy.HomeHouse
		profile.OnboardingComplete = true
		profile.OnboardingStep = 5
		return profile
	}()); err != nil {
		t.Fatal(err)
	}
	if err := a.energyStore.UpsertAsset(energy.Asset{
		ID: "asset-foreign", TenantSlug: "other-house", Kind: "pv", Name: "FOREIGN_ASSET_MARKER", Confirmed: true,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.energyStore.PutImport(energy.ImportRecord{
		ID: "import-foreign", TenantSlug: "other-house", Filename: "foreign.csv",
		SHA256: "foreign-sha", Format: "smart-meter-csv",
		Payload: []byte("FOREIGN_IMPORT_MARKER"), ImportedAt: time.Now(),
	}, nil); err != nil {
		t.Fatal(err)
	}
	if err := a.auditStore.Append(auditEvent{
		TenantSlug: "jhw22", ActorEmail: "third-party-auditor@example.invalid", ActorRole: roleManager,
		Action: store.AuditActionEnergyTarget, TargetType: "home-energy", TargetID: "jhw22",
		Summary: "Energieziel aktualisiert",
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.auditStore.Append(auditEvent{
		TenantSlug: "jhw22", ActorEmail: "owner@example.com", ActorRole: roleOwner,
		Action: store.AuditActionEnergyOnboarding, TargetType: "home-energy", TargetID: "jhw22",
		Summary: "Energieprofil eingerichtet",
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.auditStore.Append(auditEvent{
		TenantSlug: "other-house", ActorEmail: "foreign@example.invalid", ActorRole: roleOwner,
		Action: store.AuditActionEnergyOnboarding, TargetType: "home-energy", TargetID: "other-house",
		Summary: "FOREIGN_AUDIT_MARKER",
	}); err != nil {
		t.Fatal(err)
	}

	ac := authCtx{email: "owner@example.com", role: roleOwner, tenant: a.tenants["jhw22"]}
	payload, filename, counts, err := a.buildEnergyDataPackage(ac, time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("buildEnergyDataPackage: %v", err)
	}
	if filename != "hausv-energiedaten-jhw22-20260729.zip" {
		t.Fatalf("filename = %q", filename)
	}
	if counts["assets"] != 1 || counts["mappings"] != 1 || counts["intervals"] != 1 || counts["raw_imports"] != 1 {
		t.Fatalf("export counts = %+v", counts)
	}

	files := readEnergyZIPHAUSV410(t, payload)
	for _, name := range []string{"manifest.json", "energiedaten.json", "viertelstundenwerte.csv", "rohimporte/01-eigene-werte.csv"} {
		if _, ok := files[name]; !ok {
			t.Fatalf("export missing %q; files=%v", name, mapKeysHAUSV410(files))
		}
	}
	all := bytes.Join(mapValuesHAUSV410(files), []byte("\n"))
	for _, forbidden := range []string{
		"https://home-assistant-secret.invalid",
		"HOME-ASSISTANT-TOKEN-SENTINEL",
		"ASSET-TOKEN-SENTINEL",
		"ASSET-API-KEY-SENTINEL",
		"ASSET-AUTHORIZATION-SENTINEL",
		"third-party-auditor@example.invalid",
		"FOREIGN_HOME_MARKER",
		"FOREIGN_ASSET_MARKER",
		"FOREIGN_IMPORT_MARKER",
		"FOREIGN_AUDIT_MARKER",
	} {
		if bytes.Contains(all, []byte(forbidden)) {
			t.Fatalf("export leaked %q", forbidden)
		}
	}
	for _, want := range []string{"CURRENT_IMPORT_MARKER", "Sicherer Hersteller", "Andere berechtigte Person", `"actor": "Sie"`} {
		if !bytes.Contains(all, []byte(want)) {
			t.Fatalf("export missing %q", want)
		}
	}

	var metadata struct {
		EnergyAudit []struct {
			Actor string `json:"actor"`
		} `json:"energy_audit_live"`
	}
	if err := json.Unmarshal(files["energiedaten.json"], &metadata); err != nil {
		t.Fatalf("decode energy metadata: %v", err)
	}
	if len(metadata.EnergyAudit) != 2 {
		t.Fatalf("energy audit rows = %+v", metadata.EnergyAudit)
	}
}

func TestMeasurementDeletionKeepsSetupAndResetsDerivedRecommendationState(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email: "owner@example.com", Role: roleOwner, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods(),
	})
	profile := saveClaimedEnergyProfileHAUSV410(t, a, energy.HomeHouse)
	profile.RecommendationID = "shift-load"
	profile.RecommendationStatus = "deferred"
	if err := a.energyStore.SaveProfile(profile); err != nil {
		t.Fatal(err)
	}
	if err := a.energyStore.UpsertAsset(energy.Asset{
		ID: "asset-keep", TenantSlug: "jhw22", Kind: "pv", Name: "PV", Confirmed: true,
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.energyStore.UpsertMapping(energy.EntityMapping{
		ID: "mapping-keep", TenantSlug: "jhw22", EntityID: "sensor.pv",
		AssetID: "asset-keep", Metric: energy.MetricPVPower, DisplayName: "PV", Confirmed: true,
	}); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 7, 29, 8, 0, 0, 0, time.UTC)
	peak := 8.4
	if _, err := a.energyStore.PutImport(energy.ImportRecord{
		ID: "import-delete", TenantSlug: "jhw22", Filename: "delete.csv",
		SHA256: "delete-sha", Format: "smart-meter-csv", Payload: []byte("delete me"), ImportedAt: at,
	}, []energy.Interval{{
		TenantSlug: "jhw22", StartsAt: at, Duration: 15 * time.Minute,
		ImportKWh: 2.1, AverageKW: peak, Quality: "measured", Source: "smart-meter",
	}}); err != nil {
		t.Fatal(err)
	}
	if err := a.energyStore.SaveTariffAssessment(energy.TariffAssessment{
		ID: "assessment-delete", TenantSlug: "jhw22", AssessmentMonth: "2026-07",
		ProfileID: "draft-at", ProfileVersion: "2026-07", ProfileStatus: "draft", PeakKW: peak, CreatedAt: at,
	}); err != nil {
		t.Fatal(err)
	}
	beforeFrom := at.Add(-24 * time.Hour)
	afterTo := at.Add(24 * time.Hour)
	if err := a.energyStore.UpsertMeasure(energy.Measure{
		ID: "measure-keep", TenantSlug: "jhw22", IssueID: "issue-1",
		RecommendationID: "shift-load", Title: "Last verschieben", Status: energy.MeasureCompleted,
		BeforeFrom: &beforeFrom, AfterTo: &afterTo, BeforePeakKW: &peak, BeforeQuality: "measured",
	}); err != nil {
		t.Fatal(err)
	}

	rr := authedFormRequest(t, a, "owner@example.com", "/app/settings/energy-data/history/delete", url.Values{
		"confirmation": {"MESSVERLAUF LÖSCHEN"},
	})
	if rr.Code != http.StatusSeeOther || rr.Header().Get("Location") != "/app/settings/energy-data?result=history-deleted" {
		t.Fatalf("history delete = %d location=%q body=%s", rr.Code, rr.Header().Get("Location"), rr.Body.String())
	}
	stored, exists, err := a.energyStore.Profile("jhw22")
	if err != nil || !exists {
		t.Fatalf("profile after history deletion: exists=%v err=%v", exists, err)
	}
	if stored.HouseholdName != "Test Zuhause" || !stored.OnboardingComplete ||
		stored.RecommendationID != "" || stored.RecommendationStatus != "" {
		t.Fatalf("profile after history deletion = %+v", stored)
	}
	assets, _ := a.energyStore.ListAssets("jhw22")
	mappings, _ := a.energyStore.ListMappings("jhw22")
	imports, _ := a.energyStore.ListImportsForExport("jhw22")
	intervals, _ := a.energyStore.ListIntervals("jhw22", time.Time{}, time.Time{})
	assessments, _ := a.energyStore.ListTariffAssessments("jhw22")
	if len(assets) != 1 || len(mappings) != 1 || len(imports) != 0 || len(intervals) != 0 || len(assessments) != 0 {
		t.Fatalf("history deletion boundaries: assets=%d mappings=%d imports=%d intervals=%d assessments=%d",
			len(assets), len(mappings), len(imports), len(intervals), len(assessments))
	}
	measure, exists, err := a.energyStore.GetMeasure("jhw22", "measure-keep")
	if err != nil || !exists {
		t.Fatalf("measure after history deletion: exists=%v err=%v", exists, err)
	}
	if measure.BeforeFrom != nil || measure.AfterTo != nil || measure.BeforePeakKW != nil ||
		measure.BeforeQuality != "" || measure.Title != "Last verschieben" {
		t.Fatalf("measure comparison was not selectively reset: %+v", measure)
	}
	assertEnergyDeleteAuditStagesHAUSV410(t, a, store.AuditActionEnergyHistoryDelete)
}

func TestProfileDeletionAllowsCleanReonboardingPreservesEntitlementAndRevokesEnvGrant(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email: "owner@example.com", Role: roleOwner, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods(),
	})
	profile := saveClaimedEnergyProfileHAUSV410(t, a, energy.HomeHouse)
	if profile.FreeStartedAt == nil {
		t.Fatal("claimed profile has no free-period start")
	}
	started := *profile.FreeStartedAt
	if err := a.energyStore.UpsertAsset(energy.Asset{
		ID: "asset-delete", TenantSlug: "jhw22", Kind: "pv", Name: "PV", Confirmed: true,
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.energyStore.UpsertMapping(energy.EntityMapping{
		ID: "mapping-delete", TenantSlug: "jhw22", EntityID: "sensor.pv",
		AssetID: "asset-delete", Metric: energy.MetricPVPower, DisplayName: "PV", Confirmed: true,
	}); err != nil {
		t.Fatal(err)
	}
	a.profiles["helper@example.com"] = userProfile{
		Email: "helper@example.com", Role: roleResident,
		Permissions: []string{permissionParking, permissionEnergyView, permissionEnergyConfigure, permissionEnergyCaretaker},
		Tenants:     []string{"jhw22"}, AuthMethods: defaultAuthMethods(),
	}
	a.energyChartCache = map[string]energyChartCacheEntry{
		"jhw22|sensor.private|24h": {
			View: energyChartView{Summary: "transient private history"}, ExpiresAt: time.Now().Add(5 * time.Minute),
		},
		"other-house|sensor.keep|24h": {
			View: energyChartView{Summary: "other house"}, ExpiresAt: time.Now().Add(5 * time.Minute),
		},
	}

	rr := authedFormRequest(t, a, "owner@example.com", "/app/settings/energy-data/profile/delete", url.Values{
		"confirmation": {"ENERGIEPROFIL LÖSCHEN"},
	})
	if rr.Code != http.StatusSeeOther || rr.Header().Get("Location") != "/app/zuhause/onboarding?reset=1" {
		t.Fatalf("profile delete = %d location=%q body=%s", rr.Code, rr.Header().Get("Location"), rr.Body.String())
	}
	stored, exists, err := a.energyStore.Profile("jhw22")
	if err != nil || !exists {
		t.Fatalf("placeholder after deletion: exists=%v err=%v", exists, err)
	}
	if !energyProfileUnclaimed(stored) || stored.FreeStartedAt == nil || !stored.FreeStartedAt.Equal(started) {
		t.Fatalf("placeholder after deletion = %+v", stored)
	}
	assets, _ := a.energyStore.ListAssets("jhw22")
	mappings, _ := a.energyStore.ListMappings("jhw22")
	if len(assets) != 0 || len(mappings) != 0 {
		t.Fatalf("profile content remained after deletion: assets=%+v mappings=%+v", assets, mappings)
	}
	if _, exists := a.energyChartCache["jhw22|sensor.private|24h"]; exists {
		t.Fatal("tenant Home Assistant history remained in memory after profile deletion")
	}
	if _, exists := a.energyChartCache["other-house|sensor.keep|24h"]; !exists {
		t.Fatal("profile deletion cleared another tenant's chart cache")
	}

	helper, ok := a.inviteStore.Get("helper@example.com")
	if !ok || !helper.Adopted {
		t.Fatalf("env helper was not converted to a revocable override: %+v ok=%v", helper, ok)
	}
	effective := a.profileForTenant("helper@example.com", "jhw22")
	if effective.HasPermission(permissionEnergyView) ||
		effective.HasPermission(permissionEnergyConfigure) ||
		effective.HasPermission(permissionEnergyControl) ||
		effective.HasPermission(permissionEnergyCaretaker) {
		t.Fatalf("technical energy grant remained effective: %+v", effective.Permissions)
	}
	if !effective.HasPermission(permissionParking) {
		t.Fatalf("unrelated helper permission was removed: %+v", effective.Permissions)
	}

	const seed = `[{"tenant_slug":"jhw22","household_name":"MUST NOT RETURN","home_type":"house","assets":["pv"],"complete":true}]`
	if err := energy.ApplyProfileSeeds(a.energyStore, seed, map[string]struct{}{"jhw22": {}}, time.Now()); err != nil {
		t.Fatalf("apply profile seed after deletion: %v", err)
	}
	stored, _, _ = a.energyStore.Profile("jhw22")
	assets, _ = a.energyStore.ListAssets("jhw22")
	if !energyProfileUnclaimed(stored) || stored.HouseholdName != "" || len(assets) != 0 {
		t.Fatalf("declarative seed resurrected deleted profile: profile=%+v assets=%+v", stored, assets)
	}
	onboarding := authedRequest(t, a, "owner@example.com", "/app/zuhause/onboarding?reset=1")
	if onboarding.Code != http.StatusOK || !strings.Contains(onboarding.Body.String(), "Womit möchten Sie beginnen? Mit Ihrem Zuhause.") {
		t.Fatalf("clean re-onboarding unavailable: status=%d body=%s", onboarding.Code, onboarding.Body.String())
	}
	assertEnergyDeleteAuditStagesHAUSV410(t, a, store.AuditActionEnergyProfileDelete)
}

func TestEnergyChartCachePrunesExpiredEntries(t *testing.T) {
	now := time.Now()
	a := &app{energyChartCache: map[string]energyChartCacheEntry{
		"expired|sensor.old|24h": {
			View: energyChartView{Summary: "expired"}, ExpiresAt: now.Add(-time.Second),
		},
	}}
	a.cacheEnergyChart(
		"current|sensor.new|24h",
		energyChartView{Summary: "current"},
		now.Add(5*time.Minute),
	)
	if _, exists := a.energyChartCache["expired|sensor.old|24h"]; exists {
		t.Fatal("expired transient Home Assistant history remained cached")
	}
	if _, exists := a.energyChartCache["current|sensor.new|24h"]; !exists {
		t.Fatal("current chart was not cached")
	}
}

func saveClaimedEnergyProfileHAUSV410(t *testing.T, a *app, homeType string) energy.HomeProfile {
	t.Helper()
	now := time.Date(2026, 7, 29, 7, 0, 0, 0, time.UTC)
	profile := energy.DefaultProfile("jhw22", now)
	profile.HomeType = homeType
	profile.HouseholdName = "Test Zuhause"
	profile.OnboardingStep = 5
	profile.OnboardingComplete = true
	started := now
	profile.FreeStartedAt = &started
	if err := a.energyStore.SaveProfile(profile); err != nil {
		t.Fatalf("save claimed profile: %v", err)
	}
	stored, _, _ := a.energyStore.Profile("jhw22")
	return stored
}

func readEnergyZIPHAUSV410(t *testing.T, payload []byte) map[string][]byte {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		t.Fatalf("open energy ZIP: %v", err)
	}
	files := make(map[string][]byte, len(reader.File))
	for _, file := range reader.File {
		source, err := file.Open()
		if err != nil {
			t.Fatalf("open ZIP entry %q: %v", file.Name, err)
		}
		content, readErr := io.ReadAll(source)
		closeErr := source.Close()
		if readErr != nil {
			t.Fatalf("read ZIP entry %q: %v", file.Name, readErr)
		}
		if closeErr != nil {
			t.Fatalf("close ZIP entry %q: %v", file.Name, closeErr)
		}
		files[file.Name] = content
	}
	return files
}

func mapKeysHAUSV410(items map[string][]byte) []string {
	out := make([]string, 0, len(items))
	for key := range items {
		out = append(out, key)
	}
	return out
}

func mapValuesHAUSV410(items map[string][]byte) [][]byte {
	out := make([][]byte, 0, len(items))
	for _, value := range items {
		out = append(out, value)
	}
	return out
}

func assertEnergyDeleteAuditStagesHAUSV410(t *testing.T, a *app, action string) {
	t.Helper()
	events := a.auditStore.List(auditFilter{TenantSlug: "jhw22", Action: action, Limit: 10})
	if len(events) != 2 {
		t.Fatalf("%s audit events = %+v, want requested and completed", action, events)
	}
	stages := map[string]bool{}
	for _, event := range events {
		stages[event.Details["stage"]] = true
	}
	if !stages["requested"] || !stages["completed"] {
		t.Fatalf("%s audit stages = %+v", action, stages)
	}
}
