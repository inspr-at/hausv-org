package server

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	appdb "github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/dbtest"
	"github.com/inspr-at/hausv-org/internal/integrations"
	storepkg "github.com/inspr-at/hausv-org/internal/store"
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
// The ledger now runs on the lane seam: the read on For(tenant), the write on
// Unscoped(store.HealOrphanReason). On PostgreSQL that is the difference between
// the heal working and the heal being refused — the orphan is unreachable from
// the tenant lane under migration 0003 — so this test runs on whichever engine
// dbtest selects: SQLite by default, and the real policy when
// HAUSV_STORE_TEST_POSTGRES is set. TestImportLedgerHealRequiresTheMaintenanceLane
// below is the PostgreSQL-only half that shows the tenant lane cannot do it.
//
// Where it is blind: it drives the two ledger functions directly rather than a
// full upload through the HTTP handler, so it proves the SQL heals and says
// nothing about the surrounding import flow (TestCAMT053ImportLedgerIsDurableAndTenantBound
// and the eb-interface equivalent cover that).
func TestImportLedgerHealsRowsLeftWithoutAnIdentity(t *testing.T) {
	digest := strings.Repeat("d", 64)

	t.Run("camt053", func(t *testing.T) {
		a, database, refs := ledgerTestApp(t, "demo")
		demo := refs["demo"]
		seedLegacyLedgerRow(t, database, "demo", string(integrations.FormatCAMT053), digest)

		if a.paymentImportAlreadyApplied(demo, digest) {
			t.Fatal("fixture does not reproduce the state: the legacy row is invisible to the tenant_id read")
		}
		preview := camtImportPreview{FileDigest: digest, SourceVersion: "2019/camt.053.001.08"}
		if err := a.recordPaymentImportLedger(demo, "manager@example.com", preview,
			unitPaymentImportReport{Assigned: 1}); err != nil {
			t.Fatalf("record ledger: %v", err)
		}

		assertLedgerRowOwned(t, database, string(integrations.FormatCAMT053), digest, demo.ID)
		if !a.paymentImportAlreadyApplied(demo, digest) {
			t.Error("the ledger still cannot see its own row: every re-upload of this file is applied again")
		}
		// And the columns DO NOTHING protects are untouched: the ledger says when
		// the file was FIRST applied.
		assertLedgerAppliedBy(t, database, digest, "old@example.com")
	})

	t.Run("ebinterface", func(t *testing.T) {
		a, database, refs := ledgerTestApp(t, "demo")
		demo := refs["demo"]
		seedLegacyLedgerRow(t, database, "demo", string(integrations.FormatEBInterface), digest)

		if a.ebInterfaceImportAlreadyStored(demo, digest) {
			t.Fatal("fixture does not reproduce the state: the legacy row is invisible to the tenant_id read")
		}
		preview := ebInterfaceImportPreview{FileDigest: digest, SourceVersion: "ebInterface 6.1"}
		if err := a.recordEBInterfaceImportLedger(demo, "manager@example.com", preview); err != nil {
			t.Fatalf("record ledger: %v", err)
		}

		assertLedgerRowOwned(t, database, string(integrations.FormatEBInterface), digest, demo.ID)
		if !a.ebInterfaceImportAlreadyStored(demo, digest) {
			t.Error("the ledger still cannot see its own row: every re-upload of this file is stored again")
		}
		assertLedgerAppliedBy(t, database, digest, "old@example.com")
	})
}

// TestImportLedgerHealRequiresTheMaintenanceLane is why the ledger write is on
// Unscoped(store.HealOrphanReason) and not on For(tenant): the same upsert, on
// the tenant lane, is refused by the row-level policy the moment it lands on a
// row without an identity. Pinned as behaviour so that moving the write back
// onto the tenant lane — which is what the flip will eventually do, once no
// orphan can exist — is a decision made against a failing test rather than a
// silent regression on the engine that has RLS.
func TestImportLedgerHealRequiresTheMaintenanceLane(t *testing.T) {
	if dbtest.Backend() != appdb.BackendPostgres {
		t.Skip("row-level security exists only on PostgreSQL; SQLite has one pool and no policy")
	}
	digest := strings.Repeat("e", 64)
	a, database, refs := ledgerTestApp(t, "demo")
	demo := refs["demo"]
	seedLegacyLedgerRow(t, database, "demo", string(integrations.FormatCAMT053), digest)

	_, err := a.tenantDB.For(demo).Exec(
		`INSERT INTO integration_imports(
			tenant_id, tenant_slug, format, file_digest, source_version, applied_at, applied_by,
			assigned, changed, unclear, rejected
		) VALUES($1, $2, $3, $4, 'v2', '2026-02-01T00:00:00Z', 'new@example.com', 1, 0, 0, 0)
		ON CONFLICT(tenant_slug, format, file_digest) DO UPDATE SET
		  tenant_id=coalesce(integration_imports.tenant_id, excluded.tenant_id)`,
		demo.ID, demo.Slug, string(integrations.FormatCAMT053), digest)
	if err == nil {
		t.Fatal("the tenant lane adopted an orphan ledger row; if migration 0003 now lets a tenant lane reach a NULL tenant_id row, the ledger write can go back to For(tenant)")
	}
	if !strings.Contains(err.Error(), "row-level security") {
		t.Fatalf("the tenant lane failed for a different reason than the policy: %v", err)
	}
}

// ledgerTestApp returns an app whose ledger runs on real lanes over the dbtest
// engine, the pool for fixtures and assertions (deliberately outside every lane,
// so an assertion cannot be fooled by the lane it is checking), and the tenant
// references the database minted for the given slugs.
func ledgerTestApp(t *testing.T, slugs ...string) (*app, *sql.DB, map[string]storepkg.TenantRef) {
	t.Helper()
	database, cfg := dbtest.OpenWithConfig(t)
	scoped, err := appdb.NewScoped(cfg, database)
	if err != nil {
		t.Fatalf("open scoped test lanes: %v", err)
	}
	t.Cleanup(func() { _ = scoped.Close() })
	configured := make([]storepkg.TenantIdentity, 0, len(slugs))
	for _, slug := range slugs {
		configured = append(configured, storepkg.TenantIdentity{Slug: slug, Name: slug})
	}
	identities, err := storepkg.EnsureTenantIdentities(context.Background(), database, configured)
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
