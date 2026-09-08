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
		`grid-template-columns:repeat(auto-fit,minmax(260px,1fr))`,
		`.issue-board-main :is(.portal-section-header,.portal-section-context,.portal-section-content){padding-inline:32px}`,
		`--context-bar-h:40px`, `/assets/issue-board.js?v=board-test`,
		`draggable="true" data-board-card data-issue-status="Neu"`,
		`data-board-status="Angenommen"`, `data-board-status="Termin vereinbart"`,
		`role="status" aria-live="polite" aria-atomic="true" data-board-feedback`,
		`aria-label="Karte ziehen: Tür prüfen"`, `touch-action:none`,
		`<summary>Verschieben nach …<span class="disclosure-chevron" aria-hidden="true">`,
		`method="post" action="/app/anliegen/workflow" data-board-move`,
		`name="id" value="board-706"`, `name="priority" value="Hoch"`,
		`name="assignee_email" value="manager@example.com"`,
		`for="board-target-board-706"`, `id="board-target-board-706" name="status" required`,
		`value="Neu" selected`, `value="In Bearbeitung"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("board contract missing %q", want)
		}
	}
	if got := strings.Count(html, `data-board-status=`); got != 7 {
		t.Errorf("status lanes = %d, want all 7", got)
	}
	if strings.Contains(html, `ondrop=`) || strings.Contains(html, `ondragstart=`) {
		t.Error("board must keep scripts in its CSP-safe asset")
	}
}

func TestIssueBoardKeepsDisplayLabelsSeparateFromStoredStatus(t *testing.T) {
	options := []view.SelectOption{{Value: "In Bearbeitung", Label: "In Arbeit"}}
	columns := issueBoardColumns(options, []view.IssueView{{ID: "working", Status: "In Bearbeitung"}})
	if len(columns) != 1 || columns[0].Status != "In Bearbeitung" || columns[0].StatusLabel != "In Arbeit" || len(columns[0].Issues) != 1 {
		t.Fatalf("display labels must not create a new status lane: %+v", columns)
	}
	body := renderComponent(t, IssueBoardBody(IssueBoardPageData{
		Filters: view.IssueBoardFilterView{StatusOptions: options},
		Issues:  []view.IssueView{{ID: "working", Status: "In Bearbeitung"}},
	}))
	if !strings.Contains(body, `data-board-status="In Bearbeitung"`) || !strings.Contains(body, `>In Arbeit</h2>`) {
		t.Error("lane must show the option label and submit the stored status")
	}
}
