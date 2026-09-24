package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestLegalHelpRenderingAndSources(t *testing.T) {
	a, _, _ := newInboxTestApp(t, roleAdmin)
	page := authedRequest(t, a, "vera@example.com", "/demo/app/hilfe")
	if page.Code != http.StatusOK {
		t.Fatal(page.Code)
	}
	for _, want := range []string{"Rechtliche Grundlagen und Entscheidungen", `id="recht-wirksamwerden"`, `id="recht-weg"`, `id="recht-mieweg"`, "Keine Rechtsberatung", "24.09.2026", "5 Ob 127/25z", "1 Ob 2/26i", "Bitterl", "§ 24b", `href="/demo/app/verwaltung/einstellungen"`, `https://www.wko.at/wirtschaftsrecht/wertsicherung-miet-und-pachtvertraege`, `https://www.ovi.at/aktuelles/detailansicht/mietpreisbremse-3-milg-im-nationalrat-beschlossen`} {
		if !strings.Contains(page.Body.String(), want) {
			t.Errorf("missing %q", want)
		}
	}
}
