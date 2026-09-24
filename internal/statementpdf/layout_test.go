package statementpdf

import (
	"bytes"
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/pdf"
)

func TestOptionalLetterheadFieldsAreOmittedWithoutSpacing(t *testing.T) {
	run := fixture()
	run.Input.Presentation.ContactAddress = "\n Musterstraße 12\n \n8010 Graz\n"
	docs, err := Documents(run, "a", "owner@example.com")
	if err != nil {
		t.Fatal(err)
	}
	complete := docs[0]
	for _, line := range complete.Pages()[0].Lines {
		if line.Text == "Vera Verwalter" && line.Y != 764 {
			t.Fatalf("empty sender address line reserved space: %+v", line)
		}
	}
	run.Input.Presentation.ContactAddress = " \n "
	run.Input.Presentation.ContactName = "\t"
	run.Input.Presentation.ContactPhone = " "
	run.Input.Parties[2].Name = "\n"
	run.Input.Parties[2].Address = " "
	docs, _ = Documents(run, "a", "owner@example.com")
	if len(docs[0].Sender) != 2 || len(docs[0].Address) != 1 {
		t.Fatalf("optional field reserved a line: sender=%q recipient=%q", docs[0].Sender, docs[0].Address)
	}
	for _, line := range docs[0].Pages()[0].Lines {
		if line.Text == "verwaltung@musterstadt.example" && line.Y != 786 {
			t.Fatalf("omitted sender fields reserved space: %+v", line)
		}
	}
	// No sender at all must also work on continuation pages and the house notice.
	run.Input.Presentation.Organisation = ""
	run.Input.Presentation.ContactEmail = ""
	run.Input.Presentation.EstateName = ""
	run.Input.Presentation.EstateAddress = ""
	run.Input.Structure.Legal.Regime = "mrg_voll"
	docs, _ = Documents(run, "a", "owner@example.com")
	if len(docs[0].Sender) != 0 || docs[0].Contact != "" {
		t.Fatal("invented sender")
	}
	for _, page := range docs[0].Pages() {
		for _, line := range page.Lines {
			if strings.TrimSpace(line.Text) == "" || strings.Contains(line.Text, "fehlt") || line.Text == "Liegenschaft" {
				t.Fatalf("empty/placeholder letter field: %q", line.Text)
			}
		}
		for _, line := range page.Footer {
			if strings.TrimSpace(line) == "" {
				t.Fatal("empty footer line")
			}
		}
	}
	for _, render := range []func() ([]byte, error){func() ([]byte, error) { return Render(run, "", "") }, func() ([]byte, error) { return RenderAushang(run) }} {
		raw, err := render()
		if err != nil || bytes.Contains(raw, []byte("fehlt")) || bytes.Contains(raw, []byte("TODO")) {
			t.Fatal("placeholder in rendered customer document", err)
		}
	}
}

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
