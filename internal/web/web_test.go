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
		`.side-version { justify-self: end; min-height: 44px; }`,
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
	if got := strings.Count(PageTemplates, `{{template "designTokens" .}}`); got != 4 {
		t.Fatalf("design token partial is used %d times, want home, landing, privacy and app styles", got)
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
	if got := strings.Count(PageTemplates, `data-dialog="announcement-create"`); got != 1 {
		t.Fatalf("announcement create should have one entry point, got %d", got)
	}
	for _, want := range []string{`class="announcement-body"`, `class="dialog-optional full"`, `Aushang veröffentlichen`} {
		if !strings.Contains(PageTemplates, want) {
			t.Fatalf("announcement flow missing progressive-disclosure marker %q", want)
		}
	}
	if got := strings.Count(PageTemplates, `data-dialog="event-create"`); got != 1 {
		t.Fatalf("event create should have one entry point, got %d", got)
	}
	for _, want := range []string{`class="calendar-subscription"`, `class="event-history"`, `Termin veröffentlichen`, `Ende, Details oder Anhang`} {
		if !strings.Contains(PageTemplates, want) {
			t.Fatalf("event flow missing progressive-disclosure marker %q", want)
		}
	}
	if got := strings.Count(PageTemplates, `data-dialog="document-upload"`); got != 1 {
		t.Fatalf("document upload should have one entry point, got %d", got)
	}
	for _, want := range []string{`class="document-toolbar"`, `class="document-actions"`, `class="document-admin-tools"`, `Dokument veröffentlichen`, `Auf eine Einheit begrenzen`, `Keine Dokumente gefunden`} {
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

func TestAppShellLoadsSharedSubmitGuard(t *testing.T) {
	if !strings.Contains(PageTemplates, `<script src="/assets/app.js?v={{.AssetVersion}}" defer></script>`) {
		t.Fatal("app shell must load the shared submit guard")
	}
	if strings.Contains(PageTemplates, `<a class="side-mark" href="/app">WEG</a>`) || strings.Contains(PageTemplates, `<span class="landing-mark">HV</span>`) || strings.Contains(PageTemplates, `<span class="mark">WEG</span>`) || strings.Contains(PageTemplates, `{{template "hausvMark" .}}`) || strings.Contains(PageTemplates, `class="logo-dot"`) {
		t.Fatal("app shell should not use the old WEG/HV text or dot placeholder logo")
	}
	for _, want := range []string{
		`<div class="side-map" role="group" aria-label="Fester Kartenausschnitt für {{.Tenant.Address}}">`,
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
	for _, want := range []string{`data-mail-local`, `mailto:`, `encodeURIComponent`} {
		if !strings.Contains(landingText, want) {
			t.Fatalf("landing contact script missing %q", want)
		}
	}
	attachmentJS, err := os.ReadFile("assets/attachments.js")
	if err != nil {
		t.Fatalf("read attachment script: %v", err)
	}
	attachmentText := string(attachmentJS)
	for _, want := range []string{`DataTransfer`, `attachment-picker-item`, `Datei entfernen`, `Wird beim Speichern hochgeladen`} {
		if !strings.Contains(attachmentText, want) {
			t.Fatalf("attachment picker script missing %q", want)
		}
	}

	issueJS, err := os.ReadFile("assets/issues.js")
	if err != nil {
		t.Fatalf("read issue wizard script: %v", err)
	}
	issueText := string(issueJS)
	for _, want := range []string{`data-issue-step`, `reportValidity`, `suggestedTitle`, `data-issue-summary`, `scrollIntoView`} {
		if !strings.Contains(issueText, want) {
			t.Fatalf("issue wizard script missing %q", want)
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
		`Rechte und „Nur beobachten“ bleiben unverändert.`,
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

func TestIssueCreationUsesThreeFocusedSteps(t *testing.T) {
	for _, want := range []string{
		`<script src="/assets/issues.js?v={{.AssetVersion}}" defer></script>`,
		`data-issue-wizard`,
		`data-issue-step="1"`,
		`Schritt 1 von 3`,
		`Was ist passiert?`,
		`Foto hinzufügen`,
		`data-issue-step="2"`,
		`Schritt 2 von 3`,
		`Wo ist es?`,
		`data-issue-step="3"`,
		`Schritt 3 von 3`,
		`Stimmt alles?`,
		`data-issue-summary="body"`,
		`Wird aus Ihrer Beschreibung vorgeschlagen und kann geändert werden.`,
	} {
		if !strings.Contains(PageTemplates, want) {
			t.Fatalf("issue creation flow missing %q", want)
		}
	}
	if strings.Contains(PageTemplates, `Ort genauer angeben oder Datei anhängen`) {
		t.Fatal("issue creation should not mix location and attachments in one disclosure")
	}
}
