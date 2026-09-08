package server

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestContactsUXPrioritizesOfficialActionsAndProgressiveManagement(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "manager@example.com",
		Role:        roleManager,
		Tenants:     []string{"demo"},
		AuthMethods: defaultAuthMethods(),
	})
	a.serviceAccessEnabled = true
	tenant := a.tenants["demo"]
	tenant.ContactName = "Hausverwaltung Nord"
	tenant.ContactEmail = "office@example.com"
	tenant.ContactPhone = "+43 316 100"
	tenant.EmergencyName = "Wasser-Notdienst"
	tenant.EmergencyPhone = "+43 316 911"
	tenant.CaretakerName = "Hausmeister Max"
	tenant.CaretakerEmail = "hausmeister@example.com"
	tenant.CaretakerPhone = "+43 664 300"
	a.tenants["demo"] = tenant
	a.profiles["resident@example.com"] = userProfile{
		Email:          "resident@example.com",
		FirstName:      "Resi",
		LastName:       "Dent",
		Phone:          "+43 664 400",
		DirectoryOptIn: true,
		Role:           roleResident,
		Tenants:        []string{"demo"},
		AuthMethods:    defaultAuthMethods(),
	}
	if _, _, err := testRepositories(a, "demo").contacts.Upsert(managedContact{
		TenantSlug: "demo", Kind: "Dienstleister", Name: "Liftservice", Email: "lift@example.com", Phone: "+43 316 500", Active: true,
	}); err != nil {
		t.Fatalf("seed active contact: %v", err)
	}
	if _, _, err := testRepositories(a, "demo").contacts.Upsert(managedContact{
		TenantSlug: "demo", Kind: "Sonstiges", Name: "Alter Kontakt", Email: "alt@example.com", Active: false,
	}); err != nil {
		t.Fatalf("seed inactive contact: %v", err)
	}

	body := authedRequest(t, a, "manager@example.com", "/demo/app/kontakte").Body.String()
	quick := strings.Index(body, "Schnell erreichen")
	managed := strings.Index(body, "Weitere Kontakte")
	directory := strings.Index(body, "Freiwilliges Verzeichnis")
	if quick < 0 || managed <= quick || directory <= managed {
		t.Fatalf("contact section order quick=%d managed=%d directory=%d", quick, managed, directory)
	}
	for _, want := range []string{
		`class="contact-route" href="tel:`,
		`43 316 911`,
		`class="contact-route" href="mailto:office@example.com"`,
		`<details class="contact-add" id="contact-add"`,
		`<summary>Kontakt hinzufügen</summary>`,
		`<summary>Inaktive Kontakte (1)</summary>`,
		`href="/demo/app/settings/building?section=contacts"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("contacts UX missing %q", want)
		}
	}

	residentBody := authedRequest(t, a, "resident@example.com", "/demo/app/kontakte").Body.String()
	for _, forbidden := range []string{`id="contact-add"`, "Alter Kontakt", "Inaktive Kontakte", "Kontakte der Liegenschaft pflegen"} {
		if strings.Contains(residentBody, forbidden) {
			t.Fatalf("resident contacts leaked management UI %q", forbidden)
		}
	}
}

func TestContactsUXUsesOneHelpfulEmptyState(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "resident@example.com",
		Role:        roleResident,
		Tenants:     []string{"demo"},
		AuthMethods: defaultAuthMethods(),
	})
	body := authedRequest(t, a, "resident@example.com", "/demo/app/kontakte").Body.String()
	if got := strings.Count(body, `class="blank"`); got != 1 {
		t.Fatalf("blank-state count = %d, want 1", got)
	}
	if !strings.Contains(body, "Noch keine Kontakte hinterlegt") || !strings.Contains(body, "Die Hausverwaltung hat für diese Liegenschaft") {
		t.Fatalf("helpful combined empty state missing:\n%s", body)
	}
}

func TestContactWritesReturnToTheirWorkArea(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "manager@example.com",
		Role:        roleManager,
		Tenants:     []string{"demo"},
		AuthMethods: defaultAuthMethods(),
	})
	a.serviceAccessEnabled = true

	invalid := authedFormRequest(t, a, "manager@example.com", "/demo/app/kontakte", url.Values{
		"kind": {"Hausmeister"},
	})
	if invalid.Code != http.StatusSeeOther || invalid.Header().Get("Location") != "/demo/app/kontakte?contact=invalid#contact-add" {
		t.Fatalf("invalid redirect = %d %q", invalid.Code, invalid.Header().Get("Location"))
	}

	saved := authedFormRequest(t, a, "manager@example.com", "/demo/app/kontakte", url.Values{
		"kind": {"Hausmeister"}, "name": {"Hausmeister Max"}, "phone": {"+43 664 300"}, "active": {"true"},
	})
	if saved.Code != http.StatusSeeOther || saved.Header().Get("Location") != "/demo/app/kontakte?contact=saved#contact-book" {
		t.Fatalf("saved redirect = %d %q", saved.Code, saved.Header().Get("Location"))
	}
}
