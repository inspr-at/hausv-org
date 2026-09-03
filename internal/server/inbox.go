package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/web"
)

func (a *app) inboxPage(w http.ResponseWriter, r *http.Request, ac authCtx) {
	a.renderInbox(w, r, ac, "", false)
}
func (a *app) inboxCasePage(w http.ResponseWriter, r *http.Request, ac authCtx) {
	a.renderInbox(w, r, ac, strings.TrimSpace(r.PathValue("id")), true)
}

func (a *app) renderInbox(w http.ResponseWriter, r *http.Request, ac authCtx, selectedID string, full bool) {
	orgKey, ok := a.inboxOrganisationKey(&ac)
	if !ok {
		http.Redirect(w, r, "/app/verwaltung?flash="+url.QueryEscape("Posteingang braucht eine Organisation"), http.StatusSeeOther)
		return
	}
	if a.intake == nil {
		http.Error(w, "Posteingang nicht verfügbar.", http.StatusServiceUnavailable)
		return
	}
	data, err := a.inboxData(r.Context(), orgKey, &ac, r.URL.Query(), selectedID, full)
	if err != nil {
		logError("inbox load failed", err, "organisation", orgKey)
		http.Error(w, "Posteingang nicht verfügbar.", http.StatusInternalServerError)
		return
	}
	var rendered bytes.Buffer
	component := web.InboxPage(a.verwaltungShell(&ac, "inbox"), data)
	if full {
		component = web.InboxCasePage(a.verwaltungShell(&ac, "inbox"), data)
	}
	if err := component.Render(r.Context(), &rendered); err != nil {
		logError("inbox render failed", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, prefixTenantHTMLPaths(rendered.String(), ac.tenant.Slug))
}

func (a *app) inboxData(ctx context.Context, orgKey string, ac *authCtx, query url.Values, selectedID string, full bool) (web.InboxData, error) {
	repo := a.intake(orgKey)
	filter := a.managedIntakeFilter(ac, []store.IntakeStatus{store.IntakeStatusOpen, store.IntakeStatusProposed, store.IntakeStatusRejected})
	filter.Limit = inboxListLimit(query)
	switch query.Get("source") {
	case "email":
		filter.Sources = []store.IntakeSource{store.IntakeSourceEmail}
	case "phone":
		filter.Sources = []store.IntakeSource{store.IntakeSourcePhone}
	case "portal":
		filter.Sources = []store.IntakeSource{store.IntakeSourcePortal}
	}
	if a.actorManagesTenant(ac, query.Get("house")) {
		filter.TenantSlug = query.Get("house")
	}
	filter.Assignee = strings.TrimSpace(query.Get("assignee"))
	switch query.Get("status") {
	case "open":
		filter.Statuses = []store.IntakeStatus{store.IntakeStatusOpen}
	case "proposed":
		filter.Statuses = []store.IntakeStatus{store.IntakeStatusProposed}
	case "rejected":
		filter.Statuses = []store.IntakeStatus{store.IntakeStatusRejected}
	}
	items, err := repo.List(ctx, filter)
	if err != nil {
		return web.InboxData{}, err
	}
	if query.Get("sort") == "age" {
		for left, right := 0, len(items)-1; left < right; left, right = left+1, right-1 {
			items[left], items[right] = items[right], items[left]
		}
	}
	now := time.Now()
	allAuto, err := repo.List(ctx, a.managedIntakeFilter(ac, []store.IntakeStatus{store.IntakeStatusAuto}))
	if err != nil {
		return web.InboxData{}, err
	}
	auto := make([]store.IntakeItem, 0, len(allAuto))
	for _, item := range allAuto {
		if intakeHandledToday(item, now) {
			auto = append(auto, item)
		}
	}
	countFilter := filter
	countFilter.Limit = 0
	countFilter.Offset = 0
	countFilter.Sort = ""
	countFilter.Statuses = []store.IntakeStatus{store.IntakeStatusOpen, store.IntakeStatusProposed, store.IntakeStatusRejected}
	openCount, err := repo.Count(ctx, countFilter)
	if err != nil {
		return web.InboxData{}, err
	}
	proposedFilter := countFilter
	proposedFilter.Statuses = []store.IntakeStatus{store.IntakeStatusProposed}
	proposedCount, err := repo.Count(ctx, proposedFilter)
	if err != nil {
		return web.InboxData{}, err
	}
	unassignedFilter := countFilter
	unassignedFilter.Unassigned = true
	unassignedCount, err := repo.Count(ctx, unassignedFilter)
	if err != nil {
		return web.InboxData{}, err
	}
	houses := a.inboxHouses(ac)
	houseNames := map[string]string{}
	for _, house := range houses {
		houseNames[house.Slug] = house.Name
	}
	if selectedID == "" && len(items) > 0 {
		selectedID = items[0].ID
	}
	data := web.InboxData{Eyebrow: strings.ToUpper(a.inboxOrganisationName(ac)) + " · POSTEINGANG", Lede: fmt.Sprintf("%d offen · %d mit Vorschlag · %d unzugeordnet · %d heute automatisch erledigt", openCount, proposedCount, unassignedCount, len(auto)), Houses: houses, Sort: query.Get("sort"), FullPage: full, Flash: query.Get("flash"), ProviderFootline: a.inboxProviderFootline()}
	for _, house := range houses {
		data.HouseFilter = append(data.HouseFilter, web.InboxOption{Value: house.Slug, Label: house.Name, Selected: filter.TenantSlug == house.Slug})
	}
	data.SourceFilter = []web.InboxOption{{Value: "email", Label: "E-Mail", Selected: query.Get("source") == "email"}, {Value: "phone", Label: "Telefon", Selected: query.Get("source") == "phone"}, {Value: "portal", Label: "Portal", Selected: query.Get("source") == "portal"}}
	data.StatusFilter = []web.InboxOption{{Value: "open", Label: "Ohne Vorschlag", Selected: query.Get("status") == "open"}, {Value: "proposed", Label: "Mit Vorschlag", Selected: query.Get("status") == "proposed"}, {Value: "rejected", Label: "Manuell", Selected: query.Get("status") == "rejected"}}
	for _, assignee := range a.organisationAssigneeHints(orgKey) {
		data.AssigneeFilter = append(data.AssigneeFilter, web.InboxOption{Value: assignee.Key, Label: assignee.Name, Selected: filter.Assignee == assignee.Key})
	}
	for _, item := range items {
		data.Items = append(data.Items, inboxQueueView(item, houseNames, selectedID, now))
	}
	for _, item := range auto {
		data.AutoItems = append(data.AutoItems, inboxQueueView(item, houseNames, selectedID, now))
	}
	if selectedID != "" {
		selected, getErr := repo.Get(ctx, selectedID)
		if getErr == nil && a.actorCanAccessIntake(ac, selected) {
			view, viewErr := a.inboxCaseView(ctx, orgKey, selected, houses, query.Get("edit") == "1")
			if viewErr != nil {
				return web.InboxData{}, viewErr
			}
			data.Selected = &view
		}
	}
	return data, nil
}

func intakeHandledToday(item store.IntakeItem, now time.Time) bool {
	handledAt := item.ReceivedAt
	if item.Handling != nil && !item.Handling.At.IsZero() {
		handledAt = item.Handling.At
	}
	return sameLocalDate(handledAt.In(time.Local), now.In(time.Local))
}

func inboxQueueView(item store.IntakeItem, names map[string]string, selectedID string, now time.Time) web.InboxItem {
	house := names[item.TenantSlug]
	if house == "" {
		house = "Nicht zugeordnet"
	}
	view := web.InboxItem{ID: item.ID, Source: inboxSourceLabel(item.Source), Subject: item.Subject, House: house, Unit: item.Unit, Age: relativeAge(now, item.ReceivedAt), Time: item.ReceivedAt.In(time.Local).Format("15:04"), Selected: item.ID == selectedID, Unassigned: item.TenantSlug == ""}
	if item.Suggestion != nil {
		view.Priority = store.NormalizeIssuePriority(item.Suggestion.Priority)
		view.Proposal = "Vorschlag: " + intakeCategoryShort(item.Suggestion.Category) + " · " + view.Priority
	}
	switch item.Status {
	case store.IntakeStatusRejected:
		view.Status = "Manuell"
		view.Proposal = "Manuell"
	case store.IntakeStatusApproved, store.IntakeStatusEdited:
		view.Status = "Freigegeben"
		view.Proposal = "Freigegeben"
	case store.IntakeStatusAuto:
		view.Status = "Automatisch"
		view.Proposal = "Automatisch"
	}
	return view
}

func (a *app) inboxCaseView(ctx context.Context, orgKey string, item store.IntakeItem, houses []web.InboxHouse, editing bool) (web.InboxCase, error) {
	provider := a.inboxProviderLabel()
	suggestion := item.Suggestion
	view := web.InboxCase{ID: item.ID, Subject: item.Subject, Meta: inboxSourceLabel(item.Source) + " · " + firstNonEmpty(item.TenantSlug, "nicht zugeordnet") + " · " + item.Unit, Sender: firstNonEmpty(item.FromName, item.FromEmail, item.FromPhone, "Unbekannt"), Received: item.ReceivedAt.In(time.Local).Format("02.01.2006 · 15:04"), Body: item.Body, Status: string(item.Status), HasSuggestion: suggestion != nil, CanSuggest: a.triage != nil, Editing: editing, ProviderLabel: provider, Created: "Eingegangen " + item.ReceivedAt.In(time.Local).Format("02.01. · 15:04")}
	category, priority, houseSlug, unit, assignee, templateKey, reply := "", store.IssuePriorityNorm, item.TenantSlug, item.Unit, "", "", ""
	if suggestion != nil {
		category = suggestion.Category
		priority = store.NormalizeIssuePriority(suggestion.Priority)
		houseSlug = firstNonEmpty(suggestion.TenantSlug, item.TenantSlug)
		unit = firstNonEmpty(suggestion.Unit, item.Unit)
		assignee = suggestion.Assignee
		templateKey = suggestion.TemplateKey
		reply = suggestion.Reply
		view.Actions = append([]string(nil), suggestion.Actions...)
		view.Confidence = int(suggestion.Confidence["overall"]*100 + 0.5)
		view.Model = suggestion.Model
		view.PromptHash = prefixString(suggestion.PromptHash, 8)
	}
	view.Category = category
	view.CategoryLabel = intakeCategoryLabel(category)
	view.Priority = priority
	view.HouseSlug = houseSlug
	view.Unit = unit
	view.Assignee = assignee
	view.TemplateKey = templateKey
	view.Reply = reply
	due := item.DueAt
	if due.IsZero() {
		due = intakeDueAt(time.Now(), priority)
	}
	view.Due = due.In(time.Local).Format("Mo, 02.01.2006")
	view.DueValue = due.In(time.Local).Format("2006-01-02")
	for _, c := range store.IntakeCategories() {
		view.Categories = append(view.Categories, web.InboxOption{Value: c.Key, Label: c.Label, Selected: c.Key == category})
	}
	for _, p := range []string{store.IssuePriorityLow, store.IssuePriorityNorm, store.IssuePriorityHigh, store.IssuePriorityUrgent} {
		view.Priorities = append(view.Priorities, web.InboxOption{Value: p, Label: p, Selected: p == priority})
	}
	for _, h := range houses {
		selected := h.Slug == houseSlug
		view.Houses = append(view.Houses, web.InboxOption{Value: h.Slug, Label: h.Name, Selected: selected})
		if selected {
			view.HouseLabel = h.Name
		}
	}
	if view.HouseLabel == "" {
		view.HouseLabel = "Nicht zugeordnet"
	}
	for _, hint := range a.organisationAssigneeHints(orgKey) {
		selected := hint.Key == assignee
		view.Assignees = append(view.Assignees, web.InboxOption{Value: hint.Key, Label: hint.Name, Selected: selected})
		if selected {
			view.AssigneeLabel = hint.Name
		}
	}
	if view.AssigneeLabel == "" {
		view.AssigneeLabel = firstNonEmpty(assignee, "Nicht zugeordnet")
	}
	if a.textbausteine != nil {
		templates, err := a.textbausteine(orgKey).List(ctx)
		if err != nil {
			return view, err
		}
		for _, t := range templates {
			if !t.Active {
				continue
			}
			view.Templates = append(view.Templates, web.InboxOption{Value: t.Key, Label: t.Title, Selected: t.Key == templateKey})
			if t.Key == templateKey {
				view.TemplateTitle = t.Title
			}
		}
	}
	if item.Handling != nil {
		view.Handling = firstNonEmpty(item.Handling.ByName, item.Handling.ByEmail) + " · " + item.Handling.At.In(time.Local).Format("02.01. · 15:04")
	}
	return view, nil
}

func (a *app) inboxCaseAction(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	orgKey, ok := a.inboxOrganisationKey(&ac)
	if !ok || a.intake == nil {
		http.Error(w, "Posteingang nicht verfügbar.", http.StatusServiceUnavailable)
		return
	}
	repo := a.intake(orgKey)
	id := strings.TrimSpace(r.PathValue("id"))
	item, err := repo.Get(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if !a.actorCanAccessIntake(&ac, item) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	action := strings.TrimSpace(r.FormValue("action"))
	profile := a.profileForTenant(ac.email, ac.tenant.Slug)
	actorName := profile.DisplayName()
	switch action {
	case "approve", "edit":
		if item.Suggestion == nil {
			a.inboxRedirect(w, r, id, "Kein Vorschlag verfügbar")
			return
		}
		if action == "edit" {
			applySuggestionForm(&item, r)
			if !validIntakeCategory(item.Suggestion.Category) || store.NormalizeIssuePriority(item.Suggestion.Priority) == "" {
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}
			if !a.actorManagesTenant(&ac, firstNonEmpty(item.Suggestion.TenantSlug, item.TenantSlug)) {
				http.Error(w, "Dieses Haus ist nicht verfügbar.", http.StatusForbidden)
				return
			}
			if err := repo.UpdateSuggestion(r.Context(), id, *item.Suggestion, store.IntakeStatusProposed); err != nil {
				a.inboxError(w, err)
				return
			}
		}
		if !a.actorManagesTenant(&ac, firstNonEmpty(item.Suggestion.TenantSlug, item.TenantSlug)) {
			http.Error(w, "Dieses Haus ist nicht verfügbar.", http.StatusForbidden)
			return
		}
		status := store.IntakeStatusApproved
		auditAction := store.AuditActionIssueAIAccept
		counter := "approved"
		handling := "approved"
		if action == "edit" {
			status = store.IntakeStatusEdited
			auditAction = store.AuditActionIssueAIEdit
			counter = "edited"
			handling = "edited"
		}
		if err := a.handleIntake(r.Context(), orgKey, item, intakeHandleOptions{status: status, action: handling, actorEmail: ac.email, actorName: actorName, auditAction: auditAction, counter: counter}); err != nil {
			a.inboxError(w, err)
			return
		}
		house := a.tenants[firstNonEmpty(item.Suggestion.TenantSlug, item.TenantSlug)]
		next := a.nextOpenIntakeID(r.Context(), orgKey, &ac, id)
		location := "/app/verwaltung/posteingang"
		if next != "" {
			location += "/" + url.PathEscape(next)
		}
		location += "?flash=" + url.QueryEscape("Antwort gesendet, Anliegen angelegt · "+houseDisplayName(house))
		http.Redirect(w, r, location, http.StatusSeeOther)
	case "reject":
		if item.Suggestion == nil {
			a.inboxRedirect(w, r, id, "Kein Vorschlag verfügbar")
			return
		}
		if !a.actorManagesTenant(&ac, firstNonEmpty(item.Suggestion.TenantSlug, item.TenantSlug)) {
			http.Error(w, "Dieses Haus ist nicht verfügbar.", http.StatusForbidden)
			return
		}
		if err := a.handleIntake(r.Context(), orgKey, item, intakeHandleOptions{status: store.IntakeStatusRejected, action: "rejected", actorEmail: ac.email, actorName: actorName, auditAction: store.AuditActionIssueAIReject, counter: "rejected"}); err != nil {
			a.inboxError(w, err)
			return
		}
		a.inboxRedirect(w, r, id, "Manuell übernommen")
	case "assign":
		house := normalizeSlug(r.FormValue("house"))
		if !a.actorManagesTenant(&ac, house) {
			http.Error(w, "Dieses Haus ist nicht verfügbar.", http.StatusForbidden)
			return
		}
		if err := repo.Assign(r.Context(), id, house, strings.TrimSpace(r.FormValue("unit"))); err != nil {
			a.inboxError(w, err)
			return
		}
		item, err = repo.Get(r.Context(), id)
		if err != nil {
			a.inboxError(w, err)
			return
		}
		item.Suggestion = nil
		item.Status = store.IntakeStatusOpen
		if err := repo.Create(r.Context(), item); err != nil {
			a.inboxError(w, err)
			return
		}
		a.recordIntakeAudit(house, ac.email, store.AuditActionIntakeAssign, item, store.IntakeSuggestion{}, "Haus zugeordnet")
		if a.triage != nil {
			_, _ = a.processIntake(r.Context(), orgKey, item, ac.email)
		}
		a.inboxRedirect(w, r, id, "Haus zugeordnet")
	case "restore":
		if item.Status != store.IntakeStatusAuto || item.Suggestion == nil {
			http.Error(w, "Anliegen kann nicht wiederhergestellt werden.", http.StatusConflict)
			return
		}
		item.Status = store.IntakeStatusProposed
		item.Handling = nil
		item.UpdatedAt = time.Now().UTC()
		if err := repo.Create(r.Context(), item); err != nil {
			a.inboxError(w, err)
			return
		}
		settings, err := a.orgSettings(orgKey).Get(r.Context())
		if err == nil && settings.Counters.Auto > 0 {
			settings.Counters.Auto--
			_ = a.orgSettings(orgKey).Save(r.Context(), settings)
		}
		a.recordIntakeAudit(item.TenantSlug, ac.email, store.AuditActionIssueAIRestore, item, *item.Suggestion, "Automatische Bearbeitung zurückgenommen")
		a.inboxRedirect(w, r, id, "Zurück in den Eingang verschoben")
	case "suggest":
		if a.triage == nil {
			a.inboxRedirect(w, r, id, "Kein Vorschlag verfügbar")
			return
		}
		item.Suggestion = nil
		item.Status = store.IntakeStatusOpen
		if err := repo.Create(r.Context(), item); err != nil {
			a.inboxError(w, err)
			return
		}
		if _, err := a.processIntake(r.Context(), orgKey, item, ac.email); err != nil {
			a.inboxRedirect(w, r, id, "Kein Vorschlag verfügbar")
			return
		}
		a.inboxRedirect(w, r, id, "Vorschlag erstellt")
	case "template":
		if item.Suggestion == nil || a.textbausteine == nil {
			a.inboxRedirect(w, r, id, "Kein Vorschlag verfügbar")
			return
		}
		template, err := a.textbausteine(orgKey).Get(r.Context(), r.FormValue("template"))
		if err != nil {
			a.inboxRedirect(w, r, id, "Textbaustein nicht verfügbar")
			return
		}
		item.Suggestion.TemplateKey = template.Key
		item.Suggestion.Reply = store.RenderTextbaustein(template.Body, map[string]string{"Name": item.FromName, "Haus": item.Suggestion.TenantSlug, "Einheit": firstNonEmpty(item.Suggestion.Unit, item.Unit)})
		if err := repo.UpdateSuggestion(r.Context(), id, *item.Suggestion, item.Status); err != nil {
			a.inboxError(w, err)
			return
		}
		a.inboxRedirect(w, r, id, "Textbaustein eingesetzt")
	default:
		http.Error(w, "Bad request", http.StatusBadRequest)
	}
}

func applySuggestionForm(item *store.IntakeItem, r *http.Request) {
	s := item.Suggestion
	s.Category = strings.TrimSpace(r.FormValue("category"))
	s.Priority = store.NormalizeIssuePriority(r.FormValue("priority"))
	s.TenantSlug = normalizeSlug(r.FormValue("house"))
	s.Unit = strings.TrimSpace(r.FormValue("unit"))
	s.Assignee = strings.TrimSpace(r.FormValue("assignee"))
	s.TemplateKey = strings.TrimSpace(r.FormValue("template"))
	s.Reply = strings.TrimSpace(r.FormValue("reply"))
	if due, err := time.ParseInLocation("2006-01-02", r.FormValue("due"), time.Local); err == nil {
		item.DueAt = time.Date(due.Year(), due.Month(), due.Day(), 17, 0, 0, 0, time.Local).UTC()
	}
}

func (a *app) phoneNoteAction(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	orgKey, ok := a.inboxOrganisationKey(&ac)
	if !ok || a.intake == nil {
		http.Error(w, "Posteingang nicht verfügbar.", http.StatusServiceUnavailable)
		return
	}
	house := normalizeSlug(r.FormValue("house"))
	if !a.actorManagesTenant(&ac, house) {
		http.Error(w, "Dieses Haus ist nicht verfügbar.", http.StatusForbidden)
		return
	}
	subject := strings.TrimSpace(r.FormValue("subject"))
	body := strings.TrimSpace(r.FormValue("body"))
	name := strings.TrimSpace(r.FormValue("from_name"))
	if subject == "" || body == "" || name == "" {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	id, err := randomToken(12)
	if err != nil {
		a.inboxError(w, err)
		return
	}
	now := time.Now().UTC()
	item := store.IntakeItem{ID: id, Organisation: orgKey, TenantSlug: house, Unit: strings.TrimSpace(r.FormValue("unit")), Source: store.IntakeSourcePhone, ExternalRef: normalizeEmail(ac.email) + "|" + now.Format(time.RFC3339Nano), FromName: name, FromPhone: strings.TrimSpace(r.FormValue("from_phone")), Subject: subject, Body: body, ReceivedAt: now, Status: store.IntakeStatusOpen, CreatedAt: now, UpdatedAt: now}
	if err := a.intake(orgKey).Create(r.Context(), item); err != nil {
		a.inboxError(w, err)
		return
	}
	a.recordIntakeAudit(house, ac.email, store.AuditActionIntakePhoneNote, item, store.IntakeSuggestion{}, "Telefonnotiz aufgenommen")
	if _, err := a.processIntake(r.Context(), orgKey, item, ac.email); err != nil {
		logError("phone note triage unavailable", err, "intake_id", id)
	}
	a.inboxRedirect(w, r, id, "Telefonnotiz aufgenommen")
}

func (a *app) inboxOpenCount(ac *authCtx) int {
	orgKey, ok := a.inboxOrganisationKey(ac)
	if !ok || a.intake == nil {
		return 0
	}
	count, err := a.intake(orgKey).Count(context.Background(), a.managedIntakeFilter(ac, []store.IntakeStatus{store.IntakeStatusOpen, store.IntakeStatusProposed, store.IntakeStatusRejected}))
	if err != nil {
		return 0
	}
	return count
}

func (a *app) managedIntakeFilter(ac *authCtx, statuses []store.IntakeStatus) store.IntakeFilter {
	filter := store.IntakeFilter{Statuses: statuses, IncludeUnassigned: true}
	for _, tenant := range a.managedTenants(ac) {
		filter.TenantSlugs = append(filter.TenantSlugs, tenant.Ref.Slug)
	}
	return filter
}

func (a *app) inboxOrganisationKey(ac *authCtx) (string, bool) {
	if org, ok := a.organisationFor(ac); ok && normalizeSlug(org.Key) != "" {
		return normalizeSlug(org.Key), true
	}
	return "", false
}
func (a *app) inboxOrganisationName(ac *authCtx) string {
	if org, ok := a.organisationFor(ac); ok {
		return org.Name
	}
	return "Hausverwaltung"
}
func (a *app) inboxHouses(ac *authCtx) []web.InboxHouse {
	out := []web.InboxHouse{}
	for _, managed := range a.managedTenants(ac) {
		house := web.InboxHouse{Slug: managed.Config.Slug, Name: houseDisplayName(managed.Config), Address: managed.Config.Address}
		if units := a.repositoriesFor(managed.Ref).units; units != nil {
			for _, unit := range units.List() {
				house.Units = append(house.Units, unit.Label)
			}
		}
		out = append(out, house)
	}
	return out
}
func (a *app) actorManagesTenant(ac *authCtx, slug string) bool {
	slug = normalizeSlug(slug)
	if slug == "" {
		return false
	}
	for _, tenant := range a.managedTenants(ac) {
		if tenant.Ref.Slug == slug {
			return true
		}
	}
	return false
}
func (a *app) actorCanAccessIntake(ac *authCtx, item store.IntakeItem) bool {
	if item.TenantSlug == "" {
		return len(a.managedTenants(ac)) > 0
	}
	return a.actorManagesTenant(ac, item.TenantSlug)
}
func (a *app) inboxProviderLabel() string {
	if a != nil && a.triage != nil {
		return a.triage.Label()
	}
	return "nicht konfiguriert"
}
func (a *app) inboxProviderFootline() string {
	label := a.inboxProviderLabel()
	if strings.Contains(strings.ToLower(label), "openrouter") {
		return "KI: Cloud (OpenRouter) · Zielbetrieb lokal im Büro · jeder Vorschlag und jede Freigabe wird protokolliert"
	}
	return "KI läuft " + label + " · jeder Vorschlag und jede Freigabe wird protokolliert"
}
func (a *app) nextOpenIntakeID(ctx context.Context, orgKey string, ac *authCtx, current string) string {
	// The current action already proved access; use the same managed-house scope
	// for selecting the next queue item so navigation cannot cross house boundaries.
	items, err := a.intake(orgKey).List(ctx, a.managedIntakeFilter(ac, []store.IntakeStatus{store.IntakeStatusOpen, store.IntakeStatusProposed, store.IntakeStatusRejected}))
	if err != nil {
		return ""
	}
	for _, item := range items {
		if item.ID != current {
			return item.ID
		}
	}
	return ""
}
func (a *app) inboxRedirect(w http.ResponseWriter, r *http.Request, id, flash string) {
	location := "/app/verwaltung/posteingang"
	if id != "" {
		location += "/" + url.PathEscape(id)
	}
	if flash != "" {
		location += "?flash=" + url.QueryEscape(flash)
	}
	http.Redirect(w, r, location, http.StatusSeeOther)
}
func (a *app) inboxError(w http.ResponseWriter, err error) {
	logError("inbox action failed", err)
	http.Error(w, "Posteingang konnte nicht aktualisiert werden.", http.StatusInternalServerError)
}
func inboxSourceLabel(source store.IntakeSource) string {
	switch source {
	case store.IntakeSourceEmail:
		return "E-MAIL"
	case store.IntakeSourcePhone:
		return "TELEFON"
	default:
		return "PORTAL"
	}
}
func intakeCategoryShort(key string) string {
	label := intakeCategoryLabel(key)
	if cut := strings.IndexAny(label, "/"); cut > 0 {
		return label[:cut]
	}
	return label
}
func validIntakeCategory(key string) bool {
	for _, category := range store.IntakeCategories() {
		if category.Key == key {
			return true
		}
	}
	return false
}
func relativeAge(now, at time.Time) string {
	d := now.Sub(at)
	if d < time.Minute {
		return "jetzt"
	}
	if d < time.Hour {
		return fmt.Sprintf("vor %d Min.", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("vor %d Std.", int(d.Hours()))
	}
	return fmt.Sprintf("vor %d T.", int(d.Hours()/24))
}
func prefixString(value string, n int) string {
	if len(value) <= n {
		return value
	}
	return value[:n]
}
func settingsDiff(before, after store.OrgSettings) string {
	parts := []string{}
	if before.AutoThreshold != after.AutoThreshold {
		parts = append(parts, fmt.Sprintf("Schwellwert %d%% → %d%%", int(before.AutoThreshold*100), int(after.AutoThreshold*100)))
	}
	if before.AutoEnabled != after.AutoEnabled {
		parts = append(parts, "Automatik "+strconv.FormatBool(after.AutoEnabled))
	}
	for _, category := range store.IntakeCategories() {
		if before.TrustLevels[category.Key] != after.TrustLevels[category.Key] {
			parts = append(parts, category.Label+": "+before.TrustLevels[category.Key]+" → "+after.TrustLevels[category.Key])
		}
	}
	sort.Strings(parts)
	if len(parts) == 0 {
		return "keine Änderung"
	}
	return strings.Join(parts, "; ")
}

// inboxListLimit keeps the queue readable: the newest 80 items by default,
// ?limit=<n> up to 1000 for a full backlog view.
func inboxListLimit(query url.Values) int {
	limit := 80
	if raw := strings.TrimSpace(query.Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 1000 {
		limit = 1000
	}
	return limit
}
