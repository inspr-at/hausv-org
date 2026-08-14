package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/db"
)

func TestTelegramStorageParity(t *testing.T) {
	backends := map[string]func(t *testing.T) TelegramStorage{
		"json": func(t *testing.T) TelegramStorage {
			s, err := NewTelegramStore(filepath.Join(t.TempDir(), "telegram.json"))
			if err != nil {
				t.Fatalf("json store: %v", err)
			}
			return s
		},
		"sqlite": func(t *testing.T) TelegramStorage {
			database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
			if err != nil {
				t.Fatalf("db open: %v", err)
			}
			t.Cleanup(func() { database.Close() })
			return NewSQLTelegramStore(database)
		},
	}

	for name, build := range backends {
		t.Run(name, func(t *testing.T) {
			s := build(t)

			// Offset round-trips; setting the same value again is a no-op.
			if got := s.Offset(); got != 0 {
				t.Fatalf("initial offset = %d", got)
			}
			if err := s.SetOffset(42); err != nil {
				t.Fatalf("set offset: %v", err)
			}
			if got := s.Offset(); got != 42 {
				t.Fatalf("offset = %d, want 42", got)
			}
			if err := s.SetOffset(42); err != nil {
				t.Fatalf("idempotent set offset: %v", err)
			}

			// Codes require a valid email.
			if _, err := s.CreateLinkCode("", "admin@example.com", time.Hour); err == nil {
				t.Fatal("empty email must error")
			}

			code, err := s.CreateLinkCode("user@example.com", "admin@example.com", time.Hour)
			if err != nil || len(code) != 8 {
				t.Fatalf("create code: err=%v code=%q", err, code)
			}

			// A fresh code for the same email replaces the old one.
			code2, err := s.CreateLinkCode("user@example.com", "admin@example.com", time.Hour)
			if err != nil {
				t.Fatalf("recreate code: %v", err)
			}
			if _, err := s.ConsumeLinkCode(code, 111, "Old"); err == nil {
				t.Fatal("the replaced code must no longer work")
			}

			// Redeeming links the chat.
			link, err := s.ConsumeLinkCode(code2, 111, "  Max  ")
			if err != nil {
				t.Fatalf("consume: %v", err)
			}
			if link.ChatID != 111 || link.Email != "user@example.com" || link.Name != "Max" {
				t.Fatalf("link = %+v", link)
			}
			if got, ok := s.LinkByChat(111); !ok || got.Email != "user@example.com" {
				t.Fatalf("LinkByChat = %+v ok=%v", got, ok)
			}
			if got := s.ChatsByEmail("user@example.com"); len(got) != 1 || got[0] != 111 {
				t.Fatalf("ChatsByEmail = %v", got)
			}
			if got := s.Links(); len(got) != 1 {
				t.Fatalf("Links = %+v", got)
			}

			// A code cannot be redeemed twice.
			if _, err := s.ConsumeLinkCode(code2, 222, "Other"); err == nil {
				t.Fatal("a consumed code must not work again")
			}
			// Unknown code is rejected.
			if _, err := s.ConsumeLinkCode("NOPE1234", 333, ""); err == nil {
				t.Fatal("unknown code must error")
			}
			// A non-positive TTL falls back to the 24h default rather than
			// minting an already-dead code.
			defaulted, err := s.CreateLinkCode("late@example.com", "admin@example.com", -time.Minute)
			if err != nil {
				t.Fatalf("create with non-positive ttl: %v", err)
			}
			if _, err := s.ConsumeLinkCode(defaulted, 444, ""); err != nil {
				t.Fatalf("a non-positive ttl must default to 24h, got: %v", err)
			}
			// Keep the link set clean for the assertions below.
			if err := s.Unlink(444); err != nil {
				t.Fatalf("unlink 444: %v", err)
			}

			// Re-linking the same chat to a new email replaces the old link.
			code3, err := s.CreateLinkCode("second@example.com", "admin@example.com", time.Hour)
			if err != nil {
				t.Fatalf("create third: %v", err)
			}
			if _, err := s.ConsumeLinkCode(code3, 111, "Max2"); err != nil {
				t.Fatalf("relink: %v", err)
			}
			if got := s.Links(); len(got) != 1 {
				t.Fatalf("relink must not duplicate: %+v", got)
			}
			if got, _ := s.LinkByChat(111); got.Email != "second@example.com" {
				t.Fatalf("relink email = %q", got.Email)
			}
			if got := s.ChatsByEmail("user@example.com"); len(got) != 0 {
				t.Fatalf("old email must have no chats: %v", got)
			}

			// Unlink removes it; unlinking again is harmless.
			if err := s.Unlink(111); err != nil {
				t.Fatalf("unlink: %v", err)
			}
			if _, ok := s.LinkByChat(111); ok {
				t.Fatal("unlinked chat must be gone")
			}
			if err := s.Unlink(111); err != nil {
				t.Fatalf("second unlink: %v", err)
			}
		})
	}
}

func TestSQLTelegramImportFromJSON(t *testing.T) {
	jsonStore, err := NewTelegramStore(filepath.Join(t.TempDir(), "telegram.json"))
	if err != nil {
		t.Fatalf("json store: %v", err)
	}
	if err := jsonStore.SetOffset(99); err != nil {
		t.Fatalf("offset: %v", err)
	}
	code, err := jsonStore.CreateLinkCode("user@example.com", "admin@example.com", time.Hour)
	if err != nil {
		t.Fatalf("code: %v", err)
	}
	if _, err := jsonStore.ConsumeLinkCode(code, 555, "Max"); err != nil {
		t.Fatalf("consume: %v", err)
	}
	pending, err := jsonStore.CreateLinkCode("pending@example.com", "admin@example.com", time.Hour)
	if err != nil {
		t.Fatalf("pending code: %v", err)
	}

	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db open: %v", err)
	}
	defer database.Close()
	sqlStore := NewSQLTelegramStore(database)

	for i := 0; i < 2; i++ {
		if err := sqlStore.ImportTelegram(jsonStore); err != nil {
			t.Fatalf("import %d: %v", i, err)
		}
	}
	if got := sqlStore.Offset(); got != 99 {
		t.Fatalf("imported offset = %d", got)
	}
	if got := sqlStore.Links(); len(got) != 1 || got[0].ChatID != 555 {
		t.Fatalf("imported links = %+v", got)
	}
	// The still-pending code survives the import and remains redeemable.
	if _, err := sqlStore.ConsumeLinkCode(pending, 666, "Pending"); err != nil {
		t.Fatalf("imported pending code must be redeemable: %v", err)
	}
}
