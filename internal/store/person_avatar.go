package store

// Profilbild einer Person (HAUSV-675).
//
// Das Bild gehört zur Person, nicht zu einem Haus: dieselbe Entscheidung, die
// profile_overlays trägt. Deshalb ist der Schlüssel die normalisierte
// Login-E-Mail und die Tabelle hat keine tenant_id.
//
// Gespeichert wird ausschliesslich das serverseitig neu kodierte Bild. Das
// Neukodieren ist kein Nebeneffekt der Grössenanpassung, sondern der Zweck:
// es entfernt jede Metadatenspur (EXIF, GPS, Kameramodell) aus der Datei, die
// jemand hochgeladen hat, und es macht aus einer beliebigen Bilddatei genau ein
// Format, das ausgeliefert wird.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png" // Profilbilder dürfen als PNG hochgeladen werden.
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
	"github.com/inspr-at/hausv-org/internal/ulid"
)

const (
	// MaxProfilePictureBytes ist die Obergrenze für die hochgeladene Datei.
	// Der Browser schneidet vorher zu und lädt ein JPEG von wenigen hundert
	// Kilobyte hoch; die Grenze gilt dem Fall, in dem er das nicht getan hat.
	MaxProfilePictureBytes = 8 << 20
	// MaxProfilePictureFormBytes begrenzt den ganzen Request-Body, nicht nur
	// die Datei — dieselbe Paarung wie bei Dokumenten und Titelbildern.
	MaxProfilePictureFormBytes = MaxProfilePictureBytes + (1 << 20)
	// ProfilePictureDimension ist die Kantenlänge des gespeicherten Bildes.
	ProfilePictureDimension = 512
	// ProfilePictureQuality ist die JPEG-Qualität des gespeicherten Bildes.
	ProfilePictureQuality = 86
	// ProfilePictureContentType ist das einzige ausgelieferte Format.
	ProfilePictureContentType = "image/jpeg"
	// maxProfilePictureSourcePixels begrenzt die Fläche, die dekodiert wird.
	// Eine kleine Datei kann ein sehr grosses Bild beschreiben; ohne diese
	// Grenze entscheidet der Angreifer, wie viel Speicher der Server belegt.
	maxProfilePictureSourcePixels = 50 << 20
)

var (
	// ErrProfilePictureType meldet ein Format, das der Server nicht annimmt.
	ErrProfilePictureType = errors.New("store: unsupported profile picture type")
	// ErrProfilePictureTooLarge meldet eine zu grosse Datei.
	ErrProfilePictureTooLarge = errors.New("store: profile picture too large")
	// ErrProfilePictureUnreadable meldet Bytes, die kein lesbares Bild sind.
	ErrProfilePictureUnreadable = errors.New("store: profile picture unreadable")
)

// PersonAvatar ist ein gespeichertes Profilbild.
type PersonAvatar struct {
	// AvatarID ist der öffentliche, nicht erratbare Teil der Bild-URL. Er wird
	// bei jedem neuen Bild neu vergeben, damit eine ausgelieferte URL genau
	// einen Bildinhalt bezeichnet.
	AvatarID    string
	ContentType string
	Image       []byte
	ByteSize    int64
	SHA256      string
	UpdatedAt   time.Time
}

// PersonAvatarStorage ist die Schnittstelle, die der In-Memory-Store (Tests)
// und der SQL-Store (Betrieb) erfüllen.
type PersonAvatarStorage interface {
	// Avatar liefert das Bild einer Person.
	Avatar(email string) (PersonAvatar, bool)
	// AvatarByID liefert das Bild zu einer Bild-URL samt der E-Mail seiner
	// Besitzerin, damit der Aufrufer den Zugriff entscheiden kann.
	AvatarByID(avatarID string) (PersonAvatar, string, bool)
	// SetAvatar ersetzt das Bild einer Person.
	SetAvatar(email string, avatar PersonAvatar) error
	// DeleteAvatar entfernt das Bild einer Person und meldet, ob eines da war.
	DeleteAvatar(email string) (bool, error)
}

var (
	_ PersonAvatarStorage = (*MemoryPersonAvatarStore)(nil)
	_ PersonAvatarStorage = (*SQLPersonAvatarStore)(nil)
)

// EncodeProfilePicture prüft die hochgeladenen Bytes und macht daraus genau ein
// gespeichertes Bild: quadratisch beschnitten auf die Mitte, höchstens
// 512x512, JPEG, ohne Metadaten.
//
// Serverseitig angenommen werden JPEG und PNG. WebP fehlt bewusst: der
// Decoder dafür ist nicht im Modul, und der Browser lädt ohnehin immer ein
// JPEG hoch (er zeichnet den Zuschnitt auf ein Canvas). Ein Upload ohne
// Browser-Zuschnitt bekommt darum eine klare Absage statt eines stillen
// Fehlers.
func EncodeProfilePicture(raw []byte, at time.Time) (PersonAvatar, error) {
	if len(raw) == 0 {
		return PersonAvatar{}, ErrProfilePictureUnreadable
	}
	if len(raw) > MaxProfilePictureBytes {
		return PersonAvatar{}, ErrProfilePictureTooLarge
	}
	switch http.DetectContentType(raw) {
	case "image/jpeg", "image/png":
	default:
		return PersonAvatar{}, ErrProfilePictureType
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return PersonAvatar{}, ErrProfilePictureUnreadable
	}
	if config.Width < 1 || config.Height < 1 {
		return PersonAvatar{}, ErrProfilePictureUnreadable
	}
	if int64(config.Width)*int64(config.Height) > maxProfilePictureSourcePixels {
		return PersonAvatar{}, ErrProfilePictureTooLarge
	}
	decoded, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return PersonAvatar{}, ErrProfilePictureUnreadable
	}
	square := ResizeImageNearest(cropImageToSquare(decoded), ProfilePictureDimension)
	var out bytes.Buffer
	if err := jpeg.Encode(&out, square, &jpeg.Options{Quality: ProfilePictureQuality}); err != nil {
		return PersonAvatar{}, ErrProfilePictureUnreadable
	}
	avatarID, err := ulid.New()
	if err != nil {
		return PersonAvatar{}, fmt.Errorf("store: mint profile picture id: %w", err)
	}
	digest := sha256.Sum256(out.Bytes())
	if at.IsZero() {
		at = time.Now()
	}
	return PersonAvatar{
		AvatarID:    avatarID,
		ContentType: ProfilePictureContentType,
		Image:       out.Bytes(),
		ByteSize:    int64(out.Len()),
		SHA256:      hex.EncodeToString(digest[:]),
		UpdatedAt:   at.UTC(),
	}, nil
}

// cropImageToSquare schneidet das Bild mittig auf ein Quadrat. Der Browser
// liefert bereits ein Quadrat; das hier ist die Absicherung für alles, was
// nicht durch den Zuschnitt-Dialog gegangen ist, damit ein Porträt im runden
// Rahmen nicht verzerrt statt beschnitten wird.
func cropImageToSquare(src image.Image) image.Image {
	bounds := src.Bounds()
	side := bounds.Dx()
	if bounds.Dy() < side {
		side = bounds.Dy()
	}
	if side < 1 || (bounds.Dx() == side && bounds.Dy() == side) {
		return src
	}
	rect := image.Rect(0, 0, side, side).Add(image.Pt(
		bounds.Min.X+(bounds.Dx()-side)/2,
		bounds.Min.Y+(bounds.Dy()-side)/2,
	))
	if sub, ok := src.(interface {
		SubImage(image.Rectangle) image.Image
	}); ok {
		return sub.SubImage(rect)
	}
	return src
}

// NormalizeProfilePictureEmail is the one place the avatar key is spelled, so
// storage and lookup can never disagree about casing or whitespace.
func NormalizeProfilePictureEmail(email string) string { return textutil.Email(email) }

// NormalizeAvatarID trims a URL segment down to a plausible avatar id. It is
// deliberately strict: only a valid ULID can ever address a stored row, so a
// probe with a crafted segment never reaches the database.
func NormalizeAvatarID(raw string) string {
	raw = strings.ToUpper(strings.TrimSpace(raw))
	if !ulid.Valid(raw) {
		return ""
	}
	return raw
}

// MemoryPersonAvatarStore keeps avatars in memory. Used by the server tests,
// which build an app without a database.
type MemoryPersonAvatarStore struct {
	mu    sync.Mutex
	items map[string]PersonAvatar
}

func NewMemoryPersonAvatarStore() *MemoryPersonAvatarStore {
	return &MemoryPersonAvatarStore{items: map[string]PersonAvatar{}}
}

func (s *MemoryPersonAvatarStore) Avatar(email string) (PersonAvatar, bool) {
	if s == nil {
		return PersonAvatar{}, false
	}
	email = NormalizeProfilePictureEmail(email)
	if email == "" {
		return PersonAvatar{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	avatar, ok := s.items[email]
	return avatar, ok
}

func (s *MemoryPersonAvatarStore) AvatarByID(avatarID string) (PersonAvatar, string, bool) {
	if s == nil {
		return PersonAvatar{}, "", false
	}
	avatarID = NormalizeAvatarID(avatarID)
	if avatarID == "" {
		return PersonAvatar{}, "", false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for email, avatar := range s.items {
		if avatar.AvatarID == avatarID {
			return avatar, email, true
		}
	}
	return PersonAvatar{}, "", false
}

func (s *MemoryPersonAvatarStore) SetAvatar(email string, avatar PersonAvatar) error {
	if s == nil {
		return fmt.Errorf("store: profile picture store unavailable")
	}
	email = NormalizeProfilePictureEmail(email)
	if email == "" {
		return fmt.Errorf("store: invalid profile picture email")
	}
	if avatar.AvatarID == "" || len(avatar.Image) == 0 {
		return fmt.Errorf("store: incomplete profile picture")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[email] = avatar
	return nil
}

func (s *MemoryPersonAvatarStore) DeleteAvatar(email string) (bool, error) {
	if s == nil {
		return false, nil
	}
	email = NormalizeProfilePictureEmail(email)
	if email == "" {
		return false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[email]; !ok {
		return false, nil
	}
	delete(s.items, email)
	return true, nil
}
