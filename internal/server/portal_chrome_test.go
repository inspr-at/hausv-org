package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/web"
)

func TestPortalSidebarWidthCookieIsRenderedBeforeAssets(t *testing.T) {
	a, _, _ := newInboxTestApp(t, roleAdmin)
	token, _, err := a.sessions.Put("vera@example.com", "demo", authMethodEmail, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	for _, route := range []string{
		"/demo/app", "/demo/app/announcements", "/demo/app/anliegen/board",
		"/demo/app/verwaltung", "/demo/app/verwaltung/posteingang",
		"/demo/app/verwaltung/rechte", "/demo/app/verwaltung/textbausteine",
		"/demo/app/verwaltung/einstellungen", "/demo/app/settings/modules",
		"/demo/app/parking/settings",
	} {
		for _, tc := range []struct{ cookie, want string }{
			{"", "280"}, {"240", "240"}, {"360", "360"}, {"420", "420"},
			{"239", "280"}, {"421", "280"}, {"NaN", "280"}, {"360px", "280"},
			{"360.5", "280"}, {"9999999999999999999999999", "280"}, {"360;background:red", "280"},
		} {
			t.Run(route+"/"+tc.cookie, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "http://hausv.org"+route, nil)
				req.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
				if tc.cookie != "" {
					req.Header.Add("Cookie", `hausv-sidebar-w="`+tc.cookie+`"`)
				}
				response := httptest.NewRecorder()
				a.handler().ServeHTTP(response, req)
				if response.Code != http.StatusOK {
					t.Fatalf("status %d", response.Code)
				}
				html := response.Body.String()
				want := `<html lang="de-AT" style="--sidebar-w:` + tc.want + `px;">`
				if !strings.Contains(html, want) {
					t.Fatalf("document must start with validated width %s", tc.want)
				}
				critical := strings.Index(html, "<style data-shell-critical>")
				asset := strings.Index(html, `/assets/portal-shell.css?v=`)
				if critical < 0 || asset <= critical || asset >= strings.Index(html, "</head>") {
					t.Fatal("critical shell must precede the external sheet in the head")
				}
			})
		}
	}
}

func TestAuthenticatedPortalShellsIncludeMapTiles(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "admin@example.com",
		Role:        roleAdmin,
		Tenants:     []string{"demo"},
		AuthMethods: defaultAuthMethods(),
	})
	if view := sidebarMapForTenant(a.tenants["demo"]); !view.Configured {
		t.Fatal("demo tenant must have coordinates so the shell can emit map tiles")
	}

	routes := []string{
		"/demo/app",
		"/demo/app/audit",
		"/demo/app/announcements",
		"/demo/app/events",
		"/demo/app/kontakte",
		"/demo/app/dokumente",
		"/demo/app/anliegen",
		"/demo/app/anliegen/board",
		"/demo/app/abstimmungen",
		"/demo/app/parking",
		"/demo/app/uebergaben",
		"/demo/app/zuhause/onboarding",
		"/demo/app/hilfe",
		"/demo/app/settings",
		"/demo/app/settings/home",
		"/demo/app/energie",
	}
	for _, route := range routes {
		page := authedRequest(t, a, "admin@example.com", route)
		if page.Code == http.StatusSeeOther || page.Code == http.StatusFound {
			location := page.Header().Get("Location")
			if location == "" {
				t.Fatalf("%s redirected without Location", route)
			}
			page = authedRequest(t, a, "admin@example.com", location)
		}
		if page.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want 200", route, page.Code)
		}
		body := page.Body.String()
		if !strings.Contains(body, "/map-tiles/") {
			t.Fatalf("%s authenticated shell is missing /map-tiles/ — the portal-context builder forgot Map", route)
		}
		if !strings.Contains(body, `class="map `) {
			t.Fatalf("%s authenticated shell dropped aside.sidebar a.map", route)
		}
	}

	ac := authCtx{email: "admin@example.com", role: roleAdmin, tenant: a.tenants["demo"], tenantRef: testTenantRef("demo")}
	for name, data := range map[string]web.PortalPageData{
		"energy":   a.energyPortalContext(ac, "Energie"),
		"settings": a.settingsPortalContext(ac, "Einstellungen", "settings"),
		"parking":  a.parkingPortalContext(ac),
		"contacts": a.contactsPortalContext(ac),
	} {
		if !data.Map.Configured || len(data.Map.Tiles) == 0 {
			t.Fatalf("%s wrapper lost Map tiles: %+v", name, data.Map)
		}
		if data.HeroImageURL == "" {
			t.Fatalf("%s wrapper lost HeroImageURL", name)
		}
	}

	home := authedRequest(t, a, "admin@example.com", "/demo/app").Body.String()
	if got := strings.Count(home, `data-portal-section-hero`); got != 1 {
		t.Fatalf("home must render exactly one shared hero, got %d", got)
	}
	if strings.Count(home, `<div class="greeting">`) != 1 {
		t.Fatal("home greeting must be the title slot inside the shared hero")
	}
	if !strings.Contains(home, `<img class="portal-section-hero-image" src="/demo`+defaultTenantHeroImageURL) {
		t.Fatalf("home hero must use <img src> so prefixTenantHTMLPaths can rewrite it")
	}
	if strings.Contains(home, "url('") && strings.Contains(home, defaultTenantHeroImageURL) {
		t.Fatal("home hero must not put the tenant image in a quoted CSS url()")
	}
}
