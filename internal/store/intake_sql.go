package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

var ErrIntakeNotFound = errors.New("intake item not found")

type sqlIntakeRepository struct {
	begin  func(context.Context, string) (*sql.Tx, error)
	orgKey string
}

func beginOrgTx(ctx context.Context, database *sql.DB, orgKey string) (*sql.Tx, error) {
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	if strings.Contains(fmt.Sprintf("%T", database.Driver()), "stdlib") {
		if _, err := tx.ExecContext(ctx, `SELECT set_config('app.org_key',$1,true)`, orgKey); err != nil {
			tx.Rollback()
			return nil, err
		}
	}
	return tx, nil
}

func BindIntakeRepository(database *sql.DB, orgKey string) IntakeRepository {
	return &sqlIntakeRepository{
		begin:  func(ctx context.Context, orgKey string) (*sql.Tx, error) { return beginOrgTx(ctx, database, orgKey) },
		orgKey: textutil.Slug(orgKey),
	}
}

func validIntakeSource(value IntakeSource) bool {
	return value == IntakeSourcePortal || value == IntakeSourceEmail || value == IntakeSourcePhone
}

func validIntakeStatus(value IntakeStatus) bool {
	switch value {
	case IntakeStatusOpen, IntakeStatusProposed, IntakeStatusApproved, IntakeStatusEdited, IntakeStatusRejected, IntakeStatusAuto:
		return true
	default:
		return false
	}
}

func (r *sqlIntakeRepository) Create(ctx context.Context, item IntakeItem) error {
	if r.begin == nil || r.orgKey == "" {
		return fmt.Errorf("intake repository requires database and organisation")
	}
	item.ID = strings.TrimSpace(item.ID)
	item.Organisation = r.orgKey
	item.TenantSlug = textutil.Slug(item.TenantSlug)
	if item.ID == "" || !validIntakeSource(item.Source) || !validIntakeStatus(item.Status) || item.ReceivedAt.IsZero() {
		return fmt.Errorf("invalid intake item")
	}
	now := time.Now().UTC()
	item.ReceivedAt = item.ReceivedAt.UTC()
	if !item.DueAt.IsZero() {
		item.DueAt = item.DueAt.UTC()
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
	blob, err := json.Marshal(item)
	if err != nil {
		return err
	}
	tx, err := r.begin(ctx, r.orgKey)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO intake_items(org_key,id,status,source,tenant_slug,received_at,due_at,data)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT(org_key,id) DO UPDATE SET status=excluded.status,source=excluded.source,
		`+"tenant_"+`slug=excluded.tenant_slug,received_at=excluded.received_at,due_at=excluded.due_at,data=excluded.data`,
		r.orgKey, item.ID, item.Status, item.Source, item.TenantSlug, item.ReceivedAt.Format(time.RFC3339Nano), formatOptionalTime(item.DueAt), string(blob))
	if err != nil {
		return err
	}
	return tx.Commit()
}

func formatOptionalTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func (r *sqlIntakeRepository) Get(ctx context.Context, id string) (IntakeItem, error) {
	var data string
	if r.begin == nil || r.orgKey == "" {
		return IntakeItem{}, ErrIntakeNotFound
	}
	tx, err := r.begin(ctx, r.orgKey)
	if err != nil {
		return IntakeItem{}, err
	}
	defer tx.Rollback()
	err = tx.QueryRowContext(ctx, `SELECT data FROM intake_items WHERE org_key=$1 AND id=$2`, r.orgKey, strings.TrimSpace(id)).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return IntakeItem{}, ErrIntakeNotFound
	}
	if err != nil {
		return IntakeItem{}, err
	}
	var item IntakeItem
	if err := json.Unmarshal([]byte(data), &item); err != nil {
		return IntakeItem{}, err
	}
	if err := tx.Commit(); err != nil {
		return IntakeItem{}, err
	}
	return item, nil
}

func intakeWhere(orgKey string, filter IntakeFilter) (string, []any) {
	parts := []string{"org_key=$1"}
	args := []any{orgKey}
	addList := func(column string, values []string) {
		if len(values) == 0 {
			return
		}
		marks := make([]string, 0, len(values))
		for _, value := range values {
			args = append(args, value)
			marks = append(marks, fmt.Sprintf("$%d", len(args)))
		}
		parts = append(parts, column+" IN ("+strings.Join(marks, ",")+")")
	}
	statuses := make([]string, 0, len(filter.Statuses))
	for _, value := range filter.Statuses {
		if validIntakeStatus(value) {
			statuses = append(statuses, string(value))
		}
	}
	sources := make([]string, 0, len(filter.Sources))
	for _, value := range filter.Sources {
		if validIntakeSource(value) {
			sources = append(sources, string(value))
		}
	}
	addList("status", statuses)
	addList("source", sources)
	if filter.Unassigned {
		parts = append(parts, "tenant_"+"slug=''")
	} else if slug := textutil.Slug(filter.TenantSlug); slug != "" {
		args = append(args, slug)
		parts = append(parts, fmt.Sprintf("tenant_"+"slug=$%d", len(args)))
	} else if len(filter.TenantSlugs) > 0 {
		slugs := make([]string, 0, len(filter.TenantSlugs))
		for _, value := range filter.TenantSlugs {
			if slug := textutil.Slug(value); slug != "" {
				slugs = append(slugs, slug)
			}
		}
		if len(slugs) > 0 {
			marks := make([]string, 0, len(slugs))
			for _, slug := range slugs {
				args = append(args, slug)
				marks = append(marks, fmt.Sprintf("$%d", len(args)))
			}
			clause := "tenant_" + "slug IN (" + strings.Join(marks, ",") + ")"
			if filter.IncludeUnassigned {
				clause = "(" + clause + " OR tenant_" + "slug='')"
			}
			parts = append(parts, clause)
		} else if filter.IncludeUnassigned {
			parts = append(parts, "tenant_"+"slug=''")
		}
	} else if filter.IncludeUnassigned {
		parts = append(parts, "tenant_"+"slug=''")
	}
	if !filter.Since.IsZero() {
		args = append(args, filter.Since.UTC().Format(time.RFC3339Nano))
		parts = append(parts, fmt.Sprintf("received_at >= $%d", len(args)))
	}
	return strings.Join(parts, " AND "), args
}

func (r *sqlIntakeRepository) List(ctx context.Context, filter IntakeFilter) ([]IntakeItem, error) {
	where, args := intakeWhere(r.orgKey, filter)
	query := `SELECT data FROM intake_items WHERE ` + where
	if filter.Sort == "due" {
		query += ` ORDER BY CASE WHEN due_at='' THEN 1 ELSE 0 END, due_at ASC, received_at DESC`
	} else {
		query += ` ORDER BY received_at DESC`
	}
	if filter.Assignee == "" && filter.Limit > 0 {
		args = append(args, filter.Limit)
		query += fmt.Sprintf(" LIMIT $%d", len(args))
		if filter.Offset > 0 {
			args = append(args, filter.Offset)
			query += fmt.Sprintf(" OFFSET $%d", len(args))
		}
	} else if filter.Assignee == "" && filter.Offset > 0 {
		args = append(args, filter.Offset)
		query += fmt.Sprintf(" LIMIT -1 OFFSET $%d", len(args))
	}
	if r.begin == nil || r.orgKey == "" {
		return nil, fmt.Errorf("intake repository is not bound")
	}
	tx, err := r.begin(ctx, r.orgKey)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []IntakeItem{}
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var item IntakeItem
		if err := json.Unmarshal([]byte(data), &item); err != nil {
			return nil, err
		}
		if filter.Assignee != "" && (item.Suggestion == nil || item.Suggestion.Assignee != strings.TrimSpace(filter.Assignee)) {
			continue
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	if filter.Assignee != "" {
		start := min(filter.Offset, len(items))
		end := len(items)
		if filter.Limit > 0 {
			end = min(start+filter.Limit, end)
		}
		items = items[start:end]
	}
	return items, nil
}

func (r *sqlIntakeRepository) Count(ctx context.Context, filter IntakeFilter) (int, error) {
	items, err := r.List(ctx, IntakeFilter{Statuses: filter.Statuses, Sources: filter.Sources, TenantSlug: filter.TenantSlug, TenantSlugs: filter.TenantSlugs, Unassigned: filter.Unassigned, IncludeUnassigned: filter.IncludeUnassigned, Assignee: filter.Assignee, Since: filter.Since})
	return len(items), err
}

func (r *sqlIntakeRepository) UpdateSuggestion(ctx context.Context, id string, suggestion IntakeSuggestion, status IntakeStatus) error {
	if !validIntakeStatus(status) || status == IntakeStatusOpen {
		return fmt.Errorf("invalid suggestion status")
	}
	item, err := r.Get(ctx, id)
	if err != nil {
		return err
	}
	if suggestion.CreatedAt.IsZero() {
		suggestion.CreatedAt = time.Now().UTC()
	}
	item.Suggestion = &suggestion
	item.Status = status
	item.UpdatedAt = time.Now().UTC()
	return r.Create(ctx, item)
}

func (r *sqlIntakeRepository) UpdateHandling(ctx context.Context, id string, handling IntakeHandling, status IntakeStatus, issueID string) error {
	if !validIntakeStatus(status) || status == IntakeStatusOpen {
		return fmt.Errorf("invalid handling status")
	}
	item, err := r.Get(ctx, id)
	if err != nil {
		return err
	}
	if handling.At.IsZero() {
		handling.At = time.Now().UTC()
	}
	item.Handling = &handling
	item.Status = status
	item.IssueID = strings.TrimSpace(issueID)
	item.UpdatedAt = time.Now().UTC()
	return r.Create(ctx, item)
}

func (r *sqlIntakeRepository) Assign(ctx context.Context, id, tenantSlug, unit string) error {
	item, err := r.Get(ctx, id)
	if err != nil {
		return err
	}
	item.TenantSlug = textutil.Slug(tenantSlug)
	item.Unit = strings.TrimSpace(unit)
	item.UpdatedAt = time.Now().UTC()
	return r.Create(ctx, item)
}
