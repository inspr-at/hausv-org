# Strukturierte Rohdatenübergabe `raw-v0`

Dieses Dokument beschreibt ein formatneutrales Übergabe-CSV. Es ist weder ein
BMD- noch ein RZL-Importformat und behauptet keine Kompatibilität mit einem
Buchhaltungssystem.

Status: intern implementiert und mit Golden File geprüft; noch ohne produktive
Export-UI.

## Ziel

hausv.org soll Informationen sauber weitergeben koennen, ohne selbst Buchhaltung,
Steuerlogik, Mahnungen oder Zahlungsauftraege zu erzeugen. `raw-v0` ist deshalb
eine kontrollierte Rohdaten-/Beleguebergabe:

- Haus/Mandant und interne Referenz bleiben nachvollziehbar.
- Einheit, Zeitraum, Betrag und Status werden maschinenlesbar uebergeben.
- Belege werden referenziert, aber nicht oeffentlich verlinkt.
- Soll-/Haben-Konten, Steuerkennzeichen und Buchungslogik bleiben außerhalb von
  hausv.org.

## Kandidatenformat `raw-v0`

Datei: UTF-8 CSV mit Semikolon als Trennzeichen.

Golden File:
`internal/integrations/testdata/structured-handoff-raw-v0.csv`

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
| `verification_note` | Hinweis zur kontrollierten Übergabe | optional |

Nicht erlaubt im Kandidatenformat:

- `debit_account`, `credit_account`, `account`, `contra_account`
- `konto`, `gegenkonto`
- `tax_code`, `vat_code`

Diese Felder wuerden aus Rohdaten faktisch Buchungssaetze machen und sind daher
kein Bestandteil des neutralen Formats. Benötigt ein Zielsystem solche Felder,
braucht es einen eigenen Adapter mit dokumentiertem Zielvertrag und klarer
Verantwortung außerhalb der Portal-Workflows.

## Betreiber-Nachweis

Der Nachweis ist ohne externe Prüfstelle reproduzierbar:

- Adaptertests prüfen Header, Semikolon, UTF-8-Inhalte, Dezimaldarstellung und
  deterministische Feldreihenfolge.
- Das Golden File enthält keine realen Personen- oder Kontodaten.
- Tests lehnen Konten-, Gegenkonten- und Steuerfelder sowie unbekannte Felder ab.
- Der Exportbericht benennt angenommene und abgelehnte Datensätze.
- Eine manuelle Stichprobe prüft, dass das CSV in einem Standard-
  Tabellenprogramm lesbar ist und keine BMD-/RZL-Kompatibilität behauptet.

## Gate für spätere Zieladapter

Ein BMD- oder RZL-Adapter ist erst fertig, wenn:

- der offizielle Zielvertrag oder eine vom Betreiber kontrollierte
  Importvorlage mit Version und Quelle dokumentiert ist,
- Eingabe, Ausgabe, Pflichtfelder, Feldlimits und Nicht-Ziele festgehalten sind,
- ein eigenes Golden File und automatisierte Negativtests existieren,
- soweit ein Betreiberzugang verfügbar ist, ein kontrollierter Testimport mit
  anonymisierten Daten protokolliert ist,
- anderenfalls die fehlende Zielumgebung sichtbar bleibt und der Adapter nicht
  als produktionsbereit bezeichnet wird.

Eine externe Steuerberater- oder Auditor-Freigabe ist dafür nicht erforderlich
und wird nicht vorgetäuscht. Offizielle Spezifikation, reproduzierbare Tests und
Betreiberprotokoll bilden den Nachweis.
