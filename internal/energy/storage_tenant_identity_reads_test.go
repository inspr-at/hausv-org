package energy_test

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/dbtest"
	"github.com/inspr-at/hausv-org/internal/energy"
	"github.com/inspr-at/hausv-org/internal/store"
)

// TestEnergyReadsAddressRowsByTenantIdentityNotTheLabel closes the blind spot
// the previous release wrote down and could not test.
//
// Until now every energy read filtered on tenant_slug, so a row carrying the
// WRONG tenant_id was indistinguishable from a correct one: the boot-time
// completeness check counts NULLs and a wrong id is not NULL, and no read ever
// looked at the column. This seeds exactly that row — labelled with one house's
// slug, owned by another house's identity — and asks both houses for it.
//
// Where it is blind: it proves the READ follows the identity. The conflict
// arbiters still name tenant_slug, because every energy primary key leads with
// it on both engines, so this is a dual-key layer and not a slug-free one.
func TestEnergyReadsAddressRowsByTenantIdentityNotTheLabel(t *testing.T) {
	database, lanes := openEnergyLanes(t)
	identities, err := store.EnsureTenantIdentities(t.Context(), database, []store.TenantIdentity{
		{Slug: "haus-a", Name: "Haus A"},
		{Slug: "haus-b", Name: "Haus B"},
	})
	if err != nil {
		t.Fatalf("boot: %v", err)
	}
	now := time.Date(2026, time.August, 18, 9, 0, 0, 0, time.UTC)
	storage := energy.NewSQLStore(lanes)
	for _, slug := range []string{"haus-a", "haus-b"} {
		if err := storage.SaveProfile(energy.DefaultProfile(slug, now)); err != nil {
			t.Fatalf("save %s profile: %v", slug, err)
		}
	}

	// The row a mis-stamped write would leave behind: haus-a's label, haus-b's
	// identity. The label is what the previous release filtered on.
	seedAsset(t, database, seededAsset{
		id: "asset-mislabelled", tenantID: identities["haus-b"].ID, slug: "haus-a",
		homeKey: "default", name: "Fremdes PV", now: now,
	})

	a, err := storage.ListAssets("haus-a")
	if err != nil {
		t.Fatalf("list haus-a assets: %v", err)
	}
	if len(a) != 0 {
		t.Errorf("haus-a sees a row it does not own: %+v — the read is still following the label", a)
	}
	b, err := storage.ListAssets("haus-b")
	if err != nil {
		t.Fatalf("list haus-b assets: %v", err)
	}
	// The symmetric half. Without it the check above passes for a read that
	// returns nothing at all, which is the same shape of vacuity the
	// cross-tenant asset test used to have.
	if len(b) != 1 || b[0].ID != "asset-mislabelled" {
		t.Errorf("haus-b does not see the row its identity owns: %+v", b)
	}
}

// TestARowFromThePreviousReleaseIsHealedByTheNextWrite is the runtime evidence
// the package never had.
//
// TestEveryEnergyUpsertHealsTheIdentity reads string literals: it can see that
// coalesce(<table>.tenant_id, excluded.tenant_id) is written down, and nothing
// more. Nothing here could ever BUILD the state it reasons about, because every
// fixture writes through the Storage API and that always binds an identity. So
// this seeds the rollback window's row shape directly — tenant_slug set,
// tenant_id NULL, exactly what the previous release wrote — and drives the
// three different repair mechanisms this package has over it.
//
// It also pins the trap that makes the repair necessary rather than merely
// tidy: without it, ensureProfile finds no profile through the tenant_id filter
// and writes a DEFAULT one, and that upsert would overwrite a real household's
// name, mode and onboarding state.
func TestARowFromThePreviousReleaseIsHealedByTheNextWrite(t *testing.T) {
	database, lanes := openEnergyLanes(t)
	// On PostgreSQL the rows this seeds cannot exist since migration 0006 made
	// tenant_id NOT NULL; the helper proves that refusal, and with it there is
	// nothing left here to assert on that engine. On SQLite — production, and
	// the open rollback window — it proves the row IS accepted and carries on.
	if dbtest.RollbackWindowClosed(t, database) {
		return
	}
	if _, err := store.EnsureTenantIdentities(t.Context(), database,
		[]store.TenantIdentity{{Slug: "haus-a", Name: "Haus A"}}); err != nil {
		t.Fatalf("boot: %v", err)
	}
	now := time.Date(2026, time.August, 18, 9, 0, 0, 0, time.UTC)
	storage := energy.NewSQLStore(lanes)
	if err := storage.SaveProfile(energy.DefaultProfile("haus-a", now)); err != nil {
		t.Fatalf("save profile: %v", err)
	}

	// Everything below belongs to a second home of the same house and is
	// written the way the PREVIOUS release wrote it: slug, no identity.
	const home = "altbau"
	seedUnownedProfile(t, database, "haus-a", home, "Altbestand", now)
	seedAsset(t, database, seededAsset{id: "legacy-asset-1", slug: "haus-a", homeKey: home, name: "Alte PV", now: now})
	seedAsset(t, database, seededAsset{id: "legacy-asset-2", slug: "haus-a", homeKey: home, name: "Alte WP", now: now})
	seedUnownedImport(t, database, "haus-a", home, "legacy-sha", now)

	legacy := storage.ForHome(home)

	// State of the window before any write: the rows exist and are invisible.
	// Saying so out loud is the point — this is what a boot without
	// store.BackfillTenantIDs would serve.
	if _, found, err := legacy.Profile("haus-a"); err != nil || found {
		t.Fatalf("an identity-less profile must not be visible to a tenant_id read: found=%v err=%v", found, err)
	}
	if assets, err := legacy.ListAssets("haus-a"); err != nil || len(assets) != 0 {
		t.Fatalf("identity-less assets must not be visible: %+v err=%v", assets, err)
	}

	// One write through the new binary.
	if err := legacy.UpsertAsset(energy.Asset{
		ID: "legacy-asset-1", TenantSlug: "haus-a", Kind: "pv", Name: "PV neu", Confirmed: true,
	}); err != nil {
		t.Fatalf("upsert over the unowned row: %v", err)
	}

	profile, found, err := legacy.Profile("haus-a")
	if err != nil || !found {
		t.Fatalf("the profile must be adopted, not left invisible: found=%v err=%v", found, err)
	}
	if profile.HouseholdName != "Altbestand" {
		t.Errorf("the existing household was overwritten with defaults: household_name = %q, want %q",
			profile.HouseholdName, "Altbestand")
	}

	assets, err := legacy.ListAssets("haus-a")
	if err != nil {
		t.Fatalf("list assets after the heal: %v", err)
	}
	if len(assets) != 1 || assets[0].ID != "legacy-asset-1" {
		t.Fatalf("exactly the written-over row must have become visible, got %+v", assets)
	}
	// And it became visible BECAUSE it was healed, not because the filter is
	// loose: its untouched sibling is still unowned and still invisible.
	if id := storedTenantID(t, database, "legacy-asset-2"); id.Valid {
		t.Errorf("legacy-asset-2 was never written and must still be unowned, carries %q", id.String)
	}

	// energy_imports is the one upsert with no healing conflict clause, so its
	// repair is a separate statement — and it must not disturb the
	// already-imported answer the caller depends on.
	inserted, err := legacy.PutImport(energy.ImportRecord{
		TenantSlug: "haus-a", Filename: "export.csv", SHA256: "legacy-sha",
		Format: "smart-meter-csv", Payload: []byte("timestamp;import_kwh\n"), ImportedAt: now,
	}, nil)
	if err != nil {
		t.Fatalf("re-import the same file: %v", err)
	}
	if inserted {
		t.Error("re-importing an identical file must still report it as already imported")
	}
	imports, err := legacy.ListImports("haus-a")
	if err != nil {
		t.Fatalf("list imports after the heal: %v", err)
	}
	if len(imports) != 1 || imports[0].SHA256 != "legacy-sha" {
		t.Fatalf("the re-imported file must have been adopted and be visible, got %+v", imports)
	}
}

type seededAsset struct {
	id       string
	tenantID string
	slug     string
	homeKey  string
	name     string
	now      time.Time
}

// seedAsset writes a row the Storage API cannot produce: an explicit identity
// that is not this slug's, or none at all.
func seedAsset(t *testing.T, database *sql.DB, asset seededAsset) {
	t.Helper()
	var tenantID any
	if asset.tenantID != "" {
		tenantID = asset.tenantID
	}
	stamp := asset.now.UTC().Format(time.RFC3339Nano)
	if _, err := database.ExecContext(t.Context(),
		`INSERT INTO energy_assets(id,tenant_id,tenant_slug,home_key,kind,name,rated_power_kw,flexibility,source,confirmed,metadata_json,created_at,updated_at)
		 VALUES($1,$2,$3,$4,'pv',$5,NULL,'unknown','manual',$6,'{}',$7,$8)`,
		asset.id, tenantID, asset.slug, asset.homeKey, asset.name, false, stamp, stamp); err != nil {
		t.Fatalf("seed asset %s: %v", asset.id, err)
	}
}

func seedUnownedProfile(t *testing.T, database *sql.DB, slug, homeKey, household string, now time.Time) {
	t.Helper()
	stamp := now.UTC().Format(time.RFC3339Nano)
	if _, err := database.ExecContext(t.Context(),
		`INSERT INTO home_profiles(tenant_slug,home_key,unit_id,home_type,household_name,operating_mode,automation_stage,
		 onboarding_step,onboarding_complete,target_peak_kw,agreed_power_kw,recommendation_id,recommendation_status,
		 free_started_at,free_until_at,created_at,updated_at)
		 VALUES($1,$2,'','apartment',$3,'observe','observe',1,$4,NULL,NULL,'','',NULL,NULL,$5,$6)`,
		slug, homeKey, household, true, stamp, stamp); err != nil {
		t.Fatalf("seed unowned profile %s/%s: %v", slug, homeKey, err)
	}
}

func seedUnownedImport(t *testing.T, database *sql.DB, slug, homeKey, sha string, now time.Time) {
	t.Helper()
	stamp := now.UTC().Format(time.RFC3339Nano)
	if _, err := database.ExecContext(t.Context(),
		`INSERT INTO energy_imports(id,tenant_slug,home_key,filename,sha256,format,payload,imported_at)
		 VALUES($1,$2,$3,'export.csv',$4,'smart-meter-csv',$5,$6)`,
		"legacy-import-"+sha, slug, homeKey, sha, []byte("timestamp;import_kwh\n"), stamp); err != nil {
		t.Fatalf("seed unowned import %s: %v", sha, err)
	}
}

func storedTenantID(t *testing.T, database *sql.DB, assetID string) sql.NullString {
	t.Helper()
	var id sql.NullString
	if err := database.QueryRowContext(t.Context(),
		`SELECT tenant_id FROM energy_assets WHERE id=$1`, assetID).Scan(&id); err != nil {
		t.Fatalf("read tenant_id of %s: %v", assetID, err)
	}
	return id
}

// TestSubSecondTimestampsSurviveTheRoundTripOnBothEngines is the evidence for
// the trap that turned out NOT to be present here, which is worth a test rather
// than a claim.
//
// The store port lost sub-second precision because its timestamps went through
// engine-native types. Every *_at column in all eight energy tables is `text`
// on PostgreSQL and TEXT on SQLite, written and parsed with RFC3339Nano, so
// there is no conversion to lose anything in. This asserts that instead of
// asserting the schema: if anyone ever retypes one of these columns to
// timestamptz, the nanoseconds go and this fails.
func TestSubSecondTimestampsSurviveTheRoundTripOnBothEngines(t *testing.T) {
	storage := openEnergyStore(t)
	precise := time.Date(2026, time.August, 18, 9, 30, 15, 123456789, time.UTC)

	profile := energy.DefaultProfile("haus-a", precise)
	profile.FreeStartedAt = &precise
	if err := storage.SaveProfile(profile); err != nil {
		t.Fatalf("save profile: %v", err)
	}
	stored, ok, err := storage.Profile("haus-a")
	if err != nil || !ok {
		t.Fatalf("load profile: ok=%v err=%v", ok, err)
	}
	if stored.FreeStartedAt == nil || !stored.FreeStartedAt.Equal(precise) {
		t.Errorf("free_started_at came back as %v, want %v", stored.FreeStartedAt, precise)
	}

	if err := storage.PutInterval(energy.Interval{
		TenantSlug: "haus-a", StartsAt: precise, Duration: 15 * time.Minute,
		ImportKWh: 0.25, AverageKW: 1, Quality: energy.QualityMeasured, Source: "home-assistant",
	}); err != nil {
		t.Fatalf("put interval: %v", err)
	}
	intervals, err := storage.ListIntervals("haus-a", precise.Add(-time.Second), time.Time{})
	if err != nil {
		t.Fatalf("list intervals: %v", err)
	}
	if len(intervals) != 1 || !intervals[0].StartsAt.Equal(precise) {
		t.Fatalf("interval start came back as %+v, want %v", intervals, precise)
	}

	if err := storage.UpsertMeasure(energy.Measure{
		ID: "measure-precise", TenantSlug: "haus-a", IssueID: "issue-precise",
		Title: "Termin", Status: energy.MeasureScheduled, AppointmentAt: &precise,
	}); err != nil {
		t.Fatalf("upsert measure: %v", err)
	}
	measure, ok, err := storage.GetMeasure("haus-a", "measure-precise")
	if err != nil || !ok {
		t.Fatalf("load measure: ok=%v err=%v", ok, err)
	}
	if measure.AppointmentAt == nil || !measure.AppointmentAt.Equal(precise) {
		t.Errorf("appointment_at came back as %v, want %v", measure.AppointmentAt, precise)
	}
}

// TestABoundStoreRefusesAnotherHousesSlug proves the boundary is load-bearing
// rather than decorative.
//
// ForTenant is where the server hands the storage layer the identity the
// request was authorized with. If a slug that disagrees with it were quietly
// answered anyway, binding the reference would buy nothing: the query would go
// on being steered by whatever string reached it last, which is the shape the
// port set out to remove.
func TestABoundStoreRefusesAnotherHousesSlug(t *testing.T) {
	database, lanes := openEnergyLanes(t)
	identities, err := store.EnsureTenantIdentities(t.Context(), database, []store.TenantIdentity{
		{Slug: "haus-a", Name: "Haus A"},
		{Slug: "haus-b", Name: "Haus B"},
	})
	if err != nil {
		t.Fatalf("boot: %v", err)
	}
	now := time.Date(2026, time.August, 18, 9, 0, 0, 0, time.UTC)
	root := energy.NewSQLStore(lanes)
	for _, slug := range []string{"haus-a", "haus-b"} {
		profile := energy.DefaultProfile(slug, now)
		profile.HouseholdName = "Haushalt " + slug
		if err := root.SaveProfile(profile); err != nil {
			t.Fatalf("save %s profile: %v", slug, err)
		}
	}

	bound := root.ForTenant(identities["haus-a"].Ref())
	if _, _, err := bound.Profile("haus-b"); !errors.Is(err, energy.ErrTenantScopeMismatch) {
		t.Fatalf("a store bound to haus-a must refuse haus-b, got err=%v", err)
	}
	// The same handle still answers for the house it was bound to, so the guard
	// is a guard and not a general refusal.
	profile, ok, err := bound.Profile("haus-a")
	if err != nil || !ok || profile.HouseholdName != "Haushalt haus-a" {
		t.Fatalf("the bound house must still be readable: %+v ok=%v err=%v", profile, ok, err)
	}
}
