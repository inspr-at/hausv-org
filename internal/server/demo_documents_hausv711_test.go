package server

import (
	"net/http"
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/dbtest"
	"github.com/inspr-at/hausv-org/internal/demo"
	"github.com/inspr-at/hausv-org/internal/store"
)

func TestDemoDocumentsPageAndDownloadsRespectRoles(t *testing.T) {
	database, config := dbtest.OpenWithConfig(t)
	scoped, err := db.NewScoped(config, database)
	if err != nil {
		t.Fatal(err)
	}
	defer scoped.Close()
	dir := t.TempDir()
	if _, err := demo.Load(t.Context(), database, "../../scripts/demo/seed", demo.SeedOptions{DocumentDir: dir}); err != nil {
		t.Fatal(err)
	}
	identities, err := store.EnsureTenantIdentities(t.Context(), database, nil)
	if err != nil {
		t.Fatal(err)
	}
	const house = "janusbergweg-123"
	for _, role := range []string{roleOwner, roleRenter, roleResident, roleBeirat, roleManager, roleAdmin} {
		t.Run(role, func(t *testing.T) {
			email := "person@example.com"
			if role == roleOwner {
				email = "alina.eigentuemer@musterstadt.example"
			}
			if role == roleRenter {
				email = "matthias.mieter@musterstadt.example"
			}
			a := newTestPortalApp(t, userProfile{Email: email, Role: role, Tenants: []string{house}, AuthMethods: defaultAuthMethods()})
			a.tenants = map[string]tenantConfig{house: {Slug: house, Name: "Janusbergweg 123"}}
			a.defaultTenant = house
			a.tenantIdentities = identities
			a.documentStore = store.NewSQLDocumentStore(store.NewTenantDB(scoped), dir)
			page := archiveDemoRequest(t, a, email, http.MethodGet, "/app/dokumente", nil)
			if page.Code != http.StatusOK {
				t.Fatalf("documents status=%d", page.Code)
			}
			for _, title := range []string{"Hausordnung", "Protokoll der Eigentümerversammlung 2025", "Wartungsvertrag Lift", "Jahresabrechnung 2025", "Energieausweis"} {
				if !strings.Contains(page.Body.String(), title) {
					t.Errorf("missing %s", title)
				}
			}
			boardAllowed := role == roleBeirat || role == roleManager || role == roleAdmin
			if strings.Contains(page.Body.String(), "Angebot Fassadensanierung") != boardAllowed {
				t.Fatal("board document visibility incorrect")
			}
			if role == roleOwner || role == roleRenter || role == roleResident {
				if !strings.Contains(page.Body.String(), "5 Dokumente") {
					t.Fatal("expected five released documents")
				}
			}
			for _, suffix := range []string{"download", "preview"} {
				for _, id := range []string{"hausordnung", "fassadenangebot"} {
					response := archiveDemoRequest(t, a, email, http.MethodGet, "/app/dokumente/demo-document-"+id+"/"+suffix, nil)
					want := http.StatusOK
					if id == "fassadenangebot" && !boardAllowed {
						want = http.StatusForbidden
					}
					if response.Code != want {
						t.Fatalf("%s/%s status=%d want=%d", id, suffix, response.Code, want)
					}
					if want == http.StatusOK && (!strings.HasPrefix(response.Body.String(), "%PDF-") || response.Header().Get("Content-Type") != "application/pdf") {
						t.Fatal("expected real PDF response")
					}
				}
			}
			if role == roleManager || role == roleAdmin {
				repo, _ := store.BindDocumentRepository(a.documentStore, identities[house].Ref())
				for _, item := range repo.ListCurrent() {
					if !strings.Contains(page.Body.String(), item.Title) {
						t.Errorf("management missing %s", item.Title)
					}
				}
			}
		})
	}
}

func TestBoardDocumentCanBeUploadedThroughExistingReleaseForm(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	response := authedMultipartFileRequest(t, a, "manager@example.com", "/demo/app/dokumente", map[string]string{
		"title": "Beiratsangebot", "category": store.DocumentCategoryContract, "visibility": store.DocumentVisibilityBoardOnly,
	}, "document", "angebot.pdf", []byte("%PDF-1.4\nfixture\n%%EOF\n"))
	if response.Code != http.StatusSeeOther {
		t.Fatalf("upload status=%d", response.Code)
	}
	items := documentRepositoryForTest(a, "demo").ListCurrent()
	if len(items) != 1 || items[0].Visibility != store.DocumentVisibilityBoardOnly {
		t.Fatalf("board release not persisted: %+v", items)
	}
	for _, role := range []string{roleOwner, roleResident, roleRenter, roleBeirat, roleManager, roleAdmin, roleServiceProvider} {
		want := role == roleBeirat || role == roleManager || role == roleAdmin
		if a.canViewDocument(testTenantRef("demo"), items[0], "reader@example.com", role) != want {
			t.Errorf("incorrect board access for %s", role)
		}
	}
	page := authedRequest(t, a, "manager@example.com", "/demo/app/dokumente")
	if !strings.Contains(page.Body.String(), `value="board-only"`) || !strings.Contains(page.Body.String(), "Nur Beirat und Verwaltung") {
		t.Fatal("release form or label missing")
	}
}
