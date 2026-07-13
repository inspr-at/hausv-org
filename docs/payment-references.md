# Zahlungsreferenzen

Automatischer Zahlungsabgleich braucht eine kurze, eindeutige Referenz. hausv.org
verwendet dafür ein eigenes, bewusst schmales Format:

```text
HV-<TENANT>-<PERIOD>-<HASH>
```

Beispiel:

```text
HV-JHW22-202606-K7M4Q2P9RA
```

## Regeln

- maximal 35 Zeichen
- nur `A-Z`, `0-9` und Bindestrich
- immer Großbuchstaben
- keine Leerzeichen, Umlaute oder Sonderzeichen
- Tenant-Teil: lesbarer ASCII-Token, maximal 10 Zeichen
- Zeitraum-Teil: lesbarer ASCII-Token, maximal 8 Zeichen, z. B. `202606`
- Hash-Teil: 10 Zeichen aus Tenant, Scope, Zielobjekt, Zeitraum und Kollisionszähler

Die Referenz ist pro Tenant eindeutig. Wenn eine deterministische Referenz
bereits existiert, erzeugt der Generator mit einem Kollisionszähler eine neue
gültige Referenz.

## Produktgrenze

Die Referenz dient nur zur Zuordnung und Status-Transparenz. Sie ist keine
Buchungsnummer, keine Rechnungsnummer und löst keine Zahlung aus.
