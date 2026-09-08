package web

import (
	"strings"
	"testing"
	"time"
)

func TestPortfolioTemplateRendersEmptyActionState(t *testing.T) {
	body := renderComponent(t, PortfolioPage(VerwaltungShell{OrganisationName: "Hausverwaltung Musterstadt"}, PortfolioData{
		Eyebrow:   "HAUSVERWALTUNG MUSTERSTADT · PORTFOLIO",
		Greeting:  "Guten Morgen, Vera.",
		TodayLine: "Mittwoch, 9. September 2026 · 0 Liegenschaften · 0 Einheiten",
		Sort:      "need",
	}))
	if !strings.Contains(body, "Keine Liegenschaften mit Handlungsbedarf") {
		t.Fatalf("portfolio empty state missing: %s", body)
	}
}

func TestPortfolioTemplateRendersCollapsedQuietHouses(t *testing.T) {
	body := renderComponent(t, PortfolioPage(VerwaltungShell{OrganisationName: "Hausverwaltung Musterstadt"}, PortfolioData{
		Eyebrow:         "HAUSVERWALTUNG MUSTERSTADT · PORTFOLIO",
		Greeting:        "Guten Morgen, Vera.",
		TodayLine:       "Mittwoch, 9. September 2026 · 2 Liegenschaften · 12 Einheiten",
		Sort:            "need",
		QuietHouseCount: 2,
		QuietHouses: []PortfolioHouse{
			{Slug: "birke", Role: "Verwalter", Name: "Birkenhaus", Address: "Birkenweg 1", Oldest: "—", NextDate: "—", NextTitle: "—", Assignee: "—", ActionClass: "green"},
			{Slug: "eiche", Role: "Verwalter", Name: "Eichenhaus", Address: "Eichenweg 2", Oldest: "—", NextDate: "—", NextTitle: "—", Assignee: "—", ActionClass: "green"},
		},
	}))
	for _, want := range []string{"2 weitere Liegenschaften ohne Handlungsbedarf anzeigen", "Birkenhaus", "Eichenhaus", `<details class="portfolio-quiet">`} {
		if !strings.Contains(body, want) {
			t.Fatalf("portfolio quiet group missing %q: %s", want, body)
		}
	}
}

func TestPortfolioAgendaGroupsActualDatesAndShowsTimes(t *testing.T) {
	today := time.Date(2026, 9, 8, 9, 0, 0, 0, time.Local)
	first := PortfolioAppointment{Title: "Versammlung", House: "Haus A", Day: "08", Month: "Sep", Time: "09:00", At: today}
	second := PortfolioAppointment{Title: "Wartung", House: "Haus B", Day: "09", Month: "Sep", Time: "13:00", At: today.AddDate(0, 0, 1).Add(4 * time.Hour)}
	data := PortfolioData{Today: "Dienstag, 8. September 2026", Appointments: []PortfolioAppointment{first, first, second}}
	body := renderComponent(t, PortfolioPage(VerwaltungShell{}, data))
	for _, heading := range []string{"Heute · 08. Sep", "Morgen · 09. Sep"} {
		if strings.Count(body, heading) != 1 {
			t.Fatalf("agenda must group %q once", heading)
		}
	}
	for _, label := range []string{"09:00 Uhr", "13:00 Uhr", "Versammlung", "Wartung", "Haus A", "Haus B"} {
		if !strings.Contains(body, label) {
			t.Fatalf("agenda missing %q", label)
		}
	}
	data.Appointments = []PortfolioAppointment{second}
	body = renderComponent(t, PortfolioPage(VerwaltungShell{}, data))
	if !strings.Contains(body, "Morgen · 09. Sep") || strings.Contains(body, "Heute · 09. Sep") {
		t.Fatal("a tomorrow-only agenda must not call its first group today")
	}
}
