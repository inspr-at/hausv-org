package store

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/dbtest"
	"github.com/inspr-at/hausv-org/internal/textutil"
)

func TestIntakeQueryContract(t *testing.T) {
	database := dbtest.Open(t)
	repo := BindIntakeRepository(database, "org-a").(*sqlIntakeRepository)
	other := BindIntakeRepository(database, "org-b")
	base := time.Date(2026, 9, 3, 8, 0, 0, 0, time.UTC)
	items := []IntakeItem{
		{ID: "u-open", Source: IntakeSourceEmail, Status: IntakeStatusOpen, Subject: "U", Body: "U", ReceivedAt: base},
		{ID: "a-vera", TenantSlug: "house-a", Source: IntakeSourceEmail, Status: IntakeStatusProposed, Subject: "A", Body: "A", ReceivedAt: base.Add(time.Hour), DueAt: base.Add(30 * time.Hour), Suggestion: &IntakeSuggestion{Assignee: "vera"}},
		{ID: "a-other", TenantSlug: "house-a", Source: IntakeSourcePhone, Status: IntakeStatusOpen, Subject: "A2", Body: "A2", ReceivedAt: base.Add(2 * time.Hour), Suggestion: &IntakeSuggestion{Assignee: "other"}},
		{ID: "b-vera", TenantSlug: "house-b", Source: IntakeSourcePortal, Status: IntakeStatusApproved, Subject: "B", Body: "B", ReceivedAt: base.Add(3 * time.Hour), DueAt: base.Add(10 * time.Hour), Suggestion: &IntakeSuggestion{Assignee: "vera"}},
		{ID: "b-none", TenantSlug: "house-b", Source: IntakeSourceEmail, Status: IntakeStatusRejected, Subject: "B2", Body: "B2", ReceivedAt: base.Add(4 * time.Hour)},
		{ID: "c-vera", TenantSlug: "house-c", Source: IntakeSourceEmail, Status: IntakeStatusOpen, Subject: "C", Body: "C", ReceivedAt: base.Add(5 * time.Hour), Suggestion: &IntakeSuggestion{Assignee: "vera"}},
		{ID: "u-vera", Source: IntakeSourcePhone, Status: IntakeStatusProposed, Subject: "UV", Body: "UV", ReceivedAt: base.Add(6 * time.Hour), Suggestion: &IntakeSuggestion{Assignee: "vera"}},
		{ID: "a-amp", TenantSlug: "House_A", Source: IntakeSourceEmail, Status: IntakeStatusOpen, Subject: "Amp", Body: "Amp", ReceivedAt: base.Add(7 * time.Hour), Suggestion: &IntakeSuggestion{Assignee: "a&b"}},
	}
	insertIntakeItems(t, repo, items)
	if err := other.Create(t.Context(), IntakeItem{ID: "foreign", Source: IntakeSourceEmail, Status: IntakeStatusOpen, Subject: "other org", Body: "x", ReceivedAt: base}); err != nil {
		t.Fatal(err)
	}

	filters := []IntakeFilter{
		{},
		{Statuses: []IntakeStatus{IntakeStatusProposed}},
		{Statuses: []IntakeStatus{IntakeStatusOpen, IntakeStatusProposed}},
		{Statuses: []IntakeStatus{"not-a-status"}},
		{Sources: []IntakeSource{IntakeSourceEmail}},
		{Sources: []IntakeSource{IntakeSourcePhone, IntakeSourcePortal}},
		{TenantSlug: "house-a"},
		{TenantSlug: "House_A"},
		{TenantSlugs: []string{"house-a", "house-b"}},
		{TenantSlugs: []string{"house-a", "house-b"}, IncludeUnassigned: true},
		{Unassigned: true},
		{IncludeUnassigned: true},
		{Assignee: "vera"},
		{Assignee: "  vera  "},
		{Assignee: "a&b"},
		{Assignee: "nobody"},
		{Since: base.Add(3 * time.Hour)},
		{TenantSlugs: []string{"house-a", "house-b"}, IncludeUnassigned: true, Assignee: "vera", Statuses: []IntakeStatus{IntakeStatusOpen, IntakeStatusProposed}},
		{TenantSlugs: []string{"\x00"}},
		{TenantSlug: "\x00"},
		{TenantSlugs: []string{"house-a", "\x00"}},
		{TenantSlugs: []string{"\x00"}, IncludeUnassigned: true},
		{TenantSlugs: []string{"house-a", "\x00"}, IncludeUnassigned: true},
		{TenantSlugs: []string{"", "  "}},
		{TenantSlugs: []string{"", "  "}, IncludeUnassigned: true},
		{Sort: "due"},
		{Limit: 2, Offset: 1},
		{Offset: 2},
		{TenantSlugs: []string{"house-a", "house-b"}, IncludeUnassigned: true, Assignee: "vera", Sort: "due", Limit: 1, Offset: 1},
	}
	for i, filter := range filters {
		t.Run(fmt.Sprintf("%02d-%s", i, filterLabel(filter)), func(t *testing.T) {
			assertCountMatchesList(t, repo, filter)
		})
	}

	scoped := IntakeFilter{TenantSlugs: []string{"house-a", "house-b"}, IncludeUnassigned: true}
	assertPagesCover(t, repo, scoped, 3)
	assertPagesCover(t, repo, IntakeFilter{Assignee: "vera"}, 2)
	assertPagesCover(t, repo, IntakeFilter{Sort: "due"}, 3)

	page, err := repo.List(t.Context(), IntakeFilter{TenantSlugs: []string{"house-a", "house-b"}, IncludeUnassigned: true, Assignee: "vera", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 1 || page[0].ID != "u-vera" {
		t.Fatalf("first vera page = %#v, want u-vera", page)
	}

	all, err := repo.List(t.Context(), IntakeFilter{})
	if err != nil {
		t.Fatal(err)
	}
	offsetOnly, err := repo.List(t.Context(), IntakeFilter{Offset: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(offsetOnly) != len(all)-2 {
		t.Fatalf("offset-only returned %d rows, want %d", len(offsetOnly), len(all)-2)
	}
	for i := range offsetOnly {
		if offsetOnly[i].ID != all[i+2].ID {
			t.Fatalf("offset-only[%d] = %s, want %s", i, offsetOnly[i].ID, all[i+2].ID)
		}
	}

	foreign, err := repo.List(t.Context(), IntakeFilter{})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range foreign {
		if item.ID == "foreign" || item.Organisation != "org-a" {
			t.Fatalf("organisation scope leaked: %#v", item)
		}
	}
	assertIDs(t, repo, IntakeFilter{TenantSlugs: []string{"\x00"}}, nil)
	assertIDs(t, repo, IntakeFilter{TenantSlug: "\x00"}, nil)
	assertIDs(t, repo, IntakeFilter{TenantSlugs: []string{"house-a", "\x00"}}, []string{"a-amp", "a-other", "a-vera"})
	assertIDs(t, repo, IntakeFilter{TenantSlugs: []string{"\x00"}, IncludeUnassigned: true}, []string{"u-vera", "u-open"})
	assertIDs(t, repo, IntakeFilter{TenantSlugs: []string{"house-a", "\x00"}, IncludeUnassigned: true}, []string{"a-amp", "u-vera", "a-other", "a-vera", "u-open"})
}

func assertIDs(t *testing.T, repo *sqlIntakeRepository, filter IntakeFilter, want []string) {
	t.Helper()
	list, err := repo.List(t.Context(), filter)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, len(list))
	for i, item := range list {
		got[i] = item.ID
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("ids = %v, want %v", got, want)
	}
	count, err := repo.Count(t.Context(), filter)
	if err != nil {
		t.Fatal(err)
	}
	if count != len(want) {
		t.Fatalf("count = %d, want %d", count, len(want))
	}
}

func TestIntakeQueryContractBoundsRows(t *testing.T) {
	database := dbtest.Open(t)
	repo := BindIntakeRepository(database, "org-large").(*sqlIntakeRepository)
	const n = 5000
	const pageSize = 25
	base := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	items := make([]IntakeItem, n)
	for i := range items {
		item := IntakeItem{
			ID:         fmt.Sprintf("m-%04d", i),
			TenantSlug: "house-a",
			Source:     IntakeSourceEmail,
			Status:     IntakeStatusOpen,
			Subject:    "m",
			Body:       "m",
			ReceivedAt: base.Add(time.Duration(i) * time.Second),
		}
		if i%10 == 0 {
			item.Suggestion = &IntakeSuggestion{Assignee: "vera"}
		}
		items[i] = item
	}
	insertIntakeItems(t, repo, items)

	var countRows, listRows int
	repo.observe = func(op string, rows int) {
		switch op {
		case "count":
			countRows = rows
		case "list":
			listRows = rows
		}
	}
	count, err := repo.Count(t.Context(), IntakeFilter{Assignee: "vera"})
	if err != nil {
		t.Fatal(err)
	}
	if count != n/10 || countRows != 1 {
		t.Fatalf("count = %d from %d rows; want %d from 1 row", count, countRows, n/10)
	}
	page, err := repo.List(t.Context(), IntakeFilter{Assignee: "vera", Limit: pageSize, Offset: pageSize})
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != pageSize || listRows != pageSize {
		t.Fatalf("page returned %d items from %d rows; want %d", len(page), listRows, pageSize)
	}
	if page[0].ID != "m-4740" {
		t.Fatalf("page start = %s, want m-4740", page[0].ID)
	}

	where, args := intakeWhere(repo.orgKey, IntakeFilter{Assignee: "vera"}, repo.postgres)
	plan := explainIntake(t, repo, `SELECT COUNT(*) FROM intake_items WHERE `+where, args...)
	t.Logf("count plan (%s):\n%s", map[bool]string{true: "postgres", false: "sqlite"}[repo.postgres], plan)
}

func assertCountMatchesList(t *testing.T, repo *sqlIntakeRepository, filter IntakeFilter) {
	t.Helper()
	unlimited := filter
	unlimited.Limit = 0
	unlimited.Offset = 0
	count, err := repo.Count(t.Context(), filter)
	if err != nil {
		t.Fatal(err)
	}
	list, err := repo.List(t.Context(), unlimited)
	if err != nil {
		t.Fatal(err)
	}
	if count != len(list) {
		t.Fatalf("count %d != len(list) %d", count, len(list))
	}
	if filter.Limit > 0 || filter.Offset > 0 {
		page, err := repo.List(t.Context(), filter)
		if err != nil {
			t.Fatal(err)
		}
		start := filter.Offset
		if start > len(list) {
			start = len(list)
		}
		end := len(list)
		if filter.Limit > 0 && start+filter.Limit < end {
			end = start + filter.Limit
		}
		if len(page) != end-start {
			t.Fatalf("page len %d, want %d", len(page), end-start)
		}
		for i := range page {
			if page[i].ID != list[start+i].ID {
				t.Fatalf("page[%d] = %s, want %s", i, page[i].ID, list[start+i].ID)
			}
		}
	}
}

func assertPagesCover(t *testing.T, repo *sqlIntakeRepository, filter IntakeFilter, pageSize int) {
	t.Helper()
	filter.Limit = 0
	filter.Offset = 0
	all, err := repo.List(t.Context(), filter)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for offset := 0; ; offset += pageSize {
		pageFilter := filter
		pageFilter.Limit = pageSize
		pageFilter.Offset = offset
		page, err := repo.List(t.Context(), pageFilter)
		if err != nil {
			t.Fatal(err)
		}
		if len(page) > pageSize {
			t.Fatalf("page at %d has %d rows, limit %d", offset, len(page), pageSize)
		}
		for _, item := range page {
			if seen[item.ID] {
				t.Fatalf("page overlap on %s", item.ID)
			}
			seen[item.ID] = true
		}
		if len(page) < pageSize {
			break
		}
	}
	if len(seen) != len(all) {
		t.Fatalf("pages cover %d of %d", len(seen), len(all))
	}
}

func insertIntakeItems(t *testing.T, repo *sqlIntakeRepository, items []IntakeItem) {
	t.Helper()
	tx, err := repo.begin(t.Context(), repo.orgKey)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	for _, item := range items {
		item.Organisation = repo.orgKey
		item.TenantSlug = textutil.Slug(item.TenantSlug)
		if item.CreatedAt.IsZero() {
			item.CreatedAt = item.ReceivedAt
		}
		item.UpdatedAt = item.CreatedAt
		blob, err := json.Marshal(item)
		if err != nil {
			t.Fatal(err)
		}
		_, err = tx.ExecContext(t.Context(), `INSERT INTO intake_items(org_key,id,status,source,tenant_slug,received_at,due_at,data)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8)`,
			repo.orgKey, item.ID, item.Status, item.Source, item.TenantSlug, item.ReceivedAt.UTC().Format(time.RFC3339Nano), formatOptionalTime(item.DueAt), string(blob))
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func explainIntake(t *testing.T, repo *sqlIntakeRepository, query string, args ...any) string {
	t.Helper()
	tx, err := repo.begin(t.Context(), repo.orgKey)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	prefix := "EXPLAIN QUERY PLAN "
	if repo.postgres {
		prefix = "EXPLAIN "
	}
	rows, err := tx.QueryContext(t.Context(), prefix+query, args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for rows.Next() {
		raw := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range raw {
			ptrs[i] = &raw[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			t.Fatal(err)
		}
		parts := make([]string, len(cols))
		for i, value := range raw {
			parts[i] = fmt.Sprint(value)
		}
		lines = append(lines, strings.Join(parts, " | "))
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return strings.Join(lines, "\n")
}

func filterLabel(filter IntakeFilter) string {
	var parts []string
	if len(filter.Statuses) > 0 {
		parts = append(parts, "status")
	}
	if len(filter.Sources) > 0 {
		parts = append(parts, "source")
	}
	if filter.TenantSlug != "" {
		parts = append(parts, "slug")
	}
	if len(filter.TenantSlugs) > 0 {
		parts = append(parts, "slugs")
	}
	if filter.Unassigned {
		parts = append(parts, "unassigned")
	}
	if filter.IncludeUnassigned {
		parts = append(parts, "include-unassigned")
	}
	if filter.Assignee != "" {
		parts = append(parts, "assignee")
	}
	if !filter.Since.IsZero() {
		parts = append(parts, "since")
	}
	if filter.Sort != "" {
		parts = append(parts, "sort-"+filter.Sort)
	}
	if filter.Limit > 0 || filter.Offset > 0 {
		parts = append(parts, "page")
	}
	if len(parts) == 0 {
		return "all"
	}
	return strings.Join(parts, "+")
}
