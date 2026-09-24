package statementpdf

import (
	"strings"
	"testing"
)

func TestReceiptInspectionAppendix(t *testing.T) {
	run := fixture()
	run.Input.Structure.Legal.Regime = "mrg_voll"
	run.Input.Structure.Legal.InspectionPlace = "Büro Graz"
	run.Input.Structure.Legal.InspectionPeriod = "01.07. bis 31.07., Mo–Fr 9–12 Uhr"
	run.Input.Structure.Legal.InspectionContact = "verwaltung@example.com"
	run.Input.Receipts[0].Supplier = "Wasserwerke"
	run.Input.Receipts[0].DocumentID = "original-4711"
	run.Input.Receipts[0].InvoiceDate = "2025-04-03"
	docs, err := Documents(run, "a", "owner@example.com")
	if err != nil {
		t.Fatal(err)
	}
	var text string
	for _, p := range docs[0].Pages() {
		for _, l := range p.Lines {
			text += " " + l.Text
		}
	}
	for _, want := range []string{"Einsicht in die Belege", "Büro Graz", "verwaltung@example.com", "Belegverzeichnis", "Wasserwerke", "03.04.2025", "original-4711"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %s", want)
		}
	}
}
