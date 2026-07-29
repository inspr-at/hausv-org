# JHW22: Art.-28-/TOM-Freigabepaket

Stand: 29. Juli 2026
Status: **Entwurf – nicht durch die verantwortliche Stelle freigegeben**

Dieses Dokument ergänzt die unveränderten
[EU-Standardvertragsklauseln 2021/915 für Verantwortliche und
Auftragsverarbeiter](https://eur-lex.europa.eu/eli/dec_impl/2021/915/oj?locale=de)
um die konkreten Anhänge für den technischen Betrieb von hausv.org am
Janischhofweg 22. Es ist kein Ersatz für die Zustimmung der tatsächlichen
verantwortlichen Stelle.

Bis die genaue Vertragspartei und eine vertretungsbefugte Person dieses Paket
nachweisbar genehmigt haben, bleibt
`SERVICE_PROVIDER_ACCESS_ENABLED=false`. Eine Freigabe durch den technischen
Betreiber allein ersetzt diese Genehmigung nicht.

## 1. Parteien und Freigabe

### Verantwortliche Stelle

Als sachlich naheliegende Vertragspartei ist im Entwurf die
Wohnungseigentümergemeinschaft Janischhofweg 22, 8043 Graz, vertreten durch die
Hausverwaltung, vorgesehen.

Die öffentlich recherchierte Hausverwaltung ist:

> Alwog-Allgemeine Wohnbaugesellschaft m.b.H.  
> Naglergasse 10  
> 8010 Graz  
> Firmenbuchnummer FN 58724i, Landesgericht für ZRS Graz

Quelle:
[WKO Firmen A–Z](https://firmen.wko.at/alwog-allgemeine-wohnbaugesellschaft-mbh/steiermark/?firmaid=add5c503-f1ba-4b5e-9811-2c6c38d3cb6f).

Vor Freigabe muss ALWOG bestätigen:

1. ob die Wohnungseigentümergemeinschaft, ALWOG oder beide als verantwortliche
   Stelle auftreten,
2. wer vertretungsbefugt zustimmt,
3. welche Kontaktadresse für Betroffenen- und Vorfallanfragen gilt,
4. dass Zweck, Datenarten, Fristen, Unterauftragsverarbeiter und
   Drittlandtransfers dieses Dokuments angewiesen beziehungsweise genehmigt
   sind.

### Auftragsverarbeiter

> Ing. Markus Barta  
> Janischhofweg 22/11  
> 8043 Graz, Österreich

Technischer Kontakt: `hello@hausv.org`

### Dokumentierte Zustimmung

| Rolle | Name/Funktion | Datum | Zustimmung/Signatur |
|---|---|---|---|
| Verantwortliche Stelle | noch zu bestätigen |  |  |
| Auftragsverarbeiter | Ing. Markus Barta |  |  |

Eine elektronisch nachvollziehbare Zustimmung ist ausreichend. Das freigegebene
Exemplar beziehungsweise der eindeutige Zustimmungsnachweis wird außerhalb des
öffentlichen Repositories bei der Vertragsdokumentation aufbewahrt; Datum,
Prüfrevision und Ergebnis werden im PPM-Ticket dokumentiert.

## 2. Beschreibung der Verarbeitung

| Merkmal | Vereinbarung |
|---|---|
| Gegenstand | Technischer Betrieb des privaten Hausportals hausv.org für JHW22 |
| Dauer | Ab dokumentierter Freigabe bis Beendigung des Betriebsauftrags; Löschung oder Rückgabe nach Abschnitt 6 |
| Art | Erheben durch Eingaben, Speichern, Anzeigen, Ordnen, Übermitteln an berechtigte Empfänger, Sichern, Exportieren und Löschen |
| Zweck | Hauskommunikation, Dokumente, Termine, Anliegenbearbeitung, Abstimmungen, Parkplatz-/Ladeverwaltung, Zugriffsschutz und Sicherheitsnachweis; Erfassen des Energie-Inventars, lesende Visualisierung bestätigter Messpunkte, Viertelstunden-/Spitzenplanung sowie nachvollziehbare Energie-, Optimierungs- und Wartungsempfehlungen |
| Betroffene | Eigentümer, Bewohner, Beirat, Hausverwaltung, eingeladene Dienstleister und technische Administratoren |
| Datenarten | Name, E-Mail, Telefon, Einheit und Hauszugehörigkeit; Rollen/Rechte; Anmelde- und Auditdaten; Aushänge, Termine, Anliegen, Kommentare, Abstimmungen; Dokumente und Anhänge; Parkplatz-/Lademesswerte; Zuhause-/Energieprofil, Anlagen und bestätigte Home-Assistant-Zuordnungen; Smart-Meter-Originale und Viertelstundenwerte; daraus abgeleitete Spitzen, Tarifstände, Empfehlungen und Maßnahmen |
| Besondere Daten | Nicht beabsichtigt. Freitext und Bilder dürfen keine Gesundheitsdaten, Ausweiskopien oder sonstigen Art.-9-/10-Daten enthalten. Versehentliche Inhalte werden nach Meldung eingeschränkt und gelöscht. |
| Weisungen | Dieses Paket, die sichtbare Datenschutzinformation, dokumentierte spätere Weisungen der verantwortlichen Stelle und zwingendes EU-/österreichisches Recht |

Der vollständige technische Feld- und Datenfluss steht in
[`datenfluss-dienstleister.md`](datenfluss-dienstleister.md).

Die Energieverarbeitung bleibt in diesem Paket ausschließlich lesend und
empfehlend. Sie schaltet keine Geräte und trifft keine ausschließlich
automatisierte Entscheidung mit rechtlicher oder ähnlich erheblicher Wirkung.
Eine spätere aktive Steuerung benötigt eine neue, ausdrückliche Freigabe und
Datenschutzprüfung.

## 3. Pflichten und Zusammenarbeit

Der Auftragsverarbeiter:

- verarbeitet Daten nur auf dokumentierte Weisung und weist unverzüglich auf
  eine aus seiner Sicht rechtswidrige Weisung hin;
- stellt sicher, dass nur zur Vertraulichkeit verpflichtete Personen Zugriff
  erhalten;
- unterstützt die verantwortliche Stelle bei Auskunft, Export, Berichtigung,
  Einschränkung, Löschung, Widerspruch und Datenübertragbarkeit;
- informiert die verantwortliche Stelle unverzüglich über eine Verletzung des
  Schutzes personenbezogener Daten und liefert die verfügbaren Angaben zu
  Umfang, Betroffenen, Folgen und Abhilfe;
- unterstützt die Pflichten nach Art. 32 bis 36 DSGVO;
- stellt die für Rechenschaft und angemessene Prüfung erforderlichen Nachweise
  bereit, ohne Geheimnisse oder Daten anderer Systeme offenzulegen;
- setzt keine neuen Unterauftragsverarbeiter ohne die allgemeine Genehmigung
  und das Änderungsverfahren aus Abschnitt 5 ein.

Mangels absehbar verfügbarer externer Prüfstelle erfolgt die regelmäßige
Kontrolle als dokumentierte Betreiber-Selbstprüfung gegen Primärquellen,
Produkt- und Infrastrukturtests sowie Anbieterunterlagen. Das beseitigt weder
die Kontrollrechte der verantwortlichen Stelle noch die Pflicht, bei einem
nicht ausreichend geminderten hohen Risiko die Datenschutzbehörde nach Art. 36
DSGVO zu konsultieren.

## 4. Technische und organisatorische Maßnahmen

### Vertraulichkeit und Zugriff

- Dienstleisterzugriff ist fail-closed und erfordert einen expliziten
  Aktivierungsschalter plus exakt passende Prüfrevision.
- Rollen und Rechte gelten je Haus. Dienstleister sehen nur offene, ihnen
  konkret zugewiesene Anliegen; Entzug und Schließen wirken sofort.
- Dateiabrufe prüfen Sitzung, Haus, Rolle und konkretes Objekt. Negative
  Berechtigungstests laufen automatisiert.
- Anmeldung erfolgt über kurzlebige, einmal nutzbare E-Mail-Links oder das
  selbst betriebene Zitadel. Sitzungen sind signiert, zeitlich begrenzt und bei
  Sperre nicht weiter nutzbar.
- Verwaltungsänderungen erfordern authentifizierte, berechtigte Sitzungen und
  Schutz gegen herkunftsfremde Formularanfragen.
- Geheimnisse liegen in agenix/Hostkonfiguration und nicht im
  Anwendungsrepository.

### Übertragung und Trennung

- Öffentliche Verbindungen sind TLS-geschützt; Cloudflare und Traefik vermitteln
  den Webzugriff.
- Datenzugriff wird nach Haus, Rolle und Objekt getrennt. Die
  Anwendungsablage ist ausschließlich dem HAUSV-Dienst zugeordnet.
- Das Frontend lädt keine externen Analyse-, Werbe- oder Schriftressourcen.
- Sicherungen werden vor der Übertragung mit Restic verschlüsselt.

### Integrität und Nachvollziehbarkeit

- Sicherheitsrelevante Anmeldungen, Rechte-, Zuweisungs-, Datei- und
  Löschvorgänge werden ohne unnötige Inhaltsdaten protokolliert.
- Schreibvorgänge und Migrationen sind automatisiert getestet; Dateiablagen
  werden objektbezogen referenziert.
- Versionierte Releases, reproduzierbare Container-Builds, manuelle
  Produktionsfreigabe und ein dokumentierter Rollback begrenzen
  Änderungsrisiken.

### Verfügbarkeit und Wiederherstellung

- HAUSV läuft auf dem Netcup-Host `csb1` in Wien mit persistenter
  Dienstablage.
- Verschlüsselte Restic-Sicherungen der Dienstablage laufen täglich in eine
  Hetzner Storage Box. Die Aufbewahrung umfasst tägliche, wöchentliche,
  monatliche und jährliche Stände.
- Gesundheitsprüfungen kontrollieren Anwendung, Datenbank und beschreibbare
  Ablage; Betriebsüberwachung meldet fehlgeschlagene Sicherungen.
- Wiederherstellung und Rollback sind im Betriebsrunbook beschrieben. Ein
  Restore-Test ist nach wesentlichen Speicheränderungen und mindestens jährlich
  zu dokumentieren.

### Regelmäßige Bewertung

- Automatisierte Tests prüfen Rollen-, Objekt- und Mandantengrenzen sowie den
  geschlossenen Gate-Zustand.
- Anbieter-DPA, Unterauftragsverarbeiter, Speicherorte und
  Drittlandmechanismen werden mindestens jährlich und bei
  Änderungsmitteilungen geprüft.
- Neue Datenarten, Empfänger, Speicherorte, Hochrisikoverarbeitung oder
  Sicherheitsvorfälle schließen das Gate bis zur Neubewertung.

## 5. Genehmigte Unterauftragsverarbeiter

Vorgesehen ist eine **allgemeine schriftliche Genehmigung** der folgenden
Liste. Der Auftragsverarbeiter informiert die verantwortliche Stelle mindestens
14 Tage vor einer Hinzufügung oder Ersetzung. Ein begründeter Widerspruch wird
vor Einsatz geklärt; andernfalls bleibt die betroffene Funktion geschlossen.

| Anbieter | Leistung und Ort | Daten/Schutzmechanismus | Vertragsnachweis |
|---|---|---|---|
| netcup GmbH | VPS-Hosting, Serverstandort Wien, Österreich | Fachdaten, selbst betriebenes Zitadel, Verbindungs- und Betriebsdaten; EU-Verarbeitung | Kundenkonto-AVV/TOM vor Freigabe exportieren oder anderweitig nachweisbar bestätigen |
| Cloudflare, Inc. | DNS, Reverse Proxy, DDoS-/Webschutz, globales Netz | technisch notwendige Verbindungsdaten und vermittelter Webverkehr; DPA, EU-SCC und veröffentlichte TOM | DPA Version 6.4 vom 03.04.2026 geprüft; kontobezogenen Geltungsnachweis vor Freigabe sichern |
| Hetzner Online GmbH | Storage Box für Sicherungen, EU | ausschließlich Restic-verschlüsselte Sicherungsdaten; DPA/TOM | DPA im Kundenkonto und aktuelle Unterauftragnehmerliste vor Freigabe nachweisbar bestätigen |
| Resend, Inc. | Transaktionsmail; Versandregion `eu-west-1` (Irland) | Empfänger, Betreff, Inhalt; Accountdaten, Metadaten, Logs und API-Aufzeichnungen in den USA; DPA, SCC 2021/914 Module 2/3 und EU-US DPF | aktive Domain `notify.hausv.org` und Region am 26.07.2026 per API bestätigt; DPA-Fassung 31.12.2025 geprüft; ausgeführtes Exemplar aus dem Dashboard noch zu sichern |

Zitadel ist selbst auf `csb1` betrieben und daher kein zusätzlicher externer
Empfänger. Home Assistant erhält im internen Netz keine Dienstleisterdaten.

Resend veröffentlicht eine allgemeine Unterauftragsverarbeiter-Genehmigung mit
14 Tagen Änderungsfrist. Die am 26.07.2026 geprüfte, seit 15.07.2026 gültige
Liste nennt AWS, Anthropic, Attio, Cloudflare, Datadog, Elastic, Estuary,
Google, Inngest, Liveblocks, Metabase, Not Just Tickets, PlanetScale, Retool,
RunPod, Salesforce, Snowflake, Stripe, Supabase, Svix, Tinybird und Vercel,
jeweils in den USA. Maßgeblich ist die
[aktuelle Resend-Liste](https://resend.com/legal/subprocessors).

## 6. Aufbewahrung, Rückgabe und Löschung

- Einmalige Magic-Links: 15 Minuten; OIDC-Vorgänge: 10 Minuten.
- Sitzungen: regulär höchstens 30 Tage beziehungsweise bis Abmeldung oder
  Sperre.
- Zugehörigkeiten und Dienstleisterprofile: bis Entzug und Wegfall des Zwecks.
- Geschlossene Anliegen, Kommentare und Anhänge: jährliche Prüfung, regulär
  drei Jahre nach Abschluss; länger nur mit dokumentierter offener
  Gewährleistungs-, Rechts- oder Nachweispflicht.
- Gelöschte Anhangdateien: sofort; verbleibende Löschmarkierung nach einem Jahr.
- Smart-Meter-Originaldateien werden nach 30 Tagen, normalisierte
  Viertelstundenwerte nach 13 Monaten und gespeicherte Tarifbewertungen nach
  drei Jahren zur Löschung fällig. Die technische Bereinigung läuft beim Start
  und anschließend alle sechs Stunden; die tatsächliche Löschung erfolgt damit
  spätestens sechs Stunden nach dem jeweiligen Fristablauf.
- Energieprofil, Anlagen und bestätigte Zuordnungen: bis zur Korrektur, Trennung
  oder ausdrücklichen Löschung. Home-Assistant-Zustände und -Verläufe werden
  nur transient für die Anzeige gelesen.
- Der einmalige Beginn des dreijährigen kostenlosen Nutzungszeitraums bleibt
  auch nach Löschung des Energieprofils bis zum Ende des Anspruchs- oder
  Portalverhältnisses erhalten. Er verhindert einen Neustart des Zeitraums und
  enthält keine Messwerte.
- Auditdaten werden nach drei Jahren zur Löschung fällig und spätestens beim
  nächsten sechsstündlichen Bereinigungslauf entfernt.
- Resend hält reguläre E-Mail-Inhalte laut Anbieter 30 Tage. Nach
  Vertragsbeendigung löscht Resend verbleibende Kunden-/Nutzerdaten laut DPA
  innerhalb von 90 Tagen.

Bei Ende des Betriebsauftrags stellt der Auftragsverarbeiter auf Weisung einen
maschinenlesbaren Export bereit und löscht die produktiven Daten nach
bestätigter Übergabe. Sicherungskopien laufen entsprechend ihrer technischen
Rotation aus und bleiben bis dahin gesperrt und zweckgebunden. Gesetzlich
erforderliche Restdaten werden isoliert und nur für diesen Nachweiszweck
aufbewahrt.

## 7. Kontroll- und Freigabeprotokoll

Vor Aktivierung sind alle Punkte mit Datum im PPM-Ticket nachzuweisen:

- [ ] genaue verantwortliche Vertragspartei und vertretungsbefugte Person
      bestätigt;
- [ ] EU-Standardvertragsklauseln 2021/915 plus diese Anhänge nachweisbar
      genehmigt;
- [ ] allgemeine Unterauftragsverarbeiter-Genehmigung einschließlich
      Drittlandtransfers erteilt;
- [ ] kontobezogene AVV-/DPA-Nachweise für Netcup, Cloudflare, Hetzner und
      Resend gesichert;
- [ ] Kontakt- und Löschprozess organisatorisch zugewiesen;
- [ ] Datenschutzinformation gegen freigegebenes Exemplar geprüft;
- [ ] explizite Go-Entscheidung mit Datum und Prüfrevision dokumentiert.

Erst danach werden die beiden Gate-Werte gesetzt und der begrenzte
Dienstleister-Pilot mit einem Testkonto geprüft. Ohne diese Punkte ist das
Ergebnis **No-Go**.

## 8. Primärquellen

- [DSB: Pflichten von Auftragsverarbeitern](https://dsb.gv.at/rechte-pflichten/ihre-pflichten-als-auftragsverarbeiterin)
- [EU-Standardvertragsklauseln 2021/915](https://eur-lex.europa.eu/eli/dec_impl/2021/915/oj?locale=de)
- [Resend DPA](https://resend.com/legal/dpa)
- [Resend Datenregionen](https://resend.com/docs/dashboard/domains/regions)
- [Resend Unterauftragsverarbeiter](https://resend.com/legal/subprocessors)
- [Cloudflare Customer DPA](https://www.cloudflare.com/cloudflare-customer-dpa/)
- [Hetzner: Datenschutz und AVV](https://docs.hetzner.com/de/general/company-and-policy/data-protection-at-hetzner/)
- [Hetzner TOM](https://docs.hetzner.com/general/security-and-identify/technical-and-organizational-measures/)
