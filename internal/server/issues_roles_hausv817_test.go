package server

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
)

func TestHausv817BoardPrivacyAcrossRoutes(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "board@example.com", Role: roleBeirat, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	if err := testUnitRepository(t, a, "demo").SetUnits([]unit{
		{ID: "top-2", Label: "Top 2", UnitType: unitTypeResidential, OwnerEmails: []string{"board@example.com"}},
		{ID: "top-7", Label: "Top 7", UnitType: unitTypeResidential, RenterEmails: []string{"resident@example.com"}},
	}); err != nil {
		t.Fatal(err)
	}
	items := []residentIssue{
		{ID: "own", Title: "Own unit report", LocationType: issueLocationUnit, LocationDetail: "Top 2"},
		{ID: "common", Title: "Common area report", LocationType: issueLocationCommon},
		{ID: "private", Title: "Private other unit report", LocationType: issueLocationUnit, LocationDetail: "Top 7"},
	}
	for _, item := range items {
		item.TenantSlug = "demo"
		item.AuthorEmail = "resident@example.com"
		item.Category = "Reparatur"
		item.Body = "Please investigate."
		if _, err := issueRepositoryForTest(a, "demo").Create(item); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{"/demo/app", "/demo/app/anliegen"} {
		page := authedRequest(t, a, "board@example.com", path)
		if page.Code != http.StatusOK {
			t.Fatalf("%s status=%d", path, page.Code)
		}
		body := page.Body.String()
		for _, title := range []string{"Own unit report", "Common area report"} {
			if !strings.Contains(body, title) {
				t.Errorf("%s missing %s", path, title)
			}
		}
		if strings.Contains(body, "Private other unit report") {
			t.Errorf("%s leaked private title", path)
		}
		if path == "/demo/app" {
			for _, metric := range []string{
				`class="metric" href="/demo/app/anliegen"><strong>2</strong>`,
				`class="count-tile" href="/demo/app/anliegen"><strong>2</strong>`,
			} {
				if !strings.Contains(body, metric) {
					t.Errorf("home count must exclude the other unit's private issue: missing %s", metric)
				}
			}
		}
	}
	for _, id := range []string{"own", "common", "private"} {
		want := http.StatusOK
		if id == "private" {
			want = http.StatusForbidden
		}
		if got := authedRequest(t, a, "board@example.com", "/demo/app/anliegen/"+id).Code; got != want {
			t.Errorf("%s=%d want %d", id, got, want)
		}
	}
	if got := len(a.visibleIssuesForActor(testTenantRef("demo"), "board@example.com", roleBeirat)); got != 2 {
		t.Errorf("count=%d want 2", got)
	}
	for _, tc := range []struct {
		name, location, detail, unit string
		want                         int
	}{
		{"common", issueLocationCommon, "Keller", "", http.StatusSeeOther},
		{"own", issueLocationUnit, "Badezimmer", "top-2", http.StatusSeeOther},
		{"foreign-id", issueLocationUnit, "Badezimmer", "top-7", http.StatusForbidden},
		{"foreign-label", issueLocationUnit, "Top 7", "", http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rr := authedMultipartRequest(t, a, "board@example.com", "/demo/app/anliegen", map[string]string{"category": "Reparatur", "title": tc.name, "body": "Ein neues Anliegen.", "location_type": tc.location, "location_detail": tc.detail, "unit_id": tc.unit}, "", nil)
			if rr.Code != tc.want {
				t.Fatalf("status=%d want %d", rr.Code, tc.want)
			}
		})
	}
	authored := issueRepositoryForTest(a, "demo").ListAuthor("board@example.com")
	if len(authored) != 2 {
		t.Fatalf("created %d reports", len(authored))
	}
	for _, item := range authored {
		if item.LocationType == issueLocationUnit && item.UnitID != "top-2" {
			t.Fatalf("unit binding=%q", item.UnitID)
		}
		response := authedFormRequest(t, a, "board@example.com", "/demo/app/anliegen/comment", url.Values{"id": {item.ID}, "body": {"Meine Ergänzung."}})
		if response.Code != http.StatusSeeOther {
			t.Errorf("own comment=%d", response.Code)
		}
	}
}

func TestHausv817SingleOwnerDoesNotReachQuorum(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "owner@example.com", Role: roleOwner, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	units := testUnitRepository(t, a, "demo")
	if err := units.SetUnits([]unit{
		{ID: "top-1", Label: "Top 1", MiteigentumsanteilPPM: 83333, OwnerEmails: []string{"owner@example.com"}},
		{ID: "rest", Label: "Rest", MiteigentumsanteilPPM: 916667, OwnerEmails: []string{"other@example.com"}},
	}); err != nil {
		t.Fatal(err)
	}
	item := ballot{Weighting: ballotWeightingPerShare, QuorumPPM: 500000, Options: []string{"Ja", "Nein"}, Votes: map[string]store.BallotVote{"owner@example.com": {Option: "Ja", Weight: 83333, At: time.Now()}}}
	result := a.computeBallotTally(units, testTenantRef("demo"), item)
	if result.QuorumReached || result.ParticipationPPM != 83333 {
		t.Fatalf("tally=%+v", result)
	}
}
