package server

import (
	"context"
	"fmt"
	"mime"
	"net/http"
	"net/mail"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	appmail "github.com/inspr-at/hausv-org/internal/mail"
	"github.com/inspr-at/hausv-org/internal/statementpdf"
	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/view"
	"github.com/inspr-at/hausv-org/internal/web"
)

func (a *app) tenantStatementRepo(w http.ResponseWriter, ac authCtx) (*store.TenantStatementRepository, bool) {
	if _, _, _, _, ok := a.buildingSettingsContext(w, ac); !ok {
		return nil, false
	}
	repo, ok := store.BindTenantStatementRepository(a.tenantDB, ac.tenantRef)
	if !ok {
		http.Error(w, "Mieterabrechnungen derzeit nicht verfügbar.", 503)
	}
	return repo, ok
}

func (a *app) rentalManagementSave(w http.ResponseWriter, r *http.Request, ac authCtx) {
	repo, ok := a.tenantStatementRepo(w, ac)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formular ungültig.", 400)
		return
	}
	if ac.repositories.annualStatementRuns == nil || ac.repositories.leases == nil {
		http.Error(w, "Grundlagen fehlen.", 503)
		return
	}
	run, found, err := ac.repositories.annualStatementRuns.Get(r.PathValue("runID"))
	if err != nil {
		http.Error(w, "WEG-Lauf nicht lesbar.", 500)
		return
	}
	if !found {
		http.NotFound(w, r)
		return
	}
	m := store.RentalManagement{UnitID: r.FormValue("unit_id"), OwnerEmail: r.FormValue("owner_email"), Active: r.FormValue("active") == "1", HeatingMonthlyConfirmed: r.FormValue("heating_monthly") == "1", UpdatedBy: ac.email, Passable: map[string]bool{}, ContractCosts: map[string][]string{}}
	allowed := map[string]bool{}
	for _, key := range r.Form["passable"] {
		allowed[key] = true
	}
	for _, c := range run.Input.Structure.CostTypes {
		m.Passable[c.Key] = allowed[c.Key]
	}
	history, err := ac.repositories.leases.History(m.UnitID)
	if err != nil {
		http.Error(w, "Mietverträge nicht lesbar.", 500)
		return
	}
	for _, l := range history {
		m.ContractCosts[l.ID] = append([]string{}, r.Form["contract_"+l.ID]...)
	}
	if err = repo.SaveManagement(m); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	a.recordAudit(auditEvent{TenantSlug: ac.tenant.Slug, ActorEmail: ac.email, ActorRole: ac.role, Action: store.AuditActionAnnualRunCreate, TargetType: "rental_management", TargetID: m.UnitID, Summary: "Mietverwaltung und Kostenkatalog gespeichert"})
	tenantStatementRedirect(w, r, run, "")
}

func tenantStatementRedirect(w http.ResponseWriter, r *http.Request, run store.AnnualStatementRun, message string) {
	q := url.Values{"year": {strconv.Itoa(run.PeriodYear)}, "run": {run.ID}}
	if message != "" {
		q.Set("tenant-message", message)
	}
	http.Redirect(w, r, "/app/settings/annual-statement?"+q.Encode()+"#mieterabrechnungen", http.StatusSeeOther)
}

func (a *app) tenantStatementCreate(w http.ResponseWriter, r *http.Request, ac authCtx) {
	repo, ok := a.tenantStatementRepo(w, ac)
	if !ok {
		return
	}
	if ac.repositories.annualStatementRuns == nil {
		http.Error(w, "WEG-Läufe fehlen.", 503)
		return
	}
	run, found, err := ac.repositories.annualStatementRuns.Get(r.PathValue("runID"))
	if err != nil {
		http.Error(w, "WEG-Lauf nicht lesbar.", 500)
		return
	}
	if !found {
		http.NotFound(w, r)
		return
	}
	s, err := repo.Create(run, r.FormValue("unit_id"), r.FormValue("statement_on"), r.FormValue("contract_due_on"), ac.email)
	if err != nil {
		tenantStatementRedirect(w, r, run, err.Error())
		return
	}
	a.recordAudit(auditEvent{TenantSlug: ac.tenant.Slug, ActorEmail: ac.email, ActorRole: ac.role, Action: store.AuditActionAnnualRunCreate, TargetType: "tenant_statement", TargetID: s.ID, Summary: "Mieterabrechnung als Entwurf erstellt"})
	tenantStatementRedirect(w, r, run, "Mieterabrechnung erstellt. Bitte Vorschau prüfen und freigeben.")
}

func (a *app) loadTenantStatement(w http.ResponseWriter, r *http.Request, ac authCtx) (*store.TenantStatementRepository, store.TenantStatement, bool) {
	repo, ok := a.tenantStatementRepo(w, ac)
	if !ok {
		return nil, store.TenantStatement{}, false
	}
	s, found, err := repo.Get(r.PathValue("statementID"))
	if err != nil {
		http.Error(w, "Mieterabrechnung nicht lesbar.", 500)
		return nil, s, false
	}
	if !found {
		http.NotFound(w, r)
		return nil, s, false
	}
	return repo, s, true
}

func (a *app) tenantStatementApprove(w http.ResponseWriter, r *http.Request, ac authCtx) {
	repo, s, ok := a.loadTenantStatement(w, r, ac)
	if !ok {
		return
	}
	if _, err := statementpdf.RenderTenant(s, ""); err != nil {
		http.Error(w, "Kein abrechenbarer Mieteranteil vorhanden.", 409)
		return
	}
	if _, err := repo.Approve(s.ID, ac.email, ac.role); err != nil {
		http.Error(w, err.Error(), 409)
		return
	}
	a.recordAudit(auditEvent{TenantSlug: ac.tenant.Slug, ActorEmail: ac.email, ActorRole: ac.role, Action: store.AuditActionAnnualRunApprove, TargetType: "tenant_statement", TargetID: s.ID, Summary: "Mieterabrechnung freigegeben"})
	tenantStatementRedirect(w, r, s.Source, "Mieterabrechnung freigegeben.")
}

func (a *app) tenantStatementPDF(w http.ResponseWriter, r *http.Request, ac authCtx) {
	_, s, ok := a.loadTenantStatement(w, r, ac)
	if !ok {
		return
	}
	data, err := statementpdf.RenderTenant(s, r.URL.Query().Get("account"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": s.ID + ".pdf"}))
	w.Write(data)
}

type tenantStatementRecipient struct {
	Key, Email, Name, Account string
	Owner                     bool
}

func tenantStatementRecipients(s store.TenantStatement) []tenantStatementRecipient {
	out := []tenantStatementRecipient{{Key: "owner", Email: s.Owner.ID, Name: s.Owner.Name, Owner: true}}
	for _, a := range s.Accounts {
		if a.Landlord || len(a.Lines) == 0 && a.OperatingPrepaidCents == 0 && a.HeatingPrepaidCents == 0 {
			continue
		}
		for _, p := range a.Parties {
			out = append(out, tenantStatementRecipient{Key: a.ID + "/" + p.ID, Email: p.Email, Name: p.Name, Account: a.ID})
		}
	}
	return out
}

func tenantStatementArchiveDocs(repo store.DocumentRepository, s store.TenantStatement) map[string]store.DocumentRecord {
	out := map[string]store.DocumentRecord{}
	if repo == nil {
		return out
	}
	for _, d := range repo.List() {
		if m := d.AnnualStatementArchive; m != nil && m.RunID == s.ID && m.Revision == s.Revision {
			out[m.PartyID] = d
		}
	}
	return out
}

func (a *app) tenantStatementArchive(w http.ResponseWriter, r *http.Request, ac authCtx) {
	_, s, ok := a.loadTenantStatement(w, r, ac)
	if !ok {
		return
	}
	if s.Approval == nil {
		http.Error(w, "Zuerst Mieterabrechnung freigeben.", 409)
		return
	}
	if ac.repositories.documents == nil {
		http.Error(w, "Archiv fehlt.", 503)
		return
	}
	for _, p := range tenantStatementRecipients(s) {
		data, err := statementpdf.RenderTenant(s, p.Account)
		if err != nil {
			http.Error(w, "PDF nicht erstellbar.", 500)
			return
		}
		title := "Mieter-Betriebskostenabrechnung"
		if p.Owner {
			title += " · Eigentümerkopie"
		}
		_, err = ac.repositories.documents.CreateGenerated(store.DocumentRecord{Title: fmt.Sprintf("%s %d · %s · %s", title, s.Source.PeriodYear, s.UnitLabel, p.Name), Category: store.DocumentCategoryBilling, Visibility: store.DocumentVisibilityManagerOnly, UnitID: s.UnitID, UploadedBy: ac.email, AnnualStatementArchive: &store.AnnualStatementArchiveMetadata{RunID: s.ID, Revision: s.Revision, PeriodYear: s.Source.PeriodYear, PartyID: p.Key}}, s.ID+".pdf", "application/pdf", data, time.Now())
		if err != nil {
			http.Error(w, "Archivierung unvollständig. Erneut versuchen.", 500)
			return
		}
	}
	a.recordAudit(auditEvent{TenantSlug: ac.tenant.Slug, ActorEmail: ac.email, ActorRole: ac.role, Action: store.AuditActionAnnualRunArchive, TargetType: "tenant_statement", TargetID: s.ID, Summary: "Mieterabrechnung und Eigentümerkopie archiviert"})
	tenantStatementRedirect(w, r, s.Source, "Mieterabrechnung und Eigentümerkopie archiviert.")
}

func (a *app) tenantStatementSend(w http.ResponseWriter, r *http.Request, ac authCtx) {
	_, s, ok := a.loadTenantStatement(w, r, ac)
	if !ok {
		return
	}
	if s.Approval == nil {
		http.Error(w, "Zuerst Mieterabrechnung freigeben.", 409)
		return
	}
	if a.mailer == nil || !a.mailer.Configured() || ac.repositories.annualStatementDeliveries == nil {
		http.Error(w, "Versand derzeit nicht verfügbar.", 409)
		return
	}
	archived := tenantStatementArchiveDocs(ac.repositories.documents, s)
	recipients := tenantStatementRecipients(s)
	for _, p := range recipients {
		if _, ok := archived[p.Key]; !ok {
			http.Error(w, "Zuerst alle Mieterabrechnungen und die Eigentümerkopie archivieren.", 409)
			return
		}
	}
	sent, failed, skipped := 0, 0, 0
	for _, p := range recipients {
		d := archived[p.Key]
		item := store.AnnualStatementDelivery{RunID: s.ID, Revision: s.Revision, PartyID: p.Key, UnitID: s.UnitID, DocumentID: d.ID, SHA256: d.AnnualStatementArchive.SHA256, Recipient: p.Email, Actor: ac.email}
		result, skip, err := ac.repositories.annualStatementDeliveries.Attempt(r.Context(), item, func(ctx context.Context) error {
			address, err := mail.ParseAddress(p.Email)
			if err != nil || address.Address != p.Email {
				return fmt.Errorf("Gültige Empfängeradresse fehlt.")
			}
			data, err := readAnnualStatementArchive(ac.repositories.documents, d)
			if err != nil {
				return err
			}
			subject := fmt.Sprintf("Mieter-Betriebskostenabrechnung %d · %s", s.Source.PeriodYear, s.UnitLabel)
			if p.Owner {
				subject += " · Eigentümerkopie"
			}
			body := "Guten Tag " + p.Name + ",\n\nanbei die freigegebene Mieter-Betriebskostenabrechnung im Namen und auf Rechnung von " + s.Owner.Name + ".\n\nFreundliche Grüße\n" + s.Source.Input.Presentation.Organisation
			if err := a.mailer.SendDocument(ctx, p.Email, subject, body, appmail.Attachment{Filename: d.Filename, ContentType: "application/pdf", Data: data}); err != nil {
				return fmt.Errorf("E-Mail konnte nicht versendet werden.")
			}
			return nil
		})
		if err != nil {
			http.Error(w, "Versandstatus vor erneutem Versuch prüfen.", 409)
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
	message := fmt.Sprintf("Mieterabrechnung: %d gesendet · %d fehlgeschlagen · %d übersprungen", sent, failed, skipped)
	a.recordAudit(auditEvent{TenantSlug: ac.tenant.Slug, ActorEmail: ac.email, ActorRole: ac.role, Action: store.AuditActionAnnualRunSend, TargetType: "tenant_statement", TargetID: s.ID, Summary: message})
	tenantStatementRedirect(w, r, s.Source, message)
}

func (a *app) tenantStatementPanel(ac authCtx, runID, message string) web.TenantStatementPanelData {
	out := web.TenantStatementPanelData{Message: message}
	if runID == "" || !ac.can(capabilityManageLeases) || ac.repositories.annualStatementRuns == nil || ac.repositories.leases == nil {
		return out
	}
	repo, ok := store.BindTenantStatementRepository(a.tenantDB, ac.tenantRef)
	if !ok {
		return out
	}
	run, found, err := ac.repositories.annualStatementRuns.Get(runID)
	if err != nil || !found || run.Approval == nil || run.Input.Structure.Legal.Regime != "weg" {
		return out
	}
	out.Enabled = true
	out.RunID = runID
	out.StatementOn = time.Now().Format(time.DateOnly)
	history, err := ac.repositories.leases.List()
	if err != nil {
		out.Message = "Mietverträge konnten nicht geladen werden."
		return out
	}
	for _, u := range run.Result.Units {
		var leases []store.Lease
		for _, l := range history {
			if l.UnitID == u.UnitID && l.LeaseKind == store.LeaseKindHauptmiete && l.Status != store.LeaseStatusDraft {
				leases = append(leases, l)
			}
		}
		if len(leases) == 0 {
			continue
		}
		m, err := repo.Management(u.UnitID)
		if err != nil {
			out.Message = "Mietverwaltung konnte nicht geladen werden."
			return out
		}
		unit := web.TenantStatementUnitView{ID: u.UnitID, Label: u.Label, Active: m.Active, HeatingMonthlyConfirmed: m.HeatingMonthlyConfirmed, OwnerEmail: m.OwnerEmail}
		for _, p := range run.Input.Parties {
			if p.UnitID == u.UnitID && p.Owner {
				unit.Owners = append(unit.Owners, web.TenantStatementOption{Key: p.ID, Label: p.Name, Selected: p.ID == m.OwnerEmail})
				if p.ID == m.OwnerEmail {
					unit.OwnerName = p.Name
				}
			}
		}
		for _, c := range run.Input.Structure.CostTypes {
			if store.IsAnnualHeatingCost(c.Key) {
				continue
			}
			pass := store.DefaultTenantPassable(c.Key)
			if v, ok := m.Passable[c.Key]; ok {
				pass = v
			}
			unit.Costs = append(unit.Costs, web.TenantStatementOption{Key: c.Key, Label: c.Name, Selected: pass})
		}
		for _, l := range leases {
			names := []string{}
			for _, p := range l.Parties {
				names = append(names, p.Name)
			}
			lv := web.TenantStatementLeaseView{ID: l.ID, Label: strings.Join(names, ", "), Scope: mrgLabel(l.MRGScope), Contract: l.MRGScope != store.MRGVoll}
			for _, c := range unit.Costs {
				option := c
				option.Selected = false
				for _, key := range m.ContractCosts[l.ID] {
					if key == c.Key {
						option.Selected = true
					}
				}
				lv.Costs = append(lv.Costs, option)
			}
			unit.Leases = append(unit.Leases, lv)
		}
		out.Units = append(out.Units, unit)
	}
	sort.SliceStable(out.Units, func(i, j int) bool { return store.UnitLabelLess(out.Units[i].Label, out.Units[j].Label) })
	statements, err := repo.List(runID)
	if err != nil {
		out.Message = "Mieterabrechnungen konnten nicht geladen werden."
		return out
	}
	for _, s := range statements {
		v := web.TenantStatementView{ID: s.ID, Label: s.UnitLabel, Date: tenantStatementDate(s.StatementOn), Approved: s.Approval != nil, Owner: s.Owner.Name, SourcePassed: view.FormatEURCents(s.SourcePassedCents), SourceRetained: view.FormatEURCents(s.SourceRetainedCents)}
		if a.mailer == nil || !a.mailer.Configured() || ac.repositories.annualStatementDeliveries == nil {
			v.SendIssue = "E-Mail-Versand ist nicht eingerichtet."
		}
		archived := tenantStatementArchiveDocs(ac.repositories.documents, s)
		v.Archived = len(archived) == len(tenantStatementRecipients(s))
		for _, account := range s.Accounts {
			if len(account.Lines) == 0 && account.OperatingPrepaidCents == 0 && account.HeatingPrepaidCents == 0 {
				continue
			}
			names := []string{}
			for _, p := range account.Parties {
				names = append(names, p.Name)
			}
			if account.Landlord {
				names = []string{"Eigentümer (Leerstand)"}
			}
			result, amount := tenantStatementResult(account.BalanceCents())
			v.Accounts = append(v.Accounts, web.TenantStatementAccountView{ID: account.ID, Names: strings.Join(names, ", "), Result: result, Amount: amount})
		}
		if ac.repositories.annualStatementDeliveries != nil {
			deliveries, e := ac.repositories.annualStatementDeliveries.List(s.ID)
			if e != nil {
				out.Message = "Versandprotokoll konnte nicht gelesen werden."
			} else {
				location, err := time.LoadLocation("Europe/Vienna")
				if err != nil {
					location = time.UTC
				}
				for _, d := range deliveries {
					status, detail := "Fehlgeschlagen", d.Error
					switch d.Status {
					case "sent":
						status = "Gesendet"
					case "pending":
						if time.Since(d.SentAt) < store.AnnualStatementDeliveryPendingTTL {
							status = "Wird gesendet"
						} else {
							detail = store.AnnualStatementDeliveryInterrupted
						}
					}
					v.Deliveries = append(v.Deliveries, web.AnnualStatementDeliveryView{Recipient: d.Recipient, Status: status, Error: detail, Time: d.SentAt.In(location).Format("02.01.2006 15:04")})
				}
			}
		}
		out.Statements = append(out.Statements, v)
	}
	return out
}

// tenantStatementDate shows a stored ISO date in Austrian notation.
func tenantStatementDate(iso string) string {
	if at, err := time.Parse(time.DateOnly, iso); err == nil {
		return at.Format("02.01.2006")
	}
	return iso
}

// tenantStatementResult names the tenant's settlement instead of a signed saldo.
func tenantStatementResult(balance int64) (string, string) {
	switch {
	case balance > 0:
		return "Nachzahlung", view.FormatEURCents(balance)
	case balance < 0:
		return "Guthaben", view.FormatEURCents(-balance)
	default:
		return "Ausgeglichen", view.FormatEURCents(0)
	}
}
