package statementpdf

import (
	"github.com/inspr-at/hausv-org/internal/store"
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
