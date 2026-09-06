package store

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/dbtest"
)

func TestAnnualStatementDeliveryStorage(t *testing.T) {
	database, lanes := testLanes(t)
	storage := NewSQLAnnualStatementDeliveryStore(lanes)
	repo, ok := BindAnnualStatementDeliveryRepository(storage, testTenantRef("demo"))
	if !ok {
		t.Fatal("bind")
	}
	if _, ok := BindAnnualStatementDeliveryRepository(storage, TenantRef{}); ok {
		t.Fatal("invalid tenant")
	}
	item := AnnualStatementDelivery{RunID: "run", Revision: 1, PartyID: "a@example.com", UnitID: "top-1", DocumentID: "archive", SHA256: "abc", Recipient: "a@example.com", Actor: "manager@example.com"}
	calls := 0
	send := func(context.Context) error { calls++; return nil }
	first, skip, err := repo.Attempt(t.Context(), item, func(context.Context) error { calls++; return errors.New("Testfehler") })
	if err != nil || skip || first.Status != "failed" || first.Attempt != 1 || first.Error != "Testfehler" {
		t.Fatal(first, skip, err)
	}
	second, skip, err := repo.Attempt(t.Context(), item, send)
	if err != nil || skip || second.Status != "sent" || second.Attempt != 2 || second.Error != "" {
		t.Fatal(second, skip, err)
	}
	if _, skip, err := repo.Attempt(t.Context(), item, send); err != nil || !skip || calls != 2 {
		t.Fatal("duplicate send", skip, calls, err)
	}
	rows, err := repo.List("run")
	if err != nil || len(rows) != 2 || rows[0].ID == rows[1].ID || rows[0].Attempt != 1 || rows[1].Attempt != 2 {
		t.Fatal(rows, err)
	}
	// RFC3339Nano omits zero fractions, so timestamp text is not chronological.
	for i, at := range []string{"2026-09-06T12:00:00Z", "2026-09-06T12:00:00.1Z"} {
		if _, err := database.Exec(`UPDATE annual_statement_deliveries SET sent_at=$1 WHERE tenant_id=$2 AND id=$3`, at, testTenantID("demo"), rows[i].ID); err != nil {
			t.Fatal(err)
		}
	}
	if ordered, err := repo.List("run"); err != nil || len(ordered) != 2 || ordered[0].ID != first.ID || ordered[1].ID != second.ID {
		t.Fatal("attempt order depends on timestamp precision", ordered, err)
	}
	if rows, err := repo.List("foreign-run"); err != nil || len(rows) != 0 {
		t.Fatal("run isolation", err)
	}
	other, _ := BindAnnualStatementDeliveryRepository(storage, testTenantRef("other"))
	if rows, err := other.List("run"); err != nil || len(rows) != 0 {
		t.Fatal("tenant isolation", err)
	}
	if _, skip, err := other.Attempt(t.Context(), item, send); err != nil || skip {
		t.Fatal("foreign tenant blocked", err)
	}
	item.UnitID = "top-2"
	if _, skip, err := repo.Attempt(t.Context(), item, send); err != nil || skip {
		t.Fatal("same party other unit skipped", err)
	}
	item.Revision = 2
	if _, skip, err := repo.Attempt(t.Context(), item, send); err != nil || skip {
		t.Fatal("new revision skipped", err)
	}
	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM annual_statement_deliveries`).Scan(&count); err != nil || count != 5 {
		t.Fatal(count, err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, _, err := repo.Attempt(ctx, item, send); err == nil {
		t.Fatal("canceled accepted")
	}
	item.Revision = 3
	ctx, cancel = context.WithCancel(t.Context())
	result, _, err := repo.Attempt(ctx, item, func(context.Context) error { cancel(); return errors.New("Versand abgebrochen.") })
	if err != nil || result.Status != "failed" {
		t.Fatal("cancellation not recorded", result, err)
	}
}

func TestAnnualStatementDeliveryConcurrentStores(t *testing.T) {
	_, lanes := testLanes(t)
	first, _ := BindAnnualStatementDeliveryRepository(NewSQLAnnualStatementDeliveryStore(lanes), testTenantRef("demo"))
	second, _ := BindAnnualStatementDeliveryRepository(NewSQLAnnualStatementDeliveryStore(lanes), testTenantRef("demo"))
	item := AnnualStatementDelivery{RunID: "run", Revision: 1, PartyID: "a@example.com", UnitID: "top-1", DocumentID: "archive", SHA256: "abc", Recipient: "a@example.com", Actor: "manager@example.com"}
	var calls atomic.Int32
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, repo := range []AnnualStatementDeliveryRepository{first, second} {
		go func() {
			<-start
			_, _, err := repo.Attempt(t.Context(), item, func(context.Context) error { calls.Add(1); return nil })
			results <- err
		}()
	}
	close(start)
	// Only the typed reservation conflict is an expected error.
	for range 2 {
		if err := <-results; err != nil && !errors.Is(err, ErrDeliveryInProgress) {
			t.Errorf("unexpected concurrent delivery error: %v", err)
		}
	}
	rows, err := first.List("run")
	if err != nil || len(rows) != 1 || rows[0].Status != "sent" || calls.Load() != 1 {
		t.Fatal(rows, calls.Load(), err)
	}
}

func TestAnnualStatementDeliveryUniqueViolation(t *testing.T) {
	database, lanes := testLanes(t)
	repo, _ := BindAnnualStatementDeliveryRepository(NewSQLAnnualStatementDeliveryStore(lanes), testTenantRef("demo"))
	item := AnnualStatementDelivery{RunID: "run", Revision: 1, PartyID: "a@example.com", UnitID: "top-1", DocumentID: "archive", Actor: "manager@example.com"}
	if _, _, err := repo.Attempt(t.Context(), item, func(context.Context) error { return errors.New("Testfehler") }); err != nil {
		t.Fatal(err)
	}
	// Force an INSERT collision on both real engines without relying on timing.
	// SQLite normally serializes writers before they can select the same attempt.
	if _, err := database.Exec(`CREATE UNIQUE INDEX delivery_test_collision ON annual_statement_deliveries(tenant_id,run_id,revision,unit_id,party_id)`); err != nil {
		t.Fatal(err)
	}
	calls := 0
	_, skip, err := repo.Attempt(t.Context(), item, func(context.Context) error { calls++; return nil })
	if !errors.Is(err, ErrDeliveryInProgress) || skip || calls != 0 {
		t.Fatal("unique violation must return a typed conflict without sending", skip, calls, err)
	}
	if rows, err := repo.List("run"); err != nil || len(rows) != 1 || rows[0].Status != "failed" {
		t.Fatal("conflicting reservation was not rolled back", rows, err)
	}
}

func TestAnnualStatementDeliveryPostgresBlockedDoubleSubmit(t *testing.T) {
	if dbtest.Backend() != "postgres" {
		t.Skip("PostgreSQL waits on the uncommitted unique key; SQLite serializes writers")
	}
	database, lanes := testLanes(t)
	first, _ := BindAnnualStatementDeliveryRepository(NewSQLAnnualStatementDeliveryStore(lanes), testTenantRef("demo"))
	second, _ := BindAnnualStatementDeliveryRepository(NewSQLAnnualStatementDeliveryStore(lanes), testTenantRef("demo"))
	item := AnnualStatementDelivery{RunID: "run", Revision: 1, PartyID: "a@example.com", UnitID: "top-1", DocumentID: "archive", Actor: "manager@example.com"}
	started, release := make(chan struct{}), make(chan struct{})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	firstResult, secondResult := make(chan error, 1), make(chan error, 1)
	var calls atomic.Int32
	go func() {
		_, _, err := first.Attempt(t.Context(), item, func(context.Context) error {
			calls.Add(1)
			close(started)
			<-release
			return nil
		})
		firstResult <- err
	}()
	select {
	case <-started:
	case err := <-firstResult:
		t.Fatalf("first delivery never reached send: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("first delivery did not start")
	}
	go func() {
		_, _, err := second.Attempt(t.Context(), item, func(context.Context) error { calls.Add(1); return nil })
		secondResult <- err
	}()
	// Wait for actual engine contention on this test's table before committing
	// the winner; a sleep alone could let the second request start too late.
	deadline := time.Now().Add(5 * time.Second)
	for {
		var blocked bool
		err := database.QueryRow(`SELECT EXISTS (SELECT 1 FROM pg_locks WHERE relation='annual_statement_deliveries'::regclass AND cardinality(pg_blocking_pids(pid)) > 0)`).Scan(&blocked)
		if err != nil {
			t.Fatal(err)
		}
		if blocked {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("second delivery did not block on the reservation")
		}
		time.Sleep(10 * time.Millisecond)
	}
	close(release)
	if err := <-firstResult; err != nil {
		t.Fatal("winner failed", err)
	}
	if err := <-secondResult; !errors.Is(err, ErrDeliveryInProgress) {
		t.Fatal("loser must return the typed conflict", err)
	}
	if rows, err := first.List("run"); err != nil || len(rows) != 1 || rows[0].Status != "sent" || calls.Load() != 1 {
		t.Fatal("double submit must send and persist once", rows, calls.Load(), err)
	}
}
