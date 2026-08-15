package server

import (
	"bytes"
	"net/http"

	"github.com/inspr-at/hausv-org/internal/version"
	"github.com/inspr-at/hausv-org/internal/web"
)

// templExample is a toolchain smoke test, not a migrated product route. It is
// only registered when TEMPL_EXAMPLE_ENABLED is true.
func (a *app) templExample(w http.ResponseWriter, r *http.Request) {
	var rendered bytes.Buffer
	if err := web.TemplExample(version.AssetVersion()).Render(r.Context(), &rendered); err != nil {
		logError("templ example render failed", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = rendered.WriteTo(w)
}
