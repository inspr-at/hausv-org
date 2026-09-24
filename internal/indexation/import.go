package indexation

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// ImportOptions requires a verified final-through month: the OGD CSV does not
// encode final/preliminary status. Unknown finality must never imply finality.
type ImportOptions struct {
	Series       Series
	Source       string
	FetchedAt    time.Time
	FinalThrough Month
}

type Imported struct {
	Values []IndexValue
	Annual []AnnualValue
}

// ImportOGD parses Statistik Austria's semicolon-separated OGD VPI CSV (legacy
// COICOP or COICOP18), selecting only the total index. Annual means stay separate.
// It does not fetch URLs or infer publication dates from the current clock.
func ImportOGD(r io.Reader, opts ImportOptions) (Imported, error) {
	if !opts.Series.Valid() || opts.Source == "" || opts.FetchedAt.IsZero() || !opts.FinalThrough.Valid() || opts.FinalThrough > monthOf(opts.FetchedAt) {
		return Imported{}, fmt.Errorf("series, source, retrieval date and verified final-through month required")
	}
	reader := csv.NewReader(r)
	reader.Comma = ';'
	header, err := reader.Read()
	if err != nil {
		return Imported{}, fmt.Errorf("OGD header: %w", err)
	}
	columns := make(map[string]int)
	for i, h := range header {
		h = strings.TrimPrefix(h, "\ufeff")
		if _, exists := columns[h]; exists {
			return Imported{}, fmt.Errorf("duplicate OGD column %s", h)
		}
		columns[h] = i
	}
	period, periodOK := columns["C-VPIZR-0"]
	value, valueOK := columns["F-VPIMZBM"]
	dimension, dimensionCount := 0, 0
	for _, key := range []string{"C-VPI1-0", "C-VPI5-0", "C-VPI5NEU-0", "C-VPICOICOP18_5-0"} {
		if pos, ok := columns[key]; ok {
			dimension, dimensionCount = pos, dimensionCount+1
		}
	}
	if !periodOK || !valueOK || dimensionCount != 1 {
		return Imported{}, fmt.Errorf("unsupported OGD columns")
	}
	var result Imported
	seen := make(map[string]bool)
	for line := 2; ; line++ {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return Imported{}, fmt.Errorf("OGD row %d: %w", line, err)
		}
		if row[dimension] != "VPI-0" && row[dimension] != "VPICOICOP18-0" {
			continue
		}
		code := row[period]
		if !strings.HasPrefix(code, "VPIZR-") || seen[code] {
			return Imported{}, fmt.Errorf("invalid or duplicate period at row %d", line)
		}
		seen[code] = true
		code = strings.TrimPrefix(code, "VPIZR-")
		n, err := ParseDecimal(row[value])
		if err != nil || n <= 0 {
			return Imported{}, fmt.Errorf("invalid index value at row %d", line)
		}
		switch len(code) {
		case 4:
			year, err := strconv.Atoi(code)
			if err != nil || year < 1 || year > opts.FetchedAt.Year() {
				return Imported{}, fmt.Errorf("invalid annual period at row %d", line)
			}
			result.Annual = append(result.Annual, AnnualValue{Series: opts.Series, Year: year, Value: n, Preliminary: Month(code+"-12") > opts.FinalThrough, Source: opts.Source, FetchedAt: opts.FetchedAt})
		case 6:
			month, err := ParseMonth(code[:4] + "-" + code[4:])
			if err != nil || month > monthOf(opts.FetchedAt) {
				return Imported{}, fmt.Errorf("invalid monthly period at row %d", line)
			}
			result.Values = append(result.Values, IndexValue{Series: opts.Series, Month: month, Value: n, Preliminary: month > opts.FinalThrough, Source: opts.Source, FetchedAt: opts.FetchedAt})
		default:
			return Imported{}, fmt.Errorf("unsupported period at row %d", line)
		}
	}
	if len(result.Values) == 0 {
		return Imported{}, fmt.Errorf("OGD has no monthly total-index values")
	}
	return result, nil
}
