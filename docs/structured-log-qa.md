# Strukturierte Log-QA

Die Betriebslogs sind JSON-Zeilen auf `stderr`. Request-Einträge enthalten die
stabile Route, das Haus, Status, Laufzeit und eine zufällige Request-ID. Der
konkrete URL-Pfad wird bewusst nicht gespeichert: Kalender- und
Übergabe-Adressen enthalten Zugriffstoken; andere Pfade enthalten unnötige
Objektkennungen.

Der normale Playwright-Hauptlauf prüft die dabei entstehenden lokalen Logs
automatisch mit erfundenen Daten:

```fish
scripts/qa-main-flows.fish
```

Eine Produktionsstichprobe kann geprüft werden, ohne Logwerte auf dem lokalen
Bildschirm auszugeben:

```fish
ssh -p 2222 mba@cs1.barta.cm \
  "docker logs hausv-org --since 24h 2>&1" \
  | scripts/check-structured-logs.py
```

Die Prüfung schlägt fehl bei ungültigem JSON, unbekannter Severity, fehlenden
Request-Feldern, konkreten Pfaden, Klartext-E-Mail-Adressen, sensitiven
Query-Parametern sowie Feldern für Token, Passwörter, Chat-IDs oder
Request-Bodies. Sie gibt ausschließlich Zähler aus.

Es gibt derzeit kein externes Logbackend und keine automatische Alarmierung.
Der Deploy-Runner prüft die frischen Containerlogs auf Start-, Import-, Panic-
und Fatalfehler; tiefergehende Filter werden bei Bedarf direkt auf den
JSON-Feldern `level`, `msg`, `route`, `tenant`, `status` und `request_id`
ausgeführt.
