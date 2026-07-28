package energy

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

type ImportRecord struct {
	ID         string
	TenantSlug string
	Filename   string
	SHA256     string
	Format     string
	Payload    []byte
	ImportedAt time.Time
}

func ParseSmartMeterCSV(reader io.Reader, tenantSlug, filename string, location *time.Location) (ImportRecord, []Interval, error) {
	if location == nil {
		location = time.Local
	}
	raw, err := io.ReadAll(io.LimitReader(reader, 16<<20))
	if err != nil {
		return ImportRecord{}, nil, errors.New("smart meter file could not be read")
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return ImportRecord{}, nil, errors.New("smart meter file is empty")
	}
	delimiter := ','
	firstLine := string(raw)
	if index := strings.IndexByte(firstLine, '\n'); index >= 0 {
		firstLine = firstLine[:index]
	}
	if strings.Count(firstLine, ";") > strings.Count(firstLine, ",") {
		delimiter = ';'
	}
	parser := csv.NewReader(bytes.NewReader(raw))
	parser.Comma = delimiter
	parser.TrimLeadingSpace = true
	parser.FieldsPerRecord = -1
	rows, err := parser.ReadAll()
	if err != nil || len(rows) < 2 {
		return ImportRecord{}, nil, errors.New("unknown smart meter CSV format")
	}
	timeColumn, valueColumn, durationColumn := smartMeterColumns(rows[0])
	if timeColumn < 0 || valueColumn < 0 {
		return ImportRecord{}, nil, errors.New("CSV needs timestamp and import_kwh columns")
	}
	intervals := make([]Interval, 0, len(rows)-1)
	seen := map[string]struct{}{}
	for rowNumber, row := range rows[1:] {
		if isBlankCSVRow(row) {
			continue
		}
		if timeColumn >= len(row) || valueColumn >= len(row) {
			return ImportRecord{}, nil, fmt.Errorf("row %d is incomplete", rowNumber+2)
		}
		at, parseErr := parseSmartMeterTime(row[timeColumn], location)
		if parseErr != nil {
			return ImportRecord{}, nil, fmt.Errorf("row %d has an invalid timestamp", rowNumber+2)
		}
		if at.Minute()%15 != 0 || at.Second() != 0 {
			return ImportRecord{}, nil, fmt.Errorf("row %d does not start on a quarter hour", rowNumber+2)
		}
		value, parseErr := parseLocalizedFloat(row[valueColumn])
		if parseErr != nil || value < 0 {
			return ImportRecord{}, nil, fmt.Errorf("row %d has an invalid import_kwh value", rowNumber+2)
		}
		duration := 15
		if durationColumn >= 0 && durationColumn < len(row) && strings.TrimSpace(row[durationColumn]) != "" {
			duration, parseErr = strconv.Atoi(strings.TrimSpace(row[durationColumn]))
			if parseErr != nil || duration != 15 {
				return ImportRecord{}, nil, fmt.Errorf("row %d must cover 15 minutes", rowNumber+2)
			}
		}
		key := at.UTC().Format(time.RFC3339)
		if _, duplicate := seen[key]; duplicate {
			return ImportRecord{}, nil, fmt.Errorf("row %d duplicates a quarter", rowNumber+2)
		}
		seen[key] = struct{}{}
		intervals = append(intervals, Interval{
			TenantSlug: tenantSlug,
			StartsAt:   at.UTC(),
			Duration:   time.Duration(duration) * time.Minute,
			ImportKWh:  value,
			AverageKW:  AveragePowerKW(value, time.Duration(duration)*time.Minute),
			Quality:    QualityMeasured,
			Source:     "smart-meter",
		})
	}
	if len(intervals) == 0 {
		return ImportRecord{}, nil, errors.New("smart meter CSV contains no intervals")
	}
	sum := sha256.Sum256(raw)
	return ImportRecord{
		ID:         NewID("import"),
		TenantSlug: normalizeSlug(tenantSlug),
		Filename:   strings.TrimSpace(filename),
		SHA256:     hex.EncodeToString(sum[:]),
		Format:     "quarter-hour-import-v1",
		Payload:    append([]byte(nil), raw...),
		ImportedAt: time.Now().UTC(),
	}, intervals, nil
}

func smartMeterColumns(header []string) (int, int, int) {
	timeColumn, valueColumn, durationColumn := -1, -1, -1
	for index, raw := range header {
		name := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(raw, "\ufeff")))
		name = strings.NewReplacer(" ", "_", "-", "_", "ä", "ae", "ö", "oe", "ü", "ue").Replace(name)
		switch name {
		case "timestamp", "start", "starts_at", "zeitpunkt", "datum_zeit", "von":
			timeColumn = index
		case "import_kwh", "energy_kwh", "verbrauch_kwh", "netzbezug_kwh", "wert":
			valueColumn = index
		case "duration_minutes", "dauer_minuten":
			durationColumn = index
		}
	}
	return timeColumn, valueColumn, durationColumn
}

func parseSmartMeterTime(raw string, location *time.Location) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if value, err := time.Parse(layout, raw); err == nil {
			return value, nil
		}
	}
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02 15:04", "02.01.2006 15:04"} {
		if value, err := time.ParseInLocation(layout, raw, location); err == nil {
			return value, nil
		}
	}
	return time.Time{}, errors.New("invalid timestamp")
}

func parseLocalizedFloat(raw string) (float64, error) {
	raw = strings.TrimSpace(strings.ReplaceAll(raw, " ", ""))
	if strings.Count(raw, ",") == 1 && !strings.Contains(raw, ".") {
		raw = strings.ReplaceAll(raw, ",", ".")
	}
	return strconv.ParseFloat(raw, 64)
}

func isBlankCSVRow(row []string) bool {
	for _, item := range row {
		if strings.TrimSpace(item) != "" {
			return false
		}
	}
	return true
}
