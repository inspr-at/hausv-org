# Jahresabrechnung: implementierte Berechnung

Technische Spezifikation zu HAUSV-767, Berechnungsversion `2`. Sie beschreibt
das Verhalten im Code, keine rechtliche Freigabe für eine konkrete Abrechnung.
Unfreigegebene PDFs tragen den Hinweis „Entwurf zur Prüfung — keine Rechtsauskunft
nach WEG/MRG“. Die einmalige Freigabe speichert Datum, Person und Rolle separat
vom unveränderlichen Lauf. Freigegebene PDFs tragen Datum und Rolle statt
„Entwurf“. Archiv und Versand verlangen die Freigabe. Bereits archivierte
Altentwürfe benötigen einen neuen Lauf; ihre Dateien bleiben unverändert.
Die Freigabe wird als `annual-statement.run.approve` protokolliert.

## Ablauf und Eingaben

`internal/server/annual_statement.go` lädt die gewählte Periode, ihre Struktur,
aktuelle Einheiten/Parteien, bestätigte Belege, Akonto und Verbrauchsnachweise.
Die Arbeitsvorschau zeigt die bisher erfassten umlagefähigen Belege; ein
gespeicherter Lauf verlangt zusätzlich vollständige Eingaben für alle Einheiten.
`AnnualStatementRunRepository.Create` liest diese erneut und ruft
`CalculateAnnualStatementRun` auf. Nur ein vollständig gültiges Ergebnis wird
als neue Revision gespeichert.

Eine Periode hat einen Jahresschlüssel sowie inklusive Anfangs-/Enddaten.
Gültige Datumswerte und Ende ≥ Beginn werden geprüft. Der Jahresschlüssel wird
nicht gegen das Kalenderjahr der Datumswerte geprüft; periodenübergreifende
Datumsbereiche sind technisch möglich. Belege werden ausdrücklich einer Periode
zugeordnet. Das gültige Rechnungsdatum muss nicht innerhalb ihres Zeitraums liegen.

Kostenarten, Umlagefähigkeit, Verteilerschlüssel und Einheitenbasen werden pro
Periode gespeichert und können dort bearbeitet werden. Das Folgejahr übernimmt
nur diese Struktur, keine Belege, Vorauszahlungen oder Messwerte. Einheiten und
Parteien kommen beim neuen Lauf aus dem aktuellen Register. Fehlt danach eine
Einheit in der Periodenstruktur oder steht dort eine inzwischen entfernte
Einheit, wird die Berechnung gesperrt.

## Schlüssel und Anteile

Jede umlagefähige Kostenart hat genau einen dieser Schlüssel:

| Schlüssel | Basis je Einheit | Vollständigkeitsregel |
| --- | --- | --- |
| `nutzwert` | Miteigentumsanteil in Millionsteln (PPM) | Jede Einheit > 0; Haussumme genau 1.000.000. Null bedeutet nicht erfasst. |
| `flaeche` | Nutzfläche in ganzen Hundertstel m² | Explizit erfasst und ≥ 0; Haussumme > 0. |
| `personen` | Ganze Personenzahl | Explizit erfasst und ≥ 0; Haussumme > 0. |
| `verbrauch` | Differenz kumulativer Zählerstände in ganzen Mikroeinheiten | Nur `heizung`/`warmwasser`; vollständiger, einheitlicher Messvektor; Summe > 0. |

Bei Fläche/Personen ist explizite Null gültig und erhält null Anteil. Ein leeres
Feld sperrt den verwendeten Schlüssel. Basis-Summenüberlauf sperrt ebenfalls.
Verbrauchssummen und Multiplikationen bei der Anteilsbildung verwenden `big.Int`.

Der Anteil wird zunächst als `Basis / Summe(Basis) * 1.000.000` berechnet.
Ganzzahlige Abrundung und Vergabe der fehlenden Millionstel nach größtem Rest
ergeben exakt 1.000.000 PPM. Gleiche Reste werden im Lauf und in der monetären
Arbeitsvorschau nach lexikografisch aufsteigender Einheiten-ID aufgelöst
(`top-10` vor `top-2`). Die Anzeige kann Einheiten natürlich sortieren.

## Beträge und Rundung

Geld wird als `int64` in Cent erfasst, summiert, verteilt und formatiert.
Keine Gleitkommaarithmetik liegt im Geldpfad. Gleitkomma wird unter anderem für
Flächen-/Prozentanzeige und PDF-Farben verwendet, nicht für Geldbeträge.

Zuerst werden alle Belege **je Kostenart** summiert. Dann erhält jede Einheit
`Kostenart-Cent * Anteil-PPM / 1.000.000`, ganzzahlig abgerundet. Die noch
fehlenden Cent erhalten die größten Divisionsreste, bei Gleichstand wieder die
frühere Einheiten-ID. Jede Kostenart wird separat gerundet, auch wenn mehrere
denselben Schlüssel verwenden. Dadurch entspricht die Summe der Einheiten je
Kostenart exakt der Belegsumme. Eine Nullbasis erhält auch keinen Restcent.

Die PPM sind eine tatsächliche Zwischenrundung, nicht nur eine Anzeige:
drei gleiche Basen ergeben 333.334 / 333.333 / 333.333 PPM. Eine Verteilung
direkt aus dem ungerundeten Bruch kann deshalb bei großen Beträgen abweichen.
Die Geldberechnung verwendet weiterhin diese bestehende Regel (Version 1).

Ein gespeicherter Lauf verlangt je umlagefähiger Kostenart mindestens einen
bestätigten Beleg. Belege müssen positive Centbeträge und jeweils unterschiedliche
IDs sowie Dokument-IDs besitzen. Originale müssen beim Lauf lesbar, nicht leer
und von der gespeicherten Dateigröße sein; ihre Metadaten müssen auf PDF/Bild
verweisen. Nicht umlagefähige Kosten werden separat summiert und nicht verteilt.
Überläufe der umlagefähigen oder ausgeschlossenen Summe sperren den gesamten
Lauf; es gibt keine Teilbeträge aus einer gesperrten Berechnung.

Akonto ist ein expliziter, nicht negativer Perioden-Gesamtbetrag je Einheit.
Nicht erfasst ist anders als ausdrücklich null. Es gibt keine monatliche
Hochrechnung. `Saldo = zugeteilte Kosten − Akonto`: positiv ist Nachzahlung,
negativ Guthaben, null ausgeglichen. Das Formular akzeptiert ausschließlich
Ziffern mit genau zwei Nachkommastellen und Komma oder Punkt als Dezimalzeichen.
Vorzeichen, einschließlich `-0,50`, werden abgelehnt.

## Zeitanteile, Leerstand und Parteien

Außer dem ausdrücklich hinterlegten Leerstand gibt es keine zeitabhängigen
Basen oder Eigentums-/Mietzeiträume. Ein unterjähriger Personen-, Eigentümer-
oder Mieterwechsel wird derzeit nicht anteilig berechnet.

Ohne hinterlegten Leerstandszeitraum bleibt die Einheit in der Verteilung:
Fläche und Miteigentum laufen weiter, null erfasste Personen ergeben null
Personenkosten, null gemessener Verbrauch ergibt null Verbrauchskosten.
Fehlende Parteien verschieben keine Kosten auf andere Einheiten. Ein Lauf ist
dann berechenbar, aber ein vollständiges PDF-Paket/Archiv benötigt mindestens
eine gespeicherte Partei je Einheit.

Ein Leerstand je Einheit ist ein inklusiver Zeitraum (`vacant_from`/`vacant_to`)
in der Periodenstruktur. Bei `mrg_voll` und `mrg_teil` behält die Einheit ihren
Anteil. Der auf die leeren Tage entfallende Betrag — tageweise, inklusive
Schaltjahr — wird dem Eigentümer als Zeile „Leerstand – Eigentümeranteil“
berechnet und nicht auf die anderen Einheiten umgelegt. Die Haussumme bleibt
100 %. Die Beträge der nicht leeren Einheiten sind dieselben wie ohne
Leerstand; der Restcent des Tagesanteils bleibt bei der Einheit. Bei WEG und
Ausnahme ändert der Zeitraum das Ergebnis nicht, weil dort der Eigentümer
ohnehin die Partei ist beziehungsweise der Vertrag gilt. Gespeicherte Läufe
ohne Leerstandsdaten wiederholen sich unverändert; die Berechnungsversion
bleibt ohne Umsatzsteuerausweis 2, weil sich der Algorithmus für diese Eingaben
nicht ändert.

Ohne Leerstandsanteil erhält jede gespeicherte Eigentümer-/Mietpartei eine
adressierte Kopie des vollständigen Einheitenergebnisses. Bei MRG-Leerstand
erhält die reine Eigentümerpartei den Leerstandsanteil ohne Akonto; die Mietpartei
behält den bewohnten Rest und die erfassten Akontos, auch im HeizKG-Nachweis.
Eine Partei mit beiden Rollen erhält die zusammengeführten Kostenarten und
jedes Heizkosten-Akonto genau einmal. Miteigentümer werden nicht untereinander
aufgeteilt. Deshalb dürfen die Summen aller Parteien-PDFs nicht als Haussumme
addiert werden.

## Verbrauchsnachweise

Die Grenzmessungen liegen exakt auf 00:00 Uhr Europe/Vienna am Periodenbeginn
und am Folgetag nach dem inklusiven Periodenende. Der Folgetag wird mit
Kalenderarithmetik bestimmt; Schalttag sowie 23-/25-Stunden-Tage funktionieren.
Alle Messungen innerhalb dieser Grenzen werden auf Quellenwechsel, wechselnde
Maßeinheiten und rückläufige Zähler geprüft. Fehlende Grenzen, Reset oder
Mehrdeutigkeit sperren die Kostenart; keine Interpolation oder Schätzung.
Alle Einheiten müssen dieselbe unterstützte Maßeinheit verwenden.

Der validierte Vektor ist die Berechnungsgrundlage. Nur die zwei Grenzmessungen
je Einheit werden als Belege zusätzlich im Lauf gespeichert; Zwischenmessungen
bleiben im Messwertspeicher. Ein Replay rekonstruiert daher keinen Vektor allein
aus den Grenzen. Bei aktivem HeizKG verwendet Version 2 die unten beschriebene Mischverteilung.
Version 1 bleibt für historische Wiederholungen unverändert.

## Snapshot, PDF, Archiv und Versand

Ein Lauf enthält Periode, Periodenstruktur, Einheiten-Identität/Typ/Label,
Parteien mit Rollen und gelieferten Anschriften, Belege, Vorauszahlungen,
Metadaten lesbarer Originale, Messvektoren/Grenznachweise sowie Darstellungs- und
Verwaltungskontaktangaben. Hinzu kommen Ergebnis, Berechnungsversion, ID,
Revision, Ersteller, UTC-Zeit und SHA-256 der kanonisch sortierten Eingabe-JSON.
Die Daten werden vor dem Sortieren kopiert, einschließlich verschachtelter
Messvektoren. Deren Zeilenfolge beeinflusst den Hash nicht.

PostgreSQL erstellt den Lauf in einer serialisierbaren Transaktion. Konflikte
verlangen einen neuen Versuch. Der Datenbanktrigger verbietet UPDATE/DELETE
gespeicherter PostgreSQL-Läufe. SQLite hat dieselbe Repository-Schnittstelle und
Persistenztests, jedoch keinen entsprechenden Immutabilitätstrigger. Die
Memory-Implementierung kopiert gespeicherte/rückgegebene Läufe, hat aber keinen
datenbankweiten Transaktionssnapshot über ihre verschiedenen Quellspeicher.

PDFs lesen ausschließlich den gespeicherten Lauf, nicht aktuelle Stammdaten.
Mit demselben Renderer erzeugt derselbe Lauf auch nach JSON-Roundtrip identische
Bytes. Neu erstellte Läufe enthalten neue ID/Revision/Zeit; ihre PDFs sind daher
trotz gleicher Zahlen nicht bytegleich. Der Snapshot speichert keine
Originaldatei-Bytes, keinen Hash der Belegoriginale und keinen Rendererstand.

Der A4-Briefkopf trennt Verwaltung, Empfängerfenster und Abrechnungsdaten.
Optionale Absender- und Empfängerangaben werden bei Leerwerten ohne Platzhalter
oder Leerzeilen ausgelassen. Die Verwaltungsanschrift wird unter
`/app/verwaltung/einstellungen#hausverwaltung` gepflegt und migrationsfrei im
bestehenden organisationsbezogenen JSON-Datensatz `org_settings.data` gespeichert.
Neue Läufe bevorzugen diese Anschrift; bestehende Liegenschaftskontaktdaten bleiben
der Rückfallwert. Vor Freigabe weist die Seite auf eine fehlende Briefkopfanschrift
hin. Ein gespeicherter Lauf behält seinen Stand: Nach einer Ergänzung muss ein
neuer Lauf berechnet werden. Der Demobriefkopf verwendet den vollen Firmennamen,
Musterstraße 12, 8010 Graz und +43 316 555 100.
Die Kurzfassung steht vor der proportional gesetzten Kostentabelle: Kosten,
geleistete Vorauszahlungen, Ergebnis mit Frist und neue monatliche Vorschläge.
Zahlungsbedingungen, Belegeinsicht und HeizKG-Einwendungen stehen unter
„Hinweise“. Messnachweise und Belegverzeichnis nutzen dieselben Seitenränder;
Tabellenköpfe wiederholen sich beim Seitenwechsel. Laufkennung und Revision
stehen nur als kleine Fußreferenz, Partei-E-Mail-Adressen nicht im Briefkopf.
Standard-PDF-Schriftmetriken bestimmen Zeilenumbrüche und rechtsbündige Beträge
in Punkten; installierte Systemschriften beeinflussen die Ausgabe nicht.

Im Lauf bleiben einzeilige Einheiten bei Desktopbreite rund 56 px hoch. „Details“
öffnet die Kostenarten unter der Zeile und meldet den Zustand per `aria-expanded`.
Ohne JavaScript bleibt die native, tastaturbedienbare Aufklappansicht verfügbar.
Mehrere Parteien behalten je eine zugeordnete PDF-Zeile und 44-px-Bedienflächen.

Das Archiv legt je Partei ein PDF sowie zuletzt das Gesamtpaket ab. IDs sind
aus Lauf/Revision/Einheit/Partei abgeleitet; Wiederholungen ergänzen ein partielles
Archiv, ohne bereits abgelegte Dateien zu ersetzen. Archivdateien tragen Größe
und SHA-256. Versand setzt das vollständige Archiv voraus, prüft vor jedem
Anhang dessen Bytes und verwendet die im Lauf gespeicherte E-Mail-Adresse.
Erfolgreiche Sendungen werden übersprungen, aktive Versuche reserviert,
Fehlversuche protokolliert und erneut versucht. SMTP und Datenbank sind keine
gemeinsame Transaktion: nach Prozessabbruch im Bestätigungsfenster ist eine
doppelte E-Mail technisch möglich; das Protokoll ist kein Zustellnachweis.

## Ausführbare Nachweise

`annual_statement_invariants_hausv767_test.go` prüft 400 deterministisch erzeugte
Fälle über alle vier Schlüssel: Cent-/Anteilsummen, Nullbasen, Saldo,
Eingabereihenfolge und Arbeitsvorschau/Lauf-Parität. Tabellenfälle prüfen
Restcent-Ties, `MaxInt64`, Leerstand, Schalttag und Sommerzeitgrenzen.
`annual_statement_preview_hausv767_test.go`,
`annual_statement_snapshot_hausv767_test.go` und
`annual_statement_amount_hausv767_test.go` sichern die Fehlerkorrekturen ab.
`internal/statementpdf/calculation_hausv767_test.go` prüft berechnete Ergebnisse
bis zu jeder adressierten PDF-Kopie und deren Bytes nach Speicherung.
Die bestehenden Run-, PDF-, Archiv- und Versandtests ergänzen diese um
Persistenz, Berechtigungen, Mandantentrennung, Prüfsummen, Wiederholung und
Unterbrechung. Konkrete Ausführungsergebnisse gehören in den Lane-Bericht.

## Rechtsgrundlage je Periode

Die Periodenstruktur hält `weg`, `mrg_voll`, `mrg_teil` oder `ausnahme` und
`heizkg_applies`. Bestehende Perioden beginnen mit WEG und ohne HeizKG; die
Verwaltung muss die tatsächliche Anwendbarkeit prüfen. Die Fristerinnerung
verwendet 30. Juni nach dem Abrechnungsjahr bei MRG Vollanwendung, sonst sechs
Monate ab Periodenende bei WEG/HeizKG. Bei überlappenden Fristen zählt die frühere.
Teilanwendung/Ausnahme ohne HeizKG verweist auf den Vertrag. Der PDF-Kopf nennt
Regime und Rechtsgrundlage aus dem Snapshot. Änderungen erfordern eine neue
Revision; freigegebene Läufe werden niemals umgeschrieben.

## Zahlungsbedingungen

Das Abrechnungsdatum ist das Freigabedatum in Europe/Vienna, im Entwurf das
Erstellungsdatum. WEG: Guthaben wird auf künftige Vorauszahlungen angerechnet,
Nachzahlung binnen zwei Kalendermonaten. HeizKG: Guthaben und Nachzahlung binnen
zwei Monaten. Monatsenden werden auf den letzten Tag des Zielmonats begrenzt.
MRG Vollanwendung: übernächster monatlicher Zinstermin; die aktuelle Umsetzung
nimmt den 5. an und weist diese Annahme aus. Abweichende vertragliche Zinstermine
sind noch nicht modelliert. Teilanwendung und Ausnahme verweisen auf den Vertrag.

## Belegeinsicht und Aushang

Ort, Zeitraum/Öffnungszeiten und Kontakt werden pro Periode gespeichert und im
Lauf eingefroren. Der PDF-Anhang führt Rechnungsdatum, bestätigten Lieferanten,
Kostenart und Betrag mit einer fortlaufenden Belegnummer auf. Interne
Dokumentkennungen erscheinen nicht im Kundendokument. Fehlende Altangaben werden
sichtbar benannt. MRG Vollanwendung bietet zusätzlich ein druckbares Aushang-PDF
mit Haussummen und Einsichtshinweis ohne Namen oder Salden einzelner Parteien.

## HeizKG-Verteilung ab Berechnungsversion 2

Heizung und Warmwasser bleiben getrennte Kostenarten. Bei aktivem HeizKG
müssen beide den Schlüssel `verbrauch` verwenden. Jeder zugehörige Beleg ist
`energie` oder `sonstige_betriebskosten`. Pro Haus/Periode werden der
Verbrauchsanteil (55–85 %, Vorgabe 70 %) und die versorgbare Nutzfläche jeder
Einheit explizit gespeichert. Nicht versorgte Einheiten erhalten ausdrücklich
0 m²; fehlende Fläche, ungültiger Anteil oder fehlende Belegart sperren den Lauf.

Der Energiepool wird auf Hausebene mit dem Verbrauchsprozentsatz multipliziert
und kaufmännisch auf Cent gerundet. Der exakte Rest plus alle sonstigen
Betriebskosten bildet den Flächenpool. Jeder Pool wird mit den bisherigen
PPM-/Restcent-Regeln verteilt. Beide Pools zusammen ergeben centgenau die
Belegsumme. Die Geldvorschau verwendet für HeizKG denselben Laufrechner.
`ReplayAnnualStatementRun` dispatcht anhand der gespeicherten Version:
Version 1 behält ihre ursprüngliche Verteilung mit einem einzelnen Schlüssel.

Geleistetes Heizungs-/Warmwasser-Akonto wird je Einheit ausdrücklich erfasst;
die Komponentensumme darf das Gesamtakonto nicht überschreiten. PDFs weisen
Energie-/sonstige Kosten, Flächen, Verbrauch, Verhältnis, Einheitenergebnis,
Komponenten-Akonto und -Saldo sowie die sechsmonatige Einwendungsfrist aus.
Die Summenzeile zeigt bei HeizKG keinen einzelnen Prozentsatz: Verbrauch und
Fläche sind getrennte Schlüssel und im PDF-Messnachweis erläutert.

## Neue monatliche Vorauszahlungen

Version 2 speichert je Einheit und Kostenart einen Monatsvorschlag im Ergebnis.
MRG Vollanwendung verwendet die Jahreskosten / 12 (Cent, kaufmännisch gerundet),
mit Hinweis bei einer manuellen Vorgabe über 110 % dieser Basis. HeizKG verwendet
zwingend den Vorperiodenanteil / 12. WEG und vertragliche Regime benötigen eine
manuelle Vorausschau; fehlende Werte heißen „Noch festzulegen“. Das Gültig-ab-Datum
ist pro Periode wählbar; ohne Vorgabe gilt im PDF der erste Tag des Monats nach
der Freigabe (im Entwurf nach Erstellung).

## Rücklage (WEG)

Nur bei Regime `weg`. Buchungen liegen in `annual_statement_reserve_entries`
und sind nur einfügbar; eine Korrektur ist eine weitere Buchung. Arten:
`opening`, `contribution`, `withdrawal`, `interest`, `closing_check`.
Entnahmen verweisen auf ein Dokument. Der Endstand ist

`Anfangsstand + Zuführungen − Entnahmen + Zinsen`

in Cent, ohne Gleitkomma. `closing_check` geht nicht in die Formel ein; weicht
die Summe der Kontrollbuchungen ab, entsteht ein Hinweis.

Der Anteil je Einheit verwendet die bestehende Nutzwert-Basis und die
Centverteilung nach größtem Rest. Die Summe der Anteile ist der Endstand.
Historische Läufe ohne Buchungs-Snapshot (`reserve` fehlt in der Eingabe)
rechnen die Kosten unverändert und ohne Rücklage nach. Die Berechnungsversion
bleibt ohne Umsatzsteuerausweis `2`, mit Umsatzsteuerausweis gilt `3`.
Der Rücklageblock steht im Parteienbrief nach Kostentabelle und Einheitsbasis,
vor den Hinweisen, auf demselben gemessenen Satzspiegel wie der übrige Brief.

Die Mindestprüfung warnt nur. Ab 2026 gilt 1,12 €/m²/Monat Nutzfläche
(WEG 2002 § 31; 0,90 × 128,1 / 102,6 = 1,1237, angesetzt mit 1,12; Quelle WKO/ÖVI).
Fläche ist die Summe der erfassten Nutzflächen der Periode in Hundertstel m².
Ein Rest von 0,50 Cent wird abgerundet. Zuführungen unter
`Monatsminimum × Monate der Periode` erzeugen den Hinweis, der Schwellenwert
selbst nicht. Fehlende Nutzfläche warnt ebenfalls, sperrt den Lauf aber nicht.

Im Folgejahr füllt der jüngste freigegebene Vorjahreslauf die leeren Akontofelder
mit zwölf Monatsbeträgen vor. Es bleibt ausdrücklich ein ungespeicherter
Vorschlag, bis die Verwaltung die tatsächlichen Zahlungen bestätigt; vorhandene
Zahlungseinträge bleiben erhalten. Beim Klonen einer Periode werden geleistete
Heizkosten-Akontos, manuelle Monatsvorgaben, Gültig-ab-Datum und Einsichtszeitraum
geleert; Regime, Flächen, Ort und Kontakt bleiben als Vorlage erhalten.

## Umsatzsteuer je Kostenart

Belege speichern den Bruttobetrag der Rechnung. Die Periode schaltet
„Umsatzsteuer ausweisen“ standardmäßig aus; dann bleiben Verteilung, Saldo und
PDF unverändert und der Lauf bleibt bei Berechnungsversion 2. Version 1 und 2
werden ohne Steueraufteilung wiederholt.

Ist die Anzeige an, trägt jede Kostenart der Periode 0, 10 oder 20 %. Heizung
und Warmwasser beginnen mit 20 %, alle übrigen mit 10 %. Der Bruttoanteil der
Einheit bleibt die bisherige Centverteilung. Pro Steuersatz wird die Umsatzsteuer
einmal aus der Haus-Bruttosumme dieser Gruppe kaufmännisch gerundet
(10 % = Brutto × 10/110, 20 % = Brutto × 20/120, 0 % = 0). Die Cent gehen nach
größtem Rest auf die bewohnten und leerstehenden Einheitsanteile und innerhalb
dieser Anteile auf die Kostenarten. Der Eigentümeranteil bei Leerstand bleibt
Teil der Haus-Steuersumme; Parteienbrief und Aushang enthalten auch seine
Netto- und Steuerbeträge.
Netto ist Brutto minus Umsatzsteuer, daher stimmt jede Zeile, jede Gruppe und
die Gesamtsumme auf den Cent. Solche Läufe speichern Berechnungsversion 3.
Das PDF zeigt dann Netto, USt-Satz, USt und Brutto sowie die Summen je Satz.

## Bedienung der Jahresabrechnung

Berechenbare Perioden zeigen „Grundlagen und Belege prüfen oder bearbeiten“
zunächst geschlossen, auch ohne gespeicherten Lauf. Blockierende Angaben öffnen
die Vorbereitung; der erste zuordenbare Blocker wird angesprungen und markiert.
Die Hinweisliste verlinkt weitere betroffene Abschnitte. Rückmeldungen zu
Änderungen bleiben sichtbar. HeizKG-Flächen und geleistete Komponenten-Akontos
haben getrennte Aufklappbereiche; ihre Werte werden weiterhin zusammen mit der
Rechtsgrundlage gespeichert.

Partei-PDFs öffnen im selben Browser-Tab (`Content-Disposition: inline`).
„Herunterladen“ verwendet dieselbe Route mit `download=1` und liefert dieselben
PDF-Daten als Attachment. Die Schritte lauten Freigabe → Dokumentenarchiv →
E-Mail-Versand; unter Dokumente erscheinen die Abrechnungen nach dem Archivieren.
Gesperrte Aktionen zeigen den Grund direkt daneben.

Das PDF übersetzt die gespeicherte Vorschlagsbasis in Eigentümersprache:
„vereinbarte monatliche Vorauszahlung“ bzw. „Vorauszahlung auf Basis des Vorjahres“.
Die gespeicherten Basiskennungen und Rechenregeln bleiben unverändert.

## Optionale Mieterabrechnung aus WEG-Läufen (HAUSV-798)

Ein freigegebener WEG-Lauf zeigt für Einheiten mit Hauptmietverträgen den
Bereich „Mieterabrechnungen“. „Mietverwaltung aktiv“ bindet die Einheit an
einen hinterlegten Eigentümer. Der Kostenartenfilter für MRG-Vollanwendung
beginnt mit eindeutigen §-21-Positionen; Versicherungen, Verwaltungshonorar
und Gemeinschaftsanlagen müssen wegen zusätzlicher Voraussetzungen ausdrücklich
geprüft werden. Teilanwendung und Ausnahme verwenden einen eigenen vereinbarten
Kostenkatalog je Mietvertrag und eine ausdrücklich eingegebene Vertragsfälligkeit.
Die Rücklage wird auch bei eingeschaltetem Filter niemals überwälzt.

`rental_management` speichert das Mandat. `tenant_statements` speichert den
WEG-Lauf, seine Freigabe, Mandat, Mietverträge, zeitlich gültige Parteien und
Rechenergebnis zusammen in einer unveränderlichen Momentaufnahme. Die Ableitung
liest diese Grundlagen in einer mandantengebundenen serialisierbaren Transaktion.
`tenant_statement_approvals` enthält die unveränderliche gesonderte Freigabe.
SQLite-Migration 0065 und PostgreSQL-Migration 0038 führen die Tabellen ein;
PostgreSQL erzwingt RLS. `dbmove` übernimmt Tabellen und Freigaben gemeinsam.

MRG-Jahrespauschalen werden mit der Partei am übernächsten Zinstermin abgerechnet.
Gültigkeitsintervalle der Verträge und Parteien sind am Ende exklusiv. Bei Leerstand
am Fälligkeitstag verbleiben BK-Kosten und Jahresakonto beim Eigentümer.
Heizkosten stammen aus dem freigegebenen HeizKG-Einheitsanteil und werden nach
gleichen Monatsanteilen auf die gültigen Mietparteien verteilt; Teilmonate folgen
den tatsächlichen Tagen des jeweiligen Monats. Größte Reste erhalten einzelne
Cent. Dafür muss die Verwaltung bestätigen, dass keine Zwischenermittlung vorliegt.
Vorhandene Zwischenablesungen sowie vertragliche Mieterwechsel außerhalb der
MRG-Vollanwendung benötigen eine gesonderte Abrechnung; v1 erfindet dafür keine
Vertragsregel. Es werden Kalenderjahre unterstützt.

BK- und Heiz-Akontos kommen aus den datierten Mietzinsbestandteilen, einschließlich
deren hinterlegter Umsatzsteuer. Es handelt sich um vertragliche Akontos, nicht um
einen Zahlungsabgleich mit dem Bankkonto. Fehlende Beträge gelten nicht als Null.
Bei USt-Option muss der WEG-Lauf Netto und Steuer ausweisen. Aus dessen Nettoanteil
entstehen 10 % für Wohnungs-BK bzw. 20 % für Geschäft, Garage und Heizkosten;
ohne Option bleibt der Bruttoaufwand ohne gesonderten Steuerausweis. Der Abgleich
`SourcePassedCents + SourceRetainedCents = WEG-Einheitsanteil` bleibt unabhängig
von einem abweichenden Steuersatz des Mietvertrags erhalten.

Vorschau, gesonderte Freigabe, Archivierung und Versand laufen über die neuen
mandantengebundenen `/app/settings/tenant-statements/{statementID}/…`-Routen.
Archiv und Versand verwenden die bestehenden Dokumente, Prüfsummen und
Versandreservierungen. Jede zeitlich abgegrenzte Mietpartei erhält nur ihren
Abrechnungsteil; der Eigentümer bekommt eine Gesamtkopie. Erfolgreiche Empfänger
werden bei Wiederholung übersprungen. Zugriff benötigt die Mietvertrags- und
Gebäudeverwaltungsberechtigung. Für Vertrags- oder Filteränderungen wird ein neuer
Entwurf erstellt; bereits freigegebene Dokumente ändern sich nicht.

Rechtsgrundlagen: [§ 21 MRG](https://www.ris.bka.gv.at/NormDokument.wxe?Abfrage=Bundesnormen&Gesetzesnummer=10002531&Paragraf=21),
[§ 23 HeizKG](https://www.ris.bka.gv.at/NormDokument.wxe?Abfrage=Bundesnormen&Gesetzesnummer=10007277&Paragraf=23),
[BMF zur Umsatzsteuer bei Vermietung](https://www.bmf.gv.at/themen/steuern/immobilien-grundstuecke/vermietung-verpachtung/vermietung-und-verpachtung-in-der-umsatzsteuer.html).
