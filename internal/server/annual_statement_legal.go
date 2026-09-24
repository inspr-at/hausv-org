package server

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
)

func (a *app) saveAnnualStatementLegal(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actor, role, _, ok := a.buildingSettingsContext(w, ac)
	if !ok {
		return
	}
	if r.ParseForm() != nil || ac.repositories.annualStatementPeriods == nil {
		http.Error(w, "Ungültige Eingabe.", 400)
		return
	}
	year, err := strconv.Atoi(r.FormValue("year"))
	legal := store.AnnualStatementLegalSettings{InspectionPlace: r.FormValue("inspection_place"), InspectionPeriod: r.FormValue("inspection_period"), InspectionContact: r.FormValue("inspection_contact"), Regime: r.FormValue("regime"), HeizKGApplies: r.FormValue("heizkg_applies") == "on"}
	legal.HeatingConsumptionPercent = 70
	if raw := r.FormValue("heating_consumption_percent"); raw != "" {
		n, e := strconv.Atoi(raw)
		if e != nil {
			http.Error(w, "Ungültiger Verbrauchsanteil.", 400)
			return
		}
		legal.HeatingConsumptionPercent = n
	}
	legal.NextPrepaymentOn = r.FormValue("next_prepayment_on")
	legal.MonthlyProposals = map[string]map[string]int64{}
	structure, _ := ac.repositories.annualStatementPeriods.Structure(year)
	legal.HeatableAreas = map[string]int{}
	legal.HeatingPrepayments = map[string]map[string]int64{}
	if ac.repositories.units != nil {
		for _, unit := range ac.repositories.units.List() {
			for _, cost := range structure.CostTypes {
				raw := r.FormValue("monthly_proposal_" + cost.Key + "_" + unit.ID)
				if raw == "" {
					continue
				}
				cents, ok := parseAnnualStatementPrepaymentAmount(raw)
				if !ok {
					http.Error(w, "Ungültiger Akontovorschlag.", 400)
					return
				}
				if legal.MonthlyProposals[unit.ID] == nil {
					legal.MonthlyProposals[unit.ID] = map[string]int64{}
				}
				legal.MonthlyProposals[unit.ID][cost.Key] = cents
			}

			for _, key := range []string{"heizung", "warmwasser"} {
				raw := r.FormValue("heating_prepayment_" + key + "_" + unit.ID)
				if raw == "" {
					continue
				}
				cents, ok := parseAnnualStatementPrepaymentAmount(raw)
				if !ok {
					http.Error(w, "Ungültiges Heizkosten-Akonto.", 400)
					return
				}
				if legal.HeatingPrepayments[unit.ID] == nil {
					legal.HeatingPrepayments[unit.ID] = map[string]int64{}
				}
				legal.HeatingPrepayments[unit.ID][key] = cents
			}

			raw := r.FormValue("heatable_area_" + unit.ID)
			if strings.ContainsAny(raw, "+-") {
				http.Error(w, "Ungültige versorgbare Nutzfläche.", 400)
				return
			}
			area, recorded, ok := parseAnnualStatementArea(raw)
			if !ok {
				http.Error(w, "Ungültige versorgbare Nutzfläche.", 400)
				return
			}
			if recorded {
				legal.HeatableAreas[unit.ID] = area
			}
		}
	}
	if err != nil || ac.repositories.annualStatementPeriods.SaveLegal(year, legal) != nil {
		http.Error(w, "Bitte Rechtsgrundlage und Periode prüfen.", 400)
		return
	}
	a.recordAudit(auditEvent{TenantSlug: tenant.Slug, ActorEmail: actor, ActorRole: role, Action: store.AuditActionAnnualPeriodSave, TargetType: "annual-statement-period", TargetID: strconv.Itoa(year), Summary: "Rechtsgrundlage der Periode gespeichert"})
	http.Redirect(w, r, "/app/settings/annual-statement?year="+strconv.Itoa(year)+"#rechtsgrundlage", http.StatusSeeOther)
}
func annualLegalDeadline(period store.AnnualStatementPeriod, legal store.AnnualStatementLegalSettings) string {
	raw := legal.Deadline(period)
	if raw == "" {
		return "Laut Vertrag"
	}
	at, _ := time.Parse("2006-01-02", raw)
	return at.Format("02.01.2006")
}

// Proposed instalments are not evidence of money received. The page only
// prefills unsaved fields; existing recorded payments always win.
func annualStatementPrepaymentPrefill(repo store.AnnualStatementRunRepository, year int) map[string]int64 {
	out := map[string]int64{}
	if repo == nil {
		return out
	}
	runs, err := repo.List(year - 1)
	if err != nil {
		return out
	}
	for _, run := range runs {
		if run.Approval == nil {
			continue
		}
		invalid := map[string]bool{}
		for _, p := range run.Result.Proposals {
			if p.Missing || p.MonthlyCents < 0 || p.MonthlyCents > (math.MaxInt64-out[p.UnitID])/12 {
				invalid[p.UnitID] = true
				continue
			}
			out[p.UnitID] += p.MonthlyCents * 12
		}
		for id := range invalid {
			delete(out, id)
		}
		break
	}
	return out
}
