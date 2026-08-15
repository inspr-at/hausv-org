package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/db"
)

func TestEventStorageParity(t *testing.T) {
	backends := map[string]func(t *testing.T) EventStorage{
		"json": func(t *testing.T) EventStorage {
			s, err := NewEventStore(filepath.Join(t.TempDir(), "e.json"))
			if err != nil {
				t.Fatalf("json: %v", err)
			}
			return s
		},
		"sqlite": func(t *testing.T) EventStorage {
			database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
			if err != nil {
				t.Fatalf("db: %v", err)
			}
			t.Cleanup(func() { database.Close() })
			return NewSQLEventStore(database)
		},
	}
	now := time.Now().UTC()
	for name, build := range backends {
		t.Run(name, func(t *testing.T) {
			s, ok := BindEventRepository(build(t), "demo")
			if !ok {
				t.Fatal("bind event repository")
			}
			if _, err := s.Create(HouseEvent{TenantSlug: "demo", Title: ""}); err == nil {
				t.Fatal("event without title/start must error")
			}
			created, err := s.Create(HouseEvent{TenantSlug: "demo", Title: "Versammlung", StartsAt: now.Add(24 * time.Hour), AuthorEmail: "a@example.com"})
			if err != nil || created.ID == "" {
				t.Fatalf("create: %v %+v", err, created)
			}
			id := created.ID
			if got := s.List(); len(got) != 1 || got[0].ID != id {
				t.Fatalf("list = %+v", got)
			}
			updatedOK, err := s.Update(id, HouseEvent{TenantSlug: "demo", Title: "Versammlung 2", StartsAt: now.Add(48 * time.Hour)})
			if err != nil || !updatedOK {
				t.Fatalf("update: %v ok=%v", err, updatedOK)
			}
			if got := s.List(); got[0].Title != "Versammlung 2" || !got[0].CreatedAt.Equal(created.CreatedAt) {
				t.Fatalf("update result: %+v", got[0])
			}
			// past event -> not upcoming
			_, _ = s.Create(HouseEvent{TenantSlug: "demo", Title: "Vergangen", StartsAt: now.Add(-72 * time.Hour)})
			up := s.Upcoming(now)
			if len(up) != 1 || up[0].Title != "Versammlung 2" {
				t.Fatalf("upcoming = %d %+v", len(up), up)
			}
			del, err := s.Delete(id)
			if err != nil || !del {
				t.Fatalf("delete: %v %v", err, del)
			}
			if d2, _ := s.Delete("nope"); d2 {
				t.Fatal("delete unknown must be false")
			}
		})
	}
}
