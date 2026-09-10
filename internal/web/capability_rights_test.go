package web

import (
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/authz"
	"github.com/inspr-at/hausv-org/internal/store"
)

func TestRechteFormsAndProtectedActionsHAUSV699(t *testing.T) {
	data := RechteData{Editable: true, OrgKey: "org", Policy: &authz.Policy{Rules: store.CapabilityRules{OrgKey: "org"}}}
	body := renderComponent(t, rechteContent(data))
	if strings.Count(body, `data-family=`) != 45 {
		t.Fatal("matrix shape changed")
	}
	if !strings.Contains(body, `method="post" action="/app/verwaltung/rechte"`) || !strings.Contains(body, `name="allowed"`) {
		t.Fatal("switches must work without JavaScript")
	}
	for _, protected := range []string{"manage-users", "platform-admin"} {
		if strings.Contains(body, `name="capability" value="`+protected+`"`) {
			t.Fatalf("protected %s has an editable form", protected)
		}
	}
	for _, family := range authz.RoleFamilies {
		for _, area := range authz.RoleAreas {
			if len(rechteActionOptions(family.Key, area.Key)) != 5 {
				t.Fatalf("missing action %s/%s", family.Key, area.Key)
			}
		}
	}
	if strings.Contains(body, "<script") || strings.Contains(body, "onclick=") || strings.Contains(body, "@Disclosure") {
		t.Fatal("inline script or templ text leaked")
	}
}

func TestUserRightsProtectedCapabilitiesAndScopedFormsHAUSV699(t *testing.T) {
	data := UserRightsData{Enabled: true, Editable: true, CurrentTenant: "demo", Scopes: map[string][]RightsScope{"person@example.com": {{Slug: "demo", Label: "Demoliegenschaft"}}}}
	body := renderComponent(t, UserOwnRights(data, "person@example.com"))
	for _, protected := range []string{"manage-users", "platform-admin"} {
		if strings.Contains(body, `value="`+protected+`"`) {
			t.Fatal("protected user option offered")
		}
	}
	for _, marker := range []string{`name="tenant_slug"`, `value="grant"`, `value="deny"`, `value="Standard"`, "Profil anwenden", `/app/settings/users/rights`} {
		if !strings.Contains(body, marker) {
			t.Fatalf("missing form contract %s", marker)
		}
	}
}
