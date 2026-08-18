package db

import (
	"database/sql"
	"fmt"
	"os"
	"sort"
	"strings"
)

// OpenSQLiteReadOnly opens an existing SQLite database without running a single
// migration and without the ability to write to it.
//
// It exists for the data mover, which reads a production file that the running
// application owns. Open would run the migration runner against it, and while
// that is a no-op on a file the same binary already migrated, a copy tool has
// no business holding a write lock on the file it is copying from. mode=ro
// makes every write fail at the driver, so a defect in the mover cannot become
// a change in the source.
//
// The file must exist: SQLite's default is to create a missing file, and a
// mover pointed at a typo would otherwise happily copy zero rows out of an empty
// database it just created.
func OpenSQLiteReadOnly(path string) (*sql.DB, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("db: path required")
	}
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("db: source %s: %w", path, err)
	}
	dsn := "file:" + path +
		"?mode=ro" +
		"&_pragma=busy_timeout(5000)" +
		"&_pragma=query_only(1)"
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("db: open %s read-only: %w", path, err)
	}
	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("db: ping %s: %w", path, err)
	}
	return sqlDB, nil
}

// SQLiteMigrationNames lists the embedded SQLite migrations in the order the
// runner applies them. A tool that reads a SQLite file this binary did not
// migrate itself can compare the file's schema_migrations against this list and
// refuse a file that is behind or ahead of the code it is running.
func SQLiteMigrationNames() []string {
	return embeddedMigrationNames(migrationsFS, "migrations")
}

// PostgresMigrationNames is the PostgreSQL counterpart of SQLiteMigrationNames.
func PostgresMigrationNames() []string {
	return embeddedMigrationNames(postgresMigrationsFS, "postgres/migrations")
}

func embeddedMigrationNames(fsys interface {
	ReadDir(name string) ([]os.DirEntry, error)
}, dir string) []string {
	entries, err := fsys.ReadDir(dir)
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || strings.HasPrefix(name, ".") || !strings.HasSuffix(name, ".sql") {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
