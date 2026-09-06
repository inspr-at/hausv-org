package server

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/web"
)

func (a *app) createAnnualStatementRun(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actor, role, _, ok := a.buildingSettingsContext(w, ac)
	if !ok {
		return
	}
	if r.ParseForm() != nil {
		http.Error(w, "Ungültige Formulardaten.", http.StatusBadRequest)
		return
	}
	if ac.repositories.annualStatementRuns == nil {
		http.Error(w, "Abrechnungslauf derzeit nicht verfügbar.", http.StatusServiceUnavailable)
		return
	}
	year, err := strconv.Atoi(r.FormValue("year"))
	if err != nil || year < 1 || year > 9999 {
		http.Error(w, "Bitte eine gespeicherte Abrechnungsperiode wählen.", http.StatusBadRequest)
		return
	}
	run, err := ac.repositories.annualStatementRuns.Create(year, actor, time.Now())
	status := "created"
	if err != nil {
		var blocked *store.AnnualStatementRunBlockedError
		if errors.As(err, &blocked) {
			status = "blocked"
		} else {
			status = "error"
			logError("annual statement run failed", err, "tenant", tenant.Slug, "year", year)
			var conflict interface{ SQLState() string }
			if errors.As(err, &conflict) && (conflict.SQLState() == "40001" || conflict.SQLState() == "23505") {
				status = "conflict"
			}
		}
	} else {
		a.recordAudit(auditEvent{TenantSlug: tenant.Slug, ActorEmail: actor, ActorRole: role, Action: store.AuditActionAnnualRunCreate, TargetType: "annual_statement_run", TargetID: run.ID, Summary: "Abrechnungslauf berechnet", Details: map[string]string{"period_year": strconv.Itoa(year), "revision": strconv.Itoa(run.Revision), "input_hash": run.InputHash}})
	}
	target := "/app/settings/annual-statement?year=" + strconv.Itoa(year) + "&run-status=" + status
	if run.ID != "" {
		target += "&run=" + url.QueryEscape(run.ID)
	}
	http.Redirect(w, r, target+"#abrechnungslauf", http.StatusSeeOther)
}

func annualStatementRunView(repository store.AnnualStatementRunRepository, year int, selectedID, status string, consumption map[string]store.AnnualStatementConsumptionVector) web.AnnualStatementRunView {
	out := web.AnnualStatementRunView{Year: year}
	if repository == nil {
		out.Issues = []string{"Abrechnungslauf derzeit nicht verfügbar."}
		return out
	}
	input, _, err := repository.Preview(year, consumption)
	out.Ready = err == nil
	if err != nil {
		var blocked *store.AnnualStatementRunBlockedError
		if errors.As(err, &blocked) {
			for _, issue := range blocked.Issues {
				out.Issues = append(out.Issues, annualStatementRunIssueMessage(issue, input))
			}
		} else {
			out.Issues = append(out.Issues, "Die Abrechnungsgrundlagen konnten nicht vollständig gelesen werden. Bitte erneut versuchen.")
		}
	}
	switch status {
	case "created":
		out.Message = "Abrechnungslauf für alle Einheiten gespeichert."
		out.MessageOK = true
	case "blocked":
		out.Message = "Es wurde kein Lauf gespeichert. Bitte die fehlenden Grundlagen ergänzen."
	case "conflict":
		out.Message = "Parallel wurde ein weiterer Abrechnungslauf gestartet. Bitte die Seite neu laden und den gespeicherten Stand prüfen."
	case "error":
		out.Message = "Der Lauf konnte nicht gespeichert werden. Bitte erneut versuchen."
	}
	runs, err := repository.List(year)
	if err != nil {
		out.Ready = false
		out.Issues = append(out.Issues, "Gespeicherte Abrechnungsläufe konnten nicht gelesen werden.")
		return out
	}
	for i, run := range runs {
		selected := selectedID == run.ID || (selectedID == "" && i == 0)
		title := "Lauf " + strconv.Itoa(run.Revision)
		out.History = append(out.History, web.AnnualStatementRunLinkView{ID: run.ID, Label: title, Selected: selected})
		if !selected {
			continue
		}
		out.ID = run.ID
		out.Revision = run.Revision
		if location, err := time.LoadLocation("Europe/Vienna"); err == nil {
			out.CreatedAt = run.CreatedAt.In(location).Format("02.01.2006 15:04 MST")
		} else {
			out.CreatedAt = run.CreatedAt.Format("02.01.2006 15:04 UTC")
		}
		out.CreatedBy = run.CreatedBy
		out.Total = formatAnnualStatementMoney(run.Result.TotalCents)
		out.Excluded = formatAnnualStatementMoney(run.Result.ExcludedCents)
		for _, unit := range run.Result.Units {
			row := web.AnnualStatementRunUnitView{Label: unit.Label, Allocated: formatAnnualStatementMoney(unit.AllocatedCents), Prepaid: formatAnnualStatementMoney(unit.PrepaidCents), Balance: formatAnnualStatementBalance(-unit.BalanceCents)}
			for _, cost := range unit.Costs {
				row.Costs = append(row.Costs, web.AnnualStatementRunCostView{Name: cost.Name, Key: annualStatementAllocationKeyLabel(cost.AllocationKey), Share: formatAnnualStatementShare(cost.SharePPM, true), Amount: formatAnnualStatementMoney(cost.AmountCents)})
			}
			out.Units = append(out.Units, row)
		}
	}
	return out
}

func annualStatementRunIssueMessage(issue store.AnnualStatementRunIssue, input store.AnnualStatementRunInput) string {
	cost := issue.CostTypeKey
	unit := issue.UnitID
	for _, item := range input.Structure.CostTypes {
		if item.Key == cost {
			cost = item.Name
			break
		}
	}
	for _, item := range input.Units {
		if item.ID == unit {
			unit = item.Label
			break
		}
	}
	switch issue.Code {
	case "period":
		return "Abrechnungsperiode fehlt oder ist ungültig. Bitte eine gespeicherte Periode wählen."
	case "units":
		return "Die Einheiten sind unvollständig oder nicht eindeutig. Bitte die Einheitenzuordnung prüfen."
	case "structure":
		return "Die Einheiten und die Verteilerbasis des Abrechnungsjahres stimmen nicht überein. Bitte die Periodenbasis für alle Einheiten speichern."
	case "cost-types":
		return "Der Kostenartenkatalog fehlt oder enthält eine unklare Zuordnung."
	case "key":
		return cost + ": Verteilerschlüssel fehlt oder ist unbekannt. Es wird keine Ersatzregel angenommen."
	case "basis":
		return cost + ": Verteilerbasis fehlt für mindestens eine Einheit oder ergibt keine positive Gesamtsumme."
	case "nutzwert-total":
		return cost + ": Miteigentumsanteile ergeben nicht 1.000.000 Millionstel. Bitte die Vollständigkeit prüfen."
	case "missing-receipt":
		return cost + ": Kein bestätigter Beleg erfasst. Fehlende Belege gelten nicht als Nullkosten."
	case "receipt":
		return cost + ": Belegbetrag oder Zuordnung ist ungültig bzw. doppelt. Bitte die Belege prüfen."
	case "document":
		return cost + ": Das Originaldokument eines Belegs fehlt oder ist nicht lesbar."
	case "prepayment":
		return unit + ": Akonto fehlt oder ist ungültig. Auch 0,00 € muss ausdrücklich erfasst sein."
	case "consumption":
		return cost + ": Vollständige, vergleichbare Periodenmessungen für alle Einheiten fehlen. Es wird kein Verbrauch geschätzt."
	case "measurement-rule":
		return cost + ": Keine bekannte Messregel hinterlegt. Bitte die Zuordnung klären."
	case "overflow":
		return "Die Beträge oder Verteilerbasen sind zu groß für eine sichere Berechnung. Bitte die Eingaben prüfen."
	default:
		return "Eine Abrechnungsgrundlage ist ungeklärt. Der Lauf bleibt gesperrt."
	}
}
