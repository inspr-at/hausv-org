package store

import (
	"crypto/subtle"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

const (
	HomeConnectorPairing   = "pairing"
	HomeConnectorConnected = "connected"
	HomeConnectorRevoked   = "revoked"
)

type HomeConnector struct {
	Slug                 string
	Status               string
	CredentialHash       []byte
	Generation           int
	PairingHash          []byte
	PairingExpiresAt     *time.Time
	ConnectorVersion     string
	HomeAssistantVersion string
	EntityCount          int
	CreatedAt            time.Time
	UpdatedAt            time.Time
	PairedAt             *time.Time
	LastSeenAt           *time.Time
}

type HomeConnectorHeartbeat struct {
	ConnectorVersion     string
	HomeAssistantVersion string
	EntityCount          int
}

type HomeConnectorStorage interface {
	StartPairing(slug string, pairingHash []byte, expiresAt, now time.Time) (HomeConnector, error)
	ExchangePairing(pairingHash, credentialHash []byte, heartbeat HomeConnectorHeartbeat, now time.Time) (HomeConnector, bool, error)
	Heartbeat(credentialHash []byte, heartbeat HomeConnectorHeartbeat, now time.Time) (HomeConnector, bool, error)
	Get(slug string) (HomeConnector, bool, error)
	Revoke(slug string, now time.Time) (HomeConnector, bool, error)
}

type MemoryHomeConnectorStore struct {
	mu    sync.Mutex
	items map[string]HomeConnector
}

func NewMemoryHomeConnectorStore() *MemoryHomeConnectorStore {
	return &MemoryHomeConnectorStore{items: map[string]HomeConnector{}}
}

func (s *MemoryHomeConnectorStore) StartPairing(slug string, pairingHash []byte, expiresAt, now time.Time) (HomeConnector, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	slug = textutil.Slug(slug)
	now = homeReservationTime(now)
	expiresAt = homeReservationTime(expiresAt)
	if slug == "" || len(pairingHash) == 0 || !expiresAt.After(now) {
		return HomeConnector{}, fmt.Errorf("invalid home connector pairing")
	}
	item, found := s.items[slug]
	if !found {
		item = HomeConnector{Slug: slug, Status: HomeConnectorPairing, CreatedAt: now}
	}
	if len(item.CredentialHash) == 0 {
		item.Status = HomeConnectorPairing
	}
	item.PairingHash = cloneConnectorHash(pairingHash)
	item.PairingExpiresAt = connectorTimePointer(expiresAt)
	item.UpdatedAt = now
	s.items[slug] = item
	return cloneHomeConnector(item), nil
}

func (s *MemoryHomeConnectorStore) ExchangePairing(pairingHash, credentialHash []byte, heartbeat HomeConnectorHeartbeat, now time.Time) (HomeConnector, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now = homeReservationTime(now)
	if len(pairingHash) == 0 || len(credentialHash) == 0 {
		return HomeConnector{}, false, nil
	}
	for slug, item := range s.items {
		if item.PairingExpiresAt == nil || !item.PairingExpiresAt.After(now) || !connectorHashesEqual(item.PairingHash, pairingHash) {
			continue
		}
		applyConnectorExchange(&item, credentialHash, heartbeat, now)
		s.items[slug] = item
		return cloneHomeConnector(item), true, nil
	}
	return HomeConnector{}, false, nil
}

func (s *MemoryHomeConnectorStore) Heartbeat(credentialHash []byte, heartbeat HomeConnectorHeartbeat, now time.Time) (HomeConnector, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now = homeReservationTime(now)
	for slug, item := range s.items {
		if item.Status != HomeConnectorConnected || !connectorHashesEqual(item.CredentialHash, credentialHash) {
			continue
		}
		applyConnectorHeartbeat(&item, heartbeat, now)
		s.items[slug] = item
		return cloneHomeConnector(item), true, nil
	}
	return HomeConnector{}, false, nil
}

func (s *MemoryHomeConnectorStore) Get(slug string) (HomeConnector, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, found := s.items[textutil.Slug(slug)]
	return cloneHomeConnector(item), found, nil
}

func (s *MemoryHomeConnectorStore) Revoke(slug string, now time.Time) (HomeConnector, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	slug = textutil.Slug(slug)
	item, found := s.items[slug]
	if !found {
		return HomeConnector{}, false, nil
	}
	item.Status = HomeConnectorRevoked
	item.CredentialHash = nil
	item.PairingHash = nil
	item.PairingExpiresAt = nil
	item.ConnectorVersion = ""
	item.HomeAssistantVersion = ""
	item.EntityCount = 0
	item.LastSeenAt = nil
	item.UpdatedAt = homeReservationTime(now)
	s.items[slug] = item
	return cloneHomeConnector(item), true, nil
}

type SQLHomeConnectorStore struct{ db *sql.DB }

func NewSQLHomeConnectorStore(db *sql.DB) *SQLHomeConnectorStore {
	return &SQLHomeConnectorStore{db: db}
}

func (s *SQLHomeConnectorStore) StartPairing(slug string, pairingHash []byte, expiresAt, now time.Time) (HomeConnector, error) {
	if s == nil || s.db == nil {
		return HomeConnector{}, fmt.Errorf("home connector store unavailable")
	}
	slug = textutil.Slug(slug)
	now = homeReservationTime(now)
	expiresAt = homeReservationTime(expiresAt)
	if slug == "" || len(pairingHash) == 0 || !expiresAt.After(now) {
		return HomeConnector{}, fmt.Errorf("invalid home connector pairing")
	}
	_, err := s.db.Exec(`INSERT INTO home_connectors
		(slug,status,credential_hash,generation,pairing_hash,pairing_expires_at,created_at,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT(slug) DO UPDATE SET
		status=CASE WHEN home_connectors.credential_hash IS NULL THEN excluded.status ELSE home_connectors.status END,
		pairing_hash=excluded.pairing_hash,pairing_expires_at=excluded.pairing_expires_at,updated_at=excluded.updated_at`,
		slug, HomeConnectorPairing, nil, 0, pairingHash, homeReservationTimestamp(expiresAt), homeReservationTimestamp(now), homeReservationTimestamp(now))
	if err != nil {
		return HomeConnector{}, err
	}
	item, found, err := s.Get(slug)
	if err != nil || !found {
		return HomeConnector{}, err
	}
	return item, nil
}

func (s *SQLHomeConnectorStore) ExchangePairing(pairingHash, credentialHash []byte, heartbeat HomeConnectorHeartbeat, now time.Time) (HomeConnector, bool, error) {
	if s == nil || s.db == nil {
		return HomeConnector{}, false, fmt.Errorf("home connector store unavailable")
	}
	now = homeReservationTime(now)
	if len(pairingHash) == 0 || len(credentialHash) == 0 {
		return HomeConnector{}, false, nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return HomeConnector{}, false, err
	}
	defer tx.Rollback()
	var slug string
	var pairingExpires string
	err = tx.QueryRow(`SELECT slug,pairing_expires_at FROM home_connectors WHERE pairing_hash=$1`,
		pairingHash).Scan(&slug, &pairingExpires)
	if errors.Is(err, sql.ErrNoRows) {
		return HomeConnector{}, false, nil
	}
	if err != nil {
		return HomeConnector{}, false, err
	}
	if !parseHomeReservationTimestamp(pairingExpires).After(now) {
		return HomeConnector{}, false, nil
	}
	result, err := tx.Exec(`UPDATE home_connectors SET status=$1,credential_hash=$2,generation=generation+1,
		pairing_hash=NULL,pairing_expires_at=NULL,connector_version=$3,ha_version=$4,entity_count=$5,
		paired_at=$6,last_seen_at=$7,updated_at=$8 WHERE slug=$9 AND pairing_hash=$10`,
		HomeConnectorConnected, credentialHash, heartbeat.ConnectorVersion, heartbeat.HomeAssistantVersion,
		heartbeat.EntityCount, homeReservationTimestamp(now), homeReservationTimestamp(now), homeReservationTimestamp(now), slug, pairingHash)
	if err != nil {
		return HomeConnector{}, false, err
	}
	count, err := result.RowsAffected()
	if err != nil || count != 1 {
		return HomeConnector{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return HomeConnector{}, false, err
	}
	item, found, err := s.Get(slug)
	return item, found, err
}

func (s *SQLHomeConnectorStore) Heartbeat(credentialHash []byte, heartbeat HomeConnectorHeartbeat, now time.Time) (HomeConnector, bool, error) {
	if s == nil || s.db == nil {
		return HomeConnector{}, false, fmt.Errorf("home connector store unavailable")
	}
	now = homeReservationTime(now)
	result, err := s.db.Exec(`UPDATE home_connectors SET connector_version=$1,ha_version=$2,entity_count=$3,last_seen_at=$4,updated_at=$5
		WHERE status=$6 AND credential_hash=$7`, heartbeat.ConnectorVersion, heartbeat.HomeAssistantVersion, heartbeat.EntityCount,
		homeReservationTimestamp(now), homeReservationTimestamp(now), HomeConnectorConnected, credentialHash)
	if err != nil {
		return HomeConnector{}, false, err
	}
	count, err := result.RowsAffected()
	if err != nil || count == 0 {
		return HomeConnector{}, false, err
	}
	return getHomeConnector(s.db.QueryRow, "credential_hash=$1", credentialHash)
}

func (s *SQLHomeConnectorStore) Get(slug string) (HomeConnector, bool, error) {
	if s == nil || s.db == nil {
		return HomeConnector{}, false, fmt.Errorf("home connector store unavailable")
	}
	return getHomeConnector(s.db.QueryRow, "slug=$1", textutil.Slug(slug))
}

func (s *SQLHomeConnectorStore) Revoke(slug string, now time.Time) (HomeConnector, bool, error) {
	if s == nil || s.db == nil {
		return HomeConnector{}, false, fmt.Errorf("home connector store unavailable")
	}
	slug = textutil.Slug(slug)
	result, err := s.db.Exec(`UPDATE home_connectors SET status=$1,credential_hash=NULL,pairing_hash=NULL,
		pairing_expires_at=NULL,connector_version='',ha_version='',entity_count=0,last_seen_at=NULL,updated_at=$2 WHERE slug=$3`,
		HomeConnectorRevoked, homeReservationTimestamp(now), slug)
	if err != nil {
		return HomeConnector{}, false, err
	}
	count, err := result.RowsAffected()
	if err != nil || count == 0 {
		return HomeConnector{}, false, err
	}
	item, found, err := s.Get(slug)
	return item, found, err
}

type homeConnectorQueryRow func(string, ...any) *sql.Row

func getHomeConnector(query homeConnectorQueryRow, where string, value any) (HomeConnector, bool, error) {
	var item HomeConnector
	var pairingExpires, pairedAt, lastSeen sql.NullString
	var createdAt, updatedAt string
	err := query(`SELECT slug,status,credential_hash,generation,pairing_hash,pairing_expires_at,
		connector_version,ha_version,entity_count,created_at,updated_at,paired_at,last_seen_at
		FROM home_connectors WHERE `+where, value).Scan(&item.Slug, &item.Status, &item.CredentialHash, &item.Generation,
		&item.PairingHash, &pairingExpires, &item.ConnectorVersion, &item.HomeAssistantVersion, &item.EntityCount,
		&createdAt, &updatedAt, &pairedAt, &lastSeen)
	if errors.Is(err, sql.ErrNoRows) {
		return HomeConnector{}, false, nil
	}
	if err != nil {
		return HomeConnector{}, false, err
	}
	item.CreatedAt = parseHomeReservationTimestamp(createdAt)
	item.UpdatedAt = parseHomeReservationTimestamp(updatedAt)
	item.PairingExpiresAt = parseConnectorNullTime(pairingExpires)
	item.PairedAt = parseConnectorNullTime(pairedAt)
	item.LastSeenAt = parseConnectorNullTime(lastSeen)
	return item, true, nil
}

func applyConnectorExchange(item *HomeConnector, credentialHash []byte, heartbeat HomeConnectorHeartbeat, now time.Time) {
	item.Status = HomeConnectorConnected
	item.CredentialHash = cloneConnectorHash(credentialHash)
	item.Generation++
	item.PairingHash = nil
	item.PairingExpiresAt = nil
	item.PairedAt = connectorTimePointer(now)
	applyConnectorHeartbeat(item, heartbeat, now)
}

func applyConnectorHeartbeat(item *HomeConnector, heartbeat HomeConnectorHeartbeat, now time.Time) {
	item.ConnectorVersion = heartbeat.ConnectorVersion
	item.HomeAssistantVersion = heartbeat.HomeAssistantVersion
	item.EntityCount = heartbeat.EntityCount
	item.LastSeenAt = connectorTimePointer(now)
	item.UpdatedAt = now
}

func cloneHomeConnector(item HomeConnector) HomeConnector {
	item.CredentialHash = cloneConnectorHash(item.CredentialHash)
	item.PairingHash = cloneConnectorHash(item.PairingHash)
	return item
}

func cloneConnectorHash(value []byte) []byte { return append([]byte(nil), value...) }

func connectorHashesEqual(left, right []byte) bool {
	return len(left) == len(right) && len(left) > 0 && subtle.ConstantTimeCompare(left, right) == 1
}

func connectorTimePointer(value time.Time) *time.Time {
	value = homeReservationTime(value)
	return &value
}

func parseConnectorNullTime(value sql.NullString) *time.Time {
	if !value.Valid {
		return nil
	}
	parsed := parseHomeReservationTimestamp(value.String)
	return &parsed
}
