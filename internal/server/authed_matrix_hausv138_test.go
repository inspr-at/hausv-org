package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// HAUSV-138: the capability guard for uniform-capability routes now lives in the
// authed()/authedAction() wrappers instead of per-handler preambles. These tests
// lock that an under-privileged role is refused at every hoisted route — a gap
// the existing authmatrix test (unauthenticated only) did not cover.

var hoistedCapabilityRoutes = []struct {
	method string
	path   string
}{
	{"GET", "/app/parking/settings"},
	{"POST", "/app/parking/settings"},
	{"POST", "/app/parking/charging/settings"},
	{"POST", "/app/parking/charging/telegram/link"},
	{"POST", "/app/parking/charging/telegram/unlink"},
	{"POST", "/app/dokumente"},
	{"POST", "/app/dokumente/replace"},
	{"GET", "/app/settings/users"},
	{"POST", "/app/settings/users"},
	{"POST", "/app/settings/users/edit"},
	{"POST", "/app/settings/users/delete"},
}

func TestHoistedRoutesRefuseUnderprivilegedRole(t *testing.T) {
	// A resident has none of manageParking / manageUsers / manageDocuments, so
	// every capability-gated route must refuse them with 403.
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	for _, rt := range hoistedCapabilityRoutes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			var res *httptest.ResponseRecorder
			if rt.method == http.MethodGet {
				res = authedRequest(t, a, "resident@example.com", rt.path)
			} else {
				// same-origin so it passes CSRF and reaches the capability gate.
				res = authedFormRequest(t, a, "resident@example.com", rt.path, url.Values{})
			}
			if res.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403 for an under-privileged role", res.Code)
			}
		})
	}
}

func TestHoistedRoutesAllowCapableRole(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	for _, path := range []string{"/app/settings/users", "/app/parking/settings"} {
		if res := authedRequest(t, a, "admin@example.com", path); res.Code == http.StatusForbidden {
			t.Fatalf("%s: an admin must not be forbidden (got 403)", path)
		}
	}
}

func TestHoistedRoutesAreCapabilitySpecific(t *testing.T) {
	// A manager has manageUsers + manageDocuments but NOT manageParking — the
	// wrapper must be capability-specific, not merely admin/non-admin.
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	if res := authedRequest(t, a, "manager@example.com", "/app/parking/settings"); res.Code != http.StatusForbidden {
		t.Fatalf("manager on parking settings = %d, want 403 (no manageParking)", res.Code)
	}
	if res := authedRequest(t, a, "manager@example.com", "/app/settings/users"); res.Code == http.StatusForbidden {
		t.Fatal("manager must be allowed on user settings (has manageUsers)")
	}
}
