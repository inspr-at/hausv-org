package statementpdf

import (
	"bytes"
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/store"
)

func TestVATOffKeepsTheGrossCostTable(t *testing.T) {
	run := fixture()
	first, err := Render(run, "a", "owner@example.com")
	if err != nil {
		t.Fatal(err)
	}
	run.Input.Structure.CostTypes[0].VATRatePercent = 10
	run.Input.Structure.CostTypes[1].VATRatePercent = 20
	second, err := Render(run, "a", "owner@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("stored rates without the period switch changed the PDF")
	}
	docs, err := Documents(run, "a", "owner@example.com")
	if err != nil {
		t.Fatal(err)
	}
	var text []string
	for _, line := range docs[0].Pages()[0].Lines {
		text = append(text, line.Text)
	}
	joined := strings.Join(text, "\n")
	if !strings.Contains(joined, "Gesamtkosten") || strings.Contains(joined, "Netto") || strings.Contains(joined, "USt-Satz") {
		t.Fatalf("off table changed: %s", joined)
	}
}

func TestVATOnShowsRateColumnsAndSummary(t *testing.T) {
	run := fixture()
	run.CalculationVersion = store.AnnualStatementCalculationVersionVAT
	run.Input.Structure.Legal.ShowVAT = true
	run.Input.Structure.Legal.Regime = "weg"
	for i := range run.Result.Units {
		var groups = map[int]*store.AnnualStatementVATGroup{}
		for j := range run.Result.Units[i].Costs {
			line := &run.Result.Units[i].Costs[j]
			rate := 10
			if line.CostTypeKey == "heizung" {
				rate = 20
			}
			line.VATRatePercent = rate
			line.VATCents = line.AmountCents * int64(rate) / int64(100+rate)
			line.NetCents = line.AmountCents - line.VATCents
			group := groups[rate]
			if group == nil {
				group = &store.AnnualStatementVATGroup{RatePercent: rate}
				groups[rate] = group
			}
			group.NetCents += line.NetCents
			group.VATCents += line.VATCents
			group.GrossCents += line.AmountCents
		}
		for _, rate := range []int{10, 20} {
			run.Result.Units[i].VAT = append(run.Result.Units[i].VAT, *groups[rate])
		}
	}
	docs, err := Documents(run, "a", "owner@example.com")
	if err != nil {
		t.Fatal(err)
	}
	var text []string
	for _, page := range docs[0].Pages() {
		for _, line := range page.Lines {
			text = append(text, line.Text)
		}
	}
	joined := strings.Join(text, "\n")
	for _, want := range []string{"Netto", "USt-Satz", "USt", "Brutto", "10 % · Netto", "20 % · Netto"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in %s", want, joined)
		}
	}
	for _, line := range text {
		if line == "Gesamtkosten" {
			t.Fatal("gross-only heading remained")
		}
	}
}
