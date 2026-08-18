package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/dbmove"
	"github.com/inspr-at/hausv-org/internal/dbtest"
	"github.com/inspr-at/hausv-org/internal/store"
)

func noEnv(string) string { return "" }

func TestMigrateDataTakesTheDSNFromTheEnvironmentOnly(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runMigrateData([]string{"--from", "/nonexistent.db"}, &stdout, &stderr, noEnv)
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("without DATABASE_URL the command must say so, got %v", err)
	}
	// There is no --dsn / --to flag, on purpose.
	for _, flag := range []string{"--to", "--dsn", "--database-url"} {
		err := runMigrateData([]string{flag, "postgres://x"}, &stdout, &stderr, noEnv)
		if err == nil || !strings.Contains(err.Error(), "flag provided but not defined") {
			t.Fatalf("%s must not exist as a flag, got %v", flag, err)
		}
	}
	err = runMigrateData([]string{}, &stdout, &stderr, noEnv)
	if err == nil || !strings.Contains(err.Error(), "--from") {
		t.Fatalf("without a source the command must ask for --from, got %v", err)
	}
	err = runMigrateData([]string{"--dry-run", "--verify-only", "--from", "x"}, &stdout, &stderr, noEnv)
	if err == nil || !strings.Contains(err.Error(), "exclude") {
		t.Fatalf("--dry-run and --verify-only must exclude each other, got %v", err)
	}
	// A source that does not exist is refused before anything opens, so a
	// typo cannot become "moved zero rows".
	err = runMigrateData([]string{"--from", filepath.Join(t.TempDir(), "typo.db")}, &stdout, &stderr,
		func(key string) string {
			if key == "DATABASE_URL" {
				return "postgres://never-dialled.invalid/x"
			}
			return ""
		})
	if err == nil || !strings.Contains(err.Error(), "typo.db") {
		t.Fatalf("a missing source must be refused by name, got %v", err)
	}
}

func TestMigrateDataUsageCarriesTheOperatorNote(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := runMigrateData([]string{"--help"}, &stdout, &stderr, noEnv); err != nil {
		t.Fatalf("--help is not a failure: %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("--help must not open anything, but wrote to stdout: %s", stdout.String())
	}
	help := stderr.String()
	for _, want := range []string{
		"BEFORE the application is switched",
		"DATABASE_URL",
		"never a flag",
		"--force",
		"--dry-run",
		"--verify-only",
		"EMPTY",
		"MISMATCH",
		"tenant_id",
		"NOBYPASSRLS",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("--help does not mention %q", want)
		}
	}
}

// The command end to end: read-only source, DSN from the environment, dry run,
// real run, verify-only, and a second real run refused because the target is
// no longer empty. The DSN must never reach the output.
func TestMigrateDataEndToEnd(t *testing.T) {
	if dbtest.Backend() != db.BackendPostgres {
		t.Skip("the mover's target is PostgreSQL: set HAUSV_STORE_TEST_POSTGRES and HAUSV_TEST_POSTGRES_DSN")
	}
	sourcePath := filepath.Join(t.TempDir(), "hausv.db")
	sqlite, err := db.Open(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	identities, err := store.EnsureTenantIdentities(context.Background(), sqlite, []store.TenantIdentity{{Slug: "jhw22", Name: "JHW 22"}, {Slug: "ww87", Name: "WW 87"}})
	if err != nil {
		t.Fatal(err)
	}
	for slug, identity := range identities {
		if _, err := sqlite.Exec(`INSERT INTO contacts(tenant_id, tenant_slug, id, active, data) VALUES($1, $2, $3, $4, $5)`,
			identity.ID, slug, "c-"+slug, true, `{"id":"c-`+slug+`","kind":"dienstleister","name":"Huber"}`); err != nil {
			t.Fatal(err)
		}
	}
	if err := sqlite.Close(); err != nil {
		t.Fatal(err)
	}

	pg, cfg := dbtest.OpenWithConfig(t)
	getenv := func(key string) string {
		if key == "DATABASE_URL" {
			return cfg.DSN
		}
		return ""
	}
	run := func(args ...string) (string, error) {
		var stdout, stderr bytes.Buffer
		err := runMigrateData(args, &stdout, &stderr, getenv)
		out := stdout.String() + stderr.String()
		if strings.Contains(out, cfg.DSN) {
			t.Fatalf("the DSN reached the output")
		}
		return out, err
	}

	out, err := run("--from", sourcePath, "--dry-run")
	if err != nil || !strings.Contains(out, "ROLLED BACK") {
		t.Fatalf("dry run: err=%v\n%s", err, out)
	}
	var contacts int
	if err := pg.QueryRow(`SELECT count(*) FROM contacts`).Scan(&contacts); err != nil || contacts != 0 {
		t.Fatalf("dry run left rows: %d err=%v", contacts, err)
	}

	out, err = run("--from", sourcePath)
	if err != nil || !strings.Contains(out, fmt.Sprintf("COMMITTED: 4 rows in %d tables", len(dbmove.TableOrder))) {
		t.Fatalf("real run: err=%v\n%s", err, out)
	}
	if !strings.Contains(out, "contacts.active: integer -> boolean") {
		t.Fatalf("report does not name the boolean conversion:\n%s", out)
	}
	if err := pg.QueryRow(`SELECT count(*) FROM contacts WHERE active`).Scan(&contacts); err != nil || contacts != 2 {
		t.Fatalf("moved contacts: %d err=%v", contacts, err)
	}

	out, err = run("--from", sourcePath, "--verify-only")
	if err != nil || !strings.Contains(out, fmt.Sprintf("verify: all %d tables match", len(dbmove.TableOrder))) {
		t.Fatalf("verify-only: err=%v\n%s", err, out)
	}

	_, err = run("--from", sourcePath)
	if !errors.Is(err, dbmove.ErrTargetNotEmpty) {
		t.Fatalf("a second run onto the moved target must be refused, got %v", err)
	}
	// The source stayed read-only and untouched: nothing wrote to it, and the
	// mover cannot even if it tried.
	reopened, err := db.OpenSQLiteReadOnly(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if _, err := reopened.Exec(`INSERT INTO app_meta(key, value) VALUES('x', 'y')`); err == nil {
		t.Fatal("a read-only source accepted a write")
	}
}
