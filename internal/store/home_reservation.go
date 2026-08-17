package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

const (
	HomeReservationEmailPending   = "email-pending"
	HomeReservationEmailConfirmed = "email-confirmed"
	HomeReservationActive         = "active"
)

var ErrHomeReservationConflict = errors.New("home path already reserved")

type HomeReservation struct {
	Slug                   string
	HouseholdName          string
	OwnerEmail             string
	AuthorizationConfirmed bool
	Status                 string
	CreatedAt              time.Time
	UpdatedAt              time.Time
	ConfirmedAt            *time.Time
}

type HomeReservationStorage interface {
	Reserve(HomeReservation, time.Time) (HomeReservation, error)
	Confirm(slug, ownerEmail string, at time.Time) (HomeReservation, bool, error)
	Get(slug string) (HomeReservation, bool, error)
	PurgePendingBefore(time.Time) (int64, error)
}

type MemoryHomeReservationStore struct {
	mu    sync.Mutex
	items map[string]HomeReservation
}

func NewMemoryHomeReservationStore() *MemoryHomeReservationStore {
	return &MemoryHomeReservationStore{items: map[string]HomeReservation{}}
}

func (s *MemoryHomeReservationStore) Reserve(item HomeReservation, at time.Time) (HomeReservation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item = normalizeHomeReservation(item, at)
	if item.Slug == "" || item.HouseholdName == "" || item.OwnerEmail == "" || !item.AuthorizationConfirmed {
		return HomeReservation{}, fmt.Errorf("invalid home reservation")
	}
	if existing, ok := s.items[item.Slug]; ok {
		if existing.OwnerEmail != item.OwnerEmail {
			return HomeReservation{}, ErrHomeReservationConflict
		}
		existing.HouseholdName = item.HouseholdName
		existing.AuthorizationConfirmed = true
		existing.UpdatedAt = item.UpdatedAt
		s.items[item.Slug] = existing
		return existing, nil
	}
	s.items[item.Slug] = item
	return item, nil
}

func (s *MemoryHomeReservationStore) Confirm(slug, ownerEmail string, at time.Time) (HomeReservation, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	slug = textutil.Slug(slug)
	ownerEmail = textutil.Email(ownerEmail)
	item, ok := s.items[slug]
	if !ok || item.OwnerEmail != ownerEmail {
		return HomeReservation{}, false, nil
	}
	at = homeReservationTime(at)
	if item.Status != HomeReservationActive {
		item.Status = HomeReservationEmailConfirmed
	}
	item.UpdatedAt = at
	item.ConfirmedAt = &at
	s.items[slug] = item
	return item, true, nil
}

func (s *MemoryHomeReservationStore) Get(slug string) (HomeReservation, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[textutil.Slug(slug)]
	return item, ok, nil
}

func (s *MemoryHomeReservationStore) markActive(slug, ownerEmail string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[textutil.Slug(slug)]
	if !ok || item.OwnerEmail != textutil.Email(ownerEmail) ||
		(item.Status != HomeReservationEmailConfirmed && item.Status != HomeReservationActive) {
		return ErrHomePortalActivationDenied
	}
	item.Status = HomeReservationActive
	item.UpdatedAt = homeReservationTime(at)
	s.items[item.Slug] = item
	return nil
}

func (s *MemoryHomeReservationStore) PurgePendingBefore(before time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	before = homeReservationTime(before)
	var removed int64
	for slug, item := range s.items {
		if item.Status == HomeReservationEmailPending && item.UpdatedAt.Before(before) {
			delete(s.items, slug)
			removed++
		}
	}
	return removed, nil
}

type SQLHomeReservationStore struct {
	db *sql.DB
}

func NewSQLHomeReservationStore(db *sql.DB) *SQLHomeReservationStore {
	return &SQLHomeReservationStore{db: db}
}

func (s *SQLHomeReservationStore) Reserve(item HomeReservation, at time.Time) (HomeReservation, error) {
	if s == nil || s.db == nil {
		return HomeReservation{}, fmt.Errorf("home reservation store unavailable")
	}
	item = normalizeHomeReservation(item, at)
	if item.Slug == "" || item.HouseholdName == "" || item.OwnerEmail == "" || !item.AuthorizationConfirmed {
		return HomeReservation{}, fmt.Errorf("invalid home reservation")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return HomeReservation{}, err
	}
	defer tx.Rollback()
	existing, found, err := getHomeReservation(tx.QueryRow, item.Slug)
	if err != nil {
		return HomeReservation{}, err
	}
	if found {
		if existing.OwnerEmail != item.OwnerEmail {
			return HomeReservation{}, ErrHomeReservationConflict
		}
		if _, err := tx.Exec(`UPDATE home_reservations
			SET household_name=$1, authorization_confirmed=1, updated_at=$2 WHERE slug=$3`,
			item.HouseholdName, homeReservationTimestamp(item.UpdatedAt), item.Slug); err != nil {
			return HomeReservation{}, err
		}
		if err := tx.Commit(); err != nil {
			return HomeReservation{}, err
		}
		existing.HouseholdName = item.HouseholdName
		existing.AuthorizationConfirmed = true
		existing.UpdatedAt = item.UpdatedAt
		return existing, nil
	}
	if _, err := tx.Exec(`INSERT INTO home_reservations
		(slug,household_name,owner_email,authorization_confirmed,status,created_at,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7)`, item.Slug, item.HouseholdName, item.OwnerEmail, 1, item.Status,
		homeReservationTimestamp(item.CreatedAt), homeReservationTimestamp(item.UpdatedAt)); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return HomeReservation{}, ErrHomeReservationConflict
		}
		return HomeReservation{}, err
	}
	if err := tx.Commit(); err != nil {
		return HomeReservation{}, err
	}
	return item, nil
}

func (s *SQLHomeReservationStore) Confirm(slug, ownerEmail string, at time.Time) (HomeReservation, bool, error) {
	if s == nil || s.db == nil {
		return HomeReservation{}, false, fmt.Errorf("home reservation store unavailable")
	}
	slug = textutil.Slug(slug)
	ownerEmail = textutil.Email(ownerEmail)
	at = homeReservationTime(at)
	result, err := s.db.Exec(`UPDATE home_reservations
		SET status=CASE WHEN status=$1 THEN status ELSE $2 END, confirmed_at=COALESCE(confirmed_at,$3), updated_at=$4
		WHERE slug=$5 AND owner_email=$6`, HomeReservationActive, HomeReservationEmailConfirmed,
		homeReservationTimestamp(at), homeReservationTimestamp(at), slug, ownerEmail)
	if err != nil {
		return HomeReservation{}, false, err
	}
	count, err := result.RowsAffected()
	if err != nil || count == 0 {
		return HomeReservation{}, false, err
	}
	item, found, err := s.Get(slug)
	return item, found, err
}

func (s *SQLHomeReservationStore) Get(slug string) (HomeReservation, bool, error) {
	if s == nil || s.db == nil {
		return HomeReservation{}, false, fmt.Errorf("home reservation store unavailable")
	}
	return getHomeReservation(s.db.QueryRow, textutil.Slug(slug))
}

func (s *SQLHomeReservationStore) PurgePendingBefore(before time.Time) (int64, error) {
	if s == nil || s.db == nil {
		return 0, fmt.Errorf("home reservation store unavailable")
	}
	result, err := s.db.Exec(`DELETE FROM home_reservations WHERE status=$1 AND updated_at<$2`,
		HomeReservationEmailPending, homeReservationTimestamp(before))
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

type homeReservationQueryRow func(string, ...any) *sql.Row

func getHomeReservation(query homeReservationQueryRow, slug string) (HomeReservation, bool, error) {
	var item HomeReservation
	var authorization int
	var createdAt, updatedAt string
	var confirmedAt sql.NullString
	err := query(`SELECT slug,household_name,owner_email,authorization_confirmed,status,created_at,updated_at,confirmed_at
		FROM home_reservations WHERE slug=$1`, slug).Scan(&item.Slug, &item.HouseholdName, &item.OwnerEmail,
		&authorization, &item.Status, &createdAt, &updatedAt, &confirmedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return HomeReservation{}, false, nil
	}
	if err != nil {
		return HomeReservation{}, false, err
	}
	item.AuthorizationConfirmed = authorization != 0
	item.CreatedAt = parseHomeReservationTimestamp(createdAt)
	item.UpdatedAt = parseHomeReservationTimestamp(updatedAt)
	if confirmedAt.Valid {
		value := parseHomeReservationTimestamp(confirmedAt.String)
		item.ConfirmedAt = &value
	}
	return item, true, nil
}

func normalizeHomeReservation(item HomeReservation, at time.Time) HomeReservation {
	at = homeReservationTime(at)
	item.Slug = textutil.Slug(item.Slug)
	item.HouseholdName = strings.TrimSpace(item.HouseholdName)
	item.OwnerEmail = textutil.Email(item.OwnerEmail)
	item.Status = HomeReservationEmailPending
	item.CreatedAt = at
	item.UpdatedAt = at
	return item
}

func homeReservationTime(at time.Time) time.Time {
	if at.IsZero() {
		at = time.Now()
	}
	return at.UTC()
}

func homeReservationTimestamp(at time.Time) string {
	return homeReservationTime(at).Format(time.RFC3339Nano)
}

func parseHomeReservationTimestamp(raw string) time.Time {
	value, _ := time.Parse(time.RFC3339Nano, raw)
	return value.UTC()
}
