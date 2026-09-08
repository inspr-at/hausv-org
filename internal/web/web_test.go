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
	}
	for _, want := range wants {
		if !strings.Contains(PageTemplates, want) {
			t.Fatalf("PageTemplates missing shared design-system marker %q", want)
		}
	}
	if got := strings.Count(PageTemplates, "--ink:#20251f"); got != 1 {
		t.Fatalf("color token block is duplicated %d times, want once", got)
	}
	if got := strings.Count(PageTemplates, `{{template "designTokens" .}}`); got != 7 {
		t.Fatalf("design token partial is used %d times, want home, landing, HAUSV Home start, imprint, privacy, app styles and the error page", got)
	}
	// HAUSV-705: the public templates no longer own authenticated navigation.
	if strings.Contains(PageTemplates, `<span class="nav-label">`) {
		t.Fatal("authenticated navigation returned to public templates")
	}
}

// The detailed behavior checks for these routes live beside the surviving templ
// components. This string-template oracle now guards the actual migration
// boundary: converted portal pages must not creep back into PageTemplates.
func TestPageTemplatesExcludeConvertedPortalDefinitions(t *testing.T) {
	for _, name := range []string{
		"portal", "contacts", "issues", "issueTriage", "announcements", "events",
		"documents", "ballots", "handovers", "parking", "help", "settingsHub",
		"homeIdentitySettings", "auditLog", "buildingSettings", "profileSettings",
		"notificationSettings", "userSettings", "homeOnboarding", "energyCockpit",
		"issueResidentDetail", "parkingSettings", "parkingMonth", "parkingAccessSettings",
		"portalModuleSettings", "structuredExport", "energyData", "ebInterfaceImport", "paymentImport",
		"appOpen", "appClose", "sidebar", "releaseHistoryDialog",
	} {
		if strings.Contains(PageTemplates, `{{define "`+name+`"}}`) {
			t.Errorf("converted portal template %q returned to PageTemplates", name)
		}
	}
}

// HAUSV-705: keyboard semantics now come from the native details menu in
// PortalShell; the deleted legacy JS toggle is no longer a rendered control.
func TestAuthenticatedAppShellIsKeyboardOperable(t *testing.T) {
	html := renderComponent(t, ParkingSettingsPage(ParkingSettingsPageData{Portal: legacyShellTestPortal()}))
	for _, want := range []string{
		`<a class="skip-link" href="#main-content">Zum Inhalt springen</a>`,
		`<main id="main-content" tabindex="-1"`,
		`class="context-bar mobile-head"`, `data-context-bar`,
		`<details class="menu"><summary aria-label="Navigation öffnen">`,
		`class="menu-panel"`, `<aside class="sidebar" aria-label="Navigation der Liegenschaft" data-navigation-surface="sidebar">`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("shared shell missing keyboard convention %q", want)
		}
	}
	if strings.Count(html, `id="main-content"`) != 1 {
		t.Fatal("page needs exactly one skip target")
	}
	if strings.Contains(html, `class="nav-toggle"`) || strings.Contains(html, `data-mobile-menu-toggle`) {
		t.Fatal("shared shell must use the native, keyboard-operable details menu")
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
	shell := renderComponent(t, ParkingSettingsPage(ParkingSettingsPageData{Portal: legacyShellTestPortal()}))
	if !strings.Contains(shell, `<script src="/assets/app.js?v=`) {
		t.Fatal("app shell must load the shared submit guard")
	}
	if strings.Contains(PageTemplates, `<a class="side-mark" href="/app">WEG</a>`) || strings.Contains(PageTemplates, `<span class="landing-mark">HV</span>`) || strings.Contains(PageTemplates, `<span class="mark">WEG</span>`) || strings.Contains(PageTemplates, `{{template "hausvMark" .}}`) || strings.Contains(PageTemplates, `class="logo-dot"`) {
		t.Fatal("app shell should not use the old WEG/HV text or dot placeholder logo")
	}
	for _, want := range []string{
		`{{define "hausvLandingMark"}}`,
		`{{define "hausvPlatformMark"}}`,
		`{{template "tenantBrandMark" .}}`,
		`rel="icon" type="image/svg+xml" href="/favicon.svg"`,
	} {
		if !strings.Contains(PageTemplates, want) {
			t.Fatalf("app shell missing logo/fav icon convention %q", want)
		}
	}
	for _, want := range []string{`class="map side-map"`, `target="_blank"`, `rel="noopener noreferrer"`, `class="side-map-pin-shape"`, `class="side-map-pin-mark"`, `data-dialog="release-history"`, `Versionsverlauf`, `vtest`} {
		if !strings.Contains(shell, want) {
			t.Fatalf("shared shell missing map/version convention %q", want)
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
		`data-landing-menu-toggle aria-expanded="false" aria-controls="landing-navigation"><svg viewBox="0 0 24 24"`,
		`<noscript><style>`,
		`@media (prefers-reduced-motion: reduce)`,
		`<h1>Ein Hausportal. Alles, was Menschen und Gebäude verbindet.</h1>`,
		`<h3>HAUSV Free</h3>`,
		`<h3>HAUSV Home</h3>`,
		`<h3>HAUSV Professional</h3>`,
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
