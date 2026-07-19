package server

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// Role gating for the new charging routes: residents with the parking
// permission may toggle, only managers may change controller settings or
// touch Telegram links.

func chargingUITestApp(t *testing.T) *app {
	t.Helper()
	a := newTestPortalApp(t, userProfile{
		Email: "admin@example.com", Role: roleAdmin,
		Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods(),
	})
	a.profiles["joerg@example.com"] = userProfile{
		Email: "joerg@example.com", Role: "Bewohner",
		Tenants: []string{"jhw22"}, Permissions: []string{permissionParking},
		AuthMethods: defaultAuthMethods(),
	}
	a.profiles["nopark@example.com"] = userProfile{
		Email: "nopark@example.com", Role: "Bewohner",
		Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods(),
	}
	tgStore, err := newTelegramStore("")
	if err != nil {
		t.Fatal(err)
	}
	a.telegramStore = tgStore
	return a
}

func validChargingForm() url.Values {
	return url.Values{
		"controller_enabled": {"1"},
		"shadow_mode":        {"1"},
		"start_soc_percent":  {"99"},
		"stop_soc_percent":   {"95"},
		"start_feed_in_w":    {"3300"},
		"stop_feed_in_w":     {"1500"},
		"stop_delay_minutes": {"10"},
		"min_on_minutes":     {"10"},
		"min_off_minutes":    {"5"},
	}
}

func TestChargingSettingsAdminOnly(t *testing.T) {
	a := chargingUITestApp(t)
	if rr := authedFormRequest(t, a, "joerg@example.com", "/app/parking/charging/settings", validChargingForm()); rr.Code != http.StatusForbidden {
		t.Fatalf("resident settings post: %d", rr.Code)
	}
	if rr := authedFormRequest(t, a, "admin@example.com", "/app/parking/charging/settings", validChargingForm()); rr.Code != http.StatusSeeOther {
		t.Fatalf("admin settings post: %d", rr.Code)
	}
	data := a.parkingStore.TenantData("jhw22")
	if !data.Settings.Charging.Enabled || !data.Settings.Charging.ShadowMode {
		t.Fatalf("settings not saved: %+v", data.Settings.Charging)
	}
}

func TestChargingSettingsRejectsInvertedHysteresis(t *testing.T) {
	a := chargingUITestApp(t)
	form := validChargingForm()
	form.Set("stop_feed_in_w", "4000") // above start
	rr := authedFormRequest(t, a, "admin@example.com", "/app/parking/charging/settings", form)
	if rr.Code != http.StatusSeeOther || !strings.Contains(rr.Header().Get("Location"), "charging=invalid") {
		t.Fatalf("expected invalid redirect, got %d %s", rr.Code, rr.Header().Get("Location"))
	}
}

func TestChargingToggleNeedsParkingPermission(t *testing.T) {
	a := chargingUITestApp(t)
	// Without HA configured the action must still be authz-gated first:
	// no-permission → 404, permission → redirect with an error flash.
	if rr := authedFormRequest(t, a, "nopark@example.com", "/app/parking/charging/on", url.Values{}); rr.Code != http.StatusNotFound {
		t.Fatalf("no-permission toggle: %d", rr.Code)
	}
	rr := authedFormRequest(t, a, "joerg@example.com", "/app/parking/charging/on", url.Values{})
	if rr.Code != http.StatusSeeOther || !strings.Contains(rr.Header().Get("Location"), "charging=error") {
		t.Fatalf("permitted toggle without HA: %d %s", rr.Code, rr.Header().Get("Location"))
	}
}

func TestTelegramLinkRoutesAdminOnly(t *testing.T) {
	a := chargingUITestApp(t)
	form := url.Values{"email": {"joerg@example.com"}}
	if rr := authedFormRequest(t, a, "joerg@example.com", "/app/parking/charging/telegram/link", form); rr.Code != http.StatusForbidden {
		t.Fatalf("resident link post: %d", rr.Code)
	}
	rr := authedFormRequest(t, a, "admin@example.com", "/app/parking/charging/telegram/link", form)
	if rr.Code != http.StatusSeeOther || !strings.Contains(rr.Header().Get("Location"), "tgcode=") {
		t.Fatalf("admin link post: %d %s", rr.Code, rr.Header().Get("Location"))
	}
}

func TestParkingPageRendersLiveCardStructures(t *testing.T) {
	a := chargingUITestApp(t)
	rr := authedRequest(t, a, "joerg@example.com", "/app/parking")
	if rr.Code != http.StatusOK {
		t.Fatalf("parking page: %d", rr.Code)
	}
	// Without charging entities the live card stays hidden but the page must
	// render — the template handles a zero Live view.
	if strings.Contains(rr.Body.String(), "parking-live") {
		t.Fatal("live card must be hidden when charging is not configured")
	}
}
