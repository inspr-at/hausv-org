package server

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/pdf"
	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/version"
	"github.com/inspr-at/hausv-org/internal/view"
	"github.com/inspr-at/hausv-org/internal/web"
)

// The audit vocabulary lives in the store (it is what gets persisted and
// validated); these alias back so handover.go reads unchanged.
const (
	auditActionHandoverCreate  = store.AuditActionHandoverCreate
	auditActionHandoverConfirm = store.AuditActionHandoverConfirm
	auditActionHandoverFile    = store.AuditActionHandoverFile
)

const (
	handoverStatusDraft     = "Entwurf"
	handoverStatusPending   = "Wartet auf Bestätigung"
	handoverStatusConfirmed = "Bestätigt"
	handoverStatusFiled     = "Abgelegt"
)

type handoverView struct {
	ID                 string
	Title              string
	Type               string
	UnitLabel          string
	HasUnit            bool
	ScheduledAt        string
	HasScheduledAt     bool
	Status             string
	StatusClass        string
	NextStep           string
	NextStepDetail     string
	ConfirmedCount     int
	ConfirmationCount  int
	IsPending          bool
	IsReady            bool
	IsFiled            bool
	CanFile            bool
	CanChangeFiles     bool
	CanManageDocuments bool
	Outgoing           string
	Incoming           string
	Rooms              []handoverRoom
	HasRooms           bool
	Meters             []handoverMeter
	HasMeters          bool
	Keys               []handoverKey
	HasKeys            bool
	Notes              string
	HasNotes           bool
	Confirmations      []handoverConfirmationView
	HasConfirmations   bool
	Attachments        []attachmentView
	HasAttachments     bool
	AttachmentGroup    attachmentGroup
	ProtocolURL        string
	FileURL            string
	FiledDocumentID    string
	HasFiledDocument   bool
	FiledDocumentURL   string
	CreatedAt          string
	UpdatedAt          string
}

type handoverSectionView struct {
	Title       string
	Description string
	Count       int
	Items       []handoverView
	HasItems    bool
	Open        bool
}

type handoverConfirmationView struct {
	Role         string
	Name         string
	Email        string
	Status       string
	StatusClass  string
	ConfirmedAt  string
	HasConfirmed bool
}

func handoverUnitOptions(units []unit, selected string) []selectOption {
	selected = normalizeUnitID(selected)
	options := []selectOption{}
	for _, item := range units {
		id := normalizeUnitID(item.ID)
		label := strings.TrimSpace(item.Label)
		if id == "" || label == "" {
			continue
		}
		options = append(options, selectOption{Value: id, Label: label, Selected: selected == id})
	}
	return options
}

func (a *app) handovers(w http.ResponseWriter, r *http.Request, ac authCtx) {
	email, role := ac.email, ac.role
	if !canManageHandovers(ac.actor(), ac.resource()) {
		http.Error(w, "Übergabeprotokolle sind der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	items := []handoverRecord{}
	if ac.repositories.handovers != nil {
		items = ac.repositories.handovers.List()
	}
	msg, okMsg := handoverMessage(r.URL.Query().Get("handover"))
	views := a.handoverViewsForActor(ac.tenantRef, email, role, items)
	sections := handoverSections(views)
	a.renderHandoversTempl(w, r, web.HandoversPageData{
		Portal:            a.handoverPortalContext(ac),
		AssetVersion:      version.AssetVersion(),
		NowInput:          formatLocalDateTimeInput(time.Now()),
		Message:           msg,
		MessageOK:         okMsg,
		HasHandovers:      len(items) > 0,
		CanManageBuilding: ac.can(capabilityManageBuilding),
		OpenCount:         sections[0].Count,
		ReadyCount:        sections[1].Count,
		FiledCount:        sections[2].Count,
		Sections:          handoverTemplSections(sections),
		Empty:             emptyState("Noch keine Übergaben", "Neue Nutzerwechsel werden hier mit Räumen, Zählern, Schlüsseln, Fotos und Bestätigung dokumentiert."),
		UnitOptions:       handoverUnitOptions(ac.repositories.units.List(), ""),
	})
}

func (a *app) handoverPortalContext(ac authCtx) web.PortalPageData {
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
		Title:               "Übergaben · " + houseDisplayName(ac.tenant) + " · " + ac.role,
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
		ActivePage:          "handovers",
		Modules:             web.PortalModules{Energy: modules.Energy, Announcements: modules.Announcements, Events: modules.Events, Contacts: modules.Contacts, Documents: modules.Documents, Issues: modules.Issues, Votes: modules.Votes, Parking: modules.Parking, Handovers: modules.Handovers, Users: modules.Users, Audit: modules.Audit, Help: modules.Help},
		CanUseResidentAreas: roleCanUseResidentAreas(ac.role),
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

func (a *app) renderHandoversTempl(w http.ResponseWriter, r *http.Request, data web.HandoversPageData) {
	var rendered bytes.Buffer
	if err := web.HandoversPage(data).Render(r.Context(), &rendered); err != nil {
		logError("templ handovers render failed", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, prefixTenantHTMLPaths(rendered.String(), data.Portal.TenantSlug))
}

func handoverTemplSections(sections []handoverSectionView) []web.HandoverSection {
	result := make([]web.HandoverSection, 0, len(sections))
	for _, section := range sections {
		items := make([]web.HandoverItem, 0, len(section.Items))
		for _, item := range section.Items {
			items = append(items, handoverTemplItem(item))
		}
		result = append(result, web.HandoverSection{Title: section.Title, Description: section.Description, Items: items, Open: section.Open})
	}
	return result
}

func handoverTemplItem(item handoverView) web.HandoverItem {
	rooms := make([]web.HandoverRoom, 0, len(item.Rooms))
	for _, room := range item.Rooms {
		rooms = append(rooms, web.HandoverRoom{Name: room.Name, Condition: room.Condition, Defects: room.Defects})
	}
	meters := make([]web.HandoverMeter, 0, len(item.Meters))
	for _, meter := range item.Meters {
		meters = append(meters, web.HandoverMeter{Label: meter.Label, Value: meter.Value, Unit: meter.Unit})
	}
	keys := make([]web.HandoverKey, 0, len(item.Keys))
	for _, key := range item.Keys {
		keys = append(keys, web.HandoverKey{Label: key.Label, Count: key.Count})
	}
	confirmations := make([]web.HandoverConfirmation, 0, len(item.Confirmations))
	for _, confirmation := range item.Confirmations {
		confirmations = append(confirmations, web.HandoverConfirmation{
			Role: confirmation.Role, Name: confirmation.Name, Email: confirmation.Email,
			Status: confirmation.Status, StatusClass: confirmation.StatusClass,
			ConfirmedAt: confirmation.ConfirmedAt, HasConfirmed: confirmation.HasConfirmed,
		})
	}
	return web.HandoverItem{
		ID: item.ID, Title: item.Title, Type: item.Type, UnitLabel: item.UnitLabel,
		ScheduledAt: item.ScheduledAt, Status: item.Status, StatusClass: item.StatusClass,
		NextStep: item.NextStep, NextStepDetail: item.NextStepDetail,
		Outgoing: item.Outgoing, Incoming: item.Incoming, Notes: item.Notes,
		ProtocolURL: item.ProtocolURL, FileURL: item.FileURL, FiledDocumentURL: item.FiledDocumentURL,
		CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
		ConfirmedCount: item.ConfirmedCount, ConfirmationCount: item.ConfirmationCount,
		HasUnit: item.HasUnit, HasScheduledAt: item.HasScheduledAt, IsReady: item.IsReady, IsFiled: item.IsFiled,
		CanFile: item.CanFile, CanChangeFiles: item.CanChangeFiles, CanManageDocuments: item.CanManageDocuments,
		HasFiledDocument: item.HasFiledDocument, Rooms: rooms, Meters: meters, Keys: keys,
		Confirmations: confirmations, Attachments: item.Attachments,
	}
}

func handoverMessage(status string) (string, bool) {
	switch status {
	case "created":
		return "Übergabeprotokoll angelegt. Bestätigungslinks wurden vorbereitet und, falls Mail aktiv ist, versendet.", true
	case "confirmed":
		return "Bestätigung gespeichert.", true
	case "filed":
		return "Protokoll im Dokumentenbereich abgelegt.", true
	case "attachments":
		return "Fotos und Dateien ergänzt.", true
	case "locked":
		return "Das Protokoll ist nach der ersten Bestätigung unveränderlich.", false
	case "pending":
		return "Vor der Ablage fehlen noch Bestätigungen.", false
	case "invalid":
		return "Bitte Titel, Einheit und mindestens einen Protokollpunkt prüfen.", false
	case "missing":
		return "Dieses Übergabeprotokoll wurde nicht gefunden.", false
	case "error":
		return "Das Übergabeprotokoll konnte nicht gespeichert werden.", false
	default:
		return "", false
	}
}

func (a *app) addHandoverAttachments(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email := ac.tenant, ac.email
	if !canManageHandovers(ac.actor(), ac.resource()) {
		http.Error(w, "Übergabeprotokolle sind der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if err := parseMaybeMultipartForm(w, r, maxIssueAttachmentFormBytes, maxAttachmentBytes); err != nil {
		http.Redirect(w, r, "/app/uebergaben?handover=invalid", http.StatusSeeOther)
		return
	}
	id := strings.TrimSpace(r.FormValue("id"))
	item, found := ac.repositories.handovers.Get(id)
	if !found {
		http.Redirect(w, r, "/app/uebergaben?handover=missing", http.StatusSeeOther)
		return
	}
	if !handoverCanChangeFiles(item) {
		http.Redirect(w, r, "/app/uebergaben?handover=locked#handover-"+url.PathEscape(id), http.StatusSeeOther)
		return
	}
	headers, err := attachmentFormHeaders(r, maxIssueAttachmentCount, "attachments")
	if err != nil || len(headers) == 0 || ac.repositories.attachments == nil {
		http.Redirect(w, r, "/app/uebergaben?handover=invalid#handover-"+url.PathEscape(id), http.StatusSeeOther)
		return
	}
	if len(ac.repositories.attachments.ListEntity("handover", id))+len(headers) > maxIssueAttachmentCount {
		http.Redirect(w, r, "/app/uebergaben?handover=invalid#handover-"+url.PathEscape(id), http.StatusSeeOther)
		return
	}
	if _, err := ac.repositories.attachments.CreateUploaded("handover", id, email, uploadedFilesFromHeaders(headers), time.Now()); err != nil {
		logHandoverError("attachments", tenant.Slug, id, err)
		http.Redirect(w, r, "/app/uebergaben?handover=error#handover-"+url.PathEscape(id), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/app/uebergaben?handover=attachments#handover-"+url.PathEscape(id), http.StatusSeeOther)
}

func (a *app) createHandover(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	if !canManageHandovers(ac.actor(), ac.resource()) {
		http.Error(w, "Übergabeprotokolle sind der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if ac.repositories.handovers == nil {
		http.Redirect(w, r, "/app/uebergaben?handover=error", http.StatusSeeOther)
		return
	}
	if err := parseMaybeMultipartForm(w, r, maxIssueAttachmentFormBytes, maxAttachmentBytes); err != nil {
		http.Redirect(w, r, "/app/uebergaben?handover=invalid", http.StatusSeeOther)
		return
	}
	now := time.Now()
	item, deliveries, err := handoverFromRequest(r, tenant.Slug, email, now)
	if err != nil {
		http.Redirect(w, r, "/app/uebergaben?handover=invalid", http.StatusSeeOther)
		return
	}
	attachmentHeaders, err := attachmentFormHeaders(r, maxIssueAttachmentCount, "attachments", "photos", "photo")
	if err != nil {
		http.Redirect(w, r, "/app/uebergaben?handover=invalid", http.StatusSeeOther)
		return
	}
	var uploaded []attachmentRecord
	if len(attachmentHeaders) > 0 {
		if ac.repositories.attachments == nil {
			http.Redirect(w, r, "/app/uebergaben?handover=invalid", http.StatusSeeOther)
			return
		}
		uploaded, err = ac.repositories.attachments.CreateUploaded("handover", item.ID, email, uploadedFilesFromHeaders(attachmentHeaders), now)
		if err != nil {
			http.Redirect(w, r, "/app/uebergaben?handover=invalid", http.StatusSeeOther)
			return
		}
	}
	created, err := ac.repositories.handovers.Create(item)
	if err != nil {
		for _, attachment := range uploaded {
			_, _, _ = ac.repositories.attachments.Delete(attachment.ID, time.Now())
		}
		logHandoverError("create", tenant.Slug, item.ID, err)
		http.Redirect(w, r, "/app/uebergaben?handover=error", http.StatusSeeOther)
		return
	}
	a.notifyHandoverParticipants(r, tenant, created, deliveries)
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: email,
		ActorRole:  role,
		Action:     auditActionHandoverCreate,
		TargetType: "handover",
		TargetID:   created.ID,
		Summary:    "Übergabeprotokoll angelegt",
		Details: map[string]string{
			"title":      created.Title,
			"type":       created.HandoverType,
			"unit":       documentUnitAuditLabel(created.UnitID),
			"recipients": strconv.Itoa(len(deliveries)),
			"file_count": strconv.Itoa(len(uploaded)),
		},
	})
	http.Redirect(w, r, "/app/uebergaben?handover=created#handover-"+url.PathEscape(created.ID), http.StatusSeeOther)
}

func handoverFromRequest(r *http.Request, tenantSlug string, actorEmail string, now time.Time) (handoverRecord, []handoverTokenDelivery, error) {
	id, err := randomToken(10)
	if err != nil {
		return handoverRecord{}, nil, err
	}
	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		return handoverRecord{}, nil, fmt.Errorf("title required")
	}
	scheduledAt, err := parseOptionalLocalDateTime(r.FormValue("scheduled_at"), time.Time{})
	if err != nil {
		return handoverRecord{}, nil, err
	}
	confirmations, deliveries, err := handoverConfirmationsFromForm(r)
	if err != nil {
		return handoverRecord{}, nil, err
	}
	item := handoverRecord{
		ID:            id,
		TenantSlug:    tenantSlug,
		UnitID:        normalizeUnitID(r.FormValue("unit_id")),
		Title:         title,
		HandoverType:  r.FormValue("handover_type"),
		ScheduledAt:   scheduledAt,
		OutgoingName:  r.FormValue("outgoing_name"),
		OutgoingEmail: r.FormValue("outgoing_email"),
		IncomingName:  r.FormValue("incoming_name"),
		IncomingEmail: r.FormValue("incoming_email"),
		Rooms:         parseHandoverRooms(r.FormValue("rooms_text")),
		Meters:        parseHandoverMeters(r.FormValue("meters_text")),
		Keys:          parseHandoverKeys(r.FormValue("keys_text")),
		Notes:         r.FormValue("notes"),
		Confirmations: confirmations,
		CreatedBy:     actorEmail,
		CreatedAt:     now.UTC(),
		UpdatedAt:     now.UTC(),
	}
	item = normalizeHandover(item)
	if item.UnitID == "" || (len(item.Rooms) == 0 && len(item.Meters) == 0 && len(item.Keys) == 0 && item.Notes == "") {
		return handoverRecord{}, nil, fmt.Errorf("unit and content required")
	}
	return item, deliveries, nil
}

func handoverConfirmationsFromForm(r *http.Request) ([]handoverConfirmation, []handoverTokenDelivery, error) {
	raw := []struct {
		Role  string
		Name  string
		Email string
	}{
		{Role: "Ausziehend", Name: r.FormValue("outgoing_name"), Email: r.FormValue("outgoing_email")},
		{Role: "Einziehend", Name: r.FormValue("incoming_name"), Email: r.FormValue("incoming_email")},
	}
	confirmations := []handoverConfirmation{}
	deliveries := []handoverTokenDelivery{}
	for _, candidate := range raw {
		name := truncateRunes(strings.TrimSpace(candidate.Name), 120)
		email := normalizeEmail(candidate.Email)
		if email == "" {
			continue
		}
		if _, err := mail.ParseAddress(email); err != nil {
			return nil, nil, err
		}
		token, err := randomToken(18)
		if err != nil {
			return nil, nil, err
		}
		confirmations = append(confirmations, handoverConfirmation{
			Role:      candidate.Role,
			Name:      name,
			Email:     email,
			TokenHash: handoverTokenHash(token),
		})
		if email != "" {
			deliveries = append(deliveries, handoverTokenDelivery{Role: candidate.Role, Name: name, Email: email, Token: token})
		}
	}
	return confirmations, deliveries, nil
}

func parseHandoverRooms(raw string) []handoverRoom {
	rows := parseStructuredLines(raw)
	out := make([]handoverRoom, 0, len(rows))
	for _, row := range rows {
		item := handoverRoom{Name: row[0]}
		if len(row) > 1 {
			item.Condition = row[1]
		}
		if len(row) > 2 {
			item.Defects = strings.Join(row[2:], " · ")
		}
		out = append(out, item)
	}
	return normalizeHandoverRooms(out)
}

func parseHandoverMeters(raw string) []handoverMeter {
	rows := parseStructuredLines(raw)
	out := make([]handoverMeter, 0, len(rows))
	for _, row := range rows {
		item := handoverMeter{Label: row[0]}
		if len(row) > 1 {
			item.Value = row[1]
		}
		if len(row) > 2 {
			item.Unit = row[2]
		}
		out = append(out, item)
	}
	return normalizeHandoverMeters(out)
}

func parseHandoverKeys(raw string) []handoverKey {
	rows := parseStructuredLines(raw)
	out := make([]handoverKey, 0, len(rows))
	for _, row := range rows {
		count := 1
		if len(row) > 1 {
			if parsed, err := strconv.Atoi(strings.TrimSpace(row[1])); err == nil {
				count = parsed
			}
		}
		out = append(out, handoverKey{Label: row[0], Count: count})
	}
	return normalizeHandoverKeys(out)
}

func parseStructuredLines(raw string) [][]string {
	out := [][]string{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		row := []string{}
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				row = append(row, part)
			}
		}
		if len(row) > 0 {
			out = append(out, row)
		}
	}
	return out
}

func (a *app) notifyHandoverParticipants(r *http.Request, tenant tenantConfig, item handoverRecord, deliveries []handoverTokenDelivery) {
	if a == nil || a.mailer == nil || !a.mailer.Configured() {
		return
	}
	for _, delivery := range deliveries {
		if delivery.Email == "" || delivery.Token == "" {
			continue
		}
		link := a.publicBaseURL(r, tenant) + "/handover/" + url.PathEscape(delivery.Token)
		subject := "Übergabeprotokoll bestätigen: " + item.Title
		body := strings.Join([]string{
			"Guten Tag,",
			"",
			"für " + tenant.Address + " wurde ein Übergabeprotokoll vorbereitet.",
			"Rolle: " + delivery.Role,
			"",
			"Bitte prüfen und bestätigen:",
			link,
			"",
			"Dieser Link ist nur für dieses Protokoll bestimmt.",
		}, "\n")
		if err := a.mailer.SendNotification(delivery.Email, subject, body); err != nil {
			logWarn("handover notification failed",
				"tenant", tenant.Slug,
				"error_type", fmt.Sprintf("%T", err),
			)
		}
	}
}

func (a *app) handoverConfirmPage(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.PathValue("token"))
	if a == nil || a.handoverStore == nil {
		http.NotFound(w, r)
		return
	}
	item, idx, found := a.handoverStore.GetByToken(token)
	if !found || idx < 0 || idx >= len(item.Confirmations) {
		http.NotFound(w, r)
		return
	}
	tenant, ok := a.tenantBySlug(item.TenantSlug)
	if !ok {
		http.NotFound(w, r)
		return
	}
	identity, ok := a.tenantIdentity(tenant.Slug)
	if !ok {
		http.NotFound(w, r)
		return
	}
	confirmation := item.Confirmations[idx]
	handover := a.handoverViewForActor(identity.Ref(), "", roleManager, item)
	for attachmentIndex := range handover.Attachments {
		attachment := &handover.Attachments[attachmentIndex]
		base := "/handover/" + url.PathEscape(token) + "/attachments/" + url.PathEscape(attachment.ID)
		attachment.URL = base
		attachment.PreviewURL = base + "/preview"
		attachment.ThumbURL = base + "/thumb"
		attachment.CanDelete = false
	}
	handover.AttachmentGroup = attachmentGroup{Attachments: handover.Attachments, HasAttachments: len(handover.Attachments) > 0}
	msg, okMsg := handoverMessage(r.URL.Query().Get("handover"))
	a.executeTemplate(w, "handoverConfirm", map[string]any{
		"Title":        "Übergabe bestätigen",
		"Tenant":       tenant,
		"Handover":     handover,
		"Confirmation": handoverConfirmationViewFrom(confirmation),
		"Token":        token,
		"Msg":          msg,
		"MsgOK":        okMsg,
		"AppVersion":   version.BuildLabel(),
	})
}

// handoverAttachment lets a participant inspect exactly the evidence attached
// to the handover addressed by their personal confirmation token. The token is
// checked again for every request and can never open another handover's file.
func (a *app) handoverAttachment(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.PathValue("token"))
	if a == nil || a.handoverStore == nil || a.attachmentStore == nil {
		http.NotFound(w, r)
		return
	}
	handover, confirmationIndex, found := a.handoverStore.GetByToken(token)
	if !found || confirmationIndex < 0 || confirmationIndex >= len(handover.Confirmations) {
		http.NotFound(w, r)
		return
	}
	identity, ok := a.tenantIdentity(handover.TenantSlug)
	if !ok {
		http.NotFound(w, r)
		return
	}
	attachments, ok := store.BindAttachmentRepository(a.attachmentStore, identity.Ref())
	if !ok {
		http.NotFound(w, r)
		return
	}
	attachment, found := attachments.Get(strings.TrimSpace(r.PathValue("id")))
	if !found || normalizeAttachmentEntity(attachment.EntityType) != "handover" || attachment.EntityID != handover.ID {
		http.NotFound(w, r)
		return
	}
	variant := strings.ToLower(strings.TrimSpace(r.PathValue("variant")))
	if variant != "" && variant != "preview" && variant != "thumb" && variant != "thumbnail" {
		http.NotFound(w, r)
		return
	}
	path, contentType, _, ok := attachments.FilePath(attachment, variant)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if _, err := os.Stat(path); err != nil {
		logError("handover attachment file open failed", err, "tenant", handover.TenantSlug, "attachment_id", attachment.ID)
		http.NotFound(w, r)
		return
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; img-src 'self' data:; style-src 'unsafe-inline'; sandbox")
	w.Header().Set("Cache-Control", "private, no-store")
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.Header().Set("Content-Disposition", mime.FormatMediaType(attachmentDisposition(contentType), map[string]string{"filename": attachment.Filename}))
	if (variant == "" || variant == "preview") && strings.TrimSpace(r.Header.Get("Range")) == "" {
		confirmation := handover.Confirmations[confirmationIndex]
		a.recordAudit(auditEvent{
			TenantSlug: handover.TenantSlug,
			ActorEmail: confirmation.Email,
			ActorRole:  confirmation.Role,
			Action:     auditActionAttachmentView,
			TargetType: "attachment",
			TargetID:   attachment.ID,
			Summary:    "Übergabeanhang angezeigt",
			Details: map[string]string{
				"entity_type": "handover",
				"entity_id":   handover.ID,
				"access":      "Bestätigungslink",
			},
		})
	}
	http.ServeFile(w, r, path)
}

func (a *app) confirmHandover(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.PathValue("token"))
	if a == nil || a.handoverStore == nil {
		http.NotFound(w, r)
		return
	}
	if !sameOriginPost(r) {
		http.Error(w, "Bad request", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/handover/"+url.PathEscape(token)+"?handover=invalid", http.StatusSeeOther)
		return
	}
	if r.FormValue("confirm") != "yes" {
		http.Redirect(w, r, "/handover/"+url.PathEscape(token)+"?handover=invalid", http.StatusSeeOther)
		return
	}
	confirmationTime := time.Now()
	item, confirmation, found, err := a.handoverStore.ConfirmByToken(token, r.FormValue("name"), r.FormValue("note"), confirmationTime)
	if err != nil {
		logHandoverError("confirm", "", "", err)
		http.Redirect(w, r, "/handover/"+url.PathEscape(token)+"?handover=error", http.StatusSeeOther)
		return
	}
	if !found {
		http.NotFound(w, r)
		return
	}
	// ConfirmByToken intentionally returns the original confirmation on a
	// repeated submit. Only the request whose exact timestamp was persisted may
	// add an audit event; refreshing or re-posting an old token stays read-only.
	if !confirmation.ConfirmedAt.Equal(confirmationTime.UTC()) {
		http.Redirect(w, r, "/handover/"+url.PathEscape(token)+"?handover=confirmed", http.StatusSeeOther)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: item.TenantSlug,
		ActorEmail: confirmation.Email,
		ActorRole:  confirmation.Role,
		Action:     auditActionHandoverConfirm,
		TargetType: "handover",
		TargetID:   item.ID,
		Summary:    "Übergabe bestätigt",
		Details: map[string]string{
			"title": item.Title,
			"role":  confirmation.Role,
		},
	})
	http.Redirect(w, r, "/handover/"+url.PathEscape(token)+"?handover=confirmed", http.StatusSeeOther)
}

func (a *app) handoverProtocol(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	if !canManageHandovers(ac.actor(), ac.resource()) {
		http.Error(w, "Dieses Protokoll ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	item, found := ac.repositories.handovers.Get(strings.TrimSpace(r.PathValue("id")))
	if !found {
		http.NotFound(w, r)
		return
	}
	attachments := []attachmentRecord{}
	if ac.repositories.attachments != nil {
		attachments = ac.repositories.attachments.ListEntity("handover", item.ID)
	}
	pdf := handoverPDF(tenant, item, attachments, time.Now())
	filename := "uebergabe-" + item.ID + "-protokoll.pdf"
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: email,
		ActorRole:  role,
		Action:     auditActionDocumentDownload,
		TargetType: "handover",
		TargetID:   item.ID,
		Summary:    "Übergabeprotokoll exportiert",
		Details: map[string]string{
			"title": item.Title,
		},
	})
	_, _ = w.Write(pdf)
}

// filerOrFallback returns the wired protocol filer, or derives a sequential one
// from the current stores. newApp always wires it, but an app assembled
// directly (as tests do) may not — filing must still work there, just without
// the shared transaction.
func (a *app) filerOrFallback() protocolFiler {
	if a.protocolFiler != nil {
		return a.protocolFiler
	}
	return newSequentialProtocolFiler(a.documentStore, a.handoverStore)
}

func (a *app) fileHandoverProtocol(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	if !canManageHandovers(ac.actor(), ac.resource()) || !ac.can(capabilityManageDocuments) {
		http.Error(w, "Ablage im Dokumentenbereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/app/uebergaben?handover=invalid", http.StatusSeeOther)
		return
	}
	item, found := ac.repositories.handovers.Get(strings.TrimSpace(r.FormValue("id")))
	if !found {
		http.Redirect(w, r, "/app/uebergaben?handover=missing", http.StatusSeeOther)
		return
	}
	// Idempotent: if already filed, don't generate a SECOND document (HAUSV-148).
	// This catches the common retry — a double submit or a click after a slow
	// response — where the first filing already set FiledDocumentID.
	if item.FiledDocumentID != "" {
		http.Redirect(w, r, "/app/uebergaben?handover=filed#handover-"+url.PathEscape(item.ID), http.StatusSeeOther)
		return
	}
	if handoverStatus(item) == handoverStatusPending {
		http.Redirect(w, r, "/app/uebergaben?handover=pending#handover-"+url.PathEscape(item.ID), http.StatusSeeOther)
		return
	}
	attachments := []attachmentRecord{}
	if ac.repositories.attachments != nil {
		attachments = ac.repositories.attachments.ListEntity("handover", item.ID)
	}
	pdf := handoverPDF(tenant, item, attachments, time.Now())
	// One call: the document and the link on the handover are written together,
	// so a crash can no longer orphan a protocol whose retry duplicates it. The
	// filer re-checks "already filed" inside its transaction, which is the
	// authoritative guard; the early return above only avoids the wasted PDF.
	created, _, alreadyFiled, err := a.filerOrFallback().FileHandoverProtocol(
		ac.tenantRef, item.ID,
		documentRecord{
			TenantSlug: tenant.Slug,
			Title:      "Übergabeprotokoll " + item.Title,
			Category:   documentCategoryProtocol,
			Visibility: documentVisibilityOwnersOnly,
			UnitID:     item.UnitID,
			UploadedBy: email,
		},
		"uebergabe-"+item.ID+"-protokoll.pdf", "application/pdf", pdf, time.Now(),
	)
	if err != nil {
		logHandoverError("file", tenant.Slug, item.ID, err)
		http.Redirect(w, r, "/app/uebergaben?handover=error", http.StatusSeeOther)
		return
	}
	if alreadyFiled {
		http.Redirect(w, r, "/app/uebergaben?handover=filed#handover-"+url.PathEscape(item.ID), http.StatusSeeOther)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: email,
		ActorRole:  role,
		Action:     auditActionHandoverFile,
		TargetType: "handover",
		TargetID:   item.ID,
		Summary:    "Übergabeprotokoll abgelegt",
		Details: map[string]string{
			"title":       item.Title,
			"document_id": created.ID,
			"unit":        documentUnitAuditLabel(item.UnitID),
		},
	})
	http.Redirect(w, r, "/app/uebergaben?handover=filed#handover-"+url.PathEscape(item.ID), http.StatusSeeOther)
}

func (a *app) handoverViewsForActor(tenant store.TenantRef, email string, role string, items []handoverRecord) []handoverView {
	views := make([]handoverView, 0, len(items))
	for _, item := range items {
		views = append(views, a.handoverViewForActor(tenant, email, role, item))
	}
	return views
}

func handoverSections(views []handoverView) []handoverSectionView {
	sections := []handoverSectionView{
		{Title: "Jetzt offen", Description: "Diese Übergaben warten noch auf Bestätigung.", Open: true},
		{Title: "Bereit zur Ablage", Description: "Vollständig geprüft und bereit für den Dokumentenbereich.", Open: true},
		{Title: "Abgeschlossen", Description: "Fertig abgelegte Protokolle."},
	}
	for _, view := range views {
		index := 0
		if view.IsFiled {
			index = 2
		} else if view.IsReady {
			index = 1
		}
		sections[index].Items = append(sections[index].Items, view)
		sections[index].Count++
		sections[index].HasItems = true
	}
	return sections
}

func (a *app) handoverViewForActor(tenant store.TenantRef, email string, role string, item handoverRecord) handoverView {
	tenantSlug := tenant.Slug
	item = normalizeHandover(item)
	unitLabel := item.UnitID
	if a != nil && a.unitStore != nil && item.UnitID != "" {
		units, _ := store.BindUnitRepository(a.unitStore, tenant)
		if units != nil {
			if label := handoverUnitLabel(units.List(), item.UnitID); label != "" {
				unitLabel = label
			}
		}
	}
	attachments := a.attachmentViewsForEntity(tenant, "handover", item.ID, email, role)
	for idx := range attachments {
		attachments[idx].DeleteRedirect = "/app/uebergaben#handover-" + url.PathEscape(item.ID)
		if !handoverCanChangeFiles(item) {
			attachments[idx].CanDelete = false
		}
	}
	confirmations := make([]handoverConfirmationView, 0, len(item.Confirmations))
	for _, confirmation := range item.Confirmations {
		confirmations = append(confirmations, handoverConfirmationViewFrom(confirmation))
	}
	status := handoverStatus(item)
	confirmedCount := 0
	for _, confirmation := range item.Confirmations {
		if !confirmation.ConfirmedAt.IsZero() {
			confirmedCount++
		}
	}
	nextStep := "Protokoll vervollständigen"
	nextStepDetail := "Inhalte und Dateien prüfen"
	if status == handoverStatusPending {
		nextStep = "Auf Bestätigung warten"
		nextStepDetail = fmt.Sprintf("%d von %d bestätigt", confirmedCount, len(item.Confirmations))
	} else if status == handoverStatusConfirmed || status == handoverStatusDraft {
		nextStep = "Protokoll endgültig ablegen"
		nextStepDetail = "Bestätigungen vollständig"
	} else if status == handoverStatusFiled {
		nextStep = "Übergabe abgeschlossen"
		nextStepDetail = "Protokoll im Dokumentenbereich"
	}
	view := handoverView{
		ID:                 item.ID,
		Title:              item.Title,
		Type:               item.HandoverType,
		UnitLabel:          unitLabel,
		HasUnit:            unitLabel != "",
		Status:             status,
		StatusClass:        handoverStatusClass(status),
		NextStep:           nextStep,
		NextStepDetail:     nextStepDetail,
		ConfirmedCount:     confirmedCount,
		ConfirmationCount:  len(item.Confirmations),
		IsPending:          status == handoverStatusPending,
		IsReady:            status == handoverStatusConfirmed || status == handoverStatusDraft,
		IsFiled:            status == handoverStatusFiled,
		CanFile:            status == handoverStatusConfirmed || status == handoverStatusDraft,
		CanChangeFiles:     handoverCanChangeFiles(item),
		CanManageDocuments: can(actorFor(email, tenantSlug, role), capabilityManageDocuments, resourceFor(item.TenantSlug)),
		Outgoing:           handoverPartyLabel(item.OutgoingName, item.OutgoingEmail),
		Incoming:           handoverPartyLabel(item.IncomingName, item.IncomingEmail),
		Rooms:              item.Rooms,
		HasRooms:           len(item.Rooms) > 0,
		Meters:             item.Meters,
		HasMeters:          len(item.Meters) > 0,
		Keys:               item.Keys,
		HasKeys:            len(item.Keys) > 0,
		Notes:              item.Notes,
		HasNotes:           item.Notes != "",
		Confirmations:      confirmations,
		HasConfirmations:   len(confirmations) > 0,
		Attachments:        attachments,
		HasAttachments:     len(attachments) > 0,
		AttachmentGroup:    attachmentGroup{Attachments: attachments, HasAttachments: len(attachments) > 0},
		ProtocolURL:        "/app/uebergaben/" + url.PathEscape(item.ID) + "/protokoll",
		FileURL:            "/app/uebergaben/file",
		FiledDocumentID:    item.FiledDocumentID,
		HasFiledDocument:   item.FiledDocumentID != "",
		CreatedAt:          formatLocalDateTime(item.CreatedAt),
		UpdatedAt:          formatLocalDateTime(item.UpdatedAt),
	}
	if item.FiledDocumentID != "" {
		view.FiledDocumentURL = "/app/dokumente/" + url.PathEscape(item.FiledDocumentID) + "/download"
	}
	if !item.ScheduledAt.IsZero() {
		view.ScheduledAt = formatLocalDateTime(item.ScheduledAt)
		view.HasScheduledAt = true
	}
	return view
}

func handoverCanChangeFiles(item handoverRecord) bool {
	if item.FiledDocumentID != "" {
		return false
	}
	for _, confirmation := range item.Confirmations {
		if !confirmation.ConfirmedAt.IsZero() {
			return false
		}
	}
	return true
}

func handoverUnitLabel(units []unit, unitID string) string {
	unitID = normalizeUnitID(unitID)
	for _, item := range units {
		if normalizeUnitID(item.ID) == unitID && strings.TrimSpace(item.Label) != "" {
			return strings.TrimSpace(item.Label)
		}
	}
	return unitID
}

func handoverConfirmationViewFrom(item handoverConfirmation) handoverConfirmationView {
	confirmed := !item.ConfirmedAt.IsZero()
	status := "Offen"
	statusClass := "unread"
	if confirmed {
		status = "Bestätigt"
		statusClass = "ok"
	}
	return handoverConfirmationView{
		Role:         item.Role,
		Name:         item.Name,
		Email:        item.Email,
		Status:       status,
		StatusClass:  statusClass,
		ConfirmedAt:  formatLocalDateTime(item.ConfirmedAt),
		HasConfirmed: confirmed,
	}
}

func handoverStatus(item handoverRecord) string {
	if item.FiledDocumentID != "" {
		return handoverStatusFiled
	}
	if len(item.Confirmations) == 0 {
		return handoverStatusDraft
	}
	for _, confirmation := range item.Confirmations {
		if confirmation.ConfirmedAt.IsZero() {
			return handoverStatusPending
		}
	}
	return handoverStatusConfirmed
}

func handoverStatusClass(status string) string {
	switch status {
	case handoverStatusConfirmed, handoverStatusFiled:
		return "ok"
	case handoverStatusPending:
		return "unread"
	default:
		return ""
	}
}

func handoverPartyLabel(name string, email string) string {
	name = strings.TrimSpace(name)
	email = normalizeEmail(email)
	switch {
	case name != "" && email != "":
		return name + " · " + email
	case name != "":
		return name
	case email != "":
		return email
	default:
		return "Nicht angegeben"
	}
}

func handoverPDF(tenant tenantConfig, item handoverRecord, attachments []attachmentRecord, generatedAt time.Time) []byte {
	lines := []string{
		"Uebergabeprotokoll",
		tenant.Name + " - " + tenant.Address,
		"",
		"Titel: " + item.Title,
		"Typ: " + item.HandoverType,
		"Einheit: " + item.UnitID,
		"Termin: " + formatLocalDateTime(item.ScheduledAt),
		"Ausziehend: " + handoverPartyLabel(item.OutgoingName, item.OutgoingEmail),
		"Einziehend: " + handoverPartyLabel(item.IncomingName, item.IncomingEmail),
		"Status: " + handoverStatus(item),
		"Erstellt: " + formatLocalDateTime(item.CreatedAt),
		"Generiert: " + formatLocalDateTime(generatedAt),
		"Hinweis: Zustandsdokumentation; keine Kautions-, Schaden- oder sonstige Abrechnung.",
		"",
		"Raeume / Zustand / Maengel",
	}
	if len(item.Rooms) == 0 {
		lines = append(lines, "- Keine Raeume erfasst")
	}
	for _, room := range item.Rooms {
		lines = append(lines, "- "+strings.Join(nonEmptyParts(room.Name, room.Condition, room.Defects), " | "))
	}
	lines = append(lines, "", "Zaehlerstaende")
	if len(item.Meters) == 0 {
		lines = append(lines, "- Keine Zaehlerstaende erfasst")
	}
	for _, meter := range item.Meters {
		lines = append(lines, "- "+strings.Join(nonEmptyParts(meter.Label, meter.Value, meter.Unit), " | "))
	}
	lines = append(lines, "", "Schluessel")
	if len(item.Keys) == 0 {
		lines = append(lines, "- Keine Schluessel erfasst")
	}
	for _, key := range item.Keys {
		lines = append(lines, "- "+key.Label+": "+strconv.Itoa(key.Count))
	}
	lines = append(lines, "", "Bestaetigungen")
	if len(item.Confirmations) == 0 {
		lines = append(lines, "- Keine externen Bestaetigungen vorgesehen")
	}
	for _, confirmation := range item.Confirmations {
		status := "offen"
		if !confirmation.ConfirmedAt.IsZero() {
			status = "bestaetigt am " + formatLocalDateTime(confirmation.ConfirmedAt)
		}
		lines = append(lines, "- "+confirmation.Role+": "+handoverPartyLabel(confirmation.Name, confirmation.Email)+" - "+status)
	}
	lines = append(lines, "", "Fotos / Anhaenge")
	if len(attachments) == 0 {
		lines = append(lines, "- Keine Anhaenge")
	}
	for _, attachment := range attachments {
		lines = append(lines, "- "+attachment.Filename+" ("+formatBytes(attachment.Size)+")")
	}
	if item.Notes != "" {
		lines = append(lines, "", "Notizen")
		lines = append(lines, pdf.WrapText(item.Notes, 92)...)
	}
	return pdf.Simple(lines)
}

func nonEmptyParts(parts ...string) []string {
	out := []string{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func logHandoverError(action string, tenantSlug string, id string, err error) {
	logError("handover action failed", err, "action", action, "tenant", tenantSlug, "handover_id", id)
}
