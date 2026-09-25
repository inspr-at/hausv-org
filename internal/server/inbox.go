package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/ai"
	"github.com/inspr-at/hausv-org/internal/store"
	viewutil "github.com/inspr-at/hausv-org/internal/view"
	"github.com/inspr-at/hausv-org/internal/web"
)

const defaultInboxSuggestTimeout = 45 * time.Second

type suggestJob struct {
	started   time.Time
	cancel    context.CancelFunc
	done      bool
	cancelled bool
	errClass  string
	finished  time.Time
}

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
	nonce, err := randomToken(24)
	if err != nil {
		http.Error(w, "Posteingang nicht verfügbar.", http.StatusInternalServerError)
		return
	}
	data.ScriptNonce = nonce
	var rendered bytes.Buffer
	component := web.InboxPage(a.verwaltungShell(r.Context(), &ac, "inbox"), data)
	if full {
		component = web.InboxCasePage(a.verwaltungShell(r.Context(), &ac, "inbox"), data)
	}
	if err := component.Render(r.Context(), &rendered); err != nil {
		logError("inbox render failed", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", inboxContentSecurityPolicy(nonce))
	_, _ = io.WriteString(w, prefixTenantHTMLPaths(rendered.String(), ac.tenant.Slug))
}

func inboxContentSecurityPolicy(nonce string) string {
	return "default-src 'self'; img-src 'self' blob:; style-src 'self' 'unsafe-inline'; script-src 'self' 'nonce-" + nonce + "'; font-src 'self'; connect-src 'self'; form-action 'self'; base-uri 'self'; frame-ancestors 'none'"
}

func (a *app) inboxSuggestionPartial(w http.ResponseWriter, r *http.Request, ac authCtx) {
	orgKey, ok := a.inboxOrganisationKey(&ac)
	if !ok || a.intake == nil {
		a.inboxSuggestionStatusError(w, r, ac, http.StatusServiceUnavailable)
		return
	}
	item, err := a.intake(orgKey).Get(r.Context(), strings.TrimSpace(r.PathValue("id")))
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, store.ErrIntakeNotFound) {
			status = http.StatusNotFound
		} else {
			logError("inbox suggestion load failed", err, "organisation", orgKey)
		}
		a.inboxSuggestionStatusError(w, r, ac, status)
		return
	}
	if !a.actorCanAccessIntake(&ac, item) {
		a.inboxSuggestionStatusError(w, r, ac, http.StatusForbidden)
		return
	}
	view, err := a.inboxCaseView(r.Context(), orgKey, item, a.inboxHouses(&ac), false)
	if err != nil {
		logError("inbox suggestion view failed", err, "organisation", orgKey)
		a.inboxSuggestionStatusError(w, r, ac, http.StatusInternalServerError)
		return
	}
	view.QueueQuery = inboxQueueQuery(r.URL.Query())
	var rendered bytes.Buffer
	if err := web.InboxSuggestionPartial(view).Render(r.Context(), &rendered); err != nil {
		logError("inbox suggestion render failed", err)
		a.inboxSuggestionStatusError(w, r, ac, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, prefixTenantHTMLPaths(rendered.String(), ac.tenant.Slug))
}

// A failed status read must not redirect or replace the case workflow. The
// client preserves its existing controls; direct partial requests get the same
// recoverable state without exposing repository errors or another tenant's data.
func (a *app) inboxSuggestionStatusError(w http.ResponseWriter, r *http.Request, ac authCtx, status int) {
	view := web.InboxCase{
		ID:              strings.TrimSpace(r.PathValue("id")),
		QueueQuery:      inboxQueueQuery(r.URL.Query()),
		SuggestionState: "status_error",
	}
	var rendered bytes.Buffer
	if err := web.InboxSuggestionState(view).Render(context.WithoutCancel(r.Context()), &rendered); err != nil {
		logError("inbox suggestion error render failed", err)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, prefixTenantHTMLPaths(rendered.String(), ac.tenant.Slug))
}

func (a *app) inboxData(ctx context.Context, orgKey string, ac *authCtx, query url.Values, selectedID string, full bool) (web.InboxData, error) {
	repo := a.intake(orgKey)
	filter := a.inboxListFilter(ac, query)
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
	houseRoles := map[string]string{}
	for _, house := range houses {
		houseNames[house.Slug] = house.Name
		houseRoles[house.Slug] = house.Role
	}
	if selectedID == "" && len(items) > 0 {
		selectedID = items[0].ID
	}
	queueQuery := inboxQueueQuery(query)
	lede := fmt.Sprintf("%d offen · %d mit Vorschlag · %d nicht zugeordnet · %d heute automatisch erledigt", openCount, proposedCount, unassignedCount, len(auto))
	if suggester, _ := a.triageFor(ctx, orgKey); suggester == nil {
		lede += ". Vorschläge und Sicherheitswerte sind Beispielvorschläge."
	}
	data := web.InboxData{
		Eyebrow:          strings.ToUpper(a.inboxOrganisationName(ac)) + " · POSTEINGANG",
		Lede:             lede,
		Houses:           houses,
		Sort:             query.Get("sort"),
		FullPage:         full,
		Flash:            query.Get("flash"),
		ProviderFootline: a.inboxProviderFootlineFor(ctx, orgKey),
		SessionSlug:      ac.tenant.Slug,
		QueueQuery:       queueQuery,
		OpenCount:        openCount,
		UnassignedCount:  unassignedCount,
		ProposedCount:    proposedCount,
		AllFilterURL:     inboxFilterURL(query, "", ""),
		UnassignedURL:    inboxFilterURL(query, "unassigned", "1"),
		OpenFilterURL:    inboxFilterURL(query, "status", "open"),
		ProposedURL:      inboxFilterURL(query, "status", "proposed"),
	}
	for _, house := range houses {
		data.HouseFilter = append(data.HouseFilter, web.InboxOption{Value: house.Slug, Label: house.Name, Selected: filter.TenantSlug == house.Slug})
	}
	data.SourceFilter = []web.InboxOption{{Value: "email", Label: "E-Mail", Selected: query.Get("source") == "email"}, {Value: "phone", Label: "Telefon", Selected: query.Get("source") == "phone"}, {Value: "portal", Label: "Portal", Selected: query.Get("source") == "portal"}}
	data.StatusFilter = []web.InboxOption{{Value: "open", Label: "Ohne Vorschlag", Selected: query.Get("status") == "open"}, {Value: "proposed", Label: "Mit Vorschlag", Selected: query.Get("status") == "proposed"}, {Value: "rejected", Label: "Manuell", Selected: query.Get("status") == "rejected"}}
	for _, assignee := range a.organisationAssigneeHints(orgKey) {
		data.AssigneeFilter = append(data.AssigneeFilter, web.InboxOption{Value: assignee.Key, Label: assignee.Name, Selected: filter.Assignee == assignee.Key})
	}
	for _, item := range items {
		data.Items = append(data.Items, inboxQueueView(item, houseNames, houseRoles, ac.tenant.Slug, selectedID, now))
	}
	for _, item := range auto {
		data.AutoItems = append(data.AutoItems, inboxQueueView(item, houseNames, houseRoles, ac.tenant.Slug, selectedID, now))
	}
	if selectedID != "" {
		selected, getErr := repo.Get(ctx, selectedID)
		if getErr == nil && a.actorCanAccessIntake(ac, selected) {
			view, viewErr := a.inboxCaseView(ctx, orgKey, selected, houses, query.Get("edit") == "1")
			if viewErr != nil {
				return web.InboxData{}, viewErr
			}
			view.QueueQuery = queueQuery
			data.Selected = &view
		}
	}
	for index, item := range items {
		if item.ID != selectedID || data.Selected == nil {
			continue
		}
		data.Selected.Position = index + 1
		data.Selected.Total = openCount
		if index > 0 {
			data.Selected.PrevURL = "/app/verwaltung/posteingang/" + url.PathEscape(items[index-1].ID) + queueQuery
		}
		if index+1 < len(items) {
			data.Selected.NextURL = "/app/verwaltung/posteingang/" + url.PathEscape(items[index+1].ID) + queueQuery
		}
		break
	}
	return data, nil
}

func (a *app) inboxListFilter(ac *authCtx, query url.Values) store.IntakeFilter {
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
	filter.Unassigned = query.Get("unassigned") == "1"
	switch query.Get("status") {
	case "open":
		filter.Statuses = []store.IntakeStatus{store.IntakeStatusOpen}
	case "proposed":
		filter.Statuses = []store.IntakeStatus{store.IntakeStatusProposed}
	case "rejected":
		filter.Statuses = []store.IntakeStatus{store.IntakeStatusRejected}
	}
	return filter
}

func inboxQueueQuery(query url.Values) string {
	clean := url.Values{}
	for _, key := range []string{"source", "house", "assignee", "status", "sort", "unassigned", "limit"} {
		if value := strings.TrimSpace(query.Get(key)); value != "" {
			clean.Set(key, value)
		}
	}
	if encoded := clean.Encode(); encoded != "" {
		return "?" + encoded
	}
	return ""
}

func inboxFilterURL(query url.Values, key, value string) string {
	clean, _ := url.ParseQuery(strings.TrimPrefix(inboxQueueQuery(query), "?"))
	clean.Del("status")
	clean.Del("unassigned")
	if key != "" && value != "" {
		clean.Set(key, value)
	}
	if encoded := clean.Encode(); encoded != "" {
		return "/app/verwaltung/posteingang?" + encoded
	}
	return "/app/verwaltung/posteingang"
}

func intakeHandledToday(item store.IntakeItem, now time.Time) bool {
	handledAt := item.ReceivedAt
	if item.Handling != nil && !item.Handling.At.IsZero() {
		handledAt = item.Handling.At
	}
	return sameLocalDate(handledAt.In(time.Local), now.In(time.Local))
}

func inboxQueueView(item store.IntakeItem, names, roles map[string]string, sessionSlug, selectedID string, now time.Time) web.InboxItem {
	house := names[item.TenantSlug]
	if house == "" {
		house = "Nicht zugeordnet"
	}
	statusLabel, statusTone := inboxStatus(item)
	slug := normalizeSlug(item.TenantSlug)
	nextPath, foreign := OrganisationHousePath(sessionSlug, slug, "/app/verwaltung/posteingang/"+item.ID)
	if slug == "" || nextPath == "/app" {
		foreign = false
	}
	view := web.InboxItem{ID: item.ID, Source: inboxSourceLabel(item.Source), Subject: item.Subject, House: house, TenantSlug: slug, Role: roles[slug], Foreign: foreign, Unit: item.Unit, Age: relativeAge(now, item.ReceivedAt), Time: item.ReceivedAt.In(time.Local).Format("15:04"), Status: statusLabel, StatusTone: statusTone, Selected: item.ID == selectedID, Unassigned: item.TenantSlug == ""}
	if item.Suggestion != nil {
		view.Priority = store.NormalizeIssuePriority(item.Suggestion.Priority)
		view.Proposal = "Vorschlag: " + viewutil.BreakAfterSlashes(intakeCategoryShort(item.Suggestion.Category)) + " · " + view.Priority
	}
	return view
}

func inboxStatus(item store.IntakeItem) (label, tone string) {
	if item.TenantSlug == "" {
		return "Nicht zugeordnet", "neutral"
	}
	switch item.Status {
	case store.IntakeStatusProposed:
		return "Vorschlag liegt vor", "ok"
	case store.IntakeStatusApproved, store.IntakeStatusEdited:
		return "Freigegeben", "info"
	case store.IntakeStatusAuto:
		return "Automatisch erledigt", "neutral"
	default:
		return "Offen", "gold"
	}
}

func (a *app) inboxCaseView(ctx context.Context, orgKey string, item store.IntakeItem, houses []web.InboxHouse, editing bool) (web.InboxCase, error) {
	suggester, provider := a.triageFor(ctx, orgKey)
	if provider == "" {
		provider = "nicht konfiguriert"
	}
	suggestion := item.Suggestion
	if suggestion != nil && suggestion.Provider != "" {
		provider = suggestion.Provider
	}
	statusLabel, statusTone := inboxStatus(item)
	view := web.InboxCase{ID: item.ID, Subject: item.Subject, Sender: firstNonEmpty(item.FromName, item.FromEmail, item.FromPhone, "Unbekannt"), Received: item.ReceivedAt.In(time.Local).Format("02.01.2006 · 15:04"), Body: item.Body, Status: statusLabel, StatusTone: statusTone, HasSuggestion: suggestion != nil, CanSuggest: suggester != nil, Unassigned: item.TenantSlug == "", Editing: editing, ProviderLabel: provider, Created: "Eingegangen " + item.ReceivedAt.In(time.Local).Format("02.01. · 15:04")}
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
		view.UnfilledLabels = intakeUnfilledLabels(suggestion.Unfilled)
	}
	// An e-mail case names its channel and address and lists what came with
	// it; the files themselves move to the Anliegen on approval.
	if item.Source == store.IntakeSourceEmail {
		view.SourceLabel = "E-Mail"
		view.SenderEmail = item.FromEmail
	}
	view.DroppedFiles = item.DroppedFiles
	for _, attachment := range item.Attachments {
		size := fmt.Sprintf("%d KB", (attachment.Size+1023)/1024)
		if attachment.Size >= 1<<20 {
			size = fmt.Sprintf("%.1f MB", float64(attachment.Size)/float64(1<<20))
		}
		view.Attachments = append(view.Attachments, web.InboxAttachment{Filename: attachment.Filename, Size: size})
	}
	view.Category = category
	view.CategoryLabel = viewutil.BreakAfterSlashes(intakeCategoryLabel(category))
	view.Priority = priority
	view.HouseSlug = houseSlug
	view.Unit = unit
	view.Occupancy = occupancyLabel(a.unitOccupancy(houseSlug, unit))
	view.Assignee = assignee
	view.TemplateKey = templateKey
	view.Reply = reply
	due := item.DueAt
	if due.IsZero() {
		due = intakeDueAt(time.Now(), priority)
	}
	view.Due = viewutil.GermanDateShort(due.In(time.Local))
	view.DueValue = due.In(time.Local).Format("2006-01-02")
	for _, c := range store.IntakeCategories() {
		view.Categories = append(view.Categories, web.InboxOption{Value: c.Key, Label: viewutil.BreakAfterSlashes(c.Label), Selected: c.Key == category})
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
	metaParts := []string{inboxSourceLabel(item.Source), view.HouseLabel}
	if item.Unit != "" {
		metaParts = append(metaParts, item.Unit)
	}
	metaParts = append(metaParts, relativeAge(time.Now(), item.ReceivedAt))
	view.Meta = strings.Join(metaParts, " · ")
	for _, hint := range a.organisationAssigneeHints(orgKey) {
		selected := hint.Key == assignee
		view.Assignees = append(view.Assignees, web.InboxOption{Value: hint.Key, Label: hint.Name, Selected: selected})
		if selected {
			view.AssigneeLabel = hint.Name
		}
	}
	if view.AssigneeLabel == "" {
		view.AssigneeLabel = "Noch niemand"
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
	a.applySuggestJobView(&view)
	return view, nil
}

func (a *app) applySuggestJobView(view *web.InboxCase) {
	if a == nil || view == nil {
		return
	}
	a.suggestJobsMu.Lock()
	job := a.suggestJobs[view.ID]
	if job != nil {
		started, done, cancelled, errClass, finished := job.started, job.done, job.cancelled, job.errClass, job.finished
		a.suggestJobsMu.Unlock()
		view.SuggestionElapsed = int(time.Since(started).Seconds())
		if view.SuggestionElapsed < 0 {
			view.SuggestionElapsed = 0
		}
		view.SuggestionTimeout = int(a.suggestionTimeout().Seconds())
		switch {
		case !done:
			view.SuggestionState = "running"
		case cancelled:
			view.SuggestionState = "cancelled"
		case errClass != "":
			view.SuggestionState = "failed"
			view.SuggestionError = errClass
		default:
			view.SuggestionState = "arrived"
		}
		if !finished.IsZero() {
			view.SuggestionFinished = finished.In(time.Local).Format("15:04")
		}
		return
	}
	a.suggestJobsMu.Unlock()
}

func (a *app) suggestionTimeout() time.Duration {
	if a == nil || a.inboxSuggestTimeout <= 0 {
		return defaultInboxSuggestTimeout
	}
	return a.inboxSuggestTimeout
}

func (a *app) startSuggestJob(requestCtx context.Context, orgKey string, item store.IntakeItem, actor string) (bool, error) {
	a.suggestJobsMu.Lock()
	if a.suggestJobs == nil {
		a.suggestJobs = map[string]*suggestJob{}
	}
	if existing := a.suggestJobs[item.ID]; existing != nil && !existing.done {
		a.suggestJobsMu.Unlock()
		return false, nil
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(requestCtx), a.suggestionTimeout())
	job := &suggestJob{started: time.Now(), cancel: cancel}
	a.suggestJobs[item.ID] = job
	a.suggestJobsMu.Unlock()

	item.Suggestion = nil
	item.Status = store.IntakeStatusOpen
	item.UpdatedAt = time.Now().UTC()
	if err := a.intake(orgKey).Create(requestCtx, item); err != nil {
		cancel()
		a.suggestJobsMu.Lock()
		if a.suggestJobs[item.ID] == job {
			delete(a.suggestJobs, item.ID)
		}
		a.suggestJobsMu.Unlock()
		return false, err
	}

	go a.runSuggestJob(ctx, orgKey, item, actor, job)
	return true, nil
}

func (a *app) runSuggestJob(ctx context.Context, orgKey string, item store.IntakeItem, actor string, job *suggestJob) {
	defer job.cancel()
	result, err := a.processIntake(ctx, orgKey, item, actor)
	errClass := inboxSuggestionError(err)
	if err == nil && result.Suggestion == nil {
		errClass = "Antwort unbrauchbar"
	}
	finished := time.Now()
	a.suggestJobsMu.Lock()
	defer a.suggestJobsMu.Unlock()
	if current := a.suggestJobs[item.ID]; current != job || job.cancelled {
		return
	}
	job.done = true
	job.errClass = errClass
	job.finished = finished
}

func inboxSuggestionError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "Zeitüberschreitung"
	}
	lower := strings.ToLower(err.Error())
	if errors.Is(err, ai.ErrUncertain) || strings.Contains(lower, "decode") || strings.Contains(lower, "no choices") || strings.Contains(lower, "choice was empty") || strings.Contains(lower, "too large") || strings.Contains(lower, "unknown category") || strings.Contains(lower, "unknown priority") || strings.Contains(lower, "unknown house") || strings.Contains(lower, "unknown assignee") || strings.Contains(lower, "unknown template") || strings.Contains(lower, "unbrauchbar") {
		return "Antwort unbrauchbar"
	}
	return "Anbieter nicht erreichbar"
}

func (a *app) cancelSuggestJob(item store.IntakeItem) bool {
	a.suggestJobsMu.Lock()
	job := a.suggestJobs[item.ID]
	if job == nil || job.done {
		a.suggestJobsMu.Unlock()
		return false
	}
	job.cancelled = true
	job.done = true
	job.finished = time.Now()
	job.cancel()
	a.suggestJobsMu.Unlock()
	return true
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
	queueQuery, _ := url.ParseQuery(strings.TrimPrefix(r.FormValue("queue_query"), "?"))
	profile := a.profileForTenant(ac.email, ac.tenant.Slug)
	actorName := profile.DisplayName()
	switch action {
	case "approve", "approve_next", "edit":
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
				http.Error(w, "Diese Liegenschaft ist nicht verfügbar.", http.StatusForbidden)
				return
			}
			if err := repo.UpdateSuggestion(r.Context(), id, *item.Suggestion, store.IntakeStatusProposed); err != nil {
				a.inboxError(w, err)
				return
			}
		}
		next := ""
		if action == "approve_next" {
			next = a.nextOpenIntakeID(r.Context(), orgKey, &ac, id, queueQuery)
		}
		if !a.actorManagesTenant(&ac, firstNonEmpty(item.Suggestion.TenantSlug, item.TenantSlug)) {
			http.Error(w, "Diese Liegenschaft ist nicht verfügbar.", http.StatusForbidden)
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
		flash := "Antwort gesendet, Anliegen angelegt · " + houseDisplayName(house)
		if action == "approve_next" {
			if next == "" {
				a.inboxRedirectWithQuery(w, r, "", queueQuery, "Alle offenen Anliegen erledigt")
				return
			}
			a.inboxRedirectWithQuery(w, r, next, queueQuery, flash)
			return
		}
		a.inboxRedirectWithQuery(w, r, id, queueQuery, flash)
	case "reject":
		if item.Suggestion == nil {
			a.inboxRedirect(w, r, id, "Kein Vorschlag verfügbar")
			return
		}
		if !a.actorManagesTenant(&ac, firstNonEmpty(item.Suggestion.TenantSlug, item.TenantSlug)) {
			http.Error(w, "Diese Liegenschaft ist nicht verfügbar.", http.StatusForbidden)
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
			http.Error(w, "Diese Liegenschaft ist nicht verfügbar.", http.StatusForbidden)
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
		a.recordIntakeAudit(house, ac.email, store.AuditActionIntakeAssign, item, store.IntakeSuggestion{}, "Liegenschaft zugeordnet")
		a.inboxRedirectWithQuery(w, r, id, queueQuery, "Liegenschaft zugeordnet")
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
		if !a.hasTriage(r.Context(), orgKey) {
			a.inboxRedirect(w, r, id, "Kein Vorschlag verfügbar")
			return
		}
		if _, err := a.startSuggestJob(r.Context(), orgKey, item, ac.email); err != nil {
			a.inboxError(w, err)
			return
		}
		a.inboxRedirectWithQuery(w, r, id, queueQuery, "")
	case "suggest_cancel":
		if a.cancelSuggestJob(item) {
			item.Suggestion = nil
			item.Status = store.IntakeStatusOpen
			item.UpdatedAt = time.Now().UTC()
			if err := repo.Create(r.Context(), item); err != nil {
				a.inboxError(w, err)
				return
			}
			a.recordIntakeAudit(item.TenantSlug, ac.email, store.AuditActionIssueAICancel, item, store.IntakeSuggestion{}, "KI-Anfrage abgebrochen")
		}
		a.inboxRedirectWithQuery(w, r, id, queueQuery, "")
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
		item.Suggestion.Reply, item.Suggestion.Unfilled = store.FillReply(template.Body, a.intakeReplyValues(orgKey, item, *item.Suggestion))
		item.Suggestion.Reply = store.TidyReply(item.Suggestion.Reply)
		if len(item.Suggestion.Unfilled) > 0 && item.Suggestion.Confidence["overall"] > 0.5 {
			item.Suggestion.Confidence["overall"] = 0.5
		}
		if err := repo.UpdateSuggestion(r.Context(), id, *item.Suggestion, item.Status); err != nil {
			a.inboxError(w, err)
			return
		}
		a.inboxRedirect(w, r, id, "Textbaustein eingesetzt")
	default:
		http.Error(w, "Bad request", http.StatusBadRequest)
	}
}

func intakeUnfilledLabels(keys []string) []string {
	labels := map[string]string{
		"Name": "Name", "Haus": "Liegenschaft", "Einheit": "Einheit", "Nummer": "Nummer",
		"Zuständig": "Zuständig", "Handwerker": "Handwerker", "Frist": "Frist",
	}
	items := make([]string, 0, len(keys))
	for _, key := range keys {
		if label := labels[key]; label != "" {
			items = append(items, label)
		}
	}
	return items
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
		http.Error(w, "Diese Liegenschaft ist nicht verfügbar.", http.StatusForbidden)
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
	return a.organisationAccessFor(ac).intakeFilter(statuses)
}

func (a *app) inboxOrganisationKey(ac *authCtx) (string, bool) {
	access := a.organisationAccessFor(ac)
	if !access.allowed() || access.Key == "" {
		return "", false
	}
	return access.Key, true
}
func (a *app) inboxOrganisationName(ac *authCtx) string {
	if org, ok := a.organisationFor(ac); ok {
		return org.Name
	}
	return "Hausverwaltung"
}
func (a *app) inboxHouses(ac *authCtx) []web.InboxHouse {
	out := []web.InboxHouse{}
	for _, managed := range a.organisationAccessFor(ac).Houses {
		house := web.InboxHouse{Slug: managed.Config.Slug, Name: houseDisplayName(managed.Config), Address: managed.Config.Address, Role: managed.Role}
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
	return a.organisationAccessFor(ac).managesHouse(slug)
}
func (a *app) actorCanAccessIntake(ac *authCtx, item store.IntakeItem) bool {
	access := a.organisationAccessFor(ac)
	if !access.allowed() {
		return false
	}
	if item.Organisation != "" && access.Key != "" && normalizeSlug(item.Organisation) != access.Key {
		return false
	}
	if normalizeSlug(item.TenantSlug) == "" {
		return access.SeesUnassigned
	}
	return access.managesHouse(item.TenantSlug)
}
func (a *app) inboxProviderLabel() string {
	return a.inboxProviderLabelFor(context.Background(), "")
}

func (a *app) inboxProviderLabelFor(ctx context.Context, orgKey string) string {
	_, label := a.triageFor(ctx, orgKey)
	if label != "" {
		return label
	}
	return "nicht konfiguriert"
}

func (a *app) inboxProviderFootline() string {
	return a.inboxProviderFootlineFor(context.Background(), "")
}

func (a *app) inboxProviderFootlineFor(ctx context.Context, orgKey string) string {
	if suggester, _ := a.triageFor(ctx, orgKey); suggester == nil {
		return "Kein KI-Anbieter konfiguriert · jeder Vorschlag und jede Freigabe wird protokolliert"
	}
	label := a.inboxProviderLabelFor(ctx, orgKey)
	if strings.Contains(strings.ToLower(label), "openrouter") {
		return "KI: Cloud (OpenRouter) · Zielbetrieb lokal im Büro · jeder Vorschlag und jede Freigabe wird protokolliert"
	}
	return "KI läuft " + label + " · jeder Vorschlag und jede Freigabe wird protokolliert"
}
func (a *app) nextOpenIntakeID(ctx context.Context, orgKey string, ac *authCtx, current string, query url.Values) string {
	// The current action already proved access; use the same managed-house scope
	// and visible filter/sort for navigation so it cannot cross queue boundaries.
	filter := a.inboxListFilter(ac, query)
	items, err := a.intake(orgKey).List(ctx, filter)
	if err != nil {
		return ""
	}
	if query.Get("sort") == "age" {
		for left, right := 0, len(items)-1; left < right; left, right = left+1, right-1 {
			items[left], items[right] = items[right], items[left]
		}
	}
	foundCurrent := false
	for _, item := range items {
		if item.ID == current {
			foundCurrent = true
			continue
		}
		if foundCurrent && item.TenantSlug != "" && (item.Status == store.IntakeStatusOpen || item.Status == store.IntakeStatusProposed) {
			return item.ID
		}
	}
	return ""
}
func (a *app) inboxRedirect(w http.ResponseWriter, r *http.Request, id, flash string) {
	a.inboxRedirectWithQuery(w, r, id, nil, flash)
}
func (a *app) inboxRedirectWithQuery(w http.ResponseWriter, r *http.Request, id string, query url.Values, flash string) {
	location := "/app/verwaltung/posteingang"
	if id != "" {
		location += "/" + url.PathEscape(id)
	}
	clean, _ := url.ParseQuery(strings.TrimPrefix(inboxQueueQuery(query), "?"))
	if flash != "" {
		clean.Set("flash", flash)
	}
	if encoded := clean.Encode(); encoded != "" {
		location += "?" + encoded
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
