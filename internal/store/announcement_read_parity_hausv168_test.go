package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/markus-barta/hausv-org/internal/db"
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
			database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
			if err != nil {
				t.Fatalf("db open: %v", err)
			}
			t.Cleanup(func() { database.Close() })
			return NewSQLAnnouncementReadStore(database)
		},
	}
	for name, build := range backends {
		t.Run(name, func(t *testing.T) {
			s := build(t)
			if !s.LastSeen("jhw22", "nobody@example.com").IsZero() {
				t.Fatal("unseen must be zero time")
			}
			when := time.Date(2026, 7, 20, 10, 0, 0, 500, time.UTC)
			if err := s.MarkSeen("jhw22", "Person@Example.com", when); err != nil {
				t.Fatalf("mark: %v", err)
			}
			if got := s.LastSeen("jhw22", "person@example.com"); !got.Equal(when) {
				t.Fatalf("last seen = %v, want %v", got, when)
			}
			// Different tenant is isolated.
			if !s.LastSeen("other", "person@example.com").IsZero() {
				t.Fatal("other tenant must be isolated")
			}
			// MarkSeen upserts to the newer time.
			later := when.Add(time.Hour)
			_ = s.MarkSeen("jhw22", "person@example.com", later)
			if got := s.LastSeen("jhw22", "person@example.com"); !got.Equal(later) {
				t.Fatalf("upsert last seen = %v, want %v", got, later)
			}
			// Empty inputs are no-ops.
			_ = s.MarkSeen("", "x@example.com", when)
			if !s.LastSeen("", "x@example.com").IsZero() {
				t.Fatal("empty tenant must be a no-op")
			}
		})
	}
}
