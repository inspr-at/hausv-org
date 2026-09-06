package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// ErrDeliveryInProgress means another request reserved the same delivery.
var ErrDeliveryInProgress = errors.New("annual statement delivery in progress")

type AnnualStatementDelivery struct {
	ID, RunID                                      string
	Revision                                       int
	PartyID, UnitID, DocumentID, SHA256, Recipient string
	SentAt                                         time.Time
	Status, Error, Actor                           string
	Attempt                                        int
}

type AnnualStatementDeliveryRepository interface {
	List(runID string) ([]AnnualStatementDelivery, error)
	// Attempt serializes delivery of a party/unit and skips any previous success.
	// send must return an error safe to display to the Verwaltung.
	Attempt(ctx context.Context, item AnnualStatementDelivery, send func(context.Context) error) (AnnualStatementDelivery, bool, error)
}

type SQLAnnualStatementDeliveryStore struct{ db *TenantDB }

func NewSQLAnnualStatementDeliveryStore(db *TenantDB) *SQLAnnualStatementDeliveryStore {
	return &SQLAnnualStatementDeliveryStore{db}
}

type boundAnnualStatementDeliveryRepository struct {
	storage *SQLAnnualStatementDeliveryStore
	tenant  TenantRef
}

func BindAnnualStatementDeliveryRepository(storage *SQLAnnualStatementDeliveryStore, tenant TenantRef) (AnnualStatementDeliveryRepository, bool) {
	resolved, valid := validTenantRef(tenant)
	if storage == nil || !valid {
		return nil, false
	}
	return &boundAnnualStatementDeliveryRepository{storage, resolved}, true
}

func (r *boundAnnualStatementDeliveryRepository) List(runID string) ([]AnnualStatementDelivery, error) {
	rows, err := r.storage.db.For(r.tenant).Query(`SELECT id,run_id,revision,party_id,unit_id,document_id,sha256,recipient,sent_at,status,error,actor,attempt FROM annual_statement_deliveries WHERE tenant_id=$1 AND run_id=$2 ORDER BY attempt,id`, r.tenant.ID, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AnnualStatementDelivery{}
	for rows.Next() {
		var item AnnualStatementDelivery
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

func (r *boundAnnualStatementDeliveryRepository) Attempt(ctx context.Context, item AnnualStatementDelivery, send func(context.Context) error) (AnnualStatementDelivery, bool, error) {
	if err := ctx.Err(); err != nil {
		return item, false, err
	}
	if item.RunID == "" || item.Revision < 1 || item.UnitID == "" || item.DocumentID == "" || item.Actor == "" || send == nil {
		return item, false, fmt.Errorf("invalid annual statement delivery")
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return item, false, err
	}
	item.ID = fmt.Sprintf("%x", id)
	handle, ok := r.storage.db.For(r.tenant).(interface {
		BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	})
	if !ok {
		return item, false, fmt.Errorf("delivery transactions unavailable")
	}
	// Retain a bounded opportunity to record the outcome when the request is
	// canceled during SMTP. There is no atomic commit spanning SMTP and SQL.
	recordCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	tx, err := handle.BeginTx(recordCtx, nil)
	if err != nil {
		return item, false, err
	}
	defer tx.Rollback()
	item.SentAt = time.Now().UTC()
	// The unique attempt key serializes concurrent stores before either can
	// send. A conflicting transaction fails closed and can be retried by the user.
	_, err = tx.ExecContext(recordCtx, `INSERT INTO annual_statement_deliveries(tenant_id,tenant_slug,id,run_id,revision,party_id,unit_id,document_id,sha256,recipient,sent_at,status,error,actor,attempt)
 SELECT CAST($1 AS VARCHAR(26)),$2,$3,$4,CAST($5 AS BIGINT),$6,$7,$8,$9,$10,$11,'failed','Versand unterbrochen',$12,COALESCE(MAX(attempt),0)+1 FROM annual_statement_deliveries WHERE tenant_id=$1 AND run_id=$4 AND revision=$5 AND party_id=$6 AND unit_id=$7`, r.tenant.ID, r.tenant.Slug, item.ID, item.RunID, item.Revision, item.PartyID, item.UnitID, item.DocumentID, item.SHA256, item.Recipient, item.SentAt.Format(time.RFC3339Nano), item.Actor)
	if err != nil {
		var pgErr *pgconn.PgError
		var sqliteErr *sqlite.Error
		if (errors.As(err, &pgErr) && pgErr.Code == "23505") ||
			(errors.As(err, &sqliteErr) && (sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE || sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY)) {
			return item, false, fmt.Errorf("%w: %w", ErrDeliveryInProgress, err)
		}
		return item, false, err
	}
	var sent int
	err = tx.QueryRowContext(recordCtx, `SELECT COUNT(*) FROM annual_statement_deliveries WHERE tenant_id=$1 AND run_id=$2 AND revision=$3 AND party_id=$4 AND unit_id=$5 AND status='sent'`, r.tenant.ID, item.RunID, item.Revision, item.PartyID, item.UnitID).Scan(&sent)
	if err != nil {
		return item, false, err
	}
	if sent > 0 {
		return item, true, nil
	}
	if err := ctx.Err(); err != nil {
		return item, false, err
	}
	item.Status, item.Error = "sent", ""
	if err := send(ctx); err != nil {
		item.Status, item.Error = "failed", err.Error()
	}
	item.SentAt = time.Now().UTC()
	_, err = tx.ExecContext(recordCtx, `UPDATE annual_statement_deliveries SET status=$1,error=$2,sent_at=$3 WHERE tenant_id=$4 AND id=$5`, item.Status, item.Error, item.SentAt.Format(time.RFC3339Nano), r.tenant.ID, item.ID)
	if err != nil {
		return item, false, err
	}
	if err = tx.QueryRowContext(recordCtx, `SELECT attempt FROM annual_statement_deliveries WHERE tenant_id=$1 AND id=$2`, r.tenant.ID, item.ID).Scan(&item.Attempt); err != nil {
		return item, false, err
	}
	if err = tx.Commit(); err != nil {
		return item, false, err
	}
	return item, false, nil
}
