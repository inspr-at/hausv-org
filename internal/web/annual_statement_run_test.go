package web

import (
	"bytes"
	"context"
	"html"
	"net/url"
	"strings"
	"testing"
)

func TestAnnualStatementRunPanelBlockedAndHistoricalResults(t *testing.T) {
	var body bytes.Buffer
	blocked := AnnualStatementRunView{Year: 2025, Issues: []string{"Top <2>: Akonto fehlt."}}
	if err := AnnualStatementRunPanel(blocked).Render(context.Background(), &body); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Abrechnungslauf gesperrt", "Top &lt;2&gt;: Akonto fehlt.", "disabled", "Für alle Einheiten berechnen"} {
		if !strings.Contains(body.String(), want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(body.String(), "data-annual-statement-run") {
		t.Fatal("partial result rendered")
	}
	body.Reset()
	ready := AnnualStatementRunView{Year: 2025, Ready: true, ID: "run-1", Revision: 1, Total: "100,00 €", Excluded: "0,00 €", Units: []AnnualStatementRunUnitView{{Label: "Top 1", Allocated: "25,00 €", Prepaid: "30,00 €", Balance: "Guthaben 5,00 €"}, {Label: "Top 2", Allocated: "75,00 €", Prepaid: "0,00 €", Balance: "Nachzahlung 75,00 €"}}}
	if err := AnnualStatementRunPanel(ready).Render(context.Background(), &body); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Bereit zur Berechnung", `data-annual-statement-run="run-1"`, "Guthaben 5,00 €", "Nachzahlung 75,00 €", "Top 1", "Top 2"} {
		if !strings.Contains(body.String(), want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestAnnualStatementRunPanelEscapesHistoryQuery(t *testing.T) {
	id := "run&year=1999#other?x=1"
	var body bytes.Buffer
	data := AnnualStatementRunView{Year: 2025, History: []AnnualStatementRunLinkView{{ID: id, Label: "Lauf 1"}}}
	if err := AnnualStatementRunPanel(data).Render(context.Background(), &body); err != nil {
		t.Fatal(err)
	}
	want := `href="` + html.EscapeString("/app/settings/annual-statement?year=2025&run="+url.QueryEscape(id)+"#abrechnungslauf") + `"`
	if !strings.Contains(body.String(), want) {
		t.Fatalf("missing escaped history URL %q", want)
	}
}

func TestAnnualStatementRunPanelPDFLinks(t *testing.T) {
	var body bytes.Buffer
	data := AnnualStatementRunView{ID: "stored-run", AllPDFURL: "/app/settings/annual-statement/runs/stored-run/pdf", Units: []AnnualStatementRunUnitView{{Label: "Top 1", PDFs: []AnnualStatementRunPDFView{{Label: "Anna <Groß>", URL: "/app/settings/annual-statement/runs/stored-run/pdf?party=anna%40example.com&unit=top-1"}, {Label: "Mieter", URL: "/app/settings/annual-statement/runs/stored-run/pdf?party=mieter%40example.com&unit=top-1"}}}}}
	if err := AnnualStatementRunPanel(data).Render(context.Background(), &body); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Alle Dokumente (PDF)", "PDF für Anna &lt;Groß&gt;", "PDF für Mieter", "party=anna%40example.com&amp;unit=top-1", "download"} {
		if !strings.Contains(body.String(), want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestAnnualStatementRunArchiveActionAndStatus(t *testing.T) {
	data := AnnualStatementRunView{ID: "run-1", AllPDFURL: "/pdf", ArchiveAction: "/app/settings/annual-statement/runs/run-1/archive", ArchiveURL: "/app/dokumente?q=Abrechnung"}
	var body bytes.Buffer
	if err := AnnualStatementRunPanel(data).Render(context.Background(), &body); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`method="post" action="/app/settings/annual-statement/runs/run-1/archive"`, "Im Archiv ablegen", "Sichtbar nur für die Verwaltung"} {
		if !strings.Contains(body.String(), want) {
			t.Errorf("missing %q", want)
		}
	}
	data.ArchivedAt, data.ArchiveCount = "06.09.2026 20:00", 31
	body.Reset()
	if err := AnnualStatementRunPanel(data).Render(context.Background(), &body); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Archiviert am 06.09.2026 20:00 · 31 Dokumente", `href="/app/dokumente?q=Abrechnung"`} {
		if !strings.Contains(body.String(), want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(body.String(), "Im Archiv ablegen") {
		t.Fatal("completed archive has create action")
	}
}

func TestAnnualStatementDeliveryPanel(t *testing.T) {
	data := AnnualStatementRunView{ID: "run", SendAction: "/app/settings/annual-statement/runs/run/send", MailMode: "Postausgang als Datei — Testmodus", SendIssue: "Zuerst im Archiv ablegen", DeliverySummary: "1 gesendet · 1 fehlgeschlagen · 2 übersprungen", Deliveries: []AnnualStatementDeliveryView{{Party: "Anna <Groß>", Recipient: "anna@example.com", Time: "06.09.2026 20:00 CEST", Status: "Fehlgeschlagen", Error: "Datei <fehlt>"}}}
	var body bytes.Buffer
	if err := AnnualStatementRunPanel(data).Render(t.Context(), &body); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`method="post" action="/app/settings/annual-statement/runs/run/send"`, `disabled`, "Per E-Mail senden", "Postausgang als Datei — Testmodus", "Zuerst im Archiv ablegen", "Versandprotokoll", "Adresse", "Zeit", "Status", "Fehler", "Anna &lt;Groß&gt;", "Datei &lt;fehlt&gt;", "1 gesendet · 1 fehlgeschlagen · 2 übersprungen"} {
		if !strings.Contains(body.String(), want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestAnnualStatementRunCompactRowsKeepPartyDocuments(t *testing.T) {
	data := AnnualStatementRunView{ID: "run", Units: []AnnualStatementRunUnitView{{Label: "Top 1", Costs: []AnnualStatementRunCostView{{Name: "Wasser", Amount: "12,00 €"}}, PDFs: []AnnualStatementRunPDFView{
		{Label: "Anna <Groß>", URL: "/pdf?party=anna&unit=top-1", ArchiveURL: "/app/dokumente#document-anna"},
		{Label: "Anna <Groß>", URL: "/pdf?party=other&unit=top-1", ArchiveURL: "/app/dokumente#document-other"},
	}}}}
	var body bytes.Buffer
	if err := AnnualStatementRunPanel(data).Render(t.Context(), &body); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`<table class="annual-units" role="table">`, `<th role="columnheader" scope="col">Partei</th>`, `<details class="annual-costs">`, `PDF für Anna &lt;Groß&gt; · Top 1`, `Archiv für Anna &lt;Groß&gt; · Top 1`, `Wasser`, `12,00 €`} {
		if !strings.Contains(body.String(), want) {
			t.Errorf("missing %q", want)
		}
	}
	for _, id := range []string{"anna", "other"} {
		if strings.Count(body.String(), `href="/app/dokumente#document-`+id+`"`) != 1 {
			t.Errorf("archive link for %s missing or duplicated", id)
		}
	}
	if strings.Contains(body.String(), `class="readonly"`) || strings.Contains(body.String(), `<details class="annual-costs" open`) {
		t.Fatal("result must be compact and collapsed")
	}
}

func TestAnnualStatementPreparationCollapsesOnlyWithoutFeedback(t *testing.T) {
	for _, tc := range []struct {
		name string
		data AnnualStatementPageData
		open bool
	}{
		{"first preparation", AnnualStatementPageData{}, true},
		{"saved result", AnnualStatementPageData{Run: AnnualStatementRunView{ID: "run"}}, false},
		{"correction feedback", AnnualStatementPageData{Run: AnnualStatementRunView{ID: "run"}, ReceiptMsg: "Betrag korrigiert"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var body bytes.Buffer
			if err := AnnualStatementBody(tc.data).Render(t.Context(), &body); err != nil {
				t.Fatal(err)
			}
			if got := strings.Contains(body.String(), `class="annual-preparation" open`); got != tc.open {
				t.Fatalf("open=%v want %v", got, tc.open)
			}
		})
	}
}

func TestAnnualStatementPreparationPrecedesSavedRunsHAUSV660(t *testing.T) {
	var body bytes.Buffer
	data := AnnualStatementPageData{Year: 2025, Run: AnnualStatementRunView{
		Year: 2025, Ready: true, ID: "run-2025", Revision: 1,
		History: []AnnualStatementRunLinkView{{ID: "run-2025", Label: "Lauf 1"}},
	}}
	if err := AnnualStatementBody(data).Render(t.Context(), &body); err != nil {
		t.Fatal(err)
	}
	html := body.String()
	previous := -1
	for _, marker := range []string{`id="abrechnungslauf"`, "Abrechnungslauf berechnen", "Für alle Einheiten berechnen</button></form>", `<details class="annual-preparation"`, "Grundlagen und Belege prüfen oder bearbeiten · 2025", "Kostenarten, Einheiten, Parteien, Periode", "Gespeicherte Läufe", `data-annual-statement-run="run-2025"`} {
		pos := strings.Index(html, marker)
		if pos <= previous {
			t.Fatalf("%q missing or out of order", marker)
		}
		previous = pos
	}
	if !strings.Contains(html, ".annual-preparation>summary::after") || !strings.Contains(html, ".annual-preparation[open]>summary::after") {
		t.Fatal("preparation must have an expanding chevron")
	}
}
