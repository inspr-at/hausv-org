package demo

import (
	"path/filepath"
	"testing"

	"github.com/inspr-at/hausv-org/internal/dbtest"
	"github.com/inspr-at/hausv-org/internal/store"
)

func TestHAUSV619LoadWritesFixtureUnitsToUnitSink(t *testing.T) {
	database := dbtest.Open(t)
	units, err := store.NewUnitStore(filepath.Join(t.TempDir(), "units.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Load(t.Context(), database, filepath.Join("..", "..", "scripts", "demo", "seed"), SeedOptions{Reset: true, Units: units}); err != nil {
		t.Fatal(err)
	}
	identities, err := store.EnsureTenantIdentities(t.Context(), database, []store.TenantIdentity{{Slug: "janusbergweg-123"}})
	if err != nil {
		t.Fatal(err)
	}
	repository, ok := store.BindUnitRepository(units, identities["janusbergweg-123"].Ref())
	if !ok {
		t.Fatal("bind unit repository")
	}
	items := repository.List()
	if got, want := len(items), 24; got != want {
		t.Fatalf("Janusbergweg units = %d, want %d", got, want)
	}
	top1 := repository.MembersForUnit("Top 1")
	if !top1.Found || len(top1.Owners) != 1 || top1.Owners[0] != "alina.eigentuemer@musterstadt.example" {
		t.Fatalf("Top 1 owners = %#v", top1)
	}
	top3 := repository.MembersForUnit("Top 3")
	if !top3.Found || len(top3.Renters) != 1 || top3.Renters[0] != "matthias.mieter@musterstadt.example" {
		t.Fatalf("Top 3 renters = %#v", top3)
	}
	parking := 0
	for _, item := range items {
		if item.UnitType == store.UnitTypeParking {
			parking++
		}
	}
	if parking != 6 {
		t.Fatalf("parking spaces = %d, want 6", parking)
	}
}
