package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

type HandoverRepository interface {
	Create(item HandoverRecord) (HandoverRecord, error)
	List() []HandoverRecord
	Get(id string) (HandoverRecord, bool)
	SetFiledDocument(id string, documentID string, at time.Time) (HandoverRecord, bool, error)
}

// HandoverStorage retains only the deliberately global confirmation-token
// operations. Tenant-scoped access is available only through a bound repository.
type HandoverStorage interface {
	handoverStorage()
	GetByToken(token string) (HandoverRecord, int, bool)
	ConfirmByToken(token string, name string, note string, at time.Time) (HandoverRecord, HandoverConfirmation, bool, error)
}

var (
	_ HandoverStorage = (*HandoverStore)(nil)
	_ HandoverStorage = (*SQLHandoverStore)(nil)
)

type boundHandoverRepository struct {
	storage handoverBackend
	tenant  TenantRef
}

type handoverBackend interface {
	create(tenant TenantRef, item HandoverRecord) (HandoverRecord, error)
	list(tenant TenantRef) []HandoverRecord
	get(tenant TenantRef, id string) (HandoverRecord, bool)
	setFiledDocument(tenant TenantRef, id string, documentID string, at time.Time) (HandoverRecord, bool, error)
}

func BindHandoverRepository(storage HandoverStorage, tenant TenantRef) (HandoverRepository, bool) {
	resolvedTenant, tenantOK := validTenantRef(tenant)
	backend, ok := storage.(handoverBackend)
	if !ok || !tenantOK {
		return nil, false
	}
	return &boundHandoverRepository{storage: backend, tenant: resolvedTenant}, true
}

func (r *boundHandoverRepository) Create(item HandoverRecord) (HandoverRecord, error) {
	return r.storage.create(r.tenant, item)
}
func (r *boundHandoverRepository) List() []HandoverRecord { return r.storage.list(r.tenant) }
func (r *boundHandoverRepository) Get(id string) (HandoverRecord, bool) {
	return r.storage.get(r.tenant, id)
}
func (r *boundHandoverRepository) SetFiledDocument(id string, documentID string, at time.Time) (HandoverRecord, bool, error) {
	return r.storage.setFiledDocument(r.tenant, id, documentID, at)
}

// SQLHandoverStore keeps each handover as a JSON document keyed by (tenant, id).
// Table from migration 0010. Foundation for the atomic PDF-filing flow once the
// document store also lives in SQLite (HAUSV-145/148).
type SQLHandoverStore struct {
	db *TenantDB
}

func NewSQLHandoverStore(db *TenantDB) *SQLHandoverStore {
	return &SQLHandoverStore{db: db}
}

func (*SQLHandoverStore) handoverStorage() {}

func (s *SQLHandoverStore) writeTx(tx *sql.Tx, tenant TenantRef, item HandoverRecord) error {
	blob, err := json.Marshal(item)
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		`INSERT INTO handovers(tenant_id, tenant_slug, id, data) VALUES($1, $2, $3, $4)
		 ON CONFLICT(tenant_slug, id) DO UPDATE SET data=excluded.data,
		   tenant_id=coalesce(handovers.tenant_id, excluded.tenant_id)`,
		tenant.ID, tenant.Slug, item.ID, string(blob),
	)
	return err
}

func (s *SQLHandoverStore) create(tenant TenantRef, item HandoverRecord) (HandoverRecord, error) {
	tenantSlug := tenant.Slug
	if s == nil {
		return HandoverRecord{}, fmt.Errorf("handover store unavailable")
	}
	item.TenantSlug = tenantSlug
	item = NormalizeHandover(item)
	if item.ID == "" || item.TenantSlug == "" || item.Title == "" || item.CreatedBy == "" {
		return HandoverRecord{}, fmt.Errorf("invalid handover")
	}
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return HandoverRecord{}, err
	}
	defer tx.Rollback()
	var exists int
	if err := tx.QueryRow(
		`SELECT 1 FROM handovers WHERE tenant_id=$1 AND id=$2`, tenant.ID, item.ID,
	).Scan(&exists); err == nil {
		return HandoverRecord{}, fmt.Errorf("handover exists")
	} else if err != sql.ErrNoRows {
		return HandoverRecord{}, err
	}
	if err := s.writeTx(tx, tenant, item); err != nil {
		return HandoverRecord{}, err
	}
	if err := tx.Commit(); err != nil {
		return HandoverRecord{}, err
	}
	return CopyHandover(item), nil
}

func (s *SQLHandoverStore) list(tenant TenantRef) []HandoverRecord {
	tenantSlug := tenant.Slug
	if s == nil {
		return nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	rows, err := s.db.For(tenant).Query(`SELECT data FROM handovers WHERE tenant_id=$1`, tenant.ID)
	if err != nil {
		return []HandoverRecord{}
	}
	defer rows.Close()
	out := []HandoverRecord{}
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			continue
		}
		var item HandoverRecord
		if err := json.Unmarshal([]byte(data), &item); err != nil {
			continue
		}
		out = append(out, CopyHandover(item))
	}
	SortHandovers(out)
	return out
}

func (s *SQLHandoverStore) get(tenant TenantRef, id string) (HandoverRecord, bool) {
	tenantSlug := tenant.Slug
	if s == nil {
		return HandoverRecord{}, false
	}
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	var data string
	if err := s.db.For(tenant).QueryRow(
		`SELECT data FROM handovers WHERE tenant_id=$1 AND id=$2`, tenant.ID, id,
	).Scan(&data); err != nil {
		return HandoverRecord{}, false
	}
	var item HandoverRecord
	if err := json.Unmarshal([]byte(data), &item); err != nil {
		return HandoverRecord{}, false
	}
	return CopyHandover(item), true
}

// allRecords loads every handover across all tenants for global token lookups.
func (s *SQLHandoverStore) allRecords(q interface {
	Query(string, ...any) (*sql.Rows, error)
}) []HandoverRecord {
	rows, err := q.Query(`SELECT data FROM handovers`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []HandoverRecord{}
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			continue
		}
		var item HandoverRecord
		if err := json.Unmarshal([]byte(data), &item); err != nil {
			continue
		}
		out = append(out, item)
	}
	return out
}

func (s *SQLHandoverStore) GetByToken(token string) (HandoverRecord, int, bool) {
	if s == nil || strings.TrimSpace(token) == "" {
		return HandoverRecord{}, -1, false
	}
	hash := HandoverTokenHash(token)
	byToken := s.db.Unscoped("handover confirmation link: the token is the only identifier the recipient has, and it names no tenant")
	for _, item := range s.allRecords(byToken) {
		for idx, confirmation := range item.Confirmations {
			if confirmation.TokenHash != "" && SubtleConstantStringCompare(confirmation.TokenHash, hash) {
				return CopyHandover(item), idx, true
			}
		}
	}
	return HandoverRecord{}, -1, false
}

func (s *SQLHandoverStore) ConfirmByToken(token string, name string, note string, at time.Time) (HandoverRecord, HandoverConfirmation, bool, error) {
	if s == nil || strings.TrimSpace(token) == "" {
		return HandoverRecord{}, HandoverConfirmation{}, false, nil
	}
	hash := HandoverTokenHash(token)
	name = textutil.Truncate(strings.TrimSpace(name), 120)
	note = textutil.Truncate(strings.TrimSpace(note), 500)
	if at.IsZero() {
		at = time.Now()
	}
	byToken := s.db.Unscoped("handover confirmation link: the token is the only identifier the recipient has, and it names no tenant")
	tx, err := byToken.Begin()
	if err != nil {
		return HandoverRecord{}, HandoverConfirmation{}, false, err
	}
	defer tx.Rollback()
	for _, item := range s.allRecords(tx) {
		for j, confirmation := range item.Confirmations {
			if confirmation.TokenHash == "" || !SubtleConstantStringCompare(confirmation.TokenHash, hash) {
				continue
			}
			if !confirmation.ConfirmedAt.IsZero() {
				return CopyHandover(item), confirmation, true, nil
			}
			confirmation.ConfirmedAt = at.UTC()
			confirmation.ConfirmedBy = textutil.FirstNonEmpty(name, confirmation.Name, confirmation.Email)
			confirmation.Note = note
			item.Confirmations[j] = confirmation
			item.UpdatedAt = at.UTC()
			saved := NormalizeHandover(item)
			// Token confirmation is deliberately cross-tenant: the token is the
			// only thing the confirming party has. The row's own identity is the
			// tenant here, not a bound one.
			confirmedTenant, refErr := tenantRefFor(tx, saved.TenantSlug)
			if refErr != nil {
				return HandoverRecord{}, HandoverConfirmation{}, true, refErr
			}
			if err := s.writeTx(tx, confirmedTenant, saved); err != nil {
				return HandoverRecord{}, HandoverConfirmation{}, true, err
			}
			if err := tx.Commit(); err != nil {
				return HandoverRecord{}, HandoverConfirmation{}, true, err
			}
			return CopyHandover(saved), confirmation, true, nil
		}
	}
	return HandoverRecord{}, HandoverConfirmation{}, false, nil
}

func (s *SQLHandoverStore) setFiledDocument(tenant TenantRef, id string, documentID string, at time.Time) (HandoverRecord, bool, error) {
	tenantSlug := tenant.Slug
	if s == nil {
		return HandoverRecord{}, false, fmt.Errorf("handover store unavailable")
	}
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	documentID = strings.TrimSpace(documentID)
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return HandoverRecord{}, false, err
	}
	defer tx.Rollback()
	var data string
	if err := tx.QueryRow(`SELECT data FROM handovers WHERE tenant_id=$1 AND id=$2`, tenant.ID, id).Scan(&data); err != nil {
		return HandoverRecord{}, false, nil
	}
	var item HandoverRecord
	if err := json.Unmarshal([]byte(data), &item); err != nil {
		return HandoverRecord{}, false, nil
	}
	item.FiledDocumentID = documentID
	if at.IsZero() {
		at = time.Now()
	}
	item.UpdatedAt = at.UTC()
	saved := NormalizeHandover(item)
	if err := s.writeTx(tx, tenant, saved); err != nil {
		return HandoverRecord{}, true, err
	}
	if err := tx.Commit(); err != nil {
		return HandoverRecord{}, true, err
	}
	return CopyHandover(saved), true, nil
}

// ImportHandovers copies records from a JSON store, each only if absent
// (clobber-safe) (HAUSV-170).
func (s *SQLHandoverStore) ImportHandovers(src *HandoverStore) error {
	if src == nil {
		return nil
	}
	src.mu.Lock()
	snapshot := append([]HandoverRecord(nil), src.data.Handovers...)
	src.mu.Unlock()
	imports := s.db.Unscoped("boot import replay of the JSON handover snapshot: it spans every tenant and runs before the first request")
	tenants := newTenantIDCache(imports)
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
		if _, err := imports.Exec(
			`INSERT INTO handovers(tenant_id, tenant_slug, id, data) VALUES($1, $2, $3, $4) ON CONFLICT(tenant_slug, id) DO NOTHING`,
			tenant.ID, tenant.Slug, item.ID, string(blob),
		); err != nil {
			return err
		}
	}
	return nil
}
