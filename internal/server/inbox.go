package server

import "net/http"

func (a *app) inboxPage(w http.ResponseWriter, r *http.Request, ac authCtx) {
	a.renderVerwaltungPage(w, r, ac, "inbox", "Posteingang")
}

func (a *app) inboxCasePage(w http.ResponseWriter, r *http.Request, ac authCtx) {
	a.renderVerwaltungPage(w, r, ac, "inbox", "Posteingang")
}

func (a *app) inboxCaseAction(w http.ResponseWriter, r *http.Request, _ authCtx) {
	http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)
}

func (a *app) phoneNoteAction(w http.ResponseWriter, r *http.Request, _ authCtx) {
	http.Redirect(w, r, "/app/verwaltung/posteingang", http.StatusSeeOther)
}

func (a *app) verwaltungSettingsPage(w http.ResponseWriter, r *http.Request, ac authCtx) {
	a.renderVerwaltungPage(w, r, ac, "settings", "Einstellungen")
}

func (a *app) verwaltungSettingsAction(w http.ResponseWriter, r *http.Request, _ authCtx) {
	http.Redirect(w, r, "/app/verwaltung/einstellungen", http.StatusSeeOther)
}
