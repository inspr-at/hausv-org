# Jahresabrechnung: implementierte Berechnung

Technische Spezifikation der Jahresabrechnung (HAUSV-767/794/802), mit
Berechnungsversionen `1` bis `5`. Sie beschreibt
das Verhalten im Code, keine rechtliche Freigabe für eine konkrete Abrechnung.
Unfreigegebene PDFs tragen den Hinweis „Entwurf zur Prüfung — keine Rechtsauskunft
nach WEG/MRG“. Die einmalige Freigabe speichert Datum, Person und Rolle separat
vom unveränderlichen Lauf. Neue Freigaben speichern zusätzlich den Anzeigenamen
in `approved_name` im bestehenden Freigabe-JSON. Freigegebene PDFs tragen Datum
in Europe/Vienna und diesen Namen statt „Entwurf“; ältere Freigaben ohne Namen
verwenden die gespeicherte Personenkennung. Spätere Profiländerungen verändern
diesen Freigabetext nicht. Archiv und Versand verlangen die Freigabe. Bereits archivierte
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
| `vereinbart` | Vereinbarte PPM je Kostenart und Einheit | Jede Einheit ausdrücklich ≥ 0; Kostensumme genau 1.000.000. |
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

Parteien können inklusive Gültigkeitsdaten (`valid_from` / `valid_to`) tragen.
Leere Grenzen sind unbefristet. Pro E-Mail und Einheit wird ein zusammenhängender
Zeitraum geführt, gemeinsam für ihre Eigentümer-/Mieterrollen. Die Einheitenmaske
pflegt die Grenzen; frühere Parteien bleiben für die Abrechnung in der Zuordnung.
Aktuelle Einheitenmitgliedschaften berücksichtigen die Gültigkeit. Die additive
Spalte `units.party_validity` enthält die Grenzen als JSON und übernimmt die
bestehende Tenant-Isolation/RLS von `units`; dbmove kopiert sie unverändert.
Es gibt keine neue Tenant-Tabelle. Zeitabhängige Nutzflächen-/Personenbasen sind
weiterhin nicht modelliert.

Neue WEG-Läufe übernehmen ausschließlich Eigentümer in ihre Empfängerliste;
Mieter/Bewohner erhalten daraus weder Partei-PDF noch Archiv- oder Versandauftrag.
Die Mieterabrechnung durch den jeweiligen Vermieter ist nicht Teil dieses WEG-Laufs.
Dies gilt auch ohne Parteienwechsel. Bereits gespeicherte Empfängerlisten bleiben
unverändert. Datierte abrechnungsberechtigte Parteien benötigen mindestens **Version 4**;
Datumsgrenzen reiner WEG-Mieter beeinflussen weder Version noch Heizkostenanteile.
Auf den betroffenen Einheiten verdrängt die Fälligkeitsregel den früheren Leerstands-
Tagesanteil. MRG Vollanwendung ordnet sämtliche gewöhnlichen Kosten und Akontos
der Mietpartei am übernächsten Zinstermin zu (Annahme: jeweils 5.). Fehlt dort
eine Mietpartei, erhält die Eigentümerpartei den ganzen Betriebskostensaldo.
WEG ordnet ihn dem Eigentümer am Fälligkeitstag zu (Abrechnungsdatum + zwei
Kalendermonate, § 34 Abs. 4 WEG). MRG Teilanwendung/Ausnahme verwenden hierfür
die ausdrücklich erfasste vertragliche Fälligkeit, ohne § 21 MRG zu unterstellen.
Fehlende oder mehrdeutige Empfänger sperren den Lauf; überlappende gemeinsame
Parteien werden nicht nach erfundenen Eigentumsquoten aufgeteilt.

Das Abrechnungsdatum ist bei datierten Parteien ab Version 4 im Snapshot als Wiener Erstellungsdatum
festgehalten. Eine spätere Freigabe verändert weder Stichtag noch Empfänger.
Bei einem anderen Rechnungsdatum muss ein neuer Lauf erstellt werden.
Versionen 1–3 bleiben im Replay und in ihren PDF-Zuordnungen unverändert.
Die Kostenberechnung von Einheiten ohne datierte Empfänger bleibt unverändert.

Bei WEG wird jeder HeizKG-Zeitraum der Eigentümerpartei zugeordnet; bei MRG
der Mietpartei, ersatzweise der Eigentümerpartei. Die Verbrauchskosten verwenden
Zwischenablesungen an den Wechselgrenzen, soweit eine vollständige Messkette vorliegt; ohne Zwischenablesung werden
alle Heizkosten nach gleichen Monatsanteilen verteilt (§ 23 HeizKG). Die
Flächenkosten und das erfasste Einheiten-Heizkostenakonto folgen Monatsanteilen.
Dies setzt gleichmäßige monatliche Akontos voraus; individuelle Zahlungsverläufe
je Partei werden noch nicht erfasst. Ein Teilmonat zählt als Anteil seiner
Kalendertage an diesem Monat (Februar und Juli bleiben gleich gewichtete Monate).
Lücken gehen an eine gültige Eigentümerpartei; unklare Überlappungen sperren.
Zwischenmessungen werden aus dem vorhandenen Messwertspeicher übernommen;
Grenznachweise und genau die relevanten Zwischenablesungen sind im Snapshot.
Eine nur teilweise erfasste Messkette oder eine Abweichung vom Jahresverbrauch
sperrt den Lauf, statt eine vorhandene Messung durch eine Schätzung zu ersetzen.

Geld und Umsatzsteuer werden centgenau nach größtem Rest verteilt, bei Gleichstand
nach Partei-ID. Parteisummen entsprechen den betroffenen Einheitenbeträgen,
Akontos und Salden. Die PDF zeigt ausschließlich den gespeicherten Partei-Anteil
mit einem erläuternden Satz; in datierten Partei-Abrechnungen werden Kostenzeilen
und USt-Gruppen mit 0,00 Euro weggelassen. Nur der Fälligkeitsempfänger erhält die Rücklageninformation und
künftige Einheiten-Akontovorschläge. Der Lauf zeigt beide Parteien mit Zeitraum
und individuellem Saldo. Das WEG-Demo Janusbergweg 123, Top 3 wechselt am
01.07.2025 von Clara Berger zu Daniel Leitner; der Mieter Matthias Dorn bleibt unverändert und erhält keine
WEG-Abrechnung. Im MRG-Zinshaus Musterstraße 12, Top 2 wechselt zum selben Datum
die Mietpartei von Theresa Aichner zu Lena Krainer. Offene Zeitgrenzen erscheinen
als „bis 30.06.2025“ beziehungsweise „ab 01.07.2025“.

Quellen für die im Auftrag festgelegte WEG-Verwalterabrechnung:
[WEG § 34 Abs. 1 und 4](https://ris.bka.gv.at/NormDokument.wxe?Abfrage=Bundesnormen&Anlage=&Artikel=&FassungVom=2026-05-31&Gesetzesnummer=20001921&Paragraf=34&Uebergangsrecht=),
[HeizKG § 2 Z 4 lit. c](https://www.ris.bka.gv.at/Dokument.wxe?Abfrage=Bundesnormen&Dokumentnummer=NOR40234261)
und [HeizKG § 23](https://ris.bka.gv.at/NormDokument.wxe?Abfrage=Bundesnormen&Anlage=&Artikel=&FassungVom=2026-06-08&Gesetzesnummer=10007277&Paragraf=23&Uebergangsrecht=).
Gesonderte Versorgungsverträge bzw. Gleichstellungen nach § 24b HeizKG werden
hier nicht modelliert; diese Regel beschreibt die WEG-Verwalterabrechnung.

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

In neuen WEG-Läufen ohne datierte Eigentümer erhält jede gespeicherte
Eigentümerpartei eine adressierte Kopie des vollständigen Einheitenergebnisses.
Bei anderen Regimen ohne datierte Parteien und ohne Leerstandsanteil erhält jede
gespeicherte Eigentümer-/Mietpartei eine solche Kopie. Bei MRG-Leerstand
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
Tabellenköpfe wiederholen sich beim Seitenwechsel. Laufnummer und Erstellzeit
stehen als kleine Fußreferenz; die technische Laufkennung erscheint dort nicht.
Messwerte verwenden deutsche Zahlformatierung mit höchstens zwei Nachkommastellen,
Messquellen heißen „Zähler“. Die gespeicherten Mikroeinheiten bleiben unverändert.
Standard-PDF-Schriftmetriken bestimmen Zeilenumbrüche und rechtsbündige Beträge
in Punkten; installierte Systemschriften beeinflussen die Ausgabe nicht.

Im Lauf bleiben einzeilige Einheiten bei Desktopbreite rund 56 px hoch. „Details“
öffnet die Kostenarten unter der Zeile und meldet den Zustand per `aria-expanded`.
Ohne JavaScript bleibt die native, tastaturbedienbare Aufklappansicht verfügbar.
Jede Partei erhält eine eigene Ergebniszeile mit Zeitraum, Kostenanteil, Akonto,
Saldo und PDF-/Downloadlinks, auf schmalen Bildschirmen eine eigene Karte.
Die Einheit gruppiert diese Zeilen; die aufklappbaren Kostenarten zeigen weiterhin
die Einheitsbeträge. Parteibeträge verwenden dieselbe gespeicherte Projektion wie
das jeweilige PDF, einschließlich der historischen Leerstandsbehandlung.
Bedienflächen bleiben mindestens 44 px hoch. Der Lauf nennt den Anzeigenamen der
erstellenden Person; das Freigabedatum verwendet den Wiener Kalendertag.

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

`annual_statement_invariants_hausv767_test.go` prüft 500 deterministisch erzeugte
Fälle über alle fünf Schlüssel: Cent-/Anteilsummen, Nullbasen, Saldo,
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
Die Liste zeigt am selben Datum zuerst den Anfangsstand. Entnahmen zeigen
Notiz und Dokumenttitel; nur ohne Titel erscheint der Dateiname.
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

## Vereinbarte Anteile und HeizKG-Abrechnungsinformationen (HAUSV-802)

Der zusätzliche Schlüssel `vereinbart` speichert einen eigenen PPM-Vektor je
Kostenart in `annual_statement_periods.legal_settings.agreed_shares_ppm`.
Die Eingabetabelle zeigt Prozent mit bis zu vier Nachkommastellen (deutsches
Komma); 0,0001 % entspricht einem Millionstel. Client und Server prüfen die
Summe von genau 100 %, ohne Gleitkommarundung bei der Speicherung.
Jede aktuelle Einheit muss ausdrücklich erfasst sein; 0 ist eine vereinbarte
Ausnahme, etwa für Erdgeschoßwohnungen beim Lift. Negative Werte, fehlende oder
zusätzliche Einheiten sowie eine Summe ungleich 1.000.000 sperren die Berechnung.
Die Verwaltung bearbeitet die Tabelle je Kostenart; die laufende Summenanzeige
und die serverseitige Prüfung verlangen genau 1.000.000. Verschiedene
Kostenarten mit diesem Schlüssel behalten unabhängige Anteile. Die bestehende
Centverteilung nach größtem Rest gilt unverändert, einschließlich der
lexikografischen Restcent-Reihenfolge und der Kostenfreiheit einer Nullbasis.

Läufe mit diesem neuen Schlüssel speichern Berechnungsversion **5** (mit oder
ohne Umsatzsteuerausweis oder datierte Parteien). Version 5 umfasst auch die
Parteien- und Stichtagsregeln von Version 4. Historische Versionen 1–4 behalten
ihren bisherigen Rechenweg und akzeptieren den neuen Schlüssel nicht. Neue
Läufe wählen die niedrigste Version, die alle verwendeten Merkmale abdeckt:

| Version | Rechenweg / Auswahl für neue Läufe |
| --- | --- |
| 1 | Historischer Rechenweg mit einem Heizkostenschlüssel; nur Replay. |
| 2 | Basisberechnung mit HeizKG-Aufteilung, Leerstand und Rücklage. |
| 3 | Zusätzlich Umsatzsteuerausweis. |
| 4 | Datierte abrechnungsberechtigte Parteien, optional mit Umsatzsteuer. |
| 5 | Vereinbarte Anteile, optional mit Umsatzsteuer und datierten Parteien. |

Die Anteile werden als Vorlage ins Folgejahr kopiert. Die Daten
liegen im vorhandenen mandantengeschützten Perioden-JSON und im unveränderlichen
Lauf-JSON; es gibt keine zusätzliche Tabelle oder Datenbankmigration.

`legal_settings.heating_information` enthält pro Periode mehrere Energiebezüge:
Kostenart, Lieferant, Energieträger, Menge, Mengeneinheit, tatsächlicher Bruttopreis
je Mengeneinheit und Preisstand/Zeitraum. Menge und Europreis verwenden ganze
Mikroeinheiten (bis sechs Nachkommastellen), ohne Gleitkomma. Bei bevorrateten
Energieträgern ist der tatsächlich bezahlte Preis einzutragen. Diese Angaben
informieren; nur bestätigte Belege bestimmen die Kostenpools. Ergänzende Felder
halten Steuern/Abgaben/Zolltarife, Mess-/Berechnungskosten, sonstige Betriebskosten,
bei Fernwärmeanlagen über 20 MW Brennstoffmix und jährliche Treibhausgasemissionen,
Beschwerdekontakt und den Zugang zur monatlichen Verbrauchsinformation fest.
Die Energieinformationen werden beim Anlegen des Folgejahres geleert.
Der PDF-Anhang gliedert Messnachweis, Energiebezüge/Preise, Verbrauchsvergleich
und Verbraucherinformation mit eigenen Überschriften; kurze Fakten erscheinen
als Beschriftung/Wert. Mess- und Vergleichswerte verwenden Tausenderpunkte und
höchstens zwei Dezimalstellen, Energiepreise das Eurozeichen bei unveränderter
Preispräzision. Die gespeicherten Rechenwerte bleiben unverändert.

Neue Läufe frieren die höchste gespeicherte Revision des gleichen
Vorjahreszeitraums samt Einheitsverbrauch als `previous_heating` ein. Beginn und
Ende müssen jeweils zwölf Kalendermonate zuvor liegen (Monatsenden werden
begrenzt). Fehlt der Lauf, die Einheit oder eine identische Maßeinheit, wird die
fehlende Vergleichbarkeit ausdrücklich genannt; es gibt keine Schätzung oder
stillschweigende Einheitenumrechnung. Nachträgliche Vorjahresrevisionen verändern
das bereits gespeicherte Vergleichsmaterial nicht. PostgreSQL liest es innerhalb
der Transaktion des aktuellen Laufs. Der Vergleich zeigt tatsächliche Mengen;
für Heizung können fachlich ermittelte Klimafaktoren beider Perioden (Eingabe
als Dezimalfaktor, intern weiterhin Millionstel) samt
Quelle/Methode hinterlegt werden. Der korrigierte Verbrauch ist
`Verbrauch × Klimafaktor / 1.000.000`. Ohne diese Eingabe kennzeichnet das PDF die
fehlende Klimabereinigung ausdrücklich. HAUSV beschafft keine Wetterdaten.

Der Hausvergleich bildet das arithmetische Mittel der gemessenen Verbräuche
versorgter Einheiten derselben gespeicherten Nutzerkategorie (`UnitType`),
einschließlich der betrachteten Einheit. Nullverbrauch zählt mit; ausdrücklich
nicht versorgte Einheiten (0 m²) und andere Kategorien zählen nicht mit. Das PDF
nennt Vergleichsgruppe, Methode und Mengeneinheit. Größen-/Nutzungsunterschiede
werden als Einschränkung genannt. Bei fehlender Kategorie erscheint kein
vermeintlich passender Vergleich. Die Zuordnung der Registertypen zur für die
konkrete Abrechnung geeigneten Nutzerkategorie bleibt fachlich zu prüfen.

Der HeizKG-Anhang nennt außerdem Verbraucherberatung (Arbeiterkammer),
Energieeffizienz und Gerätevergleich (Topprodukte/Österreichische Energieagentur),
Beschwerdewege zur zuständigen kommunalen Schlichtungsstelle bzw. zum
Bezirksgericht sowie Verbraucherschlichtung Austria mit Zuständigkeitsvorbehalt.
Bei fernablesbaren Zählern erinnert er an die monatliche Information innerhalb
der Heiz-/Kühlperiode; die Jahresabrechnung ersetzt deren tatsächliche
Bereitstellung nicht. Ein monatlicher Versanddienst ist nicht Teil dieser
Änderung. Fehlende Sachangaben werden sichtbar benannt; ein berechenbarer Lauf
ist damit keine automatische Bestätigung einer vollständig erfüllten
Informationspflicht. Historische Läufe ohne diese Angaben werden nicht ergänzt.

Rechtsquellen, abgerufen am 24.09.2026:

- [RIS HeizKG § 18, NOR40251314, Fassung BGBl. I Nr. 22/2023](https://www.ris.bka.gv.at/Dokumente/Bundesnormen/NOR40251314/NOR40251314.html): Abs. 1 Z 1a–1c Preise, bedingte Fernwärmeangaben und Mengen; Z 6/6a Messmethode/-kosten und klimabezogener Vorperiodenvergleich; Z 8 sonstige Betriebskosten; Z 12–15 Folgen, Verbraucherinformation, Beschwerdeverfahren und Durchschnittsabnehmer derselben Nutzerkategorie; Abs. 5 kostenfreie Information und Datenzugang.
- [RIS HeizKG § 17, NOR40234276](https://www.ris.bka.gv.at/Dokumente/Bundesnormen/NOR40234276/NOR40234276.html): Abs. 3 Rechnungsabgrenzung bei Bevorratung; Abs. 5 monatliche Verbrauchsinformation bei Fernablesbarkeit ab 01.01.2022.
- [Arbeiterkammer: Heizkostenabrechnung nach dem HeizKG](https://wien.arbeiterkammer.at/beratung/Wohnen/Heizkostenabrechnung/Heizkostenabrechnung_nach_dem_HeizKG.html), [Topprodukte und Kontakt der Energieagentur](https://www.klimaaktiv.at/private/topprodukte), [Verbraucherschlichtung: Antrag und Zuständigkeit](https://www.verbraucherschlichtung.at/antrag/).

Die Janusbergweg-Demoperiode 2025 enthält Liftkosten von 1.800,00 EUR; Top 1–4
(EG) und Stellplätze erhalten 0, die übrigen Wohnungen teilen 1.000.000 PPM.
Der Fernwärmebezug beträgt 100.000 kWh zu 0,09 EUR/kWh brutto (9.000,00 EUR).
Brennstoffmix und Emissionsmenge sind ausdrücklich synthetische Demodaten.

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
SQLite-Migration 0067 und PostgreSQL-Migration 0040 führen die Tabellen ein;
PostgreSQL erzwingt RLS. `dbmove` übernimmt Tabellen und Freigaben gemeinsam.

Die WEG-Empfängerliste bleibt auf Eigentümer beschränkt; Mietparteien kommen
aus den gespeicherten Mietverträgen. Teilt ein v4-/v5-WEG-Lauf eine Einheit
zwischen aufeinanderfolgenden Eigentümern auf, sperrt v1 die Mieterableitung
mit einem Hinweis auf die gesondert zuzuordnenden Vermieterzeiträume und Akontos.
Der gesamte Einheitsbetrag darf nicht nochmals jedem Eigentümer zugerechnet
werden; die nötige zeitliche Vermieterzuordnung ist noch nicht modelliert.

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
