package demo

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/textutil"
)

// PDFs travel in the JSON fixture as base64 bytes, with a reproducible digest.
type seedDocument struct {
	ID         string    `json:"id"`
	House      string    `json:"house"`
	Title      string    `json:"title"`
	Category   string    `json:"category"`
	Visibility string    `json:"visibility"`
	Filename   string    `json:"filename"`
	UploadedAt time.Time `json:"uploaded_at"`
	PDF        []byte    `json:"pdf"`
	SHA256     string    `json:"sha256"`
}

func loadDocumentFixture(dir, documentDir string, houses []seedHouse) ([]seedDocument, error) {
	var documents []seedDocument
	err := readJSON(filepath.Join(dir, "documents.json"), &documents)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil // Older, database-only fixtures remain supported.
	}
	if err != nil {
		return nil, err
	}
	if len(documents) > 0 && strings.TrimSpace(documentDir) == "" {
		return nil, fmt.Errorf("demo document fixture requires DocumentDir")
	}
	knownHouses := map[string]bool{}
	for _, house := range houses {
		knownHouses[house.Slug] = true
	}
	seen := map[string]bool{}
	for _, item := range documents {
		key := item.House + "/" + item.ID
		if !knownHouses[item.House] || item.House != textutil.Slug(item.House) || seen[key] || !strings.HasPrefix(item.ID, "demo-document-") || item.ID != textutil.Slug(item.ID) || (item.Filename != item.ID+".pdf" && item.Filename != strings.TrimPrefix(item.ID, "demo-document-")+".pdf") || strings.TrimSpace(item.Title) == "" || item.UploadedAt.IsZero() || store.NormalizeDocumentCategory(item.Category) != item.Category || store.NormalizeDocumentVisibility(item.Visibility) == "" {
			return nil, fmt.Errorf("invalid demo document %s", key)
		}
		if len(item.PDF) > int(store.MaxDocumentBytes) || !bytes.HasPrefix(item.PDF, []byte("%PDF-")) || !bytes.HasSuffix(bytes.TrimSpace(item.PDF), []byte("%%EOF")) || fmt.Sprintf("%x", sha256.Sum256(item.PDF)) != item.SHA256 {
			return nil, fmt.Errorf("invalid demo PDF %s", key)
		}
		seen[key] = true
	}
	return documents, nil
}

func seedDocuments(ctx context.Context, database *sql.DB, documents []seedDocument, identities map[string]store.TenantIdentity, documentDir string) error {
	if len(documents) == 0 {
		return nil
	}
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if strings.Contains(fmt.Sprintf("%T", database.Driver()), "stdlib") {
		if _, err := tx.ExecContext(ctx, `SET LOCAL hausv.cross_tenant = 'on'`); err != nil {
			return err
		}
	}
	for _, item := range documents {
		identity := identities[item.House]
		dir := filepath.Join(documentDir, identity.Slug)
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
		// Fixed fixture-owned IDs and paths restore missing files on reseed,
		// without adding another upload/version or touching user documents.
		if err := os.WriteFile(filepath.Join(dir, item.ID+".pdf"), item.PDF, 0600); err != nil {
			return err
		}
		record := store.NormalizeDocumentRecord(store.DocumentRecord{
			ID: item.ID, SeriesID: item.ID, Version: 1, Current: true,
			TenantSlug: identity.Slug, Title: item.Title, Category: item.Category,
			Visibility: item.Visibility, Filename: item.Filename, StoredFilename: item.ID + ".pdf",
			Size: int64(len(item.PDF)), ContentType: "application/pdf",
			UploadedBy: "verwaltung@musterstadt.example", UploadedAt: item.UploadedAt,
		})
		if err := upsertJSON(ctx, tx, "documents", identity, item.ID, record); err != nil {
			return err
		}
	}
	return tx.Commit()
}
