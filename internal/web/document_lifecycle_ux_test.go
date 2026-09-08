package web

import (
	"strings"
	"testing"
)

func TestEBInterfaceLifecycleShowsCompactProgressAndTrustBoundary(t *testing.T) {
	// Both steps must retain their own state and the complete trust copy.
	initial := renderComponent(t, EbInterfaceImportPage(EbInterfaceImportPageData{}))
	preview := renderComponent(t, EbInterfaceImportPage(EbInterfaceImportPageData{EBInterfacePreview: &EbInterfaceImportPreviewView{CanStore: true}}))
	css, err := Assets.ReadFile("assets/legacy-routes.css")
	if err != nil {
		t.Fatal(err)
	}
	content := initial + preview + string(css)
	for _, html := range []string{initial, preview} {
		if strings.Count(html, `aria-current="step"`) != 1 {
			t.Fatal("invoice import needs exactly one current step")
		}
	}
	for _, want := range []string{
		`E-Rechnung ablegen`,
		`aria-current="step"`,
		`Lokal geprüft · nicht extern gesendet · nach 15 Minuten gelöscht.`,
		`class="button small ghost" href="/app/dokumente/rechnungen/import"`,
		`Nur Verwaltung · keine Buchung oder Zahlung.`,
		`data-simple-file`, `data-default-file-label="XML auswählen"`,
		`.invoice-import .flow-step small { display: none; }`,
	} {
		if !strings.Contains(content, want) {
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
