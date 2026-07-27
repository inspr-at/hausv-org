package server

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	appdb "github.com/markus-barta/hausv-org/internal/db"
)

func TestCAMT053PortalPreviewApplyAndIdempotency(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	if err := a.unitStore.SetTenantUnits("jhw22", []unit{{
		ID:         "top-1",
		TenantSlug: "jhw22",
		Label:      "Top 1",
		UnitType:   unitTypeResidential,
	}}); err != nil {
		t.Fatalf("SetTenantUnits: %v", err)
	}
	period := "2026-07"
	candidates, err := unitPaymentReferenceCandidates("jhw22", period, a.unitStore.ListTenant("jhw22"), nil)
	if err != nil || len(candidates) != 1 {
		t.Fatalf("reference candidates = %+v err=%v", candidates, err)
	}
	xml := testCAMT053XML("urn:iso:std:iso:20022:tech:xsd:camt.053.001.08", candidates[0].Reference)

	page := authedRequest(t, a, "manager@example.com", "/app/settings/payments/import?period="+period)
	if page.Code != http.StatusOK {
		t.Fatalf("import page status = %d", page.Code)
	}
	for _, want := range []string{"Zahlungen aus Bankdatei", "Erst prüfen, dann übernehmen", "Top 1", candidates[0].Reference, "camt.053.001.02", "und .001.08"} {
		if !strings.Contains(page.Body.String(), want) {
			t.Fatalf("import page missing %q:\n%s", want, page.Body.String())
		}
	}

	preview := authedMultipartFileRequest(t, a, "manager@example.com", "/app/settings/payments/import/preview", map[string]string{
		"period": period,
	}, "camt_file", "kontoauszug.xml", []byte(xml))
	token := paymentImportPreviewToken(t, preview)
	previewPage := authedRequest(t, a, "manager@example.com", "/app/settings/payments/import?period="+period+"&preview="+url.QueryEscape(token))
	if previewPage.Code != http.StatusOK {
		t.Fatalf("preview page status = %d", previewPage.Code)
	}
	body := previewPage.Body.String()
	for _, want := range []string{"kontoauszug.xml", "2019/camt.053.001.08", "Zuordnen", "Top 1", candidates[0].Reference, "42,00 €", "Eindeutige Treffer übernehmen"} {
		if !strings.Contains(body, want) {
			t.Fatalf("preview missing %q:\n%s", want, body)
		}
	}
	for _, forbidden := range []string{"Max Geheim", "AT483200000012345864", "NTR-SECRET"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("preview leaks %q:\n%s", forbidden, body)
		}
	}
	a.paymentImportMu.Lock()
	stored := a.paymentImportPreviews[token]
	a.paymentImportMu.Unlock()
	if len(stored.Payments) != 1 || stored.Payments[0].DebtorName != "" || stored.Payments[0].DebtorIBAN != "" || len(stored.Payments[0].RemittanceLines) != 0 {
		t.Fatalf("preview retained private bank fields: %+v", stored.Payments)
	}

	apply := authedFormRequest(t, a, "manager@example.com", "/app/settings/payments/import/apply", url.Values{
		"preview_token": {token},
	})
	if apply.Code != http.StatusSeeOther || !strings.Contains(apply.Header().Get("Location"), "result=applied") || !strings.Contains(apply.Header().Get("Location"), "changed=1") {
		t.Fatalf("apply status=%d location=%q", apply.Code, apply.Header().Get("Location"))
	}
	status, ok := a.unitPaymentStore.Get("jhw22", "top-1")
	if !ok || status.Status != unitPaymentStatusPaid || status.UpdatedBy != "manager@example.com" {
		t.Fatalf("payment status after import = %+v ok=%v", status, ok)
	}
	unitEvents := a.auditStore.List(auditFilter{TenantSlug: "jhw22", Action: auditActionUnitPayment, Limit: 20})
	importEvents := a.auditStore.List(auditFilter{TenantSlug: "jhw22", Action: auditActionIntegrationImport, Limit: 20})
	if len(unitEvents) != 1 || len(importEvents) != 1 || importEvents[0].Details["changed"] != "1" {
		t.Fatalf("audit after import unit=%+v import=%+v", unitEvents, importEvents)
	}

	secondPreview := authedMultipartFileRequest(t, a, "manager@example.com", "/app/settings/payments/import/preview", map[string]string{
		"period": period,
	}, "camt_file", "kontoauszug.xml", []byte(xml))
	secondToken := paymentImportPreviewToken(t, secondPreview)
	secondApply := authedFormRequest(t, a, "manager@example.com", "/app/settings/payments/import/apply", url.Values{
		"preview_token": {secondToken},
	})
	if secondApply.Code != http.StatusSeeOther || !strings.Contains(secondApply.Header().Get("Location"), "result=already") {
		t.Fatalf("second apply status=%d location=%q", secondApply.Code, secondApply.Header().Get("Location"))
	}
	if got := len(a.auditStore.List(auditFilter{TenantSlug: "jhw22", Action: auditActionUnitPayment, Limit: 20})); got != 1 {
		t.Fatalf("repeated import duplicated unit audit: %d", got)
	}
	if got := len(a.auditStore.List(auditFilter{TenantSlug: "jhw22", Action: auditActionIntegrationImport, Limit: 20})); got != 1 {
		t.Fatalf("repeated import duplicated integration audit: %d", got)
	}
}

func TestCAMT053PortalRejectsUnauthorizedCrossOriginAndInvalidInput(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	if page := authedRequest(t, a, "resident@example.com", "/app/settings/payments/import"); page.Code != http.StatusForbidden {
		t.Fatalf("resident import page status = %d, want 403", page.Code)
	}
	if post := paymentImportMultipartRequest(t, a, "resident@example.com", "http://jhw22.hausv.org", []byte("<Document/>")); post.Code != http.StatusForbidden {
		t.Fatalf("resident import preview status = %d, want 403", post.Code)
	}

	a = newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	if err := a.unitStore.SetTenantUnits("jhw22", []unit{{ID: "top-1", TenantSlug: "jhw22", Label: "Top 1", UnitType: unitTypeResidential}}); err != nil {
		t.Fatalf("SetTenantUnits: %v", err)
	}
	if cross := paymentImportMultipartRequest(t, a, "manager@example.com", "https://evil.example", []byte("<Document/>")); cross.Code != http.StatusForbidden {
		t.Fatalf("cross-origin import preview status = %d, want 403", cross.Code)
	}
	invalid := authedMultipartFileRequest(t, a, "manager@example.com", "/app/settings/payments/import/preview", map[string]string{
		"period": "2026-07",
	}, "camt_file", "kontoauszug.txt", []byte("not xml"))
	if invalid.Code != http.StatusSeeOther || !strings.Contains(invalid.Header().Get("Location"), "result=invalid") {
		t.Fatalf("invalid import status=%d location=%q", invalid.Code, invalid.Header().Get("Location"))
	}
	oversized := bytes.Repeat([]byte("x"), maxCAMTImportBytes+1)
	tooLarge := authedMultipartFileRequest(t, a, "manager@example.com", "/app/settings/payments/import/preview", map[string]string{
		"period": "2026-07",
	}, "camt_file", "kontoauszug.xml", oversized)
	if tooLarge.Code != http.StatusSeeOther || !strings.Contains(tooLarge.Header().Get("Location"), "result=invalid") {
		t.Fatalf("oversized import status=%d location=%q", tooLarge.Code, tooLarge.Header().Get("Location"))
	}
}

func TestCAMT053PortalShowsUnsupportedProfileWithoutApply(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	if err := a.unitStore.SetTenantUnits("jhw22", []unit{{ID: "top-1", TenantSlug: "jhw22", Label: "Top 1", UnitType: unitTypeResidential}}); err != nil {
		t.Fatalf("SetTenantUnits: %v", err)
	}
	preview := authedMultipartFileRequest(t, a, "manager@example.com", "/app/settings/payments/import/preview", map[string]string{
		"period": "2026-07",
	}, "camt_file", "future.xml", []byte(`<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.14"><BkToCstmrStmt/></Document>`))
	token := paymentImportPreviewToken(t, preview)
	page := authedRequest(t, a, "manager@example.com", "/app/settings/payments/import?period=2026-07&preview="+url.QueryEscape(token))
	body := page.Body.String()
	for _, want := range []string{"Nicht unterstützt", "Abgelehnt", "Dieses camt.053-Profil wird nicht unterstützt", "Keine Übernahme möglich"} {
		if !strings.Contains(body, want) {
			t.Fatalf("unsupported profile page missing %q:\n%s", want, body)
		}
	}
}

func TestCAMT053ImportLedgerIsDurableAndTenantBound(t *testing.T) {
	database, err := appdb.Open(filepath.Join(t.TempDir(), "hausv.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	a := &app{db: database}
	preview := camtImportPreview{
		FileDigest:    strings.Repeat("a", 64),
		SourceVersion: "2019/camt.053.001.08",
	}
	report := unitPaymentImportReport{Assigned: 2, Changed: 1, Unclear: 1}

	if err := a.recordPaymentImportLedger("jhw22", "manager@example.com", preview, report); err != nil {
		t.Fatalf("record ledger: %v", err)
	}
	if !a.paymentImportAlreadyApplied("jhw22", preview.FileDigest) {
		t.Fatal("same tenant and digest not found in durable ledger")
	}
	if a.paymentImportAlreadyApplied("other-house", preview.FileDigest) {
		t.Fatal("digest leaked across tenant boundary")
	}
	if err := a.recordPaymentImportLedger("jhw22", "manager@example.com", preview, report); err != nil {
		t.Fatalf("repeat record ledger: %v", err)
	}
	var rows int
	if err := database.QueryRow(
		`SELECT COUNT(*) FROM integration_imports WHERE tenant_slug = ? AND format = ? AND file_digest = ?`,
		"jhw22", "camt.053", preview.FileDigest,
	).Scan(&rows); err != nil {
		t.Fatalf("count ledger: %v", err)
	}
	if rows != 1 {
		t.Fatalf("ledger rows = %d, want 1", rows)
	}
}

func paymentImportPreviewToken(t *testing.T, response *httptest.ResponseRecorder) string {
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

func paymentImportMultipartRequest(t *testing.T, a *app, email, origin string, fileBody []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("period", "2026-07"); err != nil {
		t.Fatalf("WriteField: %v", err)
	}
	part, err := writer.CreateFormFile("camt_file", "kontoauszug.xml")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write(fileBody); err != nil {
		t.Fatalf("write multipart body: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}
	token, _, err := a.sessions.Put(email, "jhw22", authMethodEmail, time.Hour)
	if err != nil {
		t.Fatalf("put session: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "http://jhw22.hausv.org/app/settings/payments/import/preview", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Origin", origin)
	req.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	response := httptest.NewRecorder()
	a.handler().ServeHTTP(response, req)
	return response
}

func testCAMT053XML(namespace, reference string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="%s">
  <BkToCstmrStmt><Stmt><Id>OTHER</Id><Acct><Ownr><Nm>Fremder Kontoinhaber</Nm></Ownr></Acct>
    <Ntry><NtryRef>NTR-SECRET</NtryRef><Amt Ccy="EUR">42.00</Amt><CdtDbtInd>CRDT</CdtDbtInd><BookgDt><Dt>2026-07-08</Dt></BookgDt>
      <NtryDtls><TxDtls><Refs><EndToEndId>%s</EndToEndId></Refs>
        <RltdPties><Dbtr><Nm>Max Geheim</Nm></Dbtr><DbtrAcct><Id><IBAN>AT483200000012345864</IBAN></Id></DbtrAcct></RltdPties>
        <RmtInf><Ustrd>Zahlung %s</Ustrd></RmtInf>
      </TxDtls></NtryDtls>
    </Ntry>
  </Stmt></BkToCstmrStmt>
</Document>`, namespace, reference, reference)
}
