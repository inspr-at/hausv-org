package main

import (
	"context"
	"encoding/json"
	"html/template"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestSMTPMailerAllowsInternalRelayWithoutAuth(t *testing.T) {
	m := smtpMailer{
		host: "smtp",
		port: "25",
		from: "WEG Portal <noreply@hausv.org>",
	}

	if !m.Configured() {
		t.Fatal("mailer should be configured with host, port, and from")
	}
	if err := m.Validate(); err != nil {
		t.Fatalf("relay mailer should validate without auth: %v", err)
	}
	if auth := m.auth(); auth != nil {
		t.Fatal("relay mailer without user/pass should not create smtp auth")
	}
}

func TestSMTPMailerRequiresPairedCredentials(t *testing.T) {
	m := smtpMailer{
		host: "smtp",
		port: "25",
		user: "user",
		from: "WEG Portal <noreply@hausv.org>",
	}

	if err := m.Validate(); err == nil {
		t.Fatal("mailer should reject partial smtp credentials")
	}
}

func TestInviteStoreAddDedupeGetList(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invites.json")
	store, err := newInviteStore(path)
	if err != nil {
		t.Fatalf("newInviteStore: %v", err)
	}
	p := userProfile{Email: "New.Person@example.com", FirstName: "New", LastName: "Person", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	added, err := store.Add(p)
	if err != nil || !added {
		t.Fatalf("first Add: added=%v err=%v", added, err)
	}
	// dedupe is case-insensitive and must not overwrite the stored profile
	again, err := store.Add(userProfile{Email: "new.person@example.com", Role: roleAdmin})
	if err != nil {
		t.Fatalf("dedupe Add err: %v", err)
	}
	if again {
		t.Fatal("dedupe Add should return false for an existing email")
	}
	got, ok := store.Get("NEW.PERSON@example.com")
	if !ok || got.LastName != "Person" || got.Role != roleResident {
		t.Fatalf("Get returned %+v ok=%v; dedupe must not have overwritten the role", got, ok)
	}
	if n := len(store.List()); n != 1 {
		t.Fatalf("List len = %d, want 1", n)
	}
	reopened, err := newInviteStore(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if _, ok := reopened.Get("new.person@example.com"); !ok {
		t.Fatal("invite did not persist across reopen")
	}
}

func TestUnitStoreSetListResolvePersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "units.json")
	store, err := newUnitStore(path)
	if err != nil {
		t.Fatalf("newUnitStore: %v", err)
	}
	if err := store.SetTenantUnits("jhw22", []unit{
		{
			ID:                    "Top_2",
			Label:                 "Top 2",
			MiteigentumsanteilPPM: 12345,
			OwnerEmails:           []string{"Owner@Example.com", "owner@example.com"},
			RenterEmails:          []string{"Renter@Example.com"},
		},
		{
			Label:                 "Top 1",
			MiteigentumsanteilPPM: 22222,
			OwnerEmails:           []string{"second-owner@example.com"},
		},
	}); err != nil {
		t.Fatalf("SetTenantUnits: %v", err)
	}

	units := store.ListTenant("JHW22")
	if len(units) != 2 {
		t.Fatalf("ListTenant len = %d, want 2", len(units))
	}
	if units[0].ID != "top-1" || units[0].Label != "Top 1" || units[1].ID != "top-2" {
		t.Fatalf("units not normalized/sorted: %+v", units)
	}
	if got := units[1].OwnerEmails; len(got) != 1 || got[0] != "owner@example.com" {
		t.Fatalf("owners not normalized/deduped: %+v", got)
	}
	if units[1].MiteigentumsanteilPPM != 12345 {
		t.Fatalf("share = %d, want 12345", units[1].MiteigentumsanteilPPM)
	}

	memberships := store.UnitsForEmail("jhw22", "OWNER@example.com")
	if len(memberships) != 1 || memberships[0].Unit.ID != "top-2" || memberships[0].Relation != roleOwner {
		t.Fatalf("owner memberships = %+v", memberships)
	}
	renterMemberships := store.UnitsForEmail("jhw22", "renter@example.com")
	if len(renterMemberships) != 1 || renterMemberships[0].Relation != roleRenter {
		t.Fatalf("renter memberships = %+v", renterMemberships)
	}
	members := store.MembersForUnit("jhw22", "top-2")
	if !members.Found || len(members.Owners) != 1 || members.Owners[0] != "owner@example.com" || len(members.Renters) != 1 || members.Renters[0] != "renter@example.com" {
		t.Fatalf("members = %+v", members)
	}

	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("store file mode = %v err=%v, want 0600", info.Mode().Perm(), err)
	}
	reopened, err := newUnitStore(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if got := reopened.UnitCount("jhw22"); got != 2 {
		t.Fatalf("reopened UnitCount = %d, want 2", got)
	}
}

func TestDirectoryProfileEnvWinsAndInviteGrantsLogin(t *testing.T) {
	store, err := newInviteStore("")
	if err != nil {
		t.Fatalf("newInviteStore: %v", err)
	}
	// an invite trying to claim admin for an email that is an env resident
	if _, err := store.Add(userProfile{Email: "resident@example.com", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	// a brand-new invited-only user
	if _, err := store.Add(userProfile{Email: "invited@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	a := &app{
		defaultTenant: "jhw22",
		profiles: map[string]userProfile{
			"resident@example.com": {Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()},
		},
		inviteStore: store,
	}
	// env is authoritative: the invite must NOT escalate the env resident to admin
	if p, ok := a.directoryProfile("resident@example.com"); !ok || p.Role != roleResident {
		t.Fatalf("env must win: got role %q ok=%v", p.Role, ok)
	}
	// the invited-only user resolves from the store and may log into the tenant
	if p, ok := a.directoryProfile("invited@example.com"); !ok || p.Role != roleResident {
		t.Fatalf("invited user should resolve: got %+v ok=%v", p, ok)
	}
	if !a.isAllowed("invited@example.com", "jhw22") {
		t.Fatal("invited user should be allowed for jhw22")
	}
	// an unknown email is still denied
	if a.isAllowed("stranger@example.com", "jhw22") {
		t.Fatal("stranger must not be allowed")
	}
}

func TestInviteStoreUpdateRekeyAndDelete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invites.json")
	store, err := newInviteStore(path)
	if err != nil {
		t.Fatalf("newInviteStore: %v", err)
	}
	mustAdd := func(email, first string) {
		if _, err := store.Add(userProfile{Email: email, FirstName: first, Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}); err != nil {
			t.Fatalf("Add %s: %v", email, err)
		}
	}
	mustAdd("old@example.com", "Old")
	mustAdd("other@example.com", "Other")

	// updating a non-invite -> false, no error
	if ok, err := store.Update("ghost@example.com", userProfile{Email: "ghost@example.com"}); ok || err != nil {
		t.Fatalf("Update of non-invite: ok=%v err=%v (want false,nil)", ok, err)
	}

	// re-key old -> new + role change (case-insensitive key)
	ok, err := store.Update("old@example.com", userProfile{Email: "New@example.com", FirstName: "New", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	if err != nil || !ok {
		t.Fatalf("Update rekey: ok=%v err=%v", ok, err)
	}
	if _, found := store.Get("old@example.com"); found {
		t.Fatal("old email should be gone after re-key")
	}
	got, found := store.Get("new@example.com")
	if !found || got.Role != roleAdmin || got.FirstName != "New" {
		t.Fatalf("re-keyed entry = %+v found=%v", got, found)
	}

	// re-keying onto an existing invite -> error, entry unchanged
	if ok, err := store.Update("new@example.com", userProfile{Email: "other@example.com"}); ok || err == nil {
		t.Fatalf("Update onto existing email: ok=%v err=%v (want false,err)", ok, err)
	}
	if _, found := store.Get("new@example.com"); !found {
		t.Fatal("entry must survive a rejected re-key")
	}

	// delete
	if ok, err := store.Delete("new@example.com"); !ok || err != nil {
		t.Fatalf("Delete: ok=%v err=%v", ok, err)
	}
	if _, found := store.Get("new@example.com"); found {
		t.Fatal("entry should be gone after delete")
	}
	if ok, _ := store.Delete("new@example.com"); ok {
		t.Fatal("second delete should report false")
	}
	if n := len(store.List()); n != 1 {
		t.Fatalf("List len = %d, want 1 (other@ remains)", n)
	}
}

func TestActivityStoreTouchGetPersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "activity.json")
	store, err := newActivityStore(path)
	if err != nil {
		t.Fatalf("newActivityStore: %v", err)
	}
	if _, ok := store.Get("nobody@example.com"); ok {
		t.Fatal("empty store should have no records")
	}
	when := time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC)
	if err := store.Touch("Person@Example.com", when, authMethodEmail); err != nil {
		t.Fatalf("Touch: %v", err)
	}
	rec, ok := store.Get("person@example.com") // case-insensitive key
	if !ok || !rec.LastLogin.Equal(when) {
		t.Fatalf("Get = %+v ok=%v", rec, ok)
	}
	reopened, err := newActivityStore(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if _, ok := reopened.Get("person@example.com"); !ok {
		t.Fatal("activity did not persist across reopen")
	}
}

func TestUserRowsDeriveStatusFromActivity(t *testing.T) {
	act, _ := newActivityStore("")
	_ = act.Touch("loggedin@example.com", time.Date(2026, 7, 6, 9, 0, 0, 0, time.UTC), authMethodOIDC)
	inv, _ := newInviteStore("")
	_, _ = inv.Add(userProfile{Email: "loggedin@example.com", Role: roleResident, Status: "Eingeladen", Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	_, _ = inv.Add(userProfile{Email: "never@example.com", Role: roleResident, Status: "Eingeladen", Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	a := &app{defaultTenant: "jhw22", profiles: map[string]userProfile{}, inviteStore: inv, activityStore: act}

	byEmail := map[string]userRow{}
	for _, r := range a.userRows("jhw22") {
		byEmail[r.Email] = r
	}
	if got := byEmail["loggedin@example.com"]; got.Status != "Aktiv" || !strings.Contains(got.LastSeen, "zuletzt angemeldet") {
		t.Fatalf("logged-in invite: status=%q lastseen=%q (want Aktiv + zuletzt)", got.Status, got.LastSeen)
	}
	if got := byEmail["never@example.com"]; got.Status != "Eingeladen" || got.LastSeen != "noch nie angemeldet" {
		t.Fatalf("never-logged-in invite: status=%q lastseen=%q", got.Status, got.LastSeen)
	}
}

func TestRunHealthcheckAcceptsExpectedPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"service":"weg-portal","status":"ok"}`))
	}))
	defer server.Close()

	if err := runHealthcheck(server.URL); err != nil {
		t.Fatalf("healthcheck should accept expected payload: %v", err)
	}
}

func TestRunHealthcheckRejectsWrongPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"service":"other","status":"ok"}`))
	}))
	defer server.Close()

	if err := runHealthcheck(server.URL); err == nil {
		t.Fatal("healthcheck should reject unexpected payload")
	}
}

func TestBuildLabelUsesSemverAndCommit(t *testing.T) {
	origVersion := appVersion
	origCommit := gitCommit
	t.Cleanup(func() {
		appVersion = origVersion
		gitCommit = origCommit
	})

	appVersion = "v0.1.0"
	gitCommit = "abc1234"

	if got := buildLabel(); got != "0.1.0 (abc1234)" {
		t.Fatalf("build label = %q, want semver and commit", got)
	}
}

func TestHomeUsesUnitCountFromStore(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	if err := a.unitStore.SetTenantUnits("jhw22", []unit{
		{ID: "top-1", Label: "Top 1", OwnerEmails: []string{"owner1@example.com"}},
		{ID: "top-2", Label: "Top 2", OwnerEmails: []string{"owner2@example.com"}},
	}); err != nil {
		t.Fatalf("SetTenantUnits: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://jhw22.hausv.org/", nil)
	rr := httptest.NewRecorder()
	a.home(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("home status = %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "<strong>2</strong>") || !strings.Contains(body, "Wohneinheiten im Haus") {
		t.Fatalf("home should render real unit count, body: %s", body)
	}
	if strings.Contains(body, "12 Wohneinheiten") {
		t.Fatal("home must not render the old hardcoded unit count")
	}
}

func TestSessionSecretRequiredForPublicBaseURL(t *testing.T) {
	t.Setenv("SESSION_KEY", "")

	if _, err := sessionSecret(true); err == nil {
		t.Fatal("public deployment should require SESSION_KEY")
	}
}

func TestSessionSecretCanBeEphemeralForLocalDev(t *testing.T) {
	t.Setenv("SESSION_KEY", "")

	secret, err := sessionSecret(false)
	if err != nil {
		t.Fatalf("local development should allow generated session secret: %v", err)
	}
	if len(secret) != 32 {
		t.Fatalf("generated session secret length = %d, want 32", len(secret))
	}
}

func TestSignedSessionRoundTripSurvivesNewStore(t *testing.T) {
	secret := []byte(strings.Repeat("s", 32))
	first := newSessionStore(secret)

	token, _, err := first.Put("Markus@Barta.com", "JHW22", authMethodOIDC, time.Hour)
	if err != nil {
		t.Fatalf("put session: %v", err)
	}

	second := newSessionStore(secret)
	email, tenantSlug, authMethod, ok := second.Get(token)
	if !ok {
		t.Fatal("session should verify in a new store with the same secret")
	}
	if email != "markus@barta.com" || tenantSlug != "jhw22" || authMethod != authMethodOIDC {
		t.Fatalf("unexpected session claims: %s %s %s", email, tenantSlug, authMethod)
	}

	other := newSessionStore([]byte(strings.Repeat("x", 32)))
	if _, _, _, ok := other.Get(token); ok {
		t.Fatal("session should not verify with a different secret")
	}
}

func TestParseUserProfilesNormalizesAuthMethods(t *testing.T) {
	raw := `[{"email":"joerg.lehner@gmx.at","first_name":"Jörg","last_name":"Lehner","tenants":["jhw22"],"auth_methods":["zitadel"]}]`

	profiles, err := parseUserProfiles(raw, map[string]struct{}{}, map[string]struct{}{}, "jhw22")
	if err != nil {
		t.Fatalf("parse profiles: %v", err)
	}
	profile := profiles["joerg.lehner@gmx.at"]
	if !profile.AllowsAuthMethod(authMethodOIDC) {
		t.Fatal("profile should allow OIDC")
	}
	if profile.AllowsAuthMethod(authMethodEmail) {
		t.Fatal("profile should not allow email login")
	}
	if got := profile.UserRow().AuthLabel; got != "Zitadel SSO" {
		t.Fatalf("auth label = %q", got)
	}
}

func TestOIDCLoginDefersUnavailableDiscovery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	login, err := newOIDCLogin(ctx, "https://auth.invalid.example", "client-id", "", "", "Zitadel")
	if err != nil {
		t.Fatalf("new OIDC login should not fail hard when discovery is unavailable: %v", err)
	}
	if !login.Configured() {
		t.Fatal("OIDC should remain configured so discovery can be retried later")
	}
	if login.provider != nil || login.verifier != nil {
		t.Fatal("provider should not be initialized after canceled discovery")
	}
	if err := login.EnsureProvider(ctx); err == nil {
		t.Fatal("retry with canceled context should still report discovery failure")
	}
}

func TestNormalizeRoleAliasesAndCapabilityMatrix(t *testing.T) {
	aliases := map[string]string{
		"admin":            roleAdmin,
		"property-manager": roleManager,
		"Hausverwaltung":   roleManager,
		"Eigentuemer":      roleOwner,
		"eigentümer":       roleOwner,
		"owner":            roleOwner,
		"tenant":           roleRenter,
		"Mieter":           roleRenter,
		"advisory-board":   roleBeirat,
		"Beirat":           roleBeirat,
		"resident":         roleResident,
		"Bewohner":         roleResident,
	}
	for raw, want := range aliases {
		if got := normalizeRole(raw); got != want {
			t.Fatalf("normalizeRole(%q) = %q, want %q", raw, got, want)
		}
	}

	cases := []struct {
		role string
		cap  capability
		want bool
	}{
		{roleAdmin, capabilityManageUsers, true},
		{roleAdmin, capabilityManageParking, true},
		{roleManager, capabilityManageAnnouncements, true},
		{roleManager, capabilityManageUsers, false},
		{roleOwner, capabilityVote, true},
		{roleOwner, capabilityOwnerDocuments, true},
		{roleRenter, capabilityVote, false},
		{roleBeirat, capabilityOversight, true},
		{roleBeirat, capabilityManageAnnouncements, false},
		{roleResident, capabilityOversight, false},
	}
	for _, tc := range cases {
		if got := hasCapability(tc.role, tc.cap); got != tc.want {
			t.Fatalf("hasCapability(%q, %q) = %v, want %v", tc.role, tc.cap, got, tc.want)
		}
	}
}

func TestSettingsHubVisibleToResidentWithoutAdminSections(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})

	rr := authedRequest(t, a, "resident@example.com", "/app/settings", a.settingsHub)
	if rr.Code != http.StatusOK {
		t.Fatalf("settings hub status = %d", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{`href="/app/settings"`, "Profil", "Benachrichtigungen"} {
		if !strings.Contains(body, want) {
			t.Fatalf("settings hub should contain %q", want)
		}
	}
	if strings.Contains(body, `href="/app/parking/settings"`) {
		t.Fatal("resident settings hub must not expose parking settings")
	}
}

func TestSettingsHubAdminLinksManagementSections(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})

	rr := authedRequest(t, a, "admin@example.com", "/app/settings", a.settingsHub)
	if rr.Code != http.StatusOK {
		t.Fatalf("settings hub status = %d", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{`href="/app/settings/users"`, `href="/app/parking/settings"`, "Parkplatz-Abrechnung", "Gebäude"} {
		if !strings.Contains(body, want) {
			t.Fatalf("admin settings hub should contain %q", want)
		}
	}
}

func TestRoleManagementUIOffersAllEffectiveRoles(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	for _, profile := range []userProfile{
		{Email: "owner@example.com", FirstName: "Eva", LastName: "Owner", Role: roleOwner, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()},
		{Email: "renter@example.com", FirstName: "Max", LastName: "Renter", Role: roleRenter, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()},
		{Email: "manager@example.com", FirstName: "Mara", LastName: "Manager", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()},
		{Email: "board@example.com", FirstName: "Berta", LastName: "Beirat", Role: roleBeirat, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()},
	} {
		a.profiles[profile.Email] = profile
	}

	rr := authedRequest(t, a, "admin@example.com", "/app/settings/users", a.userSettings)
	if rr.Code != http.StatusOK {
		t.Fatalf("user settings status = %d", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{
		`value="Mieter"`, `value="Eigentümer"`, `value="Beirat"`, `value="Verwalter"`, `value="Admin"`,
		"role-owner", "role-renter", "role-manager", "role-beirat",
		"Eigentümer-Dokumente", "Abstimmungen", "Aushang verwalten", "Leserechte",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("role management UI should contain %q", want)
		}
	}
}

func TestManagerCanManageAnnouncementsButNotPlatformSettings(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})

	users := authedRequest(t, a, "manager@example.com", "/app/settings/users", a.userSettings)
	if users.Code != http.StatusForbidden {
		t.Fatalf("manager user settings status = %d, want 403", users.Code)
	}
	values := url.Values{
		"title": {"Manager post"},
		"body":  {"Allowed"},
	}
	write := authedFormRequest(t, a, "manager@example.com", "/app/announcements", values, a.createAnnouncement)
	if write.Code != http.StatusSeeOther {
		t.Fatalf("manager announcement create status = %d, want redirect", write.Code)
	}
}

func TestParkingSettingsIsParkingSpecificNotGlobalSettings(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})

	rr := authedRequest(t, a, "admin@example.com", "/app/parking/settings", a.parkingSettings)
	if rr.Code != http.StatusOK {
		t.Fatalf("parking settings status = %d", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{"Parkplatz-Abrechnung", `href="/app/settings"`, "Zurück zu Einstellungen"} {
		if !strings.Contains(body, want) {
			t.Fatalf("parking settings should contain %q", want)
		}
	}
	if strings.Contains(body, "<h1>Einstellungen</h1>") {
		t.Fatal("parking settings page must not use generic Einstellungen heading")
	}
}

func TestAnnouncementStoreCRUDVisibleSortPersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "announcements.json")
	store, err := newAnnouncementStore(path)
	if err != nil {
		t.Fatalf("newAnnouncementStore: %v", err)
	}
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	expiredAt := now.Add(-time.Hour)
	future, err := store.Create(announcement{TenantSlug: "jhw22", Title: "Future", Body: "Later", Category: "Info", PublishedAt: now.Add(time.Hour)})
	if err != nil || future.ID == "" {
		t.Fatalf("create future: item=%+v err=%v", future, err)
	}
	expired, _ := store.Create(announcement{TenantSlug: "jhw22", Title: "Expired", Body: "Old", Category: "Info", PublishedAt: now.Add(-2 * time.Hour), ExpiresAt: &expiredAt})
	normal, _ := store.Create(announcement{TenantSlug: "jhw22", Title: "Normal", Body: "Visible", Category: "Termin", PublishedAt: now.Add(-30 * time.Minute)})
	pinned, _ := store.Create(announcement{TenantSlug: "jhw22", Title: "Pinned", Body: "Top", Category: "Dringend", Pinned: true, PublishedAt: now.Add(-2 * time.Hour)})
	_, _ = store.Create(announcement{TenantSlug: "other", Title: "Other", Body: "Hidden", Category: "Info", PublishedAt: now.Add(-time.Hour)})

	visible := store.Visible("jhw22", now)
	if len(visible) != 2 {
		t.Fatalf("visible len = %d, want 2 (future=%s expired=%s normal=%s pinned=%s)", len(visible), future.ID, expired.ID, normal.ID, pinned.ID)
	}
	if visible[0].Title != "Pinned" || visible[1].Title != "Normal" {
		t.Fatalf("visible order = %q, %q; want pinned first then recent", visible[0].Title, visible[1].Title)
	}

	updated := normal
	updated.Title = "Updated"
	if ok, err := store.Update(normal.ID, updated); !ok || err != nil {
		t.Fatalf("update: ok=%v err=%v", ok, err)
	}
	if removed, err := store.Delete("jhw22", pinned.ID); !removed || err != nil {
		t.Fatalf("delete: removed=%v err=%v", removed, err)
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("store file mode = %v err=%v, want 0600", info.Mode().Perm(), err)
	}
	reopened, err := newAnnouncementStore(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	all := reopened.ListTenant("jhw22")
	if len(all) != 3 {
		t.Fatalf("reopened list len = %d, want 3 after delete", len(all))
	}
	foundUpdated := false
	for _, item := range all {
		if item.ID == normal.ID && item.Title == "Updated" {
			foundUpdated = true
		}
	}
	if !foundUpdated {
		t.Fatal("updated announcement did not persist")
	}
}

func TestAnnouncementReadStorePersistsSeenState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "announcement_reads.json")
	store, err := newAnnouncementReadStore(path)
	if err != nil {
		t.Fatalf("newAnnouncementReadStore: %v", err)
	}
	seenAt := time.Date(2026, 7, 6, 12, 30, 0, 0, time.UTC)
	if err := store.MarkSeen("jhw22", "Resident@Example.com", seenAt); err != nil {
		t.Fatalf("mark seen: %v", err)
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("read store file mode = %v err=%v, want 0600", info.Mode().Perm(), err)
	}
	reopened, err := newAnnouncementReadStore(path)
	if err != nil {
		t.Fatalf("reopen read store: %v", err)
	}
	if got := reopened.LastSeen("jhw22", "resident@example.com"); !got.Equal(seenAt) {
		t.Fatalf("last seen = %v, want %v", got, seenAt)
	}
}

func TestPortalUsesAnnouncementEmptyStateWithoutPrototypeCopy(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})

	rr := authedRequest(t, a, "resident@example.com", "/app", a.portal)
	if rr.Code != http.StatusOK {
		t.Fatalf("portal status = %d", rr.Code)
	}
	body := rr.Body.String()
	for _, forbidden := range []string{"Willkommen im Prototyp", "Beispielmodule", "Nächste Ausbaustufe"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("portal must not contain placeholder copy %q", forbidden)
		}
	}
	if !strings.Contains(body, "Noch keine Beiträge") || !strings.Contains(body, `href="/app/announcements"`) {
		t.Fatal("portal should show announcement empty state and real archive link")
	}
}

func TestAnnouncementArchiveFiltersSearchesAndIncludesPast(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	now := time.Now().Add(-2 * time.Hour)
	expiredAt := time.Now().Add(-time.Hour)
	_, _ = a.announcementStore.Create(announcement{TenantSlug: "jhw22", Title: "Liftwartung", Body: "Lift Freitag", Category: "Wartung", PublishedAt: now})
	_, _ = a.announcementStore.Create(announcement{TenantSlug: "jhw22", Title: "Hausfest", Body: "Sommertermin", Category: "Termin", PublishedAt: now})
	_, _ = a.announcementStore.Create(announcement{TenantSlug: "jhw22", Title: "Alter Hinweis", Body: "Vergangen", Category: "Info", PublishedAt: now.Add(-time.Hour), ExpiresAt: &expiredAt})
	_, _ = a.announcementStore.Create(announcement{TenantSlug: "jhw22", Title: "Geplant", Body: "Noch nicht sichtbar", Category: "Info", PublishedAt: time.Now().Add(time.Hour)})

	all := authedRequest(t, a, "resident@example.com", "/app/announcements", a.announcements)
	if all.Code != http.StatusOK {
		t.Fatalf("archive status = %d", all.Code)
	}
	body := all.Body.String()
	for _, want := range []string{"Liftwartung", "Hausfest", "Alter Hinweis", "Abgelaufen", "Suche", "Wartung"} {
		if !strings.Contains(body, want) {
			t.Fatalf("archive should contain %q", want)
		}
	}
	if strings.Contains(body, "Geplant") {
		t.Fatal("archive must not expose future announcements to residents")
	}

	filtered := authedRequest(t, a, "resident@example.com", "/app/announcements?category=Wartung&q=Lift", a.announcements)
	if filtered.Code != http.StatusOK {
		t.Fatalf("filtered archive status = %d", filtered.Code)
	}
	body = filtered.Body.String()
	if !strings.Contains(body, "Liftwartung") || strings.Contains(body, "Hausfest") || strings.Contains(body, "Alter Hinweis") {
		t.Fatalf("filtered archive body did not match expected search/category result:\n%s", body)
	}
}

func TestAnnouncementUnreadBadgeClearsAfterArchiveView(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	_, _ = a.announcementStore.Create(announcement{TenantSlug: "jhw22", Title: "Neue Wartung", Body: "Heute", Category: "Wartung", PublishedAt: time.Now().Add(-time.Hour)})

	before := authedRequest(t, a, "resident@example.com", "/app", a.portal)
	if before.Code != http.StatusOK {
		t.Fatalf("portal before status = %d", before.Code)
	}
	if body := before.Body.String(); !strings.Contains(body, "Neue Wartung") || !strings.Contains(body, `pill unread">neu`) || !strings.Contains(body, `<span class="nav-badge">1</span>`) {
		t.Fatalf("portal should show unread announcement and nav badge before archive view:\n%s", body)
	}

	archive := authedRequest(t, a, "resident@example.com", "/app/announcements", a.announcements)
	if archive.Code != http.StatusOK {
		t.Fatalf("archive status = %d", archive.Code)
	}
	if got := a.announcementReadStore.LastSeen("jhw22", "resident@example.com"); got.IsZero() {
		t.Fatal("archive view should mark announcements as seen")
	}

	after := authedRequest(t, a, "resident@example.com", "/app", a.portal)
	if after.Code != http.StatusOK {
		t.Fatalf("portal after status = %d", after.Code)
	}
	body := after.Body.String()
	if strings.Contains(body, `<span class="nav-badge">`) || strings.Contains(body, `pill unread">neu`) {
		t.Fatalf("portal should clear unread badges after archive view:\n%s", body)
	}
}

func TestAnnouncementRoutesGateWritesAndAllowResidentRead(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})

	read := authedRequest(t, a, "resident@example.com", "/app/announcements", a.announcements)
	if read.Code != http.StatusOK {
		t.Fatalf("resident announcement read status = %d", read.Code)
	}
	write := authedFormRequest(t, a, "resident@example.com", "/app/announcements", url.Values{
		"title": {"Resident post"},
		"body":  {"Nope"},
	}, a.createAnnouncement)
	if write.Code != http.StatusForbidden {
		t.Fatalf("resident create status = %d, want 403", write.Code)
	}
}

func TestAnnouncementCreateRejectsCrossOriginAndPersistsSameOrigin(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	values := url.Values{
		"title":        {"Liftwartung"},
		"body":         {"Der Lift ist am Freitag vormittags außer Betrieb."},
		"category":     {"Wartung"},
		"published_at": {"2026-07-06T12:30"},
		"pinned":       {"true"},
	}

	cross := authedFormRequestWithOrigin(t, a, "admin@example.com", "/app/announcements", values, "https://evil.example", a.createAnnouncement)
	if cross.Code != http.StatusForbidden {
		t.Fatalf("cross-origin create status = %d, want 403", cross.Code)
	}

	same := authedFormRequest(t, a, "admin@example.com", "/app/announcements", values, a.createAnnouncement)
	if same.Code != http.StatusSeeOther {
		t.Fatalf("same-origin create status = %d, want redirect", same.Code)
	}
	visible := a.announcementStore.Visible("jhw22", time.Date(2026, 7, 6, 13, 0, 0, 0, time.Local))
	if len(visible) != 1 || visible[0].Title != "Liftwartung" || !visible[0].Pinned {
		t.Fatalf("created visible announcement = %+v", visible)
	}
}

func newTestPortalApp(t *testing.T, profile userProfile) *app {
	t.Helper()
	tmpl, err := template.New("pages").Parse(pageTemplates)
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}
	profile.Email = normalizeEmail(profile.Email)
	profile.Tenants = normalizeTenants(profile.Tenants, "jhw22")
	authMethods, err := normalizeAuthMethods(profile.AuthMethods)
	if err != nil {
		t.Fatalf("normalize auth methods: %v", err)
	}
	profile.AuthMethods = authMethods
	parkingStore, err := newParkingStore("")
	if err != nil {
		t.Fatalf("parking store: %v", err)
	}
	announcementStore, err := newAnnouncementStore("")
	if err != nil {
		t.Fatalf("announcement store: %v", err)
	}
	announcementReadStore, err := newAnnouncementReadStore("")
	if err != nil {
		t.Fatalf("announcement read store: %v", err)
	}
	unitStore, err := newUnitStore("")
	if err != nil {
		t.Fatalf("unit store: %v", err)
	}
	return &app{
		baseURL:       "http://localhost:8080",
		rootDomain:    "hausv.org",
		defaultTenant: "jhw22",
		tenants: map[string]tenantConfig{
			"jhw22": {Slug: "jhw22", Name: "WEG Portal", Address: "Janischhofweg 22", Host: "jhw22.hausv.org"},
		},
		profiles: map[string]userProfile{
			profile.Email: profile,
		},
		allowed:               map[string]struct{}{},
		admins:                map[string]struct{}{},
		sessionTTL:            time.Hour,
		sessions:              newSessionStore([]byte(strings.Repeat("s", 32))),
		oidc:                  &oidcLogin{},
		mailer:                smtpMailer{},
		templates:             tmpl,
		announcementStore:     announcementStore,
		announcementReadStore: announcementReadStore,
		unitStore:             unitStore,
		parkingStore:          parkingStore,
	}
}

func authedRequest(t *testing.T, a *app, email string, path string, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	token, _, err := a.sessions.Put(email, "jhw22", authMethodEmail, time.Hour)
	if err != nil {
		t.Fatalf("put session: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://jhw22.hausv.org"+path, nil)
	req.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	rr := httptest.NewRecorder()
	handler(rr, req)
	return rr
}

func authedFormRequest(t *testing.T, a *app, email string, path string, values url.Values, handler http.HandlerFunc) *httptest.ResponseRecorder {
	return authedFormRequestWithOrigin(t, a, email, path, values, "http://jhw22.hausv.org", handler)
}

func authedFormRequestWithOrigin(t *testing.T, a *app, email string, path string, values url.Values, origin string, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	token, _, err := a.sessions.Put(email, "jhw22", authMethodEmail, time.Hour)
	if err != nil {
		t.Fatalf("put session: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "http://jhw22.hausv.org"+path, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", origin)
	req.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	rr := httptest.NewRecorder()
	handler(rr, req)
	return rr
}

func TestParkingHourlyUsageAppliesHourlyAwattarPrices(t *testing.T) {
	base := time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)
	energy := []parkingNumericSample{
		{At: base, Value: 100},
		{At: base.Add(2 * time.Hour), Value: 102},
	}
	prices := []parkingNumericSample{
		{At: base, Value: 0.20},
		{At: base.Add(time.Hour), Value: 0.40},
	}

	hours := calculateParkingHourlyUsage(energy, prices, 0.10, base.Add(3*time.Hour))
	if len(hours) != 2 {
		t.Fatalf("hour buckets = %d, want 2", len(hours))
	}
	assertClose(t, hours[0].KWh, 1)
	assertClose(t, hours[0].EnergyCost, 0.20)
	assertClose(t, hours[0].GridCost, 0.10)
	assertClose(t, hours[1].KWh, 1)
	assertClose(t, hours[1].EnergyCost, 0.40)
	assertClose(t, hours[1].GridCost, 0.10)
}

func TestParkingMonthsExposeCostsAndPaidFlag(t *testing.T) {
	base := time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)
	data := parkingTenantData{
		Settings: parkingSettings{GridFeeEURPerKWh: 0.10},
		Months: map[string]parkingMonthState{
			"2026-06": {Paid: true},
		},
		EnergySamples: []parkingNumericSample{
			{At: base, Value: 100},
			{At: base.Add(2 * time.Hour), Value: 102},
		},
		PriceSamples: []parkingNumericSample{
			{At: base, Value: 0.20},
			{At: base.Add(time.Hour), Value: 0.40},
		},
	}

	months := calculateParkingMonths(data, base.Add(3*time.Hour), time.UTC)
	if len(months) != 1 {
		t.Fatalf("months = %d, want 1", len(months))
	}
	month := months[0]
	if month.Month != "2026-06" || !month.Paid {
		t.Fatalf("month state = %+v, want paid 2026-06", month)
	}
	if month.KWh != "2,00 kWh" || month.EnergyCost != "0,60 €" || month.GridCost != "0,20 €" || month.TotalCost != "0,80 €" {
		t.Fatalf("unexpected formatted costs: %+v", month)
	}
	if month.AverageAwattar != "0,300 €/kWh" || month.EffectivePrice != "0,400 €/kWh" {
		t.Fatalf("unexpected average prices: %+v", month)
	}
	if month.DetailPath != "/app/parking/month/2026-06" {
		t.Fatalf("detail path = %q", month.DetailPath)
	}
	if month.HourCount != 2 {
		t.Fatalf("hour count = %d, want 2", month.HourCount)
	}
}

func TestParkingMonthDetailsExposeHourlyRows(t *testing.T) {
	base := time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)
	data := parkingTenantData{
		Settings: parkingSettings{GridFeeEURPerKWh: 0.10},
		EnergySamples: []parkingNumericSample{
			{At: base, Value: 100},
			{At: base.Add(2 * time.Hour), Value: 102},
		},
		PriceSamples: []parkingNumericSample{
			{At: base, Value: 0.20},
			{At: base.Add(time.Hour), Value: 0.40},
		},
	}

	detail := calculateParkingMonthDetails(data, "2026-06", base.Add(3*time.Hour), time.UTC)
	if !detail.HasHours || len(detail.Hours) != 2 {
		t.Fatalf("detail hours = %d, has=%v; want 2 true", len(detail.Hours), detail.HasHours)
	}
	if detail.Summary.TotalCost != "0,80 €" || detail.Summary.AverageAwattar != "0,300 €/kWh" {
		t.Fatalf("unexpected detail summary: %+v", detail.Summary)
	}
	if detail.Hours[0].AtLabel != "25.06. 10:00" || detail.Hours[0].TotalCost != "0,30 €" {
		t.Fatalf("unexpected first hour: %+v", detail.Hours[0])
	}
	if detail.Hours[0].KWhTitle != "Verbrauch: 1,000000 kWh" {
		t.Fatalf("unexpected kWh title: %q", detail.Hours[0].KWhTitle)
	}
	if detail.Hours[0].EnergyCostTitle != "Stromkosten: 0,200000 € = 1,000000 kWh × 0,200000 €/kWh" {
		t.Fatalf("unexpected energy title: %q", detail.Hours[0].EnergyCostTitle)
	}
	if detail.Hours[0].TotalCostTitle != "Summe: 0,300000 € = Strom 0,200000 € + Netzgebühr 0,100000 €" {
		t.Fatalf("unexpected total title: %q", detail.Hours[0].TotalCostTitle)
	}
	if detail.Hours[1].AtLabel != "25.06. 11:00" || detail.Hours[1].TotalCost != "0,50 €" {
		t.Fatalf("unexpected second hour: %+v", detail.Hours[1])
	}
}

func TestParkingStorePersistsPaidFlagAndGridFee(t *testing.T) {
	path := filepath.Join(t.TempDir(), "parking.json")
	store, err := newParkingStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	if err := store.SetGridFee("jhw22", 0.123); err != nil {
		t.Fatalf("set grid fee: %v", err)
	}
	if err := store.SetMonthPaid("jhw22", "2026-06", true); err != nil {
		t.Fatalf("set paid flag: %v", err)
	}

	loaded, err := newParkingStore(path)
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}
	data := loaded.TenantData("jhw22")
	assertClose(t, data.Settings.GridFeeEURPerKWh, 0.123)
	if !data.Months["2026-06"].Paid {
		t.Fatal("paid flag was not persisted")
	}
}

func TestSamplesFromStatisticsUsesMillisecondsAndPreferredFields(t *testing.T) {
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	state := 123.45
	sum := 456.78
	mean := 0.31
	stats := []haStatistic{
		{Start: json.RawMessage(strconvFormatInt(start.UnixMilli())), State: &state, Sum: &sum, Mean: &mean},
	}

	energy := samplesFromStatistics(stats, "state", "sum")
	if len(energy) != 1 {
		t.Fatalf("energy samples = %d, want 1", len(energy))
	}
	if !energy[0].At.Equal(start) {
		t.Fatalf("sample time = %s, want %s", energy[0].At, start)
	}
	assertClose(t, energy[0].Value, state)

	price := samplesFromStatistics(stats, "mean")
	if len(price) != 1 {
		t.Fatalf("price samples = %d, want 1", len(price))
	}
	assertClose(t, price[0].Value, mean)
}

func TestSamplesFromStatisticsFallsBackWhenColumnsAreOmitted(t *testing.T) {
	start := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	sum := 44.5
	stats := []haStatistic{
		{Start: json.RawMessage(`"` + start.Format(time.RFC3339) + `"`), Sum: &sum},
	}

	samples := samplesFromStatistics(stats, "state", "sum")
	if len(samples) != 1 {
		t.Fatalf("samples = %d, want 1", len(samples))
	}
	assertClose(t, samples[0].Value, sum)
}

func TestParseHistoryStartDefaultsToCurrentYear(t *testing.T) {
	now := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)
	start, err := parseHistoryStart("", now)
	if err != nil {
		t.Fatalf("parse history start: %v", err)
	}
	if start.Year() != 2026 || start.Month() != time.January || start.Day() != 1 {
		t.Fatalf("history start = %s, want first day of 2026", start)
	}
}

func assertClose(t *testing.T, got float64, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.000001 {
		t.Fatalf("got %.6f, want %.6f", got, want)
	}
}

func strconvFormatInt(value int64) string {
	return strconv.FormatInt(value, 10)
}
