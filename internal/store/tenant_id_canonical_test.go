package store

import (
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

// This file replaces tenant_id_ambiguity_test.go. Every scenario that file
// exercised is still exercised here; what changed is the answer the code gives.
//
// The old answer was to LINK ONE CANDIDATE AND HIDE THE REST: a tenant whose
// slug was stored under two spellings had two candidates for one logical key,
// and a rule in Go picked between them and quarantined the loser — silently,
// permanently, with a log line as the only trace. Three rounds of that rule
// produced three regressions, the last of which hid the previous release's
// newest write behind a stale row.
//
// The new answer is that the situation cannot exist by the time linking runs:
// the spelling is rewritten first, and if the database will not accept the
// rewrite, the boot refuses and names the row. Nothing is chosen, so nothing can
// be chosen wrongly.

// TestBackfillHealsTheRollbackWindowAcrossAFullCycle is the one property that
// must survive every redesign of this file, driven as the whole cycle rather
// than as one boot.
//
// tenant_slug is still written on 33 INSERTs for exactly one reason: the
// previous release addresses rows by it. So a rollback is a supported move, and
// while it is in effect the old binary writes rows with a canonical tenant_slug
// and a NULL tenant_id. Rolling forward must HEAL those rows — link them, make
// them visible, keep the newest write authoritative. It must never hide one and
// it must never refuse the boot over one.
//
// Where it is blind: it drives two tables (an insert-only one keyed on a random
// id, and an upsert one keyed on a natural key), not fifteen, and it simulates
// the previous release with raw SQL in the shape that release used rather than
// by running the previous binary.
func TestBackfillHealsTheRollbackWindowAcrossAFullCycle(t *testing.T) {
	database, lanes := testLanes(t)
	tenant := testTenantRef("demo")
	now := time.Now().UTC().Truncate(time.Second)

	// 1. The new release runs. Every row it writes carries an identity.
	announcements, _ := BindAnnouncementRepository(NewSQLAnnouncementStore(lanes), tenant)
	if _, err := announcements.Create(Announcement{Title: "written by the new release", Body: "x"}); err != nil {
		t.Fatalf("new release announcement: %v", err)
	}
	payments, _ := BindUnitPaymentStatusRepository(NewSQLUnitPaymentStatusStore(lanes), tenant)
	if _, err := payments.Set(UnitPaymentStatus{
		UnitID: "u1", Status: UnitPaymentStatusOpen, UpdatedBy: "new@example.com",
	}); err != nil {
		t.Fatalf("new release payment: %v", err)
	}

	// 2. Roll back. The previous release knows nothing about tenant_id: it
	//    inserts a new announcement without one, updates the payment status it
	//    finds by (tenant_slug, unit_id), and creates a second payment row.
	seedSlugOnlyRow(t, database, "announcements", "demo", "written-by-the-old-release")
	if _, err := database.ExecContext(t.Context(),
		`UPDATE unit_payment_status SET status=$1, updated_at=$2, updated_by=$3
		 WHERE tenant_slug=$4 AND unit_id=$5`,
		UnitPaymentStatusPaid, now.Format(time.RFC3339Nano), "old@example.com", "demo", "u1"); err != nil {
		t.Fatalf("old release update: %v", err)
	}
	seedPaymentRow(t, database, "demo", "u2", UnitPaymentStatusPaid, now)

	// 3. Roll forward. This is the boot the product actually performs.
	if _, err := EnsureTenantIdentities(t.Context(), database,
		[]TenantIdentity{{Slug: "demo", Name: "Demo"}}); err != nil {
		t.Fatalf("rolling forward must heal the previous release's rows, not refuse: %v", err)
	}

	// 4. Both writes are present and both are visible.
	listed := announcements.List()
	if len(listed) != 2 {
		t.Errorf("announcements.List() = %d rows, want 2: the previous release's insert is invisible", len(listed))
	}
	got, ok := payments.Get("u1")
	if !ok || got.Status != UnitPaymentStatusPaid {
		t.Errorf("payments.Get(u1) = %+v ok=%v, want the status the previous release wrote (%q)",
			got, ok, UnitPaymentStatusPaid)
	}
	if _, ok := payments.Get("u2"); !ok {
		t.Error("payments.Get(u2) is invisible: a row the previous release created was not healed")
	}
	if listed := payments.List(); len(listed) != 2 {
		t.Errorf("payments.List() = %d rows, want 2", len(listed))
	}
	for _, table := range []string{"announcements", "unit_payment_status"} {
		if missing := countMissingTenantIDs(t, database, table); missing != 0 {
			t.Errorf("%s: %d rows left without an identity after rolling forward", table, missing)
		}
	}
}

// TestBackfillCanonicalisesEveryStoredSpellingBeforeLinking is the redesign
// itself, observed rather than described.
//
// Two spellings of one tenant on two different keys were already linkable under
// the old code — it linked each row under the spelling it carried and left the
// spelling alone. That left the table holding two labels for one tenant, which
// is the state every ambiguity rule existed to cope with. Now the label is
// repaired too, so the state the rules coped with no longer occurs.
func TestBackfillCanonicalisesEveryStoredSpellingBeforeLinking(t *testing.T) {
	database, lanes := testLanes(t)
	now := time.Now().UTC().Truncate(time.Second)
	seedPaymentRow(t, database, "Demo", "u1", UnitPaymentStatusOpen, now)
	seedPaymentRow(t, database, "demo ", "u2", UnitPaymentStatusPaid, now)
	seedPaymentRow(t, database, "demo", "u3", UnitPaymentStatusPaid, now)

	if err := BackfillTenantIDs(t.Context(), database); err != nil {
		t.Fatalf("backfill: %v", err)
	}

	payments, _ := BindUnitPaymentStatusRepository(NewSQLUnitPaymentStatusStore(lanes), testTenantRef("demo"))
	if listed := payments.List(); len(listed) != 3 {
		t.Errorf("payments.List() = %d rows, want 3: every row of this tenant must be visible", len(listed))
	}
	assertOneSpellingPerTenant(t, database)
	// And the previous release still finds all three by the label it uses.
	var addressable int
	if err := database.QueryRowContext(t.Context(),
		`SELECT count(*) FROM unit_payment_status WHERE tenant_slug='demo'`).Scan(&addressable); err != nil {
		t.Fatalf("count by label: %v", err)
	}
	if addressable != 3 {
		t.Errorf("%d of 3 rows are addressable by tenant_slug='demo'; the rollback window is closed", addressable)
	}
}

// TestBackfillIsIdempotentAfterCanonicalising: a second boot must reach the same
// state as the first and must not pay for the pass again.
func TestBackfillIsIdempotentAfterCanonicalising(t *testing.T) {
	database := testDB(t)
	now := time.Now().UTC().Truncate(time.Second)
	seedPaymentRow(t, database, "Demo", "u1", UnitPaymentStatusOpen, now)

	first := ""
	for attempt := 1; attempt <= 2; attempt++ {
		if err := BackfillTenantIDs(t.Context(), database); err != nil {
			t.Fatalf("backfill %d: %v", attempt, err)
		}
		var slug, owner string
		if err := database.QueryRowContext(t.Context(),
			`SELECT tenant_slug, tenant_id FROM unit_payment_status WHERE unit_id='u1'`).Scan(&slug, &owner); err != nil {
			t.Fatalf("read the row after boot %d: %v", attempt, err)
		}
		if slug != "demo" {
			t.Fatalf("boot %d left tenant_slug = %q, want the canonical %q", attempt, slug, "demo")
		}
		if attempt == 1 {
			first = owner
			continue
		}
		if owner != first {
			t.Fatalf("identity changed across boots: %q -> %q", first, owner)
		}
	}
}

// TestBackfillRefusesASpellingThatCannotBeCanonicalised is the deliberate
// behaviour change, stated as a test so it cannot be mistaken for an accident.
//
// Each case is one the old code answered by quarantining a row: it returned nil,
// the boot succeeded, and one row stayed invisible with only a log line to say
// so. The first case is the regression that prompted this rewrite — a row the
// PREVIOUS RELEASE wrote, hidden behind a stale legacy row that already held its
// key, permanently and unrepairably.
//
// The new answer is to refuse and name the row. That is worse for availability
// and better for the data: the failure is visible, nothing is written, and the
// repair is one UPDATE the operator can see the shape of from the message.
//
// None of these states can be produced by the running product — every write path
// canonicalises before it stores — and a census of the live database on
// 2026-08-17 found zero non-canonical slugs in all 21 tenant_slug tables.
//
// Where it is blind: it proves the pass REFUSES and names the table and both
// spellings. It does not prove the message is enough to repair every table, and
// it says nothing about how an operator is alerted to a boot that will not come
// back up.
func TestBackfillRefusesASpellingThatCannotBeCanonicalised(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	cases := map[string]struct {
		seed  func(t *testing.T, database *sql.DB)
		table string
	}{
		// The regression: an already-linked legacy row holds u1, and the
		// previous release's newer write for u1 arrives under the canonical
		// spelling. The old code quarantined the NEWER row.
		"a linked legacy row already holds the key": {
			table: "unit_payment_status",
			seed: func(t *testing.T, database *sql.DB) {
				seedPaymentRow(t, database, "Demo", "u1", UnitPaymentStatusOpen, now)
				if _, err := database.ExecContext(t.Context(),
					`UPDATE unit_payment_status SET tenant_id=$1 WHERE tenant_slug='Demo'`,
					testTenantID("demo")); err != nil {
					t.Fatalf("link the legacy row as an earlier boot would have: %v", err)
				}
				seedPaymentRow(t, database, "demo", "u1", UnitPaymentStatusPaid, now)
			},
		},
		"two unowned spellings hold the same key": {
			table: "unit_payment_status",
			seed: func(t *testing.T, database *sql.DB) {
				seedPaymentRow(t, database, "Demo", "u1", UnitPaymentStatusOpen, now)
				seedPaymentRow(t, database, "demo", "u1", UnitPaymentStatusPaid, now)
			},
		},
		"no spelling is the canonical one": {
			table: "unit_payment_status",
			seed: func(t *testing.T, database *sql.DB) {
				seedPaymentRow(t, database, "Demo", "u1", UnitPaymentStatusOpen, now)
				seedPaymentRow(t, database, "DEMO", "u1", UnitPaymentStatusPaid, now)
			},
		},
		// integration_imports is keyed on (tenant_slug, format, file_digest):
		// a composite key, not a single id column.
		"the collision is on a composite key": {
			table: "integration_imports",
			seed: func(t *testing.T, database *sql.DB) {
				seedImportRow(t, database, "Demo", "camt053", "digest-1", "old@example.com")
				seedImportRow(t, database, "demo", "camt053", "digest-1", "new@example.com")
			},
		},
		// home_portals is PRIMARY KEY (slug): the tenant label IS the whole key,
		// so two spellings are two candidates for one row.
		"the tenant label is the whole key": {
			table: "home_portals",
			seed: func(t *testing.T, database *sql.DB) {
				for _, slug := range []string{"Demo", "demo"} {
					seedReservationAndPortal(t, database, slug)
				}
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			database := testDB(t)
			tc.seed(t, database)

			err := BackfillTenantIDs(t.Context(), database)
			if !errors.Is(err, ErrTenantSlugNotCanonical) {
				t.Fatalf("backfill error = %v, want ErrTenantSlugNotCanonical", err)
			}
			message := err.Error()
			for _, want := range []string{tc.table, `"Demo"`, `"demo"`} {
				if !strings.Contains(message, want) {
					t.Errorf("the error must name %s so the row can be found, got: %s", want, message)
				}
			}
			// Nothing was written: the whole pass is one transaction, so a
			// refusal leaves the data exactly as the operator will find it.
			var rows int
			if err := database.QueryRowContext(t.Context(),
				`SELECT count(*) FROM `+tc.table).Scan(&rows); err != nil {
				t.Fatalf("count %s: %v", tc.table, err)
			}
			if rows != 2 {
				t.Errorf("%s has %d rows, want the 2 that were seeded: the refusal must not have written anything",
					tc.table, rows)
			}
		})
	}
}

// TestBackfillRefusesToCanonicaliseALabelTiedByAForeignKey pins the second way
// the rewrite can be refused, and the reason it cannot simply be worked around.
//
// The energy tables carry FOREIGN KEY (tenant_slug, home_key) REFERENCES
// home_profiles, and the onboarding chain carries FOREIGN KEY (slug) REFERENCES
// home_reservations(slug). Neither constraint is deferrable on PostgreSQL, so
// there is no order in which the parent and its children can be rewritten:
// parent-first orphans the children, child-first points them at a parent that
// does not exist yet. Both engines refuse both directions — measured, not
// assumed.
//
// So this condition is fatal, exactly like a collision, and for the same reason:
// the alternative is to link the rows under a label the rest of the system
// cannot resolve, which is the state migration 0033 already shipped once.
func TestBackfillRefusesToCanonicaliseALabelTiedByAForeignKey(t *testing.T) {
	database := testDB(t)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := database.ExecContext(t.Context(),
		`INSERT INTO home_profiles(tenant_slug,home_key,created_at,updated_at) VALUES($1,'default',$2,$3)`,
		"Demo", now, now); err != nil {
		t.Fatalf("seed home_profiles: %v", err)
	}
	if _, err := database.ExecContext(t.Context(),
		`INSERT INTO energy_assets(id,tenant_slug,home_key,kind,name,created_at,updated_at)
		 VALUES('a1',$1,'default','pv','PV',$2,$3)`, "Demo", now, now); err != nil {
		t.Fatalf("seed energy_assets: %v", err)
	}

	err := BackfillTenantIDs(t.Context(), database)
	if !errors.Is(err, ErrTenantSlugNotCanonical) {
		t.Fatalf("backfill error = %v, want ErrTenantSlugNotCanonical", err)
	}
	if !strings.Contains(err.Error(), `"Demo"`) {
		t.Errorf("the error must name the offending spelling, got: %s", err)
	}
}

// assertOneSpellingPerTenant is the invariant the whole redesign rests on: after
// a successful pass, no table holds two labels for one tenant, so no linking
// decision can have two candidates.
func assertOneSpellingPerTenant(t *testing.T, database *sql.DB) {
	t.Helper()
	for _, table := range tenantIDTables {
		rows, err := database.QueryContext(t.Context(),
			`SELECT DISTINCT `+table.slugColumn+` FROM `+table.name+
				` WHERE `+table.slugColumn+` IS NOT NULL`)
		if err != nil {
			t.Fatalf("read stored slugs in %s: %v", table.name, err)
		}
		seen := map[string][]string{}
		for rows.Next() {
			var slug string
			if err := rows.Scan(&slug); err != nil {
				rows.Close()
				t.Fatalf("scan stored slug in %s: %v", table.name, err)
			}
			seen[textutil.Slug(slug)] = append(seen[textutil.Slug(slug)], slug)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			t.Fatalf("iterate stored slugs in %s: %v", table.name, err)
		}
		for canonical, spellings := range seen {
			if len(spellings) > 1 {
				t.Errorf("%s stores tenant %q under %d spellings %v — linking has a choice to make again",
					table.name, canonical, len(spellings), spellings)
			}
		}
	}
}

func seedReservationAndPortal(t *testing.T, database *sql.DB, slug string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := database.ExecContext(t.Context(),
		`INSERT INTO home_reservations(slug, household_name, owner_email, authorization_confirmed, status, created_at, updated_at)
		 VALUES($1,$2,$3,$4,$5,$6,$7)`,
		slug, slug, "a@example.com", true, "aktiv", now, now); err != nil {
		t.Fatalf("seed reservation %s: %v", slug, err)
	}
	if _, err := database.ExecContext(t.Context(),
		`INSERT INTO home_portals(slug, household_name, owner_email, activated_at, updated_at)
		 VALUES($1,$2,$3,$4,$5)`,
		slug, slug, "a@example.com", now, now); err != nil {
		t.Fatalf("seed portal %s: %v", slug, err)
	}
}

func seedPaymentRow(t *testing.T, database *sql.DB, slug, unitID, status string, at time.Time) {
	t.Helper()
	if _, err := database.ExecContext(t.Context(),
		`INSERT INTO unit_payment_status(tenant_slug, unit_id, status, updated_at, updated_by)
		 VALUES($1,$2,$3,$4,$5)`,
		slug, unitID, status, at.Format(time.RFC3339Nano), "seed@example.com"); err != nil {
		t.Fatalf("seed unit_payment_status(%q,%q): %v", slug, unitID, err)
	}
}

func seedImportRow(t *testing.T, database *sql.DB, slug, format, digest, actor string) {
	t.Helper()
	if _, err := database.ExecContext(t.Context(),
		`INSERT INTO integration_imports(tenant_slug, format, file_digest, source_version, applied_at, applied_by)
		 VALUES($1,$2,$3,'v1','2026-01-01T00:00:00Z',$4)`,
		slug, format, digest, actor); err != nil {
		t.Fatalf("seed integration_imports(%q,%q): %v", slug, digest, err)
	}
}
