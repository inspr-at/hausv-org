# Strukturierte Rohdatenübergabe `raw-v0`

Dieses Dokument beschreibt ein formatneutrales Übergabe-CSV. Es ist weder ein
BMD- noch ein RZL-Importformat und behauptet keine Kompatibilität mit einem
Buchhaltungssystem.

Status: produktiv als geschützte Portalstrecke implementiert und mit Golden
File, Paket-, Rollen-, Mandanten- und Audit-Tests geprüft.

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

## Portalstrecke

Die Verwaltung erreicht den Export unter
`/app/settings/data-export`. Der Ablauf erzwingt eine bewusste Auswahl:

1. Eine berechtigte Person wählt einzelne Datenbereiche. Nichts ist
   vorausgewählt.
2. Das Portal erzeugt eine Vorschau mit Datenbereichen, Datensatzanzahl und
   CSV-Prüfsumme.
3. Erst die zweite Aktion lädt ein einmaliges ZIP-Paket herunter.

Die Vorschau liegt höchstens 15 Minuten im Arbeitsspeicher, ist an Mandant und
Person gebunden und wird nach einem erfolgreichen Download gelöscht. Der
Download wird im Aktivitätsverlauf protokolliert. Das Audit enthält Format,
Version, gewählte Bereiche, Anzahl und gekürzte Prüfsummen, aber keine
exportierten Fachdaten.

### Freigegebene Datenbereiche

| Datenbereich | Berechtigung | Enthalten |
| --- | --- | --- |
| Zahlungsstatus der Einheiten | Hausverwaltung und Admin | Einheit, normalisierter Status, Änderungsdatum |
| Parkplatz-Monatsabrechnungen | nur Admin | Zeitraum, Gesamtbetrag, Status, optionale Zahlungsreferenz |

Eigentümer-, Mieter- und E-Mail-Listen werden nicht übernommen. Beim
Einheitenstatus ist eine Personenreferenz deshalb bewusst leer. Weitere
Portalmodule müssen einzeln freigegeben, abgebildet und getestet werden; ein
generischer Datenbankabzug ist nicht vorgesehen.

## ZIP-Paket und Manifest

Das Paket enthält genau:

- `hausv-raw-v0.csv`
- `manifest.json`

Das Manifest trägt die stabile Schema-Kennung
`hausv.raw-export-manifest` und enthält:

- `format`: `manual-csv`
- `format_version`: `raw-v0`
- Erzeugungszeitpunkt und Mandant
- bewusst ausgewählte Datenbereiche
- angenommene und abgelehnte Datensatzanzahl
- Dateiname und SHA-256-Prüfsumme des CSV
- `target_system_compatibility: false`
- den Zweck als neutrale Rohdatenübergabe ohne Buchungs- oder Steuerlogik

### Betreiber-Stichprobe

Eine reproduzierbare Stichprobe besteht aus:

1. Vorschau für einen einzelnen Datenbereich erstellen.
2. Angezeigte Anzahl mit dem Portalbestand vergleichen.
3. ZIP entpacken und die SHA-256-Prüfsumme des CSV mit `manifest.json`
   vergleichen.
4. CSV in einem Standard-Tabellenprogramm öffnen und Spalten, Umlaute,
   Semikolon und Dezimaldarstellung prüfen.
5. Aktivitätsverlauf auf genau einen Export-Eintrag prüfen.
6. Erneuten Download mit demselben Token ablehnen lassen.

Diese Betreiberprüfung ersetzt keine behauptete Drittfreigabe. Sie ist die
real verfügbare, dokumentierte Evidenz, solange kein externer Auditor zur
Verfügung steht.
