package ai

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

type cliTruth struct {
	Category    string `json:"category"`
	Priority    string `json:"priority"`
	Assignee    string `json:"assignee"`
	TemplateKey string `json:"template_key"`
}

type cliIntakeItem struct {
	ID           string         `json:"id"`
	Organisation string         `json:"organisation"`
	Source       string         `json:"source"`
	ReceivedAt   string         `json:"received_at"`
	House        string         `json:"house"`
	Unit         string         `json:"unit"`
	FromName     string         `json:"from_name"`
	FromEmail    string         `json:"from_email"`
	FromPhone    string         `json:"from_phone"`
	Subject      string         `json:"subject"`
	Body         string         `json:"body"`
	Categories   []CategoryRule `json:"categories"`
	Assignees    []AssigneeHint `json:"assignees"`
	Truth        *cliTruth      `json:"truth"`
}

type cliHouseHint struct {
	Slug    string `json:"slug"`
	Name    string `json:"name"`
	Address string `json:"address"`
	Units   []struct {
		Label string `json:"label"`
	} `json:"units"`
}

// RunCLI implements the ai-triage subcommand. Provider labels and batch
// accuracy are written to stderr, leaving stdout as JSON suggestions suitable
// for piping to another command.
func RunCLI(args []string, stdout, stderr io.Writer, getenv func(string) string) error {
	flags := flag.NewFlagSet("ai-triage", flag.ContinueOnError)
	flags.SetOutput(stderr)
	filePath := flags.String("file", "", "intake item JSON file")
	housesPath := flags.String("houses", "", "houses JSON file")
	templatesPath := flags.String("templates", "", "text templates JSON file")
	limit := flags.Int("n", 1, "maximum intake items to process")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*filePath) == "" {
		return errors.New("ai-triage: -file is required")
	}
	if *limit <= 0 {
		return errors.New("ai-triage: -n must be positive")
	}
	suggester, err := NewFromEnv(getenv)
	if err != nil {
		return err
	}
	if suggester == nil {
		return ErrUnavailable
	}
	items, err := readIntakeItems(*filePath)
	if err != nil {
		return err
	}
	houses, err := readHouseHints(*housesPath)
	if err != nil {
		return err
	}
	templates, err := readTemplateHints(*templatesPath)
	if err != nil {
		return err
	}
	if *limit < len(items) {
		items = items[:*limit]
	}
	fmt.Fprintf(stderr, "AI-Anbieter: %s\n", suggester.Label())
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	categoryMatches, priorityMatches, truthCount := 0, 0, 0
	for _, item := range items {
		receivedAt, err := parseCLIReceivedAt(item.ReceivedAt)
		if err != nil {
			return fmt.Errorf("ai-triage: item %q: %w", item.ID, err)
		}
		categories := item.Categories
		if len(categories) == 0 {
			categories = categoriesFromTemplates(templates)
		}
		suggestion, suggestErr := suggester.Suggest(context.Background(), TriageInput{
			Organisation: item.Organisation, Source: item.Source, Subject: item.Subject,
			Body: item.Body, FromName: item.FromName, FromEmail: item.FromEmail,
			FromPhone: item.FromPhone, ReceivedAt: receivedAt, Houses: houses,
			Categories: categories, Templates: templates, Assignees: item.Assignees,
		})
		if suggestErr != nil && !errors.Is(suggestErr, ErrUncertain) {
			return fmt.Errorf("ai-triage: item %q: %w", item.ID, suggestErr)
		}
		if err := encoder.Encode(suggestion); err != nil {
			return fmt.Errorf("ai-triage: write suggestion: %w", err)
		}
		if errors.Is(suggestErr, ErrUncertain) {
			fmt.Fprintf(stderr, "%s: unsicher\n", displayItemID(item.ID))
		}
		if item.Truth != nil {
			truthCount++
			if suggestion.Category == item.Truth.Category {
				categoryMatches++
			}
			if suggestion.Priority == item.Truth.Priority {
				priorityMatches++
			}
		}
	}
	if truthCount > 0 {
		fmt.Fprintln(stderr, "Trefferquote (Wahrheit vorhanden)")
		fmt.Fprintf(stderr, "Kategorie\t%d/%d\t%.0f%%\n", categoryMatches, truthCount, percent(categoryMatches, truthCount))
		fmt.Fprintf(stderr, "Priorität\t%d/%d\t%.0f%%\n", priorityMatches, truthCount, percent(priorityMatches, truthCount))
	}
	return nil
}

func readIntakeItems(path string) ([]cliIntakeItem, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ai-triage: read intake file: %w", err)
	}
	var items []cliIntakeItem
	if err := json.Unmarshal(data, &items); err == nil {
		if len(items) == 0 {
			return nil, errors.New("ai-triage: intake array is empty")
		}
		return items, nil
	}
	var item cliIntakeItem
	if err := json.Unmarshal(data, &item); err != nil {
		return nil, fmt.Errorf("ai-triage: decode intake file: %w", err)
	}
	return []cliIntakeItem{item}, nil
}

func readHouseHints(path string) ([]HouseHint, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ai-triage: read houses file: %w", err)
	}
	var fileHints []cliHouseHint
	if err := json.Unmarshal(data, &fileHints); err != nil {
		return nil, fmt.Errorf("ai-triage: decode houses file: %w", err)
	}
	hints := make([]HouseHint, 0, len(fileHints))
	for _, fileHint := range fileHints {
		hint := HouseHint{Slug: fileHint.Slug, Name: fileHint.Name, Address: fileHint.Address}
		for _, unit := range fileHint.Units {
			hint.Units = append(hint.Units, unit.Label)
		}
		hints = append(hints, hint)
	}
	return hints, nil
}

func readTemplateHints(path string) ([]TemplateHint, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ai-triage: read templates file: %w", err)
	}
	var hints []TemplateHint
	if err := json.Unmarshal(data, &hints); err != nil {
		return nil, fmt.Errorf("ai-triage: decode templates file: %w", err)
	}
	return hints, nil
}

func parseCLIReceivedAt(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("received_at must be RFC3339: %w", err)
	}
	return parsed, nil
}

func categoriesFromTemplates(templates []TemplateHint) []CategoryRule {
	seen := make(map[string]struct{})
	var categories []CategoryRule
	for _, template := range templates {
		key := strings.TrimSpace(template.Category)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		categories = append(categories, CategoryRule{Key: key, Label: key})
	}
	return categories
}

func displayItemID(id string) string {
	if id == "" {
		return "Eintrag"
	}
	return id
}

func percent(matches, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(matches) / float64(total) * 100
}
