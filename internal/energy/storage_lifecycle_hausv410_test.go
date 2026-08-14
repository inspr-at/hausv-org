package energy_test

import (
	"path/filepath"
	"testing"
	"time"

	appdb "github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/energy"
)

func lifecycleStoreFactories() map[string]func(*testing.T) energy.Storage {
	return map[string]func(*testing.T) energy.Storage{
		"memory": func(*testing.T) energy.Storage {
			return energy.NewMemoryStore()
		},
		"sqlite": func(t *testing.T) energy.Storage {
			t.Helper()
			database, err := appdb.Open(filepath.Join(t.TempDir(), "energy-lifecycle.db"))
			if err != nil {
				t.Fatalf("open lifecycle database: %v", err)
			}
			t.Cleanup(func() { _ = database.Close() })
			return energy.NewSQLStore(database)
		},
	}
}

func TestFreePeriodStartCannotBeClearedOrRestartedHAUSV410(t *testing.T) {
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	original := now.AddDate(-1, 0, 0)
	replacement := now
	for name, factory := range lifecycleStoreFactories() {
		t.Run(name, func(t *testing.T) {
			storage := factory(t)
			profile := energy.DefaultProfile("home-a", now)
			profile.FreeStartedAt = &original
			if err := storage.SaveProfile(profile); err != nil {
				t.Fatalf("save original profile: %v", err)
			}

			profile.FreeStartedAt = nil
			profile.HouseholdName = "Neu eingerichtet"
			if err := storage.SaveProfile(profile); err != nil {
				t.Fatalf("save profile without marker: %v", err)
			}
			stored, exists, err := storage.Profile("home-a")
			if err != nil || !exists || stored.FreeStartedAt == nil || !stored.FreeStartedAt.Equal(original) {
				t.Fatalf("clearing changed free-period start: profile=%+v exists=%v err=%v", stored, exists, err)
			}

			profile.FreeStartedAt = &replacement
			if err := storage.SaveProfile(profile); err != nil {
				t.Fatalf("save profile with replacement marker: %v", err)
			}
			stored, exists, err = storage.Profile("home-a")
			if err != nil || !exists || stored.FreeStartedAt == nil || !stored.FreeStartedAt.Equal(original) {
				t.Fatalf("replacement changed free-period start: profile=%+v exists=%v err=%v", stored, exists, err)
			}
		})
	}
}

func TestDeleteMeasurementDataKeepsConfigurationAndResetsDerivedStateHAUSV410(t *testing.T) {
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	for name, factory := range lifecycleStoreFactories() {
		t.Run(name, func(t *testing.T) {
			storage := factory(t)
			seedLifecycleTenant(t, storage, "home-a", now)
			seedLifecycleTenant(t, storage, "home-b", now)

			summary, err := storage.DeleteMeasurementData(" HOME-A ")
			if err != nil {
				t.Fatalf("delete measurement data: %v", err)
			}
			if summary.Imports != 1 || summary.Intervals != 1 || summary.TariffAssessments != 1 || summary.Measures != 1 {
				t.Fatalf("delete summary = %+v", summary)
			}
			if summary.Profiles != 0 || summary.Assets != 0 || summary.Mappings != 0 || summary.Maintenance != 0 {
				t.Fatalf("measurement delete removed configuration: %+v", summary)
			}

			profile, exists, err := storage.Profile("home-a")
			if err != nil || !exists {
				t.Fatalf("profile after measurement delete: exists=%v err=%v", exists, err)
			}
			if profile.HouseholdName != "Zuhause home-a" || profile.UnitID != "top-home-a" {
				t.Fatalf("profile identity changed: %+v", profile)
			}
			if profile.RecommendationID != "" || profile.RecommendationStatus != "" {
				t.Fatalf("derived recommendation was retained: %+v", profile)
			}
			if profile.FreeStartedAt == nil || !profile.FreeStartedAt.Equal(now.AddDate(-1, 0, 0)) {
				t.Fatalf("free-period start changed: %v", profile.FreeStartedAt)
			}

			assertLifecycleCounts(t, storage, "home-a", lifecycleCounts{
				assets: 1, mappings: 1, maintenance: 1, measures: 1,
			})
			measure, ok, err := storage.GetMeasure("home-a", "measure-home-a")
			if err != nil || !ok {
				t.Fatalf("measure after measurement delete: ok=%v err=%v", ok, err)
			}
			if measure.BeforeFrom != nil || measure.BeforeTo != nil || measure.AfterFrom != nil || measure.AfterTo != nil ||
				measure.BeforePeakKW != nil || measure.AfterPeakKW != nil ||
				measure.BeforeQuality != "" || measure.AfterQuality != "" {
				t.Fatalf("measure comparison survived measurement delete: %+v", measure)
			}
			if measure.Title != "Peak glätten" || measure.Status != energy.MeasureCompleted {
				t.Fatalf("measure metadata changed: %+v", measure)
			}

			assertLifecycleCounts(t, storage, "home-b", lifecycleCounts{
				imports: 1, intervals: 1, assessments: 1, assets: 1, mappings: 1, maintenance: 1, measures: 1,
			})
			foreign, _, _ := storage.Profile("home-b")
			if foreign.RecommendationID == "" || foreign.RecommendationStatus == "" {
				t.Fatalf("other tenant recommendation was reset: %+v", foreign)
			}
		})
	}
}

func TestDeleteProfileRetainsTrialMarkerAndBlocksSeedResurrectionHAUSV410(t *testing.T) {
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	freeStartedAt := now.AddDate(-1, 0, 0)
	const replacementSeed = `[{
		"tenant_slug":"home-a",
		"household_name":"Seed resurrected this",
		"home_type":"house",
		"assets":["battery"],
		"complete":true
	}]`

	for name, factory := range lifecycleStoreFactories() {
		t.Run(name, func(t *testing.T) {
			storage := factory(t)
			seedLifecycleTenant(t, storage, "home-a", now)
			seedLifecycleTenant(t, storage, "home-b", now)

			summary, err := storage.DeleteProfile("home-a")
			if err != nil {
				t.Fatalf("delete profile: %v", err)
			}
			if summary.Profiles != 1 || summary.Assets != 1 || summary.Mappings != 1 ||
				summary.Intervals != 1 || summary.Imports != 1 || summary.Maintenance != 1 ||
				summary.TariffAssessments != 1 || summary.Measures != 1 {
				t.Fatalf("profile delete summary = %+v", summary)
			}

			placeholder, exists, err := storage.Profile("home-a")
			if err != nil || !exists {
				t.Fatalf("deletion marker: exists=%v err=%v", exists, err)
			}
			if placeholder.HouseholdName != "" || placeholder.UnitID != "" ||
				placeholder.OnboardingComplete || placeholder.OnboardingStep != 1 ||
				placeholder.OperatingMode != energy.ModeObserve || placeholder.AutomationStage != energy.StageObserve {
				t.Fatalf("deletion marker contains live profile data: %+v", placeholder)
			}
			if placeholder.FreeStartedAt == nil || !placeholder.FreeStartedAt.Equal(freeStartedAt) {
				t.Fatalf("free-period start not retained: got=%v want=%v", placeholder.FreeStartedAt, freeStartedAt)
			}
			assertLifecycleCounts(t, storage, "home-a", lifecycleCounts{})

			if err := energy.ApplyProfileSeeds(
				storage,
				replacementSeed,
				map[string]struct{}{"home-a": {}},
				now.Add(24*time.Hour),
			); err != nil {
				t.Fatalf("reapply declarative seed: %v", err)
			}
			afterSeed, exists, err := storage.Profile("home-a")
			if err != nil || !exists {
				t.Fatalf("profile after seed reapply: exists=%v err=%v", exists, err)
			}
			if afterSeed.HouseholdName != "" || afterSeed.OnboardingComplete {
				t.Fatalf("seed resurrected deleted profile: %+v", afterSeed)
			}
			if afterSeed.FreeStartedAt == nil || !afterSeed.FreeStartedAt.Equal(freeStartedAt) {
				t.Fatalf("seed reset retained free-period start: %v", afterSeed.FreeStartedAt)
			}
			assertLifecycleCounts(t, storage, "home-a", lifecycleCounts{})

			assertLifecycleCounts(t, storage, "home-b", lifecycleCounts{
				imports: 1, intervals: 1, assessments: 1, assets: 1, mappings: 1, maintenance: 1, measures: 1,
			})
			foreign, exists, err := storage.Profile("home-b")
			if err != nil || !exists || foreign.HouseholdName != "Zuhause home-b" {
				t.Fatalf("other tenant changed: %+v exists=%v err=%v", foreign, exists, err)
			}
		})
	}
}

func TestPurgeExpiredUsesStrictRetentionCutoffsHAUSV410(t *testing.T) {
	cutoff := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	for name, factory := range lifecycleStoreFactories() {
		t.Run(name, func(t *testing.T) {
			storage := factory(t)
			if err := storage.SaveProfile(energy.DefaultProfile("home-a", cutoff)); err != nil {
				t.Fatalf("save profile: %v", err)
			}

			for _, item := range []struct {
				suffix string
				at     time.Time
			}{
				{suffix: "old", at: cutoff.Add(-time.Second)},
				{suffix: "boundary", at: cutoff},
				{suffix: "new", at: cutoff.Add(time.Second)},
			} {
				if inserted, err := storage.PutImport(energy.ImportRecord{
					ID:         "import-" + item.suffix,
					TenantSlug: "home-a",
					Filename:   item.suffix + ".csv",
					SHA256:     "sha-" + item.suffix,
					Format:     "test",
					Payload:    []byte(item.suffix),
					ImportedAt: item.at,
				}, nil); err != nil || !inserted {
					t.Fatalf("put %s import: inserted=%v err=%v", item.suffix, inserted, err)
				}
				if err := storage.PutInterval(energy.Interval{
					TenantSlug: "home-a",
					StartsAt:   item.at,
					Duration:   15 * time.Minute,
					ImportKWh:  1,
					AverageKW:  4,
					Quality:    energy.QualityMeasured,
					Source:     item.suffix,
					CreatedAt:  item.at,
				}); err != nil {
					t.Fatalf("put %s interval: %v", item.suffix, err)
				}
				if err := storage.SaveTariffAssessment(energy.TariffAssessment{
					ID:              "assessment-" + item.suffix,
					TenantSlug:      "home-a",
					AssessmentMonth: "2026-07",
					ProfileID:       "at-power",
					ProfileVersion:  "draft",
					ProfileStatus:   "draft",
					PeakKW:          5,
					DataQuality:     energy.QualityMeasured,
					CreatedAt:       item.at,
				}); err != nil {
					t.Fatalf("save %s assessment: %v", item.suffix, err)
				}
			}

			summary, err := storage.PurgeExpired(cutoff, cutoff, cutoff)
			if err != nil {
				t.Fatalf("purge expired: %v", err)
			}
			if summary.Imports != 1 || summary.Intervals != 1 || summary.TariffAssessments != 1 {
				t.Fatalf("purge summary = %+v", summary)
			}

			imports, err := storage.ListImports("home-a")
			if err != nil || len(imports) != 2 {
				t.Fatalf("retained imports = %+v err=%v", imports, err)
			}
			intervals, err := storage.ListIntervals("home-a", time.Time{}, time.Time{})
			if err != nil || len(intervals) != 2 {
				t.Fatalf("retained intervals = %+v err=%v", intervals, err)
			}
			assessments, err := storage.ListTariffAssessments("home-a")
			if err != nil || len(assessments) != 2 {
				t.Fatalf("retained assessments = %+v err=%v", assessments, err)
			}
			for _, item := range imports {
				if item.ID == "import-old" {
					t.Fatalf("expired import retained: %+v", item)
				}
			}
			for _, item := range intervals {
				if item.Source == "old" {
					t.Fatalf("expired interval retained: %+v", item)
				}
			}
			for _, item := range assessments {
				if item.ID == "assessment-old" {
					t.Fatalf("expired assessment retained: %+v", item)
				}
			}

			repeated, err := storage.PurgeExpired(cutoff, cutoff, cutoff)
			if err != nil {
				t.Fatalf("repeat purge: %v", err)
			}
			if repeated.Imports != 0 || repeated.Intervals != 0 || repeated.TariffAssessments != 0 {
				t.Fatalf("purge is not idempotent: %+v", repeated)
			}
		})
	}
}

type lifecycleCounts struct {
	imports     int
	intervals   int
	assessments int
	assets      int
	mappings    int
	maintenance int
	measures    int
}

func seedLifecycleTenant(t *testing.T, storage energy.Storage, tenant string, now time.Time) {
	t.Helper()
	freeStartedAt := now.AddDate(-1, 0, 0)
	profile := energy.DefaultProfile(tenant, now)
	profile.UnitID = "top-" + tenant
	profile.HouseholdName = "Zuhause " + tenant
	profile.OnboardingStep = 5
	profile.OnboardingComplete = true
	profile.RecommendationID = "observe"
	profile.RecommendationStatus = "deferred"
	profile.FreeStartedAt = &freeStartedAt
	if err := storage.SaveProfile(profile); err != nil {
		t.Fatalf("save %s profile: %v", tenant, err)
	}
	assetID := "asset-" + tenant
	if err := storage.UpsertAsset(energy.Asset{
		ID: assetID, TenantSlug: tenant, Kind: "pv", Name: "PV", Confirmed: true,
	}); err != nil {
		t.Fatalf("save %s asset: %v", tenant, err)
	}
	if err := storage.UpsertMapping(energy.EntityMapping{
		ID: "mapping-" + tenant, TenantSlug: tenant, EntityID: "sensor." + tenant,
		AssetID: assetID, Metric: energy.MetricPVPower, DisplayName: "PV", Unit: "W", Confirmed: true,
	}); err != nil {
		t.Fatalf("save %s mapping: %v", tenant, err)
	}
	if inserted, err := storage.PutImport(energy.ImportRecord{
		ID: "import-" + tenant, TenantSlug: tenant, Filename: tenant + ".csv",
		SHA256: "sha-" + tenant, Format: "test", Payload: []byte("private"),
		ImportedAt: now,
	}, []energy.Interval{{
		TenantSlug: tenant, StartsAt: now, Duration: 15 * time.Minute,
		ImportKWh: 1, AverageKW: 4, Quality: energy.QualityMeasured, Source: "smart-meter",
	}}); err != nil || !inserted {
		t.Fatalf("save %s import: inserted=%v err=%v", tenant, inserted, err)
	}
	if err := storage.UpsertMaintenance(energy.MaintenancePlan{
		ID: "maintenance-" + tenant, TenantSlug: tenant, AssetID: assetID,
		Title: "PV prüfen", IntervalMonths: 12, NextDueAt: now.AddDate(1, 0, 0), Active: true,
	}); err != nil {
		t.Fatalf("save %s maintenance: %v", tenant, err)
	}
	if err := storage.SaveTariffAssessment(energy.TariffAssessment{
		ID: "assessment-" + tenant, TenantSlug: tenant, AssessmentMonth: "2026-07",
		ProfileID: "at-power", ProfileVersion: "draft", ProfileStatus: "draft",
		PeakKW: 8, DataQuality: energy.QualityMeasured, CreatedAt: now,
	}); err != nil {
		t.Fatalf("save %s assessment: %v", tenant, err)
	}
	beforeFrom := now.AddDate(0, -1, 0)
	beforeTo := beforeFrom.Add(7 * 24 * time.Hour)
	afterFrom := now
	afterTo := now.Add(7 * 24 * time.Hour)
	beforePeak, afterPeak := 9.2, 6.4
	if err := storage.UpsertMeasure(energy.Measure{
		ID: "measure-" + tenant, TenantSlug: tenant, IssueID: "issue-" + tenant,
		RecommendationID: "observe", Title: "Peak glätten", Status: energy.MeasureCompleted,
		BeforeFrom: &beforeFrom, BeforeTo: &beforeTo, AfterFrom: &afterFrom, AfterTo: &afterTo,
		BeforePeakKW: &beforePeak, AfterPeakKW: &afterPeak,
		BeforeQuality: energy.QualityMeasured, AfterQuality: energy.QualityMeasured,
	}); err != nil {
		t.Fatalf("save %s measure: %v", tenant, err)
	}
}

func assertLifecycleCounts(t *testing.T, storage energy.Storage, tenant string, want lifecycleCounts) {
	t.Helper()
	imports, err := storage.ListImports(tenant)
	if err != nil {
		t.Fatalf("%s imports: %v", tenant, err)
	}
	intervals, err := storage.ListIntervals(tenant, time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("%s intervals: %v", tenant, err)
	}
	assessments, err := storage.ListTariffAssessments(tenant)
	if err != nil {
		t.Fatalf("%s assessments: %v", tenant, err)
	}
	assets, err := storage.ListAssets(tenant)
	if err != nil {
		t.Fatalf("%s assets: %v", tenant, err)
	}
	mappings, err := storage.ListMappings(tenant)
	if err != nil {
		t.Fatalf("%s mappings: %v", tenant, err)
	}
	maintenance, err := storage.ListMaintenance(tenant)
	if err != nil {
		t.Fatalf("%s maintenance: %v", tenant, err)
	}
	measures, err := storage.ListMeasures(tenant)
	if err != nil {
		t.Fatalf("%s measures: %v", tenant, err)
	}
	got := lifecycleCounts{
		imports: len(imports), intervals: len(intervals), assessments: len(assessments),
		assets: len(assets), mappings: len(mappings), maintenance: len(maintenance), measures: len(measures),
	}
	if got != want {
		t.Fatalf("%s lifecycle counts = %+v, want %+v", tenant, got, want)
	}
}
