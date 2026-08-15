package server

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/version"
	"github.com/inspr-at/hausv-org/internal/view"
	"github.com/inspr-at/hausv-org/internal/web"
)

func (a *app) announcements(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	if denyServiceProviderArea(w, role) {
		return
	}
	canManage := canManageAnnouncements(ac.actor(), ac.resource())
	now := time.Now()
	selectedCategory := selectedAnnouncementCategory(r.URL.Query().Get("category"))
	searchQuery := strings.TrimSpace(r.URL.Query().Get("q"))
	lastSeen := time.Time{}
	announcements := ac.repositories.announcements
	announcementReads := ac.repositories.announcementReads
	if announcementReads != nil {
		lastSeen = announcementReads.LastSeen(email)
	}
	archive := []announcement{}
	filtered := []announcement{}
	if announcements != nil {
		archive = announcements.Archive(now)
		if canManage {
			archive = announcements.List()
		}
		filtered = filterAnnouncements(archive, selectedCategory, searchQuery)
	}
	views := a.announcementViewsWithReadState(tenant.Slug, filtered, now, true, lastSeen, email, role)
	pinned, latest := splitPinnedAnnouncements(views)
	newCount := unreadAnnouncementViewCount(views)
	pageData := map[string]any{
		"Title":                  "Aushang",
		"CanManageAnnouncements": canManage,
		"ActivePage":             "announcements",
		"Announcements":          views,
		"PinnedAnnouncements":    pinned,
		"HasPinnedAnnouncements": len(pinned) > 0,
		"LatestAnnouncements":    latest,
		"HasLatestAnnouncements": len(latest) > 0,
		"NewAnnouncements":       newCount,
		"HasNewAnnouncements":    newCount > 0,
		"HasAnnouncements":       len(filtered) > 0,
		"HasAnyAnnouncements":    len(archive) > 0,
		"AnnouncementsEmpty":     emptyState("Keine Beiträge", "Für diese Suche oder Kategorie gibt es keinen Aushang."),
		"AnnouncementsBlank":     emptyState("Noch keine Beiträge", "Sobald ein Aushang veröffentlicht ist, erscheint er hier."),
		"AnnounceMsg":            announcementMessage(r.URL.Query().Get("announce")),
		"NowInput":               formatLocalDateTimeInput(now),
		"SearchQuery":            searchQuery,
		"SelectedCategory":       selectedCategory,
		"CategoryFilters":        announcementFilterViews(searchQuery, selectedCategory),
		"UnreadAnnouncements":    0,
		"HasUnreadAnnouncements": false,
	}
	if a.portalTemplEnabled {
		a.renderAnnouncementsTempl(w, r, ac, web.AnnouncementsPageData{
			Portal:                 a.announcementPortalContext(ac),
			AssetVersion:           version.AssetVersion(),
			CanManageAnnouncements: canManage,
			CanManageIssues:        ac.can(capabilityManageIssues),
			Announcements:          views,
			PinnedAnnouncements:    pinned,
			LatestAnnouncements:    latest,
			NewAnnouncements:       newCount,
			AnnounceMessage:        announcementMessage(r.URL.Query().Get("announce")),
			NowInput:               formatLocalDateTimeInput(now),
			SearchQuery:            searchQuery,
			SelectedCategory:       selectedCategory,
			CategoryFilters:        announcementFilterViews(searchQuery, selectedCategory),
			AnnouncementsEmpty:     emptyState("Keine Beiträge", "Für diese Suche oder Kategorie gibt es keinen Aushang."),
			AnnouncementsBlank:     emptyState("Noch keine Beiträge", "Sobald ein Aushang veröffentlicht ist, erscheint er hier."),
			HasAnyAnnouncements:    len(archive) > 0,
		})
	} else {
		a.render(w, "announcements", a.withBase(ac, pageData))
	}
	if announcementReads != nil {
		if err := announcementReads.MarkSeen(email, now); err != nil {
			logError("announcement read mark failed", err, "tenant", tenant.Slug, "actor", redactedEmail(email))
		}
	}
}

func (a *app) announcementPortalContext(ac authCtx) web.PortalPageData {
	profile := a.profileForTenant(ac.email, ac.tenant.Slug)
	modules := a.portalModulesFor(ac.tenant.Slug)
	contexts := a.portalContextsFor(ac.email, ac.tenant.Slug, ac.role)
	portalContexts := make([]web.PortalContext, 0, len(contexts))
	for _, context := range contexts {
		portalContexts = append(portalContexts, web.PortalContext{
			TenantSlug: context.TenantSlug,
			HouseName:  context.HouseName,
			Address:    context.Address,
			Role:       context.Role,
			Current:    context.Current,
		})
	}
	openIssues := 0
	if a.issueStore != nil {
		openIssues = issueOpenCount(a.visibleIssuesForActor(ac.tenant.Slug, ac.email, ac.role))
	}
	return web.PortalPageData{
		Title:               "Aushang · " + houseDisplayName(ac.tenant) + " · " + ac.role,
		TenantSlug:          ac.tenant.Slug,
		HouseName:           houseDisplayName(ac.tenant),
		Address:             ac.tenant.Address,
		MapURL:              tenantMapURL(ac.tenant.Address),
		DisplayName:         profile.DisplayName(),
		Initials:            profile.Initials(),
		Role:                ac.role,
		DisplayVersion:      version.DisplayVersion(version.Version),
		ActivePage:          "announcements",
		Modules:             web.PortalModules{Energy: modules.Energy, Announcements: modules.Announcements, Events: modules.Events, Contacts: modules.Contacts, Documents: modules.Documents, Issues: modules.Issues, Votes: modules.Votes, Parking: modules.Parking, Handovers: modules.Handovers, Users: modules.Users, Audit: modules.Audit, Help: modules.Help},
		CanUseResidentAreas: roleCanUseResidentAreas(ac.role),
		CanViewEnergy:       modules.Energy && a.canViewEnergy(ac),
		CanManageIssues:     ac.can(capabilityManageIssues),
		CanSeeParking:       modules.Parking && (ac.can(capabilityPlatformAdmin) || profile.HasPermission(permissionParking)),
		CanManageHandovers:  canManageHandovers(ac.actor(), ac.resource()),
		CanManageUsers:      ac.can(capabilityManageUsers),
		CanViewAudit:        canViewAudit(ac.actor(), ac.resource()),
		Issues:              make([]view.IssueView, openIssues),
		UnreadAnnouncements: 0,
		Contexts:            portalContexts,
		ReleaseNotes:        version.Notes(),
	}
}

func (a *app) renderAnnouncementsTempl(w http.ResponseWriter, r *http.Request, ac authCtx, data web.AnnouncementsPageData) {
	var rendered bytes.Buffer
	if err := web.AnnouncementsPage(data).Render(r.Context(), &rendered); err != nil {
		logError("templ announcements render failed", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, prefixTenantHTMLPaths(rendered.String(), ac.tenant.Slug))
}

func (a *app) createAnnouncement(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email := ac.tenant, ac.email
	if !canManageAnnouncements(ac.actor(), ac.resource()) {
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
	announcements := ac.repositories.announcements
	created, err := announcements.Create(item)
	if err != nil {
		logError("announcement create failed", err, "tenant", tenant.Slug)
		http.Redirect(w, r, "/app/announcements?announce=error", http.StatusSeeOther)
		return
	}
	if len(attachmentHeaders) > 0 {
		if ac.repositories.attachments == nil {
			_, _ = announcements.Delete(created.ID)
			http.Redirect(w, r, "/app/announcements?announce=invalid", http.StatusSeeOther)
			return
		}
		if _, err := ac.repositories.attachments.CreateUploaded("announcement", created.ID, email, uploadedFilesFromHeaders(attachmentHeaders), time.Now()); err != nil {
			_, _ = announcements.Delete(created.ID)
			http.Redirect(w, r, "/app/announcements?announce=invalid", http.StatusSeeOther)
			return
		}
	}
	a.notifyAnnouncementPublished(tenant, created, email)
	http.Redirect(w, r, "/app/announcements?announce=created#announcement-"+url.PathEscape(created.ID), http.StatusSeeOther)
}

func (a *app) editAnnouncement(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email := ac.tenant, ac.email
	if !canManageAnnouncements(ac.actor(), ac.resource()) {
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
		uploaded, err = ac.repositories.attachments.CreateUploaded("announcement", id, email, uploadedFilesFromHeaders(attachmentHeaders), time.Now())
		if err != nil {
			http.Redirect(w, r, "/app/announcements?announce=invalid", http.StatusSeeOther)
			return
		}
	}
	ok, err := ac.repositories.announcements.Update(id, item)
	if err != nil {
		for _, attachment := range uploaded {
			_, _, _ = ac.repositories.attachments.Delete(attachment.ID, time.Now())
		}
		logError("announcement update failed", err, "tenant", tenant.Slug)
		http.Redirect(w, r, "/app/announcements?announce=error", http.StatusSeeOther)
		return
	}
	if !ok {
		for _, attachment := range uploaded {
			_, _, _ = ac.repositories.attachments.Delete(attachment.ID, time.Now())
		}
		http.Redirect(w, r, "/app/announcements?announce=missing", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/app/announcements?announce=updated", http.StatusSeeOther)
}

func (a *app) deleteAnnouncement(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant := ac.tenant
	if !canManageAnnouncements(ac.actor(), ac.resource()) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if err := parseMaybeMultipartForm(w, r, maxIssueAttachmentFormBytes, maxAttachmentBytes); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	removed, err := ac.repositories.announcements.Delete(strings.TrimSpace(r.FormValue("id")))
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

// splitPinnedAnnouncements separates the board into the notices the management
// deliberately kept on top and the ordinary chronological feed below.
func splitPinnedAnnouncements(items []announcementView) ([]announcementView, []announcementView) {
	pinned := make([]announcementView, 0, len(items))
	latest := make([]announcementView, 0, len(items))
	for _, item := range items {
		if item.Pinned {
			pinned = append(pinned, item)
			continue
		}
		latest = append(latest, item)
	}
	return pinned, latest
}

func unreadAnnouncementViewCount(items []announcementView) int {
	count := 0
	for _, item := range items {
		if item.Unread {
			count++
		}
	}
	return count
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
	for i := range views {
		views[i].CanManage = canManageAnnouncements(actorFor(actorEmail, tenantSlug, role), resourceFor(items[i].TenantSlug))
	}
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

func (a *app) enrichUnreadAnnouncementData(data map[string]any, announcements announcementRepository, announcementReads announcementReadRepository) {
	if _, ok := data["UnreadAnnouncements"]; ok {
		if _, hasFlag := data["HasUnreadAnnouncements"]; !hasFlag {
			if count, ok := data["UnreadAnnouncements"].(int); ok {
				data["HasUnreadAnnouncements"] = count > 0
			}
		}
		return
	}
	tenant, ok := data["Tenant"].(tenantConfig)
	if !ok || tenant.Slug == "" || announcements == nil || announcementReads == nil {
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
	lastSeen := announcementReads.LastSeen(email)
	count := unreadAnnouncementCount(announcements.Visible(now), lastSeen, now)
	data["UnreadAnnouncements"] = count
	data["HasUnreadAnnouncements"] = count > 0
}
