package store

import (
	"crypto/sha256"
	"fmt"
	"time"
)

type ValorisationArchiveMetadata struct {
	RunID, ItemID, SHA256, LetterDate, CollectableFrom string
	ArchivedBy                                         string
	ArchivedAt                                         time.Time
}

func documentArchiveSHA(item DocumentRecord) string {
	if item.ValorisationArchive != nil {
		return item.ValorisationArchive.SHA256
	}
	if item.AnnualStatementArchive != nil {
		return item.AnnualStatementArchive.SHA256
	}
	return ""
}
func prepareValorisationArchive(item DocumentRecord, filename, contentType string, data []byte, now time.Time) (DocumentRecord, error) {
	a := item.ValorisationArchive
	if a == nil || a.RunID == "" || a.ItemID == "" || a.LetterDate == "" || item.UnitID == "" || item.UploadedBy == "" || item.TenantSlug == "" || contentType != "application/pdf" || len(data) == 0 || len(data) > MaxDocumentBytes {
		return item, fmt.Errorf("invalid valorisation archive")
	}
	a.SHA256 = fmt.Sprintf("%x", sha256.Sum256(data))
	item.ID = fmt.Sprintf("valorisation-%x", sha256.Sum256([]byte(a.RunID+"/"+a.ItemID+"/"+a.LetterDate+"/"+a.SHA256)))
	item.SeriesID, item.Version, item.Current = item.ID, 1, true
	item.Category, item.Visibility = DocumentCategoryBilling, DocumentVisibilityManagerOnly
	item.Filename, item.StoredFilename = filename, item.ID+".pdf"
	item.ContentType, item.Size, item.UploadedAt = contentType, int64(len(data)), now.UTC()
	a.ArchivedBy, a.ArchivedAt = item.UploadedBy, now.UTC()
	return NormalizeDocumentRecord(item), nil
}
