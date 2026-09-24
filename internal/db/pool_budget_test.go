package db

import (
	"os"
	"strings"
	"testing"
	"time"
)

// An idle cap below the open cap means the pool closes connections it is about
// to need again. Measured on the current defaults (20 open / 5 idle): fifteen
// idle-closes of pure churn under a burst that the pool was sized for. The two
// caps have no reason to differ here — a connection this process is allowed to
// open is a connection it is allowed to keep.
func TestPostgresDefaultsKeepEveryOpenConnectionIdle(t *testing.T) {
	cfg := Config{}
	applyPostgresDefaults(&cfg)
	if cfg.MaxIdleConns != cfg.MaxOpenConns {
		t.Fatalf("default pool = %d open / %d idle; an idle cap below the open cap is measured churn",
			cfg.MaxOpenConns, cfg.MaxIdleConns)
	}
}

// An explicit idle cap is still honoured: dbtest and the migration tests size
// their pools deliberately and must not be widened behind their back.
func TestPostgresDefaultsHonourAnExplicitIdleCap(t *testing.T) {
	cfg := Config{MaxOpenConns: 4, MaxIdleConns: 1}
	applyPostgresDefaults(&cfg)
	if cfg.MaxIdleConns != 1 || cfg.MaxOpenConns != 4 {
		t.Fatalf("explicit pool sizing was overwritten: %d open / %d idle", cfg.MaxOpenConns, cfg.MaxIdleConns)
	}
}

// Measured hard ceiling: 94 lanes of one connection each against a server with
// max_connections=100 ends in FATAL 53300 "remaining connection slots are
// reserved for roles with the SUPERUSER attribute" — not at 100, because the
// reserved slots and this process's own unscoped pool are part of the budget.
// So the arithmetic that decides whether a lane plan fits has to count all
// three, and it is worth having as a pure function because that is the part a
// test can watch fail without a server.
func TestConnectionBudgetRefusesAPlanThatCannotFit(t *testing.T) {
	budget := ConnectionBudget{LaneCap: 94, PerLaneMax: 1, ProcessPool: 20}
	err := budget.fits(100, 3)
	if err == nil {
		t.Fatal("94 lanes plus a 20-connection process pool cannot fit in 100 slots, 3 of them reserved")
	}
	for _, want := range []string{"114", "97"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("budget error must name the numbers it compared, got: %v", err)
		}
	}
}

func TestConnectionBudgetAcceptsAPlanThatFits(t *testing.T) {
	budget := ConnectionBudget{LaneCap: 24, PerLaneMax: 3, ProcessPool: 20}
	if err := budget.fits(100, 3); err != nil {
		t.Fatalf("24x3 lanes plus a 20-connection process pool fit in 100 slots: %v", err)
	}
}

func TestScopedBudgetUsesActualProcessPoolLimit(t *testing.T) {
	scoped := offlineScoped(t, 2)
	scoped.process.SetMaxOpenConns(7)
	if got := scoped.Budget().Peak(); got != 2*3+7 {
		t.Fatalf("budget uses a stale config instead of the pool cap: %d", got)
	}
}

func TestScopedRejectsUnboundedProcessPool(t *testing.T) {
	scoped := offlineScoped(t, 2)
	scoped.process.SetMaxOpenConns(0)
	if _, err := NewScoped(scoped.cfg, scoped.process); err == nil {
		t.Fatal("an unbounded process pool cannot have a finite connection budget")
	}
	if err := scoped.Budget().fits(100, 3); err == nil {
		t.Fatal("an existing factory must reject a process pool changed to unlimited")
	}
}

// The same assertion against a real server, because the pure arithmetic cannot
// tell whether max_connections is being read correctly.
func TestVerifyConnectionBudgetAgainstLivePostgres(t *testing.T) {
	baseDSN := strings.TrimSpace(os.Getenv("HAUSV_TEST_POSTGRES_DSN"))
	if baseDSN == "" {
		if os.Getenv("HAUSV_TEST_POSTGRES_REQUIRED") == "true" {
			t.Fatal("HAUSV_TEST_POSTGRES_DSN is required when HAUSV_TEST_POSTGRES_REQUIRED=true")
		}
		t.Skip("set HAUSV_TEST_POSTGRES_DSN to a disposable database owned by a NOSUPERUSER NOBYPASSRLS role")
	}
	database, err := OpenConfig(t.Context(), Config{
		Backend:         BackendPostgres,
		DSN:             isolatedPostgresSchema(t, baseDSN),
		ConnectTimeout:  3 * time.Second,
		MaxOpenConns:    1,
		ConnMaxLifetime: time.Minute,
	})
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if err := VerifyConnectionBudget(t.Context(), database, ConnectionBudget{LaneCap: 1, PerLaneMax: 1, ProcessPool: 1}); err != nil {
		t.Fatalf("a three-connection plan must fit any usable server: %v", err)
	}
	if err := VerifyConnectionBudget(t.Context(), database, ConnectionBudget{LaneCap: 1_000_000, PerLaneMax: 1, ProcessPool: 1}); err == nil {
		t.Fatal("a million lanes must be refused by the live check")
	}
}
