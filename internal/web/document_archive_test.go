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
