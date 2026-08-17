package server

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/version"
	"github.com/inspr-at/hausv-org/internal/view"
	"github.com/inspr-at/hausv-org/internal/web"
)

func (a *app) helpPage(w http.ResponseWriter, r *http.Request, ac authCtx) {
	a.renderHelpPage(w, r, ac, false, "", time.Now())
}

func (a *app) startAppHomeConnectorPairing(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.canManageEnergy(ac) {
		http.Error(w, "Nur Eigentümer, Hausadministration oder freigegebene technische Vertrauenspersonen können die Verbindung vorbereiten.", http.StatusForbidden)
		return
	}
	if a.homeConnectors == nil || len(a.homeConnectorHashKey) == 0 {
		http.Error(w, "Die Energieverbindung ist derzeit nicht verfügbar.", http.StatusServiceUnavailable)
		return
	}
	pairingCode, err := randomToken(32)
	if err != nil {
		http.Error(w, "Kopplung konnte nicht vorbereitet werden", http.StatusInternalServerError)
		return
	}
	now := time.Now()
	if _, err := a.homeConnectors.StartPairing(ac.tenant.Slug,
		homeConnectorHash(a.homeConnectorHashKey, pairingCode), now.Add(homeConnectorPairingTTL), now); err != nil {
		logWarn("app home connector pairing failed", "error_type", "store")
		http.Error(w, "Kopplung konnte nicht vorbereitet werden", http.StatusInternalServerError)
		return
	}
	a.renderHelpPage(w, r, ac, true, pairingCode, now)
}

func (a *app) revokeAppHomeConnector(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.canManageEnergy(ac) {
		http.Error(w, "Nur Eigentümer, Hausadministration oder freigegebene technische Vertrauenspersonen können die Verbindung widerrufen.", http.StatusForbidden)
		return
	}
	if a.homeConnectors == nil {
		http.Error(w, "Die Energieverbindung ist derzeit nicht verfügbar.", http.StatusServiceUnavailable)
		return
	}
	if _, _, err := a.homeConnectors.Revoke(ac.tenant.Slug, time.Now()); err != nil {
		logWarn("app home connector revoke failed", "error_type", "store")
		http.Error(w, "Verbindung konnte nicht widerrufen werden", http.StatusInternalServerError)
		return
	}
	if a.homeConnectorReadings != nil {
		if err := a.homeConnectorReadings.Clear(ac.tenant.Slug); err != nil {
			logWarn("app home connector readings revoke failed", "error_type", "store")
			http.Error(w, "Verbindung konnte nicht widerrufen werden", http.StatusInternalServerError)
			return
		}
	}
	http.Redirect(w, r, "/app/hilfe?connector=revoked", http.StatusSeeOther)
}

func (a *app) renderHelpPage(w http.ResponseWriter, r *http.Request, ac authCtx, pairingCreated bool, pairingCode string, now time.Time) {
	connector := store.HomeConnector{}
	connectorAvailable := a.homeConnectors != nil && len(a.homeConnectorHashKey) > 0
	if connectorAvailable {
		var err error
		connector, _, err = a.homeConnectors.Get(ac.tenant.Slug)
		if err != nil {
			logWarn("app home connector status failed", "error_type", "store")
			connectorAvailable = false
		}
	}

	connected := connector.Status == store.HomeConnectorConnected && connector.LastSeenAt != nil
	fresh := connected && now.Sub(*connector.LastSeenAt) <= homeConnectorFreshFor
	pairingPending := connector.PairingExpiresAt != nil && connector.PairingExpiresAt.After(now)
	state := "Noch nicht verbunden"
	detail := "HAUSV funktioniert auch ohne Energieverbindung. Sie können den Connector später in Ruhe einrichten."
	stateTone := "idle"
	if connected && fresh {
		state = "Verbunden und aktuell"
		detail = "Der lokale Connector meldet sich regelmäßig und liefert ausschließlich die ausgewählten Energiewerte."
		stateTone = "ready"
	} else if connected {
		state = "Verbindung prüfen"
		detail = "Der Connector hat sich länger als 90 Sekunden nicht gemeldet. Prüfen Sie den lokalen Prozess und die Internetverbindung."
		stateTone = "attention"
	} else if connector.Status == store.HomeConnectorRevoked {
		state = "Verbindung widerrufen"
		detail = "Der frühere Zugang ist ungültig und die zuletzt gepufferten Messwerte wurden entfernt."
		stateTone = "idle"
	} else if pairingPending {
		state = "Wartet auf Kopplung"
		detail = "Ein Einmal-Code ist noch gültig. Falls er nicht mehr vorliegt, erzeugen Sie einfach einen neuen."
		stateTone = "attention"
	}
	if !connectorAvailable {
		state = "Derzeit nicht verfügbar"
		detail = "Die Connector-Funktion ist in dieser Installation nicht aktiviert."
		stateTone = "attention"
	}
	pairingExpires := ""
	if pairingCreated {
		pairingExpires = formatLocalTime(now.Add(homeConnectorPairingTTL))
	}
	connectorLastSeen := ""
	if connector.LastSeenAt != nil {
		connectorLastSeen = formatLocalDateTime(*connector.LastSeenAt)
	}

	if a.portalTemplEnabled {
		modules := a.portalModulesFor(ac.tenant.Slug)
		a.renderHelpTempl(w, r, web.HelpPageData{
			Portal:                  a.helpPortalContext(ac),
			ConnectorAvailable:      connectorAvailable,
			ConnectorState:          state,
			ConnectorStateTone:      stateTone,
			ConnectorDetail:         detail,
			ConnectorConnected:      connected,
			ConnectorFresh:          fresh,
			ConnectorPairingPending: pairingPending,
			PairingCreated:          pairingCreated,
			PairingCode:             pairingCode,
			PairingExpires:          pairingExpires,
			ConnectorLastSeen:       connectorLastSeen,
			ConnectorVersion:        connector.ConnectorVersion,
			HomeAssistantVersion:    connector.HomeAssistantVersion,
			ConnectorEntityCount:    connector.EntityCount,
			CanManageEnergy:         modules.Energy && a.canManageEnergy(ac),
		})
		return
	}

	data := map[string]any{
		"Title":                   "Hilfe",
		"ActivePage":              "help",
		"ConnectorAvailable":      connectorAvailable,
		"ConnectorState":          state,
		"ConnectorStateTone":      stateTone,
		"ConnectorDetail":         detail,
		"ConnectorConnected":      connected,
		"ConnectorFresh":          fresh,
		"ConnectorPairingPending": pairingPending,
		"PairingCreated":          pairingCreated,
		"PairingCode":             pairingCode,
		"PairingExpires":          pairingExpires,
		"ConnectorLastSeen":       connectorLastSeen,
		"ConnectorVersion":        connector.ConnectorVersion,
		"HomeAssistantVersion":    connector.HomeAssistantVersion,
		"ConnectorEntityCount":    connector.EntityCount,
	}
	a.render(w, "help", a.withBase(ac, data))
}

func (a *app) helpPortalContext(ac authCtx) web.PortalPageData {
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
		Title:               "Hilfe · " + houseDisplayName(tenant) + " · " + role,
		TenantSlug:          tenant.Slug,
		HouseName:           houseDisplayName(tenant),
		Address:             tenant.Address,
		MapURL:              tenantMapURL(tenant.Address),
		DisplayName:         profile.DisplayName(),
		Initials:            profile.Initials(),
		Role:                role,
		DisplayVersion:      version.DisplayVersion(version.Version),
		ActivePage:          "help",
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

func (a *app) renderHelpTempl(w http.ResponseWriter, r *http.Request, data web.HelpPageData) {
	var rendered bytes.Buffer
	if err := web.HelpPage(data).Render(r.Context(), &rendered); err != nil {
		logError("templ help render failed", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, prefixTenantHTMLPaths(rendered.String(), data.Portal.TenantSlug))
}
