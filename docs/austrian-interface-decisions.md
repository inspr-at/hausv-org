# Österreich-first Schnittstellenentscheidungen

hausv.org bleibt Kommunikations- und Transparenzsystem. Schnittstellen dürfen
Daten strukturieren, prüfen, importieren oder exportieren. Sie dürfen keine
Buchhaltung, Steuerlogik, Mahnlogik oder Zahlungsaufträge erzeugen.

## Dienstleister-Zugang: Tenant-Rolle mit Magic-Link

**Freigabestatus:** technisch umgesetzt, aber standardmäßig nicht für den realen
Betrieb freigegeben. Da auf absehbare Zeit keine externe Prüfstelle verfügbar
ist, dokumentiert HAUSV-86 eine versionierte Betreiber-Selbstprüfung anhand
primärer EU- und österreichischer Quellen. Vor echten Einladungen müssen deren
Rollen-, Vertrags-, Aufbewahrungs- und Informationsregeln bewusst bestätigt
werden; das Produkt verlangt dafür neben dem Feature-Flag die exakte
Prüfungsrevision. HAUSV-128 bleibt bis zur dokumentierten Betreiberaktivierung
in QA. Die Selbstprüfung ist kein Zertifikat und keine Rechtsberatung.

Entscheidung: Dienstleister erhalten in v1 ein eingeladenes, tenant-begrenztes
Profil mit der Rolle `Dienstleister`. Ein kurzlebiger Magic-Link meldet dieses
Profil an und führt direkt zum zugewiesenen Anliegen. Die Berechtigung entsteht
nicht durch den Link selbst, sondern durch Tenant-Mitgliedschaft und aktuelle
Anliegen-Zuordnung.

Leitplanken:

- der Magic-Link gilt nur für den Tenant, läuft nach 15 Minuten ab und enthält
  einen Deeplink zum Anlass der Einladung
- Dienstleister sehen nur offene Anliegen im Tenant, die ihrer E-Mail-Adresse
  aktuell zugewiesen sind; mehrere parallele Zuweisungen sind möglich
- Aushänge, Bewohner-Dokumente, Abstimmungen, Einheiten und nicht zugewiesene
  Anliegen bleiben unsichtbar
- Verwaltung kann eine Zuweisung entziehen; erledigte oder entzogene Anliegen
  sind danach nicht mehr zugänglich
- die Firma im Adressbuch und das eingeladene Rollenprofil bleiben getrennte
  Konzepte; es gibt keine globale Dienstleisterrolle über Häuser hinweg

Eine weitere Tenant-Mitgliedschaft für dieselbe Identität braucht immer eine
eigene, ausdrückliche Zuordnung und den passenden Vertrags-/Rollenrahmen.

## Neutrale Übergabe zuerst, Zieladapter getrennt

Entscheidung: Der vorhandene `raw-v0`-Export ist eine formatneutrale,
strukturierte Übergabe und kein BMD- oder RZL-Importformat. BMD NTCS und RZL
FIBU werden als getrennte Zieladapter geplant. Der Scope bleibt
Rohdaten-/Belegübergabe, nicht fertige Buchungssätze.

Freigegebene Rohdaten:

- Zahlungsstatus der Einheiten mit Tenant/Haus, Einheit und Änderungsdatum
- Parkplatz-Monatsabrechnungen mit Zeitraum, Betrag, Status und optionaler
  Zahlungsreferenz
- Beträge nur als Übergabewerte, nicht als gebuchte Soll/Haben-Sätze
- keine Eigentümer-, Mieter- oder E-Mail-Listen

Produktiver Stand:

- Profil `manual-csv/raw-v0` als Semikolon-CSV für kontrollierte Übergaben
- Golden File
  `internal/integrations/testdata/structured-handoff-raw-v0.csv`
- Formatvertrag `docs/structured-handoff-export.md`
- geschützte, bewusst leere Auswahl mit Vorschau und einmaligem ZIP-Download
- Manifest mit Version, Zeitpunkt, Datensatzanzahl und CSV-SHA-256
- Vorschau an Haus und Person gebunden; Download im Aktivitätsverlauf
- Konten- und Steuerfelder werden als außerhalb des Produkts abgelehnt
- keine Behauptung, dass BMD NTCS oder RZL dieses CSV direkt importieren kann

## BMD NTCS als eigener Zieladapter

Öffentliche BMD-Seiten bestätigen Importwege für XLS, CSV, TXT und PDF sowie
Importvorlagen und ein Schnittstellenhandbuch. Sie liefern aber keinen
ausreichenden öffentlichen Feldvertrag, der `raw-v0` als NTCS-kompatibel
belegen würde.

Betreiberprüfung vom 27.07.2026:

- Die öffentliche technische Dokumentation enthält Installations-, System- und
  EDI-Unterlagen, aber keinen FIBU-Feldvertrag für Buchungsimporte.
- Die offizielle BMD-Schulung nennt vier unterschiedliche
  Buchungsimportvorlagen (`KA`, `BK`, `ER`, `AR`), die
  Fleximport-Excel-Schnittstelle und ein Schnittstellenhandbuch. Handbuch und
  Vorlagen werden als Schulungsunterlagen bereitgestellt und sind nicht als
  öffentlicher Feldvertrag abrufbar.
- Die weiterführende BMD-Kundeninformation erfordert eine Anmeldung.
- Im Betreiberbestand liegt derzeit weder eine rechtmäßig nutzbare
  NTCS-Importvorlage noch eine kontrollierte NTCS-Zielumgebung vor.

Damit sind Zielprofil, Pflichtfelder, Feldlimits und Mapping nicht belastbar
bestimmbar. HAUSV baut deshalb keinen geratenen Adapter. Das fehlende Artefakt
ist ein technischer Input und kein Auditor-Gate: Sobald eine offizielle
Vorlage/ein Handbuch oder eine betreibereigene Importvorlage samt konkreter
NTCS-Version vorliegt, kann die Selbstprüfung ohne externe Fachperson
fortgesetzt werden.

Gate vor einer BMD-Kompatibilitätsbehauptung:

- konkretes NTCS-Importziel und Version benennen
- offizielles Schnittstellenhandbuch oder betreibereigene Importvorlage
  versioniert dokumentieren
- Pflichtfelder, Feldlimits und Mapping als eigenen Adapter umsetzen
- eigenes Golden File sowie Positiv- und Negativtests
- wenn eine betreiberkontrollierte NTCS-Umgebung verfügbar ist:
  anonymisierten Testimport protokollieren
- keine eigene Buchungs-, Steuer-, Mahn- oder Zahlungslogik

Eine externe Steuerberater- oder Auditor-Freigabe ist kein Gate. Fehlt der
Zielvertrag oder eine kontrollierte Zielumgebung, bleibt BMD sichtbar
unverifiziert und wird nicht als produktionsbereit bezeichnet.

## RZL FIBU als autorisierter Fast-Follow

RZL beschreibt den Selbstimport in der öffentlichen Online-Hilfe. Das aktuelle
FIBU-Importhandbuch (Stand Juli 2026) ist auffindbar, erklärt seine Nutzung aber
ausdrücklich als berechtigten RZL-Nutzern vorbehalten. Der Zieladapter kann ohne
externe Fachperson entwickelt werden, aber nicht ohne autorisierten
Betreiberzugang oder eine von RZL freigegebene Schnittstellenbeschreibung.

Betreiberprüfung vom 28.07.2026:

- Die öffentliche RZL-Online-Hilfe bestätigt eine fixe RZL-Schnittstelle für
  Sach- und Personenkonten, Salden/Konten, Buchungen, offene Posten und
  Journalzeilen. Sie verweist für die genaue Aufbereitung auf die
  Schnittstellenbeschreibung im Anwenderbereich oder auf einen Bezug bei RZL.
- RZL listet getrennte Beschreibungen für `RZL FIBU Import` und
  `RZL FIBU Next`. Vor einem Adapter muss deshalb auch das konkrete Zielprodukt
  feststehen.
- Das direkt auffindbare Handbuch mit Stand Juli 2026 erklärt sich
  ausdrücklich als nur für berechtigte Benutzerinnen und Benutzer der
  RZL-Programme bestimmt und untersagt Reproduktion, Vervielfältigung oder
  Verbreitung ohne schriftliche Einwilligung.
- Für HAUSV ist derzeit weder eine RZL-Betreiberberechtigung noch eine
  schriftliche Nutzungserlaubnis oder kontrollierte RZL-Zielumgebung
  dokumentiert.

Der öffentlich erreichbare Download allein begründet kein Nutzungsrecht.
HAUSV übernimmt deshalb weder Felddefinitionen noch Prüfsumme oder Beispiele
aus dem geschützten Handbuch und baut keinen geratenen Adapter. Das fehlende
Nutzungsrecht und die Zielauswahl sind technische Inputs, kein
Auditor-Gate. Nach dokumentierter Berechtigung kann die Betreiber-Selbstprüfung
ohne externe Fachperson fortgesetzt werden.

Gate:

- Nutzungsberechtigung und konkretes RZL-Profil samt Quellstand festhalten
- kanonische Exportdaten auf die offiziellen RZL-Felder abbilden
- Pflichtfelder und Limits automatisiert prüfen
- eigenes anonymisiertes Golden File und Negativtests
- Betreiberprotokoll für Parser-/Formatprüfung; kontrollierter Testimport, sobald
  eine RZL-Umgebung verfügbar ist
- keine Buchungslogik im Portal

## ebInterface empfangen, nicht buchen

Entscheidung: ebInterface ist für Dienstleister-Rechnungen relevant, aber nur
als Empfangs- und Ablageformat. Die XML-Datei wird als geschütztes Dokument
abgelegt; Metadaten dienen der Auffindbarkeit und Weitergabe, nicht der Buchung.

Priorität:

- ebInterface 6.0 zuerst
- ebInterface 5.0 als Fallback
- ebInterface 6.1 separat bewerten, sobald ein realer Lieferant dieses Profil
  liefert
- ältere 4.x/3.x-Versionen vorerst nicht unterstützen

Zielverhalten:

- XML gegen Schema/Fixtures prüfen
- Rechnungsmetadaten lesen
- Originaldatei als Dokument oder Anhang ablegen
- Fehlerbericht mit validen und abgelehnten Datensätzen erzeugen
- keine Buchung, kein Zahlungsauftrag, keine Steuerlogik

Quellen fuer das Profil-Gate:

- AUSTRIAPRO/ebInterface-Dokumentation beschreibt den Namespace
  `http://www.ebinterface.at/schema/6p0/` fuer ebInterface 6.0.
- labs.ebinterface.at bietet eine Schema-Pruefung fuer ebInterface 5.0, 6.0 und
  6.1; diese Versionen bleiben die praktische Referenz fuer Testdateien.

Eine externe Freigabe ist nicht erforderlich. Offizielle XSD-/Validator-
Ergebnisse, Repository-Fixtures, Negativtests und ein datiertes
Betreiberprotokoll bilden den Nachweis.

## Peppol, DATEV und ZUGFeRD nachrangig

Peppol bleibt strategisch interessant, wird aber nur über einen akkreditierten
Access-Point-Anbieter umgesetzt, wenn realer Bedarf bestätigt ist. hausv.org
baut keinen eigenen Peppol Access Point.

Betreiberprüfung vom 28.07.2026:

- In Österreich ist die strukturierte E-Rechnung für Lieferanten des Bundes
  verpflichtend; die Einbringung kann unter anderem über Peppol erfolgen.
  HAUSV stellt jedoch keine Rechnungen im Namen einer Verwaltung aus.
- Für den privaten Rechnungseingang ist derzeit kein konkreter
  Peppol-Handelspartner, keine Participant-ID und kein erwartetes
  Nachrichtenvolumen dokumentiert.
- ViDA macht strukturierte E-Rechnungen für grenzüberschreitende
  EU-B2B-Transaktionen ab 01.07.2030 relevant. Das ist ein Beobachtungspunkt,
  aber kein heutiger Produktbedarf.
- Endnutzer senden und empfangen über einen frei gewählten
  Peppol-akkreditierten Service Provider. HAUSV wird weder OpenPeppol-Mitglied
  noch eigener Access Point.

Vorbewerteter Anbieter: ecosio ist in der aktuellen OpenPeppol-Liste als
zertifizierter Access-Point- und SMP-Anbieter geführt. Öffentlich belegt sind
API-Integration, Peppol-Routing, Monitoring, Rollensteuerung, dreimonatige
Standardaufbewahrung sowie ein Art.-28-Vertrag mit Datenrückgabe beim
Vertragsende. Gegen eine Auswahl ohne Bedarf sprechen fehlende öffentliche
Preise und die veröffentlichten Mindestlaufzeiten. Ein Angebot wird deshalb
erst mit realem Fall und Volumen eingeholt.

Rollentrennung:

- ebInterface: lokales Rechnungsformat prüfen und das empfangene Original
  geschützt ablegen
- Peppol: Transport und Teilnehmeradressierung über den akkreditierten Anbieter
- HAUSV: empfangene Rechnung in das kanonische `Invoice`-Modell übernehmen,
  ohne Buchung, Steuerlogik, Zahlung oder eigenes Rechnungsausstellen

Entscheidung: derzeit No-Go. Entblockt wird die Anbieterentscheidung durch
einen benannten Sender oder Empfänger, Sende-/Empfangsrichtung, erwartetes
Monatsvolumen und benötigtes Peppol-Profil.

DATEV und ZUGFeRD sind für die Österreich-first Roadmap nachrangig. Sie werden
nicht umgesetzt, solange kein bestätigter Kundenbedarf vorliegt. Deutscher
Content darf die österreichische Priorisierung nicht treiben.

Entscheidungsvorlage:

| Thema | AT-Relevanz | Aufwand | Entscheidung |
| --- | --- | --- | --- |
| Peppol | mittelfristig strategisch | Anbieterintegration nötig | akkreditierten Anbieter nutzen, nicht selbst AP bauen |
| DATEV | niedrig für AT-Fokus | mittel bis hoch | kein Scope ohne Bedarf |
| ZUGFeRD | niedrig für AT-Fokus | mittel | kein Scope ohne Bedarf |

Primärquellen der Betreiberprüfung:

- OpenPeppol für Endnutzer und Liste zertifizierter Anbieter:
  `https://peppol.org/about/for-end-users/` und
  `https://peppol.org/members/peppol-certified-service-providers/`
- österreichisches Unternehmensserviceportal zur E-Rechnung an die
  öffentliche Verwaltung:
  `https://www.usp.gv.at/themen/steuern-finanzen/umsatzsteuer-ueberblick/weitere-informationen-zur-umsatzsteuer/vorsteuerabzug-und-rechnung/e-rechnung-an-die-oeffentliche-verwaltung.html`
- EU-Kommission zu ViDA:
  `https://taxation-customs.ec.europa.eu/taxation/vat/vat-digital-age-vida_en`
- ecosio Produkt-, Datenschutz- und Vertragsinformationen:
  `https://ecosio.com/en/products/einvoicing/`,
  `https://ecosio.com/en/data-protection/` und
  `https://ecosio.com/app/uploads/2025/10/2025-10-20_agb_ecosio_all-annexes_en.pdf`

## Bankdateien: camt.053 vor camt.054 vor MT940

Entscheidung: camt.053 ist der primäre Importweg für Zahlungsstatus und
Bankbewegungen. camt.054 wird als Zahlungsavis-/Detailabgleich unterstützt,
wenn eine Bank Detailavise getrennt vom Kontoauszug liefert; fachlich bleibt es
derselbe geschützte Statusabgleich. `camt.054.001.08` ist derzeit nur mit dem
Repository-Golden-File getestet; die produktive Freigabe braucht weiterhin die
unten genannten Profil-Nachweise. Für `.02` wird lediglich der Namespace
akzeptiert; ohne eigene Fixture und eigenes Golden File bleibt das Profil
unverifiziert. MT940 bleibt ein bewertbarer Legacy-Fallback,
wird aber erst gebaut, wenn ein echter Kunde nur MT940 liefern kann und
Testdateien freigibt.

Gate für camt.054:

- offizielle ISO-Profilquelle und unterstützte Version dokumentieren
- synthetische Fixture und Golden File je behauptetem Profil
- reale oder anonymisierte Beispiel-Datei je Bankprofil, sobald verfügbar
- Golden File und Importprotokoll
- gleiche Zahlungsreferenz-Regeln wie camt.053
- keine automatische Buchung, nur Status-/Referenzabgleich

Gate für MT940:

- öffentliche Spezifikation oder reale/anonymisierte Beispiel-Datei ohne
  Secrets
- Feldmapping zu hausv.org-Zahlungsreferenzen
- Golden File und Importprotokoll
- keine automatische Buchung, nur Status-/Referenzabgleich

## Kein Zahlungsauftrag in v1

Entscheidung: SEPA-Zahlungsaufträge wie pain.001 oder pain.008 sind nicht Teil
von v1. hausv.org markiert und erklärt Zahlungsstatus, erzeugt aber keine
Überweisungen, Lastschriften oder Zahlungsfreigaben.

Grund:

- höhere Haftung und Freigabeprozesse
- stärkere Bank-/Mandatsanforderungen
- Risiko, die Kommunikationsplattform in Buchhaltung/Zahlungsverkehr zu ziehen

Eine spätere Umsetzung braucht eigene Tickets, Sicherheitsprüfung und einen klar
abgegrenzten Zahlungsprodukt-Scope.
