// Package versionbundle verifies the complete, immutable offline presentation.
package versionbundle

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
)

const Revision = "0cdcd1bb9b899d6f316a43563f824aae86075569"
const ConfigSHA256 = "7843f3515ce329277d2d576000bd60ac410d725b241d502a9a3fecb2533d956d"
const ManifestSHA256 = "b02ba5605831ded80f8c1c191a9ef6e8a8c3e470eb1aeffc31056d76e5f48e32"
const Directory = "assets/versioning"

type Manifest struct {
	Repository, Revision, ExpectedConfigSHA256, Schema string
	Files                                              []struct {
		OutputPath, SHA256 string
		Size               int
	}
}

func Verify(files fs.FS, directory string) error {
	raw, err := fs.ReadFile(files, path.Join(directory, "manifest.json"))
	if err != nil {
		return err
	}
	if fmt.Sprintf("%x", sha256.Sum256(raw)) != ManifestSHA256 {
		return fmt.Errorf("presentation manifest pin mismatch")
	}
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return err
	}
	if m.Repository != "inspr-at/inspr" || m.Revision != Revision || m.ExpectedConfigSHA256 != ConfigSHA256 || m.Schema != "inspr.calendar-version-display.v2" {
		return fmt.Errorf("presentation provenance mismatch")
	}
	expected := map[string]bool{"manifest.json": true}
	for _, file := range m.Files {
		if path.Base(file.OutputPath) != file.OutputPath || expected[file.OutputPath] {
			return fmt.Errorf("invalid presentation path")
		}
		expected[file.OutputPath] = true
		data, err := fs.ReadFile(files, path.Join(directory, file.OutputPath))
		if err != nil {
			return err
		}
		if len(data) != file.Size || fmt.Sprintf("%x", sha256.Sum256(data)) != file.SHA256 {
			return fmt.Errorf("presentation payload mismatch: %s", file.OutputPath)
		}
	}
	entries, err := fs.ReadDir(files, directory)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !entry.Type().IsRegular() || !expected[entry.Name()] {
			return fmt.Errorf("unexpected presentation payload: %s", entry.Name())
		}
	}
	if len(entries) != len(expected) {
		return fmt.Errorf("incomplete presentation bundle")
	}
	return nil
}
