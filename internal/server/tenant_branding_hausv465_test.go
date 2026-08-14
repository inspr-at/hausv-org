package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPathTenantUsesConfiguredMapAndPrivateHeroSeed(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "admin@example.com",
		Role:        roleAdmin,
		Tenants:     []string{"demo"},
		AuthMethods: defaultAuthMethods(),
	})
	seedDir := t.TempDir()
	seedName := "demo-hero.jpg"
	seed := []byte("fixture hero")
	if err := os.WriteFile(filepath.Join(seedDir, seedName), seed, 0o600); err != nil {
		t.Fatalf("write hero seed: %v", err)
	}
	tenant := a.tenants["demo"]
	tenant.HeroImageFile = seedName
	a.tenants["demo"] = tenant
	a.tenantHeroSeedDir = seedDir

	page := httptest.NewRecorder()
	a.handler().ServeHTTP(page, httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/", nil))
	if page.Code != http.StatusOK {
		t.Fatalf("tenant page status = %d", page.Code)
	}
	for _, want := range []string{
		`url('/demo/tenant-hero/demo')`,
		`class="location-map location-map-configured"`,
		`data-map-tile="/demo/map-tiles/17/`,
	} {
		if !strings.Contains(page.Body.String(), want) {
			t.Fatalf("path tenant page missing %q", want)
		}
	}

	hero := httptest.NewRecorder()
	a.handler().ServeHTTP(hero, httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/tenant-hero/demo", nil))
	if hero.Code != http.StatusOK || hero.Body.String() != string(seed) {
		t.Fatalf("seed hero response = %d/%q", hero.Code, hero.Body.String())
	}
}
