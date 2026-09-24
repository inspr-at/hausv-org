package server

import (
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"

	appmail "github.com/inspr-at/hausv-org/internal/mail"
	"github.com/inspr-at/hausv-org/internal/statementpdf"
	"github.com/inspr-at/hausv-org/internal/store"
)

func TestTenantStatementRoutesArchiveDeliveryAndDenials(t *testing.T) {
	a, repos, _ := newArchiveDemoApp(t)
	a.leaseStore = store.NewSQLLeaseStore(a.tenantDB)
	a.profiles[archiveDemoManager] = userProfile{Email: archiveDemoManager, Role: roleAdmin, Tenants: []string{archiveDemoTenant}, AuthMethods: defaultAuthMethods()}
	structure, _ := repos.annualStatementPeriods.Structure(2025)
	structure.Legal.ShowVAT = true
	if err := repos.annualStatementPeriods.SaveLegal(2025, structure.Legal); err != nil {
		t.Fatal(err)
	}
	run := createArchiveDemoRun(t, a, repos)
	repo, _ := store.BindTenantStatementRepository(a.tenantDB, a.tenantIdentities[archiveDemoTenant].Ref())
	path := "/app/settings/annual-statement/runs/" + run.ID + "/tenant-statements"
	create := archiveDemoRequest(t, a, archiveDemoManager, "POST", path, url.Values{"unit_id": {"top-2"}, "statement_on": {"2026-06-20"}})
	if create.Code != 303 {
		t.Fatal(create.Code, create.Body.String())
	}
	list, err := repo.List(run.ID)
	if err != nil || len(list) != 1 {
		t.Fatalf("creation %s: %v %+v", create.Header().Get("Location"), err, list)
	}
	s := list[0]
	route := "/app/settings/tenant-statements/" + s.ID
	d, err := statementpdf.TenantDocument(s, s.Accounts[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Join(append(append(d.Basis, d.PaymentTerms...), d.Inspection...), "\n")
	for _, want := range []string{"Im Namen und auf Rechnung von", "30. Juni 2026", "übernächsten Zinstermin", "05.08.2026", "HeizKG", structure.Legal.InspectionPlace} {
		if !strings.Contains(text, want) {
			t.Errorf("PDF missing %s: %s", want, text)
		}
	}
	for _, account := range s.Accounts {
		for _, line := range account.Lines {
			if line.Key == "versicherung" {
				t.Fatal("nonpassable insurance")
			}
		}
	}
	for _, action := range []string{"archive", "send"} {
		w := archiveDemoRequest(t, a, archiveDemoManager, "POST", route+"/"+action, nil)
		if w.Code != 409 {
			t.Fatal(action, w.Code)
		}
	}
	pdf := archiveDemoRequest(t, a, archiveDemoManager, "GET", route+"/pdf", nil)
	if pdf.Code != 200 || !strings.HasPrefix(pdf.Body.String(), "%PDF") {
		t.Fatal("PDF", pdf.Code)
	}
	for _, role := range []string{roleOwner, roleResident, roleBeirat} {
		email := strings.ToLower(role) + "@example.test"
		a.profiles[email] = userProfile{Email: email, Role: role, Tenants: []string{archiveDemoTenant}, AuthMethods: defaultAuthMethods()}
		for _, action := range []string{"approve", "archive", "send"} {
			w := archiveDemoRequest(t, a, email, "POST", route+"/"+action, nil)
			if w.Code != 403 {
				t.Fatalf("%s %s: %d", role, action, w.Code)
			}
		}
		if w := archiveDemoRequest(t, a, email, "GET", route+"/pdf", nil); w.Code != 403 {
			t.Fatal("PDF disclosure", role, w.Code)
		}
	}
	for _, action := range []string{"approve", "archive", "archive"} {
		w := archiveDemoRequest(t, a, archiveDemoManager, "POST", route+"/"+action, nil)
		if w.Code != 303 {
			t.Fatal(action, w.Code, w.Body.String())
		}
	}
	saved, _, _ := repo.Get(s.ID)
	documents := tenantStatementArchiveDocs(repos.documents, saved)
	if len(documents) != 2 {
		t.Fatalf("archive recipients %+v", documents)
	}
	outbox := t.TempDir()
	a.mailer = appmail.NewSMTP("", "", "", "", "verwaltung@example.test").WithOutbox(outbox)
	for i := 0; i < 2; i++ {
		w := archiveDemoRequest(t, a, archiveDemoManager, "POST", route+"/send", nil)
		if w.Code != 303 {
			t.Fatal("send", w.Code, w.Body.String())
		}
	}
	files, err := os.ReadDir(outbox)
	if err != nil || len(files) != 2 {
		t.Fatal("duplicate or missing mail", len(files), err)
	}
	deliveries, err := repos.annualStatementDeliveries.List(s.ID)
	if err != nil || len(deliveries) != 2 {
		t.Fatal("delivery records", deliveries, err)
	}
	ownerCopy := false
	for _, item := range deliveries {
		if item.PartyID == "owner" && item.Recipient == s.Owner.ID {
			ownerCopy = true
		}
		if item.Status != "sent" {
			t.Fatal(item)
		}
	}
	if !ownerCopy {
		t.Fatal("owner copy missing")
	}
	page := archiveDemoRequest(t, a, archiveDemoManager, http.MethodGet, "/app/settings/annual-statement?year=2025&run="+run.ID, nil)
	for _, want := range []string{"Mieterabrechnungen", "Mietverwaltung aktiv", "Matthias Dorn", "Eigentümerkopie ansehen", "Archiviert"} {
		if !strings.Contains(page.Body.String(), want) {
			t.Errorf("UI missing %s", want)
		}
	}
}
