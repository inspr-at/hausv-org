package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/inspr-at/hausv-org/internal/tenantid"
	"github.com/inspr-at/hausv-org/internal/textutil"
)

// This file is the one place that knows how a legacy slug becomes an immutable
// tenant_id, and the one place that can state — by counting, not by assertion —
// that no tenant-bound row is missing its identity.
//
// It exists because migration 0033 could not do the job. That migration runs
// inside db.OpenConfig, which is called BEFORE EnsureTenantIdentities has put a
// single row in `tenant`, so its correlated backfill subquery resolved to NULL
// for every row it touched. A NULL there is silent: the UPDATE succeeds, the
// migration is recorded as applied, and every row is left without an identity.
// Switching the query layer onto tenant_id in that state would have hidden the
// entire database behind a filter that matches nothing.
//
// It is deliberately in two steps — CANONICALISE, then LINK — because linking
// alone cannot be made safe. A tenant whose slug is stored under two spellings
// has two candidates for every logical key, only one of which reads ever
// matched, and every rule for picking between them is a guess: three successive
// attempts to encode one (quarantine, ownership bookkeeping, exact-row UPDATEs
// with an affected-count guard) each fixed the reported defect and introduced a
// new one in the same file. Rewriting the spelling first removes the choice
// instead of automating it. After step one there is exactly one spelling per
// tenant, so step two is unambiguous by construction and needs no rules at all.

// tenantIDTable names one tenant-bound table and the legacy column its identity
// is derived from while both columns coexist.
type tenantIDTable struct {
	name string
	// slugColumn is `tenant_slug` for the tables that carry the tenant label,
	// and `slug` for the four onboarding tables whose own primary key IS the
	// tenant label (migrations 0027-0030).
	slugColumn string
	// preTenant marks a table whose rows legitimately exist before any tenant
	// identity does. Only home_reservations qualifies: a reservation is the
	// request for a house, made before the house exists, and its identity is
	// minted at activation. Rows there may carry a NULL tenant_id forever if the
	// reservation is never activated, so it is excluded from the completeness
	// check and no identity is invented from its slug.
	preTenant bool
}

// tenantIDTables is the authoritative list of tables migration 0033 gave a
// tenant_id column. It matches expectedTenantTables in internal/db/postgres_test.go;
// the two are cross-checked by TestTenantIDTablesMatchTheMigratedSchema.
var tenantIDTables = []tenantIDTable{
	{name: "announcement_reads", slugColumn: "tenant_slug"},
	{name: "announcements", slugColumn: "tenant_slug"},
	{name: "attachments", slugColumn: "tenant_slug"},
	{name: "ballots", slugColumn: "tenant_slug"},
	{name: "contacts", slugColumn: "tenant_slug"},
	{name: "documents", slugColumn: "tenant_slug"},
	{name: "energy_assets", slugColumn: "tenant_slug"},
	{name: "energy_entity_mappings", slugColumn: "tenant_slug"},
	{name: "energy_imports", slugColumn: "tenant_slug"},
	{name: "energy_intervals", slugColumn: "tenant_slug"},
	{name: "energy_maintenance_plans", slugColumn: "tenant_slug"},
	{name: "energy_measures", slugColumn: "tenant_slug"},
	{name: "energy_tariff_assessments", slugColumn: "tenant_slug"},
	{name: "events", slugColumn: "tenant_slug"},
	{name: "handovers", slugColumn: "tenant_slug"},
	{name: "home_connector_readings", slugColumn: "slug"},
	{name: "home_connectors", slugColumn: "slug"},
	{name: "home_portals", slugColumn: "slug"},
	{name: "home_profiles", slugColumn: "tenant_slug"},
	{name: "home_reservations", slugColumn: "slug", preTenant: true},
	{name: "house_memberships", slugColumn: "tenant_slug"},
	{name: "integration_imports", slugColumn: "tenant_slug"},
	{name: "issues", slugColumn: "tenant_slug"},
	{name: "unit_payment_status", slugColumn: "tenant_slug"},
	{name: "units", slugColumn: "tenant_slug"},
}

// ErrTenantIDMissing reports rows that reached the completeness check without an
// identity. It is deliberately fatal at boot: the alternative is serving a
// tenant an empty house because its rows are invisible to a tenant_id filter.
var ErrTenantIDMissing = errors.New("store: tenant-bound rows without a tenant_id")

// ErrTenantSlugNotCanonical reports a stored slug that cannot be rewritten into
// its canonical spelling because the database refuses the rewrite.
//
// There are exactly two ways that happens, and the engine's own message says
// which: a UNIQUE or PRIMARY KEY collision — the canonical spelling of this row
// is already taken by another row of the same tenant — or a FOREIGN KEY that
// ties the label into another table, which is the case for home_profiles and
// the onboarding chain rooted at home_reservations(slug).
//
// This is fatal, and that is a decision rather than an omission. The three
// options for a row whose canonical spelling is already occupied are: pick a
// winner (the quarantine/ownership machinery this replaced — it hid a newer
// write behind a stale row and could not be undone), merge (impossible for the
// nine tables that keep their payload in an opaque `data` column with no
// timestamp), or refuse and say exactly which row to repair. Only the third is
// reversible by an operator, and only the third cannot be wrong.
//
// The condition cannot arise from the running product: every write path
// canonicalises through textutil.Slug before it stores anything, and a census of
// the live database on 2026-08-17 found zero non-canonical slugs across all 21
// tenant_slug tables. Reaching this error takes a hand-edited or imported row.
var ErrTenantSlugNotCanonical = errors.New("store: a stored tenant slug cannot be canonicalised")

// healOrphanReason is the Unscoped reason for the handful of writes that can
// land on a row LEFT BEHIND BY THE PREVIOUS RELEASE — one whose tenant_id is
// still NULL because it was written before the column existed.
//
// It is a lane decision, not a preference. PostgreSQL migration 0003 scopes a
// session by tenant_id and gives NO escape for a row that has none:
//
//	USING (coalesce(current_setting('hausv.tenant_id', true), '') = ''
//	       OR tenant_id = current_setting('hausv.tenant_id', true))
//
// A tenant lane always sets hausv.tenant_id, so the first branch is false and
// the second cannot match NULL. An orphan row is therefore invisible AND
// unwritable from every tenant lane, and the healing upsert that is supposed to
// give it its identity fails outright:
//
//	ERROR: new row violates row-level security policy (USING expression)
//	       for table "unit_payment_status" (SQLSTATE 42501)
//
// measured on the three tables whose rows are addressable by a NATURAL key
// (unit_payment_status, announcement_reads, units) — those are the only ones a
// new write can collide with an orphan on. The other tenant-bound tables key on
// a random id, so no write ever lands on a legacy row there and their upserts
// stay on the tenant lane.
//
// So the adoption runs on the declared cross-tenant lane, where the policy's
// first branch applies and the row can be reached. The statement itself still
// writes tenant_id=$1 and still filters on the tenant, so nothing widens: what
// changes is which lane is allowed to touch a row that belongs to no lane.
//
// This is time-boxed by the rollback window. When tenant_id becomes NOT NULL
// there are no orphans left, the heal is dead code, and these three sites go
// back to For(tenant). Until then, removing this is what re-opens HAUSV-145's
// "wrote a row it cannot see".
const healOrphanReason = "adopts a row the previous release left with a NULL tenant_id, which migration 0003 makes unreachable from every tenant lane; reverts to For(tenant) when tenant_id goes NOT NULL"

// tenantIDQuerier is the subset of *sql.DB and *sql.Tx the resolver needs, so
// the same code works inside and outside a transaction.
type tenantIDQuerier = tenantid.Querier

// lookupTenantID returns the identity of an existing tenant. Absence is not an
// error: onboarding rows can legitimately precede the identity.
func lookupTenantID(q tenantIDQuerier, slug string) (string, bool) {
	return tenantid.Lookup(q, slug)
}

// ensureTenantID resolves a slug to its identity, minting one if it has none.
func ensureTenantID(q tenantIDQuerier, slug string) (string, error) {
	return tenantid.Ensure(q, slug)
}

// tenantRefFor resolves a slug to the full reference the repository layer needs.
func tenantRefFor(q tenantIDQuerier, slug string) (TenantRef, error) {
	id, err := ensureTenantID(q, slug)
	if err != nil {
		return TenantRef{}, err
	}
	ref, ok := validTenantRef(TenantRef{ID: id, Slug: slug})
	if !ok {
		return TenantRef{}, fmt.Errorf("store: resolved tenant %s is not usable", slug)
	}
	return ref, nil
}

// tenantIDCache resolves slugs once per bulk operation. The JSON->SQL imports
// replay whole stores row by row and would otherwise turn one migration into one
// identity lookup per record.
type tenantIDCache struct {
	q      tenantIDQuerier
	byslug map[string]TenantRef
}

func newTenantIDCache(q tenantIDQuerier) *tenantIDCache {
	return &tenantIDCache{q: q, byslug: map[string]TenantRef{}}
}

func (c *tenantIDCache) ref(slug string) (TenantRef, error) {
	slug = textutil.Slug(slug)
	if ref, ok := c.byslug[slug]; ok {
		return ref, nil
	}
	ref, err := tenantRefFor(c.q, slug)
	if err != nil {
		return TenantRef{}, err
	}
	c.byslug[slug] = ref
	return ref, nil
}

// BackfillTenantIDs gives every tenant-bound row its identity and then proves
// none is left without one. It is idempotent: on an already-complete database
// every UPDATE matches zero rows.
//
// Run it AFTER the configured identities exist. Migration 0033 could not,
// which is the defect this repairs.
func BackfillTenantIDs(ctx context.Context, database *sql.DB) error {
	if database == nil {
		return fmt.Errorf("store: tenant id backfill requires a database")
	}
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin tenant id backfill: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, table := range tenantIDTables {
		if err := repairTenantIDs(ctx, tx, table); err != nil {
			return err
		}
	}
	if err := verifyTenantIDsTx(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit tenant id backfill: %w", err)
	}
	return nil
}

// repairTenantIDs canonicalises one table's stored slugs and then links every
// row that still has no identity.
//
// The order is the whole design. Canonicalising first means the link below can
// be a single wholesale UPDATE per tenant: after it, one tenant has exactly one
// spelling in this table, so no two rows can be candidates for the same logical
// key and there is nothing left to choose between.
//
// preTenant tables get the link but never the mint: a reservation is the request
// for a house made before the house exists, and its identity is minted at
// activation. Inventing one here would create a tenant nobody asked for.
//
// The steady state costs exactly one query. A table whose rows all carry an
// identity returns on the first statement, as it did before this file was
// rewritten — the canonicalisation pass is reached only while a table still has
// unowned rows, which is the migration boot and the rollback window.
func repairTenantIDs(ctx context.Context, tx *sql.Tx, table tenantIDTable) error {
	unowned, err := distinctSlugs(ctx, tx, table, true)
	if err != nil {
		return err
	}
	if len(unowned) == 0 {
		return nil
	}
	if err := canonicaliseSlugs(ctx, tx, table); err != nil {
		return err
	}
	linked := map[string]bool{}
	for _, raw := range unowned {
		canonical := textutil.Slug(raw)
		if canonical == "" {
			// Nothing can be resolved from a blank label. verifyTenantIDsTx
			// reports the row, with the offending value, instead.
			continue
		}
		if linked[canonical] {
			continue
		}
		linked[canonical] = true
		id, ok, err := identityForSlug(tx, canonical, !table.preTenant)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE `+table.name+` SET tenant_id=$1 WHERE tenant_id IS NULL AND `+table.slugColumn+`=$2`,
			id, canonical); err != nil {
			return fmt.Errorf("store: link tenant_id in %s: %w", table.name, err)
		}
	}
	return nil
}

// canonicaliseSlugs rewrites every stored spelling of this table's slug column
// into the one textutil.Slug produces — the same function the mint, the
// configuration layer and every write path already use, so there is one
// definition of what a slug IS rather than one in Go and another in SQL.
//
// It reads the WHOLE table's spellings, not just those of unowned rows. An
// already-linked row holding a non-canonical spelling is exactly what the
// previous design could not survive: it occupied a logical key that a newer,
// unowned row also wanted, and the row that lost was hidden permanently.
//
// A rewrite the database refuses is fatal and named. See ErrTenantSlugNotCanonical.
func canonicaliseSlugs(ctx context.Context, tx *sql.Tx, table tenantIDTable) error {
	stored, err := distinctSlugs(ctx, tx, table, false)
	if err != nil {
		return err
	}
	for _, raw := range stored {
		canonical := textutil.Slug(raw)
		if canonical == raw || canonical == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE `+table.name+` SET `+table.slugColumn+`=$1 WHERE `+table.slugColumn+`=$2`,
			canonical, raw); err != nil {
			return fmt.Errorf("%w: %s.%s %s -> %s: %w", ErrTenantSlugNotCanonical,
				table.name, table.slugColumn, strconv.Quote(raw), strconv.Quote(canonical), err)
		}
	}
	return nil
}

// distinctSlugs returns the raw, un-normalized spellings this table stores —
// either all of them, or only those of rows that have no identity yet.
func distinctSlugs(ctx context.Context, tx *sql.Tx, table tenantIDTable, unownedOnly bool) ([]string, error) {
	where := table.slugColumn + " IS NOT NULL"
	if unownedOnly {
		where = "tenant_id IS NULL AND " + where
	}
	rows, err := tx.QueryContext(ctx,
		`SELECT DISTINCT `+table.slugColumn+` FROM `+table.name+` WHERE `+where)
	if err != nil {
		return nil, fmt.Errorf("store: read stored slugs in %s: %w", table.name, err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			return nil, fmt.Errorf("store: scan stored slug in %s: %w", table.name, err)
		}
		out = append(out, slug)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate stored slugs in %s: %w", table.name, err)
	}
	return out, nil
}

// identityForSlug resolves one CANONICAL slug to its identity. It mints one when
// asked to: without that the backfill would silently skip a decommissioned
// house, a slug dropped from configuration while its rows remain, or a house
// activated through the portal before this code existed.
//
// The mint is tenantid.Ensure rather than a second INSERT written here. The
// difference is not style: this one ran with a bare INSERT and no recovery, so
// two replicas booting against one PostgreSQL could both find no tenant row,
// both INSERT, and the loser would take a unique violation on tenant.slug —
// aborting the backfill, and with it newApp. Harmless on today's single
// container and a live hazard the day the cutover runs two. tenantid.Ensure
// already answers a lost race by re-reading the row the winner committed, which
// is the resolution rather than a retry loop; there is no reason for a second
// copy of that logic to exist and drift.
//
// *sql.Tx satisfies tenantid.Querier, so the mint happens inside this
// transaction and is rolled back with everything else if the pass fails. That
// interface is context-free, which is why this function no longer takes a ctx:
// both statements are single-row lookups on `tenant`, and the connection's
// statement timeout still bounds them. Taking the interface rather than the
// *sql.Tx is also what lets a test put the losing side of that race under it.
func identityForSlug(q tenantIDQuerier, canonical string, mint bool) (string, bool, error) {
	if !mint {
		id, ok := lookupTenantID(q, canonical)
		return id, ok, nil
	}
	id, err := ensureTenantID(q, canonical)
	if err != nil {
		return "", false, fmt.Errorf("store: mint tenant id for %s: %w", canonical, err)
	}
	return id, true, nil
}

// verifyTenantIDsTx is the contradiction the migration never had. It counts what
// the switch depends on instead of assuming it.
//
// It names the offending slug values, not just the table and the count. This
// error stops the boot, so it is the only thing an operator has to work from;
// "announcements=1" tells them the product is down and nothing about which row
// to repair.
//
// Nothing is tolerated. The pass above either links a row or fails loudly, so a
// NULL that survives it is a slug that canonicalises to nothing — an empty or
// whitespace-only label — and there is no identity such a row could ever get.
func verifyTenantIDsTx(ctx context.Context, tx *sql.Tx) error {
	gaps := []string{}
	for _, table := range tenantIDTables {
		if table.preTenant {
			continue
		}
		var missing int
		if err := tx.QueryRowContext(ctx,
			`SELECT count(*) FROM `+table.name+` WHERE tenant_id IS NULL`).Scan(&missing); err != nil {
			return fmt.Errorf("store: count missing tenant_id in %s: %w", table.name, err)
		}
		if missing > 0 {
			gaps = append(gaps, fmt.Sprintf("%s=%d slugs=%s",
				table.name, missing, describeUnownedSlugs(ctx, tx, table)))
		}
	}
	if len(gaps) > 0 {
		return fmt.Errorf("%w: %s", ErrTenantIDMissing, strings.Join(gaps, " "))
	}
	return nil
}

// unownedSlugSampleSize caps the lead an error message carries. An error is not
// a report; a handful of distinct values is enough to find the rows.
const unownedSlugSampleSize = 5

// describeUnownedSlugs renders the slug values that could not be resolved, %q so
// "" and "   " are distinguishable. It never returns an error: it is called from
// the error path, and losing the real failure to a secondary query failure would
// be worse than losing the lead.
func describeUnownedSlugs(ctx context.Context, tx *sql.Tx, table tenantIDTable) string {
	rows, err := tx.QueryContext(ctx, `SELECT DISTINCT `+table.slugColumn+` FROM `+table.name+`
		WHERE tenant_id IS NULL ORDER BY `+table.slugColumn+` LIMIT `+strconv.Itoa(unownedSlugSampleSize+1))
	if err != nil {
		return "[unavailable]"
	}
	defer rows.Close()
	shown := []string{}
	more := false
	for rows.Next() {
		var slug sql.NullString
		if err := rows.Scan(&slug); err != nil {
			return "[unavailable]"
		}
		if len(shown) == unownedSlugSampleSize {
			more = true
			break
		}
		if !slug.Valid {
			shown = append(shown, "NULL")
			continue
		}
		shown = append(shown, strconv.Quote(slug.String))
	}
	if more {
		shown = append(shown, "...")
	}
	return "[" + strings.Join(shown, ", ") + "]"
}
