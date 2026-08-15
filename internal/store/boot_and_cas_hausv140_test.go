package store

import (
	"bytes"
	"encoding/base64"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// HAUSV-140: boot behavior of every JSON-backed store constructor (a malformed
// file must be a HARD error, never a silent empty start that looks like data
// loss), plus the two riskiest write paths — attachment rollback and the
// document compare-and-swap replace.

// onePixelPNG is a minimal valid 1x1 PNG so content-type sniffing and image
// variant decoding accept it.
var onePixelPNG = mustDecodeBase64("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==")

func mustDecodeBase64(s string) []byte {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

type nopReadSeekCloser struct{ *bytes.Reader }

func (nopReadSeekCloser) Close() error { return nil }

func uploadFrom(name string, data []byte) UploadedFile {
	return UploadedFile{
		Filename: name,
		Size:     int64(len(data)),
		Open: func() (io.ReadSeekCloser, error) {
			return nopReadSeekCloser{bytes.NewReader(data)}, nil
		},
	}
}

func TestStoreConstructorBootBehavior(t *testing.T) {
	fileDir := t.TempDir()
	// AuditStore is excluded on purpose: it is append-JSONL with its own
	// torn-trailing-line vs corrupt-middle-line contract, covered by
	// durability_hausv136_137_test.go.
	constructors := map[string]func(path string) (any, error){
		"invite":            func(p string) (any, error) { return NewInviteStore(p) },
		"announcement":      func(p string) (any, error) { return NewAnnouncementStore(p) },
		"announcementRead":  func(p string) (any, error) { return NewAnnouncementReadStore(p) },
		"notificationPref":  func(p string) (any, error) { return NewNotificationPrefStore(p) },
		"profileOverlay":    func(p string) (any, error) { return NewProfileOverlayStore(p) },
		"contactBook":       func(p string) (any, error) { return NewContactBookStore(p) },
		"unitPaymentStatus": func(p string) (any, error) { return NewUnitPaymentStatusStore(p) },
		"activity":          func(p string) (any, error) { return NewActivityStore(p) },
		"event":             func(p string) (any, error) { return NewEventStore(p) },
		"vote":              func(p string) (any, error) { return NewVoteStore(p) },
		"unit":              func(p string) (any, error) { return NewUnitStore(p) },
		"telegram":          func(p string) (any, error) { return NewTelegramStore(p) },
		"parking":           func(p string) (any, error) { return NewParkingStore(p) },
		"handover":          func(p string) (any, error) { return NewHandoverStore(p) },
		"issue":             func(p string) (any, error) { return NewIssueStore(p, fileDir) },
		"attachment":        func(p string) (any, error) { return NewAttachmentStore(p, fileDir) },
		"document":          func(p string) (any, error) { return NewDocumentStore(p, fileDir) },
	}
	for name, ctor := range constructors {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()

			// 1) missing file -> empty store, no error.
			if _, err := ctor(filepath.Join(dir, "missing.json")); err != nil {
				t.Fatalf("missing file must not error, got %v", err)
			}

			// 2) empty / whitespace-only file -> empty store, no error.
			empty := filepath.Join(dir, "empty.json")
			if err := os.WriteFile(empty, []byte("   \n"), 0o600); err != nil {
				t.Fatalf("write empty: %v", err)
			}
			if _, err := ctor(empty); err != nil {
				t.Fatalf("empty file must not error, got %v", err)
			}

			// 3) malformed JSON -> HARD error. A silent empty start would look
			// exactly like total data loss.
			bad := filepath.Join(dir, "bad.json")
			if err := os.WriteFile(bad, []byte("}{ not valid json"), 0o600); err != nil {
				t.Fatalf("write bad: %v", err)
			}
			if _, err := ctor(bad); err == nil {
				t.Fatal("malformed JSON must be a hard error, got nil")
			}
		})
	}
}

func TestAttachmentCreateRollsBackOnPartialFailure(t *testing.T) {
	fileDir := t.TempDir()
	store, err := NewAttachmentStore(filepath.Join(t.TempDir(), "attachments.json"), fileDir)
	if err != nil {
		t.Fatalf("new attachment store: %v", err)
	}
	attachments, _ := BindAttachmentRepository(store, "demo")
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)

	// First upload is a valid PNG (its file gets written); the second is a text
	// file that fails the type check — the whole batch must roll back so the
	// already-written file and record don't survive as orphans.
	uploads := []UploadedFile{
		uploadFrom("photo.png", onePixelPNG),
		uploadFrom("notes.txt", []byte("this is not an image")),
	}
	if _, err := attachments.CreateUploaded("issue", "issue-1", "admin@example.com", uploads, now); err == nil {
		t.Fatal("a batch containing an invalid upload must fail")
	}

	if got := attachments.ListEntity("issue", "issue-1"); len(got) != 0 {
		t.Fatalf("rollback must leave no records, got %d", len(got))
	}

	tenantDir := filepath.Join(fileDir, "demo")
	entries, _ := os.ReadDir(tenantDir)
	files := 0
	for _, e := range entries {
		if !e.IsDir() {
			files++
		}
	}
	if files != 0 {
		t.Fatalf("rollback must delete created files, found %d in %s", files, tenantDir)
	}
}

func TestDocumentReplaceCASVersioning(t *testing.T) {
	fileDir := t.TempDir()
	store, err := NewDocumentStore(filepath.Join(t.TempDir(), "documents.json"), fileDir)
	if err != nil {
		t.Fatalf("new document store: %v", err)
	}
	documents, _ := BindDocumentRepository(store, "demo")
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)

	created, err := documents.Create(
		DocumentRecord{TenantSlug: "demo", Title: "Hausordnung", Visibility: "all", UploadedBy: "admin@example.com"},
		uploadFrom("v1.png", onePixelPNG), now)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Version != 1 || !created.Current {
		t.Fatalf("v1 should be version 1 and current: %+v", created)
	}

	// Replacing the current version bumps to v2; v1 becomes non-current.
	v2, v1, err := documents.Replace(created.ID, "admin@example.com", uploadFrom("v2.png", onePixelPNG), now.Add(time.Hour))
	if err != nil {
		t.Fatalf("replace current: %v", err)
	}
	if v2.Version != 2 || !v2.Current || v2.SupersedesID != created.ID {
		t.Fatalf("v2 should be version 2, current, superseding v1: %+v", v2)
	}
	if v1.Current || v1.ReplacedByID != v2.ID {
		t.Fatalf("v1 should be non-current and replaced by v2: %+v", v1)
	}

	// CAS: replacing the now-superseded v1 id must fail.
	if _, _, err := documents.Replace(created.ID, "admin@example.com", uploadFrom("stale.png", onePixelPNG), now.Add(2*time.Hour)); err == nil {
		t.Fatal("replacing a superseded version must fail (compare-and-swap)")
	}

	// Replacing a missing id must fail.
	if _, _, err := documents.Replace("does-not-exist", "admin@example.com", uploadFrom("x.png", onePixelPNG), now); err == nil {
		t.Fatal("replacing a missing document must fail")
	}

	// The current version stays replaceable (v2 -> v3).
	v3, _, err := documents.Replace(v2.ID, "admin@example.com", uploadFrom("v3.png", onePixelPNG), now.Add(3*time.Hour))
	if err != nil {
		t.Fatalf("replace current v2: %v", err)
	}
	if v3.Version != 3 || !v3.Current {
		t.Fatalf("v3 should be version 3 and current: %+v", v3)
	}
}
