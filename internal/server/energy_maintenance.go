package server

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/energy"
	"github.com/inspr-at/hausv-org/internal/store"
)

type energyMaintenanceView struct {
	ID             string
	AssetID        string
	AssetName      string
	Title          string
	IntervalMonths int
	NextDueValue   string
	DueLabel       string
	Tone           string
	LastCompleted  string
	ContactID      string
	ContactName    string
	DocumentID     string
	DocumentTitle  string
	IssueID        string
	IssueTitle     string
	EvidenceNote   string
	Active         bool
}

type energyMeasureView struct {
	ID               string
	IssueID          string
	Title            string
	Status           string
	StatusLabel      string
	ContactID        string
	ContactName      string
	OfferNote        string
	AppointmentValue string
	Appointment      string
	WorkNote         string
	EvidenceNote     string
	BeforeFrom       string
	BeforeTo         string
	AfterFrom        string
	AfterTo          string
	BeforePeak       string
	AfterPeak        string
	BeforeQuality    string
	AfterQuality     string
	HasComparison    bool
	Completed        string
}

func (a *app) energyContactOptions(contacts contactBookRepository) ([]energyOption, map[string]string) {
	options := []energyOption{}
	names := map[string]string{}
	if contacts == nil {
		return options, names
	}
	for _, item := range contacts.List(false) {
		label := managedContactDisplayName(item)
		if item.Company != "" && !strings.EqualFold(item.Company, label) {
			label += " · " + item.Company
		}
		if item.ServiceRegion != "" {
			label += " · " + item.ServiceRegion
		}
		names[item.ID] = label
		options = append(options, energyOption{Value: item.ID, Label: label})
	}
	sort.Slice(options, func(i, j int) bool { return strings.ToLower(options[i].Label) < strings.ToLower(options[j].Label) })
	return options, names
}

func (a *app) energyContact(contacts contactBookRepository, id string) (managedContact, bool) {
	if contacts == nil || strings.TrimSpace(id) == "" {
		return managedContact{}, false
	}
	for _, item := range contacts.List(false) {
		if item.ID == id {
			return item, true
		}
	}
	return managedContact{}, false
}

func (a *app) energyDocumentOptions(tenant store.TenantRef) ([]energyOption, map[string]string) {
	options := []energyOption{}
	names := map[string]string{}
	documents, ok := store.BindDocumentRepository(a.documentStore, tenant)
	if !ok {
		return options, names
	}
	for _, item := range documents.ListCurrent() {
		names[item.ID] = item.Title
		options = append(options, energyOption{Value: item.ID, Label: item.Title})
	}
	sort.Slice(options, func(i, j int) bool { return strings.ToLower(options[i].Label) < strings.ToLower(options[j].Label) })
	return options, names
}

func (a *app) energyIssueOptions(tenant store.TenantRef) ([]energyOption, map[string]string) {
	options := []energyOption{}
	names := map[string]string{}
	issues, ok := store.BindIssueRepository(a.issueStore, tenant)
	if !ok {
		return options, names
	}
	for _, item := range issues.List() {
		names[item.ID] = item.Title
		options = append(options, energyOption{Value: item.ID, Label: item.Title})
	}
	sort.Slice(options, func(i, j int) bool { return strings.ToLower(options[i].Label) < strings.ToLower(options[j].Label) })
	return options, names
}

func buildEnergyMaintenanceViews(plans []energy.MaintenancePlan, assets []energy.Asset, contacts, documents, issues map[string]string, now time.Time) []energyMaintenanceView {
	assetNames := map[string]string{}
	for _, asset := range assets {
		assetNames[asset.ID] = asset.Name
	}
	out := make([]energyMaintenanceView, 0, len(plans))
	for _, plan := range plans {
		days := int(plan.NextDueAt.Sub(now).Hours() / 24)
		dueLabel := "Fällig am " + plan.NextDueAt.In(time.Local).Format("02.01.2006")
		tone := ""
		if days < 0 {
			dueLabel = "Überfällig seit " + plan.NextDueAt.In(time.Local).Format("02.01.2006")
			tone = "danger"
		} else if days <= 30 {
			tone = "warning"
		}
		lastCompleted := ""
		if plan.LastCompletedAt != nil {
			lastCompleted = plan.LastCompletedAt.In(time.Local).Format("02.01.2006")
		}
		out = append(out, energyMaintenanceView{
			ID: plan.ID, AssetID: plan.AssetID, AssetName: firstNonEmpty(assetNames[plan.AssetID], "Anlage"),
			Title: plan.Title, IntervalMonths: plan.IntervalMonths, NextDueValue: plan.NextDueAt.In(time.Local).Format("2006-01-02"),
			DueLabel: dueLabel, Tone: tone, LastCompleted: lastCompleted, ContactID: plan.ContactID,
			ContactName: contacts[plan.ContactID], DocumentID: plan.DocumentID, DocumentTitle: documents[plan.DocumentID],
			IssueID: plan.IssueID, IssueTitle: issues[plan.IssueID], EvidenceNote: plan.EvidenceNote, Active: plan.Active,
		})
	}
	return out
}

func buildEnergyMeasureViews(items []energy.Measure, contacts map[string]string) []energyMeasureView {
	out := make([]energyMeasureView, 0, len(items))
	for _, item := range items {
		view := energyMeasureView{
			ID: item.ID, IssueID: item.IssueID, Title: item.Title, Status: item.Status,
			StatusLabel: energyMeasureStatusLabel(item.Status), ContactID: item.ContactID, ContactName: contacts[item.ContactID],
			OfferNote: item.OfferNote, WorkNote: item.WorkNote, EvidenceNote: item.EvidenceNote,
			BeforeQuality: energyQualityLabel(item.BeforeQuality), AfterQuality: energyQualityLabel(item.AfterQuality),
		}
		if item.AppointmentAt != nil {
			view.AppointmentValue = item.AppointmentAt.In(time.Local).Format("2006-01-02T15:04")
			view.Appointment = item.AppointmentAt.In(time.Local).Format("02.01.2006 15:04")
		}
		if item.CompletedAt != nil {
			view.Completed = item.CompletedAt.In(time.Local).Format("02.01.2006")
		}
		view.BeforeFrom = energyDateValue(item.BeforeFrom)
		view.BeforeTo = energyDateValue(item.BeforeTo)
		view.AfterFrom = energyDateValue(item.AfterFrom)
		view.AfterTo = energyDateValue(item.AfterTo)
		if item.BeforePeakKW != nil && item.AfterPeakKW != nil {
			view.BeforePeak = formatEnergyValueUnit(formatEnergyNumber(*item.BeforePeakKW), "kW")
			view.AfterPeak = formatEnergyValueUnit(formatEnergyNumber(*item.AfterPeakKW), "kW")
			view.HasComparison = true
		}
		out = append(out, view)
	}
	return out
}

func energyMeasureStatusLabel(status string) string {
	switch status {
	case energy.MeasureRequested:
		return "Anfrage vorbereitet"
	case energy.MeasureAssigned:
		return "Kontakt ausgewählt"
	case energy.MeasureScheduled:
		return "Termin vereinbart"
	case energy.MeasureCompleted:
		return "Abgeschlossen"
	case energy.MeasureCancelled:
		return "Nicht weiterverfolgt"
	default:
		return "Entwurf"
	}
}

func energyAssetBelongsToTenant(storage energy.Storage, tenantSlug, assetID string) bool {
	items, err := storage.ListAssets(tenantSlug)
	if err != nil {
		return false
	}
	for _, item := range items {
		if item.ID == assetID {
			return true
		}
	}
	return false
}

func findMaintenancePlan(storage energy.Storage, tenantSlug, id string) (energy.MaintenancePlan, bool) {
	items, err := storage.ListMaintenance(tenantSlug)
	if err != nil {
		return energy.MaintenancePlan{}, false
	}
	for _, item := range items {
		if item.ID == id {
			return item, true
		}
	}
	return energy.MaintenancePlan{}, false
}

func findMaintenancePlanByAsset(storage energy.Storage, tenantSlug, assetID string) (energy.MaintenancePlan, bool) {
	items, err := storage.ListMaintenance(tenantSlug)
	if err != nil {
		return energy.MaintenancePlan{}, false
	}
	for _, item := range items {
		if item.AssetID == assetID {
			return item, true
		}
	}
	return energy.MaintenancePlan{}, false
}

func (a *app) validEnergyReferences(tenant store.TenantRef, contacts contactBookRepository, contactID, documentID, issueID string) bool {
	if contactID != "" {
		if _, ok := a.energyContact(contacts, contactID); !ok {
			return false
		}
	}
	if documentID != "" {
		documents, ok := store.BindDocumentRepository(a.documentStore, tenant)
		if !ok {
			return false
		}
		if _, ok := documents.Get(documentID); !ok {
			return false
		}
	}
	if issueID != "" {
		issues, ok := store.BindIssueRepository(a.issueStore, tenant)
		if !ok {
			return false
		}
		if _, ok := issues.Get(issueID); !ok {
			return false
		}
	}
	return true
}

func (a *app) completeEnergyMeasureRanges(item *energy.Measure, r *http.Request) error {
	beforeFrom, err := parseEnergyLocalDate(r.FormValue("before_from"))
	if err != nil {
		return err
	}
	beforeTo, err := parseEnergyLocalDate(r.FormValue("before_to"))
	if err != nil {
		return err
	}
	afterFrom, err := parseEnergyLocalDate(r.FormValue("after_from"))
	if err != nil {
		return err
	}
	afterTo, err := parseEnergyLocalDate(r.FormValue("after_to"))
	if err != nil {
		return err
	}
	if beforeTo.Before(beforeFrom) || afterTo.Before(afterFrom) || !beforeTo.Before(afterFrom) ||
		beforeTo.Sub(beforeFrom) > 180*24*time.Hour || afterTo.Sub(afterFrom) > 180*24*time.Hour {
		return fmt.Errorf("invalid comparison ranges")
	}
	beforeIntervals, err := a.energyStore.ListIntervals(item.TenantSlug, beforeFrom, beforeTo.AddDate(0, 0, 1))
	if err != nil {
		return err
	}
	afterIntervals, err := a.energyStore.ListIntervals(item.TenantSlug, afterFrom, afterTo.AddDate(0, 0, 1))
	if err != nil {
		return err
	}
	item.BeforeFrom, item.BeforeTo = &beforeFrom, &beforeTo
	item.AfterFrom, item.AfterTo = &afterFrom, &afterTo
	item.BeforePeakKW, item.BeforeQuality = energy.PeakForRange(beforeIntervals)
	item.AfterPeakKW, item.AfterQuality = energy.PeakForRange(afterIntervals)
	if item.BeforePeakKW == nil || item.AfterPeakKW == nil ||
		strings.TrimSpace(item.WorkNote) == "" || strings.TrimSpace(item.EvidenceNote) == "" {
		return fmt.Errorf("comparison evidence incomplete")
	}
	return nil
}

func (a *app) createEnergyMeasure(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.canViewEnergy(ac) || !canCreateResidentIssue(ac.actor(), ac.resource()) {
		http.Error(w, "Kein Zugriff", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ungültige Eingabe", http.StatusBadRequest)
		return
	}
	profile, exists, err := a.energyFor(ac).Profile(ac.tenant.Slug)
	if err != nil || !exists {
		http.Error(w, "Hausprofil fehlt.", http.StatusBadRequest)
		return
	}
	assets, _ := a.energyFor(ac).ListAssets(ac.tenant.Slug)
	mappings, _ := a.energyFor(ac).ListMappings(ac.tenant.Slug)
	intervals, _ := a.energyFor(ac).ListIntervals(ac.tenant.Slug, time.Now().AddDate(0, -1, 0), time.Time{})
	recommendation := energy.NextRecommendation(profile, assets, mappings, intervals)
	if maintenance, listErr := a.energyFor(ac).ListMaintenance(ac.tenant.Slug); listErr == nil {
		if due, ok := energy.MaintenanceRecommendation(time.Now(), maintenance); ok {
			recommendation = due
		}
	}
	if strings.TrimSpace(r.FormValue("recommendation_id")) != recommendation.ID {
		http.Error(w, "Empfehlung ist nicht mehr aktuell.", http.StatusConflict)
		return
	}
	shared := []string{}
	for _, value := range r.Form["share"] {
		switch value {
		case "inventory", "measurements", "contact":
			shared = append(shared, value)
		}
	}
	pkg := energy.NewMeasurePackage(ac.tenant.Slug, recommendation)
	pkg.SharedFields = shared
	body := "Ziel: " + pkg.Goal + "\n\nAusgangslage: " + pkg.Baseline +
		"\n\nGewünschte Leistung: " + pkg.RequestedWork +
		"\n\nErwarteter Nachweis: " + pkg.ExpectedEvidence +
		"\n\nBewusst freigegebene Daten: " + energySharedFieldLabels(shared) +
		"\n\nOffene Vor-Ort-Fragen:\n- " + strings.Join(pkg.OpenSiteQuestions, "\n- ") +
		"\n\nKeine automatische Beauftragung, Preiszusage oder Vermittlungsprovision."
	now := time.Now().UTC()
	created, err := ac.repositories.issues.Create(residentIssue{
		TenantSlug:     ac.tenant.Slug,
		AuthorEmail:    normalizeEmail(ac.email),
		AuthorName:     a.profileForTenant(ac.email, ac.tenant.Slug).DisplayName(),
		Category:       "Sonstiges",
		Title:          "Energiemaßnahme: " + recommendation.Title,
		Body:           body,
		LocationType:   issueLocationCommon,
		LocationDetail: "Privates Hausprofil",
		Status:         issueStatusOpen,
		Priority:       issuePriorityNorm,
		CreatedAt:      now,
		UpdatedAt:      now,
	})
	if err != nil {
		http.Error(w, "Maßnahme konnte nicht angelegt werden.", http.StatusInternalServerError)
		return
	}
	profile.RecommendationID = recommendation.ID
	profile.RecommendationStatus = "measure-created"
	if err := a.energyFor(ac).SaveProfile(profile); err != nil {
		http.Error(w, "Maßnahme wurde angelegt, der Hausstatus konnte aber nicht gespeichert werden.", http.StatusInternalServerError)
		return
	}
	measure := energy.Measure{
		TenantSlug:       ac.tenant.Slug,
		IssueID:          created.ID,
		RecommendationID: recommendation.ID,
		Title:            "Energiemaßnahme: " + recommendation.Title,
		Status:           energy.MeasureRequested,
		SharedFields:     shared,
	}
	if err := a.energyFor(ac).UpsertMeasure(measure); err != nil {
		http.Error(w, "Hausaufgabe wurde angelegt, der Maßnahmenkontext konnte aber nicht gespeichert werden.", http.StatusInternalServerError)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: ac.tenant.Slug,
		ActorEmail: ac.email,
		ActorRole:  ac.role,
		Action:     "energy.measure.create",
		TargetType: "issue",
		TargetID:   created.ID,
		Summary:    "Energieempfehlung als Maßnahme übernommen",
		Details: map[string]string{
			"recommendation_id": recommendation.ID,
			"shared_fields":     strings.Join(shared, ","),
			"marketplace_gate":  pkg.MarketplaceGate,
		},
	})
	http.Redirect(w, r, "/app/energie?measure=created#naechster-schritt", http.StatusSeeOther)
}

func (a *app) upsertEnergyMaintenance(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.canManageEnergy(ac) {
		http.Error(w, "Kein Zugriff", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ungültige Eingabe", http.StatusBadRequest)
		return
	}
	assetID := strings.TrimSpace(r.FormValue("asset_id"))
	if !energyAssetBelongsToTenant(a.energyStore, ac.tenant.Slug, assetID) {
		http.Error(w, "Anlage gehört nicht zu dieser Liegenschaft.", http.StatusBadRequest)
		return
	}
	months, err := strconv.Atoi(strings.TrimSpace(r.FormValue("interval_months")))
	if err != nil {
		http.Redirect(w, r, "/app/energie?maintenance=invalid#wartung", http.StatusSeeOther)
		return
	}
	due, err := parseEnergyLocalDate(r.FormValue("next_due"))
	if err != nil {
		http.Redirect(w, r, "/app/energie?maintenance=invalid#wartung", http.StatusSeeOther)
		return
	}
	contactID := strings.TrimSpace(r.FormValue("contact_id"))
	documentID := strings.TrimSpace(r.FormValue("document_id"))
	issueID := strings.TrimSpace(r.FormValue("issue_id"))
	if !a.validEnergyReferences(ac.tenantRef, ac.repositories.contacts, contactID, documentID, issueID) {
		http.Error(w, "Verknüpfung gehört nicht zu dieser Liegenschaft.", http.StatusBadRequest)
		return
	}
	plan := energy.MaintenancePlan{
		ID:             strings.TrimSpace(r.FormValue("id")),
		TenantSlug:     ac.tenant.Slug,
		AssetID:        assetID,
		Title:          cleanEnergyText(r.FormValue("title"), 140),
		IntervalMonths: months,
		NextDueAt:      due,
		ContactID:      contactID,
		DocumentID:     documentID,
		IssueID:        issueID,
		EvidenceNote:   cleanEnergyText(r.FormValue("evidence_note"), 500),
		Active:         r.FormValue("active") != "false",
	}
	if plan.ID == "" {
		plan.ID = energy.NewID("maintenance")
	}
	if existing, ok := findMaintenancePlanByAsset(a.energyStore, ac.tenant.Slug, assetID); ok {
		plan.ID = existing.ID
		plan.CreatedAt = existing.CreatedAt
		plan.LastCompletedAt = existing.LastCompletedAt
	}
	if err := a.energyFor(ac).UpsertMaintenance(plan); err != nil {
		http.Redirect(w, r, "/app/energie?maintenance=invalid#wartung", http.StatusSeeOther)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: ac.tenant.Slug, ActorEmail: ac.email, ActorRole: ac.role,
		Action: "energy.maintenance.save", TargetType: "energy-maintenance", TargetID: plan.ID,
		Summary: "Wartungsplan gespeichert", Details: map[string]string{"asset_id": assetID, "interval_months": strconv.Itoa(months)},
	})
	http.Redirect(w, r, "/app/energie?maintenance=saved#wartung", http.StatusSeeOther)
}

func (a *app) completeEnergyMaintenance(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.canManageEnergy(ac) {
		http.Error(w, "Kein Zugriff", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ungültige Eingabe", http.StatusBadRequest)
		return
	}
	plan, ok := findMaintenancePlan(a.energyStore, ac.tenant.Slug, strings.TrimSpace(r.FormValue("id")))
	if !ok {
		http.Error(w, "Wartungsplan nicht gefunden.", http.StatusNotFound)
		return
	}
	completed := time.Now().UTC()
	if raw := strings.TrimSpace(r.FormValue("completed_at")); raw != "" {
		parsed, err := parseEnergyLocalDate(raw)
		if err != nil {
			http.Redirect(w, r, "/app/energie?maintenance=invalid#wartung", http.StatusSeeOther)
			return
		}
		completed = parsed
	}
	plan.LastCompletedAt = &completed
	plan.NextDueAt = completed.AddDate(0, plan.IntervalMonths, 0)
	plan.EvidenceNote = cleanEnergyText(r.FormValue("evidence_note"), 500)
	if issueID := strings.TrimSpace(r.FormValue("issue_id")); issueID != "" {
		if _, found := ac.repositories.issues.Get(issueID); !found {
			http.Error(w, "Nachweis-Aufgabe gehört nicht zu dieser Liegenschaft.", http.StatusBadRequest)
			return
		}
		plan.IssueID = issueID
	}
	if plan.EvidenceNote == "" && plan.IssueID == "" {
		http.Redirect(w, r, "/app/energie?maintenance=invalid#wartung", http.StatusSeeOther)
		return
	}
	if err := a.energyFor(ac).UpsertMaintenance(plan); err != nil {
		http.Error(w, "Wartung konnte nicht abgeschlossen werden.", http.StatusInternalServerError)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: ac.tenant.Slug, ActorEmail: ac.email, ActorRole: ac.role,
		Action: "energy.maintenance.complete", TargetType: "energy-maintenance", TargetID: plan.ID,
		Summary: "Wartung abgeschlossen", Details: map[string]string{"next_due": plan.NextDueAt.Format("2006-01-02")},
	})
	http.Redirect(w, r, "/app/energie?maintenance=completed#wartung", http.StatusSeeOther)
}

func (a *app) updateEnergyMeasure(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.canManageEnergy(ac) {
		http.Error(w, "Kein Zugriff", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ungültige Eingabe", http.StatusBadRequest)
		return
	}
	item, ok, err := a.energyFor(ac).GetMeasure(ac.tenant.Slug, strings.TrimSpace(r.FormValue("id")))
	if err != nil || !ok {
		http.Error(w, "Maßnahme nicht gefunden.", http.StatusNotFound)
		return
	}
	issue, found := ac.repositories.issues.Get(item.IssueID)
	if !found {
		http.Error(w, "Verknüpftes Anliegen nicht gefunden.", http.StatusConflict)
		return
	}
	status := strings.TrimSpace(r.FormValue("status"))
	switch status {
	case energy.MeasureRequested, energy.MeasureAssigned, energy.MeasureScheduled, energy.MeasureCompleted, energy.MeasureCancelled:
		item.Status = status
	default:
		http.Redirect(w, r, "/app/energie?measure_status=invalid#fachhilfe", http.StatusSeeOther)
		return
	}
	item.ContactID = strings.TrimSpace(r.FormValue("contact_id"))
	contact, contactOK := a.energyContact(ac.repositories.contacts, item.ContactID)
	if item.ContactID != "" && !contactOK {
		http.Error(w, "Fachkontakt gehört nicht zu dieser Liegenschaft.", http.StatusBadRequest)
		return
	}
	item.OfferNote = cleanEnergyText(r.FormValue("offer_note"), 1000)
	item.WorkNote = cleanEnergyText(r.FormValue("work_note"), 1000)
	item.EvidenceNote = cleanEnergyText(r.FormValue("evidence_note"), 1000)
	item.AppointmentAt = nil
	if raw := strings.TrimSpace(r.FormValue("appointment_at")); raw != "" {
		appointment, parseErr := parseEnergyLocalDateTime(raw)
		if parseErr != nil {
			http.Redirect(w, r, "/app/energie?measure_status=invalid#fachhilfe", http.StatusSeeOther)
			return
		}
		item.AppointmentAt = &appointment
	}
	if item.Status == energy.MeasureScheduled && item.AppointmentAt == nil {
		http.Redirect(w, r, "/app/energie?measure_status=appointment#fachhilfe", http.StatusSeeOther)
		return
	}
	if item.Status == energy.MeasureCompleted {
		if err := a.completeEnergyMeasureRanges(&item, r); err != nil {
			http.Redirect(w, r, "/app/energie?measure_status=ranges#fachhilfe", http.StatusSeeOther)
			return
		}
		completed := time.Now().UTC()
		item.CompletedAt = &completed
	}
	if item.ContactID != "" && item.Status == energy.MeasureRequested {
		item.Status = energy.MeasureAssigned
	}
	if err := a.energyFor(ac).UpsertMeasure(item); err != nil {
		http.Error(w, "Maßnahme konnte nicht gespeichert werden.", http.StatusInternalServerError)
		return
	}
	if contactOK && a.serviceAccessEnabled && normalizeEmail(contact.Email) != "" {
		updated, changed, updateErr := ac.repositories.issues.UpdateWorkflow(issue.ID, issueWorkflowUpdate{
			Status: issue.Status, Priority: issue.Priority, AssigneeEmail: normalizeEmail(contact.Email),
			ActorEmail: ac.email, ActorName: a.profileForTenant(ac.email, ac.tenant.Slug).DisplayName(), ChangedAt: time.Now(),
		})
		if updateErr != nil {
			http.Error(w, "Maßnahme ist gespeichert, aber der Dienstleisterzugriff konnte nicht aktualisiert werden.", http.StatusInternalServerError)
			return
		}
		if changed {
			a.handleIssueServiceAssignmentChange(r, ac.tenant, issue, updated, ac.email, ac.role)
		}
	}
	a.recordAudit(auditEvent{
		TenantSlug: ac.tenant.Slug, ActorEmail: ac.email, ActorRole: ac.role,
		Action: "energy.measure.update", TargetType: "energy-measure", TargetID: item.ID,
		Summary: "Energiemaßnahme aktualisiert", Details: map[string]string{"status": item.Status, "contact_id": item.ContactID},
	})
	http.Redirect(w, r, "/app/energie?measure_status=saved#fachhilfe", http.StatusSeeOther)
}

func energySharedFieldLabels(values []string) string {
	labels := map[string]string{
		"inventory":    "Anlageninventar",
		"measurements": "zusammengefasste Messwerte",
		"contact":      "Kontaktdaten",
	}
	out := []string{}
	for _, value := range values {
		if label := labels[value]; label != "" {
			out = append(out, label)
		}
	}
	if len(out) == 0 {
		return "keine"
	}
	return strings.Join(out, ", ")
}
