package store

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func tenantStatementFixture() (AnnualStatementRun, RentalManagement, []Lease) {
	run := AnnualStatementRun{ID: "weg-run", PeriodYear: 2025, Revision: 1, Approval: &AnnualStatementRunApproval{ApprovedBy: "manager@example.test", Role: RoleManager, ApprovedAt: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)}, Input: AnnualStatementRunInput{Period: AnnualStatementPeriod{Year: 2025, StartsOn: "2025-01-01", EndsOn: "2025-12-31"}, Parties: []AnnualStatementRunParty{{UnitID: "top-1", ID: "owner@example.test", Name: "Eigentümer", Owner: true}}, Structure: AnnualStatementPeriodStructure{Legal: AnnualStatementLegalSettings{Regime: "weg", ShowVAT: true, HeizKGApplies: true, InspectionPlace: "Büro", InspectionPeriod: "Mo–Fr 9–12 Uhr", InspectionContact: "Verwaltung"}}}, Result: AnnualStatementRunResult{Units: []AnnualStatementRunUnit{{UnitID: "top-1", Label: "Top 1", AllocatedCents: 39101, Costs: []AnnualStatementRunCost{{CostTypeKey: "wasser", Name: "Wasser", AmountCents: 11000, NetCents: 10000, VATCents: 1000}, {CostTypeKey: "heizung", Name: "Heizung", AmountCents: 12001, NetCents: 10001, VATCents: 2000}, {CostTypeKey: "reparatur", Name: "Reparatur", AmountCents: 5100, NetCents: 4250, VATCents: 850}, {CostTypeKey: "ruecklage", Name: "Rücklage", AmountCents: 11000, NetCents: 11000}}}}}}
	m := RentalManagement{UnitID: "top-1", OwnerEmail: "owner@example.test", Active: true, HeatingMonthlyConfirmed: true, Passable: map[string]bool{"ruecklage": true}}
	l := Lease{ID: "lease-1", UnitID: "top-1", Status: LeaseStatusActive, LeaseKind: LeaseKindHauptmiete, UseKind: UseKindWohnung, MRGScope: MRGVoll, RentRegime: RentRegimeFrei, ConcludedOn: "2024-01-01", StartsOn: "2024-01-01", VATOpted: true, ZinsterminDay: 5, Parties: []LeaseParty{{ID: "party-1", Name: "Erste Mietpartei", Email: "first@example.test", Role: PartyHauptmieter, ValidFrom: "2024-01-01"}}, Components: []RentComponent{{Kind: ComponentBKAkonto, NetCents: 500, VATRateBP: 1000, ValidFrom: "2024-01-01"}, {Kind: ComponentHeizAkonto, NetCents: 600, VATRateBP: 2000, ValidFrom: "2024-01-01"}}}
	return run, m, []Lease{l}
}

func TestTenantStatementExclusionsVATAndConservation(t *testing.T) {
	for _, use := range []string{UseKindWohnung, UseKindGarage, UseKindGeschaeft} {
		t.Run(use, func(t *testing.T) {
			run, m, leases := tenantStatementFixture()
			leases[0].UseKind = use
			s, err := DeriveTenantStatement(run, m, leases, "2026-06-20", "")
			if err != nil {
				t.Fatal(err)
			}
			if s.SourcePassedCents != 23001 || s.SourceRetainedCents != 16100 || s.SourcePassedCents+s.SourceRetainedCents != run.Result.Units[0].AllocatedCents {
				t.Fatalf("reconciliation %+v", s)
			}
			a := s.Accounts[0]
			if len(a.Lines) != 2 {
				t.Fatalf("nonpassable leakage %+v", a.Lines)
			}
			rate := 10
			if use != UseKindWohnung {
				rate = 20
			}
			if a.Lines[0].VATRate != rate || a.Lines[1].VATRate != 20 {
				t.Fatal(a.Lines)
			}
			if a.OperatingPrepaidCents != 6600 || a.HeatingPrepaidCents != 8640 || a.OperatingDueOn != "2026-08-05" {
				t.Fatalf("prepayments/due %+v", a)
			}
		})
	}
	run, m, leases := tenantStatementFixture()
	leases[0].VATOpted = false
	s, err := DeriveTenantStatement(run, m, leases, "2026-06-01", "")
	if err != nil {
		t.Fatal(err)
	}
	if s.Accounts[0].Lines[0].VATCents != 0 || s.Accounts[0].Lines[0].GrossCents != 11000 {
		t.Fatal("VAT without opt-in")
	}
}

func TestTenantStatementChangeUsesDueDateForBKAndMonthsForHeat(t *testing.T) {
	run, m, leases := tenantStatementFixture()
	first := leases[0]
	first.EndsOn = "2025-07-01"
	first.Parties[0].ValidTo = "2025-07-01"
	next := leases[0]
	next.ID = "lease-2"
	next.StartsOn = "2025-07-01"
	next.Parties = []LeaseParty{{ID: "party-2", Name: "Zweite Mietpartei", Email: "next@example.test", Role: PartyHauptmieter, ValidFrom: "2025-07-01"}}
	s, err := DeriveTenantStatement(run, m, []Lease{first, next}, "2026-06-20", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Accounts) != 2 {
		t.Fatalf("accounts %+v", s.Accounts)
	}
	a, b := s.Accounts[0], s.Accounts[1]
	if a.OperatingCents != 0 || a.OperatingPrepaidCents != 0 || b.OperatingCents != 11000 || b.OperatingPrepaidCents != 6600 {
		t.Fatalf("due-date BK: %+v %+v", a, b)
	}
	if a.Lines[0].SourceGrossCents != 6001 || b.Lines[1].SourceGrossCents != 6000 || a.HeatingPrepaidCents != 4320 || b.HeatingPrepaidCents != 4320 {
		t.Fatalf("monthly heat %+v %+v", a, b)
	}
	if s.SourcePassedCents+s.SourceRetainedCents != 39101 {
		t.Fatal("lost cent")
	}
	// An unlet unit on the due date settles BK with the landlord.
	next.EndsOn = "2026-07-01"
	next.Parties[0].ValidTo = "2026-07-01"
	s, err = DeriveTenantStatement(run, m, []Lease{first, next}, "2026-06-20", "")
	if err != nil {
		t.Fatal(err)
	}
	owner := s.Accounts[len(s.Accounts)-1]
	if !owner.Landlord || owner.OperatingCents != 11000 || owner.OperatingPrepaidCents != 6600 {
		t.Fatalf("vacant due date %+v", owner)
	}
}

func TestTenantStatementPartyChangeWithinLease(t *testing.T) {
	run, m, leases := tenantStatementFixture()
	leases[0].Parties[0].ValidTo = "2025-07-01"
	leases[0].Parties = append(leases[0].Parties, LeaseParty{ID: "replacement", Name: "Neue Partei", Email: "new@example.test", Role: PartyHauptmieter, ValidFrom: "2025-07-01"})
	s, err := DeriveTenantStatement(run, m, leases, "2026-06-20", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Accounts) != 2 || s.Accounts[0].OperatingCents != 0 || s.Accounts[1].OperatingCents != 11000 {
		t.Fatal(s.Accounts)
	}
}

func TestTenantStatementReturningJointPartyDoesNotDuplicateAkonto(t *testing.T) {
	run, m, leases := tenantStatementFixture()
	leases[0].Parties = append(leases[0].Parties, LeaseParty{ID: "joint", Name: "Mitmieter", Role: PartyMitmieter, ValidFrom: "2025-07-01", ValidTo: "2025-10-01"})
	s, err := DeriveTenantStatement(run, m, leases, "2026-06-20", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Accounts) != 2 || len(s.Accounts[0].Intervals) != 2 {
		t.Fatal(s.Accounts)
	}
	if s.Accounts[0].HeatingPrepaidCents != 6480 || s.Accounts[1].HeatingPrepaidCents != 2160 || s.Accounts[0].OperatingPrepaidCents != 6600 {
		t.Fatalf("duplicated interval prepayment: %+v", s.Accounts)
	}
}

func TestTenantStatementContractAndFailClosed(t *testing.T) {
	run, m, leases := tenantStatementFixture()
	leases[0].MRGScope = MRGTeil
	if _, err := DeriveTenantStatement(run, m, leases, "2026-06-20", "2026-08-01"); err == nil {
		t.Fatal("missing catalogue accepted")
	}
	m.ContractCosts = map[string][]string{"lease-1": {"wasser", "ruecklage"}}
	s, err := DeriveTenantStatement(run, m, leases, "2026-06-20", "2026-08-01")
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Accounts[0].Lines) != 2 || s.Accounts[0].OperatingDueOn != "2026-08-01" {
		t.Fatal(s)
	}
	tests := []struct {
		name   string
		mutate func(*AnnualStatementRun, *RentalManagement, []Lease)
	}{{"unapproved", func(r *AnnualStatementRun, m *RentalManagement, l []Lease) { r.Approval = nil }}, {"not WEG", func(r *AnnualStatementRun, m *RentalManagement, l []Lease) {
		r.Input.Structure.Legal.Regime = "mrg_voll"
	}}, {"inactive", func(r *AnnualStatementRun, m *RentalManagement, l []Lease) { m.Active = false }}, {"unknown owner", func(r *AnnualStatementRun, m *RentalManagement, l []Lease) { m.OwnerEmail = "stranger@example.test" }}, {"VAT basis", func(r *AnnualStatementRun, m *RentalManagement, l []Lease) { r.Input.Structure.Legal.ShowVAT = false }}, {"missing Akonto", func(r *AnnualStatementRun, m *RentalManagement, l []Lease) { l[0].Components = nil }}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, m, l := tenantStatementFixture()
			tt.mutate(&r, &m, l)
			if _, err := DeriveTenantStatement(r, m, l, "2026-06-20", ""); err == nil {
				t.Fatal("unsafe derivation accepted")
			}
		})
	}
}

func TestTenantStatementRepositoryIsolationAndImmutableSnapshots(t *testing.T) {
	database, lanes := testLanes(t)
	tenant := testTenantRef("demo")
	foreign := testTenantRef("other")
	run, m, leases := tenantStatementFixture()
	units, _ := BindUnitRepository(NewSQLUnitStore(lanes), tenant)
	if err := units.SetUnits([]Unit{{ID: "top-1", Label: "Top 1", OwnerEmails: []string{m.OwnerEmail}}}); err != nil {
		t.Fatal(err)
	}
	repo, _ := BindTenantStatementRepository(lanes, tenant)
	if err := repo.SaveManagement(m); err != nil {
		t.Fatal(err)
	}
	lr, _ := BindLeaseRepository(NewSQLLeaseStore(lanes), tenant)
	if _, err := lr.Create(leases[0]); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(run)
	approval, _ := json.Marshal(run.Approval)
	if _, err := database.Exec(`INSERT INTO annual_statement_runs(tenant_id,tenant_slug,id,period_year,revision,data) VALUES($1,$2,$3,2025,1,$4)`, tenant.ID, tenant.Slug, run.ID, string(raw)); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO annual_statement_run_approvals(tenant_id,tenant_slug,run_id,data) VALUES($1,$2,$3,$4)`, tenant.ID, tenant.Slug, run.ID, string(approval)); err != nil {
		t.Fatal(err)
	}
	s, err := repo.Create(run, "top-1", "2026-06-20", "", "manager@example.test")
	if err != nil {
		t.Fatal(err)
	}
	other, _ := BindTenantStatementRepository(lanes, foreign)
	if _, found, err := other.Get(s.ID); err != nil || found {
		t.Fatal("cross-tenant read", err)
	}
	if _, err := other.Create(run, "top-1", "2026-06-20", "", "manager@example.test"); err == nil {
		t.Fatal("cross-tenant source accepted")
	}
	if _, err := repo.Approve(s.ID, "owner@example.test", RoleOwner); err == nil {
		t.Fatal("owner approval")
	}
	approved, err := repo.Approve(s.ID, "manager@example.test", RoleManager)
	if err != nil || approved.Approval == nil {
		t.Fatal(err)
	}
	m.Passable["wasser"] = false
	if err := repo.SaveManagement(m); err != nil {
		t.Fatal(err)
	}
	saved, _, err := repo.Get(s.ID)
	if err != nil || saved.SourcePassedCents != s.SourcePassedCents {
		t.Fatal("snapshot drift")
	}
	if _, err := database.Exec(`UPDATE tenant_statements SET data='{}' WHERE tenant_id=$1 AND id=$2`, tenant.ID, s.ID); err == nil {
		t.Fatal("mutable snapshot")
	}
	if _, err := database.Exec(`UPDATE tenant_statement_approvals SET data='{}' WHERE tenant_id=$1 AND statement_id=$2`, tenant.ID, s.ID); err == nil {
		t.Fatal("mutable approval")
	}
	if _, err := repo.Create(AnnualStatementRun{ID: "missing"}, "top-1", "2026-06-20", "", "manager@example.test"); err == nil || !strings.Contains(err.Error(), "WEG") {
		t.Fatal("missing source", err)
	}
}
