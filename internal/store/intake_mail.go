package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

// IntakeAttachment is a file that arrived with a mail. It lives beside the
// intake item until approval hands it to the Anliegen through the regular
// attachment store, which is where the durable copy belongs.
type IntakeAttachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	// Path is where the bytes wait, relative to the intake mail directory.
	Path string `json:"path"`
}

// IntakeMailSeenRepository remembers which mails an organisation has already
// taken, by Message-ID. It is the idempotency ledger of the mail intake.
type IntakeMailSeenRepository interface {
	Seen(context.Context, string) (bool, error)
	Record(context.Context, string, string) error
	Count(context.Context) (int, error)
}

type sqlIntakeMailSeenRepository struct {
	begin  func(context.Context, string) (*sql.Tx, error)
	orgKey string
}

func BindIntakeMailSeenRepository(database *sql.DB, orgKey string) IntakeMailSeenRepository {
	return &sqlIntakeMailSeenRepository{
		begin:  func(ctx context.Context, key string) (*sql.Tx, error) { return beginOrgTx(ctx, database, key) },
		orgKey: textutil.Slug(orgKey),
	}
}

func normalizeMessageID(id string) string {
	return strings.Trim(strings.TrimSpace(id), "<> ")
}

func (r *sqlIntakeMailSeenRepository) bound() error {
	if r.begin == nil || r.orgKey == "" {
		return fmt.Errorf("intake mail repository is not bound")
	}
	return nil
}

func (r *sqlIntakeMailSeenRepository) Seen(ctx context.Context, messageID string) (bool, error) {
	if err := r.bound(); err != nil {
		return false, err
	}
	messageID = normalizeMessageID(messageID)
	if messageID == "" {
		return false, fmt.Errorf("message id is required")
	}
	tx, err := r.begin(ctx, r.orgKey)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM intake_mail_seen WHERE org_key=$1 AND message_id=$2`,
		r.orgKey, messageID).Scan(&count); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *sqlIntakeMailSeenRepository) Record(ctx context.Context, messageID string, intakeID string) error {
	if err := r.bound(); err != nil {
		return err
	}
	messageID = normalizeMessageID(messageID)
	if messageID == "" {
		return fmt.Errorf("message id is required")
	}
	tx, err := r.begin(ctx, r.orgKey)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO intake_mail_seen(org_key,message_id,intake_id,seen_at) VALUES($1,$2,$3,$4)
		ON CONFLICT(org_key,message_id) DO NOTHING`,
		r.orgKey, messageID, strings.TrimSpace(intakeID), time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *sqlIntakeMailSeenRepository) Count(ctx context.Context) (int, error) {
	if err := r.bound(); err != nil {
		return 0, err
	}
	tx, err := r.begin(ctx, r.orgKey)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM intake_mail_seen WHERE org_key=$1`, r.orgKey).Scan(&count); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return count, nil
}
