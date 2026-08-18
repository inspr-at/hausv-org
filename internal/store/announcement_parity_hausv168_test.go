package store

import (
	"path/filepath"
	"testing"
	"time"
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
			database, lanes := testLanes(t)
			t.Cleanup(func() { database.Close() })
			return NewSQLAnnouncementStore(lanes)
		},
	}
	now := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	for name, build := range backends {
		t.Run(name, func(t *testing.T) {
			s, ok := BindAnnouncementRepository(build(t), testTenantRef("demo"))
			if !ok {
				t.Fatal("bind announcement repository")
			}
			// Create: gets id, defaults category Info, published now.
			created, err := s.Create(Announcement{TenantSlug: "demo", Title: "Hallo", Body: "b", AuthorEmail: "a@example.com", PublishedAt: now})
			if err != nil || created.ID == "" || created.Category != "Info" || created.PublishedAt.IsZero() {
				t.Fatalf("create: %v %+v", err, created)
			}
			id := created.ID
			if got := s.List(); len(got) != 1 || got[0].ID != id {
				t.Fatalf("list = %+v", got)
			}
			// Update preserves CreatedAt + id, changes title.
			updatedOK, err := s.Update(id, Announcement{TenantSlug: "demo", Title: "Neu", Body: "b2"})
			if err != nil || !updatedOK {
				t.Fatalf("update: %v ok=%v", err, updatedOK)
			}
			got := s.List()
			if got[0].Title != "Neu" || !got[0].CreatedAt.Equal(created.CreatedAt) {
				t.Fatalf("update result: %+v", got[0])
			}
			// A future-published + an expired one.
			future := now.Add(24 * time.Hour)
			_, _ = s.Create(Announcement{TenantSlug: "demo", Title: "Later", Body: "x", PublishedAt: future})
			past := now.Add(-48 * time.Hour)
			exp := now.Add(-time.Hour)
			_, _ = s.Create(Announcement{TenantSlug: "demo", Title: "Old", Body: "x", PublishedAt: past, ExpiresAt: &exp})
			vis := s.Visible(now) // excludes future + expired
			arc := s.Archive(now) // excludes future, includes expired
			if len(vis) != 1 {
				t.Fatalf("visible = %d (%+v)", len(vis), titles(vis))
			}
			if len(arc) != 2 {
				t.Fatalf("archive = %d (%+v)", len(arc), titles(arc))
			}
			// Delete.
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

func titles(a []Announcement) []string {
	out := []string{}
	for _, x := range a {
		out = append(out, x.Title)
	}
	return out
}
