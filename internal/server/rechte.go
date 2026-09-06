package server

import (
	"bytes"
	"io"
	"net/http"

	"github.com/inspr-at/hausv-org/internal/web"
)

func (a *app) rechtePage(w http.ResponseWriter, r *http.Request, ac authCtx) {
	var rendered bytes.Buffer
	if err := web.RechtePage(a.verwaltungShell(r.Context(), &ac, "rechte")).Render(r.Context(), &rendered); err != nil {
		logError("templ rechte render failed", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, prefixTenantHTMLPaths(rendered.String(), ac.tenant.Slug))
}
