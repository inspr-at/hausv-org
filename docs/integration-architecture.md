# Schnittstellen-Architektur

hausv.org bleibt ein Kommunikations- und Transparenzportal. Schnittstellen
importieren oder exportieren strukturierte Statusdaten, Rechnungen und Nachweise,
aber keine Buchungssätze mit eigener Buchungslogik.

## Kanonische Modelle

Die App trennt Formatadapter von Produktdaten über kleine interne Modelle:

- `canonicalPayment`: Zahlungseingang aus Kontoauszügen oder Avise-Dateien, z. B.
  camt.053 oder camt.054. Enthält
  Betrag, Währung, Buchungs-/Valutadatum, Referenz, optional Debitor-Text und
  Herkunft. Die Zuordnung erfolgt über eine saubere Zahlungsreferenz.
- `canonicalInvoice`: empfangene Dienstleisterrechnung oder Kostennachweis, z. B.
  ebInterface. Enthält Rechnungsnummer, Aussteller, Betrag, Daten und optional
  eine geschützte Attachment-ID.
- `canonicalExportRecord`: formatneutrale Rohdaten für kontrollierte
  CSV-Übergaben und spätere BMD-/RZL-Zieladapter. Das
  Modell enthält fachliche Felder und Statusinformationen, aber keine Soll/Haben-
  Buchungsvorgaben.

## Adapter-Grenzen

Adapter hängen an Interfaces für Zahlungsimport, Rechnungsempfang und Rohdaten-
Export. Sie erhalten ein `integrationSource` mit Format/Version/Dateiname und
geben pro Datensatz entweder ein kanonisches Modell oder einen strukturierten
Fehler zurück.

So bleiben camt, BMD, RZL und ebInterface austauschbar. Das neutrale
`manual-csv/raw-v0` ist ausdrücklich kein BMD- oder RZL-Importformat.
Formatbesonderheiten, Schemas und Golden Files gehören in den jeweiligen
Zieladapter, nicht in die Portal-Workflows.

## Fehlerberichte

Imports sollen nicht unnötig komplett abbrechen. Wo möglich wird pro Datensatz
entschieden:

- akzeptiert und in einen Status-/Dokumentationsworkflow übergeben
- abgelehnt mit Feld, Datensatz-ID und verständlicher Meldung
- unklar und manuell zu prüfen, falls spätere Adapter diese Kategorie brauchen

## Produktgrenze

Keine Schnittstelle darf im Produkt eigene Buchhaltung, Mahnwesen, Steuerlogik
oder Zahlungsaufträge einführen. Zahlungsdaten sind Transparenzstatus. Exporte
liefern strukturierte Rohdaten für kontrollierte Übergaben; Kompatibilität mit
einem Zielsystem wird nur für einen eigens dagegen geprüften Adapter behauptet.

Zahlungsabgleich verwendet das kurze Referenzformat aus
`docs/payment-references.md`; Freitext-Verwendungszwecke sind nur Zusatzkontext.
Die ersten Zahlungsadapter sind in `docs/camt053-import.md` und
`docs/camt054-evaluation.md` beschrieben. Der Rechnungsadapter ist in
`docs/ebinterface-import.md` beschrieben. Das neutrale Übergabeformat liegt in
`docs/structured-handoff-export.md`. Die
gemeinsamen QA-Gates liegen in `docs/interface-qa.md`.
