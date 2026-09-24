package store

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	appdb "github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/dbtest"
)

func TestImportLedgerRollbackAndRetry(t *testing.T) {
	for _, failure := range []string{"callback", "cancel", "late-invalid-status"} {
		t.Run(failure, func(t *testing.T) {
			database, lanes := testLanes(t)
			ledger := NewImportLedger(lanes)
			tenant := testTenantRef("demo")
			key := ImportKey{Format: "camt.053", AppliedBy: "manager@example.com"}
			digest := strings.Repeat("a", 64)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			late := errors.New("failure after effects")
			_, err := ledger.Apply(ctx, tenant, key, digest, func(tx *ImportTx) error {
				for _, unit := range []string{"top-1", "top-2"} {
					if _, changed, err := tx.SetPaymentStatus(UnitPaymentStatus{UnitID: unit, Status: UnitPaymentStatusPaid, UpdatedBy: key.AppliedBy}); err != nil || !changed {
						return fmt.Errorf("set status: changed=%v: %w", changed, err)
					}
				}
				switch failure {
				case "cancel":
					cancel()
					return nil
				case "late-invalid-status":
					_, _, err := tx.SetPaymentStatus(UnitPaymentStatus{UnitID: "top-3", Status: "invalid", UpdatedBy: key.AppliedBy})
					return err
				default:
					return late
				}
			})
			if err == nil {
				t.Fatal("late failure accepted")
			}
			for _, table := range []string{"integration_imports", "unit_payment_status"} {
				var n int
				if err := database.QueryRow(`SELECT count(*) FROM ` + table).Scan(&n); err != nil || n != 0 {
					t.Fatalf("rollback %s: count=%d, err=%v", table, n, err)
				}
			}
			var calls int
			for attempt := range 2 {
				already, err := ledger.Apply(t.Context(), tenant, key, digest, func(tx *ImportTx) error {
					calls++
					_, _, err := tx.SetPaymentStatus(UnitPaymentStatus{UnitID: "top-1", Status: UnitPaymentStatusPaid, UpdatedBy: key.AppliedBy})
					tx.Counts = ImportCounts{Assigned: 1, Changed: 1}
					return err
				})
				if err != nil || already != (attempt == 1) {
					t.Fatalf("retry %d: already=%v, err=%v", attempt, already, err)
				}
			}
			if calls != 1 {
				t.Fatalf("effect calls=%d", calls)
			}
			var changed int
			if err := database.QueryRow(`SELECT changed FROM integration_imports`).Scan(&changed); err != nil || changed != 1 {
				t.Fatalf("committed counts: changed=%d, err=%v", changed, err)
			}
		})
	}
}

func TestImportLedgerConcurrentConnections(t *testing.T) {
	if dbtest.Backend() != appdb.BackendPostgres {
		t.Skip("independent PostgreSQL tenant pools required")
	}
	database, cfg := dbtest.OpenWithConfig(t)
	seedFixtureTenants(t, database)
	first, second := NewImportLedger(lanesOver(t, database, cfg)), NewImportLedger(lanesOver(t, database, cfg))
	tenant := testTenantRef("demo")
	key := ImportKey{Format: "camt.053", AppliedBy: "manager@example.com"}
	digest := strings.Repeat("b", 64)
	entered, release, started := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	type outcome struct {
		already bool
		err     error
	}
	results := make(chan outcome, 2)
	go func() {
		already, err := first.Apply(t.Context(), tenant, key, digest, func(tx *ImportTx) error {
			calls.Add(1)
			close(entered)
			<-release
			_, _, err := tx.SetPaymentStatus(UnitPaymentStatus{UnitID: "top-1", Status: UnitPaymentStatusPaid, UpdatedBy: key.AppliedBy})
			return err
		})
		results <- outcome{already, err}
	}()
	select {
	case <-entered:
	case result := <-results:
		t.Fatalf("first import failed before effect: %v", result.err)
	case <-time.After(10 * time.Second):
		t.Fatal("first import did not reach effect")
	}
	go func() {
		close(started)
		already, err := second.Apply(t.Context(), tenant, key, digest, func(tx *ImportTx) error {
			calls.Add(1)
			return nil
		})
		results <- outcome{already, err}
	}()
	<-started
	// While the first transaction holds its reservation, the second may
	// neither execute its callback nor report success from a dirty read.
	select {
	case result := <-results:
		close(release)
		t.Fatalf("second import escaped uncommitted reservation: %+v", result)
	case <-time.After(100 * time.Millisecond):
	}
	close(release)
	alreadyCount := 0
	for range 2 {
		select {
		case result := <-results:
			if result.err != nil {
				t.Fatal(result.err)
			}
			if result.already {
				alreadyCount++
			}
		case <-time.After(10 * time.Second):
			t.Fatal("concurrent import did not complete")
		}
	}
	if calls.Load() != 1 || alreadyCount != 1 {
		t.Fatalf("effects=%d, already=%d", calls.Load(), alreadyCount)
	}
	// The same bytes remain importable in another house, through its own RLS lane.
	already, err := second.Apply(t.Context(), testTenantRef("other"), key, digest, func(tx *ImportTx) error {
		_, _, err := tx.SetPaymentStatus(UnitPaymentStatus{UnitID: "top-1", Status: UnitPaymentStatusPartial, UpdatedBy: key.AppliedBy})
		return err
	})
	if err != nil || already {
		t.Fatalf("other house: already=%v err=%v", already, err)
	}
	for slug, want := range map[string]string{"demo": UnitPaymentStatusPaid, "other": UnitPaymentStatusPartial} {
		repo, _ := BindUnitPaymentStatusRepository(NewSQLUnitPaymentStatusStore(second.db), testTenantRef(slug))
		got, ok := repo.Get("top-1")
		if !ok || got.Status != want {
			t.Fatalf("%s: %+v, %v", slug, got, ok)
		}
	}
}

func importDocumentFixture() (ImportKey, DocumentRecord, []byte, string) {
	data := []byte("<Invoice>synthetic recovery fixture</Invoice>")
	return ImportKey{Format: "ebinterface", AppliedBy: "first@example.com"}, DocumentRecord{
		Title: "E-Rechnung 42", Category: DocumentCategoryBilling, Visibility: DocumentVisibilityManagerOnly, UploadedBy: "first@example.com",
	}, data, fmt.Sprintf("%x", sha256.Sum256(data))
}

func TestImportDocumentPendingRecovery(t *testing.T) {
	for _, boundary := range []string{"before-blob", "after-blob", "torn-blob", "completion-rollback"} {
		t.Run(boundary, func(t *testing.T) {
			database, lanes := testLanes(t)
			ledger := NewImportLedger(lanes)
			tenant := testTenantRef("demo")
			key, item, data, digest := importDocumentFixture()
			documents := NewSQLDocumentStore(lanes, t.TempDir())
			reserved, complete, err := ledger.reserveDocument(t.Context(), tenant, key, digest, item, "first.xml", len(data))
			if err != nil || complete {
				t.Fatalf("reserve: %v %v", complete, err)
			}
			if complete, err := ledger.Completed(t.Context(), tenant, key, digest); err != nil || complete {
				t.Fatalf("pending counts as complete: %v %v", complete, err)
			}
			if boundary != "before-blob" {
				if err := writeImportBlob(documents.fileDir, reserved, data); err != nil {
					t.Fatal(err)
				}
			}
			path := filepath.Join(documents.fileDir, tenant.Slug, reserved.StoredFilename)
			if boundary == "torn-blob" {
				if err := os.WriteFile(path, []byte("partial"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if boundary == "completion-rollback" {
				// Refuse metadata insertion AFTER the pending row was updated to
				// complete, proving both changes roll back together.
				trigger := `CREATE TRIGGER reject_invoice BEFORE INSERT ON documents BEGIN SELECT RAISE(ABORT, 'injected'); END`
				if dbtest.Backend() == appdb.BackendPostgres {
					trigger = `CREATE FUNCTION reject_invoice_fn() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected'; END $$;
					CREATE TRIGGER reject_invoice BEFORE INSERT ON documents FOR EACH ROW EXECUTE FUNCTION reject_invoice_fn()`
				}
				if _, err := database.Exec(trigger); err != nil {
					t.Fatal(err)
				}
				if _, _, err := ledger.completeDocument(t.Context(), tenant, key, digest, reserved); err == nil {
					t.Fatal("injected completion succeeded")
				}
				if complete, err := ledger.Completed(t.Context(), tenant, key, digest); err != nil || complete {
					t.Fatalf("failed completion persisted: %v %v", complete, err)
				}
				drop := `DROP TRIGGER reject_invoice`
				if dbtest.Backend() == appdb.BackendPostgres {
					drop += ` ON documents`
				}
				if _, err := database.Exec(drop); err != nil {
					t.Fatal(err)
				}
			}
			if got := documents.listTenant(tenant); len(got) != 0 {
				t.Fatalf("pending document visible: %+v", got)
			}
			// Simulate a new service instance and a different confirming actor.
			ledger = NewImportLedger(lanes)
			key.AppliedBy, item.UploadedBy, item.Title = "retry@example.com", "retry@example.com", "changed title"
			for attempt := range 2 {
				created, already, err := ledger.ImportDocument(t.Context(), tenant, key, digest, documents, item, "retry.xml", data)
				if err != nil || already != (attempt == 1) {
					t.Fatalf("attempt %d: %v %v", attempt, already, err)
				}
				if created.ID != reserved.ID || created.UploadedBy != reserved.UploadedBy || !created.UploadedAt.Equal(reserved.UploadedAt) || created.Title != reserved.Title || created.Filename != "first.xml" {
					t.Fatalf("reservation changed: got=%+v want=%+v", created, reserved)
				}
			}
			if got := documents.listTenant(tenant); len(got) != 1 {
				t.Fatalf("documents=%d", len(got))
			}
			stored, err := os.ReadFile(path)
			if err != nil || string(stored) != string(data) {
				t.Fatalf("blob recovery: %v", err)
			}
			var actor, status string
			if err := database.QueryRow(`SELECT applied_by, status FROM integration_imports`).Scan(&actor, &status); err != nil || actor != "first@example.com" || status != "complete" {
				t.Fatalf("ledger changed: %s %s %v", actor, status, err)
			}
		})
	}
}

func TestImportDocumentConcurrentConnections(t *testing.T) {
	if dbtest.Backend() != appdb.BackendPostgres {
		t.Skip("independent PostgreSQL tenant pools required")
	}
	database, cfg := dbtest.OpenWithConfig(t)
	seedFixtureTenants(t, database)
	key, item, data, digest := importDocumentFixture()
	dir := t.TempDir()
	start := make(chan struct{})
	type outcome struct {
		item    DocumentRecord
		already bool
		err     error
	}
	results := make(chan outcome, 2)
	for range 2 {
		lanes := lanesOver(t, database, cfg)
		go func() {
			<-start
			created, already, err := NewImportLedger(lanes).ImportDocument(t.Context(), testTenantRef("demo"), key, digest, NewSQLDocumentStore(lanes, dir), item, "invoice.xml", data)
			results <- outcome{created, already, err}
		}()
	}
	close(start)
	first, second := <-results, <-results
	if first.err != nil || second.err != nil || first.already == second.already || first.item.ID != second.item.ID {
		t.Fatalf("concurrent completion: %+v %+v", first, second)
	}
	lanes := lanesOver(t, database, cfg)
	if got := NewSQLDocumentStore(lanes, dir).listTenant(testTenantRef("demo")); len(got) != 1 {
		t.Fatalf("documents=%d", len(got))
	}
}
