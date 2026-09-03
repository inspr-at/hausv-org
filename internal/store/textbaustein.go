package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

type Textbaustein struct {
	Key          string   `json:"key"`
	Organisation string   `json:"organisation"`
	Category     string   `json:"category"`
	Title        string   `json:"title"`
	Body         string   `json:"body"`
	Placeholders []string `json:"placeholders,omitempty"`
	Active       bool     `json:"active"`
}

type TextbausteinRepository interface {
	List(context.Context) ([]Textbaustein, error)
	Get(context.Context, string) (Textbaustein, error)
	Upsert(context.Context, Textbaustein) error
}

type sqlTextbausteinRepository struct {
	begin  func(context.Context, string) (*sql.Tx, error)
	orgKey string
}

func BindTextbausteinRepository(database *sql.DB, orgKey string) TextbausteinRepository {
	return &sqlTextbausteinRepository{
		begin:  func(ctx context.Context, orgKey string) (*sql.Tx, error) { return beginOrgTx(ctx, database, orgKey) },
		orgKey: textutil.Slug(orgKey),
	}
}

func (r *sqlTextbausteinRepository) List(ctx context.Context) ([]Textbaustein, error) {
	if r.begin == nil || r.orgKey == "" {
		return nil, fmt.Errorf("textbaustein repository is not bound")
	}
	tx, err := r.begin(ctx, r.orgKey)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT data FROM textbausteine WHERE org_key=$1 ORDER BY key`, r.orgKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Textbaustein{}
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var item Textbaustein
		if err := json.Unmarshal([]byte(data), &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *sqlTextbausteinRepository) Get(ctx context.Context, key string) (Textbaustein, error) {
	var data string
	if r.begin == nil || r.orgKey == "" {
		return Textbaustein{}, fmt.Errorf("textbaustein repository is not bound")
	}
	tx, err := r.begin(ctx, r.orgKey)
	if err != nil {
		return Textbaustein{}, err
	}
	defer tx.Rollback()
	err = tx.QueryRowContext(ctx, `SELECT data FROM textbausteine WHERE org_key=$1 AND key=$2`, r.orgKey, strings.TrimSpace(key)).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return Textbaustein{}, ErrIntakeNotFound
	}
	if err != nil {
		return Textbaustein{}, err
	}
	var item Textbaustein
	if err := json.Unmarshal([]byte(data), &item); err != nil {
		return Textbaustein{}, err
	}
	if err := tx.Commit(); err != nil {
		return Textbaustein{}, err
	}
	return item, nil
}

func (r *sqlTextbausteinRepository) Upsert(ctx context.Context, item Textbaustein) error {
	item.Key = strings.TrimSpace(item.Key)
	item.Organisation = r.orgKey
	item.Category = strings.TrimSpace(item.Category)
	item.Title = strings.TrimSpace(item.Title)
	if r.begin == nil || r.orgKey == "" || item.Key == "" || item.Category == "" || item.Title == "" {
		return fmt.Errorf("invalid textbaustein")
	}
	item.Placeholders = append([]string(nil), item.Placeholders...)
	sort.Strings(item.Placeholders)
	blob, err := json.Marshal(item)
	if err != nil {
		return err
	}
	tx, err := r.begin(ctx, r.orgKey)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO textbausteine(org_key,key,data) VALUES($1,$2,$3)
		ON CONFLICT(org_key,key) DO UPDATE SET data=excluded.data`, r.orgKey, item.Key, string(blob))
	if err != nil {
		return err
	}
	return tx.Commit()
}

var textbausteinPlaceholder = regexp.MustCompile(`\{\{([[:alnum:]_]+)\}\}`)

func RenderTextbaustein(body string, values map[string]string) string {
	return textbausteinPlaceholder.ReplaceAllStringFunc(body, func(token string) string {
		matches := textbausteinPlaceholder.FindStringSubmatch(token)
		return values[matches[1]]
	})
}
