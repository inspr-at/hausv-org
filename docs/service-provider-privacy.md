# Dienstleister-Zugriff und Datenschutz

hausv.org behandelt externe Dienstleister als eng begrenzte Rolle. Der Zugang ist
für konkrete Anliegen gedacht, nicht als allgemeiner Bewohner- oder Verwaltungszugang.

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
