package store

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestAnnualStatementCombinedFeatures(t *testing.T) {
	for _, regime := range []string{"weg", "mrg_voll", "mrg_teil"} {
		t.Run(regime, func(t *testing.T) {
			in := heatingFixture()
			in.Structure.Legal.Regime = regime
			in.Structure.Legal.ShowVAT = true
			in.Structure.CostTypes[0].VATRatePercent = 20
			in.Structure.CostTypes = append(in.Structure.CostTypes, AnnualStatementCostType{Key: "wasser", Name: "Wasser", Allocatable: true, AllocationKey: AllocationKeyNutzwert, VATRatePercent: 10})
			in.Receipts = append(in.Receipts, AnnualStatementReceipt{ID: "r3", DocumentID: "d3", PeriodYear: 2025, CostTypeKey: "wasser", AmountCents: 11001, InvoiceDate: "2025-02-01"})
			in.Structure.UnitBases[0].VacantFrom, in.Structure.UnitBases[0].VacantTo = "2025-03-01", "2025-05-31"
			in.Reserve = []AnnualStatementReserveEntry{{Kind: ReserveKindOpening, AmountCents: 10000}, {Kind: ReserveKindContribution, AmountCents: 5000}, {Kind: ReserveKindWithdrawal, AmountCents: 2000}, {Kind: ReserveKindInterest, AmountCents: 123}}
			result, issues := CalculateAnnualStatementRun(in)
			if len(issues) != 0 {
				t.Fatal(issues)
			}
			if result.TotalCents != 131001 {
				t.Fatalf("total: %d", result.TotalCents)
			}
			if regime == "weg" {
				if len(result.Vacancy) != 0 || result.Reserve == nil || result.Reserve.ClosingCents != 13123 {
					t.Fatalf("WEG result: %+v", result)
				}
				var shares int64
				for _, share := range result.Reserve.Shares {
					shares += share.AmountCents
				}
				if shares != 13123 {
					t.Fatal(shares)
				}
				if result.Units[0].AllocatedCents != 45250 || result.Units[1].AllocatedCents != 85751 {
					t.Fatal(result.Units)
				}
			} else {
				// A's 42500 heat + 2750 water split separately over 92/365 days.
				if result.Reserve != nil || len(result.Vacancy) != 1 || result.Vacancy[0].AmountCents != 11405 || result.Units[0].AllocatedCents != 33845 || result.Units[1].AllocatedCents != 85751 {
					t.Fatalf("MRG result: %+v", result)
				}
			}
			check := func(costs []AnnualStatementRunCost, groups []AnnualStatementVATGroup, total int64) {
				t.Helper()
				var gross int64
				for _, line := range costs {
					if line.NetCents+line.VATCents != line.AmountCents {
						t.Fatalf("line does not reconcile: %+v", line)
					}
					gross += line.AmountCents
				}
				var groupGross int64
				for _, group := range groups {
					if group.NetCents+group.VATCents != group.GrossCents {
						t.Fatal(group)
					}
					groupGross += group.GrossCents
				}
				if gross != total || groupGross != total {
					t.Fatalf("line/group/total: %d/%d/%d", gross, groupGross, total)
				}
			}
			var allocated int64
			for _, unit := range result.Units {
				check(unit.Costs, unit.VAT, unit.AllocatedCents)
				allocated += unit.AllocatedCents
				if unit.BalanceCents != unit.AllocatedCents-unit.PrepaidCents {
					t.Fatal(unit)
				}
			}
			for _, line := range result.Vacancy {
				check(line.Costs, line.VAT, line.AmountCents)
				allocated += line.AmountCents
			}
			if allocated != result.TotalCents {
				t.Fatal(allocated)
			}
			wantGroups := []AnnualStatementVATGroup{{RatePercent: 10, NetCents: 10001, VATCents: 1000, GrossCents: 11001}, {RatePercent: 20, NetCents: 100000, VATCents: 20000, GrossCents: 120000}}
			if !reflect.DeepEqual(result.VATGroups, wantGroups) {
				t.Fatalf("house VAT: %+v", result.VATGroups)
			}
			run, err := newAnnualStatementRun(in, result, 1, "manager@example.com", time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))
			if err != nil || run.CalculationVersion != 3 {
				t.Fatalf("new VAT run: %d %v", run.CalculationVersion, err)
			}
			raw, err := json.Marshal(run)
			if err != nil {
				t.Fatal(err)
			}
			var stored AnnualStatementRun
			if err := json.Unmarshal(raw, &stored); err != nil {
				t.Fatal(err)
			}
			replay, issues := ReplayAnnualStatementRun(stored)
			if len(issues) != 0 || !reflect.DeepEqual(replay, result) {
				t.Fatalf("v3 replay: %+v %v", replay, issues)
			}
			// Historical snapshots omit later optional inputs. Their heating algorithm
			// remains version-specific even if the VAT display flag is present.
			in.Reserve = nil
			in.Structure.UnitBases[0].VacantFrom, in.Structure.UnitBases[0].VacantTo = "", ""
			for version, wantA := range map[int]int64{1: 32750, 2: 45250} {
				old, issues := ReplayAnnualStatementRun(AnnualStatementRun{CalculationVersion: version, Input: in})
				if len(issues) != 0 || old.Units[0].AllocatedCents != wantA || old.TotalCents != 131001 || old.Reserve != nil || old.Vacancy != nil || old.VATGroups != nil {
					t.Fatalf("v%d replay: %+v %v", version, old, issues)
				}
			}
		})
	}
}
