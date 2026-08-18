package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/dbmove"
)

const migrateDataUsage = `usage: hausv-org migrate-data [--from PATH] [--dry-run | --verify-only] [--force]

Copies every row of the SQLite database into the PostgreSQL database and proves
the copy, table by table. Run it ONCE, BEFORE the application is switched to
DB_BACKEND=postgres: the first PostgreSQL boot mints tenant identities of its
own, and after that the target is no longer empty and the mover refuses it.

What it needs
  DATABASE_URL     the PostgreSQL DSN, from the environment only — never a flag,
                   so the role password cannot land in a shell history or a
                   process list. The role must be the application role
                   (NOSUPERUSER, NOBYPASSRLS); the mover writes through the same
                   declared cross-tenant lane the application uses.
  The target's schema is created or brought up to date on open, with the same
  migration runner the application boots with — an empty database is the
  expected starting point.
  --from PATH      the SQLite file. Defaults to DB_PATH, then to hausv.db beside
                   PARKING_DATA_PATH — the same rule the application applies —
                   so inside the production container the flag can be omitted.
                   The file is opened read-only and is never modified.
  The application must be STOPPED (or the file otherwise quiescent) so the
  snapshot the mover reads is the state the application last committed.

What it refuses
  * a source whose schema_migrations differ from this binary's (behind: boot the
    application on this version first; ahead: use the version that migrated it)
  * a target whose schema differs from the source in tables or columns, or a
    table the mover has no load order for
  * a target that is not EMPTY in every governed table, unless --force, which
    wipes every governed table inside the same transaction as the load — it
    never merges into existing rows
  * a target role that is superuser or BYPASSRLS

Modes
  (default)      load, verify, commit — all in one transaction
  --dry-run      load and verify exactly as the real run would, then roll back;
                 the rehearsal to run first
  --verify-only  compare source and target without writing; the check to run
                 after the move and before the switch
  --force        proceed on a non-empty target by wiping it first (see above)

The report
  One line per table: rows read from the source, rows found in the target
  after the load, and a content hash of each side computed the same way —
  every column of every row, canonicalised after type conversion, order-
  independent. "ok" means count AND hash agree. Any "MISMATCH" rolls the whole
  load back, prints which tables differ, and exits non-zero. Rows without a
  tenant_id are counted per table: they move unchanged, but under a
  fail-closed row-level-security policy no tenant lane can read them until the
  application's orphan-adoption paths assign one.

Exit status
  0  committed (or dry-run / verify-only succeeded)
  1  refused, failed, or a mismatch — the target is exactly as it was found
`

// runMigrateData is the entry point behind `hausv-org migrate-data`.
func runMigrateData(args []string, stdout, stderr io.Writer, getenv func(string) string) error {
	flags := flag.NewFlagSet("migrate-data", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() { fmt.Fprint(stderr, migrateDataUsage) }
	from := flags.String("from", "", "SQLite source file (default: DB_PATH, then hausv.db beside PARKING_DATA_PATH)")
	force := flags.Bool("force", false, "wipe a non-empty target inside the load transaction instead of refusing it")
	dryRun := flags.Bool("dry-run", false, "load and verify, then roll back")
	verifyOnly := flags.Bool("verify-only", false, "compare source and target without writing")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			// The usage text is the operator note; asking for it is not a failure.
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		flags.Usage()
		return fmt.Errorf("unexpected argument %q", flags.Arg(0))
	}
	if *dryRun && *verifyOnly {
		return fmt.Errorf("--dry-run and --verify-only exclude each other")
	}

	sourcePath := strings.TrimSpace(*from)
	if sourcePath == "" {
		sourcePath = strings.TrimSpace(getenv("DB_PATH"))
	}
	if sourcePath == "" {
		if parking := strings.TrimSpace(getenv("PARKING_DATA_PATH")); parking != "" {
			sourcePath = filepath.Join(filepath.Dir(parking), "hausv.db")
		}
	}
	if sourcePath == "" {
		return fmt.Errorf("--from is required (or set DB_PATH / PARKING_DATA_PATH)")
	}
	dsn := strings.TrimSpace(getenv("DATABASE_URL"))
	if dsn == "" {
		return fmt.Errorf("DATABASE_URL is required in the environment (it is never a flag)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	source, err := db.OpenSQLiteReadOnly(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()
	fmt.Fprintf(stdout, "source: %s (read-only)\n", sourcePath)

	// The target is opened exactly as the application opens it: the same DSN
	// parsing, the same role check, the same migration runner. That is what
	// makes the schema the mover fills the schema the application will boot on.
	targetCfg := db.Config{
		Backend:      db.BackendPostgres,
		DSN:          dsn,
		MaxOpenConns: 2,
		MaxIdleConns: 2,
		LaneCap:      2,
		LaneMaxConns: 2,
		// One transaction carries the whole load; the per-statement limit only
		// needs to outlast the largest single batch and the largest hash scan.
		StatementTimeout: 5 * time.Minute,
	}
	target, err := db.OpenConfig(ctx, targetCfg)
	if err != nil {
		// Never echo the DSN: it carries the role password.
		return fmt.Errorf("open postgres target: %w", err)
	}
	defer target.Close()
	scoped, err := db.NewScoped(targetCfg, target)
	if err != nil {
		return err
	}
	defer scoped.Close()
	// The one cross-tenant decision of this command, written where the
	// inventory (internal/store TestUnscopedCallSiteInventory) records it.
	lane := scoped.Unscoped("data mover: copies every tenant's rows from SQLite into an empty PostgreSQL target before the cutover; there is no request and no tenant in scope, and a fail-closed policy would otherwise refuse every insert")
	fmt.Fprintf(stdout, "target: postgres, %d migrations applied, writing through the declared cross-tenant lane\n", len(db.PostgresMigrationNames()))

	if *verifyOnly {
		_, err := dbmove.Verify(ctx, source, lane, stdout)
		return err
	}
	report, err := dbmove.Move(ctx, source, lane, dbmove.Options{Force: *force, DryRun: *dryRun, Out: stdout})
	if err != nil {
		if errors.Is(err, dbmove.ErrTargetNotEmpty) {
			fmt.Fprintln(stderr, "the target already holds rows; if the application was already booted on PostgreSQL, those are its own — stop it, decide whether to wipe, then re-run with --force")
		}
		return err
	}
	if !report.AllMatch() {
		return fmt.Errorf("report shows a mismatch but no error was raised — refusing to report success")
	}
	return nil
}
