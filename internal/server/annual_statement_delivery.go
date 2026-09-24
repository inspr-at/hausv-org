package server

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	appmail "github.com/inspr-at/hausv-org/internal/mail"
	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/web"
)

func (a *app) sendAnnualStatementRun(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actor, role, _, ok := a.buildingSettingsContext(w, ac)
	if !ok {
		return
	}
	repos := ac.repositories
	if repos.annualStatementRuns == nil || repos.documents == nil || repos.annualStatementDeliveries == nil {
		http.Error(w, "Versand derzeit nicht verfügbar.", http.StatusServiceUnavailable)
		return
	}
	run, found, err := repos.annualStatementRuns.Get(r.PathValue("runID"))
	if err != nil {
		http.Error(w, "Abrechnungslauf konnte nicht gelesen werden.", 500)
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
	archived := annualStatementArchiveDocuments(repos.documents, run)
	if !annualStatementArchiveComplete(run, archived) {
		http.Error(w, "Zuerst im Archiv ablegen", http.StatusConflict)
		return
	}
	if a.mailer == nil || !a.mailer.Configured() {
		http.Error(w, "E-Mail-Versand ist nicht eingerichtet.", http.StatusConflict)
		return
	}
	sent, failed, skipped := 0, 0, 0
	defer func() {
		a.recordAudit(auditEvent{TenantSlug: tenant.Slug, ActorEmail: actor, ActorRole: role, Action: store.AuditActionAnnualRunSend, TargetType: "annual_statement_run", TargetID: run.ID, Summary: fmt.Sprintf("Jahresabrechnung: %d gesendet · %d fehlgeschlagen · %d übersprungen", sent, failed, skipped), Details: map[string]string{"run_id": run.ID, "revision": strconv.Itoa(run.Revision), "sent": strconv.Itoa(sent), "failed": strconv.Itoa(failed), "skipped": strconv.Itoa(skipped)}})
	}()
	for _, party := range run.Input.Parties {
		if r.Context().Err() != nil {
			return
		}
		document := archived[store.AnnualStatementArchiveID(run.ID, run.Revision, party.UnitID, party.ID)]
		item := store.AnnualStatementDelivery{RunID: run.ID, Revision: run.Revision, PartyID: party.ID, UnitID: party.UnitID, DocumentID: document.ID, SHA256: document.AnnualStatementArchive.SHA256, Recipient: party.ID, Actor: actor}
		result, skip, err := repos.annualStatementDeliveries.Attempt(r.Context(), item, func(ctx context.Context) error {
			recipient, err := mail.ParseAddress(party.ID)
			if err != nil || recipient.Address != party.ID {
				return fmt.Errorf("Für diese Partei fehlt eine gültige E-Mail-Adresse.")
			}
			data, err := readAnnualStatementArchive(repos.documents, document)
			if err != nil {
				return err
			}
			if ctx.Err() != nil {
				return fmt.Errorf("Versand abgebrochen.")
			}
			subject, body := annualStatementMailText(run, party)
			if err := a.mailer.SendDocument(ctx, recipient.Address, subject, body, appmail.Attachment{Filename: document.Filename, ContentType: "application/pdf", Data: data}); err != nil {
				if ctx.Err() != nil {
					return fmt.Errorf("Versand abgebrochen.")
				}
				return fmt.Errorf("E-Mail konnte nicht versendet werden. Bitte erneut versuchen.")
			}
			return nil
		})
		if err != nil {
			if errors.Is(err, store.ErrDeliveryInProgress) {
				http.Error(w, "Versand läuft bereits — bitte Seite neu laden", http.StatusConflict)
				return
			}
			logError("annual statement delivery record failed", err, "tenant", tenant.Slug, "run", run.ID)
			http.Error(w, "Versandprotokoll konnte nicht gespeichert werden. Bitte den Versandstatus vor einem erneuten Versuch prüfen.", http.StatusInternalServerError)
			return
		}
		if skip {
			skipped++
		} else if result.Status == "sent" {
			sent++
		} else {
			failed++
		}
	}
	target := "/app/settings/annual-statement?year=" + strconv.Itoa(run.PeriodYear) + "&run=" + url.QueryEscape(run.ID) + "&sent=" + strconv.Itoa(sent) + "&failed=" + strconv.Itoa(failed) + "&skipped=" + strconv.Itoa(skipped) + "#abrechnungsergebnis"
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func annualStatementArchiveComplete(run store.AnnualStatementRun, archived map[string]store.DocumentRecord) bool {
	if len(run.Input.Parties) == 0 {
		return false
	}
	if _, ok := archived[store.AnnualStatementArchiveID(run.ID, run.Revision, "", "")]; !ok {
		return false
	}
	for _, party := range run.Input.Parties {
		if _, ok := archived[store.AnnualStatementArchiveID(run.ID, run.Revision, party.UnitID, party.ID)]; !ok {
			return false
		}
	}
	return true
}

func readAnnualStatementArchive(repository store.DocumentRepository, document store.DocumentRecord) ([]byte, error) {
	path, ok := repository.FilePath(document)
	if !ok {
		return nil, fmt.Errorf("Archivdatei ist nicht verfügbar.")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() != document.Size || info.Size() > store.MaxDocumentBytes {
		return nil, fmt.Errorf("Archivdatei ist nicht verfügbar oder beschädigt.")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Archivdatei ist nicht lesbar.")
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, store.MaxDocumentBytes+1))
	if err != nil {
		return nil, fmt.Errorf("Archivdatei ist nicht lesbar.")
	}
	if int64(len(data)) != document.Size || fmt.Sprintf("%x", sha256.Sum256(data)) != document.AnnualStatementArchive.SHA256 {
		return nil, fmt.Errorf("Prüfsumme der Archivdatei stimmt nicht überein. Kein Versand.")
	}
	return data, nil
}

func annualStatementMailText(run store.AnnualStatementRun, party store.AnnualStatementRunParty) (string, string) {
	presentation := run.Input.Presentation
	label := party.UnitID
	for _, unit := range run.Input.Units {
		if unit.ID == party.UnitID {
			label = unit.Label
			break
		}
	}
	estate := firstNonEmpty(presentation.EstateName, presentation.EstateAddress, presentation.EstateSlug)
	subject := fmt.Sprintf("Jahresabrechnung %d · %s · %s", run.PeriodYear, estate, label)
	greeting := "Guten Tag,"
	if party.Name != "" {
		greeting = "Guten Tag " + party.Name + ","
	}
	lines := []string{greeting, "", fmt.Sprintf("anbei erhalten Sie die archivierte Jahresabrechnung %d für %s in der Liegenschaft %s (Lauf %d).", run.PeriodYear, label, estate, run.Revision), "", "Freigegebene Jahresabrechnung", "", "Bei Fragen wenden Sie sich bitte an Ihre Verwaltung.", "", "Freundliche Grüße", firstNonEmpty(presentation.Organisation, presentation.ContactName, "Ihre Verwaltung")}
	for _, contact := range []string{presentation.ContactName, presentation.ContactAddress, presentation.ContactEmail, presentation.ContactPhone} {
		if contact != "" {
			lines = append(lines, contact)
		}
	}
	return subject, strings.Join(lines, "\n")
}

func (a *app) annualStatementDeliveryView(view web.AnnualStatementRunView, repositories requestRepositories, query url.Values) web.AnnualStatementRunView {
	if view.ID == "" {
		return view
	}
	repository := repositories.annualStatementDeliveries
	partyNames, unitNames := map[string]string{}, map[string]string{}
	if repositories.annualStatementRuns != nil {
		if run, found, err := repositories.annualStatementRuns.Get(view.ID); err == nil && found {
			for _, party := range run.Input.Parties {
				partyNames[party.ID] = firstNonEmpty(party.Name, party.ID)
			}
			for _, unit := range run.Input.Units {
				unitNames[unit.ID] = unit.Label
			}
		}
	}
	view.SendAction = "/app/settings/annual-statement/runs/" + url.PathEscape(view.ID) + "/send"
	view.MailMode = "Versand per E-Mail an die hinterlegten Adressen"
	if mode, ok := a.mailer.(interface{ Mode() string }); ok {
		view.MailMode = mode.Mode()
	}
	switch {
	case !view.Approved:
		view.SendIssue = "Zuerst den Abrechnungslauf freigeben"
	case view.ArchivedAt == "":
		view.SendIssue = "Zuerst im Archiv ablegen"
	case a.mailer == nil || !a.mailer.Configured():
		view.SendIssue = "E-Mail-Versand ist nicht eingerichtet."
	case repository == nil:
		view.SendIssue = "Versandprotokoll derzeit nicht verfügbar."
	}
	if repository != nil {
		deliveries, err := repository.List(view.ID)
		if err != nil {
			view.SendIssue = "Versandprotokoll konnte nicht gelesen werden."
		} else {
			location, err := time.LoadLocation("Europe/Vienna")
			if err != nil {
				location = time.UTC
			}
			for _, item := range deliveries {
				status, detail := "Fehlgeschlagen", item.Error
				switch item.Status {
				case "sent":
					status = "Gesendet"
				case "pending":
					// A reservation older than the TTL was left by a process that
					// died mid-send; the next attempt records it as interrupted.
					if time.Since(item.SentAt) < store.AnnualStatementDeliveryPendingTTL {
						status = "Wird gesendet"
					} else {
						detail = store.AnnualStatementDeliveryInterrupted
					}
				}
				view.Deliveries = append(view.Deliveries, web.AnnualStatementDeliveryView{Party: firstNonEmpty(partyNames[item.PartyID], item.PartyID) + " · " + firstNonEmpty(unitNames[item.UnitID], item.UnitID), Recipient: item.Recipient, Time: item.SentAt.In(location).Format("02.01.2006 15:04 MST"), Status: status, Error: detail})
			}
		}
	}
	counts := []int{}
	for _, key := range []string{"sent", "failed", "skipped"} {
		n, err := strconv.Atoi(query.Get(key))
		if err != nil || n < 0 {
			return view
		}
		counts = append(counts, n)
	}
	view.DeliverySummary = fmt.Sprintf("%d gesendet · %d fehlgeschlagen · %d übersprungen", counts[0], counts[1], counts[2])
	return view
}
