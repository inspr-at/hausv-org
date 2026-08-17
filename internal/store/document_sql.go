package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

type DocumentRepository interface {
	Create(item DocumentRecord, upload UploadedFile, now time.Time) (DocumentRecord, error)
	CreateGenerated(item DocumentRecord, filename string, contentType string, data []byte, now time.Time) (DocumentRecord, error)
	Replace(id string, uploadedBy string, upload UploadedFile, now time.Time) (DocumentRecord, DocumentRecord, error)
	List() []DocumentRecord
	ListCurrent() []DocumentRecord
	Versions(seriesID string) []DocumentRecord
	Get(id string) (DocumentRecord, bool)
	FilePath(item DocumentRecord) (string, bool)
}

type DocumentStorage interface{ documentStorage() }

var (
	_ DocumentStorage = (*DocumentStore)(nil)
	_ DocumentStorage = (*SQLDocumentStore)(nil)
)

type documentBackend interface {
	create(tenantSlug string, item DocumentRecord, upload UploadedFile, now time.Time) (DocumentRecord, error)
	createGenerated(tenantSlug string, item DocumentRecord, filename string, contentType string, data []byte, now time.Time) (DocumentRecord, error)
	replace(tenantSlug string, id string, uploadedBy string, upload UploadedFile, now time.Time) (DocumentRecord, DocumentRecord, error)
	listTenant(tenantSlug string) []DocumentRecord
	listCurrentTenant(tenantSlug string) []DocumentRecord
	versions(tenantSlug string, seriesID string) []DocumentRecord
	get(tenantSlug string, id string) (DocumentRecord, bool)
	filePath(item DocumentRecord) (string, bool)
}

type boundDocumentRepository struct {
	storage    documentBackend
	tenantSlug string
}

func BindDocumentRepository(storage DocumentStorage, tenantSlug string) (DocumentRepository, bool) {
	tenantSlug = textutil.Slug(tenantSlug)
	backend, ok := storage.(documentBackend)
	if !ok || tenantSlug == "" {
		return nil, false
	}
	return &boundDocumentRepository{storage: backend, tenantSlug: tenantSlug}, true
}

func (r *boundDocumentRepository) Create(item DocumentRecord, upload UploadedFile, now time.Time) (DocumentRecord, error) {
	return r.storage.create(r.tenantSlug, item, upload, now)
}
func (r *boundDocumentRepository) CreateGenerated(item DocumentRecord, filename, contentType string, data []byte, now time.Time) (DocumentRecord, error) {
	return r.storage.createGenerated(r.tenantSlug, item, filename, contentType, data, now)
}
func (r *boundDocumentRepository) Replace(id, uploadedBy string, upload UploadedFile, now time.Time) (DocumentRecord, DocumentRecord, error) {
	return r.storage.replace(r.tenantSlug, id, uploadedBy, upload, now)
}
func (r *boundDocumentRepository) List() []DocumentRecord { return r.storage.listTenant(r.tenantSlug) }
func (r *boundDocumentRepository) ListCurrent() []DocumentRecord {
	return r.storage.listCurrentTenant(r.tenantSlug)
}
func (r *boundDocumentRepository) Versions(seriesID string) []DocumentRecord {
	return r.storage.versions(r.tenantSlug, seriesID)
}
func (r *boundDocumentRepository) Get(id string) (DocumentRecord, bool) {
	return r.storage.get(r.tenantSlug, id)
}
func (r *boundDocumentRepository) FilePath(item DocumentRecord) (string, bool) {
	if textutil.Slug(item.TenantSlug) != r.tenantSlug {
		return "", false
	}
	return r.storage.filePath(item)
}

// SQLDocumentStore keeps each document's metadata as a JSON document keyed by
// (tenant, id); the file itself stays on disk. Table from migration 0011.
//
// Retention (HAUSV-146): superseded versions and their files are kept
// DELIBERATELY and indefinitely. The version history is the feature — a WEG must
// be able to show which Hausordnung was in force when — so there is no pruning
// policy here on purpose. Growth is bounded in practice by how often a document
// is actually replaced, which at WEG scale is a handful of times per document
// per decade. If that ever changes, prune by SeriesID keeping the current
// version plus the N most recent, rather than by age.
type SQLDocumentStore struct {
	db      *sql.DB
	fileDir string
}

func NewSQLDocumentStore(db *sql.DB, fileDir string) *SQLDocumentStore {
	return &SQLDocumentStore{db: db, fileDir: fileDir}
}

func (*SQLDocumentStore) documentStorage() {}

func (s *SQLDocumentStore) writeTx(tx *sql.Tx, item DocumentRecord) error {
	blob, err := json.Marshal(item)
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		`INSERT INTO documents(tenant_slug, id, data) VALUES($1, $2, $3)
		 ON CONFLICT(tenant_slug, id) DO UPDATE SET data=excluded.data`,
		textutil.Slug(item.TenantSlug), item.ID, string(blob),
	)
	return err
}

// insertOne persists a freshly created record, removing the stored file if the
// metadata write fails (mirrors the JSON store's rollback).
func (s *SQLDocumentStore) insertOne(item DocumentRecord, filePath string) (DocumentRecord, error) {
	tx, err := s.db.Begin()
	if err != nil {
		_ = os.Remove(filePath)
		return DocumentRecord{}, err
	}
	defer tx.Rollback()
	if err := s.writeTx(tx, item); err != nil {
		_ = os.Remove(filePath)
		return DocumentRecord{}, err
	}
	if err := tx.Commit(); err != nil {
		_ = os.Remove(filePath)
		return DocumentRecord{}, err
	}
	return CopyDocument(item), nil
}

func (s *SQLDocumentStore) create(tenantSlug string, item DocumentRecord, upload UploadedFile, now time.Time) (DocumentRecord, error) {
	if s == nil {
		return DocumentRecord{}, fmt.Errorf("document store unavailable")
	}
	if now.IsZero() {
		now = time.Now()
	}
	item.TenantSlug = textutil.Slug(tenantSlug)
	item.UploadedAt = now.UTC()
	fileSave, err := saveUploadedDocumentFileIn(s.fileDir, item.TenantSlug, upload)
	if err != nil {
		return DocumentRecord{}, err
	}
	item.ID = fileSave.ID
	item.SeriesID = fileSave.ID
	item.Version = 1
	item.Current = true
	item.Filename = fileSave.Filename
	item.StoredFilename = fileSave.StoredFilename
	item.Size = fileSave.Size
	item.ContentType = fileSave.ContentType
	item = NormalizeDocumentRecord(item)
	if item.TenantSlug == "" || item.Title == "" || item.Category == "" || item.Visibility == "" || item.UploadedBy == "" {
		_ = os.Remove(fileSave.Path)
		return DocumentRecord{}, fmt.Errorf("invalid document metadata")
	}
	return s.insertOne(item, fileSave.Path)
}

func (s *SQLDocumentStore) createGenerated(tenantSlug string, item DocumentRecord, filename string, contentType string, data []byte, now time.Time) (DocumentRecord, error) {
	if s == nil {
		return DocumentRecord{}, fmt.Errorf("document store unavailable")
	}
	item.TenantSlug = textutil.Slug(tenantSlug)
	item, path, err := prepareGeneratedDocument(s.fileDir, item, filename, contentType, data, now)
	if err != nil {
		return DocumentRecord{}, err
	}
	return s.insertOne(item, path)
}

// Replace supersedes the current version and inserts the new one in ONE
// transaction, so a crash can no longer leave both marked current or the old
// one orphaned (HAUSV-145).
func (s *SQLDocumentStore) replace(tenantSlug string, id string, uploadedBy string, upload UploadedFile, now time.Time) (DocumentRecord, DocumentRecord, error) {
	if s == nil {
		return DocumentRecord{}, DocumentRecord{}, fmt.Errorf("document store unavailable")
	}
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	uploadedBy = textutil.Email(uploadedBy)
	if tenantSlug == "" || id == "" || uploadedBy == "" {
		return DocumentRecord{}, DocumentRecord{}, fmt.Errorf("invalid document replacement")
	}
	existing, found := s.get(tenantSlug, id)
	if !found || !existing.Current {
		return DocumentRecord{}, DocumentRecord{}, fmt.Errorf("document not found")
	}
	if now.IsZero() {
		now = time.Now()
	}
	fileSave, err := saveUploadedDocumentFileIn(s.fileDir, tenantSlug, upload)
	if err != nil {
		return DocumentRecord{}, DocumentRecord{}, err
	}

	tx, err := s.db.Begin()
	if err != nil {
		_ = os.Remove(fileSave.Path)
		return DocumentRecord{}, DocumentRecord{}, err
	}
	defer tx.Rollback()

	// Re-read inside the transaction: the row must still be the current version.
	var data string
	if err := tx.QueryRow(`SELECT data FROM documents WHERE tenant_slug=$1 AND id=$2`, tenantSlug, existing.ID).Scan(&data); err != nil {
		_ = os.Remove(fileSave.Path)
		return DocumentRecord{}, DocumentRecord{}, fmt.Errorf("document not current")
	}
	var current DocumentRecord
	if err := json.Unmarshal([]byte(data), &current); err != nil || !current.Current {
		_ = os.Remove(fileSave.Path)
		return DocumentRecord{}, DocumentRecord{}, fmt.Errorf("document not current")
	}

	replacement := current
	replacement.ID = fileSave.ID
	replacement.Version = current.Version + 1
	replacement.Current = true
	replacement.SupersedesID = current.ID
	replacement.ReplacedByID = ""
	replacement.Filename = fileSave.Filename
	replacement.StoredFilename = fileSave.StoredFilename
	replacement.Size = fileSave.Size
	replacement.ContentType = fileSave.ContentType
	replacement.UploadedBy = uploadedBy
	replacement.UploadedAt = now.UTC()
	replacement = NormalizeDocumentRecord(replacement)

	current.Current = false
	current.ReplacedByID = replacement.ID
	replaced := NormalizeDocumentRecord(current)

	if err := s.writeTx(tx, replaced); err != nil {
		_ = os.Remove(fileSave.Path)
		return DocumentRecord{}, DocumentRecord{}, err
	}
	if err := s.writeTx(tx, replacement); err != nil {
		_ = os.Remove(fileSave.Path)
		return DocumentRecord{}, DocumentRecord{}, err
	}
	if err := tx.Commit(); err != nil {
		_ = os.Remove(fileSave.Path)
		return DocumentRecord{}, DocumentRecord{}, err
	}
	return CopyDocument(replacement), CopyDocument(replaced), nil
}

func (s *SQLDocumentStore) allForTenant(tenantSlug string) []DocumentRecord {
	rows, err := s.db.Query(`SELECT data FROM documents WHERE tenant_slug=$1`, tenantSlug)
	if err != nil {
		return []DocumentRecord{}
	}
	defer rows.Close()
	out := []DocumentRecord{}
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			continue
		}
		var item DocumentRecord
		if err := json.Unmarshal([]byte(data), &item); err != nil {
			continue
		}
		out = append(out, item)
	}
	return out
}

func (s *SQLDocumentStore) listTenant(tenantSlug string) []DocumentRecord {
	if s == nil {
		return nil
	}
	out := s.allForTenant(textutil.Slug(tenantSlug))
	SortDocuments(out)
	return out
}

func (s *SQLDocumentStore) listCurrentTenant(tenantSlug string) []DocumentRecord {
	all := s.listTenant(tenantSlug)
	out := []DocumentRecord{}
	for _, item := range all {
		if item.Current {
			out = append(out, item)
		}
	}
	SortDocuments(out)
	return out
}

func (s *SQLDocumentStore) versions(tenantSlug string, seriesID string) []DocumentRecord {
	if s == nil {
		return nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	seriesID = strings.TrimSpace(seriesID)
	if tenantSlug == "" || seriesID == "" {
		return nil
	}
	out := []DocumentRecord{}
	for _, item := range s.allForTenant(tenantSlug) {
		if item.SeriesID == seriesID {
			out = append(out, item)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Version != out[j].Version {
			return out[i].Version > out[j].Version
		}
		return out[i].UploadedAt.After(out[j].UploadedAt)
	})
	return out
}

func (s *SQLDocumentStore) get(tenantSlug string, id string) (DocumentRecord, bool) {
	if s == nil {
		return DocumentRecord{}, false
	}
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	if tenantSlug == "" || id == "" {
		return DocumentRecord{}, false
	}
	var data string
	if err := s.db.QueryRow(`SELECT data FROM documents WHERE tenant_slug=$1 AND id=$2`, tenantSlug, id).Scan(&data); err != nil {
		return DocumentRecord{}, false
	}
	var item DocumentRecord
	if err := json.Unmarshal([]byte(data), &item); err != nil {
		return DocumentRecord{}, false
	}
	return CopyDocument(item), true
}

func (s *SQLDocumentStore) filePath(item DocumentRecord) (string, bool) {
	if s == nil {
		return "", false
	}
	return documentFilePathIn(s.fileDir, item)
}

// ImportDocuments copies metadata records from a JSON store, each only if absent
// (clobber-safe). Files on disk are already shared, so nothing is copied
// (HAUSV-170).
func (s *SQLDocumentStore) ImportDocuments(src *DocumentStore) error {
	if src == nil {
		return nil
	}
	src.mu.Lock()
	snapshot := append([]DocumentRecord(nil), src.data.Documents...)
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
			`INSERT INTO documents(tenant_slug, id, data) VALUES($1, $2, $3) ON CONFLICT(tenant_slug, id) DO NOTHING`,
			textutil.Slug(item.TenantSlug), item.ID, string(blob),
		); err != nil {
			return err
		}
	}
	return nil
}
