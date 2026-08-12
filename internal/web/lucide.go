package web

import (
	"io/fs"
	"sort"
	"strings"
	"sync"
)

var (
	lucideIconsOnce sync.Once
	lucideIconNames []string
	lucideIconSet   map[string]struct{}
)

// LucideIconNames returns every SVG name vendored from the pinned
// lucide-static package. The returned slice is a copy so callers cannot mutate
// the process-wide whitelist used for persisted consumer icons.
func LucideIconNames() []string {
	loadLucideIcons()
	return append([]string(nil), lucideIconNames...)
}

// IsLucideIcon is the single server-side trust boundary for user-selected
// icon names. A value is accepted only when the matching local SVG exists.
func IsLucideIcon(name string) bool {
	loadLucideIcons()
	_, ok := lucideIconSet[strings.TrimSpace(strings.ToLower(name))]
	return ok
}

func loadLucideIcons() {
	lucideIconsOnce.Do(func() {
		lucideIconSet = map[string]struct{}{}
		entries, err := fs.ReadDir(Assets, "assets/icons/lucide")
		if err != nil {
			return
		}
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(name, ".svg") {
				continue
			}
			name = strings.TrimSuffix(name, ".svg")
			lucideIconSet[name] = struct{}{}
			lucideIconNames = append(lucideIconNames, name)
		}
		sort.Strings(lucideIconNames)
	})
}
