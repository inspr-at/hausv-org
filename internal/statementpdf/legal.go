package statementpdf

import (
	"fmt"
	"github.com/inspr-at/hausv-org/internal/pdf"
	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/view"
	"sort"
	"strings"
	"time"
	_ "time/tzdata"
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
	for i, receipt := range receipts {
		appendix = append(appendix, fmt.Sprintf("Beleg %d · ", i+1)+date(receipt.InvoiceDate)+" · "+fallback(receipt.Supplier)+" · "+names[receipt.CostTypeKey]+" · "+money(receipt.AmountCents))
	}
	return inspection, appendix
}

// Aushang contains house totals and inspection instructions, never party data.
func RenderAushang(run store.AnnualStatementRun) ([]byte, error) {
	if run.Input.Structure.Legal.Regime != "mrg_voll" {
		return nil, ErrNotFound
	}
	d := letterDocument(run, "Liegenschaft")
	d.Title = fmt.Sprintf("Jahresabrechnung %d — Aushang", run.PeriodYear)
	d.Total = money(run.Result.TotalCents)
	d.Inspection, _ = inspectionAppendix(run)
	for _, cost := range run.Input.Structure.CostTypes {
		if !cost.Allocatable {
			continue
		}
		var total int64
		for _, receipt := range run.Input.Receipts {
			if receipt.CostTypeKey == cost.Key {
				total += receipt.AmountCents
			}
		}
		row := CostRow{Name: cost.Name, Amount: money(total)}
		if run.Input.Structure.Legal.ShowVAT {
			var net, vat int64
			var rate int
			for _, unit := range run.Result.Units {
				for _, line := range unit.Costs {
					if line.CostTypeKey != cost.Key {
						continue
					}
					net += line.NetCents
					vat += line.VATCents
					rate = line.VATRatePercent
				}
			}
			row.Net, row.Rate, row.VAT, row.Gross = money(net), fmt.Sprintf("%d %%", rate), money(vat), money(total)
			d.ShowVAT = true
		}
		d.Costs = append(d.Costs, row)
	}
	if d.ShowVAT {
		for _, group := range run.Result.VATGroups {
			d.VATSummary = append(d.VATSummary, fmt.Sprintf("%d %% · Netto %s · USt %s · Brutto %s", group.RatePercent, money(group.NetCents), money(group.VATCents), money(group.GrossCents)))
		}
	}
	return pdf.Pages(d.Pages(), statementPalette), nil
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

func proposalLines(run store.AnnualStatementRun, unitID string) []string {
	if len(run.Result.Proposals) == 0 {
		return nil
	}
	effective := run.Input.Structure.Legal.NextPrepaymentOn
	if effective == "" {
		at := statementDate(run)
		effective = time.Date(at.Year(), at.Month()+1, 1, 0, 0, 0, 0, at.Location()).Format("2006-01-02")
	}
	lines := []string{"Neue monatliche Vorauszahlung ab " + date(effective) + ":"}
	for _, p := range run.Result.Proposals {
		if p.UnitID != unitID {
			continue
		}
		amount := money(p.MonthlyCents)
		if p.Missing {
			amount = "Noch festzulegen"
		}
		basis := p.Basis
		switch basis {
		case "Manuelle Vorausschau / Vereinbarung":
			basis = "vereinbarte monatliche Vorauszahlung"
		case "Vorperiode / 12":
			basis = "Vorauszahlung auf Basis des Vorjahres"
		}
		lines = append(lines, p.Name+": "+amount+" · "+basis)
		if p.AboveTenPercent {
			lines = append(lines, "Hinweis: Der Vorschlag liegt mehr als 10 % über den Vorjahreskosten / 12. Bitte prüfen (§ 21 Abs. 3 MRG).")
		}
	}
	return lines
}
