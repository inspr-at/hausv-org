package db

import (
	"context"
	"database/sql"
	"fmt"
)

// ConnectionBudget is the peak number of PostgreSQL backends this process can
// hold open at once, stated before anything is opened so a plan that cannot fit
// fails at startup instead of at the first busy minute.
//
// The obvious rule — lane_cap x per_lane_max < max_connections — is the wrong
// one, and measurably so: 94 lanes of a single connection each against a server
// with max_connections=100 died at FATAL 53300, "remaining connection slots are
// reserved for roles with the SUPERUSER attribute", well short of 100. Two
// things the naive rule ignores were the difference: the slots the server holds
// back for superusers, and this process's own unscoped pool, which is open the
// whole time the lanes are.
type ConnectionBudget struct {
	// LaneCap is the maximum number of tenant-pinned pools kept alive at once.
	LaneCap int
	// PerLaneMax is MaxOpenConns of a single lane.
	PerLaneMax int
	// ProcessPool is MaxOpenConns of the unscoped process pool.
	ProcessPool int
}

// Peak reports the largest number of server connections this plan can hold.
func (b ConnectionBudget) Peak() int {
	return b.LaneCap*b.PerLaneMax + b.ProcessPool
}

// fits compares the plan against a server's advertised limits. Strictly less
// than, not at most: a process that can consume the last usable slot leaves
// nothing for a second replica, a migration run, or a human with psql.
func (b ConnectionBudget) fits(maxConnections int, reserved int) error {
	available := maxConnections - reserved
	if b.Peak() < available {
		return nil
	}
	return fmt.Errorf(
		"db: connection budget %d (%d lanes x %d + %d process pool) does not fit %d usable connections "+
			"(max_connections %d minus %d reserved)",
		b.Peak(), b.LaneCap, b.PerLaneMax, b.ProcessPool, available, maxConnections, reserved)
}

// VerifyConnectionBudget refuses to continue when the plan cannot fit the
// server it is about to be pointed at. PostgreSQL only: SQLite has no such
// limit and callers pass their process pool a file, not a socket.
func VerifyConnectionBudget(ctx context.Context, database *sql.DB, budget ConnectionBudget) error {
	var maxConnections, reserved int
	// reserved_connections only exists from PostgreSQL 16 on, so both names are
	// summed out of pg_settings rather than read with current_setting, which
	// raises on an unknown name and would make this check version-specific.
	if err := database.QueryRowContext(ctx, `
		SELECT current_setting('max_connections')::int,
		       (SELECT coalesce(sum(setting::int), 0) FROM pg_settings
		         WHERE name IN ('superuser_reserved_connections', 'reserved_connections'))
	`).Scan(&maxConnections, &reserved); err != nil {
		return fmt.Errorf("db: read connection limits: %w", err)
	}
	return budget.fits(maxConnections, reserved)
}
