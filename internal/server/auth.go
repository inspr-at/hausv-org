package server

import (
	"context"
	"errors"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"

	oidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/markus-barta/hausv-org/internal/auth"
	"golang.org/x/oauth2"
)

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
	unitWeight := 0
	if a.unitStore != nil {
		unitWeight = a.unitStore.BillableUnitWeight(tenant.Slug)
	}
	titleName := firstNonEmpty(tenant.Name, "WEG Portal")
	a.render(w, "home", map[string]any{
		"Title":               titleName + " " + tenant.Address,
		"Tenant":              tenant,
		"Email":               email,
		"UnitCount":           formatBillableUnitWeight(unitWeight),
		"UnitCountLabel":      billableUnitCountLabel(unitWeight),
		"HasUnitCount":        unitWeight > 0,
		"Sent":                r.URL.Query().Get("sent") == "1",
		"MailConfigured":      a.mailer.Configured(),
		"DevLoginLink":        "",
		"Denied":              r.URL.Query().Get("denied") == "1",
		"OIDCConfigured":      a.oidc.Configured(),
		"OIDCProviderName":    a.oidc.ProviderName(),
		"EmailLoginAvailable": a.emailLoginAvailable(),
	})
}

func (a *app) marketingLanding(w http.ResponseWriter, r *http.Request) {
	a.render(w, "landing", map[string]any{
		"Title":          "hausv.org - kostenlose Hausverwaltung",
		"ContactLocal":   "hello",
		"ContactDomain":  "hausv.org",
		"ContactDisplay": "hello [at] hausv [dot] org",
		"PrimaryAppURL":  "https://jhw22.hausv.org/",
		"RequestedHost":  normalizeHost(r.Host),
		"LandingHeroURL": "/assets/hausv-landing-hero.png",
	})
}

func (a *app) privacyNotice(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	contactEmail := firstNonEmpty(tenant.ContactEmail, "hello@hausv.org")
	a.render(w, "privacy", map[string]any{
		"Title":                         "Datenschutz · hausv.org",
		"Tenant":                        tenant,
		"HouseContactName":              firstNonEmpty(tenant.ContactName, tenant.Name, "Hausverwaltung"),
		"HouseContactEmail":             contactEmail,
		"HouseContactPhone":             tenant.ContactPhone,
		"TechnicalContactEmail":         "hello@hausv.org",
		"ServiceProviderEnabled":        a.serviceAccessEnabled,
		"ServiceProviderAssessment":     serviceProviderAssessmentVersion,
		"ServiceProviderRetentionYears": 3,
	})
}

func (a *app) requestLogin(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	if !sameOriginPost(r) {
		http.Error(w, "Bad request", http.StatusForbidden)
		return
	}
	if !a.emailLoginAvailable() {
		http.Redirect(w, r, "/?denied=1", http.StatusSeeOther)
		return
	}
	if err := parseMaybeMultipartForm(w, r, maxIssueAttachmentFormBytes, maxAttachmentBytes); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	email := normalizeEmail(r.FormValue("email"))
	if _, err := mail.ParseAddress(email); err != nil {
		http.Redirect(w, r, "/?denied=1", http.StatusSeeOther)
		return
	}
	if a.serviceProviderAccessClosedFor(tenant.Slug, email) {
		http.Error(w, serviceProviderAccessClosedMessage, http.StatusForbidden)
		return
	}
	if !a.isAllowed(email, tenant.Slug) || !a.isAuthMethodAllowed(email, tenant.Slug, authMethodEmail) {
		http.Redirect(w, r, "/?denied=1", http.StatusSeeOther)
		return
	}

	token, err := randomToken(32)
	if err != nil {
		http.Error(w, "Could not create login link", http.StatusInternalServerError)
		return
	}
	a.tokens.Put(token, email, tenant.Slug, 15*time.Minute)

	link := a.publicBaseURL(r, tenant) + "/auth/verify?token=" + url.QueryEscape(token)
	if a.localDevLogin && !a.mailer.Configured() {
		a.render(w, "home", map[string]any{
			"Title":               "WEG Portal " + tenant.Address,
			"Tenant":              tenant,
			"Email":               email,
			"Sent":                true,
			"MailConfigured":      false,
			"DevLoginLink":        link,
			"Denied":              false,
			"OIDCConfigured":      a.oidc.Configured(),
			"OIDCProviderName":    a.oidc.ProviderName(),
			"EmailLoginAvailable": a.emailLoginAvailable(),
		})
		return
	}

	if err := a.mailer.SendMagicLink(email, link); err != nil {
		logError("magic link delivery failed", err, "recipient", redactedEmail(email))
		http.Redirect(w, r, "/?sent=1", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/?sent=1", http.StatusSeeOther)
}

func (a *app) verifyLogin(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	email, tenantSlug, redirectPath, ok := a.tokens.Consume(token)
	if !ok {
		http.Error(w, "Dieser Anmeldelink ist abgelaufen oder wurde bereits verwendet.", http.StatusUnauthorized)
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

func (a *app) startOIDCLogin(w http.ResponseWriter, r *http.Request) {
	if !a.oidc.Configured() {
		http.NotFound(w, r)
		return
	}
	tenant := a.tenantForRequest(r)
	if err := a.oidc.EnsureProvider(r.Context()); err != nil {
		logError("OIDC discovery failed during login start", err)
		http.Error(w, "SSO ist gerade nicht erreichbar. Bitte später erneut versuchen oder den E-Mail-Link verwenden.", http.StatusServiceUnavailable)
		return
	}
	state, err := randomToken(32)
	if err != nil {
		http.Error(w, "Could not start SSO login", http.StatusInternalServerError)
		return
	}
	nonce, err := randomToken(32)
	if err != nil {
		http.Error(w, "Could not start SSO login", http.StatusInternalServerError)
		return
	}
	codeVerifier, err := randomToken(32)
	if err != nil {
		http.Error(w, "Could not start SSO login", http.StatusInternalServerError)
		return
	}
	a.oidcFlows.Put(state, auth.NewOIDCFlow(tenant.Slug, nonce, codeVerifier), 10*time.Minute)

	redirectURL := a.oidc.RedirectURL(a.publicBaseURL(r, tenant))
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
	if err := a.oidc.EnsureProvider(r.Context()); err != nil {
		logError("OIDC discovery failed during login callback", err)
		http.Error(w, "SSO ist gerade nicht erreichbar. Bitte später erneut versuchen.", http.StatusServiceUnavailable)
		return
	}
	if errText := strings.TrimSpace(r.URL.Query().Get("error")); errText != "" {
		logWarn("OIDC login rejected", "provider_error", errText)
		http.Redirect(w, r, "/?denied=1", http.StatusSeeOther)
		return
	}
	flow, ok := a.oidcFlows.Consume(r.URL.Query().Get("state"))
	if !ok {
		http.Error(w, "Diese SSO-Anmeldung ist abgelaufen. Bitte erneut anmelden.", http.StatusUnauthorized)
		return
	}
	tenant, ok := a.tenantBySlug(flow.TenantSlug())
	if !ok {
		http.Error(w, "Unknown tenant", http.StatusUnauthorized)
		return
	}
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		http.Error(w, "SSO-Anmeldung ohne Code.", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	oauthConfig := a.oidc.OAuthConfig(a.oidc.RedirectURL(a.publicBaseURL(r, tenant)))
	token, err := oauthConfig.Exchange(
		ctx,
		code,
		oauth2.SetAuthURLParam("code_verifier", flow.CodeVerifier()),
	)
	if err != nil {
		logError("OIDC token exchange failed", err)
		http.Error(w, "SSO-Anmeldung konnte nicht abgeschlossen werden.", http.StatusUnauthorized)
		return
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		logWarn("OIDC token exchange returned no ID token")
		http.Error(w, "SSO-Anmeldung konnte nicht geprüft werden.", http.StatusUnauthorized)
		return
	}
	idToken, err := a.oidc.Verifier().Verify(ctx, rawIDToken)
	if err != nil {
		logError("OIDC ID token verification failed", err)
		http.Error(w, "SSO-Anmeldung konnte nicht geprüft werden.", http.StatusUnauthorized)
		return
	}
	if idToken.Nonce != flow.Nonce() {
		logWarn("OIDC nonce mismatch")
		http.Error(w, "SSO-Anmeldung konnte nicht geprüft werden.", http.StatusUnauthorized)
		return
	}

	claims := oidcUserClaims{}
	if err := idToken.Claims(&claims); err != nil {
		logError("OIDC claims decode failed", err)
		http.Error(w, "SSO-Anmeldung konnte nicht gelesen werden.", http.StatusUnauthorized)
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
		http.Redirect(w, r, "/?denied=1", http.StatusSeeOther)
		return
	}
	if a.serviceProviderAccessClosedFor(tenant.Slug, email) {
		http.Error(w, serviceProviderAccessClosedMessage, http.StatusForbidden)
		return
	}
	if !a.isAllowed(email, tenant.Slug) || !a.isAuthMethodAllowed(email, tenant.Slug, authMethodOIDC) {
		http.Redirect(w, r, "/?denied=1", http.StatusSeeOther)
		return
	}
	if err := a.startSession(w, email, tenant.Slug, authMethodOIDC); err != nil {
		http.Error(w, "Could not create session", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/app", http.StatusSeeOther)
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
