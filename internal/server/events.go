package server

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/version"
	"github.com/inspr-at/hausv-org/internal/web"
)

func (a *app) events(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	if denyServiceProviderArea(w, role) {
		return
	}
	canManage := canManageEvents(ac.actor(), ac.resource())
	now := time.Now()
	events := ac.repositories.events
	upcoming := []houseEvent{}
	past := []houseEvent{}
	if events != nil {
		upcoming = events.Upcoming(now)
		all := events.List()
		for i := len(all) - 1; i >= 0; i-- {
			if !eventRollsOffAt(all[i]).After(now) {
				past = append(past, all[i])
			}
		}
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	status := r.URL.Query().Get("status")
	if status != "current" && status != "past" {
		status = "all"
	}
	upcoming = filterEvents(upcoming, query)
	past = filterEvents(past, query)
	currentCount, pastCount := len(upcoming), len(past)
	if status == "past" {
		upcoming = nil
	}
	if status == "current" {
		past = nil
	}
	calendarFeedURL := ""
	if token, err := a.calendarFeedToken(email, tenant.Slug); err == nil {
		calendarFeedURL = a.publicBaseURL(r, tenant) + "/calendar/" + url.PathEscape(token) + ".ics"
	}
	msg, msgOK := eventMessage(r.URL.Query().Get("event"))
	upcomingViews := a.eventViews(ac.tenantRef, upcoming, now, email, role)
	if len(upcomingViews) > 0 {
		upcomingViews[0].IsNext = true
	}
	pastViews := a.eventViews(ac.tenantRef, past, now, email, role)
	monthGroups := eventMonthGroups(upcoming, upcomingViews, time.Local)
	months := make([]web.EventsMonth, 0, len(monthGroups))
	for _, month := range monthGroups {
		months = append(months, web.EventsMonth{Label: month.Label, Events: month.Events})
	}
	a.renderEventsTempl(w, r, web.EventsPageData{
		Portal:          a.eventsPortalContext(ac),
		AssetVersion:    version.AssetVersion(),
		NowInput:        formatLocalDateTimeInput(now),
		CalendarFeedURL: calendarFeedURL,
		Message:         msg,
		MessageOK:       msgOK,
		CanManageEvents: canManage,
		CanManageIssues: ac.can(capabilityManageIssues),
		Upcoming:        upcomingViews,
		Months:          months,
		Past:            pastViews,
		SearchQuery:     query,
		SelectedStatus:  status,
		CurrentCount:    currentCount,
		PastCount:       pastCount,
	})
}

func (a *app) eventsPortalContext(ac authCtx) web.PortalPageData {
	return a.portalBaseData(ac, "events", "Termine")
}

func (a *app) renderEventsTempl(w http.ResponseWriter, r *http.Request, data web.EventsPageData) {
	var rendered bytes.Buffer
	if err := web.EventsPage(data).Render(r.Context(), &rendered); err != nil {
		logError("templ events render failed", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, prefixTenantHTMLPaths(rendered.String(), data.Portal.TenantSlug))
}

func (a *app) createEvent(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email := ac.tenant, ac.email
	if !canManageEvents(ac.actor(), ac.resource()) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if err := parseMaybeMultipartForm(w, r, maxIssueAttachmentFormBytes, maxAttachmentBytes); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	profile := a.profileForTenant(email, tenant.Slug)
	item, err := eventFromForm(r, tenant.Slug, profile)
	if err != nil {
		http.Redirect(w, r, "/app/events?event=invalid", http.StatusSeeOther)
		return
	}
	attachmentHeaders, err := attachmentFormHeaders(r, maxIssueAttachmentCount, "attachments")
	if err != nil {
		http.Redirect(w, r, "/app/events?event=invalid", http.StatusSeeOther)
		return
	}
	events := ac.repositories.events
	created, err := events.Create(item)
	if err != nil {
		logError("event create failed", err, "tenant", tenant.Slug)
		http.Redirect(w, r, "/app/events?event=error", http.StatusSeeOther)
		return
	}
	if len(attachmentHeaders) > 0 {
		if ac.repositories.attachments == nil {
			_, _ = events.Delete(created.ID)
			http.Redirect(w, r, "/app/events?event=invalid", http.StatusSeeOther)
			return
		}
		if _, err := ac.repositories.attachments.CreateUploaded("event", created.ID, email, uploadedFilesFromHeaders(attachmentHeaders), time.Now()); err != nil {
			_, _ = events.Delete(created.ID)
			http.Redirect(w, r, "/app/events?event=invalid", http.StatusSeeOther)
			return
		}
	}
	a.recordEventAudit(ac, auditActionEventCreate, created.ID, created.Category, len(attachmentHeaders))
	http.Redirect(w, r, "/app/events?event=created#event-"+url.PathEscape(created.ID), http.StatusSeeOther)
}

func (a *app) editEvent(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email := ac.tenant, ac.email
	if !canManageEvents(ac.actor(), ac.resource()) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if err := parseMaybeMultipartForm(w, r, maxIssueAttachmentFormBytes, maxAttachmentBytes); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	id := strings.TrimSpace(r.FormValue("id"))
	profile := a.profileForTenant(email, tenant.Slug)
	item, err := eventFromForm(r, tenant.Slug, profile)
	if err != nil {
		http.Redirect(w, r, "/app/events?event=invalid", http.StatusSeeOther)
		return
	}
	attachmentHeaders, err := attachmentFormHeaders(r, maxIssueAttachmentCount, "attachments")
	if err != nil {
		http.Redirect(w, r, "/app/events?event=invalid", http.StatusSeeOther)
		return
	}
	var uploaded []attachmentRecord
	if len(attachmentHeaders) > 0 {
		if a.attachmentStore == nil {
			http.Redirect(w, r, "/app/events?event=invalid", http.StatusSeeOther)
			return
		}
		uploaded, err = ac.repositories.attachments.CreateUploaded("event", id, email, uploadedFilesFromHeaders(attachmentHeaders), time.Now())
		if err != nil {
			http.Redirect(w, r, "/app/events?event=invalid", http.StatusSeeOther)
			return
		}
	}
	ok, err := ac.repositories.events.Update(id, item)
	if err != nil {
		for _, attachment := range uploaded {
			_, _, _ = ac.repositories.attachments.Delete(attachment.ID, time.Now())
		}
		logError("event update failed", err, "tenant", tenant.Slug)
		http.Redirect(w, r, "/app/events?event=error", http.StatusSeeOther)
		return
	}
	if !ok {
		for _, attachment := range uploaded {
			_, _, _ = ac.repositories.attachments.Delete(attachment.ID, time.Now())
		}
		http.Redirect(w, r, "/app/events?event=missing", http.StatusSeeOther)
		return
	}
	a.recordEventAudit(ac, auditActionEventUpdate, id, item.Category, len(uploaded))
	http.Redirect(w, r, "/app/events?event=updated#event-"+url.PathEscape(id), http.StatusSeeOther)
}

func (a *app) deleteEvent(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant := ac.tenant
	if !canManageEvents(ac.actor(), ac.resource()) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	id := strings.TrimSpace(r.FormValue("id"))
	removed, err := ac.repositories.events.Delete(id)
	if err != nil {
		logError("event delete failed", err, "tenant", tenant.Slug)
		http.Redirect(w, r, "/app/events?event=error", http.StatusSeeOther)
		return
	}
	if !removed {
		http.Redirect(w, r, "/app/events?event=missing", http.StatusSeeOther)
		return
	}
	a.recordEventAudit(ac, auditActionEventDelete, id, "", 0)
	http.Redirect(w, r, "/app/events?event=deleted", http.StatusSeeOther)
}

// recordEventAudit deliberately keeps free-text calendar content out of the
// audit trail. Titles, descriptions, locations and attachment names remain in
// their access-controlled stores; the audit log only records the mutation,
// target, enumerated category and attachment count.
func (a *app) recordEventAudit(ac authCtx, action string, targetID string, category string, fileCount int) {
	summary := map[string]string{
		auditActionEventCreate: "Kalendertermin angelegt",
		auditActionEventUpdate: "Kalendertermin geändert",
		auditActionEventDelete: "Kalendertermin gelöscht",
	}[action]
	details := map[string]string{}
	if strings.TrimSpace(category) != "" {
		details["category"] = normalizeEventCategory(category)
	}
	if fileCount > 0 {
		details["file_count"] = fmt.Sprintf("%d", fileCount)
	}
	a.recordAudit(auditEvent{
		TenantSlug: ac.tenant.Slug,
		ActorEmail: ac.email,
		ActorRole:  ac.role,
		Action:     action,
		TargetType: "event",
		TargetID:   targetID,
		Summary:    summary,
		Details:    details,
	})
}

func eventMessage(status string) (string, bool) {
	switch status {
	case "created":
		return "Termin gespeichert.", true
	case "updated":
		return "Termin aktualisiert.", true
	case "deleted":
		return "Termin gelöscht.", true
	case "invalid":
		return "Bitte Titel und Datum prüfen.", false
	case "missing":
		return "Dieser Termin wurde nicht gefunden.", false
	case "error":
		return "Der Termin konnte nicht gespeichert werden.", false
	default:
		return "", false
	}
}

func eventFromForm(r *http.Request, tenantSlug string, author userProfile) (houseEvent, error) {
	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		return houseEvent{}, fmt.Errorf("title is required")
	}
	if len([]rune(title)) > 140 {
		return houseEvent{}, fmt.Errorf("title too long")
	}
	body := strings.TrimSpace(r.FormValue("body"))
	if len([]rune(body)) > 3000 {
		return houseEvent{}, fmt.Errorf("body too long")
	}
	startsAt, err := parseRequiredLocalDateTime(r.FormValue("starts_at"))
	if err != nil {
		return houseEvent{}, err
	}
	endsAt, err := parseOptionalEventEnd(r.FormValue("ends_at"), startsAt)
	if err != nil {
		return houseEvent{}, err
	}
	item := houseEvent{
		TenantSlug:  normalizeSlug(tenantSlug),
		Title:       title,
		Body:        body,
		Category:    normalizeEventCategory(r.FormValue("category")),
		Location:    strings.TrimSpace(r.FormValue("location")),
		StartsAt:    startsAt.UTC(),
		EndsAt:      endsAt,
		AuthorEmail: normalizeEmail(author.Email),
		AuthorName:  author.DisplayName(),
	}
	if item.TenantSlug == "" {
		return houseEvent{}, fmt.Errorf("tenant is required")
	}
	return item, nil
}

// eventMonthGroup breaks the agenda at month boundaries. A calendar is read by
// time, so the list keeps its chronological order and only gains a heading
// whenever the month changes.
type eventMonthGroup struct {
	Label  string
	Count  int
	Events []houseEventView
}

func eventMonthGroups(items []houseEvent, views []houseEventView, loc *time.Location) []eventMonthGroup {
	if loc == nil {
		loc = time.Local
	}
	groups := []eventMonthGroup{}
	current := ""
	currentYear := ""
	for i := range views {
		if i >= len(items) {
			break
		}
		month := items[i].StartsAt.In(loc).Format("2006-01")
		if month != current {
			year := items[i].StartsAt.In(loc).Format("2006")
			label := formatMonthLabel(month, loc)
			if year == currentYear {
				label = strings.TrimSuffix(label, " "+year)
			}
			groups = append(groups, eventMonthGroup{Label: label})
			current = month
			currentYear = year
		}
		group := &groups[len(groups)-1]
		group.Events = append(group.Events, views[i])
		group.Count = len(group.Events)
	}
	return groups
}

func eventViews(items []houseEvent, now time.Time) []houseEventView {
	views := make([]houseEventView, 0, len(items))
	for _, item := range items {
		views = append(views, eventViewFrom(item, now))
	}
	return views
}

func (a *app) eventViews(tenant store.TenantRef, items []houseEvent, now time.Time, actorEmail string, role string) []houseEventView {
	tenantSlug := tenant.Slug
	views := eventViews(items, now)
	for i := range views {
		views[i].CanManage = canManageEvents(actorFor(actorEmail, tenantSlug, role), resourceFor(items[i].TenantSlug))
		if a == nil || a.attachmentStore == nil {
			continue
		}
		attachments := a.attachmentViewsForEntity(tenant, "event", views[i].ID, actorEmail, role)
		if len(attachments) == 0 {
			continue
		}
		views[i].Attachments = attachments
		views[i].HasAttachments = true
	}
	return views
}

// Search before applying the time filter so each segment shows its matching count.
func filterEvents(items []houseEvent, query string) []houseEvent {
	query = strings.ToLower(query)
	if query == "" {
		return items
	}
	filtered := make([]houseEvent, 0, len(items))
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Title+"\n"+item.Category+"\n"+item.Body+"\n"+item.Location), query) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}
