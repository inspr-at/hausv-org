package server

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestDocumentsAlwaysUsesTemplRenderer(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})

	body := authedRequest(t, a, "manager@example.com", "/demo/app/dokumente").Body.String()
	if !strings.Contains(body, "data-templ-documents") {
		t.Fatal("documents response must use the templ renderer")
	}
}

func TestDocumentsTemplKeepsDocumentLifecycleReachable(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	documents := documentRepositoryForTest(a, "demo")
	created, err := documents.Create(documentRecord{
		TenantSlug: "demo",
		Title:      "Hausordnung",
		Category:   documentCategoryRules,
		Visibility: documentVisibilityAllResidents,
		UploadedBy: "manager@example.com",
	}, testMultipartHeader(t, "document", "hausordnung-v1.pdf", []byte("%PDF-1.4\nv1\n")), time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("create document: %v", err)
	}
	current, _, err := documents.Replace(created.ID, "manager@example.com", testMultipartHeader(t, "document", "hausordnung-v2.pdf", []byte("%PDF-1.4\nv2\n")), time.Date(2026, 2, 1, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("replace document: %v", err)
	}

	body := authedRequest(t, a, "manager@example.com", "/demo/app/dokumente?q=Hausordnung&sort=oldest").Body.String()
	for _, want := range []string{
		"Hausordnung",
		"hausordnung-v2.pdf",
		"Version 2",
		"Versionsverlauf",
		"Version 1",
		`href="/demo/app/dokumente/` + current.ID + `/preview"`,
		`href="/demo/app/dokumente/` + current.ID + `/download"`,
		`action="/demo/app/dokumente/replace"`,
		`action="/demo/app/dokumente"`,
		`href="/demo/app/dokumente/rechnungen/import"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("templ documents lifecycle should contain %q", want)
		}
	}
}

func TestDocumentsTemplUsesSharedPortalNavigationForEveryResidentRole(t *testing.T) {
	roles := []struct {
		name  string
		email string
		role  string
	}{
		{name: "admin", email: "admin@example.com", role: roleAdmin},
		{name: "manager", email: "manager@example.com", role: roleManager},
		{name: "owner", email: "owner@example.com", role: roleOwner},
		{name: "resident", email: "resident@example.com", role: roleResident},
	}
	for _, test := range roles {
		t.Run(test.name, func(t *testing.T) {
			a := newTestPortalApp(t, userProfile{Email: test.email, Role: test.role, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})

			response := authedRequest(t, a, test.email, "/demo/app/dokumente")
			if response.Code != http.StatusOK {
				t.Fatalf("documents status = %d, want 200", response.Code)
			}
			body := response.Body.String()
			for _, want := range []string{
				"data-templ-documents",
				`href="/demo/app/dokumente" class="nav-item active" aria-current="page"`,
				"Noch keine Dokumente",
				"Was hier abgelegt wird",
			} {
				if !strings.Contains(body, want) {
					t.Fatalf("templ documents for %s should contain %q", test.role, want)
				}
			}
			canManage := test.role == roleAdmin || test.role == roleManager
			if strings.Contains(body, `data-dialog="document-upload"`) != canManage {
				t.Fatalf("upload gate for %s does not match document-management capability", test.role)
			}
		})
	}
}
