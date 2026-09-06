package dbmove_test

// The round-trip proof for the data mover.
//
// A count-only comparison is not a proof: a mover that drops a column silently
// passes it. So this file builds a SQLite database from the real migrations,
// fills EVERY table through the application's own write paths (and, for the
// three tables no store writes, the same INSERTs the application uses), moves
// it into a fresh PostgreSQL schema through the mover, and then compares in
// two independent ways:
//
//   - every table, row by row, value by value, with a normaliser written here
//     and NOT shared with the mover — so a converter defect and a hash defect
//     cannot cancel out;
//   - the application's own read paths, bound over the moved tenant on
//     PostgreSQL, against the same calls on the SQLite side.
//
// It runs in the PostgreSQL suite (HAUSV_STORE_TEST_POSTGRES + DSN) and skips
// otherwise, because half of it is PostgreSQL. In that suite it must PASS.

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	appdb "github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/dbmove"
	"github.com/inspr-at/hausv-org/internal/dbtest"
	"github.com/inspr-at/hausv-org/internal/energy"
	"github.com/inspr-at/hausv-org/internal/store"
)

// testLaneReason is the maintenance-lane reason the proof runs the mover on;
// the production reason lives at the call site in cmd/hausv-org.
const testLaneReason = "test: the mover proof loads and reads every tenant's rows on the target, which is what the mover does at cutover"

func requirePostgres(t *testing.T) {
	t.Helper()
	if dbtest.Backend() != appdb.BackendPostgres {
		t.Skip("the mover's target is PostgreSQL: set HAUSV_STORE_TEST_POSTGRES and HAUSV_TEST_POSTGRES_DSN")
	}
}

// source is a migrated SQLite database with the application's store layer
// wired over it, exactly as production wires it.
type source struct {
	path     string
	db       *sql.DB
	lanes    *store.TenantDB
	files    string
	tenants  map[string]store.TenantIdentity
	energy   *energy.SQLStore
	identity *store.SQLIdentityStore
}

func openSource(t *testing.T) *source {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "hausv.db")
	database, err := appdb.Open(path)
	if err != nil {
		t.Fatalf("open sqlite source: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	scoped, err := appdb.NewScoped(appdb.Config{Backend: appdb.BackendSQLite, DSN: path}, database)
	if err != nil {
		t.Fatalf("scoped over sqlite: %v", err)
	}
	t.Cleanup(func() { _ = scoped.Close() })
	lanes := store.NewTenantDB(scoped)
	return &source{
		path:     path,
		db:       database,
		lanes:    lanes,
		files:    filepath.Join(dir, "files"),
		energy:   energy.NewSQLStore(lanes),
		identity: store.NewSQLIdentityStore(lanes),
	}
}

// target is an isolated PostgreSQL schema plus the maintenance lane the mover
// writes through, and the store layer over it for the read-path comparison.
type target struct {
	db     *sql.DB
	cfg    appdb.Config
	scoped *appdb.Scoped
	lane   appdb.Handle
	lanes  *store.TenantDB
}

func openTarget(t *testing.T) *target {
	t.Helper()
	database, cfg := dbtest.OpenWithConfig(t)
	scoped, err := appdb.NewScoped(cfg, database)
	if err != nil {
		t.Fatalf("scoped over postgres: %v", err)
	}
	t.Cleanup(func() { _ = scoped.Close() })
	return &target{
		db:     database,
		cfg:    cfg,
		scoped: scoped,
		lane:   scoped.Unscoped(testLaneReason),
		lanes:  store.NewTenantDB(scoped),
	}
}

// The QA fixture tenants and users, transcribed from scripts/snapshot/env.sh so
// the largest seed the repository has is the one this proof moves.
var qaTenants = []store.TenantIdentity{
	{Slug: "demo", Name: "Demohaus"},
	{Slug: "haus-a", Name: "Haus A"},
	{Slug: "haus-b", Name: "Haus B"},
	{Slug: "cockpit", Name: "Energiehaus"},
}

var qaUsers = []store.UserProfile{
	{Email: "admin@example.com", FirstName: "Ada", LastName: "Admin", Role: "Admin", Status: "Aktiv", Tenants: []string{"demo"}, AuthMethods: []string{"email"}},
	{Email: "multi@example.com", FirstName: "Mara", LastName: "Mehrhaus", Role: "Admin", Status: "Aktiv", Tenants: []string{"demo", "haus-b"}, AuthMethods: []string{"email"}},
	{Email: "verwalter@example.com", FirstName: "Vera", LastName: "Verwalter", Role: "Verwalter", Status: "Aktiv", Tenants: []string{"demo"}, AuthMethods: []string{"email"}},
	{Email: "owner@example.com", FirstName: "Otto", LastName: "Eigentuemer", Role: "Eigentümer", Status: "Aktiv", Tenants: []string{"demo"}, AuthMethods: []string{"email"}},
	{Email: "resident@example.com", FirstName: "Rita", LastName: "Bewohnerin", Role: "Bewohner", Status: "Aktiv", Tenants: []string{"demo"}, AuthMethods: []string{"email"}},
	{Email: "house-a-owner@example.com", FirstName: "Alex", LastName: "Eigentuemer", Role: "Eigentümer", Status: "Aktiv", Tenants: []string{"haus-a"}, AuthMethods: []string{"email"}},
	{Email: "house-b-owner@example.com", FirstName: "Bianca", LastName: "Eigentuemerin", Role: "Eigentümer", Status: "Aktiv", Tenants: []string{"haus-b"}, AuthMethods: []string{"email"}},
	{Email: "cockpit-owner@example.com", FirstName: "Clara", LastName: "Eigentuemerin", Role: "Eigentümer", Status: "Aktiv", Tenants: []string{"cockpit"}, AuthMethods: []string{"email"}, Title: "Dr.", Deactivated: true, Adopted: true},
}

const qaHomeProfileSeeds = `[{"tenant_slug":"demo","household_name":"QA Zuhause","home_type":"apartment","assets":["pv","ev","wallbox"]},{"tenant_slug":"haus-a","household_name":"Haus A","home_type":"house","assets":["pv","ev","hot-water","heat-pump"]},{"tenant_slug":"haus-b","household_name":"Haus B","home_type":"house","assets":["pv","battery","ev"]},{"tenant_slug":"cockpit","household_name":"Energiehaus","home_type":"house","complete":true,"assets":["pv","battery","ev","wallbox","heat-pump","hot-water",{"kind":"sauna","name":"Sauna Keller","rated_power_kw":8,"flexibility":"shift"}]}]`

var onePixelPNG = mustHex("89504e470d0a1a0a0000000d4948445200000001000000010804000000b51c0c020000000b4944415478da6364f8cf500f00038601805a347d6b0000000049454e44ae426082")

func mustHex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

type nopReadSeekCloser struct{ *bytes.Reader }

func (nopReadSeekCloser) Close() error { return nil }

func uploadFrom(name string, data []byte) store.UploadedFile {
	return store.UploadedFile{
		Filename: name,
		Size:     int64(len(data)),
		Open: func() (io.ReadSeekCloser, error) {
			return nopReadSeekCloser{bytes.NewReader(data)}, nil
		},
	}
}

func must(t *testing.T, what string, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", what, err)
	}
}

// seedSmall is the small fixture: one tenant, one person, one membership, a
// few contacts and units. Most tables stay empty, which is a case of its own —
// zero rows must move as cleanly as many.
func seedSmall(t *testing.T) *source {
	t.Helper()
	src := openSource(t)
	ctx := context.Background()
	identities, err := store.EnsureTenantIdentities(ctx, src.db, qaTenants[:1])
	must(t, "tenants", err)
	src.tenants = identities
	if _, err := src.identity.Add(qaUsers[0]); err != nil {
		t.Fatalf("add user: %v", err)
	}
	demo := identities["demo"].Ref()
	contacts, _ := store.BindContactBookRepository(store.NewSQLContactBookStore(src.lanes), demo)
	if _, _, err := contacts.Upsert(store.ManagedContact{TenantSlug: "demo", Kind: "dienstleister", Name: "Installateur Huber", Phone: "+43 316 123", Active: true}); err != nil {
		t.Fatalf("contact: %v", err)
	}
	units, _ := store.BindUnitRepository(store.NewSQLUnitStore(src.lanes), demo)
	must(t, "units", units.SetUnits([]store.Unit{
		{ID: "top-1", Label: "Top 1", UnitType: store.UnitTypeResidential, MiteigentumsanteilPPM: 500_000, OwnerEmails: []string{"admin@example.com"}},
		{ID: "top-2", Label: "Top 2", UnitType: store.UnitTypeCommercial, MiteigentumsanteilPPM: 500_000},
	}))
	return src
}

// seedFull fills every table. The list of tables it must reach is not written
// here: assertEveryTableSeeded reads the catalog, so a table added by a later
// migration fails this seed until it is covered.
func seedFull(t *testing.T) *source {
	t.Helper()
	src := openSource(t)
	ctx := context.Background()
	now := time.Date(2026, 8, 17, 9, 30, 0, 123456789, time.UTC)

	identities, err := store.EnsureTenantIdentities(ctx, src.db, qaTenants)
	must(t, "tenants", err)
	src.tenants = identities
	demo := identities["demo"].Ref()
	hausA := identities["haus-a"].Ref()

	// persons, house_memberships — through the identity store, like an invite.
	for _, user := range qaUsers {
		if _, err := src.identity.Add(user); err != nil {
			t.Fatalf("add %s: %v", user.Email, err)
		}
	}
	// house_memberships.directory_opt_in is a NULLABLE boolean: one true, one
	// false, the rest NULL, so all three states cross.
	if _, err := src.identity.SetTenantDirectoryOptIn("owner@example.com", "demo", true); err != nil {
		t.Fatalf("opt in: %v", err)
	}
	if _, err := src.identity.SetTenantDirectoryOptIn("resident@example.com", "demo", false); err != nil {
		t.Fatalf("opt out: %v", err)
	}

	// login_activity, profile_overlays, notification_prefs, telegram_*.
	activity := store.NewSQLActivityStore(src.lanes)
	must(t, "activity", activity.Touch("admin@example.com", now, "email"))
	must(t, "activity", activity.Touch("owner@example.com", now.Add(-48*time.Hour), "oidc"))
	overlays := store.NewSQLProfileOverlayStore(src.lanes)
	must(t, "overlay", overlays.Set("owner@example.com", store.ProfileOverlay{Title: "Ing.", FirstName: "Otto", LastName: "Eigentümer", Phone: "+43 664 1", DirectoryOptIn: true}))
	must(t, "overlay", overlays.Set("resident@example.com", store.ProfileOverlay{FirstName: "Rita"}))
	prefs := store.NewSQLNotificationPrefStore(src.lanes)
	must(t, "prefs", prefs.Set("owner@example.com", store.NotificationPreferences{Email: map[string]bool{"announcement": true, "ballot": false}}))
	must(t, "prefs", prefs.Set("resident@example.com", store.NotificationPreferences{Email: map[string]bool{}, Unsubscribed: true}))
	telegram := store.NewSQLTelegramStore(src.lanes)
	must(t, "telegram offset", telegram.SetOffset(4711))
	code, err := telegram.CreateLinkCode("owner@example.com", "admin@example.com", time.Hour)
	must(t, "telegram code", err)
	if _, err := telegram.ConsumeLinkCode(code, 987654321, "Otto (Telegram)"); err != nil {
		t.Fatalf("telegram link: %v", err)
	}
	if _, err := telegram.CreateLinkCode("resident@example.com", "admin@example.com", time.Hour); err != nil {
		t.Fatalf("telegram open code: %v", err)
	}

	// app_meta has no writer in the application; it is the migration proof
	// table. Seed it the way db_test does.
	if _, err := src.db.Exec(`INSERT INTO app_meta(key, value) VALUES('schema', 'baseline'), ('seed', 'roundtrip')`); err != nil {
		t.Fatalf("app_meta: %v", err)
	}

	for _, tenant := range []store.TenantRef{demo, hausA} {
		slug := tenant.Slug
		contacts, _ := store.BindContactBookRepository(store.NewSQLContactBookStore(src.lanes), tenant)
		if _, _, err := contacts.Upsert(store.ManagedContact{TenantSlug: slug, Kind: "dienstleister", Name: "Installateur Huber", Phone: "+43 316 123", Active: true, EnergyCapabilities: []string{"pv", "wallbox"}}); err != nil {
			t.Fatalf("%s contact: %v", slug, err)
		}
		inactive, _, err := contacts.Upsert(store.ManagedContact{TenantSlug: slug, Kind: "notdienst", Company: "Notdienst GmbH", Email: "not@example.com", Active: true})
		must(t, slug+" contact 2", err)
		if _, err := contacts.Deactivate(inactive.ID, now); err != nil {
			t.Fatalf("%s deactivate: %v", slug, err)
		}

		announcements, _ := store.BindAnnouncementRepository(store.NewSQLAnnouncementStore(src.lanes), tenant)
		expires := now.Add(72 * time.Hour)
		if _, err := announcements.Create(store.Announcement{TenantSlug: slug, Title: "Hausversammlung", Body: "Am Donnerstag im Hof — „Tagesordnung“ folgt.", Category: "Info", Pinned: true, PublishedAt: now, ExpiresAt: &expires, AuthorEmail: "admin@example.com", AuthorName: "Ada Admin"}); err != nil {
			t.Fatalf("%s announcement: %v", slug, err)
		}
		if _, err := announcements.Create(store.Announcement{TenantSlug: slug, Title: "Wasser abgestellt", Body: "Von 8 bis 12 Uhr.", PublishedAt: now.Add(-24 * time.Hour), AuthorEmail: "verwalter@example.com"}); err != nil {
			t.Fatalf("%s announcement 2: %v", slug, err)
		}
		reads, _ := store.BindAnnouncementReadRepository(store.NewSQLAnnouncementReadStore(src.lanes), tenant)
		must(t, slug+" reads", reads.MarkSeen("owner@example.com", now))
		must(t, slug+" reads", reads.MarkSeen("resident@example.com", now.Add(-time.Hour)))

		events, _ := store.BindEventRepository(store.NewSQLEventStore(src.lanes), tenant)
		ends := now.Add(26 * time.Hour)
		if _, err := events.Create(store.HouseEvent{TenantSlug: slug, Title: "Eigentümerversammlung", Body: "Beschlüsse 2026", Category: "Versammlung", Location: "Hof", StartsAt: now.Add(24 * time.Hour), EndsAt: &ends, AuthorEmail: "admin@example.com"}); err != nil {
			t.Fatalf("%s event: %v", slug, err)
		}
		if _, err := events.Create(store.HouseEvent{TenantSlug: slug, Title: "Kehrtermin", StartsAt: now.Add(-72 * time.Hour), AuthorEmail: "verwalter@example.com"}); err != nil {
			t.Fatalf("%s event 2: %v", slug, err)
		}

		handovers, _ := store.BindHandoverRepository(store.NewSQLHandoverStore(src.lanes), tenant)
		if _, err := handovers.Create(store.HandoverRecord{
			ID: "h-" + slug, TenantSlug: slug, UnitID: "top-1", Title: "Übergabe Top 1", HandoverType: "move-out",
			ScheduledAt: now.Add(7 * 24 * time.Hour), OutgoingName: "Alt Mieter", IncomingName: "Neu Mieter", IncomingEmail: "neu@example.com",
			Rooms:  []store.HandoverRoom{{Name: "Küche", Condition: "gut"}, {Name: "Bad", Condition: "Mängel", Defects: "Fuge"}},
			Meters: []store.HandoverMeter{{Label: "Strom", Value: "12345,6", Unit: "kWh"}},
			Keys:   []store.HandoverKey{{Label: "Haustür", Count: 2}},
			Notes:  "Schlüssel übergeben",
			Confirmations: []store.HandoverConfirmation{
				{Role: "incoming", Name: "Neu Mieter", Email: "neu@example.com", TokenHash: store.HandoverTokenHash("token-" + slug)},
			},
			CreatedBy: "admin@example.com",
		}); err != nil {
			t.Fatalf("%s handover: %v", slug, err)
		}

		documents, _ := store.BindDocumentRepository(store.NewSQLDocumentStore(src.lanes, filepath.Join(src.files, "documents")), tenant)
		first, err := documents.Create(store.DocumentRecord{TenantSlug: slug, Title: "Hausordnung", Visibility: "all", UploadedBy: "admin@example.com"}, uploadFrom("hausordnung.png", onePixelPNG), now)
		must(t, slug+" document", err)
		if _, _, err := documents.Replace(first.ID, "admin@example.com", uploadFrom("hausordnung-v2.png", onePixelPNG), now.Add(time.Hour)); err != nil {
			t.Fatalf("%s document replace: %v", slug, err)
		}
		receiptDocument, err := documents.CreateGenerated(store.DocumentRecord{TenantSlug: slug, Title: "Protokoll", Category: "Protokoll", Visibility: "alle", UploadedBy: "admin@example.com"}, "protokoll.pdf", "application/pdf", []byte("%PDF-1.4 Protokoll der Versammlung"), now)
		if err != nil {
			t.Fatalf("%s generated document: %v", slug, err)
		}

		attachments, _ := store.BindAttachmentRepository(store.NewSQLAttachmentStore(src.lanes, filepath.Join(src.files, "attachments")), tenant)
		created, err := attachments.CreateUploaded("issue", "issue-1", "admin@example.com", []store.UploadedFile{uploadFrom("photo.png", onePixelPNG), uploadFrom("photo2.png", onePixelPNG)}, now)
		must(t, slug+" attachments", err)
		if _, _, err := attachments.Delete(created[1].ID, now.Add(time.Minute)); err != nil {
			t.Fatalf("%s attachment tombstone: %v", slug, err)
		}

		units, _ := store.BindUnitRepository(store.NewSQLUnitStore(src.lanes), tenant)
		must(t, slug+" units", units.SetUnits([]store.Unit{
			{ID: "top-1", Label: "Top 1", UnitType: store.UnitTypeResidential, MiteigentumsanteilPPM: 400_000, OwnerEmails: []string{"owner@example.com"}, RenterEmails: []string{"resident@example.com"}},
			{ID: "top-2", Label: "Top 2", UnitType: store.UnitTypeResidential, MiteigentumsanteilPPM: 400_000, OwnerEmails: []string{"multi@example.com"}},
			{ID: "garage-1", Label: "Garage 1", UnitType: store.UnitTypeParking, BillableWeightPPM: 0, MiteigentumsanteilPPM: 200_000, UsableAreaM2Hundredths: 0, UsableAreaRecorded: true, Persons: 0, PersonsRecorded: true},
		}))
		if unknown, err := units.UpdateAllocationBases([]store.UnitAllocationBasisUpdate{{UnitID: "top-1", UsableAreaM2Hundredths: 7_250, UsableAreaRecorded: true, Persons: 2, PersonsRecorded: true}}); err != nil || unknown {
			t.Fatalf("%s allocation bases: unknown=%t err=%v", slug, unknown, err)
		}
		costTypes, _ := store.BindAnnualStatementCostTypeRepository(store.NewSQLAnnualStatementCostTypeStore(src.lanes), tenant)
		if _, err := costTypes.Save(store.AnnualStatementCostType{Key: "grundsteuer", Name: "Grundsteuer", Allocatable: true, AllocationKey: store.AllocationKeyNutzwert, UpdatedAt: now, UpdatedBy: "verwalter@example.com"}); err != nil {
			t.Fatalf("%s annual statement cost type: %v", slug, err)
		}
		periods, _ := store.BindAnnualStatementPeriodRepository(store.NewSQLAnnualStatementPeriodStore(src.lanes), tenant)
		if _, err := periods.SaveWithStructure(store.AnnualStatementPeriod{
			Year: 2026, StartsOn: "2026-01-01", EndsOn: "2026-12-31", UpdatedAt: now, UpdatedBy: "verwalter@example.com",
		}, costTypes.List(), units.List()); err != nil {
			t.Fatalf("%s annual statement period: %v", slug, err)
		}
		consumption, _ := store.BindAnnualStatementConsumptionRepository(store.NewSQLAnnualStatementConsumptionStore(src.lanes), tenant)
		for _, evidence := range []store.AnnualStatementConsumptionEvidence{
			{UnitID: "top-1", CostTypeKey: "heizung", SourceKind: store.ConsumptionSourceEntity, SourceID: "sensor." + slug + "_heat", MeasuredAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), ValueMicros: 1_000_000, MeasurementUnit: "kWh", ReceivedAt: now},
			{UnitID: "top-1", CostTypeKey: "heizung", SourceKind: store.ConsumptionSourceEntity, SourceID: "sensor." + slug + "_heat", MeasuredAt: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC), ValueMicros: 1_250_000, MeasurementUnit: "kWh", ReceivedAt: now},
		} {
			if _, inserted, err := consumption.Append(evidence); err != nil || !inserted {
				t.Fatalf("%s annual statement consumption: inserted=%t err=%v", slug, inserted, err)
			}
		}
		prepayments, _ := store.BindAnnualStatementPrepaymentRepository(store.NewSQLAnnualStatementPrepaymentStore(src.lanes), tenant)
		if _, _, err := prepayments.Save(store.AnnualStatementPrepayment{PeriodYear: 2026, UnitID: "top-1", AmountCents: 12_550, UpdatedAt: now, UpdatedBy: "verwalter@example.com"}); err != nil {
			t.Fatalf("%s annual statement prepayment: %v", slug, err)
		}
		receipts, _ := store.BindAnnualStatementReceiptRepository(store.NewSQLAnnualStatementReceiptStore(src.lanes), tenant)
		if _, err := receipts.Create(store.AnnualStatementReceipt{
			DocumentID: receiptDocument.ID, PeriodYear: 2026, CostTypeKey: "grundsteuer", AmountCents: 45678,
			InvoiceDate: "2026-07-31", CreatedAt: now, CreatedBy: "verwalter@example.com",
		}); err != nil {
			t.Fatalf("%s annual statement receipt: %v", slug, err)
		}
		// HAUSV-580: a stored run is an immutable calculation with its input
		// snapshot. The seed inserts the row directly: a valid run through the
		// repository would need complete unit bases, receipts and prepayments for
		// every unit, and the round trip proves table values, not the calculation.
		if _, err := src.db.Exec(`INSERT INTO annual_statement_runs(tenant_id, tenant_slug, id, period_year, revision, data) VALUES(?, ?, ?, ?, ?, ?)`,
			tenant.ID, slug, "run-2026-1", 2026, 1,
			`{"id":"run-2026-1","period_year":2026,"revision":1,"calculation_version":1,"created_at":"2026-09-06T10:00:00Z","created_by":"verwalter@example.com","input_hash":"seed","input":{},"result":{}}`); err != nil {
			t.Fatalf("%s annual statement run: %v", slug, err)
		}
		if _, err := src.db.Exec(`INSERT INTO annual_statement_deliveries(tenant_id,tenant_slug,id,run_id,revision,party_id,unit_id,document_id,sha256,recipient,sent_at,status,error,actor,attempt) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, tenant.ID, slug, "delivery-1", "run-2026-1", 1, "owner@example.com", "top-1", receiptDocument.ID, "seed-hash", "owner@example.com", now.UTC().Format(time.RFC3339Nano), "sent", "", "verwalter@example.com", 1); err != nil {
			t.Fatalf("%s annual statement delivery: %v", slug, err)
		}
		payments, _ := store.BindUnitPaymentStatusRepository(store.NewSQLUnitPaymentStatusStore(src.lanes), tenant)
		if _, err := payments.Set(store.UnitPaymentStatus{TenantSlug: slug, UnitID: "top-1", Status: store.UnitPaymentStatusPaid, UpdatedAt: now, UpdatedBy: "verwalter@example.com"}); err != nil {
			t.Fatalf("%s payment: %v", slug, err)
		}
		if _, err := payments.Set(store.UnitPaymentStatus{TenantSlug: slug, UnitID: "top-2", Status: store.UnitPaymentStatusOverdue, UpdatedAt: now}); err != nil {
			t.Fatalf("%s payment 2: %v", slug, err)
		}

		votes, _ := store.BindVoteRepository(store.NewSQLVoteStore(src.lanes), tenant)
		ballot, err := votes.Create(store.Ballot{
			TenantSlug: slug, Title: "Fassadensanierung", Description: "Angebot Firma Bunt", Options: []string{"Ja", "Nein", "Enthaltung"},
			Type: store.BallotTypeCircular, Weighting: store.BallotWeightingPerHead, QuorumPPM: 500_000,
			ClosesAt: now.Add(14 * 24 * time.Hour), CreatedBy: "admin@example.com", ReminderBeforeMinutes: 1440,
		})
		must(t, slug+" ballot", err)
		if _, _, err := votes.Open(ballot.ID, now); err != nil {
			t.Fatalf("%s ballot open: %v", slug, err)
		}
		if _, _, err := votes.CastVote(ballot.ID, "owner@example.com", "Ja", 1, now.Add(time.Hour)); err != nil {
			t.Fatalf("%s vote: %v", slug, err)
		}
		if _, _, err := votes.MarkReminderSent(ballot.ID, []string{"multi@example.com"}, now.Add(2*time.Hour)); err != nil {
			t.Fatalf("%s reminder: %v", slug, err)
		}
		if _, err := votes.Create(store.Ballot{TenantSlug: slug, Title: "Entwurf", Options: []string{"A", "B"}, Type: store.BallotTypeCircular, Weighting: store.BallotWeightingPerHead, CreatedBy: "admin@example.com"}); err != nil {
			t.Fatalf("%s draft ballot: %v", slug, err)
		}

		issues, _ := store.BindIssueRepository(store.NewSQLIssueStore(src.lanes, filepath.Join(src.files, "issues")), tenant)
		issue, err := issues.Create(store.ResidentIssue{TenantSlug: slug, AuthorEmail: "resident@example.com", AuthorName: "Rita", Category: "Schaden", LocationType: "Wohnung", LocationDetail: "Bad", Title: "Wasserhahn tropft", Body: "Seit gestern tropft der Hahn im Bad — bitte prüfen."})
		must(t, slug+" issue", err)
		if _, _, err := issues.AddComment(issue.ID, store.IssueComment{AuthorEmail: "verwalter@example.com", AuthorName: "Vera", Body: "Installateur beauftragt", CreatedAt: now.Add(time.Hour)}); err != nil {
			t.Fatalf("%s comment: %v", slug, err)
		}
		if _, err := issues.Create(store.ResidentIssue{TenantSlug: slug, AuthorEmail: "owner@example.com", AuthorName: "Otto", Category: "Sonstiges", LocationType: "Gemeinschaft", Title: "Licht im Stiegenhaus", Body: "Flackert."}); err != nil {
			t.Fatalf("%s issue 2: %v", slug, err)
		}

		// integration_imports is written by internal/server's import ledger with
		// this exact statement shape.
		if _, err := src.lanes.Unscoped(store.HealOrphanReason).Exec(
			`INSERT INTO integration_imports(tenant_id, tenant_slug, format, file_digest, source_version, applied_at, applied_by, assigned, changed, unclear, rejected)
			 VALUES($1, $2, 'camt.053', $3, '2019', $4, 'verwalter@example.com', 12, 3, 1, 0)`,
			tenant.ID, slug, "sha256:"+slug+"-digest", now.Format(time.RFC3339Nano)); err != nil {
			t.Fatalf("%s integration import: %v", slug, err)
		}
	}

	// The energy chain: the QA seeds first (home_profiles + energy_assets for
	// all four tenants), then everything the seeds do not reach, on demo.
	known := map[string]struct{}{}
	for _, tenant := range qaTenants {
		known[tenant.Slug] = struct{}{}
	}
	must(t, "energy seeds", energy.ApplyProfileSeeds(src.energy, qaHomeProfileSeeds, known, now))
	demoEnergy := src.energy.ForTenant(demo)
	profile, ok, err := demoEnergy.Profile("demo")
	if err != nil || !ok {
		t.Fatalf("demo profile: ok=%v err=%v", ok, err)
	}
	target, agreed := 7.5, 12.0
	freeStart := now.Add(-30 * 24 * time.Hour)
	profile.TargetPeakKW = &target
	profile.AgreedPowerKW = &agreed
	profile.FreeStartedAt = &freeStart
	profile.UnitID = "top-1"
	profile.RecommendationID = "peak"
	profile.RecommendationStatus = "open"
	must(t, "save profile", demoEnergy.SaveProfile(profile))
	pvID := energy.StableAssetID("demo", "pv")
	must(t, "mapping", demoEnergy.UpsertMapping(energy.EntityMapping{ID: "mapping-demo-pv", TenantSlug: "demo", EntityID: "sensor.pv_power", AssetID: pvID, Metric: energy.MetricPVPower, DisplayName: "PV-Leistung", Unit: "kW", DeviceClass: "power", Confirmed: true}))
	must(t, "mapping 2", demoEnergy.UpsertMapping(energy.EntityMapping{ID: "mapping-demo-grid", TenantSlug: "demo", EntityID: "sensor.grid_import", Metric: energy.MetricGridImportEnergy, DisplayName: "Netzbezug", Unit: "kWh", Confirmed: false}))
	vienna, err := time.LoadLocation("Europe/Vienna")
	must(t, "tz", err)
	// Eighteen days of quarter hours: 1,728 intervals. That puts the full seed
	// above the ~1,633 rows the production census counted, and makes the loader
	// flush many batches (insertBatch is 200), so the multi-batch path is on
	// the proof rather than beside it.
	csvBody := "timestamp;import_kwh\n"
	for i := 0; i < 18*96; i++ {
		at := time.Date(2026, 7, 28, 0, 0, 0, 0, vienna).Add(time.Duration(i) * 15 * time.Minute)
		csvBody += at.Format(time.RFC3339) + ";" + strconv.FormatFloat(0.125+float64(i%7)*0.05, 'f', 3, 64) + "\n"
	}
	record, intervals, err := energy.ParseSmartMeterCSV(strings.NewReader(csvBody), "demo", "export.csv", vienna)
	must(t, "csv", err)
	if inserted, err := demoEnergy.PutImport(record, intervals); err != nil || !inserted {
		t.Fatalf("import: inserted=%v err=%v", inserted, err)
	}
	must(t, "interval", demoEnergy.PutInterval(energy.Interval{TenantSlug: "demo", StartsAt: now.Truncate(15 * time.Minute), Duration: 15 * time.Minute, ImportKWh: 0.4, AverageKW: 1.6, Quality: energy.QualityMeasured, Source: "home-assistant"}))
	completed := now.Add(-24 * time.Hour)
	must(t, "maintenance", demoEnergy.UpsertMaintenance(energy.MaintenancePlan{ID: "maintenance-demo-pv", TenantSlug: "demo", AssetID: pvID, Title: "PV-Sichtprüfung", IntervalMonths: 12, LastCompletedAt: &completed, NextDueAt: now.Add(14 * 24 * time.Hour), ContactID: "contact-1", Active: true}))
	must(t, "maintenance 2", demoEnergy.UpsertMaintenance(energy.MaintenancePlan{ID: "maintenance-demo-ev", TenantSlug: "demo", AssetID: energy.StableAssetID("demo", "ev"), Title: "Wallbox-Check", IntervalMonths: 24, NextDueAt: now.Add(60 * 24 * time.Hour), Active: false}))
	for index, version := range []string{"draft-2027-v1", "draft-2027-v2"} {
		must(t, "tariff", demoEnergy.SaveTariffAssessment(energy.TariffAssessment{
			ID: "tariff-" + version, TenantSlug: "demo", AssessmentMonth: "2026-07", ProfileID: "at-grid-power", ProfileVersion: version, ProfileStatus: "draft",
			SourceURL: "https://example.invalid/rules", PeakKW: 8.25 + float64(index), BilledKW: 8.25 + float64(index), AnnualPowerEUR: 99 + float64(index),
			DataQuality: energy.QualityMeasured, CreatedAt: now.Add(time.Duration(index) * time.Minute),
		}))
	}
	appointment := now.Add(48 * time.Hour)
	beforeFrom := now.AddDate(0, -1, 0)
	beforeTo := beforeFrom.Add(7 * 24 * time.Hour)
	beforePeak := 9.4
	must(t, "measure", demoEnergy.UpsertMeasure(energy.Measure{
		ID: "measure-demo", TenantSlug: "demo", IssueID: "issue-a", RecommendationID: "peak", Title: "Lastspitze glätten", Status: energy.MeasureCompleted,
		ContactID: "contact-1", SharedFields: []string{"measurements", "inventory"}, OfferNote: "Angebot geprüft", AppointmentAt: &appointment,
		WorkNote: "Wallbox begrenzt", CompletedAt: &now, EvidenceNote: "Messung geprüft", BeforeFrom: &beforeFrom, BeforeTo: &beforeTo,
		BeforePeakKW: &beforePeak, BeforeQuality: energy.QualityMeasured, AfterQuality: energy.QualityEstimated,
	}))
	must(t, "measure 2", demoEnergy.UpsertMeasure(energy.Measure{ID: "measure-demo-2", TenantSlug: "demo", IssueID: "issue-b", RecommendationID: "other", Title: "Entwurf"}))

	// The home onboarding chain: reserve, confirm, activate (which mints a
	// fifth tenant identity), pair a connector, and receive readings.
	reservations := store.NewSQLHomeReservationStore(src.lanes)
	if _, err := reservations.Reserve(store.HomeReservation{Slug: "stadtpark-home", HouseholdName: "Zuhause am Stadtpark", OwnerEmail: "owner@example.com", AuthorizationConfirmed: true}, now); err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if _, ok, err := reservations.Confirm("stadtpark-home", "owner@example.com", now.Add(time.Minute)); err != nil || !ok {
		t.Fatalf("confirm ok=%v err=%v", ok, err)
	}
	if _, err := reservations.Reserve(store.HomeReservation{Slug: "pending-home", HouseholdName: "Noch offen", OwnerEmail: "resident@example.com", AuthorizationConfirmed: true}, now); err != nil {
		t.Fatalf("reserve pending: %v", err)
	}
	portals := store.NewSQLHomePortalStore(src.lanes)
	if _, _, err := portals.Activate("stadtpark-home", "owner@example.com", now.Add(2*time.Minute)); err != nil {
		t.Fatalf("activate: %v", err)
	}
	connectors := store.NewSQLHomeConnectorStore(src.lanes)
	pairing := bytes.Repeat([]byte{1, 2, 3, 4}, 8)
	credential := bytes.Repeat([]byte{9, 8, 7, 6}, 8)
	if _, err := connectors.StartPairing("stadtpark-home", pairing, now.Add(10*time.Minute), now); err != nil {
		t.Fatalf("pairing: %v", err)
	}
	if _, ok, err := connectors.ExchangePairing(pairing, credential, store.HomeConnectorHeartbeat{ConnectorVersion: "0.83.0", HomeAssistantVersion: "2026.8.1", EntityCount: 321}, now.Add(time.Minute)); err != nil || !ok {
		t.Fatalf("exchange ok=%v err=%v", ok, err)
	}
	readings := store.NewSQLHomeConnectorReadingStore(src.lanes)
	must(t, "readings", readings.Upsert("stadtpark-home", []store.HomeConnectorReading{
		{EntityID: "sensor.grid_power", State: "1.234", DisplayName: "Netz", Unit: "kW", DeviceClass: "power", StateClass: "measurement", LastUpdated: now.Add(-time.Minute)},
		{EntityID: "sensor.pv_energy", State: "4567.8", DisplayName: "PV", Unit: "kWh", DeviceClass: "energy", StateClass: "total_increasing", LastUpdated: now.Add(-2 * time.Minute)},
	}, now))

	// Refresh the identity map: activation minted "stadtpark-home".
	identities, err = store.EnsureTenantIdentities(ctx, src.db, qaTenants)
	must(t, "tenants after activation", err)
	src.tenants = identities
	// Organisation-scoped tables (HAUSV-600): keyed by org_key, no tenant_id.
	orgKey := "musterstadt"
	intake := store.BindIntakeRepository(src.db, orgKey)
	must(t, "intake", intake.Create(ctx, store.IntakeItem{
		ID: "in-rt-1", Organisation: orgKey, TenantSlug: "demo", Unit: "Top 1", Source: store.IntakeSourceEmail,
		FromName: "Rita Bewohnerin", FromEmail: "resident@example.com", Subject: "Wasserfleck im Bad", Body: "Seit gestern feucht.",
		ReceivedAt: now, DueAt: now.Add(24 * time.Hour), Status: store.IntakeStatusOpen, CreatedAt: now, UpdatedAt: now,
	}))
	must(t, "organisation", store.BindOrganisationRepository(src.db, orgKey).Save(ctx, store.Organisation{
		Key: orgKey, Name: "Hausverwaltung Musterstadt", ContactName: "Vera Verwalter",
		ContactEmail: "buero@musterstadt.example", ContactPhone: "+43 316 123456",
		Houses: []string{"demo", "haus-a"}, UpdatedAt: now,
	}))
	must(t, "organisation member", store.BindOrganisationMemberRepository(src.db, orgKey).Save(ctx, store.OrganisationMember{
		Email: "sachbearbeiter@example.com", Role: store.OrganisationRoleClerk,
		Granted: map[string]string{"demo": "bewohner", "haus-a": ""}, CreatedAt: now,
	}))
	must(t, "intake mail seen", store.BindIntakeMailSeenRepository(src.db, orgKey).Record(ctx, "<seed-mail@example.com>", "in-0001"))
	must(t, "org settings", store.BindOrgSettingsRepository(src.db, orgKey).Save(ctx, store.OrgSettings{
		Organisation: orgKey, Name: "Hausverwaltung Musterstadt", TrustLevels: map[string]string{"beleg": "auto", "reparatur": "propose"},
		AutoThreshold: 0.9, AutoEnabled: true, UpdatedAt: now,
	}))
	must(t, "textbaustein", store.BindTextbausteinRepository(src.db, orgKey).Upsert(ctx, store.Textbaustein{
		Key: "reparatur-beauftragt", Organisation: orgKey, Category: "reparatur", Title: "Reparatur – Handwerker beauftragt",
		Body: "Sehr geehrte{{Anrede}} {{Name}}, wir haben {{Handwerker}} beauftragt.", Placeholders: []string{"Anrede", "Name", "Handwerker"}, Active: true,
	}))

	return src
}

// listTables reads the governed tables straight from the SQLite catalog, so
// the seed coverage check and the row comparison cannot be narrower than the
// schema.
func listTables(t *testing.T, database *sql.DB) []string {
	t.Helper()
	rows, err := database.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name <> 'schema_migrations' ORDER BY name`)
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		out = append(out, name)
	}
	return out
}

func columnsOf(t *testing.T, database *sql.DB, table string) []string {
	t.Helper()
	rows, err := database.Query(`PRAGMA table_info("` + table + `")`)
	if err != nil {
		t.Fatalf("columns of %s: %v", table, err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var (
			cid, notNull, pk int
			name, typ        string
			dflt             sql.NullString
		)
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk); err != nil {
			t.Fatal(err)
		}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func assertEveryTableSeeded(t *testing.T, database *sql.DB) map[string]int64 {
	t.Helper()
	counts := map[string]int64{}
	var empty []string
	for _, table := range listTables(t, database) {
		var n int64
		if err := database.QueryRow(`SELECT count(*) FROM "` + table + `"`).Scan(&n); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		counts[table] = n
		if n == 0 {
			empty = append(empty, table)
		}
	}
	if len(empty) > 0 {
		t.Fatalf("the full seed leaves tables empty, so the round trip would not prove them: %s", strings.Join(empty, ", "))
	}
	return counts
}

// normalise is the test's OWN canonical form for one raw driver value. It is
// deliberately not the mover's: SQLite hands back int64 for a boolean column
// and PostgreSQL hands back bool; both become "1"/"0" here. Everything else is
// spelled by value.
func normalise(v any) string {
	switch x := v.(type) {
	case nil:
		return "<null>"
	case bool:
		if x {
			return "1"
		}
		return "0"
	case int64:
		return strconv.FormatInt(x, 10)
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case string:
		return "'" + x + "'"
	case []byte:
		return "x" + hex.EncodeToString(x)
	}
	return fmt.Sprintf("?%T", v)
}

type rowReader interface {
	Query(query string, args ...any) (*sql.Rows, error)
}

// snapshot reads one table as sorted canonical rows.
func snapshot(t *testing.T, q rowReader, table string, columns []string) []string {
	t.Helper()
	quoted := make([]string, len(columns))
	for i, c := range columns {
		quoted[i] = `"` + c + `"`
	}
	rows, err := q.Query(`SELECT ` + strings.Join(quoted, ", ") + ` FROM "` + table + `"`)
	if err != nil {
		t.Fatalf("snapshot %s: %v", table, err)
	}
	defer rows.Close()
	raw := make([]any, len(columns))
	ptrs := make([]any, len(columns))
	for i := range raw {
		ptrs[i] = &raw[i]
	}
	var out []string
	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			t.Fatalf("scan %s: %v", table, err)
		}
		parts := make([]string, len(columns))
		for i, c := range columns {
			parts[i] = c + "=" + normalise(raw[i])
		}
		out = append(out, strings.Join(parts, " | "))
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate %s: %v", table, err)
	}
	sort.Strings(out)
	return out
}

// assertTablesIdentical compares every table value by value between the SQLite
// source and the PostgreSQL target, read through the maintenance lane.
func assertTablesIdentical(t *testing.T, src *source, tgt *target) (tables, rows int) {
	t.Helper()
	for _, table := range listTables(t, src.db) {
		columns := columnsOf(t, src.db, table)
		want := snapshot(t, src.db, table, columns)
		got := snapshot(t, tgt.lane, table, columns)
		if len(want) != len(got) {
			t.Errorf("%s: %d rows in sqlite, %d in postgres", table, len(want), len(got))
			continue
		}
		for i := range want {
			if want[i] != got[i] {
				t.Errorf("%s row %d differs\n  sqlite:   %s\n  postgres: %s", table, i, want[i], got[i])
				break
			}
		}
		tables++
		rows += len(want)
	}
	return tables, rows
}

func move(t *testing.T, src *source, tgt *target, opts dbmove.Options) (*dbmove.Report, string) {
	t.Helper()
	var out bytes.Buffer
	opts.Out = &out
	report, err := dbmove.Move(context.Background(), src.db, tgt.lane, opts)
	if err != nil {
		t.Fatalf("move: %v\n%s", err, out.String())
	}
	return report, out.String()
}

func TestRoundTripSmallFixture(t *testing.T) {
	requirePostgres(t)
	src := seedSmall(t)
	tgt := openTarget(t)

	report, out := move(t, src, tgt, dbmove.Options{})
	if !report.Committed || !report.AllMatch() {
		t.Fatalf("report: committed=%v allmatch=%v\n%s", report.Committed, report.AllMatch(), out)
	}
	tables, rows := assertTablesIdentical(t, src, tgt)
	if tables != len(dbmove.TableOrder) {
		t.Fatalf("compared %d tables, plan has %d", tables, len(dbmove.TableOrder))
	}
	if rows != int(report.TotalSourceRows()) || rows < 6 {
		t.Fatalf("compared %d rows, report says %d", rows, report.TotalSourceRows())
	}
	t.Logf("small fixture: %d tables, %d rows moved and compared value by value", tables, rows)
}

func TestRoundTripFullSeedMovesEveryTableValueForValue(t *testing.T) {
	requirePostgres(t)
	src := seedFull(t)
	sourceCounts := assertEveryTableSeeded(t, src.db)
	tgt := openTarget(t)

	report, out := move(t, src, tgt, dbmove.Options{})
	if !report.Committed || !report.AllMatch() {
		t.Fatalf("report: committed=%v allmatch=%v\n%s", report.Committed, report.AllMatch(), out)
	}
	if len(report.Tables) != len(dbmove.TableOrder) {
		t.Fatalf("report covers %d tables, plan has %d", len(report.Tables), len(dbmove.TableOrder))
	}
	for _, table := range report.Tables {
		if table.SourceRows != sourceCounts[table.Name] || table.TargetRows != sourceCounts[table.Name] {
			t.Errorf("%s: seed has %d rows, report says source %d target %d", table.Name, sourceCounts[table.Name], table.SourceRows, table.TargetRows)
		}
		// A pending home reservation legitimately precedes its tenant identity
		// (store.tenantIDTables marks home_reservations preTenant), so the seed
		// carries exactly one row with a NULL tenant_id — and it must cross as
		// NULL, counted and reported, not invented or dropped.
		wantOrphans := int64(0)
		if table.Name == "home_reservations" {
			wantOrphans = 1
		}
		if table.NullTenantIDs != wantOrphans {
			t.Errorf("%s: %d rows without tenant_id, want %d", table.Name, table.NullTenantIDs, wantOrphans)
		}
	}
	if !strings.Contains(out, "1 rows without tenant_id") {
		t.Errorf("report does not flag the orphan row\n%s", out)
	}
	// The report must have seen the conversions this schema needs, by name.
	joined := out
	for _, want := range []string{
		"contacts.active: integer -> boolean",
		"persons.deactivated: integer -> boolean",
		"house_memberships.directory_opt_in: integer -> boolean",
		"home_profiles.onboarding_complete: integer -> boolean",
		"energy_imports.payload: blob -> bytea",
		"home_connectors.credential_hash: blob -> bytea",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("report does not name the conversion %q\n%s", want, joined)
		}
	}
	tables, rows := assertTablesIdentical(t, src, tgt)
	if rows != int(report.TotalSourceRows()) {
		t.Fatalf("compared %d rows, report says %d", rows, report.TotalSourceRows())
	}
	t.Logf("full seed: %d tables, %d rows moved and compared value by value\n%s", tables, rows, out)
}

func TestMoveRefusesNonEmptyTargetAndForceWipesIt(t *testing.T) {
	requirePostgres(t)
	src := seedSmall(t)
	tgt := openTarget(t)

	// A target the application has already booted on: one minted tenant row
	// with an id the source does not know.
	if _, err := store.EnsureTenantIdentities(context.Background(), tgt.db, []store.TenantIdentity{{Slug: "demo", Name: "Booted"}}); err != nil {
		t.Fatalf("pre-populate target: %v", err)
	}
	if _, err := tgt.lane.Exec(`INSERT INTO app_meta(key, value) VALUES('stray', 'row')`); err != nil {
		t.Fatalf("stray row: %v", err)
	}

	var out bytes.Buffer
	report, err := dbmove.Move(context.Background(), src.db, tgt.lane, dbmove.Options{Out: &out})
	if !errors.Is(err, dbmove.ErrTargetNotEmpty) {
		t.Fatalf("expected refusal, got err=%v\n%s", err, out.String())
	}
	if report.TargetRowsBefore["tenant"] != 1 || report.TargetRowsBefore["app_meta"] != 1 {
		t.Fatalf("refusal must say what it found: %v", report.TargetRowsBefore)
	}
	if !strings.Contains(out.String(), "target is NOT empty") {
		t.Fatalf("report does not print the finding:\n%s", out.String())
	}
	// Refusal left the target as it was.
	var strayCount int
	if err := tgt.lane.QueryRow(`SELECT count(*) FROM app_meta`).Scan(&strayCount); err != nil || strayCount != 1 {
		t.Fatalf("refusal changed the target: count=%d err=%v", strayCount, err)
	}
	var contacts int
	if err := tgt.lane.QueryRow(`SELECT count(*) FROM contacts`).Scan(&contacts); err != nil || contacts != 0 {
		t.Fatalf("refusal loaded rows: contacts=%d err=%v", contacts, err)
	}

	// --force wipes and loads: the booted tenant id is gone, the source's is in.
	report, _ = move(t, src, tgt, dbmove.Options{Force: true})
	if !report.Committed {
		t.Fatal("force run did not commit")
	}
	if err := tgt.lane.QueryRow(`SELECT count(*) FROM app_meta WHERE key='stray'`).Scan(&strayCount); err != nil || strayCount != 0 {
		t.Fatalf("stray row survived --force: count=%d err=%v", strayCount, err)
	}
	var tenantID string
	if err := tgt.lane.QueryRow(`SELECT tenant_id FROM tenant WHERE slug='demo'`).Scan(&tenantID); err != nil {
		t.Fatalf("tenant after force: %v", err)
	}
	if tenantID != src.tenants["demo"].ID {
		t.Fatalf("tenant id after --force = %s, want the source's %s", tenantID, src.tenants["demo"].ID)
	}
	assertTablesIdentical(t, src, tgt)
}

func TestDryRunLoadsVerifiesAndLeavesTargetEmpty(t *testing.T) {
	requirePostgres(t)
	src := seedSmall(t)
	tgt := openTarget(t)
	report, out := move(t, src, tgt, dbmove.Options{DryRun: true})
	if report.Committed || !report.DryRun || !report.AllMatch() {
		t.Fatalf("dry run: committed=%v dryrun=%v allmatch=%v\n%s", report.Committed, report.DryRun, report.AllMatch(), out)
	}
	if !strings.Contains(out, "ROLLED BACK") {
		t.Fatalf("dry run report does not say it rolled back:\n%s", out)
	}
	for _, table := range dbmove.TableOrder {
		var n int
		if err := tgt.lane.QueryRow(`SELECT count(*) FROM "` + table + `"`).Scan(&n); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if n != 0 {
			t.Fatalf("dry run left %d rows in %s", n, table)
		}
	}
	// And a real run afterwards succeeds against the still-empty target.
	report, _ = move(t, src, tgt, dbmove.Options{})
	if !report.Committed {
		t.Fatal("real run after dry run did not commit")
	}
}

// The hash must see a changed VALUE, not just a changed count — otherwise it is
// the count check wearing a longer name. Flip one boolean and one text on the
// target after a clean move and demand that Verify names exactly those tables.
func TestVerifySeesAChangedValueNotJustACount(t *testing.T) {
	requirePostgres(t)
	src := seedFull(t)
	tgt := openTarget(t)
	move(t, src, tgt, dbmove.Options{})

	var out bytes.Buffer
	if _, err := dbmove.Verify(context.Background(), src.db, tgt.lane, &out); err != nil {
		t.Fatalf("verify after a clean move: %v\n%s", err, out.String())
	}

	if _, err := tgt.lane.Exec(`UPDATE contacts SET active = NOT active WHERE id = (SELECT id FROM contacts ORDER BY id LIMIT 1)`); err != nil {
		t.Fatalf("flip contact: %v", err)
	}
	if _, err := tgt.lane.Exec(`UPDATE persons SET last_name = last_name || '!' WHERE email = 'owner@example.com'`); err != nil {
		t.Fatalf("edit person: %v", err)
	}
	out.Reset()
	report, err := dbmove.Verify(context.Background(), src.db, tgt.lane, &out)
	if !errors.Is(err, dbmove.ErrVerifyMismatch) {
		t.Fatalf("verify must report the mismatch, got %v\n%s", err, out.String())
	}
	var mismatched []string
	for _, table := range report.Tables {
		if table.SourceRows != table.TargetRows {
			t.Errorf("%s: counts differ (%d vs %d) although only values changed", table.Name, table.SourceRows, table.TargetRows)
		}
		if !table.Match {
			mismatched = append(mismatched, table.Name)
		}
	}
	sort.Strings(mismatched)
	if want := []string{"contacts", "persons"}; !reflect.DeepEqual(mismatched, want) {
		t.Fatalf("mismatched tables = %v, want %v\n%s", mismatched, want, out.String())
	}
}

// The application's own read paths, bound over the moved tenant on PostgreSQL,
// must return what they return on the SQLite side. This is the half of the
// proof that a raw comparison cannot give: it goes through the tenant lanes,
// so on PostgreSQL every read here is filtered by row-level security on the
// tenant_id the mover wrote.
func TestReadPathsAgreeAfterMove(t *testing.T) {
	requirePostgres(t)
	src := seedFull(t)
	tgt := openTarget(t)
	move(t, src, tgt, dbmove.Options{})

	demo := src.tenants["demo"].Ref()
	hausA := src.tenants["haus-a"].Ref()
	files := t.TempDir()

	type probe struct {
		name string
		read func(lanes *store.TenantDB) (any, error)
	}
	probes := []probe{
		{"contacts.List", func(l *store.TenantDB) (any, error) {
			r, _ := store.BindContactBookRepository(store.NewSQLContactBookStore(l), demo)
			return r.List(true), nil
		}},
		{"contacts.List(haus-a)", func(l *store.TenantDB) (any, error) {
			r, _ := store.BindContactBookRepository(store.NewSQLContactBookStore(l), hausA)
			return r.List(true), nil
		}},
		{"units.List", func(l *store.TenantDB) (any, error) {
			r, _ := store.BindUnitRepository(store.NewSQLUnitStore(l), demo)
			return r.List(), nil
		}},
		{"units.UnitsForEmail", func(l *store.TenantDB) (any, error) {
			r, _ := store.BindUnitRepository(store.NewSQLUnitStore(l), demo)
			return r.UnitsForEmail("owner@example.com"), nil
		}},
		{"unitPayments.List", func(l *store.TenantDB) (any, error) {
			r, _ := store.BindUnitPaymentStatusRepository(store.NewSQLUnitPaymentStatusStore(l), demo)
			return r.List(), nil
		}},
		{"announcements.List", func(l *store.TenantDB) (any, error) {
			r, _ := store.BindAnnouncementRepository(store.NewSQLAnnouncementStore(l), demo)
			return r.List(), nil
		}},
		{"announcementReads.LastSeen", func(l *store.TenantDB) (any, error) {
			r, _ := store.BindAnnouncementReadRepository(store.NewSQLAnnouncementReadStore(l), demo)
			return r.LastSeen("owner@example.com"), nil
		}},
		{"events.List", func(l *store.TenantDB) (any, error) {
			r, _ := store.BindEventRepository(store.NewSQLEventStore(l), demo)
			return r.List(), nil
		}},
		{"handovers.List", func(l *store.TenantDB) (any, error) {
			r, _ := store.BindHandoverRepository(store.NewSQLHandoverStore(l), demo)
			return r.List(), nil
		}},
		{"documents.List", func(l *store.TenantDB) (any, error) {
			r, _ := store.BindDocumentRepository(store.NewSQLDocumentStore(l, files), demo)
			return r.List(), nil
		}},
		{"attachments.ListEntity", func(l *store.TenantDB) (any, error) {
			r, _ := store.BindAttachmentRepository(store.NewSQLAttachmentStore(l, files), demo)
			return r.ListEntity("issue", "issue-1"), nil
		}},
		{"votes.List", func(l *store.TenantDB) (any, error) {
			r, _ := store.BindVoteRepository(store.NewSQLVoteStore(l), demo)
			return r.List(), nil
		}},
		{"issues.List", func(l *store.TenantDB) (any, error) {
			r, _ := store.BindIssueRepository(store.NewSQLIssueStore(l, files), demo)
			return r.List(), nil
		}},
		{"identity.List", func(l *store.TenantDB) (any, error) {
			return store.NewSQLIdentityStore(l).List(), nil
		}},
		{"identity.Get", func(l *store.TenantDB) (any, error) {
			p, ok := store.NewSQLIdentityStore(l).Get("multi@example.com")
			return []any{p, ok}, nil
		}},
		{"identity.MembershipsForPerson", func(l *store.TenantDB) (any, error) {
			s := store.NewSQLIdentityStore(l)
			person, ok := s.PersonByEmail("owner@example.com")
			if !ok {
				return nil, errors.New("owner not found")
			}
			return s.MembershipsForPerson(person.ID), nil
		}},
		{"activity.Get", func(l *store.TenantDB) (any, error) {
			rec, ok := store.NewSQLActivityStore(l).Get("admin@example.com")
			return []any{rec, ok}, nil
		}},
		{"overlays.Get", func(l *store.TenantDB) (any, error) {
			o, ok := store.NewSQLProfileOverlayStore(l).Get("owner@example.com")
			return []any{o, ok}, nil
		}},
		{"prefs.Get", func(l *store.TenantDB) (any, error) {
			return []any{store.NewSQLNotificationPrefStore(l).Get("owner@example.com"), store.NewSQLNotificationPrefStore(l).Get("resident@example.com")}, nil
		}},
		{"telegram", func(l *store.TenantDB) (any, error) {
			s := store.NewSQLTelegramStore(l)
			return []any{s.Offset(), s.Links(), s.ChatsByEmail("owner@example.com")}, nil
		}},
		{"energy.Profile", func(l *store.TenantDB) (any, error) {
			p, ok, err := energy.NewSQLStore(l).ForTenant(demo).Profile("demo")
			return []any{p, ok}, err
		}},
		{"energy.ListAssets", func(l *store.TenantDB) (any, error) {
			return energy.NewSQLStore(l).ForTenant(demo).ListAssets("demo")
		}},
		{"energy.ListMappings", func(l *store.TenantDB) (any, error) {
			return energy.NewSQLStore(l).ForTenant(demo).ListMappings("demo")
		}},
		{"energy.ListIntervals", func(l *store.TenantDB) (any, error) {
			return energy.NewSQLStore(l).ForTenant(demo).ListIntervals("demo", time.Time{}, time.Time{})
		}},
		{"energy.ListImportsForExport", func(l *store.TenantDB) (any, error) {
			return energy.NewSQLStore(l).ForTenant(demo).ListImportsForExport("demo")
		}},
		{"energy.ListMaintenance", func(l *store.TenantDB) (any, error) {
			return energy.NewSQLStore(l).ForTenant(demo).ListMaintenance("demo")
		}},
		{"energy.ListTariffAssessments", func(l *store.TenantDB) (any, error) {
			return energy.NewSQLStore(l).ForTenant(demo).ListTariffAssessments("demo")
		}},
		{"energy.ListMeasures", func(l *store.TenantDB) (any, error) {
			return energy.NewSQLStore(l).ForTenant(demo).ListMeasures("demo")
		}},
		{"energy.ListAssets(cockpit)", func(l *store.TenantDB) (any, error) {
			return energy.NewSQLStore(l).ForTenant(src.tenants["cockpit"].Ref()).ListAssets("cockpit")
		}},
		{"homeReservations.Get", func(l *store.TenantDB) (any, error) {
			r, ok, err := store.NewSQLHomeReservationStore(l).Get("stadtpark-home")
			return []any{r, ok}, err
		}},
		{"homePortals.ListByOwner", func(l *store.TenantDB) (any, error) {
			return store.NewSQLHomePortalStore(l).ListByOwner("owner@example.com")
		}},
		{"homeConnectors.Get", func(l *store.TenantDB) (any, error) {
			c, ok, err := store.NewSQLHomeConnectorStore(l).Get("stadtpark-home")
			return []any{c, ok}, err
		}},
		{"homeConnectorReadings.List", func(l *store.TenantDB) (any, error) {
			return store.NewSQLHomeConnectorReadingStore(l).List("stadtpark-home")
		}},
		{"annualStatementConsumption.ConsumptionVector", func(l *store.TenantDB) (any, error) {
			r, _ := store.BindAnnualStatementConsumptionRepository(store.NewSQLAnnualStatementConsumptionStore(l), demo)
			return r.ConsumptionVector(store.AnnualStatementPeriod{Year: 2026, StartsOn: "2026-01-01", EndsOn: "2026-12-31"}, "heizung", []string{"top-1"}, time.UTC)
		}},
	}
	compared := 0
	for _, p := range probes {
		want, err := p.read(src.lanes)
		if err != nil {
			t.Fatalf("%s on sqlite: %v", p.name, err)
		}
		got, err := p.read(tgt.lanes)
		if err != nil {
			t.Fatalf("%s on postgres: %v", p.name, err)
		}
		wantJSON, err := json.Marshal(want)
		if err != nil {
			t.Fatalf("%s: marshal sqlite result: %v", p.name, err)
		}
		gotJSON, err := json.Marshal(got)
		if err != nil {
			t.Fatalf("%s: marshal postgres result: %v", p.name, err)
		}
		if string(wantJSON) == "null" || string(wantJSON) == "[]" || string(wantJSON) == "{}" {
			t.Fatalf("%s: the sqlite side returned nothing (%s) — the probe proves nothing", p.name, wantJSON)
		}
		if !bytes.Equal(wantJSON, gotJSON) {
			t.Errorf("%s differs after the move\n  sqlite:   %s\n  postgres: %s", p.name, wantJSON, gotJSON)
		}
		compared++
	}
	if compared < len(probes) {
		t.Fatalf("compared %d of %d probes", compared, len(probes))
	}
	t.Logf("%d read paths agree between the SQLite source and the moved PostgreSQL tenant", compared)
}

// The production condition the mover is built for: a target whose row-level
// security is FAIL-CLOSED, where an unscoped session sees and writes nothing
// and only the declared maintenance lane (hausv.cross_tenant = 'on') crosses
// tenants. The repository's own migrations are not touched here — the policy
// is rewritten on this test's isolated schema in the shape the flip takes — so
// this proves the mover against the contract rather than against today's
// still-open policy.

// installFailClosedPolicy re-keys every governed table's policy so that ONLY a
// session carrying hausv.cross_tenant='on' or the matching hausv.tenant_id can see
// or write rows — the posture the RLS flip installs. Shared by the tests below.
func installFailClosedPolicy(t *testing.T, tgt *target) {
	t.Helper()
	if _, err := tgt.db.Exec(`
DO $$
DECLARE
    table_name text;
BEGIN
    FOR table_name IN
        SELECT c.table_name
        FROM information_schema.columns c
        JOIN pg_catalog.pg_tables t
          ON t.schemaname = c.table_schema AND t.tablename = c.table_name
        WHERE c.table_schema = current_schema()
          AND c.column_name = 'tenant_id'
          AND c.table_name <> 'tenant'
    LOOP
        EXECUTE format('DROP POLICY IF EXISTS tenant_isolation ON %I', table_name);
        EXECUTE format(
            'CREATE POLICY tenant_isolation ON %I '
            'USING (current_setting(''hausv.cross_tenant'', true) = ''on'' '
            '       OR tenant_id = current_setting(''hausv.tenant_id'', true)) '
            'WITH CHECK (current_setting(''hausv.cross_tenant'', true) = ''on'' '
            '            OR tenant_id = current_setting(''hausv.tenant_id'', true))',
            table_name
        );
    END LOOP;
END;
$$`); err != nil {
		t.Fatalf("install fail-closed policy: %v", err)
	}

}

func TestMoveWorksThroughTheMaintenanceLaneWhenRLSIsFailClosed(t *testing.T) {
	requirePostgres(t)
	src := seedFull(t)
	tgt := openTarget(t)
	installFailClosedPolicy(t, tgt)

	report, out := move(t, src, tgt, dbmove.Options{})
	if !report.Committed || !report.AllMatch() {
		t.Fatalf("move under fail-closed RLS: committed=%v allmatch=%v\n%s", report.Committed, report.AllMatch(), out)
	}
	// The plain pool, which carries neither setting, must now see nothing in a
	// tenant table — that is what "fail-closed" means, and it is why every
	// count and every insert of the mover goes through the lane.
	var plain, viaLane int
	if err := tgt.db.QueryRow(`SELECT count(*) FROM contacts`).Scan(&plain); err != nil {
		t.Fatalf("plain count: %v", err)
	}
	if err := tgt.lane.QueryRow(`SELECT count(*) FROM contacts`).Scan(&viaLane); err != nil {
		t.Fatalf("lane count: %v", err)
	}
	if plain != 0 || viaLane != 4 {
		t.Fatalf("contacts: plain pool sees %d, maintenance lane sees %d; want 0 and 4", plain, viaLane)
	}
	// And a tenant lane sees exactly its own tenant's rows through the
	// application's read path.
	demo := src.tenants["demo"].Ref()
	repo, _ := store.BindContactBookRepository(store.NewSQLContactBookStore(tgt.lanes), demo)
	if got := repo.List(true); len(got) != 2 {
		t.Fatalf("demo contacts through the tenant lane = %d, want 2", len(got))
	}
	// A second run without --force is refused: the emptiness check on the lane
	// sees the rows a plain session would have missed.
	var refused bytes.Buffer
	if _, err := dbmove.Move(context.Background(), src.db, tgt.lane, dbmove.Options{Out: &refused}); !errors.Is(err, dbmove.ErrTargetNotEmpty) {
		t.Fatalf("under fail-closed RLS the emptiness check must still see the moved rows, got %v\n%s", err, refused.String())
	}
	assertTablesIdentical(t, src, tgt)
}

// A verification mismatch must roll the whole load back: an operator who sees
// ErrVerifyMismatch must be able to trust that the target is exactly as empty as
// it was, not half-loaded. Proven here by making one row impossible to load
// faithfully — a BEFORE INSERT trigger that rewrites a value on the way in, so
// the target's content hash cannot match the source's — and asserting both the
// error and the zero row count afterwards. Until this test existed the property
// was proven only by an ad-hoc probe during review.
func TestVerifyMismatchRollsTheWholeLoadBack(t *testing.T) {
	requirePostgres(t)
	src := seedFull(t)
	tgt := openTarget(t)
	if _, err := tgt.db.Exec(`
CREATE OR REPLACE FUNCTION dbmove_tamper() RETURNS trigger AS $$
BEGIN
    NEW.last_name := NEW.last_name || '-tampered';
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER dbmove_tamper BEFORE INSERT ON persons
    FOR EACH ROW EXECUTE FUNCTION dbmove_tamper()`); err != nil {
		t.Fatalf("install tamper trigger: %v", err)
	}
	// Call Move directly: the shared helper fatals on any error, and the error IS
	// the assertion here.
	var out bytes.Buffer
	report, err := dbmove.Move(context.Background(), src.db, tgt.lane, dbmove.Options{Out: &out})
	if !errors.Is(err, dbmove.ErrVerifyMismatch) {
		t.Fatalf("err = %v, want ErrVerifyMismatch\n%s", err, out.String())
	}
	if report != nil && report.Committed {
		t.Fatalf("a load whose content cannot match the source COMMITTED:\n%s", out.String())
	}
	for _, table := range []string{"persons", "contacts", "tenant"} {
		var n int
		if err := tgt.lane.QueryRow(`SELECT count(*) FROM ` + table).Scan(&n); err != nil {
			t.Fatalf("count %s after rollback: %v", table, err)
		}
		if n != 0 {
			t.Errorf("%s holds %d rows after a rolled-back load, want 0", table, n)
		}
	}
}

// --force must work under the fail-closed posture too: the wipe and the reload both
// happen inside the maintenance lane, and a plain session must still see nothing
// afterwards. This is the operator's recovery path if a first attempt is found wrong
// after commit, so it has to be proven under the policy that will actually be live.
func TestForceWipesAndReloadsUnderFailClosedRLS(t *testing.T) {
	requirePostgres(t)
	src := seedFull(t)
	tgt := openTarget(t)
	installFailClosedPolicy(t, tgt)

	first, out := move(t, src, tgt, dbmove.Options{})
	if !first.Committed || !first.AllMatch() {
		t.Fatalf("first move: committed=%v allmatch=%v\n%s", first.Committed, first.AllMatch(), out)
	}
	if _, err := tgt.lane.Exec(`UPDATE contacts SET active = NOT active`); err != nil {
		t.Fatalf("tamper contacts: %v", err)
	}
	second, out := move(t, src, tgt, dbmove.Options{Force: true})
	if !second.Committed || !second.AllMatch() {
		t.Fatalf("forced reload under fail-closed: committed=%v allmatch=%v\n%s", second.Committed, second.AllMatch(), out)
	}
	var plain, viaLane int
	if err := tgt.db.QueryRow(`SELECT count(*) FROM contacts`).Scan(&plain); err != nil {
		t.Fatalf("plain count: %v", err)
	}
	if err := tgt.lane.QueryRow(`SELECT count(*) FROM contacts`).Scan(&viaLane); err != nil {
		t.Fatalf("lane count: %v", err)
	}
	if plain != 0 || viaLane != 4 {
		t.Fatalf("after forced reload: plain pool sees %d, lane sees %d; want 0 and 4", plain, viaLane)
	}
}
