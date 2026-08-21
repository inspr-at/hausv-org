package server

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/version"
	"github.com/inspr-at/hausv-org/internal/view"
	"github.com/inspr-at/hausv-org/internal/web"
)

func (a *app) issues(w http.ResponseWriter, r *http.Request, ac authCtx) {
	a.renderIssuesPage(w, r, ac, false)
}

func (a *app) issueBoard(w http.ResponseWriter, r *http.Request, ac authCtx) {
	a.renderIssuesPage(w, r, ac, true)
}

func (a *app) issueTriage(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !ac.can(capabilityManageIssues) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	item, found := ac.repositories.issues.Get(id)
	if !found {
		http.Redirect(w, r, "/app/anliegen/board?issue=missing", http.StatusSeeOther)
		return
	}
	views := a.issueViewsForActor(ac.tenantRef, []residentIssue{item}, ac.role, ac.email)
	if len(views) != 1 {
		http.Redirect(w, r, "/app/anliegen/board?issue=missing", http.StatusSeeOther)
		return
	}
	step := strings.TrimSpace(r.URL.Query().Get("step"))
	switch step {
	case "1", "2", "done", "message", "sent", "resolution-sent":
	default:
		if normalizeIssueStatus(item.Status) == issueStatusOpen {
			step = "1"
		} else {
			step = "message"
		}
	}
	portal := a.issuesPortalContext(ac)
	portal.Title = views[0].Title + " · " + portal.Title
	a.renderIssueTriageTempl(w, r, web.IssueTriagePageData{
		Portal:       portal,
		AssetVersion: version.AssetVersion(),
		ActorEmail:   normalizeEmail(ac.email),
		Issue:        views[0],
		TriageStep:   step,
	})
}

func (a *app) issueResidentDetail(w http.ResponseWriter, r *http.Request, ac authCtx) {
	id := strings.TrimSpace(r.PathValue("id"))
	item, found := ac.repositories.issues.Get(id)
	if !found {
		http.Redirect(w, r, "/app/anliegen?issue=missing", http.StatusSeeOther)
		return
	}
	if !a.canViewIssueForActor(ac.tenantRef, item, ac.email, ac.role) {
		http.Error(w, "Dieses Anliegen ist für diesen Zugang nicht sichtbar.", http.StatusForbidden)
		return
	}
	views := a.issueViewsForActor(ac.tenantRef, []residentIssue{item}, ac.role, ac.email)
	if len(views) != 1 {
		http.Redirect(w, r, "/app/anliegen?issue=missing", http.StatusSeeOther)
		return
	}
	if ac.can(capabilityManageIssues) {
		http.Redirect(w, r, "/app/anliegen/board/"+url.PathEscape(item.ID), http.StatusSeeOther)
		return
	}
	a.render(w, "issueResidentDetail", a.withBase(ac, map[string]any{
		"Title":        item.Title,
		"ActivePage":   "issues",
		"Issue":        views[0],
		"IssueCreated": r.URL.Query().Get("created") == "1" && normalizeEmail(item.AuthorEmail) == normalizeEmail(ac.email),
	}))
}

func (a *app) renderIssuesPage(w http.ResponseWriter, r *http.Request, ac authCtx, boardOnly bool) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	issues := []issueView{}
	manageIssues := []issueView{}
	canManageIssues := ac.can(capabilityManageIssues)
	canCreateIssue := canCreateResidentIssue(ac.actor(), ac.resource())
	totalIssueCount := 0
	openIssueCount := 0
	urgentIssueCount := 0
	if boardOnly && !canManageIssues {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	filters := issueBoardFiltersFromQuery(r.URL.Query())
	if ac.repositories.issues != nil {
		allTenantIssues := ac.repositories.issues.List()
		totalIssueCount = len(allTenantIssues)
		openIssueCount = issueOpenCount(allTenantIssues)
		urgentIssueCount = issuePriorityCount(allTenantIssues, issuePriorityUrgent)
		if !boardOnly {
			if canManageIssues {
				issues = a.issueViewsForActor(ac.tenantRef, ac.repositories.issues.ListAuthor(email), role, email)
			} else {
				issues = a.issueViewsForActor(ac.tenantRef, a.visibleIssuesForActor(ac.tenantRef, email, role), role, email)
			}
		}
		if canManageIssues {
			filteredIssues := filterIssueBoard(allTenantIssues, filters)
			if boardOnly {
				manageIssues = a.issueViewsForActor(ac.tenantRef, filteredIssues, role, email)
			}
		}
	}
	msg, msgOK := issueMessage(r.URL.Query().Get("issue"))
	openIssueCreate := canCreateIssue && (!msgOK && msg != "" || r.URL.Query().Get("new") == "1" || len(issues) == 0)
	calendarFeedURL := ""
	if token, err := a.calendarFeedToken(email, tenant.Slug); err == nil {
		calendarFeedURL = a.publicBaseURL(r, tenant) + "/calendar/" + url.PathEscape(token) + ".ics"
	}
	serviceContacts := a.serviceContactOptions(ac.repositories.contacts)
	if boardOnly {
		a.renderIssueBoardTempl(w, r, web.IssueBoardPageData{
			Portal:                  a.issuesPortalContext(ac),
			AssetVersion:            version.AssetVersion(),
			ActorEmail:              email,
			BoardAction:             issueBoardAction(boardOnly),
			CalendarFeedURL:         calendarFeedURL,
			CanCreateIssue:          canCreateIssue,
			CanManageAnnouncements:  canManageAnnouncements(ac.actor(), ac.resource()),
			IsServiceProvider:       isServiceProviderRole(role),
			Filters:                 issueBoardFilterOptions(filters),
			ServiceProviderContacts: serviceContacts,
			Issues:                  manageIssues,
			TotalIssueCount:         totalIssueCount,
			OpenIssueCount:          openIssueCount,
			UrgentIssueCount:        urgentIssueCount,
			IssuesEmpty:             emptyState("Keine Anliegen im Haus", "Sobald ein Anliegen gemeldet wird, erscheint es hier für die Bearbeitung."),
		})
		return
	}
	a.renderIssuesTempl(w, r, web.IssuesPageData{
		Portal:            a.issuesPortalContext(ac),
		AssetVersion:      version.AssetVersion(),
		CalendarFeedURL:   calendarFeedURL,
		CanManageIssues:   canManageIssues,
		CanCreateIssue:    canCreateIssue,
		IsServiceProvider: isServiceProviderRole(role),
		Issues:            issues,
		IssuesEmpty:       emptyState("Noch kein Anliegen", "Nach dem Absenden erscheint das Anliegen hier mit Status und Rückfragen."),
		Message:           msg,
		MessageOK:         msgOK,
		OpenIssueCreate:   openIssueCreate,
	})
}

func (a *app) issuesPortalContext(ac authCtx) web.PortalPageData {
	tenant, email, role := ac.tenant, ac.email, ac.role
	profile := a.profileForTenant(email, tenant.Slug)
	modules := a.portalModulesFor(tenant.Slug)
	contexts := a.portalContextsFor(email, tenant.Slug, role)
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
	unreadAnnouncements := 0
	if ac.repositories.announcements != nil && ac.repositories.announcementReads != nil && strings.TrimSpace(email) != "" {
		now := time.Now()
		unreadAnnouncements = unreadAnnouncementCount(ac.repositories.announcements.Visible(now), ac.repositories.announcementReads.LastSeen(email), now)
	}
	openIssues := 0
	if a.issueStore != nil {
		openIssues = issueOpenCount(a.visibleIssuesForActor(ac.tenantRef, email, role))
	}
	return web.PortalPageData{
		Title:               "Anliegen · " + houseDisplayName(tenant) + " · " + role,
		TenantSlug:          tenant.Slug,
		HouseName:           houseDisplayName(tenant),
		Address:             tenant.Address,
		MapURL:              tenantMapURL(tenant.Address),
		HeroImageURL:        tenant.HeroImageURL,
		BrandIcon:           tenant.BrandIcon,
		BrandMarkSVG:        tenantBrandMarkSVG(tenant.BrandIcon),
		Map:                 portalMapForTenant(tenant),
		DisplayName:         profile.DisplayName(),
		Initials:            profile.Initials(),
		Role:                role,
		DisplayVersion:      version.DisplayVersion(version.Version),
		ActivePage:          "issues",
		Modules:             web.PortalModules{Energy: modules.Energy, Announcements: modules.Announcements, Events: modules.Events, Contacts: modules.Contacts, Documents: modules.Documents, Issues: modules.Issues, Votes: modules.Votes, Parking: modules.Parking, Handovers: modules.Handovers, Users: modules.Users, Audit: modules.Audit, Help: modules.Help},
		CanUseResidentAreas: roleCanUseResidentAreas(role),
		CanViewEnergy:       modules.Energy && a.canViewEnergy(ac),
		CanManageIssues:     ac.can(capabilityManageIssues),
		CanSeeParking:       modules.Parking && (ac.can(capabilityPlatformAdmin) || profile.HasPermission(permissionParking)),
		CanManageHandovers:  modules.Handovers && canManageHandovers(ac.actor(), ac.resource()),
		CanManageUsers:      modules.Users && ac.can(capabilityManageUsers),
		CanViewAudit:        modules.Audit && canViewAudit(ac.actor(), ac.resource()),
		Issues:              make([]view.IssueView, openIssues),
		UnreadAnnouncements: unreadAnnouncements,
		Contexts:            portalContexts,
		ReleaseNotes:        version.Notes(),
	}
}

func (a *app) renderIssuesTempl(w http.ResponseWriter, r *http.Request, data web.IssuesPageData) {
	var rendered bytes.Buffer
	if err := web.IssuesPage(data).Render(r.Context(), &rendered); err != nil {
		logError("templ issues render failed", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, prefixTenantHTMLPaths(rendered.String(), data.Portal.TenantSlug))
}

func (a *app) renderIssueBoardTempl(w http.ResponseWriter, r *http.Request, data web.IssueBoardPageData) {
	var rendered bytes.Buffer
	if err := web.IssueBoardPage(data).Render(r.Context(), &rendered); err != nil {
		logError("templ issue board render failed", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, prefixTenantHTMLPaths(rendered.String(), data.Portal.TenantSlug))
}

func (a *app) renderIssueTriageTempl(w http.ResponseWriter, r *http.Request, data web.IssueTriagePageData) {
	var rendered bytes.Buffer
	if err := web.IssueTriagePage(data).Render(r.Context(), &rendered); err != nil {
		logError("templ issue triage render failed", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, prefixTenantHTMLPaths(rendered.String(), data.Portal.TenantSlug))
}

func issueMessage(status string) (string, bool) {
	switch status {
	case "created":
		return "Anliegen gespeichert. Die Verwaltung sieht es im nächsten Bearbeitungsschritt.", true
	case "updated":
		return "Anliegen aktualisiert.", true
	case "invalid":
		return "Bitte Kategorie, Ort, Titel und Beschreibung prüfen.", false
	case "photo":
		return "Anhänge konnten nicht übernommen werden. Erlaubt sind Bilddateien oder PDF bis 10 MB, maximal 10 Dateien.", false
	case "missing":
		return "Dieses Anliegen wurde nicht gefunden.", false
	case "termin":
		return "Für den Status \"Termin vereinbart\" bitte Datum und Uhrzeit angeben.", false
	case "error":
		return "Das Anliegen konnte nicht gespeichert werden.", false
	default:
		return "", false
	}
}

func (a *app) createIssue(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email := ac.tenant, ac.email
	if !canCreateResidentIssue(ac.actor(), ac.resource()) {
		http.Error(w, "Dieser Zugang kann keine neuen Anliegen anlegen.", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxIssueAttachmentFormBytes)
	if err := r.ParseMultipartForm(maxAttachmentBytes); err != nil {
		http.Redirect(w, r, "/app/anliegen?issue=invalid", http.StatusSeeOther)
		return
	}
	profile := a.profileForTenant(email, tenant.Slug)
	now := time.Now()
	item, err := issueFromForm(r, tenant.Slug, profile, now)
	if err != nil {
		http.Redirect(w, r, "/app/anliegen?issue=invalid", http.StatusSeeOther)
		return
	}
	attachmentHeaders, err := issueAttachmentHeaders(r)
	if err != nil {
		http.Redirect(w, r, "/app/anliegen?issue=photo", http.StatusSeeOther)
		return
	}
	var uploaded []attachmentRecord
	if len(attachmentHeaders) > 0 {
		if a.attachmentStore == nil {
			http.Redirect(w, r, "/app/anliegen?issue=photo", http.StatusSeeOther)
			return
		}
		uploaded, err = ac.repositories.attachments.CreateUploaded("issue", item.ID, email, uploadedFilesFromHeaders(attachmentHeaders), now)
		if err != nil {
			http.Redirect(w, r, "/app/anliegen?issue=photo", http.StatusSeeOther)
			return
		}
	}
	createdID := item.ID
	if ac.repositories.issues != nil {
		created, err := ac.repositories.issues.Create(item)
		if err != nil {
			for _, attachment := range uploaded {
				_, _, _ = ac.repositories.attachments.Delete(attachment.ID, time.Now())
			}
			logError("issue create failed", err, "tenant", tenant.Slug, "actor", redactedEmail(email))
			http.Redirect(w, r, "/app/anliegen?issue=error", http.StatusSeeOther)
			return
		}
		createdID = created.ID
		a.notifyIssueCreated(tenant, created)
	}
	http.Redirect(w, r, "/app/anliegen/"+url.PathEscape(createdID)+"?created=1", http.StatusSeeOther)
}

func (a *app) addIssueComment(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	r.Body = http.MaxBytesReader(w, r.Body, maxIssueAttachmentFormBytes)
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type"))), "multipart/form-data") {
		if err := r.ParseMultipartForm(maxAttachmentBytes); err != nil {
			http.Redirect(w, r, "/app/anliegen?issue=invalid", http.StatusSeeOther)
			return
		}
	} else if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	id := strings.TrimSpace(r.FormValue("id"))
	existing, found := ac.repositories.issues.Get(id)
	if !found {
		http.Redirect(w, r, "/app/anliegen?issue=missing", http.StatusSeeOther)
		return
	}
	actor := actorFor(email, tenant.Slug, role)
	resource := resourceFor(existing.TenantSlug)
	canManage := can(actor, capabilityManageIssues, resource)
	readOnly := can(actor, capabilityOversight, resource) && !canManage
	if !a.canViewIssueForActor(ac.tenantRef, existing, email, role) {
		http.Error(w, "Dieser Kommentar ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	isOwner := normalizeEmail(existing.AuthorEmail) == normalizeEmail(email)
	canServiceComment := isServiceProviderRole(role) && issueAssignedToActor(existing, email) && issueIsOpen(existing)
	if readOnly || (!canManage && !isOwner && !canServiceComment) {
		http.Error(w, "Dieser Kommentar ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	body := strings.TrimSpace(r.FormValue("body"))
	if len([]rune(body)) > 3000 {
		http.Redirect(w, r, "/app/anliegen?issue=invalid", http.StatusSeeOther)
		return
	}
	attachmentHeaders, err := issueAttachmentHeaders(r)
	if err != nil {
		http.Redirect(w, r, "/app/anliegen?issue=photo", http.StatusSeeOther)
		return
	}
	// A comment must carry either text or at least one photo (HAUSV-128:
	// standalone photo upload). Reject only the truly-empty submit.
	if body == "" && len(attachmentHeaders) == 0 {
		http.Redirect(w, r, "/app/anliegen?issue=invalid", http.StatusSeeOther)
		return
	}
	commentID, err := randomToken(10)
	if err != nil {
		http.Redirect(w, r, "/app/anliegen?issue=error", http.StatusSeeOther)
		return
	}
	var uploaded []attachmentRecord
	if len(attachmentHeaders) > 0 {
		if a.attachmentStore == nil {
			http.Redirect(w, r, "/app/anliegen?issue=photo", http.StatusSeeOther)
			return
		}
		uploaded, err = ac.repositories.attachments.CreateUploaded("issue-comment", commentID, email, uploadedFilesFromHeaders(attachmentHeaders), time.Now())
		if err != nil {
			http.Redirect(w, r, "/app/anliegen?issue=photo", http.StatusSeeOther)
			return
		}
	}
	profile := a.profileForTenant(email, tenant.Slug)
	commentKind := issueCommentKindNeutral
	if canManage {
		commentKind = normalizeIssueCommentKind(r.FormValue("message_type"))
		if commentKind != issueCommentKindQuestion {
			commentKind = issueCommentKindInformation
		}
	} else if isOwner {
		if _, open := issueOpenQuestion(existing.Comments); open {
			commentKind = issueCommentKindAnswer
		}
	}
	updated, ok, err := ac.repositories.issues.AddComment(id, issueComment{
		ID:          commentID,
		AuthorEmail: email,
		AuthorName:  profile.DisplayName(),
		Body:        body,
		Kind:        commentKind,
		CreatedAt:   time.Now(),
	})
	if err != nil {
		for _, attachment := range uploaded {
			_, _, _ = ac.repositories.attachments.Delete(attachment.ID, time.Now())
		}
		logError("issue comment failed", err, "tenant", tenant.Slug, "issue_id", id)
		http.Redirect(w, r, "/app/anliegen?issue=error", http.StatusSeeOther)
		return
	}
	if !ok {
		for _, attachment := range uploaded {
			_, _, _ = ac.repositories.attachments.Delete(attachment.ID, time.Now())
		}
		http.Redirect(w, r, "/app/anliegen?issue=missing", http.StatusSeeOther)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: email,
		ActorRole:  role,
		Action:     auditActionIssueComment,
		TargetType: "issue",
		TargetID:   updated.ID,
		Summary:    "Kommentar zu Anliegen hinzugefügt",
		Details: map[string]string{
			"comment_id":   commentID,
			"message_type": commentKind,
			"has_file":     strconv.FormatBool(len(uploaded) > 0),
			"file_count":   strconv.Itoa(len(uploaded)),
		},
	})
	a.notifyIssueUpdated(tenant, updated, email, "Neuer Kommentar zu Anliegen \""+updated.Title+"\"")
	if canManage {
		if redirect := issueContextRedirect(updated.ID, internalTenantPath(r, r.FormValue("redirect"))); redirect != "" {
			http.Redirect(w, r, redirect, http.StatusSeeOther)
			return
		}
	}
	if isOwner {
		if redirect := issueResidentContextRedirect(updated.ID, internalTenantPath(r, r.FormValue("redirect"))); redirect != "" {
			http.Redirect(w, r, redirect, http.StatusSeeOther)
			return
		}
	}
	http.Redirect(w, r, "/app/anliegen?issue=updated", http.StatusSeeOther)
}

func (a *app) deleteIssueComment(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	commentID := strings.TrimSpace(r.FormValue("comment_id"))
	issue, comment, found := a.issueCommentTarget(ac.tenantRef, commentID)
	if !found {
		http.Redirect(w, r, "/app/anliegen?issue=missing", http.StatusSeeOther)
		return
	}
	if !a.canDeleteIssueComment(ac.tenantRef, issue, comment, email, role) {
		http.Error(w, "Dieser Kommentar kann mit diesem Zugang nicht gelöscht werden.", http.StatusForbidden)
		return
	}
	updated, deleted, err := ac.repositories.issues.DeleteComment(issue.ID, comment.ID, time.Now())
	if err != nil {
		logError("issue comment delete failed", err, "tenant", tenant.Slug, "issue_id", issue.ID, "comment_id", comment.ID)
		http.Redirect(w, r, "/app/anliegen?issue=error", http.StatusSeeOther)
		return
	}
	if !deleted {
		http.Redirect(w, r, "/app/anliegen?issue=missing", http.StatusSeeOther)
		return
	}
	if ac.repositories.attachments != nil {
		for _, attachment := range ac.repositories.attachments.ListEntity("issue-comment", comment.ID) {
			_, _, _ = ac.repositories.attachments.Delete(attachment.ID, time.Now())
		}
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: email,
		ActorRole:  role,
		Action:     auditActionIssueCommentDelete,
		TargetType: "issue",
		TargetID:   updated.ID,
		Summary:    "Kommentar zu Anliegen gelöscht",
		Details: map[string]string{
			"comment_id": comment.ID,
		},
	})
	a.notifyIssueUpdated(tenant, updated, email, "Kommentar zu Anliegen \""+updated.Title+"\" gelöscht")
	http.Redirect(w, r, "/app/anliegen?issue=updated", http.StatusSeeOther)
}

func (a *app) confirmIssueResolution(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	id := strings.TrimSpace(r.FormValue("id"))
	existing, found := ac.repositories.issues.Get(id)
	if !found {
		http.Redirect(w, r, "/app/anliegen?issue=missing", http.StatusSeeOther)
		return
	}
	isOwner := normalizeEmail(existing.AuthorEmail) == normalizeEmail(ac.email)
	if !isOwner || ac.can(capabilityManageIssues) || normalizeIssueStatus(existing.Status) != issueStatusDone {
		http.Error(w, "Diese Rückmeldung ist nur für die meldende Person möglich.", http.StatusForbidden)
		return
	}
	resolved := strings.TrimSpace(r.FormValue("resolved"))
	if resolved != "yes" && resolved != "no" {
		http.Redirect(w, r, "/app/anliegen/"+url.PathEscape(id), http.StatusSeeOther)
		return
	}
	status := issueStatusDone
	if resolved == "no" {
		status = issueStatusNew
	}
	profile := a.profileForTenant(ac.email, ac.tenant.Slug)
	updated, ok, err := ac.repositories.issues.UpdateWorkflow(id, issueWorkflowUpdate{
		Status:              status,
		Priority:            existing.Priority,
		AssigneeEmail:       existing.AssigneeEmail,
		ResolutionConfirmed: resolved == "yes",
		UpdateResolution:    true,
		ActorEmail:          ac.email,
		ActorName:           profile.DisplayName(),
		ChangedAt:           time.Now(),
	})
	if err != nil || !ok {
		http.Redirect(w, r, "/app/anliegen?issue=error", http.StatusSeeOther)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: ac.tenant.Slug,
		ActorEmail: ac.email,
		ActorRole:  ac.role,
		Action:     auditActionIssueWorkflow,
		TargetType: "issue",
		TargetID:   updated.ID,
		Summary:    "Lösungsstatus zu Anliegen bestätigt",
		Details: map[string]string{
			"resolution": resolved,
			"status":     normalizeIssueStatus(updated.Status),
		},
	})
	a.notifyIssueUpdated(ac.tenant, updated, ac.email, "Rückmeldung zur Lösung von Anliegen \""+updated.Title+"\"")
	http.Redirect(w, r, "/app/anliegen/"+url.PathEscape(updated.ID), http.StatusSeeOther)
}

func (a *app) updateIssueWorkflow(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	if err := parseMaybeMultipartForm(w, r, maxIssueAttachmentFormBytes, maxAttachmentBytes); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	id := strings.TrimSpace(r.FormValue("id"))
	existing, found := ac.repositories.issues.Get(id)
	if !found {
		http.Redirect(w, r, "/app/anliegen?issue=missing", http.StatusSeeOther)
		return
	}

	actor := actorFor(email, tenant.Slug, role)
	resource := resourceFor(existing.TenantSlug)
	canManage := can(actor, capabilityManageIssues, resource)
	readOnly := can(actor, capabilityOversight, resource) && !canManage
	if !a.canViewIssueForActor(ac.tenantRef, existing, email, role) {
		http.Error(w, "Dieser Statuswechsel ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	isOwner := normalizeEmail(existing.AuthorEmail) == normalizeEmail(email)
	status := normalizeIssueStatus(r.FormValue("status"))
	if status == "" {
		http.Redirect(w, r, "/app/anliegen?issue=invalid", http.StatusSeeOther)
		return
	}
	priority := normalizeIssuePriority(r.FormValue("priority"))
	assignee := normalizeEmail(r.FormValue("assignee_email"))
	if r.FormValue("remove_assignee") == "1" {
		assignee = ""
	}
	if canManage && !a.serviceAccessEnabled && assignee != normalizeEmail(existing.AssigneeEmail) && a.shouldInviteServiceProvider(tenant.Slug, assignee) {
		http.Error(w, serviceProviderAccessClosedMessage, http.StatusForbidden)
		return
	}
	proposal, serviceProposalProvided, err := issueServiceProposalFromForm(r.Form)
	if err != nil {
		http.Redirect(w, r, "/app/anliegen?issue=invalid", http.StatusSeeOther)
		return
	}
	estimateAmount, estimateNote, estimateProvided, err := issueEstimateFromForm(r.Form)
	if err != nil {
		http.Redirect(w, r, "/app/anliegen?issue=invalid", http.StatusSeeOther)
		return
	}
	estimateHeaders, err := attachmentFormHeaders(r, 1, "estimate_attachment")
	if err != nil {
		http.Redirect(w, r, "/app/anliegen?issue=invalid", http.StatusSeeOther)
		return
	}
	if len(estimateHeaders) > 0 {
		estimateProvided = true
		if a.attachmentStore == nil {
			http.Redirect(w, r, "/app/anliegen?issue=invalid", http.StatusSeeOther)
			return
		}
	}
	if !canManage {
		canServiceAct := isServiceProviderRole(role) && issueAssignedToActor(existing, email) && issueIsOpen(existing)
		if canServiceAct {
			existingStatus := normalizeIssueStatus(existing.Status)
			if existingStatus == "" {
				existingStatus = issueStatusOpen
			}
			statusUnchanged := existingStatus == status
			if readOnly || r.FormValue("priority") != "" || r.FormValue("assignee_email") != "" || (!statusUnchanged && !canServiceProviderTransition(existing.Status, status)) {
				http.Error(w, "Dieser Statuswechsel ist der Verwaltung vorbehalten.", http.StatusForbidden)
				return
			}
		} else {
			if readOnly || !isOwner || r.FormValue("priority") != "" || r.FormValue("assignee_email") != "" || serviceProposalProvided || estimateProvided || !canResidentTransition(existing.Status, status) {
				http.Error(w, "Dieser Statuswechsel ist der Verwaltung vorbehalten.", http.StatusForbidden)
				return
			}
		}
		priority = normalizeIssuePriority(existing.Priority)
		assignee = normalizeEmail(existing.AssigneeEmail)
	}
	if priority == "" {
		http.Redirect(w, r, "/app/anliegen?issue=invalid", http.StatusSeeOther)
		return
	}
	if status == issueStatusScheduled {
		scheduledStart := existing.ServiceProposedStart
		if serviceProposalProvided {
			scheduledStart = proposal.Start
		}
		if scheduledStart.IsZero() {
			http.Redirect(w, r, "/app/anliegen?issue=termin", http.StatusSeeOther)
			return
		}
	}
	profile := a.profileForTenant(email, tenant.Slug)
	detailsUpdate, detailsProvided, err := issueDetailsFromForm(r.Form)
	if err != nil {
		http.Redirect(w, r, "/app/anliegen?issue=invalid", http.StatusSeeOther)
		return
	}
	if detailsProvided && !canManage {
		http.Error(w, "Dieser Statuswechsel ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	var uploadedEstimates []attachmentRecord
	if len(estimateHeaders) > 0 {
		uploadedEstimates, err = ac.repositories.attachments.CreateUploaded("issue-estimate", id, email, uploadedFilesFromHeaders(estimateHeaders), time.Now())
		if err != nil {
			logError("issue estimate upload failed", err, "tenant", tenant.Slug, "issue_id", id)
			http.Redirect(w, r, "/app/anliegen?issue=invalid", http.StatusSeeOther)
			return
		}
	}
	updated, _, err := ac.repositories.issues.UpdateWorkflow(id, issueWorkflowUpdate{
		Status:                status,
		Priority:              priority,
		AssigneeEmail:         assignee,
		Body:                  detailsUpdate.Body,
		LocationType:          detailsUpdate.LocationType,
		LocationDetail:        detailsUpdate.LocationDetail,
		UpdateDetails:         detailsProvided,
		ServiceProposal:       proposal.Text,
		ServiceProposedStart:  proposal.Start,
		ServiceProposedEnd:    proposal.End,
		UpdateServiceProposal: serviceProposalProvided,
		EstimateAmountCents:   estimateAmount,
		EstimateNote:          estimateNote,
		UpdateEstimate:        estimateProvided,
		ActorEmail:            email,
		ActorName:             profile.DisplayName(),
		ChangedAt:             time.Now(),
	})
	if err != nil {
		for _, attachment := range uploadedEstimates {
			_, _, _ = ac.repositories.attachments.Delete(attachment.ID, time.Now())
		}
		logError("issue workflow update failed", err, "tenant", tenant.Slug, "issue_id", id)
		http.Redirect(w, r, "/app/anliegen?issue=error", http.StatusSeeOther)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: email,
		ActorRole:  role,
		Action:     auditActionIssueWorkflow,
		TargetType: "issue",
		TargetID:   updated.ID,
		Summary:    "Anliegen bearbeitet",
		Details: map[string]string{
			"status":   normalizeIssueStatus(updated.Status),
			"priority": normalizeIssuePriority(updated.Priority),
		},
	})
	if estimateProvided {
		details := map[string]string{
			"has_file": strconv.FormatBool(len(uploadedEstimates) > 0),
		}
		if updated.EstimateAmountCents > 0 {
			details["estimate_amount"] = formatIssueEstimateAmount(updated.EstimateAmountCents)
		}
		if len(uploadedEstimates) > 0 {
			details["file_count"] = strconv.Itoa(len(uploadedEstimates))
		}
		a.recordAudit(auditEvent{
			TenantSlug: tenant.Slug,
			ActorEmail: email,
			ActorRole:  role,
			Action:     auditActionIssueEstimate,
			TargetType: "issue",
			TargetID:   updated.ID,
			Summary:    "Kostenvoranschlag aktualisiert",
			Details:    details,
		})
	}
	a.handleIssueServiceAssignmentChange(r, tenant, existing, updated, email, role)
	a.notifyIssueUpdated(tenant, updated, email, "Anliegen \""+updated.Title+"\" aktualisiert")
	if canManage {
		if redirect := issueContextRedirect(updated.ID, internalTenantPath(r, r.FormValue("redirect"))); redirect != "" {
			http.Redirect(w, r, redirect, http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/app/anliegen/board?issue=updated#issue-"+url.PathEscape(updated.ID), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/app/anliegen?issue=updated", http.StatusSeeOther)
}

func issueContextRedirect(issueID string, requested string) string {
	base := "/app/anliegen/board/" + url.PathEscape(strings.TrimSpace(issueID))
	switch strings.TrimSpace(requested) {
	case base, base + "?step=1", base + "?step=2", base + "?step=done", base + "?step=message", base + "?step=sent", base + "?step=resolution-sent":
		return strings.TrimSpace(requested)
	default:
		return ""
	}
}

func issueResidentContextRedirect(issueID string, requested string) string {
	base := "/app/anliegen/" + url.PathEscape(strings.TrimSpace(issueID))
	if strings.TrimSpace(requested) == base {
		return base
	}
	return ""
}

func (a *app) issueManagerEmails(tenantSlug string) []string {
	tenantSlug = normalizeSlug(tenantSlug)
	recipients := []string{}
	for email, profile := range a.profiles {
		if profile.HasTenant(tenantSlug) {
			membership := profile.ForTenant(tenantSlug)
			if can(actorFor(email, tenantSlug, membership.Role), capabilityManageIssues, resourceFor(tenantSlug)) {
				recipients = append(recipients, email)
			}
		}
	}
	for email := range a.admins {
		recipients = append(recipients, email)
	}
	if a.inviteStore != nil {
		for _, profile := range a.inviteStore.List() {
			if profile.HasTenant(tenantSlug) {
				membership := profile.ForTenant(tenantSlug)
				if can(actorFor(profile.Email, tenantSlug, membership.Role), capabilityManageIssues, resourceFor(tenantSlug)) {
					recipients = append(recipients, profile.Email)
				}
			}
		}
	}
	return uniqueEmails(recipients)
}

func issueFromForm(r *http.Request, tenantSlug string, author userProfile, now time.Time) (residentIssue, error) {
	id, err := randomToken(12)
	if err != nil {
		return residentIssue{}, err
	}
	body := strings.TrimSpace(r.FormValue("body"))
	if body == "" || len([]rune(body)) > 4000 {
		return residentIssue{}, fmt.Errorf("invalid issue text")
	}
	title := strings.Join(strings.Fields(r.FormValue("title")), " ")
	if title == "" {
		title = issueTitleFromBody(body)
	}
	if title == "" || len([]rune(title)) > 140 {
		return residentIssue{}, fmt.Errorf("invalid issue text")
	}
	locationType := normalizeIssueLocation(r.FormValue("location_type"))
	if locationType == "" {
		return residentIssue{}, fmt.Errorf("invalid location")
	}
	locationDetail := strings.TrimSpace(r.FormValue("location_detail"))
	if len([]rune(locationDetail)) > 160 {
		return residentIssue{}, fmt.Errorf("location detail too long")
	}
	category := normalizeIssueCategory(r.FormValue("category"))
	if category == "" {
		return residentIssue{}, fmt.Errorf("invalid category")
	}
	return residentIssue{
		ID:             id,
		TenantSlug:     normalizeSlug(tenantSlug),
		AuthorEmail:    normalizeEmail(author.Email),
		AuthorName:     author.DisplayName(),
		Category:       category,
		Title:          title,
		Body:           body,
		LocationType:   locationType,
		LocationDetail: locationDetail,
		Status:         issueStatusOpen,
		Priority:       issuePriorityNorm,
		CreatedAt:      now.UTC(),
		UpdatedAt:      now.UTC(),
	}, nil
}

// issueTitleFromBody derives the compact, canonical title used when the
// resident leaves the optional title untouched. The server owns this fallback
// so the form remains fully usable without JavaScript.
func issueTitleFromBody(body string) string {
	clean := strings.Join(strings.Fields(body), " ")
	if clean == "" {
		return ""
	}

	runes := []rune(clean)
	end := len(runes)
	for index, char := range runes {
		if char != '.' && char != '!' && char != '?' {
			continue
		}
		if index == len(runes)-1 || runes[index+1] == ' ' {
			// A period near the start is commonly part of a German abbreviation
			// (for example "z. B." or "z.B."). It is only a sentence boundary
			// once the prefix carries enough meaning, unless it ends the input.
			prefix := strings.TrimSpace(string(runes[:index]))
			if index < len(runes)-1 && len([]rune(prefix)) < 8 {
				continue
			}
			end = index
			break
		}
	}
	title := strings.TrimSpace(string(runes[:end]))
	if title == "" {
		title = clean
	}

	runes = []rune(title)
	if len(runes) <= 76 {
		return title
	}
	cut := 75
	for index := cut - 1; index >= 0; index-- {
		if runes[index] != ' ' {
			continue
		}
		if index >= 38 {
			cut = index
		}
		break
	}
	return strings.TrimSpace(string(runes[:cut])) + "…"
}

type serviceProposalInput struct {
	Text  string
	Start time.Time
	End   time.Time
}

func issueServiceProposalFromForm(values url.Values) (serviceProposalInput, bool, error) {
	_, hasText := values["service_proposal"]
	_, hasStart := values["service_start"]
	_, hasEnd := values["service_end"]
	if !hasText && !hasStart && !hasEnd {
		return serviceProposalInput{}, false, nil
	}
	text := strings.TrimSpace(values.Get("service_proposal"))
	if len([]rune(text)) > 180 {
		return serviceProposalInput{}, true, fmt.Errorf("service proposal too long")
	}
	start, err := parseOptionalLocalDateTime(values.Get("service_start"), time.Time{})
	if err != nil {
		return serviceProposalInput{}, true, err
	}
	end, err := parseOptionalLocalDateTime(values.Get("service_end"), time.Time{})
	if err != nil {
		return serviceProposalInput{}, true, err
	}
	if !start.IsZero() && !end.IsZero() && end.Before(start) {
		return serviceProposalInput{}, true, fmt.Errorf("appointment end before start")
	}
	return serviceProposalInput{Text: text, Start: start, End: end}, true, nil
}

func issueEstimateFromForm(values url.Values) (int64, string, bool, error) {
	if values == nil {
		return 0, "", false, nil
	}
	_, amountProvided := values["estimate_amount"]
	_, noteProvided := values["estimate_note"]
	if !amountProvided && !noteProvided {
		return 0, "", false, nil
	}
	amount, err := parseIssueEstimateAmountCents(values.Get("estimate_amount"))
	if err != nil {
		return 0, "", true, err
	}
	note := strings.TrimSpace(values.Get("estimate_note"))
	if len([]rune(note)) > 240 {
		return 0, "", true, fmt.Errorf("estimate note too long")
	}
	return amount, note, true, nil
}

type issueDetailsInput struct {
	Body           string
	LocationType   string
	LocationDetail string
}

func issueDetailsFromForm(values url.Values) (issueDetailsInput, bool, error) {
	if values == nil {
		return issueDetailsInput{}, false, nil
	}
	if values.Get("update_details") != "1" {
		return issueDetailsInput{}, false, nil
	}
	body := strings.TrimSpace(values.Get("body"))
	locationType := normalizeIssueLocation(values.Get("location_type"))
	locationDetail := strings.TrimSpace(values.Get("location_detail"))

	if body == "" || len([]rune(body)) > 4000 {
		return issueDetailsInput{}, true, fmt.Errorf("invalid body")
	}
	if locationType == "" {
		return issueDetailsInput{}, true, fmt.Errorf("invalid location_type")
	}
	if len([]rune(locationDetail)) > 160 {
		return issueDetailsInput{}, true, fmt.Errorf("location_detail too long")
	}

	return issueDetailsInput{
		Body:           body,
		LocationType:   locationType,
		LocationDetail: locationDetail,
	}, true, nil
}

func issuePhotoHeader(r *http.Request) (*multipart.FileHeader, bool) {
	if r.MultipartForm == nil {
		return nil, false
	}
	files := r.MultipartForm.File["photo"]
	if len(files) == 0 {
		files = r.MultipartForm.File["photos"]
	}
	if len(files) == 0 || files[0] == nil || files[0].Filename == "" || files[0].Size == 0 {
		return nil, false
	}
	return files[0], true
}

func issueAttachmentHeaders(r *http.Request) ([]*multipart.FileHeader, error) {
	return attachmentFormHeaders(r, maxIssueAttachmentCount, "attachments", "photos", "photo")
}

func issueBoardAction(boardOnly bool) string {
	if boardOnly {
		return "/app/anliegen/board"
	}
	return "/app/anliegen"
}

func issueBoardFiltersFromQuery(values url.Values) issueBoardFilterView {
	priority := ""
	if raw := strings.TrimSpace(values.Get("priority")); raw != "" {
		priority = normalizeIssuePriority(raw)
	}
	filters := issueBoardFilterView{
		Status:   normalizeIssueStatus(values.Get("status")),
		Priority: priority,
		Category: normalizeIssueCategory(values.Get("category")),
		Assignee: normalizeEmail(values.Get("assignee")),
		Sort:     normalizeIssueBoardSort(values.Get("sort")),
	}
	if filters.Sort == "" {
		filters.Sort = "updated"
	}
	filters.HasActive = filters.Status != "" || filters.Priority != "" || filters.Category != "" || filters.Assignee != "" || filters.Sort != "updated"
	return filters
}

func issuePriorityRank(priority string) int {
	switch normalizeIssuePriority(priority) {
	case issuePriorityUrgent:
		return 4
	case issuePriorityHigh:
		return 3
	case issuePriorityNorm:
		return 2
	case issuePriorityLow:
		return 1
	default:
		return 0
	}
}

func issueStatusRank(status string) int {
	switch normalizeIssueStatus(status) {
	case issueStatusNew:
		return 1
	case issueStatusProgress:
		return 2
	case issueStatusDone:
		return 3
	case issueStatusRejected:
		return 4
	case issueStatusDuplicate:
		return 5
	default:
		return 9
	}
}

func issueOpenCount(items []residentIssue) int {
	count := 0
	for _, item := range items {
		if issueIsOpen(item) {
			count++
		}
	}
	return count
}

func issuePriorityCount(items []residentIssue, priority string) int {
	priority = normalizeIssuePriority(priority)
	count := 0
	for _, item := range items {
		if normalizeIssuePriority(item.Priority) == priority {
			count++
		}
	}
	return count
}

func issueIsOpen(item residentIssue) bool {
	switch normalizeIssueStatus(item.Status) {
	case issueStatusDone, issueStatusRejected, issueStatusDuplicate:
		return false
	default:
		return true
	}
}

func (a *app) visibleIssuesForActor(tenant store.TenantRef, email string, role string) []residentIssue {
	if a == nil {
		return nil
	}
	issues, ok := store.BindIssueRepository(a.issueStore, tenant)
	if !ok {
		return nil
	}
	all := issues.List()
	out := make([]residentIssue, 0, len(all))
	for _, item := range all {
		if a.canViewIssueForActor(tenant, item, email, role) {
			out = append(out, item)
		}
	}
	sortIssues(out)
	return out
}

func (a *app) canViewIssueForActor(tenant store.TenantRef, item residentIssue, email string, role string) bool {
	tenantSlug := normalizeSlug(tenant.Slug)
	email = normalizeEmail(email)
	if tenantSlug == "" || normalizeSlug(item.TenantSlug) != tenantSlug || email == "" {
		return false
	}
	actor := actorFor(email, tenantSlug, role)
	resource := resourceFor(item.TenantSlug)
	if can(actor, capabilityManageIssues, resource) || can(actor, capabilityOversight, resource) {
		return true
	}
	if isServiceProviderRole(role) {
		return issueIsOpen(item) && issueAssignedToActor(item, email)
	}
	if normalizeEmail(item.AuthorEmail) == email {
		return true
	}
	return normalizeIssueLocation(item.LocationType) == issueLocationCommon && a.actorCanSeeCommonIssues(tenant, email, role)
}

func issueAssignedToActor(item residentIssue, email string) bool {
	return normalizeEmail(item.AssigneeEmail) != "" && normalizeEmail(item.AssigneeEmail) == normalizeEmail(email)
}

func (a *app) canDeleteIssueComment(tenant store.TenantRef, issue residentIssue, comment issueComment, email string, role string) bool {
	tenantSlug := tenant.Slug
	email = normalizeEmail(email)
	if email == "" || !a.canViewIssueForActor(tenant, issue, email, role) {
		return false
	}
	actor := actorFor(email, tenantSlug, role)
	resource := resourceFor(issue.TenantSlug)
	if can(actor, capabilityManageIssues, resource) || can(actor, capabilityPlatformAdmin, resource) {
		return true
	}
	return normalizeEmail(comment.AuthorEmail) == email
}

func (a *app) issueCommentTarget(tenant store.TenantRef, commentID string) (residentIssue, issueComment, bool) {
	if a == nil {
		return residentIssue{}, issueComment{}, false
	}
	issues, ok := store.BindIssueRepository(a.issueStore, tenant)
	if !ok {
		return residentIssue{}, issueComment{}, false
	}
	commentID = strings.TrimSpace(commentID)
	if commentID == "" {
		return residentIssue{}, issueComment{}, false
	}
	for _, issue := range issues.List() {
		for _, comment := range issue.Comments {
			if comment.ID == commentID {
				return issue, comment, true
			}
		}
	}
	return residentIssue{}, issueComment{}, false
}

func (a *app) issueViewsForActor(tenant store.TenantRef, items []residentIssue, role string, actorEmail string) []issueView {
	views := issueViewsForActor(tenant.Slug, items, role, actorEmail)
	if a == nil {
		return views
	}
	for i := range views {
		attachments := a.attachmentViewsForEntity(tenant, "issue", views[i].ID, actorEmail, role)
		photoCount := 0
		for _, attachment := range attachments {
			if attachment.IsImage {
				photoCount++
			}
		}
		if len(attachments) > 0 {
			views[i].Attachments = attachments
			views[i].HasAttachments = true
		}
		estimateAttachments := a.attachmentViewsForEntity(tenant, "issue-estimate", views[i].ID, actorEmail, role)
		if len(estimateAttachments) > 0 {
			views[i].EstimateAttachments = estimateAttachments
			views[i].HasEstimateAttachments = true
			views[i].EstimateAttachmentGroup = attachmentGroup{Attachments: estimateAttachments, HasAttachments: true}
			views[i].HasEstimate = true
		}
		views[i].PhotoCount = photoCount
		views[i].HasPhotos = photoCount > 0
		for j := range views[i].Comments {
			comment, found := issueCommentByID(items[i].Comments, views[i].Comments[j].ID)
			if found && a.canDeleteIssueComment(tenant, items[i], comment, actorEmail, role) {
				views[i].Comments[j].CanDelete = true
				views[i].Comments[j].DeleteURL = "/app/anliegen/comment/delete"
			}
			commentAttachments := a.attachmentViewsForEntity(tenant, "issue-comment", views[i].Comments[j].ID, actorEmail, role)
			if len(commentAttachments) == 0 {
				continue
			}
			views[i].Comments[j].Attachments = commentAttachments
			views[i].Comments[j].HasAttachments = true
			for _, attachment := range commentAttachments {
				if attachment.IsImage {
					views[i].PhotoCount++
					views[i].HasPhotos = true
				}
			}
		}
	}
	return views
}

func issueCommentByID(comments []issueComment, id string) (issueComment, bool) {
	id = strings.TrimSpace(id)
	for _, comment := range comments {
		if comment.ID == id {
			return comment, true
		}
	}
	return issueComment{}, false
}

func issueViews(items []residentIssue) []issueView {
	return issueViewsForActor("", items, "", "")
}

func issueViewsForActor(tenantSlug string, items []residentIssue, role string, actorEmail string) []issueView {
	views := make([]issueView, 0, len(items))
	actorEmail = normalizeEmail(actorEmail)
	for _, item := range items {
		actor := actorFor(actorEmail, tenantSlug, role)
		resource := resourceFor(item.TenantSlug)
		canManage := can(actor, capabilityManageIssues, resource)
		readOnly := can(actor, capabilityOversight, resource) && !canManage
		photoCount := 0
		comments := issueCommentViews(item.Comments)
		status := normalizeIssueStatus(item.Status)
		if status == "" {
			status = issueStatusOpen
		}
		priority := normalizeIssuePriority(item.Priority)
		if priority == "" {
			priority = issuePriorityNorm
		}
		isOwner := normalizeEmail(item.AuthorEmail) == actorEmail
		author := strings.TrimSpace(item.AuthorName)
		if author == "" {
			author = item.AuthorEmail
		}
		canResidentAct := !canManage && !readOnly && isOwner
		canServiceAct := isServiceProviderRole(role) && issueAssignedToActor(item, actorEmail) && issueIsOpen(item)
		hasEstimate := item.EstimateAmountCents > 0 || strings.TrimSpace(item.EstimateNote) != ""
		openQuestion, hasOpenQuestion := issueOpenQuestion(item.Comments)
		residentState := "waiting"
		detailURL := "/app/anliegen/" + url.PathEscape(item.ID)
		detailAction := "Ansehen"
		if canManage {
			detailURL = "/app/anliegen/board/" + url.PathEscape(item.ID)
			if status == issueStatusOpen {
				detailAction = "Priorisieren"
			} else {
				detailAction = "Weiter bearbeiten"
			}
		} else if canResidentAct && hasOpenQuestion {
			residentState = "question"
			detailAction = "Antworten"
		} else if canResidentAct && status == issueStatusDone && item.ResolutionConfirmedAt.IsZero() {
			residentState = "resolution"
			detailAction = "Lösung prüfen"
		} else if status == issueStatusDone && !item.ResolutionConfirmedAt.IsZero() {
			residentState = "done"
		}
		nextStep := issueNextStep(status, canManage)
		switch residentState {
		case "question":
			nextStep = "Die Verwaltung braucht Ihre Antwort."
		case "resolution":
			nextStep = "Bitte prüfen Sie die vorgeschlagene Lösung."
		case "done":
			nextStep = "Von Ihnen als erledigt bestätigt."
		}
		views = append(views, issueView{
			ID:                    item.ID,
			Title:                 item.Title,
			Body:                  item.Body,
			Description:           view.TruncateIssueDescription(item.Body),
			Author:                author,
			AuthorEmail:           item.AuthorEmail,
			Category:              item.Category,
			Status:                status,
			StatusClass:           issueStatusClass(status),
			NextStep:              nextStep,
			DetailURL:             detailURL,
			DetailAction:          detailAction,
			ResidentState:         residentState,
			HasOpenQuestion:       hasOpenQuestion,
			OpenQuestion:          issueCommentViewFrom(openQuestion),
			ResolutionConfirmed:   !item.ResolutionConfirmedAt.IsZero(),
			Priority:              priority,
			AssigneeEmail:         item.AssigneeEmail,
			HasAssignee:           item.AssigneeEmail != "",
			Location:              issueLocationLabel(item.LocationType, item.LocationDetail),
			LocationType:          item.LocationType,
			LocationDetail:        item.LocationDetail,
			CreatedAt:             formatLocalDateTime(item.CreatedAt),
			CanComment:            canManage || canResidentAct || canServiceAct,
			CanClose:              canResidentAct && canResidentTransition(status, issueStatusDone),
			CanReopen:             canResidentAct && canResidentTransition(status, issueStatusNew),
			CanServiceUpdate:      canServiceAct,
			ServiceProposal:       item.ServiceProposal,
			HasServiceProposal:    strings.TrimSpace(item.ServiceProposal) != "",
			ServiceAppointment:    formatIssueAppointment(item.ServiceProposedStart, item.ServiceProposedEnd),
			HasServiceAppointment: !item.ServiceProposedStart.IsZero(),
			ServiceStartInput:     issueAppointmentInput(item.ServiceProposedStart),
			ServiceEndInput:       issueAppointmentInput(item.ServiceProposedEnd),
			CanEditEstimate:       canManage || canServiceAct,
			EstimateAmount:        formatIssueEstimateAmount(item.EstimateAmountCents),
			EstimateAmountValue:   formatIssueEstimateInput(item.EstimateAmountCents),
			EstimateNote:          item.EstimateNote,
			HasEstimate:           hasEstimate,
			PhotoCount:            photoCount,
			HasPhotos:             photoCount > 0,
			Comments:              comments,
			HasComments:           len(comments) > 0,
			StatusOptions:         issueSelectOptions(issueStatuses(), status),
			ServiceStatusOptions:  issueSelectOptions(serviceProviderIssueStatuses(), status),
			PriorityOptions:       issueSelectOptions(issuePriorities(), priority),
		})
	}
	return views
}

func issueNextStep(status string, canManage bool) string {
	if canManage {
		switch normalizeIssueStatus(status) {
		case issueStatusOpen:
			return "Als Nächstes: Priorität und Zuständigkeit festlegen."
		case issueStatusProgress:
			return "Als Nächstes: Bearbeitung dokumentieren oder Status aktualisieren."
		case issueStatusAccepted:
			return "Als Nächstes: Bearbeitung dokumentieren oder Status aktualisieren."
		case issueStatusScheduled:
			return "Als Nächstes: Termin prüfen und danach abschließen."
		case issueStatusDone:
			return "Abgeschlossen."
		}
	}
	switch normalizeIssueStatus(status) {
	case issueStatusOpen:
		return "Als Nächstes prüft die Verwaltung Ihre Meldung."
	case issueStatusProgress:
		return "Die Verwaltung bearbeitet das Anliegen."
	case issueStatusAccepted:
		return "Die Verwaltung hat das Anliegen angenommen."
	case issueStatusScheduled:
		return "Bitte beachten Sie den vereinbarten Termin."
	case issueStatusDone:
		return "Abgeschlossen. Bei Bedarf können Sie das Anliegen wieder öffnen."
	default:
		return "Der aktuelle Stand ist oben sichtbar."
	}
}

func issueCommentViews(comments []issueComment) []issueCommentView {
	views := make([]issueCommentView, 0, len(comments))
	sort.SliceStable(comments, func(i, j int) bool {
		return comments[i].CreatedAt.Before(comments[j].CreatedAt)
	})
	for _, comment := range comments {
		author := strings.TrimSpace(comment.AuthorName)
		if author == "" {
			author = comment.AuthorEmail
		}
		view := issueCommentViewFrom(comment)
		view.Author = author
		views = append(views, view)
	}
	return views
}

func issueCommentViewFrom(comment issueComment) issueCommentView {
	author := strings.TrimSpace(comment.AuthorName)
	if author == "" {
		author = comment.AuthorEmail
	}
	kind := normalizeIssueCommentKind(comment.Kind)
	label := ""
	switch kind {
	case issueCommentKindInformation:
		label = "Information der Verwaltung"
	case issueCommentKindQuestion:
		label = "Rückfrage der Verwaltung"
	case issueCommentKindAnswer:
		label = "Antwort"
	}
	return issueCommentView{
		ID:         comment.ID,
		Author:     author,
		Body:       comment.Body,
		Kind:       kind,
		KindLabel:  label,
		IsQuestion: kind == issueCommentKindQuestion,
		IsAnswer:   kind == issueCommentKindAnswer,
		CreatedAt:  formatLocalDateTime(comment.CreatedAt),
	}
}

func issueOpenQuestion(comments []issueComment) (issueComment, bool) {
	sorted := append([]issueComment(nil), comments...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].CreatedAt.Before(sorted[j].CreatedAt)
	})
	var open issueComment
	found := false
	for _, comment := range sorted {
		switch normalizeIssueCommentKind(comment.Kind) {
		case issueCommentKindQuestion:
			open = comment
			found = true
		case issueCommentKindAnswer:
			found = false
		}
	}
	return open, found
}
