package store

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestAnnualStatementLowestFeatureVersionAndReplay(t *testing.T) {
	for _, dated := range []bool{false, true} {
		for _, agreed := range []bool{false, true} {
			for _, vat := range []bool{false, true} {
				t.Run(fmt.Sprintf("dated=%t/agreed=%t/vat=%t", dated, agreed, vat), func(t *testing.T) {
					in := datedInput(false)
					if !dated {
						in.Parties = nil
						in.StatementOn = ""
					}
					in.Structure.Legal.ShowVAT = vat
					in.Structure.CostTypes[0].VATRatePercent = 20
					wantVersion, wantA := 2, int64(2551)
					if vat {
						wantVersion = 3
					}
					if dated {
						wantVersion = 4
					}
					if agreed {
						wantVersion, wantA = 5, 51
						in.Structure.CostTypes[0].AllocationKey = AllocationKeyAgreed
						in.Structure.Legal.AgreedShares = map[string]map[string]int{"tax": {"a": 0, "b": 1_000_000}}
					}
					result, issues := CalculateAnnualStatementRun(in)
					if len(issues) > 0 || result.Units[0].AllocatedCents != wantA || result.TotalCents != 10102 {
						t.Fatal(result, issues)
					}
					run, err := newAnnualStatementRun(in, result, 1, "manager@example.com", time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC))
					if err != nil || run.CalculationVersion != wantVersion {
						t.Fatal(run.CalculationVersion, err)
					}
					raw, err := json.Marshal(run)
					if err != nil {
						t.Fatal(err)
					}
					var stored AnnualStatementRun
					if err := json.Unmarshal(raw, &stored); err != nil {
						t.Fatal(err)
					}
					replayed, issues := ReplayAnnualStatementRun(stored)
					if len(issues) > 0 || !reflect.DeepEqual(replayed, stored.Result) {
						t.Fatalf("v%d replay changed stored result: %+v %v", wantVersion, replayed, issues)
					}
					if dated {
						old := partyResult(t, replayed, "a-old@example.com")
						next := partyResult(t, replayed, "b-new@example.com")
						if old.Unit.AllocatedCents != 0 || old.SettlementRecipient || next.Unit.AllocatedCents != wantA || next.Unit.PrepaidCents != 3000 || next.DueOn != "2026-07-05" || !next.SettlementRecipient {
							t.Fatal(old, next)
						}
					}
				})
			}
		}
	}
}

func TestAnnualStatementV4AndV5HeatingPartyReplay(t *testing.T) {
	for _, version := range []int{4, 5} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			in := datedInput(true)
			in.Structure.Legal.ShowVAT = true
			in.Structure.CostTypes[0].VATRatePercent = 20
			if version == 5 {
				in.Structure.CostTypes = append(in.Structure.CostTypes, AnnualStatementCostType{Key: "lift", Name: "Lift", Allocatable: true, AllocationKey: AllocationKeyAgreed})
				in.Structure.Legal.AgreedShares = map[string]map[string]int{"lift": {"a": 0, "b": 1_000_000}}
				in.Receipts = append(in.Receipts, AnnualStatementReceipt{ID: "r3", DocumentID: "d3", PeriodYear: 2025, CostTypeKey: "lift", AmountCents: 10001, InvoiceDate: "2025-02-01"})
			}
			run := AnnualStatementRun{CalculationVersion: version, Input: in}
			result, issues := ReplayAnnualStatementRun(run)
			if len(issues) > 0 {
				t.Fatal(issues)
			}
			old, next := partyResult(t, result, "a-old@example.com"), partyResult(t, result, "b-new@example.com")
			if old.Unit.AllocatedCents != 21250 || next.Unit.AllocatedCents != 21250 || old.Unit.PrepaidCents != 500 || next.Unit.PrepaidCents != 2500 || next.DueOn != "2026-07-05" {
				t.Fatal(old, next)
			}
			if result.Units[0].AllocatedCents != 42500 || result.Units[0].Costs[0].VATCents != 7083 {
				t.Fatal(result.Units[0])
			}
			if version == 5 && result.Units[1].AllocatedCents != 87501 {
				t.Fatal(result.Units[1])
			}
		})
	}
}
