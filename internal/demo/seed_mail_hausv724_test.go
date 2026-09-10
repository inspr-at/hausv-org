package demo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/dbtest"
	"github.com/inspr-at/hausv-org/internal/mailintake"
	"github.com/inspr-at/hausv-org/internal/store"
)

func TestDemoResetClearsMailboxLedgerAndRemovesOrphanedMailIssues(t *testing.T) {
	database, config := dbtest.OpenWithConfig(t)
	scoped, err := db.NewScoped(config, database)
	if err != nil {
		t.Fatal(err)
	}
	defer scoped.Close()
	const seedDir = "../../scripts/demo/seed"
	documents := t.TempDir()
	if _, err := Load(t.Context(), database, seedDir, SeedOptions{DocumentDir: documents}); err != nil {
		t.Fatal(err)
	}
	identities, err := store.EnsureTenantIdentities(t.Context(), database, []store.TenantIdentity{{Slug: "janusbergweg-123"}})
	if err != nil {
		t.Fatal(err)
	}
	issues, ok := store.BindIssueRepository(store.NewSQLIssueStore(store.NewTenantDB(scoped), t.TempDir()), identities["janusbergweg-123"].Ref())
	if !ok {
		t.Fatal("issue repository unavailable")
	}
	// Exactly the residues of two intake passes separated by an old reset:
	// generated issue IDs remain, while their intake items have disappeared.
	for i := range 2 {
		_, err := issues.Create(store.ResidentIssue{TenantSlug: "janusbergweg-123", Source: "email", IntakeID: fmt.Sprintf("import-%d", i), AuthorEmail: "alina.eigentuemer@musterstadt.example", AuthorName: "Alina Auer", Title: "Beleg Fensterreinigung Stiegenhaus", Body: "Beleg", Category: "Frage", Status: store.IssueStatusNew, LocationType: store.IssueLocationCommon})
		if err != nil {
			t.Fatal(err)
		}
	}
	// Similar user-created content must survive the narrowly scoped cleanup.
	unrelated, err := issues.Create(store.ResidentIssue{TenantSlug: "janusbergweg-123", AuthorEmail: "alina.eigentuemer@musterstadt.example", Title: "Beleg Fensterreinigung Stiegenhaus", Body: "Portal", Category: "Frage", Status: store.IssueStatusNew, LocationType: store.IssueLocationCommon})
	if err != nil {
		t.Fatal(err)
	}
	// A linked mail issue is still reset after a manager changes its title.
	intake := store.BindIntakeRepository(database, "musterstadt")
	if err := intake.Create(t.Context(), store.IntakeItem{ID: "linked-mail", Source: store.IntakeSourceEmail, Status: store.IntakeStatusApproved, ReceivedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	linked, err := issues.Create(store.ResidentIssue{TenantSlug: "janusbergweg-123", Source: "email", IntakeID: "linked-mail", AuthorEmail: "alina.eigentuemer@musterstadt.example", Title: "Überarbeiteter Betreff", Body: "Beleg", Category: "Frage", LocationType: store.IssueLocationCommon})
	if err != nil {
		t.Fatal(err)
	}
	otherLedger := store.BindIntakeMailSeenRepository(database, "other")
	if err := otherLedger.Record(t.Context(), "previously-imported@example.com", "other-intake"); err != nil {
		t.Fatal(err)
	}
	ledger := store.BindIntakeMailSeenRepository(database, "musterstadt")
	if err := ledger.Record(t.Context(), "previously-imported@example.com", "gone"); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if _, err := Load(t.Context(), database, seedDir, SeedOptions{Reset: true, DocumentDir: documents}); err != nil {
			t.Fatal(err)
		}
		for _, issue := range issues.List() {
			if issue.Source == "email" && issue.Title == "Beleg Fensterreinigung Stiegenhaus" {
				t.Errorf("reset kept an orphaned mailbox issue: %s", issue.ID)
			}
		}
		if _, found := issues.Get(unrelated.ID); !found {
			t.Error("reset deleted unrelated portal content")
		}
		if _, found := issues.Get(linked.ID); found {
			t.Error("reset kept a mail issue with an edited title")
		}
		if _, err := intake.Get(t.Context(), "linked-mail"); err != store.ErrIntakeNotFound {
			t.Errorf("reset kept the linked intake: %v", err)
		}
		if count, err := ledger.Count(t.Context()); err != nil || count != 0 {
			t.Errorf("reset must empty its organisation's ledger: count=%d err=%v", count, err)
		}
		if seen, err := otherLedger.Seen(t.Context(), "previously-imported@example.com"); err != nil || !seen {
			t.Errorf("reset changed another organisation's ledger: seen=%v err=%v", seen, err)
		}
		files, err := filepath.Glob(filepath.Join(seedDir, "mail", "*.eml"))
		if err != nil || len(files) != 5 {
			t.Fatalf("mail fixtures: count=%d err=%v", len(files), err)
		}
		for _, file := range files {
			raw, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			message, err := mailintake.Parse(raw, mailintake.Limits{})
			if err != nil || message.MessageID == "" {
				t.Fatalf("mail fixture lacks stable Message-ID: %s err=%v", file, err)
			}
			if seen, err := ledger.Seen(t.Context(), message.DedupeKey()); err != nil || seen {
				t.Errorf("reset must allow fixture %s: seen=%v err=%v", file, seen, err)
			}
			// Simulate a successful import between the two resets.
			if err := ledger.Record(t.Context(), message.DedupeKey(), "imported"); err != nil {
				t.Fatal(err)
			}
			if seen, err := store.BindIntakeMailSeenRepository(database, "other").Seen(t.Context(), message.MessageID); err != nil || seen {
				t.Errorf("fixture leaked into another organisation: seen=%v err=%v", seen, err)
			}
		}
	}
	// The seed's existing auto_done receipts still retain their status.
	var fixtures []seedIntake
	raw, err := os.ReadFile(filepath.Join(seedDir, "intake.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &fixtures); err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures {
		if fixture.StatusHint == "auto_done" && fixture.Precomputed != nil && fixture.Precomputed.Category == "beleg" {
			item, err := store.BindIntakeRepository(database, "musterstadt").Get(t.Context(), fixture.ID)
			if err != nil || item.Status != store.IntakeStatusAuto {
				t.Errorf("seed receipt changed status: %s status=%s err=%v", fixture.ID, item.Status, err)
			}
		}
	}
}
