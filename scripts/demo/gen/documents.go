package main

import (
	"crypto/sha256"
	"fmt"

	"github.com/inspr-at/hausv-org/internal/pdf"
	"github.com/inspr-at/hausv-org/internal/store"
)

func buildDocuments(houses []house) []map[string]any {
	kinds := []struct {
		key, title, category string
		lines                []string
	}{
		{"hausordnung", "Hausordnung", store.DocumentCategoryRules, []string{"Gültig ab 01.01.2026", "Ruhezeiten: 22:00 bis 06:00 Uhr; bitte Rücksicht auf Nachbarn nehmen.", "Stiegenhaus und Fluchtwege freihalten. Fahrräder im Fahrradraum abstellen.", "Abfälle getrennt in den gekennzeichneten Behältern entsorgen.", "Schäden an Gemeinschaftsflächen bitte über Anliegen im Portal melden."}},
		{"protokoll-2025", "Protokoll der Eigentümerversammlung 2025", store.DocumentCategoryProtocol, []string{"Versammlung am 18.06.2025, 18:00 Uhr im Gemeinschaftsraum", "Die Verwaltung berichtet über Instandhaltung und laufende Betriebskosten.", "Beschluss: jährliche Liftwartung fortführen; drei Angebote wurden verglichen.", "Die Jahresabrechnung 2025 wird nach Abschluss des Wirtschaftsjahres vorgelegt.", "Nächster Schritt: Angebote zur Fassadenpflege durch den Beirat prüfen lassen."}},
		{"liftwartung", "Wartungsvertrag Lift", store.DocumentCategoryContract, []string{"Vertragslaufzeit: 01.01.2026 bis 31.12.2026", "Leistung: regelmäßige Wartung des Personenaufzugs und Funktionsprüfung.", "Enthalten: Kontrolle der Türen, Notrufeinrichtung und Sicherheitsschalter.", "Termine werden rechtzeitig am Aushang bekannt gegeben.", "Bei Störungen bitte die Verwaltung verständigen; Notruf im Lift verwenden."}},
		{"jahresabrechnung-2025", "Jahresabrechnung 2025", store.DocumentCategoryBilling, []string{"Abrechnungszeitraum: 01.01.2025 bis 31.12.2025", "Allgemeine Übersicht der Liegenschaft; keine personenbezogenen Einzelbeträge.", "Reinigung: 4.800,00 EUR | Wasser und Kanal: 6.200,00 EUR", "Liftwartung: 3.600,00 EUR | Versicherung: 2.400,00 EUR", "Summe der dargestellten Kosten: 17.000,00 EUR", "Die persönliche Abrechnung erhalten Sie gesondert von der Verwaltung."}},
		{"energieausweis", "Energieausweis", store.DocumentCategoryOther, []string{"Ausstellungsdatum: 15.03.2025 | Gültig bis: 14.03.2035", "Gebäudenutzung: Mehrparteienwohnhaus", "Heizwärmebedarf: 68 kWh/m²a | Gesamtenergieeffizienz-Faktor: 1,25", "Empfehlung: Dämmung der obersten Geschoßdecke und hydraulischer Abgleich.", "Die Kennzahlen sind Beispielwerte für diese Liegenschaft."}},
		{"fassadenangebot", "Angebot Fassadensanierung", store.DocumentCategoryContract, []string{"Angebot vom 02.09.2026 zur Vorprüfung durch den Beirat", "Leistung: Gerüst, Reinigung und Ausbesserung der hofseitigen Fassade.", "Unverbindlicher Beispielpreis: 48.000,00 EUR inklusive Umsatzsteuer.", "Ausführung nach Freigabe und Abstimmung mit der Eigentümergemeinschaft.", "Nur für Beirat und Verwaltung freigegeben; noch kein Auftrag erteilt."}},
	}
	var documents []map[string]any
	for _, house := range houses {
		for _, kind := range kinds {
			id := "demo-document-" + kind.key
			lines := []string{kind.title, house.Name, house.Address, "Hausverwaltung Musterstadt GmbH", ""}
			for _, line := range kind.lines {
				lines = append(lines, pdf.WrapText(line, 88)...)
			}
			data := pdf.Simple(lines)
			visibility := store.DocumentVisibilityAllResidents
			if kind.key == "fassadenangebot" {
				visibility = store.DocumentVisibilityBoardOnly
			}
			documents = append(documents, map[string]any{
				"id": id, "house": house.Slug, "title": kind.title, "category": kind.category,
				"visibility": visibility, "filename": id + ".pdf", "uploaded_at": "2026-09-01T08:00:00Z",
				"pdf": data, "sha256": fmt.Sprintf("%x", sha256.Sum256(data)),
			})
		}
	}
	return documents
}
