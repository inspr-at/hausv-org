package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/inspr-at/hausv-org/internal/version"
)

const (
	nominatimBaseURL       = "https://nominatim.openstreetmap.org"
	geocodingResponseBytes = 256 << 10
	geocodingCacheTTL      = 30 * 24 * time.Hour
	geocodingMinInterval   = 1100 * time.Millisecond
	mapPreviewTTL          = 10 * time.Minute
)

type addressGeocoder interface {
	Search(context.Context, string) ([]geocodeResult, error)
}

type geocodeResult struct {
	Label     string
	Latitude  float64
	Longitude float64
}

type geocodeCacheEntry struct {
	Results   []geocodeResult
	ExpiresAt time.Time
}

type nominatimGeocoder struct {
	baseURL     string
	client      *http.Client
	mu          sync.Mutex
	lastRequest time.Time
	cache       map[string]geocodeCacheEntry
}

func newNominatimGeocoder(baseURL string) *nominatimGeocoder {
	return &nominatimGeocoder{
		baseURL: strings.TrimRight(firstNonEmpty(strings.TrimSpace(baseURL), nominatimBaseURL), "/"),
		client:  &http.Client{Timeout: 7 * time.Second},
		cache:   map[string]geocodeCacheEntry{},
	}
}

func (g *nominatimGeocoder) Search(ctx context.Context, address string) ([]geocodeResult, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return nil, fmt.Errorf("address required")
	}
	key := strings.ToLower(strings.Join(strings.Fields(address), " "))

	g.mu.Lock()
	defer g.mu.Unlock()
	if entry, ok := g.cache[key]; ok && time.Now().Before(entry.ExpiresAt) {
		return append([]geocodeResult(nil), entry.Results...), nil
	}
	if wait := geocodingMinInterval - time.Since(g.lastRequest); wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
		}
	}

	query := url.Values{
		"q":              {address},
		"format":         {"jsonv2"},
		"limit":          {"3"},
		"addressdetails": {"1"},
		"countrycodes":   {"at"},
		"layer":          {"address"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.baseURL+"/search?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "de")
	req.Header.Set("Referer", "https://hausv.org/")
	req.Header.Set("User-Agent", fmt.Sprintf("hausv.org/%s (+https://hausv.org; contact: hello@hausv.org)", version.DisplayVersion(version.Version)))
	g.lastRequest = time.Now()
	response, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("geocoding returned status %d", response.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, geocodingResponseBytes+1))
	if err != nil || len(raw) > geocodingResponseBytes {
		return nil, fmt.Errorf("invalid geocoding response")
	}
	var payload []struct {
		DisplayName string `json:"display_name"`
		Latitude    string `json:"lat"`
		Longitude   string `json:"lon"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("decode geocoding response: %w", err)
	}
	results := make([]geocodeResult, 0, len(payload))
	for _, item := range payload {
		latitude, latErr := strconv.ParseFloat(item.Latitude, 64)
		longitude, lonErr := strconv.ParseFloat(item.Longitude, 64)
		if latErr != nil || lonErr != nil || strings.TrimSpace(item.DisplayName) == "" {
			continue
		}
		if _, _, _, ok := tenantMapCoordinates(tenantConfig{MapLatitude: latitude, MapLongitude: longitude, MapZoom: defaultMapZoom}); !ok {
			continue
		}
		results = append(results, geocodeResult{Label: strings.TrimSpace(item.DisplayName), Latitude: latitude, Longitude: longitude})
	}
	if len(g.cache) >= 512 {
		for cachedKey, entry := range g.cache {
			if time.Now().After(entry.ExpiresAt) || len(g.cache) >= 512 {
				delete(g.cache, cachedKey)
			}
			if len(g.cache) < 512 {
				break
			}
		}
	}
	g.cache[key] = geocodeCacheEntry{Results: append([]geocodeResult(nil), results...), ExpiresAt: time.Now().Add(geocodingCacheTTL)}
	return results, nil
}

type mapTileKey struct {
	Z int
	X int
	Y int
}

type geocodeTileView struct {
	URL   string `json:"url"`
	Style string `json:"style"`
}

type geocodeResultView struct {
	Label     string            `json:"label"`
	Latitude  float64           `json:"latitude"`
	Longitude float64           `json:"longitude"`
	Position  string            `json:"position"`
	Tiles     []geocodeTileView `json:"tiles"`
}

func (a *app) geocodeBuildingAddress(w http.ResponseWriter, r *http.Request, ac authCtx) {
	address := strings.TrimSpace(r.FormValue("address"))
	if address == "" || len([]rune(address)) > 500 {
		http.Error(w, "Bitte eine gültige Adresse eingeben.", http.StatusBadRequest)
		return
	}
	if a.geocoder == nil {
		http.Error(w, "Die Standortsuche ist derzeit nicht verfügbar.", http.StatusServiceUnavailable)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	results, err := a.geocoder.Search(ctx, address)
	if err != nil {
		logError("building geocoding failed", err, "tenant", ac.tenant.Slug)
		http.Error(w, "Die Adresse konnte gerade nicht gesucht werden.", http.StatusBadGateway)
		return
	}
	if len(results) == 0 {
		http.Error(w, "Keine passende Adresse gefunden.", http.StatusNotFound)
		return
	}
	views := make([]geocodeResultView, 0, len(results))
	for _, result := range results {
		tenant := ac.tenant
		tenant.MapLatitude = result.Latitude
		tenant.MapLongitude = result.Longitude
		tenant.MapZoom = defaultMapZoom
		mapView := mapViewForTenant(tenant, 600, 260)
		a.allowMapPreviewTiles(ac.tenant.Slug, mapView.Tiles)
		tiles := make([]geocodeTileView, 0, len(mapView.Tiles))
		for _, tile := range mapView.Tiles {
			tiles = append(tiles, geocodeTileView{URL: ac.tenant.PublicURL(tile.URL), Style: string(tile.Style)})
		}
		views = append(views, geocodeResultView{
			Label: result.Label, Latitude: result.Latitude, Longitude: result.Longitude,
			Position: fmt.Sprintf("%.6f, %.6f", result.Latitude, result.Longitude), Tiles: tiles,
		})
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(map[string]any{"results": views})
}

func (a *app) allowMapPreviewTiles(tenantSlug string, tiles []sidebarMapTileView) {
	if a == nil {
		return
	}
	now := time.Now()
	a.mapPreviewMu.Lock()
	defer a.mapPreviewMu.Unlock()
	if a.mapPreviewTiles == nil {
		a.mapPreviewTiles = map[string]map[mapTileKey]time.Time{}
	}
	allowed := a.mapPreviewTiles[tenantSlug]
	if allowed == nil {
		allowed = map[mapTileKey]time.Time{}
		a.mapPreviewTiles[tenantSlug] = allowed
	}
	for key, expires := range allowed {
		if now.After(expires) {
			delete(allowed, key)
		}
	}
	for _, tile := range tiles {
		allowed[mapTileKey{Z: tile.Z, X: tile.X, Y: tile.Y}] = now.Add(mapPreviewTTL)
	}
}

func (a *app) mapPreviewTileAllowed(tenantSlug string, key mapTileKey) bool {
	if a == nil {
		return false
	}
	a.mapPreviewMu.Lock()
	defer a.mapPreviewMu.Unlock()
	expires := a.mapPreviewTiles[tenantSlug][key]
	if expires.IsZero() || time.Now().After(expires) {
		if a.mapPreviewTiles[tenantSlug] != nil {
			delete(a.mapPreviewTiles[tenantSlug], key)
		}
		return false
	}
	return true
}
