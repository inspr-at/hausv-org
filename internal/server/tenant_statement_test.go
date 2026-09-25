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
	if run.CalculationVersion != 5 || run.Approval == nil {
		t.Fatal("tenant statements must derive from the approved v5 WEG run")
	}
	for _, party := range run.Input.Parties {
		if !party.Owner {
			t.Fatal("WEG source includes a tenant recipient", party)
		}
	}
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
	if s.Source.ID != run.ID || s.Source.CalculationVersion != 5 || s.Source.Approval == nil || len(s.Accounts) == 0 || len(s.Accounts[0].Parties) == 0 {
		t.Fatal("tenant snapshot lost its approved owner-only source or lease parties")
	}
	route := "/app/settings/tenant-statements/" + s.ID
	d, err := statementpdf.TenantDocument(s, s.Accounts[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Join(append(append(d.Basis, d.PaymentTerms...), d.Inspection...), "\n")
	for _, cost := range d.Costs {
		if cost.Name == "Heizung" && !strings.Contains(strings.Join(cost.Measurements, "\n"), "Energiebezüge und Preise") {
			t.Fatal("tenant PDF lost the source's HeizKG information", cost.Measurements)
		}
	}
	for _, field := range d.Info {
		text += "\n" + field.Label + ": " + field.Value
	}
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

func TestTenantDemoAkontoAmounts(t *testing.T) {
	a, repos, _ := newArchiveDemoApp(t)
	a.leaseStore = store.NewSQLLeaseStore(a.tenantDB)
	run := createArchiveDemoRun(t, a, repos)
	if !run.Input.Structure.Legal.ShowVAT {
		t.Fatal("demo VAT basis missing")
	}
	leasesRepo, _ := store.BindLeaseRepository(a.leaseStore, a.tenantIdentities[archiveDemoTenant].Ref())
	leases, err := leasesRepo.List()
	if err != nil {
		t.Fatal(err)
	}
	for _, lease := range leases {
		m := store.RentalManagement{UnitID: lease.UnitID, Active: true, HeatingMonthlyConfirmed: true, ContractCosts: map[string][]string{lease.ID: {"abfall", "reinigung", "versicherung", "lift"}}}
		for _, p := range run.Input.Parties {
			if p.UnitID == lease.UnitID && p.Owner {
				m.OwnerEmail = p.ID
				break
			}
		}
		s, err := store.DeriveTenantStatement(run, m, []store.Lease{lease}, "2026-06-20", "2026-08-05")
		if lease.UnitID == "top-3" {
			if err == nil || !strings.Contains(err.Error(), "Eigentümerwechsel") {
				t.Fatal("split owner must remain blocked", err)
			}
			continue
		}
		if err != nil {
			t.Fatal(lease.UnitID, err)
		}
		for _, account := range s.Accounts {
			if account.Landlord {
				continue
			}
			for _, component := range [][2]int64{{account.OperatingCents, account.OperatingPrepaidCents}, {account.HeatingCents, account.HeatingPrepaidCents}} {
				if component[1]*100 < component[0]*85 || component[1]*100 > component[0]*115 {
					t.Errorf("%s akonto %d is not within 15%% of %d", lease.UnitID, component[1], component[0])
				}
			}
			if account.BalanceCents() == 0 {
				t.Error("demo balance must be realistic and nonzero", lease.UnitID)
			}
			t.Logf("%s passed=%d retained=%d bk=%d heat=%d prepaid=%d saldo=%d", lease.UnitID, s.SourcePassedCents, s.SourceRetainedCents, account.OperatingCents, account.HeatingCents, account.OperatingPrepaidCents+account.HeatingPrepaidCents, account.BalanceCents())
		}
	}
	// VAT is a breakdown of the same owner expense, never an added charge.
	legal := run.Input.Structure.Legal
	legal.ShowVAT = false
	if err := repos.annualStatementPeriods.SaveLegal(2025, legal); err != nil {
		t.Fatal(err)
	}
	_, withoutVAT, err := repos.annualStatementRuns.Preview(2025, nil)
	if err != nil {
		t.Fatal(err)
	}
	for i, unit := range run.Result.Units {
		before := withoutVAT.Units[i]
		if unit.UnitID != before.UnitID || unit.AllocatedCents != before.AllocatedCents || unit.PrepaidCents != before.PrepaidCents {
			t.Fatalf("VAT changed owner result: %+v / %+v", unit, before)
		}
		t.Logf("owner %s costs=%d prepaid=%d", unit.UnitID, unit.AllocatedCents, unit.PrepaidCents)
		if unit.UnitID == "top-1" && (unit.AllocatedCents != 87169 || unit.PrepaidCents != 60000) {
			t.Fatal("Top 1 changed", unit)
		}
	}
}
