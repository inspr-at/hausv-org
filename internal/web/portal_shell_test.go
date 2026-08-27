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
		HeroImageURL:           "/assets/hausv-landing-hero.png",
		BrandIcon:              "single-home",
		BrandMarkSVG:           `<svg class="hausv-mark tenant-brand-mark" viewBox="0 0 64 48" aria-hidden="true" focusable="false"><path d="M12 35h40"/><path d="M16 35V22.5L32 11l16 11.5V35"/><path d="M26.5 35v-9h11v9"/><path d="M21.5 27.5h5M37.5 27.5h5"/></svg>`,
		Map:                    PortalMap{Configured: true, Tiles: []PortalMapTile{{URL: "/map-tiles/17/1/2.png", Style: "left:calc(50% + 0.00px);top:calc(50% + 0.00px)"}}},
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
		{"EnergyPage", EnergyPage(EnergyPageData{Portal: portal, CanManageEnergy: true, HasMetrics: true}), "data-templ-energy", []string{"energy-flow.js"}},
		{"EventsPage", EventsPage(EventsPageData{Portal: portal, AssetVersion: assetVersion, CanManageEvents: true}), "data-templ-events", []string{"announcements.js", "attachments.js"}},
		{"HandoversPage", HandoversPage(HandoversPageData{Portal: portal, AssetVersion: assetVersion}), "data-templ-handovers", []string{"attachments.js"}},
		{"HelpPage", HelpPage(HelpPageData{Portal: portal, ConnectorAvailable: true, ConnectorConnected: true, CanManageEnergy: true}), "data-templ-help", nil},
		{"IssueBoardPage", IssueBoardPage(IssueBoardPageData{Portal: portal, AssetVersion: assetVersion, BoardAction: "/app/anliegen/board", CanCreateIssue: true, CanManageAnnouncements: true, TotalIssueCount: 1, OpenIssueCount: 1, UrgentIssueCount: 1, Issues: []view.IssueView{{ID: "1", Title: "Kellerlicht defekt", StatusClass: "status-open"}}}), "data-templ-issue-board", []string{"attachments.js", "issues.js"}},
		{"IssueTriagePage", IssueTriagePage(IssueTriagePageData{Portal: portal, AssetVersion: assetVersion, TriageStep: "1", Issue: view.IssueView{ID: "1", Title: "Kellerlicht defekt", StatusClass: "status-open"}}), "data-templ-issue-triage", []string{"attachments.js"}},
		{"HomeSettingsPage", HomeSettingsPage(HomeSettingsPageData{Portal: portal, HouseholdName: "Dachwohnung", HomeTypeLabel: "Wohnung", HasUnitOptions: true, UnitOptions: []view.SelectOption{{Value: "1", Label: "Top 1", Selected: true}}}), "data-templ-settings", nil},
		{"IssuesPage", IssuesPage(IssuesPageData{Portal: portal, AssetVersion: assetVersion, CanCreateIssue: true, OpenIssueCreate: true}), "data-templ-issues", []string{"attachments.js", "issues.js"}},
		{"ParkingPage", ParkingPage(ParkingPageData{Portal: portal, AssetVersion: assetVersion, IsAdmin: true, CanManageParkingPayments: true, StatementYear: 2026}), "data-templ-parking", []string{"attachments.js"}},
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
			if !strings.Contains(html, "/map-tiles/17/1/2.png") || !strings.Contains(html, `class="side-map-tile"`) {
				t.Errorf("authenticated shell is missing OSM map tiles")
			}
			// Context switch is now inside the map overlay (.side-place-copy)
			if i, j := strings.Index(html, `class="side-map`), strings.Index(html, `class="side-place-copy"`); i < 0 || j < 0 || i >= j {
				t.Errorf("place copy overlay must come after the map anchor")
			}
			if !strings.Contains(html, `class="side-map-pin-mark"`) {
				t.Errorf("map pin is missing the brand mark")
			}
			if !strings.Contains(html, `<link rel="stylesheet" href="/assets/portal-shell.css?v=`) {
				t.Error("portal shell CSS link is missing from authenticated pages")
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

func TestPrimaryNavigationLandingsUseSharedChromeKit(t *testing.T) {
	portal := PortalPageData{
		Title: "Portal", HouseName: "Haus am Park", Address: "Parkgasse 1",
		GreetingName: "Ada", Today: "Donnerstag, 20. August", DisplayName: "Ada Beispiel",
		Initials: "AB", Role: "Verwaltung", HeroImageURL: "/assets/hausv-landing-hero.png",
		MapURL: "https://www.openstreetmap.org/", Dense: true,
		CanUseResidentAreas: true, CanViewEnergy: true, CanManageIssues: true,
		CanCreateResidentIssue: true, CanSeeParking: true, CanManageHandovers: true,
		CanManageUsers: true, CanViewAudit: true,
		Modules: PortalModules{
			Energy: true, Announcements: true, Events: true, Contacts: true,
			Documents: true, Issues: true, Votes: true, Parking: true,
			Handovers: true, Users: true, Audit: true, Help: true,
		},
		Map: PortalMap{Configured: true, Tiles: []PortalMapTile{{
			URL: "/map-tiles/17/1/2.png", Style: "left:0;top:0",
		}}},
		Contexts: []PortalContext{
			{TenantSlug: "park", HouseName: "Haus am Park", Role: "Verwaltung", Current: true},
			{TenantSlug: "see", HouseName: "Haus am See", Role: "Bewohner"},
		},
	}

	type landing struct {
		route, name, action string
		hero                bool
		component           templ.Component
	}
	landings := []landing{
		{"/app", "Hausüberblick", "Anliegen melden", true, PortalPage(portal)},
		{"/app/energie", "Mein Zuhause", "Zuhause bearbeiten", false, EnergyPage(EnergyPageData{Portal: portal, HouseholdName: "Dachwohnung", HomeTypeLabel: "Wohnung", CanManageHomeIdentity: true})},
		{"/app/announcements", "Aushang", "Aushang erstellen", true, AnnouncementsPage(AnnouncementsPageData{Portal: portal, AssetVersion: "test", CanManageAnnouncements: true})},
		{"/app/events", "Termine", "Termin erstellen", true, EventsPage(EventsPageData{Portal: portal, AssetVersion: "test", CanManageEvents: true})},
		{"/app/kontakte", "Kontakte", "Kontakt hinzufügen", false, ContactsPage(ContactsPageData{Portal: portal, AssetVersion: "test", CanManageContacts: true})},
		{"/app/dokumente", "Dokumente", "Hochladen", false, DocumentsPage(DocumentsPageData{Portal: portal, AssetVersion: "test", CanManageDocuments: true, HasAnyDocuments: true})},
		{"/app/anliegen", "Anliegen für Bewohner", "Triage-Board", false, IssuesPage(IssuesPageData{Portal: portal, AssetVersion: "test", CanManageIssues: true})},
		{"/app/anliegen/board", "Anliegen-Board", "Kalender abonnieren", false, IssueBoardPage(IssueBoardPageData{Portal: portal, AssetVersion: "test", IsServiceProvider: true, CalendarFeedURL: "/calendar.ics"})},
		{"/app/abstimmungen", "Abstimmungen", "Abstimmung anlegen", false, BallotsPage(BallotsPageData{Portal: portal, AssetVersion: "test", CanManageVotes: true, HasBallots: true})},
		{"/app/parking", "Parkplatznutzung", "Mehr", false, ParkingPage(ParkingPageData{Portal: portal, AssetVersion: "test", StatementYear: 2026})},
		{"/app/uebergaben", "Übergaben", "Übergabe anlegen", false, HandoversPage(HandoversPageData{Portal: portal, AssetVersion: "test", HasHandovers: true})},
		{"/app/settings/users", "Benutzer & Rechte", "", false, UserSettingsPage(UserSettingsPageData{Portal: portal, AssetVersion: "test"})},
		{"/app/audit", "Verlauf", "Einstellungen", false, AuditPage(AuditPageData{Portal: portal, AuditPageTitle: "Verlauf", AuditLede: "Änderungen nachvollziehen."})},
		{"/app/settings", "Einstellungen", "", false, SettingsHubPage(SettingsHubPageData{Portal: portal})},
		{"/app/hilfe", "Hilfe", "", false, HelpPage(HelpPageData{Portal: portal})},
	}

	for _, page := range landings {
		t.Run(page.route, func(t *testing.T) {
			html := renderComponent(t, page.component)
			for _, marker := range []string{
				`data-portal-shell`, `data-portal-section-landing`,
				`data-portal-section-header`, `class="sidebar"`,
				`class="side-map`, `class="side-address-label"`, `class="account"`,
			} {
				if !strings.Contains(html, marker) {
					t.Errorf("%s is missing shared chrome marker %q", page.name, marker)
				}
			}
			if got := strings.Count(html, `data-portal-section-hero`); got != boolInt(page.hero) {
				t.Errorf("shared hero count = %d, want %d", got, boolInt(page.hero))
			}
			if !strings.Contains(html, `data-portal-hero="`+portalBool(page.hero)+`"`) {
				t.Errorf("hero contract is not declared as %t", page.hero)
			}

			header := portalTestElement(html, `data-portal-section-header`, "</header>")
			if page.action != "" && !strings.Contains(header, page.action) {
				t.Errorf("header action %q is missing", page.action)
			}
			if strings.Contains(header, `class="button primary"`) {
				t.Errorf("header action is filled; header actions must use outline .button")
			}

			mobile := portalTestElement(html, `class="mobile-head"`, "</header>")
			for _, role := range []string{"Verwaltung</small>", "Bewohner</small>"} {
				if strings.Contains(mobile, role) {
					t.Errorf("mobile chrome exposes role %q", role)
				}
			}
			if !strings.Contains(html, `<footer class="account"`) || !strings.Contains(html, `<small>Verwaltung</small>`) {
				t.Error("the signed-in role must remain in the account footer")
			}
		})
	}

	home := renderComponent(t, PortalPage(portal))
	if !strings.Contains(home, `<link rel="stylesheet" href="/assets/portal-shell.css?v=`) {
		t.Error("hero landings must link to the external portal shell CSS (which ensures flush start without inherited top padding)")
	}
	energy := renderComponent(t, EnergyPage(EnergyPageData{Portal: portal}))
	for _, contract := range []string{
		".energy-mode-strip{container-type:inline-size;container-name:energy-strip;",
		"display:flex;flex-wrap:wrap;",
		"@container energy-strip (max-width:920px)",
	} {
		if !strings.Contains(energy, contract) {
			t.Errorf("energy strip lost HAUSV-558 contract %q", contract)
		}
	}
}

func TestSharedSectionTitlesLeaveRoomForDescenders(t *testing.T) {
	portal := PortalPageData{
		Title: "Portal", GreetingName: "Peggy", Dense: true,
		HeroImageURL:    "/assets/hausv-landing-hero.png",
		CanManageIssues: true,
	}
	pages := []struct {
		name  string
		title string
		page  templ.Component
	}{
		{"home", "Hallo Peggy.", PortalPage(portal)},
		{"announcements", "Aushang", AnnouncementsPage(AnnouncementsPageData{Portal: portal})},
		{"events", "Termine", EventsPage(EventsPageData{Portal: portal})},
		{"issues", "Anliegen", IssuesPage(IssuesPageData{Portal: portal})},
		{"issue-board", "Anliegen bearbeiten", IssueBoardPage(IssueBoardPageData{Portal: portal})},
	}

	// The descender rule is now in the external portal-shell.css file (HAUSV-549)
	const descenderRule = ".portal-section-title h1{min-width:0;max-width:100%;margin:0;overflow:hidden;font-family:var(--font-serif);font-size:42px;font-weight:600;line-height:1.08;padding-bottom:.08em;"

	// Verify the external CSS file contains the descender rule
	cssContent, err := os.ReadFile("assets/portal-shell.css")
	if err != nil {
		t.Fatalf("Failed to read portal-shell.css: %v", err)
	}
	if !strings.Contains(string(cssContent), descenderRule) {
		t.Fatal("portal-shell.css must contain the descender rule for section titles")
	}

	for _, page := range pages {
		t.Run(page.name, func(t *testing.T) {
			html := renderComponent(t, page.page)
			// Verify the CSS is linked
			if !strings.Contains(html, `<link rel="stylesheet" href="/assets/portal-shell.css?v=`) {
				t.Fatal("shared section title CSS must be linked via portal-shell.css")
			}
			if !strings.Contains(html, page.title) {
				t.Fatalf("shared section header is missing title %q", page.title)
			}
		})
	}
}

func TestDenseIssueLocationsShareSpaceAndStayBounded(t *testing.T) {
	const location = "Gemeinschaft · Heizungsraum, Tiefenbohrung unter der Wiese"
	html := renderComponent(t, PortalPage(PortalPageData{
		Title: "Portal", Dense: true, Modules: PortalModules{Issues: true},
		Issues: []view.IssueView{{
			ID: "issue-1", Status: "Neu", Title: "Wärmepumpe prüfen",
			Location: location, DetailURL: "/app/anliegen/issue-1",
		}},
	}))

	for _, rule := range []string{
		".issues-card table{table-layout:fixed}",
		".issues-card th:nth-child(2),.issues-card td:nth-child(2),.issues-card th:nth-child(3),.issues-card td:nth-child(3){width:auto}",
		".place-text{display:-webkit-box;overflow:hidden;overflow-wrap:normal;word-break:normal;hyphens:auto;-webkit-box-orient:vertical;-webkit-line-clamp:2}",
	} {
		if !strings.Contains(html, rule) {
			t.Errorf("dense issue table is missing bounded location rule %q", rule)
		}
	}
	if !strings.Contains(html, `class="place-text" title="`+location+`">`+location+`</span>`) {
		t.Fatal("dense issue location must retain its full value in a native tooltip")
	}
}

func TestPortalPhoneCompositionOverridesTheSharedDesktopHideRule(t *testing.T) {
	html := renderComponent(t, PortalPage(PortalPageData{Title: "Portal"}))
	for _, rule := range []string{
		".portal-home-landing .mobile-content{display:block",
		".portal-home-landing .thumb-zone{position:fixed",
	} {
		if !strings.Contains(html, rule) {
			t.Fatalf("portal phone composition is missing its page-scoped display rule %q", rule)
		}
	}
	css, err := os.ReadFile("assets/portal-shell.css")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(css), ".portal-home-landing.dense-main .portal-section-content,.portal-home-landing.calm-main .portal-section-content{width:100%;padding:0}") {
		t.Fatal("shared shell CSS does not remove desktop composition padding at the phone breakpoint")
	}
}

func TestPortalPagesDoNotOwnSharedChromeCSS(t *testing.T) {
	sources, err := filepath.Glob("*.templ")
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range sources {
		if source == "portal.templ" || source == "templ_example.templ" {
			continue
		}
		body, err := os.ReadFile(source)
		if err != nil {
			t.Fatal(err)
		}
		for _, clone := range []string{
			".home-hero", ".page-head", ".page-top", ".page-heading", ".crumb",
			".lede{", ".page-actions", ".shell{", ".sidebar{", ".mobile-head{",
			".mobile-identity",
		} {
			if bytes.Contains(body, []byte(clone)) {
				t.Errorf("%s still owns shared chrome selector %s", source, clone)
			}
		}
	}
}

func portalTestElement(html, marker, closing string) string {
	markerAt := strings.Index(html, marker)
	if markerAt < 0 {
		return ""
	}
	start := strings.LastIndex(html[:markerAt], "<")
	end := strings.Index(html[markerAt:], closing)
	if start < 0 || end < 0 {
		return ""
	}
	return html[start : markerAt+end+len(closing)]
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
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

	base := strings.Index(html, ".side-brand{")    // only PortalBaseStyles defines this
	page := strings.Index(html, ".settings-main{") // only the settings page block does
	shellLink := strings.Index(html, `<link rel="stylesheet" href="/assets/portal-shell.css?v=`)

	if base < 0 || page < 0 || shellLink < 0 {
		t.Fatalf("markers missing: base=%d page=%d shellLink=%d", base, page, shellLink)
	}
	if base > page {
		t.Error("PortalBaseStyles must be emitted BEFORE the page styles, so page rules keep winning")
	}
	// The portal shell CSS is now an external file loaded via <link> tag.
	// The link tag must be AFTER inline page styles to ensure the shell rules win (HAUSV-563).
	// This mimics the old PortalShellStyles() position.
	if shellLink < page {
		t.Error("portal-shell.css link must be AFTER inline page styles (to win in cascade), but still in <head>")
	}
}

func TestEnergyNavCarriesTheHouseholdIdentity(t *testing.T) {
	// The page about a household should say which household. Legacy rendered the
	// name and unit here; the conversion replaced both with a static label, and
	// qa-main-flows.mjs asserts the pair — which is how it was found, one CI run
	// after templ became the default.
	html := renderComponent(t, PortalPage(PortalPageData{
		Title: "Portal", CanUseResidentAreas: true, CanViewEnergy: true,
		Modules: PortalModules{Energy: true},
		HomeIdentity: view.HomeIdentityView{
			DisplayName: "Haus Musterweg", UnitLabel: "Top 4",
			AriaLabel: "Haus Musterweg, Top 4", HasDisplayName: true, HasUnit: true,
		},
	}))
	for _, want := range []string{
		`data-home-identity="nav"`, "data-home-display-name", "data-home-unit-label",
		"Haus Musterweg", "Top 4",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("energy nav entry is missing %q", want)
		}
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
