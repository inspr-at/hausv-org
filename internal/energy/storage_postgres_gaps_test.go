package energy_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/energy"
)

func postgresGapStoreFactories() map[string]func(*testing.T) energy.Storage {
	return map[string]func(*testing.T) energy.Storage{
		"memory": func(*testing.T) energy.Storage {
			return energy.NewMemoryStore()
		},
		"sql": func(t *testing.T) energy.Storage {
			t.Helper()
			return openEnergyStore(t)
		},
	}
}

func TestUpdateAssetPrioritiesStorageParity(t *testing.T) {
	for name, factory := range postgresGapStoreFactories() {
		t.Run(name, func(t *testing.T) {
			storage := factory(t)
			for _, asset := range []energy.Asset{
				{ID: "grid", TenantSlug: "home-a", Kind: "grid", Name: "Netz", Metadata: map[string]string{"source": "test"}},
				{ID: "pv", TenantSlug: "home-a", Kind: "pv", Name: "PV"},
				{ID: "battery", TenantSlug: "home-a", Kind: "battery", Name: "Batterie"},
			} {
				if err := storage.UpsertAsset(asset); err != nil {
					t.Fatalf("seed asset %s: %v", asset.ID, err)
				}
			}

			updated, err := storage.UpdateAssetPriorities(" HOME-A ", []string{"battery", "grid", "pv"})
			if err != nil || !updated {
				t.Fatalf("update priorities: updated=%v err=%v", updated, err)
			}
			assertAssetPriorities(t, storage, map[string]string{
				"battery": "1",
				"grid":    "2",
				"pv":      "3",
			})

			updated, err = storage.UpdateAssetPriorities("home-a", []string{"grid", "missing"})
			if err != nil || updated {
				t.Fatalf("missing asset must reject the whole order: updated=%v err=%v", updated, err)
			}
			assertAssetPriorities(t, storage, map[string]string{
				"battery": "1",
				"grid":    "2",
				"pv":      "3",
			})

			assets, err := storage.ListAssets("home-a")
			if err != nil {
				t.Fatalf("list assets after priority update: %v", err)
			}
			for _, asset := range assets {
				if asset.ID == "grid" && asset.Metadata["source"] != "test" {
					t.Fatalf("priority update discarded unrelated metadata: %+v", asset.Metadata)
				}
			}
		})
	}
}

func assertAssetPriorities(t *testing.T, storage energy.Storage, want map[string]string) {
	t.Helper()
	assets, err := storage.ListAssets("home-a")
	if err != nil {
		t.Fatalf("list assets: %v", err)
	}
	if len(assets) != len(want) {
		t.Fatalf("assets = %+v, want %d", assets, len(want))
	}
	for _, asset := range assets {
		if got := asset.Metadata["priority"]; got != want[asset.ID] {
			t.Fatalf("asset %s priority = %q, want %q", asset.ID, got, want[asset.ID])
		}
	}
}

func TestDeleteMappingStorageParity(t *testing.T) {
	for name, factory := range postgresGapStoreFactories() {
		t.Run(name, func(t *testing.T) {
			storage := factory(t)
			for _, mapping := range []energy.EntityMapping{
				{ID: "mapping-a", TenantSlug: "home-a", EntityID: "sensor.grid_a", DisplayName: "Netz A"},
				{ID: "mapping-b", TenantSlug: "home-b", EntityID: "sensor.grid_b", DisplayName: "Netz B"},
			} {
				if err := storage.UpsertMapping(mapping); err != nil {
					t.Fatalf("seed mapping %s: %v", mapping.ID, err)
				}
			}

			deleted, err := storage.DeleteMapping(" HOME-A ", " mapping-a ")
			if err != nil || !deleted {
				t.Fatalf("delete mapping: deleted=%v err=%v", deleted, err)
			}
			if mappings, err := storage.ListMappings("home-a"); err != nil || len(mappings) != 0 {
				t.Fatalf("deleted mapping still visible: %+v err=%v", mappings, err)
			}
			if mappings, err := storage.ListMappings("home-b"); err != nil || len(mappings) != 1 || mappings[0].ID != "mapping-b" {
				t.Fatalf("other tenant mapping changed: %+v err=%v", mappings, err)
			}
			deleted, err = storage.DeleteMapping("home-a", "mapping-a")
			if err != nil || deleted {
				t.Fatalf("second delete must report absence: deleted=%v err=%v", deleted, err)
			}
		})
	}
}

func TestListImportsForExportStorageParity(t *testing.T) {
	base := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	for name, factory := range postgresGapStoreFactories() {
		t.Run(name, func(t *testing.T) {
			storage := factory(t)
			for _, record := range []energy.ImportRecord{
				{ID: "older", TenantSlug: "home-a", Filename: "older.csv", SHA256: "sha-older", Format: "csv", Payload: []byte("older payload"), ImportedAt: base},
				{ID: "newer", TenantSlug: "home-a", Filename: "newer.csv", SHA256: "sha-newer", Format: "csv", Payload: []byte("newer payload"), ImportedAt: base.Add(time.Hour)},
				{ID: "foreign", TenantSlug: "home-b", Filename: "foreign.csv", SHA256: "sha-foreign", Format: "csv", Payload: []byte("foreign payload"), ImportedAt: base.Add(2 * time.Hour)},
			} {
				inserted, err := storage.PutImport(record, nil)
				if err != nil || !inserted {
					t.Fatalf("seed import %s: inserted=%v err=%v", record.ID, inserted, err)
				}
			}

			listed, err := storage.ListImports("home-a")
			if err != nil || len(listed) != 2 {
				t.Fatalf("list imports: %+v err=%v", listed, err)
			}
			for _, record := range listed {
				if len(record.Payload) != 0 {
					t.Fatalf("ordinary import listing exposed payload for %s", record.ID)
				}
			}

			exported, err := storage.ListImportsForExport(" HOME-A ")
			if err != nil {
				t.Fatalf("list imports for export: %v", err)
			}
			if len(exported) != 2 || exported[0].ID != "newer" || exported[1].ID != "older" {
				t.Fatalf("export order or tenant scope = %+v", exported)
			}
			if !bytes.Equal(exported[0].Payload, []byte("newer payload")) || !bytes.Equal(exported[1].Payload, []byte("older payload")) {
				t.Fatalf("export payloads = %q, %q", exported[0].Payload, exported[1].Payload)
			}

			exported[0].Payload[0] = 'X'
			again, err := storage.ListImportsForExport("home-a")
			if err != nil || len(again) != 2 || !bytes.Equal(again[0].Payload, []byte("newer payload")) {
				t.Fatalf("caller mutated stored export payload: %+v err=%v", again, err)
			}
		})
	}
}

func TestDeleteMaintenanceStorageParity(t *testing.T) {
	now := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	for name, factory := range postgresGapStoreFactories() {
		t.Run(name, func(t *testing.T) {
			storage := factory(t)
			for _, tenant := range []string{"home-a", "home-b"} {
				assetID := "asset-" + tenant
				if err := storage.UpsertAsset(energy.Asset{ID: assetID, TenantSlug: tenant, Kind: "pv", Name: "PV"}); err != nil {
					t.Fatalf("seed %s asset: %v", tenant, err)
				}
				if err := storage.UpsertMaintenance(energy.MaintenancePlan{
					ID: "maintenance-" + tenant, TenantSlug: tenant, AssetID: assetID,
					Title: "Sichtprüfung", IntervalMonths: 12, NextDueAt: now.AddDate(1, 0, 0), Active: true,
				}); err != nil {
					t.Fatalf("seed %s maintenance: %v", tenant, err)
				}
			}

			deleted, err := storage.DeleteMaintenance(" HOME-A ", " maintenance-home-a ")
			if err != nil || !deleted {
				t.Fatalf("delete maintenance: deleted=%v err=%v", deleted, err)
			}
			if plans, err := storage.ListMaintenance("home-a"); err != nil || len(plans) != 0 {
				t.Fatalf("deleted maintenance still visible: %+v err=%v", plans, err)
			}
			if plans, err := storage.ListMaintenance("home-b"); err != nil || len(plans) != 1 || plans[0].ID != "maintenance-home-b" {
				t.Fatalf("other tenant maintenance changed: %+v err=%v", plans, err)
			}
			deleted, err = storage.DeleteMaintenance("home-a", "maintenance-home-a")
			if err != nil || deleted {
				t.Fatalf("second delete must report absence: deleted=%v err=%v", deleted, err)
			}
		})
	}
}
