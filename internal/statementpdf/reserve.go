package statementpdf

import (
	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/view"
)

// ReserveLines is the party-statement Rücklage block for one unit.
// An empty result means this run has no WEG Rücklage snapshot.
func ReserveLines(run store.AnnualStatementRun, unitID string) []string {
	if run.Input.Structure.Legal.Regime != "weg" || run.Result.Reserve == nil {
		return nil
	}
	reserve := run.Result.Reserve
	lines := []string{
		"Anfangsstand: " + money(reserve.OpeningCents),
		"Zuführungen: " + money(reserve.ContributionCents),
		"Entnahmen: " + money(reserve.WithdrawalCents),
		"Zinsen: " + money(reserve.InterestCents),
		"Endstand: " + money(reserve.ClosingCents),
	}
	share, found := reserveShare(reserve, unitID)
	if found {
		lines = append(lines, "Anteil dieser Einheit (MEA): "+money(share))
	} else {
		lines = append(lines, "Anteil dieser Einheit (MEA): nicht ermittelbar")
	}
	if reserve.ClosingMismatch {
		lines = append(lines, "Hinweis: Eine gebuchte Endstand-Kontrolle weicht vom errechneten Endstand ab.")
	}
	if reserve.MinimumUnavailable {
		lines = append(lines, "Hinweis: Die Mindest-Rücklage konnte für diesen Zeitraum nicht vollständig geprüft werden.")
	} else if reserve.AreaIncomplete {
		lines = append(lines, "Hinweis: Die Mindest-Rücklage konnte nicht geprüft werden, weil die Nutzfläche unvollständig ist.")
	} else if run.CalculationVersion < store.AnnualStatementCalculationVersionReserveRates {
		// Preserve the text of historical PDFs alongside their v1–v5 calculation.
		if reserve.MinimumWarning {
			lines = append(lines, "Hinweis: Die Zuführungen liegen unter der Mindest-Rücklage von 1,12 € je m² und Monat (WEG 2002 § 31).")
		}
	} else if len(reserve.MinimumRates) == 0 {
		lines = append(lines, "Vor 01.07.2022 gab es keinen gesetzlichen Mindestbetrag je m². Eine angemessene Rücklage war dennoch zu bilden.")
	} else if reserve.MinimumWarning {
		lines = append(lines, "Hinweis: Die Zuführungen liegen unter der Mindest-Rücklage: "+view.AnnualStatementReserveRateLabel(reserve.MinimumRates)+" (WEG 2002 § 31).")
	} else {
		lines = append(lines, "Mindest-Rücklage: "+view.AnnualStatementReserveRateLabel(reserve.MinimumRates)+".")
	}
	return lines
}

func reserveShare(reserve *store.AnnualStatementReserveResult, unitID string) (int64, bool) {
	for _, share := range reserve.Shares {
		if share.UnitID == unitID {
			return share.AmountCents, true
		}
	}
	return 0, false
}
