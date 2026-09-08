package server

import (
	"net/http"
	"sort"
	"strings"

	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/view"
	"github.com/inspr-at/hausv-org/internal/web"
)

type portalModuleID string

const (
	portalModuleEnergy        portalModuleID = "energy"
	portalModuleAnnouncements portalModuleID = "announcements"
	portalModuleEvents        portalModuleID = "events"
	portalModuleContacts      portalModuleID = "contacts"
	portalModuleDocuments     portalModuleID = "documents"
	portalModuleIssues        portalModuleID = "issues"
	portalModuleVotes         portalModuleID = "votes"
	portalModuleParking       portalModuleID = "parking"
	portalModuleHandovers     portalModuleID = "handovers"
	portalModuleUsers         portalModuleID = "users"
	portalModuleAudit         portalModuleID = "audit"
	portalModuleHelp          portalModuleID = "help"
)

type portalModuleDefinition struct {
	ID          portalModuleID
	Label       string
	Description string
	Icon        string
}

func notificationOptionsForModules(options []view.NotificationEventOption, modules portalModuleFlags) []view.NotificationEventOption {
	out := make([]view.NotificationEventOption, 0, len(options))
	for _, option := range options {
		enabled := true
		switch option.Key {
		case store.NotificationEventAnnouncement:
			enabled = modules.Announcements
		case store.NotificationEventIssue:
			enabled = modules.Issues
		case store.NotificationEventVote:
			enabled = modules.Votes
		case store.NotificationEventDocument:
			enabled = modules.Documents
		case store.NotificationEventPayment:
			enabled = modules.Contacts
		case store.NotificationEventCharging:
			enabled = modules.Parking
		}
		if enabled {
			out = append(out, option)
		}
	}
	return out
}

var portalModuleCatalog = []portalModuleDefinition{
	{portalModuleEnergy, "Energie", "Energiefluss, Messwerte und Verbraucher", "energy"},
	{portalModuleAnnouncements, "Aushang", "Mitteilungen für die Liegenschaft", "announcement"},
	{portalModuleEvents, "Termine", "Kalender, Wartungen und Versammlungen", "calendar"},
	{portalModuleContacts, "Kontakte", "Verwaltung, Beirat und Dienstleister", "contact"},
	{portalModuleDocuments, "Dokumente", "Unterlagen, Vorschau und Download", "document"},
	{portalModuleIssues, "Anliegen", "Meldungen, Rückfragen und Bearbeitung", "issue"},
	{portalModuleVotes, "Abstimmungen", "Beschlüsse und nachvollziehbare Ergebnisse", "vote"},
	{portalModuleParking, "Parkplatznutzung", "Laden, Berechtigungen und Monatswerte", "parking"},
	{portalModuleHandovers, "Übergaben", "Digitale Übergabeprotokolle", "handover"},
	{portalModuleUsers, "Benutzer & Rechte", "Einladungen, Rollen und Zugänge", "users"},
	{portalModuleAudit, "Verlauf", "Änderungen und Zugriffe nachvollziehen", "audit"},
	{portalModuleHelp, "Hilfe", "Anleitungen und Connector-Erklärung", "help"},
}

type portalModuleFlags struct {
	Energy        bool
	Announcements bool
	Events        bool
	Contacts      bool
	Documents     bool
	Issues        bool
	Votes         bool
	Parking       bool
	Handovers     bool
	Users         bool
	Audit         bool
	Help          bool
}

func (f portalModuleFlags) Enabled(module portalModuleID) bool {
	switch module {
	case portalModuleEnergy:
		return f.Energy
	case portalModuleAnnouncements:
		return f.Announcements
	case portalModuleEvents:
		return f.Events
	case portalModuleContacts:
		return f.Contacts
	case portalModuleDocuments:
		return f.Documents
	case portalModuleIssues:
		return f.Issues
	case portalModuleVotes:
		return f.Votes
	case portalModuleParking:
		return f.Parking
	case portalModuleHandovers:
		return f.Handovers
	case portalModuleUsers:
		return f.Users
	case portalModuleAudit:
		return f.Audit
	case portalModuleHelp:
		return f.Help
	default:
		return false
	}
}

func allPortalModulesEnabled() portalModuleFlags {
	return portalModuleFlags{true, true, true, true, true, true, true, true, true, true, true, true}
}

func (a *app) portalModulesFor(tenantSlug string) portalModuleFlags {
	flags := allPortalModulesEnabled()
	if a == nil || a.tenantOverrides == nil {
		return flags
	}
	override, ok := a.tenantOverrides.Get(tenantSlug)
	if !ok {
		return flags
	}
	for _, raw := range override.DisabledModules {
		setPortalModule(&flags, portalModuleID(raw), false)
	}
	return flags
}

func setPortalModule(flags *portalModuleFlags, module portalModuleID, enabled bool) {
	switch module {
	case portalModuleEnergy:
		flags.Energy = enabled
	case portalModuleAnnouncements:
		flags.Announcements = enabled
	case portalModuleEvents:
		flags.Events = enabled
	case portalModuleContacts:
		flags.Contacts = enabled
	case portalModuleDocuments:
		flags.Documents = enabled
	case portalModuleIssues:
		flags.Issues = enabled
	case portalModuleVotes:
		flags.Votes = enabled
	case portalModuleParking:
		flags.Parking = enabled
	case portalModuleHandovers:
		flags.Handovers = enabled
	case portalModuleUsers:
		flags.Users = enabled
	case portalModuleAudit:
		flags.Audit = enabled
	case portalModuleHelp:
		flags.Help = enabled
	}
}

func normalizeDisabledPortalModules(raw []string) []string {
	known := map[string]bool{}
	for _, item := range portalModuleCatalog {
		known[string(item.ID)] = true
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		item = strings.ToLower(strings.TrimSpace(item))
		if !known[item] || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func portalModuleForPath(path string) (portalModuleID, bool) {
	path = strings.TrimSuffix(strings.TrimSpace(path), "/")
	prefix := func(value string) bool { return path == value || strings.HasPrefix(path, value+"/") }
	switch {
	case prefix("/app/energie"), prefix("/app/zuhause/onboarding"), prefix("/app/settings/energy-data"), prefix("/app/settings/home"):
		return portalModuleEnergy, true
	case prefix("/app/announcements"):
		return portalModuleAnnouncements, true
	case prefix("/app/events"):
		return portalModuleEvents, true
	case prefix("/app/kontakte"):
		return portalModuleContacts, true
	case prefix("/app/dokumente"):
		return portalModuleDocuments, true
	case prefix("/app/anliegen"):
		return portalModuleIssues, true
	case prefix("/app/abstimmungen"):
		return portalModuleVotes, true
	case prefix("/app/parking"), prefix("/app/settings/parking-access"):
		return portalModuleParking, true
	case prefix("/app/uebergaben"):
		return portalModuleHandovers, true
	case prefix("/app/settings/users"):
		return portalModuleUsers, true
	case prefix("/app/audit"):
		return portalModuleAudit, true
	case prefix("/app/hilfe"):
		return portalModuleHelp, true
	default:
		return "", false
	}
}

type portalModuleOptionView = web.PortalModuleOptionView

func (a *app) portalModuleSettings(w http.ResponseWriter, r *http.Request, ac authCtx) {
	flags := a.portalModulesFor(ac.tenant.Slug)
	options := make([]portalModuleOptionView, 0, len(portalModuleCatalog))
	for _, item := range portalModuleCatalog {
		options = append(options, portalModuleOptionView{
			ID: string(item.ID), Label: item.Label, Description: item.Description,
			Icon: item.Icon, Enabled: flags.Enabled(item.ID),
		})
	}
	a.renderSettingsComponent(w, r, ac.tenant.Slug, web.PortalModuleSettingsPage(web.PortalModuleSettingsPageData{
		Portal:              a.settingsPortalContext(ac, "Portalbereiche", "settings"),
		HouseName:           houseDisplayName(ac.tenant),
		PortalModuleOptions: options,
		PortalModulesSaved:  r.URL.Query().Get("saved") == "1",
		PortalModulesError:  r.URL.Query().Get("error") == "1",
	}))
}

func (a *app) updatePortalModules(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	enabled := map[string]bool{}
	for _, raw := range r.Form["modules"] {
		enabled[strings.ToLower(strings.TrimSpace(raw))] = true
	}
	disabled := make([]string, 0, len(portalModuleCatalog))
	for _, item := range portalModuleCatalog {
		if !enabled[string(item.ID)] {
			disabled = append(disabled, string(item.ID))
		}
	}
	if a.tenantOverrides != nil {
		if err := a.tenantOverrides.SetDisabledModules(ac.tenant.Slug, disabled); err != nil {
			http.Redirect(w, r, "/app/settings/modules?error=1", http.StatusSeeOther)
			return
		}
	}
	a.recordAudit(auditEvent{
		TenantSlug: ac.tenant.Slug, ActorEmail: ac.email, ActorRole: ac.role,
		Action: auditActionPortalModulesUpdate, TargetType: "building", TargetID: ac.tenant.Slug,
		Summary: "Portalbereiche geändert",
		Details: map[string]string{"disabled_modules": strings.Join(disabled, ", ")},
	})
	http.Redirect(w, r, "/app/settings/modules?saved=1", http.StatusSeeOther)
}
