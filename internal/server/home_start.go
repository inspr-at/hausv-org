package server

import (
	"context"
	"crypto/sha256"
	"errors"
	"net/http"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/inspr-at/hausv-org/internal/store"
)

const (
	homeSetupCookieName  = "hausv_home_setup"
	homeSetupLinkTTL     = 15 * time.Minute
	homeSetupSessionTTL  = 30 * time.Minute
	maxHomeStartBody     = 16 << 10
	homePendingRetention = 24 * time.Hour
	homeRetentionSweep   = time.Hour
)

func homeSetupSecret(secret []byte) []byte {
	digest := sha256.Sum256(append([]byte("hausv-home-setup-v1\x00"), secret...))
	return digest[:]
}

var (
	homeStartSourcePolicy  = authRatePolicy{scope: "home-start-source", limit: 10, window: authRateWindow}
	homeStartAccountPolicy = authRatePolicy{scope: "home-start-account", limit: 3, window: authRateWindow}
	homePathPattern        = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{1,30}[a-z0-9])?$`)
	homeReservedPaths      = map[string]struct{}{
		"app": {}, "assets": {}, "auth": {}, "calendar": {}, "datenschutz": {}, "favicon.ico": {},
		"favicon.svg": {}, "handover": {}, "healthz": {}, "impressum": {}, "map-tiles": {}, "start": {},
		"tenant-hero": {}, "www": {},
	}
)

func (a *app) homeStartPage(w http.ResponseWriter, r *http.Request) {
	if !a.isMarketingHost(r) {
		http.NotFound(w, r)
		return
	}
	a.render(w, "homeStart", map[string]any{
		"Title":   "HAUSV Home einrichten",
		"Sent":    r.URL.Query().Get("sent") == "1",
		"Expired": r.URL.Query().Get("link") == "expired",
	})
}

func (a *app) requestHomeStart(w http.ResponseWriter, r *http.Request) {
	if !a.isMarketingHost(r) {
		http.NotFound(w, r)
		return
	}
	if !sameOriginPost(r) {
		http.Error(w, "Bad request", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxHomeStartBody)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	email := normalizeEmail(r.FormValue("email"))
	slug := normalizeSlug(r.FormValue("slug"))
	accountKey := slug + "|" + firstNonEmpty(email, r.FormValue("email"))
	if !a.allowAuthRequest(w, r, homeStartSourcePolicy, homeStartAccountPolicy, accountKey) {
		return
	}
	if !validHomeStartName(r.FormValue("household_name")) || !a.homePathAvailable(slug) ||
		r.FormValue("authority") != "1" || !validHomeStartEmail(email) || a.homeReservations == nil ||
		a.homeSetupTokens == nil || a.homeSetupSessions == nil || a.mailer == nil || !a.mailer.Configured() {
		http.Redirect(w, r, "/start?sent=1", http.StatusSeeOther)
		return
	}

	reservation, err := a.homeReservations.Reserve(store.HomeReservation{
		Slug:                   slug,
		HouseholdName:          strings.TrimSpace(r.FormValue("household_name")),
		OwnerEmail:             email,
		AuthorizationConfirmed: true,
	}, time.Now())
	if errors.Is(err, store.ErrHomeReservationConflict) {
		http.Redirect(w, r, "/start?sent=1", http.StatusSeeOther)
		return
	}
	if err != nil {
		logWarn("home reservation failed", "error_type", "store")
		http.Redirect(w, r, "/start?sent=1", http.StatusSeeOther)
		return
	}

	token, err := randomToken(32)
	if err != nil {
		logWarn("home confirmation creation failed", "error_type", "random_token")
		http.Redirect(w, r, "/start?sent=1", http.StatusSeeOther)
		return
	}
	a.homeSetupTokens.Put(token, reservation.OwnerEmail, reservation.Slug, homeSetupLinkTTL)
	link := strings.TrimRight(a.baseURL, "/") + "/start/verify?token=" + url.QueryEscape(token)
	if !a.enqueueMagicLinkDelivery(magicLinkDeliveryJob{
		mailer:           a.mailer,
		to:               reservation.OwnerEmail,
		link:             link,
		address:          reservation.HouseholdName,
		homeConfirmation: true,
		invalidate: func() {
			a.homeSetupTokens.Invalidate(token)
		},
	}) {
		a.homeSetupTokens.Invalidate(token)
		logWarn("home confirmation delivery not queued")
	}
	http.Redirect(w, r, "/start?sent=1", http.StatusSeeOther)
}

func (a *app) verifyHomeStart(w http.ResponseWriter, r *http.Request) {
	if !a.isMarketingHost(r) || a.homeSetupTokens == nil || a.homeSetupSessions == nil || a.homeReservations == nil {
		http.NotFound(w, r)
		return
	}
	email, slug, _, ok := a.homeSetupTokens.Consume(r.URL.Query().Get("token"))
	if !ok {
		http.Redirect(w, r, "/start?link=expired", http.StatusSeeOther)
		return
	}
	reservation, confirmed, err := a.homeReservations.Confirm(slug, email, time.Now())
	if err != nil || !confirmed {
		logWarn("home confirmation failed", "error_type", "store")
		http.Redirect(w, r, "/start?link=expired", http.StatusSeeOther)
		return
	}
	sessionToken, expiresAt, err := a.homeSetupSessions.Put(reservation.OwnerEmail, reservation.Slug, authMethodEmail, homeSetupSessionTTL)
	if err != nil {
		http.Error(w, "Could not create setup session", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: homeSetupCookieName, Value: sessionToken, Path: "/start", Expires: expiresAt,
		HttpOnly: true, Secure: a.sessionSecure, SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/start/connector", http.StatusSeeOther)
}

func (a *app) homeConnectorStart(w http.ResponseWriter, r *http.Request) {
	reservation, ok := a.homeSetupReservation(r)
	if !ok || a.homeConnectors == nil {
		http.Redirect(w, r, "/start", http.StatusSeeOther)
		return
	}
	connector, _, err := a.homeConnectors.Get(reservation.Slug)
	if err != nil {
		logWarn("home connector status failed", "error_type", "store")
		http.Error(w, "Verbindungsstatus konnte nicht geladen werden", http.StatusInternalServerError)
		return
	}
	a.renderHomeConnectorStart(w, reservation, connector, false, "", time.Now())
}

func (a *app) renderHomeConnectorStart(w http.ResponseWriter, reservation store.HomeReservation, connector store.HomeConnector, pairingCreated bool, pairingCode string, now time.Time) {
	connected := connector.Status == store.HomeConnectorConnected && connector.LastSeenAt != nil
	fresh := connected && now.Sub(*connector.LastSeenAt) <= homeConnectorFreshFor
	state := "Noch nicht gekoppelt"
	detail := "Der lokale Connector hat sich noch nicht bei HAUSV gemeldet."
	if connected && fresh {
		state = "Verbunden · nur lesen"
		detail = "Die lokale Verbindung ist aktuell. Messwerte werden in diesem Schritt noch nicht übernommen."
	} else if connected {
		state = "Verbindung nicht aktuell"
		detail = "Der lokale Connector hat sich länger nicht gemeldet. Bestehende Zugangsdaten bleiben unverändert."
	} else if connector.Status == store.HomeConnectorRevoked {
		state = "Verbindung widerrufen"
		detail = "Der bisherige Connector-Zugang ist nicht mehr gültig."
	}
	pairingPending := connector.PairingExpiresAt != nil && connector.PairingExpiresAt.After(now)
	data := map[string]any{
		"Title":                "Zuhause bestätigt",
		"HouseholdName":        reservation.HouseholdName,
		"PublicPath":           "/" + reservation.Slug,
		"ConnectorState":       state,
		"ConnectorDetail":      detail,
		"Connected":            connected,
		"Fresh":                fresh,
		"PairingPending":       pairingPending,
		"PairingCreated":       pairingCreated,
		"PairingCode":          pairingCode,
		"EntityCount":          connector.EntityCount,
		"HomeAssistantVersion": connector.HomeAssistantVersion,
		"LastSeen": func() string {
			if connector.LastSeenAt == nil {
				return ""
			}
			return formatLocalDateTime(*connector.LastSeenAt)
		}(),
	}
	if pairingCreated {
		data["PairingExpires"] = formatLocalTime(now.Add(homeConnectorPairingTTL))
	}
	a.render(w, "homeConnectorStart", data)
}

func (a *app) homePathAvailable(slug string) bool {
	if !homePathPattern.MatchString(slug) {
		return false
	}
	if _, reserved := homeReservedPaths[slug]; reserved {
		return false
	}
	_, configured := a.tenantBySlug(slug)
	return !configured
}

func validHomeStartName(value string) bool {
	value = strings.TrimSpace(value)
	if utf8.RuneCountInString(value) < 2 || utf8.RuneCountInString(value) > 80 {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func validHomeStartEmail(value string) bool {
	parsed, err := mail.ParseAddress(value)
	return err == nil && normalizeEmail(parsed.Address) == value && len(value) <= 254
}

func (a *app) purgeExpiredHomeReservations(now time.Time) {
	if a == nil || a.homeReservations == nil {
		return
	}
	removed, err := a.homeReservations.PurgePendingBefore(now.Add(-homePendingRetention))
	if err != nil {
		logError("expired home reservation purge failed", err)
		return
	}
	if removed > 0 {
		logInfo("expired home reservations purged", "count", removed)
	}
}

// StartHomeReservationRetentionWorker enforces the published 24-hour maximum
// for unconfirmed path reservations independently of deploys or page traffic.
func (a *app) StartHomeReservationRetentionWorker() func() {
	if a == nil || a.homeReservations == nil {
		return func() {}
	}
	a.purgeExpiredHomeReservations(time.Now())
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(homeRetentionSweep)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				a.purgeExpiredHomeReservations(now)
			}
		}
	}()
	return cancel
}
