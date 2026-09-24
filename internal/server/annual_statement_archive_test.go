package server

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/demo"
	"github.com/inspr-at/hausv-org/internal/store"
)

const archiveDemoTenant = "janusbergweg-123"
const archiveDemoManager = "vera.verwalter@musterstadt.example"

func newArchiveDemoApp(t *testing.T) (*app, requestRepositories, func() error) {
	t.Helper()
	a := newTestPortalApp(t, userProfile{Email: archiveDemoManager, Role: roleManager, Tenants: []string{archiveDemoTenant}, AuthMethods: defaultAuthMethods()})
	database, err := db.Open(filepath.Join(t.TempDir(), "archive.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	scoped, err := db.NewScoped(db.Config{Backend: db.BackendSQLite}, database)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { scoped.Close() })
	options := demo.SeedOptions{Reset: true, DocumentDir: t.TempDir(), Anchor: time.Date(2030, 8, 1, 0, 0, 0, 0, time.UTC)}
	if _, err := demo.Load(t.Context(), database, "../../scripts/demo/seed", options); err != nil {
		t.Fatal(err)
	}
	identities, err := store.EnsureTenantIdentities(t.Context(), database, []store.TenantIdentity{{Slug: archiveDemoTenant}})
	if err != nil {
		t.Fatal(err)
	}
	a.pool, a.scopedDB, a.tenantDB = database, scoped, store.NewTenantDB(scoped)
	a.tenantIdentities = identities
	a.defaultTenant = archiveDemoTenant
	a.tenants = map[string]tenantConfig{archiveDemoTenant: {Slug: archiveDemoTenant, Name: "Janusbergweg 123", ContactName: "Hausverwaltung Musterstadt"}}
	a.documentStore = store.NewSQLDocumentStore(a.tenantDB, options.DocumentDir)
	a.unitStore = store.NewSQLUnitStore(a.tenantDB)
	a.annualStatementPeriods = store.NewSQLAnnualStatementPeriodStore(a.tenantDB)
	a.annualStatementCostTypes = store.NewSQLAnnualStatementCostTypeStore(a.tenantDB)
	a.annualStatementReceipts = store.NewSQLAnnualStatementReceiptStore(a.tenantDB)
	a.annualStatementAkontos = store.NewSQLAnnualStatementPrepaymentStore(a.tenantDB)
	a.annualConsumption = store.NewSQLAnnualStatementConsumptionStore(a.tenantDB)
	a.annualStatementRuns = store.NewSQLAnnualStatementRunStore(a.tenantDB, a.documentStore)
	a.annualStatementDeliveries = store.NewSQLAnnualStatementDeliveryStore(a.tenantDB)
	return a, a.repositoriesForTenant(identities[archiveDemoTenant].Ref()), func() error {
		_, err := demo.Load(t.Context(), database, "../../scripts/demo/seed", options)
		return err
	}
}

func archiveDemoRequest(t *testing.T, a *app, actor, method, path string, values url.Values) *httptest.ResponseRecorder {
	t.Helper()
	return archiveDemoRequestWithOrigin(t, a, actor, method, path, values, "http://hausv.org")
}

func archiveDemoRequestWithOrigin(t *testing.T, a *app, actor, method, path string, values url.Values, origin string) *httptest.ResponseRecorder {
	t.Helper()
	token, _, err := a.sessions.Put(actor, archiveDemoTenant, authMethodEmail, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(method, "http://hausv.org/"+archiveDemoTenant+path, strings.NewReader(values.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Origin", origin)
	r.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	w := httptest.NewRecorder()
	a.handler().ServeHTTP(w, r)
	return w
}

func createArchiveDemoRun(t *testing.T, a *app, repos requestRepositories) store.AnnualStatementRun {
	t.Helper()
	w := archiveDemoRequest(t, a, archiveDemoManager, http.MethodPost, "/app/settings/annual-statement/runs", url.Values{"year": {"2025"}})
	if w.Code != http.StatusSeeOther || !strings.Contains(w.Header().Get("Location"), "run-status=created") {
		t.Fatalf("create=%d %s", w.Code, w.Body.String())
	}
	runs, err := repos.annualStatementRuns.List(2025)
	if err != nil || len(runs) == 0 {
		t.Fatal("missing run", err)
	}
	w = archiveDemoRequest(t, a, archiveDemoManager, http.MethodPost, "/app/settings/annual-statement/runs/"+runs[0].ID+"/approve", nil)
	if w.Code != http.StatusSeeOther {
		t.Fatal(w.Code, w.Body.String())
	}
	run, _, err := repos.annualStatementRuns.Get(runs[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	return run
}

func replaceArchiveDemoDocument(t *testing.T, a *app, id string) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("id", id); err != nil {
		t.Fatal(err)
	}
	file, err := writer.CreateFormFile("document", "replacement.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("%PDF-1.4 changed")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	token, _, err := a.sessions.Put(archiveDemoManager, archiveDemoTenant, authMethodEmail, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, "http://hausv.org/"+archiveDemoTenant+"/app/dokumente/replace", &body)
	r.Header.Set("Content-Type", writer.FormDataContentType())
	r.Header.Set("Origin", "http://hausv.org")
	r.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	w := httptest.NewRecorder()
	a.handler().ServeHTTP(w, r)
	return w
}

func TestMusterstadt2025ArchiveThroughRoutes(t *testing.T) {
	a, repos, reseed := newArchiveDemoApp(t)
	if len(repos.documents.List()) != 9 {
		t.Fatal("seed must contain three original receipts and six portal documents")
	}
	run := createArchiveDemoRun(t, a, repos)
	if len(run.Result.Units) != 24 || len(run.Input.Parties) != 30 {
		t.Fatal("fixture incomplete")
	}
	route := "/app/settings/annual-statement/runs/" + run.ID + "/archive"
	pageRoute := "/app/settings/annual-statement?year=2025&run=" + run.ID
	beforePage := archiveDemoRequest(t, a, archiveDemoManager, http.MethodGet, pageRoute, nil)
	if !strings.Contains(beforePage.Body.String(), "Im Archiv ablegen") || len(repos.documents.List()) != 9 {
		t.Fatal("GET archive action missing or wrote documents")
	}
	response := archiveDemoRequest(t, a, archiveDemoManager, http.MethodPost, route, nil)
	if response.Code != http.StatusSeeOther || !strings.Contains(response.Header().Get("Location"), "run-status=archived") {
		t.Fatalf("archive=%d %s", response.Code, response.Header().Get("Location"))
	}
	archived := annualStatementArchiveDocuments(repos.documents, run)
	if len(archived) != 31 || len(repos.documents.List()) != 40 {
		t.Fatalf("archive=%d total=%d", len(archived), len(repos.documents.List()))
	}
	bytesBefore := map[string][]byte{}
	for _, item := range archived {
		metadata := item.AnnualStatementArchive
		if metadata.RunID != run.ID || metadata.Revision != 1 || metadata.PeriodYear != 2025 || metadata.ArchivedBy != archiveDemoManager || metadata.ArchivedAt.IsZero() || item.Visibility != store.DocumentVisibilityManagerOnly || item.Category != store.DocumentCategoryBilling || item.SeriesID != item.ID || item.Version != 1 || !item.Current {
			t.Fatalf("incorrect archive record: %+v", item)
		}
		path, _ := repos.documents.FilePath(item)
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		bytesBefore[item.ID] = raw
		pdf := archiveDemoRequest(t, a, archiveDemoManager, http.MethodGet, annualStatementPDFURL(run.ID, item.UnitID, metadata.PartyID), nil)
		download := archiveDemoRequest(t, a, archiveDemoManager, http.MethodGet, "/app/dokumente/"+item.ID+"/download", nil)
		if pdf.Code != 200 || download.Code != 200 || !bytes.Equal(raw, pdf.Body.Bytes()) || !bytes.Equal(raw, download.Body.Bytes()) || metadata.SHA256 != fmt.Sprintf("%x", sha256.Sum256(raw)) {
			t.Fatalf("archive differs from download for unit=%s party=%s", item.UnitID, metadata.PartyID)
		}
	}
	page := archiveDemoRequest(t, a, archiveDemoManager, http.MethodGet, pageRoute, nil)
	for _, want := range []string{"Archiviert am", "31 Dokumente", "Im Dokumentenarchiv ansehen", "Archiv für", "/app/dokumente?q=Abrechnung#document-"} {
		if !strings.Contains(page.Body.String(), want) {
			t.Errorf("run view missing %q", want)
		}
	}
	if strings.Contains(page.Body.String(), "Im Archiv ablegen") {
		t.Fatal("completed archive still offered")
	}
	library := archiveDemoRequest(t, a, archiveDemoManager, http.MethodGet, "/app/dokumente?q=Jahresabrechnung", nil)
	if strings.Count(library.Body.String(), "Archiviert · unveränderlich") != 31 {
		t.Fatal("archive library badges missing")
	}
	// The public demo summary is replaceable; each archived row must remain immutable.
	for _, item := range archived {
		marker := `<article class="document-row" id="document-` + item.ID + `">`
		_, row, found := strings.Cut(library.Body.String(), marker)
		row, _, closed := strings.Cut(row, "</article>")
		if !found || !closed || !strings.Contains(row, "Archiviert · unveränderlich") || strings.Contains(row, "Neue Version hochladen") {
			t.Fatalf("archive row %s is missing or mutable", item.ID)
		}
	}
	for _, want := range []string{"Für Alina Auer", `title="alina.eigentuemer@musterstadt.example"`} {
		if !strings.Contains(library.Body.String(), want) {
			t.Errorf("archive library missing %q", want)
		}
	}
	if strings.Contains(library.Body.String(), "Für alina.eigentuemer@") {
		t.Fatal("email displayed instead of snapshot name")
	}
	last := -1
	for _, unit := range store.AnnualStatementRunDisplayOrder(run) {
		label := "Jahresabrechnung 2025 · " + unit.Label + " · Lauf 1"
		index := strings.Index(library.Body.String(), label)
		if index <= last {
			t.Fatalf("archive library out of register order at %s", unit.Label)
		}
		last = index
	}
	events := a.auditStore.List(auditFilter{TenantSlug: archiveDemoTenant, Action: store.AuditActionAnnualRunArchive, Limit: 10})
	if len(events) != 1 || events[0].TargetID != run.ID || events[0].Details["run_id"] != run.ID || events[0].Details["revision"] != "1" || events[0].Details["document_count"] != "31" {
		t.Fatalf("audit=%+v", events)
	}
	response = archiveDemoRequest(t, a, archiveDemoManager, http.MethodPost, route, nil)
	if response.Code != http.StatusSeeOther || !reflect.DeepEqual(archived, annualStatementArchiveDocuments(repos.documents, run)) || len(repos.documents.List()) != 40 {
		t.Fatal("second archive changed documents")
	}
	for _, item := range archived {
		replaced := replaceArchiveDemoDocument(t, a, item.ID)
		if replaced.Code != http.StatusConflict || !strings.Contains(replaced.Body.String(), "weder ersetzt noch gelöscht") {
			t.Fatalf("replace=%d %s", replaced.Code, replaced.Body.String())
		}
		// No document-delete API exists. Probe conventional paths and the actual
		// download route with DELETE to prevent a future deletion bypass.
		for _, path := range []string{"/app/dokumente/delete", "/app/dokumente/" + item.ID, "/app/dokumente/" + item.ID + "/download"} {
			deleted := archiveDemoRequest(t, a, archiveDemoManager, http.MethodDelete, path, url.Values{"id": {item.ID}})
			if deleted.Code != http.StatusNotFound && deleted.Code != http.StatusMethodNotAllowed {
				t.Fatalf("delete=%d %s", deleted.Code, path)
			}
		}
	}
	postDelete := archiveDemoRequest(t, a, archiveDemoManager, http.MethodPost, "/app/dokumente/delete", url.Values{"id": {store.AnnualStatementArchiveID(run.ID, 1, "", "")}})
	if postDelete.Code != http.StatusNotFound && postDelete.Code != http.StatusMethodNotAllowed {
		t.Fatal("POST deletion not refused", postDelete.Code)
	}
	// A fresh revision keeps the complete previous archive visible and current.
	if _, _, err := repos.annualStatementAkontos.Save(store.AnnualStatementPrepayment{PeriodYear: 2025, UnitID: run.Result.Units[0].UnitID, AmountCents: 1, UpdatedBy: archiveDemoManager}); err != nil {
		t.Fatal(err)
	}
	next := createArchiveDemoRun(t, a, repos)
	response = archiveDemoRequest(t, a, archiveDemoManager, http.MethodPost, "/app/settings/annual-statement/runs/"+next.ID+"/archive", nil)
	if next.Revision != 2 || response.Code != http.StatusSeeOther || len(annualStatementArchiveDocuments(repos.documents, next)) != 31 || len(repos.documents.List()) != 71 {
		t.Fatal("new revision archive failed")
	}
	// Reset remains an input operation and cannot destroy previously archived files.
	if err := reseed(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(archived, annualStatementArchiveDocuments(repos.documents, run)) {
		t.Fatal("earlier revision metadata changed")
	}
	for id, raw := range bytesBefore {
		item, found := repos.documents.Get(id)
		path, _ := repos.documents.FilePath(item)
		after, err := os.ReadFile(path)
		if !found || err != nil || !bytes.Equal(after, raw) {
			t.Fatal("earlier revision bytes changed", err)
		}
	}
	t.Log("fresh SQLite seed → 24 units / 30 parties → 31 archived PDFs; byte-identical routes, retry adds zero; replace/delete refused; revision 2 and reseed preserve revision 1")
}

func TestAnnualStatementArchiveAuthorizationAndTenantIsolation(t *testing.T) {
	a, repos, _ := newArchiveDemoApp(t)
	run := createArchiveDemoRun(t, a, repos)
	route := "/app/settings/annual-statement/runs/" + run.ID + "/archive"
	for i, role := range []string{roleResident, roleOwner, roleServiceProvider} {
		actor := fmt.Sprintf("role-%d@example.com", i)
		a.profiles[actor] = userProfile{Email: actor, Role: role, Tenants: []string{archiveDemoTenant}, AuthMethods: defaultAuthMethods()}
		if w := archiveDemoRequest(t, a, actor, http.MethodPost, route, nil); w.Code != http.StatusForbidden {
			t.Fatalf("role %s archive=%d", role, w.Code)
		}
	}
	foreignOrigin := archiveDemoRequestWithOrigin(t, a, archiveDemoManager, http.MethodPost, route, nil, "https://foreign.example")
	if foreignOrigin.Code != http.StatusForbidden {
		t.Fatal("foreign origin accepted")
	}
	if w := archiveDemoRequest(t, a, archiveDemoManager, http.MethodGet, route, nil); w.Code != http.StatusMethodNotAllowed && w.Code != http.StatusNotFound {
		t.Fatal("GET archive accepted", w.Code)
	}
	if w := archiveDemoRequest(t, a, archiveDemoManager, http.MethodPost, "/app/settings/annual-statement/runs/missing/archive", nil); w.Code != http.StatusNotFound {
		t.Fatal("missing run accepted")
	}
	if len(repos.documents.List()) != 9 {
		t.Fatal("refused archive wrote documents")
	}
	foreignRepos := a.repositoriesForTenant(testTenantRef("other"))
	ac := authCtx{email: archiveDemoManager, role: roleManager, tenant: tenantConfig{Slug: "other"}, tenantRef: testTenantRef("other"), repositories: foreignRepos}
	req := httptest.NewRequest(http.MethodPost, route, nil)
	req.SetPathValue("runID", run.ID)
	w := httptest.NewRecorder()
	a.archiveAnnualStatementRun(w, req, ac)
	if w.Code != http.StatusNotFound {
		t.Fatal("foreign run accessible", w.Code)
	}
	archiveDemoRequest(t, a, archiveDemoManager, http.MethodPost, route, nil)
	for _, party := range run.Input.Parties {
		role := roleOwner
		if party.Renter {
			role = roleResident
		}
		a.profiles[party.ID] = userProfile{Email: party.ID, Role: role, Tenants: []string{archiveDemoTenant}, AuthMethods: defaultAuthMethods()}
	}
	privacyActors := map[string]string{}
	for _, party := range run.Input.Parties {
		if party.Owner {
			privacyActors["owner"] = party.ID
		}
		if party.Renter {
			privacyActors["renter"] = party.ID
		}
	}
	if len(privacyActors) != 2 {
		t.Fatal("fixture must cover owners and renters")
	}
	for _, actor := range privacyActors {
		page := archiveDemoRequest(t, a, actor, http.MethodGet, "/app/dokumente", nil)
		if page.Code != http.StatusOK || strings.Contains(page.Body.String(), "Archiviert · unveränderlich") {
			t.Fatal("resident archive listing leak", page.Code)
		}
		for _, item := range annualStatementArchiveDocuments(repos.documents, run) {
			if strings.Contains(page.Body.String(), item.ID) || strings.Contains(page.Body.String(), item.Title) {
				t.Fatal("resident archive metadata leaked")
			}
			for _, suffix := range []string{"/download", "/preview"} {
				w := archiveDemoRequest(t, a, actor, http.MethodGet, "/app/dokumente/"+item.ID+suffix, nil)
				if w.Code != http.StatusForbidden {
					t.Fatalf("resident read archive: %d", w.Code)
				}
			}
		}
	}
}

type interruptedArchiveRepository struct {
	store.DocumentRepository
	remaining int
}

func (r *interruptedArchiveRepository) CreateGenerated(item store.DocumentRecord, filename, contentType string, data []byte, now time.Time) (store.DocumentRecord, error) {
	if r.remaining == 0 {
		return store.DocumentRecord{}, errors.New("injected archive storage failure")
	}
	r.remaining--
	return r.DocumentRepository.CreateGenerated(item, filename, contentType, data, now)
}

type archiveSnapshotRepository struct {
	store.AnnualStatementRunRepository
	run store.AnnualStatementRun
}

func (r archiveSnapshotRepository) Get(string) (store.AnnualStatementRun, bool, error) {
	return r.run, true, nil
}

func TestAnnualStatementArchiveIncompleteRunAndInterruptedAttempt(t *testing.T) {
	a, repos, _ := newArchiveDemoApp(t)
	run := createArchiveDemoRun(t, a, repos)
	ac := authCtx{email: archiveDemoManager, role: roleManager, tenant: a.tenants[archiveDemoTenant], tenantRef: a.tenantIdentities[archiveDemoTenant].Ref(), repositories: repos}
	request := func() *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, "/app/settings/annual-statement/runs/"+run.ID+"/archive", nil)
		r.SetPathValue("runID", run.ID)
		w := httptest.NewRecorder()
		a.archiveAnnualStatementRun(w, r, ac)
		return w
	}
	incomplete := run
	incomplete.Input.Parties = nil
	ac.repositories.annualStatementRuns = archiveSnapshotRepository{AnnualStatementRunRepository: repos.annualStatementRuns, run: incomplete}
	if w := request(); w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "fehlen gespeicherte Parteien") {
		t.Fatal("incomplete run archived", w.Code)
	}
	if len(repos.documents.List()) != 9 {
		t.Fatal("incomplete run wrote documents")
	}
	ac.repositories.annualStatementRuns = repos.annualStatementRuns
	ac.repositories.documents = &interruptedArchiveRepository{DocumentRepository: repos.documents, remaining: 2}
	if w := request(); !strings.Contains(w.Header().Get("Location"), "run-status=archive-error") {
		t.Fatal("storage failure not reported")
	}
	partial := annualStatementArchiveDocuments(repos.documents, run)
	if len(partial) != 2 {
		t.Fatal("partial documents not preserved")
	}
	page := annualStatementRunView(repos.annualStatementRuns, repos.documents, 2025, run.ID, "archive-error", nil)
	if page.ArchivedAt != "" || page.ArchiveCount != 0 || !strings.Contains(page.Message, "nicht abgeschlossen") {
		t.Fatal("partial archive reported complete")
	}
	if len(a.auditStore.List(auditFilter{TenantSlug: archiveDemoTenant, Action: store.AuditActionAnnualRunArchive})) != 0 {
		t.Fatal("failed archive logged as complete")
	}
	ac.repositories.documents = repos.documents
	if w := request(); !strings.Contains(w.Header().Get("Location"), "run-status=archived") {
		t.Fatal("archive retry failed")
	}
	if len(annualStatementArchiveDocuments(repos.documents, run)) != 31 {
		t.Fatal("retry did not finish the set")
	}
	for id, before := range partial {
		after, found := repos.documents.Get(id)
		if !found || !reflect.DeepEqual(before, after) {
			t.Fatal("retry replaced a partial archive copy")
		}
	}
}
