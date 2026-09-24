package server

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/indexation"
	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/view"
	"github.com/inspr-at/hausv-org/internal/web"
)

const maxLeaseImportBytes = 1 << 20

func (a *app) unitLeasePage(w http.ResponseWriter, r *http.Request, ac authCtx) {
	unit, repo, ok := a.leaseUnit(w, r, ac)
	if !ok {
		return
	}
	history, err := repo.History(unit.ID)
	if err != nil {
		logError("lease list failed", err, "tenant", ac.tenant.Slug, "unit", unit.ID)
		http.Error(w, "Der Mietvertrag konnte nicht geladen werden.", http.StatusInternalServerError)
		return
	}
	current, has := currentHauptmiete(history)
	message, messageOK := leaseFlash(r.URL.Query().Get("lease"))
	canEnd := has && current.Status == store.LeaseStatusActive
	editing := r.URL.Query().Get("bearbeiten") == "1"
	ending := canEnd && r.URL.Query().Get("beenden") == "1"
	if ending {
		editing = false
	}
	page := web.LeasePageData{
		Portal:    a.settingsPortalContext(ac, "Mietvertrag", "settings"),
		UnitID:    unit.ID,
		UnitLabel: unit.Label,
		Subtitle:  leaseSubtitle(unit.Label, current, has),
		Editing:   editing,
		Ending:    ending,
		HasLease:  has,
		History:   leaseDetails(history),
		Form:      leaseForm(current, has),
		Message:   message,
		OK:        messageOK,
		CanEnd:    canEnd,
	}
	if has {
		page.Lease = leaseDetail(current)
		if runs, ok := a.valorisationRepo(ac); ok {
			list, err := runs.List()
			if err != nil {
				a.valorisationError(w, err)
				return
			}
			for _, run := range list {
				for _, item := range run.Items {
					if item.LeaseID != current.ID {
						continue
					}
					entry := web.LeaseValorisationView{URL: "/app/settings/valorisation#item-" + item.ID, Date: web.LeaseDate(run.EffectiveOn), Amount: web.LeaseMoney(item.NewCents), Due: web.LeaseDate(item.CollectableFrom), Status: valorisationHistoryStatus(run.Status)}
					if item.LetterDocumentID != "" {
						entry.PDFURL = "/app/settings/valorisation/runs/" + run.ID + "/items/" + item.ID + "/pdf"
					}
					page.Lease.Valorisation = append(page.Lease.Valorisation, entry)
				}
			}
		}
	}
	a.renderSettingsComponent(w, r, ac.tenant.Slug, web.LeasePage(page))
}

func (a *app) saveUnitLease(w http.ResponseWriter, r *http.Request, ac authCtx) {
	unit, repo, ok := a.leaseUnit(w, r, ac)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		a.redirectLease(w, r, unit.ID, "invalid")
		return
	}
	lease, party, component, clause, addComponent, err := leaseFromForm(r, unit.ID, ac.email)
	if err != nil {
		a.redirectLease(w, r, unit.ID, "invalid")
		return
	}
	if strings.TrimSpace(r.FormValue("lease_id")) == "" {
		lease.Parties = []store.LeaseParty{party}
		if addComponent {
			lease.Components = []store.RentComponent{component}
		}
		lease.Clauses = []store.IndexClause{clause}
		saved, err := repo.Create(lease)
		if err != nil {
			a.redirectLease(w, r, unit.ID, leaseErrorCode(err))
			return
		}
		a.auditLease(ac, store.AuditActionLeaseCreate, saved.ID, "Mietvertrag angelegt", map[string]string{"unit_id": unit.ID})
		a.auditLease(ac, store.AuditActionLeasePartyChange, saved.ID, "Vertragspartei geändert", map[string]string{"unit_id": unit.ID})
		if addComponent {
			a.auditLease(ac, store.AuditActionRentComponentAdd, saved.ID, "Mietzinsbestandteil ergänzt", map[string]string{"unit_id": unit.ID, "kind": component.Kind})
		}
		a.auditLease(ac, store.AuditActionClauseCreate, saved.ID, "Wertsicherungsklausel angelegt", map[string]string{"unit_id": unit.ID})
		a.redirectLease(w, r, unit.ID, "saved")
		return
	}
	current, found, err := repo.Get(lease.ID)
	if err != nil || !found || current.UnitID != unit.ID {
		a.redirectLease(w, r, unit.ID, "invalid")
		return
	}
	if _, err := repo.Update(lease); err != nil {
		a.redirectLease(w, r, unit.ID, leaseErrorCode(err))
		return
	}
	a.auditLease(ac, store.AuditActionLeaseUpdate, lease.ID, "Mietvertrag geändert", map[string]string{"unit_id": unit.ID})
	parties := replaceHauptmieter(current.Parties, party)
	if err := repo.ReplaceParties(lease.ID, parties); err != nil {
		a.redirectLease(w, r, unit.ID, "error")
		return
	}
	a.auditLease(ac, store.AuditActionLeasePartyChange, lease.ID, "Vertragspartei geändert", map[string]string{"unit_id": unit.ID})
	if addComponent {
		if _, err := repo.AddRentComponent(lease.ID, component); err != nil {
			a.redirectLease(w, r, unit.ID, "error")
			return
		}
		a.auditLease(ac, store.AuditActionRentComponentAdd, lease.ID, "Mietzinsbestandteil ergänzt", map[string]string{"unit_id": unit.ID, "kind": component.Kind})
	}
	if len(current.Clauses) > 0 && clause.ID == "" {
		clause.ID = current.Clauses[0].ID
	}
	previousReview := ""
	if len(current.Clauses) > 0 {
		previousReview = current.Clauses[0].ReviewStatus
	}
	savedClause, err := repo.SetClause(clause)
	if err != nil {
		a.redirectLease(w, r, unit.ID, "error")
		return
	}
	action := store.AuditActionClauseUpdate
	summary := "Wertsicherungsklausel geändert"
	if len(current.Clauses) == 0 {
		action = store.AuditActionClauseCreate
		summary = "Wertsicherungsklausel angelegt"
	}
	a.auditLease(ac, action, lease.ID, summary, map[string]string{"unit_id": unit.ID, "clause_id": savedClause.ID})
	if savedClause.ReviewStatus != previousReview {
		if _, err := repo.ReviewClause(savedClause.ID, savedClause.ReviewStatus, savedClause.ReviewNote); err != nil {
			a.redirectLease(w, r, unit.ID, "error")
			return
		}
		a.auditLease(ac, store.AuditActionClauseReview, lease.ID, "Klausel geprüft", map[string]string{"unit_id": unit.ID, "review": savedClause.ReviewStatus})
	}
	a.redirectLease(w, r, unit.ID, "saved")
}

func (a *app) endUnitLease(w http.ResponseWriter, r *http.Request, ac authCtx) {
	unit, repo, ok := a.leaseUnit(w, r, ac)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		a.redirectLease(w, r, unit.ID, "invalid")
		return
	}
	ended, err := repo.End(r.FormValue("lease_id"), r.FormValue("ends_on"), ac.email)
	if err != nil {
		a.redirectLease(w, r, unit.ID, leaseErrorCode(err))
		return
	}
	a.auditLease(ac, store.AuditActionLeaseEnd, ended.ID, "Mietvertrag beendet", map[string]string{"unit_id": unit.ID, "ends_on": ended.EndsOn})
	a.redirectLease(w, r, unit.ID, "ended")
}

func (a *app) leaseImportPage(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if ac.repositories.leases == nil {
		http.Error(w, "Mietverträge sind nicht verfügbar.", http.StatusInternalServerError)
		return
	}
	message, ok := leaseImportFlash(r.URL.Query().Get("import"), r.URL.Query().Get("count"))
	a.renderSettingsComponent(w, r, ac.tenant.Slug, web.LeaseImportPage(web.LeaseImportPageData{
		Portal:  a.settingsPortalContext(ac, "Mietverträge importieren", "settings"),
		Message: message,
		OK:      ok,
	}))
}

func (a *app) leaseImport(w http.ResponseWriter, r *http.Request, ac authCtx) {
	repo := ac.repositories.leases
	if repo == nil || ac.repositories.units == nil {
		http.Error(w, "Mietverträge sind nicht verfügbar.", http.StatusInternalServerError)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxLeaseImportBytes+(64<<10))
	if err := r.ParseMultipartForm(maxLeaseImportBytes); err != nil {
		a.renderLeaseImport(w, r, ac, nil, "Die Datei konnte nicht gelesen werden.", false)
		return
	}
	file, header, err := r.FormFile("leases_file")
	if err != nil || header == nil || header.Size <= 0 || header.Size > maxLeaseImportBytes {
		a.renderLeaseImport(w, r, ac, nil, "Bitte eine CSV-Datei wählen.", false)
		return
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, maxLeaseImportBytes+1))
	if err != nil || len(raw) > maxLeaseImportBytes {
		a.renderLeaseImport(w, r, ac, nil, "Die Datei ist zu groß.", false)
		return
	}
	drafts, err := store.ParseLeaseCSV(raw)
	if err != nil {
		a.renderLeaseImport(w, r, ac, nil, "Die Spalten der Tabelle stimmen nicht. Einheit, Abschluss, Beginn, Nutzung, MRG, Regime und HMZ werden gebraucht.", false)
		return
	}
	commit := r.FormValue("commit") == "1"
	report, err := repo.Import(drafts, ac.repositories.units.List(), commit, ac.email)
	if err != nil {
		logError("lease import failed", err, "tenant", ac.tenant.Slug)
		a.renderLeaseImport(w, r, ac, nil, "Der Import konnte nicht gespeichert werden.", false)
		return
	}
	if commit && !report.HasErrors() {
		a.auditLease(ac, store.AuditActionLeaseCreate, ac.tenant.Slug, "Mietverträge importiert", map[string]string{"count": strconv.Itoa(report.Committed)})
		http.Redirect(w, r, "/app/settings/building/leases/import?import=saved&count="+strconv.Itoa(report.Committed), http.StatusSeeOther)
		return
	}
	message := "Prüfung ohne Speichern."
	if commit && report.HasErrors() {
		message = "Nichts übernommen. Bitte die markierten Zeilen korrigieren."
	}
	a.renderLeaseImport(w, r, ac, &report, message, !report.HasErrors())
}

func (a *app) renderLeaseImport(w http.ResponseWriter, r *http.Request, ac authCtx, report *store.LeaseImportReport, message string, ok bool) {
	page := web.LeaseImportPageData{
		Portal:  a.settingsPortalContext(ac, "Mietverträge importieren", "settings"),
		Message: message,
		OK:      ok,
	}
	if report != nil {
		page.HasRows = len(report.Rows) > 0
		page.Committed = report.Committed
		page.Rows = leaseImportRows(report.Rows)
	}
	a.renderSettingsComponent(w, r, ac.tenant.Slug, web.LeaseImportPage(page))
}

func (a *app) leaseUnit(w http.ResponseWriter, r *http.Request, ac authCtx) (store.Unit, store.LeaseRepository, bool) {
	repo := ac.repositories.leases
	if repo == nil || ac.repositories.units == nil {
		http.Error(w, "Mietverträge sind nicht verfügbar.", http.StatusInternalServerError)
		return store.Unit{}, nil, false
	}
	unit, found := leaseUnitByID(ac.repositories.units.List(), r.PathValue("unitID"))
	if !found {
		http.NotFound(w, r)
		return store.Unit{}, nil, false
	}
	return unit, repo, true
}

func leaseUnitByID(units []store.Unit, raw string) (store.Unit, bool) {
	id := store.NormalizeUnitID(raw)
	for _, unit := range units {
		if unit.ID == id {
			return unit, true
		}
	}
	return store.Unit{}, false
}

func (a *app) redirectLease(w http.ResponseWriter, r *http.Request, unitID, code string) {
	target := "/app/settings/building/units/" + unitID + "/lease?lease=" + code
	if code != "saved" && code != "ended" {
		target += "&bearbeiten=1"
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func (a *app) auditLease(ac authCtx, action, id, summary string, details map[string]string) {
	a.recordAudit(store.AuditEvent{
		TenantSlug: ac.tenant.Slug, ActorEmail: ac.email, ActorRole: ac.role,
		Action: action, TargetType: "lease", TargetID: id, Summary: summary, Details: details,
	})
}

func leaseErrorCode(err error) string {
	switch {
	case errors.Is(err, store.ErrLeaseOverlap):
		return "overlap"
	case errors.Is(err, store.ErrLeaseInvalid), errors.Is(err, store.ErrUnitNotFound), errors.Is(err, store.ErrLeaseNotFound):
		return "invalid"
	default:
		return "error"
	}
}

func leaseFlash(code string) (string, bool) {
	switch code {
	case "saved":
		return "Der Mietvertrag wurde gespeichert.", true
	case "ended":
		return "Der Mietvertrag wurde beendet.", true
	case "overlap":
		return "Für diese Einheit gibt es in dem Zeitraum schon eine Hauptmiete.", false
	case "invalid":
		return "Bitte die Angaben prüfen. Datum, Regime und Beträge müssen vollständig sein.", false
	case "error":
		return "Der Mietvertrag konnte nicht gespeichert werden.", false
	default:
		return "", false
	}
}

func leaseImportFlash(code, count string) (string, bool) {
	switch code {
	case "saved":
		return count + " Mietverträge wurden übernommen.", true
	default:
		return "", false
	}
}

func currentHauptmiete(leases []store.Lease) (store.Lease, bool) {
	var latest store.Lease
	found := false
	for _, lease := range leases {
		if lease.LeaseKind != store.LeaseKindHauptmiete {
			continue
		}
		if lease.Status == store.LeaseStatusActive {
			return lease, true
		}
		if !found || lease.StartsOn > latest.StartsOn {
			latest = lease
			found = true
		}
	}
	return latest, found
}

func leaseFromForm(r *http.Request, unitID, actor string) (store.Lease, store.LeaseParty, store.RentComponent, store.IndexClause, bool, error) {
	value := r.FormValue
	lease := store.Lease{
		ID: value("lease_id"), UnitID: unitID, Status: store.LeaseStatusActive,
		ConcludedOn: value("concluded_on"), StartsOn: value("starts_on"), EndsOn: value("ends_on"),
		LeaseKind: value("lease_kind"), UseKind: value("use_kind"), MRGScope: value("mrg_scope"), RentRegime: value("rent_regime"),
		PriceRestrictedSet: value("price_restricted_set") == "1", PriceRestricted: checked(r, "price_restricted"),
		LandlordIsBusiness: checked(r, "landlord_is_business"), TenantIsConsumer: checked(r, "tenant_is_consumer"),
		VATOpted: value("component_vat") != "0" && value("component_vat") != "",
		Notes:    value("notes"), UpdatedAt: time.Now().UTC(), UpdatedBy: actor,
	}
	if day, err := strconv.Atoi(value("zinstermin_day")); err == nil {
		lease.ZinsterminDay = day
	}
	party := store.LeaseParty{
		Name: value("party_name"), Email: value("party_email"), Address: value("party_address"),
		Role: value("party_role"), ValidFrom: firstDate(value("party_from"), lease.StartsOn), ValidTo: value("party_to"),
	}
	if party.Name == "" {
		party.Name = "Mieter"
		party.Role = store.PartyHauptmieter
	}
	addComponent := strings.TrimSpace(value("component_net")) != ""
	component := store.RentComponent{Kind: value("component_kind"), ValidFrom: firstDate(value("component_from"), lease.StartsOn), Origin: store.OriginManual}
	if addComponent {
		cents, err := store.ParseMoneyCents(value("component_net"))
		if err != nil {
			return store.Lease{}, store.LeaseParty{}, store.RentComponent{}, store.IndexClause{}, false, err
		}
		component.NetCents = cents
		component.VATRateBP = vatBasisPoints(value("component_vat"))
	}
	clause := store.IndexClause{
		ID: value("clause_id"), LeaseID: lease.ID, ClauseType: value("clause_type"), Series: value("series"),
		BasePeriod: value("base_period"), BaseValue: value("base_value"), ThresholdKind: value("threshold_kind"),
		ThresholdValue: value("threshold"), ThresholdInclusive: checked(r, "threshold_inclusive"),
		FullChangeOnTrigger: true, TwoWay: checked(r, "two_way"), ClauseText: value("clause_text"),
		ReviewStatus: value("review_status"), ValidFrom: lease.StartsOn,
	}
	if clause.ClauseType == store.ClauseStaffel {
		dates, kinds, values := r.Form["staffel_date"], r.Form["staffel_kind"], r.Form["staffel_value"]
		if len(dates) == 0 || len(dates) != len(kinds) || len(dates) != len(values) || len(dates) > indexation.MaxStaffelSteps {
			return lease, party, component, clause, addComponent, store.ErrLeaseInvalid
		}
		for i, date := range dates {
			v := strings.TrimSpace(values[i])
			if kinds[i] == "percent" {
				v += "%"
			} else if kinds[i] != "amount" {
				return lease, party, component, clause, addComponent, store.ErrLeaseInvalid
			}
			step, err := store.ParseStaffelStep(date, v)
			if err != nil || date <= clause.ValidFrom {
				return lease, party, component, clause, addComponent, store.ErrLeaseInvalid
			}
			clause.StaffelSteps = append(clause.StaffelSteps, step)
		}
		if err := indexation.ValidateStaffelSteps(clause.StaffelSteps); err != nil {
			return lease, party, component, clause, addComponent, err
		}
	}
	return lease, party, component, clause, addComponent, nil
}

func checked(r *http.Request, name string) bool {
	value := r.FormValue(name)
	return value == "on" || value == "1" || value == "true"
}

func firstDate(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func vatBasisPoints(raw string) int {
	switch strings.TrimSpace(raw) {
	case "0":
		return 0
	case "20":
		return 2000
	default:
		return 1000
	}
}

func replaceHauptmieter(current []store.LeaseParty, party store.LeaseParty) []store.LeaseParty {
	kept := []store.LeaseParty{}
	for _, item := range current {
		if item.Role != store.PartyHauptmieter {
			kept = append(kept, item)
		}
	}
	return append([]store.LeaseParty{party}, kept...)
}

func leaseDetails(leases []store.Lease) []web.LeaseDetail {
	out := make([]web.LeaseDetail, 0, len(leases))
	for _, lease := range leases {
		out = append(out, leaseDetail(lease))
	}
	return out
}

func leaseDetail(lease store.Lease) web.LeaseDetail {
	class := store.ClassifyLease(lease)
	detail := web.LeaseDetail{
		ID: lease.ID, Status: leaseStatusLabel(lease.Status), Kind: leaseKindLabel(lease.LeaseKind),
		Use: useKindLabel(lease.UseKind), MRG: mrgLabel(lease.MRGScope), Regime: regimeLabel(lease.RentRegime),
		Concluded: web.LeaseDate(lease.ConcludedOn), Starts: web.LeaseDate(lease.StartsOn), Ends: web.LeaseDate(lease.EndsOn),
		Zinstermin: strconv.Itoa(lease.ZinsterminDay) + ". des Monats", Notes: lease.Notes,
		MieWeG: yesNo(class.MieWeG), SpecialCap: yesNo(class.SpecialCap),
		MieWeGReason: class.MieWeGReason, CapReason: class.CapReason,
	}
	for _, party := range lease.Parties {
		detail.Parties = append(detail.Parties, web.LeasePartyView{
			Name: party.Name, Email: party.Email, Address: party.Address, Role: partyRoleLabel(party.Role),
			From: web.LeaseDate(party.ValidFrom), To: web.LeaseDate(party.ValidTo),
		})
	}
	today := time.Now().UTC().Format("2006-01-02")
	currentMoney := currentRentComponents(lease.Components, today)
	var netSum, vatSum int64
	for _, component := range lease.Components {
		gross := web.LeaseGrossCents(component.NetCents, component.VATRateBP)
		detail.Components = append(detail.Components, web.LeaseMoneyView{
			Kind: componentKindLabel(component.Kind), From: web.LeaseDate(component.ValidFrom),
			Net: web.LeaseMoney(component.NetCents), VAT: web.LeaseMoney(gross - component.NetCents),
			Gross: web.LeaseMoney(gross), Origin: originLabel(component.Origin),
		})
	}
	for _, component := range currentMoney {
		gross := web.LeaseGrossCents(component.NetCents, component.VATRateBP)
		netSum += component.NetCents
		vatSum += gross - component.NetCents
	}
	if len(currentMoney) > 0 {
		detail.HasMonthly = true
		detail.MonthlyNet = web.LeaseMoney(netSum)
		detail.MonthlyVAT = web.LeaseMoney(vatSum)
		detail.MonthlyGross = web.LeaseMoney(netSum + vatSum)
	}
	if len(lease.Clauses) > 0 && lease.Clauses[0].ClauseType != store.ClauseNone {
		clause := lease.Clauses[0]
		detail.HasClause = true
		detail.Clause = web.LeaseClauseView{
			Type: clauseTypeLabel(clause.ClauseType), Series: web.LeaseSeriesLabel(clause.Series),
			Base:      web.LeaseMonthName(clause.BasePeriod) + " = " + strings.ReplaceAll(clause.BaseValue, ".", ","),
			Line:      web.LeaseIndexLine(clause.Series, clause.BasePeriod, clause.BaseValue),
			Threshold: thresholdLabel(clause), Text: clause.ClauseText,
			Review: reviewLabel(clause.ReviewStatus), ReviewClass: reviewClass(clause.ReviewStatus), Note: clause.ReviewNote,
		}
		if clause.ClauseType == store.ClauseStaffel {
			detail.Clause.Line, detail.Clause.Threshold = "", ""
			for _, step := range clause.StaffelSteps {
				value := "+" + strings.ReplaceAll(step.Percent, ".", ",") + " % auf den vorigen Vertragsbetrag"
				if step.NetCents != nil {
					value = web.LeaseMoney(*step.NetCents) + " HMZ netto"
				}
				detail.Clause.Staffel = append(detail.Clause.Staffel, "Ab "+web.LeaseDate(step.EffectiveOn)+": "+value)
			}
		}
		if clause.State != nil && clause.State.ContractBasePeriod != "" {
			detail.Anchor = "Ausgangswert " + strings.ReplaceAll(clause.State.ContractValue, ".", ",") + " €, Index " + web.LeaseMonth(clause.State.ContractBasePeriod) + " = " + strings.ReplaceAll(clause.State.ContractBaseValue, ".", ",") + "."
		}
	}
	return detail
}

func leaseForm(lease store.Lease, has bool) web.LeaseForm {
	form := web.LeaseForm{
		Zinstermin: "5", Consumer: true, Business: true, TwoWay: true,
		Kinds:          leaseOptions(store.LeaseKindHauptmiete, "hauptmiete", "Hauptmiete", "untermiete", "Untermiete"),
		Uses:           leaseOptions(store.UseKindWohnung, "wohnung", "Wohnung", "geschaeft", "Geschäft", "garage", "Garage", "sonstiges", "Sonstiges"),
		Scopes:         leaseOptions(store.MRGTeil, "voll", "Vollanwendung", "teil", "Teilanwendung", "ausnahme", "Ausnahme", "wgg", "WGG"),
		Regimes:        leaseOptions(store.RentRegimeFrei, "richtwert", "Richtwert", "kategorie", "Kategorie", "angemessen", "Angemessen", "frei", "Frei", "sonstig", "Sonstig"),
		PartyRoles:     leaseOptions(store.PartyHauptmieter, "hauptmieter", "Hauptmieter", "mitmieter", "Mitmieter"),
		ComponentKinds: leaseOptions(store.ComponentHMZ, "hmz", "Hauptmietzins", "bk_akonto", "BK-Akonto", "heiz_akonto", "Heizungsakonto", "lift", "Lift", "moebel", "Möbel", "stellplatz", "Stellplatz", "sonstiges", "Sonstiges"),
		VATRates:       leaseOptions("10", "0", "0 %", "10", "10 %", "20", "20 %"),
		ClauseTypes:    leaseOptions(store.ClauseVPIThreshold, "mieweg_model", "MieWeG-Modell", "vpi_threshold", "VPI mit Schwelle", "vpi_periodic", "VPI periodisch", "staffel", "Staffel", "none", "Keine"),
		ThresholdKinds: leaseOptions("percent", "percent", "%", "points", "Punkte"),
		Reviews:        leaseOptions(store.ReviewUnreviewed, "unreviewed", "ungeprüft", "ok", "geprüft", "doubtful", "zweifelhaft", "invalid", "ungültig"),
		SeriesOptions:  leaseSeriesOptions(""),
	}
	if !has {
		return form
	}
	form.ID = lease.ID
	form.Concluded = lease.ConcludedOn
	form.Starts = lease.StartsOn
	form.Ends = lease.EndsOn
	form.Notes = lease.Notes
	form.Zinstermin = strconv.Itoa(lease.ZinsterminDay)
	form.PriceRestricted = lease.PriceRestricted
	form.Consumer = lease.TenantIsConsumer
	form.Business = lease.LandlordIsBusiness
	form.Kinds = leaseOptions(lease.LeaseKind, "hauptmiete", "Hauptmiete", "untermiete", "Untermiete")
	form.Uses = leaseOptions(lease.UseKind, "wohnung", "Wohnung", "geschaeft", "Geschäft", "garage", "Garage", "sonstiges", "Sonstiges")
	form.Scopes = leaseOptions(lease.MRGScope, "voll", "Vollanwendung", "teil", "Teilanwendung", "ausnahme", "Ausnahme", "wgg", "WGG")
	form.Regimes = leaseOptions(lease.RentRegime, "richtwert", "Richtwert", "kategorie", "Kategorie", "angemessen", "Angemessen", "frei", "Frei", "sonstig", "Sonstig")
	if len(lease.Parties) > 0 {
		party := lease.Parties[0]
		for _, item := range lease.Parties {
			if item.Role == store.PartyHauptmieter {
				party = item
			}
		}
		form.PartyName = party.Name
		form.PartyEmail = party.Email
		form.PartyAddress = party.Address
		form.PartyFrom = party.ValidFrom
		form.PartyTo = party.ValidTo
		form.PartyRoles = leaseOptions(party.Role, "hauptmieter", "Hauptmieter", "mitmieter", "Mitmieter")
	}
	if len(lease.Clauses) > 0 {
		clause := lease.Clauses[0]
		form.ClauseID = clause.ID
		form.Series = clause.Series
		form.SeriesOptions = leaseSeriesOptions(clause.Series)
		form.BasePeriod = clause.BasePeriod
		form.BaseValue = strings.ReplaceAll(clause.BaseValue, ".", ",")
		form.Threshold = strings.ReplaceAll(clause.ThresholdValue, ".", ",")
		form.ClauseText = clause.ClauseText
		for _, step := range clause.StaffelSteps {
			row := web.LeaseStaffelRow{Date: step.EffectiveOn, Value: strings.ReplaceAll(step.Percent, ".", ","), Percent: step.Percent != ""}
			if step.NetCents != nil {
				row.Value = strings.TrimSuffix(web.LeaseMoney(*step.NetCents), " €")
			}
			form.Staffel = append(form.Staffel, row)
		}
		form.TwoWay = clause.TwoWay
		form.Inclusive = clause.ThresholdInclusive
		form.ClauseTypes = leaseOptions(clause.ClauseType, "mieweg_model", "MieWeG-Modell", "vpi_threshold", "VPI mit Schwelle", "vpi_periodic", "VPI periodisch", "staffel", "Staffel", "none", "Keine")
		form.ThresholdKinds = leaseOptions(clause.ThresholdKind, "percent", "Prozent", "points", "Punkte")
		form.Reviews = leaseOptions(clause.ReviewStatus, "unreviewed", "ungeprüft", "ok", "geprüft", "doubtful", "zweifelhaft", "invalid", "ungültig")
	}
	return form
}

func leaseOptions(selected string, pairs ...string) []view.SelectOption {
	out := make([]view.SelectOption, 0, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		out = append(out, view.SelectOption{Value: pairs[i], Label: pairs[i+1], Selected: pairs[i] == selected})
	}
	return out
}

func leaseImportRows(rows []store.LeaseImportRow) []web.LeaseImportRowView {
	out := make([]web.LeaseImportRowView, 0, len(rows))
	for _, row := range rows {
		out = append(out, web.LeaseImportRowView{
			Line: row.Line, Unit: row.Unit, HasErrors: len(row.Errors) > 0, HasWarnings: len(row.Warnings) > 0,
			Errors: strings.Join(leaseIssueLabels(row.Errors), ", "), Warnings: strings.Join(leaseIssueLabels(row.Warnings), ", "),
		})
	}
	return out
}

func leaseIssueLabels(codes []string) []string {
	out := make([]string, 0, len(codes))
	for _, code := range codes {
		switch code {
		case "unknown_unit":
			out = append(out, "Einheit ist nicht vorhanden")
		case "invalid_amount":
			out = append(out, "Betrag ist ungültig")
		case "invalid_staffel":
			out = append(out, "Staffelstufen prüfen: eindeutige aufsteigende Daten und je ein Betrag oder Prozentsatz erforderlich.")
		case "invalid_lease":
			out = append(out, "Vertragsdaten sind ungültig")
		case "overlap":
			out = append(out, "Hauptmiete überschneidet sich")
		case store.ImportWarningIndexUnchecked:
			out = append(out, "Indexwert wurde nicht geprüft")
		case store.ImportWarningBaseMismatch:
			out = append(out, "Basiswert weicht vom veröffentlichten Index ab")
		default:
			out = append(out, code)
		}
	}
	return out
}

func yesNo(value bool) string {
	if value {
		return "ja"
	}
	return "nein"
}

func leaseStatusLabel(raw string) string {
	switch raw {
	case store.LeaseStatusDraft:
		return "Entwurf"
	case store.LeaseStatusEnded:
		return "beendet"
	default:
		return "laufend"
	}
}

func leaseKindLabel(raw string) string {
	if raw == store.LeaseKindUntermiete {
		return "Untermiete"
	}
	return "Hauptmiete"
}

func useKindLabel(raw string) string {
	switch raw {
	case store.UseKindGeschaeft:
		return "Geschäft"
	case store.UseKindGarage:
		return "Garage"
	case store.UseKindSonstiges:
		return "Sonstiges"
	default:
		return "Wohnung"
	}
}

func mrgLabel(raw string) string {
	switch raw {
	case store.MRGVoll:
		return "MRG-Vollanwendung"
	case store.MRGTeil:
		return "MRG-Teilanwendung"
	case store.MRGAusnahme:
		return "MRG-Ausnahme"
	case store.MRGWGG:
		return "WGG"
	default:
		return raw
	}
}

func regimeLabel(raw string) string {
	switch raw {
	case store.RentRegimeRichtwert:
		return "Richtwertmietzins"
	case store.RentRegimeKategorie:
		return "Kategoriemietzins"
	case store.RentRegimeAngemessen:
		return "angemessener Mietzins"
	case store.RentRegimeFrei:
		return "freier Mietzins"
	default:
		return "sonstiger Mietzins"
	}
}

func partyRoleLabel(raw string) string {
	if raw == store.PartyMitmieter {
		return "Mitmieter"
	}
	return "Hauptmieter"
}

func componentKindLabel(raw string) string {
	switch raw {
	case store.ComponentHMZ:
		return "Hauptmietzins"
	case store.ComponentBKAkonto:
		return "BK-Akonto"
	case store.ComponentHeizAkonto:
		return "Heizungsakonto"
	case store.ComponentLift:
		return "Lift"
	case store.ComponentMoebel:
		return "Möbel"
	case store.ComponentStellplatz:
		return "Stellplatz"
	default:
		return "Sonstiges"
	}
}

func vatLabel(bp int) string {
	switch bp {
	case 0:
		return "0 %"
	case 2000:
		return "20 %"
	default:
		return "10 %"
	}
}

func originLabel(raw string) string {
	if strings.HasPrefix(raw, "valorisation_item:") {
		return "Wertsicherung"
	}
	if raw == store.OriginImport {
		return "Import"
	}
	return "manuell"
}

func clauseTypeLabel(raw string) string {
	switch raw {
	case store.ClauseMieWeG:
		return "MieWeG-Modell"
	case store.ClauseVPIThreshold:
		return "VPI mit Schwelle"
	case store.ClauseVPIPeriodic:
		return "VPI periodisch"
	case store.ClauseStaffel:
		return "Staffel"
	default:
		return "Keine Klausel"
	}
}

func reviewLabel(raw string) string {
	switch raw {
	case store.ReviewOK:
		return "geprüft"
	case store.ReviewDoubtful:
		return "zweifelhaft"
	case store.ReviewInvalid:
		return "ungültig"
	default:
		return "ungeprüft"
	}
}

func reviewClass(raw string) string {
	switch raw {
	case store.ReviewOK:
		return "is-ok"
	case store.ReviewDoubtful:
		return "is-doubt"
	case store.ReviewInvalid:
		return "is-bad"
	default:
		return "is-open"
	}
}

func leaseSubtitle(unitLabel string, lease store.Lease, has bool) string {
	if !has {
		return unitLabel
	}
	name := ""
	for _, party := range lease.Parties {
		if party.Role == store.PartyHauptmieter || name == "" {
			name = party.Name
		}
		if party.Role == store.PartyHauptmieter {
			break
		}
	}
	since := web.LeaseDate(lease.StartsOn)
	if name == "" {
		return unitLabel + " · seit " + since
	}
	return unitLabel + " · " + name + " · seit " + since
}

func currentRentComponents(items []store.RentComponent, today string) []store.RentComponent {
	best := map[string]store.RentComponent{}
	for _, item := range items {
		if item.ValidFrom > today {
			continue
		}
		prev, ok := best[item.Kind]
		if !ok || item.ValidFrom >= prev.ValidFrom {
			best[item.Kind] = item
		}
	}
	out := make([]store.RentComponent, 0, len(best))
	for _, item := range best {
		out = append(out, item)
	}
	return out
}

func leaseSeriesOptions(selected string) []view.SelectOption {
	years := []string{"2025", "2020", "2015", "2010", "2005", "2000", "1996", "1986", "1976", "1966"}
	out := make([]view.SelectOption, 0, len(years))
	known := false
	for _, year := range years {
		value := "vpi" + year
		if value == selected {
			known = true
		}
		out = append(out, view.SelectOption{Value: value, Label: "VPI " + year, Selected: value == selected})
	}
	if selected != "" && !known {
		out = append(out, view.SelectOption{Value: selected, Label: web.LeaseSeriesLabel(selected), Selected: true})
	}
	return out
}

func thresholdLabel(clause store.IndexClause) string {
	if clause.ThresholdValue == "" {
		return ""
	}
	kind := "Punkte"
	if clause.ThresholdKind == "percent" {
		kind = "%"
	}
	text := strings.ReplaceAll(clause.ThresholdValue, ".", ",") + " " + kind
	if clause.ThresholdInclusive {
		return text + ", inklusive"
	}
	return text
}

func valorisationHistoryStatus(status string) string {
	switch status {
	case "draft":
		return "Entwurf"
	case "approved":
		return "Freigegeben"
	case "sent":
		return "Versandt"
	case "cancelled":
		return "Storniert"
	}
	return status
}
