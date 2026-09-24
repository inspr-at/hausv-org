# camt.053-Import

Der camt.053-Adapter liest Kontoauszüge als Statusquelle. Er bucht nichts und
legt keine Zahlungsaufträge an.

Verwalter und Admins starten den tenantgebundenen Produktfluss unter
`/app/settings/payments/import`: Monat wählen, XML-Datei bis 2 MiB prüfen und
erst danach eindeutige Treffer übernehmen. Die Datei selbst wird nicht
gespeichert. Eine Vorschau lebt höchstens 15 Minuten im Arbeitsspeicher und
enthält weder IBAN noch Debitorenname oder Verwendungszweck.

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

## Sichere Übernahme

- Der Upload ist rollen-, haus- und Origin-gebunden.
- Eine Vorschau speichert nur datensparsame Normaldaten im Arbeitsspeicher.
- Beim Übernehmen werden Referenzen und Einheiten erneut berechnet. Änderungen
  seit der Vorschau verhindern eine stille Übernahme.
- Nur eindeutige Treffer ändern den manuellen Zahlungsstatus.
- Ein mandantenbezogenes Digest-Ledger verhindert die doppelte Übernahme
  derselben Datei, ohne die Bankdatei abzulegen.
- Ledger-Reservierung, sämtliche Statusänderungen und Ergebniszahlen werden
  in einer gemeinsamen Datenbanktransaktion gespeichert. Ein später Fehler
  verwirft die gesamte Übernahme; ein erneuter Versuch bleibt möglich. Der
  eindeutige Datenbankschlüssel schützt auch bei mehreren Serverprozessen.
- Der Audit-Eintrag enthält Profil, gekürzten Digest und Summen, aber keine
  Bank- oder Personendaten.
- Der separate Audit-Verlauf wird nach dem Commit ergänzt. Ein Fehler dort
  wird protokolliert und hebt die dauerhafte Importsperre nicht auf. Ein
  Prozessabbruch zwischen Commit und Audit kann einen Audit-Eintrag auslassen.

## Reproduzierbarer Betreiber-Nachweis

Die Selbstprüfung vom 27.07.2026 verwendet die offiziellen ISO-20022-
Nachrichtenlisten und das Nachrichtenarchiv als Primärquellen:

- `https://www.iso20022.org/iso-20022-message-definitions`
- `https://www.iso20022.org/catalogue-messages/iso-20022-messages-archive`

Versionierte synthetische Golden Files decken `.001.02` und `.001.08` ab.
Positive, negative, rollenbezogene, mandantenbezogene, Größen-, Datenschutz-,
Idempotenz- und Browser-Flows laufen reproduzierbar mit:

```sh
nix-shell -p go_1_26 --run 'go test ./internal/integrations ./internal/server ./internal/db'
```

Eine reale oder anonymisierte Datei einer konkreten Bank erweitert später die
Profilabdeckung, ist aber kein externes Human-Gate. Die Selbstprüfung ist kein
Zertifikat und keine Steuer- oder Rechtsberatung.
