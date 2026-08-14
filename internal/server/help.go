package server

import (
	"net/http"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
)

func (a *app) helpPage(w http.ResponseWriter, r *http.Request, ac authCtx) {
	a.renderHelpPage(w, ac, false, "", time.Now())
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
	a.renderHelpPage(w, ac, true, pairingCode, now)
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

func (a *app) renderHelpPage(w http.ResponseWriter, ac authCtx, pairingCreated bool, pairingCode string, now time.Time) {
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
		"PairingExpires": func() string {
			if !pairingCreated {
				return ""
			}
			return formatLocalTime(now.Add(homeConnectorPairingTTL))
		}(),
		"ConnectorLastSeen": func() string {
			if connector.LastSeenAt == nil {
				return ""
			}
			return formatLocalDateTime(*connector.LastSeenAt)
		}(),
		"ConnectorVersion":     connector.ConnectorVersion,
		"HomeAssistantVersion": connector.HomeAssistantVersion,
		"ConnectorEntityCount": connector.EntityCount,
	}
	a.render(w, "help", a.withBase(ac, data))
}
