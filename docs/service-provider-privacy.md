# Dienstleister-Zugriff: Betreiber-Selbstprüfung

Stand: 26. Juli 2026, Prüfrevision `2026-07-26`.

hausv.org behandelt externe Dienstleister als eng begrenzte Rolle für ein
konkret zugewiesenes Anliegen. Es gibt auf absehbare Zeit keine verfügbare
externe Datenschutz- oder Rechtsprüfung. Das ist kein heimlicher offener Punkt
mehr: Die Freigabe stützt sich stattdessen auf eine dokumentierte
Betreiber-Selbstprüfung anhand der DSGVO, der österreichischen
Datenschutzbehörde (DSB), des Europäischen Datenschutzausschusses (EDSA) und der
tatsächlich eingesetzten Auftragsverarbeiter.

Diese Selbstprüfung ist **kein Zertifikat und keine unabhängige Rechtsberatung**.
Sie erfüllt die Rechenschaftslogik: Annahmen, Zwecke, Mittel, Risiken,
Schutzmaßnahmen und Prüfauslöser sind nachvollziehbar. Wenn der konkrete Betrieb
von diesen Annahmen abweicht oder ein hohes, nicht ausreichend gemindertes Risiko
entsteht, bleibt die Funktion geschlossen und die zuständige Stelle muss nach
Art. 36 DSGVO die DSB konsultieren.

## Technisches Freigabe-Gate

Der Zugang ist standardmäßig geschlossen. Für eine Aktivierung müssen **beide**
Werte bewusst gesetzt sein:

```text
SERVICE_PROVIDER_ACCESS_ENABLED=true
SERVICE_PROVIDER_ASSESSMENT_VERSION=2026-07-26
```

Ein veraltetes oder fehlendes Prüfdatum hält den Zugang geschlossen. Ohne Gate
werden Zuordnungen, Kontakte, Einladungen, Magic-Link/OIDC-Anmeldung, bestehende
Sitzungen, Benachrichtigungen und Parkplatzrechte vor Daten-, Mail-, Audit- oder
Sitzungsänderungen abgewiesen. Entzug und Deaktivierung bleiben möglich.

Die Einstellung ist eine technische Attestierung des Betreibers. Sie darf erst
gesetzt werden, wenn Verantwortlichen- und Kontaktangaben für das Haus stimmen,
die folgende Bewertung zum realen Auftrag passt und die Hinweise vor der
Einladung bereitgestellt werden.

## Rollenannahme

Die Begriffe sind funktional: Maßgeblich ist, wer tatsächlich Zwecke und Mittel
bestimmt.

- Die Eigentümergemeinschaft beziehungsweise ihre beauftragte Hausverwaltung
  bestimmt Zweck und Inhalt der Hausverwaltung und ist dafür Verantwortliche
  oder gemeinsam Verantwortliche.
- Der technische hausv.org-Betrieb verarbeitet Portal-Fachdaten
  weisungsgebunden. Für einen Betrieb für Dritte ist eine Vereinbarung nach
  Art. 28 DSGVO samt technischen und organisatorischen Maßnahmen erforderlich.
- Ein Handwerks-/Dienstleistungsbetrieb erhält Daten zur Durchführung seines
  konkreten Auftrags. Ob er für einzelne Verarbeitungsschritte eigener
  Verantwortlicher oder Auftragsverarbeiter ist, ergibt sich aus dem realen
  Vertrag, nicht aus der Rollenbezeichnung im Portal.
- Die für Hosting, Webzugriff, Identität, Mail und Sicherungen eingesetzten
  Anbieter werden je Betrieb dokumentiert und vertraglich geprüft. Das Portal
  selbst setzt keinen bestimmten Anbieter voraus.

## Zwecke und Rechtsgrundlagen der Selbstprüfung

| Vorgang | Zweck | Betreiberbewertung |
|---|---|---|
| Einladung, Anmeldung, Rolle | sichere Identifikation und Zugriff | Art. 6 Abs. 1 lit. b, c oder f DSGVO – abhängig von Vertrags-/Verwaltungsbeziehung |
| Zuweisung eines Anliegens | erforderliche Weitergabe zur Beauftragung und Mangelbehebung | Art. 6 Abs. 1 lit. b oder f; nur das einzelne offene Anliegen |
| Kommentare, Fotos, Angebot, Termin | Auftragsklärung und Dokumentation | gleiche Grundlage wie das Anliegen; keine besondere Datenkategorie beabsichtigt |
| Audit | Zugriffsschutz, Fehlerklärung und Rechtsverteidigung | Art. 6 Abs. 1 lit. f, gegebenenfalls lit. c |
| Transaktionsmail | Einladung, Anmeldung, notwendige Benachrichtigung | akzessorisch zum jeweiligen Hauptzweck |

Einwilligung ist nicht die pauschale Grundlage des normalen Hausbetriebs.
Gesundheitsdaten, Ausweiskopien und andere Daten nach Art. 9 oder 10 DSGVO sind
nicht Zweck des Systems. Nutzerhinweise untersagen solche Inhalte in Freitext
und Bildern; versehentlich eingestellte Inhalte sind nach Meldung zu entfernen.

## DPO- und DSFA-Vorabprüfung

Für den aktuellen Einzelhaus-Pilot gibt es keine Behörde, keine umfangreiche
systematische Überwachung, kein Profiling, keine automatisierte Entscheidung und
keine beabsichtigte umfangreiche Verarbeitung besonderer Datenkategorien.
Deshalb ist nach der dokumentierten Vorabprüfung weder ein verpflichtender
Datenschutzbeauftragter nach Art. 37 Abs. 1 noch eine zwingende DSFA nach den
bekannten Art.-35-/DSFA-V-Kriterien ersichtlich.

Diese Einschätzung wird neu geprüft bei mehreren Häusern in erheblichem Umfang,
systematischer Verhaltensanalyse, KI-Auswertung von Inhalten, Standort- oder
Bildüberwachung, besonderen Datenkategorien, neuen Datenempfängern oder einem
Sicherheitsvorfall mit erhöhtem Risiko.

## Sichtbarkeit und technische Minimierung

Ein Dienstleister sieht ausschließlich offene Anliegen, die seiner
normalisierten E-Mail-Adresse zugewiesen sind:

- Titel, Kategorie, Priorität, Status und notwendiger Ort
- Beschreibung, Kommentare und geschützte Anhänge dieses Anliegens
- eigener Statusbeitrag, Terminvorschlag und Kostenschätzung

Hausübersicht, Bewohner-/Benutzerlisten, allgemeine Dokumente, Abstimmungen,
Parkplatzbereich, Einstellungen und fremde oder geschlossene Anliegen bleiben
unsichtbar. Entzug oder Schließen wirkt sofort. Dateiabrufe prüfen Haus, Sitzung,
Rolle und konkretes Objekt; negative Zugriffstests sichern diese Grenzen.

## Auftragsverarbeiter und Drittland

Hosting, Webschutz, Identitätsdienst, Mailversand und verschlüsselte Sicherungen
sind betreiberspezifisch. Vor Freigabe werden die tatsächlich eingesetzten
Anbieter, Verarbeitungsregionen, Aufbewahrungsfristen, DPAs, TOMs,
Unterauftragnehmer und allfällige Drittlandtransfers dokumentiert. Die Vorlage
dafür ist das
[`Art.-28-/TOM-Freigabepaket`](processor-approval-template.md).

Ein Anbieterwechsel oder neuer Drittlandtransfer schließt das technische Gate,
bis Vertrag, Risikoprüfung und öffentliche Datenschutzinformation aktualisiert
sind.

Das Web-Frontend lädt keine externen Schriften, Analyse- oder Werbeskripte.

## Information und Betroffenenrechte

`/<portal>/datenschutz` ist vor der Anmeldung öffentlich erreichbar und wird in
Einladungs- und Magic-Link-Mails verlinkt. Die Seite nennt Zwecke, Kategorien,
Empfänger, Drittlandtransfer, Aufbewahrung, Kontakte, Gate-Status und Rechte.
Anfragen nach Art. 12 bis 22 DSGVO gehen an den dort genannten Hauskontakt; der
technische Betrieb unterstützt bei Export, Berichtigung, Einschränkung und
Löschung.

## Aufbewahrung

- Magic-Link: 15 Minuten, einmalig, nur im Arbeitsspeicher.
- OIDC-Flow: 10 Minuten; Sitzung regulär höchstens 30 Tage.
- Mitgliedschaft/Dienstleisterprofil: bis Entzug; ohne weitere
  Hausmitgliedschaft wird die nicht mehr benötigte Person entfernt.
- Geschlossene Anliegen, Kommentare und Anhänge: jährliche Prüfung, regulär
  drei Jahre ab Abschluss; länger nur bei dokumentierter offener
  Gewährleistungs-, Rechts- oder Nachweispflicht. Bis zur automatischen
  Löschfunktion ist dies ein dokumentierter manueller Betreiberprozess.
- Anhangdatei bei Löschung sofort; Tombstone nach einem Jahr.
- Audit-Live-Datei rotiert nach 10 MiB, 90 Tagen oder 20.000 Einträgen.
  Auditdaten werden nach drei Jahren zur Löschung fällig und beim nächsten
  sechsstündlichen Bereinigungslauf aus Live-Datei und Archiven entfernt.
- Betriebsabhängige Mail- und Metadatenfristen werden in der öffentlichen
  Datenschutzinformation des jeweiligen Betreibers ausgewiesen.

## Wiederholbare Betreiberentscheidung

Vor `SERVICE_PROVIDER_ACCESS_ENABLED=true` wird dokumentiert:

1. Hauskontakt und Verantwortlichenrolle sind für den realen Betrieb bestätigt.
2. Vertragliche Rolle des beauftragten Dienstleisters passt zur Datenweitergabe.
3. Art.-28-Vereinbarung/TOMs für den technischen Betrieb sind vorhanden, soweit
   der Betrieb für eine andere verantwortliche Stelle erfolgt.
4. Unterauftragsverarbeiter, kontobezogene AVV-/DPA-Nachweise,
   Transfermechanismen und aktuelle Subprozessoren wurden geprüft.
5. Einladung verweist auf `/<portal>/datenschutz`; Freitext-/Foto-Hinweise sind sichtbar.
6. Es gibt keine beabsichtigten Art.-9-/10-Daten oder ein anderes
   DSFA-Hochrisikomerkmal.
7. Lösch- und Betroffenenprozess ist organisatorisch zugewiesen.
8. Go/No-Go mit Datum, Haus und Prüfrevision ist im PPM-Ticket dokumentiert.

Die Prüfung wird mindestens jährlich und bei jedem genannten Prüfauslöser
wiederholt. Eine externe Prüfung bleibt willkommen, ist aber kein vorgespieltes
oder unerreichbares Abnahmekriterium.

## Primär- und Betreiberquellen

- [DSGVO, Verordnung (EU) 2016/679](https://eur-lex.europa.eu/eli/reg/2016/679/oj)
- [DSB: Pflichten von Verantwortlichen](https://dsb.gv.at/rechte-pflichten/ihre-pflichten-als-verantwortlicher)
- [DSB: Pflichten von Auftragsverarbeitern](https://dsb.gv.at/rechte-pflichten/ihre-pflichten-als-auftragsverarbeiterin)
- [DSB: Rechte betroffener Personen](https://dsb.gv.at/rechte-pflichten/ihre-rechte-als-betroffene-person)
- [EDSA: Accountability](https://www.edpb.europa.eu/topics/accountability-and-compliance-tools/accountability_en)
- [EDSA: Guidelines 07/2020 zu Verantwortlichen/Auftragsverarbeitern](https://www.edpb.europa.eu/our-work-tools/our-documents/guidelines/guidelines-072020-concepts-controller-and-processor-gdpr_en)
