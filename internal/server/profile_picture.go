package server

// Profilbild: hochladen, ersetzen, entfernen, ausliefern (HAUSV-675).
//
// Drei Entscheidungen, die hier festgehalten werden, weil sie sonst nur aus dem
// Code zu erraten wären:
//
//  1. Das Bild gehört zur PERSON, nicht zum Haus. Gespeichert wird es unter der
//     normalisierten Login-E-Mail, genau wie die selbst gepflegten Profildaten
//     in profile_overlays. Wer in drei Häusern ist, hat ein Bild, nicht drei.
//
//  2. Setzen und Entfernen betreffen ausschliesslich die eigene Person. Es gibt
//     keinen Weg, über den jemand das Bild einer anderen Person ändert — auch
//     nicht als Admin. Deshalb nimmt keiner der beiden POST-Handler eine
//     Ziel-E-Mail entgegen: die Zielperson ist immer die angemeldete.
//
//  3. Ausgeliefert wird unter einer nicht erratbaren ULID-URL, und zusätzlich
//     nur an angemeldete Personen, die mindestens ein Haus mit der Besitzerin
//     teilen. Die ULID allein wäre schon praktisch nicht zu erraten; die
//     Hausprüfung ist die zweite, überprüfbare Schranke, damit ein einmal
//     geteilter Link nicht zum dauerhaften Zugang für Fremde wird.

import (
	"bytes"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
)

const (
	maxProfilePictureBytes     = store.MaxProfilePictureBytes
	maxProfilePictureFormBytes = store.MaxProfilePictureFormBytes
	auditActionProfilePicture  = store.AuditActionProfilePicture
	profilePictureURLPrefix    = "/profilbild/"
)

// profilePictureURL is the src the shell and the settings card render. Empty
// when the person has no picture, which is exactly what the templates test.
func (a *app) profilePictureURL(email string) string {
	if a == nil || a.personAvatars == nil {
		return ""
	}
	avatar, ok := a.personAvatars.Avatar(email)
	if !ok || strings.TrimSpace(avatar.AvatarID) == "" {
		return ""
	}
	return profilePictureURLPrefix + avatar.AvatarID
}

// canViewProfilePicture is the access rule, in one place so the handler and its
// test talk about the same sentence: the owner always, everybody else only when
// they share at least one house with the owner.
func (a *app) canViewProfilePicture(viewerEmail string, ownerEmail string) bool {
	if a == nil {
		return false
	}
	viewerEmail = normalizeEmail(viewerEmail)
	ownerEmail = normalizeEmail(ownerEmail)
	if viewerEmail == "" || ownerEmail == "" {
		return false
	}
	if viewerEmail == ownerEmail {
		return true
	}
	owned := make(map[string]struct{})
	for _, slug := range a.ownTenantSlugs(ownerEmail) {
		owned[slug] = struct{}{}
	}
	for _, slug := range a.ownTenantSlugs(viewerEmail) {
		if _, ok := owned[slug]; ok {
			return true
		}
	}
	return false
}

func (a *app) updateProfilePicture(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if a.personAvatars == nil {
		a.redirectProfilePicture(w, r, "fehler")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxProfilePictureFormBytes)
	if err := r.ParseMultipartForm(maxProfilePictureBytes); err != nil {
		a.redirectProfilePicture(w, r, "zu-gross")
		return
	}
	header := profilePictureHeader(r)
	if header == nil {
		a.redirectProfilePicture(w, r, "fehlt")
		return
	}
	if header.Size > maxProfilePictureBytes {
		a.redirectProfilePicture(w, r, "zu-gross")
		return
	}
	file, err := header.Open()
	if err != nil {
		a.redirectProfilePicture(w, r, "fehler")
		return
	}
	raw, readErr := io.ReadAll(io.LimitReader(file, maxProfilePictureBytes+1))
	_ = file.Close()
	if readErr != nil {
		a.redirectProfilePicture(w, r, "fehler")
		return
	}
	if len(raw) > maxProfilePictureBytes {
		a.redirectProfilePicture(w, r, "zu-gross")
		return
	}
	avatar, err := store.EncodeProfilePicture(raw, time.Now())
	if err != nil {
		switch {
		case errors.Is(err, store.ErrProfilePictureTooLarge):
			a.redirectProfilePicture(w, r, "zu-gross")
		case errors.Is(err, store.ErrProfilePictureType), errors.Is(err, store.ErrProfilePictureUnreadable):
			a.redirectProfilePicture(w, r, "typ")
		default:
			logError("profile picture encode failed", err)
			a.redirectProfilePicture(w, r, "fehler")
		}
		return
	}
	if err := a.personAvatars.SetAvatar(ac.email, avatar); err != nil {
		logError("profile picture save failed", err)
		a.redirectProfilePicture(w, r, "fehler")
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: ac.tenant.Slug,
		ActorEmail: ac.email,
		ActorRole:  ac.role,
		Action:     auditActionProfilePicture,
		TargetType: "profile_picture",
		TargetID:   ac.email,
		Summary:    "Profilbild gesetzt",
		Details:    map[string]string{"size": formatBytes(avatar.ByteSize)},
	})
	a.redirectProfilePicture(w, r, "gespeichert")
}

func (a *app) removeProfilePicture(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if a.personAvatars == nil {
		a.redirectProfilePicture(w, r, "fehler")
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	removed, err := a.personAvatars.DeleteAvatar(ac.email)
	if err != nil {
		logError("profile picture remove failed", err)
		a.redirectProfilePicture(w, r, "fehler")
		return
	}
	if removed {
		a.recordAudit(auditEvent{
			TenantSlug: ac.tenant.Slug,
			ActorEmail: ac.email,
			ActorRole:  ac.role,
			Action:     auditActionProfilePicture,
			TargetType: "profile_picture",
			TargetID:   ac.email,
			Summary:    "Profilbild entfernt",
		})
	}
	a.redirectProfilePicture(w, r, "entfernt")
}

// serveProfilePicture answers the <img> in the shell. It lives outside /app on
// purpose: securityHeaders forces Cache-Control: no-store on everything under
// /app, and the sidebar tile is on every single page — re-fetching the same
// bytes on every navigation is exactly what the content-addressed URL exists to
// avoid.
func (a *app) serveProfilePicture(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if a.personAvatars == nil {
		http.NotFound(w, r)
		return
	}
	avatarID := store.NormalizeAvatarID(r.PathValue("avatarID"))
	if avatarID == "" {
		http.NotFound(w, r)
		return
	}
	avatar, ownerEmail, ok := a.personAvatars.AvatarByID(avatarID)
	if !ok || len(avatar.Image) == 0 {
		http.NotFound(w, r)
		return
	}
	if !a.canViewProfilePicture(ac.email, ownerEmail) {
		// Deliberately 404, not 403: a person who may not see this picture also
		// may not learn that the id exists.
		http.NotFound(w, r)
		return
	}
	contentType := strings.TrimSpace(avatar.ContentType)
	if contentType == "" {
		contentType = store.ProfilePictureContentType
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; img-src 'self' data:; style-src 'unsafe-inline'; sandbox")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": "profilbild.jpg"}))
	// no-cache means "revalidate", not "do not store": with the ETag below the
	// browser keeps the bytes and asks only whether they are still current, so
	// the tile costs a 304 rather than a download on every page.
	//
	// The bounded alternative — max-age — was measured and rejected. A picture
	// removed with "Profilbild entfernen" stayed visible to anyone who had
	// already loaded it for the whole max-age window, because their browser
	// never asked again. Revalidation makes the removal take effect at once,
	// which is the only behaviour that matches what the button promises.
	w.Header().Set("Cache-Control", "private, no-cache")
	if avatar.SHA256 != "" {
		w.Header().Set("ETag", `"`+avatar.SHA256+`"`)
	}
	http.ServeContent(w, r, "profilbild.jpg", avatar.UpdatedAt, bytes.NewReader(avatar.Image))
}

func (a *app) redirectProfilePicture(w http.ResponseWriter, r *http.Request, status string) {
	http.Redirect(w, r, "/app/settings?bild="+status, http.StatusSeeOther)
}

func profilePictureHeader(r *http.Request) *multipart.FileHeader {
	if r.MultipartForm == nil {
		return nil
	}
	files := r.MultipartForm.File["profile_picture"]
	if len(files) == 0 || files[0] == nil || files[0].Size == 0 {
		return nil
	}
	return files[0]
}

func profilePictureMessage(status string) (string, bool) {
	switch status {
	case "gespeichert":
		return "Profilbild gespeichert.", true
	case "entfernt":
		return "Profilbild entfernt.", true
	case "fehlt":
		return "Bitte zuerst ein Bild auswählen.", false
	case "typ":
		return "Dieses Format wird nicht unterstützt. Bitte ein JPEG- oder PNG-Bild wählen.", false
	case "zu-gross":
		return "Das Bild ist zu gross. Erlaubt sind bis zu 8 MB.", false
	case "fehler":
		return "Das Profilbild konnte nicht gespeichert werden.", false
	default:
		return "", false
	}
}
