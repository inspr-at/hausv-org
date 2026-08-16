package web

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/inspr-at/hausv-org/internal/view"
)

func TestAuthenticatedTemplPagesUsePortalDocument(t *testing.T) {
	portal := PortalPageData{
		Title:                  "Testportal",
		HouseName:              "Haus am Park",
		Address:                "Parkgasse 1",
		DisplayName:            "Ada Beispiel",
		Initials:               "AB",
		Role:                   "Verwaltung",
		CanUseResidentAreas:    true,
		CanManageIssues:        true,
		CanCreateResidentIssue: true,
		Modules: PortalModules{
			Announcements: true, Events: true, Contacts: true, Documents: true,
			Issues: true, Votes: true, Handovers: true, Users: true, Audit: true, Help: true,
		},
		Contexts: []PortalContext{
			{TenantSlug: "park", HouseName: "Haus am Park", Role: "Verwaltung", Current: true},
			{TenantSlug: "see", HouseName: "Haus am See", Role: "Bewohner"},
		},
	}
	const assetVersion = "shell-test"
	pages := []struct {
		name         string
		component    templ.Component
		marker       string
		extraScripts []string
	}{
		{"PortalPage", PortalPage(portal), "data-templ-portal", nil},
		{"AnnouncementsPage", AnnouncementsPage(AnnouncementsPageData{Portal: portal, AssetVersion: assetVersion, CanManageAnnouncements: true}), "data-templ-portal", []string{"announcements.js", "attachments.js"}},
		{"AuditPage", AuditPage(AuditPageData{Portal: portal}), "data-templ-audit", nil},
		{"BallotsPage", BallotsPage(BallotsPageData{Portal: portal, AssetVersion: assetVersion, CanManageVotes: true}), "data-templ-ballots", []string{"announcements.js", "attachments.js"}},
		{"ContactsPage", ContactsPage(ContactsPageData{Portal: portal, AssetVersion: assetVersion, CanManageContacts: true}), "data-templ-contacts", nil},
		{"DocumentsPage", DocumentsPage(DocumentsPageData{Portal: portal, AssetVersion: assetVersion, CanManageDocuments: true}), "data-templ-documents", []string{"announcements.js", "attachments.js"}},
		{"EventsPage", EventsPage(EventsPageData{Portal: portal, AssetVersion: assetVersion, CanManageEvents: true}), "data-templ-events", []string{"announcements.js", "attachments.js"}},
		{"HandoversPage", HandoversPage(HandoversPageData{Portal: portal, AssetVersion: assetVersion}), "data-templ-handovers", []string{"attachments.js"}},
		{"HelpPage", HelpPage(HelpPageData{Portal: portal, ConnectorAvailable: true, ConnectorConnected: true, CanManageEnergy: true}), "data-templ-help", nil},
		{"IssuesPage", IssuesPage(IssuesPageData{Portal: portal, AssetVersion: assetVersion, CanCreateIssue: true, OpenIssueCreate: true}), "data-templ-issues", []string{"attachments.js", "issues.js"}},
		{"OnboardingPage", OnboardingPage(OnboardingPageData{Portal: portal, Step: 2, Progress: 40, CanControlEnergy: true}), "data-templ-onboarding", nil},
		{"SettingsHubPage", SettingsHubPage(SettingsHubPageData{Portal: portal}), "data-templ-settings", nil},
		{"ProfileSettingsPage", ProfileSettingsPage(ProfileSettingsPageData{Portal: portal}), "data-templ-settings", nil},
		{"NotificationSettingsPage", NotificationSettingsPage(NotificationSettingsPageData{Portal: portal}), "data-templ-settings", nil},
		{"BuildingSettingsPage", BuildingSettingsPage(BuildingSettingsPageData{Portal: portal, AssetVersion: assetVersion, BuildingSection: "units", Units: []view.BuildingUnitView{{ID: "1", Label: "Top 1"}}}), "data-templ-settings", []string{"attachments.js", "building-settings.js"}},
		{"UserSettingsPage", UserSettingsPage(UserSettingsPageData{Portal: portal, AssetVersion: assetVersion, IsAdmin: true}), "data-templ-settings", []string{"users.js"}},
	}

	assertPageTableCoversTemplPages(t, pages)
	for _, page := range pages {
		t.Run(page.name, func(t *testing.T) {
			html := renderComponent(t, page.component)
			bodyStart := html[strings.Index(html, "<body"):]
			bodyStart = bodyStart[:strings.Index(bodyStart, ">")]
			if !strings.Contains(bodyStart, "data-authenticated-app") {
				t.Fatalf("authenticated body marker missing from %s", bodyStart)
			}
			if !strings.Contains(bodyStart, page.marker) {
				t.Fatalf("page body marker %q missing from %s", page.marker, bodyStart)
			}

			appScript := strings.Index(html, `/assets/app.js?v=`)
			if appScript < 0 || strings.Count(html, `/assets/app.js?v=`) != 1 {
				t.Fatalf("app.js must be loaded exactly once")
			}
			// The script set must match the legacy route EXACTLY. A superset check
			// passes a page that gained a script, and a stray issues.js binds
			// listeners to markup that was never written for it.
			var loaded []string
			for _, m := range regexp.MustCompile(`/assets/([a-z-]+\.js)\?v=`).FindAllStringSubmatch(html, -1) {
				if m[1] != "app.js" {
					loaded = append(loaded, m[1])
				}
			}
			want := append([]string(nil), page.extraScripts...)
			sort.Strings(want)
			sort.Strings(loaded)
			if !reflect.DeepEqual(want, loaded) {
				t.Errorf("script set mismatch: want %v, got %v", want, loaded)
			}
			for _, script := range page.extraScripts {
				if scriptIndex := strings.Index(html, "/assets/"+script+"?v="); scriptIndex >= 0 && scriptIndex < appScript {
					t.Errorf("%s loaded before app.js", script)
				}
			}
			if styleIndex := strings.Index(html, "<style>"); styleIndex >= 0 && styleIndex < appScript {
				t.Errorf("styles rendered before app.js")
			}

			if strings.Count(html, `action="/app/context"`) < 2 || !strings.Contains(html, "mobile-context-switch") {
				t.Errorf("desktop and mobile context switches must both be rendered")
			}
			if !strings.Contains(html, ".mobile-context-switch>summary{min-height:44px") {
				t.Errorf("mobile context switch lacks its 44px touch target")
			}

			if strings.Contains(html, `class="anchor-dialog unit-dialog-shell"`) {
				if !strings.Contains(html, "/assets/building-settings.js?v=") || !strings.Contains(html, "data-unit-dialog-close") {
					t.Errorf("building settings hooks rendered without their script")
				}
				if !strings.Contains(html, ".anchor-dialog:target{display:grid}") || !strings.Contains(html, ".unit-dialog-shell.is-open{display:grid}") {
					t.Errorf("unit dialog must retain both no-JS and enhanced open states")
				}
			}
		})
	}
}

func TestContactsFlashDistinguishesSuccessAndFailure(t *testing.T) {
	portal := PortalPageData{Title: "Kontakte"}
	// Keyed on ContactOK, the value the server actually computes — not inferred
	// from ContactFormOpen. Those agree across today's four statuses, which makes
	// an inference test assert a coincidence rather than the contract.
	success := renderComponent(t, ContactsPage(ContactsPageData{Portal: portal, AssetVersion: "test", ContactMessage: "Kontakt gespeichert.", ContactOK: true}))
	if !strings.Contains(success, `<p class="flash ok">Kontakt gespeichert.</p>`) || !strings.Contains(success, ".flash.ok{") {
		t.Fatal("successful contact flash lacks its styled success state")
	}
	failure := renderComponent(t, ContactsPage(ContactsPageData{Portal: portal, AssetVersion: "test", ContactMessage: "Fehler", ContactFormOpen: true}))
	if strings.Contains(failure, `<p class="flash ok">Fehler</p>`) || !strings.Contains(failure, `<p class="flash">Fehler</p>`) {
		t.Fatal("failed contact flash must not use the success state")
	}
	// A failure that leaves the form closed must still not read as success. This
	// is the case the ContactFormOpen inference got wrong.
	closedFailure := renderComponent(t, ContactsPage(ContactsPageData{Portal: portal, AssetVersion: "test", ContactMessage: "Fehler"}))
	if strings.Contains(closedFailure, `<p class="flash ok">Fehler</p>`) {
		t.Fatal("a failure with the form closed must not render as success")
	}
}

func TestMobileContextSwitchIsHiddenForOneContext(t *testing.T) {
	html := renderComponent(t, PortalPage(PortalPageData{
		Title: "Ein Portal",
		Contexts: []PortalContext{{
			TenantSlug: "park", HouseName: "Haus am Park", Role: "Bewohner", Current: true,
		}},
	}))
	if strings.Contains(html, `class="context-switch mobile-context-switch"`) || strings.Contains(html, `action="/app/context"`) {
		t.Fatal("context switch must stay hidden when there is only one context")
	}
}

func TestBaseStylesAreEmittedBeforePageStyles(t *testing.T) {
	// Order is load-bearing, and specificity does not save us. Hoisted rules sit
	// at (0,1,0) and their @media counterparts in the page block sit at (0,1,0)
	// too, so whichever comes last wins. Put the shared block after the page
	// block and rules the pages rely on start losing — in states a screenshot
	// never reaches, because they live behind a breakpoint or a closed <details>.
	html := renderComponent(t, SettingsHubPage(SettingsHubPageData{Portal: PortalPageData{Title: "Einstellungen"}}))

	base := strings.Index(html, ".house-name{")    // only PortalBaseStyles defines this
	page := strings.Index(html, ".settings-main{") // only the settings page block does
	shell := strings.Index(html, ".mobile-context-switch{")

	if base < 0 || page < 0 || shell < 0 {
		t.Fatalf("markers missing: base=%d page=%d shell=%d", base, page, shell)
	}
	if base > page {
		t.Error("PortalBaseStyles must be emitted BEFORE the page styles, so page rules keep winning")
	}
	if shell < page {
		t.Error("PortalShellStyles must stay AFTER the page styles; it exists to override them")
	}
}

func TestOnlyPortalDocumentOwnsTheDocument(t *testing.T) {
	// Rendering the right output is not the same as sharing a shell: fifteen
	// copied document shells would satisfy every other test here, and would drift
	// apart again exactly as they did the first time. So assert the structure, by
	// reading the sources.
	sources, err := filepath.Glob("*.templ")
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) < 10 {
		t.Fatalf("expected the templ sources next to this test, found %d", len(sources))
	}
	for _, source := range sources {
		if source == "portal.templ" || source == "templ_example.templ" {
			continue // portal.templ defines PortalDocument; the example is not a portal page
		}
		body, err := os.ReadFile(source)
		if err != nil {
			t.Fatal(err)
		}
		for _, owned := range []string{"<!doctype", "<html ", "<head>", "<body"} {
			if bytes.Contains(bytes.ToLower(body), []byte(owned)) {
				t.Errorf("%s builds its own %s — the document belongs to PortalDocument alone", source, owned)
			}
		}
	}
}

func TestNotificationStatusLineIsAddressableByAppJS(t *testing.T) {
	// app.js finds [data-notification-status] inside [data-notification-form] and
	// keeps the sentence current as topics are toggled. The server already renders
	// the right text; without the hook it simply freezes at its initial value.
	html := renderComponent(t, NotificationSettingsPage(NotificationSettingsPageData{
		Portal:                    PortalPageData{Title: "Benachrichtigungen"},
		EmailNotificationsEnabled: true,
		NotificationEnabledCount:  1,
		NotificationEventCount:    2,
	}))
	for _, hook := range []string{"data-notification-form", "data-notification-master", "data-notification-status", "data-notification-count"} {
		if !strings.Contains(html, hook) {
			t.Errorf("notification settings page is missing %s, so app.js cannot drive it", hook)
		}
	}
}

func TestUserCardsKeepRoleAndStatusOnNarrowScreens(t *testing.T) {
	// The legacy table stacks on a phone and drops the labels for role and status
	// on purpose, because the pills say what they are — but it keeps the VALUES.
	// Hiding the pills instead means a phone cannot tell an Admin from a resident,
	// which markup-level checks cannot see: the elements are rendered, CSS removes
	// them.
	html := renderComponent(t, UserSettingsPage(UserSettingsPageData{
		Portal:    PortalPageData{Title: "Benutzer"},
		UserCount: 1,
		Users: []view.UserRow{{
			DisplayName: "Ada Beispiel", Email: "ada@example.test",
			Initials: "AB", Role: "Admin", RoleClass: "role-admin", Status: "Aktiv",
		}},
	}))
	if !strings.Contains(html, ">Admin</span>") || !strings.Contains(html, ">Aktiv</span>") {
		t.Fatal("user card must render the role and status values")
	}
	// Match any way a rule can take the pills off the screen, not just the exact
	// shape the bug happened to use. `display:none` was how HAUSV-545 did it;
	// `!important`, `visibility:hidden` and zero sizing hide just as completely.
	hide := regexp.MustCompile(`[^{};]*\.user-card>\.pill[^{}]*\{[^{}]*(display:\s*none|visibility:\s*hidden|opacity:\s*0)[^{}]*\}`)
	for _, rule := range hide.FindAllString(html, -1) {
		t.Errorf("narrow-screen rule hides the role/status pill: %s", rule)
	}
}

func assertPageTableCoversTemplPages(t *testing.T, pages []struct {
	name         string
	component    templ.Component
	marker       string
	extraScripts []string
}) {
	t.Helper()
	want := make([]string, 0, len(pages))
	for _, page := range pages {
		want = append(want, page.name)
	}
	sort.Strings(want)

	declaration := regexp.MustCompile(`(?m)^templ ([A-Z][A-Za-z0-9]*Page)\(`)
	files, err := filepath.Glob("*.templ")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, file := range files {
		source, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, match := range declaration.FindAllSubmatch(source, -1) {
			got = append(got, string(match[1]))
		}
	}
	sort.Strings(got)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("authenticated page table is stale\nfound: %v\ntable: %v", got, want)
	}
}

func renderComponent(t *testing.T, component templ.Component) string {
	t.Helper()
	var output bytes.Buffer
	if err := component.Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	return output.String()
}
