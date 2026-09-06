package store

import (
	"context"
	"testing"

	"github.com/inspr-at/hausv-org/internal/dbtest"
)

func TestOrganisationRoundTripsNameContactAndHouses(t *testing.T) {
	database := dbtest.Open(t)
	repo := BindOrganisationRepository(database, "musterstadt")
	ctx := context.Background()

	if _, ok, err := repo.Get(ctx); err != nil || ok {
		t.Fatalf("empty store reported ok=%v err=%v, want false/nil", ok, err)
	}

	err := repo.Save(ctx, Organisation{
		Name:         "Hausverwaltung Musterstadt GmbH",
		ContactName:  "Vera Verwalter",
		ContactEmail: "  Buero@Musterstadt.example ",
		ContactPhone: "+43 316 123456",
		Houses:       []string{"janusbergweg-123", "muenzgrabenstrasse-9", "janusbergweg-123", " "},
	})
	if err != nil {
		t.Fatal(err)
	}

	got, ok, err := repo.Get(ctx)
	if err != nil || !ok {
		t.Fatalf("Get after Save = ok %v, err %v", ok, err)
	}
	if got.Key != "musterstadt" || got.Name != "Hausverwaltung Musterstadt GmbH" {
		t.Fatalf("identity = %+v", got)
	}
	if got.ContactEmail != "buero@musterstadt.example" {
		t.Fatalf("contact email not normalized: %q", got.ContactEmail)
	}
	if len(got.Houses) != 2 || got.Houses[0] != "janusbergweg-123" || got.Houses[1] != "muenzgrabenstrasse-9" {
		t.Fatalf("houses = %v, want the two slugs sorted and de-duplicated", got.Houses)
	}
	if got.UpdatedAt.IsZero() {
		t.Fatal("UpdatedAt was not stamped")
	}
}

func TestOrganisationSetHousesKeepsEditedContactData(t *testing.T) {
	database := dbtest.Open(t)
	repo := BindOrganisationRepository(database, "musterstadt")
	ctx := context.Background()
	if err := repo.Save(ctx, Organisation{
		Name:        "Hausverwaltung Musterstadt GmbH",
		ContactName: "Vera Verwalter",
		Houses:      []string{"janusbergweg-123"},
	}); err != nil {
		t.Fatal(err)
	}

	// This is what a boot-time reconcile does. It must not reset the fields a
	// person edited in the app.
	if err := repo.SetHouses(ctx, []string{"muenzgrabenstrasse-9", "janusbergweg-123"}); err != nil {
		t.Fatal(err)
	}

	got, ok, err := repo.Get(ctx)
	if err != nil || !ok {
		t.Fatalf("Get = ok %v, err %v", ok, err)
	}
	if got.ContactName != "Vera Verwalter" || got.Name != "Hausverwaltung Musterstadt GmbH" {
		t.Fatalf("SetHouses changed organisation data: %+v", got)
	}
	if len(got.Houses) != 2 {
		t.Fatalf("houses = %v", got.Houses)
	}
}

func TestOrganisationRejectsEmptyIdentity(t *testing.T) {
	database := dbtest.Open(t)
	ctx := context.Background()
	if err := BindOrganisationRepository(database, "musterstadt").Save(ctx, Organisation{Name: "  "}); err == nil {
		t.Fatal("a nameless organisation was accepted")
	}
	if err := BindOrganisationRepository(database, " ").Save(ctx, Organisation{Name: "Egal"}); err == nil {
		t.Fatal("an unkeyed repository accepted a write")
	}
}

func TestOrganisationsAreIsolatedFromEachOther(t *testing.T) {
	database := dbtest.Open(t)
	ctx := context.Background()
	if err := BindOrganisationRepository(database, "musterstadt").Save(ctx, Organisation{
		Name:   "Hausverwaltung Musterstadt GmbH",
		Houses: []string{"janusbergweg-123"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := BindOrganisationRepository(database, "seestadt").Save(ctx, Organisation{
		Name:   "Seestadt Immobilien",
		Houses: []string{"seepromenade-2"},
	}); err != nil {
		t.Fatal(err)
	}

	got, ok, err := BindOrganisationRepository(database, "seestadt").Get(ctx)
	if err != nil || !ok {
		t.Fatalf("Get = ok %v, err %v", ok, err)
	}
	if got.Name != "Seestadt Immobilien" || len(got.Houses) != 1 || got.Houses[0] != "seepromenade-2" {
		t.Fatalf("second organisation sees the wrong rows: %+v", got)
	}
}
