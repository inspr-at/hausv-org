package web

import (
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/view"
)

func TestIssueBoardWideLayoutAndAccessibleMoves(t *testing.T) {
	issue := view.IssueView{ID: "board-706", Title: "Tür prüfen", Status: "Neu", StatusClass: "status-open", Priority: "Hoch", AssigneeEmail: "manager@example.com"}
	html := renderComponent(t, IssueBoardPage(IssueBoardPageData{
		AssetVersion: "board-test", BoardAction: "/app/anliegen/board", TotalIssueCount: 1,
		Filters: view.IssueBoardFilterOptions(view.IssueBoardFilterView{}), Issues: []view.IssueView{issue},
	}))
	for _, want := range []string{
		`.portal-section-landing.issue-board-main{--portal-content-width:100%}`,
		`grid-template-columns:repeat(5,minmax(0,1fr))`,
		`body[data-authenticated-app] .issue-board-main:is(.portal-section-landing) :is(.portal-section-header,.portal-section-context,.portal-section-content){padding-inline:32px}`,
		`--context-bar-h:72px`, `/assets/issue-board.js?v=board-test`,
		`draggable="true" data-board-card data-issue-status="Neu"`,
		`data-board-status="Angenommen"`, `data-board-status="Termin vereinbart"`,
		`role="status" aria-live="polite" aria-atomic="true" data-board-feedback`,
		`aria-label="Karte ziehen: Tür prüfen"`, `touch-action:none`,
		`tabindex="0" role="button" aria-label="Tür prüfen · Neu"`,
		`data-board-panel`, `grid-template-columns:minmax(0,1fr) 360px`,
		`data-assignee="manager@example.com"`, `Hierher ziehen`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("board contract missing %q", want)
		}
	}
	if got := strings.Count(html, `data-board-status=`); got != 5 {
		t.Errorf("status lanes = %d, want 5 main lanes", got)
	}
	if strings.Contains(html, `ondrop=`) || strings.Contains(html, `ondragstart=`) {
		t.Error("board must keep scripts in its CSP-safe asset")
	}
}

func TestIssueBoardPanelKeepsWorkflowFieldsAndHistory(t *testing.T) {
	issue := view.IssueView{ID: "panel", Title: "Tür prüfen", Body: "Beschreibung nur im Panel", Status: "Neu", Priority: "Hoch", AssigneeEmail: "manager@example.com", HasAssignee: true}
	html := renderComponent(t, IssueBoardPanel(IssueBoardPanelData{Issue: issue, History: []IssueBoardHistoryView{{Action: "Neu → Angenommen", Actor: "Mara", Age: "vor 2 T."}}}))
	for _, want := range []string{`data-board-move`, `data-board-assign`, `name="status"`, `name="priority" value="Hoch"`, `name="assignee_email" value="manager@example.com"`, `name="redirect" value="/app/anliegen/board"`, `name="service_start"`, `name="service_end"`, `Neu → Angenommen`, `Mara`, `vor 2 T.`, `Beschreibung nur im Panel`, `href="/app/anliegen/board/panel"`} {
		if !strings.Contains(html, want) {
			t.Errorf("panel missing %q", want)
		}
	}
	card := renderComponent(t, IssueWorkCard(issue))
	for _, forbidden := range []string{"Beschreibung nur im Panel", "Nächster Schritt", "Verschieben nach", ">Bearbeiten<", "data-board-move"} {
		if strings.Contains(card, forbidden) {
			t.Errorf("compact card contains %q", forbidden)
		}
	}
}

func TestIssueBoardClosedStatusesRemainInCompletedLane(t *testing.T) {
	columns := issueBoardColumns(view.IssueBoardFilterOptions(view.IssueBoardFilterView{}).StatusOptions, []view.IssueView{{ID: "rejected", Status: "Abgelehnt"}, {ID: "duplicate", Status: "Duplikat"}})
	if len(columns) != 5 || len(columns[4].Issues) != 2 || columns[4].Issues[0].Status != "Abgelehnt" || columns[4].Issues[1].Status != "Duplikat" {
		t.Fatalf("closed statuses must remain intact in the completed lane: %+v", columns)
	}
}

func TestIssueBoardKeepsDisplayLabelsSeparateFromStoredStatus(t *testing.T) {
	options := []view.SelectOption{{Value: "In Bearbeitung", Label: "In Arbeit"}}
	columns := issueBoardColumns(options, []view.IssueView{{ID: "working", Status: "In Bearbeitung"}})
	if len(columns) != 1 || columns[0].Status != "In Bearbeitung" || columns[0].StatusLabel != "In Bearbeitung" || len(columns[0].Issues) != 1 {
		t.Fatalf("display labels must not create a new status lane: %+v", columns)
	}
	body := renderComponent(t, IssueBoardBody(IssueBoardPageData{
		Filters: view.IssueBoardFilterView{StatusOptions: options},
		Issues:  []view.IssueView{{ID: "working", Status: "In Bearbeitung"}},
	}))
	if !strings.Contains(body, `data-board-status="In Bearbeitung"`) || !strings.Contains(body, `>In Bearbeitung</h2>`) {
		t.Error("lane must show the full process name and submit the stored status")
	}
}

func TestIssueCategoryLabelKeepsKeysAndShowsUmlauts(t *testing.T) {
	for _, test := range []struct{ in, want string }{
		{"schluessel", "Schlüssel/Zutritt"},
		{"uebergabe", "Übergabe"},
		{"betriebskosten", "Betriebskosten/Vorschreibung"},
		{"Reparatur", "Reparatur"},
		{"", "Sonstiges"},
	} {
		if got := issueCategoryLabel(test.in); got != test.want {
			t.Errorf("issueCategoryLabel(%q) = %q, want %q", test.in, got, test.want)
		}
	}
}
