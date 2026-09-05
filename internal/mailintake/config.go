// Package mailintake fetches a Verwaltung's mailbox and turns each unread
// message into something the Posteingang can take: sender, subject, text, and
// attachments within the same limits the portal applies to uploads.
//
// It knows nothing about houses, persons or triage. Matching a message to a
// house and a person is the server's job; this package only reads mail.
package mailintake

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

// Config is one organisation's mailbox. The password is never part of the
// configuration text: like the Home Assistant connectors, it is referenced by
// file or by environment variable name and resolved at connect time.
type Config struct {
	Organisation string        `json:"-"`
	Host         string        `json:"host"`
	Port         int           `json:"port,omitempty"`
	Username     string        `json:"username"`
	PasswordFile string        `json:"password_file,omitempty"`
	PasswordEnv  string        `json:"password_env,omitempty"`
	Folder       string        `json:"folder,omitempty"`
	Interval     time.Duration `json:"-"`
	IntervalRaw  string        `json:"interval,omitempty"`
	// Insecure connects without TLS. Only the test server speaks plain IMAP;
	// a production mailbox must never be reached this way.
	Insecure bool `json:"insecure,omitempty"`
}

const (
	defaultPort     = 993
	defaultFolder   = "INBOX"
	defaultInterval = 2 * time.Minute
	minInterval     = 15 * time.Second
)

// ParseConfigs reads INTAKE_MAIL_JSON: an object keyed by organisation.
//
//	{"musterstadt": {"host": "imap.example", "username": "post@…",
//	                 "password_env": "INTAKE_MAIL_PASSWORD_MUSTERSTADT",
//	                 "folder": "INBOX", "interval": "2m"}}
func ParseConfigs(raw string) (map[string]Config, error) {
	out := map[string]Config{}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return out, nil
	}
	var parsed map[string]Config
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, errors.New("invalid INTAKE_MAIL_JSON")
	}
	for rawKey, config := range parsed {
		key := textutil.Slug(rawKey)
		if key == "" {
			return nil, errors.New("INTAKE_MAIL_JSON: organisation key is required")
		}
		config.Organisation = key
		config.Host = strings.TrimSpace(config.Host)
		config.Username = strings.TrimSpace(config.Username)
		if config.Host == "" || config.Username == "" {
			return nil, fmt.Errorf("INTAKE_MAIL_JSON: %s needs host and username", key)
		}
		if config.Port == 0 {
			config.Port = defaultPort
		}
		if config.Port < 1 || config.Port > 65535 {
			return nil, fmt.Errorf("INTAKE_MAIL_JSON: %s has an invalid port", key)
		}
		config.Folder = strings.TrimSpace(config.Folder)
		if config.Folder == "" {
			config.Folder = defaultFolder
		}
		config.PasswordFile = strings.TrimSpace(config.PasswordFile)
		config.PasswordEnv = strings.TrimSpace(config.PasswordEnv)
		if (config.PasswordFile == "") == (config.PasswordEnv == "") {
			return nil, fmt.Errorf("INTAKE_MAIL_JSON: %s needs exactly one of password_file or password_env", key)
		}
		if strings.ContainsAny(config.PasswordEnv, " \t\r\n=") {
			return nil, fmt.Errorf("INTAKE_MAIL_JSON: %s has an invalid password_env name", key)
		}
		config.Interval = defaultInterval
		if strings.TrimSpace(config.IntervalRaw) != "" {
			interval, err := time.ParseDuration(strings.TrimSpace(config.IntervalRaw))
			if err != nil || interval < minInterval {
				return nil, fmt.Errorf("INTAKE_MAIL_JSON: %s interval must be a duration of at least %s", key, minInterval)
			}
			config.Interval = interval
		}
		out[key] = config
	}
	return out, nil
}

// Password resolves the referenced secret. The value never appears in logs or
// errors: every failure reports which source failed, not what it held.
func (c Config) Password(getenv func(string) string) (string, error) {
	if c.PasswordFile != "" {
		raw, err := os.ReadFile(c.PasswordFile)
		if err != nil {
			return "", errors.New("mail intake: password file unavailable")
		}
		password := strings.TrimSpace(string(raw))
		if password == "" {
			return "", errors.New("mail intake: password file empty")
		}
		return password, nil
	}
	if getenv == nil {
		getenv = os.Getenv
	}
	password := strings.TrimSpace(getenv(c.PasswordEnv))
	if password == "" {
		return "", errors.New("mail intake: password env unset")
	}
	return password, nil
}

// Address is host:port for dialling.
func (c Config) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// Keys lists the configured organisations in a stable order.
func Keys(configs map[string]Config) []string {
	keys := make([]string, 0, len(configs))
	for key := range configs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
