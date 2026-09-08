package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/auth"
	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/web"
)

const (
	rolePreviewTTL                    = 15 * time.Minute
	rolePreviewEndExpired             = "expired"
	rolePreviewEndActorContextInvalid = "actor_context_invalid"
	rolePreviewReadOnlyMessage        = "Die Ansicht ist schreibgeschützt. Beenden Sie die Ansicht, um Änderungen vorzunehmen."
)

type rolePreviewContext struct {
	Role      string
	StartedAt time.Time
	ExpiresAt time.Time
}

func (a *app) rolePreviewStart(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if ac.preview != nil || normalizeRole(ac.realRole) != roleAdmin {
		http.Error(w, "Dieser Bereich ist Admins vorbehalten.", http.StatusForbidden)
		return
	}
	previewRole := normalizeRole(r.FormValue("role"))
	if previewRole != roleOwner && previewRole != roleResident {
		http.Error(w, "Ungültige Rollenansicht.", http.StatusForbidden)
		return
	}
	cookie, err := r.Cookie("weg_session")
	if err != nil {
		http.Error(w, "Anmeldung erforderlich", http.StatusUnauthorized)
		return
	}
	current, ok := a.sessions.GetSession(cookie.Value)
	if !ok || current.Email != ac.email || current.TenantSlug != ac.tenant.Slug || current.PreviewRole != "" {
		http.Error(w, "Anmeldung erforderlich", http.StatusUnauthorized)
		return
	}
	current.Role = roleAdmin
	now := time.Now().UTC().Truncate(time.Second)
	parentExpiresAt := time.Unix(current.ExpiresAt, 0)
	previewExpiresAt := now.Add(rolePreviewTTL)
	if parentExpiresAt.Before(previewExpiresAt) {
		previewExpiresAt = parentExpiresAt
	}
	if !previewExpiresAt.After(now) {
		http.Error(w, "Anmeldung erforderlich", http.StatusUnauthorized)
		return
	}
	token, expiresAt, err := a.sessions.PutRolePreview(current, previewRole, now, previewExpiresAt)
	if err != nil {
		http.Error(w, "Rollenansicht konnte nicht gestartet werden.", http.StatusInternalServerError)
		return
	}
	setRolePreviewCookie(w, token, expiresAt, a.sessionSecure)
	a.sessions.Delete(cookie.Value)
	a.recordAudit(auditEvent{
		TenantSlug: current.TenantSlug,
		ActorEmail: current.Email,
		ActorRole:  roleAdmin,
		Action:     store.AuditActionRolePreviewStart,
		TargetType: "role",
		TargetID:   previewRole,
		Summary:    "Rollenansicht gestartet",
		Details: map[string]string{
			"preview_role": previewRole,
			"tenant":       current.TenantSlug,
			"expires_at":   previewExpiresAt.Format(time.RFC3339),
		},
	})
	http.Redirect(w, r, strings.TrimRight(a.baseURL, "/")+ac.tenant.PublicURL("/app"), http.StatusSeeOther)
}

func (a *app) rolePreviewEnd(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if ac.preview == nil || normalizeRole(ac.realRole) != roleAdmin {
		http.Error(w, "Keine aktive Rollenansicht.", http.StatusForbidden)
		return
	}
	cookie, err := r.Cookie("weg_session")
	if err != nil {
		http.Error(w, "Anmeldung erforderlich", http.StatusUnauthorized)
		return
	}
	current, ok := a.sessions.GetSession(cookie.Value)
	if !ok || current.PreviewRole == "" || current.Email != ac.email || current.TenantSlug != ac.tenant.Slug {
		http.Error(w, "Anmeldung erforderlich", http.StatusUnauthorized)
		return
	}
	token, expiresAt, err := a.sessions.EndRolePreview(current)
	if err != nil {
		http.Error(w, "Rollenansicht konnte nicht beendet werden.", http.StatusInternalServerError)
		return
	}
	setRolePreviewCookie(w, token, expiresAt, a.sessionSecure)
	a.recordRolePreviewEnd(current, "explicit")
	a.sessions.Delete(cookie.Value)
	http.Redirect(w, r, strings.TrimRight(a.baseURL, "/")+ac.tenant.PublicURL("/app"), http.StatusSeeOther)
}

func (a *app) rolePreviewSessionEndReason(session auth.Session) string {
	if session.PreviewRole == "" {
		return rolePreviewEndActorContextInvalid
	}
	if session.PreviewExpiresAt <= time.Now().Unix() {
		return rolePreviewEndExpired
	}
	if session.Role != roleAdmin || a.roleFor(session.Email, session.TenantSlug) != roleAdmin ||
		!a.ownPortalContextAllowed(session.Email, session.TenantSlug, roleAdmin, session.AuthMethod) {
		return rolePreviewEndActorContextInvalid
	}
	return ""
}

func (a *app) terminateRolePreview(w http.ResponseWriter, r *http.Request, session auth.Session, reason string) {
	if cookie, err := r.Cookie("weg_session"); err == nil {
		a.sessions.Delete(cookie.Value)
	}
	restored := false
	parentExpiresAt := time.Unix(session.ExpiresAt, 0)
	realRole := a.roleFor(session.Email, session.TenantSlug)
	if parentExpiresAt.After(time.Now()) && a.ownPortalContextAllowed(session.Email, session.TenantSlug, realRole, session.AuthMethod) {
		if token, expiresAt, err := a.sessions.PutSession(session.Email, session.TenantSlug, session.AuthMethod, realRole, parentExpiresAt); err == nil {
			setRolePreviewCookie(w, token, expiresAt, a.sessionSecure)
			restored = true
		}
	}
	if !restored {
		clearRolePreviewCookie(w, a.sessionSecure)
	}
	a.recordRolePreviewEnd(session, reason)
	target := "/"
	if restored {
		if tenant, ok := a.tenantBySlug(session.TenantSlug); ok {
			target = tenant.PublicURL("/app")
		}
	}
	http.Redirect(w, r, strings.TrimRight(a.baseURL, "/")+target, http.StatusSeeOther)
}

func (a *app) recordRolePreviewEnd(session auth.Session, reason string) {
	if session.PreviewRole == "" {
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: session.TenantSlug,
		ActorEmail: session.Email,
		ActorRole:  session.Role,
		Action:     store.AuditActionRolePreviewEnd,
		TargetType: "role",
		TargetID:   session.PreviewRole,
		Summary:    "Rollenansicht beendet",
		Details: map[string]string{
			"preview_role": session.PreviewRole,
			"tenant":       session.TenantSlug,
			"reason":       reason,
		},
	})
}

func setRolePreviewCookie(w http.ResponseWriter, token string, expiresAt time.Time, secure bool) {
	http.SetCookie(w, &http.Cookie{Name: "weg_session", Value: token, Path: "/", Expires: expiresAt, HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode})
}

func clearRolePreviewCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{Name: "weg_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode})
}

func rolePreviewPortalData(ac *authCtx) *web.RolePreviewState {
	if ac == nil || ac.preview == nil {
		return nil
	}
	minutes := int(time.Until(ac.preview.ExpiresAt).Minutes()) + 1
	if minutes < 1 {
		minutes = 1
	}
	return &web.RolePreviewState{RoleLabel: ac.preview.Role, HouseName: houseDisplayName(ac.tenant), EndsInMinutes: minutes, EndPath: "/app/ansicht/ende"}
}

func (a *app) rolePreviewChoices(ac *authCtx) []web.RolePreviewChoice {
	if ac == nil || ac.preview != nil || normalizeRole(ac.realRole) != roleAdmin {
		return nil
	}
	return []web.RolePreviewChoice{
		{Role: roleAdmin, Label: "Hausverwaltung", Description: "Ihre Rolle · Portfolio, Posteingang, alle Liegenschaften", Current: true},
		{Role: roleOwner, Label: roleOwner, Description: "Ruhiger Hausüberblick, Dokumente, Abstimmungen", StartPath: "/app/ansicht/start"},
		{Role: roleResident, Label: roleResident, Description: "Hausüberblick, Aushang, eigene Anliegen", StartPath: "/app/ansicht/start"},
	}
}

func rolePreviewGreetingName(ac *authCtx, name string) string {
	if ac != nil && ac.preview != nil {
		return ""
	}
	return name
}
