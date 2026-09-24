package db

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jackc/pgx/v5/stdlib"
)

type migrationFixture struct {
	backend Backend
	files   fs.FS
	dir     string
	open    func() *sql.DB
}

func newMigrationFixture(t *testing.T, backend Backend) migrationFixture {
	t.Helper()
	if backend == BackendSQLite {
		path := t.TempDir() + "/integrity.db"
		return migrationFixture{backend, migrationsFS, "migrations", func() *sql.DB {
			database, err := sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(1)")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = database.Close() })
			return database
		}}
	}
	baseDSN := strings.TrimSpace(os.Getenv("HAUSV_TEST_POSTGRES_DSN"))
	if baseDSN == "" {
		if os.Getenv("HAUSV_TEST_POSTGRES_REQUIRED") == "true" {
			t.Fatal("HAUSV_TEST_POSTGRES_DSN is required")
		}
		t.Skip("set HAUSV_TEST_POSTGRES_DSN to test PostgreSQL migration integrity")
	}
	cfg, err := postgresConnConfig(Config{DSN: isolatedPostgresSchema(t, baseDSN)})
	if err != nil {
		t.Fatal(err)
	}
	return migrationFixture{backend, postgresMigrationsFS, "postgres/migrations", func() *sql.DB {
		database := stdlib.OpenDB(*cfg)
		// One connection proves the advisory lock cannot deadlock its own pool.
		database.SetMaxOpenConns(1)
		t.Cleanup(func() { _ = database.Close() })
		return database
	}}
}

func (f migrationFixture) run(ctx context.Context, database *sql.DB) error {
	return migrateFiles(ctx, database, f.backend, f.files, f.dir)
}

func TestMigrationChecksumsFreshAndLegacyUpgrade(t *testing.T) {
	for _, backend := range []Backend{BackendSQLite, BackendPostgres} {
		for _, legacy := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/legacy=%t", backend, legacy), func(t *testing.T) {
				fixture := newMigrationFixture(t, backend)
				database := fixture.open()
				var timestamps map[string]string
				if legacy {
					applyWithoutChecksums(t, database, fixture)
					timestamps = migrationTimestamps(t, database)
					// Real data must survive the bootstrap without replaying SQL.
					if _, err := database.Exec(`INSERT INTO app_meta(key,value) VALUES('integrity','preserved')`); err != nil {
						t.Fatal(err)
					}
				}
				var logs bytes.Buffer
				oldLogger := slog.Default()
				slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
				defer slog.SetDefault(oldLogger)
				if err := fixture.run(t.Context(), database); err != nil {
					t.Fatalf("first startup: %v", err)
				}
				assertMigrationChecksums(t, database, fixture)
				if legacy && !reflect.DeepEqual(timestamps, migrationTimestamps(t, database)) {
					t.Fatal("backfill changed migration identities or application timestamps")
				}
				if err := database.Close(); err != nil {
					t.Fatal(err)
				}
				database = fixture.open()
				if err := fixture.run(t.Context(), database); err != nil {
					t.Fatalf("restart: %v", err)
				}
				assertMigrationChecksums(t, database, fixture)
				wantLogs := 0
				if legacy {
					wantLogs = 1
					var value string
					if err := database.QueryRow(`SELECT value FROM app_meta WHERE key='integrity'`).Scan(&value); err != nil || value != "preserved" {
						t.Fatalf("existing data changed: value=%q err=%v", value, err)
					}
				}
				if got := strings.Count(logs.String(), "trust-on-first-use"); got != wantLogs {
					t.Fatalf("backfill log count = %d, want %d", got, wantLogs)
				}
			})
		}
	}
}

// Reproduce main's old runner: original two-column ledger, full embedded SQL,
// and a separate transaction recording only the filename for each migration.
// Do not emulate an upgrade by migrating with the new runner and dropping a column.
func applyWithoutChecksums(t *testing.T, database *sql.DB, fixture migrationFixture) {
	t.Helper()
	create := `CREATE TABLE schema_migrations (version TEXT PRIMARY KEY, applied_at TEXT NOT NULL DEFAULT (datetime('now')))`
	if fixture.backend == BackendPostgres {
		create = `CREATE TABLE schema_migrations (version text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())`
	}
	if _, err := database.Exec(create); err != nil {
		t.Fatal(err)
	}
	files, err := readMigrationFiles(fixture.files, fixture.dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		tx, err := database.Begin()
		if err != nil {
			t.Fatal(err)
		}
		if _, err = tx.Exec(file.sql); err == nil {
			_, err = tx.Exec(`INSERT INTO schema_migrations(version) VALUES($1)`, file.name)
		}
		if err != nil {
			tx.Rollback()
			t.Fatalf("old runner %s: %v", file.name, err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}
	}
}

func migrationTimestamps(t *testing.T, database *sql.DB) map[string]string {
	t.Helper()
	rows, err := database.Query(`SELECT version, CAST(applied_at AS TEXT) FROM schema_migrations`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var name, stamp string
		if err := rows.Scan(&name, &stamp); err != nil {
			t.Fatal(err)
		}
		out[name] = stamp
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func assertMigrationChecksums(t *testing.T, database *sql.DB, fixture migrationFixture) {
	t.Helper()
	files, err := readMigrationFiles(fixture.files, fixture.dir)
	if err != nil {
		t.Fatal(err)
	}
	var count int
	if err := database.QueryRow(`SELECT count(*) FROM schema_migrations`).Scan(&count); err != nil || count != len(files) {
		t.Fatalf("migration count = %d, want %d: %v", count, len(files), err)
	}
	for _, file := range files {
		var got string
		if err := database.QueryRow(`SELECT checksum FROM schema_migrations WHERE version=$1`, file.name).Scan(&got); err != nil {
			t.Fatal(err)
		}
		want := fmt.Sprintf("%x", sha256.Sum256([]byte(file.sql)))
		if got != want {
			t.Fatalf("%s checksum = %q, want %q", file.name, got, want)
		}
	}
}

func copyMigrationFiles(t *testing.T, fixture migrationFixture) fstest.MapFS {
	t.Helper()
	files, err := readMigrationFiles(fixture.files, fixture.dir)
	if err != nil {
		t.Fatal(err)
	}
	out := fstest.MapFS{}
	for _, file := range files {
		out[fixture.dir+"/"+file.name] = &fstest.MapFile{Data: []byte(file.sql)}
	}
	return out
}

func TestMigrationTamperingFailsBeforePendingFilesOrBackfill(t *testing.T) {
	for _, backend := range []Backend{BackendSQLite, BackendPostgres} {
		t.Run(string(backend), func(t *testing.T) {
			fixture := newMigrationFixture(t, backend)
			database := fixture.open()
			if err := fixture.run(t.Context(), database); err != nil {
				t.Fatal(err)
			}
			files, err := readMigrationFiles(fixture.files, fixture.dir)
			if err != nil {
				t.Fatal(err)
			}
			name := files[len(files)-1].name
			changed := copyMigrationFiles(t, fixture)
			changed[fixture.dir+"/"+name].Data = append(changed[fixture.dir+"/"+name].Data, []byte("\n-- changed after apply\n")...)
			// Sorts BEFORE the tampered file: verification must be a preflight.
			changed[fixture.dir+"/0000_pending.sql"] = &fstest.MapFile{Data: []byte(`CREATE TABLE must_not_run (id integer);`)}
			if _, err := database.Exec(`UPDATE schema_migrations SET checksum=NULL WHERE version=$1`, files[0].name); err != nil {
				t.Fatal(err)
			}
			fixture.files = changed
			err = fixture.run(t.Context(), database)
			if err == nil || !strings.Contains(err.Error(), name) || !strings.Contains(err.Error(), "checksum mismatch") {
				t.Fatalf("tampered file must fail startup with filename: %v", err)
			}
			if backend == BackendPostgres {
				assertPostgresMigrationLockReleased(t, fixture.open(), []int32{postgresBackendPID(t, database)})
			}
			var count int
			if err := database.QueryRow(`SELECT count(*) FROM schema_migrations WHERE version='0000_pending.sql'`).Scan(&count); err != nil || count != 0 {
				t.Fatalf("pending file ran: count=%d err=%v", count, err)
			}
			var checksum sql.NullString
			if err := database.QueryRow(`SELECT checksum FROM schema_migrations WHERE version=$1`, files[0].name).Scan(&checksum); err != nil || checksum.Valid {
				t.Fatalf("backfill ran before integrity failure: checksum=%v err=%v", checksum, err)
			}
			// Restoring the exact original bytes allows startup again and proves
			// the failure released the PostgreSQL lock as well.
			delete(changed, fixture.dir+"/0000_pending.sql")
			changed[fixture.dir+"/"+name].Data = []byte(files[len(files)-1].sql)
			if err := fixture.run(t.Context(), database); err != nil {
				t.Fatalf("restart after restoring original file: %v", err)
			}
		})
	}
}

func TestMigrationFailureRollsBackFileAndChecksum(t *testing.T) {
	for _, backend := range []Backend{BackendSQLite, BackendPostgres} {
		t.Run(string(backend), func(t *testing.T) {
			fixture := newMigrationFixture(t, backend)
			files := fstest.MapFS{
				fixture.dir + "/0001_good.sql": {Data: []byte(`CREATE TABLE effects (id integer); INSERT INTO effects VALUES(1);`)},
				fixture.dir + "/0002_fail.sql": {Data: []byte(`INSERT INTO effects VALUES(2); SELECT * FROM missing_table;`)},
			}
			fixture.files = files
			database := fixture.open()
			if err := fixture.run(t.Context(), database); err == nil || !strings.Contains(err.Error(), "0002_fail.sql") {
				t.Fatalf("expected migration SQL failure: %v", err)
			}
			if backend == BackendPostgres {
				assertPostgresMigrationLockReleased(t, fixture.open(), []int32{postgresBackendPID(t, database)})
			}
			var effects, records int
			if err := database.QueryRow(`SELECT count(*) FROM effects`).Scan(&effects); err != nil || effects != 1 {
				t.Fatalf("file effects did not roll back: %d, %v", effects, err)
			}
			if err := database.QueryRow(`SELECT count(*) FROM schema_migrations`).Scan(&records); err != nil || records != 1 {
				t.Fatalf("failed file was recorded: %d, %v", records, err)
			}
			files[fixture.dir+"/0002_fail.sql"].Data = []byte(`INSERT INTO effects VALUES(2);`)
			if err := fixture.run(t.Context(), database); err != nil {
				t.Fatalf("retry failed: %v", err)
			}
			assertMigrationChecksums(t, database, fixture)
		})
	}
}

func TestPostgresConcurrentMigrationRunners(t *testing.T) {
	fixture := newMigrationFixture(t, BackendPostgres)
	files := copyMigrationFiles(t, fixture)
	files[fixture.dir+"/9999_exactly_once.sql"] = &fstest.MapFile{Data: []byte(`CREATE TABLE migration_effects (id integer); INSERT INTO migration_effects VALUES(1);`)}
	fixture.files = files
	first, second, blocker := fixture.open(), fixture.open(), fixture.open()
	runnerPIDs := []int32{postgresBackendPID(t, first), postgresBackendPID(t, second)}
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	if _, err := blocker.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, postgresMigrationLockKey); err != nil {
		t.Fatal(err)
	}
	defer blocker.Exec(`SELECT pg_advisory_unlock($1)`, postgresMigrationLockKey)
	results := make(chan error, 2)
	go func() { results <- fixture.run(ctx, first) }()
	go func() { results <- fixture.run(ctx, second) }()
	// Observe both independent sessions waiting before any bootstrap SQL runs.
	waitForMigrationWaiters(t, ctx, blocker, runnerPIDs)
	var exists bool
	if err := blocker.QueryRowContext(ctx, `SELECT to_regclass('schema_migrations') IS NOT NULL`).Scan(&exists); err != nil || exists {
		t.Fatalf("bootstrap escaped the lock: exists=%t err=%v", exists, err)
	}
	if _, err := blocker.ExecContext(ctx, `SELECT pg_advisory_unlock($1)`, postgresMigrationLockKey); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := <-results; err != nil {
			t.Fatalf("concurrent runner: %v", err)
		}
	}
	assertMigrationChecksums(t, first, fixture)
	var count int
	if err := first.QueryRow(`SELECT count(*) FROM migration_effects`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("migration effects = %d, want 1: %v", count, err)
	}
	assertPostgresMigrationLockReleased(t, blocker, runnerPIDs)
}

func TestPostgresMigrationLockWaitCancellation(t *testing.T) {
	fixture := newMigrationFixture(t, BackendPostgres)
	blocker, runner := fixture.open(), fixture.open()
	waiterPID := postgresBackendPID(t, runner)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	if _, err := blocker.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, postgresMigrationLockKey); err != nil {
		t.Fatal(err)
	}
	defer blocker.Exec(`SELECT pg_advisory_unlock($1)`, postgresMigrationLockKey)
	waitCtx, stopWait := context.WithCancel(ctx)
	defer stopWait()
	result := make(chan error, 1)
	go func() { result <- fixture.run(waitCtx, runner) }()
	waitForMigrationWaiters(t, ctx, blocker, []int32{waiterPID})
	stopWait()
	if err := <-result; err == nil || !strings.Contains(err.Error(), "acquire postgres migration lock") {
		t.Fatalf("cancelled waiter must fail: %v", err)
	}
	if _, err := blocker.ExecContext(ctx, `SELECT pg_advisory_unlock($1)`, postgresMigrationLockKey); err != nil {
		t.Fatal(err)
	}
	if err := fixture.run(ctx, runner); err != nil {
		t.Fatalf("restart after cancellation: %v", err)
	}
	assertPostgresMigrationLockReleased(t, blocker, []int32{waiterPID, postgresBackendPID(t, runner)})
}

func postgresBackendPID(t *testing.T, database *sql.DB) int32 {
	t.Helper()
	var pid int32
	if err := database.QueryRow(`SELECT pg_backend_pid()`).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	return pid
}

func waitForMigrationWaiters(t *testing.T, ctx context.Context, database *sql.DB, pids []int32) {
	t.Helper()
	for {
		var count int
		err := database.QueryRowContext(ctx, `SELECT count(*) FROM pg_locks WHERE locktype='advisory'
			AND database=(SELECT oid FROM pg_database WHERE datname=current_database())
			AND classid=$1 AND objid=$2 AND objsubid=1 AND NOT granted AND pid=ANY($3)`,
			postgresMigrationLockKey>>32, postgresMigrationLockKey&0xffffffff, pids).Scan(&count)
		if err != nil {
			t.Fatal(err)
		}
		if count == len(pids) {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatal("migration runners did not wait on the advisory lock")
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func assertPostgresMigrationLockReleased(t *testing.T, database *sql.DB, pids []int32) {
	t.Helper()
	// Other test packages legitimately migrate other schemas in this database.
	// Assert OUR sessions released the lock, not that nobody else acquired it.
	var count int
	err := database.QueryRow(`SELECT count(*) FROM pg_locks WHERE locktype='advisory'
		AND classid=$1 AND objid=$2 AND objsubid=1 AND pid=ANY($3)`,
		postgresMigrationLockKey>>32, postgresMigrationLockKey&0xffffffff, pids).Scan(&count)
	if err != nil || count != 0 {
		t.Fatalf("migration lock leaked by runner sessions: count=%d err=%v", count, err)
	}
}
