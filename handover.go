package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mime"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	auditActionHandoverCreate  = "handover.create"
	auditActionHandoverConfirm = "handover.confirm"
	auditActionHandoverFile    = "handover.file"

	handoverStatusDraft     = "Entwurf"
	handoverStatusPending   = "Wartet auf Bestätigung"
	handoverStatusConfirmed = "Bestätigt"
	handoverStatusFiled     = "Abgelegt"
)

type handoverStore struct {
	mu   sync.Mutex
	path string
	data handoverStoreData
}

type handoverStoreData struct {
	Handovers []handoverRecord `json:"handovers"`
}

type handoverRecord struct {
	ID              string                 `json:"id"`
	TenantSlug      string                 `json:"tenant"`
	UnitID          string                 `json:"unit_id,omitempty"`
	Title           string                 `json:"title"`
	HandoverType    string                 `json:"handover_type"`
	ScheduledAt     time.Time              `json:"scheduled_at,omitempty"`
	OutgoingName    string                 `json:"outgoing_name,omitempty"`
	OutgoingEmail   string                 `json:"outgoing_email,omitempty"`
	IncomingName    string                 `json:"incoming_name,omitempty"`
	IncomingEmail   string                 `json:"incoming_email,omitempty"`
	Rooms           []handoverRoom         `json:"rooms,omitempty"`
	Meters          []handoverMeter        `json:"meters,omitempty"`
	Keys            []handoverKey          `json:"keys,omitempty"`
	Notes           string                 `json:"notes,omitempty"`
	Confirmations   []handoverConfirmation `json:"confirmations,omitempty"`
	FiledDocumentID string                 `json:"filed_document_id,omitempty"`
	CreatedBy       string                 `json:"created_by"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

type handoverRoom struct {
	Name      string `json:"name"`
	Condition string `json:"condition,omitempty"`
	Defects   string `json:"defects,omitempty"`
}

type handoverMeter struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Unit  string `json:"unit,omitempty"`
}

type handoverKey struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

type handoverConfirmation struct {
	Role        string    `json:"role"`
	Name        string    `json:"name,omitempty"`
	Email       string    `json:"email,omitempty"`
	TokenHash   string    `json:"token_hash,omitempty"`
	ConfirmedAt time.Time `json:"confirmed_at,omitempty"`
	ConfirmedBy string    `json:"confirmed_by,omitempty"`
	Note        string    `json:"note,omitempty"`
}

type handoverTokenDelivery struct {
	Role  string
	Name  string
	Email string
	Token string
}

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

func newHandoverStore(path string) (*handoverStore, error) {
	store := &handoverStore{path: path, data: handoverStoreData{Handovers: []handoverRecord{}}}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read handover data")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("invalid handover data")
	}
	store.data.Handovers = normalizeHandovers(store.data.Handovers)
	return store, nil
}

func (s *handoverStore) Create(item handoverRecord) (handoverRecord, error) {
	if s == nil {
		return handoverRecord{}, fmt.Errorf("handover store unavailable")
	}
	item = normalizeHandover(item)
	if item.ID == "" || item.TenantSlug == "" || item.Title == "" || item.CreatedBy == "" {
		return handoverRecord{}, fmt.Errorf("invalid handover")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.data.Handovers {
		if normalizeSlug(existing.TenantSlug) == item.TenantSlug && existing.ID == item.ID {
			return handoverRecord{}, fmt.Errorf("handover exists")
		}
	}
	s.data.Handovers = append(s.data.Handovers, item)
	sortHandovers(s.data.Handovers)
	if err := s.saveLocked(); err != nil {
		return handoverRecord{}, err
	}
	return copyHandover(item), nil
}

func (s *handoverStore) ListTenant(tenantSlug string) []handoverRecord {
	if s == nil {
		return nil
	}
	tenantSlug = normalizeSlug(tenantSlug)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []handoverRecord{}
	for _, item := range s.data.Handovers {
		if normalizeSlug(item.TenantSlug) == tenantSlug {
			out = append(out, copyHandover(item))
		}
	}
	sortHandovers(out)
	return out
}

func (s *handoverStore) Get(tenantSlug string, id string) (handoverRecord, bool) {
	if s == nil {
		return handoverRecord{}, false
	}
	tenantSlug = normalizeSlug(tenantSlug)
	id = strings.TrimSpace(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.data.Handovers {
		if normalizeSlug(item.TenantSlug) == tenantSlug && item.ID == id {
			return copyHandover(item), true
		}
	}
	return handoverRecord{}, false
}

func (s *handoverStore) GetByToken(token string) (handoverRecord, int, bool) {
	if s == nil || strings.TrimSpace(token) == "" {
		return handoverRecord{}, -1, false
	}
	hash := handoverTokenHash(token)
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.data.Handovers {
		for idx, confirmation := range item.Confirmations {
			if confirmation.TokenHash != "" && subtleConstantStringCompare(confirmation.TokenHash, hash) {
				return copyHandover(item), idx, true
			}
		}
	}
	return handoverRecord{}, -1, false
}

func (s *handoverStore) ConfirmByToken(token string, name string, note string, at time.Time) (handoverRecord, handoverConfirmation, bool, error) {
	if s == nil || strings.TrimSpace(token) == "" {
		return handoverRecord{}, handoverConfirmation{}, false, nil
	}
	hash := handoverTokenHash(token)
	name = truncateRunes(strings.TrimSpace(name), 120)
	note = truncateRunes(strings.TrimSpace(note), 500)
	if at.IsZero() {
		at = time.Now()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, item := range s.data.Handovers {
		for j, confirmation := range item.Confirmations {
			if confirmation.TokenHash == "" || !subtleConstantStringCompare(confirmation.TokenHash, hash) {
				continue
			}
			if !confirmation.ConfirmedAt.IsZero() {
				return copyHandover(item), confirmation, true, nil
			}
			confirmation.ConfirmedAt = at.UTC()
			confirmation.ConfirmedBy = firstNonEmpty(name, confirmation.Name, confirmation.Email)
			confirmation.Note = note
			item.Confirmations[j] = confirmation
			item.UpdatedAt = at.UTC()
			s.data.Handovers[i] = normalizeHandover(item)
			sortHandovers(s.data.Handovers)
			if err := s.saveLocked(); err != nil {
				return handoverRecord{}, handoverConfirmation{}, true, err
			}
			return copyHandover(s.data.Handovers[i]), confirmation, true, nil
		}
	}
	return handoverRecord{}, handoverConfirmation{}, false, nil
}

func (s *handoverStore) SetFiledDocument(tenantSlug string, id string, documentID string, at time.Time) (handoverRecord, bool, error) {
	if s == nil {
		return handoverRecord{}, false, fmt.Errorf("handover store unavailable")
	}
	tenantSlug = normalizeSlug(tenantSlug)
	id = strings.TrimSpace(id)
	documentID = strings.TrimSpace(documentID)
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, item := range s.data.Handovers {
		if normalizeSlug(item.TenantSlug) != tenantSlug || item.ID != id {
			continue
		}
		item.FiledDocumentID = documentID
		if at.IsZero() {
			at = time.Now()
		}
		item.UpdatedAt = at.UTC()
		s.data.Handovers[i] = normalizeHandover(item)
		if err := s.saveLocked(); err != nil {
			return handoverRecord{}, true, err
		}
		return copyHandover(s.data.Handovers[i]), true, nil
	}
	return handoverRecord{}, false, nil
}

func (s *handoverStore) saveLocked() error {
	return saveJSONAtomic(s.path, s.data, "handover")
}

func normalizeHandovers(items []handoverRecord) []handoverRecord {
	out := make([]handoverRecord, 0, len(items))
	for _, item := range items {
		item = normalizeHandover(item)
		if item.ID == "" || item.TenantSlug == "" || item.Title == "" {
			continue
		}
		out = append(out, item)
	}
	sortHandovers(out)
	return out
}

func normalizeHandover(item handoverRecord) handoverRecord {
	item.ID = strings.TrimSpace(item.ID)
	item.TenantSlug = normalizeSlug(item.TenantSlug)
	item.UnitID = normalizeUnitID(item.UnitID)
	item.Title = truncateRunes(strings.TrimSpace(item.Title), 160)
	item.HandoverType = normalizeHandoverType(item.HandoverType)
	item.OutgoingName = truncateRunes(strings.TrimSpace(item.OutgoingName), 120)
	item.OutgoingEmail = normalizeEmail(item.OutgoingEmail)
	item.IncomingName = truncateRunes(strings.TrimSpace(item.IncomingName), 120)
	item.IncomingEmail = normalizeEmail(item.IncomingEmail)
	item.Notes = truncateRunes(strings.TrimSpace(item.Notes), 3000)
	item.CreatedBy = normalizeEmail(item.CreatedBy)
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now().UTC()
	} else {
		item.CreatedAt = item.CreatedAt.UTC()
	}
	if item.UpdatedAt.IsZero() {
		item.UpdatedAt = item.CreatedAt
	} else {
		item.UpdatedAt = item.UpdatedAt.UTC()
	}
	if !item.ScheduledAt.IsZero() {
		item.ScheduledAt = item.ScheduledAt.UTC()
	}
	item.Rooms = normalizeHandoverRooms(item.Rooms)
	item.Meters = normalizeHandoverMeters(item.Meters)
	item.Keys = normalizeHandoverKeys(item.Keys)
	item.Confirmations = normalizeHandoverConfirmations(item.Confirmations)
	item.FiledDocumentID = strings.TrimSpace(item.FiledDocumentID)
	return item
}

func normalizeHandoverType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "einzug", "move-in", "in":
		return "Einzug"
	case "auszug", "move-out", "out":
		return "Auszug"
	case "wechsel", "nutzerwechsel", "handover", "":
		return "Nutzerwechsel"
	default:
		return truncateRunes(strings.TrimSpace(raw), 80)
	}
}

func normalizeHandoverRooms(items []handoverRoom) []handoverRoom {
	out := []handoverRoom{}
	for _, item := range items {
		item.Name = truncateRunes(strings.TrimSpace(item.Name), 100)
		item.Condition = truncateRunes(strings.TrimSpace(item.Condition), 120)
		item.Defects = truncateRunes(strings.TrimSpace(item.Defects), 500)
		if item.Name != "" || item.Condition != "" || item.Defects != "" {
			out = append(out, item)
		}
	}
	return out
}

func normalizeHandoverMeters(items []handoverMeter) []handoverMeter {
	out := []handoverMeter{}
	for _, item := range items {
		item.Label = truncateRunes(strings.TrimSpace(item.Label), 100)
		item.Value = truncateRunes(strings.TrimSpace(item.Value), 80)
		item.Unit = truncateRunes(strings.TrimSpace(item.Unit), 40)
		if item.Label != "" && item.Value != "" {
			out = append(out, item)
		}
	}
	return out
}

func normalizeHandoverKeys(items []handoverKey) []handoverKey {
	out := []handoverKey{}
	for _, item := range items {
		item.Label = truncateRunes(strings.TrimSpace(item.Label), 100)
		if item.Count < 0 {
			item.Count = 0
		}
		if item.Label != "" && item.Count > 0 {
			out = append(out, item)
		}
	}
	return out
}

func normalizeHandoverConfirmations(items []handoverConfirmation) []handoverConfirmation {
	out := []handoverConfirmation{}
	for _, item := range items {
		item.Role = truncateRunes(strings.TrimSpace(item.Role), 80)
		item.Name = truncateRunes(strings.TrimSpace(item.Name), 120)
		item.Email = normalizeEmail(item.Email)
		item.TokenHash = strings.TrimSpace(item.TokenHash)
		item.ConfirmedBy = truncateRunes(strings.TrimSpace(item.ConfirmedBy), 120)
		item.Note = truncateRunes(strings.TrimSpace(item.Note), 500)
		if !item.ConfirmedAt.IsZero() {
			item.ConfirmedAt = item.ConfirmedAt.UTC()
		}
		if item.Role != "" && (item.Name != "" || item.Email != "") {
			out = append(out, item)
		}
	}
	return out
}

func sortHandovers(items []handoverRecord) {
	sort.SliceStable(items, func(i, j int) bool {
		leftDate := items[i].ScheduledAt
		rightDate := items[j].ScheduledAt
		if leftDate.IsZero() {
			leftDate = items[i].CreatedAt
		}
		if rightDate.IsZero() {
			rightDate = items[j].CreatedAt
		}
		if !leftDate.Equal(rightDate) {
			return leftDate.After(rightDate)
		}
		return strings.ToLower(items[i].Title) < strings.ToLower(items[j].Title)
	})
}

func copyHandover(item handoverRecord) handoverRecord {
	item.Rooms = append([]handoverRoom(nil), item.Rooms...)
	item.Meters = append([]handoverMeter(nil), item.Meters...)
	item.Keys = append([]handoverKey(nil), item.Keys...)
	item.Confirmations = append([]handoverConfirmation(nil), item.Confirmations...)
	return item
}

func canManageHandovers(role string) bool {
	return hasCapability(role, capabilityManageDocuments) || hasCapability(role, capabilityManageBuilding) || hasCapability(role, capabilityManageUsers)
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

func (a *app) handovers(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !canManageHandovers(role) {
		http.Error(w, "Übergabeprotokolle sind der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	profile := a.profileForTenant(email, tenant.Slug)
	items := []handoverRecord{}
	if a.handoverStore != nil {
		items = a.handoverStore.ListTenant(tenant.Slug)
	}
	msg, okMsg := handoverMessage(r.URL.Query().Get("handover"))
	a.render(w, "handovers", map[string]any{
		"Title":              "Übergaben",
		"Tenant":             tenant,
		"Email":              email,
		"DisplayName":        profile.DisplayName(),
		"Initials":           profile.Initials(),
		"Role":               role,
		"IsAdmin":            hasCapability(role, capabilityPlatformAdmin),
		"CanSeeParking":      hasCapability(role, capabilityPlatformAdmin) || profile.HasPermission(permissionParking),
		"CanManageHandovers": true,
		"ActivePage":         "handovers",
		"Handovers":          a.handoverViewsForActor(tenant.Slug, email, role, items),
		"HasHandovers":       len(items) > 0,
		"HandoversEmpty":     emptyState("Noch keine Übergaben", "Neue Nutzerwechsel werden hier mit Räumen, Zählern, Schlüsseln, Fotos und Bestätigung dokumentiert."),
		"HandoverMsg":        msg,
		"HandoverOK":         okMsg,
		"UnitOptions":        handoverUnitOptions(a.unitStore.ListTenant(tenant.Slug), ""),
		"NowInput":           formatLocalDateTimeInput(time.Now()),
	})
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

func (a *app) createHandover(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !canManageHandovers(role) {
		http.Error(w, "Übergabeprotokolle sind der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if !sameOriginPost(r) {
		http.Error(w, "Bad request", http.StatusForbidden)
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
			log.Printf("handover notification failed for %s/%s: %v", tenant.Slug, redactedEmail(delivery.Email), err)
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
		"AppVersion":   buildLabel(),
	}); err != nil {
		log.Printf("render handoverConfirm failed: %v", err)
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

func (a *app) handoverProtocol(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
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

func (a *app) fileHandoverProtocol(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !canManageHandovers(role) || !hasCapability(role, capabilityManageDocuments) {
		http.Error(w, "Ablage im Dokumentenbereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if !sameOriginPost(r) {
		http.Error(w, "Bad request", http.StatusForbidden)
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
	attachments := []attachmentRecord{}
	if a.attachmentStore != nil {
		attachments = a.attachmentStore.ListEntity(tenant.Slug, "handover", item.ID)
	}
	pdf := handoverPDF(tenant, item, attachments, time.Now())
	created, err := a.documentStore.CreateGenerated(documentRecord{
		TenantSlug: tenant.Slug,
		Title:      "Übergabeprotokoll " + item.Title,
		Category:   documentCategoryProtocol,
		Visibility: documentVisibilityOwnersOnly,
		UnitID:     item.UnitID,
		UploadedBy: email,
	}, "uebergabe-"+item.ID+"-protokoll.pdf", "application/pdf", pdf, time.Now())
	if err != nil {
		logHandoverError("file", tenant.Slug, item.ID, err)
		http.Redirect(w, r, "/app/uebergaben?handover=error", http.StatusSeeOther)
		return
	}
	_, _, err = a.handoverStore.SetFiledDocument(tenant.Slug, item.ID, created.ID, time.Now())
	if err != nil {
		logHandoverError("file-state", tenant.Slug, item.ID, err)
		http.Redirect(w, r, "/app/uebergaben?handover=error", http.StatusSeeOther)
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

func handoverTokenHash(token string) string {
	sum := sha256.Sum256([]byte("handover-confirmation-v1:" + strings.TrimSpace(token)))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func subtleConstantStringCompare(a string, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var out byte
	for i := 0; i < len(a); i++ {
		out |= a[i] ^ b[i]
	}
	return out == 0
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
		lines = append(lines, wrapPDFText(item.Notes, 92)...)
	}
	return simplePDF(lines)
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

func wrapPDFText(text string, max int) []string {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n"))
	if text == "" {
		return nil
	}
	if max <= 0 {
		max = 92
	}
	out := []string{}
	for _, paragraph := range strings.Split(text, "\n") {
		words := strings.Fields(paragraph)
		line := ""
		for _, word := range words {
			if line == "" {
				line = word
				continue
			}
			if len([]rune(line))+1+len([]rune(word)) > max {
				out = append(out, line)
				line = word
				continue
			}
			line += " " + word
		}
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func simplePDF(lines []string) []byte {
	const maxLinesPerPage = 42
	pages := [][]string{}
	for len(lines) > 0 {
		n := maxLinesPerPage
		if len(lines) < n {
			n = len(lines)
		}
		pages = append(pages, append([]string(nil), lines[:n]...))
		lines = lines[n:]
	}
	if len(pages) == 0 {
		pages = [][]string{{"Protokoll"}}
	}
	var objects []string
	objects = append(objects, "<< /Type /Catalog /Pages 2 0 R >>")
	kids := []string{}
	for i := range pages {
		pageObj := 3 + i*2
		kids = append(kids, strconv.Itoa(pageObj)+" 0 R")
	}
	objects = append(objects, fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), len(pages)))
	for i, pageLines := range pages {
		pageObj := 3 + i*2
		contentObj := pageObj + 1
		stream := pdfContentStream(pageLines)
		objects = append(objects, fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 << /Type /Font /Subtype /Type1 /BaseFont /Helvetica >> /F2 << /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold >> >> >> /Contents %d 0 R >>", contentObj))
		objects = append(objects, fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream))
	}
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	offsets := []int{0}
	for i, obj := range objects {
		offsets = append(offsets, buf.Len())
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xref := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n0000000000 65535 f \n", len(offsets))
	for i := 1; i < len(offsets); i++ {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets), xref)
	return buf.Bytes()
}

func pdfContentStream(lines []string) string {
	var b strings.Builder
	b.WriteString("BT\n/F2 18 Tf\n50 790 Td\n")
	for i, line := range lines {
		if i == 1 {
			b.WriteString("/F1 11 Tf\n")
		}
		if i == 3 {
			b.WriteString("/F1 10 Tf\n")
		}
		if i > 0 {
			b.WriteString("0 -17 Td\n")
		}
		b.WriteString("(")
		b.WriteString(pdfEscapeASCII(line))
		b.WriteString(") Tj\n")
	}
	b.WriteString("ET")
	return b.String()
}

func pdfEscapeASCII(raw string) string {
	raw = strings.NewReplacer(
		"Ä", "Ae", "Ö", "Oe", "Ü", "Ue", "ä", "ae", "ö", "oe", "ü", "ue", "ß", "ss", "€", "EUR",
		"–", "-", "—", "-", "·", "-", "„", "\"", "“", "\"", "”", "\"", "’", "'",
	).Replace(raw)
	var b strings.Builder
	for _, r := range raw {
		switch r {
		case '\\', '(', ')':
			b.WriteRune('\\')
			b.WriteRune(r)
		case '\t', '\n', '\r':
			b.WriteRune(' ')
		default:
			if r >= 32 && r <= 126 {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

func logHandoverError(action string, tenantSlug string, id string, err error) {
	log.Printf("handover %s failed for %s/%s: %v", action, tenantSlug, id, err)
}
