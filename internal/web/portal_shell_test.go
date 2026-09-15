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
		{"PortalErrorPage", PortalErrorPage(portal, 403, "Nicht freigegeben", "Keine Berechtigung", "", "/app", "Zum Hausüberblick", nil), "data-portal-error", nil},
		{"SupportViewPage", SupportViewPage(portal, nil), "data-templ-settings", nil},
		{"PortalPage", PortalPage(portal), "data-templ-portal", nil},
		{"PortalOrganisationPage", PortalOrganisationPage(portal, "Portfolio", nil, nil, nil, VerwaltungPlaceholder()), "data-templ-verwaltung", nil},
		{"AnnouncementsPage", AnnouncementsPage(AnnouncementsPageData{Portal: portal, AssetVersion: assetVersion, CanManageAnnouncements: true}), "data-templ-portal", []string{"announcements.js", "attachments.js"}},
		{"AuditPage", AuditPage(AuditPageData{Portal: portal}), "data-templ-audit", nil},
		{"BallotsPage", BallotsPage(BallotsPageData{Portal: portal, AssetVersion: assetVersion, CanManageVotes: true}), "data-templ-ballots", []string{"announcements.js", "attachments.js", "ballots.js"}},
		{"ContactsPage", ContactsPage(ContactsPageData{Portal: portal, AssetVersion: assetVersion, CanManageContacts: true}), "data-templ-contacts", []string{"contacts.js"}},
		{"DocumentsPage", DocumentsPage(DocumentsPageData{Portal: portal, AssetVersion: assetVersion, CanManageDocuments: true}), "data-templ-documents", []string{"announcements.js", "attachments.js"}},
		{"EnergyPage", EnergyPage(EnergyPageData{Portal: portal, CanManageEnergy: true, HasMetrics: true}), "data-templ-energy", []string{"energy-flow.js"}},
		{"EventsPage", EventsPage(EventsPageData{Portal: portal, AssetVersion: assetVersion, CanManageEvents: true}), "data-templ-events", []string{"announcements.js", "attachments.js"}},
		{"HandoversPage", HandoversPage(HandoversPageData{Portal: portal, AssetVersion: assetVersion}), "data-templ-handovers", []string{"attachments.js"}},
		{"HelpPage", HelpPage(HelpPageData{Portal: portal, ConnectorAvailable: true, ConnectorConnected: true, CanManageEnergy: true}), "data-templ-help", nil},
		{"IssueBoardPage", IssueBoardPage(IssueBoardPageData{Portal: portal, AssetVersion: assetVersion, BoardAction: "/app/anliegen/board", CanCreateIssue: true, CanManageAnnouncements: true, TotalIssueCount: 1, OpenIssueCount: 1, UrgentIssueCount: 1, Issues: []view.IssueView{{ID: "1", Title: "Kellerlicht defekt", StatusClass: "status-open"}}}), "data-templ-issue-board", []string{"attachments.js", "issues.js", "issue-board.js"}},
		{"IssueTriagePage", IssueTriagePage(IssueTriagePageData{Portal: portal, AssetVersion: assetVersion, TriageStep: "1", Issue: view.IssueView{ID: "1", Title: "Kellerlicht defekt", StatusClass: "status-open"}}), "data-templ-issue-triage", []string{"attachments.js"}},
		{"HomeSettingsPage", HomeSettingsPage(HomeSettingsPageData{Portal: portal, HouseholdName: "Dachwohnung", HomeTypeLabel: "Wohnung", HasUnitOptions: true, UnitOptions: []view.SelectOption{{Value: "1", Label: "Top 1", Selected: true}}}), "data-templ-settings", nil},
		{"IssuesPage", IssuesPage(IssuesPageData{Portal: portal, AssetVersion: assetVersion, CanCreateIssue: true, OpenIssueCreate: true}), "data-templ-issues", []string{"attachments.js", "issues.js"}},
		{"ParkingPage", ParkingPage(ParkingPageData{Portal: portal, AssetVersion: assetVersion, IsAdmin: true, CanManageParkingPayments: true, StatementYear: 2026}), "data-templ-parking", []string{"attachments.js"}},
		{"OnboardingPage", OnboardingPage(OnboardingPageData{Portal: portal, Step: 2, Progress: 40, CanControlEnergy: true}), "data-templ-onboarding", nil},
		{"SettingsHubPage", SettingsHubPage(SettingsHubPageData{Portal: portal}), "data-templ-settings", []string{"profile-picture.js"}},
		{"AnnualStatementPage", AnnualStatementPage(AnnualStatementPageData{Portal: portal, EstateName: "Haus am Park", EstateAddress: "Parkgasse 1"}), "data-templ-settings", nil},
		{"ProfileSettingsPage", ProfileSettingsPage(ProfileSettingsPageData{Portal: portal}), "data-templ-settings", nil},
		{"NotificationSettingsPage", NotificationSettingsPage(NotificationSettingsPageData{Portal: portal}), "data-templ-settings", nil},
		{"BuildingSettingsPage", BuildingSettingsPage(BuildingSettingsPageData{Portal: portal, AssetVersion: assetVersion, BuildingSection: "units", Units: []view.BuildingUnitView{{ID: "1", Label: "Top 1"}}}), "data-templ-settings", []string{"attachments.js", "building-settings.js"}},
		{"UserSettingsPage", UserSettingsPage(UserSettingsPageData{Portal: portal, AssetVersion: assetVersion, IsAdmin: true}), "data-templ-settings", []string{"users.js"}},
		{"LiegenschaftenPage", LiegenschaftenPage(portal, nil, "", "", "", 0, 1), "data-templ-liegenschaften", nil},
		{"IssueResidentDetailPage", IssueResidentDetailPage(IssueResidentDetailPageData{Portal: portal}), "data-templ-legacy", []string{"attachments.js"}},
		{"ParkingSettingsPage", ParkingSettingsPage(ParkingSettingsPageData{Portal: portal}), "data-templ-legacy", nil},
		{"ParkingMonthPage", ParkingMonthPage(ParkingMonthPageData{Portal: portal}), "data-templ-legacy", []string{"attachments.js"}},
		{"ParkingAccessSettingsPage", ParkingAccessSettingsPage(ParkingAccessSettingsPageData{Portal: portal}), "data-templ-legacy", nil},
		{"PortalModuleSettingsPage", PortalModuleSettingsPage(PortalModuleSettingsPageData{Portal: portal}), "data-templ-legacy", nil},
		{"StructuredExportPage", StructuredExportPage(StructuredExportPageData{Portal: portal}), "data-templ-legacy", nil},
		{"EnergyDataPage", EnergyDataPage(EnergyDataPageData{Portal: portal}), "data-templ-legacy", nil},
		{"EbInterfaceImportPage", EbInterfaceImportPage(EbInterfaceImportPageData{Portal: portal}), "data-templ-legacy", []string{"attachments.js"}},
		{"PaymentImportPage", PaymentImportPage(PaymentImportPageData{Portal: portal}), "data-templ-legacy", []string{"attachments.js"}},
	}

	assertPageTableCoversTemplPages(t, pages)
	for _, page := range pages {
		t.Run(page.name, func(t *testing.T) {
			html := renderComponent(t, page.component)
			if strings.Contains(html, "@DisclosureChevron") || strings.Contains(html, "⌄") {
				t.Fatal("disclosure component call or text arrow leaked into rendered HTML")
			}
			if strings.Count(html, "<style data-shell-critical>") != 1 || !strings.Contains(html, `<html lang="de-AT" style="--sidebar-w:280px;">`) {
				t.Fatal("every authenticated page needs the initial width and critical shell CSS")
			}
			critical := strings.Index(html, "<style data-shell-critical>")
			asset := strings.Index(html, `/assets/portal-shell.css?v=`)
			if asset <= critical || asset >= strings.Index(html, "</head>") {
				t.Fatal("every authenticated page must load critical shell CSS before the external sheet in the head")
			}
			var resized bytes.Buffer
			if err := page.component.Render(WithSidebarWidth(t.Context(), 360), &resized); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(resized.String(), `<html lang="de-AT" style="--sidebar-w:360px;">`) {
				t.Fatal("every authenticated page must preserve the server-provided sidebar width")
			}
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
			// The script set must match each route contract EXACTLY. A superset check
			// passes a page that gained a script, and a stray issues.js binds
			// listeners to markup that was never written for it.
			var loaded []string
			for _, m := range regexp.MustCompile(`/assets/([a-z-]+\.js)\?v=`).FindAllStringSubmatch(html, -1) {
				// app.js and switcher.js belong to the shell itself and ship with
				// every portal page; the per-page set is what must match exactly.
				if m[1] != "app.js" && m[1] != "switcher.js" && m[1] != "support-view.js" && m[1] != "product-version.js" {
					loaded = append(loaded, m[1])
				}
			}
			// The shared asset installs the scroll observer before body parsing.
			// Its existing controls still bind at DCL, after app.js.
			scrollScripts := regexp.MustCompile(`<script src="/assets/switcher\.js\?v=[^"]+"[^>]*>`).FindAllString(html, -1)
			if len(scrollScripts) != 1 {
				t.Fatal("exactly one switcher script is required")
			}
			scrollScript := scrollScripts[0]
			position := strings.Index(html, scrollScript)
			if strings.Contains(scrollScript, "defer") || strings.Contains(scrollScript, "async") || position <= asset || position >= strings.Index(html, "</head>") {
				t.Fatal("scroll observer must load synchronously after the stylesheet and before body parsing")
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

			if strings.Count(html, `data-house-picker-shell=`) != 1 || strings.Count(html, `data-context-bar`) != 1 {
				t.Errorf("one responsive header and property switcher required")
			}
			if !strings.Contains(html, "/map-tiles/17/1/2.png") || !strings.Contains(html, `class="side-map-tile"`) {
				t.Errorf("authenticated shell is missing OSM map tiles")
			}
			// The large location map follows the house copy inside the sidebar
			// (the mobile drawer, rendered earlier, carries the copy without a map).
			side := html
			if at := strings.Index(html, `<aside class="sidebar"`); at >= 0 {
				side = html[at:]
			}
			if i, j := strings.Index(side, `class="map side-map"`), strings.Index(side, `class="house-header-copy"`); i < 0 || j >= 0 {
				t.Errorf("sidebar keeps map without duplicating the header picker")
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
		{"/app/announcements", "Aushang", "Aushang erstellen", false, AnnouncementsPage(AnnouncementsPageData{Portal: portal, AssetVersion: "test", CanManageAnnouncements: true})},
		{"/app/events", "Termine", "Termin erstellen", true, EventsPage(EventsPageData{Portal: portal, AssetVersion: "test", CanManageEvents: true})},
		{"/app/kontakte", "Kontakte", "Kontakt hinzufügen", false, ContactsPage(ContactsPageData{Portal: portal, AssetVersion: "test", CanManageContacts: true})},
		{"/app/dokumente", "Dokumente", "Hochladen", false, DocumentsPage(DocumentsPageData{Portal: portal, AssetVersion: "test", CanManageDocuments: true, HasAnyDocuments: true})},
		{"/app/anliegen", "Anliegen für Bewohner", "Triage-Board", false, IssuesPage(IssuesPageData{Portal: portal, AssetVersion: "test", CanManageIssues: true})},
		{"/app/anliegen/board", "Anliegen-Board", "Kalender abonnieren", false, IssueBoardPage(IssueBoardPageData{Portal: portal, AssetVersion: "test", IsServiceProvider: true, CalendarFeedURL: "/calendar.ics"})},
		{"/app/abstimmungen", "Abstimmungen", "Abstimmung anlegen", false, BallotsPage(BallotsPageData{Portal: portal, AssetVersion: "test", CanManageVotes: true, HasBallots: true})},
		{"/app/parking", "Parkplatznutzung", "Mehr", false, ParkingPage(ParkingPageData{Portal: portal, AssetVersion: "test", StatementYear: 2026})},
		{"/app/uebergaben", "Übergaben", "Übergabe anlegen", false, HandoversPage(HandoversPageData{Portal: portal, AssetVersion: "test", HasHandovers: true})},
		{"/app/settings/users", "Benutzer & Rechte", "Person einladen", false, UserSettingsPage(UserSettingsPageData{Portal: portal, AssetVersion: "test"})},
		{"/app/audit", "Verlauf", "Einstellungen", false, AuditPage(AuditPageData{Portal: portal, AuditPageTitle: "Verlauf", AuditLede: "Änderungen nachvollziehen."})},
		{"/app/settings", "Einstellungen", "", false, SettingsHubPage(SettingsHubPageData{Portal: portal})},
		{"/app/hilfe", "Hilfe", "", false, HelpPage(HelpPageData{Portal: portal})},
		{"/app/hilfe/energie", "Hilfe", "", false, HelpPage(HelpPageData{Portal: portal, EnergyHelp: true})},
	}

	for _, page := range landings {
		t.Run(page.route, func(t *testing.T) {
			html := renderComponent(t, page.component)
			for _, marker := range []string{
				`data-portal-shell`, `data-portal-section-landing`,
				`data-portal-section-header`, `class="sidebar"`,
				`class="map side-map"`, `class="house-header-copy"`, `class="sidebar-release"`,
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
			// Header actions stay ghost buttons on every landing (portal chrome
			// rule, enforced by qa-main-flows as well); primary weight belongs to
			// the dialog or form the action opens.
			if got := strings.Count(header, `class="button primary"`); got != 0 {
				t.Errorf("primary header actions = %d, want 0", got)
			}

			if !strings.Contains(html, `class="context-role">Verwaltung</span>`) || !strings.Contains(html, `data-context-account`) {
				t.Error("the signed-in role must remain visible in the shared context bar")
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

func TestHousePickerUsesOneResponsiveID(t *testing.T) {
	portal := PortalPageData{
		Title: "Portal", TenantSlug: "park", HouseName: "Haus am Park", Address: "Parkgasse 1, 8010 Graz",
		MapURL: "https://www.openstreetmap.org/", DisplayName: "Vera Verwaltung", Initials: "VV", Role: "Admin",
		CanUseResidentAreas: true,
		Shell: PortalShellData{Ready: true, IsOrganisationMember: true, ManagedHouses: []PortalHouse{
			{Slug: "park", Name: "Haus am Park", Address: "Parkgasse 1, 8010 Graz", Role: "Admin", Current: true},
			{Slug: "see", Name: "Haus am See", Address: "Seegasse 2, 8010 Graz", Role: "Admin"},
		}},
	}
	html := renderComponent(t, PortalPage(portal))
	for _, marker := range []string{
		`id="context-property"`, `data-house-picker-shell="scope"`,
		`class="map side-map"`, `title="Parkgasse 1, 8010 Graz in OpenStreetMap öffnen"`,
	} {
		if !strings.Contains(html, marker) {
			t.Errorf("house picker marker %q missing", marker)
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
		".issues-card th:nth-child(3),.issues-card td:nth-child(3){width:auto}",
		".issues-card th{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}",
		".place-text{display:block;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}",
	} {
		if !strings.Contains(html, rule) {
			t.Errorf("dense issue table is missing bounded location rule %q", rule)
		}
	}
	if !strings.Contains(html, `class="place-text" title="`+location+`">`+location+`</span>`) {
		t.Fatal("dense issue location must retain its full value in a native tooltip")
	}
}

func TestPortalPagesDoNotOwnSharedChromeCSS(t *testing.T) {
	sources, err := filepath.Glob("*.templ")
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range sources {
		// The shared first-paint component is part of PortalDocument, not a page.
		if source == "portal.templ" || source == "portal_shell_critical.templ" || source == "templ_example.templ" {
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

func TestMobileContextSwitcherRemainsAvailableForOneContext(t *testing.T) {
	html := renderComponent(t, PortalPage(PortalPageData{
		Title: "Ein Portal",
		Contexts: []PortalContext{{
			TenantSlug: "park", HouseName: "Haus am Park", Role: "Bewohner", Current: true,
		}},
	}))
	if !strings.Contains(html, `data-switcher`) || !strings.Contains(html, `class="switcher-row current"`) || !strings.Contains(html, `disabled`) {
		t.Fatal("one context still has a switcher with the active context disabled")
	}
}

func TestBaseStylesAreEmittedBeforePageStyles(t *testing.T) {
	// Order is load-bearing, and specificity does not save us. Hoisted rules sit
	// at (0,1,0) and their @media counterparts in the page block sit at (0,1,0)
	// too, so whichever comes last wins. Put the shared block after the page
	// block and rules the pages rely on start losing — in states a screenshot
	// never reaches, because they live behind a breakpoint or a closed <details>.
	html := renderComponent(t, SettingsHubPage(SettingsHubPageData{Portal: PortalPageData{Title: "Einstellungen"}}))

	base := strings.Index(html, ".side-map{")      // only PortalBaseStyles defines this
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

// HAUSV-672: a tenant without a display name is shown under its postal address,
// so the house card would otherwise print the place twice.
func TestHouseCardDoesNotRepeatThePlaceOfAnAddressUsedAsName(t *testing.T) {
	for _, tc := range []struct{ name, address, wantTitle, wantPlace string }{
		{"Janischhofweg 22, 8043 Graz", "Janischhofweg 22, 8043 Graz", "Janischhofweg 22", "8043 Graz"},
		{"Haus am Park", "Parkgasse 1, 8010 Graz", "Haus am Park", "8010 Graz"},
		{"Parkgasse 1, 8010 Graz", "Parkgasse 1, 8010 Graz, Österreich", "Parkgasse 1, 8010 Graz", "Österreich"},
		{"8010 Graz", "8010 Graz", "8010 Graz", "8010 Graz"},
	} {
		if got := portalHouseTitle(tc.name, tc.address); got != tc.wantTitle {
			t.Errorf("portalHouseTitle(%q, %q) = %q, want %q", tc.name, tc.address, got, tc.wantTitle)
		}
		if got := portalHousePlace(tc.address); got != tc.wantPlace {
			t.Errorf("portalHousePlace(%q) = %q, want %q", tc.address, got, tc.wantPlace)
		}
	}
	portal := PortalPageData{
		Title: "Portal", TenantSlug: "jhw22", HouseName: "Janischhofweg 22, 8043 Graz", Address: "Janischhofweg 22, 8043 Graz",
		MapURL: "https://www.openstreetmap.org/", DisplayName: "Markus", Initials: "MB", Role: "Admin",
		Shell: PortalShellData{Ready: true},
	}
	html := renderComponent(t, PortalPage(portal))
	if !strings.Contains(html, `<strong>Janischhofweg 22</strong><small>8043 Graz</small>`) {
		t.Errorf("house card should show the street above the place once, got %q", between(html, `class="house-header-copy"`, `</span>`))
	}
	if !strings.Contains(html, `.house-header-copy strong{display:block;overflow:hidden;overflow-wrap:normal;text-overflow:ellipsis;white-space:nowrap`) {
		t.Errorf("sidebar house title must stay on one line and ellipsize long addresses")
	}
	if !strings.Contains(html, `title="Janischhofweg 22"`) {
		t.Errorf("sidebar house title must remain available in full when truncated")
	}
}

func between(s, start, end string) string {
	i := strings.Index(s, start)
	if i < 0 {
		return ""
	}
	rest := s[i:]
	if j := strings.Index(rest, end); j >= 0 {
		return rest[:j]
	}
	return rest
}

// The location map is the first sidebar block, before organisation and navigation.
func TestSidebarStartsWithLocationMapBeforeNavigation(t *testing.T) {
	portal := PortalPageData{
		Title: "Portal", TenantSlug: "park", HouseName: "Haus am Park", Address: "Parkgasse 1, 8010 Graz",
		MapURL: "https://www.openstreetmap.org/", DisplayName: "Vera Verwaltung", Initials: "VV", Role: "Admin",
		CanUseResidentAreas: true,
		Map:                 PortalMap{Configured: true, Tiles: []PortalMapTile{{URL: "/map-tiles/17/1/2.png", Style: "left:calc(50% + 0px);top:calc(50% + 0px)"}}},
		Shell:               PortalShellData{Ready: true},
	}
	html := renderComponent(t, PortalPage(portal))
	side := html[strings.Index(html, `<aside class="sidebar"`):]
	hero, card := strings.Index(side, `class="side-map-hero"`), strings.Index(side, `class="house-header-card`)
	label := strings.Index(side, `class="nav-group nav-level-label nav-house-label"`)
	navigation := strings.Index(side, `class="nav-house-items`)
	if label < 0 || card != -1 || hero < 0 || label <= hero || navigation <= hero {
		t.Fatalf("sidebar must show map, label, navigation without a duplicate picker (label=%d card=%d hero=%d navigation=%d)", label, card, hero, navigation)
	}
	if strings.Count(html, `class="side-map-hero`) != 2 {
		t.Errorf("desktop sidebar and mobile drawer must each carry the hero, found %d", strings.Count(html, `class="side-map-hero`))
	}
	if strings.Contains(html, "house-map-thumb") {
		t.Errorf("the compact house card must not carry the old thumbnail")
	}
	for _, marker := range []string{`class="map side-map"`, `title="Parkgasse 1, 8010 Graz in OpenStreetMap öffnen"`, `class="side-map-pin"`, `.side-map-hero{position:relative;min-width:0;height:208px;margin:8px calc(var(--sidebar-pad-x,18px)*-1) 8px`} {
		if !strings.Contains(html, marker) {
			t.Errorf("hero marker %q missing", marker)
		}
	}
	portal.Map = PortalMap{}
	html = renderComponent(t, PortalPage(portal))
	if !strings.Contains(html, `class="side-map-hero side-map-hero-empty"`) || !strings.Contains(html, "Standort nicht hinterlegt") {
		t.Errorf("without coordinates the hero collapses to the placeholder band")
	}
}

// Grouping must retain the handler's order inside each status and never drop
// issues, including a newly introduced status not yet in the filter options.
func TestIssueBoardColumnsPreserveIssuesAndStatusOrder(t *testing.T) {
	options := []view.SelectOption{{Value: "", Label: "Alle Status"}, {Value: "Neu"}, {Value: "In Bearbeitung"}, {Value: "Erledigt"}}
	issues := []view.IssueView{
		{ID: "first", Title: "Erste Meldung", Status: "Neu", Age: "vor 2 T."},
		{ID: "working", Title: "Arbeit läuft", Status: "In Bearbeitung"},
		{ID: "second", Title: "Zweite Meldung", Status: "Neu"},
		{ID: "future", Title: "Weiterer Status", Status: "Wartet"},
	}
	columns := issueBoardColumns(options, issues)
	var statuses []string
	var ids [][]string
	for _, column := range columns {
		statuses = append(statuses, column.Status)
		var lane []string
		for _, issue := range column.Issues {
			lane = append(lane, issue.ID)
		}
		ids = append(ids, lane)
	}
	if !reflect.DeepEqual(statuses, []string{"Neu", "In Bearbeitung", "Erledigt", "Wartet"}) ||
		!reflect.DeepEqual(ids, [][]string{{"first", "second"}, {"working"}, nil, {"future"}}) {
		t.Fatalf("board lost status order or issues: statuses=%v ids=%v", statuses, ids)
	}
	body := renderComponent(t, IssueBoardBody(IssueBoardPageData{
		Issues: issues, TotalIssueCount: len(issues), Filters: view.IssueBoardFilterView{StatusOptions: options},
	}))
	for _, issue := range issues {
		if strings.Count(body, `id="issue-`+issue.ID+`"`) != 1 || !strings.Contains(body, `href="/app/anliegen/board/`+issue.ID+`/panel"`) {
			t.Errorf("issue %s must appear exactly once and keep its edit action", issue.ID)
		}
	}
	if !strings.Contains(body, `vor 2 T.`) || !strings.Contains(body, `Hierher ziehen`) {
		t.Error("board must show actual age and distinguish an empty status")
	}
}

func TestResidentOverviewUsesFullWidthAndOptionalOrganisationBlock(t *testing.T) {
	portal := PortalPageData{Title: "Portal", HouseName: "Park", Role: "Eigentümer", CanUseResidentAreas: true,
		Modules: PortalModules{Issues: true, Events: true, Announcements: true},
		Issues:  []view.IssueView{{Title: "Testanliegen", Location: "Eigene Einheit · Top 1"}},
		Shell:   PortalShellData{Ready: true},
	}
	body := renderComponent(t, PortalPage(portal))
	if !strings.Contains(body, `<span class="issue-location">Eigene Einheit · Top 1</span>`) {
		t.Fatal("unit label must remain a single item")
	}
	calm := renderComponent(t, PortalCalm(portal))
	issues, updates, events, announcements := strings.Index(calm, `class="module calm-issues"`), strings.Index(calm, `class="calm-updates"`), strings.Index(calm, `class="module events"`), strings.Index(calm, `class="module announcements"`)
	if issues < 0 || updates <= issues || events <= updates || announcements <= events {
		t.Fatal("expected issues left, then events and announcements in shared right column")
	}
	for _, organisation := range []bool{false, true} {
		portal.Shell.IsOrganisationMember = organisation
		html := renderComponent(t, PortalSidebar(portal))
		names := []string{}
		for _, match := range regexp.MustCompile(`data-navigation-block="([^"]+)"`).FindAllStringSubmatch(html, -1) {
			names = append(names, match[1])
		}
		want := []string{"map", "house-navigation", "release"}
		if organisation {
			want = []string{"map", "organisation-identity", "organisation", "house-navigation", "release"}
		}
		if !reflect.DeepEqual(names, want) {
			t.Fatalf("organisation=%v blocks=%v want=%v", organisation, names, want)
		}
		if !organisation && strings.Contains(html, `class="portal-organisation-identity"`) {
			t.Fatal("empty organisation header rendered")
		}
	}
	css, err := os.ReadFile("assets/portal-shell.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`.portal-home-landing.calm-main .portal-section-content{width:100%;margin-inline:0;padding-inline:var(--home-inset)}`,
		`@media(min-width:1100px){.calm-column .disclosures{grid-template-columns:minmax(0,1fr) minmax(0,1fr)}}`,
		`.calm-issues .row .issue-location{grid-area:location;min-width:0;white-space:normal;overflow-wrap:anywhere;line-height:1.4}`,
		`.nav[data-two-level="false"]>.nav-house-label{margin-top:0;padding-top:0;border-top:0}`,
	} {
		if !strings.Contains(string(css), want) {
			t.Errorf("missing resident layout contract %s", want)
		}
	}
}
