package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/homeassistant"
	"github.com/inspr-at/hausv-org/internal/store"
)

// Fake Home Assistant: /api/states/{id} + switch services against a virtual
// plug, scriptable values and a fail toggle.
type fakeHA struct {
	mu          sync.Mutex
	soc         string
	feed        string
	meter       string
	plugOn      bool
	fail        bool
	switchCalls []string
	srv         *httptest.Server
}

func newFakeHA(t *testing.T) *fakeHA {
	t.Helper()
	f := &fakeHA{soc: "100", feed: "5000", meter: "2550.13"}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.fail {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		switch {
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/services/switch/"):
			service := strings.TrimPrefix(r.URL.Path, "/api/services/switch/")
			f.switchCalls = append(f.switchCalls, service)
			f.plugOn = service == "turn_on"
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("[]"))
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/states/"):
			entity := strings.TrimPrefix(r.URL.Path, "/api/states/")
			state := ""
			switch entity {
			case "switch.plug":
				state = "off"
				if f.plugOn {
					state = "on"
				}
			case "sensor.soc":
				state = f.soc
			case "sensor.feed":
				state = f.feed
			case "sensor.meter":
				state = f.meter
			default:
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"entity_id":    entity,
				"state":        state,
				"last_updated": time.Now().UTC().Format(time.RFC3339),
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeHA) calls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.switchCalls...)
}

func newChargingTestApp(t *testing.T, ha *fakeHA) (*app, tenantConfig, *recordingMailer) {
	t.Helper()
	parkingStore, err := newParkingStore(filepath.Join(t.TempDir(), "parking.json"))
	if err != nil {
		t.Fatal(err)
	}
	inviteStore, err := newInviteStore("")
	if err != nil {
		t.Fatal(err)
	}
	haCfg := homeassistant.NewConfig(ha.srv.URL, "test-token", "sensor.meter", "sensor.power", "sensor.price").
		WithChargingEntities("switch.plug", "sensor.soc", "sensor.feed")
	tenant := tenantConfig{Slug: "demo", Name: "Test", HA: haCfg}
	mailer := &recordingMailer{}
	a := &app{
		defaultTenant:          "demo",
		tenants:                map[string]tenantConfig{"demo": tenant},
		profiles:               map[string]userProfile{},
		admins:                 map[string]struct{}{"admin@example.com": {}},
		allowed:                map[string]struct{}{},
		mailer:                 mailer,
		inviteStore:            inviteStore,
		parkingStore:           parkingStore,
		chargingTickInterval:   time.Second,
		chargingStaleAfter:     10 * time.Minute,
		chargingConfirmTimeout: 2 * time.Minute,
		chargingHAFailLimit:    2,
		chargingHAFails:        map[string]int{},
		chargingShadow:         map[string]chargingControllerState{},
		chargingShadowPlug:     map[string]bool{},
		chargingEvents:         &chargingEventRing{},
		chargingLastPoll:       map[string]time.Time{},
	}
	if err := parkingStore.SetChargingControl("demo", chargingControlSettings{Enabled: true}); err != nil {
		t.Fatal(err)
	}
	return a, tenant, mailer
}

func TestChargingControllerStartsSessionEndToEnd(t *testing.T) {
	ha := newFakeHA(t)
	a, tenant, mailer := newChargingTestApp(t, ha)

	a.tickChargingTenant(context.Background(), tenant)

	data := a.parkingStore.TenantData("demo")
	if len(data.ChargingSessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(data.ChargingSessions))
	}
	session := data.ChargingSessions[0]
	if session.Mode != store.ChargingModeSurplus || !session.End.IsZero() || session.StartKWh != 2550.13 {
		t.Fatalf("session = %+v", session)
	}
	if data.Charging.Phase != chargingPhaseSurplus || data.Charging.ActiveSessionID != session.ID {
		t.Fatalf("controller state = %+v", data.Charging)
	}
	if calls := ha.calls(); len(calls) != 1 || calls[0] != "turn_on" {
		t.Fatalf("switch calls = %v", calls)
	}
	if len(data.EnergySamples) == 0 || data.EnergySamples[len(data.EnergySamples)-1].Value != 2550.13 {
		t.Fatal("boundary sample missing")
	}
	if len(mailer.notifications) != 1 || !strings.Contains(mailer.notifications[0].Subject, "gestartet") {
		t.Fatalf("notifications = %+v", mailer.notifications)
	}

	// Fake plug already confirmed (service call flips it). Further ticks must
	// neither switch again nor notify again.
	a.tickChargingTenant(context.Background(), tenant)
	a.tickChargingTenant(context.Background(), tenant)
	if calls := ha.calls(); len(calls) != 1 {
		t.Fatalf("oscillation: switch calls = %v", calls)
	}
	if len(mailer.notifications) != 1 {
		t.Fatalf("notification spam: %+v", mailer.notifications)
	}
	data = a.parkingStore.TenantData("demo")
	if data.Charging.PendingConfirm != "" {
		t.Fatalf("pending confirm not cleared: %+v", data.Charging)
	}
}

func TestChargingControllerRecoveryAfterRestart(t *testing.T) {
	ha := newFakeHA(t)
	a, _, _ := newChargingTestApp(t, ha)

	start := time.Now().UTC().Add(-time.Hour)
	session, err := a.parkingStore.StartChargingSession("demo", chargingSession{
		Start: start, StartKWh: 2500, Mode: store.ChargingModeSurplus, TriggerSource: chargingTriggerAuto, StartedBy: "system",
	}, chargingControllerState{Phase: chargingPhaseSurplus, LastSwitchAt: start})
	if err != nil {
		t.Fatal(err)
	}
	if err := a.parkingStore.AppendReadings("demo", []parkingNumericSample{{At: start.Add(30 * time.Minute), Value: 2504.5}}, nil); err != nil {
		t.Fatal(err)
	}

	// Plug on: session must survive recovery.
	ha.mu.Lock()
	ha.plugOn = true
	ha.mu.Unlock()
	a.recoverChargingTenants(context.Background())
	data := a.parkingStore.TenantData("demo")
	if data.Charging.ActiveSessionID != session.ID {
		t.Fatalf("resume lost the session: %+v", data.Charging)
	}

	// Plug off: recovery must close it at the last stored sample.
	ha.mu.Lock()
	ha.plugOn = false
	ha.mu.Unlock()
	a.recoverChargingTenants(context.Background())
	data = a.parkingStore.TenantData("demo")
	if data.Charging.ActiveSessionID != "" || data.Charging.Phase != chargingPhaseIdle {
		t.Fatalf("recovery did not close: %+v", data.Charging)
	}
	closed := data.ChargingSessions[0]
	if closed.End.IsZero() || closed.EndReason != "restart" || closed.EndKWh != 2504.5 {
		t.Fatalf("closed session = %+v", closed)
	}
}

func TestChargingControllerHAOutageAlertsOnce(t *testing.T) {
	ha := newFakeHA(t)
	a, tenant, mailer := newChargingTestApp(t, ha)
	ha.mu.Lock()
	ha.fail = true
	ha.mu.Unlock()

	for i := 0; i < 4; i++ {
		a.tickChargingTenant(context.Background(), tenant)
	}
	alerts := 0
	for _, n := range mailer.notifications {
		if strings.Contains(n.Subject, "Störung") {
			alerts++
		}
	}
	if alerts != 1 {
		t.Fatalf("expected exactly 1 outage alert, got %d (%+v)", alerts, mailer.notifications)
	}

	ha.mu.Lock()
	ha.fail = false
	ha.mu.Unlock()
	a.tickChargingTenant(context.Background(), tenant)
	recoveries := 0
	for _, n := range mailer.notifications {
		if strings.Contains(n.Body, "wieder erreichbar") {
			recoveries++
		}
	}
	if recoveries != 1 {
		t.Fatalf("expected exactly 1 recovery notice, got %d", recoveries)
	}
}

func TestChargingControllerShadowNeverSwitches(t *testing.T) {
	ha := newFakeHA(t)
	a, tenant, mailer := newChargingTestApp(t, ha)
	if err := a.parkingStore.SetChargingControl("demo", chargingControlSettings{Enabled: true, ShadowMode: true}); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 3; i++ {
		a.tickChargingTenant(context.Background(), tenant)
	}
	if calls := ha.calls(); len(calls) != 0 {
		t.Fatalf("shadow mode switched the plug: %v", calls)
	}
	data := a.parkingStore.TenantData("demo")
	if len(data.ChargingSessions) != 0 {
		t.Fatalf("shadow mode created sessions: %+v", data.ChargingSessions)
	}
	if len(mailer.notifications) != 0 {
		t.Fatalf("shadow mode notified: %+v", mailer.notifications)
	}
	events := a.chargingEvents.list("demo", 0)
	if len(events) == 0 {
		t.Fatal("shadow mode must log events")
	}
	for _, event := range events {
		if !event.Shadow || !strings.Contains(event.Detail, "[Test]") {
			t.Fatalf("event not marked as shadow: %+v", event)
		}
	}
}
