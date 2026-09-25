package main

func buildDemoContacts() []map[string]any {
	return []map[string]any{
		{"id": "demo-contact-lift",
			"kind":       "Notdienst",
			"company":    "Musterstadt Liftservice",
			"email":      "lift@musterstadt.example",
			"phone":      "+43 316 555 210",
			"notes":      "Aufzugwartung und Liftstörungen am Janusbergweg 123. Notruf bei eingeschlossenen Personen rund um die Uhr; Standort und Liftnummer bereithalten.",
			"active":     true,
			"created_at": "2026-09-01T09:00:00+02:00",
			"updated_at": "2026-09-01T09:00:00+02:00"},
		{"id": "demo-contact-waerme",
			"kind":       "Dienstleister",
			"company":    "Musterstadt Wärme & Wasser",
			"email":      "waerme@musterstadt.example",
			"phone":      "+43 316 555 211",
			"notes":      "Heizung, Sanitär und Wasserleitungen. Termine koordiniert Paul Sommer. Bei Wasseraustritt sofort Verwaltung und Notdienst verständigen.",
			"active":     true,
			"created_at": "2026-09-01T09:00:00+02:00",
			"updated_at": "2026-09-01T09:00:00+02:00"},
		{"id": "demo-contact-elektrik",
			"kind":       "Dienstleister",
			"company":    "Musterstadt Elektrotechnik",
			"email":      "elektrik@musterstadt.example",
			"phone":      "+43 316 555 212",
			"notes":      "Beleuchtung, Gegensprechanlage und Allgemeinstrom. Störungen mit Ort und betroffener Sicherung an die Verwaltung melden.",
			"active":     true,
			"created_at": "2026-09-01T09:00:00+02:00",
			"updated_at": "2026-09-01T09:00:00+02:00"},
		{"id": "demo-contact-hausbetreuung",
			"kind":       "Hausmeister",
			"company":    "Musterstadt Hausbetreuung",
			"email":      "hausbetreuung@musterstadt.example",
			"phone":      "+43 316 555 213",
			"notes":      "Reinigung, Winterdienst und Grünflächen am Janusbergweg 123. Erreichbar Mo–Fr 7–16 Uhr; Hinweise zu Glätte bitte telefonisch melden.",
			"active":     true,
			"created_at": "2026-09-01T09:00:00+02:00",
			"updated_at": "2026-09-01T09:00:00+02:00"}}
}

func buildDemoBallots(home house) []map[string]any {
	// Same real unit shares as the annual-statement seed, summed per owner.
	weights := map[string]int{}
	for i, unit := range home.Units {
		weights[unit.OwnerEmail] += demoUnitSharePPM(i, len(home.Units))
	}
	votes := map[string]any{}
	for i, owner := range []string{"alina.eigentuemer", "hedwig.beirat", "jonas.meier", "anna.gruber", "nina.hofer", "felix.wagner", "laura.schmid", "sara.klein", "thomas.berger"} {
		email := owner + "@musterstadt.example"
		option := "Ja"
		if i == 7 {
			option = "Nein"
		}
		if i == 8 {
			option = "Enthaltung"
		}
		votes[email] = map[string]any{"option": option, "weight": weights[email], "at": "2026-09-03T10:00:00+02:00"}
	}
	return []map[string]any{
		{"id": "demo-ballot-fassade-2027",
			"title":       "Sanierung Fassade 2027",
			"description": "Die Fassade soll 2027 instand gesetzt werden. Die Verwaltung holt drei vergleichbare Angebote ein und legt Kosten und Bauzeiten vor einer Beauftragung zur Freigabe vor.",
			"options":     []string{"Ja", "Nein", "Enthaltung"},
			"type":        "Umlaufbeschluss",
			"weighting":   "per-share",
			"quorum_ppm":  500000,
			"opens_at":    "2026-08-20T09:00:00+02:00",
			"closes_at":   "2026-09-04T18:00:00+02:00",
			"created_by":  "vera.verwalter@musterstadt.example",
			"created_at":  "2026-08-19T09:00:00+02:00",
			"updated_at":  "2026-09-04T18:00:00+02:00",
			"status":      "closed",
			"votes":       votes},
		{"id": "demo-ballot-ebikes",
			"title":       "Fahrradraum: Ladepunkte für E-Bikes",
			"description": "Soll die Verwaltung die technische Machbarkeit und Kosten von vier abschließbaren Ladepunkten prüfen lassen? Die Prüfung umfasst Stromversorgung, Brandschutz und eine getrennte Verbrauchserfassung. Über die Umsetzung wird anschließend gesondert entschieden.",
			"options":     []string{"Ja", "Nein", "Enthaltung"},
			"type":        "Umlaufbeschluss",
			"weighting":   "per-share",
			"quorum_ppm":  500000,
			"opens_at":    "2026-09-07T09:00:00+02:00",
			"closes_at":   "2026-09-23T18:00:00+02:00",
			"created_by":  "paul.verwalter@musterstadt.example",
			"created_at":  "2026-09-06T09:00:00+02:00",
			"updated_at":  "2026-09-07T09:00:00+02:00",
			"status":      "open"}}
}

func buildDemoMembers() []map[string]any {
	_, people := buildHousesAndPersons()
	_, people = buildZinshaus(people)
	var members []map[string]any
	for i, role := range []string{"admin", "sachbearbeiter"} {
		previous := map[string]any{}
		for _, membership := range people[i].Memberships {
			previous[membership.House] = map[string]any{
				"role": membership.Role, "permissions": []string{},
				"status": "Aktiv", "directory_opt_in": nil,
			}
		}
		members = append(members, map[string]any{
			"email": people[i].Email, "role": role, "previous_memberships": previous,
		})
	}
	return members
}

func demoUnitSharePPM(index, count int) int {
	ppm := 1_000_000 / count
	if index < 1_000_000%count {
		ppm++
	}
	return ppm
}
