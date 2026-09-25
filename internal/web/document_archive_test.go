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
	if blocks[1].ManagerOnly || blocks[1].Date != "" {
		t.Fatalf("incomplete archive views must not invent a chip or date: %+v", blocks[1])
	}
}

func TestDocumentLibraryUsesCompactRows(t *testing.T) {
	data := DocumentsPageData{
		CanManageDocuments: true,
		HasDocuments:       true,
		HasAnyDocuments:    true,
		DocumentSections: []view.DocumentCategoryView{{
			Category: "Abrechnung",
			Documents: []view.DocumentView{
				{ID: "a", Title: "Jahresabrechnung 2025 · Top 1", Archived: true, ArchiveRunID: "run", ArchiveYear: 2025, ArchiveRevision: 1, Visibility: "Nur Verwaltung", UploadedDate: "02.03.2026", VersionLabel: "Version 1", Size: "20 KB", Filename: "a.pdf", FileKind: "PDF", CanPreview: true, PreviewURL: "/app/dokumente/a/preview", DownloadURL: "/app/dokumente/a/download"},
				{ID: "b", Title: "Jahresabrechnung 2025 · Top 2", Archived: true, ArchiveRunID: "run", ArchiveYear: 2025, ArchiveRevision: 1, Visibility: "Nur Verwaltung", UploadedDate: "02.03.2026", VersionLabel: "Version 1", Size: "20 KB", Filename: "b.pdf", FileKind: "PDF", CanPreview: true, PreviewURL: "/app/dokumente/b/preview", DownloadURL: "/app/dokumente/b/download"},
				{ID: "rules", Title: "Hausordnung", Visibility: "Alle Bewohner", HasUnit: true, UnitLabel: "Top 2", VersionLabel: "Version 1", UploadedDate: "01.09.2026", Size: "12 KB", Filename: "hausordnung.pdf", FileKind: "PDF", CanPreview: true, PreviewURL: "/app/dokumente/rules/preview", DownloadURL: "/app/dokumente/rules/download", ReplaceDialogID: "document-replace-rules"},
			},
		}},
	}
	html := renderComponent(t, DocumentLibrary(data))
	if !strings.Contains(html, `<details class="document-archive-group">`) || strings.Contains(html, `class="document-archive-group" open`) {
		t.Fatal("archive group must start collapsed")
	}
	for _, want := range []string{
		"Jahresabrechnung 2025 · Lauf 1",
		"2 Dokumente",
		"02.03.2026",
		"Nur Verwaltung",
		`class="document-title"`,
		`aria-label="Vorschau: Hausordnung"`,
		`aria-label="Herunterladen"`,
		">Mehr",
		"Neue Version hochladen",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("library missing %q", want)
		}
	}
	_, row, found := strings.Cut(html, `<article class="document-row" id="document-rules">`)
	row, _, closed := strings.Cut(row, "</article>")
	if !found || !closed || strings.Contains(row, `class="button primary"`) {
		t.Fatal("document row still carries a filled button")
	}
	if strings.Contains(html, ">Vorschau<") {
		t.Fatal("preview is still a separate filled action")
	}
	if strings.Contains(html, "@UnitLabel") || strings.Count(html, ">Einheit Top\u00a02</span>") != 2 {
		t.Fatal("document metadata and details must render the actual nonbreaking unit label")
	}
}
