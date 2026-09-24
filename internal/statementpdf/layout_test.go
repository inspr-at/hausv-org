package statementpdf

import (
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/pdf"
)

func TestLetterLayoutSummaryGridAndPrivateReferences(t *testing.T) {
	run := fixture()
	run.Input.Structure.Legal.Regime = "weg"
	docs, _ := Documents(run, "a", "owner@example.com")
	pages := docs[0].Pages()
	var summaryY, tableY float64
	var columns []float64
	for _, page := range pages {
		for _, line := range page.Lines {
			if strings.Contains(line.Text, run.ID) || strings.Contains(line.Text, "owner@example.com") || strings.Contains(line.Text, "heat-a") {
				t.Fatalf("internal identity outside footer: %q", line.Text)
			}
			if line.Style == pdf.Table {
				t.Fatal("monospaced cost table")
			}
			if line.Text == "Kurzfassung" {
				summaryY = line.Y
			}
			if line.Text == "Kostenart" {
				tableY = line.Y
			}
			if line.Text == "250,00 €" || line.Text == "50,00 €" {
				columns = append(columns, line.X+pdf.TextWidth(line.Text, line.Style, line.Size))
			}
		}
	}
	if summaryY <= tableY || len(columns) != 2 || columns[0] != columns[1] {
		t.Fatalf("summary/table grid: %v %v %v", summaryY, tableY, columns)
	}
	if !strings.Contains(strings.Join(pages[0].Footer, " "), "Ref. "+run.ID) {
		t.Fatal("missing footer reference")
	}
	if docs[0].Title != "Jahresabrechnung 2025 — Top 1" {
		t.Fatal(docs[0].Title)
	}
}

func TestLongLetterFieldsStayInsidePageAndPreserveContent(t *testing.T) {
	run := fixture()
	run.Input.Presentation.Organisation = strings.Repeat("HausverwaltungSehrLangerName", 20)
	run.Input.Presentation.ContactAddress = strings.Repeat("Lange Anschrift\n", 20) + "EndeVerwaltung"
	run.Input.Presentation.ContactEmail = strings.Repeat("lange", 80) + "@example.com"
	run.Input.Presentation.EstateName = strings.Repeat("Liegenschaft", 100) + "EndeLiegenschaft"
	run.Input.Parties[2].Address = strings.Repeat("Lange Anschrift\n", 20) + "EndeEmpfänger"
	run.Result.Units[1].Label = strings.Repeat("LangeEinheit", 100) + "EndeEinheit"
	docs, _ := Documents(run, "a", "owner@example.com")
	pages := docs[0].Pages()
	var content strings.Builder
	for _, page := range pages {
		for _, line := range page.Lines {
			content.WriteString(line.Text)
			if line.X < left || line.X+pdf.TextWidth(line.Text, line.Style, line.Size) > left+measure+.01 || line.Y < bottom || line.Y+line.Size > 834 {
				t.Fatalf("page boundary: %+v", line)
			}
		}
		for _, line := range page.Footer {
			if pdf.TextWidth(line, pdf.Body, 7) > measure {
				t.Fatalf("footer overflow: %s", line)
			}
		}
	}
	for _, marker := range []string{"EndeVerwaltung", "EndeLiegenschaft", "EndeEmpfänger", "EndeEinheit"} {
		if !strings.Contains(content.String(), marker) {
			t.Errorf("lost %s", marker)
		}
	}
}
