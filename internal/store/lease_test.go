package store

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestLeaseStoreCreateHistoryOverlapAndImport(t *testing.T) {
	database, lanes := testLanes(t)
	demo := testTenantRef("demo")
	other := testTenantRef("other")
	for _, tenant := range []TenantRef{demo, other} {
		for _, id := range []string{"top-1", "top-2"} {
			if _, err := database.Exec(`INSERT INTO units(tenant_id,tenant_slug,id,data) VALUES($1,$2,$3,'{}')`, tenant.ID, tenant.Slug, id); err != nil {
				t.Fatal(err)
			}
		}
	}
	repo, ok := BindLeaseRepository(NewSQLLeaseStore(lanes), demo)
	if !ok {
		t.Fatal("bind")
	}
	foreign, ok := BindLeaseRepository(NewSQLLeaseStore(lanes), other)
	if !ok {
		t.Fatal("bind other")
	}
	when := time.Date(2026, 9, 24, 8, 0, 0, 0, time.UTC)
	created, err := repo.Create(Lease{
		ID: "lease-1", UnitID: "Top 1", ConcludedOn: "2018-03-15", StartsOn: "2018-04-01",
		UseKind: UseKindWohnung, MRGScope: MRGTeil, RentRegime: RentRegimeFrei,
		LandlordIsBusiness: true, TenantIsConsumer: true, UpdatedAt: when, UpdatedBy: "Vera@Example.com",
		Parties:    []LeaseParty{{Name: "Eva Huber", Email: "Eva@Example.com", Role: PartyHauptmieter, ValidFrom: "2018-04-01"}},
		Components: []RentComponent{{Kind: ComponentHMZ, NetCents: 100000, VATRateBP: 1000, ValidFrom: "2018-04-01", Origin: OriginManual}},
		Clauses: []IndexClause{{
			ClauseType: ClauseVPIThreshold, Series: "vpi2020", BasePeriod: "2024-09", BaseValue: "123,6",
			ThresholdKind: "percent", ThresholdValue: "5", FullChangeOnTrigger: true, TwoWay: true,
			ClauseText: "Der Hauptmietzins ist wertgesichert.", ReviewStatus: ReviewOK, ValidFrom: "2018-04-01",
			State: &ValorisationState{ContractValue: "1000.00", ContractBasePeriod: "2024-09", ContractBaseValue: "123.6", CapValue: "1000.00", CapAnchorPeriod: "2024-09"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.UpdatedBy != "vera@example.com" || created.PriceRestricted || created.Clauses[0].BaseValue != "123.6" {
		t.Fatalf("normalized = %+v restricted=%v base=%s", created, created.PriceRestricted, created.Clauses[0].BaseValue)
	}
	class := ClassifyLease(created)
	if !class.MieWeG || class.SpecialCap || !strings.Contains(class.MieWeGReason, "Teilanwendung") {
		t.Fatalf("classify partial = %+v", class)
	}
	if _, err := repo.Create(Lease{
		UnitID: "top-1", ConcludedOn: "2024-01-01", StartsOn: "2024-02-01",
		LeaseKind: LeaseKindHauptmiete, UseKind: UseKindWohnung, MRGScope: MRGVoll, RentRegime: RentRegimeRichtwert,
	}); !errors.Is(err, ErrLeaseOverlap) {
		t.Fatalf("overlap = %v", err)
	}
	if _, err := repo.Create(Lease{
		UnitID: "top-1", ConcludedOn: "2024-01-01", StartsOn: "2024-02-01",
		LeaseKind: LeaseKindUntermiete, UseKind: UseKindWohnung, MRGScope: MRGTeil, RentRegime: RentRegimeFrei,
	}); err != nil {
		t.Fatalf("sublease may overlap: %v", err)
	}
	ended, err := repo.End("lease-1", "2024-09-01", "vera@example.com")
	if err != nil || ended.Status != LeaseStatusEnded || ended.EndsOn != "2024-09-01" {
		t.Fatalf("end = %+v err=%v", ended, err)
	}
	next, err := repo.Create(Lease{
		ID: "lease-2", UnitID: "top-1", ConcludedOn: "2024-08-01", StartsOn: "2024-09-01",
		UseKind: UseKindWohnung, MRGScope: MRGVoll, RentRegime: RentRegimeRichtwert,
	})
	if err != nil || !next.PriceRestricted {
		t.Fatalf("successor = %+v err=%v", next, err)
	}
	full := ClassifyLease(next)
	if !full.MieWeG || !full.SpecialCap {
		t.Fatalf("classify full = %+v", full)
	}
	added, err := repo.AddRentComponent("lease-2", RentComponent{Kind: ComponentHMZ, NetCents: 104028, VATRateBP: 1000, ValidFrom: "2026-04-01", Origin: OriginManual})
	if err != nil {
		t.Fatal(err)
	}
	again, err := repo.AddRentComponent("lease-2", RentComponent{Kind: ComponentBKAkonto, NetCents: 18000, VATRateBP: 1000, ValidFrom: "2024-09-01"})
	if err != nil {
		t.Fatal(err)
	}
	if added.ID == again.ID {
		t.Fatal("components must be new rows")
	}
	clause, err := repo.SetClause(IndexClause{LeaseID: "lease-2", ClauseType: ClauseNone, ValidFrom: "2024-09-01", ClauseText: "keine"})
	if err != nil {
		t.Fatal(err)
	}
	reviewed, err := repo.ReviewClause(clause.ID, ReviewDoubtful, "unklar")
	if err != nil || reviewed.ReviewStatus != ReviewDoubtful || reviewed.ReviewNote != "unklar" {
		t.Fatalf("review = %+v err=%v", reviewed, err)
	}
	history, err := repo.History("Top 1")
	if err != nil || len(history) != 3 {
		t.Fatalf("history = %+v err=%v", history, err)
	}
	listed, err := repo.List()
	if err != nil || len(listed) != 3 {
		t.Fatalf("list = %d err=%v", len(listed), err)
	}
	if foreignList, err := foreign.List(); err != nil || len(foreignList) != 0 {
		t.Fatalf("other tenant saw %+v err=%v", foreignList, err)
	}
	got, found, err := repo.Get("lease-2")
	if err != nil || !found || len(got.Components) != 2 || got.Clauses[0].ReviewStatus != ReviewDoubtful {
		t.Fatalf("get = %+v found=%v err=%v", got, found, err)
	}
	commercial := ClassifyLease(Lease{UseKind: UseKindGeschaeft, MRGScope: MRGVoll, LeaseKind: LeaseKindHauptmiete})
	if commercial.MieWeG || commercial.SpecialCap {
		t.Fatalf("commercial = %+v", commercial)
	}

	csv := []byte("einheit;mieter_name;mieter_email;vertragsabschluss;beginn;ende;nutzung;mrg;regime;hmz_netto;bk_akonto;ust_satz;klausel_typ;index;basis_monat;basis_wert;schwelle;schwelle_art;schwelle_inkl;letzte_valorisierung_monat;letzte_valorisierung_wert;hmz_nach_letzter_valorisierung;klausel_text\n" +
		"Top 2;Lukas Steiner;lukas@example.com;01.03.2019;01.04.2019;;Wohnung;teil;frei;1.000,00;120,00;10;vpi_threshold;vpi2020;2024-09;123,6;5;prozent;nein;2024-09;123,6;1000,00;Schwelle fünf Prozent\n" +
		"Top 9;Niemand;x@example.com;01.01.2020;01.02.2020;;Wohnung;voll;frei;800;0;10;none;;;;;;;;;;;\n")
	drafts, err := ParseLeaseCSV(csv)
	if err != nil {
		t.Fatal(err)
	}
	units := []Unit{{ID: "top-1", Label: "Top 1"}, {ID: "top-2", Label: "Top 2"}}
	preview, err := repo.Import(drafts, units, false, "vera@example.com")
	if err != nil || preview.Committed != 0 || !preview.HasErrors() {
		t.Fatalf("dry run = %+v err=%v", preview, err)
	}
	if len(preview.Rows) != 2 || preview.Rows[0].Warnings[0] != ImportWarningIndexUnchecked || preview.Rows[1].Errors[0] != "unknown_unit" {
		t.Fatalf("rows = %+v", preview.Rows)
	}
	if listed, _ = repo.List(); len(listed) != 3 {
		t.Fatal("dry run wrote rows")
	}
	committed, err := repo.Import(drafts[:1], units, true, "vera@example.com")
	if err != nil || committed.Committed != 1 {
		t.Fatalf("commit = %+v err=%v", committed, err)
	}
	top2, err := repo.History("top-2")
	var hmz int64
	if err == nil && len(top2) == 1 {
		for _, component := range top2[0].Components {
			if component.Kind == ComponentHMZ {
				hmz = component.NetCents
			}
		}
	}
	if err != nil || len(top2) != 1 || hmz != 100000 || top2[0].Clauses[0].State == nil || top2[0].Clauses[0].State.ContractBaseValue != "123.6" {
		t.Fatalf("imported hmz=%d err=%v lease=%+v", hmz, err, top2)
	}
}

func TestIndexDecimalKeepsPublishedText(t *testing.T) {
	got, err := NormalizeIndexDecimal("123,6")
	if err != nil || got != "123.6" {
		t.Fatalf("got %q err=%v", got, err)
	}
	if _, err := NormalizeIndexDecimal("12a"); err == nil {
		t.Fatal("letters must fail")
	}
}
