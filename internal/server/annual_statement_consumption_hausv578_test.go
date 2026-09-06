package server

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
)

func TestAnnualStatementConsumptionPeriodSharesAndBlockedPage(t *testing.T) {
	const email = "manager@example.com"
	a := newTestPortalApp(t, userProfile{Email: email, Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	repos := testRepositories(a, "demo")
	units := []store.Unit{{ID: "top-1", Label: "Top 1"}, {ID: "top-2", Label: "Top 2"}}
	if err := repos.units.SetUnits(units); err != nil {
		t.Fatal(err)
	}
	period := store.AnnualStatementPeriod{Year: 2025, StartsOn: "2025-01-01", EndsOn: "2025-12-31", UpdatedBy: email}
	costs := []store.AnnualStatementCostType{{Key: "heizung", Name: "Heizung", Allocatable: true, AllocationKey: store.AllocationKeyVerbrauch, UpdatedBy: email}}
	if _, err := repos.annualStatementPeriods.SaveWithStructure(period, costs, units); err != nil {
		t.Fatal(err)
	}
	location, err := time.LoadLocation("Europe/Vienna")
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, location)
	end := start.AddDate(1, 0, 0)
	appendReading := func(unit string, at time.Time, value int64) {
		t.Helper()
		if _, _, err := repos.annualConsumption.Append(store.AnnualStatementConsumptionEvidence{UnitID: unit, CostTypeKey: "heizung", SourceKind: store.ConsumptionSourceEntity, SourceID: "sensor." + unit, MeasuredAt: at, ValueMicros: value, MeasurementUnit: "kWh"}); err != nil {
			t.Fatal(err)
		}
	}
	appendReading("top-1", start, 100_000_000)
	appendReading("top-1", end, 101_000_000)
	appendReading("top-2", start, 500_000_000)
	page := authedRequest(t, a, email, "/demo/app/settings/annual-statement?year=2025")
	if page.Code != http.StatusOK {
		t.Fatalf("status=%d", page.Code)
	}
	for _, want := range []string{"Verteilung blockiert", "nicht gemessen", "1,000000 kWh", "Endmessung der Abrechnungsperiode fehlt.", "kein Verbrauch geschätzt"} {
		if !strings.Contains(page.Body.String(), want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(page.Body.String(), "100,00 %") {
		t.Fatal("partial data must not be normalized to a full share")
	}
	appendReading("top-2", end, 503_000_000)
	page = authedRequest(t, a, email, "/demo/app/settings/annual-statement?year=2025")
	for _, want := range []string{"Verteilung vollständig", "25,00 %", "75,00 %", "3,000000 kWh", "Vollständig gemessen"} {
		if !strings.Contains(page.Body.String(), want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(page.Body.String(), "Verteilung blockiert") {
		t.Fatal("unused hot-water gap must not block heating")
	}
	// A different period cannot reuse this year's boundary facts.
	other := annualStatementConsumption(repos.annualConsumption, store.AnnualStatementPeriod{Year: 2024, StartsOn: "2024-01-01", EndsOn: "2024-12-31"}, units, costs)
	if other.View.Groups[0].Complete {
		t.Fatal("period evidence leaked into another year")
	}
}

type countedConsumptionRepository struct {
	store.AnnualStatementConsumptionRepository
	reports, vectors int
}

func (r *countedConsumptionRepository) ConsumptionReport(period store.AnnualStatementPeriod, key string, ids []string, location *time.Location) (store.AnnualStatementConsumptionReport, error) {
	r.reports++
	return r.AnnualStatementConsumptionRepository.ConsumptionReport(period, key, ids, location)
}
func (r *countedConsumptionRepository) ConsumptionVector(period store.AnnualStatementPeriod, key string, ids []string, location *time.Location) (store.AnnualStatementConsumptionVector, error) {
	r.vectors++
	return r.AnnualStatementConsumptionRepository.ConsumptionVector(period, key, ids, location)
}
func TestAnnualStatementConsumptionReadsEachUsedKindOnce(t *testing.T) {
	repo, _ := store.BindAnnualStatementConsumptionRepository(store.NewMemoryAnnualStatementConsumptionStore(), testTenantRef("demo"))
	counted := &countedConsumptionRepository{AnnualStatementConsumptionRepository: repo}
	units := []store.Unit{{ID: "a"}, {ID: "b"}, {ID: "c"}}
	period := store.AnnualStatementPeriod{Year: 2025, StartsOn: "2025-01-01", EndsOn: "2025-12-31"}
	unused := annualStatementConsumption(counted, period, units, nil)
	if len(unused.View.Groups) != 0 || counted.reports != 0 || counted.vectors != 0 {
		t.Fatal("unused measurement kinds queried or shown")
	}
	used := annualStatementConsumption(counted, period, units, []store.AnnualStatementCostType{{Key: "heizung", Allocatable: true, AllocationKey: store.AllocationKeyVerbrauch}})
	if len(used.View.Groups) != 1 || counted.reports != 1 || counted.vectors != 0 {
		t.Fatalf("report queries=%d vector queries=%d groups=%d", counted.reports, counted.vectors, len(used.View.Groups))
	}
}
