package server

import (
	"database/sql"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	appdb "github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/dbtest"
	"github.com/inspr-at/hausv-org/internal/store"
)

func sqlImportPortal(t *testing.T) (*app, *sql.DB) {
	t.Helper()
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	ledgerApp, database, refs := ledgerTestApp(t, "demo")
	a.tenantDB, a.scopedDB = ledgerApp.tenantDB, ledgerApp.scopedDB
	a.tenantIdentities["demo"] = store.TenantIdentity{ID: refs["demo"].ID, Slug: "demo", Name: "Demo"}
	a.unitPaymentStore = store.NewSQLUnitPaymentStatusStore(a.tenantDB)
	a.documentStore = store.NewSQLDocumentStore(a.tenantDB, t.TempDir())
	return a, database
}

func TestCAMTSQLPortalRollsBackLateFailureAndRetries(t *testing.T) {
	a, database := sqlImportPortal(t)
	if err := testUnitRepository(t, a, "demo").SetUnits([]unit{{ID: "top-1", TenantSlug: "demo", Label: "Top 1", UnitType: unitTypeResidential}}); err != nil {
		t.Fatal(err)
	}
	candidates, err := unitPaymentReferenceCandidates("demo", "2026-07", testUnitRepository(t, a, "demo").List(), nil)
	if err != nil {
		t.Fatal(err)
	}
	data := []byte(testCAMT053XML("urn:iso:std:iso:20022:tech:xsd:camt.053.001.08", candidates[0].Reference))
	preview := authedMultipartFileRequest(t, a, "manager@example.com", "/demo/app/settings/payments/import/preview", map[string]string{"period": "2026-07"}, "camt_file", "bank.xml", data)
	token := paymentImportPreviewToken(t, preview)
	// The count update happens after all status effects. Refusing that update
	// exercises the real handler's late-error path, not just a mock callback.
	trigger := `CREATE TRIGGER reject_import_counts BEFORE UPDATE ON integration_imports BEGIN SELECT RAISE(ABORT, 'injected'); END`
	if dbtest.Backend() == appdb.BackendPostgres {
		trigger = `CREATE FUNCTION reject_import_counts_fn() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected'; END $$;
		CREATE TRIGGER reject_import_counts BEFORE UPDATE ON integration_imports FOR EACH ROW EXECUTE FUNCTION reject_import_counts_fn()`
	}
	if _, err := database.Exec(trigger); err != nil {
		t.Fatal(err)
	}
	apply := func() string {
		response := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/payments/import/apply", url.Values{"preview_token": {token}})
		if response.Code != http.StatusSeeOther {
			t.Fatalf("status=%d", response.Code)
		}
		return response.Header().Get("Location")
	}
	if location := apply(); !strings.Contains(location, "result=error") {
		t.Fatalf("failure redirect=%s", location)
	}
	for _, table := range []string{"unit_payment_status", "integration_imports"} {
		var n int
		if err := database.QueryRow(`SELECT count(*) FROM ` + table).Scan(&n); err != nil || n != 0 {
			t.Fatalf("%s rows=%d err=%v", table, n, err)
		}
	}
	if got := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionIntegrationImport}); len(got) != 0 {
		t.Fatal("failed import emitted completion audit")
	}
	drop := `DROP TRIGGER reject_import_counts`
	if dbtest.Backend() == appdb.BackendPostgres {
		drop += ` ON integration_imports`
	}
	if _, err := database.Exec(drop); err != nil {
		t.Fatal(err)
	}
	if location := apply(); !strings.Contains(location, "result=applied") || !strings.Contains(location, "changed=1") {
		t.Fatalf("retry=%s", location)
	}
	preview = authedMultipartFileRequest(t, a, "manager@example.com", "/demo/app/settings/payments/import/preview", map[string]string{"period": "2026-07"}, "camt_file", "bank.xml", data)
	token = paymentImportPreviewToken(t, preview)
	// Losing the separate audit file must not reopen a committed SQL import.
	a.auditStore, err = newAuditStore("")
	if err != nil {
		t.Fatal(err)
	}
	if location := apply(); !strings.Contains(location, "result=already") {
		t.Fatalf("repeat=%s", location)
	}
}

func TestEBInterfaceSQLPortalPendingRecovery(t *testing.T) {
	a, database := sqlImportPortal(t)
	dir := t.TempDir()
	blocked := filepath.Join(dir, "blocked")
	if err := os.WriteFile(blocked, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	a.documentStore = store.NewSQLDocumentStore(a.tenantDB, blocked)
	data, err := os.ReadFile("../integrations/testdata/ebinterface-6p0.xml")
	if err != nil {
		t.Fatal(err)
	}
	preview := authedMultipartFileRequest(t, a, "manager@example.com", "/demo/app/dokumente/rechnungen/import/preview", nil, "invoice_file", "invoice.xml", data)
	token := ebInterfaceImportPreviewToken(t, preview)
	apply := func() string {
		response := authedFormRequest(t, a, "manager@example.com", "/demo/app/dokumente/rechnungen/import/store", url.Values{"preview_token": {token}})
		if response.Code != http.StatusSeeOther {
			t.Fatalf("status=%d", response.Code)
		}
		return response.Header().Get("Location")
	}
	if location := apply(); !strings.Contains(location, "result=error") {
		t.Fatalf("failure redirect=%s", location)
	}
	var status, key string
	if err := database.QueryRow(`SELECT status, blob_key FROM integration_imports`).Scan(&status, &key); err != nil || status != "pending" || key == "" {
		t.Fatalf("reservation: %s %s %v", status, key, err)
	}
	if got := sqlImportDocuments(a).List(); len(got) != 0 {
		t.Fatal("pending invoice exposed")
	}
	// Recreate the preview and store, as happens after restart and re-upload.
	a.documentStore = store.NewSQLDocumentStore(a.tenantDB, dir)
	a.ebInterfaceImportPreviews = nil
	preview = authedMultipartFileRequest(t, a, "manager@example.com", "/demo/app/dokumente/rechnungen/import/preview", nil, "invoice_file", "invoice.xml", data)
	token = ebInterfaceImportPreviewToken(t, preview)
	if location := apply(); !strings.Contains(location, "doc=invoice-imported#document-") {
		t.Fatalf("recovery=%s", location)
	}
	if err := database.QueryRow(`SELECT status, blob_key FROM integration_imports`).Scan(&status, &key); err != nil || status != "complete" {
		t.Fatalf("completion=%s %v", status, err)
	}
	documents := sqlImportDocuments(a).List()
	if len(documents) != 1 || documents[0].StoredFilename != key {
		t.Fatalf("documents=%+v", documents)
	}
	stored, err := os.ReadFile(filepath.Join(dir, "demo", key))
	if err != nil || string(stored) != string(data) {
		t.Fatalf("stored XML differs: %v", err)
	}
	a.auditStore, err = newAuditStore("")
	if err != nil {
		t.Fatal(err)
	}
	preview = authedMultipartFileRequest(t, a, "manager@example.com", "/demo/app/dokumente/rechnungen/import/preview", nil, "invoice_file", "invoice.xml", data)
	token = ebInterfaceImportPreviewToken(t, preview)
	if location := apply(); !strings.Contains(location, "result=already") {
		t.Fatalf("repeat=%s", location)
	}
}

func sqlImportDocuments(a *app) store.DocumentRepository {
	repo, _ := store.BindDocumentRepository(a.documentStore, a.tenantIdentities["demo"].Ref())
	return repo
}
