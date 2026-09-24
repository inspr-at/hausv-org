package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/indexation"
)

type IndexObservation struct {
	ID, Series, Period, Status, Source string
	Value                              indexation.Decimal
	FetchedAt                          time.Time
}

func (v IndexObservation) key() string { return v.Series + "/" + v.Period }

type IndexChange struct {
	Series, Period, Kind, Before, After, BeforeStatus, AfterStatus string
}

type IndexImport struct {
	ID, URL, SHA256, DataSHA256, StatusSHA256, Actor string
	FetchedAt                                        time.Time
	RowsAdded, RowsRevised, RowsFlagged              int
	Changes                                          []IndexChange
	Duplicate                                        bool
}

func (v IndexImport) SeriesLabel() string {
	snapshot, err := publishedIndices()
	if err == nil {
		for _, source := range snapshot.Manifest.Sources {
			if source.URL == v.URL {
				return strings.Replace(string(source.Series), "VPI", "VPI ", 1)
			}
		}
	}
	return "VPI"
}

// IndexReferenceStore is a system service, not a tenant mutation repository.
// Its only caller on the serve path is the administrator refresh action.
type IndexReferenceStore struct{ db *TenantDB }

func NewIndexReferenceStore(db *TenantDB) *IndexReferenceStore { return &IndexReferenceStore{db: db} }

// Refresh retrieves only pinned official URLs. One bad source does not prevent
// the remaining series from refreshing; each validated release is atomic.
func (s *IndexReferenceStore) Refresh(ctx context.Context, client indexation.HTTPDoer, actor string, now time.Time) ([]IndexImport, error) {
	snapshot, err := publishedIndices()
	if err != nil {
		return nil, err
	}
	var results []IndexImport
	var failures []error
	for _, source := range snapshot.Manifest.Sources {
		if err := ctx.Err(); err != nil {
			failures = append(failures, err)
			break
		}
		fetched, err := indexation.FetchOGD(ctx, client, source, now)
		if err == nil {
			var result IndexImport
			result, err = s.Apply(ctx, fetched, actor)
			if err == nil {
				results = append(results, result)
			}
		}
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", source.Series, err))
		}
	}
	return results, errors.Join(failures...)
}

// Apply appends revisions and moves only superseded_by on the old observation.
// Final corrections remain visible conflicts; the active value is preserved.
func (s *IndexReferenceStore) Apply(ctx context.Context, fetched indexation.FetchedOGD, actor string) (IndexImport, error) {
	result := IndexImport{URL: fetched.URL, SHA256: fetched.SHA256, DataSHA256: fetched.DataSHA256, StatusSHA256: fetched.StatusSHA256, Actor: actor, FetchedAt: fetched.FetchedAt}
	if strings.TrimSpace(actor) == "" || len(fetched.SHA256) != 64 || len(fetched.DataSHA256) != 64 || len(fetched.StatusSHA256) != 64 || fetched.FetchedAt.IsZero() || len(fetched.Imported.Values) == 0 {
		return result, fmt.Errorf("invalid index import")
	}
	tx, err := s.db.Unscoped("VPI system importer writes global Statistics Austria reference data and revision audit").Begin()
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	// The first statement is a write: serialize importers across processes and
	// avoid SQLite read-to-write upgrades. No network request holds this lock.
	if err = lockIndexReference(ctx, tx); err != nil {
		return result, err
	}
	err = tx.QueryRowContext(ctx, `SELECT id FROM index_imports WHERE url=$1 AND sha256=$2`, result.URL, result.SHA256).Scan(&result.ID)
	if err == nil {
		result.Duplicate = true
		return result, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return result, err
	}
	current, err := currentIndexObservations(tx)
	if err != nil {
		return result, err
	}
	result.ID, err = newLeaseID()
	if err != nil {
		return result, err
	}
	incoming := importedObservations(fetched.Imported)
	type revision struct {
		before, after IndexObservation
		baseline      bool
	}
	var revisions []revision
	seen := map[string]bool{}
	for _, value := range incoming {
		if seen[value.key()] || value.Value <= 0 || !indexation.Series(value.Series).Valid() || value.Source != result.URL || !value.FetchedAt.Equal(result.FetchedAt) {
			return result, fmt.Errorf("invalid or duplicate imported observation")
		}
		seen[value.key()] = true
		old, exists := current[value.key()]
		if exists && old.Value == value.Value && old.Status == value.Status {
			continue
		}
		kind := "added"
		if exists {
			kind = "revised"
			if old.Status == "final" {
				kind = "final_conflict"
			}
		}
		change := IndexChange{Series: value.Series, Period: value.Period, Kind: kind, After: value.Value.String(), AfterStatus: value.Status}
		if exists {
			change.Before, change.BeforeStatus = old.Value.String(), old.Status
		}
		result.Changes = append(result.Changes, change)
		if kind == "final_conflict" {
			result.RowsFlagged++
			continue
		}
		value.ID, err = newLeaseID()
		if err != nil {
			return result, err
		}
		baseline := exists && old.ID == ""
		if baseline {
			old.ID, err = newLeaseID()
			if err != nil {
				return result, err
			}
		}
		revisions = append(revisions, revision{before: old, after: value, baseline: baseline})
		if exists {
			result.RowsRevised++
		} else {
			result.RowsAdded++
		}
	}
	raw, err := json.Marshal(result.Changes)
	if err != nil {
		return result, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO index_imports(id,url,sha256,data_sha256,status_sha256,actor,fetched_at,rows_added,rows_revised,rows_flagged,changes) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, result.ID, result.URL, result.SHA256, result.DataSHA256, result.StatusSHA256, actor, stamp(result.FetchedAt), result.RowsAdded, result.RowsRevised, result.RowsFlagged, string(raw))
	if err != nil {
		return result, err
	}
	insert := func(v IndexObservation, superseded any) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO index_values(id,series,period,value_millionths,status,source,fetched_at,import_id,superseded_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, v.ID, v.Series, v.Period, int64(v.Value), v.Status, v.Source, stamp(v.FetchedAt), result.ID, superseded)
		return err
	}
	for _, revision := range revisions {
		if revision.baseline {
			// Preserve the embedded predecessor as evidence on its first revision.
			if err = insert(revision.before, revision.after.ID); err != nil {
				return result, err
			}
		} else if revision.before.ID != "" {
			if _, err = tx.ExecContext(ctx, `UPDATE index_values SET superseded_by=$1 WHERE id=$2 AND superseded_by IS NULL`, revision.after.ID, revision.before.ID); err != nil {
				return result, err
			}
		}
		if err = insert(revision.after, nil); err != nil {
			return result, err
		}
	}
	return result, tx.Commit()
}

func lockIndexReference(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO app_meta(key,value) VALUES('index_refresh_lock','1') ON CONFLICT(key) DO UPDATE SET value=excluded.value`)
	return err
}

func (s *IndexReferenceStore) Recent(tenant TenantRef) ([]IndexImport, error) {
	rows, err := s.db.For(tenant).Query(`SELECT id,url,sha256,data_sha256,status_sha256,actor,fetched_at,rows_added,rows_revised,rows_flagged,changes FROM index_imports ORDER BY fetched_at DESC,id DESC LIMIT 20`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []IndexImport
	for rows.Next() {
		var v IndexImport
		var fetched, raw string
		if err = rows.Scan(&v.ID, &v.URL, &v.SHA256, &v.DataSHA256, &v.StatusSHA256, &v.Actor, &fetched, &v.RowsAdded, &v.RowsRevised, &v.RowsFlagged, &raw); err != nil {
			return nil, err
		}
		if v.FetchedAt, err = time.Parse(time.RFC3339Nano, fetched); err != nil {
			return nil, err
		}
		if err = json.Unmarshal([]byte(raw), &v.Changes); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, rows.Err()
}

type indexQuerier interface {
	Query(string, ...any) (*sql.Rows, error)
}

func importedObservations(imported indexation.Imported) []IndexObservation {
	var values []IndexObservation
	status := func(preliminary bool) string {
		if preliminary {
			return "preliminary"
		}
		return "final"
	}
	for _, v := range imported.Values {
		values = append(values, IndexObservation{Series: string(v.Series), Period: string(v.Month), Value: v.Value, Status: status(v.Preliminary), Source: v.Source, FetchedAt: v.FetchedAt})
	}
	for _, v := range imported.Annual {
		values = append(values, IndexObservation{Series: string(v.Series), Period: strconv.Itoa(v.Year), Value: v.Value, Status: status(v.Preliminary), Source: v.Source, FetchedAt: v.FetchedAt})
	}
	return values
}

func currentIndexObservations(q indexQuerier) (map[string]IndexObservation, error) {
	snapshot, err := publishedIndices()
	if err != nil {
		return nil, err
	}
	values := map[string]IndexObservation{}
	for _, v := range importedObservations(indexation.Imported{Values: snapshot.Data.Values(), Annual: snapshot.Annual}) {
		values[v.key()] = v
	}
	rows, err := q.Query(`SELECT id,series,period,value_millionths,status,source,fetched_at FROM index_values WHERE superseded_by IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var v IndexObservation
		var fetched string
		if err = rows.Scan(&v.ID, &v.Series, &v.Period, &v.Value, &v.Status, &v.Source, &fetched); err != nil {
			return nil, err
		}
		v.FetchedAt, err = time.Parse(time.RFC3339Nano, fetched)
		if err != nil {
			return nil, err
		}
		values[v.key()] = v
	}
	return values, rows.Err()
}

// runtimeIndexSnapshot is the single lookup snapshot used by preview, creation
// and approval. Runtime observations override embedded values; missing months
// retain the offline data and its official chaining factors.
func runtimeIndexSnapshot(q indexQuerier) (indexation.Snapshot, error) {
	snapshot, err := publishedIndices()
	if err != nil {
		return snapshot, err
	}
	observations, err := currentIndexObservations(q)
	if err != nil {
		return snapshot, err
	}
	keys := make([]string, 0, len(observations))
	for key := range observations {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var values []indexation.IndexValue
	snapshot.Annual = nil
	var runtimeIDs []string
	for _, key := range keys {
		v := observations[key]
		if v.ID != "" {
			runtimeIDs = append(runtimeIDs, v.ID)
		}
		if len(v.Period) == 4 {
			year, err := strconv.Atoi(v.Period)
			if err != nil {
				return snapshot, err
			}
			snapshot.Annual = append(snapshot.Annual, indexation.AnnualValue{RevisionID: v.ID, Runtime: v.ID != "", Series: indexation.Series(v.Series), Year: year, Value: v.Value, Preliminary: v.Status == "preliminary", Source: v.Source, FetchedAt: v.FetchedAt})
		} else {
			values = append(values, indexation.IndexValue{RevisionID: v.ID, Runtime: v.ID != "", Series: indexation.Series(v.Series), Month: indexation.Month(v.Period), Value: v.Value, Preliminary: v.Status == "preliminary", Source: v.Source, FetchedAt: v.FetchedAt})
		}
	}
	snapshot.Data, err = indexation.NewDataset(values, snapshot.Manifest.Factors)
	if len(runtimeIDs) > 0 {
		snapshot.Manifest.DataSHA256 = fmt.Sprintf("%x", sha256.Sum256([]byte(snapshot.Manifest.DataSHA256+strings.Join(runtimeIDs, ","))))
	}
	return snapshot, err
}

func indexRunRevised(q indexQuerier, run ValorisationRun) (bool, error) {
	values, err := currentIndexObservations(q)
	if err != nil {
		return false, err
	}
	for _, used := range run.IndexSnapshot {
		v, ok := values[used.Series+"/"+used.Period]
		if !ok || v.Value.String() != used.Value || v.Status != used.Status || v.ID != used.RevisionID {
			return true, nil
		}
	}
	// A disputed final figure also deserves review although we kept it active.
	rows, err := q.Query(`SELECT changes FROM index_imports WHERE rows_flagged > 0`)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var raw string
		var changes []IndexChange
		if err = rows.Scan(&raw); err != nil {
			return false, err
		}
		if err = json.Unmarshal([]byte(raw), &changes); err != nil {
			return false, err
		}
		for _, change := range changes {
			if change.Kind != "final_conflict" {
				continue
			}
			for _, used := range run.IndexSnapshot {
				if used.Series == change.Series && used.Period == change.Period {
					return true, nil
				}
			}
		}
	}
	return false, rows.Err()
}
