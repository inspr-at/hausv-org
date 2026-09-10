package demo

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/mailintake"
	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/textutil"
)

func loadMailboxFixture(dir string) ([]mailintake.Message, error) {
	entries, err := os.ReadDir(filepath.Join(dir, "mail"))
	if os.IsNotExist(err) {
		return nil, nil // Mailbox fixtures are optional.
	}
	if err != nil {
		return nil, fmt.Errorf("read demo mailbox fixture: %w", err)
	}
	var messages []mailintake.Message
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || !strings.EqualFold(filepath.Ext(entry.Name()), ".eml") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, "mail", entry.Name()))
		if err != nil {
			return nil, err
		}
		message, err := mailintake.Parse(raw, mailintake.Limits{})
		if err != nil {
			return nil, fmt.Errorf("parse demo mailbox fixture %s: %w", entry.Name(), err)
		}
		messages = append(messages, message)
	}
	return messages, nil
}

// resetMailboxFixture runs inside the reset transaction. The seed owns the
// clean demo day; the mailbox's static files must not reopen it afterwards.
// Old resets removed intake items but left their randomly named issues behind.
func resetMailboxFixture(ctx context.Context, tx *sql.Tx, orgKey string, houses []seedHouse, messages []mailintake.Message) error {
	if len(messages) == 0 {
		return nil
	}
	for _, message := range messages {
		if _, err := tx.ExecContext(ctx, `INSERT INTO intake_mail_seen(org_key,message_id,intake_id,seen_at) VALUES($1,$2,'',$3)
			ON CONFLICT(org_key,message_id) DO NOTHING`, orgKey, message.DedupeKey(), time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			return err
		}
	}
	for _, house := range houses {
		slug := textutil.Slug(house.Slug)
		rows, err := tx.QueryContext(ctx, `SELECT id,data FROM issues WHERE tenant_slug=$1`, slug)
		if err != nil {
			return err
		}
		var obsolete []string
		for rows.Next() {
			var id string
			var raw []byte
			if err := rows.Scan(&id, &raw); err != nil {
				rows.Close()
				return err
			}
			var issue store.ResidentIssue
			if err := json.Unmarshal(raw, &issue); err != nil {
				rows.Close()
				return err
			}
			if issue.Source != string(store.IntakeSourceEmail) || issue.IntakeID == "" {
				continue
			}
			// Earlier resets erased the Message-ID linkage. Restrict legacy
			// cleanup to mail-created issues matching a local fixture's sender
			// and subject, in the reset's own demo houses.
			for _, message := range messages {
				if strings.EqualFold(strings.TrimSpace(issue.AuthorEmail), message.FromEmail) && strings.TrimSpace(issue.Title) == message.Subject {
					obsolete = append(obsolete, id)
					break
				}
			}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		if err := rows.Close(); err != nil {
			return err
		}
		for _, id := range obsolete {
			if _, err := tx.ExecContext(ctx, `DELETE FROM issues WHERE tenant_slug=$1 AND id=$2`, slug, id); err != nil {
				return err
			}
		}
	}
	return nil
}
