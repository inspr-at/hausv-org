package server

import (
	"html"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/store"
)

// HAUSV-698: physical building terms remain valid. Hausüberblick and
// Hausgemeinschaft are explicitly retained pending the owner's decision.
var terminologyPhysicalWhitelist = []string{
	"Hausordnung", "Haustor", "Hausmeister", "Hausbetreuung", "Haustechnik",
	"Hausanschluss", "Hauseingang", "Hausverbrauch", "Haus-Schalter",
	"Hausüberblick", "Hausgemeinschaft",
}

var (
	terminologyNonContent = regexp.MustCompile(`(?s)<(?:style|script)\b[^>]*>.*?</(?:style|script)>|<!--.*?-->`)
	terminologyAriaLabel  = regexp.MustCompile(`aria-label="([^"]*)"`)
	terminologyTag        = regexp.MustCompile(`<[^>]*>`)
	// The plural rule also catches every numeric counter, including "12 Häuser".
	terminologyOldLabels = regexp.MustCompile(`\bHäuser\b|\bHaus\s*·\s*Einheit\b|\bHaus\s+zugeordnet\b`)
)

func terminologyPageText(page string) string {
	// Historical release copy is immutable and deliberately outside this task.
	page = terminologyNonContent.ReplaceAllString(withoutReleaseNotes(page), " ")
	labels := terminologyAriaLabel.FindAllStringSubmatch(page, -1)
	text := terminologyTag.ReplaceAllString(page, " ")
	for _, label := range labels {
		text += " " + label[1]
	}
	return strings.Join(strings.Fields(html.UnescapeString(text)), " ")
}

func TestTerminologyHAUSV698MainPagesByRole(t *testing.T) {
	// Both management roles exercise the Verwaltung family; owners and residents
	// cover the other two families with their actual permissions and HTTP pages.
	for _, role := range []string{roleAdmin, roleManager, roleOwner, roleResident} {
		t.Run(role, func(t *testing.T) {
			a, intake, _ := newInboxTestApp(t, role)
			pages := []string{
				"/app", "/app/anliegen", "/app/announcements", "/app/events",
				"/app/kontakte", "/app/dokumente", "/app/settings",
				"/app/settings/profile", "/app/settings/notifications", "/app/audit", "/app/hilfe",
			}
			if role == roleAdmin || role == roleManager {
				seedInboxItem(t, intake, "terms", store.IntakeStatusProposed, inboxSuggestion(.9))
				pages = append(pages, "/app/verwaltung", "/app/verwaltung/posteingang",
					"/app/verwaltung/posteingang/terms", "/app/verwaltung/rechte",
					"/app/settings/building", "/app/settings/users", "/app/anliegen/board")
			}
			if role == roleAdmin {
				pages = append(pages, "/app/verwaltung/einstellungen")
			}
			for _, path := range pages {
				t.Run(path, func(t *testing.T) {
					response := authedRequest(t, a, "vera@example.com", "/demo"+path)
					if response.Code != http.StatusOK {
						t.Fatalf("page status = %d, want 200", response.Code)
					}
					text := terminologyPageText(response.Body.String())
					if old := terminologyOldLabels.FindAllString(text, -1); len(old) > 0 {
						t.Errorf("obsolete management labels: %q", old)
					}
					if !strings.Contains(text, "Liegenschaft") {
						t.Error("page must retain its Liegenschaft context")
					}
				})
			}
		})
	}
}

func TestTerminologyHAUSV698GuardAllowsPhysicalTerms(t *testing.T) {
	for _, term := range terminologyPhysicalWhitelist {
		if terminologyOldLabels.MatchString(terminologyPageText("<p>" + term + "</p>")) {
			t.Errorf("physical or owner-retained term rejected: %s", term)
		}
	}
	for _, page := range []string{
		"<h2>Häuser</h2>", "<span>12 Häuser</span>",
		"<p>Haus <span>·</span> Einheit</p>", "<p>Haus&nbsp;zugeordnet</p>",
		`<nav aria-label="Häuser"></nav>`,
	} {
		if !terminologyOldLabels.MatchString(terminologyPageText(page)) {
			t.Errorf("obsolete label escaped the page guard: %s", page)
		}
	}
}
