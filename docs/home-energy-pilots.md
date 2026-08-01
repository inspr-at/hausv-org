# HAUSV Zuhause: Pilot- und Connector-Konfiguration

Diese Spezifikation beschreibt die konkreten Konfigurationsformate für private
Hausprofile und Home-Assistant-Connectoren. Produktstatus und Backlog bleiben in
PPM.

## Drei getrennte Pilotprofile

`HOME_PROFILE_SEEDS_JSON` legt ausschließlich noch nicht vorhandene Profile an.
Bestehende Profile werden bei einem Neustart nie überschrieben. Ein Seed kann
keinen Steuerungsmodus setzen: jedes Profil beginnt immer mit
`Nur beobachten`.

```json
[
  {
    "tenant_slug": "jhw22",
    "household_name": "Wohnung Barta",
    "home_type": "apartment",
    "assets": ["ev", "wallbox"]
  },
  {
    "tenant_slug": "eltern",
    "household_name": "Haus Eltern",
    "home_type": "house",
    "assets": ["pv", "ev", "hot-water", "heat-pump"]
  },
  {
    "tenant_slug": "schwiegereltern",
    "household_name": "Haus Schwiegereltern",
    "home_type": "house",
    "assets": ["pv", "battery", "ev"]
  }
]
```

### Benannte Verbraucher

Ein Eintrag in `assets` ist entweder eine Vorlagen-Art als Zeichenkette oder ein
Objekt mit eigenen Eigenschaften. Beide Schreibweisen dürfen gemischt werden;
die Kurzform bleibt unverändert gültig.

```json
{
  "tenant_slug": "eltern",
  "household_name": "Haus Eltern",
  "home_type": "house",
  "assets": [
    "pv",
    "ev",
    { "kind": "sauna", "name": "Sauna Keller", "rated_power_kw": 8, "flexibility": "shift" },
    { "kind": "other", "name": "Werkstatt", "rated_power_kw": 4, "flexibility": "throttle" }
  ]
}
```

`flexibility` akzeptiert `shift`, `throttle`, `fixed` oder `unknown`; alles
andere wird stillschweigend zu `unknown`. Nur `shift` und `throttle` zählen in
die Peak-Wirkung, `fixed` und `unknown` nicht.

Fehlt eine Angabe, greift die Vorbelegung der Art — ein Seed ist eine Vorlage,
keine Messung:

| fehlendes Feld | Vorbelegung |
|---|---|
| `flexibility` | `ev`, `wallbox`, `hot-water`, `sauna` → `shift`; `heat-pump`, `battery`, `air-conditioning` → `throttle`; sonst `unknown` |
| `rated_power_kw` | `ev`, `wallbox`, `battery` → 3 kW; `hot-water`, `heat-pump` → 1 kW; sonst 0 kW |

Ein `{"kind":"ev","name":"Zweitauto"}` ohne weitere Felder wird also mit 3 kW
als verschiebbar gerechnet. Wer das nicht will, setzt `flexibility` auf `fixed`.

Zwei Dinge sind ausgenommen: eine PV-Anlage zählt nie in die Peak-Wirkung —
Erzeugung verschiebt die Bezugsspitze nicht —, und `ev` und `wallbox` zählen
zusammen nur einmal, weil sie dieselbe Ladelast beschreiben. Von beiden gewinnt
der Eintrag mit erfasster `rated_power_kw`.

Benannte Verbraucher erhalten eine aus Art und Name abgeleitete, stabile ID.
Mehrere Verbraucher derselben Art bestehen damit nebeneinander, und ein
Neustart verdoppelt sie nicht. Zwei Namen, die auf dieselbe ID normalisieren
(`"Sauna Keller"` und `"sauna-keller"`), lassen den Start scheitern statt sich
gegenseitig zu überschreiben — ebenso ein Eintrag ohne `kind` oder ein Name
ganz ohne Buchstaben und Ziffern. Seeding läuft genau einmal pro Zuhause; ein
still verworfener Eintrag käme nie wieder.

`complete` ist optional und hat eine Nebenwirkung, die man kennen muss: es
überspringt das Onboarding auf Schritt 5 **und startet die drei Jahre
kostenfreie Nutzung**. Ohne das Feld beginnt der Haushalt regulär bei Schritt 1.
Eine Betriebsart kann ein Seed nie setzen — er kann keine Steuerung
freischalten.

Genau dafür führt `scripts/snapshot/env.sh` einen vierten Mandanten `cockpit`
mit `complete: true`. Die drei anderen QA-Mandanten fahren im Prüflauf den
Einrichtungsassistenten durch und dürfen deshalb nicht vorab abgeschlossen sein
— ohne diesen vierten zeigte jede Aufnahme von `/app/energie` den Assistenten
statt des Cockpits. Der Mandant liest bewusst dieselbe Home-Assistant-Fixture
wie `jhw22` und braucht keine eigene Kopie. Für eine sofort reproduzierbare
Momentaufnahme importiert der Prüflauf weiterhin eine Viertelstunden-CSV **des
laufenden Kalendermonats**: Der Home-Assistant-Sampler schreibt erst nach einer
vollständig abgelaufenen Viertelstunde. `PeakForMonth` filtert außerdem nach
Kalendermonat; ältere Zeitstempel lassen jede Kennzahl auf null stehen.

Die drei Slugs müssen zuvor jeweils als eigener Eintrag in `WEG_TENANTS_JSON`
existieren. So bleiben Personen, Daten, Geräte und Berechtigungen strikt
hausbezogen.

Der aktuelle Pilot unterstützt bewusst genau ein wirksames Zuhause-Profil und
einen Home-Assistant-Connector je Tenant. Mehrere Zuhause oder mehrere
Connectoren innerhalb desselben Tenants sind noch kein unterstützter
Produktumfang. Diese Erweiterung gehört zu HAUSV-402; bis dahin erhalten
getrennte Pilot-Haushalte jeweils einen eigenen Tenant.

## Home Assistant pro Haus

`HA_CONNECTORS_JSON` enthält nur nicht geheime Zuordnung. Genau eine
Credential-Quelle ist je Haus erforderlich:

- `token_file`: Pfad zu einer von agenix/Hostkonfiguration bereitgestellten
  Datei, oder
- `token_env`: Name einer extern gesetzten Umgebungsvariable.

Inline-Tokens sind nicht Teil des Formats.

```json
[
  {
    "tenant_slug": "jhw22",
    "base_url": "https://home-assistant.example.internal",
    "token_file": "/run/agenix/hausv-jhw22-ha-token"
  },
  {
    "tenant_slug": "eltern",
    "base_url": "https://home-assistant-eltern.example.internal",
    "token_file": "/run/agenix/hausv-eltern-ha-token"
  },
  {
    "tenant_slug": "schwiegereltern",
    "base_url": "https://home-assistant-schwiegereltern.example.internal",
    "token_file": "/run/agenix/hausv-schwiegereltern-ha-token"
  }
]
```

Discovery ruft ausschließlich `GET /api/states` ab. Gefundene Sensoren werden
erst nach verständlicher Bestätigung als Messpunkte gespeichert. Schalter und
Leuchten werden nicht vorgeschlagen und Discovery erteilt keinerlei
Steuerrecht.

Ein bestätigter Messpunkt kann optional mit genau einer Anlage desselben Hauses
verbunden sein. PV- und Batteriesensoren werden nur dann automatisch
zugeordnet, wenn im Haus genau eine passende Anlage existiert. Alle anderen
Zuordnungen bleiben eine bewusste Auswahl. Im Cockpit steht anschließend in
Alltagssprache, welche Bereiche tatsächlich gemessen und welche lediglich
erfasst sind; technische Entity-IDs bleiben in eingeklappten Einrichtungsdetails.

## Viertelstunden aus Home Assistant

Ist im Haus genau ein Messpunkt mit der Metrik `grid-import-power` **bestätigt**,
zeichnet HAUSV die abgeschlossenen Viertelstunden selbst auf. Ohne diese
Aufzeichnung entsteht auf einem Haus ohne Smart-Meter-Export nie eine
Bemessungsgrundlage — die Tarifkarte bliebe dauerhaft leer.

- Abtastung alle `ENERGY_SAMPLE_INTERVAL` (Vorgabe `30s`, also 30 Messwerte je
  Viertelstunde). `0` schaltet die Aufzeichnung ab.
- Gelesen wird ausschließlich `GET /api/states/<entity>`. Kein Dienstaufruf,
  kein Schreibzugriff; die Aufzeichnung läuft daher auch in `observe`.
- Geschrieben wird eine Viertelstunde erst, wenn sie vorbei ist, ausgerichtet
  auf :00/:15/:30/:45 in der Zeitzone des Hauses.
- Ein Messwert gilt höchstens zwei Abtastschritte lang weiter. Mindestens 90 %
  der Viertelstunde müssen so belegt sein, sonst wird kein Wert geschrieben.
- Die Güte ist höchstens `estimated`: zwischen zwei Abtastungen bleibt die
  Leistung angenommen, nicht gemessen. `gap` steht für fehlende, `stale` für
  eingefrorene Messwerte; beide werden ohne Wert festgehalten, statt über die
  Lücke zu mitteln.
- Quelle ist `home-assistant`. Sie bleibt neben `smart-meter` bestehen und wird
  nie mit ihr verschmolzen; das Cockpit stellt beide Monatsspitzen gegenüber.
- Aufbewahrung, Export und Löschung sind dieselben wie für importierte
  Viertelstundenwerte.

Gemessen wird die Momentanleistung und nicht der kumulative kWh-Zähler
(`grid-import-energy`): Der Leistungstarif bemisst die mittlere Bezugsleistung
einer Viertelstunde, und Zählerstände in Home Assistant sind typischerweise auf
0,1 oder 1 kWh gerundet — über eine Viertelstunde wären das mehrere hundert Watt
Unsicherheit auf einem verrechneten Wert.

## Smart-Meter-Referenz

Unterstützt wird UTF-8-CSV mit Komma oder Semikolon:

```csv
timestamp;import_kwh
2026-07-01T00:00:00+02:00;0,42
2026-07-01T00:15:00+02:00;0,38
```

- `timestamp` muss exakt auf Minute 00, 15, 30 oder 45 liegen. Mit Zeitzone
  (`2026-07-01T00:15:00+02:00`) ist die Angabe eindeutig — das ist die
  empfohlene Form. Ohne Zeitzone werden auch `2026-07-01 00:15:00`,
  `2026-07-01 00:15` und `01.07.2026 00:15` gelesen und in der Zeitzone des
  Hauses ausgelegt; bei der Umstellung auf Winterzeit ist eine solche Angabe
  zweideutig.
- `import_kwh` ist die in dieser Viertelstunde aus dem Netz bezogene Energie.
- Dieselbe Datei kann wiederholt importiert werden, ohne doppelte Intervalle zu
  erzeugen.
- Originaldatei und abgeleitete Intervalle bleiben strikt dem Haus zugeordnet.
  Originaldateien werden nach 30 Tagen und normalisierte Viertelstundenwerte
  nach 13 Monaten zur Löschung fällig; der automatische Lauf entfernt sie
  spätestens sechs Stunden später. Eigentümer und Hausadministration können
  beides vorher unter `/app/settings/energy-data` exportieren oder den gesamten
  Messverlauf beziehungsweise das Energieprofil bewusst löschen.

## Sicherheitszustände

1. `observe`: lesen und erklären; niemals schalten.
2. `recommend`: Empfehlungen, weiterhin ohne Gerätewirkung.
3. `shadow`: Entscheidungen protokollieren, weiterhin ohne Gerätewirkung.
4. `active`: nur nach separater Gerätefreigabe; stale Daten oder Adapterfehler
   führen zu Nicht-Eingreifen.

Der prominent sichtbare Hausschalter startet zunächst nur einen wirkungslosen
`shadow`-Testlauf. Dabei werden mögliche Entscheidungen protokolliert, aber kein
Gerät geschaltet. Zum Start muss ein Eigentümer oder die Hausadministration die
Folgen bestätigen und `TESTLAUF` eingeben. Dieser Testlauf ist keine Freigabe
für aktive Steuerung; dafür ist später eine separate, gerätespezifische
Freigabe erforderlich. Technische Vertrauenspersonen dürfen beim Einrichten
helfen, können den Testlauf aber niemals starten – auch nicht mit einem
historisch noch gespeicherten `energy-control`-Recht. Die Rückkehr zu
`Nur beobachten` wirkt unmittelbar.
