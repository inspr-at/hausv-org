package store

import (
	"testing"
	"time"
)

func hausv516DB(t *testing.T) *SQLIdentityStore {
	t.Helper()
	database := testDB(t)
	t.Cleanup(func() { database.Close() })
	return NewSQLIdentityStore(database)
}

func TestBoundAnnouncementRepositoryExcludesOtherTenants(t *testing.T) {
	identity := hausv516DB(t)
	demo, _ := BindAnnouncementRepository(NewSQLAnnouncementStore(identity.db), testTenantRef("demo"))
	other, _ := BindAnnouncementRepository(NewSQLAnnouncementStore(identity.db), testTenantRef("other"))
	_, _ = demo.Create(Announcement{TenantSlug: "other", Title: "Demo", Body: "x"})
	_, _ = other.Create(Announcement{TenantSlug: "demo", Title: "Other", Body: "x"})
	if got := demo.List(); len(got) != 1 || got[0].Title != "Demo" || got[0].TenantSlug != "demo" {
		t.Fatalf("demo announcements = %+v", got)
	}
}

func TestRepositoryBoundaryValidatesTenantIdentity(t *testing.T) {
	storage := NewSQLAnnouncementStore(testDB(t))
	if _, ok := BindAnnouncementRepository(storage, TenantRef{Slug: "demo"}); ok {
		t.Fatal("repository accepted an empty tenant id")
	}
	if _, ok := BindAnnouncementRepository(storage, TenantRef{ID: "not-a-tenant-id", Slug: "demo"}); ok {
		t.Fatal("repository accepted a malformed tenant id")
	}
	if _, ok := BindAnnouncementRepository(storage, TenantRef{ID: testTenantID("demo")}); ok {
		t.Fatal("repository accepted an empty tenant slug")
	}
	if _, ok := BindAnnouncementRepository(storage, testTenantRef("demo")); !ok {
		t.Fatal("repository rejected a valid tenant id")
	}
}

func TestBoundEventRepositoryExcludesOtherTenants(t *testing.T) {
	identity := hausv516DB(t)
	demo, _ := BindEventRepository(NewSQLEventStore(identity.db), testTenantRef("demo"))
	other, _ := BindEventRepository(NewSQLEventStore(identity.db), testTenantRef("other"))
	start := time.Now().Add(time.Hour)
	_, _ = demo.Create(HouseEvent{TenantSlug: "other", Title: "Demo", StartsAt: start})
	_, _ = other.Create(HouseEvent{TenantSlug: "demo", Title: "Other", StartsAt: start})
	if got := demo.List(); len(got) != 1 || got[0].Title != "Demo" || got[0].TenantSlug != "demo" {
		t.Fatalf("demo events = %+v", got)
	}
}

func TestBoundContactBookRepositoryExcludesOtherTenants(t *testing.T) {
	identity := hausv516DB(t)
	demo, _ := BindContactBookRepository(NewSQLContactBookStore(identity.db), testTenantRef("demo"))
	other, _ := BindContactBookRepository(NewSQLContactBookStore(identity.db), testTenantRef("other"))
	_, _, _ = demo.Upsert(ManagedContact{TenantSlug: "other", Kind: "Notdienst", Name: "Demo", Phone: "1", Active: true})
	_, _, _ = other.Upsert(ManagedContact{TenantSlug: "demo", Kind: "Notdienst", Name: "Other", Phone: "2", Active: true})
	if got := demo.List(true); len(got) != 1 || got[0].Name != "Demo" || got[0].TenantSlug != "demo" {
		t.Fatalf("demo contacts = %+v", got)
	}
}

func TestBoundHandoverRepositoryExcludesOtherTenants(t *testing.T) {
	identity := hausv516DB(t)
	demo, _ := BindHandoverRepository(NewSQLHandoverStore(identity.db), testTenantRef("demo"))
	other, _ := BindHandoverRepository(NewSQLHandoverStore(identity.db), testTenantRef("other"))
	_, _ = demo.Create(HandoverRecord{ID: "demo", TenantSlug: "other", Title: "Demo", CreatedBy: "a@example.com"})
	_, _ = other.Create(HandoverRecord{ID: "other", TenantSlug: "demo", Title: "Other", CreatedBy: "a@example.com"})
	if got := demo.List(); len(got) != 1 || got[0].Title != "Demo" || got[0].TenantSlug != "demo" {
		t.Fatalf("demo handovers = %+v", got)
	}
}

func TestBoundIdentityRepositoryExcludesOtherTenants(t *testing.T) {
	identity := hausv516DB(t)
	person, err := identity.UpsertPerson(Person{Email: "a@example.com"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	for _, slug := range []string{"demo", "other"} {
		if _, err := identity.SetMembership(HouseMembership{PersonID: person.ID, TenantSlug: slug, Role: RoleOwner}, time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	demo, _ := BindIdentityRepository(identity, testTenantRef("demo"))
	if got := demo.ListHouseMembers(); len(got) != 1 || got[0].Membership.TenantSlug != "demo" {
		t.Fatalf("demo members = %+v", got)
	}
}
