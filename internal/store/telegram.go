package store

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/markus-barta/hausv-org/internal/textutil"
)

// TelegramStore persists what the bot must not lose across restarts: the
// getUpdates offset, chat↔user links, and pending one-time link codes.
// Chat IDs live only here (in /data), never in env or git.
type TelegramStore struct {
	mu   sync.Mutex
	path string
	data TelegramStoreData
}

type TelegramStoreData struct {
	Offset int64              `json:"offset"`
	Links  []TelegramLink     `json:"links,omitempty"`
	Codes  []TelegramLinkCode `json:"codes,omitempty"`
}

type TelegramLink struct {
	ChatID   int64     `json:"chat_id"`
	Email    string    `json:"email"`
	Name     string    `json:"name,omitempty"`
	LinkedAt time.Time `json:"linked_at"`
}

type TelegramLinkCode struct {
	Code      string    `json:"code"`
	Email     string    `json:"email"`
	CreatedBy string    `json:"created_by,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
}

func NewTelegramStore(path string) (*TelegramStore, error) {
	store := &TelegramStore{path: path}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read telegram data")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("invalid telegram data")
	}
	return store, nil
}

func (s *TelegramStore) saveLocked() error {
	return SaveJSONAtomic(s.path, s.data, "telegram")
}

func (s *TelegramStore) Offset() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data.Offset
}

func (s *TelegramStore) SetOffset(offset int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if offset == s.data.Offset {
		return nil
	}
	s.data.Offset = offset
	return s.saveLocked()
}

func (s *TelegramStore) Links() []TelegramLink {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]TelegramLink(nil), s.data.Links...)
}

func (s *TelegramStore) LinkByChat(chatID int64) (TelegramLink, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, link := range s.data.Links {
		if link.ChatID == chatID {
			return link, true
		}
	}
	return TelegramLink{}, false
}

func (s *TelegramStore) ChatsByEmail(email string) []int64 {
	email = textutil.Email(email)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []int64{}
	for _, link := range s.data.Links {
		if link.Email == email {
			out = append(out, link.ChatID)
		}
	}
	return out
}

// CreateLinkCode mints a one-time code bound to a user email; sending
// /start <code> to the bot links the sender's chat to that email.
func (s *TelegramStore) CreateLinkCode(email, createdBy string, ttl time.Duration) (string, error) {
	email = textutil.Email(email)
	if email == "" {
		return "", fmt.Errorf("invalid email")
	}
	code, err := randomLinkCode()
	if err != nil {
		return "", fmt.Errorf("could not create link code")
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// One pending code per email: a fresh code replaces the old one.
	kept := s.data.Codes[:0]
	for _, existing := range s.data.Codes {
		if existing.Email != email && existing.ExpiresAt.After(time.Now()) {
			kept = append(kept, existing)
		}
	}
	s.data.Codes = append(kept, TelegramLinkCode{
		Code:      code,
		Email:     email,
		CreatedBy: textutil.Email(createdBy),
		ExpiresAt: time.Now().Add(ttl).UTC(),
	})
	if err := s.saveLocked(); err != nil {
		return "", err
	}
	return code, nil
}

// ConsumeLinkCode redeems a code: the chat is linked to the code's email.
// A chat can only be linked once; redeeming re-links it to the new email.
func (s *TelegramStore) ConsumeLinkCode(code string, chatID int64, name string) (TelegramLink, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" || chatID == 0 {
		return TelegramLink{}, fmt.Errorf("invalid link code")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	email := ""
	kept := s.data.Codes[:0]
	for _, candidate := range s.data.Codes {
		if candidate.Code == code && candidate.ExpiresAt.After(time.Now()) {
			email = candidate.Email
			continue
		}
		if candidate.ExpiresAt.After(time.Now()) {
			kept = append(kept, candidate)
		}
	}
	s.data.Codes = kept
	if email == "" {
		_ = s.saveLocked()
		return TelegramLink{}, fmt.Errorf("unknown or expired link code")
	}
	link := TelegramLink{ChatID: chatID, Email: email, Name: strings.TrimSpace(name), LinkedAt: time.Now().UTC()}
	links := s.data.Links[:0]
	for _, existing := range s.data.Links {
		if existing.ChatID != chatID {
			links = append(links, existing)
		}
	}
	s.data.Links = append(links, link)
	if err := s.saveLocked(); err != nil {
		return TelegramLink{}, err
	}
	return link, nil
}

// randomLinkCode mints an 8-char code from an unambiguous uppercase
// alphabet (no 0/O, 1/I) — it gets typed on a phone. 32 characters divide
// 256 evenly, so the modulo introduces no bias.
func randomLinkCode() (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, len(buf))
	for i, b := range buf {
		out[i] = alphabet[int(b)%len(alphabet)]
	}
	return string(out), nil
}

func (s *TelegramStore) Unlink(chatID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	links := s.data.Links[:0]
	for _, existing := range s.data.Links {
		if existing.ChatID != chatID {
			links = append(links, existing)
		}
	}
	s.data.Links = links
	return s.saveLocked()
}
