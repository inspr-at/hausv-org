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

func buildDemoBallots() []map[string]any {
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
			"votes": map[string]any{"alina.eigentuemer@musterstadt.example": map[string]any{"option": "Ja",
				"weight": 1000000,
				"at":     "2026-09-03T10:00:00+02:00"}}},
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
	adminHouses, clerkHouses := map[string]string{}, map[string]string{}
	for _, house := range houseSpecs {
		adminHouses[house.slug] = "Admin"
		clerkHouses[house.slug] = "Verwalter"
	}
	adminHouses[zinshausSlug] = "Admin"
	clerkHouses[zinshausSlug] = "Verwalter"
	return []map[string]any{
		{"email": "vera.verwalter@musterstadt.example", "role": "admin", "granted": adminHouses},
		{"email": "paul.verwalter@musterstadt.example", "role": "sachbearbeiter", "granted": clerkHouses},
	}
}
