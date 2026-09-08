package server

import (
	"html"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestAnnouncementSortKeepsPinnedGroupAndFilters(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	// Cross a month boundary so a lexical sort of the displayed date is wrong.
	older := time.Date(2025, 8, 31, 9, 0, 0, 0, time.UTC)
	newer := older.Add(24 * time.Hour)
	for _, item := range []announcement{
		{Title: "Lift alt", PublishedAt: older},
		{Title: "Lift neu", PublishedAt: newer},
		{Title: "Lift fixiert alt", PublishedAt: older, Pinned: true},
		{Title: "Lift fixiert neu", PublishedAt: newer, Pinned: true},
		{Title: "Anderes Thema", PublishedAt: newer},
		{Title: "Lift Termin", Category: "Termin", PublishedAt: older},
		{Title: "Lift geplant", PublishedAt: time.Now().Add(time.Hour)},
	} {
		item.TenantSlug = "demo"
		if item.Category == "" {
			item.Category = "Wartung"
		}
		if _, err := testRepositories(a, "demo").announcements.Create(item); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		order string
		want  []string
	}{
		{"", []string{"Lift fixiert neu", "Lift fixiert alt", "Lift neu", "Lift alt"}},
		{"oldest", []string{"Lift fixiert alt", "Lift fixiert neu", "Lift alt", "Lift neu"}},
		{"newest", []string{"Lift fixiert neu", "Lift fixiert alt", "Lift neu", "Lift alt"}},
		{"unknown", []string{"Lift fixiert neu", "Lift fixiert alt", "Lift neu", "Lift alt"}},
	} {
		t.Run(tc.order, func(t *testing.T) {
			response := authedRequest(t, a, "resident@example.com", "/demo/app/announcements?q=Lift&category=Wartung&sort="+tc.order)
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d", response.Code)
			}
			body := html.UnescapeString(response.Body.String())
			var got []string
			for _, match := range regexp.MustCompile(`<span class="announcement-card-title">([^<]+)</span>`).FindAllStringSubmatch(body, -1) {
				got = append(got, match[1])
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("titles = %v, want %v", got, tc.want)
			}
			for _, want := range []string{`name="q" value="Lift"`, `name="category" value="Wartung"`, `name="sort" form="announcement-filter"`, "4 Aushänge", "Oben fixiert", "Weitere Beiträge"} {
				if !strings.Contains(body, want) {
					t.Errorf("missing filter/group state %q", want)
				}
			}
			if tc.order == "oldest" {
				for _, want := range []string{`href="/demo/app/announcements?category=Info&q=Lift&sort=oldest"`, `<option value="oldest" selected`} {
					if !strings.Contains(body, want) {
						t.Errorf("oldest sort must survive category navigation and form submission: %q", want)
					}
				}
			}
		})
	}
}
