# Schnittstellen-QA

Schnittstellen duerfen erst als produktive Funktion gelten, wenn sie nicht nur
"irgendwie parsen", sondern reproduzierbar gegen Profile, Beispiele und
Feldgrenzen geprueft sind.

## Gemeinsame Gates

- Profil-/Namespace-Pruefung gegen das konkret unterstuetzte Format.
- Golden Files mit anonymisierten realen oder verifizierten Beispieldateien.
- Versionsmatrix je Adapter, z. B. camt 2009/2019 oder ebInterface 5.0/6.0.
- Harte Feldlimits: Zahlungsreferenz maximal 35 Zeichen, nur sichere
  ASCII-Zeichen; Verwendungszweck als Zusatzkontext, nicht als sichere ID.
- Datensatzfehler statt unnoetigem Komplettabbruch, sofern das Format mehrere
  Saetze enthaelt.
- Keine Ausgabe von IBAN, Debitor-Details oder personenbezogenen Rohdaten in
  Bewohneransichten oder zusammenfassenden Statusreports.
- Keine Buchung, keine Steuerlogik, keine Zahlungsauftraege.

## Aktueller Stand

| Adapter | Profilstand | Golden Files | Entscheidung |
| --- | --- | --- | --- |
| camt.053 | `camt.053.001.02`, `camt.053.001.08` | 2009 und 2019 | Primaerer Zahlungsstatus-Import |
| camt.054 | `camt.054.001.02`, `camt.054.001.08` | 2019 | Optionaler Detail-/Aviskanal |
| BMD/RZL | BMD `raw-v0` Kandidat, RZL spaeter | BMD Golden File vorhanden, externe NTCS-Pruefung offen | Keine fertigen Buchungssaetze |
| ebInterface | 6.0 priorisiert, 5.0 Fallback | 5.0 und 6.0 | Empfang und Ablage, nicht buchen |

## XSD-Gate

Offizielle XSD-Validierung bleibt das Ziel fuer produktive Formatfreigaben. Weil
XSD-Dateien lizenz- und quellenabhaengig gepflegt werden muessen, wird kein
Adapter ohne dokumentierte Quelle, Profilname und Golden File als "fertig"
bewertet. Bis zur vendorten oder CI-verfuegbaren XSD pruefen Adapter mindestens
Profil/Namespace, Pflichtfelder, Feldlimits und fachliche Richtung.

Fuer ebInterface ist labs.ebinterface.at der praktische externe Gegencheck: dort
koennen ebInterface 5.0, 6.0 und 6.1 gegen das XML Schema geprueft werden. Die
App unterstuetzt derzeit 5.0 und 6.0; 6.1 bleibt ein eigenes Folgeprofil.

## Steuerberater-Gate

BMD/RZL-Ausgaben brauchen zusaetzlich einen echten Testimport mit einem
oesterreichischen Steuerberater. Ohne dokumentierten Gegencheck wird kein Export
als produktiv markiert.

Fuer BMD gibt es ein internes Pruefpaket in
`docs/bmd-rawdata-verification.md` und ein Golden File
`testdata/bmd-raw-v0.csv`. Der Kandidat prueft nur Rohdatenfelder und lehnt
Konten-/Steuerkennzeichen ab, bis ein reales BMD-NTCS-Mapping bestaetigt ist.
