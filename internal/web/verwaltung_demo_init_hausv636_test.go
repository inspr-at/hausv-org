package web

import (
	"strings"
	"testing"
)

// HAUSV-636: "Demodaten initialisieren" is one confirmation. With JavaScript the
// settings card opens a native <dialog>; without it the card's link leads to a
// single confirmation page. Both end in the same one POST.

func TestSettingsDemoCardOpensOneDialog(t *testing.T) {
	html := renderComponent(t, VerwaltungSettingsPage(organisationPortalFixture(PortalPageData{Organisation: VerwaltungShell{OrganisationName: "Musterstadt"}}), VerwaltungSettingsData{
		Threshold: 90, AIProvider: "environment", DemoResetAvailable: true,
	}))
	for _, want := range []string{
		"<h2>Demodaten initialisieren</h2>",
		`<a class="settings-secondary" href="/app/verwaltung/einstellungen/demo" data-demo-init-open aria-haspopup="dialog" aria-controls="demo-init-dialog">Demodaten initialisieren …</a>`,
		`<dialog class="demo-init-dialog" id="demo-init-dialog" aria-labelledby="demo-init-dialog-title">`,
		`<form method="post" action="/app/verwaltung/einstellungen/demo">`,
		`<h2 id="demo-init-dialog-title">Demodaten jetzt initialisieren?</h2>`,
		`<button class="settings-save" type="submit">Ja, initialisieren</button>`,
		`<button class="settings-secondary" type="button" data-close-dialog>Abbrechen</button>`,
		`.demo-init-dialog::backdrop`,
		`/assets/demo-init.js?v=`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("settings page missing %q", want)
		}
	}
	for _, gone := range []string{"Demo zurücksetzen", "ZURÜCKSETZEN", "confirm_word", `name="step"`, "<script>", `data-dialog="demo-init-dialog"`} {
		if strings.Contains(html, gone) {
			t.Errorf("settings page still carries %q", gone)
		}
	}
	if strings.Count(html, "<dialog") != 1 {
		t.Errorf("exactly one dialog expected, got %d", strings.Count(html, "<dialog"))
	}
	if i, j := strings.Index(html, "data-demo-init-open"), strings.Index(html, `id="demo-init-dialog"`); i < 0 || j < 0 || i > j {
		t.Error("the card's link must precede the dialog it opens")
	}
	if i, j := strings.Index(html, "/assets/app.js?v="), strings.Index(html, "/assets/demo-init.js?v="); i < 0 || j < 0 || i > j {
		t.Error("demo-init.js must load after app.js, which closes dialogs and returns focus")
	}
}

func TestSettingsWithoutDemoLoadsNoDemoScript(t *testing.T) {
	html := renderComponent(t, VerwaltungSettingsPage(organisationPortalFixture(PortalPageData{Organisation: VerwaltungShell{OrganisationName: "Musterstadt"}}), VerwaltungSettingsData{
		Threshold: 90, AIProvider: "environment",
	}))
	for _, gone := range []string{"demo-init.js", "<dialog", "Demodaten initialisieren", "data-demo-init-open"} {
		if strings.Contains(html, gone) {
			t.Errorf("settings page without demo still renders %q", gone)
		}
	}
	// The other verwaltung pages keep the plain shell: no page script at all.
	if got := strings.Count(html, "<script"); got != 2 {
		t.Errorf("expected only app.js and switcher.js, got %d script tags", got)
	}
}

func TestDemoConfirmationPageIsOneFormWithoutJavaScript(t *testing.T) {
	html := renderComponent(t, VerwaltungDemoResetPage(organisationPortalFixture(PortalPageData{Organisation: VerwaltungShell{OrganisationName: "Musterstadt"}}), VerwaltungDemoResetData{}))
	for _, want := range []string{
		"<title>Demodaten initialisieren · Musterstadt</title>",
		"<h1>Demodaten initialisieren</h1>",
		`<form class="demo-reset-card" method="post" action="/app/verwaltung/einstellungen/demo">`,
		"Demodaten jetzt initialisieren?",
		`<button class="demo-reset-primary" type="submit">Ja, initialisieren</button>`,
		`<a class="demo-reset-cancel" href="/app/verwaltung/einstellungen">Abbrechen</a>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("confirmation page missing %q", want)
		}
	}
	for _, gone := range []string{"confirm_word", `name="step"`, "ZURÜCKSETZEN", "Weiter", "<dialog", "demo-init.js", `type="checkbox"`} {
		if strings.Contains(html, gone) {
			t.Errorf("confirmation page still carries %q", gone)
		}
	}
	// The shell carries its own forms (context switch, sign-out); the page itself
	// posts to the demo route exactly once.
	if got := strings.Count(html, `action="/app/verwaltung/einstellungen/demo"`); got != 1 {
		t.Errorf("one confirmation form expected, got %d", got)
	}
	// The dialog and the page word the same question, so they cannot drift apart.
	settings := renderComponent(t, VerwaltungSettingsPage(organisationPortalFixture(PortalPageData{Organisation: VerwaltungShell{OrganisationName: "Musterstadt"}}), VerwaltungSettingsData{DemoResetAvailable: true}))
	for _, shared := range []string{"Demodaten jetzt initialisieren?", "Ja, initialisieren", "Ihre Anmeldung bleibt gültig."} {
		if !strings.Contains(settings, shared) || !strings.Contains(html, shared) {
			t.Errorf("dialog and page must share %q", shared)
		}
	}
}

func TestDemoResultPageNamesTheAnchorAndCounts(t *testing.T) {
	html := renderComponent(t, VerwaltungDemoResetPage(organisationPortalFixture(PortalPageData{Organisation: VerwaltungShell{OrganisationName: "Musterstadt"}}), VerwaltungDemoResetData{
		Done: true, Anchor: "06.09.2026", Duration: "1.2s",
		Statuses:   []VerwaltungDemoResetCount{{Label: "open", Count: 3}},
		Categories: []VerwaltungDemoResetCount{{Label: "Reparatur/Mangel", Count: 4}},
	}))
	for _, want := range []string{"Demodaten wurden initialisiert", "Ankerdatum: 06.09.2026 · Dauer: 1.2s", "<span>open</span><strong>3</strong>", "<span>Reparatur/Mangel</span><strong>4</strong>", "Zum Posteingang"} {
		if !strings.Contains(html, want) {
			t.Errorf("result page missing %q", want)
		}
	}
	if strings.Contains(html, `action="/app/verwaltung/einstellungen/demo"`) {
		t.Error("result page must not offer another submit")
	}
}

func TestDemoInitScriptOpensTheDialogFromTheLink(t *testing.T) {
	scriptBytes, err := Assets.ReadFile("assets/demo-init.js")
	if err != nil {
		t.Fatalf("demo-init.js lesen: %v", err)
	}
	script := string(scriptBytes)
	for _, want := range []string{
		`querySelector("[data-demo-init-open]")`, `getElementById("demo-init-dialog")`,
		"typeof dialog.showModal", "event.preventDefault()", "dialog.showModal()", "dialog._returnFocus = trigger",
		`"aria-expanded"`,
	} {
		if !strings.Contains(script, want) {
			t.Errorf("demo-init.js contract missing %q", want)
		}
	}
	// Closing belongs to app.js ([data-close-dialog]); the asset must not grow its own.
	if strings.Contains(script, "dialog.close(") || strings.Contains(script, "confirm(") {
		t.Error("demo-init.js must leave closing to app.js and never fall back to window.confirm")
	}
	appBytes, err := Assets.ReadFile("assets/app.js")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(appBytes), `closest("[data-close-dialog]")`) || !strings.Contains(string(appBytes), "dialog._returnFocus") {
		t.Error("app.js no longer offers [data-close-dialog] closing with focus return, which demo-init.js relies on")
	}
}
