package store

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
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

// deliveryBlockedInSend starts an Attempt whose send blocks until release is
// closed and returns once the send has begun.
func deliveryBlockedInSend(t *testing.T, repo AnnualStatementDeliveryRepository, item AnnualStatementDelivery, calls *atomic.Int32) (release chan struct{}, result chan deliveryOutcome) {
	t.Helper()
	started := make(chan struct{})
	release, result = make(chan struct{}), make(chan deliveryOutcome, 1)
	go func() {
		row, skip, err := repo.Attempt(t.Context(), item, func(context.Context) error {
			calls.Add(1)
			close(started)
			<-release
			return nil
		})
		result <- deliveryOutcome{row, skip, err}
	}()
	select {
	case <-started:
	case outcome := <-result:
		t.Fatalf("delivery never reached send: %v", outcome.err)
	case <-time.After(5 * time.Second):
		t.Fatal("delivery did not start")
	}
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
	})
	return release, result
}

type deliveryOutcome struct {
	row  AnnualStatementDelivery
	skip bool
	err  error
}

// HAUSV-642: the reservation commits before the mail exchange, so a slow mail
// server never holds SQLite's process-wide write lock (busy_timeout 5 s).
// While one party is in flight, another delivery of the same run commits at
// once and the run page sees the reservation as pending.
func TestAnnualStatementDeliveryReleasesTheWriteLockDuringSend(t *testing.T) {
	_, lanes := testLanes(t)
	repo, _ := BindAnnualStatementDeliveryRepository(NewSQLAnnualStatementDeliveryStore(lanes), testTenantRef("demo"))
	item := AnnualStatementDelivery{RunID: "run", Revision: 1, PartyID: "a@example.com", UnitID: "top-1", DocumentID: "archive", Actor: "manager@example.com"}
	var calls atomic.Int32
	release, result := deliveryBlockedInSend(t, repo, item, &calls)
	other := item
	other.UnitID = "top-2"
	begin := time.Now()
	row, skip, err := repo.Attempt(t.Context(), other, func(context.Context) error { calls.Add(1); return nil })
	if err != nil || skip || row.Status != "sent" || time.Since(begin) > 4*time.Second {
		t.Fatal("another writer waited on the in-flight send", row, skip, err, time.Since(begin))
	}
	rows, err := repo.List("run")
	if err != nil || len(rows) != 2 {
		t.Fatal(rows, err)
	}
	for _, row := range rows {
		if row.UnitID == "top-1" && row.Status != "pending" {
			t.Fatal("in-flight delivery must be visible as pending", row)
		}
	}
	close(release)
	if outcome := <-result; outcome.err != nil || outcome.skip || outcome.row.Status != "sent" || outcome.row.Attempt != 1 {
		t.Fatal(outcome)
	}
	rows, _ = repo.List("run")
	for _, row := range rows {
		if row.Status != "sent" {
			t.Fatal("outcome not recorded", row)
		}
	}
	if calls.Load() != 2 {
		t.Fatal(calls.Load())
	}
}

// A second submit for a party whose reservation is live fails closed with the
// typed conflict, on both engines and without waiting on any lock.
func TestAnnualStatementDeliveryInFlightReservationRefusesSecondSubmit(t *testing.T) {
	_, lanes := testLanes(t)
	first, _ := BindAnnualStatementDeliveryRepository(NewSQLAnnualStatementDeliveryStore(lanes), testTenantRef("demo"))
	second, _ := BindAnnualStatementDeliveryRepository(NewSQLAnnualStatementDeliveryStore(lanes), testTenantRef("demo"))
	item := AnnualStatementDelivery{RunID: "run", Revision: 1, PartyID: "a@example.com", UnitID: "top-1", DocumentID: "archive", Actor: "manager@example.com"}
	var calls atomic.Int32
	release, result := deliveryBlockedInSend(t, first, item, &calls)
	begin := time.Now()
	_, skip, err := second.Attempt(t.Context(), item, func(context.Context) error { calls.Add(1); return nil })
	if !errors.Is(err, ErrDeliveryInProgress) || skip || calls.Load() != 1 || time.Since(begin) > 4*time.Second {
		t.Fatal("loser must return the typed conflict without sending or waiting", skip, calls.Load(), err, time.Since(begin))
	}
	close(release)
	if outcome := <-result; outcome.err != nil {
		t.Fatal("winner failed", outcome.err)
	}
	if rows, err := first.List("run"); err != nil || len(rows) != 1 || rows[0].Status != "sent" || calls.Load() != 1 {
		t.Fatal("double submit must send and persist once", rows, calls.Load(), err)
	}
}

// A reservation older than the TTL was left by a process that died mid-send:
// the next attempt records it as interrupted and delivers. Should the old
// process still report late, the stored verdict wins.
func TestAnnualStatementDeliveryTakesOverAnAbandonedReservation(t *testing.T) {
	database, lanes := testLanes(t)
	repo, _ := BindAnnualStatementDeliveryRepository(NewSQLAnnualStatementDeliveryStore(lanes), testTenantRef("demo"))
	item := AnnualStatementDelivery{RunID: "run", Revision: 1, PartyID: "a@example.com", UnitID: "top-1", DocumentID: "archive", Actor: "manager@example.com"}
	var calls atomic.Int32
	release, result := deliveryBlockedInSend(t, repo, item, &calls)
	aged := time.Now().Add(-AnnualStatementDeliveryPendingTTL - time.Minute).UTC().Format(time.RFC3339Nano)
	if _, err := database.Exec(`UPDATE annual_statement_deliveries SET sent_at=$1 WHERE tenant_id=$2 AND status='pending'`, aged, testTenantID("demo")); err != nil {
		t.Fatal(err)
	}
	row, skip, err := repo.Attempt(t.Context(), item, func(context.Context) error { calls.Add(1); return nil })
	if err != nil || skip || row.Status != "sent" || row.Attempt != 2 || calls.Load() != 2 {
		t.Fatal("abandoned reservation not taken over", row, skip, err, calls.Load())
	}
	close(release)
	outcome := <-result
	if outcome.err != nil || outcome.skip || outcome.row.Status != "failed" || outcome.row.Error != AnnualStatementDeliveryInterrupted || outcome.row.Attempt != 1 {
		t.Fatal("late result must keep the stored verdict", outcome)
	}
	rows, err := repo.List("run")
	if err != nil || len(rows) != 2 || rows[0].Status != "failed" || rows[0].Error != AnnualStatementDeliveryInterrupted || rows[1].Status != "sent" {
		t.Fatal(rows, err)
	}
	if _, skip, err := repo.Attempt(t.Context(), item, func(context.Context) error { calls.Add(1); return nil }); err != nil || !skip || calls.Load() != 2 {
		t.Fatal("delivered party must be skipped", skip, calls.Load(), err)
	}
}
