package server

// The dedicated hausv.org Telegram bot: long-polls getUpdates for commands
// (/pp20ein, /pp20aus, /pp20auto, /pp20status, /start <code>) and delivers
// charging notifications. Chats are linked to portal users via one-time
// codes minted in the admin UI — chat IDs live only in the telegram store
// under /data, never in env or git.

import (
	"context"
	"fmt"
	"sort"
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

// handleTelegramCommand routes one message and returns the reply text.
// A linked chat acts as its linked user. The target house is chosen from
// that user's permissions at this moment; a remembered slug is only a
// preference and is ignored once it is no longer allowed.
func (a *app) handleTelegramCommand(ctx context.Context, chatID int64, senderName, text string) string {
	command, arg := splitTelegramCommand(text)
	link, linked := a.telegramStore.LinkByChat(chatID)

	if command == "/start" {
		if linked {
			a.auditTelegramCommand(link.Email, "", command, "already_linked")
			return "Dieser Chat ist bereits mit " + link.Email + " verknüpft.\n\n" + telegramHelpText()
		}
		if arg == "" {
			a.auditTelegramCommand("", "", command, "unlinked")
			return "Dieser Chat ist nicht verknüpft. Zum Verknüpfen brauchst du einen Code aus dem Portal: /start <CODE>"
		}
		consumed, err := a.telegramStore.ConsumeLinkCode(arg, chatID, senderName)
		if err != nil {
			a.auditTelegramCommand("", "", command, "link_rejected")
			return "Dieser Code ist unbekannt oder abgelaufen. Bitte im Portal einen neuen erzeugen."
		}
		// A previous person's house choice must not survive a new link.
		a.forgetTelegramHouse(chatID)
		a.auditTelegramCommand(consumed.Email, "", command, "linked")
		return "Verknüpft! Du bekommst jetzt Lade-Benachrichtigungen.\n\n" + telegramHelpText()
	}

	if !linked {
		a.forgetTelegramHouse(chatID)
		a.auditTelegramCommand("", "", command, "unlinked")
		return "Dieser Chat ist nicht verknüpft."
	}
	if command == "/haus" {
		return a.telegramSelectHouse(chatID, link.Email, arg)
	}
	if isTelegramChargingCommand(command) {
		return a.telegramRunCharging(ctx, chatID, link.Email, command)
	}
	a.auditTelegramCommand(link.Email, "", command, "help")
	return telegramHelpText()
}

func isTelegramChargingCommand(command string) bool {
	switch command {
	case "/pp20ein", "/pp20aus", "/pp20auto", "/pp20status":
		return true
	default:
		return false
	}
}

// telegramChargingContext resolves the houses this user may control right
// now. On failure it writes the audit line and the reply; callers must not
// audit again.
func (a *app) telegramChargingContext(email, command string) ([]tenantConfig, string, bool) {
	profile, ok := a.directoryProfile(email)
	if !ok {
		a.auditTelegramCommand(email, "", command, "denied")
		return nil, "Dein Portal-Zugang wurde nicht gefunden. Bitte an die Verwaltung wenden.", false
	}
	if profile.Deactivated && !a.telegramBreakGlass(email) {
		a.auditTelegramCommand(email, "", command, "denied")
		return nil, "Dir fehlt die Berechtigung für die Ladesteuerung.", false
	}
	if len(a.tenants) == 0 {
		a.auditTelegramCommand(email, "", command, "unconfigured")
		return nil, "Kein Gebäude konfiguriert.", false
	}
	houses := a.telegramAuthorisedHouses(email, profile)
	if len(houses) == 0 {
		a.auditTelegramCommand(email, "", command, "denied")
		return nil, "Dir fehlt die Berechtigung für die Ladesteuerung.", false
	}
	return houses, "", true
}

func (a *app) telegramBreakGlass(email string) bool {
	if a == nil || a.admins == nil {
		return false
	}
	_, ok := a.admins[normalizeEmail(email)]
	return ok
}

// telegramAuthorisedHouses returns the houses whose current membership
// grants charging control. Profile-wide parking permission applies only
// inside a house the user belongs to, and only when that house does not
// replace the permission list.
func (a *app) telegramAuthorisedHouses(email string, profile userProfile) []tenantConfig {
	houses := make([]tenantConfig, 0, 1)
	for slug, tenant := range a.tenants {
		if tenant.Slug == "" {
			tenant.Slug = slug
		}
		// The label follows a portal rename. The connector stays the one configured for this house.
		connector := tenant.HA
		if resolved, found := a.tenantBySlug(tenant.Slug); found {
			tenant = resolved
			if tenant.Slug == "" {
				tenant.Slug = slug
			}
		}
		tenant.HA = connector
		if !profile.HasTenant(tenant.Slug) {
			continue
		}
		scoped := profile.ForTenant(tenant.Slug)
		actor := a.actorFor(email, tenant.Slug, scoped.Role)
		if can(actor, capabilityPlatformAdmin, resourceFor(tenant.Slug)) || scoped.HasPermission(permissionParking) {
			houses = append(houses, tenant)
		}
	}
	sort.Slice(houses, func(i, j int) bool {
		left, right := telegramHouseLabel(houses[i]), telegramHouseLabel(houses[j])
		if left == right {
			return houses[i].Slug < houses[j].Slug
		}
		return left < right
	})
	return houses
}

func (a *app) telegramRunCharging(ctx context.Context, chatID int64, email, command string) string {
	houses, reply, ok := a.telegramChargingContext(email, command)
	if !ok {
		a.forgetTelegramHouse(chatID)
		return reply
	}
	house, chosen := a.telegramResolveHouse(chatID, houses)
	if !chosen {
		a.auditTelegramCommand(email, "", command, "ambiguous")
		return telegramHousePrompt(houses)
	}
	body := ""
	outcome := "ok"
	switch command {
	case "/pp20ein":
		result := a.requestManualCharging(ctx, house, true, email, chargingTriggerTelegram)
		body, outcome = result.Message, telegramResultOutcome(result.OK)
	case "/pp20aus":
		result := a.requestManualCharging(ctx, house, false, email, chargingTriggerTelegram)
		body, outcome = result.Message, telegramResultOutcome(result.OK)
	case "/pp20auto":
		result := a.requestAutomaticCharging(ctx, house, email, chargingTriggerTelegram)
		body, outcome = result.Message, telegramResultOutcome(result.OK)
	case "/pp20status":
		body = a.chargingStatusText(ctx, house)
	default:
		a.auditTelegramCommand(email, house.Slug, command, "help")
		return telegramHelpText()
	}
	a.auditTelegramCommand(email, house.Slug, command, outcome)
	return telegramReplyForHouse(body, house, houses)
}

func telegramResultOutcome(ok bool) string {
	if ok {
		return "ok"
	}
	return "refused"
}

func (a *app) telegramSelectHouse(chatID int64, email, name string) string {
	houses, reply, ok := a.telegramChargingContext(email, "/haus")
	if !ok {
		a.forgetTelegramHouse(chatID)
		return reply
	}
	name = strings.TrimSpace(name)
	if name == "" {
		if len(houses) == 1 {
			a.auditTelegramCommand(email, houses[0].Slug, "/haus", "selected")
			return "Für dich ist nur " + telegramHouseLabel(houses[0]) + " freigegeben. Die Ladesteuerung gilt für dieses Haus."
		}
		a.auditTelegramCommand(email, "", "/haus", "ambiguous")
		return telegramHousePrompt(houses)
	}
	match, found := matchTelegramHouse(houses, name)
	if !found {
		a.auditTelegramCommand(email, "", "/haus", "denied")
		return "Dieses Haus ist nicht freigegeben.\n\n" + telegramHouseChoices(houses)
	}
	if len(houses) == 1 {
		a.forgetTelegramHouse(chatID)
	} else {
		a.rememberTelegramHouse(chatID, match.Slug)
	}
	a.auditTelegramCommand(email, match.Slug, "/haus", "selected")
	return "Ausgewählt: " + telegramDistinctHouseLabel(match, houses) + ". Die Auswahl gilt nur für diese Sitzung und wird bei jedem Befehl neu geprüft."
}

// telegramResolveHouse uses the single authorised house, or the chat's
// remembered slug when it is still in the freshly computed set.
func (a *app) telegramResolveHouse(chatID int64, houses []tenantConfig) (tenantConfig, bool) {
	if len(houses) == 1 {
		a.forgetTelegramHouse(chatID)
		return houses[0], true
	}
	slug := a.telegramHouseSlug(chatID)
	if slug != "" {
		for _, house := range houses {
			if house.Slug == slug {
				return house, true
			}
		}
		a.forgetTelegramHouse(chatID)
	}
	return tenantConfig{}, false
}

func (a *app) telegramHouseSlug(chatID int64) string {
	if a == nil {
		return ""
	}
	a.telegramHouseMu.Lock()
	defer a.telegramHouseMu.Unlock()
	if a.telegramHouseChoice == nil {
		return ""
	}
	return a.telegramHouseChoice[chatID]
}

func (a *app) rememberTelegramHouse(chatID int64, slug string) {
	if a == nil || slug == "" {
		return
	}
	a.telegramHouseMu.Lock()
	defer a.telegramHouseMu.Unlock()
	if a.telegramHouseChoice == nil {
		a.telegramHouseChoice = map[int64]string{}
	}
	a.telegramHouseChoice[chatID] = slug
}

func (a *app) forgetTelegramHouse(chatID int64) {
	if a == nil {
		return
	}
	a.telegramHouseMu.Lock()
	defer a.telegramHouseMu.Unlock()
	delete(a.telegramHouseChoice, chatID)
}

func matchTelegramHouse(houses []tenantConfig, query string) (tenantConfig, bool) {
	query = strings.Join(strings.Fields(query), " ")
	if query == "" {
		return tenantConfig{}, false
	}
	matches := make([]tenantConfig, 0, 1)
	for _, house := range houses {
		if strings.EqualFold(query, telegramHouseLabel(house)) || strings.EqualFold(query, house.Slug) {
			matches = append(matches, house)
		}
	}
	if len(matches) != 1 {
		return tenantConfig{}, false
	}
	return matches[0], true
}

func telegramHouseLabel(tenant tenantConfig) string {
	name := strings.TrimSpace(tenant.Name)
	if name == "" {
		return tenant.Slug
	}
	return name
}

// telegramDistinctHouseLabel adds the slug when two authorised houses share a name.
func telegramDistinctHouseLabel(house tenantConfig, houses []tenantConfig) string {
	label := telegramHouseLabel(house)
	same := 0
	for _, other := range houses {
		if strings.EqualFold(telegramHouseLabel(other), label) {
			same++
		}
	}
	if same > 1 && house.Slug != "" && !strings.EqualFold(label, house.Slug) {
		return label + " (" + house.Slug + ")"
	}
	return label
}

func telegramReplyForHouse(body string, house tenantConfig, houses []tenantConfig) string {
	label := "Haus: " + telegramDistinctHouseLabel(house, houses)
	body = strings.TrimSpace(body)
	if body == "" {
		return label
	}
	return body + "\n" + label
}

func telegramHousePrompt(houses []tenantConfig) string {
	labelCount := map[string]int{}
	for _, house := range houses {
		labelCount[strings.ToLower(telegramHouseLabel(house))]++
	}
	lines := []string{"Mehrere Häuser sind freigegeben. Welches soll die Ladesteuerung verwenden?"}
	for _, house := range houses {
		label := telegramHouseLabel(house)
		// Identical names cannot be told apart. Offer the slug, which also matches.
		if labelCount[strings.ToLower(label)] > 1 {
			label = house.Slug
		}
		lines = append(lines, "/haus "+label)
	}
	lines = append(lines, "Die Auswahl gilt nur für diese Sitzung und wird bei jedem Befehl neu geprüft.")
	return strings.Join(lines, "\n")
}

func telegramHouseChoices(houses []tenantConfig) string {
	if len(houses) == 1 {
		return "Für dich ist nur " + telegramHouseLabel(houses[0]) + " freigegeben."
	}
	return telegramHousePrompt(houses)
}

func (a *app) auditTelegramCommand(email, house, command, outcome string) {
	user := ""
	if strings.TrimSpace(email) != "" {
		user = redactedEmail(email)
	}
	logInfo("Telegram command",
		"user", user,
		"house", house,
		"command", telegramAuditCommand(command),
		"outcome", outcome,
	)
}

func telegramAuditCommand(command string) string {
	switch command {
	case "/start", "/haus", "/pp20ein", "/pp20aus", "/pp20auto", "/pp20status":
		return command
	default:
		return "other"
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
		// House names may contain spaces. Link codes stay a single token.
		arg = strings.Join(fields[1:], " ")
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
		"/haus <Name> — Haus auswählen, wenn mehrere freigegeben sind",
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
