package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/dbtest"
)

func TestAnnouncementReadStorageParity(t *testing.T) {
	backends := map[string]func(t *testing.T) AnnouncementReadStorage{
		"json": func(t *testing.T) AnnouncementReadStorage {
			s, err := NewAnnouncementReadStore(filepath.Join(t.TempDir(), "ar.json"))
			if err != nil {
				t.Fatalf("json store: %v", err)
			}
			return s
		},
		"sqlite": func(t *testing.T) AnnouncementReadStorage {
			database := dbtest.Open(t)
			t.Cleanup(func() { database.Close() })
			return NewSQLAnnouncementReadStore(database)
		},
	}
	for name, build := range backends {
		t.Run(name, func(t *testing.T) {
			storage := build(t)
			demo, ok := BindAnnouncementReadRepository(storage, "demo")
			if !ok {
				t.Fatal("bind demo repository")
			}
			other, ok := BindAnnouncementReadRepository(storage, "other")
			if !ok {
				t.Fatal("bind other repository")
			}
			if !demo.LastSeen("nobody@example.com").IsZero() {
				t.Fatal("unseen must be zero time")
			}
			when := time.Date(2026, 7, 20, 10, 0, 0, 500, time.UTC)
			if err := demo.MarkSeen("Person@Example.com", when); err != nil {
				t.Fatalf("mark: %v", err)
			}
			if got := demo.LastSeen("person@example.com"); !got.Equal(when) {
				t.Fatalf("last seen = %v, want %v", got, when)
			}
			// Different tenant is isolated.
			if !other.LastSeen("person@example.com").IsZero() {
				t.Fatal("other tenant must be isolated")
			}
			// MarkSeen upserts to the newer time.
			later := when.Add(time.Hour)
			_ = demo.MarkSeen("person@example.com", later)
			if got := demo.LastSeen("person@example.com"); !got.Equal(later) {
				t.Fatalf("upsert last seen = %v, want %v", got, later)
			}
			// An unscoped repository cannot be constructed.
			if unscoped, ok := BindAnnouncementReadRepository(storage, ""); ok || unscoped != nil {
				t.Fatal("empty tenant must not produce a repository")
			}
		})
	}
}
