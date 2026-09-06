package main

import "github.com/inspr-at/hausv-org/internal/store"

// Synthetic demonstration inputs, not a legal allocation rule. The PPM sum
// covers all 18 flats and 6 parking spaces; every base is explicit.
func buildAnnualStatement(home house) any {
	costs := []map[string]any{
		{"key": "versicherung", "name": "Gebäudeversicherung", "allocation_key": store.AllocationKeyNutzwert, "amount_cents": 480000, "invoice_date": "2025-02-01"},
		{"key": "reinigung", "name": "Hausreinigung", "allocation_key": store.AllocationKeyFlaeche, "amount_cents": 360000, "invoice_date": "2025-06-30"},
		{"key": "abfall", "name": "Abfallentsorgung", "allocation_key": store.AllocationKeyPersonen, "amount_cents": 240000, "invoice_date": "2025-12-15"},
	}
	bases := []map[string]any{}
	for i, unit := range home.Units {
		ppm := 1_000_000 / len(home.Units)
		if i < 1_000_000%len(home.Units) {
			ppm++
		}
		area, persons, prepaid := 5500+(i%5)*500, 1+i%3, 60000
		if unit.UnitType == "Stellplatz" {
			area, persons, prepaid = 1200, 0, 20000
		}
		bases = append(bases, map[string]any{"unit_id": store.NormalizeUnitID(unit.Label), "miteigentumsanteil_ppm": ppm, "usable_area_m2_hundredths": area, "persons": persons, "prepaid_cents": prepaid})
	}
	return map[string]any{"house": home.Slug, "year": 2025, "starts_on": "2025-01-01", "ends_on": "2025-12-31", "recorded_at": "2026-01-15T09:00:00Z", "cost_types": costs, "unit_bases": bases}
}
