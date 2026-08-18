package db

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// Handle is the database surface the store layer already uses: the four
// non-context methods of *sql.DB that every SQL store calls, and nothing else.
// *sql.DB satisfies it unchanged, and so does anything this package hands back,
// so a store can be pointed at a scoped connection without a single call site
// changing shape.
type Handle interface {
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
	Exec(query string, args ...any) (sql.Result, error)
	Begin() (*sql.Tx, error)
}

// Compile-time proof that the plain pool is a Handle: the whole seam rests on
// being able to hand back either one.
var _ Handle = (*sql.DB)(nil)

const (
	tenantParam      = "hausv.tenant_id"
	crossTenantParam = "hausv.cross_tenant"
	// crossTenantOn is the value a declared maintenance lane sends.
	crossTenantOn = "on"
	// tenantSentinel is what an unusable tenant id becomes. A malformed or empty
	// id must not error at the call site — there are 130-odd of them — and it must
	// not silently become "no scope": under migration 0003 that was the whole
	// database, and since 0006 it is nothing, which reads as fail-closed but for
	// the wrong reason. A value no row can ever carry fails closed instead, and
	// keeps doing so whichever way the policy is written.
	tenantSentinel = "-"
)

// Scoped hands out database handles whose tenant scope is established when the
// connection is BORN, not by running SQL on a connection that already exists.
//
// Each lane is a *sql.DB built from the parsed connection config with one extra
// RuntimeParams entry, which pgx sends in the PostgreSQL startup packet. That
// placement is the point: the scope is part of the session's initial state, so
// it survives RESET, RESET ALL, SET ... = DEFAULT and DISCARD ALL (all of which
// restore the value the startup packet carried), and it cannot leak into the
// next borrower of a pooled connection because every connection in the lane was
// born with the same scope. A *sql.Tx opened on a lane inherits it.
//
// On SQLite there is no such thing, and no RLS to need it: every accessor
// returns the one process pool.
type Scoped struct {
	backend  Backend
	process  *sql.DB
	template *pgx.ConnConfig
	cfg      Config

	// baseParams is the tuned runtime-parameter set every lane starts from. It
	// is read-only after construction and is COPIED into each lane, never
	// handed out.
	baseParams map[string]string

	laneCap    int
	perLaneMax int

	mu    sync.Mutex
	lanes map[string]*sql.DB
	// order is lane keys least-recently-used first.
	order []string
}

// NewScoped prepares the lane factory for an already-open process pool. It does
// not connect: lanes are lazy, and the only thing that can fail here is parsing
// a DSN that openPostgres has already parsed once.
func NewScoped(cfg Config, process *sql.DB) (*Scoped, error) {
	if process == nil {
		return nil, fmt.Errorf("db: scoped access requires an open pool")
	}
	backend := normalizeBackend(cfg.Backend)
	if backend == BackendPostgres {
		applyPostgresDefaults(&cfg)
	}
	scoped := &Scoped{
		backend:    backend,
		process:    process,
		cfg:        cfg,
		laneCap:    cfg.LaneCap,
		perLaneMax: cfg.LaneMaxConns,
		lanes:      map[string]*sql.DB{},
	}
	if backend != BackendPostgres {
		return scoped, nil
	}
	template, err := postgresConnConfig(cfg)
	if err != nil {
		return nil, err
	}
	scoped.template = template
	scoped.baseParams = template.RuntimeParams
	return scoped, nil
}

// For returns a handle whose every connection is born scoped to one tenant.
func (s *Scoped) For(tenantID string) Handle {
	if s.backend != BackendPostgres {
		return s.process
	}
	value := tenantSentinel
	if validULID(tenantID) {
		value = tenantID
	}
	return s.lane("t:"+value, tenantParam, value)
}

// Unscoped returns the declared maintenance lane: a handle that can see across
// tenants. Since PostgreSQL migration 0006 this declaration is the ONLY way a
// session sees more than one tenant — an undeclared session sees nothing.
//
// reason is not read here and is not meant to be. It exists so that crossing
// tenants cannot be done without writing down why, in the call itself, where
// review and grep will find it. All maintenance callers share one lane.
func (s *Scoped) Unscoped(reason string) Handle {
	if s.backend != BackendPostgres {
		return s.process
	}
	return s.lane("x", crossTenantParam, crossTenantOn)
}

func (s *Scoped) lane(key string, param string, value string) Handle {
	s.mu.Lock()
	defer s.mu.Unlock()
	if lane, ok := s.lanes[key]; ok {
		s.touch(key)
		return lane
	}
	s.evictForRoom()
	lane := stdlib.OpenDB(s.laneConfig(param, value))
	lane.SetMaxOpenConns(s.perLaneMax)
	lane.SetMaxIdleConns(s.perLaneMax)
	lane.SetConnMaxLifetime(s.cfg.ConnMaxLifetime)
	lane.SetConnMaxIdleTime(s.cfg.ConnMaxIdleTime)
	s.lanes[key] = lane
	s.order = append(s.order, key)
	return lane
}

// evictForRoom makes space for one more lane, least-recently-used first.
//
// It skips any pool with connections checked out. Evicting means Close(), and
// the caller is still holding the Handle it was given: a store takes a handle,
// begins, runs several statements and commits, without re-asking the cache in
// between. Closing that pool underneath it makes the next statement fail with
// "sql: database is closed" — measured, and a cache-size policy has no business
// failing a request.
//
// So when every candidate is busy the cache runs over its cap for a while, and
// the next call — after those connections go back — brings it down again. That
// is a bounded overshoot, not a leak: connections inside a surviving lane are
// still returned by ConnMaxIdleTime, which reaped 62 idle backends to 2 in the
// measurement this is sized from.
//
// Caller holds s.mu.
func (s *Scoped) evictForRoom() {
	for len(s.lanes) >= s.laneCap {
		victim := -1
		for i, key := range s.order {
			if lane, ok := s.lanes[key]; ok && lane.Stats().InUse == 0 {
				victim = i
				break
			}
		}
		if victim < 0 {
			return
		}
		key := s.order[victim]
		if lane, ok := s.lanes[key]; ok {
			_ = lane.Close()
		}
		delete(s.lanes, key)
		s.order = append(s.order[:victim], s.order[victim+1:]...)
	}
}

// openLanes reports how many lane pools are currently held.
func (s *Scoped) openLanes() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.lanes)
}

// Budget states the peak connection plan this factory can reach.
func (s *Scoped) Budget() ConnectionBudget {
	return ConnectionBudget{LaneCap: s.laneCap, PerLaneMax: s.perLaneMax, ProcessPool: s.cfg.MaxOpenConns}
}

// VerifyBudget is the startup assertion: refuse to boot a plan the server
// cannot serve, instead of discovering it as connection refusals under load.
// SQLite has neither lanes nor a connection limit, so there is nothing to check.
func (s *Scoped) VerifyBudget(ctx context.Context) error {
	if s.backend != BackendPostgres {
		return nil
	}
	return VerifyConnectionBudget(ctx, s.process, s.Budget())
}

// laneConfig produces the connection config for one lane.
//
// The dereference below reads like a deep copy and is not one. stdlib.OpenDB
// takes a pgx.ConnConfig BY VALUE, but RuntimeParams is a map, so a struct copy
// copies the map HEADER: every pool built this way would share one map. Setting
// the tenant for a second lane then silently re-points the first — measured, and
// invisible, because database/sql connects lazily, so the first lane's very
// first query already runs under the second lane's tenant.
//
// So the map is REPLACED with a fresh copy of the tuned base, never mutated in
// place. That is the whole fix, and it is one line that must not be simplified.
func (s *Scoped) laneConfig(param string, value string) pgx.ConnConfig {
	cfg := *s.template
	params := make(map[string]string, len(s.baseParams)+1)
	for name, base := range s.baseParams {
		params[name] = base
	}
	params[param] = value
	cfg.RuntimeParams = params
	return cfg
}

// Close closes every lane. The process pool is not closed here: its owner opened
// it and closes it.
func (s *Scoped) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var firstErr error
	for key, lane := range s.lanes {
		if err := lane.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		delete(s.lanes, key)
	}
	s.order = nil
	return firstErr
}

// touch moves a key to the most-recently-used end.
func (s *Scoped) touch(key string) {
	for i, existing := range s.order {
		if existing == key {
			s.order = append(append(s.order[:i:i], s.order[i+1:]...), key)
			return
		}
	}
	s.order = append(s.order, key)
}

func normalizeBackend(backend Backend) Backend {
	if backend == "" {
		return BackendSQLite
	}
	return backend
}
