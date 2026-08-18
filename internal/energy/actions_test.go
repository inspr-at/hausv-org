package energy_test

import (
	"errors"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/energy"
)

func TestHomeProfileUnitScopeStorageParity(t *testing.T) {
	now := time.Date(2026, 7, 29, 15, 0, 0, 0, time.UTC)
	factories := map[string]func(*testing.T) energy.Storage{
		"memory": func(t *testing.T) energy.Storage {
			return energy.NewMemoryStore()
		},
		"sql": func(t *testing.T) energy.Storage {
			return openEnergyStore(t)
		},
	}
	for name, factory := range factories {
		t.Run(name, func(t *testing.T) {
			storage := factory(t)
			profile := energy.DefaultProfile("home-a", now)
			profile.HouseholdName = "Dachwohnung"
			profile.UnitID = " Wohnung Günter/1.2 "
			if err := storage.SaveProfile(profile); err != nil {
				t.Fatalf("save profile: %v", err)
			}

			stored, ok, err := storage.Profile("home-a")
			if err != nil || !ok {
				t.Fatalf("load profile: ok=%v err=%v", ok, err)
			}
			if stored.UnitID != "wohnung-günter-1.2" || stored.HouseholdName != "Dachwohnung" {
				t.Fatalf("stored profile = %+v", stored)
			}

			stored.UnitID = ""
			if err := storage.SaveProfile(stored); err != nil {
				t.Fatalf("clear unit scope: %v", err)
			}
			cleared, ok, err := storage.Profile("home-a")
			if err != nil || !ok || cleared.UnitID != "" {
				t.Fatalf("cleared profile = %+v ok=%v err=%v", cleared, ok, err)
			}
		})
	}
}

func TestEnergyActionStorageParityAndHistory(t *testing.T) {
	now := time.Date(2026, 7, 28, 8, 0, 0, 0, time.UTC)
	factories := map[string]func(*testing.T) energy.Storage{
		"memory": func(t *testing.T) energy.Storage {
			return energy.NewMemoryStore()
		},
		"sql": func(t *testing.T) energy.Storage {
			return openEnergyStore(t)
		},
	}
	for name, factory := range factories {
		t.Run(name, func(t *testing.T) {
			store := factory(t)
			for _, tenant := range []string{"home-a", "home-b"} {
				if err := store.SaveProfile(energy.DefaultProfile(tenant, now)); err != nil {
					t.Fatalf("save profile: %v", err)
				}
				if err := store.UpsertAsset(energy.Asset{
					ID: energy.StableAssetID(tenant, "pv"), TenantSlug: tenant, Kind: "pv", Name: "PV", Confirmed: true,
				}); err != nil {
					t.Fatalf("save %s asset: %v", tenant, err)
				}
			}
			// The error is asserted, not discarded: a query that fails outright
			// returns an empty slice, and "home-a has no foreign asset" is then
			// true for the wrong reason.
			got, err := store.ListAssets("home-a")
			if err != nil {
				t.Fatalf("list home-a assets: %v", err)
			}
			if len(got) != 1 || got[0].ID != energy.StableAssetID("home-a", "pv") {
				t.Fatalf("home-a assets = %+v", got)
			}
			mapping := energy.EntityMapping{
				ID: "mapping-a", TenantSlug: "home-a", EntityID: "sensor.pv_power",
				AssetID: energy.StableAssetID("home-a", "pv"), Metric: energy.MetricPVPower,
				DisplayName: "PV-Leistung", Unit: "kW", Confirmed: true,
			}
			if err := store.UpsertMapping(mapping); err != nil {
				t.Fatalf("save asset mapping: %v", err)
			}
			foreign := mapping
			foreign.ID = "mapping-foreign"
			foreign.EntityID = "sensor.foreign_pv_power"
			foreign.AssetID = energy.StableAssetID("home-b", "pv")
			// Named, not merely non-nil: any dialect failure on the way to the
			// guard satisfied the old assertion identically, and this is the
			// only test covering the mapping/asset cross-tenant guard.
			if err := store.UpsertMapping(foreign); !errors.Is(err, energy.ErrMappingAssetForeign) {
				t.Fatalf("cross-tenant asset mapping must be refused with %v, got: %v",
					energy.ErrMappingAssetForeign, err)
			}
			if mappings, err := store.ListMappings("home-a"); err != nil || len(mappings) != 1 ||
				mappings[0].AssetID != energy.StableAssetID("home-a", "pv") {
				t.Fatalf("asset mappings = %+v err=%v", mappings, err)
			}

			completed := now.Add(-24 * time.Hour)
			plan := energy.MaintenancePlan{
				ID: "maintenance-a", TenantSlug: "home-a", AssetID: energy.StableAssetID("home-a", "pv"),
				Title: "PV-Sichtprüfung", IntervalMonths: 12, LastCompletedAt: &completed,
				NextDueAt: now.Add(14 * 24 * time.Hour), Active: true,
			}
			if err := store.UpsertMaintenance(plan); err != nil {
				t.Fatalf("save maintenance: %v", err)
			}
			plans, err := store.ListMaintenance("home-a")
			if err != nil || len(plans) != 1 || plans[0].ID != plan.ID {
				t.Fatalf("maintenance = %+v err=%v", plans, err)
			}
			recommendation, ok := energy.MaintenanceRecommendation(now, plans)
			if !ok || recommendation.State != "now" || recommendation.Title != "PV-Sichtprüfung" {
				t.Fatalf("maintenance recommendation = %+v ok=%v", recommendation, ok)
			}

			for index, version := range []string{"draft-2027-v1", "draft-2027-v2"} {
				if err := store.SaveTariffAssessment(energy.TariffAssessment{
					ID: "tariff-" + version, TenantSlug: "home-a", AssessmentMonth: "2026-07",
					ProfileID: "at-grid-power", ProfileVersion: version, ProfileStatus: "draft",
					SourceURL: "https://example.invalid/rules", PeakKW: 8.25 + float64(index),
					BilledKW: 8.25 + float64(index), AnnualPowerEUR: 99 + float64(index),
					DataQuality: energy.QualityMeasured, CreatedAt: now.Add(time.Duration(index) * time.Minute),
				}); err != nil {
					t.Fatalf("save tariff assessment: %v", err)
				}
			}
			assessments, err := store.ListTariffAssessments("home-a")
			if err != nil || len(assessments) != 2 || assessments[0].ProfileVersion != "draft-2027-v2" ||
				assessments[1].ProfileVersion != "draft-2027-v1" {
				t.Fatalf("tariff history = %+v err=%v", assessments, err)
			}

			appointment := now.Add(48 * time.Hour)
			beforeFrom := now.AddDate(0, -1, 0)
			beforeTo := beforeFrom.Add(7 * 24 * time.Hour)
			afterFrom := now
			afterTo := now.Add(7 * 24 * time.Hour)
			beforePeak, afterPeak := 9.4, 6.7
			if err := store.UpsertMeasure(energy.Measure{
				ID: "measure-a", TenantSlug: "home-a", IssueID: "issue-a", RecommendationID: "peak",
				Title: "Lastspitze glätten", Status: energy.MeasureCompleted, ContactID: "contact-a",
				SharedFields: []string{"measurements", "inventory"}, OfferNote: "Angebot geprüft",
				AppointmentAt: &appointment, WorkNote: "Wallbox begrenzt", CompletedAt: &now,
				EvidenceNote: "Messung geprüft", BeforeFrom: &beforeFrom, BeforeTo: &beforeTo,
				AfterFrom: &afterFrom, AfterTo: &afterTo, BeforePeakKW: &beforePeak, AfterPeakKW: &afterPeak,
				BeforeQuality: energy.QualityMeasured, AfterQuality: energy.QualityEstimated,
			}); err != nil {
				t.Fatalf("save measure: %v", err)
			}
			measure, ok, err := store.GetMeasure("home-a", "measure-a")
			if err != nil || !ok || measure.AppointmentAt == nil || measure.BeforeFrom == nil || measure.AfterTo == nil ||
				measure.BeforePeakKW == nil || *measure.BeforePeakKW != 9.4 || len(measure.SharedFields) != 2 {
				t.Fatalf("measure = %+v ok=%v err=%v", measure, ok, err)
			}

			if deleted, err := store.DeleteAsset("home-a", energy.StableAssetID("home-a", "pv")); err != nil || !deleted {
				t.Fatalf("delete asset: deleted=%v err=%v", deleted, err)
			}
			if plans, err := store.ListMaintenance("home-a"); err != nil || len(plans) != 0 {
				t.Fatalf("maintenance should cascade with asset: %+v err=%v", plans, err)
			}
			if mappings, err := store.ListMappings("home-a"); err != nil || len(mappings) != 1 || mappings[0].AssetID != "" {
				t.Fatalf("deleting asset should keep measurement and clear link: %+v err=%v", mappings, err)
			}
			if assets, err := store.ListAssets("home-b"); err != nil || len(assets) != 1 {
				t.Fatalf("home-b asset affected by home-a delete: %+v err=%v", assets, err)
			}
		})
	}
}

func TestPeakForRangeReportsQuality(t *testing.T) {
	peak, quality := energy.PeakForRange([]energy.Interval{
		{AverageKW: 4.2, Quality: energy.QualityMeasured},
		{AverageKW: 5.3, Quality: energy.QualityEstimated},
	})
	if peak == nil || *peak != 5.3 || quality != energy.QualityEstimated {
		t.Fatalf("peak=%v quality=%q", peak, quality)
	}
	if peak, quality := energy.PeakForRange([]energy.Interval{{AverageKW: 7, Quality: energy.QualityGap}}); peak != nil || quality != energy.QualityUnavailable {
		t.Fatalf("gap-only peak=%v quality=%q", peak, quality)
	}
}
