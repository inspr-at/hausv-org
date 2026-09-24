package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func smallScoped(t *testing.T) (*Scoped, *sql.DB) {
	t.Helper()
	return scopedPostgresConfig(t, func(cfg *Config) {
		cfg.LaneCap = 1
		cfg.LaneMaxConns = 1
	})
}

// This is A06's exact delayed-first-use probe, followed by real churn both
// before Begin and between uses of the same retained handle.
func TestBorrowedHandleSurvivesEvictionBeforeBegin(t *testing.T) {
	s, _ := smallScoped(t)
	held := s.For(laneID(0))
	s.For(laneID(1))
	for round := range 3 {
		for i := 1; i < 8; i++ {
			if got := scopeOf(t, s.For(laneID(i))); got != laneID(i) {
				t.Fatalf("churn scope = %q", got)
			}
		}
		tx, err := held.Begin()
		if err != nil {
			t.Fatalf("retained handle Begin, round %d: %v", round, err)
		}
		var scope string
		err = tx.QueryRow(`SELECT current_setting('hausv.tenant_id')`).Scan(&scope)
		commitErr := tx.Commit()
		if err != nil || commitErr != nil || scope != laneID(0) {
			t.Fatalf("scope=%q query=%v commit=%v", scope, err, commitErr)
		}
	}
}

func TestScopedUnitOfWorkPinsItsLane(t *testing.T) {
	for _, kind := range []string{"commit", "rollback", "rows-close", "rows-eof", "row-scan", "rows-cancel", "tx-cancel"} {
		t.Run(kind, func(t *testing.T) {
			s, _ := smallScoped(t)
			h := s.For(scopedTenantA).(ContextHandle)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			var finish func()
			switch kind {
			case "commit", "rollback", "tx-cancel":
				tx, err := h.BeginTx(ctx, nil)
				if err != nil {
					t.Fatal(err)
				}
				defer tx.Rollback()
				finish = func() {
					if kind == "tx-cancel" {
						cancel()
						return
					}
					var scope string
					if err := tx.QueryRow(`SELECT current_setting('hausv.tenant_id')`).Scan(&scope); err != nil || scope != scopedTenantA {
						t.Fatalf("held transaction: scope=%q err=%v", scope, err)
					}
					var err error
					if kind == "commit" {
						err = tx.Commit()
					} else {
						err = tx.Rollback()
					}
					if err != nil {
						t.Fatal(err)
					}
				}
			case "row-scan":
				row := h.QueryRowContext(ctx, `SELECT current_setting('hausv.tenant_id')`)
				finish = func() {
					var scope string
					if err := row.Scan(&scope); err != nil || scope != scopedTenantA {
						t.Fatalf("held Row: scope=%q err=%v", scope, err)
					}
				}
			default:
				rows, err := h.QueryContext(ctx, `SELECT current_setting('hausv.tenant_id') FROM generate_series(1, 3)`)
				if err != nil {
					t.Fatal(err)
				}
				defer rows.Close()
				finish = func() {
					if kind == "rows-cancel" {
						cancel()
						return
					}
					if kind == "rows-close" {
						if err := rows.Close(); err != nil {
							t.Fatal(err)
						}
						return
					}
					n := 0
					for rows.Next() {
						var scope string
						if err := rows.Scan(&scope); err != nil || scope != scopedTenantA {
							t.Fatalf("held Rows: scope=%q err=%v", scope, err)
						}
						n++
					}
					if err := rows.Err(); err != nil || n != 3 {
						t.Fatalf("rows=%d err=%v", n, err)
					}
				}
			}

			waitCtx, stop := context.WithTimeout(t.Context(), 5*time.Second)
			defer stop()
			done := make(chan error, 1)
			go func() {
				_, err := s.For(scopedTenantB).(ContextHandle).ExecContext(waitCtx, `SELECT 1`)
				done <- err
			}()
			select {
			case err := <-done:
				t.Fatalf("other tenant proceeded before %s: %v", kind, err)
			case <-time.After(25 * time.Millisecond):
			}
			if got := s.openLanes(); got != 1 {
				t.Fatalf("busy lane cap exceeded: %d", got)
			}
			finish()
			select {
			case err := <-done:
				if err != nil {
					t.Fatalf("waiter after %s: %v", kind, err)
				}
			case <-waitCtx.Done():
				t.Fatal("completion did not release capacity")
			}
		})
	}
}

func TestScopedCapacityWaitIsCancellable(t *testing.T) {
	s, _ := smallScoped(t)
	tx, err := s.For(scopedTenantA).Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	for _, tenant := range []string{scopedTenantA, scopedTenantB} {
		for _, operation := range []string{"exec", "query", "row", "begin"} {
			t.Run(tenant+"/"+operation, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(t.Context(), 25*time.Millisecond)
				defer cancel()
				h := s.For(tenant).(ContextHandle)
				var err error
				switch operation {
				case "exec":
					_, err = h.ExecContext(ctx, `SELECT 1`)
				case "query":
					_, err = h.QueryContext(ctx, `SELECT 1`)
				case "row":
					row := h.QueryRowContext(ctx, `SELECT 1`)
					if !errors.Is(row.Err(), context.DeadlineExceeded) {
						t.Fatalf("Row.Err = %v", row.Err())
					}
					err = row.Scan(new(int))
				case "begin":
					_, err = h.BeginTx(ctx, nil)
				}
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("waiting %s error = %v", operation, err)
				}
			})
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	if _, err := s.For(scopedTenantB).(ContextHandle).ExecContext(ctx, `SELECT 1`); err != nil {
		t.Fatalf("cancelled waiters leaked capacity: %v", err)
	}
}

func TestScopedFailuresReleaseCapacity(t *testing.T) {
	s, _ := smallScoped(t)
	h := s.For(scopedTenantA).(ContextHandle)
	for _, operation := range []string{"exec", "query", "row", "scan", "begin"} {
		t.Run(operation, func(t *testing.T) {
			var err error
			switch operation {
			case "exec":
				_, err = h.Exec(`SELECT 1/0`)
			case "query":
				_, err = h.Query(`SELECT 1/0`)
			case "row":
				err = h.QueryRow(`SELECT 1/0`).Scan(new(int))
			case "scan":
				err = h.QueryRow(`SELECT 'not an integer'`).Scan(new(int))
			case "begin":
				_, err = h.BeginTx(t.Context(), &sql.TxOptions{Isolation: sql.IsolationLevel(999)})
			}
			if err == nil {
				t.Fatal("expected operation failure")
			}
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			if _, err := s.For(scopedTenantB).(ContextHandle).ExecContext(ctx, `SELECT 1`); err != nil {
				t.Fatalf("failure leaked lease: %v", err)
			}
		})
	}
}

func TestScopedCloseWakesWaitersAndPreventsReopening(t *testing.T) {
	s, process := smallScoped(t)
	tx, err := s.For(scopedTenantA).Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	h := s.For(scopedTenantB).(ContextHandle)
	done := make(chan error, 1)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	go func() {
		_, err := h.ExecContext(ctx, `SELECT 1`)
		done <- err
	}()
	select {
	case err := <-done:
		t.Fatalf("waiter proceeded with all lanes busy: %v", err)
	case <-time.After(25 * time.Millisecond):
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, errScopedClosed) {
			t.Fatalf("waiter after Close: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("Close did not wake waiter")
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("existing transaction cannot finish: %v", err)
	}
	if err := h.QueryRowContext(ctx, `SELECT 1`).Scan(new(int)); !errors.Is(err, errScopedClosed) {
		t.Fatalf("retained handle reopened factory: %v", err)
	}
	if _, err := s.For(scopedTenantA).Begin(); !errors.Is(err, errScopedClosed) {
		t.Fatalf("new handle reopened factory: %v", err)
	}
	if err := s.Close(); err != nil || s.openLanes() != 0 {
		t.Fatalf("Close not idempotent: %v", err)
	}
	if err := process.PingContext(ctx); err != nil {
		t.Fatalf("Close shut down process pool: %v", err)
	}
}

func TestExplicitMaintenanceLeaseUsesTheSameHardBudget(t *testing.T) {
	s, _ := smallScoped(t)
	pool, release, err := LeasePool(t.Context(), s.Unscoped("fixture needs a concrete pool"))
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	var declared string
	if err := pool.QueryRow(`SELECT current_setting('hausv.cross_tenant')`).Scan(&declared); err != nil || declared != crossTenantOn {
		t.Fatalf("maintenance declaration=%q err=%v", declared, err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 25*time.Millisecond)
	defer cancel()
	if _, err := s.For(scopedTenantA).(ContextHandle).ExecContext(ctx, `SELECT 1`); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("idle explicit lease did not retain capacity: %v", err)
	}
	release()
	release() // teardown may safely repeat it
	if got := scopeOf(t, s.For(scopedTenantA)); got != scopedTenantA {
		t.Fatalf("scope after explicit lease release=%q", got)
	}
}

type delayedCloseConn struct {
	net.Conn
	once    sync.Once
	entered chan struct{}
	proceed chan struct{}
}

func (c *delayedCloseConn) Close() error {
	c.once.Do(func() {
		close(c.entered)
		<-c.proceed
	})
	return c.Conn.Close()
}

func TestCapacityWaitCanCancelWhileVictimIsStillClosing(t *testing.T) {
	s, _ := smallScoped(t)
	entered, proceed := make(chan struct{}), make(chan struct{})
	unblock := sync.OnceFunc(func() { close(proceed) })
	defer unblock()
	var dials atomic.Int64
	dial := s.template.DialFunc
	s.template.DialFunc = func(ctx context.Context, network, address string) (net.Conn, error) {
		conn, err := dial(ctx, network, address)
		if err == nil && dials.Add(1) == 1 {
			return &delayedCloseConn{Conn: conn, entered: entered, proceed: proceed}, nil
		}
		return conn, err
	}
	if _, err := s.For(scopedTenantA).Exec(`SELECT 1`); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := s.For(scopedTenantB).(ContextHandle).ExecContext(ctx, `SELECT 1`)
		done <- err
	}()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("eviction never reached physical Close")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancel during retirement: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("physical Close blocked context cancellation")
	}
	if dials.Load() != 1 || s.openLanes() != 1 {
		t.Fatal("replacement escaped the budget while victim was still closing")
	}
	unblock()
	if got := scopeOf(t, s.For(scopedTenantB)); got != scopedTenantB {
		t.Fatalf("replacement scope=%q", got)
	}
}

func TestCancelledQueryKeepsSocketBudgetUntilPGXCleanupFinishes(t *testing.T) {
	s, _ := smallScoped(t)
	entered, proceed := make(chan struct{}), make(chan struct{})
	unblock := sync.OnceFunc(func() { close(proceed) })
	defer unblock()
	var dials atomic.Int64
	dial := s.template.DialFunc
	s.template.DialFunc = func(ctx context.Context, network, address string) (net.Conn, error) {
		conn, err := dial(ctx, network, address)
		if err == nil && dials.Add(1) == 1 {
			return &delayedCloseConn{Conn: conn, entered: entered, proceed: proceed}, nil
		}
		return conn, err
	}
	h := s.For(scopedTenantA).(ContextHandle)
	if _, err := h.Exec(`SELECT 1`); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 25*time.Millisecond)
	defer cancel()
	if _, err := h.ExecContext(ctx, `SELECT pg_sleep(10)`); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("cancelled query: %v", err)
	}
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("pgx did not start asynchronous socket cleanup")
	}
	waitCtx, stop := context.WithTimeout(t.Context(), 25*time.Millisecond)
	defer stop()
	if _, err := s.For(scopedTenantB).(ContextHandle).ExecContext(waitCtx, `SELECT 1`); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("replacement escaped unfinished pgx cleanup: %v", err)
	}
	unblock()
	if got := scopeOf(t, s.For(scopedTenantB)); got != scopedTenantB {
		t.Fatalf("scope after pgx cleanup=%q", got)
	}
}

func TestLaneRollbackRestoresStartupScopeOnReusedConnection(t *testing.T) {
	s, _ := smallScoped(t)
	h := s.For(scopedTenantA)
	tx, err := h.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	var before, after int
	if err := tx.QueryRow(`SELECT pg_backend_pid()`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(`SELECT set_config('hausv.tenant_id',$1,true)`, scopedTenantB); err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	var scope string
	if err := h.QueryRow(`SELECT pg_backend_pid(), current_setting('hausv.tenant_id')`).Scan(&after, &scope); err != nil || before != after || scope != scopedTenantA {
		t.Fatalf("rollback reuse: pid=%d/%d scope=%q err=%v", before, after, scope, err)
	}
}

// Count sockets independently of the lane map, including evicted pools. A
// cache-only count would hide sockets still closing in a removed pool.
type countedLaneConn struct {
	net.Conn
	once sync.Once
	open *atomic.Int64
}

func (c *countedLaneConn) Close() error {
	err := c.Conn.Close()
	c.once.Do(func() { c.open.Add(-1) })
	return err
}

func TestScopedSixtyHousesStayWithinHardConnectionBudget(t *testing.T) {
	const houses, workers, writes = 60, 2, 4
	s, process := scopedPostgresConfig(t, func(cfg *Config) {
		cfg.LaneCap = 3
		cfg.LaneMaxConns = 2
		cfg.MaxOpenConns = 2
	})
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	var open, peak atomic.Int64
	dial := s.template.DialFunc
	s.template.DialFunc = func(ctx context.Context, network, address string) (net.Conn, error) {
		conn, err := dial(ctx, network, address)
		if err != nil {
			return nil, err
		}
		n := open.Add(1)
		for prev := peak.Load(); n > prev; prev = peak.Load() {
			if peak.CompareAndSwap(prev, n) {
				break
			}
		}
		return &countedLaneConn{Conn: conn, open: &open}, nil
	}
	for house := range houses {
		if _, err := process.ExecContext(ctx, `INSERT INTO tenant(tenant_id,slug,name) VALUES($1,$2,$2)`, laneID(house), fmt.Sprintf("house-%d", house)); err != nil {
			t.Fatal(err)
		}
	}
	// Hold the entire process allowance open, so the observed peak really
	// includes both sides of cap * per-lane + process.
	for range 2 {
		conn, err := process.Conn(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
	}
	var held []*sql.Tx
	for house := range 3 {
		for range 2 {
			tx, err := s.For(laneID(house)).(ContextHandle).BeginTx(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback()
			held = append(held, tx)
		}
	}
	if got := open.Load() + int64(process.Stats().OpenConnections); got != int64(s.Budget().Peak()) {
		t.Fatalf("stress did not saturate advertised budget: %d vs %d", got, s.Budget().Peak())
	}
	start := make(chan struct{})
	errs := make(chan error, houses*workers)
	var wg sync.WaitGroup
	for house := range houses {
		h := s.For(laneID(house)).(ContextHandle)
		for worker := range workers {
			wg.Go(func() {
				<-start
				for n := range writes {
					if err := stressScopedWrite(ctx, h, house, worker, n); err != nil {
						errs <- err
						return
					}
				}
			})
		}
	}
	close(start)
	waitCtx, stop := context.WithTimeout(ctx, 25*time.Millisecond)
	_, err := s.For(laneID(59)).(ContextHandle).ExecContext(waitCtx, `SELECT 1`)
	stop()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("all-busy load escaped cap: %v", err)
	}
	for _, tx := range held {
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	if t.Failed() {
		return
	}
	for house := range houses {
		var count, misplaced int
		err := s.For(laneID(house)).(ContextHandle).QueryRowContext(ctx, `
			SELECT count(*), count(*) FILTER (WHERE id NOT LIKE $1) FROM contacts
		`, fmt.Sprintf("h%d-%%", house)).Scan(&count, &misplaced)
		if err != nil || count != workers*writes || misplaced != 0 {
			t.Fatalf("house %d writes=%d misplaced=%d err=%v", house, count, misplaced, err)
		}
	}
	var total int
	if err := s.Unscoped("verify stress writes across all houses").(ContextHandle).QueryRowContext(ctx, `SELECT count(*) FROM contacts`).Scan(&total); err != nil || total != houses*workers*writes {
		t.Fatalf("maintenance total=%d err=%v", total, err)
	}
	observed := peak.Load() + int64(process.Stats().OpenConnections)
	if observed > int64(s.Budget().Peak()) || s.openLanes() > s.laneCap {
		t.Fatalf("peak sockets=%d budget=%d lanes=%d cap=%d", observed, s.Budget().Peak(), s.openLanes(), s.laneCap)
	}
	t.Logf("%d houses x %d workers x %d writes; peak sockets=%d, budget=%d", houses, workers, writes, observed, s.Budget().Peak())
}

func stressScopedWrite(ctx context.Context, h ContextHandle, house, worker, n int) error {
	tx, err := h.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("house %d Begin: %w", house, err)
	}
	defer tx.Rollback()
	var scope string
	if err := tx.QueryRowContext(ctx, `SELECT current_setting('hausv.tenant_id')`).Scan(&scope); err != nil {
		return err
	}
	if scope != laneID(house) {
		return fmt.Errorf("house %d borrowed tenant %q", house, scope)
	}
	// Derive tenant from the connection, not the requested house, so any
	// accidental scope switch is observable as a misplaced persisted write.
	if _, err := tx.ExecContext(ctx, `INSERT INTO contacts(tenant_id,id) VALUES(current_setting('hausv.tenant_id'),$1)`, fmt.Sprintf("h%d-w%d-n%d", house, worker, n)); err != nil {
		return err
	}
	return tx.Commit()
}
