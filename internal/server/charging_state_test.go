package server

import (
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
)

// The pure state machine. Every scenario here is a race or failure mode the
// old Node-RED flow got wrong (or never handled).

func chargingTestSettings() chargingControlSettings {
	return store.NormalizeChargingControlSettings(chargingControlSettings{
		Enabled:          true,
		StartSocPercent:  99,
		StopSocPercent:   95,
		StartFeedInW:     3300,
		StopFeedInW:      1500,
		StopDelayMinutes: 10,
		MinOnMinutes:     10,
		MinOffMinutes:    5,
	})
}

var chargingT0 = time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)

func chargingOK(now time.Time, soc, feed float64, plugOn bool) chargingInputs {
	return chargingInputs{Now: now, SocPercent: soc, FeedInW: feed, PlugOn: plugOn, MeterKWh: 2550, ReadOK: true}
}

var chargingTestTuning = chargingTuning{ConfirmTimeout: 2 * time.Minute}

func TestChargingStartsWhenEligible(t *testing.T) {
	state, action := nextChargingState(chargingControllerState{}, chargingOK(chargingT0, 100, 5000, false), chargingTestSettings(), chargingTestTuning)
	if action.SwitchPlug != "on" || !action.StartSession {
		t.Fatalf("expected start, got %+v", action)
	}
	if state.Phase != chargingPhaseSurplus || state.PendingConfirm != "on" {
		t.Fatalf("state = %+v", state)
	}
}

func TestChargingDoesNotStartBelowThresholds(t *testing.T) {
	cases := []chargingInputs{
		chargingOK(chargingT0, 98, 5000, false),  // SOC too low
		chargingOK(chargingT0, 100, 3200, false), // feed-in too low
		chargingOK(chargingT0, 100, 5000, true),  // plug externally on
	}
	for i, in := range cases {
		_, action := nextChargingState(chargingControllerState{}, in, chargingTestSettings(), chargingTestTuning)
		if action.SwitchPlug != "" || action.StartSession {
			t.Fatalf("case %d: unexpected action %+v", i, action)
		}
	}
}

func TestChargingDisabledNeverStarts(t *testing.T) {
	cfg := chargingTestSettings()
	cfg.Enabled = false
	_, action := nextChargingState(chargingControllerState{}, chargingOK(chargingT0, 100, 5000, false), cfg, chargingTestTuning)
	if action.SwitchPlug != "" || action.StartSession {
		t.Fatalf("disabled controller acted: %+v", action)
	}
}

// The core Node-RED race: after switching on, the meter's confirmed state lags.
// The machine must NOT re-evaluate anything until the read-back matches.
func TestChargingConfirmGracePreventsOscillation(t *testing.T) {
	cfg := chargingTestSettings()
	state, _ := nextChargingState(chargingControllerState{ActiveSessionID: "s1", Phase: chargingPhaseSurplus, PendingConfirm: "on", PendingSince: chargingT0, LastSwitchAt: chargingT0}, chargingOK(chargingT0.Add(3*time.Second), 100, 5000, false), cfg, chargingTestTuning)
	// Plug still reads off 3s after the command — old flow reset here.
	if state.Phase != chargingPhaseSurplus || state.PendingConfirm != "on" {
		t.Fatalf("grace period violated: %+v", state)
	}
	// Read-back arrives: pending clears, no action.
	state, action := nextChargingState(state, chargingOK(chargingT0.Add(20*time.Second), 100, 5000, true), cfg, chargingTestTuning)
	if state.PendingConfirm != "" || action.SwitchPlug != "" {
		t.Fatalf("confirm did not clear cleanly: %+v %+v", state, action)
	}
}

func TestChargingConfirmTimeoutRetriesThenGivesUp(t *testing.T) {
	cfg := chargingTestSettings()
	state := chargingControllerState{Phase: chargingPhaseSurplus, ActiveSessionID: "s1", PendingConfirm: "on", PendingSince: chargingT0, LastSwitchAt: chargingT0}
	// First timeout: retry the switch command.
	state, action := nextChargingState(state, chargingOK(chargingT0.Add(3*time.Minute), 100, 5000, false), cfg, chargingTestTuning)
	if action.SwitchPlug != "on" || state.PendingRetries != 1 {
		t.Fatalf("expected retry, got action=%+v state=%+v", action, state)
	}
	// Second timeout: give up, close the session, alert.
	state, action = nextChargingState(state, chargingOK(chargingT0.Add(6*time.Minute), 100, 5000, false), cfg, chargingTestTuning)
	if !action.EndSession || action.EndReason != "confirm-timeout" {
		t.Fatalf("expected give-up, got %+v", action)
	}
	if state.Phase != chargingPhaseIdle || state.PendingConfirm != "" || state.LastError != "confirm-timeout" {
		t.Fatalf("state after give-up: %+v", state)
	}
	failed := false
	for _, event := range action.Events {
		if event.Kind == "confirm-failed" {
			failed = true
		}
	}
	if !failed {
		t.Fatal("expected confirm-failed event")
	}
}

// The old flow's 60-min delay queue is replaced by a sustained-below timer
// that re-arms cleanly when the feed-in recovers.
func TestChargingFeedInStopNeedsSustainedDrop(t *testing.T) {
	cfg := chargingTestSettings()
	base := chargingControllerState{Phase: chargingPhaseSurplus, ActiveSessionID: "s1", LastSwitchAt: chargingT0}

	// Drop below stop threshold: timer starts, no stop yet.
	state, action := nextChargingState(base, chargingOK(chargingT0.Add(20*time.Minute), 100, 800, true), cfg, chargingTestTuning)
	if action.SwitchPlug != "" || state.BelowStopSince.IsZero() {
		t.Fatalf("expected armed stop timer, got %+v %+v", state, action)
	}
	// Recovers after 5 min: timer must clear (re-arm).
	state, action = nextChargingState(state, chargingOK(chargingT0.Add(25*time.Minute), 100, 4000, true), cfg, chargingTestTuning)
	if !state.BelowStopSince.IsZero() || action.SwitchPlug != "" {
		t.Fatalf("stop timer must clear on recovery: %+v", state)
	}
	// Drops again and stays low past the delay: stop fires once.
	state, _ = nextChargingState(state, chargingOK(chargingT0.Add(30*time.Minute), 100, 900, true), cfg, chargingTestTuning)
	state, action = nextChargingState(state, chargingOK(chargingT0.Add(41*time.Minute), 100, 900, true), cfg, chargingTestTuning)
	if action.SwitchPlug != "off" || !action.EndSession || action.EndReason != "feedin-low" {
		t.Fatalf("expected feedin-low stop, got %+v", action)
	}
	if state.Phase != chargingPhaseIdle || state.PendingConfirm != "off" {
		t.Fatalf("state after stop: %+v", state)
	}
}

func TestChargingMinOnBlocksEarlyStop(t *testing.T) {
	cfg := chargingTestSettings()
	state := chargingControllerState{Phase: chargingPhaseSurplus, ActiveSessionID: "s1", LastSwitchAt: chargingT0, BelowStopSince: chargingT0.Add(-20 * time.Minute)}
	// SOC collapsed AND feed-in low past delay, but we switched on 2 min ago.
	_, action := nextChargingState(state, chargingOK(chargingT0.Add(2*time.Minute), 90, 0, true), cfg, chargingTestTuning)
	if action.SwitchPlug != "" || action.EndSession {
		t.Fatalf("min-on violated: %+v", action)
	}
}

func TestChargingMinOffBlocksImmediateRestart(t *testing.T) {
	cfg := chargingTestSettings()
	state := chargingControllerState{Phase: chargingPhaseIdle, LastSwitchAt: chargingT0}
	_, action := nextChargingState(state, chargingOK(chargingT0.Add(2*time.Minute), 100, 5000, false), cfg, chargingTestTuning)
	if action.SwitchPlug != "" {
		t.Fatalf("min-off violated: %+v", action)
	}
	_, action = nextChargingState(state, chargingOK(chargingT0.Add(6*time.Minute), 100, 5000, false), cfg, chargingTestTuning)
	if action.SwitchPlug != "on" {
		t.Fatalf("restart after min-off must work: %+v", action)
	}
}

func TestChargingSocStopAfterMinOn(t *testing.T) {
	cfg := chargingTestSettings()
	state := chargingControllerState{Phase: chargingPhaseSurplus, ActiveSessionID: "s1", LastSwitchAt: chargingT0}
	_, action := nextChargingState(state, chargingOK(chargingT0.Add(30*time.Minute), 90, 5000, true), cfg, chargingTestTuning)
	if action.SwitchPlug != "off" || action.EndReason != "soc-low" {
		t.Fatalf("expected soc-low stop, got %+v", action)
	}
}

func TestChargingExternalOffEndsSession(t *testing.T) {
	cfg := chargingTestSettings()
	state := chargingControllerState{Phase: chargingPhaseSurplus, ActiveSessionID: "s1", LastSwitchAt: chargingT0}
	state, action := nextChargingState(state, chargingOK(chargingT0.Add(30*time.Minute), 100, 5000, false), cfg, chargingTestTuning)
	if !action.EndSession || action.EndReason != "external-off" {
		t.Fatalf("expected external-off end, got %+v", action)
	}
	if state.Phase != chargingPhaseIdle {
		t.Fatalf("phase = %v", state.Phase)
	}
}

func TestChargingManualOffBlocksAutoStart(t *testing.T) {
	cfg := chargingTestSettings()
	state := chargingControllerState{Phase: chargingPhaseManualOff, LastSwitchAt: chargingT0}
	// Perfect surplus conditions, hours later: must stay off.
	_, action := nextChargingState(state, chargingOK(chargingT0.Add(5*time.Hour), 100, 9000, false), cfg, chargingTestTuning)
	if action.SwitchPlug != "" || action.StartSession {
		t.Fatalf("manual_off must block auto start: %+v", action)
	}
}

func TestChargingUnreadableInputsFreezeEverything(t *testing.T) {
	cfg := chargingTestSettings()
	before := chargingControllerState{Phase: chargingPhaseSurplus, ActiveSessionID: "s1", LastSwitchAt: chargingT0, BelowStopSince: chargingT0.Add(10 * time.Minute)}
	after, action := nextChargingState(before, chargingInputs{Now: chargingT0.Add(90 * time.Minute), ReadOK: false}, cfg, chargingTestTuning)
	if action.SwitchPlug != "" || action.EndSession || action.StartSession {
		t.Fatalf("acted on unreadable inputs: %+v", action)
	}
	if after != before {
		t.Fatalf("state advanced on unreadable inputs: %+v -> %+v", before, after)
	}
}

// Full sequence: the exact 2026-07-19 Telegram-spam scenario, replayed against
// the new machine. Sonnen ticks every 3 s, the plug confirms after 9 s. The
// old flow produced a true/false Telegram pair per tick; the new one must
// produce exactly one start and zero further switch commands.
func TestChargingNoOscillationDuringSlowPlugConfirm(t *testing.T) {
	cfg := chargingTestSettings()
	state := chargingControllerState{}
	switches := 0
	events := 0
	plugConfirmAt := chargingT0.Add(9 * time.Second)
	for i := 0; i < 40; i++ {
		now := chargingT0.Add(time.Duration(i*3) * time.Second)
		plugOn := !now.Before(plugConfirmAt)
		var action chargingAction
		state, action = nextChargingState(state, chargingOK(now, 100, 5428, plugOn), cfg, chargingTestTuning)
		if action.SwitchPlug != "" {
			switches++
		}
		for _, event := range action.Events {
			if strings.HasPrefix(event.Kind, "surplus") {
				events++
			}
		}
	}
	if switches != 1 {
		t.Fatalf("expected exactly 1 switch command, got %d", switches)
	}
	if events != 1 {
		t.Fatalf("expected exactly 1 surplus event, got %d", events)
	}
	if state.Phase != chargingPhaseSurplus || state.PendingConfirm != "" {
		t.Fatalf("expected settled surplus state, got %+v", state)
	}
}
