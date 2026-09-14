package server

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/auth"
	"github.com/inspr-at/hausv-org/internal/web"
)

const supportViewTTL = 15 * time.Minute

const (
	supportViewEndExpired                = "expired"
	supportViewEndActorContextInvalid    = "actor_context_invalid"
	supportViewEndActorPermissionRevoked = "actor_permission_revoked"
	supportViewEndTargetContextInvalid   = "target_context_invalid"
	supportViewEndTargetRoleChanged      = "target_role_changed"
	supportViewEndTargetDeactivated      = "target_deactivated"
)

type supportViewContext struct {
	ActorEmail  string
	ActorRole   string
	TargetEmail string
	TargetName  string
	TargetRole  string
	StartedAt   time.Time
	ExpiresAt   time.Time
}

func supportViewPortalData(ac *authCtx) *web.SupportViewData {
	if ac == nil || ac.supportView == nil {
		return nil
	}
	minutes := int(time.Until(ac.supportView.ExpiresAt).Minutes()) + 1
	if minutes < 1 {
		minutes = 1
	}
	return &web.SupportViewData{TargetName: ac.supportView.TargetName, TargetRole: ac.supportView.TargetRole, HouseName: houseDisplayName(ac.tenant), EndsInMinutes: minutes, EndPath: ac.tenant.PublicURL("/app/support-view/end")}
}

func supportPermissionsForActor(ac authCtx, permissions []string, previous bool) []string {
	if normalizeRole(ac.realRole) != roleAdmin {
		return setPermission(permissions, permissionSupportView, previous)
	}
	return permissions
}

func (a *app) canStartSupportView(ac *authCtx) bool {
	return ac != nil && ac.preview == nil && ac.supportView == nil && normalizeRole(ac.realRole) == roleAdmin && a.profileForTenant(ac.email, ac.tenant.Slug).HasPermission(permissionSupportView)
}

func (a *app) supportViewPage(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.canStartSupportView(&ac) {
		http.Error(w, "Supportansicht nicht freigegeben.", http.StatusForbidden)
		return
	}
	targets := []web.SupportViewTarget{}
	for _, row := range a.userRows(ac.tenantRef) {
		if normalizeEmail(row.Email) == ac.email || !a.isAllowed(row.Email, ac.tenant.Slug) || a.serviceProviderAccessClosedFor(ac.tenant.Slug, row.Email) {
			continue
		}
		profile, ok := a.directoryProfile(row.Email)
		if !ok || !profile.HasTenant(ac.tenant.Slug) {
			continue
		}
		profile = profile.ForTenant(ac.tenant.Slug)
		if profile.Deactivated {
			continue
		}
		targets = append(targets, web.SupportViewTarget{Email: row.Email, Name: profile.DisplayName(), Role: profile.Role})
	}
	a.renderSettingsComponent(w, r, ac.tenant.Slug, web.SupportViewPage(a.portalBaseData(ac, "settings", "Supportansicht"), targets))
}

// recordAuthenticatedReadAudit keeps authorization on the effective support
// target while attributing the resulting read access to the real signed-in
// principal. The target context is deliberately limited to identity and role;
// no document or attachment content is added here.
func (a *app) recordAuthenticatedReadAudit(ac authCtx, event auditEvent) {
	event.ActorEmail = ac.realEmail
	event.ActorRole = ac.realRole
	if event.ActorEmail == "" {
		event.ActorEmail = ac.email
	}
	if event.ActorRole == "" {
		event.ActorRole = ac.role
	}
	if ac.supportView != nil {
		details := make(map[string]string, len(event.Details)+2)
		for key, value := range event.Details {
			details[key] = value
		}
		details["support_target_email"] = ac.supportView.TargetEmail
		details["support_target_role"] = ac.supportView.TargetRole
		event.Details = details
	}
	a.recordAudit(event)
}

func (a *app) sessionForRequest(r *http.Request) (auth.Session, bool) {
	if a == nil || a.sessions == nil || r == nil {
		return auth.Session{}, false
	}
	cookie, err := r.Cookie("weg_session")
	if err != nil {
		return auth.Session{}, false
	}
	return a.sessions.GetSession(cookie.Value)
}

func (a *app) supportSessionAllowed(session auth.Session) bool {
	return a.supportSessionEndReason(session) == ""
}

// supportSessionEndReason classifies every live invalidation without putting
// profile values into the reason. Authentication calls it before currentUser
// so a support token can never linger after the actor or target changes.
func (a *app) supportSessionEndReason(session auth.Session) string {
	if session.SupportTargetEmail == "" || session.SupportTargetRole == "" {
		return supportViewEndTargetContextInvalid
	}
	if session.SupportExpiresAt <= time.Now().Unix() {
		return supportViewEndExpired
	}
	if !a.ownPortalContextAllowed(session.Email, session.TenantSlug, session.Role, session.AuthMethod) {
		return supportViewEndActorContextInvalid
	}
	actor := a.profileForTenant(session.Email, session.TenantSlug)
	if normalizeRole(session.Role) != roleAdmin || !actor.HasPermission(permissionSupportView) {
		return supportViewEndActorPermissionRevoked
	}
	if session.SupportTargetEmail == session.Email {
		return supportViewEndTargetContextInvalid
	}
	target, ok := a.directoryProfile(session.SupportTargetEmail)
	if !ok || !target.HasTenant(session.TenantSlug) {
		return supportViewEndTargetContextInvalid
	}
	target = target.ForTenant(session.TenantSlug)
	if target.Deactivated {
		return supportViewEndTargetDeactivated
	}
	if !a.isAllowed(session.SupportTargetEmail, session.TenantSlug) || a.serviceProviderAccessClosedFor(session.TenantSlug, session.SupportTargetEmail) {
		return supportViewEndTargetContextInvalid
	}
	if normalizeRole(target.Role) != normalizeRole(session.SupportTargetRole) {
		return supportViewEndTargetRoleChanged
	}
	return ""
}

func setSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     "weg_session",
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (a *app) startSupportView(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.canStartSupportView(&ac) {
		http.Error(w, "Supportansicht nicht freigegeben.", http.StatusForbidden)
		return
	}
	actor := a.profileForTenant(ac.realEmail, ac.tenant.Slug)
	if !actor.HasPermission(permissionSupportView) {
		http.Error(w, "Supportansicht nicht freigegeben.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	if normalizeSlug(r.FormValue("tenant")) != ac.tenant.Slug {
		http.Error(w, "Ungültige Liegenschaft.", http.StatusForbidden)
		return
	}
	targetEmail := normalizeEmail(r.FormValue("target_email"))
	targetRole := normalizeRole(r.FormValue("target_role"))
	if targetEmail == "" || targetEmail == ac.realEmail || a.serviceProviderAccessClosedFor(ac.tenant.Slug, targetEmail) || !a.isAllowed(targetEmail, ac.tenant.Slug) {
		http.Error(w, "Ungültiges Supportziel.", http.StatusForbidden)
		return
	}
	target, ok := a.directoryProfile(targetEmail)
	if !ok || !target.HasTenant(ac.tenant.Slug) {
		http.Error(w, "Ungültiges Supportziel.", http.StatusForbidden)
		return
	}
	target = target.ForTenant(ac.tenant.Slug)
	if target.Deactivated || normalizeRole(target.Role) != targetRole {
		http.Error(w, "Ungültige Rolle für dieses Supportziel.", http.StatusForbidden)
		return
	}
	cookie, err := r.Cookie("weg_session")
	if err != nil {
		http.Error(w, "Anmeldung erforderlich", http.StatusUnauthorized)
		return
	}
	current, ok := a.sessions.GetSession(cookie.Value)
	if !ok || current.PreviewRole != "" || current.SupportTargetEmail != "" || current.Email != ac.realEmail || current.TenantSlug != ac.tenant.Slug {
		http.Error(w, "Anmeldung erforderlich", http.StatusUnauthorized)
		return
	}
	now := time.Now().UTC().Truncate(time.Second)
	parentExpiresAt := time.Unix(current.ExpiresAt, 0)
	actorRole := normalizeRole(current.Role)
	if actorRole == "" {
		actorRole = normalizeRole(ac.realRole)
	}
	supportExpiresAt := now.Add(supportViewTTL)
	if parentExpiresAt.Before(supportExpiresAt) {
		supportExpiresAt = parentExpiresAt
	}
	token, cookieExpiresAt, err := a.sessions.PutSupportView(current.Email, current.TenantSlug, current.AuthMethod, actorRole, targetEmail, targetRole, now, supportExpiresAt, parentExpiresAt)
	if err != nil {
		http.Error(w, "Supportansicht konnte nicht gestartet werden.", http.StatusInternalServerError)
		return
	}
	if a.auditStore == nil {
		http.Error(w, "Supportansicht benötigt ein verfügbares Protokoll.", http.StatusServiceUnavailable)
		return
	}
	if err := a.auditStore.Append(auditEvent{
		TenantSlug: ac.tenant.Slug,
		ActorEmail: current.Email,
		ActorRole:  actorRole,
		Action:     auditActionSupportViewStart,
		TargetType: "user_role",
		TargetID:   targetEmail,
		Summary:    "Supportansicht gestartet",
		Details: map[string]string{
			"target_role": targetRole,
			"started_at":  now.Format(time.RFC3339),
			"expires_at":  supportExpiresAt.UTC().Format(time.RFC3339),
		},
	}); err != nil {
		logError("support start audit failed", err, "tenant", ac.tenant.Slug)
		http.Error(w, "Supportansicht konnte nicht protokolliert werden.", http.StatusServiceUnavailable)
		return
	}
	setSessionCookie(w, token, cookieExpiresAt, a.sessionSecure)
	a.sessions.Delete(cookie.Value)

	http.Redirect(w, r, strings.TrimRight(a.baseURL, "/")+ac.tenant.PublicURL("/app"), http.StatusSeeOther)
}

func (a *app) endSupportView(w http.ResponseWriter, r *http.Request) {
	if !sameOriginPost(r) {
		http.Error(w, "Bad request", http.StatusForbidden)
		return
	}
	ac, ok := a.authenticate(w, r)
	if !ok {
		return
	}
	if ac.supportView == nil {
		http.Error(w, "Keine aktive Supportansicht.", http.StatusForbidden)
		return
	}
	cookie, err := r.Cookie("weg_session")
	if err != nil {
		http.Error(w, "Anmeldung erforderlich", http.StatusUnauthorized)
		return
	}
	current, ok := a.sessions.GetSession(cookie.Value)
	if !ok || current.SupportTargetEmail == "" {
		http.Error(w, "Anmeldung erforderlich", http.StatusUnauthorized)
		return
	}
	parentExpiresAt := time.Unix(current.ExpiresAt, 0)
	token, expiresAt, err := a.sessions.PutSession(current.Email, current.TenantSlug, current.AuthMethod, current.Role, parentExpiresAt)
	if err != nil {
		http.Error(w, "Supportansicht konnte nicht beendet werden.", http.StatusInternalServerError)
		return
	}
	setSessionCookie(w, token, expiresAt, a.sessionSecure)
	a.sessions.Delete(cookie.Value)
	a.recordSupportViewEnd(current, "explicit")
	http.Redirect(w, r, strings.TrimRight(a.baseURL, "/")+ac.tenant.PublicURL("/app/settings/users"), http.StatusSeeOther)
}

func (a *app) terminateSupportView(w http.ResponseWriter, r *http.Request, session auth.Session, reason string) {
	if cookie, cookieErr := r.Cookie("weg_session"); cookieErr == nil {
		a.sessions.Delete(cookie.Value)
	}

	restored := false
	parentExpiresAt := time.Unix(session.ExpiresAt, 0)
	if parentExpiresAt.After(time.Now()) && a.ownPortalContextAllowed(session.Email, session.TenantSlug, session.Role, session.AuthMethod) {
		if token, expiresAt, err := a.sessions.PutSession(session.Email, session.TenantSlug, session.AuthMethod, session.Role, parentExpiresAt); err == nil {
			setSessionCookie(w, token, expiresAt, a.sessionSecure)
			restored = true
		}
	}
	if !restored {
		http.SetCookie(w, &http.Cookie{Name: "weg_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: a.sessionSecure, SameSite: http.SameSiteLaxMode})
	}
	a.recordSupportViewEnd(session, reason)
	target := "/"
	if restored {
		if tenant, ok := a.tenantBySlug(session.TenantSlug); ok {
			target = tenant.PublicURL("/app")
		}
	}
	http.Redirect(w, r, strings.TrimRight(a.baseURL, "/")+target, http.StatusSeeOther)
}

func (a *app) recordSupportViewEnd(session auth.Session, reason string) {
	a.recordAudit(auditEvent{
		TenantSlug: session.TenantSlug,
		ActorEmail: session.Email,
		ActorRole:  session.Role,
		Action:     auditActionSupportViewEnd,
		TargetType: "user_role",
		TargetID:   session.SupportTargetEmail,
		Summary:    "Supportansicht beendet",
		Details: map[string]string{
			"target_role": session.SupportTargetRole,
			"started_at":  time.Unix(session.SupportStartedAt, 0).UTC().Format(time.RFC3339),
			"ended_at":    time.Now().UTC().Format(time.RFC3339),
			"reason":      reason,
		},
	})
}

// Support reads must not expose reusable bearer credentials for the target.
func (a *app) calendarFeedTokenForActor(ac authCtx) (string, error) {
	if ac.supportView != nil {
		return "", fmt.Errorf("calendar subscription unavailable in support view")
	}
	return a.calendarFeedToken(ac.email, ac.tenant.Slug)
}

func (a *app) portalContextsForActor(ac *authCtx) []portalContextView {
	if ac.supportView != nil {
		return []portalContextView{{TenantSlug: ac.tenant.Slug, HouseName: houseDisplayName(ac.tenant), Address: ac.tenant.Address, Role: ac.role, Current: true}}
	}
	return a.portalContextsFor(ac.email, ac.tenant.Slug, ac.role)
}

func (a *app) supportContextForSession(session auth.Session) *supportViewContext {
	profile := a.profileForTenant(session.SupportTargetEmail, session.TenantSlug)
	return &supportViewContext{ActorEmail: session.Email, ActorRole: session.Role, TargetEmail: session.SupportTargetEmail, TargetName: profile.DisplayName(), TargetRole: session.SupportTargetRole, StartedAt: time.Unix(session.SupportStartedAt, 0), ExpiresAt: time.Unix(session.SupportExpiresAt, 0)}
}
