package view

import (
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
)

// AnnualStatementReserveRateLabel describes the rates saved with a reserve check.
func AnnualStatementReserveRateLabel(rates []store.AnnualStatementReserveMinimumRate) string {
	labels := make([]string, 0, len(rates))
	for _, rate := range rates {
		start, _ := time.Parse("2006-01-02", rate.StartsOn)
		end, _ := time.Parse("2006-01-02", rate.EndsOn)
		labels = append(labels, FormatEURCents(rate.CentsPerSquareMetreMonth)+" je m² und Monat ("+start.Format("02.01.2006")+" bis "+end.Format("02.01.2006")+")")
	}
	return strings.Join(labels, "; ")
}
