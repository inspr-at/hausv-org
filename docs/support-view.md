# Supportansicht

Die Supportansicht öffnet die konkrete Mitgliedschaft einer anderen Person für höchstens 15 Minuten. Die Auswahl erfolgt innerhalb der aktuell geöffneten Liegenschaft über **Ansicht als … → Bestimmte Person (Supportansicht)**.

## Freigabe

Erforderlich sind die echte Rolle `Admin` und das separate Profilrecht `support-view` in der gewählten Liegenschaft. Das Recht wird nicht aus der Adminrolle abgeleitet. Bei bearbeitbaren Zugängen kann ein Admin es unter **Benutzer & Rechte → Zugang bearbeiten → Anmeldung & Sonderrechte** vergeben. Geschützte, deklarativ konfigurierte Administratorkonten erhalten es ausdrücklich in ihrer Profilkonfiguration; die üblichen Schutzregeln für diese Konten gelten weiter.

## Session-Vertrag

- `Email`, `Role`, `TenantSlug`, `AuthMethod` und `ExpiresAt` bewahren den ursprünglichen Login. `SupportTargetEmail`, `SupportTargetRole`, `SupportStartedAt` und `SupportExpiresAt` gehören zur verschlüsselten, authentifizierten Session; URL-Parameter aktivieren keine Supportansicht.
- Beginn und Ende rotieren die Session. Die ursprüngliche Loginfrist wird nicht verlängert. Eine Supportansicht kann nicht mit der allgemeinen Rollenvorschau kombiniert werden.
- Jede authentifizierte Anfrage prüft Freigabe, aktiven Adminzugang und unveränderte, aktive Zielmitgliedschaft erneut. Die ausgewählte Liegenschaft bleibt verbindlich, auch bei Personen mit mehreren Mitgliedschaften.
- Autorisierung, persönliche Sonderrechte und Datenfilter verwenden die Zielperson. Alle Portal-POSTs sind gesperrt; nur das explizite Ende und Abmelden bleiben möglich. Mitteilungen werden nicht als gelesen markiert. Kalender-Abonnementlinks werden nicht ausgegeben, damit keine länger nutzbaren Zugangstoken für die Zielperson entstehen.
- `POST /app/support-view/start` verlangt `tenant`, `target_email`, `target_role`, eine gültige Session und dieselbe Origin. `POST /app/support-view/end` stellt den weiterhin gültigen eigenen Kontext wieder her.
- Start benötigt ein erfolgreich geschriebenes Audit-Ereignis. `session.support-view.start` und `session.support-view.end` enthalten die echte Administratoridentität, Zielperson, Liegenschaft, Zielrolle und Zeitpunkte. Dateiabrufe tragen zusätzlich den Zielkontext. Abmelden, erneute Anmeldung, Rechteentzug und Ablauf beenden die Ansicht. Bei Ablauf oder Rechteentzug wird das Ende beim nächsten Request protokolliert; ein ohne weitere Anfrage geschlossener Browser erzeugt kein zusätzliches Endereignis. Der protokollierte Ablaufzeitpunkt begrenzt den Zugang trotzdem.

## Prüfung

`go test ./internal/auth ./internal/server -run 'Support|NonAdminCannotGrant|Relogin'` prüft unter anderem Session-Manipulation, Freigaben, dynamische Sonderrechte, alle registrierten Portal-Schreibaktionen, Tenant-Isolation, Lesezustände und Audit-Zuordnung.

`scripts/qa-main-flows.sh` enthält zusätzlich `scripts/snapshot/qa-support-view.mjs`: Auswahl, Beginn und Ende sowie die permanente Kopfzeile auf mehreren Seiten bei 320, 390, 768 und 1440 Pixel Breite. Das Skript akzeptiert ausschließlich lokale QA-Fixtures.
