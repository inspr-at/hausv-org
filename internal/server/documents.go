package server

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/version"
	"github.com/inspr-at/hausv-org/internal/web"
)

func (a *app) documents(w http.ResponseWriter, r *http.Request, ac authCtx) {
	email, role := ac.email, ac.role
	if denyServiceProviderArea(w, role) {
		return
	}
	canManage := ac.can(capabilityManageDocuments)
	visible := []documentRecord{}
	if ac.repositories.documents != nil {
		visible = a.visibleDocumentsForActor(ac.tenantRef, email, role)
	}
	searchQuery := strings.TrimSpace(r.URL.Query().Get("q"))
	sortMode := selectedDocumentSort(r.URL.Query().Get("sort"))
	documents := sortDocumentsForView(filterDocuments(visible, searchQuery), sortMode)
	documentMsg, documentOK := documentMessage(r.URL.Query().Get("doc"))
	documentCountLabel := fmt.Sprintf("%d Dokumente", len(documents))
	if len(documents) == 1 {
		documentCountLabel = "1 Dokument"
	}
	guide := documentCategoryGuide()
	documentGuide := make([]web.DocumentGuideEntry, 0, len(guide))
	for _, entry := range guide {
		documentGuide = append(documentGuide, web.DocumentGuideEntry{Category: entry.Category, Detail: entry.Detail})
	}
	a.renderDocumentsTempl(w, r, web.DocumentsPageData{
		Portal:             a.documentsPortalContext(ac),
		AssetVersion:       version.AssetVersion(),
		CanManageDocuments: canManage,
		DocumentSections:   a.documentCategorySectionsForActor(ac.tenantRef, email, role, documents, false),
		HasDocuments:       len(documents) > 0,
		HasAnyDocuments:    len(visible) > 0,
		DocumentsEmpty:     emptyState("Noch keine Dokumente", "Sobald die Verwaltung eine Unterlage freigibt, erscheint sie hier – mit Kategorie, Datum und Download."),
		DocumentGuide:      documentGuide,
		DocumentCountLabel: documentCountLabel,
		DocumentMessage:    documentMsg,
		DocumentMessageOK:  documentOK,
		SearchQuery:        searchQuery,
		HasSearchQuery:     searchQuery != "",
		SortOptions:        documentSortOptions(sortMode),
		CategoryOptions:    documentCategoryOptions(""),
		VisibilityOptions:  documentVisibilityOptions(""),
		UnitOptions:        documentUnitOptions(ac.repositories.units.List(), ""),
		MaxDocumentSize:    formatBytes(maxDocumentBytes),
	})
}

func (a *app) documentsPortalContext(ac authCtx) web.PortalPageData {
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
	unreadAnnouncements := 0
	if ac.repositories.announcements != nil && ac.repositories.announcementReads != nil && strings.TrimSpace(ac.email) != "" {
		now := time.Now()
		unreadAnnouncements = unreadAnnouncementCount(ac.repositories.announcements.Visible(now), ac.repositories.announcementReads.LastSeen(ac.email), now)
	}
	openIssues := 0
	if a.issueStore != nil {
		openIssues = issueOpenCount(a.visibleIssuesForActor(ac.tenantRef, ac.email, ac.role))
	}
	return web.PortalPageData{
		SupportView:         supportViewPortalData(ac),
		Title:               "Dokumente · " + houseDisplayName(ac.tenant) + " · " + ac.role,
		TenantSlug:          ac.tenant.Slug,
		HouseName:           houseDisplayName(ac.tenant),
		Address:             ac.tenant.Address,
		MapURL:              tenantMapURL(ac.tenant.Address),
		HeroImageURL:        ac.tenant.HeroImageURL,
		BrandIcon:           ac.tenant.BrandIcon,
		BrandMarkSVG:        tenantBrandMarkSVG(ac.tenant.BrandIcon),
		Map:                 portalMapForTenant(ac.tenant),
		DisplayName:         profile.DisplayName(),
		Initials:            profile.Initials(),
		Role:                ac.role,
		DisplayVersion:      version.DisplayVersion(version.Version),
		ActivePage:          "documents",
		Modules:             web.PortalModules{Energy: modules.Energy, Announcements: modules.Announcements, Events: modules.Events, Contacts: modules.Contacts, Documents: modules.Documents, Issues: modules.Issues, Votes: modules.Votes, Parking: modules.Parking, Handovers: modules.Handovers, Users: modules.Users, Audit: modules.Audit, Help: modules.Help},
		CanUseResidentAreas: roleCanUseResidentAreas(ac.role),
		CanViewEnergy:       modules.Energy && a.canViewEnergy(ac),
		CanManageIssues:     ac.can(capabilityManageIssues),
		CanSeeParking:       modules.Parking && (ac.can(capabilityPlatformAdmin) || profile.HasPermission(permissionParking)),
		CanManageHandovers:  modules.Handovers && canManageHandovers(ac.actor(), ac.resource()),
		CanManageUsers:      modules.Users && ac.can(capabilityManageUsers),
		CanViewAudit:        modules.Audit && canViewAudit(ac.actor(), ac.resource()),
		Issues:              make([]issueView, openIssues),
		UnreadAnnouncements: unreadAnnouncements,
		Contexts:            portalContexts,
		ReleaseNotes:        version.Notes(),
	}
}

func (a *app) renderDocumentsTempl(w http.ResponseWriter, r *http.Request, data web.DocumentsPageData) {
	var rendered bytes.Buffer
	if err := web.DocumentsPage(data).Render(r.Context(), &rendered); err != nil {
		logError("templ documents render failed", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, prefixTenantHTMLPaths(rendered.String(), data.Portal.TenantSlug))
}

func (a *app) uploadDocument(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	if a.documentStore == nil {
		http.Redirect(w, r, "/app/dokumente?doc=invalid", http.StatusSeeOther)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxDocumentFormBytes)
	if err := r.ParseMultipartForm(maxDocumentBytes); err != nil {
		http.Redirect(w, r, "/app/dokumente?doc=invalid", http.StatusSeeOther)
		return
	}
	header := documentFileHeader(r)
	if header == nil {
		http.Redirect(w, r, "/app/dokumente?doc=invalid", http.StatusSeeOther)
		return
	}
	created, err := ac.repositories.documents.Create(documentRecord{
		TenantSlug: tenant.Slug,
		Title:      strings.TrimSpace(r.FormValue("title")),
		Category:   normalizeDocumentCategory(r.FormValue("category")),
		Visibility: normalizeDocumentVisibility(r.FormValue("visibility")),
		UnitID:     normalizeUnitID(r.FormValue("unit_id")),
		UploadedBy: email,
	}, uploadedFileFromHeader(header), time.Now())
	if err != nil {
		logError("document upload failed", err, "tenant", tenant.Slug, "actor", redactedEmail(email))
		http.Redirect(w, r, "/app/dokumente?doc=invalid", http.StatusSeeOther)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: email,
		ActorRole:  role,
		Action:     auditActionDocumentUpload,
		TargetType: "document",
		TargetID:   created.ID,
		Summary:    "Dokument hochgeladen",
		Details: map[string]string{
			"title":        created.Title,
			"category":     created.Category,
			"visibility":   documentVisibilityLabel(created.Visibility),
			"unit":         documentUnitAuditLabel(created.UnitID),
			"size":         formatBytes(created.Size),
			"content_type": created.ContentType,
		},
	})
	http.Redirect(w, r, "/app/dokumente?doc=uploaded#document-"+url.PathEscape(created.ID), http.StatusSeeOther)
}

func (a *app) replaceDocument(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	if a.documentStore == nil {
		http.Redirect(w, r, "/app/dokumente?doc=invalid", http.StatusSeeOther)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxDocumentFormBytes)
	if err := r.ParseMultipartForm(maxDocumentBytes); err != nil {
		http.Redirect(w, r, "/app/dokumente?doc=invalid", http.StatusSeeOther)
		return
	}
	header := documentFileHeader(r)
	if header == nil {
		http.Redirect(w, r, "/app/dokumente?doc=invalid", http.StatusSeeOther)
		return
	}
	replacement, replaced, err := ac.repositories.documents.Replace(strings.TrimSpace(r.FormValue("id")), email, uploadedFileFromHeader(header), time.Now())
	if err != nil {
		logError("document replace failed", err, "tenant", tenant.Slug, "actor", redactedEmail(email))
		http.Redirect(w, r, "/app/dokumente?doc=invalid", http.StatusSeeOther)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: email,
		ActorRole:  role,
		Action:     auditActionDocumentReplace,
		TargetType: "document",
		TargetID:   replacement.ID,
		Summary:    "Dokument ersetzt",
		Details: map[string]string{
			"title":        replacement.Title,
			"category":     replacement.Category,
			"visibility":   documentVisibilityLabel(replacement.Visibility),
			"version":      documentVersionLabel(replacement.Version),
			"previous":     documentVersionLabel(replaced.Version),
			"previous_id":  replaced.ID,
			"content_type": replacement.ContentType,
			"size":         formatBytes(replacement.Size),
		},
	})
	http.Redirect(w, r, "/app/dokumente?doc=replaced#document-"+url.PathEscape(replacement.ID), http.StatusSeeOther)
}

func (a *app) downloadDocument(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	if a.documentStore == nil {
		http.NotFound(w, r)
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	item, found := ac.repositories.documents.Get(id)
	if !found {
		http.NotFound(w, r)
		return
	}
	if !a.canViewDocument(ac.tenantRef, item, email, role) {
		http.Error(w, "Dieses Dokument ist für diesen Zugang nicht freigegeben.", http.StatusForbidden)
		return
	}
	path, ok := ac.repositories.documents.FilePath(item)
	if !ok {
		http.NotFound(w, r)
		return
	}
	file, err := os.Open(path)
	if err != nil {
		logError("document file open failed", err, "tenant", tenant.Slug, "document_id", item.ID)
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": item.Filename}))
	if item.ContentType != "" {
		w.Header().Set("Content-Type", item.ContentType)
	}
	a.recordAuthenticatedReadAudit(ac, auditEvent{
		TenantSlug: tenant.Slug,
		Action:     auditActionDocumentDownload,
		TargetType: "document",
		TargetID:   item.ID,
		Summary:    "Dokument heruntergeladen",
		Details: map[string]string{
			"title":      item.Title,
			"category":   item.Category,
			"visibility": documentVisibilityLabel(item.Visibility),
			"unit":       documentUnitAuditLabel(item.UnitID),
			"version":    documentVersionLabel(item.Version),
		},
	})
	http.ServeContent(w, r, item.Filename, item.UploadedAt, file)
}

func (a *app) previewDocument(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	if a.documentStore == nil {
		http.NotFound(w, r)
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	item, found := ac.repositories.documents.Get(id)
	if !found {
		http.NotFound(w, r)
		return
	}
	if !a.canViewDocument(ac.tenantRef, item, email, role) {
		http.Error(w, "Dieses Dokument ist für diesen Zugang nicht freigegeben.", http.StatusForbidden)
		return
	}
	if !documentCanPreview(item.ContentType) {
		http.NotFound(w, r)
		return
	}
	path, ok := ac.repositories.documents.FilePath(item)
	if !ok {
		http.NotFound(w, r)
		return
	}
	file, err := os.Open(path)
	if err != nil {
		logError("document preview open failed", err, "tenant", tenant.Slug, "document_id", item.ID)
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; img-src 'self' data:; style-src 'unsafe-inline'; sandbox")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": item.Filename}))
	if item.ContentType != "" {
		w.Header().Set("Content-Type", item.ContentType)
	}
	http.ServeContent(w, r, item.Filename, item.UploadedAt, file)
}

func (a *app) serveAttachment(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	if a.attachmentStore == nil {
		http.NotFound(w, r)
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	variant := strings.ToLower(strings.TrimSpace(r.PathValue("variant")))
	if variant != "" && variant != "preview" && variant != "thumb" && variant != "thumbnail" {
		http.NotFound(w, r)
		return
	}
	item, found := ac.repositories.attachments.Get(id)
	if !found {
		http.NotFound(w, r)
		return
	}
	if !a.canViewAttachment(ac.tenantRef, item, email, role) {
		http.Error(w, "Dieser Anhang ist für diesen Zugang nicht freigegeben.", http.StatusForbidden)
		return
	}
	path, contentType, _, ok := ac.repositories.attachments.FilePath(item, variant)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if _, err := os.Stat(path); err != nil {
		logError("attachment file open failed", err, "tenant", tenant.Slug, "entity_type", item.EntityType, "attachment_id", item.ID)
		http.NotFound(w, r)
		return
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; img-src 'self' data:; style-src 'unsafe-inline'; sandbox")
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	disposition := attachmentDisposition(contentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": item.Filename}))
	w.Header().Set("Cache-Control", "private, max-age=300")
	if (variant == "" || variant == "preview") && strings.TrimSpace(r.Header.Get("Range")) == "" {
		access := "Datei"
		if variant == "preview" {
			access = "Vorschau"
		}
		a.recordAuthenticatedReadAudit(ac, auditEvent{
			TenantSlug: tenant.Slug,
			Action:     auditActionAttachmentView,
			TargetType: "attachment",
			TargetID:   item.ID,
			Summary:    "Anhang angezeigt",
			Details: map[string]string{
				"entity_type":  normalizeAttachmentEntity(item.EntityType),
				"entity_id":    item.EntityID,
				"access":       access,
				"content_type": contentType,
			},
		})
	}
	http.ServeFile(w, r, path)
}

func (a *app) deleteAttachment(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	if a.attachmentStore == nil {
		http.Redirect(w, r, "/app/anliegen?issue=missing", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	id := strings.TrimSpace(r.FormValue("id"))
	item, found := ac.repositories.attachments.Get(id)
	if !found {
		http.Redirect(w, r, redirectAfterAttachmentChange(r, "/app/anliegen?issue=missing"), http.StatusSeeOther)
		return
	}
	if !a.canDeleteAttachment(ac.tenantRef, item, email, role) {
		http.Error(w, "Dieser Anhang kann nur von Verwaltung oder Ersteller entfernt werden.", http.StatusForbidden)
		return
	}
	if normalizeAttachmentEntity(item.EntityType) == "handover" && ac.repositories.handovers != nil {
		handover, found := ac.repositories.handovers.Get(item.EntityID)
		if !found || !handoverCanChangeFiles(handover) {
			http.Error(w, "Bestätigte oder abgelegte Protokolle sind unveränderlich.", http.StatusConflict)
			return
		}
	}
	if _, removed, err := ac.repositories.attachments.Delete(id, time.Now()); err != nil {
		logError("attachment delete failed", err, "tenant", tenant.Slug, "attachment_id", id)
		http.Redirect(w, r, redirectAfterAttachmentChange(r, "/app/anliegen?issue=error"), http.StatusSeeOther)
		return
	} else if !removed {
		http.Redirect(w, r, redirectAfterAttachmentChange(r, "/app/anliegen?issue=missing"), http.StatusSeeOther)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: email,
		ActorRole:  role,
		Action:     auditActionAttachmentDelete,
		TargetType: "attachment",
		TargetID:   item.ID,
		Summary:    "Anhang entfernt",
		Details: map[string]string{
			"entity_type": normalizeAttachmentEntity(item.EntityType),
			"entity_id":   item.EntityID,
		},
	})
	http.Redirect(w, r, redirectAfterAttachmentChange(r, "/app/anliegen?issue=updated"), http.StatusSeeOther)
}

func attachmentDisposition(contentType string) string {
	contentType = strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	if isImageContentType(contentType) || contentType == "application/pdf" {
		return "inline"
	}
	return "attachment"
}

func documentFileHeader(r *http.Request) *multipart.FileHeader {
	if r == nil || r.MultipartForm == nil {
		return nil
	}
	for _, name := range []string{"document", "file"} {
		files := r.MultipartForm.File[name]
		if len(files) > 0 {
			return files[0]
		}
	}
	return nil
}

func (a *app) visibleDocumentsForActor(tenant store.TenantRef, email string, role string) []documentRecord {
	if a == nil {
		return nil
	}
	documents, ok := store.BindDocumentRepository(a.documentStore, tenant)
	if !ok {
		return nil
	}
	all := documents.ListCurrent()
	out := make([]documentRecord, 0, len(all))
	for _, item := range all {
		if a.canViewDocument(tenant, item, email, role) {
			out = append(out, item)
		}
	}
	return out
}

func (a *app) canViewDocument(tenant store.TenantRef, item documentRecord, email string, role string) bool {
	tenantSlug := tenant.Slug
	if normalizeSlug(item.TenantSlug) != normalizeSlug(tenantSlug) {
		return false
	}
	if isServiceProviderRole(role) {
		return false
	}
	if can(actorFor(email, tenantSlug, role), capabilityManageDocuments, resourceFor(item.TenantSlug)) {
		return true
	}
	switch normalizeDocumentVisibility(item.Visibility) {
	case documentVisibilityAllResidents:
		return true
	case documentVisibilityOwnersOnly:
		return a.isDocumentOwner(tenant, email, role, item.UnitID)
	case documentVisibilityManagerOnly:
		return false
	default:
		return false
	}
}

func (a *app) isDocumentOwner(tenant store.TenantRef, email string, role string, unitID string) bool {
	tenantSlug := normalizeSlug(tenant.Slug)
	email = normalizeEmail(email)
	unitID = normalizeUnitID(unitID)
	if tenantSlug == "" || email == "" {
		return false
	}
	if a != nil && a.unitStore != nil {
		units, _ := store.BindUnitRepository(a.unitStore, tenant)
		if units != nil && unitID != "" {
			members := units.MembersForUnit(unitID)
			return members.Found && emailListContains(members.Owners, email)
		}
		if units != nil {
			for _, membership := range units.UnitsForEmail(email) {
				if membership.Relation == roleOwner {
					return true
				}
			}
		}
	}
	return normalizeRole(role) == roleOwner
}

func documentMessage(status string) (string, bool) {
	switch status {
	case "uploaded":
		return "Dokument hochgeladen.", true
	case "replaced":
		return "Neue Version gespeichert.", true
	case "invoice-imported":
		return "E-Rechnung geprüft und geschützt abgelegt.", true
	case "invalid":
		return "Bitte Titel, Kategorie, Sichtbarkeit und Datei prüfen. Erlaubt sind PDF, JPG, PNG oder WebP bis 20 MB.", false
	default:
		return "", false
	}
}

func attachmentFormHeaders(r *http.Request, maxCount int, names ...string) ([]*multipart.FileHeader, error) {
	if r.MultipartForm == nil {
		return nil, nil
	}
	out := []*multipart.FileHeader{}
	for _, name := range names {
		for _, header := range r.MultipartForm.File[name] {
			if header == nil || strings.TrimSpace(header.Filename) == "" || header.Size == 0 {
				continue
			}
			out = append(out, header)
			if maxCount > 0 && len(out) > maxCount {
				return nil, fmt.Errorf("too many attachments")
			}
		}
	}
	return out, nil
}

func (a *app) canViewAttachment(tenant store.TenantRef, item attachmentRecord, email string, role string) bool {
	tenantSlug := normalizeSlug(tenant.Slug)
	if tenantSlug == "" || normalizeSlug(item.TenantSlug) != tenantSlug || normalizeEmail(email) == "" {
		return false
	}
	entityType := normalizeAttachmentEntity(item.EntityType)
	actor := actorFor(email, tenantSlug, role)
	resource := resourceFor(item.TenantSlug)
	if isServiceProviderRole(role) && entityType != "issue" && entityType != "issue-comment" && entityType != "issue-estimate" {
		return false
	}
	switch entityType {
	case "issue", "issue-estimate":
		issues, ok := store.BindIssueRepository(a.issueStore, tenant)
		if !ok {
			return false
		}
		issue, found := issues.Get(item.EntityID)
		return found && a.canViewIssueForActor(tenant, issue, email, role)
	case "issue-comment":
		issue, _, found := a.issueCommentTarget(tenant, item.EntityID)
		return found && a.canViewIssueForActor(tenant, issue, email, role)
	case "document":
		documents, ok := store.BindDocumentRepository(a.documentStore, tenant)
		if !ok {
			return false
		}
		doc, found := documents.Get(item.EntityID)
		return found && a.canViewDocument(tenant, doc, email, role)
	case "announcement", "event", "building":
		return true
	case "ballot":
		return can(actor, capabilityManageVotes, resource) || can(actor, capabilityVote, resource) || can(actor, capabilityOversight, resource)
	case "handover":
		return canManageHandovers(actorFor(email, tenantSlug, role), resourceFor(item.TenantSlug))
	case "parking":
		return can(actor, capabilityManageParking, resource) || can(actor, capabilityPlatformAdmin, resource) || a.profileForTenant(email, tenantSlug).HasPermission(permissionParking)
	default:
		return false
	}
}

func (a *app) canDeleteAttachment(tenant store.TenantRef, item attachmentRecord, email string, role string) bool {
	tenantSlug := normalizeSlug(tenant.Slug)
	email = normalizeEmail(email)
	if tenantSlug == "" || normalizeSlug(item.TenantSlug) != tenantSlug || email == "" {
		return false
	}
	actor := actorFor(email, tenantSlug, role)
	resource := resourceFor(item.TenantSlug)
	if can(actor, capabilityPlatformAdmin, resource) || normalizeEmail(item.UploadedBy) == email {
		return true
	}
	switch normalizeAttachmentEntity(item.EntityType) {
	case "issue", "issue-estimate":
		if can(actor, capabilityManageIssues, resource) {
			return true
		}
		issues, ok := store.BindIssueRepository(a.issueStore, tenant)
		if !ok {
			return false
		}
		issue, found := issues.Get(item.EntityID)
		return found && normalizeEmail(issue.AuthorEmail) == email
	case "issue-comment":
		if can(actor, capabilityManageIssues, resource) {
			return true
		}
		issue, comment, found := a.issueCommentTarget(tenant, item.EntityID)
		return found && (normalizeEmail(issue.AuthorEmail) == email || normalizeEmail(comment.AuthorEmail) == email)
	case "document":
		return can(actor, capabilityManageDocuments, resource)
	case "announcement":
		return canManageAnnouncements(actorFor(email, tenantSlug, role), resourceFor(item.TenantSlug))
	case "event":
		return canManageEvents(actorFor(email, tenantSlug, role), resourceFor(item.TenantSlug))
	case "ballot":
		return can(actor, capabilityManageVotes, resource)
	case "handover":
		return canManageHandovers(actorFor(email, tenantSlug, role), resourceFor(item.TenantSlug))
	case "parking":
		return can(actor, capabilityManageParking, resource)
	case "building":
		return can(actor, capabilityManageBuilding, resource)
	default:
		return false
	}
}

func (a *app) attachmentViewsForEntity(tenant store.TenantRef, entityType string, entityID string, actorEmail string, role string) []attachmentView {
	if a == nil {
		return nil
	}
	attachments, ok := store.BindAttachmentRepository(a.attachmentStore, tenant)
	if !ok {
		return nil
	}
	records := attachments.ListEntity(entityType, entityID)
	views := make([]attachmentView, 0, len(records))
	for _, record := range records {
		views = append(views, attachmentViewFromRecord(record, a.canDeleteAttachment(tenant, record, actorEmail, role)))
	}
	return views
}

func documentViews(items []documentRecord) []documentView {
	views := make([]documentView, 0, len(items))
	for _, item := range items {
		views = append(views, documentViewFrom(item))
	}
	return views
}

func (a *app) documentViewsForActor(tenant store.TenantRef, email string, role string, items []documentRecord) []documentView {
	views := make([]documentView, 0, len(items))
	if a == nil {
		return views
	}
	documents, _ := store.BindDocumentRepository(a.documentStore, tenant)
	for _, item := range items {
		view := documentViewFrom(item)
		if a != nil && documents != nil {
			for _, version := range documents.Versions(item.SeriesID) {
				if version.Current || !a.canViewDocument(tenant, version, email, role) {
					continue
				}
				view.Versions = append(view.Versions, documentVersionView{
					ID:          version.ID,
					Version:     documentVersionLabel(version.Version),
					Filename:    version.Filename,
					Size:        formatBytes(version.Size),
					UploadedAt:  formatLocalDateTime(version.UploadedAt),
					DownloadURL: "/app/dokumente/" + url.PathEscape(version.ID) + "/download",
				})
			}
		}
		view.HasVersions = len(view.Versions) > 0
		views = append(views, view)
	}
	return views
}

func (a *app) documentCategorySectionsForActor(tenant store.TenantRef, email string, role string, items []documentRecord, includeEmpty bool) []documentCategoryView {
	byCategory := map[string][]documentRecord{}
	for _, item := range items {
		byCategory[item.Category] = append(byCategory[item.Category], item)
	}
	sections := []documentCategoryView{}
	for _, category := range documentCategories() {
		docs := byCategory[category]
		if len(docs) == 0 && !includeEmpty {
			continue
		}
		sections = append(sections, documentCategoryView{
			Category:     category,
			Documents:    a.documentViewsForActor(tenant, email, role, docs),
			HasDocuments: len(docs) > 0,
			EmptyMessage: "Keine passenden Dokumente in dieser Kategorie.",
		})
	}
	return sections
}

// documentGuideEntry erklärt eine Kategorie der Hausablage. Die leere Ablage
// ist der Normalfall eines neuen Hauses: dort erklärt der Leitfaden, was hier
// erwartet wird, statt nur festzustellen, dass nichts da ist.
type documentGuideEntry struct {
	Category string
	Detail   string
}

func documentCategoryGuide() []documentGuideEntry {
	categories := documentCategories()
	guide := make([]documentGuideEntry, 0, len(categories))
	for _, category := range categories {
		guide = append(guide, documentGuideEntry{Category: category, Detail: documentCategoryDetail(category)})
	}
	return guide
}

func documentCategoryDetail(category string) string {
	switch category {
	case documentCategoryProtocol:
		return "Beschlüsse und Mitschriften der Eigentümerversammlungen."
	case documentCategoryBilling:
		return "Jahresabrechnung, Wirtschaftsplan und Betriebskosten."
	case documentCategoryRules:
		return "Regeln für das Zusammenleben im Haus."
	case documentCategoryContract:
		return "Vereinbarungen mit Dienstleistern und Versorgern."
	case documentCategoryPlan:
		return "Grundrisse, Leitungspläne und technische Zeichnungen."
	default:
		return "Unterlagen, die in keine der übrigen Kategorien passen."
	}
}

func documentCategorySections(items []documentRecord, includeEmpty bool) []documentCategoryView {
	byCategory := map[string][]documentRecord{}
	for _, item := range items {
		byCategory[item.Category] = append(byCategory[item.Category], item)
	}
	sections := []documentCategoryView{}
	for _, category := range documentCategories() {
		docs := byCategory[category]
		if len(docs) == 0 && !includeEmpty {
			continue
		}
		sections = append(sections, documentCategoryView{
			Category:     category,
			Documents:    documentViews(docs),
			HasDocuments: len(docs) > 0,
			EmptyMessage: "Keine passenden Dokumente in dieser Kategorie.",
		})
	}
	return sections
}
