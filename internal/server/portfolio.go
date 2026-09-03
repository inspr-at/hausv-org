package server

import "net/http"

func (a *app) portfolioPage(w http.ResponseWriter, r *http.Request, ac authCtx) {
	a.renderVerwaltungPage(w, r, ac, "portfolio", "Portfolio")
}
