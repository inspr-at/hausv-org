package server

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/version"
)

const (
	defaultMapZoom       = 17
	mapTileSize          = 256
	sidebarMapWidth      = 264
	sidebarMapHeight     = 210
	publicMapWidth       = 420
	publicMapHeight      = 92
	mapTileCacheMaxAge   = 30 * 24 * time.Hour
	mapTileResponseBytes = 1 << 20
	osmTileBaseURL       = "https://tile.openstreetmap.org"
)

type sidebarMapTileView struct {
	URL   string
	Style template.CSS
	Z     int
	X     int
	Y     int
}

type sidebarMapView struct {
	Configured bool
	Tiles      []sidebarMapTileView
}

func sidebarMapForTenant(tenant tenantConfig) sidebarMapView {
	return mapViewForTenant(tenant, sidebarMapWidth, sidebarMapHeight)
}

func publicMapForTenant(tenant tenantConfig) sidebarMapView {
	return mapViewForTenant(tenant, publicMapWidth, publicMapHeight)
}

func mapViewForTenant(tenant tenantConfig, width, height int) sidebarMapView {
	latitude, longitude, zoom, ok := tenantMapCoordinates(tenant)
	if !ok {
		return sidebarMapView{}
	}

	n := math.Exp2(float64(zoom))
	x := (longitude + 180) / 360 * n
	latitudeRadians := latitude * math.Pi / 180
	y := (1 - math.Asinh(math.Tan(latitudeRadians))/math.Pi) / 2 * n
	globalX, globalY := x*mapTileSize, y*mapTileSize
	startX := int(math.Floor((globalX - float64(width)/2) / mapTileSize))
	endX := int(math.Floor((globalX + float64(width)/2 - 0.01) / mapTileSize))
	startY := int(math.Floor((globalY - float64(height)/2) / mapTileSize))
	endY := int(math.Floor((globalY + float64(height)/2 - 0.01) / mapTileSize))

	tiles := make([]sidebarMapTileView, 0, 4)
	for tileY := startY; tileY <= endY; tileY++ {
		for tileX := startX; tileX <= endX; tileX++ {
			left := float64(tileX*mapTileSize) - globalX
			top := float64(tileY*mapTileSize) - globalY
			tiles = append(tiles, sidebarMapTileView{
				URL:   fmt.Sprintf("/map-tiles/%d/%d/%d.png", zoom, tileX, tileY),
				Style: template.CSS(fmt.Sprintf("left:calc(50%% + %.2fpx);top:calc(50%% + %.2fpx)", left, top)),
				Z:     zoom,
				X:     tileX,
				Y:     tileY,
			})
		}
	}
	return sidebarMapView{Configured: true, Tiles: tiles}
}

func tenantMapCoordinates(tenant tenantConfig) (float64, float64, int, bool) {
	latitude, longitude := tenant.MapLatitude, tenant.MapLongitude
	if latitude < -85.0511 || latitude > 85.0511 || longitude < -180 || longitude > 180 ||
		(latitude == 0 && longitude == 0) {
		return 0, 0, 0, false
	}
	zoom := tenant.MapZoom
	if zoom == 0 {
		zoom = defaultMapZoom
	}
	if zoom < 15 || zoom > 18 {
		return 0, 0, 0, false
	}
	return latitude, longitude, zoom, true
}

func (a *app) mapTile(w http.ResponseWriter, r *http.Request) {
	z, errZ := strconv.Atoi(r.PathValue("z"))
	x, errX := strconv.Atoi(r.PathValue("x"))
	y, errY := strconv.Atoi(strings.TrimSuffix(r.PathValue("tile"), ".png"))
	if errZ != nil || errX != nil || errY != nil || !strings.HasSuffix(r.PathValue("tile"), ".png") {
		http.NotFound(w, r)
		return
	}

	tenant := a.tenantForRequest(r)
	allowed := false
	allowedTiles := append(sidebarMapForTenant(tenant).Tiles, publicMapForTenant(tenant).Tiles...)
	for _, tile := range allowedTiles {
		if tile.Z == z && tile.X == x && tile.Y == y {
			allowed = true
			break
		}
	}
	if !allowed {
		allowed = a.mapPreviewTileAllowed(tenant.Slug, mapTileKey{Z: z, X: x, Y: y})
	}
	if !allowed {
		http.NotFound(w, r)
		return
	}

	a.mapTileMu.Lock()
	defer a.mapTileMu.Unlock()

	cachePath := ""
	if strings.TrimSpace(a.dataDir) != "" {
		cachePath = filepath.Join(a.dataDir, "map-tiles", strconv.Itoa(z), strconv.Itoa(x), strconv.Itoa(y)+".png")
		if info, err := os.Stat(cachePath); err == nil && time.Since(info.ModTime()) < mapTileCacheMaxAge {
			serveMapTileFile(w, r, cachePath)
			return
		}
	}

	baseURL := strings.TrimRight(firstNonEmpty(a.mapTileBaseURL, osmTileBaseURL), "/")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	upstreamURL := fmt.Sprintf("%s/%d/%d/%d.png", baseURL, z, x, y)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, upstreamURL, nil)
	if err != nil {
		http.Error(w, "Kartenausschnitt nicht verfügbar", http.StatusBadGateway)
		return
	}
	req.Header.Set("User-Agent", fmt.Sprintf("hausv.org/%s (+https://hausv.org; contact: hello@hausv.org)", version.DisplayVersion(version.Version)))
	referer := tenant.PublicURL("/app")
	if !strings.HasPrefix(referer, "https://") && !strings.HasPrefix(referer, "http://") {
		referer = strings.TrimRight(a.baseURL, "/") + referer
	}
	req.Header.Set("Referer", referer)

	response, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		http.Error(w, "Kartenausschnitt nicht verfügbar", http.StatusBadGateway)
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.HasPrefix(response.Header.Get("Content-Type"), "image/png") {
		http.Error(w, "Kartenausschnitt nicht verfügbar", http.StatusBadGateway)
		return
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, mapTileResponseBytes+1))
	if err != nil || len(raw) == 0 || len(raw) > mapTileResponseBytes {
		http.Error(w, "Kartenausschnitt nicht verfügbar", http.StatusBadGateway)
		return
	}

	if cachePath != "" {
		if err := os.MkdirAll(filepath.Dir(cachePath), 0o750); err == nil {
			temporary := cachePath + ".tmp"
			if err := os.WriteFile(temporary, raw, 0o640); err == nil {
				if err := os.Rename(temporary, cachePath); err != nil {
					_ = os.Remove(temporary)
				}
			}
		}
	}
	serveMapTileBytes(w, raw)
}

func serveMapTileFile(w http.ResponseWriter, r *http.Request, path string) {
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=604800, stale-while-revalidate=2592000")
	http.ServeFile(w, r, path)
}

func serveMapTileBytes(w http.ResponseWriter, raw []byte) {
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=604800, stale-while-revalidate=2592000")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}
