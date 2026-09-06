package server

import (
	"fmt"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/web"
)

type annualStatementConsumptionData struct {
	View    web.AnnualStatementConsumptionView
	Vectors map[string]store.AnnualStatementConsumptionVector
}

// Reading a statement never ingests or updates evidence. Only exact period
// boundary facts may produce a consumption basis; latest counters cannot.
func annualStatementConsumption(repository store.AnnualStatementConsumptionRepository, period store.AnnualStatementPeriod, units []store.Unit, costTypes []store.AnnualStatementCostType) annualStatementConsumptionData {
	out := annualStatementConsumptionData{Vectors: map[string]store.AnnualStatementConsumptionVector{}}
	location, locationErr := time.LoadLocation("Europe/Vienna")
	ids := make([]string, len(units))
	for i, unit := range units {
		ids[i] = unit.ID
	}
	used := map[string]bool{}
	for _, cost := range costTypes {
		used[cost.Key] = cost.Allocatable && cost.AllocationKey == store.AllocationKeyVerbrauch
	}
	for _, definition := range []struct{ key, label string }{{"heizung", "Heizung"}, {"warmwasser", "Warmwasser"}} {
		if !used[definition.key] {
			continue
		}
		group := web.AnnualStatementConsumptionGroupView{Kind: definition.key, Label: definition.label, SourceLabel: "Zählerdifferenz der Abrechnungsperiode (Europe/Vienna)"}
		report := store.AnnualStatementConsumptionReport{Vector: store.AnnualStatementConsumptionVector{PeriodYear: period.Year, CostTypeKey: definition.key}}
		if repository != nil && period.Year != 0 && len(units) > 0 && locationErr == nil {
			loaded, err := repository.ConsumptionReport(period, definition.key, ids, location)
			if err == nil {
				report = loaded
			}
		}
		vector := report.Vector
		out.Vectors[definition.key] = vector
		shares, complete := store.AnnualStatementConsumptionShares(vector, units)
		group.Complete = complete
		byUnit := map[string]int{}
		for _, share := range shares {
			byUnit[store.NormalizeUnitID(share.UnitID)] = share.SharePPM
		}
		for _, unit := range units {
			row := web.AnnualStatementConsumptionRowView{Label: unit.Label, Value: "–", Share: "–", Explanation: "Anfangs- und Endmessung der gewählten Periode fehlen."}
			for _, item := range report.MeasuredUnits {
				if item.UnitID == store.NormalizeUnitID(unit.ID) {
					row.Measured = true
					row.Value = formatAnnualStatementConsumption(item)
					row.Share = formatAnnualStatementShare(byUnit[item.UnitID], complete)
					row.Explanation = annualStatementDateRange(period.StartsOn, period.EndsOn)
					group.MeasuredCount++
				}
			}
			for _, gap := range vector.Gaps {
				if gap.UnitID == store.NormalizeUnitID(unit.ID) {
					row.Explanation = annualStatementConsumptionGapMessage(gap.Reason)
				}
			}
			group.Rows = append(group.Rows, row)
		}
		group.GapCount = len(units) - group.MeasuredCount
		out.View.Groups = append(out.View.Groups, group)
	}
	return out
}

func formatAnnualStatementConsumption(item store.AnnualStatementUnitConsumption) string {
	return fmt.Sprintf("%d,%06d %s", item.ValueMicros/1_000_000, item.ValueMicros%1_000_000, item.MeasurementUnit)
}

func annualStatementConsumptionGapMessage(reason store.AnnualStatementConsumptionGapReason) string {
	switch reason {
	case store.ConsumptionGapMissingSourceMapping:
		return "Kein gespeicherter Messnachweis dieser Kostenart zugeordnet."
	case store.ConsumptionGapMissingStartEvidence:
		return "Anfangsmessung der Abrechnungsperiode fehlt."
	case store.ConsumptionGapMissingEndEvidence:
		return "Endmessung der Abrechnungsperiode fehlt."
	case store.ConsumptionGapAmbiguousSource:
		return "Mehrere Messquellen; die Zuordnung ist nicht eindeutig."
	case store.ConsumptionGapAmbiguousMeasurementUnit:
		return "Die Maßeinheit ist unbekannt oder widersprüchlich."
	case store.ConsumptionGapCounterReset:
		return "Zählerstand ist gesunken; Zählerwechsel oder Rücksetzung ungeklärt."
	default:
		return "Messgrundlage ist ungeklärt."
	}
}

func annualStatementConsumptionAllocationViews(costTypes []store.AnnualStatementCostType, units []store.Unit, consumption annualStatementConsumptionData) []web.AnnualStatementAllocationPreviewView {
	out := []web.AnnualStatementAllocationPreviewView{}
	for _, costType := range costTypes {
		if !costType.Allocatable || costType.AllocationKey != store.AllocationKeyVerbrauch {
			continue
		}
		preview := web.AnnualStatementAllocationPreviewView{Key: store.AllocationKeyVerbrauch, Label: "Verbrauch · " + costType.Name, CostTypes: costType.Name, Blocked: true, Sourceless: true,
			TotalNotice: "Ohne vollständige, vergleichbare Periodenmessungen für jede Einheit bleibt diese Kostenart blockiert. Es wird kein Verbrauch geschätzt."}
		for _, group := range consumption.View.Groups {
			if group.Kind != costType.Key {
				continue
			}
			preview.Blocked = !group.Complete
			preview.Sourceless = group.MeasuredCount == 0
			for _, row := range group.Rows {
				preview.Shares = append(preview.Shares, web.AnnualStatementUnitShareView{Label: row.Label, Basis: row.Value, Share: row.Share, Mapped: row.Measured})
				if !row.Measured {
					preview.UnmappedUnits = strings.TrimPrefix(preview.UnmappedUnits+", "+row.Label, ", ")
				}
			}
			if group.Complete {
				preview.TotalNotice = "Gemessene Zählerdifferenzen der gewählten Abrechnungsperiode."
			}
		}
		if len(preview.Shares) == 0 {
			preview.TotalNotice = "Für diese Kostenart ist keine Messregel hinterlegt. Der Abrechnungslauf bleibt blockiert."
			for _, unit := range units {
				preview.Shares = append(preview.Shares, web.AnnualStatementUnitShareView{Label: unit.Label, Basis: "fehlt", Share: "–"})
			}
		}
		out = append(out, preview)
	}
	return out
}
