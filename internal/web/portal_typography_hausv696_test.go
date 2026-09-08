package web

import (
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func TestBothShellsPreloadOneVersionedFontHAUSV696(t *testing.T) {
	for _, page := range []templ.Component{
		PortalPage(PortalPageData{Title: "Portal"}),
		PortalVerwaltungPage(VerwaltungShell{}, "Verwaltung", templ.Raw("")),
		InboxPage(VerwaltungShell{}, InboxData{}),
		ParkingPage(ParkingPageData{AssetVersion: "test"}),
		IssuesPage(IssuesPageData{AssetVersion: "test"}),
		ContactsPage(ContactsPageData{AssetVersion: "test"}),
		DocumentsPage(DocumentsPageData{AssetVersion: "test"}),
		HandoversPage(HandoversPageData{AssetVersion: "test"}),
	} {
		body := renderComponent(t, page)
		head, _, _ := strings.Cut(body, "</head>")
		prefix := `<link rel="preload" href="`
		_, link, ok := strings.Cut(head, prefix)
		if !ok || strings.Count(head, prefix) != 1 {
			t.Fatal("each document must preload its font once in head")
		}
		fontURL, _, _ := strings.Cut(link, `"`)
		if !strings.HasPrefix(fontURL, "/assets/source-serif-4-semibold.woff2?v=") || !strings.Contains(link, `as="font" type="font/woff2" crossorigin="anonymous"`) {
			t.Fatalf("font preload attributes missing: %s", fontURL)
		}
		if !strings.Contains(head, `src:url("`+fontURL+`") format("woff2")`) || strings.Count(body, "@font-face") != 1 || !strings.Contains(head, "font-display:block") {
			t.Fatal("one font-face must reuse the preload URL with swap")
		}
	}
}
