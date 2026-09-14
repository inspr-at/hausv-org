// verify-release is the offline gate for every supported build entry point.
package main

import (
	"encoding/json"
	"fmt"
	"github.com/inspr-at/hausv-org/internal/version"
	"github.com/inspr-at/hausv-org/internal/versionbundle"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	if err := verify(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("Release identity and complete offline presentation verified")
}
func verify() error {
	if err := versionbundle.Verify(os.DirFS("internal/web"), versionbundle.Directory); err != nil {
		return err
	}
	dir := "internal/web/" + versionbundle.Directory
	// Git archives and Docker source contexts contain no Git database. Exact
	// independent digest pins remain mandatory in those offline source packages.
	if _, err := os.Stat(".git"); err == nil {
		entries, _ := os.ReadDir(dir)
		for _, entry := range entries {
			if err := exec.Command("git", "ls-files", "--error-unmatch", "--", filepath.Join(dir, entry.Name())).Run(); err != nil {
				return fmt.Errorf("untracked presentation payload: %s", entry.Name())
			}
		}
	}
	raw, err := os.ReadFile("VERSION")
	if err != nil {
		return err
	}
	value := strings.TrimSuffix(string(raw), "\n")
	if _, err := version.ParseCalendar(value); err != nil {
		return err
	}
	if version.Notes()[0].Version != value || version.Notes()[0].Scheme != version.Scheme {
		return fmt.Errorf("release notes and VERSION differ")
	}
	record, err := os.ReadFile("internal/version/release.json")
	if err != nil {
		return err
	}
	var release struct {
		Version  string `json:"version"`
		Scheme   string `json:"version_scheme"`
		Sequence uint64 `json:"release_sequence"`
	}
	if err := json.Unmarshal(record, &release); err != nil {
		return err
	}
	if release.Version != value || release.Scheme != version.Scheme || release.Sequence != version.ReleaseSequence {
		return fmt.Errorf("release record differs from source")
	}
	if requested := os.Getenv("APP_VERSION"); requested != "" && requested != value {
		return fmt.Errorf("APP_VERSION must equal canonical VERSION")
	}
	return nil
}
