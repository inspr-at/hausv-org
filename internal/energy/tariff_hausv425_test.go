package energy

import (
	"math"
	"strings"
	"testing"
	"time"
)

// Die verrechnete Leistung ist eine Geldgröße. Diese Tests fixieren die drei
// Regeln des Entwurfs, die sie bestimmen, und die Trennung der beiden 10-kW-
// Begriffe.

func TestEstimateUsesMeasuredPeakWhenItDominates(t *testing.T) {
	profile := AustrianDraft2027()
	got := profile.Estimate(8.4, 14)
	if math.Abs(got.BilledKW-8.4) > 0.001 {
		t.Fatalf("verrechnete Leistung = %v, erwartet die gemessene Spitze 8.4", got.BilledKW)
	}
	if got.MinimumReason != "" {
		t.Fatalf("keine Mindestbemessung erwartet, bekam %q", got.MinimumReason)
	}
}

func TestEstimateAppliesAgreedPowerMinimum(t *testing.T) {
	profile := AustrianDraft2027()
	// 20 % von 25 kW sind 5 kW und liegen über der gemessenen Spitze.
	got := profile.Estimate(3.1, 25)
	if math.Abs(got.BilledKW-5) > 0.001 {
		t.Fatalf("verrechnete Leistung = %v, erwartet 5 (20 %% von 25)", got.BilledKW)
	}
	if got.MinimumReason == "" {
		t.Fatal("die Mindestbemessung muss begründet werden, sonst wirkt sie wie ein Rechenfehler")
	}
}

func TestEstimateFallsBackToFloorWithoutAgreedPower(t *testing.T) {
	profile := AustrianDraft2027()
	// agreedKW = 0 heißt "nicht erfasst": nur der 2-kW-Sockel darf greifen,
	// keine stillschweigende Annahme über die vereinbarte Leistung.
	got := profile.Estimate(0.4, 0)
	if math.Abs(got.BilledKW-2) > 0.001 {
		t.Fatalf("verrechnete Leistung = %v, erwartet den 2-kW-Sockel", got.BilledKW)
	}
	if got.AboveKW != 0 {
		t.Fatalf("unterhalb der Staffel darf kein Anteil in der höheren Stufe liegen, bekam %v", got.AboveKW)
	}
}

func TestEstimateSplitsAtTierThreshold(t *testing.T) {
	profile := AustrianDraft2027()
	got := profile.Estimate(16, 0)
	if math.Abs(got.BelowKW-10) > 0.001 || math.Abs(got.AboveKW-6) > 0.001 {
		t.Fatalf("Aufteilung = %v/%v, erwartet 10/6", got.BelowKW, got.AboveKW)
	}
	// Der höhere Satz ist im Entwurf doppelt so hoch; die Summe muss das zeigen.
	want := 10*profile.AnnualBelowEURPerKW + 6*profile.AnnualAboveEURPerKW
	if math.Abs(got.AnnualPowerEUR-want) > 0.01 {
		t.Fatalf("Jahresbetrag = %v, erwartet %v", got.AnnualPowerEUR, want)
	}
}

func TestEstimateNeverPromisesGuarantee(t *testing.T) {
	got := AustrianDraft2027().Estimate(12, 20)
	if got.Guaranteed {
		t.Fatal("eine Entwurfsannahme darf nie als garantiert ausgewiesen werden")
	}
	if got.ProfileID == "" || got.AssumptionLabel == "" {
		t.Fatal("jede Schätzung braucht Regelprofil und sichtbaren Annahmecharakter")
	}
}

func TestDraftProfileSeparatesTheTwoTenKilowattTerms(t *testing.T) {
	profile := AustrianDraft2027()
	if profile.TierThresholdKW != 10 {
		t.Fatalf("Staffelschwelle = %v, erwartet 10", profile.TierThresholdKW)
	}
	// Der § 18-Referenzwert ist bewusst NICHT modelliert: er ist eine einmalige
	// Übergangsbestimmung und darf nicht in die laufende Rechnung geraten.
	found := false
	for _, assumption := range profile.Assumptions {
		if strings.Contains(assumption, "§ 18") {
			found = true
		}
	}
	if !found {
		t.Fatal("die Annahmen müssen die Staffel vom § 18-Referenzwert abgrenzen")
	}
}

func TestNormalizeProfileRejectsImplausibleAgreedPower(t *testing.T) {
	for _, value := range []float64{0, -5, 5000} {
		v := value
		got := NormalizeProfile(HomeProfile{TenantSlug: "haus", AgreedPowerKW: &v}, time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC))
		if got.AgreedPowerKW != nil {
			t.Fatalf("unplausible Anschlussleistung %v wurde übernommen", value)
		}
	}
	valid := 14.25
	got := NormalizeProfile(HomeProfile{TenantSlug: "haus", AgreedPowerKW: &valid}, time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC))
	if got.AgreedPowerKW == nil || math.Abs(*got.AgreedPowerKW-14.25) > 0.001 {
		t.Fatal("eine plausible Anschlussleistung muss erhalten bleiben")
	}
}
