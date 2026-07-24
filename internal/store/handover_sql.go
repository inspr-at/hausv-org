package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/markus-barta/hausv-org/internal/textutil"
)

// HandoverStorage is the behaviour both the JSON HandoverStore and the SQLite
// SQLHandoverStore satisfy (HAUSV-168). Confirmation tokens are looked up
// globally (across tenants), matching the JSON store.
type HandoverStorage interface {
	Create(item HandoverRecord) (HandoverRecord, error)
	ListTenant(tenantSlug string) []HandoverRecord
	Get(tenantSlug string, id string) (HandoverRecord, bool)
	GetByToken(token string) (HandoverRecord, int, bool)
	ConfirmByToken(token string, name string, note string, at time.Time) (HandoverRecord, HandoverConfirmation, bool, error)
	SetFiledDocument(tenantSlug string, id string, documentID string, at time.Time) (HandoverRecord, bool, error)
}

var (
	_ HandoverStorage = (*HandoverStore)(nil)
	_ HandoverStorage = (*SQLHandoverStore)(nil)
)

// SQLHandoverStore keeps each handover as a JSON document keyed by (tenant, id).
// Table from migration 0010. Foundation for the atomic PDF-filing flow once the
// document store also lives in SQLite (HAUSV-145/148).
type SQLHandoverStore struct {
	db *sql.DB
}

func NewSQLHandoverStore(db *sql.DB) *SQLHandoverStore {
	return &SQLHandoverStore{db: db}
}

func (s *SQLHandoverStore) writeTx(tx *sql.Tx, item HandoverRecord) error {
	blob, err := json.Marshal(item)
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		`INSERT INTO handovers(tenant_slug, id, data) VALUES(?, ?, ?)
		 ON CONFLICT(tenant_slug, id) DO UPDATE SET data=excluded.data`,
		textutil.Slug(item.TenantSlug), item.ID, string(blob),
	)
	return err
}

func (s *SQLHandoverStore) Create(item HandoverRecord) (HandoverRecord, error) {
	if s == nil {
		return HandoverRecord{}, fmt.Errorf("handover store unavailable")
	}
	item = NormalizeHandover(item)
	if item.ID == "" || item.TenantSlug == "" || item.Title == "" || item.CreatedBy == "" {
		return HandoverRecord{}, fmt.Errorf("invalid handover")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return HandoverRecord{}, err
	}
	defer tx.Rollback()
	var exists int
	if err := tx.QueryRow(
		`SELECT 1 FROM handovers WHERE tenant_slug=? AND id=?`, item.TenantSlug, item.ID,
	).Scan(&exists); err == nil {
		return HandoverRecord{}, fmt.Errorf("handover exists")
	} else if err != sql.ErrNoRows {
		return HandoverRecord{}, err
	}
	if err := s.writeTx(tx, item); err != nil {
		return HandoverRecord{}, err
	}
	if err := tx.Commit(); err != nil {
		return HandoverRecord{}, err
	}
	return CopyHandover(item), nil
}

func (s *SQLHandoverStore) ListTenant(tenantSlug string) []HandoverRecord {
	if s == nil {
		return nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	rows, err := s.db.Query(`SELECT data FROM handovers WHERE tenant_slug=?`, tenantSlug)
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

func (s *SQLHandoverStore) Get(tenantSlug string, id string) (HandoverRecord, bool) {
	if s == nil {
		return HandoverRecord{}, false
	}
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	var data string
	if err := s.db.QueryRow(
		`SELECT data FROM handovers WHERE tenant_slug=? AND id=?`, tenantSlug, id,
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
	for _, item := range s.allRecords(s.db) {
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
	tx, err := s.db.Begin()
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
			if err := s.writeTx(tx, saved); err != nil {
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

func (s *SQLHandoverStore) SetFiledDocument(tenantSlug string, id string, documentID string, at time.Time) (HandoverRecord, bool, error) {
	if s == nil {
		return HandoverRecord{}, false, fmt.Errorf("handover store unavailable")
	}
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	documentID = strings.TrimSpace(documentID)
	tx, err := s.db.Begin()
	if err != nil {
		return HandoverRecord{}, false, err
	}
	defer tx.Rollback()
	var data string
	if err := tx.QueryRow(`SELECT data FROM handovers WHERE tenant_slug=? AND id=?`, tenantSlug, id).Scan(&data); err != nil {
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
	if err := s.writeTx(tx, saved); err != nil {
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
	for _, item := range snapshot {
		if strings.TrimSpace(item.ID) == "" || textutil.Slug(item.TenantSlug) == "" {
			continue
		}
		blob, err := json.Marshal(item)
		if err != nil {
			return err
		}
		if _, err := s.db.Exec(
			`INSERT INTO handovers(tenant_slug, id, data) VALUES(?, ?, ?) ON CONFLICT(tenant_slug, id) DO NOTHING`,
			textutil.Slug(item.TenantSlug), item.ID, string(blob),
		); err != nil {
			return err
		}
	}
	return nil
}
