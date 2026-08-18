package db

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenSQLiteReadOnlyRefusesWritesAndMissingFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.db")
	if _, err := OpenSQLiteReadOnly(path); err == nil {
		t.Fatal("a missing source must be refused, not created")
	}

	writable, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := writable.Exec(`INSERT INTO app_meta(key, value) VALUES('k', 'v')`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	// Left open on purpose: at cutover the file may still have its WAL and
	// -shm beside it, and the read-only opener must cope with that.
	defer writable.Close()

	readOnly, err := OpenSQLiteReadOnly(path)
	if err != nil {
		t.Fatalf("open read-only: %v", err)
	}
	defer readOnly.Close()
	var value string
	if err := readOnly.QueryRow(`SELECT value FROM app_meta WHERE key='k'`).Scan(&value); err != nil || value != "v" {
		t.Fatalf("read-only read: value=%q err=%v", value, err)
	}
	if _, err := readOnly.Exec(`INSERT INTO app_meta(key, value) VALUES('x', 'y')`); err == nil {
		t.Fatal("a read-only source accepted a write")
	}
	if _, err := readOnly.Exec(`DELETE FROM app_meta`); err == nil {
		t.Fatal("a read-only source accepted a delete")
	}
	var rows int
	if err := writable.QueryRow(`SELECT count(*) FROM app_meta`).Scan(&rows); err != nil || rows != 1 {
		t.Fatalf("the refused writes changed the file: rows=%d err=%v", rows, err)
	}
}

func TestMigrationNamesMatchTheEmbeddedFiles(t *testing.T) {
	lite := SQLiteMigrationNames()
	pg := PostgresMigrationNames()
	if len(lite) < 33 || len(pg) < 5 {
		t.Fatalf("migration lists too short: sqlite %d, postgres %d", len(lite), len(pg))
	}
	for _, list := range [][]string{lite, pg} {
		for i, name := range list {
			if !strings.HasSuffix(name, ".sql") {
				t.Errorf("%q is not a migration file name", name)
			}
			if i > 0 && list[i-1] >= name {
				t.Errorf("%q is not sorted after %q", name, list[i-1])
			}
		}
	}
	if lite[0] != "0001_baseline.sql" || pg[0] != "0001_target_schema.sql" {
		t.Fatalf("first migrations = %q / %q", lite[0], pg[0])
	}
}
