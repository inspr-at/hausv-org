package web

import (
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/view"
)

// The live card is the only part of /app/parking that carries state-changing
// buttons. Which button appears depends on three flags, and the wrong branch
// means a resident either cannot stop a charge or cannot start one.
func TestParkingLiveCardRendersOneControlPerState(t *testing.T) {
	base := view.ParkingLiveView{
		Available: true, CanToggle: true,
		ModeLabel: "Überschussladen", ModeClass: "live-surplus", ModeDetail: "Lädt aus dem Dach.",
	}
	cases := []struct {
		name    string
		live    view.ParkingLiveView
		state   string
		actions []string
		absent  []string
	}{
		{
			name:    "auto paused",
			live:    func() view.ParkingLiveView { l := base; l.AutoPaused = true; return l }(),
			state:   "Automatik pausiert",
			actions: []string{`action="/app/parking/charging/auto"`, `action="/app/parking/charging/on"`, "Manuell steuern"},
			absent:  []string{`action="/app/parking/charging/off"`},
		},
		{
			name:    "charging",
			live:    func() view.ParkingLiveView { l := base; l.ToggleOn = true; return l }(),
			state:   "Lädt",
			actions: []string{`action="/app/parking/charging/off"`, "Ladung ausschalten"},
			absent:  []string{`action="/app/parking/charging/on"`, `action="/app/parking/charging/auto"`},
		},
		{
			name:    "idle",
			live:    base,
			state:   "Bereit",
			actions: []string{`action="/app/parking/charging/on"`, "Jetzt laden · Normaltarif"},
			absent:  []string{`action="/app/parking/charging/off"`, `action="/app/parking/charging/auto"`},
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			html := renderComponent(t, ParkingPage(ParkingPageData{
				Portal: PortalPageData{Title: "Parkplatznutzung"}, AssetVersion: "test", Live: testCase.live,
			}))
			if !strings.Contains(html, testCase.state) {
				t.Errorf("live card should report %q", testCase.state)
			}
			for _, want := range testCase.actions {
				if !strings.Contains(html, want) {
					t.Errorf("live card is missing %q", want)
				}
			}
			for _, forbidden := range testCase.absent {
				if strings.Contains(html, forbidden) {
					t.Errorf("live card must not offer %q in this state", forbidden)
				}
			}
		})
	}
}

func TestParkingLiveCardHidesControlsWithoutPermission(t *testing.T) {
	html := renderComponent(t, ParkingPage(ParkingPageData{
		Portal:       PortalPageData{Title: "Parkplatznutzung"},
		AssetVersion: "test",
		Live:         view.ParkingLiveView{Available: true, ModeLabel: "Bereit", ModeClass: "live-idle"},
	}))
	for _, forbidden := range []string{
		`action="/app/parking/charging/on"`,
		`action="/app/parking/charging/off"`,
		`action="/app/parking/charging/auto"`,
	} {
		if strings.Contains(html, forbidden) {
			t.Errorf("a viewer who cannot toggle must not see %q", forbidden)
		}
	}
}

func TestParkingLiveCardIsAbsentWhenChargingIsNotConfigured(t *testing.T) {
	html := renderComponent(t, ParkingPage(ParkingPageData{
		Portal: PortalPageData{Title: "Parkplatznutzung"}, AssetVersion: "test",
	}))
	if strings.Contains(html, `id="parking-live"`) {
		t.Fatal("the live card must stay hidden while charging is unavailable")
	}
	if strings.Contains(html, `class="parking-switcher"`) {
		t.Fatal("the section switcher needs both sections to switch between")
	}
}

// The month detail page renders the same sessions. Sharing the component is
// what keeps the two from drifting apart, so assert the component itself.
func TestParkingSessionListMarksSurplusAndRunningSessions(t *testing.T) {
	html := renderComponent(t, ParkingSessionList([]view.ChargingSessionView{
		{StartLabel: "12:00", DurationLabel: "1 h", KWh: "4,0 kWh", ModeLabel: "Überschuss", ModeClass: "mode-surplus", Cost: "0,80 €", Active: true},
		{StartLabel: "08:00", DurationLabel: "30 min", KWh: "2,0 kWh", ModeLabel: "Normal", ModeClass: "mode-normal", Cost: "0,60 €"},
	}))
	for _, want := range []string{"☀️", "Überschuss", "läuft", "0,80 €", "Normal", "0,60 €"} {
		if !strings.Contains(html, want) {
			t.Errorf("session list is missing %q", want)
		}
	}
	if strings.Count(html, "läuft") != 1 {
		t.Error("only the active session may be marked as running")
	}
}

func TestParkingMonthRowsCarryStatusAndDeepLink(t *testing.T) {
	html := renderComponent(t, ParkingPage(ParkingPageData{
		Portal:              PortalPageData{Title: "Parkplatznutzung"},
		AssetVersion:        "test",
		CurrentMonthHeading: "Aktueller Monat",
		HasOlderMonths:      true,
		Accounting: view.ParkingAccountingView{
			HasMonths: true, Message: "Aus Zählerdifferenz und aWATTar-Preis.",
			HasOutstanding: true, Outstanding: "1,40 €",
			HasOverdue: true, Overdue: "0,60 €",
		},
		CurrentMonth: view.ParkingMonthView{
			Month: "2026-06", MonthLabel: "Juni 2026", DetailPath: "/app/parking/month/2026-06",
			TotalCost: "0,80 €", KWh: "4,0 kWh", GridCost: "0,20 €", PaidLabel: "OFFEN", Overdue: true,
		},
		OlderMonths: []view.ParkingMonthView{{
			Month: "2026-05", MonthLabel: "Mai 2026", DetailPath: "/app/parking/month/2026-05",
			TotalCost: "0,60 €", PaidLabel: "BEZAHLT", Paid: true, Partial: true, HourCount: 3,
		}},
	}))
	for _, want := range []string{
		`id="parking-month-2026-06"`,
		`href="/app/parking/month/2026-06"`,
		`id="parking-month-2026-05"`,
		`href="/app/parking/month/2026-05"`,
		"1,40 € offen",
		"0,60 € überfällig",
		"Teilmonat · ",
		"3 Stunden",
		"Letzter Messpunkt: –",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("month list is missing %q", want)
		}
	}
	if !strings.Contains(html, `class="pill dringend"`) {
		t.Error("an overdue month must keep its urgent status pill")
	}
	if !strings.Contains(html, `class="pill ok"`) {
		t.Error("a paid month must keep its settled status pill")
	}
}
