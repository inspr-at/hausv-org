// Package textutil holds tiny, dependency-free string helpers shared across
// packages. It sits at the bottom of the dependency graph: it may import
// nothing from this module.
package textutil

import "strings"

// Slug normalizes an identifier: lowercase, trimmed, underscores to hyphens.
// Tenant slugs, unit IDs and document keys are all compared post-Slug, so this
// must stay byte-identical to the original normalizeSlug.
func Slug(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	raw = strings.ReplaceAll(raw, "_", "-")
	return raw
}

// FirstNonEmpty returns the first value that is not blank after trimming.
// Note it returns the ORIGINAL value, not the trimmed one.
func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
