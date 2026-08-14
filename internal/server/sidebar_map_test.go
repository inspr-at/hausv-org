package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestMapPositionFromFormAcceptsCoordinatesAndAllowsClear(t *testing.T) {
	latitude, longitude, zoom, err := mapPositionFromForm("48.208200, 16.373800")
	if err != nil || latitude != 48.2082 || longitude != 16.3738 || zoom != defaultMapZoom {
		t.Fatalf("map position = %v, %v, %v, %v", latitude, longitude, zoom, err)
	}
	latitude, longitude, zoom, err = mapPositionFromForm("  ")
	if err != nil || latitude != 0 || longitude != 0 || zoom != 0 {
		t.Fatalf("cleared map position = %v, %v, %v, %v", latitude, longitude, zoom, err)
	}
	for _, raw := range []string{"48.2", "north, east", "91, 16", "0, 0"} {
		if _, _, _, err := mapPositionFromForm(raw); err == nil {
			t.Fatalf("map position %q should be rejected", raw)
		}
	}
}

func TestTenantOverridePersistsDynamicPortalMapPosition(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	if err := a.tenantOverrides.SetMeta("demo", tenantOverride{
		MetaSet:      true,
		Name:         "Aktuelles Haus",
		Address:      "Neue Adresse 7",
		MapSet:       true,
		MapLatitude:  48.2082,
		MapLongitude: 16.3738,
		MapZoom:      defaultMapZoom,
		BrandIcon:    tenantBrandCommunity,
	}); err != nil {
		t.Fatalf("SetMeta: %v", err)
	}
	tenant, ok := a.tenantBySlug("demo")
	if !ok || tenant.Name != "Aktuelles Haus" || tenant.Address != "Neue Adresse 7" {
		t.Fatalf("tenant = %+v, ok=%v", tenant, ok)
	}
	if view := sidebarMapForTenant(tenant); !view.Configured || len(view.Tiles) == 0 {
		t.Fatalf("persisted map view = %+v", view)
	}
}

func TestSidebarMapUsesOnlyVisibleTilesAndExactHouseCentre(t *testing.T) {
	view := sidebarMapForTenant(tenantConfig{
		Slug:         "home",
		Address:      "Testweg 1",
		MapLatitude:  48.2082,
		MapLongitude: 16.3738,
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
		MapLatitude:  48.2082,
		MapLongitude: 16.3738,
		MapZoom:      17,
	})
	if !view.Configured || len(view.Tiles) < 2 || len(view.Tiles) > 6 {
		t.Fatalf("public map = %#v", view)
	}
	if got := string(view.Tiles[0].Style); !strings.Contains(got, "left:calc(50%") {
		t.Fatalf("public tile style = %q", got)
	}
}

func TestMapTileProxyRestrictsTilesAndCachesUpstreamResponse(t *testing.T) {
	var requests atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if !strings.HasPrefix(r.Header.Get("User-Agent"), "hausv.org/") {
			t.Errorf("user agent = %q", r.Header.Get("User-Agent"))
		}
		if r.Header.Get("Referer") != "http://localhost:8080/demo/app" {
			t.Errorf("referer = %q", r.Header.Get("Referer"))
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("\x89PNG\r\n\x1a\nfixture"))
	}))
	defer upstream.Close()

	a := newTestPortalApp(t, userProfile{
		Email:       "resident@example.com",
		Role:        roleResident,
		Tenants:     []string{"demo"},
		AuthMethods: defaultAuthMethods(),
	})
	a.dataDir = t.TempDir()
	a.mapTileBaseURL = upstream.URL
	tile := sidebarMapForTenant(a.tenants["demo"]).Tiles[0]

	for range 2 {
		request := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo"+tile.URL, nil)
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
	a.handler().ServeHTTP(denied, httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/map-tiles/17/1/1.png", nil))
	if denied.Code != http.StatusNotFound {
		t.Fatalf("unconfigured tile status = %d, want 404", denied.Code)
	}
}
