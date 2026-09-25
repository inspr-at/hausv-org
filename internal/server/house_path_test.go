package server

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestOrganisationHousePathNeverDoublePrefixes(t *testing.T) {
	path, foreign := OrganisationHousePath("janusbergweg-123", "musterstrasse-12", "/app/settings/valorisation#run-1")
	if !foreign || path != "/app/settings/valorisation#run-1" {
		t.Fatalf("foreign path = %q foreign=%v", path, foreign)
	}
	if strings.Contains(path, "janusbergweg-123") || strings.Contains(path, "musterstrasse-12") {
		t.Fatalf("helper embedded a tenant: %s", path)
	}
	same, foreign := OrganisationHousePath("janusbergweg-123", "janusbergweg-123", "/app/settings/valorisation")
	if foreign || same != "/app/settings/valorisation" {
		t.Fatalf("same-house path = %q foreign=%v", same, foreign)
	}
	rendered := prefixTenantHTMLPaths(`<form action="`+same+`/runs"></form><a href="`+same+`">öffnen</a>`, "janusbergweg-123")
	if strings.Contains(rendered, "musterstrasse-12") || strings.Count(rendered, "janusbergweg-123") != 2 {
		t.Fatalf("prefixed once: %s", rendered)
	}
	again := prefixTenantHTMLPaths(rendered, "janusbergweg-123")
	if again != rendered {
		t.Fatalf("second prefix changed %s", again)
	}
	next := prefixTenantHTMLPaths(`<input name="next" value="`+path+`"/>`, "janusbergweg-123")
	if next != `<input name="next" value="`+path+`"/>` {
		t.Fatalf("return path was prefixed: %s", next)
	}
}

func TestPortalReturnRejectsOpenRedirectsAndAcceptsHousePaths(t *testing.T) {
	for _, path := range []string{
		"https://example.com", "//example.com", "/app/../auth/logout", "/app/context",
		"/app/settings?unsafe=1", "/musterstrasse-12/app/settings/valorisation",
		"/app/settings/annual-statement?year=2025&next=https://example.com",
	} {
		if got := portalContextNext(path); got != "/app" {
			t.Fatalf("unsafe %q => %q", path, got)
		}
	}
	for _, path := range []string{
		"/app/dokumente",
		"/app/settings/valorisation#run-1",
		"/app/settings/building/units/top-1/lease",
		"/app/verwaltung/posteingang/in-0001",
	} {
		if got := portalContextNext(path); got != path {
			t.Fatalf("safe %q => %q", path, got)
		}
	}
	got := portalContextNext("/app/settings/annual-statement?year=2025&run=abc")
	if !strings.Contains(got, "year=2025") || !strings.Contains(got, "run=abc") || strings.Contains(got, "://") {
		t.Fatalf("annual return %q", got)
	}
}

func TestSwitchReturnStaysInsideTheDestinationHouse(t *testing.T) {
	a := newPortalContextTestApp(t)
	token, _, err := a.sessions.PutSession("multi@example.com", "demo", authMethodEmail, roleOwner, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	response := portalContextPost(t, a, token, url.Values{"tenant": {"haus-b"}, "role": {roleRenter}, "next": {"/app/settings/valorisation#run-1"}})
	if response.Code != http.StatusSeeOther || !strings.HasSuffix(response.Header().Get("Location"), "/haus-b/app/settings/valorisation#run-1") {
		t.Fatalf("switch: status %d location %s", response.Code, response.Header().Get("Location"))
	}
	refusedToken, _, err := a.sessions.PutSession("multi@example.com", "demo", authMethodEmail, roleOwner, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	refused := portalContextPost(t, a, refusedToken, url.Values{"tenant": {"haus-b"}, "role": {roleRenter}, "next": {"https://example.com/app"}})
	if refused.Code != http.StatusSeeOther || !strings.HasSuffix(refused.Header().Get("Location"), "/haus-b/app") {
		t.Fatalf("open redirect: status %d location %s", refused.Code, refused.Header().Get("Location"))
	}
}
