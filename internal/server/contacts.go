package server

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/authz"
	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/version"
	"github.com/inspr-at/hausv-org/internal/web"
)

func (a *app) contacts(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, role := ac.tenant, ac.role
	if denyServiceProviderArea(w, role) {
		return
	}
	profile := a.profileForTenant(ac.email, tenant.Slug)
	canManageContacts := canManageContacts(ac.actor(), ac.resource())
	managerContacts := managerContactViews(tenant)
	emergencyContacts := emergencyContactViews(tenant)
	contacts := ac.repositories.contacts
	managedContacts := a.managedContactViews(contacts, canManageContacts)
	activeManagedContacts, inactiveManagedContacts := splitManagedContactViews(managedContacts)
	boardContacts := a.boardContactViews(ac.tenantRef)
	residentContacts := a.residentDirectoryViews(ac.tenantRef)
	canViewUnitOccupancies := authz.Can(ac.actor(), authz.CapabilityManageBuilding, ac.resource()) || authz.Can(ac.actor(), authz.CapabilityOversight, ac.resource())
	managedEmptyTitle := "Noch kein Adressbucheintrag"
	managedEmptyMessage := "Dienstleister, Hausmeister und Notdienste können hier zentral hinterlegt werden."
	if !a.serviceAccessEnabled {
		managedEmptyMessage = "Hausmeister, Notdienste und weitere wichtige Kontakte können hier zentral hinterlegt werden."
	}
	if !canManageContacts {
		managedEmptyTitle = "Noch keine Kontakte hinterlegt"
		managedEmptyMessage = "Die Hausverwaltung hat für dieses Haus noch keine allgemeinen Kontakte hinterlegt. Verwaltung, Notdienst und Hausmeister trägt die Verwaltung ein."
	}
	managedEmpty := emptyState(managedEmptyTitle, managedEmptyMessage)
	contactMsg, contactOK := contactMessage(r.URL.Query().Get("contact"))
	groups := groupManagedContactsByKind(activeManagedContacts)
	templGroups := make([]web.ContactKindGroup, 0, len(groups))
	for _, group := range groups {
		templGroups = append(templGroups, web.ContactKindGroup{Kind: group.Kind, Contacts: group.Contacts})
	}
	a.renderContactsTempl(w, r, web.ContactsPageData{
		Portal:                       a.contactsPortalContext(ac),
		AssetVersion:                 version.AssetVersion(),
		ContactMessage:               contactMsg,
		ContactOK:                    contactOK,
		CanManageContacts:            canManageContacts,
		CanManageIssues:              ac.can(capabilityManageIssues),
		CanJoinDirectory:             residentDirectoryRole(role),
		DirectoryOptIn:               profile.DirectoryOptIn,
		ServiceProviderAccessEnabled: a.serviceAccessEnabled,
		ContactFormOpen:              r.URL.Query().Get("contact") == "invalid" || r.URL.Query().Get("contact") == "error",
		ManagerContacts:              managerContacts,
		EmergencyContacts:            emergencyContacts,
		ManagedContacts:              activeManagedContacts,
		ManagedGroups:                templGroups,
		InactiveContacts:             inactiveManagedContacts,
		BoardContacts:                boardContacts,
		ResidentContacts:             residentContacts,
		CanViewUnitOccupancies:       canViewUnitOccupancies,
		UnitOccupancies:              a.unitOccupancies(ac.tenantRef),
		ContactKindOptions:           contactKindOptionsForServiceProviderAccess("", a.serviceAccessEnabled),
		ManagedEmpty:                 managedEmpty,
	})
}

func (a *app) contactsPortalContext(ac authCtx) web.PortalPageData {
	data := a.eventsPortalContext(ac)
	data.Title = "Kontakte · " + houseDisplayName(ac.tenant) + " · " + ac.role
	data.ActivePage = "contacts"
	return data
}

func (a *app) renderContactsTempl(w http.ResponseWriter, r *http.Request, data web.ContactsPageData) {
	var rendered bytes.Buffer
	if err := web.ContactsPage(data).Render(r.Context(), &rendered); err != nil {
		logError("templ contacts render failed", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, prefixTenantHTMLPaths(rendered.String(), data.Portal.TenantSlug))
}

func (a *app) upsertManagedContact(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actorEmail, role := ac.tenant, ac.email, ac.role
	if !canManageContacts(ac.actor(), ac.resource()) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	item, err := managedContactFromForm(tenant.Slug, r.Form)
	if err != nil {
		http.Redirect(w, r, "/app/kontakte?contact=invalid#contact-add", http.StatusSeeOther)
		return
	}
	contacts := ac.repositories.contacts
	if !a.serviceAccessEnabled && (normalizeContactKind(item.Kind) == roleServiceProvider || a.isExistingServiceProviderContact(contacts, item.ID)) {
		http.Error(w, serviceProviderAccessClosedMessage, http.StatusForbidden)
		return
	}
	saved, created, err := contacts.Upsert(item)
	if err != nil {
		logError("contact save failed", err, "tenant", tenant.Slug, "contact", redactedEmail(item.Email))
		http.Redirect(w, r, "/app/kontakte?contact=error#contact-book", http.StatusSeeOther)
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
	http.Redirect(w, r, "/app/kontakte?contact=saved#contact-book", http.StatusSeeOther)
}

func (a *app) isExistingServiceProviderContact(contacts contactBookRepository, id string) bool {
	if a == nil || contacts == nil || strings.TrimSpace(id) == "" {
		return false
	}
	for _, item := range contacts.List(true) {
		if item.ID == id {
			return normalizeContactKind(item.Kind) == roleServiceProvider
		}
	}
	return false
}

func (a *app) deactivateManagedContact(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actorEmail, role := ac.tenant, ac.email, ac.role
	if !canManageContacts(ac.actor(), ac.resource()) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	id := strings.TrimSpace(r.FormValue("id"))
	removed, err := ac.repositories.contacts.Deactivate(id, time.Now())
	if err != nil {
		http.Redirect(w, r, "/app/kontakte?contact=error#contact-book", http.StatusSeeOther)
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
	http.Redirect(w, r, "/app/kontakte?contact=deleted#contact-book", http.StatusSeeOther)
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

func (a *app) managedContactViews(contacts contactBookRepository, includeInactive bool) []managedContactView {
	if a == nil || contacts == nil {
		return nil
	}
	items := contacts.List(includeInactive)
	views := make([]managedContactView, 0, len(items))
	for _, item := range items {
		itemView := managedContactViewFrom(item)
		itemView.KindOptions = contactKindOptionsForServiceProviderAccess(item.Kind, a.serviceAccessEnabled)
		itemView.CanEdit = a.serviceAccessEnabled || normalizeContactKind(item.Kind) != roleServiceProvider
		views = append(views, itemView)
	}
	return views
}

// contactKindGroup turns the address book into a directory: entries are read by
// function ("who repairs the lift?"), so they are grouped by contact kind in a
// fixed urgency-first order instead of one undifferentiated list.
type contactKindGroup struct {
	Kind     string
	Count    int
	Contacts []managedContactView
}

func groupManagedContactsByKind(items []managedContactView) []contactKindGroup {
	order := []string{"Notdienst", "Hausmeister", "Verwaltung", "Energie-Fachbetrieb", "Dienstleister", "Sonstiges"}
	rank := map[string]int{}
	for index, kind := range order {
		rank[kind] = index
	}
	groups := make([]contactKindGroup, 0, len(order))
	index := map[string]int{}
	for _, item := range items {
		kind := strings.TrimSpace(item.Kind)
		if kind == "" {
			kind = "Sonstiges"
		}
		position, ok := index[kind]
		if !ok {
			groups = append(groups, contactKindGroup{Kind: kind})
			position = len(groups) - 1
			index[kind] = position
		}
		groups[position].Contacts = append(groups[position].Contacts, item)
		groups[position].Count = len(groups[position].Contacts)
	}
	sort.SliceStable(groups, func(i, j int) bool {
		left, leftKnown := rank[groups[i].Kind]
		right, rightKnown := rank[groups[j].Kind]
		if leftKnown != rightKnown {
			return leftKnown
		}
		if leftKnown {
			return left < right
		}
		return groups[i].Kind < groups[j].Kind
	})
	return groups
}

func splitManagedContactViews(items []managedContactView) ([]managedContactView, []managedContactView) {
	active := make([]managedContactView, 0, len(items))
	inactive := make([]managedContactView, 0)
	for _, item := range items {
		if item.Active {
			active = append(active, item)
		} else {
			inactive = append(inactive, item)
		}
	}
	return active, inactive
}

func (a *app) serviceContactOptions(contacts contactBookRepository) []contactOptionView {
	if a == nil || !a.serviceAccessEnabled || contacts == nil {
		return nil
	}
	items := contacts.List(false)
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

func (a *app) boardContactViews(tenant store.TenantRef) []contactCardView {
	contacts := []contactCardView{}
	for _, row := range a.userRows(tenant) {
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

func (a *app) residentDirectoryViews(tenant store.TenantRef) []contactCardView {
	contacts := []contactCardView{}
	for _, row := range a.userRows(tenant) {
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
		ID:                 strings.TrimSpace(values.Get("id")),
		TenantSlug:         tenantSlug,
		Kind:               values.Get("kind"),
		Name:               values.Get("name"),
		Company:            values.Get("company"),
		Email:              values.Get("email"),
		Phone:              values.Get("phone"),
		Notes:              values.Get("notes"),
		ServiceRegion:      values.Get("service_region"),
		Qualification:      values.Get("qualification"),
		EnergyCapabilities: values["energy_capabilities"],
		Active:             values.Get("active") != "false",
	})
}
