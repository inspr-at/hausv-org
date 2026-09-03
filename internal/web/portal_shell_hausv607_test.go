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
	base.RolePreviewChoices = []RolePreviewChoice{{Role: "Eigentümer", Label: "Eigentümer", Description: "Eigentümeransicht", StartPath: "/app/ansicht/start", Current: true}}
	with := renderComponent(t, PortalPage(base))
	for _, want := range []string{
		"Ansicht als Eigentümer · Haus A · schreibgeschützt · endet in 12 Min",
		`action="/app/ansicht/ende"`, `class="role-preview-chooser"`,
		`action="/app/ansicht/start"`, `name="role" value="Eigentümer"`,
		"15 Minuten · schreibgeschützt · wird protokolliert", "Aktuell",
	} {
		if !strings.Contains(with, want) {
			t.Fatalf("configured preview missing %q", want)
		}
	}
}
