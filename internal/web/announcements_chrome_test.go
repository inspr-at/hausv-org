package web

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/inspr-at/hausv-org/internal/view"
)

// TestAnnouncementsPageRestoresRemovedCSS validates that PR #100's CSS rules
// that were inadvertently removed are now restored (HAUSV-0.98.10 fixes).
func TestAnnouncementsPageRestoresRemovedCSS(t *testing.T) {
	portal := PortalPageData{
		Title:     "Aushang Test",
		HouseName: "Test House",
	}
	data := AnnouncementsPageData{
		Portal:       portal,
		AssetVersion: "test-0.98.10",
	}

	html := renderComponent(t, AnnouncementsPage(data))

	// Restored page structure rules that were removed in PR #100
	requiredRules := []string{
		".page-main{",
		".page-head{",
		".eyebrow{",
		".lede{",
		".button.primary{",
		".button.ghost{",
		".button.small{",
		".flash{",
		".dialog{",
		".dialog::backdrop{",
		".dialog-head{",
		".dialog-close{",
		".dialog-body{",
	}

	for _, rule := range requiredRules {
		if !strings.Contains(html, rule) {
			t.Errorf("Announcements page is missing restored CSS rule: %s", rule)
		}
	}
}

// TestAnnouncementCategoryPillsHaveColors validates that category pills in the
// legend have the same background colors as their corresponding card ticks.
func TestAnnouncementCategoryPillsHaveColors(t *testing.T) {
	portal := PortalPageData{Title: "Aushang"}
	data := AnnouncementsPageData{
		Portal:       portal,
		AssetVersion: "test",
		Announcements: []view.AnnouncementView{
			{ID: "1", Title: "Info", Category: "Info", CategoryClass: "info"},
			{ID: "2", Title: "Termin", Category: "Termin", CategoryClass: "termin"},
			{ID: "3", Title: "Wartung", Category: "Wartung", CategoryClass: "wartung"},
			{ID: "4", Title: "Dringend", Category: "Dringend", CategoryClass: "dringend"},
		},
		HasAnyAnnouncements: true,
	}

	html := renderComponent(t, AnnouncementsPage(data))

	// Check that category pills have matching colors to card ticks
	categoryColors := map[string]string{
		"info":     "#c8993f",
		"termin":   "#8a7b3f",
		"wartung":  "#2f6b4a",
		"dringend": "#a8593c",
	}

	for category, color := range categoryColors {
		// Pills should have background color matching the tick
		pillRule := ".pill." + category + "{background:" + color
		if !strings.Contains(html, pillRule) {
			t.Errorf("Category pill .%s is missing its background color %s", category, color)
		}

		// Ticks should have the same color
		tickRule := ".announcement-card-tick." + category + "{background:" + color
		if !strings.Contains(html, tickRule) {
			t.Errorf("Card tick .%s is missing its background color %s", category, color)
		}
	}
}

// TestKategorienLegendIsVisible validates that the Kategorien legend sidebar
// is visible by having the open attribute on the details element.
func TestKategorienLegendIsVisible(t *testing.T) {
	portal := PortalPageData{Title: "Aushang"}
	data := AnnouncementsPageData{
		Portal:       portal,
		AssetVersion: "test",
	}

	html := renderComponent(t, AnnouncementsPage(data))

	// The legend must either have open attribute or not be a closed details
	if !strings.Contains(html, `class="aside-panel guide" open`) {
		t.Error("Kategorien legend details element is missing the 'open' attribute - legend will be hidden")
	}

	// Verify the legend content is present
	if !strings.Contains(html, "Was die Farben bedeuten") {
		t.Error("Kategorien legend is missing its title")
	}

	if !strings.Contains(html, `<ul class="legend">`) {
		t.Error("Kategorien legend <ul> element is missing")
	}

	// Check that all four category explanations are present
	categories := []string{"Dringend", "Wartung", "Termin", "Info"}
	for _, cat := range categories {
		if !strings.Contains(html, `<span class="pill `+strings.ToLower(cat)+`">`+cat+`</span>`) {
			t.Errorf("Legend is missing category: %s", cat)
		}
	}
}

// TestAnnouncementPageDoesNotBreakMobileChrome validates that the restored
// CSS rules don't interfere with mobile styling, particularly mobile-head.
func TestAnnouncementPageDoesNotBreakMobileChrome(t *testing.T) {
	portal := PortalPageData{Title: "Aushang", HouseName: "Test House"}
	data := AnnouncementsPageData{
		Portal:       portal,
		AssetVersion: "test",
	}

	html := renderComponent(t, AnnouncementsPage(data))

	// The mobile head styling from PortalShellStyles should not be overridden
	// by page-level .mobile-head rules. The doctrine says:
	// "do not restyle shared .mobile-head position/height"
	
	// Verify mobile-head is present from PortalMobileHeader
	if !strings.Contains(html, `class="mobile-head"`) {
		t.Error("Mobile header is missing")
	}

	// Verify no page-scoped .mobile-head rules that would break Energie smoke
	// The page should only have [data-templ-portal] scoped rules
	if strings.Contains(html, "[data-templ-portal] .mobile-head{") {
		t.Error("Announcements page added [data-templ-portal] .mobile-head rules that could break Energie mobile")
	}
}

// TestAnnouncementTruncationIsRuneSafe validates that German text with
// multi-byte characters is not cut mid-rune.
func TestAnnouncementTruncationIsRuneSafe(t *testing.T) {
	// textutil.Truncate is rune-safe, but let's verify the integration
	germanText := "Köln Düsseldorf München Überlingen"
	truncated := truncateText(germanText, 15)

	// Should not panic or produce invalid UTF-8
	if !strings.HasSuffix(truncated, "…") {
		t.Error("Truncated text should end with ellipsis")
	}

	// Verify it's valid UTF-8
	if !utf8.ValidString(truncated[:len(truncated)-len("…")]) {
		t.Error("Truncated text produced invalid UTF-8")
	}

	// Should be limited to roughly 15 runes (plus ellipsis)
	runes := []rune(truncated[:len(truncated)-len("…")])
	if len(runes) > 15 {
		t.Errorf("Truncated text has %d runes, expected <= 15", len(runes))
	}
}
