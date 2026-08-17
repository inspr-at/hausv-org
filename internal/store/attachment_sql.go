package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

type AttachmentRepository interface {
	CreateUploaded(entityType string, entityID string, uploadedBy string, uploads []UploadedFile, now time.Time) ([]AttachmentRecord, error)
	ListEntity(entityType string, entityID string) []AttachmentRecord
	Get(id string) (AttachmentRecord, bool)
	Delete(id string, deletedAt time.Time) (AttachmentRecord, bool, error)
	FilePath(item AttachmentRecord, variant string) (string, string, int64, bool)
}

type AttachmentStorage interface{ attachmentStorage() }

var (
	_ AttachmentStorage = (*AttachmentStore)(nil)
	_ AttachmentStorage = (*SQLAttachmentStore)(nil)
)

type attachmentStoreBackend interface {
	createUploaded(tenant TenantRef, entityType string, entityID string, uploadedBy string, uploads []UploadedFile, now time.Time) ([]AttachmentRecord, error)
	listEntity(tenant TenantRef, entityType string, entityID string) []AttachmentRecord
	get(tenant TenantRef, id string) (AttachmentRecord, bool)
	delete(tenant TenantRef, id string, deletedAt time.Time) (AttachmentRecord, bool, error)
	filePath(item AttachmentRecord, variant string) (string, string, int64, bool)
}

type boundAttachmentRepository struct {
	storage attachmentStoreBackend
	tenant  TenantRef
}

func BindAttachmentRepository(storage AttachmentStorage, tenant TenantRef) (AttachmentRepository, bool) {
	resolvedTenant, tenantOK := validTenantRef(tenant)
	backend, ok := storage.(attachmentStoreBackend)
	if !ok || !tenantOK {
		return nil, false
	}
	return &boundAttachmentRepository{storage: backend, tenant: resolvedTenant}, true
}

func (r *boundAttachmentRepository) CreateUploaded(entityType, entityID, uploadedBy string, uploads []UploadedFile, now time.Time) ([]AttachmentRecord, error) {
	return r.storage.createUploaded(r.tenant, entityType, entityID, uploadedBy, uploads, now)
}
func (r *boundAttachmentRepository) ListEntity(entityType, entityID string) []AttachmentRecord {
	return r.storage.listEntity(r.tenant, entityType, entityID)
}
func (r *boundAttachmentRepository) Get(id string) (AttachmentRecord, bool) {
	return r.storage.get(r.tenant, id)
}
func (r *boundAttachmentRepository) Delete(id string, deletedAt time.Time) (AttachmentRecord, bool, error) {
	return r.storage.delete(r.tenant, id, deletedAt)
}
func (r *boundAttachmentRepository) FilePath(item AttachmentRecord, variant string) (string, string, int64, bool) {
	if textutil.Slug(item.TenantSlug) != r.tenant.Slug {
		return "", "", 0, false
	}
	return r.storage.filePath(item, variant)
}

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

func (*SQLAttachmentStore) attachmentStorage() {}

func (s *SQLAttachmentStore) writeTx(tx *sql.Tx, tenant TenantRef, item AttachmentRecord) error {
	blob, err := json.Marshal(item)
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		`INSERT INTO attachments(tenant_id, tenant_slug, id, data) VALUES($1, $2, $3, $4)
		 ON CONFLICT(tenant_slug, id) DO UPDATE SET data=excluded.data,
		   tenant_id=coalesce(attachments.tenant_id, excluded.tenant_id)`,
		tenant.ID, tenant.Slug, item.ID, string(blob),
	)
	return err
}

// CreateUploaded writes every accepted upload to disk and then persists ALL
// records in one transaction, so a batch upload can never land half-recorded.
// Any failure removes the files written so far.
func (s *SQLAttachmentStore) createUploaded(tenant TenantRef, entityType string, entityID string, uploadedBy string, uploads []UploadedFile, now time.Time) ([]AttachmentRecord, error) {
	tenantSlug := tenant.Slug
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
		if err := s.writeTx(tx, tenant, item); err != nil {
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

func (s *SQLAttachmentStore) allForTenant(tenant TenantRef) []AttachmentRecord {
	rows, err := s.db.Query(`SELECT data FROM attachments WHERE tenant_id=$1`, tenant.ID)
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

func (s *SQLAttachmentStore) listEntity(tenant TenantRef, entityType string, entityID string) []AttachmentRecord {
	tenantSlug := tenant.Slug
	if s == nil {
		return nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	entityType = NormalizeAttachmentEntity(entityType)
	entityID = strings.TrimSpace(entityID)
	items := []AttachmentRecord{}
	for _, item := range s.allForTenant(tenant) {
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

func (s *SQLAttachmentStore) get(tenant TenantRef, id string) (AttachmentRecord, bool) {
	tenantSlug := tenant.Slug
	if s == nil {
		return AttachmentRecord{}, false
	}
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	var data string
	if err := s.db.QueryRow(`SELECT data FROM attachments WHERE tenant_id=$1 AND id=$2`, tenant.ID, id).Scan(&data); err != nil {
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
func (s *SQLAttachmentStore) delete(tenant TenantRef, id string, deletedAt time.Time) (AttachmentRecord, bool, error) {
	tenantSlug := tenant.Slug
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
	if err := tx.QueryRow(`SELECT data FROM attachments WHERE tenant_id=$1 AND id=$2`, tenant.ID, id).Scan(&data); err != nil {
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
	if err := s.writeTx(tx, tenant, marked); err != nil {
		return AttachmentRecord{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return AttachmentRecord{}, false, err
	}
	removeAttachmentFilesIn(s.fileDir, item)
	return item, true, nil
}

func (s *SQLAttachmentStore) filePath(item AttachmentRecord, variant string) (string, string, int64, bool) {
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
	tenants := newTenantIDCache(s.db)
	for _, item := range snapshot {
		if strings.TrimSpace(item.ID) == "" || textutil.Slug(item.TenantSlug) == "" {
			continue
		}
		tenant, err := tenants.ref(item.TenantSlug)
		if err != nil {
			return err
		}
		blob, err := json.Marshal(item)
		if err != nil {
			return err
		}
		if _, err := s.db.Exec(
			`INSERT INTO attachments(tenant_id, tenant_slug, id, data) VALUES($1, $2, $3, $4) ON CONFLICT(tenant_slug, id) DO NOTHING`,
			tenant.ID, tenant.Slug, item.ID, string(blob),
		); err != nil {
			return err
		}
	}
	return nil
}

// AttachmentTombstoneRetention is how long a soft-deleted attachment record is
// kept after its files are gone. The record itself carries no content — the
// files are hard-deleted at delete time — so it exists only to make the row's
// former presence explainable. The audit log is the durable record of the
// deletion, so purging the tombstone loses nothing (HAUSV-146).
//
// Generous on purpose: at WEG scale this bounds an already tiny table rather
// than reclaiming meaningful space.
const AttachmentTombstoneRetention = 365 * 24 * time.Hour

// PurgeDeletedBefore removes soft-deleted attachment records whose deletion is
// older than cutoff, so the table cannot grow without bound (HAUSV-146).
// Live attachments are never touched.
func (s *SQLAttachmentStore) PurgeDeletedBefore(cutoff time.Time) (int, error) {
	if s == nil {
		return 0, nil
	}
	rows, err := s.db.Query(`SELECT tenant_id, id, data FROM attachments`)
	if err != nil {
		return 0, err
	}
	type key struct{ tenant, id string }
	stale := []key{}
	for rows.Next() {
		var k key
		var data string
		if err := rows.Scan(&k.tenant, &k.id, &data); err != nil {
			continue
		}
		var item AttachmentRecord
		if err := json.Unmarshal([]byte(data), &item); err != nil {
			continue
		}
		// Only tombstones, and only ones older than the cutoff.
		if item.DeletedAt == nil || !item.DeletedAt.Before(cutoff) {
			continue
		}
		stale = append(stale, k)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	for _, k := range stale {
		if _, err := tx.Exec(`DELETE FROM attachments WHERE tenant_id=$1 AND id=$2`, k.tenant, k.id); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return len(stale), nil
}
