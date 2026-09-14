package server

import (
	"context"
	"github.com/inspr-at/hausv-org/internal/demo"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/web"
)

// Full responses, including real permissions and request-bound repositories.
// The organisation scope changes the card, never the navigation structure.
func TestNavigationShellAcrossRoutesAndRolesHAUSV704(t *testing.T) {
	for _, role := range []string{roleAdmin, roleManager, roleOwner, roleResident} {
		t.Run(role, func(t *testing.T) {
			a, _, _ := newInboxTestApp(t, role)
			a.demoReset = func(context.Context, time.Time, io.Writer) (demo.SeedResult, error) { return demo.SeedResult{}, nil }
			routes := []string{"/app", "/app/announcements", "/app/events", "/app/kontakte", "/app/dokumente", "/app/anliegen", "/app/abstimmungen", "/app/settings", "/app/hilfe", "/app/liegenschaften"}
			if role == roleAdmin || role == roleManager {
				routes = append(routes, "/app/anliegen/board", "/app/uebergaben", "/app/audit", "/app/verwaltung", "/app/verwaltung/posteingang")
			}
			if role == roleAdmin {
				routes = append(routes, "/app/parking", "/app/settings/users", "/app/verwaltung/textbausteine", "/app/verwaltung/textbausteine/neu", "/app/verwaltung/rechte", "/app/verwaltung/einstellungen", "/app/verwaltung/einstellungen/demo")
			}
			var baseline []string
			for _, route := range routes {
				t.Run(route, func(t *testing.T) {
					response := authedRequest(t, a, "vera@example.com", "/demo"+route)
					if response.Code != http.StatusOK {
						t.Fatalf("status %d", response.Code)
					}
					body := response.Body.String()
					if strings.Count(body, `data-portal-shell`) != 1 || strings.Count(body, `data-portal-section-header`) != 1 {
						t.Fatal("expected one shared shell and section header")
					}
					if strings.Count(body, "<h1") != 1 {
						t.Fatalf("expected one H1, got %d", strings.Count(body, "<h1"))
					}
					start := strings.Index(body, `<aside class="sidebar"`)
					if start < 0 {
						t.Fatal("shared sidebar absent")
					}
					side := body[start:]
					side = side[:strings.Index(side, "</aside>")]
					blocks := regexp.MustCompile(`data-navigation-block="([^"]+)"`).FindAllStringSubmatch(side, -1)
					var names []string
					for _, block := range blocks {
						names = append(names, block[1])
					}
					want := []string{"map", "house-navigation", "release"}
					if role == roleAdmin || role == roleManager {
						want = append([]string{"organisation-identity", "organisation"}, want...)
					}
					if !reflect.DeepEqual(names, want) {
						t.Fatalf("block order %v, want %v", names, want)
					}
					if baseline == nil {
						baseline = names
					}
					if !reflect.DeepEqual(names, baseline) {
						t.Fatal("sidebar changes between routes")
					}
					// Both surfaces render every block exactly once, with the same classes.
					for _, block := range names {
						matches := regexp.MustCompile(`<[^>]+class="([^"]+)"[^>]+data-navigation-block="`+block+`"`).FindAllStringSubmatch(body, -1)
						if len(matches) != 2 || matches[0][1] != matches[1][1] {
							t.Fatalf("desktop/mobile block %s differs: %v", block, matches)
						}
					}
					if strings.HasPrefix(route, "/app/verwaltung") {
						if !strings.Contains(body, "Alle Liegenschaften") || !strings.Contains(side, "side-map-portfolio") {
							t.Fatal("overview must keep the house card and neutral map")
						}
						for _, path := range []string{"/demo/app", "/demo/app/announcements", "/demo/app/settings"} {
							if !strings.Contains(side, `href="`+path+`"`) {
								t.Fatalf("last selected house link %s absent", path)
							}
						}
					}
					if strings.Contains(side, `class="organisation-identity"`) || strings.Contains(side, "verwaltung-sidebar") {
						t.Fatal("old organisation shell remains")
					}
				})
			}
		})
	}
}

func TestPortalBaseDataPreservesHomeIdentityAcrossBuildersHAUSV704(t *testing.T) {
	a := newHomeIdentityTemplApp(t, "vera@example.com", roleOwner, "einheit-12", []string{"vera@example.com"})
	ac := authCtx{email: "vera@example.com", role: roleOwner, tenant: a.tenants["demo"], tenantRef: testTenantRef("demo"), repositories: testRequestRepositories(t, a, "demo")}
	base := a.portalBaseData(ac, "home", "Hausüberblick")
	if !base.HomeIdentity.HasUnit || base.HomeIdentity.UnitLabel != "Einheit 12" {
		t.Fatal("fixture must contain a real home and assigned unit")
	}
	pages := []web.PortalPageData{a.announcementPortalContext(ac), a.eventsPortalContext(ac), a.contactsPortalContext(ac), a.documentsPortalContext(ac), a.issuesPortalContext(ac), a.ballotsPortalContext(ac), a.handoverPortalContext(ac), a.helpPortalContext(ac), a.auditPortalContext(ac, "Verlauf"), a.parkingPortalContext(ac), a.settingsPortalContext(ac, "Einstellungen", "settings"), a.onboardingPortalContext(ac), a.energyPortalContext(ac, "Mein Zuhause")}
	for _, page := range pages {
		if !reflect.DeepEqual(page.HomeIdentity, base.HomeIdentity) || !reflect.DeepEqual(page.Shell, base.Shell) || !reflect.DeepEqual(page.Modules, base.Modules) {
			t.Fatalf("%s changed the common identity, shell or modules", page.ActivePage)
		}
	}
}

func TestOrganisationResumesLastAuthorisedHouseHAUSV704(t *testing.T) {
	const email = "multi@example.com"
	a := newTestPortalApp(t, userProfile{Email: email, Role: roleAdmin, Tenants: []string{"demo", "haus-b", "privat"}, TenantMemberships: map[string]tenantMembership{"demo": {Role: roleAdmin}, "haus-b": {Role: roleManager}, "privat": {Role: roleOwner}}, AuthMethods: defaultAuthMethods()})
	addTestTenant(a, tenantConfig{Slug: "haus-b", Name: "Haus B", Address: "Nebenweg 2"})
	addTestTenant(a, tenantConfig{Slug: "privat", Name: "Privates Home", PortalType: "home"})
	ac := authCtx{email: email, role: roleOwner, tenant: a.tenants["privat"], tenantRef: testTenantRef("privat"), repositories: testRequestRepositories(t, a, "privat")}
	fallback := a.organisationHouseContext(t.Context(), &ac)
	managed := a.organisationManagedTenants(t.Context(), &ac)
	if len(managed) == 0 || fallback.tenant.Slug != managed[0].Config.Slug {
		t.Fatalf("fallback should use the first managed house, got %s", fallback.tenant.Slug)
	}
	a.recordAudit(auditEvent{TenantSlug: "privat", ActorEmail: email, ActorRole: roleOwner, Action: auditActionContextSwitch, Details: map[string]string{"tenant_from": "haus-b", "role_from": roleManager, "tenant_to": "privat", "role_to": roleOwner}})
	last := a.organisationHouseContext(t.Context(), &ac)
	if last.tenant.Slug != "haus-b" || last.role != roleManager || last.tenantRef.Slug != "haus-b" {
		t.Fatalf("last house context = %s/%s/%s", last.tenant.Slug, last.role, last.tenantRef.Slug)
	}
	page := a.verwaltungShell(t.Context(), &ac, "portfolio")
	if page.NavigationTenant != "haus-b" || page.NavigationRole != roleManager {
		t.Fatal("navigation must resume the chosen context through the switcher")
	}
	// Another person's history and a no-longer-owned target are both ignored.
	a.recordAudit(auditEvent{TenantSlug: "privat", ActorEmail: "other@example.com", Action: auditActionContextSwitch, Details: map[string]string{"tenant_from": "demo", "role_from": roleAdmin, "tenant_to": "privat"}})
	a.recordAudit(auditEvent{TenantSlug: "privat", ActorEmail: email, Action: auditActionContextSwitch, Details: map[string]string{"tenant_from": "haus-b", "role_from": roleAdmin, "tenant_to": "unassigned", "role_to": roleAdmin}})
	last = a.organisationHouseContext(t.Context(), &ac)
	if last.tenant.Slug != "haus-b" || last.role != roleManager {
		t.Fatal("history may only restore the actor's currently assigned role")
	}
}

func TestContextContinuationIsBoundedHAUSV704(t *testing.T) {
	for _, path := range []string{"https://example.com", "//example.com", "/app/../auth/logout", "/app/context", "/app/settings?unsafe=1"} {
		if got := portalContextNext(path); got != "/app" {
			t.Fatalf("unapproved continuation %q => %q", path, got)
		}
	}
	a := newPortalContextTestApp(t)
	token, _, err := a.sessions.PutSession("multi@example.com", "demo", authMethodEmail, roleOwner, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	response := portalContextPost(t, a, token, url.Values{"tenant": {"haus-b"}, "role": {roleRenter}, "next": {"/app/dokumente"}})
	if response.Code != http.StatusSeeOther || !strings.HasSuffix(response.Header().Get("Location"), "/haus-b/app/dokumente") {
		t.Fatalf("continuation: status %d location %s", response.Code, response.Header().Get("Location"))
	}
	session, ok := a.sessions.GetSession(sessionCookieFrom(t, response).Value)
	if !ok || session.TenantSlug != "haus-b" || session.Role != roleRenter {
		t.Fatal("continuation lost the validated context")
	}
}
