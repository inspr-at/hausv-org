package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

// ImportKey identifies the kind of import and records its first application.
// File content is deduplicated per house and format, regardless of preview,
// filename, actor or selected period.
type ImportKey struct {
	Format, SourceVersion, AppliedBy string
}

type ImportCounts struct {
	Assigned, Changed, Unclear, Rejected int
}

type ImportLedger struct{ db *TenantDB }

func NewImportLedger(db *TenantDB) *ImportLedger { return &ImportLedger{db: db} }

// ImportTx exposes only effects that run on the transaction's tenant lane.
// Callers must not perform external writes inside Apply's callback.
type ImportTx struct {
	setPaymentStatus func(UnitPaymentStatus) (UnitPaymentStatus, bool, error)
	Counts           ImportCounts
}

func (s *ImportLedger) validate(tenant TenantRef, key ImportKey, digest string) error {
	if s == nil || s.db == nil || !tenant.Valid() || strings.TrimSpace(key.Format) == "" || strings.TrimSpace(digest) == "" {
		return fmt.Errorf("import ledger: database, tenant, format and digest required")
	}
	return nil
}

// Completed is advisory for previews. Apply's insert, not this lookup or any
// process-local lock, serializes concurrent imports. Boot backfills legacy
// SQLite identities; PostgreSQL prohibits NULL identities and enforces RLS.
func (s *ImportLedger) Completed(ctx context.Context, tenant TenantRef, key ImportKey, digest string) (bool, error) {
	if err := s.validate(tenant, key, digest); err != nil {
		return false, err
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	var complete bool
	err := s.db.For(tenant).QueryRow(`SELECT EXISTS(SELECT 1 FROM integration_imports
		WHERE tenant_id=$1 AND format=$2 AND file_digest=$3 AND status='complete')`,
		tenant.ID, key.Format, strings.TrimSpace(digest)).Scan(&complete)
	return complete, err
}

// Apply inserts the unique ledger key FIRST and commits it with every callback
// effect. already is true only for an existing completed import. A callback
// failure rolls both back. After an ambiguous commit error, retrying the key
// resolves the outcome without replaying an already committed effect.
func (s *ImportLedger) Apply(ctx context.Context, tenant TenantRef, key ImportKey, digest string, effect func(*ImportTx) error) (already bool, err error) {
	if err := s.validate(tenant, key, digest); err != nil {
		return false, err
	}
	if effect == nil {
		return false, fmt.Errorf("import ledger: effect required")
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	inserted, err := insertImport(ctx, tx, tenant, key, digest, "complete", "", "")
	if err != nil {
		return false, err
	}
	if !inserted {
		var status string
		err := tx.QueryRowContext(ctx, `SELECT status FROM integration_imports WHERE tenant_id=$1 AND format=$2 AND file_digest=$3`,
			tenant.ID, key.Format, strings.TrimSpace(digest)).Scan(&status)
		// A pre-identity SQLite row may conflict without being readable until
		// boot backfills it. Preserve that evidence and never replay effects.
		if errors.Is(err, sql.ErrNoRows) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
		if status != "complete" {
			return false, fmt.Errorf("import ledger: pending document requires completion")
		}
		return true, nil
	}
	unit := &ImportTx{setPaymentStatus: func(item UnitPaymentStatus) (UnitPaymentStatus, bool, error) {
		return setImportedPaymentStatus(ctx, tx, tenant, item)
	}}
	if err := effect(unit); err != nil {
		return false, err
	}
	c := unit.Counts
	if _, err := tx.ExecContext(ctx, `UPDATE integration_imports SET assigned=$1, changed=$2, unclear=$3, rejected=$4
		WHERE tenant_id=$5 AND format=$6 AND file_digest=$7`,
		c.Assigned, c.Changed, c.Unclear, c.Rejected, tenant.ID, key.Format, strings.TrimSpace(digest)); err != nil {
		return false, err
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	return false, tx.Commit()
}

func insertImport(ctx context.Context, tx *sql.Tx, tenant TenantRef, key ImportKey, digest, status, blobKey, documentData string) (bool, error) {
	result, err := tx.ExecContext(ctx, `INSERT INTO integration_imports(
		tenant_id, tenant_slug, format, file_digest, source_version, applied_at, applied_by, status, blob_key, document_data)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT(tenant_slug, format, file_digest) DO NOTHING`,
		tenant.ID, tenant.Slug, key.Format, strings.TrimSpace(digest), strings.TrimSpace(key.SourceVersion),
		time.Now().UTC().Format(time.RFC3339Nano), textutil.Email(key.AppliedBy), status, blobKey, documentData)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}

// SetPaymentStatus updates only actual status changes, within the import's
// transaction. Errors are propagated rather than mistaken for missing rows.
func (u *ImportTx) SetPaymentStatus(item UnitPaymentStatus) (UnitPaymentStatus, bool, error) {
	return u.setPaymentStatus(item)
}

func setImportedPaymentStatus(ctx context.Context, tx *sql.Tx, tenant TenantRef, item UnitPaymentStatus) (UnitPaymentStatus, bool, error) {
	item.TenantSlug = tenant.Slug
	item, err := NormalizeUnitPaymentRecord(item)
	if err != nil {
		return UnitPaymentStatus{}, false, err
	}
	item.UpdatedAt = time.Now().UTC().Truncate(time.Second)
	result, err := tx.ExecContext(ctx, `INSERT INTO unit_payment_status(tenant_id, tenant_slug, unit_id, status, updated_at, updated_by)
		VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(tenant_slug, unit_id) DO UPDATE SET
		status=excluded.status, updated_at=excluded.updated_at, updated_by=excluded.updated_by,
		tenant_id=coalesce(unit_payment_status.tenant_id, excluded.tenant_id)
		WHERE unit_payment_status.status<>excluded.status`, tenant.ID, tenant.Slug,
		item.UnitID, item.Status, item.UpdatedAt.Format(time.RFC3339Nano), item.UpdatedBy)
	if err != nil {
		return UnitPaymentStatus{}, false, err
	}
	n, err := result.RowsAffected()
	return item, n == 1, err
}
