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

## BMD zuerst

Entscheidung: BMD wird als erste Kanzlei-/Steuerberater-Schnittstelle behandelt.
Der Scope ist Rohdaten-/Belegübergabe, nicht fertige Buchungssätze.

Geplante Rohdaten:

- Tenant/Haus, Einheit, Zeitraum und interne Referenz
- Zahlungsstatus und Zahlungsreferenz
- Dokument-/Anhangsreferenzen zu Belegen oder Dienstleister-Rechnungen
- Beträge nur als Übergabewerte, nicht als gebuchte Soll/Haben-Sätze

Interner Kandidat:

- Profil `raw-v0` als Semikolon-CSV fuer die Steuerberater-Pruefung
- Golden File `internal/integrations/testdata/bmd-raw-v0.csv`
- Pruefpaket `docs/bmd-rawdata-verification.md`
- Konten- und Steuerfelder werden vor externer BMD-NTCS-Bestaetigung abgelehnt

Abnahme-Gate vor produktiver Freigabe:

- Feldmapping mit einem realen Steuerberater prüfen
- Kontenbedarf und Importziel in BMD NTCS festhalten
- vorhandenes Golden File mit dem bestaetigten Mapping aktualisieren
- Testimport in BMD NTCS dokumentieren

Ohne dieses Gate wird kein BMD-Export als produktive Funktion freigeschaltet.

## RZL als Fast-Follow

Entscheidung: RZL wird nach der BMD-Klärung geprüft. Der bevorzugte Weg ist,
dieselben kanonischen Export-Rohdaten zu verwenden und nur den Adapter zu
wechseln.

Zu klären:

- ob RZL die BMD-Rohdatenstruktur direkt oder mit kleinem Mapping akzeptiert
- welche Pflichtfelder/Kontenfelder abweichen
- ob Zielkunden RZL tatsächlich benötigen

Falls RZL umgesetzt wird, gelten dieselben Gates wie bei BMD: Golden File,
Testimport und keine Buchungslogik im Produkt.

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

## Peppol, DATEV und ZUGFeRD nachrangig

Peppol bleibt strategisch interessant, wird aber erst umgesetzt, wenn ein
Access-Point-Ansatz und realer Bedarf bestätigt sind.

DATEV und ZUGFeRD sind für die Österreich-first Roadmap nachrangig. Sie werden
nicht umgesetzt, solange kein bestätigter Kundenbedarf vorliegt. Deutscher
Content darf die österreichische Priorisierung nicht treiben.

Entscheidungsvorlage:

| Thema | AT-Relevanz | Aufwand | Entscheidung |
| --- | --- | --- | --- |
| Peppol | mittelfristig strategisch | hoch, Access Point nötig | beobachten, nicht bauen |
| DATEV | niedrig für AT-Fokus | mittel bis hoch | kein Scope ohne Bedarf |
| ZUGFeRD | niedrig für AT-Fokus | mittel | kein Scope ohne Bedarf |

## Bankdateien: camt.053 vor camt.054 vor MT940

Entscheidung: camt.053 ist der primäre Importweg für Zahlungsstatus und
Bankbewegungen. camt.054 wird als Zahlungsavis-/Detailabgleich unterstützt,
wenn eine Bank Detailavise getrennt vom Kontoauszug liefert; fachlich bleibt es
derselbe geschützte Statusabgleich. `camt.054.001.08` ist derzeit nur mit dem
Repository-Golden-File getestet; die produktive Freigabe braucht weiterhin die
unten genannten Bankprofil-Nachweise. Für `.02` wird lediglich der Namespace
akzeptiert; ohne eigene Fixture und eigenes Golden File bleibt das Profil
unverifiziert. MT940 bleibt ein bewertbarer Legacy-Fallback,
wird aber erst gebaut, wenn ein echter Kunde nur MT940 liefern kann und
Testdateien freigibt.

Gate für camt.054:

- reale oder anonymisierte Beispiel-Datei je Bankprofil
- Golden File und Importprotokoll
- gleiche Zahlungsreferenz-Regeln wie camt.053
- keine automatische Buchung, nur Status-/Referenzabgleich

Gate für MT940:

- reale Beispiel-Datei ohne Secrets
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
