package server

// PP20 Überschussladen: the charging controller that replaced the Node-RED
// "TG Charging" flow. The decision core is a pure state machine
// (nextChargingState) — no clock, no IO — so every race the old flow had is
// covered by table tests. The worker around it polls Home Assistant, applies
// actions, and persists state through the parking store.
//
// Design notes carried over from the Node-RED post-mortem:
//   - The old flow judged the plug's confirmed state instantly after sending
//     the ON command and oscillated true/false on every battery tick. Here a
//     pending-confirm phase suspends all start/stop evaluation until the plug
//     read-back matches (with timeout + one retry).
//   - The old flow queued resets in a delay node; a burst re-fired an hour
//     later. Here stop timers are absolute timestamps in persisted state,
//     re-evaluated fresh on every tick.

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/markus-barta/hausv-org/internal/store"
)

type chargingControllerState = store.ChargingControllerState
type chargingControlSettings = store.ChargingControlSettings

const (
	chargingModeSurplus     = store.ChargingModeSurplus
	chargingModeManual      = store.ChargingModeManual
	chargingPhaseIdle       = store.ChargingPhaseIdle
	chargingPhaseSurplus    = store.ChargingPhaseSurplus
	chargingPhaseManualOn   = store.ChargingPhaseManualOn
	chargingPhaseManualOff  = store.ChargingPhaseManualOff
	chargingTriggerAuto     = store.ChargingTriggerAuto
	chargingTriggerTelegram = store.ChargingTriggerTelegram
	chargingTriggerWeb      = store.ChargingTriggerWeb

	notificationEventCharging   = store.NotificationEventCharging
	auditActionChargingSettings = store.AuditActionChargingSettings
	auditActionChargingManual   = store.AuditActionChargingManual
	auditActionChargingSession  = store.AuditActionChargingSession
)

// chargingInputs is one consistent snapshot of the world. ReadOK=false means
// "act on nothing": HA unreachable, a value unparseable, or sensors stale.
type chargingInputs struct {
	Now        time.Time
	SocPercent float64
	FeedInW    float64
	PlugOn     bool
	MeterKWh   float64
	ReadOK     bool
}

type chargingEvent struct {
	At     time.Time
	Tenant string
	Kind   string
	Detail string
	Shadow bool
}

type chargingAction struct {
	SwitchPlug   string // "", "on", "off"
	StartSession bool
	EndSession   bool
	EndReason    string
	EndedBy      string
	Events       []chargingEvent
}

type chargingTuning struct {
	ConfirmTimeout time.Duration
}

func chargingMinutes(m int) time.Duration { return time.Duration(m) * time.Minute }

// nextChargingState is the pure decision core. It never performs IO; the
// returned action tells the caller what to do.
func nextChargingState(state chargingControllerState, in chargingInputs, cfg chargingControlSettings, tuning chargingTuning) (chargingControllerState, chargingAction) {
	cfg = store.NormalizeChargingControlSettings(cfg)
	state = store.NormalizeChargingControllerState(state)
	action := chargingAction{}
	if !in.ReadOK {
		return state, action
	}

	// Pending confirmation: the switch command is out, the read-back has not
	// matched yet. All other evaluation is suspended — this grace period is
	// what the Node-RED flow lacked.
	if state.PendingConfirm != "" {
		want := state.PendingConfirm == "on"
		if in.PlugOn == want {
			state.PendingConfirm = ""
			state.PendingSince = time.Time{}
			state.PendingRetries = 0
			state.LastError = ""
			return state, action
		}
		if in.Now.Sub(state.PendingSince) < tuning.ConfirmTimeout {
			return state, action
		}
		if state.PendingRetries < 1 {
			state.PendingRetries++
			state.PendingSince = in.Now
			action.SwitchPlug = state.PendingConfirm
			action.Events = append(action.Events, chargingEvent{
				Kind:   "confirm-retry",
				Detail: "Steckdose bestätigt „" + state.PendingConfirm + "“ nicht, zweiter Versuch",
			})
			return state, action
		}
		direction := state.PendingConfirm
		state.LastError = "confirm-timeout"
		state.LastErrorAt = in.Now
		state.PendingConfirm = ""
		state.PendingSince = time.Time{}
		state.PendingRetries = 0
		state.Phase = chargingPhaseIdle
		state.BelowStopSince = time.Time{}
		if state.ActiveSessionID != "" {
			action.EndSession = true
			action.EndReason = "confirm-timeout"
			action.EndedBy = "system"
		}
		action.Events = append(action.Events, chargingEvent{
			Kind:   "confirm-failed",
			Detail: "Steckdose bestätigt „" + direction + "“ auch nach Wiederholung nicht — Automatik pausiert diese Runde",
		})
		return state, action
	}

	switch state.Phase {
	case chargingPhaseSurplus:
		if !in.PlugOn {
			// Turned off underneath us (HA UI, breaker, device reboot).
			state.Phase = chargingPhaseIdle
			state.LastSwitchAt = in.Now
			state.BelowStopSince = time.Time{}
			if state.ActiveSessionID != "" {
				action.EndSession = true
				action.EndReason = "external-off"
				action.EndedBy = "system"
			}
			action.Events = append(action.Events, chargingEvent{Kind: "surplus-end", Detail: "extern ausgeschaltet"})
			return state, action
		}
		if in.FeedInW < cfg.StopFeedInW {
			if state.BelowStopSince.IsZero() {
				state.BelowStopSince = in.Now
			}
		} else {
			state.BelowStopSince = time.Time{}
		}
		stopReason := ""
		stopDetail := ""
		if in.SocPercent < cfg.StopSocPercent {
			stopReason = "soc-low"
			stopDetail = fmt.Sprintf("Akku unter %.0f %%", cfg.StopSocPercent)
		} else if !state.BelowStopSince.IsZero() && in.Now.Sub(state.BelowStopSince) >= chargingMinutes(cfg.StopDelayMinutes) {
			stopReason = "feedin-low"
			stopDetail = fmt.Sprintf("Einspeisung seit %d min unter %.0f W", cfg.StopDelayMinutes, cfg.StopFeedInW)
		}
		if stopReason != "" && in.Now.Sub(state.LastSwitchAt) >= chargingMinutes(cfg.MinOnMinutes) {
			state.Phase = chargingPhaseIdle
			state.PendingConfirm = "off"
			state.PendingSince = in.Now
			state.PendingRetries = 0
			state.LastSwitchAt = in.Now
			state.BelowStopSince = time.Time{}
			action.SwitchPlug = "off"
			action.EndSession = true
			action.EndReason = stopReason
			action.EndedBy = "system"
			action.Events = append(action.Events, chargingEvent{Kind: "surplus-end", Detail: stopDetail})
		}
		return state, action

	case chargingPhaseManualOn:
		if !in.PlugOn {
			state.Phase = chargingPhaseIdle
			state.LastSwitchAt = in.Now
			if state.ActiveSessionID != "" {
				action.EndSession = true
				action.EndReason = "external-off"
				action.EndedBy = "system"
			}
			action.Events = append(action.Events, chargingEvent{Kind: "manual-end", Detail: "extern ausgeschaltet"})
		}
		return state, action

	case chargingPhaseManualOff:
		// Explicitly parked: automatic starts stay blocked until /pp20auto.
		return state, action

	default: // idle
		if !cfg.Enabled {
			return state, action
		}
		if in.PlugOn {
			// Externally switched on — not our session, bills at normal tariff.
			return state, action
		}
		if in.SocPercent < cfg.StartSocPercent || in.FeedInW < cfg.StartFeedInW {
			return state, action
		}
		if !state.LastSwitchAt.IsZero() && in.Now.Sub(state.LastSwitchAt) < chargingMinutes(cfg.MinOffMinutes) {
			return state, action
		}
		state.Phase = chargingPhaseSurplus
		state.PendingConfirm = "on"
		state.PendingSince = in.Now
		state.PendingRetries = 0
		state.LastSwitchAt = in.Now
		state.BelowStopSince = time.Time{}
		action.SwitchPlug = "on"
		action.StartSession = true
		action.Events = append(action.Events, chargingEvent{
			Kind:   "surplus-start",
			Detail: fmt.Sprintf("Akku %.0f %%, Einspeisung %.0f W", in.SocPercent, in.FeedInW),
		})
		return state, action
	}
}

// ── event ring (diagnostics for the admin UI) ───────────────────────────────

type chargingEventRing struct {
	mu    sync.Mutex
	items []chargingEvent
}

const chargingEventRingCap = 200

func (r *chargingEventRing) add(events ...chargingEvent) {
	if r == nil || len(events) == 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = append(r.items, events...)
	if overflow := len(r.items) - chargingEventRingCap; overflow > 0 {
		r.items = append([]chargingEvent(nil), r.items[overflow:]...)
	}
}

// list returns events newest-first.
func (r *chargingEventRing) list(tenantSlug string, limit int) []chargingEvent {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []chargingEvent{}
	for i := len(r.items) - 1; i >= 0 && (limit <= 0 || len(out) < limit); i-- {
		if tenantSlug == "" || r.items[i].Tenant == tenantSlug {
			out = append(out, r.items[i])
		}
	}
	return out
}

// ── worker ──────────────────────────────────────────────────────────────────

func (a *app) StartChargingController() func() { return a.startChargingController() }

func (a *app) startChargingController() func() {
	if a.chargingTickInterval <= 0 {
		log.Printf("charging controller disabled")
		return func() {}
	}
	configured := 0
	for _, tenant := range a.tenants {
		if tenant.HA.ChargingConfigured() {
			configured++
		}
	}
	if configured == 0 {
		log.Printf("charging controller disabled: no tenant with charging entities")
		return func() {}
	}
	log.Printf("charging controller enabled for %d tenant(s), tick %s", configured, a.chargingTickInterval)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		a.recoverChargingTenants(ctx)
		a.tickChargingTenants(ctx)
		ticker := time.NewTicker(a.chargingTickInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.tickChargingTenants(ctx)
			}
		}
	}()
	return cancel
}

// recoverChargingTenants reconciles persisted state with reality after a
// restart or redeploy: an open session with the plug still on resumes
// silently; with the plug off it is closed at the last stored meter sample.
func (a *app) recoverChargingTenants(ctx context.Context) {
	for _, tenant := range a.tenants {
		if !tenant.HA.ChargingConfigured() {
			continue
		}
		a.chargingMu.Lock()
		data := a.parkingStore.TenantData(tenant.Slug)
		state := data.Charging
		if state.ActiveSessionID == "" {
			a.chargingMu.Unlock()
			continue
		}
		cctx, cancel := context.WithTimeout(ctx, 8*time.Second)
		plug, err := tenant.HA.State(cctx, tenant.HA.PlugSwitchEntity())
		cancel()
		if err != nil {
			log.Printf("charging recovery deferred for %s: %v", tenant.Slug, err)
			a.chargingMu.Unlock()
			continue
		}
		if strings.EqualFold(plug.State, "on") {
			log.Printf("charging recovery: resuming open session %s for %s", state.ActiveSessionID, tenant.Slug)
			a.chargingMu.Unlock()
			continue
		}
		end := time.Now().UTC()
		endKWh := 0.0
		if n := len(data.EnergySamples); n > 0 {
			last := data.EnergySamples[n-1]
			endKWh = last.Value
			if !last.At.IsZero() && last.At.Before(end) {
				end = last.At
			}
		}
		state.Phase = chargingPhaseIdle
		state.PendingConfirm = ""
		state.PendingSince = time.Time{}
		state.PendingRetries = 0
		if _, _, err := a.parkingStore.EndChargingSession(tenant.Slug, state.ActiveSessionID, end, endKWh, "system", "restart", state); err != nil {
			log.Printf("charging recovery close failed for %s: %v", tenant.Slug, err)
		} else {
			log.Printf("charging recovery: closed session %s for %s (plug off after restart)", state.ActiveSessionID, tenant.Slug)
		}
		a.chargingMu.Unlock()
	}
}

func (a *app) tickChargingTenants(ctx context.Context) {
	for _, tenant := range a.tenants {
		if !tenant.HA.ChargingConfigured() {
			continue
		}
		a.tickChargingTenant(ctx, tenant)
	}
}

func (a *app) tickChargingTenant(ctx context.Context, tenant tenantConfig) {
	a.chargingMu.Lock()
	defer a.chargingMu.Unlock()
	data := a.parkingStore.TenantData(tenant.Slug)
	cfg := data.Settings.Charging
	if !cfg.Enabled {
		return
	}
	in := a.readChargingInputs(ctx, tenant)
	if !in.ReadOK {
		a.noteChargingReadFailure(tenant, data.Charging, in.Now)
		return
	}
	a.chargingLastPoll[tenant.Slug] = in.Now
	a.noteChargingReadRecovery(tenant, data.Charging, in.Now)

	if cfg.ShadowMode {
		a.tickChargingShadow(tenant, in, cfg)
		return
	}

	prev := data.Charging
	next, action := nextChargingState(prev, in, cfg, chargingTuning{ConfirmTimeout: a.chargingConfirmTimeout})
	a.applyChargingAction(ctx, tenant, prev, next, action, in)
}

// tickChargingShadow runs the same machine against an in-memory state and
// only logs what it would do. It never switches, never creates sessions,
// never persists — Node-RED (or nothing) is still in control. The machine
// sees a VIRTUAL plug that follows the shadow decisions; the real plug
// belongs to the old system and would only confuse the state machine.
func (a *app) tickChargingShadow(tenant tenantConfig, in chargingInputs, cfg chargingControlSettings) {
	state := a.chargingShadow[tenant.Slug]
	in.PlugOn = a.chargingShadowPlug[tenant.Slug]
	next, action := nextChargingState(state, in, cfg, chargingTuning{ConfirmTimeout: a.chargingConfirmTimeout})
	if action.SwitchPlug != "" {
		// Virtual plug confirms instantly.
		a.chargingShadowPlug[tenant.Slug] = action.SwitchPlug == "on"
		next.PendingConfirm = ""
		next.PendingSince = time.Time{}
		next.PendingRetries = 0
	}
	a.chargingShadow[tenant.Slug] = next
	for _, event := range action.Events {
		event.At = in.Now
		event.Tenant = tenant.Slug
		event.Shadow = true
		event.Detail = "[Test] " + event.Detail
		a.chargingEvents.add(event)
		log.Printf("charging shadow %s: %s — %s", tenant.Slug, event.Kind, event.Detail)
	}
	if action.SwitchPlug != "" {
		event := chargingEvent{
			At:     in.Now,
			Tenant: tenant.Slug,
			Kind:   "shadow-switch",
			Detail: "[Test] würde Steckdose „" + action.SwitchPlug + "“ schalten",
			Shadow: true,
		}
		a.chargingEvents.add(event)
		log.Printf("charging shadow %s: %s", tenant.Slug, event.Detail)
	}
}

func (a *app) applyChargingAction(ctx context.Context, tenant tenantConfig, prev, next chargingControllerState, action chargingAction, in chargingInputs) {
	slug := tenant.Slug
	statePersisted := false
	switch {
	case action.EndSession && prev.ActiveSessionID != "":
		closed, found, err := a.parkingStore.EndChargingSession(slug, prev.ActiveSessionID, in.Now, in.MeterKWh, action.EndedBy, action.EndReason, next)
		if err != nil {
			log.Printf("charging session end failed for %s: %v", slug, err)
			return
		}
		statePersisted = true
		a.appendChargingBoundarySample(slug, in)
		if found {
			a.recordChargingSessionAudit(tenant, closed, "beendet ("+action.EndReason+")")
			if closed.Mode == chargingModeSurplus {
				a.notifyChargingSessionEnd(tenant, closed, action.EndReason)
			}
		}
	case action.StartSession:
		session, err := a.parkingStore.StartChargingSession(slug, chargingSession{
			Start:         in.Now,
			StartKWh:      in.MeterKWh,
			Mode:          chargingModeSurplus,
			TriggerSource: chargingTriggerAuto,
			StartedBy:     "system",
		}, next)
		if err != nil {
			log.Printf("charging session start failed for %s: %v", slug, err)
			return
		}
		statePersisted = true
		a.appendChargingBoundarySample(slug, in)
		a.recordChargingSessionAudit(tenant, session, "gestartet")
		a.notifyChargingSessionStart(tenant, in)
	}
	if !statePersisted && next != prev {
		if err := a.parkingStore.SetChargingState(slug, next); err != nil {
			log.Printf("charging state save failed for %s: %v", slug, err)
		}
	}
	if action.SwitchPlug != "" {
		on := action.SwitchPlug == "on"
		cctx, cancel := context.WithTimeout(ctx, 8*time.Second)
		if err := tenant.HA.SetSwitch(cctx, tenant.HA.PlugSwitchEntity(), on); err != nil {
			// The pending-confirm loop notices the missing read-back and
			// retries / gives up on its own.
			log.Printf("charging switch %s failed for %s: %v", action.SwitchPlug, slug, err)
		}
		cancel()
	}
	for _, event := range action.Events {
		event.At = in.Now
		event.Tenant = slug
		a.chargingEvents.add(event)
		log.Printf("charging %s: %s — %s", slug, event.Kind, event.Detail)
		if event.Kind == "confirm-failed" {
			a.notifyChargingError(tenant, event.Detail)
		}
	}
}

// appendChargingBoundarySample stores the meter reading at a session
// boundary so billing intervals align exactly with session edges.
func (a *app) appendChargingBoundarySample(slug string, in chargingInputs) {
	if err := a.parkingStore.AppendReadings(slug, []parkingNumericSample{{At: in.Now, Value: in.MeterKWh}}, nil); err != nil {
		log.Printf("charging boundary sample failed for %s: %v", slug, err)
	}
}

func (a *app) readChargingInputs(ctx context.Context, tenant tenantConfig) chargingInputs {
	in := chargingInputs{Now: time.Now().UTC()}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	ha := tenant.HA
	plug, err := ha.State(ctx, ha.PlugSwitchEntity())
	if err != nil {
		return in
	}
	plugState := strings.ToLower(strings.TrimSpace(plug.State))
	if plugState != "on" && plugState != "off" {
		return in
	}
	soc, err := ha.State(ctx, ha.BatterySocEntity())
	if err != nil {
		return in
	}
	feed, err := ha.State(ctx, ha.GridFeedInEntity())
	if err != nil {
		return in
	}
	meter, err := ha.State(ctx, ha.MeterEnergyEntity())
	if err != nil {
		return in
	}
	socValue, err := parseHAFloat(soc.State)
	if err != nil {
		return in
	}
	feedValue, err := parseHAFloat(feed.State)
	if err != nil {
		return in
	}
	meterValue, err := parseHAFloat(meter.State)
	if err != nil {
		return in
	}
	if a.chargingStaleAfter > 0 {
		for _, entity := range []haState{soc, feed} {
			updated := entity.LastUpdated
			if updated.IsZero() {
				updated = entity.LastChanged
			}
			if !updated.IsZero() && in.Now.Sub(updated) > a.chargingStaleAfter {
				return in
			}
		}
	}
	in.SocPercent = socValue
	in.FeedInW = feedValue
	in.MeterKWh = meterValue
	in.PlugOn = plugState == "on"
	in.ReadOK = true
	return in
}

// ── HA failure accounting (in-memory count, persisted once-only alert) ──────

func (a *app) noteChargingReadFailure(tenant tenantConfig, state chargingControllerState, now time.Time) {
	a.chargingHAFails[tenant.Slug]++
	fails := a.chargingHAFails[tenant.Slug]
	if fails != a.chargingHAFailLimit {
		return
	}
	if !state.ErrorNotifiedAt.IsZero() {
		return
	}
	state.LastError = "ha-unreachable"
	state.LastErrorAt = now
	state.ErrorNotifiedAt = now
	if err := a.parkingStore.SetChargingState(tenant.Slug, state); err != nil {
		log.Printf("charging error state save failed for %s: %v", tenant.Slug, err)
	}
	a.chargingEvents.add(chargingEvent{At: now, Tenant: tenant.Slug, Kind: "ha-unreachable", Detail: "Home Assistant liefert keine verwertbaren Daten"})
	a.notifyChargingError(tenant, "Home Assistant ist nicht erreichbar oder liefert veraltete Werte. Die Ladesteuerung pausiert, bis wieder Daten kommen.")
}

func (a *app) noteChargingReadRecovery(tenant tenantConfig, state chargingControllerState, now time.Time) {
	if a.chargingHAFails[tenant.Slug] == 0 {
		return
	}
	hadAlert := !state.ErrorNotifiedAt.IsZero()
	a.chargingHAFails[tenant.Slug] = 0
	if !hadAlert {
		return
	}
	state.LastError = ""
	state.ErrorNotifiedAt = time.Time{}
	if err := a.parkingStore.SetChargingState(tenant.Slug, state); err != nil {
		log.Printf("charging recovery state save failed for %s: %v", tenant.Slug, err)
	}
	a.chargingEvents.add(chargingEvent{At: now, Tenant: tenant.Slug, Kind: "ha-recovered", Detail: "Home Assistant wieder erreichbar"})
	a.notifyChargingError(tenant, "Home Assistant ist wieder erreichbar — die Ladesteuerung läuft normal weiter.")
}

// ── manual override (Telegram + web share this path) ────────────────────────

type chargingCommandResult struct {
	OK      bool
	Message string
}

func (a *app) requestManualCharging(ctx context.Context, tenant tenantConfig, on bool, actorEmail, source string) chargingCommandResult {
	a.chargingMu.Lock()
	defer a.chargingMu.Unlock()
	if !tenant.HA.ChargingConfigured() {
		return chargingCommandResult{Message: "Die Ladesteuerung ist nicht konfiguriert."}
	}
	data := a.parkingStore.TenantData(tenant.Slug)
	cfg := store.NormalizeChargingControlSettings(data.Settings.Charging)
	if cfg.Enabled && cfg.ShadowMode {
		return chargingCommandResult{Message: "Testbetrieb aktiv — die Steuerung läuft noch über das alte System."}
	}
	in := a.readChargingInputs(ctx, tenant)
	if !in.ReadOK {
		return chargingCommandResult{Message: "Home Assistant ist derzeit nicht erreichbar. Bitte später erneut versuchen."}
	}
	state := data.Charging
	if on {
		if in.PlugOn {
			if state.Phase == chargingPhaseSurplus {
				return chargingCommandResult{OK: true, Message: "Überschussladen läuft bereits."}
			}
			return chargingCommandResult{OK: true, Message: "Die Ladung ist bereits eingeschaltet."}
		}
		if wait := chargingMinutes(cfg.MinOffMinutes) - in.Now.Sub(state.LastSwitchAt); !state.LastSwitchAt.IsZero() && wait > 0 {
			return chargingCommandResult{Message: fmt.Sprintf("Bitte noch %s warten (Schaltpause zum Schutz der Steckdose).", chargingWaitLabel(wait))}
		}
		state.Phase = chargingPhaseManualOn
		state.PendingConfirm = "on"
		state.PendingSince = in.Now
		state.PendingRetries = 0
		state.LastSwitchAt = in.Now
		state.BelowStopSince = time.Time{}
		if _, err := a.parkingStore.StartChargingSession(tenant.Slug, chargingSession{
			Start:         in.Now,
			StartKWh:      in.MeterKWh,
			Mode:          chargingModeManual,
			TriggerSource: source,
			StartedBy:     actorEmail,
		}, state); err != nil {
			log.Printf("manual charging start failed for %s: %v", tenant.Slug, err)
			return chargingCommandResult{Message: "Speichern fehlgeschlagen. Bitte erneut versuchen."}
		}
		a.appendChargingBoundarySample(tenant.Slug, in)
		a.switchChargingPlug(ctx, tenant, true)
		a.recordChargingManualAudit(tenant, actorEmail, source, "eingeschaltet")
		return chargingCommandResult{OK: true, Message: "Ladung eingeschaltet — Normaltarif aktiv."}
	}

	// off
	if wait := chargingMinutes(cfg.MinOnMinutes) - in.Now.Sub(state.LastSwitchAt); in.PlugOn && !state.LastSwitchAt.IsZero() && wait > 0 && state.ActiveSessionID != "" {
		return chargingCommandResult{Message: fmt.Sprintf("Bitte noch %s warten (Mindest-Einschaltdauer).", chargingWaitLabel(wait))}
	}
	nextState := state
	nextState.Phase = chargingPhaseManualOff
	nextState.PendingConfirm = "off"
	nextState.PendingSince = in.Now
	nextState.PendingRetries = 0
	nextState.LastSwitchAt = in.Now
	nextState.BelowStopSince = time.Time{}
	if state.ActiveSessionID != "" {
		closed, found, err := a.parkingStore.EndChargingSession(tenant.Slug, state.ActiveSessionID, in.Now, in.MeterKWh, actorEmail, "manual", nextState)
		if err != nil {
			log.Printf("manual charging end failed for %s: %v", tenant.Slug, err)
			return chargingCommandResult{Message: "Speichern fehlgeschlagen. Bitte erneut versuchen."}
		}
		a.appendChargingBoundarySample(tenant.Slug, in)
		if found && closed.Mode == chargingModeSurplus {
			a.notifyChargingSessionEnd(tenant, closed, "manual")
		}
	} else if err := a.parkingStore.SetChargingState(tenant.Slug, nextState); err != nil {
		log.Printf("manual charging state save failed for %s: %v", tenant.Slug, err)
		return chargingCommandResult{Message: "Speichern fehlgeschlagen. Bitte erneut versuchen."}
	}
	a.switchChargingPlug(ctx, tenant, false)
	a.recordChargingManualAudit(tenant, actorEmail, source, "ausgeschaltet")
	return chargingCommandResult{OK: true, Message: "Ladung ausgeschaltet. Die Automatik bleibt pausiert — „Automatik“ aktiviert sie wieder."}
}

func (a *app) requestAutomaticCharging(ctx context.Context, tenant tenantConfig, actorEmail, source string) chargingCommandResult {
	a.chargingMu.Lock()
	defer a.chargingMu.Unlock()
	data := a.parkingStore.TenantData(tenant.Slug)
	state := data.Charging
	switch state.Phase {
	case chargingPhaseManualOff:
		state.Phase = chargingPhaseIdle
		if err := a.parkingStore.SetChargingState(tenant.Slug, state); err != nil {
			log.Printf("charging auto-resume failed for %s: %v", tenant.Slug, err)
			return chargingCommandResult{Message: "Speichern fehlgeschlagen. Bitte erneut versuchen."}
		}
		a.recordChargingManualAudit(tenant, actorEmail, source, "Automatik aktiviert")
		return chargingCommandResult{OK: true, Message: "Automatik wieder aktiv."}
	case chargingPhaseManualOn:
		in := a.readChargingInputs(ctx, tenant)
		if !in.ReadOK {
			return chargingCommandResult{Message: "Home Assistant ist derzeit nicht erreichbar. Bitte später erneut versuchen."}
		}
		nextState := state
		nextState.Phase = chargingPhaseIdle
		nextState.PendingConfirm = "off"
		nextState.PendingSince = in.Now
		nextState.PendingRetries = 0
		nextState.LastSwitchAt = in.Now
		if state.ActiveSessionID != "" {
			if _, _, err := a.parkingStore.EndChargingSession(tenant.Slug, state.ActiveSessionID, in.Now, in.MeterKWh, actorEmail, "auto-resume", nextState); err != nil {
				log.Printf("charging auto-resume end failed for %s: %v", tenant.Slug, err)
				return chargingCommandResult{Message: "Speichern fehlgeschlagen. Bitte erneut versuchen."}
			}
			a.appendChargingBoundarySample(tenant.Slug, in)
		} else if err := a.parkingStore.SetChargingState(tenant.Slug, nextState); err != nil {
			return chargingCommandResult{Message: "Speichern fehlgeschlagen. Bitte erneut versuchen."}
		}
		a.switchChargingPlug(ctx, tenant, false)
		a.recordChargingManualAudit(tenant, actorEmail, source, "Automatik aktiviert, manuelle Ladung beendet")
		return chargingCommandResult{OK: true, Message: "Manuelle Ladung beendet — die Automatik übernimmt wieder."}
	default:
		return chargingCommandResult{OK: true, Message: "Automatik ist bereits aktiv."}
	}
}

func (a *app) switchChargingPlug(ctx context.Context, tenant tenantConfig, on bool) {
	cctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	if err := tenant.HA.SetSwitch(cctx, tenant.HA.PlugSwitchEntity(), on); err != nil {
		log.Printf("charging manual switch failed for %s: %v", tenant.Slug, err)
	}
}

func chargingWaitLabel(wait time.Duration) string {
	minutes := int(wait.Round(time.Minute) / time.Minute)
	if minutes < 1 {
		minutes = 1
	}
	if minutes == 1 {
		return "1 Minute"
	}
	return fmt.Sprintf("%d Minuten", minutes)
}

// ── notifications & audit ───────────────────────────────────────────────────

func (a *app) chargingNotificationRecipients(tenant tenantConfig) []string {
	recipients := []string{}
	for email, profile := range a.profiles {
		if profile.HasTenant(tenant.Slug) && profile.HasPermission(permissionParking) {
			recipients = append(recipients, email)
		}
	}
	for email := range a.admins {
		recipients = append(recipients, email)
	}
	return uniqueEmails(recipients)
}

func (a *app) notifyChargingSessionStart(tenant tenantConfig, in chargingInputs) {
	a.notifyCharging(tenant, a.chargingNotificationRecipients(tenant),
		"PV-Überschussladen gestartet",
		[]string{
			"Parkplatz 20 lädt jetzt mit Sonnenstrom (" + formatEURPerKWh(a.chargingSurplusRateNow(tenant)) + ").",
			fmt.Sprintf("Hausakku: %.0f %% · Einspeisung: %.0f W", in.SocPercent, in.FeedInW),
		})
}

func (a *app) notifyChargingSessionEnd(tenant tenantConfig, session chargingSession, reason string) {
	kWh := session.EndKWh - session.StartKWh
	cost := kWh * a.chargingSurplusRateNow(tenant)
	duration := session.End.Sub(session.Start).Round(time.Minute)
	a.notifyCharging(tenant, a.chargingNotificationRecipients(tenant),
		"PV-Überschussladen beendet",
		[]string{
			fmt.Sprintf("Geladen: %s in %s.", formatKWh(kWh), formatChargingDuration(duration)),
			"Kosten: " + formatEUR(cost) + " (" + chargingEndReasonLabel(reason) + ")",
		})
}

func (a *app) notifyChargingError(tenant tenantConfig, detail string) {
	recipients := []string{}
	for email := range a.admins {
		recipients = append(recipients, email)
	}
	a.notifyCharging(tenant, uniqueEmails(recipients), "Ladesteuerung: Störung", []string{detail})
}

// notifyCharging fans out through the existing notification stack (email
// prefs apply) and, in parallel, to the recipients' linked Telegram chats.
func (a *app) notifyCharging(tenant tenantConfig, recipients []string, subject string, lines []string) {
	a.notify(portalNotification{
		Event:      notificationEventCharging,
		Tenant:     tenant,
		Recipients: recipients,
		Subject:    subject,
		Lines:      lines,
	})
	a.telegramChargingBroadcast(recipients, subject, lines)
}

func (a *app) chargingSurplusRateNow(tenant tenantConfig) float64 {
	data := a.parkingStore.TenantData(tenant.Slug)
	return surplusRate(parkingTariffAt(data.Settings, time.Now(), time.Local))
}

func formatChargingDuration(d time.Duration) string {
	if d < time.Minute {
		d = time.Minute
	}
	hours := int(d / time.Hour)
	minutes := int((d % time.Hour) / time.Minute)
	if hours == 0 {
		return fmt.Sprintf("%d min", minutes)
	}
	return fmt.Sprintf("%d h %02d min", hours, minutes)
}

func chargingEndReasonLabel(reason string) string {
	switch reason {
	case "feedin-low":
		return "Einspeisung zu niedrig"
	case "soc-low":
		return "Akkustand gesunken"
	case "manual":
		return "manuell beendet"
	case "external-off":
		return "extern ausgeschaltet"
	case "confirm-timeout":
		return "Steckdose reagierte nicht"
	case "auto-resume":
		return "Automatik übernommen"
	case "restart":
		return "Neustart der Plattform"
	default:
		return reason
	}
}

func (a *app) recordChargingSessionAudit(tenant tenantConfig, session chargingSession, what string) {
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: "system",
		Action:     auditActionChargingSession,
		TargetType: "charging",
		TargetID:   session.ID,
		Summary:    "Ladevorgang " + what,
		Details: map[string]string{
			"mode":    session.Mode,
			"trigger": session.TriggerSource,
		},
	})
}

func (a *app) recordChargingManualAudit(tenant tenantConfig, actorEmail, source, what string) {
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: actorEmail,
		Action:     auditActionChargingManual,
		TargetType: "charging",
		TargetID:   tenant.Slug,
		Summary:    "Ladung " + what,
		Details:    map[string]string{"source": source},
	})
}
