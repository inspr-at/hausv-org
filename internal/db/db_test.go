package db

import (
	"path/filepath"
	"testing"
)

func TestOpenRunsBaselineMigration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	database, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer database.Close()

	// The baseline table exists and round-trips.
	if _, err := database.Exec("INSERT INTO app_meta(key, value) VALUES('schema', 'baseline')"); err != nil {
		t.Fatalf("insert: %v", err)
	}
	var got string
	if err := database.QueryRow("SELECT value FROM app_meta WHERE key = 'schema'").Scan(&got); err != nil {
		t.Fatalf("select: %v", err)
	}
	if got != "baseline" {
		t.Fatalf("value = %q, want baseline", got)
	}

	// The migration is recorded exactly once.
	var count int
	if err := database.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if count != 1 {
		t.Fatalf("recorded migrations = %d, want 1", count)
	}
}

func TestOpenIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")

	first, err := Open(path)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	if _, err := first.Exec("INSERT INTO app_meta(key, value) VALUES('k', 'v')"); err != nil {
		t.Fatalf("seed: %v", err)
	}
	first.Close()

	// Re-opening runs the migration runner again: it must be a no-op (no error,
	// no duplicate migration rows) and must not touch existing data.
	second, err := Open(path)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	defer second.Close()

	var migrations int
	if err := second.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&migrations); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if migrations != 1 {
		t.Fatalf("migrations after reopen = %d, want 1 (idempotent)", migrations)
	}
	var value string
	if err := second.QueryRow("SELECT value FROM app_meta WHERE key = 'k'").Scan(&value); err != nil || value != "v" {
		t.Fatalf("existing data lost after reopen: value=%q err=%v", value, err)
	}
}

func TestOpenRequiresPath(t *testing.T) {
	if _, err := Open("   "); err == nil {
		t.Fatal("empty path must error")
	}
}

func TestForeignKeysEnforced(t *testing.T) {
	// Proves the foreign_keys pragma is live on the connection (it is per-conn,
	// which is why we set it via the DSN, not a one-off Exec).
	path := filepath.Join(t.TempDir(), "fk.db")
	database, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer database.Close()

	if _, err := database.Exec(`CREATE TABLE parent(id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("create parent: %v", err)
	}
	if _, err := database.Exec(`CREATE TABLE child(id INTEGER PRIMARY KEY, parent_id INTEGER REFERENCES parent(id))`); err != nil {
		t.Fatalf("create child: %v", err)
	}
	if _, err := database.Exec(`INSERT INTO child(id, parent_id) VALUES (1, 999)`); err == nil {
		t.Fatal("insert violating a foreign key must fail with foreign_keys ON")
	}
}
