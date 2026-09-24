package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ImportDocument commits a pending identity before publishing the blob, then
// commits metadata and completion together. Retrying the same bytes reuses the
// first actor, timestamp, identity and blob key even after a process restart.
// DocumentStorage must be the SQL store attached to this same tenant database.
func (s *ImportLedger) ImportDocument(ctx context.Context, tenant TenantRef, key ImportKey, digest string, storage DocumentStorage, item DocumentRecord, filename string, data []byte) (DocumentRecord, bool, error) {
	if err := s.validate(tenant, key, digest); err != nil {
		return DocumentRecord{}, false, err
	}
	documents, ok := storage.(*SQLDocumentStore)
	if !ok || documents == nil || documents.db != s.db || documents.fileDir == "" {
		return DocumentRecord{}, false, fmt.Errorf("import ledger: matching SQL document store required")
	}
	if len(data) == 0 || len(data) > MaxDocumentBytes || fmt.Sprintf("%x", sha256.Sum256(data)) != digest {
		return DocumentRecord{}, false, fmt.Errorf("import ledger: document digest mismatch")
	}
	item, complete, err := s.reserveDocument(ctx, tenant, key, digest, item, filename, len(data))
	if err != nil || complete {
		return item, complete, err
	}
	if err := ctx.Err(); err != nil {
		return DocumentRecord{}, false, err
	}
	if err := writeImportBlob(documents.fileDir, item, data); err != nil {
		return DocumentRecord{}, false, err
	}
	return s.completeDocument(ctx, tenant, key, digest, item)
}

func (s *ImportLedger) reserveDocument(ctx context.Context, tenant TenantRef, key ImportKey, digest string, item DocumentRecord, filename string, size int) (DocumentRecord, bool, error) {
	// This key is a function of the house, format and exact bytes, never of
	// process state or user-supplied paths. Metadata remains private to the house.
	identity, _ := json.Marshal([]string{tenant.ID, key.Format, digest})
	item.ID = fmt.Sprintf("import-%x", sha256.Sum256(identity))
	item.TenantSlug, item.SeriesID = tenant.Slug, item.ID
	item.Version, item.Current = 1, true
	item.SupersedesID, item.ReplacedByID = "", ""
	item.Filename, item.StoredFilename = SanitizeDocumentFilename(filename), item.ID+".xml"
	item.Size, item.ContentType = int64(size), "application/xml"
	item.UploadedAt = time.Now().UTC()
	item = NormalizeDocumentRecord(item)
	if item.Title == "" || item.Category != DocumentCategoryBilling || item.Visibility != DocumentVisibilityManagerOnly || item.UploadedBy == "" || item.AnnualStatementArchive != nil {
		return DocumentRecord{}, false, fmt.Errorf("import ledger: invalid invoice document metadata")
	}
	blob, err := json.Marshal(item)
	if err != nil {
		return DocumentRecord{}, false, err
	}
	if err := ctx.Err(); err != nil {
		return DocumentRecord{}, false, err
	}
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return DocumentRecord{}, false, err
	}
	defer tx.Rollback()
	if _, err := insertImport(ctx, tx, tenant, key, digest, "pending", item.StoredFilename, string(blob)); err != nil {
		return DocumentRecord{}, false, err
	}
	var status, blobKey, metadata string
	if err := tx.QueryRowContext(ctx, `SELECT status, blob_key, document_data FROM integration_imports
		WHERE tenant_id=$1 AND format=$2 AND file_digest=$3`,
		tenant.ID, key.Format, digest).Scan(&status, &blobKey, &metadata); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DocumentRecord{}, true, nil
		}
		return DocumentRecord{}, false, err
	}
	if metadata == "" && status == "complete" {
		// Pre-migration completed imports have no stored document identity.
		return DocumentRecord{}, true, nil
	}
	var reserved DocumentRecord
	if err := json.Unmarshal([]byte(metadata), &reserved); err != nil {
		return DocumentRecord{}, false, err
	}
	if reserved.ID != item.ID || reserved.TenantSlug != tenant.Slug || blobKey != item.StoredFilename || reserved.StoredFilename != blobKey || reserved.Size != int64(size) {
		return DocumentRecord{}, false, fmt.Errorf("import ledger: reserved document identity mismatch")
	}
	if err := ctx.Err(); err != nil {
		return DocumentRecord{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return DocumentRecord{}, false, err
	}
	return reserved, status == "complete", nil
}

func (s *ImportLedger) completeDocument(ctx context.Context, tenant TenantRef, key ImportKey, digest string, item DocumentRecord) (DocumentRecord, bool, error) {
	if err := ctx.Err(); err != nil {
		return DocumentRecord{}, false, err
	}
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return DocumentRecord{}, false, err
	}
	defer tx.Rollback()
	// Conditional UPDATE locks the row on PostgreSQL (and the writer on
	// SQLite). A competing completer waits, then observes zero affected rows.
	result, err := tx.ExecContext(ctx, `UPDATE integration_imports SET status='complete', assigned=1, changed=1
		WHERE tenant_id=$1 AND format=$2 AND file_digest=$3 AND status='pending'`,
		tenant.ID, key.Format, digest)
	if err != nil {
		return DocumentRecord{}, false, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return DocumentRecord{}, false, err
	}
	if n == 0 {
		var status string
		if err := tx.QueryRowContext(ctx, `SELECT status FROM integration_imports
			WHERE tenant_id=$1 AND format=$2 AND file_digest=$3`,
			tenant.ID, key.Format, digest).Scan(&status); err != nil {
			return DocumentRecord{}, false, err
		}
		if status != "complete" {
			return DocumentRecord{}, false, sql.ErrNoRows
		}
		return item, true, nil
	}
	blob, err := json.Marshal(item)
	if err != nil {
		return DocumentRecord{}, false, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO documents(tenant_id, tenant_slug, id, data) VALUES($1,$2,$3,$4)`,
		tenant.ID, tenant.Slug, item.ID, string(blob)); err != nil {
		return DocumentRecord{}, false, err
	}
	if err := ctx.Err(); err != nil {
		return DocumentRecord{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return DocumentRecord{}, false, err
	}
	return CopyDocument(item), false, nil
}

// Publishing complete bytes by rename prevents a torn write from becoming the
// stable blob. Competing publishers have identical, digest-checked bytes. Never
// remove that stable file on SQL rollback: a retry must be able to adopt it.
func writeImportBlob(fileDir string, item DocumentRecord, data []byte) error {
	dir := filepath.Join(fileDir, item.TenantSlug)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(dir, ".import-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if _, err := file.Write(data); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(file.Name(), filepath.Join(dir, item.StoredFilename)); err != nil {
		return err
	}
	directory, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return err
	}
	// Persist creation of the house directory as well as its blob entry.
	parent, err := os.Open(fileDir)
	if err != nil {
		return err
	}
	defer parent.Close()
	return parent.Sync()
}
