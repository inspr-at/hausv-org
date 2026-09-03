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
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

type Textbaustein struct {
	Key          string    `json:"key"`
	Organisation string    `json:"organisation"`
	Category     string    `json:"category"`
	Title        string    `json:"title"`
	Body         string    `json:"body"`
	Placeholders []string  `json:"placeholders,omitempty"`
	Active       bool      `json:"active"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UnmarshalJSON keeps catalogue entries and rows written before the active flag
// backwards-compatible: an omitted active property means active, while an
// explicit false remains false.
func (t *Textbaustein) UnmarshalJSON(data []byte) error {
	type plain Textbaustein
	decoded := struct {
		plain
		Active *bool `json:"active"`
	}{plain: plain{Active: true}}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*t = Textbaustein(decoded.plain)
	if decoded.Active != nil {
		t.Active = *decoded.Active
	}
	return nil
}

type TextbausteinRepository interface {
	List(context.Context) ([]Textbaustein, error)
	ListActive(context.Context) ([]Textbaustein, error)
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
	return r.list(ctx, false)
}

func (r *sqlTextbausteinRepository) ListActive(ctx context.Context) ([]Textbaustein, error) {
	return r.list(ctx, true)
}

func (r *sqlTextbausteinRepository) list(ctx context.Context, activeOnly bool) ([]Textbaustein, error) {
	if r.begin == nil || r.orgKey == "" {
		return nil, fmt.Errorf("textbaustein repository is not bound")
	}
	tx, err := r.begin(ctx, r.orgKey)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	query := `SELECT data, active, updated_at FROM textbausteine WHERE org_key=$1 ORDER BY key`
	args := []any{r.orgKey}
	if activeOnly {
		query = `SELECT data, active, updated_at FROM textbausteine WHERE org_key=$1 AND active=$2 ORDER BY key`
		args = append(args, true)
	}
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Textbaustein{}
	for rows.Next() {
		item, err := scanTextbaustein(rows)
		if err != nil {
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
	if r.begin == nil || r.orgKey == "" {
		return Textbaustein{}, fmt.Errorf("textbaustein repository is not bound")
	}
	tx, err := r.begin(ctx, r.orgKey)
	if err != nil {
		return Textbaustein{}, err
	}
	defer tx.Rollback()
	row := tx.QueryRowContext(ctx, `SELECT data, active, updated_at FROM textbausteine WHERE org_key=$1 AND key=$2`, r.orgKey, strings.TrimSpace(key))
	item, err := scanTextbaustein(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Textbaustein{}, ErrIntakeNotFound
	}
	if err != nil {
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
	item.UpdatedAt = time.Now().UTC()
	blob, err := json.Marshal(item)
	if err != nil {
		return err
	}
	tx, err := r.begin(ctx, r.orgKey)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO textbausteine(org_key,key,data,active,updated_at) VALUES($1,$2,$3,$4,$5)
		ON CONFLICT(org_key,key) DO UPDATE SET data=excluded.data, active=excluded.active, updated_at=excluded.updated_at`, r.orgKey, item.Key, string(blob), item.Active, item.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	return tx.Commit()
}

type textbausteinScanner interface {
	Scan(...any) error
}

func scanTextbaustein(scanner textbausteinScanner) (Textbaustein, error) {
	var data, updatedAt string
	var active bool
	if err := scanner.Scan(&data, &active, &updatedAt); err != nil {
		return Textbaustein{}, err
	}
	var item Textbaustein
	if err := json.Unmarshal([]byte(data), &item); err != nil {
		return Textbaustein{}, err
	}
	item.Active = active
	if updatedAt != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, updatedAt); err == nil {
			item.UpdatedAt = parsed.UTC()
		}
	}
	return item, nil
}

func TextbausteinPlaceholders() []string {
	return []string{"{{Anrede}}", "{{Name}}", "{{Haus}}", "{{Einheit}}", "{{Nummer}}", "{{Zuständig}}", "{{Handwerker}}", "{{Frist}}"}
}

var textbausteinPlaceholder = regexp.MustCompile(`\{\{([\p{L}\p{N}_]+)\}\}`)

var (
	emptySalutation = regexp.MustCompile(`Sehr geehrte\{\{Anrede\}\}(?:\s+(?:Frau|Herr))?\s+\{\{Name\}\}`)
	emptyForNumber  = regexp.MustCompile(`\bfür\s*,\s*,\s*unter\s+\s*und\b`)
	emptyForSlot    = regexp.MustCompile(`\bfür\s*,\s*`)
	emptyUnder      = regexp.MustCompile(`\bunter\s+\s*und\b`)
	commaBeforeVerb = regexp.MustCompile(`,\s+(wurde|werden|ist|sind|und)\b`)
	spaces          = regexp.MustCompile(`[\t ]{2,}`)
)

var textbausteinFallbacks = map[string]string{
	"Haus":       "Ihrer Liegenschaft",
	"Einheit":    "Ihrer Einheit",
	"Nummer":     "Ihrem Anliegen",
	"Zuständig":  "die zuständige Person",
	"Handwerker": "einem Fachbetrieb",
	"Frist":      "in Kürze",
}

// FillReply replaces textbaustein placeholders with known values. Missing
// business values receive neutral wording and are returned for human review.
func FillReply(text string, values map[string]string) (string, []string) {
	if values == nil {
		values = map[string]string{}
	}
	unfilled := []string{}
	seen := map[string]bool{}
	addUnfilled := func(key string) {
		if !seen[key] {
			seen[key] = true
			unfilled = append(unfilled, key)
		}
	}
	if strings.TrimSpace(values["Name"]) == "" && emptySalutation.MatchString(text) {
		text = emptySalutation.ReplaceAllString(text, "Sehr geehrte Damen und Herren")
		addUnfilled("Name")
	} else if strings.TrimSpace(values["Anrede"]) == "" && emptySalutation.MatchString(text) {
		// Without a known form of address "Sehr geehrte Simon Schober" is wrong
		// in both genders; the neutral greeting keeps the name.
		text = emptySalutation.ReplaceAllString(text, "Guten Tag "+strings.TrimSpace(values["Name"]))
	}
	text = textbausteinPlaceholder.ReplaceAllStringFunc(text, func(token string) string {
		matches := textbausteinPlaceholder.FindStringSubmatch(token)
		key := matches[1]
		if value := strings.TrimSpace(values[key]); value != "" {
			return value
		}
		// Anrede is intentionally optional; the seed salutations work without it.
		if key == "Anrede" {
			return ""
		}
		if fallback, ok := textbausteinFallbacks[key]; ok {
			addUnfilled(key)
			return fallback
		}
		addUnfilled(key)
		if key == "Name" {
			return "Sie"
		}
		return ""
	})
	return text, unfilled
}

// TidyReply removes punctuation gaps that may be left by a model that copied a
// textbaustein body but replaced its placeholders with empty text.
func TidyReply(text string) string {
	text = emptyForNumber.ReplaceAllString(text, "und")
	text = emptyForSlot.ReplaceAllString(text, "für ")
	text = emptyUnder.ReplaceAllString(text, "und")
	text = commaBeforeVerb.ReplaceAllString(text, " $1")
	text = spaces.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

func RenderTextbaustein(body string, values map[string]string) string {
	rendered, _ := FillReply(body, values)
	return rendered
}
