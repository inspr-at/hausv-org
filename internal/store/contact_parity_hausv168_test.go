package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/dbtest"
)

func TestContactBookStorageParity(t *testing.T) {
	backends := map[string]func(t *testing.T) ContactBookStorage{
		"json": func(t *testing.T) ContactBookStorage {
			s, err := NewContactBookStore(filepath.Join(t.TempDir(), "contacts.json"))
			if err != nil {
				t.Fatalf("json store: %v", err)
			}
			return s
		},
		"sqlite": func(t *testing.T) ContactBookStorage {
			database := dbtest.Open(t)
			t.Cleanup(func() { database.Close() })
			return NewSQLContactBookStore(database)
		},
	}

	for name, build := range backends {
		t.Run(name, func(t *testing.T) {
			s, ok := BindContactBookRepository(build(t), testTenantRef("demo"))
			if !ok {
				t.Fatal("bind contact repository")
			}

			if _, _, err := s.Upsert(ManagedContact{TenantSlug: "demo", Kind: "dienstleister"}); err == nil {
				t.Fatal("contact without name/route must error")
			}

			created, isNew, err := s.Upsert(ManagedContact{
				TenantSlug: "demo", Kind: "energie-fachbetrieb", Name: "Test AG", Phone: "+43 1 234", Active: true,
				ServiceRegion: "Wien und Umgebung", Qualification: "Elektrotechnik",
				EnergyCapabilities: []string{"metering", "home-assistant", "unknown", "metering"},
			})
			if err != nil || !isNew {
				t.Fatalf("create: err=%v isNew=%v", err, isNew)
			}
			if created.ID == "" || created.CreatedAt.IsZero() {
				t.Fatalf("created missing id/timestamp: %+v", created)
			}
			if created.Kind != "Energie-Fachbetrieb" || created.ServiceRegion != "Wien und Umgebung" ||
				created.Qualification != "Elektrotechnik" || len(created.EnergyCapabilities) != 2 {
				t.Fatalf("energy contact fields = %+v", created)
			}
			id := created.ID

			if got := s.List(false); len(got) != 1 || got[0].ID != id {
				t.Fatalf("list active = %+v", got)
			}

			// Update by id: isNew=false, CreatedAt preserved, fields changed.
			updated, isNew2, err := s.Upsert(ManagedContact{
				TenantSlug: "demo", ID: id, Kind: "hausmeister", Name: "Test AG 2", Phone: "+43 1 999", Active: true,
			})
			if err != nil || isNew2 {
				t.Fatalf("update: err=%v isNew=%v", err, isNew2)
			}
			if !updated.CreatedAt.Equal(created.CreatedAt) {
				t.Fatalf("CreatedAt not preserved on update: %v vs %v", updated.CreatedAt, created.CreatedAt)
			}
			if updated.Kind != "Hausmeister" || updated.Name != "Test AG 2" {
				t.Fatalf("updated fields: %+v", updated)
			}

			// Deactivate: excluded from active list, present with includeInactive.
			deact, err := s.Deactivate(id, time.Time{})
			if err != nil || deact.Active {
				t.Fatalf("deactivate: err=%v active=%v", err, deact.Active)
			}
			if got := s.List(false); len(got) != 0 {
				t.Fatalf("deactivated must be excluded from active list: %+v", got)
			}
			if got := s.List(true); len(got) != 1 {
				t.Fatalf("includeInactive must show it: %+v", got)
			}

			// Deactivate unknown -> empty, no error.
			if c, err := s.Deactivate("does-not-exist", time.Time{}); err != nil || c.ID != "" {
				t.Fatalf("deactivate unknown: err=%v c=%+v", err, c)
			}
		})
	}
}

func TestSQLContactImportFromJSON(t *testing.T) {
	jsonStore, err := NewContactBookStore(filepath.Join(t.TempDir(), "contacts.json"))
	if err != nil {
		t.Fatalf("json store: %v", err)
	}
	jsonRepo, _ := BindContactBookRepository(jsonStore, testTenantRef("demo"))
	a, _, _ := jsonRepo.Upsert(ManagedContact{TenantSlug: "demo", Kind: "dienstleister", Name: "Alpha", Phone: "+43 1 1", Active: true})
	_, _, _ = jsonRepo.Upsert(ManagedContact{TenantSlug: "demo", Kind: "notdienst", Company: "Beta GmbH", Email: "b@example.com", Active: true})

	database := dbtest.Open(t)
	defer database.Close()
	sqlStore := NewSQLContactBookStore(database)

	for i := 0; i < 2; i++ {
		if err := sqlStore.ImportContacts(jsonStore); err != nil {
			t.Fatalf("import %d: %v", i, err)
		}
	}
	sqlRepo, _ := BindContactBookRepository(sqlStore, testTenantRef("demo"))
	list := sqlRepo.List(true)
	if len(list) != 2 {
		t.Fatalf("imported %d contacts, want 2: %+v", len(list), list)
	}
	if _, err := sqlRepo.Deactivate(a.ID, time.Time{}); err != nil {
		t.Fatalf("deactivate imported: %v", err)
	}
	if got := sqlRepo.List(false); len(got) != 1 {
		t.Fatalf("after deactivate one, active should be 1: %+v", got)
	}
}
