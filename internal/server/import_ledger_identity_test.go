package server

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	appdb "github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/dbtest"
	"github.com/inspr-at/hausv-org/internal/integrations"
	storepkg "github.com/inspr-at/hausv-org/internal/store"
)

// SQLite can still contain pre-identity rows. They are already-imported
// evidence, not a reason to replay effects or cross onto a maintenance lane.
func TestImportLedgerPreservesLegacyCompletion(t *testing.T) {
	for _, format := range []string{string(integrations.FormatCAMT053), string(integrations.FormatEBInterface)} {
		t.Run(format, func(t *testing.T) {
			a, database, refs := ledgerTestApp(t, "demo")
			if dbtest.RollbackWindowClosed(t, database) {
				return
			}
			digest := strings.Repeat("d", 64)
			seedLegacyLedgerRow(t, database, "demo", format, digest)
			ledger := storepkg.NewImportLedger(a.tenantDB)
			key := storepkg.ImportKey{Format: format}
			complete, err := ledger.Completed(t.Context(), refs["demo"], key, digest)
			if err != nil || complete {
				t.Fatalf("unbackfilled row should be invisible: %v, %v", complete, err)
			}
			already, err := ledger.Apply(t.Context(), refs["demo"], key, digest, func(*storepkg.ImportTx) error {
				t.Fatal("legacy file was replayed")
				return nil
			})
			if err != nil || !already {
				t.Fatalf("legacy replay = %v, %v", already, err)
			}
			assertLedgerAppliedBy(t, database, digest, "old@example.com")
		})
	}
}

// Fixture setup now goes through the same store transaction as production;
// there is deliberately no handler-owned SQL ledger recorder.
func recordTestImportLedger(a *app, tenant storepkg.TenantRef, format, digest, version string, counts storepkg.ImportCounts) error {
	_, err := storepkg.NewImportLedger(a.tenantDB).Apply(context.Background(), tenant, storepkg.ImportKey{
		Format: format, SourceVersion: version, AppliedBy: "manager@example.com",
	}, digest, func(tx *storepkg.ImportTx) error { tx.Counts = counts; return nil })
	return err
}

// TestImportLedgerOrphanCannotExistOnPostgres is what became of the test that
// pinned WHY the ledger write sits on Unscoped(store.HealOrphanReason): under
// migration 0003 the same upsert on the tenant lane was refused by the policy
// the moment it landed on a row without an identity, and that refusal was
// measured here. Migration 0006 removed the row itself — tenant_id is NOT NULL
// on integration_imports, so the legacy ledger row cannot be seeded from ANY
// lane, and there is no orphan left for the maintenance lane to be required
// for.
//
// So this now pins the stronger property, and with it the fact that the ledger
// write CAN go back to For(tenant): the seed is refused by the column (SQLSTATE
// 23502) even on the maintenance lane, and the tenant lane's own upsert onto a
// row it wrote itself succeeds. Moving the write is a follow-up made against
// this test, not part of the flip.
func TestImportLedgerOrphanCannotExistOnPostgres(t *testing.T) {
	if dbtest.Backend() != appdb.BackendPostgres {
		t.Skip("row-level security and the NOT NULL flip exist only on PostgreSQL; SQLite has one pool, no policy, and an open rollback window")
	}
	digest := strings.Repeat("e", 64)
	a, database, refs := ledgerTestApp(t, "demo")
	demo := refs["demo"]

	// database is the maintenance lane, the one handle the policy lets through:
	// what refuses this is the column, not the policy.
	_, err := database.Exec(
		`INSERT INTO integration_imports(tenant_slug, format, file_digest, source_version, applied_at, applied_by,
			assigned, changed, unclear, rejected)
		 VALUES($1, $2, $3, 'v1', '2026-01-01T00:00:00Z', 'old@example.com', 0, 0, 0, 0)`,
		"demo", string(integrations.FormatCAMT053), digest)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23502" {
		t.Fatalf("a legacy ledger row without an identity was accepted (err=%v); migration 0006 says it cannot exist", err)
	}

	// And the tenant lane can do the ledger's own write, twice, on its own row.
	upsert := `INSERT INTO integration_imports(
			tenant_id, tenant_slug, format, file_digest, source_version, applied_at, applied_by,
			assigned, changed, unclear, rejected
		) VALUES($1, $2, $3, $4, 'v2', '2026-02-01T00:00:00Z', 'new@example.com', 1, 0, 0, 0)
		ON CONFLICT(tenant_slug, format, file_digest) DO UPDATE SET
		  tenant_id=coalesce(integration_imports.tenant_id, excluded.tenant_id)`
	for attempt := 1; attempt <= 2; attempt++ {
		if _, err := a.tenantDB.For(demo).Exec(upsert, demo.ID, demo.Slug, string(integrations.FormatCAMT053), digest); err != nil {
			t.Fatalf("attempt %d: the tenant lane cannot write its own ledger row: %v", attempt, err)
		}
	}
	assertLedgerRowOwned(t, database, string(integrations.FormatCAMT053), digest, demo.ID)
}

// ledgerTestApp returns an app whose ledger runs on real lanes over the dbtest
// engine, the maintenance view for fixtures and assertions (deliberately
// outside every TENANT lane, so an assertion cannot be fooled by the lane it is
// checking; see dbtest.MaintenanceView for why it is no longer the pool), and
// the tenant references the database minted for the given slugs.
func ledgerTestApp(t *testing.T, slugs ...string) (*app, *sql.DB, map[string]storepkg.TenantRef) {
	t.Helper()
	pool, cfg := dbtest.OpenWithConfig(t)
	scoped, err := appdb.NewScoped(cfg, pool)
	if err != nil {
		t.Fatalf("open scoped test lanes: %v", err)
	}
	t.Cleanup(func() { _ = scoped.Close() })
	database := dbtest.MaintenanceView(t, scoped)
	configured := make([]storepkg.TenantIdentity, 0, len(slugs))
	for _, slug := range slugs {
		configured = append(configured, storepkg.TenantIdentity{Slug: slug, Name: slug})
	}
	// The boot path runs on the pool, exactly as newApp does.
	identities, err := storepkg.EnsureTenantIdentities(context.Background(), pool, configured)
	if err != nil {
		t.Fatalf("ensure tenant identities: %v", err)
	}
	refs := map[string]storepkg.TenantRef{}
	for _, slug := range slugs {
		identity, ok := identities[slug]
		if !ok || !identity.Ref().Valid() {
			t.Fatalf("tenant identity missing for %s", slug)
		}
		refs[slug] = identity.Ref()
	}
	return &app{tenantDB: storepkg.NewTenantDB(scoped), scopedDB: scoped}, database, refs
}

func seedLegacyLedgerRow(t *testing.T, database *sql.DB, slug, format, digest string) {
	t.Helper()
	if _, err := database.Exec(
		`INSERT INTO integration_imports(tenant_slug, format, file_digest, source_version, applied_at, applied_by,
			assigned, changed, unclear, rejected)
		 VALUES($1, $2, $3, 'v1', '2026-01-01T00:00:00Z', 'old@example.com', 0, 0, 0, 0)`,
		slug, format, digest); err != nil {
		t.Fatalf("seed legacy ledger row: %v", err)
	}
}

func assertLedgerRowOwned(t *testing.T, database *sql.DB, format, digest, wantOwner string) {
	t.Helper()
	var rows int
	if err := database.QueryRow(
		`SELECT COUNT(*) FROM integration_imports WHERE format = $1 AND file_digest = $2`,
		format, digest).Scan(&rows); err != nil {
		t.Fatalf("count ledger rows: %v", err)
	}
	if rows != 1 {
		t.Fatalf("ledger rows = %d, want 1", rows)
	}
	var owner sql.NullString
	if err := database.QueryRow(
		`SELECT tenant_id FROM integration_imports WHERE format = $1 AND file_digest = $2`,
		format, digest).Scan(&owner); err != nil {
		t.Fatalf("read ledger identity: %v", err)
	}
	if !owner.Valid {
		t.Fatal("the ledger row still has no tenant identity, so the read that filters on it never will find it")
	}
	if owner.String != wantOwner {
		t.Fatalf("ledger row adopted by %q, want the tenant that wrote it %q", owner.String, wantOwner)
	}
}

func assertLedgerAppliedBy(t *testing.T, database *sql.DB, digest, want string) {
	t.Helper()
	var appliedBy string
	if err := database.QueryRow(
		`SELECT applied_by FROM integration_imports WHERE file_digest = $1`, digest).Scan(&appliedBy); err != nil {
		t.Fatalf("read applied_by: %v", err)
	}
	if appliedBy != want {
		t.Errorf("applied_by = %q, want %q: the heal must not rewrite when the file was first applied", appliedBy, want)
	}
}
