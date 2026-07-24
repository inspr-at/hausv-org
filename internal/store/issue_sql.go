package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/markus-barta/hausv-org/internal/textutil"
)

// IssueStorage is the behaviour both the JSON IssueStore and the SQLite
// SQLIssueStore satisfy (HAUSV-168). SavePhoto is deliberately absent: it has
// no callers left — issue photos go through the attachment store.
type IssueStorage interface {
	Create(item ResidentIssue) (ResidentIssue, error)
	ListTenant(tenantSlug string) []ResidentIssue
	ListAuthor(tenantSlug string, email string) []ResidentIssue
	Get(tenantSlug string, id string) (ResidentIssue, bool)
	UpdateWorkflow(tenantSlug string, id string, update IssueWorkflowUpdate) (ResidentIssue, bool, error)
	AddComment(tenantSlug string, id string, comment IssueComment) (ResidentIssue, bool, error)
	DeleteComment(tenantSlug string, id string, commentID string, at time.Time) (ResidentIssue, bool, error)
	AttachmentDir() string
}

var (
	_ IssueStorage = (*IssueStore)(nil)
	_ IssueStorage = (*SQLIssueStore)(nil)
)

// SQLIssueStore keeps each issue — comments and status history included — as
// one JSON document keyed by (tenant, id). Table from migration 0016.
type SQLIssueStore struct {
	db            *sql.DB
	attachmentDir string
}

func NewSQLIssueStore(db *sql.DB, attachmentDir string) *SQLIssueStore {
	return &SQLIssueStore{db: db, attachmentDir: attachmentDir}
}

func (s *SQLIssueStore) AttachmentDir() string {
	if s == nil {
		return ""
	}
	return s.attachmentDir
}

func (s *SQLIssueStore) writeTx(tx *sql.Tx, item ResidentIssue) error {
	blob, err := json.Marshal(item)
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		`INSERT INTO issues(tenant_slug, id, data) VALUES(?, ?, ?)
		 ON CONFLICT(tenant_slug, id) DO UPDATE SET data=excluded.data`,
		textutil.Slug(item.TenantSlug), item.ID, string(blob),
	)
	return err
}

func loadIssueTx(tx *sql.Tx, tenantSlug string, id string) (ResidentIssue, bool) {
	var data string
	if err := tx.QueryRow(`SELECT data FROM issues WHERE tenant_slug=? AND id=?`, tenantSlug, id).Scan(&data); err != nil {
		return ResidentIssue{}, false
	}
	var item ResidentIssue
	if err := json.Unmarshal([]byte(data), &item); err != nil {
		return ResidentIssue{}, false
	}
	return item, true
}

func (s *SQLIssueStore) Create(item ResidentIssue) (ResidentIssue, error) {
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
	tx, err := s.db.Begin()
	if err != nil {
		return ResidentIssue{}, err
	}
	defer tx.Rollback()
	if err := s.writeTx(tx, item); err != nil {
		return ResidentIssue{}, err
	}
	if err := tx.Commit(); err != nil {
		return ResidentIssue{}, err
	}
	return item, nil
}

func (s *SQLIssueStore) allForTenant(tenantSlug string) []ResidentIssue {
	rows, err := s.db.Query(`SELECT data FROM issues WHERE tenant_slug=?`, tenantSlug)
	if err != nil {
		return []ResidentIssue{}
	}
	defer rows.Close()
	out := []ResidentIssue{}
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			continue
		}
		var item ResidentIssue
		if err := json.Unmarshal([]byte(data), &item); err != nil {
			continue
		}
		out = append(out, item)
	}
	return out
}

func (s *SQLIssueStore) ListTenant(tenantSlug string) []ResidentIssue {
	if s == nil {
		return nil
	}
	out := []ResidentIssue{}
	for _, item := range s.allForTenant(textutil.Slug(tenantSlug)) {
		out = append(out, CopyIssue(item))
	}
	SortIssues(out)
	return out
}

func (s *SQLIssueStore) ListAuthor(tenantSlug string, email string) []ResidentIssue {
	if s == nil {
		return nil
	}
	email = textutil.Email(email)
	out := []ResidentIssue{}
	for _, item := range s.allForTenant(textutil.Slug(tenantSlug)) {
		if textutil.Email(item.AuthorEmail) == email {
			out = append(out, CopyIssue(item))
		}
	}
	SortIssues(out)
	return out
}

func (s *SQLIssueStore) Get(tenantSlug string, id string) (ResidentIssue, bool) {
	if s == nil {
		return ResidentIssue{}, false
	}
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	if tenantSlug == "" || id == "" {
		return ResidentIssue{}, false
	}
	var data string
	if err := s.db.QueryRow(`SELECT data FROM issues WHERE tenant_slug=? AND id=?`, tenantSlug, id).Scan(&data); err != nil {
		return ResidentIssue{}, false
	}
	var item ResidentIssue
	if err := json.Unmarshal([]byte(data), &item); err != nil {
		return ResidentIssue{}, false
	}
	return CopyIssue(item), true
}

func (s *SQLIssueStore) UpdateWorkflow(tenantSlug string, id string, update IssueWorkflowUpdate) (ResidentIssue, bool, error) {
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

	tx, err := s.db.Begin()
	if err != nil {
		return ResidentIssue{}, false, err
	}
	defer tx.Rollback()
	existing, found := loadIssueTx(tx, tenantSlug, id)
	if !found {
		return ResidentIssue{}, false, nil
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
		start := update.ServiceProposedStart
		end := update.ServiceProposedEnd
		if !start.IsZero() {
			start = start.UTC()
		}
		if !end.IsZero() {
			end = end.UTC()
		}
		updated.ServiceProposedStart = start
		updated.ServiceProposedEnd = end
		if updated.ServiceProposal == "" && start.IsZero() {
			updated.ServiceProposedBy = ""
			updated.ServiceProposedAt = time.Time{}
			updated.ServiceProposedEnd = time.Time{}
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
	if err := s.writeTx(tx, updated); err != nil {
		return ResidentIssue{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return ResidentIssue{}, false, err
	}
	return CopyIssue(updated), true, nil
}

func (s *SQLIssueStore) AddComment(tenantSlug string, id string, comment IssueComment) (ResidentIssue, bool, error) {
	if s == nil {
		return ResidentIssue{}, false, nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	comment.Body = strings.TrimSpace(comment.Body)
	comment.AuthorEmail = textutil.Email(comment.AuthorEmail)
	comment.AuthorName = strings.TrimSpace(comment.AuthorName)
	// Body may be empty for a photo-only comment (HAUSV-128).
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
	tx, err := s.db.Begin()
	if err != nil {
		return ResidentIssue{}, false, err
	}
	defer tx.Rollback()
	existing, found := loadIssueTx(tx, tenantSlug, id)
	if !found {
		return ResidentIssue{}, false, nil
	}
	updated := existing
	updated.Comments = append(updated.Comments, comment)
	updated.UpdatedAt = comment.CreatedAt
	if err := s.writeTx(tx, updated); err != nil {
		return ResidentIssue{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return ResidentIssue{}, false, err
	}
	return CopyIssue(updated), true, nil
}

func (s *SQLIssueStore) DeleteComment(tenantSlug string, id string, commentID string, at time.Time) (ResidentIssue, bool, error) {
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
	tx, err := s.db.Begin()
	if err != nil {
		return ResidentIssue{}, false, err
	}
	defer tx.Rollback()
	existing, found := loadIssueTx(tx, tenantSlug, id)
	if !found {
		return ResidentIssue{}, false, nil
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
	if err := s.writeTx(tx, updated); err != nil {
		return ResidentIssue{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return ResidentIssue{}, false, err
	}
	return CopyIssue(updated), true, nil
}

// ImportIssues copies records from a JSON store, each only if absent
// (clobber-safe) (HAUSV-170).
func (s *SQLIssueStore) ImportIssues(src *IssueStore) error {
	if src == nil {
		return nil
	}
	src.mu.Lock()
	snapshot := append([]ResidentIssue(nil), src.data.Issues...)
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
			`INSERT INTO issues(tenant_slug, id, data) VALUES(?, ?, ?) ON CONFLICT(tenant_slug, id) DO NOTHING`,
			tenant, item.ID, string(blob),
		); err != nil {
			return err
		}
	}
	return nil
}
