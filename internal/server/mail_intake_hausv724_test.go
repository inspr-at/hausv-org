package server

import (
	"context"
	"errors"
	"os"
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
func (receiptSuggester724) Suggest(context.Context, ai.TriageInput) (ai.TriageSuggestion, error) {
	return ai.TriageSuggestion{Category: "beleg", Priority: store.IssuePriorityLow, HouseSlug: "janusbergweg-123", Confidence: map[string]float64{"overall": .99}}, nil
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
			raw, err := os.ReadFile(seedDir + "/mail/01-alina-beleg-fensterreinigung.eml")
			if err != nil {
				t.Fatal(err)
			}
			mailbox := startTestMailbox(t, string(raw))
			if filed, err := a.pollMailIntakeOnce(t.Context(), "musterstadt", mailbox); err != nil || filed != 1 {
				t.Fatalf("initial intake: filed=%d err=%v", filed, err)
			}
			items, err := a.intake("musterstadt").List(t.Context(), store.IntakeFilter{IncludeUnassigned: true})
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, item := range items {
				if item.ExternalRef != "demo-mail-0001@musterstadt.example" {
					continue
				}
				found = true
				want := store.IntakeStatusOpen
				if automatic {
					want = store.IntakeStatusAuto
				}
				if item.Status != want {
					t.Fatalf("receipt classification: got %s want %s", item.Status, want)
				}
			}
			if !found {
				t.Fatal("receipt never arrived")
			}
			issues := a.repositoriesForTenant(identities[house.Slug].Ref()).issues
			countReceiptIssues := func() int {
				count := 0
				for _, issue := range issues.List() {
					if issue.Title == "Beleg Fensterreinigung Stiegenhaus" {
						count++
					}
				}
				return count
			}
			if got := countReceiptIssues(); (automatic && got != 1) || (!automatic && got != 0) {
				t.Fatalf("initial receipt issues: %d (automatic=%v)", got, automatic)
			}
			options.Reset = true
			for range 2 {
				load()
				mailbox = startTestMailbox(t, string(raw)) // restart: unread again
				if filed, err := a.pollMailIntakeOnce(t.Context(), "musterstadt", mailbox); err != nil || filed != 0 {
					t.Errorf("reset reimported a fixture: filed=%d err=%v", filed, err)
				}
				if got := countReceiptIssues(); got != 0 {
					t.Errorf("reset/mailbox cycle left %d receipt issues in the short list", got)
				}
			}
			fresh := strings.ReplaceAll(string(raw), "demo-mail-0001@musterstadt.example", "fresh-after-reset@musterstadt.example")
			mailbox = startTestMailbox(t, fresh)
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
