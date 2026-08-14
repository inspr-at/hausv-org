package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// HAUSV-138: the auth invariant, enforced as a test across EVERY /demo/app route
// instead of trusted to 50-plus copy-pasted guards. An unauthenticated request
// to any /demo/app route must be refused (redirect to "/" or 4xx) — never a 200 with
// content. This closes the verification gap the snapshot oracle leaves on the
// POST handlers, and would catch any future handler that forgets its session
// check.
func TestEveryAppRouteRefusesUnauthenticated(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email: "admin@example.com", Role: roleAdmin,
		Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods(),
	})
	handler := a.handler()

	// Concrete path per route; params filled with dummy values (the session
	// check must fire before any param handling).
	type route struct{ method, path string }
	routes := []route{
		{"GET", "/demo/app"},
		{"GET", "/demo/app/hilfe"}, {"POST", "/demo/app/hilfe/connector/pairing"}, {"POST", "/demo/app/hilfe/connector/revoke"},
		{"GET", "/demo/app/announcements"}, {"POST", "/demo/app/announcements"},
		{"POST", "/demo/app/announcements/edit"}, {"POST", "/demo/app/announcements/delete"},
		{"GET", "/demo/app/events"}, {"POST", "/demo/app/events"},
		{"POST", "/demo/app/events/edit"}, {"POST", "/demo/app/events/delete"},
		{"GET", "/demo/app/dokumente"}, {"POST", "/demo/app/dokumente"}, {"POST", "/demo/app/dokumente/replace"},
		{"GET", "/demo/app/dokumente/x/preview"}, {"GET", "/demo/app/dokumente/x/download"},
		{"GET", "/demo/app/attachments/x"}, {"GET", "/demo/app/attachments/x/thumb"}, {"POST", "/demo/app/attachments/delete"},
		{"GET", "/demo/app/abstimmungen"}, {"POST", "/demo/app/abstimmungen"},
		{"POST", "/demo/app/abstimmungen/open"}, {"POST", "/demo/app/abstimmungen/close"},
		{"GET", "/demo/app/abstimmungen/x/protokoll"},
		{"GET", "/demo/app/uebergaben"}, {"POST", "/demo/app/uebergaben"}, {"POST", "/demo/app/uebergaben/attachments"}, {"POST", "/demo/app/uebergaben/file"},
		{"GET", "/demo/app/uebergaben/x/protokoll"},
		{"GET", "/demo/app/kontakte"}, {"POST", "/demo/app/kontakte"}, {"POST", "/demo/app/kontakte/delete"},
		{"GET", "/demo/app/anliegen"}, {"GET", "/demo/app/anliegen/board"},
		{"GET", "/demo/app/anliegen/x/photos/0"}, {"POST", "/demo/app/anliegen"},
		{"POST", "/demo/app/anliegen/comment"}, {"POST", "/demo/app/anliegen/comment/delete"},
		{"POST", "/demo/app/anliegen/workflow"},
		{"GET", "/demo/app/parking"}, {"GET", "/demo/app/parking/settings"},
		{"GET", "/demo/app/parking/month/2026-06"}, {"GET", "/demo/app/parking/export/2026"},
		{"POST", "/demo/app/parking/settings"},
		{"GET", "/demo/app/parking/charging/status"},
		{"POST", "/demo/app/parking/charging/on"}, {"POST", "/demo/app/parking/charging/off"},
		{"POST", "/demo/app/parking/charging/auto"}, {"POST", "/demo/app/parking/charging/settings"},
		{"POST", "/demo/app/parking/charging/telegram/link"}, {"POST", "/demo/app/parking/charging/telegram/unlink"},
		{"GET", "/demo/app/settings"}, {"GET", "/demo/app/settings/profile"},
		{"GET", "/demo/app/settings/notifications"}, {"GET", "/demo/app/settings/building"},
		{"GET", "/demo/app/settings/users"}, {"GET", "/demo/app/audit"},
		{"POST", "/demo/app/settings/users"}, {"POST", "/demo/app/settings/users/edit"},
		{"POST", "/demo/app/settings/users/delete"},
		{"POST", "/demo/app/settings/building/units"}, {"POST", "/demo/app/settings/building/units/delete"},
		{"POST", "/demo/app/settings/parking-access"},
	}

	for _, rt := range routes {
		// No session cookie; but a valid Origin so we isolate the AUTH check
		// from the CSRF check on POSTs.
		req := httptest.NewRequest(rt.method, "http://hausv.org"+rt.path, nil)
		if rt.method == http.MethodPost {
			req.Header.Set("Origin", "http://hausv.org/demo")
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
