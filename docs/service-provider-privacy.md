# Dienstleister-Zugriff und Datenschutz

hausv.org behandelt externe Dienstleister als eng begrenzte Rolle. Der Zugang ist
für konkrete Anliegen gedacht, nicht als allgemeiner Bewohner- oder Verwaltungszugang.

## Freigabestatus

Stand 18. Juli 2026 hat noch keine externe fachkundige Person die konkrete
Datenschutz- und Rechtsgestaltung geprüft oder freigegeben. Ein formell
bestellter Datenschutzbeauftragter ist dafür nicht vorausgesetzt; ob eine solche
Rolle überhaupt erforderlich ist, wurde ebenfalls noch nicht beurteilt.

Bis die Prüfung in HAUSV-86 dokumentiert abgeschlossen ist, darf der
Dienstleister-Zugang im Betrieb nicht aktiviert oder für echte Einladungen
verwendet werden. HAUSV-86 ist das fachliche Freigabe-Ticket und blockiert
HAUSV-128, bis die Prüfung und die Betreiberentscheidung dokumentiert sind.

Die Anwendung erzwingt diesen Zustand standardmäßig: Ohne die ausdrückliche
Server-Einstellung `SERVICE_PROVIDER_ACCESS_ENABLED=true` werden neue
Dienstleister-Zuordnungen, -Kontakte und -Zugänge vor dem Speichern abgelehnt.
Dabei entstehen weder Profil noch Einladung, E-Mail oder Sitzung. Bereits
vorhandene Dienstleister-Sitzungen und Anmeldeversuche werden ebenfalls
abgewiesen. Die Einstellung bleibt bis zur dokumentierten Freigabe auf `false`.

## Sichtbare Daten

Ein Dienstleister sieht nur offene Anliegen, bei denen seine E-Mail-Adresse als
zuständige Person eingetragen ist. In dieser Ansicht werden nur die Daten gezeigt,
die für die Bearbeitung nötig sind:

- Titel, Kategorie, Priorität und Status des Anliegens
- Ort im Haus, soweit vom Ersteller angegeben
- Beschreibung, Kommentare und Anhänge dieses Anliegens
- eigener Statusbeitrag und ein kurzer Terminvorschlag

Nicht sichtbar sind Hausübersicht, Bewohner- oder Benutzerlisten, Dokumente,
Abstimmungen, Parkplatzbereich, Einstellungen und andere Anliegen.

## Begrenzung und Entzug

Der Zugriff ist inhaltlich an das einzelne Anliegen gebunden. Wird die Zuordnung
entfernt oder das Anliegen geschlossen, verliert der Dienstleister den Zugriff.
Einladungen laufen über einen Magic-Link mit Deeplink zum betroffenen Anliegen.

## Dateiwege

Anhänge werden nicht als öffentliche Dateien ausgeliefert. Uploads liegen im
Dateisystem des Servers, die Auslieferung erfolgt ausschließlich über
authentifizierte App-Routen wie `/app/attachments/...`. Diese Route prüft Tenant,
Sitzung, Rolle und das konkrete Anliegen, bevor Original, Vorschau oder Thumbnail
ausgegeben werden. Bilder erhalten kleine Vorschauvarianten, bleiben aber unter
dem gleichen Zugriffsschutz.

## Betreiberprozess

Ob ein Dienstleister datenschutzrechtlich als Auftragsverarbeiter oder eigener
Verantwortlicher einzuordnen ist, hängt vom realen Einsatz ab. Für den Betrieb
gilt daher:

- Dienstleister nur dann einladen, wenn die Weitergabe für das konkrete Anliegen
  erforderlich ist.
- Keine unnötigen personenbezogenen Daten in Titel, Beschreibung, Fotos oder
  Kommentaren erfassen.
- Bestehende vertragliche Grundlagen der Hausverwaltung nutzen; falls nötig,
  Auftragsverarbeitervereinbarung oder vergleichbare Vereinbarung außerhalb des
  Portals klären.
- Zugriff nach Abschluss oder Fehlzuordnung sofort entziehen.

Das Produkt unterstützt diese Linie technisch über Least-Privilege-Rollen,
geschützte Dateiwege, Audit-Ereignisse für Einladung/Entzug und testsichere
Sperren für nicht zugewiesene oder geschlossene Anliegen.

## Fachliche Referenzen

- [Österreichische Datenschutzbehörde: Pflichten von Verantwortlichen](https://dsb.gv.at/rechte-pflichten/ihre-pflichten-als-verantwortlicher)
  und [Voraussetzungen für Datenschutzbeauftragte](https://dsb.gv.at/rechte-pflichten/datenschutzbeauftragter)
- [EU-Datenschutz-Grundverordnung](https://eur-lex.europa.eu/eli/reg/2016/679/oj),
  insbesondere Rollen, Transparenz, Datenminimierung und Speicherbegrenzung
- [Europäischer Datenschutzausschuss: Leitlinien zu Verantwortlichen und
  Auftragsverarbeitern](https://www.edpb.europa.eu/documents/guideline/guidelines-072020-on-the-concepts-of-controller-and-processor-in-the-gdpr_en)
