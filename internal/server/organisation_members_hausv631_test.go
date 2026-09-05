package server

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/dbtest"
	"github.com/inspr-at/hausv-org/internal/store"
)

func memberTestApp(t *testing.T, email string) *app {
	t.Helper()
	a := organisationTestApp(t, email)
	database := dbtest.Open(t)
	a.organisationMemberRepo = func(orgKey string) store.OrganisationMemberRepository {
		return store.BindOrganisationMemberRepository(database, orgKey)
	}
	a.syncOrganisations(context.Background())
	return a
}

// addTestPerson gives an e-mail address a profile in one house, the way an
// invite would, so the roster has someone to take on.
func addTestPerson(t *testing.T, a *app, email string, slug string, role string) {
	t.Helper()
	added, err := a.inviteStore.Add(store.UserProfile{
		Email: email, FirstName: "Test", LastName: "Person", Role: role, Tenants: []string{slug},
	})
	if err != nil || !added {
		t.Fatalf("add person %s: added=%v err=%v", email, added, err)
	}
}

func TestOrganisationMemberGetsTheRoleInEveryHouse(t *testing.T) {
	const admin = "verwaltung@example.com"
	a := memberTestApp(t, admin)
	ctx := context.Background()
	addTestPerson(t, a, "paul@example.com", "demo", roleResident)

	ac := authCtx{email: admin, role: roleAdmin, tenant: a.tenants["demo"], tenantRef: testTenantRef("demo")}
	if err := a.addOrganisationMember(ctx, &ac, "musterstadt", "paul@example.com", store.OrganisationRoleClerk); err != nil {
		t.Fatal(err)
	}

	for _, slug := range []string{"demo", "zweitehaus"} {
		if got := normalizeRole(a.roleFor("paul@example.com", slug)); got != roleManager {
			t.Fatalf("role in %s = %q, want %q", slug, got, roleManager)
		}
	}
	member, ok, err := a.organisationMemberRepo("musterstadt").Get(ctx, "paul@example.com")
	if err != nil || !ok {
		t.Fatalf("member row missing: ok=%v err=%v", ok, err)
	}
	if member.Granted["demo"] != roleResident {
		t.Fatalf("grant record for demo = %q, want the previous role %q", member.Granted["demo"], roleResident)
	}
	if member.Granted["zweitehaus"] != "" {
		t.Fatalf("grant record for zweitehaus = %q, want empty (no membership before)", member.Granted["zweitehaus"])
	}
}

func TestOrganisationAdminRoleGrantsTheAdminRolePerHouse(t *testing.T) {
	const admin = "verwaltung@example.com"
	a := memberTestApp(t, admin)
	ctx := context.Background()
	addTestPerson(t, a, "paula@example.com", "demo", roleResident)
	ac := authCtx{email: admin, role: roleAdmin, tenant: a.tenants["demo"], tenantRef: testTenantRef("demo")}
	if err := a.addOrganisationMember(ctx, &ac, "musterstadt", "paula@example.com", store.OrganisationRoleAdmin); err != nil {
		t.Fatal(err)
	}
	if got := normalizeRole(a.roleFor("paula@example.com", "zweitehaus")); got != roleAdmin {
		t.Fatalf("Verwaltungs-Admin got %q in a house, want %q", got, roleAdmin)
	}
}

func TestLeavingRestoresExactlyWhatTheMembershipGranted(t *testing.T) {
	const admin = "verwaltung@example.com"
	a := memberTestApp(t, admin)
	ctx := context.Background()
	addTestPerson(t, a, "paul@example.com", "demo", roleResident)
	ac := authCtx{email: admin, role: roleAdmin, tenant: a.tenants["demo"], tenantRef: testTenantRef("demo")}
	if err := a.addOrganisationMember(ctx, &ac, "musterstadt", "paul@example.com", store.OrganisationRoleClerk); err != nil {
		t.Fatal(err)
	}

	if err := a.removeOrganisationMember(ctx, &ac, "musterstadt", "paul@example.com"); err != nil {
		t.Fatal(err)
	}
	if got := normalizeRole(a.roleFor("paul@example.com", "demo")); got != roleResident {
		t.Fatalf("role in the person's own house = %q, want the previous %q", got, roleResident)
	}
	profile, ok := a.directoryProfile("paul@example.com")
	if !ok {
		t.Fatal("the person lost their profile entirely")
	}
	if profile.HasTenant("zweitehaus") {
		t.Fatal("the house the membership added is still attached after leaving")
	}
	if _, ok, err := a.organisationMemberRepo("musterstadt").Get(ctx, "paul@example.com"); err != nil || ok {
		t.Fatalf("member row survived removal: ok=%v err=%v", ok, err)
	}
}

func TestLeavingLeavesARoleSomeoneChangedByHand(t *testing.T) {
	const admin = "verwaltung@example.com"
	a := memberTestApp(t, admin)
	ctx := context.Background()
	addTestPerson(t, a, "paul@example.com", "demo", roleResident)
	ac := authCtx{email: admin, role: roleAdmin, tenant: a.tenants["demo"], tenantRef: testTenantRef("demo")}
	if err := a.addOrganisationMember(ctx, &ac, "musterstadt", "paul@example.com", store.OrganisationRoleClerk); err != nil {
		t.Fatal(err)
	}

	// After the bundled grant, someone sets one house by hand.
	if _, _, err := a.inviteStore.SetTenantMembership("paul@example.com", "zweitehaus", roleOwner, nil); err != nil {
		t.Fatal(err)
	}
	if err := a.removeOrganisationMember(ctx, &ac, "musterstadt", "paul@example.com"); err != nil {
		t.Fatal(err)
	}
	if got := normalizeRole(a.roleFor("paul@example.com", "zweitehaus")); got != roleOwner {
		t.Fatalf("the hand-set role was withdrawn: %q, want %q", got, roleOwner)
	}
}

func TestRosterActionsRequireAnOrganisationAdmin(t *testing.T) {
	const manager = "sachbearbeiter@example.com"
	a := memberTestApp(t, manager)
	// A manager (not admin) in one house must not change the roster.
	if _, _, err := a.inviteStore.SetTenantMembership(manager, "demo", roleManager, nil); err != nil {
		t.Fatal(err)
	}
	a.profiles[manager] = userProfile{Email: manager, Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}

	got := authedFormRequest(t, a, manager, "/demo/app/verwaltung/einstellungen/mitarbeiter", url.Values{
		"member_email": {"paul@example.com"}, "member_role": {"admin"},
	})
	if got.Code != http.StatusForbidden {
		t.Fatalf("roster action as manager = %d, want 403", got.Code)
	}
}

func TestRosterRefusesAnUnknownPerson(t *testing.T) {
	const admin = "verwaltung@example.com"
	a := memberTestApp(t, admin)
	got := authedFormRequest(t, a, admin, "/demo/app/verwaltung/einstellungen/mitarbeiter", url.Values{
		"member_email": {"niemand@example.com"}, "member_role": {"sachbearbeiter"},
	})
	if got.Code != http.StatusBadRequest {
		t.Fatalf("unknown person = %d, want 400", got.Code)
	}
	if !strings.Contains(got.Body.String(), "noch nicht angelegt") {
		t.Fatalf("unhelpful message: %s", got.Body.String())
	}
}

func TestRosterAppearsOnTheSettingsPage(t *testing.T) {
	const admin = "verwaltung@example.com"
	a := memberTestApp(t, admin)
	ctx := context.Background()
	addTestPerson(t, a, "paul@example.com", "demo", roleResident)
	ac := authCtx{email: admin, role: roleAdmin, tenant: a.tenants["demo"], tenantRef: testTenantRef("demo")}
	if err := a.addOrganisationMember(ctx, &ac, "musterstadt", "paul@example.com", store.OrganisationRoleClerk); err != nil {
		t.Fatal(err)
	}
	database := dbtest.Open(t)
	a.orgSettings = func(orgKey string) store.OrgSettingsRepository {
		return store.BindOrgSettingsRepository(database, orgKey)
	}

	page := authedRequest(t, a, admin, "/demo/app/verwaltung/einstellungen")
	body := page.Body.String()
	if !strings.Contains(body, "Mitarbeiter") || !strings.Contains(body, "paul@example.com") {
		t.Fatal("the settings page does not list the roster")
	}
	if !strings.Contains(body, "Sachbearbeiter") {
		t.Fatal("the roster does not name the organisation role")
	}
}
