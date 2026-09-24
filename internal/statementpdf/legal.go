package statementpdf

import (
	"fmt"
	"github.com/inspr-at/hausv-org/internal/pdf"
	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/view"
	"sort"
	"strings"
	"time"
)

func statementDate(run store.AnnualStatementRun) time.Time {
	at := run.CreatedAt
	if run.Approval != nil {
		at = run.Approval.ApprovedAt
	}
	loc, err := time.LoadLocation("Europe/Vienna")
	if err == nil {
		at = at.In(loc)
	}
	return time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, at.Location())
}
func paymentTerms(run store.AnnualStatementRun, unit store.AnnualStatementRunUnit) []string {
	legal := run.Input.Structure.Legal
	if legal.Regime == "" {
		return nil
	}
	at := statementDate(run)
	terms := []string{"Abrechnungsdatum: " + at.Format("02.01.2006")}
	switch legal.Regime {
	case "mrg_voll":
		months := 1
		if at.Day() >= 5 {
			months = 2
		}
		due := time.Date(at.Year(), at.Month()+time.Month(months), 5, 0, 0, 0, 0, at.Location())
		terms = append(terms, "Betriebskosten: Guthaben oder Nachzahlung zum übernächsten Zinstermin am "+due.Format("02.01.2006")+" (§ 21 Abs. 3 MRG; monatlicher Zinstermin: 5.).")
	case "weg":
		terms = append(terms, "WEG: Ein Guthaben wird auf künftige Vorauszahlungen angerechnet. Eine Nachzahlung ist bis "+store.ShiftStatementDate(at, 2).Format("02.01.2006")+" fällig (§ 34 Abs. 4 WEG).")
	default:
		terms = append(terms, "Betriebskosten: Fälligkeit und Behandlung des Saldos laut Vertrag.")
	}
	if legal.HeizKGApplies {
		terms = append(terms, "HeizKG-Anteil: Guthaben wird bis "+store.ShiftStatementDate(at, 2).Format("02.01.2006")+" zurückgezahlt; eine Nachzahlung ist bis zu diesem Tag fällig (§ 21 HeizKG).")
	}
	return terms
}

func inspectionAppendix(run store.AnnualStatementRun) ([]string, []string) {
	legal := run.Input.Structure.Legal
	if legal.Regime == "" {
		return nil, nil
	}
	fallback := func(value string) string {
		if strings.TrimSpace(value) == "" {
			return "Noch nicht hinterlegt"
		}
		return value
	}
	inspection := []string{"Ort: " + fallback(legal.InspectionPlace), "Zeitraum: " + fallback(legal.InspectionPeriod), "Kontakt: " + fallback(legal.InspectionContact)}
	receipts := append([]store.AnnualStatementReceipt(nil), run.Input.Receipts...)
	sort.Slice(receipts, func(i, j int) bool {
		if receipts[i].InvoiceDate != receipts[j].InvoiceDate {
			return receipts[i].InvoiceDate < receipts[j].InvoiceDate
		}
		return receipts[i].ID < receipts[j].ID
	})
	names := map[string]string{}
	for _, cost := range run.Input.Structure.CostTypes {
		names[cost.Key] = cost.Name
	}
	var appendix []string
	for _, receipt := range receipts {
		appendix = append(appendix, date(receipt.InvoiceDate)+" · "+fallback(receipt.Supplier)+" · "+names[receipt.CostTypeKey]+" · "+money(receipt.AmountCents), "Dokument: "+receipt.DocumentID)
	}
	return inspection, appendix
}

// Aushang contains house totals and inspection instructions, never party data.
func RenderAushang(run store.AnnualStatementRun) ([]byte, error) {
	if run.Input.Structure.Legal.Regime != "mrg_voll" {
		return nil, ErrNotFound
	}
	d := Document{Title: "Jahresabrechnung · Aushang", UnitLabel: "Liegenschaft", Header: []string{run.Input.Presentation.EstateName, run.Input.Presentation.EstateAddress, "Abrechnungsperiode: " + date(run.Input.Period.StartsOn) + " bis " + date(run.Input.Period.EndsOn), run.Input.Structure.Legal.Basis()}, Contact: run.Input.Structure.Legal.InspectionContact, Total: money(run.Result.TotalCents)}
	d.Inspection, _ = inspectionAppendix(run)
	for _, cost := range run.Input.Structure.CostTypes {
		var total int64
		for _, receipt := range run.Input.Receipts {
			if receipt.CostTypeKey == cost.Key {
				total += receipt.AmountCents
			}
		}
		d.Basis = append(d.Basis, cost.Name+": "+money(total))
	}
	if run.Approval != nil {
		d.ApprovalNotice = "Freigegeben: " + timestamp(run.Approval.ApprovedAt) + " · " + run.Approval.Role
	}
	return pdf.Pages(d.Pages(), pdf.Palette{Paper: [3]uint8{247, 243, 234}, Ink: [3]uint8{32, 37, 31}, Accent: [3]uint8{200, 153, 63}}), nil
}

func heatingDetails(run store.AnnualStatementRun, unit store.AnnualStatementRunUnit, cost store.AnnualStatementRunCost) []string {
	legal := run.Input.Structure.Legal
	energy, other, consumed, area := store.AnnualStatementHeatingPools(run.Input, cost.CostTypeKey)
	var totalArea int
	for _, u := range run.Input.Units {
		totalArea += legal.HeatableAreas[u.ID]
	}
	areaText := func(a int) string { return view.FormatDecimal(float64(a)/100, 2) + " m²" }
	return []string{
		"Heizkostenabrechnung nach § 18 HeizKG",
		"Energiekosten gesamt: " + money(energy) + "; sonstige Betriebskosten: " + money(other),
		fmt.Sprintf("Energieaufteilung: %d %% Verbrauch / %d %% versorgbare Nutzfläche", legal.HeatingConsumptionPercent, 100-legal.HeatingConsumptionPercent),
		"Verbrauchskosten-Pool: " + money(consumed) + "; Flächenkosten-Pool einschließlich sonstiger Betriebskosten: " + money(area),
		"Versorgbare Nutzfläche Einheit: " + areaText(legal.HeatableAreas[unit.UnitID]) + "; gesamt: " + areaText(totalArea),
		"Anteil am gemessenen Verbrauch: " + view.FormatDecimal(float64(cost.SharePPM)/10000, 2) + " %; Methode: Zählerdifferenz",
		"Kostenanteil Einheit: " + money(cost.AmountCents),
		"Geleistetes Akonto dieser Heizkostenart: " + money(legal.HeatingPrepayments[unit.UnitID][cost.CostTypeKey]) + "; Saldo (Nachzahlung positiv, Guthaben negativ): " + money(cost.AmountCents-legal.HeatingPrepayments[unit.UnitID][cost.CostTypeKey]),
		"Einwendungen sind binnen sechs Monaten ab Rechnungslegung zu erheben; sonst gilt die Abrechnung als genehmigt (§ 24 HeizKG).",
	}
}
