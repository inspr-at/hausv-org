package server

import "github.com/inspr-at/hausv-org/internal/web"

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
