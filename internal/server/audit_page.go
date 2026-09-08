package server

import (
	"bytes"
	"io"
	"net/http"

	"github.com/inspr-at/hausv-org/internal/web"
)

func (a *app) auditPortalContext(ac authCtx, title string) web.PortalPageData {
	return a.portalBaseData(ac, "audit", title)
}

func (a *app) renderAuditTempl(w http.ResponseWriter, r *http.Request, data web.AuditPageData) {
	var rendered bytes.Buffer
	if err := web.AuditPage(data).Render(r.Context(), &rendered); err != nil {
		logError("templ audit render failed", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, prefixTenantHTMLPaths(rendered.String(), data.Portal.TenantSlug))
}
