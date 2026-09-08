package server

import (
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/web"
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

	if !strings.Contains(body, `<summary>Kontakt hinzufügen<span class="disclosure-chevron" aria-hidden="true">`) {
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

	if !strings.Contains(body, `<summary>Notiz, Region und Qualifikation<span class="disclosure-chevron" aria-hidden="true">`) {
		t.Fatal("optional fields summary missing")
	}

	css, err := web.Assets.ReadFile("assets/portal-shell.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{".disclosure-chevron", "border-radius:50%", "details[open]>summary>.disclosure-chevron svg{transform:rotate(180deg)}", "transition:transform .2s ease", "prefers-reduced-motion:reduce"} {
		if !strings.Contains(string(css), marker) {
			t.Errorf("shared disclosure state/animation missing %q", marker)
		}
	}
}
