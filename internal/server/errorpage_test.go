package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func ownerTestApp(t *testing.T) *app {
	t.Helper()
	return newTestPortalApp(t, userProfile{
		Email:       "owner@example.com",
		Role:        roleOwner,
		Tenants:     []string{"demo"},
		AuthMethods: defaultAuthMethods(),
	})
}

// A refused page must stay a page: same status, the handler's own wording, and
// a way onward. The plain-text body it used to return was the only horizontal
// overflow in the product, because a <pre> does not wrap.
func TestRefusedPageRendersBrandedErrorPage(t *testing.T) {
	response := authedRequest(t, ownerTestApp(t), "owner@example.com", "/demo/app/uebergaben")

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", response.Code)
	}
	if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/html") {
		t.Fatalf("content type = %q, want text/html", contentType)
	}
	body := response.Body.String()
	if strings.Contains(body, "<pre") {
		t.Fatal("error page still renders a <pre>")
	}
	for _, want := range []string{
		// The handler's precise wording survives; a generic message would throw
		// away the only part that tells the reader why.
		"Übergabeprotokolle sind der Verwaltung vorbehalten.",
		"Fehler 403",
		`href="/demo/app"`,
		"Zum Hausüberblick",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("error page missing %q", want)
		}
	}
}

// The onward links exist to help, not to advertise. Naming a locked area would
// tell this owner that handovers exist and are closed to them.
func TestErrorPageOnwardLinksOnlyOfferReachableAreas(t *testing.T) {
	body := authedRequest(t, ownerTestApp(t), "owner@example.com", "/demo/app/uebergaben").Body.String()

	for _, forbidden := range []string{"/demo/app/uebergaben\"", "/demo/app/settings/users", "/demo/app/audit", "/demo/app/parking\""} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("error page offers unreachable area %q", forbidden)
		}
	}
	if !strings.Contains(body, `href="/demo/app/anliegen"`) {
		t.Fatal("error page does not offer Anliegen, which this role can open")
	}
}

// A service-provider account has no Hausüberblick, so the primary action must
// not point at one.
func TestErrorPageSendsServiceProvidersToTheirOwnArea(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "service@example.com",
		Role:        roleServiceProvider,
		Tenants:     []string{"demo"},
		AuthMethods: defaultAuthMethods(),
	})
	a.serviceAccessEnabled = true
	body := authedRequest(t, a, "service@example.com", "/demo/app/dokumente").Body.String()

	if !strings.Contains(body, "Zu den Anliegen") {
		t.Fatalf("service provider error page has no reachable primary action: %q", firstChars(body, 400))
	}
	if strings.Contains(body, "Zum Hausüberblick") {
		t.Fatal("service provider error page offers a Hausüberblick they cannot open")
	}
}

// A mistyped or stale URL is the other half of the defect. It must not fall
// through to the standard library's English plain-text 404.
func TestUnknownURLRendersBrandedNotFoundPage(t *testing.T) {
	a := ownerTestApp(t)
	response := httptest.NewRecorder()
	a.handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/app/gibtesnicht", nil))

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", response.Code)
	}
	body := response.Body.String()
	if strings.Contains(body, "404 page not found") {
		t.Fatal("the standard library's English message reached the user")
	}
	if !strings.Contains(body, "Nicht gefunden") || !strings.Contains(body, "Fehler 404") {
		t.Fatalf("not-found page is not the branded one: %q", firstChars(body, 400))
	}
}

// Signed in or not, the way onward has to actually lead somewhere.
func TestUnknownURLOffersSignInToVisitorsAndNavigationToMembers(t *testing.T) {
	a := ownerTestApp(t)

	anonymous := httptest.NewRecorder()
	a.handler().ServeHTTP(anonymous, httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/app/gibtesnicht", nil))
	if !strings.Contains(anonymous.Body.String(), "Zur Anmeldung") {
		t.Fatal("visitor is not offered the sign-in page")
	}

	member := authedRequest(t, a, "owner@example.com", "/demo/app/gibtesnicht")
	if !strings.Contains(member.Body.String(), "Zum Hausüberblick") {
		t.Fatal("signed-in member is not offered their own navigation")
	}
}

// The seam is one-sided on purpose. Form posts and API-shaped callers keep the
// plain-text contract they have today; only a person navigating to a page gets
// a page.
func TestPlainTextErrorsSurviveWhereTheyBelong(t *testing.T) {
	a := ownerTestApp(t)

	post := authedFormRequest(t, a, "owner@example.com", "/demo/app/announcements", url.Values{"title": {"x"}, "body": {"y"}})
	if post.Code != http.StatusForbidden {
		t.Fatalf("post status = %d, want 403", post.Code)
	}
	if strings.TrimSpace(post.Body.String()) != "Dieser Bereich ist der Verwaltung vorbehalten." {
		t.Fatalf("form post no longer answers in plain text: %q", post.Body.String())
	}

	for _, subresource := range []string{"empty", "image"} {
		token, _, err := a.sessions.Put("owner@example.com", "demo", authMethodEmail, time.Hour)
		if err != nil {
			t.Fatalf("put session: %v", err)
		}
		request := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/app/uebergaben", nil)
		request.Header.Set("Sec-Fetch-Dest", subresource)
		request.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
		response := httptest.NewRecorder()
		a.handler().ServeHTTP(response, request)

		if response.Code != http.StatusForbidden {
			t.Fatalf("%s status = %d, want 403", subresource, response.Code)
		}
		if strings.Contains(response.Body.String(), "<!doctype html>") {
			t.Fatalf("Sec-Fetch-Dest=%s received a page instead of a status", subresource)
		}
	}
}

// A page that succeeds must not pay for any of this — the wrapper has to stay
// transparent for HTML, and for the binary and text formats page routes serve.
func TestErrorPageWrapperLeavesSuccessfulPagesAlone(t *testing.T) {
	a := ownerTestApp(t)
	response := authedRequest(t, a, "owner@example.com", "/demo/app/dokumente")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if !strings.Contains(response.Body.String(), `data-templ-documents`) {
		t.Fatal("the documents page no longer renders itself")
	}
}

func TestErrorPageCopyKeepsHandlerWordingAndDropsBoilerplate(t *testing.T) {
	if _, message, _ := errorPageCopy(http.StatusForbidden, "Dieses Dokument ist für diesen Zugang nicht freigegeben."); message != "Dieses Dokument ist für diesen Zugang nicht freigegeben." {
		t.Fatalf("handler wording was replaced: %q", message)
	}
	if _, message, _ := errorPageCopy(http.StatusNotFound, "404 page not found"); message == "404 page not found" {
		t.Fatal("the standard library's message was passed through")
	}
	if headline, _, _ := errorPageCopy(http.StatusTooManyRequests, ""); headline != "Zu viele Versuche" {
		t.Fatalf("429 headline = %q", headline)
	}
}

func firstChars(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
