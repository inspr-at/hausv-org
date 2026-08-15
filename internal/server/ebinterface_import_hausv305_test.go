package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	appdb "github.com/inspr-at/hausv-org/internal/db"
)

func TestEBInterfacePortalPreviewStoreProtectionAndIdempotency(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{
		Email:       "resident@example.com",
		Role:        roleResident,
		Tenants:     []string{"demo"},
		AuthMethods: defaultAuthMethods(),
	}
	data, err := os.ReadFile("../integrations/testdata/ebinterface-6p0.xml")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	page := authedRequest(t, a, "manager@example.com", "/demo/app/dokumente/rechnungen/import")
	if page.Code != http.StatusOK {
		t.Fatalf("import page status = %d", page.Code)
	}
	for _, want := range []string{"E-Rechnung ablegen", "ebInterface 5.0 und 6.0", "Vorschau erstellen", "nicht extern gesendet"} {
		if !strings.Contains(page.Body.String(), want) {
			t.Fatalf("import page missing %q:\n%s", want, page.Body.String())
		}
	}

	preview := authedMultipartFileRequest(t, a, "manager@example.com", "/demo/app/dokumente/rechnungen/import/preview", nil, "invoice_file", "rechnung.xml", data)
	token := ebInterfaceImportPreviewToken(t, preview)
	previewPage := authedRequest(t, a, "manager@example.com", "/demo/app/dokumente/rechnungen/import?preview="+url.QueryEscape(token))
	if previewPage.Code != http.StatusOK {
		t.Fatalf("preview page status = %d", previewPage.Code)
	}
	body := previewPage.Body.String()
	for _, want := range []string{
		"rechnung.xml", "ebInterface 6.0", "RE-2026-0006", "Hausservice Beispiel e.U.",
		"demo", "99,90 €", "09.07.2026", "23.07.2026", "02.07.2026 – 09.07.2026",
		"Geschützt ablegen", "keine Buchung oder Zahlung",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("preview missing %q:\n%s", want, body)
		}
	}
	for _, forbidden := range []string{"ATU00000003", "Musterstraße 6", "Synthetische Hausserviceleistung", "<Invoice"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("preview leaks source detail %q:\n%s", forbidden, body)
		}
	}

	a.ebInterfaceImportMu.Lock()
	storedPreview := a.ebInterfaceImportPreviews[token]
	a.ebInterfaceImportMu.Unlock()
	if string(storedPreview.RawXML) != string(data) || storedPreview.TenantSlug != "demo" {
		t.Fatalf("stored preview is not exact and tenant-bound: tenant=%q bytes=%d", storedPreview.TenantSlug, len(storedPreview.RawXML))
	}

	store := authedFormRequest(t, a, "manager@example.com", "/demo/app/dokumente/rechnungen/import/store", url.Values{
		"preview_token": {token},
	})
	if store.Code != http.StatusSeeOther || !strings.Contains(store.Header().Get("Location"), "doc=invoice-imported#document-") {
		t.Fatalf("store status=%d location=%q body=%s", store.Code, store.Header().Get("Location"), store.Body.String())
	}
	documents := documentRepositoryForTest(a, "demo").List()
	if len(documents) != 1 {
		t.Fatalf("documents = %+v", documents)
	}
	created := documents[0]
	if created.Visibility != documentVisibilityManagerOnly || created.Category != documentCategoryBilling ||
		created.ContentType != "application/xml" || !strings.Contains(created.Title, "RE-2026-0006") {
		t.Fatalf("stored document metadata = %+v", created)
	}
	storedPath, ok := documentRepositoryForTest(a, "demo").FilePath(created)
	if !ok {
		t.Fatal("stored document path missing")
	}
	storedData, err := os.ReadFile(storedPath)
	if err != nil {
		t.Fatalf("read stored document: %v", err)
	}
	if string(storedData) != string(data) {
		t.Fatal("stored original XML changed")
	}
	importEvents := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionIntegrationImport, Limit: 20})
	if len(importEvents) != 1 || importEvents[0].Details["document_id"] != created.ID ||
		importEvents[0].Details["visibility"] != documentVisibilityLabel(documentVisibilityManagerOnly) {
		t.Fatalf("import audit = %+v", importEvents)
	}

	residentPage := authedRequest(t, a, "resident@example.com", "/demo/app/dokumente")
	if strings.Contains(residentPage.Body.String(), "RE-2026-0006") || strings.Contains(residentPage.Body.String(), created.Filename) {
		t.Fatalf("resident document list exposes E-Rechnung:\n%s", residentPage.Body.String())
	}
	residentDownload := authedRequest(t, a, "resident@example.com", "/demo/app/dokumente/"+created.ID+"/download")
	if residentDownload.Code != http.StatusForbidden {
		t.Fatalf("resident download status = %d, want 403", residentDownload.Code)
	}
	managerDownload := authedRequest(t, a, "manager@example.com", "/demo/app/dokumente/"+created.ID+"/download")
	if managerDownload.Code != http.StatusOK || managerDownload.Body.String() != string(data) {
		t.Fatalf("manager download status=%d bytes=%d", managerDownload.Code, managerDownload.Body.Len())
	}

	secondPreview := authedMultipartFileRequest(t, a, "manager@example.com", "/demo/app/dokumente/rechnungen/import/preview", nil, "invoice_file", "rechnung-nochmal.xml", data)
	secondToken := ebInterfaceImportPreviewToken(t, secondPreview)
	secondStore := authedFormRequest(t, a, "manager@example.com", "/demo/app/dokumente/rechnungen/import/store", url.Values{
		"preview_token": {secondToken},
	})
	if secondStore.Code != http.StatusSeeOther || !strings.Contains(secondStore.Header().Get("Location"), "result=already") {
		t.Fatalf("repeated store status=%d location=%q", secondStore.Code, secondStore.Header().Get("Location"))
	}
	if got := len(documentRepositoryForTest(a, "demo").List()); got != 1 {
		t.Fatalf("repeated import created %d documents", got)
	}
	if got := len(a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionIntegrationImport, Limit: 20})); got != 1 {
		t.Fatalf("repeated import created %d import audits", got)
	}
}

func TestEBInterfacePortalRejectsUnauthorizedCrossOriginAndInvalidInput(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	if page := authedRequest(t, a, "resident@example.com", "/demo/app/dokumente/rechnungen/import"); page.Code != http.StatusForbidden {
		t.Fatalf("resident import page status = %d, want 403", page.Code)
	}
	if post := ebInterfaceMultipartRequest(t, a, "resident@example.com", "http://hausv.org/demo", "invoice.xml", []byte("<Invoice/>")); post.Code != http.StatusForbidden {
		t.Fatalf("resident preview status = %d, want 403", post.Code)
	}
	adminApp := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	if page := authedRequest(t, adminApp, "admin@example.com", "/demo/app/dokumente/rechnungen/import"); page.Code != http.StatusOK {
		t.Fatalf("admin import page status = %d, want 200", page.Code)
	}

	a = newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	if cross := ebInterfaceMultipartRequest(t, a, "manager@example.com", "https://evil.example", "invoice.xml", []byte("<Invoice/>")); cross.Code != http.StatusForbidden {
		t.Fatalf("cross-origin preview status = %d, want 403", cross.Code)
	}
	invalid := authedMultipartFileRequest(t, a, "manager@example.com", "/demo/app/dokumente/rechnungen/import/preview", nil, "invoice_file", "invoice.txt", []byte("not xml"))
	if invalid.Code != http.StatusSeeOther || !strings.Contains(invalid.Header().Get("Location"), "result=invalid") {
		t.Fatalf("invalid input status=%d location=%q", invalid.Code, invalid.Header().Get("Location"))
	}
	oversized := bytes.Repeat([]byte("x"), maxEBInterfaceImportBytes+1)
	tooLarge := authedMultipartFileRequest(t, a, "manager@example.com", "/demo/app/dokumente/rechnungen/import/preview", nil, "invoice_file", "invoice.xml", oversized)
	if tooLarge.Code != http.StatusSeeOther || !strings.Contains(tooLarge.Header().Get("Location"), "result=invalid") {
		t.Fatalf("oversized input status=%d location=%q", tooLarge.Code, tooLarge.Header().Get("Location"))
	}
}

func TestEBInterfacePortalExplainsUnsupportedAndInvalidProfiles(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	for _, tc := range []struct {
		name string
		xml  string
		want string
	}{
		{"6.1", `<Invoice xmlns="http://www.ebinterface.at/schema/6p1/"><InvoiceNumber>FUTURE</InvoiceNumber></Invoice>`, "Dieses ebInterface-Profil wird noch nicht unterstützt."},
		{"invalid", `<Invoice xmlns="http://www.ebinterface.at/schema/6p0/"><InvoiceNumber>BAD</InvoiceNumber></Invoice>`, "Der Bruttobetrag oder die Währung fehlt oder ist ungültig."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := authedMultipartFileRequest(t, a, "manager@example.com", "/demo/app/dokumente/rechnungen/import/preview", nil, "invoice_file", tc.name+".xml", []byte(tc.xml))
			token := ebInterfaceImportPreviewToken(t, response)
			page := authedRequest(t, a, "manager@example.com", "/demo/app/dokumente/rechnungen/import?preview="+url.QueryEscape(token))
			if !strings.Contains(page.Body.String(), tc.want) || !strings.Contains(page.Body.String(), "Ablage nicht möglich") ||
				strings.Contains(page.Body.String(), `type="submit">Geschützt ablegen`) {
				t.Fatalf("%s preview is unclear:\n%s", tc.name, page.Body.String())
			}
		})
	}
}

func TestEBInterfacePreviewIsTenantBoundAndExpires(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	preview := ebInterfaceImportPreview{
		TenantSlug: "demo",
		CreatedAt:  time.Now().UTC(),
		RawXML:     []byte("<Invoice/>"),
	}
	token, err := a.storeEBInterfaceImportPreview(preview)
	if err != nil {
		t.Fatalf("store preview: %v", err)
	}
	if _, ok := a.ebInterfaceImportPreview(token, "other-house"); ok {
		t.Fatal("preview crossed tenant boundary")
	}
	a.ebInterfaceImportMu.Lock()
	expired := a.ebInterfaceImportPreviews[token]
	expired.CreatedAt = expired.CreatedAt.Add(-ebInterfaceImportPreviewTTL - 1)
	a.ebInterfaceImportPreviews[token] = expired
	a.ebInterfaceImportMu.Unlock()
	if _, ok := a.ebInterfaceImportPreview(token, "demo"); ok {
		t.Fatal("expired preview remained available")
	}
}

func TestEBInterfaceImportLedgerIsDurableAndTenantBound(t *testing.T) {
	database, err := appdb.Open(filepath.Join(t.TempDir(), "hausv.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	data := []byte("synthetic ebInterface invoice")
	digestBytes := sha256.Sum256(data)
	digest := hex.EncodeToString(digestBytes[:])
	a := &app{db: database}
	preview := ebInterfaceImportPreview{FileDigest: digest, SourceVersion: "6.0"}
	if err := a.recordEBInterfaceImportLedger("demo", "manager@example.com", preview); err != nil {
		t.Fatalf("record ledger: %v", err)
	}
	if !a.ebInterfaceImportAlreadyStored("demo", digest) {
		t.Fatal("same tenant and digest not found in durable ledger")
	}
	if a.ebInterfaceImportAlreadyStored("other-house", digest) {
		t.Fatal("digest leaked across tenant boundary")
	}
	if err := a.recordEBInterfaceImportLedger("demo", "manager@example.com", preview); err != nil {
		t.Fatalf("repeat ledger: %v", err)
	}
	var rows int
	if err := database.QueryRow(
		`SELECT COUNT(*) FROM integration_imports WHERE tenant_slug = ? AND format = ? AND file_digest = ?`,
		"demo", "ebinterface", digest,
	).Scan(&rows); err != nil {
		t.Fatalf("count ledger: %v", err)
	}
	if rows != 1 {
		t.Fatalf("ledger rows = %d, want 1", rows)
	}
}

func ebInterfaceImportPreviewToken(t *testing.T, response *httptest.ResponseRecorder) string {
	t.Helper()
	if response.Code != http.StatusSeeOther {
		t.Fatalf("preview status = %d, want redirect; body=%s", response.Code, response.Body.String())
	}
	location, err := url.Parse(response.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse preview location: %v", err)
	}
	token := location.Query().Get("preview")
	if token == "" {
		t.Fatalf("preview redirect has no token: %q", response.Header().Get("Location"))
	}
	return token
}

func ebInterfaceMultipartRequest(t *testing.T, a *app, email, origin, filename string, fileBody []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("invoice_file", filename)
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write(fileBody); err != nil {
		t.Fatalf("write multipart: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}
	token, _, err := a.sessions.Put(email, "demo", authMethodEmail, time.Hour)
	if err != nil {
		t.Fatalf("put session: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "http://hausv.org/demo/app/dokumente/rechnungen/import/preview", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Origin", origin)
	req.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	response := httptest.NewRecorder()
	a.handler().ServeHTTP(response, req)
	return response
}
