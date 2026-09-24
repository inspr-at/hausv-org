package main

import "github.com/inspr-at/hausv-org/internal/store"

// Synthetic demonstration inputs, not a legal allocation rule. The PPM sum
// covers all 18 flats and 6 parking spaces; every base is explicit.
func buildAnnualStatement(home house) any {
	costs := []map[string]any{
		{"key": "versicherung", "name": "Gebäudeversicherung", "allocation_key": store.AllocationKeyNutzwert, "amount_cents": 480000, "invoice_date": "2025-02-01", "supplier": "Musterstadt Versicherung"},
		{"key": "reinigung", "name": "Hausreinigung", "allocation_key": store.AllocationKeyFlaeche, "amount_cents": 360000, "invoice_date": "2025-06-30", "supplier": "Musterstadt Reinigung"},
		{"key": "abfall", "name": "Abfallentsorgung", "allocation_key": store.AllocationKeyPersonen, "amount_cents": 240000, "invoice_date": "2025-12-15", "supplier": "Stadtservice Musterstadt"},
		{"key": "heizung", "name": "Heizung", "allocation_key": store.AllocationKeyVerbrauch, "amount_cents": 900000, "operating_amount_cents": 90000, "invoice_date": "2025-12-31", "supplier": "Wärme Musterstadt", "heating_category": "energie"},
	}
	bases := []map[string]any{}
	legal := store.DefaultAnnualStatementLegalSettings()
	legal.HeizKGApplies = true
	legal.InspectionPlace = "Hausverwaltung Musterstadt, Janusbergweg 123, 8010 Graz"
	legal.InspectionPeriod = "Nach Terminvereinbarung, Montag bis Freitag 9–12 Uhr"
	legal.InspectionContact = "vera.verwalter@musterstadt.example"
	legal.NextPrepaymentOn = "2026-11-01"
	legal.HeatableAreas = map[string]int{}
	legal.HeatingPrepayments = map[string]map[string]int64{}
	legal.MonthlyProposals = map[string]map[string]int64{}
	for i, unit := range home.Units {
		ppm := 1_000_000 / len(home.Units)
		if i < 1_000_000%len(home.Units) {
			ppm++
		}
		area, persons, prepaid := 5500+(i%5)*500, 1+i%3, 60000
		if unit.UnitType == "Stellplatz" {
			area, persons, prepaid = 1200, 0, 20000
		}
		id := store.NormalizeUnitID(unit.Label)
		heatArea, heatPrepay, kwh := area, 25000, 3500+(i%6)*400
		if unit.UnitType == "Stellplatz" {
			heatArea, heatPrepay, kwh = 0, 0, 0
		}
		legal.HeatableAreas[id] = heatArea
		legal.HeatingPrepayments[id] = map[string]int64{"heizung": int64(heatPrepay)}
		legal.MonthlyProposals[id] = map[string]int64{"versicherung": 1700, "reinigung": 1500, "abfall": 700}
		if unit.UnitType == "Stellplatz" {
			legal.MonthlyProposals[id] = map[string]int64{"versicherung": 1700, "reinigung": 350, "abfall": 0}
		}
		bases = append(bases, map[string]any{"unit_id": store.NormalizeUnitID(unit.Label), "miteigentumsanteil_ppm": ppm, "usable_area_m2_hundredths": area, "persons": persons, "prepaid_cents": prepaid, "heating_consumption_kwh": kwh})
	}
	return map[string]any{"legal": legal, "house": home.Slug, "year": 2025, "starts_on": "2025-01-01", "ends_on": "2025-12-31", "recorded_at": "2026-01-15T09:00:00Z", "cost_types": costs, "unit_bases": bases}
}
