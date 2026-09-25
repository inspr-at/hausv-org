package statementpdf

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/pdf"
	"github.com/inspr-at/hausv-org/internal/store"
)

func TestHeizKG18AnnexContentAndHistoricalSnapshot(t *testing.T) {
	run := combinedRun(t, "weg")
	for i := range run.Input.Units {
		run.Input.Units[i].UnitType = "wohnung"
	}
	run.Input.Structure.Legal.HeatingInformation = &store.AnnualStatementHeatingInformation{
		Purchases: []store.AnnualStatementEnergyPurchase{{CostTypeKey: "heizung", Supplier: "Energie Muster", Carrier: "Fernwaerme", QuantityMicros: 10_000_000, Unit: "kWh", PriceMicros: 123456, PriceNote: "31.12.2025, Bruttopreis"}},
		TaxesNote: "20 % Umsatzsteuer", DistrictHeatingOver20MW: true, FuelMix: "80 % Biomasse", Emissions: "12.000 t CO2 pro Jahr", MeteringCostsNote: "300,00 EUR", OperatingCostsNote: "Wartung 600,00 EUR", RemoteMeters: "yes", MonthlyInformation: "Zugang beim Versorger", ComplaintContact: "Verwaltung", ClimateCurrentPPM: 1_100_000, ClimatePreviousPPM: 900000, ClimateSource: "Heizgradtage laut Fachgutachten",
	}
	run.Input.PreviousHeating = &store.AnnualStatementPreviousHeating{RunID: "prior", Revision: 2, Period: store.AnnualStatementPeriod{Year: 2024, StartsOn: "2024-01-01", EndsOn: "2024-12-31"}, Consumption: map[string]store.AnnualStatementConsumptionVector{"heizung": {PeriodYear: 2024, CostTypeKey: "heizung", Units: []store.AnnualStatementUnitConsumption{{UnitID: "a", ValueMicros: 2_000_000, MeasurementUnit: "kWh"}}}}}
	docs, err := Documents(run, "a", "owner")
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	headings := map[string]bool{"Energiebezüge und Preise (§ 18 Abs. 1 Z 1a–1c HeizKG)": false, "Verbrauchsvergleich": false, "Verbraucherinformation (§ 18 Abs. 1 Z 13–15 HeizKG)": false}
	for _, page := range docs[0].Pages() {
		for _, line := range page.Lines {
			lines = append(lines, line.Text)
			if _, heading := headings[line.Text]; heading {
				headings[line.Text] = line.Style == pdf.Strong
			}
			if line.X < left || line.X+pdf.TextWidth(line.Text, line.Style, line.Size) > left+measure+.01 || line.Y < bottom {
				t.Fatalf("out of bounds: %+v", line)
			}
		}
	}
	for heading, bold := range headings {
		if !bold {
			t.Fatalf("missing bold subsection: %s", heading)
		}
	}
	text := strings.Join(lines, " ")
	for _, want := range []string{"Energie Muster", "10 kWh", "0,123456 €/kWh", "31.12.2025", "20 % Umsatzsteuer", "Brennstoffmix", "80 % Biomasse", "12.000 t CO2", "Mess- und Berechnungskosten", "Wartung 600,00 EUR", "Vorperiode 01.01.2024", "Revision 2", "Klimabereinigt: aktuell 1,1 kWh; Vorperiode 1,8 kWh", "2 kWh je Einheit", "Vergleichsgruppe: 2", "Verbraucherinformation", "www.topprodukte.at", "www.verbraucherschlichtung.at/antrag/", "Schlichtungsstelle", "Monatliche Verbrauchsinformation", "Zugang beim Versorger"} {
		if !strings.Contains(text, want) {
			t.Fatalf("annex missing %q", want)
		}
	}
	raw, err := Render(run, "a", "owner")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Energie Muster", "0,123456 \\200/kWh", "Klimabereinigt", "www.verbraucherschlichtung.at/antrag/"} {
		if !bytes.Contains(raw, []byte(want)) {
			t.Fatalf("PDF omits %q", want)
		}
	}
	encoded, _ := json.Marshal(run)
	var restored store.AnnualStatementRun
	if err := json.Unmarshal(encoded, &restored); err != nil {
		t.Fatal(err)
	}
	again, err := Render(restored, "a", "owner")
	if err != nil || !bytes.Equal(raw, again) {
		t.Fatal("snapshot rendering changed", err)
	}
	result, issues := store.ReplayAnnualStatementRun(restored)
	if len(issues) > 0 || result.TotalCents != run.Result.TotalCents {
		t.Fatal("historical algorithm changed", issues)
	}
}

func TestHeatingComparisonsExcludeOtherCategoriesAndFlagMissingData(t *testing.T) {
	run := combinedRun(t, "weg")
	run.Input.Units[0].UnitType = "wohnung"
	run.Input.Units[1].UnitType = "geschaeft"
	info := &store.AnnualStatementHeatingInformation{}
	text := strings.Join(heatingComparisons(run, "a", "heizung", info), " ")
	if !strings.Contains(text, "1 kWh je Einheit") || !strings.Contains(text, "Vergleichsgruppe: 1") || !strings.Contains(text, "Vorperiodenvergleich nicht verfügbar") {
		t.Fatal(text)
	}
	run.Input.PreviousHeating = &store.AnnualStatementPreviousHeating{Consumption: run.Input.Consumption}
	text = strings.Join(heatingComparisons(run, "a", "heizung", info), " ")
	if !strings.Contains(text, "Klimakorrektur fehlt") {
		t.Fatal(text)
	}
	run.Input.PreviousHeating.Consumption = map[string]store.AnnualStatementConsumptionVector{"heizung": {Units: []store.AnnualStatementUnitConsumption{{UnitID: "a", MeasurementUnit: "MWh", ValueMicros: 100}}}}
	text = strings.Join(heatingComparisons(run, "a", "heizung", info), " ")
	if !strings.Contains(text, "Vorperiodenvergleich nicht verfügbar") {
		t.Fatal(text)
	}
}
