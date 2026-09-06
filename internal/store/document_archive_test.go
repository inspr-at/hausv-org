package store

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"
)

func archiveTestDocument() DocumentRecord {
	return DocumentRecord{TenantSlug: "demo", Title: "Jahresabrechnung 2025 · Top 1 · Lauf 1", UnitID: "top-1", UploadedBy: "manager@example.com",
		AnnualStatementArchive: &AnnualStatementArchiveMetadata{RunID: "run-1", Revision: 1, PeriodYear: 2025, PartyID: "owner@example.com"}}
}

func TestDocumentArchivePersistenceIdempotencyAndImmutability(t *testing.T) {
	for _, backend := range []string{"json", "sqlite"} {
		t.Run(backend, func(t *testing.T) {
			dir := t.TempDir()
			var open func() DocumentRepository
			if backend == "json" {
				open = func() DocumentRepository {
					s, err := NewDocumentStore(filepath.Join(dir, "documents.json"), filepath.Join(dir, "files"))
					if err != nil {
						t.Fatal(err)
					}
					r, _ := BindDocumentRepository(s, testTenantRef("demo"))
					return r
				}
			} else {
				database, lanes := testLanes(t)
				t.Cleanup(func() { database.Close() })
				open = func() DocumentRepository {
					r, _ := BindDocumentRepository(NewSQLDocumentStore(lanes, filepath.Join(dir, "files")), testTenantRef("demo"))
					return r
				}
			}
			repo := open()
			item := archiveTestDocument()
			data := []byte("%PDF-1.4 archive")
			now := time.Date(2026, 9, 6, 10, 0, 0, 0, time.UTC)
			first, err := repo.CreateGenerated(item, "archive.pdf", "application/pdf", data, now)
			if err != nil {
				t.Fatal(err)
			}
			path, _ := repo.FilePath(first)
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if first.SeriesID != first.ID || first.Version != 1 || !first.Current || first.Category != DocumentCategoryBilling || first.Visibility != DocumentVisibilityManagerOnly || first.UnitID != item.UnitID {
				t.Fatalf("wrong archive metadata: %+v", first)
			}
			if first.AnnualStatementArchive.SHA256 != fmt.Sprintf("%x", sha256.Sum256(data)) || first.AnnualStatementArchive.ArchivedBy != item.UploadedBy || !first.AnnualStatementArchive.ArchivedAt.Equal(now) {
				t.Fatal("wrong archive provenance", first.AnnualStatementArchive)
			}
			if info.Mode().Perm() != 0400 {
				t.Fatalf("archive mode=%v", info.Mode())
			}
			repo = open()
			// Later actors, titles and renderer bytes cannot change an existing copy.
			item.UploadedBy, item.Title = "other@example.com", "Changed"
			retry, err := repo.CreateGenerated(item, "new.pdf", "application/pdf", []byte("%PDF-1.4 new"), now.Add(time.Hour))
			if err != nil || !reflect.DeepEqual(retry, first) {
				t.Fatal("retry changed archive", err)
			}
			if _, _, err := repo.Replace(first.ID, "other@example.com", uploadFrom("new.png", onePixelPNG), now); !errors.Is(err, ErrDocumentArchived) {
				t.Fatal("replacement must be refused", err)
			}
			copy, _ := repo.Get(first.ID)
			copy.AnnualStatementArchive.RunID = "mutated"
			stored, _ := repo.Get(first.ID)
			if !reflect.DeepEqual(stored, first) {
				t.Fatal("Get leaked mutable metadata")
			}
			item.AnnualStatementArchive.RunID, item.AnnualStatementArchive.Revision = "run-2", 2
			next, err := repo.CreateGenerated(item, "next.pdf", "application/pdf", data, now.Add(time.Hour))
			if err != nil || next.ID == first.ID || len(repo.ListCurrent()) != 2 {
				t.Fatal("revision did not create its own set", err)
			}
			stored, _ = repo.Get(first.ID)
			after, _ := os.Stat(path)
			raw, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(raw, data) || !after.ModTime().Equal(info.ModTime()) || !reflect.DeepEqual(stored, first) {
				t.Fatal("original archive changed", err)
			}
		})
	}
}

func TestDocumentArchiveConcurrentCreate(t *testing.T) {
	database, lanes := testLanes(t)
	defer database.Close()
	fileDir := t.TempDir()
	var wg sync.WaitGroup
	errors := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Independent store instances exercise the database key, not a Go lock.
			repo, _ := BindDocumentRepository(NewSQLDocumentStore(lanes, fileDir), testTenantRef("demo"))
			_, err := repo.CreateGenerated(archiveTestDocument(), "a.pdf", "application/pdf", []byte("%PDF-1.4 concurrent"), time.Now())
			errors <- err
		}()
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	repo, _ := BindDocumentRepository(NewSQLDocumentStore(lanes, fileDir), testTenantRef("demo"))
	if len(repo.List()) != 1 {
		t.Fatal("duplicate archive copies")
	}
	foreign, _ := BindDocumentRepository(NewSQLDocumentStore(lanes, fileDir), testTenantRef("other"))
	if _, found := foreign.Get(repo.List()[0].ID); found {
		t.Fatal("cross-tenant archive leak")
	}
}

func TestDocumentArchiveExclusiveFileAndOrphanRecovery(t *testing.T) {
	database, lanes := testLanes(t)
	defer database.Close()
	dir := t.TempDir()
	repo, _ := BindDocumentRepository(NewSQLDocumentStore(lanes, dir), testTenantRef("demo"))
	data := []byte("%PDF-1.4 original")
	item, err := prepareArchiveRecord(archiveTestDocument(), "a.pdf", "application/pdf", data, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := writeArchiveFile(dir, item, data); err != nil {
		t.Fatal(err)
	}
	path, _ := documentFilePathIn(dir, item)
	before, _ := os.Stat(path)
	// A complete file left by a failed metadata commit can be adopted unchanged.
	if _, err := repo.CreateGenerated(archiveTestDocument(), "a.pdf", "application/pdf", data, time.Now()); err != nil {
		t.Fatal(err)
	}
	after, _ := os.Stat(path)
	if !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("orphan was rewritten")
	}
	changed := CopyDocument(item)
	changed.AnnualStatementArchive.SHA256 = fmt.Sprintf("%x", sha256.Sum256([]byte("different")))
	if err := writeArchiveFile(dir, changed, []byte("different")); err == nil {
		t.Fatal("exclusive file was overwritten")
	}
	raw, _ := os.ReadFile(path)
	if !bytes.Equal(raw, data) {
		t.Fatal("archive bytes changed")
	}
	// A corrupt orphan fails closed and leaves no document row behind.
	broken := archiveTestDocument()
	broken.AnnualStatementArchive.RunID = "broken-run"
	prepared, err := prepareArchiveRecord(broken, "b.pdf", "application/pdf", data, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := writeArchiveFile(dir, prepared, []byte("partial")); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateGenerated(broken, "b.pdf", "application/pdf", data, time.Now()); err == nil {
		t.Fatal("corrupt orphan was adopted")
	}
	if len(repo.List()) != 1 {
		t.Fatal("failed archive wrote metadata")
	}
}
