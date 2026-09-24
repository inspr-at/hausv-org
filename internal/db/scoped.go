package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"sync"
	"time"

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
// Handles resolve and lease a lane at the start of each operation. Each lane
// is a *sql.DB built from the parsed connection config with one extra
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

	mu      sync.Mutex
	lanes   map[string]*scopedLane
	closed  bool
	changed chan struct{}
	// order is lane keys least-recently-used first.
	order []string
}

// scopedLane is protected by Scoped.mu. A lease covers connection acquisition,
// execution, and returning the connection to the pool, including Rows and Tx.
type scopedLane struct {
	pool     *sql.DB
	leases   int
	retiring bool
	retired  chan struct{}
	closeErr error
}

var errScopedClosed = errors.New("db: scoped access is closed")

// NewScoped prepares the lane factory for an already-open process pool. It does
// not connect: lanes are lazy. PostgreSQL requires a bounded process pool and
// a parseable DSN; the owner must retain that process-pool limit.
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
		lanes:      map[string]*scopedLane{},
		changed:    make(chan struct{}),
	}
	if backend != BackendPostgres {
		return scoped, nil
	}
	if process.Stats().MaxOpenConnections <= 0 {
		return nil, fmt.Errorf("db: scoped access requires a bounded process pool")
	}
	template, err := postgresConnConfig(cfg)
	if err != nil {
		return nil, err
	}
	scoped.template = template
	scoped.baseParams = template.RuntimeParams
	return scoped, nil
}

// For returns a stable handle whose every connection is born scoped to one
// tenant. Merely holding it reserves no connections; each operation leases its
// current lane, so eviction cannot invalidate a handle waiting for first use.
func (s *Scoped) For(tenantID string) Handle {
	if s.backend != BackendPostgres {
		return s.process
	}
	value := tenantSentinel
	if validULID(tenantID) {
		value = tenantID
	}
	return scopedHandle{s: s, key: "t:" + value, param: tenantParam, value: value}
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
	return scopedHandle{s: s, key: "x", param: crossTenantParam, value: crossTenantOn}
}

// LeasePool reserves an already-declared handle's pool for integrations that
// require a concrete *sql.DB, such as database test fixtures. Prefer the handle
// for ordinary operations: its leases end automatically with each unit of work.
//
// The caller must finish all pool use before calling release, must not close or
// reconfigure the borrowed pool, and must not retain it after release. release
// is idempotent. This lease counts against LaneCap, including while idle.
func LeasePool(ctx context.Context, handle Handle) (*sql.DB, func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if pool, ok := handle.(*sql.DB); ok {
		return pool, func() {}, nil
	}
	h, ok := handle.(scopedHandle)
	if !ok {
		return nil, nil, fmt.Errorf("db: handle does not support a pool lease")
	}
	lane, err := h.s.acquire(ctx, h)
	if err != nil {
		return nil, nil, err
	}
	return lane.pool, sync.OnceFunc(func() { h.s.release(lane) }), nil
}

// acquire reserves capacity before exposing a pool. Busy lanes never overflow
// the cap: new tenants wait until a lane has no leases and can be closed.
func (s *Scoped) acquire(ctx context.Context, h scopedHandle) (*scopedLane, error) {
	for {
		s.mu.Lock()
		if err := ctx.Err(); err != nil {
			s.mu.Unlock()
			return nil, err
		}
		if s.closed {
			s.mu.Unlock()
			return nil, errScopedClosed
		}
		lane, exists := s.lanes[h.key]
		if exists && !lane.retiring {
			lane.leases++
			s.touch(h.key)
			s.mu.Unlock()
			return lane, nil
		}
		cleanupPending := false
		if !exists && len(s.lanes) >= s.laneCap {
			cleanupPending = s.evictForRoom()
		}
		if !exists && len(s.lanes) < s.laneCap {
			pool := sql.OpenDB(laneConnector{stdlib.GetConnector(s.laneConfig(h.param, h.value))})
			pool.SetMaxOpenConns(s.perLaneMax)
			pool.SetMaxIdleConns(s.perLaneMax)
			pool.SetConnMaxLifetime(s.cfg.ConnMaxLifetime)
			pool.SetConnMaxIdleTime(s.cfg.ConnMaxIdleTime)
			lane := &scopedLane{pool: pool, leases: 1, retired: make(chan struct{})}
			s.lanes[h.key] = lane
			s.order = append(s.order, h.key)
			s.mu.Unlock()
			return lane, nil
		}
		changed := s.changed
		s.mu.Unlock()

		// database/sql may still be discarding a bad/expired connection after
		// Conn.Close reports ErrConnDone. Its InUse count keeps that socket
		// budgeted until physical Close finishes. That cleanup has no public
		// notification; only in this case recheck periodically.
		var retry <-chan time.Time
		var timer *time.Timer
		if cleanupPending {
			timer = time.NewTimer(10 * time.Millisecond)
			retry = timer.C
		}
		select {
		case <-ctx.Done():
		case <-changed:
		case <-retry:
		}
		if timer != nil {
			timer.Stop()
		}
	}
}

// evictForRoom starts closing one idle, unleased pool in LRU order. Caller holds
// s.mu. The retiring lane still consumes capacity until physical Close finishes;
// closing outside the mutex keeps every capacity wait context-cancellable.
func (s *Scoped) evictForRoom() (cleanupPending bool) {
	for _, key := range s.order {
		lane := s.lanes[key]
		if lane.retiring || lane.leases != 0 {
			continue
		}
		if lane.pool.Stats().InUse != 0 {
			cleanupPending = true
			continue
		}
		lane.retiring = true
		go s.retire(key, lane)
		break
	}
	return cleanupPending
}

func (s *Scoped) retire(key string, lane *scopedLane) {
	err := lane.pool.Close()
	s.mu.Lock()
	defer s.mu.Unlock()
	lane.closeErr = err
	delete(s.lanes, key)
	for i, existing := range s.order {
		if existing == key {
			s.order = append(s.order[:i], s.order[i+1:]...)
			break
		}
	}
	close(lane.retired)
	s.notify()
}

// pgx can close a cancelled connection asynchronously. Keep database/sql's
// connection slot occupied until that cleanup really finishes, even when pgx
// has already marked the connection closed. Embedding preserves pgx's optional
// driver interfaces and argument handling.
type laneConnector struct{ driver.Connector }

func (c laneConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.Connector.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return &laneConn{Conn: conn.(*stdlib.Conn)}, nil
}

type laneConn struct{ *stdlib.Conn }

func (c *laneConn) Close() error {
	err := c.Conn.Close()
	<-c.Conn.Conn().PgConn().CleanupDone()
	return err
}

func (s *Scoped) release(lane *scopedLane) {
	s.mu.Lock()
	defer s.mu.Unlock()
	lane.leases--
	s.notify()
}

// notify wakes all capacity waiters. Caller holds s.mu.
func (s *Scoped) notify() {
	close(s.changed)
	s.changed = make(chan struct{})
}

// scopedHandle is a comparable value, not a cache entry. Repeated For calls
// share the factory and scope without retaining an unbounded map of handles.
// Its context methods are available through ContextHandle; existing stores
// keep using Handle's non-context methods unchanged.
type scopedHandle struct {
	s                 *Scoped
	key, param, value string
}

// ContextHandle is the optional cancellable surface implemented by both scoped
// handles and *sql.DB. Handle stays unchanged for existing stores and fixtures.
type ContextHandle interface {
	Handle
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

var _ ContextHandle = scopedHandle{}
var _ ContextHandle = (*sql.DB)(nil)

func (h scopedHandle) Exec(query string, args ...any) (sql.Result, error) {
	return h.ExecContext(context.Background(), query, args...)
}

func (h scopedHandle) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	lane, err := h.s.acquire(ctx, h)
	if err != nil {
		return nil, err
	}
	defer h.s.release(lane)
	return lane.pool.ExecContext(ctx, query, args...)
}

// checkout keeps the lane leased while waiting for a connection as well as
// while using it. finish must run only after the operation has begun.
func (h scopedHandle) checkout(ctx context.Context) (*sql.Conn, func(), error) {
	lane, err := h.s.acquire(ctx, h)
	if err != nil {
		return nil, nil, err
	}
	conn, err := lane.pool.Conn(ctx)
	if err != nil {
		h.s.release(lane)
		return nil, nil, err
	}
	finish := func() {
		// Conn.Close blocks until Rows/Tx releases the checked-out connection.
		// Return it before dropping the lease; otherwise eviction could close
		// the pool while a physical connection still consumes its budget.
		_ = conn.Close()
		h.s.release(lane)
	}
	return conn, finish, nil
}

func (h scopedHandle) Query(query string, args ...any) (*sql.Rows, error) {
	return h.QueryContext(context.Background(), query, args...)
}

func (h scopedHandle) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	conn, finish, err := h.checkout(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := conn.QueryContext(ctx, query, args...)
	if err != nil {
		finish()
		return nil, err
	}
	go finish()
	return rows, nil
}

func (h scopedHandle) QueryRow(query string, args ...any) *sql.Row {
	return h.QueryRowContext(context.Background(), query, args...)
}

func (h scopedHandle) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	conn, finish, err := h.checkout(ctx)
	if err != nil {
		// sql.Row's error field is private. Use database/sql itself to retain
		// its Scan/Err contract, including the original cancellation error.
		failed := sql.OpenDB(failedConnector{err: err})
		row := failed.QueryRowContext(ctx, query, args...)
		_ = failed.Close()
		return row
	}
	row := conn.QueryRowContext(ctx, query, args...)
	go finish()
	return row
}

func (h scopedHandle) Begin() (*sql.Tx, error) {
	return h.BeginTx(context.Background(), nil)
}

func (h scopedHandle) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	conn, finish, err := h.checkout(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := conn.BeginTx(ctx, opts)
	if err != nil {
		finish()
		return nil, err
	}
	go finish()
	return tx, nil
}

// failedConnector never connects; it only constructs a standard deferred Row
// error when a handle cannot acquire capacity.
type failedConnector struct{ err error }

func (c failedConnector) Connect(context.Context) (driver.Conn, error) { return nil, c.err }
func (c failedConnector) Driver() driver.Driver                        { return stdlib.GetDefaultDriver() }

// openLanes reports how many lane pools are currently held.
func (s *Scoped) openLanes() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.lanes)
}

// Budget states the peak connection plan this factory can reach.
func (s *Scoped) Budget() ConnectionBudget {
	return ConnectionBudget{LaneCap: s.laneCap, PerLaneMax: s.perLaneMax, ProcessPool: s.process.Stats().MaxOpenConnections}
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
	s.closed = true
	s.notify()
	lanes := make([]*scopedLane, 0, len(s.lanes))
	for key, lane := range s.lanes {
		lanes = append(lanes, lane)
		if !lane.retiring {
			lane.retiring = true
			go s.retire(key, lane)
		}
	}
	s.mu.Unlock()
	var firstErr error
	for _, lane := range lanes {
		<-lane.retired
		if lane.closeErr != nil && firstErr == nil {
			firstErr = lane.closeErr
		}
	}
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
