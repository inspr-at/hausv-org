package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

type attachmentBackend struct {
	store   AttachmentRepository
	fileDir string
}

func TestAttachmentStorageParity(t *testing.T) {
	backends := map[string]func(t *testing.T) attachmentBackend{
		"json": func(t *testing.T) attachmentBackend {
			dir := t.TempDir()
			fileDir := filepath.Join(dir, "files")
			s, err := NewAttachmentStore(filepath.Join(dir, "attachments.json"), fileDir)
			if err != nil {
				t.Fatalf("json store: %v", err)
			}
			repository, _ := BindAttachmentRepository(s, testTenantRef("demo"))
			return attachmentBackend{repository, fileDir}
		},
		"sqlite": func(t *testing.T) attachmentBackend {
			dir := t.TempDir()
			fileDir := filepath.Join(dir, "files")
			database := testDB(t)
			t.Cleanup(func() { database.Close() })
			repository, _ := BindAttachmentRepository(NewSQLAttachmentStore(database, fileDir), testTenantRef("demo"))
			return attachmentBackend{repository, fileDir}
		},
	}

	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	for name, build := range backends {
		t.Run(name, func(t *testing.T) {
			b := build(t)
			s := b.store

			// Missing target is rejected.
			if _, err := s.CreateUploaded("issue", "", "a@example.com", []UploadedFile{uploadFrom("p.png", onePixelPNG)}, now); err == nil {
				t.Fatal("missing entity id must error")
			}

			created, err := s.CreateUploaded("issue", "issue-1", "admin@example.com",
				[]UploadedFile{uploadFrom("photo.png", onePixelPNG)}, now)
			if err != nil || len(created) != 1 {
				t.Fatalf("create: err=%v n=%d", err, len(created))
			}
			rec := created[0]
			if rec.ID == "" || rec.ContentType != "image/png" || rec.Size == 0 {
				t.Fatalf("record bad: %+v", rec)
			}

			// Original resolves and exists on disk.
			path, ctype, size, ok := s.FilePath(rec, "")
			if !ok || ctype != "image/png" || size != rec.Size {
				t.Fatalf("FilePath = %q %q %d ok=%v", path, ctype, size, ok)
			}
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("stored file missing: %v", err)
			}
			// A thumb variant falls back sensibly and always resolves.
			if _, _, _, ok := s.FilePath(rec, "thumb"); !ok {
				t.Fatal("thumb variant must resolve")
			}

			if got := s.ListEntity("issue", "issue-1"); len(got) != 1 || got[0].ID != rec.ID {
				t.Fatalf("list = %+v", got)
			}
			// Scoped to its entity.
			if got := s.ListEntity("issue", "other"); len(got) != 0 {
				t.Fatalf("other entity must be empty: %+v", got)
			}
			if got, ok := s.Get(rec.ID); !ok || got.ID != rec.ID {
				t.Fatalf("get = %+v ok=%v", got, ok)
			}

			// Delete: record disappears from reads and the files are removed.
			deleted, ok, err := s.Delete(rec.ID, now.Add(time.Hour))
			if err != nil || !ok || deleted.ID != rec.ID {
				t.Fatalf("delete: err=%v ok=%v rec=%+v", err, ok, deleted)
			}
			if _, ok := s.Get(rec.ID); ok {
				t.Fatal("deleted attachment must not be gettable")
			}
			if got := s.ListEntity("issue", "issue-1"); len(got) != 0 {
				t.Fatalf("deleted attachment still listed: %+v", got)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatalf("file must be removed on delete, stat err = %v", err)
			}
			// Deleting twice is a no-op, not an error.
			if _, ok, err := s.Delete(rec.ID, now); ok || err != nil {
				t.Fatalf("second delete: ok=%v err=%v", ok, err)
			}
			if _, ok, err := s.Delete("does-not-exist", now); ok || err != nil {
				t.Fatalf("delete unknown: ok=%v err=%v", ok, err)
			}
		})
	}
}

// A batch containing an invalid upload must leave NO records and NO files —
// the same all-or-nothing contract HAUSV-140 pinned for the JSON store.
func TestAttachmentBatchRollbackParity(t *testing.T) {
	backends := map[string]func(t *testing.T) attachmentBackend{
		"json": func(t *testing.T) attachmentBackend {
			dir := t.TempDir()
			fileDir := filepath.Join(dir, "files")
			s, err := NewAttachmentStore(filepath.Join(dir, "attachments.json"), fileDir)
			if err != nil {
				t.Fatalf("json store: %v", err)
			}
			repository, _ := BindAttachmentRepository(s, testTenantRef("demo"))
			return attachmentBackend{repository, fileDir}
		},
		"sqlite": func(t *testing.T) attachmentBackend {
			dir := t.TempDir()
			fileDir := filepath.Join(dir, "files")
			database := testDB(t)
			t.Cleanup(func() { database.Close() })
			repository, _ := BindAttachmentRepository(NewSQLAttachmentStore(database, fileDir), testTenantRef("demo"))
			return attachmentBackend{repository, fileDir}
		},
	}

	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	for name, build := range backends {
		t.Run(name, func(t *testing.T) {
			b := build(t)
			// First upload is a valid PNG whose file gets written; the second
			// fails the type check, so the whole batch must roll back.
			uploads := []UploadedFile{
				uploadFrom("photo.png", onePixelPNG),
				uploadFrom("notes.txt", []byte("this is not an image")),
			}
			if _, err := b.store.CreateUploaded("issue", "issue-1", "admin@example.com", uploads, now); err == nil {
				t.Fatal("a batch containing an invalid upload must fail")
			}
			if got := b.store.ListEntity("issue", "issue-1"); len(got) != 0 {
				t.Fatalf("rollback must leave no records, got %d", len(got))
			}
			if n := countFiles(t, b.fileDir); n != 0 {
				t.Fatalf("rollback must leave no files, got %d", n)
			}
		})
	}
}

func TestSQLAttachmentImportFromJSON(t *testing.T) {
	dir := t.TempDir()
	fileDir := filepath.Join(dir, "files")
	jsonStore, err := NewAttachmentStore(filepath.Join(dir, "attachments.json"), fileDir)
	if err != nil {
		t.Fatalf("json store: %v", err)
	}
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	jsonAttachments, _ := BindAttachmentRepository(jsonStore, testTenantRef("demo"))
	created, err := jsonAttachments.CreateUploaded("issue", "issue-1", "admin@example.com",
		[]UploadedFile{uploadFrom("a.png", onePixelPNG), uploadFrom("b.png", onePixelPNG)}, now)
	if err != nil || len(created) != 2 {
		t.Fatalf("seed: err=%v n=%d", err, len(created))
	}
	// One is deleted before the import: the soft-deleted state must carry over.
	if _, _, err := jsonAttachments.Delete(created[1].ID, now.Add(time.Minute)); err != nil {
		t.Fatalf("seed delete: %v", err)
	}

	database := testDB(t)
	defer database.Close()
	sqlStore := NewSQLAttachmentStore(database, fileDir)

	for i := 0; i < 2; i++ {
		if err := sqlStore.ImportAttachments(jsonStore); err != nil {
			t.Fatalf("import %d: %v", i, err)
		}
	}
	sqlAttachments, _ := BindAttachmentRepository(sqlStore, testTenantRef("demo"))
	live := sqlAttachments.ListEntity("issue", "issue-1")
	if len(live) != 1 || live[0].ID != created[0].ID {
		t.Fatalf("after import expected only the live attachment, got %+v", live)
	}
	if _, ok := sqlAttachments.Get(created[1].ID); ok {
		t.Fatal("soft-deleted attachment must stay deleted after import")
	}
	// The file behind the surviving record is still readable.
	path, _, _, ok := sqlAttachments.FilePath(live[0], "")
	if !ok {
		t.Fatal("FilePath after import failed")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file behind imported metadata missing: %v", err)
	}
}
