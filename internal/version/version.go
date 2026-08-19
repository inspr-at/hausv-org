// Package version carries the build identity injected at link time.
//
// Version and Commit are set with -ldflags -X. They MUST stay exported and
// this import path MUST match the Dockerfile, or the -X silently no-ops: the
// build still succeeds, the tests still pass, and every deploy ships "dev"
// forever. See TestBuildLabelUsesSemverAndCommit.
package version

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	// Version is the semver of the build. Overridden via:
	//   -X github.com/inspr-at/hausv-org/internal/version.Version=1.2.3
	Version = "dev"
	// Commit is the short git SHA. Overridden the same way.
	Commit = "dev"
	nonce  = strconv.FormatInt(time.Now().Unix(), 36)
)

// Note is one release-notes entry rendered in the app's release dialog.
type Note struct {
	Version  string
	Date     string
	Kind     string
	Headline string
	Intro    string
	Items    []NoteItem
}

type NoteItem struct {
	Label string
	Text  string
}

func BuildLabel() string {
	version := strings.TrimPrefix(strings.TrimSpace(Version), "v")
	if version == "" {
		version = "dev"
	}
	commit := strings.TrimSpace(Commit)
	if commit == "" {
		commit = "dev"
	}
	return fmt.Sprintf("%s (%s)", version, commit)
}

func AssetVersion() string {
	version := strings.TrimPrefix(strings.TrimSpace(Version), "v")
	if version == "" {
		version = "dev"
	}
	commit := strings.TrimSpace(Commit)
	parts := []string{version}
	if commit == "" || commit == "dev" {
		commit = ""
	}
	if commit != "" {
		parts = append(parts, commit)
	}
	if nonce != "" {
		parts = append(parts, nonce)
	}
	return url.QueryEscape(strings.Join(parts, "-"))
}

func DisplayVersion(version string) string {
	version = strings.TrimPrefix(strings.TrimSpace(version), "v")
	if version == "" {
		return "dev"
	}
	var b strings.Builder
	for _, r := range version {
		if (r >= '0' && r <= '9') || r == '.' {
			b.WriteRune(r)
			continue
		}
		break
	}
	if b.Len() == 0 {
		return version
	}
	return b.String()
}

func Notes() []Note {
	return []Note{
		{
			Version:  "0.98.1",
			Date:     "19. August 2026",
			Kind:     "Oberfläche",
			Headline: "Am Telefon und Tablet sitzt jetzt alles, wo es hingehört.",
			Intro:    "Eine vollständige Prüfung aller Bildschirme in allen Breiten hat einige Stellen gefunden, an denen die neue Oberfläche auf kleinen Geräten noch nicht sauber war. Alle sind behoben.",
			Items: []NoteItem{
				{Label: "Dialoge", Text: "Kopf und Speichern-Schaltfläche bleiben am Telefon immer sichtbar; nur der Inhalt scrollt."},
				{Label: "Tastziele", Text: "Schließen, Herunterladen, Aufklappen und Kontaktlinks sind am Telefon jetzt durchgehend groß genug."},
				{Label: "Verwaltung", Text: "Termine anlegen und Dokumente hochladen geht jetzt auch am Telefon."},
				{Label: "Energie", Text: "Die Sicherheitsleiste bleibt am Telefon kompakt, das Flussbild passt auf jedes Tablet, und die Kontrastfarben sind wieder da."},
			},
		},
		{
			Version:  "0.98.0",
			Date:     "18. August 2026",
			Kind:     "Oberfläche",
			Headline: "Die alte Darstellung ist entfernt — es gibt nur noch die neue Oberfläche.",
			Intro:    "Seit 0.96.0 lief das Portal auf der neuen Oberfläche; die alte lief im Hintergrund weiter mit. Sie ist jetzt vollständig entfernt. Dabei sind zwei Bedienfehler aufgefallen und behoben worden.",
			Items: []NoteItem{
				{Label: "Tastatur", Text: "Der Sprung zum Inhalt landet auf dem Telefon jetzt wirklich beim Inhalt, und Escape schließt das mobile Menü."},
				{Label: "Energie", Text: "Nach dem Schließen eines Dialogs kehrt der Fokus zur Schaltfläche zurück, von der er kam."},
				{Label: "Aufgeräumt", Text: "Rund 8.400 Zeilen der alten Darstellung sind entfernt; jede Änderung muss nur noch einmal gemacht werden."},
			},
		},
		{
			Version:  "0.97.0",
			Date:     "18. August 2026",
			Kind:     "Technik",
			Headline: "Jedes Haus hat jetzt eine dauerhafte Kennung.",
			Intro:    "Bisher war der Kurzname eines Hauses zugleich seine Kennung in der Datenbank. Ab jetzt trägt jedes Haus eine eigene, unveränderliche Kennung — die Grundlage dafür, ein Haus später umbenennen zu können, ohne dass Daten verloren gehen.",
			Items: []NoteItem{
				{Label: "Dauerhaft", Text: "Die Kennung eines Hauses bleibt gleich, auch wenn sich sein Name ändert."},
				{Label: "Sicher", Text: "Die Trennung zwischen Häusern wird jetzt strukturell erzwungen und nicht mehr an jeder einzelnen Stelle neu hergestellt."},
				{Label: "Geprüft", Text: "Automatische Prüfungen schlagen fehl, sobald eine Abfrage die Haustrennung vergisst."},
			},
		},
		{
			Version:  "0.96.0",
			Date:     "17. August 2026",
			Kind:     "Oberfläche",
			Headline: "Auch „Mein Zuhause“ läuft jetzt auf der neuen Oberfläche.",
			Intro:    "Der Energiebereich war der letzte Bereich in der alten Darstellung; damit ist das Portal durchgehend einheitlich.",
			Items: []NoteItem{
				{Label: "Vollständig", Text: "Alle Bereiche des Portals nutzen dieselbe Oberfläche."},
				{Label: "Bedienbar", Text: "Aufklappbare Inhalte reagieren wieder überall, und Tastziele sind am Telefon groß genug."},
				{Label: "Robust", Text: "Lange Haus- und Wohnungsnamen werden gekürzt statt die Ansicht zu verbreitern."},
			},
		},
		{
			Version:  "0.95.0",
			Date:     "17. August 2026",
			Kind:     "Oberfläche",
			Headline: "Das Portal läuft durchgehend auf der neuen Oberfläche.",
			Intro:    "Alle Bereiche werden einheitlich dargestellt, und die mobile Bedienung ist auf jeder Seite gleich aufgebaut.",
			Items: []NoteItem{
				{Label: "Einheitlich", Text: "Schaltflächen, Abstände und Navigation sehen in jedem Bereich gleich aus."},
				{Label: "Unterwegs", Text: "Die Navigation steht oben, und Haus und Rolle lassen sich auch am Telefon wechseln."},
			},
		},
		{
			Version:  "0.94.1",
			Date:     "15. August 2026",
			Kind:     "Erscheinungsbild",
			Headline: "Vorschau und Portal-Symbol entsprechen jetzt dem echten Auftritt.",
			Intro:    "Die Seitenleiste wird realitätsnah dargestellt und die Portal-Kennung kann jedes lokal eingebundene Lucide-SVG verwenden.",
			Items: []NoteItem{
				{Label: "Realistisch", Text: "Kartenkopf, Portalwechsel, Navigation und Konto erscheinen in der Vorschau wie im Portal."},
				{Label: "Freie Symbolwahl", Text: "Die vollständige Lucide-Bibliothek ist durchsuchbar; die bisherigen Vorgaben bleiben erhalten."},
			},
		},
		{
			Version:  "0.94.0",
			Date:     "15. August 2026",
			Kind:     "Erscheinungsbild",
			Headline: "Das Portal lässt sich jetzt mit direkter Vorschau gestalten.",
			Intro:    "Eine sichtbare SVG-Auswahl und realistische Vorschauen verbinden Portal-Kennung, Symbol und Titelbild zu einem verständlichen Arbeitsbereich.",
			Items: []NoteItem{
				{Label: "Sichtbar gewählt", Text: "Alle vorhandenen Portal-Symbole stehen als direkte Auswahl bereit."},
				{Label: "Sofort geprüft", Text: "Anmeldung und Seitenleiste zeigen Änderungen bereits vor dem Speichern."},
				{Label: "Klar gegliedert", Text: "Kennung und Titelbild bleiben getrennt gespeichert und sind dennoch gemeinsam im Kontext sichtbar."},
			},
		},
		{
			Version:  "0.93.0",
			Date:     "15. August 2026",
			Kind:     "Einheitenverwaltung",
			Headline: "Einheiten lassen sich jetzt übersichtlich und ohne Seitensprünge verwalten.",
			Intro:    "Eine klare Liste und ein eigener Bearbeitungsdialog verbinden Stammdaten, Zuordnungen und Zahlungsstatus in einem ruhigen Ablauf.",
			Items: []NoteItem{
				{Label: "Auf einen Blick", Text: "Typ, Personen, Anteil und Zahlungsstatus stehen direkt in der Einheitenliste."},
				{Label: "Im Kontext", Text: "Hinzufügen und Bearbeiten öffnen als eigener Dialog, ohne die Seite zu verschieben."},
				{Label: "Gemeinsam gespeichert", Text: "Stammdaten, Zuordnungen und Zahlungsstatus werden in einem Schritt übernommen."},
			},
		},
		{
			Version:  "0.92.0",
			Date:     "15. August 2026",
			Kind:     "Gebäudeeinstellungen",
			Headline: "Hausdaten und Kontakte lassen sich jetzt klar und direkt pflegen.",
			Intro:    "Eigene Arbeitsbereiche ersetzen die lange Einstellungsseite; eine bewusste Adresssuche verbindet Stammdaten und Kartenposition.",
			Items: []NoteItem{
				{Label: "Direkt", Text: "Stammdaten, Kontakte, Einheiten und Erscheinungsbild öffnen als getrennte Ansichten ohne Seitensprünge."},
				{Label: "Einfach verortet", Text: "Die Adresse lässt sich per Klick über OpenStreetMap suchen, prüfen und erst danach speichern."},
				{Label: "Aufgeräumt", Text: "Hausverwaltung, Notdienst und Hausmeister stehen in drei klaren Kontaktkarten."},
			},
		},
		{
			Version:  "0.91.1",
			Date:     "14. August 2026",
			Kind:     "Portalwechsel",
			Headline: "Der Kartenkopf zeigt alle eigenen Zuhause klar und vollständig.",
			Intro:    "Der Wechsler sitzt oben auf der Karte, vermeidet doppelte Ortsangaben und behält neu angelegte Portale zuverlässig in der Auswahl.",
			Items: []NoteItem{
				{Label: "Übersichtlich", Text: "Die aktive Liegenschaft erscheint einmal im oberen Kartenverlauf; lange Einträge dürfen im Menü umbrechen."},
				{Label: "Vollständig", Text: "Aktivierte Home-Portale bleiben nach jedem Wechsel auswählbar."},
			},
		},
		{
			Version:  "0.91.0",
			Date:     "14. August 2026",
			Kind:     "Portalwechsel",
			Headline: "Karte und Portalwechsel bilden jetzt einen klaren Ortskopf.",
			Intro:    "Das aktive Haus, die gewählte Rolle und die passende OpenStreetMap-Ansicht bleiben beim Wechsel sichtbar und konsistent.",
			Items: []NoteItem{
				{Label: "Kompakt", Text: "Der Portalwechsler sitzt direkt im Kartenkopf und bleibt auch auf kleinen Bildschirmen gut erreichbar."},
				{Label: "Aktuell", Text: "Name, Rolle, Adresse, Markierung und Kartenkacheln wechseln gemeinsam zur ausgewählten Liegenschaft."},
				{Label: "Konfigurierbar", Text: "Die Kartenposition kann in den Gebäudestammdaten hinterlegt oder wieder entfernt werden."},
				{Label: "Ehrlich", Text: "Ohne Koordinaten weist der Kartenkopf klar auf die fehlende Standortangabe hin."},
			},
		},
		{
			Version:  "0.90.0",
			Date:     "14. August 2026",
			Kind:     "Portalbereiche",
			Headline: "Jede Liegenschaft zeigt nur die Funktionen, die sie wirklich braucht.",
			Intro:    "Verwaltungen stellen die sichtbaren Portalbereiche zentral zusammen; Hausüberblick und Einstellungen bleiben dabei immer verfügbar.",
			Items: []NoteItem{
				{Label: "Passend zum Haus", Text: "Energie, Aushang, Termine, Kontakte, Dokumente, Anliegen und weitere Bereiche lassen sich einzeln ein- oder ausblenden."},
				{Label: "Konsequent", Text: "Ausgeschaltete Bereiche verschwinden aus Navigation, Hausüberblick und Einstellungen und sind auch über direkte Links nicht erreichbar."},
				{Label: "Sicherer Standard", Text: "Bestehende Liegenschaften starten unverändert mit allen bisherigen Funktionen."},
			},
		},
		{
			Version:  "0.89.1",
			Date:     "14. August 2026",
			Kind:     "Portalwechsel",
			Headline: "Die Rollenwahl entspricht jetzt exakt den hinterlegten Zuordnungen.",
			Intro:    "Der Portalwechsler verbindet mehrere echte Rollen innerhalb einer Liegenschaft, ohne zusätzliche Perspektiven zu erfinden.",
			Items: []NoteItem{
				{Label: "Nachvollziehbar", Text: "Eigentümer- und Mieterrollen werden aus den persönlichen Einheiten-Zuordnungen abgeleitet."},
				{Label: "Eindeutig", Text: "Nicht zugeordnete Rollen erscheinen nicht im Wechsler."},
				{Label: "Abgesichert", Text: "Auch direkte Anfragen mit einer fremden Rolle werden serverseitig abgelehnt."},
			},
		},
		{
			Version:  "0.89.0",
			Date:     "14. August 2026",
			Kind:     "Portalwechsel",
			Headline: "Eigene Liegenschaften und Perspektiven sind ohne neue Anmeldung erreichbar.",
			Intro:    "Ein kompakter Wechsler in der Navigation verbindet alle freigegebenen Zuhause einer Person und hält jeden Kontext sicher getrennt.",
			Items: []NoteItem{
				{Label: "Ein Klick", Text: "Der Wechsel öffnet direkt den Hausüberblick der gewählten eigenen Liegenschaft."},
				{Label: "Eigene Perspektive", Text: "Eigentümer können ihre eigene Mieteransicht prüfen und danach wieder in die Eigentümeransicht zurückkehren."},
				{Label: "Strikt getrennt", Text: "Tenant und Rolle werden serverseitig validiert; fremde oder höhere Berechtigungen lassen sich nicht auswählen."},
			},
		},
		{
			Version:  "0.88.1",
			Date:     "14. August 2026",
			Kind:     "Stabilität",
			Headline: "Die zentrale Anmeldung führt jedes Zuhause zuverlässig an den richtigen Ort zurück.",
			Intro:    "Ein gemeinsamer Rücksprungpunkt macht neue und bestehende Portalpfade unabhängig von zusätzlichen Freischaltungen beim Anmeldedienst.",
			Items: []NoteItem{
				{Label: "Für alle Zuhause", Text: "Die Anmeldung verwendet eine stabile Plattformadresse und funktioniert damit auch für neu eingerichtete Portalpfade."},
				{Label: "Richtig zurück", Text: "Nach erfolgreicher Anmeldung öffnet sich gezielt das Zuhause, aus dem der Vorgang gestartet wurde."},
			},
		},
		{
			Version:  "0.88.0",
			Date:     "14. August 2026",
			Kind:     "Hilfe",
			Headline: "Die lokale Energieverbindung ist verständlich erklärt und dauerhaft erreichbar.",
			Intro:    "Ein neuer Hilfebereich verbindet Einsteigerführung, technischen Hintergrund, aktuellen Status und sichere Connector-Einrichtung an einem Ort.",
			Items: []NoteItem{
				{Label: "Schritt für Schritt", Text: "Der Weg von der Kopplung über die Messwertauswahl bis zum Live-Energieportal ist klar beschrieben."},
				{Label: "Jederzeit erreichbar", Text: "Berechtigte Personen erzeugen direkt im Portal einen neuen Einmal-Code oder widerrufen die Verbindung."},
				{Label: "Technisch transparent", Text: "Lokale Geheimnisse, übertragene Daten, reine Lesezugriffe und häufige Fehlerbilder sind konkret erklärt."},
			},
		},
		{
			Version:  "0.87.0",
			Date:     "14. August 2026",
			Kind:     "HAUSV Home",
			Headline: "Zwölf kostenlose Monate für neue Zuhause, bestehende Pilot-Zusagen bleiben erhalten.",
			Intro:    "Der kostenlose Nutzungszeitraum ist jetzt als fixes Start- und Enddatum je Zuhause festgeschrieben; eine Zahlung oder Paywall bleibt weiterhin inaktiv.",
			Items: []NoteItem{
				{Label: "Klar geregelt", Text: "Neue HAUSV-Home-Portale nutzen ab abgeschlossener Einrichtung zwölf Monate den vollen Produktumfang kostenlos."},
				{Label: "Fair geschützt", Text: "Bestehende Pilot-Zuhause behalten ihr bereits zugesagtes dreijähriges Enddatum ohne Verkürzung."},
				{Label: "Dauerhaft gebunden", Text: "Löschen, Neueinrichtung, weitere Berechtigte oder ein Eigentümerwechsel starten den Anspruch nicht neu."},
			},
		},
		{
			Version:  "0.86.0",
			Date:     "14. August 2026",
			Kind:     "HAUSV Home",
			Headline: "Ausgewählte Energiewerte fließen sicher vom eigenen Zuhause ins Live-Cockpit.",
			Intro:    "Der lokale Connector ergänzt den bisherigen Verbindungsstatus um bestätigte Messwerte, ohne Home-Assistant-Adresse oder Zugangstoken an HAUSV zu übertragen.",
			Items: []NoteItem{
				{Label: "Gezielt ausgewählt", Text: "Nach dem ersten sicheren Energiekatalog sendet der Connector nur die im Portal bestätigten Sensoren."},
				{Label: "Live sichtbar", Text: "Die ausgewählten Werte stehen im Energie-Onboarding und im persönlichen Live-Cockpit zur Verfügung."},
				{Label: "Klar getrennt", Text: "Messwerte bleiben dem reservierten Zuhause zugeordnet und werden beim Widerruf der Verbindung entfernt."},
				{Label: "Direkt zugänglich", Text: "Bestehende Konten erkennen zusätzlich aktivierte Zuhause-Portale, ohne Rollen aus anderen Häusern zu übernehmen."},
			},
		},
		{
			Version:  "0.85.0",
			Date:     "14. August 2026",
			Kind:     "HAUSV Home",
			Headline: "Die Einrichtung führt verständlich vom bestätigten Zuhause zur optionalen Energieverbindung.",
			Intro:    "Portalaktivierung und Energieverbindung sind zwei klare Schritte; technische Details bleiben bei Bedarf erreichbar, ohne den Hauptweg zu überladen.",
			Items: []NoteItem{
				{Label: "Klar geführt", Text: "Der aktuelle Schritt und die nächste Handlung sind jederzeit direkt sichtbar."},
				{Label: "Technik nach Bedarf", Text: "Home-Assistant-Begriffe und Installationsbefehle stehen in einer getrennten, aufklappbaren Anleitung."},
				{Label: "Durchgehend geprüft", Text: "Ein Browserlauf belegt E-Mail-Bestätigung, Portalaktivierung, lokale Kopplung und dauerhafte Trennung der Häuser."},
			},
		},
		{
			Version:  "0.84.0",
			Date:     "14. August 2026",
			Kind:     "HAUSV Home",
			Headline: "Der reservierte Zuhause-Bereich wird mit einem Klick zum privaten Portal.",
			Intro:    "Nach der E-Mail-Bestätigung aktiviert HAUSV den persönlichen Pfad und meldet die Eigentümerin oder den Eigentümer direkt im Hausüberblick an.",
			Items: []NoteItem{
				{Label: "Direkter Einstieg", Text: "Die Aktivierung führt ohne weiteren Anmeldeschritt in das neue private Portal."},
				{Label: "Dauerhafter Pfad", Text: "Der Zuhause-Bereich bleibt auch nach Neustarts und Aktualisierungen automatisch erreichbar."},
				{Label: "Sicher verbunden", Text: "Portal und hausbezogener Eigentümerzugang werden gemeinsam gespeichert; die Energieverbindung bleibt optional."},
			},
		},
		{
			Version:  "0.83.0",
			Date:     "14. August 2026",
			Kind:     "HAUSV Home",
			Headline: "Home Assistant lässt sich sicher aus dem eigenen Netzwerk koppeln.",
			Intro:    "Ein lokaler Connector verbindet das Zuhause ausgehend mit HAUSV, ohne die Home-Assistant-Adresse oder den Zugangstoken an das Portal zu übertragen.",
			Items: []NoteItem{
				{Label: "Einmal-Code", Text: "Die Kopplung beginnt mit einem zehn Minuten gültigen Code, der nur einmal verwendet werden kann."},
				{Label: "Lokal geschützt", Text: "Home-Assistant-Adresse und Zugangstoken bleiben auf dem eigenen System; HAUSV erhält nur den Connector-Status."},
				{Label: "Unter Kontrolle", Text: "Der Zugang kann erneuert oder sofort widerrufen werden; Messwerte und Gerätesteuerung sind noch nicht freigeschaltet."},
			},
		},
		{
			Version:  "0.82.0",
			Date:     "14. August 2026",
			Kind:     "HAUSV Home",
			Headline: "Der eigene Zuhause-Bereich beginnt mit einer sicheren Reservierung.",
			Intro:    "Eigentümer können ihren gewünschten HAUSV-Home-Pfad vormerken und ihre E-Mail bestätigen, bevor technische Verbindungen eingerichtet werden.",
			Items: []NoteItem{
				{Label: "Eigener Pfad", Text: "Name und persönliche hausv.org-Adresse werden eindeutig reserviert."},
				{Label: "Bestätigung", Text: "Ein 15 Minuten gültiger Einmal-Link bestätigt die angegebene Eigentümer-E-Mail."},
				{Label: "Datensparsam", Text: "Home-Assistant-Adresse und Zugangstoken werden in diesem ersten Schritt bewusst noch nicht abgefragt."},
			},
		},
		{
			Version:  "0.81.2",
			Date:     "14. August 2026",
			Kind:     "Portalauftritt",
			Headline: "Standort und Titelbild folgen wieder der Portalinstanz.",
			Intro:    "Getrennt betriebene Portale können ihre Standortkarte und ihr individuelles Titelbild vollständig in der privaten Instanzkonfiguration verwalten.",
			Items: []NoteItem{
				{Label: "Karte", Text: "Der konfigurierte OpenStreetMap-Ausschnitt erscheint auch auf der öffentlichen Anmeldeseite."},
				{Label: "Titelbild", Text: "Ein privates Instanz-Asset ersetzt zuverlässig das allgemeine Standardbild."},
				{Label: "Trennung", Text: "Konkrete Standort- und Bilddaten bleiben außerhalb des allgemeinen Produktrepositorys."},
			},
		},
		{
			Version:  "0.81.1",
			Date:     "14. August 2026",
			Kind:     "Portalbetrieb",
			Headline: "Portalinstanzen werden eindeutig und sicher ausgeliefert.",
			Intro:    "Der geprüfte Releasepfad kann jetzt mehrere pfadbasierte Portalinstanzen auf demselben Host zuverlässig voneinander unterscheiden.",
			Items: []NoteItem{
				{Label: "Instanzen", Text: "Jede Instanz kann einen eigenen Container, Datenpfad und öffentlichen Portalpfad verwenden."},
				{Label: "Sicherheit", Text: "Gesundheit, Image-Identität und Startprotokoll werden weiterhin am tatsächlich ausgelieferten Container geprüft."},
				{Label: "Kompatibilität", Text: "Bestehende Installationen funktionieren ohne zusätzliche Konfiguration unverändert weiter."},
			},
		},
		{
			Version:  "0.81.0",
			Date:     "14. August 2026",
			Kind:     "Portalbetrieb",
			Headline: "Eine Domain, ein stabiler Pfad je Portal.",
			Intro:    "Hausportale laufen jetzt unter hausv.org mit einem eigenen Kurzcode, ohne zusätzliche DNS-Einträge und ohne Verlust des Portalbezugs bei Links oder Weiterleitungen.",
			Items: []NoteItem{
				{Label: "Adresse", Text: "Anmeldung, Formulare, Downloads, Kalender und Benachrichtigungen bleiben zuverlässig im Pfad des gewählten Portals."},
				{Label: "Betrieb", Text: "Konkrete Instanz- und Infrastrukturkonfiguration kann getrennt vom allgemeinen Produktcode verwaltet werden."},
				{Label: "HAUSV Free", Text: "Die öffentliche Quellcode-Veröffentlichung ist klar für Version 1.0 angekündigt; bis dahin führt kein öffentlicher Link in ein privates Repository."},
			},
		},
		{
			Version:  "0.80.0",
			Date:     "14. August 2026",
			Kind:     "Produkte & Navigation",
			Headline: "HAUSV Free macht den freien Einstieg sichtbar.",
			Intro:    "Die Produktfamilie reicht jetzt von der dauerhaft kostenlosen, selbst betriebenen Open-Source-Lösung bis zu den betreuten Home- und Professional-Angeboten.",
			Items: []NoteItem{
				{Label: "HAUSV Free", Text: "Der vollständige AGPL-3.0-Kern bleibt kostenlos und darf frei selbst betrieben werden; Support erfolgt über GitHub-Tickets und Pull Requests."},
				{Label: "Support", Text: "HAUSV Home umfasst E-Mail-Support, HAUSV Professional Telefon- und E-Mail-Support."},
				{Label: "Mobiles Menü", Text: "Ein korrektes Lucide-SVG sorgt im schmalen Layout für ein ruhiges, gleichmäßiges Menüsymbol."},
			},
		},
		{
			Version:  "0.79.0",
			Date:     "14. August 2026",
			Kind:     "Produkte & Startseite",
			Headline: "HAUSV Home und Professional zeigen ihren Nutzen auf den ersten Blick.",
			Intro:    "Zehn visuell erklärte Funktionen, zwei klar getrennte Produkte und eine einfache Preislogik machen den Einstieg für Selbstverwaltung und Hausverwaltungen leichter.",
			Items: []NoteItem{
				{Label: "Top-Funktionen", Text: "Hausüberblick, Kommunikation, Termine, Anliegen, Dokumente, Übergaben, Abstimmungen, Rechte, Energie und Integrationen erhalten jeweils eine eigene anschauliche Einführung."},
				{Label: "HAUSV Home", Text: "Der gehostete Weg für die Selbstverwaltung startet zwölf Monate kostenlos und kostet danach 12 Euro pro Jahr."},
				{Label: "HAUSV Professional", Text: "Hausverwaltungen können hosted oder selbst betrieben starten; 25 Einheiten sind kostenlos, darüber gelten 3 Euro je Einheit und Monat."},
				{Label: "Sicherer Build", Text: "Die Anwendung wird mit Go 1.26.6 und den aktuellen Sicherheitskorrekturen gebaut."},
			},
		},
		{
			Version:  "0.78.0",
			Date:     "13. August 2026",
			Kind:     "Energie & Zuhause",
			Headline: "Mehrere Zuhause bleiben auch im selben Portal sauber getrennt.",
			Intro:    "Energieprofile, Messwerte und Home-Assistant-Verbindungen erhalten einen stabilen Zuhause-Bezug, während bestehende Einzel-Zuhause ohne zusätzliche Konfiguration weiterlaufen.",
			Items: []NoteItem{
				{Label: "Sichere Trennung", Text: "Anlagen, Zuordnungen, Wartungen, Maßnahmen und Messverläufe werden je Zuhause gespeichert und abgefragt."},
				{Label: "Bestehende Daten", Text: "Der bisherige Datenbestand wird verlustfrei dem Standard-Zuhause zugeordnet."},
				{Label: "Home Assistant", Text: "Mehrere lesende Verbindungen je Portal können jeweils einem eigenen Zuhause folgen."},
			},
		},
		{
			Version:  "0.77.0",
			Date:     "13. August 2026",
			Kind:     "Produkt & Positionierung",
			Headline: "Gemeinschaft und Zuhause werden als zwei klare Wege sichtbar.",
			Intro:    "Die öffentliche Seite verbindet beide Produktlinien über einen gemeinsamen Vertrauenskern und hält Preise, Pilotstatus und noch geschlossene Funktionen sauber auseinander.",
			Items: []NoteItem{
				{Label: "HAUSV Gemeinschaft", Text: "Kommunikation, Dokumente, Anliegen und Entscheidungen für Mehrparteienhäuser stehen als eigener Produktweg im Mittelpunkt."},
				{Label: "HAUSV Zuhause", Text: "Hauszustand, Wartung, Energiebeobachtung und ein verständlicher Fahrplan werden ohne Technikjargon erklärt."},
				{Label: "Ehrliche Grenzen", Text: "Verrechnung, allgemeine aktive Steuerung, offene Dienstleister-Zugänge und Marktplatz bleiben ausdrücklich außerhalb des aktuellen Angebots."},
			},
		},
		{
			Version:  "0.76.3",
			Date:     "13. August 2026",
			Kind:     "Energie & Navigation",
			Headline: "Ladestände werden vollständiger sichtbar.",
			Intro:    "Speicher und entsprechend konfigurierte Verbraucher verbinden Prozentwert und Energiemenge in einer Unterzeile; die Seitenleiste bleibt zugleich bis zum Seitenende vollständig stehen.",
			Items: []NoteItem{
				{Label: "Navigation", Text: "Auch auf großen Bildschirmen schließt die Seitenleiste der langen Energieübersicht bündig mit dem Browserfenster ab."},
			},
		},
		{
			Version:  "0.76.2",
			Date:     "12. August 2026",
			Kind:     "Seitenleiste",
			Headline: "Der Kontoabschluss bleibt vollständig sichtbar.",
			Intro:    "Version und Kartenhinweis haben jetzt jeweils einen verlässlichen Platz, ohne Namen oder Quellenangabe abzuschneiden.",
			Items:    []NoteItem{},
		},
		{
			Version:  "0.76.1",
			Date:     "12. August 2026",
			Kind:     "Energie & Seitenleiste",
			Headline: "Speicher-Messwerte und Kontoangaben bleiben zuverlässig zugeordnet.",
			Intro:    "Ältere Lade- und Entladesensoren landen beim Gestalten weiterhin in den richtigen Feldern; Name, Rolle und Kontoaktionen bleiben vollständig lesbar.",
			Items:    []NoteItem{},
		},
		{
			Version:  "0.76.0",
			Date:     "12. August 2026",
			Kind:     "Energie-Cockpit",
			Headline: "Energiefluss und Seitenleiste sind ruhiger und frei konfigurierbar.",
			Intro:    "Gleich große Kacheln zeigen frei wählbare Farben und zweite Home-Assistant-Kennzahlen; Empfehlung, Version und Kontoaktionen sitzen kompakt am passenden Ort.",
			Items:    []NoteItem{},
		},
		{
			Version:  "0.75.3",
			Date:     "12. August 2026",
			Kind:     "Energie-Cockpit",
			Headline: "Live-Werte aktualisieren sich automatisch.",
			Intro:    "Der Energiefluss lädt alle zehn Sekunden neue Home-Assistant-Werte, zeigt das Aktualisierungsalter und macht eine unterbrochene Verbindung sichtbar.",
			Items:    []NoteItem{},
		},
		{
			Version:  "0.75.2",
			Date:     "12. August 2026",
			Kind:     "Energie-Cockpit",
			Headline: "E-Autos zeigen die aktuelle Ladeleistung zuverlässig.",
			Intro:    "Eindeutig passende Zuhause-Messwerte aus Home Assistant werden bei benannten Fahrzeugen dauerhaft zugeordnet.",
			Items:    []NoteItem{},
		},
		{
			Version:  "0.75.1",
			Date:     "12. August 2026",
			Kind:     "Energie-Cockpit",
			Headline: "Ruhige Energieflüsse und Empfehlungen ohne Seitensprung.",
			Intro:    "Bewegte Punkte zeigen die Flussrichtung; Empfehlungen und Simulationen öffnen sich kompakt in einem eigenen Dialog.",
			Items:    []NoteItem{},
		},
		{
			Version:  "0.75.0",
			Date:     "12. August 2026",
			Kind:     "Energie-Cockpit",
			Headline: "Energieflüsse sind ruhiger und besser lesbar.",
			Intro:    "Leichte Leistungsbahnen erhalten Breite, Farbe und Richtung; Wert und Einheit bleiben in allen Anzeigen des Energie-Cockpits zuverlässig zusammen.",
			Items: []NoteItem{
				{Label: "Elegante Flüsse", Text: "Transparente Bahnen, eine feine Farblinie und kompakte Richtungspfeile ersetzen die dominanten Blockpfeile."},
				{Label: "Geschützte Einheiten", Text: "Zwischen Zahlen und Einheiten steht nun immer ein geschützter Abstand, damit Angaben nicht auseinanderbrechen."},
			},
		},
		{
			Version:  "0.74.0",
			Date:     "12. August 2026",
			Kind:     "Energie-Cockpit",
			Headline: "Alle Energiefluss-Knoten lassen sich direkt konfigurieren.",
			Intro:    "Haus, Netz, PV, Speicher und Verbraucher verwenden einen gemeinsamen Dialog; der Speicher kann seine Lade- und Entladeleistung aus getrennten Home-Assistant-Sensoren beziehen.",
			Items: []NoteItem{
				{Label: "Einheitlicher Dialog", Text: "Name, Lucide-Symbol und relevante Messwerte werden direkt am jeweiligen Knoten gepflegt."},
				{Label: "Speicherleistung", Text: "Nettoleistung sowie getrennte Lade- und Entladeleistung können passend zur vorhandenen Installation gewählt werden."},
				{Label: "Parkplatz 20", Text: "Der Ladepunkt ist jetzt wie andere Verbraucher bearbeitbar, priorisierbar und entfernbar."},
			},
		},
		{
			Version:  "0.73.0",
			Date:     "12. August 2026",
			Kind:     "Energie-Cockpit",
			Headline: "Verbraucher erhalten eigene Messwerte und die ganze Lucide-Auswahl.",
			Intro:    "Aktuelle Leistung und Energiezähler lassen sich direkt aus Home Assistant zuordnen; jedes lokal ausgelieferte Lucide-Symbol ist auffindbar und Verbraucher können bewusst entfernt werden.",
			Items: []NoteItem{
				{Label: "Eigene Messwerte", Text: "Leistung und Energiezähler bleiben pro Verbraucher nachvollziehbar vom gesamten Hausverbrauch getrennt."},
				{Label: "Alle Lucide-Symbole", Text: "Die schnellen Vorschläge bleiben erhalten; die Suche umfasst zusätzlich den vollständigen lokalen Katalog."},
				{Label: "Sicher entfernen", Text: "Jeder Verbraucher lässt sich nach einer zweiten Bestätigung löschen, ohne die Home-Assistant-Entity zu verändern."},
			},
		},
		{
			Version:  "0.72.0",
			Date:     "12. August 2026",
			Kind:     "Energie-Cockpit",
			Headline: "Verbraucher lassen sich direkt und ohne Springen bearbeiten.",
			Intro:    "Ein Klick auf eine Verbraucher-Kachel öffnet den neuen Dialog für Name, Priorität, Symbol und technische Angaben; Hinzufügen funktioniert am selben Ort.",
			Items: []NoteItem{
				{Label: "Ruhige Kacheln", Text: "Beim Überfahren wechseln nur Symbol und Hinweistext, ohne den Energiefluss zu verschieben."},
				{Label: "Lucide-Symbole", Text: "Die durchsuchbare Auswahl verwendet die etablierte, lokal ausgelieferte SVG-Bibliothek."},
				{Label: "Ein klarer Ort", Text: "Verbraucher werden im Energiefluss gepflegt; der Anlagenbereich bleibt Erzeugung, Speicher und Service vorbehalten."},
			},
		},
		{
			Version:  "0.71.1",
			Date:     "11. August 2026",
			Kind:     "Energie-Cockpit",
			Headline: "Symbole sitzen exakt in ihren Kreisen.",
			Intro:    "Die gezeichneten Linien jedes Symbols sind jetzt vermessen und mittig gesetzt; das Plus der Hinzufügen-Kachel ist ein echtes Vektorsymbol.",
			Items:    []NoteItem{},
		},
		{
			Version:  "0.71.0",
			Date:     "11. August 2026",
			Kind:     "Energie-Cockpit",
			Headline: "Der Energiefluss rechnet vor.",
			Intro:    "Jedes Element zeigt beim Überfahren seine Teilleistungen, die Flusslinien sind breiter und tragen nur noch Zielfarben, und Verbraucher lassen sich bearbeiten, entfernen und flüssiger verschieben.",
			Items: []NoteItem{
				{Label: "Schlüssige Bilanz", Text: "Der Hausverbrauch rechnet sich sichtbar aus PV, Speicher und Netzbezug zusammen; ein Rest wird als solcher ausgewiesen."},
				{Label: "Bearbeiten & Entfernen", Text: "Jede Verbraucher-Kachel bietet die Aktionen beim Überfahren an; eigene Geräte lassen sich direkt entfernen."},
				{Label: "Flüssiges Verschieben", Text: "Beim Ziehen macht die Zielposition Platz und die Liste sortiert live mit."},
			},
		},
		{
			Version:  "0.70.0",
			Date:     "11. August 2026",
			Kind:     "Energie-Cockpit",
			Headline: "Verbraucher lassen sich jetzt wirklich umsortieren.",
			Intro:    "Die Prioritätenliste im Energiefluss ist bedienbar: Kacheln lassen sich ziehen oder per Pfeiltasten verschieben, die Reihenfolge bleibt gespeichert.",
			Items: []NoteItem{
				{Label: "Ziehen oder Tasten", Text: "Der Griff rechts an jeder Kachel nimmt Maus wie Tastatur an; die Nummern folgen sofort."},
				{Label: "Nachvollziehbar", Text: "Jede Änderung der Reihenfolge landet im Verlauf des Hauses."},
			},
		},
		{
			Version:  "0.69.0",
			Date:     "11. August 2026",
			Kind:     "Energie-Cockpit",
			Headline: "Der Energiefluss zeigt jetzt, woher der Strom kommt und wohin er geht.",
			Intro:    "Das Cockpit ordnet sich neu: PV oben, das Zuhause in der Mitte, Netz unten, Speicher links – und rechts die Verbraucher nach Priorität. Die Flusslinien tragen Richtungspfeile und Farbverläufe, die die Zielanteile zeigen.",
			Items: []NoteItem{
				{Label: "Lebendiger Energiefluss", Text: "Die Linienbreite entspricht der Leistung, weiche Farbbänder zeigen anteilig, wie viel in Haus, Speicher, Netz und Verbraucher fließt."},
				{Label: "Verbraucher im Blick", Text: "E-Auto, Boiler, Wärmepumpe und weitere Geräte stehen als Prioritätenliste rechts im Fluss – mit Messwert-Zeile, sobald ein Wert vorliegt."},
				{Label: "Ruhiger Kopf", Text: "Die Monatsspitze wandert als kompakte Zeile in die Sicherheitsleiste; der Seitenkopf braucht nur noch eine Zeile."},
				{Label: "Eine Breite, eine Schrift", Text: "Alle Karten teilen dieselbe Spaltenbreite und dieselbe Titel-Typografie; Leistungswerte stehen durchgängig in kW."},
			},
		},
		{
			Version:  "0.68.0",
			Date:     "11. August 2026",
			Kind:     "Auftritt & Klarheit",
			Headline: "Die öffentliche Seite spricht Hausverwaltungen direkt an.",
			Intro:    "Preise erscheinen jetzt als klare Servicepauschale mit inkludierten Einheiten, die rechtlichen Angaben haben eine eigene, ruhige Seite, und das Logo verhält sich einfach und vorhersehbar.",
			Items: []NoteItem{
				{Label: "Servicepauschale", Text: "Bis 25 Wohneinheiten inkludiert, darüber 1 € je Einheit und Monat – die Höhe der Pauschale klärt das persönliche Gespräch."},
				{Label: "Impressum & Infos", Text: "Anschrift, rechtliche Details und der Hinweis auf professionelle Services haben eine eigene Unterseite."},
				{Label: "Offener Kern", Text: "Die AGPL-3.0-Lizenz des quelloffenen Kerns ist direkt von der Startseite verlinkt."},
				{Label: "Ruhiges Logo", Text: "Das drehende Zeichen steht fest über der Überschrift; beim Scrollen erscheint schlicht die kleine Wortmarke oben links."},
			},
		},
		{
			Version:  "0.67.0",
			Date:     "11. August 2026",
			Kind:     "Produkt & Preise",
			Headline: "Ein klarer Weg für Hausverwaltungen.",
			Intro:    "Die öffentliche Seite zeigt jetzt, wie es weitergeht: Der Kern von hausv.org ist Open Source und bleibt kostenlos, der betreute Betrieb für Hausverwaltungen bekommt eine klare Preisbasis.",
			Items: []NoteItem{
				{Label: "Open-Source-Kern", Text: "Der Kern der Lösung ist quelloffen und bleibt für Hausgemeinschaften dauerhaft frei nutzbar."},
				{Label: "Service für Hausverwaltungen", Text: "Betreuter Betrieb über HAUSV Professional: 500 € je Monat für kleinere, 900 € je Monat für größere Hausverwaltungen, zuzüglich 1 € je Wohneinheit und Monat."},
				{Label: "Eine Adresse", Text: "www.hausv.org führt jetzt direkt auf hausv.org – eine Adresse, ein Lesezeichen, keine Duplikate."},
			},
		},
		{
			Version:  "0.66.1",
			Date:     "2. August 2026",
			Kind:     "Energie-Cockpit",
			Headline: "Das Energie-Cockpit sitzt jetzt bis ins Detail.",
			Intro:    "Präzisere Proportionen, vertraute Symbole und belastbare Vergleiche machen die Energieansicht auf jedem Bildschirm noch ruhiger.",
			Items: []NoteItem{
				{Label: "Feinschliff", Text: "Abstände, Größen und gerade Flusslinien bilden vom Mobiltelefon bis zum großen Desktop eine klare Einheit."},
				{Label: "Ehrlicher Vergleich", Text: "Der Balken der Monatsspitze entspricht jetzt exakt ihrem Verhältnis zur verrechneten Leistung."},
				{Label: "Verlässliche Bilder", Text: "Eigene und mitgelieferte Hausbilder laden in allen angemeldeten Ansichten korrekt."},
			},
		},
		{
			Version:  "0.66.0",
			Date:     "2. August 2026",
			Kind:     "Energie-Cockpit",
			Headline: "Energie wird auf einen Blick verständlich.",
			Intro:    "Das Energie-Cockpit zeigt den aktuellen Fluss, die Monatsspitze und den nächsten Schritt in einer ruhigen, klaren Ansicht.",
			Items: []NoteItem{
				{Label: "Energiefluss", Text: "Erzeugung, Hausverbrauch, Netz und Speicher sind mit geraden, eindeutigen Flussrichtungen verbunden."},
				{Label: "Wissen bei Bedarf", Text: "Kennzahlen, Herleitung, Tarifdetails und Quellen bleiben über dezente Info-Schaltflächen erreichbar."},
				{Label: "Monatsspitze", Text: "Die wichtigsten Werte stehen zuerst; alles Weitere öffnet sich kompakt über „Mehr erfahren“."},
				{Label: "Ehrlicher Fortschritt", Text: "Das Cockpit zählt tatsächlich erfasste Viertelstunden und zeigt, wie lange es bis zur ersten Empfehlung noch dauert."},
				{Label: "Überall sauber", Text: "Die Ansicht bleibt von 320 bis 1440 Pixel klar; Seitenleiste und Energieverlauf bleiben unverändert vertraut."},
			},
		},
		{
			Version:  "0.65.0",
			Date:     "1. August 2026",
			Kind:     "Bedienung & Orientierung",
			Headline: "Der Weg durchs Portal fühlt sich jetzt deutlich einfacher an.",
			Intro:    "HAUSV führt vom Anmelden bis zum erledigten Anliegen, Dokument oder Übergabe mit weniger Ablenkung und klaren nächsten Schritten.",
			Items: []NoteItem{
				{Label: "Schneller ans Ziel", Text: "Hauptaktionen stehen im Vordergrund; seltene Funktionen und Erklärungen bleiben ruhig im Hintergrund."},
				{Label: "Anmelden mit Orientierung", Text: "Unter der Anmeldung zeigt eine echte OpenStreetMap-Karte die vertraute Straße und Umgebung."},
				{Label: "Mobil vollständig", Text: "Navigation, Dialoge, Energie, Zahlungsimport und Parkplatz bleiben bis 320 Pixel Breite lesbar und bedienbar."},
				{Label: "Sauber abgeschlossen", Text: "Anliegen, Übergaben, Dokumente und E-Rechnungen führen verständlich bis zum sichtbaren Ergebnis."},
				{Label: "Verlässlich bedienen", Text: "Tastaturfokus, Menüs, Dialogaktionen und große Tippflächen funktionieren über alle Hauptwege einheitlich."},
			},
		},
		{
			Version:  "0.64.0",
			Date:     "1. August 2026",
			Kind:     "Energie",
			Headline: "Ihre Monatsspitze entsteht jetzt von selbst.",
			Intro:    "Wer Home Assistant verbunden hat, musste bisher zusätzlich einen Smart-Meter-Export hochladen, damit im Energie-Cockpit überhaupt eine Viertelstunde erschien. HAUSV zeichnet die abgeschlossenen Viertelstunden ab sofort laufend selbst auf.",
			Items: []NoteItem{
				{Label: "Laufend aufgezeichnet", Text: "Der Netzbezug wird alle 30 Sekunden gelesen; jede vollständige Viertelstunde wird festgehalten, sobald sie vorbei ist."},
				{Label: "Nur gelesen", Text: "Auch im Beobachtungsmodus: HAUSV liest Home Assistant und schaltet nichts."},
				{Label: "Ehrlich beschriftet", Text: "Aus laufenden Messwerten verdichtete Viertelstunden heißen „teilweise geschätzt“, und eine Messlücke bleibt eine Messlücke statt eines erfundenen Werts."},
				{Label: "Zwei Quellen im Vergleich", Text: "Liegt zusätzlich ein Smart-Meter-Export vor, stellt das Cockpit beide Monatsspitzen gegenüber."},
			},
		},
		{
			Version:  "0.63.0",
			Date:     "1. August 2026",
			Kind:     "Oberfläche",
			Headline: "Jede Fläche zeigt jetzt den Stand — nicht nur eine Überschrift.",
			Intro:    "Der Hausüberblick, das Energie-Cockpit und alle Listenseiten wurden neu aufgebaut: dichter, ruhiger und auf beiden Bildschirmgrößen ohne Brüche.",
			Items: []NoteItem{
				{Label: "Hausüberblick", Text: "Was heute ansteht, die nächsten Termine, der Aushang und offene Anliegen auf einen Blick — ohne Wiederholungen."},
				{Label: "Monatsspitze zuerst", Text: "Das Energie-Cockpit beginnt mit der verrechneten Leistung und nennt, auf wie vielen Viertelstunden sie beruht."},
				{Label: "Ehrlich beziffert", Text: "Der Betrag im Cockpit heißt Leistungsanteil des Netztarifs und nennt ausdrücklich, was er nicht enthält."},
				{Label: "Nichts läuft mehr über", Text: "Alle Bereiche wurden über zehn Bildschirmbreiten von 360 bis 1440 Pixeln geprüft."},
			},
		},
		{
			Version:  "0.62.0",
			Date:     "1. August 2026",
			Kind:     "Energie & Transparenz",
			Headline: "Jeder Verbraucher zählt — und das Modell verspricht nur, was Sie erklärt haben.",
			Intro:    "Was Strom braucht, lässt sich jetzt einzeln erfassen: mit eigenem Namen, eigener Leistung und eigener Flexibilität. Auch mehrere Geräte derselben Art.",
			Items: []NoteItem{
				{Label: "Eigene Verbraucher anlegen", Text: "Sauna, Werkstatt oder Infrarotkabine sind keine Sonderfälle mehr — sie zählen im Energie-Cockpit wie jedes andere Gerät."},
				{Label: "Ehrlich rechnen", Text: "Ein ausdrücklich als fest erklärtes Gerät zählt nicht mehr als Flexibilität, und eine PV-Anlage nicht mehr als abschaltbare Last."},
				{Label: "Angaben bleiben erhalten", Text: "Erfasste Nennleistungen überstehen das erneute Speichern der Geräteauswahl im Onboarding."},
			},
		},
		{
			Version:  "0.61.0",
			Date:     "31. Juli 2026",
			Kind:     "Energie & Transparenz",
			Headline: "Das Energie-Cockpit zeigt, welche Leistung nach dem Tarifentwurf 2027 verrechnet würde.",
			Intro:    "Neben der höchsten Viertelstunde des Monats steht jetzt die tatsächlich bemessene Leistung — mit Begründung, wenn beide auseinanderfallen.",
			Items: []NoteItem{
				{Label: "Anschlussleistung erfassen", Text: "Die auf der Netzrechnung vereinbarte Leistung bestimmt die Mindestbemessung und lässt sich im Energie-Cockpit hinterlegen."},
				{Label: "Wirksam kappen", Text: "Der Anteil oberhalb der Staffelschwelle wird getrennt ausgewiesen, weil er im Entwurf doppelt so hoch bemessen wird."},
				{Label: "Ehrlich rechnen", Text: "Unterhalb der Mindestbemessung weist das Portal aus, dass der verrechnete Betrag nicht weiter sinkt."},
			},
		},
		{
			Version:  "0.60.2",
			Date:     "31. Juli 2026",
			Kind:     "Bedienbarkeit",
			Headline: "Schmale Bildschirme haben wieder ein vollständiges Menü.",
			Intro:    "Eine Schaltfläche in der Kopfzeile öffnet die Navigation; nach der Auswahl schließt sie sich von selbst.",
			Items: []NoteItem{
				{Label: "Überall navigieren", Text: "Funktionen, Sicherheit, Preise, Impressum und Kontakt sind auch auf dem Telefon erreichbar."},
				{Label: "Klar lesen", Text: "Marke und Menü stehen auf einem eigenen Streifen und bleiben sichtbar, wenn heller Inhalt darunter scrollt."},
			},
		},
		{
			Version:  "0.60.1",
			Date:     "31. Juli 2026",
			Kind:     "Feinschliff",
			Headline: "Das Markenzeichen sitzt auf jeder Bildschirmgröße sauber zur Überschrift.",
			Intro:    "Größe und Position richten sich automatisch nach dem freien Raum über dem Text.",
			Items: []NoteItem{
				{Label: "Ruhig ausrichten", Text: "Das Markenzeichen steht bündig zur Überschrift und wandert beim Verändern der Fenstergröße nicht mehr."},
				{Label: "Einfach scrollen", Text: "Der gefaltete Seitenübergang entfällt; die Seite scrollt wieder ohne Sonderfreigabe des Browsers."},
			},
		},
		{
			Version:  "0.60.0",
			Date:     "31. Juli 2026",
			Kind:     "Auftritt & Orientierung",
			Headline: "Die Startseite zeigt ein dreidimensionales Markenzeichen und behält die Kopfzeile im Blick.",
			Intro:    "Das Markenzeichen empfängt Besucher groß über der Überschrift und wandert beim Scrollen ruhig in die feste Kopfzeile.",
			Items: []NoteItem{
				{Label: "Marke erleben", Text: "Das Markenzeichen steht als Glasobjekt über der Überschrift und lässt sich mit der Maus drehen."},
				{Label: "Überall navigieren", Text: "Funktionen, Sicherheit, Preise, Impressum und Kontakt bleiben beim Scrollen erreichbar; die Marke führt zurück an den Seitenanfang."},
				{Label: "Angebot benennen", Text: "Überschrift und Einleitung nennen transparentes Energiemanagement mit automatisierbarem Peak-Shaving für Eigentümer, Mieter, Beiräte und Hausverwaltungen."},
			},
		},
		{
			Version:  "0.59.2",
			Date:     "30. Juli 2026",
			Kind:     "Klarheit & Stabilität",
			Headline: "Exporte und Hausnamen bleiben klar und unabhängig von alten Portalbezeichnungen.",
			Intro:    "Parkplatzabrechnungen benennen sich neutral; ohne eigenen Hausnamen führt stattdessen die vertraute Adresse.",
			Items: []NoteItem{
				{Label: "Dokumente erkennen", Text: "Jahres- und Monatsabrechnungen tragen einen kurzen, eindeutigen Titel ohne überholten Produktnamen."},
				{Label: "Häuser zuordnen", Text: "Ist kein Anzeigename eingerichtet, verwendet das Portal die Adresse als verständliche Bezeichnung."},
				{Label: "Geprüft freigeben", Text: "Der Sicherheitsprüfer läuft in einer bewusst festgelegten Version; erfolgreiche Produktionsstände bleiben eindeutig markiert."},
			},
		},
		{
			Version:  "0.59.1",
			Date:     "30. Juli 2026",
			Kind:     "Stabilität & Datenschutz",
			Headline: "Anmeldemails bleiben auch bei einem trägen Mailserver verlässlich begrenzt.",
			Intro:    "Feste Zeitgrenzen schützen Anmeldung und Neustart; nicht zugestellte Einmal-Links werden sicher verworfen.",
			Items: []NoteItem{
				{Label: "Begrenzt warten", Text: "Verbindungsaufbau und E-Mail-Übertragung besitzen klare Obergrenzen, damit ein Mailserver das Portal nicht festhält."},
				{Label: "Links entwerten", Text: "Fehlgeschlagene oder beim Herunterfahren verworfene Anmeldelinks bleiben nicht weiter verwendbar."},
				{Label: "Privat protokollieren", Text: "Betriebsprotokolle nennen bei Mailfehlern weder Empfänger noch Link, Token oder Antworttext des Mailservers."},
				{Label: "Geordnet neu starten", Text: "Das Portal beendet angenommene Anmeldemails innerhalb eines festen Zeitfensters und bleibt danach sicher startbereit."},
			},
		},
		{
			Version:  "0.59.0",
			Date:     "30. Juli 2026",
			Kind:     "Sicherheit & Stabilität",
			Headline: "Anmeldung und Produktionsfreigabe haben jetzt einen zusätzlichen Sicherheitsgurt.",
			Intro:    "Geschützte Ansichten bleiben nach dem Abmelden geschlossen, und nur ein vollständig geprüfter Programmstand darf live gehen.",
			Items: []NoteItem{
				{Label: "Ruhig anmelden", Text: "Anmeldeanfragen antworten einheitlich und warten nicht auf den E-Mail-Versand; begrenzte Wiederholungen schützen Zugang und Konten."},
				{Label: "Sicher abmelden", Text: "Geschützte Seiten und Dateien werden nicht aus einem alten Browser-Zwischenspeicher wieder sichtbar."},
				{Label: "Rollen prüfen", Text: "Bewohner- und Administrationswege laufen vor jeder Freigabe automatisch auf Desktop und Mobil durch."},
				{Label: "Geprüft freigeben", Text: "Produktion akzeptiert nur genau den grünen Code-Stand und kontrolliert danach Zustand, sichtbare Version und Startprotokoll."},
				{Label: "Daten bewahren", Text: "Vor Datenbankänderungen entsteht ein konsistenter Wiederherstellungspunkt; Rückkehrwege unterscheiden Programm und Daten bewusst."},
			},
		},
		{
			Version:  "0.58.0",
			Date:     "29. Juli 2026",
			Kind:     "Mein Zuhause",
			Headline: "Ihre Energiedaten gehören sichtbar Ihnen.",
			Intro:    "Eigentümer und Hausadministration können den gespeicherten Umfang verstehen, vollständig mitnehmen und bewusst löschen.",
			Items: []NoteItem{
				{Label: "Alles mitnehmen", Text: "Profil, Anlagen, Zuordnungen, Messwerte, Auswertungen und Energieprotokoll stehen als maschinenlesbares ZIP bereit."},
				{Label: "Gezielt löschen", Text: "Messverlauf und vollständiges Energieprofil haben getrennte, klar bestätigte Löschwege; technische Freigaben enden mit dem Profil sofort."},
				{Label: "Fristen verstehen", Text: "Die Energiedaten-Zentrale erklärt, welche Daten kurz, länger oder als notwendiger Vertragsmarker aufbewahrt werden."},
				{Label: "Privat einordnen", Text: "Home-Assistant-Zugangsdaten bleiben in der geschützten Host-Konfiguration; gelesene Live-Verläufe werden nicht dauerhaft als Portalhistorie gespeichert."},
				{Label: "Fair bleiben", Text: "Die Hausgemeinschaft bleibt im Pilot bis 25 Wohneinheiten kostenfrei; private Hausprofile behalten ihren eigenen dreijährigen Gratiszeitraum."},
			},
		},
		{
			Version:  "0.57.1",
			Date:     "29. Juli 2026",
			Kind:     "Mein Zuhause",
			Headline: "Der sichere Energie-Testlauf sagt jetzt genau, was er tut.",
			Intro:    "HAUSV protokolliert dabei nur mögliche Entscheidungen und schaltet ausdrücklich kein Gerät.",
			Items: []NoteItem{
				{Label: "Bewusst starten", Text: "Eigentümer oder Hausadministration bestätigen ausdrücklich einen wirkungslosen Testlauf – keine allgemeine Steuerungsfreigabe."},
				{Label: "Ohne Gerätewirkung", Text: "Der sichtbare Status erklärt dauerhaft, dass mögliche Regelentscheidungen nur aufgezeichnet werden."},
				{Label: "Aktiv klar trennen", Text: "Eine spätere aktive Steuerung benötigt eine eigene Freigabe für konkrete Geräte und kann durch den Testlauf nicht vorweggenommen werden."},
				{Label: "Sicher zurückkehren", Text: "Die unmittelbare Rückkehr zu „Nur beobachten“ bleibt jederzeit verfügbar."},
			},
		},
		{
			Version:  "0.57.0",
			Date:     "29. Juli 2026",
			Kind:     "Mein Zuhause",
			Headline: "Der persönliche Anzeigename führt jetzt einheitlich durch das Portal.",
			Intro:    "Der gewählte Name steht im Vordergrund; die offizielle Wohneinheit bleibt direkt darunter ruhig und eindeutig sichtbar.",
			Items: []NoteItem{
				{Label: "Schneller erkennen", Text: "Seitenleiste, Mobilmenü, Energieüberblick und Einstellungen nennen dasselbe Zuhause gleich."},
				{Label: "Sauber unterscheiden", Text: "„QA Zuhause“ kann persönlich heißen, während „Einheit 12“ die unveränderte offizielle Stammdatenbezeichnung bleibt."},
				{Label: "Überall mitnehmen", Text: "Der Anzeigename bleibt auch während eines noch laufenden Onboardings auf anderen Portalseiten erhalten."},
				{Label: "Mobil lesbar bleiben", Text: "Auch lange Namen ordnen sich ohne horizontales Überlaufen in die klare zweistufige Hierarchie ein."},
			},
		},
		{
			Version:  "0.56.0",
			Date:     "29. Juli 2026",
			Kind:     "Mein Zuhause",
			Headline: "Hausprofil und offizielle Wohnung sind jetzt klar verbunden und leicht bearbeitbar.",
			Intro:    "Der freundliche Cockpitname bleibt von den Stammdaten getrennt und gehört zugleich eindeutig zur richtigen Wohnung.",
			Items: []NoteItem{
				{Label: "Namen einordnen", Text: "Ein Anzeigename wie „Dachwohnung“ steht sichtbar neben seiner offiziellen Wohnung, zum Beispiel „Einheit 12“."},
				{Label: "Direkt bearbeiten", Text: "Energieüberblick, Einstellungen und Einheitenverwaltung führen ohne erneutes Onboarding zu Name, Art und Wohnungszuordnung."},
				{Label: "Stammdaten bewahren", Text: "Eine Änderung am Hausprofil lässt Adresse, Gebäudename und offizielle Einheitsbezeichnungen unverändert."},
				{Label: "Historie schützen", Text: "Mehrdeutige Altbestände bleiben ungeöffnet, bis die Hausverwaltung die richtige Wohnung bewusst zuordnet."},
				{Label: "Verantwortung begrenzen", Text: "Zugriff folgt der Wohnung; delegierte technische Betreuung bleibt auf Messwerte und Einrichtung beschränkt."},
			},
		},
		{
			Version:  "0.55.0",
			Date:     "29. Juli 2026",
			Kind:     "Mein Zuhause",
			Headline: "Energiefluss und Tagesverlauf sind jetzt lebendiger, klarer und größer.",
			Intro:    "Eigene Zeichen erklären die aktuellen Flüsse; der Speicher zeigt Füllstand und Richtung, während das Diagramm zwischen rollenden 24 Stunden und dem vollständigen heutigen Tag wechselt.",
			Items: []NoteItem{
				{Label: "Flüsse erkennen", Text: "Hausverbrauch, PV, Netzbezug und Einspeisung erhalten eine konsistente, unabhängig von Home Assistant gelieferte SVG-Bildsprache."},
				{Label: "Speicher verstehen", Text: "Die Batterie lässt freie Kapazität proportional sichtbar und zeigt Laden oder Entladen mit einer ruhigen, abschaltbaren Bewegung."},
				{Label: "Heute überblicken", Text: "Die feste Achse von 00 bis 24 Uhr zeigt nur bereits gemessene Werte; zukünftige Stunden bleiben ehrlich leer."},
				{Label: "Bildschirm ausnutzen", Text: "Der Energieverlauf öffnet sich wahlweise vergrößert oder im echten Vollbild und bleibt per Maus, Touch und Tastatur erkundbar."},
			},
		},
		{
			Version:  "0.54.1",
			Date:     "29. Juli 2026",
			Kind:     "Hausportal",
			Headline: "Der Ortskopf ist jetzt präziser, ruhiger und vollständig in die Seitenleiste integriert.",
			Intro:    "Die Karte nutzt die ganze Breite, läuft weich in die Navigation aus und markiert das Haus mit der Spitze eines kompakten Pins.",
			Items: []NoteItem{
				{Label: "Adresse genau treffen", Text: "Die Pin-Spitze sitzt auf der konfigurierten Position statt den Pin-Körper um die Adresse zu zentrieren."},
				{Label: "Fläche konsequent nutzen", Text: "Der feste Kartenausschnitt kommt ohne Rahmen, abgerundete Zusatzkarte oder ungenutzten Seitenrand aus."},
				{Label: "Typografie beruhigen", Text: "Adresse, reduzierter Trenner und Portalname folgen einer schlichten Sans-Serif-Hierarchie ohne Link-Unterstreichung."},
				{Label: "Quelle sauber nennen", Text: "Die OpenStreetMap-Quellenangabe steht dezent im Fußbereich statt über der Karte."},
			},
		},
		{
			Version:  "0.54.0",
			Date:     "29. Juli 2026",
			Kind:     "Hausportal",
			Headline: "Die Seitenleiste verortet das eigene Haus jetzt ruhig und eindeutig.",
			Intro:    "Ein fester OpenStreetMap-Ausschnitt verbindet Adresse, konfigurierbares Hauszeichen und Portalname zu einem klaren Ortskopf.",
			Items: []NoteItem{
				{Label: "Haus wiedererkennen", Text: "Das gewählte Hauszeichen sitzt direkt im Pin an der konfigurierten Adresse."},
				{Label: "Karte ruhig halten", Text: "Der Ausschnitt lässt sich weder verschieben noch zoomen und lädt keine Karten-Skripte."},
				{Label: "Adresse öffnen", Text: "Ein dezenter Link unter der Karte führt bei Bedarf zur vollständigen OpenStreetMap-Ansicht."},
				{Label: "Mobil Platz bewahren", Text: "Die Ortsmarke wird auf kleinen Bildschirmen kompakt, ohne Menü oder Inhalt zu verdrängen."},
			},
		},
		{
			Version:  "0.53.0",
			Date:     "29. Juli 2026",
			Kind:     "Mein Zuhause",
			Headline: "Der Energieverlauf lässt sich jetzt groß öffnen und Punkt für Punkt lesen.",
			Intro:    "Ein ruhiger Detailmodus zeigt die vier Energieflüsse mit Uhrzeit und Viertelstundenwert – per Maus, Touch oder Tastatur.",
			Items: []NoteItem{
				{Label: "Groß ansehen", Text: "Der neue Vergrößern-Knopf öffnet Kurven, Skala und Planungsgrenze in einem eigenen übersichtlichen Detailmodus."},
				{Label: "Werte ablesen", Text: "Ein Fadenkreuz verbindet den gewählten Zeitpunkt mit Hausverbrauch, PV-Erzeugung, Netzbezug oder Einspeisung sowie Laden oder Entladen des Speichers."},
				{Label: "Ohne Maus bedienen", Text: "Pfeiltasten, Escape, Fokus-Rückkehr und ausreichend große Touch-Ziele machen die Interaktion auf Desktop und Mobil vollständig zugänglich."},
			},
		},
		{
			Version:  "0.52.0",
			Date:     "29. Juli 2026",
			Kind:     "Mein Zuhause",
			Headline: "Verbrauch, Planungsgrenze und Speicher sind jetzt auf einen Blick verständlich.",
			Intro:    "Der Tagesverlauf erhält eine ruhige, symmetrische Skala; der Live-Bereich zeigt den Speicher wie eine Batterie und führt dauerhaft zum Messwert-Setup.",
			Items: []NoteItem{
				{Label: "Verbrauch erkennen", Text: "Eine gedämpfte rote Linie mit leichter Fläche hebt den Hausverbrauch von PV, Netz und Speicher ab."},
				{Label: "Skala vergleichen", Text: "25 Prozent Luft und runde 5-kW-Grenzen halten Bezug, Einspeisung, Laden und Entladen gemeinsam lesbar."},
				{Label: "Planungswert einordnen", Text: "Die 10-kW-Linie ist ausdrücklich konfigurierbar und bleibt eine Modellannahme statt eines Tarifversprechens."},
				{Label: "Speicher verstehen", Text: "Batterie-Füllstand und aktuelle Lade- oder Entladeleistung stehen kompakt zusammen."},
				{Label: "Messwerte zuordnen", Text: "Sechs verständliche Rollen erklären, welche Sensoren benötigt und welche Spitzen daraus automatisch berechnet werden."},
			},
		},
		{
			Version:  "0.51.0",
			Date:     "29. Juli 2026",
			Kind:     "Mein Zuhause",
			Headline: "Die letzten 24 Stunden erzählen jetzt verständlich, was energetisch im Haus passiert ist.",
			Intro:    "Ein ruhiger Verlauf verbindet Hausverbrauch, PV, Netz und Speicher – ohne Technik-Dashboard und ohne externe Diagrammverbindung.",
			Items: []NoteItem{
				{Label: "Tagesverlauf verstehen", Text: "Vier vergleichbare Leistungslinien zeigen, wann viel Energie gebraucht und woher sie bezogen wurde."},
				{Label: "Spitze einordnen", Text: "Eine kurze Zusammenfassung nennt Zeitpunkt und Höhe der größten Last sowie die gemessene Herkunft."},
				{Label: "Mobil lesen", Text: "Eine eigene schmale Darstellung hält Achsen, Zeiten und Linien auch auf kleinen Bildschirmen klar."},
				{Label: "Aktualität erkennen", Text: "Der Datenstatus folgt den tatsächlich gerade gelesenen Home-Assistant-Werten."},
			},
		},
		{
			Version:  "0.50.0",
			Date:     "29. Juli 2026",
			Kind:     "Mein Zuhause",
			Headline: "Viele Energiemesswerte werden jetzt zu einem ruhigen, verständlichen Hausbild.",
			Intro:    "Der Live-Bereich erklärt den Energiefluss, während technische und kumulierte Werte erst bei Bedarf erscheinen.",
			Items: []NoteItem{
				{Label: "Haus im Blick", Text: "Hausverbrauch, PV, Netz, Speicher und Batteriestand stehen kompakt in einem gemeinsamen Zusammenhang."},
				{Label: "Werte leichter lesen", Text: "Große Wattwerte erscheinen passend in Kilowatt; Laden und Entladen werden als ein Speicherzustand zusammengefasst."},
				{Label: "Details dosieren", Text: "Weitere Messwerte bleiben erreichbar, verdrängen aber weder die nächste Empfehlung noch den Energie-Fahrplan."},
				{Label: "Aufgabe sicher öffnen", Text: "Das Freigabeformular bleibt im normalen Seitenfluss und überlagert keine folgenden Inhalte mehr."},
			},
		},
		{
			Version:  "0.49.1",
			Date:     "29. Juli 2026",
			Kind:     "Mein Zuhause",
			Headline: "Zuhause-Art und Hauszeichen sind jetzt auf den ersten Blick verständlicher.",
			Intro:    "Das Onboarding erklärt die Wirkung jeder Auswahl direkt; die Seitenleiste gibt dem bestehenden Drei-Häuser-Zeichen deutlich mehr Raum.",
			Items: []NoteItem{
				{Label: "Auswahl verstehen", Text: "Wohnung, Einfamilienhaus und Hausgemeinschaft beschreiben sofort, welchen Bereich der spätere Überblick umfasst."},
				{Label: "Sicherheitsgrenze kennen", Text: "Die Auswahl verändert weder Zugriffsrechte noch den dauerhaft sichtbaren Modus „Nur beobachten“."},
				{Label: "Haus wiedererkennen", Text: "Das vertraute Drei-Häuser-Zeichen steht größer und ruhiger neben Adresse und Portalbezeichnung."},
			},
		},
		{
			Version:  "0.49.0",
			Date:     "28. Juli 2026",
			Kind:     "Mein Zuhause",
			Headline: "Das Cockpit zeigt jetzt sofort, welche Anlagen wirklich gemessen und welche erst erfasst sind.",
			Intro:    "Messpunkte bleiben hausbezogen, lesend und verständlich. Die drei privaten Pilotkonstellationen sind getrennt auf Desktop und Mobil durchgespielt.",
			Items: []NoteItem{
				{Label: "Messung zuordnen", Text: "PV- und Speicherwerte finden bei eindeutigem Bestand die richtige Anlage; manuelle Zuordnungen bleiben bewusst steuerbar."},
				{Label: "Lücken überblicken", Text: "Eine flache Übersicht unterscheidet gemessene Bereiche von lediglich erfassten Verbrauchern, ohne Sensor-IDs zu zeigen."},
				{Label: "Piloten absichern", Text: "Wohnung, Haus A ohne Speicher und Haus B mit Speicher bleiben getrennt und starten ausschließlich mit „Nur beobachten“."},
				{Label: "Technikhilfe begrenzen", Text: "Eigene Betreuungskonten dürfen einrichten, erhalten aber niemals den Eigentümer-Schalter für die aktive Steuerung."},
			},
		},
		{
			Version:  "0.48.2",
			Date:     "28. Juli 2026",
			Kind:     "Sicherheitsmodus",
			Headline: "Den Hausmodus dürfen ausschließlich Eigentümer oder Hausadministration bewusst umlegen.",
			Intro:    "Technische Vertrauenspersonen können Energiedaten weiterhin ansehen und die Einrichtung unterstützen, erhalten aber keine Freigabe für die hausweite Steuerung.",
			Items: []NoteItem{
				{Label: "Hausentscheidung schützen", Text: "Eigentümer, Verwalter beziehungsweise Hausadministration und Admin behalten die bewusste Modusfreigabe."},
				{Label: "Technikhilfe begrenzen", Text: "Ansehen und Einrichten bleiben delegierbar; ein früheres Steuerungsrecht erweitert den Zugriff nicht."},
				{Label: "Rechte verständlich halten", Text: "Einladung und Rechteverwaltung zeigen nur die tatsächlich delegierbaren Aufgaben."},
			},
		},
		{
			Version:  "0.48.1",
			Date:     "28. Juli 2026",
			Kind:     "Mein Zuhause",
			Headline: "Home Assistant schlägt jetzt nur wenige, eindeutig lesende Hausenergie-Messwerte vor.",
			Intro:    "Die Auswahl bleibt ruhig und verständlich: sichere Empfehlungen stehen vorn, technische Alternativen öffnen sich nur bei Bedarf und bestehende Zuordnungen bleiben erhalten.",
			Items: []NoteItem{
				{Label: "Geräte-Rauschen ausblenden", Text: "Handy-, Schloss-, Roboter-, Fahrzeug- und Prognosewerte gelangen nicht in die Hausenergie-Auswahl."},
				{Label: "Wenige Werte verstehen", Text: "Höchstens fünf Empfehlungen tragen Alltagsnamen und den sichtbaren Hinweis „Nur lesen“; Sensor-IDs bleiben in den Technikdetails."},
				{Label: "Bewusste Auswahl schützen", Text: "Bestätigte oder manuell benannte Messwerte werden durch eine neue Suche oder den Offline-Weg nicht entfernt."},
			},
		},
		{
			Version:  "0.48.0",
			Date:     "28. Juli 2026",
			Kind:     "Mein Zuhause",
			Headline: "Wartung, Tarife und Fachhilfe führen jetzt vom Hinweis bis zum belegten Ergebnis.",
			Intro:    "Das Energie-Cockpit zeigt weiterhin genau einen nächsten Schritt; weiterführende Arbeit bleibt bei Bedarf erreichbar und sicher dem richtigen Haus zugeordnet.",
			Items: []NoteItem{
				{Label: "Wartung vorausplanen", Text: "Wiederkehrende Termine verbinden Anlage, Fachkontakt, Unterlage, Aufgabe und Erledigungsnachweis."},
				{Label: "Tarife nachvollziehen", Text: "Monatliche Bewertungen bewahren Messspitze, Datenqualität, Regelprofil und Regelversion als unveränderlichen Wissensstand."},
				{Label: "Fachhilfe koordinieren", Text: "Bekannte Energie-Fachbetriebe und Maßnahmen verbinden Angebot, Termin, Arbeit, Nachweis sowie Vorher-/Nachher-Vergleich, ohne automatisch Portalzugang oder Zahlung auszulösen."},
				{Label: "Betreuung sauber trennen", Text: "Technische Vertrauenspersonen lassen sich direkt für ein Haus einladen; Ansehen, Einrichten und Steuerungsfreigabe bleiben getrennt und widerrufbar."},
			},
		},
		{
			Version:  "0.47.0",
			Date:     "28. Juli 2026",
			Kind:     "Mein Zuhause",
			Headline: "Das private Haus-Cockpit führt vom Verstehen zur sicheren Energieplanung.",
			Intro:    "Ein ruhiges Onboarding, lesende Messquellen und genau eine nächste Empfehlung machen Peak Shaving auch ohne technische Vorkenntnisse zugänglich.",
			Items: []NoteItem{
				{Label: "Sicher beginnen", Text: "Der dauerhaft sichtbare Modus „Nur beobachten“ ist Standard; eine bewusste Freigabe startet zunächst ohne Gerätewirkung im Testlauf."},
				{Label: "Messung verstehen", Text: "Home Assistant, Smart Meter, Viertelstunden-Spitze, Datenqualität und unverbindliche Szenarien bleiben nachvollziehbar getrennt."},
				{Label: "Hilfe gezielt teilen", Text: "Technische Rechte und Datenfreigaben sind hausbezogen, widerrufbar und erzeugen weder automatische Aufträge noch Zahlungen."},
				{Label: "Preisrahmen einordnen", Text: "Private Hausprofile nutzen drei Jahre den vollen Umfang kostenlos; die Gemeinschaft bleibt im Pilot bis 25 Wohneinheiten kostenfrei und behält ihren gesonderten Zukunftsrichtwert je Wohneinheit."},
			},
		},
		{
			Version:  "0.46.0",
			Date:     "28. Juli 2026",
			Kind:     "Öffentliche Startseite",
			Headline: "Die Startseite erklärt Nutzen, Produktstand und Preisrichtung in etwa halb so viel Raum.",
			Intro:    "Fünf Kernaufgaben und zwei klare Produktzustände ersetzen lange Listen; Details bleiben bei Bedarf direkt erreichbar.",
			Items: []NoteItem{
				{Label: "Schneller verstehen", Text: "Informieren, Anliegen klären, Unterlagen ordnen, entscheiden und Rechte schützen bilden den gesamten Einstieg."},
				{Label: "Ehrlich einordnen", Text: "Heute nutzbare Bereiche und nächste Ausbaustufen stehen sichtbar getrennt, ohne interne Freigabebegriffe."},
				{Label: "Preise vergleichen", Text: "Kostenfreier Pilotrahmen, Zukunftsrichtwert und Beispiele für Haus und Verwaltung stehen kompakt beieinander."},
			},
		},
		{
			Version:  "0.45.1",
			Date:     "28. Juli 2026",
			Kind:     "Datenschutz & Betrieb",
			Headline: "Betriebsprotokolle bleiben aussagekräftig, ohne konkrete Portaladressen zu speichern.",
			Intro:    "Stabile Routennamen ersetzen konkrete Pfade; eine automatische Prüfung schützt zusätzlich vor unnötigen persönlichen oder geheimen Angaben.",
			Items: []NoteItem{
				{Label: "Token schützen", Text: "Kalender- und Übergabezugriffe hinterlassen nur den Routentyp, niemals den geheimen Bestandteil ihrer Adresse."},
				{Label: "Datensparsam prüfen", Text: "Die Logprüfung erkennt Klartext-E-Mail-Adressen, sensitive Felder und fehlende Betriebsangaben automatisch."},
				{Label: "Diskret auswerten", Text: "Produktionsstichproben zeigen ausschließlich Zähler und Abweichungsarten statt konkreter Logwerte."},
			},
		},
		{
			Version:  "0.45.0",
			Date:     "28. Juli 2026",
			Kind:     "Orientierung",
			Headline: "Leere Seiten und der persönliche Verlauf erklären sich jetzt ohne Systemwissen.",
			Intro:    "Werkzeuge erscheinen erst, wenn sie gebraucht werden; Verlaufseinträge nennen verständliche Vorgänge statt technischer Kennungen.",
			Items: []NoteItem{
				{Label: "Ruhig starten", Text: "Leere Ablagen und Listen zeigen eine kurze Erklärung, ohne Suche und Filter ohne Inhalt anzubieten."},
				{Label: "Vorgänge verstehen", Text: "Anliegen erscheinen im persönlichen Verlauf mit ihrem Titel und klaren Änderungen wie Rückfrage oder Bearbeitung."},
				{Label: "Technik schützen", Text: "Interne Begriffe bleiben aus der normalen Oberfläche heraus, während die vollständigen Auditdaten für berechtigte Verwaltung erhalten bleiben."},
			},
		},
		{
			Version:  "0.44.0",
			Date:     "28. Juli 2026",
			Kind:     "Hausüberblick",
			Headline: "Der Hausüberblick führt jetzt mit einer Aufgabe statt mit vielen Karten.",
			Intro:    "Eine klare Hauptaktion und wenige ruhige Folgepunkte zeigen sofort, was zu tun ist und was lediglich im Blick bleiben soll.",
			Items: []NoteItem{
				{Label: "Direkt handeln", Text: "Anliegen, Rückfragen, Aushänge und offene Zahlungsstatus führen ohne Umweg zum passenden nächsten Schritt."},
				{Label: "Weniger wiederholen", Text: "Inhalte aus Aushang, Anliegen, Terminen und Dokumenten werden nicht mehr als konkurrierende Dashboard-Karten dupliziert."},
				{Label: "Verwaltung einordnen", Text: "Parkplatznutzung, Übergaben und Benutzerrechte bleiben erreichbar, stehen aber sichtbar hinter dem normalen Hausalltag."},
			},
		},
		{
			Version:  "0.43.0",
			Date:     "28. Juli 2026",
			Kind:     "Anmeldung",
			Headline: "Der Hauszugang ist jetzt einfacher, persönlicher und frei von Anbieterjargon.",
			Intro:    "Hausname, Adresse und eine einzige klare Anmeldung stehen im Mittelpunkt; der E-Mail-Link bleibt als verständliche Alternative erreichbar.",
			Items: []NoteItem{
				{Label: "Einfach anmelden", Text: "Ein neutraler Anmeldeknopf führt zum sicheren Hauszugang, ohne technische Produktnamen erklären zu müssen."},
				{Label: "Freundlich zurückfinden", Text: "Abgelaufene Links führen zur vertrauten Startseite und direkt zur Anforderung eines neuen Links."},
				{Label: "Haus im Mittelpunkt", Text: "Das reduzierte Hauszeichen, die Adresse und ein datensparsamer Kartenhinweis geben sofort Orientierung."},
			},
		},
		{
			Version:  "0.42.0",
			Date:     "28. Juli 2026",
			Kind:     "Anliegen",
			Headline: "Ein neues Anliegen entsteht in drei kurzen, verständlichen Schritten.",
			Intro:    "Ereignis, Ort und Prüfung stehen nacheinander im Mittelpunkt, damit Bewohner nie eine ganze Formularwand verarbeiten müssen.",
			Items: []NoteItem{
				{Label: "Ruhig beschreiben", Text: "Art, kurze Beschreibung und ein optionales Foto gehören direkt zum Ereignis."},
				{Label: "Ort ergänzen", Text: "Der Bereich und eine freiwillige genauere Angabe bilden einen eigenen, kurzen Schritt."},
				{Label: "Vor dem Senden prüfen", Text: "Eine lesbare Zusammenfassung und ein editierbarer Titel machen die fertige Meldung transparent."},
			},
		},
		{
			Version:  "0.41.0",
			Date:     "28. Juli 2026",
			Kind:     "Anliegen",
			Headline: "Rückfragen und Lösungen zeigen genau die passende nächste Aufgabe.",
			Intro:    "Informationen bleiben ruhig im Verlauf; offene Fragen führen direkt zur Antwort und abgeschlossene Arbeiten zur gemeinsamen Bestätigung.",
			Items: []NoteItem{
				{Label: "Gezielt antworten", Text: "Nur eine echte Rückfrage öffnet ein Antwortfeld mit einer klaren Hauptaktion."},
				{Label: "Ohne Druck informiert", Text: "Reine Informationen sind sichtbar, verlangen aber keine Reaktion."},
				{Label: "Gemeinsam abschließen", Text: "Bewohner bestätigen die vorgeschlagene Lösung oder öffnen das Anliegen wieder."},
			},
		},
		{
			Version:  "0.40.0",
			Date:     "28. Juli 2026",
			Kind:     "Anliegen",
			Headline: "Anliegen werden in zwei klaren Entscheidungen bearbeitet.",
			Intro:    "Dringlichkeit und Zuständigkeit stehen nacheinander im Mittelpunkt; Verlauf, Dateien und Kosten bleiben bei Bedarf erreichbar.",
			Items: []NoteItem{
				{Label: "Einfach priorisieren", Text: "Alltagsnahe Auswahlmöglichkeiten übersetzen die technische Priorität in eine verständliche Entscheidung."},
				{Label: "Klar zuordnen", Text: "Die Verwaltung übernimmt ein Anliegen selbst oder lässt die Zuständigkeit bewusst noch offen."},
				{Label: "Im Kontext bleiben", Text: "Nach jedem Schritt bleibt dasselbe Anliegen geöffnet und bestätigt den erreichten Stand."},
			},
		},
		{
			Version:  "0.39.0",
			Date:     "27. Juli 2026",
			Kind:     "Datenübergabe",
			Headline: "Ausgewählte Hausdaten werden als prüfbares Paket weitergegeben.",
			Intro:    "Ein kurzer Dreischritt verbindet bewusste Auswahl, klare Vorschau und einen geschützten, nachvollziehbaren Download.",
			Items: []NoteItem{
				{Label: "Bewusst auswählen", Text: "Nichts ist vorausgewählt; Verwalter und Admins bestimmen den Datenbereich vor jeder Übergabe."},
				{Label: "Einfach prüfen", Text: "CSV, Manifest, Datensatzanzahl und SHA-256-Prüfsumme machen den Inhalt transparent und reproduzierbar."},
				{Label: "Sicher begrenzen", Text: "Vorschau und Download bleiben haus- und personengebunden; raw-v0 erzeugt keine Buchungen und verspricht keine BMD-/RZL-Kompatibilität."},
			},
		},
		{
			Version:  "0.38.0",
			Date:     "27. Juli 2026",
			Kind:     "E-Rechnungen",
			Headline: "E-Rechnungen werden vor der geschützten Ablage verständlich geprüft.",
			Intro:    "Ein kurzer ebInterface-Ablauf zeigt nur die entscheidenden Rechnungsdaten und speichert das unveränderte Original ausschließlich für die Verwaltung.",
			Items: []NoteItem{
				{Label: "Vorher verstehen", Text: "Rechnungsnummer, Betrag, Beteiligte und Termine stehen vor jeder Ablage in einer ruhigen Vorschau."},
				{Label: "Geschützt ablegen", Text: "Bestätigte Originaldateien bleiben für Bewohner unsichtbar und können nicht doppelt übernommen werden."},
				{Label: "Sauber begrenzen", Text: "Der Ablauf bucht und bezahlt nichts; ebInterface 5.0 und 6.0 sind mit offiziellen Schema-Checks und automatisierten Tests nachgewiesen."},
			},
		},
		{
			Version:  "0.37.1",
			Date:     "27. Juli 2026",
			Kind:     "Zahlungsstatus",
			Headline: "Der Bankdatei-Import ist auf kleinen Bildschirmen leichter zu bedienen.",
			Intro:    "Die letzten Aktionsfelder des neuen Ablaufs folgen nun auch bei 390 Pixeln vollständig den großen mobilen Touch-Zielen.",
			Items: []NoteItem{
				{Label: "Sicher tippen", Text: "Zeitraum, Zurück-Aktion und Navigationswege bieten mindestens 44 Pixel hohe Trefferflächen."},
			},
		},
		{
			Version:  "0.37.0",
			Date:     "27. Juli 2026",
			Kind:     "Zahlungsstatus",
			Headline: "Bankdateien werden erst geprüft und danach gezielt übernommen.",
			Intro:    "Ein kurzer, geschützter Ablauf verbindet Monatsreferenzen, datensparsame Vorschau und nachvollziehbare Statusänderungen.",
			Items: []NoteItem{
				{Label: "Vorher prüfen", Text: "Verwalter und Admins sehen eindeutige, unklare und abgelehnte Treffer, bevor sich ein Zahlungsstatus ändert."},
				{Label: "Daten schützen", Text: "Die Bankdatei wird nicht gespeichert; IBAN, Namen und Verwendungszwecke erscheinen weder in Vorschau noch Audit."},
				{Label: "Einmal übernehmen", Text: "Nur eindeutige Treffer werden protokolliert gespeichert, während derselbe Dateinachweis keine doppelte Übernahme erlaubt."},
			},
		},
		{
			Version:  "0.36.0",
			Date:     "27. Juli 2026",
			Kind:     "Portal",
			Headline: "Die mobile Navigation ist leichter zu treffen und durchgängig konsistent.",
			Intro:    "Der abschließende Gesamtrundgang verbindet alle überarbeiteten Portalwege: Rollen, Navigation, Dialoge und kleine Bildschirme folgen denselben verlässlichen Mustern.",
			Items: []NoteItem{
				{Label: "Sicher tippen", Text: "Menü, Navigationswege, Versionsverlauf und Abmelden bieten auf kleinen Bildschirmen mindestens 44 Pixel hohe Touch-Ziele."},
				{Label: "Ruhig orientieren", Text: "Längere Navigationsbezeichnungen bleiben in der mobilen Zweispaltenansicht stabil und frei von seitlichem Überlauf."},
				{Label: "Gemeinsam geprüft", Text: "Eigentümer-, Verwalter- und Admin-Wege sowie die geschlossene Dienstleister-Freigabe wurden über Desktop und Mobil hinweg gegengeprüft."},
			},
		},
		{
			Version:  "0.35.0",
			Date:     "27. Juli 2026",
			Kind:     "Parkplatz-Verwaltung",
			Headline: "Zugriff, Abrechnung, Laderegeln und Telegram sind klare einzelne Aufgaben.",
			Intro:    "Eine gemeinsame Abschnittsnavigation ersetzt die lange Sammelseite; pro Arbeitsbereich stehen Zustand, nächste Aktion und nur die dafür nötigen Entscheidungen im Vordergrund.",
			Items: []NoteItem{
				{Label: "Zugriff pflegen", Text: "Kompakte Personenkarten zeigen Rolle, Freigabe und genau die zulässige Aktion, ohne Zahlungsdaten zu vermischen."},
				{Label: "Sicher einstellen", Text: "Tarife und Laderegeln erklären zuerst ihre Wirkung; Historie, Grenzwerte und Regler-Ereignisse öffnen sich bei Bedarf."},
				{Label: "Richtig begrenzen", Text: "Navigation und Speicherung beachten Verwaltungsrolle und Hauszugehörigkeit auch bei gezielten Parkplatzänderungen."},
			},
		},
		{
			Version:  "0.34.0",
			Date:     "27. Juli 2026",
			Kind:     "Parkplatz",
			Headline: "Laden, Monatskosten und Zahlung bilden einen kurzen, sicheren Ablauf.",
			Intro:    "Der aktuelle Zustand und genau eine nächste Aktion stehen zuerst; Abrechnungs- und Technikdetails bleiben vollständig, öffnen sich aber nur bei Bedarf.",
			Items: []NoteItem{
				{Label: "Sicher laden", Text: "Automatik und manuelles Normalladen sind klar getrennt, damit die beabsichtigte Aktion eindeutig bleibt."},
				{Label: "Monat verstehen", Text: "Gesamtbetrag, Verbrauch und Status führen in eine eigene Abrechnung ohne doppelte Kennzahlen."},
				{Label: "Gezielt nachsehen", Text: "Kosten, Stunden, Ladevorgänge sowie CSV- und PDF-Ausgabe bleiben in kurzen, progressiven Bereichen erreichbar."},
			},
		},
		{
			Version:  "0.33.0",
			Date:     "27. Juli 2026",
			Kind:     "Verlauf",
			Headline: "Änderungen und Zugriffe lassen sich wie eine ruhige Zeitleiste lesen.",
			Intro:    "Ein kompakter Überblick, passende Filter und direkt am Ereignis erreichbare Details ersetzen die bisher gleichzeitig sichtbare Informationsdichte.",
			Items: []NoteItem{
				{Label: "Schnell verstehen", Text: "Zeit, verständlicher Vorgang und freigegebener Kontext bilden pro Ereignis eine klare Zeile."},
				{Label: "Gezielt eingrenzen", Text: "Der Filter bietet nur Änderungsarten an, die im eigenen sichtbaren Verlauf tatsächlich vorkommen."},
				{Label: "Sicher nachsehen", Text: "Technische Details bleiben auf Wunsch erreichbar, während Rollen- und Datenschutzgrenzen unverändert gelten."},
			},
		},
		{
			Version:  "0.32.0",
			Date:     "27. Juli 2026",
			Kind:     "Einstellungen",
			Headline: "Profil und Benachrichtigungen sind einfacher zu verstehen und schneller eingestellt.",
			Intro:    "Konto, Sichtbarkeit und Kommunikation bilden klare Bereiche; der aktuelle Zustand und die jeweils nächste Aktion bleiben auf Desktop und Mobil sofort erkennbar.",
			Items: []NoteItem{
				{Label: "Konto überblicken", Text: "Name, E-Mail, Rolle, Profil und Benachrichtigungsstatus stehen gemeinsam in einer ruhigen Übersicht."},
				{Label: "Sichtbarkeit verstehen", Text: "Die freiwillige Kontaktfreigabe erklärt, welche persönlichen Angaben für die Hausgemeinschaft sichtbar werden."},
				{Label: "E-Mails steuern", Text: "Der Versand lässt sich insgesamt pausieren, ohne die Auswahl der sechs verständlich gruppierten Themen zu verlieren."},
			},
		},
		{
			Version:  "0.31.0",
			Date:     "26. Juli 2026",
			Kind:     "Übergaben",
			Headline: "Übergaben führen klar vom Erfassen bis zur sicheren Ablage.",
			Intro:    "Arbeitskörbe und kompakte Karten zeigen den nächsten Schritt; bestätigende Personen prüfen den tatsächlichen Protokollinhalt vor ihrer ausdrücklichen Bestätigung.",
			Items: []NoteItem{
				{Label: "Arbeit überblicken", Text: "Offene, ablagebereite und abgeschlossene Übergaben sind klar getrennt."},
				{Label: "Inhalt prüfen", Text: "Räume, Zählerstände, Schlüssel, Notizen und Dateien bleiben vor der Bestätigung übersichtlich nachvollziehbar."},
				{Label: "Stand schützen", Text: "Ab der ersten Bestätigung sind Dateien unveränderlich; die Ablage folgt erst nach allen vorgesehenen Bestätigungen."},
			},
		},
		{
			Version:  "0.30.0",
			Date:     "26. Juli 2026",
			Kind:     "Gebäude",
			Headline: "Gebäude, Einheiten und Zahlungsstatus sind deutlich ruhiger zu verwalten.",
			Intro:    "Klare Bereiche und kompakte Einheitenzeilen zeigen zuerst den Überblick; vollständige Formulare öffnen sich erst bei Bedarf.",
			Items: []NoteItem{
				{Label: "Schnell orientieren", Text: "Stammdaten, Hauskontakte, Einheiten und Erscheinungsbild lassen sich direkt anspringen."},
				{Label: "Kompakt verwalten", Text: "Typ, Anteil, Personen und Zahlungsstatus stehen gemeinsam an der jeweiligen Einheit."},
				{Label: "Mobil speichern", Text: "Kontextbezogene Speichern-Aktionen bleiben auf kleinen Bildschirmen gut erreichbar."},
			},
		},
		{
			Version:  "0.29.0",
			Date:     "26. Juli 2026",
			Kind:     "Kontakte",
			Headline: "Die richtige Ansprechperson ist schneller gefunden und direkt erreichbar.",
			Intro:    "Notdienst und Hausverwaltung stehen zuerst; weitere Hauskontakte, Adressbuch und freiwilliges Bewohnerverzeichnis bleiben klar getrennt.",
			Items: []NoteItem{
				{Label: "Direkt erreichen", Text: "Anrufen und E-Mail sind als gut antippbare Hauptaktionen an jedem freigegebenen Kontakt verfügbar."},
				{Label: "Ruhig verwalten", Text: "Neue und inaktive Adressbucheinträge sind progressiv offengelegt, ohne die Kontaktübersicht zu überladen."},
				{Label: "Klar einordnen", Text: "Offizielle Haus-Ansprechpersonen und freiwillig freigegebene Bewohnerkontakte erscheinen in getrennten Bereichen."},
			},
		},
		{
			Version:  "0.28.0",
			Date:     "26. Juli 2026",
			Kind:     "Zugänge",
			Headline: "Benutzer und Rechte lassen sich schneller erfassen und sicherer ändern.",
			Intro:    "Handlungsbedarf, Rolle, Einheit und Status stehen im Vordergrund; optionale Identitäts- und Anmeldeinformationen bleiben gezielt erreichbar.",
			Items: []NoteItem{
				{Label: "Schnell überblicken", Text: "Aktive, eingeladene und deaktivierte Zugänge sind kompakt zusammengefasst und sinnvoll sortiert."},
				{Label: "Kurz einladen", Text: "E-Mail und Rolle bilden den Standardweg; Name, Sonderrechte und Anmeldewege sind optional."},
				{Label: "Sicher verwalten", Text: "Deaktivieren bleibt reaktivierbar, dauerhaftes Entfernen ist klar getrennt und bestätigt."},
			},
		},
		{
			Version:  "0.27.0",
			Date:     "26. Juli 2026",
			Kind:     "Abstimmungen",
			Headline: "Abstimmungen führen klarer von der Frage bis zum nachvollziehbaren Ergebnis.",
			Intro:    "Handlungsbedarf, Frist, Stimmgewicht und Hauptaktion stehen im Vordergrund; Regeln und Verwaltung bleiben dort erreichbar, wo sie gebraucht werden.",
			Items: []NoteItem{
				{Label: "Klar abstimmen", Text: "Eigentümer sehen Frage, Frist, Gewicht und Antwortmöglichkeiten in einer ruhigen Entscheidungsfolge."},
				{Label: "Sicher steuern", Text: "Entwurf, Öffnen und Schließen finden in einer gemeinsamen Verwaltungsübersicht statt."},
				{Label: "Ergebnis verstehen", Text: "Teilnahme, Quorum, Auszählung und Protokoll ersetzen nach Abschluss die Eingabeflächen."},
			},
		},
		{
			Version:  "0.26.0",
			Date:     "26. Juli 2026",
			Kind:     "Dokumente",
			Headline: "Die Hausablage zeigt schneller die richtige Unterlage und den nächsten sinnvollen Schritt.",
			Intro:    "Nur belegte Kategorien bleiben sichtbar; Suche, Vorschau, Download und Verwaltung sind klar nach ihrer täglichen Bedeutung geordnet.",
			Items: []NoteItem{
				{Label: "Schneller finden", Text: "Leere Kategorien entfallen und Suche sowie Sortierung bilden eine kompakte Werkzeugleiste."},
				{Label: "Klar ansehen", Text: "Die Vorschau ist die Hauptaktion; Dateidetails, Versionsverlauf und Verwaltung bleiben bei Bedarf erreichbar."},
				{Label: "Sicher veröffentlichen", Text: "Titel, Kategorie, Sichtbarkeit und Datei stehen zuerst; optionale Zuordnung und Ersetzen erklären ihre Wirkung."},
			},
		},
		{
			Version:  "0.25.0",
			Date:     "26. Juli 2026",
			Kind:     "Termine & Triage",
			Headline: "Termine und offene Anliegen lassen sich schneller erfassen und bearbeiten.",
			Intro:    "Die Oberfläche zeigt zuerst den nächsten sinnvollen Schritt und hält zusätzliche Angaben kompakt im Hintergrund bereit.",
			Items: []NoteItem{
				{Label: "Termine", Text: "Der nächste Termin steht im Mittelpunkt; Vergangenheit, Kalender-Abo und weitere Details bleiben bei Bedarf erreichbar."},
				{Label: "Schneller veröffentlichen", Text: "Der Termindialog startet mit den wesentlichen Angaben und führt nach dem Speichern direkt zum Ergebnis."},
				{Label: "Ruhige Triage", Text: "Filter, Status, nächster Schritt und Hauptaktion bilden im Anliegen-Board eine klarere visuelle Einheit."},
			},
		},
		{
			Version:  "0.24.0",
			Date:     "26. Juli 2026",
			Kind:     "Portal-Alltag",
			Headline: "Die wichtigsten Wege sind ruhiger, kürzer und klar auf den nächsten Schritt ausgerichtet.",
			Intro:    "Hausüberblick, Anliegen und Aushänge zeigen zuerst das Wesentliche; weitere Angaben bleiben bei Bedarf erreichbar.",
			Items: []NoteItem{
				{Label: "Klare Orientierung", Text: "Die mobile Kopfzeile nennt die aktuelle Seite und der Hausüberblick priorisiert echten Handlungsbedarf."},
				{Label: "Anliegen", Text: "Bewohner sehen Status und nächsten Schritt sofort; die Verwaltung arbeitet in einem kompakten Triage-Board weiter."},
				{Label: "Aushänge", Text: "Ein eindeutiger Erstellen-Einstieg und automatisch geöffnete neue Beiträge vereinfachen Veröffentlichen und Lesen."},
			},
		},
		{
			Version:  "0.23.1",
			Date:     "26. Juli 2026",
			Kind:     "Datenschutz",
			Headline: "Verantwortliche Stelle, Haus und technische Empfänger sind eindeutig beschrieben.",
			Intro:    "Die Datenschutzinformation trennt die Anschrift des Hausbetriebs von der Adresse des betroffenen Hauses und nennt die tatsächliche Betriebs- und Sicherungsinfrastruktur.",
			Items: []NoteItem{
				{Label: "Klare Kontakte", Text: "Hausverwaltung, betroffenes Haus und technischer Betrieb erscheinen als getrennte Angaben."},
				{Label: "Vollständiger Datenfluss", Text: "Hosting, Webschutz, verschlüsselte Sicherungen, Identitätsdienst und Mailversand sind ihrem tatsächlichen Zweck entsprechend erklärt."},
			},
		},
		{
			Version:  "0.23.0",
			Date:     "26. Juli 2026",
			Kind:     "Transparenz",
			Headline: "Betreiber, Hauskontakt und privater Pilot sind klar eingeordnet.",
			Intro:    "Impressum und Datenschutzinformation nennen die verantwortlichen Kontakte vollständig und grenzen den persönlich abgestimmten Pilot von einem öffentlichen Vertragsangebot ab.",
			Items: []NoteItem{
				{Label: "Nachvollziehbar", Text: "Betreiberangaben, Prüftermin und österreichische Primärquellen sind direkt erreichbar."},
				{Label: "Ehrlicher Pilot", Text: "Kostenwerte bleiben eine unverbindliche Zukunftsorientierung; ein öffentlicher Online-Vertragsabschluss findet derzeit nicht statt."},
			},
		},
		{
			Version:  "0.22.0",
			Date:     "26. Juli 2026",
			Kind:     "Nachvollziehbarkeit",
			Headline: "Das Audit-Log zeigt jeder Rolle genau ihre freigegebene Historie.",
			Intro:    "Bewohner und Dienstleister können eigene Aktionen und aktuell zugängliche Vorgänge nachvollziehen, ohne interne Verwaltungsdetails zu sehen.",
			Items: []NoteItem{
				{Label: "Datensparsam", Text: "Personenbezogene Verwaltungsangaben, Dateinamen, Zahlungsreferenzen, Beträge und Bankdaten bleiben außerhalb der persönlichen Ansicht."},
				{Label: "Vollständiger", Text: "Anhangzugriffe, Löschungen und verarbeitete Zahlungsimporte ergänzen die nachvollziehbare Historie."},
			},
		},
		{
			Version:  "0.21.0",
			Date:     "26. Juli 2026",
			Kind:     "Übergaben",
			Headline: "Übergabeprotokolle lassen sich am Handy störungsfrei erfassen.",
			Intro:    "Fotos können vor dem Speichern geprüft werden, ohne dass die Speichern-Schaltfläche den Foto-Bereich verdeckt.",
			Items: []NoteItem{
				{Label: "Fotos", Text: "Anhänge lassen sich vor und nach dem Speichern entfernen; danach führt die Seite direkt zum Protokoll zurück."},
				{Label: "Klare Aufgabe", Text: "Übergabe, Bestätigung und PDF grenzen Zustandsdokumentation verständlich von Kaution, Schadenabrechnung und Buchhaltung ab."},
			},
		},
		{
			Version:  "0.20.0",
			Date:     "26. Juli 2026",
			Kind:     "Produkt-Ausblick",
			Headline: "Der Ausblick zeigt Status, Nutzen und Grenzen auf einen Blick.",
			Intro:    "Sieben zentrale Vorhaben sind jetzt eindeutig als verfügbar oder in Arbeit gekennzeichnet.",
			Items: []NoteItem{
				{Label: "Klarer Nutzen", Text: "Jeder Eintrag beschreibt kurz, welchen Alltagsschritt das Portal erleichtert."},
				{Label: "Klare Grenzen", Text: "Die Startseite nennt auch Aufgaben, die das Portal in der ersten Produktstufe bewusst nicht übernimmt."},
			},
		},
		{
			Version:  "0.19.0",
			Date:     "26. Juli 2026",
			Kind:     "Nachvollziehbarkeit",
			Headline: "Kalenderänderungen erscheinen in der Verwaltungshistorie.",
			Intro:    "Angelegte, geänderte und gelöschte Termine lassen sich im Audit-Log nach Aktion filtern.",
			Items: []NoteItem{
				{Label: "Datensparsam", Text: "Freitext, Ort und Dateinamen bleiben außerhalb des Audit-Protokolls."},
				{Label: "Klar zugeordnet", Text: "Akteur, Zeitpunkt, Änderungstyp und betroffener Termin bleiben nachvollziehbar."},
			},
		},
		{
			Version:  "0.18.1",
			Date:     "26. Juli 2026",
			Kind:     "Betrieb",
			Headline: "Betriebsmeldungen lassen sich gezielter auswerten.",
			Intro:    "Hintergrundvorgänge protokollieren Bereich, Haus und betroffene Objekte jetzt als getrennte Felder.",
			Items: []NoteItem{
				{Label: "Datensparsam", Text: "Personenbezüge bleiben in Betriebsmeldungen gekürzt statt im Klartext."},
				{Label: "Stabilität", Text: "Automatische Tests und Sicherheitsprüfungen laufen über die etablierte Blacksmith-Infrastruktur."},
			},
		},
		{
			Version:  "0.18.0",
			Date:     "26. Juli 2026",
			Kind:     "Datenschutz & Betrieb",
			Headline: "Freigaben und Betrieb werden nachvollziehbarer.",
			Intro:    "Datenschutzinformationen sind vor der Anmeldung erreichbar; Dienstleister bleiben bis zu einer dokumentierten, versionierten Betreiberfreigabe geschlossen.",
			Items: []NoteItem{
				{Label: "Datensparsam", Text: "Web-Schriften werden lokal vom Gerät verwendet statt von einem externen Anbieter geladen."},
				{Label: "Gesundheit", Text: "Die Betriebsprüfung kontrolliert jetzt Datenbank und beschreibbaren Datenspeicher."},
				{Label: "Nachvollziehbarkeit", Text: "Request-Logs nennen das betroffene Haus; alte Audit-Archive werden nach der Aufbewahrungsfrist entfernt."},
			},
		},
		{
			Version:  "0.17.7",
			Date:     "24. Juli 2026",
			Kind:     "Wartung",
			Headline: "Datenbestand bleibt dauerhaft überschaubar.",
			Intro:    "Reste gelöschter Anhänge werden nach einem Jahr endgültig aufgeräumt.",
			Items:    []NoteItem{{Label: "Unverändert", Text: "Bestehende Anhänge und der Versionsverlauf von Dokumenten bleiben erhalten."}},
		},
		{
			Version:  "0.17.6",
			Date:     "24. Juli 2026",
			Kind:     "Wartung",
			Headline: "Alte Foto-Sonderbehandlung bei Anliegen entfernt.",
			Intro:    "Nachdem alle älteren Fotos übernommen sind, entfällt der frühere Sonderweg; Fotos laufen einheitlich über die Anhang-Verwaltung.",
			Items:    []NoteItem{{Label: "Unverändert", Text: "Alle Fotos bleiben an ihrem Anliegen sichtbar."}},
		},
		{
			Version:  "0.17.5",
			Date:     "24. Juli 2026",
			Kind:     "Wartung",
			Headline: "Ältere Anliegen-Fotos in die reguläre Anhang-Verwaltung übernommen.",
			Intro:    "Fotos aus der Anfangszeit lagen noch in einer eigenen Ablage und wurden verlustfrei in die normale Anhang-Verwaltung übernommen.",
			Items:    []NoteItem{{Label: "Unverändert", Text: "Die Fotos bleiben an ihrem Anliegen sichtbar."}},
		},
		{
			Version:  "0.17.4",
			Date:     "24. Juli 2026",
			Kind:     "Wartung",
			Headline: "Aufräumarbeiten im Hintergrund.",
			Intro:    "Nicht mehr genutzter Code für den alten Foto-Upload an Anliegen wurde entfernt.",
			Items:    []NoteItem{{Label: "Unverändert", Text: "Bestehende Fotos bleiben unverändert abrufbar."}},
		},
		{
			Version:  "0.17.3",
			Date:     "24. Juli 2026",
			Kind:     "Zuverlässigkeit",
			Headline: "Einladungen werden vollständig oder gar nicht gespeichert.",
			Intro:    "Beim Einladen entstehen Person und Haus-Zugehörigkeit jetzt gemeinsam in einem Vorgang.",
			Items:    []NoteItem{{Label: "Keine Reste", Text: "Bricht der Vorgang ab, bleibt kein halb angelegter Zugang zurück."}},
		},
		{
			Version:  "0.17.2",
			Date:     "24. Juli 2026",
			Kind:     "Wartung",
			Headline: "Umstellung der Datenspeicherung abgeschlossen.",
			Intro:    "Die Anwendung arbeitet jetzt ausschließlich mit der eingebetteten Datenbank; die alte Dateispeicherung wird nicht mehr als Ausweichweg genutzt.",
			Items:    []NoteItem{{Label: "Unverändert", Text: "Für Nutzer ändert sich nichts. Die bisherigen Dateien bleiben als Sicherung erhalten."}},
		},
		{
			Version:  "0.17.1",
			Date:     "24. Juli 2026",
			Kind:     "Verbessert",
			Headline: "Verzeichnis-Sichtbarkeit je Haus, klarere Zugangsbearbeitung.",
			Intro:    "Die Anzeige im Kontakte-Verzeichnis lässt sich künftig je Haus festlegen, und die Zugangsbearbeitung zeigt nur noch, was hier auch änderbar ist.",
			Items: []NoteItem{
				{Label: "Je Haus", Text: "Wer zu mehreren Häusern gehört, entscheidet die Sichtbarkeit pro Haus. Ohne eigene Wahl bleibt alles wie bisher."},
				{Label: "Klarer", Text: "Name und E-Mail sind in der Hausverwaltung nur noch lesbar, mit Hinweis auf die zentrale Pflege."},
			},
		},
		{
			Version:  "0.17.0",
			Date:     "24. Juli 2026",
			Kind:     "Sicherheit",
			Headline: "Personen und Häuser sauber getrennt.",
			Intro:    "Eine Person kann zu mehreren Häusern gehören. Eine Hausverwaltung verwaltet ab sofort ausschließlich die Zugehörigkeit zum eigenen Haus.",
			Items: []NoteItem{
				{Label: "Getrennt", Text: "Rolle und Rechte gelten je Haus; eine Änderung wirkt nicht mehr in anderen Häusern."},
				{Label: "Entfernen", Text: "Entfernen löst nur die Zugehörigkeit zum eigenen Haus, nicht die Person."},
				{Label: "Identität", Text: "E-Mail, Titel und Name ändert nur die Plattform-Administration."},
			},
		},
		{
			Version:  "0.16.1",
			Date:     "24. Juli 2026",
			Kind:     "Wartung",
			Headline: "Vorbereitung: Personen und Häuser sauber getrennt.",
			Intro:    "Im Hintergrund entsteht ein Datenmodell, in dem eine Person zu mehreren Häusern gehören kann, ohne dass sich die Häuser gegenseitig beeinflussen.",
			Items:    []NoteItem{{Label: "Unverändert", Text: "Anmeldung, Rollen und Rechte funktionieren unverändert; es wird noch nichts daraus gelesen."}},
		},
		{
			Version:  "0.16.0",
			Date:     "24. Juli 2026",
			Kind:     "Wartung",
			Headline: "Umstellung auf die neue Datenspeicherung abgeschlossen.",
			Intro:    "Anliegen, Abstimmungen, Wohneinheiten und der Telegram-Bot nutzen jetzt ebenfalls die eingebettete Datenbank. Damit ist die schrittweise Umstellung abgeschlossen.",
			Items: []NoteItem{
				{Label: "Unverändert", Text: "Bestehende Daten wurden verlustfrei übernommen; für Nutzer ändert sich nichts."},
				{Label: "Robuster", Text: "Zusammengehörige Änderungen werden jetzt gemeinsam gespeichert."},
			},
		},
		{
			Version:  "0.15.12",
			Date:     "24. Juli 2026",
			Kind:     "Wartung",
			Headline: "Anhänge auf die neue Datenspeicherung umgestellt.",
			Intro:    "Anhänge (Fotos und Dateien an Anliegen, Aushängen und Übergaben) nutzen jetzt die eingebettete Datenbank; die Dateien liegen unverändert im Dateispeicher.",
			Items:    []NoteItem{{Label: "Unverändert", Text: "Für Nutzer ändert sich nichts."}},
		},
		{
			Version:  "0.15.11",
			Date:     "24. Juli 2026",
			Kind:     "Zuverlässigkeit",
			Headline: "Übergabeprotokolle werden zuverlässig abgelegt.",
			Intro:    "Das Ablegen eines Übergabeprotokolls im Dokumentenbereich passiert jetzt in einem Zug: Dokument und Verknüpfung werden gemeinsam gespeichert.",
			Items: []NoteItem{
				{Label: "Keine Doppel", Text: "Ein wiederholtes Ablegen erzeugt kein zweites Protokoll mehr."},
				{Label: "Keine Reste", Text: "Bricht der Vorgang ab, bleibt keine verwaiste Datei zurück."},
			},
		},
		{
			Version:  "0.15.10",
			Date:     "24. Juli 2026",
			Kind:     "Wartung",
			Headline: "Dokumente auf die neue Datenspeicherung umgestellt.",
			Intro:    "Die Dokumentenverwaltung nutzt jetzt die eingebettete Datenbank; Dateien liegen unverändert im Dateispeicher.",
			Items:    []NoteItem{{Label: "Unverändert", Text: "Für Nutzer ändert sich nichts."}},
		},
		{
			Version:  "0.15.9",
			Date:     "24. Juli 2026",
			Kind:     "Wartung",
			Headline: "Wohnungsübergaben auf die neue Datenspeicherung umgestellt.",
			Intro:    "Übergabeprotokolle (Ein-/Auszug) laufen jetzt über die eingebettete Datenbank — verlustfrei übernommen, mit Rückfallebene.",
			Items:    []NoteItem{{Label: "Unverändert", Text: "Für Nutzer ändert sich nichts."}},
		},
		{
			Version:  "0.15.8",
			Date:     "23. Juli 2026",
			Kind:     "Wartung",
			Headline: "Termine auf die neue Datenspeicherung umgestellt.",
			Intro:    "Termine und Kalender laufen jetzt über die eingebettete Datenbank — verlustfrei übernommen, mit Rückfallebene.",
			Items:    []NoteItem{{Label: "Unverändert", Text: "Für Nutzer ändert sich nichts."}},
		},
		{
			Version:  "0.15.7",
			Date:     "23. Juli 2026",
			Kind:     "Wartung",
			Headline: "Aushänge auf die neue Datenspeicherung umgestellt.",
			Intro:    "Aushänge und Hausjournal laufen jetzt über die eingebettete Datenbank — verlustfrei übernommen, mit Rückfallebene.",
			Items:    []NoteItem{{Label: "Unverändert", Text: "Für Nutzer ändert sich nichts."}},
		},
		{
			Version:  "0.15.6",
			Date:     "23. Juli 2026",
			Kind:     "Wartung",
			Headline: "Weitere Bereiche auf die neue Datenspeicherung umgestellt.",
			Intro:    "Adressbuch/Kontakte und der Gelesen-Status von Aushängen laufen jetzt über die eingebettete Datenbank — verlustfrei übernommen, mit Rückfallebene.",
			Items: []NoteItem{
				{Label: "Unverändert", Text: "Für Nutzer ändert sich nichts."},
			},
		},
		{
			Version:  "0.15.5",
			Date:     "23. Juli 2026",
			Kind:     "Wartung",
			Headline: "Zahlungsstatus auf die neue Datenspeicherung umgestellt.",
			Intro:    "Der manuelle Zahlungsstatus je Einheit läuft jetzt über die eingebettete Datenbank — verlustfrei übernommen, mit Rückfallebene.",
			Items: []NoteItem{
				{Label: "Unverändert", Text: "Für Nutzer ändert sich nichts."},
			},
		},
		{
			Version:  "0.15.4",
			Date:     "23. Juli 2026",
			Kind:     "Wartung",
			Headline: "Benachrichtigungs-Einstellungen auf die neue Datenspeicherung umgestellt.",
			Intro:    "Die E-Mail-Benachrichtigungs-Einstellungen laufen jetzt ebenfalls über die eingebettete Datenbank — mit Rückfallebene und ohne Änderung für Nutzer.",
			Items: []NoteItem{
				{Label: "Unverändert", Text: "Bestehende Einstellungen wurden verlustfrei übernommen."},
			},
		},
		{
			Version:  "0.15.3",
			Date:     "23. Juli 2026",
			Kind:     "Wartung",
			Headline: "Erste Bereiche auf die neue Datenspeicherung umgestellt.",
			Intro:    "Anmelde-Verlauf und Profilangaben laufen jetzt über die neue, transaktionssichere Datenhaltung — mit automatischer Rückfallebene, falls nötig.",
			Items: []NoteItem{
				{Label: "Unverändert", Text: "Für Nutzer ändert sich nichts; bestehende Angaben wurden übernommen."},
			},
		},
		{
			Version:  "0.15.2",
			Date:     "23. Juli 2026",
			Kind:     "Wartung",
			Headline: "Grundlage für robustere Datenspeicherung.",
			Intro:    "Im Hintergrund entsteht eine transaktionssichere Datenhaltung; die Umstellung erfolgt schrittweise und für Nutzer unbemerkt.",
			Items: []NoteItem{
				{Label: "Unverändert", Text: "Bestehende Daten und Abläufe bleiben gleich; die Umstellung passiert im Hintergrund."},
			},
		},
		{
			Version:  "0.15.1",
			Date:     "23. Juli 2026",
			Kind:     "Sicherheit",
			Headline: "Zugriffsprüfungen zentral abgesichert.",
			Intro:    "Die Rechteprüfung für geschützte Verwaltungsbereiche läuft jetzt an einer zentralen Stelle statt in jeder Seite einzeln.",
			Items: []NoteItem{
				{Label: "Robustheit", Text: "Neue geschützte Seiten erben die Rechteprüfung automatisch; sie kann nicht mehr versehentlich ausgelassen werden."},
				{Label: "Abgesichert", Text: "Eine Rechte-Matrix im Test weist unberechtigte Zugriffe nachweislich ab."},
			},
		},
		{
			Version:  "0.15.0",
			Date:     "23. Juli 2026",
			Kind:     "Parkplatz",
			Headline: "Monatsabrechnung exportieren und drucken.",
			Intro:    "Jeder Parkplatzmonat lässt sich als CSV mit Stundendetails exportieren und über eine aufgeräumte Ansicht drucken oder als PDF speichern.",
			Items: []NoteItem{
				{Label: "CSV", Text: "Export mit Stundenwerten (Verbrauch, Preis, Kosten); die Summen entsprechen genau der Bildschirmansicht."},
				{Label: "Druck", Text: "Eine aufgeräumte Druckansicht eignet sich zum privaten Teilen oder als PDF."},
				{Label: "Datenqualität", Text: "Fehlen für einen Monat Messpunkte, wird vor dem Export deutlich darauf hingewiesen."},
			},
		},
		{
			Version:  "0.14.0",
			Date:     "23. Juli 2026",
			Kind:     "Verwaltung",
			Headline: "Zugänge vorübergehend sperren.",
			Intro:    "Ein Zugang lässt sich jetzt deaktivieren, ohne ihn zu löschen — die Anmeldung ist gesperrt, der Eintrag bleibt erhalten.",
			Items: []NoteItem{
				{Label: "Reversibel", Text: "Deaktivierte Zugänge lassen sich jederzeit wieder aktivieren; Rechte und Verlauf bleiben erhalten."},
				{Label: "Sicherheit", Text: "Man kann sich nicht selbst sperren, und der Notfall-Admin bleibt jederzeit anmeldefähig."},
			},
		},
		{
			Version:  "0.13.0",
			Date:     "23. Juli 2026",
			Kind:     "Verwaltung",
			Headline: "Benutzer & Rechte direkt in der App.",
			Intro:    "Zugänge, die bisher fest in der Konfiguration hinterlegt waren, lassen sich jetzt im Portal bearbeiten — ohne neue Auslieferung.",
			Items: []NoteItem{
				{Label: "Bearbeiten", Text: "Rolle, Sonderrechte und Anmeldeweg eines Zugangs wirken sofort."},
				{Label: "Anmeldung", Text: "Pro Person wählbar: Anmeldung per E-Mail-Link, über Zitadel-SSO oder auf beiden Wegen."},
				{Label: "Sicherheit", Text: "Die letzte Administrator-Rolle bleibt geschützt; ein Aussperren der eigenen Verwaltung ist ausgeschlossen."},
			},
		},
		{
			Version:  "0.12.2",
			Date:     "19. Juli 2026",
			Kind:     "Zahlungen",
			Headline: "Zahlungen klarer erfasst.",
			Intro:    "Der Zahlungsbereich im Monatsdetail ist sauber abgesetzt und hält optional Datum, Art und Referenz der Zahlung fest.",
			Items: []NoteItem{
				{Label: "Referenz", Text: "Details einer Zahlung lassen sich auch nachträglich zu markierten Monaten ergänzen."},
				{Label: "Rollen", Text: "Bewohner sehen weiterhin nur den Status der Zahlung — Details bleiben der Verwaltung vorbehalten."},
			},
		},
		{
			Version:  "0.12.1",
			Date:     "19. Juli 2026",
			Kind:     "Oberfläche",
			Headline: "Aufgeräumte Parkplatz-Einstellungen.",
			Intro:    "Tarif, Laderegelung, Regler-Status und Telegram sind jetzt kompakt und klar gruppiert.",
			Items: []NoteItem{
				{Label: "Formulare", Text: "Eingabefelder zeigen ihre Einheit direkt im Feld; zusammengehörige Werte stehen nebeneinander."},
				{Label: "Schalter", Text: "Regler und Testbetrieb sind als klare Auswahlkarten bedienbar."},
			},
		},
		{
			Version:  "0.12.0",
			Date:     "19. Juli 2026",
			Kind:     "Laden",
			Headline: "PV-Überschussladen für Parkplatz 20.",
			Intro:    "Die Plattform steuert das Laden jetzt selbst nach Sonnenstrom und rechnet Überschuss-Strom zum Fixpreis ab. Der Start erfolgt im Beobachtungsbetrieb.",
			Items: []NoteItem{
				{Label: "Automatik", Text: "Ist der Hausakku voll und wird eingespeist, startet die Ladung automatisch; Überschuss-Strom kostet fix 0,10 € je kWh."},
				{Label: "Live-Ansicht", Text: "Die neue „Jetzt“-Karte zeigt Ladequelle, Hausakku, Einspeisung und die Aufteilung Überschuss/Normal — mit Ein/Aus-Schalter."},
				{Label: "Telegram", Text: "Status und Ein/Aus laufen über einen eigenen Bot; Benachrichtigungen kommen nur bei echten Änderungen."},
				{Label: "Abrechnung", Text: "Monate, Stundenwerte und CSV-Export weisen Überschuss- und Normalanteile getrennt aus; bestehende Monate bleiben unverändert."},
			},
		},
		{
			Version:  "0.11.0",
			Date:     "18. Juli 2026",
			Kind:     "Faire Nutzung",
			Headline: "Mehr Spielraum für kleine Hausgemeinschaften.",
			Intro:    "Der kostenfreie Rahmen und die Zählweise für Wohneinheiten sind jetzt klar und nachvollziehbar beschrieben.",
			Items: []NoteItem{
				{Label: "Fair Use", Text: "Bis zu 25 Wohneinheiten können den kostenfreien Rahmen nutzen."},
				{Label: "Zählweise", Text: "Wohnungen und vergleichbare Nutzungseinheiten zählen vollständig; Zubehör wie Keller und Stellplätze nicht automatisch."},
				{Label: "Übersicht", Text: "Dokumente nutzen auf der Startseite die volle Breite; längere Texte brechen in Karten ruhiger um."},
				{Label: "Datenschutz", Text: "Die Dienstleister-Koordination wird bis zur dokumentierten Betreiberfreigabe als technisch vorbereitet statt als freigegebener Pilot ausgewiesen."},
				{Label: "Sicherheit", Text: "Die Anwendung wird mit Go 1.26.5 und den aktuellen TLS-Sicherheitskorrekturen gebaut."},
			},
		},
		{
			Version:  "0.10.0",
			Date:     "14. Juli 2026",
			Kind:     "Dienstleister",
			Headline: "Handwerker-Koordination technisch vorbereitet.",
			Intro:    "Der eingeschränkte Arbeitsablauf ist umgesetzt; echte Einladungen bleiben aus, bis eine externe fachkundige Person die Datenschutz- und Rechtsfragen geprüft und die Freigabe dokumentiert hat.",
			Items: []NoteItem{
				{Label: "Ablauf", Text: "Annehmen, Termin vereinbaren, bearbeiten und erledigen folgen einem nachvollziehbaren Statusweg."},
				{Label: "Fotos", Text: "Fotos lassen sich auch ohne zusätzlichen Text direkt am Anliegen ergänzen."},
				{Label: "Nachvollziehbarkeit", Text: "Aktionen externer Dienstleister erscheinen vollständig im Aktivitätsprotokoll der Verwaltung."},
				{Label: "Freigabe", Text: "Vor der Aktivierung bleiben Rechts-, Datenschutz- und Aufbewahrungsfragen zu bestätigen."},
			},
		},
		{
			Version:  "0.9.0",
			Date:     "13. Juli 2026",
			Kind:     "Sicherheit",
			Headline: "Zugriffsschutz zentral abgesichert.",
			Intro:    "Anmeldung, Hauszuordnung und Formularschutz werden jetzt an einer gemeinsamen Stelle zuverlässig geprüft.",
			Items: []NoteItem{
				{Label: "Zugriff", Text: "Neue geschützte Seiten übernehmen automatisch dieselben Anmelde- und Hausprüfungen."},
				{Label: "Formulare", Text: "Absendeaktionen sind einheitlich gegen missbräuchliche Fremdaufrufe abgesichert."},
			},
		},
		{
			Version:  "0.8.0",
			Date:     "13. Juli 2026",
			Kind:     "Stabilität",
			Headline: "Sicherheit und Datenhaltbarkeit gestärkt.",
			Intro:    "Hausgrenzen, gespeicherte Daten und der Betrieb reagieren robuster auf Fehler und unerwartete Unterbrechungen.",
			Items: []NoteItem{
				{Label: "Datenschutz", Text: "Zugriffe zwischen Häusern werden strenger geprüft und Formulare zusätzlich geschützt."},
				{Label: "Daten", Text: "Gespeicherte Änderungen werden zuverlässiger auf die Festplatte geschrieben."},
				{Label: "Betrieb", Text: "Unerwartete Fehler führen nicht mehr zum Abbruch; Aktualisierungen fahren sauber herunter."},
			},
		},
		{
			Version:  "0.7.0",
			Date:     "13. Juli 2026",
			Kind:     "Fundament",
			Headline: "Fundament für sichere Weiterentwicklung gelegt.",
			Intro:    "Die Anwendung ist intern klarer gegliedert und lässt sich dadurch gezielter prüfen und erweitern.",
			Items: []NoteItem{
				{Label: "Bausteine", Text: "Funktionen sind in klar getrennte technische Bereiche gegliedert."},
				{Label: "Qualität", Text: "Automatische und visuelle Vergleiche helfen, unbeabsichtigte Änderungen früh zu erkennen."},
			},
		},
		{
			Version:  "0.6.13",
			Date:     "9. Juli 2026",
			Kind:     "Produktpflege",
			Headline: "Formulare bleiben auf kleinen Bildschirmen handhabbar.",
			Intro:    "Längere Dialoge führen jetzt klarer zum Speichern, ohne dass der Abschluss aus dem Blick rutscht.",
			Items: []NoteItem{
				{Label: "Dialoge", Text: "Formularinhalte scrollen innerhalb des Fensters; der wichtige Abschluss bleibt erreichbar."},
				{Label: "Mobil", Text: "Auch umfangreiche Eingaben wie Abstimmungen behalten eine sichtbare Aktion."},
			},
		},
		{
			Version:  "0.6.12",
			Date:     "9. Juli 2026",
			Kind:     "Produktpflege",
			Headline: "Die Seitenleiste bleibt aufgeräumt.",
			Intro:    "Auch lange Namen und alle freigeschalteten Bereiche passen jetzt sauber in die Portalnavigation.",
			Items: []NoteItem{
				{Label: "Navigation", Text: "Menüpunkte behalten verlässlichen Platz und bleiben vollständig erreichbar."},
				{Label: "Profilbereich", Text: "Version, Rolle und Abmeldung bleiben sichtbar, ohne den unteren Rand zu berühren."},
			},
		},
		{
			Version:  "0.6.11",
			Date:     "9. Juli 2026",
			Kind:     "Produktpflege",
			Headline: "Der Audit-Log liest sich wie eine klare Historie.",
			Intro:    "Sensible Aktionen erscheinen jetzt als ruhige Zeitlinie mit Überblick und Details bei Bedarf.",
			Items: []NoteItem{
				{Label: "Überblick", Text: "Ereignisse, beteiligte Personen und heutige Aktionen stehen kompakt am Anfang."},
				{Label: "Historie", Text: "Einträge sind nach Tagen gruppiert; Details öffnen erst, wenn sie gebraucht werden."},
			},
		},
		{
			Version:  "0.6.10",
			Date:     "9. Juli 2026",
			Kind:     "Produktpflege",
			Headline: "Anliegen sind klarer geführt.",
			Intro:    "Melden, eigene Anliegen und Verwaltung sind jetzt deutlich getrennt, damit die Seite nicht mehr wie ein langer Formularstapel wirkt.",
			Items: []NoteItem{
				{Label: "Anliegen", Text: "Die Übersicht zeigt Statuskarten, einen kompakten Meldebereich und eine kurze Verwaltungsvorschau."},
				{Label: "Triage", Text: "Das Board bleibt vollständig, zeigt Bearbeitung aber erst bei Bedarf pro Anliegen."},
			},
		},
		{
			Version:  "0.6.9",
			Date:     "9. Juli 2026",
			Kind:     "Produktpflege",
			Headline: "Das Portal startet mobil schneller beim Inhalt.",
			Intro:    "Die Navigation bleibt vollständig erreichbar, nimmt auf dem Handy aber nicht mehr den ersten Bildschirm ein.",
			Items: []NoteItem{
				{Label: "Mobil", Text: "Ein kompakter Kopfbereich zeigt Haus, Adresse und Menü, damit die eigentliche Aufgabe sofort sichtbar wird."},
				{Label: "Navigation", Text: "Das Menü öffnet bei Bedarf die bekannten Bereiche, Benutzerinfo, Version und Abmeldung."},
			},
		},
		{
			Version:  "0.6.8",
			Date:     "9. Juli 2026",
			Kind:     "Produktpflege",
			Headline: "Dokumente sind ruhiger und schneller erfassbar.",
			Intro:    "Das Archiv zeigt Dateien kompakter, klarer und ohne harte Umbrüche in langen Dateinamen.",
			Items: []NoteItem{
				{Label: "Dokumente", Text: "Titel, Sichtbarkeit, Version und Dateiname stehen jetzt in einer aufgeräumten Zeile."},
				{Label: "Mobil", Text: "Download, Vorschau und Ersetzen bleiben erreichbar, ohne den Inhalt zusammenzudrücken."},
			},
		},
		{
			Version:  "0.6.7",
			Date:     "9. Juli 2026",
			Kind:     "Produktpflege",
			Headline: "Der Kostenblock bricht mobil sauber um.",
			Intro:    "Die Preiszeile bleibt kompakt und benennt die Wohneinheit im erklärenden Text.",
			Items: []NoteItem{
				{Label: "Preise", Text: "1 € pro Monat bleibt als klare Zeile sichtbar, je Wohneinheit erklärt im Begleittext."},
				{Label: "Mobil", Text: "Der Bereich vermeidet seitliches Scrollen auf kleinen Bildschirmen."},
			},
		},
		{
			Version:  "0.6.6",
			Date:     "9. Juli 2026",
			Kind:     "Produktpflege",
			Headline: "Faire Kosten sind präziser formuliert.",
			Intro:    "Die Startseite benennt die Preislogik kürzer und klarer, besonders auf kleinen Bildschirmen.",
			Items: []NoteItem{
				{Label: "Preise", Text: "Der Richtwert wurde kompakter auf die Wohneinheit bezogen."},
				{Label: "Mobil", Text: "Der Kostenblock bricht ruhiger um und bleibt leichter scanbar."},
			},
		},
		{
			Version:  "0.6.5",
			Date:     "9. Juli 2026",
			Kind:     "Produktpflege",
			Headline: "Benutzer & Rechte lesen sich mobil ruhiger.",
			Intro:    "Zugänge, Rollen und Sonderrechte bleiben auf kleinen Bildschirmen klar gegliedert.",
			Items: []NoteItem{
				{Label: "Mobil", Text: "Rollenhinweise und Rechte erscheinen als lesbare Chips, ohne unschöne Worttrennungen."},
				{Label: "Übersicht", Text: "Personenkarten haben mehr Luft, klare Abschnitte und stabile Aktionsflächen."},
			},
		},
		{
			Version:  "0.6.4",
			Date:     "9. Juli 2026",
			Kind:     "Produktpflege",
			Headline: "Verwaltungslisten sitzen besser auf dem Handy.",
			Intro:    "Aushänge und Termine lassen sich mobil ruhiger bearbeiten, ohne dass Aktionen aus der Karte rutschen.",
			Items: []NoteItem{
				{Label: "Mobil", Text: "Bearbeiten und Löschen bleiben in Aushang- und Terminlisten sichtbar im jeweiligen Eintrag."},
				{Label: "Bedienung", Text: "Die Aktionsflächen haben mehr verlässlichen Platz und lassen sich besser treffen."},
			},
		},
		{
			Version:  "0.6.3",
			Date:     "9. Juli 2026",
			Kind:     "Produktpflege",
			Headline: "Parkplatznutzung führt klarer in den Start.",
			Intro:    "Wenn noch keine Monatswerte vorhanden sind, zeigt die Seite jetzt die nächsten sinnvollen Schritte statt einer leeren Fläche.",
			Items: []NoteItem{
				{Label: "Einstieg", Text: "Konfiguration, Zugriff und Monatswerte sind als ruhiger Ablauf sichtbar."},
				{Label: "Rollen", Text: "Verwaltung und Bewohner sehen jeweils nur die passenden Aktionen."},
				{Label: "Klarheit", Text: "Die Seite bleibt bewusst bei Transparenz und privater Stellplatznutzung, ohne Buchhaltung oder Mahnwesen zu versprechen."},
			},
		},
		{
			Version:  "0.6.2",
			Date:     "9. Juli 2026",
			Kind:     "Produktpflege",
			Headline: "Audit-Log auf kleinen Bildschirmen klarer.",
			Intro:    "Sensible Aktionen bleiben unterwegs besser lesbar und passen sauber in den verfügbaren Platz.",
			Items: []NoteItem{
				{Label: "Mobil", Text: "Filter und Audit-Einträge stapeln sich jetzt ohne seitliches Scrollen."},
				{Label: "Lesbarkeit", Text: "Zeitpunkt, Aktion, Person, Ziel und Details wirken wie ruhige Karten statt wie eine gedrückte Tabelle."},
			},
		},
		{
			Version:  "0.6.1",
			Date:     "8. Juli 2026",
			Kind:     "Schnittstellen",
			Headline: "BMD-Prüfung konkreter vorbereitet.",
			Intro:    "Für die Abstimmung mit dem Steuerberater gibt es jetzt ein klares Rohdaten-Beispiel, ohne Buchhaltung ins Portal zu ziehen.",
			Items: []NoteItem{
				{Label: "BMD", Text: "Ein internes Kandidatenformat beschreibt Haus, Einheit, Zeitraum, Referenz, Betrag und Status als Rohdaten."},
				{Label: "Prüfung", Text: "Golden File und Checkliste machen den nächsten BMD-NTCS-Testimport nachvollziehbar."},
				{Label: "Grenzen", Text: "Konten- und Steuerfelder bleiben gesperrt, bis ein reales Mapping bestätigt ist."},
			},
		},
		{
			Version:  "0.6.0",
			Date:     "8. Juli 2026",
			Kind:     "Schnittstellen",
			Headline: "E-Rechnungen besser vorbereitet.",
			Intro:    "hausv.org kann ebInterface-Rechnungen als geschützte Unterlagen einordnen, ohne daraus Buchhaltung zu machen.",
			Items: []NoteItem{
				{Label: "ebInterface", Text: "Rechnungen in den Profilen 5.0 und 6.0 werden als Metadaten gelesen und als geschütztes XML-Dokument abgelegt."},
				{Label: "Dokumente", Text: "XML-Dateien können im Dokumentenbereich sicher gespeichert werden."},
				{Label: "Grenzen", Text: "Rechnungen bleiben Empfang und Ablage: keine Buchung, keine Steuerlogik und kein Zahlungsauftrag."},
				{Label: "Qualität", Text: "Golden Files und Fehlerberichte sichern die unterstützten Profile ab."},
			},
		},
		{
			Version:  "0.5.0",
			Date:     "8. Juli 2026",
			Kind:     "Klarheit",
			Headline: "Positionierung und Schnittstellen klarer.",
			Intro:    "Die Startseite erklärt jetzt noch genauer, wofür hausv.org steht: Kommunikation, Transparenz und Anschlussfähigkeit ohne eigene Buchhaltung.",
			Items: []NoteItem{
				{Label: "Startseite", Text: "Funktionen, Pilotbausteine und Ausblick sind sauberer getrennt und leichter einzuordnen."},
				{Label: "Faire Kosten", Text: "Die Preislogik spricht konsequent von Wohneinheiten und erklärt Zubehör wie Keller oder Stellplätze transparenter."},
				{Label: "Österreich", Text: "camt.053 und camt.054 sind als Zahlungsstatusquellen eingeordnet; BMD/RZL und ebInterface bleiben Übergaben an bestehende Systeme."},
				{Label: "Qualität", Text: "Schnittstellen erhalten klare Gates für Profile, Golden Files, Feldlimits und spätere XSD-Prüfung."},
			},
		},
		{
			Version:  "0.4.0",
			Date:     "8. Juli 2026",
			Kind:     "Neue Verwaltung",
			Headline: "Übergaben sauber im Griff.",
			Intro:    "Nutzerwechsel lassen sich jetzt strukturiert erfassen, bestätigen und sicher ablegen.",
			Items: []NoteItem{
				{Label: "Übergaben", Text: "Räume, Zähler, Schlüssel, Fotos und Notizen werden in einem klaren Protokoll gesammelt."},
				{Label: "Bestätigung", Text: "Ausziehende und einziehende Personen können vorbereitete Protokolle per begrenztem Link bestätigen."},
				{Label: "Dokumente", Text: "Protokolle lassen sich als PDF exportieren und direkt zur passenden Einheit ablegen."},
				{Label: "Roadmap", Text: "Dienstleisterzugang, Bankdateien und Zahlungsaufträge sind für die österreichische Roadmap sauber abgegrenzt."},
			},
		},
		{
			Version:  "0.3.0",
			Date:     "8. Juli 2026",
			Kind:     "Verwaltung",
			Headline: "Besser steuern, gezielter teilen.",
			Intro:    "Dienstleister, Kontakte, Kalender und Zahlungsstatus sind klarer in den Hausalltag eingebunden.",
			Items: []NoteItem{
				{Label: "Dienstleister", Text: "Externe Helfer sehen nur zugewiesene Anliegen und können dort Rückfragen oder Terminvorschläge ergänzen."},
				{Label: "Adressbuch", Text: "Hausmeister, Notdienste und wiederkehrende Firmen lassen sich pro Haus gepflegt hinterlegen."},
				{Label: "Kalender", Text: "Termine und passende Dienstleister-Zeitfenster können als geschützter Kalenderfeed abonniert werden."},
				{Label: "Zahlungsstatus", Text: "Einheiten können transparent als offen, bezahlt, teilbezahlt oder überfällig markiert werden."},
			},
		},
		{
			Version:  "0.2.0",
			Date:     "8. Juli 2026",
			Kind:     "Produktpflege",
			Headline: "Mehr Ruhe, mehr Überblick.",
			Intro:    "Die Oberfläche führt jetzt klarer durch Hausalltag, Anhänge und neue Informationen.",
			Items: []NoteItem{
				{Label: "Startseite", Text: "Vertrauen, Datenschutz und faire Kosten sind verständlicher getrennt und leichter zu erfassen."},
				{Label: "Faire Nutzung", Text: "Kostenfreie Nutzung und spätere Richtpreise orientieren sich an Wohneinheiten statt an Hausadressen."},
				{Label: "Anhänge", Text: "Ausgewählte Dateien zeigen Vorschau, Größe und Entfernen vor dem Speichern."},
				{Label: "Verwaltung", Text: "Audit-Log, Rollenhilfe und Abstimmungen sind besser lesbar und geben mehr Rückmeldung."},
				{Label: "Frische Updates", Text: "Neue Versionen laden ihre aktuellen Skripte zuverlässig nach dem Deployment."},
			},
		},
		{
			Version:  "0.1.0",
			Date:     "7. Juli 2026",
			Kind:     "Erster Pilot",
			Headline: "Das Hausportal geht an den Start.",
			Intro:    "Die ersten zentralen Wege für eine digitale Hausgemeinschaft sind verfügbar.",
			Items: []NoteItem{
				{Label: "Hausüberblick", Text: "Aushänge, Termine, Anliegen, Dokumente und Parkplatznutzung laufen in einem privaten Portal zusammen."},
				{Label: "Rollen", Text: "Eigentümer, Mieter, Beirat und Verwaltung erhalten getrennte Sichtbarkeit."},
				{Label: "Sicherheit", Text: "Zugang, Anhänge und sensible Inhalte bleiben auf geschützte App-Routen begrenzt."},
			},
		},
	}
}
