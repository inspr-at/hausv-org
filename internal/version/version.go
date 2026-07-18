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
	//   -X github.com/markus-barta/hausv-org/internal/version.Version=1.2.3
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
			Version:  "0.11.0",
			Date:     "18. Juli 2026",
			Kind:     "Faire Nutzung",
			Headline: "Mehr Spielraum für kleine Hausgemeinschaften.",
			Intro:    "Der kostenfreie Rahmen und die Zählweise für Wohneinheiten sind jetzt klar und nachvollziehbar beschrieben.",
			Items: []NoteItem{
				{Label: "Fair Use", Text: "Bis zu 25 Wohneinheiten können den kostenfreien Rahmen nutzen."},
				{Label: "Zählweise", Text: "Wohnungen und vergleichbare Nutzungseinheiten zählen vollständig; Zubehör wie Keller und Stellplätze nicht automatisch."},
				{Label: "Übersicht", Text: "Dokumente nutzen auf der Startseite die volle Breite; längere Texte brechen in Karten ruhiger um."},
				{Label: "Datenschutz", Text: "Die Dienstleister-Koordination wird bis zur externen Prüfung als technisch vorbereitet statt als freigegebener Pilot ausgewiesen."},
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
