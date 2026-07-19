package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// HAUSV-138: the auth invariant, enforced as a test across EVERY /app route
// instead of trusted to 50-plus copy-pasted guards. An unauthenticated request
// to any /app route must be refused (redirect to "/" or 4xx) — never a 200 with
// content. This closes the verification gap the snapshot oracle leaves on the
// POST handlers, and would catch any future handler that forgets its session
// check.
func TestEveryAppRouteRefusesUnauthenticated(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email: "admin@example.com", Role: roleAdmin,
		Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods(),
	})
	handler := a.handler()

	// Concrete path per route; params filled with dummy values (the session
	// check must fire before any param handling).
	type route struct{ method, path string }
	routes := []route{
		{"GET", "/app"},
		{"GET", "/app/announcements"}, {"POST", "/app/announcements"},
		{"POST", "/app/announcements/edit"}, {"POST", "/app/announcements/delete"},
		{"GET", "/app/events"}, {"POST", "/app/events"},
		{"POST", "/app/events/edit"}, {"POST", "/app/events/delete"},
		{"GET", "/app/dokumente"}, {"POST", "/app/dokumente"}, {"POST", "/app/dokumente/replace"},
		{"GET", "/app/dokumente/x/preview"}, {"GET", "/app/dokumente/x/download"},
		{"GET", "/app/attachments/x"}, {"GET", "/app/attachments/x/thumb"}, {"POST", "/app/attachments/delete"},
		{"GET", "/app/abstimmungen"}, {"POST", "/app/abstimmungen"},
		{"POST", "/app/abstimmungen/open"}, {"POST", "/app/abstimmungen/close"},
		{"GET", "/app/abstimmungen/x/protokoll"},
		{"GET", "/app/uebergaben"}, {"POST", "/app/uebergaben"}, {"POST", "/app/uebergaben/file"},
		{"GET", "/app/uebergaben/x/protokoll"},
		{"GET", "/app/kontakte"}, {"POST", "/app/kontakte"}, {"POST", "/app/kontakte/delete"},
		{"GET", "/app/anliegen"}, {"GET", "/app/anliegen/board"},
		{"GET", "/app/anliegen/x/photos/0"}, {"POST", "/app/anliegen"},
		{"POST", "/app/anliegen/comment"}, {"POST", "/app/anliegen/comment/delete"},
		{"POST", "/app/anliegen/workflow"},
		{"GET", "/app/parking"}, {"GET", "/app/parking/settings"},
		{"GET", "/app/parking/month/2026-06"}, {"GET", "/app/parking/export/2026"},
		{"POST", "/app/parking/settings"},
		{"GET", "/app/parking/charging/status"},
		{"POST", "/app/parking/charging/on"}, {"POST", "/app/parking/charging/off"},
		{"POST", "/app/parking/charging/auto"}, {"POST", "/app/parking/charging/settings"},
		{"POST", "/app/parking/charging/telegram/link"}, {"POST", "/app/parking/charging/telegram/unlink"},
		{"GET", "/app/settings"}, {"GET", "/app/settings/profile"},
		{"GET", "/app/settings/notifications"}, {"GET", "/app/settings/building"},
		{"GET", "/app/settings/users"}, {"GET", "/app/audit"},
		{"POST", "/app/settings/users"}, {"POST", "/app/settings/users/edit"},
		{"POST", "/app/settings/users/delete"},
		{"POST", "/app/settings/building/units"}, {"POST", "/app/settings/building/units/delete"},
		{"POST", "/app/settings/parking-access"},
	}

	for _, rt := range routes {
		// No session cookie; but a valid Origin so we isolate the AUTH check
		// from the CSRF check on POSTs.
		req := httptest.NewRequest(rt.method, "http://jhw22.hausv.org"+rt.path, nil)
		if rt.method == http.MethodPost {
			req.Header.Set("Origin", "http://jhw22.hausv.org")
		}
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		body := rr.Body.String()
		leaked := rr.Code == http.StatusOK && strings.Contains(body, "class=\"sidebar\"")
		refused := rr.Code == http.StatusSeeOther || rr.Code == http.StatusFound ||
			(rr.Code >= 400 && rr.Code < 500)
		if leaked || !refused {
			t.Errorf("%s %s: unauthenticated request not refused (status=%d, leaked-shell=%v)",
				rt.method, rt.path, rr.Code, leaked)
		}
	}
}
