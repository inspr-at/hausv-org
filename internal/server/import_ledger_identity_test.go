package server

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	appdb "github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/integrations"
)

// TestImportLedgerHealsRowsLeftWithoutAnIdentity is the one tenant-scoped upsert
// that could not heal, reproduced.
//
// integration_imports is written with ON CONFLICT ... DO NOTHING, because the
// ledger records when a file was FIRST applied and a re-upload must not rewrite
// that. Its dedup read, however, was switched onto tenant_id. A ledger row
// written by the previous release carries a slug and no identity, so:
//
//	the read cannot see it            -> the file counts as never imported
//	the write conflicts and does nothing -> the row never gets its identity
//
// which is not a transient gap but a permanent one: every re-upload of that file
// is applied again, and the boot-time backfill is the only thing that could ever
// repair it. Every other tenant-scoped upsert in the product heals the row it
// lands on; this one had to as well, without touching the columns DO NOTHING is
// there to protect.
//
// Where it is blind: it drives the two ledger functions directly rather than a
// full upload through the HTTP handler, so it proves the SQL heals and says
// nothing about the surrounding import flow (TestCAMT053ImportLedgerIsDurableAndTenantBound
// and the eb-interface equivalent cover that). It runs on SQLite only, like the
// rest of this package's tests.
func TestImportLedgerHealsRowsLeftWithoutAnIdentity(t *testing.T) {
	digest := strings.Repeat("d", 64)

	t.Run("camt053", func(t *testing.T) {
		database := ledgerTestDB(t)
		a := &app{db: database}
		seedLegacyLedgerRow(t, database, "demo", string(integrations.FormatCAMT053), digest)

		if a.paymentImportAlreadyApplied("demo", digest) {
			t.Fatal("fixture does not reproduce the state: the legacy row is invisible to the tenant_id read")
		}
		preview := camtImportPreview{FileDigest: digest, SourceVersion: "2019/camt.053.001.08"}
		if err := a.recordPaymentImportLedger("demo", "manager@example.com", preview,
			unitPaymentImportReport{Assigned: 1}); err != nil {
			t.Fatalf("record ledger: %v", err)
		}

		assertLedgerRowOwned(t, database, string(integrations.FormatCAMT053), digest)
		if !a.paymentImportAlreadyApplied("demo", digest) {
			t.Error("the ledger still cannot see its own row: every re-upload of this file is applied again")
		}
		// And the columns DO NOTHING protects are untouched: the ledger says when
		// the file was FIRST applied.
		assertLedgerAppliedBy(t, database, digest, "old@example.com")
	})

	t.Run("ebinterface", func(t *testing.T) {
		database := ledgerTestDB(t)
		a := &app{db: database}
		seedLegacyLedgerRow(t, database, "demo", string(integrations.FormatEBInterface), digest)

		if a.ebInterfaceImportAlreadyStored("demo", digest) {
			t.Fatal("fixture does not reproduce the state: the legacy row is invisible to the tenant_id read")
		}
		preview := ebInterfaceImportPreview{FileDigest: digest, SourceVersion: "ebInterface 6.1"}
		if err := a.recordEBInterfaceImportLedger("demo", "manager@example.com", preview); err != nil {
			t.Fatalf("record ledger: %v", err)
		}

		assertLedgerRowOwned(t, database, string(integrations.FormatEBInterface), digest)
		if !a.ebInterfaceImportAlreadyStored("demo", digest) {
			t.Error("the ledger still cannot see its own row: every re-upload of this file is stored again")
		}
		assertLedgerAppliedBy(t, database, digest, "old@example.com")
	})
}

func ledgerTestDB(t *testing.T) *sql.DB {
	t.Helper()
	database, err := appdb.Open(filepath.Join(t.TempDir(), "hausv.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}

func seedLegacyLedgerRow(t *testing.T, database *sql.DB, slug, format, digest string) {
	t.Helper()
	if _, err := database.Exec(
		`INSERT INTO integration_imports(tenant_slug, format, file_digest, source_version, applied_at, applied_by,
			assigned, changed, unclear, rejected)
		 VALUES(?, ?, ?, 'v1', '2026-01-01T00:00:00Z', 'old@example.com', 0, 0, 0, 0)`,
		slug, format, digest); err != nil {
		t.Fatalf("seed legacy ledger row: %v", err)
	}
}

func assertLedgerRowOwned(t *testing.T, database *sql.DB, format, digest string) {
	t.Helper()
	var rows int
	if err := database.QueryRow(
		`SELECT COUNT(*) FROM integration_imports WHERE format = ? AND file_digest = ?`,
		format, digest).Scan(&rows); err != nil {
		t.Fatalf("count ledger rows: %v", err)
	}
	if rows != 1 {
		t.Fatalf("ledger rows = %d, want 1", rows)
	}
	var owner sql.NullString
	if err := database.QueryRow(
		`SELECT tenant_id FROM integration_imports WHERE format = ? AND file_digest = ?`,
		format, digest).Scan(&owner); err != nil {
		t.Fatalf("read ledger identity: %v", err)
	}
	if !owner.Valid {
		t.Fatal("the ledger row still has no tenant identity, so the read that filters on it never will find it")
	}
}

func assertLedgerAppliedBy(t *testing.T, database *sql.DB, digest, want string) {
	t.Helper()
	var appliedBy string
	if err := database.QueryRow(
		`SELECT applied_by FROM integration_imports WHERE file_digest = ?`, digest).Scan(&appliedBy); err != nil {
		t.Fatalf("read applied_by: %v", err)
	}
	if appliedBy != want {
		t.Errorf("applied_by = %q, want %q: the heal must not rewrite when the file was first applied", appliedBy, want)
	}
}
