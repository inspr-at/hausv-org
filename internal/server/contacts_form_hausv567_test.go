package server

import (
	"strings"
	"testing"
)

func TestContactsFormStaysClosedWhenManagedContactsEmpty(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "manager@example.com",
		Role:        roleManager,
		Tenants:     []string{"demo"},
		AuthMethods: defaultAuthMethods(),
	})

	body := authedRequest(t, a, "manager@example.com", "/demo/app/kontakte").Body.String()

	if !strings.Contains(body, `<details class="contact-add" id="contact-add"`) {
		t.Fatal("contact-add details element missing")
	}

	if strings.Contains(body, `<details class="contact-add" id="contact-add" open`) {
		t.Fatal("contact-add form should NOT be auto-opened when managed contacts are empty")
	}

	if !strings.Contains(body, `<summary>Kontakt hinzufügen</summary>`) {
		t.Fatal("add contact summary missing")
	}

	if !strings.Contains(body, `href="#contact-add"`) {
		t.Fatal("header link to #contact-add missing")
	}
}

func TestContactsFormOpensOnValidationError(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "manager@example.com",
		Role:        roleManager,
		Tenants:     []string{"demo"},
		AuthMethods: defaultAuthMethods(),
	})

	body := authedRequest(t, a, "manager@example.com", "/demo/app/kontakte?contact=invalid").Body.String()

	if !strings.Contains(body, `<details class="contact-add" id="contact-add" open`) {
		openIdx := strings.Index(body, `<details class="contact-add"`)
		snippet := ""
		if openIdx >= 0 && openIdx+150 < len(body) {
			snippet = body[openIdx : openIdx+150]
		}
		t.Fatalf("contact-add form SHOULD be opened when ContactFormOpen is true (validation redisplay)\nSnippet: %s", snippet)
	}
}

func TestOptionalFieldsHaveDisclosureMarker(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "manager@example.com",
		Role:        roleManager,
		Tenants:     []string{"demo"},
		AuthMethods: defaultAuthMethods(),
	})

	body := authedRequest(t, a, "manager@example.com", "/demo/app/kontakte").Body.String()

	if !strings.Contains(body, `<details class="contact-add-optional">`) {
		t.Fatal("contact-add-optional details missing")
	}

	if !strings.Contains(body, `<summary>Region, Qualifikation und Energie-Fähigkeiten</summary>`) {
		t.Fatal("optional fields summary missing")
	}

	styleBlock := body[strings.Index(body, ".contact-form details>summary"):strings.Index(body, ".optional-grid")]
	if !strings.Contains(styleBlock, `content:"▶"`) {
		t.Fatal("disclosure marker CSS missing from .contact-form details>summary")
	}
	if !strings.Contains(styleBlock, "rotate(90deg)") {
		t.Fatal("open state disclosure marker CSS missing")
	}
}
