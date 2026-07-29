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
				{Label: "Piloten absichern", Text: "Wohnung, Eltern-Haus ohne Speicher und Schwiegereltern-Haus mit Speicher bleiben getrennt und starten ausschließlich mit „Nur beobachten“."},
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
				{Label: "Vollständiger Datenfluss", Text: "Netcup, Cloudflare, Hetzner, das selbst betriebene Zitadel und Resend sind ihrem tatsächlichen Zweck entsprechend erklärt."},
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
