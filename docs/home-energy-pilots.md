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
    "assets": ["ev", "wallbox"],
    "complete": true
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

Die drei Slugs müssen zuvor jeweils als eigener Eintrag in `WEG_TENANTS_JSON`
existieren. So bleiben Personen, Daten, Geräte und Berechtigungen strikt
hausbezogen.

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
- Originaldatei und abgeleitete Intervalle bleiben dem Haus zugeordnet.

## Sicherheitszustände

1. `observe`: lesen und erklären; niemals schalten.
2. `recommend`: Empfehlungen, weiterhin ohne Gerätewirkung.
3. `shadow`: Entscheidungen protokollieren, weiterhin ohne Gerätewirkung.
4. `active`: nur nach separater Gerätefreigabe; stale Daten oder Adapterfehler
   führen zu Nicht-Eingreifen.

Der prominent sichtbare Hausschalter öffnet zunächst nur `shadow`. Zur
Freigabe muss ein Eigentümer, Haus-Admin oder eine ausdrücklich mit
`energy-control` berechtigte Vertrauensperson die Folgen bestätigen und
`AKTIVIEREN` eingeben. Die Rückkehr zu `Nur beobachten` wirkt unmittelbar.
