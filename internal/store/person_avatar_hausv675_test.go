package store

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
	"time"
)

func testPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 251), G: uint8(y % 241), B: 90, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func testJPEG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 200, G: uint8(x % 255), B: uint8(y % 255), A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	return buf.Bytes()
}

// HAUSV-675: what a person uploads is never what gets stored. Every accepted
// upload leaves this function as one shape — square, at most 512px, JPEG — and
// everything else is refused with a distinguishable error, because the handler
// turns each error into a different German sentence.
func TestEncodeProfilePictureNormalisesEveryAcceptedUpload(t *testing.T) {
	at := time.Date(2026, 9, 7, 10, 0, 0, 0, time.UTC)

	for _, tc := range []struct {
		name string
		raw  []byte
	}{
		{"png", testPNG(t, 900, 900)},
		{"jpeg", testJPEG(t, 700, 700)},
		{"landscape jpeg is centre-cropped", testJPEG(t, 1200, 400)},
		{"portrait png is centre-cropped", testPNG(t, 300, 1000)},
		{"already small stays small", testPNG(t, 64, 64)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			avatar, err := EncodeProfilePicture(tc.raw, at)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			if avatar.ContentType != ProfilePictureContentType {
				t.Fatalf("content type = %q", avatar.ContentType)
			}
			if avatar.ByteSize != int64(len(avatar.Image)) || avatar.ByteSize == 0 {
				t.Fatalf("byte size %d does not match %d bytes", avatar.ByteSize, len(avatar.Image))
			}
			if len(avatar.SHA256) != 64 {
				t.Fatalf("sha256 = %q", avatar.SHA256)
			}
			if NormalizeAvatarID(avatar.AvatarID) == "" {
				t.Fatalf("avatar id %q is not a ULID", avatar.AvatarID)
			}
			if !avatar.UpdatedAt.Equal(at) {
				t.Fatalf("updated at = %s, want %s", avatar.UpdatedAt, at)
			}
			// The stored bytes are a JPEG whatever went in, which is what strips
			// the metadata of the original file.
			decoded, format, err := image.Decode(bytes.NewReader(avatar.Image))
			if err != nil {
				t.Fatalf("decode stored image: %v", err)
			}
			if format != "jpeg" {
				t.Fatalf("stored format = %q, want jpeg", format)
			}
			bounds := decoded.Bounds()
			if bounds.Dx() != bounds.Dy() {
				t.Fatalf("stored image is %dx%d, want a square", bounds.Dx(), bounds.Dy())
			}
			if bounds.Dx() > ProfilePictureDimension {
				t.Fatalf("stored image is %dpx, want at most %d", bounds.Dx(), ProfilePictureDimension)
			}
		})
	}

	// Two uploads of the same bytes get different URLs: the id is minted per
	// upload so a replaced picture can never be served from a stale cache.
	first, err := EncodeProfilePicture(testPNG(t, 100, 100), at)
	if err != nil {
		t.Fatalf("encode first: %v", err)
	}
	second, err := EncodeProfilePicture(testPNG(t, 100, 100), at)
	if err != nil {
		t.Fatalf("encode second: %v", err)
	}
	if first.AvatarID == second.AvatarID {
		t.Fatal("two uploads must not share an avatar id")
	}
	if first.SHA256 != second.SHA256 {
		t.Fatal("identical input must hash identically")
	}
}

func TestEncodeProfilePictureRefusesWhatItCannotServe(t *testing.T) {
	// A WebP header. The module carries no WebP decoder, so the server says so
	// rather than storing something it cannot re-encode. The browser converts to
	// JPEG on the crop canvas, so this path is only reached without JavaScript.
	webp := append([]byte("RIFF\x00\x00\x00\x00WEBPVP8 "), bytes.Repeat([]byte{0}, 64)...)

	for _, tc := range []struct {
		name string
		raw  []byte
		want error
	}{
		{"empty", nil, ErrProfilePictureUnreadable},
		{"plain text", []byte(strings.Repeat("kein Bild ", 40)), ErrProfilePictureType},
		{"pdf", append([]byte("%PDF-1.7\n"), bytes.Repeat([]byte("x"), 600)...), ErrProfilePictureType},
		{"webp", webp, ErrProfilePictureType},
		{"png header without pixels", []byte("\x89PNG\r\n\x1a\n" + strings.Repeat("x", 100)), ErrProfilePictureUnreadable},
		{"oversized file", bytes.Repeat([]byte{0xff, 0xd8, 0xff}, MaxProfilePictureBytes), ErrProfilePictureTooLarge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := EncodeProfilePicture(tc.raw, time.Now()); err != tc.want {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

// HAUSV-675: the two backends must behave identically, because the server tests
// run on the memory one and production runs on the SQL one.
func TestPersonAvatarStorageParity(t *testing.T) {
	backends := map[string]func(t *testing.T) PersonAvatarStorage{
		"memory": func(t *testing.T) PersonAvatarStorage { return NewMemoryPersonAvatarStore() },
		"sql": func(t *testing.T) PersonAvatarStorage {
			database, lanes := testLanes(t)
			t.Cleanup(func() { database.Close() })
			return NewSQLPersonAvatarStore(lanes)
		},
	}

	for name, build := range backends {
		t.Run(name, func(t *testing.T) {
			s := build(t)
			if _, ok := s.Avatar("nobody@example.com"); ok {
				t.Fatal("unknown email must not be found")
			}
			if _, _, ok := s.AvatarByID("01JZZZZZZZZZZZZZZZZZZZZZZZ"); ok {
				t.Fatal("unknown avatar id must not be found")
			}
			if _, _, ok := s.AvatarByID("../../etc/passwd"); ok {
				t.Fatal("a crafted path segment must not resolve")
			}
			if err := s.SetAvatar("", PersonAvatar{AvatarID: "X", Image: []byte{1}}); err == nil {
				t.Fatal("empty email must error")
			}

			avatar, err := EncodeProfilePicture(testPNG(t, 300, 300), time.Date(2026, 9, 7, 8, 0, 0, 0, time.UTC))
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			if err := s.SetAvatar("Person@Example.com", avatar); err != nil {
				t.Fatalf("set: %v", err)
			}
			got, ok := s.Avatar("person@example.com") // normalized lookup
			if !ok {
				t.Fatal("stored avatar must be found via the normalized email")
			}
			if got.AvatarID != avatar.AvatarID || got.SHA256 != avatar.SHA256 || !bytes.Equal(got.Image, avatar.Image) {
				t.Fatalf("round-trip mismatch: %+v", got)
			}
			if got.ByteSize != avatar.ByteSize || got.ContentType != ProfilePictureContentType {
				t.Fatalf("round-trip metadata mismatch: %+v", got)
			}
			if !got.UpdatedAt.Equal(avatar.UpdatedAt) {
				t.Fatalf("updated at = %s, want %s", got.UpdatedAt, avatar.UpdatedAt)
			}

			byID, owner, ok := s.AvatarByID(avatar.AvatarID)
			if !ok {
				t.Fatal("stored avatar must be addressable by its id")
			}
			if owner != "person@example.com" {
				t.Fatalf("owner = %q", owner)
			}
			if !bytes.Equal(byID.Image, avatar.Image) {
				t.Fatal("id lookup returned different bytes")
			}

			// Replacing retires the old id, so a link handed out earlier stops
			// resolving instead of serving a picture that was taken down.
			replacement, err := EncodeProfilePicture(testJPEG(t, 200, 200), time.Now())
			if err != nil {
				t.Fatalf("encode replacement: %v", err)
			}
			if err := s.SetAvatar("person@example.com", replacement); err != nil {
				t.Fatalf("replace: %v", err)
			}
			if _, _, ok := s.AvatarByID(avatar.AvatarID); ok {
				t.Fatal("the previous avatar id must stop resolving")
			}
			if current, _ := s.Avatar("person@example.com"); current.AvatarID != replacement.AvatarID {
				t.Fatalf("current avatar id = %q, want %q", current.AvatarID, replacement.AvatarID)
			}

			removed, err := s.DeleteAvatar("person@example.com")
			if err != nil || !removed {
				t.Fatalf("delete = %t, %v", removed, err)
			}
			if _, ok := s.Avatar("person@example.com"); ok {
				t.Fatal("deleted avatar must be gone")
			}
			if _, _, ok := s.AvatarByID(replacement.AvatarID); ok {
				t.Fatal("deleted avatar must stop resolving by id")
			}
			removed, err = s.DeleteAvatar("person@example.com")
			if err != nil || removed {
				t.Fatalf("second delete = %t, %v", removed, err)
			}
		})
	}
}
