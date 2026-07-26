package server

import (
	"fmt"
	"mime"
	"net/http"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/markus-barta/hausv-org/internal/pdf"
	"github.com/markus-barta/hausv-org/internal/store"
	"github.com/markus-barta/hausv-org/internal/version"
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
	ID               string
	Title            string
	Type             string
	UnitLabel        string
	HasUnit          bool
	ScheduledAt      string
	HasScheduledAt   bool
	Status           string
	StatusClass      string
	Outgoing         string
	Incoming         string
	Rooms            []handoverRoom
	HasRooms         bool
	Meters           []handoverMeter
	HasMeters        bool
	Keys             []handoverKey
	HasKeys          bool
	Notes            string
	HasNotes         bool
	Confirmations    []handoverConfirmationView
	HasConfirmations bool
	Attachments      []attachmentView
	HasAttachments   bool
	AttachmentGroup  attachmentGroup
	ProtocolURL      string
	FileURL          string
	FiledDocumentID  string
	HasFiledDocument bool
	FiledDocumentURL string
	CreatedAt        string
	UpdatedAt        string
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
	tenant, email, role := ac.tenant, ac.email, ac.role
	if !canManageHandovers(role) {
		http.Error(w, "Übergabeprotokolle sind der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	items := []handoverRecord{}
	if a.handoverStore != nil {
		items = a.handoverStore.ListTenant(tenant.Slug)
	}
	msg, okMsg := handoverMessage(r.URL.Query().Get("handover"))
	a.render(w, "handovers", a.withBase(ac, map[string]any{
		"Title":              "Übergaben",
		"CanManageHandovers": true,
		"ActivePage":         "handovers",
		"Handovers":          a.handoverViewsForActor(tenant.Slug, email, role, items),
		"HasHandovers":       len(items) > 0,
		"HandoversEmpty":     emptyState("Noch keine Übergaben", "Neue Nutzerwechsel werden hier mit Räumen, Zählern, Schlüsseln, Fotos und Bestätigung dokumentiert."),
		"HandoverMsg":        msg,
		"HandoverOK":         okMsg,
		"UnitOptions":        handoverUnitOptions(a.unitStore.ListTenant(tenant.Slug), ""),
		"NowInput":           formatLocalDateTimeInput(time.Now()),
	}))
}

func handoverMessage(status string) (string, bool) {
	switch status {
	case "created":
		return "Übergabeprotokoll angelegt. Bestätigungslinks wurden vorbereitet und, falls Mail aktiv ist, versendet.", true
	case "confirmed":
		return "Bestätigung gespeichert.", true
	case "filed":
		return "Protokoll im Dokumentenbereich abgelegt.", true
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

func (a *app) createHandover(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	if !canManageHandovers(role) {
		http.Error(w, "Übergabeprotokolle sind der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if a.handoverStore == nil {
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
		if a.attachmentStore == nil {
			http.Redirect(w, r, "/app/uebergaben?handover=invalid", http.StatusSeeOther)
			return
		}
		uploaded, err = a.attachmentStore.CreateUploaded(tenant.Slug, "handover", item.ID, email, uploadedFilesFromHeaders(attachmentHeaders), now)
		if err != nil {
			http.Redirect(w, r, "/app/uebergaben?handover=invalid", http.StatusSeeOther)
			return
		}
	}
	created, err := a.handoverStore.Create(item)
	if err != nil {
		for _, attachment := range uploaded {
			_, _, _ = a.attachmentStore.Delete(tenant.Slug, attachment.ID, time.Now())
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
			logError("handover notification failed", err, "tenant", tenant.Slug, "recipient", redactedEmail(delivery.Email))
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
	confirmation := item.Confirmations[idx]
	msg, okMsg := handoverMessage(r.URL.Query().Get("handover"))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.templates.ExecuteTemplate(w, "handoverConfirm", map[string]any{
		"Title":        "Übergabe bestätigen",
		"Tenant":       tenant,
		"Handover":     a.handoverViewForActor(item.TenantSlug, "", roleManager, item),
		"Confirmation": handoverConfirmationViewFrom(confirmation),
		"Token":        token,
		"Msg":          msg,
		"MsgOK":        okMsg,
		"AppVersion":   version.BuildLabel(),
	}); err != nil {
		logError("handover confirmation render failed", err)
	}
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
	item, confirmation, found, err := a.handoverStore.ConfirmByToken(token, r.FormValue("name"), r.FormValue("note"), time.Now())
	if err != nil {
		logHandoverError("confirm", "", "", err)
		http.Redirect(w, r, "/handover/"+url.PathEscape(token)+"?handover=error", http.StatusSeeOther)
		return
	}
	if !found {
		http.NotFound(w, r)
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
	if !canManageHandovers(role) {
		http.Error(w, "Dieses Protokoll ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	item, found := a.handoverStore.Get(tenant.Slug, strings.TrimSpace(r.PathValue("id")))
	if !found {
		http.NotFound(w, r)
		return
	}
	attachments := []attachmentRecord{}
	if a.attachmentStore != nil {
		attachments = a.attachmentStore.ListEntity(tenant.Slug, "handover", item.ID)
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
	if !canManageHandovers(role) || !hasCapability(role, capabilityManageDocuments) {
		http.Error(w, "Ablage im Dokumentenbereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/app/uebergaben?handover=invalid", http.StatusSeeOther)
		return
	}
	item, found := a.handoverStore.Get(tenant.Slug, strings.TrimSpace(r.FormValue("id")))
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
	attachments := []attachmentRecord{}
	if a.attachmentStore != nil {
		attachments = a.attachmentStore.ListEntity(tenant.Slug, "handover", item.ID)
	}
	pdf := handoverPDF(tenant, item, attachments, time.Now())
	// One call: the document and the link on the handover are written together,
	// so a crash can no longer orphan a protocol whose retry duplicates it. The
	// filer re-checks "already filed" inside its transaction, which is the
	// authoritative guard; the early return above only avoids the wasted PDF.
	created, _, alreadyFiled, err := a.filerOrFallback().FileHandoverProtocol(
		tenant.Slug, item.ID,
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

func (a *app) handoverViewsForActor(tenantSlug string, email string, role string, items []handoverRecord) []handoverView {
	views := make([]handoverView, 0, len(items))
	for _, item := range items {
		views = append(views, a.handoverViewForActor(tenantSlug, email, role, item))
	}
	return views
}

func (a *app) handoverViewForActor(tenantSlug string, email string, role string, item handoverRecord) handoverView {
	item = normalizeHandover(item)
	unitLabel := item.UnitID
	if a != nil && a.unitStore != nil && item.UnitID != "" {
		if label := handoverUnitLabel(a.unitStore.ListTenant(tenantSlug), item.UnitID); label != "" {
			unitLabel = label
		}
	}
	attachments := a.attachmentViewsForEntity(tenantSlug, "handover", item.ID, email, role)
	for idx := range attachments {
		attachments[idx].DeleteRedirect = "/app/uebergaben#handover-" + url.PathEscape(item.ID)
	}
	confirmations := make([]handoverConfirmationView, 0, len(item.Confirmations))
	for _, confirmation := range item.Confirmations {
		confirmations = append(confirmations, handoverConfirmationViewFrom(confirmation))
	}
	status := handoverStatus(item)
	view := handoverView{
		ID:               item.ID,
		Title:            item.Title,
		Type:             item.HandoverType,
		UnitLabel:        unitLabel,
		HasUnit:          unitLabel != "",
		Status:           status,
		StatusClass:      handoverStatusClass(status),
		Outgoing:         handoverPartyLabel(item.OutgoingName, item.OutgoingEmail),
		Incoming:         handoverPartyLabel(item.IncomingName, item.IncomingEmail),
		Rooms:            item.Rooms,
		HasRooms:         len(item.Rooms) > 0,
		Meters:           item.Meters,
		HasMeters:        len(item.Meters) > 0,
		Keys:             item.Keys,
		HasKeys:          len(item.Keys) > 0,
		Notes:            item.Notes,
		HasNotes:         item.Notes != "",
		Confirmations:    confirmations,
		HasConfirmations: len(confirmations) > 0,
		Attachments:      attachments,
		HasAttachments:   len(attachments) > 0,
		AttachmentGroup:  attachmentGroup{Attachments: attachments, HasAttachments: len(attachments) > 0},
		ProtocolURL:      "/app/uebergaben/" + url.PathEscape(item.ID) + "/protokoll",
		FileURL:          "/app/uebergaben/file",
		FiledDocumentID:  item.FiledDocumentID,
		HasFiledDocument: item.FiledDocumentID != "",
		CreatedAt:        formatLocalDateTime(item.CreatedAt),
		UpdatedAt:        formatLocalDateTime(item.UpdatedAt),
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
