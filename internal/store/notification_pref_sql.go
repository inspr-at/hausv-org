package store

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

// NotificationPrefStorage is the behaviour both the JSON NotificationPrefStore
// and the SQLite SQLNotificationPrefStore satisfy (HAUSV-168).
type NotificationPrefStorage interface {
	Get(email string) NotificationPreferences
	Set(email string, prefs NotificationPreferences) error
	EmailEnabled(email string, event string) bool
}

var (
	_ NotificationPrefStorage = (*NotificationPrefStore)(nil)
	_ NotificationPrefStorage = (*SQLNotificationPrefStore)(nil)
)

// SQLNotificationPrefStore stores each user's preferences as a JSON document in
// a TEXT column — the value carries a per-event map and is never queried
// relationally, so a JSON column preserves exact semantics without a schema for
// every event. Table from migration 0004_notification_prefs.
type SQLNotificationPrefStore struct {
	db *sql.DB
}

func NewSQLNotificationPrefStore(db *sql.DB) *SQLNotificationPrefStore {
	return &SQLNotificationPrefStore{db: db}
}

func (s *SQLNotificationPrefStore) Get(email string) NotificationPreferences {
	prefs := DefaultNotificationPreferences()
	if s == nil {
		return prefs
	}
	email = textutil.Email(email)
	if email == "" {
		return prefs
	}
	var raw string
	if err := s.db.QueryRow(`SELECT prefs FROM notification_prefs WHERE email = $1`, email).Scan(&raw); err != nil {
		return prefs // not found (or read error) -> default, matching the JSON store
	}
	var stored NotificationPreferences
	if err := json.Unmarshal([]byte(raw), &stored); err != nil {
		return prefs
	}
	return MergeNotificationPreferences(stored)
}

func (s *SQLNotificationPrefStore) Set(email string, prefs NotificationPreferences) error {
	if s == nil {
		return nil
	}
	email = textutil.Email(email)
	if email == "" {
		return fmt.Errorf("invalid notification preference email")
	}
	prefs = NormalizeNotificationPreferences(prefs)
	raw, err := json.Marshal(prefs)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`INSERT INTO notification_prefs(email, prefs) VALUES($1, $2)
		 ON CONFLICT(email) DO UPDATE SET prefs = excluded.prefs`,
		email, string(raw),
	)
	return err
}

func (s *SQLNotificationPrefStore) EmailEnabled(email string, event string) bool {
	event = NormalizeNotificationEvent(event)
	if event == "" {
		return false
	}
	prefs := DefaultNotificationPreferences()
	if s != nil {
		prefs = s.Get(email)
	}
	if prefs.Unsubscribed {
		return false
	}
	if prefs.Email == nil {
		return true
	}
	enabled, ok := prefs.Email[event]
	if !ok {
		return true
	}
	return enabled
}

// ImportPrefs copies records from a JSON NotificationPrefStore, each only if
// absent (clobber-safe on every boot) (HAUSV-170).
func (s *SQLNotificationPrefStore) ImportPrefs(src *NotificationPrefStore) error {
	if src == nil {
		return nil
	}
	src.mu.Lock()
	snapshot := make(map[string]NotificationPreferences, len(src.data.Users))
	for email, prefs := range src.data.Users {
		snapshot[email] = prefs
	}
	src.mu.Unlock()
	for rawEmail, prefs := range snapshot {
		email := textutil.Email(rawEmail)
		if email == "" {
			continue
		}
		blob, err := json.Marshal(NormalizeNotificationPreferences(prefs))
		if err != nil {
			return err
		}
		if _, err := s.db.Exec(
			`INSERT INTO notification_prefs(email, prefs) VALUES($1, $2) ON CONFLICT(email) DO NOTHING`,
			email, string(blob),
		); err != nil {
			return err
		}
	}
	return nil
}
