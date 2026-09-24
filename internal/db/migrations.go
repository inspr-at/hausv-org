package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"strings"
	"time"
)

// postgresMigrationLockKey is the fixed, database-local advisory-lock namespace
// for HAUSV migrations: ASCII "HAUSV" followed by "MIG". All schemas in one
// database deliberately share it. Keep it stable across releases/processes.
const postgresMigrationLockKey int64 = 0x48415553564d4947

type migrationFile struct {
	name, sql, checksum string
}

// migrateFiles pins one connection for the entire run. PostgreSQL's session
// lock covers bootstrap, integrity verification/backfill and every pending file,
// while preserving per-file transactions (including SET LOCAL lifetimes).
func migrateFiles(ctx context.Context, database *sql.DB, backend Backend, files fs.FS, dir string) (retErr error) {
	migrations, err := readMigrationFiles(files, dir)
	if err != nil {
		return err
	}
	conn, err := database.Conn(ctx)
	if err != nil {
		return fmt.Errorf("db: acquire %s migration connection: %w", backend, err)
	}
	defer conn.Close()
	if backend == BackendPostgres {
		if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, postgresMigrationLockKey); err != nil {
			// A cancellation can race with acquisition. Never return a possibly
			// locked session to the pool, even when acquisition reported an error.
			_ = conn.Raw(func(any) error { return driver.ErrBadConn })
			return fmt.Errorf("db: acquire postgres migration lock: %w", err)
		}
		defer func() {
			// Release even after the startup context is cancelled. If cleanup
			// fails, physically discard the connection to release its locks.
			unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			var unlocked bool
			err := conn.QueryRowContext(unlockCtx, `SELECT pg_advisory_unlock($1)`, postgresMigrationLockKey).Scan(&unlocked)
			if err == nil && !unlocked {
				err = fmt.Errorf("session did not hold the migration lock")
			}
			if err != nil {
				_ = conn.Raw(func(any) error { return driver.ErrBadConn })
				retErr = errors.Join(retErr, fmt.Errorf("db: release postgres migration lock: %w", err))
			}
		}()
	}

	applied, err := prepareMigrationLedger(ctx, conn, backend, migrations)
	if err != nil {
		return err
	}
	for _, migration := range migrations {
		if _, ok := applied[migration.name]; ok {
			continue
		}
		if err := applyMigrationFile(ctx, conn, migration); err != nil {
			return err
		}
	}
	return nil
}

func readMigrationFiles(files fs.FS, dir string) ([]migrationFile, error) {
	entries, err := fs.ReadDir(files, dir) // fs.ReadDir sorts by filename.
	if err != nil {
		return nil, fmt.Errorf("db: read migrations: %w", err)
	}
	var migrations []migrationFile
	for _, entry := range entries {
		name := entry.Name()
		// Ignore macOS AppleDouble sidecars, just as the original runners did.
		if entry.IsDir() || strings.HasPrefix(name, ".") || !strings.HasSuffix(name, ".sql") {
			continue
		}
		raw, err := fs.ReadFile(files, dir+"/"+name)
		if err != nil {
			return nil, fmt.Errorf("db: read migration %s: %w", name, err)
		}
		migrations = append(migrations, migrationFile{name, string(raw), fmt.Sprintf("%x", sha256.Sum256(raw))})
	}
	return migrations, nil
}

// prepareMigrationLedger is an additive bootstrap, not a numbered migration.
// A nullable column lets old binaries keep recording filenames on rollback.
// Only NULL is trusted on first use; an empty or different hash is corruption.
// Validate every applied embedded file before backfilling or applying anything.
func prepareMigrationLedger(ctx context.Context, conn *sql.Conn, backend Backend, migrations []migrationFile) (map[string]sql.NullString, error) {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("db: begin migration bootstrap: %w", err)
	}
	defer tx.Rollback()
	create := `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`
	if backend == BackendPostgres {
		create = `CREATE TABLE IF NOT EXISTS schema_migrations (
			version text PRIMARY KEY,
			applied_at timestamptz NOT NULL DEFAULT now()
		)`
	}
	if _, err := tx.ExecContext(ctx, create); err != nil {
		return nil, fmt.Errorf("db: ensure %s schema_migrations: %w", backend, err)
	}
	if backend == BackendPostgres {
		_, err = tx.ExecContext(ctx, `ALTER TABLE schema_migrations ADD COLUMN IF NOT EXISTS checksum text`)
	} else {
		// SQLite has no ADD COLUMN IF NOT EXISTS; probe on the same transaction.
		var count int
		err = tx.QueryRowContext(ctx, `SELECT count(*) FROM pragma_table_info('schema_migrations') WHERE name = 'checksum'`).Scan(&count)
		if err == nil && count == 0 {
			_, err = tx.ExecContext(ctx, `ALTER TABLE schema_migrations ADD COLUMN checksum TEXT`)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("db: bootstrap %s migration checksums: %w", backend, err)
	}
	rows, err := tx.QueryContext(ctx, `SELECT version, checksum FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("db: read migration ledger: %w", err)
	}
	applied := make(map[string]sql.NullString)
	for rows.Next() {
		var name string
		var checksum sql.NullString
		if err := rows.Scan(&name, &checksum); err != nil {
			rows.Close()
			return nil, fmt.Errorf("db: scan migration ledger: %w", err)
		}
		applied[name] = checksum
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, fmt.Errorf("db: iterate migration ledger: %w", err)
	}
	for _, migration := range migrations {
		if stored, ok := applied[migration.name]; ok && stored.Valid && stored.String != migration.checksum {
			return nil, fmt.Errorf("db: %s migration %s checksum mismatch: applied file differs from embedded content; restore the original migration file", backend, migration.name)
		}
	}
	backfilled := 0
	for _, migration := range migrations {
		if stored, ok := applied[migration.name]; ok && !stored.Valid {
			if _, err := tx.ExecContext(ctx, `UPDATE schema_migrations SET checksum = $1 WHERE version = $2`, migration.checksum, migration.name); err != nil {
				return nil, fmt.Errorf("db: backfill migration %s checksum: %w", migration.name, err)
			}
			backfilled++
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("db: commit migration bootstrap: %w", err)
	}
	if backfilled > 0 {
		slog.Info("db: migration checksums backfilled from embedded files (trust-on-first-use)", "backend", backend, "migrations", backfilled)
	}
	return applied, nil
}

func applyMigrationFile(ctx context.Context, conn *sql.Conn, migration migrationFile) error {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("db: begin migration %s: %w", migration.name, err)
	}
	defer tx.Rollback()
	// Execute the whole file: both drivers handle statements and SQL comments.
	if _, err := tx.ExecContext(ctx, migration.sql); err != nil {
		return fmt.Errorf("db: migration %s failed: %w", migration.name, err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(version, checksum) VALUES($1, $2)`, migration.name, migration.checksum); err != nil {
		return fmt.Errorf("db: record migration %s: %w", migration.name, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("db: commit migration %s: %w", migration.name, err)
	}
	return nil
}
