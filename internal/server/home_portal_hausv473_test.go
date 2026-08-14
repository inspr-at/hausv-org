package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
)

func TestHomePortalActivationCreatesTenantOwnerSessionAndDirectEntry(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	handler := a.handler()
	setupCookie := confirmedHomeSetupCookieHAUSV471(t, a, "stadtpark-home", "owner@example.com")

	setup := homeConnectorSetupRequestHAUSV471(t, handler, http.MethodGet, "/start/connector", setupCookie, nil)
	for _, want := range []string{"Portal jetzt aktivieren", "gemeinsam an", "hausv.org/stadtpark-home"} {
		if setup.Code != http.StatusOK || !strings.Contains(setup.Body.String(), want) {
			t.Fatalf("setup missing %q status=%d", want, setup.Code)
		}
	}
	activation := homeConnectorSetupRequestHAUSV471(t, handler, http.MethodPost, "/start/activate", setupCookie, url.Values{})
	if activation.Code != http.StatusSeeOther || activation.Header().Get("Location") != "/stadtpark-home/app?activated=1" {
		t.Fatalf("activation status=%d location=%q body=%q", activation.Code, activation.Header().Get("Location"), activation.Body.String())
	}
	var portalCookie *http.Cookie
	for _, cookie := range activation.Result().Cookies() {
		if cookie.Name == "weg_session" {
			portalCookie = cookie
		}
	}
	if portalCookie == nil || !portalCookie.HttpOnly || portalCookie.Path != "/" {
		t.Fatalf("portal cookie=%+v", portalCookie)
	}
	tenant, found := a.tenantBySlug("stadtpark-home")
	if !found || tenant.PortalType != "house" || tenant.Name != "Zuhause am Stadtpark" || tenant.BrandIcon != tenantBrandSingleHome {
		t.Fatalf("activated tenant=%+v found=%v", tenant, found)
	}
	if !a.isAllowed("owner@example.com", "stadtpark-home") || a.roleFor("owner@example.com", "stadtpark-home") != roleOwner {
		t.Fatal("confirmed owner did not receive tenant-scoped owner access")
	}
	portalRequest := httptest.NewRequest(http.MethodGet, "http://hausv.org/stadtpark-home/app", nil)
	portalRequest.AddCookie(portalCookie)
	portalPage := httptest.NewRecorder()
	handler.ServeHTTP(portalPage, portalRequest)
	if portalPage.Code != http.StatusOK || !strings.Contains(portalPage.Body.String(), "Zuhause am Stadtpark") {
		t.Fatalf("portal status=%d body=%q", portalPage.Code, portalPage.Body.String())
	}

	repeat := homeConnectorSetupRequestHAUSV471(t, handler, http.MethodPost, "/start/activate", setupCookie, url.Values{})
	if repeat.Code != http.StatusSeeOther || repeat.Header().Get("Location") != activation.Header().Get("Location") {
		t.Fatalf("repeat activation status=%d location=%q", repeat.Code, repeat.Header().Get("Location"))
	}
	confirmed, _, _ := a.homeReservations.Get("stadtpark-home")
	if confirmed.Status != store.HomeReservationActive {
		t.Fatalf("reservation status=%q", confirmed.Status)
	}
	if !a.homePathReservableBy("stadtpark-home", "owner@example.com") || a.homePathReservableBy("stadtpark-home", "other@example.com") {
		t.Fatal("only the confirmed owner may request a fresh setup link for an active path")
	}
	activeSetup := homeConnectorSetupRequestHAUSV471(t, handler, http.MethodGet, "/start/connector", setupCookie, nil)
	if !strings.Contains(activeSetup.Body.String(), "Privates Portal öffnen") || strings.Contains(activeSetup.Body.String(), "Portal jetzt aktivieren") {
		t.Fatalf("active setup body=%q", activeSetup.Body.String())
	}
}

func TestHomePortalActivationRejectsForeignSessionAndStaticCollision(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	handler := a.handler()
	ownerCookie := confirmedHomeSetupCookieHAUSV471(t, a, "sicheres-home", "owner@example.com")
	foreignToken, _, err := a.homeSetupSessions.Put("other@example.com", "sicheres-home", authMethodEmail, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	foreign := homeConnectorSetupRequestHAUSV471(t, handler, http.MethodPost, "/start/activate", &http.Cookie{Name: homeSetupCookieName, Value: foreignToken}, url.Values{})
	if foreign.Code != http.StatusSeeOther || foreign.Header().Get("Location") != "/start" {
		t.Fatalf("foreign activation status=%d location=%q", foreign.Code, foreign.Header().Get("Location"))
	}
	wrongOrigin := httptest.NewRequest(http.MethodPost, "http://hausv.org/start/activate", nil)
	wrongOrigin.Header.Set("Origin", "https://example.test")
	wrongOrigin.AddCookie(ownerCookie)
	blocked := httptest.NewRecorder()
	handler.ServeHTTP(blocked, wrongOrigin)
	if blocked.Code != http.StatusForbidden {
		t.Fatalf("wrong-origin activation status=%d", blocked.Code)
	}

	// A path that becomes statically configured after reservation still wins and
	// cannot be shadowed by a self-service portal.
	a.tenants["sicheres-home"] = tenantConfig{Slug: "sicheres-home", Name: "Konfiguriert", Address: "Fix"}
	collision := homeConnectorSetupRequestHAUSV471(t, handler, http.MethodPost, "/start/activate", ownerCookie, url.Values{})
	if collision.Code != http.StatusConflict {
		t.Fatalf("static collision status=%d body=%q", collision.Code, collision.Body.String())
	}
	if _, found, _ := a.homePortals.Get("sicheres-home"); found {
		t.Fatal("static collision created a dynamic portal")
	}
}
