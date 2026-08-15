package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type stubAddressGeocoder struct {
	results []geocodeResult
	err     error
}

func (g stubAddressGeocoder) Search(context.Context, string) ([]geocodeResult, error) {
	return g.results, g.err
}

func TestBuildingGeocodingReturnsReviewablePreviewWithoutPersisting(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.geocoder = stubAddressGeocoder{results: []geocodeResult{{Label: "Musterweg 1, 8010 Graz", Latitude: 47.0707, Longitude: 15.4395}}}

	response := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/building/geocode", url.Values{"address": {"Musterweg 1, Graz"}})
	if response.Code != http.StatusOK {
		t.Fatalf("geocode status = %d, body = %s", response.Code, response.Body.String())
	}
	var payload struct {
		Results []geocodeResultView `json:"results"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode geocode response: %v", err)
	}
	if len(payload.Results) != 1 || payload.Results[0].Position != "47.070700, 15.439500" || len(payload.Results[0].Tiles) == 0 {
		t.Fatalf("geocode response = %+v", payload.Results)
	}
	if !strings.HasPrefix(payload.Results[0].Tiles[0].URL, "/demo/map-tiles/") {
		t.Fatalf("preview tile URL = %q", payload.Results[0].Tiles[0].URL)
	}
	if tenant, _ := a.tenantBySlug("demo"); tenant.MapLatitude == 47.0707 || tenant.MapLongitude == 15.4395 {
		t.Fatal("preview must not persist coordinates before the settings form is saved")
	}
}

func TestNominatimGeocoderIdentifiesAndCachesRequests(t *testing.T) {
	requests := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/search" || r.URL.Query().Get("countrycodes") != "at" || r.URL.Query().Get("limit") != "3" {
			t.Fatalf("unexpected search request: %s", r.URL.String())
		}
		if !strings.Contains(r.Header.Get("User-Agent"), "hausv.org/") || r.Header.Get("Referer") != "https://hausv.org/" {
			t.Fatalf("missing provider identification headers: %#v", r.Header)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"display_name":"Musterweg 1, Graz","lat":"47.0707","lon":"15.4395"}]`))
	}))
	defer upstream.Close()

	geocoder := newNominatimGeocoder(upstream.URL)
	for range 2 {
		results, err := geocoder.Search(context.Background(), " Musterweg 1, Graz ")
		if err != nil || len(results) != 1 {
			t.Fatalf("search results = %+v, err = %v", results, err)
		}
	}
	if requests != 1 {
		t.Fatalf("upstream requests = %d, want one cached request", requests)
	}
}

func TestBuildingSettingsRendersOnlySelectedWorkspace(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})

	overview := authedRequest(t, a, "manager@example.com", "/demo/app/settings/building?section=overview").Body.String()
	for _, want := range []string{"Standort aus Adresse ermitteln", "Kartenposition", `data-building-form`} {
		if !strings.Contains(overview, want) {
			t.Fatalf("overview missing %q", want)
		}
	}
	if strings.Contains(overview, `id="contacts"`) || strings.Contains(overview, `id="units"`) {
		t.Fatal("overview must not render inactive workspaces")
	}

	contacts := authedRequest(t, a, "manager@example.com", "/demo/app/settings/building?section=contacts").Body.String()
	for _, want := range []string{"Hausverwaltung", "Notdienst", "Hausmeister", "Kontakte speichern", `action="/demo/app/settings/building/contacts"`} {
		if !strings.Contains(contacts, want) {
			t.Fatalf("contacts workspace missing %q", want)
		}
	}
	if strings.Contains(contacts, "Standort aus Adresse ermitteln") {
		t.Fatal("contacts must not render the master-data workspace")
	}
}

func TestBuildingOnlySavePreservesContactsAndAppearance(t *testing.T) {
	tenant := tenantConfig{Name: "Alt", Address: "Altweg 1", BrandIcon: tenantBrandParking, BrandAbbreviation: "ALT", ContactName: "Verwaltung", ContactEmail: "office@example.com"}
	override, err := tenantBuildingOverrideFromForm(tenant, url.Values{"name": {"Neu"}, "address": {"Neuweg 2"}, "map_position": {"47.070700, 15.439500"}})
	if err != nil {
		t.Fatalf("building override: %v", err)
	}
	if override.ContactName != tenant.ContactName || override.ContactEmail != tenant.ContactEmail || override.BrandIcon != tenant.BrandIcon || override.BrandAbbreviation != tenant.BrandAbbreviation {
		t.Fatalf("unrelated metadata changed: %+v", override)
	}
}
