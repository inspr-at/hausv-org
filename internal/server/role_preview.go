package server

import "github.com/inspr-at/hausv-org/internal/web"

import "net/http"

func (a *app) rolePreviewStart(w http.ResponseWriter, r *http.Request, ac authCtx) {
	a.rolePreviewPlaceholder(w, r, ac)
}

func (a *app) rolePreviewEnd(w http.ResponseWriter, r *http.Request, ac authCtx) {
	a.rolePreviewPlaceholder(w, r, ac)
}

func (a *app) rolePreviewPlaceholder(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if normalizeRole(ac.role) != roleAdmin {
		http.Error(w, "Dieser Bereich ist Admins vorbehalten.", http.StatusForbidden)
		return
	}
	http.Redirect(w, r, "/app?flash=Wird%20gebaut", http.StatusSeeOther)
}

// Stubs for the role-preview hooks in the portal page builder; HAUSV-605 replaces them.
func rolePreviewPortalData(ac *authCtx) *web.RolePreviewState { return nil }

func (a *app) rolePreviewChoices(ac *authCtx) []web.RolePreviewChoice { return nil }

func rolePreviewGreetingName(ac *authCtx, name string) string { return name }
