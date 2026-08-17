package store

import (
	"path/filepath"
	"testing"
)

func TestNotificationPrefStorageParity(t *testing.T) {
	backends := map[string]func(t *testing.T) NotificationPrefStorage{
		"json": func(t *testing.T) NotificationPrefStorage {
			s, err := NewNotificationPrefStore(filepath.Join(t.TempDir(), "np.json"))
			if err != nil {
				t.Fatalf("json store: %v", err)
			}
			return s
		},
		"sqlite": func(t *testing.T) NotificationPrefStorage {
			database := testDB(t)
			t.Cleanup(func() { database.Close() })
			return NewSQLNotificationPrefStore(database)
		},
	}

	for name, build := range backends {
		t.Run(name, func(t *testing.T) {
			s := build(t)

			// Unknown user -> defaults; unknown/known events default to enabled.
			if !s.EmailEnabled("nobody@example.com", "announcement") {
				t.Fatal("default must be enabled for an unknown user")
			}
			if s.EmailEnabled("nobody@example.com", "") {
				t.Fatal("empty event must be false")
			}

			if err := s.Set("", NotificationPreferences{}); err == nil {
				t.Fatal("empty email must error")
			}

			// Explicitly disable one event; another stays default-enabled.
			if err := s.Set("Person@Example.com", NotificationPreferences{
				Email: map[string]bool{"announcement": false},
			}); err != nil {
				t.Fatalf("set: %v", err)
			}
			if s.EmailEnabled("person@example.com", "announcement") {
				t.Fatal("explicitly disabled event must be off")
			}
			if !s.EmailEnabled("person@example.com", "issue") {
				t.Fatal("unset event must default enabled")
			}

			// Global unsubscribe wins over per-event flags.
			if err := s.Set("person@example.com", NotificationPreferences{
				Unsubscribed: true,
				Email:        map[string]bool{"announcement": true},
			}); err != nil {
				t.Fatalf("unsubscribe set: %v", err)
			}
			if s.EmailEnabled("person@example.com", "announcement") {
				t.Fatal("unsubscribed user must get nothing even if event is true")
			}
			if got := s.Get("person@example.com"); !got.Unsubscribed {
				t.Fatalf("Get must reflect unsubscribed: %+v", got)
			}
		})
	}
}

func TestSQLNotificationPrefImportFromJSON(t *testing.T) {
	jsonStore, err := NewNotificationPrefStore(filepath.Join(t.TempDir(), "np.json"))
	if err != nil {
		t.Fatalf("json store: %v", err)
	}
	_ = jsonStore.Set("a@example.com", NotificationPreferences{Unsubscribed: true})
	_ = jsonStore.Set("b@example.com", NotificationPreferences{Email: map[string]bool{"issue": false}})

	database := testDB(t)
	defer database.Close()
	sqlStore := NewSQLNotificationPrefStore(database)

	// A newer SQLite write must survive re-import.
	if err := sqlStore.Set("a@example.com", NotificationPreferences{Email: map[string]bool{"x": true}}); err != nil {
		t.Fatalf("pre-set: %v", err)
	}
	for i := 0; i < 2; i++ {
		if err := sqlStore.ImportPrefs(jsonStore); err != nil {
			t.Fatalf("import %d: %v", i, err)
		}
	}
	if sqlStore.Get("a@example.com").Unsubscribed {
		t.Fatal("import clobbered a newer SQLite write for a@")
	}
	if sqlStore.EmailEnabled("b@example.com", "issue") {
		t.Fatal("import missed b@'s disabled 'event'")
	}
}
