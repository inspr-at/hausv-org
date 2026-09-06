package store

import (
	"context"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/dbtest"
)

func TestOrganisationMemberRoundTripsRoleAndGrantRecord(t *testing.T) {
	database := dbtest.Open(t)
	repo := BindOrganisationMemberRepository(database, "musterstadt")
	ctx := context.Background()

	if _, ok, err := repo.Get(ctx, "paul@example.com"); err != nil || ok {
		t.Fatalf("empty store reported ok=%v err=%v", ok, err)
	}

	err := repo.Save(ctx, OrganisationMember{
		Email:   "  Paul@Example.com ",
		Role:    OrganisationRoleClerk,
		Granted: map[string]string{"janusbergweg-123": "", "muenzgrabenstrasse-9": "bewohner"},
	})
	if err != nil {
		t.Fatal(err)
	}

	got, ok, err := repo.Get(ctx, "paul@example.com")
	if err != nil || !ok {
		t.Fatalf("Get = ok %v, err %v", ok, err)
	}
	if got.Email != "paul@example.com" || got.Role != OrganisationRoleClerk {
		t.Fatalf("member = %+v", got)
	}
	if got.Granted["janusbergweg-123"] != "" || got.Granted["muenzgrabenstrasse-9"] != "bewohner" {
		t.Fatalf("grant record = %v", got.Granted)
	}
	if got.CreatedAt.IsZero() || got.UpdatedAt.IsZero() {
		t.Fatalf("timestamps missing: %+v", got)
	}
}

func TestOrganisationMemberUnknownRoleFallsBackToClerk(t *testing.T) {
	database := dbtest.Open(t)
	repo := BindOrganisationMemberRepository(database, "musterstadt")
	ctx := context.Background()
	if err := repo.Save(ctx, OrganisationMember{Email: "eve@example.com", Role: "superuser"}); err != nil {
		t.Fatal(err)
	}
	got, _, err := repo.Get(ctx, "eve@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got.Role != OrganisationRoleClerk {
		t.Fatalf("unknown role became %q, want the clerk role", got.Role)
	}
}

func TestOrganisationMemberRejectsBrokenIdentity(t *testing.T) {
	database := dbtest.Open(t)
	repo := BindOrganisationMemberRepository(database, "musterstadt")
	ctx := context.Background()
	if err := repo.Save(ctx, OrganisationMember{Email: "   ", Role: OrganisationRoleAdmin}); err == nil {
		t.Fatal("a member without an address was accepted")
	}
	if err := repo.Save(ctx, OrganisationMember{Email: "kein-at-zeichen", Role: OrganisationRoleAdmin}); err == nil {
		t.Fatal("a member without an e-mail address was accepted")
	}
}

func TestOrganisationMemberListDeleteAndIsolation(t *testing.T) {
	database := dbtest.Open(t)
	ctx := context.Background()
	musterstadt := BindOrganisationMemberRepository(database, "musterstadt")
	seestadt := BindOrganisationMemberRepository(database, "seestadt")

	for _, email := range []string{"vera@example.com", "paul@example.com"} {
		if err := musterstadt.Save(ctx, OrganisationMember{Email: email, Role: OrganisationRoleAdmin, CreatedAt: time.Now().UTC()}); err != nil {
			t.Fatal(err)
		}
	}
	if err := seestadt.Save(ctx, OrganisationMember{Email: "sonja@example.com", Role: OrganisationRoleClerk}); err != nil {
		t.Fatal(err)
	}

	members, err := musterstadt.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 2 || members[0].Email != "paul@example.com" || members[1].Email != "vera@example.com" {
		t.Fatalf("members = %+v, want both sorted by address", members)
	}
	others, err := seestadt.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(others) != 1 || others[0].Email != "sonja@example.com" {
		t.Fatalf("the second organisation sees the wrong rows: %+v", others)
	}

	removed, err := musterstadt.Delete(ctx, "paul@example.com")
	if err != nil || !removed {
		t.Fatalf("Delete = %v, err %v", removed, err)
	}
	if removed, err := musterstadt.Delete(ctx, "paul@example.com"); err != nil || removed {
		t.Fatalf("second Delete = %v, err %v, want false", removed, err)
	}
	members, err = musterstadt.List(ctx)
	if err != nil || len(members) != 1 {
		t.Fatalf("after delete: %+v, err %v", members, err)
	}
}
