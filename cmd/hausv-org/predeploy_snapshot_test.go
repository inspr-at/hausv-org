package main

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestCreatePredeploySnapshot(t *testing.T) {
	base := t.TempDir()
	source := filepath.Join(base, "source")
	root := filepath.Join(base, "snapshots")
	if err := os.Mkdir(source, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(source, "blobs"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "blobs", "document.txt"), []byte("fixture"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("document.txt", filepath.Join(source, "blobs", "latest")); err != nil {
		t.Fatal(err)
	}
	database, err := sql.Open("sqlite", filepath.Join(source, "hausv.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec("PRAGMA journal_mode=WAL; CREATE TABLE fixture(value TEXT); INSERT INTO fixture VALUES ('before')"); err != nil {
		t.Fatal(err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
	// Reproduce production's normal clean-stop state: the database header stays
	// WAL-mode while close removes the sidecars, and the source mount is read-only.
	if _, err := os.Stat(filepath.Join(source, "hausv.db-wal")); !os.IsNotExist(err) {
		t.Fatalf("clean close left unexpected WAL sidecar: %v", err)
	}
	if err := os.Chmod(source, 0o555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(source, 0o750)

	snapshot := filepath.Join(root, "0.99.11-aaaaaaa")
	stale := filepath.Join(root, ".0.99.11-aaaaaaa.staging")
	if err := os.Mkdir(stale, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stale, "partial"), []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	created := time.Date(2026, 8, 27, 10, 11, 12, 987, time.UTC)
	config := predeploySnapshotConfig{
		source: source, snapshot: snapshot,
		sourceVersion: "0.99.10", sourceCommit: "bbbbbbb",
		targetVersion: "0.99.11", targetCommit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	actual, err := createPredeploySnapshot(config, created)
	if err != nil {
		t.Fatal(err)
	}
	if want := created.Truncate(time.Second); actual != want {
		t.Fatalf("created = %s, want %s", actual, want)
	}
	if mode := mustStat(t, root).Mode().Perm(); mode != 0o700 {
		t.Fatalf("snapshot root mode = %o, want 700", mode)
	}
	if mode := mustStat(t, snapshot).Mode().Perm(); mode != 0o700 {
		t.Fatalf("snapshot mode = %o, want 700", mode)
	}
	content, err := os.ReadFile(filepath.Join(snapshot, "blobs", "document.txt"))
	if err != nil || string(content) != "fixture" {
		t.Fatalf("blob copy = %q, %v", content, err)
	}
	link, err := os.Readlink(filepath.Join(snapshot, "blobs", "latest"))
	if err != nil || link != "document.txt" {
		t.Fatalf("symlink = %q, %v", link, err)
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		if _, err := os.Stat(filepath.Join(snapshot, "hausv.db"+suffix)); !os.IsNotExist(err) {
			t.Fatalf("checkpointed snapshot retained %s sidecar: %v", suffix, err)
		}
	}
	check, err := sql.Open("sqlite", filepath.Join(snapshot, "hausv.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer check.Close()
	var value string
	if err := check.QueryRow("SELECT value FROM fixture").Scan(&value); err != nil || value != "before" {
		t.Fatalf("snapshot value = %q, %v", value, err)
	}
	metadataBytes, err := os.ReadFile(filepath.Join(snapshot, "PREDEPLOY-SNAPSHOT.json"))
	if err != nil {
		t.Fatal(err)
	}
	var metadata predeploySnapshotMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata.CreatedUTC != "2026-08-27T10:11:12Z" || metadata.SQLite != "ok" || metadata.TargetVersion != "0.99.11" {
		t.Fatalf("unexpected metadata: %+v", metadata)
	}
	if _, err := createPredeploySnapshot(config, created); err == nil {
		t.Fatal("second snapshot unexpectedly overwrote the recovery point")
	}
}

func TestCreatePredeploySnapshotRejectsNestedPaths(t *testing.T) {
	base := t.TempDir()
	config := predeploySnapshotConfig{
		source: base, snapshot: filepath.Join(base, "snapshot"),
		sourceVersion: "0.99.10", sourceCommit: "bbbbbbb",
		targetVersion: "0.99.11", targetCommit: "aaaaaaa",
	}
	if _, err := createPredeploySnapshot(config, time.Now()); err == nil {
		t.Fatal("nested snapshot path unexpectedly accepted")
	}
}

func TestCreatePredeploySnapshotCheckpointsCommittedWAL(t *testing.T) {
	base := t.TempDir()
	active := filepath.Join(base, "active")
	source := filepath.Join(base, "quiesced")
	root := filepath.Join(base, "snapshots")
	for _, directory := range []string{active, source, root} {
		if err := os.Mkdir(directory, 0o750); err != nil {
			t.Fatal(err)
		}
	}
	activeDatabase := filepath.Join(active, "hausv.db")
	database, err := sql.Open("sqlite", activeDatabase)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec("PRAGMA journal_mode=WAL; PRAGMA wal_autocheckpoint=0; CREATE TABLE fixture(value TEXT); INSERT INTO fixture VALUES ('committed in wal')"); err != nil {
		t.Fatal(err)
	}
	// Freeze the database plus live sidecars as if the process had been killed
	// after its stop timeout. The resulting set receives no further writes.
	for _, suffix := range []string{"", "-wal", "-shm"} {
		content, err := os.ReadFile(activeDatabase + suffix)
		if err != nil {
			t.Fatalf("read active SQLite%s: %v", suffix, err)
		}
		if err := os.WriteFile(filepath.Join(source, "hausv.db"+suffix), content, 0o640); err != nil {
			t.Fatal(err)
		}
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(source, 0o555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(source, 0o750)

	snapshot := filepath.Join(root, "0.99.11-aaaaaaa")
	config := predeploySnapshotConfig{
		source: source, snapshot: snapshot,
		sourceVersion: "0.99.10", sourceCommit: "bbbbbbb",
		targetVersion: "0.99.11", targetCommit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	if _, err := createPredeploySnapshot(config, time.Now()); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		if _, err := os.Stat(filepath.Join(snapshot, "hausv.db"+suffix)); !os.IsNotExist(err) {
			t.Fatalf("checkpointed pending-WAL snapshot retained %s sidecar: %v", suffix, err)
		}
	}
	check, err := sql.Open("sqlite", filepath.Join(snapshot, "hausv.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer check.Close()
	var value string
	if err := check.QueryRow("SELECT value FROM fixture").Scan(&value); err != nil || value != "committed in wal" {
		t.Fatalf("snapshot lost committed WAL row: value=%q err=%v", value, err)
	}
}

func mustStat(t *testing.T, path string) os.FileInfo {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info
}
