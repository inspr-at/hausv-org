package server

import (
	"html"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/energy"
)

// These are the pre-migration form contracts. Existing integration tests exercise
// their successful POST/redirect/flash paths; this matrix binds the actual
// rendered forms to those same endpoints and checks the Origin gate on each.
func TestLegacyRoutesUseSharedShellAndPreserveFormsHAUSV705(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	if _, err := a.inviteStore.Add(userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}); err != nil {
		t.Fatal(err)
	}
	saveClaimedEnergyProfileHAUSV410(t, a, energy.HomeHouse)
	item, err := issueRepositoryForTest(a, "demo").Create(residentIssue{TenantSlug: "demo", AuthorEmail: "resident@example.com", Title: "Kellerlicht", Body: "Bitte prüfen.", Category: "Reparatur", LocationType: issueLocationCommon, Status: issueStatusProgress, Priority: issuePriorityNorm})
	if err != nil {
		t.Fatal(err)
	}
	question := authedFormRequest(t, a, "admin@example.com", "/demo/app/anliegen/comment", url.Values{"id": {item.ID}, "body": {"Welches Licht?"}, "message_type": {issueCommentKindQuestion}})
	if question.Code != http.StatusSeeOther {
		t.Fatalf("question setup: %d", question.Code)
	}
	base := time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)
	if err := a.parkingStore.AppendReadings("demo", []parkingNumericSample{{At: base, Value: 100}, {At: base.Add(2 * time.Hour), Value: 102}}, []parkingNumericSample{{At: base, Value: .2}, {At: base.Add(time.Hour), Value: .4}}); err != nil {
		t.Fatal(err)
	}
	if err := testUnitRepository(t, a, "demo").SetUnits([]unit{{ID: "top-1", TenantSlug: "demo", Label: "Top 1", UnitType: unitTypeResidential}}); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		route, email string
		forms        map[string][]string
	}{
		{"/app/anliegen/" + item.ID + "?created=1", "resident@example.com", map[string][]string{"/app/anliegen/comment": {"id", "redirect", "body", "attachments"}}},
		{"/app/parking/settings", "admin@example.com", map[string][]string{"/app/parking/settings": {"effective_from", "grid_fee_eur_per_kwh", "base_fee_eur", "surplus_rate_eur_per_kwh"}, "/app/parking/reminders": {}}},
		{"/app/parking/settings?section=charging", "admin@example.com", map[string][]string{"/app/parking/charging/settings": {"controller_enabled", "shadow_mode", "start_soc_percent", "start_feed_in_w", "stop_soc_percent", "stop_feed_in_w", "stop_delay_minutes", "min_on_minutes", "min_off_minutes"}}},
		{"/app/parking/settings?section=telegram", "admin@example.com", map[string][]string{"/app/parking/charging/telegram/link": {"email"}}},
		{"/app/parking/month/2026-06", "admin@example.com", map[string][]string{"/app/parking/month": {"month", "paid", "paid_at", "payment_method", "payment_reference"}}},
		{"/app/settings/parking-access", "admin@example.com", map[string][]string{"/app/settings/parking-access": {"email", "parking"}}},
		{"/app/settings/modules", "admin@example.com", map[string][]string{"/app/settings/modules": {"modules"}}},
		{"/app/settings/data-export", "admin@example.com", map[string][]string{"/app/settings/data-export/preview": {"source"}}},
		{"/app/settings/energy-data", "admin@example.com", map[string][]string{"/app/settings/energy-data/export": {}, "/app/settings/energy-data/history/delete": {"confirmation"}, "/app/settings/energy-data/profile/delete": {"confirmation"}}},
		{"/app/dokumente/rechnungen/import", "admin@example.com", map[string][]string{"/app/dokumente/rechnungen/import/preview": {"invoice_file"}}},
		{"/app/settings/payments/import?period=2026-06", "admin@example.com", map[string][]string{"/app/settings/payments/import/preview": {"period", "camt_file"}}},
	}
	for _, tc := range cases {
		t.Run(tc.route, func(t *testing.T) {
			rr := authedRequest(t, a, tc.email, "/demo"+tc.route)
			if rr.Code != http.StatusOK {
				t.Fatalf("GET status = %d, want 200", rr.Code)
			}
			body := rr.Body.String()
			for _, want := range []string{"data-templ-legacy", "data-context-bar", "data-switcher", `class="sidebar"`, "data-portal-section-header", `/demo/assets/portal-shell.css?v=`, `/demo/assets/switcher.js?v=`} {
				if !strings.Contains(body, want) {
					t.Fatalf("shared page missing %q", want)
				}
			}
			for _, one := range []string{`<aside class="sidebar"`, `id="main-content"`, `data-portal-section-header`, `<h1>`} {
				if strings.Count(body, one) != 1 {
					t.Fatalf("want exactly one %q", one)
				}
			}
			if strings.Contains(body, `class="app-shell"`) || strings.Contains(body, `class="side-foot"`) {
				t.Fatal("legacy shell returned")
			}
			forms := legacyRouteFormsHAUSV705(t, body)
			for action, expected := range tc.forms {
				form, ok := forms["/demo"+action]
				if !ok {
					t.Fatalf("form action %q missing", action)
				}
				if form.method != "post" {
					t.Fatalf("%s method = %q", action, form.method)
				}
				actual := make([]string, 0, len(form.values))
				for name := range form.values {
					actual = append(actual, name)
				}
				sort.Strings(actual)
				sort.Strings(expected)
				if !reflect.DeepEqual(actual, expected) {
					t.Fatalf("%s fields = %v, want %v", action, actual, expected)
				}
				if strings.Contains(action, "import/preview") || action == "/app/anliegen/comment" {
					if form.enctype != "multipart/form-data" {
						t.Fatalf("%s lost multipart encoding", action)
					}
				}
				blocked := authedFormRequestWithOrigin(t, a, tc.email, "/demo"+action, form.values, "https://other.example")
				if blocked.Code != http.StatusForbidden {
					t.Fatalf("%s foreign Origin status = %d, want 403", action, blocked.Code)
				}
			}
			if strings.Contains(tc.route, "created=1") && !strings.Contains(body, "Anliegen gemeldet") {
				t.Fatal("creation flash missing")
			}
		})
	}
}

type legacyRouteFormHAUSV705 struct {
	method, enctype string
	values          url.Values
}

func legacyRouteFormsHAUSV705(t *testing.T, body string) map[string]legacyRouteFormHAUSV705 {
	t.Helper()
	// Parse only the emitted form/field tags; this oracle does not need a new
	// production dependency for a full HTML parser.
	attr := func(tag, key string) string {
		match := regexp.MustCompile(`(?:^|\s)` + key + `="([^"]*)"`).FindStringSubmatch(tag)
		if len(match) == 2 {
			return html.UnescapeString(match[1])
		}
		return ""
	}
	forms := map[string]legacyRouteFormHAUSV705{}
	for _, form := range regexp.MustCompile(`(?s)<form\b([^>]*)>(.*?)</form>`).FindAllStringSubmatch(body, -1) {
		values := url.Values{}
		for _, field := range regexp.MustCompile(`<(?:input|select|textarea|button)\b[^>]*>`).FindAllString(form[2], -1) {
			if name := attr(field, "name"); name != "" {
				values.Add(name, attr(field, "value"))
			}
		}
		forms[attr(form[1], "action")] = legacyRouteFormHAUSV705{attr(form[1], "method"), attr(form[1], "enctype"), values}
	}
	return forms
}
