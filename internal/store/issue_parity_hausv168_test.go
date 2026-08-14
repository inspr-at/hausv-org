package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/db"
)

func sampleIssue() ResidentIssue {
	return ResidentIssue{
		TenantSlug:   "demo",
		AuthorEmail:  "resident@example.com",
		AuthorName:   "Resident",
		Category:     "Schaden",
		LocationType: "Wohnung",
		Title:        "Wasserhahn tropft",
		Body:         "Seit gestern tropft der Hahn im Bad.",
	}
}

func TestIssueStorageParity(t *testing.T) {
	backends := map[string]func(t *testing.T) IssueStorage{
		"json": func(t *testing.T) IssueStorage {
			dir := t.TempDir()
			s, err := NewIssueStore(filepath.Join(dir, "issues.json"), filepath.Join(dir, "issue-attachments"))
			if err != nil {
				t.Fatalf("json store: %v", err)
			}
			return s
		},
		"sqlite": func(t *testing.T) IssueStorage {
			dir := t.TempDir()
			database, err := db.Open(filepath.Join(dir, "test.db"))
			if err != nil {
				t.Fatalf("db open: %v", err)
			}
			t.Cleanup(func() { database.Close() })
			return NewSQLIssueStore(database, filepath.Join(dir, "issue-attachments"))
		},
	}

	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	for name, build := range backends {
		t.Run(name, func(t *testing.T) {
			s := build(t)

			// Missing body is rejected.
			bad := sampleIssue()
			bad.Body = ""
			if _, err := s.Create(bad); err == nil {
				t.Fatal("issue without body must error")
			}

			created, err := s.Create(sampleIssue())
			if err != nil {
				t.Fatalf("create: %v", err)
			}
			if created.ID == "" || created.Status != IssueStatusOpen || created.Priority != IssuePriorityNorm {
				t.Fatalf("created bad: %+v", created)
			}
			if created.StatusChangedBy != "resident@example.com" {
				t.Fatalf("StatusChangedBy should default to the author: %+v", created)
			}
			id := created.ID

			if got, ok := s.Get("demo", id); !ok || got.Title != "Wasserhahn tropft" {
				t.Fatalf("get = %+v ok=%v", got, ok)
			}
			if _, ok := s.Get("demo", "nope"); ok {
				t.Fatal("unknown issue must not be found")
			}
			if got := s.ListTenant("demo"); len(got) != 1 {
				t.Fatalf("list = %+v", got)
			}
			if got := s.ListAuthor("demo", "resident@example.com"); len(got) != 1 {
				t.Fatalf("list author = %+v", got)
			}
			if got := s.ListAuthor("demo", "someone@example.com"); len(got) != 0 {
				t.Fatalf("other author must be empty: %+v", got)
			}

			// Workflow: a status change appends history; an unchanged status does not.
			updated, ok, err := s.UpdateWorkflow("demo", id, IssueWorkflowUpdate{
				Status: IssueStatusProgress, Priority: IssuePriorityNorm,
				ActorEmail: "manager@example.com", ActorName: "Manager", ChangedAt: now,
			})
			if err != nil || !ok {
				t.Fatalf("workflow: err=%v ok=%v", err, ok)
			}
			if updated.Status != IssueStatusProgress || len(updated.StatusHistory) != 1 {
				t.Fatalf("status history = %+v", updated.StatusHistory)
			}
			if updated.StatusHistory[0].To != IssueStatusProgress || updated.StatusHistory[0].ActorEmail != "manager@example.com" {
				t.Fatalf("history entry = %+v", updated.StatusHistory[0])
			}
			same, _, err := s.UpdateWorkflow("demo", id, IssueWorkflowUpdate{
				Status: IssueStatusProgress, Priority: IssuePriorityHigh,
				ActorEmail: "manager@example.com", ChangedAt: now.Add(time.Minute),
			})
			if err != nil {
				t.Fatalf("same-status workflow: %v", err)
			}
			if len(same.StatusHistory) != 1 {
				t.Fatalf("unchanged status must not append history: %+v", same.StatusHistory)
			}
			if same.Priority != IssuePriorityHigh {
				t.Fatalf("priority not applied: %q", same.Priority)
			}
			if _, ok, _ := s.UpdateWorkflow("demo", "missing", IssueWorkflowUpdate{
				Status: IssueStatusProgress, Priority: IssuePriorityNorm, ChangedAt: now,
			}); ok {
				t.Fatal("workflow on unknown issue must report not found")
			}
			confirmed, ok, err := s.UpdateWorkflow("demo", id, IssueWorkflowUpdate{
				Status: IssueStatusDone, Priority: IssuePriorityHigh,
				ResolutionConfirmed: true, UpdateResolution: true,
				ActorEmail: "resident@example.com", ChangedAt: now.Add(90 * time.Second),
			})
			if err != nil || !ok || confirmed.ResolutionConfirmedBy != "resident@example.com" || confirmed.ResolutionConfirmedAt.IsZero() {
				t.Fatalf("resolution confirmation: err=%v ok=%v issue=%+v", err, ok, confirmed)
			}
			reopened, ok, err := s.UpdateWorkflow("demo", id, IssueWorkflowUpdate{
				Status: IssueStatusProgress, Priority: IssuePriorityHigh,
				ActorEmail: "manager@example.com", ChangedAt: now.Add(100 * time.Second),
			})
			if err != nil || !ok || !reopened.ResolutionConfirmedAt.IsZero() || reopened.ResolutionConfirmedBy != "" {
				t.Fatalf("reopen must clear resolution confirmation: err=%v ok=%v issue=%+v", err, ok, reopened)
			}

			// Comments.
			withComment, ok, err := s.AddComment("demo", id, IssueComment{
				AuthorEmail: "manager@example.com", AuthorName: "Manager",
				Body: "Wir schauen uns das an.", Kind: IssueCommentKindQuestion, CreatedAt: now.Add(2 * time.Minute),
			})
			if err != nil || !ok || len(withComment.Comments) != 1 {
				t.Fatalf("add comment: err=%v ok=%v comments=%+v", err, ok, withComment.Comments)
			}
			commentID := withComment.Comments[0].ID
			if commentID == "" {
				t.Fatal("comment id must be generated")
			}
			if withComment.Comments[0].Kind != IssueCommentKindQuestion {
				t.Fatalf("comment kind not persisted: %+v", withComment.Comments[0])
			}
			// A photo-only (empty body) comment is allowed.
			if _, _, err := s.AddComment("demo", id, IssueComment{
				AuthorEmail: "resident@example.com", CreatedAt: now.Add(3 * time.Minute),
			}); err != nil {
				t.Fatalf("photo-only comment must be allowed: %v", err)
			}
			// A comment without an author is not.
			if _, _, err := s.AddComment("demo", id, IssueComment{Body: "x"}); err == nil {
				t.Fatal("comment without author must error")
			}

			deleted, ok, err := s.DeleteComment("demo", id, commentID, now.Add(4*time.Minute))
			if err != nil || !ok || len(deleted.Comments) != 1 {
				t.Fatalf("delete comment: err=%v ok=%v comments=%+v", err, ok, deleted.Comments)
			}
			if _, ok, err := s.DeleteComment("demo", id, commentID, now); ok || err != nil {
				t.Fatalf("deleting the same comment twice: ok=%v err=%v", ok, err)
			}
		})
	}
}

func TestSQLIssueImportFromJSON(t *testing.T) {
	dir := t.TempDir()
	jsonStore, err := NewIssueStore(filepath.Join(dir, "issues.json"), filepath.Join(dir, "issue-attachments"))
	if err != nil {
		t.Fatalf("json store: %v", err)
	}
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	created, err := jsonStore.Create(sampleIssue())
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, _, err := jsonStore.UpdateWorkflow("demo", created.ID, IssueWorkflowUpdate{
		Status: IssueStatusProgress, Priority: IssuePriorityNorm,
		ActorEmail: "manager@example.com", ChangedAt: now,
	}); err != nil {
		t.Fatalf("seed workflow: %v", err)
	}
	if _, _, err := jsonStore.AddComment("demo", created.ID, IssueComment{
		AuthorEmail: "manager@example.com", Body: "Notiz", Kind: IssueCommentKindInformation, CreatedAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("seed comment: %v", err)
	}

	database, err := db.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("db open: %v", err)
	}
	defer database.Close()
	sqlStore := NewSQLIssueStore(database, filepath.Join(dir, "issue-attachments"))

	for i := 0; i < 2; i++ {
		if err := sqlStore.ImportIssues(jsonStore); err != nil {
			t.Fatalf("import %d: %v", i, err)
		}
	}
	if got := sqlStore.ListTenant("demo"); len(got) != 1 {
		t.Fatalf("imported %d issues, want 1", len(got))
	}
	// Comments and status history survive the import.
	got, ok := sqlStore.Get("demo", created.ID)
	if !ok {
		t.Fatal("imported issue not found")
	}
	if len(got.Comments) != 1 || got.Comments[0].Body != "Notiz" || got.Comments[0].Kind != IssueCommentKindInformation {
		t.Fatalf("comments lost in import: %+v", got.Comments)
	}
	if len(got.StatusHistory) != 1 || got.Status != IssueStatusProgress {
		t.Fatalf("status history lost in import: %+v", got.StatusHistory)
	}
}
