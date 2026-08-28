package store

import (
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// writersCoveredElsewhere names the tenant-scoped tables this test does NOT
// write to, and where their writers are covered instead. It is the complement of
// what the loop below checks, and the two together must cover tenantIDTables
// exactly.
//
// A hand-written list of CHECKED tables could be shrunk silently: deleting a
// table from it left the suite green, so breaking the events writer's
// tenant_slug binding and deleting "events" in the same edit passed both the
// SQLite and the PostgreSQL suites.
//
// A list of EXCUSED tables is not enough on its own either. Deleting a writer
// call from the fixture and adding its table HERE also left the suite green —
// the loop only rejects an excuse for a table that has rows, and a table nobody
// writes to has none. So the list is checked against the SOURCE as well, by
// TestWritersCoveredElsewhereAreNotWrittenHere: a table this package INSERTs
// into cannot be excused, whatever the fixture does.
var writersCoveredElsewhere = map[string]string{
	"energy_assets":             "internal/energy, TestEveryEnergyWriteRecordsATenantIdentity",
	"energy_entity_mappings":    "internal/energy, TestEveryEnergyWriteRecordsATenantIdentity",
	"energy_imports":            "internal/energy, TestEveryEnergyWriteRecordsATenantIdentity",
	"energy_intervals":          "internal/energy, TestEveryEnergyWriteRecordsATenantIdentity",
	"energy_maintenance_plans":  "internal/energy, TestEveryEnergyWriteRecordsATenantIdentity",
	"energy_measures":           "internal/energy, TestEveryEnergyWriteRecordsATenantIdentity",
	"energy_tariff_assessments": "internal/energy, TestEveryEnergyWriteRecordsATenantIdentity",
	"home_profiles":             "internal/energy owns this table's writer",
	"integration_imports":       "internal/server's import ledger, TestImportLedgerHealsRowsLeftWithoutAnIdentity",
}

// TestWritersCoveredElsewhereAreNotWrittenHere is the closure check that makes
// the excuse list mean something.
//
// "Covered elsewhere" is only true of a table this package never writes. The
// claim is therefore checked against the SQL this package actually issues rather
// than accepted: every tenant-scoped table internal/store INSERTs into must be
// exercised by TestEveryStoreWriteRecordsATenantIdentity, and every table it
// does not must be excused. Removing a writer call from that fixture and adding
// its table to writersCoveredElsewhere used to be a green two-line edit; it now
// fails here.
//
// It caught a live gap on the way in: home_portals was excused as "internal/server
// activates a portal; no store repository writes it", and SQLHomePortalStore.Activate
// has been inserting into it all along.
//
// Where it is blind: it reads INSERT statements as string literals, so a
// statement assembled from a variable at runtime is invisible to it, and it
// proves the excuse is not a lie about THIS package rather than proving the
// coverage it names exists. internal/energy asks the same question of itself.
func TestWritersCoveredElsewhereAreNotWrittenHere(t *testing.T) {
	written := tenantScopedTablesInsertedHere(t)
	if len(written) < 15 {
		t.Fatalf("only %d tenant-scoped INSERT targets were found — the probe is broken, not the package", len(written))
	}
	for _, table := range written {
		if reason, excused := writersCoveredElsewhere[table]; excused {
			t.Errorf("%s is excused as %q but internal/store INSERTs into it — "+
				"exercise its writer in TestEveryStoreWriteRecordsATenantIdentity instead", table, reason)
		}
	}
	writes := map[string]bool{}
	for _, table := range written {
		writes[table] = true
	}
	for table := range writersCoveredElsewhere {
		if !tenantIDTableNames()[table] {
			t.Errorf("%s is excused but is not a tenant-scoped table at all", table)
		}
	}
	for _, table := range tenantIDTables {
		if _, excused := writersCoveredElsewhere[table.name]; !excused && !writes[table.name] {
			t.Errorf("%s is neither written by internal/store nor excused — nothing checks its identity", table.name)
		}
	}
}

func tenantIDTableNames() map[string]bool {
	out := map[string]bool{}
	for _, table := range tenantIDTables {
		out[table.name] = true
	}
	return out
}

// tenantScopedTablesInsertedHere reads the package's own SQL and returns every
// tenant-scoped table it INSERTs into.
func tenantScopedTablesInsertedHere(t *testing.T) []string {
	t.Helper()
	scoped := tenantIDTableNames()
	found := map[string]bool{}
	for _, text := range sqlLiteralsIn(t, ".") {
		for _, match := range insertedTable.FindAllStringSubmatch(text, -1) {
			if scoped[strings.ToLower(match[1])] {
				found[strings.ToLower(match[1])] = true
			}
		}
	}
	out := make([]string, 0, len(found))
	for table := range found {
		out = append(out, table)
	}
	sort.Strings(out)
	return out
}

// TestEveryStoreWriteRecordsATenantIdentity is the contradiction the boot-time
// completeness check needs in order to mean anything.
//
// BackfillTenantIDs proves a database is complete at boot. It says nothing about
// what happens next: before this slice, every INSERT in this package wrote the
// slug and omitted tenant_id, so a database that booted complete drifted back to
// NULL with the first request. This exercises one write through each repository
// and then asks the database whether an identity was recorded — the same
// question the boot check asks, but of rows the running product just created.
//
// It asks two questions of every table, not one. tenant_id must be populated —
// that is the switch working — AND the row must still be addressable by its slug
// column, because tenant_slug is written on 33 INSERTs for exactly one reason:
// the previous release finds rows by it, so the rollback window depends on it.
// Only `announcements` had that second guarantee before, which made it
// toothless: binding "" for tenant_slug in any other writer left the full SQLite
// suite and both PostgreSQL suites green.
//
// Where it is blind: it covers one write per repository, not every code path
// into each table, and it says nothing about internal/energy (covered by
// TestEveryEnergyWriteRecordsATenantIdentity) or internal/server's import
// ledger — those are the entries in writersCoveredElsewhere, and this test
// cannot tell whether the coverage it names is real, only that a claim was made.
// It checks the slug column equals THIS tenant's slug, which catches an empty or
// wrong binding but not a slug that is right and an identity that belongs to a
// different tenant — TestBoundRepositoriesSeparateTenantsWithDistinctIdentities
// is where that is asked. home_reservations is exercised but exempt from the
// identity half: a reservation exists before its house does, so its tenant_id is
// NULL by design until activation.
func TestEveryStoreWriteRecordsATenantIdentity(t *testing.T) {
	database, lanes := testLanes(t)
	tenant := testTenantRef("demo")
	now := time.Now().UTC()
	fileDir := t.TempDir()

	announcements, _ := BindAnnouncementRepository(NewSQLAnnouncementStore(lanes), tenant)
	if _, err := announcements.Create(Announcement{Title: "A", Body: "b"}); err != nil {
		t.Fatalf("announcement: %v", err)
	}
	reads, _ := BindAnnouncementReadRepository(NewSQLAnnouncementReadStore(lanes), tenant)
	if err := reads.MarkSeen("a@example.com", now); err != nil {
		t.Fatalf("announcement read: %v", err)
	}
	contacts, _ := BindContactBookRepository(NewSQLContactBookStore(lanes), tenant)
	if _, _, err := contacts.Upsert(ManagedContact{Kind: "Notdienst", Name: "N", Phone: "1", Active: true}); err != nil {
		t.Fatalf("contact: %v", err)
	}
	events, _ := BindEventRepository(NewSQLEventStore(lanes), tenant)
	if _, err := events.Create(HouseEvent{Title: "E", StartsAt: now.Add(time.Hour)}); err != nil {
		t.Fatalf("event: %v", err)
	}
	handovers, _ := BindHandoverRepository(NewSQLHandoverStore(lanes), tenant)
	if _, err := handovers.Create(HandoverRecord{ID: "h1", Title: "H", CreatedBy: "a@example.com"}); err != nil {
		t.Fatalf("handover: %v", err)
	}
	issues, _ := BindIssueRepository(NewSQLIssueStore(lanes, fileDir), tenant)
	if _, err := issues.Create(ResidentIssue{
		AuthorEmail: "a@example.com", Category: "Schaden", LocationType: IssueLocationCommon,
		Title: "I", Body: "b",
	}); err != nil {
		t.Fatalf("issue: %v", err)
	}
	units, _ := BindUnitRepository(NewSQLUnitStore(lanes), tenant)
	if err := units.SetUnits([]Unit{{ID: "u1", Label: "Top 1"}}); err != nil {
		t.Fatalf("units: %v", err)
	}
	periods, _ := BindAnnualStatementPeriodRepository(NewSQLAnnualStatementPeriodStore(lanes), tenant)
	if _, err := periods.Save(AnnualStatementPeriod{Year: 2026, StartsOn: "2026-01-01", EndsOn: "2026-12-31", UpdatedAt: now, UpdatedBy: "a@example.com"}); err != nil {
		t.Fatalf("annual statement period: %v", err)
	}
	costTypes, _ := BindAnnualStatementCostTypeRepository(NewSQLAnnualStatementCostTypeStore(lanes), tenant)
	if _, err := costTypes.Save(AnnualStatementCostType{Key: "grundsteuer", Name: "Grundsteuer", Allocatable: true, AllocationKey: AllocationKeyNutzwert, UpdatedAt: now, UpdatedBy: "a@example.com"}); err != nil {
		t.Fatalf("annual statement cost type: %v", err)
	}
	prepayments, _ := BindAnnualStatementPrepaymentRepository(NewSQLAnnualStatementPrepaymentStore(lanes), tenant)
	if _, _, err := prepayments.Save(AnnualStatementPrepayment{PeriodYear: 2026, UnitID: "u1", AmountCents: 12345, UpdatedAt: now, UpdatedBy: "a@example.com"}); err != nil {
		t.Fatalf("annual statement prepayment: %v", err)
	}
	payments, _ := BindUnitPaymentStatusRepository(NewSQLUnitPaymentStatusStore(lanes), tenant)
	if _, err := payments.Set(UnitPaymentStatus{UnitID: "u1", Status: UnitPaymentStatusPaid, UpdatedBy: "a@example.com"}); err != nil {
		t.Fatalf("unit payment status: %v", err)
	}
	votes, _ := BindVoteRepository(NewSQLVoteStore(lanes), tenant)
	if _, err := votes.Create(Ballot{
		Title: "B", Options: []string{"Ja", "Nein"}, Type: BallotTypeCircular,
		Weighting: BallotWeightingPerHead, CreatedBy: "a@example.com",
	}); err != nil {
		t.Fatalf("ballot: %v", err)
	}
	documents, _ := BindDocumentRepository(NewSQLDocumentStore(lanes, filepath.Join(fileDir, "docs")), tenant)
	receiptDocument, err := documents.CreateGenerated(DocumentRecord{
		Title: "D", Category: "Protokoll", Visibility: "alle", UploadedBy: "a@example.com",
	}, "d.pdf", "application/pdf", []byte("%PDF-1.4 x"), now)
	if err != nil {
		t.Fatalf("document: %v", err)
	}
	receipts, _ := BindAnnualStatementReceiptRepository(NewSQLAnnualStatementReceiptStore(lanes), tenant)
	if _, err := receipts.Create(AnnualStatementReceipt{
		DocumentID: receiptDocument.ID, PeriodYear: 2026, CostTypeKey: "grundsteuer", AmountCents: 12345,
		InvoiceDate: "2026-06-30", CreatedAt: now, CreatedBy: "a@example.com",
	}); err != nil {
		t.Fatalf("annual statement receipt: %v", err)
	}
	attachments, _ := BindAttachmentRepository(NewSQLAttachmentStore(lanes, filepath.Join(fileDir, "att")), tenant)
	if _, err := attachments.CreateUploaded("issue", "i1", "a@example.com",
		[]UploadedFile{uploadFrom("photo.png", onePixelPNG)}, now); err != nil {
		t.Fatalf("attachment: %v", err)
	}
	identity := NewSQLIdentityStore(lanes)
	person, err := identity.UpsertPerson(Person{Email: "a@example.com"}, now)
	if err != nil {
		t.Fatalf("person: %v", err)
	}
	if _, err := identity.SetMembership(HouseMembership{
		PersonID: person.ID, TenantSlug: "demo", Role: RoleOwner, Status: "Aktiv",
	}, now); err != nil {
		t.Fatalf("membership: %v", err)
	}
	// home_connectors chains a foreign key to the reservation that created the
	// house, so the onboarding row has to exist first.
	reservations := NewSQLHomeReservationStore(lanes)
	if _, err := reservations.Reserve(HomeReservation{
		Slug: "demo", HouseholdName: "Demo", OwnerEmail: "a@example.com", AuthorizationConfirmed: true,
	}, now); err != nil {
		t.Fatalf("reservation: %v", err)
	}
	// home_portals: activation is the moment a reservation acquires an identity,
	// and SQLHomePortalStore.Activate is the only writer of this table. It was
	// excused as "internal/server activates a portal" until the closure check in
	// TestWritersCoveredElsewhereAreNotWrittenHere read the SQL and disagreed.
	if _, _, err := reservations.Confirm("demo", "a@example.com", now); err != nil {
		t.Fatalf("confirm reservation: %v", err)
	}
	portals := NewSQLHomePortalStore(lanes)
	if _, _, err := portals.Activate("demo", "a@example.com", now); err != nil {
		t.Fatalf("activate portal: %v", err)
	}
	connectors := NewSQLHomeConnectorStore(lanes)
	if _, err := connectors.StartPairing("demo", []byte("pairing-hash"), now.Add(time.Hour), now); err != nil {
		t.Fatalf("connector: %v", err)
	}
	readings := NewSQLHomeConnectorReadingStore(lanes)
	if err := readings.Upsert("demo", []HomeConnectorReading{{
		EntityID: "sensor.x", State: "1", LastUpdated: now,
	}}, now); err != nil {
		t.Fatalf("connector reading: %v", err)
	}

	for _, table := range tenantIDTables {
		var rows, withIdentity int
		if err := database.QueryRowContext(t.Context(),
			`SELECT count(*), count(tenant_id) FROM `+table.name).Scan(&rows, &withIdentity); err != nil {
			t.Fatalf("inspect %s: %v", table.name, err)
		}
		reason, excused := writersCoveredElsewhere[table.name]
		if rows == 0 {
			if !excused {
				t.Errorf("%s is tenant-scoped and nothing above writes to it: either exercise its "+
					"writer here or say in writersCoveredElsewhere where it IS exercised", table.name)
			}
			continue
		}
		if excused {
			t.Errorf("%s is listed in writersCoveredElsewhere (%s) but this test writes %d rows to it — "+
				"remove it from that list so those rows are checked", table.name, reason, rows)
			continue
		}
		// home_reservations is the one table whose rows legitimately have no
		// identity: the reservation is the request for a house, made before the
		// house exists. Its slug is still checked below.
		if withIdentity != rows && !table.preTenant {
			t.Errorf("%s: %d of %d rows written by the product have no tenant identity",
				table.name, rows-withIdentity, rows)
		}
		// The symmetric question, and the one that guards the rollback window.
		// tenant_slug is still written on 33 INSERTs for exactly one reason: the
		// previous release addresses rows by it, so rolling back must not lose
		// them. Until this loop asked, that contract was asserted for
		// `announcements` alone — binding "" for tenant_slug in any other writer
		// left the whole SQLite suite and both PostgreSQL suites green.
		var addressable int
		if err := database.QueryRowContext(t.Context(),
			`SELECT count(*) FROM `+table.name+` WHERE `+table.slugColumn+` = $1`, tenant.Slug).Scan(&addressable); err != nil {
			t.Fatalf("inspect %s.%s: %v", table.name, table.slugColumn, err)
		}
		if addressable != rows {
			t.Errorf("%s: %d of %d rows written by the product are not addressable by %s=%q — "+
				"a rollback to the previous release would not find them",
				table.name, rows-addressable, rows, table.slugColumn, tenant.Slug)
		}
	}

	// And the boot check agrees, on a database the product wrote rather than a
	// fixture built for it.
	if err := BackfillTenantIDs(t.Context(), database); err != nil {
		t.Fatalf("boot completeness check on freshly written rows: %v", err)
	}
}
