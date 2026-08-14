package server

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/telegram"
)

type recordedTelegram struct {
	mu       sync.Mutex
	messages []struct {
		ChatID int64
		Text   string
	}
}

func (f *recordedTelegram) Configured() bool { return true }
func (f *recordedTelegram) GetUpdates(_ context.Context, _ int64, _ time.Duration) ([]telegram.Update, error) {
	return nil, nil
}
func (f *recordedTelegram) SendMessage(_ context.Context, chatID int64, text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.messages = append(f.messages, struct {
		ChatID int64
		Text   string
	}{chatID, text})
	return nil
}
func (f *recordedTelegram) sent() []struct {
	ChatID int64
	Text   string
} {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]struct {
		ChatID int64
		Text   string
	}(nil), f.messages...)
}

func newTelegramTestApp(t *testing.T) (*app, *fakeHA, *recordedTelegram) {
	t.Helper()
	ha := newFakeHA(t)
	a, _, _ := newChargingTestApp(t, ha)
	tg := &recordedTelegram{}
	tgStore, err := store.NewTelegramStore(filepath.Join(t.TempDir(), "telegram.json"))
	if err != nil {
		t.Fatal(err)
	}
	a.telegram = tg
	a.telegramStore = tgStore
	a.telegramPollTimeout = time.Second
	a.profiles = map[string]userProfile{
		"joerg@example.com": {
			Email:       "joerg@example.com",
			FirstName:   "Jörg",
			Role:        "Bewohner",
			Tenants:     []string{"demo"},
			Permissions: []string{permissionParking},
		},
		"nopark@example.com": {
			Email:   "nopark@example.com",
			Role:    "Bewohner",
			Tenants: []string{"demo"},
		},
	}
	return a, ha, tg
}

const joergChat = int64(420702602)

func linkChat(t *testing.T, a *app, chatID int64, email string) {
	t.Helper()
	code, err := a.telegramStore.CreateLinkCode(email, "admin@example.com", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	reply := a.handleTelegramCommand(context.Background(), chatID, "Test", "/start "+code)
	if !strings.Contains(reply, "Verknüpft") {
		t.Fatalf("link failed: %q", reply)
	}
}

func TestTelegramLinkCodeFlow(t *testing.T) {
	a, _, _ := newTelegramTestApp(t)

	// Unknown chat, no code: generic onboarding hint, no information leak.
	reply := a.handleTelegramCommand(context.Background(), joergChat, "Jörg", "/start")
	if !strings.Contains(reply, "Code") {
		t.Fatalf("reply = %q", reply)
	}
	// Bad code.
	reply = a.handleTelegramCommand(context.Background(), joergChat, "Jörg", "/start WRONGCODE")
	if !strings.Contains(reply, "unbekannt oder abgelaufen") {
		t.Fatalf("reply = %q", reply)
	}
	// Real code links, and the code is single-use.
	code, err := a.telegramStore.CreateLinkCode("joerg@example.com", "admin@example.com", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	reply = a.handleTelegramCommand(context.Background(), joergChat, "Jörg", "/start "+code)
	if !strings.Contains(reply, "Verknüpft") {
		t.Fatalf("reply = %q", reply)
	}
	if _, err := a.telegramStore.ConsumeLinkCode(code, 999, "X"); err == nil {
		t.Fatal("link code must be single-use")
	}
	link, ok := a.telegramStore.LinkByChat(joergChat)
	if !ok || link.Email != "joerg@example.com" {
		t.Fatalf("link = %+v ok=%v", link, ok)
	}
}

func TestTelegramUnknownChatsGetSilence(t *testing.T) {
	a, _, _ := newTelegramTestApp(t)
	for _, text := range []string{"/pp20status", "/pp20ein", "hello"} {
		if reply := a.handleTelegramCommand(context.Background(), 555, "X", text); reply != "" {
			t.Fatalf("unknown chat got a reply for %q: %q", text, reply)
		}
	}
}

func TestTelegramPermissionRequired(t *testing.T) {
	a, _, _ := newTelegramTestApp(t)
	linkChat(t, a, 777, "nopark@example.com")
	reply := a.handleTelegramCommand(context.Background(), 777, "X", "/pp20ein")
	if !strings.Contains(reply, "Berechtigung") {
		t.Fatalf("reply = %q", reply)
	}
}

func TestTelegramManualCommandsRoundTrip(t *testing.T) {
	a, ha, _ := newTelegramTestApp(t)
	linkChat(t, a, joergChat, "joerg@example.com")

	reply := a.handleTelegramCommand(context.Background(), joergChat, "Jörg", "/pp20ein")
	if !strings.Contains(reply, "eingeschaltet") {
		t.Fatalf("reply = %q", reply)
	}
	data := a.parkingStore.TenantData("demo")
	if len(data.ChargingSessions) != 1 || data.ChargingSessions[0].Mode != chargingModeManual {
		t.Fatalf("sessions = %+v", data.ChargingSessions)
	}
	if data.ChargingSessions[0].TriggerSource != chargingTriggerTelegram || data.ChargingSessions[0].StartedBy != "joerg@example.com" {
		t.Fatalf("session provenance = %+v", data.ChargingSessions[0])
	}
	if calls := ha.calls(); len(calls) != 1 || calls[0] != "turn_on" {
		t.Fatalf("switch calls = %v", calls)
	}

	// Status mentions the manual mode (command with @botname suffix).
	reply = a.handleTelegramCommand(context.Background(), joergChat, "Jörg", "/PP20STATUS@HausvOrgBot")
	if !strings.Contains(reply, "Parkplatz 20") || !strings.Contains(reply, "Manuell eingeschaltet") {
		t.Fatalf("status = %q", reply)
	}

	// Immediate off is refused: min-on protects the relay.
	reply = a.handleTelegramCommand(context.Background(), joergChat, "Jörg", "/pp20aus")
	if !strings.Contains(reply, "warten") {
		t.Fatalf("reply = %q", reply)
	}
	if calls := ha.calls(); len(calls) != 1 {
		t.Fatalf("relay protection violated: %v", calls)
	}
}

func TestTelegramBroadcastRateLimit(t *testing.T) {
	a, _, tg := newTelegramTestApp(t)
	linkChat(t, a, joergChat, "joerg@example.com")
	base := len(tg.sent())

	for i := 0; i < chargingTelegramWindowLimit+3; i++ {
		a.telegramChargingBroadcast([]string{"joerg@example.com"}, "Test", nil)
	}
	if got := len(tg.sent()) - base; got != chargingTelegramWindowLimit {
		t.Fatalf("expected %d broadcasts, got %d", chargingTelegramWindowLimit, got)
	}
}
