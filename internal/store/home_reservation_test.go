package store_test

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/store"
)

func TestHomeReservationStoreParity(t *testing.T) {
	database, err := db.Open(filepath.Join(t.TempDir(), "hausv.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })

	stores := map[string]store.HomeReservationStorage{
		"memory": store.NewMemoryHomeReservationStore(),
		"sqlite": store.NewSQLHomeReservationStore(database),
	}
	for name, backend := range stores {
		t.Run(name, func(t *testing.T) {
			now := time.Date(2026, time.August, 14, 11, 0, 0, 0, time.UTC)
			created, err := backend.Reserve(store.HomeReservation{
				Slug: "mein-zuhause", HouseholdName: "Zuhause am Park", OwnerEmail: "owner@example.com",
				AuthorizationConfirmed: true,
			}, now)
			if err != nil {
				t.Fatal(err)
			}
			if created.Status != store.HomeReservationEmailPending || created.Slug != "mein-zuhause" {
				t.Fatalf("created reservation = %+v", created)
			}

			confirmed, ok, err := backend.Confirm("mein-zuhause", "owner@example.com", now.Add(time.Minute))
			if err != nil || !ok || confirmed.Status != store.HomeReservationEmailConfirmed || confirmed.ConfirmedAt == nil {
				t.Fatalf("confirmed reservation = %+v, ok=%v, err=%v", confirmed, ok, err)
			}
			updated, err := backend.Reserve(store.HomeReservation{
				Slug: "mein-zuhause", HouseholdName: "Neuer Name", OwnerEmail: "owner@example.com",
				AuthorizationConfirmed: true,
			}, now.Add(2*time.Minute))
			if err != nil || updated.HouseholdName != "Neuer Name" || updated.Status != store.HomeReservationEmailConfirmed {
				t.Fatalf("same-owner update = %+v, err=%v", updated, err)
			}
			if _, err := backend.Reserve(store.HomeReservation{
				Slug: "mein-zuhause", HouseholdName: "Fremd", OwnerEmail: "other@example.com",
				AuthorizationConfirmed: true,
			}, now); !errors.Is(err, store.ErrHomeReservationConflict) {
				t.Fatalf("conflict error = %v", err)
			}
			if _, ok, err := backend.Confirm("mein-zuhause", "other@example.com", now); err != nil || ok {
				t.Fatalf("foreign confirmation ok=%v err=%v", ok, err)
			}
			if removed, err := backend.PurgePendingBefore(now.Add(24 * time.Hour)); err != nil || removed != 0 {
				t.Fatalf("confirmed reservation purge = %d, err=%v", removed, err)
			}
			pendingSlug := "verfallen-" + name
			if _, err := backend.Reserve(store.HomeReservation{
				Slug: pendingSlug, HouseholdName: "Unbestätigt", OwnerEmail: name + "@example.com",
				AuthorizationConfirmed: true,
			}, now); err != nil {
				t.Fatal(err)
			}
			if removed, err := backend.PurgePendingBefore(now.Add(time.Hour)); err != nil || removed != 1 {
				t.Fatalf("pending reservation purge = %d, err=%v", removed, err)
			}
			if _, found, err := backend.Get(pendingSlug); err != nil || found {
				t.Fatalf("expired pending reservation remains, found=%v err=%v", found, err)
			}
		})
	}
}

func TestSQLHomeReservationSurvivesReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hausv.db")
	database, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	backend := store.NewSQLHomeReservationStore(database)
	if _, err := backend.Reserve(store.HomeReservation{
		Slug: "dauerhaft", HouseholdName: "Dauerhaftes Zuhause", OwnerEmail: "owner@example.com",
		AuthorizationConfirmed: true,
	}, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}

	database, err = db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	item, found, err := store.NewSQLHomeReservationStore(database).Get("dauerhaft")
	if err != nil || !found || item.OwnerEmail != "owner@example.com" {
		t.Fatalf("reopened reservation = %+v, found=%v, err=%v", item, found, err)
	}
}
