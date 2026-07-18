package server

import (
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (a *app) contacts(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	if denyServiceProviderArea(w, role) {
		return
	}
	profile := a.profileForTenant(email, tenant.Slug)
	isAdmin := hasCapability(role, capabilityPlatformAdmin)
	canManageContacts := canManageContacts(role)
	managerContacts := managerContactViews(tenant)
	emergencyContacts := emergencyContactViews(tenant)
	managedContacts := a.managedContactViews(tenant.Slug, canManageContacts)
	boardContacts := a.boardContactViews(tenant.Slug)
	residentContacts := a.residentDirectoryViews(tenant.Slug)
	managedEmptyMessage := "Dienstleister, Hausmeister und Notdienste können hier zentral hinterlegt werden."
	if !a.serviceAccessEnabled {
		managedEmptyMessage = "Hausmeister, Notdienste und weitere wichtige Kontakte können hier zentral hinterlegt werden."
	}
	contactMsg, contactOK := contactMessage(r.URL.Query().Get("contact"))
	a.render(w, "contacts", map[string]any{
		"Title":                "Kontakte",
		"Tenant":               tenant,
		"Email":                email,
		"DisplayName":          profile.DisplayName(),
		"Initials":             profile.Initials(),
		"Role":                 role,
		"IsAdmin":              isAdmin,
		"CanSeeParking":        isAdmin || profile.HasPermission(permissionParking),
		"CanManageContacts":    canManageContacts,
		"ActivePage":           "contacts",
		"ContactMsg":           contactMsg,
		"ContactOK":            contactOK,
		"ManagerContacts":      managerContacts,
		"HasManagerContacts":   len(managerContacts) > 0,
		"ManagerEmpty":         emptyState("Kein Verwaltungskontakt", "Der Kontaktblock wird in den Gebäude-Einstellungen gepflegt."),
		"EmergencyContacts":    emergencyContacts,
		"HasEmergencyContacts": len(emergencyContacts) > 0,
		"EmergencyEmpty":       emptyState("Kein Notdienst hinterlegt", "Notdienst und Hausmeister werden in den Gebäude-Einstellungen gepflegt."),
		"ManagedContacts":      managedContacts,
		"HasManagedContacts":   len(managedContacts) > 0,
		"ManagedEmpty":         emptyState("Noch kein Adressbucheintrag", managedEmptyMessage),
		"ContactKindOptions":   contactKindOptionsForServiceProviderAccess("", a.serviceAccessEnabled),
		"BoardContacts":        boardContacts,
		"HasBoardContacts":     len(boardContacts) > 0,
		"BoardEmpty":           emptyState("Kein Beirat hinterlegt", "Beiräte erscheinen hier, sobald sie in Benutzer & Rechte die Beirat-Rolle haben."),
		"ResidentContacts":     residentContacts,
		"HasResidentContacts":  len(residentContacts) > 0,
		"ResidentEmpty":        emptyState("Keine freigegebenen Kontakte", "Kontakte aus der Hausgemeinschaft erscheinen nur nach ausdrücklicher Freigabe im Profil."),
	})
}

func (a *app) upsertManagedContact(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actorEmail, role := ac.tenant, ac.email, ac.role
	if !canManageContacts(role) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	item, err := managedContactFromForm(tenant.Slug, r.Form)
	if err != nil {
		http.Redirect(w, r, "/app/kontakte?contact=invalid", http.StatusSeeOther)
		return
	}
	if !a.serviceAccessEnabled && (normalizeContactKind(item.Kind) == roleServiceProvider || a.isExistingServiceProviderContact(tenant.Slug, item.ID)) {
		http.Error(w, serviceProviderAccessClosedMessage, http.StatusForbidden)
		return
	}
	saved, created, err := a.contactStore.Upsert(item)
	if err != nil {
		log.Printf("contact save failed for %s/%s: %v", tenant.Slug, redactedEmail(item.Email), err)
		http.Redirect(w, r, "/app/kontakte?contact=error", http.StatusSeeOther)
		return
	}
	action := "aktualisiert"
	if created {
		action = "angelegt"
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: actorEmail,
		ActorRole:  role,
		Action:     auditActionContactSave,
		TargetType: "contact",
		TargetID:   saved.ID,
		Summary:    "Adressbuch-Kontakt " + action,
		Details: map[string]string{
			"type":   saved.Kind,
			"status": contactStatusLabel(saved.Active),
		},
	})
	http.Redirect(w, r, "/app/kontakte?contact=saved", http.StatusSeeOther)
}

func (a *app) isExistingServiceProviderContact(tenantSlug string, id string) bool {
	if a == nil || a.contactStore == nil || strings.TrimSpace(id) == "" {
		return false
	}
	for _, item := range a.contactStore.ListTenant(tenantSlug, true) {
		if item.ID == id {
			return normalizeContactKind(item.Kind) == roleServiceProvider
		}
	}
	return false
}

func (a *app) deactivateManagedContact(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actorEmail, role := ac.tenant, ac.email, ac.role
	if !canManageContacts(role) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	id := strings.TrimSpace(r.FormValue("id"))
	removed, err := a.contactStore.Deactivate(tenant.Slug, id, time.Now())
	if err != nil {
		http.Redirect(w, r, "/app/kontakte?contact=error", http.StatusSeeOther)
		return
	}
	if removed.ID != "" {
		a.recordAudit(auditEvent{
			TenantSlug: tenant.Slug,
			ActorEmail: actorEmail,
			ActorRole:  role,
			Action:     auditActionContactDelete,
			TargetType: "contact",
			TargetID:   removed.ID,
			Summary:    "Adressbuch-Kontakt deaktiviert",
			Details: map[string]string{
				"type":   removed.Kind,
				"status": contactStatusLabel(removed.Active),
			},
		})
	}
	http.Redirect(w, r, "/app/kontakte?contact=deleted", http.StatusSeeOther)
}

func contactMessage(status string) (string, bool) {
	switch status {
	case "saved":
		return "Kontakt gespeichert.", true
	case "deleted":
		return "Kontakt deaktiviert.", true
	case "invalid":
		return "Bitte Art, Name/Firma und Kontaktdaten prüfen.", false
	case "error":
		return "Der Kontakt konnte nicht gespeichert werden.", false
	default:
		return "", false
	}
}

func (a *app) managedContactViews(tenantSlug string, includeInactive bool) []managedContactView {
	if a == nil || a.contactStore == nil {
		return nil
	}
	items := a.contactStore.ListTenant(tenantSlug, includeInactive)
	views := make([]managedContactView, 0, len(items))
	for _, item := range items {
		itemView := managedContactViewFrom(item)
		itemView.KindOptions = contactKindOptionsForServiceProviderAccess(item.Kind, a.serviceAccessEnabled)
		views = append(views, itemView)
	}
	return views
}

func (a *app) serviceContactOptions(tenantSlug string) []contactOptionView {
	if a == nil || !a.serviceAccessEnabled || a.contactStore == nil {
		return nil
	}
	items := a.contactStore.ListTenant(tenantSlug, false)
	options := []contactOptionView{}
	for _, item := range items {
		email := normalizeEmail(item.Email)
		if email == "" {
			continue
		}
		label := managedContactDisplayName(item)
		if item.Kind != "" {
			label += " · " + item.Kind
		}
		options = append(options, contactOptionView{Email: email, Label: label})
	}
	return options
}

func contactKindOptionsForServiceProviderAccess(selected string, enabled bool) []selectOption {
	options := contactKindOptions(selected)
	if enabled {
		return options
	}
	filtered := make([]selectOption, 0, len(options))
	for _, option := range options {
		if normalizeContactKind(option.Value) != roleServiceProvider {
			filtered = append(filtered, option)
		}
	}
	return filtered
}

func (a *app) boardContactViews(tenantSlug string) []contactCardView {
	contacts := []contactCardView{}
	for _, row := range a.userRows(tenantSlug) {
		if row.Role != roleBeirat || normalizeEmail(row.Email) == "" {
			continue
		}
		contacts = append(contacts, contactCardView{
			Name:        row.DisplayName,
			Role:        roleBeirat,
			Description: "Beirat",
			Email:       row.Email,
			Phone:       row.Phone,
			HasEmail:    row.Email != "",
			HasPhone:    row.Phone != "",
		})
	}
	return contacts
}

func (a *app) residentDirectoryViews(tenantSlug string) []contactCardView {
	contacts := []contactCardView{}
	for _, row := range a.userRows(tenantSlug) {
		if !row.DirectoryOptIn || !residentDirectoryRole(row.Role) || normalizeEmail(row.Email) == "" {
			continue
		}
		contacts = append(contacts, contactCardView{
			Name:        row.DisplayName,
			Role:        row.Role,
			Description: "Hausgemeinschaft",
			Email:       row.Email,
			Phone:       row.Phone,
			HasEmail:    row.Email != "",
			HasPhone:    row.Phone != "",
		})
	}
	return contacts
}

func managedContactFromForm(tenantSlug string, values url.Values) (managedContact, error) {
	return normalizeManagedContact(managedContact{
		ID:         strings.TrimSpace(values.Get("id")),
		TenantSlug: tenantSlug,
		Kind:       values.Get("kind"),
		Name:       values.Get("name"),
		Company:    values.Get("company"),
		Email:      values.Get("email"),
		Phone:      values.Get("phone"),
		Notes:      values.Get("notes"),
		Active:     values.Get("active") != "false",
	})
}
