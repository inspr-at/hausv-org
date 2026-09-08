package web

import (
	"strconv"
	"strings"

	"github.com/inspr-at/hausv-org/internal/view"
)

// These groups only arrange the labels supplied by the existing access model.
// They do not evaluate, grant, or change capabilities.
type userAccessGroup struct {
	Label string
	Items []string
}

func userAccessEditable(data UserSettingsPageData, user view.UserRow) bool {
	return (user.Editable || (user.IsConfig && !user.Protected)) && (data.ServiceProviderAccessEnabled || user.Role != "Dienstleister")
}

func userAccessProfile(user view.UserRow) string {
	if len(user.PermissionList) == 0 || (len(user.PermissionList) == 1 && user.PermissionList[0] == "Standard") {
		return "Standard"
	}
	return "Mit Sonderrechten"
}

func userFilterRoles(users []view.UserRow) []string {
	var roles []string
	seen := map[string]bool{}
	for _, user := range users {
		if !seen[user.Role] {
			roles = append(roles, user.Role)
			seen[user.Role] = true
		}
	}
	return roles
}

func userUnitGroups(user view.UserRow) []userAccessGroup {
	var groups []userAccessGroup
	kinds := []struct{ kind, label string }{{"Wohnung", "Wohnungen"}, {"Stellplatz", "Stellplätze"}, {"Geschäftslokal", "Geschäftslokale"}, {"Keller / Lager", "Keller / Lager"}, {"Sonstiges", "Sonstige Einheiten"}, {"Einheit", "Einheiten"}}
	for _, kind := range kinds {
		group := userAccessGroup{Label: kind.label}
		for _, unit := range user.UnitAssignments {
			if unit.Kind == kind.kind {
				group.Items = append(group.Items, unit.Label)
			}
		}
		if len(group.Items) > 0 {
			groups = append(groups, group)
		}
	}
	// Older callers can still supply display labels without structural metadata.
	if len(user.UnitAssignments) == 0 && len(user.UnitList) > 0 {
		groups = append(groups, userAccessGroup{Label: "Einheiten", Items: user.UnitList})
	}
	return groups
}

func userUnitSummary(user view.UserRow) string {
	var parts []string
	for _, group := range userUnitGroups(user) {
		label := group.Label
		if len(group.Items) == 1 {
			switch label {
			case "Wohnungen":
				label = "Wohnung"
			case "Stellplätze":
				label = "Stellplatz"
			case "Geschäftslokale":
				label = "Geschäftslokal"
			case "Sonstige Einheiten":
				label = "sonstige Einheit"
			case "Einheiten":
				label = "Einheit"
			}
		}
		parts = append(parts, strconv.Itoa(len(group.Items))+" "+label)
	}
	return strings.Join(parts, " · ")
}

func userRoleFamily(role string) string {
	switch role {
	case "Admin", "Verwalter":
		return "Verwaltung"
	case "Eigentümer", "Beirat":
		return "Eigentümergemeinschaft"
	case "Dienstleister":
		return "Dienstleistung"
	default:
		return "Wohnen"
	}
}

func userCapabilityGroups(user view.UserRow) []userAccessGroup {
	var groups []userAccessGroup
	for _, right := range user.RoleCapabilities {
		area := "Allgemeiner Zugang"
		switch right {
		case "Plattformverwaltung", "Alle Bereiche":
			area = "Organisation"
		case "Aushang verwalten":
			area = "Hausgemeinschaft"
		case "Dokumente verwalten", "Eigentümer-Dokumente":
			area = "Dokumente"
		case "Benutzer verwalten":
			area = "Benutzer & Rechte"
		case "Gebäude verwalten":
			area = "Gebäude & Einheiten"
		case "Abstimmungen":
			area = "Entscheidungen"
		case "Übersicht", "Leserechte":
			area = "Aufsicht"
		case "Bewohnerbereich":
			area = "Hausgemeinschaft"
		case "Zugewiesene Anliegen":
			area = "Anliegen"
		}
		found := false
		for i := range groups {
			if groups[i].Label == area {
				groups[i].Items = append(groups[i].Items, right)
				found = true
				break
			}
		}
		if !found {
			groups = append(groups, userAccessGroup{Label: area, Items: []string{right}})
		}
	}
	return groups
}
