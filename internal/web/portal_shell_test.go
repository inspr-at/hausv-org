package web

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
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
			if strings.Contains(html, "data-confirm") && appScript < 0 {
				t.Fatalf("data-confirm rendered without app.js")
			}
			for _, script := range page.extraScripts {
				scriptIndex := strings.Index(html, "/assets/"+script+"?v=")
				if scriptIndex < 0 {
					t.Errorf("required script %s missing", script)
				} else if scriptIndex < appScript {
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
	success := renderComponent(t, ContactsPage(ContactsPageData{Portal: portal, AssetVersion: "test", ContactMessage: "Kontakt gespeichert."}))
	if !strings.Contains(success, `<p class="flash ok">Kontakt gespeichert.</p>`) || !strings.Contains(success, ".flash.ok{") {
		t.Fatal("successful contact flash lacks its styled success state")
	}
	failure := renderComponent(t, ContactsPage(ContactsPageData{Portal: portal, AssetVersion: "test", ContactMessage: "Fehler", ContactFormOpen: true}))
	if strings.Contains(failure, `<p class="flash ok">Fehler</p>`) || !strings.Contains(failure, `<p class="flash">Fehler</p>`) {
		t.Fatal("failed contact flash must not use the success state")
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
	for _, rule := range regexp.MustCompile(`[^{};]+\{display:none\}`).FindAllString(html, -1) {
		if strings.Contains(rule, ".user-card>.pill") {
			t.Errorf("narrow-screen rule hides the role/status pill: %s", rule)
		}
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
