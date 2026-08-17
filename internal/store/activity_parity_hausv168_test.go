package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/dbtest"
)

// TestActivityStorageParity runs the JSON and SQLite backends through identical
// assertions — the parity oracle that lets the backend be swapped with
// confidence (HAUSV-168). Any behaviour the two disagree on fails here.
func TestActivityStorageParity(t *testing.T) {
	backends := map[string]func(t *testing.T) ActivityStorage{
		"json": func(t *testing.T) ActivityStorage {
			s, err := NewActivityStore(filepath.Join(t.TempDir(), "activity.json"))
			if err != nil {
				t.Fatalf("json store: %v", err)
			}
			return s
		},
		"sqlite": func(t *testing.T) ActivityStorage {
			database := dbtest.Open(t)
			t.Cleanup(func() { database.Close() })
			return NewSQLActivityStore(database)
		},
	}

	for name, build := range backends {
		t.Run(name, func(t *testing.T) {
			s := build(t)
			when := time.Date(2026, 7, 20, 8, 30, 0, 123456789, time.UTC)

			if _, ok := s.Get("nobody@example.com"); ok {
				t.Fatal("unknown email must not be found")
			}

			if err := s.Touch("Person@Example.com", when, "email"); err != nil {
				t.Fatalf("touch: %v", err)
			}
			rec, ok := s.Get("person@example.com") // normalized lookup
			if !ok {
				t.Fatal("touched email must be found via its normalized form")
			}
			if !rec.LastLogin.Equal(when) {
				t.Fatalf("last_login = %v, want %v", rec.LastLogin, when)
			}
			if rec.AuthMethod != "email" {
				t.Fatalf("auth_method = %q, want email", rec.AuthMethod)
			}

			// Touch is an upsert: the newer login overwrites.
			later := when.Add(time.Hour)
			if err := s.Touch("person@example.com", later, "oidc"); err != nil {
				t.Fatalf("touch again: %v", err)
			}
			rec, _ = s.Get("person@example.com")
			if !rec.LastLogin.Equal(later) || rec.AuthMethod != "oidc" {
				t.Fatalf("upsert failed: %+v", rec)
			}

			// Empty email is a no-op and never stored.
			if err := s.Touch("", when, "email"); err != nil {
				t.Fatalf("empty touch: %v", err)
			}
			if _, ok := s.Get(""); ok {
				t.Fatal("empty email must not be found")
			}
		})
	}
}

// TestSQLActivityImportFromJSON proves the per-store JSON→SQLite import is
// lossless and idempotent (the data-migration leg for HAUSV-170).
func TestSQLActivityImportFromJSON(t *testing.T) {
	jsonStore, err := NewActivityStore(filepath.Join(t.TempDir(), "activity.json"))
	if err != nil {
		t.Fatalf("json store: %v", err)
	}
	a := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	b := time.Date(2026, 6, 7, 8, 9, 10, 0, time.UTC)
	_ = jsonStore.Touch("a@example.com", a, "email")
	_ = jsonStore.Touch("b@example.com", b, "oidc")

	database := dbtest.Open(t)
	defer database.Close()
	sqlStore := NewSQLActivityStore(database)

	// Run the import twice: it must be idempotent.
	for i := 0; i < 2; i++ {
		if err := sqlStore.ImportActivity(jsonStore); err != nil {
			t.Fatalf("import %d: %v", i, err)
		}
	}
	for email, want := range map[string]ActivityRecord{
		"a@example.com": {LastLogin: a, AuthMethod: "email"},
		"b@example.com": {LastLogin: b, AuthMethod: "oidc"},
	} {
		got, ok := sqlStore.Get(email)
		if !ok || !got.LastLogin.Equal(want.LastLogin) || got.AuthMethod != want.AuthMethod {
			t.Fatalf("%s: got %+v ok=%v, want %+v", email, got, ok, want)
		}
	}
}
