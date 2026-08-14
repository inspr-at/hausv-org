package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/db"
)

// HAUSV-148: filing a handover protocol writes a document AND the link on the
// handover. A retry must never produce a second protocol document, and a failed
// filing must not leave an orphaned file or document record behind.

type filingBackend struct {
	filer     ProtocolFiler
	documents DocumentStorage
	handovers HandoverStorage
	fileDir   string
}

func countFiles(t *testing.T, dir string) int {
	t.Helper()
	n := 0
	_ = filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		n++
		return nil
	})
	return n
}

func protocolDoc() DocumentRecord {
	return DocumentRecord{
		TenantSlug: "demo",
		Title:      "Übergabeprotokoll Test",
		Category:   "Protokoll",
		Visibility: "owner",
		UploadedBy: "admin@example.com",
	}
}

func TestProtocolFilerParity(t *testing.T) {
	backends := map[string]func(t *testing.T) filingBackend{
		"sequential": func(t *testing.T) filingBackend {
			dir := t.TempDir()
			fileDir := filepath.Join(dir, "files")
			docs, err := NewDocumentStore(filepath.Join(dir, "documents.json"), fileDir)
			if err != nil {
				t.Fatalf("json docs: %v", err)
			}
			hs, err := NewHandoverStore(filepath.Join(dir, "handovers.json"))
			if err != nil {
				t.Fatalf("json handovers: %v", err)
			}
			return filingBackend{NewSequentialProtocolFiler(docs, hs), docs, hs, fileDir}
		},
		"sqlite-atomic": func(t *testing.T) filingBackend {
			dir := t.TempDir()
			fileDir := filepath.Join(dir, "files")
			database, err := db.Open(filepath.Join(dir, "test.db"))
			if err != nil {
				t.Fatalf("db open: %v", err)
			}
			t.Cleanup(func() { database.Close() })
			docs := NewSQLDocumentStore(database, fileDir)
			hs := NewSQLHandoverStore(database)
			filer := NewSQLProtocolFiler(docs, hs)
			if filer == nil {
				t.Fatal("expected an atomic filer when both stores share a db")
			}
			return filingBackend{filer, docs, hs, fileDir}
		},
	}

	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	for name, build := range backends {
		t.Run(name, func(t *testing.T) {
			b := build(t)
			if _, err := b.handovers.Create(sampleHandover("h1")); err != nil {
				t.Fatalf("seed handover: %v", err)
			}

			created, updated, already, err := b.filer.FileHandoverProtocol(
				"demo", "h1", protocolDoc(), "protokoll.pdf", "application/pdf", []byte("%PDF-1.4 fake"), now)
			if err != nil || already {
				t.Fatalf("file: err=%v already=%v", err, already)
			}
			if created.ID == "" || created.ContentType != "application/pdf" {
				t.Fatalf("created doc bad: %+v", created)
			}
			if updated.FiledDocumentID != created.ID {
				t.Fatalf("handover not linked: %+v", updated)
			}
			// Link is persisted, not just returned.
			if got, _ := b.handovers.Get("demo", "h1"); got.FiledDocumentID != created.ID {
				t.Fatalf("persisted link = %q, want %q", got.FiledDocumentID, created.ID)
			}
			if docs := b.documents.ListTenant("demo"); len(docs) != 1 {
				t.Fatalf("want exactly 1 document, got %d", len(docs))
			}

			// Retry: must NOT create a second protocol document.
			_, again, already2, err := b.filer.FileHandoverProtocol(
				"demo", "h1", protocolDoc(), "protokoll.pdf", "application/pdf", []byte("%PDF-1.4 fake"), now.Add(time.Hour))
			if err != nil {
				t.Fatalf("refile: %v", err)
			}
			if !already2 {
				t.Fatal("refiling an already-filed handover must report alreadyFiled")
			}
			if again.FiledDocumentID != created.ID {
				t.Fatalf("refile changed the link: %+v", again)
			}
			if docs := b.documents.ListTenant("demo"); len(docs) != 1 {
				t.Fatalf("retry created a duplicate document: %d", len(docs))
			}
			// And no orphaned file was left by the retry.
			if n := countFiles(t, b.fileDir); n != 1 {
				t.Fatalf("want exactly 1 stored file after retry, got %d", n)
			}

			// Unknown handover: no document, no file left behind.
			if _, _, _, err := b.filer.FileHandoverProtocol(
				"demo", "does-not-exist", protocolDoc(), "x.pdf", "application/pdf", []byte("%PDF-1.4 fake"), now); err == nil {
				t.Fatal("filing an unknown handover must error")
			}
			if docs := b.documents.ListTenant("demo"); len(docs) != 1 {
				t.Fatalf("failed filing left a document behind: %d", len(docs))
			}
			if n := countFiles(t, b.fileDir); n != 1 {
				t.Fatalf("failed filing left an orphaned file: %d files", n)
			}
		})
	}
}

// The duplicate bug in the wild: two filings racing for the same handover.
// Whatever order they land in, exactly ONE protocol document may exist and
// exactly one file may remain on disk.
func TestSQLProtocolFilerConcurrentFilingCreatesOneDocument(t *testing.T) {
	dir := t.TempDir()
	fileDir := filepath.Join(dir, "files")
	database, err := db.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("db open: %v", err)
	}
	defer database.Close()
	docs := NewSQLDocumentStore(database, fileDir)
	handovers := NewSQLHandoverStore(database)
	filer := NewSQLProtocolFiler(docs, handovers)
	if _, err := handovers.Create(sampleHandover("h1")); err != nil {
		t.Fatalf("seed: %v", err)
	}

	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	start := make(chan struct{})
	done := make(chan struct{}, 2)
	for i := 0; i < 2; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			<-start
			// Errors are acceptable for the loser of the race; the invariant
			// below is what must hold.
			_, _, _, _ = filer.FileHandoverProtocol(
				"demo", "h1", protocolDoc(), "protokoll.pdf", "application/pdf", []byte("%PDF-1.4 fake"), now)
		}()
	}
	close(start)
	<-done
	<-done

	list := docs.ListTenant("demo")
	if len(list) != 1 {
		t.Fatalf("concurrent filing produced %d documents, want exactly 1", len(list))
	}
	linked, _ := handovers.Get("demo", "h1")
	if linked.FiledDocumentID != list[0].ID {
		t.Fatalf("handover links %q but the only document is %q", linked.FiledDocumentID, list[0].ID)
	}
	if n := countFiles(t, fileDir); n != 1 {
		t.Fatalf("concurrent filing left %d files, want exactly 1", n)
	}
}

// The atomic filer is only handed out when both stores really share one DB;
// otherwise the caller must fall back to the sequential path.
func TestSQLProtocolFilerRequiresSharedDatabase(t *testing.T) {
	dirA, dirB := t.TempDir(), t.TempDir()
	dbA, err := db.Open(filepath.Join(dirA, "a.db"))
	if err != nil {
		t.Fatalf("db a: %v", err)
	}
	defer dbA.Close()
	dbB, err := db.Open(filepath.Join(dirB, "b.db"))
	if err != nil {
		t.Fatalf("db b: %v", err)
	}
	defer dbB.Close()

	if f := NewSQLProtocolFiler(NewSQLDocumentStore(dbA, dirA), NewSQLHandoverStore(dbB)); f != nil {
		t.Fatal("filer must be nil when the stores use different databases")
	}
	if f := NewSQLProtocolFiler(nil, NewSQLHandoverStore(dbA)); f != nil {
		t.Fatal("filer must be nil without a document store")
	}
	if f := NewSQLProtocolFiler(NewSQLDocumentStore(dbA, dirA), nil); f != nil {
		t.Fatal("filer must be nil without a handover store")
	}
	if f := NewSQLProtocolFiler(NewSQLDocumentStore(dbA, dirA), NewSQLHandoverStore(dbA)); f == nil {
		t.Fatal("filer must be returned when both stores share a database")
	}
}
