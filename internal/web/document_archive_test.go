package web

import (
	"bytes"
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/view"
)

func TestArchivedDocumentRowHasBadgeAndNoReplacementControls(t *testing.T) {
	document := view.DocumentViewFrom(store.DocumentRecord{ID: "archived", Title: "Jahresabrechnung 2025", Visibility: store.DocumentVisibilityManagerOnly,
		AnnualStatementArchive: &store.AnnualStatementArchiveMetadata{PartyID: "Anna <owner@example.com>"}})
	var body bytes.Buffer
	if err := DocumentRow(document, true).Render(t.Context(), &body); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Archiviert · unveränderlich", "Anna &lt;owner@example.com&gt;", "/app/dokumente/archived/download"} {
		if !strings.Contains(body.String(), want) {
			t.Errorf("missing %q", want)
		}
	}
	for _, forbidden := range []string{"Neue Version", "document-replace-", "/app/dokumente/replace"} {
		if strings.Contains(body.String(), forbidden) {
			t.Errorf("archive exposes %q", forbidden)
		}
	}
}

func TestDocumentBlocksGroupOneArchiveRunCollapsed(t *testing.T) {
	docs := []view.DocumentView{
		{ID: "summary", Title: "Jahresabrechnung 2025"},
		{ID: "a", Archived: true, ArchiveRunID: "run", ArchiveYear: 2025, ArchiveRevision: 1, Title: "Jahresabrechnung 2025 · Top 1 · Lauf 1"},
		{ID: "b", Archived: true, ArchiveRunID: "run", ArchiveYear: 2025, ArchiveRevision: 1, Title: "Jahresabrechnung 2025 · Top 2 · Lauf 1"},
		{ID: "rules", Title: "Hausordnung"},
	}
	blocks := documentBlocks(docs)
	if len(blocks) != 3 || blocks[0].Grouped || !blocks[1].Grouped || blocks[1].Count != 2 || blocks[1].Label != "Jahresabrechnung 2025 · Lauf 1" || blocks[2].Grouped {
		t.Fatalf("blocks = %+v", blocks)
	}
	if got := documentArchiveCount(31); got != "31 Dokumente" {
		t.Fatalf("count = %q", got)
	}
	html := renderComponent(t, DocumentRow(blocks[1].Documents[0], true))
	if !strings.Contains(html, `id="document-a"`) || !strings.Contains(html, "Archiviert · unveränderlich") {
		t.Fatalf("grouped row lost the archive article:\n%s", html)
	}
}
