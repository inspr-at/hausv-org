# camt-Abgleich und Zahlungsstatus

Status: Parser, Abgleich und Statusübernahme sind als geschützter Vorschau- und
Übernahmefluss unter `/app/settings/payments/import` implementiert und
reproduzierbar getestet. Die Betreiber-Selbstprüfung ist in
`docs/camt053-import.md` dokumentiert.

Der camt.053-Adapter liefert intern `canonicalPayment`-Datensätze. Die
Zuordnung zum Portal passiert erst danach über bekannte Zahlungsreferenzen pro
Einheit.

## Ablauf

1. Pro Einheit und Zeitraum wird eine eindeutige Referenz erzeugt.
2. Der camt.053-Parser liest eingehende Zahlungen und extrahiert die Referenz.
3. Der Abgleich vergleicht importierte Zahlungen mit den erwarteten Referenzen.
4. Nur eindeutige Treffer ändern den manuellen Zahlungsstatus.
5. Unklare oder abgelehnte Treffer bleiben im Importbericht und werden nicht
   still gesetzt.

## Statusvorschlag

- Referenz eindeutig, Betrag deckt den erwarteten Wert: `bezahlt`
- Referenz eindeutig, Betrag ist niedriger als erwartet: `teilbezahlt`
- Referenz eindeutig, kein erwarteter Betrag hinterlegt: `bezahlt`
- Referenz fehlt, ist unbekannt, doppelt oder Tenant-fremd: nicht setzen

## Importbericht

Der Bericht zählt:

- `eindeutig`: Treffer darf übernommen werden
- `unklar`: Treffer existiert, darf aber nicht automatisch gesetzt werden
- `abgelehnt`: keine gültige oder bekannte Referenz

Berichtszeilen enthalten Referenz, Einheit, Statusvorschlag, Betrag und Grund.
Sie enthalten keine IBAN, keinen Debitorennamen und keine unnötigen
personenbezogenen Bankdaten.

Nach der Bestätigung zeigt die Rückmeldung getrennt, wie viele eindeutige
Treffer vorlagen und wie viele Status tatsächlich geändert wurden. Derselbe
Datei-Digest kann pro Haus nur einmal übernommen werden.

## Produktscope

Der Abgleich ist Status-Transparenz, keine Buchhaltung. Er erzeugt keine
Sollstellung, keine Verbuchung, keine Zinsen, keine Mahngebühr und keinen
Zahlungsauftrag.
