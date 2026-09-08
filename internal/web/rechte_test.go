package web

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/authz"
)

func TestRechteMatrixKeepsEveryGrantAndLinkedRestriction(t *testing.T) {
	body := renderComponent(t, RechteContent())
	if strings.Count(body, `scope="row"`) != len(authz.RoleAreas) || strings.Count(body, `scope="col"`) != len(authz.RoleFamilies)+1 {
		t.Fatal("matrix must have one row per area and one column per role family")
	}
	for _, cell := range authz.RoleMatrix {
		marker := fmt.Sprintf(`<td data-family="%s" data-area="%s">`, cell.FamilyKey, cell.AreaKey)
		_, tail, ok := strings.Cut(body, marker)
		if !ok || strings.Count(body, marker) != 1 {
			t.Fatalf("matrix cell must appear exactly once: %s", marker)
		}
		content, _, _ := strings.Cut(tail, "</td>")
		if strings.Count(content, `class="rechte-chip"`) != len(cell.Grants) {
			t.Fatalf("grant count changed for %s/%s", cell.FamilyKey, cell.AreaKey)
		}
		for _, grant := range cell.Grants {
			if !strings.Contains(content, ">"+string(grant.Action)+"</span>") {
				t.Fatalf("lost grant %s for %s/%s", grant.Action, cell.FamilyKey, cell.AreaKey)
			}
		}
		if len(cell.Grants) == 0 && !strings.Contains(content, `aria-label="keine Aktion">—</span>`) {
			t.Fatalf("empty permission must stay explicit: %s/%s", cell.FamilyKey, cell.AreaKey)
		}
		if cell.Note != "" {
			link := regexp.MustCompile(`href="#(rechte-note-\d+)"`).FindStringSubmatch(content)
			if len(link) != 2 || !strings.Contains(html.UnescapeString(body), `<li id="`+link[1]+`">`+cell.Note+".</li>") {
				t.Fatalf("lost linked restriction for %s/%s", cell.FamilyKey, cell.AreaKey)
			}
		}
	}
}
