package server

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/dbtest"
	"github.com/inspr-at/hausv-org/internal/demo"
	"github.com/inspr-at/hausv-org/internal/store"
)

func assertClosedIntakeIssue724(t *testing.T, issue store.ResidentIssue) {
	t.Helper()
	if issue.Status != store.IssueStatusDone {
		t.Fatalf("issue %s status=%s, want Erledigt", issue.ID, issue.Status)
	}
	if issueOpenCount([]residentIssue{issue}) != 0 || portfolioIssueIsOpen(issue) {
		t.Errorf("completed issue %s counted as open on board/portfolio", issue.ID)
	}
	if got := newestOpenIssueSummaries([]residentIssue{issue}, time.Now()); len(got) != 0 {
		t.Errorf("completed issue %s appears in the short list", issue.ID)
	}
}

func TestIntakeHandlingIssueStatusAndReply(t *testing.T) {
	for _, backend := range []string{"json", "sql"} {
		for _, action := range []string{"auto", "approve", "edit"} {
			for _, existing := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/existing=%v", backend, action, existing), func(t *testing.T) {
					a, intake, _ := newInboxTestApp(t, roleManager)
					if backend == "sql" {
						database, config := dbtest.OpenWithConfig(t)
						scoped, err := db.NewScoped(config, database)
						if err != nil {
							t.Fatal(err)
						}
						defer scoped.Close()
						identities, err := store.EnsureTenantIdentities(t.Context(), database, []store.TenantIdentity{{Slug: "demo"}})
						if err != nil {
							t.Fatal(err)
						}
						a.tenantIdentities["demo"] = identities["demo"]
						a.issueStore = store.NewSQLIssueStore(store.NewTenantDB(scoped), t.TempDir())
					}
					mailer := &intakeReplyMailer{}
					a.mailer = mailer
					options := intakeHandleOptions{status: store.IntakeStatusApproved, action: action, actorEmail: "vera@example.com", actorName: "Vera"}
					wantStatus := store.IssueStatusNew
					if action == "auto" {
						options.status, options.actorEmail, options.actorName = store.IntakeStatusAuto, "", "System (KI)"
						wantStatus = store.IssueStatusDone
					} else if action == "edit" {
						options.status = store.IntakeStatusEdited
					}
					item := store.IntakeItem{ID: "receipt-724", TenantSlug: "demo", Source: store.IntakeSourceEmail, FromName: "Alina", FromEmail: "alina@example.com", Subject: "Beleg Fensterreinigung Stiegenhaus", Body: "Anbei der Beleg.", Status: store.IntakeStatusProposed, ReceivedAt: time.Now(), Suggestion: &store.IntakeSuggestion{TenantSlug: "demo", Category: "beleg", Priority: store.IssuePriorityLow, Reply: "Vielen Dank für den Beleg."}}
					issues := a.repositoriesForTenant(a.tenantIdentities["demo"].Ref()).issues
					if existing {
						oldStatus := store.IssueStatusNew
						if action != "auto" {
							oldStatus = store.IssueStatusDone
						}
						_, err := issues.Create(store.ResidentIssue{TenantSlug: "demo", IntakeID: item.ID, Source: "email", AuthorEmail: item.FromEmail, Title: item.Subject, Body: item.Body, Category: "Frage", LocationType: store.IssueLocationUnit, Status: oldStatus})
						if err != nil {
							t.Fatal(err)
						}
					}
					if err := intake.Create(t.Context(), item); err != nil {
						t.Fatal(err)
					}
					for range 2 {
						if err := a.handleIntake(t.Context(), "musterstadt", item, options); err != nil {
							t.Fatal(err)
						}
						got, err := intake.Get(t.Context(), item.ID)
						if err != nil || got.Status != options.status || got.IssueID == "" || got.Handling == nil || got.Handling.ByName != options.actorName {
							t.Fatalf("intake handling=%+v err=%v", got, err)
						}
						item = got // Retry uses the persisted issue linkage.
					}
					all := issues.List()
					if len(all) != 1 || all[0].Status != wantStatus {
						t.Fatalf("issues=%+v, want one with status %s", all, wantStatus)
					}
					issue := all[0]
					wantActor := item.FromEmail
					if action == "auto" {
						wantActor = "System (KI)"
						assertClosedIntakeIssue724(t, issue)
					} else {
						if existing {
							wantActor = options.actorEmail
						}
						if issueOpenCount(all) != 1 || len(newestOpenIssueSummaries(all, time.Now())) != 1 || !portfolioIssueIsOpen(issue) {
							t.Error("manually handled issue must remain in open lists")
						}
					}
					if issue.StatusChangedBy != wantActor || issue.StatusChangedAt.IsZero() {
						t.Errorf("status attribution=%q at=%v, want %q", issue.StatusChangedBy, issue.StatusChangedAt, wantActor)
					}
					if existing && (len(issue.StatusHistory) != 1 || issue.StatusHistory[0].ActorName != options.actorName || issue.StatusHistory[0].ActorEmail != options.actorEmail) {
						t.Errorf("status history=%+v", issue.StatusHistory)
					}
					if len(issue.Comments) != 1 || issue.Comments[0].Body != item.Suggestion.Reply || issue.Comments[0].AuthorName != options.actorName || issue.Comments[0].Kind != store.IssueCommentKindInformation {
						t.Errorf("reply comments=%+v", issue.Comments)
					}
					replies := 0
					mailer.mu.Lock()
					for _, sent := range mailer.sent {
						if sent.To == item.FromEmail && strings.Contains(sent.Subject, mailIntakeSubjectTag(item.ID)) && strings.Contains(sent.Body, item.Suggestion.Reply) {
							replies++
						}
					}
					mailer.mu.Unlock()
					if replies != 1 {
						t.Errorf("mail replies=%d, want exactly one", replies)
					}
				})
			}
		}
	}
}

func TestDemoSeedAutoDoneIssuesStayOutOfOpenLists(t *testing.T) {
	database, config := dbtest.OpenWithConfig(t)
	scoped, err := db.NewScoped(config, database)
	if err != nil {
		t.Fatal(err)
	}
	defer scoped.Close()
	if _, err := demo.Load(t.Context(), database, "../../scripts/demo/seed", demo.SeedOptions{Reset: true, DocumentDir: t.TempDir()}); err != nil {
		t.Fatal(err)
	}
	items, err := store.BindIntakeRepository(database, "musterstadt").List(t.Context(), store.IntakeFilter{Statuses: []store.IntakeStatus{store.IntakeStatusAuto}})
	if err != nil || len(items) == 0 {
		t.Fatalf("auto_done fixtures=%d err=%v", len(items), err)
	}
	storage := store.NewSQLIssueStore(store.NewTenantDB(scoped), t.TempDir())
	checked := 0
	for _, item := range items {
		// Unassigned seed items have no house issue and cannot enter its lists.
		if item.TenantSlug == "" {
			continue
		}
		checked++
		identities, err := store.EnsureTenantIdentities(t.Context(), database, []store.TenantIdentity{{Slug: item.TenantSlug}})
		if err != nil {
			t.Fatal(err)
		}
		issues, ok := store.BindIssueRepository(storage, identities[item.TenantSlug].Ref())
		if !ok {
			t.Fatalf("issue repository missing for %s", item.TenantSlug)
		}
		issue, ok := issues.Get(item.IssueID)
		if !ok {
			t.Fatalf("auto_done fixture %s lacks its issue", item.ID)
		}
		assertClosedIntakeIssue724(t, issue)
	}
	if checked == 0 {
		t.Fatal("no assigned auto_done seed issues checked")
	}
}
