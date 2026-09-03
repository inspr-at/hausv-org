package web

import (
	"bytes"
	"strings"
	"testing"
)

func TestInboxPageRendersQueueCaseAndScopedControls(t *testing.T) {
	data := InboxData{
		Eyebrow:          "HAUSVERWALTUNG MUSTERSTADT · POSTEINGANG",
		Lede:             "37 offen · 29 mit Vorschlag · 5 nicht zugeordnet · 11 heute automatisch erledigt",
		Houses:           []InboxHouse{{Slug: "haus-a", Name: "Haus A"}},
		Items:            []InboxItem{{ID: "in-1", Source: "E-MAIL", Subject: "Wasser im Keller", House: "Haus A", Unit: "Top 1", Status: "Vorschlag liegt vor", StatusTone: "ok", Proposal: "Vorschlag: Reparatur · Hoch", Priority: "Hoch", Selected: true}},
		AutoItems:        []InboxItem{{ID: "auto-1", Subject: "Termin", Time: "09:15"}},
		Selected:         &InboxCase{ID: "in-1", Subject: "Wasser im Keller", Body: "Nachricht", Status: "Vorschlag liegt vor", StatusTone: "ok", HasSuggestion: true, ProviderLabel: "Cloud (OpenRouter)", CategoryLabel: "Reparatur/Mangel", Priority: "Hoch", Confidence: 95, HouseLabel: "Haus A", Unit: "Top 1", AssigneeLabel: "Vera", Due: "morgen", Reply: "Wir kümmern uns.", Created: "Eingegangen", Actions: []string{"Hausbetreuung informieren"}, Position: 2, Total: 37},
		ProviderFootline: "KI: Cloud (OpenRouter) · Zielbetrieb lokal im Büro",
		OpenCount:        37,
		UnassignedCount:  5,
		ProposedCount:    29,
		AllFilterURL:     "/app/verwaltung/posteingang",
		UnassignedURL:    "/app/verwaltung/posteingang?unassigned=1",
		OpenFilterURL:    "/app/verwaltung/posteingang?status=open",
		ProposedURL:      "/app/verwaltung/posteingang?status=proposed",
		ScriptNonce:      "test-nonce",
	}
	var out bytes.Buffer
	if err := InboxPage(VerwaltungShell{OrganisationName: "Musterstadt", Active: "inbox"}, data).Render(t.Context(), &out); err != nil {
		t.Fatal(err)
	}
	body := out.String()
	for _, want := range []string{"class=\"queue\"", "class=\"case\"", "class=\"conf\"", "class=\"bar\"", "Telefonnotiz", "Übernehmen &amp; weiter", "J/K Wechseln", "Vorschlag liegt vor", "Heute automatisch erledigt · 1", "Zielbetrieb lokal im Büro", "htmx.min.js", "nonce=\"test-nonce\""} {
		if !strings.Contains(body, want) {
			t.Fatalf("render missing %q", want)
		}
	}
	if strings.Contains(body, "class=\"assign\"") {
		t.Fatal("queue must not contain inline assignment controls")
	}
}

func TestInboxSuggestionStatesAndUnassignedCase(t *testing.T) {
	for _, test := range []struct {
		state string
		want  string
	}{
		{state: "running", want: "Vorschlag wird erstellt"},
		{state: "arrived", want: "Vorschlag eingetroffen"},
		{state: "failed", want: "Erneut versuchen"},
		{state: "cancelled", want: "Abgebrochen"},
	} {
		var out bytes.Buffer
		item := InboxCase{ID: "in-1", SuggestionState: test.state, SuggestionTimeout: 45, ProviderLabel: "Cloud (OpenRouter)", SuggestionError: "Zeitüberschreitung", SuggestionFinished: "12:04", CanSuggest: true}
		if err := InboxSuggestionPartial(item).Render(t.Context(), &out); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), test.want) {
			t.Fatalf("state %s missing %q: %s", test.state, test.want, out.String())
		}
	}

	var out bytes.Buffer
	item := InboxCase{ID: "in-2", Subject: "Ohne Haus", Status: "Nicht zugeordnet", StatusTone: "neutral", Unassigned: true, Body: "Nachricht", Houses: []InboxOption{{Value: "haus-a", Label: "Haus A"}}}
	if err := InboxCaseDetail(item).Render(t.Context(), &out); err != nil {
		t.Fatal(err)
	}
	body := out.String()
	if !strings.Contains(body, "Haus zuordnen") || strings.Contains(body, "Vorschlag anfordern") || strings.Contains(body, "Einordnung") {
		t.Fatalf("unexpected unassigned case controls: %s", body)
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
