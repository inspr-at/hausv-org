package main

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
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
	if want := created.Truncate(time.Second); actual.Created != want {
		t.Fatalf("created = %s, want %s", actual.Created, want)
	}
	if actual.PostgresDumpSHA256 != "" {
		t.Fatalf("SQLite-only snapshot reported a PostgreSQL dump: %+v", actual)
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
	if metadata.PostgresDump != "" || metadata.PostgresDumpBytes != 0 || metadata.PostgresDumpSHA256 != "" {
		t.Fatalf("SQLite-only metadata carries PostgreSQL fields: %+v", metadata)
	}
	if _, err := os.Stat(filepath.Join(snapshot, postgresDumpName)); !os.IsNotExist(err) {
		t.Fatalf("SQLite-only snapshot wrote a PostgreSQL dump: %v", err)
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

// quiescedSource builds a read-only HAUSV data directory with one committed row.
func quiescedSource(t *testing.T, base string) string {
	t.Helper()
	source := filepath.Join(base, "source")
	if err := os.Mkdir(source, 0o750); err != nil {
		t.Fatal(err)
	}
	database, err := sql.Open("sqlite", filepath.Join(source, "hausv.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec("CREATE TABLE fixture(value TEXT); INSERT INTO fixture VALUES ('before')"); err != nil {
		t.Fatal(err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(source, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(source, 0o750) })
	return source
}

func fakePostgresDump(size int) []byte {
	dump := make([]byte, size)
	copy(dump, postgresDumpMagic)
	for index := len(postgresDumpMagic); index < size; index++ {
		dump[index] = byte(index*7 + 3)
	}
	return dump
}

// Production runs PostgreSQL: the recovery point must carry the dump next to
// hausv.db, byte-exact, with its digest in the metadata (HAUSV-562).
func TestCreatePredeploySnapshotStoresPostgresDump(t *testing.T) {
	base := t.TempDir()
	source := quiescedSource(t, base)
	root := filepath.Join(base, "snapshots")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	dump := fakePostgresDump(4096)
	snapshot := filepath.Join(root, "1.3.4-aaaaaaa")
	config := predeploySnapshotConfig{
		source: source, snapshot: snapshot,
		sourceVersion: "1.3.3", sourceCommit: "bbbbbbb",
		targetVersion: "1.3.4", targetCommit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		postgresDumpBytes: int64(len(dump)), postgresDump: bytes.NewReader(dump),
	}
	result, err := createPredeploySnapshot(config, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	stored, err := os.ReadFile(filepath.Join(snapshot, postgresDumpName))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stored, dump) {
		t.Fatalf("stored dump differs from the streamed archive (%d vs %d bytes)", len(stored), len(dump))
	}
	if mode := mustStat(t, filepath.Join(snapshot, postgresDumpName)).Mode().Perm(); mode != 0o600 {
		t.Fatalf("dump mode = %o, want 600", mode)
	}
	sum := sha256.Sum256(dump)
	if want := hex.EncodeToString(sum[:]); result.PostgresDumpSHA256 != want {
		t.Fatalf("dump sha256 = %s, want %s", result.PostgresDumpSHA256, want)
	}
	metadataBytes, err := os.ReadFile(filepath.Join(snapshot, "PREDEPLOY-SNAPSHOT.json"))
	if err != nil {
		t.Fatal(err)
	}
	var metadata predeploySnapshotMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata.PostgresDump != postgresDumpName || metadata.PostgresDumpBytes != int64(len(dump)) || metadata.PostgresDumpSHA256 != result.PostgresDumpSHA256 {
		t.Fatalf("metadata does not describe the dump: %+v", metadata)
	}
	if metadata.Scope != "pre-schema SQLite, blob and PostgreSQL recovery point" || metadata.SQLite != "ok" {
		t.Fatalf("unexpected scope: %+v", metadata)
	}
}

// A dump that is shorter than announced, longer than announced, or not a
// pg_dump archive is a broken transport. Nothing may be published from it.
func TestCreatePredeploySnapshotRefusesBrokenPostgresDump(t *testing.T) {
	base := t.TempDir()
	source := quiescedSource(t, base)
	root := filepath.Join(base, "snapshots")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	dump := fakePostgresDump(2048)
	foreign := append([]byte("SQLite"), dump[6:]...)
	cases := []struct {
		name      string
		announced int64
		stream    []byte
		want      string
	}{
		{"short", int64(len(dump)) + 100, dump, "ended after"},
		{"long", int64(len(dump)) - 100, dump, "longer than the announced"},
		{"foreign", int64(len(foreign)), foreign, "not a custom-format pg_dump"},
		{"announced without input", int64(len(dump)), nil, "no input is available"},
	}
	for index, tc := range cases {
		snapshot := filepath.Join(root, "1.3.4-"+string(rune('a'+index))+"aaaaaa")
		config := predeploySnapshotConfig{
			source: source, snapshot: snapshot,
			sourceVersion: "1.3.3", sourceCommit: "bbbbbbb",
			targetVersion: "1.3.4", targetCommit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			postgresDumpBytes: tc.announced,
		}
		if tc.stream != nil {
			config.postgresDump = bytes.NewReader(tc.stream)
		}
		_, err := createPredeploySnapshot(config, time.Now())
		if err == nil || !bytes.Contains([]byte(err.Error()), []byte(tc.want)) {
			t.Fatalf("%s: err = %v, want %q", tc.name, err, tc.want)
		}
		if _, statErr := os.Stat(snapshot); !os.IsNotExist(statErr) {
			t.Fatalf("%s: broken dump published a recovery point: %v", tc.name, statErr)
		}
		if _, statErr := os.Stat(filepath.Join(root, "."+filepath.Base(snapshot)+".staging")); !os.IsNotExist(statErr) {
			t.Fatalf("%s: broken dump left a staging directory: %v", tc.name, statErr)
		}
	}
}
