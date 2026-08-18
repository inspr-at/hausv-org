package server

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	appdb "github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/dbtest"
	"github.com/inspr-at/hausv-org/internal/energy"
	storepkg "github.com/inspr-at/hausv-org/internal/store"
)

// This file is the engine-honest oracle for the tenant-scoping work.
//
// Before it existed, no test in internal/server touched PostgreSQL — exactly one
// test in the whole tree did. Every claim about tenant isolation at the handler
// level was therefore proven on SQLite, which has no row-level security at all,
// so the layer that is supposed to enforce the scope on the only engine that can
// enforce it was never exercised by the layer that calls it. A leak introduced
// while wiring scoped sessions would have shipped green.
//
// newSQLTestApp builds the app on whichever engine dbtest selects: SQLite by
// default, PostgreSQL in an isolated schema when HAUSV_STORE_TEST_POSTGRES and
// HAUSV_TEST_POSTGRES_DSN are set. The same test body runs on both, so it is
// still a real test on the default path and becomes the RLS oracle on the other.

type sqlTestTenant struct {
	Slug string
	Name string
}

// newSQLTestApp returns an app whose tenant-bound repositories are served by a
// real database, together with the tenant identities the database minted.
//
// The identities come from the database rather than from testTenantRef: the
// point of this harness is that the ids the handlers use are the ids the rows
// carry, and inventing them in the test would break exactly the link it is here
// to check.
func newSQLTestApp(t *testing.T, tenants []sqlTestTenant, profiles ...userProfile) (*app, map[string]storepkg.TenantRef) {
	t.Helper()
	if len(tenants) == 0 {
		t.Fatal("a tenant-scoping harness needs at least one tenant")
	}
	a := newTestPortalApp(t, profiles[0])
	database, dbCfg := dbtest.OpenWithConfig(t)
	a.pool = database
	// The stores below reach the database only through this seam, and on
	// PostgreSQL every handle it hands out is a real lane dialled from this
	// test schema's DSN. That is the only reason this oracle can see a scoping
	// mistake at all: handing the stores the plain pool instead would leave
	// every lane in the converted stores untested while still passing.
	scoped, err := appdb.NewScoped(dbCfg, database)
	if err != nil {
		t.Fatalf("open scoped test lanes: %v", err)
	}
	t.Cleanup(func() { _ = scoped.Close() })
	a.scopedDB = scoped
	a.tenantDB = storepkg.NewTenantDB(scoped)
	a.dataDir = t.TempDir()

	configured := make([]storepkg.TenantIdentity, 0, len(tenants))
	for _, tenant := range tenants {
		configured = append(configured, storepkg.TenantIdentity{Slug: tenant.Slug, Name: tenant.Name})
	}
	identities, err := storepkg.EnsureTenantIdentities(context.Background(), database, configured)
	if err != nil {
		t.Fatalf("ensure tenant identities: %v", err)
	}

	a.tenants = map[string]tenantConfig{}
	a.tenantIdentities = map[string]storepkg.TenantIdentity{}
	refs := map[string]storepkg.TenantRef{}
	for _, tenant := range tenants {
		identity, ok := identities[tenant.Slug]
		if !ok || !identity.Ref().Valid() {
			t.Fatalf("tenant identity missing for %s", tenant.Slug)
		}
		a.tenants[tenant.Slug] = tenantConfig{Slug: tenant.Slug, Name: tenant.Name, HeroImageURL: defaultTenantHeroImageURL}
		a.tenantIdentities[tenant.Slug] = identity
		refs[tenant.Slug] = identity.Ref()
	}
	a.defaultTenant = tenants[0].Slug

	a.profiles = map[string]userProfile{}
	for _, profile := range profiles {
		a.profiles[normalizeEmail(profile.Email)] = profile
	}

	// Every tenant-bound store the request repositories are built from, served
	// by the database instead of by the in-memory doubles. A store left on its
	// double would silently opt out of the oracle.
	files := t.TempDir()
	a.announcementStore = newSQLAnnouncementStore(a.tenantDB)
	a.announcementReadStore = newSQLAnnouncementReadStore(a.tenantDB)
	a.eventStore = newSQLEventStore(a.tenantDB)
	a.contactStore = newSQLContactBookStore(a.tenantDB)
	a.documentStore = newSQLDocumentStore(a.tenantDB, filepath.Join(files, "documents"))
	a.attachmentStore = newSQLAttachmentStore(a.tenantDB, filepath.Join(files, "attachments"))
	a.handoverStore = newSQLHandoverStore(a.tenantDB)
	a.issueStore = newSQLIssueStore(a.tenantDB, filepath.Join(files, "issues"))
	a.unitStore = newSQLUnitStore(a.tenantDB)
	a.unitPaymentStore = newSQLUnitPaymentStatusStore(a.tenantDB)
	a.voteStore = newSQLVoteStore(a.tenantDB)
	a.activityStore = newSQLActivityStore(a.tenantDB)
	a.notificationPrefs = newSQLNotificationPrefStore(a.tenantDB)
	a.profileOverlays = newSQLProfileOverlayStore(a.tenantDB)
	a.identityStore = newSQLIdentityStore(a.tenantDB)
	a.inviteStore = a.identityStore
	a.energyStore = energy.NewSQLStore(database)
	a.homeReservations = storepkg.NewSQLHomeReservationStore(a.tenantDB)
	a.homePortals = storepkg.NewSQLHomePortalStore(a.tenantDB)
	a.homeConnectors = storepkg.NewSQLHomeConnectorStore(a.tenantDB)
	a.homeConnectorReadings = storepkg.NewSQLHomeConnectorReadingStore(a.tenantDB)

	for _, profile := range profiles {
		if _, err := a.inviteStore.Add(profile); err != nil {
			t.Fatalf("seed profile %s: %v", profile.Email, err)
		}
	}
	return a, refs
}

// sqlSessionGet drives the real handler chain — recover, security headers,
// canonical host, tenant path resolution, the session guard, the handler — for a
// signed-in member of one tenant.
func sqlSessionGet(t *testing.T, a *app, email string, tenantSlug string, path string) *httptest.ResponseRecorder {
	t.Helper()
	token, _, err := a.sessions.Put(email, tenantSlug, authMethodEmail, time.Hour)
	if err != nil {
		t.Fatalf("put session: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://hausv.org"+path, nil)
	req.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	rr := httptest.NewRecorder()
	a.handler().ServeHTTP(rr, req)
	return rr
}

func sqlSessionPost(t *testing.T, a *app, email string, tenantSlug string, path string, values url.Values) *httptest.ResponseRecorder {
	t.Helper()
	token, _, err := a.sessions.Put(email, tenantSlug, authMethodEmail, time.Hour)
	if err != nil {
		t.Fatalf("put session: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "http://hausv.org"+path, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://hausv.org")
	req.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	rr := httptest.NewRecorder()
	a.handler().ServeHTTP(rr, req)
	return rr
}

// TestTwoTenantsCannotSeeEachOtherThroughTheHandlerChain is the assertion the
// scoped-session work has to keep true. Both houses are written through the real
// POST handlers and read back through the real GET handlers, so a scope that is
// applied on the write path but not the read path (or the reverse) fails here.
func TestTwoTenantsCannotSeeEachOtherThroughTheHandlerChain(t *testing.T) {
	managerA := userProfile{Email: "manager-a@example.com", Role: roleManager, Tenants: []string{"haus-a"}, AuthMethods: defaultAuthMethods()}
	managerB := userProfile{Email: "manager-b@example.com", Role: roleManager, Tenants: []string{"haus-b"}, AuthMethods: defaultAuthMethods()}
	a, refs := newSQLTestApp(t,
		[]sqlTestTenant{{Slug: "haus-a", Name: "Haus A"}, {Slug: "haus-b", Name: "Haus B"}},
		managerA, managerB,
	)

	const secretA = "Nur-fuer-Haus-A-Aushang"
	const secretB = "Nur-fuer-Haus-B-Aushang"
	for _, seed := range []struct {
		email  string
		tenant string
		secret string
	}{
		{managerA.Email, "haus-a", secretA},
		{managerB.Email, "haus-b", secretB},
	} {
		create := sqlSessionPost(t, a, seed.email, seed.tenant, "/"+seed.tenant+"/app/announcements", url.Values{
			"title":    {seed.secret},
			"body":     {seed.secret + "-Text"},
			"category": {"Info"},
		})
		if create.Code != http.StatusSeeOther {
			t.Fatalf("create announcement for %s = %d, want 303:\n%s", seed.tenant, create.Code, create.Body.String())
		}
	}

	// The read path, through the handler chain, for each house.
	for _, want := range []struct {
		email    string
		tenant   string
		visible  string
		hidden   string
		otherRef storepkg.TenantRef
	}{
		{managerA.Email, "haus-a", secretA, secretB, refs["haus-b"]},
		{managerB.Email, "haus-b", secretB, secretA, refs["haus-a"]},
	} {
		page := sqlSessionGet(t, a, want.email, want.tenant, "/"+want.tenant+"/app/announcements")
		if page.Code != http.StatusOK {
			t.Fatalf("%s announcements status = %d", want.tenant, page.Code)
		}
		body := page.Body.String()
		if !strings.Contains(body, want.visible) {
			t.Fatalf("%s cannot see its own announcement %q", want.tenant, want.visible)
		}
		if strings.Contains(body, want.hidden) {
			t.Fatalf("SECURITY: %s was served %q, which belongs to tenant %s", want.tenant, want.hidden, want.otherRef.ID)
		}
	}

	// And the rows really do carry distinct identities — a page that renders
	// nothing would satisfy the assertion above without proving anything.
	assertOneRowPerTenant(t, testPool(t, a), "announcements", refs["haus-a"].ID, refs["haus-b"].ID)
}

// TestSessionOfOneTenantCannotDriveAnotherTenantsPath pins the other half: the
// scope a request runs under comes from the session, so a member of one house
// pointing their browser at another house's prefix must not be served it.
func TestSessionOfOneTenantCannotDriveAnotherTenantsPath(t *testing.T) {
	managerA := userProfile{Email: "manager-a@example.com", Role: roleManager, Tenants: []string{"haus-a"}, AuthMethods: defaultAuthMethods()}
	managerB := userProfile{Email: "manager-b@example.com", Role: roleManager, Tenants: []string{"haus-b"}, AuthMethods: defaultAuthMethods()}
	a, _ := newSQLTestApp(t,
		[]sqlTestTenant{{Slug: "haus-a", Name: "Haus A"}, {Slug: "haus-b", Name: "Haus B"}},
		managerA, managerB,
	)

	const secretB = "Fremd-Aushang-Haus-B"
	create := sqlSessionPost(t, a, managerB.Email, "haus-b", "/haus-b/app/announcements", url.Values{
		"title": {secretB}, "body": {secretB + "-Text"}, "category": {"Info"},
	})
	if create.Code != http.StatusSeeOther {
		t.Fatalf("seed announcement = %d", create.Code)
	}

	// Session says haus-a, path says haus-b.
	crossed := sqlSessionGet(t, a, managerA.Email, "haus-a", "/haus-b/app/announcements")
	if crossed.Code == http.StatusOK && strings.Contains(crossed.Body.String(), secretB) {
		t.Fatal("SECURITY: a haus-a session was served haus-b content")
	}
	if crossed.Code != http.StatusSeeOther {
		t.Fatalf("cross-tenant path status = %d, want a redirect away", crossed.Code)
	}
}

// testPool is the process pool behind an app, for fixtures and assertions that
// must read OUTSIDE every lane. The app itself only holds the pool's lifecycle
// surface — that narrowing is what keeps handlers off it — so a test that needs
// the statement surface says so here, in one place, by asserting the concrete
// type.
func testPool(t *testing.T, a *app) *sql.DB {
	t.Helper()
	pool, ok := a.pool.(*sql.DB)
	if !ok {
		t.Fatalf("app.pool is %T, not the *sql.DB the fixture needs", a.pool)
	}
	return pool
}

func assertOneRowPerTenant(t *testing.T, database *sql.DB, table string, tenantIDs ...string) {
	t.Helper()
	for _, tenantID := range tenantIDs {
		var count int
		if err := database.QueryRow(`SELECT count(*) FROM `+table+` WHERE tenant_id=$1`, tenantID).Scan(&count); err != nil {
			t.Fatalf("count %s rows for %s: %v", table, tenantID, err)
		}
		if count != 1 {
			t.Fatalf("%s rows for tenant %s = %d, want 1", table, tenantID, count)
		}
	}
}

// The seam has to exist on a real boot, not only in the package that defines
// it. Nothing consumes it yet — that is deliberate — so without this the whole
// construction could be dropped from newApp and every other test would stay
// green.
func TestBootConstructsTheScopedTenantSeam(t *testing.T) {
	t.Setenv("DB_PATH", filepath.Join(t.TempDir(), "boot.db"))
	t.Setenv("PARKING_DATA_PATH", filepath.Join(t.TempDir(), "parking.json"))
	t.Setenv("BASE_URL", "http://localhost:8080")

	a, err := newApp()
	if err != nil {
		t.Fatalf("boot: %v", err)
	}
	t.Cleanup(a.closeMagicLinkDelivery)
	if a.tenantDB == nil {
		t.Fatal("newApp did not construct the scoped tenant seam")
	}
	// On SQLite every accessor is the one process pool, which is what keeps this
	// batch behaviour-neutral.
	if a.tenantDB.Unscoped("boot check") != testPool(t, a) {
		t.Fatal("the seam must hand back the process pool on SQLite")
	}
	// And the lanes are owned: shutdown closes them, not just the process pool.
	if a.scopedDB == nil {
		t.Fatal("nothing owns the lane pools, so nothing closes them")
	}
	if err := a.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

// An unusable lane plan must stop the boot. It is the only new way this batch
// can refuse to start, so it is worth being explicit about.
func TestBootRefusesAnUnusableLanePlan(t *testing.T) {
	t.Setenv("DB_PATH", filepath.Join(t.TempDir(), "boot.db"))
	t.Setenv("PARKING_DATA_PATH", filepath.Join(t.TempDir(), "parking.json"))
	t.Setenv("BASE_URL", "http://localhost:8080")
	t.Setenv("DB_LANE_CAP", "0")

	if _, err := newApp(); err == nil {
		t.Fatal("a lane cap of zero must be refused at boot")
	}
}
