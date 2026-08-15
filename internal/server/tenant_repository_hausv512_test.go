package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAuthenticatedHandlerWithoutResolvedTenantFailsClosed(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "resident@example.com",
		Role:        roleResident,
		Tenants:     []string{"demo"},
		AuthMethods: defaultAuthMethods(),
	})
	token, _, err := a.sessions.Put("resident@example.com", "demo", authMethodEmail, time.Hour)
	if err != nil {
		t.Fatalf("put session: %v", err)
	}

	repositoryReached := false
	handler := a.page(func(w http.ResponseWriter, r *http.Request, ac authCtx) {
		repositoryReached = true
		if ac.repositories.announcementReads != nil {
			_, _ = w.Write([]byte("tenant data"))
		}
	})
	req := httptest.NewRequest(http.MethodGet, "http://hausv.org/app", nil)
	req.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	response := httptest.NewRecorder()

	// Calling the route wrapper directly deliberately bypasses tenantPaths.
	handler.ServeHTTP(response, req)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if repositoryReached {
		t.Fatal("handler reached repository without a resolved tenant")
	}
	if strings.Contains(response.Body.String(), "tenant data") {
		t.Fatal("response exposed tenant data")
	}
}

func TestTenantMiddlewareProvidesBoundAnnouncementReadRepository(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "resident@example.com",
		Role:        roleResident,
		Tenants:     []string{"demo"},
		AuthMethods: defaultAuthMethods(),
	})

	resolved := false
	handler := a.tenantPaths(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scope, ok := resolvedTenantFromContext(r.Context())
		resolved = ok && scope.tenant.Slug == "demo" && scope.repositories.announcementReads != nil
		w.WriteHeader(http.StatusNoContent)
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/app", nil))

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if !resolved {
		t.Fatal("tenant middleware did not provide the tenant-bound repository")
	}
}
