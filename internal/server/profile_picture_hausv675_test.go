package server

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	storepkg "github.com/inspr-at/hausv-org/internal/store"
)

func profilePicturePNG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 255), G: 40, B: uint8(y % 255), A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

// profilePictureGet drives a GET through the full stack with a session for a
// chosen tenant. authedRequest pins "demo"; the cross-house case needs another.
func profilePictureGet(t *testing.T, a *app, email string, tenantSlug string, path string) *httptest.ResponseRecorder {
	t.Helper()
	token, _, err := a.sessions.Put(email, tenantSlug, authMethodEmail, time.Hour)
	if err != nil {
		t.Fatalf("put session: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://hausv.org"+path, nil)
	req.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	rr := httptest.NewRecorder()
	a.handler().ServeHTTP(rr, req)
	return rr
}

func profilePictureLocation(t *testing.T, res *httptest.ResponseRecorder) string {
	t.Helper()
	if res.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303; body = %s", res.Code, res.Body.String())
	}
	return res.Result().Header.Get("Location")
}

// HAUSV-675: the whole loop a person walks — upload, see it, replace it, remove
// it — plus the audit trail each step leaves.
func TestProfilePictureUploadReplaceAndRemove(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email: "verwalter@example.com", Role: roleManager,
		Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods(),
	})

	if got := a.profilePictureURL("verwalter@example.com"); got != "" {
		t.Fatalf("a person without a picture must have no URL, got %q", got)
	}

	upload := authedMultipartFileRequest(t, a, "verwalter@example.com", "/demo/app/settings/profilbild",
		nil, "profile_picture", "portrait.png", profilePicturePNG(t, 900, 640))
	if location := profilePictureLocation(t, upload); location != "/demo/app/settings?bild=gespeichert" {
		t.Fatalf("upload redirect = %q", location)
	}

	stored, ok := a.personAvatars.Avatar("verwalter@example.com")
	if !ok {
		t.Fatal("the upload must be stored")
	}
	if stored.ContentType != storepkg.ProfilePictureContentType {
		t.Fatalf("stored content type = %q, want JPEG — the server re-encodes", stored.ContentType)
	}
	decoded, format, err := image.Decode(bytes.NewReader(stored.Image))
	if err != nil {
		t.Fatalf("decode stored picture: %v", err)
	}
	if format != "jpeg" || decoded.Bounds().Dx() != decoded.Bounds().Dy() || decoded.Bounds().Dx() > 512 {
		t.Fatalf("stored picture is %s %dx%d", format, decoded.Bounds().Dx(), decoded.Bounds().Dy())
	}
	if want := "/profilbild/" + stored.AvatarID; a.profilePictureURL("verwalter@example.com") != want {
		t.Fatalf("picture URL = %q, want %q", a.profilePictureURL("verwalter@example.com"), want)
	}

	events := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionProfilePicture, Limit: 10})
	if len(events) != 1 || events[0].Summary != "Profilbild gesetzt" {
		t.Fatalf("audit after upload = %+v", events)
	}

	// A replacement retires the old URL, which is what makes the long cache
	// lifetime on the image route safe.
	firstID := stored.AvatarID
	replace := authedMultipartFileRequest(t, a, "verwalter@example.com", "/demo/app/settings/profilbild",
		nil, "profile_picture", "neu.png", profilePicturePNG(t, 300, 300))
	profilePictureLocation(t, replace)
	replaced, _ := a.personAvatars.Avatar("verwalter@example.com")
	if replaced.AvatarID == firstID {
		t.Fatal("a replacement must mint a new picture URL")
	}
	if _, _, ok := a.personAvatars.AvatarByID(firstID); ok {
		t.Fatal("the previous picture URL must stop resolving")
	}

	remove := authedFormRequest(t, a, "verwalter@example.com", "/demo/app/settings/profilbild/entfernen", url.Values{})
	if location := profilePictureLocation(t, remove); location != "/demo/app/settings?bild=entfernt" {
		t.Fatalf("remove redirect = %q", location)
	}
	if _, ok := a.personAvatars.Avatar("verwalter@example.com"); ok {
		t.Fatal("the picture must be gone after removing it")
	}
	if a.profilePictureURL("verwalter@example.com") != "" {
		t.Fatal("a removed picture must leave no URL behind")
	}
	events = a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionProfilePicture, Limit: 10})
	if len(events) != 3 {
		t.Fatalf("audit trail = %d events, want 3 (set, replace, remove)", len(events))
	}
	if events[0].Summary != "Profilbild entfernt" {
		t.Fatalf("newest audit event = %q", events[0].Summary)
	}
}

// HAUSV-675: every refusal has its own German sentence on the settings page, so
// the status codes the handler redirects with must stay distinguishable.
func TestProfilePictureUploadRefusesWhatItCannotStore(t *testing.T) {
	oversized := bytes.Repeat([]byte{0xff, 0xd8, 0xff, 0xe0}, storepkg.MaxProfilePictureBytes/4+16)

	for _, tc := range []struct {
		name     string
		filename string
		body     []byte
		want     string
	}{
		{"a text file", "notizen.txt", []byte(strings.Repeat("kein Bild ", 80)), "typ"},
		{"a PDF", "vertrag.pdf", append([]byte("%PDF-1.7\n"), bytes.Repeat([]byte("x"), 900)...), "typ"},
		{"a WebP the browser did not convert", "foto.webp", append([]byte("RIFF\x00\x00\x00\x00WEBPVP8 "), bytes.Repeat([]byte{0}, 200)...), "typ"},
		{"a file over the size cap", "riesig.jpg", oversized, "zu-gross"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := newTestPortalApp(t, userProfile{
				Email: "verwalter@example.com", Role: roleManager,
				Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods(),
			})
			res := authedMultipartFileRequest(t, a, "verwalter@example.com", "/demo/app/settings/profilbild",
				nil, "profile_picture", tc.filename, tc.body)
			if location := profilePictureLocation(t, res); location != "/demo/app/settings?bild="+tc.want {
				t.Fatalf("redirect = %q, want status %q", location, tc.want)
			}
			if _, ok := a.personAvatars.Avatar("verwalter@example.com"); ok {
				t.Fatal("a refused upload must not be stored")
			}
			if message, okFlag := profilePictureMessage(tc.want); message == "" || okFlag {
				t.Fatalf("status %q has no German failure message", tc.want)
			}
		})
	}

	t.Run("no file at all", func(t *testing.T) {
		a := newTestPortalApp(t, userProfile{
			Email: "verwalter@example.com", Role: roleManager,
			Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods(),
		})
		res := authedMultipartFilesRequest(t, a, "verwalter@example.com", "/demo/app/settings/profilbild", nil, nil)
		if location := profilePictureLocation(t, res); location != "/demo/app/settings?bild=fehlt" {
			t.Fatalf("redirect = %q", location)
		}
	})
}

// HAUSV-675: who may fetch a picture. The id is an unguessable ULID, and on top
// of that the route is closed to anyone who is not signed in and does not share
// a house with the owner.
func TestProfilePictureIsServedOnlyToTheHouseholdOfItsOwner(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email: "verwalter@example.com", Role: roleManager,
		Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods(),
	})
	addTestTenant(a, tenantConfig{Slug: "haus-b", Name: "Haus B", Address: "Bahnweg 2"})
	a.profiles["nachbar@example.com"] = userProfile{
		Email: "nachbar@example.com", Role: roleResident,
		Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods(),
	}
	a.profiles["fremd@example.com"] = userProfile{
		Email: "fremd@example.com", Role: roleResident,
		Tenants: []string{"haus-b"}, AuthMethods: defaultAuthMethods(),
	}

	upload := authedMultipartFileRequest(t, a, "verwalter@example.com", "/demo/app/settings/profilbild",
		nil, "profile_picture", "portrait.png", profilePicturePNG(t, 400, 400))
	profilePictureLocation(t, upload)
	stored, _ := a.personAvatars.Avatar("verwalter@example.com")
	path := "/demo/profilbild/" + stored.AvatarID

	owner := profilePictureGet(t, a, "verwalter@example.com", "demo", path)
	if owner.Code != http.StatusOK {
		t.Fatalf("owner status = %d, want 200", owner.Code)
	}
	if got := owner.Result().Header.Get("Content-Type"); got != storepkg.ProfilePictureContentType {
		t.Fatalf("content type = %q", got)
	}
	// Revalidation rather than a lifetime: a removed picture must stop being
	// visible to a browser that already has it (HAUSV-675).
	if got := owner.Result().Header.Get("Cache-Control"); got != "private, no-cache" {
		t.Fatalf("cache-control = %q — the picture must be revalidated, never held for a window", got)
	}
	if got := owner.Result().Header.Get("ETag"); got != `"`+stored.SHA256+`"` {
		t.Fatalf("etag = %q", got)
	}
	if got := owner.Result().Header.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("x-content-type-options = %q", got)
	}
	if !bytes.Equal(owner.Body.Bytes(), stored.Image) {
		t.Fatal("the served bytes must be the stored bytes")
	}

	housemate := profilePictureGet(t, a, "nachbar@example.com", "demo", path)
	if housemate.Code != http.StatusOK {
		t.Fatalf("housemate status = %d, want 200", housemate.Code)
	}

	// The control for the case below: the same person, the same session, the
	// same route — their OWN picture comes back. Without this the 404 that
	// follows could just as well mean the route never ran.
	own, err := storepkg.EncodeProfilePicture(profilePicturePNG(t, 200, 200), time.Now())
	if err != nil {
		t.Fatalf("encode stranger picture: %v", err)
	}
	if err := a.personAvatars.SetAvatar("fremd@example.com", own); err != nil {
		t.Fatalf("store stranger picture: %v", err)
	}
	ownFetch := profilePictureGet(t, a, "fremd@example.com", "haus-b", "/haus-b/profilbild/"+own.AvatarID)
	if ownFetch.Code != http.StatusOK {
		t.Fatalf("own picture status = %d, want 200", ownFetch.Code)
	}

	// Another house: the picture is not theirs to see, and the answer must not
	// even confirm that the id exists.
	stranger := profilePictureGet(t, a, "fremd@example.com", "haus-b", "/haus-b/profilbild/"+stored.AvatarID)
	if stranger.Code != http.StatusNotFound {
		t.Fatalf("stranger status = %d, want 404", stranger.Code)
	}
	if bytes.Contains(stranger.Body.Bytes(), stored.Image) {
		t.Fatal("a refused request must not carry the picture")
	}

	// The rule itself, stated once so a future change has to argue with it.
	for _, tc := range []struct {
		viewer, owner string
		want          bool
	}{
		{"verwalter@example.com", "verwalter@example.com", true},
		{"nachbar@example.com", "verwalter@example.com", true},
		{"fremd@example.com", "verwalter@example.com", false},
		{"verwalter@example.com", "fremd@example.com", false},
		{"", "verwalter@example.com", false},
		{"verwalter@example.com", "", false},
	} {
		if got := a.canViewProfilePicture(tc.viewer, tc.owner); got != tc.want {
			t.Errorf("canViewProfilePicture(%q, %q) = %t, want %t", tc.viewer, tc.owner, got, tc.want)
		}
	}

	anonymous := httptest.NewRecorder()
	a.handler().ServeHTTP(anonymous, httptest.NewRequest(http.MethodGet, "http://hausv.org"+path, nil))
	if anonymous.Code == http.StatusOK {
		t.Fatalf("an unauthenticated request must not be served, got %d", anonymous.Code)
	}

	missing := profilePictureGet(t, a, "verwalter@example.com", "demo", "/demo/profilbild/01ARZ3NDEKTSV4RRFFQ69G5FAV")
	if missing.Code != http.StatusNotFound {
		t.Fatalf("unknown id status = %d, want 404", missing.Code)
	}
	crafted := profilePictureGet(t, a, "verwalter@example.com", "demo", "/demo/profilbild/nicht-ulid")
	if crafted.Code != http.StatusNotFound {
		t.Fatalf("crafted id status = %d, want 404", crafted.Code)
	}
}

// HAUSV-675: the sidebar tile is assembled once. If the shell stopped carrying
// the URL, every page would silently fall back to the initials.
func TestProfilePictureReachesTheShellAndTheSettingsCard(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email: "verwalter@example.com", Role: roleManager, FirstName: "Vera", LastName: "Verwalter",
		Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods(),
	})

	before := authedRequest(t, a, "verwalter@example.com", "/demo/app/settings")
	if before.Code != http.StatusOK {
		t.Fatalf("settings status = %d", before.Code)
	}
	if strings.Contains(before.Body.String(), "/profilbild/") {
		t.Fatal("a person without a picture must not get a picture URL")
	}
	if !strings.Contains(before.Body.String(), "Profilbild wählen") {
		t.Fatal("the settings card must offer the upload control")
	}
	if strings.Contains(before.Body.String(), "Profilbild entfernen") {
		t.Fatal("there is nothing to remove yet")
	}

	upload := authedMultipartFileRequest(t, a, "verwalter@example.com", "/demo/app/settings/profilbild",
		nil, "profile_picture", "portrait.png", profilePicturePNG(t, 400, 400))
	profilePictureLocation(t, upload)
	stored, _ := a.personAvatars.Avatar("verwalter@example.com")

	for _, path := range []string{"/demo/app", "/demo/app/settings", "/demo/app/dokumente"} {
		res := authedRequest(t, a, "verwalter@example.com", path)
		if res.Code != http.StatusOK {
			t.Fatalf("%s status = %d", path, res.Code)
		}
		want := `src="/demo/profilbild/` + stored.AvatarID + `"`
		if !strings.Contains(res.Body.String(), want) {
			t.Fatalf("%s does not render the tenant-prefixed picture URL %s", path, want)
		}
	}

	after := authedRequest(t, a, "verwalter@example.com", "/demo/app/settings")
	if !strings.Contains(after.Body.String(), "Profilbild entfernen") {
		t.Fatal("the settings card must offer removal once a picture exists")
	}
	if !strings.Contains(after.Body.String(), "Profilbild ändern") {
		t.Fatal("the upload control must say what it does once a picture exists")
	}
}
