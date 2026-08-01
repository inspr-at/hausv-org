package web

import (
	"strings"
	"testing"
)

func TestDocumentLifecycleKeepsSecondaryActionsQuietAndDialogFooterStable(t *testing.T) {
	for _, want := range []string{
		`aria-label="Dokument hochladen"`,
		`class="button ghost" href="/app/dokumente/rechnungen/import"`,
		`class="doc-blank-side"`,
		`Optional: bestimmte Einheit`,
		`class="dialog-footer"><button class="button primary" type="submit">Hochladen`,
		`class="button small ghost" href="{{.DownloadURL}}"`,
		`.documents-screen .document-actions .primary { grid-column: auto; }`,
		`.documents-screen .document-icon { display: none; }`,
	} {
		if !strings.Contains(PageTemplates, want) {
			t.Fatalf("document lifecycle UX missing %q", want)
		}
	}
	if strings.Contains(PageTemplates, `#document-upload .dialog-body > button:last-child`) {
		t.Fatal("document upload submit must stay in the dedicated dialog footer")
	}
}

func TestEBInterfaceLifecycleShowsCompactProgressAndTrustBoundary(t *testing.T) {
	for _, want := range []string{
		`E-Rechnung ablegen`,
		`aria-current="step"`,
		`Lokal geprüft · nicht extern gesendet · nach 15 Minuten gelöscht.`,
		`class="button small ghost" href="/app/dokumente/rechnungen/import"`,
		`Nur Verwaltung · keine Buchung oder Zahlung.`,
		`data-simple-file data-default-file-label="XML auswählen"`,
		`.invoice-import .flow-step small { display: none; }`,
	} {
		if !strings.Contains(PageTemplates, want) {
			t.Fatalf("e-invoice lifecycle UX missing %q", want)
		}
	}
	script, err := Assets.ReadFile("assets/attachments.js")
	if err != nil {
		t.Fatalf("read attachment behavior: %v", err)
	}
	for _, want := range []string{`hasAttribute("data-simple-file")`, `files[0].name`, `classList.toggle("is-filled"`} {
		if !strings.Contains(string(script), want) {
			t.Fatalf("simple invoice file picker behavior missing %q", want)
		}
	}
}
