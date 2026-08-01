package web

// These tests assert on the template string and on the asset files themselves,
// so they live with the package that owns them: Go resolves a test's relative
// paths from its package directory, which is where assets/ now is.

import (
	"os"
	"strings"
	"testing"
)

func TestPageTemplatesConsolidateDesignTokensAndComponents(t *testing.T) {
	wants := []string{
		`{{define "designTokens"}}`,
		`{{template "designTokens" .}}`,
		`--font-sans:`,
		`--space-6:24px`,
		`--radius-sm:8px`,
		`--shadow-panel:0 12px 30px rgba(32,37,31,.04)`,
		`Shared components: panel, button, pill, quick-row, table-wrap, dialog, flash and empty-state.`,
		`.panel { background: var(--panel); border: 1px solid var(--line); border-radius: var(--radius-sm); padding: var(--space-6); box-shadow: var(--shadow-panel); }`,
		`.empty-state { border: 1px solid var(--line); border-radius: var(--radius-sm);`,
		`.access-row { display: grid; grid-template-columns: minmax(220px, 1fr) 126px 132px minmax(106px, auto);`,
		`aria-label="Parkplatz-Verwaltung"`,
		`.mobile-menu-toggle { min-height: 44px;`,
		`.side-version { justify-self: end; min-width: 44px; min-height: 44px; display: grid; place-items: center; }`,
		`.logout-button { min-height: 44px; }`,
	}
	for _, want := range wants {
		if !strings.Contains(PageTemplates, want) {
			t.Fatalf("PageTemplates missing shared design-system marker %q", want)
		}
	}
	if got := strings.Count(PageTemplates, "--ink:#20251f"); got != 1 {
		t.Fatalf("color token block is duplicated %d times, want once", got)
	}
	if got := strings.Count(PageTemplates, `{{template "designTokens" .}}`); got != 5 {
		t.Fatalf("design token partial is used %d times, want home, landing, privacy, app styles and the error page", got)
	}
	if got := strings.Count(PageTemplates, `<span class="nav-label">`); got != 13 {
		t.Fatalf("navigation labels are wrapped inconsistently: got %d, want 13", got)
	}
}

func TestPageTemplatesExposeAccessibilityConventions(t *testing.T) {
	wants := []string{
		`:where(a, button, input, select, textarea, summary, [tabindex]):focus-visible`,
		`aria-haspopup="dialog" aria-controls="announcement-create"`,
		`<dialog id="announcement-create" class="dialog" aria-labelledby="announcement-create-title">`,
		`aria-haspopup="dialog" aria-controls="event-create"`,
		`<dialog id="event-create" class="dialog" aria-labelledby="event-create-title">`,
		`aria-describedby="role-help"`,
		`<span id="role-help" class="popup" role="tooltip">`,
		`aria-label="E-Mail-Adresse" autocomplete="email" required`,
		`aria-label="Nachricht an Bewohner"`,
		`aria-label="Ihre Antwort"`,
		`Keine Gesundheitsdaten, Ausweiskopien oder unnötig abgebildete Personen`,
		`aria-label="Parkplatzbereiche"`,
		`aria-label="Monatsabrechnungen"`,
		`aria-label="Ältere Monate"`,
		`aria-label="Aktueller Ladezustand Parkplatz 20"`,
	}
	for _, want := range wants {
		if !strings.Contains(PageTemplates, want) {
			t.Fatalf("PageTemplates missing accessibility convention %q", want)
		}
	}
	for _, path := range []string{"assets/announcements.js", "assets/users.js"} {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(body)
		for _, want := range []string{"dialogTriggers", `aria-expanded`, "focusFirstDialogField"} {
			if !strings.Contains(text, want) {
				t.Fatalf("%s missing dialog focus convention %q", path, want)
			}
		}
	}
	// Two deliberate triggers: the toolbar button that is present in every state
	// and the primary action inside the empty state, so a house without a single
	// notice is not a dead end. More than that would be an accidental duplicate.
	if got := strings.Count(PageTemplates, `data-dialog="announcement-create"`); got != 2 {
		t.Fatalf("announcement create should have the toolbar and empty-state entry point, got %d", got)
	}
	for _, want := range []string{`class="announcement-body"`, `class="dialog-optional full"`, `Aushang veröffentlichen`} {
		if !strings.Contains(PageTemplates, want) {
			t.Fatalf("announcement flow missing progressive-disclosure marker %q", want)
		}
	}
	if got := strings.Count(PageTemplates, `data-dialog="event-create"`); got != 2 {
		t.Fatalf("event create should have the toolbar and empty-state entry point, got %d", got)
	}
	// The calendar feed is a one-click action, so it stays visible in the side
	// column instead of hiding behind a collapsed disclosure.
	for _, want := range []string{`class="events-aside"`, `Kalender abonnieren`, `class="event-history"`, `Termin veröffentlichen`, `Ende, Details oder Anhang`} {
		if !strings.Contains(PageTemplates, want) {
			t.Fatalf("event flow missing progressive-disclosure marker %q", want)
		}
	}
	if got := strings.Count(PageTemplates, `data-dialog="document-upload"`); got != 1 {
		t.Fatalf("document upload should have one entry point, got %d", got)
	}
	for _, want := range []string{`class="document-toolbar"`, `class="document-actions"`, `class="document-admin-tools"`, `class="dialog-footer"><button class="button primary" type="submit">Hochladen`, `Optional: bestimmte Einheit`, `Keine Dokumente gefunden`} {
		if !strings.Contains(PageTemplates, want) {
			t.Fatalf("document flow missing hierarchy/progressive-disclosure marker %q", want)
		}
	}
	if got := strings.Count(PageTemplates, `data-dialog="ballot-create"`); got != 1 {
		t.Fatalf("ballot create should have one entry point, got %d", got)
	}
	for _, want := range []string{`class="vote-overview {{.VoteOverviewClass}}"`, `Ihre Stimme ist gefragt`, `Ihre Stimme zählt:`, `Details zur Abstimmung`, `Abstimmungsregeln`, `Entwurf anlegen`, `{{template "ballotResult" .}}`} {
		if !strings.Contains(PageTemplates, want) {
			t.Fatalf("ballot flow missing hierarchy/progressive-disclosure marker %q", want)
		}
	}
}

func TestAuthenticatedAppShellIsKeyboardOperable(t *testing.T) {
	for _, want := range []string{
		`<a class="skip-link" href="#main-content">Zum Inhalt springen</a>`,
		`<button class="mobile-menu-toggle" type="button" data-mobile-menu-toggle`,
		`aria-controls="portal-navigation portal-account"`,
		`aria-expanded="false" aria-label="Navigation öffnen"`,
		`<nav id="portal-navigation" class="side-nav">`,
		`<div id="portal-account" class="side-foot">`,
		`<noscript><style>`,
		`.sidebar.nav-open .side-nav, .sidebar.nav-open .side-foot { display: grid; }`,
	} {
		if !strings.Contains(PageTemplates, want) {
			t.Fatalf("authenticated app shell missing keyboard convention %q", want)
		}
	}
	if strings.Contains(PageTemplates, `class="nav-toggle"`) {
		t.Fatal("authenticated app shell must not use a hidden checkbox as its menu control")
	}
	appOpens := strings.Count(PageTemplates, `{{template "appOpen" .}}`)
	mainTargets := strings.Count(PageTemplates, `id="main-content" tabindex="-1" class="app-main`)
	appCloses := strings.Count(PageTemplates, `{{template "appClose" .}}`)
	if appOpens != 27 || mainTargets != appOpens || appCloses != appOpens {
		t.Fatalf("authenticated templates must each have one skip target: opens=%d targets=%d closes=%d", appOpens, mainTargets, appCloses)
	}

	body, err := os.ReadFile("assets/app.js")
	if err != nil {
		t.Fatalf("read app shell behavior: %v", err)
	}
	text := string(body)
	for _, want := range []string{
		`function setMobileMenuOpen(open, options)`,
		`mobileMenuToggle.setAttribute("aria-expanded"`,
		`mobileNavigation.hidden`,
		`mobileAccount.hidden`,
		`event.key !== "Escape"`,
		`setMobileMenuOpen(false, { returnFocus: true })`,
		`mobileMenuQuery.addEventListener("change"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("shared app behavior missing mobile-menu convention %q", want)
		}
	}
}

func TestAdminDialogsKeepActionsVisibleAtShortHeights(t *testing.T) {
	for _, want := range []string{
		`.dialog[open] { display: grid; grid-template-rows: auto minmax(0,1fr) auto; }`,
		`.dialog > form { min-width: 0; min-height: 0; grid-column: 1; grid-row: 1 / -1;`,
		`.dialog-body { min-width: 0; min-height: 0;`,
		`.dialog-footer { position: relative; z-index: 2;`,
		`.dialog-close { width: 44px; height: 44px;`,
		`<form id="{{.EditDialogID}}-form" method="post" action="/app/kontakte">`,
		`class="dialog-footer contact-dialog-footer"`,
		`form="{{.EditDialogID}}-form">Änderungen speichern`,
		`class="dialog-footer handover-dialog-submit"`,
	} {
		if !strings.Contains(PageTemplates, want) {
			t.Fatalf("bounded admin-dialog contract missing %q", want)
		}
	}
	if strings.Contains(PageTemplates, `.dialog form {`) {
		t.Fatal("shared dialog layout must not turn nested action forms into dialog shells")
	}
	handoverBody := strings.Index(PageTemplates, `<div class="dialog-body">
            <p class="document-dialog-intro">`)
	handoverFooter := strings.Index(PageTemplates, `<div class="dialog-footer handover-dialog-submit">`)
	if handoverBody < 0 || handoverFooter < handoverBody {
		t.Fatal("handover submit footer must follow its independently scrolling dialog body")
	}
}

func TestAdminPagesConstrainKnownResponsiveMinContent(t *testing.T) {
	for _, want := range []string{
		`.announce .announce-feed { grid-template-columns: minmax(0,1fr); gap: 14px; }`,
		`.announce .entry-head { display: grid; grid-template-columns: minmax(0,1fr); }`,
		`@media (min-width: 761px) and (max-width: 1120px)`,
		`.building .home-profile-context > p, .building .home-profile-context > .button { grid-column: 2; }`,
		`.building .home-profile-context > .button { justify-self: start; }`,
	} {
		if !strings.Contains(PageTemplates, want) {
			t.Fatalf("responsive admin-page constraint missing %q", want)
		}
	}
}

func TestResidentContentFlowsStayCompactAndProgressivelyDisclosed(t *testing.T) {
	for _, want := range []string{
		`.guide-disclosure > summary { min-height: 44px;`,
		`.guide-disclosure[open] > summary::after { transform: rotate(90deg); }`,
		`@media (prefers-reduced-motion: reduce) { .guide-disclosure > summary::after { transition: none; } }`,
		`class="panel compact announce-aside-panel guide-disclosure"`,
		`class="panel compact events-aside-panel guide-disclosure"`,
		`class="panel compact contacts-aside-panel guide-disclosure"`,
		`@media (max-width: 900px) and (min-width: 721px)`,
		`.announce .filter-form { grid-template-columns: minmax(0,1fr) auto; align-items: end; }`,
		`.announce .announcement-entry h3 { overflow-wrap: anywhere; }`,
		`.announce .announcement-entry h3 { font-size: 18px; line-height: 1.18; }`,
		`.announce .filter-form .button, .announce .filter-tab, .announce .announcement-body > summary, .announce .entry-actions .button { min-height: 44px; }`,
		`.events-page .event-details > summary, .events-page .event-history > summary { min-height: 44px;`,
		`.vote-actions .button, .vote-management-actions .button, .vote-result-actions .button { min-height: 44px;`,
		`class="audit-help-disclosure guide-disclosure"`,
		`<summary><strong>Einträge verstehen</strong></summary>`,
		`min-height: clamp(280px,34vh,360px)`,
		`class="vote-readonly-note"`,
		`class="button ghost" href="/app/settings">Einstellungen`,
	} {
		if !strings.Contains(PageTemplates, want) {
			t.Fatalf("resident content UX contract missing %q", want)
		}
	}
	if got := strings.Count(PageTemplates, `min-height: clamp(280px,34vh,360px)`); got != 3 {
		t.Fatalf("resident empty-state height is constrained %d times, want announcements, events and contacts", got)
	}
}

func TestSettingsAndParkingKeepReducedMobileInteractionContracts(t *testing.T) {
	for _, want := range []string{
		`<a class="side-map side-address" href="{{.MapURL}}"`,
		`<a class="side-place-copy" href="/app" aria-label="Hausportal für {{.SidebarAddress.Full}} öffnen">`,
		`side-address-short`,
		`@media (max-width: 350px)`,
		`min-height: 44px !important;`,
		`.payment-import .apply-bar { position: static;`,
		`settings-guide-details`,
		`<details class="account-details profile-visibility-details">`,
		`<details class="notification-details">`,
		`Tarif, zwei Zählerstände und ein Preis genügen.`,
		`<strong>Messwerte</strong>`,
		`<strong>Nur Nachweis.</strong>`,
		`access-fixed`,
		`data-help="Fest vergeben · hier nicht änderbar"`,
		`min-width: 44px; min-height: 44px;`,
		`.users .dlg-x button { width: 44px; height: 44px;`,
		`.users .edit-dialog h2 { padding-right: 48px; font-size: 23px; }`,
		`<span class="file-control"><span>Datei auswählen</span><input id="camt-file"`,
	} {
		if !strings.Contains(PageTemplates, want) {
			t.Fatalf("settings/parking mobile contract missing %q", want)
		}
	}
	if strings.Contains(PageTemplates, `<details class="account-details" open>`) {
		t.Fatal("profile account metadata should use progressive disclosure")
	}
}

func TestHandoverKeepsComfortableMobileTouchTargets(t *testing.T) {
	for _, want := range []string{
		`.attachment-delete button { position: relative; width: 44px; height: 44px;`,
		`.attachment-delete button::before { content: "\00d7";`,
		`.handover-add-files .button, .handover-detail-actions .button { min-height: 44px;`,
	} {
		if !strings.Contains(PageTemplates, want) {
			t.Fatalf("handover touch-target contract missing %q", want)
		}
	}
}

func TestAppShellLoadsSharedSubmitGuard(t *testing.T) {
	if !strings.Contains(PageTemplates, `<script src="/assets/app.js?v={{.AssetVersion}}" defer></script>`) {
		t.Fatal("app shell must load the shared submit guard")
	}
	if strings.Contains(PageTemplates, `<a class="side-mark" href="/app">WEG</a>`) || strings.Contains(PageTemplates, `<span class="landing-mark">HV</span>`) || strings.Contains(PageTemplates, `<span class="mark">WEG</span>`) || strings.Contains(PageTemplates, `{{template "hausvMark" .}}`) || strings.Contains(PageTemplates, `class="logo-dot"`) {
		t.Fatal("app shell should not use the old WEG/HV text or dot placeholder logo")
	}
	for _, want := range []string{
		`<a class="side-map side-address" href="{{.MapURL}}" target="_blank" rel="noopener noreferrer" aria-label="{{.Tenant.Address}} in OpenStreetMap öffnen"`,
		`<svg class="side-map-pin-shape" viewBox="0 0 44 56" focusable="false">`,
		`<span class="side-map-pin-mark">{{template "tenantBrandMark" .}}</span>`,
		`{{define "hausvLandingMark"}}`,
		`{{define "hausvPlatformMark"}}`,
		`{{template "tenantBrandMark" .}}`,
		`rel="icon" type="image/svg+xml" href="/favicon.svg"`,
		`data-dialog="release-history"`,
		`Versionsverlauf`,
		`v{{.DisplayVersion}}`,
	} {
		if !strings.Contains(PageTemplates, want) {
			t.Fatalf("app shell missing logo/fav icon convention %q", want)
		}
	}
	landingMarkStart := strings.Index(PageTemplates, `{{define "hausvLandingMark"}}`)
	if landingMarkStart < 0 {
		t.Fatal("simple public landing mark template is missing")
	}
	landingMarkEnd := strings.Index(PageTemplates[landingMarkStart:], `{{end}}`)
	if landingMarkEnd < 0 {
		t.Fatal("simple public landing mark template is incomplete")
	}
	landingMark := PageTemplates[landingMarkStart : landingMarkStart+landingMarkEnd]
	if strings.Contains(landingMark, "mark-frame") || strings.Contains(landingMark, "<text") {
		t.Fatal("public landing mark should contain only the three houses")
	}
	if strings.Contains(PageTemplates, `inset: 0 0 0 34%`) || !strings.Contains(PageTemplates, `.home-hero::before { content: ""; position: absolute; z-index: -2; inset: 0;`) {
		t.Fatal("home overview hero image should span the full header width")
	}
	body, err := os.ReadFile("assets/app.js")
	if err != nil {
		t.Fatalf("read submit guard: %v", err)
	}
	text := string(body)
	for _, want := range []string{`dataset.submitting`, `Bitte warten`, `dataset.confirm`, `setTimeout`, `data-notification-form`, `email-paused`, `data-notification-count`, `data-home-type-select`, `data-home-type-explanation`, `dataset.description`, `data-energy-chart-interactive`, `data-chart-tooltip`, `ArrowLeft`, `ArrowRight`, `requestFullscreen`, `fullscreenchange`, `is-fullscreen-fallback`} {
		if !strings.Contains(text, want) {
			t.Fatalf("submit guard missing %q", want)
		}
	}
	for _, want := range []string{`data-dialog="energy-chart-dialog"`, `{{.Chart.DialogTitle}}`, `data-chart-hit`, `data-chart-marker-index`, `zeitraum=heute`, `data-energy-fullscreen`, `data-energy-fullscreen-surface`} {
		if !strings.Contains(PageTemplates, want) {
			t.Fatalf("energy chart interaction missing %q", want)
		}
	}
	landingJS, err := os.ReadFile("assets/landing.js")
	if err != nil {
		t.Fatalf("read landing script: %v", err)
	}
	landingText := string(landingJS)
	for _, want := range []string{`data-mail-local`, `mailto:`, `encodeURIComponent`, `data-landing-menu-toggle`, `aria-expanded`, `closeMenu`, `matchMedia`} {
		if !strings.Contains(landingText, want) {
			t.Fatalf("landing contact script missing %q", want)
		}
	}
	for _, want := range []string{
		`data-landing-menu-toggle aria-expanded="false" aria-controls="landing-navigation"`,
		`<noscript><style>`,
		`@media (prefers-reduced-motion: reduce)`,
		`<h1>Alles, was Zuhause anfällt.</h1>`,
		`<strong>Energie verstehen</strong>`,
		`<h2>{{if .Sent}}E-Mail prüfen`,
		`<details class="login-retry">`,
		`>Weiter zum Portal</a>`,
		`<script src="/assets/home.js?v={{.AssetVersion}}" defer></script>`,
		`class="location-map-tile" data-map-tile="{{.URL}}"`,
		`Fester Kartenausschnitt rund um {{.Tenant.Address}}`,
		`© OpenStreetMap`,
		`Anmeldeseite und in der Portalnavigation`,
		`Erst beim bewussten Öffnen des Kartenlinks`,
		`← Zurück zur Startseite`,
	} {
		if !strings.Contains(PageTemplates, want) {
			t.Fatalf("public/auth flow missing %q", want)
		}
	}
	if strings.Contains(PageTemplates, `class="landing-nav-toggle"`) || strings.Contains(PageTemplates, `Lokalen Testzugang öffnen`) {
		t.Fatal("public/auth flow should not retain the checkbox menu or development-heavy action copy")
	}
	homeJS, err := os.ReadFile("assets/home.js")
	if err != nil {
		t.Fatalf("read public home script: %v", err)
	}
	for _, want := range []string{`location-map-configured`, `data-map-tile`, `map-tile-failed`, `new Image()`} {
		if !strings.Contains(string(homeJS), want) {
			t.Fatalf("public location fallback script missing %q", want)
		}
	}
	if strings.Contains(PageTemplates, `onerror=`) {
		t.Fatal("public map fallback must not depend on CSP-blocked inline handlers")
	}
	attachmentJS, err := os.ReadFile("assets/attachments.js")
	if err != nil {
		t.Fatalf("read attachment script: %v", err)
	}
	attachmentText := string(attachmentJS)
	for _, want := range []string{`DataTransfer`, `attachment-picker-item`, `Datei entfernen`, `Wird beim Speichern hochgeladen`, `name.title`} {
		if !strings.Contains(attachmentText, want) {
			t.Fatalf("attachment picker script missing %q", want)
		}
	}

	issueJS, err := os.ReadFile("assets/issues.js")
	if err != nil {
		t.Fatalf("read issue wizard script: %v", err)
	}
	issueText := string(issueJS)
	for _, want := range []string{`data-issue-step`, `form.noValidate`, `suggestedTitle`, `data-issue-summary`, `scrollIntoView`, `popstate`, `prefers-reduced-motion`, `Bitte beschreiben Sie kurz`} {
		if !strings.Contains(issueText, want) {
			t.Fatalf("issue wizard script missing %q", want)
		}
	}
}

func TestHomeIdentityUXSeparatesFriendlyNameFromOfficialUnits(t *testing.T) {
	for _, want := range []string{
		`{{define "homeIdentitySettings"}}`,
		`href="/app/settings/home?from=energy"`,
		`href="/app/settings/home?from=building"`,
		`Anzeigename für „Mein Zuhause“`,
		`Offizielle Bezeichnung`,
		`Zugeordnete offizielle Wohnung`,
		`Die Sichtbarkeit folgt dieser Wohnung`,
		`aria-describedby="home-settings-unit-help"`,
		`data-home-identity="nav"`,
		`data-home-identity="energy-heading"`,
		`data-home-identity="settings"`,
		`data-home-identity="building-context"`,
		`data-home-identity="building-unit"`,
		`data-home-identity="onboarding-summary"`,
		`data-home-display-name`,
		`data-home-unit-label`,
		`{{if .CanManageHomeIdentity}}`,
	} {
		if !strings.Contains(PageTemplates, want) {
			t.Fatalf("home identity UX missing %q", want)
		}
	}
}

func TestHomeTypeGuidanceAndSidebarBrandHierarchy(t *testing.T) {
	for _, want := range []string{
		`data-home-type-select`,
		`id="home-type-explanation"`,
		`data-description="Ein einzelner Haushalt in einem Mehrparteienhaus.`,
		`data-description="Ein Haushalt mit eigenem Gebäude.`,
		`data-description="Mehrere Parteien und gemeinsam genutzte Anlagen.`,
		`Die Auswahl kann Geltungsbereich und Sichtbarkeit ändern.`,
		`„Nur beobachten“ bleibt unverändert.`,
		`.side-map { position: relative; width: 100%; height: 210px;`,
		`.side-map-pin-mark svg { width: 25px; height: 21px;`,
		`Kartendaten © OpenStreetMap`,
		`text-decoration: none;`,
		`Hausportal</strong><span>· hausv.org`,
	} {
		if !strings.Contains(PageTemplates, want) {
			t.Fatalf("home-type/sidebar polish missing %q", want)
		}
	}
}

func TestEnergyLiveCardUsesIndependentIconsAndAccessibleMotion(t *testing.T) {
	for _, want := range []string{
		`{{define "energyMetricIcon"}}`,
		`energy-metric-icon load`,
		`energy-metric-icon pv`,
		`energy-metric-icon grid-import`,
		`energy-metric-icon grid-export`,
		`energy-battery-visual {{.Live.Battery.Direction}}`,
		`data-energy-direction="{{.Live.Battery.Direction}}"`,
		`@media (prefers-reduced-motion: reduce)`,
		`.energy-flow-item:last-child:nth-child(odd)`,
	} {
		if !strings.Contains(PageTemplates, want) {
			t.Fatalf("energy live-card polish missing %q", want)
		}
	}
}

func TestEnergyGeometryKeepsSafetyAndLiveFlowFirst(t *testing.T) {
	for _, want := range []string{
		`--soft:#716d62; --gold-ink:#705c22; --energy-focus-ring:#ad862c;`,
		`.app-shell { --mobile-nav-height:68px; display: block; }`,
		`top: var(--mobile-nav-height); grid-template-columns: 30px minmax(0,1fr) auto;`,
		`grid-template-columns: minmax(0,1fr); align-content: start;`,
		`energy-mode-action-compact`,
		`@media (max-width: 1279px)`,
		`grid-template-columns: minmax(0,1fr) minmax(440px,440px);`,
		`Liest und empfiehlt. Keine Steuerung.`,
		`Freigabe nur für Eigentümer oder Hausadministration`,
		`action="/app/energie/mode"`,
		`Sofort zurück zu „Nur beobachten“`,
		`Testlauf bewusst starten`,
	} {
		if !strings.Contains(PageTemplates, want) {
			t.Fatalf("energy geometry/safety slice missing %q", want)
		}
	}

	cockpitStart := strings.Index(PageTemplates, `{{define "energyCockpit"}}`)
	if cockpitStart < 0 {
		t.Fatal("energy cockpit template is missing")
	}
	cockpit := PageTemplates[cockpitStart:]
	lead := strings.Index(cockpit, `{{template "energyLead" .}}`)
	tariff := strings.Index(cockpit, `class="energy-card energy-tariff"`)
	if lead < 0 || tariff < 0 || lead >= tariff {
		t.Fatalf("energy cockpit source order is not live/next before tariff: lead=%d tariff=%d", lead, tariff)
	}

	// This slice changes hierarchy only; the legally/product-relevant tariff
	// qualifications must remain in the rendered source until its focused route
	// is implemented separately.
	for _, caveat := range []string{
		`Das ist nicht Ihre Stromrechnung.`,
		`Arbeitspreis, Energiekosten, Abgaben und Steuern`,
		`Niedertarif-Fenster (SNAP, WiNAP)`,
		`Diesen Stand festhalten`,
	} {
		if !strings.Contains(cockpit, caveat) {
			t.Fatalf("energy hierarchy slice removed tariff caveat %q", caveat)
		}
	}
}

func TestIssueCreationUsesTwoFocusedSteps(t *testing.T) {
	for _, want := range []string{
		`<script src="/assets/issues.js?v={{.AssetVersion}}" defer></script>`,
		`data-issue-wizard`,
		`data-issue-step="describe"`,
		`Schritt 1 von 2`,
		`Was ist passiert?`,
		`Foto oder Datei hinzufügen`,
		`data-issue-step="review"`,
		`Schritt 2 von 2`,
		`Prüfen &amp; senden`,
		`data-issue-summary="body"`,
		`data-issue-review-expand`,
		`Titel ändern`,
		`data-busy-label="Meldung wird gesendet…"`,
		`class="wizard-exit" href="/app">Abbrechen</a>`,
		`Akute Gefahr? 112 anrufen.`,
		`href="/app/kontakte">Hauskontakte</a>`,
		`class="issue-resident-report"`,
		`.issue-resident-report .attachment-delete button { position: relative; width: 44px; height: 44px;`,
		`Neuigkeiten zum Anliegen`,
	} {
		if !strings.Contains(PageTemplates, want) {
			t.Fatalf("issue creation flow missing %q", want)
		}
	}
	if strings.Contains(PageTemplates, `Ort genauer angeben oder Datei anhängen`) {
		t.Fatal("issue creation should not mix location and attachments in one disclosure")
	}
	if strings.Contains(PageTemplates, `name="title" maxlength="140" required`) || strings.Contains(PageTemplates, `Schritt 3 von 3`) {
		t.Fatal("issue creation must keep the title optional and stop after two focused steps")
	}
}
