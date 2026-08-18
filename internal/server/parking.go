package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/homeassistant"
	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/version"
	"github.com/inspr-at/hausv-org/internal/web"
)

func (a *app) parking(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	if denyServiceProviderArea(w, role) {
		return
	}
	profile := a.profileForTenant(email, tenant.Slug)
	if !ac.can(capabilityPlatformAdmin) && !profile.HasPermission(permissionParking) {
		http.NotFound(w, r)
		return
	}
	isAdmin := ac.can(capabilityPlatformAdmin)
	parkingMsg, parkingOK := parkingMessage(r.URL.Query().Get("month"), r.URL.Query().Get("reminder"))
	if parkingMsg == "" {
		parkingMsg, parkingOK = chargingFlashMessage(r.URL.Query())
	}
	accounting := a.parkingAccounting(r.Context(), tenant)
	accounting.Months = a.hydrateParkingMonths(ac.tenantRef, email, role, accounting.Months)
	var currentMonth parkingMonthView
	var olderMonths []parkingMonthView
	currentMonthHeading := "Neuester Monat"
	if len(accounting.Months) > 0 {
		currentMonth = accounting.Months[0]
		olderMonths = accounting.Months[1:]
		if currentMonth.Month == time.Now().In(time.Local).Format("2006-01") {
			currentMonthHeading = "Aktueller Monat"
		}
	}
	live := a.chargingLiveView(r.Context(), tenant, isAdmin, true)
	canManageParkingPayments := ac.can(capabilityManageUsers) || ac.can(capabilityManageParking)
	a.renderParkingTempl(w, r, web.ParkingPageData{
		Portal:                   a.parkingPortalContext(ac),
		AssetVersion:             version.AssetVersion(),
		IsAdmin:                  isAdmin,
		CanManageParkingPayments: canManageParkingPayments,
		Message:                  parkingMsg,
		MessageOK:                parkingOK,
		StatementYear:            time.Now().In(time.Local).Year(),
		CurrentMonthHeading:      currentMonthHeading,
		HasOlderMonths:           len(olderMonths) > 0,
		Accounting:               accounting,
		Live:                     live,
		CurrentMonth:             currentMonth,
		OlderMonths:              olderMonths,
	})
}

// parkingPortalContext mirrors the legacy base-context override: /app/parking
// is reachable through the explicit per-user parking permission as well as
// through platform admin, and the handler has already established one of the
// two. The sidebar entry stays gated on the parking module either way.
func (a *app) parkingPortalContext(ac authCtx) web.PortalPageData {
	data := a.auditPortalContext(ac, "Parkplatznutzung")
	data.ActivePage = "parking"
	data.CanSeeParking = true
	return data
}

func (a *app) renderParkingTempl(w http.ResponseWriter, r *http.Request, data web.ParkingPageData) {
	var rendered bytes.Buffer
	if err := web.ParkingPage(data).Render(r.Context(), &rendered); err != nil {
		logError("templ parking render failed", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, prefixTenantHTMLPaths(rendered.String(), data.Portal.TenantSlug))
}

// chargingFlashMessage surfaces the redirect outcome of a charging action.
// The action result text travels in the query so the flash matches what the
// Telegram reply would have said.
func chargingFlashMessage(query url.Values) (string, bool) {
	status := query.Get("charging")
	if status == "" {
		return "", false
	}
	message := strings.TrimSpace(query.Get("chargingmsg"))
	if message == "" {
		if status == "ok" {
			message = "Erledigt."
		} else {
			message = "Aktion fehlgeschlagen."
		}
	}
	return message, status == "ok"
}

func (a *app) parkingSettings(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant := ac.tenant
	settingsMsg, settingsOK := parkingSettingsMessage(r.URL.Query().Get("settings"))
	chargingStatus := r.URL.Query().Get("charging")
	chargingMsg, chargingOK := chargingSettingsMessage(chargingStatus)
	section := parkingAdminSection(r.URL.Query().Get("section"), chargingStatus)
	a.render(w, "parkingSettings", a.withBase(ac, map[string]any{
		"Title": "Parkplatz verwalten",
		// This route is capability-gated before the handler. Preserve its
		// deliberate admin presentation for delegated parking managers.
		"IsAdmin":                true,
		"CanSeeParking":          true,
		"ActivePage":             "settings",
		"Accounting":             a.parkingAccounting(r.Context(), tenant),
		"SettingsMsg":            settingsMsg,
		"SettingsOK":             settingsOK,
		"ChargingMsg":            chargingMsg,
		"ChargingOK":             chargingOK,
		"Charging":               a.chargingAdminView(tenant, r.URL.Query()),
		"ParkingSection":         section,
		"SectionAccounting":      section == "accounting",
		"SectionCharging":        section == "charging",
		"SectionTelegram":        section == "telegram",
		"CanManageParkingConfig": true,
	}))
}

func parkingAdminSection(requested string, chargingStatus string) string {
	switch strings.TrimSpace(requested) {
	case "accounting", "charging", "telegram":
		return strings.TrimSpace(requested)
	}
	if strings.HasPrefix(strings.TrimSpace(chargingStatus), "tg") {
		return "telegram"
	}
	if strings.TrimSpace(chargingStatus) != "" {
		return "charging"
	}
	return "accounting"
}

func (a *app) parkingMonth(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	profile := a.profileForTenant(email, tenant.Slug)
	if !ac.can(capabilityPlatformAdmin) && !profile.HasPermission(permissionParking) {
		http.NotFound(w, r)
		return
	}
	month := strings.TrimSpace(r.PathValue("month"))
	if _, err := time.Parse("2006-01", month); err != nil {
		http.NotFound(w, r)
		return
	}
	view := a.parkingMonthDetails(r.Context(), tenant, month)
	view.Summary = a.hydrateParkingMonth(ac.tenantRef, email, role, view.Summary)
	if !view.HasHours && view.Summary.Month == "" {
		http.NotFound(w, r)
		return
	}
	parkingMsg, parkingOK := parkingMessage(r.URL.Query().Get("month"), "")
	a.render(w, "parkingMonth", a.withBase(ac, map[string]any{
		"Title":                    "Parkplatznutzung · " + view.MonthLabel,
		"CanSeeParking":            true,
		"CanManageParkingPayments": ac.can(capabilityManageUsers) || ac.can(capabilityManageParking),
		"CanMarkParkingPayment":    ac.can(capabilityPlatformAdmin) || ac.can(capabilityManageUsers) || ac.can(capabilityManageParking) || profile.HasPermission(permissionParking),
		"ActivePage":               "parking",
		"Detail":                   view,
		"ParkingMsg":               parkingMsg,
		"ParkingOK":                parkingOK,
	}))
}

func (a *app) hydrateParkingMonths(tenant store.TenantRef, email string, role string, months []parkingMonthView) []parkingMonthView {
	for i := range months {
		months[i] = a.hydrateParkingMonth(tenant, email, role, months[i])
	}
	return months
}

func (a *app) hydrateParkingMonth(tenant store.TenantRef, email string, role string, month parkingMonthView) parkingMonthView {
	if a == nil || a.attachmentStore == nil || month.Month == "" {
		return month
	}
	attachments := a.attachmentViewsForEntity(tenant, "parking", month.Month, email, role)
	if len(attachments) == 0 {
		return month
	}
	month.Attachments = attachments
	month.HasAttachments = true
	return month
}

func (a *app) parkingStatement(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	year, ok := parkingStatementYear(r.PathValue("year"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	actor := a.profileForTenant(email, tenant.Slug)
	targetEmail := normalizeEmail(r.URL.Query().Get("user"))
	if targetEmail == "" {
		targetEmail = email
	}
	target, ok := a.parkingStatementTarget(tenant.Slug, email, role, actor, targetEmail)
	if !ok {
		http.NotFound(w, r)
		return
	}
	statement := a.buildParkingStatement(r.Context(), tenant, target, year)
	filename := "parkplatzabrechnung-" + strconv.Itoa(year) + "-" + safeFilenamePart(target.Email) + ".csv"
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	if err := writeParkingStatementCSV(w, statement); err != nil {
		logError("parking statement export failed", err, "tenant", tenant.Slug, "year", year)
	}
}

func (a *app) parkingMonthExport(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email := ac.tenant, ac.email
	profile := a.profileForTenant(email, tenant.Slug)
	if !ac.can(capabilityPlatformAdmin) && !profile.HasPermission(permissionParking) {
		http.NotFound(w, r)
		return
	}
	month := strings.TrimSpace(r.PathValue("month"))
	if _, err := time.Parse("2006-01", month); err != nil {
		http.NotFound(w, r)
		return
	}
	view := a.parkingMonthDetails(r.Context(), tenant, month)
	if !view.HasHours && view.Summary.Month == "" {
		http.NotFound(w, r)
		return
	}
	filename := "parkplatzabrechnung-" + month + ".csv"
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	if err := writeParkingMonthCSV(w, tenant, view); err != nil {
		logError("parking month export failed", err, "tenant", tenant.Slug, "month", month)
	}
}

func parkingStatementYear(raw string) (int, bool) {
	year, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || year < 2000 || year > 2100 {
		return 0, false
	}
	return year, true
}

func (a *app) parkingStatementTarget(tenantSlug string, actorEmail string, actorRole string, actor userProfile, targetEmail string) (userProfile, bool) {
	tenantSlug = normalizeSlug(tenantSlug)
	targetEmail = normalizeEmail(targetEmail)
	if tenantSlug == "" || targetEmail == "" {
		return userProfile{}, false
	}
	authorizationActor := actorFor(actorEmail, tenantSlug, actorRole)
	resource := resourceFor(tenantSlug)
	isManager := can(authorizationActor, capabilityManageUsers, resource) || can(authorizationActor, capabilityPlatformAdmin, resource)
	if targetEmail != normalizeEmail(actorEmail) && !isManager {
		return userProfile{}, false
	}
	target := a.profileForTenant(targetEmail, tenantSlug)
	if !target.HasTenant(tenantSlug) {
		return userProfile{}, false
	}
	targetIsAdmin := roleHasCapability(target.Role, capabilityPlatformAdmin)
	targetCanPark := targetIsAdmin || target.HasPermission(permissionParking)
	if !targetCanPark {
		return userProfile{}, false
	}
	if !isManager {
		actorCanPark := can(authorizationActor, capabilityPlatformAdmin, resource) || actor.HasPermission(permissionParking)
		if !actorCanPark || targetEmail != normalizeEmail(actorEmail) {
			return userProfile{}, false
		}
	}
	return target, true
}

func (a *app) buildParkingStatement(ctx context.Context, tenant tenantConfig, user userProfile, year int) parkingStatementView {
	seedCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	if err := a.seedParkingHistory(seedCtx, tenant, a.parkingHistoryStart, time.Now()); err != nil && tenant.HA.Configured() {
		logError("parking history seed failed", err, "tenant", tenant.Slug)
	}
	data := a.parkingStore.TenantData(tenant.Slug)
	months := []parkingMonthView{}
	totalKWh := 0.0
	energyCost := 0.0
	gridCost := 0.0
	baseFee := 0.0
	totalCost := 0.0
	surplusKWh := 0.0
	surplusCost := 0.0
	normalKWh := 0.0
	for _, month := range calculateParkingMonths(data, time.Now(), time.Local) {
		if !strings.HasPrefix(month.Month, strconv.Itoa(year)+"-") {
			continue
		}
		months = append(months, month)
		totalKWh += month.KWhValue
		energyCost += month.EnergyCostValue
		gridCost += month.GridCostValue
		baseFee += month.BaseFeeValue
		totalCost += month.TotalCostValue
		surplusKWh += month.SurplusKWhValue
		surplusCost += month.SurplusCostValue
		normalKWh += month.NormalKWhValue
	}
	return parkingStatementView{
		Tenant:       tenant,
		User:         user,
		Year:         year,
		GeneratedAt:  formatLocalDateTime(time.Now()),
		GridFeeLabel: parkingStatementTariffLabel(data.Settings),
		Months:       months,
		HasMonths:    len(months) > 0,
		TotalKWh:     formatKWh(totalKWh),
		EnergyCost:   formatEUR(energyCost),
		GridCost:     formatEUR(gridCost),
		BaseFee:      formatEUR(baseFee),
		TotalCost:    formatEUR(totalCost),
		SurplusKWh:   formatKWh(surplusKWh),
		SurplusCost:  formatEUR(surplusCost),
		NormalKWh:    formatKWh(normalKWh),
		HasSurplus:   surplusKWh > 0,
	}
}

func (a *app) updateParkingSettings(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actorEmail, role := ac.tenant, ac.email, ac.role
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	tariff, err := parkingTariffFromForm(r.Form)
	if err != nil {
		http.Redirect(w, r, "/app/parking/settings?section=accounting&settings=invalid", http.StatusSeeOther)
		return
	}
	if err := a.parkingStore.UpsertTariff(tenant.Slug, tariff); err != nil {
		logError("parking settings save failed", err, "tenant", tenant.Slug)
		http.Error(w, "Could not save parking settings", http.StatusInternalServerError)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: actorEmail,
		ActorRole:  role,
		Action:     auditActionParkingSettings,
		TargetType: "parking",
		TargetID:   tenant.Slug,
		Summary:    "Parkplatz-Abrechnung geändert",
		Details: map[string]string{
			"effective_from": formatParkingTariffDate(tariff.EffectiveFrom),
			"grid_fee":       formatEURPerKWh(tariff.GridFeeEURPerKWh),
			"base_fee":       formatEUR(tariff.BaseFeeEUR),
		},
	})
	http.Redirect(w, r, "/app/parking/settings?section=accounting&settings=saved", http.StatusSeeOther)
}

func parkingTariffFromForm(values url.Values) (parkingTariff, error) {
	effectiveFrom := normalizeParkingTariffDate(values.Get("effective_from"))
	if effectiveFrom == "" {
		effectiveFrom = time.Now().In(time.Local).Format("2006-01-02")
	}
	gridFee, err := parseDecimal(values.Get("grid_fee_eur_per_kwh"))
	if err != nil || gridFee < 0 || gridFee > 5 {
		return parkingTariff{}, fmt.Errorf("invalid grid fee")
	}
	baseFee := 0.0
	if strings.TrimSpace(values.Get("base_fee_eur")) != "" {
		baseFee, err = parseDecimal(values.Get("base_fee_eur"))
		if err != nil || baseFee < 0 || baseFee > 5000 {
			return parkingTariff{}, fmt.Errorf("invalid base fee")
		}
	}
	surplus := 0.0
	if strings.TrimSpace(values.Get("surplus_rate_eur_per_kwh")) != "" {
		surplus, err = parseDecimal(values.Get("surplus_rate_eur_per_kwh"))
		if err != nil || surplus < 0 || surplus > 5 {
			return parkingTariff{}, fmt.Errorf("invalid surplus rate")
		}
	}
	return normalizeParkingTariff(parkingTariff{
		EffectiveFrom:        effectiveFrom,
		GridFeeEURPerKWh:     gridFee,
		BaseFeeEUR:           baseFee,
		SurplusRateEURPerKWh: surplus,
	}), nil
}

func (a *app) updateParkingMonth(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actorEmail, role := ac.tenant, ac.email, ac.role
	actor := a.profileForTenant(actorEmail, tenant.Slug)
	canManagePayment := ac.can(capabilityManageUsers) || ac.can(capabilityManageParking) || ac.can(capabilityPlatformAdmin)
	canMarkPayment := canManagePayment || actor.HasPermission(permissionParking)
	if !canMarkPayment {
		http.Error(w, "Dieser Bereich ist Admins vorbehalten.", http.StatusForbidden)
		return
	}
	if err := parseMaybeMultipartForm(w, r, maxIssueAttachmentFormBytes, maxAttachmentBytes); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	month := strings.TrimSpace(r.FormValue("month"))
	if _, err := time.Parse("2006-01", month); err != nil {
		http.Redirect(w, r, "/app/parking?month=invalid", http.StatusSeeOther)
		return
	}
	returnPath := "/app/parking/month/" + url.PathEscape(month)
	payment, err := parkingPaymentFromForm(r.Form, actorEmail)
	if err != nil {
		http.Redirect(w, r, returnPath+"?month=invalid", http.StatusSeeOther)
		return
	}
	if !payment.Paid && !canManagePayment {
		http.Error(w, "Dieser Bereich ist Admins vorbehalten.", http.StatusForbidden)
		return
	}
	attachmentHeaders, err := attachmentFormHeaders(r, maxIssueAttachmentCount, "attachments")
	if err != nil {
		http.Redirect(w, r, returnPath+"?month=invalid", http.StatusSeeOther)
		return
	}
	var uploaded []attachmentRecord
	if len(attachmentHeaders) > 0 {
		if a.attachmentStore == nil {
			http.Redirect(w, r, returnPath+"?month=invalid", http.StatusSeeOther)
			return
		}
		uploaded, err = ac.repositories.attachments.CreateUploaded("parking", month, actorEmail, uploadedFilesFromHeaders(attachmentHeaders), time.Now())
		if err != nil {
			logError("parking attachment upload failed", err, "tenant", tenant.Slug, "month", month, "actor", redactedEmail(actorEmail))
			http.Redirect(w, r, returnPath+"?month=invalid", http.StatusSeeOther)
			return
		}
	}
	if err := a.parkingStore.SetMonthPayment(tenant.Slug, month, payment); err != nil {
		for _, attachment := range uploaded {
			_, _, _ = ac.repositories.attachments.Delete(attachment.ID, time.Now())
		}
		logError("parking month save failed", err, "tenant", tenant.Slug)
		http.Error(w, "Could not save parking month", http.StatusInternalServerError)
		return
	}
	details := map[string]string{
		"month": formatMonthLabel(month, time.Local),
		"paid":  paidLabel(payment.Paid),
	}
	if payment.Paid {
		if !payment.PaidAt.IsZero() {
			details["paid_at"] = formatLocalDate(payment.PaidAt.In(time.Local))
		}
		if payment.PaidBy != "" {
			details["paid_by"] = payment.PaidBy
		}
		if payment.PaymentMethod != "" {
			details["payment_method"] = payment.PaymentMethod
		}
		if payment.PaymentReference != "" {
			details["payment_reference"] = payment.PaymentReference
		}
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: actorEmail,
		ActorRole:  role,
		Action:     auditActionParkingMonth,
		TargetType: "parking",
		TargetID:   month,
		Summary:    "Monatsstatus geändert",
		Details:    details,
	})
	http.Redirect(w, r, returnPath+"?month=saved", http.StatusSeeOther)
}

func (a *app) sendParkingReminders(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actorEmail, role := ac.tenant, ac.email, ac.role
	if !ac.can(capabilityManageUsers) && !ac.can(capabilityManageParking) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	returnToAccess := r.FormValue("return_to") == "parking_access"
	sent := a.sendParkingPaymentReminders(tenant, ac.tenantRef, actorEmail, role, time.Now())
	status := "none"
	if sent > 0 {
		status = "sent"
	}
	if returnToAccess {
		http.Redirect(w, r, "/app/settings/parking-access?parking_access=reminder_"+status, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/app/parking?reminder="+status, http.StatusSeeOther)
}

func (a *app) sendParkingPaymentReminders(tenant tenantConfig, tenantRef store.TenantRef, actorEmail string, actorRole string, now time.Time) int {
	if a == nil || a.parkingStore == nil {
		return 0
	}
	if now.IsZero() {
		now = time.Now()
	}
	data := a.parkingStore.TenantData(tenant.Slug)
	months := calculateParkingMonths(data, now, time.Local)
	if len(months) == 0 {
		return 0
	}
	sentTotal := 0
	for _, row := range a.userRows(tenantRef) {
		if !row.ParkingChecked {
			continue
		}
		recipientMonths := parkingReminderMonths(months, data.Months, row.Email)
		if len(recipientMonths) == 0 {
			continue
		}
		monthIDs := make([]string, 0, len(recipientMonths))
		balance := 0.0
		lines := []string{
			"Für " + tenant.Address + " sind Parkplatz-Abrechnungen überfällig.",
			"",
			"Überfällige Monate:",
		}
		for _, month := range recipientMonths {
			monthIDs = append(monthIDs, month.Month)
			balance += month.TotalCostValue
			lines = append(lines, month.MonthLabel+": "+month.TotalCost)
		}
		lines = append(lines, "", "Offener Betrag: "+formatEUR(balance))
		actionURL := tenant.PublicURL("/app/parking")
		if len(recipientMonths) > 0 {
			actionURL = tenant.PublicURL("/app/parking#parking-month-" + url.PathEscape(recipientMonths[0].Month))
		}
		sent := a.notify(portalNotification{
			Event:      notificationEventPayment,
			Tenant:     tenant,
			Recipients: []string{row.Email},
			ActorEmail: actorEmail,
			Subject:    "Zahlungserinnerung Parkplatznutzung",
			ActionText: "Parkplatzabrechnung öffnen",
			ActionURL:  actionURL,
			Lines:      lines,
		})
		if len(sent) == 0 {
			continue
		}
		if err := a.parkingStore.MarkPaymentReminderSent(tenant.Slug, monthIDs, sent, now); err != nil {
			logError("parking reminder mark failed", err, "tenant", tenant.Slug, "recipient", redactedEmail(row.Email))
			continue
		}
		a.recordAudit(auditEvent{
			TenantSlug: tenant.Slug,
			ActorEmail: actorEmail,
			ActorRole:  actorRole,
			Action:     auditActionParkingReminder,
			TargetType: "parking",
			TargetID:   row.Email,
			Summary:    "Zahlungserinnerung gesendet",
			Details: map[string]string{
				"recipients": strconv.Itoa(len(sent)),
				"balance":    formatEUR(balance),
				"month":      strings.Join(monthIDs, ", "),
			},
		})
		sentTotal += len(sent)
	}
	return sentTotal
}

func parkingReminderMonths(months []parkingMonthView, states map[string]parkingMonthState, email string) []parkingMonthView {
	email = normalizeEmail(email)
	if email == "" {
		return nil
	}
	out := []parkingMonthView{}
	for _, month := range months {
		if !month.Overdue {
			continue
		}
		state := normalizeParkingMonthState(states[month.Month])
		if _, ok := state.ReminderSentAt[email]; ok {
			continue
		}
		out = append(out, month)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Month < out[j].Month
	})
	return out
}

func parkingPaymentFromForm(values url.Values, actorEmail string) (parkingMonthState, error) {
	paid := parseBool(values.Get("paid"))
	state := parkingMonthState{Paid: paid}
	if !paid {
		return state, nil
	}
	paidAt := time.Now().In(time.Local)
	if raw := strings.TrimSpace(values.Get("paid_at")); raw != "" {
		parsed, err := parseParkingPaidAt(raw)
		if err != nil {
			return parkingMonthState{}, err
		}
		paidAt = parsed
	}
	state.PaidAt = paidAt.UTC()
	state.PaidBy = normalizeEmail(actorEmail)
	state.PaymentMethod = cleanParkingPaymentField(values.Get("payment_method"))
	state.PaymentReference = cleanParkingPaymentField(values.Get("payment_reference"))
	return state, nil
}

func parkingMessage(monthStatus string, reminderStatus string) (string, bool) {
	switch reminderStatus {
	case "sent":
		return "Zahlungserinnerungen gesendet.", true
	case "none":
		return "Keine überfälligen offenen Parkplatzbeträge mit aktiver Benachrichtigung gefunden.", false
	}
	switch monthStatus {
	case "saved":
		return "Zahlungsstatus gespeichert.", true
	case "invalid":
		return "Bitte Monat und Zahlungsdaten prüfen.", false
	default:
		return "", false
	}
}

func parkingSettingsMessage(status string) (string, bool) {
	switch status {
	case "saved":
		return "Parkplatz-Abrechnung gespeichert.", true
	case "invalid":
		return "Bitte Gültigkeitsdatum, Netzgebühr und Basisgebühr prüfen.", false
	default:
		return "", false
	}
}

func (a *app) parkingAccessSettings(w http.ResponseWriter, r *http.Request, ac authCtx) {
	_, _, _, _, ok := a.parkingAccessContext(w, ac)
	if !ok {
		return
	}
	accessMsg, accessOK := parkingAccessMessage(r.URL.Query().Get("parking_access"))
	rows := a.parkingAccessRows(ac.tenantRef)
	a.render(w, "parkingAccessSettings", a.withBase(ac, map[string]any{
		"Title":                  "Parkplatz-Zugriff",
		"ActivePage":             "settings",
		"ParkingSection":         "access",
		"CanManageParkingConfig": ac.can(capabilityManageParking),
		"AccessRows":             rows,
		"HasAccessRows":          len(rows) > 0,
		"AccessRowsEmpty":        emptyState("Noch keine Zugänge", "Sobald Personen eingeladen sind, kann der Parkplatz-Zugriff hier gepflegt werden."),
		"AccessMsg":              accessMsg,
		"AccessOK":               accessOK,
		"StatementYear":          time.Now().In(time.Local).Year(),
	}))
}

func (a *app) updateParkingAccess(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actorEmail, role, _, ok := a.parkingAccessContext(w, ac)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	targetEmail := normalizeEmail(r.FormValue("email"))
	enabled := r.FormValue("parking") == "1"
	if a.inviteStore == nil {
		http.Redirect(w, r, "/app/settings/parking-access?parking_access=error", http.StatusSeeOther)
		return
	}
	existing, isInvite := a.inviteStore.Get(targetEmail)
	if !isInvite {
		http.Redirect(w, r, "/app/settings/parking-access?parking_access=not_editable", http.StatusSeeOther)
		return
	}
	if !existing.HasTenant(tenant.Slug) {
		http.Redirect(w, r, "/app/settings/parking-access?parking_access=invalid", http.StatusSeeOther)
		return
	}
	effectiveProfile := existing.ForTenant(tenant.Slug)
	if !a.serviceAccessEnabled && isServiceProviderRole(effectiveProfile.Role) {
		http.Error(w, serviceProviderAccessClosedMessage, http.StatusForbidden)
		return
	}
	if normalizeRole(effectiveProfile.Role) == roleAdmin && !ac.can(capabilityPlatformAdmin) {
		http.Redirect(w, r, "/app/settings/parking-access?parking_access=not_editable", http.StatusSeeOther)
		return
	}
	// Permissions are house-scoped in the SQLite identity model. Writing the
	// flat profile would be ignored by an explicit membership and could affect
	// another house, so update exactly this tenant's effective membership.
	updated, changed, err := a.inviteStore.MutateTenantPermissions(
		targetEmail,
		tenant.Slug,
		func(permissions []string) []string {
			return setPermission(permissions, permissionParking, enabled)
		},
	)
	if err != nil {
		logError("parking access update failed", err, "actor", redactedEmail(targetEmail))
		http.Redirect(w, r, "/app/settings/parking-access?parking_access=error", http.StatusSeeOther)
		return
	}
	if !changed {
		http.Redirect(w, r, "/app/settings/parking-access?parking_access=not_editable", http.StatusSeeOther)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: actorEmail,
		ActorRole:  role,
		Action:     auditActionInviteUpdate,
		TargetType: "user",
		TargetID:   updated.Email,
		Summary:    "Parkplatz-Zugriff geändert",
		Details: map[string]string{
			"changed_fields":   "Parkplatz-Zugriff",
			"permissions_from": strings.Join(permissionLabelList(effectiveProfile.Permissions), ", "),
			"permissions_to":   strings.Join(permissionLabelList(updated.ForTenant(tenant.Slug).Permissions), ", "),
		},
	})
	status := "revoked"
	if enabled {
		status = "granted"
	}
	http.Redirect(w, r, "/app/settings/parking-access?parking_access="+status, http.StatusSeeOther)
}

func (a *app) parkingAccessContext(w http.ResponseWriter, ac authCtx) (tenantConfig, string, string, userProfile, bool) {
	if !ac.can(capabilityManageUsers) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return tenantConfig{}, "", "", userProfile{}, false
	}
	return ac.tenant, ac.email, ac.role, a.profileForTenant(ac.email, ac.tenant.Slug), true
}

func parkingAccessMessage(status string) (string, bool) {
	switch status {
	case "granted":
		return "Parkplatz-Zugriff freigegeben.", true
	case "revoked":
		return "Parkplatz-Zugriff entzogen.", true
	case "reminder_sent":
		return "Zahlungserinnerungen gesendet.", true
	case "reminder_none":
		return "Keine überfälligen offenen Parkplatzbeträge mit aktiver Benachrichtigung gefunden.", false
	case "not_editable":
		return "Dieser Eintrag kommt aus der Konfiguration und kann hier nicht geändert werden.", false
	case "invalid":
		return "Bitte den Zugang prüfen.", false
	case "error":
		return "Der Parkplatz-Zugriff konnte nicht gespeichert werden.", false
	default:
		return "", false
	}
}

func (a *app) parkingAccessRows(tenant store.TenantRef) []userRow {
	rows := a.userRows(tenant)
	balance := a.parkingBalance(tenant.Slug)
	for i := range rows {
		if !rows[i].ParkingChecked || balance.Outstanding <= 0 {
			continue
		}
		rows[i].OutstandingBalance = formatEUR(balance.Outstanding)
		rows[i].HasOutstanding = true
	}
	return rows
}

func (a *app) parkingBalance(tenantSlug string) parkingBalanceView {
	if a == nil || a.parkingStore == nil {
		return parkingBalanceView{}
	}
	data := a.parkingStore.TenantData(tenantSlug)
	return parkingBalanceSummary(calculateParkingMonths(data, time.Now(), time.Local))
}

func (a *app) parkingTelemetry(ctx context.Context, tenant tenantConfig) parkingTelemetry {
	ha := tenant.HA
	telemetry := parkingTelemetry{
		Configured: ha.BaseURL() != "",
		Entities: []parkingEntityRef{
			{Label: "Zählerstand", EntityID: ha.MeterEnergyEntity()},
			{Label: "Leistung", EntityID: ha.PowerEntity()},
			{Label: "aWATTar Preis", EntityID: ha.PriceEntity()},
		},
	}
	if ha.BaseURL() == "" {
		telemetry.Message = "Home Assistant ist lokal noch nicht konfiguriert."
		return telemetry
	}
	if ha.Token() == "" {
		telemetry.Message = "Home Assistant ist vorbereitet, aber lokal fehlt noch ein HA_TOKEN."
		return telemetry
	}

	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	var liveEnergy float64
	var livePrice float64
	var hasLiveEnergy bool
	var hasLivePrice bool
	states := []struct {
		label  string
		entity string
	}{
		{label: "Zählerstand", entity: ha.MeterEnergyEntity()},
		{label: "Aktuelle Leistung", entity: ha.PowerEntity()},
		{label: "aWATTar Gesamtpreis", entity: ha.PriceEntity()},
	}
	for _, item := range states {
		state, err := ha.State(ctx, item.entity)
		if err != nil {
			telemetry.Message = "Home Assistant konnte gerade nicht gelesen werden."
			return telemetry
		}
		telemetry.Metrics = append(telemetry.Metrics, parkingMetric{
			Label:  item.label,
			Value:  formatHAValue(state),
			Detail: item.entity,
		})
		switch item.entity {
		case ha.MeterEnergyEntity():
			if value, err := homeassistant.ParseFloat(state.State); err == nil {
				liveEnergy = value
				hasLiveEnergy = true
			}
		case ha.PriceEntity():
			if value, err := homeassistant.ParseFloat(state.State); err == nil {
				livePrice = value
				hasLivePrice = true
			}
		}
	}
	if hasLiveEnergy && hasLivePrice {
		if err := a.parkingStore.AppendSamples(tenant.Slug, []parkingStoredSample{{
			At:             time.Now().UTC(),
			EnergyKWh:      liveEnergy,
			PriceEURPerKWh: livePrice,
		}}); err != nil {
			logError("parking live sample save failed", err, "tenant", tenant.Slug)
		}
	}
	telemetry.Connected = true
	telemetry.Message = "Live aus Home Assistant gelesen."
	return telemetry
}

func (a *app) parkingAccounting(ctx context.Context, tenant tenantConfig) parkingAccountingView {
	seedCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	if err := a.seedParkingHistory(seedCtx, tenant, a.parkingHistoryStart, time.Now()); err != nil && tenant.HA.Configured() {
		logError("parking history seed failed", err, "tenant", tenant.Slug)
	}

	data := a.parkingStore.TenantData(tenant.Slug)
	months := calculateParkingMonths(data, time.Now(), time.Local)
	balance := parkingBalanceSummary(months)
	currentTariff := parkingTariffAt(data.Settings, time.Now(), time.Local)
	tariffs := parkingTariffViews(data.Settings)
	view := parkingAccountingView{
		GridFeeValue:       formatInputFloat(currentTariff.GridFeeEURPerKWh),
		GridFeeLabel:       formatEURPerKWh(currentTariff.GridFeeEURPerKWh),
		BaseFeeValue:       formatInputFloat(currentTariff.BaseFeeEUR),
		BaseFeeLabel:       formatEUR(currentTariff.BaseFeeEUR),
		EffectiveFrom:      currentTariff.EffectiveFrom,
		EffectiveFromLabel: formatParkingTariffDate(currentTariff.EffectiveFrom),
		Tariffs:            tariffs,
		HasTariffs:         len(tariffs) > 0,
		Months:             months,
		HasMonths:          len(months) > 0,
		OutstandingValue:   balance.Outstanding,
		Outstanding:        formatEUR(balance.Outstanding),
		HasOutstanding:     balance.Outstanding > 0,
		OverdueValue:       balance.Overdue,
		Overdue:            formatEUR(balance.Overdue),
		HasOverdue:         balance.Overdue > 0,
		HistoryAvailable:   len(data.EnergySamples) >= 2 && len(data.PriceSamples) > 0,
	}
	if len(data.EnergySamples) > 0 {
		last := data.EnergySamples[len(data.EnergySamples)-1].At.In(time.Local)
		view.LastSampleLabel = formatLocalDateTime(last)
	}
	if len(months) == 0 {
		view.Message = "Noch nicht genug Messpunkte für eine Monatsabrechnung. Die App sammelt ab jetzt eigene Messpunkte und liest zusätzlich verfügbare Home-Assistant-Historie ein."
	} else {
		view.Message = "Kosten werden stündlich aus Zählerdifferenz, aWATTar-Preis und Netzbetreibergebühren berechnet. Historie wird ab " + formatLocalDate(a.parkingHistoryStart) + " aus Home Assistant nachgezogen, soweit dort Statistikdaten vorhanden sind."
	}
	return view
}

func (a *app) parkingMonthDetails(ctx context.Context, tenant tenantConfig, month string) parkingMonthDetailView {
	seedCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	if err := a.seedParkingHistory(seedCtx, tenant, a.parkingHistoryStart, time.Now()); err != nil && tenant.HA.Configured() {
		logError("parking history seed failed", err, "tenant", tenant.Slug)
	}

	data := a.parkingStore.TenantData(tenant.Slug)
	view := calculateParkingMonthDetails(data, month, time.Now(), time.Local)
	view.BackPath = "/app/parking"
	view.GridFeeLabel = formatEURPerKWh(parkingTariffForMonth(data.Settings, month, time.Local).GridFeeEURPerKWh)
	view.Message = "Stundenwerte aus Zählerdifferenz, aWATTar-Preis und stündlicher Netzgebühr; die Monatsbasis steht in der Zusammenfassung."
	if len(data.EnergySamples) > 0 {
		last := data.EnergySamples[len(data.EnergySamples)-1].At.In(time.Local)
		view.LastSampleLabel = formatLocalDateTime(last)
	}
	return view
}

func (a *app) seedParkingHistory(ctx context.Context, tenant tenantConfig, start time.Time, end time.Time) error {
	ha := tenant.HA
	if ha.BaseURL() == "" || ha.Token() == "" || ha.MeterEnergyEntity() == "" || ha.PriceEntity() == "" {
		return nil
	}
	energySamples, priceSamples, err := ha.Statistics(ctx, start, end)
	if err != nil {
		logError("parking statistics backfill failed", err, "tenant", tenant.Slug)
	}
	restStart := end.Add(-35 * 24 * time.Hour)
	if restStart.Before(start) {
		restStart = start
	}
	history, err := ha.History(ctx, restStart, end, []string{ha.MeterEnergyEntity(), ha.PriceEntity()})
	if err != nil {
		if len(energySamples) == 0 && len(priceSamples) == 0 {
			return err
		}
		logError("parking REST history fallback failed", err, "tenant", tenant.Slug)
		return a.parkingStore.AppendReadings(tenant.Slug, energySamples, priceSamples)
	}
	energySamples = append(energySamples, homeassistant.SamplesFromHistory(history[ha.MeterEnergyEntity()])...)
	priceSamples = append(priceSamples, homeassistant.SamplesFromHistory(history[ha.PriceEntity()])...)
	if len(energySamples) == 0 && len(priceSamples) == 0 {
		return nil
	}
	logInfo("parking history seed found samples", "tenant", tenant.Slug, "energy_samples", len(energySamples), "price_samples", len(priceSamples))
	return a.parkingStore.AppendReadings(tenant.Slug, energySamples, priceSamples)
}

func (a *app) startParkingSampler() func() {
	if a.parkingSampleInterval <= 0 {
		logInfo("parking sampler disabled", "reason", "interval_disabled")
		return func() {}
	}
	configuredTenants := 0
	for _, tenant := range a.tenants {
		if tenant.HA.Configured() {
			configuredTenants++
		}
	}
	if configuredTenants == 0 {
		logInfo("parking sampler disabled", "reason", "no_configured_home_assistant_tenants")
		return func() {}
	}
	logInfo("parking sampler enabled", "tenants", configuredTenants, "interval", a.parkingSampleInterval)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		a.backfillParkingTenants(ctx)
		a.sampleParkingTenants(ctx)
		ticker := time.NewTicker(a.parkingSampleInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.sampleParkingTenants(ctx)
			}
		}
	}()
	return cancel
}

func (a *app) backfillParkingTenants(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for _, tenant := range a.tenants {
		if tenant.HA.BaseURL() == "" || tenant.HA.Token() == "" {
			continue
		}
		if err := a.seedParkingHistory(ctx, tenant, a.parkingHistoryStart, time.Now()); err != nil {
			logError("parking startup history seed failed", err, "tenant", tenant.Slug)
			continue
		}
		logInfo("parking startup history seed completed", "tenant", tenant.Slug)
	}
}

func (a *app) sampleParkingTenants(ctx context.Context) {
	for _, tenant := range a.tenants {
		if tenant.HA.BaseURL() == "" || tenant.HA.Token() == "" {
			continue
		}
		if err := a.sampleParkingTenant(ctx, tenant); err != nil {
			logError("parking sample failed", err, "tenant", tenant.Slug)
			continue
		}
		logInfo("parking sample saved", "tenant", tenant.Slug)
	}
}

func (a *app) sampleParkingTenant(ctx context.Context, tenant tenantConfig) error {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	energy, err := tenant.HA.State(ctx, tenant.HA.MeterEnergyEntity())
	if err != nil {
		return err
	}
	price, err := tenant.HA.State(ctx, tenant.HA.PriceEntity())
	if err != nil {
		return err
	}
	energyValue, err := homeassistant.ParseFloat(energy.State)
	if err != nil {
		return err
	}
	priceValue, err := homeassistant.ParseFloat(price.State)
	if err != nil {
		return err
	}
	return a.parkingStore.AppendSamples(tenant.Slug, []parkingStoredSample{{
		At:             time.Now().UTC(),
		EnergyKWh:      energyValue,
		PriceEURPerKWh: priceValue,
	}})
}

func parkingTariffAt(settings parkingSettings, at time.Time, loc *time.Location) parkingTariff {
	settings = normalizeParkingSettings(settings)
	if loc == nil {
		loc = time.Local
	}
	if at.IsZero() {
		at = time.Now()
	}
	day := at.In(loc).Format("2006-01-02")
	selected := settings.Tariffs[0]
	for _, tariff := range settings.Tariffs {
		if tariff.EffectiveFrom <= day {
			selected = tariff
			continue
		}
		break
	}
	return selected
}

func parkingTariffForMonth(settings parkingSettings, month string, loc *time.Location) parkingTariff {
	if loc == nil {
		loc = time.Local
	}
	first, err := time.ParseInLocation("2006-01", month, loc)
	if err != nil {
		return parkingTariffAt(settings, time.Now(), loc)
	}
	return parkingTariffAt(settings, first, loc)
}

func parkingTariffViews(settings parkingSettings) []parkingTariffView {
	settings = normalizeParkingSettings(settings)
	views := make([]parkingTariffView, 0, len(settings.Tariffs))
	for i := len(settings.Tariffs) - 1; i >= 0; i-- {
		tariff := settings.Tariffs[i]
		views = append(views, parkingTariffView{
			EffectiveFrom:      formatParkingTariffDate(tariff.EffectiveFrom),
			EffectiveFromInput: tariff.EffectiveFrom,
			GridFee:            formatEURPerKWh(tariff.GridFeeEURPerKWh),
			BaseFee:            formatEUR(tariff.BaseFeeEUR),
		})
	}
	return views
}

func parkingBalanceSummary(months []parkingMonthView) parkingBalanceView {
	var summary parkingBalanceView
	for _, month := range months {
		if month.Outstanding {
			summary.Outstanding += month.TotalCostValue
		}
		if month.Overdue {
			summary.Overdue += month.TotalCostValue
		}
	}
	return summary
}

func parkingPaymentDetails(state parkingMonthState, loc *time.Location) string {
	state = normalizeParkingMonthState(state)
	if !state.Paid {
		return ""
	}
	if loc == nil {
		loc = time.Local
	}
	parts := []string{}
	if !state.PaidAt.IsZero() {
		parts = append(parts, "bezahlt am "+formatDateTimeIn(state.PaidAt, loc, deATDateLayout))
	}
	if state.PaymentMethod != "" {
		parts = append(parts, state.PaymentMethod)
	}
	if state.PaymentReference != "" {
		parts = append(parts, "Ref. "+state.PaymentReference)
	}
	if state.PaidBy != "" {
		parts = append(parts, "erfasst von "+state.PaidBy)
	}
	return strings.Join(parts, " · ")
}

// StartParkingSampler starts the Home Assistant sampling worker; the returned
// func stops it.
func (a *app) StartParkingSampler() func() { return a.startParkingSampler() }
