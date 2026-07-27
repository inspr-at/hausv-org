# ebInterface-Import

Status: Parser und geschuetzte Ablage sind intern implementiert und getestet,
aber noch nicht an eine produktive Upload-UI angeschlossen. Offizielle
Schema-Validierung und der reproduzierbare Betreiber-Check aus
`docs/interface-qa.md` bleiben das Freigabe-Gate.

Der ebInterface-Adapter kann Dienstleister-Rechnungen als strukturierte
Metadaten lesen und die Original-XML ueber den internen Speicherhelfer als
geschuetztes Dokument ablegen. Das ist ein Empfangs- und Weitergabeweg, keine
Buchhaltung.

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

Der interne Speicherhelfer legt die XML-Datei im Dokumentenbereich als
`Abrechnung` mit `verwalter-only`-Sichtbarkeit ab. Damit bleibt sie ueber
geschuetzte App-Routen erreichbar und wird nicht als oeffentlicher Datei-Link
geteilt. Ein produktiver Importweg ruft diesen Helfer derzeit noch nicht auf.

## Grenzen

- keine Buchung
- keine Steuerlogik
- keine Zahlungsfreigabe
- keine Zahlungsausloesung
- kein Mahnwesen
