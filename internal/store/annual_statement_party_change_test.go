package store

import (
	"math"
	"math/big"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

func datedParties() []AnnualStatementRunParty {
	return []AnnualStatementRunParty{{UnitID: "a", ID: "a-old@example.com", Renter: true, ValidTo: "2025-06-30"}, {UnitID: "a", ID: "b-new@example.com", Renter: true, ValidFrom: "2025-07-01"}, {UnitID: "a", ID: "owner@example.com", Owner: true}}
}
func datedInput(heating bool) AnnualStatementRunInput {
	in := annualRunFixture()
	if heating {
		in = heatingFixture()
	}
	in.Parties = datedParties()
	in.StatementOn = "2026-06-01"
	in.Structure.Legal.Regime = "mrg_voll"
	return in
}
func partyResult(t *testing.T, r AnnualStatementRunResult, email string) AnnualStatementPartyShare {
	t.Helper()
	for _, s := range r.PartyShares {
		if s.PartyID == email {
			return s
		}
	}
	t.Fatal("party missing", email)
	return AnnualStatementPartyShare{}
}
func TestPartyDueDateRegimesAndVacancy(t *testing.T) {
	for _, regime := range []string{"mrg_voll", "mrg_teil", "weg", "ausnahme"} {
		t.Run(regime, func(t *testing.T) {
			in := datedInput(false)
			in.Structure.Legal.Regime = regime
			in.Structure.Legal.PartyDueOn = "2026-08-17"
			if regime == "weg" {
				in.Parties = in.Parties[:2]
				for i := range in.Parties {
					in.Parties[i].Owner = true
					in.Parties[i].Renter = false
				}
			}
			result, issues := CalculateAnnualStatementRun(in)
			if len(issues) > 0 {
				t.Fatal(issues)
			}
			old, new := partyResult(t, result, "a-old@example.com"), partyResult(t, result, "b-new@example.com")
			if old.Unit.BalanceCents != 0 || old.Unit.PrepaidCents != 0 || new.Unit.BalanceCents != -449 || new.Unit.PrepaidCents != 3000 {
				t.Fatal(result.PartyShares)
			}
			wantDue := "2026-08-17"
			if regime == "weg" {
				wantDue = "2026-08-01"
			}
			if regime == "mrg_voll" {
				wantDue = "2026-07-05"
			}
			if new.DueOn != wantDue {
				t.Fatal(new.DueOn, wantDue)
			}
			if regime != "weg" {
				in.Parties[1].ValidTo = "2026-06-30"
				result, issues = CalculateAnnualStatementRun(in)
				if len(issues) > 0 {
					t.Fatal(issues)
				}
				owner := partyResult(t, result, "owner@example.com")
				if owner.Unit.BalanceCents != -449 || !strings.Contains(owner.Note, "Kein Mieter") {
					t.Fatal(owner)
				}
			}
		})
	}
}
func TestPartyHeatingMonthlyInterimVATAndConservation(t *testing.T) {
	for _, interim := range []bool{false, true} {
		for _, vat := range []bool{false, true} {
			in := datedInput(true)
			in.Structure.Legal.ShowVAT = vat
			in.Structure.CostTypes[0].VATRatePercent = 20
			in.Receipts[1].AmountCents = 20002
			in.Structure.Legal.HeatingPrepayments["a"]["heizung"] = 1001
			in.Prepayments[0].AmountCents = 3001
			if interim {
				loc, _ := time.LoadLocation("Europe/Vienna")
				for i, item := range []struct {
					date  string
					value int64
				}{{"2025-01-01", 1000000}, {"2025-07-01", 1800000}, {"2026-01-01", 2000000}} {
					at, _ := time.ParseInLocation("2006-01-02", item.date, loc)
					e := AnnualStatementConsumptionEvidence{UnitID: "a", CostTypeKey: "heizung", SourceKind: "entity", SourceID: "meter", MeasurementUnit: "kWh", MeasuredAt: at, ValueMicros: item.value}
					if i == 1 {
						in.PartyEvidence = append(in.PartyEvidence, e)
					} else {
						in.Evidence = append(in.Evidence, e)
					}
				}
			}
			result, issues := CalculateAnnualStatementRun(in)
			if len(issues) > 0 {
				t.Fatal(issues)
			}
			old, new := partyResult(t, result, "a-old@example.com"), partyResult(t, result, "b-new@example.com")
			wantOld, wantNew := int64(21251), int64(21250)
			if interim {
				wantOld, wantNew = 26501, 16000
			}
			if old.Unit.AllocatedCents != wantOld || new.Unit.AllocatedCents != wantNew || old.Unit.PrepaidCents != 501 || new.Unit.PrepaidCents != 2500 {
				t.Fatalf("interim=%v: %+v", interim, result.PartyShares)
			}
			var cents, paid, balance, tax int64
			for _, p := range result.PartyShares {
				cents += p.Unit.AllocatedCents
				paid += p.Unit.PrepaidCents
				balance += p.Unit.BalanceCents
				for _, c := range p.Unit.Costs {
					tax += c.VATCents
					if c.AmountCents != c.NetCents+c.VATCents {
						t.Fatal(c)
					}
				}
			}
			unit := result.Units[0]
			if cents != unit.AllocatedCents || paid != unit.PrepaidCents || balance != unit.BalanceCents || tax != unit.Costs[0].VATCents {
				t.Fatal("cent loss", result)
			}
			slices.Reverse(in.Parties)
			again, issues := CalculateAnnualStatementRun(in)
			if len(issues) > 0 || !reflect.DeepEqual(result, again) {
				t.Fatal("input order changed result", issues)
			}
		}
	}
}
func TestPartyDatesRejectAmbiguousOrInvalidInputs(t *testing.T) {
	for name, mutate := range map[string]func(*AnnualStatementRunInput){
		"invalid":       func(in *AnnualStatementRunInput) { in.Parties[0].ValidTo = "2025-02-30" },
		"overlap":       func(in *AnnualStatementRunInput) { in.Parties[0].ValidTo = "2026-12-31" },
		"missing owner": func(in *AnnualStatementRunInput) { in.Parties = in.Parties[:2]; in.Parties[1].ValidTo = "2026-01-01" },
		"contract date": func(in *AnnualStatementRunInput) { in.Structure.Legal.Regime = "mrg_teil" },
		"coverage":      func(in *AnnualStatementRunInput) { in.Parties = in.Parties[:2]; in.Parties[0].ValidTo = "2025-05-31" },
	} {
		t.Run(name, func(t *testing.T) {
			in := datedInput(true)
			mutate(&in)
			r, issues := CalculateAnnualStatementRun(in)
			if len(issues) == 0 || len(r.Units) > 0 {
				t.Fatal("must block without partial money", r, issues)
			}
		})
	}
}
func TestPartyInterimReadingMustReconcileWithAnnualConsumption(t *testing.T) {
	for _, scenario := range []string{"missing boundary", "annual mismatch", "backwards", "different source"} {
		t.Run(scenario, func(t *testing.T) {
			in := datedInput(true)
			loc, _ := time.LoadLocation("Europe/Vienna")
			for i, date := range []string{"2025-01-01", "2025-07-01", "2026-01-01"} {
				at, _ := time.ParseInLocation("2006-01-02", date, loc)
				e := AnnualStatementConsumptionEvidence{UnitID: "a", CostTypeKey: "heizung", SourceKind: "entity", SourceID: "meter", MeasurementUnit: "kWh", MeasuredAt: at, ValueMicros: []int64{0, 800000, 1000000}[i]}
				if i == 1 {
					in.PartyEvidence = append(in.PartyEvidence, e)
				} else {
					in.Evidence = append(in.Evidence, e)
				}
			}
			switch scenario {
			case "missing boundary":
				in.Evidence = in.Evidence[:1]
			case "annual mismatch":
				in.Evidence[1].ValueMicros++
			case "backwards":
				in.PartyEvidence[0].ValueMicros = 2000000
			case "different source":
				in.PartyEvidence[0].SourceID = "other-meter"
			}
			r, issues := CalculateAnnualStatementRun(in)
			if len(issues) != 1 || issues[0].Code != "party-reading" || len(r.Units) > 0 {
				t.Fatal("incomplete or inconsistent readings must block", r, issues)
			}
		})
	}
}
func TestPartyHistoricalReplayAndExactMonthWeights(t *testing.T) {
	for _, version := range []int{1, 2, 3} {
		in := datedInput(true)
		in.Structure.Legal.ShowVAT = true
		in.Structure.CostTypes[0].VATRatePercent = 20
		before, issues := ReplayAnnualStatementRun(AnnualStatementRun{CalculationVersion: version, Input: in})
		if len(issues) > 0 {
			t.Fatal(issues)
		}
		in.Parties = nil
		in.StatementOn = ""
		after, issues := ReplayAnnualStatementRun(AnnualStatementRun{CalculationVersion: version, Input: in})
		if len(issues) > 0 || !reflect.DeepEqual(before, after) {
			t.Fatal("historical result changed", version, issues)
		}
	}
	in := datedInput(true)
	in.Period.StartsOn = "2024-01-01"
	in.Period.EndsOn = "2024-12-31"
	in.Parties[0].ValidTo = "2024-02-29"
	in.Parties[1].ValidFrom = "2024-03-01"
	segments, ok := annualPartySegments(in, in.Parties)
	if !ok {
		t.Fatal("segments")
	}
	weights := monthlyPartyWeights(segments, 3)
	if weights[0].Cmp(big.NewRat(2, 1)) != 0 || weights[1].Cmp(big.NewRat(10, 1)) != 0 {
		t.Fatal(weights)
	}
	cents := partyCents(math.MaxInt64, weights)
	if cents[0]+cents[1]+cents[2] != math.MaxInt64 {
		t.Fatal("overflow", cents)
	}
}

func TestPartyValidityPersistenceAndSnapshot(t *testing.T) {
	for _, kind := range []string{"memory", "sql"} {
		t.Run(kind, func(t *testing.T) {
			var sources AnnualStatementRunSources
			var storage AnnualStatementRunStorage
			if kind == "memory" {
				periods := NewMemoryAnnualStatementPeriodStore()
				units, _ := NewUnitStore("")
				docs, _ := NewDocumentStore("", filepath.Join(t.TempDir(), "docs"))
				sources = AnnualStatementRunSources{Periods: periods, Units: units, Documents: docs, Receipts: NewMemoryAnnualStatementReceiptStore(periods, NewMemoryAnnualStatementCostTypeStore(), docs), Prepayments: NewMemoryAnnualStatementPrepaymentStore(periods, units), Consumption: NewMemoryAnnualStatementConsumptionStore()}
				storage = NewMemoryAnnualStatementRunStore(sources)
			} else {
				_, lanes := testLanes(t)
				docs := NewSQLDocumentStore(lanes, filepath.Join(t.TempDir(), "docs"))
				sources = AnnualStatementRunSources{Periods: NewSQLAnnualStatementPeriodStore(lanes), Units: NewSQLUnitStore(lanes), Documents: docs, Receipts: NewSQLAnnualStatementReceiptStore(lanes), Prepayments: NewSQLAnnualStatementPrepaymentStore(lanes), Consumption: NewSQLAnnualStatementConsumptionStore(lanes)}
				storage = NewSQLAnnualStatementRunStore(lanes, docs)
			}
			tenant := testTenantRef("demo")
			in := datedInput(true)
			seedAnnualRunInput(t, sources, tenant, in)
			periods, _ := BindAnnualStatementPeriodRepository(sources.Periods, tenant)
			if err := periods.SaveLegal(2025, in.Structure.Legal); err != nil {
				t.Fatal(err)
			}
			units, _ := BindUnitRepository(sources.Units, tenant)
			update := UnitPartyUpdate{UnitID: "a", SetOwners: true, SetRenters: true, OwnerEmails: []string{"owner@example.com"}, RenterEmails: []string{"a-old@example.com", "b-new@example.com"}, Contacts: []UnitPartyContact{{Email: "a-old@example.com", ValidTo: "2025-06-30"}, {Email: "b-new@example.com", ValidFrom: "2025-07-01"}}}
			if _, err := units.UpdateParties([]UnitPartyUpdate{update}); err != nil {
				t.Fatal(err)
			}
			if len(units.UnitsForEmail("a-old@example.com")) != 0 || len(units.MembersForUnit("a").Renters) != 1 {
				t.Fatal("ended assignment grants membership")
			}
			if got := units.List()[0].PartyContacts; len(got) != 2 || got[0].ValidTo != "2025-06-30" {
				t.Fatal(got)
			}
			other, _ := BindUnitRepository(sources.Units, testTenantRef("other"))
			if len(other.List()) != 0 {
				t.Fatal("cross tenant leak")
			}
			consumption, _ := BindAnnualStatementConsumptionRepository(sources.Consumption, tenant)
			loc, _ := time.LoadLocation("Europe/Vienna")
			for _, id := range []string{"a", "b"} {
				for i, date := range []string{"2025-01-01", "2025-07-01", "2026-01-01"} {
					at, _ := time.ParseInLocation("2006-01-02", date, loc)
					value := []int64{0, 800000, 1000000}[i]
					if id == "b" {
						value *= 3
					}
					if _, _, err := consumption.Append(AnnualStatementConsumptionEvidence{UnitID: id, CostTypeKey: "heizung", SourceKind: ConsumptionSourceEntity, SourceID: "meter-" + id, MeasuredAt: at, ValueMicros: value, MeasurementUnit: "kWh", ReceivedAt: at.Add(time.Hour)}); err != nil {
						t.Fatal(err)
					}
				}
			}
			repo, _ := BindAnnualStatementRunRepository(storage, tenant)
			at := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
			run, err := repo.Create(2025, "manager@example.com", at)
			if err != nil {
				t.Fatal(err)
			}
			if run.CalculationVersion != 4 || len(run.Input.PartyEvidence) != 1 || run.Input.StatementOn != "2026-06-01" {
				t.Fatal(run)
			}
			old, next := partyResult(t, run.Result, "a-old@example.com"), partyResult(t, run.Result, "b-new@example.com")
			if old.Unit.AllocatedCents != 26500 || next.Unit.AllocatedCents != 16000 || !strings.Contains(old.Note, "Zwischenablesung") {
				t.Fatal("persisted run did not use the recorded 80/20 interim consumption split", run.Result.PartyShares)
			}
			replay, issues := ReplayAnnualStatementRun(run)
			if len(issues) > 0 || !reflect.DeepEqual(replay, run.Result) {
				t.Fatal("replay mismatch", issues)
			}
			update.Contacts[0].ValidTo = "2024-06-30"
			if _, err := units.UpdateParties([]UnitPartyUpdate{update}); err != nil {
				t.Fatal(err)
			}
			loaded, _, err := repo.Get(run.ID)
			if err != nil || !reflect.DeepEqual(run, loaded) {
				t.Fatal("snapshot changed", err)
			}
			invalid := update
			invalid.Contacts = []UnitPartyContact{{Email: "b-new@example.com", ValidFrom: "2026-02-30"}}
			if _, err := units.UpdateParties([]UnitPartyUpdate{invalid}); err == nil {
				t.Fatal("invalid date accepted")
			}
		})
	}
}
