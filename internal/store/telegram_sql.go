package store

import (
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/markus-barta/hausv-org/internal/textutil"
)

// TelegramStorage is the behaviour both the JSON TelegramStore and the SQLite
// SQLTelegramStore satisfy (HAUSV-168).
type TelegramStorage interface {
	Offset() int64
	SetOffset(offset int64) error
	Links() []TelegramLink
	LinkByChat(chatID int64) (TelegramLink, bool)
	ChatsByEmail(email string) []int64
	CreateLinkCode(email string, createdBy string, ttl time.Duration) (string, error)
	ConsumeLinkCode(code string, chatID int64, name string) (TelegramLink, error)
	Unlink(chatID int64) error
}

var (
	_ TelegramStorage = (*TelegramStore)(nil)
	_ TelegramStorage = (*SQLTelegramStore)(nil)
)

const telegramOffsetKey = "offset"

// SQLTelegramStore keeps the bot offset, the chat links and the pending link
// codes in three tables from migration 0013.
type SQLTelegramStore struct {
	db *sql.DB
}

func NewSQLTelegramStore(db *sql.DB) *SQLTelegramStore {
	return &SQLTelegramStore{db: db}
}

func telegramTime(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func parseTelegramTime(raw string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

func (s *SQLTelegramStore) Offset() int64 {
	if s == nil {
		return 0
	}
	var raw string
	if err := s.db.QueryRow(`SELECT value FROM telegram_state WHERE key=?`, telegramOffsetKey).Scan(&raw); err != nil {
		return 0
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func (s *SQLTelegramStore) SetOffset(offset int64) error {
	if s == nil {
		return nil
	}
	if offset == s.Offset() {
		return nil
	}
	_, err := s.db.Exec(
		`INSERT INTO telegram_state(key, value) VALUES(?, ?)
		 ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
		telegramOffsetKey, strconv.FormatInt(offset, 10),
	)
	return err
}

func (s *SQLTelegramStore) scanLinks(rows *sql.Rows) []TelegramLink {
	defer rows.Close()
	out := []TelegramLink{}
	for rows.Next() {
		var link TelegramLink
		var linkedAt string
		if err := rows.Scan(&link.ChatID, &link.Email, &link.Name, &linkedAt); err != nil {
			continue
		}
		link.LinkedAt = parseTelegramTime(linkedAt)
		out = append(out, link)
	}
	return out
}

func (s *SQLTelegramStore) Links() []TelegramLink {
	if s == nil {
		return nil
	}
	rows, err := s.db.Query(`SELECT chat_id, email, name, linked_at FROM telegram_links`)
	if err != nil {
		return []TelegramLink{}
	}
	out := s.scanLinks(rows)
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].LinkedAt.Equal(out[j].LinkedAt) {
			return out[i].LinkedAt.Before(out[j].LinkedAt)
		}
		return out[i].ChatID < out[j].ChatID
	})
	return out
}

func (s *SQLTelegramStore) LinkByChat(chatID int64) (TelegramLink, bool) {
	if s == nil {
		return TelegramLink{}, false
	}
	var link TelegramLink
	var linkedAt string
	if err := s.db.QueryRow(
		`SELECT chat_id, email, name, linked_at FROM telegram_links WHERE chat_id=?`, chatID,
	).Scan(&link.ChatID, &link.Email, &link.Name, &linkedAt); err != nil {
		return TelegramLink{}, false
	}
	link.LinkedAt = parseTelegramTime(linkedAt)
	return link, true
}

func (s *SQLTelegramStore) ChatsByEmail(email string) []int64 {
	if s == nil {
		return nil
	}
	email = textutil.Email(email)
	rows, err := s.db.Query(`SELECT chat_id FROM telegram_links WHERE email=? ORDER BY chat_id`, email)
	if err != nil {
		return []int64{}
	}
	defer rows.Close()
	out := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			continue
		}
		out = append(out, id)
	}
	return out
}

// CreateLinkCode mints a one-time code bound to an email. One pending code per
// email: a fresh code replaces the old one, and expired codes are pruned.
func (s *SQLTelegramStore) CreateLinkCode(email string, createdBy string, ttl time.Duration) (string, error) {
	if s == nil {
		return "", fmt.Errorf("telegram store unavailable")
	}
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
	now := time.Now()
	tx, err := s.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	if err := pruneTelegramCodesTx(tx, now); err != nil {
		return "", err
	}
	if _, err := tx.Exec(`DELETE FROM telegram_link_codes WHERE email=?`, email); err != nil {
		return "", err
	}
	if _, err := tx.Exec(
		`INSERT INTO telegram_link_codes(code, email, created_by, expires_at) VALUES(?, ?, ?, ?)`,
		code, email, textutil.Email(createdBy), telegramTime(now.Add(ttl)),
	); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return code, nil
}

// pruneTelegramCodesTx drops codes that have expired. Times are compared in Go
// because RFC3339Nano strings do not sort reliably (variable fraction length).
func pruneTelegramCodesTx(tx *sql.Tx, now time.Time) error {
	rows, err := tx.Query(`SELECT code, expires_at FROM telegram_link_codes`)
	if err != nil {
		return err
	}
	stale := []string{}
	for rows.Next() {
		var code, expires string
		if err := rows.Scan(&code, &expires); err != nil {
			continue
		}
		if !parseTelegramTime(expires).After(now) {
			stale = append(stale, code)
		}
	}
	rows.Close()
	for _, code := range stale {
		if _, err := tx.Exec(`DELETE FROM telegram_link_codes WHERE code=?`, code); err != nil {
			return err
		}
	}
	return nil
}

// ConsumeLinkCode redeems a code and links the chat to its email. A chat can be
// linked only once; redeeming re-links it. Expired codes are pruned either way.
func (s *SQLTelegramStore) ConsumeLinkCode(code string, chatID int64, name string) (TelegramLink, error) {
	if s == nil {
		return TelegramLink{}, fmt.Errorf("telegram store unavailable")
	}
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" || chatID == 0 {
		return TelegramLink{}, fmt.Errorf("invalid link code")
	}
	now := time.Now()
	tx, err := s.db.Begin()
	if err != nil {
		return TelegramLink{}, err
	}
	defer tx.Rollback()

	email := ""
	var expires string
	if err := tx.QueryRow(`SELECT email, expires_at FROM telegram_link_codes WHERE code=?`, code).Scan(&email, &expires); err == nil {
		if !parseTelegramTime(expires).After(now) {
			email = ""
		}
	} else {
		email = ""
	}
	if err := pruneTelegramCodesTx(tx, now); err != nil {
		return TelegramLink{}, err
	}
	if email == "" {
		// Still commit the pruning, then report the failure like the JSON store.
		_ = tx.Commit()
		return TelegramLink{}, fmt.Errorf("unknown or expired link code")
	}
	if _, err := tx.Exec(`DELETE FROM telegram_link_codes WHERE code=?`, code); err != nil {
		return TelegramLink{}, err
	}
	link := TelegramLink{ChatID: chatID, Email: email, Name: strings.TrimSpace(name), LinkedAt: now.UTC()}
	if _, err := tx.Exec(
		`INSERT INTO telegram_links(chat_id, email, name, linked_at) VALUES(?, ?, ?, ?)
		 ON CONFLICT(chat_id) DO UPDATE SET email=excluded.email, name=excluded.name, linked_at=excluded.linked_at`,
		link.ChatID, link.Email, link.Name, telegramTime(link.LinkedAt),
	); err != nil {
		return TelegramLink{}, err
	}
	if err := tx.Commit(); err != nil {
		return TelegramLink{}, err
	}
	return link, nil
}

func (s *SQLTelegramStore) Unlink(chatID int64) error {
	if s == nil {
		return nil
	}
	_, err := s.db.Exec(`DELETE FROM telegram_links WHERE chat_id=?`, chatID)
	return err
}

// ImportTelegram copies offset, links and codes from a JSON store, each only if
// absent (clobber-safe) (HAUSV-170).
func (s *SQLTelegramStore) ImportTelegram(src *TelegramStore) error {
	if src == nil {
		return nil
	}
	src.mu.Lock()
	offset := src.data.Offset
	links := append([]TelegramLink(nil), src.data.Links...)
	codes := append([]TelegramLinkCode(nil), src.data.Codes...)
	src.mu.Unlock()

	if offset != 0 {
		if _, err := s.db.Exec(
			`INSERT INTO telegram_state(key, value) VALUES(?, ?) ON CONFLICT(key) DO NOTHING`,
			telegramOffsetKey, strconv.FormatInt(offset, 10),
		); err != nil {
			return err
		}
	}
	for _, link := range links {
		if link.ChatID == 0 {
			continue
		}
		if _, err := s.db.Exec(
			`INSERT INTO telegram_links(chat_id, email, name, linked_at) VALUES(?, ?, ?, ?)
			 ON CONFLICT(chat_id) DO NOTHING`,
			link.ChatID, textutil.Email(link.Email), strings.TrimSpace(link.Name), telegramTime(link.LinkedAt),
		); err != nil {
			return err
		}
	}
	for _, code := range codes {
		if strings.TrimSpace(code.Code) == "" {
			continue
		}
		if _, err := s.db.Exec(
			`INSERT INTO telegram_link_codes(code, email, created_by, expires_at) VALUES(?, ?, ?, ?)
			 ON CONFLICT(code) DO NOTHING`,
			code.Code, textutil.Email(code.Email), textutil.Email(code.CreatedBy), telegramTime(code.ExpiresAt),
		); err != nil {
			return err
		}
	}
	return nil
}
