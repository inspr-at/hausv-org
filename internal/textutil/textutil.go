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

// UnitID applies the canonical, deliberately Unicode-preserving unit key
// normalization shared by the administrative and energy stores.
func UnitID(raw string) string {
	raw = Slug(raw)
	raw = strings.Join(strings.Fields(raw), "-")
	return strings.ReplaceAll(raw, "/", "-")
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

// Email normalizes an address for comparison: lowercase, trimmed.
func Email(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

// Truncate trims and cuts to a rune limit (not a byte limit).
func Truncate(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}
