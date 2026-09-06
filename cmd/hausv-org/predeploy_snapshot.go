package main

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// postgresDumpName is the PostgreSQL half of a recovery point: a pg_dump in
// custom format, taken while the service was stopped, next to hausv.db and
// the blobs. Production runs PostgreSQL (HAUSV-562); a recovery point without
// it would let an operator restore files against a migrated database.
const postgresDumpName = "postgres.pgdump"

// postgresDumpMagic opens every custom-format pg_dump archive.
var postgresDumpMagic = []byte("PGDMP")

type predeploySnapshotConfig struct {
	source        string
	snapshot      string
	sourceVersion string
	sourceCommit  string
	targetVersion string
	targetCommit  string
	// postgresDumpBytes > 0 announces a PostgreSQL dump of exactly that many
	// bytes arriving on postgresDump; it is stored as postgres.pgdump. Zero
	// means the host runs SQLite only and no dump is expected.
	postgresDumpBytes int64
	postgresDump      io.Reader
}

type predeploySnapshotMetadata struct {
	CreatedUTC         string `json:"created_utc"`
	Scope              string `json:"scope"`
	SourceCommit       string `json:"source_commit"`
	SourceVersion      string `json:"source_version"`
	SQLite             string `json:"sqlite_integrity"`
	TargetCommit       string `json:"target_commit"`
	TargetVersion      string `json:"target_version"`
	PostgresDump       string `json:"postgres_dump,omitempty"`
	PostgresDumpBytes  int64  `json:"postgres_dump_bytes,omitempty"`
	PostgresDumpSHA256 string `json:"postgres_dump_sha256,omitempty"`
}

type predeploySnapshotResult struct {
	Created            time.Time
	PostgresDumpSHA256 string
}

func runPredeploySnapshot(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("predeploy-snapshot", flag.ContinueOnError)
	flags.SetOutput(stderr)
	config := predeploySnapshotConfig{postgresDump: stdin}
	flags.StringVar(&config.source, "source", "", "read-only HAUSV data directory")
	flags.StringVar(&config.snapshot, "snapshot", "", "write-once snapshot directory")
	flags.StringVar(&config.sourceVersion, "source-version", "", "live source version")
	flags.StringVar(&config.sourceCommit, "source-commit", "", "live source commit")
	flags.StringVar(&config.targetVersion, "target-version", "", "target version")
	flags.StringVar(&config.targetCommit, "target-commit", "", "target commit")
	flags.Int64Var(&config.postgresDumpBytes, "postgres-dump-bytes", 0, "size of the custom-format pg_dump arriving on stdin; 0 for a SQLite-only host")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	if os.Geteuid() != 0 {
		return fmt.Errorf("snapshot command requires container UID 0")
	}
	result, err := createPredeploySnapshot(config, time.Now().UTC())
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "snapshot-path=%s\n", config.snapshot)
	fmt.Fprintf(stdout, "snapshot-created=%s\n", result.Created.Format(time.RFC3339))
	fmt.Fprintln(stdout, "snapshot-integrity=ok")
	if config.postgresDumpBytes > 0 {
		fmt.Fprintf(stdout, "postgres-dump=%s\n", filepath.Join(config.snapshot, postgresDumpName))
		fmt.Fprintf(stdout, "postgres-dump-bytes=%d\n", config.postgresDumpBytes)
		fmt.Fprintf(stdout, "postgres-dump-sha256=%s\n", result.PostgresDumpSHA256)
	}
	return nil
}

func createPredeploySnapshot(config predeploySnapshotConfig, created time.Time) (predeploySnapshotResult, error) {
	var result predeploySnapshotResult
	source, err := checkedSnapshotPath(config.source, "source")
	if err != nil {
		return result, err
	}
	snapshot, err := checkedSnapshotPath(config.snapshot, "snapshot")
	if err != nil {
		return result, err
	}
	if source == snapshot || pathWithin(source, snapshot) || pathWithin(snapshot, source) {
		return result, fmt.Errorf("source and snapshot paths must be separate")
	}
	if config.postgresDumpBytes < 0 {
		return result, fmt.Errorf("PostgreSQL dump size must not be negative")
	}
	if config.postgresDumpBytes > 0 && config.postgresDump == nil {
		return result, fmt.Errorf("PostgreSQL dump was announced but no input is available")
	}
	for name, value := range map[string]string{
		"source version": config.sourceVersion,
		"source commit":  config.sourceCommit,
		"target version": config.targetVersion,
		"target commit":  config.targetCommit,
	} {
		if strings.TrimSpace(value) == "" {
			return result, fmt.Errorf("%s is required", name)
		}
	}
	database := filepath.Join(source, "hausv.db")
	if info, statErr := os.Stat(database); statErr != nil || !info.Mode().IsRegular() {
		return result, fmt.Errorf("required HAUSV database is unavailable")
	}
	if _, statErr := os.Stat(snapshot); statErr == nil {
		return result, fmt.Errorf("snapshot path already exists; the existing recovery point was preserved")
	} else if !os.IsNotExist(statErr) {
		return result, fmt.Errorf("inspect snapshot path: %w", statErr)
	}
	parent := filepath.Dir(snapshot)
	if info, statErr := os.Stat(parent); statErr != nil || !info.IsDir() {
		return result, fmt.Errorf("snapshot parent is unavailable")
	}
	if err := os.Chmod(parent, 0o700); err != nil {
		return result, fmt.Errorf("secure snapshot parent: %w", err)
	}

	staging := filepath.Join(parent, "."+filepath.Base(snapshot)+".staging")
	// A staging directory is never a published recovery point. A cancelled
	// container can leave it behind, so remove that exact versioned path before
	// recreating it; the write-once published path above remains untouched.
	if err := os.RemoveAll(staging); err != nil {
		return result, fmt.Errorf("remove stale snapshot staging directory: %w", err)
	}
	if err := os.Mkdir(staging, 0o700); err != nil {
		return result, fmt.Errorf("create snapshot staging directory: %w", err)
	}
	published := false
	defer func() {
		if !published {
			_ = os.RemoveAll(staging)
		}
	}()

	if err := copySnapshotTree(source, staging); err != nil {
		return result, err
	}
	if err := checkpointSnapshotSQLite(filepath.Join(staging, "hausv.db")); err != nil {
		return result, err
	}
	metadata := predeploySnapshotMetadata{
		CreatedUTC:    created.Truncate(time.Second).Format(time.RFC3339),
		Scope:         "pre-schema SQLite and blob recovery point",
		SourceCommit:  config.sourceCommit,
		SourceVersion: config.sourceVersion,
		SQLite:        "ok",
		TargetCommit:  config.targetCommit,
		TargetVersion: config.targetVersion,
	}
	if config.postgresDumpBytes > 0 {
		sum, err := writePostgresDump(filepath.Join(staging, postgresDumpName), config.postgresDump, config.postgresDumpBytes)
		if err != nil {
			return result, err
		}
		metadata.Scope = "pre-schema SQLite, blob and PostgreSQL recovery point"
		metadata.PostgresDump = postgresDumpName
		metadata.PostgresDumpBytes = config.postgresDumpBytes
		metadata.PostgresDumpSHA256 = sum
		result.PostgresDumpSHA256 = sum
	}
	encoded, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return result, fmt.Errorf("encode snapshot metadata: %w", err)
	}
	encoded = append(encoded, '\n')
	if err := writeSyncedFile(filepath.Join(staging, "PREDEPLOY-SNAPSHOT.json"), encoded, 0o600); err != nil {
		return result, err
	}
	if err := os.Chmod(staging, 0o700); err != nil {
		return result, fmt.Errorf("secure snapshot: %w", err)
	}
	if err := syncSnapshotDirectories(staging); err != nil {
		return result, err
	}
	if err := os.Rename(staging, snapshot); err != nil {
		return result, fmt.Errorf("publish snapshot: %w", err)
	}
	published = true
	if err := syncDirectory(parent); err != nil {
		return result, err
	}
	result.Created = created.Truncate(time.Second)
	return result, nil
}

// writePostgresDump stores exactly announced bytes of a custom-format pg_dump
// archive and returns its SHA-256. The caller measured the archive inside the
// database container before streaming it here, so a short or long stream is a
// broken transport, not a smaller database — both refuse the recovery point.
func writePostgresDump(destination string, dump io.Reader, announced int64) (string, error) {
	file, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", fmt.Errorf("create PostgreSQL dump: %w", err)
	}
	digest := sha256.New()
	written, copyErr := io.CopyN(io.MultiWriter(file, digest), dump, announced)
	if copyErr != nil && copyErr != io.EOF {
		file.Close()
		return "", fmt.Errorf("store PostgreSQL dump: %w", copyErr)
	}
	if written != announced {
		file.Close()
		return "", fmt.Errorf("PostgreSQL dump ended after %d of %d announced bytes", written, announced)
	}
	var extra [1]byte
	if n, _ := io.ReadFull(dump, extra[:]); n != 0 {
		file.Close()
		return "", fmt.Errorf("PostgreSQL dump is longer than the announced %d bytes", announced)
	}
	if err := verifyPostgresDumpHeader(destination); err != nil {
		file.Close()
		return "", err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func verifyPostgresDumpHeader(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	header := make([]byte, len(postgresDumpMagic))
	if _, err := io.ReadFull(file, header); err != nil || !bytes.Equal(header, postgresDumpMagic) {
		return fmt.Errorf("PostgreSQL dump is not a custom-format pg_dump archive")
	}
	return nil
}

func checkedSnapshotPath(raw, label string) (string, error) {
	if raw == "" || !filepath.IsAbs(raw) {
		return "", fmt.Errorf("%s path must be absolute", label)
	}
	clean := filepath.Clean(raw)
	if clean == string(filepath.Separator) {
		return "", fmt.Errorf("%s path must not be filesystem root", label)
	}
	return clean, nil
}

func pathWithin(path, parent string) bool {
	rel, err := filepath.Rel(parent, path)
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func copySnapshotTree(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		target := filepath.Join(destination, relative)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		switch {
		case entry.Type()&os.ModeSymlink != 0:
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		case entry.IsDir():
			if err := os.Mkdir(target, 0o700); err != nil {
				return err
			}
			return os.Chmod(target, info.Mode().Perm())
		case info.Mode().IsRegular():
			return copySnapshotFile(path, target, info.Mode().Perm())
		default:
			return fmt.Errorf("unsupported special file in HAUSV data: %s", relative)
		}
	})
}

func copySnapshotFile(source, destination string, mode fs.FileMode) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(output, input); err != nil {
		output.Close()
		return err
	}
	if err := output.Sync(); err != nil {
		output.Close()
		return err
	}
	if err := output.Close(); err != nil {
		return err
	}
	return os.Chmod(destination, mode)
}

func checkpointSnapshotSQLite(databasePath string) error {
	// The service is stopped before this command runs, so hausv.db and any WAL
	// sidecars were copied as one quiesced set. Open only the writable copy:
	// opening a clean WAL-mode database directly on the read-only source mount
	// would try to create -wal/-shm and fail even when no sidecars remain.
	database, err := sql.Open("sqlite", databasePath)
	if err != nil {
		return fmt.Errorf("open snapshot SQLite: %w", err)
	}
	database.SetMaxOpenConns(1)
	var busy, logPages, checkpointedPages int
	if err := database.QueryRow("PRAGMA wal_checkpoint(TRUNCATE)").Scan(&busy, &logPages, &checkpointedPages); err != nil {
		database.Close()
		return fmt.Errorf("checkpoint snapshot SQLite: %w", err)
	}
	if busy != 0 {
		database.Close()
		return fmt.Errorf("snapshot SQLite checkpoint remained busy")
	}
	var integrity string
	if err := database.QueryRow("PRAGMA integrity_check").Scan(&integrity); err != nil {
		database.Close()
		return fmt.Errorf("check snapshot SQLite: %w", err)
	}
	if integrity != "ok" {
		database.Close()
		return fmt.Errorf("snapshot integrity check failed")
	}
	if err := database.Close(); err != nil {
		return fmt.Errorf("close snapshot SQLite: %w", err)
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		if err := os.Remove(databasePath + suffix); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove checkpointed SQLite sidecar: %w", err)
		}
	}
	file, err := os.Open(databasePath)
	if err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return nil
}

func writeSyncedFile(path string, content []byte, mode fs.FileMode) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	if _, err := file.Write(content); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

func syncSnapshotDirectories(root string) error {
	directories := make([]string, 0, 8)
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			directories = append(directories, path)
		}
		return nil
	}); err != nil {
		return err
	}
	for index := len(directories) - 1; index >= 0; index-- {
		if err := syncDirectory(directories[index]); err != nil {
			return err
		}
	}
	return nil
}
