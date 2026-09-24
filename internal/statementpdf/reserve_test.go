package statementpdf

import (
	"bytes"
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/pdf"
	"github.com/inspr-at/hausv-org/internal/store"
)

func TestReserveRenderedInMeasuredLetter(t *testing.T) {
	run := fixture()
	run.Input.Structure.Legal.Regime = "weg"
	run.Result.Reserve = &store.AnnualStatementReserveResult{
		OpeningCents: 10000, ContributionCents: 5000, WithdrawalCents: 2000,
		InterestCents: 123, ClosingCents: 13123, MinimumWarning: true,
		Shares: []store.AnnualStatementReserveShare{{UnitID: "a", AmountCents: 3281}},
	}
	docs, err := Documents(run, "a", "owner@example.com")
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, page := range docs[0].Pages() {
		for _, line := range page.Lines {
			lines = append(lines, line.Text)
			if line.Text == "Rücklage" && (line.X != left || line.Style != pdf.Strong) {
				t.Fatalf("reserve not on letter grid: %+v", line)
			}
		}
	}
	text := strings.Join(lines, "\n")
	if !(strings.Index(text, "Kostenart") < strings.Index(text, "Rücklage") && strings.Index(text, "Rücklage") < strings.Index(text, "Hinweise")) {
		t.Fatalf("reserve section order: %s", text)
	}
	raw, err := Render(run, "a", "owner@example.com")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`R\374cklage`, "Anfangsstand: 100,00", `Zuf\374hrungen: 50,00`, "Entnahmen: 20,00", "Zinsen: 1,23", "Endstand: 131,23", "Anteil dieser Einheit", "32,81", `Mindest-R\374cklage`} {
		if !bytes.Contains(raw, []byte(want)) {
			t.Errorf("rendered PDF misses %q", want)
		}
	}
	run.Input.Structure.Legal.Regime = "mrg_voll"
	if lines := ReserveLines(run, "a"); len(lines) != 0 {
		t.Fatal("MRG reserve", lines)
	}
}
