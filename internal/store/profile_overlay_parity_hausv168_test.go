package store

import (
	"path/filepath"
	"testing"
)

// TestProfileOverlayStorageParity runs the JSON and SQLite backends through
// identical assertions (HAUSV-168).
func TestProfileOverlayStorageParity(t *testing.T) {
	backends := map[string]func(t *testing.T) ProfileOverlayStorage{
		"json": func(t *testing.T) ProfileOverlayStorage {
			s, err := NewProfileOverlayStore(filepath.Join(t.TempDir(), "profile.json"))
			if err != nil {
				t.Fatalf("json store: %v", err)
			}
			return s
		},
		"sqlite": func(t *testing.T) ProfileOverlayStorage {
			database, lanes := testLanes(t)
			t.Cleanup(func() { database.Close() })
			return NewSQLProfileOverlayStore(lanes)
		},
	}

	for name, build := range backends {
		t.Run(name, func(t *testing.T) {
			s := build(t)

			if _, ok := s.Get("nobody@example.com"); ok {
				t.Fatal("unknown email must not be found")
			}
			if err := s.Set("", ProfileOverlay{FirstName: "X"}); err == nil {
				t.Fatal("empty email must error")
			}

			if err := s.Set("Person@Example.com", ProfileOverlay{
				Title: "Dr.", FirstName: "Resi", LastName: "Dent", Phone: "+43 1 234", DirectoryOptIn: true,
			}); err != nil {
				t.Fatalf("set: %v", err)
			}
			got, ok := s.Get("person@example.com") // normalized lookup
			if !ok {
				t.Fatal("set overlay must be found via normalized email")
			}
			if got.Title != "Dr." || got.FirstName != "Resi" || got.LastName != "Dent" || got.Phone != "+43 1 234" || !got.DirectoryOptIn {
				t.Fatalf("round-trip mismatch: %+v", got)
			}
			if got.UpdatedAt.IsZero() {
				t.Fatal("Set must stamp UpdatedAt")
			}

			// Set overwrites; DirectoryOptIn can go back to false.
			if err := s.Set("person@example.com", ProfileOverlay{FirstName: "Neu"}); err != nil {
				t.Fatalf("overwrite: %v", err)
			}
			got, _ = s.Get("person@example.com")
			if got.FirstName != "Neu" || got.Title != "" || got.DirectoryOptIn {
				t.Fatalf("overwrite mismatch: %+v", got)
			}
		})
	}
}

func TestSQLProfileOverlayImportFromJSON(t *testing.T) {
	jsonStore, err := NewProfileOverlayStore(filepath.Join(t.TempDir(), "profile.json"))
	if err != nil {
		t.Fatalf("json store: %v", err)
	}
	_ = jsonStore.Set("a@example.com", ProfileOverlay{FirstName: "Aa", DirectoryOptIn: true})
	_ = jsonStore.Set("b@example.com", ProfileOverlay{Title: "Mag.", LastName: "Bee"})

	database, lanes := testLanes(t)
	defer database.Close()
	sqlStore := NewSQLProfileOverlayStore(lanes)

	// A newer write already in SQLite must NOT be clobbered by a re-import.
	if err := sqlStore.Set("a@example.com", ProfileOverlay{FirstName: "SQLITE-NEWER"}); err != nil {
		t.Fatalf("pre-set: %v", err)
	}
	for i := 0; i < 2; i++ {
		if err := sqlStore.ImportOverlays(jsonStore); err != nil {
			t.Fatalf("import %d: %v", i, err)
		}
	}
	if got, _ := sqlStore.Get("a@example.com"); got.FirstName != "SQLITE-NEWER" {
		t.Fatalf("import clobbered a newer SQLite write: %+v", got)
	}
	if got, ok := sqlStore.Get("b@example.com"); !ok || got.Title != "Mag." || got.LastName != "Bee" {
		t.Fatalf("import missed a JSON-only record: %+v ok=%v", got, ok)
	}
}
