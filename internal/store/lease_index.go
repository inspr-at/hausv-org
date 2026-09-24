package store

import (
	"strings"
	"sync"

	"github.com/inspr-at/hausv-org/internal/indexation"
)

var publishedIndices = sync.OnceValues(indexation.LoadSnapshot)

// PublishedIndexValue accepts only published, final values of the named base.
func PublishedIndexValue(series, period string) (string, bool) {
	snapshot, err := publishedIndices()
	if err != nil {
		return "", false
	}
	value, found, err := snapshot.Data.Lookup(indexation.Series(strings.ToUpper(series)), indexation.Month(period))
	if err != nil || !found || value.Preliminary || value.ChainSource != "" {
		return "", false
	}
	return value.Value.String(), true
}
