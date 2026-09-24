package store

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// ErrDeliveryInProgress means another request reserved the same delivery.

// ValorisationDeliveryPendingTTL bounds how long a committed reservation
// (status pending) keeps other requests away from the same party. It exceeds
// the mailer's connect and transaction timeouts plus the archive read, so a
// live send is never taken over; a reservation older than this was left by a
// process that died mid-send and is finalized as interrupted by the next
// attempt.
const ValorisationDeliveryPendingTTL = 2 * time.Minute

// ValorisationDeliveryInterrupted is the error text of a reservation whose
// outcome was never recorded.
const ValorisationDeliveryInterrupted = "Versand unterbrochen"

type ValorisationDelivery struct {
	ID, RunID                                      string
	Revision                                       int
	PartyID, UnitID, DocumentID, SHA256, Recipient string
	SentAt                                         time.Time
	Status, Error, Actor                           string
	Attempt                                        int
}

type ValorisationDeliveryRepository interface {
	List(runID string) ([]ValorisationDelivery, error)
	// Attempt serializes delivery of a party/unit and skips any previous success.
	// send must return an error safe to display to the Verwaltung.
	Attempt(ctx context.Context, item ValorisationDelivery, send func(context.Context) error) (ValorisationDelivery, bool, error)
}

type SQLValorisationDeliveryStore struct{ db *TenantDB }

func NewSQLValorisationDeliveryStore(db *TenantDB) *SQLValorisationDeliveryStore {
	return &SQLValorisationDeliveryStore{db}
}

type boundValorisationDeliveryRepository struct {
	storage *SQLValorisationDeliveryStore
	tenant  TenantRef
}

func BindValorisationDeliveryRepository(storage *SQLValorisationDeliveryStore, tenant TenantRef) (ValorisationDeliveryRepository, bool) {
	resolved, valid := validTenantRef(tenant)
	if storage == nil || !valid {
		return nil, false
	}
	return &boundValorisationDeliveryRepository{storage, resolved}, true
}

func (r *boundValorisationDeliveryRepository) List(runID string) ([]ValorisationDelivery, error) {
	rows, err := r.storage.db.For(r.tenant).Query(`SELECT id,run_id,revision,party_id,unit_id,document_id,sha256,recipient,sent_at,status,error,actor,attempt FROM valorisation_deliveries WHERE tenant_id=$1 AND run_id=$2 ORDER BY attempt,id`, r.tenant.ID, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ValorisationDelivery{}
	for rows.Next() {
		var item ValorisationDelivery
		var at string
		if err := rows.Scan(&item.ID, &item.RunID, &item.Revision, &item.PartyID, &item.UnitID, &item.DocumentID, &item.SHA256, &item.Recipient, &at, &item.Status, &item.Error, &item.Actor, &item.Attempt); err != nil {
			return nil, err
		}
		item.SentAt, err = time.Parse(time.RFC3339Nano, at)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *boundValorisationDeliveryRepository) Attempt(ctx context.Context, item ValorisationDelivery, send func(context.Context) error) (ValorisationDelivery, bool, error) {
	if err := ctx.Err(); err != nil {
		return item, false, err
	}
	if item.RunID == "" || item.Revision < 1 || item.UnitID == "" || item.DocumentID == "" || item.Actor == "" || send == nil {
		return item, false, fmt.Errorf("invalid valorisation delivery")
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return item, false, err
	}
	item.ID = fmt.Sprintf("%x", id)
	handle, ok := r.storage.db.For(r.tenant).(txBeginner)
	if !ok {
		return item, false, fmt.Errorf("delivery transactions unavailable")
	}
	// Two short transactions bracket the mail exchange instead of one spanning
	// it: on SQLite a write transaction is the process-wide write lock, and a
	// slow mail server must not stall every other writer of every tenant
	// (HAUSV-642). A canceled request keeps a bounded opportunity to record the
	// outcome. There is no atomic commit spanning SMTP and SQL.
	recordCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	item, skip, err := r.reserve(recordCtx, handle, item)
	if err != nil || skip {
		return item, skip, err
	}
	item.Status, item.Error = "sent", ""
	if err := ctx.Err(); err != nil {
		item.Status, item.Error = "failed", "Versand abgebrochen."
		recorded, recordErr := r.record(recordCtx, handle, item)
		if recordErr != nil {
			return recorded, false, recordErr
		}
		return recorded, false, err
	}
	if err := send(ctx); err != nil {
		item.Status, item.Error = "failed", err.Error()
	}
	item, err = r.record(recordCtx, handle, item)
	return item, false, err
}

// reserve commits one attempt row in status pending. It reports skip when the
// party already has a successful delivery, fails closed with
// ErrDeliveryInProgress while another reservation of the party is live, and
// finalizes reservations abandoned by a dead process as interrupted.
func (r *boundValorisationDeliveryRepository) reserve(ctx context.Context, handle txBeginner, item ValorisationDelivery) (ValorisationDelivery, bool, error) {
	tx, err := handle.BeginTx(ctx, nil)
	if err != nil {
		return item, false, err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `UPDATE valorisation_runs SET status=status WHERE tenant_id=$1 AND id=$2 AND status='approved'`, r.tenant.ID, item.RunID)
	if err != nil {
		return item, false, err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return item, false, ErrValorisationConflict
	}
	item.SentAt = time.Now().UTC()
	item.Status, item.Error = "pending", ""
	// The INSERT comes first so the transaction is a writer from its first
	// statement: SQLite in WAL mode refuses a read transaction that upgrades to
	// a write after another writer committed. The unique attempt key serializes
	// stores that reserve concurrently; the loser fails closed and can retry.
	_, err = tx.ExecContext(ctx, `INSERT INTO valorisation_deliveries(tenant_id,tenant_slug,id,run_id,revision,party_id,unit_id,document_id,sha256,recipient,sent_at,status,error,actor,attempt)
 SELECT CAST($1 AS VARCHAR(26)),$2,$3,$4,CAST($5 AS BIGINT),$6,$7,$8,$9,$10,$11,'pending','',$12,COALESCE(MAX(attempt),0)+1 FROM valorisation_deliveries WHERE tenant_id=$1 AND run_id=$4 AND revision=$5 AND party_id=$6 AND unit_id=$7`, r.tenant.ID, r.tenant.Slug, item.ID, item.RunID, item.Revision, item.PartyID, item.UnitID, item.DocumentID, item.SHA256, item.Recipient, item.SentAt.Format(time.RFC3339Nano), item.Actor)
	if err != nil {
		var pgErr *pgconn.PgError
		var sqliteErr *sqlite.Error
		if (errors.As(err, &pgErr) && pgErr.Code == "23505") ||
			(errors.As(err, &sqliteErr) && (sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE || sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY)) {
			return item, false, fmt.Errorf("%w: %w", ErrDeliveryInProgress, err)
		}
		return item, false, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT id,status,sent_at FROM valorisation_deliveries WHERE tenant_id=$1 AND run_id=$2 AND revision=$3 AND party_id=$4 AND unit_id=$5 AND id<>$6 AND status IN ('sent','pending')`, r.tenant.ID, item.RunID, item.Revision, item.PartyID, item.UnitID, item.ID)
	if err != nil {
		return item, false, err
	}
	var sent, live bool
	var abandoned []string
	for rows.Next() {
		var id, status, at string
		if err := rows.Scan(&id, &status, &at); err != nil {
			rows.Close()
			return item, false, err
		}
		if status == "sent" {
			sent = true
			continue
		}
		reservedAt, err := time.Parse(time.RFC3339Nano, at)
		if err != nil {
			rows.Close()
			return item, false, err
		}
		if item.SentAt.Sub(reservedAt) < ValorisationDeliveryPendingTTL {
			live = true
		} else {
			abandoned = append(abandoned, id)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return item, false, err
	}
	if sent {
		// The rollback drops this reservation; nothing is recorded for a skip.
		return item, true, nil
	}
	if live {
		return item, false, fmt.Errorf("%w: reservation of %s/%s is live", ErrDeliveryInProgress, item.UnitID, item.PartyID)
	}
	for _, id := range abandoned {
		if _, err := tx.ExecContext(ctx, `UPDATE valorisation_deliveries SET status='failed',error=$1 WHERE tenant_id=$2 AND id=$3 AND status='pending'`, ValorisationDeliveryInterrupted, r.tenant.ID, id); err != nil {
			return item, false, err
		}
	}
	if err := tx.QueryRowContext(ctx, `SELECT attempt FROM valorisation_deliveries WHERE tenant_id=$1 AND id=$2`, r.tenant.ID, item.ID).Scan(&item.Attempt); err != nil {
		return item, false, err
	}
	if err := tx.Commit(); err != nil {
		return item, false, err
	}
	return item, false, nil
}

// record finalizes the reservation with the outcome of the send. A reservation
// that a later attempt already finalized as interrupted keeps that verdict: the
// stored row wins over this process's late result.
func (r *boundValorisationDeliveryRepository) record(ctx context.Context, handle txBeginner, item ValorisationDelivery) (ValorisationDelivery, error) {
	tx, err := handle.BeginTx(ctx, nil)
	if err != nil {
		return item, err
	}
	defer tx.Rollback()
	item.SentAt = time.Now().UTC()
	result, err := tx.ExecContext(ctx, `UPDATE valorisation_deliveries SET status=$1,error=$2,sent_at=$3 WHERE tenant_id=$4 AND id=$5 AND status='pending'`, item.Status, item.Error, item.SentAt.Format(time.RFC3339Nano), r.tenant.ID, item.ID)
	if err != nil {
		return item, err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return item, err
	}
	if changed == 0 {
		var at string
		if err := tx.QueryRowContext(ctx, `SELECT status,error,sent_at FROM valorisation_deliveries WHERE tenant_id=$1 AND id=$2`, r.tenant.ID, item.ID).Scan(&item.Status, &item.Error, &at); err != nil {
			return item, err
		}
		if item.SentAt, err = time.Parse(time.RFC3339Nano, at); err != nil {
			return item, err
		}
	}
	if err := tx.Commit(); err != nil {
		return item, err
	}
	return item, nil
}
