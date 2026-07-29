// Package store is the persistence layer: one atomic-JSON store per domain.
//
// It must not import HTTP, mail, templates, or the app.
package store

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/markus-barta/hausv-org/internal/textutil"
)

const (
	UnitPaymentStatusOpen    = "offen"
	UnitPaymentStatusPaid    = "bezahlt"
	UnitPaymentStatusPartial = "teilbezahlt"
	UnitPaymentStatusOverdue = "ueberfaellig"
)

type AnnouncementReadStore struct {
	mu   sync.Mutex
	path string
	data AnnouncementReadStoreData
}

type AnnouncementReadStoreData struct {
	Seen map[string]map[string]time.Time `json:"seen"`
}

type NotificationPrefStore struct {
	mu   sync.Mutex
	path string
	data NotificationPrefStoreData
}

type NotificationPrefStoreData struct {
	Users map[string]NotificationPreferences `json:"users"`
}

type ProfileOverlayStore struct {
	mu   sync.Mutex
	path string
	data ProfileOverlayStoreData
}

type ProfileOverlayStoreData struct {
	Profiles map[string]ProfileOverlay `json:"profiles"`
}

type ProfileOverlay struct {
	Title          string    `json:"title"`
	FirstName      string    `json:"first_name"`
	LastName       string    `json:"last_name"`
	Phone          string    `json:"phone,omitempty"`
	DirectoryOptIn bool      `json:"directory_opt_in,omitempty"`
	UpdatedAt      time.Time `json:"updated_at,omitempty"`
}

type NotificationPreferences struct {
	Email        map[string]bool `json:"email"`
	Unsubscribed bool            `json:"unsubscribed,omitempty"`
}

type UnitPaymentStatusStore struct {
	mu   sync.Mutex
	path string
	data UnitPaymentStatusData
}

type UnitPaymentStatusData struct {
	Statuses []UnitPaymentStatus `json:"statuses"`
}

type UnitPaymentStatus struct {
	TenantSlug string    `json:"tenant"`
	UnitID     string    `json:"unit_id"`
	Status     string    `json:"status"`
	UpdatedAt  time.Time `json:"updated_at"`
	UpdatedBy  string    `json:"updated_by,omitempty"`
}

type ContactBookStore struct {
	mu   sync.Mutex
	path string
	data ContactBookStoreData
}

type ContactBookStoreData struct {
	Contacts []ManagedContact `json:"contacts"`
}

type ManagedContact struct {
	ID                 string    `json:"id"`
	TenantSlug         string    `json:"tenant"`
	Kind               string    `json:"kind"`
	Name               string    `json:"name,omitempty"`
	Company            string    `json:"company,omitempty"`
	Email              string    `json:"email,omitempty"`
	Phone              string    `json:"phone,omitempty"`
	Notes              string    `json:"notes,omitempty"`
	ServiceRegion      string    `json:"service_region,omitempty"`
	Qualification      string    `json:"qualification,omitempty"`
	EnergyCapabilities []string  `json:"energy_capabilities,omitempty"`
	Active             bool      `json:"active"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func ManagedContactDisplayName(item ManagedContact) string {
	return textutil.FirstNonEmpty(item.Name, item.Company, item.Email, item.Phone, "Kontakt")
}

func NormalizeUnitPaymentRecord(item UnitPaymentStatus) (UnitPaymentStatus, error) {
	item.TenantSlug = textutil.Slug(item.TenantSlug)
	item.UnitID = NormalizeUnitID(item.UnitID)
	item.Status = NormalizeUnitPaymentStatus(item.Status)
	item.UpdatedBy = textutil.Email(item.UpdatedBy)
	if !item.UpdatedAt.IsZero() {
		item.UpdatedAt = item.UpdatedAt.UTC().Truncate(time.Second)
	}
	if item.TenantSlug == "" || item.UnitID == "" || item.Status == "" {
		return UnitPaymentStatus{}, fmt.Errorf("invalid unit payment status")
	}
	return item, nil
}

func NormalizeUnitPaymentStatus(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", UnitPaymentStatusOpen, "open":
		return UnitPaymentStatusOpen
	case UnitPaymentStatusPaid, "paid":
		return UnitPaymentStatusPaid
	case UnitPaymentStatusPartial, "teilweise", "partial", "partial-paid":
		return UnitPaymentStatusPartial
	case UnitPaymentStatusOverdue, "überfällig", "overdue":
		return UnitPaymentStatusOverdue
	default:
		return ""
	}
}

func SortUnitPaymentStatuses(items []UnitPaymentStatus) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].TenantSlug != items[j].TenantSlug {
			return items[i].TenantSlug < items[j].TenantSlug
		}
		return items[i].UnitID < items[j].UnitID
	})
}

// The notification event vocabulary. The store owns the KEYS (persisted and
// validated here); the human labels are a view concern and stay in the UI layer.
const (
	NotificationEventAnnouncement = "announcement"
	NotificationEventIssue        = "issue"
	NotificationEventVote         = "vote"
	NotificationEventDocument     = "document"
	NotificationEventPayment      = "payment"
	NotificationEventCharging     = "charging"
)

// NotificationEvents lists the valid event keys in display order.
var NotificationEvents = []string{
	NotificationEventAnnouncement,
	NotificationEventIssue,
	NotificationEventVote,
	NotificationEventDocument,
	NotificationEventPayment,
	NotificationEventCharging,
}

func DefaultNotificationPreferences() NotificationPreferences {
	prefs := NotificationPreferences{Email: map[string]bool{}}
	for _, event := range NotificationEvents {
		prefs.Email[event] = true
	}
	return prefs
}

func MergeNotificationPreferences(prefs NotificationPreferences) NotificationPreferences {
	merged := DefaultNotificationPreferences()
	merged.Unsubscribed = prefs.Unsubscribed
	for event, enabled := range prefs.Email {
		event = NormalizeNotificationEvent(event)
		if event == "" {
			continue
		}
		merged.Email[event] = enabled
	}
	return merged
}

func NormalizeNotificationPreferences(prefs NotificationPreferences) NotificationPreferences {
	normalized := NotificationPreferences{
		Email:        map[string]bool{},
		Unsubscribed: prefs.Unsubscribed,
	}
	merged := MergeNotificationPreferences(prefs)
	for _, event := range NotificationEvents {
		normalized.Email[event] = merged.Email[event]
	}
	return normalized
}

func NormalizeNotificationEvent(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	for _, event := range NotificationEvents {
		if raw == event {
			return event
		}
	}
	return ""
}

func NewAnnouncementReadStore(path string) (*AnnouncementReadStore, error) {
	store := &AnnouncementReadStore{path: path, data: AnnouncementReadStoreData{Seen: map[string]map[string]time.Time{}}}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read announcement read data")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("invalid announcement read data")
	}
	if store.data.Seen == nil {
		store.data.Seen = map[string]map[string]time.Time{}
	}
	return store, nil
}

func (s *AnnouncementReadStore) LastSeen(tenantSlug string, email string) time.Time {
	if s == nil {
		return time.Time{}
	}
	tenantSlug = textutil.Slug(tenantSlug)
	email = textutil.Email(email)
	s.mu.Lock()
	defer s.mu.Unlock()
	if tenantSlug == "" || email == "" {
		return time.Time{}
	}
	return s.data.Seen[tenantSlug][email]
}

func (s *AnnouncementReadStore) MarkSeen(tenantSlug string, email string, seenAt time.Time) error {
	if s == nil {
		return nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	email = textutil.Email(email)
	if tenantSlug == "" || email == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.Seen == nil {
		s.data.Seen = map[string]map[string]time.Time{}
	}
	if s.data.Seen[tenantSlug] == nil {
		s.data.Seen[tenantSlug] = map[string]time.Time{}
	}
	s.data.Seen[tenantSlug][email] = seenAt.UTC()
	return s.saveLocked()
}

func (s *AnnouncementReadStore) saveLocked() error {
	return SaveJSONAtomic(s.path, s.data, "announcement read")
}

func NewNotificationPrefStore(path string) (*NotificationPrefStore, error) {
	store := &NotificationPrefStore{path: path, data: NotificationPrefStoreData{Users: map[string]NotificationPreferences{}}}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read notification preference data")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("invalid notification preference data")
	}
	if store.data.Users == nil {
		store.data.Users = map[string]NotificationPreferences{}
	}
	normalized := map[string]NotificationPreferences{}
	for email, prefs := range store.data.Users {
		email = textutil.Email(email)
		if email == "" {
			continue
		}
		normalized[email] = NormalizeNotificationPreferences(prefs)
	}
	store.data.Users = normalized
	return store, nil
}

func NewProfileOverlayStore(path string) (*ProfileOverlayStore, error) {
	store := &ProfileOverlayStore{path: path, data: ProfileOverlayStoreData{Profiles: map[string]ProfileOverlay{}}}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read profile data")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("invalid profile data")
	}
	if store.data.Profiles == nil {
		store.data.Profiles = map[string]ProfileOverlay{}
	}
	normalized := map[string]ProfileOverlay{}
	for email, overlay := range store.data.Profiles {
		email = textutil.Email(email)
		if email == "" {
			continue
		}
		normalized[email] = NormalizeProfileOverlay(overlay)
	}
	store.data.Profiles = normalized
	return store, nil
}

func (s *ProfileOverlayStore) Get(email string) (ProfileOverlay, bool) {
	if s == nil {
		return ProfileOverlay{}, false
	}
	email = textutil.Email(email)
	if email == "" {
		return ProfileOverlay{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	overlay, ok := s.data.Profiles[email]
	return overlay, ok
}

func (s *ProfileOverlayStore) Set(email string, overlay ProfileOverlay) error {
	if s == nil {
		return nil
	}
	email = textutil.Email(email)
	if email == "" {
		return fmt.Errorf("invalid profile email")
	}
	overlay = NormalizeProfileOverlay(overlay)
	overlay.UpdatedAt = time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.Profiles == nil {
		s.data.Profiles = map[string]ProfileOverlay{}
	}
	s.data.Profiles[email] = overlay
	return s.saveLocked()
}

func (s *ProfileOverlayStore) saveLocked() error {
	return SaveJSONAtomic(s.path, s.data, "profile")
}

func NormalizeProfileOverlay(overlay ProfileOverlay) ProfileOverlay {
	overlay.Title = strings.TrimSpace(overlay.Title)
	overlay.FirstName = strings.TrimSpace(overlay.FirstName)
	overlay.LastName = strings.TrimSpace(overlay.LastName)
	overlay.Phone = strings.TrimSpace(overlay.Phone)
	if !overlay.UpdatedAt.IsZero() {
		overlay.UpdatedAt = overlay.UpdatedAt.UTC()
	}
	return overlay
}

func (s *NotificationPrefStore) Get(email string) NotificationPreferences {
	prefs := DefaultNotificationPreferences()
	if s == nil {
		return prefs
	}
	email = textutil.Email(email)
	if email == "" {
		return prefs
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if stored, ok := s.data.Users[email]; ok {
		prefs = MergeNotificationPreferences(stored)
	}
	return prefs
}

func (s *NotificationPrefStore) Set(email string, prefs NotificationPreferences) error {
	if s == nil {
		return nil
	}
	email = textutil.Email(email)
	if email == "" {
		return fmt.Errorf("invalid notification preference email")
	}
	prefs = NormalizeNotificationPreferences(prefs)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.Users == nil {
		s.data.Users = map[string]NotificationPreferences{}
	}
	s.data.Users[email] = prefs
	return s.saveLocked()
}

func (s *NotificationPrefStore) EmailEnabled(email string, event string) bool {
	event = NormalizeNotificationEvent(event)
	if event == "" {
		return false
	}
	prefs := DefaultNotificationPreferences()
	if s != nil {
		prefs = s.Get(email)
	}
	if prefs.Unsubscribed {
		return false
	}
	if prefs.Email == nil {
		return true
	}
	enabled, ok := prefs.Email[event]
	if !ok {
		return true
	}
	return enabled
}

func (s *NotificationPrefStore) saveLocked() error {
	return SaveJSONAtomic(s.path, s.data, "notification preference")
}

func NewContactBookStore(path string) (*ContactBookStore, error) {
	store := &ContactBookStore{path: path, data: ContactBookStoreData{Contacts: []ManagedContact{}}}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read contact data")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("invalid contact data")
	}
	return store, nil
}

func NewUnitPaymentStatusStore(path string) (*UnitPaymentStatusStore, error) {
	store := &UnitPaymentStatusStore{path: path, data: UnitPaymentStatusData{Statuses: []UnitPaymentStatus{}}}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read unit payment status data")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("invalid unit payment status data")
	}
	return store, nil
}

func (s *UnitPaymentStatusStore) Set(item UnitPaymentStatus) (UnitPaymentStatus, error) {
	if s == nil {
		return UnitPaymentStatus{}, fmt.Errorf("unit payment status store not configured")
	}
	item, err := NormalizeUnitPaymentRecord(item)
	if err != nil {
		return UnitPaymentStatus{}, err
	}
	now := time.Now().UTC().Truncate(time.Second)
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.data.Statuses {
		if textutil.Slug(existing.TenantSlug) != item.TenantSlug || NormalizeUnitID(existing.UnitID) != item.UnitID {
			continue
		}
		item.UpdatedAt = now
		s.data.Statuses[i] = item
		SortUnitPaymentStatuses(s.data.Statuses)
		if err := s.saveLocked(); err != nil {
			return UnitPaymentStatus{}, err
		}
		return item, nil
	}
	item.UpdatedAt = now
	s.data.Statuses = append(s.data.Statuses, item)
	SortUnitPaymentStatuses(s.data.Statuses)
	if err := s.saveLocked(); err != nil {
		return UnitPaymentStatus{}, err
	}
	return item, nil
}

func (s *UnitPaymentStatusStore) Get(tenantSlug string, unitID string) (UnitPaymentStatus, bool) {
	if s == nil {
		return UnitPaymentStatus{}, false
	}
	tenantSlug = textutil.Slug(tenantSlug)
	unitID = NormalizeUnitID(unitID)
	if tenantSlug == "" || unitID == "" {
		return UnitPaymentStatus{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.data.Statuses {
		normalized, err := NormalizeUnitPaymentRecord(item)
		if err != nil {
			continue
		}
		if normalized.TenantSlug == tenantSlug && normalized.UnitID == unitID {
			return normalized, true
		}
	}
	return UnitPaymentStatus{}, false
}

func (s *UnitPaymentStatusStore) ListTenant(tenantSlug string) []UnitPaymentStatus {
	if s == nil {
		return nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []UnitPaymentStatus{}
	for _, item := range s.data.Statuses {
		normalized, err := NormalizeUnitPaymentRecord(item)
		if err != nil || normalized.TenantSlug != tenantSlug {
			continue
		}
		out = append(out, normalized)
	}
	SortUnitPaymentStatuses(out)
	return out
}

func (s *UnitPaymentStatusStore) saveLocked() error {
	return SaveJSONAtomic(s.path, s.data, "unit payment status")
}

func (s *ContactBookStore) Upsert(item ManagedContact) (ManagedContact, bool, error) {
	if s == nil {
		return ManagedContact{}, false, fmt.Errorf("contact store not configured")
	}
	item, err := NormalizeManagedContact(item)
	if err != nil {
		return ManagedContact{}, false, err
	}
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.data.Contacts {
		if textutil.Slug(existing.TenantSlug) != item.TenantSlug || existing.ID != item.ID || item.ID == "" {
			continue
		}
		item.CreatedAt = existing.CreatedAt
		if item.CreatedAt.IsZero() {
			item.CreatedAt = now
		}
		item.UpdatedAt = now
		s.data.Contacts[i] = item
		SortManagedContacts(s.data.Contacts)
		if err := s.saveLocked(); err != nil {
			return ManagedContact{}, false, err
		}
		return item, false, nil
	}
	if item.ID == "" {
		id, err := randomToken(10)
		if err != nil {
			return ManagedContact{}, false, err
		}
		item.ID = id
	}
	item.CreatedAt = now
	item.UpdatedAt = now
	s.data.Contacts = append(s.data.Contacts, item)
	SortManagedContacts(s.data.Contacts)
	if err := s.saveLocked(); err != nil {
		return ManagedContact{}, false, err
	}
	return item, true, nil
}

func (s *ContactBookStore) Deactivate(tenantSlug string, id string, at time.Time) (ManagedContact, error) {
	if s == nil {
		return ManagedContact{}, fmt.Errorf("contact store not configured")
	}
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	if tenantSlug == "" || id == "" {
		return ManagedContact{}, fmt.Errorf("invalid contact")
	}
	if at.IsZero() {
		at = time.Now().UTC()
	} else {
		at = at.UTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.data.Contacts {
		if textutil.Slug(existing.TenantSlug) != tenantSlug || existing.ID != id {
			continue
		}
		existing.Active = false
		existing.UpdatedAt = at
		s.data.Contacts[i] = existing
		SortManagedContacts(s.data.Contacts)
		if err := s.saveLocked(); err != nil {
			return ManagedContact{}, err
		}
		return existing, nil
	}
	return ManagedContact{}, nil
}

func (s *ContactBookStore) ListTenant(tenantSlug string, includeInactive bool) []ManagedContact {
	if s == nil {
		return nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []ManagedContact{}
	for _, item := range s.data.Contacts {
		normalized, err := NormalizeManagedContact(item)
		if err != nil || normalized.TenantSlug != tenantSlug {
			continue
		}
		if !includeInactive && !normalized.Active {
			continue
		}
		out = append(out, normalized)
	}
	SortManagedContacts(out)
	return out
}

func (s *ContactBookStore) saveLocked() error {
	return SaveJSONAtomic(s.path, s.data, "contact")
}

func NormalizeManagedContact(item ManagedContact) (ManagedContact, error) {
	item.TenantSlug = textutil.Slug(item.TenantSlug)
	item.ID = strings.TrimSpace(item.ID)
	item.Kind = NormalizeContactKind(item.Kind)
	item.Name = textutil.Truncate(strings.TrimSpace(item.Name), 120)
	item.Company = textutil.Truncate(strings.TrimSpace(item.Company), 140)
	item.Email = textutil.Email(item.Email)
	item.Phone = textutil.Truncate(strings.TrimSpace(item.Phone), 80)
	item.Notes = textutil.Truncate(strings.TrimSpace(item.Notes), 300)
	item.ServiceRegion = textutil.Truncate(strings.TrimSpace(item.ServiceRegion), 120)
	item.Qualification = textutil.Truncate(strings.TrimSpace(item.Qualification), 240)
	item.EnergyCapabilities = NormalizeEnergyCapabilities(item.EnergyCapabilities)
	if item.TenantSlug == "" || item.Kind == "" {
		return ManagedContact{}, fmt.Errorf("invalid contact")
	}
	if item.Name == "" && item.Company == "" {
		return ManagedContact{}, fmt.Errorf("contact name required")
	}
	if item.Email == "" && item.Phone == "" {
		return ManagedContact{}, fmt.Errorf("contact route required")
	}
	if item.Email != "" {
		if _, err := mail.ParseAddress(item.Email); err != nil {
			return ManagedContact{}, fmt.Errorf("invalid email")
		}
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now().UTC()
	} else {
		item.CreatedAt = item.CreatedAt.UTC()
	}
	if item.UpdatedAt.IsZero() {
		item.UpdatedAt = item.CreatedAt
	} else {
		item.UpdatedAt = item.UpdatedAt.UTC()
	}
	return item, nil
}

func NormalizeEnergyCapabilities(values []string) []string {
	allowed := map[string]bool{
		"metering":       true,
		"smart-meter":    true,
		"home-assistant": true,
		"pv":             true,
		"battery":        true,
		"wallbox":        true,
		"heat-pump":      true,
		"electrical":     true,
	}
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if !allowed[value] || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func NormalizeContactKind(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "dienstleister", "handwerker", "service":
		return "Dienstleister"
	case "energie-fachbetrieb", "energiefachbetrieb", "energy-specialist":
		return "Energie-Fachbetrieb"
	case "hausmeister", "caretaker":
		return "Hausmeister"
	case "notdienst", "emergency":
		return "Notdienst"
	case "verwaltung", "manager":
		return "Verwaltung"
	case "sonstiges", "other":
		return "Sonstiges"
	default:
		return ""
	}
}

func SortManagedContacts(items []ManagedContact) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Active != items[j].Active {
			return items[i].Active
		}
		if items[i].Kind != items[j].Kind {
			return items[i].Kind < items[j].Kind
		}
		left := strings.ToLower(ManagedContactDisplayName(items[i]))
		right := strings.ToLower(ManagedContactDisplayName(items[j]))
		if left != right {
			return left < right
		}
		return items[i].ID < items[j].ID
	})
}

// SaveJSONAtomic is the store layer's single persistence primitive: marshal,
// write to a temp file, rename into place. The rename is what makes it atomic —
// a crash mid-write leaves the previous file intact rather than a truncated one.
//
// An empty path means "in-memory only" and is a no-op. Every store constructor
// accepts path == "" and the entire test suite relies on it.
//
// noun appears in the error messages ("could not encode <noun> data"), which is
// why it is a parameter rather than derived: it keeps the 17 stores' existing
// error strings byte-identical.
func SaveJSONAtomic(path string, v any, noun string) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("could not create %s data directory", noun)
	}
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("could not encode %s data", noun)
	}
	tmp := path + ".tmp"
	// Write + fsync the temp file, then rename, then fsync the directory.
	// Without the fsyncs the rename is atomic but NOT durable: a power cut can
	// leave a zero-length/garbage file or revert the rename, even though the
	// user was told "saved" (HAUSV-137).
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("could not write %s data", noun)
	}
	if _, err := f.Write(raw); err != nil {
		f.Close()
		return fmt.Errorf("could not write %s data", noun)
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return fmt.Errorf("could not write %s data", noun)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("could not write %s data", noun)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("could not replace %s data", noun)
	}
	syncDir(filepath.Dir(path))
	return nil
}

// syncDir fsyncs a directory so a rename into it becomes durable. Best-effort:
// some filesystems don't support directory fsync, and a failure here should not
// fail an otherwise-successful write.
func syncDir(dir string) {
	d, err := os.Open(dir)
	if err != nil {
		return
	}
	_ = d.Sync()
	_ = d.Close()
}

type ActivityStore struct {
	path string
	mu   sync.Mutex
	data map[string]ActivityRecord
}

type ActivityRecord struct {
	LastLogin  time.Time `json:"last_login"`
	AuthMethod string    `json:"auth_method,omitempty"`
}

func NewActivityStore(path string) (*ActivityStore, error) {
	store := &ActivityStore{path: path, data: map[string]ActivityRecord{}}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read activity data")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("invalid activity data")
	}
	if store.data == nil {
		store.data = map[string]ActivityRecord{}
	}
	return store, nil
}

// Touch records a successful login. Best-effort: callers log failures but do
// not block login on a persistence error.
func (s *ActivityStore) Touch(email string, at time.Time, authMethod string) error {
	email = textutil.Email(email)
	if email == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[email] = ActivityRecord{LastLogin: at.UTC(), AuthMethod: authMethod}
	return s.saveLocked()
}

func (s *ActivityStore) Get(email string) (ActivityRecord, bool) {
	email = textutil.Email(email)
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.data[email]
	return rec, ok
}

func (s *ActivityStore) saveLocked() error {
	return SaveJSONAtomic(s.path, s.data, "activity")
}

func NormalizeUnitID(raw string) string {
	return textutil.UnitID(raw)
}

// randomToken mints contact-book IDs. A package-local copy (identical to the
// one in main) so the store depends on nothing above it.
func randomToken(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
