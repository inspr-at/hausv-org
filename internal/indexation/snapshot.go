package indexation

import (
	"crypto/sha256"
	"embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

//go:embed data/vpi.csv data/manifest.json
var snapshotFiles embed.FS

type SnapshotSource struct {
	Series Series `json:"series"`
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

type Manifest struct {
	Version      string           `json:"version"`
	RetrievedAt  time.Time        `json:"retrieved_at"`
	FinalThrough Month            `json:"final_through"`
	StatusSource string           `json:"status_source"`
	License      string           `json:"license"`
	LicenseURL   string           `json:"license_url"`
	Attribution  string           `json:"attribution"`
	Changes      string           `json:"changes"`
	DataSHA256   string           `json:"data_sha256"`
	Sources      []SnapshotSource `json:"sources"`
	Factors      []ChainFactor    `json:"factors"`
}

type Snapshot struct {
	Manifest Manifest
	Data     Dataset
	Annual   []AnnualValue
}

// LoadSnapshot returns a fresh, independently owned copy of the committed data.
// asOf in Evaluate means observation cutoff in this data vintage, not a replay
// of what had been published on a historical date.
func LoadSnapshot() (Snapshot, error) {
	var s Snapshot
	b, err := snapshotFiles.ReadFile("data/manifest.json")
	if err != nil {
		return s, err
	}
	if err = json.Unmarshal(b, &s.Manifest); err != nil {
		return s, err
	}
	b, err = snapshotFiles.ReadFile("data/vpi.csv")
	if err != nil {
		return s, err
	}
	if fmt.Sprintf("%x", sha256.Sum256(b)) != s.Manifest.DataSHA256 {
		return s, fmt.Errorf("snapshot checksum mismatch")
	}
	sources := make(map[Series]string)
	for _, source := range s.Manifest.Sources {
		sources[source.Series] = source.URL
	}
	r := csv.NewReader(strings.NewReader(string(b)))
	header, err := r.Read()
	if err != nil || strings.Join(header, ",") != "series,period,value,preliminary" {
		return s, fmt.Errorf("invalid snapshot header")
	}
	var values []IndexValue
	seen := make(map[string]bool)
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return s, err
		}
		series, period := Series(row[0]), row[1]
		value, err := ParseDecimal(row[2])
		if err != nil || value <= 0 || !series.Valid() || sources[series] == "" || seen[row[0]+period] {
			return s, fmt.Errorf("invalid snapshot observation")
		}
		seen[row[0]+period] = true
		preliminary, err := strconv.ParseBool(row[3])
		if err != nil {
			return s, err
		}
		if len(period) == 4 {
			year, err := strconv.Atoi(period)
			if err != nil || year < 1 {
				return s, fmt.Errorf("invalid snapshot year")
			}
			s.Annual = append(s.Annual, AnnualValue{Series: series, Year: year, Value: value, Preliminary: preliminary, Source: sources[series], FetchedAt: s.Manifest.RetrievedAt})
		} else {
			values = append(values, IndexValue{Series: series, Month: Month(period), Value: value, Preliminary: preliminary, Source: sources[series], FetchedAt: s.Manifest.RetrievedAt})
		}
	}
	s.Data, err = NewDataset(values, s.Manifest.Factors)
	return s, err
}
