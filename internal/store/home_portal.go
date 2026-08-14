package store

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

var ErrHomePortalActivationDenied = errors.New("home portal activation denied")

type HomePortal struct {
	Slug          string
	HouseholdName string
	OwnerEmail    string
	ActivatedAt   time.Time
	UpdatedAt     time.Time
}

type HomePortalStorage interface {
	Activate(slug, ownerEmail string, at time.Time) (HomePortal, bool, error)
	Get(slug string) (HomePortal, bool, error)
	ListByOwner(ownerEmail string) ([]HomePortal, error)
}

// MemoryHomePortalStore mirrors the production activation boundary for handler
// tests: the portal is only published after its owner membership was stored.
type MemoryHomePortalStore struct {
	mu           sync.Mutex
	items        map[string]HomePortal
	reservations HomeReservationStorage
	profiles     ProfileStorage
}

func NewMemoryHomePortalStore(reservations HomeReservationStorage, profiles ProfileStorage) *MemoryHomePortalStore {
	return &MemoryHomePortalStore{
		items:        map[string]HomePortal{},
		reservations: reservations,
		profiles:     profiles,
	}
}

func (s *MemoryHomePortalStore) Activate(slug, ownerEmail string, at time.Time) (HomePortal, bool, error) {
	if s == nil || s.reservations == nil || s.profiles == nil {
		return HomePortal{}, false, fmt.Errorf("home portal store unavailable")
	}
	slug = textutil.Slug(slug)
	ownerEmail = textutil.Email(ownerEmail)
	at = homeReservationTime(at)
	s.mu.Lock()
	defer s.mu.Unlock()

	reservation, found, err := s.reservations.Get(slug)
	if err != nil {
		return HomePortal{}, false, err
	}
	if !found || reservation.OwnerEmail != ownerEmail ||
		(reservation.Status != HomeReservationEmailConfirmed && reservation.Status != HomeReservationActive) {
		return HomePortal{}, false, ErrHomePortalActivationDenied
	}
	if existing, ok := s.items[slug]; ok && existing.OwnerEmail != ownerEmail {
		return HomePortal{}, false, ErrHomePortalActivationDenied
	}

	profile, profileFound := s.profiles.Get(ownerEmail)
	if !profileFound {
		created, err := s.profiles.Add(UserProfile{
			Email: ownerEmail, Role: RoleOwner, Status: "Aktiv", Tenants: []string{slug}, AuthMethods: DefaultAuthMethods(),
		})
		if err != nil {
			return HomePortal{}, false, err
		}
		if !created {
			return HomePortal{}, false, fmt.Errorf("owner profile could not be created")
		}
	} else if !profile.HasTenant(slug) || profile.ForTenant(slug).Role != RoleOwner {
		if _, found, err := s.profiles.SetTenantMembership(ownerEmail, slug, RoleOwner, nil); err != nil || !found {
			if err != nil {
				return HomePortal{}, false, err
			}
			return HomePortal{}, false, fmt.Errorf("owner membership could not be created")
		}
	}
	reservationActivator, ok := s.reservations.(interface {
		markActive(slug, ownerEmail string, at time.Time) error
	})
	if !ok {
		return HomePortal{}, false, fmt.Errorf("home reservation activation unavailable")
	}
	if err := reservationActivator.markActive(slug, ownerEmail, at); err != nil {
		return HomePortal{}, false, err
	}

	portal, existed := s.items[slug]
	if !existed {
		portal.ActivatedAt = at
	}
	portal.Slug = slug
	portal.HouseholdName = reservation.HouseholdName
	portal.OwnerEmail = ownerEmail
	portal.UpdatedAt = at
	s.items[slug] = portal
	return portal, !existed, nil
}

func (s *MemoryHomePortalStore) Get(slug string) (HomePortal, bool, error) {
	if s == nil {
		return HomePortal{}, false, fmt.Errorf("home portal store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[textutil.Slug(slug)]
	return item, ok, nil
}

func (s *MemoryHomePortalStore) ListByOwner(ownerEmail string) ([]HomePortal, error) {
	if s == nil {
		return nil, fmt.Errorf("home portal store unavailable")
	}
	ownerEmail = textutil.Email(ownerEmail)
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]HomePortal, 0)
	for _, item := range s.items {
		if item.OwnerEmail == ownerEmail {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Slug < items[j].Slug })
	return items, nil
}

type SQLHomePortalStore struct {
	db *sql.DB
}

func NewSQLHomePortalStore(db *sql.DB) *SQLHomePortalStore {
	return &SQLHomePortalStore{db: db}
}

// Activate publishes the tenant and grants its confirmed owner access in one
// transaction. No caller has to repair a portal without an owner membership.
func (s *SQLHomePortalStore) Activate(slug, ownerEmail string, at time.Time) (HomePortal, bool, error) {
	if s == nil || s.db == nil {
		return HomePortal{}, false, fmt.Errorf("home portal store unavailable")
	}
	slug = textutil.Slug(slug)
	ownerEmail = textutil.Email(ownerEmail)
	at = homeReservationTime(at)
	tx, err := s.db.Begin()
	if err != nil {
		return HomePortal{}, false, err
	}
	defer tx.Rollback()

	reservation, found, err := getHomeReservation(tx.QueryRow, slug)
	if err != nil {
		return HomePortal{}, false, err
	}
	if !found || reservation.OwnerEmail != ownerEmail ||
		(reservation.Status != HomeReservationEmailConfirmed && reservation.Status != HomeReservationActive) {
		return HomePortal{}, false, ErrHomePortalActivationDenied
	}

	existing, existed, err := getHomePortal(tx.QueryRow, slug)
	if err != nil {
		return HomePortal{}, false, err
	}
	if existed && existing.OwnerEmail != ownerEmail {
		return HomePortal{}, false, ErrHomePortalActivationDenied
	}

	identity := NewSQLIdentityStore(s.db)
	person, err := scanPerson(tx.QueryRow(`SELECT `+personColumns+` FROM persons WHERE email=?`, ownerEmail).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		person, err = identity.upsertPersonTx(tx, Person{Email: ownerEmail, AuthMethods: DefaultAuthMethods()}, at)
	}
	if err != nil {
		return HomePortal{}, false, err
	}
	membership, membershipFound := HouseMembership{}, false
	if item, scanErr := scanMembership(tx.QueryRow(
		`SELECT `+membershipColumns+` FROM house_memberships WHERE person_id=? AND tenant_slug=?`, person.ID, slug,
	).Scan); scanErr == nil {
		membership, membershipFound = item, true
	} else if !errors.Is(scanErr, sql.ErrNoRows) {
		return HomePortal{}, false, scanErr
	}
	if !membershipFound {
		membership = HouseMembership{PersonID: person.ID, TenantSlug: slug}
	}
	membership.Role = RoleOwner
	membership.Status = "Aktiv"
	if _, err := identity.setMembershipTx(tx, membership, at); err != nil {
		return HomePortal{}, false, err
	}

	activatedAt := at
	if existed {
		activatedAt = existing.ActivatedAt
	}
	if _, err := tx.Exec(`INSERT INTO home_portals(slug,household_name,owner_email,activated_at,updated_at)
		VALUES(?,?,?,?,?) ON CONFLICT(slug) DO UPDATE SET household_name=excluded.household_name,
		owner_email=excluded.owner_email, updated_at=excluded.updated_at`,
		slug, strings.TrimSpace(reservation.HouseholdName), ownerEmail,
		homeReservationTimestamp(activatedAt), homeReservationTimestamp(at)); err != nil {
		return HomePortal{}, false, err
	}
	if _, err := tx.Exec(`UPDATE home_reservations SET status=?, updated_at=? WHERE slug=? AND owner_email=?`,
		HomeReservationActive, homeReservationTimestamp(at), slug, ownerEmail); err != nil {
		return HomePortal{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return HomePortal{}, false, err
	}
	return HomePortal{
		Slug: slug, HouseholdName: strings.TrimSpace(reservation.HouseholdName), OwnerEmail: ownerEmail,
		ActivatedAt: activatedAt, UpdatedAt: at,
	}, !existed, nil
}

func (s *SQLHomePortalStore) Get(slug string) (HomePortal, bool, error) {
	if s == nil || s.db == nil {
		return HomePortal{}, false, fmt.Errorf("home portal store unavailable")
	}
	return getHomePortal(s.db.QueryRow, textutil.Slug(slug))
}

func (s *SQLHomePortalStore) ListByOwner(ownerEmail string) ([]HomePortal, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("home portal store unavailable")
	}
	rows, err := s.db.Query(`SELECT slug,household_name,owner_email,activated_at,updated_at
		FROM home_portals WHERE owner_email=? ORDER BY slug`, textutil.Email(ownerEmail))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]HomePortal, 0)
	for rows.Next() {
		var item HomePortal
		var activatedAt, updatedAt string
		if err := rows.Scan(&item.Slug, &item.HouseholdName, &item.OwnerEmail, &activatedAt, &updatedAt); err != nil {
			return nil, err
		}
		item.ActivatedAt = parseHomeReservationTimestamp(activatedAt)
		item.UpdatedAt = parseHomeReservationTimestamp(updatedAt)
		items = append(items, item)
	}
	return items, rows.Err()
}

func getHomePortal(query homeReservationQueryRow, slug string) (HomePortal, bool, error) {
	var item HomePortal
	var activatedAt, updatedAt string
	err := query(`SELECT slug,household_name,owner_email,activated_at,updated_at FROM home_portals WHERE slug=?`, slug).
		Scan(&item.Slug, &item.HouseholdName, &item.OwnerEmail, &activatedAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return HomePortal{}, false, nil
	}
	if err != nil {
		return HomePortal{}, false, err
	}
	item.ActivatedAt = parseHomeReservationTimestamp(activatedAt)
	item.UpdatedAt = parseHomeReservationTimestamp(updatedAt)
	return item, true, nil
}
