package energy_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/markus-barta/hausv-org/internal/db"
	"github.com/markus-barta/hausv-org/internal/energy"
)

func TestHomeScopedStoresIsolateHomesWithinTenant(t *testing.T) {
	stores := map[string]func(*testing.T) energy.Storage{
		"memory": func(*testing.T) energy.Storage { return energy.NewMemoryStore() },
		"sqlite": func(t *testing.T) energy.Storage {
			database, err := db.Open(filepath.Join(t.TempDir(), "energy.db"))
			if err != nil {
				t.Fatalf("open db: %v", err)
			}
			t.Cleanup(func() { database.Close() })
			return energy.NewSQLStore(database)
		},
	}

	for name, newStore := range stores {
		t.Run(name, func(t *testing.T) {
			root := newStore(t)
			now := time.Date(2026, 8, 13, 9, 0, 0, 0, time.UTC)
			first := root.ForHome("top-11")
			second := root.ForHome("top-12")
			for _, item := range []struct {
				store energy.Storage
				home  string
				unit  string
				asset string
			}{
				{first, "top-11", "unit-11", "asset-top-11"},
				{second, "top-12", "unit-12", "asset-top-12"},
			} {
				profile := energy.DefaultProfileForHome("jhw22", item.home, now)
				profile.UnitID = item.unit
				profile.HouseholdName = item.home
				if err := item.store.SaveProfile(profile); err != nil {
					t.Fatalf("save profile %s: %v", item.home, err)
				}
				if err := item.store.UpsertAsset(energy.Asset{ID: item.asset, TenantSlug: "jhw22", Kind: "ev", Name: item.home}); err != nil {
					t.Fatalf("save asset %s: %v", item.home, err)
				}
				if err := item.store.UpsertMapping(energy.EntityMapping{ID: "mapping-" + item.home, TenantSlug: "jhw22", AssetID: item.asset, EntityID: "sensor.shared_name", Metric: "load-power"}); err != nil {
					t.Fatalf("save mapping %s: %v", item.home, err)
				}
				if err := item.store.PutInterval(energy.Interval{TenantSlug: "jhw22", StartsAt: now, ImportKWh: 1, AverageKW: 4, Source: "home-assistant"}); err != nil {
					t.Fatalf("save interval %s: %v", item.home, err)
				}
			}

			profiles, err := root.ListProfiles("jhw22")
			if err != nil || len(profiles) != 2 {
				t.Fatalf("profiles = %d, err=%v", len(profiles), err)
			}
			for _, item := range []struct {
				store energy.Storage
				home  string
				asset string
			}{
				{first, "top-11", "asset-top-11"},
				{second, "top-12", "asset-top-12"},
			} {
				assets, _ := item.store.ListAssets("jhw22")
				mappings, _ := item.store.ListMappings("jhw22")
				intervals, _ := item.store.ListIntervals("jhw22", now.Add(-time.Minute), now.Add(time.Minute))
				if len(assets) != 1 || assets[0].ID != item.asset || assets[0].HomeKey != item.home {
					t.Fatalf("%s assets leaked: %#v", item.home, assets)
				}
				if len(mappings) != 1 || mappings[0].HomeKey != item.home || mappings[0].EntityID != "sensor.shared_name" {
					t.Fatalf("%s mappings leaked: %#v", item.home, mappings)
				}
				if len(intervals) != 1 || intervals[0].HomeKey != item.home {
					t.Fatalf("%s intervals leaked: %#v", item.home, intervals)
				}
			}

			if _, err := first.DeleteProfile("jhw22"); err != nil {
				t.Fatalf("delete first home: %v", err)
			}
			if assets, _ := second.ListAssets("jhw22"); len(assets) != 1 || assets[0].ID != "asset-top-12" {
				t.Fatalf("deleting first home affected second: %#v", assets)
			}
		})
	}
}

func TestDefaultStoreKeepsSingleHomeBehavior(t *testing.T) {
	store := energy.NewMemoryStore()
	if err := store.SaveProfile(energy.DefaultProfile("jhw22", time.Now())); err != nil {
		t.Fatalf("save profile: %v", err)
	}
	profile, ok, err := store.Profile("jhw22")
	if err != nil || !ok || profile.HomeKey != energy.DefaultHomeKey {
		t.Fatalf("default profile = %#v, ok=%v err=%v", profile, ok, err)
	}
}
