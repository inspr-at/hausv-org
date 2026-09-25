package dbtest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestThrowawayPostgresUsesTmpfs fails when a repo script or workflow starts
// the PostgreSQL image with docker run and does not mount the data directory
// on tmpfs. The image declares VOLUME /var/lib/postgresql/data, so a container
// without that mount leaves an anonymous volume behind.
func TestThrowawayPostgresUsesTmpfs(t *testing.T) {
	root := filepath.Join("..", "..")
	var failures []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "node_modules", "doctrine", "testdata":
				return filepath.SkipDir
			}
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		if !scanForPostgresDockerRun(rel) {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		text := string(raw)
		failures = append(failures, postgresDockerRunWithoutTmpfs(rel, text)...)
		failures = append(failures, postgresServiceWithoutTmpfs(rel, text)...)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(failures) > 0 {
		t.Fatalf("throwaway PostgreSQL without --tmpfs /var/lib/postgresql/data:\n%s", strings.Join(failures, "\n"))
	}
}

func scanForPostgresDockerRun(rel string) bool {
	if strings.HasPrefix(rel, ".github/workflows/") && strings.HasSuffix(rel, ".yml") {
		return true
	}
	if strings.HasPrefix(rel, "scripts/") && (strings.HasSuffix(rel, ".sh") || strings.HasSuffix(rel, ".py")) {
		return true
	}
	return false
}

func postgresDockerRunWithoutTmpfs(rel, content string) []string {
	lines := strings.Split(flattenPythonDockerRun(content), "\n")
	var failures []string
	for index := 0; index < len(lines); index++ {
		if !strings.Contains(lines[index], "docker run") && !strings.Contains(lines[index], `"docker", "run"`) {
			continue
		}
		command, end := continuedCommand(lines, index)
		index = end
		if !dockerRunStartsPostgres(command) {
			continue
		}
		if strings.Contains(command, "--tmpfs /var/lib/postgresql/data") {
			continue
		}
		failures = append(failures, fmt.Sprintf("%s: docker run of postgres lacks --tmpfs /var/lib/postgresql/data", rel))
	}
	return failures
}

func continuedCommand(lines []string, start int) (string, int) {
	var parts []string
	index := start
	for ; index < len(lines); index++ {
		line := strings.TrimRight(lines[index], " \t")
		parts = append(parts, strings.TrimSpace(line))
		if !strings.HasSuffix(line, `\`) {
			break
		}
	}
	return strings.Join(parts, " "), index
}

// flattenPythonDockerRun turns a docker argument list into one physical line
// so the same docker-run check covers scripts/snapshot/set-handover-fixture-tokens.py.
func flattenPythonDockerRun(content string) string {
	lines := strings.Split(content, "\n")
	var out []string
	for index := 0; index < len(lines); index++ {
		if !strings.Contains(lines[index], "command = [") {
			out = append(out, lines[index])
			continue
		}
		var items []string
		index++
		for ; index < len(lines); index++ {
			if strings.TrimSpace(lines[index]) == "]" || strings.HasPrefix(strings.TrimSpace(lines[index]), "]") {
				break
			}
			item := strings.TrimSpace(lines[index])
			item = strings.TrimSuffix(item, ",")
			items = append(items, strings.Trim(item, `"`))
		}
		out = append(out, strings.Join(items, " "))
	}
	return strings.Join(out, "\n")
}

func postgresServiceWithoutTmpfs(rel, content string) []string {
	if !strings.Contains(rel, ".github/workflows/") {
		return nil
	}
	if !strings.Contains(content, "image: postgres:") {
		return nil
	}
	section := content
	if start := strings.Index(content, "services:"); start >= 0 {
		section = content[start:]
		if end := strings.Index(section, "\n    steps:"); end >= 0 {
			section = section[:end]
		}
	}
	if strings.Contains(section, "--tmpfs /var/lib/postgresql/data") {
		return nil
	}
	return []string{rel + ": postgres service lacks --tmpfs /var/lib/postgresql/data"}
}

func dockerRunStartsPostgres(command string) bool {
	markers := []string{
		"postgres:",
		"$postgres_image",
		"${postgres_image}",
		"$hausv_postgres_image",
		"${hausv_postgres_image}",
		"POSTGRES_IMAGE",
	}
	for _, marker := range markers {
		if strings.Contains(command, marker) {
			return true
		}
	}
	return false
}
