package server

import (
	"net/http"
	"strings"
	"testing"
)

// HAUSV-179: the server ignores identity edits from a house admin (HAUSV-169
// AC8). The form must say so rather than offering fields that silently do
// nothing.

func usersPageFor(t *testing.T, actorRole string) string {
	t.Helper()
	a := newTestPortalApp(t, userProfile{
		Email: "actor@example.com", Role: actorRole,
		Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods(),
	})
	if _, err := a.inviteStore.Add(userProfile{
		Email: "anna@example.com", FirstName: "Anna", LastName: "Muster",
		Role: roleRenter, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods(),
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	rr := authedRequest(t, a, "actor@example.com", "/demo/app/settings/users")
	if rr.Code != http.StatusOK {
		t.Fatalf("users page status = %d", rr.Code)
	}
	body := rr.Body.String()
	// Scope to the EDIT dialog. The invite form above it also has name inputs,
	// and that one is fine: creating a person establishes an identity that does
	// not exist yet. AC8 is about CHANGING an existing one.
	idx := strings.Index(body, `name="orig_email"`)
	if idx < 0 {
		t.Fatal("edit dialog not rendered")
	}
	return body[idx:]
}

func TestHouseAdminCannotEditIdentityFieldsInForm(t *testing.T) {
	body := usersPageFor(t, roleManager)

	// Target the EDIT dialog's inputs by their class. The invite form also has
	// first_name/last_name, andthat one is fine: creating a person establishes an
	// identity that does not exist yet. AC8 is about CHANGING an existing one.
	for _, field := range []string{`class="f-vorname"`, `class="f-nachname"`, `class="f-titel"`, `class="f-email"`} {
		idx := strings.Index(body, field)
		if idx < 0 {
			t.Fatalf("field %s missing from the form", field)
		}
		// The input must carry readonly; look within the tag itself.
		end := strings.Index(body[idx:], ">")
		if end < 0 {
			t.Fatalf("malformed input for %s", field)
		}
		tag := body[idx : idx+end]
		if !strings.Contains(tag, "readonly") {
			t.Fatalf("a house admin must not be offered an editable %s: <input %s>", field, tag)
		}
	}
	if !strings.Contains(body, "zentral von der Plattform-Administration") {
		t.Fatal("the form should explain why identity is not editable here")
	}
}

func TestPlatformAdminKeepsEditableIdentityFields(t *testing.T) {
	body := usersPageFor(t, roleAdmin)

	idx := strings.Index(body, `class="f-vorname"`)
	if idx < 0 {
		t.Fatal("edit-dialog first_name field missing")
	}
	end := strings.Index(body[idx:], ">")
	tag := body[idx : idx+end]
	if strings.Contains(tag, "readonly") {
		t.Fatalf("a platform admin must keep identity editable: <input %s>", tag)
	}
	if strings.Contains(body, "zentral von der Plattform-Administration") {
		t.Fatal("the read-only hint should not show for a platform admin")
	}
}
