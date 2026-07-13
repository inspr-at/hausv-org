# ebInterface-Import

Der ebInterface-Adapter liest Dienstleister-Rechnungen als strukturierte
Metadaten und legt die Original-XML als geschuetztes Dokument ab. Das ist ein
Empfangs- und Weitergabeweg, keine Buchhaltung.

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

## Ablage

Die XML-Datei wird im Dokumentenbereich als `Abrechnung` mit
`verwalter-only`-Sichtbarkeit gespeichert. Damit bleibt sie ueber geschuetzte
App-Routen erreichbar und wird nicht als oeffentlicher Datei-Link geteilt.

## Grenzen

- keine Buchung
- keine Steuerlogik
- keine Zahlungsfreigabe
- keine Zahlungsausloesung
- kein Mahnwesen
