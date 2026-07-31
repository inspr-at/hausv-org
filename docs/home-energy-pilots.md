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

`flexibility` akzeptiert `shift`, `throttle`, `fixed` oder `unknown`. Nur
`shift` und `throttle` zählen zusammen mit `rated_power_kw` in die
Peak-Wirkung; ohne beides erscheint der Verbraucher im Verbrauch, verspricht
aber nichts. Fehlt `flexibility`, gilt die Vorbelegung der Art.

Benannte Verbraucher erhalten eine aus Art und Name abgeleitete, stabile ID.
Mehrere Verbraucher derselben Art bestehen damit nebeneinander, und ein
Neustart verdoppelt sie nicht.

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

## Smart-Meter-Referenz

Unterstützt wird UTF-8-CSV mit Komma oder Semikolon:

```csv
timestamp;import_kwh
2026-07-01T00:00:00+02:00;0,42
2026-07-01T00:15:00+02:00;0,38
```

- `timestamp` muss eine Zeitzone enthalten und exakt auf Minute 00, 15, 30 oder
  45 liegen.
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
