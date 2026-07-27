# Schnittstellen-QA

Schnittstellen duerfen erst als produktive Funktion gelten, wenn sie nicht nur
"irgendwie parsen", sondern reproduzierbar gegen Profile, Beispiele und
Feldgrenzen geprueft sind.

## Gemeinsame Gates

- Profil-/Namespace-Pruefung gegen das konkret unterstuetzte Format.
- Golden Files mit synthetischen oder anonymisierten Beispieldateien und
  dokumentierter Herkunft.
- Versionsmatrix je Adapter, z. B. camt 2009/2019 oder ebInterface 5.0/6.0.
- Harte Feldlimits: Zahlungsreferenz maximal 35 Zeichen, nur sichere
  ASCII-Zeichen; Verwendungszweck als Zusatzkontext, nicht als sichere ID.
- Datensatzfehler statt unnoetigem Komplettabbruch, sofern das Format mehrere
  Saetze enthaelt.
- Keine Ausgabe von IBAN, Debitor-Details oder personenbezogenen Rohdaten in
  Bewohneransichten oder zusammenfassenden Statusreports.
- Keine Buchung, keine Steuerlogik, keine Zahlungsauftraege.

## Aktueller Stand

| Adapter | Profilstand | Golden Files | Verfügbarkeit | Entscheidung |
| --- | --- | --- | --- | --- |
| camt.053 | `camt.053.001.02`, `camt.053.001.08` | synthetische 2009- und 2019-Fixtures | geschützter Vorschau-/Übernahmefluss implementiert und getestet | Primärer Zahlungsstatus-Import; reale/anonymisierte Bankdatei ist zusätzliche Profilabdeckung, kein Human-Gate |
| camt.054 | `.08` mit synthetischer Repository-Fixture getestet; für `.02` nur Namespace akzeptiert | synthetisches 2019-Golden-File für `.08`; keines für `.02` | intern implementiert/getestet; keine produktive Upload-UI | `.02` bleibt bis Fixture und Golden File unverifiziert |
| Neutrale Übergabe | `manual-csv/raw-v0` | Golden File und Paket-/Manifest-Tests vorhanden | geschützte Auswahl, Vorschau und einmaliger ZIP-Download produktiv | Mandanten-/Personenbindung, Audit, keine BMD-/RZL-Kompatibilitätsbehauptung oder Buchungssätze |
| BMD NTCS | noch kein Zielvertrag | keines | geplant | Erst mit offizieller Importbeschreibung oder betreibereigener Vorlage als Zieladapter bauen |
| RZL FIBU | offizielle Import-Schnittstelle, Stand Juli 2026; Nutzung laut Handbuch nur für berechtigte RZL-Nutzer | noch keines | geplant | Erst mit autorisiertem Betreiberzugang oder von RZL freigegebener Spezifikation als Zieladapter bauen |
| ebInterface | 6.0 priorisiert, 5.0 Fallback | synthetische, offiziell schema-validierte 5.0- und 6.0-Fixtures | geschützter Vorschau-/Ablagefluss implementiert und getestet | Empfang und Ablage, nicht buchen; 6.1 bleibt ein separates Folgeprofil |

## XSD-Gate

Offizielle XSD-Validierung bleibt das Ziel fuer produktive Formatfreigaben. Kein
Adapter wird ohne dokumentierte Quelle, Profilname und Golden File als "fertig"
bewertet. Kann eine offizielle XSD aus Lizenz- oder Verfügbarkeitsgründen nicht
versioniert werden, dokumentiert der Betreiber URL, Abrufdatum, Prüfsumme und
Validator-Ergebnis. Der Adapter prüft weiterhin Namespace, Pflichtfelder,
Feldlimits und fachliche Richtung automatisiert.

Fuer ebInterface ist labs.ebinterface.at der offizielle Online-Gegencheck: dort
koennen ebInterface 5.0, 6.0 und 6.1 gegen das XML Schema geprueft werden. Die
App unterstuetzt derzeit 5.0 und 6.0; 6.1 bleibt ein eigenes Folgeprofil. Der
reproduzierbare Check `scripts/validate-ebinterface-fixtures.fish` sendet
ausschließlich die synthetischen Repository-Fixtures. Echte Rechnungen werden
weder im Produkt noch im Betreibercheck an den externen Validator übertragen.

Betreiberprotokoll vom 27.07.2026:

- Quelle: offizielle ebInterface-Dokumentation 5.0/6.0 und
  `https://labs.ebinterface.at/`
- Profile: ebInterface 5.0 und 6.0
- Ergebnis: beide synthetischen Fixtures laut offiziellem Validator gültig
- Produkttests: positive Profile, fehlende Pflichtwerte, 6.1-Ablehnung,
  Dateityp/-größe, CSRF, Rollen, Hausgrenze, Vorschau-Ablauf,
  Originaltreue, geschützte Auslieferung, Digest-Idempotenz und Audit

## Betreiber-Selbstprüfung statt externer Freigabe

Auf absehbare Zeit steht kein externer Auditor oder Steuerberater für Abnahmen
zur Verfügung. Das ist kein unsichtbarer Dauerblocker: HAUSV arbeitet mit
primären Hersteller-/Standardquellen, versionierten Fixtures, automatisierten
Positiv- und Negativtests sowie einem datierten Betreiberprotokoll. Diese
Selbstprüfung ist kein Zertifikat und keine Steuer- oder Rechtsberatung.

Der neutrale Export ist in `docs/structured-handoff-export.md` beschrieben. BMD
und RZL brauchen davon getrennte Zieladapter. Für BMD muss zuerst ein offizieller
Importvertrag oder eine betreibereigene NTCS-Vorlage vorliegen. Die offizielle
RZL-FIBU-Importschnittstelle ist zwar auffindbar, laut Handbuch aber nur für
berechtigte RZL-Nutzer bestimmt; vor Verwendung braucht HAUSV daher einen
autorisierten Betreiberzugang oder eine von RZL freigegebene Spezifikation. Ein
Testimport in einer vom Betreiber kontrollierten Zielumgebung erhöht den
Nachweis; fehlt diese Umgebung, bleibt genau dieser Punkt sichtbar und die
Zielsystem-Kompatibilität wird nicht als produktionsbereit behauptet.

## Primärquellen (verifiziert 27.07.2026)

- ISO 20022 Message Definitions und Message Archive für camt.053/.054:
  `https://www.iso20022.org/iso-20022-message-definitions` und
  `https://www.iso20022.org/catalogue-messages/iso-20022-messages-archive`
- ebInterface 5.0/6.0 Dokumentation und offizieller Validator:
  `https://www.ebinterface.at/download/documentation/ebInvoice_5p0.pdf`,
  `https://www.ebinterface.at/download/documentation/ebInvoice_6p0.pdf`,
  `https://labs.ebinterface.at/`
- RZL FIBU Import-Schnittstelle, Stand Juli 2026:
  `https://rzlsoftware.at/fileadmin/user_upload/PDF_Schnittstelle/RZL_FIBU_Import_Schnittstelle.pdf`
- BMD NTCS Standardschnittstellen und Import-Schulung:
  `https://www.bmd.com/at/akademie/akademieshop/seminar/d/fibu-standardschnittstellen-10943/14`
  und
  `https://www.bmd.com/at/akademie/akademieshop/seminar/d/die-10-wichtigsten-excelfunktionen-um-buchungen-in-ntcs-zu-importieren-11540`
