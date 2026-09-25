package demo

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/dbtest"
	"github.com/inspr-at/hausv-org/internal/store"
)

func TestDemoDocumentsReseedRestoresFilesWithoutDuplicates(t *testing.T) {
	database, config := dbtest.OpenWithConfig(t)
	scoped, err := db.NewScoped(config, database)
	if err != nil {
		t.Fatal(err)
	}
	defer scoped.Close()
	lanes := store.NewTenantDB(scoped)
	dir := t.TempDir()
	seedDir := "../../scripts/demo/seed"
	var houses []seedHouse
	if err := readJSON(filepath.Join(seedDir, "houses.json"), &houses); err != nil {
		t.Fatal(err)
	}
	fixtures, err := loadDocumentFixture(seedDir, dir, houses)
	if err != nil {
		t.Fatal(err)
	}
	documented := 0
	for _, house := range houses {
		if house.Slug != "musterstrasse-12" {
			documented++
		}
	}
	if len(fixtures) != documented*6 || len(houses) != documented+1 {
		t.Fatalf("documents=%d houses=%d", len(fixtures), len(houses))
	}
	for _, reset := range []bool{false, false, true} {
		if _, err := Load(t.Context(), database, seedDir, SeedOptions{DocumentDir: dir, Reset: reset, DiscardAnnualStatements: reset}); err != nil {
			t.Fatal(err)
		}
		identities, err := store.EnsureTenantIdentities(t.Context(), database, nil)
		if err != nil {
			t.Fatal(err)
		}
		for _, house := range houses {
			repository, ok := store.BindDocumentRepository(store.NewSQLDocumentStore(lanes, dir), identities[house.Slug].Ref())
			if !ok {
				t.Fatal("document repository unavailable")
			}
			public, board := 0, 0
			for _, item := range repository.List() {
				if !strings.HasPrefix(item.ID, "demo-document-") {
					continue
				}
				if strings.Contains(strings.ToLower(item.Filename), "demo") {
					t.Fatalf("fixture prefix visible in filename: %s", item.Filename)
				}
				if !item.Current || item.Version != 1 || item.SeriesID != item.ID {
					t.Fatalf("unexpected version: %+v", item)
				}
				switch item.Visibility {
				case store.DocumentVisibilityAllResidents:
					public++
				case store.DocumentVisibilityBoardOnly:
					board++
				default:
					t.Fatalf("unexpected visibility: %s", item.Visibility)
				}
				path, ok := repository.FilePath(item)
				if !ok {
					t.Fatal("file path unavailable")
				}
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if int64(len(data)) != item.Size || len(data) > 5000 || !bytes.HasPrefix(data, []byte("%PDF-")) || !bytes.Contains(data, []byte("xref\n")) {
					t.Fatalf("invalid PDF %s", path)
				}
				for _, fixture := range fixtures {
					if fixture.House == house.Slug && fixture.ID == item.ID && fixture.SHA256 != fmt.Sprintf("%x", sha256.Sum256(data)) {
						t.Fatal("PDF digest changed")
					}
				}
			}
			if house.Slug == "musterstrasse-12" {
				if public != 0 || board != 0 {
					t.Fatalf("%s: public=%d board=%d", house.Slug, public, board)
				}
				continue
			}
			if public != 5 || board != 1 {
				t.Fatalf("%s: public=%d board=%d", house.Slug, public, board)
			}
		}
		// Simulate a lost file without deleting: the next load must restore it.
		if err := os.Rename(filepath.Join(dir, houses[0].Slug, fixtures[0].ID+".pdf"), filepath.Join(t.TempDir(), "saved.pdf")); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDemoDocumentsRejectInvalidFixturesBeforeLoading(t *testing.T) {
	seedDir := "../../scripts/demo/seed"
	var houses []seedHouse
	if err := readJSON(filepath.Join(seedDir, "houses.json"), &houses); err != nil {
		t.Fatal(err)
	}
	if _, err := loadDocumentFixture(seedDir, "", houses); err == nil {
		t.Fatal("DocumentDir must be required")
	}
	documents, err := loadDocumentFixture(seedDir, t.TempDir(), houses)
	if err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*seedDocument){
		func(d *seedDocument) { d.PDF = []byte("not a PDF") },
		func(d *seedDocument) { d.SHA256 = "wrong" },
		func(d *seedDocument) { d.Filename = "../outside.pdf" },
		func(d *seedDocument) { d.House = "unknown" },
		func(d *seedDocument) { d.Visibility = "unknown" },
	} {
		item := documents[0]
		mutate(&item)
		dir := t.TempDir()
		writeFixtureJSONHAUSV711(t, dir, []seedDocument{item})
		if _, err := loadDocumentFixture(dir, t.TempDir(), houses); err == nil {
			t.Fatal("invalid document fixture accepted")
		}
	}
}

func writeFixtureJSONHAUSV711(t *testing.T, dir string, items []seedDocument) {
	t.Helper()
	data, err := json.Marshal(items)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "documents.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
}
