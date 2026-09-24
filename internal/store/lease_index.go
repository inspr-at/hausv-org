package store

// PublishedIndexValue is the hook for w-wsi's published-index lookup.
// TODO(w-wsi): call the indexation lookup when that package is on this branch.
// Until then every check returns unchecked and the importer warns
// index_check_unavailable.
func PublishedIndexValue(series, period string) (value string, checked bool) {
	return "", false
}
