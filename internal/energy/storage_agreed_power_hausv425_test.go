package energy_test

import (
	"math"
	"path/filepath"
	"testing"
	"time"

	appdb "github.com/markus-barta/hausv-org/internal/db"
	"github.com/markus-barta/hausv-org/internal/energy"
)

// Die vereinbarte Anschlussleistung steuert die Mindestbemessung und damit eine
// Geldgröße. Sie muss den Rundlauf durch beide Speicher unverändert überstehen
// — und "nicht erfasst" muss von "null kW" unterscheidbar bleiben.
func TestAgreedPowerSurvivesRoundTripHAUSV425(t *testing.T) {
	for name, factory := range lifecycleStoreFactories() {
		t.Run(name, func(t *testing.T) {
			store := factory(t)
			profile := energy.DefaultProfile("haus", time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC))
			profile.HouseholdName = "Haus Muster"
			agreed := 14.0
			profile.AgreedPowerKW = &agreed
			if err := store.SaveProfile(profile); err != nil {
				t.Fatalf("Profil speichern: %v", err)
			}

			loaded, ok, err := store.Profile("haus")
			if err != nil || !ok {
				t.Fatalf("Profil laden: %v (gefunden=%v)", err, ok)
			}
			if loaded.AgreedPowerKW == nil {
				t.Fatal("vereinbarte Anschlussleistung ging beim Speichern verloren")
			}
			if math.Abs(*loaded.AgreedPowerKW-14) > 0.001 {
				t.Fatalf("Anschlussleistung = %v, erwartet 14", *loaded.AgreedPowerKW)
			}

			// Leeren muss die Mindestbemessung wieder verstummen lassen.
			loaded.AgreedPowerKW = nil
			if err := store.SaveProfile(loaded); err != nil {
				t.Fatalf("Profil erneut speichern: %v", err)
			}
			cleared, _, err := store.Profile("haus")
			if err != nil {
				t.Fatalf("Profil erneut laden: %v", err)
			}
			if cleared.AgreedPowerKW != nil {
				t.Fatalf("geleerte Anschlussleistung blieb erhalten: %v", *cleared.AgreedPowerKW)
			}
		})
	}
}

// Ein Bestandsprofil ohne den Wert darf nicht plötzlich mit 0 kW rechnen.
func TestExistingProfileWithoutAgreedPowerStaysUnsetHAUSV425(t *testing.T) {
	database, err := appdb.Open(filepath.Join(t.TempDir(), "agreed.db"))
	if err != nil {
		t.Fatalf("Datenbank öffnen: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	store := energy.NewSQLStore(database)

	if err := store.SaveProfile(energy.DefaultProfile("altbestand", time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC))); err != nil {
		t.Fatalf("Profil speichern: %v", err)
	}
	loaded, ok, err := store.Profile("altbestand")
	if err != nil || !ok {
		t.Fatalf("Profil laden: %v (gefunden=%v)", err, ok)
	}
	if loaded.AgreedPowerKW != nil {
		t.Fatalf("ohne Erfassung muss der Wert leer bleiben, war %v", *loaded.AgreedPowerKW)
	}

	// Ohne erfassten Wert greift nur der 2-kW-Sockel, nicht die 20-%-Regel.
	estimate := energy.AustrianDraft2027().Estimate(1.2, 0)
	if math.Abs(estimate.BilledKW-2) > 0.001 {
		t.Fatalf("verrechnete Leistung = %v, erwartet den 2-kW-Sockel", estimate.BilledKW)
	}
}
