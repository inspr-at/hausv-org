// Package db owns the database connections and forward-only migrations.
//
// Driver: modernc.org/sqlite — pure Go, so the CGO_ENABLED=0 distroless build
// keeps its static binary (mattn/go-sqlite3 would break it). Pragmas are set via
// the DSN so they apply to every pooled connection (foreign_keys is per-conn).
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Open opens (creating if needed) the SQLite database at path with WAL mode and
// sensible pragmas, then applies every pending migration. It is safe to call on
// each boot: migrations already recorded are skipped.
func Open(path string) (*sql.DB, error) {
	return OpenConfig(context.Background(), Config{DSN: path})
}

func openSQLite(path string) (*sql.DB, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("db: path required")
	}
	dsn := "file:" + path +
		"?_pragma=busy_timeout(5000)" +
		"&_pragma=foreign_keys(1)" +
		"&_pragma=journal_mode(WAL)" +
		"&_pragma=synchronous(NORMAL)"
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("db: open %s: %w", path, err)
	}
	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("db: ping %s: %w", path, err)
	}
	if err := migrate(sqlDB); err != nil {
		sqlDB.Close()
		return nil, err
	}
	return sqlDB, nil
}

// migrate applies embedded migrations/*.sql in lexical order inside a
// transaction each, recording applied files in schema_migrations. A file is
// applied at most once; a failing migration rolls back and aborts (a broken
// schema must not boot silently).
func migrate(sqlDB *sql.DB) error {
	if _, err := sqlDB.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version    TEXT PRIMARY KEY,
		applied_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("db: ensure schema_migrations: %w", err)
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("db: read migrations: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		// Skip dotfiles: macOS tar (bsdtar) injects AppleDouble "._name" sidecars
		// into the build context, which go:embed would otherwise pick up as junk
		// migrations. Belt to COPYFILE_DISABLE=1's suspenders on the deploy side.
		if e.IsDir() || strings.HasPrefix(name, ".") || !strings.HasSuffix(name, ".sql") {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		var applied int
		if err := sqlDB.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", name).Scan(&applied); err != nil {
			return fmt.Errorf("db: check migration %s: %w", name, err)
		}
		if applied > 0 {
			continue
		}
		raw, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("db: read migration %s: %w", name, err)
		}
		tx, err := sqlDB.Begin()
		if err != nil {
			return fmt.Errorf("db: begin migration %s: %w", name, err)
		}
		// Execute the whole file in one Exec: modernc.org/sqlite runs every
		// statement, and SQLite itself handles comments (including ";" inside a
		// comment) — naive ";"-splitting would break on those.
		if _, err := tx.Exec(string(raw)); err != nil {
			tx.Rollback()
			return fmt.Errorf("db: migration %s failed: %w", name, err)
		}
		if _, err := tx.Exec("INSERT INTO schema_migrations(version) VALUES(?)", name); err != nil {
			tx.Rollback()
			return fmt.Errorf("db: record migration %s: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("db: commit migration %s: %w", name, err)
		}
	}
	return nil
}
