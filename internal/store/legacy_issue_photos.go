package store

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/markus-barta/hausv-org/internal/textutil"
)

// MigrateLegacyIssuePhotos moves photos recorded in the legacy
// ResidentIssue.PhotoPaths field into the attachment store, then clears the
// field (HAUSV-175).
//
// The write path that produced these (IssueStore.SavePhoto) is long gone; only
// the read path kept them reachable. Once an issue's photos are attachments,
// nothing needs PhotoPaths any more.
//
// Idempotent by construction: clearing the field is what marks an issue done, so
// a second run finds nothing to do. A photo whose file is missing on disk is
// skipped WITHOUT clearing the field, so a bad path is never silently forgotten.
func MigrateLegacyIssuePhotos(
	issues IssueStorage,
	attachments AttachmentStorage,
	issueAttachmentDir string,
	tenantSlugs []string,
	now time.Time,
) (migrated int, err error) {
	if issues == nil || attachments == nil || strings.TrimSpace(issueAttachmentDir) == "" {
		return 0, nil
	}
	if now.IsZero() {
		now = time.Now()
	}
	for _, rawSlug := range tenantSlugs {
		tenantSlug := textutil.Slug(rawSlug)
		if tenantSlug == "" {
			continue
		}
		for _, issue := range issues.ListTenant(tenantSlug) {
			if len(issue.PhotoPaths) == 0 {
				continue
			}
			uploads := make([]UploadedFile, 0, len(issue.PhotoPaths))
			for _, photoPath := range issue.PhotoPaths {
				// Only the base name is trustworthy: the stored value is a display
				// path, and the file always lives under <dir>/<tenant>/.
				name := filepath.Base(strings.TrimSpace(photoPath))
				if name == "" || name == "." || name == string(filepath.Separator) {
					continue
				}
				src := filepath.Join(issueAttachmentDir, tenantSlug, name)
				info, statErr := os.Stat(src)
				if statErr != nil || info.IsDir() || info.Size() == 0 {
					continue
				}
				uploads = append(uploads, UploadedFile{
					Filename: name,
					Size:     info.Size(),
					Open: func() (io.ReadSeekCloser, error) {
						return os.Open(src)
					},
				})
			}
			if len(uploads) != len(issue.PhotoPaths) {
				// At least one file could not be read. Leave the field intact so the
				// legacy reference is not lost, and report it.
				return migrated, fmt.Errorf(
					"issue %s/%s: %d of %d legacy photos unreadable under %s",
					tenantSlug, issue.ID, len(issue.PhotoPaths)-len(uploads), len(issue.PhotoPaths), issueAttachmentDir,
				)
			}
			uploadedBy := textutil.Email(issue.AuthorEmail)
			if uploadedBy == "" {
				uploadedBy = "system@migration.local"
			}
			createdAt := issue.CreatedAt
			if createdAt.IsZero() {
				createdAt = now
			}
			created, createErr := attachments.CreateUploaded(tenantSlug, "issue", issue.ID, uploadedBy, uploads, createdAt)
			if createErr != nil {
				return migrated, fmt.Errorf("issue %s/%s: %w", tenantSlug, issue.ID, createErr)
			}
			if len(created) == 0 {
				return migrated, fmt.Errorf("issue %s/%s: no attachment created", tenantSlug, issue.ID)
			}
			// Only now is the legacy field safe to drop.
			if _, clearErr := issues.ClearPhotoPaths(tenantSlug, issue.ID); clearErr != nil {
				return migrated, fmt.Errorf("issue %s/%s: clear photo paths: %w", tenantSlug, issue.ID, clearErr)
			}
			migrated += len(created)
		}
	}
	return migrated, nil
}
