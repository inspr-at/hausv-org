package server

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
