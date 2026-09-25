# Wertsicherung: Daten und Rechenkern

HAUSV-768, Rechenkern: `internal/indexation`; HAUSV-778, Läufe und Schreiben:
`internal/store/valorisation*`, `internal/valorisationpdf`. Stand der Daten und
Rechtsquellen: 24.09.2026. Diese Schnittstelle berechnet Vertragsklauseln,
gesetzliche Vergleichswerte und Termine getrennt. Sie speichert keine
Mietverträge, erstellt keine Vorschreibungen und verschickt keine Schreiben.

## Offline-Datenbestand

`LoadSnapshot()` lädt ausschließlich eingebettete Dateien:

- `internal/indexation/data/vpi.csv`: 3.092 monatliche Gesamtindexwerte und
  251 gesonderte Jahresdurchschnitte. Keine COICOP-Untergruppen.
- `data/manifest.json`: Datenversion `260924101853.0.0`, Abrufzeit,
  Quellen je Reihe, SHA-256 der ursprünglichen CSV-Dateien und des normalisierten
  Bestands, Lizenz, Namensnennung und amtliche Verkettungsfaktoren.
- `data/statutory-rates.csv`: veröffentlichte Richtwerte und Kategoriebeträge
  mit Gültigkeitsintervallen und einem Quellenlink je Zeile.

| Reihe | Erster Monatswert | Letzter Monatswert |
|---|---|---|
| VPI 1966 | 1967-01 | 2026-08 |
| VPI 1976 | 1977-01 | 2026-08 |
| VPI 1986 | 1987-01 | 2026-08 |
| VPI 1996 | 1997-01 | 2026-08 |
| VPI 2000 | 2001-01 | 2026-08 |
| VPI 2005 | 2006-01 | 2026-08 |
| VPI 2010 | 2011-01 | 2026-08 |
| VPI 2015 | 2016-01 | 2026-08 |
| VPI 2020 | 2021-01 | 2026-08 |
| VPI 2025 | 2026-01 | 2026-08 |

Bis Juli 2026 sind die Werte endgültig; August 2026 ist vorläufig.
Beleg: [Statistik Austria, Veröffentlichung vom 17.09.2026, Tabelle 1](https://www.statistik.at/fileadmin/announcement/2026/09/20260917VPIAugust2026.pdf).
Ein Abfragemonat ist **kein historischer Veröffentlichungsstand**: `asOf` in
`Evaluate` begrenzt die Beobachtungsmonate innerhalb dieses Datenbestands.
Historische Revisionsstände und tatsächliche Veröffentlichungstage je Wert sind
nicht enthalten. Der Terminhelfer verlangt deshalb ein belegtes
`FinalPublishedOn` vom Aufrufer.

Die aktuellen OGD-Datensätze der Reihen 2005, 2010, 2015, 2020 und 2025 heißen
`OGD_vpiXXc18_VPI_YYYYCOICOP18_1`. Sie enthalten auch die fortgeschriebenen
älteren Basen bis August 2026. Die historischen Datensätze **ohne** `c18`
enden dagegen im Dezember 2025. VPI 2000, 1996, 1986, 1976 und 1966 verwenden
weiterhin ihre bisherigen Datensatznamen. Die vollständigen, tatsächlich
abgerufenen URLs stehen im Manifest. Somit werden alle zehn Reihen als amtlich
veröffentlichte Werte geliefert; keine Reihe im Snapshot ist selbst verkettet.

### Import und Lizenz

`ImportOGD(io.Reader, ImportOptions)` liest das amtliche Semikolon-CSV:

- Zeitraumspalte `C-VPIZR-0`, Messzahl `F-VPIMZBM`;
- Gesamtindex `VPI-0` oder `VPICOICOP18-0`;
- `VPIZR-YYYYMM` für Monate und `VPIZR-YYYY` für Jahresdurchschnitte;
- Dezimalkomma, optional UTF-8-BOM und CRLF;
- alte und neue COICOP-Dimensionsspalten.

Die Optionen verlangen Reihe, Quellen-URL, Abrufzeit und `FinalThrough`.
Die Messzahlen-CSV selbst enthält keine Statusspalte. Der Aufrufer muss den
endgültigen Stand anhand der Veröffentlichung oder des zugehörigen
Periodenverzeichnisses (`_C-VPIZR-0.csv`, Kennzeichnung „vorl.“) belegen.
Ein fehlender Statusnachweis wird nicht als endgültig interpretiert. Doppelte
Perioden, ungültige Werte, unbekannte Formate und künftige Monate werden abgelehnt.
Ein Fehler liefert keinen teilweise verwendbaren Import zurück.

Die Daten stehen unter [CC BY 4.0 laut OGD-Nutzungsbedingungen](https://data.statistik.gv.at/web/?page=terms).
Namensnennung gemäß der abgerufenen englischen Fassung:
**Data source: Statistics Austria — data.statistik.gv.at**.
HAUSV kennzeichnet im Manifest seine Änderungen: Auswahl des Gesamtindex,
ISO-Zeiträume, Dezimalpunkt, Trennung der Jahresmittel und ergänzter Status.
Die amtlichen Fakten stammen von Statistik Austria; die Mietberechnung ist
eine HAUSV-Berechnung. Aktualisieren bedeutet einen neuen geprüften Snapshot
inklusive Quellen-Digests, Datenversion und Statusnachweis einzuspielen.

Für diesen Slice genügt das Embed: kleine öffentliche Referenztabellen,
reproduzierbare Berechnungen und kein veränderlicher Laufzeitbestand. Daher
keine SQLite-/PostgreSQL-Migration. Ein späterer automatischer Import braucht
eine append-only Import-/Revisionshistorie; er darf abgeschlossene Läufe nicht
nachträglich verändern.

## Zahlen und Vertragsklausel

Öffentliche Geldbeträge sind `int64`-Cent. `Decimal` speichert Indexpunkte,
Prozente und Faktoren als Millionstel: `5 * Unit` bedeutet **5 %**.
`ParseDecimal` akzeptiert Punkt oder Komma, keine Exponenten oder
Gruppierungszeichen. Zwischenrechnungen verwenden `math/big.Rat` und
`math/big.Int`, niemals binäre Fließkommazahlen. Ein Überlauf des Cent-Ergebnisses
ist ein Fehler.

`Clause` hält Reihe, endgültigen Basismonat und Basiswert, Vertragsbetrag,
Schwelle in Prozent oder Punkten sowie folgende Optionen:

- `Crossing = Exceeds` (Nullwert): bis einschließlich der Schwelle unverändert.
  `Reaches` löst bereits bei Gleichheit aus.
- `ChangeMode = FullChange` (Nullwert): bei Überschreitung zählt die gesamte
  Änderung. `ExcessChange` zählt nur den übersteigenden Teil.
- Steigerungen und Senkungen werden symmetrisch geprüft.
- Indexpunkte werden auf eine Dezimalstelle gerundet.
  `PercentRounding = OneDecimalPercent` (Nullwert) rundet auch die
  Veränderungsrate auf eine Dezimalstelle **vor** dem Schwellenvergleich.
  `UnroundedPercent` verwendet den exakten Quotienten.
- `MinimumChangeCents` ist eine optionale symmetrische Bagatellgrenze.
  Unterhalb der Grenze bleibt auch die Basis unverändert.
- `ExactAmountCents` bewahrt als Zeichenfolge eines ganzzahligen Bruchs die
  exakte Vertragskurve über mehrere Sprünge. Cent-Bruchteile werden nicht
  jährlich verloren; die angezeigten Vertrags-Cent werden kaufmännisch gerundet.

Diese Standardwerte folgen dem [WKO-Klauselbeispiel](https://www.wko.at/wirtschaftsrecht/wertsicherung-miet-und-pachtvertraege).
Die Klausel muss im Einzelfall geprüft und passend parametrisiert werden.

`Evaluate(clause, dataset, asOf)` gibt den **ersten** zulässigen Sprung zurück:
Auslösemonat, Veränderungsrate, neue Basis, neuer Betrag, `NextClause` und
strukturierte `ExplanationStep`-Einträge. Mit `NextClause` kann der Aufrufer
weitere Sprünge chronologisch berechnen. Endgültiger Basiswert und Datenbestand
müssen übereinstimmen. Eine Lücke innerhalb der Reihe ist ein Fehler;
vorläufige oder noch nicht publizierte Monate erscheinen als `PendingMonth`
und werden nicht übersprungen. `NewBase` ist die auslösende Indexzahl,
nicht ein gesetzlich gekappter Zwischenwert.

`Dataset.Lookup` bevorzugt veröffentlichte Werte. Fehlt ein Wert, kann es für
Monate ab Jänner 2026 aus VPI 2025 die ältere Basis ableiten. Beispiel:
`103,1 × 1,535 = 158,2585 → 158,3` für VPI 2010.
Die [amtlichen Faktoren vom 26.02.2026](https://www.statistik.at/fileadmin/pages/214/Factsheet_Verkettungsfaktoren.pdf)
sind im Manifest gespeichert. Solche Ableitungen tragen `ChainSource` und
benötigen bei `Evaluate` ausdrücklich `AllowDerived = true`. Standardmäßig
blockiert `index_derived`; eine gerundete Verkettung kann eine Schwelle ändern.
Keine Rückrechnung dieser Faktoren in frühere Zeiträume.

## Gesetzliche Parallelrechnung ab 2026

Quellen: [MieWeG §§ 1 und 4, BGBl. I 114/2025](https://www.ris.bka.gv.at/Dokumente/BgblAuth/BGBLA_2025_I_114/BGBLA_2025_I_114.pdf)
und [ErlRV 269, insbesondere die Erläuterungen zu § 1](https://www.parlament.gv.at/dokument/XXVIII/I/269/fnameorig_1718389.html).

`CapCurve(anchor, startCents, annualAverages, restricted, throughYear)` bildet
die gesetzliche Kurve unabhängig vom Vertrag. Bei Altverträgen ist der Anker
der Indexmonat der letzten Valorisierung, andernfalls der Abschlussmonat.
Für einen Jahresdurchschnitt als letzte Basis ist Dezember einzusetzen.

Für jedes betroffene Jahr wird der exakte Quotient der endgültigen
VPI-2020-Jahresdurchschnitte verglichen. **Nicht** die auf eine Dezimalstelle
veröffentlichte Inflationsrate verwenden. Über 3 % zählt der Mehrbetrag zur
Hälfte. Bei Preisbeschränkung wird die Inflation 2025 auf 1 % und 2026 auf 2 %
begrenzt; das betrifft auch angemessene Hauptmietzinse in Vollanwendung.
Erst danach folgt im ersten Teiljahr die Aliquotierung nach vollen Monaten
nach dem Anker. Der Abschlussmonat zählt auch bei Abschluss am Monatsersten
nicht mit. Dezember trägt null Monate bei. Negative Änderungen bleiben erhalten.

Die Faktoren werden exakt multipliziert, ohne Zwischenrundung auf Cent.
`AnnualStep` enthält beide Jahresmittel, exakte ursprüngliche und begrenzte
Prozentsätze, Monatsanteil und exakten Kurvenstand. Gerundet wird erst das
Ergebnis: **genau ein halber Cent wird abgerundet**, mehr als ein halber
aufgerundet. `AnnualCeiling` ist die Variante für ein einzelnes Jahr;
mehrjährige Berechnungen verwenden `CapCurve` mit unverändertem Startanker.

`ApplyMieWeG` kombiniert ein Vertragsergebnis mit dieser Kurve:

- Wohnen in Voll- und Teilanwendung: der niedrigere exakte Betrag gewinnt.
  Eine Erhöhung wird auf einen gültigen 1. April verschoben und nie vor ihren
  vertraglichen Wirksamkeitstermin zurückgezogen.
- Gewerbe (einschließlich eigener Klassifikation `CommercialFullMRG`) und
  Vollausnahmen bleiben von dieser gesetzlichen Kurve ausgenommen.
- `IsSublease` trennt die weiterhin anwendbare MieWeG-Begrenzung von der
  Hauptmietvertrags-Zustellregel. Nicht klassifizierte Fälle und WGG werden
  abgelehnt; die Sonderfälle sind nicht automatisiert.
- Eine günstigere vertragliche Senkung wird nicht allein wegen MieWeG auf
  den nächsten April verschoben. Ohne Vertragssprung wird keine Erhöhung erzeugt.
- Optional kann ein extern geprüfter `MaxRentCents` die §-16-Obergrenze setzen.
  Der Kern ermittelt keine Richtwertzuschläge oder rechtlich angemessene Miete.

Ergebnisse: `CalculatedCents` (Vertrag), `PermittedCents` (nach Begrenzung),
`AllowedNowCents` (nach Wirksamkeitsdatum), `DeferredCents` (zeitlich verschoben)
und `CappedCents` (gekappter Rest). **Der gekappte Rest ist keine spätere
Forderung.** `AllowedNowCents` ist noch keine Zustell- oder Einhebungserlaubnis;
`Timing` muss anschließend das Schreiben prüfen.
Die Vertragsbasis bleibt `NextClause`; die gesetzliche Kurve behält ihren
eigenen Anker. Keine der beiden Kurven wird auf die tatsächlich vorgeschriebene
niedrigere Miete zurückgesetzt. Vor 2026 wirksam gewordene, übersehene Erhöhungen
werden als `missed_pre2026` zur gesonderten Prüfung zurückgegeben.

## Termine und Schreiben

`Timing(TimingInput)` verwendet Kalenderdaten unabhängig von der Zeitzone.
Der Modus ist verpflichtend:

- `CautiousTiming` (`wko`, Standard): übernächster Monatserster nach endgültiger Veröffentlichung.
  April-Index, endgültig 17. Juni → 1. August wirksam → bei Zustellung ab
  1. August und Zinstermin am 5. erstmals September einhebbar.
- `OEVITiming` (`oevi`): Tag der endgültigen Veröffentlichung. April-Index,
  endgültig 17. Juni → bei rechtzeitigem Zugang erstmals 5. Juli einhebbar.
  Das ist eine offene Auslegungsfrage, keine gesicherte Rechtsprechung.
- `ContractualTiming` (`contract`): ausdrücklich übergebener Vertragstermin,
  mindestens endgültige Veröffentlichung; bei Hauptmiete in Vollanwendung
  zusätzlich mindestens der Termin aus `LegalFloorMode`. Die Vertragsauswahl
  übernimmt dafür die Organisationsvorgabe. Eine Organisationsvorgabe `contract`
  oder eine fehlende Vorgabe verwendet WKO als Untergrenze.
  Der Laufstichtag gilt bei Schwellenklauseln als geprüfter Vertragstermin,
  bei periodischen Klauseln der hinterlegte Anpassungsmonat.
- `MieWeGTiming`: ausdrücklich übergebener 1. April aus der Parallelrechnung.
  Für diese Fälle gilt nicht zusätzlich die vorsichtige Verschiebung.

Die Organisationswerte `cautious`/`contractual` werden zu `wko`/`contract`
migriert. `leases.wirksamwerden_mode` ist leer (Organisationsstandard) oder
`wko`, `oevi`, `contract`; die Wahl gilt ausschließlich für Indexerhöhungen
bei Hauptmiete in Vollanwendung außerhalb des MieWeG. Feste Staffeltermine
bleiben davon unabhängig. Eingaben und gewählter Modus sind im Lauf eingefroren.
Die Hilfe unter `/app/hilfe#recht-wirksamwerden` erläutert die Quellen.

Bei `RequiresMRGNotice` werden **Ausstellungs- und tatsächliches Zugangsdatum**
geprüft. Vor Wirksamkeit datierte oder zugegangene Schreiben sind ungültig;
der Kern schiebt sie nicht stillschweigend auf einen späteren Termin.
Der erste Zinstermin muss mindestens 14 Kalendertage nach Zugang liegen.
Standard-Zinstermin ist der 5.; Tage 29–31 werden in kürzeren Monaten auf den
letzten Kalendertag begrenzt. Ohne tatsächliche Schreibenstermine liefert der
Kern eine als `PlannedNotice` erkennbare früheste Planung. Freigabe und
PDF-Erstellung rechnen mit Zugang am Schreibendatum; beide Annahmen werden
im `TimingInput` gespeichert und in der Oberfläche ausdrücklich als Annahme
angezeigt. Das ist kein Zustellnachweis. Tatsächliche E-Mail-Versandzeiten
stehen getrennt in `valorisation_deliveries`. Dort werden auch bestätigter
Zugang (`received_on`) und der daraus folgende erste Zinstermin
(`receipt_due_on`) pro Empfänger gespeichert; die Erfassung wird protokolliert.
Die Freigabeberechnung bleibt unverändert, ein späterer Zugang wird separat
sichtbar und kann keine rückwirkende Zahlungspflicht erzeugen.

MieWeG-Beispiele: Zugang 21.04.2026 → 05.05.2026; Zugang 22.04.2026 → 05.06.2026.
Ein verpasster April-Versand kann später einen künftigen Zinstermin erreichen,
ohne Rückwirkung in Vollanwendung. Quellen: ErlRV zu § 1 Abs. 5 und WKO oben.

## Richtwerte und Kategorien

`Richtwert(state, date)` und `CategoryAmount(category, date)` liefern amtliche
Cent/m², Gültigkeitsintervall und Beleg. Wien: 667 Cent ab 01.04.2023,
674 Cent ab 01.04.2026. Kategorie A: 423 Cent am 01.04.2023, 447 Cent ab
01.07.2023, 451 Cent ab 01.04.2026. 2024/2025 bleiben die letzten Werte gültig.
Kategorie D wird ausdrücklich in brauchbar/unbrauchbar unterschieden:
ab April 2026 225/113 Cent. Unbekannte Länder/Kategorien und Daten ab
01.04.2027 liefern einen Fehler statt einer ungeprüften Fortschreibung.

Alle neun Länder und Kategorien sind zeilenweise belegt durch:

- [BMJ, BGBl. II 81/2023: Richtwerte](https://www.ris.bka.gv.at/Dokumente/BgblAuth/BGBLA_2023_II_81/BGBLA_2023_II_81.pdf).
- [BMJ, BGBl. II 170/2023: alte und neue Kategoriebeträge](https://www.ris.bka.gv.at/Dokumente/BgblAuth/BGBLA_2023_II_170/BGBLA_2023_II_170.html).
- [Statistik Austria, Veröffentlichung gemäß RichtWG/MRG vom 31.03.2026](https://www.statistik.at/fileadmin/pages/2114/VOe_Richtwerte_Kategoriemieten_20260401.pdf).

## Numerische Prüfbeispiele

Die ersten drei Erwartungen wurden aus dem
[Statistik-Austria-Handbuch des Wertsicherungsrechners, Seiten 7–12](https://www.statistik.at/fileadmin/pages/214/Handbuch_Wertsicherungsrechner_VPI.pdf)
übernommen, nicht mit dem neuen Kern erzeugt. Es handelt sich um veröffentlichte
Rechnerbeispiele; eine separate Live-Browser-Abfrage wurde nicht durchgeführt.

| Beispiel | Eingabe | Erwartung |
|---|---|---|
| Handbuch 1 | VPI 2015, Jän. 2016 99,8 → Dez. 2022 125,6, 500 € | +25,9 %, 629,50 € |
| Handbuch 2a | VPI 2010, Jän. 2020 119,1 → Nov. 2021 125,6, 1.000 €, Schwelle ab 5 % | +5,5 %, 1.055,00 € |
| Handbuch 2b | neue Basis Nov. 2021 125,6 → Juni 2022 133,6 | +6,4 %, 1.122,52 € |
| Grenzprüfung Handbuch | Okt. 2021 gegen Jän. 2020; Mai 2022 gegen neue Basis | kein Sprung |
| Rechtsrecherche E1 | Anker Sep. 2024, 1.000 €, Jahresmittel 120,3 / 123,8 / 128,2, Teilanwendung | Deckelkurve 1.040,28 € am 01.04.2026 |
| Rechtsrecherche E2 | wie E1, aber Preisbeschränkung in Vollanwendung | Deckelkurve 1.017,35 € |
| Rechtsrecherche E3 | Abschluss Feb. 2026, 1.200 €, illustrative Inflation 3,4 %, 10/12 | 1.232,00 €; mit 2-%-Deckel 1.220,00 € |

E1 verwendet eine **ungerundete** vertragliche Veränderungsrate:
129,8 / 123,6 − 1 = 5,016… %. Die Standardklausel mit einer Dezimalstelle
und striktem Überschreiten löst bei gerundeten 5,0 % noch nicht aus.
Die gesetzliche E1-Berechnung bleibt ungerundet; der in der Rechtsrecherche
erwähnte ÖVI-Vergleich von 1.040,26 € verwendet dagegen eine gerundete Rate.

Zusätzlich geprüft: Prozent-/Punkteschwellen, inklusive/exklusive Grenze,
ganzer Sprung/Mehrbetrag, Senkung, Bagatellgrenze, exakte Weiterführung,
Datenlücken, endgültige Basis, vorläufiger August, direkte und abgeleitete
Reihen, Überläufe, Aliquotierung, Dezember-Anker, Gewerbe-Ausnahme, 2026/2027-
Deckel, ungültige Schreiben, 14-Tage-Grenze, Jahreswechsel und Schaltjahr.

## Grenzen des Rechenkerns

Die Vertragswirksamkeit, MRG-Einstufung, Haupt-/Untermiete, Rechtsgrundlage einer
Staffel und der historische Anker müssen vor Nutzung geprüft sein. `Evaluate` implementiert die Monats-/Schwellenklausel.
Der Laufadapter ruft für periodische Klauseln `EvaluateReference` mit dem
geprüften Referenzmonat auf; Zwischenmonate lösen dabei nicht aus. Reine
§-2-Verweisklauseln verwenden die gesetzliche Kurve. Eine Staffel ohne
strukturierte Stufen bleibt `clause_invalid`; aus Freitext wird kein Betrag
erfunden. Veröffentlichungsdaten und Laufpersistenz sind im Adapter ergänzt.

Rechtlich offen bleiben insbesondere der Wirksamkeitszeitpunkt kommerzieller
Hauptmietverträge in Vollanwendung, Richtwertkopplung gegenüber dem
1-%-Deckel, die verbindliche Behandlung von Zwischenrundungen und die Form
bzw. der Nachweis elektronischer Erhöhungsbegehren. Die gewählte exakte
Parallelrechnung entspricht der gelieferten Rechtsrecherche und den ErlRV;
Klausel- und Versandfreigaben sind keine Leistung dieser Rechenbibliothek.

Der VPI-2020-Bestand enthält Jahresmittel ab 2021. Fehlt für einen älteren
Anker ein benötigtes Vorjahresmittel, verweigert `CapCurve` die Berechnung;
die Ausnahme nennt Reihe und fehlendes Jahresmittel. Fehlende Monatswerte
oder Veröffentlichungsnachweise nennen ebenfalls den konkreten Monat.
Es wird kein Mittel einer anderen Basis stillschweigend eingesetzt. BK- und
Heizungsakonti gehören nicht in `AmountCents`.

## Persistierte Läufe (HAUSV-778)

`PreviewValorisation` berechnet eine Liegenschaft ohne HTTP und ohne Schreibzugriff.
Die Organisationsseite bündelt nur die ausgewählten, berechtigten Häuser; jedes
Haus besitzt einen eigenen, mandantengebundenen Lauf. `ValorisationRepository`
lädt die geprüften Verträge, erzeugt Revisionen und speichert Eingabe-SHA-256,
Index-Snapshot und Berechnungsevidenz. Die Gruppen sind Bereit, Unverändert
und Ausnahmen; jede Ausnahme trägt einen stabilen Code und deutschen Text.

`PublishedIndexValue` verwendet den eingebetteten endgültigen Originalwert.
Der Adapter ergänzt amtliche Veröffentlichungskalender 2021–2026 aus den in
`valorisation_publications.go` verlinkten Tabellen. Die erste Veröffentlichung
ist vorläufig; endgültig wird der Monat am Erstveröffentlichungstag des
Folgemonats. Fehlende Kalender- oder Indexdaten sperren eine benötigte Berechnung.
Ein heutiger Snapshot rekonstruiert keine früheren Revisionsstände.

Die Tabellen `valorisation_runs`, `valorisation_items`, `valorisation_deliveries`
und `valorisation_events` haben Mandanten-IDs und in PostgreSQL erzwungenes RLS.
SQL-Trigger sperren nach Freigabe Änderungen der Berechnungen sowie Hinzufügen
und Löschen von Positionen. Versandstatus, Stornovermerk und neue Briefreferenzen
bleiben getrennt veränderbar. Ein Datenbankumzug stellt eingefrorene Positionen
innerhalb einer Transaktion wieder her und aktiviert die Einfügesperre vor der
abschließenden Prüfung erneut.

Freigabe benötigt `manage_leases` und `approve_valorisation`, bei aktivierter
Vier-Augen-Regel zusätzlich eine andere Person. Geänderte Verträge, Einstellungen
oder Index-Snapshots erzwingen eine neue Berechnung. Die Transaktion archiviert
unveränderliche PDF-Dateien mit SHA-256, ergänzt den HMZ-Bestandteil ab
`wirksam_on` mit `origin=valorisation_item:<id>` und schreibt den Kurvenzustand
fort. Die exakte Vertragskurve und der ursprüngliche Deckelanker bleiben im
vorherigen Lauf erhalten; der niedrigere vorgeschriebene Betrag setzt sie nicht
zurück. Ausnahmen müssen bearbeitet oder begründet ausgeschlossen sein.
Solange Ausnahmen offen sind, ist „Freigeben und archivieren“ deaktiviert;
der Hinweis nennt die Anzahl betroffener Verträge. Ein trotzdem gesendeter
Freigabe-POST führt mit deutscher Fehlermeldung zum Lauf in der Anwendung
zurück. Eine reine Datumssperre entfällt, sobald die Wirksamkeit erreicht ist.

Organisationsadministratoren konfigurieren unter Verwaltung → Einstellungen
`wirksamwerden_mode` (Standard `cautious`), die Behandlung ungeprüfter Klauseln,
Vier-Augen-Freigabe und die Absenderzeile. Eine manuelle Betragsentscheidung
benötigt Freigaberecht und Begründung; sie darf den berechneten zulässigen Betrag
nicht erhöhen und erscheint ausdrücklich im Schreiben. Rechtliche Ausnahmen
lassen sich damit nicht umgehen.

Versand verwendet eine Reservierung mit `pending`/`sent`/`failed`, Wiederholung
fehlgeschlagener Versuche und höchstens einen erfolgreichen Eintrag je
Lauf/Einheit/E-Mail. Vor dem Versand wird der Brief zum tatsächlichen Tag erneut
terminiert und unveränderlich archiviert; jeder Versand speichert genau dessen
Dokument-ID und SHA-256. Die Planung nimmt Zugang am Versandtag an. Ein späterer
Zugang verschiebt die Einhebbarkeit; SMTP-Erfolg ist kein Zugangsnachweis.
Storno verlangt einen Grund und bewahrt bereits gebuchte Mietzinse und Schreiben.
Eine nötige Mietzinskorrektur bzw. ein Korrekturschreiben erfolgt gesondert.

Die fünf Briefvarianten verwenden `internal/pdf` und österreichisches Geldformat:
MieWeG Voll-/Teilanwendung, vertragliche Vollanwendung/Ausnahme und Verminderung.
Nach Freigabe entfällt „Entwurf“. Haupt-/Untermiete, mehrere Empfänger, unveränderte
weitere Mietbestandteile, zwei Kurven, USt und Einhebungstermin sind sichtbar.
Beide Brieftypen verwenden `pdf.Letterhead` für Absender, Fensteranschrift und
Infoblock. Überlange Angaben laufen in einen gesonderten Abschnitt weiter.
Deutsche Dezimalzahlen und Geldbeträge sind reine Darstellung; exakte Brüche
bleiben in den gespeicherten Berechnungsschritten und Kurven erhalten.
Entwürfe und archivierte Anpassungsschreiben werden mit `Content-Disposition:
inline` geöffnet. Lauf-ID, Referenz und Eingabe-Prüfsumme stehen ausschließlich
im Lauf beziehungsweise in den Archivmetadaten, nicht im Mieterbrief.

Hausroute: `/app/settings/valorisation`; Organisationsroute:
`/app/verwaltung/wertsicherung`. Mutationen liegen unter
`/app/settings/valorisation/runs`, ergänzt um `/{runID}/approve`, `/send`, `/cancel`
und `/items/{itemID}`. Das PDF liegt unter `/items/{itemID}/pdf`.
Mietvertragsseiten zeigen die zugehörige Laufhistorie. Der Demoreset erzeugt
für Janusbergweg 123 einen April-2026-Entwurf mit zwölf Verträgen, E1 1.040,28 €,
E2 1.017,35 €, sieben bereiten Anpassungen (einschließlich Gewerbe außerhalb
MieWeG und der strukturierten Staffel), zwei unveränderten Verträgen und genau
drei Ausnahmen: keine Klausel, ungeprüfte Klausel und einseitige Verbraucherklausel.
Top 8 besitzt den geprüften Jänner-Referenzmonat und einen importierten
Kurvenstand von 2025; Top 11 ebenfalls einen dokumentierten Deckelanker von
2025. Eine Garage ohne HMZ bleibt mit ausdrücklichem Grund unverändert, statt
fälschlich einen fehlenden Index zu melden.

Im Zinshaus Musterstraße 12 verwendet Top 8 den dokumentierten Staffelstand
vom April 2024 mit 910,00 € als Deckelanker. Die festen Schritte im April 2026
und 2027 benötigen keinen monatlichen Index und keinen Veröffentlichungsnachweis;
die gesetzliche Vergleichsrechnung verwendet weiterhin die Jahresmittel.
Für April 2026 sind vertraglich und nach Vergleich 928,20 € berechnet.

Die Zeilen und Briefe nennen die eingefrorene Einheitenbezeichnung und den
Hauptmieter. Freigegebene Schreiben sind direkt aus der eingeklappten Zeile
erreichbar. Die erste Indextabelle zeigt Basis, Auslöser und benötigte
Jahresmittel; die vollständige geprüfte Monatsreihe liegt unter „Alle Indexwerte
anzeigen“. Manuelle Entscheidungen und Storno verlangen weiterhin eine
Begründung, ihre Formulare werden erst nach Öffnen des jeweiligen Bereichs
sichtbar. Die technische Prüfsumme liegt in einem eigenen Detailbereich.

Browserprüfung gegen ein frisches Demo-Rig (verändert ausschließlich lokale
Demodaten): `node scripts/snapshot/qa-valorisation.mjs http://localhost:8309
/absoluter/artefaktpfad`. Sie prüft Vorschau, Beträge, Freigabe, PDF, Rollenabsage
und 390/1440 Pixel. Browseroracle und PDF-Inhaltstests benötigen `pdftotext`; mit
`HAUSV_VALORISATION_PDF_DIR` werden die fünf Testbriefe zusätzlich exportiert.
