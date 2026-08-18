package energy_test

import (
	"errors"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/dbtest"
	"github.com/inspr-at/hausv-org/internal/energy"
)

// Vorarbeit zu HAUSV-422: sobald zusätzliche Verbraucher zufällige IDs
// bekommen, trägt die globale Eindeutigkeit der Asset-ID die Mandantentrennung
// mit. Beide Speicher müssen sich dabei gleich verhalten — in SQLite ist `id`
// Primärschlüssel, der Memory-Store schlüsselte dagegen nach (Haus, ID) und
// ließ dieselbe ID für zwei Häuser stillschweigend zu.
func TestAssetIDIsGloballyUniqueInBothStoresHAUSV422(t *testing.T) {
	now := time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC)

	stores := map[string]func(*testing.T) energy.Storage{
		"memory": func(*testing.T) energy.Storage { return energy.NewMemoryStore() },
		"sql": func(t *testing.T) energy.Storage {
			t.Helper()
			return energy.NewSQLStore(dbtest.Open(t))
		},
	}

	for name, factory := range stores {
		t.Run(name, func(t *testing.T) {
			store := factory(t)
			for _, tenant := range []string{"haus-a", "haus-b"} {
				if err := store.SaveProfile(energy.DefaultProfile(tenant, now)); err != nil {
					t.Fatalf("Profil %s: %v", tenant, err)
				}
			}
			if err := store.UpsertAsset(energy.Asset{
				ID: "asset-geteilt", TenantSlug: "haus-a", Kind: "pv", Confirmed: true,
			}); err != nil {
				t.Fatalf("erstes Asset: %v", err)
			}

			// Dieselbe ID für ein anderes Haus muss abgewiesen werden — sonst
			// könnte ein fremdes Haus ein Asset überschreiben.
			//
			// Die Prüfung nennt den Fehler beim Namen. Vorher genügte
			// irgendein Fehler, und genau das machte den Test wertlos: auf
			// PostgreSQL scheiterte dieselbe Anweisung am Dialekt (42P10,
			// ON CONFLICT ohne passenden Unique-Index), und der Test hätte die
			// Mandantentrennung als bewiesen gemeldet, obwohl sie dort gar
			// nicht existierte.
			err := store.UpsertAsset(energy.Asset{
				ID: "asset-geteilt", TenantSlug: "haus-b", Kind: "pv", Confirmed: true,
			})
			if !errors.Is(err, energy.ErrAssetIDTaken) {
				t.Fatalf("dieselbe Asset-ID für zwei Häuser muss mit %v abgewiesen werden, war: %v",
					energy.ErrAssetIDTaken, err)
			}

			a, err := store.ListAssets("haus-a")
			if err != nil {
				t.Fatalf("Assets von haus-a lesen: %v", err)
			}
			b, err := store.ListAssets("haus-b")
			if err != nil {
				t.Fatalf("Assets von haus-b lesen: %v", err)
			}
			// Beide Hälften: haus-b hat nichts bekommen UND haus-a hat sein
			// Asset behalten. Ohne die zweite wäre eine fehlgeschlagene Abfrage
			// von echter Trennung nicht zu unterscheiden.
			if len(a) != 1 || a[0].ID != "asset-geteilt" {
				t.Fatalf("haus-a sollte genau sein Asset behalten, hatte %+v", a)
			}
			if len(b) != 0 {
				t.Fatalf("haus-b darf kein Asset haben, hatte %d", len(b))
			}
		})
	}
}

// Zwei Verbraucher derselben Art sind heute nicht anlegbar, weil die ID aus
// (Haus, Art) abgeleitet wird — der zweite überschreibt den ersten. Genau das
// ist der Kern von HAUSV-422; dieser Test hält den Ist-Zustand fest, damit die
// spätere Änderung nachweislich etwas bewirkt.
func TestSecondAssetOfSameKindOverwritesTodayHAUSV422(t *testing.T) {
	store := energy.NewMemoryStore()
	now := time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC)
	if err := store.SaveProfile(energy.DefaultProfile("haus", now)); err != nil {
		t.Fatalf("Profil: %v", err)
	}
	for _, name := range []string{"Sauna", "Infrarotkabine"} {
		if err := store.UpsertAsset(energy.Asset{
			ID:         energy.StableAssetID("haus", "sauna"),
			TenantSlug: "haus",
			Kind:       "sauna",
			Name:       name,
			Confirmed:  true,
		}); err != nil {
			t.Fatalf("Asset %s: %v", name, err)
		}
	}
	assets, _ := store.ListAssets("haus")
	if len(assets) != 1 {
		t.Fatalf("Ist-Zustand erwartet: ein Asset überschreibt das andere, waren %d", len(assets))
	}
	if assets[0].Name != "Infrarotkabine" {
		t.Fatalf("der zweite Verbraucher sollte den ersten überschrieben haben, war %q", assets[0].Name)
	}
}
