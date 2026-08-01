package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestSidebarMapUsesOnlyVisibleTilesAndExactHouseCentre(t *testing.T) {
	view := sidebarMapForTenant(tenantConfig{
		Slug:         "home",
		Address:      "Testweg 1",
		MapLatitude:  47.1008592,
		MapLongitude: 15.4717681,
		MapZoom:      17,
	})
	if !view.Configured || len(view.Tiles) == 0 || len(view.Tiles) > 4 {
		t.Fatalf("sidebar map = %#v", view)
	}
	for _, tile := range view.Tiles {
		if !strings.HasPrefix(tile.URL, "/map-tiles/17/") || !strings.Contains(string(tile.Style), "left:calc(50%") {
			t.Fatalf("tile = %#v", tile)
		}
	}
}

func TestPublicMapCoversTheWidestLoggedOutCard(t *testing.T) {
	view := publicMapForTenant(tenantConfig{
		Slug:         "home",
		Address:      "Testweg 1",
		MapLatitude:  47.1008592,
		MapLongitude: 15.4717681,
		MapZoom:      17,
	})
	if !view.Configured || len(view.Tiles) != 2 {
		t.Fatalf("public map = %#v", view)
	}
	if got := string(view.Tiles[0].Style); !strings.Contains(got, "left:calc(50% + -281.31px)") {
		t.Fatalf("left public tile style = %q", got)
	}
	if got := string(view.Tiles[1].Style); !strings.Contains(got, "left:calc(50% + -25.31px)") {
		t.Fatalf("right public tile style = %q", got)
	}
}

func TestMapTileProxyRestrictsTilesAndCachesUpstreamResponse(t *testing.T) {
	var requests atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if !strings.HasPrefix(r.Header.Get("User-Agent"), "hausv.org/") {
			t.Errorf("user agent = %q", r.Header.Get("User-Agent"))
		}
		if r.Header.Get("Referer") != "https://jhw22.hausv.org/app" {
			t.Errorf("referer = %q", r.Header.Get("Referer"))
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("\x89PNG\r\n\x1a\nfixture"))
	}))
	defer upstream.Close()

	a := newTestPortalApp(t, userProfile{
		Email:       "resident@example.com",
		Role:        roleResident,
		Tenants:     []string{"jhw22"},
		AuthMethods: defaultAuthMethods(),
	})
	a.dataDir = t.TempDir()
	a.mapTileBaseURL = upstream.URL
	tile := sidebarMapForTenant(a.tenants["jhw22"]).Tiles[0]

	for range 2 {
		request := httptest.NewRequest(http.MethodGet, "http://jhw22.hausv.org"+tile.URL, nil)
		response := httptest.NewRecorder()
		a.handler().ServeHTTP(response, request)
		if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "image/png" {
			t.Fatalf("tile status/content type = %d/%q, body=%q", response.Code, response.Header().Get("Content-Type"), response.Body.String())
		}
		if !strings.Contains(response.Header().Get("Cache-Control"), "max-age=604800") {
			t.Fatalf("cache control = %q", response.Header().Get("Cache-Control"))
		}
	}
	if requests.Load() != 1 {
		t.Fatalf("upstream requests = %d, want 1", requests.Load())
	}

	denied := httptest.NewRecorder()
	a.handler().ServeHTTP(denied, httptest.NewRequest(http.MethodGet, "http://jhw22.hausv.org/map-tiles/17/1/1.png", nil))
	if denied.Code != http.StatusNotFound {
		t.Fatalf("unconfigured tile status = %d, want 404", denied.Code)
	}
}
