package server

import (
	"net/http"
	"strconv"

	"github.com/inspr-at/hausv-org/internal/web"
)

// Read the width before rendering so the first paint matches the next page.
// An integer in this closed range is the only cookie data admitted into CSS.
func portalChromePreferences(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		width := 280
		if cookie, err := r.Cookie("hausv-sidebar-w"); err == nil {
			if value, err := strconv.Atoi(cookie.Value); err == nil && value >= 240 && value <= 420 {
				width = value
			}
		}
		next.ServeHTTP(w, r.WithContext(web.WithSidebarWidth(r.Context(), width)))
	})
}

// portalMapForTenant copies the OSM tile set onto the shared templ shell.
// Every portal-context builder must set Map from this helper; a missed page
// renders the SVG stub even when the tenant has coordinates.
func portalMapForTenant(tenant tenantConfig) web.PortalMap {
	view := sidebarMapForTenant(tenant)
	tiles := make([]web.PortalMapTile, 0, len(view.Tiles))
	for _, tile := range view.Tiles {
		tiles = append(tiles, web.PortalMapTile{URL: tile.URL, Style: string(tile.Style)})
	}
	return web.PortalMap{Configured: view.Configured, Tiles: tiles}
}
