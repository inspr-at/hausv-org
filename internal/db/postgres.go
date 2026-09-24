package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
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
	// defaultLaneCap x defaultLaneMaxConns plus defaultMaxOpenConns must stay
	// under a stock server's usable connections: 24 x 3 + 20 = 92 against
	// max_connections 100 less 3 reserved. VerifyBudget checks it for real.
	defaultLaneCap      = 24
	defaultLaneMaxConns = 3
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
	// LaneCap bounds how many tenant-pinned pools Scoped keeps alive at once.
	LaneCap int
	// LaneMaxConns is MaxOpenConns of a single tenant lane.
	LaneMaxConns int
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
	pgCfg, err := postgresConnConfig(cfg)
	if err != nil {
		return nil, err
	}

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

// postgresConnConfig parses the DSN and applies every tuning decision a
// connection from this process carries. It is the ONLY place that does so:
// tenant lanes are built from the same function, because a lane assembled from
// the DSN alone comes back untuned — measured, statement_timeout 0 where the
// process pool says 30s — and nothing about the resulting pool looks wrong until
// a pathological query holds a backend open forever.
//
// cfg is taken by value and defaulted here, so a caller cannot get a config
// tuned against numbers it did not see.
func postgresConnConfig(cfg Config) (*pgx.ConnConfig, error) {
	applyPostgresDefaults(&cfg)
	pgCfg, err := pgx.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("db: parse postgres DSN: %w", err)
	}
	pgCfg.ConnectTimeout = cfg.ConnectTimeout
	// A fresh map, never the parsed config's own and never one another pool may
	// still be holding. See Scoped.lane for the trap this guards.
	params := make(map[string]string, len(pgCfg.RuntimeParams)+2)
	for name, value := range pgCfg.RuntimeParams {
		params[name] = value
	}
	params["statement_timeout"] = strconv.FormatInt(cfg.StatementTimeout.Milliseconds(), 10)
	pgCfg.RuntimeParams = params
	return pgCfg, nil
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
		// Equal to MaxOpenConns on purpose. A lower idle cap makes the pool close
		// connections it is about to need again: at 20 open / 5 idle a burst the
		// pool was sized for cost fifteen measured idle-closes of pure churn, each
		// one a TCP setup, a TLS handshake and a fresh backend on the server.
		// ConnMaxIdleTime, not the idle cap, is what returns unused connections.
		cfg.MaxIdleConns = cfg.MaxOpenConns
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
	if cfg.LaneCap <= 0 {
		cfg.LaneCap = defaultLaneCap
	}
	if cfg.LaneMaxConns <= 0 {
		cfg.LaneMaxConns = defaultLaneMaxConns
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
	return migrateFiles(ctx, database, BackendPostgres, postgresMigrationsFS, "postgres/migrations")
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
