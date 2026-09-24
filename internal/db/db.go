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

// migrate retains SQLite's single-process startup contract. SQLite is used by
// offline legacy tools and tests, not a shared product server; callers must not
// migrate the same file concurrently. No separate filesystem lock is required
// within that contract (the SQL transactions still protect each file's writes).
func migrate(sqlDB *sql.DB) error {
	return migrateFiles(context.Background(), sqlDB, BackendSQLite, migrationsFS, "migrations")
}
