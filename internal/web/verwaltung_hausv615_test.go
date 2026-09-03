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
	for _, want := range []string{"Münzgrabenstraße 12", "Vera Verwalter", ".verwaltung-page .portfolio-desktop-table{overflow-x:auto}", "minmax(95px,.72fr)"} {
		if !strings.Contains(body, want) {
			t.Fatalf("portfolio render missing %q", want)
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

	result := renderComponent(t, VerwaltungDemoResetPage(VerwaltungShell{OrganisationName: "Musterstadt"}, VerwaltungDemoResetData{Step: 3}))
	for _, want := range []string{"Zum Posteingang", "Zurück zu den Einstellungen"} {
		if !strings.Contains(result, want) {
			t.Fatalf("demo result missing %q", want)
		}
	}
}
