package main

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestRunDemoSeed(t *testing.T) {
	var stdout, stderr bytes.Buffer
	dbPath := filepath.Join(t.TempDir(), "demo.db")
	err := runDemoSeed([]string{"-dir", filepath.Join("..", "..", "internal", "demo", "testdata"), "-stats"}, &stdout, &stderr, func(key string) string {
		if key == "DB_PATH" {
			return dbPath
		}
		return ""
	})
	if err != nil {
		t.Fatalf("run demo seed: %v (%s)", err, stderr.String())
	}
	if stdout.Len() == 0 {
		t.Fatal("expected stats output")
	}
}
