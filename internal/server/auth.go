package server

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"

	oidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/inspr-at/hausv-org/internal/auth"
	"github.com/inspr-at/hausv-org/internal/config"
	"golang.org/x/oauth2"
)

type publicHomeCopy struct {
	Eyebrow       string
	Headline      string
	Lead          string
	FirstDetail   string
	SecondDetail  string
	PrivacyDetail string
}

func (a *app) home(w http.ResponseWriter, r *http.Request) {
	if a.isMarketingHost(r) {
		a.marketingLanding(w, r)
		return
	}
	tenant := a.tenantForRequest(r)
	email, _, tenantSlug, ok := a.currentUser(r)
	if ok && tenantSlug == tenant.Slug {
		http.Redirect(w, r, "/app", http.StatusSeeOther)
		return
	}
	houseName := houseDisplayName(tenant)
	copy := a.publicHomeCopy(tenant.Slug)
	a.render(w, "home", map[string]any{
		"Title":               houseName + " · Hausportal",
		"Tenant":              tenant,
		"HouseName":           houseName,
		"Email":               email,
		"Sent":                r.URL.Query().Get("sent") == "1",
		"Expired":             r.URL.Query().Get("login") == "expired",
		"MailConfigured":      a.mailer.Configured(),
		"DevLoginLink":        "",
		"Denied":              r.URL.Query().Get("denied") == "1",
		"OIDCConfigured":      a.oidc.Configured(),
		"EmailLoginAvailable": a.emailLoginAvailable(),
		"DemoLoginEnabled":    a.demoLogin,
		"MapURL":              tenantMapURL(tenant.Address),
		"LocationMap":         publicMapForTenant(tenant),
		"HomeCopy":            copy,
	})
}

func (a *app) marketingLanding(w http.ResponseWriter, r *http.Request) {
	a.render(w, "landing", map[string]any{
		"Title":          "hausv.org - Free, Home und Professional",
		"ContactLocal":   "hello",
		"ContactDomain":  "hausv.org",
		"ContactDisplay": "hello [at] hausv [dot] org",
		"PrimaryAppURL":  primaryAppURL(),
		"RequestedHost":  normalizeHost(r.Host),
		"LandingHeroURL": "/assets/hausv-landing-hero.png",
		"OperatorName":   platformOperatorName(),
	})
}

func (a *app) imprintPage(w http.ResponseWriter, r *http.Request) {
	a.render(w, "imprint", map[string]any{
		"Title":                      "Impressum & Infos · hausv.org",
		"ContactLocal":               "hello",
		"ContactDomain":              "hausv.org",
		"ContactDisplay":             "hello [at] hausv [dot] org",
		"OperatorName":               platformOperatorName(),
		"OperatorAddress":            platformOperatorAddress(),
		"ProfessionalServicesNotice": professionalServicesNotice(),
		"LegalReviewDate":            legalReviewDate,
	})
}

func (a *app) privacyNotice(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	contactEmail := firstNonEmpty(tenant.ContactEmail, platformContactEmail)
	energyProfileExists := false
	if a.energyStore != nil {
		if profile, exists, err := a.energyStore.Profile(tenant.Slug); err == nil && exists {
			if !energyProfileUnclaimed(profile) {
				energyProfileExists = true
			}
		}
	}
	portalType, portalClassified := classifiedPortalType(tenant)
	a.render(w, "privacy", map[string]any{
		"Title":                         "Datenschutz · hausv.org",
		"Tenant":                        tenant,
		"HouseContactName":              firstNonEmpty(tenant.ContactName, tenant.Name, "Hausverwaltung"),
		"HouseContactAddress":           tenant.ContactAddress,
		"HouseContactEmail":             contactEmail,
		"HouseContactPhone":             tenant.ContactPhone,
		"TechnicalOperatorName":         platformOperatorName(),
		"TechnicalOperatorAddress":      platformOperatorAddress(),
		"TechnicalContactEmail":         platformContactEmail,
		"IdentityStorageNotice":         identityStorageNotice(),
		"BackupStorageNotice":           backupStorageNotice(),
		"WebAccessNotice":               webAccessNotice(),
		"MailDeliveryNotice":            mailDeliveryNotice(),
		"LegalReviewDate":               legalReviewDate,
		"ServiceProviderEnabled":        a.serviceAccessEnabled,
		"ServiceProviderAssessment":     serviceProviderAssessmentVersion,
		"ServiceProviderRetentionYears": 3,
		"IsPrivateHome":                 portalType == config.PortalTypeApartment || portalType == config.PortalTypeHouse,
		"PortalClassified":              portalClassified,
		"EnergyProfileExists":           energyProfileExists,
	})
}

func (a *app) requestLogin(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	if !sameOriginPost(r) {
		http.Error(w, "Bad request", http.StatusForbidden)
		return
	}
	if err := parseMaybeMultipartForm(w, r, maxIssueAttachmentFormBytes, maxAttachmentBytes); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	email := normalizeEmail(r.FormValue("email"))
	accountKey := tenant.Slug + "|" + firstNonEmpty(email, r.FormValue("email"))
	if !a.allowAuthRequest(w, r, magicLinkSourcePolicy, magicLinkAccountPolicy, accountKey) {
		return
	}
	if !a.emailLoginAvailable() {
		http.Redirect(w, r, "/?sent=1", http.StatusSeeOther)
		return
	}
	if a.demoLogin {
		code := strings.TrimSpace(r.FormValue("access_code"))
		if code == "" || subtle.ConstantTimeCompare([]byte(code), []byte(a.demoLoginCode)) != 1 {
			logWarn("demo login: wrong access code", "tenant", tenant.Slug)
			a.renderDemoCodeWrong(w, tenant)
			return
		}
	}
	if _, err := mail.ParseAddress(email); err != nil {
		http.Redirect(w, r, "/?sent=1", http.StatusSeeOther)
		return
	}
	if a.serviceProviderAccessClosedFor(tenant.Slug, email) {
		http.Redirect(w, r, "/?sent=1", http.StatusSeeOther)
		return
	}
	if !a.isAllowed(email, tenant.Slug) || !a.isAuthMethodAllowed(email, tenant.Slug, authMethodEmail) {
		http.Redirect(w, r, "/?sent=1", http.StatusSeeOther)
		return
	}

	token, err := randomToken(32)
	if err != nil {
		logWarn("magic link creation failed", "error_type", "random_token")
		http.Redirect(w, r, "/?sent=1", http.StatusSeeOther)
		return
	}
	a.tokens.Put(token, email, tenant.Slug, 15*time.Minute)

	link := a.publicBaseURL(r, tenant) + "/auth/verify?token=" + url.QueryEscape(token)
	devLink := (a.localDevLogin || a.demoLogin) && !a.mailer.Configured()
	if devLink {
		copy := a.publicHomeCopy(tenant.Slug)
		a.render(w, "home", map[string]any{
			"Title":               tenant.Address + " · Hausportal",
			"Tenant":              tenant,
			"HouseName":           houseDisplayName(tenant),
			"Email":               email,
			"Sent":                true,
			"Expired":             false,
			"MailConfigured":      false,
			"DevLoginLink":        link,
			"DemoLoginEnabled":    a.demoLogin,
			"DemoCodeWrong":       false,
			"Denied":              false,
			"OIDCConfigured":      a.oidc.Configured(),
			"EmailLoginAvailable": a.emailLoginAvailable(),
			"MapURL":              tenantMapURL(tenant.Address),
			"LocationMap":         publicMapForTenant(tenant),
			"HomeCopy":            copy,
		})
		return
	}

	if !a.enqueueMagicLinkDelivery(magicLinkDeliveryJob{
		mailer:  a.mailer,
		to:      email,
		link:    link,
		address: tenant.Address,
		invalidate: func() {
			a.tokens.Invalidate(token)
		},
	}) {
		// Do not retain a valid token that can never reach its intended
		// recipient. Queue pressure stays invisible on the public response.
		a.tokens.Invalidate(token)
		logWarn("magic link delivery not queued")
	}

	http.Redirect(w, r, "/?sent=1", http.StatusSeeOther)
}

func (a *app) renderDemoCodeWrong(w http.ResponseWriter, tenant tenantConfig) {
	copy := a.publicHomeCopy(tenant.Slug)
	a.render(w, "home", map[string]any{
		"Title":               tenant.Address + " · Hausportal",
		"Tenant":              tenant,
		"HouseName":           houseDisplayName(tenant),
		"Email":               "",
		"Sent":                true,
		"Expired":             false,
		"MailConfigured":      false,
		"DevLoginLink":        "",
		"DemoLoginEnabled":    true,
		"DemoCodeWrong":       true,
		"Denied":              false,
		"OIDCConfigured":      a.oidc.Configured(),
		"EmailLoginAvailable": a.emailLoginAvailable(),
		"MapURL":              tenantMapURL(tenant.Address),
		"LocationMap":         publicMapForTenant(tenant),
		"HomeCopy":            copy,
	})
}

func (a *app) publicHomeCopy(tenantSlug string) publicHomeCopy {
	tenant, ok := a.tenantBySlug(tenantSlug)
	if !ok {
		return neutralPublicHomeCopy()
	}
	portalType, classified := classifiedPortalType(tenant)
	if !classified {
		return neutralPublicHomeCopy()
	}
	switch portalType {
	case config.PortalTypeApartment:
		return publicHomeCopy{
			Eyebrow:       "Ihr Zuhause-Portal",
			Headline:      "Alles Wichtige für Ihre Wohnung.",
			Lead:          "Aushänge, Termine, Dokumente, Anliegen und Energie – privat an einem Ort.",
			FirstDetail:   "Neuigkeiten und Unterlagen im Blick.",
			SecondDetail:  "Aufgaben und Energie verständlich gebündelt.",
			PrivacyDetail: "Nur für eingeladene Personen.",
		}
	case config.PortalTypeHouse:
		return publicHomeCopy{
			Eyebrow:       "Ihr Zuhause-Portal",
			Headline:      "Alles Wichtige für Ihr Zuhause.",
			Lead:          "Termine, Dokumente, Aufgaben und Energie – privat an einem Ort.",
			FirstDetail:   "Unterlagen und Wartung im Blick.",
			SecondDetail:  "Energie verstehen und Schritt für Schritt planen.",
			PrivacyDetail: "Nur für eingeladene Personen.",
		}
	case config.PortalTypeCommunity:
		return publicHomeCopy{
			Eyebrow:       "Ihr Hausportal",
			Headline:      "Alles Wichtige rund um unser Haus.",
			Lead:          "Aushänge, Termine, Dokumente und Anliegen – privat an einem Ort.",
			FirstDetail:   "Wichtige Aushänge und Neuigkeiten.",
			SecondDetail:  "Termine und Aufgaben im Blick.",
			PrivacyDetail: "Nur für unsere Hausgemeinschaft.",
		}
	default:
		return neutralPublicHomeCopy()
	}
}

func classifiedPortalType(tenant tenantConfig) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(tenant.PortalType)) {
	case config.PortalTypeApartment:
		return config.PortalTypeApartment, true
	case config.PortalTypeHouse:
		return config.PortalTypeHouse, true
	case config.PortalTypeCommunity:
		return config.PortalTypeCommunity, true
	case "":
		// Existing tenant declarations predate portal_type and all describe WEG
		// portals. Private B2C portals must opt in explicitly.
		return config.PortalTypeCommunity, true
	default:
		return "", false
	}
}

func neutralPublicHomeCopy() publicHomeCopy {
	return publicHomeCopy{
		Eyebrow:       "Ihr privates Portal",
		Headline:      "Alles Wichtige an einem Ort.",
		Lead:          "Aushänge, Termine, Dokumente und Anliegen – nur für eingeladene Personen.",
		FirstDetail:   "Neuigkeiten und Unterlagen im Blick.",
		SecondDetail:  "Termine und Aufgaben verständlich gebündelt.",
		PrivacyDetail: "Nur für eingeladene Personen.",
	}
}

func (a *app) verifyLogin(w http.ResponseWriter, r *http.Request) {
	if !a.allowAuthRequest(w, r, loginCompletionSourcePolicy, authRatePolicy{}, "") {
		return
	}
	token := r.URL.Query().Get("token")
	if email, tenantSlug, _, ok := a.tokens.Peek(token); ok {
		if !a.allowAuthRequest(
			w,
			r,
			authRatePolicy{},
			loginCompletionAccountPolicy,
			tenantSlug+"|"+email,
		) {
			return
		}
	}
	email, tenantSlug, redirectPath, ok := a.tokens.Consume(token)
	if !ok {
		http.Redirect(w, r, "/?login=expired#login", http.StatusSeeOther)
		return
	}

	if err := a.startSession(w, email, tenantSlug, authMethodEmail); errors.Is(err, errServiceProviderAccessClosed) {
		http.Error(w, serviceProviderAccessClosedMessage, http.StatusForbidden)
		return
	} else if err != nil {
		http.Error(w, "Could not create session", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, firstNonEmpty(redirectPath, "/app"), http.StatusSeeOther)
}

func tenantMapURL(address string) string {
	query := strings.TrimSpace(address)
	if query == "" {
		query = "Wien, Österreich"
	}
	return "https://www.openstreetmap.org/search?query=" + url.QueryEscape(query)
}

func houseDisplayName(tenant tenantConfig) string {
	name := strings.TrimSpace(tenant.Name)
	lower := strings.ToLower(name)
	if name == "" || lower == "weg portal" || strings.Contains(lower, "-portal") {
		return firstNonEmpty(tenant.Address, "Hausportal")
	}
	return name
}

type sidebarAddressView struct {
	Full        string
	Primary     string
	Locality    string
	HasLocality bool
}

// sidebarAddressForTenant keeps the mobile house identity recognisable instead
// of clipping an arbitrary part of the postal address. The full address stays
// available to assistive technology and on wider screens.
func sidebarAddressForTenant(tenant tenantConfig) sidebarAddressView {
	full := strings.TrimSpace(tenant.Address)
	primary := ""
	if full != "" && !strings.HasPrefix(strings.ToLower(full), "pilot ") {
		primary = strings.TrimSpace(strings.Split(full, ",")[0])
	}
	if primary == "" {
		primary = strings.TrimSpace(tenant.Name)
		lower := strings.ToLower(primary)
		switch {
		case strings.HasSuffix(lower, "-portal"):
			primary = strings.TrimSpace(primary[:len(primary)-len("-portal")])
		case strings.HasSuffix(lower, " portal"):
			primary = strings.TrimSpace(primary[:len(primary)-len(" portal")])
		}
	}
	primary = firstNonEmpty(primary, "Hausportal")

	parts := strings.Split(full, ",")
	locality := ""
	if len(parts) > 1 {
		candidate := strings.TrimSpace(parts[len(parts)-1])
		if strings.EqualFold(candidate, "Österreich") && len(parts) > 2 {
			candidate = strings.TrimSpace(parts[len(parts)-2])
		}
		fields := strings.Fields(candidate)
		if len(fields) > 1 && allASCIIDigits(fields[0]) {
			candidate = strings.Join(fields[1:], " ")
		}
		if !strings.EqualFold(candidate, primary) {
			locality = candidate
		}
	}

	return sidebarAddressView{
		Full:        firstNonEmpty(full, primary),
		Primary:     primary,
		Locality:    locality,
		HasLocality: locality != "",
	}
}

func allASCIIDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func (a *app) startOIDCLogin(w http.ResponseWriter, r *http.Request) {
	if !a.oidc.Configured() {
		http.NotFound(w, r)
		return
	}
	if !a.allowAuthRequest(w, r, oidcStartSourcePolicy, authRatePolicy{}, "") {
		return
	}
	tenant := a.tenantForRequest(r)
	if err := a.oidc.EnsureProvider(r.Context()); err != nil {
		logError("OIDC discovery failed during login start", err)
		http.Error(w, "Die Anmeldung ist gerade nicht erreichbar. Bitte später erneut versuchen oder den E-Mail-Link verwenden.", http.StatusServiceUnavailable)
		return
	}
	state, err := randomToken(32)
	if err != nil {
		http.Error(w, "Anmeldung konnte nicht gestartet werden.", http.StatusInternalServerError)
		return
	}
	nonce, err := randomToken(32)
	if err != nil {
		http.Error(w, "Anmeldung konnte nicht gestartet werden.", http.StatusInternalServerError)
		return
	}
	codeVerifier, err := randomToken(32)
	if err != nil {
		http.Error(w, "Anmeldung konnte nicht gestartet werden.", http.StatusInternalServerError)
		return
	}
	a.oidcFlows.Put(state, auth.NewOIDCFlow(tenant.Slug, nonce, codeVerifier), 10*time.Minute)

	redirectURL := a.oidcRedirectURL()
	oauthConfig := a.oidc.OAuthConfig(redirectURL)
	authCodeURL := oauthConfig.AuthCodeURL(
		state,
		oidc.Nonce(nonce),
		oauth2.SetAuthURLParam("code_challenge", pkceChallenge(codeVerifier)),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
	http.Redirect(w, r, authCodeURL, http.StatusSeeOther)
}

func (a *app) finishOIDCLogin(w http.ResponseWriter, r *http.Request) {
	if !a.oidc.Configured() {
		http.NotFound(w, r)
		return
	}
	if !a.allowAuthRequest(w, r, loginCompletionSourcePolicy, authRatePolicy{}, "") {
		return
	}
	if err := a.oidc.EnsureProvider(r.Context()); err != nil {
		logError("OIDC discovery failed during login callback", err)
		http.Error(w, "Die Anmeldung ist gerade nicht erreichbar. Bitte später erneut versuchen.", http.StatusServiceUnavailable)
		return
	}
	if errText := strings.TrimSpace(r.URL.Query().Get("error")); errText != "" {
		logWarn("OIDC login rejected", "provider_error", errText)
		http.Redirect(w, r, "/?denied=1", http.StatusSeeOther)
		return
	}
	flow, ok := a.oidcFlows.Consume(r.URL.Query().Get("state"))
	if !ok {
		http.Redirect(w, r, "/?login=expired#login", http.StatusSeeOther)
		return
	}
	tenant, ok := a.tenantBySlug(flow.TenantSlug())
	if !ok {
		http.Error(w, "Unknown tenant", http.StatusUnauthorized)
		return
	}
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		http.Error(w, "Die Anmeldung konnte nicht abgeschlossen werden.", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	oauthConfig := a.oidc.OAuthConfig(a.oidcRedirectURL())
	token, err := oauthConfig.Exchange(
		ctx,
		code,
		oauth2.SetAuthURLParam("code_verifier", flow.CodeVerifier()),
	)
	if err != nil {
		logError("OIDC token exchange failed", err)
		http.Error(w, "Die Anmeldung konnte nicht abgeschlossen werden.", http.StatusUnauthorized)
		return
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		logWarn("OIDC token exchange returned no ID token")
		http.Error(w, "Die Anmeldung konnte nicht geprüft werden.", http.StatusUnauthorized)
		return
	}
	idToken, err := a.oidc.Verifier().Verify(ctx, rawIDToken)
	if err != nil {
		logError("OIDC ID token verification failed", err)
		http.Error(w, "Die Anmeldung konnte nicht geprüft werden.", http.StatusUnauthorized)
		return
	}
	if idToken.Nonce != flow.Nonce() {
		logWarn("OIDC nonce mismatch")
		http.Error(w, "Die Anmeldung konnte nicht geprüft werden.", http.StatusUnauthorized)
		return
	}

	claims := oidcUserClaims{}
	if err := idToken.Claims(&claims); err != nil {
		logError("OIDC claims decode failed", err)
		http.Error(w, "Die Anmeldung konnte nicht gelesen werden.", http.StatusUnauthorized)
		return
	}
	if claims.Email == "" || claims.EmailVerified == nil {
		userInfo, err := a.oidc.Provider().UserInfo(ctx, oauth2.StaticTokenSource(token))
		if err != nil {
			logError("OIDC userinfo failed", err)
		} else {
			var extra oidcUserClaims
			if err := userInfo.Claims(&extra); err == nil {
				claims.Merge(extra)
			}
		}
	}
	email := normalizeEmail(claims.Email)
	if email == "" || claims.EmailVerified == nil || !*claims.EmailVerified {
		http.Redirect(w, r, tenant.PublicURL("/?denied=1"), http.StatusSeeOther)
		return
	}
	if !a.allowAuthRequest(
		w,
		r,
		authRatePolicy{},
		loginCompletionAccountPolicy,
		tenant.Slug+"|"+email,
	) {
		return
	}
	if a.serviceProviderAccessClosedFor(tenant.Slug, email) {
		http.Error(w, serviceProviderAccessClosedMessage, http.StatusForbidden)
		return
	}
	if !a.isAllowed(email, tenant.Slug) || !a.isAuthMethodAllowed(email, tenant.Slug, authMethodOIDC) {
		http.Redirect(w, r, tenant.PublicURL("/?denied=1"), http.StatusSeeOther)
		return
	}
	if err := a.startSession(w, email, tenant.Slug, authMethodOIDC); err != nil {
		http.Error(w, "Could not create session", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, tenant.PublicURL("/app"), http.StatusSeeOther)
}

// oidcRedirectURL deliberately uses one platform callback for every tenant.
// The signed, single-use OIDC flow carries the tenant slug and routes the user
// back to the correct portal after authentication.
func (a *app) oidcRedirectURL() string {
	return a.oidc.RedirectURL(a.baseURL)
}

func (a *app) startSession(w http.ResponseWriter, email string, tenantSlug string, authMethod string) error {
	if a.serviceProviderAccessClosedFor(tenantSlug, email) {
		return errServiceProviderAccessClosed
	}
	token, expiresAt, err := a.sessions.Put(email, tenantSlug, authMethod, a.sessionTTL)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "weg_session",
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   a.sessionSecure,
		SameSite: http.SameSiteLaxMode,
	})
	if a.activityStore != nil {
		if err := a.activityStore.Touch(email, time.Now(), authMethod); err != nil {
			logError("activity record failed", err, "actor", redactedEmail(email))
		}
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenantSlug,
		ActorEmail: email,
		ActorRole:  a.roleFor(email, tenantSlug),
		Action:     auditActionLogin,
		TargetType: "session",
		TargetID:   email,
		Summary:    "Anmeldung erfolgreich",
		Details: map[string]string{
			"auth_method": strings.Join(authMethodsLabelList([]string{authMethod}), ", "),
		},
	})
	return nil
}

func (a *app) logout(w http.ResponseWriter, r *http.Request) {
	if !sameOriginPost(r) {
		http.Error(w, "Bad request", http.StatusForbidden)
		return
	}
	if c, err := r.Cookie("weg_session"); err == nil {
		if session, ok := a.sessions.GetSession(c.Value); ok && session.PreviewRole != "" {
			a.recordRolePreviewEnd(session, "logout")
		}
		a.sessions.Delete(c.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "weg_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   a.sessionSecure,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
