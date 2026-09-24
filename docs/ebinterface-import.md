# ebInterface-Import

Status: Der produktive, geschützte Vorschau- und Ablageweg ist implementiert.
Verwalter und Admins erreichen ihn im Dokumentenbereich über
`/app/dokumente/rechnungen/import`.

Der ebInterface-Adapter liest Dienstleister-Rechnungen als strukturierte
Metadaten. Vor der Ablage zeigt das Portal Rechnungsnummer, Aussteller,
Empfänger, Betrag, Rechnungs- und Fälligkeitsdatum sowie den Leistungszeitraum.
Erst nach einer ausdrücklichen Bestätigung wird die unveränderte Original-XML
als geschütztes Dokument abgelegt. Das ist ein Empfangs- und Weitergabeweg,
keine Buchhaltung.

## Unterstuetzte Profile

- `http://www.ebinterface.at/schema/5p0/` als ebInterface 5.0
- `http://www.ebinterface.at/schema/6p0/` als ebInterface 6.0

ebInterface 6.1 ist auf der offiziellen Pruefplattform sichtbar, bleibt aber ein
separates Folgeprofil. 4.x und 3.x werden nicht akzeptiert.

## Gelesene Metadaten

- Rechnungsnummer
- Rechnungsdatum
- Rechnungssteller und Empfaenger
- Bruttobetrag und Waehrung
- Faelligkeit
- Leistungszeitraum, falls vorhanden

## Produktablauf

1. Eine Verwaltungsperson wählt genau eine XML-Datei bis 4 MB.
2. Das Portal bindet die Vorschau an das aktuelle Haus und verwirft sie nach
   15 Minuten. Die Datei verlässt den Server nicht.
3. Unterstütztes Profil und erforderliche Rechnungsmetadaten werden geprüft.
   Fehler werden ohne XML-Inhalt verständlich angezeigt.
4. Nach Bestätigung wird das Original im Dokumentenbereich als `Abrechnung`
   mit `verwalter-only`-Sichtbarkeit abgelegt.
5. Ein hausbezogener SHA-256-Nachweis sperrt doppelte Ablagen. Das Import-Ledger
   hält außerdem den Ablagestatus, den festen Dateischlüssel und die ursprünglichen
   Dokumentmetadaten; die Original-XML liegt ausschließlich im Dokumentenspeicher.
   Der Audit-Verlauf enthält Format, Profil, gekürzte Prüfsumme, Rechnungsnummer
   und Dokumentbezug.

Die Ablage erfolgt wiederaufnehmbar: Zuerst wird eine `pending`-Reservierung
dauerhaft gespeichert, dann die vollständige Datei atomar veröffentlicht.
Dokumentmetadaten und `complete` werden anschließend gemeinsam committed.
Nach einem Abbruch kann dieselbe Datei erneut bestätigt werden. Der Versuch
verwendet die bestehende Dokument-ID und die ursprünglichen Metadaten; bis zur
vollständigen Ablage erscheint kein Dokument in der Liste. Gleichzeitige
Bestätigungen werden durch den eindeutigen Ledger-Schlüssel und die
Abschlusstransaktion auf eine Ablage begrenzt. Alle Serverprozesse müssen wie
bisher denselben Dokumentenspeicher verwenden.

Der separate Audit-Verlauf wird nach dem erfolgreichen Commit ergänzt. Ein
Audit-Fehler wird angezeigt, erzeugt bei Wiederholung aber kein zweites Dokument.
Ein Prozessabbruch zwischen Commit und Audit kann den Audit-Eintrag auslassen.

Die Originaldatei ist ausschließlich über authentisierte App-Routen erreichbar.
Bewohner sehen sie weder in der Dokumentliste noch über einen öffentlichen
Datei-Link.

## Reproduzierbarer Betreibercheck

Die synthetischen Repository-Fixtures werden mit
`scripts/validate-ebinterface-fixtures.sh` gegen den offiziellen
ebInterface-Validator geprüft. Das Skript akzeptiert absichtlich keine
Dateiangaben und kann daher keine echte Rechnung versehentlich an den externen
Dienst übertragen. Produktive Uploads werden niemals an diesen Validator
gesendet.

Am 27.07.2026 bestanden beide synthetischen Fixtures die offizielle
Schema-Prüfung:

- `internal/integrations/testdata/ebinterface-5p0.xml`: ebInterface 5.0
  (`SHA-256 193aa847f0d10effa6f29783adcede6c1b188747a878a0b7b112401d291e2043`)
- `internal/integrations/testdata/ebinterface-6p0.xml`: ebInterface 6.0
  (`SHA-256 21a27027b9dc509ef524b7708d86b2b40f63c2b85b9249cf409e93802a9ec33e`)

Der Betreibercheck mit Primärquelle, versionierten Fixtures sowie automatischen
Positiv-, Negativ-, Rollen-, Datenschutz- und Idempotenztests ersetzt die auf
absehbare Zeit nicht verfügbare externe Fachabnahme. Er ist kein Zertifikat und
keine Steuer- oder Rechtsberatung.

## Grenzen

- keine Buchung
- keine Steuerlogik
- keine Zahlungsfreigabe
- keine Zahlungsausloesung
- kein Mahnwesen
- ebInterface 6.1 bleibt ein separates Folgeprofil
