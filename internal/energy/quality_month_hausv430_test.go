package energy

import (
	"testing"
	"time"
)

// Die Datenlage darf nicht beruhigen, solange für den laufenden Monat keine
// Bemessungsgrundlage besteht. Eine frische Home-Assistant-Verbindung sagt
// nichts darüber aus, ob eine Viertelstunde des Monats gemessen wurde — und der
// Leistungstarif bemisst genau diese.

func TestFreshLiveValuesDoNotImplyAMonthlyBasisHAUSV430(t *testing.T) {
	now := time.Now()
	quality := AssessQuality(now, now.Add(-time.Minute), 0, 0, 0)
	if quality.Status == QualityMeasured {
		t.Fatalf("ohne Viertelstunde im Monat darf die Datenlage nicht als gemessen gelten: %+v", quality)
	}
	if quality.NextAction == "Keine Aktion nötig." {
		t.Fatalf("ohne Bemessungsgrundlage ist Handeln nötig: %+v", quality)
	}
}

func TestMonthlyBasisRestoresTheCalmStateHAUSV430(t *testing.T) {
	now := time.Now()
	quality := AssessQuality(now, now.Add(-time.Minute), 0, 0, 1)
	if quality.Status != QualityMeasured {
		t.Fatalf("mit frischen Werten und Monatsgrundlage erwartet: gemessen, war %+v", quality)
	}
}

// Eine unterbrochene Verbindung bleibt das dringendere Problem: sie zu melden
// hilft mehr als der Hinweis, Viertelstunden zu importieren.
func TestBrokenConnectionOutranksTheMissingBasisHAUSV430(t *testing.T) {
	now := time.Now()
	quality := AssessQuality(now, now.Add(-time.Hour), 0, 0, 0)
	if quality.Status != QualityStale {
		t.Fatalf("veraltete Werte müssen zuerst gemeldet werden: %+v", quality)
	}
}

// Ohne jeden Messwert bleibt die bestehende, deutlichere Aussage bestehen.
func TestNoMeasurementAtAllKeepsItsOwnMessageHAUSV430(t *testing.T) {
	quality := AssessQuality(time.Now(), time.Time{}, 0, 0, 0)
	if quality.Status != QualityUnavailable {
		t.Fatalf("ohne jeden Messwert erwartet: nicht verfügbar, war %+v", quality)
	}
}
