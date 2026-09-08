package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestGeneralHelpForEveryPortalRole(t *testing.T) {
	for _, role := range []string{roleAdmin, roleManager, roleOwner, roleRenter, roleResident, roleBeirat} {
		t.Run(role, func(t *testing.T) {
			a, _, _ := newInboxTestApp(t, role)
			tenant := a.tenants["demo"]
			tenant.ContactName, tenant.ContactEmail, tenant.ContactPhone = "Verwaltung Park", "park@example.com", "+43 316 123456"
			a.tenants["demo"] = tenant
			page := authedRequest(t, a, "vera@example.com", "/demo/app/hilfe")
			if page.Code != http.StatusOK {
				t.Fatalf("status=%d", page.Code)
			}
			body := page.Body.String()
			for _, want := range []string{`data-general-help`, `<h1>Hilfe</h1>`, "Was finde ich wo", "Hausüberblick", "Aushang", "Termine", "Kontakte", "Dokumente", "Anliegen melden &amp; verfolgen", "Einstellungen &amp; Benachrichtigungen", "Liegenschaft wechseln", "Abmelden", "E-Mail-Code", "Profil und Sichtbarkeit", "Wer sieht was", "Verwaltung Park", `href="mailto:park@example.com"`, "+43 316 123456", `href="/demo/app/hilfe/energie"`, `href="/demo/app/hilfe" class="nav-item active"`} {
				if !strings.Contains(body, want) {
					t.Errorf("missing %q", want)
				}
			}
			if strings.Contains(body, `data-pairing-code`) || strings.Contains(body, `/connector/pairing`) || strings.Contains(body, "GET /api/states") {
				t.Fatal("connector setup leaked into general help")
			}
			content := body[strings.Index(body, `data-general-help`):]
			votes := role == roleOwner || role == roleBeirat
			if strings.Contains(content, `href="/demo/app/abstimmungen"`) != votes {
				t.Fatal("wrong vote help for role")
			}
			management := role == roleAdmin || role == roleManager
			if strings.Contains(content, "Für die Verwaltung") != management {
				t.Fatal("wrong management help for role")
			}
			if management {
				for _, want := range []string{"Portfolio", "Posteingang", "Textbausteine", "Rechte", "Triage-Board"} {
					if !strings.Contains(content, want) {
						t.Errorf("missing management help %s", want)
					}
				}
			}
		})
	}
}

func TestEnergyHelpRevokeRedirectAndOneTimePairing(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "owner@example.com", Role: roleOwner, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	pairing := authedFormRequest(t, a, "owner@example.com", "/demo/app/hilfe/energie", nil)
	if pairing.Code != http.StatusOK || !strings.Contains(pairing.Body.String(), "data-pairing-code") {
		t.Fatal("pairing code missing")
	}
	get := authedRequest(t, a, "owner@example.com", "/demo/app/hilfe/energie")
	if strings.Contains(get.Body.String(), "data-pairing-code") {
		t.Fatal("pairing code displayed again")
	}
	revoke := authedFormRequest(t, a, "owner@example.com", "/demo/app/hilfe/connector/revoke", nil)
	if revoke.Code != http.StatusSeeOther || revoke.Header().Get("Location") != "/demo/app/hilfe/energie?connector=revoked" {
		t.Fatalf("revoke redirect=%d %s", revoke.Code, revoke.Header().Get("Location"))
	}
	for _, role := range []string{roleRenter, roleResident, roleBeirat} {
		a.profiles["restricted@example.com"] = userProfile{Email: "restricted@example.com", Role: role, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
		post := authedFormRequest(t, a, "restricted@example.com", "/demo/app/hilfe/energie", nil)
		if post.Code != http.StatusForbidden {
			t.Fatalf("role %s new pairing endpoint=%d", role, post.Code)
		}
	}
}
