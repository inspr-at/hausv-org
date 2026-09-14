package web

import (
	"strings"
	"testing"
)

func TestRolePreviewBandAndChooserRenderOnlyWhenConfigured(t *testing.T) {
	base := PortalPageData{
		Title: "Portal", HouseName: "Haus A", Address: "Musterweg 1",
		DisplayName: "Ada Admin", Initials: "AA", Role: "Admin",
	}
	without := renderComponent(t, PortalPage(base))
	for _, unexpected := range []string{"role-preview-band", "role-preview-chooser", "Ansicht beenden"} {
		if strings.Contains(without, unexpected) {
			t.Fatalf("empty preview data unexpectedly rendered %q", unexpected)
		}
	}

	base.RolePreview = &RolePreviewState{RoleLabel: "Eigentümer", HouseName: "Haus A", EndsInMinutes: 12, EndPath: "/app/ansicht/ende"}
	base.RolePreviewChoices = []RolePreviewChoice{
		{Role: "Hausverwaltung", Label: "Hausverwaltung", Description: "Ihre Rolle · Portfolio, Posteingang, alle Häuser", Current: true},
		{Role: "Eigentümer", Label: "Eigentümer", Description: "Ruhiger Hausüberblick, Dokumente, Abstimmungen", StartPath: "/app/ansicht/start"},
		{Role: "Bewohner", Label: "Bewohner", Description: "Hausüberblick, Aushang, eigene Anliegen", StartPath: "/app/ansicht/start"},
	}
	with := renderComponent(t, PortalPage(base))
	for _, want := range []string{
		"Ansicht als Eigentümer · Haus A · schreibgeschützt · endet in 12 Min",
		`action="/app/ansicht/ende"`, `role-preview-chooser`,
		`action="/app/ansicht/start"`, `name="role" value="Eigentümer"`,
		"Hausverwaltung", "Ihre Rolle · Portfolio, Posteingang, alle Häuser",
		"Ruhiger Hausüberblick, Dokumente, Abstimmungen", "Bewohner",
		"Hausüberblick, Aushang, eigene Anliegen", "Bestimmte Person (Supportansicht)",
		"Separate Freigabe erforderlich", `class="role-preview-row role-preview-row-disabled" aria-disabled="true"`,
		"15 Minuten · schreibgeschützt · wird protokolliert", "Aktuell",
	} {
		if !strings.Contains(with, want) {
			t.Fatalf("configured preview missing %q", want)
		}
	}
	if strings.Contains(with, `name="role" value="Hausverwaltung"`) {
		t.Fatal("the current role must not be a submit target")
	}
}
