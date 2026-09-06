package statementpdf

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
)

func fixture() store.AnnualStatementRun {
	at := time.Date(2026, 2, 3, 10, 20, 0, 0, time.UTC)
	return store.AnnualStatementRun{ID: "run-2025-frozen", PeriodYear: 2025, Revision: 2, CreatedAt: at,
		Input: store.AnnualStatementRunInput{
			Presentation: store.AnnualStatementRunPresentation{Organisation: "Hausverwaltung Musterstadt", EstateSlug: "janusbergweg-123", EstateName: "Janusbergweg 123", EstateAddress: "Janusbergweg 123, 8010 Graz", ContactName: "Vera Verwalter", ContactEmail: "verwaltung@musterstadt.example", ContactPhone: "+43 316 555 100"},
			Period:       store.AnnualStatementPeriod{Year: 2025, StartsOn: "2025-01-01", EndsOn: "2025-12-31"},
			Parties:      []store.AnnualStatementRunParty{{UnitID: "b", ID: "zoe@example.com", Name: "Zoë Groß", Owner: true}, {UnitID: "a", ID: "tenant@example.com", Name: "Jürgen Mieter", Address: "Janusbergweg 123/1\n8010 Graz", Renter: true}, {UnitID: "a", ID: "owner@example.com", Name: "Anna Eigentümer", Address: "Andere Gasse 2\n8020 Graz", Owner: true}},
			Structure:    store.AnnualStatementPeriodStructure{CostTypes: []store.AnnualStatementCostType{{Key: "water", Name: "Wasser", Allocatable: true}, {Key: "heizung", Name: "Heizung", Allocatable: true}, {Key: "reserve", Name: "Rücklage", Allocatable: false}}, UnitBases: []store.AnnualStatementPeriodUnitBasis{{UnitID: "a", MiteigentumsanteilPPM: 250000, UsableAreaM2Hundredths: 6000, UsableAreaRecorded: true, Persons: 1, PersonsRecorded: true}}},
			Receipts:     []store.AnnualStatementReceipt{{CostTypeKey: "water", AmountCents: 100000}, {CostTypeKey: "heizung", AmountCents: 20000}, {CostTypeKey: "reserve", AmountCents: 5000}},
			Consumption:  map[string]store.AnnualStatementConsumptionVector{"heizung": {Units: []store.AnnualStatementUnitConsumption{{UnitID: "a", ValueMicros: 100000000, MeasurementUnit: "kWh"}, {UnitID: "b", ValueMicros: 300000000, MeasurementUnit: "kWh"}}}},
			Evidence:     []store.AnnualStatementConsumptionEvidence{{UnitID: "a", CostTypeKey: "heizung", MeasuredAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), ValueMicros: 500000000, MeasurementUnit: "kWh", SourceKind: "meter", SourceID: "heat-a"}, {UnitID: "a", CostTypeKey: "heizung", MeasuredAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), ValueMicros: 600000000, MeasurementUnit: "kWh", SourceKind: "meter", SourceID: "heat-a"}},
		}, Result: store.AnnualStatementRunResult{TotalCents: 120000, ExcludedCents: 5000, Units: []store.AnnualStatementRunUnit{{UnitID: "b", Label: "Top 2", AllocatedCents: 90000, PrepaidCents: 80000, BalanceCents: 10000, Costs: []store.AnnualStatementRunCost{{CostTypeKey: "water", Name: "Wasser", AllocationKey: store.AllocationKeyFlaeche, SharePPM: 750000, AmountCents: 75000}, {CostTypeKey: "heizung", Name: "Heizung", AllocationKey: store.AllocationKeyVerbrauch, SharePPM: 750000, AmountCents: 15000}}}, {UnitID: "a", Label: "Top 1", AllocatedCents: 30000, PrepaidCents: 35000, BalanceCents: -5000, Costs: []store.AnnualStatementRunCost{{CostTypeKey: "water", Name: "Wasser", AllocationKey: store.AllocationKeyFlaeche, SharePPM: 250000, AmountCents: 25000}, {CostTypeKey: "heizung", Name: "Heizung", AllocationKey: store.AllocationKeyVerbrauch, SharePPM: 250000, AmountCents: 5000}}}}}}
}

func TestDocumentStoredFiguresPartiesAndMeasurementEvidence(t *testing.T) {
	run := fixture()
	docs, err := Documents(run, "", "")
	if err != nil || len(docs) != 3 {
		t.Fatalf("documents=%+v err=%v", docs, err)
	}
	d := docs[0]
	if d.UnitID != "a" || d.PartyID != "owner@example.com" || d.Total != "300,00 €" || d.Prepaid != "350,00 €" || d.Balance != "Guthaben 50,00 €" || d.Costs[0].Total != "1.000,00 €" || d.Costs[0].Amount != "250,00 €" || d.Costs[0].Share != "25,00 %" {
		t.Fatalf("model=%+v", d)
	}
	raw, _ := json.Marshal(d)
	for _, want := range []string{"01.01.2025 bis 31.12.2025", "Revision 2", "03.02.2026 11:20 CET", "60,00 m²", "Personen: 1", "100,000000 kWh / 400,000000 kWh", "Grenzmessung Beginn", "Grenzmessung Ende", "500,000000 kWh", "600,000000 kWh", "Rücklage: 50,00 €"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("missing %q", want)
		}
	}
	if docs[1].Total != d.Total || docs[1].Balance != d.Balance || docs[1].Address[1] == d.Address[1] {
		t.Fatal("party figures/address mismatch")
	}
	if docs[2].Balance != "Nachzahlung 100,00 €" || !strings.Contains(strings.Join(docs[2].Address, " "), "Anschrift fehlt") {
		t.Fatal(docs[2])
	}
	run.Result.Units[0].BalanceCents = 0
	docs, _ = Documents(run, "b", "zoe@example.com")
	if docs[0].Balance != "Ausgeglichen 0,00 €" {
		t.Fatal(docs)
	}
	for _, pair := range [][2]string{{"missing", "owner@example.com"}, {"a", "zoe@example.com"}, {"", "owner@example.com"}, {"a", ""}} {
		if _, err := Documents(run, pair[0], pair[1]); !errors.Is(err, ErrNotFound) {
			t.Fatal(pair, err)
		}
	}
	run.Input.Parties = nil
	if _, err := Documents(run, "", ""); !errors.Is(err, ErrNotFound) {
		t.Fatal("old run silently supplemented")
	}
}

func TestByteIdenticalPDFAfterStoredJSONRoundTrip(t *testing.T) {
	run := fixture()
	raw, _ := json.Marshal(run)
	var loaded store.AnnualStatementRun
	if err := json.Unmarshal(raw, &loaded); err != nil {
		t.Fatal(err)
	}
	first, err := Render(run, "", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TZ", "Pacific/Honolulu")
	second, err := Render(loaded, "", "")
	if err != nil || !bytes.Equal(first, second) {
		t.Fatal("PDF depends on ambient state", err)
	}
	if !bytes.Contains(first, []byte(`/WinAnsiEncoding`)) || !bytes.Contains(first, []byte(`keine Rechtsauskunft nach WEG/MRG`)) {
		t.Fatal("PDF missing encoding/draft notice")
	}
	if path := os.Getenv("HAUSV_PDF_TEST_OUTPUT"); path != "" {
		if err := os.WriteFile(path, first, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLongDocumentPaginationDoesNotDropRowsOrFooter(t *testing.T) {
	run := fixture()
	row := run.Result.Units[1].Costs[0]
	row.Name = strings.Repeat("ÜberlangeKostenbezeichnung", 6)
	for range 30 {
		run.Result.Units[1].Costs = append(run.Result.Units[1].Costs, row)
	}
	docs, _ := Documents(run, "a", "owner@example.com")
	pages := docs[0].Pages()
	if len(pages) < 3 {
		t.Fatal("no pagination")
	}
	for _, page := range pages {
		if len(page.Lines) > 43 || len(page.Footer) < 3 || page.Footer[0] != DraftNotice {
			t.Fatalf("bad page: %+v", page)
		}
		for _, line := range page.Lines {
			if len([]rune(line.Text)) > 91 {
				t.Fatalf("line overflow: %s", line.Text)
			}
		}
	}
}

func TestStoredLargeAmountsRetainEveryCent(t *testing.T) {
	run := fixture()
	run.Result.Units[1].AllocatedCents = 9_223_372_036_854_775_807
	docs, err := Documents(run, "a", "owner@example.com")
	if err != nil || docs[0].Total != "92.233.720.368.547.758,07 €" {
		t.Fatal(docs, err)
	}
}

func TestMeasuredVectorTotalCannotOverflowOrRoundStoredMicros(t *testing.T) {
	run := fixture()
	vector := run.Input.Consumption["heizung"]
	for i := range vector.Units {
		vector.Units[i].ValueMicros = 9223372036854775807
	}
	run.Input.Consumption["heizung"] = vector
	docs, err := Documents(run, "a", "owner@example.com")
	if err != nil {
		t.Fatal(err)
	}
	want := "9.223.372.036.854,775807 kWh / 18.446.744.073.709,551614 kWh"
	if !strings.Contains(docs[0].Costs[1].Measurements[0], want) {
		t.Fatal(docs[0].Costs[1].Measurements)
	}
}

// HAUSV-639: the combined document pages units the way the register reads
// them — parking last, labels naturally — not by ID string.
func TestDocumentsFollowTheRegisterOrder(t *testing.T) {
	run := fixture()
	run.Input.Units = []store.AnnualStatementRunUnitIdentity{
		{ID: "a", Label: "Top 1", UnitType: store.UnitTypeResidential},
		{ID: "b", Label: "Top 2", UnitType: store.UnitTypeResidential},
		{ID: "0-stellplatz", Label: "Stellplatz 1", UnitType: store.UnitTypeParking},
		{ID: "c", Label: "Top 10", UnitType: store.UnitTypeResidential},
	}
	run.Input.Parties = append(run.Input.Parties,
		store.AnnualStatementRunParty{UnitID: "0-stellplatz", ID: "p@example.com", Name: "Parker", Owner: true},
		store.AnnualStatementRunParty{UnitID: "c", ID: "ten@example.com", Name: "Zehn", Owner: true})
	run.Result.Units = append(run.Result.Units,
		store.AnnualStatementRunUnit{UnitID: "0-stellplatz", Label: "Stellplatz 1"},
		store.AnnualStatementRunUnit{UnitID: "c", Label: "Top 10"})
	docs, err := Documents(run, "", "")
	if err != nil {
		t.Fatal(err)
	}
	var labels []string
	for _, doc := range docs {
		if len(labels) == 0 || labels[len(labels)-1] != doc.UnitLabel {
			labels = append(labels, doc.UnitLabel)
		}
	}
	if got := strings.Join(labels, ", "); got != "Top 1, Top 2, Top 10, Stellplatz 1" {
		t.Fatalf("document order = %q", got)
	}
}
