package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

// IssueRepository is an issue store already bound to one tenant.
type IssueRepository interface {
	Create(item ResidentIssue) (ResidentIssue, error)
	List() []ResidentIssue
	ListAuthor(email string) []ResidentIssue
	Get(id string) (ResidentIssue, bool)
	UpdateWorkflow(id string, update IssueWorkflowUpdate) (ResidentIssue, bool, error)
	AddComment(id string, comment IssueComment) (ResidentIssue, bool, error)
	DeleteComment(id string, commentID string, at time.Time) (ResidentIssue, bool, error)
	// ClearPhotoPaths drops the legacy photo list after those photos have been
	// moved into the attachment store (HAUSV-175).
	ClearPhotoPaths(id string) (bool, error)
}

// IssueStorage is the unbound backend. Tenant-aware operations stay private to
// this package; request handlers receive only IssueRepository.
type IssueStorage interface {
	issueStorage()
}

var (
	_ IssueStorage = (*IssueStore)(nil)
	_ IssueStorage = (*SQLIssueStore)(nil)
)

type issueBackend interface {
	create(tenant TenantRef, item ResidentIssue) (ResidentIssue, error)
	listTenant(tenant TenantRef) []ResidentIssue
	listAuthor(tenant TenantRef, email string) []ResidentIssue
	get(tenant TenantRef, id string) (ResidentIssue, bool)
	updateWorkflow(tenant TenantRef, id string, update IssueWorkflowUpdate) (ResidentIssue, bool, error)
	addComment(tenant TenantRef, id string, comment IssueComment) (ResidentIssue, bool, error)
	deleteComment(tenant TenantRef, id string, commentID string, at time.Time) (ResidentIssue, bool, error)
	clearPhotoPaths(tenant TenantRef, id string) (bool, error)
}

type boundIssueRepository struct {
	storage issueBackend
	tenant  TenantRef
}

func BindIssueRepository(storage IssueStorage, tenant TenantRef) (IssueRepository, bool) {
	resolvedTenant, tenantOK := validTenantRef(tenant)
	backend, ok := storage.(issueBackend)
	if !ok || !tenantOK {
		return nil, false
	}
	return &boundIssueRepository{storage: backend, tenant: resolvedTenant}, true
}

func (r *boundIssueRepository) Create(item ResidentIssue) (ResidentIssue, error) {
	return r.storage.create(r.tenant, item)
}
func (r *boundIssueRepository) List() []ResidentIssue { return r.storage.listTenant(r.tenant) }
func (r *boundIssueRepository) ListAuthor(email string) []ResidentIssue {
	return r.storage.listAuthor(r.tenant, email)
}
func (r *boundIssueRepository) Get(id string) (ResidentIssue, bool) {
	return r.storage.get(r.tenant, id)
}
func (r *boundIssueRepository) UpdateWorkflow(id string, update IssueWorkflowUpdate) (ResidentIssue, bool, error) {
	return r.storage.updateWorkflow(r.tenant, id, update)
}
func (r *boundIssueRepository) AddComment(id string, comment IssueComment) (ResidentIssue, bool, error) {
	return r.storage.addComment(r.tenant, id, comment)
}
func (r *boundIssueRepository) DeleteComment(id string, commentID string, at time.Time) (ResidentIssue, bool, error) {
	return r.storage.deleteComment(r.tenant, id, commentID, at)
}
func (r *boundIssueRepository) ClearPhotoPaths(id string) (bool, error) {
	return r.storage.clearPhotoPaths(r.tenant, id)
}

// SQLIssueStore keeps each issue — comments and status history included — as
// one JSON document keyed by (tenant, id). Table from migration 0016.
type SQLIssueStore struct {
	db            *TenantDB
	attachmentDir string
}

func NewSQLIssueStore(db *TenantDB, attachmentDir string) *SQLIssueStore {
	return &SQLIssueStore{db: db, attachmentDir: attachmentDir}
}

func (*SQLIssueStore) issueStorage() {}

func (s *SQLIssueStore) writeTx(tx *sql.Tx, tenant TenantRef, item ResidentIssue) error {
	blob, err := json.Marshal(item)
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		`INSERT INTO issues(tenant_id, tenant_slug, id, data) VALUES($1, $2, $3, $4)
		 ON CONFLICT(tenant_slug, id) DO UPDATE SET data=excluded.data,
		   tenant_id=coalesce(issues.tenant_id, excluded.tenant_id)`,
		tenant.ID, tenant.Slug, item.ID, string(blob),
	)
	return err
}

func loadIssueTx(tx *sql.Tx, tenant TenantRef, id string) (ResidentIssue, bool) {
	var data string
	if err := tx.QueryRow(`SELECT data FROM issues WHERE tenant_id=$1 AND id=$2`, tenant.ID, id).Scan(&data); err != nil {
		return ResidentIssue{}, false
	}
	var item ResidentIssue
	if err := json.Unmarshal([]byte(data), &item); err != nil {
		return ResidentIssue{}, false
	}
	return item, true
}

func (s *SQLIssueStore) create(tenant TenantRef, item ResidentIssue) (ResidentIssue, error) {
	tenantSlug := tenant.Slug
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
	item.TenantSlug = textutil.Slug(tenantSlug)
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
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return ResidentIssue{}, err
	}
	defer tx.Rollback()
	if err := s.writeTx(tx, tenant, item); err != nil {
		return ResidentIssue{}, err
	}
	if err := tx.Commit(); err != nil {
		return ResidentIssue{}, err
	}
	return item, nil
}

func (s *SQLIssueStore) allForTenant(tenant TenantRef) []ResidentIssue {
	rows, err := s.db.For(tenant).Query(`SELECT data FROM issues WHERE tenant_id=$1`, tenant.ID)
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

func (s *SQLIssueStore) listTenant(tenant TenantRef) []ResidentIssue {
	if s == nil {
		return nil
	}
	out := []ResidentIssue{}
	for _, item := range s.allForTenant(tenant) {
		out = append(out, CopyIssue(item))
	}
	SortIssues(out)
	return out
}

func (s *SQLIssueStore) listAuthor(tenant TenantRef, email string) []ResidentIssue {
	if s == nil {
		return nil
	}
	email = textutil.Email(email)
	out := []ResidentIssue{}
	for _, item := range s.allForTenant(tenant) {
		if textutil.Email(item.AuthorEmail) == email {
			out = append(out, CopyIssue(item))
		}
	}
	SortIssues(out)
	return out
}

func (s *SQLIssueStore) get(tenant TenantRef, id string) (ResidentIssue, bool) {
	tenantSlug := tenant.Slug
	if s == nil {
		return ResidentIssue{}, false
	}
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	if tenantSlug == "" || id == "" {
		return ResidentIssue{}, false
	}
	var data string
	if err := s.db.For(tenant).QueryRow(`SELECT data FROM issues WHERE tenant_id=$1 AND id=$2`, tenant.ID, id).Scan(&data); err != nil {
		return ResidentIssue{}, false
	}
	var item ResidentIssue
	if err := json.Unmarshal([]byte(data), &item); err != nil {
		return ResidentIssue{}, false
	}
	return CopyIssue(item), true
}

func (s *SQLIssueStore) updateWorkflow(tenant TenantRef, id string, update IssueWorkflowUpdate) (ResidentIssue, bool, error) {
	tenantSlug := tenant.Slug
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

	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return ResidentIssue{}, false, err
	}
	defer tx.Rollback()
	existing, found := loadIssueTx(tx, tenant, id)
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
	if status != IssueStatusDone {
		updated.ResolutionConfirmedBy = ""
		updated.ResolutionConfirmedAt = time.Time{}
	}
	if update.UpdateResolution {
		if update.ResolutionConfirmed && status == IssueStatusDone {
			updated.ResolutionConfirmedBy = actorEmail
			updated.ResolutionConfirmedAt = changedAt
		} else {
			updated.ResolutionConfirmedBy = ""
			updated.ResolutionConfirmedAt = time.Time{}
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
	if err := s.writeTx(tx, tenant, updated); err != nil {
		return ResidentIssue{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return ResidentIssue{}, false, err
	}
	return CopyIssue(updated), true, nil
}

func (s *SQLIssueStore) addComment(tenant TenantRef, id string, comment IssueComment) (ResidentIssue, bool, error) {
	tenantSlug := tenant.Slug
	if s == nil {
		return ResidentIssue{}, false, nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	comment.Body = strings.TrimSpace(comment.Body)
	comment.AuthorEmail = textutil.Email(comment.AuthorEmail)
	comment.AuthorName = strings.TrimSpace(comment.AuthorName)
	comment.Kind = NormalizeIssueCommentKind(comment.Kind)
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
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return ResidentIssue{}, false, err
	}
	defer tx.Rollback()
	existing, found := loadIssueTx(tx, tenant, id)
	if !found {
		return ResidentIssue{}, false, nil
	}
	updated := existing
	updated.Comments = append(updated.Comments, comment)
	updated.UpdatedAt = comment.CreatedAt
	if err := s.writeTx(tx, tenant, updated); err != nil {
		return ResidentIssue{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return ResidentIssue{}, false, err
	}
	return CopyIssue(updated), true, nil
}

func (s *SQLIssueStore) deleteComment(tenant TenantRef, id string, commentID string, at time.Time) (ResidentIssue, bool, error) {
	tenantSlug := tenant.Slug
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
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return ResidentIssue{}, false, err
	}
	defer tx.Rollback()
	existing, found := loadIssueTx(tx, tenant, id)
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
	if err := s.writeTx(tx, tenant, updated); err != nil {
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
	imports := s.db.Unscoped("boot import replay of the JSON issue snapshot: it spans every tenant and runs before the first request")
	tenants := newTenantIDCache(imports)
	for _, item := range snapshot {
		slug := textutil.Slug(item.TenantSlug)
		if slug == "" || strings.TrimSpace(item.ID) == "" {
			continue
		}
		tenant, err := tenants.ref(slug)
		if err != nil {
			return err
		}
		blob, err := json.Marshal(item)
		if err != nil {
			return err
		}
		if _, err := imports.Exec(
			`INSERT INTO issues(tenant_id, tenant_slug, id, data) VALUES($1, $2, $3, $4) ON CONFLICT(tenant_slug, id) DO NOTHING`,
			tenant.ID, tenant.Slug, item.ID, string(blob),
		); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLIssueStore) clearPhotoPaths(tenant TenantRef, id string) (bool, error) {
	tenantSlug := tenant.Slug
	if s == nil {
		return false, nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	id = strings.TrimSpace(id)
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	existing, found := loadIssueTx(tx, tenant, id)
	if !found || len(existing.PhotoPaths) == 0 {
		return false, nil
	}
	existing.PhotoPaths = nil
	if err := s.writeTx(tx, tenant, existing); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}
