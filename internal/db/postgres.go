package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// Backend names a supported database implementation. An empty backend keeps
// SQLite as the default so existing callers and deployments are unchanged.
type Backend string

const (
	BackendSQLite   Backend = "sqlite"
	BackendPostgres Backend = "postgres"
)

const (
	defaultConnectTimeout   = 5 * time.Second
	defaultStatementTimeout = 30 * time.Second
	defaultConnMaxLifetime  = 30 * time.Minute
	defaultConnMaxIdleTime  = 5 * time.Minute
	defaultMaxOpenConns     = 20
	defaultMaxIdleConns     = 5
)

// Config describes a database connection without reading process state. The
// composition root can select a backend later without changing store code.
type Config struct {
	Backend          Backend
	DSN              string
	ConnectTimeout   time.Duration
	StatementTimeout time.Duration
	MaxOpenConns     int
	MaxIdleConns     int
	ConnMaxLifetime  time.Duration
	ConnMaxIdleTime  time.Duration
}

//go:embed postgres/migrations/*.sql
var postgresMigrationsFS embed.FS

// OpenConfig opens the selected backend and applies its migrations. PostgreSQL
// remains opt-in; Open and an empty Backend continue to select SQLite.
func OpenConfig(ctx context.Context, cfg Config) (*sql.DB, error) {
	backend := Backend(strings.ToLower(strings.TrimSpace(string(cfg.Backend))))
	if backend == "" {
		backend = BackendSQLite
	}
	switch backend {
	case BackendSQLite:
		return openSQLite(cfg.DSN)
	case BackendPostgres:
		return openPostgres(ctx, cfg)
	default:
		return nil, fmt.Errorf("db: unsupported backend %q", backend)
	}
}

func openPostgres(ctx context.Context, cfg Config) (*sql.DB, error) {
	if strings.TrimSpace(cfg.DSN) == "" {
		return nil, fmt.Errorf("db: postgres DSN required")
	}
	applyPostgresDefaults(&cfg)
	pgCfg, err := pgx.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("db: parse postgres DSN: %w", err)
	}
	pgCfg.ConnectTimeout = cfg.ConnectTimeout
	if pgCfg.RuntimeParams == nil {
		pgCfg.RuntimeParams = make(map[string]string)
	}
	pgCfg.RuntimeParams["statement_timeout"] = strconv.FormatInt(cfg.StatementTimeout.Milliseconds(), 10)

	database := stdlib.OpenDB(*pgCfg)
	database.SetMaxOpenConns(cfg.MaxOpenConns)
	database.SetMaxIdleConns(cfg.MaxIdleConns)
	database.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	database.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	pingCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()
	if err := database.PingContext(pingCtx); err != nil {
		database.Close()
		return nil, fmt.Errorf("db: ping postgres: %w", err)
	}
	if err := verifyPostgresRole(ctx, database); err != nil {
		database.Close()
		return nil, err
	}
	if err := migratePostgres(ctx, database); err != nil {
		database.Close()
		return nil, err
	}
	return database, nil
}

func applyPostgresDefaults(cfg *Config) {
	if cfg.ConnectTimeout <= 0 {
		cfg.ConnectTimeout = defaultConnectTimeout
	}
	if cfg.StatementTimeout <= 0 {
		cfg.StatementTimeout = defaultStatementTimeout
	}
	if cfg.MaxOpenConns <= 0 {
		cfg.MaxOpenConns = defaultMaxOpenConns
	}
	if cfg.MaxIdleConns <= 0 {
		cfg.MaxIdleConns = defaultMaxIdleConns
	}
	if cfg.MaxIdleConns > cfg.MaxOpenConns {
		cfg.MaxIdleConns = cfg.MaxOpenConns
	}
	if cfg.ConnMaxLifetime <= 0 {
		cfg.ConnMaxLifetime = defaultConnMaxLifetime
	}
	if cfg.ConnMaxIdleTime <= 0 {
		cfg.ConnMaxIdleTime = defaultConnMaxIdleTime
	}
}

func verifyPostgresRole(ctx context.Context, database *sql.DB) error {
	var superuser, bypassRLS bool
	if err := database.QueryRowContext(ctx, `
		SELECT rolsuper, rolbypassrls FROM pg_roles WHERE rolname = current_user
	`).Scan(&superuser, &bypassRLS); err != nil {
		return fmt.Errorf("db: inspect postgres role: %w", err)
	}
	if superuser || bypassRLS {
		return fmt.Errorf("db: postgres application role must be NOSUPERUSER NOBYPASSRLS")
	}
	return nil
}

func migratePostgres(ctx context.Context, database *sql.DB) error {
	if _, err := database.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version text PRIMARY KEY,
		applied_at timestamptz NOT NULL DEFAULT now()
	)`); err != nil {
		return fmt.Errorf("db: ensure postgres schema_migrations: %w", err)
	}
	entries, err := postgresMigrationsFS.ReadDir("postgres/migrations")
	if err != nil {
		return fmt.Errorf("db: read postgres migrations: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() && !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".sql") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		tx, err := database.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("db: begin postgres migration %s: %w", name, err)
		}
		var applied bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)`, name).Scan(&applied); err != nil {
			tx.Rollback()
			return fmt.Errorf("db: check postgres migration %s: %w", name, err)
		}
		if applied {
			tx.Rollback()
			continue
		}
		raw, err := postgresMigrationsFS.ReadFile("postgres/migrations/" + name)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("db: read postgres migration %s: %w", name, err)
		}
		if _, err := tx.ExecContext(ctx, string(raw)); err != nil {
			tx.Rollback()
			return fmt.Errorf("db: postgres migration %s failed: %w", name, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(version) VALUES($1)`, name); err != nil {
			tx.Rollback()
			return fmt.Errorf("db: record postgres migration %s: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("db: commit postgres migration %s: %w", name, err)
		}
	}
	return nil
}

// BeginTenantTx starts a transaction and sets its RLS tenant scope with SET
// LOCAL. PostgreSQL discards the setting at commit/rollback, even when the
// underlying pooled connection is reused.
func BeginTenantTx(ctx context.Context, database *sql.DB, tenantID string, opts *sql.TxOptions) (*sql.Tx, error) {
	if !validULID(tenantID) {
		return nil, fmt.Errorf("db: tenant ID must be an uppercase ULID")
	}
	tx, err := database.BeginTx(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("db: begin tenant transaction: %w", err)
	}
	// tenantID has been reduced to the fixed ULID alphabet above, so it cannot
	// alter this SET statement. PostgreSQL does not parameterize SET LOCAL.
	if _, err := tx.ExecContext(ctx, "SET LOCAL hausv.tenant_id = '"+tenantID+"'"); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("db: set tenant transaction scope: %w", err)
	}
	return tx, nil
}

func validULID(value string) bool {
	if len(value) != 26 || value[0] > '7' {
		return false
	}
	for _, char := range value {
		if !strings.ContainsRune("0123456789ABCDEFGHJKMNPQRSTVWXYZ", char) {
			return false
		}
	}
	return true
}
