package demo

import (
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/dbtest"
	"github.com/inspr-at/hausv-org/internal/store"
)

func TestDemoContentSurvivesReseed(t *testing.T) {
	database, config := dbtest.OpenWithConfig(t)
	scoped, err := db.NewScoped(config, database)
	if err != nil {
		t.Fatal(err)
	}
	defer scoped.Close()
	lanes := store.NewTenantDB(scoped)
	anchor := time.Date(2030, 8, 1, 12, 0, 0, 0, seedDemoDay.Location())
	options := SeedOptions{Reset: true, Anchor: anchor, DocumentDir: t.TempDir()}
	load := func() {
		t.Helper()
		if _, err := Load(t.Context(), database, "../../scripts/demo/seed", options); err != nil {
			t.Fatal(err)
		}
	}
	load()
	identities, err := store.EnsureTenantIdentities(t.Context(), database, []store.TenantIdentity{{Slug: "janusbergweg-123"}})
	if err != nil {
		t.Fatal(err)
	}
	tenant := identities["janusbergweg-123"].Ref()
	contacts, _ := store.BindContactBookRepository(store.NewSQLContactBookStore(lanes), tenant)
	votes, _ := store.BindVoteRepository(store.NewSQLVoteStore(lanes), tenant)
	members := store.BindOrganisationMemberRepository(database, "musterstadt")
	settings := store.BindOrgSettingsRepository(database, "musterstadt")
	verify := func() {
		t.Helper()
		items := contacts.List(false)
		if len(items) != 4 {
			t.Fatalf("active contacts = %d", len(items))
		}
		for _, item := range items {
			if item.Phone == "" || item.Email == "" || item.Notes == "" || item.Company == "" {
				t.Fatalf("incomplete contact: %+v", item)
			}
		}
		if got := votes.List(); len(got) != 2 {
			t.Fatalf("ballots = %d", len(got))
		}
		closed, found := votes.Get("demo-ballot-fassade-2027")
		if !found || closed.Status != store.BallotStatusClosed || len(closed.Votes) != 1 || !closed.ClosesAt.Before(anchor) {
			t.Fatalf("closed ballot: %+v", closed)
		}
		for _, vote := range closed.Votes {
			if vote.Option != "Ja" || vote.Weight != 1000000 || vote.At.Before(closed.OpensAt) || vote.At.After(closed.ClosesAt) {
				t.Fatalf("vote: %+v", vote)
			}
		}
		open, found := votes.Get("demo-ballot-ebikes")
		if !found || open.Status != store.BallotStatusOpen || !open.OpensAt.Before(anchor) || !open.ClosesAt.After(anchor) || len(open.Votes) != 0 {
			t.Fatalf("open ballot: %+v", open)
		}
		staff, err := members.List(t.Context())
		if err != nil || len(staff) != 3 {
			t.Fatalf("staff (including unrelated member): %+v %v", staff, err)
		}
		for email, role := range map[string]string{"vera.verwalter@musterstadt.example": store.OrganisationRoleAdmin, "paul.verwalter@musterstadt.example": store.OrganisationRoleClerk} {
			member, found, err := members.Get(t.Context(), email)
			if err != nil || !found || member.Role != role || len(member.Granted) != 12 {
				t.Fatalf("member: %+v %v", member, err)
			}
		}
		got, err := settings.Get(t.Context())
		want := store.OrgCounters{Approved: 144, Edited: 6, Rejected: 32, Auto: 30}
		if err != nil || got.Counters != want {
			t.Fatalf("counters=%+v want=%+v err=%v", got.Counters, want, err)
		}
		repo := store.BindIntakeRepository(database, "musterstadt")
		var raw []seedIntake
		if err := readJSON("../../scripts/demo/seed/intake.json", &raw); err != nil {
			t.Fatal(err)
		}
		for _, fixture := range raw {
			if fixture.StatusHint != "edited" {
				continue
			}
			item, err := repo.Get(t.Context(), fixture.ID)
			if err != nil || item.Status != store.IntakeStatusEdited || item.Handling == nil || item.Suggestion == nil || item.Suggestion.Reply != fixture.HandledReply || item.Suggestion.Reply == fixture.Precomputed.Reply || item.IssueID == "" {
				t.Fatalf("edited intake: %+v %v", item, err)
			}
		}
	}
	// Non-fixture employees must remain intact; rerunning must repair fixture
	// state without accumulating counters or retaining demo votes/reminders.
	if err := members.Save(t.Context(), store.OrganisationMember{Email: "extra@musterstadt.example", Role: store.OrganisationRoleClerk}); err != nil {
		t.Fatal(err)
	}
	verify()
	if _, err := contacts.Deactivate("demo-contact-lift", anchor); err != nil {
		t.Fatal(err)
	}
	if _, _, err := votes.CastVote("demo-ballot-ebikes", "alina.eigentuemer@musterstadt.example", "Nein", 1000000, anchor); err != nil {
		t.Fatal(err)
	}
	if err := settings.Increment(t.Context(), "approved"); err != nil {
		t.Fatal(err)
	}
	if err := members.Save(t.Context(), store.OrganisationMember{Email: "vera.verwalter@musterstadt.example", Role: store.OrganisationRoleClerk}); err != nil {
		t.Fatal(err)
	}
	load()
	verify()
	options.Reset = false
	load()
	verify()
}

func TestDemoEventDetailsArePresent(t *testing.T) {
	var events []seedEvent
	if err := readJSON("../../scripts/demo/seed/events.json", &events); err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if len(event.Description) < 100 {
			t.Fatalf("missing useful details for %s", event.ID)
		}
	}
}
