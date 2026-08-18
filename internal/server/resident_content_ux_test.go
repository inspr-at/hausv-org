package server

import (
	"strings"
	"testing"
	"time"
)

func TestEventMonthGroupsRepeatsYearOnlyAtYearBoundary(t *testing.T) {
	loc := time.FixedZone("Europe/Vienna", 60*60)
	items := []houseEvent{
		{Title: "August", StartsAt: time.Date(2026, time.August, 10, 18, 0, 0, 0, loc)},
		{Title: "September", StartsAt: time.Date(2026, time.September, 12, 18, 0, 0, 0, loc)},
		{Title: "Jänner", StartsAt: time.Date(2027, time.January, 8, 18, 0, 0, 0, loc)},
		{Title: "Februar", StartsAt: time.Date(2027, time.February, 9, 18, 0, 0, 0, loc)},
	}
	groups := eventMonthGroups(items, eventViews(items, time.Date(2026, time.July, 1, 0, 0, 0, 0, loc)), loc)
	want := []string{"August 2026", "September", "Jänner 2027", "Februar"}
	if len(groups) != len(want) {
		t.Fatalf("event month groups = %+v, want %d", groups, len(want))
	}
	for i := range want {
		if groups[i].Label != want[i] {
			t.Fatalf("event month group %d label = %q, want %q", i, groups[i].Label, want[i])
		}
	}
}

func TestBallotOverviewExplainsReadOnlyOpenBallotTruthfully(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "owner@example.com",
		Role:        roleOwner,
		Tenants:     []string{"demo"},
		AuthMethods: defaultAuthMethods(),
	})
	created, err := testVoteRepository(t, a, "demo").Create(ballot{
		TenantSlug: "demo",
		Title:      "Innenhof",
		Options:    []string{"Ja", "Nein"},
		Type:       ballotTypeCircular,
		Weighting:  ballotWeightingPerShare,
		CreatedBy:  "manager@example.com",
	})
	if err != nil {
		t.Fatalf("create ballot: %v", err)
	}
	if _, _, err := testVoteRepository(t, a, "demo").Open(created.ID, time.Now()); err != nil {
		t.Fatalf("open ballot: %v", err)
	}

	body := authedRequest(t, a, "owner@example.com", "/demo/app/abstimmungen").Body.String()
	for _, want := range []string{
		"Offene Abstimmung zur Information",
		"Für diesen Zugang ist keine Stimmabgabe hinterlegt.",
		`class="vote-note"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("read-only ballot overview missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "Alles erledigt") || strings.Contains(body, "Ihre Stimme ist gespeichert") {
		t.Fatalf("read-only ballot must not claim that a vote was saved:\n%s", body)
	}
}
