package web

import (
	"strings"
	"testing"
)

func TestHausv615VerwaltungNavigationOrderCountAndActiveState(t *testing.T) {
	body := renderComponent(t, PortalNavigation(organisationPortalFixture(PortalPageData{Organisation: VerwaltungShell{Active: "inbox", ShowInboxNav: true, InboxOpenCount: 17, CanManageSettings: true, Houses: []VerwaltungHouse{{Slug: "muenze", Name: "Münzgrabenstraße 12", Role: "Verwalter"}, {Slug: "haupt", Name: "Hauptplatz 3", Role: "Verwalter"}}}}), true, true))
	labels := []string{"Portfolio", "Posteingang", "Textbausteine", "Rechte", "Einstellungen"}
	last := -1
	for _, label := range labels {
		index := strings.Index(body, label)
		if index <= last {
			t.Fatalf("navigation order for %q is wrong: %s", label, body)
		}
		last = index
	}
	for _, want := range []string{`data-navigation-block="map"`, `href="/app/verwaltung/posteingang" class="nav-item active"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("navigation missing %q: %s", want, body)
		}
	}
}

func TestHausv615PortfolioShellKeepsTableContentInsideCard(t *testing.T) {
	body := renderComponent(t, PortfolioPage(organisationPortalFixture(PortalPageData{Organisation: VerwaltungShell{OrganisationName: "Hausverwaltung Musterstadt"}}), PortfolioData{
		Houses: []PortfolioHouse{{Name: "Münzgrabenstraße 12", Address: "Münzgrabenstraße 12, 8010 Graz", Assignee: "Vera Verwalter"}},
	}))
	// Preserve the no-clipping/column-priority contract with the larger Batch 3
	// type and 32px page gutters. Desktops show seven, six or five columns;
	// stacked tablets recover the sixth at 860px and the seventh at 980px.
	for _, want := range []string{
		"Münzgrabenstraße 12", "Vera Verwalter",
		".portfolio-table-head,.portfolio-house-row{min-width:0;display:grid;grid-template-columns:",
		"@media(min-width:1024px) and (max-width:1365px){.portfolio-table-head,.portfolio-house-row{grid-template-columns:12px minmax(120px,1.5fr) 32px 56px 56px minmax(96px,1fr)}.portfolio-table-head>span:last-child,.portfolio-house-row>span:last-child{display:none}}",
		"@media(min-width:1024px) and (max-width:1199px){.portfolio-layout{grid-template-columns:minmax(0,1.8fr) minmax(280px,1fr)}.portfolio-table-head,.portfolio-house-row{grid-template-columns:12px minmax(90px,1fr) 32px 56px 48px}.portfolio-table-head>span:nth-child(6),.portfolio-house-row>span:nth-child(6){display:none}}",
		"@media(max-width:1023px){.portfolio-layout{grid-template-columns:minmax(0,1fr)}",
		"@media(min-width:860px) and (max-width:979px){.portfolio-table-head,.portfolio-house-row{grid-template-columns:12px minmax(136px,1fr) 32px 56px 48px minmax(88px,1fr)}.portfolio-table-head>span:last-child,.portfolio-house-row>span:last-child{display:none}}",
		"@media(min-width:761px) and (max-width:859px){.portfolio-table-head,.portfolio-house-row{grid-template-columns:12px minmax(112px,1fr) 32px 56px 48px}.portfolio-table-head>span:nth-child(n+6),.portfolio-house-row>span:nth-child(n+6){display:none}}",
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
	body := renderComponent(t, TextbausteinListPage(organisationPortalFixture(PortalPageData{Organisation: VerwaltungShell{OrganisationName: "Musterstadt"}}), TextbausteinListData{
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
	settings := renderComponent(t, VerwaltungSettingsPage(organisationPortalFixture(PortalPageData{Organisation: VerwaltungShell{OrganisationName: "Musterstadt"}}), VerwaltungSettingsData{
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

	result := renderComponent(t, VerwaltungDemoResetPage(organisationPortalFixture(PortalPageData{Organisation: VerwaltungShell{OrganisationName: "Musterstadt"}}), VerwaltungDemoResetData{Done: true}))
	for _, want := range []string{"Zum Posteingang", "Zurück zu den Einstellungen"} {
		if !strings.Contains(result, want) {
			t.Fatalf("demo result missing %q", want)
		}
	}
}
