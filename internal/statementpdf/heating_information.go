package statementpdf

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/view"
)

func heatingInformation(run store.AnnualStatementRun, unitID, costKey string) []string {
	info := run.Input.Structure.Legal.HeatingInformation
	if info == nil {
		info = &store.AnnualStatementHeatingInformation{}
	}
	fallback := func(s string) string {
		if strings.TrimSpace(s) == "" {
			return "Noch nicht hinterlegt"
		}
		return s
	}
	lines := []string{"Energiebezüge und Preise (§ 18 Abs. 1 Z 1a–1c HeizKG)"}
	n := 0
	for _, p := range info.Purchases {
		if p.CostTypeKey != costKey {
			continue
		}
		n++
		decimal := func(value int64) string {
			return strings.ReplaceAll(strings.TrimRight(strings.TrimRight(new(big.Rat).SetFrac(big.NewInt(value), big.NewInt(1_000_000)).FloatString(6), "0"), "."), ".", ",")
		}
		lines = append(lines, p.Supplier+" · "+p.Carrier+": "+measurement(p.QuantityMicros, p.Unit)+"; Preis: "+decimal(p.PriceMicros)+" EUR/"+p.Unit, "Preisstand: "+p.PriceNote)
	}
	if n == 0 {
		lines = append(lines, "Energiebezugsmenge und tatsächliche Preise: Noch nicht hinterlegt")
	}
	lines = append(lines, "Steuern, Abgaben und Zolltarife: "+fallback(info.TaxesNote), "Mess- und Berechnungskosten: "+fallback(info.MeteringCostsNote), "Sonstige Betriebskosten, Aufschlüsselung: "+fallback(info.OperatingCostsNote))
	if info.DistrictHeatingOver20MW {
		lines = append(lines, "Fernwärmeanlage über 20 MW · Brennstoffmix: "+fallback(info.FuelMix), "Jährliche Treibhausgasemissionen: "+fallback(info.Emissions))
	}
	lines = append(lines, heatingComparisons(run, unitID, costKey, info)...)
	lines = append(lines,
		"Verbraucherinformation (§ 18 Abs. 1 Z 13–15 HeizKG)",
		"Energieeffizienz, Verbrauchsvergleich und Geräteinformationen: Arbeiterkammer – www.arbeiterkammer.at/beratung/konsument/Energie/; effiziente Geräte: www.topprodukte.at (Österreichische Energieagentur).",
		"Beschwerden: zunächst schriftlich an die Verwaltung bzw. den Abgeber richten. Kontakt: "+fallback(info.ComplaintContact),
		"Überprüfung nach HeizKG: zuständige kommunale Schlichtungsstelle, andernfalls Bezirksgericht (§ 25 HeizKG). Verbraucherberatung und Kontakte: www.arbeiterkammer.at.",
		"Alternative Streitbeilegung bei Verbrauchergeschäften: Verbraucherschlichtung Austria – www.verbraucherschlichtung.at/antrag/ (Zuständigkeit und Verfahrensvoraussetzungen prüfen).",
		"Folgen der Abrechnung (§§ 21–24 HeizKG): Guthaben und Nachforderung werden ausgeglichen; Korrekturen sind abzurechnen (§ 22). Bei Nutzerwechsel ist eine Zwischenabrechnung nach § 23 zu beachten. Schriftlich begründete Einwendungen binnen sechs Monaten, sonst gilt die Abrechnung als genehmigt (§ 24).",
	)
	switch info.RemoteMeters {
	case "yes":
		lines = append(lines, "Fernablesbare Zähler: Monatliche Verbrauchsinformation während der Heiz- und Kühlperioden ist nach § 17 Abs. 5 HeizKG bereitzustellen. Zugang/Kontakt: "+fallback(info.MonthlyInformation))
	case "no":
		lines = append(lines, "Keine fernablesbaren Zähler hinterlegt. Die monatliche Verbrauchsinformation nach § 17 Abs. 5 HeizKG setzt Fernablesbarkeit voraus.")
	default:
		lines = append(lines, "Fernablesbarkeit noch zu prüfen: Bei fernablesbaren Zählern sind während der Heiz- und Kühlperioden monatliche Verbrauchsinformationen nach § 17 Abs. 5 HeizKG bereitzustellen.")
	}
	return append(lines, "Diese Jahresabrechnung ersetzt die monatliche Verbrauchsinformation nicht. Abrechnungs- und Verbrauchsinformationen sowie Zugang zu Verbrauchsdaten sind kostenfrei (§ 18 Abs. 5 HeizKG).")
}

func heatingComparisons(run store.AnnualStatementRun, unitID, key string, info *store.AnnualStatementHeatingInformation) []string {
	vector := run.Input.Consumption[key]
	var current store.AnnualStatementUnitConsumption
	found := false
	for _, value := range vector.Units {
		if value.UnitID == unitID {
			current = value
			found = true
			break
		}
	}
	if !found {
		return []string{"Verbrauchsvergleich: Verbrauch der Einheit fehlt."}
	}
	lines := []string{"Verbrauchsvergleich · aktuelle Periode: " + measurement(current.ValueMicros, current.MeasurementUnit)}
	previous := run.Input.PreviousHeating
	comparable := false
	if previous != nil {
		for _, value := range previous.Consumption[key].Units {
			if value.UnitID != unitID {
				continue
			}
			if value.MeasurementUnit != current.MeasurementUnit {
				break
			}
			comparable = true
			lines = append(lines, fmt.Sprintf("Vorperiode %s bis %s (Revision %d): %s", date(previous.Period.StartsOn), date(previous.Period.EndsOn), previous.Revision, measurement(value.ValueMicros, value.MeasurementUnit)))
			if key == "heizung" {
				if info.ClimateCurrentPPM > 0 && info.ClimatePreviousPPM > 0 && strings.TrimSpace(info.ClimateSource) != "" {
					corrected := func(value int64, factor int) string {
						r := new(big.Rat).SetFrac(new(big.Int).Mul(big.NewInt(value), big.NewInt(int64(factor))), big.NewInt(1_000_000_000_000))
						return strings.ReplaceAll(r.FloatString(2), ".", ",") + " " + current.MeasurementUnit
					}
					lines = append(lines, "Klimabereinigt: aktuell "+corrected(current.ValueMicros, info.ClimateCurrentPPM)+"; Vorperiode "+corrected(value.ValueMicros, info.ClimatePreviousPPM), "Klimakorrektur – Quelle/Methode: "+info.ClimateSource)
				} else {
					lines = append(lines, "Klimakorrektur fehlt: Die unbereinigten Heizwerte sind kein klimabereinigter Vergleich nach § 18 Abs. 1 Z 6a HeizKG.")
				}
			}
			break
		}
	}
	if !comparable {
		lines = append(lines, "Vorperiodenvergleich nicht verfügbar: Kein gespeicherter Lauf des gleichen Vorjahreszeitraums mit vergleichbarem Einheitsverbrauch vorhanden.")
	}
	category := ""
	for _, unit := range run.Input.Units {
		if unit.ID == unitID {
			category = unit.UnitType
			break
		}
	}
	if area, recorded := run.Input.Structure.Legal.HeatableAreas[unitID]; recorded && area == 0 {
		return append(lines, "Hausvergleich entfällt: Diese Einheit wird nicht versorgt (0 m² versorgbare Nutzfläche).")
	}
	if category == "" || run.Input.Structure.Legal.HeatableAreas[unitID] <= 0 {
		return append(lines, "Hausvergleich nicht verfügbar: Nutzerkategorie oder versorgbare Fläche fehlt.")
	}
	peers := map[string]bool{}
	for _, unit := range run.Input.Units {
		if unit.UnitType == category && run.Input.Structure.Legal.HeatableAreas[unit.ID] > 0 {
			peers[unit.ID] = true
		}
	}
	total := new(big.Int)
	count := 0
	for _, value := range vector.Units {
		if peers[value.UnitID] && value.MeasurementUnit == current.MeasurementUnit {
			total.Add(total, big.NewInt(value.ValueMicros))
			count++
		}
	}
	if count == 0 || count != len(peers) {
		return append(lines, "Hausvergleich nicht verfügbar: Vergleichswerte derselben Nutzerkategorie fehlen.")
	}
	average := new(big.Rat).SetFrac(total, big.NewInt(int64(count)*1_000_000))
	lines = append(lines, fmt.Sprintf("Durchschnittsabnehmer derselben Nutzerkategorie (%s): %s %s je Einheit; Vergleichsgruppe: %d versorgte Einheiten dieser Liegenschaft einschließlich Ihrer Einheit.", view.UnitTypeLabel(category), strings.ReplaceAll(average.FloatString(2), ".", ","), current.MeasurementUnit, count))
	return append(lines, "Vergleichsmethode: arithmetischer Mittelwert der gemessenen Periodenverbräuche derselben Nutzerkategorie. Wohnungsgröße, Nutzung und Lage können den Verbrauch beeinflussen.")
}
