package store

import (
	"fmt"
	"time"
)

// SQLPersonAvatarStore is the database-backed profile picture store. Table
// created by migration 0049_person_avatars (SQLite) / 0022_person_avatars
// (PostgreSQL).
type SQLPersonAvatarStore struct {
	db *TenantDB
}

func NewSQLPersonAvatarStore(db *TenantDB) *SQLPersonAvatarStore {
	return &SQLPersonAvatarStore{db: db}
}

// personAvatarLane is the one reason string this store uses. person_avatars has
// no tenant_id for the same reason profile_overlays has none: the picture is
// part of the person and follows them into every house they belong to.
const personAvatarLane = "person_avatars has no tenant_id: the profile picture follows the person across every house"

func (s *SQLPersonAvatarStore) Avatar(email string) (PersonAvatar, bool) {
	if s == nil {
		return PersonAvatar{}, false
	}
	email = NormalizeProfilePictureEmail(email)
	if email == "" {
		return PersonAvatar{}, false
	}
	row := s.db.Unscoped(personAvatarLane).QueryRow(
		`SELECT avatar_id, content_type, image, byte_size, sha256, updated_at
		 FROM person_avatars WHERE email = $1`, email)
	avatar, err := scanPersonAvatar(row.Scan)
	if err != nil {
		return PersonAvatar{}, false
	}
	return avatar, true
}

func (s *SQLPersonAvatarStore) AvatarByID(avatarID string) (PersonAvatar, string, bool) {
	if s == nil {
		return PersonAvatar{}, "", false
	}
	avatarID = NormalizeAvatarID(avatarID)
	if avatarID == "" {
		return PersonAvatar{}, "", false
	}
	var email string
	row := s.db.Unscoped(personAvatarLane).QueryRow(
		`SELECT avatar_id, content_type, image, byte_size, sha256, updated_at, email
		 FROM person_avatars WHERE avatar_id = $1`, avatarID)
	avatar, err := scanPersonAvatarWithEmail(row.Scan, &email)
	if err != nil {
		return PersonAvatar{}, "", false
	}
	return avatar, email, true
}

func (s *SQLPersonAvatarStore) SetAvatar(email string, avatar PersonAvatar) error {
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
	if avatar.ContentType == "" {
		avatar.ContentType = ProfilePictureContentType
	}
	if avatar.UpdatedAt.IsZero() {
		avatar.UpdatedAt = time.Now()
	}
	_, err := s.db.Unscoped(personAvatarLane).Exec(
		`INSERT INTO person_avatars(email, avatar_id, content_type, image, byte_size, sha256, updated_at)
		 VALUES($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT(email) DO UPDATE SET
		   avatar_id=excluded.avatar_id, content_type=excluded.content_type, image=excluded.image,
		   byte_size=excluded.byte_size, sha256=excluded.sha256, updated_at=excluded.updated_at`,
		email, avatar.AvatarID, avatar.ContentType, avatar.Image, avatar.ByteSize, avatar.SHA256,
		avatar.UpdatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("store: save profile picture: %w", err)
	}
	return nil
}

func (s *SQLPersonAvatarStore) DeleteAvatar(email string) (bool, error) {
	if s == nil {
		return false, nil
	}
	email = NormalizeProfilePictureEmail(email)
	if email == "" {
		return false, nil
	}
	result, err := s.db.Unscoped(personAvatarLane).Exec(`DELETE FROM person_avatars WHERE email = $1`, email)
	if err != nil {
		return false, fmt.Errorf("store: remove profile picture: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("store: remove profile picture: %w", err)
	}
	return affected > 0, nil
}

func scanPersonAvatar(scan func(dest ...any) error) (PersonAvatar, error) {
	var (
		avatar    PersonAvatar
		updatedAt string
	)
	if err := scan(&avatar.AvatarID, &avatar.ContentType, &avatar.Image, &avatar.ByteSize, &avatar.SHA256, &updatedAt); err != nil {
		return PersonAvatar{}, err
	}
	if parsed, err := time.Parse(time.RFC3339Nano, updatedAt); err == nil {
		avatar.UpdatedAt = parsed.UTC()
	}
	return avatar, nil
}

func scanPersonAvatarWithEmail(scan func(dest ...any) error, email *string) (PersonAvatar, error) {
	var (
		avatar    PersonAvatar
		updatedAt string
	)
	if err := scan(&avatar.AvatarID, &avatar.ContentType, &avatar.Image, &avatar.ByteSize, &avatar.SHA256, &updatedAt, email); err != nil {
		return PersonAvatar{}, err
	}
	if parsed, err := time.Parse(time.RFC3339Nano, updatedAt); err == nil {
		avatar.UpdatedAt = parsed.UTC()
	}
	return avatar, nil
}
