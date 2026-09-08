package web

import (
	"os"
	"strings"
	"testing"
)

func TestParkingReusesUnchangedPortalIconHAUSV702(t *testing.T) {
	body := renderComponent(t, ParkingBlank(ParkingPageData{}))
	art := between(body, `<div class="parking-empty-art"`, `</div>`)
	// Pinned pre-batch PortalIcon("parking") geometry, including every path.
	const want = `<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M5 16h14"></path><path d="m7 16 1.5-5h7L17 16"></path><path d="M7 16v3M17 16v3"></path><path d="M7 19h1M16 19h1"></path></svg>`
	if !strings.Contains(art, want) || strings.Count(art, "<svg") != 1 {
		t.Fatal("parking artwork must render the complete, unchanged library icon")
	}
	source, err := os.ReadFile("parking.templ")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(between(string(source), `<div class="parking-empty-art"`, `</div>`), `@PortalIcon("parking")`) {
		t.Fatal("parking must reference the library instead of copying paths")
	}
}

func TestReusedLucideArtworkIsUnchangedHAUSV702(t *testing.T) {
	for _, name := range []string{"download", "wrench", "message-square", "lightbulb", "ellipsis", "house", "door-open", "lock-keyhole", "pencil", "building-2", "map-pin"} {
		asset, err := Assets.ReadFile("assets/icons/lucide/" + name + ".svg")
		if err != nil {
			t.Fatal(err)
		}
		if got := renderComponent(t, PortalAssetIcon(name)); got != `<span aria-hidden="true" style="display:contents">`+string(asset)+`</span>` {
			t.Errorf("%s: render must preserve the checked-in artwork byte-for-byte", name)
		}
	}
	if portalAssetIconSVG("../../app.js") != "" || portalAssetIconSVG("missing-icon") != "" {
		t.Fatal("only checked-in Lucide names may supply raw markup")
	}
}
