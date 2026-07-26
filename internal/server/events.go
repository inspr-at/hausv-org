package server

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (a *app) events(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	if denyServiceProviderArea(w, role) {
		return
	}
	profile := a.profileForTenant(email, tenant.Slug)
	isAdmin := hasCapability(role, capabilityPlatformAdmin)
	canManage := canManageEvents(role)
	now := time.Now()
	upcoming := []houseEvent{}
	all := []houseEvent{}
	if a.eventStore != nil {
		upcoming = a.eventStore.Upcoming(tenant.Slug, now)
		if canManage {
			all = a.eventStore.ListTenant(tenant.Slug)
		}
	}
	calendarFeedURL := ""
	if token, err := a.calendarFeedToken(email, tenant.Slug); err == nil {
		calendarFeedURL = a.publicBaseURL(r, tenant) + "/calendar/" + url.PathEscape(token) + ".ics"
	}
	a.render(w, "events", map[string]any{
		"Title":                  "Termine",
		"Tenant":                 tenant,
		"Email":                  email,
		"DisplayName":            profile.DisplayName(),
		"Initials":               profile.Initials(),
		"Role":                   role,
		"IsAdmin":                isAdmin,
		"CanSeeParking":          isAdmin || profile.HasPermission(permissionParking),
		"CanManageAnnouncements": canManageAnnouncements(role),
		"CanManageEvents":        canManage,
		"ActivePage":             "events",
		"Events":                 a.eventViews(tenant.Slug, upcoming, now, email, role),
		"HasEvents":              len(upcoming) > 0,
		"EventsEmpty":            emptyState("Noch keine kommenden Termine", "Geplante Versammlungen, Wartungen und Fristen erscheinen hier."),
		"CalendarFeedURL":        calendarFeedURL,
		"HasCalendarFeedURL":     calendarFeedURL != "",
		"AllEvents":              a.eventViews(tenant.Slug, all, now, email, role),
		"HasAllEvents":           len(all) > 0,
		"AllEventsEmpty":         emptyState("Noch kein Termin gespeichert", "Neue Termine erscheinen hier nach dem Speichern."),
		"EventMsg":               eventMessage(r.URL.Query().Get("event")),
		"NowInput":               formatLocalDateTimeInput(now),
	})
}

func (a *app) createEvent(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	if !canManageEvents(role) {
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
	created, err := a.eventStore.Create(item)
	if err != nil {
		logError("event create failed", err, "tenant", tenant.Slug)
		http.Redirect(w, r, "/app/events?event=error", http.StatusSeeOther)
		return
	}
	if len(attachmentHeaders) > 0 {
		if a.attachmentStore == nil {
			_, _ = a.eventStore.Delete(tenant.Slug, created.ID)
			http.Redirect(w, r, "/app/events?event=invalid", http.StatusSeeOther)
			return
		}
		if _, err := a.attachmentStore.CreateUploaded(tenant.Slug, "event", created.ID, email, uploadedFilesFromHeaders(attachmentHeaders), time.Now()); err != nil {
			_, _ = a.eventStore.Delete(tenant.Slug, created.ID)
			http.Redirect(w, r, "/app/events?event=invalid", http.StatusSeeOther)
			return
		}
	}
	http.Redirect(w, r, "/app/events?event=created", http.StatusSeeOther)
}

func (a *app) editEvent(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	if !canManageEvents(role) {
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
		uploaded, err = a.attachmentStore.CreateUploaded(tenant.Slug, "event", id, email, uploadedFilesFromHeaders(attachmentHeaders), time.Now())
		if err != nil {
			http.Redirect(w, r, "/app/events?event=invalid", http.StatusSeeOther)
			return
		}
	}
	ok, err := a.eventStore.Update(id, item)
	if err != nil {
		for _, attachment := range uploaded {
			_, _, _ = a.attachmentStore.Delete(tenant.Slug, attachment.ID, time.Now())
		}
		logError("event update failed", err, "tenant", tenant.Slug)
		http.Redirect(w, r, "/app/events?event=error", http.StatusSeeOther)
		return
	}
	if !ok {
		for _, attachment := range uploaded {
			_, _, _ = a.attachmentStore.Delete(tenant.Slug, attachment.ID, time.Now())
		}
		http.Redirect(w, r, "/app/events?event=missing", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/app/events?event=updated", http.StatusSeeOther)
}

func (a *app) deleteEvent(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, role := ac.tenant, ac.role
	if !canManageEvents(role) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	removed, err := a.eventStore.Delete(tenant.Slug, strings.TrimSpace(r.FormValue("id")))
	if err != nil {
		logError("event delete failed", err, "tenant", tenant.Slug)
		http.Redirect(w, r, "/app/events?event=error", http.StatusSeeOther)
		return
	}
	if !removed {
		http.Redirect(w, r, "/app/events?event=missing", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/app/events?event=deleted", http.StatusSeeOther)
}

func eventMessage(status string) string {
	switch status {
	case "created":
		return "Termin gespeichert."
	case "updated":
		return "Termin aktualisiert."
	case "deleted":
		return "Termin gelöscht."
	case "invalid":
		return "Bitte Titel und Datum prüfen."
	case "missing":
		return "Dieser Termin wurde nicht gefunden."
	case "error":
		return "Der Termin konnte nicht gespeichert werden."
	default:
		return ""
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

func eventViews(items []houseEvent, now time.Time) []houseEventView {
	views := make([]houseEventView, 0, len(items))
	for _, item := range items {
		views = append(views, eventViewFrom(item, now))
	}
	return views
}

func (a *app) eventViews(tenantSlug string, items []houseEvent, now time.Time, actorEmail string, role string) []houseEventView {
	views := eventViews(items, now)
	if a == nil || a.attachmentStore == nil {
		return views
	}
	for i := range views {
		attachments := a.attachmentViewsForEntity(tenantSlug, "event", views[i].ID, actorEmail, role)
		if len(attachments) == 0 {
			continue
		}
		views[i].Attachments = attachments
		views[i].HasAttachments = true
	}
	return views
}
