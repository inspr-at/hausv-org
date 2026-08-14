package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/db"
)

// HAUSV-146: soft-deleted attachment records must not accumulate forever.
func TestPurgeDeletedAttachmentTombstones(t *testing.T) {
	dir := t.TempDir()
	database, err := db.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("db open: %v", err)
	}
	defer database.Close()
	s := NewSQLAttachmentStore(database, filepath.Join(dir, "files"))

	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	created, err := s.CreateUploaded("demo", "issue", "issue-1", "a@example.com",
		[]UploadedFile{uploadFrom("a.png", onePixelPNG), uploadFrom("b.png", onePixelPNG)}, now)
	if err != nil || len(created) != 2 {
		t.Fatalf("seed: err=%v n=%d", err, len(created))
	}
	// One is deleted long ago, the other stays live.
	if _, _, err := s.Delete("demo", created[0].ID, now.Add(-400*24*time.Hour)); err != nil {
		t.Fatalf("delete: %v", err)
	}

	// A cutoff before the deletion keeps everything.
	n, err := s.PurgeDeletedBefore(now.Add(-500 * 24 * time.Hour))
	if err != nil || n != 0 {
		t.Fatalf("premature purge: n=%d err=%v", n, err)
	}

	n, err = s.PurgeDeletedBefore(now.Add(-AttachmentTombstoneRetention))
	if err != nil || n != 1 {
		t.Fatalf("purge: n=%d err=%v, want 1", n, err)
	}

	// The live attachment is untouched; the tombstone is gone.
	live := s.ListEntity("demo", "issue", "issue-1")
	if len(live) != 1 || live[0].ID != created[1].ID {
		t.Fatalf("purge disturbed the live attachment: %+v", live)
	}
	var remaining int
	if err := database.QueryRow(`SELECT count(*) FROM attachments`).Scan(&remaining); err != nil {
		t.Fatalf("count: %v", err)
	}
	if remaining != 1 {
		t.Fatalf("expected only the live row to remain, got %d", remaining)
	}
	// Idempotent.
	if n, err := s.PurgeDeletedBefore(now.Add(-AttachmentTombstoneRetention)); err != nil || n != 0 {
		t.Fatalf("second purge: n=%d err=%v", n, err)
	}
}
