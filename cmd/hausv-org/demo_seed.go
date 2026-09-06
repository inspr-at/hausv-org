package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/demo"
	"github.com/inspr-at/hausv-org/internal/store"
)

func runDemoSeed(args []string, stdout, stderr io.Writer, getenv func(string) string) error {
	flags := flag.NewFlagSet("demo-seed", flag.ContinueOnError)
	anchor := flags.String("anchor", "today", "shift fixture dates so the demo day lands on this day: today, none, or YYYY-MM-DD")
	flags.SetOutput(stderr)
	dir := flags.String("dir", "", "directory containing the demo seed JSON files")
	reset := flags.Bool("reset", false, "remove fixture-owned rows before loading")
	stats := flags.Bool("stats", false, "print intake counts by status and category")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || strings.TrimSpace(*dir) == "" {
		return fmt.Errorf("usage: hausv-org demo-seed -dir PATH [-reset] [-stats]")
	}
	backend := db.Backend(strings.ToLower(strings.TrimSpace(getenv("DB_BACKEND"))))
	dsn := strings.TrimSpace(getenv("DB_PATH"))
	if backend == db.BackendPostgres {
		dsn = strings.TrimSpace(getenv("DATABASE_URL"))
	} else if dsn == "" {
		if parking := strings.TrimSpace(getenv("PARKING_DATA_PATH")); parking != "" {
			dsn = filepath.Join(filepath.Dir(parking), "hausv.db")
		}
	}
	if dsn == "" {
		return fmt.Errorf("DB_PATH is required (or DATABASE_URL with DB_BACKEND=postgres)")
	}
	database, err := db.OpenConfig(context.Background(), db.Config{Backend: backend, DSN: dsn})
	if err != nil {
		return err
	}
	defer database.Close()
	anchorTime, err := parseSeedAnchor(*anchor)
	if err != nil {
		return err
	}
	documentDir := strings.TrimSpace(getenv("DOC_FILE_DIR"))
	if documentDir == "" {
		documentDataPath := strings.TrimSpace(getenv("DOC_DATA_PATH"))
		if documentDataPath == "" {
			documentDataPath = "tmp/documents.json"
		}
		documentDir = filepath.Join(filepath.Dir(documentDataPath), "documents")
	}
	options := demo.SeedOptions{Reset: *reset, Stats: *stats, Out: stdout, Anchor: anchorTime, DocumentDir: documentDir}
	if unitPath := strings.TrimSpace(getenv("UNIT_DATA_PATH")); unitPath != "" {
		units, err := store.NewUnitStore(unitPath)
		if err != nil {
			return err
		}
		options.Units = units
	}
	_, err = demo.Load(context.Background(), database, *dir, options)
	return err
}

func parseSeedAnchor(value string) (time.Time, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "today", "now":
		return time.Now(), nil
	case "none", "fixed":
		return time.Time{}, nil
	}
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, fmt.Errorf("demo-seed: -anchor must be today, none or YYYY-MM-DD: %w", err)
	}
	return parsed, nil
}
