package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/db"
)

func sampleDocumentRecord() DocumentRecord {
	return DocumentRecord{
		TenantSlug: "demo",
		Title:      "Hausordnung",
		Category:   "",    // -> Sonstiges
		Visibility: "all", // -> alle Bewohner
		UploadedBy: "admin@example.com",
	}
}

func TestDocumentStorageParity(t *testing.T) {
	backends := map[string]func(t *testing.T) DocumentStorage{
		"json": func(t *testing.T) DocumentStorage {
			dir := t.TempDir()
			s, err := NewDocumentStore(filepath.Join(dir, "documents.json"), filepath.Join(dir, "files"))
			if err != nil {
				t.Fatalf("json store: %v", err)
			}
			return s
		},
		"sqlite": func(t *testing.T) DocumentStorage {
			dir := t.TempDir()
			database, err := db.Open(filepath.Join(dir, "test.db"))
			if err != nil {
				t.Fatalf("db open: %v", err)
			}
			t.Cleanup(func() { database.Close() })
			return NewSQLDocumentStore(database, filepath.Join(dir, "files"))
		},
	}

	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	for name, build := range backends {
		t.Run(name, func(t *testing.T) {
			s := build(t)

			// Invalid metadata (no title) is rejected.
			bad := sampleDocumentRecord()
			bad.Title = ""
			if _, err := s.Create(bad, uploadFrom("x.png", onePixelPNG), now); err == nil {
				t.Fatal("document without title must error")
			}

			created, err := s.Create(sampleDocumentRecord(), uploadFrom("hausordnung.png", onePixelPNG), now)
			if err != nil {
				t.Fatalf("create: %v", err)
			}
			if created.ID == "" || created.Version != 1 || !created.Current {
				t.Fatalf("created bad: %+v", created)
			}
			if created.SeriesID != created.ID {
				t.Fatalf("series id should equal first id: %+v", created)
			}
			if created.ContentType != "image/png" {
				t.Fatalf("sniffed content type = %q", created.ContentType)
			}

			// File landed on disk.
			path, ok := s.FilePath(created)
			if !ok {
				t.Fatal("FilePath not resolvable")
			}
			if st, err := os.Stat(path); err != nil || st.Size() == 0 {
				t.Fatalf("stored file missing: err=%v", err)
			}

			if got, ok := s.Get("demo", created.ID); !ok || got.Title != "Hausordnung" {
				t.Fatalf("get = %+v ok=%v", got, ok)
			}
			if _, ok := s.Get("demo", "nope"); ok {
				t.Fatal("get unknown must be false")
			}
			if got := s.ListCurrentTenant("demo"); len(got) != 1 {
				t.Fatalf("current list = %+v", got)
			}

			// Replace: supersede + insert must be consistent.
			replacement, replaced, err := s.Replace("demo", created.ID, "boss@example.com", uploadFrom("neu.png", onePixelPNG), now.Add(time.Hour))
			if err != nil {
				t.Fatalf("replace: %v", err)
			}
			if replacement.Version != 2 || !replacement.Current || replacement.SupersedesID != created.ID {
				t.Fatalf("replacement bad: %+v", replacement)
			}
			if replaced.Current || replaced.ReplacedByID != replacement.ID {
				t.Fatalf("replaced not superseded: %+v", replaced)
			}
			if replacement.SeriesID != created.SeriesID {
				t.Fatalf("series must be stable: %q vs %q", replacement.SeriesID, created.SeriesID)
			}
			if replacement.UploadedBy != "boss@example.com" {
				t.Fatalf("uploadedBy = %q", replacement.UploadedBy)
			}

			// Exactly one current version remains.
			if got := s.ListCurrentTenant("demo"); len(got) != 1 || got[0].ID != replacement.ID {
				t.Fatalf("after replace current = %+v", got)
			}
			if got := s.ListTenant("demo"); len(got) != 2 {
				t.Fatalf("after replace all = %+v", got)
			}

			// Versions: newest first.
			versions := s.Versions("demo", created.SeriesID)
			if len(versions) != 2 || versions[0].Version != 2 || versions[1].Version != 1 {
				t.Fatalf("versions = %+v", versions)
			}

			// Replacing a superseded (non-current) version is refused.
			if _, _, err := s.Replace("demo", created.ID, "boss@example.com", uploadFrom("x.png", onePixelPNG), now); err == nil {
				t.Fatal("replacing a superseded version must error")
			}
			if _, _, err := s.Replace("demo", "missing", "boss@example.com", uploadFrom("x.png", onePixelPNG), now); err == nil {
				t.Fatal("replacing unknown must error")
			}

			// Generated document (handover PDF path).
			gen, err := s.CreateGenerated(sampleDocumentRecord(), "protokoll.png", "image/png", onePixelPNG, now)
			if err != nil {
				t.Fatalf("CreateGenerated: %v", err)
			}
			if gen.Version != 1 || !gen.Current || gen.Size != int64(len(onePixelPNG)) {
				t.Fatalf("generated bad: %+v", gen)
			}
			genPath, ok := s.FilePath(gen)
			if !ok {
				t.Fatal("generated FilePath not resolvable")
			}
			if _, err := os.Stat(genPath); err != nil {
				t.Fatalf("generated file missing: %v", err)
			}
		})
	}
}

func TestSQLDocumentImportFromJSON(t *testing.T) {
	dir := t.TempDir()
	fileDir := filepath.Join(dir, "files")
	jsonStore, err := NewDocumentStore(filepath.Join(dir, "documents.json"), fileDir)
	if err != nil {
		t.Fatalf("json store: %v", err)
	}
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	first, err := jsonStore.Create(sampleDocumentRecord(), uploadFrom("a.png", onePixelPNG), now)
	if err != nil {
		t.Fatalf("seed a: %v", err)
	}
	if _, _, err := jsonStore.Replace("demo", first.ID, "boss@example.com", uploadFrom("b.png", onePixelPNG), now.Add(time.Hour)); err != nil {
		t.Fatalf("seed replace: %v", err)
	}

	database, err := db.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("db open: %v", err)
	}
	defer database.Close()
	// Same fileDir: files already live on disk, only metadata is imported.
	sqlStore := NewSQLDocumentStore(database, fileDir)

	for i := 0; i < 2; i++ {
		if err := sqlStore.ImportDocuments(jsonStore); err != nil {
			t.Fatalf("import %d: %v", i, err)
		}
	}
	if got := sqlStore.ListTenant("demo"); len(got) != 2 {
		t.Fatalf("imported %d, want 2: %+v", len(got), got)
	}
	// Version history and current-flag survive the import.
	if got := sqlStore.ListCurrentTenant("demo"); len(got) != 1 || got[0].Version != 2 {
		t.Fatalf("current after import = %+v", got)
	}
	if got := sqlStore.Versions("demo", first.SeriesID); len(got) != 2 {
		t.Fatalf("versions after import = %+v", got)
	}
	// The file referenced by imported metadata is still readable.
	cur := sqlStore.ListCurrentTenant("demo")[0]
	path, ok := sqlStore.FilePath(cur)
	if !ok {
		t.Fatal("FilePath after import failed")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file behind imported metadata missing: %v", err)
	}
}
