package server

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"time"

	"github.com/inspr-at/hausv-org/internal/statementpdf"
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
	presentation := store.AnnualStatementRunPresentation{EstateSlug: tenant.Slug, EstateName: tenant.Name, EstateAddress: tenant.Address, Organisation: tenant.ContactName, ContactName: tenant.ContactName, ContactAddress: tenant.ContactAddress, ContactEmail: tenant.ContactEmail, ContactPhone: tenant.ContactPhone}
	if org, found := a.organisationRecordFor(r.Context(), &ac); found {
		presentation.Organisation = org.Name
		presentation.ContactName = firstNonEmpty(org.ContactName, presentation.ContactName)
		presentation.ContactEmail = firstNonEmpty(org.ContactEmail, presentation.ContactEmail)
		presentation.ContactPhone = firstNonEmpty(org.ContactPhone, presentation.ContactPhone)
	}
	run, err := ac.repositories.annualStatementRuns.Create(year, actor, time.Now(), presentation)
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
	http.Redirect(w, r, target+"#abrechnungsergebnis", http.StatusSeeOther)
}

func (a *app) archiveAnnualStatementRun(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actor, role, _, ok := a.buildingSettingsContext(w, ac)
	if !ok {
		return
	}
	if ac.repositories.annualStatementRuns == nil || ac.repositories.documents == nil {
		http.Error(w, "Archiv derzeit nicht verfügbar.", http.StatusServiceUnavailable)
		return
	}
	run, found, err := ac.repositories.annualStatementRuns.Get(r.PathValue("runID"))
	if err != nil {
		http.Error(w, "Abrechnungslauf konnte nicht gelesen werden.", http.StatusInternalServerError)
		return
	}
	if !found {
		http.NotFound(w, r)
		return
	}
	if run.Approval == nil {
		http.Error(w, "Zuerst den Abrechnungslauf freigeben.", http.StatusConflict)
		return
	}
	selection, err := statementpdf.Documents(run, "", "")
	if err != nil {
		http.Error(w, "Für das Archiv fehlen gespeicherte Parteien. Bitte die Parteien zuordnen und einen neuen Lauf berechnen.", http.StatusConflict)
		return
	}
	// The combined document is persisted last. Partial attempts remain immutable
	// and can be completed by retrying the same action.
	selection = append(selection, statementpdf.Document{UnitLabel: "Alle Dokumente"})
	now := time.Now()
	status := "archived"
	for _, document := range selection {
		data, renderErr := statementpdf.Render(run, document.UnitID, document.PartyID)
		if renderErr != nil {
			err = renderErr
			break
		}
		_, err = ac.repositories.documents.CreateGenerated(store.DocumentRecord{
			Title:    fmt.Sprintf("Jahresabrechnung %d · %s · Lauf %d", run.PeriodYear, document.UnitLabel, run.Revision),
			Category: store.DocumentCategoryBilling, Visibility: store.DocumentVisibilityManagerOnly,
			UnitID: document.UnitID, UploadedBy: actor,
			AnnualStatementArchive: &store.AnnualStatementArchiveMetadata{RunID: run.ID, Revision: run.Revision, PeriodYear: run.PeriodYear, PartyID: document.PartyID},
		}, annualStatementPDFFilename(run, document.UnitID), "application/pdf", data, now)
		if err != nil {
			break
		}
	}
	if err != nil {
		status = "archive-error"
		logError("annual statement archive failed", err, "tenant", tenant.Slug, "run", run.ID)
	} else {
		a.recordAudit(auditEvent{TenantSlug: tenant.Slug, ActorEmail: actor, ActorRole: role, Action: store.AuditActionAnnualRunArchive, TargetType: "annual_statement_run", TargetID: run.ID, Summary: "Jahresabrechnung im Archiv abgelegt", Details: map[string]string{"run_id": run.ID, "revision": strconv.Itoa(run.Revision), "document_count": strconv.Itoa(len(selection))}})
	}
	http.Redirect(w, r, "/app/settings/annual-statement?year="+strconv.Itoa(run.PeriodYear)+"&run="+url.QueryEscape(run.ID)+"&run-status="+status+"#abrechnungsergebnis", http.StatusSeeOther)
}

func annualStatementRunView(repository store.AnnualStatementRunRepository, documents store.DocumentRepository, year int, selectedID, status string, consumption map[string]store.AnnualStatementConsumptionVector) web.AnnualStatementRunView {
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
	case "approved":
		out.Message = "Abrechnungslauf freigegeben. Die PDFs sind jetzt endgültig."
		out.MessageOK = true
	case "archived":
		out.Message = "Alle PDFs dieses Laufs sind unveränderlich im Archiv abgelegt."
		out.MessageOK = true
	case "archive-error":
		out.Message = "Die Archivierung konnte nicht abgeschlossen werden. Bereits abgelegte Dokumente bleiben erhalten. Bitte erneut versuchen."
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
		if _, err := statementpdf.Documents(run, "", ""); err == nil {
			out.AllPDFURL = annualStatementPDFURL(run.ID, "", "")
			if run.Input.Structure.Legal.Regime == "mrg_voll" {
				out.AushangURL = out.AllPDFURL + "?aushang=1"
			}
		}
		out.Approved = run.Approval != nil
		if run.Approval != nil {
			out.ApprovedAt = run.Approval.ApprovedAt.Format("02.01.2006")
		} else {
			out.ApproveAction = "/app/settings/annual-statement/runs/" + url.PathEscape(run.ID) + "/approve"
		}
		out.ID = run.ID
		out.Revision = run.Revision
		out.ArchiveURL = "/app/dokumente?q=" + url.QueryEscape(store.DocumentCategoryBilling)
		out.ArchiveAction = "/app/settings/annual-statement/runs/" + url.PathEscape(run.ID) + "/archive"
		archived := annualStatementArchiveDocuments(documents, run)
		if annualStatementArchiveComplete(run, archived) && out.AllPDFURL != "" {
			var completedAt time.Time
			for _, document := range archived {
				if document.AnnualStatementArchive.ArchivedAt.After(completedAt) {
					completedAt = document.AnnualStatementArchive.ArchivedAt
				}
			}
			if location, err := time.LoadLocation("Europe/Vienna"); err == nil {
				out.ArchivedAt = completedAt.In(location).Format("02.01.2006 15:04 MST")
			} else {
				out.ArchivedAt = completedAt.Format("02.01.2006 15:04 UTC")
			}
			out.ArchiveCount = len(archived)
		}
		if combined, ok := archived[store.AnnualStatementArchiveID(run.ID, run.Revision, "", "")]; ok {
			out.ArchiveURL += "#document-" + url.PathEscape(combined.ID)
		}
		if location, err := time.LoadLocation("Europe/Vienna"); err == nil {
			out.CreatedAt = run.CreatedAt.In(location).Format("02.01.2006 15:04 MST")
		} else {
			out.CreatedAt = run.CreatedAt.Format("02.01.2006 15:04 UTC")
		}
		out.CreatedBy = run.CreatedBy
		out.Total = formatAnnualStatementMoney(run.Result.TotalCents)
		out.Excluded = formatAnnualStatementMoney(run.Result.ExcludedCents)
		for _, unit := range store.AnnualStatementRunDisplayOrder(run) {
			row := web.AnnualStatementRunUnitView{Label: unit.Label, Allocated: formatAnnualStatementMoney(unit.AllocatedCents), Prepaid: formatAnnualStatementMoney(unit.PrepaidCents), Balance: formatAnnualStatementBalance(-unit.BalanceCents)}
			for _, cost := range unit.Costs {
				row.Costs = append(row.Costs, web.AnnualStatementRunCostView{Name: cost.Name, Key: annualStatementAllocationKeyLabel(cost.AllocationKey), Share: formatAnnualStatementShare(cost.SharePPM, true), Amount: formatAnnualStatementMoney(cost.AmountCents)})
			}
			for _, party := range run.Input.Parties {
				if party.UnitID == unit.UnitID {
					row.PDFs = append(row.PDFs, web.AnnualStatementRunPDFView{Label: firstNonEmpty(party.Name, party.ID), URL: annualStatementPDFURL(run.ID, unit.UnitID, party.ID)})
					if document, ok := archived[store.AnnualStatementArchiveID(run.ID, run.Revision, unit.UnitID, party.ID)]; ok {
						row.PDFs[len(row.PDFs)-1].ArchiveURL = "/app/dokumente?q=" + url.QueryEscape(store.DocumentCategoryBilling) + "#document-" + url.PathEscape(document.ID)
					}
				}
			}
			out.Units = append(out.Units, row)
		}
	}
	return out
}

func annualStatementArchiveDocuments(repository store.DocumentRepository, run store.AnnualStatementRun) map[string]store.DocumentRecord {
	out := map[string]store.DocumentRecord{}
	if repository == nil {
		return out
	}
	for _, document := range repository.List() {
		metadata := document.AnnualStatementArchive
		if metadata != nil && metadata.RunID == run.ID && metadata.Revision == run.Revision {
			out[document.ID] = document
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

func (a *app) approveAnnualStatementRun(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actor, role, _, ok := a.buildingSettingsContext(w, ac)
	if !ok {
		return
	}
	if ac.repositories.annualStatementRuns == nil {
		http.Error(w, "Abrechnungslauf derzeit nicht verfügbar.", 503)
		return
	}
	run, found, err := ac.repositories.annualStatementRuns.Get(r.PathValue("runID"))
	if err != nil {
		http.Error(w, "Abrechnungslauf konnte nicht gelesen werden.", 500)
		return
	}
	if !found {
		http.NotFound(w, r)
		return
	}
	if run.Approval == nil && ac.repositories.annualStatementPeriods != nil {
		structure, ok := ac.repositories.annualStatementPeriods.Structure(run.PeriodYear)
		if !ok || !reflect.DeepEqual(structure.Legal, run.Input.Structure.Legal) {
			http.Error(w, "Die Rechtsgrundlage wurde geändert. Bitte einen neuen Lauf berechnen.", 409)
			return
		}
	}
	if _, err := statementpdf.Documents(run, "", ""); err != nil {
		http.Error(w, "Bitte zuerst alle Parteien zuordnen und einen neuen Lauf berechnen.", 409)
		return
	}
	run, changed, err := ac.repositories.annualStatementRuns.Approve(run.ID, actor, role, time.Now())
	if err != nil {
		http.Error(w, "Freigabe nicht möglich. Bereits archivierte Entwürfe benötigen einen neuen Lauf.", 409)
		return
	}
	if changed {
		a.recordAudit(auditEvent{TenantSlug: tenant.Slug, ActorEmail: actor, ActorRole: role, Action: store.AuditActionAnnualRunApprove, TargetType: "annual_statement_run", TargetID: run.ID, Summary: "Abrechnungslauf freigegeben", Details: map[string]string{"revision": strconv.Itoa(run.Revision), "input_hash": run.InputHash}})
	}
	http.Redirect(w, r, "/app/settings/annual-statement?year="+strconv.Itoa(run.PeriodYear)+"&run="+url.QueryEscape(run.ID)+"&run-status=approved#abrechnungsergebnis", http.StatusSeeOther)
}
