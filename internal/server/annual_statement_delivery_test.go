package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	appmail "github.com/inspr-at/hausv-org/internal/mail"
	"github.com/inspr-at/hausv-org/internal/store"
)

func archiveDeliveryRun(t *testing.T, a *app, run store.AnnualStatementRun) {
	t.Helper()
	w := archiveDemoRequest(t, a, archiveDemoManager, http.MethodPost, "/app/settings/annual-statement/runs/"+run.ID+"/archive", nil)
	if w.Code != http.StatusSeeOther {
		t.Fatal(w.Code, w.Body.String())
	}
}

type conflictingDeliveryRepository struct {
	store.AnnualStatementDeliveryRepository
	err error
}

func (r conflictingDeliveryRepository) Attempt(_ context.Context, item store.AnnualStatementDelivery, _ func(context.Context) error) (store.AnnualStatementDelivery, bool, error) {
	return item, false, r.err
}

func TestAnnualStatementDeliveryRecordErrors(t *testing.T) {
	a, repos, _ := newArchiveDemoApp(t)
	directory := t.TempDir()
	a.mailer = appmail.NewSMTP("", "", "", "", "verwaltung@example.com").WithOutbox(directory)
	run := createArchiveDemoRun(t, a, repos)
	archiveDeliveryRun(t, a, run)
	for _, tc := range []struct {
		name    string
		err     error
		code    int
		message string
	}{
		{"in progress", fmt.Errorf("reserve: %w", store.ErrDeliveryInProgress), http.StatusConflict, "Versand läuft bereits — bitte Seite neu laden"},
		{"database failure", errors.New("test database failure"), http.StatusInternalServerError, "Versandprotokoll konnte nicht gespeichert werden."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ac := authCtx{email: archiveDemoManager, role: roleManager, tenant: a.tenants[archiveDemoTenant], tenantRef: a.tenantIdentities[archiveDemoTenant].Ref(), repositories: repos}
			ac.repositories.annualStatementDeliveries = conflictingDeliveryRepository{AnnualStatementDeliveryRepository: repos.annualStatementDeliveries, err: tc.err}
			r := httptest.NewRequest(http.MethodPost, "/app/settings/annual-statement/runs/"+run.ID+"/send", nil)
			r.SetPathValue("runID", run.ID)
			w := httptest.NewRecorder()
			a.sendAnnualStatementRun(w, r, ac)
			if w.Code != tc.code || !strings.Contains(w.Body.String(), tc.message) {
				t.Fatal(w.Code, w.Body.String())
			}
		})
	}
	if entries, err := os.ReadDir(directory); err != nil || len(entries) != 0 {
		t.Fatal("reservation error sent mail", entries, err)
	}
}

func TestMusterstadt2025DeliveryThroughRoutes(t *testing.T) {
	a, repos, reseed := newArchiveDemoApp(t)
	directory := t.TempDir()
	a.mailer = appmail.NewSMTP("", "", "", "", "verwaltung@example.com").WithOutbox(directory)
	run := createArchiveDemoRun(t, a, repos)
	route := "/app/settings/annual-statement/runs/" + run.ID + "/send"
	pageRoute := "/app/settings/annual-statement?year=2025&run=" + run.ID
	before := archiveDemoRequest(t, a, archiveDemoManager, http.MethodPost, route, nil)
	if before.Code != http.StatusConflict || !strings.Contains(before.Body.String(), "Zuerst im Archiv ablegen") {
		t.Fatal(before.Code, before.Body.String())
	}
	page := archiveDemoRequest(t, a, archiveDemoManager, http.MethodGet, pageRoute, nil)
	if !strings.Contains(page.Body.String(), "Per E-Mail senden") || !strings.Contains(page.Body.String(), "Postausgang als Datei — Testmodus") {
		t.Fatal("send UI missing")
	}
	archiveDeliveryRun(t, a, run)
	response := archiveDemoRequest(t, a, archiveDemoManager, http.MethodPost, route, nil)
	if response.Code != http.StatusSeeOther {
		t.Fatal(response.Code, response.Body.String())
	}
	location := response.Header().Get("Location")
	if !strings.Contains(location, "sent=31") || !strings.Contains(location, "failed=0") || !strings.Contains(location, "skipped=0") {
		t.Fatal(location)
	}
	rows, err := repos.annualStatementDeliveries.List(run.ID)
	if err != nil || len(rows) != 31 {
		t.Fatal(len(rows), err)
	}
	expected := map[string]store.AnnualStatementDelivery{}
	for _, row := range rows {
		if row.Status != "sent" || row.Error != "" || row.Attempt != 1 || row.Actor != archiveDemoManager {
			t.Fatal(row)
		}
		expected[row.DocumentID] = row
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 31 {
		t.Fatal(len(entries), err)
	}
	seen := map[string]bool{}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".eml") {
			t.Fatal(entry.Name())
		}
		raw, err := os.ReadFile(filepath.Join(directory, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		message, err := mail.ReadMessage(bytes.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		subject, err := new(mime.WordDecoder).DecodeHeader(message.Header.Get("Subject"))
		if err != nil || !strings.HasPrefix(subject, "Jahresabrechnung 2025 · Janusbergweg 123 · ") {
			t.Fatal(subject, err)
		}
		_, params, err := mime.ParseMediaType(message.Header.Get("Content-Type"))
		if err != nil {
			t.Fatal(err)
		}
		reader := multipart.NewReader(message.Body, params["boundary"])
		body, err := reader.NextPart()
		if err != nil {
			t.Fatal(err)
		}
		text, err := io.ReadAll(body)
		if err != nil || !bytes.Contains(text, []byte("Freigegebene Jahresabrechnung")) || !bytes.Contains(text, []byte("Hausverwaltung Musterstadt")) {
			t.Fatal(string(text), err)
		}
		attachment, err := reader.NextPart()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, attachment))
		if err != nil {
			t.Fatal(err)
		}
		matched := false
		for id, row := range expected {
			document, _ := repos.documents.Get(id)
			path, _ := repos.documents.FilePath(document)
			archived, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Equal(data, archived) {
				if message.Header.Get("To") != row.Recipient || seen[id] {
					t.Fatal("wrong recipient or duplicate")
				}
				seen[id], matched = true, true
				break
			}
		}
		if !matched {
			t.Fatal("attachment is not an archived party PDF")
		}
		if _, err := reader.NextPart(); err != io.EOF {
			t.Fatal("extra attachment", err)
		}
	}
	if len(seen) != 31 {
		t.Fatal("not all archives sent")
	}
	page = archiveDemoRequest(t, a, archiveDemoManager, http.MethodGet, strings.Split(strings.TrimPrefix(location, "/"+archiveDemoTenant), "#")[0], nil)
	for _, want := range []string{"31 gesendet · 0 fehlgeschlagen · 0 übersprungen", "Versandprotokoll", "Adresse", "Gesendet"} {
		if !strings.Contains(page.Body.String(), want) {
			t.Fatal("page missing", want)
		}
	}
	response = archiveDemoRequest(t, a, archiveDemoManager, http.MethodPost, route, nil)
	if response.Code != http.StatusSeeOther || !strings.Contains(response.Header().Get("Location"), "skipped=31") {
		t.Fatal(response.Code, response.Header())
	}
	entries, _ = os.ReadDir(directory)
	rows, _ = repos.annualStatementDeliveries.List(run.ID)
	if len(entries) != 31 || len(rows) != 31 {
		t.Fatal("retry created mail or row")
	}
	events := a.auditStore.List(auditFilter{TenantSlug: archiveDemoTenant, Action: store.AuditActionAnnualRunSend, Limit: 10})
	if len(events) != 2 || events[0].Details["skipped"] != "31" || events[1].Details["sent"] != "31" {
		t.Fatal(events)
	}
	if err := reseed(); err != nil {
		t.Fatal(err)
	}
	rows, _ = repos.annualStatementDeliveries.List(run.ID)
	if len(rows) != 31 {
		t.Fatal("reseed lost deliveries")
	}
	t.Log("fresh SQLite seed → run → archive → send: 31 MIME files, each byte-identical to its party archive; 31 sent rows; retry sends zero")
}

type failingDocumentMailer struct {
	appmail.Mailer
	calls  int
	cancel context.CancelFunc
}

func (m *failingDocumentMailer) SendDocument(ctx context.Context, to, subject, body string, attachment appmail.Attachment) error {
	m.calls++
	if m.cancel != nil {
		m.cancel()
		return ctx.Err()
	}
	return errors.New("private upstream details")
}

func TestAnnualStatementDeliveryFailureRetryIntegrityAndCancellation(t *testing.T) {
	a, repos, _ := newArchiveDemoApp(t)
	directory := t.TempDir()
	sink := appmail.NewSMTP("", "", "", "", "verwaltung@example.com").WithOutbox(directory)
	run := createArchiveDemoRun(t, a, repos)
	archiveDeliveryRun(t, a, run)
	route := "/app/settings/annual-statement/runs/" + run.ID + "/send"
	a.mailer = appmail.NewSMTP("", "", "", "", "")
	if w := archiveDemoRequest(t, a, archiveDemoManager, http.MethodPost, route, nil); w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "nicht eingerichtet") {
		t.Fatal(w.Code, w.Body.String())
	}
	failure := &failingDocumentMailer{Mailer: sink}
	a.mailer = failure
	w := archiveDemoRequest(t, a, archiveDemoManager, http.MethodPost, route, nil)
	if w.Code != http.StatusSeeOther || !strings.Contains(w.Header().Get("Location"), "failed=31") || failure.calls != 31 {
		t.Fatal(w.Code, w.Header(), failure.calls)
	}
	rows, err := repos.annualStatementDeliveries.List(run.ID)
	if err != nil || len(rows) != 31 {
		t.Fatal(err)
	}
	for _, row := range rows {
		if row.Status != "failed" || strings.Contains(row.Error, "private") {
			t.Fatal(row)
		}
	}
	party := run.Input.Parties[0]
	document, _ := repos.documents.Get(store.AnnualStatementArchiveID(run.ID, run.Revision, party.UnitID, party.ID))
	path, _ := repos.documents.FilePath(document)
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	corrupt := bytes.Clone(original)
	corrupt[len(corrupt)-1] ^= 1
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, corrupt, 0400); err != nil {
		t.Fatal(err)
	}
	a.mailer = sink
	w = archiveDemoRequest(t, a, archiveDemoManager, http.MethodPost, route, nil)
	if w.Code != http.StatusSeeOther || !strings.Contains(w.Header().Get("Location"), "sent=30") || !strings.Contains(w.Header().Get("Location"), "failed=1") {
		t.Fatal(w.Code, w.Header())
	}
	entries, _ := os.ReadDir(directory)
	if len(entries) != 30 {
		t.Fatal(len(entries))
	}
	rows, _ = repos.annualStatementDeliveries.List(run.ID)
	hashFailure := false
	for _, row := range rows {
		if row.PartyID == party.ID && row.UnitID == party.UnitID && row.Attempt == 2 {
			hashFailure = row.Status == "failed" && strings.Contains(row.Error, "Prüfsumme")
		}
	}
	if !hashFailure {
		t.Fatal("integrity failure missing")
	}
	// Restore the deliberately damaged test fixture; production never repairs archives.
	if err := os.WriteFile(path, original, 0400); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0400); err != nil {
		t.Fatal(err)
	}
	w = archiveDemoRequest(t, a, archiveDemoManager, http.MethodPost, route, nil)
	if !strings.Contains(w.Header().Get("Location"), "sent=1") || !strings.Contains(w.Header().Get("Location"), "skipped=30") {
		t.Fatal(w.Header())
	}
	entries, _ = os.ReadDir(directory)
	rows, _ = repos.annualStatementDeliveries.List(run.ID)
	if len(entries) != 31 || len(rows) != 63 {
		t.Fatal(len(entries), len(rows))
	}
	next := createArchiveDemoRun(t, a, repos)
	archiveDeliveryRun(t, a, next)
	ctx, cancel := context.WithCancel(t.Context())
	failure = &failingDocumentMailer{Mailer: sink, cancel: cancel}
	a.mailer = failure
	req := httptest.NewRequest(http.MethodPost, route, nil).WithContext(ctx)
	req.SetPathValue("runID", next.ID)
	ac := authCtx{email: archiveDemoManager, role: roleManager, tenant: a.tenants[archiveDemoTenant], tenantRef: a.tenantIdentities[archiveDemoTenant].Ref(), repositories: repos}
	a.sendAnnualStatementRun(httptest.NewRecorder(), req, ac)
	rows, err = repos.annualStatementDeliveries.List(next.ID)
	if err != nil || failure.calls != 1 || len(rows) != 1 || rows[0].Status != "failed" || rows[0].Error != "Versand abgebrochen." {
		t.Fatal(rows, failure.calls, err)
	}
}

func TestAnnualStatementDeliveryAuthorizationAndMissingEmail(t *testing.T) {
	a, repos, _ := newArchiveDemoApp(t)
	directory := t.TempDir()
	a.mailer = appmail.NewSMTP("", "", "", "", "verwaltung@example.com").WithOutbox(directory)
	run := createArchiveDemoRun(t, a, repos)
	route := "/app/settings/annual-statement/runs/" + run.ID + "/send"
	for i, role := range []string{roleResident, roleOwner, roleServiceProvider} {
		actor := fmt.Sprintf("role%d@example.com", i)
		a.profiles[actor] = userProfile{Email: actor, Role: role, Tenants: []string{archiveDemoTenant}, AuthMethods: defaultAuthMethods()}
		if w := archiveDemoRequest(t, a, actor, http.MethodPost, route, nil); w.Code != http.StatusForbidden {
			t.Fatal(role, w.Code)
		}
	}
	if w := archiveDemoRequestWithOrigin(t, a, archiveDemoManager, http.MethodPost, route, nil, "https://foreign.example"); w.Code != http.StatusForbidden {
		t.Fatal("origin", w.Code)
	}
	if w := archiveDemoRequest(t, a, archiveDemoManager, http.MethodGet, route, nil); w.Code != http.StatusNotFound && w.Code != http.StatusMethodNotAllowed {
		t.Fatal("GET", w.Code)
	}
	if w := archiveDemoRequest(t, a, archiveDemoManager, http.MethodPost, "/app/settings/annual-statement/runs/missing/send", nil); w.Code != http.StatusNotFound {
		t.Fatal("missing", w.Code)
	}
	foreign := a.repositoriesForTenant(testTenantRef("other"))
	ac := authCtx{email: archiveDemoManager, role: roleManager, tenant: tenantConfig{Slug: "other"}, tenantRef: testTenantRef("other"), repositories: foreign}
	request := httptest.NewRequest(http.MethodPost, route, nil)
	request.SetPathValue("runID", run.ID)
	w := httptest.NewRecorder()
	a.sendAnnualStatementRun(w, request, ac)
	if w.Code != http.StatusNotFound {
		t.Fatal("foreign run", w.Code)
	}
	archiveDeliveryRun(t, a, run)
	// A historical snapshot may contain an identifier that is not an email.
	run.Input.Parties[0].ID = "missing-email"
	party := run.Input.Parties[0]
	_, err := repos.documents.CreateGenerated(store.DocumentRecord{Title: "Archiv", UploadedBy: archiveDemoManager, UnitID: party.UnitID, AnnualStatementArchive: &store.AnnualStatementArchiveMetadata{RunID: run.ID, Revision: run.Revision, PeriodYear: run.PeriodYear, PartyID: party.ID}}, "test.pdf", "application/pdf", []byte("%PDF-test"), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	ac = authCtx{email: archiveDemoManager, role: roleManager, tenant: a.tenants[archiveDemoTenant], tenantRef: a.tenantIdentities[archiveDemoTenant].Ref(), repositories: repos}
	ac.repositories.annualStatementRuns = archiveSnapshotRepository{AnnualStatementRunRepository: repos.annualStatementRuns, run: run}
	w = httptest.NewRecorder()
	a.sendAnnualStatementRun(w, request, ac)
	if w.Code != http.StatusSeeOther || !strings.Contains(w.Header().Get("Location"), "failed=1") {
		t.Fatal(w.Code, w.Body.String())
	}
	rows, err := repos.annualStatementDeliveries.List(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, row := range rows {
		if row.PartyID == "missing-email" {
			found = row.Status == "failed" && strings.Contains(row.Error, "gültige E-Mail-Adresse")
		}
	}
	if !found {
		t.Fatal("missing email not recorded")
	}
}
