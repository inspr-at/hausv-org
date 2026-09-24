// Package dbmove copies every row of a SQLite database into an empty
// PostgreSQL database that carries the same schema, and proves it did.
//
// Production runs SQLite: one file, two tenants, some 1,600 rows across the
// tenant tables. Every code surface can already run on PostgreSQL, but nothing
// moved the rows — and without the rows there is no cutover, whatever else is
// ready. This is that tool. It ships inside the application binary
// (`hausv-org migrate-data`) so the cutover needs no second image and no
// second copy of the schema knowledge.
//
// What it does, in order:
//
//  1. Reads the source read-only and refuses a file whose schema_migrations
//     disagree with the migrations compiled into this binary.
//  2. Reads both catalogs and refuses if the two schemas differ in table or
//     column sets, or if a table exists that Plan does not know how to order.
//  3. Through the declared maintenance lane, counts every governed table on
//     the target and refuses unless all are empty (or --force wipes them,
//     inside the same transaction as the load).
//  4. Copies table by table in foreign-key order, converting every value by
//     the TARGET column's declared type — bool from 0/1, bytea from BLOB, and
//     so on — through a fixed converter table that errors on anything it does
//     not recognise, rather than by whatever the two drivers happen to agree on.
//  5. Verifies, still inside the transaction: per-table row counts and a
//     per-table content hash, computed by the same function on both sides.
//     Any mismatch rolls the whole load back and returns an error.
//
// One transaction, not one per table. The failure mode this tool exists to
// prevent is a HALF-FULL target — it is the one input the tool itself refuses,
// so it must never produce one. With one transaction there are two reachable
// end states, "everything, verified" and "nothing", and the second is exactly
// the state a re-run wants to start from. The row volume makes the cost of a
// single transaction irrelevant.
package dbmove

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/db"
)

// TableOrder is the foreign-key-safe load order, and the complete list of
// tables the mover will touch. It is deliberately explicit rather than derived
// from the catalog at run time: a reader can see the plan, and a table that
// appears in either schema without appearing here stops the run — that is what
// keeps this list honest when a migration adds a table.
//
// The dependencies that fix the order (PostgreSQL side, which is the stricter):
//
//	tenant                    <- every tenant_id column
//	persons                   <- house_memberships.person_id
//	home_profiles             <- the six energy_* tables (tenant_slug, home_key)
//	energy_assets             <- energy_maintenance_plans.asset_id
//	home_reservations         <- home_connectors.slug, home_portals.slug
//	home_connectors           <- home_connector_readings.slug
//
// TestTableOrderRespectsForeignKeys reads the real constraints from both
// engines and fails if this list ever contradicts them.
var TableOrder = []string{
	// Registries and global tables: nothing points at them but tenant_id and
	// person_id, and they point at nothing.
	"tenant",
	"persons",
	"app_meta",
	"index_imports",
	"index_values",
	"login_activity",
	"profile_overlays",
	"notification_prefs",
	"person_avatars",
	"telegram_state",
	"telegram_links",
	"telegram_link_codes",
	// Organisation-scoped tables (HAUSV-600): keyed by org_key, no tenant_id,
	// nothing points at them.
	"organisations",
	"organisation_houses",
	"organisation_members",
	"org_settings",
	// HAUSV-699: configurable rights, keyed by org_key like org_settings.
	"organisation_capability_overrides",
	"user_capability_grants",
	"capability_profiles",
	"textbausteine",
	"intake_items",
	"intake_mail_seen",
	// Tenant-bound tables that depend only on tenant.
	"house_memberships",
	"annual_statement_cost_types",
	"annual_statement_periods",
	"annual_statement_consumption_evidence",
	"annual_statement_period_cost_types",
	"annual_statement_period_unit_bases",
	"annual_statement_prepayments",
	"annual_statement_receipts",
	"annual_statement_reserve_entries",
	"annual_statement_runs",
	"annual_statement_run_approvals",
	"annual_statement_deliveries",
	"unit_payment_status",
	"contacts",
	"announcement_reads",
	"announcements",
	"events",
	"handovers",
	"documents",
	"attachments",
	"units",
	"leases",
	"lease_parties",
	"rent_components",
	"index_clauses",
	"valorisation_state",
	"valorisation_runs",
	"valorisation_items",
	"valorisation_deliveries",
	"valorisation_events",
	"ballots",
	"issues",
	"parking",
	"integration_imports",
	// The energy chain.
	"home_profiles",
	"energy_assets",
	"energy_entity_mappings",
	"energy_intervals",
	"energy_imports",
	"energy_tariff_assessments",
	"energy_measures",
	"energy_maintenance_plans",
	// The home onboarding chain.
	"home_reservations",
	"home_connectors",
	"home_portals",
	"home_connector_readings",
}

// ignoredTable names bookkeeping that legitimately differs between the engines
// and is owned by each engine's own migration runner.
func ignoredTable(name string) bool {
	return name == "schema_migrations"
}

// ErrTargetNotEmpty is returned when the target holds rows and --force was not
// given.
var ErrTargetNotEmpty = errors.New("dbmove: target is not empty")

// ErrVerifyMismatch is returned when a table's count or content hash differs
// between source and target after the load. The load has been rolled back.
var ErrVerifyMismatch = errors.New("dbmove: verification mismatch")

// ErrSchemaDrift is returned when the two schemas, or the plan and a schema,
// disagree.
var ErrSchemaDrift = errors.New("dbmove: schema drift")

// ErrSourceMigrations is returned when the source file's schema_migrations do
// not match the migrations compiled into this binary.
var ErrSourceMigrations = errors.New("dbmove: source migration state")

// Options steer one run.
type Options struct {
	// Force wipes a non-empty target inside the load transaction instead of
	// refusing it. It never merges: the wipe empties every governed table.
	Force bool
	// DryRun does everything — the emptiness check, the load, the verification
	// — and then rolls back instead of committing. It is the rehearsal.
	DryRun bool
	// Out receives the human-readable report as it is produced. nil discards.
	Out io.Writer
}

// TableResult is the verification outcome for one table.
type TableResult struct {
	Name          string
	SourceRows    int64
	TargetRows    int64
	SourceHash    string
	TargetHash    string
	Match         bool
	NullTenantIDs int64
	// Conversions lists the columns whose type changes representation on the
	// way over, as "column: sqlite-type -> postgres-type".
	Conversions []string
}

// Report is what a run leaves behind.
type Report struct {
	Tables    []TableResult
	Committed bool
	DryRun    bool
	// TargetRowsBefore holds the pre-load counts of every governed table that
	// was not empty, so the operator sees what a refusal or a wipe was about.
	TargetRowsBefore map[string]int64
	Duration         time.Duration
}

// TotalSourceRows sums the rows read from the source.
func (r *Report) TotalSourceRows() int64 {
	var n int64
	for _, table := range r.Tables {
		n += table.SourceRows
	}
	return n
}

// AllMatch reports whether every table verified.
func (r *Report) AllMatch() bool {
	for _, table := range r.Tables {
		if !table.Match {
			return false
		}
	}
	return len(r.Tables) > 0
}

// column is one target column with the converter its declared type selects.
type column struct {
	name       string
	sqliteType string
	pgType     string
	convert    converter
}

// converter turns a value as the SQLite driver returned it into the value the
// PostgreSQL driver must be handed, or refuses.
type converter func(v any) (any, error)

// convertersByPostgresType is the whole type story, in one place. A target
// column whose data_type is not a key here stops the run before a single row
// moves.
var convertersByPostgresType = map[string]converter{
	"text":              toText,
	"character varying": toText,
	"boolean":           toBool,
	"bigint":            toInt,
	"integer":           toInt,
	"double precision":  toFloat,
	"bytea":             toBytes,
}

func toText(v any) (any, error) {
	switch x := v.(type) {
	case nil:
		return nil, nil
	case string:
		return x, nil
	case []byte:
		return string(x), nil
	}
	return nil, fmt.Errorf("text column holds %T", v)
}

func toBool(v any) (any, error) {
	switch x := v.(type) {
	case nil:
		return nil, nil
	case bool:
		return x, nil
	case int64:
		switch x {
		case 0:
			return false, nil
		case 1:
			return true, nil
		}
		return nil, fmt.Errorf("boolean column holds integer %d", x)
	}
	return nil, fmt.Errorf("boolean column holds %T", v)
}

func toInt(v any) (any, error) {
	switch x := v.(type) {
	case nil:
		return nil, nil
	case int64:
		return x, nil
	}
	return nil, fmt.Errorf("integer column holds %T", v)
}

func toFloat(v any) (any, error) {
	switch x := v.(type) {
	case nil:
		return nil, nil
	case float64:
		return x, nil
	case int64:
		// SQLite's REAL affinity stores an integral value as an integer on disk
		// and hands it back as one; the value is the same number.
		return float64(x), nil
	}
	return nil, fmt.Errorf("double precision column holds %T", v)
}

func toBytes(v any) (any, error) {
	switch x := v.(type) {
	case nil:
		return nil, nil
	case []byte:
		return x, nil
	case string:
		return []byte(x), nil
	}
	return nil, fmt.Errorf("bytea column holds %T", v)
}

// table is the resolved plan for one table.
type table struct {
	name    string
	columns []column
	// hasTenantID marks the 25 tenant-bound tables, for the orphan count.
	hasTenantID bool
}

// plan reads both catalogs and resolves the load plan, or explains why the two
// databases cannot be moved between. It writes nothing.
func plan(ctx context.Context, source querier, lane db.Handle) ([]table, error) {
	sourceSchema, err := sqliteSchema(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("dbmove: read source schema: %w", err)
	}
	targetSchema, err := postgresSchema(lane)
	if err != nil {
		return nil, fmt.Errorf("dbmove: read target schema: %w", err)
	}
	if len(sourceSchema) == 0 {
		return nil, fmt.Errorf("%w: source has no tables — is this a hausv database?", ErrSchemaDrift)
	}
	if len(targetSchema) == 0 {
		return nil, fmt.Errorf("%w: target has no tables — were the PostgreSQL migrations applied?", ErrSchemaDrift)
	}

	var problems []string
	planned := map[string]bool{}
	for _, name := range TableOrder {
		planned[name] = true
		if _, ok := sourceSchema[name]; !ok {
			problems = append(problems, fmt.Sprintf("plan names %q but the source has no such table", name))
		}
		if _, ok := targetSchema[name]; !ok {
			problems = append(problems, fmt.Sprintf("plan names %q but the target has no such table", name))
		}
	}
	for _, name := range sortedKeys(sourceSchema) {
		if !planned[name] {
			problems = append(problems, fmt.Sprintf("source table %q has no entry in TableOrder — add it in foreign-key order", name))
		}
	}
	for _, name := range sortedKeys(targetSchema) {
		if !planned[name] {
			problems = append(problems, fmt.Sprintf("target table %q has no entry in TableOrder — add it in foreign-key order", name))
		}
	}
	if len(problems) > 0 {
		return nil, fmt.Errorf("%w:\n  %s", ErrSchemaDrift, strings.Join(problems, "\n  "))
	}

	var resolvedPlan []table
	for _, name := range TableOrder {
		sourceCols := sourceSchema[name]
		targetCols := targetSchema[name]
		missing, extra := diffKeys(sourceCols, targetCols)
		if len(missing) > 0 {
			problems = append(problems, fmt.Sprintf("%s: columns missing from target: %s", name, strings.Join(missing, ", ")))
		}
		if len(extra) > 0 {
			problems = append(problems, fmt.Sprintf("%s: columns only in target: %s", name, strings.Join(extra, ", ")))
		}
		if len(missing) > 0 || len(extra) > 0 {
			continue
		}
		resolved := table{name: name}
		for _, colName := range sortedKeys(targetCols) {
			pgType := targetCols[colName]
			convert, ok := convertersByPostgresType[pgType]
			if !ok {
				problems = append(problems, fmt.Sprintf("%s.%s: no converter for PostgreSQL type %q", name, colName, pgType))
				continue
			}
			if colName == "tenant_id" && name != "tenant" {
				resolved.hasTenantID = true
			}
			resolved.columns = append(resolved.columns, column{
				name:       colName,
				sqliteType: sourceCols[colName],
				pgType:     pgType,
				convert:    convert,
			})
		}
		resolvedPlan = append(resolvedPlan, resolved)
	}
	if len(problems) > 0 {
		return nil, fmt.Errorf("%w:\n  %s", ErrSchemaDrift, strings.Join(problems, "\n  "))
	}
	return resolvedPlan, nil
}

// CheckSourceMigrations refuses a source whose applied migrations are not
// exactly the ones compiled into this binary. Behind means the running
// application has not yet migrated the file (boot it on this version first);
// ahead means this binary is older than the file and cannot know its shape.
func CheckSourceMigrations(ctx context.Context, source querier) error {
	rows, err := source.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return fmt.Errorf("%w: read schema_migrations: %v", ErrSourceMigrations, err)
	}
	defer rows.Close()
	applied := map[string]bool{}
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return fmt.Errorf("%w: scan schema_migrations: %v", ErrSourceMigrations, err)
		}
		applied[version] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("%w: iterate schema_migrations: %v", ErrSourceMigrations, err)
	}
	var behind, ahead []string
	for _, name := range db.SQLiteMigrationNames() {
		if !applied[name] {
			behind = append(behind, name)
		}
		delete(applied, name)
	}
	for name := range applied {
		ahead = append(ahead, name)
	}
	sort.Strings(ahead)
	if len(behind) > 0 {
		return fmt.Errorf("%w: source is missing %s — boot the application on this version against the file first", ErrSourceMigrations, strings.Join(behind, ", "))
	}
	if len(ahead) > 0 {
		return fmt.Errorf("%w: source carries %s which this binary does not know — run the mover from the same version that last migrated the file", ErrSourceMigrations, strings.Join(ahead, ", "))
	}
	return nil
}

// Move copies the source into the target through lane, verifies, and commits
// unless opts.DryRun. On any error the target is left exactly as it was found.
//
// lane MUST be the declared maintenance lane (db.Scoped.Unscoped): the emptiness
// check, the wipe, the load and the verification all run on it. On a fail-closed
// row-level-security target a plain connection would count zero rows in a full
// table, insert nothing, and verify nothing — the emptiness check in particular
// is only meaningful on a lane that can see every tenant.
func Move(ctx context.Context, source *sql.DB, lane db.Handle, opts Options) (*Report, error) {
	started := time.Now()
	out := opts.Out
	if out == nil {
		out = io.Discard
	}
	report := &Report{DryRun: opts.DryRun, TargetRowsBefore: map[string]int64{}}

	// One read transaction on the source for the whole run: the load and the
	// verification then see the same snapshot even if something is still
	// writing to the file, and the mismatch they would otherwise report would
	// be about the writer, not the move.
	sourceTx, err := source.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return report, fmt.Errorf("dbmove: begin source read: %w", err)
	}
	defer func() { _ = sourceTx.Rollback() }()

	if err := CheckSourceMigrations(ctx, sourceTx); err != nil {
		return report, err
	}
	tables, err := plan(ctx, sourceTx, lane)
	if err != nil {
		return report, err
	}
	fmt.Fprintf(out, "plan: %d tables, load order %s\n", len(tables), strings.Join(TableOrder, " → "))

	tx, err := lane.Begin()
	if err != nil {
		return report, fmt.Errorf("dbmove: begin target transaction: %w", err)
	}
	// Rollback after Commit is a no-op; on every other path this is what makes
	// "the target is left exactly as it was found" true.
	defer func() { _ = tx.Rollback() }()

	// Emptiness, on the lane, inside the transaction, so what is counted is what
	// the load would land beside.
	for _, t := range tables {
		count, err := countRows(ctx, tx, t.name)
		if err != nil {
			return report, fmt.Errorf("dbmove: count target %s: %w", t.name, err)
		}
		if count > 0 {
			report.TargetRowsBefore[t.name] = count
		}
	}
	if len(report.TargetRowsBefore) > 0 {
		fmt.Fprintf(out, "target is NOT empty: %s\n", describeCounts(report.TargetRowsBefore))
		if !opts.Force {
			return report, fmt.Errorf("%w: %s (pass --force to wipe every governed table first; the mover never merges)", ErrTargetNotEmpty, describeCounts(report.TargetRowsBefore))
		}
		fmt.Fprintf(out, "--force: wiping %d governed tables inside the load transaction\n", len(tables))
		// These immutable snapshots have foreign keys between the four tables;
		// an explicit full-target replacement truncates the group together.
		if _, err := tx.ExecContext(ctx, `TRUNCATE TABLE valorisation_events,valorisation_deliveries,valorisation_items,valorisation_runs`); err != nil {
			return report, fmt.Errorf("dbmove: wipe valorisation: %w", err)
		}
		for i := len(tables) - 1; i >= 0; i-- {
			wipe := `DELETE FROM ` + quoteIdent(tables[i].name)
			if tables[i].name == "annual_statement_runs" || tables[i].name == "annual_statement_run_approvals" {
				// Row-level changes are forbidden for immutable runs. Only this
				// explicit full-target replacement clears the table as a whole;
				// TRUNCATE stays inside the load transaction and uses no CASCADE.
				wipe = `TRUNCATE TABLE ` + quoteIdent(tables[i].name)
			}
			if _, err := tx.ExecContext(ctx, wipe); err != nil {
				return report, fmt.Errorf("dbmove: wipe target %s: %w", tables[i].name, err)
			}
		}
		for _, t := range tables {
			count, err := countRows(ctx, tx, t.name)
			if err != nil {
				return report, fmt.Errorf("dbmove: recount target %s: %w", t.name, err)
			}
			if count > 0 {
				return report, fmt.Errorf("dbmove: %s still holds %d rows after the wipe — the lane cannot see or delete every row; refusing to load beside them", t.name, count)
			}
		}
	} else {
		fmt.Fprintf(out, "target check: all %d governed tables empty\n", len(tables))
	}

	// Restoring an approved snapshot inserts its frozen items after its parent.
	// Only this guard is suspended, transactionally; a failed restore rolls it back.
	if _, err := tx.ExecContext(ctx, `ALTER TABLE valorisation_items DISABLE TRIGGER valorisation_item_no_insert`); err != nil {
		return report, err
	}
	// Load.
	for _, t := range tables {
		moved, nullTenants, err := copyTable(ctx, sourceTx, tx, t)
		if err != nil {
			return report, fmt.Errorf("dbmove: copy %s: %w", t.name, err)
		}
		result := TableResult{Name: t.name, SourceRows: moved, NullTenantIDs: nullTenants}
		for _, c := range t.columns {
			if !sameRepresentation(c.sqliteType, c.pgType) {
				result.Conversions = append(result.Conversions, fmt.Sprintf("%s: %s -> %s", c.name, c.sqliteType, c.pgType))
			}
		}
		report.Tables = append(report.Tables, result)
	}

	if _, err := tx.ExecContext(ctx, `ALTER TABLE valorisation_items ENABLE TRIGGER valorisation_item_no_insert`); err != nil {
		return report, err
	}
	// Verify, before anything is committed.
	mismatches := 0
	for i := range report.Tables {
		t := tables[i]
		sourceCount, sourceHash, err := hashTable(ctx, sourceTx, t, true)
		if err != nil {
			return report, fmt.Errorf("dbmove: hash source %s: %w", t.name, err)
		}
		targetCount, targetHash, err := hashTable(ctx, tx, t, false)
		if err != nil {
			return report, fmt.Errorf("dbmove: hash target %s: %w", t.name, err)
		}
		r := &report.Tables[i]
		r.SourceRows, r.TargetRows = sourceCount, targetCount
		r.SourceHash, r.TargetHash = sourceHash, targetHash
		r.Match = sourceCount == targetCount && sourceHash == targetHash
		if !r.Match {
			mismatches++
		}
	}
	writeTableReport(out, report)
	if mismatches > 0 {
		return report, fmt.Errorf("%w: %d of %d tables differ after the load; nothing was committed", ErrVerifyMismatch, mismatches, len(tables))
	}
	if opts.DryRun {
		fmt.Fprintf(out, "dry-run: %d rows in %d tables loaded and verified, ROLLED BACK on purpose (%s)\n", report.TotalSourceRows(), len(tables), time.Since(started).Round(time.Millisecond))
		report.Duration = time.Since(started)
		return report, nil
	}
	if err := tx.Commit(); err != nil {
		return report, fmt.Errorf("dbmove: commit: %w", err)
	}
	report.Committed = true
	report.Duration = time.Since(started)
	fmt.Fprintf(out, "COMMITTED: %d rows in %d tables, every table verified by count and content hash (%s)\n", report.TotalSourceRows(), len(tables), report.Duration.Round(time.Millisecond))
	return report, nil
}

// Verify compares source and target without writing anything. It is what a
// re-check after the move runs, and what a test uses to prove the hash sees a
// changed value and not just a changed count.
func Verify(ctx context.Context, source *sql.DB, lane db.Handle, out io.Writer) (*Report, error) {
	if out == nil {
		out = io.Discard
	}
	report := &Report{TargetRowsBefore: map[string]int64{}}
	sourceTx, err := source.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return report, fmt.Errorf("dbmove: begin source read: %w", err)
	}
	defer func() { _ = sourceTx.Rollback() }()
	if err := CheckSourceMigrations(ctx, sourceTx); err != nil {
		return report, err
	}
	tables, err := plan(ctx, sourceTx, lane)
	if err != nil {
		return report, err
	}
	mismatches := 0
	for _, t := range tables {
		sourceCount, sourceHash, err := hashTable(ctx, sourceTx, t, true)
		if err != nil {
			return report, fmt.Errorf("dbmove: hash source %s: %w", t.name, err)
		}
		targetCount, targetHash, err := hashTable(ctx, lane, t, false)
		if err != nil {
			return report, fmt.Errorf("dbmove: hash target %s: %w", t.name, err)
		}
		r := TableResult{
			Name: t.name, SourceRows: sourceCount, TargetRows: targetCount,
			SourceHash: sourceHash, TargetHash: targetHash,
			Match: sourceCount == targetCount && sourceHash == targetHash,
		}
		if !r.Match {
			mismatches++
		}
		report.Tables = append(report.Tables, r)
	}
	writeTableReport(out, report)
	if mismatches > 0 {
		return report, fmt.Errorf("%w: %d of %d tables differ", ErrVerifyMismatch, mismatches, len(tables))
	}
	fmt.Fprintf(out, "verify: all %d tables match by count and content hash (%d rows)\n", len(tables), report.TotalSourceRows())
	return report, nil
}

// querier is what hashTable and countRows need: *sql.DB, *sql.Tx and db.Handle
// all provide it, once the context variants are wrapped.
type querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// handleQuerier adapts a db.Handle, whose four methods take no context.
type handleQuerier struct{ h db.Handle }

func (q handleQuerier) QueryContext(_ context.Context, query string, args ...any) (*sql.Rows, error) {
	return q.h.Query(query, args...)
}
func (q handleQuerier) QueryRowContext(_ context.Context, query string, args ...any) *sql.Row {
	return q.h.QueryRow(query, args...)
}

func asQuerier(x any) querier {
	switch v := x.(type) {
	case querier:
		return v
	case db.Handle:
		return handleQuerier{h: v}
	}
	panic(fmt.Sprintf("dbmove: %T is not a querier", x))
}

func countRows(ctx context.Context, q any, name string) (int64, error) {
	var count int64
	err := asQuerier(q).QueryRowContext(ctx, `SELECT count(*) FROM `+quoteIdent(name)).Scan(&count)
	return count, err
}

const insertBatch = 200

// copyTable streams the source table into the target transaction. It returns
// the number of rows written and how many of them carry a NULL tenant_id.
func copyTable(ctx context.Context, source querier, tx *sql.Tx, t table) (int64, int64, error) {
	rows, err := source.QueryContext(ctx, selectAll(t))
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	tenantIndex := -1
	for i, c := range t.columns {
		if t.hasTenantID && c.name == "tenant_id" {
			tenantIndex = i
		}
	}

	var (
		moved, nullTenants int64
		batch              []any
		batchRows          int
	)
	flush := func() error {
		if batchRows == 0 {
			return nil
		}
		if _, err := tx.ExecContext(ctx, insertStatement(t, batchRows), batch...); err != nil {
			return err
		}
		moved += int64(batchRows)
		batch = batch[:0]
		batchRows = 0
		return nil
	}

	raw := make([]any, len(t.columns))
	ptrs := make([]any, len(t.columns))
	for i := range raw {
		ptrs[i] = &raw[i]
	}
	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			return moved, nullTenants, err
		}
		for i, c := range t.columns {
			value, err := c.convert(raw[i])
			if err != nil {
				return moved, nullTenants, fmt.Errorf("row %d column %s: %w", moved+int64(batchRows)+1, c.name, err)
			}
			if i == tenantIndex && value == nil {
				nullTenants++
			}
			batch = append(batch, value)
		}
		batchRows++
		if batchRows >= insertBatch {
			if err := flush(); err != nil {
				return moved, nullTenants, err
			}
		}
	}
	if err := rows.Err(); err != nil {
		return moved, nullTenants, err
	}
	if err := flush(); err != nil {
		return moved, nullTenants, err
	}
	return moved, nullTenants, nil
}

// hashTable returns the row count and an order-independent content hash of a
// table. Values are canonicalised through the same converter set the load
// uses, so a source integer 1 in a boolean column and a target boolean true
// hash identically — and a source value that the load would have refused is
// refused here too. Row hashes are sorted before they are combined, so the two
// engines' physical orders do not matter and duplicate rows still count twice.
func hashTable(ctx context.Context, q any, t table, isSource bool) (int64, string, error) {
	rows, err := asQuerier(q).QueryContext(ctx, selectAll(t))
	if err != nil {
		return 0, "", err
	}
	defer rows.Close()

	raw := make([]any, len(t.columns))
	ptrs := make([]any, len(t.columns))
	for i := range raw {
		ptrs[i] = &raw[i]
	}
	var (
		count  int64
		hashes []string
	)
	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			return count, "", err
		}
		digest := sha256.New()
		for i, c := range t.columns {
			value := raw[i]
			if isSource {
				converted, err := c.convert(value)
				if err != nil {
					return count, "", fmt.Errorf("row %d column %s: %w", count+1, c.name, err)
				}
				value = converted
			}
			encoded, err := canonical(value)
			if err != nil {
				return count, "", fmt.Errorf("row %d column %s: %w", count+1, c.name, err)
			}
			digest.Write([]byte(c.name))
			digest.Write([]byte{0})
			digest.Write([]byte(encoded))
			digest.Write([]byte{0})
		}
		hashes = append(hashes, hex.EncodeToString(digest.Sum(nil)))
		count++
	}
	if err := rows.Err(); err != nil {
		return count, "", err
	}
	sort.Strings(hashes)
	total := sha256.New()
	for _, h := range hashes {
		total.Write([]byte(h))
	}
	return count, hex.EncodeToString(total.Sum(nil)), nil
}

// canonical writes one already-converted value in a form that depends only on
// its value and its converted kind — never on which driver produced it.
func canonical(v any) (string, error) {
	switch x := v.(type) {
	case nil:
		return "N", nil
	case string:
		return "S" + strconv.Itoa(len(x)) + ":" + x, nil
	case bool:
		if x {
			return "B1", nil
		}
		return "B0", nil
	case int64:
		return "I" + strconv.FormatInt(x, 10), nil
	case float64:
		return "F" + strconv.FormatFloat(x, 'g', -1, 64), nil
	case []byte:
		return "X" + hex.EncodeToString(x), nil
	}
	return "", fmt.Errorf("value of type %T has no canonical form", v)
}

func selectAll(t table) string {
	names := make([]string, len(t.columns))
	for i, c := range t.columns {
		names[i] = quoteIdent(c.name)
	}
	return `SELECT ` + strings.Join(names, ", ") + ` FROM ` + quoteIdent(t.name)
}

func insertStatement(t table, rowCount int) string {
	names := make([]string, len(t.columns))
	for i, c := range t.columns {
		names[i] = quoteIdent(c.name)
	}
	var b strings.Builder
	b.WriteString(`INSERT INTO ` + quoteIdent(t.name) + ` (` + strings.Join(names, ", ") + `) VALUES `)
	param := 1
	for r := 0; r < rowCount; r++ {
		if r > 0 {
			b.WriteString(", ")
		}
		b.WriteString("(")
		for c := range t.columns {
			if c > 0 {
				b.WriteString(", ")
			}
			b.WriteString("$" + strconv.Itoa(param))
			param++
		}
		b.WriteString(")")
	}
	return b.String()
}

// quoteIdent double-quotes an identifier that came from a catalog, so a name
// that happens to be a keyword cannot break the statement. Names originate in
// sqlite_master and information_schema, never in input.
func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// sameRepresentation reports whether a value crosses the engines unchanged.
// It exists for the report only; the converters decide what actually happens.
func sameRepresentation(sqliteType, pgType string) bool {
	switch pgType {
	case "text", "character varying":
		return sqliteType == "text"
	case "bigint", "integer":
		return sqliteType == "integer"
	case "double precision":
		return sqliteType == "real"
	}
	return false
}

func describeCounts(counts map[string]int64) string {
	parts := make([]string, 0, len(counts))
	for _, name := range sortedKeys(counts) {
		parts = append(parts, fmt.Sprintf("%s=%d", name, counts[name]))
	}
	return strings.Join(parts, ", ")
}

func writeTableReport(out io.Writer, report *Report) {
	fmt.Fprintf(out, "%-26s %8s %8s  %-14s %-14s  %s\n", "table", "source", "target", "source-hash", "target-hash", "status")
	var orphans int64
	for _, t := range report.Tables {
		status := "ok"
		if !t.Match {
			status = "MISMATCH"
		}
		note := ""
		if t.NullTenantIDs > 0 {
			note = fmt.Sprintf("  (%d rows without tenant_id)", t.NullTenantIDs)
			orphans += t.NullTenantIDs
		}
		fmt.Fprintf(out, "%-26s %8d %8d  %-14s %-14s  %s%s\n", t.Name, t.SourceRows, t.TargetRows, short(t.SourceHash), short(t.TargetHash), status, note)
	}
	if orphans > 0 {
		fmt.Fprintf(out, "note: %d rows carry no tenant_id; they move unchanged, and under a fail-closed policy no tenant lane can read them until the application's orphan-adoption paths assign one\n", orphans)
	}
	var conversions []string
	for _, t := range report.Tables {
		for _, c := range t.Conversions {
			conversions = append(conversions, t.Name+"."+c)
		}
	}
	if len(conversions) > 0 {
		fmt.Fprintf(out, "type conversions applied (%d columns): %s\n", len(conversions), strings.Join(conversions, "; "))
	}
}

func short(hash string) string {
	if len(hash) > 12 {
		return hash[:12]
	}
	return hash
}

// schema maps table -> column -> declared type, lower-cased.
type schema map[string]map[string]string

func sqliteSchema(ctx context.Context, database querier) (schema, error) {
	rows, err := database.QueryContext(ctx,
		`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		return nil, err
	}
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return nil, err
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	out := schema{}
	for _, name := range tables {
		if ignoredTable(name) {
			continue
		}
		colRows, err := database.QueryContext(ctx, `PRAGMA table_info(`+quoteIdent(name)+`)`)
		if err != nil {
			return nil, err
		}
		cols := map[string]string{}
		for colRows.Next() {
			var (
				cid        int
				colName    string
				typ        string
				notNull    int
				dflt       sql.NullString
				primaryKey int
			)
			if err := colRows.Scan(&cid, &colName, &typ, &notNull, &dflt, &primaryKey); err != nil {
				colRows.Close()
				return nil, err
			}
			cols[colName] = strings.ToLower(typ)
		}
		if err := colRows.Err(); err != nil {
			colRows.Close()
			return nil, err
		}
		colRows.Close()
		out[name] = cols
	}
	return out, nil
}

func postgresSchema(lane db.Handle) (schema, error) {
	// current_schema() follows the search_path the lane was dialled with: the
	// isolated schema under test, public in production.
	rows, err := lane.Query(
		`SELECT c.table_name, c.column_name, c.data_type
		   FROM information_schema.columns c
		   JOIN information_schema.tables t
		     ON t.table_schema = c.table_schema AND t.table_name = c.table_name
		  WHERE c.table_schema = current_schema()
		    AND t.table_type = 'BASE TABLE'
		  ORDER BY c.table_name, c.column_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := schema{}
	for rows.Next() {
		var tableName, columnName, dataType string
		if err := rows.Scan(&tableName, &columnName, &dataType); err != nil {
			return nil, err
		}
		if ignoredTable(tableName) {
			continue
		}
		if out[tableName] == nil {
			out[tableName] = map[string]string{}
		}
		out[tableName][columnName] = strings.ToLower(dataType)
	}
	return out, rows.Err()
}

func diffKeys[V any](want, got map[string]V) (missing, extra []string) {
	for key := range want {
		if _, ok := got[key]; !ok {
			missing = append(missing, key)
		}
	}
	for key := range got {
		if _, ok := want[key]; !ok {
			extra = append(extra, key)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	return missing, extra
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for key := range m {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}
