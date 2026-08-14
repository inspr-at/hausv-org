package server

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestHAUSV200UserRowsPutAttentionFirstAndExposeUnits(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email: "admin@example.com", Role: roleAdmin,
		Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods(),
	})
	if _, err := a.inviteStore.Add(userProfile{
		Email: "disabled@example.com", FirstName: "Dora", LastName: "Deaktiviert",
		Role: roleRenter, Status: "Aktiv", Deactivated: true,
		Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods(),
	}); err != nil {
		t.Fatalf("seed disabled user: %v", err)
	}
	a.profiles["owner@example.com"] = userProfile{
		Email: "owner@example.com", FirstName: "Otto", LastName: "Eigentümer",
		Role: roleOwner, Status: "Eingeladen",
		Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods(),
	}
	if err := a.unitStore.SetTenantUnits("demo", []unit{{
		ID:          "einheit-12",
		TenantSlug:  "demo",
		Label:       "Einheit 12",
		OwnerEmails: []string{"owner@example.com"},
	}}); err != nil {
		t.Fatalf("seed unit: %v", err)
	}

	rows := a.userRows("demo")
	if len(rows) < 3 {
		t.Fatalf("rows = %+v, want seeded users and admin", rows)
	}
	if rows[0].Email != "disabled@example.com" {
		t.Fatalf("attention order starts with %q, want deactivated user", rows[0].Email)
	}
	if !(userStatusSortRank("Deaktiviert") < userStatusSortRank("Eingeladen") &&
		userStatusSortRank("Eingeladen") < userStatusSortRank("Aktiv")) {
		t.Fatal("status ranks must put deactivated and invited access before active access")
	}
	owner := userRowForEmail(t, rows, "owner@example.com")
	if !owner.HasUnits || len(owner.UnitList) != 1 || owner.UnitList[0] != "Einheit 12 · Eigentümer" {
		t.Fatalf("owner units = %+v, has=%v", owner.UnitList, owner.HasUnits)
	}
}

func TestHAUSV200UserSettingsUsesProgressiveDisclosureAndContextualGate(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email: "admin@example.com", Role: roleAdmin,
		Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods(),
	})
	if _, err := a.inviteStore.Add(userProfile{
		Email: "invitee@example.com", Role: roleRenter, Status: "Eingeladen",
		Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods(),
	}); err != nil {
		t.Fatalf("seed invite: %v", err)
	}

	page := authedRequest(t, a, "admin@example.com", "/demo/app/settings/users")
	if page.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", page.Code)
	}
	body := page.Body.String()
	for _, want := range []string{
		`class="access-metrics"`,
		`id="invite"`,
		`Name ergänzen`,
		`Zugang anpassen`,
		`Einheit verknüpfen`,
		`Zuordnung bei Gebäude &amp; Einheiten verwalten`,
		`Anmeldung &amp; Sonderrechte`,
		`Zugang dauerhaft entfernen`,
		`class="col-secondary"`,
		`class="col-auth"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("user settings should contain %q", want)
		}
	}
	inviteStart := strings.Index(body, `id="invite"`)
	gate := strings.Index(body, "Dienstleister-Zugänge können noch nicht")
	if inviteStart < 0 || gate < inviteStart {
		t.Fatal("service-provider gate must be contextual inside the invitation flow")
	}
	if strings.Contains(body[:inviteStart], "Dienstleister-Zugänge können noch nicht") {
		t.Fatal("service-provider gate must not be a page-wide warning")
	}
}

func TestHAUSV200UserActionRedirectsReturnToRelevantSection(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email: "admin@example.com", Role: roleAdmin,
		Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods(),
	})

	invalid := authedFormRequest(t, a, "admin@example.com", "/demo/app/settings/users", url.Values{
		"email": {"not-an-email"},
	})
	if invalid.Code != http.StatusSeeOther || !strings.HasSuffix(invalid.Header().Get("Location"), "#invite") {
		t.Fatalf("invalid invite redirect = %d %q", invalid.Code, invalid.Header().Get("Location"))
	}

	if _, err := a.inviteStore.Add(userProfile{
		Email: "invitee@example.com", Role: roleRenter,
		Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods(),
	}); err != nil {
		t.Fatalf("seed invite: %v", err)
	}
	updated := authedFormRequest(t, a, "admin@example.com", "/demo/app/settings/users/edit", url.Values{
		"orig_email":   {"invitee@example.com"},
		"email":        {"invitee@example.com"},
		"role":         {roleOwner},
		"auth_methods": {"email"},
	})
	if updated.Code != http.StatusSeeOther || !strings.HasSuffix(updated.Header().Get("Location"), "#access-list") {
		t.Fatalf("edit redirect = %d %q", updated.Code, updated.Header().Get("Location"))
	}
}
