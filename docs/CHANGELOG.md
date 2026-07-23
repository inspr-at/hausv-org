# Versionsverlauf

Alle sichtbaren Änderungen werden hier in kurzer, kundenorientierter Sprache gesammelt.

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
