package web

import (
	"strings"
	"testing"
)

func TestPortfolioTemplateRendersEmptyActionState(t *testing.T) {
	body := renderComponent(t, PortfolioPage(VerwaltungShell{OrganisationName: "Hausverwaltung Musterstadt"}, PortfolioData{
		Eyebrow:   "HAUSVERWALTUNG MUSTERSTADT · PORTFOLIO",
		Greeting:  "Guten Morgen, Vera.",
		TodayLine: "Mittwoch, 9. September 2026 · 0 Häuser · 0 Einheiten",
		Sort:      "need",
	}))
	if !strings.Contains(body, "Keine Häuser mit Handlungsbedarf") {
		t.Fatalf("portfolio empty state missing: %s", body)
	}
}

func TestPortfolioTemplateRendersCollapsedQuietHouses(t *testing.T) {
	body := renderComponent(t, PortfolioPage(VerwaltungShell{OrganisationName: "Hausverwaltung Musterstadt"}, PortfolioData{
		Eyebrow:         "HAUSVERWALTUNG MUSTERSTADT · PORTFOLIO",
		Greeting:        "Guten Morgen, Vera.",
		TodayLine:       "Mittwoch, 9. September 2026 · 2 Häuser · 12 Einheiten",
		Sort:            "need",
		QuietHouseCount: 2,
		QuietHouses: []PortfolioHouse{
			{Slug: "birke", Role: "Verwalter", Name: "Birkenhaus", Address: "Birkenweg 1", Oldest: "—", NextDate: "—", NextTitle: "—", Assignee: "—", ActionClass: "green"},
			{Slug: "eiche", Role: "Verwalter", Name: "Eichenhaus", Address: "Eichenweg 2", Oldest: "—", NextDate: "—", NextTitle: "—", Assignee: "—", ActionClass: "green"},
		},
	}))
	for _, want := range []string{"2 weitere Häuser ohne Handlungsbedarf anzeigen", "Birkenhaus", "Eichenhaus", `<details class="portfolio-quiet">`} {
		if !strings.Contains(body, want) {
			t.Fatalf("portfolio quiet group missing %q: %s", want, body)
		}
	}
}
