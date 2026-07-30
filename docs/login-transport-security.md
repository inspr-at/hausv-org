# Login- und HTTP-Schutzkette

Stand: 30.07.2026 · HAUSV-407

Dieses Dokument beschreibt den konkreten Vertrag zwischen Cloudflare,
Traefik/cloudflarewarp und der HAUSV-Anwendung. Es ist die code-nahe,
versionierte Referenz für Header, Grenzwerte und reproduzierbare Negativtests.
Betriebs- und Freigabestatus bleiben in PPM Knowledge.

## Eine Verantwortung je Schutz

| Schicht | Verantwortung | Bewusst nicht verantwortlich |
| --- | --- | --- |
| Cloudflare | öffentliche Kante, DDoS-/Bot-Grundschutz und Weiterleitung von HTTP auf HTTPS | Benutzerkonto-Limits, Anwendungsantworten und eigene HSTS-Abweichungen |
| Traefik + cloudflarewarp | einziger Netzwerkzugang zum Container, TLS-Terminierung, ein bereinigtes `X-Real-IP` und genau eine HSTS-Policy am HAUSV-Router | Kontenlogik, Magic-Link-Limits und zusätzliche Cache-Header |
| HAUSV | explizite Proxy-Allowlist, Source-/Account-Limits, generische Login-Antworten, `Retry-After`, `no-store` sowie HTTP-Servergrenzen | Vertrauen in frei gesetzte Weiterleitungsheader oder eine eigene HSTS-Kopie |

Die Anwendung akzeptiert `X-Real-IP` nur von einer in
`TRUSTED_PROXY_CIDRS` ausdrücklich genannten Gegenstelle. Die kommaseparierte
Liste akzeptiert einzelne IP-Adressen und CIDRs; im Produktivaufbau ist die
stabile Traefik-Adresse als exakte `/32`- beziehungsweise `/128`-Adresse
einzutragen. Das gemeinsame Docker-Netz ist keine Vertrauensgrenze. Fehlt die
Variable bei einer öffentlichen `BASE_URL`, verweigert die Anwendung den
Start. Die sichere Cookie-Policy folgt ausschließlich der konfigurierten
`BASE_URL`, nicht einem weitergeleiteten Proto-Header. Nur bei einer lokalen
`BASE_URL` gilt ohne Konfiguration ein sicherer Loopback-Default
(`127.0.0.1/32,::1/128`).

cloudflarewarp überschreibt am erlaubten Traefik-Peer `X-Real-IP` mit der durch
Cloudflare bestätigten Clientadresse. Bei allen anderen Peers – einschließlich
anderer Container im selben privaten Netz – ignoriert HAUSV
Weiterleitungsheader und verwendet die tatsächliche Gegenstelle.
`X-Forwarded-For` wird für Login-Limits nie ausgewertet.

Die Limits sind derzeit pro Prozess korrekt, weil genau eine HAUSV-Replik
betrieben wird. Vor einer zweiten Replik ist ein gemeinsamer Limiter
einzuführen; mehrere unkoordinierte In-Memory-Limiter wären kein äquivalenter
Schutz.

## Login-Limits und Antworten

Alle Fenster sind feste 15-Minuten-Fenster:

| Vorgang | Quelle | Konto |
| --- | ---: | ---: |
| E-Mail-Anmeldelink anfordern | 20 | 5 |
| OIDC-Anmeldung starten | 20 | – |
| Magic-Link/OIDC abschließen | 60 | 10 |

Quellen und Konten werden im Speicher nur als SHA-256-Schlüssel gehalten.
Ein erschöpftes Limit antwortet immer mit Status `429`, der generischen
Nachricht „Anfrage vorübergehend begrenzt. Bitte später erneut versuchen.“ und
einem ganzzahligen `Retry-After` in Sekunden. Ein Account-Limit wird vor dem
Verbrauchen eines gültigen Einmal-Tokens geprüft.

Die öffentliche E-Mail-Anforderung antwortet für eingeladene, unbekannte,
deaktivierte, nicht für E-Mail freigegebene und vorläufig geschlossene
Dienstleister-Konten gleich: `303` nach `/?sent=1`. Nur für tatsächlich
freigegebene Konten wird im Hintergrund ein Link erstellt und versandt. Der
öffentliche Request wartet dabei nie auf SMTP: Eine feste Versand-Queue hält
höchstens 32 wartende Sendungen, und genau ein Worker stellt jeweils eine
weitere zu. Eine volle oder bereits geschlossene Queue sowie Erzeugungs- und
Versandfehler
bleiben bei derselben generischen Weiterleitung; ein nicht eingereihter Token
wird sofort entwertet. Fehlerlogs enthalten weder Empfänger, Link/Token noch
SMTP-Antworttext. Beim geordneten Prozessende wird die Queue geschlossen,
vollständig abgearbeitet und erst danach die Datenbank geschlossen. Damit
lässt sich aus Status, Ziel, Antworttext oder grober SMTP-Laufzeit nicht
ablesen, ob eine Adresse existiert.

## HTTP- und Browsergrenzen

Die Go-Anwendung setzt:

| Grenze | Wert | Zweck |
| --- | ---: | --- |
| `ReadHeaderTimeout` | 5 Sekunden | langsam gesendete Header abbrechen |
| `ReadTimeout` | 60 Sekunden | vollständige Requests einschließlich begrenzter Uploads deckeln |
| `WriteTimeout` | 120 Sekunden | größere, aber begrenzte Exporte zulassen |
| `IdleTimeout` | 90 Sekunden | untätige Keep-Alive-Verbindungen schließen |
| `MaxHeaderBytes` | 32 KiB konfiguriert | übergroße Request-Header vor dem Handler abweisen |

Go reserviert beim Lesen zusätzlich 4 KiB Parserpuffer. Die wirksame harte
Obergrenze für Request-Zeile plus Header liegt daher bei 36 KiB; alles darüber
wird garantiert mit `431` vor dem Anwendungs-Handler abgewiesen.

Fehlerantworten, `/auth/*`, tenantbezogene Loginseiten und sämtliche geschützte
`/app`-Antworten erhalten `Cache-Control: no-store`. Das gilt ausdrücklich auch
für Inline-Vorschauen und Downloads mit `Content-Disposition`; eine zuvor im
Handler gesetzte private Cache-Policy wird überschrieben.
Nach dem Abmelden ist das Session-Token serverseitig widerrufen. Wird eine
geschützte Ansicht aus dem Browser-Back/Forward-Cache wiederhergestellt, lädt
die Seite einmal neu und durchläuft dadurch die Sessionprüfung.

Am äußeren HAUSV-Router setzt ausschließlich Traefik:

```text
Strict-Transport-Security: max-age=31536000
```

`includeSubDomains` und `preload` bleiben bewusst aus, bis alle bestehenden und
künftigen Subdomains separat dafür freigegeben sind. Cloudflare reicht den Wert
unverändert durch; die Anwendung erzeugt bewusst keinen zweiten Header. Dadurch
erhalten auch vom HAUSV-Router erzeugte Antworten die Policy, selbst wenn sie
den Go-Handler nicht erreichen.

## Reproduzierbare Negativtests

Lokal beziehungsweise in CI:

```fish
go test ./internal/server -run 'Test(Auth|Magic|Security|Logout)'
go test ./cmd/hausv-org -run 'TestHTTPServer'
go test -race ./...
```

Die Tests belegen unter anderem:

- Quell- und Kontenlimit inklusive `Retry-After`;
- gleiche öffentliche Antwort für existierende und unbekannte Konten;
- ein limitierter gültiger Magic Link bleibt unverbraucht;
- gespooftes `X-Forwarded-For`, öffentlich gespooftes `X-Real-IP` und Header
  eines fremden privaten Peers werden nicht vertraut;
- kurze injizierte Read-Header-, Read-, Write- und Idle-Grenzen werden an
  echten TCP-Verbindungen erzwungen;
- Header oberhalb der wirksamen Grenze erreichen den Anwendungs-Handler nicht;
- Auth-/Fehlerseiten sowie `/app`-Inline- und Download-Antworten sind nicht
  speicherbar;
- die Anwendung erzeugt selbst auf TLS- oder Forwarded-Requests kein HSTS;
- ein widerrufenes Session-Token öffnet auch über „Zurück“ keine geschützte
  Ansicht.

Nach jedem Deployment ist die unveränderte äußere Antwort zusätzlich
read-only zu prüfen:

```fish
curl -sS -D - -o /dev/null https://jhw22.hausv.org/ \
  | string match -r -i '^(HTTP/|strict-transport-security:|cache-control:)'

curl -sS -D - -o /dev/null 'https://jhw22.hausv.org/auth/verify?token=ungueltiger-qa-token' \
  | string match -r -i '^(HTTP/|location:|strict-transport-security:|cache-control:)'

curl -sS -D - -o /dev/null https://jhw22.hausv.org/absichtlich-nicht-vorhanden \
  | string match -r -i '^(HTTP/|strict-transport-security:|cache-control:)'
```

Erwartet werden genau eine HSTS-Zeile mit dem oben genannten Wert und
`Cache-Control: no-store` auf Login-, Auth- und Fehlerantworten. Der
read-only-Befund vor dieser Änderung am 30.07.2026 zeigte auf der Live-Kante
weder HSTS noch ein ausdrückliches `no-store`; deshalb ist die
Post-Deployment-Prüfung ein offener Freigabeschritt und darf nicht allein aus
lokalen Tests als erledigt gelten.
