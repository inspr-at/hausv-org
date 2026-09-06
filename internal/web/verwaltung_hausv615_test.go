package web

import (
	"strings"
	"testing"
)

func TestHausv615VerwaltungNavigationOrderCountAndActiveState(t *testing.T) {
	body := renderComponent(t, VerwaltungNavigation(VerwaltungShell{
		Active: "inbox", ShowInboxNav: true, InboxOpenCount: 17, CanManageSettings: true,
		Houses: []VerwaltungHouse{{Name: "Münzgrabenstraße 12"}, {Name: "Hauptplatz 3"}},
	}, true))
	labels := []string{"Portfolio", "Posteingang", "Häuser", "Textbausteine", "Rechte", "Einstellungen"}
	last := -1
	for _, label := range labels {
		index := strings.Index(body, label)
		if index <= last {
			t.Fatalf("navigation order for %q is wrong: %s", label, body)
		}
		last = index
	}
	for _, want := range []string{`aria-label="2 verwaltete Häuser"`, `class="verwaltung-house-count"`, `href="/app/verwaltung/posteingang" class="nav-item active"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("navigation missing %q: %s", want, body)
		}
	}
}

func TestHausv615PortfolioShellKeepsTableContentInsideCard(t *testing.T) {
	body := renderComponent(t, PortfolioPage(VerwaltungShell{OrganisationName: "Hausverwaltung Musterstadt"}, PortfolioData{
		Houses: []PortfolioHouse{{Name: "Münzgrabenstraße 12", Address: "Münzgrabenstraße 12, 8010 Graz", Assignee: "Vera Verwalter"}},
	}))
	// The table stays inside the card by column priority, not by an inner
	// scroller: overflow-x:auto without a visible bar cut the Zuständig column
	// at 1280 (HAUSV-637). Below 1260 the column leaves; below 1120 so does
	// Nächster Termin. Those drops belong to the two-column band and stop at
	// 1024: below that the card stacks above the side column and is as wide as
	// the page again, so six columns return from 761 and all seven from 860
	// (HAUSV-640).
	for _, want := range []string{
		"Münzgrabenstraße 12", "Vera Verwalter",
		".verwaltung-page .portfolio-table-head,.verwaltung-page .portfolio-house-row{min-width:0;grid-template-columns:",
		"@media(max-width:1260px) and (min-width:1024px){.verwaltung-page .portfolio-table-head,.verwaltung-page .portfolio-house-row{grid-template-columns:12px minmax(90px,1.3fr) 36px 58px 56px minmax(80px,.9fr)}.verwaltung-page .portfolio-table-head>span:last-child,.verwaltung-page .portfolio-house-row>span:last-child{display:none}}",
		"@media(max-width:1120px) and (min-width:1024px){.verwaltung-page .portfolio-table-head,.verwaltung-page .portfolio-house-row{grid-template-columns:12px minmax(90px,1.3fr) 36px 58px 56px}",
		"@media(max-width:859px) and (min-width:761px){.verwaltung-page .portfolio-table-head,.verwaltung-page .portfolio-house-row{grid-template-columns:12px minmax(150px,1.3fr) 36px 58px 56px minmax(80px,.9fr)}.verwaltung-page .portfolio-table-head>span:last-child,.verwaltung-page .portfolio-house-row>span:last-child{display:none}}",
		"@media(max-width:1023px) and (min-width:860px){.verwaltung-page .portfolio-table-head,.verwaltung-page .portfolio-house-row{grid-template-columns:12px minmax(150px,1.35fr) 36px 58px 56px minmax(88px,.9fr) minmax(76px,.72fr)}}",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("portfolio render missing %q", want)
		}
	}
	for _, banned := range []string{".portfolio-desktop-table{overflow-x:auto}", "min-width:595px"} {
		if strings.Contains(body, banned) {
			t.Fatalf("portfolio render must not scroll the Häuser table inside the card: found %q", banned)
		}
	}
}

func TestHausv615TextbausteineStackOnPhoneWithCountsAndActions(t *testing.T) {
	body := renderComponent(t, TextbausteinListPage(VerwaltungShell{OrganisationName: "Musterstadt"}, TextbausteinListData{
		Count: 2,
		Groups: []TextbausteinGroup{{Label: "Betriebskosten", Items: []TextbausteinRow{
			{Key: "betriebskosten-pruefung", Title: "Betriebskosten prüfen", Status: "Aktiv", Active: true, Updated: "03.09.2026", EditURL: "/eins"},
			{Key: "betriebskosten-rueckfrage", Title: "Rückfrage", Status: "Inaktiv", Updated: "03.09.2026", EditURL: "/zwei"},
		}}},
	}))
	for _, want := range []string{"2 Textbausteine", "betriebskosten-pruefung", "betriebskosten-rueckfrage", ".textbausteine-table tr{min-width:0;display:grid", `href="/eins"`, `href="/zwei"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("textbausteine render missing %q", want)
		}
	}
	if strings.Count(body, ">Bearbeiten</a>") != 2 {
		t.Fatalf("each row must expose Bearbeiten: %s", body)
	}
}

func TestHausv615SettingsAndDemoFlowsAreStructured(t *testing.T) {
	settings := renderComponent(t, VerwaltungSettingsPage(VerwaltungShell{OrganisationName: "Musterstadt"}, VerwaltungSettingsData{
		Threshold: 90, AIProvider: "environment",
	}))
	for _, want := range []string{"settings-input-group", "Bilanz seit Start", "Kein KI-Anbieter konfiguriert", `value="environment" checked`, "KI-Anbieter", "KI-Verbindung"} {
		if !strings.Contains(settings, want) {
			t.Fatalf("settings render missing %q", want)
		}
	}
	if strings.Index(settings, "Bilanz seit Start") > strings.Index(settings, "KI-Anbieter") {
		t.Fatal("Bilanz must follow Automatisierung before the AI cards")
	}

	result := renderComponent(t, VerwaltungDemoResetPage(VerwaltungShell{OrganisationName: "Musterstadt"}, VerwaltungDemoResetData{Done: true}))
	for _, want := range []string{"Zum Posteingang", "Zurück zu den Einstellungen"} {
		if !strings.Contains(result, want) {
			t.Fatalf("demo result missing %q", want)
		}
	}
}
