package web

import (
	"bytes"
	"strings"
	"testing"
)

func TestInboxPageRendersQueueCaseAndScopedControls(t *testing.T) {
	data := InboxData{
		Eyebrow:          "HAUSVERWALTUNG MUSTERSTADT · POSTEINGANG",
		Lede:             "37 offen · 29 mit Vorschlag · 5 unzugeordnet · 11 heute automatisch erledigt",
		Houses:           []InboxHouse{{Slug: "haus-a", Name: "Haus A"}},
		Items:            []InboxItem{{ID: "in-1", Source: "E-MAIL", Subject: "Wasser im Keller", House: "Haus A", Unit: "Top 1", Proposal: "Vorschlag: Reparatur · Hoch", Priority: "Hoch", Selected: true}},
		AutoItems:        []InboxItem{{ID: "auto-1", Subject: "Termin", Time: "09:15"}},
		Selected:         &InboxCase{ID: "in-1", Subject: "Wasser im Keller", Body: "Nachricht", HasSuggestion: true, ProviderLabel: "Cloud (OpenRouter)", CategoryLabel: "Reparatur/Mangel", Priority: "Hoch", Confidence: 95, HouseLabel: "Haus A", Unit: "Top 1", AssigneeLabel: "Vera", Due: "morgen", Reply: "Wir kümmern uns.", Created: "Eingegangen", Actions: []string{"Hausbetreuung informieren"}},
		ProviderFootline: "KI: Cloud (OpenRouter) · Zielbetrieb lokal im Büro",
	}
	var out bytes.Buffer
	if err := InboxPage(VerwaltungShell{OrganisationName: "Musterstadt", Active: "inbox"}, data).Render(t.Context(), &out); err != nil {
		t.Fatal(err)
	}
	body := out.String()
	for _, want := range []string{"class=\"queue\"", "class=\"case\"", "class=\"conf\"", "class=\"bar\"", "Telefonnotiz", "Freigeben und senden", "Heute automatisch erledigt · 1", "Zielbetrieb lokal im Büro"} {
		if !strings.Contains(body, want) {
			t.Fatalf("render missing %q", want)
		}
	}
	if strings.Count(body, "inbox-action gold") < 1 {
		t.Fatal("gold phone-note action missing")
	}
}

func TestVerwaltungSettingsPageRendersTrustLevels(t *testing.T) {
	data := VerwaltungSettingsData{Categories: []VerwaltungSettingsCategory{{Key: "reparatur", Label: "Reparatur/Mangel", Level: "auto"}}, Threshold: 90, AutoEnabled: true, ProviderLabel: "nicht konfiguriert"}
	var out bytes.Buffer
	if err := VerwaltungSettingsPage(VerwaltungShell{OrganisationName: "Musterstadt"}, data).Render(t.Context(), &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Reparatur/Mangel", "Manuell", "Vorschlag", "Automatisch", "Schwellwert für Automatisch", "nicht konfiguriert"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("settings render missing %q", want)
		}
	}
}
