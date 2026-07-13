# camt.054-Zahlungsavise

camt.054 ist fuer hausv.org kein eigenes Finanzmodul, sondern ein optionaler
Detailkanal fuer Zahlungsavise. camt.053 bleibt der primaere Kontoauszug-Import.
camt.054 wird genutzt, wenn eine Bank Detailavise getrennt liefert und diese
Avise eine saubere hausv.org-Zahlungsreferenz enthalten.

## Entscheidung

- camt.054 wird als Zahlungsstatusquelle bewertet und technisch an dasselbe
  `canonicalPayment`-Modell angebunden wie camt.053.
- Unterstützte Profile: `camt.054.001.02` und `camt.054.001.08`.
- Der erste Golden-File-Test deckt `camt.054.001.08` ab.
- Der Adapter liest nur eingehende `CRDT`-Einträge. Ausgehende oder unklare
  Einträge werden als Datensatzfehler gemeldet.

## Nicht im Scope

- keine Buchung
- keine Sollstellung
- keine Mahnung
- keine Zahlungsaufträge
- kein Ersatz fuer BMD/RZL oder Steuerberater-Systeme

## Abnahme-Gates

Vor einer produktiven UI-Freigabe fuer Bankdatei-Uploads muessen pro aktivem
Bankprofil diese Nachweise vorliegen:

- anonymisierte reale Datei oder verifizierte Bank-Testdatei
- Golden-File-Test mit akzeptierten und abgelehnten Datensaetzen
- Profil-/Namespace-Pruefung gegen das erwartete camt.054-Profil
- Zahlungsreferenz-Abgleich ueber `docs/payment-references.md`
- Importbericht ohne Debitor-/IBAN-Leaks in Bewohner- oder Statusansichten
