package statementpdf

import (
	"fmt"

	"github.com/inspr-at/hausv-org/internal/store"
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
	if reserve.AreaIncomplete {
		lines = append(lines, "Hinweis: Die Mindest-Rücklage konnte nicht geprüft werden, weil die Nutzfläche unvollständig ist.")
	} else if reserve.MinimumWarning {
		lines = append(lines, fmt.Sprintf("Hinweis: Die Zuführungen liegen unter der Mindest-Rücklage von %s je m² und Monat (WEG 2002 § 31).", money(store.WEGMinimumReserveCentsPerSquareMetreMonth)))
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
