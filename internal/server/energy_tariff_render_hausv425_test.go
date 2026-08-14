package server

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/energy"
)

// billedBlockHAUSV425 schneidet die Tarifkarte heraus. Kennzahlen und ihre
// Begründungen dürfen zugunsten einer ruhigen Erstansicht auf sichtbare Werte
// und die Offenlegung "Mehr erfahren" verteilt sein; außerhalb der Karte darf
// etwa die 10-kW-Planungsgrenze den Nachweis weiterhin nicht zufällig erfüllen.
func billedBlockHAUSV425(t *testing.T, body string) string {
	t.Helper()
	start := strings.Index(body, `<section class="energy-card energy-tariff"`)
	if start < 0 {
		t.Fatal("Tarifkarte fehlt im Cockpit")
	}
	end := strings.Index(body[start:], "</section>")
	if end < 0 {
		t.Fatal("Tarifkarte ist nicht vollständig geschlossen")
	}
	return body[start : start+end+len("</section>")]
}

// Ende-zu-Ende-Nachweis für den Tarifblock aus HAUSV-425: die verrechnete
// Leistung, ihre Begründung und die Staffelaufteilung müssen tatsächlich im
// gerenderten Cockpit stehen. Die Rechenregeln sind anderswo abgedeckt; hier
// geht es darum, dass sie den Weg bis in die Oberfläche finden.

func energyCockpitAppHAUSV425(t *testing.T, peakKW float64, agreedKW string) *app {
	t.Helper()
	a := newTestPortalApp(t, userProfile{
		Email:       "owner@example.com",
		Role:        roleOwner,
		Tenants:     []string{"demo"},
		AuthMethods: defaultAuthMethods(),
	})
	steps := []url.Values{
		{"action": {"profile"}, "household_name": {"Zuhause Test"}, "home_type": {"house"}},
		{"action": {"assets"}, "assets": {"pv", "ev", "battery"}},
		{"action": {"mappings"}},
		{"action": {"finish"}},
	}
	for i, form := range steps {
		if response := authedFormRequest(t, a, "owner@example.com", "/demo/app/zuhause/onboarding", form); response.Code != http.StatusSeeOther {
			t.Fatalf("Onboarding-Schritt %d: status=%d", i+1, response.Code)
		}
	}

	// Eine Viertelstunde im laufenden Monat, damit PeakForMonth überhaupt einen
	// Wert liefert.
	start := quarterHourThisMonth()
	if err := a.energyStore.PutInterval(energy.Interval{
		TenantSlug: "demo",
		StartsAt:   start,
		Duration:   15 * time.Minute,
		AverageKW:  peakKW,
		ImportKWh:  peakKW / 4,
		Quality:    energy.QualityMeasured,
		Source:     "test",
	}); err != nil {
		t.Fatalf("Intervall speichern: %v", err)
	}

	if agreedKW != "" {
		response := authedFormRequest(t, a, "owner@example.com", "/demo/app/energie/anschlussleistung",
			url.Values{"agreed_power_kw": {agreedKW}})
		if response.Code != http.StatusSeeOther {
			t.Fatalf("Anschlussleistung speichern: status=%d body=%s", response.Code, response.Body.String())
		}
	}
	return a
}

func TestCockpitShowsBilledPowerAndTierSplitHAUSV425(t *testing.T) {
	// 16 kW Spitze: 10 kW in der günstigeren, 6 kW in der höheren Stufe.
	a := energyCockpitAppHAUSV425(t, 16, "")
	body := authedRequest(t, a, "owner@example.com", "/demo/app/energie").Body.String()

	block := billedBlockHAUSV425(t, body)
	for _, want := range []string{
		"Höchste Viertelstunde",
		"Verrechnet",
		"Günstigere Stufe",
		"Höhere Stufe",
		"16\u00a0kW", // gemessene Spitze
		"10\u00a0kW", // günstigere Stufe bis zur Staffelschwelle
		"6\u00a0kW",  // Anteil darüber
	} {
		if !strings.Contains(block, want) {
			t.Fatalf("Kennzahlenblock ohne %q, war:\n%s", want, block)
		}
	}
	// Die Nicht-Hebel müssen dabeistehen, sonst liest sich die Seite als
	// Versprechen, das SNAP/WiNAP und Energiegemeinschaften nicht halten.
	if !strings.Contains(body, "senken den Arbeitspreis, nicht die verrechnete Leistung") {
		t.Fatal("Hinweis zu Arbeitspreis-Hebeln fehlt")
	}
	if !strings.Contains(block, `class="energy-tariff-meter peak" max="100" value="100"`) {
		t.Fatal("bei identischer Spitze und Verrechnung muss der Vergleichsbalken vollständig gefüllt sein")
	}
}

func TestCockpitExplainsMinimumChargeHAUSV425(t *testing.T) {
	// 3 kW gemessen, aber 40 kW vereinbart: 20 % davon sind 8 kW und dominieren.
	a := energyCockpitAppHAUSV425(t, 3, "40")
	body := authedRequest(t, a, "owner@example.com", "/demo/app/energie").Body.String()

	if !strings.Contains(body, "Mindestbemessung aus der vereinbarten Leistung") {
		t.Fatal("die Mindestbemessung muss begründet werden, sonst wirkt sie wie ein Rechenfehler")
	}
	if block := billedBlockHAUSV425(t, body); !strings.Contains(block, "8\u00a0kW") {
		t.Fatalf("verrechnete Leistung 8 kW fehlt im Kennzahlenblock, war:\n%s", block)
	}
	if block := billedBlockHAUSV425(t, body); !strings.Contains(block, `class="energy-tariff-meter peak" max="100" value="38"`) {
		t.Fatalf("der Vergleichsbalken muss 3 kW relativ zu 8 kW als 38 Prozent darstellen, war:\n%s", block)
	}
	// Und das Szenario darf unterhalb davon keine weitere Ersparnis andeuten.
	if !strings.Contains(body, "sinkt der verrechnete Betrag nicht weiter") {
		t.Fatal("Hinweis auf die Untergrenze fehlt im Szenario")
	}
}

func TestCockpitPromptsForAgreedPowerWhenMissingHAUSV425(t *testing.T) {
	a := energyCockpitAppHAUSV425(t, 12, "")
	body := authedRequest(t, a, "owner@example.com", "/demo/app/energie").Body.String()

	if !strings.Contains(body, "Vereinbarte Anschlussleistung") {
		t.Fatal("das Eingabefeld für die Anschlussleistung fehlt")
	}
	if !strings.Contains(body, "Ohne vereinbarte Anschlussleistung") {
		t.Fatal("ohne erfassten Wert muss die Oberfläche das benennen statt still zu rechnen")
	}
}
