package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/markus-barta/hausv-org/internal/textutil"
)

// AttachmentStorage is the behaviour both the JSON AttachmentStore and the
// SQLite SQLAttachmentStore satisfy (HAUSV-168). Like documents, the file bytes
// stay on disk; only the metadata record moves to the database. Deletion is a
// soft delete of the record plus a hard delete of the files.
type AttachmentStorage interface {
	CreateUploaded(tenantSlug string, entityType string, entityID string, uploadedBy string, uploads []UploadedFile, now time.Time) ([]AttachmentRecord, error)
	ListEntity(tenantSlug string, entityType string, entityID string) []AttachmentRecord
	Get(tenantSlug string, id string) (AttachmentRecord, bool)
	Delete(tenantSlug string, id string, deletedAt time.Time) (AttachmentRecord, bool, error)
	FilePath(item AttachmentRecord, variant string) (string, string, int64, bool)
}

var (
	_ AttachmentStorage = (*AttachmentStore)(nil)
	_ AttachmentStorage = (*SQLAttachmentStore)(nil)
)

// SQLAttachmentStore keeps each attachment's metadata as a JSON document keyed
// by (tenant, id); the file and its image variants stay on disk. Table from
// migration 0012.
type SQLAttachmentStore struct {
	db      *sql.DB
	fileDir string
}

func NewSQLAttachmentStore(db *sql.DB, fileDir string) *SQLAttachmentStore {
	return &SQLAttachmentStore{db: db, fileDir: fileDir}
}

func (s *SQLAttachmentStore) writeTx(tx *sql.Tx, item AttachmentRecord) error {
	blob, err := json.Marshal(item)
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		`INSERT INTO attachments(tenant_slug, id, data) VALUES(?, ?, ?)
		 ON CONFLICT(tenant_slug, id) DO UPDATE SET data=excluded.data`,
		textutil.Slug(item.TenantSlug), item.ID, string(blob),
	)
	return err
}

// CreateUploaded writes every accepted upload to disk and then persists ALL
// records in one transaction, so a batch upload can never land half-recorded.
// Any failure removes the files written so far.
func (s *SQLAttachmentStore) CreateUploaded(tenantSlug string, entityType string, entityID string, uploadedBy string, uploads []UploadedFile, now time.Time) ([]AttachmentRecord, error) {
	if s == nil || len(uploads) == 0 {
		return nil, nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	entityType = NormalizeAttachmentEntity(entityType)
	entityID = strings.TrimSpace(entityID)
	uploadedBy = textutil.Email(uploadedBy)
	if tenantSlug == "" || entityType == "" || entityID == "" || uploadedBy == "" {
		return nil, fmt.Errorf("attachment target required")
	}

	created := []AttachmentRecord{}
	rollback := func() {
		for _, item := range created {
			removeAttachmentFilesIn(s.fileDir, item)
		}
	}
	for _, upload := range uploads {
		if upload.Open == nil || strings.TrimSpace(upload.Filename) == "" || upload.Size == 0 {
			continue
		}
		if upload.Size > MaxAttachmentBytes {
			rollback()
			return nil, fmt.Errorf("attachment too large")
		}
		id, err := randomToken(12)
		if err != nil {
			rollback()
			return nil, err
		}
		fileSave, err := saveUploadedAttachmentFileIn(s.fileDir, tenantSlug, id, upload)
		if err != nil {
			rollback()
			return nil, err
		}
		created = append(created, AttachmentRecord{
			ID:                 id,
			TenantSlug:         tenantSlug,
			EntityType:         entityType,
			EntityID:           entityID,
			UploadedBy:         uploadedBy,
			Filename:           SanitizeDocumentFilename(upload.Filename),
			StoredFilename:     fileSave.StoredFilename,
			ContentType:        fileSave.ContentType,
			Size:               fileSave.Size,
			PreviewFilename:    fileSave.PreviewFilename,
			PreviewContentType: fileSave.PreviewContentType,
			PreviewSize:        fileSave.PreviewSize,
			ThumbFilename:      fileSave.ThumbFilename,
			ThumbContentType:   fileSave.ThumbContentType,
			ThumbSize:          fileSave.ThumbSize,
			CreatedAt:          now.UTC(),
		})
	}
	if len(created) == 0 {
		return nil, nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		rollback()
		return nil, err
	}
	defer tx.Rollback()
	for _, item := range created {
		if err := s.writeTx(tx, item); err != nil {
			rollback()
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		rollback()
		return nil, err
	}
	return created, nil
}

func (s *SQLAttachmentStore) allForTenant(tenantSlug string) []AttachmentRecord {
	rows, err := s.db.Query(`SELECT data FROM attachments WHERE tenant_slug=?`, tenantSlug)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []AttachmentRecord{}
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			continue
		}
		var item AttachmentRecord
		if err := json.Unmarshal([]byte(data), &item); err != nil {
			continue
		}
		out = append(out, item)
	}
	return out
}

func (s *SQLAttachmentStore) ListEntity(tenantSlug string, entityType string, entityID string) []AttachmentRecord {
	if s == nil {
		return nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	entityType = NormalizeAttachmentEntity(entityType)
	entityID = strings.TrimSpace(entityID)
	items := []AttachmentRecord{}
	for _, item := range s.allForTenant(tenantSlug) {
		if item.DeletedAt != nil {
			continue
		}
		if NormalizeAttachmentEntity(item.EntityType) == entityType && strings.TrimSpace(item.EntityID) == entityID {
			items = append(items, item)
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	return items
}

func (s *SQLAttachmentStore) Get(tenantSlug string, id string) (AttachmentRecord, bool) {
	if s == nil {
		return AttachmentRecord{}, false
	}
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	var data string
	if err := s.db.QueryRow(`SELECT data FROM attachments WHERE tenant_slug=? AND id=?`, tenantSlug, id).Scan(&data); err != nil {
		return AttachmentRecord{}, false
	}
	var item AttachmentRecord
	if err := json.Unmarshal([]byte(data), &item); err != nil {
		return AttachmentRecord{}, false
	}
	if item.DeletedAt != nil {
		return AttachmentRecord{}, false
	}
	return item, true
}

// Delete soft-deletes the record and then removes the files. The record is only
// marked once the transaction commits, so a failure leaves the files intact.
func (s *SQLAttachmentStore) Delete(tenantSlug string, id string, deletedAt time.Time) (AttachmentRecord, bool, error) {
	if s == nil {
		return AttachmentRecord{}, false, nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)

	tx, err := s.db.Begin()
	if err != nil {
		return AttachmentRecord{}, false, err
	}
	defer tx.Rollback()
	var data string
	if err := tx.QueryRow(`SELECT data FROM attachments WHERE tenant_slug=? AND id=?`, tenantSlug, id).Scan(&data); err != nil {
		return AttachmentRecord{}, false, nil
	}
	var item AttachmentRecord
	if err := json.Unmarshal([]byte(data), &item); err != nil {
		return AttachmentRecord{}, false, nil
	}
	if item.DeletedAt != nil {
		return AttachmentRecord{}, false, nil
	}
	deleted := deletedAt.UTC()
	marked := item
	marked.DeletedAt = &deleted
	if err := s.writeTx(tx, marked); err != nil {
		return AttachmentRecord{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return AttachmentRecord{}, false, err
	}
	removeAttachmentFilesIn(s.fileDir, item)
	return item, true, nil
}

func (s *SQLAttachmentStore) FilePath(item AttachmentRecord, variant string) (string, string, int64, bool) {
	if s == nil {
		return "", "", 0, false
	}
	return attachmentFilePathIn(s.fileDir, item, variant)
}

// ImportAttachments copies metadata records from a JSON store, each only if
// absent (clobber-safe). Files already live on disk and are not touched
// (HAUSV-170).
func (s *SQLAttachmentStore) ImportAttachments(src *AttachmentStore) error {
	if src == nil {
		return nil
	}
	src.mu.Lock()
	snapshot := append([]AttachmentRecord(nil), src.data.Attachments...)
	src.mu.Unlock()
	for _, item := range snapshot {
		if strings.TrimSpace(item.ID) == "" || textutil.Slug(item.TenantSlug) == "" {
			continue
		}
		blob, err := json.Marshal(item)
		if err != nil {
			return err
		}
		if _, err := s.db.Exec(
			`INSERT INTO attachments(tenant_slug, id, data) VALUES(?, ?, ?) ON CONFLICT(tenant_slug, id) DO NOTHING`,
			textutil.Slug(item.TenantSlug), item.ID, string(blob),
		); err != nil {
			return err
		}
	}
	return nil
}
