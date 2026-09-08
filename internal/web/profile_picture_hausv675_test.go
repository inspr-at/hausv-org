package web

import (
	"regexp"
	"strings"
	"testing"
)

const testAvatarURL = "/profilbild/01ARZ3NDEKTSV4RRFFQ69G5FAV"

func profilePicturePortal(avatarURL string) PortalPageData {
	return PortalPageData{
		Title:               "Hausüberblick",
		TenantSlug:          "demo",
		HouseName:           "WEG Portal",
		Address:             "Musterweg 1",
		DisplayName:         "Vera Verwalter",
		Initials:            "VV",
		Role:                "Verwalter",
		CanUseResidentAreas: true,
		Shell:               PortalShellData{Ready: true, RoleLabel: "Verwalter", AvatarURL: avatarURL},
	}
}

// avatarTiles returns the inner HTML of every element carrying class "avatar",
// which is the tile the sidebar and the mobile header share.
func avatarTiles(t *testing.T, html string) []string {
	t.Helper()
	pattern := regexp.MustCompile(`(?s)<(?:span|a) class="avatar"[^>]*>(.*?)</(?:span|a)>`)
	matches := pattern.FindAllStringSubmatch(html, -1)
	tiles := make([]string, 0, len(matches))
	for _, match := range matches {
		tiles = append(tiles, match[1])
	}
	if len(tiles) == 0 {
		t.Fatal("no account tile rendered — the probe is broken, not the shell")
	}
	return tiles
}

// HAUSV-675: the account tile carries the picture when there is one and the
// initials when there is not. Both halves are asserted, because a tile that
// always shows the picture and a tile that always shows the initials would each
// pass one half of this on their own.
func TestPortalAccountTileShowsThePictureOrTheInitials(t *testing.T) {
	withPicture := renderComponent(t, PortalPage(profilePicturePortal(testAvatarURL)))
	for _, tile := range avatarTiles(t, withPicture) {
		if !strings.Contains(tile, `src="`+testAvatarURL+`"`) {
			t.Fatalf("tile without the picture: %s", tile)
		}
		if !strings.Contains(tile, `class="avatar-image"`) {
			t.Fatalf("tile without the avatar-image class: %s", tile)
		}
		if !strings.Contains(tile, `alt=""`) {
			t.Fatalf("the picture is decorative and must carry an empty alt: %s", tile)
		}
		if strings.Contains(tile, "VV") {
			t.Fatalf("the initials must give way to the picture: %s", tile)
		}
	}

	withoutPicture := renderComponent(t, PortalPage(profilePicturePortal("")))
	for _, tile := range avatarTiles(t, withoutPicture) {
		if strings.Contains(tile, "<img") {
			t.Fatalf("a person without a picture must keep the initials tile: %s", tile)
		}
		if !strings.Contains(tile, "VV") {
			t.Fatalf("tile without the initials: %s", tile)
		}
	}
	if strings.Contains(withoutPicture, "/profilbild/") {
		t.Fatal("a person without a picture must not get a picture URL anywhere on the page")
	}
}

// HAUSV-675: the Verwaltung shell has its own sidebar and its own mobile header.
// They were the two places most likely to be forgotten.
func TestVerwaltungAccountTileShowsThePictureOrTheInitials(t *testing.T) {
	shell := organisationPortalFixture(PortalPageData{Organisation: VerwaltungShell{OrganisationName: "Hausverwaltung Muster", RoleLabel: "Verwalter", Active: "portfolio"}, DisplayName: "Vera Verwalter", Initials: "VV"})

	withoutPicture := renderComponent(t, PortalVerwaltungPage(shell, "Portfolio", PortalSectionTitle("Portfolio")))
	tiles := avatarTiles(t, withoutPicture)
	if len(tiles) < 2 {
		t.Fatalf("expected the sidebar and the mobile header tile, got %d", len(tiles))
	}
	for _, tile := range tiles {
		if !strings.Contains(tile, "VV") || strings.Contains(tile, "<img") {
			t.Fatalf("tile without the initials: %s", tile)
		}
	}

	shell.Shell.AvatarURL = testAvatarURL
	shell.Shell.Context.AvatarURL = testAvatarURL
	withPicture := renderComponent(t, PortalVerwaltungPage(shell, "Portfolio", PortalSectionTitle("Portfolio")))
	for _, tile := range avatarTiles(t, withPicture) {
		if !strings.Contains(tile, `src="`+testAvatarURL+`"`) {
			t.Fatalf("tile without the picture: %s", tile)
		}
	}
}

// HAUSV-675: the settings card is where the picture is set, so it has to say
// which of the two states it is in — and offer removal only in one of them.
func TestSettingsIdentityCardOffersTheProfilePictureControls(t *testing.T) {
	empty := renderComponent(t, SettingsHubPage(SettingsHubPageData{
		Portal:              profilePicturePortal(""),
		Email:               "vera@example.com",
		SettingsDisplayName: "Vera Verwalter",
	}))
	if !strings.Contains(empty, "Profilbild wählen") {
		t.Fatal("the card must offer choosing a picture")
	}
	if strings.Contains(empty, "Profilbild entfernen") {
		t.Fatal("there is nothing to remove without a picture")
	}
	if !strings.Contains(empty, `action="/app/settings/profilbild"`) {
		t.Fatal("the upload form must post to the picture route")
	}
	if !strings.Contains(empty, `enctype="multipart/form-data"`) {
		t.Fatal("the upload form must be a multipart form")
	}
	if !strings.Contains(empty, `accept="image/*"`) {
		t.Fatal("the file control must accept images")
	}
	if !strings.Contains(empty, "data-profile-picture-dialog") {
		t.Fatal("the crop dialog must be on the page")
	}
	if !strings.Contains(empty, "data-profile-picture-canvas") {
		t.Fatal("the crop dialog must carry its canvas")
	}
	if !strings.Contains(empty, "data-profile-picture-zoom") {
		t.Fatal("the crop dialog must carry its zoom control")
	}
	if strings.Contains(empty, `class="identity-avatar-image"`) {
		t.Fatal("an empty card must not render a picture")
	}
	if strings.Contains(empty, "/profilbild/") {
		t.Fatal("an empty card must not carry a picture URL")
	}

	filled := renderComponent(t, SettingsHubPage(SettingsHubPageData{
		Portal:              profilePicturePortal(testAvatarURL),
		Email:               "vera@example.com",
		SettingsDisplayName: "Vera Verwalter",
		ProfilePictureURL:   testAvatarURL,
		HasProfilePicture:   true,
		ProfilePictureMsg:   "Profilbild gespeichert.",
		ProfilePictureOK:    true,
	}))
	if !strings.Contains(filled, `class="identity-avatar-image" src="`+testAvatarURL+`"`) {
		t.Fatal("the card must render the stored picture")
	}
	if !strings.Contains(filled, "identity-avatar") {
		t.Fatal("the card tile must switch to the round picture frame")
	}
	if !strings.Contains(filled, "Profilbild ändern") {
		t.Fatal("the control must say it replaces the picture")
	}
	if !strings.Contains(filled, `action="/app/settings/profilbild/entfernen"`) {
		t.Fatal("the card must offer removal")
	}
	if !strings.Contains(filled, "Profilbild gespeichert.") {
		t.Fatal("the card must show the German status message")
	}
}

// HAUSV-675: the tile is 34px and round. A picture that is not cropped to fill
// it would either stretch or leave the gold ring showing through.
func TestAvatarTileClipsThePictureToTheCircle(t *testing.T) {
	html := renderComponent(t, PortalPage(profilePicturePortal(testAvatarURL)))
	rules := parseCSSRules(html)
	var tile, picture string
	for _, rule := range rules {
		if len(rule.context) != 0 {
			continue
		}
		for _, selector := range rule.selectors {
			switch strings.TrimSpace(selector) {
			case ".avatar":
				tile = rule.declarations
			case ".avatar-image":
				picture = rule.declarations
			}
		}
	}
	if tile == "" || picture == "" {
		t.Fatalf("avatar rules missing: tile=%q image=%q", tile, picture)
	}
	for _, want := range []string{"width:34px", "height:34px", "overflow:hidden", "border-radius:var(--radius-pill)"} {
		if !strings.Contains(tile, want) {
			t.Errorf(".avatar lost %q: %s", want, tile)
		}
	}
	for _, want := range []string{"object-fit:cover", "width:100%", "height:100%"} {
		if !strings.Contains(picture, want) {
			t.Errorf(".avatar-image lost %q: %s", want, picture)
		}
	}
}
