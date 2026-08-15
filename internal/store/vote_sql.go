package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

// VoteRepository is a vote store already bound to one tenant.
type VoteRepository interface {
	Create(item Ballot) (Ballot, error)
	Delete(id string) (bool, error)
	Open(id string, at time.Time) (Ballot, bool, error)
	Close(id string, at time.Time) (Ballot, bool, error)
	CloseExpired(at time.Time) ([]Ballot, error)
	CastVote(id string, email string, option string, weight int, at time.Time) (Ballot, bool, error)
	MarkReminderSent(id string, recipients []string, at time.Time) (Ballot, bool, error)
	List() []Ballot
	Get(id string) (Ballot, bool)
}

// VoteStorage is the unbound backend implemented by the JSON and SQLite
// stores. HTTP code receives only VoteRepository.
type VoteStorage interface {
	voteStorage()
}

var (
	_ VoteStorage = (*VoteStore)(nil)
	_ VoteStorage = (*SQLVoteStore)(nil)
)

type boundVoteRepository struct {
	storage    voteBackend
	tenantSlug string
}

type voteBackend interface {
	create(tenantSlug string, item Ballot) (Ballot, error)
	delete(tenantSlug string, id string) (bool, error)
	open(tenantSlug string, id string, at time.Time) (Ballot, bool, error)
	close(tenantSlug string, id string, at time.Time) (Ballot, bool, error)
	closeExpiredTenant(tenantSlug string, at time.Time) ([]Ballot, error)
	castVote(tenantSlug string, id string, email string, option string, weight int, at time.Time) (Ballot, bool, error)
	markReminderSent(tenantSlug string, id string, recipients []string, at time.Time) (Ballot, bool, error)
	listTenant(tenantSlug string) []Ballot
	get(tenantSlug string, id string) (Ballot, bool)
}

// BindVoteRepository binds all vote operations to one tenant.
func BindVoteRepository(storage VoteStorage, tenantSlug string) (VoteRepository, bool) {
	tenantSlug = textutil.Slug(tenantSlug)
	backend, ok := storage.(voteBackend)
	if !ok || tenantSlug == "" {
		return nil, false
	}
	return &boundVoteRepository{storage: backend, tenantSlug: tenantSlug}, true
}

func (r *boundVoteRepository) Create(item Ballot) (Ballot, error) {
	return r.storage.create(r.tenantSlug, item)
}

func (r *boundVoteRepository) Delete(id string) (bool, error) {
	return r.storage.delete(r.tenantSlug, id)
}

func (r *boundVoteRepository) Open(id string, at time.Time) (Ballot, bool, error) {
	return r.storage.open(r.tenantSlug, id, at)
}

func (r *boundVoteRepository) Close(id string, at time.Time) (Ballot, bool, error) {
	return r.storage.close(r.tenantSlug, id, at)
}

func (r *boundVoteRepository) CloseExpired(at time.Time) ([]Ballot, error) {
	return r.storage.closeExpiredTenant(r.tenantSlug, at)
}

func (r *boundVoteRepository) CastVote(id string, email string, option string, weight int, at time.Time) (Ballot, bool, error) {
	return r.storage.castVote(r.tenantSlug, id, email, option, weight, at)
}

func (r *boundVoteRepository) MarkReminderSent(id string, recipients []string, at time.Time) (Ballot, bool, error) {
	return r.storage.markReminderSent(r.tenantSlug, id, recipients, at)
}

func (r *boundVoteRepository) List() []Ballot {
	return r.storage.listTenant(r.tenantSlug)
}

func (r *boundVoteRepository) Get(id string) (Ballot, bool) {
	return r.storage.get(r.tenantSlug, id)
}

// SQLVoteStore keeps each ballot — votes included — as one JSON document keyed
// by (tenant, id). Table from migration 0015.
type SQLVoteStore struct {
	db *sql.DB
}

func NewSQLVoteStore(db *sql.DB) *SQLVoteStore {
	return &SQLVoteStore{db: db}
}

func (*SQLVoteStore) voteStorage() {}

func (s *SQLVoteStore) writeTx(tx *sql.Tx, item Ballot) error {
	blob, err := json.Marshal(item)
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		`INSERT INTO ballots(tenant_slug, id, data) VALUES(?, ?, ?)
		 ON CONFLICT(tenant_slug, id) DO UPDATE SET data=excluded.data`,
		textutil.Slug(item.TenantSlug), item.ID, string(blob),
	)
	return err
}

// loadTx reads one ballot inside a transaction for a read-modify-write.
func loadBallotTx(tx *sql.Tx, tenantSlug string, id string) (Ballot, bool) {
	var data string
	if err := tx.QueryRow(`SELECT data FROM ballots WHERE tenant_slug=? AND id=?`, tenantSlug, id).Scan(&data); err != nil {
		return Ballot{}, false
	}
	var item Ballot
	if err := json.Unmarshal([]byte(data), &item); err != nil {
		return Ballot{}, false
	}
	return item, true
}

func (s *SQLVoteStore) create(tenantSlug string, item Ballot) (Ballot, error) {
	if s == nil {
		return Ballot{}, fmt.Errorf("vote store unavailable")
	}
	now := time.Now().UTC()
	id, err := randomToken(12)
	if err != nil {
		return Ballot{}, err
	}
	item.TenantSlug = tenantSlug
	item.ID = id
	item.Status = BallotStatusDraft
	item.CreatedAt = now
	item.UpdatedAt = now
	item.Votes = nil
	item = NormalizeBallot(item)
	if item.TenantSlug == "" || item.Title == "" || len(item.Options) < 2 || item.Type == "" || item.Weighting == "" || item.CreatedBy == "" {
		return Ballot{}, fmt.Errorf("invalid Ballot")
	}
	if !item.OpensAt.IsZero() && !item.ClosesAt.IsZero() && !item.ClosesAt.After(item.OpensAt) {
		return Ballot{}, fmt.Errorf("Ballot close must be after open")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return Ballot{}, err
	}
	defer tx.Rollback()
	if err := s.writeTx(tx, item); err != nil {
		return Ballot{}, err
	}
	if err := tx.Commit(); err != nil {
		return Ballot{}, err
	}
	return CopyBallot(item), nil
}

func (s *SQLVoteStore) delete(tenantSlug string, id string) (bool, error) {
	if s == nil {
		return false, nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	if tenantSlug == "" || id == "" {
		return false, nil
	}
	res, err := s.db.Exec(`DELETE FROM ballots WHERE tenant_slug=? AND id=?`, tenantSlug, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (s *SQLVoteStore) open(tenantSlug string, id string, at time.Time) (Ballot, bool, error) {
	return s.setStatus(tenantSlug, id, BallotStatusOpen, at)
}

func (s *SQLVoteStore) close(tenantSlug string, id string, at time.Time) (Ballot, bool, error) {
	return s.setStatus(tenantSlug, id, BallotStatusClosed, at)
}

func (s *SQLVoteStore) setStatus(tenantSlug string, id string, status string, at time.Time) (Ballot, bool, error) {
	if s == nil {
		return Ballot{}, false, nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	status = NormalizeBallotStatus(status)
	if tenantSlug == "" || id == "" || status == "" {
		return Ballot{}, false, fmt.Errorf("invalid Ballot status")
	}
	if at.IsZero() {
		at = time.Now()
	}
	at = at.UTC()
	tx, err := s.db.Begin()
	if err != nil {
		return Ballot{}, false, err
	}
	defer tx.Rollback()
	item, found := loadBallotTx(tx, tenantSlug, id)
	if !found {
		return Ballot{}, false, nil
	}
	if item.Status == BallotStatusClosed && status == BallotStatusOpen {
		return Ballot{}, true, fmt.Errorf("closed Ballot cannot reopen")
	}
	item.Status = status
	if status == BallotStatusOpen && item.OpensAt.IsZero() {
		item.OpensAt = at
	}
	if status == BallotStatusClosed && (item.ClosesAt.IsZero() || item.ClosesAt.After(at)) {
		item.ClosesAt = at
	}
	item.UpdatedAt = at
	item = NormalizeBallot(item)
	if err := s.writeTx(tx, item); err != nil {
		return Ballot{}, true, err
	}
	if err := tx.Commit(); err != nil {
		return Ballot{}, true, err
	}
	return CopyBallot(item), true, nil
}

func (s *SQLVoteStore) closeExpiredTenant(tenantSlug string, at time.Time) ([]Ballot, error) {
	if s == nil {
		return nil, nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	if tenantSlug == "" {
		return nil, fmt.Errorf("invalid tenant")
	}
	if at.IsZero() {
		at = time.Now()
	}
	at = at.UTC()
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.Query(`SELECT data FROM ballots WHERE tenant_slug=?`, tenantSlug)
	if err != nil {
		return nil, err
	}
	pending := []Ballot{}
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			continue
		}
		var item Ballot
		if err := json.Unmarshal([]byte(data), &item); err != nil {
			continue
		}
		pending = append(pending, item)
	}
	rows.Close()

	closed := []Ballot{}
	for _, item := range pending {
		item = NormalizeBallot(item)
		if item.Status != BallotStatusOpen || item.ClosesAt.IsZero() || at.Before(item.ClosesAt) {
			continue
		}
		item.Status = BallotStatusClosed
		item.UpdatedAt = at
		item = NormalizeBallot(item)
		if err := s.writeTx(tx, item); err != nil {
			return nil, err
		}
		closed = append(closed, CopyBallot(item))
	}
	if len(closed) == 0 {
		return nil, nil
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	SortBallots(closed)
	return closed, nil
}

func (s *SQLVoteStore) castVote(tenantSlug string, id string, email string, option string, weight int, at time.Time) (Ballot, bool, error) {
	if s == nil {
		return Ballot{}, false, nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	email = textutil.Email(email)
	option = strings.TrimSpace(option)
	if tenantSlug == "" || id == "" || email == "" || option == "" || weight <= 0 {
		return Ballot{}, false, fmt.Errorf("invalid vote")
	}
	if at.IsZero() {
		at = time.Now()
	}
	at = at.UTC()
	tx, err := s.db.Begin()
	if err != nil {
		return Ballot{}, false, err
	}
	defer tx.Rollback()
	item, found := loadBallotTx(tx, tenantSlug, id)
	if !found {
		return Ballot{}, false, nil
	}
	item = NormalizeBallot(item)
	if item.Status != BallotStatusOpen {
		return Ballot{}, true, fmt.Errorf("Ballot is not open")
	}
	if !item.OpensAt.IsZero() && at.Before(item.OpensAt) {
		return Ballot{}, true, fmt.Errorf("Ballot is not open yet")
	}
	// Past its closing time: persist the closure, then report it. The write must
	// survive, so this branch commits before returning the error.
	if !item.ClosesAt.IsZero() && !at.Before(item.ClosesAt) {
		item.Status = BallotStatusClosed
		item.UpdatedAt = at
		item = NormalizeBallot(item)
		if err := s.writeTx(tx, item); err != nil {
			return Ballot{}, true, err
		}
		if err := tx.Commit(); err != nil {
			return Ballot{}, true, err
		}
		return Ballot{}, true, fmt.Errorf("Ballot is closed")
	}
	if !BallotHasOption(item, option) {
		return Ballot{}, true, fmt.Errorf("invalid vote option")
	}
	if item.Votes == nil {
		item.Votes = map[string]BallotVote{}
	}
	item.Votes[email] = BallotVote{Option: option, Weight: weight, At: at}
	item.UpdatedAt = at
	item = NormalizeBallot(item)
	if err := s.writeTx(tx, item); err != nil {
		return Ballot{}, true, err
	}
	if err := tx.Commit(); err != nil {
		return Ballot{}, true, err
	}
	return CopyBallot(item), true, nil
}

func (s *SQLVoteStore) markReminderSent(tenantSlug string, id string, recipients []string, at time.Time) (Ballot, bool, error) {
	if s == nil {
		return Ballot{}, false, nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	recipients = NormalizeEmailList(recipients)
	if tenantSlug == "" || id == "" || len(recipients) == 0 {
		return Ballot{}, false, fmt.Errorf("invalid reminder")
	}
	if at.IsZero() {
		at = time.Now()
	}
	at = at.UTC()
	tx, err := s.db.Begin()
	if err != nil {
		return Ballot{}, false, err
	}
	defer tx.Rollback()
	item, found := loadBallotTx(tx, tenantSlug, id)
	if !found {
		return Ballot{}, false, nil
	}
	item = NormalizeBallot(item)
	if item.ReminderSentAt == nil {
		item.ReminderSentAt = map[string]time.Time{}
	}
	for _, recipient := range recipients {
		item.ReminderSentAt[recipient] = at
	}
	item.UpdatedAt = at
	item = NormalizeBallot(item)
	if err := s.writeTx(tx, item); err != nil {
		return Ballot{}, true, err
	}
	if err := tx.Commit(); err != nil {
		return Ballot{}, true, err
	}
	return CopyBallot(item), true, nil
}

func (s *SQLVoteStore) listTenant(tenantSlug string) []Ballot {
	if s == nil {
		return nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	rows, err := s.db.Query(`SELECT data FROM ballots WHERE tenant_slug=?`, tenantSlug)
	if err != nil {
		return []Ballot{}
	}
	defer rows.Close()
	out := []Ballot{}
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			continue
		}
		var item Ballot
		if err := json.Unmarshal([]byte(data), &item); err != nil {
			continue
		}
		out = append(out, CopyBallot(item))
	}
	SortBallots(out)
	return out
}

func (s *SQLVoteStore) get(tenantSlug string, id string) (Ballot, bool) {
	if s == nil {
		return Ballot{}, false
	}
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	if tenantSlug == "" || id == "" {
		return Ballot{}, false
	}
	var data string
	if err := s.db.QueryRow(`SELECT data FROM ballots WHERE tenant_slug=? AND id=?`, tenantSlug, id).Scan(&data); err != nil {
		return Ballot{}, false
	}
	var item Ballot
	if err := json.Unmarshal([]byte(data), &item); err != nil {
		return Ballot{}, false
	}
	return CopyBallot(NormalizeBallot(item)), true
}

// ImportBallots copies records from a JSON store, each only if absent
// (clobber-safe) (HAUSV-170).
func (s *SQLVoteStore) ImportBallots(src *VoteStore) error {
	if src == nil {
		return nil
	}
	src.mu.Lock()
	snapshot := append([]Ballot(nil), src.data.Ballots...)
	src.mu.Unlock()
	for _, item := range snapshot {
		tenant := textutil.Slug(item.TenantSlug)
		if tenant == "" || strings.TrimSpace(item.ID) == "" {
			continue
		}
		blob, err := json.Marshal(item)
		if err != nil {
			return err
		}
		if _, err := s.db.Exec(
			`INSERT INTO ballots(tenant_slug, id, data) VALUES(?, ?, ?) ON CONFLICT(tenant_slug, id) DO NOTHING`,
			tenant, item.ID, string(blob),
		); err != nil {
			return err
		}
	}
	return nil
}
