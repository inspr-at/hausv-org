# Strukturierte Log-QA

Die Betriebslogs sind JSON-Zeilen auf `stderr`. Request-Einträge enthalten die
stabile Route, das Haus, Status, Laufzeit und eine zufällige Request-ID. Der
konkrete URL-Pfad wird bewusst nicht gespeichert: Kalender- und
Übergabe-Adressen enthalten Zugriffstoken; andere Pfade enthalten unnötige
Objektkennungen.

Der normale Playwright-Hauptlauf prüft die dabei entstehenden lokalen Logs
automatisch mit erfundenen Daten:

```fish
scripts/qa-main-flows.sh
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

Ein deklarativer csb1-Wächter prüft zusätzlich alle fünf Minuten das öffentliche
`/healthz`, den Containerzustand sowie neue strukturierte Fehlerklassen. Das
Healthsignal umfasst auch einen fehlgeschlagenen laufenden
Energie-Aufbewahrungslauf; der Wächter baut dafür keine zweite
Aufbewahrungslogik. Alle `ERROR`-Ereignisse und ausgewählte betriebsrelevante
`WARN`-Ereignisse werden ausschließlich in stabile Kategorien und Zähler
übersetzt. Rohe Logtexte, Fehlerwerte, Bewohnerdaten, Objektkennungen, URLs und
Secrets werden weder in den Alarm noch in dessen Zustand übernommen.

Alarm und Entwarnung werden nur bei einem Zustandswechsel über den bestehenden
csb1-Betriebskanal zugestellt. Ein Zustellfehler bleibt ausstehend und wird
erneut versucht. Der Deploy-Runner prüft weiterhin die frischen Containerlogs
auf Start-, Import-, Panic- und Fatalfehler; Ad-hoc-Filter verwenden die
JSON-Felder `level`, `msg`, `route`, `tenant`, `status` und `request_id`.
