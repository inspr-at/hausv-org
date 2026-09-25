package web

import (
	"bytes"
	"regexp"
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
		Selected:         &InboxCase{ID: "in-1", Subject: "Wasser im Keller", Body: "Nachricht", Status: "Vorschlag liegt vor", StatusTone: "ok", HasSuggestion: true, ProviderLabel: "Cloud (OpenRouter)", CategoryLabel: "Reparatur/Mangel", Priority: "Hoch", Confidence: 95, HouseLabel: "Haus A", Unit: "Top 1", AssigneeLabel: "Vera", Due: "morgen", Reply: "Wir kümmern uns.", UnfilledLabels: []string{"Liegenschaft", "Frist"}, Created: "Eingegangen", Actions: []string{"Hausbetreuung informieren"}, Position: 2, Total: 37},
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
	if err := InboxPage(organisationPortalFixture(PortalPageData{Organisation: VerwaltungShell{OrganisationName: "Musterstadt", Active: "inbox"}}), data).Render(t.Context(), &out); err != nil {
		t.Fatal(err)
	}
	body := out.String()
	for _, want := range []string{"class=\"queue\"", "class=\"case\"", "class=\"conf\"", "class=\"bar\"", "2 von 37", "Telefonnotiz", "Übernehmen &amp; weiter", "J/K Wechseln", "Vorschlag liegt vor", "Heute automatisch erledigt · 1", "Zielbetrieb lokal im Büro", "Platzhalter ohne Wert: Liegenschaft, Frist — bitte prüfen", "htmx.min.js", "/assets/inbox.js?v="} {
		if !strings.Contains(body, want) {
			t.Fatalf("render missing %q", want)
		}
	}
	if strings.Contains(body, "class=\"assign\"") {
		t.Fatal("queue must not contain inline assignment controls")
	}
}

func TestInboxScriptsStayInDocumentHead(t *testing.T) {
	for _, page := range []string{"queue", "case"} {
		t.Run(page, func(t *testing.T) {
			component := InboxPage(organisationPortalFixture(PortalPageData{Organisation: VerwaltungShell{}}), InboxData{})
			if page == "case" {
				component = InboxCasePage(organisationPortalFixture(PortalPageData{Organisation: VerwaltungShell{}}), InboxData{FullPage: true, Selected: &InboxCase{ID: "in-1"}})
			}
			body := renderComponent(t, component)
			head, _, ok := strings.Cut(body, "</head>")
			if !ok {
				t.Fatal("document head missing")
			}
			for _, asset := range []string{"htmx/2.0.10/htmx.min.js", "inbox.js"} {
				if strings.Count(body, "/assets/"+asset+"?v=") != 1 || !strings.Contains(head, "/assets/"+asset+"?v=") {
					t.Fatalf("%s must load once through PortalDocument", asset)
				}
				if _, err := Assets.ReadFile("assets/" + asset); err != nil {
					t.Fatal(err)
				}
			}
			for _, script := range regexp.MustCompile(`<script([^>]*)>(.*?)</script>`).FindAllStringSubmatch(body, -1) {
				if !strings.Contains(script[1], `src="/assets/`) || strings.TrimSpace(script[2]) != "" {
					t.Fatal("inbox must use same-origin assets without inline scripts")
				}
			}
		})
	}
}

func TestInboxQuickFilterReflectsPreservedQueue(t *testing.T) {
	for _, test := range []struct{ query, active string }{
		{"?house=haus-a&sort=age", "all"},
		{"?house=haus-a&status=open&sort=age", "open"},
		{"?assignee=vera&status=proposed", "proposed"},
		{"?source=email&unassigned=1", "unassigned"},
		{"?status=approved", ""},
	} {
		for _, filter := range []string{"all", "open", "proposed", "unassigned"} {
			if got := inboxFilterActive(test.query, filter); got != (filter == test.active) {
				t.Errorf("query %q filter %q active=%v", test.query, filter, got)
			}
		}
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
		{state: "status_error", want: "Vorschlagsstatus konnte nicht geladen werden"},
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
	if !strings.Contains(body, "Liegenschaft zuordnen") || strings.Contains(body, "Vorschlag anfordern") || strings.Contains(body, "Einordnung") {
		t.Fatalf("unexpected unassigned case controls: %s", body)
	}
	if strings.Contains(body, "von 0") || strings.Contains(body, "0 von") {
		t.Fatalf("pager must stay hidden when the case is outside the queue: %s", body)
	}
}

func TestHausv618SuggestionPollingUsesAbsolutePartialURL(t *testing.T) {
	data := InboxData{Selected: &InboxCase{
		ID:              "in-0350",
		SuggestionState: "running",
		QueueQuery:      "?status=open",
	}}
	body := renderComponent(t, InboxPage(organisationPortalFixture(PortalPageData{Organisation: VerwaltungShell{}}), data))
	if !strings.Contains(body, `id="vorschlag" hx-get="/app/verwaltung/posteingang/in-0350/vorschlag?status=open" hx-select="#vorschlag"`) {
		t.Fatalf("running suggestion polling attributes missing: %s", body)
	}
	for _, match := range regexp.MustCompile(`hx-get="([^"]+)"`).FindAllStringSubmatch(body, -1) {
		if !strings.HasPrefix(match[1], "/") {
			t.Fatalf("relative hx-get URL %q", match[1])
		}
	}
}

func TestInboxPass2ResponsiveHistoryAndLiveRegionMarkup(t *testing.T) {
	item := InboxCase{
		ID: "in-1", HasSuggestion: true, Editing: true, CategoryLabel: "Betriebskosten/\u200bVorschreibung",
		Categories:    []InboxOption{{Value: "betriebskosten", Label: "Betriebskosten/\u200bVorschreibung", Selected: true}},
		AssigneeLabel: "Noch niemand", Created: "Eingegangen 03.09. · 11:00", Handling: "Vera · 03.09. · 12:10",
		Model: "triage-1", ProviderLabel: "Cloud (OpenRouter)", PromptHash: "12345678",
	}
	body := renderComponent(t, InboxContent(InboxData{Selected: &item}))
	for _, want := range []string{
		"overflow-wrap:break-word", "filter-chips{display:flex;flex-wrap:wrap", ".action-bar{position:static;bottom:auto",
		".inbox-page:not(.full) .empty-case{display:none}", "Betriebskosten/\u200bVorschreibung", ">Noch niemand</option>",
		"Technische Details", "Modell: triage-1", "Anbieter: Cloud (OpenRouter)", "Prompt: 12345678",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("pass-2 inbox render missing %q", want)
		}
	}
	historyStart := strings.Index(body, `<div class="history">`)
	if historyStart < 0 {
		t.Fatalf("history structure missing: %s", body)
	}
	historyDetails := strings.Index(body[historyStart:], `<details class="history-technical">`)
	if historyDetails < 0 {
		t.Fatalf("history technical details missing: %s", body)
	}
	plainHistory := body[historyStart : historyStart+historyDetails]
	if !strings.Contains(plainHistory, "Vorschlag erstellt") || strings.Contains(plainHistory, "triage-1") || strings.Contains(plainHistory, "12345678") {
		t.Fatalf("technical metadata leaked into plain history: %s", plainHistory)
	}

	running := renderComponent(t, InboxSuggestionState(InboxCase{ID: "in-1", SuggestionState: "running", SuggestionTimeout: 45}))
	if strings.Contains(running, `aria-live=`) || strings.Contains(running, `role="status"`) {
		t.Fatalf("polling state must not be a live region: %s", running)
	}
	arrived := renderComponent(t, InboxSuggestionState(InboxCase{ID: "in-1", SuggestionState: "arrived", SuggestionFinished: "12:04"}))
	if strings.Count(arrived, `role="status"`) != 1 || strings.Contains(arrived, `hx-trigger=`) {
		t.Fatalf("arrival must be announced exactly once without polling: %s", arrived)
	}
}

func TestHausv615TextbausteinePhoneStatusAndActionShareOneRow(t *testing.T) {
	body := renderComponent(t, TextbausteinListPage(organisationPortalFixture(PortalPageData{Organisation: VerwaltungShell{OrganisationName: "Musterstadt"}}), TextbausteinListData{
		Count: 1,
		Groups: []TextbausteinGroup{{Label: "Betriebskosten", Items: []TextbausteinRow{{
			Key: "betriebskosten-pruefung", Title: "Betriebskosten prüfen", Status: "Aktiv", Active: true, Updated: "03.09.2026", EditURL: "/eins",
		}}}},
	}))
	for _, want := range []string{
		".textbausteine-table tr{row-gap:6px;padding:var(--space-3)}",
		".textbausteine-status-cell{grid-column:1;grid-row:4}",
		".textbausteine-action-cell{grid-column:2;grid-row:4",
		"Geändert: 03.09.2026",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("phone textbaustein render missing %q", want)
		}
	}
}

func TestVerwaltungSettingsPageRendersTrustLevels(t *testing.T) {
	data := VerwaltungSettingsData{Categories: []VerwaltungSettingsCategory{{Key: "reparatur", Label: "Reparatur/Mangel", Level: "auto"}}, Threshold: 90, AutoEnabled: true, ProviderLabel: "nicht konfiguriert"}
	var out bytes.Buffer
	if err := VerwaltungSettingsPage(organisationPortalFixture(PortalPageData{Organisation: VerwaltungShell{OrganisationName: "Musterstadt"}}), data).Render(t.Context(), &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Reparatur/Mangel", "Manuell", "Vorschlag", "Automatisch", "Schwellwert für Automatisch", "nicht konfiguriert"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("settings render missing %q", want)
		}
	}
}
