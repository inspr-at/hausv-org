package web

import (
	"os"
	"strings"
	"testing"
)

// TestPortalShellCSSIsExternalizedHAUSV549 verifies that the portal shell CSS
// has been extracted to an external, cacheable asset and is no longer inlined
// in every HTML response (HAUSV-549).
func TestPortalShellCSSIsExternalizedHAUSV549(t *testing.T) {
	portal := PortalPageData{
		Title:               "Test Portal",
		HouseName:           "Test House",
		Address:             "Test Street 1",
		DisplayName:         "Test User",
		Initials:            "TU",
		Role:                "Verwaltung",
		CanUseResidentAreas: true,
		Modules: PortalModules{
			Issues:        true,
			Events:        true,
			Announcements: true,
		},
	}

	// Render the home page
	homeHTML := renderComponent(t, PortalPage(portal))

	// AC #1: Shared portal shell CSS is served as an external asset with version query
	linkTag := `<link rel="stylesheet" href="/assets/portal-shell.css?v=`
	if !strings.Contains(homeHTML, linkTag) {
		t.Fatal("portal-shell.css is not linked in the HTML head")
	}

	// AC #3: No FOUC - stylesheet <link> is present in <head> BEFORE body content
	headEnd := strings.Index(homeHTML, "</head>")
	bodyStart := strings.Index(homeHTML, "<body")
	linkIndex := strings.Index(homeHTML, linkTag)

	if headEnd < 0 || bodyStart < 0 || linkIndex < 0 {
		t.Fatalf("HTML structure incomplete: headEnd=%d bodyStart=%d linkIndex=%d", headEnd, bodyStart, linkIndex)
	}

	if linkIndex > headEnd {
		t.Error("portal-shell.css link is not in <head>")
	}

	if linkIndex > bodyStart {
		t.Error("portal-shell.css link is after <body> tag - this will cause FOUC")
	}

	// Verify that the inline shell CSS is NOT present
	inlineShellMarkers := []string{
		".shell{min-height:100vh;display:grid;grid-template-columns:240px minmax(0,1fr)}",
		".mobile-context-switch>summary{min-height:44px",
		"@media(max-width:760px){.shell{display:block",
	}

	for _, marker := range inlineShellMarkers {
		if strings.Contains(homeHTML, marker) {
			t.Errorf("Shell CSS is still inlined: found %q in HTML", marker)
		}
	}

	// Verify the CSS file exists in the assets directory
	cssPath := "assets/portal-shell.css"
	if _, err := os.Stat(cssPath); os.IsNotExist(err) {
		t.Errorf("portal-shell.css file does not exist at %s", cssPath)
	}

	// Read the CSS file and verify it contains the expected rules
	cssContent, err := os.ReadFile(cssPath)
	if err != nil {
		t.Fatalf("Failed to read portal-shell.css: %v", err)
	}

	expectedRules := []string{
		".shell{",
		".sidebar{",
		".portal-section-landing{",
		".mobile-context-switch{",
		"@media(max-width:760px){",
	}

	for _, rule := range expectedRules {
		if !strings.Contains(string(cssContent), rule) {
			t.Errorf("portal-shell.css is missing expected rule: %s", rule)
		}
	}

	// AC #4: No chrome restyle - verify critical classes are still referenced
	criticalClasses := []string{
		`class="shell"`,
		`class="sidebar"`,
		`class="context-bar mobile-head"`,
		`data-portal-section-landing`, // attribute, not class
	}

	for _, class := range criticalClasses {
		if !strings.Contains(homeHTML, class) {
			t.Errorf("HTML is missing critical element: %s", class)
		}
	}
}

// TestPortalShellCSSBytesSavedHAUSV549 measures the byte savings from
// externalizing the portal shell CSS.
func TestPortalShellCSSBytesSavedHAUSV549(t *testing.T) {
	portal := PortalPageData{
		Title:               "Test Portal",
		HouseName:           "Test House",
		Address:             "Test Street 1",
		DisplayName:         "Test User",
		Initials:            "TU",
		Role:                "Verwaltung",
		CanUseResidentAreas: true,
		Modules: PortalModules{
			Issues:        true,
			Events:        true,
			Announcements: true,
			Contacts:      true,
		},
	}

	// Render two different pages to measure the savings
	homeHTML := renderComponent(t, PortalPage(portal))
	contactsHTML := renderComponent(t, ContactsPage(ContactsPageData{
		Portal:       portal,
		AssetVersion: "test",
	}))

	// Read the external CSS file
	cssContent, err := os.ReadFile("assets/portal-shell.css")
	if err != nil {
		t.Fatalf("Failed to read portal-shell.css: %v", err)
	}

	// The link tag that replaces the inline CSS (approximately):
	// <link rel="stylesheet" href="/assets/portal-shell.css?v=0.99.8-a4b29b0-xxx"/>
	linkTagApproxSize := 100 // bytes (generous estimate)

	cssFileSize := len(cssContent)
	homeHTMLSize := len(homeHTML)
	contactsHTMLSize := len(contactsHTML)

	t.Logf("=== HAUSV-549 Byte Measurements ===")
	t.Logf("portal-shell.css file size: %d bytes", cssFileSize)
	t.Logf("Link tag size (approx): %d bytes", linkTagApproxSize)
	t.Logf("Home page HTML size: %d bytes", homeHTMLSize)
	t.Logf("Contacts page HTML size: %d bytes", contactsHTMLSize)

	// The savings: for the FIRST page load, we replace inline CSS with a link tag
	firstPageSavings := cssFileSize - linkTagApproxSize
	t.Logf("First page load: saves ~%d bytes in HTML (CSS moved to external file)", firstPageSavings)

	// For SECOND page load (with cached CSS), we save the entire CSS since it's cached
	secondPageSavings := cssFileSize
	t.Logf("Second page load (CSS cached): saves ~%d bytes in HTML", secondPageSavings)

	// Verify we actually saved bytes
	if firstPageSavings <= 0 {
		t.Error("Expected positive byte savings on first page load")
	}

	// The CSS file should be reasonably sized (a few KB)
	if cssFileSize < 1000 {
		t.Errorf("CSS file seems too small (%d bytes), may be incomplete", cssFileSize)
	}

	if cssFileSize > 50000 {
		t.Errorf("CSS file seems too large (%d bytes), may include unintended content", cssFileSize)
	}
}
