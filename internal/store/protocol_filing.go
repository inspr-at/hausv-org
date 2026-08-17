package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// ProtocolFiler files a generated handover protocol as a document AND links it
// on the handover. Both steps must land together: a crash between them used to
// leave an orphaned PDF whose retry produced a duplicate (HAUSV-148).
//
// alreadyFiled reports that the handover was filed before this call; the caller
// should treat that as success and must NOT retry, because a retry is exactly
// what produced duplicates.
type ProtocolFiler interface {
	FileHandoverProtocol(tenant TenantRef, handoverID string, doc DocumentRecord, filename string, contentType string, data []byte, now time.Time) (created DocumentRecord, updated HandoverRecord, alreadyFiled bool, err error)
}

var (
	_ ProtocolFiler = (*SQLProtocolFiler)(nil)
	_ ProtocolFiler = (*SequentialProtocolFiler)(nil)
)

// SQLProtocolFiler writes the document row and the handover link inside ONE
// transaction. Only usable when both stores share a database.
type SQLProtocolFiler struct {
	db        *sql.DB
	documents *SQLDocumentStore
	handovers *SQLHandoverStore
}

// NewSQLProtocolFiler returns nil unless both stores are present and share the
// same database — the caller then falls back to the sequential filer.
func NewSQLProtocolFiler(documents *SQLDocumentStore, handovers *SQLHandoverStore) *SQLProtocolFiler {
	if documents == nil || handovers == nil || documents.db == nil || documents.db != handovers.db {
		return nil
	}
	return &SQLProtocolFiler{db: documents.db, documents: documents, handovers: handovers}
}

func (f *SQLProtocolFiler) FileHandoverProtocol(tenant TenantRef, handoverID string, doc DocumentRecord, filename string, contentType string, data []byte, now time.Time) (DocumentRecord, HandoverRecord, bool, error) {
	if f == nil {
		return DocumentRecord{}, HandoverRecord{}, false, fmt.Errorf("protocol filer unavailable")
	}
	resolvedTenant, tenantOK := validTenantRef(tenant)
	handoverID = strings.TrimSpace(handoverID)
	if !tenantOK || handoverID == "" {
		return DocumentRecord{}, HandoverRecord{}, false, fmt.Errorf("invalid handover reference")
	}
	tenantSlug := resolvedTenant.Slug
	if now.IsZero() {
		now = time.Now()
	}
	doc.TenantSlug = tenantSlug

	// The file must exist before the transaction; if the transaction does not
	// commit we remove it again, so no orphan is left behind.
	record, path, err := prepareGeneratedDocument(f.documents.fileDir, doc, filename, contentType, data, now)
	if err != nil {
		return DocumentRecord{}, HandoverRecord{}, false, err
	}

	tx, err := f.db.Begin()
	if err != nil {
		_ = os.Remove(path)
		return DocumentRecord{}, HandoverRecord{}, false, err
	}
	defer tx.Rollback()

	// Acquire the handover row's write lock before reading it. SQLite already
	// serialized these writers at the database level; PostgreSQL otherwise lets
	// both transactions observe an empty FiledDocumentID and create duplicates.
	locked, err := tx.Exec(`UPDATE handovers SET data=data WHERE tenant_id=$1 AND id=$2`, resolvedTenant.ID, handoverID)
	if err != nil {
		_ = os.Remove(path)
		return DocumentRecord{}, HandoverRecord{}, false, err
	}
	if count, err := locked.RowsAffected(); err != nil || count == 0 {
		_ = os.Remove(path)
		return DocumentRecord{}, HandoverRecord{}, false, fmt.Errorf("handover not found")
	}

	var raw string
	if err := tx.QueryRow(
		`SELECT data FROM handovers WHERE tenant_id=$1 AND id=$2`, resolvedTenant.ID, handoverID,
	).Scan(&raw); err != nil {
		_ = os.Remove(path)
		return DocumentRecord{}, HandoverRecord{}, false, fmt.Errorf("handover not found")
	}
	var handover HandoverRecord
	if err := json.Unmarshal([]byte(raw), &handover); err != nil {
		_ = os.Remove(path)
		return DocumentRecord{}, HandoverRecord{}, false, fmt.Errorf("handover not found")
	}

	// Authoritative idempotency check, inside the transaction: a concurrent or
	// retried filing must not produce a second protocol document.
	if strings.TrimSpace(handover.FiledDocumentID) != "" {
		_ = os.Remove(path)
		return DocumentRecord{}, CopyHandover(handover), true, nil
	}

	if err := f.documents.writeTx(tx, resolvedTenant, record); err != nil {
		_ = os.Remove(path)
		return DocumentRecord{}, HandoverRecord{}, false, err
	}
	handover.FiledDocumentID = record.ID
	handover.UpdatedAt = now.UTC()
	saved := NormalizeHandover(handover)
	if err := f.handovers.writeTx(tx, resolvedTenant, saved); err != nil {
		_ = os.Remove(path)
		return DocumentRecord{}, HandoverRecord{}, false, err
	}
	if err := tx.Commit(); err != nil {
		_ = os.Remove(path)
		return DocumentRecord{}, HandoverRecord{}, false, err
	}
	return CopyDocument(record), CopyHandover(saved), false, nil
}

// SequentialProtocolFiler reproduces the pre-SQLite two-step behaviour. Used
// when the stores are on the JSON fallback, where no shared transaction exists.
// It is idempotent but NOT atomic: a crash between the two writes can still
// orphan a document, which is precisely why the SQL filer is preferred.
type SequentialProtocolFiler struct {
	documents DocumentStorage
	handovers handoverBackend
}

func NewSequentialProtocolFiler(documents DocumentStorage, handovers HandoverStorage) *SequentialProtocolFiler {
	backend, _ := handovers.(handoverBackend)
	return &SequentialProtocolFiler{documents: documents, handovers: backend}
}

func (f *SequentialProtocolFiler) FileHandoverProtocol(tenant TenantRef, handoverID string, doc DocumentRecord, filename string, contentType string, data []byte, now time.Time) (DocumentRecord, HandoverRecord, bool, error) {
	if f == nil || f.documents == nil || f.handovers == nil {
		return DocumentRecord{}, HandoverRecord{}, false, fmt.Errorf("protocol filer unavailable")
	}
	resolvedTenant, tenantOK := validTenantRef(tenant)
	handoverID = strings.TrimSpace(handoverID)
	if !tenantOK || handoverID == "" {
		return DocumentRecord{}, HandoverRecord{}, false, fmt.Errorf("invalid handover reference")
	}
	existing, found := f.handovers.get(resolvedTenant, handoverID)
	if !found {
		return DocumentRecord{}, HandoverRecord{}, false, fmt.Errorf("handover not found")
	}
	if strings.TrimSpace(existing.FiledDocumentID) != "" {
		return DocumentRecord{}, existing, true, nil
	}
	documents, ok := BindDocumentRepository(f.documents, resolvedTenant)
	if !ok {
		return DocumentRecord{}, HandoverRecord{}, false, fmt.Errorf("document store unavailable")
	}
	created, err := documents.CreateGenerated(doc, filename, contentType, data, now)
	if err != nil {
		return DocumentRecord{}, HandoverRecord{}, false, err
	}
	updated, _, err := f.handovers.setFiledDocument(resolvedTenant, handoverID, created.ID, now)
	if err != nil {
		return DocumentRecord{}, HandoverRecord{}, false, err
	}
	return created, updated, false, nil
}
