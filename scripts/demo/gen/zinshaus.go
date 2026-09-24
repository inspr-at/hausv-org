package main

import "github.com/inspr-at/hausv-org/internal/store"

const (
	zinshausSlug    = "musterstrasse-12"
	zinshausName    = "Musterstraße 12 · Zinshaus"
	zinshausAddress = "Musterstraße 12, 8010 Graz"
	zinshausOwner   = "stiftung.hofbauer@musterstadt.example"
)

// buildZinshaus adds the MRG apartment building beside the WEG portfolio.
// Vera and Paul already manage every other house; they manage this one too.
// The landlord is a party, not a login.
func buildZinshaus(people []person) (house, []person) {
	people[0].Memberships = append(people[0].Memberships, membership{House: zinshausSlug, Role: "Admin"})
	people[1].Memberships = append(people[1].Memberships, membership{House: zinshausSlug, Role: "Verwalter"})
	specs := zinshausUnits()
	ownerUnits := make([]string, 0, len(specs))
	for _, spec := range specs {
		ownerUnits = append(ownerUnits, spec.label)
	}
	people = append(people, person{
		Name: "Familie Hofbauer Privatstiftung", Email: zinshausOwner, Phone: "+43 316 555 180",
		Address: zinshausAddress, Memberships: []membership{{House: zinshausSlug, Role: "Eigentümer", Units: ownerUnits}},
	})
	for _, spec := range specs {
		if spec.tenant == "" {
			continue
		}
		people = append(people, person{
			Name: spec.tenantName, Email: spec.tenant, Phone: spec.phone,
			Address:     "Musterstraße 12/" + spec.door + "\n8010 Graz",
			Memberships: []membership{{House: zinshausSlug, Role: "Mieter", Units: []string{spec.label}}},
		})
	}
	units := make([]unit, 0, len(specs))
	for _, spec := range specs {
		units = append(units, unit{Label: spec.label, Floor: spec.floor, UnitType: spec.kind, OwnerEmail: zinshausOwner, TenantEmail: spec.tenant})
	}
	return house{Slug: zinshausSlug, Name: zinshausName, Address: zinshausAddress, Organisation: "musterstadt", Units: units}, people
}

type zinshausUnit struct {
	label, floor, kind, door, tenant, tenantName, phone string
	area, persons, kwh                                  int
	vacantFrom, vacantTo                                string
}

func zinshausUnits() []zinshausUnit {
	return []zinshausUnit{
		{label: "Geschäft", floor: "EG", kind: "Geschäft", door: "Geschäft", tenant: "mode.musterstadt@musterstadt.example", tenantName: "Musterstadt Mode GmbH", phone: "+43 316 555 181", area: 11200, persons: 0, kwh: 6400},
		{label: "Top 1", floor: "1. OG", kind: "Wohnung", door: "1", tenant: "elias.berger@musterstadt.example", tenantName: "Elias Berger", phone: "+43 664 510 20 01", area: 7850, persons: 2, kwh: 5100},
		{label: "Top 2", floor: "1. OG", kind: "Wohnung", door: "2", tenant: "lena.krainer@musterstadt.example", tenantName: "Lena Krainer", phone: "+43 664 510 20 02", area: 6420, persons: 1, kwh: 4300},
		{label: "Top 3", floor: "2. OG", kind: "Wohnung", door: "3", tenant: "markus.fellner@musterstadt.example", tenantName: "Markus Fellner", phone: "+43 664 510 20 03", area: 9100, persons: 3, kwh: 7200},
		{label: "Top 4", floor: "2. OG", kind: "Wohnung", door: "4", tenant: "sara.holzer@musterstadt.example", tenantName: "Sara Holzer", phone: "+43 664 510 20 04", area: 5680, persons: 1, kwh: 3900},
		{label: "Top 5", floor: "3. OG", kind: "Wohnung", door: "5", tenant: "tim.oswald@musterstadt.example", tenantName: "Tim Oswald", phone: "+43 664 510 20 05", area: 7340, persons: 2, kwh: 4800},
		{label: "Top 6", floor: "3. OG", kind: "Wohnung", door: "6", area: 4960, persons: 0, kwh: 2100, vacantFrom: "2025-03-01", vacantTo: "2025-08-31"},
		{label: "Top 7", floor: "4. OG", kind: "Wohnung", door: "7", tenant: "nora.binder@musterstadt.example", tenantName: "Nora Binder", phone: "+43 664 510 20 07", area: 10240, persons: 3, kwh: 8100},
		{label: "Top 8", floor: "4. OG", kind: "Wohnung", door: "8", tenant: "julian.eder@musterstadt.example", tenantName: "Julian Eder", phone: "+43 664 510 20 08", area: 6150, persons: 2, kwh: 4500},
		{label: "Top 9", floor: "5. OG", kind: "Wohnung", door: "9", tenant: "amira.celik@musterstadt.example", tenantName: "Amira Celik", phone: "+43 664 510 20 09", area: 8730, persons: 2, kwh: 6100},
		{label: "Top 10", floor: "5. OG", kind: "Wohnung", door: "10", tenant: "paul.reiter.m12@musterstadt.example", tenantName: "Paul Reiter", phone: "+43 664 510 20 10", area: 5210, persons: 1, kwh: 3600},
	}
}

func zinshausPPM(areas []int) []int {
	sum := 0
	for _, area := range areas {
		sum += area
	}
	out := make([]int, len(areas))
	left := 1_000_000
	type rest struct{ index, value int }
	rests := make([]rest, len(areas))
	for i, area := range areas {
		out[i] = area * 1_000_000 / sum
		left -= out[i]
		rests[i] = rest{i, area * 1_000_000 % sum}
	}
	for i := 0; i < len(rests); i++ {
		for j := i + 1; j < len(rests); j++ {
			if rests[j].value > rests[i].value || (rests[j].value == rests[i].value && rests[j].index < rests[i].index) {
				rests[i], rests[j] = rests[j], rests[i]
			}
		}
	}
	for n := 0; n < left; n++ {
		out[rests[n].index]++
	}
	return out
}

// buildZinshausStatement is the 2025 MRG period: Vollanwendung, central heat
// at 65 percent consumption, VAT on, and the statutory § 21 positions.
// Verwaltungshonorar is 11 × 12 × 18,00 € = 2.376,00 €. That sits under the
// Kategorie-A monthly cap (about 26 € per object in 2025).
func buildZinshausStatement(home house) map[string]any {
	specs := zinshausUnits()
	areas := make([]int, len(specs))
	for i, spec := range specs {
		areas[i] = spec.area
	}
	ppm := zinshausPPM(areas)
	legal := store.DefaultAnnualStatementLegalSettings()
	legal.Regime = "mrg_voll"
	legal.HeizKGApplies = true
	legal.HeatingConsumptionPercent = 65
	legal.ShowVAT = true
	legal.InspectionPlace = "Hausverwaltung Musterstadt, Musterstraße 12, 8010 Graz"
	legal.InspectionPeriod = "Nach Terminvereinbarung, Montag bis Freitag 9–12 Uhr"
	legal.InspectionContact = "vera.verwalter@musterstadt.example"
	legal.NextPrepaymentOn = "2026-07-01"
	legal.HeatableAreas = map[string]int{}
	legal.HeatingPrepayments = map[string]map[string]int64{}
	legal.MonthlyProposals = map[string]map[string]int64{}
	bases := make([]map[string]any, 0, len(specs))
	for i, spec := range specs {
		id := store.NormalizeUnitID(spec.label)
		legal.HeatableAreas[id] = spec.area
		legal.HeatingPrepayments[id] = map[string]int64{"heizung": 18000}
		legal.MonthlyProposals[id] = map[string]int64{"wasser": 1400, "muell": 900, "versicherung": 1600, "hausbetreuung": 2800, "verwaltungshonorar": 1800}
		row := map[string]any{
			"unit_id": id, "miteigentumsanteil_ppm": ppm[i], "usable_area_m2_hundredths": spec.area,
			"persons": spec.persons, "prepaid_cents": 84000, "heating_consumption_kwh": spec.kwh,
		}
		if spec.vacantFrom != "" {
			row["vacant_from"] = spec.vacantFrom
			row["vacant_to"] = spec.vacantTo
			row["prepaid_cents"] = 42000
		}
		bases = append(bases, row)
	}
	costs := []map[string]any{
		{"key": "wasser", "name": "Wasser und Abwasser", "allocation_key": store.AllocationKeyNutzwert, "amount_cents": 186000, "invoice_date": "2025-03-15", "supplier": "Wasserwerk Musterstadt"},
		{"key": "muell", "name": "Müllabfuhr", "allocation_key": store.AllocationKeyPersonen, "amount_cents": 132000, "invoice_date": "2025-11-30", "supplier": "Entsorgung Musterstadt"},
		{"key": "versicherung", "name": "Gebäudeversicherung", "allocation_key": store.AllocationKeyNutzwert, "amount_cents": 248000, "invoice_date": "2025-02-01", "supplier": "Musterstädter Versicherung"},
		{"key": "hausbetreuung", "name": "Hausbetreuung", "allocation_key": store.AllocationKeyFlaeche, "amount_cents": 420000, "invoice_date": "2025-12-20", "supplier": "Hausbetreuung Lindner"},
		{"key": "verwaltungshonorar", "name": "Verwaltungshonorar", "allocation_key": store.AllocationKeyNutzwert, "amount_cents": 237600, "invoice_date": "2025-12-31", "supplier": "Hausverwaltung Musterstadt"},
		{"key": "heizung", "name": "Heizung", "allocation_key": store.AllocationKeyVerbrauch, "amount_cents": 640000, "operating_amount_cents": 96000, "invoice_date": "2025-12-31", "supplier": "Wärmeversorgung Musterstadt", "heating_category": "energie"},
	}
	return map[string]any{
		"legal": legal, "house": home.Slug, "address": zinshausAddress, "year": 2025,
		"starts_on": "2025-01-01", "ends_on": "2025-12-31", "recorded_at": "2026-01-15T09:00:00Z",
		"built_year": 1905, "cost_types": costs, "unit_bases": bases,
	}
}

func zinshausAnnouncement(slug string, number int) announcement {
	return announcement{
		ID: "an-" + sprintf3(number), House: slug, Category: "Info", PublishedAt: "2026-09-08T08:00:00+02:00",
		Title: "Betriebskosten 2025 liegen auf",
		Body:  "Musterstraße 12 ist ein Zinshaus aus dem Jahr 1905 in voller Anwendung des MRG. Die Betriebskostenabrechnung 2025 (Heizkosten nach HeizKG, Umsatzsteuer 10 und 20 Prozent, Eigentümeranteil für den Leerstand von Top 6) liegt in der Verwaltung zur Einsicht auf.",
	}
}

func sprintf3(n int) string {
	if n < 10 {
		return "00" + itoa(n)
	}
	if n < 100 {
		return "0" + itoa(n)
	}
	return itoa(n)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// buildZinshausLeases covers every let unit. Last valorisations sit in 2024
// so the April 2026 draft still has work. Richtwert, Kategorie and
// angemessener Mietzins carry the 1 percent MieWeG cap; the free residential
// lease carries the 2 percent cap. The shop is commercial at a 3 percent threshold.
func buildZinshausLeases() []map[string]any {
	vpi := func(id, unit, name, email, regime string, hmz int64, threshold string) map[string]any {
		return map[string]any{
			"id": id, "house": zinshausSlug, "unit": unit, "tenant_name": name, "tenant_email": email,
			"tenant_address": "Musterstraße 12, 8010 Graz", "concluded_on": "2019-03-01", "starts_on": "2019-04-01",
			"use_kind": "wohnung", "mrg_scope": "voll", "rent_regime": regime,
			"hmz_cents": hmz, "bk_cents": int64(14000), "heat_cents": int64(8000), "vat_bp": 1000,
			"clause_type": "vpi_threshold", "series": "vpi2020", "base_period": "2024-09", "base_value": "123.6",
			"threshold": threshold, "threshold_kind": "percent", "review_status": "ok",
			"clause_text": "Der Hauptmietzins ist wertgesichert nach VPI 2020. Ausgang September 2024, volle Veränderung, in beide Richtungen.",
			"last_month":  "2024-09", "last_value": "123.6", "hmz_after": centsEuro(hmz),
		}
	}
	shop := map[string]any{
		"id": "m12-lease-geschaeft", "house": zinshausSlug, "unit": "Geschäft",
		"tenant_name": "Musterstadt Mode GmbH", "tenant_email": "mode.musterstadt@musterstadt.example",
		"tenant_address": "Musterstraße 12, Geschäft, 8010 Graz", "concluded_on": "2022-01-10", "starts_on": "2022-02-01",
		"use_kind": "geschaeft", "mrg_scope": "voll", "rent_regime": "frei",
		"hmz_cents": int64(185000), "bk_cents": int64(22000), "heat_cents": int64(9000), "vat_bp": 2000,
		"clause_type": "vpi_threshold", "series": "vpi2020", "base_period": "2024-09", "base_value": "123.6",
		"threshold": "3", "threshold_kind": "percent", "review_status": "ok", "tenant_is_consumer": false,
		"clause_text": "Der Geschäftsmietzins ist nach VPI 2020 wertgesichert. Basis September 2024, Schwelle 3 Prozent, volle Veränderung in beide Richtungen.",
		"last_month":  "2024-09", "last_value": "123.6", "hmz_after": "1850.00",
		"notes": "Geschäftslokal im Erdgeschoss, kein MieWeG.",
	}
	free := vpi("m12-lease-top-7", "Top 7", "Nora Binder", "nora.binder@musterstadt.example", "frei", 128000, "5")
	free["clause_type"] = "mieweg_model"
	free["clause_text"] = "Der Hauptmietzins ist wertgesichert gemäß § 1 Abs 2 MieWeG. Letzte Anpassung September 2024."
	free["notes"] = "Freier Mietzins, Sonderdeckel 2 Prozent."
	delete(free, "threshold")
	delete(free, "threshold_kind")
	staffel := map[string]any{
		"id": "m12-lease-top-8", "house": zinshausSlug, "unit": "Top 8",
		"tenant_name": "Julian Eder", "tenant_email": "julian.eder@musterstadt.example",
		"tenant_address": "Musterstraße 12/8, 8010 Graz", "concluded_on": "2021-04-01", "starts_on": "2021-05-01",
		"use_kind": "wohnung", "mrg_scope": "voll", "rent_regime": "angemessen",
		"hmz_cents": int64(91000), "bk_cents": int64(13000), "heat_cents": int64(7500), "vat_bp": 1000,
		"clause_type": "staffel", "review_status": "ok",
		"clause_text": "Der Hauptmietzins steigt jeweils am 1. April um 2 Prozent, unabhängig vom Index. Letzte Staffel April 2024.",
		"notes":       "Staffelmietzins, letzte Stufe 2024.",
	}
	return []map[string]any{
		shop,
		vpi("m12-lease-top-1", "Top 1", "Elias Berger", "elias.berger@musterstadt.example", "richtwert", 72000, "5"),
		vpi("m12-lease-top-2", "Top 2", "Lena Krainer", "lena.krainer@musterstadt.example", "richtwert", 64000, "5"),
		vpi("m12-lease-top-3", "Top 3", "Markus Fellner", "markus.fellner@musterstadt.example", "angemessen", 98000, "5"),
		vpi("m12-lease-top-4", "Top 4", "Sara Holzer", "sara.holzer@musterstadt.example", "angemessen", 69000, "5"),
		vpi("m12-lease-top-5", "Top 5", "Tim Oswald", "tim.oswald@musterstadt.example", "richtwert", 81000, "40"),
		free,
		staffel,
		vpi("m12-lease-top-9", "Top 9", "Amira Celik", "amira.celik@musterstadt.example", "richtwert", 88000, "5"),
		vpi("m12-lease-top-10", "Top 10", "Paul Reiter", "paul.reiter.m12@musterstadt.example", "kategorie", 61000, "5"),
	}
}

func centsEuro(cents int64) string {
	whole := cents / 100
	frac := cents % 100
	if frac < 10 {
		return itoa(int(whole)) + ".0" + itoa(int(frac))
	}
	return itoa(int(whole)) + "." + itoa(int(frac))
}
