package store_test

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/store"
)

func TestHomeConnectorStoreParityPairRotateAndRevoke(t *testing.T) {
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	database, err := db.Open(filepath.Join(t.TempDir(), "hausv.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	reservations := store.NewSQLHomeReservationStore(database)
	if _, err := reservations.Reserve(store.HomeReservation{
		Slug: "sql-home", HouseholdName: "SQL Home", OwnerEmail: "owner@example.com", AuthorizationConfirmed: true,
	}, now); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := reservations.Confirm("sql-home", "owner@example.com", now); err != nil || !ok {
		t.Fatalf("confirm sql reservation: ok=%v err=%v", ok, err)
	}

	backends := map[string]struct {
		slug  string
		store store.HomeConnectorStorage
	}{
		"memory": {slug: "memory-home", store: store.NewMemoryHomeConnectorStore()},
		"sqlite": {slug: "sql-home", store: store.NewSQLHomeConnectorStore(database)},
	}
	for name, fixture := range backends {
		t.Run(name, func(t *testing.T) {
			pairingOne := bytes.Repeat([]byte{1}, 32)
			credentialOne := bytes.Repeat([]byte{2}, 32)
			item, err := fixture.store.StartPairing(fixture.slug, pairingOne, now.Add(10*time.Minute), now)
			if err != nil || item.Status != store.HomeConnectorPairing || item.PairingExpiresAt == nil {
				t.Fatalf("start pairing = %+v err=%v", item, err)
			}
			heartbeat := store.HomeConnectorHeartbeat{ConnectorVersion: "0.83.0", HomeAssistantVersion: "2026.8.1", EntityCount: 321}
			item, ok, err := fixture.store.ExchangePairing(pairingOne, credentialOne, heartbeat, now.Add(time.Minute))
			if err != nil || !ok || item.Status != store.HomeConnectorConnected || item.Generation != 1 || item.LastSeenAt == nil {
				t.Fatalf("exchange = %+v ok=%v err=%v", item, ok, err)
			}
			if _, replayed, err := fixture.store.ExchangePairing(pairingOne, bytes.Repeat([]byte{3}, 32), heartbeat, now.Add(2*time.Minute)); err != nil || replayed {
				t.Fatalf("pairing replay accepted=%v err=%v", replayed, err)
			}

			pairingTwo := bytes.Repeat([]byte{4}, 32)
			credentialTwo := bytes.Repeat([]byte{5}, 32)
			pending, err := fixture.store.StartPairing(fixture.slug, pairingTwo, now.Add(12*time.Minute), now.Add(2*time.Minute))
			if err != nil || pending.Status != store.HomeConnectorConnected || !bytes.Equal(pending.CredentialHash, credentialOne) {
				t.Fatalf("rotation preparation = %+v err=%v", pending, err)
			}
			if _, accepted, err := fixture.store.Heartbeat(credentialOne, heartbeat, now.Add(3*time.Minute)); err != nil || !accepted {
				t.Fatalf("old credential stopped before exchange: accepted=%v err=%v", accepted, err)
			}
			rotated, ok, err := fixture.store.ExchangePairing(pairingTwo, credentialTwo, heartbeat, now.Add(4*time.Minute))
			if err != nil || !ok || rotated.Generation != 2 || !bytes.Equal(rotated.CredentialHash, credentialTwo) {
				t.Fatalf("rotation = %+v ok=%v err=%v", rotated, ok, err)
			}
			if _, accepted, err := fixture.store.Heartbeat(credentialOne, heartbeat, now.Add(5*time.Minute)); err != nil || accepted {
				t.Fatalf("old credential survived rotation: accepted=%v err=%v", accepted, err)
			}
			if _, accepted, err := fixture.store.Heartbeat(credentialTwo, heartbeat, now.Add(5*time.Minute)); err != nil || !accepted {
				t.Fatalf("new credential rejected: accepted=%v err=%v", accepted, err)
			}

			revoked, found, err := fixture.store.Revoke(fixture.slug, now.Add(6*time.Minute))
			if err != nil || !found || revoked.Status != store.HomeConnectorRevoked || len(revoked.CredentialHash) != 0 || revoked.LastSeenAt != nil || revoked.EntityCount != 0 {
				t.Fatalf("revoke = %+v found=%v err=%v", revoked, found, err)
			}
			if _, accepted, err := fixture.store.Heartbeat(credentialTwo, heartbeat, now.Add(7*time.Minute)); err != nil || accepted {
				t.Fatalf("revoked credential accepted=%v err=%v", accepted, err)
			}
			exactExpiryPairing := bytes.Repeat([]byte{8}, 32)
			if _, err := fixture.store.StartPairing(fixture.slug, exactExpiryPairing, now.Add(8*time.Minute), now.Add(7*time.Minute)); err != nil {
				t.Fatalf("exact-expiry setup: %v", err)
			}
			if _, accepted, err := fixture.store.ExchangePairing(exactExpiryPairing, bytes.Repeat([]byte{9}, 32), heartbeat, now.Add(8*time.Minute)); err != nil || accepted {
				t.Fatalf("exact-expiry pairing accepted=%v err=%v", accepted, err)
			}
		})
	}
}

func TestHomeConnectorPairingExpiresAndSQLitePersists(t *testing.T) {
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	databasePath := filepath.Join(t.TempDir(), "hausv.db")
	database, err := db.Open(databasePath)
	if err != nil {
		t.Fatal(err)
	}
	reservations := store.NewSQLHomeReservationStore(database)
	if _, err := reservations.Reserve(store.HomeReservation{
		Slug: "persisted-home", HouseholdName: "Persisted", OwnerEmail: "owner@example.com", AuthorizationConfirmed: true,
	}, now); err != nil {
		t.Fatal(err)
	}
	backend := store.NewSQLHomeConnectorStore(database)
	pairing := bytes.Repeat([]byte{6}, 32)
	if _, err := backend.StartPairing("persisted-home", pairing, now.Add(time.Minute), now); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := backend.ExchangePairing(pairing, bytes.Repeat([]byte{7}, 32), store.HomeConnectorHeartbeat{
		ConnectorVersion: "0.83.0", HomeAssistantVersion: "2026.8.1", EntityCount: 12,
	}, now.Add(time.Minute)); err != nil || ok {
		t.Fatalf("expired exchange accepted=%v err=%v", ok, err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := db.Open(databasePath)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	item, found, err := store.NewSQLHomeConnectorStore(reopened).Get("persisted-home")
	if err != nil || !found || item.Status != store.HomeConnectorPairing || item.PairingExpiresAt == nil {
		t.Fatalf("persisted connector = %+v found=%v err=%v", item, found, err)
	}
}

func TestHomeConnectorReadingsMemoryAndSQLiteParity(t *testing.T) {
	now := time.Date(2026, 8, 14, 14, 0, 0, 0, time.UTC)
	database, err := db.Open(filepath.Join(t.TempDir(), "readings.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	reservations := store.NewSQLHomeReservationStore(database)
	if _, err := reservations.Reserve(store.HomeReservation{
		Slug: "sql-readings", HouseholdName: "SQL Readings", OwnerEmail: "owner@example.com", AuthorizationConfirmed: true,
	}, now); err != nil {
		t.Fatal(err)
	}
	if _, err := store.NewSQLHomeConnectorStore(database).StartPairing("sql-readings", bytes.Repeat([]byte{1}, 32), now.Add(time.Minute), now); err != nil {
		t.Fatal(err)
	}
	backends := map[string]struct {
		slug  string
		store store.HomeConnectorReadingStorage
	}{
		"memory": {slug: "memory-readings", store: store.NewMemoryHomeConnectorReadingStore()},
		"sqlite": {slug: "sql-readings", store: store.NewSQLHomeConnectorReadingStore(database)},
	}
	for name, fixture := range backends {
		t.Run(name, func(t *testing.T) {
			readings := []store.HomeConnectorReading{{
				EntityID: "sensor.grid_import_power", State: "1200", DisplayName: "Netzbezug",
				Unit: "W", DeviceClass: "power", StateClass: "measurement", LastUpdated: now,
			}}
			if err := fixture.store.Upsert(fixture.slug, readings, now); err != nil {
				t.Fatal(err)
			}
			readings[0].State = "1500"
			if err := fixture.store.Upsert(fixture.slug, readings, now.Add(time.Second)); err != nil {
				t.Fatal(err)
			}
			got, err := fixture.store.List(fixture.slug)
			if err != nil || len(got) != 1 || got[0].State != "1500" || got[0].ReceivedAt != now.Add(time.Second) {
				t.Fatalf("readings=%+v err=%v", got, err)
			}
			if err := fixture.store.Clear(fixture.slug); err != nil {
				t.Fatal(err)
			}
			got, err = fixture.store.List(fixture.slug)
			if err != nil || len(got) != 0 {
				t.Fatalf("cleared readings=%+v err=%v", got, err)
			}
		})
	}
}
