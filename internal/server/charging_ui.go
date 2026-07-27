package server

// Web surface for PP20 charging: the live card on /app/parking, the admin
// panels on /app/parking/settings, and the manual/settings/telegram actions.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/markus-barta/hausv-org/internal/store"
	"github.com/markus-barta/hausv-org/internal/view"
)

var errInvalidChargingForm = errors.New("invalid charging settings")

type parkingLiveView = view.ParkingLiveView
type parkingSplitView = view.ParkingSplitView
type chargingSessionView = view.ChargingSessionView
type parkingChargingAdminStrip = view.ParkingChargingAdminStrip
type parkingChargingAdminView = view.ParkingChargingAdminView
type chargingEventView = view.ChargingEventView
type telegramStatusView = view.TelegramStatusView
type telegramChatView = view.TelegramChatView
type telegramLinkOption = view.TelegramLinkOption

// ── view builders ───────────────────────────────────────────────────────────

func (a *app) chargingLiveView(ctx context.Context, tenant tenantConfig, isAdmin, canToggle bool) parkingLiveView {
	out := parkingLiveView{}
	if !tenant.HA.ChargingConfigured() {
		return out
	}
	out.Available = true
	out.CanToggle = canToggle
	data := a.parkingStore.TenantData(tenant.Slug)
	cfg := store.NormalizeChargingControlSettings(data.Settings.Charging)
	state := data.Charging
	out.Enabled = cfg.Enabled
	out.ShadowMode = cfg.Enabled && cfg.ShadowMode
	now := time.Now()
	loc := time.Local

	in := a.readChargingInputs(ctx, tenant)
	out.StaleData = !in.ReadOK
	if in.ReadOK {
		out.PlugOn = in.PlugOn
		out.ToggleOn = in.PlugOn
		out.FeedInLabel = formatWatt(in.FeedInW)
		out.BatterySOCLabel = strconv.FormatFloat(in.SocPercent, 'f', 0, 64) + " %"
		switch {
		case in.SocPercent >= cfg.StartSocPercent:
			out.BatteryClass = "full"
			out.BatteryHint = "Hausakku voll — Überschuss verfügbar"
		case in.SocPercent >= cfg.StopSocPercent:
			out.BatteryClass = "partial"
			out.BatteryHint = "Hausakku nicht ganz voll — Laden nutzt die Hausreserve"
		default:
			out.BatteryClass = "low"
			out.BatteryHint = "Hausakku niedrig — kein Überschuss"
		}
	}
	if power, err := tenant.HA.State(ctx, tenant.HA.PowerEntity()); err == nil {
		if value, err := parseHAFloat(power.State); err == nil {
			out.PowerLabel = formatWatt(value)
		}
	}

	tariff := parkingTariffAt(data.Settings, now, loc)
	spotRate := 0.0
	if n := len(data.PriceSamples); n > 0 {
		spotRate = data.PriceSamples[n-1].Value
	}
	switch {
	case state.Phase == chargingPhaseSurplus:
		out.Mode = "surplus"
		out.ModeLabel = "Überschussladen aktiv"
		out.ModeClass = "live-surplus"
		out.RateLabel = formatEURPerKWh(surplusRate(tariff))
		out.ModeDetail = "Sonnenstrom zum Fixpreis"
	case state.Phase == chargingPhaseManualOn || (in.ReadOK && in.PlugOn):
		out.Mode = "manual"
		out.ModeLabel = "Normalladen"
		out.ModeClass = "live-manual"
		out.RateLabel = formatEURPerKWh(spotRate + tariff.GridFeeEURPerKWh)
		out.ModeDetail = "aktueller aWATTar-Preis + Netzgebühr"
	case state.Phase == chargingPhaseManualOff:
		out.Mode = "off"
		out.ModeLabel = "Nicht ladend"
		out.ModeClass = "live-idle"
		out.ModeDetail = "Automatik pausiert (manuell ausgeschaltet)"
		out.AutoPaused = true
	default:
		out.Mode = "idle"
		out.ModeLabel = "Nicht ladend"
		out.ModeClass = "live-idle"
		if cfg.Enabled {
			out.ModeDetail = "Automatik wartet auf PV-Überschuss"
		} else {
			out.ModeDetail = "Automatik deaktiviert"
		}
	}

	if state.ActiveSessionID != "" {
		for _, session := range data.ChargingSessions {
			if session.ID != state.ActiveSessionID {
				continue
			}
			out.HasSession = true
			out.SessionSince = formatDateTimeIn(session.Start, loc, deATTimeLayout)
			if in.ReadOK {
				kWh := in.MeterKWh - session.StartKWh
				if kWh < 0 {
					kWh = 0
				}
				out.SessionKWh = formatKWh(kWh)
				out.SessionCost = formatEUR(kWh * a.chargingSessionRate(data, session))
			}
			break
		}
	}

	hours := calculateParkingHourlyUsageWithSettings(data.EnergySamples, data.PriceSamples, data.ChargingSessions, data.Settings, now, loc)
	out.TodaySplit = chargingSplitFor(hours, now, loc, "Heute", func(at time.Time) bool {
		y1, m1, d1 := at.In(loc).Date()
		y2, m2, d2 := now.In(loc).Date()
		return y1 == y2 && m1 == m2 && d1 == d2
	})
	month := now.In(loc).Format("2006-01")
	out.MonthSplit = chargingSplitFor(hours, now, loc, "Monat", func(at time.Time) bool {
		return at.In(loc).Format("2006-01") == month
	})

	out.Sessions = chargingSessionViews(data, 8, "")
	out.HasSessions = len(out.Sessions) > 0

	if isAdmin {
		out.Admin = a.chargingAdminStrip(tenant, state, cfg)
	}
	return out
}

func chargingSplitFor(hours []parkingHourUsage, now time.Time, loc *time.Location, label string, include func(time.Time) bool) parkingSplitView {
	surplusKWh, normalKWh, surplusCost, normalCost := 0.0, 0.0, 0.0, 0.0
	for _, hour := range hours {
		if !include(hour.At) {
			continue
		}
		surplusKWh += hour.SurplusKWh
		normalKWh += hour.KWh - hour.SurplusKWh
		surplusCost += hour.SurplusCost
		normalCost += hour.EnergyCost + hour.GridCost
	}
	total := surplusKWh + normalKWh
	pct := 0
	if total > 0 {
		pct = int(surplusKWh / total * 100)
	}
	return parkingSplitView{
		Label:       label,
		SurplusKWh:  formatKWh(surplusKWh),
		NormalKWh:   formatKWh(normalKWh),
		SurplusCost: formatEUR(surplusCost),
		NormalCost:  formatEUR(normalCost),
		SurplusPct:  pct,
		HasAny:      total > 0.005,
	}
}

// chargingSessionViews renders sessions newest-first; monthFilter "" = all.
func chargingSessionViews(data parkingTenantData, limit int, monthFilter string) []chargingSessionView {
	loc := time.Local
	sessions := append([]chargingSession(nil), data.ChargingSessions...)
	sort.Slice(sessions, func(i, j int) bool { return sessions[i].Start.After(sessions[j].Start) })
	out := []chargingSessionView{}
	for _, session := range sessions {
		if monthFilter != "" && session.Start.In(loc).Format("2006-01") != monthFilter {
			continue
		}
		if limit > 0 && len(out) >= limit {
			break
		}
		item := chargingSessionView{
			StartLabel: formatDateTimeIn(session.Start, loc, deATShortDateTimeLayout),
			ModeLabel:  chargingModeLabel(session.Mode),
			ModeClass:  "mode-normal",
			Active:     session.End.IsZero(),
		}
		if session.Mode == chargingModeSurplus {
			item.ModeClass = "mode-surplus"
		}
		if item.Active {
			item.DurationLabel = "läuft"
		} else {
			item.DurationLabel = formatChargingDuration(session.End.Sub(session.Start))
			kWh := session.EndKWh - session.StartKWh
			item.KWh = formatKWh(kWh)
			rate := 0.0
			if session.Mode == chargingModeSurplus {
				rate = store.SurplusRate(parkingTariffAt(data.Settings, session.Start, loc))
			}
			if rate > 0 {
				item.Cost = formatEUR(kWh * rate)
			}
		}
		out = append(out, item)
	}
	return out
}

func (a *app) chargingAdminStrip(tenant tenantConfig, state chargingControllerState, cfg chargingControlSettings) parkingChargingAdminStrip {
	strip := parkingChargingAdminStrip{Show: true}
	strip.PhaseLabel = chargingPhaseLabel(state.Phase)
	if !state.LastSwitchAt.IsZero() {
		strip.SinceLabel = formatDateTimeIn(state.LastSwitchAt, time.Local, deATShortDateTimeLayout)
	}
	strip.ShadowPill = cfg.Enabled && cfg.ShadowMode
	if state.LastError != "" {
		strip.ErrorDetail = chargingEndReasonLabel(state.LastError)
	}
	a.chargingMu.Lock()
	lastPoll := a.chargingLastPoll[tenant.Slug]
	a.chargingMu.Unlock()
	if !lastPoll.IsZero() {
		age := time.Since(lastPoll).Round(time.Second)
		strip.PollLabel = "vor " + age.String()
		strip.StalePill = age > 3*a.chargingTickInterval && a.chargingTickInterval > 0
	}
	for _, event := range a.chargingEvents.list(tenant.Slug, 1) {
		strip.LastReason = event.Detail
	}
	return strip
}

func chargingPhaseLabel(phase string) string {
	switch phase {
	case chargingPhaseSurplus:
		return "Überschussladen"
	case chargingPhaseManualOn:
		return "Manuell ein"
	case chargingPhaseManualOff:
		return "Manuell aus"
	default:
		return "Bereit"
	}
}

func (a *app) chargingAdminView(tenant tenantConfig, query url.Values) parkingChargingAdminView {
	data := a.parkingStore.TenantData(tenant.Slug)
	cfg := store.NormalizeChargingControlSettings(data.Settings.Charging)
	tariff := parkingTariffAt(data.Settings, time.Now(), time.Local)
	out := parkingChargingAdminView{
		Enabled:          cfg.Enabled,
		ShadowMode:       cfg.ShadowMode,
		StartSocValue:    strings.ReplaceAll(strconv.FormatFloat(cfg.StartSocPercent, 'f', -1, 64), ".", ","),
		StopSocValue:     strings.ReplaceAll(strconv.FormatFloat(cfg.StopSocPercent, 'f', -1, 64), ".", ","),
		StartFeedInValue: strconv.FormatFloat(cfg.StartFeedInW, 'f', 0, 64),
		StopFeedInValue:  strconv.FormatFloat(cfg.StopFeedInW, 'f', 0, 64),
		StopDelayValue:   strconv.Itoa(cfg.StopDelayMinutes),
		MinOnValue:       strconv.Itoa(cfg.MinOnMinutes),
		MinOffValue:      strconv.Itoa(cfg.MinOffMinutes),
		SurplusRateValue: formatInputFloat(surplusRate(tariff)),
		State:            a.chargingAdminStrip(tenant, data.Charging, cfg),
	}
	for _, event := range a.chargingEvents.list(tenant.Slug, 50) {
		out.Events = append(out.Events, chargingEventView{
			AtLabel:   formatDateTimeIn(event.At, time.Local, deATShortDateTimeLayout),
			KindLabel: chargingEventKindLabel(event.Kind),
			KindClass: chargingEventKindClass(event.Kind),
			Detail:    event.Detail,
			Shadow:    event.Shadow,
		})
	}
	out.HasEvents = len(out.Events) > 0
	out.Telegram = a.telegramAdminView(tenant, query)
	return out
}

func chargingEventKindLabel(kind string) string {
	switch kind {
	case "surplus-start", "surplus-end":
		return "Automatik"
	case "manual-end":
		return "Manuell"
	case "confirm-retry", "confirm-failed":
		return "Steckdose"
	case "ha-unreachable", "ha-recovered":
		return "Verbindung"
	case "shadow-switch":
		return "Test"
	default:
		return kind
	}
}

func chargingEventKindClass(kind string) string {
	switch kind {
	case "confirm-retry", "confirm-failed", "ha-unreachable":
		return "dringend"
	case "surplus-start", "ha-recovered":
		return "ok"
	default:
		return ""
	}
}

func (a *app) telegramAdminView(tenant tenantConfig, query url.Values) telegramStatusView {
	out := telegramStatusView{}
	if a.telegram != nil && a.telegram.Configured() {
		out.Configured = true
	}
	if a.telegramStore == nil {
		return out
	}
	for _, link := range a.telegramStore.Links() {
		name := link.Name
		if name == "" {
			name = link.Email
		}
		out.Chats = append(out.Chats, telegramChatView{
			ChatID:      link.ChatID,
			DisplayName: name,
			Email:       link.Email,
			LinkedAt:    formatDateTimeIn(link.LinkedAt, time.Local, deATDateLayout),
		})
	}
	out.HasChats = len(out.Chats) > 0
	out.PendingCode = strings.TrimSpace(query.Get("tgcode"))
	out.CodeEmail = strings.TrimSpace(query.Get("tgmail"))
	for email, profile := range a.profiles {
		if profile.HasTenant(tenant.Slug) && profile.HasPermission(permissionParking) {
			out.LinkOptions = append(out.LinkOptions, telegramLinkOption{Email: email, Label: profile.DisplayName()})
		}
	}
	for email := range a.admins {
		found := false
		for _, option := range out.LinkOptions {
			if option.Email == email {
				found = true
			}
		}
		if !found {
			out.LinkOptions = append(out.LinkOptions, telegramLinkOption{Email: email, Label: email})
		}
	}
	sort.Slice(out.LinkOptions, func(i, j int) bool { return out.LinkOptions[i].Email < out.LinkOptions[j].Email })
	return out
}

// ── handlers ────────────────────────────────────────────────────────────────

func (a *app) canUseCharging(role string, profile userProfile) bool {
	return hasCapability(role, capabilityPlatformAdmin) || profile.HasPermission(permissionParking)
}

// chargingStatus serves the live card as an HTML fragment (default) or JSON
// (?format=json) for the polling enhancer.
func (a *app) chargingStatus(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	profile := a.profileForTenant(email, tenant.Slug)
	if !a.canUseCharging(role, profile) {
		http.NotFound(w, r)
		return
	}
	isAdmin := hasCapability(role, capabilityPlatformAdmin)
	live := a.chargingLiveView(r.Context(), tenant, isAdmin, true)
	if r.URL.Query().Get("format") == "json" {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"available": live.Available,
			"mode":      live.Mode,
			"modeLabel": live.ModeLabel,
			"rate":      live.RateLabel,
			"plugOn":    live.PlugOn,
			"power":     live.PowerLabel,
			"feedIn":    live.FeedInLabel,
			"battery":   live.BatterySOCLabel,
			"stale":     live.StaleData,
			"shadow":    live.ShadowMode,
			"session":   live.HasSession,
		})
		return
	}
	if err := a.templates.ExecuteTemplate(w, "parkingLiveCard", map[string]any{"Live": live}); err != nil {
		logError("charging status render failed", err)
	}
}

func (a *app) chargingManualAction(w http.ResponseWriter, r *http.Request, ac authCtx, mode string) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	profile := a.profileForTenant(email, tenant.Slug)
	if !a.canUseCharging(role, profile) {
		http.NotFound(w, r)
		return
	}
	var result chargingCommandResult
	switch mode {
	case "on":
		result = a.requestManualCharging(r.Context(), tenant, true, email, chargingTriggerWeb)
	case "off":
		result = a.requestManualCharging(r.Context(), tenant, false, email, chargingTriggerWeb)
	default:
		result = a.requestAutomaticCharging(r.Context(), tenant, email, chargingTriggerWeb)
	}
	status := "error"
	if result.OK {
		status = "ok"
	}
	http.Redirect(w, r, "/app/parking?charging="+status+"&chargingmsg="+url.QueryEscape(result.Message), http.StatusSeeOther)
}

func (a *app) chargingOnAction(w http.ResponseWriter, r *http.Request, ac authCtx) {
	a.chargingManualAction(w, r, ac, "on")
}

func (a *app) chargingOffAction(w http.ResponseWriter, r *http.Request, ac authCtx) {
	a.chargingManualAction(w, r, ac, "off")
}

func (a *app) chargingAutoAction(w http.ResponseWriter, r *http.Request, ac authCtx) {
	a.chargingManualAction(w, r, ac, "auto")
}

func (a *app) updateChargingSettings(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	cfg, err := chargingControlFromForm(r.Form)
	if err != nil {
		http.Redirect(w, r, "/app/parking/settings?section=charging&charging=invalid", http.StatusSeeOther)
		return
	}
	if err := a.parkingStore.SetChargingControl(tenant.Slug, cfg); err != nil {
		logError("charging settings save failed", err, "tenant", tenant.Slug)
		http.Error(w, "Could not save charging settings", http.StatusInternalServerError)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: email,
		ActorRole:  role,
		Action:     auditActionChargingSettings,
		TargetType: "charging",
		TargetID:   tenant.Slug,
		Summary:    "Laderegelung geändert",
		Details: map[string]string{
			"enabled": strconv.FormatBool(cfg.Enabled),
			"shadow":  strconv.FormatBool(cfg.ShadowMode),
		},
	})
	http.Redirect(w, r, "/app/parking/settings?section=charging&charging=saved", http.StatusSeeOther)
}

func chargingControlFromForm(values url.Values) (chargingControlSettings, error) {
	cfg := chargingControlSettings{
		Enabled:    values.Get("controller_enabled") != "",
		ShadowMode: values.Get("shadow_mode") != "",
	}
	numbers := []struct {
		field string
		dst   *float64
		min   float64
		max   float64
	}{
		{"start_soc_percent", &cfg.StartSocPercent, 1, 100},
		{"stop_soc_percent", &cfg.StopSocPercent, 1, 100},
		{"start_feed_in_w", &cfg.StartFeedInW, 100, 50000},
		{"stop_feed_in_w", &cfg.StopFeedInW, 50, 50000},
	}
	for _, number := range numbers {
		value, err := parseDecimal(values.Get(number.field))
		if err != nil || value < number.min || value > number.max {
			return chargingControlSettings{}, errInvalidChargingForm
		}
		*number.dst = value
	}
	minutes := []struct {
		field string
		dst   *int
	}{
		{"stop_delay_minutes", &cfg.StopDelayMinutes},
		{"min_on_minutes", &cfg.MinOnMinutes},
		{"min_off_minutes", &cfg.MinOffMinutes},
	}
	for _, minute := range minutes {
		value, err := strconv.Atoi(strings.TrimSpace(values.Get(minute.field)))
		if err != nil || value < 1 || value > 120 {
			return chargingControlSettings{}, errInvalidChargingForm
		}
		*minute.dst = value
	}
	if cfg.StopFeedInW >= cfg.StartFeedInW || cfg.StopSocPercent > cfg.StartSocPercent {
		return chargingControlSettings{}, errInvalidChargingForm
	}
	return cfg, nil
}

func (a *app) createTelegramLinkCode(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	if a.telegramStore == nil {
		http.Redirect(w, r, "/app/parking/settings?section=telegram&charging=tgerror", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	target := normalizeEmail(r.Form.Get("email"))
	code, err := a.telegramStore.CreateLinkCode(target, email, 24*time.Hour)
	if err != nil {
		http.Redirect(w, r, "/app/parking/settings?section=telegram&charging=tgerror", http.StatusSeeOther)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: email,
		ActorRole:  role,
		Action:     auditActionChargingSettings,
		TargetType: "telegram",
		TargetID:   target,
		Summary:    "Telegram-Verknüpfungscode erzeugt",
	})
	http.Redirect(w, r, "/app/parking/settings?section=telegram&charging=tgcode&tgcode="+url.QueryEscape(code)+"&tgmail="+url.QueryEscape(target)+"#telegram", http.StatusSeeOther)
}

func (a *app) unlinkTelegramChat(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	if a.telegramStore == nil {
		http.Redirect(w, r, "/app/parking/settings?section=telegram&charging=tgerror", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	chatID, err := strconv.ParseInt(strings.TrimSpace(r.Form.Get("chat_id")), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/app/parking/settings?section=telegram&charging=tgerror", http.StatusSeeOther)
		return
	}
	if err := a.telegramStore.Unlink(chatID); err != nil {
		http.Redirect(w, r, "/app/parking/settings?section=telegram&charging=tgerror", http.StatusSeeOther)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: email,
		ActorRole:  role,
		Action:     auditActionChargingSettings,
		TargetType: "telegram",
		TargetID:   tenant.Slug,
		Summary:    "Telegram-Chat getrennt",
	})
	http.Redirect(w, r, "/app/parking/settings?section=telegram&charging=tgunlinked#telegram", http.StatusSeeOther)
}

func chargingSettingsMessage(code string) (string, bool) {
	switch code {
	case "saved":
		return "Laderegelung gespeichert.", true
	case "invalid":
		return "Bitte Eingaben prüfen: Stopp-Werte müssen unter den Start-Werten liegen, Minuten zwischen 1 und 120.", false
	case "tgcode":
		return "Verknüpfungscode erzeugt — bitte an die Person weitergeben.", true
	case "tgunlinked":
		return "Telegram-Chat getrennt.", true
	case "tgerror":
		return "Telegram-Aktion fehlgeschlagen.", false
	default:
		return "", false
	}
}

func formatWatt(value float64) string {
	if value >= 1000 {
		return strings.ReplaceAll(strconv.FormatFloat(value/1000, 'f', 1, 64), ".", ",") + " kW"
	}
	return strconv.FormatFloat(value, 'f', 0, 64) + " W"
}
