package dbtest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	postgresContractStep = "Verify PostgreSQL store CI contract"
	postgresFullStep     = "Full suite (PostgreSQL)"
)

// TestCIPostgresStoreContract closes HAUSV-757: the required CI job must run
// the full race suite on PostgreSQL and must not keep a SQLite product lane.
func TestCIPostgresStoreContract(t *testing.T) {
	workflowPath := filepath.Join("..", "..", ".github", "workflows", "ci.yml")
	raw, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("read %s: %v", workflowPath, err)
	}
	workflow := string(raw)
	if err := validatePostgresStoreCI(workflow); err != nil {
		t.Fatal(err)
	}

	mutations := map[string]string{
		"guard step removed":             removeWorkflowStep(workflow, postgresContractStep),
		"PostgreSQL step removed":        removeWorkflowStep(workflow, postgresFullStep),
		"PostgreSQL suite narrowed":      replaceInWorkflowStep(workflow, postgresFullStep, "go test -race ./...", "go test -race ./internal/store"),
		"PostgreSQL DSN empty":           replaceTestJobEnv(workflow, "HAUSV_TEST_POSTGRES_DSN: postgres://hausv_ci:hausv_ci@127.0.0.1:5432/hausv_ci?sslmode=disable", `HAUSV_TEST_POSTGRES_DSN: ""`),
		"PostgreSQL required flag empty": replaceTestJobEnv(workflow, `HAUSV_TEST_POSTGRES_REQUIRED: "true"`, `HAUSV_TEST_POSTGRES_REQUIRED: ""`),
		"PostgreSQL selector empty":      replaceTestJobEnv(workflow, `HAUSV_STORE_TEST_POSTGRES: "1"`, `HAUSV_STORE_TEST_POSTGRES: ""`),
		"SQLite store suite revived":     insertAfterContractStep(workflow, sqliteStoreLane),
	}
	for name, mutated := range mutations {
		t.Run(name, func(t *testing.T) {
			if mutated == workflow {
				t.Fatal("mutation did not change the workflow fixture")
			}
			if err := validatePostgresStoreCI(mutated); err == nil {
				t.Fatal("mutated workflow unexpectedly satisfied the PostgreSQL contract")
			}
		})
	}
}

const sqliteStoreLane = `
      - name: Store suite (SQLite)
        env:
          HAUSV_TEST_POSTGRES_DSN: ""
          HAUSV_TEST_POSTGRES_REQUIRED: ""
          HAUSV_STORE_TEST_POSTGRES: ""
        run: go test -race ./internal/store -count=1
`

func validatePostgresStoreCI(workflow string) error {
	testJob, err := workflowSection(workflow, "  test:", "  govulncheck:")
	if err != nil {
		return err
	}
	stepsAt := strings.Index(testJob, "    steps:\n")
	if stepsAt < 0 {
		return fmt.Errorf("CI test job has no steps")
	}
	jobConfig := testJob[:stepsAt]
	if err := requireMappingValue(jobConfig, "HAUSV_TEST_POSTGRES_DSN", func(value string) bool {
		return value != "" && value != `""` && value != "''"
	}); err != nil {
		return fmt.Errorf("PostgreSQL DSN selector: %w", err)
	}
	if err := requireMappingValue(jobConfig, "HAUSV_TEST_POSTGRES_REQUIRED", func(value string) bool { return value == `"true"` }); err != nil {
		return fmt.Errorf("PostgreSQL required selector: %w", err)
	}
	if err := requireMappingValue(jobConfig, "HAUSV_STORE_TEST_POSTGRES", func(value string) bool { return value == `"1"` }); err != nil {
		return fmt.Errorf("PostgreSQL store selector: %w", err)
	}

	guard, err := namedWorkflowStep(testJob, postgresContractStep)
	if err != nil {
		return err
	}
	if !containsLine(guard, `run: go test ./internal/dbtest -run '^TestCIPostgresStoreContract$' -count=1`) {
		return fmt.Errorf("%q must run the exact contract oracle", postgresContractStep)
	}

	if _, err := namedWorkflowStep(testJob, "Store suite (SQLite)"); err == nil {
		return fmt.Errorf("SQLite store suite must not remain a CI product job")
	}

	postgres, err := namedWorkflowStep(testJob, postgresFullStep)
	if err != nil {
		return err
	}
	if !containsLine(postgres, "run: go test -race ./...") {
		return fmt.Errorf("%q must run the full PostgreSQL race suite", postgresFullStep)
	}
	for _, key := range []string{"HAUSV_TEST_POSTGRES_DSN", "HAUSV_TEST_POSTGRES_REQUIRED", "HAUSV_STORE_TEST_POSTGRES"} {
		if containsMappingKey(postgres, key) {
			return fmt.Errorf("%q must inherit the required PostgreSQL %s selector", postgresFullStep, key)
		}
	}
	return nil
}

func insertAfterContractStep(workflow, extra string) string {
	step, err := namedWorkflowStep(workflow, postgresContractStep)
	if err != nil {
		return workflow
	}
	return strings.Replace(workflow, step+"\n", step+"\n"+extra, 1)
}

func workflowSection(workflow, start, end string) (string, error) {
	startAt := strings.Index(workflow, start+"\n")
	if startAt < 0 {
		return "", fmt.Errorf("workflow section %q is missing", strings.TrimSpace(start))
	}
	endAt := strings.Index(workflow[startAt+len(start)+1:], end+"\n")
	if endAt < 0 {
		return "", fmt.Errorf("workflow section %q has no %q boundary", strings.TrimSpace(start), strings.TrimSpace(end))
	}
	endAt += startAt + len(start) + 1
	return workflow[startAt:endAt], nil
}

func namedWorkflowStep(job, name string) (string, error) {
	needle := "      - name: " + name
	lines := strings.Split(job, "\n")
	start := -1
	for index, line := range lines {
		if line == needle {
			if start >= 0 {
				return "", fmt.Errorf("workflow step %q appears more than once", name)
			}
			start = index
		}
	}
	if start < 0 {
		return "", fmt.Errorf("workflow step %q is missing", name)
	}
	end := len(lines)
	for index := start + 1; index < len(lines); index++ {
		if strings.HasPrefix(lines[index], "      - ") {
			end = index
			break
		}
	}
	return strings.Join(lines[start:end], "\n"), nil
}

func requireMappingValue(block, key string, valid func(string) bool) error {
	prefix := key + ":"
	found := 0
	value := ""
	for _, line := range strings.Split(block, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, prefix) {
			continue
		}
		found++
		value = strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
	}
	if found != 1 {
		return fmt.Errorf("%s appears %d times, want exactly once", key, found)
	}
	if !valid(value) {
		return fmt.Errorf("%s has disallowed value %q", key, value)
	}
	return nil
}

func containsLine(block, want string) bool {
	for _, line := range strings.Split(block, "\n") {
		if strings.TrimSpace(line) == want {
			return true
		}
	}
	return false
}

func containsMappingKey(block, key string) bool {
	prefix := key + ":"
	for _, line := range strings.Split(block, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), prefix) {
			return true
		}
	}
	return false
}

func removeWorkflowStep(workflow, name string) string {
	step, err := namedWorkflowStep(workflow, name)
	if err != nil {
		return workflow
	}
	return strings.Replace(workflow, step+"\n", "", 1)
}

func replaceInWorkflowStep(workflow, name, old, replacement string) string {
	step, err := namedWorkflowStep(workflow, name)
	if err != nil || !strings.Contains(step, old) {
		return workflow
	}
	changed := strings.Replace(step, old, replacement, 1)
	return strings.Replace(workflow, step, changed, 1)
}

func replaceTestJobEnv(workflow, old, replacement string) string {
	testJob, err := workflowSection(workflow, "  test:", "  govulncheck:")
	if err != nil {
		return workflow
	}
	stepsAt := strings.Index(testJob, "    steps:\n")
	if stepsAt < 0 || !strings.Contains(testJob[:stepsAt], old) {
		return workflow
	}
	changed := strings.Replace(testJob[:stepsAt], old, replacement, 1) + testJob[stepsAt:]
	return strings.Replace(workflow, testJob, changed, 1)
}
