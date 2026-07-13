package store

// NOTE ON net/http: this package imports net/http for exactly one thing —
// http.DetectContentType, a pure byte-sniffing function with no transport
// behaviour. That is deliberate and is NOT a layering violation: no handler,
// no server, no request type crosses into the store. Uploads still arrive as
// UploadedFile rather than *multipart.FileHeader.

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/markus-barta/hausv-org/internal/textutil"
)

const (
	DefaultTenantHeroImageURL     = "/assets/jhw22-hero.jpg"
	MaxIssuePhotoBytes            = 5 << 20
	MaxIssueFormBytes             = MaxIssuePhotoBytes + (1 << 20)
	MaxAttachmentBytes            = 10 << 20
	MaxIssueAttachmentCount       = 10
	MaxIssueAttachmentFormBytes   = MaxAttachmentBytes*MaxIssueAttachmentCount + (1 << 20)
	AttachmentPreviewMaxDimension = 1200
	AttachmentThumbMaxDimension   = 320
	MaxTenantHeroBytes            = 5 << 20
	MaxTenantHeroFormBytes        = MaxTenantHeroBytes + (1 << 20)
	MaxDocumentBytes              = 20 << 20
	MaxDocumentFormBytes          = MaxDocumentBytes + (1 << 20)
)

const (
	DocumentCategoryProtocol = "Protokoll"
	DocumentCategoryBilling  = "Abrechnung"
	DocumentCategoryRules    = "Hausordnung"
	DocumentCategoryContract = "Vertrag"
	DocumentCategoryPlan     = "Plan"
	DocumentCategoryOther    = "Sonstiges"

	DocumentVisibilityAllResidents = "all-residents"
	DocumentVisibilityOwnersOnly   = "owners-only"
	DocumentVisibilityManagerOnly  = "verwalter-only"
)

type IssueStore struct {
	mu            sync.Mutex
	path          string
	attachmentDir string
	data          IssueStoreData
}

type IssueStoreData struct {
	Issues []ResidentIssue `json:"issues"`
}

type ResidentIssue struct {
	ID                  string              `json:"id"`
	TenantSlug          string              `json:"tenant"`
	AuthorEmail         string              `json:"author_email"`
	AuthorName          string              `json:"author_name"`
	Category            string              `json:"category"`
	Title               string              `json:"title"`
	Body                string              `json:"body"`
	LocationType        string              `json:"location_type"`
	LocationDetail      string              `json:"location_detail"`
	PhotoPaths          []string            `json:"photo_paths"`
	Status              string              `json:"status"`
	Priority            string              `json:"priority"`
	AssigneeEmail       string              `json:"assignee_email,omitempty"`
	StatusChangedAt     time.Time           `json:"status_changed_at,omitempty"`
	StatusChangedBy     string              `json:"status_changed_by,omitempty"`
	StatusHistory       []IssueStatusChange `json:"status_history,omitempty"`
	ServiceProposal     string              `json:"service_proposal,omitempty"`
	ServiceProposedBy   string              `json:"service_proposed_by,omitempty"`
	ServiceProposedAt   time.Time           `json:"service_proposed_at,omitempty"`
	EstimateAmountCents int64               `json:"estimate_amount_cents,omitempty"`
	EstimateNote        string              `json:"estimate_note,omitempty"`
	EstimateUpdatedBy   string              `json:"estimate_updated_by,omitempty"`
	EstimateUpdatedAt   time.Time           `json:"estimate_updated_at,omitempty"`
	Comments            []IssueComment      `json:"comments,omitempty"`
	CreatedAt           time.Time           `json:"created_at"`
	UpdatedAt           time.Time           `json:"updated_at"`
}

type IssueComment struct {
	ID          string    `json:"id"`
	AuthorEmail string    `json:"author_email"`
	AuthorName  string    `json:"author_name"`
	Body        string    `json:"body"`
	CreatedAt   time.Time `json:"created_at"`
}

type IssueStatusChange struct {
	From       string    `json:"from"`
	To         string    `json:"to"`
	ActorEmail string    `json:"actor_email"`
	ActorName  string    `json:"actor_name"`
	ChangedAt  time.Time `json:"changed_at"`
}

type IssueWorkflowUpdate struct {
	Status                string
	Priority              string
	AssigneeEmail         string
	ServiceProposal       string
	UpdateServiceProposal bool
	EstimateAmountCents   int64
	EstimateNote          string
	UpdateEstimate        bool
	ActorEmail            string
	ActorName             string
	ChangedAt             time.Time
}

type AttachmentStore struct {
	mu      sync.Mutex
	path    string
	fileDir string
	data    AttachmentStoreData
}

type AttachmentStoreData struct {
	Attachments []AttachmentRecord `json:"attachments"`
}

type AttachmentRecord struct {
	ID                 string     `json:"id"`
	TenantSlug         string     `json:"tenant"`
	EntityType         string     `json:"entity_type"`
	EntityID           string     `json:"entity_id"`
	UploadedBy         string     `json:"uploaded_by"`
	Filename           string     `json:"filename"`
	StoredFilename     string     `json:"stored_filename"`
	ContentType        string     `json:"content_type"`
	Size               int64      `json:"size"`
	PreviewFilename    string     `json:"preview_filename,omitempty"`
	PreviewContentType string     `json:"preview_content_type,omitempty"`
	PreviewSize        int64      `json:"preview_size,omitempty"`
	ThumbFilename      string     `json:"thumb_filename,omitempty"`
	ThumbContentType   string     `json:"thumb_content_type,omitempty"`
	ThumbSize          int64      `json:"thumb_size,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	DeletedAt          *time.Time `json:"deleted_at,omitempty"`
}

type DocumentStore struct {
	mu      sync.Mutex
	path    string
	fileDir string
	data    DocumentStoreData
}

type DocumentStoreData struct {
	Documents []DocumentRecord `json:"documents"`
}

type DocumentRecord struct {
	ID             string    `json:"id"`
	SeriesID       string    `json:"series_id,omitempty"`
	Version        int       `json:"version"`
	Current        bool      `json:"current"`
	SupersedesID   string    `json:"supersedes_id,omitempty"`
	ReplacedByID   string    `json:"replaced_by_id,omitempty"`
	TenantSlug     string    `json:"tenant"`
	Title          string    `json:"title"`
	Category       string    `json:"category"`
	Visibility     string    `json:"visibility"`
	UnitID         string    `json:"unit_id,omitempty"`
	Filename       string    `json:"filename"`
	StoredFilename string    `json:"stored_filename"`
	Size           int64     `json:"size"`
	ContentType    string    `json:"content_type"`
	UploadedBy     string    `json:"uploaded_by"`
	UploadedAt     time.Time `json:"uploaded_at"`
}

type ParkingStore struct {
	mu   sync.Mutex
	path string
	data ParkingStoreData
}

type ParkingStoreData struct {
	Tenants map[string]ParkingTenantData `json:"tenants"`
}

type ParkingTenantData struct {
	Settings      ParkingSettings              `json:"settings"`
	Months        map[string]ParkingMonthState `json:"months"`
	EnergySamples []ParkingNumericSample       `json:"energy_samples"`
	PriceSamples  []ParkingNumericSample       `json:"price_samples"`
	Samples       []ParkingStoredSample        `json:"samples,omitempty"`
}

type ParkingSettings struct {
	GridFeeEURPerKWh float64         `json:"grid_fee_eur_per_kwh"`
	BaseFeeEUR       float64         `json:"base_fee_eur,omitempty"`
	Tariffs          []ParkingTariff `json:"tariffs,omitempty"`
}

type ParkingTariff struct {
	EffectiveFrom    string  `json:"effective_from"`
	GridFeeEURPerKWh float64 `json:"grid_fee_eur_per_kwh"`
	BaseFeeEUR       float64 `json:"base_fee_eur,omitempty"`
}

type ParkingMonthState struct {
	Paid             bool                 `json:"paid"`
	PaidAt           time.Time            `json:"paid_at,omitempty"`
	PaidBy           string               `json:"paid_by,omitempty"`
	PaymentMethod    string               `json:"payment_method,omitempty"`
	PaymentReference string               `json:"payment_reference,omitempty"`
	ReminderSentAt   map[string]time.Time `json:"reminder_sent_at,omitempty"`
}

type ParkingStoredSample struct {
	At             time.Time `json:"at"`
	EnergyKWh      float64   `json:"energy_kwh"`
	PriceEURPerKWh float64   `json:"price_eur_per_kwh"`
}

type ParkingNumericSample struct {
	At    time.Time `json:"at"`
	Value float64   `json:"value"`
}

func NormalizeIssueCategory(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "reparatur", "repair", "mangel", "mängel", "schaden":
		return "Reparatur"
	case "frage", "question":
		return "Frage"
	case "vorschlag", "idee", "suggestion":
		return "Vorschlag"
	case "sonstiges", "sonstige", "other":
		return "Sonstiges"
	default:
		return ""
	}
}

func NormalizeIssueStatus(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "neu", "offen", "new", "open":
		return IssueStatusNew
	case "angenommen", "accepted":
		return IssueStatusAccepted
	case "termin vereinbart", "termin", "scheduled":
		return IssueStatusScheduled
	case "in bearbeitung", "bearbeitung", "in-arbeit", "progress", "in_progress":
		return IssueStatusProgress
	case "erledigt", "geschlossen", "done", "closed":
		return IssueStatusDone
	case "abgelehnt", "rejected":
		return IssueStatusRejected
	case "duplikat", "duplicate":
		return IssueStatusDuplicate
	default:
		return ""
	}
}

func NormalizeIssuePriority(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "niedrig", "low":
		return IssuePriorityLow
	case "", "normal", "mittel", "medium":
		return IssuePriorityNorm
	case "hoch", "high":
		return IssuePriorityHigh
	case "dringend", "urgent":
		return IssuePriorityUrgent
	default:
		return ""
	}
}

func NormalizeIssueLocation(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case IssueLocationUnit, "eigene einheit", "wohnung", "unit":
		return IssueLocationUnit
	case IssueLocationCommon, "gemeinschaft", "allgemeinbereich":
		return IssueLocationCommon
	default:
		return ""
	}
}

func NewIssueStore(path string, attachmentDir string) (*IssueStore, error) {
	if attachmentDir == "" && path != "" {
		attachmentDir = filepath.Join(filepath.Dir(path), "issue-attachments")
	}
	store := &IssueStore{path: path, attachmentDir: attachmentDir, data: IssueStoreData{Issues: []ResidentIssue{}}}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read issue data")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("invalid issue data")
	}
	if store.data.Issues == nil {
		store.data.Issues = []ResidentIssue{}
	}
	return store, nil
}

func (s *IssueStore) Create(item ResidentIssue) (ResidentIssue, error) {
	if s == nil {
		return item, nil
	}
	now := time.Now().UTC()
	if item.ID == "" {
		id, err := randomToken(12)
		if err != nil {
			return ResidentIssue{}, err
		}
		item.ID = id
	}
	item.TenantSlug = textutil.Slug(item.TenantSlug)
	item.AuthorEmail = textutil.Email(item.AuthorEmail)
	item.Category = NormalizeIssueCategory(item.Category)
	item.LocationType = NormalizeIssueLocation(item.LocationType)
	item.Title = strings.TrimSpace(item.Title)
	item.Body = strings.TrimSpace(item.Body)
	item.LocationDetail = strings.TrimSpace(item.LocationDetail)
	if item.TenantSlug == "" || item.AuthorEmail == "" || item.Category == "" || item.LocationType == "" || item.Title == "" || item.Body == "" {
		return ResidentIssue{}, fmt.Errorf("invalid issue")
	}
	item.Status = NormalizeIssueStatus(item.Status)
	if item.Status == "" {
		item.Status = IssueStatusOpen
	}
	item.Priority = NormalizeIssuePriority(item.Priority)
	if item.Priority == "" {
		item.Priority = IssuePriorityNorm
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	} else {
		item.CreatedAt = item.CreatedAt.UTC()
	}
	if item.UpdatedAt.IsZero() {
		item.UpdatedAt = item.CreatedAt
	} else {
		item.UpdatedAt = item.UpdatedAt.UTC()
	}
	if item.StatusChangedAt.IsZero() {
		item.StatusChangedAt = item.CreatedAt
	} else {
		item.StatusChangedAt = item.StatusChangedAt.UTC()
	}
	if item.StatusChangedBy == "" {
		item.StatusChangedBy = item.AuthorEmail
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Issues = append(s.data.Issues, item)
	if err := s.saveLocked(); err != nil {
		return ResidentIssue{}, err
	}
	return item, nil
}

func (s *IssueStore) ListTenant(tenantSlug string) []ResidentIssue {
	if s == nil {
		return nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []ResidentIssue{}
	for _, item := range s.data.Issues {
		if textutil.Slug(item.TenantSlug) == tenantSlug {
			out = append(out, CopyIssue(item))
		}
	}
	SortIssues(out)
	return out
}

func (s *IssueStore) ListAuthor(tenantSlug string, email string) []ResidentIssue {
	if s == nil {
		return nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	email = textutil.Email(email)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []ResidentIssue{}
	for _, item := range s.data.Issues {
		if textutil.Slug(item.TenantSlug) == tenantSlug && textutil.Email(item.AuthorEmail) == email {
			out = append(out, CopyIssue(item))
		}
	}
	SortIssues(out)
	return out
}

func (s *IssueStore) Get(tenantSlug string, id string) (ResidentIssue, bool) {
	if s == nil {
		return ResidentIssue{}, false
	}
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	if tenantSlug == "" || id == "" {
		return ResidentIssue{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.data.Issues {
		if textutil.Slug(item.TenantSlug) == tenantSlug && item.ID == id {
			return CopyIssue(item), true
		}
	}
	return ResidentIssue{}, false
}

func (s *IssueStore) UpdateWorkflow(tenantSlug string, id string, update IssueWorkflowUpdate) (ResidentIssue, bool, error) {
	if s == nil {
		return ResidentIssue{}, false, nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	status := NormalizeIssueStatus(update.Status)
	priority := NormalizeIssuePriority(update.Priority)
	if tenantSlug == "" || id == "" || status == "" || priority == "" {
		return ResidentIssue{}, false, fmt.Errorf("invalid issue workflow update")
	}
	if update.ChangedAt.IsZero() {
		update.ChangedAt = time.Now()
	}
	changedAt := update.ChangedAt.UTC()
	actorEmail := textutil.Email(update.ActorEmail)
	actorName := strings.TrimSpace(update.ActorName)
	assignee := textutil.Email(update.AssigneeEmail)

	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.data.Issues {
		if textutil.Slug(existing.TenantSlug) != tenantSlug || existing.ID != id {
			continue
		}
		updated := existing
		oldStatus := NormalizeIssueStatus(updated.Status)
		if oldStatus == "" {
			oldStatus = IssueStatusOpen
		}
		updated.Status = status
		updated.Priority = priority
		updated.AssigneeEmail = assignee
		if update.UpdateServiceProposal {
			updated.ServiceProposal = strings.TrimSpace(update.ServiceProposal)
			updated.ServiceProposedBy = actorEmail
			updated.ServiceProposedAt = changedAt
			if updated.ServiceProposal == "" {
				updated.ServiceProposedBy = ""
				updated.ServiceProposedAt = time.Time{}
			}
		}
		if update.UpdateEstimate {
			updated.EstimateAmountCents = update.EstimateAmountCents
			updated.EstimateNote = strings.TrimSpace(update.EstimateNote)
			updated.EstimateUpdatedBy = actorEmail
			updated.EstimateUpdatedAt = changedAt
			if updated.EstimateAmountCents <= 0 && updated.EstimateNote == "" {
				updated.EstimateAmountCents = 0
				updated.EstimateUpdatedBy = ""
				updated.EstimateUpdatedAt = time.Time{}
			}
		}
		updated.UpdatedAt = changedAt
		if oldStatus != status {
			updated.StatusChangedAt = changedAt
			updated.StatusChangedBy = actorEmail
			updated.StatusHistory = append(updated.StatusHistory, IssueStatusChange{
				From:       oldStatus,
				To:         status,
				ActorEmail: actorEmail,
				ActorName:  actorName,
				ChangedAt:  changedAt,
			})
		}
		s.data.Issues[i] = updated
		if err := s.saveLocked(); err != nil {
			return ResidentIssue{}, false, err
		}
		return CopyIssue(updated), true, nil
	}
	return ResidentIssue{}, false, nil
}

func (s *IssueStore) AddComment(tenantSlug string, id string, comment IssueComment) (ResidentIssue, bool, error) {
	if s == nil {
		return ResidentIssue{}, false, nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	comment.Body = strings.TrimSpace(comment.Body)
	comment.AuthorEmail = textutil.Email(comment.AuthorEmail)
	comment.AuthorName = strings.TrimSpace(comment.AuthorName)
	// Body may be empty for a photo-only comment (HAUSV-128); the handler enforces
	// that a comment carries either text or at least one attachment.
	if tenantSlug == "" || id == "" || len([]rune(comment.Body)) > 3000 || comment.AuthorEmail == "" {
		return ResidentIssue{}, false, fmt.Errorf("invalid issue comment")
	}
	if comment.ID == "" {
		commentID, err := randomToken(10)
		if err != nil {
			return ResidentIssue{}, false, err
		}
		comment.ID = commentID
	}
	if comment.CreatedAt.IsZero() {
		comment.CreatedAt = time.Now().UTC()
	} else {
		comment.CreatedAt = comment.CreatedAt.UTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.data.Issues {
		if textutil.Slug(existing.TenantSlug) != tenantSlug || existing.ID != id {
			continue
		}
		updated := existing
		updated.Comments = append(updated.Comments, comment)
		updated.UpdatedAt = comment.CreatedAt
		s.data.Issues[i] = updated
		if err := s.saveLocked(); err != nil {
			return ResidentIssue{}, false, err
		}
		return CopyIssue(updated), true, nil
	}
	return ResidentIssue{}, false, nil
}

func (s *IssueStore) DeleteComment(tenantSlug string, id string, commentID string, at time.Time) (ResidentIssue, bool, error) {
	if s == nil {
		return ResidentIssue{}, false, nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	commentID = strings.TrimSpace(commentID)
	if tenantSlug == "" || id == "" || commentID == "" {
		return ResidentIssue{}, false, fmt.Errorf("invalid issue comment")
	}
	if at.IsZero() {
		at = time.Now().UTC()
	} else {
		at = at.UTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.data.Issues {
		if textutil.Slug(existing.TenantSlug) != tenantSlug || existing.ID != id {
			continue
		}
		updated := existing
		comments := make([]IssueComment, 0, len(updated.Comments))
		deleted := false
		for _, comment := range updated.Comments {
			if comment.ID == commentID {
				deleted = true
				continue
			}
			comments = append(comments, comment)
		}
		if !deleted {
			return ResidentIssue{}, false, nil
		}
		updated.Comments = comments
		updated.UpdatedAt = at
		s.data.Issues[i] = updated
		if err := s.saveLocked(); err != nil {
			return ResidentIssue{}, false, err
		}
		return CopyIssue(updated), true, nil
	}
	return ResidentIssue{}, false, nil
}

// uploadedFile is what the store layer needs from an upload: a name, a size,
// and a way to read the bytes. Deliberately NOT *multipart.FileHeader — that
// drags mime/multipart (the HTTP transport) into the store layer and blocks
// extracting it as a package.
//
// Open returns a ReadSeekCloser because the stores sniff the first 512 bytes
// for content-type detection and then rewind; a plain io.ReadCloser would
// silently break that.
type UploadedFile struct {
	Filename string
	Size     int64
	// DeclaredType is the client-supplied Content-Type from the multipart part.
	// It is only a hint — the stores still sniff the bytes.
	DeclaredType string
	Open         func() (io.ReadSeekCloser, error)
}

func (s *IssueStore) SavePhoto(tenantSlug string, issueID string, upload UploadedFile) (string, error) {
	if s == nil || upload.Open == nil || upload.Filename == "" || upload.Size == 0 {
		return "", nil
	}
	if s.attachmentDir == "" {
		return "", fmt.Errorf("issue attachment directory unavailable")
	}
	if upload.Size > MaxIssuePhotoBytes {
		return "", fmt.Errorf("issue photo too large")
	}
	file, err := upload.Open()
	if err != nil {
		return "", fmt.Errorf("could not open issue photo")
	}
	defer file.Close()

	sniff := make([]byte, 512)
	n, readErr := file.Read(sniff)
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return "", fmt.Errorf("could not read issue photo")
	}
	contentType := http.DetectContentType(sniff[:n])
	ext, ok := IssuePhotoExtension(contentType)
	if !ok {
		return "", fmt.Errorf("unsupported issue photo type")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("could not rewind issue photo")
	}

	tenantSlug = textutil.Slug(tenantSlug)
	if tenantSlug == "" {
		tenantSlug = "tenant"
	}
	issueID = textutil.Slug(issueID)
	if issueID == "" {
		return "", fmt.Errorf("issue id required")
	}
	dir := filepath.Join(s.attachmentDir, tenantSlug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("could not create issue attachment directory")
	}
	filename := issueID + "-photo" + ext
	dest := filepath.Join(dir, filename)
	out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", fmt.Errorf("could not create issue attachment")
	}
	written, copyErr := io.Copy(out, io.LimitReader(file, MaxIssuePhotoBytes+1))
	closeErr := out.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(dest)
		return "", fmt.Errorf("could not write issue attachment")
	}
	if written > MaxIssuePhotoBytes {
		_ = os.Remove(dest)
		return "", fmt.Errorf("issue photo too large")
	}
	return filepath.ToSlash(filepath.Join(filepath.Base(s.attachmentDir), tenantSlug, filename)), nil
}

// stripContentTypeParams drops the parameters from a media type, e.g.
// "text/xml; charset=utf-8" -> "text/xml". http.DetectContentType returns the
// charset form for XML, which never matched the exact-string switches below, so
// legitimate XML uploads were silently rejected (HAUSV-144).
func stripContentTypeParams(contentType string) string {
	return strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
}

func IssuePhotoExtension(contentType string) (string, bool) {
	switch stripContentTypeParams(contentType) {
	case "image/jpeg":
		return ".jpg", true
	case "image/png":
		return ".png", true
	case "image/webp":
		return ".webp", true
	case "application/xml", "text/xml":
		return ".xml", true
	default:
		return "", false
	}
}

func (s *IssueStore) saveLocked() error {
	return SaveJSONAtomic(s.path, s.data, "issue")
}

func CopyIssue(item ResidentIssue) ResidentIssue {
	item.PhotoPaths = append([]string(nil), item.PhotoPaths...)
	item.StatusHistory = append([]IssueStatusChange(nil), item.StatusHistory...)
	item.Comments = append([]IssueComment(nil), item.Comments...)
	return item
}

func SortIssues(items []ResidentIssue) {
	sort.SliceStable(items, func(i, j int) bool {
		if !items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].UpdatedAt.After(items[j].UpdatedAt)
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
}

func NewAttachmentStore(path string, fileDir string) (*AttachmentStore, error) {
	if fileDir == "" && path != "" {
		fileDir = filepath.Join(filepath.Dir(path), "attachments")
	}
	store := &AttachmentStore{path: path, fileDir: fileDir, data: AttachmentStoreData{Attachments: []AttachmentRecord{}}}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read attachment data")
	}
	if len(raw) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("could not decode attachment data")
	}
	return store, nil
}

func (s *AttachmentStore) CreateUploaded(tenantSlug string, entityType string, entityID string, uploadedBy string, uploads []UploadedFile, now time.Time) ([]AttachmentRecord, error) {
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
	s.mu.Lock()
	defer s.mu.Unlock()

	created := []AttachmentRecord{}
	for _, upload := range uploads {
		if upload.Open == nil || strings.TrimSpace(upload.Filename) == "" || upload.Size == 0 {
			continue
		}
		if upload.Size > MaxAttachmentBytes {
			s.rollbackCreatedAttachmentsLocked(created)
			return nil, fmt.Errorf("attachment too large")
		}
		id, err := randomToken(12)
		if err != nil {
			s.rollbackCreatedAttachmentsLocked(created)
			return nil, err
		}
		fileSave, err := s.saveUploadedAttachmentFile(tenantSlug, id, upload)
		if err != nil {
			s.rollbackCreatedAttachmentsLocked(created)
			return nil, err
		}
		record := AttachmentRecord{
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
		}
		s.data.Attachments = append(s.data.Attachments, record)
		created = append(created, record)
	}
	if len(created) == 0 {
		return nil, nil
	}
	if err := s.saveLocked(); err != nil {
		s.rollbackCreatedAttachmentsLocked(created)
		return nil, err
	}
	return created, nil
}

type AttachmentFileSave struct {
	StoredFilename     string
	ContentType        string
	Size               int64
	PreviewFilename    string
	PreviewContentType string
	PreviewSize        int64
	ThumbFilename      string
	ThumbContentType   string
	ThumbSize          int64
}

func (s *AttachmentStore) saveUploadedAttachmentFile(tenantSlug string, id string, upload UploadedFile) (AttachmentFileSave, error) {
	if s.fileDir == "" {
		return AttachmentFileSave{}, fmt.Errorf("attachment file directory unavailable")
	}
	file, err := upload.Open()
	if err != nil {
		return AttachmentFileSave{}, fmt.Errorf("could not open attachment")
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, MaxAttachmentBytes+1))
	if err != nil {
		return AttachmentFileSave{}, fmt.Errorf("could not read attachment")
	}
	if int64(len(data)) > MaxAttachmentBytes {
		return AttachmentFileSave{}, fmt.Errorf("attachment too large")
	}
	if len(data) == 0 {
		return AttachmentFileSave{}, fmt.Errorf("empty attachment")
	}
	contentType := DetectAttachmentContentType(data, upload)
	if RejectActiveAttachmentContent(contentType) {
		return AttachmentFileSave{}, fmt.Errorf("unsafe attachment content")
	}
	ext, ok := AttachmentExtension(contentType)
	if !ok {
		return AttachmentFileSave{}, fmt.Errorf("unsupported attachment type")
	}
	dir := filepath.Join(s.fileDir, tenantSlug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return AttachmentFileSave{}, fmt.Errorf("could not create attachment directory")
	}
	storedFilename := id + ext
	if err := WritePrivateFile(filepath.Join(dir, storedFilename), data); err != nil {
		return AttachmentFileSave{}, fmt.Errorf("could not write attachment")
	}
	save := AttachmentFileSave{
		StoredFilename: storedFilename,
		ContentType:    contentType,
		Size:           int64(len(data)),
	}
	if IsImageContentType(contentType) {
		preview, previewSize, err := WriteImageAttachmentVariant(dir, id, "-preview", data, AttachmentPreviewMaxDimension)
		if err == nil && preview != "" {
			save.PreviewFilename = preview
			save.PreviewContentType = "image/jpeg"
			save.PreviewSize = previewSize
		}
		thumb, thumbSize, err := WriteImageAttachmentVariant(dir, id, "-thumb", data, AttachmentThumbMaxDimension)
		if err == nil && thumb != "" {
			save.ThumbFilename = thumb
			save.ThumbContentType = "image/jpeg"
			save.ThumbSize = thumbSize
		}
	}
	return save, nil
}

func (s *AttachmentStore) ListEntity(tenantSlug string, entityType string, entityID string) []AttachmentRecord {
	if s == nil {
		return nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	entityType = NormalizeAttachmentEntity(entityType)
	entityID = strings.TrimSpace(entityID)
	s.mu.Lock()
	defer s.mu.Unlock()
	items := []AttachmentRecord{}
	for _, item := range s.data.Attachments {
		if item.DeletedAt != nil {
			continue
		}
		if textutil.Slug(item.TenantSlug) == tenantSlug && NormalizeAttachmentEntity(item.EntityType) == entityType && strings.TrimSpace(item.EntityID) == entityID {
			items = append(items, item)
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	return items
}

func (s *AttachmentStore) Get(tenantSlug string, id string) (AttachmentRecord, bool) {
	if s == nil {
		return AttachmentRecord{}, false
	}
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.data.Attachments {
		if item.DeletedAt == nil && textutil.Slug(item.TenantSlug) == tenantSlug && item.ID == id {
			return item, true
		}
	}
	return AttachmentRecord{}, false
}

func (s *AttachmentStore) Delete(tenantSlug string, id string, deletedAt time.Time) (AttachmentRecord, bool, error) {
	if s == nil {
		return AttachmentRecord{}, false, nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, item := range s.data.Attachments {
		if item.DeletedAt != nil || textutil.Slug(item.TenantSlug) != tenantSlug || item.ID != id {
			continue
		}
		deleted := deletedAt.UTC()
		s.data.Attachments[i].DeletedAt = &deleted
		if err := s.saveLocked(); err != nil {
			s.data.Attachments[i].DeletedAt = nil
			return AttachmentRecord{}, false, err
		}
		s.removeAttachmentRecordLocked(item)
		return item, true, nil
	}
	return AttachmentRecord{}, false, nil
}

func (s *AttachmentStore) FilePath(item AttachmentRecord, variant string) (string, string, int64, bool) {
	if s == nil || s.fileDir == "" || item.StoredFilename == "" {
		return "", "", 0, false
	}
	filename := item.StoredFilename
	contentType := item.ContentType
	size := item.Size
	switch strings.ToLower(strings.TrimSpace(variant)) {
	case "preview":
		if item.PreviewFilename != "" {
			filename = item.PreviewFilename
			contentType = item.PreviewContentType
			size = item.PreviewSize
		}
	case "thumb", "thumbnail":
		if item.ThumbFilename != "" {
			filename = item.ThumbFilename
			contentType = item.ThumbContentType
			size = item.ThumbSize
		} else if item.PreviewFilename != "" {
			filename = item.PreviewFilename
			contentType = item.PreviewContentType
			size = item.PreviewSize
		}
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	path := filepath.Join(s.fileDir, textutil.Slug(item.TenantSlug), filename)
	return path, contentType, size, true
}

func (s *AttachmentStore) saveLocked() error {
	return SaveJSONAtomic(s.path, s.data, "attachment")
}

func (s *AttachmentStore) rollbackCreatedAttachmentsLocked(items []AttachmentRecord) {
	if len(items) == 0 {
		return
	}
	ids := map[string]struct{}{}
	for _, item := range items {
		ids[item.ID] = struct{}{}
		s.removeAttachmentRecordLocked(item)
	}
	kept := s.data.Attachments[:0]
	for _, item := range s.data.Attachments {
		if _, ok := ids[item.ID]; ok {
			continue
		}
		kept = append(kept, item)
	}
	s.data.Attachments = kept
}

func (s *AttachmentStore) removeAttachmentRecordLocked(item AttachmentRecord) {
	for _, filename := range []string{item.StoredFilename, item.PreviewFilename, item.ThumbFilename} {
		if filename == "" {
			continue
		}
		_ = os.Remove(filepath.Join(s.fileDir, textutil.Slug(item.TenantSlug), filename))
	}
}

func DetectAttachmentContentType(data []byte, upload UploadedFile) string {
	limit := len(data)
	if limit > 512 {
		limit = 512
	}
	detected := http.DetectContentType(data[:limit])
	declared := ""
	if upload.DeclaredType != "" {
		declared = strings.ToLower(strings.TrimSpace(upload.DeclaredType))
	}
	if detected == "application/octet-stream" && declared != "" {
		if _, ok := AttachmentExtension(declared); ok {
			return declared
		}
	}
	return detected
}

func AttachmentExtension(contentType string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0])) {
	case "image/jpeg", "image/jpg":
		return ".jpg", true
	case "image/png":
		return ".png", true
	case "image/webp":
		return ".webp", true
	case "image/gif":
		return ".gif", true
	case "application/pdf":
		return ".pdf", true
	default:
		return "", false
	}
}

func NormalizeAttachmentEntity(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "issue", "anliegen":
		return "issue"
	case "issue-estimate", "estimate", "kostenvoranschlag":
		return "issue-estimate"
	case "issue-comment", "comment", "anliegen-kommentar":
		return "issue-comment"
	case "announcement", "aushang":
		return "announcement"
	case "event", "termin":
		return "event"
	case "ballot", "vote", "abstimmung":
		return "ballot"
	case "handover", "uebergabe", "übergabe":
		return "handover"
	case "parking", "parkplatz":
		return "parking"
	case "document", "dokument":
		return "document"
	case "building", "tenant", "gebaeude", "gebäude":
		return "building"
	default:
		return ""
	}
}

func IsImageContentType(contentType string) bool {
	contentType = strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	return strings.HasPrefix(contentType, "image/") && contentType != "image/svg+xml"
}

func RejectActiveAttachmentContent(contentType string) bool {
	switch strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0])) {
	case "text/html", "text/javascript", "application/javascript", "application/x-javascript", "image/svg+xml", "application/xhtml+xml", "application/xml", "text/xml":
		return true
	default:
		return false
	}
}

func WriteImageAttachmentVariant(dir string, id string, suffix string, data []byte, maxDimension int) (string, int64, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", 0, err
	}
	resized := ResizeImageNearest(img, maxDimension)
	var out bytes.Buffer
	if err := jpeg.Encode(&out, resized, &jpeg.Options{Quality: 82}); err != nil {
		return "", 0, err
	}
	filename := id + suffix + ".jpg"
	if err := WritePrivateFile(filepath.Join(dir, filename), out.Bytes()); err != nil {
		return "", 0, err
	}
	return filename, int64(out.Len()), nil
}

func ResizeImageNearest(src image.Image, maxDimension int) image.Image {
	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		return src
	}
	if maxDimension <= 0 {
		maxDimension = AttachmentPreviewMaxDimension
	}
	scale := math.Min(float64(maxDimension)/float64(width), float64(maxDimension)/float64(height))
	if scale > 1 {
		scale = 1
	}
	dstWidth := int(math.Round(float64(width) * scale))
	dstHeight := int(math.Round(float64(height) * scale))
	if dstWidth < 1 {
		dstWidth = 1
	}
	if dstHeight < 1 {
		dstHeight = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, dstWidth, dstHeight))
	for y := 0; y < dstHeight; y++ {
		srcY := bounds.Min.Y + int(float64(y)*float64(height)/float64(dstHeight))
		if srcY >= bounds.Max.Y {
			srcY = bounds.Max.Y - 1
		}
		for x := 0; x < dstWidth; x++ {
			srcX := bounds.Min.X + int(float64(x)*float64(width)/float64(dstWidth))
			if srcX >= bounds.Max.X {
				srcX = bounds.Max.X - 1
			}
			dst.Set(x, y, src.At(srcX, srcY))
		}
	}
	return dst
}

func WritePrivateFile(path string, data []byte) error {
	out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, writeErr := out.Write(data)
	closeErr := out.Close()
	if writeErr != nil || closeErr != nil {
		_ = os.Remove(path)
		if writeErr != nil {
			return writeErr
		}
		return closeErr
	}
	return nil
}

func NewDocumentStore(path string, fileDir string) (*DocumentStore, error) {
	if fileDir == "" && path != "" {
		fileDir = filepath.Join(filepath.Dir(path), "documents")
	}
	store := &DocumentStore{path: path, fileDir: fileDir, data: DocumentStoreData{Documents: []DocumentRecord{}}}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read document data")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("invalid document data")
	}
	store.data.Documents = NormalizeDocuments(store.data.Documents)
	return store, nil
}

func (s *DocumentStore) Create(item DocumentRecord, upload UploadedFile, now time.Time) (DocumentRecord, error) {
	if s == nil {
		return DocumentRecord{}, fmt.Errorf("document store unavailable")
	}
	if now.IsZero() {
		now = time.Now()
	}
	item.UploadedAt = now.UTC()
	fileSave, err := s.saveUploadedDocumentFile(item.TenantSlug, upload)
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

	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Documents = append(s.data.Documents, item)
	SortDocuments(s.data.Documents)
	if err := s.saveLocked(); err != nil {
		_ = os.Remove(fileSave.Path)
		return DocumentRecord{}, err
	}
	return CopyDocument(item), nil
}

type DocumentFileSave struct {
	ID             string
	Filename       string
	StoredFilename string
	ContentType    string
	Size           int64
	Path           string
}

func (s *DocumentStore) saveUploadedDocumentFile(tenantSlug string, upload UploadedFile) (DocumentFileSave, error) {
	if upload.Open == nil || upload.Filename == "" || upload.Size <= 0 {
		return DocumentFileSave{}, fmt.Errorf("document file required")
	}
	if s.fileDir == "" {
		return DocumentFileSave{}, fmt.Errorf("document file directory unavailable")
	}
	if upload.Size > MaxDocumentBytes {
		return DocumentFileSave{}, fmt.Errorf("document file too large")
	}
	file, err := upload.Open()
	if err != nil {
		return DocumentFileSave{}, fmt.Errorf("could not open document")
	}
	defer file.Close()
	sniff := make([]byte, 512)
	n, readErr := file.Read(sniff)
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return DocumentFileSave{}, fmt.Errorf("could not read document")
	}
	if n == 0 {
		return DocumentFileSave{}, fmt.Errorf("document file required")
	}
	contentType := http.DetectContentType(sniff[:n])
	ext, ok := DocumentExtension(contentType)
	if !ok {
		return DocumentFileSave{}, fmt.Errorf("unsupported document type")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return DocumentFileSave{}, fmt.Errorf("could not rewind document")
	}
	id, err := randomToken(12)
	if err != nil {
		return DocumentFileSave{}, err
	}
	storedFilename := id + ext
	storedPath, written, err := s.writeDocumentFile(tenantSlug, storedFilename, file)
	if err != nil {
		return DocumentFileSave{}, err
	}
	return DocumentFileSave{
		ID:             id,
		Filename:       SanitizeDocumentFilename(upload.Filename),
		StoredFilename: storedFilename,
		ContentType:    contentType,
		Size:           written,
		Path:           storedPath,
	}, nil
}

func (s *DocumentStore) Replace(tenantSlug string, id string, uploadedBy string, upload UploadedFile, now time.Time) (DocumentRecord, DocumentRecord, error) {
	if s == nil {
		return DocumentRecord{}, DocumentRecord{}, fmt.Errorf("document store unavailable")
	}
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	uploadedBy = textutil.Email(uploadedBy)
	if tenantSlug == "" || id == "" || uploadedBy == "" {
		return DocumentRecord{}, DocumentRecord{}, fmt.Errorf("invalid document replacement")
	}
	existing, found := s.Get(tenantSlug, id)
	if !found || !existing.Current {
		return DocumentRecord{}, DocumentRecord{}, fmt.Errorf("document not found")
	}
	if now.IsZero() {
		now = time.Now()
	}
	fileSave, err := s.saveUploadedDocumentFile(tenantSlug, upload)
	if err != nil {
		return DocumentRecord{}, DocumentRecord{}, err
	}
	replacement := existing
	replacement.ID = fileSave.ID
	replacement.Version = existing.Version + 1
	replacement.Current = true
	replacement.SupersedesID = existing.ID
	replacement.ReplacedByID = ""
	replacement.Filename = fileSave.Filename
	replacement.StoredFilename = fileSave.StoredFilename
	replacement.Size = fileSave.Size
	replacement.ContentType = fileSave.ContentType
	replacement.UploadedBy = uploadedBy
	replacement.UploadedAt = now.UTC()
	replacement = NormalizeDocumentRecord(replacement)

	s.mu.Lock()
	defer s.mu.Unlock()
	for i, item := range s.data.Documents {
		if textutil.Slug(item.TenantSlug) != tenantSlug || item.ID != existing.ID || !item.Current {
			continue
		}
		item.Current = false
		item.ReplacedByID = replacement.ID
		replaced := NormalizeDocumentRecord(item)
		s.data.Documents[i] = replaced
		s.data.Documents = append(s.data.Documents, replacement)
		SortDocuments(s.data.Documents)
		if err := s.saveLocked(); err != nil {
			_ = os.Remove(fileSave.Path)
			return DocumentRecord{}, DocumentRecord{}, err
		}
		return CopyDocument(replacement), CopyDocument(replaced), nil
	}
	_ = os.Remove(fileSave.Path)
	return DocumentRecord{}, DocumentRecord{}, fmt.Errorf("document not current")
}

func (s *DocumentStore) writeDocumentFile(tenantSlug string, storedFilename string, file io.Reader) (string, int64, error) {
	tenantSlug = textutil.Slug(tenantSlug)
	storedFilename = filepath.Base(storedFilename)
	if tenantSlug == "" || storedFilename == "" || storedFilename == "." || storedFilename == string(filepath.Separator) {
		return "", 0, fmt.Errorf("invalid document storage target")
	}
	dir := filepath.Join(s.fileDir, tenantSlug)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", 0, fmt.Errorf("could not create document directory")
	}
	tmp, err := os.CreateTemp(dir, storedFilename+".*.tmp")
	if err != nil {
		return "", 0, fmt.Errorf("could not create document file")
	}
	tmpName := tmp.Name()
	written, copyErr := io.Copy(tmp, io.LimitReader(file, MaxDocumentBytes+1))
	chmodErr := tmp.Chmod(0o600)
	closeErr := tmp.Close()
	if copyErr != nil || chmodErr != nil || closeErr != nil {
		_ = os.Remove(tmpName)
		return "", 0, fmt.Errorf("could not write document file")
	}
	if written <= 0 || written > MaxDocumentBytes {
		_ = os.Remove(tmpName)
		return "", 0, fmt.Errorf("document file too large")
	}
	dest := filepath.Join(dir, storedFilename)
	if err := os.Rename(tmpName, dest); err != nil {
		_ = os.Remove(tmpName)
		return "", 0, fmt.Errorf("could not store document file")
	}
	return dest, written, nil
}

func (s *DocumentStore) ListTenant(tenantSlug string) []DocumentRecord {
	if s == nil {
		return nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []DocumentRecord{}
	for _, item := range s.data.Documents {
		if textutil.Slug(item.TenantSlug) == tenantSlug {
			out = append(out, CopyDocument(item))
		}
	}
	SortDocuments(out)
	return out
}

func (s *DocumentStore) ListCurrentTenant(tenantSlug string) []DocumentRecord {
	all := s.ListTenant(tenantSlug)
	out := []DocumentRecord{}
	for _, item := range all {
		if item.Current {
			out = append(out, item)
		}
	}
	SortDocuments(out)
	return out
}

func (s *DocumentStore) Versions(tenantSlug string, seriesID string) []DocumentRecord {
	if s == nil {
		return nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	seriesID = strings.TrimSpace(seriesID)
	if tenantSlug == "" || seriesID == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []DocumentRecord{}
	for _, item := range s.data.Documents {
		if textutil.Slug(item.TenantSlug) == tenantSlug && item.SeriesID == seriesID {
			out = append(out, CopyDocument(item))
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

func (s *DocumentStore) Get(tenantSlug string, id string) (DocumentRecord, bool) {
	if s == nil {
		return DocumentRecord{}, false
	}
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	if tenantSlug == "" || id == "" {
		return DocumentRecord{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.data.Documents {
		if textutil.Slug(item.TenantSlug) == tenantSlug && item.ID == id {
			return CopyDocument(item), true
		}
	}
	return DocumentRecord{}, false
}

func (s *DocumentStore) FilePath(item DocumentRecord) (string, bool) {
	if s == nil || s.fileDir == "" {
		return "", false
	}
	tenantSlug := textutil.Slug(item.TenantSlug)
	storedFilename := filepath.Base(item.StoredFilename)
	if tenantSlug == "" || storedFilename == "" || storedFilename == "." || storedFilename == string(filepath.Separator) {
		return "", false
	}
	return filepath.Join(s.fileDir, tenantSlug, storedFilename), true
}

func (s *DocumentStore) CreateGenerated(item DocumentRecord, filename string, contentType string, data []byte, now time.Time) (DocumentRecord, error) {
	if s == nil {
		return DocumentRecord{}, fmt.Errorf("document store unavailable")
	}
	if len(data) == 0 || int64(len(data)) > MaxDocumentBytes {
		return DocumentRecord{}, fmt.Errorf("document file too large")
	}
	contentType = strings.TrimSpace(strings.Split(contentType, ";")[0])
	ext, ok := DocumentExtension(contentType)
	if !ok {
		return DocumentRecord{}, fmt.Errorf("unsupported document type")
	}
	if now.IsZero() {
		now = time.Now()
	}
	id, err := randomToken(12)
	if err != nil {
		return DocumentRecord{}, err
	}
	filename = SanitizeDocumentFilename(filename)
	if filename == "dokument" {
		filename = id + ext
	}
	storedFilename := id + ext
	path, err := s.writeGeneratedDocumentFile(item.TenantSlug, storedFilename, data)
	if err != nil {
		return DocumentRecord{}, err
	}
	item.ID = id
	item.SeriesID = id
	item.Version = 1
	item.Current = true
	item.Filename = filename
	item.StoredFilename = storedFilename
	item.Size = int64(len(data))
	item.ContentType = contentType
	item.UploadedAt = now.UTC()
	item = NormalizeDocumentRecord(item)
	if item.TenantSlug == "" || item.Title == "" || item.Category == "" || item.Visibility == "" || item.UploadedBy == "" {
		_ = os.Remove(path)
		return DocumentRecord{}, fmt.Errorf("invalid document metadata")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Documents = append(s.data.Documents, item)
	SortDocuments(s.data.Documents)
	if err := s.saveLocked(); err != nil {
		_ = os.Remove(path)
		return DocumentRecord{}, err
	}
	return CopyDocument(item), nil
}

func (s *DocumentStore) writeGeneratedDocumentFile(tenantSlug string, storedFilename string, data []byte) (string, error) {
	tenantSlug = textutil.Slug(tenantSlug)
	storedFilename = filepath.Base(storedFilename)
	if s == nil || s.fileDir == "" {
		return "", fmt.Errorf("document file directory unavailable")
	}
	if tenantSlug == "" || storedFilename == "" || storedFilename == "." || storedFilename == string(filepath.Separator) {
		return "", fmt.Errorf("invalid document storage target")
	}
	dir := filepath.Join(s.fileDir, tenantSlug)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("could not create document directory")
	}
	path := filepath.Join(dir, storedFilename)
	if err := WritePrivateFile(path, data); err != nil {
		return "", fmt.Errorf("could not write document file")
	}
	return path, nil
}

func (s *DocumentStore) saveLocked() error {
	return SaveJSONAtomic(s.path, s.data, "document")
}

func NormalizeDocuments(items []DocumentRecord) []DocumentRecord {
	out := make([]DocumentRecord, 0, len(items))
	for _, item := range items {
		item = NormalizeDocumentRecord(item)
		if item.ID == "" || item.TenantSlug == "" || item.Title == "" || item.StoredFilename == "" {
			continue
		}
		out = append(out, item)
	}
	SortDocuments(out)
	return out
}

func NormalizeDocumentRecord(item DocumentRecord) DocumentRecord {
	item.ID = strings.TrimSpace(item.ID)
	item.SeriesID = strings.TrimSpace(item.SeriesID)
	if item.SeriesID == "" && item.ID != "" {
		item.SeriesID = item.ID
	}
	if item.Version <= 0 {
		item.Version = 1
	}
	item.SupersedesID = strings.TrimSpace(item.SupersedesID)
	item.ReplacedByID = strings.TrimSpace(item.ReplacedByID)
	if !item.Current && item.ReplacedByID == "" {
		item.Current = true
	}
	item.TenantSlug = textutil.Slug(item.TenantSlug)
	item.Title = TruncateAuditValue(strings.TrimSpace(item.Title), 160)
	item.Category = NormalizeDocumentCategory(item.Category)
	item.Visibility = NormalizeDocumentVisibility(item.Visibility)
	item.UnitID = NormalizeUnitID(item.UnitID)
	item.Filename = SanitizeDocumentFilename(item.Filename)
	item.StoredFilename = filepath.Base(strings.TrimSpace(item.StoredFilename))
	item.ContentType = strings.TrimSpace(item.ContentType)
	item.UploadedBy = textutil.Email(item.UploadedBy)
	if item.Size < 0 {
		item.Size = 0
	}
	if item.UploadedAt.IsZero() {
		item.UploadedAt = time.Now()
	}
	item.UploadedAt = item.UploadedAt.UTC()
	return item
}

func CopyDocument(item DocumentRecord) DocumentRecord {
	return item
}

func SortDocuments(items []DocumentRecord) {
	sort.SliceStable(items, func(i, j int) bool {
		if !items[i].UploadedAt.Equal(items[j].UploadedAt) {
			return items[i].UploadedAt.After(items[j].UploadedAt)
		}
		if items[i].Category != items[j].Category {
			return items[i].Category < items[j].Category
		}
		return strings.ToLower(items[i].Title) < strings.ToLower(items[j].Title)
	})
}

func DocumentExtension(contentType string) (string, bool) {
	switch stripContentTypeParams(contentType) {
	case "application/pdf":
		return ".pdf", true
	case "image/jpeg":
		return ".jpg", true
	case "image/png":
		return ".png", true
	case "image/webp":
		return ".webp", true
	case "application/xml", "text/xml":
		return ".xml", true
	default:
		return "", false
	}
}

func SanitizeDocumentFilename(raw string) string {
	name := filepath.Base(strings.TrimSpace(raw))
	if name == "." || name == string(filepath.Separator) {
		name = ""
	}
	name = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 || r == '/' || r == '\\' {
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	if name == "" {
		return "dokument"
	}
	if len([]rune(name)) > 120 {
		runes := []rune(name)
		name = string(runes[:120])
	}
	return name
}

func NormalizeDocumentCategory(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case strings.ToLower(DocumentCategoryProtocol):
		return DocumentCategoryProtocol
	case strings.ToLower(DocumentCategoryBilling):
		return DocumentCategoryBilling
	case strings.ToLower(DocumentCategoryRules):
		return DocumentCategoryRules
	case strings.ToLower(DocumentCategoryContract):
		return DocumentCategoryContract
	case strings.ToLower(DocumentCategoryPlan), "pläne", "plaene":
		return DocumentCategoryPlan
	case "", strings.ToLower(DocumentCategoryOther):
		return DocumentCategoryOther
	default:
		return ""
	}
}

func NormalizeDocumentVisibility(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case DocumentVisibilityAllResidents, "all", "alle", "alle-bewohner":
		return DocumentVisibilityAllResidents
	case DocumentVisibilityOwnersOnly, "owner", "owners", "eigentuemer", "eigentümer":
		return DocumentVisibilityOwnersOnly
	case DocumentVisibilityManagerOnly, "manager", "verwalter", "verwaltung":
		return DocumentVisibilityManagerOnly
	default:
		return ""
	}
}

func NewParkingStore(path string) (*ParkingStore, error) {
	store := &ParkingStore{
		path: path,
		data: ParkingStoreData{Tenants: map[string]ParkingTenantData{}},
	}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read parking data")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("invalid parking data")
	}
	if store.data.Tenants == nil {
		store.data.Tenants = map[string]ParkingTenantData{}
	}
	for slug, data := range store.data.Tenants {
		data.Settings = NormalizeParkingSettings(data.Settings)
		data.Months = NormalizeParkingMonthStates(data.Months)
		store.data.Tenants[textutil.Slug(slug)] = data
	}
	return store, nil
}

func (s *ParkingStore) TenantData(tenantSlug string) ParkingTenantData {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.tenantLocked(tenantSlug)
	data.Settings = NormalizeParkingSettings(data.Settings)
	data.Months = CopyMonthStates(data.Months)
	data.EnergySamples = append([]ParkingNumericSample(nil), data.EnergySamples...)
	data.PriceSamples = append([]ParkingNumericSample(nil), data.PriceSamples...)
	data.Samples = append([]ParkingStoredSample(nil), data.Samples...)
	return data
}

func (s *ParkingStore) SetGridFee(tenantSlug string, gridFee float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.tenantLocked(tenantSlug)
	data.Settings = NormalizeParkingSettings(data.Settings)
	if len(data.Settings.Tariffs) > 0 {
		data.Settings.Tariffs[len(data.Settings.Tariffs)-1].GridFeeEURPerKWh = gridFee
	}
	data.Settings.GridFeeEURPerKWh = gridFee
	data.Settings = NormalizeParkingSettings(data.Settings)
	s.data.Tenants[textutil.Slug(tenantSlug)] = data
	return s.saveLocked()
}

func (s *ParkingStore) UpsertTariff(tenantSlug string, tariff ParkingTariff) error {
	tariff = NormalizeParkingTariff(tariff)
	if tariff.EffectiveFrom == "" {
		return fmt.Errorf("invalid tariff")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.tenantLocked(tenantSlug)
	replaced := false
	for i, existing := range data.Settings.Tariffs {
		if existing.EffectiveFrom == tariff.EffectiveFrom {
			data.Settings.Tariffs[i] = tariff
			replaced = true
			break
		}
	}
	if !replaced {
		data.Settings.Tariffs = append(data.Settings.Tariffs, tariff)
	}
	data.Settings.GridFeeEURPerKWh = tariff.GridFeeEURPerKWh
	data.Settings.BaseFeeEUR = tariff.BaseFeeEUR
	data.Settings = NormalizeParkingSettings(data.Settings)
	s.data.Tenants[textutil.Slug(tenantSlug)] = data
	return s.saveLocked()
}

func (s *ParkingStore) SetMonthPaid(tenantSlug string, month string, paid bool) error {
	return s.SetMonthPayment(tenantSlug, month, ParkingMonthState{Paid: paid})
}

func (s *ParkingStore) SetMonthPayment(tenantSlug string, month string, state ParkingMonthState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.tenantLocked(tenantSlug)
	if data.Months == nil {
		data.Months = map[string]ParkingMonthState{}
	}
	data.Months[month] = NormalizeParkingMonthState(state)
	s.data.Tenants[textutil.Slug(tenantSlug)] = data
	return s.saveLocked()
}

func (s *ParkingStore) MarkPaymentReminderSent(tenantSlug string, months []string, recipients []string, at time.Time) error {
	tenantSlug = textutil.Slug(tenantSlug)
	months = NormalizeParkingMonths(months)
	recipients = UniqueEmails(recipients)
	if tenantSlug == "" || len(months) == 0 || len(recipients) == 0 {
		return nil
	}
	if at.IsZero() {
		at = time.Now()
	}
	at = at.UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.tenantLocked(tenantSlug)
	if data.Months == nil {
		data.Months = map[string]ParkingMonthState{}
	}
	for _, month := range months {
		state := NormalizeParkingMonthState(data.Months[month])
		if state.Paid {
			continue
		}
		if state.ReminderSentAt == nil {
			state.ReminderSentAt = map[string]time.Time{}
		}
		for _, recipient := range recipients {
			state.ReminderSentAt[recipient] = at
		}
		data.Months[month] = NormalizeParkingMonthState(state)
	}
	s.data.Tenants[textutil.Slug(tenantSlug)] = data
	return s.saveLocked()
}

func (s *ParkingStore) AppendSamples(tenantSlug string, samples []ParkingStoredSample) error {
	if len(samples) == 0 {
		return nil
	}
	energy := make([]ParkingNumericSample, 0, len(samples))
	prices := make([]ParkingNumericSample, 0, len(samples))
	for _, sample := range samples {
		energy = append(energy, ParkingNumericSample{At: sample.At, Value: sample.EnergyKWh})
		prices = append(prices, ParkingNumericSample{At: sample.At, Value: sample.PriceEURPerKWh})
	}
	return s.AppendReadings(tenantSlug, energy, prices)
}

func (s *ParkingStore) AppendReadings(tenantSlug string, energySamples []ParkingNumericSample, priceSamples []ParkingNumericSample) error {
	if len(energySamples) == 0 && len(priceSamples) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.tenantLocked(tenantSlug)
	for _, sample := range energySamples {
		if sample.At.IsZero() || sample.Value < 0 || sample.Value > 1000000 {
			continue
		}
		data.EnergySamples = append(data.EnergySamples, ParkingNumericSample{
			At:    sample.At.UTC(),
			Value: sample.Value,
		})
	}
	for _, sample := range priceSamples {
		if sample.At.IsZero() || sample.Value < -5 || sample.Value > 5 {
			continue
		}
		data.PriceSamples = append(data.PriceSamples, ParkingNumericSample{
			At:    sample.At.UTC(),
			Value: sample.Value,
		})
	}
	keepAfter := time.Now().AddDate(-1, -1, 0)
	data.EnergySamples = NormalizeNumericSamples(data.EnergySamples, keepAfter)
	data.PriceSamples = NormalizeNumericSamples(data.PriceSamples, keepAfter)
	data.Samples = nil
	s.data.Tenants[textutil.Slug(tenantSlug)] = data
	return s.saveLocked()
}

func (s *ParkingStore) tenantLocked(tenantSlug string) ParkingTenantData {
	tenantSlug = textutil.Slug(tenantSlug)
	if tenantSlug == "" {
		tenantSlug = "default"
	}
	if s.data.Tenants == nil {
		s.data.Tenants = map[string]ParkingTenantData{}
	}
	data, ok := s.data.Tenants[tenantSlug]
	if !ok {
		data = DefaultParkingTenantData()
	}
	data.Settings = NormalizeParkingSettings(data.Settings)
	if data.Months == nil {
		data.Months = map[string]ParkingMonthState{}
	}
	data.Months = NormalizeParkingMonthStates(data.Months)
	if len(data.Samples) > 0 {
		for _, sample := range data.Samples {
			data.EnergySamples = append(data.EnergySamples, ParkingNumericSample{At: sample.At, Value: sample.EnergyKWh})
			data.PriceSamples = append(data.PriceSamples, ParkingNumericSample{At: sample.At, Value: sample.PriceEURPerKWh})
		}
		data.Samples = nil
	}
	keepAfter := time.Now().AddDate(-1, -1, 0)
	data.EnergySamples = NormalizeNumericSamples(data.EnergySamples, keepAfter)
	data.PriceSamples = NormalizeNumericSamples(data.PriceSamples, keepAfter)
	s.data.Tenants[tenantSlug] = data
	return data
}

func (s *ParkingStore) saveLocked() error {
	return SaveJSONAtomic(s.path, s.data, "parking")
}

func NormalizeParkingSettings(settings ParkingSettings) ParkingSettings {
	if settings.GridFeeEURPerKWh < 0 {
		settings.GridFeeEURPerKWh = 0
	}
	if settings.GridFeeEURPerKWh > 5 {
		settings.GridFeeEURPerKWh = 5
	}
	if settings.BaseFeeEUR < 0 {
		settings.BaseFeeEUR = 0
	}
	if settings.BaseFeeEUR > 5000 {
		settings.BaseFeeEUR = 5000
	}
	tariffs := make([]ParkingTariff, 0, len(settings.Tariffs)+1)
	seen := map[string]int{}
	for _, tariff := range settings.Tariffs {
		tariff = NormalizeParkingTariff(tariff)
		if tariff.EffectiveFrom == "" {
			continue
		}
		if idx, ok := seen[tariff.EffectiveFrom]; ok {
			tariffs[idx] = tariff
			continue
		}
		seen[tariff.EffectiveFrom] = len(tariffs)
		tariffs = append(tariffs, tariff)
	}
	if len(tariffs) == 0 {
		tariffs = append(tariffs, ParkingTariff{
			EffectiveFrom:    "2000-01-01",
			GridFeeEURPerKWh: settings.GridFeeEURPerKWh,
			BaseFeeEUR:       settings.BaseFeeEUR,
		})
	}
	sort.Slice(tariffs, func(i, j int) bool {
		return tariffs[i].EffectiveFrom < tariffs[j].EffectiveFrom
	})
	settings.Tariffs = tariffs
	current := tariffs[len(tariffs)-1]
	settings.GridFeeEURPerKWh = current.GridFeeEURPerKWh
	settings.BaseFeeEUR = current.BaseFeeEUR
	return settings
}

func NormalizeParkingTariff(tariff ParkingTariff) ParkingTariff {
	tariff.EffectiveFrom = NormalizeParkingTariffDate(tariff.EffectiveFrom)
	if tariff.GridFeeEURPerKWh < 0 {
		tariff.GridFeeEURPerKWh = 0
	}
	if tariff.GridFeeEURPerKWh > 5 {
		tariff.GridFeeEURPerKWh = 5
	}
	if tariff.BaseFeeEUR < 0 {
		tariff.BaseFeeEUR = 0
	}
	if tariff.BaseFeeEUR > 5000 {
		tariff.BaseFeeEUR = 5000
	}
	return tariff
}

func NormalizeParkingTariffDate(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if t, err := time.Parse("2006-01-02", raw); err == nil {
		return t.Format("2006-01-02")
	}
	if t, err := time.Parse("2006-01", raw); err == nil {
		return t.Format("2006-01-02")
	}
	return ""
}

func NormalizeParkingMonthStates(in map[string]ParkingMonthState) map[string]ParkingMonthState {
	out := map[string]ParkingMonthState{}
	for month, state := range in {
		if _, err := time.Parse("2006-01", month); err != nil {
			continue
		}
		out[month] = NormalizeParkingMonthState(state)
	}
	return out
}

func NormalizeParkingMonthState(state ParkingMonthState) ParkingMonthState {
	if !state.Paid {
		state.PaidAt = time.Time{}
		state.PaidBy = ""
		state.PaymentMethod = ""
		state.PaymentReference = ""
	}
	if !state.PaidAt.IsZero() {
		state.PaidAt = state.PaidAt.UTC()
	}
	state.PaidBy = textutil.Email(state.PaidBy)
	state.PaymentMethod = CleanParkingPaymentField(state.PaymentMethod)
	state.PaymentReference = CleanParkingPaymentField(state.PaymentReference)
	if len(state.ReminderSentAt) == 0 {
		state.ReminderSentAt = nil
		return state
	}
	sent := map[string]time.Time{}
	for email, at := range state.ReminderSentAt {
		email = textutil.Email(email)
		if email == "" || at.IsZero() {
			continue
		}
		sent[email] = at.UTC()
	}
	if len(sent) == 0 {
		state.ReminderSentAt = nil
	} else {
		state.ReminderSentAt = sent
	}
	return state
}

func NormalizeParkingMonths(months []string) []string {
	out := []string{}
	seen := map[string]struct{}{}
	for _, month := range months {
		month = strings.TrimSpace(month)
		if _, err := time.Parse("2006-01", month); err != nil {
			continue
		}
		if _, ok := seen[month]; ok {
			continue
		}
		seen[month] = struct{}{}
		out = append(out, month)
	}
	sort.Strings(out)
	return out
}

type HandoverStore struct {
	mu   sync.Mutex
	path string
	data HandoverStoreData
}

type HandoverStoreData struct {
	Handovers []HandoverRecord `json:"handovers"`
}

type HandoverRecord struct {
	ID              string                 `json:"id"`
	TenantSlug      string                 `json:"tenant"`
	UnitID          string                 `json:"unit_id,omitempty"`
	Title           string                 `json:"title"`
	HandoverType    string                 `json:"handover_type"`
	ScheduledAt     time.Time              `json:"scheduled_at,omitempty"`
	OutgoingName    string                 `json:"outgoing_name,omitempty"`
	OutgoingEmail   string                 `json:"outgoing_email,omitempty"`
	IncomingName    string                 `json:"incoming_name,omitempty"`
	IncomingEmail   string                 `json:"incoming_email,omitempty"`
	Rooms           []HandoverRoom         `json:"rooms,omitempty"`
	Meters          []HandoverMeter        `json:"meters,omitempty"`
	Keys            []HandoverKey          `json:"keys,omitempty"`
	Notes           string                 `json:"notes,omitempty"`
	Confirmations   []HandoverConfirmation `json:"confirmations,omitempty"`
	FiledDocumentID string                 `json:"filed_document_id,omitempty"`
	CreatedBy       string                 `json:"created_by"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

type HandoverRoom struct {
	Name      string `json:"name"`
	Condition string `json:"condition,omitempty"`
	Defects   string `json:"defects,omitempty"`
}

type HandoverMeter struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Unit  string `json:"unit,omitempty"`
}

type HandoverKey struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

type HandoverConfirmation struct {
	Role        string    `json:"role"`
	Name        string    `json:"name,omitempty"`
	Email       string    `json:"email,omitempty"`
	TokenHash   string    `json:"token_hash,omitempty"`
	ConfirmedAt time.Time `json:"confirmed_at,omitempty"`
	ConfirmedBy string    `json:"confirmed_by,omitempty"`
	Note        string    `json:"note,omitempty"`
}

type HandoverTokenDelivery struct {
	Role  string
	Name  string
	Email string
	Token string
}

func NewHandoverStore(path string) (*HandoverStore, error) {
	store := &HandoverStore{path: path, data: HandoverStoreData{Handovers: []HandoverRecord{}}}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read handover data")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("invalid handover data")
	}
	store.data.Handovers = NormalizeHandovers(store.data.Handovers)
	return store, nil
}

func (s *HandoverStore) Create(item HandoverRecord) (HandoverRecord, error) {
	if s == nil {
		return HandoverRecord{}, fmt.Errorf("handover store unavailable")
	}
	item = NormalizeHandover(item)
	if item.ID == "" || item.TenantSlug == "" || item.Title == "" || item.CreatedBy == "" {
		return HandoverRecord{}, fmt.Errorf("invalid handover")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.data.Handovers {
		if textutil.Slug(existing.TenantSlug) == item.TenantSlug && existing.ID == item.ID {
			return HandoverRecord{}, fmt.Errorf("handover exists")
		}
	}
	s.data.Handovers = append(s.data.Handovers, item)
	SortHandovers(s.data.Handovers)
	if err := s.saveLocked(); err != nil {
		return HandoverRecord{}, err
	}
	return CopyHandover(item), nil
}

func (s *HandoverStore) ListTenant(tenantSlug string) []HandoverRecord {
	if s == nil {
		return nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []HandoverRecord{}
	for _, item := range s.data.Handovers {
		if textutil.Slug(item.TenantSlug) == tenantSlug {
			out = append(out, CopyHandover(item))
		}
	}
	SortHandovers(out)
	return out
}

func (s *HandoverStore) Get(tenantSlug string, id string) (HandoverRecord, bool) {
	if s == nil {
		return HandoverRecord{}, false
	}
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.data.Handovers {
		if textutil.Slug(item.TenantSlug) == tenantSlug && item.ID == id {
			return CopyHandover(item), true
		}
	}
	return HandoverRecord{}, false
}

func (s *HandoverStore) GetByToken(token string) (HandoverRecord, int, bool) {
	if s == nil || strings.TrimSpace(token) == "" {
		return HandoverRecord{}, -1, false
	}
	hash := HandoverTokenHash(token)
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.data.Handovers {
		for idx, confirmation := range item.Confirmations {
			if confirmation.TokenHash != "" && SubtleConstantStringCompare(confirmation.TokenHash, hash) {
				return CopyHandover(item), idx, true
			}
		}
	}
	return HandoverRecord{}, -1, false
}

func (s *HandoverStore) ConfirmByToken(token string, name string, note string, at time.Time) (HandoverRecord, HandoverConfirmation, bool, error) {
	if s == nil || strings.TrimSpace(token) == "" {
		return HandoverRecord{}, HandoverConfirmation{}, false, nil
	}
	hash := HandoverTokenHash(token)
	name = textutil.Truncate(strings.TrimSpace(name), 120)
	note = textutil.Truncate(strings.TrimSpace(note), 500)
	if at.IsZero() {
		at = time.Now()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, item := range s.data.Handovers {
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
			s.data.Handovers[i] = NormalizeHandover(item)
			SortHandovers(s.data.Handovers)
			if err := s.saveLocked(); err != nil {
				return HandoverRecord{}, HandoverConfirmation{}, true, err
			}
			return CopyHandover(s.data.Handovers[i]), confirmation, true, nil
		}
	}
	return HandoverRecord{}, HandoverConfirmation{}, false, nil
}

func (s *HandoverStore) SetFiledDocument(tenantSlug string, id string, documentID string, at time.Time) (HandoverRecord, bool, error) {
	if s == nil {
		return HandoverRecord{}, false, fmt.Errorf("handover store unavailable")
	}
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	documentID = strings.TrimSpace(documentID)
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, item := range s.data.Handovers {
		if textutil.Slug(item.TenantSlug) != tenantSlug || item.ID != id {
			continue
		}
		item.FiledDocumentID = documentID
		if at.IsZero() {
			at = time.Now()
		}
		item.UpdatedAt = at.UTC()
		s.data.Handovers[i] = NormalizeHandover(item)
		if err := s.saveLocked(); err != nil {
			return HandoverRecord{}, true, err
		}
		return CopyHandover(s.data.Handovers[i]), true, nil
	}
	return HandoverRecord{}, false, nil
}

func (s *HandoverStore) saveLocked() error {
	return SaveJSONAtomic(s.path, s.data, "handover")
}

func NormalizeHandovers(items []HandoverRecord) []HandoverRecord {
	out := make([]HandoverRecord, 0, len(items))
	for _, item := range items {
		item = NormalizeHandover(item)
		if item.ID == "" || item.TenantSlug == "" || item.Title == "" {
			continue
		}
		out = append(out, item)
	}
	SortHandovers(out)
	return out
}

func NormalizeHandover(item HandoverRecord) HandoverRecord {
	item.ID = strings.TrimSpace(item.ID)
	item.TenantSlug = textutil.Slug(item.TenantSlug)
	item.UnitID = NormalizeUnitID(item.UnitID)
	item.Title = textutil.Truncate(strings.TrimSpace(item.Title), 160)
	item.HandoverType = NormalizeHandoverType(item.HandoverType)
	item.OutgoingName = textutil.Truncate(strings.TrimSpace(item.OutgoingName), 120)
	item.OutgoingEmail = textutil.Email(item.OutgoingEmail)
	item.IncomingName = textutil.Truncate(strings.TrimSpace(item.IncomingName), 120)
	item.IncomingEmail = textutil.Email(item.IncomingEmail)
	item.Notes = textutil.Truncate(strings.TrimSpace(item.Notes), 3000)
	item.CreatedBy = textutil.Email(item.CreatedBy)
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
	if !item.ScheduledAt.IsZero() {
		item.ScheduledAt = item.ScheduledAt.UTC()
	}
	item.Rooms = NormalizeHandoverRooms(item.Rooms)
	item.Meters = NormalizeHandoverMeters(item.Meters)
	item.Keys = NormalizeHandoverKeys(item.Keys)
	item.Confirmations = NormalizeHandoverConfirmations(item.Confirmations)
	item.FiledDocumentID = strings.TrimSpace(item.FiledDocumentID)
	return item
}

func NormalizeHandoverType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "einzug", "move-in", "in":
		return "Einzug"
	case "auszug", "move-out", "out":
		return "Auszug"
	case "wechsel", "nutzerwechsel", "handover", "":
		return "Nutzerwechsel"
	default:
		return textutil.Truncate(strings.TrimSpace(raw), 80)
	}
}

func NormalizeHandoverRooms(items []HandoverRoom) []HandoverRoom {
	out := []HandoverRoom{}
	for _, item := range items {
		item.Name = textutil.Truncate(strings.TrimSpace(item.Name), 100)
		item.Condition = textutil.Truncate(strings.TrimSpace(item.Condition), 120)
		item.Defects = textutil.Truncate(strings.TrimSpace(item.Defects), 500)
		if item.Name != "" || item.Condition != "" || item.Defects != "" {
			out = append(out, item)
		}
	}
	return out
}

func NormalizeHandoverMeters(items []HandoverMeter) []HandoverMeter {
	out := []HandoverMeter{}
	for _, item := range items {
		item.Label = textutil.Truncate(strings.TrimSpace(item.Label), 100)
		item.Value = textutil.Truncate(strings.TrimSpace(item.Value), 80)
		item.Unit = textutil.Truncate(strings.TrimSpace(item.Unit), 40)
		if item.Label != "" && item.Value != "" {
			out = append(out, item)
		}
	}
	return out
}

func NormalizeHandoverKeys(items []HandoverKey) []HandoverKey {
	out := []HandoverKey{}
	for _, item := range items {
		item.Label = textutil.Truncate(strings.TrimSpace(item.Label), 100)
		if item.Count < 0 {
			item.Count = 0
		}
		if item.Label != "" && item.Count > 0 {
			out = append(out, item)
		}
	}
	return out
}

func NormalizeHandoverConfirmations(items []HandoverConfirmation) []HandoverConfirmation {
	out := []HandoverConfirmation{}
	for _, item := range items {
		item.Role = textutil.Truncate(strings.TrimSpace(item.Role), 80)
		item.Name = textutil.Truncate(strings.TrimSpace(item.Name), 120)
		item.Email = textutil.Email(item.Email)
		item.TokenHash = strings.TrimSpace(item.TokenHash)
		item.ConfirmedBy = textutil.Truncate(strings.TrimSpace(item.ConfirmedBy), 120)
		item.Note = textutil.Truncate(strings.TrimSpace(item.Note), 500)
		if !item.ConfirmedAt.IsZero() {
			item.ConfirmedAt = item.ConfirmedAt.UTC()
		}
		if item.Role != "" && (item.Name != "" || item.Email != "") {
			out = append(out, item)
		}
	}
	return out
}

func SortHandovers(items []HandoverRecord) {
	sort.SliceStable(items, func(i, j int) bool {
		leftDate := items[i].ScheduledAt
		rightDate := items[j].ScheduledAt
		if leftDate.IsZero() {
			leftDate = items[i].CreatedAt
		}
		if rightDate.IsZero() {
			rightDate = items[j].CreatedAt
		}
		if !leftDate.Equal(rightDate) {
			return leftDate.After(rightDate)
		}
		return strings.ToLower(items[i].Title) < strings.ToLower(items[j].Title)
	})
}

func CopyHandover(item HandoverRecord) HandoverRecord {
	item.Rooms = append([]HandoverRoom(nil), item.Rooms...)
	item.Meters = append([]HandoverMeter(nil), item.Meters...)
	item.Keys = append([]HandoverKey(nil), item.Keys...)
	item.Confirmations = append([]HandoverConfirmation(nil), item.Confirmations...)
	return item
}

func CopyMonthStates(in map[string]ParkingMonthState) map[string]ParkingMonthState {
	out := map[string]ParkingMonthState{}
	for month, state := range in {
		out[month] = NormalizeParkingMonthState(state)
	}
	return out
}

func UniqueEmails(raw []string) []string {
	return NormalizeEmailList(raw)
}

func DefaultParkingTenantData() ParkingTenantData {
	return ParkingTenantData{
		Settings: NormalizeParkingSettings(ParkingSettings{GridFeeEURPerKWh: 0.10}),
		Months:   map[string]ParkingMonthState{},
	}
}

func NormalizeNumericSamples(samples []ParkingNumericSample, keepAfter time.Time) []ParkingNumericSample {
	out := make([]ParkingNumericSample, 0, len(samples))
	for _, sample := range samples {
		if sample.At.IsZero() || sample.At.Before(keepAfter) {
			continue
		}
		out = append(out, ParkingNumericSample{At: sample.At.UTC().Truncate(time.Second), Value: sample.Value})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].At.Before(out[j].At)
	})
	deduped := out[:0]
	for _, sample := range out {
		if len(deduped) > 0 && deduped[len(deduped)-1].At.Equal(sample.At) {
			deduped[len(deduped)-1] = sample
			continue
		}
		deduped = append(deduped, sample)
	}
	return deduped
}

func CleanParkingPaymentField(raw string) string {
	raw = strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
	return TruncateAuditValue(raw, 120)
}

func HandoverTokenHash(token string) string {
	sum := sha256.Sum256([]byte("handover-confirmation-v1:" + strings.TrimSpace(token)))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func SubtleConstantStringCompare(a string, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var out byte
	for i := 0; i < len(a); i++ {
		out |= a[i] ^ b[i]
	}
	return out == 0
}

// AttachmentDir exposes the issue store's photo directory. main used to reach
// into the unexported field directly (legacy issue photos are served from disk);
// crossing a package boundary needs a real accessor.
func (s *IssueStore) AttachmentDir() string {
	if s == nil {
		return ""
	}
	return s.attachmentDir
}
