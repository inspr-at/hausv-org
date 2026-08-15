package server

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/integrations"
	"github.com/inspr-at/hausv-org/internal/store"
)

func TestStructuredExportPackageManifestAndChecksum(t *testing.T) {
	generated := time.Date(2026, 7, 27, 18, 30, 0, 0, time.UTC)
	records := []integrations.ExportRecord{{
		TenantSlug: "demo",
		RecordID:   "unit-payment-status-top-1",
		Kind:       structuredExportSourcePayments,
		Occurred:   generated,
		UnitID:     "top-1",
		Fields: map[string]string{
			"status":            "paid",
			"source":            "building",
			"description":       "Zahlungsstatus Top 1",
			"verification_note": "Neutrale Rohdatenübergabe",
		},
	}}
	pkg, manifest, filename, err := buildStructuredExportPackage(context.Background(), "demo", []string{structuredExportSourcePayments}, records, generated)
	if err != nil {
		t.Fatalf("buildStructuredExportPackage: %v", err)
	}
	if manifest.Format != string(integrations.FormatManualCSV) || manifest.FormatVersion != integrations.StructuredHandoffVersion ||
		manifest.RecordCount != 1 || manifest.RejectedRecordCount != 0 || manifest.TargetSystemCompatibility {
		t.Fatalf("manifest = %+v", manifest)
	}
	if !strings.Contains(filename, "hausv-rohdaten-demo-") {
		t.Fatalf("filename = %q", filename)
	}
	archive, err := zip.NewReader(bytes.NewReader(pkg), int64(len(pkg)))
	if err != nil {
		t.Fatalf("zip.NewReader: %v", err)
	}
	files := map[string][]byte{}
	for _, item := range archive.File {
		reader, err := item.Open()
		if err != nil {
			t.Fatalf("open %s: %v", item.Name, err)
		}
		data, err := io.ReadAll(reader)
		_ = reader.Close()
		if err != nil {
			t.Fatalf("read %s: %v", item.Name, err)
		}
		files[item.Name] = data
	}
	csvData := files["hausv-raw-v0.csv"]
	if len(csvData) == 0 || !strings.Contains(string(csvData), "unit-payment-status-top-1") {
		t.Fatalf("CSV missing record:\n%s", csvData)
	}
	digest := sha256.Sum256(csvData)
	if got := hex.EncodeToString(digest[:]); got != manifest.CSVSHA256 {
		t.Fatalf("CSV checksum = %q, manifest = %q", got, manifest.CSVSHA256)
	}
	var decoded structuredExportManifest
	if err := json.Unmarshal(files["manifest.json"], &decoded); err != nil {
		t.Fatalf("manifest JSON: %v", err)
	}
	if decoded.CSVSHA256 != manifest.CSVSHA256 || decoded.Purpose == "" {
		t.Fatalf("decoded manifest = %+v", decoded)
	}
	for _, forbidden := range []string{"debit_account", "tax_code", "BMD-kompatibel", "RZL-kompatibel"} {
		if strings.Contains(string(csvData), forbidden) || strings.Contains(string(files["manifest.json"]), forbidden) {
			t.Fatalf("package makes forbidden claim or contains accounting field %q", forbidden)
		}
	}
}

func TestStructuredExportPortalSelectionDownloadAuditAndOneTimeToken(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	if err := testUnitRepository(t, a, "demo").SetUnits([]unit{{
		ID: "top-1", TenantSlug: "demo", Label: "Top 1", UnitType: unitTypeResidential,
		OwnerEmails: []string{"owner-secret@example.com"},
	}}); err != nil {
		t.Fatalf("SetTenantUnits: %v", err)
	}
	if _, err := testUnitPaymentRepository(t, a, "demo").Set(unitPaymentStatus{
		TenantSlug: "demo",
		UnitID:     "top-1",
		Status:     unitPaymentStatusPaid,
		UpdatedAt:  time.Date(2026, 7, 27, 18, 0, 0, 0, time.UTC),
		UpdatedBy:  "manager@example.com",
	}); err != nil {
		t.Fatalf("set payment status: %v", err)
	}

	page := authedRequest(t, a, "manager@example.com", "/demo/app/settings/data-export")
	if page.Code != http.StatusOK {
		t.Fatalf("page status = %d", page.Code)
	}
	for _, want := range []string{"Daten sicher weitergeben", "Zahlungsstatus der Einheiten", "Nichts ist vorausgewählt", "Kein BMD-/RZL-Format", "Manifest mit Prüfsumme"} {
		if !strings.Contains(page.Body.String(), want) {
			t.Fatalf("page missing %q:\n%s", want, page.Body.String())
		}
	}
	if strings.Contains(page.Body.String(), `value="`+structuredExportSourceParking+`"`) {
		t.Fatal("manager page must not offer admin-only parking exports")
	}

	previewResponse := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/data-export/preview", url.Values{
		"source": {structuredExportSourcePayments},
	})
	if previewResponse.Code != http.StatusSeeOther {
		t.Fatalf("preview status = %d", previewResponse.Code)
	}
	location, err := url.Parse(previewResponse.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse preview location: %v", err)
	}
	token := location.Query().Get("preview")
	if token == "" {
		t.Fatalf("preview token missing from %q", location.String())
	}
	if _, found := a.getStructuredExportPreview(token, "anderes-haus", "manager@example.com"); found {
		t.Fatal("preview token must be tenant-bound")
	}
	previewPage := authedRequest(t, a, "manager@example.com", location.String())
	for _, want := range []string{"Auswahl geprüft", "1</strong><span>Datensätze im CSV", "CSV SHA-256", "einmalig"} {
		if !strings.Contains(previewPage.Body.String(), want) {
			t.Fatalf("preview page missing %q:\n%s", want, previewPage.Body.String())
		}
	}

	download := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/data-export/download", url.Values{
		"preview_token": {token},
	})
	if download.Code != http.StatusOK || download.Header().Get("Content-Type") != "application/zip" {
		t.Fatalf("download status=%d content-type=%q body=%q", download.Code, download.Header().Get("Content-Type"), download.Body.String())
	}
	if strings.Contains(download.Body.String(), "owner-secret@example.com") {
		t.Fatal("export package must not contain unit membership addresses")
	}
	events := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: store.AuditActionIntegrationExport, Limit: 10})
	if len(events) != 1 || events[0].ActorEmail != "manager@example.com" || events[0].Details["records"] != "1" ||
		events[0].Details["sources"] != structuredExportSourcePayments || events[0].Details["csv_checksum"] == "" {
		t.Fatalf("export audit events = %+v", events)
	}
	second := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/data-export/download", url.Values{
		"preview_token": {token},
	})
	if second.Code != http.StatusGone {
		t.Fatalf("second download status = %d, want 410", second.Code)
	}
	if events := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: store.AuditActionIntegrationExport, Limit: 10}); len(events) != 1 {
		t.Fatalf("second download created audit event: %+v", events)
	}
}

func TestStructuredExportAuthorizationAndActorBinding(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["other@example.com"] = userProfile{Email: "other@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	if _, err := testUnitPaymentRepository(t, a, "demo").Set(unitPaymentStatus{
		TenantSlug: "demo", UnitID: "top-1", Status: unitPaymentStatusOpen, UpdatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("set payment status: %v", err)
	}
	if page := authedRequest(t, a, "resident@example.com", "/demo/app/settings/data-export"); page.Code != http.StatusForbidden {
		t.Fatalf("resident page status = %d, want 403", page.Code)
	}
	tampered := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/data-export/preview", url.Values{
		"source": {structuredExportSourceParking},
	})
	if tampered.Code != http.StatusSeeOther || !strings.Contains(tampered.Header().Get("Location"), "result=selection") {
		t.Fatalf("tampered manager preview status=%d location=%q", tampered.Code, tampered.Header().Get("Location"))
	}
	previewResponse := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/data-export/preview", url.Values{
		"source": {structuredExportSourcePayments},
	})
	location, _ := url.Parse(previewResponse.Header().Get("Location"))
	token := location.Query().Get("preview")
	crossActor := authedFormRequest(t, a, "other@example.com", "/demo/app/settings/data-export/download", url.Values{
		"preview_token": {token},
	})
	if crossActor.Code != http.StatusGone {
		t.Fatalf("cross-actor download status = %d, want 410", crossActor.Code)
	}
	if events := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: store.AuditActionIntegrationExport, Limit: 10}); len(events) != 0 {
		t.Fatalf("rejected downloads created audit events: %+v", events)
	}
}

func TestStructuredExportPreviewClaimIsAtomic(t *testing.T) {
	a := &app{}
	token := "preview-token"
	a.storeStructuredExportPreview(token, structuredExportPreview{
		TenantSlug: "demo",
		ActorEmail: "manager@example.com",
		CreatedAt:  time.Now().UTC(),
		Package:    []byte("package"),
	})
	const workers = 8
	var wg sync.WaitGroup
	results := make(chan bool, workers)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, ok := a.takeStructuredExportPreview(token, "demo", "manager@example.com")
			results <- ok
		}()
	}
	wg.Wait()
	close(results)
	claimed := 0
	for ok := range results {
		if ok {
			claimed++
		}
	}
	if claimed != 1 {
		t.Fatalf("preview claims = %d, want exactly one", claimed)
	}
}
