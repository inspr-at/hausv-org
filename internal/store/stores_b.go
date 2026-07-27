package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/markus-barta/hausv-org/internal/textutil"
)

// Handover audit actions. They live here with the rest of the audit vocabulary;
// handover.go aliases them.
const (
	AuditActionHandoverCreate  = "handover.create"
	AuditActionHandoverConfirm = "handover.confirm"
	AuditActionHandoverFile    = "handover.file"
)

const (
	AuditActionLogin              = "login"
	AuditActionInviteCreate       = "invite.create"
	AuditActionInviteUpdate       = "invite.update"
	AuditActionInviteDelete       = "invite.delete"
	AuditActionBuildingUpdate     = "building.update"
	AuditActionHeroUpdate         = "building.hero"
	AuditActionUnitSave           = "building.unit.save"
	AuditActionUnitDelete         = "building.unit.delete"
	AuditActionUnitPayment        = "building.unit.payment"
	AuditActionParkingSettings    = "parking.settings"
	AuditActionParkingMonth       = "parking.month"
	AuditActionParkingReminder    = "parking.reminder"
	AuditActionChargingSettings   = "charging.settings"
	AuditActionChargingManual     = "charging.manual"
	AuditActionChargingSession    = "charging.session"
	AuditActionIssueWorkflow      = "issue.workflow"
	AuditActionIssueEstimate      = "issue.estimate"
	AuditActionIssueServiceAdd    = "issue.service.add"
	AuditActionIssueServiceDrop   = "issue.service.drop"
	AuditActionIssueComment       = "issue.comment"
	AuditActionIssueCommentDelete = "issue.comment.delete"
	AuditActionEventCreate        = "event.create"
	AuditActionEventUpdate        = "event.update"
	AuditActionEventDelete        = "event.delete"
	AuditActionContactSave        = "contact.save"
	AuditActionContactDelete      = "contact.delete"
	AuditActionDocumentUpload     = "document.upload"
	AuditActionDocumentDownload   = "document.download"
	AuditActionDocumentReplace    = "document.replace"
	AuditActionAttachmentView     = "attachment.view"
	AuditActionAttachmentDelete   = "attachment.delete"
	AuditActionIntegrationImport  = "integration.import"
	AuditActionIntegrationExport  = "integration.export"
	AuditActionVoteCreate         = "vote.create"
	AuditActionVoteOpen           = "vote.open"
	AuditActionVoteClose          = "vote.close"
	AuditActionVoteCast           = "vote.cast"
	AuditActionVoteReminder       = "vote.reminder"
)

const (
	BallotTypeMeeting  = "Versammlung"
	BallotTypeCircular = "Umlaufbeschluss"

	BallotWeightingPerShare = "per-share"
	BallotWeightingPerHead  = "per-head"

	BallotStatusDraft  = "Entwurf"
	BallotStatusOpen   = "Offen"
	BallotStatusClosed = "Geschlossen"

	DefaultBallotReminderBeforeMinutes = 24 * 60
	MaxBallotReminderBeforeMinutes     = 30 * 24 * 60
)

type AnnouncementStore struct {
	mu   sync.Mutex
	path string
	data AnnouncementStoreData
}

type AnnouncementStoreData struct {
	Announcements []Announcement `json:"announcements"`
}

type EventStore struct {
	mu   sync.Mutex
	path string
	data EventStoreData
}

type EventStoreData struct {
	Events []HouseEvent `json:"events"`
}

type Announcement struct {
	ID          string     `json:"id"`
	TenantSlug  string     `json:"tenant"`
	Title       string     `json:"title"`
	Body        string     `json:"body"`
	Category    string     `json:"category"`
	Pinned      bool       `json:"pinned"`
	PublishedAt time.Time  `json:"published_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	AuthorEmail string     `json:"author_email"`
	AuthorName  string     `json:"author_name"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type HouseEvent struct {
	ID          string     `json:"id"`
	TenantSlug  string     `json:"tenant"`
	Title       string     `json:"title"`
	Body        string     `json:"body,omitempty"`
	Category    string     `json:"category"`
	Location    string     `json:"location,omitempty"`
	StartsAt    time.Time  `json:"starts_at"`
	EndsAt      *time.Time `json:"ends_at,omitempty"`
	AuthorEmail string     `json:"author_email"`
	AuthorName  string     `json:"author_name"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type VoteStore struct {
	mu   sync.Mutex
	path string
	data VoteStoreData
}

type VoteStoreData struct {
	Ballots []Ballot `json:"ballots"`
}

type Ballot struct {
	ID                    string                `json:"id"`
	TenantSlug            string                `json:"tenant"`
	Title                 string                `json:"title"`
	Description           string                `json:"description,omitempty"`
	Options               []string              `json:"options"`
	Type                  string                `json:"type"`
	Weighting             string                `json:"weighting"`
	QuorumPPM             int                   `json:"quorum_ppm"`
	OpensAt               time.Time             `json:"opens_at,omitempty"`
	ClosesAt              time.Time             `json:"closes_at,omitempty"`
	CreatedBy             string                `json:"created_by"`
	CreatedAt             time.Time             `json:"created_at"`
	UpdatedAt             time.Time             `json:"updated_at"`
	Status                string                `json:"status"`
	Votes                 map[string]BallotVote `json:"votes,omitempty"`
	ReminderBeforeMinutes int                   `json:"reminder_before_minutes,omitempty"`
	ReminderSentAt        map[string]time.Time  `json:"reminder_sent_at,omitempty"`
}

type BallotVote struct {
	Option string    `json:"option"`
	Weight int       `json:"weight"`
	At     time.Time `json:"at"`
}

type UnitStore struct {
	mu   sync.Mutex
	path string
	data UnitStoreData
}

type UnitStoreData struct {
	Units []Unit `json:"units"`
}

type Unit struct {
	ID                    string   `json:"id"`
	TenantSlug            string   `json:"tenant"`
	Label                 string   `json:"label"`
	UnitType              string   `json:"unit_type,omitempty"`
	BillableWeightPPM     int      `json:"billable_weight_ppm,omitempty"`
	MiteigentumsanteilPPM int      `json:"miteigentumsanteil"`
	OwnerEmails           []string `json:"owner_emails,omitempty"`
	RenterEmails          []string `json:"renter_emails,omitempty"`
}

const (
	UnitTypeResidential = "residential"
	UnitTypeCommercial  = "commercial"
	UnitTypeParking     = "parking"
	UnitTypeStorage     = "storage"
	UnitTypeOther       = "other"

	UnitBillableFullPPM = 1_000_000

	// FairUseFreeUnits is the number of billable Wohneinheiten included free. It
	// is an informational fair-use marker (HAUSV-125), not an enforced limit.
	FairUseFreeUnits = 25
)

type UnitMembership struct {
	Unit     Unit
	Relation string
}

type AuditEvent struct {
	At         time.Time         `json:"at"`
	TenantSlug string            `json:"tenant"`
	ActorEmail string            `json:"actor_email"`
	ActorRole  string            `json:"actor_role,omitempty"`
	Action     string            `json:"action"`
	TargetType string            `json:"target_type,omitempty"`
	TargetID   string            `json:"target_id,omitempty"`
	Summary    string            `json:"summary"`
	Details    map[string]string `json:"details,omitempty"`
}

type AuditFilter struct {
	TenantSlug string
	Action     string
	Query      string
	Limit      int
}

type UnitMembers struct {
	Unit    Unit
	Owners  []string
	Renters []string
	Found   bool
}

func NormalizeEventCategory(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "versammlung", "eigentuemerversammlung", "eigentümerversammlung", "versammlung der eigentümer", "meeting":
		return "Eigentümerversammlung"
	case "reinigung", "cleaning":
		return "Reinigung"
	case "wartung", "maintenance":
		return "Wartung"
	case "ablesung", "ablesetermin", "reading":
		return "Ablesung"
	case "frist", "deadline":
		return "Frist"
	default:
		return "Sonstiges"
	}
}

func NormalizeAnnouncementCategory(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "dringend", "urgent":
		return "Dringend"
	case "termin", "date", "event":
		return "Termin"
	case "wartung", "maintenance":
		return "Wartung"
	default:
		return "Info"
	}
}

func EventRollsOffAt(item HouseEvent) time.Time {
	if item.EndsAt != nil {
		return *item.EndsAt
	}
	startLocal := item.StartsAt.In(time.Local)
	year, month, day := startLocal.Date()
	return time.Date(year, month, day, 23, 59, 59, 0, time.Local).UTC()
}

func NewAnnouncementStore(path string) (*AnnouncementStore, error) {
	store := &AnnouncementStore{path: path, data: AnnouncementStoreData{Announcements: []Announcement{}}}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read Announcement data")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("invalid Announcement data")
	}
	return store, nil
}

func NewEventStore(path string) (*EventStore, error) {
	store := &EventStore{path: path, data: EventStoreData{Events: []HouseEvent{}}}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read event data")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("invalid event data")
	}
	events := make([]HouseEvent, 0, len(store.data.Events))
	for _, item := range store.data.Events {
		normalized, ok := NormalizeHouseEvent(item)
		if ok {
			events = append(events, normalized)
		}
	}
	store.data.Events = events
	SortEvents(store.data.Events)
	return store, nil
}

func NewVoteStore(path string) (*VoteStore, error) {
	store := &VoteStore{path: path, data: VoteStoreData{Ballots: []Ballot{}}}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read vote data")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("invalid vote data")
	}
	store.data.Ballots = NormalizeBallots(store.data.Ballots)
	return store, nil
}

func (s *VoteStore) Create(item Ballot) (Ballot, error) {
	if s == nil {
		return Ballot{}, fmt.Errorf("vote store unavailable")
	}
	now := time.Now().UTC()
	id, err := randomToken(12)
	if err != nil {
		return Ballot{}, err
	}
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
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Ballots = append(s.data.Ballots, item)
	SortBallots(s.data.Ballots)
	if err := s.saveLocked(); err != nil {
		return Ballot{}, err
	}
	return CopyBallot(item), nil
}

func (s *VoteStore) Delete(tenantSlug string, id string) (bool, error) {
	if s == nil {
		return false, nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	if tenantSlug == "" || id == "" {
		return false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, item := range s.data.Ballots {
		if textutil.Slug(item.TenantSlug) != tenantSlug || item.ID != id {
			continue
		}
		s.data.Ballots = append(s.data.Ballots[:i], s.data.Ballots[i+1:]...)
		if err := s.saveLocked(); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (s *VoteStore) Open(tenantSlug string, id string, at time.Time) (Ballot, bool, error) {
	return s.setStatus(tenantSlug, id, BallotStatusOpen, at)
}

func (s *VoteStore) Close(tenantSlug string, id string, at time.Time) (Ballot, bool, error) {
	return s.setStatus(tenantSlug, id, BallotStatusClosed, at)
}

func (s *VoteStore) CloseExpiredTenant(tenantSlug string, at time.Time) ([]Ballot, error) {
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
	s.mu.Lock()
	defer s.mu.Unlock()
	closed := []Ballot{}
	changed := false
	for i, item := range s.data.Ballots {
		item = NormalizeBallot(item)
		if textutil.Slug(item.TenantSlug) != tenantSlug || item.Status != BallotStatusOpen || item.ClosesAt.IsZero() || at.Before(item.ClosesAt) {
			continue
		}
		item.Status = BallotStatusClosed
		item.UpdatedAt = at
		item = NormalizeBallot(item)
		s.data.Ballots[i] = item
		closed = append(closed, CopyBallot(item))
		changed = true
	}
	if !changed {
		return nil, nil
	}
	SortBallots(s.data.Ballots)
	if err := s.saveLocked(); err != nil {
		return nil, err
	}
	return closed, nil
}

func (s *VoteStore) setStatus(tenantSlug string, id string, status string, at time.Time) (Ballot, bool, error) {
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
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, item := range s.data.Ballots {
		if textutil.Slug(item.TenantSlug) != tenantSlug || item.ID != id {
			continue
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
		s.data.Ballots[i] = item
		SortBallots(s.data.Ballots)
		if err := s.saveLocked(); err != nil {
			return Ballot{}, true, err
		}
		return CopyBallot(item), true, nil
	}
	return Ballot{}, false, nil
}

func (s *VoteStore) CastVote(tenantSlug string, id string, email string, option string, weight int, at time.Time) (Ballot, bool, error) {
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
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, item := range s.data.Ballots {
		if textutil.Slug(item.TenantSlug) != tenantSlug || item.ID != id {
			continue
		}
		item = NormalizeBallot(item)
		if item.Status != BallotStatusOpen {
			return Ballot{}, true, fmt.Errorf("Ballot is not open")
		}
		if !item.OpensAt.IsZero() && at.Before(item.OpensAt) {
			return Ballot{}, true, fmt.Errorf("Ballot is not open yet")
		}
		if !item.ClosesAt.IsZero() && !at.Before(item.ClosesAt) {
			item.Status = BallotStatusClosed
			item.UpdatedAt = at
			item = NormalizeBallot(item)
			s.data.Ballots[i] = item
			SortBallots(s.data.Ballots)
			if err := s.saveLocked(); err != nil {
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
		s.data.Ballots[i] = item
		SortBallots(s.data.Ballots)
		if err := s.saveLocked(); err != nil {
			return Ballot{}, true, err
		}
		return CopyBallot(item), true, nil
	}
	return Ballot{}, false, nil
}

func (s *VoteStore) MarkReminderSent(tenantSlug string, id string, recipients []string, at time.Time) (Ballot, bool, error) {
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
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, item := range s.data.Ballots {
		if textutil.Slug(item.TenantSlug) != tenantSlug || item.ID != id {
			continue
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
		s.data.Ballots[i] = item
		SortBallots(s.data.Ballots)
		if err := s.saveLocked(); err != nil {
			return Ballot{}, true, err
		}
		return CopyBallot(item), true, nil
	}
	return Ballot{}, false, nil
}

func (s *VoteStore) ListTenant(tenantSlug string) []Ballot {
	if s == nil {
		return nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Ballot{}
	for _, item := range s.data.Ballots {
		if textutil.Slug(item.TenantSlug) == tenantSlug {
			out = append(out, CopyBallot(item))
		}
	}
	SortBallots(out)
	return out
}

func (s *VoteStore) Get(tenantSlug string, id string) (Ballot, bool) {
	if s == nil {
		return Ballot{}, false
	}
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	if tenantSlug == "" || id == "" {
		return Ballot{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.data.Ballots {
		if textutil.Slug(item.TenantSlug) == tenantSlug && item.ID == id {
			return CopyBallot(NormalizeBallot(item)), true
		}
	}
	return Ballot{}, false
}

func (s *VoteStore) saveLocked() error {
	return SaveJSONAtomic(s.path, s.data, "vote")
}

func NormalizeBallots(items []Ballot) []Ballot {
	out := make([]Ballot, 0, len(items))
	for _, item := range items {
		item = NormalizeBallot(item)
		if item.ID == "" || item.TenantSlug == "" || item.Title == "" || len(item.Options) < 2 || item.Type == "" || item.Weighting == "" {
			continue
		}
		out = append(out, item)
	}
	SortBallots(out)
	return out
}

func NormalizeBallot(item Ballot) Ballot {
	item.ID = strings.TrimSpace(item.ID)
	item.TenantSlug = textutil.Slug(item.TenantSlug)
	item.Title = TruncateAuditValue(strings.TrimSpace(item.Title), 160)
	item.Description = TruncateAuditValue(strings.TrimSpace(item.Description), 5000)
	item.Options = NormalizeBallotOptions(item.Options)
	item.Type = NormalizeBallotType(item.Type)
	item.Weighting = NormalizeBallotWeighting(item.Weighting)
	item.Status = NormalizeBallotStatus(item.Status)
	item.CreatedBy = textutil.Email(item.CreatedBy)
	if item.QuorumPPM < 0 {
		item.QuorumPPM = 0
	}
	if item.QuorumPPM > 1_000_000 {
		item.QuorumPPM = 1_000_000
	}
	if !item.OpensAt.IsZero() {
		item.OpensAt = item.OpensAt.UTC().Truncate(time.Second)
	}
	if !item.ClosesAt.IsZero() {
		item.ClosesAt = item.ClosesAt.UTC().Truncate(time.Second)
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}
	item.CreatedAt = item.CreatedAt.UTC().Truncate(time.Second)
	if item.UpdatedAt.IsZero() {
		item.UpdatedAt = item.CreatedAt
	}
	item.UpdatedAt = item.UpdatedAt.UTC().Truncate(time.Second)
	if item.ReminderBeforeMinutes <= 0 {
		item.ReminderBeforeMinutes = DefaultBallotReminderBeforeMinutes
	}
	if item.ReminderBeforeMinutes > MaxBallotReminderBeforeMinutes {
		item.ReminderBeforeMinutes = MaxBallotReminderBeforeMinutes
	}
	if len(item.Votes) == 0 {
		item.Votes = nil
	} else {
		votes := map[string]BallotVote{}
		for email, vote := range item.Votes {
			email = textutil.Email(email)
			vote.Option = strings.TrimSpace(vote.Option)
			if email == "" || vote.Option == "" || vote.Weight <= 0 || !BallotHasOption(item, vote.Option) {
				continue
			}
			if vote.At.IsZero() {
				vote.At = item.UpdatedAt
			}
			vote.At = vote.At.UTC().Truncate(time.Second)
			votes[email] = vote
		}
		item.Votes = votes
		if len(item.Votes) == 0 {
			item.Votes = nil
		}
	}
	if len(item.ReminderSentAt) == 0 {
		item.ReminderSentAt = nil
	} else {
		sent := map[string]time.Time{}
		for email, at := range item.ReminderSentAt {
			email = textutil.Email(email)
			if email == "" {
				continue
			}
			if at.IsZero() {
				at = item.UpdatedAt
			}
			sent[email] = at.UTC().Truncate(time.Second)
		}
		item.ReminderSentAt = sent
		if len(item.ReminderSentAt) == 0 {
			item.ReminderSentAt = nil
		}
	}
	return item
}

func NormalizeBallotOptions(raw []string) []string {
	out := []string{}
	seen := map[string]struct{}{}
	for _, option := range raw {
		option = TruncateAuditValue(strings.TrimSpace(option), 120)
		if option == "" {
			continue
		}
		key := strings.ToLower(option)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, option)
		if len(out) >= 12 {
			break
		}
	}
	return out
}

func NormalizeBallotType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "versammlung", "meeting", "eigentuemerversammlung", "eigentümerversammlung":
		return BallotTypeMeeting
	case "", "umlauf", "umlaufbeschluss", "circular", "resolution":
		return BallotTypeCircular
	default:
		return ""
	}
}

func NormalizeBallotWeighting(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "per-head", "head", "kopf", "pro-kopf":
		return BallotWeightingPerHead
	case "", "per-share", "share", "anteil", "miteigentumsanteil":
		return BallotWeightingPerShare
	default:
		return ""
	}
}

func NormalizeBallotStatus(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "entwurf", "draft":
		return BallotStatusDraft
	case "offen", "open":
		return BallotStatusOpen
	case "geschlossen", "closed":
		return BallotStatusClosed
	default:
		return ""
	}
}

func BallotHasOption(item Ballot, option string) bool {
	option = strings.TrimSpace(option)
	for _, existing := range item.Options {
		if existing == option {
			return true
		}
	}
	return false
}

func CopyBallot(item Ballot) Ballot {
	item.Options = append([]string(nil), item.Options...)
	if len(item.Votes) > 0 {
		votes := map[string]BallotVote{}
		for email, vote := range item.Votes {
			votes[email] = vote
		}
		item.Votes = votes
	}
	if len(item.ReminderSentAt) > 0 {
		sent := map[string]time.Time{}
		for email, at := range item.ReminderSentAt {
			sent[email] = at
		}
		item.ReminderSentAt = sent
	}
	return item
}

func SortBallots(items []Ballot) {
	sort.SliceStable(items, func(i, j int) bool {
		if !items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].UpdatedAt.After(items[j].UpdatedAt)
		}
		return strings.ToLower(items[i].Title) < strings.ToLower(items[j].Title)
	})
}

func NewUnitStore(path string) (*UnitStore, error) {
	store := &UnitStore{path: path, data: UnitStoreData{Units: []Unit{}}}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read Unit data")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("invalid Unit data")
	}
	store.data.Units = NormalizeUnits(store.data.Units, "")
	return store, nil
}

func (s *UnitStore) SetTenantUnits(tenantSlug string, units []Unit) error {
	if s == nil {
		return nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	if tenantSlug == "" {
		return nil
	}
	normalized := NormalizeUnits(units, tenantSlug)
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := s.data.Units[:0]
	for _, existing := range s.data.Units {
		if textutil.Slug(existing.TenantSlug) != tenantSlug {
			kept = append(kept, existing)
		}
	}
	s.data.Units = append(kept, normalized...)
	SortUnits(s.data.Units)
	return s.saveLocked()
}

// UpsertUnit adds or replaces a single unit within ONE lock acquisition, so a
// concurrent add/delete of a different unit is not clobbered by a whole-slice
// overwrite (HAUSV-145). origID is the unit's previous ID ("" for a new unit).
// It returns duplicate=true if the target ID collides with a different existing
// unit — mirroring the handler's original check exactly.
func (s *UnitStore) UpsertUnit(tenantSlug, origID string, item Unit) (duplicate bool, err error) {
	if s == nil {
		return false, nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	if tenantSlug == "" {
		return false, nil
	}
	item.TenantSlug = tenantSlug
	wasCreate := origID == ""

	s.mu.Lock()
	defer s.mu.Unlock()

	var mine, others []Unit
	for _, u := range s.data.Units {
		if textutil.Slug(u.TenantSlug) == tenantSlug {
			mine = append(mine, u)
		} else {
			others = append(others, u)
		}
	}
	// Duplicate iff an existing unit already has the target ID and we are either
	// creating or renaming onto it (not editing that same unit in place).
	for _, u := range mine {
		if u.ID == item.ID && (wasCreate || origID != item.ID) {
			return true, nil
		}
	}
	if wasCreate {
		origID = item.ID
	}
	replaced := false
	for i := range mine {
		if mine[i].ID == origID {
			mine[i] = item
			replaced = true
			break
		}
	}
	if !replaced {
		mine = append(mine, item)
	}
	s.data.Units = append(others, NormalizeUnits(mine, tenantSlug)...)
	SortUnits(s.data.Units)
	return false, s.saveLocked()
}

// DeleteUnit removes one unit within one lock acquisition (HAUSV-145). Returns
// removed=false if no unit had that ID.
func (s *UnitStore) DeleteUnit(tenantSlug, id string) (removed bool, removedUnit Unit, err error) {
	if s == nil {
		return false, Unit{}, nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	if tenantSlug == "" {
		return false, Unit{}, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	kept := make([]Unit, 0, len(s.data.Units))
	for _, u := range s.data.Units {
		if textutil.Slug(u.TenantSlug) == tenantSlug && u.ID == id {
			removed = true
			removedUnit = u
			continue
		}
		kept = append(kept, u)
	}
	if !removed {
		return false, Unit{}, nil
	}
	s.data.Units = kept
	SortUnits(s.data.Units)
	return true, removedUnit, s.saveLocked()
}

func (s *UnitStore) ListTenant(tenantSlug string) []Unit {
	if s == nil {
		return nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Unit{}
	for _, item := range s.data.Units {
		if textutil.Slug(item.TenantSlug) == tenantSlug {
			out = append(out, CopyUnit(item))
		}
	}
	SortUnits(out)
	return out
}

func (s *UnitStore) UnitCount(tenantSlug string) int {
	return len(s.ListTenant(tenantSlug))
}

func (s *UnitStore) BillableUnitWeight(tenantSlug string) int {
	return BillableUnitWeight(s.ListTenant(tenantSlug))
}

func BillableUnitWeight(units []Unit) int {
	total := 0
	for _, item := range units {
		total += NormalizeUnitBillableWeight(item.UnitType, item.BillableWeightPPM)
	}
	return total
}

func (s *UnitStore) UnitsForEmail(tenantSlug string, email string) []UnitMembership {
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
	out := []UnitMembership{}
	for _, item := range s.data.Units {
		if textutil.Slug(item.TenantSlug) != tenantSlug {
			continue
		}
		relation := ""
		if EmailListContains(item.OwnerEmails, email) {
			relation = RoleOwner
		} else if EmailListContains(item.RenterEmails, email) {
			relation = RoleRenter
		}
		if relation != "" {
			out = append(out, UnitMembership{Unit: CopyUnit(item), Relation: relation})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return UnitLess(out[i].Unit, out[j].Unit)
	})
	return out
}

func (s *UnitStore) MembersForUnit(tenantSlug string, unitID string) UnitMembers {
	if s == nil {
		return UnitMembers{}
	}
	tenantSlug = textutil.Slug(tenantSlug)
	unitID = NormalizeUnitID(unitID)
	if tenantSlug == "" || unitID == "" {
		return UnitMembers{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.data.Units {
		if textutil.Slug(item.TenantSlug) == tenantSlug && textutil.Slug(item.ID) == unitID {
			item = CopyUnit(item)
			return UnitMembers{Unit: item, Owners: append([]string(nil), item.OwnerEmails...), Renters: append([]string(nil), item.RenterEmails...), Found: true}
		}
	}
	return UnitMembers{}
}

func (s *UnitStore) saveLocked() error {
	return SaveJSONAtomic(s.path, s.data, "Unit")
}

func (s *EventStore) Create(item HouseEvent) (HouseEvent, error) {
	if s == nil {
		return item, nil
	}
	now := time.Now().UTC()
	item.ID = ""
	item.CreatedAt = now
	item.UpdatedAt = now
	normalized, ok := NormalizeHouseEvent(item)
	if !ok {
		return HouseEvent{}, fmt.Errorf("invalid event")
	}
	id, err := randomToken(12)
	if err != nil {
		return HouseEvent{}, err
	}
	normalized.ID = id
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Events = append(s.data.Events, normalized)
	SortEvents(s.data.Events)
	if err := s.saveLocked(); err != nil {
		return HouseEvent{}, err
	}
	return normalized, nil
}

func (s *EventStore) Update(id string, updated HouseEvent) (bool, error) {
	if s == nil {
		return false, nil
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return false, nil
	}
	normalized, ok := NormalizeHouseEvent(updated)
	if !ok {
		return false, fmt.Errorf("invalid event")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.data.Events {
		if existing.ID != id || textutil.Slug(existing.TenantSlug) != textutil.Slug(normalized.TenantSlug) {
			continue
		}
		normalized.ID = existing.ID
		normalized.CreatedAt = existing.CreatedAt
		normalized.UpdatedAt = time.Now().UTC()
		if normalized.AuthorEmail == "" {
			normalized.AuthorEmail = existing.AuthorEmail
		}
		if normalized.AuthorName == "" {
			normalized.AuthorName = existing.AuthorName
		}
		s.data.Events[i] = normalized
		SortEvents(s.data.Events)
		if err := s.saveLocked(); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (s *EventStore) Delete(tenantSlug string, id string) (bool, error) {
	if s == nil {
		return false, nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	if tenantSlug == "" || id == "" {
		return false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := s.data.Events[:0]
	removed := false
	for _, item := range s.data.Events {
		if item.ID == id && textutil.Slug(item.TenantSlug) == tenantSlug {
			removed = true
			continue
		}
		kept = append(kept, item)
	}
	if !removed {
		return false, nil
	}
	s.data.Events = kept
	if err := s.saveLocked(); err != nil {
		return false, err
	}
	return true, nil
}

func (s *EventStore) ListTenant(tenantSlug string) []HouseEvent {
	if s == nil {
		return nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []HouseEvent{}
	for _, item := range s.data.Events {
		if textutil.Slug(item.TenantSlug) == tenantSlug {
			out = append(out, CopyEvent(item))
		}
	}
	SortEvents(out)
	return out
}

func (s *EventStore) Upcoming(tenantSlug string, now time.Time) []HouseEvent {
	tenantSlug = textutil.Slug(tenantSlug)
	items := s.ListTenant(tenantSlug)
	out := []HouseEvent{}
	for _, item := range items {
		if EventRollsOffAt(item).After(now) {
			out = append(out, item)
		}
	}
	SortEvents(out)
	return out
}

func (s *EventStore) saveLocked() error {
	return SaveJSONAtomic(s.path, s.data, "event")
}

func NormalizeHouseEvent(item HouseEvent) (HouseEvent, bool) {
	item.ID = strings.TrimSpace(item.ID)
	item.TenantSlug = textutil.Slug(item.TenantSlug)
	item.Title = strings.TrimSpace(item.Title)
	item.Body = strings.TrimSpace(item.Body)
	item.Category = NormalizeEventCategory(item.Category)
	item.Location = strings.TrimSpace(item.Location)
	item.AuthorEmail = textutil.Email(item.AuthorEmail)
	item.AuthorName = strings.TrimSpace(item.AuthorName)
	item.StartsAt = item.StartsAt.UTC().Truncate(time.Second)
	if item.EndsAt != nil {
		endsAt := item.EndsAt.UTC().Truncate(time.Second)
		if !endsAt.After(item.StartsAt) {
			return HouseEvent{}, false
		}
		item.EndsAt = &endsAt
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now().UTC()
	} else {
		item.CreatedAt = item.CreatedAt.UTC().Truncate(time.Second)
	}
	if item.UpdatedAt.IsZero() {
		item.UpdatedAt = item.CreatedAt
	} else {
		item.UpdatedAt = item.UpdatedAt.UTC().Truncate(time.Second)
	}
	if item.TenantSlug == "" || item.Title == "" || item.StartsAt.IsZero() {
		return HouseEvent{}, false
	}
	return item, true
}

func CopyEvent(item HouseEvent) HouseEvent {
	if item.EndsAt != nil {
		endsAt := *item.EndsAt
		item.EndsAt = &endsAt
	}
	return item
}

func SortEvents(items []HouseEvent) {
	sort.SliceStable(items, func(i, j int) bool {
		if !items[i].StartsAt.Equal(items[j].StartsAt) {
			return items[i].StartsAt.Before(items[j].StartsAt)
		}
		if strings.ToLower(items[i].Title) != strings.ToLower(items[j].Title) {
			return strings.ToLower(items[i].Title) < strings.ToLower(items[j].Title)
		}
		return items[i].ID < items[j].ID
	})
}

func (s *AnnouncementStore) Create(item Announcement) (Announcement, error) {
	now := time.Now().UTC()
	item.ID = ""
	item.CreatedAt = now
	item.UpdatedAt = now
	if item.PublishedAt.IsZero() {
		item.PublishedAt = now
	}
	if item.Category == "" {
		item.Category = "Info"
	}
	id, err := randomToken(12)
	if err != nil {
		return Announcement{}, err
	}
	item.ID = id
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Announcements = append(s.data.Announcements, item)
	if err := s.saveLocked(); err != nil {
		return Announcement{}, err
	}
	return item, nil
}

func (s *AnnouncementStore) Update(id string, updated Announcement) (bool, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.data.Announcements {
		if existing.ID != id || textutil.Slug(existing.TenantSlug) != textutil.Slug(updated.TenantSlug) {
			continue
		}
		updated.ID = existing.ID
		updated.CreatedAt = existing.CreatedAt
		updated.UpdatedAt = time.Now().UTC()
		if updated.PublishedAt.IsZero() {
			updated.PublishedAt = existing.PublishedAt
		}
		if updated.AuthorEmail == "" {
			updated.AuthorEmail = existing.AuthorEmail
		}
		if updated.AuthorName == "" {
			updated.AuthorName = existing.AuthorName
		}
		s.data.Announcements[i] = updated
		if err := s.saveLocked(); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (s *AnnouncementStore) Delete(tenantSlug string, id string) (bool, error) {
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	if id == "" {
		return false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := s.data.Announcements[:0]
	removed := false
	for _, item := range s.data.Announcements {
		if item.ID == id && textutil.Slug(item.TenantSlug) == tenantSlug {
			removed = true
			continue
		}
		kept = append(kept, item)
	}
	if !removed {
		return false, nil
	}
	s.data.Announcements = kept
	if err := s.saveLocked(); err != nil {
		return false, err
	}
	return true, nil
}

func (s *AnnouncementStore) Visible(tenantSlug string, now time.Time) []Announcement {
	tenantSlug = textutil.Slug(tenantSlug)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Announcement{}
	for _, item := range s.data.Announcements {
		if textutil.Slug(item.TenantSlug) != tenantSlug {
			continue
		}
		if item.PublishedAt.After(now) {
			continue
		}
		if item.ExpiresAt != nil && !item.ExpiresAt.After(now) {
			continue
		}
		out = append(out, item)
	}
	SortAnnouncements(out)
	return out
}

func (s *AnnouncementStore) Archive(tenantSlug string, now time.Time) []Announcement {
	tenantSlug = textutil.Slug(tenantSlug)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Announcement{}
	for _, item := range s.data.Announcements {
		if textutil.Slug(item.TenantSlug) != tenantSlug {
			continue
		}
		if item.PublishedAt.After(now) {
			continue
		}
		out = append(out, item)
	}
	SortAnnouncements(out)
	return out
}

func (s *AnnouncementStore) ListTenant(tenantSlug string) []Announcement {
	tenantSlug = textutil.Slug(tenantSlug)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Announcement{}
	for _, item := range s.data.Announcements {
		if textutil.Slug(item.TenantSlug) == tenantSlug {
			out = append(out, item)
		}
	}
	SortAnnouncements(out)
	return out
}

func (s *AnnouncementStore) saveLocked() error {
	return SaveJSONAtomic(s.path, s.data, "Announcement")
}

func SortAnnouncements(items []Announcement) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Pinned != items[j].Pinned {
			return items[i].Pinned
		}
		if !items[i].PublishedAt.Equal(items[j].PublishedAt) {
			return items[i].PublishedAt.After(items[j].PublishedAt)
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
}

type AuditStore struct {
	path    string
	mu      sync.Mutex
	entries []AuditEvent
}

func NewAuditStore(path string) (*AuditStore, error) {
	store := &AuditStore{path: path, entries: []AuditEvent{}}
	if path == "" {
		return store, nil
	}
	if err := store.pruneArchivesLocked(time.Now()); err != nil {
		slog.Error("audit archive retention cleanup failed", "error", err)
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read audit data")
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return store, nil
	}
	if strings.HasPrefix(trimmed, "[") {
		if err := json.Unmarshal([]byte(trimmed), &store.entries); err != nil {
			return nil, fmt.Errorf("invalid audit data")
		}
		for i := range store.entries {
			store.entries[i] = NormalizeAuditEvent(store.entries[i])
		}
		return store, nil
	}
	lines := strings.Split(trimmed, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var event AuditEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			// A crash mid-append leaves a truncated final line. The audit log is
			// best-effort; it must not be able to prevent the whole app from
			// booting (HAUSV-136). Drop an unparseable LAST line; still fail on a
			// bad line in the middle, which signals real corruption, not a torn
			// append.
			if i == len(lines)-1 {
				slog.Warn("dropping unparseable trailing audit line", "line", i+1, "likely_cause", "torn_append", "error", err)
				break
			}
			return nil, fmt.Errorf("invalid audit data on line %d", i+1)
		}
		store.entries = append(store.entries, NormalizeAuditEvent(event))
	}
	return store, nil
}

func (s *AuditStore) Append(event AuditEvent) error {
	if s == nil {
		return nil
	}
	event = NormalizeAuditEvent(event)
	if event.Action == "" {
		return nil
	}
	raw, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("could not encode audit event")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.path != "" {
		if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
			return fmt.Errorf("could not create audit data directory")
		}
		f, err := os.OpenFile(s.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			return fmt.Errorf("could not open audit data")
		}
		if _, err := f.Write(append(raw, '\n')); err != nil {
			_ = f.Close()
			return fmt.Errorf("could not append audit data")
		}
		// fsync so an acknowledged audit write survives a crash, and so a crash
		// mid-append can only lose a whole line, not corrupt one (HAUSV-136/137).
		if err := f.Sync(); err != nil {
			_ = f.Close()
			return fmt.Errorf("could not sync audit data")
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("could not close audit data")
		}
	}
	s.entries = append(s.entries, CopyAuditEvent(event))
	if s.path != "" && s.shouldRotateLocked(time.Now()) {
		if err := s.rotateLocked(); err != nil {
			// The event is already durably appended; a rotation failure must not
			// fail the write. Growth continues until the next successful rotate.
			slog.Error("audit rotation failed", "error", err)
		}
	}
	return nil
}

// Rotation bounds the live audit file and the in-memory slice by count, size
// and age (HAUSV-146). Archives then have a separate retention ceiling. Vars,
// not consts, so tests can exercise the policy without large files or clocks.
var (
	auditRotateThreshold  = 20000
	auditRotateKeep       = 5000
	auditRotateMaxBytes   = int64(10 << 20)
	auditRotateMaxAge     = 90 * 24 * time.Hour
	auditArchiveRetention = 3 * 365 * 24 * time.Hour
)

func (s *AuditStore) shouldRotateLocked(now time.Time) bool {
	if len(s.entries) > auditRotateThreshold {
		return true
	}
	if info, err := os.Stat(s.path); err == nil && info.Size() >= auditRotateMaxBytes {
		return true
	}
	if len(s.entries) == 0 || auditRotateMaxAge <= 0 {
		return false
	}
	oldest := s.entries[0].At
	return !oldest.IsZero() && now.Sub(oldest) >= auditRotateMaxAge
}

// rotateLocked archives the current live file and rewrites it with only the
// recent tail. Archives older than auditArchiveRetention are removed. Must be
// called with s.mu held.
func (s *AuditStore) rotateLocked() error {
	now := time.Now()
	archive := fmt.Sprintf("%s.%d", s.path, now.UnixNano())
	if err := os.Rename(s.path, archive); err != nil {
		return err
	}
	if auditRotateMaxAge > 0 {
		cutoff := now.Add(-auditRotateMaxAge)
		recent := s.entries[:0]
		for _, event := range s.entries {
			if !event.At.Before(cutoff) {
				recent = append(recent, event)
			}
		}
		s.entries = recent
	}
	if len(s.entries) > auditRotateKeep {
		s.entries = append([]AuditEvent(nil), s.entries[len(s.entries)-auditRotateKeep:]...)
	}
	f, err := os.OpenFile(s.path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, e := range s.entries {
		raw, err := json.Marshal(e)
		if err != nil {
			return err
		}
		if _, err := f.Write(append(raw, '\n')); err != nil {
			return err
		}
	}
	if err := f.Sync(); err != nil {
		return err
	}
	return s.pruneArchivesLocked(now)
}

func (s *AuditStore) pruneArchivesLocked(now time.Time) error {
	if s.path == "" || auditArchiveRetention <= 0 {
		return nil
	}
	entries, err := os.ReadDir(filepath.Dir(s.path))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	prefix := filepath.Base(s.path) + "."
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if now.Sub(info.ModTime()) < auditArchiveRetention {
			continue
		}
		if err := os.Remove(filepath.Join(filepath.Dir(s.path), entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func (s *AuditStore) List(filter AuditFilter) []AuditEvent {
	if s == nil {
		return nil
	}
	filter.TenantSlug = textutil.Slug(filter.TenantSlug)
	filter.Action = NormalizeAuditAction(filter.Action)
	filter.Query = strings.ToLower(strings.TrimSpace(filter.Query))
	if filter.Limit <= 0 || filter.Limit > 500 {
		filter.Limit = 200
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]AuditEvent, 0, min(filter.Limit, len(s.entries)))
	for i := len(s.entries) - 1; i >= 0 && len(out) < filter.Limit; i-- {
		event := s.entries[i]
		if filter.TenantSlug != "" && textutil.Slug(event.TenantSlug) != filter.TenantSlug {
			continue
		}
		if filter.Action != "" && NormalizeAuditAction(event.Action) != filter.Action {
			continue
		}
		if filter.Query != "" && !AuditEventMatches(event, filter.Query) {
			continue
		}
		out = append(out, CopyAuditEvent(event))
	}
	return out
}

// HasTarget checks the complete live audit ledger rather than the paginated
// presentation view. Import handlers use it as their durable idempotency key:
// a successfully recorded source file must never be applied a second time.
func (s *AuditStore) HasTarget(tenantSlug string, action string, targetID string) bool {
	if s == nil {
		return false
	}
	tenantSlug = textutil.Slug(tenantSlug)
	action = NormalizeAuditAction(action)
	targetID = strings.TrimSpace(targetID)
	if tenantSlug == "" || action == "" || targetID == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := len(s.entries) - 1; i >= 0; i-- {
		event := s.entries[i]
		if textutil.Slug(event.TenantSlug) == tenantSlug &&
			NormalizeAuditAction(event.Action) == action &&
			strings.TrimSpace(event.TargetID) == targetID {
			return true
		}
	}
	return false
}

func NormalizeAuditEvent(event AuditEvent) AuditEvent {
	event.TenantSlug = textutil.Slug(event.TenantSlug)
	event.ActorEmail = textutil.Email(event.ActorEmail)
	event.ActorRole = NormalizeRole(event.ActorRole)
	event.Action = NormalizeAuditAction(event.Action)
	event.TargetType = strings.TrimSpace(event.TargetType)
	event.TargetID = strings.TrimSpace(event.TargetID)
	event.Summary = TruncateAuditValue(event.Summary, 220)
	event.Details = SanitizeAuditDetails(event.Details)
	if event.At.IsZero() {
		event.At = time.Now()
	}
	event.At = event.At.UTC().Truncate(time.Second)
	return event
}

func NormalizeAuditAction(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	switch raw {
	case AuditActionLogin, AuditActionInviteCreate, AuditActionInviteUpdate, AuditActionInviteDelete,
		AuditActionBuildingUpdate, AuditActionHeroUpdate, AuditActionUnitSave, AuditActionUnitDelete,
		AuditActionUnitPayment,
		AuditActionDocumentUpload, AuditActionDocumentDownload, AuditActionDocumentReplace,
		AuditActionAttachmentView, AuditActionAttachmentDelete, AuditActionIntegrationImport, AuditActionIntegrationExport,
		AuditActionHandoverCreate, AuditActionHandoverConfirm, AuditActionHandoverFile,
		AuditActionVoteCreate, AuditActionVoteOpen, AuditActionVoteClose, AuditActionVoteCast, AuditActionVoteReminder,
		AuditActionParkingSettings, AuditActionParkingMonth, AuditActionParkingReminder,
		AuditActionChargingSettings, AuditActionChargingManual, AuditActionChargingSession, AuditActionIssueWorkflow,
		AuditActionIssueEstimate, AuditActionIssueServiceAdd, AuditActionIssueServiceDrop,
		AuditActionIssueComment, AuditActionIssueCommentDelete,
		AuditActionEventCreate, AuditActionEventUpdate, AuditActionEventDelete,
		AuditActionContactSave, AuditActionContactDelete:
		return raw
	default:
		return ""
	}
}

func SanitizeAuditDetails(details map[string]string) map[string]string {
	if len(details) == 0 {
		return nil
	}
	out := map[string]string{}
	for key, value := range details {
		key = strings.ToLower(strings.TrimSpace(key))
		key = strings.ReplaceAll(key, " ", "_")
		if key == "" || AuditDetailKeySensitive(key) {
			continue
		}
		value = TruncateAuditValue(value, 180)
		if value == "" {
			continue
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func TruncateAuditValue(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit-1]) + "…"
}

func CopyAuditEvent(event AuditEvent) AuditEvent {
	if event.Details != nil {
		details := make(map[string]string, len(event.Details))
		for key, value := range event.Details {
			details[key] = value
		}
		event.Details = details
	}
	return event
}

func AuditEventMatches(event AuditEvent, query string) bool {
	haystack := strings.ToLower(strings.Join([]string{
		event.ActorEmail,
		event.ActorRole,
		event.Action,
		event.TargetType,
		event.TargetID,
		event.Summary,
		strings.Join(AuditDetailValues(event.Details), " "),
	}, " "))
	return strings.Contains(haystack, query)
}

func AuditDetailValues(details map[string]string) []string {
	values := make([]string, 0, len(details))
	for key, value := range details {
		values = append(values, key, value)
	}
	return values
}

func NormalizeRole(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "admin", "administrator", "platform-admin", "platform_admin":
		return RoleAdmin
	case "verwalter", "verwaltung", "hausverwaltung", "manager", "property-manager", "property_manager", "property manager":
		return RoleManager
	case "eigentuemer", "eigentümer", "wohnungseigentuemer", "wohnungseigentümer", "owner", "homeowner", "property-owner", "property_owner":
		return RoleOwner
	case "mieter", "tenant", "renter", "lessee":
		return RoleRenter
	case "beirat", "board", "advisory-board", "advisory_board", "committee":
		return RoleBeirat
	case "bewohner", "resident", "user":
		return RoleResident
	case "dienstleister", "handwerker", "service-provider", "service_provider", "service provider", "contractor", "vendor", "external":
		return RoleServiceProvider
	default:
		return strings.TrimSpace(raw)
	}
}

func NormalizeUnitType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "wohnung", "wohneinheit", "residential":
		return UnitTypeResidential
	case "geschaeft", "geschäft", "geschaeftslokal", "geschäftslokal", "commercial", "business":
		return UnitTypeCommercial
	case "stellplatz", "parkplatz", "parking":
		return UnitTypeParking
	case "keller", "lager", "storage":
		return UnitTypeStorage
	case "sonstiges", "other":
		return UnitTypeOther
	default:
		return ""
	}
}

func NormalizeUnitBillableWeight(unitType string, weight int) int {
	if weight < 0 {
		return 0
	}
	if weight > 0 {
		return weight
	}
	return DefaultUnitBillableWeight(unitType)
}

func NormalizeUnits(raw []Unit, fallbackTenant string) []Unit {
	out := make([]Unit, 0, len(raw))
	seen := map[string]struct{}{}
	for _, item := range raw {
		item.TenantSlug = textutil.Slug(textutil.FirstNonEmpty(item.TenantSlug, fallbackTenant))
		item.ID = NormalizeUnitID(item.ID)
		item.Label = strings.TrimSpace(item.Label)
		if item.ID == "" && item.Label != "" {
			item.ID = NormalizeUnitID(item.Label)
		}
		if item.Label == "" {
			item.Label = item.ID
		}
		if item.TenantSlug == "" || item.ID == "" {
			continue
		}
		if item.MiteigentumsanteilPPM < 0 {
			item.MiteigentumsanteilPPM = 0
		}
		item.UnitType = NormalizeUnitType(item.UnitType)
		if item.UnitType == "" {
			item.UnitType = UnitTypeResidential
		}
		item.BillableWeightPPM = NormalizeUnitBillableWeight(item.UnitType, item.BillableWeightPPM)
		item.OwnerEmails = NormalizeEmailList(item.OwnerEmails)
		item.RenterEmails = NormalizeEmailList(item.RenterEmails)
		key := item.TenantSlug + "/" + item.ID
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	SortUnits(out)
	return out
}

func NormalizeEmailList(raw []string) []string {
	out := []string{}
	seen := map[string]struct{}{}
	for _, item := range raw {
		email := textutil.Email(item)
		if email == "" {
			continue
		}
		if _, ok := seen[email]; ok {
			continue
		}
		seen[email] = struct{}{}
		out = append(out, email)
	}
	sort.Strings(out)
	return out
}

func EmailListContains(list []string, email string) bool {
	email = textutil.Email(email)
	for _, item := range list {
		if textutil.Email(item) == email {
			return true
		}
	}
	return false
}

func CopyUnit(item Unit) Unit {
	item.OwnerEmails = append([]string(nil), item.OwnerEmails...)
	item.RenterEmails = append([]string(nil), item.RenterEmails...)
	return item
}

func SortUnits(units []Unit) {
	sort.Slice(units, func(i, j int) bool {
		return UnitLess(units[i], units[j])
	})
}

func UnitLess(a Unit, b Unit) bool {
	if a.TenantSlug != b.TenantSlug {
		return a.TenantSlug < b.TenantSlug
	}
	if strings.ToLower(a.Label) != strings.ToLower(b.Label) {
		return strings.ToLower(a.Label) < strings.ToLower(b.Label)
	}
	return a.ID < b.ID
}

func AuditDetailKeySensitive(key string) bool {
	key = strings.ToLower(key)
	for _, marker := range []string{"secret", "token", "password", "passwd", "private_key", "client_secret"} {
		if strings.Contains(key, marker) {
			return true
		}
	}
	return false
}

func DefaultUnitBillableWeight(unitType string) int {
	switch NormalizeUnitType(unitType) {
	case UnitTypeResidential, UnitTypeCommercial:
		return UnitBillableFullPPM
	default:
		return 0
	}
}
