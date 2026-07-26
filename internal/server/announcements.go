package server

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (a *app) announcements(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	if denyServiceProviderArea(w, role) {
		return
	}
	canManage := canManageAnnouncements(role)
	now := time.Now()
	selectedCategory := selectedAnnouncementCategory(r.URL.Query().Get("category"))
	searchQuery := strings.TrimSpace(r.URL.Query().Get("q"))
	lastSeen := time.Time{}
	if a.announcementReadStore != nil {
		lastSeen = a.announcementReadStore.LastSeen(tenant.Slug, email)
	}
	archive := []announcement{}
	filtered := []announcement{}
	all := []announcementView{}
	if a.announcementStore != nil {
		archive = a.announcementStore.Archive(tenant.Slug, now)
		filtered = filterAnnouncements(archive, selectedCategory, searchQuery)
		if canManage {
			all = a.announcementViewsWithReadState(tenant.Slug, a.announcementStore.ListTenant(tenant.Slug), now, true, lastSeen, email, role)
		}
	}
	a.render(w, "announcements", a.withBase(ac, map[string]any{
		"Title":                  "Aushang",
		"CanManageAnnouncements": canManage,
		"ActivePage":             "announcements",
		"Announcements":          a.announcementViewsWithReadState(tenant.Slug, filtered, now, true, lastSeen, email, role),
		"HasAnnouncements":       len(filtered) > 0,
		"HasAnyAnnouncements":    len(archive) > 0,
		"AnnouncementsEmpty":     emptyState("Keine Beiträge", "Für diese Suche oder Kategorie gibt es keinen Aushang."),
		"AnnouncementsBlank":     emptyState("Noch keine Beiträge", "Sobald ein Aushang veröffentlicht ist, erscheint er hier."),
		"AllAnnouncements":       all,
		"HasAllAnnouncements":    len(all) > 0,
		"AllAnnouncementsEmpty":  emptyState("Noch kein Aushang gespeichert", "Neue Aushänge erscheinen hier nach dem Speichern."),
		"AnnounceMsg":            announcementMessage(r.URL.Query().Get("announce")),
		"NowInput":               formatLocalDateTimeInput(now),
		"SearchQuery":            searchQuery,
		"SelectedCategory":       selectedCategory,
		"CategoryFilters":        announcementFilterViews(searchQuery, selectedCategory),
		"UnreadAnnouncements":    0,
		"HasUnreadAnnouncements": false,
	}))
	if a.announcementReadStore != nil {
		if err := a.announcementReadStore.MarkSeen(tenant.Slug, email, now); err != nil {
			logError("announcement read mark failed", err, "tenant", tenant.Slug, "actor", redactedEmail(email))
		}
	}
}

func (a *app) createAnnouncement(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	if !canManageAnnouncements(role) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if err := parseMaybeMultipartForm(w, r, maxIssueAttachmentFormBytes, maxAttachmentBytes); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	profile := a.profileForTenant(email, tenant.Slug)
	item, err := announcementFromForm(r, tenant.Slug, profile, time.Now())
	if err != nil {
		http.Redirect(w, r, "/app/announcements?announce=invalid", http.StatusSeeOther)
		return
	}
	attachmentHeaders, err := attachmentFormHeaders(r, maxIssueAttachmentCount, "attachments")
	if err != nil {
		http.Redirect(w, r, "/app/announcements?announce=invalid", http.StatusSeeOther)
		return
	}
	created, err := a.announcementStore.Create(item)
	if err != nil {
		logError("announcement create failed", err, "tenant", tenant.Slug)
		http.Redirect(w, r, "/app/announcements?announce=error", http.StatusSeeOther)
		return
	}
	if len(attachmentHeaders) > 0 {
		if a.attachmentStore == nil {
			_, _ = a.announcementStore.Delete(tenant.Slug, created.ID)
			http.Redirect(w, r, "/app/announcements?announce=invalid", http.StatusSeeOther)
			return
		}
		if _, err := a.attachmentStore.CreateUploaded(tenant.Slug, "announcement", created.ID, email, uploadedFilesFromHeaders(attachmentHeaders), time.Now()); err != nil {
			_, _ = a.announcementStore.Delete(tenant.Slug, created.ID)
			http.Redirect(w, r, "/app/announcements?announce=invalid", http.StatusSeeOther)
			return
		}
	}
	a.notifyAnnouncementPublished(tenant, created, email)
	http.Redirect(w, r, "/app/announcements?announce=created", http.StatusSeeOther)
}

func (a *app) editAnnouncement(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	if !canManageAnnouncements(role) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if err := parseMaybeMultipartForm(w, r, maxIssueAttachmentFormBytes, maxAttachmentBytes); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	id := strings.TrimSpace(r.FormValue("id"))
	profile := a.profileForTenant(email, tenant.Slug)
	item, err := announcementFromForm(r, tenant.Slug, profile, time.Now())
	if err != nil {
		http.Redirect(w, r, "/app/announcements?announce=invalid", http.StatusSeeOther)
		return
	}
	attachmentHeaders, err := attachmentFormHeaders(r, maxIssueAttachmentCount, "attachments")
	if err != nil {
		http.Redirect(w, r, "/app/announcements?announce=invalid", http.StatusSeeOther)
		return
	}
	var uploaded []attachmentRecord
	if len(attachmentHeaders) > 0 {
		if a.attachmentStore == nil {
			http.Redirect(w, r, "/app/announcements?announce=invalid", http.StatusSeeOther)
			return
		}
		uploaded, err = a.attachmentStore.CreateUploaded(tenant.Slug, "announcement", id, email, uploadedFilesFromHeaders(attachmentHeaders), time.Now())
		if err != nil {
			http.Redirect(w, r, "/app/announcements?announce=invalid", http.StatusSeeOther)
			return
		}
	}
	ok, err := a.announcementStore.Update(id, item)
	if err != nil {
		for _, attachment := range uploaded {
			_, _, _ = a.attachmentStore.Delete(tenant.Slug, attachment.ID, time.Now())
		}
		logError("announcement update failed", err, "tenant", tenant.Slug)
		http.Redirect(w, r, "/app/announcements?announce=error", http.StatusSeeOther)
		return
	}
	if !ok {
		for _, attachment := range uploaded {
			_, _, _ = a.attachmentStore.Delete(tenant.Slug, attachment.ID, time.Now())
		}
		http.Redirect(w, r, "/app/announcements?announce=missing", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/app/announcements?announce=updated", http.StatusSeeOther)
}

func (a *app) deleteAnnouncement(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, role := ac.tenant, ac.role
	if !canManageAnnouncements(role) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if err := parseMaybeMultipartForm(w, r, maxIssueAttachmentFormBytes, maxAttachmentBytes); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	removed, err := a.announcementStore.Delete(tenant.Slug, strings.TrimSpace(r.FormValue("id")))
	if err != nil {
		logError("announcement delete failed", err, "tenant", tenant.Slug)
		http.Redirect(w, r, "/app/announcements?announce=error", http.StatusSeeOther)
		return
	}
	if !removed {
		http.Redirect(w, r, "/app/announcements?announce=missing", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/app/announcements?announce=deleted", http.StatusSeeOther)
}

func announcementMessage(status string) string {
	switch status {
	case "created":
		return "Aushang gespeichert."
	case "updated":
		return "Aushang aktualisiert."
	case "deleted":
		return "Aushang gelöscht."
	case "invalid":
		return "Bitte Titel, Text und Veröffentlichungsdatum prüfen."
	case "missing":
		return "Dieser Aushang wurde nicht gefunden."
	case "error":
		return "Der Aushang konnte nicht gespeichert werden."
	default:
		return ""
	}
}

func announcementFromForm(r *http.Request, tenantSlug string, author userProfile, now time.Time) (announcement, error) {
	title := strings.TrimSpace(r.FormValue("title"))
	body := strings.TrimSpace(r.FormValue("body"))
	if title == "" || body == "" {
		return announcement{}, fmt.Errorf("title and body are required")
	}
	if len([]rune(title)) > 140 {
		return announcement{}, fmt.Errorf("title too long")
	}
	if len([]rune(body)) > 5000 {
		return announcement{}, fmt.Errorf("body too long")
	}
	publishedAt, err := parseOptionalLocalDateTime(r.FormValue("published_at"), now)
	if err != nil {
		return announcement{}, err
	}
	expiresAt, err := parseOptionalExpiry(r.FormValue("expires_at"))
	if err != nil {
		return announcement{}, err
	}
	item := announcement{
		TenantSlug:  normalizeSlug(tenantSlug),
		Title:       title,
		Body:        body,
		Category:    normalizeAnnouncementCategory(r.FormValue("category")),
		Pinned:      parseBool(r.FormValue("pinned")),
		PublishedAt: publishedAt.UTC(),
		ExpiresAt:   expiresAt,
		AuthorEmail: normalizeEmail(author.Email),
		AuthorName:  author.DisplayName(),
	}
	if item.TenantSlug == "" {
		return announcement{}, fmt.Errorf("tenant is required")
	}
	return item, nil
}

func announcementFilterViews(query string, selectedCategory string) []announcementFilterView {
	selectedCategory = selectedAnnouncementCategory(selectedCategory)
	categories := []string{"", "Info", "Termin", "Wartung", "Dringend"}
	labels := map[string]string{"": "Alle"}
	out := make([]announcementFilterView, 0, len(categories))
	for _, category := range categories {
		label := labels[category]
		if label == "" {
			label = category
		}
		values := url.Values{}
		if strings.TrimSpace(query) != "" {
			values.Set("q", strings.TrimSpace(query))
		}
		if category != "" {
			values.Set("category", category)
		}
		filterURL := "/app/announcements"
		if encoded := values.Encode(); encoded != "" {
			filterURL += "?" + encoded
		}
		out = append(out, announcementFilterView{
			Label:  label,
			URL:    filterURL,
			Active: category == selectedCategory,
		})
	}
	return out
}

func announcementViews(items []announcement, now time.Time, includeStatus bool) []announcementView {
	return announcementViewsWithReadState(items, now, includeStatus, time.Time{})
}

func announcementViewsWithReadState(items []announcement, now time.Time, includeStatus bool, lastSeen time.Time) []announcementView {
	views := make([]announcementView, 0, len(items))
	for _, item := range items {
		views = append(views, announcementViewFrom(item, now, includeStatus, lastSeen))
	}
	return views
}

func (a *app) announcementViewsWithReadState(tenantSlug string, items []announcement, now time.Time, includeStatus bool, lastSeen time.Time, actorEmail string, role string) []announcementView {
	views := announcementViewsWithReadState(items, now, includeStatus, lastSeen)
	if a == nil || a.attachmentStore == nil {
		return views
	}
	for i := range views {
		attachments := a.attachmentViewsForEntity(tenantSlug, "announcement", views[i].ID, actorEmail, role)
		if len(attachments) == 0 {
			continue
		}
		views[i].Attachments = attachments
		views[i].HasAttachments = true
	}
	return views
}

func (a *app) enrichUnreadAnnouncementData(data map[string]any) {
	if _, ok := data["UnreadAnnouncements"]; ok {
		if _, hasFlag := data["HasUnreadAnnouncements"]; !hasFlag {
			if count, ok := data["UnreadAnnouncements"].(int); ok {
				data["HasUnreadAnnouncements"] = count > 0
			}
		}
		return
	}
	tenant, ok := data["Tenant"].(tenantConfig)
	if !ok || tenant.Slug == "" || a.announcementStore == nil || a.announcementReadStore == nil {
		data["UnreadAnnouncements"] = 0
		data["HasUnreadAnnouncements"] = false
		return
	}
	email, ok := data["Email"].(string)
	if !ok || strings.TrimSpace(email) == "" {
		data["UnreadAnnouncements"] = 0
		data["HasUnreadAnnouncements"] = false
		return
	}
	now := time.Now()
	lastSeen := a.announcementReadStore.LastSeen(tenant.Slug, email)
	count := unreadAnnouncementCount(a.announcementStore.Visible(tenant.Slug, now), lastSeen, now)
	data["UnreadAnnouncements"] = count
	data["HasUnreadAnnouncements"] = count > 0
}
