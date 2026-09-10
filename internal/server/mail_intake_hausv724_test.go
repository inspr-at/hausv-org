package server

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/ai"
	"github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/dbtest"
	"github.com/inspr-at/hausv-org/internal/demo"
	"github.com/inspr-at/hausv-org/internal/mailintake"
	"github.com/inspr-at/hausv-org/internal/store"
)

func TestMailIntakeDeduplicatesUnreadCopiesWithoutMessageID(t *testing.T) {
	a, _ := mailIntakeTestApp(t)
	raw := rawTestMail("", "Alina <alina@example.com>", "Beleg Fensterreinigung Stiegenhaus", "Anbei der Beleg.")
	raw = strings.ReplaceAll(raw, "Message-ID: <>\r\n", "")
	config := startTestMailbox(t, raw, raw)
	filed, err := a.pollMailIntakeOnce(t.Context(), "musterstadt", config)
	if err != nil || filed != 1 {
		t.Fatalf("two UIDs for the same mail: filed=%d err=%v, want 1", filed, err)
	}
	items, err := a.intake("musterstadt").List(t.Context(), store.IntakeFilter{IncludeUnassigned: true})
	if err != nil || len(items) != 1 || items[0].ExternalRef == "" {
		t.Fatalf("deduplicated intake: items=%+v err=%v", items, err)
	}
	// A new mailbox can assign different UIDs. Clear the poller's state too:
	// the evidence must live in the database, not in the poller's memory.
	config = startTestMailbox(t, rawTestMail("other@example.com", "Other <other@example.com>", "Other", "Other"), raw)
	a.mailIntake = mailIntakeState{}
	filed, err = a.pollMailIntakeOnce(t.Context(), "musterstadt", config)
	if err != nil || filed != 1 {
		t.Fatalf("mailbox restart: filed=%d err=%v, want only the new message", filed, err)
	}
	fetcher := mailintake.Fetcher{Config: config, Password: "geheim", Limits: mailIntakeLimits()}
	if unread, err := fetcher.Unread(t.Context(), 10); err != nil || len(unread) != 0 {
		t.Fatalf("known copies must also be marked seen: unread=%d err=%v", len(unread), err)
	}
}

type receiptSuggester724 struct{}

func (receiptSuggester724) Label() string { return "test receipt" }
func (receiptSuggester724) Suggest(_ context.Context, input ai.TriageInput) (ai.TriageSuggestion, error) {
	category := store.IntakeCategoryRepair
	if input.Subject == "Beleg Fensterreinigung Stiegenhaus" {
		category = "beleg"
	}
	return ai.TriageSuggestion{Category: category, Priority: store.IssuePriorityLow, HouseSlug: "janusbergweg-123", Confidence: map[string]float64{"overall": .99}}, nil
}

func TestDemoMailboxIntakeAcrossRepeatedResets(t *testing.T) {
	for _, automatic := range []bool{false, true} {
		name := "without_AI"
		if automatic {
			name = "automatic_receipt"
		}
		t.Run(name, func(t *testing.T) {
			a, _ := mailIntakeTestApp(t)
			database, config := dbtest.OpenWithConfig(t)
			scoped, err := db.NewScoped(config, database)
			if err != nil {
				t.Fatal(err)
			}
			defer scoped.Close()
			const seedDir = "../../scripts/demo/seed"
			options := demo.SeedOptions{DocumentDir: t.TempDir()}
			load := func() {
				t.Helper()
				if _, err := demo.Load(t.Context(), database, seedDir, options); err != nil {
					t.Fatal(err)
				}
			}
			load()
			a.intake = func(org string) store.IntakeRepository { return store.BindIntakeRepository(database, org) }
			a.intakeMailSeen = func(org string) store.IntakeMailSeenRepository {
				return store.BindIntakeMailSeenRepository(database, org)
			}
			a.orgSettings = func(org string) store.OrgSettingsRepository { return store.BindOrgSettingsRepository(database, org) }
			a.issueStore = store.NewSQLIssueStore(store.NewTenantDB(scoped), t.TempDir())
			house := a.tenants["demo"]
			house.Slug = "janusbergweg-123"
			house.Name = "Janusbergweg 123"
			addTestTenant(a, house)
			identities, err := store.EnsureTenantIdentities(t.Context(), database, []store.TenantIdentity{{Slug: house.Slug}})
			if err != nil {
				t.Fatal(err)
			}
			a.tenantIdentities[house.Slug] = identities[house.Slug]
			addTestPerson(t, a, "alina.eigentuemer@musterstadt.example", house.Slug, roleOwner)
			if automatic {
				a.triage = receiptSuggester724{}
			}
			files, err := filepath.Glob(filepath.Join(seedDir, "mail", "*.eml"))
			if err != nil || len(files) != 5 {
				t.Fatalf("mail fixtures: count=%d err=%v", len(files), err)
			}
			var rawMessages []string
			messages := make(map[string]mailintake.Message)
			for _, file := range files {
				raw, err := os.ReadFile(file)
				if err != nil {
					t.Fatal(err)
				}
				message, err := mailintake.Parse(raw, mailintake.Limits{})
				if err != nil || message.MessageID == "" {
					t.Fatalf("parse fixture %s: %v", file, err)
				}
				rawMessages = append(rawMessages, string(raw))
				messages[message.DedupeKey()] = message
			}
			issues := a.repositoriesForTenant(identities[house.Slug].Ref()).issues
			seedItems, err := a.intake("musterstadt").List(t.Context(), store.IntakeFilter{})
			if err != nil {
				t.Fatal(err)
			}
			const receiptKey = "demo-mail-0001@musterstadt.example"
			assertCounts := func(want int) {
				t.Helper()
				items, err := a.intake("musterstadt").List(t.Context(), store.IntakeFilter{})
				if err != nil {
					t.Fatal(err)
				}
				if len(items) != len(seedItems)+want*len(messages) {
					t.Errorf("inbox count=%d, want %d", len(items), len(seedItems)+want*len(messages))
				}
				for key, message := range messages {
					intakeCount, issueCount := 0, 0
					for _, item := range items {
						if item.ExternalRef == key {
							intakeCount++
						}
					}
					for _, issue := range issues.List() {
						if issue.Source == "email" && issue.AuthorEmail == message.FromEmail && issue.Title == message.Subject {
							issueCount++
							if automatic && key == receiptKey {
								assertClosedIntakeIssue724(t, issue)
								if issue.StatusChangedBy != "System (KI)" {
									t.Errorf("automatic receipt actor=%q", issue.StatusChangedBy)
								}
							}
						}
					}
					if intakeCount != want || issueCount != want {
						t.Errorf("fixture %s: intakes=%d issues=%d, want %d each", key, intakeCount, issueCount, want)
					}
				}
			}
			// Initial import, then two resets with a real mailbox import between
			// them. Manually approve the other messages to exercise issue cleanup too.
			for cycle := range 3 {
				if cycle > 0 {
					options.Reset = true
					load()
					assertCounts(0)
				}
				mailbox := startTestMailbox(t, rawMessages...)
				if filed, err := a.pollMailIntakeOnce(t.Context(), "musterstadt", mailbox); err != nil || filed != 5 {
					t.Fatalf("cycle %d intake: filed=%d err=%v, want 5", cycle, filed, err)
				}
				items, err := a.intake("musterstadt").List(t.Context(), store.IntakeFilter{})
				if err != nil {
					t.Fatal(err)
				}
				for _, item := range items {
					if _, fixture := messages[item.ExternalRef]; !fixture {
						continue
					}
					want := store.IntakeStatusOpen
					if automatic {
						want = store.IntakeStatusProposed
						if item.ExternalRef == receiptKey {
							want = store.IntakeStatusAuto
						}
					}
					if item.Status != want {
						t.Fatalf("cycle %d fixture %s: status=%s want=%s", cycle, item.ExternalRef, item.Status, want)
					}
					if _, found := a.findIntakeIssue(issues, item); found != (item.Status == store.IntakeStatusAuto) {
						t.Fatalf("fixture %s: issue exists before manual approval=%v, status=%s", item.ExternalRef, found, item.Status)
					}
					if item.Status != store.IntakeStatusAuto {
						if item.Suggestion == nil {
							item.Suggestion = &store.IntakeSuggestion{TenantSlug: house.Slug, Category: store.IntakeCategoryOther, Priority: store.IssuePriorityNorm}
						}
						if err := a.handleIntake(t.Context(), "musterstadt", item, intakeHandleOptions{status: store.IntakeStatusApproved, action: "approve", actorEmail: "verwaltung@example.com", actorName: "Vera"}); err != nil {
							t.Fatal(err)
						}
					}
				}
				assertCounts(1)
				// A fresh mailbox makes all fixtures unread again. Restart the
				// poller's memory too, so only the persistent ledger can dedupe.
				a.mailIntake = mailIntakeState{}
				mailbox = startTestMailbox(t, rawMessages...)
				if filed, err := a.pollMailIntakeOnce(t.Context(), "musterstadt", mailbox); err != nil || filed != 0 {
					t.Fatalf("cycle %d duplicate fetch: filed=%d err=%v", cycle, filed, err)
				}
				assertCounts(1)
				fetcher := mailintake.Fetcher{Config: mailbox, Password: "geheim", Limits: mailIntakeLimits()}
				if unread, err := fetcher.Unread(t.Context(), 10); err != nil || len(unread) != 0 {
					t.Fatalf("duplicates must be marked seen: unread=%d err=%v", len(unread), err)
				}
			}
			fresh := strings.ReplaceAll(rawMessages[0], receiptKey, "fresh-after-reset@musterstadt.example")
			mailbox := startTestMailbox(t, fresh)
			if filed, err := a.pollMailIntakeOnce(t.Context(), "musterstadt", mailbox); err != nil || filed != 1 {
				t.Fatalf("new mail after reset: filed=%d err=%v", filed, err)
			}
		})
	}
}

type pausedMailSeen724 struct {
	store.IntakeMailSeenRepository
	paused  atomic.Bool
	entered chan struct{}
	release chan struct{}
}

func (r *pausedMailSeen724) Seen(ctx context.Context, key string) (bool, error) {
	seen, err := r.IntakeMailSeenRepository.Seen(ctx, key)
	if r.paused.CompareAndSwap(false, true) {
		close(r.entered)
		<-r.release
	}
	return seen, err
}

func TestMailIntakeSerializesConcurrentPolls(t *testing.T) {
	a, _ := mailIntakeTestApp(t)
	config := startTestMailbox(t, rawTestMail("parallel@example.com", "Alina <alina@example.com>", "Beleg", "Text"))
	original := a.intakeMailSeen
	paused := &pausedMailSeen724{IntakeMailSeenRepository: original("musterstadt"), entered: make(chan struct{}), release: make(chan struct{})}
	a.intakeMailSeen = func(org string) store.IntakeMailSeenRepository {
		if org == "musterstadt" {
			return paused
		}
		return original(org)
	}
	finished := make(chan error, 1)
	go func() {
		_, err := a.pollMailIntakeOnce(t.Context(), "musterstadt", config)
		finished <- err
	}()
	defer func() {
		close(paused.release)
		select {
		case err := <-finished:
			if err != nil {
				t.Errorf("first poll: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("first poll did not finish")
		}
		items, err := a.intake("musterstadt").List(t.Context(), store.IntakeFilter{IncludeUnassigned: true})
		if err != nil || len(items) != 1 {
			t.Errorf("concurrent polls created %d items: err=%v", len(items), err)
		}
	}()
	select {
	case <-paused.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("first poll did not reach the ledger")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 150*time.Millisecond)
	defer cancel()
	if filed, err := a.pollMailIntakeOnce(ctx, " Musterstadt ", config); !errors.Is(err, context.DeadlineExceeded) || filed != 0 {
		t.Errorf("overlapping poll must wait and respect cancellation: filed=%d err=%v", filed, err)
	}
	// A slow organisation does not hold up another organisation's mailbox.
	other := startTestMailbox(t, rawTestMail("other@example.com", "Other <other@example.com>", "New", "Text"))
	if filed, err := a.pollMailIntakeOnce(t.Context(), "other", other); err != nil || filed != 1 {
		t.Errorf("independent organisation blocked: filed=%d err=%v", filed, err)
	}
}

func TestMailIntakeSkipsPersistedMessageID(t *testing.T) {
	a, _ := mailIntakeTestApp(t)
	if err := a.intakeMailSeen("musterstadt").Record(t.Context(), " <known@example.com> ", "old-intake"); err != nil {
		t.Fatal(err)
	}
	config := startTestMailbox(t, rawTestMail("known@example.com", "Alina <alina@example.com>", "Beleg", "Text"))
	filed, err := a.pollMailIntakeOnce(t.Context(), "musterstadt", config)
	if err != nil || filed != 0 {
		t.Fatalf("persisted message: filed=%d err=%v", filed, err)
	}
	items, err := a.intake("musterstadt").List(t.Context(), store.IntakeFilter{IncludeUnassigned: true})
	if err != nil || len(items) != 0 {
		t.Fatalf("known message created an intake: items=%+v err=%v", items, err)
	}
	// The key is scoped to the organisation, even for the same message.
	config = startTestMailbox(t, rawTestMail("known@example.com", "Alina <alina@example.com>", "Beleg", "Text"))
	filed, err = a.pollMailIntakeOnce(t.Context(), "other", config)
	if err != nil || filed != 1 {
		t.Fatalf("other organisation: filed=%d err=%v, want 1", filed, err)
	}
}
