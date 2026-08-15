package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/db"
)

// HAUSV-175: legacy issue photos move into the attachment store so the old
// PhotoPaths read path can finally go.

type legacyPhotoFixture struct {
	issues      IssueStorage
	attachments AttachmentStorage
	issueRepo   IssueRepository
	attachRepo  AttachmentRepository
	photoDir    string
	issueID     string
}

func newLegacyPhotoFixture(t *testing.T, backend string) legacyPhotoFixture {
	t.Helper()
	dir := t.TempDir()
	photoDir := filepath.Join(dir, "issue-attachments")
	if err := os.MkdirAll(filepath.Join(photoDir, "demo"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// The legacy file, exactly as SavePhoto used to leave it.
	if err := os.WriteFile(filepath.Join(photoDir, "demo", "abc-photo.png"), onePixelPNG, 0o600); err != nil {
		t.Fatalf("write legacy photo: %v", err)
	}

	var issues IssueStorage
	var attachments AttachmentStorage
	switch backend {
	case "json":
		is, err := NewIssueStore(filepath.Join(dir, "issues.json"), photoDir)
		if err != nil {
			t.Fatalf("issue store: %v", err)
		}
		as, err := NewAttachmentStore(filepath.Join(dir, "attachments.json"), filepath.Join(dir, "files"))
		if err != nil {
			t.Fatalf("attachment store: %v", err)
		}
		issues, attachments = is, as
	default:
		database, err := db.Open(filepath.Join(dir, "test.db"))
		if err != nil {
			t.Fatalf("db open: %v", err)
		}
		t.Cleanup(func() { database.Close() })
		issues = NewSQLIssueStore(database, photoDir)
		attachments = NewSQLAttachmentStore(database, filepath.Join(dir, "files"))
	}

	issue := sampleIssue()
	issue.PhotoPaths = []string{"issue-attachments/demo/abc-photo.png"}
	issueRepo, _ := BindIssueRepository(issues, "demo")
	attachRepo, _ := BindAttachmentRepository(attachments, "demo")
	created, err := issueRepo.Create(issue)
	if err != nil {
		t.Fatalf("seed issue: %v", err)
	}
	return legacyPhotoFixture{issues, attachments, issueRepo, attachRepo, photoDir, created.ID}
}

func TestMigrateLegacyIssuePhotos(t *testing.T) {
	for _, backend := range []string{"json", "sqlite"} {
		t.Run(backend, func(t *testing.T) {
			f := newLegacyPhotoFixture(t, backend)
			now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

			n, err := MigrateLegacyIssuePhotos(f.issues, f.attachments, f.photoDir, []string{"demo"}, now)
			if err != nil || n != 1 {
				t.Fatalf("migrate: n=%d err=%v", n, err)
			}

			// The photo is now a normal attachment on the issue.
			got := f.attachRepo.ListEntity("issue", f.issueID)
			if len(got) != 1 {
				t.Fatalf("expected 1 attachment, got %d", len(got))
			}
			if got[0].ContentType != "image/png" {
				t.Fatalf("content type = %q", got[0].ContentType)
			}
			path, _, _, ok := f.attachRepo.FilePath(got[0], "")
			if !ok {
				t.Fatal("attachment path not resolvable")
			}
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("attachment file missing: %v", err)
			}

			// The legacy field is cleared, so the old read path has nothing left.
			issue, _ := f.issueRepo.Get(f.issueID)
			if len(issue.PhotoPaths) != 0 {
				t.Fatalf("PhotoPaths not cleared: %+v", issue.PhotoPaths)
			}

			// Idempotent: a second run migrates nothing and creates no duplicate.
			n2, err := MigrateLegacyIssuePhotos(f.issues, f.attachments, f.photoDir, []string{"demo"}, now)
			if err != nil || n2 != 0 {
				t.Fatalf("second run: n=%d err=%v", n2, err)
			}
			if got := f.attachRepo.ListEntity("issue", f.issueID); len(got) != 1 {
				t.Fatalf("second run duplicated attachments: %d", len(got))
			}
		})
	}
}

// A photo whose file is gone must NOT be silently forgotten: the migration
// reports it and leaves the legacy reference intact.
func TestMigrateLegacyIssuePhotosKeepsReferenceWhenFileMissing(t *testing.T) {
	f := newLegacyPhotoFixture(t, "sqlite")
	if err := os.Remove(filepath.Join(f.photoDir, "demo", "abc-photo.png")); err != nil {
		t.Fatalf("remove: %v", err)
	}

	n, err := MigrateLegacyIssuePhotos(f.issues, f.attachments, f.photoDir, []string{"demo"}, time.Now())
	if err == nil {
		t.Fatal("a missing legacy file must be reported, not skipped silently")
	}
	if n != 0 {
		t.Fatalf("nothing should have migrated, got %d", n)
	}
	issue, _ := f.issueRepo.Get(f.issueID)
	if len(issue.PhotoPaths) != 1 {
		t.Fatalf("the legacy reference must survive so it is not lost: %+v", issue.PhotoPaths)
	}
}
