package store

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var ErrDocumentArchived = errors.New("archived documents cannot be replaced or deleted")

// AnnualStatementArchiveMetadata is stored in the existing document JSON.
// An empty PartyID and UnitID identify the combined PDF of the run.
type AnnualStatementArchiveMetadata struct {
	RunID      string    `json:"run_id"`
	Revision   int       `json:"revision"`
	PeriodYear int       `json:"period_year"`
	PartyID    string    `json:"party_id"`
	SHA256     string    `json:"sha256"`
	ArchivedBy string    `json:"archived_by"`
	ArchivedAt time.Time `json:"archived_at"`
}

// AnnualStatementArchiveID uses the existing document primary key to serialize
// concurrent creates across processes. Content and actor are deliberately not
// part of the identity: retries must return the original archive copy.
func AnnualStatementArchiveID(runID string, revision int, unitID, partyID string) string {
	key, _ := json.Marshal([]any{runID, revision, unitID, partyID})
	return fmt.Sprintf("annual-archive-%x", sha256.Sum256(key))
}

func prepareArchiveRecord(item DocumentRecord, filename, contentType string, data []byte, now time.Time) (DocumentRecord, error) {
	item = CopyDocument(item)
	item.UnitID = NormalizeUnitID(item.UnitID)
	archive := item.AnnualStatementArchive
	if archive == nil || strings.TrimSpace(archive.RunID) == "" || archive.Revision < 1 || archive.PeriodYear < 1 || archive.PeriodYear > 9999 || (item.UnitID == "") != (archive.PartyID == "") {
		return DocumentRecord{}, fmt.Errorf("invalid archive identity")
	}
	if contentType != "application/pdf" || len(data) == 0 || len(data) > MaxDocumentBytes {
		return DocumentRecord{}, fmt.Errorf("invalid archive PDF")
	}
	if now.IsZero() {
		now = time.Now()
	}
	item.ID = AnnualStatementArchiveID(archive.RunID, archive.Revision, item.UnitID, archive.PartyID)
	item.SeriesID, item.Version, item.Current = item.ID, 1, true
	item.SupersedesID, item.ReplacedByID = "", ""
	item.Category, item.Visibility = DocumentCategoryBilling, DocumentVisibilityManagerOnly
	item.Filename, item.StoredFilename = filename, item.ID+".pdf"
	item.ContentType, item.Size, item.UploadedAt = contentType, int64(len(data)), now.UTC()
	item = NormalizeDocumentRecord(item)
	if item.TenantSlug == "" || item.Title == "" || item.UploadedBy == "" {
		return DocumentRecord{}, fmt.Errorf("invalid archive metadata")
	}
	archive.SHA256 = fmt.Sprintf("%x", sha256.Sum256(data))
	archive.ArchivedBy, archive.ArchivedAt = item.UploadedBy, item.UploadedAt
	return item, nil
}

// Archive files are created exclusively and never rewritten or removed on a
// metadata rollback. A retry can adopt a complete orphan only if its bytes match.
// An incomplete orphan fails closed; silently repairing it would rewrite history.
func writeArchiveFile(fileDir string, item DocumentRecord, data []byte) error {
	if fileDir == "" {
		return fmt.Errorf("document file directory unavailable")
	}
	dir := filepath.Join(fileDir, item.TenantSlug)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	path := filepath.Join(dir, item.StoredFilename)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0400)
	if errors.Is(err, os.ErrExist) {
		return verifyArchiveFile(fileDir, item)
	}
	if err != nil {
		return err
	}
	// The exclusive create makes this call the file's only writer. A partial
	// file from a failed write must not survive: a retry would then read it as
	// an integrity mismatch and the run could never be archived.
	discard := func(cause error) error {
		file.Close()
		_ = os.Remove(path)
		return cause
	}
	if _, err := file.Write(data); err != nil {
		return discard(err)
	}
	if err := file.Sync(); err != nil {
		return discard(err)
	}
	if err := file.Close(); err != nil {
		return discard(err)
	}
	directory, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

func verifyArchiveFile(fileDir string, item DocumentRecord) error {
	path, ok := documentFilePathIn(fileDir, item)
	if !ok || item.AnnualStatementArchive == nil {
		return fmt.Errorf("invalid archive file")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() != item.Size {
		return fmt.Errorf("archive file integrity mismatch")
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	if fmt.Sprintf("%x", hash.Sum(nil)) != item.AnnualStatementArchive.SHA256 {
		return fmt.Errorf("archive file integrity mismatch")
	}
	return nil
}

func (s *SQLDocumentStore) createArchive(tenant TenantRef, item DocumentRecord, filename, contentType string, data []byte, now time.Time) (DocumentRecord, error) {
	item, err := prepareArchiveRecord(item, filename, contentType, data, now)
	if err != nil {
		return DocumentRecord{}, err
	}
	blob, err := json.Marshal(item)
	if err != nil {
		return DocumentRecord{}, err
	}
	tx, err := s.db.For(tenant).Begin()
	if err != nil {
		return DocumentRecord{}, err
	}
	defer tx.Rollback()
	result, err := tx.Exec(`INSERT INTO documents(tenant_id, tenant_slug, id, data) VALUES($1, $2, $3, $4) ON CONFLICT(tenant_slug, id) DO NOTHING`, tenant.ID, tenant.Slug, item.ID, string(blob))
	if err != nil {
		return DocumentRecord{}, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return DocumentRecord{}, err
	}
	if count == 0 {
		var existing string
		if err := tx.QueryRow(`SELECT data FROM documents WHERE tenant_id=$1 AND id=$2`, tenant.ID, item.ID).Scan(&existing); err != nil {
			return DocumentRecord{}, err
		}
		if err := json.Unmarshal([]byte(existing), &item); err != nil {
			return DocumentRecord{}, err
		}
		if err := verifyArchiveFile(s.fileDir, item); err != nil {
			return DocumentRecord{}, err
		}
	} else if err := writeArchiveFile(s.fileDir, item, data); err != nil {
		return DocumentRecord{}, err
	}
	if err := tx.Commit(); err != nil {
		return DocumentRecord{}, err
	}
	return CopyDocument(item), nil
}

func (s *DocumentStore) createArchive(item DocumentRecord, filename, contentType string, data []byte, now time.Time) (DocumentRecord, error) {
	item, err := prepareArchiveRecord(item, filename, contentType, data, now)
	if err != nil {
		return DocumentRecord{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.data.Documents {
		if existing.TenantSlug == item.TenantSlug && existing.ID == item.ID {
			if err := verifyArchiveFile(s.fileDir, existing); err != nil {
				return DocumentRecord{}, err
			}
			return CopyDocument(existing), nil
		}
	}
	if err := writeArchiveFile(s.fileDir, item, data); err != nil {
		return DocumentRecord{}, err
	}
	previous := append([]DocumentRecord(nil), s.data.Documents...)
	s.data.Documents = append(s.data.Documents, item)
	SortDocuments(s.data.Documents)
	if err := s.saveLocked(); err != nil {
		s.data.Documents = previous
		return DocumentRecord{}, err
	}
	return CopyDocument(item), nil
}
