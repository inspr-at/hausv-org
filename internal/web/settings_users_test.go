package web

import (
	"reflect"
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/view"
)

func TestUserAccessGroupsPreserveAssignmentsAndRights(t *testing.T) {
	user := view.UserRow{
		Role: "Verwalter",
		UnitAssignments: []view.UserUnitAssignment{
			{Label: "Garage hinten · Mieter", Kind: "Stellplatz"},
			{Label: "Atelier · Eigentümer", Kind: "Wohnung"},
			{Label: "Archiv · Eigentümer", Kind: "Keller / Lager"},
		},
		RoleCapabilities: []string{"Aushang verwalten", "Dokumente verwalten", "Benutzer verwalten", "Gebäude verwalten", "Weiteres bestehendes Recht"},
	}
	groups := userUnitGroups(user)
	if len(groups) != 3 || groups[0].Label != "Wohnungen" || groups[0].Items[0] != "Atelier · Eigentümer" || groups[1].Label != "Stellplätze" || groups[1].Items[0] != "Garage hinten · Mieter" {
		t.Fatalf("group by stored kind, never by a guessed name: %+v", groups)
	}
	var rights []string
	for _, group := range userCapabilityGroups(user) {
		rights = append(rights, group.Items...)
	}
	if !reflect.DeepEqual(rights, user.RoleCapabilities) {
		t.Fatalf("grouping changed existing capability labels: %v", rights)
	}
}

func TestUserAccessDetailKeepsProtectedGatesAndEscapesContent(t *testing.T) {
	data := UserSettingsPageData{Portal: PortalPageData{HouseName: "Testhaus"}}
	user := view.UserRow{Email: "person@example.com", DisplayName: "<script>name</script>", Role: "Admin", IsConfig: true, Protected: true, PermissionList: []string{"Standard"}}
	body := renderComponent(t, UserAccessDetail(data, user))
	if strings.Contains(body, "data-access-edit") || strings.Contains(body, "<script>") {
		t.Fatal("protected detail must neither offer edits nor render user markup")
	}
	if !strings.Contains(body, "Berechtigungen im Detail") || !strings.Contains(body, "schreibgeschützt") {
		t.Fatal("protected access must retain its readable detail")
	}
	user.Protected = false
	if !strings.Contains(renderComponent(t, UserAccessDetail(data, user)), "data-access-edit") {
		t.Fatal("unprotected config access must retain adopt-on-edit")
	}
	user.Role = "Dienstleister"
	if strings.Contains(renderComponent(t, UserAccessDetail(data, user)), "data-access-edit") {
		t.Fatal("closed service-provider access must stay read-only")
	}
}
