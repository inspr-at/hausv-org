package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/markus-barta/hausv-org/internal/db"
)

func TestAnnouncementStorageParity(t *testing.T) {
	backends := map[string]func(t *testing.T) AnnouncementStorage{
		"json": func(t *testing.T) AnnouncementStorage {
			s, err := NewAnnouncementStore(filepath.Join(t.TempDir(), "a.json"))
			if err != nil {
				t.Fatalf("json: %v", err)
			}
			return s
		},
		"sqlite": func(t *testing.T) AnnouncementStorage {
			database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
			if err != nil {
				t.Fatalf("db: %v", err)
			}
			t.Cleanup(func() { database.Close() })
			return NewSQLAnnouncementStore(database)
		},
	}
	now := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	for name, build := range backends {
		t.Run(name, func(t *testing.T) {
			s := build(t)
			// Create: gets id, defaults category Info, published now.
			created, err := s.Create(Announcement{TenantSlug: "jhw22", Title: "Hallo", Body: "b", AuthorEmail: "a@example.com", PublishedAt: now})
			if err != nil || created.ID == "" || created.Category != "Info" || created.PublishedAt.IsZero() {
				t.Fatalf("create: %v %+v", err, created)
			}
			id := created.ID
			if got := s.ListTenant("jhw22"); len(got) != 1 || got[0].ID != id {
				t.Fatalf("list = %+v", got)
			}
			// Update preserves CreatedAt + id, changes title.
			ok, err := s.Update(id, Announcement{TenantSlug: "jhw22", Title: "Neu", Body: "b2"})
			if err != nil || !ok {
				t.Fatalf("update: %v ok=%v", err, ok)
			}
			got := s.ListTenant("jhw22")
			if got[0].Title != "Neu" || !got[0].CreatedAt.Equal(created.CreatedAt) {
				t.Fatalf("update result: %+v", got[0])
			}
			// A future-published + an expired one.
			future := now.Add(24 * time.Hour)
			_, _ = s.Create(Announcement{TenantSlug: "jhw22", Title: "Later", Body: "x", PublishedAt: future})
			past := now.Add(-48 * time.Hour)
			exp := now.Add(-time.Hour)
			_, _ = s.Create(Announcement{TenantSlug: "jhw22", Title: "Old", Body: "x", PublishedAt: past, ExpiresAt: &exp})
			vis := s.Visible("jhw22", now) // excludes future + expired
			arc := s.Archive("jhw22", now) // excludes future, includes expired
			if len(vis) != 1 {
				t.Fatalf("visible = %d (%+v)", len(vis), titles(vis))
			}
			if len(arc) != 2 {
				t.Fatalf("archive = %d (%+v)", len(arc), titles(arc))
			}
			// Delete.
			del, err := s.Delete("jhw22", id)
			if err != nil || !del {
				t.Fatalf("delete: %v %v", err, del)
			}
			if d2, _ := s.Delete("jhw22", "nope"); d2 {
				t.Fatal("delete unknown must be false")
			}
		})
	}
}

func titles(a []Announcement) []string {
	out := []string{}
	for _, x := range a {
		out = append(out, x.Title)
	}
	return out
}
