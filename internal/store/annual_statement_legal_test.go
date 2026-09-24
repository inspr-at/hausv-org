package store

import "testing"
import "reflect"

func TestAnnualLegalRegimeDeadline(t *testing.T) {
	period := AnnualStatementPeriod{Year: 2025, StartsOn: "2025-02-01", EndsOn: "2026-01-31"}
	for _, tc := range []struct {
		regime string
		heat   bool
		want   string
	}{
		{"weg", false, "2026-07-31"}, {"mrg_voll", false, "2027-06-30"},
		{"mrg_teil", false, ""}, {"ausnahme", false, ""}, {"mrg_teil", true, "2026-07-31"},
	} {
		legal := AnnualStatementLegalSettings{Regime: tc.regime, HeizKGApplies: tc.heat}
		if got := legal.Deadline(period); got != tc.want {
			t.Errorf("%s: %s != %s", tc.regime, got, tc.want)
		}
	}
}

func TestAnnualLegalSettingsPersistenceAndClone(t *testing.T) {
	_, lanes := testLanes(t)
	for _, storage := range []AnnualStatementPeriodStorage{NewMemoryAnnualStatementPeriodStore(), NewSQLAnnualStatementPeriodStore(lanes)} {
		repo, _ := BindAnnualStatementPeriodRepository(storage, testTenantRef("demo"))
		_, err := repo.Save(AnnualStatementPeriod{Year: 2025, StartsOn: "2025-01-01", EndsOn: "2025-12-31", UpdatedBy: "manager@example.com"})
		if err != nil {
			t.Fatal(err)
		}
		settings := AnnualStatementLegalSettings{Regime: "mrg_teil", HeizKGApplies: true, HeatingConsumptionPercent: 70}
		if err := repo.SaveLegal(2025, settings); err != nil {
			t.Fatal(err)
		}
		got, ok := repo.Structure(2025)
		if !ok || !reflect.DeepEqual(got.Legal, settings) {
			t.Fatal(got, ok)
		}
		if _, created, err := repo.CloneStructure(2025, AnnualStatementPeriod{Year: 2026, StartsOn: "2026-01-01", EndsOn: "2026-12-31", UpdatedBy: "manager@example.com"}); err != nil || !created {
			t.Fatal(created, err)
		}
		next, _ := repo.Structure(2026)
		if !reflect.DeepEqual(next.Legal, settings) {
			t.Fatal(next)
		}
		other, _ := BindAnnualStatementPeriodRepository(storage, testTenantRef("other"))
		if err := other.SaveLegal(2025, settings); err == nil {
			t.Fatal("foreign update")
		}
		if err := repo.SaveLegal(2025, AnnualStatementLegalSettings{Regime: "invalid"}); err == nil {
			t.Fatal("invalid regime")
		}
	}
}
