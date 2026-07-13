# camt.053-Import

Der camt.053-Adapter liest Kontoauszüge als Statusquelle. Er bucht nichts und
legt keine Zahlungsaufträge an.

## Unterstützte Profile

- `camt.053.001.02` für ältere SEPA-/ISO-2009-Dateien
- `camt.053.001.08` für neuere ISO-2019-Dateien

Andere Namespaces werden als Importbericht mit Datensatzfehler abgelehnt. Die
Adaptergrenze ist so angelegt, dass offizielle XSD-Dateien pro Profil ergänzt
werden können, ohne die Produktmodelle zu ändern. Aktuell validiert der Adapter
Namespace, Pflichtfelder, Datum, Währung, Betrag und Eingang/Ausgang pro Eintrag
und nutzt Golden Files für beide unterstützten Profile. Das verbindliche QA-Gate
für weitere Formatfreigaben steht in `docs/interface-qa.md`.

## Zuordnung

Für automatische Zuordnung wird zuerst eine gültige hausv.org-Zahlungsreferenz
gesucht. Der Adapter prüft `EndToEndId`, weitere Referenzen und unstrukturierte
Verwendungszwecke. Freitext allein erzeugt keinen sicheren Treffer.
Die Statusübernahme ist separat in
[`camt-payment-status-reconciliation.md`](camt-payment-status-reconciliation.md)
dokumentiert.

## Fehlerbehandlung

Ein fehlerhafter Eintrag bricht den Import nicht unnötig ab. Der Importbericht
zählt akzeptierte und abgelehnte Einträge und enthält Feld, Datensatz-ID und eine
verständliche Meldung. Ausgehende Buchungen (`DBIT`) werden nicht als Zahlung
importiert.
