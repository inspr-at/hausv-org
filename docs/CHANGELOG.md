# Versionsverlauf

Alle sichtbaren Änderungen werden hier in kurzer, kundenorientierter Sprache gesammelt.

## [0.57.1] - 2026-07-29

### Verbessert

- **Der Energiemodus nennt den Testlauf jetzt eindeutig beim Namen.** HAUSV protokolliert darin nur mögliche Entscheidungen und schaltet ausdrücklich kein Gerät.
- **Aktive Steuerung bleibt klar getrennt.** Sie wird erst später mit einer eigenen Freigabe für konkrete Geräte möglich; der heutige Testlauf kann keine solche Freigabe vorwegnehmen.

## [0.57.0] - 2026-07-29

### Verbessert

- **Der persönliche Name des Zuhauses führt durch das Portal.** Seitenleiste, Mobilmenü, Energieüberblick und Einstellungen verwenden durchgehend den gewählten Anzeigenamen.
- **Die offizielle Wohneinheit bleibt direkt erkennbar.** Bezeichnungen wie „Top 11“ stehen ruhig und kleiner unter dem Anzeigenamen, statt mit ihm um Aufmerksamkeit zu konkurrieren.
- **Gebäudeübersicht und Zuhause-Einstellung sprechen dieselbe Sprache.** Die verknüpfte Einheit zeigt den persönlichen Namen, während Eingabefelder, Zahlungen, Dokumente und andere formale Bereiche weiterhin die offizielle Stammdatenbezeichnung verwenden.
- **Der Anzeigename begleitet bereits das laufende Onboarding.** Nach der Namenswahl bleibt die neue Identität auch in den folgenden Schritten sichtbar.
- **Die Darstellung ist für alle Rollen abgesichert.** Automatisierte Prüfungen kontrollieren Hierarchie, Reihenfolge, Umbenennen und mobile Darstellung sowie den Schutz vor Einblicken durch nicht zugeordnete Nutzer.

## [0.56.0] - 2026-07-29

### Neu

- **„Mein Zuhause“ hat einen eigenen, auffindbaren Einstellungsweg.** Anzeigename, Zuhause-Art und zugeordnete Wohnung lassen sich direkt vom Energieüberblick, vom Einstellungs-Hub und aus „Gebäude & Einheiten“ öffnen und ändern.
- **Anzeigename und offizielle Wohnung gehören nachvollziehbar zusammen.** Ein freundlicher Name wie „Penthouse“ wird sichtbar mit der offiziellen Einheit „Top 11“ verbunden, ohne deren Stammdaten umzubenennen.

### Verbessert

- **Offizielle Einheiten bleiben verlässlich unverändert.** Eine Umbenennung von „Mein Zuhause“ verändert weder Adresse noch Gebäudename oder Einheitsbezeichnung.
- **Zugriff folgt der zugeordneten Wohnung.** Nur deren Eigentümer sowie Hausverwaltung oder Administration können das Hausprofil ändern; Bewohner der Wohnung können den zugehörigen Überblick sehen.
- **Bestehende Energiehistorie bleibt sicher zugeordnet.** Eine alte, eindeutig erkennbare Wohnung wird beim Update einmalig verbunden; mehrdeutige Fälle bleiben für die Hausverwaltung sichtbar, ohne Daten still einer Wohnung freizugeben.
- **Technische Hilfe bleibt technisch begrenzt.** Delegierte Energiebetreuer können weiterhin Messwerte einrichten, aber weder die Identität des Zuhauses umbenennen noch dessen Betriebsmodus freigeben.
- **Die Herkunft des Cockpitnamens ist sichtbar.** Energieübersicht und Einheitenverwaltung zeigen Anzeigename und offizielle Wohnung gemeinsam und führen direkt zur passenden Einstellung.

## [0.55.0] - 2026-07-29

### Neu

- **Der Energiefluss hat eine gemeinsame Bildsprache.** Hausverbrauch, PV, Netzbezug und Einspeisung erhalten ruhige, eigenständige Zeichen, die auch ohne Farbe unterscheidbar bleiben.
- **Der Speicher zeigt Füllstand und Richtung.** Eine proportionale horizontale Batterie lässt freie Kapazität sichtbar; sanfte Pfeile zeigen Laden oder Entladen und bleiben bei reduzierter Bewegung statisch.
- **Der Tagesverlauf kann den ganzen heutigen Tag zeigen.** „Heute“ spannt die Achse fest von 00 bis 24 Uhr auf und lässt noch nicht gemessene Zukunft bewusst leer.
- **Das Diagramm nutzt auf Wunsch den ganzen Bildschirm.** Neben der vergrößerten Ansicht steht echtes Vollbild mit verständlichem Beenden, Escape-Verhalten und sicherer Rückkehr zum auslösenden Knopf bereit.

### Verbessert

- **Ungerade Messwertzahlen hinterlassen keine leere Kachel mehr.** Die letzte Energiezelle nutzt die verfügbare Breite und hält Desktop wie Mobil ausgewogen.
- **Der Zeitraumwechsel bleibt sichtbar.** Der Diagrammkopf landet auch unter den festen Sicherheitsleisten vollständig im Blick.
- **Lokale Tage bleiben zeitlich korrekt.** Tagesachsen berücksichtigen die Wiener Zeitzone einschließlich Zeitumstellungen.

## [0.54.1] - 2026-07-29

### Verbessert

- **Der Ortskopf nutzt die ganze Seitenleiste.** Die Karte steht ohne zusätzlichen Rahmen oder abgerundete Karte bündig am Rand und läuft mit einem weichen Schatten in die Navigation aus.
- **Der Pin zeigt präzise auf das Haus.** Seine Spitze sitzt auf der konfigurierten Adresse; das gewählte Hauszeichen bleibt klein und klar erkennbar.
- **Adresse, Trenner und Portalname bleiben ruhig.** Die Adresse erscheint ohne Link-Dekoration, der reduzierte Trenner erhält sichtbar Abstand und die Kartenquellenangabe steht dezent im Fußbereich.

## [0.54.0] - 2026-07-29

### Neu

- **Das eigene Haus ist sofort verortet.** Ein ruhiger, fester OpenStreetMap-Ausschnitt zeigt die konfigurierte Adresse mit dem gewählten Hauszeichen direkt im Pin.

### Verbessert

- **Adresse und Portalname bilden einen klaren Ortskopf.** Die Adresse führt dezent zur großen Karte; eine feine Messlatte trennt sie von „Hausportal · hausv.org“.
- **Die Karte bleibt bewusst still.** Sie lässt sich weder verschieben noch zoomen, lädt keine Karten-Skripte und ruft nur die tatsächlich sichtbaren, serverseitig zwischengespeicherten Kacheln ab.
- **Mobil bleibt die Navigation kompakt.** Karte, Pin und Adresse werden zu einer kleinen Ortsmarke, ohne Menü oder Seiteninhalt zu verdrängen.

## [0.53.0] - 2026-07-29

### Neu

- **Der Energieverlauf lässt sich groß öffnen.** Ein eigener Detailmodus schafft mehr Platz für Kurven, Skala und Planungsgrenze und schließt per Knopf oder Escape wieder an derselben Stelle.
- **Jeder Viertelstundenwert ist direkt ablesbar.** Maus, Touch und Pfeiltasten zeigen Uhrzeit, Hausverbrauch, PV-Erzeugung sowie verständlich benannten Netz- und Speicherfluss am gewählten Punkt.

### Verbessert

- **Nur der relevante Messpunkt wird hervorgehoben.** Eine ruhige Führungslinie und farblich passende Punkte geben Orientierung, ohne alle 97 Viertelstunden gleichzeitig auszuzeichnen.
- **Die Interaktion bleibt auch ohne Maus vollständig.** Tastatursteuerung, Fokus-Rückkehr, Live-Ansage und ausreichend große mobile Touch-Ziele sind im Rollenlauf abgesichert.

## [0.52.0] - 2026-07-29

### Verbessert

- **Verbrauch und Energiequellen lassen sich schneller unterscheiden.** Hausverbrauch erscheint als ruhige rote Linie mit leichter Fläche; PV, Netz und Speicher behalten eigene, zurückhaltende Farben.
- **Die Leistungsskala bleibt vergleichbar.** Positive und negative Werte erhalten gemeinsam 25 % Luft und werden nach außen auf volle 5 kW gerundet – einschließlich Einspeisung und Speicherladung.
- **10 kW bleiben eine sichtbare Planungsannahme.** Eine gestrichelte, ausdrücklich konfigurierbare Linie hilft bei der Orientierung, ohne sie als geltenden Tarif oder technische Grenze darzustellen.
- **Der Speicher ist wie eine Batterie lesbar.** „Energie gerade jetzt“ verbindet den aktuellen Füllstand mit Lade- oder Entladeleistung in einer kompakten Batterieanzeige.
- **Messwerte bekommen verständliche Rollen.** Ein dauerhaft erreichbares Setup erklärt Hausverbrauch, Netz, PV, Speicherleistung, Speicherfüllstand und die daraus berechneten Viertelstunden-Spitzen.

## [0.51.0] - 2026-07-29

### Neu

- **Die letzten 24 Stunden werden als ruhiger Energieverlauf sichtbar.** Hausverbrauch, PV, Netz und Speicher stehen auf einer gemeinsamen Kilowatt-Skala; eine kurze Zusammenfassung nennt die höchste Last und ordnet ihre Herkunft verständlich ein.

### Verbessert

- **Mobil bleibt der Verlauf wirklich lesbar.** Eine eigene schmale Diagrammgeometrie bewahrt Achsen, Linien und Zeitangaben ohne seitliches Scrollen.
- **Home Assistant wird geschont.** Viele tausend ausschließlich lesend geladene Zustandswechsel werden serverseitig auf 96 Viertelstunden verdichtet und pro Haus kurz zwischengespeichert; der Browser benötigt weder externe Chart-Bibliothek noch CDN.
- **Aktualität richtet sich nach den gerade gelesenen Werten.** Das Cockpit erkennt eine aktive Home-Assistant-Verbindung anhand der tatsächlichen Live-Zeitpunkte statt anhand der ursprünglichen Einrichtung.
- **Hausverbrauch bleibt Hausverbrauch.** Ältere eindeutig benannte Verbrauchssensoren werden aus einer früheren Batterie-Zuordnung sicher in die richtige Bedeutung überführt.

## [0.50.0] - 2026-07-29

### Verbessert

- **Viele Messwerte werden zu einem verständlichen Energiefluss.** Hausverbrauch, PV, Netz, Speicher und Batteriestand stehen gemeinsam in einer kompakten Live-Übersicht; kumulierte und zusätzliche Werte öffnen sich erst bei Bedarf.
- **Leistungswerte sind leichter zu lesen.** Große Wattwerte wechseln automatisch in eine passende Kilowatt-Darstellung, Lade- und Entladeleistung ergeben einen gemeinsamen Speicherzustand.
- **Home-Assistant-Technik bleibt im Hintergrund.** Normale Nutzer sehen verständliche deutsche Bedeutungen statt konkurrierender Sensorbezeichnungen; ein älterer Sonderfall bei batteriepräfixiertem Hausverbrauch wird korrekt eingeordnet.
- **Hausaufgaben bleiben im Seitenfluss.** Das geöffnete Freigabeformular verschiebt den Energie-Fahrplan zuverlässig nach unten, anstatt ihn zu überlagern.
- **Acht Live-Werte sind auf Desktop und Mobil abgesichert.** Playwright prüft Verdichtung, progressive Details, Touch-Ziele, Sicherheitsmodus und überlagerungsfreie Darstellung im vollständigen Rollenlauf.

## [0.49.1] - 2026-07-29

### Verbessert

- **Die Art des Zuhauses erklärt ihre Wirkung sofort.** Wohnung, Einfamilienhaus und Hausgemeinschaft beschreiben direkt nach der Auswahl, welchen Bereich der spätere Überblick umfasst; Rechte und der sichere Beobachtungsmodus bleiben davon unberührt.
- **Das Hauszeichen erhält mehr Präsenz.** Das bestehende Drei-Häuser-Zeichen ist in der Desktop-Seitenleiste deutlich größer und mit Adresse und Portalbezeichnung ruhiger ausbalanciert.
- **Desktop und Mobil bleiben gemeinsam abgesichert.** Playwright wechselt alle drei Zuhause-Arten, prüft die verständliche Erklärung und kontrolliert Zeichen, Touch-Ziele sowie seitlichen Überlauf auf beiden Größen.

## [0.49.0] - 2026-07-28

### Neu

- **Messwerte und Anlagen gehören sichtbar zusammen.** PV- und Speichermessungen werden bei eindeutiger Zuordnung mit der richtigen Anlage verbunden; manuelle Messpunkte lassen sich bewusst einem Verbraucher oder dem gesamten Haus zuordnen.
- **Die Messabdeckung erklärt Lücken auf einen Blick.** Das Cockpit zeigt kompakt, ob Hausanschluss, PV, Speicher, E-Auto, Warmwasser oder Wärmepumpe bereits gemessen oder bisher nur erfasst sind – ohne technische Sensor-IDs.

### Verbessert

- **Drei private Hauswege sind reproduzierbar geprüft.** Wohnung, Eltern-Haus ohne Speicher und Schwiegereltern-Haus mit Speicher laufen als getrennte, ausschließlich lesende Home-Assistant-Piloten durch Desktop- und Mobil-QA.
- **Technische Hilfe bleibt klar begrenzt.** Eine eigene Vertrauensperson kann das richtige Haus ansehen und einrichten, sieht aber niemals den Eigentümer-Schalter für die aktive Steuerung.
- **Die visuelle Hierarchie ist ruhiger.** ImageGen-Review und echte Browseraufnahmen führten zu einer flachen, immer sichtbaren Messübersicht statt eines weiteren versteckten Detailbereichs.
- **Der QA-Werkzeugkasten ist sicherer.** Playwright wurde auf eine Version ohne den gemeldeten Browser-Download-Mangel aktualisiert; Go-, Race-, Abhängigkeits-, Rollen- und Datenschutzprüfungen bleiben grün.

## [0.48.2] - 2026-07-28

### Verbessert

- **Der Hausmodus bleibt eine Eigentümerentscheidung.** Ausschließlich Eigentümer, Verwalter beziehungsweise Hausadministration und Admin dürfen „Nur beobachten“ bewusst umlegen; technische Vertrauenspersonen können weiterhin ansehen und einrichten.
- **Alte Freigaben erweitern den Zugriff nicht.** Ein früher gespeichertes Steuerungsrecht erlaubt einer technischen Hilfe keine Aktivierung und wird bei der nächsten Rechteänderung entfernt.
- **Die Rechteverwaltung spricht dieselbe klare Sprache.** Einladungen bieten nur noch Ansehen und Einrichten an; der permanente Schalter erklärt sichtbar, wer ihn für die Liegenschaft bedienen darf.

## [0.48.1] - 2026-07-28

### Verbessert

- **Home Assistant zeigt nur noch sichere Hausenergie-Vorschläge.** Netzbezug, Einspeisung, PV, Hausspeicher und Hausverbrauch werden anhand eindeutiger Sensortypen, Einheiten und Namen erkannt; Handy-, Schloss-, Roboter-, Fahrzeug- und Prognosewerte bleiben draußen.
- **Die Messwertauswahl ist deutlich ruhiger.** Höchstens fünf verständliche Empfehlungen sind vorausgewählt und jeweils als „Nur lesen“ gekennzeichnet; weitere plausible Treffer und technische Sensor-IDs öffnen sich erst bei Bedarf.
- **Bestehende Zuordnungen bleiben geschützt.** Bewusst bestätigte oder manuell benannte Messwerte werden weder durch eine neue Suche überschrieben noch durch „Ohne Verbindung starten“ entfernt.
- **Der reale Pilotweg ist reproduzierbar geprüft.** Eine Home-Assistant-Testinstanz enthält absichtlich passende und unpassende Geräte; Playwright kontrolliert Auswahl, Technikdetails, Offline-Weg, Tastatur, Rollen und mobile Darstellung.

## [0.48.0] - 2026-07-28

### Neu

- **Wiederkehrende Wartung bleibt direkt an der Anlage.** Fälligkeit, Intervall, Fachkontakt, Unterlage, Aufgabe und Erledigungsnachweis bilden einen nachvollziehbaren Wartungsplan; nach Abschluss wird der nächste Termin automatisch vorgemerkt.
- **Tarifbewertungen behalten ihren damaligen Wissensstand.** Monat, Messspitze, Datenqualität, Regelprofil und Regelversion lassen sich als unveränderlicher Verlauf festhalten, ohne einen Entwurf als gültigen Tarif auszugeben.
- **Bekannte Energie-Fachbetriebe lassen sich passend einordnen.** Region, Qualifikation und Fähigkeiten wie Leistungsmessung, Home Assistant, PV, Speicher, Wallbox oder Wärmepumpe bleiben am normalen Kontakt – ohne automatisch einen Portalzugang zu öffnen.
- **Energiemaßnahmen erhalten einen gemeinsamen Arbeitskontext.** Anfrage, bekannter Kontakt, Angebot, Termin, ausgeführte Arbeit, Nachweis und gemessener Vorher-/Nachher-Vergleich bleiben mit dem zugehörigen Anliegen verbunden.
- **Technische Vertrauenspersonen können direkt hausbezogen eingeladen werden.** Ansehen, Einrichten und Steuerungsfreigabe bleiben getrennte Rechte; vorhandene Konten behalten ihre anderen Hauszuordnungen unverändert.

### Verbessert

- **Das Energie-Cockpit ist auf den nächsten Schritt konzentriert.** Messdaten, Betreuung und Fachhilfe öffnen sich erst bei Bedarf; der Modus „Nur beobachten“ bleibt trotzdem immer prominent sichtbar.
- **Mehrere Zuhause bleiben technisch sauber getrennt.** Automatisch angelegte Anlagen verwenden hausbezogene Identitäten; Wartungen, Bewertungen und Maßnahmen sind in Speicher, Datenbank und Serverpfaden mandantensicher geprüft.
- **Der vollständige Hauptweg ist automatisiert abgesichert.** Playwright prüft Onboarding-Unterbrechung, Tastaturbedienung, Offline-Einstieg, Rollen, mobile Darstellung, Tarifverlauf, Wartung, Betreuungseinladung und kuratierte Fachhilfe.

## [0.47.0] - 2026-07-28

### Neu

- **„Mein Zuhause“ führt ohne Technikvorwissen zum ersten Energie-Fahrplan.** Wohnung oder Haus, große Verbraucher und vorhandene Messwerte werden in fünf ruhigen Schritten erfasst; PV, Speicher oder Home Assistant sind keine Voraussetzung.
- **Der Sicherheitsmodus bleibt immer sichtbar.** Jedes Haus startet mit „Nur beobachten“. Eine bewusste Freigabe beginnt zunächst in einem wirkungslosen Testlauf; der sofortige Rückweg steht dauerhaft bereit.
- **Home Assistant und Smart Meter liefern eine hausbezogene Messbasis.** Sensoren werden ausschließlich lesend vorgeschlagen und verständlich bestätigt; Smart-Meter-Dateien bleiben lokal zugeordnet und lassen sich ohne Duplikate erneut importieren.
- **Viertelstunden-Spitzen werden nachvollziehbar eingeordnet.** Datenqualität, Monatsmaximum, unverbindlicher Tarifentwurf und Was-wäre-wenn-Bandbreiten unterscheiden Messung, Annahme und fehlende Information klar.
- **Technische Hilfe lässt sich sicher delegieren.** Ansehen, Einrichten und eine spätere Steuerungsfreigabe sind getrennte, widerrufbare Rechte mit eigenem Benutzerkonto und Audit-Verlauf.
- **Empfehlungen werden zu normalen Hausaufgaben.** Nutzer wählen ausdrücklich, welche Daten in ein Anliegen übernommen werden; es entsteht weder eine automatische Beauftragung noch ein Zahlungs- oder Provisionsfluss.
- **Das private Preismodell ist transparent.** Der volle Produktumfang bleibt ab Abschluss der Einrichtung drei Jahre kostenlos; danach gilt als heutige Hypothese 12 € pro Jahr – noch ohne Paywall oder Abrechnung.

### Verbessert

- **Desktop und Mobil prüfen den Energie-Hauptweg automatisch.** Rollenverbote, Shadow-Testlauf, Sofort-Rückkehr, Touch-Ziele, Smart-Meter-Wiederholung, Datenschutzlogs und seitlicher Überlauf gehören jetzt zum Playwright-Regressionslauf.

## [0.46.0] - 2026-07-28

### Verbessert

- **Die öffentliche Startseite kommt schneller auf den Punkt.** Fünf Kernaufgaben, ein kompakter Produktstand und klare Grenzen ersetzen lange Funktions-, Rollen- und Zukunftslisten.
- **Pilot und Preisrichtung sind leichter einzuordnen.** Der kostenfreie Rahmen bis 25 Einheiten, der unverbindliche Zukunftsrichtwert von 1 € je Wohnungseinheit und drei konkrete Beispiele stehen gemeinsam an einer Stelle.
- **Details bleiben erreichbar, ohne den Einstieg zu überladen.** Produktgrenzen, rechtliche Angaben und Datenschutz öffnen sich erst bei Bedarf; Kontakt und Pilotanfrage bleiben direkt sichtbar.
- **Desktop und Mobil werden automatisch gegengeprüft.** Die Playwright-Rollenprüfung umfasst jetzt auch die öffentliche Startseite, ihre Aufklappwege, Textwahrheit, Seitenlänge und seitlichen Überlauf.

## [0.45.1] - 2026-07-28

### Verbessert

- **Betriebsprotokolle sind noch datensparsamer.** Request-Einträge verwenden stabile Routennamen statt konkreter Adressen; Zugriffstoken und unnötige Objektkennungen gelangen dadurch auch bei Kalender- und Übergabeaufrufen nicht in die Logs.
- **Die Datenschutzprüfung ist wiederholbar.** Der Rollen-Test kontrolliert seine JSON-Logs jetzt automatisch auf Struktur, Severity, notwendige Betriebsfelder, Klartext-E-Mail-Adressen, Token, Chat-IDs und Request-Inhalte.
- **Produktionsstichproben bleiben diskret.** Der Prüfweg gibt ausschließlich Zähler und Abweichungsarten aus, niemals Logwerte oder mögliche personenbezogene Daten.

## [0.45.0] - 2026-07-28

### Verbessert

- **Leere Seiten erklären sich selbst.** Dokumente, Aushang, Verlauf und die Anliegen-Verwaltung zeigen ohne Inhalte keine nutzlosen Such- oder Filterwerkzeuge mehr.
- **Der Bewohner-Verlauf spricht über echte Vorgänge.** Verständliche Titel wie „Kellerlicht defekt“ ersetzen technische Kennungen; „Rückfrage erhalten“ und „Anliegen bearbeitet“ erklären die Änderung ohne Systemjargon.
- **Technische Nachvollziehbarkeit bleibt erhalten.** Verwaltung und Audit behalten die vollständigen Ereignisdaten, während normale Bewohneransichten nur den notwendigen Kontext zeigen.
- **Dienstleister-Hinweise sind produktreif formuliert.** Interne Begriffe wie Betreiberfreigabe verschwinden aus Formularen und Fehlermeldungen; die derzeitige Verfügbarkeit bleibt eindeutig.
- **Einzahl und Mehrzahl sind konsistent.** Beiträge und Verlaufseinträge werden auch bei genau einem Treffer natürlich bezeichnet.
- **Die überarbeiteten Zustände sind auf Desktop und Mobil geprüft.** Leere Ablagen und ein gefüllter Bewohner-Verlauf bleiben bei 390 Pixeln ohne technischen Text oder seitlichen Überlauf.

## [0.44.0] - 2026-07-28

### Verbessert

- **Der Hausüberblick zeigt genau die nächste sinnvolle Aufgabe.** Rückfragen, neue Aushänge, offene Zahlungsstatus und Verwaltungsarbeit führen mit einer klaren Aktion direkt an die richtige Stelle.
- **Folgepunkte bleiben ruhig im Blick.** Bis zu drei weitere Hinweise stehen kompakt unter der Hauptaufgabe, ohne Inhalte aus Aushang, Anliegen oder Terminseiten als große Karten zu wiederholen.
- **Anliegen öffnen sofort den passenden Assistenten.** Die Verwaltung landet beim konkreten Priorisierungsschritt; Bewohner gelangen bei einer Rückfrage direkt zu ihrer Antwort.
- **Sonderfunktionen drängen sich nicht mehr vor.** Parkplatznutzung, Übergaben und Benutzerrechte stehen in einem zurückhaltenden Verwaltungsbereich statt im Mittelpunkt des Hausalltags.
- **Navigation und Kopfbereich sind leichter geworden.** Interne Kurzmarken und die auffällige Versionsplakette entfallen; Standardbereiche und Verwaltung sind klarer getrennt.
- **Admin- und Bewohnerwege sind auf Desktop und Mobil geprüft.** Der vollständige Weg vom neuen Anliegen über Priorisierung und Rückfrage bis zur Bewohnerantwort bleibt bei 390 Pixeln ohne seitlichen Überlauf.

## [0.43.0] - 2026-07-28

### Verbessert

- **Die Anmeldung spricht jetzt die Sprache der Hausgemeinschaft.** Ein neutraler „Anmelden“-Knopf ersetzt Anbieter- und Produktjargon; die E-Mail-Alternative bleibt direkt erreichbar.
- **Hausname und Adresse stehen im Mittelpunkt.** Das reduzierte Drei-Häuser-Zeichen, die Adresse und der Begriff „Hausportal“ ersetzen die bisherige technische Portalbezeichnung auch im Browser und in Anmelde-E-Mails.
- **Abgelaufene Links führen freundlich zurück.** Statt einer technischen Textseite erscheint wieder der vertraute Einstieg mit einer klaren Möglichkeit, einen neuen Link anzufordern; der alte Token bleibt nicht in der Adresse sichtbar.
- **Der Hausstandort ergänzt den Einstieg datensparsam.** Eine ruhige Kartenillustration lädt keine externen Karteninhalte; OpenStreetMap öffnet sich erst nach einem bewussten Klick.
- **Desktop und Mobil wurden mit beiden Anmeldewegen geprüft.** Der Einstieg bleibt bei 390 Pixeln ohne seitlichen Überlauf, der zentrale Login leitet korrekt weiter und der E-Mail-Testzugang führt ins Portal.

## [0.42.0] - 2026-07-28

### Verbessert

- **Ein neues Anliegen entsteht jetzt in drei kurzen Schritten.** Bewohner beschreiben zuerst das Ereignis, wählen danach den Ort und prüfen zuletzt eine verständliche Zusammenfassung.
- **Fotos gehören direkt zur Meldung.** Ein optionales Bild lässt sich beim Ereignis ergänzen, ohne Ortsangaben und Dateien in einem gemeinsamen Aufklappbereich suchen zu müssen.
- **Der Titel wird sinnvoll vorgeschlagen.** Aus der Beschreibung entsteht automatisch eine kurze Überschrift, die vor dem Senden frei angepasst werden kann.
- **Jede Seite zeigt genau die nächste Aktion.** Zurück, Weiter und Senden stehen nicht gleichzeitig im Wettbewerb; Pflichtangaben werden jeweils dort geprüft, wo sie benötigt werden.
- **Desktop und Mobil sind entlang des vollständigen Meldewegs geprüft.** Alle drei Schritte funktionieren bei 390 Pixeln ohne seitlichen Überlauf, mit mindestens 46 Pixel hohen Aktionen und ohne überlange Formularseite.

## [0.41.0] - 2026-07-28

### Verbessert

- **Bewohner sehen bei Anliegen nur noch die tatsächlich nötige nächste Aktion.** Ohne Rückfrage bleibt die Seite ruhig; bei einer Frage erscheint genau ein Antwortfeld, bei einer vorgeschlagenen Lösung eine klare Ja-/Nein-Entscheidung.
- **Verwaltungsnachrichten unterscheiden Information und Rückfrage.** Eine Information ist ohne Antwortdruck im Verlauf sichtbar, während eine Rückfrage automatisch eine eindeutige Bewohneraufgabe erzeugt.
- **Antworten schließen offene Rückfragen nachvollziehbar.** Nachrichtentypen bleiben im Anliegen gespeichert und im Aktivitätsverlauf erkennbar; vorhandene ältere Kommentare bleiben unverändert lesbar.
- **Lösungen werden gemeinsam abgeschlossen.** Die Verwaltung schlägt die erledigte Bearbeitung vor, Bewohner bestätigen sie oder öffnen das Anliegen mit einem Klick wieder.
- **Details bleiben erreichbar, ohne den Hauptweg zu überladen.** Meldung, Dateien, Kostenschätzung und bisheriger Verlauf liegen in einem ruhigen, optionalen Bereich unter der aktuellen Aufgabe.
- **Desktop und Mobil sind entlang beider Rollen geprüft.** Rückfrage, Antwort, Wartezustand und Lösungsbestätigung funktionieren bei 390 Pixeln ohne seitlichen Überlauf und mit großen Touch-Zielen.

## [0.40.0] - 2026-07-28

### Verbessert

- **Die Verwaltung bearbeitet Anliegen jetzt in zwei klaren Entscheidungen.** Dringlichkeit und Zuständigkeit erhalten eine eigene, ruhige Seite, statt gemeinsam mit Kommentaren, Dateien und Kosten in einer großen Kontrollansicht zu konkurrieren.
- **Alltagssprache ersetzt technische Felder.** „Heute kümmern“, „Diese Woche“ und „Kann warten“ machen die Priorisierung verständlich; anschließend lässt sich das Anliegen selbst übernehmen oder bewusst noch offen lassen.
- **Der Arbeitskontext bleibt erhalten.** Nach jedem Schritt bleibt dasselbe Anliegen geöffnet und bestätigt übersichtlich Priorität und Zuständigkeit.
- **Verlauf und Unterlagen bleiben vollständig erreichbar.** Kommentare, Anhänge und Kostenschätzungen sind weiterhin verfügbar, drängen sich aber erst auf Wunsch in den Vordergrund.
- **Rollen und Hausgrenzen bleiben geschützt.** Nur Verwaltung und Admins erreichen die Bearbeitung; Bewohner sehen weiterhin ausschließlich den für sie bestimmten Status.
- **Desktop und Mobil folgen demselben kurzen Weg.** Der zweistufige Ablauf ist bei 390 Pixeln ohne seitlichen Überlauf geprüft und behält klar erkennbare Hauptaktionen.

## [0.39.0] - 2026-07-27

### Neu

- **Ausgewählte Hausdaten lassen sich kontrolliert weitergeben.** Verwalter und Admins wählen einzelne Datenbereiche, prüfen den Umfang und laden erst danach ein strukturiertes ZIP-Paket herunter.
- **Jedes Paket ist nachvollziehbar.** Eine lesbare CSV-Datei und ein Manifest mit Version, Zeitpunkt, Datensatzanzahl und SHA-256-Prüfsumme machen die Übergabe reproduzierbar.
- **Haus- und Personengrenzen bleiben geschützt.** Vorschauen gelten nur 15 Minuten für den erstellenden Zugang, Downloads sind einmalig und erscheinen ohne Fachdaten im Aktivitätsverlauf.
- **Persönliche Zuordnungen bleiben draußen.** Der Einheitenstatus enthält keine Eigentümer-, Mieter- oder E-Mail-Listen; Parkplatz-Monatswerte sind ausschließlich für Admins auswählbar.
- **Die Produktgrenze ist sichtbar.** `raw-v0` ist eine neutrale Rohdatenübergabe ohne BMD-/RZL-Zusage, Buchungs-, Steuer-, Mahn- oder Zahlungslogik.
- **Desktop und Mobil führen durch denselben Dreischritt.** Auswahl, Prüfung und Download bleiben bei 390 Pixeln ohne seitlichen Überlauf und mit klaren nächsten Aktionen verständlich.

## [0.38.0] - 2026-07-27

### Neu

- **E-Rechnungen lassen sich direkt in der Dokumentablage prüfen.** Verwalter und Admins sehen vor der Ablage Rechnungsnummer, Betrag, Beteiligte und Termine in einer kurzen, verständlichen Vorschau.
- **Originale bleiben geschützt und unverändert.** Bestätigte ebInterface-Dateien werden als „Abrechnung“ ausschließlich für die Verwaltung gespeichert; Bewohner erhalten weder Listen- noch Downloadzugriff.
- **Doppelte Ablagen sind ausgeschlossen.** Ein hausbezogener Dateinachweis und der Aktivitätsverlauf machen jede Übernahme nachvollziehbar, ohne Rechnungsinhalte in technische Protokolle zu kopieren.
- **ebInterface 5.0 und 6.0 sind reproduzierbar geprüft.** Synthetische Golden Files bestehen den offiziellen Schema-Validator; Positiv-, Negativ-, Rollen-, Datenschutz- und Wiederholungstests sichern den Produktweg ab.
- **Desktop und Mobil führen durch denselben kurzen Ablauf.** Nach der Dateiwahl konzentriert sich die Seite auf die Rechnung und die nächste Entscheidung; bei 390 Pixeln bleibt alles ohne seitlichen Überlauf und mit mindestens 44 Pixel hohen Bedienelementen erreichbar.

## [0.37.1] - 2026-07-27

### Verbessert

- **Der Bankdatei-Import lässt sich auf kleinen Bildschirmen sicherer bedienen.** Zeitraum, Zurück-Aktion und Navigationswege bieten nun auch bei echten 390 Pixeln mindestens 44 Pixel hohe Touch-Ziele.

## [0.37.0] - 2026-07-27

### Neu

- **Zahlungen aus Bankdateien lassen sich sicher prüfen.** Verwalter und Admins wählen den Monat, sehen die gültigen Referenzen und erhalten vor jeder Änderung eine klare Vorschau.
- **Nur eindeutige Treffer werden übernommen.** Unklare oder abgelehnte Zeilen bleiben unverändert; die Rückmeldung trennt Treffer und tatsächlich geänderte Status.
- **Bankdaten bleiben datensparsam.** Die Datei wird nicht abgelegt, Vorschauen laufen nach 15 Minuten ab und zeigen weder IBAN noch Namen oder Verwendungszwecke.
- **Doppelte Übernahmen sind ausgeschlossen.** Ein hausbezogener Dateinachweis verhindert Wiederholungen und protokolliert Format, Prüfsumme und Ergebnis ohne private Bankdetails.
- **Desktop und Mobil folgen demselben kurzen Ablauf.** Zeitraum, Vorschau und Bestätigung bleiben ohne seitlichen Überlauf verständlich und gut bedienbar.

## [0.36.0] - 2026-07-27

### Verbessert

- **Die mobile Portalnavigation lässt sich leichter treffen.** Menü, Navigationswege, Versionsverlauf und Abmelden verwenden nun durchgängig mindestens 44 Pixel hohe Touch-Ziele.
- **Navigationsbezeichnungen verhalten sich einheitlich.** Auch längere Einträge bleiben in der zweispaltigen mobilen Navigation stabil und ohne seitlichen Überlauf.
- **Der gesamte Portalweg ist gemeinsam geprüft.** Eigentümer-, Verwalter- und Admin-Seiten wurden auf Desktop und 390 Pixeln mit Rollenabgrenzung, aktiver Navigation, Tastaturdialogen und geschlossener Dienstleister-Freigabe gegengeprüft.

## [0.35.0] - 2026-07-27

### Verbessert

- **Die Parkplatz-Verwaltung beginnt mit vier klaren Aufgaben.** Zugriff, Abrechnung, Laderegeln und Telegram sind direkt erreichbar, ohne dass alle Formulare gleichzeitig sichtbar werden.
- **Zugänge lassen sich als kurze Personenliste pflegen.** Name, Rolle, Status und genau die passende Aktion ersetzen wiederholte Feldüberschriften; Beträge und Erinnerungen bleiben bewusst in der Abrechnung.
- **Tarif und Zahlung folgen einer verständlichen Reihenfolge.** Gültigkeitsdatum und Preise stehen zuerst, Zahlungsstände führen zu den jeweiligen Monaten und Erinnerungen öffnen sich nur bei Bedarf.
- **Laderegeln erklären ihre Wirkung vor den Grenzwerten.** Automatik, Testbetrieb, Start und sicherer Stopp sind sofort verständlich; sieben technische Werte sowie der Regler-Verlauf bleiben vollständig einklappbar.
- **Telegram zeigt zuerst seine Einsatzbereitschaft.** Neue Codes und vorhandene Verknüpfungen sind getrennt; eine fehlende Server-Einrichtung wird klar erklärt, ohne gespeicherte Chats anzutasten.
- **Rollen und Hausgrenzen bleiben zuverlässig gewahrt.** Verwalter sehen keine unerreichbaren Technikwege; Parkplatzrechte werden gezielt für das aktuelle Haus gespeichert und lassen andere Berechtigungen unverändert.
- **Mobile Arbeitsseiten sind deutlich kürzer.** Zugriff benötigt rund 36 Prozent weniger Strecke; Abrechnung, Laderegeln und Telegram sind gegenüber der bisherigen Sammelseite rund 50 bis 67 Prozent kürzer.

## [0.34.0] - 2026-07-27

### Verbessert

- **Der Parkplatz zeigt zuerst den aktuellen Ladezustand.** Leistung, Hausakku und Einspeisung stehen gemeinsam in einer ruhigen Zeile; die passende nächste Aktion ist sofort erkennbar.
- **Automatik und manuelles Laden konkurrieren nicht mehr.** Ist die Automatik pausiert, wird ihre sichere Reaktivierung zur Hauptaktion; Normalladen bleibt bewusst unter „Manuell steuern“ erreichbar.
- **Monate öffnen eine eigene, klare Abrechnung.** Die neueste Abrechnung erscheint genau einmal mit Betrag, Verbrauch und Status; ältere Monate folgen als kompakte, vollständig antippbare Zeilen.
- **Das Monatsdetail beginnt mit Gesamtbetrag und Zahlung.** Kostenaufschlüsselung, Stundenwerte und Ladevorgänge öffnen sich erst bei Bedarf, ohne Informationen zu verlieren.
- **CSV und PDF bleiben direkt erreichbar.** Exporte sind an der Monatsabrechnung gebündelt und wurden für Bewohner sowie Verwaltung geprüft.
- **Die mobile Strecke ist 57 bis 67 Prozent kürzer.** Übersicht und Monatsdetail funktionieren bei 390 Pixeln ohne seitlichen Überlauf, mit großen Touch-Zielen und vollständiger Tastaturbedienung.

## [0.33.0] - 2026-07-27

### Verbessert

- **Der eigene Verlauf spricht Alltagssprache.** Bewohner sehen „Mein Verlauf“, berechtigte Verwaltungsrollen den „Aktivitätsverlauf“; der sichtbare Umfang und der Schutz interner Verwaltungsdetails werden klar erklärt.
- **Zeit, Vorgang und Kontext sind sofort erfassbar.** Datum steht nur einmal pro Tagesgruppe, während jede Zeile einen verständlichen Titel und genau eine ruhige Kontextzeile zeigt.
- **Filter bleiben kompakt und passend.** Eine schmale Übersichtsleiste ersetzt drei große Kennzahlen; angeboten werden nur Änderungsarten, die im aktuell freigegebenen Verlauf tatsächlich vorkommen.
- **Details öffnen sich direkt am Ereignis.** Die gesamte Zeile ist ein großes Tastatur- und Touch-Ziel; technische Angaben erscheinen erst auf Wunsch als klare Schlüssel-Wert-Liste.
- **Die Rollenabgrenzung bleibt unverändert sicher.** Bewohner und Dienstleister erhalten weiterhin ausschließlich aktuell freigegebene, datensparsam aufbereitete Einträge; die Verwaltung behält ihren mandantenbegrenzten Gesamtüberblick.
- **Der mobile Verlauf ist rund 52 Prozent kürzer.** Bewohner- und Admin-Beispielverläufe kommen ohne seitliches Scrollen aus und behalten alle Ereignisse sowie optionalen Details.

## [0.32.0] - 2026-07-27

### Verbessert

- **Einstellungen beginnen mit dem eigenen Konto.** Name, E-Mail, Rolle sowie die wichtigsten Wege zu Profil und Benachrichtigungen sind sofort erfassbar.
- **Das Profil trennt Angaben, Sichtbarkeit und Berechtigungen.** Persönliche Daten lassen sich ruhig bearbeiten; die freiwillige Freigabe für das Kontakte-Verzeichnis erklärt genau, was sichtbar wird.
- **Benachrichtigungen sind nach Alltagsthemen geordnet.** Der gesamte E-Mail-Versand und sechs einzelne Themen lassen sich über verständliche Schalter steuern.
- **Pausieren verliert keine Auswahl.** Gewählte Themen bleiben erhalten und werden beim erneuten Aktivieren wieder verwendet; der aktuelle Zustand ist auch in der Einstellungsübersicht sichtbar.
- **Rollen sehen nur ihre passenden Wege.** Bewohner finden den eigenen Verlauf unter einer verständlichen Bezeichnung, während Verwaltungsfunktionen ausschließlich Berechtigten angeboten werden.
- **Mobil bleibt alles lesbar und frei zugänglich.** Große Schalter und Hauptaktionen funktionieren ohne seitliches Scrollen oder überdeckte Inhalte.

## [0.31.0] - 2026-07-26

### Verbessert

- **Übergaben zeigen sofort den nächsten Schritt.** Offene Vorgänge, ablagebereite Protokolle und abgeschlossene Übergaben sind getrennt; vollständige Details bleiben bei Bedarf erreichbar.
- **Die Erfassung beginnt mit dem Wesentlichen.** Einheit, Anlass, Termin und Zustand stehen zuerst, während Personen, Zähler, Schlüssel, Notizen und Fotos gezielt ergänzt werden können.
- **Bestätigende Personen sehen den tatsächlichen Inhalt.** Räume, Zählerstände, Schlüssel, Notizen und vorhandene Dateien sind vor der ausdrücklichen, protokollierten Bestätigung übersichtlich prüfbar.
- **Bestätigte Protokolle bleiben verlässlich.** Ab der ersten Bestätigung können Fotos und Dateien nicht mehr verändert werden; die endgültige Ablage wird erst nach allen vorgesehenen Bestätigungen angeboten.
- **Mobile Übergaben sind deutlich kompakter.** Status, Fortschritt und Hauptaktion bleiben gut antippbar, ohne dass vollständige Protokolle die Übersicht überladen.

## [0.30.0] - 2026-07-26

### Verbessert

- **Gebäudedaten sind in klare Arbeitsbereiche gegliedert.** Stammdaten, Hauskontakte, Einheiten und Erscheinungsbild lassen sich direkt anspringen und bei Bedarf öffnen.
- **Einheiten bleiben als kompakte Liste erfassbar.** Typ, Anteil, verknüpfte Personen und Zahlungsstatus stehen in einer Zeile; die vollständige Bearbeitung öffnet sich erst auf Wunsch.
- **Zahlungsstatus gehört jetzt direkt zur jeweiligen Einheit.** Eine zweite, wiederholte Einheitenliste entfällt, während Speichern, Zeitangabe und Audit unverändert erhalten bleiben.
- **Neue Einheiten und das Portaldesign drängen sich nicht mehr in den Alltag.** Beide Funktionen sind weiterhin vollständig verfügbar, aber sinnvoll eingeklappt.
- **Speichern bleibt auf kleinen Bildschirmen im passenden Abschnitt erreichbar.** Die mobile Ansicht kommt ohne seitliches Scrollen aus und reduziert die Standard-Scrollstrecke bei vier Einheiten um rund 71 Prozent.

## [0.29.0] - 2026-07-26

### Verbessert

- **Dringende und offizielle Kontakte stehen zuerst.** Notdienst, Hausverwaltung und Hausmeister sind ohne Umwege erreichbar; der Beirat folgt als weitere Haus-Ansprechperson.
- **Anrufen und E-Mail sind echte Hauptaktionen.** Große, gut antippbare Schaltflächen ersetzen kleine Kontaktlinks und funktionieren auf Handy und Desktop.
- **Das Adressbuch bleibt im Alltag ruhig.** Neue Kontakte werden über eine klare Aktion ergänzt, während inaktive Einträge in einem eigenen Bereich liegen.
- **Bearbeiten und Deaktivieren sind sicher getrennt.** Kontaktdaten stehen im Bearbeitungsdialog zusammen; Deaktivieren erklärt seine Wirkung und lässt sich rückgängig machen.
- **Offizielle Stellen und Hausgemeinschaft sind klar getrennt.** Das Bewohnerverzeichnis erklärt seine freiwillige Freigabe und vermischt sich nicht mit Notdienst oder Verwaltung.
- **Leere Kontaktseiten geben einen gemeinsamen hilfreichen Hinweis.** Mehrere leere Bereiche werden nicht mehr wiederholt.

## [0.28.0] - 2026-07-26

### Verbessert

- **Benutzerzugänge lassen sich auf einen Blick erfassen.** Aktive, eingeladene und deaktivierte Personen sind zusammengefasst; Handlungsbedarf steht in der Liste zuerst.
- **Mobile Benutzerkarten zeigen nur das Wesentliche.** Name, Einheit, Rolle, Status und Bearbeiten bleiben sofort sichtbar, während wiederholte Detailinformationen zurücktreten.
- **Einladungen beginnen mit E-Mail und Rolle.** Name, Sonderrechte und Anmeldewege können bei Bedarf ergänzt werden, ohne den kurzen Standardweg zu überladen.
- **Zugänge werden ruhiger und sicherer bearbeitet.** Rolle, Status und Speichern bleiben im sichtbaren Bereich; Identitätsdaten, Anmeldewege und Sonderrechte sind logisch gruppiert.
- **Sperren und dauerhaftes Entfernen sind klar getrennt.** Die reversible Deaktivierung steht im normalen Ablauf, die dauerhafte Entfernung in einem eigenen bestätigten Gefahrenbereich.
- **Dienstleister-Hinweise erscheinen im passenden Kontext.** Der geschlossene Zugang wird bei der Rollenwahl erklärt, ohne die gesamte Seite als Warnung zu beginnen.

## [0.27.0] - 2026-07-26

### Verbessert

- **Offene Abstimmungen zeigen sofort den nächsten Schritt.** Eigentümer sehen zuerst Handlungsbedarf, Frist, Frage und ihr Stimmgewicht; Regeln und weitere Zeitangaben bleiben bei Bedarf erreichbar.
- **Stimmen lassen sich ruhiger abgeben und ändern.** Große Auswahlflächen und eine eindeutige Hauptaktion führen ohne Verwaltungsinformationen durch die Entscheidung.
- **Ergebnisse ersetzen die Eingabe statt sie zu wiederholen.** Teilnahme, Quorum, Ergebnis, Auszählung und Protokoll bilden nach Abschluss eine kompakte, verständliche Einheit.
- **Die Verwaltung arbeitet in einer gemeinsamen Übersicht.** Entwurf, Öffnen, laufende Abstimmung und Schließen stehen direkt am jeweiligen Vorgang, ohne eine doppelte Verwaltungsliste.
- **Neue Abstimmungen beginnen mit dem Wesentlichen.** Titel, Frage, Antwortmöglichkeiten und Frist stehen zuerst; Gewichtung, Quorum, Start, Erinnerung und Unterlagen sind übersichtlich eingeklappt.
- **Die Abstimmungswege nutzen kleine Bildschirme besser.** Irrelevante Bereiche und wiederholte Metadaten entfallen, während Rollen, Gewichtung, Audit und Protokollierung unverändert abgesichert bleiben.

## [0.26.0] - 2026-07-26

### Verbessert

- **Dokumente sind schneller auffindbar.** Die Hausablage zeigt nur Kategorien mit passenden Unterlagen; Suche und Sortierung bilden eine kompakte Einheit und führen bei leeren Ergebnissen verständlich zurück.
- **Ansehen ist der klare nächste Schritt.** Vorschau und Download sind eindeutig gewichtet, während Dateiname, Verwaltungsfunktionen und Versionsverlauf bei Bedarf erreichbar bleiben.
- **Veröffentlichen und Ersetzen sind übersichtlicher.** Der Upload fragt zuerst Titel, Kategorie, Sichtbarkeit und Datei ab; eine Einheit ist optional. Beim Ersetzen bleibt transparent, welche Einordnung erhalten und wo die bisherige Version abgelegt wird.
- **Neue Unterlagen führen vom Hausüberblick in den richtigen Kontext.** Bis zu zwei aktuelle Dokumente verlinken in die Hausablage statt sofort einen Download zu starten.
- **Die Dokumentablage nutzt kleine Bildschirme deutlich besser.** Leere Kategorien und der doppelte Upload-Einstieg entfallen; Bewohner sehen keine Verwaltungsaktionen.

## [0.25.0] - 2026-07-26

### Verbessert

- **Termine zeigen zuerst, was als Nächstes ansteht.** Datum, Uhrzeit, Ort und Kategorie sind auf einen Blick erfassbar; vergangene Termine und weitere Details bleiben erreichbar, ohne die Übersicht zu überladen.
- **Termine lassen sich schneller veröffentlichen.** Der Verwaltungsdialog fragt zuerst nur die wichtigsten Angaben ab und führt nach dem Speichern direkt zum neuen oder geänderten Termin.
- **Der eigene Kalender bleibt automatisch aktuell.** Das Kalender-Abo ist sowohl bei den Terminen als auch in den Einstellungen leicht auffindbar.
- **Die Anliegen-Triage wirkt ruhiger und eindeutiger.** Zusammengehörige Filter, stärkere Kartenhierarchie, klare Statusführung und eine gut erkennbare Hauptaktion erleichtern die tägliche Bearbeitung.
- **Handy und Desktop nutzen den Platz besser.** Lange Termintitel bleiben lesbar, Verwaltungsaktionen drängen sich Bewohnern nicht auf und alle geprüften Ansichten funktionieren ohne seitliches Scrollen.

## [0.24.0] - 2026-07-26

### Verbessert

- **Der Hausüberblick zeigt zuerst, was jetzt wichtig ist.** Bewohner sehen neue Informationen und eigene offene Anliegen in sinnvoller Reihenfolge; die Verwaltung gelangt direkt zur offenen Arbeit. Ohne Handlungsbedarf bleibt eine ruhige Bestätigung statt leerer oder doppelter Übersichten.
- **Anliegen sind schneller gemeldet und leichter zu verfolgen.** Die Meldung fragt zuerst nur das Wesentliche ab. Danach stehen Status und nächster Schritt direkt an der Karte; Verlauf, Anhänge und weitere Angaben bleiben bei Bedarf erreichbar.
- **Die Verwaltung bearbeitet Anliegen in einem kompakten Arbeitsablauf.** Filter und Zusatzfelder sind eingeklappt, Priorität und Zuständigkeit sofort erkennbar, und nach einer Aktualisierung bleibt die passende Karte im Triage-Board im Blick.
- **Aushänge haben einen eindeutigen Veröffentlichungs- und Lesefluss.** Es gibt nur noch einen Erstellen-Einstieg. Neue Beiträge öffnen ihren Text für Bewohner automatisch, während Planung, Fixierung und Anhänge den Hauptdialog nicht überladen.
- **Mobile Orientierung ist klarer.** Die Kopfzeile nennt die aktuelle Seite, Navigation und Inhalte bleiben ohne seitliches Scrollen bedienbar, und die häufigsten Abläufe passen kompakter auf kleine Bildschirme.

## [0.23.1] - 2026-07-26

### Verbessert

- **Verantwortliche Stelle und Hausanschrift sind eindeutig getrennt.** Die Datenschutzseite zeigt die Anschrift des verantwortlichen Hausbetriebs als eigene Angabe und kennzeichnet die Portaladresse separat als betroffenes Haus.
- **Die tatsächliche Infrastruktur ist vollständig erklärt.** Netcup-Hosting, Cloudflare-Webschutz, selbst betriebenes Zitadel, verschlüsselte Hetzner-Sicherungen und der Resend-Drittlandtransfer sind transparent dokumentiert.
- **Die Dienstleister-Freigabe ist konkret vorbereitet.** Ein hausbezogenes Art.-28-/TOM-Paket beschreibt Verarbeitung, Schutzmaßnahmen, Unterauftragsverarbeiter, Löschung und den weiterhin geschlossenen Freigabeweg.

## [0.23.0] - 2026-07-26

### Verbessert

- **Betreiber und Kontakt sind vollständig nachvollziehbar.** Impressum und Datenschutzinformation nennen den persönlichen Betreiber, seine ladungsfähige Anschrift und den dauerhaften technischen Kontakt; die verantwortliche Stelle des jeweiligen Hauses bleibt davon klar getrennt.
- **Der private Pilot ist eindeutig vom öffentlichen Angebot abgegrenzt.** Zugang, Kostenorientierung und Vertragsstatus erklären verständlich, dass derzeit kein öffentlicher Online-Vertragsabschluss und keine automatische Vertragsannahme stattfinden.
- **Die rechtliche Selbstprüfung ist transparent belegt.** Datum und österreichische Primärquellen sind direkt verlinkt; die Seiten stellen klar, dass keine externe Zertifizierung behauptet wird.

## [0.22.0] - 2026-07-26

### Neu

- **Persönlich freigegebene Vorgänge sind nachvollziehbar.** Bewohner und Dienstleister sehen im Audit-Log die eigenen Aktionen sowie aktuell freigegebene Anliegen und Einheiten. Personenbezogene Verwaltungsdetails bleiben dabei ausgeblendet.

### Verbessert

- **Anhänge und Zahlungsimporte hinterlassen eine verständliche Spur.** Aufrufe und Löschungen von Anhängen sowie verarbeitete Integrationsimporte werden protokolliert, ohne Dateinamen, Zahlungsreferenzen, Beträge oder Bankdaten in das Audit-Log zu übernehmen.
- **Audit-Einträge sind leichter einzuordnen.** Neue Aktionen und Ziele tragen verständliche Bezeichnungen und unterscheiden Hinzufügen, Ändern und Entfernen weiterhin sichtbar.

## [0.21.0] - 2026-07-26

### Verbessert

- **Übergaben lassen sich am Handy verlässlich abschließen.** Ausgewählte Fotos werden vor dem Speichern sichtbar, können vor und nach dem Speichern entfernt werden und der Speichern-Knopf verdeckt den Foto-Bereich auf kleinen Bildschirmen nicht mehr.
- **Der Zweck des Übergabeprotokolls ist klar abgegrenzt.** Erfassung, Bestätigung und PDF weisen verständlich darauf hin, dass der Zustand dokumentiert wird, ohne Kautionen, Schäden oder Buchhaltung abzurechnen.

## [0.20.0] - 2026-07-26

### Verbessert

- **Der Produkt-Ausblick zeigt einen ehrlichen Status.** Sieben zentrale Vorhaben sind jetzt klar als verfügbar oder in Arbeit gekennzeichnet und erklären jeweils den konkreten Nutzen. Zusätzlich macht die Startseite sichtbar, welche Aufgaben das Portal in der ersten Produktstufe bewusst nicht übernimmt.

## [0.19.0] - 2026-07-26

### Verbessert

- **Kalenderänderungen sind im Audit-Log nachvollziehbar.** Angelegte, geänderte und gelöschte Termine erscheinen mit Akteur, Zeitpunkt und Änderungstyp in der Verwaltungshistorie. Freitext, Ort und Dateinamen bleiben dabei bewusst außerhalb des Audit-Protokolls.

## [0.18.1] - 2026-07-26

### Verbessert

- **Betriebsmeldungen sind schneller auswertbar.** Hintergrundvorgänge protokollieren Bereich, Haus und betroffene Objekte jetzt als klar getrennte Felder. Störungen lassen sich dadurch gezielter finden, ohne personenbezogene Angaben im Klartext zu protokollieren.
- **Automatische Prüfungen laufen wieder unabhängig vom GitHub-Minutenkontingent.** Tests, Sicherheitsprüfung und Container-Bau nutzen nun die bereits etablierte Blacksmith-Infrastruktur.

## [0.18.0] - 2026-07-26

### Neu

- **Datenschutzinformationen direkt erreichbar.** Vor der Anmeldung erklärt eine eigene Seite verständlich, welche Daten das Portal verarbeitet, wofür sie benötigt werden, welche externen Dienste beteiligt sind und an wen sich Betroffene wenden können.

### Verbessert

- **Dienstleister-Freigabe nachvollziehbar abgesichert.** Der eingeschränkte Dienstleister-Zugang bleibt standardmäßig geschlossen und lässt sich erst mit einer ausdrücklich versionierten Betreiber-Selbstprüfung aktivieren. Die Freigabe hängt damit von dokumentierten Nachweisen statt von einer derzeit nicht verfügbaren externen Prüfung ab.
- **Keine extern geladenen Web-Schriften mehr.** Das Portal verwendet lokale Systemschriften. Beim Aufruf einer Seite entsteht dadurch keine Verbindung mehr zu Google Fonts.
- **Gesundheitsprüfung prüft echte Abhängigkeiten.** Der Container gilt nur dann als gesund, wenn die eingebettete Datenbank antwortet und das Datenverzeichnis beschreibbar ist.
- **Anfragen besser nachvollziehbar.** Strukturierte Request-Logs enthalten jetzt zusätzlich das betroffene Haus.
- **Audit-Aufbewahrung klar begrenzt.** Das laufende Audit-Protokoll rotiert nach Größe oder Alter; Archive oberhalb der dokumentierten Aufbewahrungsfrist werden automatisch entfernt.

## [0.17.7] - 2026-07-24

### Verbessert

- **Datenbestand bleibt dauerhaft überschaubar.** Von gelöschten Anhängen blieb bisher dauerhaft ein leerer Verweis zurück. Diese Reste werden ein Jahr nach der Löschung endgültig entfernt — die Löschung selbst bleibt im Protokoll nachvollziehbar. Bestehende Anhänge und der Versionsverlauf von Dokumenten bleiben unverändert erhalten.

## [0.17.6] - 2026-07-24

### Verbessert

- **Alte Foto-Sonderbehandlung bei Anliegen entfernt.** Nachdem die älteren Fotos in die reguläre Anhang-Verwaltung übernommen sind, entfällt der frühere Sonderweg für ihre Anzeige. Fotos an Anliegen laufen jetzt einheitlich über die Anhang-Verwaltung und bleiben unverändert sichtbar.

## [0.17.5] - 2026-07-24

### Verbessert

- **Ältere Anliegen-Fotos in die reguläre Anhang-Verwaltung übernommen.** Fotos, die vor der Umstellung auf die allgemeine Anhang-Verwaltung hochgeladen wurden, lagen noch in einer eigenen Ablage. Sie wurden verlustfrei übernommen und bleiben an ihrem Anliegen unverändert sichtbar.

## [0.17.4] - 2026-07-24

### Verbessert

- **Aufräumarbeiten im Hintergrund.** Der nicht mehr genutzte Code-Pfad für den früheren Foto-Upload an Anliegen wurde entfernt; Anhänge laufen längst über die reguläre Anhang-Verwaltung. Bereits vorhandene Fotos bleiben unverändert abrufbar.

## [0.17.3] - 2026-07-24

### Verbessert

- **Einladungen werden vollständig oder gar nicht gespeichert.** Beim Einladen einer Person entstehen der Personeneintrag und die Zugehörigkeit zum Haus jetzt gemeinsam in einem einzigen Vorgang. Bricht etwas dazwischen ab, bleibt kein halb angelegter Zugang ohne Haus zurück. Dasselbe gilt beim Bearbeiten, inklusive einer Änderung der E-Mail-Adresse.

## [0.17.2] - 2026-07-24

### Verbessert

- **Umstellung der Datenspeicherung abgeschlossen.** Die Anwendung arbeitet jetzt ausschließlich mit der eingebetteten Datenbank. Der bisherige Ausweichweg auf die alten JSON-Dateien entfällt: Ein Datenbankproblem führt jetzt zu einem klaren Startfehler statt zu einem stillen Weiterlaufen mit veralteten Daten. Die bisherigen Dateien bleiben unverändert als Sicherung liegen. Für Nutzer ändert sich nichts.

## [0.17.1] - 2026-07-24

### Verbessert

- **Sichtbarkeit im Kontakte-Verzeichnis gilt je Haus.** Das Kontakte-Verzeichnis wird pro Haus angezeigt, deshalb wird die Einstellung „Im Kontakte-Verzeichnis anzeigen" jetzt auch pro Haus gespeichert. Wer zu mehreren Häusern gehört, kann in einem Haus sichtbar und in einem anderen verborgen sein. Solange die Einstellung nicht bewusst im Haus gesetzt wird, gilt unverändert die bisherige persönliche Wahl — für bestehende Nutzer ändert sich nichts.
- **Zugangsbearbeitung zeigt nur noch Änderbares.** Name, Titel und E-Mail gehören zur Person und gelten für alle Häuser; in der Hausverwaltung werden sie jetzt nur noch lesend angezeigt, mit einem Hinweis auf die zentrale Pflege. Bisher ließen sich die Felder ausfüllen, ohne dass die Änderung gespeichert wurde.

## [0.17.0] - 2026-07-24

### Verbessert

- **Personen und Häuser sind jetzt sauber getrennt.** Eine Person kann zu mehreren Häusern gehören, und eine Hausverwaltung verwaltet ausschließlich die Zugehörigkeit zum eigenen Haus: Rolle und Rechte gelten je Haus, eine Änderung wirkt nicht mehr in anderen Häusern. „Entfernen" löst nur die Zugehörigkeit zum eigenen Haus und löscht nicht die Person. E-Mail, Titel und Name gehören zur globalen Identität und werden nur von der Plattform-Administration geändert. Bestehende Zugänge wurden verlustfrei übernommen; bei einem einzelnen Haus ändert sich nichts.

## [0.16.1] - 2026-07-24

### Verbessert

- **Vorbereitung für mehrere Häuser je Person.** Im Hintergrund entsteht ein Datenmodell, in dem Person und Haus getrennte Objekte sind: Eine Person kann zu mehreren Häusern gehören, und eine Hausverwaltung bearbeitet ausschließlich die Zugehörigkeit ihres eigenen Hauses. Für Nutzer ändert sich nichts — Anmeldung, Rollen und Rechte laufen unverändert über die bisherigen Daten.

## [0.16.0] - 2026-07-24

### Verbessert

- **Umstellung auf die neue Datenspeicherung abgeschlossen.** Anliegen (inkl. Kommentare und Statusverlauf), Abstimmungen (inkl. abgegebener Stimmen), Wohneinheiten und der Telegram-Bot laufen jetzt ebenfalls über die eingebettete Datenbank. Bestehende Daten wurden verlustfrei übernommen, mit Rückfallebene. Damit ist die schrittweise Umstellung abgeschlossen; zusammengehörige Änderungen werden gemeinsam gespeichert. Für Nutzer ändert sich nichts.

## [0.15.12] - 2026-07-24

### Verbessert

- **Anhänge auf die neue Datenspeicherung umgestellt.** Anhänge (Fotos und Dateien an Anliegen, Aushängen und Übergaben) laufen jetzt über die eingebettete Datenbank; die Dateien und ihre Vorschaubilder liegen unverändert im Dateispeicher. Ein fehlgeschlagener Mehrfach-Upload hinterlässt weiterhin garantiert keine halben Datensätze oder verwaisten Dateien. Für Nutzer ändert sich nichts.

## [0.15.11] - 2026-07-24

### Verbessert

- **Übergabeprotokolle werden zuverlässig abgelegt.** Beim Ablegen eines Übergabeprotokolls im Dokumentenbereich werden das Dokument und die Verknüpfung am Übergabe-Eintrag jetzt gemeinsam in einem Vorgang gespeichert. Ein wiederholtes Ablegen (Doppelklick, Neuladen) erzeugt damit kein zweites Protokoll mehr, und ein abgebrochener Vorgang hinterlässt keine verwaiste Datei.

## [0.15.10] - 2026-07-24

### Verbessert

- **Dokumente auf die neue Datenspeicherung umgestellt.** Die Dokumentenverwaltung (inkl. Versionsverlauf beim Ersetzen) läuft jetzt über die eingebettete Datenbank; die Dateien selbst liegen unverändert im Dateispeicher. Bestehende Dokumente wurden verlustfrei übernommen, mit Rückfallebene. Für Nutzer ändert sich nichts.

## [0.15.9] - 2026-07-24

### Verbessert

- **Wohnungsübergaben auf die neue Datenspeicherung umgestellt.** Übergabeprotokolle (Ein-/Auszug, Bestätigungen, abgelegte PDFs) laufen jetzt über die eingebettete Datenbank; bestehende Protokolle wurden verlustfrei übernommen, mit Rückfallebene. Für Nutzer ändert sich nichts.

## [0.15.8] - 2026-07-23

### Verbessert

- **Termine auf die neue Datenspeicherung umgestellt.** Termine und Kalender laufen jetzt über die eingebettete Datenbank; bestehende Einträge wurden verlustfrei übernommen, mit Rückfallebene. Für Nutzer ändert sich nichts.

## [0.15.7] - 2026-07-23

### Verbessert

- **Aushänge auf die neue Datenspeicherung umgestellt.** Aushänge und Hausjournal laufen jetzt über die eingebettete Datenbank; bestehende Einträge wurden verlustfrei übernommen, mit Rückfallebene. Für Nutzer ändert sich nichts.

## [0.15.6] - 2026-07-23

### Verbessert

- **Weitere Bereiche auf die neue Datenspeicherung umgestellt.** Adressbuch/Kontakte und der Gelesen-Status von Aushängen laufen jetzt über die eingebettete Datenbank; bestehende Daten wurden verlustfrei übernommen, mit Rückfallebene. Für Nutzer ändert sich nichts.

## [0.15.5] - 2026-07-23

### Verbessert

- **Zahlungsstatus auf die neue Datenspeicherung umgestellt.** Der manuelle Zahlungsstatus je Einheit läuft jetzt über die eingebettete Datenbank; bestehende Angaben wurden verlustfrei übernommen, mit Rückfallebene. Für Nutzer ändert sich nichts.

## [0.15.4] - 2026-07-23

### Verbessert

- **Benachrichtigungs-Einstellungen auf die neue Datenspeicherung umgestellt.** Die E-Mail-Benachrichtigungs-Präferenzen laufen jetzt über die eingebettete Datenbank; bestehende Einstellungen wurden verlustfrei übernommen, mit Rückfallebene. Für Nutzer ändert sich nichts.

## [0.15.3] - 2026-07-23

### Verbessert

- **Erste Bereiche auf die neue Datenspeicherung umgestellt.** Anmelde-Verlauf und Selbstverwaltungs-Profilangaben laufen jetzt über die eingebettete Datenbank statt über JSON-Dateien; bestehende Angaben wurden verlustfrei übernommen, mit automatischer Rückfallebene auf die bisherige Speicherung. Für Nutzer ändert sich nichts.

## [0.15.2] - 2026-07-23

### Verbessert

- **Grundlage für robustere Datenspeicherung.** Im Hintergrund wurde die Basis für eine transaktionssichere Datenhaltung (eingebettete Datenbank) gelegt. Für Nutzer ändert sich nichts; bestehende Daten und Abläufe bleiben unverändert.

## [0.15.1] - 2026-07-23

### Verbessert

- **Zugriffsprüfungen zentral abgesichert.** Die Rechteprüfung für geschützte Verwaltungsbereiche (Benutzerverwaltung, Parkplatz- und Ladeeinstellungen, Dokument-Uploads) läuft jetzt an einer zentralen Stelle statt in jeder Seite einzeln. Das verhindert, dass eine neue Seite die Prüfung versehentlich auslässt; eine Rechte-Matrix im Test sichert es ab.

## [0.15.0] - 2026-07-23

### Neu

- **Parkplatz-Monat exportieren und drucken.** Jeder Monat der Parkplatzabrechnung lässt sich jetzt als CSV mit Stundendetails herunterladen und über eine aufgeräumte Druckansicht als PDF speichern oder privat teilen. Die exportierten Summen entsprechen exakt der Bildschirmansicht; unvollständige Monate werden vorher deutlich markiert.

## [0.14.0] - 2026-07-23

### Neu

- **Zugänge deaktivieren statt löschen.** Ein Zugang kann jetzt vorübergehend gesperrt werden, ohne ihn zu entfernen: Die Anmeldung ist blockiert, der Eintrag samt Rechten und Verlauf bleibt erhalten und lässt sich jederzeit wieder aktivieren. Der Notfall-Admin bleibt geschützt, und niemand kann sich selbst aussperren.

## [0.13.0] - 2026-07-23

### Neu

- **Benutzer & Rechte vollständig in der App verwalten.** Personen, die bisher fest in der Konfiguration hinterlegt waren, lassen sich jetzt direkt im Portal bearbeiten — Rolle, Sonderrechte und Anmeldeweg ändern sich sofort, ganz ohne neue Auslieferung.
- **Anmeldeweg pro Person wählbar.** Für jeden Zugang kann festgelegt werden, ob die Anmeldung per E-Mail-Link, über Zitadel-SSO oder auf beiden Wegen möglich ist.

### Verbessert

- **Schutz vor Aussperren.** Die letzte Administrator-Rolle kann nicht entzogen und man kann sich nicht selbst die Administratorrechte entziehen. Der in der Konfiguration hinterlegte Notfall-Admin bleibt als Rückfallebene jederzeit erhalten.

## [0.12.2] - 2026-07-19

### Verbessert

- **Zahlungen klarer erfasst.** Der Zahlungsbereich im Monatsdetail ist jetzt sauber abgesetzt und kann optional Datum, Zahlungsart und Referenz festhalten — auch nachträglich für bereits markierte Monate. Bewohner sehen weiterhin nur den Bezahlt-Status.

## [0.12.1] - 2026-07-19

### Verbessert

- **Aufgeräumte Einstellungsseite.** Die Parkplatz-Einstellungen sind jetzt kompakt gruppiert: Eingabefelder mit Einheiten direkt im Feld, klare Schalter für Regler und Testbetrieb, übersichtlicher Regler-Status und Telegram-Bereich.

## [0.12.0] - 2026-07-19

### Neu

- **PV-Überschussladen für Parkplatz 20.** Die Plattform steuert das Laden jetzt selbst nach Sonnenstrom: Ist der Hausakku voll und wird eingespeist, startet die Ladung automatisch — Überschuss-Strom wird zum Fixpreis von 0,10 €/kWh abgerechnet, normales Laden weiterhin nach aWATTar-Preis plus Netzgebühr. Der Start erfolgt im Beobachtungsbetrieb.
- **Live-Ansicht auf der Parkplatzseite.** Eine neue „Jetzt“-Karte zeigt Ladequelle, Hausakku, Einspeisung und die Aufteilung Überschuss/Normal für heute und den laufenden Monat — inklusive Ein/Aus-Schalter.
- **Telegram-Bot.** Statusabfrage und Ein/Aus-Befehle laufen über einen eigenen Bot; Benachrichtigungen kommen nur noch bei echten Änderungen (Start/Ende einer Ladung), mit eingebauter Schutzbremse gegen Nachrichtenfluten.
- **Abrechnung mit Überschuss-Ausweis.** Monatsübersicht, Stundenwerte und CSV-Export weisen Überschuss- und Normalanteile getrennt aus; bestehende Monate bleiben unverändert.

## [0.11.0] - 2026-07-18

### Verbessert

- **Mehr Spielraum für kleine Hausgemeinschaften.** Der kostenfreie Fair-Use-Rahmen umfasst jetzt bis zu 25 Wohneinheiten.
- **Wohneinheiten fairer gezählt.** Wohnungen und vergleichbare Nutzungseinheiten zählen vollständig; Zubehör wie Keller und Stellplätze wird nicht automatisch wie eine Wohnung berechnet.
- **Dokumente schneller überblickt.** Die Dokumentenübersicht nutzt auf der Startseite die volle Breite; längere Texte brechen in Karten ruhiger um.
- **Freigabestatus klar benannt.** Die Dienstleister-Koordination wird bis zur externen Datenschutzprüfung als technisch vorbereitet statt als freigegebener Pilot ausgewiesen.
- **Dienstleister standardmäßig gesperrt.** Zuordnungen, Einladungen und Anmeldungen externer Dienstleister bleiben bis zur dokumentierten Datenschutzfreigabe technisch geschlossen.
- **Aktuelle Sicherheitsbasis.** Die Anwendung wird mit Go 1.26.5 und den aktuellen TLS-Sicherheitskorrekturen gebaut.

## [0.10.0] - 2026-07-14

### Verbessert

- **Handwerker-Koordination technisch vorbereitet.** Zugewiesene Dienstleister können Anliegen in klaren Schritten bearbeiten (angenommen, Termin vereinbart, in Arbeit, erledigt), Fotos auch ohne Text anhängen und einen echten Termin mit Datum und Uhrzeit vorschlagen. Echte Einladungen bleiben aus, bis eine externe fachkundige Person die Datenschutz- und Rechtsfragen geprüft und die Freigabe dokumentiert hat.
- **Mehr Nachvollziehbarkeit.** Alle Aktionen externer Dienstleister – inklusive Fotos und Rückfragen – sind jetzt lückenlos im Aktivitätsprotokoll der Verwaltung sichtbar.

## [0.9.0] - 2026-07-13

### Verbessert

- **Zugriffsschutz zentral abgesichert.** Anmeldung und Hauszuordnung werden jetzt an einer einzigen, zentralen Stelle geprüft, statt in jeder Seite einzeln. Dadurch kann keine neue Seite versehentlich ohne diese Prüfung ausgeliefert werden. Für Nutzer bleibt alles unverändert — die Absicherung wird nur strukturell zuverlässiger.
- **Formularschutz vereinheitlicht.** Alle absendenden Aktionen werden nun einheitlich gegen missbräuchliche Fremdaufrufe abgesichert.

## [0.8.0] - 2026-07-13

### Verbessert

- **Sicherheit und Datenschutz gestärkt.** Zugriffsgrenzen zwischen Häusern werden strenger geprüft; Formulare sind zusätzlich gegen missbräuchliche Fremdaufrufe abgesichert.
- **Zuverlässiger gegen Stromausfälle und Abstürze.** Gespeicherte Daten werden sicher auf die Festplatte geschrieben, und ein beschädigtes Protokoll kann den Start nicht mehr blockieren.
- **Stabiler im Betrieb.** Unerwartete Fehler führen nicht mehr zum Abbruch, und die Anwendung fährt bei Aktualisierungen sauber herunter.

## [0.7.0] - 2026-07-13

### Verbessert

- **Fundament für schnellere Weiterentwicklung gelegt.** Die Anwendung ist intern klar in Bausteine gegliedert; neue Funktionen lassen sich dadurch zügiger und sicherer ergänzen.
- **Qualitätssicherung ausgebaut.** Vergleichswerkzeuge und visuelle Prüfungen helfen, unbeabsichtigte Änderungen früh zu erkennen.

## [0.6.13] - 2026-07-09

### Verbessert

- **Formulare bleiben auf kleinen Bildschirmen handhabbar.** Längere Dialoge behalten ihre wichtigste Aktion sichtbar.
- **Speichern ist schneller erreichbar.** Umfangreiche Eingaben wie Abstimmungen scrollen ruhiger innerhalb des Fensters.

## [0.6.12] - 2026-07-09

### Verbessert

- **Die Seitenleiste bleibt aufgeräumt.** Auch lange Namen und viele freigeschaltete Bereiche passen sauber in die Portalnavigation.
- **Abmelden bleibt sichtbar.** Version, Rolle und Abmeldung halten verlässlich Abstand zum unteren Rand.

## [0.6.11] - 2026-07-09

### Verbessert

- **Der Audit-Log wirkt mehr wie eine Historie.** Sensible Aktionen sind nach Tagen gruppiert und schneller erfassbar.
- **Details bleiben erreichbar, ohne die Liste zu überladen.** Zusatzinformationen öffnen erst bei Bedarf; der Überblick bleibt ruhig.

## [0.6.10] - 2026-07-09

### Verbessert

- **Anliegen sind klarer geführt.** Melden, eigene Anliegen und Verwaltung sind jetzt sauber getrennt.
- **Das Triage-Board ist ruhiger.** Kommentar, Status und Zuständigkeit öffnen pro Anliegen erst bei Bedarf.

## [0.6.9] - 2026-07-09

### Verbessert

- **Das Portal kommt mobil schneller zum Inhalt.** Die Navigation bleibt vollständig erreichbar, nimmt aber nicht mehr den ersten Bildschirm ein.
- **Das Menü fühlt sich mehr nach App an.** Haus, Adresse, Versionshinweis und Abmelden bleiben im kompakten mobilen Menü sauber zusammen.

## [0.6.8] - 2026-07-09

### Verbessert

- **Dokumente sind leichter zu scannen.** Titel, Dateiname, Sichtbarkeit und Version bleiben kompakt zusammen.
- **Der Dokumentenbereich bleibt mobil ruhig.** Aktionen wie Vorschau, Download und Ersetzen haben Platz, ohne lange Dateinamen zu zerlegen.

## [0.6.7] - 2026-07-09

### Verbessert

- **Der Kostenblock passt besser auf kleine Bildschirme.** Die Preiszeile bleibt kurz, die Wohneinheit steht klar im erklärenden Text.
- **Kein seitliches Scrollen im Preisbereich.** Die Startseite bleibt auf dem Handy sauber im sichtbaren Bereich.

## [0.6.6] - 2026-07-09

### Verbessert

- **Faire Kosten sind klarer formuliert.** Der Richtwert bezieht sich kompakter auf Wohneinheiten.
- **Der Kostenblock liest sich mobil ruhiger.** Überschriften und Zeilen brechen sauberer um.

## [0.6.5] - 2026-07-09

### Verbessert

- **Benutzer & Rechte sind mobil lesbarer.** Rollen, Rechte und Anmeldestatus stehen jetzt in ruhigen Abschnitten statt in zu engen Spalten.
- **Keine zerhackten Rollenhinweise mehr.** Chips wie Eigentümer-Dokumente oder Plattformverwaltung bleiben auf kleinen Bildschirmen gut lesbar.

## [0.6.4] - 2026-07-09

### Verbessert

- **Aushänge und Termine sind mobil sauberer.** Bearbeiten- und Löschen-Aktionen bleiben jetzt sichtbar im jeweiligen Eintrag.
- **Mehr Ruhe in Verwaltungslisten.** Die Aktionsflächen haben auf kleinen Bildschirmen verlässlichen Platz und rutschen nicht mehr aus der Karte.

## [0.6.3] - 2026-07-09

### Verbessert

- **Parkplatznutzung startet klarer.** Wenn noch keine Monatswerte vorliegen, erklärt die Seite jetzt die nächsten Schritte statt leer zu wirken.
- **Passende Aktionen pro Rolle.** Verwaltung sieht Konfiguration und Zugriff, Bewohner bekommen einen ruhigen Weg zurück zum Hausüberblick.
- **Grenzen bleiben verständlich.** Die Seite spricht von Transparenz und privater Stellplatznutzung, nicht von Buchhaltung oder Mahnwesen.

## [0.6.2] - 2026-07-09

### Verbessert

- **Audit-Log auf dem Handy ruhiger.** Filter und Einträge passen sich jetzt sauber an kleine Bildschirme an.
- **Mehr Lesbarkeit bei sensiblen Aktionen.** Änderungen bleiben als klare Karten mit Statusmarkierung nachvollziehbar, ohne seitliches Scrollen.

## [0.6.1] - 2026-07-08

### Verbessert

- **BMD-Prüfung besser vorbereitet.** Für Steuerberater gibt es jetzt ein konkretes Rohdaten-Beispiel mit Feldliste.
- **Buchhaltung bleibt draußen.** Konten- und Steuerfelder werden im Kandidatenformat bewusst abgelehnt, bis ein echtes BMD-NTCS-Mapping bestätigt ist.

## [0.6.0] - 2026-07-08

### Neu

- **E-Rechnungen intern vorbereitet.** ebInterface-Dateien in den Profilen 5.0 und 6.0 können im internen, getesteten Importbaustein als Metadaten gelesen werden.
- **Geschützte Ablage vorbereitet.** Der interne Speicherbaustein legt E-Rechnungen geschützt ab; eine produktive Upload-Oberfläche folgt erst nach der externen Formatprüfung.

### Verbessert

- **Klare Grenze zur Buchhaltung.** ebInterface ist Empfang und Ablage, nicht Buchung, Steuerlogik oder Zahlungsauftrag.
- **Mehr Prüfbarkeit.** Golden Files und verständliche Fehlerberichte sichern den Import gegen falsche Profile und unvollständige Rechnungen ab.

## [0.5.0] - 2026-07-08

### Neu

- **Klarere Startseite.** hausv.org erklärt jetzt noch direkter, dass es um Kommunikation, Transparenz und Self-Service geht.
- **Ausblick ohne Nebel.** Pilotfunktionen und nächste Bausteine sind besser getrennt, damit nichts fertiger wirkt als es ist.
- **camt.054 vorbereitet.** Zahlungsavise können als zusätzlicher Statuskanal neben camt.053 bewertet und getestet werden.

### Verbessert

- **Faire Kosten genauer erklärt.** Die Preislogik spricht konsequent von Wohneinheiten; Zubehör wie Keller oder Stellplätze wird nicht automatisch als volle Einheit behandelt.
- **Österreich-First verständlicher.** BMD/RZL, camt und ebInterface sind als Anschluss an bestehende Systeme eingeordnet, nicht als eigene Buchhaltung.
- **Mehr Qualität bei Schnittstellen.** Golden Files, Profilprüfungen, Feldlimits und XSD-Gates sind als verbindliche Leitplanken dokumentiert.

## [0.4.0] - 2026-07-08

### Neu

- **Übergaben sauber dokumentieren.** Nutzerwechsel können mit Räumen, Zählern, Schlüsseln, Fotos und Notizen erfasst werden.
- **Bestätigung per Link.** Ausziehende und einziehende Personen können vorbereitete Protokolle über einen begrenzten Link bestätigen.
- **Protokoll als PDF.** Übergaben lassen sich exportieren und direkt als geschütztes Dokument zur passenden Einheit ablegen.

### Verbessert

- **Mehr klare Produktgrenzen.** Dienstleisterzugang, Bankdateien und Zahlungsaufträge sind für die österreichische Roadmap sauber abgegrenzt.
- **Anhänge bleiben einheitlich.** Übergabefotos nutzen dieselben geschützten Vorschauen, Lightboxen und Löschregeln wie andere Bereiche.

## [0.3.0] - 2026-07-08

### Neu

- **Dienstleister gezielt einbinden.** Externe Helfer sehen nur zugewiesene Anliegen und können dort Rückfragen, Status und Terminvorschläge ergänzen.
- **Adressbuch pro Haus.** Wiederkehrende Dienstleister, Hausmeister und Notdienste lassen sich zentral pflegen und direkt im Anliegen auswählen.
- **Kalender abonnieren.** Termine und passende Dienstleister-Zeitfenster können als geschützter Kalenderfeed genutzt werden.
- **Zahlungsstatus ohne Buchhaltung.** Verwaltung kann pro Einheit offen, bezahlt, teilbezahlt oder überfällig markieren; Bewohner sehen nur die eigene Einheit.

### Verbessert

- **Bessere Nachvollziehbarkeit.** Neue Workflows schreiben Audit-Spuren, ohne unnötige Details oder Secrets offenzulegen.
- **Sauberer Österreich-Fokus.** camt.053, Zahlungsreferenzen und Schnittstellenentscheidungen sind vorbereitet, ohne Buchhaltung ins Portal zu ziehen.
- **Mehr Sicherheit bei Dateien.** Anhänge bleiben über geschützte App-Wege erreichbar und werden rollenbasiert geprüft.

## [0.2.0] - 2026-07-08

### Neu

- **Mehr Ruhe auf der Startseite.** Vertrauen, Datenschutz und faire Kosten sind klarer getrennt und leichter zu erfassen.
- **Faire Nutzung nach Wohneinheiten.** Kostenloser Einstieg und spätere Richtpreise orientieren sich an Wohneinheiten statt an Hausadressen.
- **Versionsverlauf im Portal.** Die Versionsnummer in der Seitenleiste zeigt, was sich für die Hausgemeinschaft verbessert hat.
- **Bessere Rückmeldung bei Anhängen.** Ausgewählte Dateien zeigen Vorschau, Größe und Entfernen vor dem Speichern.

### Verbessert

- **Verwaltungsbereiche lesen sich klarer.** Audit-Log, Rollenhilfe, Abstimmungen und Zahlungswege wirken aufgeräumter.
- **Aktuelle Oberfläche nach jedem Update.** Neue Deployments laden ihre frischen Skripte zuverlässiger.

## [0.1.0] - 2026-07-07

### Neu

- **Erster Pilot für hausv.org.** Ein geschützter Portalrahmen, der Hausüberblick und die Parkplatznutzung für Janischhofweg 22 gehen an den Start.
- **Basisrollen eingerichtet.** Verwaltung und Bewohner erhalten getrennte Zugänge; weitere Rollen werden für den schrittweisen Ausbau vorbereitet.
- **Ausbau transparent vorbereitet.** Aushänge, Termine, Dokumente, Anliegen und Abstimmungen sind zunächst als nächste Portalbereiche sichtbar und folgen in späteren Releases.
