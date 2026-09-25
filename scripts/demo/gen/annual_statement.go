package main

import (
	"github.com/inspr-at/hausv-org/internal/store"
	"sort"
)

// Synthetic demonstration inputs, not a legal allocation rule. The PPM sum
// covers all 18 flats and 6 parking spaces; every base is explicit.
func buildAnnualStatement(home house) any {
	costs := []map[string]any{
		{"key": "versicherung", "name": "Gebäudeversicherung", "allocation_key": store.AllocationKeyNutzwert, "amount_cents": 480000, "invoice_date": "2025-02-01", "supplier": "Musterstädter Versicherung"},
		{"key": "reinigung", "name": "Hausreinigung", "allocation_key": store.AllocationKeyFlaeche, "amount_cents": 360000, "invoice_date": "2025-06-30", "supplier": "Reinigung Holzer"},
		{"key": "abfall", "name": "Abfallentsorgung", "allocation_key": store.AllocationKeyPersonen, "amount_cents": 240000, "invoice_date": "2025-12-15", "supplier": "Entsorgung Musterstadt"},
		{"key": "heizung", "name": "Heizung", "allocation_key": store.AllocationKeyVerbrauch, "amount_cents": 900000, "operating_amount_cents": 90000, "invoice_date": "2025-12-31", "supplier": "Stadtwerke Musterstadt", "heating_category": "energie"},
		{"key": "lift", "name": "Lift", "allocation_key": store.AllocationKeyAgreed, "amount_cents": 180000, "invoice_date": "2025-12-01", "supplier": "Aufzüge Musterstadt"},
	}
	bases := []map[string]any{}
	legal := store.DefaultAnnualStatementLegalSettings()
	legal.HeizKGApplies = true
	legal.ShowVAT = true
	legal.InspectionPlace = "Hausverwaltung Musterstadt, Janusbergweg 123, 8010 Graz"
	legal.InspectionPeriod = "Nach Terminvereinbarung, Montag bis Freitag 9–12 Uhr"
	legal.InspectionContact = "vera.verwalter@musterstadt.example"
	legal.NextPrepaymentOn = "2026-11-01"
	legal.HeatableAreas = map[string]int{}
	legal.HeatingPrepayments = map[string]map[string]int64{}
	legal.MonthlyProposals = map[string]map[string]int64{}
	legal.HeatingInformation = &store.AnnualStatementHeatingInformation{
		Purchases:               []store.AnnualStatementEnergyPurchase{{CostTypeKey: "heizung", Supplier: "Stadtwerke Musterstadt", Carrier: "Fernwärme", QuantityMicros: 100_000_000_000, Unit: "kWh", PriceMicros: 90000, PriceNote: "01.01.–31.12.2025; tatsächlicher Bruttopreis zum Ablesestichtag 31.12.2025"}},
		DistrictHeatingOver20MW: true,
		TaxesNote:               "Bruttopreis inklusive 20 % Umsatzsteuer; netto 0,075 €/kWh. Keine weiteren Abgaben oder Zolltarife verrechnet.",
		FuelMix:                 "80 % Biomasse, 20 % Erdgas laut Jahresinformation 2025 der Stadtwerke Musterstadt.",
		Emissions:               "12.000 t CO2-Äquivalente/Jahr für die Versorgungsanlage, Lieferanteninformation 2025.",
		MeteringCostsNote:       "Ablesung und Abrechnung 300,00 €, im Beleg über sonstige Betriebskosten enthalten.",
		OperatingCostsNote:      "Wartung 600,00 €; Ablesung und Abrechnung 300,00 €; insgesamt 900,00 €.",
		RemoteMeters:            "yes",
		MonthlyInformation:      "Monatliche Bereitstellung während der Heizperiode durch die Stadtwerke Musterstadt; Kontakt über Hausverwaltung Musterstadt, +43 316 555 100.",
		ComplaintContact:        "Hausverwaltung Musterstadt GmbH, Musterstraße 12, 8010 Graz; +43 316 555 100.",
	}
	lift := map[string]int{}
	var eligible []string
	for _, unit := range home.Units {
		id := store.NormalizeUnitID(unit.Label)
		lift[id] = 0
		if unit.UnitType == "Wohnung" && unit.Floor != "EG" {
			eligible = append(eligible, id)
		}
	}
	sort.Strings(eligible)
	for i, id := range eligible {
		lift[id] = 1_000_000 / len(eligible)
		if i < 1_000_000%len(eligible) {
			lift[id]++
		}
	}
	legal.AgreedShares = map[string]map[string]int{"lift": lift}
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
		legal.MonthlyProposals[id]["lift"] = (180000*int64(lift[id]) + 6_000_000) / 12_000_000
		bases = append(bases, map[string]any{"unit_id": store.NormalizeUnitID(unit.Label), "miteigentumsanteil_ppm": ppm, "usable_area_m2_hundredths": area, "persons": persons, "prepaid_cents": prepaid, "heating_consumption_kwh": kwh})
	}
	changes := []map[string]string{{"unit_id": "top-3", "role": "owner", "previous_email": "clara.berger@musterstadt.example", "current_email": "daniel.leitner@musterstadt.example", "changed_on": "2025-07-01"}}
	return map[string]any{"party_changes": changes, "legal": legal, "house": home.Slug, "year": 2025, "starts_on": "2025-01-01", "ends_on": "2025-12-31", "recorded_at": "2026-01-15T09:00:00Z", "cost_types": costs, "unit_bases": bases}
}
