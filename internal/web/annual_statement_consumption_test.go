package web

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestAnnualStatementConsumptionRendersMeasuredAndMissingBasis(t *testing.T) {
	data := AnnualStatementPageData{Consumption: AnnualStatementConsumptionView{Groups: []AnnualStatementConsumptionGroupView{{Kind: "heizung", Label: "Heizung", Rows: []AnnualStatementConsumptionRowView{
		{Label: "Top 1", Value: "1,000000 kWh", Share: "–", Measured: true, Explanation: "01.01.2025 – 31.12.2025"},
		{Label: "Top <2>", Value: "–", Share: "–", Explanation: "Endmessung der Abrechnungsperiode fehlt."},
	}}}}}
	var body bytes.Buffer
	if err := AnnualStatementBody(data).Render(context.Background(), &body); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`data-consumption-kind="heizung"`, "gemessen", "nicht gemessen", "1,000000 kWh", "Top &lt;2&gt;", "Endmessung der Abrechnungsperiode fehlt.", "Keine vollständige Verteilungsbasis", "Wärmepumpe (heat-pump) liefert die Messwerte für Heizung", "Warmwassergerät (hot-water) und Durchlauferhitzer (instant-water-heater) liefern die Messwerte für Warmwasser"} {
		if !strings.Contains(body.String(), want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(body.String(), "0,00 %") || strings.Contains(body.String(), "Vollständig gemessen") {
		t.Fatal("missing basis rendered as complete")
	}
}
