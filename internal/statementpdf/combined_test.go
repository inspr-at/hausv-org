package statementpdf

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/pdf"
	"github.com/inspr-at/hausv-org/internal/store"
)

func combinedRun(t *testing.T, regime string) store.AnnualStatementRun {
	t.Helper()
	in := store.AnnualStatementRunInput{
		Presentation: fixture().Input.Presentation,
		Period:       store.AnnualStatementPeriod{Year: 2025, StartsOn: "2025-01-01", EndsOn: "2025-12-31"},
		Units:        []store.AnnualStatementRunUnitIdentity{{ID: "a", Label: "Top 1"}, {ID: "b", Label: "Top 2"}},
		Parties:      []store.AnnualStatementRunParty{{ID: "owner", UnitID: "a", Name: "Alina Eigentümer", Owner: true}, {ID: "tenant", UnitID: "a", Name: "Matthias Mieter", Renter: true}, {ID: "other", UnitID: "b", Name: "Sophie Bewohner", Owner: true}},
		Structure: store.AnnualStatementPeriodStructure{
			Legal:     store.AnnualStatementLegalSettings{Regime: regime, ShowVAT: true, HeizKGApplies: true, HeatingConsumptionPercent: 70, HeatingPrepayments: map[string]map[string]int64{"a": {"heizung": 0}, "b": {"heizung": 0}}, HeatableAreas: map[string]int{"a": 5000, "b": 5000}, InspectionPlace: "Verwaltungsbüro", InspectionPeriod: "September 2026", InspectionContact: "Vera Verwalter"},
			CostTypes: []store.AnnualStatementCostType{{Key: "heizung", Name: "Heizung", Allocatable: true, AllocationKey: store.AllocationKeyVerbrauch, VATRatePercent: 20}, {Key: "wasser", Name: "Wasser", Allocatable: true, AllocationKey: store.AllocationKeyNutzwert, VATRatePercent: 10}},
			UnitBases: []store.AnnualStatementPeriodUnitBasis{{UnitID: "a", MiteigentumsanteilPPM: 250000, UsableAreaRecorded: true, UsableAreaM2Hundredths: 5000, VacantFrom: "2025-03-01", VacantTo: "2025-05-31"}, {UnitID: "b", MiteigentumsanteilPPM: 750000, UsableAreaRecorded: true, UsableAreaM2Hundredths: 5000}},
		},
		Receipts: []store.AnnualStatementReceipt{
			{ID: "energy", DocumentID: "d1", PeriodYear: 2025, InvoiceDate: "2025-12-31", CostTypeKey: "heizung", HeatingCategory: "energie", AmountCents: 100000, Supplier: "Wärmeversorger"},
			{ID: "operation", DocumentID: "d2", PeriodYear: 2025, InvoiceDate: "2025-12-31", CostTypeKey: "heizung", HeatingCategory: "sonstige_betriebskosten", AmountCents: 20000, Supplier: "Wärmeversorger"},
			{ID: "water", DocumentID: "d3", PeriodYear: 2025, InvoiceDate: "2025-12-31", CostTypeKey: "wasser", AmountCents: 11001, Supplier: "Wasserwerk"},
		},
		Documents:   []store.AnnualStatementRunDocument{{ID: "d1"}, {ID: "d2"}, {ID: "d3"}},
		Prepayments: []store.AnnualStatementPrepayment{{UnitID: "a", PeriodYear: 2025, AmountCents: 3000}, {UnitID: "b", PeriodYear: 2025}},
		Consumption: map[string]store.AnnualStatementConsumptionVector{"heizung": {PeriodYear: 2025, CostTypeKey: "heizung", Units: []store.AnnualStatementUnitConsumption{{UnitID: "a", ValueMicros: 1000000, MeasurementUnit: "kWh"}, {UnitID: "b", ValueMicros: 3000000, MeasurementUnit: "kWh"}}}},
		Reserve:     []store.AnnualStatementReserveEntry{{Kind: store.ReserveKindOpening, AmountCents: 10000}, {Kind: store.ReserveKindContribution, AmountCents: 5000}, {Kind: store.ReserveKindWithdrawal, AmountCents: 2000}, {Kind: store.ReserveKindInterest, AmountCents: 123}},
	}
	result, issues := store.CalculateAnnualStatementRun(in)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	return store.AnnualStatementRun{ID: "combined-" + regime, PeriodYear: 2025, Revision: 1, CalculationVersion: 3, CreatedAt: time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC), Input: in, Result: result}
}

func TestCombinedStatementPDF(t *testing.T) {
	for _, regime := range []string{"weg", "mrg_voll"} {
		t.Run(regime, func(t *testing.T) {
			run := combinedRun(t, regime)
			for _, party := range []string{"owner", "tenant"} {
				docs, err := Documents(run, "a", party)
				if err != nil {
					t.Fatal(err)
				}
				doc := docs[0]
				var texts []string
				for p, page := range doc.Pages() {
					for i, line := range page.Lines {
						texts = append(texts, line.Text)
						w := pdf.TextWidth(line.Text, line.Style, line.Size)
						if line.X < left || line.X+w > left+measure+.01 || line.Y < bottom || line.Y+line.Size > 834 {
							t.Fatalf("page %d bounds: %+v", p, line)
						}
						for _, prior := range page.Lines[:i] {
							if line.X < prior.X+pdf.TextWidth(prior.Text, prior.Style, prior.Size)-.01 && prior.X < line.X+w-.01 && line.Y < prior.Y+prior.Size*.8 && prior.Y < line.Y+line.Size*.8 {
								t.Fatalf("page %d overlap: %+v / %+v", p, prior, line)
							}
						}
					}
				}
				text := strings.Join(texts, "\n")
				for _, want := range []string{"Netto", "USt-Satz", "Heizkostenabrechnung nach § 18 HeizKG", "Verbrauchskosten-Pool", "Hinweise"} {
					if !strings.Contains(text, want) {
						t.Fatalf("missing %q", want)
					}
				}
				if regime == "weg" && !strings.Contains(text, "Endstand: 131,23 €") {
					t.Fatal("missing reserve balance")
				}
				if regime == "mrg_voll" {
					wantTotal, wantHeatNet, wantWaterVAT := "114,05 €", "89,27 €", "0,63 €"
					if party == "tenant" {
						wantTotal, wantHeatNet, wantWaterVAT = "338,45 €", "264,90 €", "1,87 €"
					}
					if doc.Total != wantTotal || doc.Costs[0].Net != wantHeatNet || doc.Costs[1].VAT != wantWaterVAT {
						t.Fatalf("party amounts: %+v", doc)
					}
					if len(doc.VATSummary) != 2 || !strings.Contains(doc.VATSummary[0], "USt "+wantWaterVAT) {
						t.Fatal(doc.VATSummary)
					}
					if party == "owner" && !strings.Contains(text, store.AnnualStatementVacancyLabel) {
						t.Fatal("missing owner vacancy")
					}
				}
				raw, err := Render(run, "a", party)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Contains(raw, []byte("Verbrauchskosten-Pool")) {
					t.Fatal("missing rendered HeizKG")
				}
				writeCombinedPDF(t, regime+"-"+party, raw)
			}
			if regime == "mrg_voll" {
				raw, err := RenderAushang(run)
				if err != nil {
					t.Fatal(err)
				}
				for _, amount := range []string{"1.000,00", "200,00", "1.200,00", "100,01", "10,00", "110,01"} {
					if !bytes.Contains(raw, []byte(amount)) {
						t.Errorf("aushang excludes vacancy in %s", amount)
					}
				}
				writeCombinedPDF(t, "mrg-aushang", raw)
				// A party holding both roles must get one reconciled VAT summary.
				run.Input.Parties[0].Renter = true
				docs, _ := Documents(run, "a", "owner")
				if docs[0].Total != "452,50 €" || !strings.Contains(docs[0].VATSummary[0], "Brutto 27,50 €") || !strings.Contains(docs[0].VATSummary[1], "Brutto 425,00 €") {
					t.Fatal(docs[0])
				}
			}
		})
	}
}

func writeCombinedPDF(t *testing.T, name string, data []byte) {
	t.Helper()
	if out := os.Getenv("HAUSV_TEST_PDF_ARTIFACT_DIR"); out != "" {
		if err := os.MkdirAll(out, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(out, fmt.Sprintf("combined-%s.pdf", name)), data, 0644); err != nil {
			t.Fatal(err)
		}
	}
}
