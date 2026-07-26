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
		`.parking-access .access-table, .parking-access .access-table tbody, .parking-access .access-table tr, .parking-access .access-table td { display: block; width: 100%; min-width: 0; }`,
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
		`aria-label="Kommentar oder Rückfrage"`,
		`Keine Gesundheitsdaten, Ausweiskopien oder unnötig abgebildete Personen`,
		`aria-label="Abrechnung in 2 Schritten.`,
		`aria-label="Abrechnungsassistent"`,
		`aria-label="Abrechnungsschritte"`,
		`aria-label="Zahlungsdetails"`,
		`role="region" aria-label="Stundenwerte Parkplatznutzung"`,
		`<caption class="sr-only">Stundenwerte Parkplatznutzung`,
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
		`<a class="side-mark" href="/app" aria-label="{{if .IsServiceProvider}}Anliegen{{else}}Hausüberblick{{end}}">`,
		`{{template "hausvLandingMark" .}}`,
		`{{define "hausvPlatformMark"}}`,
		`{{template "tenantBrandMark" .}}`,
		`side-code`,
		`class="mark-word"`,
		`hausv.org</text>`,
		`rel="icon" type="image/svg+xml" href="/favicon.svg"`,
		`data-dialog="release-history"`,
		`Versionsverlauf`,
		`v{{.DisplayVersion}}`,
	} {
		if !strings.Contains(PageTemplates, want) {
			t.Fatalf("app shell missing logo/fav icon convention %q", want)
		}
	}
	if strings.Contains(PageTemplates, `inset: 0 0 0 34%`) || !strings.Contains(PageTemplates, `.home-hero::before { content: ""; position: absolute; inset: 0;`) {
		t.Fatal("home overview hero image should span the full header width")
	}
	body, err := os.ReadFile("assets/app.js")
	if err != nil {
		t.Fatalf("read submit guard: %v", err)
	}
	text := string(body)
	for _, want := range []string{`dataset.submitting`, `Bitte warten`, `dataset.confirm`, `setTimeout`} {
		if !strings.Contains(text, want) {
			t.Fatalf("submit guard missing %q", want)
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
}
