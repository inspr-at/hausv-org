package server

import (
	"net/http"
	"regexp"
	"strconv"
	"testing"
	"time"
)

// Both responsive compositions are rendered in the same response. A visit to
// the announcement archive changes the user's read state; changing width must
// not change which count the two compositions display.
func TestPortalAnnouncementCountsShareReadStateHAUSV659(t *testing.T) {
	for _, role := range []string{roleAdmin, roleResident} {
		t.Run(role, func(t *testing.T) {
			const email = "counts@example.com"
			a := newTestPortalApp(t, userProfile{Email: email, Role: role, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
			repositories := testRepositories(a, "demo")
			now := time.Now()
			if err := repositories.announcementReads.MarkSeen(email, now.Add(-2*time.Hour)); err != nil {
				t.Fatal(err)
			}
			expired := now.Add(-30 * time.Minute)
			items := []announcement{
				{Title: "Liftwartung", PublishedAt: now.Add(-time.Hour)},
				{Title: "Hausversammlung", PublishedAt: now.Add(-time.Hour)},
				{Title: "Reinigung", PublishedAt: now.Add(-time.Hour)},
				{Title: "Schon gelesen", PublishedAt: now.Add(-3 * time.Hour)},
				{Title: "Abgelaufen", PublishedAt: now.Add(-time.Hour), ExpiresAt: &expired},
				{Title: "Geplant", PublishedAt: now.Add(time.Hour)},
			}
			for _, item := range items {
				item.TenantSlug, item.Body, item.Category = "demo", "Beitrag für den Hausüberblick", "Info"
				if _, err := repositories.announcements.Create(item); err != nil {
					t.Fatal(err)
				}
			}

			check := func(want int) {
				t.Helper()
				page := authedRequest(t, a, email, "/demo/app")
				if page.Code != http.StatusOK {
					t.Fatalf("home status = %d", page.Code)
				}
				for _, metric := range []struct{ class, label string }{{"metric", "neue Beiträge"}, {"count-tile", "ungelesen"}} {
					pattern := `<a class="` + metric.class + `" href="/demo/app/announcements"[^>]*><strong>([0-9]+)</strong><span>` + metric.label + `</span></a>`
					matches := regexp.MustCompile(pattern).FindAllStringSubmatch(page.Body.String(), -1)
					if len(matches) != 1 {
						t.Fatalf("%s: got %d count metrics, want one", metric.class, len(matches))
					}
					if matches[0][1] != strconv.Itoa(want) {
						t.Errorf("%s = %s, want %d unread announcements", metric.class, matches[0][1], want)
					}
				}
			}
			// Repeated overview loads leave all three unread. The total number of
			// announcements is deliberately different from the unread count.
			for range 3 {
				check(3)
			}
			archive := authedRequest(t, a, email, "/demo/app/announcements")
			if archive.Code != http.StatusOK {
				t.Fatalf("archive status = %d", archive.Code)
			}
			check(0)
		})
	}
}
