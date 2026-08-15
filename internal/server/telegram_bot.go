package server

// The dedicated hausv.org Telegram bot: long-polls getUpdates for commands
// (/pp20ein, /pp20aus, /pp20auto, /pp20status, /start <code>) and delivers
// charging notifications. Chats are linked to portal users via one-time
// codes minted in the admin UI — chat IDs live only in the telegram store
// under /data, never in env or git.

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/telegram"
)

type telegramStore = store.TelegramStore
type telegramLink = store.TelegramLink

var newTelegramStore = store.NewTelegramStore

// telegramAPI is the seam between the bot loop and the wire; tests plug in a
// recording fake.
type telegramAPI interface {
	Configured() bool
	GetUpdates(ctx context.Context, offset int64, timeout time.Duration) ([]telegram.Update, error)
	SendMessage(ctx context.Context, chatID int64, text string) error
}

func (a *app) StartTelegramBot() func() { return a.startTelegramBot() }

func (a *app) startTelegramBot() func() {
	if a.telegram == nil || !a.telegram.Configured() {
		logInfo("Telegram bot disabled", "reason", "not_configured")
		return func() {}
	}
	if a.telegramStore == nil {
		logWarn("Telegram bot disabled", "reason", "store_unavailable")
		return func() {}
	}
	logInfo("Telegram bot enabled", "poll_timeout", a.telegramPollTimeout)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		backoff := 5 * time.Second
		for {
			if ctx.Err() != nil {
				return
			}
			updates, err := a.telegram.GetUpdates(ctx, a.telegramStore.Offset(), a.telegramPollTimeout)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				logError("Telegram poll failed", err)
				select {
				case <-ctx.Done():
					return
				case <-time.After(backoff):
				}
				if backoff < time.Minute {
					backoff *= 2
				}
				continue
			}
			backoff = 5 * time.Second
			for _, update := range updates {
				a.handleTelegramUpdate(ctx, update)
				if err := a.telegramStore.SetOffset(update.UpdateID + 1); err != nil {
					logError("Telegram offset save failed", err, "update_id", update.UpdateID)
				}
			}
		}
	}()
	return cancel
}

func (a *app) handleTelegramUpdate(ctx context.Context, update telegram.Update) {
	if update.Message == nil {
		return
	}
	chatID := update.Message.Chat.ID
	text := strings.TrimSpace(update.Message.Text)
	if chatID == 0 || text == "" {
		return
	}
	reply := a.handleTelegramCommand(ctx, chatID, firstNameOf(update.Message), text)
	if reply == "" {
		return
	}
	sctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := a.telegram.SendMessage(sctx, chatID, reply); err != nil {
		logError("Telegram reply failed", err)
	}
}

func firstNameOf(message *telegram.Message) string {
	if message == nil || message.From == nil {
		return ""
	}
	return message.From.FirstName
}

// handleTelegramCommand routes one message and returns the reply text
// (empty = stay silent, e.g. unknown chats).
func (a *app) handleTelegramCommand(ctx context.Context, chatID int64, senderName, text string) string {
	command, arg := splitTelegramCommand(text)
	link, linked := a.telegramStore.LinkByChat(chatID)

	if command == "/start" {
		if linked {
			return "Dieser Chat ist bereits mit " + link.Email + " verknüpft.\n\n" + telegramHelpText()
		}
		if arg == "" {
			// Unlinked /start without code: point at the onboarding flow but
			// leak nothing about the platform's users.
			return "Willkommen! Zum Verknüpfen brauchst du einen Code aus dem Portal: /start <CODE>"
		}
		consumed, err := a.telegramStore.ConsumeLinkCode(arg, chatID, senderName)
		if err != nil {
			return "Dieser Code ist unbekannt oder abgelaufen. Bitte im Portal einen neuen erzeugen."
		}
		logInfo("Telegram chat linked", "actor", redactedEmail(consumed.Email))
		return "Verknüpft! Du bekommst jetzt Lade-Benachrichtigungen.\n\n" + telegramHelpText()
	}

	if !linked {
		// Unknown chats get no information at all.
		return ""
	}
	profile, ok := a.directoryProfile(link.Email)
	if !ok {
		return "Dein Portal-Zugang wurde nicht gefunden. Bitte an die Verwaltung wenden."
	}
	tenant, ok := a.tenants[a.defaultTenant]
	if !ok {
		return "Kein Gebäude konfiguriert."
	}
	role := profile.ForTenant(tenant.Slug).Role
	if !can(actorFor(link.Email, tenant.Slug, role), capabilityPlatformAdmin, resourceFor(tenant.Slug)) && !profile.HasPermission(permissionParking) {
		return "Dir fehlt die Berechtigung für die Ladesteuerung."
	}

	switch command {
	case "/pp20ein":
		result := a.requestManualCharging(ctx, tenant, true, link.Email, chargingTriggerTelegram)
		return result.Message
	case "/pp20aus":
		result := a.requestManualCharging(ctx, tenant, false, link.Email, chargingTriggerTelegram)
		return result.Message
	case "/pp20auto":
		result := a.requestAutomaticCharging(ctx, tenant, link.Email, chargingTriggerTelegram)
		return result.Message
	case "/pp20status":
		return a.chargingStatusText(ctx, tenant)
	default:
		return telegramHelpText()
	}
}

func splitTelegramCommand(text string) (string, string) {
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) == 0 {
		return "", ""
	}
	command := strings.ToLower(fields[0])
	// "/pp20status@HausvOrgBot" → "/pp20status"
	if at := strings.Index(command, "@"); at > 0 {
		command = command[:at]
	}
	arg := ""
	if len(fields) > 1 {
		arg = fields[1]
	}
	return command, arg
}

func telegramHelpText() string {
	return strings.Join([]string{
		"Befehle:",
		"/pp20status — Status von Parkplatz 20",
		"/pp20ein — Ladung einschalten (Normaltarif)",
		"/pp20aus — Ladung ausschalten (pausiert die Automatik)",
		"/pp20auto — Automatik (Überschussladen) aktivieren",
	}, "\n")
}

// chargingStatusText renders the /pp20status reply.
func (a *app) chargingStatusText(ctx context.Context, tenant tenantConfig) string {
	a.chargingMu.Lock()
	defer a.chargingMu.Unlock()
	data := a.parkingStore.TenantData(tenant.Slug)
	cfg := store.NormalizeChargingControlSettings(data.Settings.Charging)
	state := data.Charging
	lines := []string{"🅿️ Parkplatz 20"}

	mode := "Automatik bereit"
	switch state.Phase {
	case chargingPhaseSurplus:
		mode = "☀️ Überschussladen aktiv (" + formatEURPerKWh(surplusRate(parkingTariffAt(data.Settings, time.Now(), time.Local))) + ")"
	case chargingPhaseManualOn:
		mode = "⚡ Manuell eingeschaltet (Normaltarif)"
	case chargingPhaseManualOff:
		mode = "⏸ Manuell ausgeschaltet (Automatik pausiert)"
	}
	if !cfg.Enabled {
		mode = "Automatik deaktiviert"
	} else if cfg.ShadowMode {
		mode += " · Testbetrieb, Steuerung läuft noch über das alte System"
	}
	lines = append(lines, mode)

	in := a.readChargingInputs(ctx, tenant)
	if in.ReadOK {
		plug := "aus"
		if in.PlugOn {
			plug = "ein"
		}
		lines = append(lines,
			fmt.Sprintf("🔌 Steckdose: %s", plug),
			fmt.Sprintf("🔋 Hausakku: %.0f %% · Einspeisung: %.0f W", in.SocPercent, in.FeedInW),
			fmt.Sprintf("⚡ Zählerstand: %s", formatKWh(in.MeterKWh)),
		)
		if state.ActiveSessionID != "" {
			for _, session := range data.ChargingSessions {
				if session.ID != state.ActiveSessionID {
					continue
				}
				kWh := in.MeterKWh - session.StartKWh
				if kWh < 0 {
					kWh = 0
				}
				cost := kWh * a.chargingSessionRate(data, session)
				lines = append(lines, fmt.Sprintf("🚘 Lädt seit %s: %s (≈ %s)",
					formatDateTimeIn(session.Start, time.Local, deATTimeLayout), formatKWh(kWh), formatEUR(cost)))
				break
			}
		}
	} else {
		lines = append(lines, "⚠️ Home Assistant ist gerade nicht erreichbar — Werte unbekannt.")
	}
	if last, ok := lastClosedChargingSession(data.ChargingSessions); ok {
		kWh := last.EndKWh - last.StartKWh
		lines = append(lines, fmt.Sprintf("Letzte Ladung: %s, %s (%s)",
			formatDateTimeIn(last.Start, time.Local, deATShortDateTimeLayout), formatKWh(kWh), chargingModeLabel(last.Mode)))
	}
	return strings.Join(lines, "\n")
}

func (a *app) chargingSessionRate(data parkingTenantData, session chargingSession) float64 {
	tariff := parkingTariffAt(data.Settings, session.Start, time.Local)
	if session.Mode == chargingModeSurplus {
		return surplusRate(tariff)
	}
	// Normal tariff approximation for the live view: current spot + grid fee.
	price := 0.0
	if n := len(data.PriceSamples); n > 0 {
		price = data.PriceSamples[n-1].Value
	}
	return price + tariff.GridFeeEURPerKWh
}

func lastClosedChargingSession(sessions []chargingSession) (chargingSession, bool) {
	for i := len(sessions) - 1; i >= 0; i-- {
		if !sessions[i].End.IsZero() {
			return sessions[i], true
		}
	}
	return chargingSession{}, false
}

func chargingModeLabel(mode string) string {
	if mode == chargingModeSurplus {
		return "Überschuss"
	}
	return "Normal"
}

// ── notification transport ──────────────────────────────────────────────────

// telegramChargingBroadcast fans a charging notification out to the linked
// chats of the given recipients. A sliding-window limiter caps the fallout
// of any future logic bug — the old flow's failure mode was exactly this.
func (a *app) telegramChargingBroadcast(recipients []string, subject string, lines []string) {
	if a.telegram == nil || !a.telegram.Configured() || a.telegramStore == nil {
		return
	}
	now := time.Now()
	if !a.chargingTelegramAllowed(now) {
		return
	}
	text := subject
	if len(lines) > 0 {
		text += "\n" + strings.Join(lines, "\n")
	}
	seen := map[int64]bool{}
	for _, email := range recipients {
		for _, chatID := range a.telegramStore.ChatsByEmail(email) {
			if seen[chatID] {
				continue
			}
			seen[chatID] = true
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := a.telegram.SendMessage(ctx, chatID, text); err != nil {
				logError("Telegram notification failed", err, "recipient", redactedEmail(email))
			}
			cancel()
		}
	}
}

const chargingTelegramWindowLimit = 12

// chargingTelegramAllowed enforces max 12 broadcasts per sliding hour. On
// overflow it drops the message and (once per throttle episode) says so.
func (a *app) chargingTelegramAllowed(now time.Time) bool {
	a.chargingTelegramMu.Lock()
	defer a.chargingTelegramMu.Unlock()
	cutoff := now.Add(-time.Hour)
	kept := a.chargingTelegramTimes[:0]
	for _, at := range a.chargingTelegramTimes {
		if at.After(cutoff) {
			kept = append(kept, at)
		}
	}
	a.chargingTelegramTimes = kept
	if len(a.chargingTelegramTimes) >= chargingTelegramWindowLimit {
		if !a.chargingTelegramThrottled {
			a.chargingTelegramThrottled = true
			logWarn("Telegram charging notifications throttled", "hourly_limit", chargingTelegramWindowLimit)
			go a.telegramThrottleNotice()
		}
		return false
	}
	a.chargingTelegramTimes = append(a.chargingTelegramTimes, now)
	a.chargingTelegramThrottled = false
	return true
}

// telegramThrottleNotice tells the admins' chats exactly once per episode
// that messages are being dropped.
func (a *app) telegramThrottleNotice() {
	if a.telegram == nil || !a.telegram.Configured() || a.telegramStore == nil {
		return
	}
	text := "⚠️ Zu viele Lade-Benachrichtigungen — weitere werden vorerst unterdrückt. Bitte Ereignisprotokoll prüfen."
	for email := range a.admins {
		for _, chatID := range a.telegramStore.ChatsByEmail(email) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			_ = a.telegram.SendMessage(ctx, chatID, text)
			cancel()
		}
	}
}
