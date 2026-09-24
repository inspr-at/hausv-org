package server

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	appmail "github.com/inspr-at/hausv-org/internal/mail"
	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/valorisationpdf"
	"github.com/inspr-at/hausv-org/internal/web"
)

func (a *app) valorisationRepo(ac authCtx) (*store.ValorisationRepository, bool) {
	docs, _ := a.documentStore.(*store.SQLDocumentStore)
	return store.BindValorisationRepository(a.tenantDB, docs, ac.tenantRef)
}
func valorisationActor(ac authCtx) store.ValorisationActor {
	return store.ValorisationActor{Email: ac.email, Manage: ac.can(capabilityManageLeases), Approve: ac.can(capabilityApproveValorisation)}
}
func (a *app) valorisationInput(r *http.Request, ac authCtx, effective string) (store.ValorisationInput, error) {
	input := store.ValorisationInput{EffectiveOn: effective, Settings: store.DefaultValorisationSettings(), House: houseDisplayName(ac.tenant), Address: ac.tenant.Address}
	if org, ok := a.organisationRecordFor(r.Context(), &ac); ok {
		input.Organisation = org.Name
		input.Contact = strings.Join([]string{org.ContactName, org.ContactEmail, org.ContactPhone}, " · ")
	}
	if a.orgSettings != nil {
		settings, err := a.orgSettings(ac.tenant.Organisation).Get(r.Context())
		if err != nil {
			return input, err
		}
		input.Settings = settings.Valorisation.Normalized()
	}
	return input, nil
}
func (a *app) valorisationPage(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if a.tenantDB == nil {
		http.Error(w, "Wertsicherung nicht verfügbar.", 503)
		return
	}
	now := time.Now()
	next := time.Date(now.Year(), time.April, 1, 0, 0, 0, 0, now.Location())
	if !next.After(now) {
		next = next.AddDate(1, 0, 0)
	}
	effective := r.URL.Query().Get("effective_on")
	if effective == "" {
		effective = next.Format(time.DateOnly)
	}
	page := web.ValorisationPageData{Portal: a.settingsPortalContext(ac, "Wertsicherung", "settings"), EffectiveOn: effective, New: r.URL.Query().Get("new") == "1", Organisation: strings.Contains(r.URL.Path, "/verwaltung/")}
	contexts := []authCtx{ac}
	if page.Organisation {
		contexts = nil
		access := a.organisationAccessFor(&ac)
		for _, house := range a.organisationManagedTenants(r.Context(), &ac) {
			if !access.managesHouse(house.Ref.Slug) {
				continue
			}
			ctx := ac
			ctx.tenant, ctx.tenantRef, ctx.role = house.Config, house.Ref, house.Role
			// Rebind effective capabilities as well as repositories for each house.
			if !can(a.actorFor(ac.email, house.Ref.Slug, house.Role), capabilityManageLeases, resourceFor(house.Ref.Slug)) {
				continue
			}
			ctx.repositories = a.repositoriesForTenant(house.Ref)
			contexts = append(contexts, ctx)
		}
	}
	for _, ctx := range contexts {
		chosen := len(r.URL.Query()["house"]) == 0
		for _, slug := range r.URL.Query()["house"] {
			if slug == ctx.tenant.Slug {
				chosen = true
			}
		}
		page.Houses = append(page.Houses, web.ValorisationHouseView{Slug: ctx.tenant.Slug, Name: houseDisplayName(ctx.tenant), Selected: chosen})
		repo, ok := a.valorisationRepo(ctx)
		if !ok {
			continue
		}
		target := "/" + ctx.tenant.Slug + "/app/settings/valorisation"
		canApprove := can(a.actorFor(ac.email, ctx.tenant.Slug, ctx.role), capabilityApproveValorisation, resourceFor(ctx.tenant.Slug))
		runs, err := repo.List()
		if err != nil {
			a.valorisationError(w, err)
			return
		}
		deliveryRepo, _ := store.BindValorisationDeliveryRepository(store.NewSQLValorisationDeliveryStore(a.tenantDB), ctx.tenantRef)
		for _, run := range runs {
			deliveries, err := deliveryRepo.List(run.ID)
			if err != nil {
				a.valorisationError(w, err)
				return
			}
			page.Runs = append(page.Runs, web.ValorisationRunView{Run: run, URL: target, CanApprove: canApprove, Groups: web.ValorisationGroups(run), Deliveries: deliveries})
		}
		if chosen && r.URL.Query().Get("preview") == "1" {
			input, err := a.valorisationInput(r, ctx, effective)
			if err != nil {
				a.valorisationError(w, err)
				return
			}
			run, err := repo.Preview(input, now)
			if err != nil {
				a.valorisationError(w, err)
				return
			}
			page.Previews = append(page.Previews, web.ValorisationRunView{Run: run, URL: target, CanApprove: canApprove, Groups: web.ValorisationGroups(run)})
		}
	}
	a.renderSettingsComponent(w, r, ac.tenant.Slug, web.ValorisationPage(page))
}
func (a *app) createValorisation(w http.ResponseWriter, r *http.Request, ac authCtx) {
	repo, ok := a.valorisationRepo(ac)
	if !ok {
		http.Error(w, "Wertsicherung nicht verfügbar.", 503)
		return
	}
	input, err := a.valorisationInput(r, ac, r.FormValue("effective_on"))
	if err != nil {
		a.valorisationError(w, err)
		return
	}
	run, err := repo.Create(input, ac.tenant.Organisation, valorisationActor(ac), time.Now())
	if err != nil {
		a.valorisationError(w, err)
		return
	}
	a.auditValorisation(ac, run, "valorisation_run.create", "Wertsicherungslauf angelegt", nil)
	redirectValorisation(w, r, run.ID)
}
func (a *app) approveValorisation(w http.ResponseWriter, r *http.Request, ac authCtx) {
	repo, ok := a.valorisationRepo(ac)
	if !ok {
		http.Error(w, "Wertsicherung nicht verfügbar.", 503)
		return
	}
	input, err := a.valorisationInput(r, ac, "")
	if err != nil {
		a.valorisationError(w, err)
		return
	}
	run, err := repo.Approve(r.PathValue("runID"), valorisationActor(ac), input.Settings, time.Now(), valorisationpdf.Render)
	if err != nil {
		a.valorisationError(w, err)
		return
	}
	a.auditValorisation(ac, run, "valorisation_run.approve", "Wertsicherung freigegeben", nil)
	for _, item := range run.Items {
		if item.LetterDocumentID != "" {
			a.auditValorisation(ac, run, "letter.archive", "Anpassungsschreiben archiviert", map[string]string{"document_id": item.LetterDocumentID, "sha256": item.LetterSHA256})
		}
	}
	redirectValorisation(w, r, run.ID)
}
func (a *app) cancelValorisation(w http.ResponseWriter, r *http.Request, ac authCtx) {
	repo, ok := a.valorisationRepo(ac)
	if !ok {
		http.Error(w, "Wertsicherung nicht verfügbar.", 503)
		return
	}
	if err := repo.Cancel(r.PathValue("runID"), r.FormValue("reason"), valorisationActor(ac), time.Now()); err != nil {
		a.valorisationError(w, err)
		return
	}
	run, _, _ := repo.Get(r.PathValue("runID"))
	a.auditValorisation(ac, run, "valorisation_run.cancel", "Wertsicherung storniert; Korrektur gesondert prüfen", map[string]string{"reason": r.FormValue("reason")})
	redirectValorisation(w, r, run.ID)
}
func (a *app) valorisationItemAction(w http.ResponseWriter, r *http.Request, ac authCtx) {
	repo, ok := a.valorisationRepo(ac)
	if !ok {
		http.Error(w, "Wertsicherung nicht verfügbar.", 503)
		return
	}
	run, found, err := repo.Get(r.PathValue("runID"))
	if err != nil {
		a.valorisationError(w, err)
		return
	}
	if !found {
		http.NotFound(w, r)
		return
	}
	action, reason := r.FormValue("action"), strings.TrimSpace(r.FormValue("reason"))
	if action == "review" {
		if run.Status != "draft" || reason == "" {
			a.valorisationError(w, store.ErrValorisationConflict)
			return
		}
		for _, item := range run.Items {
			if item.ID == r.PathValue("itemID") {
				if _, err := ac.repositories.leases.ReviewClause(item.ClauseID, store.ReviewOK, reason); err != nil {
					a.valorisationError(w, err)
					return
				}
				a.auditLease(ac, store.AuditActionClauseReview, item.ClauseID, "Wertsicherungsklausel geprüft", map[string]string{"reason": reason})
				input, err := a.valorisationInput(r, ac, run.EffectiveOn)
				if err != nil {
					a.valorisationError(w, err)
					return
				}
				next, err := repo.Create(input, run.OrgKey, valorisationActor(ac), time.Now())
				if err != nil {
					a.valorisationError(w, err)
					return
				}
				a.auditValorisation(ac, next, "valorisation_run.create", "Wertsicherung neu berechnet", nil)
				redirectValorisation(w, r, next.ID)
				return
			}
		}
		http.NotFound(w, r)
		return
	}
	var amount *int64
	if action == "override" {
		value, err := parseIssueEstimateAmountCents(r.FormValue("amount"))
		if err != nil {
			http.Error(w, "Betrag ungültig.", 400)
			return
		}
		amount = &value
	}
	if err := repo.ItemAction(run.ID, r.PathValue("itemID"), action, reason, amount, valorisationActor(ac), time.Now()); err != nil {
		a.valorisationError(w, err)
		return
	}
	a.auditValorisation(ac, run, "valorisation_item."+action, "Wertsicherung bearbeitet", map[string]string{"item_id": r.PathValue("itemID"), "reason": reason})
	redirectValorisation(w, r, run.ID)
}
func (a *app) valorisationPDF(w http.ResponseWriter, r *http.Request, ac authCtx) {
	repo, ok := a.valorisationRepo(ac)
	if !ok {
		http.Error(w, "Wertsicherung nicht verfügbar.", 503)
		return
	}
	run, found, err := repo.Get(r.PathValue("runID"))
	if err != nil {
		a.valorisationError(w, err)
		return
	}
	if !found {
		http.NotFound(w, r)
		return
	}
	for _, item := range run.Items {
		if item.ID != r.PathValue("itemID") {
			continue
		}
		var data []byte
		if item.LetterDocumentID != "" {
			doc, ok := ac.repositories.documents.Get(item.LetterDocumentID)
			if !ok {
				http.NotFound(w, r)
				return
			}
			data, err = readValorisationDocument(ac.repositories.documents, doc)
		} else {
			data, err = valorisationpdf.Render(run, item, time.Now())
		}
		if err != nil {
			a.valorisationError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", `attachment; filename="wertsicherung.pdf"`)
		w.Header().Set("Cache-Control", "private, no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		_, _ = w.Write(data)
		return
	}
	http.NotFound(w, r)
}
func readValorisationDocument(repo store.DocumentRepository, doc store.DocumentRecord) ([]byte, error) {
	if doc.ValorisationArchive == nil {
		return nil, fmt.Errorf("Schreiben ist nicht archiviert")
	}
	path, ok := repo.FilePath(doc)
	if !ok {
		return nil, fmt.Errorf("Archivdatei fehlt")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if fmt.Sprintf("%x", sha256.Sum256(data)) != doc.ValorisationArchive.SHA256 {
		return nil, fmt.Errorf("Prüfsumme des Schreibens stimmt nicht")
	}
	return data, nil
}
func (a *app) sendValorisation(w http.ResponseWriter, r *http.Request, ac authCtx) {
	repo, ok := a.valorisationRepo(ac)
	if !ok {
		http.Error(w, "Wertsicherung nicht verfügbar.", 503)
		return
	}
	run, found, err := repo.Get(r.PathValue("runID"))
	if err != nil {
		a.valorisationError(w, err)
		return
	}
	if !found {
		http.NotFound(w, r)
		return
	}
	if run.Status == "sent" {
		redirectValorisation(w, r, run.ID)
		return
	}
	if run.Status != "approved" {
		a.valorisationError(w, store.ErrValorisationConflict)
		return
	}
	if a.mailer == nil || !a.mailer.Configured() {
		http.Error(w, "E-Mail-Versand ist nicht eingerichtet.", 409)
		return
	}
	deliveries, _ := store.BindValorisationDeliveryRepository(store.NewSQLValorisationDeliveryStore(a.tenantDB), ac.tenantRef)
	now := time.Now()
	for _, item := range run.Items {
		if item.Excluded || item.Group != "ready" || item.Outcome == "unchanged" {
			continue
		}
		doc, err := repo.PrepareLetter(run.ID, item.ID, valorisationActor(ac), now, valorisationpdf.Render)
		if err != nil {
			a.valorisationError(w, err)
			return
		}
		item.LetterDocumentID, item.LetterSHA256 = doc.ID, doc.ValorisationArchive.SHA256
		for _, party := range item.Recipients {
			delivery := store.ValorisationDelivery{RunID: run.ID, Revision: run.Revision, PartyID: party.Email, UnitID: item.UnitID, DocumentID: item.LetterDocumentID, SHA256: item.LetterSHA256, Recipient: party.Email, Actor: ac.email}
			result, skip, err := deliveries.Attempt(r.Context(), delivery, func(ctx context.Context) error {
				data, err := readValorisationDocument(ac.repositories.documents, doc)
				if err != nil {
					return fmt.Errorf("Archivdatei konnte nicht geprüft werden")
				}
				a.auditValorisation(ac, run, "letter.archive", "Schreiben für Versand aktualisiert", map[string]string{"document_id": doc.ID, "sha256": doc.ValorisationArchive.SHA256, "letter_date": doc.ValorisationArchive.LetterDate})
				if err = a.mailer.SendDocument(ctx, party.Email, "Wertsicherung · "+run.Input.House, "Guten Tag,\n\nim Anhang erhalten Sie die Mitteilung über die Anpassung Ihres Hauptmietzinses.\n\n"+run.Input.Organisation, appmail.Attachment{Filename: doc.Filename, ContentType: "application/pdf", Data: data}); err != nil {
					return fmt.Errorf("E-Mail konnte nicht versendet werden. Bitte erneut versuchen")
				}
				return nil
			})
			if err != nil {
				a.valorisationError(w, err)
				return
			}
			if !skip {
				a.auditValorisation(ac, run, "letter.send", "Anpassungsschreiben: "+result.Status, map[string]string{"recipient": party.Email, "status": result.Status, "item_id": item.ID})
			}
		}
	}
	if err = repo.FinishSend(run.ID, valorisationActor(ac), now); err != nil {
		a.valorisationError(w, err)
		return
	}
	redirectValorisation(w, r, run.ID)
}
func redirectValorisation(w http.ResponseWriter, r *http.Request, id string) {
	http.Redirect(w, r, "/app/settings/valorisation#run-"+id, http.StatusSeeOther)
}
func (a *app) valorisationError(w http.ResponseWriter, err error) {
	logError("valorisation action failed", err)
	message := "Wertsicherung konnte nicht abgeschlossen werden. Bitte Eingaben, Ausnahmen und Freigabestatus prüfen."
	if err == store.ErrValorisationDenied {
		http.Error(w, "Keine Berechtigung.", 403)
		return
	}
	for _, prefix := range []string{"Vier-Augen", "Ausnahmen", "Schreiben vor", "Keine freigabefähige", "Manuelle Entscheidung", "Begründung", "Lauf geändert", "index_revised"} {
		if strings.HasPrefix(err.Error(), prefix) {
			message = err.Error()
		}
	}
	http.Error(w, message, 409)
}
func (a *app) auditValorisation(ac authCtx, run store.ValorisationRun, action, summary string, details map[string]string) {
	if details == nil {
		details = map[string]string{}
	}
	details["inputs_sha256"] = run.InputsSHA256
	a.recordAudit(store.AuditEvent{TenantSlug: ac.tenant.Slug, ActorEmail: ac.email, ActorRole: ac.role, Action: action, TargetType: "valorisation_run", TargetID: run.ID, Summary: summary, Details: details})
}
