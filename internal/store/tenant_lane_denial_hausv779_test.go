package store

import (
	"bytes"
	"testing"
	"time"

	appdb "github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/dbtest"
)

// House A's lane must not read or change house B's rows on the paths HAUSV-779
// moved onto For(tenant). On PostgreSQL that is the fail-closed policy: the
// query names B's key and still sees nothing. On SQLite there is one pool, so
// the same statements carry the tenant_id predicate the stores use.
func TestConvertedPathsDenyTheOtherHouseLane(t *testing.T) {
	database, lanes := testLanes(t)
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	demo := testTenantRef("demo")
	other := testTenantRef("other")

	reads, ok := BindAnnouncementReadRepository(NewSQLAnnouncementReadStore(lanes), demo)
	if !ok {
		t.Fatal("bind demo announcement reads")
	}
	if err := reads.MarkSeen("a@example.com", now); err != nil {
		t.Fatalf("mark seen: %v", err)
	}
	otherReads, ok := BindAnnouncementReadRepository(NewSQLAnnouncementReadStore(lanes), other)
	if !ok {
		t.Fatal("bind other announcement reads")
	}
	if !otherReads.LastSeen("a@example.com").IsZero() {
		t.Fatal("other house read demo's announcement marker")
	}

	reservations := NewSQLHomeReservationStore(lanes)
	if _, err := reservations.Reserve(HomeReservation{
		Slug: "demo", HouseholdName: "Demo", OwnerEmail: "owner@example.com", AuthorizationConfirmed: true,
	}, now); err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if _, ok, err := reservations.Confirm("demo", "owner@example.com", now); err != nil || !ok {
		t.Fatalf("confirm reservation: ok=%v err=%v", ok, err)
	}
	connectors := NewSQLHomeConnectorStore(lanes)
	pairing := bytes.Repeat([]byte{1}, 32)
	if _, err := connectors.StartPairing("demo", pairing, now.Add(10*time.Minute), now); err != nil {
		t.Fatalf("start pairing: %v", err)
	}
	readings := NewSQLHomeConnectorReadingStore(lanes)
	if err := readings.Upsert("demo", []HomeConnectorReading{{
		EntityID: "sensor.power", State: "1.5", DisplayName: "Leistung",
	}}, now); err != nil {
		t.Fatalf("upsert reading: %v", err)
	}

	if dbtest.Backend() == appdb.BackendPostgres {
		lane := lanes.For(other)
		assertLaneCount(t, lane, `SELECT count(*) FROM announcement_reads WHERE email=$1`, "a@example.com")
		assertLaneMiss(t, lane, `UPDATE announcement_reads SET seen_at=$1 WHERE email=$2`, now.Add(2*time.Hour).Format(time.RFC3339Nano), "a@example.com")
		assertLaneCount(t, lane, `SELECT count(*) FROM home_connectors WHERE slug=$1`, "demo")
		assertLaneMiss(t, lane, `UPDATE home_connectors SET status=$1 WHERE slug=$2`, HomeConnectorRevoked, "demo")
		assertLaneCount(t, lane, `SELECT count(*) FROM home_connector_readings WHERE slug=$1`, "demo")
		assertLaneMiss(t, lane, `DELETE FROM home_connector_readings WHERE slug=$1`, "demo")
	} else {
		assertLaneCount(t, lanes.For(other), `SELECT count(*) FROM announcement_reads WHERE tenant_id=$1 AND email=$2`, other.ID, "a@example.com")
		assertLaneCount(t, lanes.For(other), `SELECT count(*) FROM home_connectors WHERE tenant_id=$1 AND slug=$2`, other.ID, "demo")
		assertLaneCount(t, lanes.For(other), `SELECT count(*) FROM home_connector_readings WHERE tenant_id=$1 AND slug=$2`, other.ID, "demo")
	}

	if err := otherReads.MarkSeen("a@example.com", now.Add(time.Hour)); err != nil {
		t.Fatalf("other house mark seen: %v", err)
	}
	if got := reads.LastSeen("a@example.com"); !got.Equal(now) {
		t.Fatalf("demo marker after the other lane = %v", got)
	}
	item, found, err := connectors.Get("demo")
	if err != nil || !found || item.Status != HomeConnectorPairing {
		t.Fatalf("demo connector after the other lane = %+v found=%v err=%v", item, found, err)
	}
	listed, err := readings.List("demo")
	if err != nil || len(listed) != 1 || listed[0].State != "1.5" {
		t.Fatalf("demo readings after the other lane = %+v err=%v", listed, err)
	}
	if err := readings.Clear("other"); err != nil {
		t.Fatalf("clear other: %v", err)
	}
	listed, err = readings.List("demo")
	if err != nil || len(listed) != 1 {
		t.Fatalf("clearing the other house removed demo readings: %+v err=%v", listed, err)
	}

	var demoConnectors int
	if err := database.QueryRow(`SELECT count(*) FROM home_connectors WHERE slug=$1`, "demo").Scan(&demoConnectors); err != nil {
		t.Fatalf("count demo connector: %v", err)
	}
	if demoConnectors != 1 {
		t.Fatalf("demo connectors = %d", demoConnectors)
	}
}

func assertLaneCount(t *testing.T, lane appdb.Handle, query string, args ...any) {
	t.Helper()
	var n int
	if err := lane.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	if n != 0 {
		t.Fatalf("%s returned %d rows, want 0", query, n)
	}
}

func assertLaneMiss(t *testing.T, lane appdb.Handle, query string, args ...any) {
	t.Helper()
	result, err := lane.Exec(query, args...)
	if err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("%s rows affected: %v", query, err)
	}
	if n != 0 {
		t.Fatalf("%s changed %d rows, want 0", query, n)
	}
}
