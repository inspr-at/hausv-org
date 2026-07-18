# BMD-Rohdaten: Steuerberater-Pruefpaket

Dieses Dokument ist das Arbeitsblatt fuer die externe BMD-NTCS-Pruefung. Es
beschreibt bewusst kein fertiges Buchungsformat, sondern ein Kandidatenformat
fuer strukturierte Rohdaten, die ein Steuerberater fachlich einordnen kann.

Status: `raw-v0`, nicht produktiv freigeschaltet.

## Ziel

hausv.org soll Informationen sauber weitergeben koennen, ohne selbst Buchhaltung,
Steuerlogik, Mahnungen oder Zahlungsauftraege zu erzeugen. Der BMD-Schritt ist
deshalb eine Rohdaten-/Beleguebergabe:

- Haus/Mandant und interne Referenz bleiben nachvollziehbar.
- Einheit, Zeitraum, Betrag und Status werden maschinenlesbar uebergeben.
- Belege werden referenziert, aber nicht oeffentlich verlinkt.
- Soll-/Haben-Konten, Steuerkennzeichen und Buchungslogik bleiben beim
  Steuerberater oder in BMD NTCS.

## Kandidatenformat `raw-v0`

Datei: UTF-8 CSV mit Semikolon als Trennzeichen.

Golden File: `internal/integrations/testdata/bmd-raw-v0.csv`

| Feld | Bedeutung | Pflicht |
| --- | --- | --- |
| `tenant_slug` | Haus/Mandant in hausv.org | ja |
| `record_id` | stabile interne Datensatz-ID | ja |
| `kind` | fachlicher Datensatztyp, z. B. `payment-status` | ja |
| `occurred_at` | fachliches Datum im Format `YYYY-MM-DD` | ja |
| `unit_id` | Einheit/Top/Stellplatzbezug, falls vorhanden | nein |
| `person_ref` | interne Personenreferenz, falls erforderlich | nein |
| `reference` | hausv.org Zahlungs- oder Vorgangsreferenz | nein |
| `amount_currency` | ISO-Waehrung, z. B. `EUR` | bei Betrag |
| `amount_cents` | Betrag in Cent als Ganzzahl | bei Betrag |
| `amount_decimal` | lesbarer Betrag mit Dezimalkomma | bei Betrag |
| `period` | Zeitraum, z. B. `2026-06` | je Datentyp |
| `status` | fachlicher Status, z. B. `open`, `paid`, `overdue` | je Datentyp |
| `source` | Modulquelle, z. B. `parking` | empfohlen |
| `document_reference` | geschuetzte interne Dokument-/Belegreferenz | optional |
| `description` | kurze fachliche Beschreibung | optional |
| `verification_note` | Hinweis fuer Testimport und Abstimmung | optional |

Nicht erlaubt im Kandidatenformat:

- `debit_account`, `credit_account`, `account`, `contra_account`
- `konto`, `gegenkonto`
- `tax_code`, `vat_code`

Diese Felder wuerden aus Rohdaten faktisch Buchungssaetze machen. Wenn BMD NTCS
sie zwingend braucht, muss der Steuerberater festlegen, ob sie im Zielsystem,
in einer Mapping-Tabelle oder in einem spaeteren explizit freigegebenen Adapter
entstehen.

## Fragen an den Steuerberater

1. Kann BMD NTCS diese Rohdaten als CSV fuer eine manuelle oder halbautomatische
   Weiterverarbeitung importieren?
2. Welches konkrete Importziel ist fachlich richtig: Beleguebergabe,
   Fremdsystemdaten, Stapelvorerfassung oder ein anderer NTCS-Bereich?
3. Welche Felder sind Pflicht, bevor ein Testimport sinnvoll ist?
4. Welche Felder duerfen leer bleiben, wenn hausv.org nur Transparenzstatus
   liefert?
5. Duerfen Zahlungsreferenz und interne Dokumentreferenz als stabile
   Nachverfolgungsschluessel verwendet werden?
6. Sind Personen- oder Einheitsreferenzen in dieser Form ausreichend, oder soll
   eine andere ID aus BMD zurueckgespielt werden?
7. Welche Feldlaengen, Zeichensaetze und Datumsformate sind fuer den Import
   verbindlich?

## Abnahme-Gate

`HAUSV-91` darf erst geschlossen werden, wenn diese Punkte dokumentiert sind:

- Export-Scope bestaetigt: Rohdaten/Beleguebergabe, keine Buchungssaetze.
- Feldmapping mit realem Steuerberater geprueft.
- BMD-NTCS-Importziel und Pflichtfelder dokumentiert.
- Golden File mit Steuerberater-Feedback aktualisiert.
- Testimport oder begruendeter Nicht-Import dokumentiert.
- Keine eigene Buchungs-, Steuer-, Mahn- oder Zahlungslogik im Produkt.

Bis dahin bleibt BMD als `raw-v0` Kandidat intern testbar, aber nicht als
produktive Exportfunktion freigeschaltet.
