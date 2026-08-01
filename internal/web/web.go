// Package web owns the presentation shell: the HTML templates and the static
// assets they reference.
//
// assets/ lives HERE rather than at the repo root because //go:embed cannot
// reference a parent directory — the embedding package must physically contain
// the files.
package web

import "embed"

//go:embed assets/*
var Assets embed.FS

// FaviconSVG is served at /favicon.svg and /favicon.ico.
const FaviconSVG = `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64">
  <rect width="64" height="64" rx="12" fill="#172019"/>
  <g fill="none" stroke="#e7c574" stroke-width="3.4" stroke-linecap="round" stroke-linejoin="round">
    <path d="M10 43h44"/>
    <path d="M13 43V31l9-7 9 7v12"/>
    <path d="M33 43V31l9-7 9 7v12"/>
    <path d="M24 43V26l8-7 8 7v17"/>
    <path d="M29 43v-8h6v8"/>
    <path d="M18.5 35h4"/>
    <path d="M41.5 35h4"/>
    <path d="M29 29h6"/>
  </g>
</svg>`

// PageTemplates holds every {{define}} block for the app, parsed once at
// startup. Splitting these into real .html files is a separate, deliberate
// change — several tests assert on this string.
const PageTemplates = `
{{define "designTokens"}}
      /* Design tokens: colors, spacing, radius, shadows and typography used by shared components. */
      --ink:#20251f; --muted:#6b6f63; --soft:#9a9485;
      --line:#e7e0d2; --paper:#f7f3ea; --panel:#fffefb; --panel-soft:#fbf8f0;
      --gold:#c8993f; --gold-ink:#8a7b3f; --gold-light:#e7c574; --leaf:#2f6b4a;
      --nav:#172019; --nav-2:#20291f;
      --font-sans: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      --font-serif: ui-serif, Georgia, Cambria, "Times New Roman", serif;
      --space-1:4px; --space-2:8px; --space-3:12px; --space-4:16px; --space-5:20px; --space-6:24px;
      --radius-xs:7px; --radius-sm:8px; --radius-md:10px; --radius-lg:12px; --radius-xl:14px; --radius-pill:999px;
      --shadow-panel:0 12px 30px rgba(32,37,31,.04);
      --shadow-dialog:0 28px 70px rgba(0,0,0,.34);
      --shadow-login:0 28px 70px rgba(0,0,0,.42);
      --surface:var(--panel); --shadow-sm:var(--shadow-panel);
      --shadow-md:0 18px 44px rgba(32,37,31,.08); --shadow-lg:0 24px 62px rgba(0,0,0,.2);
      font-family: var(--font-sans);
{{end}}
{{define "hausvLandingMark"}}
<svg class="hausv-mark" viewBox="0 0 72 42" aria-hidden="true" focusable="false">
  <path d="M9 35h54"/>
  <path d="M11 35V23l8-6 8 6v12"/>
  <path d="M45 35V23l8-6 8 6v12"/>
  <path d="M25 35V17.5L36 9l11 8.5V35"/>
  <path d="M31.5 35v-9h9v9"/>
  <path d="M15.5 27h5"/>
  <path d="M51.5 27h5"/>
  <path d="M31 21h10"/>
</svg>
{{end}}
{{define "hausvPlatformMark"}}
<svg class="hausv-mark hausv-mark-platform" viewBox="0 0 64 48" aria-hidden="true" focusable="false">
  <path class="mark-frame" d="M10 6h44c2.6 0 4.6 2 4.6 4.6v26.8c0 2.6-2 4.6-4.6 4.6H10c-2.6 0-4.6-2-4.6-4.6V10.6C5.4 8 7.4 6 10 6z"/>
  <path d="M13 34h38"/>
  <path d="M14.5 34v-9l7-5.5 7 5.5v9"/>
  <path d="M35.5 34v-9l7-5.5 7 5.5v9"/>
  <path d="M25 34V20.5L32 15l7 5.5V34"/>
  <path d="M29 34v-7h6v7"/>
  <path d="M18 28h3.5"/>
  <path d="M42.5 28H46"/>
  <path d="M29.5 23.5h5"/>
</svg>
{{end}}
{{define "tenantBrandMark"}}
{{if eq .Tenant.BrandIcon "single-home"}}
<svg class="hausv-mark tenant-brand-mark" viewBox="0 0 64 48" aria-hidden="true" focusable="false">
  <path d="M12 35h40"/>
  <path d="M16 35V22.5L32 11l16 11.5V35"/>
  <path d="M26.5 35v-9h11v9"/>
  <path d="M21.5 27.5h5M37.5 27.5h5"/>
</svg>
{{else if eq .Tenant.BrandIcon "multi-tenant"}}
<svg class="hausv-mark tenant-brand-mark" viewBox="0 0 64 48" aria-hidden="true" focusable="false">
  <path d="M13 36h38"/>
  <path d="M17 36V18h30v18"/>
  <path d="M23 36v-7h6v7M35 36v-7h6v7"/>
  <path d="M22 23h5M37 23h5M22 28h5M37 28h5"/>
  <path d="M19 18l13-8 13 8"/>
</svg>
{{else if eq .Tenant.BrandIcon "mixed-use"}}
<svg class="hausv-mark tenant-brand-mark" viewBox="0 0 64 48" aria-hidden="true" focusable="false">
  <path d="M14 36h36"/>
  <path d="M18 36V17h28v19"/>
  <path d="M18 24h28"/>
  <path d="M22 36v-7h8v7M35 36v-7h7v7"/>
  <path d="M22 21h5M36 21h5"/>
  <path d="M16 17l16-7 16 7"/>
</svg>
{{else if eq .Tenant.BrandIcon "address-plaque"}}
<svg class="hausv-mark tenant-brand-mark" viewBox="0 0 64 48" aria-hidden="true" focusable="false">
  <path d="M15 12h34a4 4 0 0 1 4 4v20a4 4 0 0 1-4 4H15a4 4 0 0 1-4-4V16a4 4 0 0 1 4-4z"/>
  <path d="M20 21h24M20 28h18"/>
  <path d="M46 28h.01"/>
</svg>
{{else if eq .Tenant.BrandIcon "parking"}}
<svg class="hausv-mark tenant-brand-mark" viewBox="0 0 64 48" aria-hidden="true" focusable="false">
  <path d="M17 36h30"/>
  <path d="M20 36l2.5-12h19L44 36"/>
  <path d="M23 36v4M41 36v4"/>
  <path d="M24 29h16"/>
  <path d="M28 20h8a5 5 0 0 1 0 10h-8V16"/>
</svg>
{{else}}
<svg class="hausv-mark tenant-brand-mark" viewBox="0 0 64 48" aria-hidden="true" focusable="false">
  <path d="M13 34h38"/>
  <path d="M14.5 34v-9l7-5.5 7 5.5v9"/>
  <path d="M35.5 34v-9l7-5.5 7 5.5v9"/>
  <path d="M25 34V20.5L32 15l7 5.5V34"/>
  <path d="M29 34v-7h6v7"/>
  <path d="M18 28h3.5M42.5 28H46"/>
</svg>
{{end}}
{{end}}
{{define "home"}}
<!doctype html>
<html lang="de">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  <link rel="icon" type="image/svg+xml" href="/favicon.svg">
  <link rel="shortcut icon" href="/favicon.svg">
  <script src="/assets/home.js?v={{.AssetVersion}}" defer></script>
  <style>
    :root {
      color-scheme: light;
{{template "designTokens" .}}
    }
    * { box-sizing: border-box; }
    body { margin: 0; color: var(--ink); background: var(--paper); }
    /* Accessibility convention: all keyboard-reachable controls keep a visible focus ring. */
    :where(a, button, input, select, textarea, summary, [tabindex]):focus-visible { outline: 3px solid var(--gold-light); outline-offset: 3px; }
    .hero { position: relative; isolation: isolate; min-height: 100vh; overflow: hidden; display: grid; grid-template-rows: auto 1fr auto; background: #10160f; }
    .hero::before {
      content: ""; position: absolute; inset: -16px;
      background: url('{{.Tenant.HeroImageURL}}') center 42% / cover no-repeat;
      filter: blur(2px) brightness(.78) saturate(.95); transform: scale(1.04); z-index: -2;
    }
    .hero::after {
      content: ""; position: absolute; inset: 0;
      background: rgba(12,18,13,.48);
      z-index: -1;
    }
    header { position: relative; z-index: 1; display: flex; justify-content: space-between; align-items: center; gap: 24px; padding: 22px clamp(20px,5vw,72px); color: var(--ink); background: rgba(255,254,251,.97); border-bottom: 1px solid rgba(231,224,210,.78); }
    .brand { display: inline-flex; align-items: center; gap: 13px; text-decoration: none; color: var(--ink); }
    .mark { width: 62px; height: 46px; display: grid; place-items: center; color: var(--gold); }
    .mark .hausv-mark { width: 62px; height: 46px; display: block; stroke: currentColor; stroke-width: 2.15; fill: none; stroke-linecap: round; stroke-linejoin: round; }
    .brand .name { font-family: var(--font-serif); font-weight: 600; font-size: 18px; }
    nav { display: flex; gap: 24px; color: var(--ink); font-size: 14px; font-weight: 650; }
    nav a { color: inherit; text-decoration: none; padding-bottom: 4px; border-bottom: 1px solid rgba(231,197,116,.65); }
    nav a:hover { border-bottom-color: var(--gold); }
    main { display: grid; grid-template-columns: minmax(0,1.1fr) minmax(320px,420px); gap: clamp(28px,6vw,64px); align-items: center; padding: clamp(34px,6vh,58px) clamp(20px,5vw,72px); }
    .copy { max-width: 760px; color: #fff; }
    .eyebrow { font-size: 12px; font-weight: 700; text-transform: uppercase; letter-spacing: .2em; color: var(--gold-light); margin-bottom: 18px; }
    h1 { margin: 0; font-family: var(--font-serif); font-weight: 500; font-size: clamp(46px,7vw,72px); line-height: 1.0; letter-spacing: -.01em; text-shadow: 0 2px 30px rgba(0,0,0,.3); }
    .lead { max-width: 560px; margin: 24px 0 0; font-size: clamp(17px,2vw,19px); line-height: 1.55; color: rgba(255,255,255,.92); }
    .meta { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 18px; margin-top: 32px; max-width: 650px; }
    .meta div { border-left: 1px solid rgba(231,197,116,.58); padding-left: 15px; min-width: 0; }
    .meta strong { display: block; font-weight: 760; font-size: 14px; color: #fff; margin-bottom: 4px; }
    .meta span { color: rgba(255,255,255,.8); font-size: 12.5px; line-height: 1.4; }
    .side-stack { width: min(100%,420px); justify-self: end; display: grid; gap: 16px; }
    .login, .location-card { background: var(--paper); border-radius: var(--radius-xl); box-shadow: var(--shadow-login); }
    .login { padding: 28px; }
    .login h2 { margin: 0; font-family: var(--font-serif); font-weight: 600; font-size: 28px; line-height: 1.08; }
    .login p { color: var(--muted); line-height: 1.5; margin: 9px 0 18px; font-size: 14.5px; }
    .login-email { margin-top: 12px; border-top: 1px solid var(--line); padding-top: 12px; }
    .login-email summary { min-height: 42px; display: flex; align-items: center; cursor: pointer; color: var(--ink); font-size: 13.5px; font-weight: 750; }
    .login-email summary::marker { color: var(--gold); }
    .login-email form { margin-top: 14px; }
    label { display: block; font-size: 11.5px; font-weight: 700; text-transform: uppercase; letter-spacing: .07em; color: var(--gold-ink); margin-bottom: 8px; }
    input { width: 100%; border: 1px solid #e2dac9; border-radius: 10px; padding: 14px 15px; font: inherit; background: #fffefb; color: var(--ink); }
    button { width: 100%; border: 0; border-radius: 10px; padding: 15px 16px; margin-top: 13px; font: inherit; font-weight: 700; color: #fff; background: var(--ink); cursor: pointer; }
    button:hover { background: #000; }
    .notice { border: 1px solid rgba(32,37,31,.16); background: rgba(200,153,63,.1); color: #6a5320; border-radius: 10px; padding: 12px 14px; font-size: 14px; line-height: 1.4; margin-bottom: 16px; }
    .notice.warn { border-color: rgba(173,92,27,.22); background: rgba(231,197,116,.2); color: #6c491a; }
    .dev-link { min-height: 48px; display: flex; align-items: center; justify-content: center; border: 1px solid var(--ink); background: var(--ink); color: #fff; border-radius: 10px; padding: 12px 14px; margin: 0 0 12px; text-align: center; text-decoration: none; font-size: 14px; font-weight: 760; }
    .dev-link:hover { background: #000; }
    .login-retry { margin-top: 10px; border-top: 1px solid var(--line); }
    .login-retry summary { min-height: 44px; display: flex; align-items: center; cursor: pointer; color: var(--muted); font-size: 13px; font-weight: 720; }
    .login-retry summary::marker { color: var(--gold); }
    .login-retry form { margin-top: 8px; }
    .login-retry .foot-note { margin-bottom: 0; }
    .sso-button { display: flex; align-items: center; justify-content: center; min-height: 48px; border-radius: 10px; background: var(--ink); color: #fff; text-decoration: none; font-weight: 700; margin-bottom: 14px; }
    .sso-button:hover { background: #000; }
    .foot-note { margin: 16px 0 0; font-size: 13px; line-height: 1.4; color: var(--soft); }
    .location-card { overflow: hidden; }
    .location-head { display: flex; align-items: center; gap: 9px; padding: 14px 17px; color: var(--ink); font-size: 13.5px; font-weight: 760; }
    .location-head svg { width: 18px; height: 18px; flex: 0 0 auto; stroke: var(--leaf); stroke-width: 2; fill: none; }
    .location-map { position: relative; height: 92px; overflow: hidden; isolation: isolate; background: #e8ebe1; }
    .location-map-fallback { position: absolute; inset: 0; display: grid; place-items: end start; padding: 10px; background: linear-gradient(135deg,#e5eadf,#f2eee4); color: var(--muted); font-size: 11.5px; font-weight: 700; }
    .location-map-configured.map-tile-failed .location-map-fallback { z-index: 3; }
    .location-map-configured.map-tile-failed .location-map-tiles, .location-map-configured.map-tile-failed .location-pin { display: none; }
    .location-map-fallback span { padding: 7px 10px; border-radius: 999px; background: rgba(255,254,251,.86); box-shadow: 0 1px 8px rgba(32,37,31,.08); }
    .location-map-tiles { position: absolute; inset: 0; z-index: 1; pointer-events: none; filter: saturate(.72) contrast(.94) brightness(1.02); }
    .location-map-tile { position: absolute; width: 256px; height: 256px; display: block; background-position: center; background-repeat: no-repeat; background-size: 256px 256px; user-select: none; pointer-events: none; }
    .location-pin { position: absolute; z-index: 2; left: 50%; top: 50%; width: 19px; height: 19px; border: 1px solid rgba(255,255,255,.72); border-radius: 50% 50% 50% 0; background: var(--nav); transform: translate(-50%,-50%) rotate(-45deg); box-shadow: 0 3px 9px rgba(0,0,0,.28); }
    .location-pin::after { content: ""; position: absolute; width: 7px; height: 7px; left: 6px; top: 6px; border-radius: 50%; background: var(--paper); }
    .location-foot { min-height: 44px; display: flex; align-items: stretch; justify-content: space-between; padding-inline: 11px 8px; }
    .location-link, .location-credit { min-height: 44px; display: flex; align-items: center; color: var(--ink); text-underline-offset: 3px; }
    .location-link { padding: 8px 6px; font-size: 12.5px; font-weight: 700; }
    .location-credit { padding: 8px; color: var(--soft); font-size: 10.5px; font-weight: 650; }
    footer { position: relative; z-index: 1; display: flex; align-items: center; justify-content: space-between; gap: 18px; padding: 18px clamp(20px,5vw,72px); color: var(--muted); background: rgba(255,254,251,.97); border-top: 1px solid rgba(231,224,210,.78); font-weight: 550; font-size: 13px; }
    .footer-links { display: flex; align-items: center; gap: 16px; }
    footer a { color: var(--ink); text-underline-offset: 3px; }
    footer .version { color: var(--soft); font-size: 11px; }
    @media (max-width: 860px) {
      header { padding-block: 14px; }
      nav { display: none; }
      main { grid-template-columns: 1fr; align-items: start; gap: 24px; padding-block: 28px; }
      .copy { max-width: 560px; }
      .eyebrow { margin-bottom: 12px; }
      .lead { margin-top: 16px; }
      .meta { grid-template-columns: repeat(3,minmax(0,1fr)); gap: 8px; margin-top: 20px; }
      .meta div { padding-left: 9px; }
      .meta span { display: none; }
      h1 { font-size: clamp(38px,11vw,52px); }
      .side-stack { justify-self: start; }
      footer { align-items: flex-start; }
    }
    @media (max-width: 520px) {
      .brand .name { font-size: 16px; }
      .mark { width: 52px; }
      .mark .hausv-mark { width: 52px; }
      main { gap: 18px; padding-block: 22px; }
      .eyebrow { margin-bottom: 9px; }
      h1 { font-size: 34px; line-height: 1.02; }
      .lead { margin-top: 12px; font-size: 15.5px; line-height: 1.48; }
      .meta { display: none; }
      .login { padding: 20px; }
      .auth-sent .side-stack { grid-row: 1; }
      .auth-sent .copy { grid-row: 2; }
      footer { display: grid; }
      .footer-links { gap: 13px; flex-wrap: wrap; }
    }
  </style>
</head>
<body{{if .Sent}} class="auth-sent"{{else if .Expired}} class="auth-expired"{{else if .Denied}} class="auth-denied"{{end}}>
  <section class="hero">
    <header>
      <a class="brand" href="/" aria-label="Hausportal {{.HouseName}}"><span class="mark">{{template "tenantBrandMark" .}}</span><span class="name">{{.HouseName}}</span></a>
      <nav aria-label="Seitennavigation">
        <a href="#login">Anmelden</a>
      </nav>
    </header>
    <main>
      <div class="copy">
        <div class="eyebrow">{{.HomeCopy.Eyebrow}}</div>
        <h1>{{.HomeCopy.Headline}}</h1>
        <p class="lead">{{.HomeCopy.Lead}}</p>
        <div class="meta" aria-label="Portalüberblick">
          <div><strong>Informiert</strong><span>{{.HomeCopy.FirstDetail}}</span></div>
          <div><strong>Organisiert</strong><span>{{.HomeCopy.SecondDetail}}</span></div>
          <div><strong>Privat</strong><span>{{.HomeCopy.PrivacyDetail}}</span></div>
        </div>
      </div>
      <div class="side-stack">
        <section id="login" class="login" aria-label="Anmeldung">
          <h2>{{if .Sent}}E-Mail prüfen{{else if .Expired}}Neuen Link anfordern{{else if .Denied}}Zugang prüfen{{else}}Anmelden{{end}}</h2>
          {{if not .Sent}}<p>{{if .Expired}}Der bisherige Link ist abgelaufen.{{else if .Denied}}Für diese Adresse besteht noch kein Zugang.{{else if .OIDCConfigured}}Sicher und ohne eigenes Passwort.{{else}}Sicher per E-Mail-Link – ohne Passwort.{{end}}</p>{{end}}
          {{if and .OIDCConfigured (not .Sent)}}<a class="sso-button" href="/auth/oidc/start">Anmelden</a>{{end}}
          {{if .Sent}}
            <div class="notice" role="status">Wenn die Adresse eingeladen ist, ist der Anmeldelink unterwegs. Er gilt 15 Minuten.</div>
            {{if .DevLoginLink}}<a class="dev-link" href="{{.DevLoginLink}}">Weiter zum Portal</a>{{end}}
          {{end}}
          {{if .Denied}}<div class="notice warn" role="status">Bitte wenden Sie sich an Ihre Hausverwaltung.</div>{{end}}
          {{if and .EmailLoginAvailable .OIDCConfigured (not .Sent)}}<details class="login-email" {{if or .Denied .Expired}}open{{end}}><summary>Anmeldelink per E-Mail erhalten</summary>{{end}}
          {{if and .EmailLoginAvailable (not .Sent)}}
            <form method="post" action="/auth/request">
              <label for="email">E-Mail-Adresse</label>
              <input id="email" name="email" type="email" inputmode="email" autocomplete="email" required placeholder="name@example.com">
              <button type="submit">Anmeldelink senden</button>
            </form>
            <p class="foot-note">15 Minuten gültig · nur für eingeladene Personen</p>
          {{else if and (not .Sent) (not .OIDCConfigured)}}
            <div class="notice">Die Anmeldung ist gerade nicht verfügbar.</div>
          {{end}}
          {{if and .EmailLoginAvailable .OIDCConfigured (not .Sent)}}</details>{{end}}
          {{if and .Sent .EmailLoginAvailable}}
            <details class="login-retry">
              <summary>Andere Adresse verwenden</summary>
              <form method="post" action="/auth/request">
                <label for="email-retry">E-Mail-Adresse</label>
                <input id="email-retry" name="email" type="email" inputmode="email" autocomplete="email" required placeholder="name@example.com">
                <button type="submit">Neuen Link senden</button>
              </form>
              <p class="foot-note">Nur für eingeladene Personen</p>
            </details>
          {{end}}
        </section>
        <section class="location-card" aria-label="Hausstandort">
          <div class="location-head"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M20 10c0 5-8 11-8 11S4 15 4 10a8 8 0 1 1 16 0z"/><circle cx="12" cy="10" r="2.5"/></svg><span>{{.Tenant.Address}}</span></div>
          <div class="location-map{{if .LocationMap.Configured}} location-map-configured{{end}}" role="img" aria-label="Fester Kartenausschnitt rund um {{.Tenant.Address}}">
            <div class="location-map-fallback"><span>Karte nicht verfügbar</span></div>
            {{if .LocationMap.Configured}}
            <div class="location-map-tiles" aria-hidden="true">
              {{range .LocationMap.Tiles}}<span class="location-map-tile" data-map-tile="{{.URL}}" style="{{.Style}};background-image:url('{{.URL}}')"></span>{{end}}
            </div>
            <span class="location-pin" aria-hidden="true"></span>
            {{end}}
          </div>
          <div class="location-foot"><a class="location-link" href="{{.MapURL}}" rel="noopener noreferrer" target="_blank" aria-label="{{.Tenant.Address}} in OpenStreetMap öffnen">Karte öffnen</a><a class="location-credit" href="https://www.openstreetmap.org/copyright" rel="noopener noreferrer" target="_blank">© OpenStreetMap</a></div>
        </section>
      </div>
    </main>
    <footer>
      <span>{{.Tenant.Address}} · Privat für eingeladene Personen</span>
      <span class="footer-links"><a href="/datenschutz">Datenschutz</a><a href="https://hausv.org/#impressum">Impressum</a><span class="version">{{.AppVersion}}</span></span>
    </footer>
  </section>
</body>
</html>
{{end}}

{{define "landing"}}
<!doctype html>
<html lang="de">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  <meta name="description" content="Sicheres Kommunikations- und Transparenzportal für WEGs, Wohnungen und Mehrparteienhäuser. Aushänge, Termine, Dokumente, Anliegen, Abstimmungen und Schnittstellen ohne eigene Buchhaltung.">
  <link rel="icon" type="image/svg+xml" href="/favicon.svg">
  <link rel="shortcut icon" href="/favicon.svg">
  <script src="/assets/landing.js?v={{.AssetVersion}}" defer></script>
  <!-- Rotating 3D brand mark. ESM (module scripts defer by default); it mounts
       only when WebGL is present, so the inline SVG below stays the fallback. -->
  <script type="module" src="/assets/hausv-mark-3d.js?v={{.AssetVersion}}"></script>
  <style>
    :root {
      color-scheme: light;
{{template "designTokens" .}}
    }
    * { box-sizing: border-box; }
    html { max-width: 100%; overflow-x: clip; scroll-behavior: smooth; }
    body { max-width: 100%; overflow-x: clip; margin: 0; color: var(--ink); background: var(--panel); font-family: var(--font-sans); }
    /* The header is fixed chrome, so an anchor jump would otherwise park the
       section headline underneath it. */
    section[id] { scroll-margin-top: 92px; }
    a { color: inherit; }
    :where(a, button):focus-visible { outline: 3px solid var(--gold-light); outline-offset: 3px; }
    /* The nav is fixed chrome now, so the hero no longer reserves a row for
       it — only padding, to keep the copy clear of the bar. */
    /* Full height, because the mark now shares the hero with the copy: at
       86svh there was no band above the headline for it to occupy, and it
       landed on the eyebrow. */
    .landing-hero { position: relative; min-height: 100svh; display: grid; grid-template-rows: minmax(0,1fr); padding-top: 90px; overflow: hidden; color: #fff; background: #162018; }
    .landing-hero::before { content: ""; position: absolute; inset: 0; background: url('{{.LandingHeroURL}}') center 48% / cover no-repeat; transform: scale(1.01); }
    .landing-hero::after { content: ""; position: absolute; inset: 0; background: linear-gradient(90deg, rgba(12,18,13,.86) 0%, rgba(12,18,13,.74) 34%, rgba(12,18,13,.32) 66%, rgba(12,18,13,.12) 100%); }
    .landing-nav, .landing-copy { position: relative; z-index: 1; width: min(1180px,100%); margin: 0 auto; padding-left: clamp(20px,4vw,42px); padding-right: clamp(20px,4vw,42px); }
    .landing-nav { display: flex; justify-content: space-between; align-items: center; gap: 18px; padding-top: 26px; padding-bottom: 20px; }
    .landing-brand { display: inline-flex; align-items: center; text-decoration: none; color: #fff; font-weight: 800; }
    .landing-mark { position: relative; width: 72px; height: 44px; display: grid; place-items: center; color: var(--gold-light); }
    /* Hidden on desktop, where the 3D mark is the logo; shown below 900px,
       where the 3D mark is not mounted at all and the bar would be empty. */
    .landing-mark .hausv-mark { width: 70px; height: 42px; display: none; stroke: currentColor; stroke-width: 2.2; fill: none; stroke-linecap: round; stroke-linejoin: round; }
    .landing-menu-toggle { display: none; }
    /* 7x the 72x44 logo slot. JS pins left/top onto the real logo position and
       drives transform/opacity from scroll; transform-origin must stay top-left
       so the shrink lands exactly on that slot. */
    /* Sticky chrome, layered above the page. */
    .landing-navbar { position: fixed; z-index: 5; top: 0; left: 0; right: 0; }
    .mark3d-veil { position: fixed; z-index: 2; top: 0; left: 0; right: 0; height: 84px; opacity: 0; pointer-events: none; background: linear-gradient(180deg, rgba(12,18,13,.82) 0%, rgba(12,18,13,.66) 34%, rgba(12,18,13,.34) 66%, rgba(12,18,13,.12) 85%, rgba(12,18,13,0) 100%); }
    /* 14x the 72x44 logo slot, drawn at that size and scaled down so it stays
       crisp. The start scale is capped by the space above the copy, so this is
       the ceiling rather than a fixed size. */
    .mark3d-stage { position: fixed; z-index: 3; width: 1008px; height: 616px; transform-origin: 0 0; pointer-events: none; will-change: transform, opacity, filter; }
    .mark3d-stage canvas, .mark3d-flat { transition: opacity .45s ease; }
    /* Parked, the glass is dropped for a flat white mark: at 72px the depth
       reads as noise, and solid white stays legible on every section. */
    .mark3d-flat { position: absolute; inset: 0; width: 100%; height: 100%; opacity: 0; fill: none; stroke: #fff; stroke-width: 2.2; stroke-linecap: round; stroke-linejoin: round; }
    .mark3d-stage[data-mark3d-frozen="true"] .mark3d-flat { opacity: 1; }
    .mark3d-stage[data-mark3d-frozen="true"] canvas { opacity: 0; }
    .mark3d-top { position: fixed; z-index: 6; width: 72px; height: 44px; padding: 0; border: 0; background: none; cursor: pointer; }
    .mark3d-top[hidden] { display: none; }
    @media (max-width: 900px) { .mark3d-veil, .mark3d-stage, .mark3d-top { display: none; } }
    .landing-links { display: flex; align-items: center; gap: 20px; font-size: 14px; font-weight: 700; }
    .landing-links a { text-decoration: none; color: rgba(255,255,255,.88); }
    .landing-links a:hover { color: #fff; }
    .landing-copy { align-self: end; padding-top: 48px; padding-bottom: clamp(54px,8vh,88px); }
    .landing-eyebrow { margin-bottom: 18px; color: var(--gold-light); font-size: 12px; font-weight: 900; letter-spacing: .18em; text-transform: uppercase; }
    .landing-copy h1 { max-width: 850px; margin: 0; font-family: var(--font-serif); font-weight: 500; font-size: clamp(48px,7.4vw,86px); line-height: .98; text-wrap: balance; }
    .landing-lead { max-width: 690px; margin: 24px 0 0; color: rgba(255,255,255,.9); font-size: clamp(18px,2vw,22px); line-height: 1.48; }
    .landing-actions { display: flex; flex-wrap: wrap; gap: 12px; margin-top: 32px; }
    .landing-button { min-height: 48px; display: inline-flex; align-items: center; justify-content: center; border-radius: 8px; padding: 12px 18px; text-decoration: none; font-weight: 900; }
    .landing-button.primary { background: #fff; color: var(--ink); }
    .landing-button.secondary { border: 1px solid rgba(255,255,255,.48); color: #fff; background: rgba(255,255,255,.08); backdrop-filter: blur(8px); }
    .landing-access { display: flex; align-items: center; gap: 9px; margin: 26px 0 0; color: rgba(255,255,255,.82); font-size: 14px; font-weight: 750; }
    .landing-access svg { width: 18px; height: 18px; flex: 0 0 auto; stroke: var(--gold-light); stroke-width: 1.9; fill: none; }
    /* ---- Section rhythm -------------------------------------------------
       Every section is built the same way: a head (kicker + headline on the
       left, lead bottom-aligned on the right) followed by full-width content
       rows. The single grid gap on .section-inner is what keeps the vertical
       rhythm identical from section to section, and the two-column head keeps
       the measure short without leaving an empty gutter beside it. */
    .section { padding: clamp(50px,5.6vw,80px) clamp(20px,4vw,42px); }
    .section-inner { position: relative; z-index: 1; width: min(1180px,100%); margin: 0 auto; display: grid; gap: clamp(22px,2.3vw,30px); }
    .section-head { display: grid; grid-template-columns: minmax(0,1fr) minmax(0,400px); gap: clamp(18px,3.2vw,56px); align-items: end; }
    .section h2 { margin: 0; max-width: 20ch; font-family: var(--font-serif); font-weight: 500; font-size: clamp(34px,4.2vw,54px); line-height: 1.04; text-wrap: balance; }
    .section-kicker { display: flex; align-items: center; gap: 10px; margin: 0 0 14px; color: var(--gold-ink); font-size: 12px; font-weight: 900; letter-spacing: .14em; text-transform: uppercase; }
    .section-kicker::before { content: ""; width: 26px; height: 2px; flex: 0 0 auto; background: var(--gold); }
    .section-lead { margin: 0; color: var(--muted); font-size: 17px; line-height: 1.6; text-wrap: pretty; }
    /* Alternating surfaces carry the rhythm; no ghosted photography behind
       content, so the cards keep full contrast. */
    .features-section, .cost-section { background: var(--paper); border-top: 1px solid var(--line); border-bottom: 1px solid var(--line); }
    .trust-section, .imprint-section { background: var(--panel); }

    /* ---- Panels ---------------------------------------------------------
       One panel language for the whole page: hairline border, 12px radius,
       hairline dividers instead of gaps, so every row shares an edge. */
    .feature-grid, .product-state, .price-panel { border: 1px solid var(--line); border-radius: var(--radius-lg); background: var(--panel); box-shadow: var(--shadow-panel); overflow: hidden; }
    .feature-grid { display: grid; grid-template-columns: repeat(5,minmax(0,1fr)); }
    /* auto/auto/1fr with a reserved two-line title row: whatever the headline
       length, every description starts on the same baseline. */
    .feature { min-width: 0; display: grid; grid-template-rows: auto auto 1fr; align-content: start; padding: 24px 20px 26px; border-right: 1px solid var(--line); }
    .feature:last-child { border-right: 0; }
    .feature-icon { width: 40px; height: 40px; margin-bottom: 18px; display: grid; place-items: center; border-radius: var(--radius-md); background: rgba(200,153,63,.1); color: var(--gold-ink); }
    .feature-icon svg { width: 22px; height: 22px; display: block; stroke: currentColor; stroke-width: 1.9; fill: none; stroke-linecap: round; stroke-linejoin: round; }
    .feature strong { display: block; min-height: 2.3em; margin-bottom: 9px; font-family: var(--font-serif); font-weight: 600; font-size: 21px; line-height: 1.15; }
    .feature p { margin: 0; color: var(--muted); font-size: 14px; line-height: 1.5; text-wrap: pretty; }
    /* Subgrid keeps the eyebrow, the headline and the body of both halves on
       the same three baselines, whatever the headline wraps to. */
    .product-state { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); grid-template-rows: auto auto auto; }
    .product-state > div { grid-row: span 3; display: grid; grid-template-rows: subgrid; align-content: start; padding: 24px 26px 26px; }
    .product-state > div + div { border-left: 1px solid var(--line); }
    .product-state span { margin-bottom: 11px; font-size: 11px; font-weight: 900; letter-spacing: .12em; text-transform: uppercase; }
    .product-state > div:first-child span { color: var(--leaf); }
    .product-state > div:last-child span { color: var(--gold-ink); }
    .product-state strong { font-family: var(--font-serif); font-weight: 600; font-size: 24px; line-height: 1.15; }
    .product-state p { margin: 9px 0 0; max-width: 42ch; color: var(--muted); line-height: 1.55; text-wrap: pretty; }

    /* ---- Disclosures ----------------------------------------------------- */
    .landing-more, .legal-details { border-top: 1px solid var(--line); border-bottom: 1px solid var(--line); }
    .landing-more summary, .legal-details summary { min-height: 56px; display: flex; align-items: center; justify-content: space-between; gap: 16px; cursor: pointer; color: var(--ink); font-weight: 850; list-style: none; }
    .landing-more summary::-webkit-details-marker, .legal-details summary::-webkit-details-marker { display: none; }
    .landing-more summary::after, .legal-details summary::after { content: "+"; color: var(--gold-ink); font-size: 24px; font-weight: 500; }
    .landing-more[open] summary::after, .legal-details[open] summary::after { content: "−"; }
    .landing-more-grid { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: clamp(20px,3vw,48px); padding: 2px 0 26px; }
    .landing-more-grid h3 { margin: 0 0 10px; font-family: var(--font-serif); font-weight: 600; font-size: 21px; }
    .landing-more-grid ul { margin: 0; padding-left: 20px; color: var(--muted); line-height: 1.7; }
    .landing-more-grid small { color: var(--gold-ink); font-weight: 800; }

    /* ---- Trust ----------------------------------------------------------
       Four principles across, hairline above, no boxes: a lighter texture than
       the panels so two consecutive sections do not read as the same object. */
    .trust-summary { display: grid; grid-template-columns: repeat(4,minmax(0,1fr)); gap: 0 clamp(24px,3vw,48px); border-top: 1px solid var(--line); }
    .trust-line { display: grid; align-content: start; padding-top: 26px; }
    .trust-number { width: 30px; height: 30px; margin-bottom: 16px; display: grid; place-items: center; border-radius: 50%; background: rgba(200,153,63,.12); color: var(--gold-ink); font-size: 11px; font-weight: 900; letter-spacing: .06em; }
    .trust-line strong { font-size: 17px; line-height: 1.3; }
    .trust-line p { margin: 7px 0 0; color: var(--muted); font-size: 15px; line-height: 1.55; text-wrap: pretty; }

    /* ---- Price -----------------------------------------------------------
       The two positions and the three worked examples live inside one panel:
       the examples are a labelled footer band, not a second floating strip
       that has to line up with the columns above it. */
    .price-panel { border-color: rgba(138,123,63,.24); box-shadow: var(--shadow-md); }
    .price-summary { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); grid-template-rows: auto auto auto; }
    .price-summary > div { grid-row: span 3; display: grid; grid-template-rows: subgrid; align-content: start; padding: 26px 28px 28px; }
    .price-summary > div + div { border-left: 1px solid var(--line); }
    .price-summary span { margin-bottom: 12px; color: var(--gold-ink); font-size: 11px; font-weight: 900; letter-spacing: .12em; text-transform: uppercase; }
    .price-summary > div:first-child span { color: var(--leaf); }
    .price-summary strong { font-family: var(--font-serif); font-weight: 600; font-size: clamp(23px,1.9vw,27px); line-height: 1.14; text-wrap: balance; }
    .price-summary p { margin: 11px 0 0; color: var(--muted); line-height: 1.55; text-wrap: pretty; }
    /* Label plus three examples in four equal columns: the middle divider then
       lands exactly under the divider of the two positions above it. */
    .price-examples { display: grid; grid-template-columns: repeat(4,minmax(0,1fr)); border-top: 1px solid var(--line); background: var(--panel-soft); }
    .price-examples-label { margin: 0; display: grid; align-content: center; padding: 20px 22px 22px 28px; border-right: 1px solid var(--line); color: var(--gold-ink); font-size: 11px; font-weight: 900; letter-spacing: .12em; line-height: 1.5; text-transform: uppercase; }
    .price-example { display: grid; align-content: start; gap: 5px; padding: 20px 20px 22px; }
    .price-example + .price-example { border-left: 1px solid var(--line); }
    .price-example:last-child { padding-right: 28px; }
    .price-example b { font-family: var(--font-serif); font-weight: 600; font-size: 26px; line-height: 1; color: var(--gold-ink); }
    .price-example span { color: var(--muted); font-size: 13px; line-height: 1.45; text-wrap: balance; }
    .price-footnote { margin: 0; max-width: 78ch; color: var(--muted); font-size: 14px; line-height: 1.6; text-wrap: pretty; }

    /* ---- Imprint and closing --------------------------------------------- */
    .imprint-grid { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 14px; }
    .imprint-card { border: 1px solid var(--line); border-radius: var(--radius-md); padding: 18px 20px; background: var(--panel-soft); }
    .imprint-card strong { display: block; font-size: 15px; }
    .imprint-card p { margin: 8px 0 0; color: var(--muted); line-height: 1.55; }
    .legal-grid { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 14px; padding: 2px 0 16px; }
    .legal-details .mini { margin: 0 0 22px; max-width: 88ch; color: var(--soft); font-size: 13px; line-height: 1.6; }
    .landing-contact { display: flex; align-items: center; justify-content: space-between; gap: 24px; padding: 26px 28px; border-radius: var(--radius-lg); background: var(--nav); color: #fff; }
    .landing-contact strong { display: block; font-family: var(--font-serif); font-weight: 600; font-size: 27px; line-height: 1.15; }
    .landing-contact p { margin: 7px 0 0; color: rgba(255,255,255,.74); }
    .landing-contact .landing-button { flex: 0 0 auto; background: #fff; color: var(--ink); }
    footer { padding: 26px clamp(20px,4vw,42px); color: var(--muted); background: var(--paper); border-top: 1px solid var(--line); }
    footer div { width: min(1180px,100%); margin: 0 auto; display: flex; justify-content: space-between; gap: 16px; flex-wrap: wrap; font-size: 14px; }

    /* ---- Responsive -------------------------------------------------------
       1100: the five-across strip gets too narrow to read, so it becomes a
       two-across list. 900: the nav collapses. 640: everything stacks. */
    @media (max-width: 1100px) {
      .section-head { grid-template-columns: minmax(0,1fr) minmax(0,320px); }
      .feature-grid { grid-template-columns: repeat(2,minmax(0,1fr)); }
      .feature { grid-template-columns: 40px minmax(0,1fr); grid-template-rows: auto auto; column-gap: 16px; padding: 22px 24px; border-bottom: 1px solid var(--line); }
      .feature:nth-child(2n), .feature:last-child { border-right: 0; }
      .feature-icon { grid-row: 1 / span 2; margin-bottom: 0; }
      .feature strong { grid-column: 2; min-height: 0; margin-bottom: 6px; }
      .feature p { grid-column: 2; }
      /* Five cards in two columns leave the fifth alone on its row: it spans
         and turns into one line, so the row is filled rather than half empty. */
      .feature:last-child { grid-column: 1 / -1; grid-template-columns: 40px auto minmax(0,1fr); align-items: center; border-bottom: 0; }
      .feature:last-child .feature-icon { grid-row: 1; }
      .feature:last-child strong { grid-row: 1; margin-bottom: 0; }
      .feature:last-child p { grid-column: 3; grid-row: 1; }
      .trust-summary { grid-template-columns: repeat(2,minmax(0,1fr)); }
      .trust-line { padding-top: 24px; }
      .trust-line:nth-child(-n+2) { padding-bottom: 24px; border-bottom: 1px solid var(--line); }
    }
    @media (max-width: 900px) {
      .landing-links { display: none; }
      .landing-mark .hausv-mark { display: block; }
      /* The scroll-driven veil is desktop-only, so below 900px the bar carries
         its own backdrop — otherwise cream sections scroll straight under the
         gold mark and the menu button with nothing behind them. */
      .landing-navbar { background: rgba(12,18,13,.58); backdrop-filter: blur(12px); }
      .landing-menu-toggle { min-height: 44px; display: inline-flex; align-items: center; gap: 8px; border: 1px solid rgba(255,255,255,.3); border-radius: var(--radius-sm); padding: 8px 13px; color: #fff; background: rgba(12,18,13,.42); backdrop-filter: blur(6px); font-size: 13px; font-weight: 850; cursor: pointer; }
      .landing-menu-toggle::before { content: ""; width: 15px; height: 10px; border-top: 2px solid currentColor; border-bottom: 2px solid currentColor; box-shadow: 0 4px 0 currentColor inset; }
      .landing-menu-toggle[aria-expanded="true"] { border-color: rgba(231,197,116,.62); background: rgba(231,197,116,.16); }
      /* Panel drops out of the bar; .landing-nav is already position:relative. */
      .landing-nav[data-menu-open="true"] .landing-links { display: grid; position: absolute; top: 100%; left: 0; right: 0; gap: 2px; padding: 8px clamp(20px,4vw,42px) 14px; border-radius: 0 0 12px 12px; background: #0f150f; box-shadow: 0 18px 40px rgba(0,0,0,.34); }
      .landing-nav[data-menu-open="true"] .landing-links a { min-height: 44px; display: flex; align-items: center; font-size: 15px; }
      .landing-hero { min-height: 88svh; }
      .landing-hero::after { background: linear-gradient(180deg, rgba(12,18,13,.78) 0%, rgba(12,18,13,.5) 46%, rgba(12,18,13,.88) 100%); }
      .landing-copy { padding-top: 64px; }
      .section-head { grid-template-columns: minmax(0,1fr); gap: 16px; align-items: start; }
      .section h2 { max-width: 24ch; }
      .imprint-grid { grid-template-columns: repeat(2,minmax(0,1fr)); }
      /* Three cards in two columns would leave a half-width orphan. */
      .imprint-card:last-child { grid-column: 1 / -1; }
      .price-examples { grid-template-columns: repeat(3,minmax(0,1fr)); }
      .price-examples-label { grid-column: 1 / -1; padding: 18px 20px 0; border-right: 0; }
      .price-example { padding: 16px 20px 20px; }
      .price-example:last-child { padding-right: 20px; }
    }
    @media (max-width: 640px) {
      .feature-grid, .product-state, .price-summary, .price-examples, .imprint-grid, .landing-more-grid, .legal-grid { grid-template-columns: minmax(0,1fr); }
      .feature:nth-child(2n) { border-bottom: 1px solid var(--line); }
      .feature, .feature:last-child { grid-column: auto; grid-template-columns: 40px minmax(0,1fr); align-items: start; padding: 18px 20px; }
      .feature:last-child .feature-icon { grid-row: 1 / span 2; }
      .feature:last-child strong { margin-bottom: 6px; }
      .feature:last-child p { grid-column: 2; grid-row: 2; }
      /* Stacked, the number belongs beside the line rather than above it —
         four full-width blocks otherwise cost a screen of scrolling. */
      .trust-summary { grid-template-columns: minmax(0,1fr); }
      .trust-line { grid-template-columns: 30px minmax(0,1fr); column-gap: 14px; padding-top: 20px; }
      .trust-line:nth-child(-n+3) { padding-bottom: 20px; border-bottom: 1px solid var(--line); }
      .trust-number { grid-row: 1 / span 2; margin-bottom: 0; }
      .trust-line strong, .trust-line p { grid-column: 2; }
      .product-state > div + div, .price-summary > div + div { border-left: 0; border-top: 1px solid var(--line); }
      .product-state > div, .price-summary > div { padding: 22px 20px; }
      .price-examples-label { padding: 18px 20px 0; }
      .price-example { gap: 3px; padding: 14px 20px 16px; }
      .price-example:last-child { padding-right: 20px; }
      .price-example + .price-example { border-left: 0; border-top: 1px solid var(--line); }
      .landing-contact { align-items: flex-start; flex-direction: column; }
    }
    @media (max-width: 520px) {
      .landing-hero { min-height: max(100svh,650px); }
      .landing-nav { padding-top: 20px; }
      .landing-copy { align-self: center; padding-top: 40px; padding-bottom: 30px; }
      .landing-copy h1 { font-size: clamp(39px,12vw,48px); line-height: .96; }
      .landing-lead { font-size: 17px; line-height: 1.46; }
      .landing-actions { display: grid; }
      .landing-button { width: 100%; }
      .landing-button.secondary { width: auto; min-height: 44px; justify-self: start; border-color: transparent; padding-inline: 2px; background: transparent; backdrop-filter: none; }
      .landing-access { display: none; }
      .section { padding: 44px 20px; }
      .section h2 { font-size: 36px; }
      .landing-contact strong { font-size: 24px; }
      footer div { display: grid; }
    }
    @media (prefers-reduced-motion: reduce) {
      html { scroll-behavior: auto; }
      *, *::before, *::after { scroll-behavior: auto !important; transition-duration: .01ms !important; animation-duration: .01ms !important; animation-iteration-count: 1 !important; }
    }
  </style>
  <noscript><style>
    @media (max-width: 900px) {
      .landing-navbar { position: absolute; }
      .landing-nav { flex-wrap: wrap; }
      .landing-menu-toggle { display: none; }
      .landing-links { flex: 1 0 100%; display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 2px 14px; padding-top: 8px; }
      .landing-links a { min-height: 44px; display: flex; align-items: center; }
      .landing-links .js-mail-link { display: none; }
      .landing-hero { padding-top: 176px; }
    }
  </style></noscript>
</head>
<body>
  <!-- Fixed chrome: sticky header, top veil and the 3D mark. -->

  <!-- Darkens the top strip once scrolled, so the header and mark keep contrast
       over the cream sections. Separate from the stage so it is never scaled. -->
  <div class="mark3d-veil" aria-hidden="true"></div>

  <!-- Scroll-driven 3D mark. Rendered at 7x and scaled down, so it stays crisp
       at every size. data-mark3d-mode="solid" switches back to the gold glass.
       The flat SVG inside is what the mark becomes once parked. -->
  <div class="mark3d-stage" data-hausv-mark-3d data-src="/assets/hausv-mark.svg?v={{.AssetVersion}}" data-glass-src="/assets/hausv-mark.glb?v={{.AssetVersion}}" data-start="center" data-min-width="900" aria-hidden="true">
    <svg class="mark3d-flat" viewBox="0 0 72 42" focusable="false" aria-hidden="true">
      <path d="M9 35h54"/><path d="M11 35V23l8-6 8 6v12"/>
      <path d="M45 35V23l8-6 8 6v12"/><path d="M25 35V17.5L36 9l11 8.5V35"/>
      <path d="M31.5 35v-9h9v9"/><path d="M15.5 27h5"/>
      <path d="M51.5 27h5"/><path d="M31 21h10"/>
    </svg>
  </div>
  <button class="mark3d-top" type="button" data-mark3d-top hidden aria-label="Zum Seitenanfang scrollen"></button>

  <header class="landing-navbar">
    <div class="landing-nav">
      <!-- The flat mark is gone: the 3D one is the logo now. The empty span is
           kept so the nav keeps its space-between layout and the home link
           keeps a click target; the 3D stage measures its vertical position. -->
      <a class="landing-brand" href="/" aria-label="hausv.org"><span class="landing-mark">{{template "hausvLandingMark" .}}</span></a>
      <button class="landing-menu-toggle" type="button" data-landing-menu-toggle aria-expanded="false" aria-controls="landing-navigation">Menü</button>
      <nav id="landing-navigation" class="landing-links" aria-label="Navigation">
        <a href="#funktionen">Funktionen</a>
        <a href="#sicherheit">Sicherheit</a>
        <a href="#preise">Preise</a>
        <a href="#impressum">Impressum</a>
        <a href="#kontakt" class="js-mail-link" data-mail-local="{{.ContactLocal}}" data-mail-domain="{{.ContactDomain}}">{{.ContactDisplay}}</a>
      </nav>
    </div>
  </header>

  <section class="landing-hero">
    <div class="landing-copy">
      <div class="landing-eyebrow">Hausverwaltung &amp; Energiemanagement</div>
      <h1>Alles, was Zuhause anfällt.</h1>
      <p class="landing-lead">Aushänge, Termine, Dokumente und Anliegen – privat an einem Ort. Energie transparent verstehen und Spitzen gezielt vermeiden.</p>
      <div class="landing-actions">
        <a class="landing-button primary js-mail-link" href="#kontakt" data-mail-local="{{.ContactLocal}}" data-mail-domain="{{.ContactDomain}}" data-mail-subject="hausv.org Pilot anfragen" data-mail-reveal="false">Pilot anfragen</a>
        <a class="landing-button secondary" href="#funktionen">Funktionen ansehen</a>
      </div>
      <p class="landing-access"><svg viewBox="0 0 24 24" aria-hidden="true"><rect x="5" y="10" width="14" height="11" rx="2"/><path d="M8 10V7a4 4 0 0 1 8 0v3"/></svg>Privat · Zugang nach Abstimmung</p>
    </div>
  </section>

  <section id="funktionen" class="section features-section">
    <div class="section-inner">
      <div class="section-head">
        <div>
          <p class="section-kicker">Der gemeinsame Arbeitsbereich</p>
          <h2>Was Sie damit tun können.</h2>
        </div>
        <p class="section-lead">Fünf klare Aufgaben statt vieler einzelner Werkzeuge.</p>
      </div>
      <div class="feature-grid">
        <div class="feature"><span class="feature-icon"><svg viewBox="0 0 24 24"><path d="M4 5h16v12H8l-4 3z"/><path d="M8 9h8M8 13h6"/></svg></span><strong>Informieren</strong><p>Aushänge, Termine und Hinweise erreichen alle am richtigen Ort.</p></div>
        <div class="feature"><span class="feature-icon"><svg viewBox="0 0 24 24"><path d="M5 18.5V7a3 3 0 0 1 3-3h8a3 3 0 0 1 3 3v5a3 3 0 0 1-3 3H10z"/><path d="M8.5 8.5h7"/></svg></span><strong>Anliegen klären</strong><p>Melden, nachfragen und den nächsten Schritt nachvollziehen.</p></div>
        <div class="feature"><span class="feature-icon"><svg viewBox="0 0 24 24"><path d="M7 3h7l4 4v14H7z"/><path d="M14 3v5h5"/><path d="M9 13h6M9 17h6"/></svg></span><strong>Unterlagen ordnen</strong><p>Dokumente, Protokolle und Nachweise passend freigeben.</p></div>
        <div class="feature"><span class="feature-icon"><svg viewBox="0 0 24 24"><path d="M6 18V9M12 18V5M18 18v-6"/><path d="M4 18h16"/></svg></span><strong>Entscheiden</strong><p>Abstimmungen und Übergaben verständlich dokumentieren.</p></div>
        <div class="feature"><span class="feature-icon"><svg viewBox="0 0 24 24"><path d="M4 13h5l2-7 3 12 2-5h4"/><path d="M5 20h14"/></svg></span><strong>Energie verstehen</strong><p>Verbrauch und Spitzen beobachten. Aktive Steuerung bleibt bewusst geschlossen.</p></div>
      </div>
      <div class="product-state" aria-label="Produktstand">
        <div><span>Heute im privaten Pilot</span><strong>Hausalltag an einem Ort</strong><p>Aushänge, Termine, Anliegen, Dokumente, Abstimmungen, Übergaben und ein verständlicher Verlauf.</p></div>
        <div><span>Nächste Ausbaustufe</span><strong>Gezielte Übergaben</strong><p>Dienstleister-Zugänge sind derzeit geschlossen. Fachsystem-Anbindungen entstehen schrittweise.</p></div>
      </div>
      <details id="ausblick" class="landing-more">
        <summary>Produktstand im Detail</summary>
        <div class="landing-more-grid">
          <div><h3>Heute nutzbar</h3><ul><li>Kalender abonnieren</li><li>Kontakte wiederverwenden</li><li>Zahlungsstatus geschützt anzeigen</li><li>Änderungen nachvollziehen</li></ul></div>
          <div><h3>Bewusst begrenzt</h3><ul><li><small>Vorbereitet</small> Dienstleister-Koordination</li><li><small>Schrittweise</small> Übergabe an bestehende Fachsysteme</li><li>Keine eigene Buchhaltung, kein Mahnwesen und keine Zahlungsaufträge</li></ul></div>
        </div>
      </details>
    </div>
  </section>

  <section id="sicherheit" class="section trust-section">
    <div class="section-inner">
      <div class="section-head">
        <div>
          <p class="section-kicker">Sicherheit & Datenschutz</p>
          <h2>Vertrauen zuerst.</h2>
        </div>
        <p class="section-lead">Einfach für die Hausgemeinschaft, nachvollziehbar für den Betrieb.</p>
      </div>
      <div class="trust-summary" aria-label="Sicherheitsprinzipien">
        <div class="trust-line"><span class="trust-number">01</span><strong>Getrennte Häuser</strong><p>Eigene Domain, eigene Rollen, eigene Sichtbarkeit.</p></div>
        <div class="trust-line"><span class="trust-number">02</span><strong>Geschützte Dateien</strong><p>Downloads nur über geprüfte App-Wege.</p></div>
        <div class="trust-line"><span class="trust-number">03</span><strong>Datensparsam</strong><p>Nur Angaben, die der Betrieb wirklich braucht.</p></div>
        <div class="trust-line"><span class="trust-number">04</span><strong>KI nur mit Opt-in</strong><p>Keine automatische Auswertung ohne Zustimmung.</p></div>
      </div>
    </div>
  </section>

  <section id="preise" class="section cost-section">
    <div class="section-inner">
      <div class="section-head">
        <div>
          <p class="section-kicker">Fair geregelt</p>
          <h2>Einfach gerechnet.</h2>
        </div>
        <p class="section-lead">Der Pilot ist persönlich abgestimmt. Für später gilt eine klare Richtung pro Wohnungseinheit – nicht pro Haus und nicht pro Zubehör.</p>
      </div>
      <div class="price-panel">
        <div class="price-summary" aria-label="Faire Nutzung und Preise">
          <div><span>Heute im Pilot</span><strong>Bis 25 Einheiten im Pilot kostenlos</strong><p>Zugang und Umfang werden persönlich abgestimmt.</p></div>
          <div><span>Richtwert für später</span><strong>1 € je Einheit und Monat</strong><p>Gemeint ist eine Wohnung oder vergleichbare Nutzungseinheit. Unverbindlicher Zukunftsrichtwert, noch kein öffentliches Vertragsangebot.</p></div>
        </div>
        <div class="price-examples" aria-label="Preisbeispiele für den Zukunftsrichtwert">
          <p class="price-examples-label">Rechenbeispiele zum Richtwert</p>
          <div class="price-example"><b>8 €</b><span>Kleines Haus · 8 Wohnungen / Monat</span></div>
          <div class="price-example"><b>25 €</b><span>Kleine Verwaltung · 25 Wohnungen / Monat</span></div>
          <div class="price-example"><b>100 €</b><span>Größere Verwaltung · 100 Wohnungen / Monat</span></div>
        </div>
      </div>
      <p class="price-footnote">Wohnungen und vergleichbare Nutzungseinheiten zählen. Zubehör wie Keller oder Stellplätze zählt nicht automatisch. Spenden bleiben freiwillig.</p>
    </div>
  </section>

  <section id="impressum" class="section imprint-section">
    <div class="section-inner">
      <div class="section-head">
        <div>
          <p class="section-kicker">Impressum</p>
          <h2>Impressum & Kontakt</h2>
        </div>
        <p class="section-lead">Direkter Kontakt statt anonymer Hotline.</p>
      </div>
      <div class="imprint-grid">
        <div class="imprint-card"><strong>Medieninhaber / Betreiber</strong><p>{{.OperatorName}} · natürliche Person</p></div>
        <div class="imprint-card"><strong>Ladungsfähige Anschrift</strong><p>{{.OperatorAddress}}</p></div>
        <div class="imprint-card"><strong>Kontakt</strong><p><a id="kontakt" class="js-mail-link" href="#kontakt" data-mail-local="{{.ContactLocal}}" data-mail-domain="{{.ContactDomain}}">{{.ContactDisplay}}</a></p></div>
      </div>
      <details class="legal-details">
        <summary>Rechtliche Details</summary>
        <div class="legal-grid">
          <div class="imprint-card"><strong>Zweck des Angebots</strong><p>Information und technischer Pilot einer Kommunikations- und Transparenzplattform für Hausgemeinschaften.</p></div>
          <div class="imprint-card"><strong>Firmenbuch / UID</strong><p>Nicht anwendbar: privates Projekt einer natürlichen Person, kein Unternehmen und derzeit kein öffentlicher Online-Vertragsabschluss.</p></div>
          <div class="imprint-card"><strong>Gewerbebehörde / Kammer</strong><p>Nicht anwendbar: der aktuelle persönliche Pilot wird nicht gewerblich angeboten.</p></div>
          <div class="imprint-card"><strong>Blattlinie</strong><p>Information über hausv.org und digitale Selbstverwaltung für Mehrparteienhäuser.</p></div>
        </div>
        <p class="mini">Betreiber-Selbstprüfung vom {{.LegalReviewDate}} anhand von <a href="https://www.ris.bka.gv.at/NormDokument.wxe?Abfrage=Bundesnormen&Gesetzesnummer=20001703&Paragraf=5" rel="noopener noreferrer">§ 5 ECG (RIS)</a>, <a href="https://www.usp.gv.at/themen/brancheninformationen/information-und-kommunikation/impressumspflicht-gemaess-para-24-mediengesetz.html" rel="noopener noreferrer">§ 24 MedienG (USP)</a> und der <a href="https://www.dsb.gv.at/" rel="noopener noreferrer">Österreichischen Datenschutzbehörde</a>. Keine externe Zertifizierung oder Rechtsberatung.</p>
      </details>
      <div class="landing-contact">
        <div><strong>Passt das zu Ihrem Haus?</strong><p>Wir klären persönlich, ob der private Pilot sinnvoll ist.</p></div>
        <a class="landing-button js-mail-link" href="#kontakt" data-mail-local="{{.ContactLocal}}" data-mail-domain="{{.ContactDomain}}" data-mail-subject="hausv.org Pilotzugang" data-mail-reveal="false">Pilot anfragen</a>
      </div>
    </div>
  </section>

  <footer>
    <div><span>hausv.org · sicher, fair und datensparsam</span><span><a href="/datenschutz">Datenschutz</a> · <a href="#impressum">Impressum</a> · <a class="js-mail-link" href="#kontakt" data-mail-local="{{.ContactLocal}}" data-mail-domain="{{.ContactDomain}}">{{.ContactDisplay}}</a> · {{.AppVersion}}</span></div>
  </footer>
</body>
</html>
{{end}}

{{define "privacy"}}
<!doctype html>
<html lang="de">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  <meta name="description" content="Datenschutzinformationen für hausv.org">
  <link rel="icon" type="image/svg+xml" href="/favicon.svg">
  <style>
    :root { color-scheme: light; {{template "designTokens" .}} }
    * { box-sizing: border-box; }
    body { margin: 0; background: var(--paper); color: var(--ink); font-family: var(--font-sans); }
    a { color: var(--leaf); }
    :where(a):focus-visible { outline: 3px solid var(--gold-light); outline-offset: 3px; }
    header, main, footer { width: min(820px, calc(100% - 40px)); margin: 0 auto; }
    header { padding: 34px 0 22px; display: flex; justify-content: space-between; gap: 20px; align-items: center; }
    header a { font-weight: 800; text-decoration: none; }
    main { padding-bottom: 56px; }
    h1, h2 { font-family: var(--font-serif); }
    h1 { margin: 20px 0 12px; font-size: clamp(38px, 7vw, 58px); line-height: 1; }
    h2 { margin: 34px 0 10px; font-size: 25px; }
    p, li { line-height: 1.62; }
    .lead { color: var(--muted); font-size: 18px; }
    .status { border: 1px solid rgba(47,107,74,.22); border-radius: var(--radius-md); background: var(--panel); padding: 16px 18px; }
    .status strong { display: block; margin-bottom: 5px; }
    dl { display: grid; grid-template-columns: 180px minmax(0,1fr); gap: 8px 18px; }
    dt { font-weight: 800; }
    dd { margin: 0; }
    footer { border-top: 1px solid var(--line); padding: 22px 0 36px; color: var(--muted); font-size: 14px; }
    @media (max-width: 620px) { dl { grid-template-columns: 1fr; } dt { margin-top: 8px; } }
  </style>
</head>
<body>
  <header><a href="/">← Zurück zur Startseite</a><span>{{.AppVersion}}</span></header>
  <main>
    <h1>Datenschutz</h1>
    <p class="lead">Diese Information beschreibt den tatsächlichen Pilotbetrieb von hausv.org. Sie ist eine dokumentierte Betreiber-Selbstprüfung nach den Grundsätzen der DSGVO, kein Zertifikat und keine unabhängige Rechtsberatung.</p>

    <div class="status">
      <strong>Dienstleister-Zugang: {{if .ServiceProviderEnabled}}für den begrenzten Pilot freigegeben{{else}}geschlossen{{end}}</strong>
      {{if .ServiceProviderEnabled}}Die Betreiberprüfung {{.ServiceProviderAssessment}} wurde ausdrücklich aktiviert. Dienstleister sehen ausschließlich offene, ihnen zugewiesene Anliegen.{{else}}Ohne ausdrücklich versionierte Betreiberfreigabe können keine Dienstleister eingeladen, zugeordnet oder angemeldet werden.{{end}}
    </div>

    <h2>Wer entscheidet worüber?</h2>
    {{if .PortalClassified}}
      {{if .IsPrivateHome}}
      <p>Bei einem privaten Zuhause entscheidet die Eigentümerin oder der Eigentümer über die eigenen Hausinhalte, Energie-Zuordnungen und ausdrücklich freigegebene Vertrauenspersonen. hausv.org verantwortet den sicheren technischen Portalbetrieb, Konten, Zugriffsschutz und die im Produkt angeforderten Auswertungen. Eine technische Vertrauensperson erhält nur den widerrufbaren, hausbezogenen Zugriff, der im Portal sichtbar freigegeben wurde.</p>
      {{else}}
      <p>Bei einer Hausgemeinschaft entscheidet die Eigentümergemeinschaft beziehungsweise die beauftragte Hausverwaltung über gemeinschaftliche Inhalte und Zwecke. hausv.org verantwortet den sicheren technischen Portalbetrieb, Konten und Zugriffsschutz und verarbeitet Hausinhalte weisungsgebunden, soweit dies für den konkreten Zweck vereinbart ist. Die datenschutzrechtliche Rolle wird deshalb je Zweck bestimmt und nicht pauschal aus einer Produktbezeichnung abgeleitet.</p>
      {{end}}
    {{else}}
    <p>Solange noch keine Wohnform gewählt wurde oder das Energieprofil zurückgesetzt ist, weist hausv.org hier keine Eigentümer- oder Hausgemeinschaftsrolle pauschal zu. Über konkrete Hausinhalte entscheidet die tatsächlich zuständige Eigentümerin, Eigentümergemeinschaft oder beauftragte Verwaltung; hausv.org verantwortet Konten, Zugriffsschutz und den technischen Portalbetrieb jeweils für den konkreten Zweck.</p>
    {{end}}
    <dl>
      <dt>{{if .PortalClassified}}{{if .IsPrivateHome}}Hauskontakt{{else}}Verantwortlicher Hausbetrieb{{end}}{{else}}Hauskontakt{{end}}</dt><dd>{{.HouseContactName}}{{if .HouseContactAddress}}, {{.HouseContactAddress}}{{end}}</dd>
      <dt>{{if .PortalClassified}}{{if .IsPrivateHome}}Betroffenes Zuhause{{else}}Betroffenes Haus{{end}}{{else}}Betroffene Adresse{{end}}</dt><dd>{{.Tenant.Address}}</dd>
      <dt>Kontakt</dt><dd><a href="mailto:{{.HouseContactEmail}}">{{.HouseContactEmail}}</a>{{if .HouseContactPhone}} · {{.HouseContactPhone}}{{end}}</dd>
      <dt>Technischer Betrieb</dt><dd>{{.TechnicalOperatorName}}, {{.TechnicalOperatorAddress}} · <a href="mailto:{{.TechnicalContactEmail}}">{{.TechnicalContactEmail}}</a></dd>
    </dl>

    <h2>Welche Daten und wofür?</h2>
    <ul>
      <li>Identität, Hauszugehörigkeit, Rollen und Rechte für Anmeldung und Zugriffsschutz.</li>
      <li>Aushänge, Termine, Dokumente, Anliegen, Kommentare, Anhänge und Abstimmungen für Kommunikation und Verwaltung des Hauses.</li>
      <li>Anmelde- und Auditdaten für Sicherheit, Fehlerklärung und nachvollziehbare Änderungen.</li>
      <li>Parkplatz- und Ladedaten nur für berechtigte Personen des jeweiligen Hauses.</li>
      <li>Der einmalige Beginn des dreijährigen kostenlosen Nutzungszeitraums bleibt als Vertrags- und Anspruchsmerkmal erhalten, damit eine Neueinrichtung den Zeitraum nicht neu startet. Dieses Datum enthält keine Messwerte.</li>
      {{if .EnergyProfileExists}}<li>Energieprofil mit Wohnform, Anzeigename, verknüpfter Einheit, Anlagen, Wartungsplänen und bestätigten Messwert-Zuordnungen.</li>
      <li>Aus Home Assistant werden bei der Einrichtung kurzzeitig verfügbare Entitäten zur Auswahl gelesen. Dauerhaft gespeichert werden nur bestätigte Zuordnungen; ausgewählte Live-Werte und Verläufe werden für die Anzeige abgerufen, aber nicht als eigene Home-Assistant-Kopie gespeichert.</li>
      <li>Ist der Netzbezug bestätigt, wird er laufend gelesen und je abgeschlossener Viertelstunde ein Mittelwert aufgezeichnet. Gespeichert wird nur dieser Viertelstundenwert mit seiner Güte, nicht der einzelne Messwert.</li>
      <li>Hochgeladene Smart-Meter-Originaldateien, daraus normalisierte Viertelstundenwerte sowie daraus abgeleitete Spitzen, Tarifstände, Empfehlungen und Vorher-/Nachher-Vergleiche.</li>{{end}}
    </ul>
    <p>Die Rechtsgrundlage wird je Zweck gewählt: objektiv notwendige Kernfunktionen auf Grundlage des angeforderten Portalvertrags (Art. 6 Abs. 1 lit. b DSGVO), Zugriffsschutz und eng begrenzte Sicherheitsnachweise auf Grundlage berechtigter Interessen (Art. 6 Abs. 1 lit. f DSGVO) und gesetzliche Pflichten nur, wenn sie im Einzelfall tatsächlich anwendbar und dokumentiert sind (Art. 6 Abs. 1 lit. c DSGVO). Freiwillige Zusatzfreigaben können widerrufen werden. Nicht erforderliche Zweitnutzungen für Werbung, Training oder allgemeine Produktanalyse finden ohne eigene Rechtsgrundlage und ausdrückliche Aktivierung nicht statt. Freitext und Fotos sollen keine Gesundheitsdaten, Ausweiskopien oder andere besonders geschützte Angaben enthalten.</p>
    {{if .EnergyProfileExists}}<p>Die Energieansicht ist zunächst <strong>nur lesend</strong>. Empfehlungen sind nachvollziehbare Hinweise; HAUSV trifft keine ausschließlich automatisierte Entscheidung mit rechtlicher oder ähnlich erheblicher Wirkung. Eine spätere aktive Steuerung bleibt gesondert geschlossen, bis sie bewusst freigegeben und datenschutzrechtlich neu geprüft wurde.</p>{{end}}

    <h2>Quellen, Empfänger und Speicherorte</h2>
    <ul>
      <li>Daten stammen von eingeladenen Personen, der Hausadministration, ausdrücklich verbundenen Home-Assistant-Instanzen und bewusst hochgeladenen Smart-Meter-Dateien.</li>
      <li>Innerhalb eines Hauses sehen nur die jeweils berechtigten Rollen die für ihre Aufgabe notwendigen Bereiche. Technische Vertrauenspersonen sehen oder konfigurieren Energie nur im sichtbar erteilten Umfang und dürfen den Haus-Schalter nicht umlegen. Der Zugriff ist widerrufbar.</li>
      <li>Die Fachdaten und das selbst betriebene Zitadel für SSO liegen auf dem Netcup-Server <code>csb1</code> in Wien und sind je Haus und Rolle getrennt.</li>
      <li>Cloudflare schützt und vermittelt den öffentlichen Webzugriff. Dabei fallen technisch notwendige Verbindungsdaten an.</li>
      <li>Verschlüsselte Sicherungen werden in einem Hetzner Storage Box Konto innerhalb der EU gespeichert.</li>
      <li>Resend versendet Transaktionsmails über die Region Irland. E-Mail-Adresse, Betreff und Inhalt sowie Kontodaten, Metadaten, Logs und API-Aufzeichnungen werden dabei auch in den USA verarbeitet. Resend stellt eine Vereinbarung zur Auftragsverarbeitung einschließlich Standardvertragsklauseln bereit und hält reguläre E-Mail-Inhalte 30 Tage vor.</li>
      <li>Es gibt keine Werbung, keine Analyse-Skripte und keine extern geladenen Web-Schriften.</li>
      <li>Die festen Kartenausschnitte auf der Anmeldeseite und in der Portalnavigation nutzen OpenStreetMap-Kartenkacheln. hausv.org ruft ausschließlich die für das konfigurierte Haus benötigten Kacheln serverseitig ab und speichert sie mindestens sieben Tage zwischen; OpenStreetMap erhält dabei weder die IP-Adresse noch Anmelde- oder Kontodaten der Portalbesuchenden. Erst beim bewussten Öffnen des Kartenlinks baut der Browser eine direkte Verbindung zu OpenStreetMap auf.</li>
      {{if .EnergyProfileExists}}<li>Home-Assistant-Endpunkt und Zugangstoken bleiben in der verschlüsselten Host-Konfiguration. Sie werden weder in der Fachdatenbank noch im Energieexport gespeichert oder angezeigt.</li>{{end}}
    </ul>

    <h2>Aufbewahrung</h2>
    <ul>
      <li>Einmalige E-Mail-Anmeldelinks: 15 Minuten; OIDC-Anmeldevorgänge: 10 Minuten; beide nur einmal nutzbar.</li>
      <li>Sitzungscookie: regulär höchstens 30 Tage oder bis zur Abmeldung beziehungsweise Sperre.</li>
      <li>Hauszugehörigkeit und Dienstleister-Zugriff: bis zum Entzug; der Zugriff endet sofort.</li>
      <li>Gelöschte Anhangdateien: sofort entfernt; leere Löschmarkierung nach einem Jahr.</li>
      <li>Geschlossene Anliegen samt Kommentaren und Anhängen: jährliche Prüfung, regulär Löschung nach {{.ServiceProviderRetentionYears}} Jahren, sofern keine offene Gewährleistungs-, Rechts- oder Dokumentationspflicht entgegensteht.</li>
      {{if .EnergyProfileExists}}<li>Smart-Meter-Originaldateien werden nach 30 Tagen, normalisierte Viertelstundenwerte nach 13 Monaten und festgehaltene Tarifbewertungen nach drei Jahren zur Löschung fällig. Die technische Löschung erfolgt beim Start und danach alle sechs Stunden, also spätestens innerhalb weiterer sechs Stunden.</li>
      <li>Energieprofil, Anlagen und bestätigte Zuordnungen: bis zur Korrektur, Trennung oder ausdrücklichen Löschung des Energieprofils. Einzelne Home-Assistant-Zustände und -Historien werden nicht dauerhaft als eigene Kopie gespeichert; aufgezeichnet wird je abgeschlossener Viertelstunde ein Mittelwert, der derselben Frist von 13 Monaten unterliegt.</li>{{end}}
      <li>Der Beginn des kostenlosen Anspruchs bleibt bis zum Ende des Anspruchs- beziehungsweise Portalverhältnisses erhalten, auch wenn das übrige Energieprofil gelöscht wird.</li>
      <li>Auditdaten werden nach drei Jahren zur Löschung fällig und spätestens beim nächsten sechsstündlichen Bereinigungslauf entfernt; das laufende Protokoll rotiert zusätzlich nach Größe oder Alter.</li>
      <li>Gelöschte Daten können bis zum Ablauf des dokumentierten betrieblichen Backup-Zyklus noch in verschlüsselten Sicherungskopien enthalten sein. Diese Kopien bleiben gesperrt und werden ausschließlich für eine kontrollierte Wiederherstellung verwendet.</li>
    </ul>

    <h2>Ihre Kontrolle und Rechte</h2>
    <p>Betroffene Personen können Information, Auskunft, Berichtigung, Löschung, Einschränkung, Datenübertragbarkeit oder Widerspruch verlangen. Eigentümer und Hausadministration können unter <a href="/app/settings/energy-data">Energiedaten &amp; Datenschutz</a> ein maschinenlesbares ZIP-Paket anfordern, den gesamten Messverlauf löschen oder das Energieprofil zurücksetzen. Unabhängige Anliegen, Dokumente und Sicherheitsnachweise folgen ihren eigenen Fristen und werden dort klar getrennt ausgewiesen.</p>
    <p>Anfragen gehen an den oben genannten Hauskontakt; technisch notwendige Unterstützung leistet hausv.org. Eine erteilte Vertrauenspersonen-Freigabe kann in der Personenverwaltung jederzeit entzogen werden. Beschwerden können an die <a href="https://dsb.gv.at/" rel="noopener noreferrer">Österreichische Datenschutzbehörde</a> gerichtet werden.</p>

    <h2>Stand und Überprüfung</h2>
    <p>Stand: {{.LegalReviewDate}}. Die Selbstprüfung stützt sich auf die <a href="https://eur-lex.europa.eu/eli/reg/2016/679/oj" rel="noopener noreferrer">DSGVO</a>, Leitlinien des <a href="https://www.edpb.europa.eu/documents/guideline/guidelines-072020-on-the-concepts-of-controller-and-processor-in-the-gdpr_en" rel="noopener noreferrer">Europäischen Datenschutzausschusses zu Verantwortlichen und Auftragsverarbeitern</a>, Informationen der <a href="https://dsb.gv.at/" rel="noopener noreferrer">Österreichischen Datenschutzbehörde</a>, <a href="https://www.ris.bka.gv.at/NormDokument.wxe?Abfrage=Bundesnormen&Gesetzesnummer=20001703&Paragraf=5" rel="noopener noreferrer">§ 5 ECG (RIS)</a> und <a href="https://www.usp.gv.at/themen/brancheninformationen/information-und-kommunikation/impressumspflicht-gemaess-para-24-mediengesetz.html" rel="noopener noreferrer">§ 24 MedienG (USP)</a>. Sie ist eine interne Betreiberbewertung anhand verfügbarer Primärquellen – ausdrücklich keine externe Zertifizierung oder Rechtsberatung. Solange kein externer Auditor verfügbar ist, bleibt diese dokumentierte Selbstprüfung der Freigabeweg. Sie wird mindestens jährlich sowie bei neuen Empfängern, Datenarten, Rechtsgrundlagen, Speicherorten oder wesentlichen Produktänderungen erneut durchgeführt. Wenn sich ein hohes, nicht ausreichend gemindertes Risiko zeigt, bleibt die Funktion geschlossen und die Datenschutzbehörde wird nach Art. 36 DSGVO konsultiert.</p>
  </main>
  <footer>hausv.org · Datenschutzinformation für den Pilotbetrieb</footer>
</body>
</html>
{{end}}

{{define "appStyles"}}
  <style>
    :root {
      color-scheme: light;
{{template "designTokens" .}}
    }
    * { box-sizing: border-box; }
    body { margin: 0; background: var(--paper); color: var(--ink); }
    a { color: inherit; }
    button, input { font: inherit; }
    [data-home-display-name], [data-home-unit-label] { min-width: 0; overflow-wrap: anywhere; }
    /* Accessibility convention: all keyboard-reachable controls keep a visible focus ring. */
    :where(a, button, input, select, textarea, summary, [tabindex]):focus-visible { outline: 3px solid var(--gold); outline-offset: 3px; }
    .app-shell { min-height: 100vh; display: grid; grid-template-columns: 264px minmax(0,1fr); background: var(--paper); }
    .sidebar { position: sticky; top: 0; height: 100vh; min-height: 0; display: flex; flex-direction: column; gap: 18px; padding: 22px 16px 18px; color: rgba(255,255,255,.86); background: radial-gradient(circle at 20% 0%, rgba(255,255,255,.08), transparent 28%), var(--nav); border-right: 1px solid rgba(255,255,255,.08); }
    .side-brand { flex: 0 0 auto; display: grid; gap: 0; margin: -22px -16px 0; padding: 0 0 10px; }
    .side-map { position: relative; width: 100%; height: 210px; overflow: hidden; border: 0; border-radius: 0; background: #d8d2c4; color: inherit; isolation: isolate; }
    .side-map::after { content: ""; position: absolute; inset: 0; z-index: 2; pointer-events: none; background: linear-gradient(180deg, rgba(247,243,234,.08) 0%, rgba(23,32,25,.06) 48%, rgba(23,32,25,.48) 73%, rgba(23,32,25,.93) 94%, var(--nav) 100%); box-shadow: inset 0 -18px 28px rgba(23,32,25,.42); }
    .side-map-tiles { position: absolute; inset: 0; z-index: 1; pointer-events: none; filter: saturate(.54) sepia(.1) contrast(.86) brightness(.97); }
    .side-map-tile { position: absolute; width: 256px; height: 256px; max-width: none; display: block; user-select: none; pointer-events: none; }
    .side-map-fallback { position: absolute; inset: 0; z-index: 1; opacity: .72; background-color: #d9d5c9; background-image: linear-gradient(28deg, transparent 47%, rgba(255,255,255,.9) 48% 52%, transparent 53%), linear-gradient(118deg, transparent 46%, rgba(255,255,255,.72) 47% 51%, transparent 52%), linear-gradient(90deg, rgba(23,38,29,.12) 1px, transparent 1px), linear-gradient(rgba(23,38,29,.12) 1px, transparent 1px); background-size: 120px 90px, 145px 110px, 42px 42px, 42px 42px; }
    .side-map-pin { position: absolute; left: 50%; top: 50%; z-index: 4; width: 44px; height: 56px; color: var(--gold-light); filter: drop-shadow(0 6px 7px rgba(23,32,25,.38)); transform: translate(-50%,-100%); }
    .side-map-pin-shape { position: absolute; inset: 0; width: 100%; height: 100%; display: block; overflow: visible; fill: var(--nav); stroke: rgba(231,216,177,.88); stroke-width: 1.25; stroke-linejoin: round; }
    .side-map-pin-mark { position: absolute; left: 50%; top: 9px; width: 25px; height: 22px; display: grid; place-items: center; transform: translateX(-50%); }
    .side-map-pin-mark svg { width: 25px; height: 21px; display: block; stroke: currentColor; stroke-width: 2.2; fill: none; stroke-linecap: round; stroke-linejoin: round; }
    .side-place-copy { position: relative; z-index: 6; min-width: 0; display: grid; gap: 12px; margin-top: -54px; padding: 0 22px 15px; color: inherit; text-decoration: none; }
    .side-address-label { min-width: 0; justify-self: start; color: rgba(255,255,255,.7); font-family: var(--font-sans); font-size: 12.5px; font-weight: 500; line-height: 1.3; }
    .side-address-short { display: none; }
    .side-place-copy:hover .side-address-label { color: rgba(255,255,255,.92); }
    .side-ruler { height: 12px; border-top: 1px solid rgba(207,171,83,.72); background: linear-gradient(rgba(207,171,83,.72),rgba(207,171,83,.72)) 25% 0/1px 6px no-repeat, linear-gradient(rgba(207,171,83,.82),rgba(207,171,83,.82)) 50% 0/1px 11px no-repeat, linear-gradient(rgba(207,171,83,.72),rgba(207,171,83,.72)) 75% 0/1px 6px no-repeat; }
    .side-portal { min-width: 0; display: flex; align-items: baseline; gap: 4px; justify-self: start; color: inherit; }
    .side-portal strong { font-family: var(--font-sans); color: rgba(255,255,255,.9); font-size: 15px; font-weight: 600; line-height: 1; }
    .side-portal span { color: rgba(255,255,255,.48); font-family: var(--font-sans); font-size: 12px; font-weight: 450; }
    .side-place-copy:hover .side-portal span { color: rgba(255,255,255,.82); }
    .side-nav { flex: 1 1 auto; min-height: 0; display: grid; align-content: start; gap: 5px; overflow-y: auto; overflow-x: hidden; padding-right: 3px; }
    .side-nav::-webkit-scrollbar { width: 7px; }
    .side-nav::-webkit-scrollbar-thumb { border-radius: var(--radius-pill); background: rgba(255,255,255,.16); }
    .mobile-menu-toggle { display: none; }
    .skip-link { position: fixed; left: 12px; top: 10px; z-index: 1000; min-height: 44px; display: flex; align-items: center; transform: translateY(calc(-100% - 16px)); border: 2px solid var(--gold); border-radius: var(--radius-xs); padding: 10px 14px; color: var(--ink); background: var(--panel); box-shadow: var(--shadow-dialog); font-weight: 850; text-decoration: none; transition: transform .16s ease; }
    .skip-link:focus { transform: none; }
    .nav-item { position: relative; min-height: 44px; display: flex; align-items: center; gap: 12px; padding: 9px 12px; border-radius: var(--radius-xs); color: rgba(255,255,255,.78); text-decoration: none; font-size: 15px; font-weight: 600; }
    .nav-item:hover { color: #fff; background: rgba(255,255,255,.06); }
    .nav-item.active { color: #fff; background: rgba(255,255,255,.08); }
    .nav-item.active::before { content: ""; position: absolute; left: -16px; top: 0; bottom: 0; width: 4px; background: var(--gold); }
    .nav-item.disabled { color: rgba(255,255,255,.38); cursor: default; }
    .nav-item.disabled:hover { background: transparent; }
    .nav-icon { width: 23px; height: 23px; display: grid; place-items: center; flex: 0 0 auto; color: currentColor; }
    .nav-icon svg { width: 22px; height: 22px; stroke: currentColor; stroke-width: 1.9; fill: none; stroke-linecap: round; stroke-linejoin: round; }
    .nav-label { min-width: 0; }
    .nav-home-identity { min-width: 0; display: grid; gap: 1px; line-height: 1.08; }
    .nav-home-identity strong { min-width: 0; overflow: hidden; color: inherit; font-size: 15px; font-weight: 700; text-overflow: ellipsis; white-space: nowrap; }
    .nav-home-identity small { min-width: 0; overflow: hidden; color: rgba(255,255,255,.48); font-size: 11px; font-weight: 550; line-height: 1.2; text-overflow: ellipsis; white-space: nowrap; }
    .nav-badge { margin-left: auto; min-width: 25px; height: 22px; display: inline-flex; align-items: center; justify-content: center; border-radius: var(--radius-pill); padding: 0 7px; background: var(--gold); color: #172019; font-size: 11px; font-weight: 900; line-height: 1; }
    .nav-group-label { margin: 10px 12px 2px; color: rgba(255,255,255,.42); font-size: 10px; font-weight: 750; letter-spacing: .1em; text-transform: uppercase; }
    .side-foot { flex: 0 0 auto; margin-top: 0; border-top: 1px solid rgba(255,255,255,.16); padding: 16px 8px 0; display: grid; gap: 12px; }
    .side-map-attribution { justify-self: start; color: rgba(255,255,255,.38); font-size: 9px; line-height: 1.2; text-decoration: none; }
    .side-map-attribution:hover { color: rgba(255,255,255,.68); }
    .side-user { display: grid; grid-template-columns: 42px 1fr; gap: 12px; align-items: center; }
    .avatar { width: 42px; height: 42px; border-radius: 50%; display: grid; place-items: center; background: var(--gold); color: #fff; font-weight: 800; border: 1px solid rgba(255,255,255,.25); }
    .side-user strong { display: -webkit-box; max-height: 2.5em; color: #fff; font-size: 14px; line-height: 1.22; overflow: hidden; overflow-wrap: anywhere; -webkit-line-clamp: 2; -webkit-box-orient: vertical; }
    .side-user span, .side-version { color: rgba(255,255,255,.64); font-size: 13px; }
    .version-button { justify-self: start; width: auto; min-height: 26px; border: 0; padding: 3px 0; background: transparent; color: rgba(255,255,255,.48); font: inherit; font-size: 11.5px; font-weight: 650; cursor: pointer; }
    .version-button:hover { color: rgba(255,255,255,.84); }
    .logout-form { margin: 0; }
    .logout-button { width: 100%; min-height: 42px; display: inline-flex; align-items: center; justify-content: center; gap: 10px; border: 1px solid rgba(255,255,255,.24); border-radius: var(--radius-xs); color: rgba(255,255,255,.92); background: transparent; font-weight: 700; cursor: pointer; }
    .logout-button:hover { border-color: var(--gold); color: #fff; }
    .app-main { min-width: 0; padding-bottom: 58px; }
    /* Energy surfaces use AA-safe text and focus aliases without changing the
       shared portal palette while the other main flows are being revised. */
    .energy-mode-strip, .energy-page, .onboarding-page, .energy-data-page, .home-identity-page { --soft:#716d62; --gold-ink:#705c22; --energy-focus-ring:#ad862c; }
    .energy-mode-strip { position: sticky; top: 0; z-index: 40; min-height: 60px; display: grid; grid-template-columns: auto minmax(0,1fr) auto; gap: 12px; align-items: center; padding: 6px clamp(24px,4vw,40px); color: #fff; background: #17261d; border-bottom: 4px solid var(--gold); box-shadow: 0 8px 24px rgba(23,38,29,.18); }
    .energy-mode-strip.active { background: #562c25; border-bottom-color: #e9b65a; }
    .energy-mode-icon { width: 36px; height: 36px; display: grid; place-items: center; border-radius: 50%; color: #17261d; background: var(--gold-light); font-size: 19px; font-weight: 900; }
    .energy-mode-copy { min-width: 0; display: grid; gap: 2px; }
    .energy-mode-copy strong { font-family: var(--font-serif); font-size: 18px; line-height: 1.05; }
    .energy-mode-copy span { color: rgba(255,255,255,.82); font-size: 12px; line-height: 1.3; }
    .energy-mode-action { min-height: 44px; border: 1px solid rgba(255,255,255,.42); border-radius: var(--radius-xs); padding: 8px 12px; color: #fff; background: rgba(255,255,255,.08); font-weight: 800; line-height: 1.2; cursor: pointer; }
    .energy-mode-action-compact { display: none; }
    .energy-mode-action:hover { border-color: var(--gold-light); background: rgba(255,255,255,.14); }
    .energy-mode-capability { max-width: 220px; color: rgba(255,255,255,.82); font-size: 11.5px; font-weight: 750; line-height: 1.3; text-align: right; }
    .energy-mode-control { position: relative; justify-self: end; }
    .energy-mode-control > summary { list-style: none; }
    .energy-mode-control > summary::-webkit-details-marker { display: none; }
    .energy-mode-popover { position: absolute; z-index: 2; right: 0; top: calc(100% + 10px); width: min(410px,calc(100vw - 32px)); max-height: calc(100dvh - 86px); display: grid; gap: 14px; overflow: auto; border: 1px solid var(--line); border-radius: var(--radius-md); padding: 18px; color: var(--ink); background: var(--surface); box-shadow: var(--shadow-lg); }
    .energy-mode-popover h3 { font-size: 19px; }
    .energy-mode-popover p { color: var(--muted); font-size: 14px; line-height: 1.45; }
    .energy-mode-popover label { display: grid; grid-template-columns: auto 1fr; gap: 9px; align-items: start; font-size: 13px; line-height: 1.4; }
    .energy-mode-popover input[type="text"] { width: 100%; min-height: 44px; border: 1px solid var(--line); border-radius: var(--radius-xs); padding: 9px 11px; background: #fff; }
    .energy-page { width: min(1160px,100%); gap: 22px; }
    .energy-page .button { min-height: 44px; }
    .energy-heading { display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 24px; align-items: end; }
    .energy-heading-copy { min-width: 0; }
    .energy-heading .eyebrow { margin-bottom: 10px; }
    .energy-heading-unit { margin: 4px 0 0; color: var(--muted); font-size: 13px; font-weight: 650; line-height: 1.3; }
    .energy-heading-context { margin: 8px 0 0; color: var(--soft); font-size: 12px; font-weight: 600; }
    .energy-heading .lede { margin-top: 10px; }
    .energy-heading-action { align-self: center; gap: 7px; }
    .energy-heading-action svg { width: 17px; height: 17px; fill: none; stroke: currentColor; stroke-width: 1.8; stroke-linecap: round; stroke-linejoin: round; }
    .energy-source { color: var(--muted); font-size: 13px; }
    /* Live and the next action lead in DOM and visual order. The complete
       tariff keeps a protected 440px column only when the content area can
       actually provide it; narrower desktop/tablet layouts stack safely. */
    .energy-health { display: grid; grid-template-columns: minmax(0,1fr) minmax(440px,440px); gap: 18px; align-items: start; }
    /* An explicit zero-minimum track keeps Linux font metrics or a translated
       button label from widening the live column beyond its mobile viewport. */
    .energy-lead-side { display: grid; grid-template-columns: minmax(0,1fr); align-content: start; gap: 18px; min-width: 0; }
    .energy-health > *, .energy-lead-side > * { min-width: 0; max-width: 100%; }
    .energy-pair { display: grid; grid-template-columns: repeat(auto-fit,minmax(min(380px,100%),1fr)); gap: 18px; align-items: start; }
    .energy-card { border: 1px solid var(--line); border-radius: var(--radius-md); background: var(--surface); box-shadow: var(--shadow-sm); }
    .energy-nextstep { display: grid; align-content: start; gap: 13px; padding: 20px 22px; }
    .energy-nextstep h2 { margin-top: 2px; font-size: 20px; }
    .energy-nextstep p { margin-top: 5px; color: var(--muted); font-size: 13.5px; line-height: 1.5; }
    .energy-nextstep-copy { min-width: 0; }
    .energy-nextstep .energy-recommendation-actions { gap: 8px; justify-content: flex-start; }
    .energy-nextstep .button { padding-left: 14px; padding-right: 14px; font-size: 13.5px; }
    /* „Später“ und „Nicht für uns“ sind Rückzieher, keine gleichrangigen
       Angebote — als drei gleich starke Knöpfe gelesen wirkt die Empfehlung
       wie eine Abstimmung. */
    .energy-nextstep .energy-recommendation-actions > form { gap: 14px; }
    .energy-nextstep .energy-recommendation-actions > form .button { min-height: 44px; border: 0; padding: 0; background: none; box-shadow: none; color: var(--muted); font-size: 12.5px; font-weight: 750; text-decoration: underline; text-underline-offset: 3px; }
    .energy-nextstep .energy-recommendation-actions > form .button:hover { color: var(--ink); }
    .energy-nextstep .energy-next-meta { margin-top: 9px; }
    .energy-next-meta { display: flex; flex-wrap: wrap; gap: 4px 6px; color: var(--soft); font-size: 12px; line-height: 1.45; }
    .energy-next-meta span:not(:last-child)::after { content: " ·"; color: var(--line); }
    .energy-recommendation-actions { display: flex; flex-wrap: wrap; gap: 9px; align-items: center; }
    .energy-recommendation-actions form { display: flex; flex-wrap: wrap; gap: 6px; }
    .energy-measure-control { max-width: 100%; }
    .energy-measure-control[open] { flex: 1 0 100%; }
    .energy-measure-control > summary { list-style: none; }
    .energy-measure-control > summary::-webkit-details-marker { display: none; }
    .energy-measure-control[open] > summary { border-color: var(--gold); }
    .energy-measure-form { width: 100%; display: grid !important; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 11px !important; margin-top: 10px; border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 17px; background: rgba(250,247,238,.84); }
    .energy-measure-form h3, .energy-measure-form p, .energy-measure-form .button { grid-column: 1 / -1; }
    .energy-measure-form p { font-size: 13px; }
    .energy-measure-form label { display: grid; grid-template-columns: auto minmax(0,1fr); gap: 9px; align-items: start; font-size: 13px; line-height: 1.4; }
    .energy-live { overflow: hidden; border: 1px solid var(--line); border-radius: var(--radius-md); background: var(--surface); box-shadow: var(--shadow-sm); }
    .energy-live-head { display: flex; justify-content: space-between; gap: 16px; align-items: start; padding: 20px 22px 16px; border-bottom: 1px solid var(--line); }
    .energy-live-head h2 { font-size: 20px; }
    .energy-live-head span, .energy-live-head small { display: block; color: var(--muted); font-size: 12px; line-height: 1.4; }
    .energy-live-tools { display: grid; gap: 5px; justify-items: end; text-align: right; }
    .energy-live-tools a { color: #715d28; font-size: 11.5px; font-weight: 800; text-decoration: none; }
    .energy-live-tools a:hover { text-decoration: underline; text-underline-offset: 3px; }
    .energy-live-main { min-height: 132px; display: grid; grid-template-columns: 66px minmax(0,1fr); gap: 20px; align-items: center; padding: 24px 22px; }
    .energy-live-main > div { display: grid; gap: 3px; }
    .energy-live-main span, .energy-flow-item span { color: var(--muted); font-size: 12px; }
    .energy-live-main strong { font-family: var(--font-serif); font-size: clamp(32px,4vw,42px); font-weight: 600; line-height: 1; }
    .energy-metric-icon { width: 56px; height: 56px; display: grid; place-items: center; border: 1px solid transparent; border-radius: 16px; }
    .energy-metric-icon svg { width: 35px; height: 35px; display: block; fill: none; stroke: currentColor; stroke-width: 1.7; stroke-linecap: round; stroke-linejoin: round; }
    .energy-metric-icon.load { color: #a24b42; border-color: rgba(162,75,66,.16); background: #fbf1ed; }
    .energy-metric-icon.pv { color: #3e704c; border-color: rgba(62,112,76,.18); background: #f0f5ee; }
    .energy-metric-icon.grid-import { color: #9a7422; border-color: rgba(154,116,34,.18); background: #faf4e5; }
    .energy-metric-icon.grid-export { color: #3f6f74; border-color: rgba(63,111,116,.18); background: #eef5f4; }
    .energy-metric-icon.battery { color: #5d7583; border-color: rgba(93,117,131,.18); background: #eff3f5; }
    .energy-storage-live { min-height: 112px; display: grid; grid-template-columns: 118px minmax(0,1fr); gap: 20px; align-items: center; border-top: 1px solid var(--line); padding: 20px 22px; background: #fbfaf6; }
    .energy-storage-live > div { min-width: 0; display: grid; gap: 2px; }
    .energy-storage-live span, .energy-storage-live small { color: var(--muted); font-size: 11.5px; }
    .energy-storage-live strong { font-family: var(--font-serif); font-size: 26px; line-height: 1.05; }
    .energy-battery-visual { min-width: 0; display: flex; gap: 11px; align-items: center; color: #5d7583; }
    .energy-battery-gauge { position: relative; box-sizing: border-box; width: 88px; height: 42px; display: flex; align-items: stretch; overflow: visible; border: 2px solid #6d706a; border-radius: 8px; padding: 4px; background: #fff; }
    /* Mittig durch Konstruktion, nicht durch Rechnung: das Pseudoelement erbt
       kein border-box, ein fester Randabstand hinge also an der Randbreite. */
    .energy-battery-gauge::after { content: ""; position: absolute; top: 50%; right: -7px; width: 5px; height: 14px; transform: translateY(-50%); border: 2px solid #6d706a; border-left: 0; border-radius: 0 4px 4px 0; background: #fff; }
    .energy-battery-gauge i { height: 100%; max-width: 100%; display: block; border-radius: 4px; background: #607d8d; transition: width .35s ease; }
    /* Die Richtung gehört in den Speicher, nicht daneben: die Pfeile laufen auf
       derselben Achse wie der Füllstand. Laden zeigt nach rechts, Entladen nach
       links — gespiegelt wird der ganze Streifen, damit Form und Bewegung nie
       auseinanderlaufen. Das gilt auch, wenn Bewegung abgeschaltet ist. */
    .energy-battery-flow { position: absolute; inset: 2px 4px; display: flex; align-items: center; justify-content: center; gap: 2px; pointer-events: none; }
    .energy-battery-chevron { width: 7px; height: 12px; display: block; opacity: 0; animation: energy-flow-forward 1.8s ease-in-out infinite; }
    .energy-battery-chevron path { fill: none; stroke: #26312a; stroke-width: 2.2; stroke-linecap: round; stroke-linejoin: round; }
    .energy-battery-chevron:nth-child(2) { animation-delay: .25s; }
    .energy-battery-chevron:nth-child(3) { animation-delay: .5s; }
    .energy-battery-visual.discharging .energy-battery-flow { transform: scaleX(-1); }
    @keyframes energy-flow-forward { 0% { opacity: 0; transform: translateX(-5px); } 38% { opacity: .85; } 100% { opacity: 0; transform: translateX(5px); } }
    @media (prefers-reduced-motion: reduce) {
      .energy-battery-gauge i { transition: none; }
      .energy-battery-chevron { opacity: .62; animation: none; }
    }
    @media (max-width: 560px) {
      .energy-live-main { min-height: 118px; grid-template-columns: 52px minmax(0,1fr); gap: 14px; padding: 20px 18px; }
      .energy-metric-icon { width: 50px; height: 50px; border-radius: 14px; }
      .energy-metric-icon svg { width: 31px; height: 31px; }
      .energy-storage-live { grid-template-columns: 108px minmax(0,1fr); gap: 14px; padding: 18px; }
      .energy-battery-gauge { width: 78px; }
    }
    .energy-flow-grid { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); border-top: 1px solid var(--line); }
    .energy-flow-item { min-height: 112px; display: grid; grid-template-columns: 56px minmax(0,1fr); align-content: center; align-items: center; gap: 16px; padding: 17px 18px; border-right: 1px solid var(--line); border-bottom: 1px solid var(--line); }
    .energy-flow-copy { min-width: 0; display: grid; gap: 3px; }
    .energy-flow-item:nth-child(2n) { border-right: 0; }
    .energy-flow-item:last-child:nth-child(odd) { grid-column: 1 / -1; border-right: 0; }
    .energy-flow-item strong { font-family: var(--font-serif); font-size: 24px; font-weight: 600; line-height: 1.1; }
    .energy-flow-item small { color: var(--muted); font-size: 11.5px; }
    .energy-flow-item.good strong { color: #315f3d; }
    .energy-flow-item.warning strong { color: #806116; }
    .energy-flow-item.danger strong { color: #96372d; }
    @media (max-width: 340px) {
      .energy-flow-grid { grid-template-columns: minmax(0,1fr); }
      .energy-flow-item { min-height: 96px; grid-template-columns: 50px minmax(0,1fr); gap: 14px; border-right: 0; }
      .energy-flow-item:last-child:nth-child(odd) { grid-column: auto; }
    }
    .energy-live-more { margin-top: -1px; }
    .energy-live-more > summary { min-height: 58px; display: flex; justify-content: space-between; gap: 14px; align-items: center; padding: 13px 22px; list-style: none; cursor: pointer; }
    .energy-live-more > summary::-webkit-details-marker { display: none; }
    .energy-live-more > summary::after { content: "⌄"; color: var(--gold-ink); font-size: 20px; transition: transform .16s ease; }
    .energy-live-more[open] > summary::after { transform: rotate(180deg); }
    .energy-live-more > summary span { display: grid; gap: 2px; }
    .energy-live-more > summary small { color: var(--muted); font-size: 11.5px; font-weight: 500; }
    .energy-live-more-list { display: grid; border-top: 1px solid var(--line); }
    .energy-live-more-row { min-height: 48px; display: flex; justify-content: space-between; gap: 18px; align-items: center; padding: 10px 22px; border-bottom: 1px solid var(--line); font-size: 13px; }
    .energy-live-more-row:last-child { border-bottom: 0; }
    .energy-live-more-row strong { font-family: var(--font-serif); font-size: 17px; }
    .energy-live-empty { display: grid; gap: 5px; padding: 24px 22px; }
    .energy-live-empty span { color: var(--muted); font-size: 13px; }
    .energy-chart { display: grid; gap: 18px; scroll-margin-top: 96px; }
    .energy-chart-head { display: flex; justify-content: space-between; gap: 18px; align-items: end; }
    .energy-chart-head h2 { font-size: 26px; }
    .energy-chart-head p, .energy-chart-head small { color: var(--muted); font-size: 12.5px; }
    .energy-chart-head p { margin-top: 4px; }
    .energy-chart-head small { text-align: right; }
    .energy-chart-head-actions { display: grid; gap: 8px; justify-items: end; }
    .energy-chart-toolbar, .energy-chart-size-actions, .energy-chart-dialog-actions { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; }
    .energy-chart-range { display: inline-flex; overflow: hidden; border: 1px solid var(--line); border-radius: 9px; background: #fff; }
    .energy-chart-range a { min-height: 44px; display: inline-flex; align-items: center; padding: 8px 12px; color: var(--muted); font-size: 11.5px; font-weight: 800; text-decoration: none; }
    .energy-chart-range a + a { border-left: 1px solid var(--line); }
    .energy-chart-range a[aria-current="page"] { color: #fff; background: var(--nav); }
    .energy-chart-range a:hover:not([aria-current="page"]) { color: var(--ink); background: #faf7ef; }
    .energy-page .button.energy-chart-size-button { min-height: 44px; flex: 0 0 auto; gap: 7px; background: #fff; }
    .energy-chart-size-button svg { width: 17px; height: 17px; }
    .energy-chart-layout { display: grid; grid-template-columns: minmax(0,1fr) minmax(220px,.28fr); gap: 24px; align-items: stretch; }
    .energy-chart-plot { min-width: 0; display: grid; gap: 10px; }
    .energy-chart-legend { display: flex; flex-wrap: wrap; gap: 8px 18px; }
    .energy-chart-legend span { display: inline-flex; align-items: center; gap: 7px; color: var(--muted); font-size: 12px; }
    .energy-chart-legend i { width: 24px; height: 3px; border-radius: 3px; background: var(--nav); }
    .energy-chart-legend .load i { background: #b85c52; }
    .energy-chart-legend .pv i { background: #477957; }
    .energy-chart-legend .grid i { background: #b18122; }
    .energy-chart-legend .battery i { background: #607987; }
    .energy-chart-legend .threshold i { height: 0; border-top: 2px dashed #6f6b63; background: transparent; }
    .energy-chart-legend .threshold small { font-size: 10.5px; }
    .energy-chart-svg { width: 100%; height: auto; min-height: 220px; display: block; overflow: visible; }
    .energy-chart-svg.mobile { display: none; }
    .energy-chart-grid { stroke: #ddd5c6; stroke-width: 1; }
    .energy-chart-grid.zero { stroke: #a9a08f; stroke-width: 1.35; }
    .energy-chart-axis-label { fill: #756f63; font-family: var(--font-sans); font-size: 11px; }
    .energy-chart-line { fill: none; stroke: var(--nav); stroke-width: 2.4; stroke-linecap: round; stroke-linejoin: round; vector-effect: non-scaling-stroke; }
    .energy-chart-line.load { stroke: #b85c52; }
    .energy-chart-line.pv { stroke: #477957; }
    .energy-chart-line.grid { stroke: #b18122; }
    .energy-chart-line.battery { stroke: #607987; stroke-width: 2.1; }
    .energy-chart-area { stroke: none; }
    .energy-chart-area.load { fill: rgba(184,92,82,.13); }
    .energy-chart-threshold { stroke: #6f6b63; stroke-width: 1.4; stroke-dasharray: 7 5; vector-effect: non-scaling-stroke; }
    .energy-chart-threshold-label { fill: #625e56; font-family: var(--font-sans); font-size: 10.5px; font-weight: 750; }
    .energy-chart-interactive { position: relative; min-width: 0; border-radius: var(--radius-xs); outline: none; }
    .energy-chart-interactive:focus-visible { outline: 3px solid var(--energy-focus-ring); outline-offset: 4px; }
    .energy-chart-hit { fill: transparent; cursor: crosshair; pointer-events: all; }
    .energy-chart-guide { stroke: rgba(27,32,26,.52); stroke-width: 1.2; stroke-dasharray: 3 3; pointer-events: none; vector-effect: non-scaling-stroke; }
    .energy-chart-marker { fill: #fffefb; stroke: var(--nav); stroke-width: 2.5; pointer-events: none; vector-effect: non-scaling-stroke; }
    .energy-chart-marker.load { stroke: #b85c52; }
    .energy-chart-marker.pv { stroke: #477957; }
    .energy-chart-marker.grid { stroke: #b18122; }
    .energy-chart-marker.battery { stroke: #607987; }
    .energy-chart-marker[hidden], .energy-chart-guide[hidden] { display: none; }
    .energy-chart-tooltip { position: absolute; z-index: 5; top: 12px; width: 210px; border: 1px solid rgba(171,157,126,.55); border-radius: 9px; padding: 11px 12px; color: var(--ink); background: rgba(255,254,251,.97); box-shadow: 0 14px 34px rgba(32,37,31,.15); pointer-events: none; }
    .energy-chart-tooltip > strong { display: block; margin-bottom: 7px; font-family: var(--font-serif); font-size: 18px; }
    .energy-chart-tooltip [data-chart-tooltip-values] { display: grid; gap: 5px; }
    .energy-chart-tooltip-row { display: grid; grid-template-columns: 8px minmax(0,1fr) auto; gap: 7px; align-items: center; color: var(--muted); font-size: 11.5px; }
    .energy-chart-tooltip-row b { color: var(--ink); font-size: 12px; }
    .energy-chart-tooltip-row i { width: 7px; height: 7px; border-radius: 50%; background: var(--nav); }
    .energy-chart-tooltip-row i.load { background: #b85c52; }
    .energy-chart-tooltip-row i.pv { background: #477957; }
    .energy-chart-tooltip-row i.grid { background: #b18122; }
    .energy-chart-tooltip-row i.battery { background: #607987; }
    .energy-chart-note { display: grid; align-content: center; gap: 8px; border-left: 1px solid var(--line); padding-left: 24px; }
    .energy-chart-note span { color: var(--gold-ink); font-size: 11px; font-weight: 850; letter-spacing: .08em; text-transform: uppercase; }
    .energy-chart-note strong { font-family: var(--font-serif); font-size: 21px; line-height: 1.25; }
    .energy-chart-note p { color: var(--muted); font-size: 13px; line-height: 1.5; }
    .energy-chart-hint { color: var(--muted); font-size: 11.5px; }
    .energy-chart-empty { display: grid; gap: 5px; border: 1px dashed var(--line); border-radius: var(--radius-sm); padding: 20px; }
    .energy-chart-empty p { color: var(--muted); font-size: 13px; }
    .energy-chart-dialog { width: min(1320px,calc(100vw - 48px)); max-width: none; max-height: calc(100dvh - 48px); border: 0; border-radius: var(--radius-md); padding: 0; overflow: hidden; color: var(--ink); background: transparent; box-shadow: 0 36px 110px rgba(18,24,19,.38); }
    .energy-chart-dialog::backdrop { background: rgba(19,27,21,.56); backdrop-filter: blur(2px); }
    .energy-chart-dialog-shell { max-height: calc(100dvh - 48px); display: grid; gap: 18px; overflow: auto; overscroll-behavior: contain; padding: clamp(24px,3vw,40px); background: #fbf7ee; }
    .energy-chart-dialog-shell > header { display: flex; justify-content: space-between; gap: 24px; align-items: start; }
    .energy-chart-dialog-shell > header h2 { margin-top: 5px; font-size: clamp(28px,3vw,42px); }
    .energy-chart-dialog-shell > header p { margin-top: 5px; color: var(--muted); font-size: 14px; }
    .energy-chart-dialog-actions .button { gap: 7px; }
    .energy-chart-dialog-actions svg { width: 17px; height: 17px; }
    .energy-chart-dialog-plot { border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 16px 18px 4px; background: #fffefb; }
    .energy-chart-dialog .energy-chart-svg { min-height: min(54vh,520px); }
    .energy-chart-dialog .energy-chart-tooltip { top: 26px; }
    .energy-chart-dialog-shell:fullscreen, .energy-chart-dialog.is-fullscreen-fallback .energy-chart-dialog-shell { width: 100vw; height: 100dvh; max-height: none; grid-template-rows: auto auto minmax(0,1fr) auto; border-radius: 0; padding: clamp(20px,2.5vw,42px); background: #fbf7ee; }
    .energy-chart-dialog.is-fullscreen-fallback { position: fixed; inset: 0; width: 100vw; height: 100dvh; max-height: none; margin: 0; border-radius: 0; }
    .energy-chart-dialog-shell:fullscreen .energy-chart-dialog-plot, .energy-chart-dialog.is-fullscreen-fallback .energy-chart-dialog-plot { min-height: 0; display: grid; align-items: center; }
    .energy-chart-dialog-shell:fullscreen .energy-chart-svg.desktop, .energy-chart-dialog.is-fullscreen-fallback .energy-chart-svg.desktop { min-height: min(66vh,680px); }
    .energy-chart-dialog-shell:fullscreen .energy-chart-tooltip, .energy-chart-dialog.is-fullscreen-fallback .energy-chart-tooltip { top: 18px; }
    .energy-card { padding: 24px; }
    .energy-card-head { display: flex; justify-content: space-between; gap: 18px; align-items: start; margin-bottom: 18px; }
    .energy-card-head p { margin-top: 6px; color: var(--muted); font-size: 14px; }
    .energy-admin { display: grid; gap: 10px; margin-top: 4px; }
    .energy-admin-head { display: flex; flex-wrap: wrap; gap: 4px 12px; align-items: baseline; border-top: 1px solid var(--line); padding-top: 16px; }
    .energy-admin-head h2 { font-size: 20px; }
    .energy-admin-head span { color: var(--soft); font-size: 12.5px; }
    .energy-card-quiet { padding: 16px 22px; background: rgba(255,255,255,.55); box-shadow: none; }
    .energy-card-quiet > .energy-card-head h2, .energy-card-quiet > summary h2 { font-family: var(--font-sans); font-size: 17px; font-weight: 800; letter-spacing: -.01em; }
    .energy-card-quiet > .energy-card-head p, .energy-card-quiet > summary p { margin-top: 3px; font-size: 13px; }
    .energy-collapsible > summary { position: relative; margin-bottom: 0; padding-right: 42px; list-style: none; cursor: pointer; }
    .energy-collapsible > summary::-webkit-details-marker { display: none; }
    .energy-collapsible > summary::after { content: "+"; position: absolute; right: 4px; top: 50%; color: var(--gold-ink); font-size: 24px; font-weight: 700; transform: translateY(-50%); }
    .energy-collapsible[open] > summary { margin-bottom: 18px; }
    .energy-collapsible[open] > summary::after { content: "−"; }
    .energy-collapsible-body { display: grid; gap: 14px; border-top: 1px solid var(--line); padding-top: 18px; }
    .energy-roadmap { display: grid; grid-template-columns: repeat(4,minmax(0,1fr)); border: 1px solid var(--line); border-radius: var(--radius-sm); overflow: hidden; }
    .energy-roadmap-step { display: grid; align-content: start; gap: 6px; padding: 14px 16px; background: #fff; border-right: 1px solid var(--line); }
    .energy-roadmap-step:last-child { border-right: 0; }
    .energy-roadmap-step.current { background: #f5f1e6; box-shadow: inset 0 4px 0 var(--gold); }
    .energy-roadmap-number { width: 22px; height: 22px; display: grid; place-items: center; border-radius: 50%; color: #fff; background: var(--nav); font-size: 11px; font-weight: 900; }
    .energy-roadmap-step strong { font-size: 15px; }
    .energy-roadmap-step p { color: var(--muted); font-size: 12.5px; line-height: 1.4; }
    .energy-roadmap-step a { min-height: 36px; display: inline-flex; align-items: center; color: #735c1b; font-size: 12.5px; font-weight: 800; }
    .energy-roadmap-state { color: #8a742d; font-size: 10px; font-weight: 850; letter-spacing: .08em; text-transform: uppercase; }
    .energy-roadmap-card .energy-card-head { margin-bottom: 14px; }
    .energy-assets { display: flex; flex-wrap: wrap; gap: 8px; }
    .energy-asset { display: inline-flex; align-items: center; gap: 7px; border: 1px solid var(--line); border-radius: var(--radius-pill); padding: 8px 12px; background: #fff; font-size: 13px; font-weight: 700; }
    .energy-business-note { display: grid; grid-template-columns: auto minmax(0,1fr); gap: 12px; align-items: center; border-top: 1px solid var(--line); margin-top: 18px; padding-top: 18px; color: var(--muted); font-size: 13px; }
    .energy-business-note strong { color: var(--ink); }
    .energy-related-links { display: flex; flex-wrap: wrap; gap: 5px 12px; margin-top: 14px; font-size: 13px; font-weight: 750; }
    .energy-related-links a, .energy-card-head > a, .energy-tariff-foot > a { min-height: 44px; display: inline-flex; align-items: center; }
    .energy-caretaker-list { display: grid; gap: 10px; }
    .energy-caretaker { display: grid; grid-template-columns: minmax(180px,1fr) repeat(3,auto) auto; gap: 12px; align-items: center; border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 14px; background: #fff; }
    .energy-caretaker > div { display: grid; gap: 2px; }
    .energy-caretaker small { color: var(--muted); overflow-wrap: anywhere; }
    .energy-caretaker label { display: flex; gap: 6px; align-items: center; font-size: 12px; font-weight: 700; }
    .energy-marketplace-gate { display: grid; gap: 3px; margin-top: 16px; border: 1px dashed #c6aa68; border-radius: var(--radius-sm); padding: 14px 16px; background: #fbf7eb; }
    .energy-marketplace-gate span { color: var(--muted); font-size: 13px; line-height: 1.4; }
    .energy-reference-grid { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 12px; }
    .energy-reference-item { display: grid; gap: 3px; border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 14px 16px; background: #fff; }
    .energy-reference-item strong { font-family: var(--font-serif); font-size: 22px; }
    .energy-reference-item span { color: var(--muted); font-size: 12px; }
    .energy-import-form { display: grid; gap: 12px; margin-top: 16px; border-top: 1px solid var(--line); padding-top: 16px; }
    .energy-import-form input[type="file"] { min-height: 48px; border: 1px solid var(--line); border-radius: var(--radius-xs); padding: 10px; background: #fff; }
    .energy-import-form small { color: var(--muted); line-height: 1.4; }
	    .energy-import-history { margin-top: 14px; }
	    .energy-data-page { width: min(1040px,100%); gap: 22px; }
	    .energy-data-heading { display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 24px; align-items: end; }
	    .energy-data-heading .lede { max-width: 700px; margin-top: 10px; }
	    .energy-data-trust { display: flex; flex-wrap: wrap; justify-content: flex-end; gap: 8px 18px; color: var(--muted); font-size: 12px; }
	    .energy-data-trust span { display: inline-flex; gap: 7px; align-items: center; }
	    .energy-data-trust span::before { content: "✓"; width: 22px; height: 22px; display: grid; place-items: center; border: 1px solid rgba(47,107,74,.28); border-radius: 50%; color: #2f6b4a; font-size: 11px; font-weight: 900; }
	    .energy-lifecycle { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); overflow: hidden; border: 1px solid var(--line); border-radius: var(--radius-md); background: rgba(255,255,255,.54); }
	    .energy-lifecycle-step { position: relative; min-height: 92px; display: grid; align-content: center; gap: 4px; padding: 18px 42px 18px 20px; }
	    .energy-lifecycle-step + .energy-lifecycle-step { border-left: 1px solid var(--line); }
	    .energy-lifecycle-step:not(:last-child)::after { content: "→"; position: absolute; right: -10px; top: 50%; z-index: 1; width: 20px; color: var(--gold-ink); background: var(--paper); text-align: center; transform: translateY(-50%); }
	    .energy-lifecycle-step strong { font-family: var(--font-serif); font-size: 18px; }
	    .energy-lifecycle-step span { color: var(--muted); font-size: 12px; line-height: 1.35; }
	    .energy-data-section { border: 1px solid var(--line); border-radius: var(--radius-md); background: var(--surface); box-shadow: var(--shadow-sm); }
	    .energy-data-section-head { display: flex; justify-content: space-between; gap: 20px; align-items: center; padding: 24px; }
	    .energy-data-section-head h2 { font-size: 25px; }
	    .energy-data-section-head p { max-width: 620px; margin-top: 5px; color: var(--muted); font-size: 13.5px; line-height: 1.48; }
	    .energy-data-export .button { flex: 0 0 auto; min-height: 48px; }
	    .energy-data-inventory { display: grid; grid-template-columns: repeat(4,minmax(0,1fr)); border-top: 1px solid var(--line); }
	    .energy-data-inventory div { min-height: 82px; display: grid; align-content: center; gap: 3px; padding: 14px 20px; border-right: 1px solid var(--line); }
	    .energy-data-inventory div:last-child { border-right: 0; }
	    .energy-data-inventory strong { font-family: var(--font-serif); font-size: 22px; }
	    .energy-data-inventory span { color: var(--muted); font-size: 11.5px; }
	    .energy-data-imports { border-top: 1px solid var(--line); }
	    .energy-data-import { min-height: 64px; display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 18px; align-items: center; padding: 12px 24px; border-bottom: 1px solid var(--line); }
	    .energy-data-import:last-child { border-bottom: 0; }
	    .energy-data-import div { min-width: 0; display: grid; gap: 3px; }
	    .energy-data-import strong { overflow: hidden; font-size: 13px; text-overflow: ellipsis; white-space: nowrap; }
	    .energy-data-import span, .energy-data-import small { color: var(--muted); font-size: 11.5px; }
	    .energy-data-empty { border-top: 1px solid var(--line); padding: 20px 24px; color: var(--muted); font-size: 13px; }
	    .energy-data-danger { border-color: rgba(140,52,52,.2); box-shadow: none; }
	    .energy-data-danger > header { padding: 24px; }
	    .energy-data-danger > header h2 { color: #7f3434; font-size: 24px; }
	    .energy-data-danger > header p { margin-top: 5px; color: var(--muted); font-size: 13px; }
	    .energy-delete-action { border-top: 1px solid rgba(140,52,52,.16); }
	    .energy-delete-action > summary { min-height: 66px; display: flex; justify-content: space-between; gap: 18px; align-items: center; padding: 14px 24px; list-style: none; cursor: pointer; }
	    .energy-delete-action > summary::-webkit-details-marker { display: none; }
	    .energy-delete-action > summary::after { content: "+"; color: #8c3434; font-size: 21px; }
	    .energy-delete-action[open] > summary::after { content: "−"; }
	    .energy-delete-action > summary span { display: grid; gap: 3px; }
	    .energy-delete-action > summary strong { font-size: 14px; }
	    .energy-delete-action > summary small { color: var(--muted); font-size: 12px; font-weight: 500; line-height: 1.4; }
	    .energy-delete-form { display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 12px; align-items: end; border-top: 1px solid rgba(140,52,52,.12); padding: 18px 24px 22px; background: rgba(140,52,52,.025); }
	    .energy-delete-form label { display: grid; gap: 6px; color: var(--muted); font-size: 12px; }
	    .energy-delete-form input { min-height: 44px; border: 1px solid var(--line); border-radius: var(--radius-xs); padding: 10px 12px; color: var(--ink); background: #fff; font: inherit; }
	    .energy-delete-button { min-height: 44px; border: 1px solid rgba(140,52,52,.38); border-radius: var(--radius-xs); padding: 10px 15px; color: #8c3434; background: #fff; font: inherit; font-weight: 800; cursor: pointer; }
	    .energy-delete-button:hover { background: rgba(140,52,52,.07); }
	    .energy-data-note { color: var(--muted); font-size: 12px; line-height: 1.5; }
	    .energy-quality { display: grid; grid-template-columns: auto minmax(0,1fr) auto; gap: 14px; align-items: center; border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 16px; background: #fff; }
    .energy-quality > span:first-child { width: 36px; height: 36px; display: grid; place-items: center; border-radius: 50%; background: #edf4ed; color: #2e6842; font-weight: 900; }
    .energy-quality strong { display: block; }
    .energy-quality p { margin-top: 3px; color: var(--muted); font-size: 13px; }
    .energy-quality small { color: #765f1d; font-weight: 750; }
    .energy-coverage { margin-top: 14px; }
    .energy-coverage-head { display: flex; justify-content: space-between; gap: 12px; align-items: baseline; margin-bottom: 9px; }
    .energy-coverage-head strong { font-size: 13px; }
    .energy-coverage-head span { color: var(--muted); font-size: 12.5px; font-weight: 750; }
    .energy-coverage-list { display: flex; flex-wrap: wrap; gap: 7px; }
    .energy-coverage-row { display: flex; gap: 7px; align-items: baseline; border: 1px solid var(--line); border-radius: var(--radius-pill); padding: 6px 12px; background: #fff; }
    .energy-coverage-row strong { font-size: 12.5px; }
    .energy-coverage-state { color: #765f1d; font-size: 11.5px; font-weight: 850; white-space: nowrap; }
    .energy-coverage-state.good { color: #2e6842; }
    .energy-scenario { display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 14px 18px; align-items: start; border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 16px 18px; background: #fff; }
    .energy-scenario strong { display: block; font-size: 15px; }
    .energy-scenario p { margin-top: 5px; color: var(--muted); font-size: 12.5px; line-height: 1.45; }
    .energy-scenario-assumptions { color: var(--soft); }
    .energy-scenario-result { display: grid; gap: 2px; text-align: right; }
    .energy-scenario-result > span:first-child { color: var(--muted); font-size: 11px; font-weight: 800; letter-spacing: .05em; text-transform: uppercase; }
    .energy-scenario-result strong { font-family: var(--font-serif); font-size: 24px; }
    .energy-scenario-billed { color: var(--muted); font-size: 12px; }
    .energy-scenario-caveat { margin-top: 11px; color: var(--soft); font-size: 12.5px; line-height: 1.5; }
    .energy-scenario-billed b { color: var(--ink); font-weight: 800; }
    .energy-scenario-floor { display: grid; grid-template-columns: auto minmax(0,1fr); gap: 9px; align-items: start; margin-top: 10px; border-radius: var(--radius-sm); padding: 11px 13px; background: rgba(200,153,63,.09); color: var(--muted); font-size: 12.5px; line-height: 1.5; }
    .energy-scenario-floor > span { width: 19px; height: 19px; display: grid; place-items: center; border-radius: 50%; background: var(--gold); color: #2b2410; font-size: 12px; font-weight: 900; }
    .energy-tariff { display: grid; align-content: start; gap: 14px; }
    .energy-tariff > .energy-card-head { margin-bottom: 0; }
    .energy-tariff-basis { display: flex; gap: 8px; align-items: baseline; color: var(--muted); font-size: 12.5px; line-height: 1.45; }
    .energy-tariff-basis > span { color: var(--gold-ink); font-weight: 800; }
    .energy-tariff-cost { display: grid; grid-template-columns: minmax(170px,auto) minmax(0,1fr); gap: 14px 22px; align-items: start; border-top: 1px solid var(--line); border-bottom: 1px solid var(--line); padding: 16px 0; }
    .energy-tariff-cost-figure { display: grid; align-content: start; gap: 3px; }
    .energy-tariff-cost-figure span { color: var(--muted); font-size: 11.5px; font-weight: 800; letter-spacing: .05em; text-transform: uppercase; }
    .energy-tariff-cost-figure strong { font-family: var(--font-serif); font-size: 34px; font-weight: 600; line-height: 1.05; }
    .energy-tariff-cost-figure small { color: var(--muted); font-size: 12px; }
    .energy-tariff-cost-copy { display: grid; gap: 7px; }
    .energy-tariff-cost-copy p { color: var(--muted); font-size: 13px; line-height: 1.5; }
    .energy-tariff-cost-copy strong { color: var(--ink); font-weight: 800; }
    .energy-tariff-notes { list-style: none; margin: 0; padding: 0; display: grid; gap: 6px; color: var(--muted); font-size: 12.5px; line-height: 1.5; }
    .energy-tariff-notes li { padding-left: 15px; text-indent: -15px; }
    .energy-tariff-notes li::before { content: "– "; color: var(--gold-ink); }
    .energy-tariff-empty { display: grid; gap: 8px; justify-items: start; border: 1px dashed rgba(200,153,63,.55); border-radius: var(--radius-sm); padding: 18px; background: rgba(200,153,63,.06); }
    .energy-tariff-empty strong { font-size: 16px; }
    .energy-tariff-empty p { color: var(--muted); font-size: 13.5px; line-height: 1.5; }
    .energy-tariff-empty-rule { color: var(--soft); font-size: 12.5px; }
    .energy-tariff-settings > summary { min-height: 44px; display: flex; gap: 10px; align-items: center; color: #765f1d; font-size: 13px; font-weight: 800; cursor: pointer; list-style: none; }
    .energy-tariff-settings > summary::-webkit-details-marker { display: none; }
    .energy-tariff-settings > summary::before { content: "+"; width: 20px; height: 20px; display: grid; place-items: center; border: 1px solid var(--line); border-radius: 50%; background: #fff; font-size: 14px; }
    .energy-tariff-settings[open] > summary::before { content: "−"; }
    .energy-tariff-settings > summary span { color: var(--muted); font-weight: 600; }
    .energy-tariff-forms { display: grid; grid-template-columns: repeat(auto-fit,minmax(240px,1fr)); gap: 14px; margin-top: 4px; }
    .energy-tariff-foot { display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 4px 18px; align-items: center; border-top: 1px solid var(--line); padding-top: 13px; color: var(--soft); font-size: 12px; }
    .energy-tariff-foot a { grid-column: 1; color: #765f1d; font-weight: 750; }
    .energy-tariff-foot form { grid-column: 2; grid-row: 1 / span 2; }
    .energy-target-form { display: grid; align-content: start; gap: 10px; border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 16px; background: #fff; }
    .energy-consumers { margin-top: 16px; display: grid; gap: 12px; }
    .energy-consumer-list { list-style: none; margin: 0; padding: 0; display: grid; gap: 8px; }
    .energy-consumer-list li { display: flex; align-items: center; justify-content: space-between; gap: 14px; border: 1px solid var(--line); border-radius: var(--radius-xs); padding: 10px 14px; background: #fff; }
    .energy-consumer-list small { display: block; color: var(--muted); font-size: 13px; }
    .energy-consumer-form { display: grid; gap: 12px; margin-top: 12px; grid-template-columns: repeat(auto-fit,minmax(190px,1fr)); align-items: end; }
    .energy-consumer-form label { display: grid; gap: 6px; font-size: 13px; font-weight: 800; }
    .energy-consumer-form input, .energy-consumer-form select { min-height: 44px; border: 1px solid var(--line); border-radius: var(--radius-xs); padding: 8px 10px; }
    .energy-billed { display: grid; grid-template-columns: repeat(auto-fit,minmax(min(146px,100%),1fr)); gap: 1px; margin: 0; border: 1px solid var(--line); border-radius: var(--radius-sm); overflow: hidden; background: var(--line); }
    .energy-billed > div { display: grid; align-content: start; gap: 3px; padding: 13px 14px; background: #fff; }
    .energy-billed span { min-height: 26px; color: var(--muted); font-size: 11px; font-weight: 800; letter-spacing: .05em; line-height: 1.2; text-transform: uppercase; }
    .energy-billed strong { font-family: var(--font-serif); font-size: 28px; font-weight: 600; line-height: 1.05; }
    .energy-billed small { color: var(--muted); font-size: 11.5px; line-height: 1.35; }
    .energy-billed-above { background: rgba(200,153,63,.09); }
    .energy-mode-strip :where(a,button,input,select,textarea,summary,[tabindex]):focus-visible,
    .energy-page :where(a,button,input,select,textarea,summary,[tabindex]):focus-visible,
    .onboarding-page :where(a,button,input,select,textarea,summary,[tabindex]):focus-visible,
    .energy-data-page :where(a,button,input,select,textarea,summary,[tabindex]):focus-visible,
    .home-identity-page :where(a,button,input,select,textarea,summary,[tabindex]):focus-visible { outline: 3px solid var(--energy-focus-ring); outline-offset: 3px; }
    .energy-target-form input { width: 100%; min-height: 46px; border: 1px solid var(--line); border-radius: var(--radius-xs); padding: 10px 12px; }
    .energy-history { display: grid; gap: 8px; margin-top: 16px; }
    .energy-history-row { display: grid; grid-template-columns: minmax(0,1fr) auto auto; gap: 12px; align-items: center; border-top: 1px solid var(--line); padding-top: 12px; font-size: 13px; }
    .energy-history-row strong { display: block; }
    .energy-history-row span, .energy-history-row small { color: var(--muted); }
    .energy-maintenance-list, .energy-measure-list { display: grid; gap: 10px; }
    .energy-maintenance-row, .energy-measure-row { border: 1px solid var(--line); border-radius: var(--radius-sm); background: #fff; }
    .energy-maintenance-summary, .energy-measure-summary { min-height: 64px; display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 14px; align-items: center; padding: 14px 16px; list-style: none; cursor: pointer; }
    .energy-maintenance-summary::-webkit-details-marker, .energy-measure-summary::-webkit-details-marker { display: none; }
    .energy-maintenance-summary > span:first-child, .energy-measure-summary > span:first-child { display: grid; gap: 3px; }
    .energy-maintenance-summary small, .energy-measure-summary small { color: var(--muted); }
    .energy-maintenance-summary::after, .energy-measure-summary::after { content: "›"; color: #876d23; font-size: 24px; transform: rotate(90deg); }
    details[open] > .energy-maintenance-summary::after, details[open] > .energy-measure-summary::after { transform: rotate(-90deg); }
    .energy-maintenance-body, .energy-measure-body { display: grid; gap: 14px; border-top: 1px solid var(--line); padding: 16px; }
    .energy-inline-form { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 12px; }
    .energy-inline-form label { display: grid; gap: 5px; color: var(--muted); font-size: 12px; font-weight: 750; }
    .energy-inline-form input, .energy-inline-form select, .energy-inline-form textarea { width: 100%; min-height: 44px; border: 1px solid var(--line); border-radius: var(--radius-xs); padding: 9px 11px; background: #fff; color: var(--ink); }
    .energy-inline-form textarea { min-height: 82px; resize: vertical; }
    .energy-inline-form .wide, .energy-inline-form .actions { grid-column: 1 / -1; }
    .energy-inline-form .actions { display: flex; flex-wrap: wrap; gap: 8px; }
    .energy-compact-create { margin-top: 14px; border: 1px dashed #c6aa68; border-radius: var(--radius-sm); background: #fbf7eb; }
    .energy-compact-create > summary { min-height: 52px; display: flex; align-items: center; justify-content: space-between; padding: 12px 15px; list-style: none; cursor: pointer; font-weight: 800; }
    .energy-compact-create > summary::-webkit-details-marker { display: none; }
    .energy-compact-create > summary::after { content: "+"; font-size: 20px; }
    .energy-compact-create[open] > summary::after { content: "−"; }
    .energy-compact-create > form { border-top: 1px solid var(--line); padding: 15px; }
    .energy-comparison { display: grid; grid-template-columns: 1fr auto 1fr; gap: 14px; align-items: center; border-radius: var(--radius-xs); padding: 13px; background: #f4f6f1; }
    .energy-comparison > span { color: var(--muted); font-size: 12px; text-align: center; }
    .energy-comparison strong { display: block; font-family: var(--font-serif); font-size: 22px; }
    .energy-comparison small { color: var(--muted); }
    .energy-status-danger { color: #9c3a31; }
    .energy-status-warning { color: #876517; }
    .onboarding-page { width: min(940px,100%); padding-top: 28px; gap: 18px; }
    .onboarding-progress { display: grid; gap: 8px; }
    .onboarding-progress-head { display: flex; justify-content: space-between; gap: 12px; color: var(--muted); font-size: 12px; font-weight: 750; }
    .onboarding-progress-track { height: 5px; overflow: hidden; border-radius: var(--radius-pill); background: #e8e1d4; }
    .onboarding-progress-track span { display: block; height: 100%; background: var(--gold); }
    .onboarding-card { border: 1px solid var(--line); border-radius: var(--radius-lg); background: var(--surface); box-shadow: var(--shadow-md); overflow: hidden; }
    .onboarding-card-head { display: grid; gap: 9px; padding: clamp(24px,4vw,38px); border-bottom: 1px solid var(--line); }
    .onboarding-card-head h1 { font-size: clamp(34px,5vw,50px); }
    .onboarding-card-head .onboarding-home-unit { margin-top: -4px; color: var(--muted); font-size: 13px; font-weight: 650; line-height: 1.35; }
    .onboarding-card-head p { max-width: 660px; color: var(--muted); line-height: 1.55; }
    .onboarding-body { display: grid; gap: 22px; padding: clamp(24px,4vw,38px); }
    .onboarding-explain { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 12px; }
    .onboarding-explain article { min-height: 142px; border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 18px; background: #fff; }
    .onboarding-explain strong { display: block; margin-bottom: 8px; font-size: 15px; }
    .onboarding-explain p { color: var(--muted); font-size: 13px; line-height: 1.45; }
    .onboarding-form { display: grid; gap: 18px; }
    .onboarding-form label > span, .onboarding-legend { display: block; margin-bottom: 7px; color: #75652d; font-size: 11px; font-weight: 850; letter-spacing: .09em; text-transform: uppercase; }
    .onboarding-form input[type="text"], .onboarding-form select { width: 100%; min-height: 48px; border: 1px solid var(--line); border-radius: var(--radius-xs); padding: 10px 12px; color: var(--ink); background: #fff; }
    .home-type-explanation { display: grid; grid-template-columns: 36px minmax(0,1fr); gap: 12px; align-items: start; margin-top: -6px; border-radius: var(--radius-sm); padding: 13px 14px; color: var(--ink); background: var(--panel-soft); }
    .home-type-explanation-icon { width: 34px; height: 34px; display: grid; place-items: center; border-radius: 50%; color: var(--leaf); background: rgba(47,107,74,.11); font-size: 18px; line-height: 1; }
    .home-type-explanation strong { display: block; font-size: 14px; line-height: 1.3; }
    .home-type-explanation p { margin: 3px 0 0; color: var(--muted); font-size: 13.5px; line-height: 1.45; text-wrap: pretty; }
    .home-type-explanation small { display: block; margin-top: 6px; color: #77786f; font-size: 11.5px; line-height: 1.4; }
    .onboarding-form .home-identity-field > span, .onboarding-form .home-identity-field > label > span { display: block; margin-bottom: 7px; color: #75652d; font-size: 11px; font-weight: 850; letter-spacing: .09em; text-transform: uppercase; }
    .onboarding-form .home-identity-readonly { min-height: 50px; display: flex; align-items: center; justify-content: space-between; gap: 12px; border: 1px solid var(--line); border-radius: var(--radius-xs); padding: 12px 14px; background: var(--panel-soft); }
    .onboarding-form .home-identity-readonly small { color: var(--muted); font-size: 12px; }
    .onboarding-form .home-identity-field-help { display: block; margin-top: 6px; color: var(--muted); font-size: 12px; font-weight: 600; line-height: 1.45; letter-spacing: normal; text-transform: none; }
    .onboarding-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }
    .onboarding-choice-grid { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 10px; }
    .onboarding-choice { min-height: 64px; display: grid; grid-template-columns: auto minmax(0,1fr); gap: 10px; align-items: center; border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 12px 14px; background: #fff; cursor: pointer; }
    .onboarding-choice:has(input:checked) { border-color: #9e8740; background: #faf5e8; box-shadow: inset 0 0 0 1px #c7a953; }
    .onboarding-choice small { display: block; margin-top: 2px; color: var(--muted); font-size: 11px; }
    .onboarding-recommended { display: grid; gap: 10px; border: 0; padding: 0; }
    .onboarding-recommended .onboarding-legend { margin-bottom: 0; }
    .onboarding-candidates { display: grid; gap: 8px; }
    .onboarding-candidate { display: grid; grid-template-columns: auto minmax(0,1fr) auto; gap: 12px; align-items: center; border: 1px solid var(--line); border-radius: var(--radius-xs); padding: 12px 14px; background: #fff; }
    .onboarding-candidate:has(input:checked) { border-color: #9ebda5; background: #f5faf5; }
    .onboarding-candidate strong { display: block; font-size: 14px; }
    .onboarding-candidate small { display: block; margin-top: 2px; color: var(--muted); font-size: 11px; }
    .onboarding-candidate-value { font-weight: 800; white-space: nowrap; }
    .onboarding-readonly { border-radius: var(--radius-pill); padding: 5px 9px; color: #2f6b4a; background: #e7f1e8; font-size: 11px; font-weight: 800; white-space: nowrap; }
    .energy-mapping-guide { display: grid; gap: 12px; }
    .energy-mapping-guide > header { display: flex; justify-content: space-between; gap: 18px; align-items: end; }
    .energy-mapping-guide > header h2 { font-size: 25px; }
    .energy-mapping-guide > header small { color: var(--muted); font-size: 12px; }
    .energy-mapping-slots { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 8px; }
    .energy-mapping-slot { min-width: 0; display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 14px; align-items: center; border: 1px solid var(--line); border-radius: var(--radius-xs); padding: 13px 14px; background: #fff; }
    .energy-mapping-slot > div { min-width: 0; }
    .energy-mapping-slot strong, .energy-mapping-slot span, .energy-mapping-slot small { display: block; }
    .energy-mapping-slot span { margin-top: 3px; color: var(--muted); font-size: 11.5px; line-height: 1.35; }
    .energy-mapping-slot p { min-width: 118px; text-align: right; }
    .energy-mapping-slot b { color: var(--muted); font-size: 12px; }
    .energy-mapping-slot small { margin-top: 3px; max-width: 190px; color: var(--soft); font-size: 10.5px; line-height: 1.3; }
    .energy-mapping-slot.good b { color: var(--leaf); }
    .energy-mapping-slot.warning { border-color: rgba(200,153,63,.5); background: #fffdf7; }
    .energy-mapping-slot.warning b { color: #8a681c; }
    .energy-mapping-slot.quiet { background: var(--panel-soft); }
    .onboarding-disclosure { border: 1px solid var(--line); border-radius: var(--radius-sm); background: #fbf8f1; overflow: hidden; }
    .onboarding-disclosure > summary { min-height: 48px; display: flex; align-items: center; gap: 9px; padding: 11px 14px; color: var(--ink); cursor: pointer; list-style: none; font-size: 13px; font-weight: 750; }
    .onboarding-disclosure > summary::-webkit-details-marker { display: none; }
    .onboarding-disclosure > summary::after { content: "+"; margin-left: auto; color: var(--gold-ink); font-size: 18px; font-weight: 500; }
    .onboarding-disclosure[open] > summary::after { content: "−"; }
    .onboarding-disclosure > summary span { color: var(--muted); font-size: 11px; font-weight: 650; }
    .onboarding-disclosure .onboarding-candidates, .onboarding-disclosure .optional-grid { display: grid; gap: 8px; border-top: 1px solid var(--line); padding: 12px; }
    .onboarding-disclosure .optional-grid { grid-template-columns: repeat(2,minmax(0,1fr)); }
    .onboarding-additional .onboarding-candidate { background: #fff; }
    .onboarding-additional code { display: block; margin-top: 3px; overflow-wrap: anywhere; color: var(--muted); font-size: 10px; }
    .onboarding-skip { margin-left: auto; }
    .onboarding-actions { display: flex; justify-content: space-between; gap: 12px; align-items: center; padding-top: 4px; }
    .onboarding-actions .button { min-height: 48px; }
    .onboarding-trust { display: grid; grid-template-columns: auto minmax(0,1fr); gap: 11px; align-items: start; border: 1px solid #b9d0bd; border-radius: var(--radius-sm); padding: 15px; background: #f1f7f1; }
    .onboarding-trust strong { display: block; margin-bottom: 3px; }
    .onboarding-trust p { color: #53675a; font-size: 13px; line-height: 1.4; }
    .content-top { height: 64px; display: flex; align-items: center; justify-content: space-between; gap: 14px; padding: 0 clamp(28px,4vw,44px); border-bottom: 1px solid var(--line); background: rgba(255,254,251,.72); }
    .crumb { display: inline-flex; align-items: center; gap: 10px; color: var(--muted); font-size: 14px; }
    .crumb svg, .action svg { width: 18px; height: 18px; stroke: currentColor; fill: none; stroke-width: 1.9; stroke-linecap: round; stroke-linejoin: round; }
    .page-actions { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; min-width: 0; }
    .page-actions form { min-width: 0; }
    .page { width: min(1220px,100%); margin: 0 auto; padding: 34px clamp(28px,4vw,44px) 0; display: grid; gap: 24px; }
    .page.wide { width: min(1280px,100%); }
    h1 { margin: 0; font-family: var(--font-serif); font-weight: 500; font-size: clamp(42px,5vw,54px); line-height: 1; }
    h2 { margin: 0; font-family: var(--font-serif); font-weight: 600; font-size: 23px; line-height: 1.1; }
    h3 { margin: 0; font-family: var(--font-serif); font-weight: 600; font-size: 20px; line-height: 1.2; }
    p { margin: 0; }
    .lede { margin-top: 14px; color: var(--muted); font-size: 16px; line-height: 1.55; }
    .muted { color: var(--muted); line-height: 1.5; }
    .subtle-note { margin-top: 6px; max-width: 760px; font-size: 14.5px; }
    .kicker { color: var(--gold-ink); font-size: 12px; font-weight: 800; letter-spacing: .12em; text-transform: uppercase; border-bottom: 2px solid var(--ink); padding-bottom: 11px; margin-bottom: 20px; }
    /* Shared components: panel, button, pill, quick-row, table-wrap, dialog, flash and empty-state. */
    .panel { background: var(--panel); border: 1px solid var(--line); border-radius: var(--radius-sm); padding: var(--space-6); box-shadow: var(--shadow-panel); }
    .panel.compact { padding: 18px; }
    .button, button.action { min-height: 38px; display: inline-flex; align-items: center; justify-content: center; gap: var(--space-2); border: 1px solid var(--line); background: var(--panel); border-radius: var(--radius-xs); color: var(--ink); padding: 8px 13px; font-weight: 700; line-height: 1.15; text-decoration: none; cursor: pointer; white-space: nowrap; }
    .button:hover, button.action:hover { border-color: var(--gold); }
    button:disabled, input[type=submit]:disabled, .button[aria-disabled="true"] { cursor: progress; opacity: .62; }
    form[aria-busy="true"] button[type=submit] { pointer-events: none; }
    .button.primary, button.primary { background: var(--ink); border-color: var(--ink); color: #fff; }
    .button.small, button.small { min-height: 31px; padding: 6px 10px; font-size: 12px; }
    .button.ghost { background: transparent; }
    .guide-disclosure > summary { min-height: 44px; display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 12px; align-items: center; cursor: pointer; list-style: none; }
    .guide-disclosure > summary::-webkit-details-marker { display: none; }
    .guide-disclosure > summary::after { content: "\203A"; color: var(--gold-ink); font-size: 22px; line-height: 1; transition: transform .16s ease; }
    .guide-disclosure[open] > summary::after { transform: rotate(90deg); }
    .guide-disclosure[open] > summary { padding-bottom: 12px; border-bottom: 1px solid var(--line); }
    .guide-disclosure > :not(summary) { margin-top: 14px; }
    @media (prefers-reduced-motion: reduce) { .guide-disclosure > summary::after { transition: none; } }
    .issue-card, .entry, .event-card, .document-row, .vote-card, tr[id^="parking-month-"] { scroll-margin-top: 82px; }
    .home-hero { position: relative; min-height: 118px; display: flex; align-items: center; overflow: hidden; border-bottom: 1px solid var(--line); background: #f7f3ea; padding: 25px clamp(28px,4vw,58px); isolation: isolate; }
    .home-hero::before { content: ""; position: absolute; z-index: -2; inset: 0; background: url('{{.Tenant.HeroImageURL}}') center 47% / cover no-repeat; filter: saturate(.72); opacity: .46; }
    .home-hero::after { content: ""; position: absolute; z-index: -1; inset: 0; background: linear-gradient(90deg, rgba(247,243,234,.99) 0%, rgba(247,243,234,.94) 38%, rgba(247,243,234,.58) 72%, rgba(247,243,234,.34) 100%); }
    .home-hero-copy { width: min(720px,100%); }
    .home-hero h1 { font-size: clamp(34px,4vw,44px); }
    .home-hero p { margin-top: 8px; color: var(--muted); font-size: 15px; line-height: 1.4; }
    /* Hausüberblick: daily focus, house board, energy signal and area tiles. */
    .home-page { width: min(1140px,100%); padding-top: 28px; padding-bottom: 10px; gap: 34px; }
    .home-focus, .home-follow, .home-utilities { display: grid; gap: 16px; }
    .home-eyebrow { color: var(--gold-ink); font-size: 11px; font-weight: 850; letter-spacing: .12em; text-transform: uppercase; }
    .portal-section { min-width: 0; display: grid; gap: 14px; }
    .portal-section-head { min-width: 0; display: grid; gap: 6px; align-items: end; border-bottom: 1px solid var(--line); padding-bottom: 11px; }
    .portal-section-head.has-action { grid-template-columns: minmax(0,1fr) auto; column-gap: 6px; }
    .portal-section-head h2 { margin-top: 4px; font-size: 22px; }
    .portal-quiet-action { min-height: 44px; display: inline-flex; align-items: center; justify-content: center; gap: 7px; align-self: end; border: 1px solid rgba(32,37,31,.14); border-radius: var(--radius-pill); padding: 7px 12px 7px 8px; color: var(--muted); background: rgba(255,254,251,.46); font-size: 11px; font-weight: 800; text-decoration: none; white-space: nowrap; }
    .portal-quiet-action > span { width: 22px; height: 22px; display: grid; place-items: center; border-radius: 50%; color: var(--gold-ink); background: rgba(200,153,63,.13); font-size: 18px; line-height: 1; }
    .portal-quiet-action:hover, .portal-quiet-action:focus-visible { border-color: rgba(200,153,63,.52); color: var(--ink); background: var(--panel); }
    @media (max-width: 430px) {
      .portal-section-head.has-action { grid-template-columns: minmax(0,1fr); }
      .portal-quiet-action { justify-self: end; }
    }
    .home-primary-task { min-width: 0; display: grid; grid-template-columns: 48px minmax(0,1fr) auto; gap: 18px; align-items: center; border: 1px solid rgba(47,107,74,.34); border-radius: var(--radius-sm); padding: 22px 24px; background: linear-gradient(135deg, rgba(47,107,74,.09), rgba(255,254,251,.98) 62%); box-shadow: var(--shadow-panel); }
    .home-task-number { width: 42px; height: 42px; display: grid; place-items: center; border-radius: 50%; color: #fff; background: var(--ink); font-family: var(--font-serif); font-size: 22px; font-weight: 700; }
    .home-task-copy { min-width: 0; }
    .home-task-copy > span { color: var(--leaf); font-size: 11px; font-weight: 850; letter-spacing: .1em; text-transform: uppercase; }
    .home-task-copy h3 { margin-top: 4px; font-size: 24px; overflow-wrap: anywhere; }
    .home-task-copy p { margin-top: 6px; color: var(--muted); font-size: 14px; line-height: 1.45; overflow-wrap: anywhere; }
    .home-task-action { min-height: 44px; padding-inline: 17px; }
    .home-calm { display: grid; grid-template-columns: 46px minmax(0,1fr); gap: 16px; align-items: center; border: 1px solid rgba(47,107,74,.24); border-radius: var(--radius-sm); padding: 22px 24px; color: var(--leaf); background: linear-gradient(135deg, rgba(47,107,74,.1), rgba(255,254,251,.96) 68%); box-shadow: var(--shadow-panel); }
    .home-calm-icon { width: 46px; height: 46px; display: grid; place-items: center; border-radius: 50%; color: #fff; background: var(--leaf); font-size: 21px; font-weight: 900; }
    .home-calm-copy strong { display: block; font-family: var(--font-serif); font-size: 22px; }
    .home-calm-copy > span { display: block; margin-top: 5px; max-width: 62ch; color: var(--muted); font-size: 14px; line-height: 1.45; }
    .home-follow { gap: 10px; }
    .home-follow h3 { color: var(--soft); font-family: var(--font-sans); font-size: 11px; font-weight: 850; letter-spacing: .1em; text-transform: uppercase; }
    .home-follow-list { overflow: hidden; border-top: 1px solid var(--line); border-bottom: 1px solid var(--line); }
    .home-follow-row { min-width: 0; display: grid; grid-template-columns: 120px minmax(0,1fr) auto; gap: 18px; align-items: center; padding: 15px 2px; color: inherit; text-decoration: none; border-bottom: 1px solid var(--line); }
    .home-follow-row:last-child { border-bottom: 0; }
    .home-follow-row:hover strong, .home-follow-row:hover .home-follow-action { color: var(--gold-ink); }
    .home-follow-kind { color: var(--soft); font-size: 11px; font-weight: 850; letter-spacing: .08em; text-transform: uppercase; }
    .home-follow-copy { min-width: 0; }
    .home-follow-copy strong { display: block; font-size: 15px; line-height: 1.3; overflow-wrap: anywhere; }
    .home-follow-copy > span { display: block; margin-top: 3px; color: var(--muted); font-size: 13px; line-height: 1.4; overflow-wrap: anywhere; }
    .home-follow-action { color: var(--ink); font-size: 13px; font-weight: 750; white-space: nowrap; }
    /* Die Karten sind unterschiedlich voll. Sie wachsen mit ihrem Inhalt, statt
       auf gleiche Höhe gezogen zu werden und über der Fußzeile eine Lücke zu
       lassen. */
    .portal-board-grid { display: grid; grid-template-columns: repeat(auto-fit,minmax(268px,1fr)); gap: 14px; align-items: start; }
    .portal-card { min-width: 0; display: flex; flex-direction: column; gap: 12px; border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 17px 18px 13px; background: var(--panel); box-shadow: var(--shadow-panel); }
    .portal-card-head { min-width: 0; display: flex; align-items: center; justify-content: space-between; gap: 10px; }
    .portal-card-head h3 { font-size: 17px; }
    .portal-card-tag { color: var(--soft); font-size: 11.5px; font-weight: 750; white-space: nowrap; }
    .portal-list { min-width: 0; margin: 0; padding: 0; list-style: none; display: grid; }
    .portal-list li { min-width: 0; border-top: 1px solid var(--line); }
    .portal-list li:first-child { border-top: 0; }
    .portal-row { min-width: 0; min-height: 44px; display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 4px 10px; align-items: center; padding: 11px 0; color: inherit; text-decoration: none; }
    .portal-row.has-date { grid-template-columns: 40px minmax(0,1fr); }
    .portal-row:hover strong { color: var(--gold-ink); }
    .portal-row-copy { min-width: 0; display: grid; gap: 2px; }
    .portal-row-copy strong { display: block; font-size: 14.5px; font-weight: 700; line-height: 1.28; overflow-wrap: anywhere; }
    .portal-row-copy span { display: -webkit-box; overflow: hidden; color: var(--muted); font-size: 12.5px; line-height: 1.35; overflow-wrap: anywhere; -webkit-line-clamp: 2; -webkit-box-orient: vertical; }
    .portal-date { width: 40px; display: grid; justify-items: center; gap: 0; border-radius: var(--radius-xs); padding: 4px 0 5px; background: var(--panel-soft); border: 1px solid var(--line); }
    .portal-date strong { font-family: var(--font-serif); font-size: 17px; line-height: 1; }
    .portal-date span { color: var(--gold-ink); font-size: 9.5px; font-weight: 850; letter-spacing: .06em; text-transform: uppercase; }
    .portal-flag { width: 9px; height: 9px; border-radius: 50%; background: var(--gold); }
    .portal-card-blank { flex: 1 1 auto; display: grid; gap: 6px; align-content: start; border-radius: var(--radius-xs); padding: 13px 14px; background: var(--panel-soft); }
    .portal-card-blank strong { font-size: 13.5px; }
    .portal-card-blank span { color: var(--muted); font-size: 12.5px; line-height: 1.4; }
    .portal-card-action { margin-top: auto; min-height: 42px; display: inline-flex; align-items: center; gap: 7px; border-top: 1px solid var(--line); padding-top: 11px; color: var(--ink); font-size: 13px; font-weight: 750; text-decoration: none; }
    .portal-card-action:hover { color: var(--gold-ink); }
    .portal-energy { min-width: 0; display: grid; grid-template-columns: minmax(0,1.05fr) minmax(0,1.35fr) auto; gap: 18px 26px; align-items: center; border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 20px 22px; color: rgba(255,255,255,.9); background: linear-gradient(120deg, #17261d 0%, #20291f 62%, #2a3327 100%); border-color: rgba(255,255,255,.12); box-shadow: var(--shadow-panel); }
    .portal-energy.setup { grid-template-columns: minmax(0,1fr) auto; }
    .portal-energy-copy { min-width: 0; display: grid; gap: 6px; align-content: start; }
    .portal-energy-kicker { color: var(--gold-light); font-size: 11px; font-weight: 850; letter-spacing: .12em; text-transform: uppercase; }
    .portal-energy-copy h3 { color: #fff; font-size: 21px; overflow-wrap: anywhere; }
    .portal-energy-mode { justify-self: start; display: inline-flex; align-items: center; gap: 8px; border-radius: var(--radius-pill); padding: 3px 11px; background: rgba(255,255,255,.1); color: rgba(255,255,255,.86); font-size: 12px; font-weight: 750; }
    .portal-energy-mode::before { content: ""; width: 8px; height: 8px; border-radius: 50%; background: var(--gold-light); }
    .portal-energy-mode.active::before { background: #e9b65a; }
    .portal-energy-message { max-width: 60ch; color: rgba(255,255,255,.68); font-size: 13px; line-height: 1.45; }
    .portal-energy-stats { min-width: 0; margin: 0; display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 14px; }
    .portal-energy-stats > div { min-width: 0; display: grid; align-content: end; border-left: 1px solid rgba(255,255,255,.14); padding-left: 14px; }
    .portal-energy-stats dt { color: rgba(255,255,255,.56); font-size: 11px; font-weight: 700; line-height: 1.3; }
    .portal-energy-stats dd { margin: 4px 0 0; color: #fff; font-family: var(--font-serif); font-size: 23px; line-height: 1.1; overflow-wrap: anywhere; }
    .portal-energy-stats dd.pending { color: rgba(255,255,255,.6); font-family: var(--font-sans); font-size: 13.5px; font-weight: 650; }
    .portal-energy-foot { min-width: 0; display: grid; gap: 9px; justify-items: start; }
    .portal-energy-foot .button { min-height: 42px; border-color: var(--gold-light); background: var(--gold-light); color: #17261d; }
    .portal-energy-foot .button:hover { border-color: #fff; background: #fff; }
    .portal-energy-foot p { max-width: 30ch; color: rgba(255,255,255,.5); font-size: 11.5px; line-height: 1.4; }
    .home-utilities { display: grid; gap: 14px; }
    .home-utilities h2 { font-size: 22px; }
    .home-utility-links { display: grid; grid-template-columns: repeat(auto-fit,minmax(226px,1fr)); gap: 10px; }
    .home-utility-links a { min-width: 0; min-height: 42px; display: grid; grid-template-columns: 30px minmax(0,1fr); gap: 12px; align-items: start; border: 1px solid var(--line); border-radius: var(--radius-xs); padding: 13px 14px; color: inherit; background: rgba(255,254,251,.68); text-decoration: none; }
    .home-utility-links a:hover { border-color: var(--gold); background: var(--panel); }
    .home-utility-links a:hover strong { color: var(--gold-ink); }
    .portal-area-icon { width: 30px; height: 30px; display: grid; place-items: center; border-radius: var(--radius-xs); background: var(--panel-soft); color: var(--gold-ink); }
    .portal-area-icon svg { width: 19px; height: 19px; stroke: currentColor; stroke-width: 1.9; fill: none; stroke-linecap: round; stroke-linejoin: round; }
    .portal-area-copy { min-width: 0; display: grid; gap: 3px; justify-items: start; }
    .home-utility-links strong { min-width: 0; font-size: 13.5px; overflow-wrap: anywhere; }
    .home-utility-links .portal-area-copy > span { min-width: 0; color: var(--muted); font-size: 12px; line-height: 1.35; overflow-wrap: anywhere; }
    .home-utility-links .portal-area-copy > em { display: inline-flex; align-items: center; margin-top: 3px; border-radius: var(--radius-pill); padding: 2px 9px; background: rgba(200,153,63,.16); color: #8a6a1f; font-size: 11px; font-style: normal; font-weight: 800; }
    .section-link { display: inline-flex; align-items: center; gap: 8px; color: var(--ink); text-decoration: none; font-size: 13px; font-weight: 700; white-space: nowrap; }
    .section-link::after { content: "›"; color: var(--gold-ink); font-size: 21px; line-height: 1; }
    .section-link:hover { color: var(--gold-ink); }
    .entries { display: grid; gap: 22px; }
    .entry + .entry { border-top: 1px solid var(--line); padding-top: 22px; }
    .entry-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 14px; }
    .entry h3 a { color: inherit; text-decoration: none; }
    .entry h3 a:hover { color: var(--gold-ink); }
    .entry p, .entry-body { margin-top: 8px; color: #5c5f54; line-height: 1.6; text-wrap: pretty; }
    .entry-meta { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; margin-top: 10px; color: var(--soft); font-size: 12.5px; }
    .entry-actions { display: flex; gap: 8px; flex-wrap: wrap; justify-content: flex-end; }
    .entry-actions form { margin: 0; }
    .announcement-entry { position: relative; border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 16px; background: var(--panel); }
    .announcement-entry + .announcement-entry { padding-top: 16px; }
    .announcement-entry.is-unread { border-color: rgba(47,107,74,.34); box-shadow: inset 4px 0 0 var(--leaf); }
    .announcement-entry.is-pinned:not(.is-unread) { box-shadow: inset 4px 0 0 var(--gold); }
    .announcement-body { margin-top: 12px; border-top: 1px solid var(--line); padding-top: 10px; }
    .announcement-body > summary { cursor: pointer; color: var(--gold-ink); font-size: 13px; font-weight: 850; }
    .announcement-body-content { display: grid; gap: 12px; padding-top: 4px; }
    .archive-tools { display: grid; gap: 12px; margin: -4px 0 20px; }
    .filter-form { display: grid; grid-template-columns: minmax(220px,1fr) auto; gap: 10px; align-items: end; }
    .filter-form.audit-filter { grid-template-columns: minmax(180px,.55fr) minmax(240px,1fr) auto; }
    .filter-form label { margin: 0; }
    .audit-page { display: grid; gap: 18px; }
    .audit-page-head { display: grid; gap: 5px; max-width: 720px; }
    .audit-stream { min-width: 0; display: grid; gap: 14px; align-content: start; }
    .audit-filter-panel { border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel); box-shadow: var(--shadow-panel); overflow: hidden; }
    .audit-filter-panel > summary { list-style: none; cursor: pointer; min-height: 54px; display: flex; justify-content: space-between; gap: 14px; align-items: center; padding: 10px 13px 10px 16px; color: var(--muted); }
    .audit-filter-panel > summary::-webkit-details-marker { display: none; }
    .audit-filter-panel > summary:hover, .audit-filter-panel > summary:focus-visible { background: var(--panel-soft); outline: 2px solid var(--gold); outline-offset: -2px; }
    .audit-overview { min-width: 0; display: flex; align-items: center; gap: 8px; color: var(--ink); font-size: 13.5px; }
    .audit-overview svg { width: 18px; height: 18px; flex: 0 0 auto; fill: none; stroke: var(--gold-ink); stroke-width: 1.8; }
    .audit-overview strong { font-size: 14px; }
    .audit-overview span { color: var(--muted); }
    .audit-filter-trigger { min-height: 36px; display: inline-flex; align-items: center; gap: 7px; border: 1px solid var(--line); border-radius: var(--radius-xs); padding: 6px 10px; background: #fffefb; color: var(--ink); font-size: 13px; font-weight: 850; white-space: nowrap; }
    .audit-filter-trigger svg { width: 17px; height: 17px; fill: none; stroke: currentColor; stroke-width: 1.8; }
    .audit-filter-trigger .chip { min-height: 20px; padding: 2px 7px; }
    .audit-filter-content { display: grid; gap: 12px; border-top: 1px solid var(--line); padding: 14px 16px 16px; background: var(--panel-soft); }
    .audit-filter-content-head { display: flex; justify-content: space-between; gap: 12px; align-items: baseline; }
    .audit-filter-content-head strong { font-family: var(--font-serif); font-size: 18px; }
    .audit-filter-content-head span { color: var(--muted); font-size: 12px; }
    .audit-filter-actions { display: flex; gap: 8px; align-items: center; }
    .audit-filter-actions .button { min-height: 42px; }
    .audit-active-filters { display: flex; gap: 8px; flex-wrap: wrap; }
    .audit-timeline { --audit-cols: 62px 14px minmax(0,1.35fr) minmax(0,.92fr) minmax(0,1.05fr) 18px; display: grid; gap: 0; border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel); box-shadow: var(--shadow-panel); overflow: hidden; }
    .audit-columns { display: grid; grid-template-columns: var(--audit-cols); gap: 12px; padding: 9px 16px; border-bottom: 1px solid var(--line); background: rgba(251,248,240,.9); color: var(--soft); font-size: 10.5px; font-weight: 850; letter-spacing: .09em; text-transform: uppercase; }
    .audit-columns span:first-child { text-align: right; }
    .audit-day { margin: 0; padding: 12px 16px 9px; color: var(--gold-ink); font-size: 12px; font-weight: 900; letter-spacing: .04em; background: rgba(251,248,240,.82); border-bottom: 1px solid var(--line); }
    .audit-event { border-bottom: 1px solid var(--line); }
    .audit-event:last-child { border-bottom: 0; }
    details.audit-event > summary { list-style: none; cursor: pointer; }
    details.audit-event > summary::-webkit-details-marker { display: none; }
    details.audit-event > summary:hover, details.audit-event > summary:focus-visible { background: var(--panel-soft); outline: 2px solid var(--gold); outline-offset: -2px; }
    .audit-row { display: grid; grid-template-columns: var(--audit-cols); gap: 12px; align-items: start; min-height: 64px; padding: 13px 16px; }
    .audit-marker { position: relative; width: 10px; height: 10px; border-radius: 50%; margin-top: 5px; background: var(--gold); box-shadow: 0 0 0 4px rgba(200,153,63,.13); }
    .audit-marker::after { content: ""; position: absolute; top: 14px; bottom: -65px; left: 4px; width: 1px; background: var(--line); }
    .audit-row.audit-add .audit-marker, .audit-event.audit-add .audit-marker { background: var(--leaf); box-shadow: 0 0 0 4px rgba(47,107,74,.11); }
    .audit-row.audit-danger .audit-marker, .audit-event.audit-danger .audit-marker { background: #9e2a2b; box-shadow: 0 0 0 4px rgba(158,42,43,.1); }
    .audit-time { color: var(--ink); font-variant-numeric: tabular-nums; text-align: right; }
    .audit-time strong { font-family: var(--font-serif); font-size: 16px; line-height: 1.15; }
    .audit-main { min-width: 0; display: grid; gap: 4px; }
    .audit-title { min-width: 0; font-size: 14.5px; line-height: 1.35; overflow-wrap: anywhere; }
    .audit-context { display: none; color: var(--muted); font-size: 12.5px; line-height: 1.4; overflow-wrap: anywhere; }
    .audit-actor, .audit-object { min-width: 0; color: var(--muted); font-size: 12.5px; line-height: 1.4; overflow-wrap: anywhere; }
    .audit-actor { color: var(--ink); font-weight: 650; }
    .audit-empty-cell { color: var(--soft); }
    .audit-row-chevron { align-self: center; color: var(--gold-ink); font-size: 23px; line-height: 1; transition: transform .16s ease; }
    details.audit-event[open] .audit-row-chevron { transform: rotate(90deg); }
    .audit-detail-list { display: grid; margin: -3px 16px 14px 88px; border: 1px solid var(--line); border-radius: var(--radius-xs); background: var(--panel-soft); overflow: hidden; }
    .audit-detail-row { display: grid; grid-template-columns: minmax(110px,.42fr) minmax(0,1fr); gap: 12px; padding: 8px 10px; border-bottom: 1px solid var(--line); font-size: 12px; }
    .audit-detail-row:last-child { border-bottom: 0; }
    .audit-detail-row dt { color: var(--muted); }
    .audit-detail-row dd { min-width: 0; margin: 0; color: var(--ink); font-weight: 700; overflow-wrap: anywhere; }
    .filter-tabs { display: flex; gap: 8px; flex-wrap: wrap; }
    .filter-tab { min-height: 34px; display: inline-flex; align-items: center; justify-content: center; border: 1px solid var(--line); border-radius: var(--radius-pill); padding: 6px 12px; color: var(--muted); background: var(--panel); text-decoration: none; font-size: 13px; font-weight: 800; }
    .filter-tab.active, .filter-tab:hover { border-color: var(--gold); color: var(--ink); background: rgba(200,153,63,.14); }
    .quick-list { display: grid; }
    .quick-row { display: grid; grid-template-columns: 30px minmax(0,1fr) auto; gap: var(--space-3); align-items: center; padding: 13px 0; border-bottom: 1px solid var(--line); color: inherit; text-decoration: none; }
    button.quick-row { width: 100%; border: 0; border-bottom: 1px solid var(--line); background: transparent; font: inherit; text-align: left; cursor: pointer; }
    button.quick-row:hover { color: var(--gold-ink); }
    .quick-row:last-child { border-bottom: 0; }
    .quick-row > div { min-width: 0; }
    .quick-row .entry-actions { justify-self: end; }
    .quick-row svg { width: 24px; height: 24px; stroke: currentColor; stroke-width: 1.8; fill: none; stroke-linecap: round; stroke-linejoin: round; color: var(--ink); }
    .quick-row h3 { font-size: 18px; }
    .quick-row p { margin-top: 3px; color: var(--soft); font-size: 13.5px; line-height: 1.35; overflow-wrap: anywhere; }
    .quick-row.disabled { cursor: default; }
    .quick-row.disabled h3, .quick-row.disabled svg { color: var(--muted); }
    .quick-row .pill { justify-self: end; }
    .quick-arrow { color: var(--gold-ink); font-size: 24px; line-height: 1; }
    .digest-panel { grid-column: 1 / -1; }
    .digest-panel .quick-list { grid-template-columns: repeat(3,minmax(0,1fr)); gap: 10px; }
    .digest-panel .quick-row { border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 13px; background: var(--panel-soft); }
    .digest-panel .quick-row:last-child { border-bottom: 1px solid var(--line); }
    .events-panel { display: grid; gap: 18px; }
    .agenda-list { display: grid; gap: 11px; }
    .event-card { display: grid; grid-template-columns: 62px minmax(0,1fr); gap: 15px; align-items: start; border: 1px solid var(--line); border-radius: 14px; padding: 16px; color: inherit; background: #fffefb; text-decoration: none; }
    .event-card.is-next { border-color: rgba(200,153,63,.52); box-shadow: 0 12px 30px rgba(38,34,25,.06); }
    .event-card.past { opacity: .72; background: var(--panel-soft); }
    .date-badge { min-height: 58px; display: grid; place-items: center; align-content: center; gap: 2px; border-radius: var(--radius-sm); background: var(--ink); color: #fff; font-weight: 800; text-align: center; }
    .date-badge strong { font-family: var(--font-serif); font-size: 24px; line-height: .95; }
    .date-badge span { font-size: 11px; text-transform: uppercase; letter-spacing: .08em; }
    .event-info { min-width: 0; display: grid; gap: 9px; }
    .event-info h3 { font-size: 20px; overflow-wrap: anywhere; }
    .event-info p { color: var(--muted); line-height: 1.45; font-size: 13.5px; }
    .event-card-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; }
    .event-next-label { display: block; margin-bottom: 3px; color: var(--gold-ink); font-size: 10.5px; font-weight: 900; letter-spacing: .08em; text-transform: uppercase; }
    .event-meta { display: flex; flex-wrap: wrap; align-items: center; gap: 7px; color: var(--soft); font-size: 12.5px; font-weight: 700; }
    .event-meta strong { color: var(--ink); font-size: 13px; }
    .event-details { border-top: 1px dashed var(--line); padding-top: 8px; }
    .event-details > summary { cursor: pointer; list-style: none; color: var(--gold-ink); font-size: 12.5px; font-weight: 850; }
    .event-details > summary::-webkit-details-marker { display: none; }
    .event-details > summary::after { content: "›"; float: right; font-size: 18px; line-height: .8; }
    .event-details[open] > summary::after { transform: rotate(90deg); }
    .event-details-body { display: grid; gap: 10px; padding-top: 10px; }
    .event-actions { display: flex; flex-wrap: wrap; gap: 7px; padding-top: 2px; }
    .event-actions form { margin: 0; }
    .event-history { border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel-soft); overflow: hidden; }
    .event-history > summary { cursor: pointer; list-style: none; padding: 12px 14px; color: var(--ink); font-size: 13.5px; font-weight: 850; }
    .event-history > summary::-webkit-details-marker { display: none; }
    .event-history > summary::after { content: "›"; float: right; color: var(--gold-ink); font-size: 20px; line-height: .8; }
    .event-history[open] > summary { border-bottom: 1px solid var(--line); }
    .event-history[open] > summary::after { transform: rotate(90deg); }
    .event-history > .agenda-list { padding: 12px; }
    .status-strip { display: grid; gap: 14px; }
    .rule { display: flex; align-items: center; justify-content: space-between; gap: 14px; flex-wrap: wrap; color: var(--ink); line-height: 1.5; }
    .rule p { flex: 1 1 640px; min-width: 0; }
    .rule .pill { margin-left: auto; }
    .pill { display: inline-flex; align-items: center; min-height: 26px; border-radius: var(--radius-pill); padding: 3px 10px; font-size: 12px; font-weight: 800; background: rgba(200,153,63,.16); color: #8a6a1f; white-space: nowrap; }
    .pill.ok { background: rgba(47,107,74,.12); color: var(--leaf); }
    .pill.ok::before { content: ""; width: 8px; height: 8px; border-radius: 50%; background: currentColor; margin-right: 8px; }
    .pill.dringend { background: rgba(158,42,43,.12); color: #9e2a2b; }
    .pill.termin { background: rgba(47,107,74,.12); color: var(--leaf); }
    .pill.wartung { background: rgba(173,92,27,.14); color: #8a551f; }
    .pill.info { background: rgba(200,153,63,.16); color: #8a6a1f; }
    .pill.versammlung { background: rgba(32,37,31,.08); color: var(--ink); }
    .pill.reinigung { background: rgba(76,103,138,.11); color: #365475; }
    .pill.ablesung { background: rgba(47,107,74,.12); color: var(--leaf); }
    .pill.frist { background: rgba(158,42,43,.12); color: #9e2a2b; }
    .pill.sonstiges { background: rgba(200,153,63,.16); color: #8a6a1f; }
    .pill.unread { background: var(--gold); color: #172019; }
    .pill.status-open { background: rgba(200,153,63,.16); color: #8a6a1f; }
    .pill.status-progress { background: rgba(32,37,31,.08); color: var(--ink); }
    .pill.status-done { background: rgba(47,107,74,.12); color: var(--leaf); }
    .pill.status-closed { background: rgba(158,42,43,.1); color: #9e2a2b; }
    .chips { display: flex; flex-wrap: wrap; gap: 6px; }
    .chip { display: inline-flex; align-items: center; gap: 5px; border: 1px solid var(--line); background: var(--panel-soft); color: #6f6a5c; border-radius: var(--radius-sm); padding: 4px 10px; font-size: 12.5px; font-weight: 600; white-space: nowrap; }
    .chip strong { color: var(--gold-ink); }
    .issue-dashboard { display: grid; gap: 18px; }
    .issue-stats { display: grid; grid-template-columns: repeat(auto-fit,minmax(170px,1fr)); gap: 12px; }
    .issue-stat { min-height: 92px; display: grid; gap: 7px; align-content: center; border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 16px; background: var(--panel); box-shadow: var(--shadow-panel); text-decoration: none; color: inherit; }
    .issue-stat:hover { border-color: var(--gold); }
    .issue-stat span { color: var(--gold-ink); font-size: 11px; font-weight: 900; letter-spacing: .08em; text-transform: uppercase; }
    .issue-stat strong { font-family: var(--font-serif); font-size: 30px; line-height: .95; }
    .issue-stat p { color: var(--muted); font-size: 13px; line-height: 1.35; }
    .issue-tabs { display: flex; flex-wrap: wrap; gap: 8px; }
    .issue-tab { min-height: 38px; display: inline-flex; align-items: center; justify-content: center; border: 1px solid var(--line); border-radius: var(--radius-pill); padding: 8px 13px; background: var(--panel); color: var(--muted); text-decoration: none; font-size: 13px; font-weight: 850; }
    .issue-tab.active, .issue-tab:hover { border-color: rgba(200,153,63,.45); background: rgba(200,153,63,.12); color: var(--ink); }
    .issue-create-panel { padding: 0; overflow: hidden; }
    .issue-create-panel > summary { min-height: 76px; display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 12px; align-items: center; padding: 18px 20px; cursor: pointer; list-style: none; }
    .issue-create-panel > summary::-webkit-details-marker { display: none; }
    .issue-create-panel > summary h2 { font-size: 24px; }
    .issue-create-panel > summary p { margin-top: 4px; color: var(--muted); font-size: 14px; line-height: 1.4; }
    .issue-create-panel > summary::after { content: "Öffnen"; min-height: 34px; display: inline-flex; align-items: center; justify-content: center; border: 1px solid var(--ink); border-radius: var(--radius-xs); padding: 7px 11px; background: var(--ink); color: #fff; font-size: 13px; font-weight: 850; }
    .issue-create-panel[open] > summary { border-bottom: 1px solid var(--line); }
    .issue-create-panel[open] > summary::after { content: "Schließen"; border-color: var(--line); background: transparent; color: var(--ink); }
    .issue-create-body { padding: 18px 20px 20px; }
    .issue-management-preview { display: grid; gap: 16px; }
    .issue-preview-list { display: grid; gap: 9px; }
    .issue-preview-card { display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 10px; align-items: center; border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 12px 13px; background: var(--panel-soft); color: inherit; text-decoration: none; }
    .issue-preview-card:hover { border-color: var(--gold); }
    .issue-preview-card strong { display: block; min-width: 0; overflow-wrap: anywhere; font-size: 14px; }
    .issue-preview-card > span:first-child > span { display: block; margin-top: 4px; color: var(--muted); font-size: 12.5px; line-height: 1.35; }
    .issue-preview-card .chips { justify-self: end; margin-top: 0; justify-content: flex-end; }
    .issue-preview-card .pill { margin-top: 0; }
    .issue-management-actions { display: flex; flex-wrap: wrap; gap: 10px; }
    .issue-layout { display: grid; grid-template-columns: minmax(0,1.45fr) minmax(280px,.8fr); gap: 22px; align-items: start; }
    .issue-form { display: grid; gap: 20px; max-width: 920px; margin: 0 auto; }
    .issue-form label { display: grid; gap: 7px; color: var(--gold-ink); font-size: 11px; font-weight: 800; letter-spacing: .06em; text-transform: uppercase; }
    .issue-form input, .issue-form select, .issue-form textarea { width: 100%; border: 1px solid #e2dac9; border-radius: var(--radius-sm); padding: 12px; font: inherit; background: #fffefb; color: var(--ink); }
    .issue-form textarea { min-height: 132px; resize: vertical; line-height: 1.45; }
    .issue-form input[type=file] { padding: 10px; color: var(--muted); }
    .file-control { position: relative; min-height: 44px; display: flex; align-items: center; gap: 10px; border: 1px solid #e2dac9; border-radius: var(--radius-sm); padding: 10px 12px; background: #fffefb; color: var(--ink); overflow: hidden; }
    .file-control input[type=file] { position: absolute; inset: 0; opacity: 0; cursor: pointer; }
    .file-control span { pointer-events: none; font-size: 13px; font-weight: 800; letter-spacing: 0; text-transform: none; }
    .file-control.is-filled { border-color: rgba(47,107,74,.34); background: rgba(47,107,74,.06); }
    .file-control.is-dragover { border-color: var(--gold); background: rgba(200,153,63,.1); }
    .attachment-picker { display: none; grid-column: 1 / -1; gap: 8px; margin-top: 8px; }
    .attachment-picker.has-files { display: grid; }
    .attachment-picker-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; color: var(--muted); font-size: 12px; font-weight: 800; }
    .attachment-picker-list { display: grid; grid-template-columns: repeat(auto-fill,minmax(210px,1fr)); gap: 8px; }
    .attachment-picker-item { display: grid; grid-template-columns: 52px minmax(0,1fr) 30px; gap: 10px; align-items: center; min-width: 0; border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 8px; background: var(--panel-soft); }
    .attachment-picker-thumb { width: 52px; height: 44px; display: grid; place-items: center; border-radius: var(--radius-xs); background: rgba(200,153,63,.14); color: var(--gold-ink); font-size: 11px; font-weight: 900; letter-spacing: .05em; overflow: hidden; }
    .attachment-picker-thumb img { width: 100%; height: 100%; object-fit: cover; display: block; }
    .attachment-picker-copy { min-width: 0; display: grid; gap: 3px; }
    .attachment-picker-name { min-width: 0; color: var(--ink); font-size: 13px; font-weight: 800; line-height: 1.25; overflow-wrap: anywhere; }
    .issue-form .attachment-picker-name { display: -webkit-box; overflow: hidden; -webkit-box-orient: vertical; -webkit-line-clamp: 2; }
    .attachment-picker-meta { color: var(--soft); font-size: 11.5px; font-weight: 700; }
    .attachment-picker-remove { width: 30px; height: 30px; display: grid; place-items: center; border: 1px solid rgba(32,37,31,.16); border-radius: var(--radius-xs); background: var(--panel); color: var(--ink); font: inherit; font-size: 18px; line-height: 1; cursor: pointer; }
    .attachment-picker-remove:hover { border-color: #9e2a2b; color: #9e2a2b; }
    .attachment-picker-progress { display: none; height: 5px; border-radius: var(--radius-pill); background: #ece5d6; overflow: hidden; }
    .attachment-picker-progress span { display: block; width: 40%; height: 100%; border-radius: inherit; background: var(--gold); animation: upload-progress 1.1s ease-in-out infinite; }
    .attachment-picker.is-uploading .attachment-picker-progress { display: block; }
    @keyframes upload-progress { 0% { transform: translateX(-120%); } 100% { transform: translateX(260%); } }
    .issue-form .hint { margin-top: 2px; color: var(--soft); font-size: 12px; font-weight: 600; letter-spacing: 0; text-transform: none; }
    .issue-create-panel { scroll-margin-top: 80px; }
    .issue-wizard-step { display: grid; gap: 18px; min-width: 0; border: 0; padding: 0; }
    .issue-wizard-step[hidden] { display: none; }
    .issue-wizard-heading { display: grid; gap: 6px; }
    .issue-wizard-progress { color: var(--gold-ink); font-size: 12px; font-weight: 850; letter-spacing: .06em; }
    .issue-wizard-heading h3 { font-family: var(--font-serif); font-size: clamp(30px,4vw,42px); line-height: 1.04; }
    .issue-wizard-heading p { color: var(--muted); font-size: 15px; line-height: 1.45; }
    .issue-safety-note { display: block; border-left: 3px solid var(--gold); border-radius: 0 var(--radius-xs) var(--radius-xs) 0; padding: 10px 12px; background: rgba(200,153,63,.08); color: var(--muted); font-size: 12.5px; font-weight: 650; line-height: 1.45; }
    .issue-safety-note a { min-height: 44px; display: inline-flex; align-items: center; margin-block: -10px; color: var(--ink); font-weight: 850; text-underline-offset: 3px; }
    .issue-field-error { color: #8f2a2b; font-size: 12.5px; font-weight: 750; letter-spacing: 0; line-height: 1.4; text-transform: none; }
    .issue-field-error[hidden] { display: none; }
    .issue-form [aria-invalid="true"] { border-color: #a43a3b; box-shadow: 0 0 0 2px rgba(164,58,59,.11); }
    .issue-category-options[aria-invalid="true"], .issue-location-options[aria-invalid="true"] { border-color: #a43a3b; box-shadow: 0 0 0 2px rgba(164,58,59,.11); }
    .issue-category { display: grid; gap: 8px; }
    .issue-category > span { color: var(--ink); font-size: 13px; font-weight: 800; }
    .issue-category-options { display: grid; grid-template-columns: repeat(4,minmax(0,1fr)); border: 1px solid #e2dac9; border-radius: var(--radius-sm); overflow: hidden; }
    .issue-form .issue-category-choice { position: relative; min-height: 64px; display: grid; place-items: center; padding: 12px 9px; color: var(--ink); font-size: 13px; font-weight: 800; letter-spacing: 0; text-align: center; text-transform: none; cursor: pointer; }
    .issue-category-choice + .issue-category-choice { border-left: 1px solid #e2dac9; }
    .issue-category-choice input { position: absolute; width: 1px; height: 1px; opacity: 0; pointer-events: none; }
    .issue-category-choice:has(input:checked) { background: rgba(200,153,63,.1); box-shadow: inset 0 0 0 2px rgba(200,153,63,.72); }
    .issue-category-choice:has(input:focus-visible) { outline: 3px solid rgba(200,153,63,.42); outline-offset: -3px; }
    .issue-file-row { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
    .issue-file-row label { min-width: 200px; }
    .issue-file-row .file-control { border-style: dashed; }
    .issue-location-options { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 10px; }
    .issue-form .issue-location-choice { position: relative; min-height: 70px; display: grid; gap: 3px; align-content: center; border: 1px solid #e2dac9; border-radius: var(--radius-sm); padding: 13px 15px; color: var(--ink); letter-spacing: 0; text-transform: none; cursor: pointer; }
    .issue-location-choice input { position: absolute; width: 1px; height: 1px; opacity: 0; pointer-events: none; }
    .issue-location-choice strong { font-size: 14px; }
    .issue-location-choice span { color: var(--muted); font-size: 12px; font-weight: 600; line-height: 1.35; }
    .issue-location-choice:has(input:checked) { border-color: rgba(200,153,63,.72); background: rgba(200,153,63,.08); box-shadow: inset 0 0 0 1px rgba(200,153,63,.46); }
    .issue-location-choice:has(input:focus-visible) { outline: 3px solid rgba(200,153,63,.42); outline-offset: 2px; }
    .issue-review { display: grid; gap: 0; border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel-soft); }
    .issue-review-enhanced { display: none; }
    .issue-form.is-enhanced .issue-review-enhanced { display: grid; }
    .issue-review-row { display: grid; grid-template-columns: 132px minmax(0,1fr); gap: 16px; padding: 13px 15px; }
    .issue-review-row + .issue-review-row { border-top: 1px solid var(--line); }
    .issue-review-row dt { color: var(--muted); font-size: 12px; font-weight: 750; }
    .issue-review-row dd { min-width: 0; color: var(--ink); font-size: 13px; font-weight: 750; line-height: 1.45; overflow-wrap: anywhere; }
    .issue-review-copy { display: grid; gap: 8px; }
    .issue-review-copy-text.is-collapsed { display: -webkit-box; overflow: hidden; -webkit-box-orient: vertical; -webkit-line-clamp: 5; }
    .issue-review-expand { width: max-content; min-height: 44px !important; border: 0 !important; padding: 0 !important; background: transparent !important; color: var(--gold-ink) !important; font-size: 12px !important; text-decoration: underline; text-underline-offset: 3px; }
    .issue-review-expand[hidden] { display: none !important; }
    .issue-title-option { border-top: 1px solid var(--line); padding-top: 2px; }
    .issue-title-option > summary { min-height: 44px; display: flex; align-items: center; justify-content: space-between; gap: 12px; cursor: pointer; list-style: none; color: var(--ink); font-size: 13px; font-weight: 800; }
    .issue-title-option > summary::-webkit-details-marker { display: none; }
    .issue-title-option > summary::after { content: "+"; color: var(--gold-ink); font-size: 18px; }
    .issue-title-option[open] > summary::after { content: "−"; }
    .issue-title-option > label { margin-top: 10px; }
    .issue-wizard-actions { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding-top: 2px; }
    .issue-wizard-actions .wizard-back, .issue-wizard-actions .wizard-cancel, .issue-wizard-actions .wizard-exit { min-height: 46px; display: inline-flex; align-items: center; border-color: transparent; background: transparent; color: var(--ink); padding-inline: 2px; text-decoration: underline; text-underline-offset: 4px; }
    .issue-wizard-actions .wizard-next, .issue-wizard-actions .wizard-submit { min-width: 220px; padding-inline: 28px; }
    .issue-form button { min-height: 46px; border: 1px solid var(--ink); border-radius: var(--radius-sm); background: var(--ink); color: #fff; font: inherit; font-weight: 800; cursor: pointer; }
    .issue-form button:hover { background: #2c3329; }
    .issue-form .wizard-only { display: none; }
    .issue-form.is-enhanced .wizard-only { display: inline-flex; align-items: center; justify-content: center; }
    .issue-flash { margin: 0 0 14px; padding: 11px 13px; border-radius: var(--radius-sm); font-size: 13.5px; font-weight: 700; border: 1px solid transparent; }
    .issue-flash.ok { background: rgba(47,107,74,.12); color: var(--leaf); border-color: rgba(47,107,74,.25); }
    .issue-flash.warn { background: rgba(200,153,63,.14); color: #93701d; border-color: rgba(200,153,63,.3); }
    .issue-list { display: grid; gap: 10px; }
    .issue-card { border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 16px; background: #fffefb; display: grid; gap: 11px; }
    .issue-card h3 { font-size: 20px; }
    .issue-card-head { display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 12px; align-items: start; }
    .issue-card-head .issue-meta { justify-content: flex-end; }
    .issue-progress { height: 5px; border-radius: var(--radius-pill); background: #ece6da; overflow: hidden; }
    .issue-progress-fill { display: block; height: 100%; border-radius: inherit; background: var(--gold); }
    .issue-progress-fill.status-open { width: 24%; }
    .issue-progress-fill.status-progress { width: 66%; background: var(--leaf); }
    .issue-progress-fill.status-done, .issue-progress-fill.status-closed { width: 100%; background: var(--leaf); }
    .issue-next-step { border-left: 3px solid var(--gold); padding: 8px 11px; color: var(--ink); background: rgba(200,153,63,.08); font-size: 13.5px; font-weight: 750; line-height: 1.4; }
    .issue-card-details { border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel-soft); overflow: hidden; }
    .issue-card-details > summary { min-height: 42px; display: flex; align-items: center; justify-content: space-between; gap: 10px; cursor: pointer; list-style: none; padding: 10px 12px; color: var(--ink); font-size: 13px; font-weight: 850; }
    .issue-card-details > summary::-webkit-details-marker { display: none; }
    .issue-card-details > summary::after { content: "Öffnen"; color: var(--gold-ink); font-size: 12px; }
    .issue-card-details[open] > summary { border-bottom: 1px solid var(--line); }
    .issue-card-details[open] > summary::after { content: "Schließen"; }
    .issue-card-details-body { display: grid; gap: 12px; padding: 12px; }
    .issue-description { display: grid; gap: 5px; }
    .issue-description strong { font-size: 12px; color: var(--gold-ink); text-transform: uppercase; letter-spacing: .06em; }
    .issue-description p { color: var(--muted); line-height: 1.5; }
    .issue-description-preview p { display: -webkit-box; max-height: 3em; overflow: hidden; -webkit-line-clamp: 2; -webkit-box-orient: vertical; }
	.issue-meta { display: flex; flex-wrap: wrap; gap: 7px; align-items: center; color: var(--soft); font-size: 12.5px; font-weight: 700; }
	.issue-location { color: var(--muted); font-size: 13px; line-height: 1.35; }
	.issue-proposal { border: 1px solid rgba(47,107,74,.18); border-radius: var(--radius-xs); padding: 9px 11px; background: rgba(47,107,74,.08); color: var(--leaf); font-size: 13px; line-height: 1.4; }
	.issue-proposal strong { color: var(--ink); }
	.issue-card-tools { border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel-soft); overflow: hidden; }
	.issue-card-tools > summary { min-height: 42px; display: flex; align-items: center; justify-content: space-between; gap: 10px; padding: 9px 11px; color: var(--ink); cursor: pointer; list-style: none; font-size: 13px; font-weight: 850; }
	.issue-card-tools > summary::-webkit-details-marker { display: none; }
	.issue-card-tools > summary::after { content: "Öffnen"; border: 1px solid var(--line); border-radius: var(--radius-xs); padding: 5px 8px; background: var(--panel); color: var(--muted); font-size: 12px; font-weight: 850; }
	.issue-card-tools[open] > summary { border-bottom: 1px solid var(--line); }
	.issue-card-tools[open] > summary::after { content: "Schließen"; }
	.issue-card-tools-body { display: grid; gap: 10px; padding: 11px; }
	.issue-estimate { display: grid; gap: 8px; border: 1px solid rgba(200,153,63,.28); border-radius: var(--radius-sm); padding: 11px; background: rgba(200,153,63,.08); }
	.issue-estimate-head { display: flex; align-items: baseline; justify-content: space-between; gap: 12px; flex-wrap: wrap; }
	.issue-estimate-head strong { font-family: var(--font-serif); font-size: 18px; }
	.issue-estimate .mini { margin: 0; }
	.issue-actions { display: flex; flex-wrap: wrap; gap: 8px; align-items: end; }
	.issue-actions form { margin: 0; }
	.issue-actions label { display: grid; gap: 5px; min-width: 150px; color: var(--gold-ink); font-size: 10.5px; font-weight: 800; letter-spacing: .06em; text-transform: uppercase; }
	.issue-actions label.assignee { flex: 1 1 240px; }
	.issue-actions label.proposal { flex: 1 1 260px; }
	.issue-actions label.note { flex: 1 1 280px; }
	.issue-actions input, .issue-actions select { width: 100%; border: 1px solid #e2dac9; border-radius: var(--radius-xs); min-height: 38px; padding: 8px 10px; color: var(--ink); background: #fffefb; font: inherit; font-size: 13px; }
	.issue-actions .hint { color: var(--soft); font-size: 11.5px; font-weight: 600; letter-spacing: 0; line-height: 1.35; text-transform: none; }
	.issue-actions .assignee-removal { display: flex; align-items: center; gap: 7px; color: var(--ink); }
	.issue-actions .assignee-removal input[type=checkbox] { width: auto; min-height: 0; padding: 0; }
    .issue-actions button { min-height: 38px; border: 1px solid var(--ink); border-radius: var(--radius-xs); padding: 8px 12px; background: var(--ink); color: #fff; font: inherit; font-size: 13px; font-weight: 800; cursor: pointer; }
    .issue-actions .ghost { background: transparent; color: var(--ink); border-color: var(--line); }
    .issue-board-filter { display: grid; grid-template-columns: repeat(12, minmax(0,1fr)); gap: 10px; margin-bottom: 14px; align-items: end; }
    .issue-board-filter label { display: grid; gap: 5px; color: var(--gold-ink); font-size: 10.5px; font-weight: 800; letter-spacing: .06em; text-transform: uppercase; grid-column: span 2; }
    .issue-board-filter label.assignee { grid-column: span 3; }
    .issue-board-filter label.sort { grid-column: span 3; }
    .issue-board-filter input, .issue-board-filter select { width: 100%; border: 1px solid #e2dac9; border-radius: var(--radius-xs); min-height: 38px; padding: 8px 10px; color: var(--ink); background: #fffefb; font: inherit; font-size: 13px; }
    .issue-board-filter .board-filter-actions { grid-column: span 2; display: flex; gap: 8px; align-items: center; }
    .issue-board-filter button, .issue-board-filter a { min-height: 38px; border: 1px solid var(--ink); border-radius: var(--radius-xs); padding: 8px 12px; background: var(--ink); color: #fff; font: inherit; font-size: 13px; font-weight: 800; cursor: pointer; text-decoration: none; display: inline-flex; align-items: center; }
    .issue-board-filter a { background: transparent; color: var(--ink); border-color: var(--line); }
    .issue-board-page { width: min(1160px,100%); }
    .issue-board-panel { display: grid; gap: 18px; border: 0; padding: 0; background: transparent; box-shadow: none; }
    .issue-board-toolbar { display: grid; grid-template-columns: auto minmax(0,1fr); align-items: stretch; border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel); box-shadow: 0 10px 30px rgba(38,34,25,.045); overflow: hidden; }
    .issue-board-summary { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; padding: 14px 16px; border-right: 1px solid var(--line); }
    .issue-board-summary .pill { min-height: 32px; padding-inline: 12px; }
    .issue-board-tools { min-width: 0; margin: 0; border: 0; background: transparent; overflow: visible; }
    .issue-board-tools > summary { min-height: 64px; display: grid; grid-template-columns: 32px auto minmax(0,1fr) auto; gap: 12px; align-items: center; cursor: pointer; list-style: none; padding: 10px 16px; }
    .issue-board-tools > summary::-webkit-details-marker { display: none; }
    .issue-board-tools > summary strong { font-size: 13.5px; }
    .issue-board-tools > summary > span:not(.issue-filter-icon) { justify-self: end; color: var(--muted); font-size: 12.5px; }
    .issue-filter-icon { width: 32px; height: 32px; display: grid; place-items: center; border-radius: 50%; background: rgba(47,107,74,.09); color: var(--leaf); }
    .issue-filter-icon svg { width: 18px; height: 18px; fill: none; stroke: currentColor; stroke-width: 1.8; stroke-linecap: round; stroke-linejoin: round; }
    .issue-board-tools > summary::after { content: "Filter öffnen"; color: var(--gold-ink); font-size: 12px; font-weight: 850; }
    .issue-board-tools[open] > summary { border-bottom: 1px solid var(--line); }
    .issue-board-tools[open] > summary::after { content: "Filter schließen"; }
    .issue-board-tools .issue-board-filter { margin: 0; padding: 14px; }
    .issue-board-tools[open] { grid-column: 1 / -1; }
    /* Sonst bliebe neben der Zaehlung eine leere Zelle stehen, sobald der
       Filterbereich in eine eigene Zeile rutscht. */
    .issue-board-toolbar:has(.issue-board-tools[open]) .issue-board-summary { grid-column: 1 / -1; border-right: 0; border-bottom: 1px solid var(--line); }
    .issue-work-card { position: relative; gap: 11px; border-color: rgba(32,37,31,.13); border-radius: 14px; padding: 17px 18px 15px 24px; background: var(--panel); box-shadow: 0 10px 28px rgba(38,34,25,.055); overflow: hidden; }
    .issue-work-card::before { content: ""; position: absolute; inset: 0 auto 0 0; width: 6px; background: var(--gold); }
    .issue-work-card.status-progress::before, .issue-work-card.status-done::before, .issue-work-card.status-closed::before { background: var(--leaf); }
    .issue-work-card .issue-card-head { align-items: center; }
    .issue-work-card h3 { font-size: 20px; line-height: 1.15; }
    .issue-work-card .issue-location { margin-top: 4px; font-size: 13px; }
    .issue-work-card .issue-next-step { position: relative; border-left: 0; border-radius: var(--radius-xs); padding: 9px 13px 9px 40px; background: rgba(200,153,63,.085); font-size: 13px; }
    .issue-work-card .issue-next-step::before { content: "→"; position: absolute; left: 12px; top: 50%; width: 21px; height: 21px; display: grid; place-items: center; border: 1px solid rgba(200,153,63,.32); border-radius: 50%; background: var(--panel); color: var(--gold-ink); font-size: 12px; transform: translateY(-50%); }
    .issue-card-foot { display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 18px; align-items: center; border-top: 1px dashed var(--line); padding-top: 11px; }
    .issue-card-foot .issue-description-preview p { color: var(--muted); font-size: 14px; }
    .issue-card-foot .issue-card-tools { min-width: 0; border: 0; background: transparent; overflow: visible; }
    .issue-card-foot .issue-card-tools > summary { width: max-content; min-height: 44px; margin-left: auto; border-radius: var(--radius-xs); padding: 10px 15px; background: var(--ink); color: #fff; font-size: 13.5px; }
    .issue-card-foot .issue-card-tools > summary::after { content: "→"; border: 0; padding: 0 0 0 13px; background: transparent; color: #fff; font-size: 18px; }
    .issue-card-foot .issue-card-tools[open] { grid-column: 1 / -1; width: 100%; }
    .issue-card-foot .issue-card-tools[open] > summary { margin-bottom: 10px; background: var(--panel-soft); color: var(--ink); }
    .issue-card-foot .issue-card-tools[open] > summary::after { content: "Schließen"; padding-left: 13px; color: var(--muted); font-size: 12px; }
    .issue-card-foot .issue-card-tools[open] > summary { border-bottom: 0; }
    .issue-card-foot .issue-card-tools-body { border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel-soft); }
    .issue-triage-link { min-width: 118px; }
    /* Triage-Board: Schnellfilter, Leerzustand und Filter-Leerzustand. Der
       Leerzustand erklaert Reihenfolge und Statusweg, statt nur zu melden,
       dass nichts da ist. */
    .issue-board-quick { grid-column: 1 / -1; display: flex; flex-wrap: wrap; gap: 7px; align-items: center; border-top: 1px solid var(--line); padding: 10px 16px; background: var(--panel-soft); }
    .issue-board-quick > span { margin-right: 3px; color: var(--gold-ink); font-size: 10.5px; font-weight: 850; letter-spacing: .09em; text-transform: uppercase; }
    .issue-board-quick a { min-height: 32px; display: inline-flex; align-items: center; border: 1px solid var(--line); border-radius: var(--radius-pill); padding: 0 13px; background: var(--panel); color: var(--muted); font-size: 12.5px; font-weight: 800; text-decoration: none; }
    .issue-board-quick a:hover { border-color: var(--gold); color: var(--gold-ink); }
    .issue-board-quick a[aria-current="true"] { border-color: var(--ink); background: var(--ink); color: #fff; }
    .board-blank { display: grid; grid-template-columns: minmax(0,1.4fr) minmax(286px,.9fr); gap: 16px; }
    .board-blank-main { display: grid; align-content: center; gap: 20px; border: 1px dashed rgba(200,153,63,.45); border-radius: var(--radius-sm); background: rgba(255,254,251,.7); padding: clamp(22px,3.2vw,36px); }
    .board-blank-lead { display: grid; justify-items: start; gap: 13px; }
    .board-blank-icon { width: 52px; height: 52px; display: grid; place-items: center; border: 1px solid rgba(200,153,63,.3); border-radius: 12px; background: rgba(200,153,63,.1); color: var(--gold-ink); }
    .board-blank-icon svg { width: 26px; height: 26px; fill: none; stroke: currentColor; stroke-width: 1.5; stroke-linecap: round; stroke-linejoin: round; }
    .board-blank-main h2 { font-size: clamp(25px,3vw,31px); }
    .board-blank-main p { max-width: 56ch; color: var(--muted); font-size: 15px; line-height: 1.55; }
    .board-blank-actions { display: flex; flex-wrap: wrap; gap: 9px; margin-top: 3px; }
    .board-blank-actions .button { min-height: 44px; }
    .board-blank-side { display: grid; align-content: start; gap: 13px; border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel); padding: 20px; box-shadow: var(--shadow-panel); }
    .board-blank-side h2 { font-size: 12px; font-weight: 800; letter-spacing: .12em; text-transform: uppercase; color: var(--gold-ink); font-family: var(--font-sans); }
    .board-blank-steps { display: grid; gap: 11px; margin: 0; padding: 0; list-style: none; counter-reset: board-step; }
    .board-blank-steps li { display: grid; grid-template-columns: 24px minmax(0,1fr); gap: 2px 11px; align-items: start; counter-increment: board-step; }
    .board-blank-steps li::before { content: counter(board-step); grid-row: 1 / span 2; width: 24px; height: 24px; display: grid; place-items: center; border-radius: 50%; background: var(--ink); color: #fff; font-size: 12px; font-weight: 850; }
    .board-blank-steps strong { grid-column: 2; font-size: 13.5px; }
    .board-blank-steps span { grid-column: 2; color: var(--muted); font-size: 12.5px; line-height: 1.4; }
    .board-blank-note { border-top: 1px solid var(--line); padding-top: 12px; color: var(--muted); font-size: 12.5px; line-height: 1.5; }
    .board-blank-facts { grid-column: 1 / -1; display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 14px; margin: 0; padding: 0; list-style: none; }
    .board-blank-facts li { display: grid; gap: 5px; border-top: 2px solid var(--ink); padding-top: 11px; }
    .board-blank-facts strong { font-size: 13.5px; }
    .board-blank-facts span { color: var(--muted); font-size: 12.5px; line-height: 1.5; }
    .board-filter-blank { min-height: max(200px, 28vh); display: grid; grid-template-columns: auto minmax(0,1fr) auto; gap: 18px; align-items: center; align-content: center; border: 1px dashed rgba(200,153,63,.45); border-radius: var(--radius-sm); background: rgba(255,254,251,.7); padding: 24px; }
    .board-filter-blank h2 { font-size: 21px; }
    .board-filter-blank p { margin-top: 4px; color: var(--muted); font-size: 13.5px; line-height: 1.5; }
    .board-filter-blank .button { min-height: 44px; }
    @media (min-width: 901px) {
      .board-blank { min-height: max(430px, calc(100vh - 356px)); grid-template-rows: minmax(0,1fr) auto; }
    }
    .issue-triage-page { width: min(1180px,100%); gap: 24px; padding-bottom: 48px; }
    /* Triage-Ansicht: Entscheidung links, Fallstand rechts. Die Fakten bleiben
       waehrend der Entscheidung sichtbar, statt hinter einer Klappe zu liegen. */
    .issue-triage-layout { display: grid; grid-template-columns: minmax(0,1fr) minmax(286px,330px); gap: 20px; align-items: start; }
    .issue-triage-main { min-width: 0; display: grid; gap: 20px; }
    .issue-triage-main .issue-triage-card { width: 100%; justify-self: stretch; }
    .issue-triage-aside { min-width: 0; display: grid; align-content: start; gap: 13px; }
    .issue-triage-state { display: grid; gap: 12px; border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel); padding: 18px; box-shadow: var(--shadow-panel); }
    .issue-triage-state h2 { font-size: 12px; font-weight: 800; letter-spacing: .12em; text-transform: uppercase; color: var(--gold-ink); font-family: var(--font-sans); }
    .issue-triage-state dl { display: grid; margin: 0; }
    .issue-triage-state dl > div { min-width: 0; display: grid; grid-template-columns: 84px minmax(0,1fr); gap: 10px; align-items: baseline; border-top: 1px solid var(--line); padding: 8px 0; }
    .issue-triage-state dl > div:first-child { border-top: 0; padding-top: 0; align-items: center; }
    .issue-triage-state dt { color: var(--muted); font-size: 12px; font-weight: 750; }
    .issue-triage-state dd { min-width: 0; margin: 0; font-size: 13.5px; font-weight: 750; overflow-wrap: anywhere; }
    .issue-triage-state-line { color: var(--muted); font-size: 12.5px; line-height: 1.45; }
    .issue-triage-state-line strong { color: var(--ink); }
    .issue-triage-state-next { border-top: 1px solid var(--line); padding-top: 12px; color: var(--muted); font-size: 12.5px; line-height: 1.45; }
    .issue-triage-aside .issue-triage-more { width: 100%; border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel); }
    .issue-triage-aside .issue-triage-more > summary { padding: 14px 16px; }
    .issue-triage-aside .issue-triage-more-body { border: 0; border-top: 1px solid var(--line); border-radius: 0; }
    @media (max-width: 1080px) {
      .issue-triage-layout { grid-template-columns: minmax(0,1fr); }
      .issue-triage-state dl { grid-template-columns: repeat(auto-fit,minmax(164px,1fr)); gap: 0 18px; }
      .issue-triage-state dl > div { grid-template-columns: minmax(0,1fr); gap: 2px; align-items: start; border-top: 0; border-bottom: 1px solid var(--line); padding: 8px 0; }
      .issue-triage-state dl > div:first-child { align-items: start; padding-top: 8px; }
    }
    .issue-triage-context { display: grid; gap: 12px; border-bottom: 1px solid var(--line); padding-bottom: 24px; }
    .issue-triage-context h1 { font-size: clamp(38px,5vw,58px); }
    .issue-triage-context > p { max-width: 720px; color: var(--muted); font-size: 16px; line-height: 1.6; }
    .issue-triage-back { width: max-content; min-height: 44px; display: inline-flex; align-items: center; color: var(--muted); font-size: 13px; font-weight: 800; text-decoration: none; }
    .issue-triage-back:hover { color: var(--ink); }
    .issue-triage-facts { display: flex; flex-wrap: wrap; gap: 7px 22px; color: var(--muted); font-size: 13.5px; font-weight: 700; }
    .issue-triage-facts span { position: relative; }
    .issue-triage-facts span + span::before { content: "·"; position: absolute; left: -13px; color: var(--gold); }
    .issue-triage-card { width: min(780px,100%); justify-self: center; display: grid; gap: 28px; border: 1px solid rgba(32,37,31,.13); border-radius: 16px; padding: clamp(22px,4vw,40px); background: var(--panel); box-shadow: 0 18px 50px rgba(38,34,25,.08); }
    .issue-triage-progress { display: flex; align-items: center; justify-content: space-between; gap: 16px; color: var(--gold-ink); font-size: 12px; font-weight: 850; }
    .issue-triage-progress > span:last-child { display: flex; gap: 6px; }
    .issue-triage-progress i { width: 26px; height: 4px; border-radius: var(--radius-pill); background: #e4e1d9; }
    .issue-triage-progress i.active { background: var(--gold); }
    .issue-triage-card fieldset { min-width: 0; display: grid; gap: 12px; border: 0; padding: 0; }
    .issue-triage-card legend { margin-bottom: 20px; font-family: var(--font-serif); color: var(--ink); font-size: clamp(27px,3.4vw,36px); font-weight: 750; line-height: 1.15; }
    .issue-triage-choice { position: relative; min-width: 0; display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 16px; align-items: center; border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 17px 18px; background: #fffefb; color: var(--ink); cursor: pointer; font-size: inherit; font-weight: inherit; letter-spacing: 0; text-transform: none; transition: border-color .16s ease, background .16s ease, box-shadow .16s ease; }
    .issue-triage-choice:hover { border-color: rgba(47,107,74,.45); }
    .issue-triage-choice:has(input:focus-visible) { outline: 3px solid rgba(200,153,63,.28); outline-offset: 2px; }
    .issue-triage-choice:has(input:checked) { border-color: var(--leaf); background: rgba(47,107,74,.07); box-shadow: inset 0 0 0 1px rgba(47,107,74,.12); }
    .issue-triage-choice.urgent:has(input:checked) { border-color: #9e2a2b; background: rgba(158,42,43,.055); box-shadow: inset 0 0 0 1px rgba(158,42,43,.1); }
    .issue-triage-choice > span { min-width: 0; display: grid; gap: 3px; }
    .issue-triage-choice strong { color: var(--ink); font-size: 16px; letter-spacing: 0; text-transform: none; }
    .issue-triage-choice small { color: var(--muted); font-size: 13px; font-weight: 600; letter-spacing: 0; line-height: 1.4; text-transform: none; }
    .issue-triage-choice input { grid-column: 2; grid-row: 1; width: 20px; height: 20px; margin: 0; accent-color: var(--leaf); }
    .issue-triage-actions { display: flex; align-items: center; justify-content: space-between; gap: 14px; border-top: 1px solid var(--line); padding-top: 22px; }
    .issue-triage-actions > a:not(.button) { color: var(--muted); font-size: 13px; font-weight: 800; }
    .issue-triage-actions button { min-width: 132px; min-height: 44px; border: 1px solid var(--ink); border-radius: var(--radius-xs); padding: 10px 18px; background: var(--ink); color: #fff; font: inherit; font-weight: 850; cursor: pointer; }
    .issue-triage-actions button:hover { background: #2c3329; }
    .issue-triage-done { grid-template-columns: auto minmax(0,1fr); align-items: start; }
    .issue-triage-done-mark { width: 46px; height: 46px; display: grid; place-items: center; border-radius: 50%; background: rgba(47,107,74,.1); color: var(--leaf); font-size: 22px; font-weight: 900; }
    .issue-triage-done h2 { margin-top: 7px; font-size: clamp(25px,3vw,34px); }
    .issue-triage-done p { margin-top: 8px; color: var(--muted); }
    .issue-triage-done .issue-triage-actions { grid-column: 1 / -1; width: 100%; }
    .issue-triage-more { width: min(780px,100%); justify-self: center; border-top: 1px solid var(--line); }
    .issue-triage-more > summary { cursor: pointer; list-style: none; padding: 16px 2px; color: var(--muted); font-size: 13px; font-weight: 850; }
    .issue-triage-more > summary::-webkit-details-marker { display: none; }
    .issue-triage-more > summary::after { content: "+"; float: right; color: var(--gold-ink); font-size: 18px; }
    .issue-triage-more[open] > summary::after { content: "−"; }
    .issue-triage-more-body { display: grid; gap: 16px; border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 16px; background: var(--panel); }
    .issue-detail-link { min-height: 52px; display: flex; align-items: center; justify-content: space-between; gap: 18px; margin-top: 13px; border-top: 1px solid var(--line); padding: 13px 2px 0; color: var(--muted); text-decoration: none; }
    .issue-detail-link strong { color: var(--ink); }
    .issue-detail-link:hover strong { color: var(--gold-ink); }
    .issue-message-form { display: grid; gap: 20px; }
    .issue-message-body { display: grid; gap: 8px; color: var(--gold-ink); font-size: 12px; font-weight: 850; letter-spacing: .08em; text-transform: uppercase; }
    .issue-message-body textarea { min-height: 130px; resize: vertical; border: 1px solid var(--line); border-radius: var(--radius-xs); padding: 14px; background: #fffefb; color: var(--ink); font: 500 15px/1.5 var(--font-sans); letter-spacing: 0; text-transform: none; }
    .issue-message-body textarea:focus { outline: 3px solid rgba(200,153,63,.22); border-color: var(--gold); }
    .issue-resolution-propose { display: flex; align-items: center; justify-content: space-between; gap: 20px; border-top: 1px solid var(--line); padding-top: 22px; }
    .issue-resolution-propose span { display: grid; gap: 3px; }
    .issue-resolution-propose small { color: var(--muted); }
    .issue-resolution-propose button { min-height: 44px; }
    .issue-resident-page { width: min(980px,100%); gap: 28px; padding-bottom: 48px; }
    .issue-created-note { width: min(780px,100%); justify-self: center; display: grid; grid-template-columns: 34px minmax(0,1fr); gap: 3px 11px; align-items: center; border: 1px solid rgba(47,107,74,.26); border-radius: var(--radius-sm); padding: 13px 16px; background: rgba(47,107,74,.07); color: var(--leaf); }
    .issue-created-note > span { grid-row: 1 / span 2; width: 34px; height: 34px; display: grid; place-items: center; border-radius: 50%; background: var(--leaf); color: #fff; font-weight: 900; }
    .issue-created-note strong { font-size: 14px; }
    .issue-created-note small { color: var(--muted); font-size: 12.5px; }
    .issue-resident-context { display: grid; gap: 14px; border-bottom: 1px solid var(--line); padding-bottom: 24px; }
    .issue-resident-title-row { display: flex; align-items: end; justify-content: space-between; gap: 22px; }
    .issue-resident-title-row h1 { font-size: clamp(38px,5vw,58px); text-wrap: balance; overflow-wrap: anywhere; }
    .issue-resident-title-row p { margin-top: 8px; color: var(--muted); font-weight: 650; }
    .issue-resident-task { width: min(780px,100%); justify-self: center; display: grid; gap: 22px; border: 1px solid rgba(32,37,31,.13); border-radius: 16px; padding: clamp(24px,4vw,40px); background: var(--panel); box-shadow: 0 18px 50px rgba(38,34,25,.08); }
    .issue-resident-task h2 { max-width: 680px; font-size: clamp(26px,3.4vw,38px); line-height: 1.2; }
    .issue-resident-task > p { color: var(--muted); }
    .issue-answer-form { display: grid; gap: 16px; }
    .issue-answer-form > label:first-of-type { display: grid; gap: 8px; color: var(--ink); font-size: 14px; font-weight: 800; }
    .issue-answer-form textarea { min-height: 150px; resize: vertical; border: 1px solid var(--line); border-radius: var(--radius-xs); padding: 14px; background: #fffefb; color: var(--ink); font: 500 15px/1.5 var(--font-sans); }
    .issue-answer-form > button { min-height: 48px; border: 1px solid var(--ink); border-radius: var(--radius-xs); background: var(--ink); color: white; font: inherit; font-weight: 850; cursor: pointer; }
    .issue-resident-task.waiting { grid-template-columns: minmax(0,1fr); align-items: start; }
    .issue-resident-task.done { grid-template-columns: auto minmax(0,1fr); align-items: start; }
    .issue-resolution-actions { display: flex; align-items: center; gap: 12px; }
    .issue-resolution-actions form { margin: 0; }
    .issue-resolution-actions button { min-height: 46px; border: 1px solid var(--ink); border-radius: var(--radius-xs); padding: 10px 20px; background: var(--ink); color: white; font: inherit; font-weight: 850; cursor: pointer; }
    .issue-resolution-actions button.ghost { background: transparent; color: var(--ink); }
    .issue-resident-report { width: min(780px,100%); justify-self: center; display: grid; gap: 15px; border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 20px; background: var(--panel); box-shadow: var(--shadow-panel); }
    .issue-resident-report-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
    .issue-resident-report-head h2 { font-size: 22px; }
    .issue-resident-report-meta { display: flex; flex-wrap: wrap; gap: 7px; }
    .issue-resident-report .issue-description p { color: var(--ink); line-height: 1.6; white-space: pre-wrap; overflow-wrap: anywhere; }
    .issue-resident-report .attachment-delete { top: 2px; right: 2px; }
    .issue-resident-report .attachment-delete button { position: relative; width: 44px; height: 44px; border-color: transparent; background: transparent; color: transparent; opacity: .7; }
    .issue-resident-report .attachment-delete button::before { content: "\00d7"; position: absolute; top: 50%; left: 50%; width: 24px; height: 24px; display: grid; place-items: center; border-radius: 999px; background: rgba(255,254,251,.82); color: var(--muted); font-size: 17px; line-height: 1; transform: translate(-50%,-50%); }
    .issue-resident-report .attachment-delete button:hover, .issue-resident-report .attachment-delete button:focus-visible { border-color: transparent; background: transparent; color: transparent; opacity: 1; }
    .issue-resident-report .attachment-delete button:hover::before, .issue-resident-report .attachment-delete button:focus-visible::before { background: var(--panel); box-shadow: 0 0 0 1px rgba(158,42,43,.28); color: #9e2a2b; }
    .issue-resident-history { width: min(780px,100%); justify-self: center; border-top: 1px solid var(--line); }
    .issue-resident-history > summary { cursor: pointer; list-style: none; padding: 17px 2px; color: var(--ink); font-size: 14px; font-weight: 850; }
    .issue-resident-history > summary::-webkit-details-marker { display: none; }
    .issue-resident-history > summary::after { content: "›"; float: right; color: var(--gold-ink); font-size: 19px; }
    .issue-resident-history[open] > summary::after { transform: rotate(90deg); }
    .issue-resident-history-body { display: grid; gap: 18px; border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 18px; background: var(--panel); }
    .issue-subtools { border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel); overflow: hidden; }
    .issue-subtools > summary { cursor: pointer; list-style: none; padding: 10px 11px; color: var(--ink); font-size: 12.5px; font-weight: 850; }
    .issue-subtools > summary::-webkit-details-marker { display: none; }
    .issue-subtools > summary::after { content: "›"; float: right; color: var(--gold-ink); font-size: 20px; line-height: .8; }
    .issue-subtools[open] > summary { border-bottom: 1px solid var(--line); }
    .issue-subtools[open] > summary::after { transform: rotate(90deg); }
    .issue-subtools-body { display: grid; gap: 10px; padding: 11px; }
    .comment-thread { display: grid; gap: 8px; border-top: 1px dashed var(--line); padding-top: 10px; }
    .comment { display: grid; gap: 5px; border-left: 3px solid rgba(200,153,63,.35); padding-left: 9px; color: var(--ink); }
    .comment-head { display: flex; align-items: center; justify-content: space-between; gap: 10px; }
    .comment-meta { color: var(--soft); font-size: 12px; font-weight: 800; }
    .comment-delete { margin: 0; }
    .comment-delete button { min-height: 28px; border: 1px solid var(--line); border-radius: var(--radius-xs); padding: 4px 8px; background: transparent; color: var(--soft); font: inherit; font-size: 12px; font-weight: 800; cursor: pointer; }
    .comment-delete button:hover { border-color: #9e2a2b; color: #9e2a2b; }
    .comment-form { display: grid; gap: 8px; }
    .comment-form textarea { min-height: 84px; }
    .comment-form button { justify-self: start; min-height: 38px; border: 1px solid var(--ink); border-radius: var(--radius-xs); padding: 8px 12px; background: var(--ink); color: #fff; font: inherit; font-size: 13px; font-weight: 800; cursor: pointer; }
    .comment-form .file-control { max-width: 420px; min-height: 38px; padding: 8px 10px; }
    .attachment-strip { display: grid; grid-template-columns: repeat(auto-fill,minmax(112px,1fr)); gap: 8px; }
    .attachment-item { position: relative; min-width: 0; }
    .attachment-open { width: 100%; min-height: 100%; display: grid; grid-template-rows: auto minmax(34px,auto); gap: 6px; border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 7px; background: var(--panel-soft); color: var(--ink); text-decoration: none; font: inherit; text-align: left; cursor: pointer; }
    .attachment-open:hover { border-color: var(--gold); }
    .attachment-open img { display: block; width: 100%; aspect-ratio: 4 / 3; object-fit: cover; border-radius: var(--radius-xs); background: #ede5d6; }
    .attachment-file-icon { width: 100%; aspect-ratio: 4 / 3; display: grid; place-items: center; border-radius: var(--radius-xs); background: rgba(200,153,63,.14); color: var(--gold-ink); font-size: 12px; font-weight: 900; letter-spacing: .08em; }
    .attachment-name { min-width: 0; color: var(--muted); font-size: 12px; font-weight: 700; line-height: 1.25; overflow: hidden; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow-wrap: anywhere; }
    .attachment-delete { position: absolute; top: 5px; right: 5px; margin: 0; }
    .attachment-delete button { width: 28px; height: 28px; display: grid; place-items: center; border: 1px solid rgba(32,37,31,.18); border-radius: var(--radius-xs); background: rgba(255,254,251,.94); color: var(--ink); font-size: 18px; line-height: 1; cursor: pointer; }
    .attachment-delete button:hover { border-color: #9e2a2b; color: #9e2a2b; }
    .attachment-lightbox { position: fixed; inset: 0; z-index: 200; display: none; grid-template-columns: 56px minmax(0,1fr) 56px; grid-template-rows: 56px minmax(0,1fr) auto; align-items: center; gap: 12px; padding: 16px; background: rgba(23,32,25,.88); color: #fff; }
    .attachment-lightbox.open { display: grid; }
    .attachment-lightbox figure { grid-column: 2; grid-row: 2; display: grid; place-items: center; gap: 10px; min-width: 0; min-height: 0; margin: 0; }
    .attachment-lightbox img { max-width: 100%; max-height: calc(100vh - 150px); object-fit: contain; border-radius: var(--radius-sm); box-shadow: 0 24px 70px rgba(0,0,0,.38); background: #111; }
    .attachment-lightbox figcaption { color: rgba(255,255,255,.82); font-size: 13px; text-align: center; overflow-wrap: anywhere; }
    .lightbox-close, .lightbox-prev, .lightbox-next { border: 1px solid rgba(255,255,255,.28); border-radius: var(--radius-xs); background: rgba(255,255,255,.08); color: #fff; font: inherit; font-weight: 900; cursor: pointer; }
    .lightbox-close { grid-column: 3; grid-row: 1; justify-self: end; width: 42px; height: 42px; font-size: 24px; }
    .lightbox-prev, .lightbox-next { width: 46px; height: 70px; font-size: 28px; }
    .lightbox-prev { grid-column: 1; grid-row: 2; }
    .lightbox-next { grid-column: 3; grid-row: 2; }
    .dialog { border: 1px solid var(--line); border-radius: var(--radius-md); padding: 0; width: min(680px, calc(100vw - 28px)); max-height: min(860px, calc(100dvh - 28px)); overflow: hidden; color: var(--ink); background: var(--panel); box-shadow: var(--shadow-dialog); }
    .dialog[open] { display: grid; grid-template-rows: auto minmax(0,1fr) auto; }
    .dialog::backdrop { background: rgba(23,32,25,.42); }
    .dialog > form { min-width: 0; min-height: 0; grid-column: 1; grid-row: 1 / -1; margin: 0; display: grid; grid-template-rows: auto minmax(0,1fr) auto; max-height: inherit; overflow: hidden; }
    .dialog-head { display: flex; justify-content: space-between; align-items: center; gap: 14px; padding: 20px 22px; border-bottom: 1px solid var(--line); }
    .dialog-head h2 { font-size: 25px; }
    .dialog-close { width: 34px; height: 34px; border: 1px solid var(--line); border-radius: var(--radius-xs); background: var(--panel-soft); color: var(--ink); font-size: 22px; line-height: 1; cursor: pointer; }
    .dialog-body { min-width: 0; min-height: 0; display: grid; gap: 14px; padding: 20px 22px 22px; overflow: auto; overscroll-behavior: contain; }
    .dialog-footer { position: relative; z-index: 2; display: flex; align-items: center; justify-content: flex-end; gap: 12px; border-top: 1px solid var(--line); padding: 14px 22px max(14px,env(safe-area-inset-bottom)); background: rgba(255,254,251,.98); }
    .dialog-body > button:last-child { position: sticky; bottom: -1px; z-index: 2; box-shadow: 0 -12px 0 12px var(--panel), 0 -10px 18px rgba(255,254,251,.92); }
    .dialog-grid { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 12px; }
    .dialog-grid .full { grid-column: 1 / -1; }
    .dialog-optional { border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel-soft); overflow: hidden; }
    .dialog-optional > summary { cursor: pointer; list-style: none; padding: 12px 13px; color: var(--ink); font-size: 13.5px; font-weight: 800; }
    .dialog-optional > summary::-webkit-details-marker { display: none; }
    .dialog-optional > summary::after { content: "›"; float: right; color: var(--gold-ink); font-size: 20px; line-height: .8; }
    .dialog-optional[open] > summary { border-bottom: 1px solid var(--line); }
    .dialog-optional[open] > summary::after { transform: rotate(90deg); }
    .dialog-optional-grid { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 12px; padding: 12px 13px 14px; }
    .dialog-optional-grid .full { grid-column: 1 / -1; }
    @media (max-width: 600px) {
      .dialog-head { padding: 14px; }
      .dialog-close { width: 44px; height: 44px; flex: 0 0 auto; }
      .dialog-body { padding: 16px 14px 18px; }
      .dialog-footer { padding: 12px 14px max(12px,env(safe-area-inset-bottom)); }
      .dialog-footer .button { width: 100%; min-height: 44px; }
    }
    .release-dialog { width: min(920px, calc(100vw - 28px)); }
    .release-history { display: grid; grid-template-columns: 210px minmax(0,1fr); gap: 20px; max-height: min(72vh, 720px); }
    .release-rail { display: grid; align-content: start; gap: 8px; border-right: 1px solid var(--line); padding-right: 16px; overflow: auto; }
    .release-rail a { display: grid; gap: 3px; border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 10px 12px; color: var(--ink); background: var(--panel-soft); text-decoration: none; }
    .release-rail a:first-child { border-color: rgba(200,153,63,.55); background: rgba(200,153,63,.1); }
    .release-rail strong { font-size: 14px; }
    .release-rail span { color: var(--muted); font-size: 12px; font-weight: 700; }
    .release-body { min-width: 0; overflow: auto; padding-right: 4px; }
    .release-entry { display: grid; gap: 14px; padding-bottom: 24px; }
    .release-entry + .release-entry { border-top: 1px solid var(--line); padding-top: 24px; }
    .release-entry h3 { font-size: clamp(24px,3vw,34px); }
    .release-meta { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; color: var(--muted); font-size: 13px; font-weight: 800; }
    .release-items { display: grid; gap: 10px; }
    .release-item { display: grid; grid-template-columns: 110px minmax(0,1fr); gap: 12px; border-top: 1px solid var(--line); padding-top: 11px; line-height: 1.45; }
    .release-item strong { color: var(--gold-ink); font-size: 12px; letter-spacing: .06em; text-transform: uppercase; }
    .release-item span { color: var(--muted); }
    textarea { width: 100%; border: 1px solid #e2dac9; border-radius: var(--radius-xs); min-height: 150px; padding: 10px 12px; color: var(--ink); background: #fffefb; resize: vertical; font: inherit; line-height: 1.45; }
    select { width: 100%; border: 1px solid #e2dac9; border-radius: var(--radius-xs); min-height: 42px; padding: 9px 12px; color: var(--ink); background: #fffefb; font: inherit; }
    .check-row { display: inline-flex; align-items: center; gap: 8px; min-height: 42px; color: var(--ink); font-size: 14px; font-weight: 700; letter-spacing: 0; text-transform: none; }
    .check-row input { width: auto; min-height: 0; }
    .metric-grid { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 16px; }
    .metric-card, .month-card { min-width: 0; background: var(--panel-soft); border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 16px; }
    .metric-label, .field-label, th { color: var(--gold-ink); font-size: 11px; font-weight: 800; text-transform: uppercase; letter-spacing: .06em; }
    .metric-value { display: block; margin-top: 7px; font-family: var(--font-serif); font-size: 26px; font-weight: 600; line-height: 1.08; }
    code, .mini { color: var(--soft); font-size: 12px; line-height: 1.35; overflow-wrap: anywhere; }
    .accounting { display: grid; gap: 16px; }
    .section-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; flex-wrap: wrap; }
    .parking-page-head { display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 18px; align-items: start; }
    .parking-primary-actions { display: flex; align-items: flex-start; justify-content: flex-end; gap: 10px; flex-wrap: wrap; }
    .parking-more { position: relative; }
    .parking-more summary { list-style: none; }
    .parking-more summary::-webkit-details-marker { display: none; }
    .parking-more-menu { position: absolute; right: 0; top: calc(100% + 8px); z-index: 20; width: min(270px,calc(100vw - 48px)); display: grid; gap: 8px; border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel); padding: 10px; box-shadow: var(--shadow-dialog); }
    .parking-more-menu .button, .parking-more-menu form, .parking-more-menu button { width: 100%; }
    .parking-guide { display: grid; grid-template-columns: minmax(210px,.65fr) minmax(0,1fr) auto; gap: 24px; align-items: center; border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel); padding: 22px 24px; box-shadow: var(--shadow-panel); }
    .parking-guide h2 { font-size: 22px; }
    .parking-guide p { margin-top: 4px; }
    .parking-guide-status { display: flex; align-items: center; justify-content: flex-end; gap: 8px; flex-wrap: wrap; }
    .parking-empty { display: grid; grid-template-columns: 190px minmax(0,1fr) auto; gap: 20px 26px; align-items: center; }
    .parking-empty-art { width: min(190px,100%); aspect-ratio: 1.25; justify-self: center; color: var(--gold); opacity: .92; }
    .parking-empty-art svg { width: 100%; height: 100%; display: block; stroke: currentColor; fill: none; stroke-width: 1.7; stroke-linecap: round; stroke-linejoin: round; }
    .parking-empty-art .soft-fill { fill: rgba(200,153,63,.11); stroke: none; }
    .parking-empty-copy { display: grid; gap: 8px; min-width: 0; }
    .parking-empty-copy h2 { font-size: clamp(28px,3.4vw,38px); }
    .parking-empty-actions { display: flex; flex-wrap: wrap; justify-content: flex-end; gap: 10px; align-self: center; }
    .parking-empty-steps { grid-column: 1 / -1; display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 8px; }
    .parking-empty-step { min-width: 0; min-height: 58px; display: grid; grid-template-columns: 36px minmax(0,1fr); gap: 10px; align-items: center; border: 1px solid var(--line); border-radius: var(--radius-xs); padding: 10px 12px; background: var(--panel-soft); }
    .parking-empty-step-number { width: 32px; height: 32px; border-radius: 50%; display: grid; place-items: center; background: rgba(200,153,63,.13); color: var(--gold-ink); font-weight: 900; }
    .parking-empty-step strong { display: block; font-family: var(--font-serif); font-size: 17px; line-height: 1.15; }
    .parking-empty-note { grid-column: 1 / -1; display: grid; grid-template-columns: 40px minmax(0,1fr); gap: 14px; align-items: center; border: 1px solid rgba(47,107,74,.18); border-radius: var(--radius-sm); background: rgba(47,107,74,.06); padding: 11px 14px; color: var(--muted); }
    .parking-empty-note-icon { width: 40px; height: 40px; border-radius: 50%; display: grid; place-items: center; background: rgba(47,107,74,.11); color: var(--leaf); }
    .parking-empty-note-icon svg { width: 19px; height: 19px; stroke: currentColor; stroke-width: 2; fill: none; stroke-linecap: round; stroke-linejoin: round; }
    .parking-empty-note strong { display: block; color: var(--ink); }
    .bar { height: 9px; border-radius: var(--radius-pill); background: #ece5d6; overflow: hidden; }
    .bar span { display: block; height: 100%; min-width: 2px; border-radius: inherit; background: var(--gold); }
    .amount { font-weight: 800; font-variant-numeric: tabular-nums; }
    .parking-workspace { display: grid; grid-template-columns: minmax(340px,.42fr) minmax(0,.58fr); gap: 18px; align-items: start; }
    .parking-assistant { display: grid; gap: 18px; align-content: start; }
    .parking-stepper { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 0; align-items: center; }
    .parking-step { position: relative; display: grid; grid-template-columns: 44px minmax(0,1fr); align-items: center; gap: 12px; color: var(--soft); font-size: 13px; font-weight: 800; text-align: left; }
    .parking-step strong, .parking-step small { display: block; }
    .parking-step small { margin-top: 2px; color: var(--soft); font-size: 12px; font-weight: 500; line-height: 1.35; }
    .parking-step::before { content: ""; position: absolute; top: 22px; left: 54px; right: 14px; height: 1px; background: var(--line); transform: translateX(100%); }
    .parking-step:last-child::before { display: none; }
    .parking-step-number { position: relative; z-index: 1; width: 44px; height: 44px; border-radius: 50%; display: grid; place-items: center; border: 1px solid var(--line); background: var(--panel-soft); color: var(--ink); font-weight: 900; }
    .parking-step.active { color: var(--gold-ink); }
    .parking-step.active .parking-step-number { border-color: var(--gold); background: var(--gold); color: #fff; }
    .parking-queue-head { display: flex; align-items: center; justify-content: space-between; gap: 14px; }
    .parking-queue-head h2 { font-size: 22px; }
    .parking-month-queue { display: grid; gap: 10px; }
    .parking-month-row { min-height: 68px; display: grid; grid-template-columns: 44px minmax(0,1fr) auto auto auto; gap: 12px; align-items: center; border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel-soft); padding: 11px 12px; color: inherit; text-decoration: none; }
    .parking-month-row:hover { border-color: var(--gold); background: #fffefb; }
    .parking-month-row strong { display: block; overflow-wrap: anywhere; }
    .parking-month-row .mini { display: block; margin-top: 3px; }
    .parking-month-icon { width: 44px; height: 44px; border-radius: var(--radius-xs); display: grid; place-items: center; background: rgba(200,153,63,.11); color: var(--gold-ink); }
    .parking-month-icon svg { width: 21px; height: 21px; stroke: currentColor; stroke-width: 1.9; fill: none; stroke-linecap: round; stroke-linejoin: round; }
    .parking-queue-action { min-height: 34px; display: inline-flex; align-items: center; justify-content: center; border: 1px solid var(--line); border-radius: var(--radius-xs); background: var(--panel); padding: 6px 12px; font-size: 12px; font-weight: 900; white-space: nowrap; }
    .parking-utility { border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel-soft); padding: 15px 16px; }
    .parking-utility summary { cursor: pointer; color: var(--ink); font-family: var(--font-serif); font-size: 18px; font-weight: 700; }
    .parking-utility-body { display: grid; gap: 12px; margin-top: 12px; }
    .parking-utility .metric-grid { grid-template-columns: 1fr; gap: 8px; }
    .parking-utility .metric-card { padding: 12px; }
    .parking-utility .metric-value { font-size: 18px; }
    .parking-detail-stack { position: sticky; top: 82px; display: grid; gap: 10px; align-self: start; }
    .parking-month-detail { display: none; gap: 18px; border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel); box-shadow: var(--shadow-panel); scroll-margin-top: 82px; overflow: hidden; }
    .parking-month-detail:first-child { display: grid; }
    .parking-month-detail:target { display: grid; }
    .parking-detail-stack:has(.parking-month-detail:target) .parking-month-detail:first-child:not(:target) { display: none; }
    .parking-detail-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 14px; padding: 22px 22px 0; }
    .parking-detail-head strong { display: block; margin-top: 4px; font-family: var(--font-serif); font-size: 31px; line-height: 1; }
    .parking-tabs { display: flex; flex-wrap: wrap; gap: 7px; padding: 0 22px; }
    .parking-tabs a, .parking-tabs span { min-height: 31px; display: inline-flex; align-items: center; justify-content: center; border: 1px solid var(--line); border-radius: var(--radius-xs); background: var(--panel-soft); padding: 6px 10px; color: var(--ink); text-decoration: none; font-size: 12px; font-weight: 800; }
    .parking-tabs span:first-child { border-color: var(--gold); color: var(--gold-ink); background: rgba(200,153,63,.1); }
    .parking-tabs a:hover { border-color: var(--gold); color: var(--gold-ink); }
    .parking-breakdown { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 8px; margin: 0; padding: 0 22px; }
    .parking-breakdown div { border: 1px solid var(--line); border-radius: var(--radius-xs); background: var(--panel); padding: 10px; }
    .parking-breakdown dt { color: var(--soft); font-size: 11px; font-weight: 900; letter-spacing: .06em; text-transform: uppercase; }
    .parking-breakdown dd { margin: 5px 0 0; font-weight: 800; font-variant-numeric: tabular-nums; }
    .parking-detail-actions { display: block; border-top: 1px solid var(--line); padding: 4px 22px 8px; }
    .parking-receipts { padding: 18px; display: grid; gap: 12px; align-content: start; }
    .pay-sec { display: grid; gap: 13px; padding: 16px 0 10px; }
    .pay-head { display: flex; align-items: baseline; gap: 12px; }
    .pay-head h3 { margin: 0; font-family: var(--font-serif); font-size: 21px; }
    .pay-state { font-size: 11.5px; font-weight: 800; letter-spacing: .08em; text-transform: uppercase; color: var(--gold-ink); }
    .pay-state.paid { color: var(--leaf); }
    .pay-row { display: flex; align-items: center; justify-content: space-between; gap: 16px; flex-wrap: wrap; }
    .pay-row .muted { margin: 0; max-width: 40ch; }
    .pay-row .button, .pay-actions .button { width: auto; margin: 0; padding: 11px 22px; }
    .pay-settled { display: flex; align-items: center; gap: 13px; flex-wrap: wrap; border: 1px solid rgba(47,107,74,.25); border-radius: var(--radius-sm); background: rgba(47,107,74,.07); padding: 13px 15px; }
    .pay-check { flex: 0 0 auto; width: 34px; height: 34px; color: var(--leaf); }
    .pay-check svg { width: 100%; height: 100%; }
    .pay-settled-meta { display: grid; gap: 2px; min-width: 0; }
    .pay-settled-meta strong { color: var(--ink); font-size: 14.5px; }
    .pay-settled-meta span { color: var(--muted); font-size: 12.5px; overflow-wrap: anywhere; }
    .pay-unmark { margin-left: auto; }
    .button.pay-ghost { width: auto; margin: 0; padding: 8px 14px; font-size: 13px; background: transparent; color: var(--leaf); border: 1px solid rgba(47,107,74,.4); }
    .button.pay-ghost:hover { background: rgba(47,107,74,.08); }
    .pay-details { border-top: 1px solid var(--line); padding-top: 11px; }
    .pay-details > summary { list-style: none; cursor: pointer; font-size: 13.5px; font-weight: 700; color: var(--gold-ink); user-select: none; display: inline-flex; align-items: center; gap: 7px; }
    .pay-details > summary::-webkit-details-marker { display: none; }
    .pay-details > summary::after { content: "▾"; font-size: 11px; transition: transform .15s ease; }
    .pay-details[open] > summary::after { transform: rotate(180deg); }
    .pay-details[open] > summary { margin-bottom: 11px; }
    .pay-fields { display: grid; grid-template-columns: 150px 170px minmax(0,1fr); gap: 11px; }
    .pay-fields label { font-size: 11px; }
    .pay-fields input { min-height: 40px; }
    .pay-actions { display: flex; justify-content: flex-end; margin-top: 11px; }
    .payment-fields { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 8px; align-items: end; }
    .payment-fields .full { grid-column: 1 / -1; }
    .payment-fields label { color: var(--gold-ink); font-size: 11px; font-weight: 800; letter-spacing: .06em; text-transform: uppercase; }
    .payment-fields input { margin-top: 5px; }
    .parking-dropzone { min-height: 112px; display: grid; place-items: center; border: 1px dashed #d8ccb8; border-radius: var(--radius-sm); background: var(--panel-soft); color: var(--muted); text-align: center; padding: 16px; }
    .parking-dropzone strong { display: block; color: var(--ink); }
    .parking-detail-foot { display: flex; justify-content: space-between; gap: 12px; flex-wrap: wrap; border-top: 1px solid var(--line); padding: 12px 22px; color: var(--soft); font-size: 12px; }
    .parking-receipts .attachment-strip { grid-template-columns: repeat(auto-fill,minmax(96px,1fr)); }
    .parking-switcher { width: min(520px,100%); display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel-soft); padding: 4px; }
    .parking-switcher a { min-height: 42px; display: grid; place-items: center; border-radius: 8px; color: var(--ink); font-weight: 850; text-decoration: none; }
    .parking-switcher a:first-child { background: var(--ink); color: #fff; box-shadow: 0 5px 14px rgba(30,39,31,.12); }
    .parking-live { display: grid; gap: 16px; scroll-margin-top: 88px; }
    .parking-live-head { display: flex; justify-content: space-between; align-items: flex-start; gap: 18px; }
    .parking-live-head h2 { margin-top: 4px; font-size: clamp(26px,3vw,34px); }
    .parking-live-head p { margin-top: 4px; }
    .parking-live .pill.live-surplus { background: rgba(200,153,63,.16); color: #8a6a1f; border: 1px solid rgba(200,153,63,.32); }
    .parking-live .pill.live-manual { background: rgba(76,103,138,.12); color: #365475; border: 1px solid rgba(76,103,138,.24); }
    .parking-live .pill.live-idle { background: var(--panel-soft); color: var(--muted); border: 1px solid var(--line); }
    .parking-live-facts { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel-soft); overflow: hidden; }
    .parking-live-facts > span { min-height: 68px; display: grid; align-content: center; gap: 3px; padding: 10px 15px; border-left: 1px solid var(--line); }
    .parking-live-facts > span:first-child { border-left: 0; }
    .parking-live-facts small, .parking-current-facts small { color: var(--soft); font-size: 10.5px; font-weight: 850; letter-spacing: .06em; text-transform: uppercase; }
    .parking-live-facts strong { font-family: var(--font-serif); font-size: 22px; }
    .parking-live .battery-full { color: var(--leaf); }
    .parking-live .battery-partial { color: #93701d; }
    .parking-live .battery-low { color: #a04545; }
    .parking-live-actions { display: flex; flex-wrap: wrap; gap: 9px; align-items: flex-start; }
    .parking-live-actions form { margin: 0; }
    .parking-manual { position: relative; }
    .parking-manual > summary { list-style: none; }
    .parking-manual > summary::-webkit-details-marker { display: none; }
    .parking-manual > form { position: absolute; z-index: 10; top: calc(100% + 7px); left: 0; width: max-content; border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel); padding: 8px; box-shadow: var(--shadow-dialog); }
    .parking-live-details, .parking-utility, .parking-month-disclosure { border-top: 1px solid var(--line); padding-top: 12px; }
    .parking-live-details > summary, .parking-utility > summary, .parking-month-disclosure > summary { min-height: 42px; display: flex; align-items: center; cursor: pointer; color: var(--gold-ink); font-weight: 850; }
    .parking-live-details-body { display: grid; gap: 13px; padding-top: 9px; }
    .parking-split-list { display: grid; gap: 7px; }
    .parking-split-list p { display: grid; grid-template-columns: 70px repeat(2,minmax(0,1fr)); gap: 8px; margin: 0; color: var(--muted); font-size: 12.5px; }
    .parking-split-list strong { color: var(--ink); }
    .parking-admin-state { display: flex; align-items: center; gap: 7px 12px; flex-wrap: wrap; border-radius: var(--radius-xs); background: var(--panel-soft); padding: 10px 12px; color: var(--muted); font-size: 12.5px; }
    .parking-session-list { display: grid; gap: 6px; margin-top: 8px; }
    .parking-session-row { display: flex; flex-wrap: wrap; align-items: center; gap: 10px; padding: 8px 10px; border: 1px solid var(--line); border-radius: 9px; background: var(--panel-soft); font-size: 13.5px; }
    .parking-session-row .pill.mode-surplus { background: rgba(200,153,63,.16); color: #8a6a1f; border: 1px solid rgba(200,153,63,.32); }
    .parking-session-row .pill.mode-normal { background: rgba(76,103,138,.12); color: #365475; border: 1px solid rgba(76,103,138,.24); }
    .parking-session-row .amount { margin-left: auto; }
    .parking-months-panel { display: grid; gap: 14px; scroll-margin-top: 88px; }
    .parking-months-head { display: flex; align-items: flex-end; justify-content: space-between; gap: 14px; }
    .parking-months-head h2 { font-size: clamp(27px,3vw,34px); }
    .parking-months-status { display: flex; gap: 7px; flex-wrap: wrap; justify-content: flex-end; }
    .parking-current-month { min-height: 132px; display: grid; grid-template-columns: minmax(150px,1fr) minmax(220px,.8fr) auto 18px; gap: 18px; align-items: center; border: 1px solid rgba(200,153,63,.32); border-radius: var(--radius-sm); background: linear-gradient(120deg,rgba(200,153,63,.08),#fffefb); padding: 18px; color: inherit; text-decoration: none; }
    .parking-current-month > span:first-child { display: grid; gap: 3px; }
    .parking-current-month > span:first-child strong { font-family: var(--font-serif); font-size: 22px; }
    .parking-current-month > span:first-child b { font-family: var(--font-serif); font-size: 32px; line-height: 1; }
    .parking-current-facts { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 0; }
    .parking-current-facts > span { display: grid; gap: 2px; border-left: 1px solid var(--line); padding-left: 18px; }
    .parking-current-facts strong { font-size: 15px; }
    .parking-row-arrow { color: var(--gold-ink); font-size: 28px; line-height: 1; }
    .parking-month-list { display: grid; border: 1px solid var(--line); border-radius: var(--radius-sm); overflow: hidden; }
    .parking-compact-month { min-height: 62px; display: grid; grid-template-columns: minmax(0,1fr) auto auto 18px; gap: 14px; align-items: center; border-top: 1px solid var(--line); padding: 10px 14px; color: inherit; text-decoration: none; background: #fffefb; }
    .parking-compact-month:first-child { border-top: 0; }
    .parking-compact-month:hover, .parking-current-month:hover { border-color: var(--gold); }
    .parking-compact-month > span:first-child { display: grid; gap: 2px; }
    .parking-compact-month small { color: var(--soft); font-size: 11.5px; }
    .parking-months-panel .parking-utility { border-right: 0; border-bottom: 0; border-left: 0; border-radius: 0; background: transparent; padding: 8px 0 0; }
    .parking-months-panel .parking-utility summary { font-family: var(--font-sans); font-size: 13px; }
    .parking-month-page { gap: 16px; }
    .parking-month-hero { display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 16px; align-items: end; }
    .parking-month-hero h1 { margin-top: 3px; }
    .parking-month-hero-actions { display: flex; gap: 8px; flex-wrap: wrap; justify-content: flex-end; }
    .parking-month-summary { display: grid; gap: 18px; }
    .parking-month-total { display: flex; justify-content: space-between; align-items: flex-start; gap: 16px; }
    .parking-month-total strong { display: block; margin-top: 4px; font-family: var(--font-serif); font-size: clamp(38px,6vw,56px); line-height: 1; }
    .parking-month-essential { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); border-top: 1px solid var(--line); border-bottom: 1px solid var(--line); }
    .parking-month-essential > div { display: grid; gap: 3px; padding: 14px; border-left: 1px solid var(--line); }
    .parking-month-essential > div:first-child { border-left: 0; }
    .parking-month-essential dt { color: var(--soft); font-size: 10.5px; font-weight: 850; letter-spacing: .06em; text-transform: uppercase; }
    .parking-month-essential dd { margin: 0; font-size: 17px; font-weight: 850; }
    .parking-month-payment { display: grid; gap: 10px; }
    .parking-month-payment .pay-row { align-items: center; }
    .parking-month-payment .pay-row .button { min-width: 220px; }
    .parking-cost-list { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 0 24px; margin: 8px 0 0; }
    .parking-cost-list div { display: flex; justify-content: space-between; gap: 12px; border-bottom: 1px solid var(--line); padding: 10px 0; }
    .parking-cost-list dt { color: var(--muted); }
    .parking-cost-list dd { margin: 0; font-weight: 850; }
    .parking-hour-list { display: grid; gap: 7px; margin-top: 8px; }
    .parking-hour-row { border: 1px solid var(--line); border-radius: var(--radius-xs); background: var(--panel-soft); overflow: hidden; }
    .parking-hour-row > summary { min-height: 52px; display: grid; grid-template-columns: minmax(0,1fr) auto auto; gap: 12px; align-items: center; cursor: pointer; list-style: none; padding: 9px 12px; }
    .parking-hour-row > summary::-webkit-details-marker { display: none; }
    .parking-hour-row > summary strong { font-variant-numeric: tabular-nums; }
    .parking-hour-details { display: grid; grid-template-columns: repeat(4,minmax(0,1fr)); gap: 8px; border-top: 1px solid var(--line); padding: 10px 12px; }
    .parking-hour-details div { display: grid; gap: 2px; }
    .parking-hour-details dt { color: var(--soft); font-size: 10px; font-weight: 850; text-transform: uppercase; }
    .parking-hour-details dd { margin: 0; font-size: 12.5px; font-weight: 750; }
    .sr-only { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip: rect(0,0,0,0); white-space: nowrap; border: 0; }
    .table-wrap { max-width: 100%; overflow-x: auto; -webkit-overflow-scrolling: touch; border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel); }
    .table-wrap:focus-visible { outline: 3px solid var(--gold); outline-offset: 3px; }
    table { width: 100%; border-collapse: collapse; min-width: 980px; }
    th, td { padding: 13px 16px; border-bottom: 1px solid var(--line); vertical-align: middle; }
    th { text-align: left; background: rgba(251,248,240,.7); }
    tbody th { background: transparent; font-weight: 600; }
    td { font-size: 14px; }
    tbody tr:last-child td { border-bottom: 0; }
    .num { text-align: right; font-variant-numeric: tabular-nums; white-space: nowrap; }
    .month-cell strong { display: block; font-family: var(--font-serif); font-size: 17px; }
    .month-cell a { text-decoration: none; }
    .month-cell a:hover { color: var(--gold-ink); }
    .month-cell span { display: block; margin-top: 3px; color: var(--soft); font-size: 12px; }
    .row-actions { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
    .status-cell { display: grid; gap: 4px; }
    .payment-form { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }
    .payment-form input { width: auto; max-width: 128px; min-height: 34px; padding: 7px 9px; border: 1px solid var(--line); border-radius: var(--radius-xs); background: #fffefb; color: var(--ink); font: inherit; font-size: 12.5px; }
    .documents-page { gap: 20px; }
    .documents-screen .content-top .page-actions .button,
    .documents-screen .document-actions .button,
    .documents-screen .document-admin-tools-body .button { min-height: 44px; }
    .documents-screen .document-file-details summary,
    .documents-screen .document-admin-tools > summary,
    .documents-screen .document-versions summary { min-height: 44px; display: flex; align-items: center; }
    .document-page-head { display: flex; align-items: flex-end; justify-content: space-between; gap: 18px; }
    .document-page-head .lede { max-width: 720px; }
    .document-library { padding: 20px; display: grid; gap: 22px; }
    .document-library-head { display: flex; align-items: center; justify-content: space-between; gap: 14px; }
    .document-toolbar { display: grid; grid-template-columns: minmax(280px,1fr) minmax(190px,.3fr) auto; align-items: stretch; gap: 0; border: 1px solid var(--line); border-radius: var(--radius-sm); background: #fffefb; overflow: hidden; box-shadow: 0 8px 24px rgba(32,37,31,.04); }
    .document-search { position: relative; min-width: 0; }
    .document-search svg { position: absolute; left: 14px; top: 50%; width: 20px; height: 20px; transform: translateY(-50%); stroke: var(--muted); fill: none; stroke-width: 1.8; pointer-events: none; }
    .document-search input, .document-sort select { width: 100%; min-height: 50px; border: 0; border-radius: 0; background: transparent; color: var(--ink); font: inherit; }
    .document-search input { padding: 11px 14px 11px 44px; }
    .document-sort { position: relative; border-left: 1px solid var(--line); }
    .document-sort select { padding: 11px 36px 11px 14px; font-weight: 750; }
    .document-toolbar .button { min-height: 50px; margin: 0; border-width: 0 0 0 1px; border-radius: 0; }
    .document-toolbar .document-reset { display: inline-flex; align-items: center; justify-content: center; padding-inline: 14px; color: var(--gold-ink); background: var(--panel-soft); font-size: 13px; font-weight: 850; text-decoration: none; border-left: 1px solid var(--line); }
    .document-sections { display: grid; gap: 24px; }
    .document-section { display: grid; gap: 10px; }
    .document-section h3 { display: flex; align-items: center; justify-content: flex-start; gap: 9px; font-size: 20px; }
    .document-section h3 .section-count { font-family: var(--font-sans); font-size: 11px; font-weight: 900; color: var(--gold-ink); background: rgba(200,153,63,.12); border: 1px solid rgba(200,153,63,.24); border-radius: var(--radius-pill); padding: 3px 8px; white-space: nowrap; }
    .document-list { display: grid; gap: 10px; }
    .document-row { display: grid; grid-template-columns: 52px minmax(0,1fr) minmax(156px,auto); gap: 12px 16px; align-items: center; border: 1px solid var(--line); border-radius: var(--radius-sm); background: #fffefb; padding: 16px; box-shadow: 0 10px 28px rgba(32,37,31,.045); }
    .document-icon { width: 50px; height: 58px; display: grid; place-items: center; border: 1px solid rgba(200,153,63,.28); border-radius: 10px; color: var(--gold-ink); background: rgba(200,153,63,.09); }
    .document-icon svg { width: 24px; height: 24px; stroke: currentColor; fill: none; stroke-width: 1.7; stroke-linecap: round; stroke-linejoin: round; }
    .document-copy { min-width: 0; }
    .document-copy > strong { display: block; font-family: var(--font-serif); font-size: 19px; line-height: 1.2; overflow-wrap: anywhere; }
    .document-meta { margin-top: 7px; min-width: 0; display: flex; flex-wrap: wrap; gap: 6px 8px; align-items: center; color: var(--muted); font-size: 12.5px; }
    .document-meta-secondary { color: var(--soft); font-weight: 700; }
    .document-file-details { margin-top: 7px; color: var(--soft); font-size: 12px; }
    .document-file-details summary { cursor: pointer; font-weight: 750; }
    .document-file { display: block; max-width: min(520px,100%); margin-top: 5px; overflow-wrap: anywhere; }
    .document-actions { min-width: 156px; display: grid; gap: 7px; }
    .document-actions .button { width: 100%; justify-content: center; }
    .document-admin-tools { grid-column: 2 / -1; border-top: 1px solid var(--line); padding-top: 10px; }
    .document-admin-tools > summary, .document-versions summary { cursor: pointer; color: var(--gold-ink); font-size: 12.5px; font-weight: 850; }
    .document-admin-tools-body { margin-top: 9px; display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
    .document-versions { grid-column: 2 / -1; border-top: 1px solid var(--line); padding-top: 10px; }
    .document-empty { min-height: 190px; display: grid; place-items: center; text-align: center; border: 1px dashed rgba(200,153,63,.42); border-radius: var(--radius-sm); background: rgba(255,254,251,.68); padding: 28px; }
    .document-empty svg { width: 42px; height: 42px; margin-bottom: 11px; stroke: var(--gold-ink); fill: none; stroke-width: 1.45; }
    .document-empty h3 { font-size: 22px; }
    .document-empty p { max-width: 450px; margin: 6px auto 0; color: var(--muted); }
    .document-empty .button { margin-top: 14px; }
    .document-dialog-intro { margin: 0 0 14px; color: var(--muted); }
    .document-replace-context { display: grid; gap: 4px; border: 1px solid var(--line); border-radius: var(--radius-xs); background: var(--panel-soft); padding: 12px 14px; margin-bottom: 14px; }
    .document-replace-context strong { font-family: var(--font-serif); font-size: 17px; }
    .ballot-dialog textarea { min-height: 92px; }
    .ballot-dialog #ballot-options { min-height: 104px; }
    .ballot-dialog .dialog-body > button:last-child { position: static; box-shadow: none; margin-top: 2px; }
    .version-list { margin-top: 8px; display: grid; gap: 7px; }
    .version-row { display: flex; align-items: center; justify-content: space-between; gap: 10px; flex-wrap: wrap; color: var(--muted); font-size: 12.5px; }
    .vote-page { gap: 20px; }
    .vote-page-head { display: flex; align-items: flex-end; justify-content: space-between; gap: 18px; }
    .vote-page-head .lede { max-width: 680px; }
    .vote-overview { display: grid; grid-template-columns: 42px minmax(0,1fr); gap: 13px; align-items: center; border: 1px solid rgba(47,107,74,.22); border-radius: var(--radius-sm); background: rgba(47,107,74,.075); padding: 14px 16px; }
    .vote-overview.action { border-color: rgba(200,153,63,.34); background: rgba(200,153,63,.085); }
    .vote-overview-icon { width: 40px; height: 40px; display: grid; place-items: center; border-radius: 50%; color: var(--leaf); background: rgba(47,107,74,.12); font-size: 21px; font-weight: 900; }
    .vote-overview.action .vote-overview-icon { color: var(--gold-ink); background: rgba(200,153,63,.16); }
    .vote-overview strong { display: block; font-family: var(--font-serif); font-size: 19px; }
    .vote-overview p { margin: 3px 0 0; color: var(--muted); font-size: 13.5px; }
    .vote-library { display: grid; gap: 18px; padding: 20px; }
    .vote-list { display: grid; gap: 14px; }
    .vote-card { border: 1px solid var(--line); border-radius: var(--radius-sm); background: #fffefb; padding: 20px; display: grid; gap: 16px; box-shadow: 0 10px 28px rgba(32,37,31,.045); scroll-margin-top: 82px; }
    .vote-card.needs-action { border-color: rgba(200,153,63,.38); box-shadow: 0 12px 34px rgba(116,87,28,.08); }
    .vote-card form { display: grid; gap: 14px; }
    .vote-card-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 14px; }
    .vote-card-head > .pill { flex: 0 0 auto; }
    .vote-eyebrow { margin-bottom: 6px; color: var(--gold-ink); font-size: 11.5px; font-weight: 900; letter-spacing: .09em; text-transform: uppercase; }
    .vote-eyebrow.ok { color: var(--leaf); }
    .vote-card h3 { font-size: 23px; line-height: 1.15; overflow-wrap: anywhere; }
    .vote-deadline { width: fit-content; display: inline-flex; align-items: center; gap: 7px; border-radius: var(--radius-pill); padding: 6px 10px; color: var(--gold-ink); background: rgba(200,153,63,.11); font-size: 12.5px; font-weight: 850; }
    .vote-deadline svg { width: 16px; height: 16px; fill: none; stroke: currentColor; stroke-width: 1.8; }
    .vote-question { max-width: 820px; margin: 0; color: var(--ink); font-size: 17px; line-height: 1.45; }
    .vote-weight { display: flex; align-items: center; gap: 8px; margin: 0; color: var(--muted); font-size: 13px; font-weight: 750; }
    .vote-weight svg { width: 18px; height: 18px; fill: none; stroke: var(--gold-ink); stroke-width: 1.7; }
    .vote-options { display: grid; gap: 8px; }
    .vote-option { min-height: 50px; display: grid; grid-template-columns: auto minmax(0,1fr); gap: 12px; align-items: center; border: 1px solid var(--line); border-radius: var(--radius-xs); background: var(--panel-soft); padding: 11px 14px; color: var(--ink); font-size: 14px; font-weight: 800; cursor: pointer; }
    .vote-option:has(input:checked) { border-color: rgba(47,107,74,.42); background: rgba(47,107,74,.08); }
    .vote-option input { width: 18px; height: 18px; min-height: 0; accent-color: var(--leaf); }
    .vote-option-static { cursor: default; }
    .vote-actions { display: grid; grid-template-columns: minmax(0,1fr) minmax(210px,auto); gap: 12px; align-items: center; padding-top: 2px; }
    .vote-actions .button, .vote-management-actions .button, .vote-result-actions .button { min-height: 44px; justify-content: center; }
    .vote-current { color: var(--muted); font-size: 12.5px; }
    .vote-management { display: flex; align-items: center; justify-content: space-between; gap: 12px; border-top: 1px solid var(--line); padding-top: 14px; }
    .vote-management-copy strong { display: block; font-size: 13.5px; }
    .vote-management-copy span { display: block; margin-top: 3px; color: var(--muted); font-size: 12.5px; }
    .vote-management-actions { display: flex; gap: 8px; flex-wrap: wrap; }
    .vote-management-actions form { margin: 0; }
    .vote-details, .vote-live-result { border-top: 1px solid var(--line); padding-top: 12px; }
    .vote-details > summary, .vote-live-result > summary { cursor: pointer; color: var(--gold-ink); font-size: 12.5px; font-weight: 850; }
    .vote-detail-body { margin-top: 10px; display: flex; flex-wrap: wrap; gap: 7px 14px; color: var(--muted); font-size: 12.5px; }
    .vote-detail-options { flex-basis: 100%; display: flex; flex-wrap: wrap; gap: 6px; margin-top: 3px; }
    .vote-result { display: grid; gap: 14px; border-top: 1px solid var(--line); padding-top: 16px; }
    .vote-live-result .vote-result { margin-top: 12px; border-top: 0; padding-top: 0; }
    .vote-result-head { display: flex; align-items: flex-end; justify-content: space-between; gap: 14px; }
    .vote-result-head strong { display: block; margin-top: 4px; font-family: var(--font-serif); font-size: 24px; }
    .vote-result-actions { display: flex; gap: 8px; flex-wrap: wrap; }
    .vote-result-stats { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 8px; }
    .vote-result-stat { border: 1px solid var(--line); border-radius: var(--radius-xs); background: var(--panel-soft); padding: 10px 12px; }
    .vote-result-stat span { display: block; color: var(--soft); font-size: 10.5px; font-weight: 850; letter-spacing: .06em; text-transform: uppercase; }
    .vote-result-stat strong { display: block; margin-top: 4px; font-size: 15px; }
    .vote-result-rows { display: grid; gap: 9px; }
    .vote-result-row { display: grid; grid-template-columns: minmax(110px,.38fr) minmax(140px,1fr) auto; gap: 10px; align-items: center; color: var(--muted); font-size: 12.5px; }
    .vote-result-row strong { color: var(--ink); overflow-wrap: anywhere; }
    .vote-bar { height: 9px; border-radius: var(--radius-pill); background: #ece5d6; overflow: hidden; }
    .vote-bar span { display: block; height: 100%; min-width: 2px; border-radius: inherit; background: var(--gold); }
    .handover-page { display: grid; gap: 18px; }
    .handover-list { display: grid; gap: 14px; }
    .handover-card { gap: 16px; }
    .handover-head { display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 16px; align-items: start; }
    .handover-head h2 { margin-top: 6px; font-size: clamp(24px,3vw,34px); overflow-wrap: anywhere; }
    .handover-meta { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; color: var(--soft); font-size: 12.5px; font-weight: 800; }
    .handover-actions { display: flex; flex-wrap: wrap; gap: 8px; justify-content: flex-end; }
    .handover-actions form { margin: 0; }
    .handover-detail-grid { display: grid; grid-template-columns: repeat(4,minmax(0,1fr)); gap: 10px; }
    .handover-detail { border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel-soft); padding: 13px; min-width: 0; }
    .handover-detail h3 { font-size: 16px; }
    .handover-detail ul { list-style: none; padding: 0; margin: 10px 0 0; display: grid; gap: 9px; }
    .handover-detail li { display: grid; gap: 3px; color: var(--muted); font-size: 13px; line-height: 1.35; overflow-wrap: anywhere; }
    .handover-detail li strong { color: var(--ink); font-size: 14px; }
    .handover-detail li em { color: #9b5f54; font-style: normal; }
    .handover-detail p { margin-top: 9px; color: var(--muted); font-size: 13px; line-height: 1.4; }
    .handover-note { border-left: 3px solid var(--gold); padding: 4px 0 4px 13px; color: var(--muted); }
    .handover-note strong { color: var(--ink); }
    .handover-note p { margin-top: 4px; line-height: 1.45; overflow-wrap: anywhere; }
    .handover-scope { border-left: 3px solid rgba(47,107,74,.35); padding: 9px 12px; background: rgba(47,107,74,.06); color: var(--muted); font-size: 13px; line-height: 1.45; }
    .handover-foot { display: flex; flex-wrap: wrap; justify-content: space-between; gap: 10px; padding-top: 4px; color: var(--soft); font-size: 12px; }
    .handover-dialog textarea { min-height: 94px; }
    .handover-confirm-body { min-height: 100vh; background: radial-gradient(circle at top left, rgba(47,107,74,.12), transparent 34%), var(--paper); }
    .handover-confirm-page { min-height: 100vh; display: grid; place-items: center; padding: 24px; }
    .handover-confirm-card { width: min(760px, 100%); }
    .handover-confirm-card h1 { font-size: clamp(34px,6vw,56px); }
    .handover-confirm-summary { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 10px; margin: 12px 0; }
    .handover-confirm-summary div { border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel-soft); padding: 12px; }
    .handover-confirm-summary span { display: block; color: var(--gold-ink); font-size: 11px; font-weight: 900; letter-spacing: .06em; text-transform: uppercase; }
    .handover-confirm-summary strong { display: block; margin-top: 5px; overflow-wrap: anywhere; }
    .handover-confirm-form { display: grid; gap: 12px; margin-top: 14px; }
    .handover-page-head { display: grid; grid-template-columns: minmax(0,1fr) minmax(280px,430px); gap: 24px; align-items: end; }
    .handover-page-head .lede { margin-top: 4px; }
    .handover-page-head .handover-scope { margin: 0; }
    .handover-metrics { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 10px; }
    .handover-metrics > div { position: relative; border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel); padding: 14px 16px 14px 30px; box-shadow: 0 7px 20px rgba(31,42,32,.04); }
    .handover-metrics > div::before { content: ""; position: absolute; left: 14px; top: 20px; width: 7px; height: 7px; border-radius: 50%; background: var(--gold); }
    .handover-metrics > .ready::before { background: var(--leaf); }
    .handover-metrics > .done::before { background: #99968d; }
    .handover-metrics strong { display: block; font-family: var(--font-serif); font-size: 27px; line-height: 1; }
    .handover-metrics span { display: block; margin-top: 5px; color: var(--muted); font-size: 12.5px; font-weight: 750; }
    .handover-sections { display: grid; gap: 12px; }
    .handover-section { border-top: 1px solid var(--line); padding-top: 2px; }
    .handover-section > summary { min-height: 62px; display: grid; grid-template-columns: minmax(0,1fr) auto; align-items: center; gap: 12px; cursor: pointer; list-style: none; }
    .handover-section > summary::-webkit-details-marker { display: none; }
    .handover-section > summary > span:first-child { display: grid; gap: 3px; }
    .handover-section > summary strong { font-family: var(--font-serif); font-size: 21px; }
    .handover-section > summary small { color: var(--muted); font-size: 12.5px; }
    .handover-section-count { min-width: 32px; height: 32px; display: grid; place-items: center; border: 1px solid var(--line); border-radius: 50%; background: var(--panel-soft); color: var(--gold-ink); font-weight: 900; }
    /* Leere Abschnitte bleiben ruhig stehen und lassen sich nicht aufklappen:
       ein Aufklapper ohne Inhalt ist ein Klick ins Leere. */
    .handover-section.is-empty { min-height: 56px; display: grid; grid-template-columns: minmax(0,1fr) auto; align-items: center; gap: 12px; }
    .handover-section.is-empty > span:first-child { display: grid; gap: 3px; }
    .handover-section.is-empty strong { font-family: var(--font-serif); font-size: 21px; color: var(--muted); }
    .handover-section.is-empty small { color: var(--soft); font-size: 12.5px; }
    .handover-section.is-empty .handover-section-count { border-style: dashed; color: var(--soft); }
    @media (max-width: 1120px) { .handover-page-head { grid-template-columns: minmax(0,1fr); } }
    /* Uebergaben-Leerzustand: erklaert Zweck und Ablauf eines Protokolls,
       statt nur zu melden, dass noch keines existiert. */
    .handover-blank { display: grid; grid-template-columns: minmax(0,1.4fr) minmax(286px,.9fr); gap: 16px; }
    .handover-blank-main { display: grid; align-content: center; gap: 20px; border: 1px dashed rgba(200,153,63,.45); border-radius: var(--radius-sm); background: rgba(255,254,251,.7); padding: clamp(22px,3.2vw,36px); }
    .handover-blank-lead { display: grid; justify-items: start; gap: 13px; }
    .handover-blank-icon { width: 52px; height: 52px; display: grid; place-items: center; border: 1px solid rgba(200,153,63,.3); border-radius: 12px; background: rgba(200,153,63,.1); color: var(--gold-ink); }
    .handover-blank-icon svg { width: 26px; height: 26px; fill: none; stroke: currentColor; stroke-width: 1.5; stroke-linecap: round; stroke-linejoin: round; }
    .handover-blank-main h2 { font-size: clamp(25px,3vw,31px); }
    .handover-blank-main p { max-width: 58ch; color: var(--muted); font-size: 15px; line-height: 1.55; }
    .handover-blank-actions { display: flex; flex-wrap: wrap; gap: 9px; margin-top: 3px; }
    .handover-blank-actions .button { min-height: 44px; }
    .handover-blank-side { display: grid; align-content: start; gap: 13px; border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel); padding: 20px; box-shadow: var(--shadow-panel); }
    .handover-blank-side h2 { font-size: 12px; font-weight: 800; letter-spacing: .12em; text-transform: uppercase; color: var(--gold-ink); font-family: var(--font-sans); }
    .handover-blank-steps { display: grid; gap: 11px; margin: 0; padding: 0; list-style: none; counter-reset: handover-step; }
    .handover-blank-steps li { display: grid; grid-template-columns: 24px minmax(0,1fr); gap: 2px 11px; align-items: start; counter-increment: handover-step; }
    .handover-blank-steps li::before { content: counter(handover-step); grid-row: 1 / span 2; width: 24px; height: 24px; display: grid; place-items: center; border-radius: 50%; background: var(--ink); color: #fff; font-size: 12px; font-weight: 850; }
    .handover-blank-steps strong { grid-column: 2; font-size: 13.5px; }
    .handover-blank-steps span { grid-column: 2; color: var(--muted); font-size: 12.5px; line-height: 1.4; }
    .handover-blank-note { border-top: 1px solid var(--line); padding-top: 12px; color: var(--muted); font-size: 12.5px; line-height: 1.5; }
    .handover-blank-facts { grid-column: 1 / -1; display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 14px; margin: 0; padding: 0; list-style: none; }
    .handover-blank-facts li { display: grid; gap: 5px; border-top: 2px solid var(--ink); padding-top: 11px; }
    .handover-blank-facts strong { font-size: 13.5px; }
    .handover-blank-facts span { color: var(--muted); font-size: 12.5px; line-height: 1.5; }
    @media (min-width: 901px) {
      .handover-blank { min-height: max(430px, calc(100vh - 336px)); grid-template-rows: minmax(0,1fr) auto; }
    }
    .handover-card { gap: 13px; padding: 18px; }
    .handover-card .handover-head { grid-template-columns: minmax(0,1fr) auto; align-items: center; }
    .handover-card .handover-meta { margin-bottom: 5px; }
    .handover-card h3 { font-size: clamp(21px,2.5vw,28px); overflow-wrap: anywhere; }
    .handover-place { display: flex; flex-wrap: wrap; gap: 5px 12px; margin-top: 5px; color: var(--muted); font-size: 13px; font-weight: 700; }
    .handover-place span::before { content: "·"; margin-right: 12px; color: var(--soft); }
    .handover-confirm-progress { min-width: 74px; display: grid; justify-items: end; gap: 2px; }
    .handover-confirm-progress strong { font-family: var(--font-serif); font-size: 22px; }
    .handover-confirm-progress span { color: var(--muted); font-size: 11.5px; }
    .handover-next { display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 14px; align-items: center; border: 1px solid rgba(200,153,63,.22); border-radius: var(--radius-sm); padding: 12px 14px; background: linear-gradient(120deg,rgba(200,153,63,.11),rgba(255,255,255,.55)); }
    .handover-next.ready { border-color: rgba(47,107,74,.22); background: linear-gradient(120deg,rgba(47,107,74,.10),rgba(255,255,255,.55)); }
    .handover-next.done { border-color: var(--line); background: var(--panel-soft); }
    .handover-next div { display: grid; gap: 2px; }
    .handover-next span { color: var(--gold-ink); font-size: 10.5px; font-weight: 900; letter-spacing: .08em; text-transform: uppercase; }
    .handover-next strong { font-size: 14px; }
    .handover-next small { color: var(--muted); font-size: 12px; }
    .handover-details { border-top: 1px solid var(--line); padding-top: 10px; }
    .handover-details > summary { cursor: pointer; color: var(--gold-ink); font-size: 13px; font-weight: 850; }
    .handover-details[open] > summary { margin-bottom: 13px; }
    .handover-parties { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 8px; margin-bottom: 10px; }
    .handover-parties div { border: 1px solid var(--line); border-radius: var(--radius-xs); padding: 10px 12px; background: var(--panel-soft); min-width: 0; }
    .handover-parties span { display: block; color: var(--soft); font-size: 10.5px; font-weight: 850; text-transform: uppercase; letter-spacing: .06em; }
    .handover-parties strong { display: block; margin-top: 3px; font-size: 12.5px; overflow-wrap: anywhere; }
    .handover-detail h4 { font-size: 15px; }
    .handover-add-files { display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 10px; align-items: end; border-top: 1px solid var(--line); margin-top: 12px; padding-top: 12px; }
    .handover-add-files label { min-width: 0; }
    .handover-detail-actions { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 10px; margin-top: 12px; color: var(--soft); font-size: 11.5px; }
    .handover-form-step { display: grid; grid-template-columns: 34px minmax(0,1fr); gap: 10px; align-items: center; margin-top: 2px; }
    .handover-form-step > span { width: 34px; height: 34px; display: grid; place-items: center; border-radius: 50%; background: var(--ink); color: #fff; font-weight: 900; }
    .handover-form-step div { display: grid; gap: 1px; }
    .handover-form-step strong { font-family: var(--font-serif); font-size: 18px; }
    .handover-form-step small, .dialog-optional-copy { color: var(--muted); font-size: 12px; }
    .handover-dialog .dialog-body { grid-auto-rows: max-content; }
    .handover-dialog .dialog-optional { min-height: 42px; }
    .handover-dialog-submit { display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 12px; align-items: center; }
    .handover-dialog-submit span { color: var(--muted); font-size: 12px; }
    .handover-confirm-page { align-items: start; }
    .handover-confirm-card { width: min(780px, 100%); gap: 16px; }
    .handover-confirm-head { display: grid; gap: 7px; }
    .handover-confirm-head h1 { font-size: clamp(32px,6vw,50px); }
    .handover-confirm-head > p { display: flex; flex-wrap: wrap; gap: 4px 12px; color: var(--muted); }
    .handover-confirm-head > p span::before { content: "·"; margin-right: 12px; }
    .handover-review { overflow: hidden; border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel-soft); }
    .handover-review-head { display: flex; justify-content: space-between; gap: 12px; align-items: center; padding: 14px 15px; border-bottom: 1px solid var(--line); }
    .handover-review-head > div { display: grid; gap: 2px; }
    .handover-review-head > div > span { color: var(--gold-ink); font-size: 10.5px; font-weight: 900; letter-spacing: .07em; text-transform: uppercase; }
    .handover-review-head h2 { font-size: 21px; }
    .handover-review details { border-bottom: 1px solid var(--line); background: rgba(255,255,255,.55); }
    .handover-review details > summary { min-height: 56px; display: flex; justify-content: space-between; align-items: center; gap: 12px; padding: 10px 14px; cursor: pointer; }
    .handover-review details > summary span { display: grid; gap: 2px; }
    .handover-review details > summary small { color: var(--muted); font-size: 11.5px; font-weight: 500; }
    .handover-review details > ul { list-style: none; display: grid; gap: 7px; margin: 0; padding: 0 14px 13px; }
    .handover-review details li { display: grid; grid-template-columns: minmax(100px,.45fr) minmax(0,1fr); gap: 10px; color: var(--muted); font-size: 12.5px; }
    .handover-review details li strong { color: var(--ink); }
    .handover-review details > p { margin: 0; padding: 0 14px 14px; color: var(--muted); line-height: 1.45; }
    .handover-review-files { display: flex; justify-content: space-between; gap: 12px; padding: 13px 14px; color: var(--muted); font-size: 12.5px; }
    .handover-review-files strong { color: var(--ink); }
    .handover-confirm-note { border: 1px solid var(--line); border-radius: var(--radius-xs); padding: 11px 12px; background: var(--panel-soft); }
    .handover-confirm-note > summary { cursor: pointer; color: var(--gold-ink); font-size: 12px; font-weight: 850; }
    .handover-confirm-note label { margin-top: 11px; }
    .handover-confirm-consent { grid-template-columns: 24px minmax(0,1fr); align-items: start; border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 13px; background: #fffefb; color: var(--ink); font-size: 13px; line-height: 1.45; letter-spacing: 0; text-transform: none; cursor: pointer; }
    .handover-confirm-consent input { width: 22px; height: 22px; min-height: 0; margin: 0; accent-color: var(--leaf); }
    .handover-confirm-submit { display: grid; gap: 7px; }
    .handover-confirm-submit .button { width: 100%; min-height: 50px; }
    .handover-confirm-submit small { color: var(--muted); font-size: 11.5px; text-align: center; }
    .empty { border: 1px solid var(--line); background: var(--panel-soft); color: #5c5f54; border-radius: var(--radius-sm); padding: 14px; line-height: 1.5; }
    .empty-state { border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel-soft); padding: 16px 18px; display: grid; grid-template-columns: 52px minmax(0,1fr) auto; gap: 16px; align-items: center; color: var(--ink); }
    .empty-state-icon { width: 52px; height: 52px; border-radius: var(--radius-sm); display: inline-grid; place-items: center; line-height: 0; background: rgba(200,153,63,.16); color: var(--gold-ink); }
    .empty-state-icon svg { display: block; width: 22px; height: 22px; margin: 0; stroke: currentColor; stroke-width: 1.9; fill: none; stroke-linecap: round; stroke-linejoin: round; overflow: visible; }
    .empty-state h3 { font-size: 18px; }
    .empty-state p { margin-top: 4px; color: var(--muted); line-height: 1.42; font-size: 13.5px; text-wrap: pretty; }
    .settings-card { max-width: 620px; display: grid; gap: 16px; }
    .form-grid { display: grid; gap: 12px; }
    label { display: grid; gap: 7px; color: var(--gold-ink); font-size: 12px; font-weight: 800; text-transform: uppercase; letter-spacing: .06em; }
    input { width: 100%; border: 1px solid #e2dac9; border-radius: var(--radius-xs); min-height: 42px; padding: 9px 12px; color: var(--ink); background: #fffefb; }
    .flash { padding: 10px 13px; border-radius: var(--radius-xs); font-size: 13.5px; font-weight: 700; border: 1px solid rgba(200,153,63,.28); background: rgba(200,153,63,.14); color: #8a6a1f; }
    .flash.ok { background: rgba(47,107,74,.12); color: var(--leaf); border-color: rgba(47,107,74,.25); }
    .legend { border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel-soft); padding: 14px; display: grid; grid-template-columns: repeat(auto-fit,minmax(210px,1fr)); gap: 12px; }
    .legend strong { display: block; font-family: var(--font-serif); margin-bottom: 3px; }
    .legend span { display: block; color: var(--muted); font-size: 13px; line-height: 1.4; }
    .bar-cell { min-width: 150px; }
    /* Die volle Seitenleiste braucht rund 1020 Pixel Hoehe. Auf einem
       Bildschirm mit 900 Pixel Fensterhoehe — dem verbreitetsten Notebook —
       lief die Navigation deshalb still ueber: "Einstellungen" war fuer
       Eigentuemer gar nicht erreichbar, ohne dass etwas darauf hindeutete.
       Die kompakte Fassung greift jetzt frueh genug. */
    /* Erst bei wirklich kurzen Fenstern schrumpft auch der Ortskopf. Er
       identifiziert das Haus; die Browser-QA verlangt auf dem Desktop
       mindestens 190 Pixel Höhe und einen 22-Pixel-Pin. */
    @media (min-width: 901px) and (max-height: 800px) {
      .side-map { height: 144px; }
      .side-map-pin { width: 35px; height: 45px; }
      .side-map-pin-mark { top: 7px; width: 20px; height: 18px; }
      .side-map-pin-mark svg { width: 20px; height: 17px; }
      .side-place-copy { margin-top: -42px; }
    }
    @media (min-width: 901px) and (max-height: 1040px) {
      .sidebar { gap: 10px; padding-top: 14px; padding-bottom: 12px; }
      .side-brand { margin-top: -14px; padding-bottom: 3px; }
      .side-place-copy { gap: 5px; padding-bottom: 8px; }
      .side-address-label { font-size: 10.5px; }
      .side-ruler { height: 8px; background-size: 1px 4px,1px 7px,1px 4px; }
      .side-portal strong { font-size: 14.5px; }
      .side-portal span { font-size: 10px; }
      .side-nav { gap: 2px; scrollbar-gutter: stable; }
      .nav-item { min-height: 36px; padding-top: 6px; padding-bottom: 6px; font-size: 13.5px; }
      .nav-icon { width: 19px; height: 19px; }
      .nav-icon svg { width: 18px; height: 18px; }
      .side-foot { gap: 7px; padding-top: 9px; }
      .side-user { grid-template-columns: 32px 1fr; gap: 8px; }
      .avatar { width: 32px; height: 32px; font-size: 11px; }
      .side-user strong { font-size: 12.5px; }
      .side-user span { font-size: 11.5px; }
      .version-button { min-height: 24px; padding: 2px 0; font-size: 10.5px; }
      .logout-button { min-height: 34px; font-size: 12.5px; }
    }
    @media (max-width: 1279px) {
      .energy-health { grid-template-columns: minmax(0,1fr); }
    }
    @media (max-width: 1120px) {
      .metric-grid, .issue-layout { grid-template-columns: 1fr; }
      .handover-detail-grid { grid-template-columns: repeat(2,minmax(0,1fr)); }
      .handover-head { grid-template-columns: 1fr; }
      .handover-actions { justify-content: flex-start; }
      .parking-page-head, .parking-guide, .parking-workspace, .parking-detail-actions { grid-template-columns: 1fr; }
      .pay-fields { grid-template-columns: 1fr; }
      .parking-primary-actions { justify-content: flex-start; }
      .parking-guide-status { justify-content: flex-start; }
      .parking-empty { grid-template-columns: minmax(0,1fr); align-items: start; }
      .parking-empty-art { width: min(190px,54vw); justify-self: start; }
      .parking-empty-actions { justify-content: flex-start; }
      .parking-empty-steps { grid-template-columns: repeat(3,minmax(0,1fr)); gap: 7px; }
      .parking-empty-step { grid-template-columns: 32px minmax(0,1fr); gap: 7px; padding: 9px; }
      .parking-detail-stack { position: static; }
      .digest-panel .quick-list { grid-template-columns: 1fr; }
      /* Two columns keep the board cards readable; the third one takes the
         full width instead of sitting alone next to a gap. */
      .portal-board-grid { grid-template-columns: repeat(2,minmax(0,1fr)); }
      .portal-board-grid > article:nth-child(3) { grid-column: 1 / -1; }
      .portal-energy { grid-template-columns: minmax(0,1fr) auto; }
      .portal-energy-stats { grid-column: 1 / -1; }
    }
	    @media (max-width: 900px) {
	      .app-shell { --mobile-nav-height:68px; display: block; }
	      .energy-mode-strip { position: sticky; top: var(--mobile-nav-height); grid-template-columns: 30px minmax(0,1fr) auto; gap: 6px; min-height: 52px; padding: 4px 12px; }
	      .energy-mode-icon { width: 30px; height: 30px; font-size: 16px; }
	      .energy-mode-copy { gap: 1px; }
	      .energy-mode-copy strong { font-size: 15px; white-space: nowrap; }
	      .energy-mode-strip.active .energy-mode-copy strong { font-size: 13.5px; line-height: 1.1; white-space: normal; }
	      .energy-mode-copy span { font-size: 10.5px; line-height: 1.2; }
	      .energy-mode-control, .energy-mode-strip > form { grid-column: 3; min-width: 0; justify-self: end; }
	      .energy-mode-action { width: min(132px,34vw); min-height: 44px; padding: 4px 6px; font-size: 11px; line-height: 1.15; white-space: normal; }
	      .energy-mode-capability { width: min(132px,34vw); font-size: 10px; line-height: 1.15; }
	      .energy-mode-popover { position: fixed; right: 12px; top: calc(var(--mobile-nav-height) + 60px); width: calc(100vw - 24px); max-height: calc(100dvh - var(--mobile-nav-height) - 72px); margin: 0; }
	      .energy-heading, .energy-health { grid-template-columns: 1fr; }
	      .energy-heading-action { justify-self: start; }
	      .energy-measure-form { grid-template-columns: 1fr; }
	      .energy-live-tools { max-width: 126px; }
	      .energy-chart { scroll-margin-top: 230px; }
	      .energy-chart-head { align-items: start; flex-direction: column; gap: 4px; }
	      .energy-chart-head small { text-align: left; }
	      .energy-chart-head-actions { width: 100%; justify-items: stretch; }
	      .energy-chart-toolbar { width: 100%; justify-content: space-between; }
	      .energy-chart-range { flex: 1 1 auto; }
	      .energy-chart-range a { flex: 1 1 50%; justify-content: center; }
	      .energy-chart-size-actions { flex: 0 1 auto; }
	      .energy-chart-layout { grid-template-columns: 1fr; gap: 14px; }
	      .energy-chart-note { border-top: 1px solid var(--line); border-left: 0; padding-top: 16px; padding-left: 0; }
	      .energy-chart-svg.desktop { display: none; }
	      .energy-chart-svg.mobile { min-height: 0; display: block; }
	      .energy-chart-dialog { width: calc(100vw - 20px); max-height: calc(100dvh - 20px); border-radius: var(--radius-sm); }
	      .energy-chart-dialog-shell { max-height: calc(100dvh - 20px); gap: 14px; padding: 18px 14px; }
	      .energy-chart-dialog-shell > header { align-items: start; flex-direction: column; }
	      .energy-chart-dialog-shell > header h2 { font-size: 27px; }
	      .energy-chart-dialog-shell > header p { font-size: 12px; }
	      .energy-chart-dialog-actions { width: 100%; }
	      .energy-chart-dialog-actions .button { flex: 1 1 auto; }
	      .energy-chart-dialog-plot { padding: 10px 6px 0; }
	      .energy-chart-dialog .energy-chart-svg.mobile { min-height: 0; }
	      .energy-chart-dialog-shell:fullscreen, .energy-chart-dialog.is-fullscreen-fallback .energy-chart-dialog-shell { padding: max(14px,env(safe-area-inset-top)) max(12px,env(safe-area-inset-right)) max(14px,env(safe-area-inset-bottom)) max(12px,env(safe-area-inset-left)); }
	      .energy-chart-tooltip { top: 8px; width: 190px; }
	      .energy-roadmap { grid-template-columns: 1fr; }
	      .energy-roadmap-step { min-height: 0; border-right: 0; border-bottom: 1px solid var(--line); }
	      .energy-roadmap-step:last-child { border-bottom: 0; }
	      .energy-card { padding: 18px; }
	      .onboarding-page { padding: 18px 14px 0; }
	      .onboarding-explain, .onboarding-grid, .onboarding-choice-grid { grid-template-columns: 1fr; }
	      .energy-mapping-guide > header { align-items: start; flex-direction: column; gap: 5px; }
	      .energy-mapping-slots { grid-template-columns: 1fr; }
	      .energy-mapping-slot { grid-template-columns: 1fr; gap: 7px; }
	      .energy-mapping-slot p { min-width: 0; text-align: left; }
	      .energy-mapping-slot small { max-width: none; }
	      .onboarding-actions { align-items: stretch; flex-direction: column-reverse; }
	      .onboarding-actions .button { width: 100%; }
	      .onboarding-candidate { grid-template-columns: auto minmax(0,1fr); }
	      .onboarding-candidate-value, .onboarding-readonly { grid-column: 2; justify-self: start; }
	      .onboarding-disclosure .optional-grid { grid-template-columns: 1fr; }
	      .onboarding-skip { margin-left: 0; }
		      .energy-reference-grid { grid-template-columns: 1fr; }
		      .energy-data-heading { grid-template-columns: 1fr; }
		      .energy-data-trust { justify-content: flex-start; }
		      .energy-lifecycle, .energy-data-inventory { grid-template-columns: 1fr; }
		      .energy-lifecycle-step + .energy-lifecycle-step, .energy-data-inventory div + div { border-top: 1px solid var(--line); border-left: 0; }
		      .energy-lifecycle-step:not(:last-child)::after { content: "↓"; right: 20px; top: auto; bottom: -11px; transform: none; }
		      .energy-data-section-head { align-items: stretch; flex-direction: column; }
		      .energy-data-export .button { width: 100%; }
		      .energy-data-inventory div { min-height: 68px; border-right: 0; }
		      .energy-data-import { grid-template-columns: 1fr; gap: 5px; align-items: start; }
		      .energy-data-import strong { overflow-wrap: anywhere; text-overflow: clip; white-space: normal; }
		      .energy-data-import small { justify-self: start; }
		      .energy-delete-form { grid-template-columns: 1fr; }
		      .energy-delete-button { width: 100%; }
		      .energy-quality, .energy-scenario, .energy-tariff-cost { grid-template-columns: 1fr; }
	      .energy-coverage-head { align-items: flex-start; flex-direction: column; gap: 2px; }
	      /* Der Tarifkopf trägt Auge und Plakette; nebeneinander bricht der
	         Monatsname auf schmalen Geräten in vier Zeilen. */
	      .energy-tariff > .energy-card-head { flex-wrap: wrap; gap: 10px; }
	      .energy-tariff > .energy-card-head > div { flex: 1 1 100%; order: 2; }
	      .energy-tariff > .energy-card-head > .pill { order: 1; }
	      .energy-tariff-foot { grid-template-columns: minmax(0,1fr); }
	      .energy-tariff-foot a, .energy-tariff-foot form { grid-column: 1; grid-row: auto; }
	      .energy-caretaker { grid-template-columns: 1fr 1fr; }
	      .energy-caretaker > div, .energy-caretaker .button { grid-column: 1 / -1; }
	      .energy-measure-control, .energy-measure-control > summary { width: 100%; }
	      .energy-scenario-result { text-align: left; }
	      .energy-history-row, .energy-inline-form { grid-template-columns: 1fr; }
	      .energy-inline-form .wide, .energy-inline-form .actions { grid-column: 1; }
	      .energy-comparison { grid-template-columns: 1fr; }
	      .energy-maintenance-summary, .energy-measure-summary { min-height: 72px; }
	      .energy-mode-action-full { display: none; }
	      .energy-mode-action-compact { display: inline; }
	      .sidebar { position: sticky; top: 0; z-index: 50; display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 10px 12px; height: auto; padding: 10px 14px; box-shadow: 0 10px 28px rgba(23,32,25,.18); }
	      .side-brand { grid-template-columns: 52px minmax(0,1fr); gap: 7px; align-items: center; margin: 0; padding: 0; min-width: 0; }
	      .side-map { width: 52px; height: 48px; }
	      .side-map-pin { width: 24px; height: 31px; }
	      .side-map-pin-mark { top: 5px; width: 14px; height: 13px; }
	      .side-map-pin-mark svg { width: 14px; height: 12px; stroke-width: 2.5; }
	      .side-place-copy { position: static; min-width: 0; min-height: 44px; align-content: center; gap: 3px; margin: 0; padding: 0; }
	      .side-address-label { max-width: 100%; font-size: 10px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
	      .side-address-full { display: none; }
	      .side-address-short { display: inline; overflow: hidden; text-overflow: ellipsis; }
	      .side-ruler { display: none; }
	      .side-portal strong { font-size: 15px; }
	      .side-portal span { display: none; }
	      .mobile-menu-toggle { min-height: 44px; max-width: min(230px,44vw); align-self: center; display: inline-flex; align-items: center; justify-content: center; gap: 7px; border: 1px solid rgba(255,255,255,.24); border-radius: var(--radius-xs); padding: 7px 10px; color: rgba(255,255,255,.92); background: rgba(255,255,255,.07); font-size: 12px; font-weight: 850; letter-spacing: .01em; cursor: pointer; }
	      .mobile-menu-toggle::before { content: ""; width: 14px; height: 10px; border-top: 2px solid currentColor; border-bottom: 2px solid currentColor; box-shadow: 0 4px 0 currentColor inset; }
	      .mobile-menu-prefix { color: rgba(255,255,255,.58); font-size: 10px; font-weight: 650; }
	      .mobile-home-identity { min-width: 0; display: grid; gap: 1px; text-align: left; line-height: 1.05; }
	      .mobile-home-identity strong, .mobile-home-identity small { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	      .mobile-home-identity strong { color: #fff; font-size: 12px; }
	      .mobile-home-identity small { color: rgba(255,255,255,.52); font-size: 10px; font-weight: 550; }
	      .mobile-menu-toggle[aria-expanded="true"] { border-color: rgba(231,197,116,.62); color: #fff; background: rgba(231,197,116,.13); }
	      .side-nav, .side-foot { grid-column: 1 / -1; display: none; }
	      .sidebar.nav-open .side-nav, .sidebar.nav-open .side-foot { display: grid; }
	      .side-nav { grid-template-columns: repeat(2,minmax(0,1fr)); gap: 6px; padding-top: 8px; border-top: 1px solid rgba(255,255,255,.14); }
	      .nav-item { min-height: 44px; padding: 8px 9px; gap: 8px; font-size: 12.5px; }
	      .nav-item.active::before { left: -14px; width: 3px; }
	      .nav-icon { width: 18px; height: 18px; }
	      .nav-icon svg { width: 17px; height: 17px; }
	      .nav-badge { min-width: 20px; height: 18px; padding: 0 6px; font-size: 10px; }
	      .nav-group-label { grid-column: 1 / -1; margin: 7px 4px 0; }
	      .side-foot { margin-top: 2px; grid-template-columns: minmax(0,1fr) auto; align-items: center; gap: 9px 10px; padding: 10px 0 0; }
	      .side-map-attribution { grid-column: 1 / -1; }
	      .side-user { grid-template-columns: 34px minmax(0,1fr); gap: 9px; min-width: 0; }
	      .avatar { width: 34px; height: 34px; font-size: 12px; }
	      .side-user strong { font-size: 13px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
	      .side-user span { font-size: 12px; }
	      .side-version { justify-self: end; min-height: 44px; }
	      .side-map-attribution { min-height: 44px; display: inline-flex; align-items: center; }
	      .logout-form { grid-column: 1 / -1; justify-self: stretch; min-width: 0; }
	      .logout-button { min-height: 44px; }
      .app-main .content-top .button, .energy-mode-popover .button { min-height: 44px !important; }
      :where(.settings-hub,.profile,.notifications,.parking-page,.parking-month-page,.parking-access,.pk-set,.users,.building,.payment-import,.raw-export,.audit-page) :where(.button,button.action,button[type="submit"],button[type="button"],input:not([type="checkbox"]):not([type="radio"]):not([type="hidden"]),select,summary) { min-height: 44px !important; }
      .content-top { height: auto; min-height: 58px; flex-direction: column; align-items: flex-start; padding-top: 12px; padding-bottom: 12px; }
      .content-top .crumb a { min-height: 44px; display: inline-flex; align-items: center; }
      .home-hero { min-height: 108px; align-items: center; padding: 22px 18px; }
      .home-hero::before { background-position: center 46%; }
      .home-hero::after { background: linear-gradient(90deg, rgba(247,243,234,.98), rgba(247,243,234,.82)); }
      .home-hero h1 { font-size: clamp(32px,9vw,40px); }
      .home-hero p { font-size: 14px; }
      .home-page { padding-top: 22px; gap: 28px; }
      .home-primary-task { grid-template-columns: 42px minmax(0,1fr); gap: 14px; padding: 18px; }
      .home-task-action { grid-column: 1 / -1; width: 100%; }
      .home-calm { grid-template-columns: 42px minmax(0,1fr); gap: 14px; padding: 18px; }
      .home-calm-icon { width: 42px; height: 42px; font-size: 19px; }
      .home-calm-copy strong { font-size: 20px; }
      .home-follow-row { grid-template-columns: minmax(0,1fr) auto; gap: 5px 12px; padding: 14px 2px; }
      .home-follow-kind { grid-column: 1; }
      .home-follow-copy { grid-column: 1; }
      .home-follow-action { grid-column: 2; grid-row: 1 / span 2; align-self: center; }
      .portal-board-grid { grid-template-columns: repeat(auto-fit,minmax(280px,1fr)); }
      .portal-board-grid > article:nth-child(3) { grid-column: auto; }
      .portal-energy, .portal-energy.setup { grid-template-columns: minmax(0,1fr); gap: 16px; padding: 18px; }
      .portal-energy-stats { grid-template-columns: repeat(3,minmax(0,1fr)); gap: 8px; }
      .portal-energy-stats > div { border-left: 0; border-top: 1px solid rgba(255,255,255,.14); padding-left: 0; padding-top: 9px; }
      .portal-energy-stats dd { font-size: 19px; }
      .portal-energy-stats dd.pending { font-size: 12.5px; }
      .portal-energy-foot { justify-items: stretch; }
      .portal-energy-foot .button { width: 100%; }
      .portal-energy-foot p { max-width: none; }
      .home-utility-links { grid-template-columns: repeat(auto-fit,minmax(240px,1fr)); }
	      .page, .page.wide { width: 100vw; max-width: 100vw; padding-left: 18px; padding-right: 18px; overflow-x: clip; }
	      .page > *, .panel, .audit-timeline, .filter-form.audit-filter { min-width: 0; max-width: 100%; }
	      h1 { font-size: clamp(34px,10.5vw,42px); }
	      .metric-grid { grid-template-columns: 1fr; }
	      .payment-fields, .parking-breakdown { grid-template-columns: 1fr; }
	      .handover-detail-grid, .handover-confirm-summary { grid-template-columns: 1fr; }
	      .handover-blank { grid-template-columns: minmax(0,1fr); gap: 14px; }
	      .handover-blank-main { padding: 22px 18px; }
	      .handover-blank-side { padding: 17px; }
	      .handover-blank-facts { grid-template-columns: minmax(0,1fr); gap: 12px; }
	      .handover-page-head, .handover-metrics { grid-template-columns: 1fr; }
	      .handover-metrics { grid-template-columns: repeat(3,minmax(0,1fr)); gap: 6px; }
	      .handover-metrics > div { padding: 11px 7px 11px 19px; }
	      .handover-metrics > div::before { left: 8px; top: 16px; width: 6px; height: 6px; }
	      .handover-metrics strong { font-size: 23px; }
	      .handover-metrics span { font-size: 10.5px; line-height: 1.2; }
	      .handover-section > summary { min-height: 58px; }
	      .handover-section > summary small { display: none; }
	      .handover-card { padding: 14px; }
	      .handover-card .handover-head { grid-template-columns: minmax(0,1fr) auto; gap: 8px; }
	      .handover-card h3 { font-size: 21px; }
	      .handover-place span { flex-basis: 100%; }
	      .handover-place span::before { display: none; }
	      .handover-next { grid-template-columns: 1fr; }
	      .handover-next .button, .handover-next form, .handover-next form button { width: 100%; }
	      .handover-parties, .handover-add-files { grid-template-columns: 1fr; }
	      .handover-add-files .button { width: 100%; }
	      .handover-dialog-submit { grid-template-columns: 1fr; }
	      .handover-dialog-submit span { display: none; }
	      .handover-dialog-submit .button { width: 100%; }
	      .handover-confirm-page { padding: 14px; }
	      .handover-confirm-card { padding: 18px 16px; }
	      .handover-confirm-head > p span { flex-basis: 100%; }
	      .handover-confirm-head > p span::before { display: none; }
	      .handover-review details li { grid-template-columns: 1fr; gap: 2px; }
	      .handover-review-files { display: grid; gap: 3px; }
	      .handover-confirm-submit { position: static; padding-top: 2px; margin: 0; background: transparent; }
	      .parking-page-head, .parking-page-head > *, .parking-primary-actions, .parking-page .panel { min-width: 0; max-width: 100%; }
	      .parking-page-head { grid-template-columns: minmax(0,1fr) auto; }
	      .parking-primary-actions { width: auto; display: block; }
	      .parking-more-menu { left: auto; right: 0; width: min(270px, calc(100vw - 36px)); }
	      .parking-month-row { grid-template-columns: 40px minmax(0,1fr) auto; }
	      .parking-month-row .amount { grid-column: 2; }
	      .parking-month-row .pill, .parking-month-row .parking-queue-action { justify-self: start; }
	      .parking-switcher { width: 100%; }
	      .parking-live-head, .parking-months-head, .parking-month-hero { align-items: flex-start; }
	      .parking-live-facts { grid-template-columns: repeat(3,minmax(0,1fr)); }
	      .parking-live-facts > span { min-height: 60px; padding: 9px; }
	      .parking-live-facts strong { font-size: 18px; }
	      .parking-live-actions { display: grid; grid-template-columns: 1fr; }
	      .parking-live-actions form, .parking-live-actions .button, .parking-manual { width: 100%; }
	      .parking-manual > form { position: static; width: 100%; margin-top: 7px; }
	      .parking-split-list p { grid-template-columns: 1fr; gap: 3px; }
	      .parking-months-head { display: grid; }
	      .parking-months-status { justify-content: flex-start; }
	      .parking-current-month { grid-template-columns: minmax(0,1fr) auto 16px; gap: 12px; padding: 15px; }
	      .parking-current-month > span:first-child { grid-column: 1; }
	      .parking-current-facts { grid-column: 1 / -1; grid-row: 2; }
	      .parking-current-facts > span:first-child { border-left: 0; padding-left: 0; }
	      .parking-current-month > .pill { grid-column: 2; grid-row: 1; }
	      .parking-current-month > .parking-row-arrow { grid-column: 3; grid-row: 1; }
	      .parking-compact-month { grid-template-columns: minmax(0,1fr) auto 16px; gap: 8px 10px; }
	      .parking-compact-month .amount { grid-column: 1; }
	      .parking-compact-month .pill { grid-column: 2; grid-row: 1 / span 2; }
	      .parking-compact-month .parking-row-arrow { grid-column: 3; grid-row: 1 / span 2; }
	      .parking-month-hero { grid-template-columns: 1fr; }
	      .parking-month-hero-actions { justify-content: flex-start; }
	      .parking-month-essential { grid-template-columns: repeat(3,minmax(0,1fr)); }
	      .parking-month-essential > div { padding: 11px 8px; }
	      .parking-month-essential dd { font-size: 14px; }
	      .parking-month-payment .pay-row { display: grid; grid-template-columns: 1fr; }
	      .parking-month-payment .pay-row .button { width: 100%; min-width: 0; }
	      .parking-cost-list { grid-template-columns: 1fr; }
	      .parking-hour-row > summary { grid-template-columns: minmax(0,1fr) auto; }
	      .parking-hour-row > summary span { grid-column: 1; }
	      .parking-hour-row > summary strong { grid-column: 2; grid-row: 1 / span 2; }
	      .parking-hour-details { grid-template-columns: repeat(2,minmax(0,1fr)); }
      .parking-stepper { grid-template-columns: 1fr; }
      .parking-step { justify-items: start; grid-template-columns: 32px minmax(0,1fr); text-align: left; }
      .parking-step-number { width: 32px; height: 32px; }
      .parking-step::before { display: none; }
      .issue-stats { grid-template-columns: repeat(2,minmax(0,1fr)); }
      .issue-preview-card { grid-template-columns: 1fr; }
      .issue-preview-card .chips { justify-self: start; justify-content: flex-start; }
      .issue-card-head { grid-template-columns: 1fr; }
      .issue-card-head .issue-meta { justify-content: flex-start; }
      .issue-form { grid-template-columns: 1fr; }
      .issue-category-options { grid-template-columns: repeat(2,minmax(0,1fr)); }
      .issue-category-choice:nth-child(3) { border-left: 0; border-top: 1px solid #e2dac9; }
      .issue-category-choice:nth-child(4) { border-top: 1px solid #e2dac9; }
      .issue-location-options { grid-template-columns: 1fr; }
      .issue-review-row { grid-template-columns: 1fr; gap: 3px; }
      .issue-wizard-actions { align-items: stretch; }
      .issue-wizard-actions .wizard-next, .issue-wizard-actions .wizard-submit { min-width: 0; flex: 1; }
      .issue-board-filter { grid-template-columns: 1fr; }
      .issue-board-filter label, .issue-board-filter label.assignee, .issue-board-filter label.sort, .issue-board-filter .board-filter-actions { grid-column: 1 / -1; }
      .issue-board-toolbar { grid-template-columns: 1fr; }
      .issue-board-summary { border-right: 0; border-bottom: 1px solid var(--line); padding: 12px 14px; }
      .issue-board-tools > summary { grid-template-columns: 32px minmax(0,1fr) auto; gap: 9px; }
      .issue-board-tools > summary > span:not(.issue-filter-icon) { grid-column: 2 / -1; justify-self: start; }
      .issue-board-tools > summary::after { grid-column: 3; grid-row: 1; justify-self: end; }
      .issue-board-quick { padding: 10px 14px; }
      .issue-work-card { padding: 15px 15px 14px 21px; }
      .issue-card-foot { grid-template-columns: 1fr; gap: 12px; }
      .issue-card-foot .issue-card-tools > summary { width: 100%; margin-left: 0; justify-content: center; }
      .board-blank { grid-template-columns: minmax(0,1fr); gap: 14px; }
      .board-blank-main { padding: 22px 18px; }
      .board-blank-side { padding: 17px; }
      .board-blank-facts { grid-template-columns: minmax(0,1fr); gap: 12px; }
      .board-filter-blank { grid-template-columns: minmax(0,1fr); justify-items: start; gap: 13px; padding: 20px 18px; }
      .issue-triage-page { gap: 20px; padding: 22px 16px 36px; }
      .issue-triage-context { gap: 9px; padding-bottom: 18px; }
      .issue-triage-context h1 { font-size: 36px; }
      .issue-triage-context > p { font-size: 14px; }
      .issue-triage-facts { gap: 4px 16px; font-size: 12px; }
      .issue-triage-facts span + span::before { left: -10px; }
      .issue-triage-card { gap: 21px; border-radius: var(--radius-sm); padding: 20px 16px; }
      .issue-triage-card legend { margin-bottom: 15px; font-size: 27px; }
      .issue-triage-choice { gap: 11px; padding: 14px; }
      .issue-triage-choice strong { font-size: 15px; }
      .issue-triage-actions { align-items: stretch; }
      .issue-triage-actions button, .issue-triage-actions .button { min-width: 0; min-height: 46px; }
      .issue-triage-done { grid-template-columns: 1fr; }
      .issue-triage-done .issue-triage-actions { grid-column: 1; display: grid; }
      .issue-resolution-propose { align-items: stretch; flex-direction: column; }
      .issue-resolution-propose button { width: 100%; }
      .issue-resident-page { gap: 20px; padding: 22px 16px 36px; }
      .issue-resident-context { gap: 10px; padding-bottom: 18px; }
      .issue-resident-title-row { align-items: start; flex-direction: column; gap: 12px; }
      .issue-resident-title-row h1 { font-size: clamp(30px,8vw,34px); line-height: 1.05; text-wrap: balance; }
      .issue-resident-task { gap: 19px; border-radius: var(--radius-sm); padding: 22px 16px; }
      .issue-resident-task h2 { font-size: 27px; }
      .issue-resident-task.waiting, .issue-resident-task.done { grid-template-columns: 1fr; }
      .issue-resident-report { padding: 17px 15px; }
      .issue-resident-report-head { align-items: flex-start; flex-direction: column; }
      .issue-resolution-actions { align-items: stretch; flex-direction: column; }
      .issue-resolution-actions form, .issue-resolution-actions button { width: 100%; }
      .event-card { grid-template-columns: 1fr; gap: 11px; padding: 13px; }
      .date-badge { width: 54px; min-height: 54px; }
      .event-card-head { display: grid; grid-template-columns: 1fr; gap: 7px; }
      .event-card-head > .pill { justify-self: start; }
      .event-info h3 { font-size: 18px; }
      .event-actions .button, .event-actions form { flex: 1 1 100px; }
      .event-actions form .button { width: 100%; }
      .quick-row { grid-template-columns: 28px minmax(0,1fr); }
	      .quick-row .entry-actions { grid-column: 2; justify-self: stretch; justify-content: flex-start; margin-top: 5px; }
	      .quick-row .entry-actions .button, .quick-row .entry-actions form { flex: 1 1 112px; min-width: 0; }
	      .quick-row .entry-actions form .button { width: 100%; }
	      .filter-form.audit-filter { grid-template-columns: 1fr; align-items: stretch; }
      .filter-form.audit-filter button, .filter-form.audit-filter .button { width: 100%; min-height: 42px; }
      .audit-filter-panel > summary { padding-left: 13px; }
      .audit-overview { gap: 6px; font-size: 12.5px; }
      .audit-overview strong { font-size: 13px; }
      .audit-filter-content-head { align-items: start; }
      .audit-filter-actions { display: grid; grid-template-columns: 1fr; }
      .audit-filter-actions .button { width: 100%; }
      .audit-timeline { --audit-cols: 48px 10px minmax(0,1fr) 16px; width: 100%; }
      .audit-columns { display: none; }
      .audit-day { padding: 11px 12px 8px; }
      .audit-row { gap: 9px; min-height: 68px; padding: 12px; }
      .audit-actor, .audit-object { display: none; }
      .audit-time { text-align: left; }
      .audit-time strong { font-size: 15.5px; }
      .audit-marker { width: 8px; height: 8px; margin-top: 5px; box-shadow: 0 0 0 3px rgba(200,153,63,.13); }
      .audit-marker::after { top: 11px; bottom: -64px; left: 3px; }
      .audit-title { font-size: 14px; }
      .audit-context { display: block; font-size: 12px; }
      .audit-detail-list { margin: -2px 12px 12px 79px; }
      .audit-detail-row { grid-template-columns: 1fr; gap: 3px; }
	      .document-page-head { align-items: flex-start; }
	      .document-page-head > .pill { margin-top: 5px; }
	      .document-library { padding: 16px; gap: 18px; }
	      .document-library-head { align-items: flex-start; }
	      .document-toolbar { grid-template-columns: 1fr auto; border-radius: var(--radius-xs); }
	      .document-search { grid-column: 1 / -1; border-bottom: 1px solid var(--line); }
	      .document-sort { border-left: 0; }
	      .document-toolbar .button, .document-toolbar .document-reset { border-left: 1px solid var(--line); min-height: 46px; }
	      .document-toolbar .button { padding-inline: 14px; }
	      .document-section h3 { font-size: 18px; }
	      .document-row { grid-template-columns: 44px minmax(0,1fr); gap: 10px 12px; padding: 14px; }
	      .document-icon { width: 42px; height: 49px; border-radius: 9px; }
	      .document-icon svg { width: 21px; height: 21px; }
	      .document-copy > strong { font-size: 18px; }
	      .document-actions { grid-column: 1 / -1; min-width: 0; grid-template-columns: 1fr 1fr; margin-top: 2px; }
	      .document-actions .primary { grid-column: 1 / -1; }
	      .document-actions .button:only-child { grid-column: 1 / -1; }
	      .document-admin-tools, .document-versions { grid-column: 1 / -1; }
	      .document-admin-tools-body { display: grid; grid-template-columns: 1fr; }
	      .document-admin-tools-body .button { width: 100%; justify-content: center; }
	      .document-file { max-width: 100%; }
	      .vote-page-head { align-items: flex-start; }
	      .vote-page-head > .pill { display: none; }
	      .vote-page-head .lede { display: none; }
	      .vote-library { padding: 16px; }
	      .vote-card { padding: 16px; gap: 14px; }
	      .vote-card-head { display: grid; grid-template-columns: 1fr; gap: 9px; }
	      .vote-card-head > .pill { justify-self: start; }
	      .vote-card h3 { font-size: 21px; }
	      .vote-actions { grid-template-columns: 1fr; }
	      .vote-actions .button { width: 100%; }
	      .vote-management { display: grid; grid-template-columns: 1fr; }
	      .vote-management-actions { display: grid; grid-template-columns: 1fr; }
	      .vote-management-actions .button { width: 100%; justify-content: center; }
	      .vote-result-head { align-items: flex-start; display: grid; grid-template-columns: 1fr; }
	      .vote-result-stats { grid-template-columns: 1fr 1fr; }
	      .vote-result-row { grid-template-columns: minmax(0,1fr) auto; gap: 5px 10px; }
	      .vote-result-row .vote-bar { grid-column: 1 / -1; grid-row: 2; }
	      .quick-row .pill { grid-column: 2; justify-self: start; }
	      .filter-form { grid-template-columns: 1fr; }
	      .filter-form.document-filter { grid-template-columns: 1fr; }
      .dialog-grid { grid-template-columns: 1fr; }
      .dialog-optional-grid { grid-template-columns: 1fr; }
      .dialog-optional-grid .full { grid-column: 1; }
      .release-history { grid-template-columns: 1fr; max-height: 76vh; }
      .release-rail { grid-template-columns: repeat(auto-fit,minmax(132px,1fr)); border-right: 0; border-bottom: 1px solid var(--line); padding-right: 0; padding-bottom: 12px; }
      .release-item { grid-template-columns: 1fr; gap: 4px; }
      .empty-state { grid-template-columns: 1fr; }
      .quick-arrow { display: none; }
    }
    /* Auf schmalen Geräten steht die Aktion einer Danach-Zeile unter dem Text.
       Daneben bliebe für die Beschreibung nur eine schmale, zerrissene Spalte. */
    @media (max-width: 600px) {
      .parking-empty-step { min-height: 76px; grid-template-columns: minmax(0,1fr); justify-items: center; gap: 5px; text-align: center; }
      .parking-empty-step-number { width: 28px; height: 28px; }
      .parking-empty-step strong { font-size: 15px; }
      .home-follow-row { grid-template-columns: minmax(0,1fr); gap: 4px; padding: 14px 2px 15px; }
      .home-follow-action { grid-column: 1; grid-row: auto; justify-self: start; margin-top: 5px; }
      .board-blank-actions, .handover-blank-actions { display: grid; }
      .board-blank-actions .button, .handover-blank-actions .button { width: 100%; justify-content: center; }
      .board-filter-blank .button { width: 100%; }
    }
    @media (max-width: 350px) {
      .side-address-locality { display: none; }
      .mobile-menu-prefix { display: none; }
      .settings-hub .settings-section-head { grid-template-columns: minmax(0,1fr); align-items: start; }
      .settings-hub .settings-section-head .settings-tag { justify-self: start; }
      .parking-month-essential > div { padding-inline: 6px; }
      .parking-month-essential dt { font-size: 9px; letter-spacing: 0; }
    }
  </style>
{{end}}

{{define "sidebar"}}
  <aside class="sidebar" aria-label="Portalnavigation">
    <div class="side-brand">
      <a class="side-map side-address" href="{{.MapURL}}" target="_blank" rel="noopener noreferrer" aria-label="{{.Tenant.Address}} in OpenStreetMap öffnen" title="In OpenStreetMap öffnen">
        {{if .SidebarMap.Configured}}
        <div class="side-map-tiles" aria-hidden="true">
          {{range .SidebarMap.Tiles}}<img class="side-map-tile" src="{{.URL}}" style="{{.Style}}" width="256" height="256" alt="" draggable="false">{{end}}
        </div>
        {{else}}<span class="side-map-fallback" aria-hidden="true"></span>{{end}}
        <span class="side-map-pin" aria-hidden="true">
          <svg class="side-map-pin-shape" viewBox="0 0 44 56" focusable="false"><path d="M22 55C18.7 49.2 4.5 36.7 4.5 22.2A17.5 17.5 0 1 1 39.5 22.2C39.5 36.7 25.3 49.2 22 55Z"/></svg>
          <span class="side-map-pin-mark">{{template "tenantBrandMark" .}}</span>
        </span>
      </a>
      <a class="side-place-copy" href="/app" aria-label="Hausportal für {{.SidebarAddress.Full}} öffnen">
        <span class="side-address-label"><span class="side-address-full">{{.SidebarAddress.Full}}</span><span class="side-address-short">{{.SidebarAddress.Primary}}{{if .SidebarAddress.HasLocality}}<span class="side-address-locality"> · {{.SidebarAddress.Locality}}</span>{{end}}</span></span>
        <span class="side-ruler" aria-hidden="true"></span>
        <span class="side-portal"><strong>Hausportal</strong><span>· hausv.org</span></span>
      </a>
	    </div>
	    <button class="mobile-menu-toggle" type="button" data-mobile-menu-toggle aria-controls="portal-navigation portal-account" aria-expanded="false" aria-label="Navigation öffnen"><span class="mobile-menu-prefix">Menü</span>{{if eq .ActivePage "energy"}}<span class="mobile-home-identity" data-home-identity="mobile-menu" aria-label="{{.HomeIdentity.AriaLabel}}"><strong data-home-display-name>{{.HomeIdentity.DisplayName}}</strong>{{if .HomeIdentity.HasUnit}}<small data-home-unit-label>{{.HomeIdentity.UnitLabel}}</small>{{end}}</span>{{else}}<span>{{if eq .ActivePage "home"}}Überblick{{else if eq .ActivePage "announcements"}}Aushang{{else if eq .ActivePage "events"}}Termine{{else if eq .ActivePage "contacts"}}Kontakte{{else if eq .ActivePage "parking"}}Parkplatz{{else if eq .ActivePage "documents"}}Dokumente{{else if eq .ActivePage "handovers"}}Übergaben{{else if eq .ActivePage "issues"}}Anliegen{{else if eq .ActivePage "abstimmungen"}}Abstimmung{{else if eq .ActivePage "users"}}Benutzer{{else if eq .ActivePage "audit"}}Audit{{else}}Einstellungen{{end}}</span>{{end}}</button>
	    <nav id="portal-navigation" class="side-nav">
	      {{if .CanUseResidentAreas}}
	      <a class="nav-item {{if eq .ActivePage "home"}}active{{end}}" href="/app"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg></span><span class="nav-label">Hausüberblick</span></a>
	      {{if .CanViewEnergy}}<a class="nav-item {{if eq .ActivePage "energy"}}active{{end}}" href="/app/energie" data-home-identity="nav" aria-label="{{.HomeIdentity.AriaLabel}}"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M4 13h5l2-7 3 12 2-5h4"/><path d="M5 20h14"/></svg></span><span class="nav-label"><span class="nav-home-identity"><strong data-home-display-name>{{.HomeIdentity.DisplayName}}</strong>{{if .HomeIdentity.HasUnit}}<small data-home-unit-label>{{.HomeIdentity.UnitLabel}}</small>{{end}}</span></span></a>{{end}}
	      <a class="nav-item {{if eq .ActivePage "announcements"}}active{{end}}" href="/app/announcements"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M4 5h16v13H7l-3 3z"/><path d="M8 9h8M8 13h6"/></svg></span><span class="nav-label">Aushang</span>{{if .HasUnreadAnnouncements}}<span class="nav-badge">{{.UnreadAnnouncements}}</span>{{end}}</a>
	      <a class="nav-item {{if eq .ActivePage "events"}}active{{end}}" href="/app/events"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M7 3v4M17 3v4"/><path d="M4.5 6h15v14h-15z"/><path d="M4.5 10h15"/><path d="M8 14h.01M12 14h.01M16 14h.01"/></svg></span><span class="nav-label">Termine</span></a>
	      <a class="nav-item {{if eq .ActivePage "contacts"}}active{{end}}" href="/app/kontakte"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M16 4h2.5A1.5 1.5 0 0 1 20 5.5v13A1.5 1.5 0 0 1 18.5 20h-13A1.5 1.5 0 0 1 4 18.5v-13A1.5 1.5 0 0 1 5.5 4H8"/><path d="M8.5 3.5h7v4h-7z"/><path d="M9 13a3 3 0 1 0 6 0"/><path d="M7.5 18a4.5 4.5 0 0 1 9 0"/></svg></span><span class="nav-label">Kontakte</span></a>
	      <a class="nav-item {{if eq .ActivePage "documents"}}active{{end}}" href="/app/dokumente"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M7 3h7l3 3v15H7z"/><path d="M14 3v4h4"/><path d="M9 13h6M9 17h6"/></svg></span><span class="nav-label">Dokumente</span></a>
	      {{end}}
	      <a class="nav-item {{if eq .ActivePage "issues"}}active{{end}}" href="{{if .CanManageIssues}}/app/anliegen/board{{else}}/app/anliegen{{end}}"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M5 18.5V6.5A2.5 2.5 0 0 1 7.5 4h9A2.5 2.5 0 0 1 19 6.5v6A2.5 2.5 0 0 1 16.5 15H10l-5 3.5z"/></svg></span><span class="nav-label">Anliegen</span>{{if .HasOpenIssues}}<span class="nav-badge">{{.OpenIssues}}</span>{{end}}</a>
	      {{if .CanUseResidentAreas}}
	      <a class="nav-item {{if eq .ActivePage "abstimmungen"}}active{{end}}" href="/app/abstimmungen"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M5 19V9M12 19V5M19 19v-7"/><path d="M3.5 19h17"/></svg></span><span class="nav-label">Abstimmungen</span></a>
	      {{if or .CanSeeParking .CanManageHandovers .CanManageUsers}}<span class="nav-group-label">{{if or .CanManageHandovers .CanManageUsers}}Verwaltung{{else}}Weitere Bereiche{{end}}</span>{{end}}
	      {{if .CanSeeParking}}<a class="nav-item {{if eq .ActivePage "parking"}}active{{end}}" href="/app/parking"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M5 16h14"/><path d="m7 16 1.5-5h7L17 16"/><path d="M7 16v3M17 16v3"/><path d="M7 19h1M16 19h1"/></svg></span><span class="nav-label">Parkplatznutzung</span></a>{{end}}
	      {{if .CanManageHandovers}}<a class="nav-item {{if eq .ActivePage "handovers"}}active{{end}}" href="/app/uebergaben"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M7 4h10v16H7z"/><path d="M9.5 8h5M9.5 12h4"/><path d="m9.5 16 1.5 1.5 3.5-4"/></svg></span><span class="nav-label">Übergaben</span></a>{{end}}
	      {{if .CanManageUsers}}<a class="nav-item {{if eq .ActivePage "users"}}active{{end}}" href="/app/settings/users"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M8.5 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6z"/><path d="M3.5 20a5 5 0 0 1 10 0"/><path d="M16 11.5a2.5 2.5 0 1 0 0-5"/><path d="M17 15a4 4 0 0 1 3.5 4"/></svg></span><span class="nav-label">Benutzer &amp; Rechte</span></a>{{end}}
	      {{if .CanViewAudit}}<a class="nav-item {{if eq .ActivePage "audit"}}active{{end}}" href="/app/audit"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M5 4h14v16H5z"/><path d="M8 8h8M8 12h8M8 16h5"/></svg></span><span class="nav-label">Verlauf</span></a>{{end}}
	      <a class="nav-item {{if eq .ActivePage "settings"}}active{{end}}" href="/app/settings"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M12 8.5a3.5 3.5 0 1 0 0 7 3.5 3.5 0 0 0 0-7z"/><path d="M19 12a7 7 0 0 0-.1-1l2-1.5-2-3.5-2.4 1a7 7 0 0 0-1.8-1L14.4 3h-4.8L9.3 6a7 7 0 0 0-1.8 1l-2.4-1-2 3.5 2 1.5A7 7 0 0 0 5 12a7 7 0 0 0 .1 1l-2 1.5 2 3.5 2.4-1a7 7 0 0 0 1.8 1l.3 3h4.8l.3-3a7 7 0 0 0 1.8-1l2.4 1 2-3.5-2-1.5a7 7 0 0 0 .1-1z"/></svg></span><span class="nav-label">Einstellungen</span></a>
	      {{end}}
	    </nav>
    <div id="portal-account" class="side-foot">
      <a class="side-map-attribution" href="https://www.openstreetmap.org/copyright" target="_blank" rel="noopener noreferrer">Kartendaten © OpenStreetMap</a>
      <div class="side-user">
        <span class="avatar">{{.Initials}}</span>
        <div><strong>{{.DisplayName}}</strong><span>{{.Role}}</span></div>
      </div>
      <button class="side-version version-button" type="button" data-dialog="release-history" aria-haspopup="dialog" aria-controls="release-history">v{{.DisplayVersion}}</button>
      <form class="logout-form" method="post" action="/auth/logout"><button class="logout-button" type="submit">Abmelden</button></form>
    </div>
  </aside>
{{end}}

{{define "releaseHistoryDialog"}}
  {{if .HasReleaseNotes}}
  <dialog id="release-history" class="dialog release-dialog" aria-labelledby="release-history-title">
    <div class="dialog-head">
      <h2 id="release-history-title">Versionsverlauf</h2>
      <button class="dialog-close" type="button" data-close-dialog aria-label="Schließen">&times;</button>
    </div>
    <div class="dialog-body">
      <div class="release-history">
        <nav class="release-rail" aria-label="Versionen">
          {{range .ReleaseNotes}}<a href="#release-{{.Version}}"><strong>v{{.Version}}</strong><span>{{.Date}}</span></a>{{end}}
        </nav>
        <div class="release-body">
          {{range .ReleaseNotes}}
          <section class="release-entry" id="release-{{.Version}}">
            <div class="release-meta"><span class="pill">{{.Kind}}</span><span>{{.Date}}</span></div>
            <h3>{{.Headline}}</h3>
            <p class="muted">{{.Intro}}</p>
            <div class="release-items">
              {{range .Items}}<div class="release-item"><strong>{{.Label}}</strong><span>{{.Text}}</span></div>{{end}}
            </div>
          </section>
          {{end}}
        </div>
      </div>
    </div>
  </dialog>
  {{end}}
{{end}}

{{define "appOpen"}}
<!doctype html>
<html lang="de">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  <link rel="icon" type="image/svg+xml" href="/favicon.svg">
  <link rel="shortcut icon" href="/favicon.svg">
  {{template "appStyles" .}}
  <noscript><style>
    @media (max-width: 900px) { .mobile-menu-toggle { display: none; } .side-nav, .side-foot { display: grid; } }
  </style></noscript>
</head>
<body data-authenticated-app>
  <a class="skip-link" href="#main-content">Zum Inhalt springen</a>
  <div class="app-shell">
    {{template "sidebar" .}}
    {{template "releaseHistoryDialog" .}}
{{end}}

{{define "appClose"}}
  </div>
  <script src="/assets/app.js?v={{.AssetVersion}}" defer></script>
</body>
</html>
{{end}}

{{define "emptyState"}}
  <div class="empty-state">
    <span class="empty-state-icon"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M5 5h14v14H5z"/><path d="M8 9h8M8 13h5"/></svg></span>
    <div><h3>{{.Title}}</h3><p>{{.Message}}</p></div>
    {{if .HasAction}}<a class="button" href="{{.ActionURL}}">{{.ActionLabel}}</a>{{end}}
  </div>
{{end}}

{{define "portalAreaIcon"}}<svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">{{if eq . "energy"}}<path d="M4 13h5l2-7 3 12 2-5h4"/><path d="M5 20h14"/>{{else if eq . "document"}}<path d="M7 3h7l3 3v15H7z"/><path d="M14 3v4h4"/><path d="M9 13h6M9 17h6"/>{{else if eq . "vote"}}<path d="M5 19V9M12 19V5M19 19v-7"/><path d="M3.5 19h17"/>{{else if eq . "contact"}}<path d="M16 4h2.5A1.5 1.5 0 0 1 20 5.5v13A1.5 1.5 0 0 1 18.5 20h-13A1.5 1.5 0 0 1 4 18.5v-13A1.5 1.5 0 0 1 5.5 4H8"/><path d="M8.5 3.5h7v4h-7z"/><path d="M9 13a3 3 0 1 0 6 0"/><path d="M7.5 18a4.5 4.5 0 0 1 9 0"/>{{else if eq . "parking"}}<path d="M5 16h14"/><path d="m7 16 1.5-5h7L17 16"/><path d="M7 16v3M17 16v3"/>{{else if eq . "handover"}}<path d="M7 4h10v16H7z"/><path d="M9.5 8h5M9.5 12h4"/><path d="m9.5 16 1.5 1.5 3.5-4"/>{{else if eq . "users"}}<path d="M8.5 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6z"/><path d="M3.5 20a5 5 0 0 1 10 0"/><path d="M16 11.5a2.5 2.5 0 1 0 0-5"/><path d="M17 15a4 4 0 0 1 3.5 4"/>{{else}}<path d="M12 8.5a3.5 3.5 0 1 0 0 7 3.5 3.5 0 0 0 0-7z"/><path d="M19 12a7 7 0 0 0-.1-1l2-1.5-2-3.5-2.4 1a7 7 0 0 0-1.8-1L14.4 3h-4.8L9.3 6a7 7 0 0 0-1.8 1l-2.4-1-2 3.5 2 1.5A7 7 0 0 0 5 12a7 7 0 0 0 .1 1l-2 1.5 2 3.5 2.4-1a7 7 0 0 0 1.8 1l.3 3h4.8l.3-3a7 7 0 0 0 1.8-1l2.4 1 2-3.5-2-1.5a7 7 0 0 0 .1-1z"/>{{end}}</svg>{{end}}

{{define "portal"}}
{{template "appOpen" .}}
    <main id="main-content" tabindex="-1" class="app-main">
      <section class="home-hero">
        <div class="home-hero-copy">
          <h1>Hallo {{.GreetingName}}.</h1>
          <p>{{.PortalToday}}</p>
        </div>
      </section>
      <section class="page home-page">
        <section class="portal-section" aria-labelledby="home-focus-title">
          <header class="portal-section-head">
            <div>
              <p class="home-eyebrow">Heute</p>
              <h2 id="home-focus-title">Was ist als Nächstes zu tun?</h2>
            </div>
          </header>
          <div class="home-focus">
            {{if .HasDashboardPrimary}}
              <article class="home-primary-task">
                <div class="home-task-number" aria-hidden="true">1</div>
                <div class="home-task-copy">
                  <span>{{.DashboardPrimary.Kind}}</span>
                  <h3>{{.DashboardPrimary.Title}}</h3>
                  <p>{{.DashboardPrimary.Detail}}</p>
                </div>
                <a class="button primary home-task-action" href="{{.DashboardPrimary.URL}}">{{.DashboardPrimary.ActionLabel}} <span aria-hidden="true">→</span></a>
              </article>
            {{else}}
              <div class="home-calm">
                <span class="home-calm-icon" aria-hidden="true">✓</span>
                <span class="home-calm-copy"><strong>Alles im Blick</strong><span>Heute ist nichts zu erledigen. Sobald etwas ansteht, erscheint es hier an erster Stelle.</span></span>
              </div>
            {{end}}
            {{if .HasDashboardFollowUps}}
            <section class="home-follow" aria-labelledby="home-follow-title">
              <h3 id="home-follow-title">{{if .HasDashboardPrimary}}Danach{{else}}Im Blick{{end}}</h3>
              <div class="home-follow-list">
                {{range .DashboardFollowUps}}
                <a class="home-follow-row" href="{{.URL}}">
                  <span class="home-follow-kind">{{.Kind}}</span>
                  <span class="home-follow-copy"><strong>{{.Title}}</strong><span>{{.Detail}}</span></span>
                  <span class="home-follow-action">{{.ActionLabel}} <span aria-hidden="true">→</span></span>
                </a>
                {{end}}
              </div>
            </section>
            {{end}}
          </div>
        </section>

        <section class="portal-section" aria-labelledby="portal-board-title">
          <header class="portal-section-head{{if .CanCreateResidentIssue}} has-action{{end}}">
            <div>
              <p class="home-eyebrow">Im Haus</p>
              <h2 id="portal-board-title">Der aktuelle Stand</h2>
            </div>
            {{if .CanCreateResidentIssue}}<a class="portal-quiet-action" href="/app/anliegen?new=1#issue-new" aria-label="Neues Anliegen melden" title="Mangel, Frage oder Vorschlag melden"><span aria-hidden="true">+</span>Anliegen melden</a>{{end}}
          </header>
          <div class="portal-board-grid">
            <article class="portal-card">
              <header class="portal-card-head">
                <h3>Nächste Termine</h3>
                {{if .PortalEventTotalLabel}}<span class="portal-card-tag">{{.PortalEventTotalLabel}}</span>{{end}}
              </header>
              {{if .HasPortalEvents}}
              <ul class="portal-list">
                {{range .PortalEvents}}
                <li>
                  <a class="portal-row has-date" href="/app/events">
                    <span class="portal-date" aria-hidden="true"><strong>{{.DateBadgeDay}}</strong><span>{{.DateBadgeMonth}}</span></span>
                    <span class="portal-row-copy"><strong>{{.Title}}</strong><span>{{.TimeRange}} · {{.Category}}{{if .HasLocation}} · {{.Location}}{{end}}</span></span>
                  </a>
                </li>
                {{end}}
              </ul>
              {{else if .PortalEventsInFocus}}
              <div class="portal-card-blank">
                <strong>Kein weiterer Termin</strong>
                <span>Der nächste Termin steht bereits oben unter Heute.</span>
              </div>
              {{else}}
              <div class="portal-card-blank">
                <strong>Kein Termin eingetragen</strong>
                <span>Versammlungen, Wartungen und Ablesungen stehen hier, sobald sie feststehen.</span>
              </div>
              {{end}}
              <a class="portal-card-action" href="/app/events">{{if .CanManageEvents}}Termin eintragen{{else}}Alle Termine ansehen{{end}} <span aria-hidden="true">→</span></a>
            </article>

            <article class="portal-card">
              <header class="portal-card-head">
                <h3>Am Aushang</h3>
                {{if .PortalUnreadLabel}}<span class="portal-card-tag">{{.PortalUnreadLabel}}</span>{{end}}
              </header>
              {{if .HasPortalAnnouncements}}
              <ul class="portal-list">
                {{range .PortalAnnouncements}}
                <li>
                  <a class="portal-row" href="/app/announcements">
                    <span class="portal-row-copy"><strong>{{.Title}}</strong><span>{{.Category}} · {{.PublishedAt}}</span></span>
                    {{if .Unread}}<span class="portal-flag" aria-label="ungelesen"></span>{{end}}
                  </a>
                </li>
                {{end}}
              </ul>
              {{else}}
              <div class="portal-card-blank">
                <strong>Der Aushang ist leer</strong>
                <span>Hinweise der Verwaltung erscheinen hier und bleiben nachlesbar.</span>
              </div>
              {{end}}
              <a class="portal-card-action" href="/app/announcements">{{if .CanManageAnnouncements}}Beitrag schreiben{{else}}Zum Aushang{{end}} <span aria-hidden="true">→</span></a>
            </article>

            <article class="portal-card">
              <header class="portal-card-head">
                <h3>Anliegen</h3>
                {{if .HasPortalOpenIssues}}<span class="portal-card-tag">{{.PortalOpenIssueLabel}}</span>{{end}}
              </header>
              {{if .HasPortalIssues}}
              <ul class="portal-list">
                {{range .PortalIssues}}
                <li>
                  <a class="portal-row" href="{{.DetailURL}}">
                    <span class="portal-row-copy"><strong>{{.Title}}</strong><span>{{.Category}} · {{.CreatedAt}}</span></span>
                    <span class="pill {{.StatusClass}}">{{.Status}}</span>
                  </a>
                </li>
                {{end}}
              </ul>
              {{else if .PortalIssuesInFocus}}
              <div class="portal-card-blank">
                <strong>Nichts weiter offen</strong>
                <span>Das offene Anliegen steht bereits oben unter Heute.</span>
              </div>
              {{else}}
              <div class="portal-card-blank">
                <strong>Nichts offen</strong>
                <span>Mängel, Fragen und Vorschläge gehen hier direkt an die Verwaltung.</span>
              </div>
              {{end}}
              {{if .CanManageIssues}}<a class="portal-card-action" href="{{.PortalIssuesURL}}">Anliegen bearbeiten <span aria-hidden="true">→</span></a>
              {{else if and .CanCreateResidentIssue (or .HasPortalIssues .PortalIssuesInFocus)}}<a class="portal-card-action" href="/app/anliegen">Anliegen ansehen <span aria-hidden="true">→</span></a>
              {{else if not .CanCreateResidentIssue}}<a class="portal-card-action" href="{{.PortalIssuesURL}}">Anliegen ansehen <span aria-hidden="true">→</span></a>{{end}}
            </article>
          </div>

          {{if .HasPortalEnergy}}
          <article class="portal-energy{{if not .PortalEnergy.Stats}} setup{{end}}" aria-labelledby="portal-energy-title">
            <div class="portal-energy-copy">
              <p class="portal-energy-kicker">Zuhause und Energie</p>
              <h3 id="portal-energy-title">{{.PortalEnergy.HomeName}}</h3>
              {{if .PortalEnergy.Ready}}<span class="portal-energy-mode{{if .PortalEnergy.ModeActive}} active{{end}}">{{.PortalEnergy.ModeLabel}}</span>{{end}}
              {{if .PortalEnergy.Message}}<p class="portal-energy-message">{{.PortalEnergy.Message}}</p>{{end}}
            </div>
            {{if .PortalEnergy.Stats}}
            <dl class="portal-energy-stats">
              {{range .PortalEnergy.Stats}}<div><dt>{{.Label}}</dt><dd{{if .Muted}} class="pending"{{end}}>{{.Value}}</dd></div>{{end}}
            </dl>
            {{end}}
            <div class="portal-energy-foot">
              <a class="button" href="{{.PortalEnergy.ActionURL}}">{{.PortalEnergy.ActionLabel}}</a>
              <p>{{.PortalEnergy.Footnote}}</p>
            </div>
          </article>
          {{end}}
        </section>

        <section class="home-utilities" aria-labelledby="home-utilities-title">
          <header class="portal-section-head">
            <div>
              <p class="home-eyebrow">Bereiche</p>
              <h2 id="home-utilities-title">{{if .HasHomeUtilities}}Weitere Bereiche und Verwaltung{{else}}Weitere Bereiche{{end}}</h2>
            </div>
          </header>
          <div class="home-utility-links">
            {{range .PortalAreas}}
            <a href="{{.URL}}">
              <span class="portal-area-icon">{{template "portalAreaIcon" .Icon}}</span>
              <span class="portal-area-copy">
                <strong>{{.Label}}</strong>
                <span>{{.Detail}}</span>
                {{if .HasNote}}<em>{{.Note}}</em>{{end}}
              </span>
            </a>
            {{end}}
          </div>
        </section>
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "contactRouteActions"}}
  <div class="route-actions">
    {{if .HasPhone}}<a class="contact-route" href="tel:{{.Phone}}"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M7 4h3l1.5 4-2 1.5a14 14 0 0 0 5 5l1.5-2L20 14v3c0 1.1-.9 2-2 2C10.8 19 5 13.2 5 6c0-1.1.9-2 2-2Z"/></svg><span>Anrufen</span></a>{{end}}
    {{if .HasEmail}}<a class="contact-route" href="mailto:{{.Email}}"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 6h16v12H4z"/><path d="m4 7 8 6 8-6"/></svg><span>E-Mail</span></a>{{end}}
  </div>
{{end}}

{{define "managedContactPublic"}}
  <article class="contact-row">
    <div class="contact-row-copy">
      <span class="contact-row-title"><strong>{{.DisplayName}}</strong><span class="pill contact-kind">{{.Kind}}</span></span>
      {{if .Description}}<span class="contact-description">{{.Description}}</span>{{end}}
      {{if .HasEnergyProfile}}<span class="contact-description"><strong>Energie-Fachhilfe:</strong> {{.EnergySummary}}{{if .Qualification}} · {{.Qualification}}{{end}}</span>{{end}}
      <span class="contact-meta">{{if .HasPhone}}<span>{{.Phone}}</span>{{end}}{{if .HasEmail}}<span>{{.Email}}</span>{{end}}</span>
    </div>
    {{template "contactRouteActions" .}}
  </article>
{{end}}

{{define "directoryContact"}}
  <article class="contact-row">
    <div class="contact-row-copy">
      <span class="contact-row-title"><strong>{{.Name}}</strong><span class="pill">{{.Role}}</span></span>
      {{if .Description}}<span class="contact-description">{{.Description}}</span>{{end}}
      <span class="contact-meta">{{if .HasPhone}}<span>{{.Phone}}</span>{{end}}{{if .HasEmail}}<span>{{.Email}}</span>{{end}}</span>
    </div>
    {{template "contactRouteActions" .}}
  </article>
{{end}}

{{define "managedContactAdmin"}}
  <article class="contact-row {{if not .Active}}inactive{{end}}">
    <div class="contact-row-copy">
      <span class="contact-row-title"><strong>{{.DisplayName}}</strong><span class="pill contact-kind">{{.Kind}}</span>{{if not .Active}}<span class="pill quiet">Inaktiv</span>{{end}}</span>
      {{if .Description}}<span class="contact-description">{{.Description}}</span>{{end}}
      {{if .HasEnergyProfile}}<span class="contact-description"><strong>Energie-Fachhilfe:</strong> {{.EnergySummary}}{{if .Qualification}} · {{.Qualification}}{{end}}</span>{{end}}
      <span class="contact-meta">{{if .HasPhone}}<span>{{.Phone}}</span>{{end}}{{if .HasEmail}}<span>{{.Email}}</span>{{end}}</span>
    </div>
    <div class="contact-row-actions">
      {{if .Active}}{{template "contactRouteActions" .}}{{end}}
      {{if .CanEdit}}
        <button class="button small" type="button" data-dialog="{{.EditDialogID}}" aria-haspopup="dialog" aria-controls="{{.EditDialogID}}">Bearbeiten</button>
      {{else if .Active}}
        <form method="post" action="/app/kontakte/delete" data-confirm="{{.DeleteConfirmLabel}}"><input type="hidden" name="id" value="{{.ID}}"><button class="button small" type="submit">Deaktivieren</button></form>
      {{end}}
    </div>
    {{if .CanEdit}}
      <dialog id="{{.EditDialogID}}" class="dialog contact-edit-dialog" aria-labelledby="{{.EditDialogID}}-title">
        <div class="dialog-head">
          <div><div class="kicker">Adressbuch</div><h2 id="{{.EditDialogID}}-title">{{.DisplayName}}</h2></div>
          <button class="dialog-close" type="button" data-close-dialog aria-label="Schließen">&times;</button>
        </div>
        <div class="dialog-body">
          <form id="{{.EditDialogID}}-form" method="post" action="/app/kontakte">
            <input type="hidden" name="id" value="{{.ID}}">
            <input type="hidden" name="active" value="{{if .Active}}true{{else}}false{{end}}">
            <div class="contact-form">
              <label>Art<select name="kind" required>{{range .KindOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}</select></label>
              <label>Name<input name="name" value="{{.Name}}" maxlength="120"></label>
              <label>Firma / Organisation<input name="company" value="{{.Company}}" maxlength="140"></label>
              <label>Telefon<input name="phone" value="{{.Phone}}" maxlength="80"></label>
              <label>E-Mail<input type="email" name="email" value="{{.Email}}"></label>
              <label class="f-wide">Notiz<input name="notes" value="{{.Notes}}" maxlength="300"></label>
              <label>Region<input name="service_region" value="{{.ServiceRegion}}" maxlength="120" placeholder="z. B. Graz und Umgebung"></label>
              <label>Qualifikation<input name="qualification" value="{{.Qualification}}" maxlength="240" placeholder="z. B. konzessionierter Elektrobetrieb"></label>
              <fieldset class="f-wide"><legend>Energie-Fähigkeiten</legend><div class="permission-grid">
                <label><input type="checkbox" name="energy_capabilities" value="metering"{{range .EnergyCapabilities}}{{if eq . "metering"}} checked{{end}}{{end}}> Leistungsmessung</label>
                <label><input type="checkbox" name="energy_capabilities" value="smart-meter"{{range .EnergyCapabilities}}{{if eq . "smart-meter"}} checked{{end}}{{end}}> Smart Meter</label>
                <label><input type="checkbox" name="energy_capabilities" value="home-assistant"{{range .EnergyCapabilities}}{{if eq . "home-assistant"}} checked{{end}}{{end}}> Home Assistant</label>
                <label><input type="checkbox" name="energy_capabilities" value="pv"{{range .EnergyCapabilities}}{{if eq . "pv"}} checked{{end}}{{end}}> PV</label>
                <label><input type="checkbox" name="energy_capabilities" value="battery"{{range .EnergyCapabilities}}{{if eq . "battery"}} checked{{end}}{{end}}> Speicher</label>
                <label><input type="checkbox" name="energy_capabilities" value="wallbox"{{range .EnergyCapabilities}}{{if eq . "wallbox"}} checked{{end}}{{end}}> Wallbox</label>
                <label><input type="checkbox" name="energy_capabilities" value="heat-pump"{{range .EnergyCapabilities}}{{if eq . "heat-pump"}} checked{{end}}{{end}}> Wärmepumpe</label>
                <label><input type="checkbox" name="energy_capabilities" value="electrical"{{range .EnergyCapabilities}}{{if eq . "electrical"}} checked{{end}}{{end}}> Elektro-Fachnachweis</label>
              </div></fieldset>
            </div>
          </form>
          {{if .Active}}
            <div class="contact-danger">
              <div><strong>Kontakt deaktivieren</strong><p>Der Eintrag verschwindet für Bewohner, bleibt aber erhalten.</p></div>
              <form method="post" action="/app/kontakte/delete" data-confirm="{{.DeleteConfirmLabel}}"><input type="hidden" name="id" value="{{.ID}}"><button class="button small" type="submit">Deaktivieren</button></form>
            </div>
          {{else}}
            <form class="contact-reactivate" method="post" action="/app/kontakte">
              <input type="hidden" name="id" value="{{.ID}}"><input type="hidden" name="kind" value="{{.Kind}}"><input type="hidden" name="name" value="{{.Name}}"><input type="hidden" name="company" value="{{.Company}}"><input type="hidden" name="email" value="{{.Email}}"><input type="hidden" name="phone" value="{{.Phone}}"><input type="hidden" name="notes" value="{{.Notes}}"><input type="hidden" name="service_region" value="{{.ServiceRegion}}"><input type="hidden" name="qualification" value="{{.Qualification}}">{{range .EnergyCapabilities}}<input type="hidden" name="energy_capabilities" value="{{.}}">{{end}}<input type="hidden" name="active" value="true">
              <p>Dieser Kontakt ist derzeit nur für die Verwaltung sichtbar.</p><button class="button primary" type="submit">Wieder aktivieren</button>
            </form>
          {{end}}
        </div>
        <div class="dialog-footer contact-dialog-footer">
          <button class="button primary" type="submit" form="{{.EditDialogID}}-form">Änderungen speichern</button>
        </div>
      </dialog>
    {{end}}
  </article>
{{end}}

{{define "contacts"}}
{{template "appOpen" .}}
    <style>
      .contacts .contacts-layout { display: grid; gap: 20px; }
      .contacts .contacts-main, .contacts .contacts-aside { min-width: 0; display: grid; gap: 16px; align-content: start; }
      .contacts .contacts-main.is-blank { align-content: start; }
      .contacts .contact-section { display: grid; gap: 16px; }
      .contacts .contact-section .section-head { align-items: flex-end; }
      .contacts .section-copy { max-width: 720px; }
      .contacts .section-copy h2 { margin-bottom: 4px; }
      .contacts .section-copy p { margin: 0; }
      .contacts .quick-panel { border-top: 3px solid var(--gold); }
      .contacts .quick-list { display: grid; grid-template-columns: repeat(auto-fill,minmax(258px,1fr)); gap: 12px; }
      .contacts .quick-card { border: 1px solid var(--line); border-radius: 10px; padding: 16px; background: var(--panel-soft); display: grid; gap: 12px; align-content: start; }
      .contacts .quick-card.urgent { border-color: rgba(172,72,42,.4); background: #fff9f5; }
      .contacts .quick-label { display: flex; justify-content: space-between; gap: 8px; align-items: center; color: var(--gold-ink); font-size: 11px; font-weight: 900; letter-spacing: .12em; text-transform: uppercase; }
      .contacts .quick-card.urgent .quick-label { color: #a4422b; }
      .contacts .quick-card h3 { margin: 0; font-family: var(--font-serif); font-size: 22px; line-height: 1.08; overflow-wrap: anywhere; }
      .contacts .quick-card p { margin: 0; color: var(--muted); font-size: 14px; }
      .contacts .contact-meta { display: flex; flex-wrap: wrap; gap: 4px 12px; color: var(--muted); font-size: 13px; overflow-wrap: anywhere; }
      .contacts .route-actions { display: flex; flex-wrap: wrap; gap: 8px; margin-top: auto; }
      .contacts .contact-route { min-height: 42px; padding: 0 13px; border: 1px solid var(--line); border-radius: 8px; background: #fff; color: var(--ink); display: inline-flex; align-items: center; justify-content: center; gap: 8px; text-decoration: none; font-weight: 850; font-size: 13.5px; }
      .contacts .contact-route:hover { border-color: var(--gold); color: var(--gold-ink); }
      .contacts .contact-route svg { width: 18px; height: 18px; fill: none; stroke: currentColor; stroke-width: 1.9; stroke-linecap: round; stroke-linejoin: round; }
      .contacts .contact-admin-link { font-size: 13px; }
      .contacts .contact-add { border: 1px solid var(--line); border-radius: 10px; background: var(--panel-soft); }
      .contacts .contact-add > summary { min-height: 48px; padding: 0 16px; display: flex; align-items: center; justify-content: space-between; gap: 12px; cursor: pointer; color: var(--ink); font-weight: 900; list-style: none; }
      .contacts .contact-add > summary::-webkit-details-marker { display: none; }
      .contacts .contact-add > summary::after { content: "+"; width: 26px; height: 26px; border-radius: 50%; background: var(--ink); color: white; display: grid; place-items: center; font-size: 18px; line-height: 1; }
      .contacts .contact-add[open] > summary::after { content: "−"; }
      .contacts .contact-add-body { padding: 4px 16px 16px; border-top: 1px solid var(--line); }
      .contacts .contact-add-hint { margin: 12px 0; color: var(--muted); font-size: 13.5px; }
      .contacts .contact-form { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 12px; align-items: end; }
      .contacts .contact-form .f-wide, .contacts .contact-form .f-actions { grid-column: 1 / -1; }
      .contacts .contact-form .f-actions { display: flex; justify-content: flex-end; }
      .contacts .contact-form fieldset { min-width: 0; margin: 0; border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 10px 14px 14px; }
      .contacts .contact-form legend { padding: 0 6px; color: var(--gold-ink); font-size: 11.5px; font-weight: 850; letter-spacing: .1em; text-transform: uppercase; }
      .contacts .contact-form legend .muted { font-size: 11px; font-weight: 700; letter-spacing: .04em; text-transform: none; }
      .contacts .contact-form .permission-grid { display: grid; grid-template-columns: repeat(auto-fit,minmax(184px,1fr)); gap: 6px 18px; }
      .contacts .contact-form .permission-grid label { min-height: 34px; grid-template-columns: auto minmax(0,1fr); align-items: center; gap: 9px; color: var(--ink); font-size: 13.5px; font-weight: 650; letter-spacing: 0; text-transform: none; }
      .contacts .contact-form .permission-grid input { width: 17px; height: 17px; min-height: 0; margin: 0; }
      .contacts .contact-add-optional { grid-column: 1 / -1; border: 1px solid var(--line); border-radius: var(--radius-sm); background: #fffefb; }
      .contacts .contact-add-optional > summary { min-height: 44px; display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 0 14px; cursor: pointer; list-style: none; color: var(--ink); font-size: 13.5px; font-weight: 850; }
      .contacts .contact-add-optional > summary::-webkit-details-marker { display: none; }
      .contacts .contact-add-optional > summary::after { content: "\203A"; color: var(--gold-ink); font-size: 20px; line-height: .8; }
      .contacts .contact-add-optional[open] > summary { border-bottom: 1px solid var(--line); }
      .contacts .contact-add-optional[open] > summary::after { transform: rotate(90deg); }
      .contacts .contact-add-optional-grid { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 12px; padding: 14px; }
      .contacts .contact-add-optional-grid fieldset { grid-column: 1 / -1; }
      .contacts .contact-list { display: grid; gap: 8px; }
      .contacts .contact-row { border: 1px solid var(--line); border-radius: 10px; padding: 13px 14px; background: var(--panel-soft); display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 12px; align-items: center; }
      .contacts .contact-row-copy { min-width: 0; display: grid; gap: 4px; }
      .contacts .contact-row-title { min-width: 0; display: flex; flex-wrap: wrap; gap: 7px; align-items: center; }
      .contacts .contact-row-title strong { font-family: var(--font-serif); font-size: 19px; line-height: 1.1; overflow-wrap: anywhere; }
      .contacts .contact-description { color: var(--muted); font-size: 13.5px; }
      .contacts .contact-row-actions { display: flex; gap: 8px; align-items: center; justify-content: flex-end; }
      .contacts .contact-row.inactive { opacity: .72; }
      .contacts .pill.quiet { background: #ece8de; color: var(--muted); }
      .contacts .inactive-contacts { border-top: 1px solid var(--line); padding-top: 12px; }
      .contacts .inactive-contacts > summary { cursor: pointer; color: var(--muted); font-size: 13.5px; font-weight: 800; }
      .contacts .inactive-contacts .contact-list { margin-top: 10px; }
      .contacts .contact-empty-line { margin: 0; padding: 14px; border-radius: 8px; background: var(--panel-soft); color: var(--muted); }
      .contacts .contact-groups { display: grid; gap: 18px; }
      .contacts .contact-group .contact-kind { display: none; }
      .contacts .contact-group.is-urgent .contact-group-head h3 { color: #a4422b; }
      .contacts .contact-group.is-urgent .contact-group-head { border-bottom-color: rgba(172,72,42,.34); }
      .contacts .contact-group.is-urgent .contact-row { border-color: rgba(172,72,42,.34); background: #fff9f5; }
      .contacts .contact-group { display: grid; gap: 9px; }
      .contacts .contact-group-head { display: flex; align-items: baseline; justify-content: space-between; gap: 10px; border-bottom: 1px solid var(--line); padding-bottom: 7px; }
      .contacts .contact-group-head h3 { color: var(--gold-ink); font-family: var(--font-sans); font-size: 11.5px; font-weight: 850; letter-spacing: .11em; text-transform: uppercase; }
      .contacts .contact-group-head span { color: var(--soft); font-size: 12px; font-weight: 750; white-space: nowrap; }
      .contacts .contacts-blank { min-height: clamp(280px,34vh,360px); display: grid; align-content: center; justify-items: center; gap: 15px; border: 1px dashed rgba(200,153,63,.42); border-radius: var(--radius-sm); background: rgba(255,254,251,.68); padding: 32px 26px; text-align: center; }
      .contacts .contacts-blank .empty-state { width: 100%; border: 0; background: transparent; padding: 0; grid-template-columns: minmax(0,1fr); justify-items: center; gap: 13px; text-align: center; }
      .contacts .contacts-blank .empty-state p { max-width: 48ch; }
      .contacts .contacts-blank-actions { display: flex; flex-wrap: wrap; gap: 10px; justify-content: center; }
      .contacts .contacts-blank-actions .button { min-height: 44px; }
      .contacts .contacts-aside-panel { display: grid; gap: 14px; padding: 20px; align-content: start; }
      .contacts .contacts-aside-panel .kicker { margin-bottom: 0; }
      .contacts .contacts-aside-panel h2 { font-size: 20px; }
      .contacts .contacts-aside-panel > p { color: var(--muted); font-size: 13.5px; line-height: 1.5; }
      .contacts .contacts-guide { margin: 0; padding: 0; list-style: none; display: grid; gap: 13px; }
      .contacts .contacts-guide li { display: grid; grid-template-columns: 9px minmax(0,1fr); gap: 11px; align-items: start; }
      .contacts .contacts-guide i { margin-top: 6px; width: 9px; height: 9px; border-radius: 50%; background: var(--gold); }
      .contacts .contacts-guide i.urgent { background: #a4422b; }
      .contacts .contacts-guide strong { display: block; font-size: 14px; line-height: 1.25; }
      .contacts .contacts-guide span { display: block; color: var(--muted); font-size: 12.8px; line-height: 1.42; }
      .contacts .contacts-links { display: grid; }
      .contacts .contacts-link { min-height: 48px; display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 10px; align-items: center; border-top: 1px solid var(--line); padding: 11px 0; color: inherit; text-decoration: none; }
      .contacts .contacts-link:first-child { border-top: 0; padding-top: 0; }
      .contacts .contacts-link strong { display: block; font-size: 14px; }
      .contacts .contacts-link small { display: block; margin-top: 2px; color: var(--muted); font-size: 12.5px; line-height: 1.35; }
      .contacts .contacts-link::after { content: "\203A"; color: var(--gold-ink); font-size: 21px; line-height: 1; }
      .contacts .contacts-link:hover strong { color: var(--gold-ink); }
      .contacts .directory-panel { background: rgba(255,255,255,.72); }
      .contacts .directory-note { display: inline-flex; align-items: center; gap: 8px; color: var(--muted); font-size: 13px; }
      .contacts .directory-note svg { width: 17px; height: 17px; fill: none; stroke: var(--gold-ink); stroke-width: 1.8; }
      .contacts .contact-edit-dialog .dialog-head { align-items: flex-start; }
      .contacts .contact-edit-dialog .dialog-head h2 { margin: 2px 0 0; }
      .contacts .contact-danger, .contacts .contact-reactivate { margin-top: 18px; padding-top: 16px; border-top: 1px solid rgba(158,70,48,.3); display: flex; gap: 14px; align-items: center; justify-content: space-between; }
      .contacts .contact-danger p, .contacts .contact-reactivate p { margin: 3px 0 0; color: var(--muted); font-size: 13px; }
      @media (min-width: 1181px) { .contacts .contacts-layout { grid-template-columns: minmax(0,1fr) 320px; } }
      @media (max-width: 1180px) and (min-width: 721px) { .contacts .contacts-aside { grid-template-columns: repeat(2,minmax(0,1fr)); align-items: start; } }
      @media (max-width: 720px) {
        .contacts .contacts-layout { gap: 12px; }
        .contacts .contacts-main, .contacts .contacts-aside { gap: 12px; }
        .contacts .contact-section { gap: 12px; }
        .contacts .contacts-blank { min-height: 0; padding: 26px 18px; }
        .contacts .contacts-blank-actions { width: 100%; }
        .contacts .contacts-blank-actions .button { width: 100%; }
        .contacts .contacts-aside-panel { padding: 16px; }
        .contacts .quick-list { grid-template-columns: 1fr; gap: 8px; }
        .contacts .quick-card { padding: 14px; gap: 9px; }
        .contacts .quick-card h3 { font-size: 20px; }
        .contacts .quick-card p { display: none; }
        .contacts .quick-card .contact-meta { font-size: 12.5px; }
        .contacts .route-actions { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); width: 100%; }
        .contacts .contact-route { min-height: 44px; }
        .contacts .contact-form { grid-template-columns: 1fr; }
        .contacts .contact-form > *, .contacts .contact-form .f-wide, .contacts .contact-form .f-actions { grid-column: 1; }
        .contacts .contact-form .f-actions .button { width: 100%; }
        .contacts .contact-add-optional-grid { grid-template-columns: 1fr; }
        .contacts .contact-row { grid-template-columns: 1fr; padding: 12px; gap: 10px; }
        .contacts .contact-row-actions { display: grid; grid-template-columns: 1fr; justify-content: stretch; }
        .contacts .contact-row-actions > .button, .contacts .contact-row-actions > form, .contacts .contact-row-actions > form .button { width: 100%; }
        .contacts .contact-row-actions .route-actions { flex: 1; }
        .contacts .contact-danger, .contacts .contact-reactivate { align-items: stretch; flex-direction: column; }
        .contacts .contact-danger .button, .contacts .contact-reactivate .button { width: 100%; }
        .contacts .section-head { align-items: flex-start; }
        .contacts .page-actions .button { min-height: 44px; }
        .contacts .contact-row-actions .button { min-height: 42px; }
        .contacts .contact-form .permission-grid label { min-height: 44px; }
        .contacts .inactive-contacts > summary { min-height: 44px; display: flex; align-items: center; }
      }
    </style>
    <main id="main-content" tabindex="-1" class="app-main contacts">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M16 4h2.5A1.5 1.5 0 0 1 20 5.5v13A1.5 1.5 0 0 1 18.5 20h-13A1.5 1.5 0 0 1 4 18.5v-13A1.5 1.5 0 0 1 5.5 4H8"/><path d="M8.5 3.5h7v4h-7z"/><path d="M9 13a3 3 0 1 0 6 0"/><path d="M7.5 18a4.5 4.5 0 0 1 9 0"/></svg><span>/</span><span>Kontakte</span></span>
        {{if .CanManageContacts}}<div class="page-actions"><a class="button primary" href="#contact-add"><svg viewBox="0 0 24 24" width="17" height="17" aria-hidden="true"><path d="M12 5v14M5 12h14" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round"/></svg>Kontakt hinzufügen</a></div>{{end}}
      </div>
      <section class="page wide">
        <div>
          <h1>Kontakte</h1>
          <p class="lede">Schnell die richtige Ansprechperson für {{.Tenant.Address}} erreichen.</p>
        </div>
        {{if .ContactMsg}}<p class="flash {{if .ContactOK}}ok{{end}}">{{.ContactMsg}}</p>{{end}}
        <div class="contacts-layout">
        <div class="contacts-main{{if and (not .HasAnyContacts) (not .CanManageContacts)}} is-blank{{end}}">
          {{if .HasQuickContacts}}
            <section class="panel contact-section quick-panel" aria-labelledby="quick-contacts-title">
              <div class="section-head">
                <div class="section-copy"><div class="kicker">Schnell erreichen</div><h2 id="quick-contacts-title">Hilfe &amp; Haus-Ansprechpersonen</h2></div>
                {{if .CanManageContacts}}<a class="section-link contact-admin-link" href="/app/settings/building#building-contact">Hauskontakte pflegen</a>{{end}}
              </div>
              <div class="quick-list">
                {{range .EmergencyContacts}}
                  {{if eq .Role "Notdienst"}}
                    <article class="quick-card urgent">
                      <div class="quick-label"><span>{{.Role}}</span><span>Dringend</span></div>
                      <h3>{{.Name}}</h3>
                      {{if .Description}}<p>{{.Description}}</p>{{end}}
                      <div class="contact-meta">{{if .HasPhone}}<span>{{.Phone}}</span>{{end}}{{if .HasEmail}}<span>{{.Email}}</span>{{end}}</div>
                      {{template "contactRouteActions" .}}
                    </article>
                  {{end}}
                {{end}}
                {{range .ManagerContacts}}
                  <article class="quick-card">
                    <div class="quick-label"><span>Hausverwaltung</span></div>
                    <h3>{{.Name}}</h3>
                    {{if .Description}}<p>{{.Description}}</p>{{end}}
                    <div class="contact-meta">{{if .HasPhone}}<span>{{.Phone}}</span>{{end}}{{if .HasEmail}}<span>{{.Email}}</span>{{end}}</div>
                    {{template "contactRouteActions" .}}
                  </article>
                {{end}}
                {{range .EmergencyContacts}}
                  {{if ne .Role "Notdienst"}}
                    <article class="quick-card">
                      <div class="quick-label"><span>{{.Role}}</span></div>
                      <h3>{{.Name}}</h3>
                      {{if .Description}}<p>{{.Description}}</p>{{end}}
                      <div class="contact-meta">{{if .HasPhone}}<span>{{.Phone}}</span>{{end}}{{if .HasEmail}}<span>{{.Email}}</span>{{end}}</div>
                      {{template "contactRouteActions" .}}
                    </article>
                  {{end}}
                {{end}}
                {{range .BoardContacts}}
                  <article class="quick-card">
                    <div class="quick-label"><span>Beirat</span></div>
                    <h3>{{.Name}}</h3>
                    {{if .Description}}<p>{{.Description}}</p>{{end}}
                    <div class="contact-meta">{{if .HasPhone}}<span>{{.Phone}}</span>{{end}}{{if .HasEmail}}<span>{{.Email}}</span>{{end}}</div>
                    {{template "contactRouteActions" .}}
                  </article>
                {{end}}
              </div>
            </section>
          {{end}}

          {{if or .CanManageContacts .HasManagedContacts}}
            <section class="panel contact-section managed-panel" id="contact-book" aria-labelledby="managed-contacts-title">
              <div class="section-head">
                <div class="section-copy">
                  <div class="kicker">Adressbuch</div>
                  <h2 id="managed-contacts-title">Weitere wichtige Kontakte</h2>
                  <p class="muted">Firmen, Dienste und wiederkehrende Ansprechpartner für das Haus.</p>
                </div>
                {{if .HasManagedContacts}}<span class="pill">{{len .ManagedContacts}} im Adressbuch</span>{{end}}
              </div>
              {{if .CanManageContacts}}
                <details class="contact-add" id="contact-add" {{if or .ContactFormOpen (not .HasManagedContacts)}}open{{end}}>
                  <summary>Kontakt hinzufügen</summary>
                  <div class="contact-add-body">
                    <p class="contact-add-hint">Name oder Firma und mindestens Telefon oder E-Mail angeben. {{if not .ServiceProviderAccessEnabled}}Dienstleister-Zugänge sind derzeit nicht verfügbar.{{end}}</p>
                    <form class="contact-form" method="post" action="/app/kontakte">
                      <input type="hidden" name="active" value="true">
                      <label>Art<select name="kind" required>{{range .ContactKindOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}</select></label>
                      <label>Name<input name="name" maxlength="120" placeholder="Ansprechperson"></label>
                      <label>Firma / Organisation<input name="company" maxlength="140" placeholder="z. B. Elektro Süd"></label>
                      <label>Telefon<input name="phone" maxlength="80" placeholder="+43 ..."></label>
                      <label>E-Mail<input type="email" name="email" placeholder="kontakt@example.com"></label>
                      <label class="f-wide">Notiz<input name="notes" maxlength="300" placeholder="z. B. Lift, Elektrik oder Erreichbarkeit"></label>
                      <details class="contact-add-optional">
                        <summary>Region, Qualifikation und Energie-Fähigkeiten</summary>
                        <div class="contact-add-optional-grid">
                          <label>Region <span class="muted">(optional)</span><input name="service_region" maxlength="120" placeholder="z. B. Graz und Umgebung"></label>
                          <label>Qualifikation <span class="muted">(optional)</span><input name="qualification" maxlength="240" placeholder="z. B. konzessionierter Elektrobetrieb"></label>
                          <fieldset><legend>Energie-Fähigkeiten <span class="muted">(optional, kein Portalzugang)</span></legend><div class="permission-grid">
                            <label><input type="checkbox" name="energy_capabilities" value="metering"> Leistungsmessung</label>
                            <label><input type="checkbox" name="energy_capabilities" value="smart-meter"> Smart Meter</label>
                            <label><input type="checkbox" name="energy_capabilities" value="home-assistant"> Home Assistant</label>
                            <label><input type="checkbox" name="energy_capabilities" value="pv"> PV</label>
                            <label><input type="checkbox" name="energy_capabilities" value="battery"> Speicher</label>
                            <label><input type="checkbox" name="energy_capabilities" value="wallbox"> Wallbox</label>
                            <label><input type="checkbox" name="energy_capabilities" value="heat-pump"> Wärmepumpe</label>
                            <label><input type="checkbox" name="energy_capabilities" value="electrical"> Elektro-Fachnachweis</label>
                          </div></fieldset>
                        </div>
                      </details>
                      <div class="f-actions"><button class="button primary" type="submit">Kontakt anlegen</button></div>
                    </form>
                  </div>
                </details>
              {{end}}
              {{if .HasManagedContacts}}
                <div class="contact-groups">
                  {{range .ManagedGroups}}
                    <section class="contact-group{{if eq .Kind "Notdienst"}} is-urgent{{end}}">
                      <div class="contact-group-head"><h3>{{.Kind}}</h3><span>{{.Count}}</span></div>
                      <div class="contact-list">
                        {{range .Contacts}}{{if $.CanManageContacts}}{{template "managedContactAdmin" .}}{{else}}{{template "managedContactPublic" .}}{{end}}{{end}}
                      </div>
                    </section>
                  {{end}}
                </div>
              {{else if .CanManageContacts}}
                <p class="contact-empty-line">Noch keine weiteren Kontakte. Das Formular oben legt den ersten an – etwa Lift, Heizung oder Elektrik.</p>
              {{end}}
              {{if and .CanManageContacts .HasInactiveContacts}}
                <details class="inactive-contacts">
                  <summary>Inaktive Kontakte ({{len .InactiveContacts}})</summary>
                  <div class="contact-list">{{range .InactiveContacts}}{{template "managedContactAdmin" .}}{{end}}</div>
                </details>
              {{end}}
            </section>
          {{end}}

          {{if .HasResidentContacts}}
            <section class="panel contact-section directory-panel" aria-labelledby="resident-directory-title">
              <div class="section-head">
                <div class="section-copy">
                  <div class="kicker">Freiwilliges Verzeichnis</div>
                  <h2 id="resident-directory-title">Hausgemeinschaft</h2>
                  <p class="muted">Nur Bewohner, die ihren Kontakt im Profil ausdrücklich für dieses Haus freigegeben haben.</p>
                </div>
                <span class="directory-note"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 3 5 6v5c0 4.5 2.8 8.1 7 10 4.2-1.9 7-5.5 7-10V6z"/><path d="m9 12 2 2 4-5"/></svg>Freiwillig freigegeben</span>
              </div>
              <div class="contact-list">{{range .ResidentContacts}}{{template "directoryContact" .}}{{end}}</div>
            </section>
          {{end}}

          {{if and (not .HasAnyContacts) (not .CanManageContacts)}}
            <section class="contacts-blank" aria-label="Kontakte des Hauses">
              {{template "emptyState" .ManagedEmpty}}
              <div class="contacts-blank-actions">
                <a class="button primary" href="{{if .CanManageIssues}}/app/anliegen/board{{else}}/app/anliegen{{end}}">Anliegen melden</a>
                <a class="button ghost" href="/app/announcements">Aushang ansehen</a>
              </div>
            </section>
          {{end}}
        </div>

        <aside class="contacts-aside" aria-label="Hilfe zu Kontakten">
          <details class="panel compact contacts-aside-panel guide-disclosure" aria-labelledby="contacts-guide-title">
            <summary><div>
              <div class="kicker">Wegweiser</div>
              <h2 id="contacts-guide-title">Wer ist wofür zuständig?</h2>
            </div></summary>
            <ul class="contacts-guide">
              <li><i class="urgent" aria-hidden="true"></i><div><strong>Notdienst</strong><span>Gefahr im Verzug, Wasserschaden, Stromausfall oder Personen im Lift.</span></div></li>
              <li><i aria-hidden="true"></i><div><strong>Hausverwaltung</strong><span>Verträge, Abrechnung, Beschlüsse und alles Kaufmännische.</span></div></li>
              <li><i aria-hidden="true"></i><div><strong>Hausmeister</strong><span>Schlüssel, Reinigung, Grünflächen und kleine Reparaturen.</span></div></li>
              <li><i aria-hidden="true"></i><div><strong>Beirat</strong><span>Vertritt die Eigentümergemeinschaft gegenüber der Verwaltung.</span></div></li>
              <li><i aria-hidden="true"></i><div><strong>Dienstleister</strong><span>Firmen für Lift, Heizung, Elektrik oder Energiethemen im Haus.</span></div></li>
            </ul>
          </details>
          <section class="panel compact contacts-aside-panel" aria-labelledby="contacts-next-title">
            <div>
              <div class="kicker">Auch hilfreich</div>
              <h2 id="contacts-next-title">Weiter im Portal</h2>
            </div>
            <div class="contacts-links">
              <a class="contacts-link" href="{{if .CanManageIssues}}/app/anliegen/board{{else}}/app/anliegen{{end}}"><span><strong>Anliegen melden</strong><small>Bleibt dokumentiert und geht nicht verloren – anders als ein Anruf.</small></span></a>
              {{if .CanManageContacts}}<a class="contacts-link" href="/app/settings/building#building-contact"><span><strong>Hauskontakte pflegen</strong><small>Verwaltung, Notdienst und Hausmeister stehen in den Gebäude-Einstellungen.</small></span></a>{{end}}
              {{if .CanJoinDirectory}}<a class="contacts-link" href="/app/settings/profile"><span><strong>Eigener Verzeichniseintrag</strong><small>{{if .DirectoryOptIn}}Ihr Kontakt ist für die Hausgemeinschaft sichtbar.{{else}}Ihr Kontakt ist derzeit nicht sichtbar.{{end}}</small></span></a>{{end}}
            </div>
          </section>
        </aside>
        </div>
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "attachmentStrip"}}
  {{if .HasAttachments}}
    <div class="attachment-strip" aria-label="Anhänge">
      {{range .Attachments}}
        <div class="attachment-item">
          {{if .IsImage}}
            <button class="attachment-open" type="button" data-lightbox-src="{{.PreviewURL}}" data-lightbox-full="{{.URL}}" data-lightbox-caption="{{.Filename}}">
              <img src="{{.ThumbURL}}" alt="{{.Filename}}" loading="lazy" decoding="async">
              <span class="attachment-name">{{.Filename}}</span>
            </button>
          {{else}}
            <a class="attachment-open" href="{{.URL}}" target="_blank" rel="noopener">
              <span class="attachment-file-icon">{{if .IsPDF}}PDF{{else}}Datei{{end}}</span>
              <span class="attachment-name">{{.Filename}}</span>
            </a>
          {{end}}
          {{if .CanDelete}}
            <form class="attachment-delete" method="post" action="{{.DeleteURL}}" data-confirm="Diesen Anhang entfernen?">
              <input type="hidden" name="id" value="{{.ID}}">
              {{if .DeleteRedirect}}<input type="hidden" name="redirect" value="{{.DeleteRedirect}}">{{end}}
              <button type="submit" aria-label="Anhang entfernen">&times;</button>
            </form>
          {{end}}
        </div>
      {{end}}
    </div>
  {{end}}
{{end}}

{{define "issueEstimate"}}
  {{if or .HasEstimate .HasEstimateAttachments}}
    <div class="issue-estimate">
      <div class="issue-estimate-head">
        <strong>Kostenvoranschlag</strong>
        {{if .EstimateAmount}}<span class="pill">{{.EstimateAmount}}</span>{{end}}
      </div>
      {{if .EstimateNote}}<p>{{.EstimateNote}}</p>{{end}}
      {{template "attachmentStrip" .EstimateAttachmentGroup}}
      <p class="mini">Orientierung für die Bearbeitung, keine Rechnung und kein Zahlungsstatus.</p>
    </div>
  {{end}}
  {{if .CanEditEstimate}}
    <form class="issue-actions" method="post" action="/app/anliegen/workflow" enctype="multipart/form-data">
      <input type="hidden" name="id" value="{{.ID}}">
      <input type="hidden" name="status" value="{{.Status}}">
      {{if not .CanServiceUpdate}}<input type="hidden" name="priority" value="{{.Priority}}"><input type="hidden" name="assignee_email" value="{{.AssigneeEmail}}">{{end}}
      <label>Kostenschätzung
        <input type="text" name="estimate_amount" value="{{.EstimateAmountValue}}" inputmode="decimal" placeholder="z. B. 240,00">
      </label>
      <label class="note">Notiz
        <input type="text" name="estimate_note" value="{{.EstimateNote}}" maxlength="240" placeholder="Kurz einordnen, kein Rechnungsstatus">
      </label>
      <label class="comment-upload">
        <span class="file-control"><input type="file" name="estimate_attachment" accept="image/jpeg,image/png,image/webp,image/gif,application/pdf"><span>Kostenvoranschlag anhängen</span></span>
      </label>
      <button type="submit">Kostenvoranschlag speichern</button>
    </form>
  {{end}}
{{end}}

{{define "issueComment"}}
  <div class="comment" id="comment-{{.ID}}">
    <div class="comment-head">
      <span class="comment-meta">{{if .KindLabel}}<strong>{{.KindLabel}}</strong> · {{end}}{{.Author}} · {{.CreatedAt}}</span>
      {{if .CanDelete}}
        <form class="comment-delete" method="post" action="{{.DeleteURL}}" data-confirm="Diesen Kommentar löschen?">
          <input type="hidden" name="comment_id" value="{{.ID}}">
          <button type="submit">Löschen</button>
        </form>
      {{end}}
    </div>
    <p>{{.Body}}</p>
    {{template "attachmentStrip" .}}
  </div>
{{end}}

{{define "issueCreateForm"}}
              <form class="issue-form" id="issue-create-form" method="post" action="/app/anliegen" enctype="multipart/form-data" data-issue-wizard>
                <fieldset class="issue-wizard-step" data-issue-step="describe">
                  <div class="issue-wizard-heading">
                    <span class="issue-wizard-progress" data-issue-progress>Schritt 1 von 2</span>
                    <h3>Was ist passiert?</h3>
                    <p>Kurz beschreiben, einordnen und bei Bedarf ein Foto ergänzen.</p>
                  </div>
                  <p class="issue-safety-note">Akute Gefahr? 112 anrufen. Bei Wasseraustritt zuerst die <a href="/app/kontakte">Hauskontakte</a> öffnen. Erst danach hier melden.</p>
                  <label for="issue-body">Kurze Beschreibung
                    <textarea id="issue-body" name="body" maxlength="4000" required aria-describedby="issue-body-error" placeholder="Zum Beispiel: Das Licht im Keller funktioniert nicht mehr."></textarea>
                    <span class="issue-field-error" id="issue-body-error" role="alert" hidden></span>
                  </label>
                  <div class="issue-category">
                    <span id="issue-category-label">Art des Anliegens</span>
                    <div class="issue-category-options" role="radiogroup" aria-labelledby="issue-category-label" aria-describedby="issue-category-error">
                      <label class="issue-category-choice"><input type="radio" name="category" value="Reparatur" required checked><span>Reparatur</span></label>
                      <label class="issue-category-choice"><input type="radio" name="category" value="Frage" required><span>Frage</span></label>
                      <label class="issue-category-choice"><input type="radio" name="category" value="Vorschlag" required><span>Vorschlag</span></label>
                      <label class="issue-category-choice"><input type="radio" name="category" value="Sonstiges" required><span>Sonstiges</span></label>
                    </div>
                    <p class="issue-field-error" id="issue-category-error" role="alert" hidden></p>
                  </div>
                  <div class="issue-location-options" role="radiogroup" aria-label="Bereich" aria-describedby="issue-location-error">
                    <label class="issue-location-choice">
                      <input type="radio" name="location_type" value="common" required checked>
                      <strong>Gemeinschaftsbereich</strong>
                      <span>Stiegenhaus, Keller oder Garage</span>
                    </label>
                    <label class="issue-location-choice">
                      <input type="radio" name="location_type" value="own-unit" required>
                      <strong>Eigene Einheit</strong>
                      <span>Wohnung oder eigener Nebenraum</span>
                    </label>
                  </div>
                  <p class="issue-field-error" id="issue-location-error" role="alert" hidden></p>
                  <label for="issue-location-detail">Wo genau? <span class="hint">Optional</span>
                    <input id="issue-location-detail" type="text" name="location_detail" maxlength="160" placeholder="Zum Beispiel: Vorraum oder neben dem Fahrradraum">
                  </label>
                  <div class="issue-file-row">
                    <label>
                      <span class="file-control"><input type="file" name="attachments" accept="image/jpeg,image/png,image/webp,image/gif,application/pdf" multiple><span>Foto oder Datei hinzufügen</span></span>
                    </label>
                    <span class="hint">Optional · bis zu 10 Dateien, jeweils 10 MB</span>
                  </div>
                  <p class="hint">Keine Gesundheitsdaten, Ausweiskopien oder unnötig abgebildete Personen.</p>
                  <div class="issue-wizard-actions">
                    {{if .HasIssues}}<button class="wizard-cancel wizard-only" type="button">Abbrechen</button>{{else}}<a class="wizard-exit" href="/app">Abbrechen</a>{{end}}
                    <button class="wizard-next wizard-only" type="button" data-issue-next>Weiter</button>
                  </div>
                </fieldset>

                <fieldset class="issue-wizard-step" data-issue-step="review">
                  <div class="issue-wizard-heading">
                    <span class="issue-wizard-progress" data-issue-progress>Schritt 2 von 2</span>
                    <h3>Prüfen &amp; senden</h3>
                    <p>Ein kurzer Blick, dann ist die Verwaltung informiert.</p>
                  </div>
                  <dl class="issue-review issue-review-enhanced" aria-label="Zusammenfassung">
                    <div class="issue-review-row"><dt>Art</dt><dd data-issue-summary="category">Reparatur</dd></div>
                    <div class="issue-review-row"><dt>Beschreibung</dt><dd class="issue-review-copy"><span class="issue-review-copy-text" data-issue-summary="body">—</span><button class="issue-review-expand" type="button" data-issue-review-expand aria-expanded="false" hidden>Vollständig lesen</button></dd></div>
                    <div class="issue-review-row"><dt>Ort</dt><dd data-issue-summary="location">Gemeinschaftsbereich</dd></div>
                    <div class="issue-review-row"><dt>Dateien</dt><dd data-issue-summary="files">Keine</dd></div>
                  </dl>
                  <details class="issue-title-option">
                    <summary>Titel ändern <span class="hint">Optional</span></summary>
                    <label for="issue-title">Eigener Titel
                      <input id="issue-title" type="text" name="title" maxlength="140" placeholder="Kurzer, passender Titel">
                      <span class="hint">Ohne Eingabe wird ein kurzer Titel aus der Beschreibung gebildet.</span>
                    </label>
                  </details>
                  <div class="issue-wizard-actions">
                    <button class="wizard-back wizard-only" type="button" data-issue-back>Zurück</button>
                    <button class="wizard-submit" type="submit" data-busy-label="Meldung wird gesendet…">Anliegen melden</button>
                  </div>
                </fieldset>
              </form>
{{end}}

{{define "issues"}}
{{template "appOpen" .}}
    <script src="/assets/attachments.js?v={{.AssetVersion}}" defer></script>
    <script src="/assets/issues.js?v={{.AssetVersion}}" defer></script>
    <style>
      /* Ohne bestehendes Anliegen steht der kurze Meldeweg direkt offen. */
      .app-main .content-top .page-actions .button { min-height: 44px; }
      .issue-start { display: grid; grid-template-columns: minmax(0,1fr); justify-items: center; }
      .issue-start > .issue-create-panel { width: min(860px,100%); }
      .issue-create-static { padding: 0; overflow: hidden; }
      .issue-create-head { border-bottom: 1px solid var(--line); padding: 17px 20px; background: var(--panel-soft); }
      .issue-create-head h2 { font-size: 22px; }
      .issue-create-head p { margin-top: 4px; color: var(--muted); font-size: 14px; line-height: 1.4; }
      .issue-start .issue-form { max-width: none; }
      @media (max-width: 900px) {
        .issue-create-head { padding: 15px 16px; }
        .issue-start { gap: 14px; }
      }
    </style>
    <main id="main-content" tabindex="-1" class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M5 18.5V6.5A2.5 2.5 0 0 1 7.5 4h9A2.5 2.5 0 0 1 19 6.5v6A2.5 2.5 0 0 1 16.5 15H10l-5 3.5z"/></svg><span>/</span><span>Anliegen</span></span>
        {{if or .CanManageIssues (and .IsServiceProvider .HasCalendarFeedURL)}}<div class="page-actions">
          {{if and .IsServiceProvider .HasCalendarFeedURL}}<a class="button" href="{{.CalendarFeedURL}}">Kalender abonnieren</a>{{end}}
          {{if .CanManageIssues}}{{if .BoardOnly}}<a class="button" href="/app/anliegen">Zurück zu Anliegen</a>{{else}}<a class="button" href="/app/anliegen/board">Triage-Board</a>{{end}}{{end}}
        </div>{{end}}
      </div>
      <section class="page{{if .BoardOnly}} issue-board-page{{end}}">
        <div>
          <h1>{{if .BoardOnly}}Anliegen bearbeiten{{else}}Anliegen{{end}}</h1>
          <p class="lede">{{if .BoardOnly}}Offene Meldungen priorisieren und in den nächsten Schritt bringen.{{else}}Mängel, Fragen und Vorschläge direkt an die Verwaltung melden.{{end}}</p>
        </div>
        {{if .HasServiceProviderContacts}}<datalist id="service-provider-contacts">{{range .ServiceProviderContacts}}<option value="{{.Email}}">{{.Label}}</option>{{end}}</datalist>{{end}}
        {{if not .BoardOnly}}
        <div class="issue-dashboard">
          {{if .IssueMsg}}<p class="issue-flash{{if .IssueOK}} ok{{else}} warn{{end}}">{{.IssueMsg}}</p>{{end}}
          {{if .HasIssues}}
          <section class="panel" id="issue-own">
            <div class="section-head">
              <div>
                <div class="kicker">{{if and .CanManageIssues (not .BoardOnly)}}Meine eigenen Anliegen{{else if eq .Role "Beirat"}}Anliegen im Haus{{else}}Meine letzten Anliegen{{end}}</div>
                <p class="muted">Status und nächster Schritt auf einen Blick.</p>
              </div>
            </div>
            <div class="issue-list">
              {{range .Issues}}
                <article class="issue-card" id="issue-{{.ID}}">
                  <div class="issue-card-head">
                    <div>
                      <h3>{{.Title}}</h3>
                      <p class="issue-location">{{.Location}}{{if .HasAssignee}} · Zuständig: {{.AssigneeEmail}}{{end}}</p>
                    </div>
                    <div class="issue-meta">
                      <span class="pill {{.StatusClass}}">{{.Status}}</span>
                      <span class="pill">{{.Category}}</span>
                      <span class="pill">{{.Priority}}</span>
                      <span>{{.CreatedAt}}</span>
                    </div>
                  </div>
                  <div class="issue-progress" aria-label="Bearbeitungsfortschritt"><span class="issue-progress-fill {{.StatusClass}}"></span></div>
                  <p class="issue-next-step">{{.NextStep}}</p>
                  {{if .HasServiceAppointment}}<p class="issue-proposal"><strong>Termin:</strong> {{.ServiceAppointment}}</p>{{end}}
                  {{if .HasServiceProposal}}<p class="issue-proposal"><strong>Hinweis:</strong> {{.ServiceProposal}}</p>{{end}}
                  {{if .CanServiceUpdate}}
                  <details class="issue-card-details">
                    <summary>Details &amp; Verlauf{{if .HasComments}} · {{len .Comments}} {{if eq (len .Comments) 1}}Beitrag{{else}}Beiträge{{end}}{{end}}{{if .HasPhotos}} · {{.PhotoCount}} Foto{{if ne .PhotoCount 1}}s{{end}}{{else if .HasAttachments}} · Anhänge{{end}}</summary>
                    <div class="issue-card-details-body">
                      <div class="issue-description"><strong>Beschreibung</strong><p>{{.Body}}</p></div>
                      {{template "issueEstimate" .}}
                      {{template "attachmentStrip" .}}
                      {{if .HasComments}}<div class="comment-thread">{{range .Comments}}{{template "issueComment" .}}{{end}}</div>{{else}}<p class="muted">Noch keine Rückmeldung. Der aktuelle Status steht oben.</p>{{end}}
                      {{if .CanComment}}<form class="comment-form" method="post" action="/app/anliegen/comment" enctype="multipart/form-data">
                        <input type="hidden" name="id" value="{{.ID}}">
                        <textarea name="body" maxlength="3000" placeholder="Kommentar oder Ergänzung schreiben" aria-label="Kommentar oder Ergänzung"></textarea>
                        <label class="comment-upload">
                          <span class="file-control"><input type="file" name="attachments" accept="image/jpeg,image/png,image/webp,image/gif,application/pdf" multiple><span>Anhang hinzufügen</span></span>
                          <span class="hint">Nur nötige Unterlagen teilen; keine Gesundheitsdaten oder Ausweiskopien.</span>
                        </label>
                        <button type="submit">Kommentar senden</button>
                      </form>{{end}}
                      {{if .CanServiceUpdate}}
                        <form class="issue-actions" method="post" action="/app/anliegen/workflow">
                          <input type="hidden" name="id" value="{{.ID}}">
                          <label>Status<select name="status">{{range .ServiceStatusOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}</select></label>
                          <label class="proposal">Termin (Start)<input type="datetime-local" name="service_start" value="{{.ServiceStartInput}}"></label>
                          <label class="proposal">Termin (Ende, optional)<input type="datetime-local" name="service_end" value="{{.ServiceEndInput}}"></label>
                          <label class="proposal">Hinweis (optional)<input type="text" name="service_proposal" value="{{.ServiceProposal}}" maxlength="180" placeholder="z. B. Zugang über Hinterhof"></label>
                          <button type="submit">Status senden</button>
                        </form>
                      {{end}}
                      {{if or .CanClose .CanReopen}}<div class="issue-actions">
                        {{if .CanClose}}<form method="post" action="/app/anliegen/workflow"><input type="hidden" name="id" value="{{.ID}}"><input type="hidden" name="status" value="Erledigt"><button class="ghost" type="submit">Als erledigt melden</button></form>{{end}}
                        {{if .CanReopen}}<form method="post" action="/app/anliegen/workflow"><input type="hidden" name="id" value="{{.ID}}"><input type="hidden" name="status" value="Neu"><button class="ghost" type="submit">Wieder öffnen</button></form>{{end}}
                      </div>{{end}}
                    </div>
                  </details>
                  {{else}}
                    <a class="issue-detail-link" href="{{.DetailURL}}"><span>{{if eq .ResidentState "question"}}Ihre Antwort wird gebraucht{{else if eq .ResidentState "resolution"}}Ist das Anliegen erledigt?{{else}}Details und Verlauf{{end}}</span><strong>{{.DetailAction}} ›</strong></a>
                  {{end}}
                </article>
              {{end}}
            </div>
          </section>
          {{end}}

          {{if .CanCreateIssue}}
          {{if .HasIssues}}
          <details class="panel issue-create-panel" id="issue-new"{{if .OpenIssueCreate}} open{{end}}>
            <summary>
              <div>
                <h2>Neues Anliegen</h2>
                <p>In zwei kurzen Schritten verständlich melden.</p>
              </div>
            </summary>
            <div class="issue-create-body">{{template "issueCreateForm" .}}</div>
          </details>
          {{else}}
          <div class="issue-start">
            <section class="panel issue-create-panel issue-create-static" id="issue-new">
              <div class="issue-create-head">
                <h2>Erstes Anliegen melden</h2>
                <p>In zwei kurzen Schritten verständlich melden.</p>
              </div>
              <div class="issue-create-body">{{template "issueCreateForm" .}}</div>
            </section>
          </div>
          {{end}}
          {{else if not .HasIssues}}
            {{template "emptyState" .IssuesEmpty}}
          {{end}}
        </div>{{end}}
        {{if and .CanManageIssues .BoardOnly}}
          <section class="panel issue-board-panel" id="issue-manage">
            {{if .TotalIssueCount}}<div class="issue-board-toolbar">
              <div class="issue-board-summary" aria-label="Anliegen-Überblick">
                <span class="pill">{{.OpenIssueCount}} offen</span>
                {{if .UrgentIssueCount}}<span class="pill dringend">{{.UrgentIssueCount}} dringend</span>{{end}}
                <span class="pill">{{.TotalIssueCount}} gesamt</span>
              </div>
              <details class="issue-board-tools"{{if .BoardFilters.HasActive}} open{{end}}>
                <summary>
                  <span class="issue-filter-icon" aria-hidden="true"><svg viewBox="0 0 24 24"><path d="M4 7h10M18 7h2M4 17h2M10 17h10M8 4v6M16 14v6"/></svg></span>
                  <strong>Filter &amp; Sortierung</strong>
                  <span>{{if .BoardFilters.HasActive}}Aktive Auswahl{{else}}Alle Anliegen · zuletzt aktualisiert{{end}}</span>
                </summary>
                <form class="issue-board-filter" method="get" action="{{.BoardAction}}">
                  <label>Status
                    <select name="status">
                      {{range .BoardFilters.StatusOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}
                    </select>
                  </label>
                  <label>Priorität
                    <select name="priority">
                      {{range .BoardFilters.PriorityOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}
                    </select>
                  </label>
                  <label>Kategorie
                    <select name="category">
                      {{range .BoardFilters.CategoryOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}
                    </select>
                  </label>
                  <label class="assignee">Zuständig
                    <input type="email" name="assignee" value="{{.BoardFilters.Assignee}}" placeholder="name@example.com"{{if .HasServiceProviderContacts}} list="service-provider-contacts"{{end}}>
                  </label>
                  <label class="sort">Sortierung
                    <select name="sort">
                      {{range .BoardFilters.SortOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}
                    </select>
                  </label>
                  <div class="board-filter-actions">
                    <button type="submit">Anwenden</button>
                    {{if .BoardFilters.HasActive}}<a href="{{.BoardAction}}">Zurücksetzen</a>{{end}}
                  </div>
                </form>
              </details>
              <nav class="issue-board-quick" aria-label="Schnellauswahl">
                <span>Ansicht</span>
                <a href="{{.BoardAction}}"{{if not .BoardFilters.HasActive}} aria-current="true"{{end}}>Alle</a>
                <a href="{{.BoardAction}}?status=Neu"{{if eq .BoardFilters.Status "Neu"}} aria-current="true"{{end}}>Neu</a>
                <a href="{{.BoardAction}}?status=In+Bearbeitung"{{if eq .BoardFilters.Status "In Bearbeitung"}} aria-current="true"{{end}}>In Bearbeitung</a>
                <a href="{{.BoardAction}}?priority=Dringend"{{if eq .BoardFilters.Priority "Dringend"}} aria-current="true"{{end}}>Dringend</a>
                <a href="{{.BoardAction}}?assignee={{.Email}}"{{if eq .BoardFilters.Assignee .Email}} aria-current="true"{{end}}>Mir zugewiesen</a>
                <a href="{{.BoardAction}}?sort=age"{{if eq .BoardFilters.Sort "age"}} aria-current="true"{{end}}>Älteste zuerst</a>
              </nav>
            </div>{{end}}
            {{if .HasManageIssues}}
              <div class="issue-list">
                {{range .ManageIssues}}
                  <article class="issue-card issue-work-card {{.StatusClass}}" id="issue-{{.ID}}">
                    <div class="issue-card-head">
                      <div>
                        <h3>{{.Title}}</h3>
                        <p class="issue-location">{{.Author}} · {{.Location}}{{if .HasAssignee}} · Zuständig: {{.AssigneeEmail}}{{end}} · {{.CreatedAt}}</p>
                      </div>
                      <div class="issue-meta">
                        <span class="pill {{.StatusClass}}">{{.Status}}</span>
                        <span class="pill">{{.Priority}}</span>
                        <span class="pill">{{.Category}}</span>
                      </div>
                    </div>
                    <div class="issue-progress" aria-label="Bearbeitungsfortschritt"><span class="issue-progress-fill {{.StatusClass}}"></span></div>
                    <p class="issue-next-step">{{.NextStep}}</p>
                    {{if .HasServiceAppointment}}<p class="issue-proposal"><strong>Termin:</strong> {{.ServiceAppointment}}</p>{{end}}
                    {{if .HasServiceProposal}}<p class="issue-proposal"><strong>Hinweis:</strong> {{.ServiceProposal}}</p>{{end}}
                    <div class="issue-card-foot">
                      <div class="issue-description issue-description-preview"><p>{{.Body}}</p></div>
                      <a class="button primary issue-triage-link" href="/app/anliegen/board/{{.ID}}">Bearbeiten</a>
                    </div>
                  </article>
                {{end}}
              </div>
            {{else if .TotalIssueCount}}
              <div class="board-filter-blank">
                <span class="board-blank-icon" aria-hidden="true"><svg viewBox="0 0 24 24"><path d="M4 6h16M7 12h10M10 18h4"/></svg></span>
                <div>
                  <h2>Kein Anliegen passt zu dieser Auswahl</h2>
                  <p>Im Haus sind {{.TotalIssueCount}} Anliegen erfasst, davon {{.OpenIssueCount}} offen. Setzen Sie die Auswahl zurück oder wählen Sie einen anderen Filter.</p>
                </div>
                <a class="button primary" href="{{.BoardAction}}">Auswahl zurücksetzen</a>
              </div>
            {{else}}
              <section class="board-blank" aria-labelledby="board-blank-title">
                <div class="board-blank-main">
                  <div class="board-blank-lead">
                    <span class="board-blank-icon" aria-hidden="true"><svg viewBox="0 0 24 24"><path d="M5 18.5V6.5A2.5 2.5 0 0 1 7.5 4h9A2.5 2.5 0 0 1 19 6.5v6A2.5 2.5 0 0 1 16.5 15H10l-5 3.5z"/><path d="M9 8h6M9 11h4"/></svg></span>
                    <h2 id="board-blank-title">{{.ManageIssuesEmpty.Title}}</h2>
                    <p>Sobald jemand im Haus etwas meldet, steht es hier: Titel, meldende Person, Ort, Status, Priorität und der nächste Schritt. Zuletzt aktualisierte Anliegen stehen oben; die Bearbeitung öffnen Sie direkt aus der Zeile.</p>
                  </div>
                  {{if or .CanCreateIssue .CanManageAnnouncements}}<div class="board-blank-actions">
                    {{if .CanCreateIssue}}<a class="button primary" href="/app/anliegen?new=1">Anliegen selbst anlegen</a>{{end}}
                    {{if .CanManageAnnouncements}}<a class="button" href="/app/announcements">Meldeweg am Aushang erklären</a>{{end}}
                  </div>{{end}}
                </div>
                <aside class="board-blank-side">
                  <h2>Der Weg eines Anliegens</h2>
                  <ol class="board-blank-steps">
                    <li><strong>Neu</strong><span>Die Meldung ist eingegangen. Erster Schritt: Dringlichkeit festlegen.</span></li>
                    <li><strong>Angenommen</strong><span>Die Zuständigkeit ist vergeben, die Bearbeitung übernommen.</span></li>
                    <li><strong>Termin vereinbart</strong><span>Ein Termin mit Bewohnerschaft oder Dienstleister steht fest.</span></li>
                    <li><strong>In Bearbeitung</strong><span>Die Arbeit läuft. Informationen und Rückfragen gehen von hier an die meldende Person.</span></li>
                    <li><strong>Erledigt</strong><span>Die Lösung geht zur Prüfung; bestätigt wird von der meldenden Person.</span></li>
                  </ol>
                  <p class="board-blank-note">Meldungen, die nicht bearbeitet werden, schließen Sie als „Abgelehnt“ oder „Duplikat“.</p>
                </aside>
                <ul class="board-blank-facts">
                  <li><strong>Reihenfolge</strong><span>Standard ist „zuletzt aktualisiert“. Ab dem ersten Anliegen stehen zusätzlich Älteste zuerst, Priorität, Status, Kategorie und Zuständigkeit als Sortierung bereit.</span></li>
                  <li><strong>Priorität</strong><span>Dringend, Hoch, Mittel und Niedrig. Der erste Triage-Schritt setzt sie; die Zählung über der Liste zeigt offene und dringende Anliegen.</span></li>
                  <li><strong>Zuständigkeit</strong><span>Der zweite Schritt ordnet das Anliegen Ihnen zu oder lässt es offen. Danach lässt sich die Liste auf Ihre eigenen Fälle einschränken.</span></li>
                </ul>
              </section>
            {{end}}
          </section>
        {{end}}
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "issueTriage"}}
{{template "appOpen" .}}
    <script src="/assets/attachments.js?v={{.AssetVersion}}" defer></script>
    <main id="main-content" tabindex="-1" class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M5 18.5V6.5A2.5 2.5 0 0 1 7.5 4h9A2.5 2.5 0 0 1 19 6.5v6A2.5 2.5 0 0 1 16.5 15H10l-5 3.5z"/></svg><a href="/app/anliegen/board">Anliegen</a><span>/</span><span>{{.Issue.Title}}</span></span>
      </div>
      <section class="page issue-triage-page">
        <header class="issue-triage-context">
          <a class="issue-triage-back" href="/app/anliegen/board">← Zurück zur Liste</a>
          <h1>{{.Issue.Title}}</h1>
          <div class="issue-triage-facts">
            <span>{{.Issue.Author}}</span>
            <span>{{.Issue.Location}}</span>
            <span>{{.Issue.CreatedAt}}</span>
            <span>{{.Issue.Category}}</span>
          </div>
          <p>{{.Issue.Body}}</p>
        </header>

        <div class="issue-triage-layout">
          <div class="issue-triage-main">
        {{if eq .TriageStep "1"}}
          <form class="issue-triage-card" method="post" action="/app/anliegen/workflow">
            <input type="hidden" name="id" value="{{.Issue.ID}}">
            <input type="hidden" name="status" value="{{.Issue.Status}}">
            <input type="hidden" name="assignee_email" value="{{.Issue.AssigneeEmail}}">
            <input type="hidden" name="redirect" value="/app/anliegen/board/{{.Issue.ID}}?step=2">
            <div class="issue-triage-progress"><span>Schritt 1 von 2</span><span aria-hidden="true"><i class="active"></i><i></i></span></div>
            <fieldset>
              <legend>Wie dringend ist das Anliegen?</legend>
              <label class="issue-triage-choice urgent">
                <input type="radio" name="priority" value="Dringend"{{if eq .Issue.Priority "Dringend"}} checked{{end}} required>
                <span><strong>Heute kümmern</strong><small>Sicherheitsrisiko oder akuter Schaden</small></span>
              </label>
              <label class="issue-triage-choice">
                <input type="radio" name="priority" value="Hoch"{{if eq .Issue.Priority "Hoch"}} checked{{end}} required>
                <span><strong>Diese Woche</strong><small>Bald bearbeiten, aber nicht akut</small></span>
              </label>
              <label class="issue-triage-choice calm">
                <input type="radio" name="priority" value="Niedrig"{{if eq .Issue.Priority "Niedrig"}} checked{{end}} required>
                <span><strong>Kann warten</strong><small>Bei Gelegenheit einplanen</small></span>
              </label>
            </fieldset>
            <div class="issue-triage-actions">
              <a href="/app/anliegen/board">Abbrechen</a>
              <button type="submit">Weiter</button>
            </div>
          </form>
        {{else if eq .TriageStep "2"}}
          <form class="issue-triage-card" method="post" action="/app/anliegen/workflow">
            <input type="hidden" name="id" value="{{.Issue.ID}}">
            <input type="hidden" name="status" value="In Bearbeitung">
            <input type="hidden" name="priority" value="{{.Issue.Priority}}">
            <input type="hidden" name="redirect" value="/app/anliegen/board/{{.Issue.ID}}?step=done">
            <div class="issue-triage-progress"><span>Schritt 2 von 2</span><span aria-hidden="true"><i class="active"></i><i class="active"></i></span></div>
            <fieldset>
              <legend>Wer kümmert sich als Nächstes?</legend>
              <label class="issue-triage-choice">
                <input type="radio" name="assignee_email" value="{{.ActorEmail}}"{{if eq .Issue.AssigneeEmail .ActorEmail}} checked{{end}} required>
                <span><strong>Ich übernehme</strong><small>Das Anliegen wird Ihnen zugeordnet</small></span>
              </label>
              <label class="issue-triage-choice calm">
                <input type="radio" name="assignee_email" value=""{{if not .Issue.HasAssignee}} checked{{end}} required>
                <span><strong>Noch offen lassen</strong><small>Die Zuständigkeit wird später festgelegt</small></span>
              </label>
              {{if and .Issue.HasAssignee (ne .Issue.AssigneeEmail .ActorEmail)}}
                <label class="issue-triage-choice">
                  <input type="radio" name="assignee_email" value="{{.Issue.AssigneeEmail}}" checked required>
                  <span><strong>Bestehende Zuordnung behalten</strong><small>{{.Issue.AssigneeEmail}}</small></span>
                </label>
              {{end}}
            </fieldset>
            <div class="issue-triage-actions">
              <a href="/app/anliegen/board/{{.Issue.ID}}">Zurück</a>
              <button type="submit">Bearbeitung starten</button>
            </div>
          </form>
        {{else if eq .TriageStep "done"}}
          <section class="issue-triage-card issue-triage-done">
            <div class="issue-triage-done-mark" aria-hidden="true">✓</div>
            <div>
              <span class="kicker">Gespeichert</span>
              <h2>Der nächste Schritt ist festgelegt.</h2>
              <p><strong>{{.Issue.Priority}}</strong> · {{if .Issue.HasAssignee}}Zuständig: {{.Issue.AssigneeEmail}}{{else}}Zuständigkeit noch offen{{end}}</p>
            </div>
            <div class="issue-triage-actions">
              <a href="/app/anliegen/board/{{.Issue.ID}}">Entscheidung ändern</a>
              <a class="button primary" href="/app/anliegen/board/{{.Issue.ID}}?step=message">Bewohner kontaktieren</a>
            </div>
          </section>
        {{else if eq .TriageStep "message"}}
          <section class="issue-triage-card issue-message-card">
            <div class="issue-triage-progress"><span>Nächster Schritt</span></div>
            <form class="issue-message-form" method="post" action="/app/anliegen/comment" enctype="multipart/form-data">
              <input type="hidden" name="id" value="{{.Issue.ID}}">
              <input type="hidden" name="redirect" value="/app/anliegen/board/{{.Issue.ID}}?step=sent">
              <fieldset>
                <legend>Was soll der Bewohner wissen?</legend>
                <label class="issue-triage-choice calm">
                  <input type="radio" name="message_type" value="information" checked required>
                  <span><strong>Information senden</strong><small>Nur informieren, keine Antwort nötig</small></span>
                </label>
                <label class="issue-triage-choice">
                  <input type="radio" name="message_type" value="question" required>
                  <span><strong>Rückfrage stellen</strong><small>Der Bewohner erhält eine klare Antwort-Aufgabe</small></span>
                </label>
              </fieldset>
              <label class="issue-message-body">Nachricht
                <textarea name="body" maxlength="3000" required placeholder="Kurz und konkret formulieren" aria-label="Nachricht an Bewohner"></textarea>
              </label>
              <label class="comment-upload">
                <span class="file-control"><input type="file" name="attachments" accept="image/jpeg,image/png,image/webp,image/gif,application/pdf" multiple><span>Datei optional hinzufügen</span></span>
              </label>
              <div class="issue-triage-actions">
                <a href="/app/anliegen/board">Zur Liste</a>
                <button type="submit">Nachricht senden</button>
              </div>
            </form>
            {{if ne .Issue.Status "Erledigt"}}
              <form class="issue-resolution-propose" method="post" action="/app/anliegen/workflow">
                <input type="hidden" name="id" value="{{.Issue.ID}}">
                <input type="hidden" name="status" value="Erledigt">
                <input type="hidden" name="priority" value="{{.Issue.Priority}}">
                <input type="hidden" name="assignee_email" value="{{.Issue.AssigneeEmail}}">
                <input type="hidden" name="redirect" value="/app/anliegen/board/{{.Issue.ID}}?step=resolution-sent">
                <span><strong>Arbeit abgeschlossen?</strong><small>Der Bewohner prüft und bestätigt die Lösung.</small></span>
                <button class="ghost" type="submit">Lösung zur Prüfung senden</button>
              </form>
            {{end}}
          </section>
        {{else if eq .TriageStep "sent"}}
          <section class="issue-triage-card issue-triage-done">
            <div class="issue-triage-done-mark" aria-hidden="true">✓</div>
            <div>
              <span class="kicker">Nachricht gesendet</span>
              {{if .Issue.HasOpenQuestion}}
                <h2>Der Bewohner sieht jetzt „Antworten“.</h2>
                <p>Bis zur Antwort ist keine weitere Aktion nötig.</p>
              {{else}}
                <h2>Die Information ist im Verlauf sichtbar.</h2>
                <p>Der Bewohner muss darauf nicht reagieren.</p>
              {{end}}
            </div>
            <div class="issue-triage-actions">
              <a href="/app/anliegen/board/{{.Issue.ID}}?step=message">Weitere Nachricht</a>
              <a class="button primary" href="/app/anliegen/board">Zur Liste</a>
            </div>
          </section>
        {{else}}
          <section class="issue-triage-card issue-triage-done">
            <div class="issue-triage-done-mark" aria-hidden="true">✓</div>
            <div>
              <span class="kicker">Lösung vorgeschlagen</span>
              <h2>Der Bewohner prüft jetzt das Ergebnis.</h2>
              <p>Er kann „Ja, erledigt“ bestätigen oder das Anliegen wieder öffnen.</p>
            </div>
            <div class="issue-triage-actions">
              <a href="/app/anliegen/board/{{.Issue.ID}}?step=message">Nachricht senden</a>
              <a class="button primary" href="/app/anliegen/board">Zur Liste</a>
            </div>
          </section>
        {{end}}
          </div>
          <aside class="issue-triage-aside">
            <section class="issue-triage-state">
              <h2>Stand des Anliegens</h2>
              <dl>
                <div><dt>Status</dt><dd><span class="pill {{.Issue.StatusClass}}">{{.Issue.Status}}</span></dd></div>
                <div><dt>Priorität</dt><dd>{{.Issue.Priority}}</dd></div>
                <div><dt>Zuständig</dt><dd>{{if .Issue.HasAssignee}}{{.Issue.AssigneeEmail}}{{else}}noch offen{{end}}</dd></div>
                <div><dt>Kategorie</dt><dd>{{.Issue.Category}}</dd></div>
                <div><dt>Gemeldet</dt><dd>{{.Issue.CreatedAt}}</dd></div>
              </dl>
              {{if .Issue.HasServiceAppointment}}<p class="issue-triage-state-line"><strong>Termin:</strong> {{.Issue.ServiceAppointment}}</p>{{end}}
              {{if .Issue.HasServiceProposal}}<p class="issue-triage-state-line"><strong>Hinweis:</strong> {{.Issue.ServiceProposal}}</p>{{end}}
              <p class="issue-triage-state-next">{{.Issue.NextStep}}</p>
            </section>
            <details class="issue-triage-more">
              <summary>Verlauf und Unterlagen{{if .Issue.HasComments}} · {{len .Issue.Comments}} {{if eq (len .Issue.Comments) 1}}Beitrag{{else}}Beiträge{{end}}{{end}}{{if .Issue.HasPhotos}} · {{.Issue.PhotoCount}} Foto{{if ne .Issue.PhotoCount 1}}s{{end}}{{end}}</summary>
              <div class="issue-triage-more-body">
                {{template "attachmentStrip" .Issue}}
                {{if .Issue.HasComments}}<div class="comment-thread">{{range .Issue.Comments}}{{template "issueComment" .}}{{end}}</div>{{else}}<p class="muted">Noch keine Rückmeldung.</p>{{end}}
                {{template "issueEstimate" .Issue}}
              </div>
            </details>
          </aside>
        </div>
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "issueResidentDetail"}}
{{template "appOpen" .}}
    <script src="/assets/attachments.js?v={{.AssetVersion}}" defer></script>
    <main id="main-content" tabindex="-1" class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M5 18.5V6.5A2.5 2.5 0 0 1 7.5 4h9A2.5 2.5 0 0 1 19 6.5v6A2.5 2.5 0 0 1 16.5 15H10l-5 3.5z"/></svg><a href="/app/anliegen">Anliegen</a><span>/</span><span>{{.Issue.Title}}</span></span>
      </div>
      <section class="page issue-resident-page" id="issue-{{.Issue.ID}}">
        {{if .IssueCreated}}
          <div class="issue-created-note" role="status">
            <span aria-hidden="true">✓</span>
            <strong>Anliegen gemeldet</strong>
            <small>Die Verwaltung wurde informiert.</small>
          </div>
        {{end}}
        <header class="issue-resident-context">
          <a class="issue-triage-back" href="/app/anliegen">← Zurück zu Anliegen</a>
          <div class="issue-resident-title-row">
            <div>
              <h1>{{.Issue.Title}}</h1>
              <p>{{.Issue.Location}} · {{.Issue.CreatedAt}}</p>
            </div>
            <span class="pill {{.Issue.StatusClass}}">{{.Issue.Status}}</span>
          </div>
        </header>

        {{if eq .Issue.ResidentState "question"}}
          <section class="issue-resident-task question">
            <span class="kicker">Rückfrage der Verwaltung</span>
            <h2>{{.Issue.OpenQuestion.Body}}</h2>
            <form class="issue-answer-form" method="post" action="/app/anliegen/comment" enctype="multipart/form-data">
              <input type="hidden" name="id" value="{{.Issue.ID}}">
              <input type="hidden" name="redirect" value="/app/anliegen/{{.Issue.ID}}">
              <label>Ihre Antwort
                <textarea name="body" maxlength="3000" required placeholder="Kurz antworten, damit es weitergehen kann" aria-label="Ihre Antwort"></textarea>
              </label>
              <label class="comment-upload">
                <span class="file-control"><input type="file" name="attachments" accept="image/jpeg,image/png,image/webp,image/gif,application/pdf" multiple><span>Foto optional hinzufügen</span></span>
              </label>
              <button type="submit">Antwort senden</button>
            </form>
          </section>
        {{else if eq .Issue.ResidentState "resolution"}}
          <section class="issue-resident-task resolution">
            <span class="kicker">Lösung prüfen</span>
            <h2>Ist das Anliegen für Sie erledigt?</h2>
            <p>Die Verwaltung hat die Bearbeitung abgeschlossen.</p>
            <div class="issue-resolution-actions">
              <form method="post" action="/app/anliegen/resolution">
                <input type="hidden" name="id" value="{{.Issue.ID}}">
                <input type="hidden" name="resolved" value="yes">
                <button type="submit">Ja, erledigt</button>
              </form>
              <form method="post" action="/app/anliegen/resolution">
                <input type="hidden" name="id" value="{{.Issue.ID}}">
                <input type="hidden" name="resolved" value="no">
                <button class="ghost" type="submit">Nein, noch offen</button>
              </form>
            </div>
          </section>
        {{else if eq .Issue.ResidentState "done"}}
          <section class="issue-resident-task done">
            <div class="issue-triage-done-mark" aria-hidden="true">✓</div>
            <div><span class="kicker">Erledigt</span><h2>Sie haben die Lösung bestätigt.</h2><p>Für Sie ist nichts mehr zu tun.</p></div>
          </section>
        {{else}}
          <section class="issue-resident-task waiting">
            <span class="kicker">Aktueller Stand</span>
            <h2>{{.Issue.NextStep}}</h2>
            <p>Sie müssen im Moment nichts tun. Sobald eine Antwort gebraucht wird, erscheint sie hier eindeutig.</p>
          </section>
        {{end}}

        <section class="issue-resident-report" aria-labelledby="issue-resident-report-title">
          <div class="issue-resident-report-head">
            <h2 id="issue-resident-report-title">Ihre Meldung</h2>
            <div class="issue-resident-report-meta"><span class="pill">{{.Issue.Category}}</span><span class="pill">{{.Issue.Location}}</span>{{if .Issue.HasPhotos}}<span class="pill">{{.Issue.PhotoCount}} Foto{{if ne .Issue.PhotoCount 1}}s{{end}}</span>{{else if .Issue.HasAttachments}}<span class="pill">Anhänge</span>{{end}}</div>
          </div>
          <div class="issue-description"><p>{{.Issue.Body}}</p></div>
          {{template "attachmentStrip" .Issue}}
        </section>

        {{if or .Issue.HasComments .Issue.HasServiceAppointment .Issue.HasServiceProposal .Issue.HasEstimate .Issue.HasEstimateAttachments}}
        <details class="issue-resident-history">
          <summary>Neuigkeiten zum Anliegen{{if .Issue.HasComments}} · {{len .Issue.Comments}} {{if eq (len .Issue.Comments) 1}}Beitrag{{else}}Beiträge{{end}}{{end}}</summary>
          <div class="issue-resident-history-body">
            {{if .Issue.HasServiceAppointment}}<p class="issue-proposal"><strong>Termin:</strong> {{.Issue.ServiceAppointment}}</p>{{end}}
            {{if .Issue.HasServiceProposal}}<p class="issue-proposal"><strong>Hinweis:</strong> {{.Issue.ServiceProposal}}</p>{{end}}
            {{if .Issue.HasComments}}<div class="comment-thread">{{range .Issue.Comments}}{{template "issueComment" .}}{{end}}</div>{{end}}
            {{template "issueEstimate" .Issue}}
          </div>
        </details>
        {{end}}
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "announcements"}}
{{template "appOpen" .}}
    <script src="/assets/announcements.js?v={{.AssetVersion}}" defer></script>
    <script src="/assets/attachments.js?v={{.AssetVersion}}" defer></script>
    <style>
      .announce .announce-layout { display: grid; gap: 20px; }
      .announce .announce-main, .announce .announce-aside { min-width: 0; display: grid; gap: 16px; align-content: start; }
      .announce .announce-main.is-blank { align-content: start; }
      .announce .announce-feed { display: grid; gap: 18px; }
      .announce .announce-feed .kicker { margin-bottom: 0; }
      .announce .section-copy h2 { margin-top: 5px; }
      .announce .announce-counts { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; }
      .announce .archive-tools { margin: 0; }
      .announce .announce-group { display: grid; gap: 13px; }
      .announce .announce-group-head { display: flex; align-items: baseline; justify-content: space-between; gap: 10px; border-bottom: 1px solid var(--line); padding-bottom: 7px; }
      .announce .announce-group-head h3 { color: var(--gold-ink); font-family: var(--font-sans); font-size: 11.5px; font-weight: 850; letter-spacing: .11em; text-transform: uppercase; }
      .announce .announce-group-head span { color: var(--soft); font-size: 12px; font-weight: 750; white-space: nowrap; }
      .announce .announce-group .entries { gap: 16px; }
      .announce .announce-group .entry + .entry { padding-top: 16px; }
      .announce .entry-head { gap: 12px; }
      .announce .entry-head > :first-child { min-width: 0; }
      .announce .announcement-entry h3 { overflow-wrap: anywhere; }
      .announce .entry-actions { flex-wrap: nowrap; flex: 0 0 auto; }
      .announce .announce-filtered-empty { display: grid; gap: 13px; justify-items: start; }
      .announce .announce-blank { min-height: clamp(280px,34vh,360px); display: grid; align-content: center; justify-items: center; gap: 15px; border: 1px dashed rgba(200,153,63,.42); border-radius: var(--radius-sm); background: rgba(255,254,251,.68); padding: 32px 26px; text-align: center; }
      .announce .announce-blank .empty-state { width: 100%; border: 0; background: transparent; padding: 0; grid-template-columns: minmax(0,1fr); justify-items: center; gap: 13px; text-align: center; }
      .announce .announce-blank .empty-state p { max-width: 48ch; }
      .announce .announce-blank-actions { display: flex; flex-wrap: wrap; gap: 10px; justify-content: center; }
      .announce .announce-blank-actions .button { min-height: 44px; }
      .announce .announce-aside-panel { display: grid; gap: 14px; padding: 20px; align-content: start; }
      .announce .announce-aside-panel .kicker { margin-bottom: 0; }
      .announce .announce-aside-panel h2 { font-size: 20px; }
      .announce .announce-aside-panel > p { color: var(--muted); font-size: 13.5px; line-height: 1.5; }
      .announce .announce-legend { margin: 0; padding: 0; list-style: none; display: grid; gap: 13px; }
      .announce .announce-legend li { display: grid; gap: 5px; justify-items: start; }
      .announce .announce-legend span { color: var(--muted); font-size: 12.8px; line-height: 1.42; }
      .announce .announce-links { display: grid; }
      .announce .announce-link { min-height: 48px; display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 10px; align-items: center; border-top: 1px solid var(--line); padding: 11px 0; color: inherit; text-decoration: none; }
      .announce .announce-link:first-child { border-top: 0; padding-top: 0; }
      .announce .announce-link strong { display: block; font-size: 14px; }
      .announce .announce-link small { display: block; margin-top: 2px; color: var(--muted); font-size: 12.5px; line-height: 1.35; }
      .announce .announce-link::after { content: "\203A"; color: var(--gold-ink); font-size: 21px; line-height: 1; }
      .announce .announce-link:hover strong { color: var(--gold-ink); }
      @media (min-width: 1181px) { .announce .announce-layout { grid-template-columns: minmax(0,1fr) 320px; } }
      @media (max-width: 1180px) and (min-width: 721px) {
        .announce .announce-aside { grid-template-columns: repeat(2,minmax(0,1fr)); align-items: start; }
        .announce .entry-head { display: grid; grid-template-columns: minmax(0,1fr); }
        .announce .entry-actions { justify-content: flex-start; }
      }
      @media (max-width: 900px) and (min-width: 721px) {
        .announce .filter-form { grid-template-columns: minmax(0,1fr) auto; align-items: end; }
      }
      .announce .filter-form .button, .announce .filter-tab, .announce .announcement-body > summary, .announce .entry-actions .button { min-height: 44px; }
      .announce .filter-tab, .announce .announcement-body > summary { display: inline-flex; align-items: center; }
      @media (max-width: 720px) {
        .announce .announce-layout { gap: 12px; }
        .announce .announce-main, .announce .announce-aside { gap: 12px; }
        .announce .announce-blank { min-height: 0; padding: 26px 18px; }
        .announce .announce-blank-actions { width: 100%; }
        .announce .announce-blank-actions .button { width: 100%; }
        .announce .announce-aside-panel { padding: 16px; }
        .announce .announce-feed { grid-template-columns: minmax(0,1fr); gap: 14px; }
        .announce .entry-head { display: grid; grid-template-columns: minmax(0,1fr); }
        .announce .announcement-entry h3 { font-size: 18px; line-height: 1.18; }
        .announce .page-actions .button { min-height: 44px; }
        .announce .filter-form .button { min-height: 44px; }
        .announce .filter-tab { padding: 6px 14px; }
      }
    </style>
    <main id="main-content" tabindex="-1" class="app-main announce">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M4 5h16v13H7l-3 3z"/><path d="M8 9h8M8 13h6"/></svg><span>/</span><span>Aushang</span></span>
        {{if .CanManageAnnouncements}}<div class="page-actions"><button class="button primary" type="button" data-dialog="announcement-create" aria-haspopup="dialog" aria-controls="announcement-create"><svg viewBox="0 0 24 24" width="17" height="17" aria-hidden="true"><path d="M12 5v14M5 12h14" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round"/></svg>Aushang erstellen</button></div>{{end}}
      </div>
      <section class="page">
        <div>
          <h1>Aushang</h1>
          <p class="lede">Offizielle Informationen, Termine und Hinweise der Hausgemeinschaft.</p>
        </div>
        {{if .AnnounceMsg}}<p class="flash ok">{{.AnnounceMsg}}</p>{{end}}
        <div class="announce-layout">
          <div class="announce-main{{if not .HasAnyAnnouncements}} is-blank{{end}}">
          {{if .HasAnyAnnouncements}}
          <section class="panel announce-feed" aria-labelledby="announce-feed-title">
            <div class="section-head">
              <div class="section-copy">
                <div class="kicker">Archiv</div>
                <h2 id="announce-feed-title">Alle Aushänge</h2>
              </div>
              <div class="announce-counts">
                {{if .HasNewAnnouncements}}<span class="pill unread">{{.NewAnnouncements}} neu</span>{{end}}
                <span class="pill">{{len .Announcements}} Treffer</span>
              </div>
            </div>
            <div class="archive-tools">
              <form class="filter-form" method="get" action="/app/announcements">
                {{if .SelectedCategory}}<input type="hidden" name="category" value="{{.SelectedCategory}}">{{end}}
                <label for="announcement-search">Suche<input id="announcement-search" name="q" value="{{.SearchQuery}}" placeholder="Titel, Text oder Kategorie"></label>
                <button class="button" type="submit">Suchen</button>
              </form>
              <div class="filter-tabs" aria-label="Aushang-Kategorien">
                {{range .CategoryFilters}}<a class="filter-tab {{if .Active}}active{{end}}" href="{{.URL}}">{{.Label}}</a>{{end}}
              </div>
            </div>
            {{if .HasAnnouncements}}
              {{if .HasPinnedAnnouncements}}
              <section class="announce-group" aria-labelledby="announce-pinned-title">
                <div class="announce-group-head"><h3 id="announce-pinned-title">Oben fixiert</h3><span>{{len .PinnedAnnouncements}}</span></div>
                <div class="entries">
                  {{range .PinnedAnnouncements}}{{template "announcementEntry" .}}{{end}}
                </div>
              </section>
              {{end}}
              {{if .HasLatestAnnouncements}}
              <section class="announce-group" aria-labelledby="announce-latest-title">
                <div class="announce-group-head"><h3 id="announce-latest-title">{{if .HasPinnedAnnouncements}}Weitere Beiträge{{else}}Neueste zuerst{{end}}</h3><span>{{len .LatestAnnouncements}}</span></div>
                <div class="entries">
                  {{range .LatestAnnouncements}}{{template "announcementEntry" .}}{{end}}
                </div>
              </section>
              {{end}}
            {{else}}
              <div class="announce-filtered-empty">
                {{template "emptyState" .AnnouncementsEmpty}}
                <a class="button" href="/app/announcements">Filter zurücksetzen</a>
              </div>
            {{end}}
          </section>
          {{else}}
          <section class="announce-blank" aria-label="Aushang des Hauses">
            {{template "emptyState" .AnnouncementsBlank}}
            <div class="announce-blank-actions">
              {{if .CanManageAnnouncements}}
                <button class="button primary" type="button" data-dialog="announcement-create" aria-haspopup="dialog" aria-controls="announcement-create">Ersten Aushang erstellen</button>
              {{else}}
                <a class="button primary" href="/app/events">Termine ansehen</a>
                <a class="button ghost" href="{{if .CanManageIssues}}/app/anliegen/board{{else}}/app/anliegen{{end}}">Anliegen melden</a>
              {{end}}
            </div>
          </section>
          {{end}}
          </div>

          <aside class="announce-aside" aria-label="Hinweise zum Aushang">
            <details class="panel compact announce-aside-panel guide-disclosure" aria-labelledby="announce-legend-title">
              <summary><div>
                <div class="kicker">Kategorien</div>
                <h2 id="announce-legend-title">Was die Farben bedeuten</h2>
              </div></summary>
              <ul class="announce-legend">
                <li><span class="pill dringend">Dringend</span><span>Betrifft Sicherheit oder Versorgung und duldet keinen Aufschub.</span></li>
                <li><span class="pill wartung">Wartung</span><span>Arbeiten am Haus mit möglicher Einschränkung, etwa Lift oder Wasser.</span></li>
                <li><span class="pill termin">Termin</span><span>Zeitpunkt, den Sie sich vormerken sollten.</span></li>
                <li><span class="pill info">Info</span><span>Allgemeine Mitteilung ohne nötige Reaktion.</span></li>
              </ul>
            </details>
            <section class="panel compact announce-aside-panel" aria-labelledby="announce-next-title">
              <div>
                <div class="kicker">Auch hilfreich</div>
                <h2 id="announce-next-title">Weiter im Portal</h2>
              </div>
              <div class="announce-links">
                <a class="announce-link" href="/app/events"><span><strong>Termine</strong><small>Versammlungen, Wartungen und Fristen mit Datum.</small></span></a>
                <a class="announce-link" href="/app/dokumente"><span><strong>Dokumente</strong><small>Protokolle, Abrechnungen und Hausordnung.</small></span></a>
                <a class="announce-link" href="/app/settings/notifications"><span><strong>Benachrichtigungen</strong><small>Neue Aushänge zusätzlich per E-Mail erhalten.</small></span></a>
              </div>
            </section>
          </aside>
        </div>
      </section>

      {{if .CanManageAnnouncements}}
      <dialog id="announcement-create" class="dialog" aria-labelledby="announcement-create-title">
        <form method="post" action="/app/announcements" enctype="multipart/form-data">
          <div class="dialog-head">
            <h2 id="announcement-create-title">Aushang erstellen</h2>
            <button class="dialog-close" type="button" data-close-dialog aria-label="Schließen">&times;</button>
          </div>
          <div class="dialog-body">
            <div class="dialog-grid">
              <label class="full" for="announcement-title">Titel<input id="announcement-title" name="title" required maxlength="140"></label>
              <label class="full" for="announcement-body">Text<textarea id="announcement-body" name="body" required placeholder="Was soll die Hausgemeinschaft wissen?"></textarea></label>
              <label for="announcement-category">Kategorie<select id="announcement-category" name="category">
                <option value="Info">Info</option>
                <option value="Termin">Termin</option>
                <option value="Wartung">Wartung</option>
                <option value="Dringend">Dringend</option>
              </select></label>
              <details class="dialog-optional full">
                <summary>Planung, Fixierung oder Anhang</summary>
                <div class="dialog-optional-grid">
                  <label for="announcement-published">Veröffentlichen<input id="announcement-published" type="datetime-local" name="published_at" value="{{.NowInput}}"></label>
                  <label for="announcement-expires">Ablauf optional<input id="announcement-expires" type="datetime-local" name="expires_at"></label>
                  <label class="check-row full"><input type="checkbox" name="pinned" value="true"> oben fixieren</label>
                  <label class="full" for="announcement-attachments">Anhänge<span class="file-control"><input id="announcement-attachments" type="file" name="attachments" accept="image/jpeg,image/png,image/webp,image/gif,application/pdf" multiple><span>Fotos oder PDF auswählen</span></span></label>
                </div>
              </details>
            </div>
            <button class="button primary" type="submit">Aushang veröffentlichen</button>
          </div>
        </form>
      </dialog>
      {{end}}
    </main>
{{template "appClose" .}}
{{end}}

{{define "announcementEntry"}}
                  <article class="entry announcement-entry{{if .Unread}} is-unread{{end}}{{if .Pinned}} is-pinned{{end}}" id="announcement-{{.ID}}">
                    <div class="entry-head">
                      <div>
                        <h3>{{.Title}}</h3>
                        <div class="entry-meta">
                          <span class="pill {{.CategoryClass}}">{{.Category}}</span>
                          {{if .Unread}}<span class="pill unread">neu</span>{{end}}
                          {{if and .Status (ne .Status "Veröffentlicht") (ne .Status "Fixiert")}}<span class="pill">{{.Status}}</span>{{end}}
                          <span>{{.PublishedAt}}</span>
                          {{if .HasExpiresAt}}<span>bis {{.ExpiresAt}}</span>{{end}}
                        </div>
                      </div>
                      {{if .CanManage}}<div class="entry-actions">
                        <button class="button small" type="button" data-dialog="{{.EditDialogID}}" aria-haspopup="dialog" aria-controls="{{.EditDialogID}}">Bearbeiten</button>
                        <form method="post" action="/app/announcements/delete" data-confirm="{{.DeleteConfirmLabel}}">
                          <input type="hidden" name="id" value="{{.ID}}">
                          <button class="button small ghost" type="submit">Löschen</button>
                        </form>
                      </div>{{end}}
                    </div>
                    <details class="announcement-body"{{if .Unread}} open{{end}}>
                      <summary>{{if .Unread}}Neuen Aushang lesen{{else}}Aushang lesen{{end}}</summary>
                      <div class="announcement-body-content">
                        <div class="entry-body">{{.BodyHTML}}</div>
                        {{template "attachmentStrip" .}}
                      </div>
                    </details>
                  </article>
                  {{if .CanManage}}
                  <dialog id="{{.EditDialogID}}" class="dialog" aria-labelledby="{{.EditDialogID}}-title">
                    <form method="post" action="/app/announcements/edit" enctype="multipart/form-data">
                      <input type="hidden" name="id" value="{{.ID}}">
                      <div class="dialog-head">
                        <h2 id="{{.EditDialogID}}-title">Aushang bearbeiten</h2>
                        <button class="dialog-close" type="button" data-close-dialog aria-label="Schließen">&times;</button>
                      </div>
                      <div class="dialog-body">
                        <div class="dialog-grid">
                          <label class="full" for="title-{{.ID}}">Titel<input id="title-{{.ID}}" name="title" value="{{.Title}}" required maxlength="140"></label>
                          <label class="full" for="body-{{.ID}}">Text<textarea id="body-{{.ID}}" name="body" required>{{.Body}}</textarea></label>
                          <label for="category-{{.ID}}">Kategorie<select id="category-{{.ID}}" name="category">
                            <option value="Info"{{if eq .Category "Info"}} selected{{end}}>Info</option>
                            <option value="Termin"{{if eq .Category "Termin"}} selected{{end}}>Termin</option>
                            <option value="Wartung"{{if eq .Category "Wartung"}} selected{{end}}>Wartung</option>
                            <option value="Dringend"{{if eq .Category "Dringend"}} selected{{end}}>Dringend</option>
                          </select></label>
                          <label for="published-{{.ID}}">Veröffentlichen<input id="published-{{.ID}}" type="datetime-local" name="published_at" value="{{.PublishedAtInput}}"></label>
                          <label for="expires-{{.ID}}">Ablauf optional<input id="expires-{{.ID}}" type="datetime-local" name="expires_at" value="{{.ExpiresAtInput}}"></label>
                          <label class="check-row"><input type="checkbox" name="pinned" value="true"{{if .PinnedChecked}} checked{{end}}> oben fixieren</label>
                          <label class="full" for="attachments-{{.ID}}">Anhänge ergänzen<span class="file-control"><input id="attachments-{{.ID}}" type="file" name="attachments" accept="image/jpeg,image/png,image/webp,image/gif,application/pdf" multiple><span>Fotos oder PDF auswählen</span></span></label>
                        </div>
                        <button class="button primary" type="submit">Änderungen speichern</button>
                      </div>
                    </form>
                  </dialog>
                  {{end}}
{{end}}

{{define "eventEditDialog"}}
  <dialog id="{{.EditDialogID}}" class="dialog" aria-labelledby="{{.EditDialogID}}-title">
    <form method="post" action="/app/events/edit" enctype="multipart/form-data">
      <input type="hidden" name="id" value="{{.ID}}">
      <div class="dialog-head">
        <h2 id="{{.EditDialogID}}-title">Termin bearbeiten</h2>
        <button class="dialog-close" type="button" data-close-dialog aria-label="Schließen">&times;</button>
      </div>
      <div class="dialog-body">
        <div class="dialog-grid event-dialog-grid">
          <label class="full" for="event-title-{{.ID}}">Titel<input id="event-title-{{.ID}}" name="title" value="{{.Title}}" required maxlength="140"></label>
          <label for="event-category-{{.ID}}">Kategorie<select id="event-category-{{.ID}}" name="category">
            <option value="Eigentümerversammlung"{{if eq .Category "Eigentümerversammlung"}} selected{{end}}>Eigentümerversammlung</option>
            <option value="Reinigung"{{if eq .Category "Reinigung"}} selected{{end}}>Reinigung</option>
            <option value="Wartung"{{if eq .Category "Wartung"}} selected{{end}}>Wartung</option>
            <option value="Ablesung"{{if eq .Category "Ablesung"}} selected{{end}}>Ablesung</option>
            <option value="Frist"{{if eq .Category "Frist"}} selected{{end}}>Frist</option>
            <option value="Sonstiges"{{if eq .Category "Sonstiges"}} selected{{end}}>Sonstiges</option>
          </select></label>
          <label for="event-start-{{.ID}}">Beginn<input id="event-start-{{.ID}}" type="datetime-local" name="starts_at" value="{{.StartsAtInput}}" required></label>
          <label class="full" for="event-location-{{.ID}}">Ort<input id="event-location-{{.ID}}" name="location" value="{{.Location}}" maxlength="160" placeholder="Optional"></label>
          <details class="dialog-optional full">
            <summary>Ende, Details oder Anhang</summary>
            <div class="dialog-optional-grid">
              <label class="full" for="event-end-{{.ID}}">Ende optional<input id="event-end-{{.ID}}" type="datetime-local" name="ends_at" value="{{.EndsAtInput}}"></label>
              <label class="full" for="event-body-{{.ID}}">Details<textarea id="event-body-{{.ID}}" name="body">{{.Body}}</textarea></label>
              <label class="full" for="event-attachments-{{.ID}}">Anhänge ergänzen<span class="file-control"><input id="event-attachments-{{.ID}}" type="file" name="attachments" accept="image/jpeg,image/png,image/webp,image/gif,application/pdf" multiple><span>Fotos oder PDF auswählen</span></span></label>
            </div>
          </details>
        </div>
        <button class="button primary" type="submit">Änderungen speichern</button>
      </div>
    </form>
  </dialog>
{{end}}

{{define "eventCard"}}
  <article class="event-card{{if .IsNext}} is-next{{end}}{{if .Past}} past{{end}}" id="event-{{.ID}}">
    <span class="date-badge"><strong>{{.DateBadgeDay}}</strong><span>{{.DateBadgeMonth}}</span></span>
    <div class="event-info">
      <div class="event-card-head">
        <div>
          {{if .IsNext}}<span class="event-next-label">Als Nächstes</span>{{end}}
          <h3>{{.Title}}</h3>
        </div>
        {{if and .Status (ne .Status "Geplant")}}<span class="pill {{if .Past}}muted{{else}}ok{{end}}">{{.Status}}</span>{{end}}
      </div>
      <div class="event-meta">
        <span class="pill {{.CategoryClass}}">{{.Category}}</span>
        <strong>{{.DateLabel}}</strong>
        <span>{{.TimeRange}}</span>
        {{if .HasLocation}}<span>{{.Location}}</span>{{end}}
      </div>
      {{if or .HasBody .HasAttachments}}
        <details class="event-details">
          <summary>Details{{if .HasAttachments}} &amp; Anhänge{{end}}</summary>
          <div class="event-details-body">
            {{if .HasBody}}<div class="entry-body">{{.BodyHTML}}</div>{{end}}
            {{template "attachmentStrip" .}}
          </div>
        </details>
      {{end}}
      {{if .CanManage}}
        <div class="event-actions">
          <button class="button small" type="button" data-dialog="{{.EditDialogID}}" aria-haspopup="dialog" aria-controls="{{.EditDialogID}}">Bearbeiten</button>
          <form method="post" action="/app/events/delete" data-confirm="{{.DeleteConfirmLabel}}">
            <input type="hidden" name="id" value="{{.ID}}">
            <button class="button small" type="submit">Löschen</button>
          </form>
        </div>
      {{end}}
    </div>
  </article>
  {{if .CanManage}}{{template "eventEditDialog" .}}{{end}}
{{end}}

{{define "events"}}
{{template "appOpen" .}}
    <script src="/assets/announcements.js?v={{.AssetVersion}}" defer></script>
    <script src="/assets/attachments.js?v={{.AssetVersion}}" defer></script>
    <style>
      .events-page .events-layout { display: grid; gap: 20px; }
      .events-page .events-main, .events-page .events-aside { min-width: 0; display: grid; gap: 16px; align-content: start; }
      .events-page .events-main.is-blank { align-content: start; }
      .events-page .events-panel { display: grid; gap: 18px; }
      .events-page .events-panel .kicker { margin-bottom: 0; }
      .events-page .events-head-copy h2 { margin-top: 5px; }
      .events-page .events-head-copy p { margin-top: 5px; color: var(--muted); font-size: 13.5px; }
      .events-page .events-month { display: grid; gap: 11px; }
      .events-page .events-month + .events-month { margin-top: 4px; }
      .events-page .events-month-head { display: flex; align-items: baseline; justify-content: space-between; gap: 10px; border-bottom: 1px solid var(--line); padding-bottom: 7px; }
      .events-page .events-month-head h3 { color: var(--gold-ink); font-family: var(--font-sans); font-size: 11.5px; font-weight: 850; letter-spacing: .11em; text-transform: uppercase; }
      .events-page .events-month-head span { color: var(--soft); font-size: 12px; font-weight: 750; white-space: nowrap; }
      .events-page .events-blank { min-height: clamp(280px,34vh,360px); display: grid; align-content: center; justify-items: center; gap: 15px; border: 1px dashed rgba(200,153,63,.42); border-radius: var(--radius-sm); background: rgba(255,254,251,.68); padding: 32px 26px; text-align: center; }
      .events-page .events-blank .empty-state { width: 100%; border: 0; background: transparent; padding: 0; grid-template-columns: minmax(0,1fr); justify-items: center; gap: 13px; text-align: center; }
      .events-page .events-blank .empty-state p { max-width: 48ch; }
      .events-page .events-blank-actions { display: flex; flex-wrap: wrap; gap: 10px; justify-content: center; }
      .events-page .events-blank-actions .button { min-height: 44px; }
      .events-page .events-aside-panel { display: grid; gap: 14px; padding: 20px; align-content: start; }
      .events-page .events-aside-panel .kicker { margin-bottom: 0; }
      .events-page .events-aside-panel h2 { font-size: 20px; }
      .events-page .events-aside-panel > p { color: var(--muted); font-size: 13.5px; line-height: 1.5; }
      .events-page .events-aside-panel .button { min-height: 44px; }
      .events-page .events-legend { margin: 0; padding: 0; list-style: none; display: grid; gap: 13px; }
      .events-page .events-legend li { display: grid; gap: 5px; justify-items: start; }
      .events-page .events-legend span { color: var(--muted); font-size: 12.8px; line-height: 1.42; }
      .events-page .events-aside-note { border-top: 1px solid var(--line); padding-top: 12px; color: var(--soft); font-size: 12.5px; line-height: 1.45; overflow-wrap: anywhere; }
      .events-page .event-history { margin-top: 2px; }
      .events-page .event-details > summary, .events-page .event-history > summary { min-height: 44px; display: flex; align-items: center; }
      @media (min-width: 1181px) { .events-page .events-layout { grid-template-columns: minmax(0,1fr) 320px; } }
      @media (max-width: 1180px) and (min-width: 721px) { .events-page .events-aside { grid-template-columns: repeat(2,minmax(0,1fr)); align-items: start; } }
      @media (max-width: 720px) {
        .events-page .events-layout { gap: 12px; }
        .events-page .events-main, .events-page .events-aside { gap: 12px; }
        .events-page .events-panel { gap: 14px; }
        .events-page .events-blank { min-height: 0; padding: 26px 18px; }
        .events-page .events-blank-actions { width: 100%; }
        .events-page .events-blank-actions .button { width: 100%; }
        .events-page .events-aside-panel { padding: 16px; }
        .events-page .events-aside-panel .button { width: 100%; }
        .events-shell .page-actions .button { min-height: 44px; }
        .events-page .event-actions .button { min-height: 42px; }
        .events-page .event-history > summary { min-height: 46px; }
      }
    </style>
    <main id="main-content" tabindex="-1" class="app-main events-shell">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M7 3v4M17 3v4"/><path d="M4.5 6h15v14h-15z"/><path d="M4.5 10h15"/><path d="M8 14h.01M12 14h.01M16 14h.01"/></svg><span>/</span><span>Termine</span></span>
        {{if .CanManageEvents}}<div class="page-actions"><button class="button primary" type="button" data-dialog="event-create" aria-haspopup="dialog" aria-controls="event-create"><svg viewBox="0 0 24 24" width="17" height="17" aria-hidden="true"><path d="M12 5v14M5 12h14" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round"/></svg>Termin erstellen</button></div>{{end}}
      </div>
      <section class="page events-page">
        <div>
          <h1>Termine</h1>
          <p class="lede">Was als Nächstes im Haus ansteht – mit Zeitpunkt, Ort und allen nötigen Details.</p>
        </div>
        {{if .EventMsg}}<p class="flash {{if .EventOK}}ok{{end}}">{{.EventMsg}}</p>{{end}}
        <div class="events-layout">
          <div class="events-main{{if and (not .HasEvents) (not .HasPastEvents)}} is-blank{{end}}">
            {{if .HasEvents}}
            <section class="panel events-panel" aria-labelledby="events-upcoming-title">
              <div class="section-head">
                <div class="events-head-copy">
                  <div class="kicker">Kommende Termine</div>
                  <h2 id="events-upcoming-title">Was als Nächstes ansteht</h2>
                  <p>Nach Monat geordnet, der nächste Termin steht zuerst.</p>
                </div>
                <span class="pill">{{len .Events}} geplant</span>
              </div>
              {{range .EventMonths}}
                <section class="events-month">
                  <div class="events-month-head"><h3>{{.Label}}</h3><span>{{.Count}} {{if eq .Count 1}}Termin{{else}}Termine{{end}}</span></div>
                  <div class="agenda-list">
                    {{range .Events}}{{template "eventCard" .}}{{end}}
                  </div>
                </section>
              {{end}}
              {{if .HasPastEvents}}
                <details class="event-history">
                  <summary>Vergangene Termine · {{len .PastEvents}}</summary>
                  <div class="agenda-list">
                    {{range .PastEvents}}{{template "eventCard" .}}{{end}}
                  </div>
                </details>
              {{end}}
            </section>
            {{else}}
            <section class="events-blank" aria-label="Termine des Hauses">
              {{template "emptyState" .EventsEmpty}}
              <div class="events-blank-actions">
                {{if .CanManageEvents}}
                  <button class="button primary" type="button" data-dialog="event-create" aria-haspopup="dialog" aria-controls="event-create">Ersten Termin erstellen</button>
                {{else}}
                  <a class="button primary" href="/app/announcements">Aushang ansehen</a>
                  <a class="button ghost" href="{{if .CanManageIssues}}/app/anliegen/board{{else}}/app/anliegen{{end}}">Anliegen melden</a>
                {{end}}
              </div>
            </section>
            {{if .HasPastEvents}}
              <details class="event-history">
                <summary>Vergangene Termine · {{len .PastEvents}}</summary>
                <div class="agenda-list">
                  {{range .PastEvents}}{{template "eventCard" .}}{{end}}
                </div>
              </details>
            {{end}}
            {{end}}
          </div>

          <aside class="events-aside" aria-label="Hinweise zu Terminen">
            {{if .HasCalendarFeedURL}}
            <section class="panel compact events-aside-panel" aria-labelledby="events-feed-title">
              <div>
                <div class="kicker">Eigener Kalender</div>
                <h2 id="events-feed-title">Einmal abonnieren</h2>
              </div>
              <p>Neue und geänderte Haustermine erscheinen danach automatisch in Ihrem Kalender – ohne weiteres Zutun.</p>
              <a class="button primary" href="{{.CalendarFeedURL}}">Kalender abonnieren</a>
              <p class="events-aside-note">Der Link ist persönlich. Bitte nicht weitergeben.</p>
            </section>
            {{end}}
            <details class="panel compact events-aside-panel guide-disclosure" aria-labelledby="events-legend-title">
              <summary><div>
                <div class="kicker">Was hier erscheint</div>
                <h2 id="events-legend-title">Termine im Haus</h2>
              </div></summary>
              <ul class="events-legend">
                <li><span class="pill versammlung">Eigentümerversammlung</span><span>Beschlüsse der Gemeinschaft. Teilnahme oder Vollmacht einplanen.</span></li>
                <li><span class="pill wartung">Wartung</span><span>Lift, Heizung oder Technik. Zugang kann kurz eingeschränkt sein.</span></li>
                <li><span class="pill ablesung">Ablesung</span><span>Zählerstände. Meist ist Zutritt zur Wohnung nötig.</span></li>
                <li><span class="pill frist">Frist</span><span>Letzter Tag für eine Rückmeldung oder Zahlung.</span></li>
                <li><span class="pill reinigung">Reinigung</span><span>Wiederkehrende Arbeiten in Haus, Hof und Grünflächen.</span></li>
              </ul>
            </details>
          </aside>
        </div>
      </section>

      {{if .CanManageEvents}}
      <dialog id="event-create" class="dialog" aria-labelledby="event-create-title">
        <form method="post" action="/app/events" enctype="multipart/form-data">
          <div class="dialog-head">
            <h2 id="event-create-title">Termin erstellen</h2>
            <button class="dialog-close" type="button" data-close-dialog aria-label="Schließen">&times;</button>
          </div>
          <div class="dialog-body">
            <div class="dialog-grid event-dialog-grid">
              <label class="full" for="event-title">Titel<input id="event-title" name="title" required maxlength="140" placeholder="Was findet statt?"></label>
              <label for="event-category">Kategorie<select id="event-category" name="category">
                <option value="Eigentümerversammlung">Eigentümerversammlung</option>
                <option value="Reinigung">Reinigung</option>
                <option value="Wartung">Wartung</option>
                <option value="Ablesung">Ablesung</option>
                <option value="Frist">Frist</option>
                <option value="Sonstiges">Sonstiges</option>
              </select></label>
              <label for="event-start">Beginn<input id="event-start" type="datetime-local" name="starts_at" value="{{.NowInput}}" required></label>
              <label class="full" for="event-location">Ort<input id="event-location" name="location" maxlength="160" placeholder="Optional"></label>
              <details class="dialog-optional full">
                <summary>Ende, Details oder Anhang</summary>
                <div class="dialog-optional-grid">
                  <label class="full" for="event-end">Ende optional<input id="event-end" type="datetime-local" name="ends_at"></label>
                  <label class="full" for="event-body">Details<textarea id="event-body" name="body" placeholder="Was müssen Bewohner wissen?"></textarea></label>
                  <label class="full" for="event-attachments">Anhänge<span class="file-control"><input id="event-attachments" type="file" name="attachments" accept="image/jpeg,image/png,image/webp,image/gif,application/pdf" multiple><span>Fotos oder PDF auswählen</span></span></label>
                </div>
              </details>
            </div>
            <button class="button primary" type="submit">Termin veröffentlichen</button>
          </div>
        </form>
      </dialog>
      {{end}}
    </main>
{{template "appClose" .}}
{{end}}

{{define "documentUploadTrigger"}}<button class="button primary" type="button" data-dialog="document-upload" aria-haspopup="dialog" aria-controls="document-upload"><svg viewBox="0 0 24 24" width="17" height="17" aria-hidden="true"><path d="M12 5v14M5 12h14" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round"/></svg>Dokument hochladen</button>{{end}}

{{define "documents"}}
{{template "appOpen" .}}
    <script src="/assets/announcements.js?v={{.AssetVersion}}" defer></script>
    <script src="/assets/attachments.js?v={{.AssetVersion}}" defer></script>
    <style>
      /* Dokumente: der leere Zustand ist der Normalfall eines neuen Hauses und
         steht deshalb als eigener Bereich – ohne Karte in der Karte. */
      .documents-screen .documents-page { gap: 20px; }
      .documents-screen .document-library-head { align-items: flex-end; border-bottom: 1px solid var(--line); padding-bottom: 12px; }
      .documents-screen .document-library-head .kicker { border-bottom: 0; padding-bottom: 0; margin-bottom: 0; }
      .documents-screen .document-library-head .mini { margin-top: 5px; color: var(--muted); font-size: 12.5px; }
      .documents-screen .document-library { gap: 18px; }
      .documents-screen .document-sections { gap: 18px; }
      .documents-screen .document-section { gap: 8px; }
      .documents-screen .document-section h3 { font-size: 18px; }
      .documents-screen .document-list { gap: 8px; }
      .documents-screen .document-row { padding: 13px 15px; gap: 10px 14px; }
      .documents-screen .document-copy > strong { font-size: 17.5px; }
      .documents-screen .document-file-details { margin-top: 4px; }
      .doc-blank { display: grid; grid-template-columns: minmax(0,1.42fr) minmax(272px,.88fr); gap: 16px; }
      .doc-blank-main { display: grid; align-content: center; gap: 20px; border: 1px dashed rgba(200,153,63,.45); border-radius: var(--radius-sm); background: rgba(255,254,251,.7); padding: clamp(22px,3.2vw,36px); }
      .doc-blank-lead { display: grid; justify-items: start; gap: 13px; }
      .doc-blank-icon { width: 52px; height: 52px; display: grid; place-items: center; border: 1px solid rgba(200,153,63,.3); border-radius: 12px; background: rgba(200,153,63,.1); color: var(--gold-ink); }
      .doc-blank-icon svg { width: 26px; height: 26px; fill: none; stroke: currentColor; stroke-width: 1.5; stroke-linecap: round; stroke-linejoin: round; }
      .doc-blank-main h2 { font-size: clamp(25px,3vw,31px); }
      .doc-blank-main p { max-width: 54ch; color: var(--muted); font-size: 15px; line-height: 1.55; }
      .doc-blank-actions { display: flex; flex-wrap: wrap; gap: 9px; margin-top: 3px; }
      .doc-blank-actions .button { min-height: 44px; }
      .doc-blank-side { display: grid; align-content: start; gap: 11px; border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel); padding: 20px; box-shadow: var(--shadow-panel); }
      .doc-blank-side h2 { font-size: 12px; font-weight: 800; letter-spacing: .12em; text-transform: uppercase; color: var(--gold-ink); font-family: var(--font-sans); }
      .doc-blank-list { display: grid; margin: 0; padding: 0; list-style: none; }
      .doc-blank-list li { display: grid; gap: 2px; border-top: 1px solid var(--line); padding: 9px 0; }
      .doc-blank-list li:first-child { border-top: 0; padding-top: 0; }
      .doc-blank-list li:last-child { padding-bottom: 0; }
      .doc-blank-list strong { font-size: 13.5px; }
      .doc-blank-list span { color: var(--muted); font-size: 12.5px; line-height: 1.4; }
      .doc-blank-facts { grid-column: 1 / -1; display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 14px; margin: 0; padding: 0; list-style: none; }
      .doc-blank-facts li { display: grid; gap: 5px; border-top: 2px solid var(--ink); padding-top: 11px; }
      .doc-blank-facts strong { font-size: 13.5px; }
      .doc-blank-facts span { color: var(--muted); font-size: 12.5px; line-height: 1.5; }
      @media (min-width: 901px) {
        .doc-blank { min-height: max(420px, calc(100vh - 348px)); grid-template-rows: minmax(0,1fr) auto; }
      }
      @media (max-width: 900px) {
        .doc-blank { grid-template-columns: minmax(0,1fr); gap: 14px; }
        .doc-blank-main { padding: 22px 18px; }
        .doc-blank-facts { grid-template-columns: minmax(0,1fr); gap: 12px; }
      }
      @media (max-width: 560px) {
        .doc-blank-actions { display: grid; }
        .doc-blank-actions .button { width: 100%; justify-content: center; }
      }
    </style>
    <main id="main-content" tabindex="-1" class="app-main documents-screen">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M7 3h7l3 3v15H7z"/><path d="M14 3v4h4"/><path d="M9 13h6M9 17h6"/></svg><span>/</span><span>Dokumente</span></span>
        {{if and .CanManageDocuments (or .HasAnyDocuments .HasSearchQuery)}}<div class="page-actions"><a class="button" href="/app/dokumente/rechnungen/import">E-Rechnung einlesen</a>{{template "documentUploadTrigger" .}}</div>{{end}}
      </div>
      <section class="page documents-page">
        <div class="document-page-head">
          <div>
            <h1>Dokumente</h1>
            <p class="lede">Die freigegebenen Unterlagen des Hauses – schnell finden, ansehen und herunterladen.</p>
          </div>
        </div>
        {{if .DocumentMsg}}<p class="flash {{if .DocumentOK}}ok{{end}}">{{.DocumentMsg}}</p>{{end}}
        {{if or .HasAnyDocuments .HasSearchQuery}}
        <section class="panel document-library">
          <div class="document-library-head">
            <div>
              <div class="kicker">Hausablage</div>
              {{if .HasSearchQuery}}<p class="mini">Ergebnis für „{{.SearchQuery}}“</p>{{end}}
            </div>
            {{if .HasAnyDocuments}}<span class="pill">{{.DocumentCountLabel}}</span>{{end}}
          </div>
          <form class="document-toolbar" method="get" action="/app/dokumente" role="search">
              <label class="document-search" for="document-search">
                <span class="sr-only">Dokumente durchsuchen</span>
                <svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="11" cy="11" r="7"/><path d="m16.5 16.5 4 4"/></svg>
                <input id="document-search" name="q" value="{{.SearchQuery}}" placeholder="Titel, Kategorie oder Datei">
              </label>
              <label class="document-sort" for="document-sort">
                <span class="sr-only">Sortierung</span>
                <select id="document-sort" name="sort">
                {{range .SortOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}
                </select>
              </label>
              <button class="button" type="submit">Anzeigen</button>
              {{if .HasSearchQuery}}<a class="document-reset" href="/app/dokumente">Zurücksetzen</a>{{end}}
          </form>
          {{if .HasDocuments}}
            <div class="document-sections">
              {{range .DocumentSections}}
                <section class="document-section">
                  <h3>{{.Category}}<span class="section-count">{{len .Documents}}</span></h3>
                    <div class="document-list">
                      {{range .Documents}}
                        <article class="document-row" id="document-{{.ID}}">
                          <span class="document-icon" aria-hidden="true"><svg viewBox="0 0 24 24"><path d="M7 3h7l3 3v15H7z"/><path d="M14 3v4h4"/><path d="M9 12h6M9 16h6"/></svg></span>
                          <div class="document-copy">
                            <strong>{{.Title}}</strong>
                            <div class="document-meta">
                              <span class="pill {{.VisibilityClass}}">{{.Visibility}}</span>
                              {{if .HasUnit}}<span class="pill">Einheit {{.UnitLabel}}</span>{{end}}
                              <span>{{.VersionLabel}}</span>
                              <span class="document-meta-secondary">{{.UploadedDate}} · {{.Size}}</span>
                            </div>
                            <details class="document-file-details">
                              <summary>Dateidetails</summary>
                              <span class="document-file">{{.Filename}} · hochgeladen {{.UploadedAt}}</span>
                            </details>
                          </div>
                          <div class="document-actions">
                            {{if .CanPreview}}
                              {{if .IsImage}}<button class="button primary" type="button" data-lightbox-src="{{.PreviewURL}}" data-lightbox-caption="{{.Filename}}">Vorschau</button>{{else}}<a class="button primary" href="{{.PreviewURL}}" target="_blank" rel="noopener">Vorschau</a>{{end}}
                              <a class="button small" href="{{.DownloadURL}}">Herunterladen</a>
                            {{else}}
                              <a class="button primary" href="{{.DownloadURL}}">Herunterladen</a>
                            {{end}}
                          </div>
                          {{if $.CanManageDocuments}}
                            <details class="document-admin-tools">
                              <summary>Dokument verwalten</summary>
                              <div class="document-admin-tools-body">
                                <button class="button small" type="button" data-dialog="{{.ReplaceDialogID}}" aria-haspopup="dialog" aria-controls="{{.ReplaceDialogID}}">Neue Version hochladen</button>
                                <span class="mini">Kategorie, Sichtbarkeit und Einheit bleiben dabei erhalten.</span>
                              </div>
                            </details>
                          {{end}}
                          {{if .HasVersions}}
                            <details class="document-versions">
                              <summary>Versionsverlauf</summary>
                              <div class="version-list">
                                {{range .Versions}}
                                  <div class="version-row">
                                    <span>{{.Version}} · {{.UploadedAt}} · {{.Size}} · {{.Filename}}</span>
                                    <a class="button small" href="{{.DownloadURL}}">Herunterladen</a>
                                  </div>
                                {{end}}
                              </div>
                            </details>
                          {{end}}
                        </article>
                        {{if $.CanManageDocuments}}
                        <dialog id="{{.ReplaceDialogID}}" class="dialog" aria-labelledby="{{.ReplaceDialogID}}-title">
                          <form method="post" action="/app/dokumente/replace" enctype="multipart/form-data">
                            <input type="hidden" name="id" value="{{.ID}}">
                            <div class="dialog-head">
                              <h2 id="{{.ReplaceDialogID}}-title">Neue Version speichern</h2>
                              <button class="dialog-close" type="button" data-close-dialog aria-label="Schließen">&times;</button>
                            </div>
                            <div class="dialog-body">
                              <div class="document-replace-context">
                                <strong>{{.Title}}</strong>
                                <span class="mini">Aktuell {{.VersionLabel}} · {{.Visibility}}{{if .HasUnit}} · Einheit {{.UnitLabel}}{{end}}</span>
                              </div>
                              <div class="dialog-grid">
                                <label class="full" for="{{.ReplaceDialogID}}-file">Neue Datei<span class="file-control"><input id="{{.ReplaceDialogID}}-file" type="file" name="document" accept="application/pdf,image/jpeg,image/png,image/webp" required><span>PDF oder Bild auswählen</span></span></label>
                              </div>
                              <button class="button primary" type="submit">Neue Version speichern</button>
                            </div>
                          </form>
                        </dialog>
                        {{end}}
                      {{end}}
                    </div>
                </section>
              {{end}}
            </div>
          {{else}}
            <div class="document-empty">
              <div>
                <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M7 3h7l3 3v15H7z"/><path d="M14 3v4h4"/><circle cx="10" cy="14" r="3.5"/><path d="m12.5 16.5 3 3"/></svg>
                <h3>Keine Dokumente gefunden</h3>
                <p>Versuchen Sie einen anderen Suchbegriff oder zeigen Sie wieder alle Dokumente.</p>
                <a class="button" href="/app/dokumente">Alle Dokumente zeigen</a>
              </div>
            </div>
          {{end}}
        </section>
        {{else}}
        <section class="doc-blank" aria-labelledby="doc-blank-title">
          <div class="doc-blank-main">
            <div class="doc-blank-lead">
              <span class="doc-blank-icon" aria-hidden="true"><svg viewBox="0 0 24 24"><path d="M7 3h7l3 3v15H7z"/><path d="M14 3v4h4"/><circle cx="10" cy="14" r="3.5"/><path d="m12.5 16.5 3 3"/></svg></span>
              <h2 id="doc-blank-title">{{.DocumentsEmpty.Title}}</h2>
              {{if .CanManageDocuments}}<p>Legen Sie die erste Unterlage ab. Titel, Kategorie und Sichtbarkeit genügen – danach ist die Datei für die gewählte Gruppe im Portal auffindbar.</p>{{else}}<p>{{.DocumentsEmpty.Message}}</p>{{end}}
            </div>
            <div class="doc-blank-actions">
              {{if .CanManageDocuments}}
                {{template "documentUploadTrigger" .}}
                <a class="button" href="/app/dokumente/rechnungen/import">E-Rechnung einlesen</a>
              {{else}}
                <a class="button primary" href="/app/anliegen?new=1">Unterlage anfragen</a>
                <a class="button ghost" href="/app/kontakte">Verwaltung im Kontaktverzeichnis</a>
              {{end}}
            </div>
          </div>
          <aside class="doc-blank-side">
            <h2>Was hier abgelegt wird</h2>
            <ul class="doc-blank-list">
              {{range .DocumentGuide}}<li><strong>{{.Category}}</strong><span>{{.Detail}}</span></li>{{end}}
            </ul>
          </aside>
          <ul class="doc-blank-facts">
            <li><strong>Sichtbarkeit</strong><span>Jede Unterlage ist für alle Bewohner, nur für Eigentümer oder nur für eine Einheit freigegeben. Sie sehen ausschließlich Ihren Teil der Ablage.</span></li>
            <li><strong>Versionen</strong><span>Wird eine Unterlage ersetzt, bleibt die frühere Fassung im Versionsverlauf abrufbar.</span></li>
            <li><strong>Suche</strong><span>Ab dem ersten Dokument stehen Suche nach Titel, Kategorie und Datei sowie die Sortierung bereit.</span></li>
          </ul>
        </section>
        {{end}}
      </section>

      {{with $}}{{if .CanManageDocuments}}
      <dialog id="document-upload" class="dialog" aria-labelledby="document-upload-title">
        <form method="post" action="/app/dokumente" enctype="multipart/form-data">
          <div class="dialog-head">
            <h2 id="document-upload-title">Dokument veröffentlichen</h2>
            <button class="dialog-close" type="button" data-close-dialog aria-label="Schließen">&times;</button>
          </div>
          <div class="dialog-body">
            <p class="document-dialog-intro">Titel, Einordnung und Datei genügen. Die Sichtbarkeit bestimmt, wer das Dokument im Portal findet.</p>
            <div class="dialog-grid">
              <label class="full" for="document-title">Titel<input id="document-title" name="title" required maxlength="160" autocomplete="off" placeholder="Zum Beispiel Hausordnung 2026"></label>
              <label for="document-category">Kategorie<select id="document-category" name="category" required>
                {{range .CategoryOptions}}<option value="{{.Value}}"{{if not .Value}} disabled{{end}}{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}
              </select></label>
              <label for="document-visibility">Sichtbarkeit<select id="document-visibility" name="visibility" required>
                {{range .VisibilityOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}
              </select></label>
              <label class="full" for="document-file">Datei<span class="file-control"><input id="document-file" type="file" name="document" accept="application/pdf,image/jpeg,image/png,image/webp" required><span>PDF oder Bild auswählen</span></span></label>
              <details class="dialog-optional full">
                <summary>Auf eine Einheit begrenzen</summary>
                <div class="dialog-optional-grid">
                  <label class="full" for="document-unit">Einheit<select id="document-unit" name="unit_id">
                    {{range .UnitOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}
                  </select></label>
                </div>
              </details>
            </div>
            <button class="button primary" type="submit">Dokument veröffentlichen</button>
          </div>
        </form>
      </dialog>
      {{end}}{{end}}
    </main>
{{template "appClose" .}}
{{end}}

{{define "ballotProtocol"}}
<!doctype html>
<html lang="de">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  <style>
    * { box-sizing: border-box; }
    body { margin: 0; background: #f7f3ea; color: #20251f; font-family: Georgia, "Times New Roman", serif; }
    main { width: min(920px,100%); margin: 0 auto; padding: 42px 28px; }
    h1, h2, h3 { font-family: Georgia, "Times New Roman", serif; margin: 0; }
    h1 { font-size: 42px; line-height: 1; }
    h2 { font-size: 24px; margin-top: 28px; }
    h3 { font-size: 18px; }
    p { margin: 0; line-height: 1.5; }
    .protocol-head { display: grid; gap: 10px; border-bottom: 2px solid #20251f; padding-bottom: 22px; }
    .meta { display: flex; flex-wrap: wrap; gap: 8px; color: #6b6f63; font-size: 13px; font-weight: 700; }
    .pill { display: inline-flex; align-items: center; min-height: 26px; border-radius: 999px; padding: 3px 10px; font-size: 12px; font-weight: 800; background: rgba(200,153,63,.16); color: #8a6a1f; }
    .pill.ok { background: rgba(47,107,74,.12); color: #2f6b4a; }
    .summary { margin-top: 24px; display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 12px; }
    .box { border: 1px solid #e7e0d2; border-radius: 8px; background: #fffefb; padding: 14px; }
    .box span { display: block; color: #8a7b3f; font-size: 11px; font-weight: 800; text-transform: uppercase; letter-spacing: .06em; }
    .box strong { display: block; margin-top: 6px; font-family: Georgia, "Times New Roman", serif; font-size: 22px; }
    table { width: 100%; margin-top: 14px; border-collapse: collapse; background: #fffefb; border: 1px solid #e7e0d2; }
    th, td { padding: 12px 14px; border-bottom: 1px solid #e7e0d2; text-align: left; }
    th { color: #8a7b3f; font-size: 11px; text-transform: uppercase; letter-spacing: .06em; }
    tr:last-child td { border-bottom: 0; }
    .num { text-align: right; font-variant-numeric: tabular-nums; }
    footer { margin-top: 28px; color: #6b6f63; font-size: 12px; }
    @media print {
      body { background: #fff; }
      main { padding: 24px 0; }
      .box, table { break-inside: avoid; }
    }
    @media (max-width: 680px) {
      main { padding: 28px 18px; }
      h1 { font-size: 34px; }
      .summary { grid-template-columns: 1fr; }
      table { font-size: 13px; }
    }
  </style>
</head>
<body>
  <main>
    <section class="protocol-head">
      <div class="meta"><span>{{.Tenant.Name}}</span><span>{{.Tenant.Address}}</span><span>Erstellt {{.GeneratedAt}}</span></div>
      <h1>Abstimmungsprotokoll</h1>
      <h2>{{.Ballot.Title}}</h2>
      <div class="meta">
        <span class="pill {{.Ballot.StatusClass}}">{{.Ballot.Status}}</span>
        <span>{{.Ballot.Type}}</span>
        <span>{{.Ballot.Weighting}}</span>
        {{if .Ballot.HasQuorum}}<span>Quorum {{.Ballot.Quorum}}</span>{{end}}
        {{if .Ballot.HasClosesAt}}<span>Frist {{.Ballot.ClosesAt}}</span>{{end}}
      </div>
      {{if .Ballot.HasDescription}}<p>{{.Ballot.Description}}</p>{{end}}
    </section>

    <section class="summary" aria-label="Zusammenfassung">
      <div class="box"><span>Teilnahme</span><strong>{{.Ballot.Participation}}</strong></div>
      <div class="box"><span>Stimmgewicht</span><strong>{{.Ballot.TotalWeightLabel}}</strong></div>
      <div class="box"><span>Quorum</span><strong>{{.Ballot.QuorumStatus}}</strong></div>
      <div class="box"><span>Stimmberechtigt</span><strong>{{.Ballot.EligibleWeightLabel}}</strong></div>
      <div class="box"><span>Stimmen</span><strong>{{.Ballot.TotalVotes}}</strong></div>
      <div class="box"><span>Ergebnis</span><strong>{{if .Ballot.HasWinner}}{{.Ballot.WinnerLabel}}{{else}}Keine Stimmen{{end}}</strong></div>
    </section>

    <section>
      <h2>Auszählung</h2>
      <table aria-label="Auszählung">
        <thead><tr><th>Option</th><th class="num">Gewicht</th><th class="num">Stimmen</th><th class="num">Anteil</th></tr></thead>
        <tbody>
          {{range .Ballot.Options}}
            <tr><td><strong>{{.Label}}</strong></td><td class="num">{{.WeightLabel}}</td><td class="num">{{.VoteCount}}</td><td class="num">{{.Percent}} %</td></tr>
          {{end}}
        </tbody>
      </table>
    </section>
    <footer>{{.AppVersion}}</footer>
  </main>
</body>
</html>
{{end}}

{{define "ballotResult"}}
  <section class="vote-result" aria-label="{{if .IsClosed}}Abstimmungsergebnis{{else}}Zwischenstand{{end}}">
    <div class="vote-result-head">
      <div>
        <div class="kicker">{{if .IsClosed}}Ergebnis{{else}}Zwischenstand{{end}}</div>
        <strong>{{if .HasWinner}}{{.WinnerLabel}}{{else}}Noch keine Stimmen{{end}}</strong>
      </div>
      <div class="vote-result-actions">
        {{if .HasProtocol}}<a class="button small" href="{{.ProtocolURL}}">Protokoll herunterladen</a>{{end}}
      </div>
    </div>
    <div class="vote-result-stats">
      <div class="vote-result-stat"><span>Teilnahme</span><strong>{{.Participation}}</strong></div>
      <div class="vote-result-stat"><span>Stimmen</span><strong>{{.TotalVotes}}</strong></div>
      <div class="vote-result-stat"><span>Quorum</span><strong>{{.QuorumStatus}}</strong></div>
    </div>
    <div class="vote-result-rows">
      {{range .Options}}
        <div class="vote-result-row">
          <strong>{{.Label}}</strong>
          <span class="vote-bar"><span style="width: {{.PercentStyle}}%;"></span></span>
          <span>{{.WeightLabel}} · {{.VoteCount}} Stimmen</span>
        </div>
      {{end}}
    </div>
  </section>
{{end}}

{{define "ballotCreateTrigger"}}<button class="button primary" type="button" data-dialog="ballot-create" aria-haspopup="dialog" aria-controls="ballot-create"><svg viewBox="0 0 24 24" width="17" height="17" aria-hidden="true"><path d="M12 5v14M5 12h14" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round"/></svg>Abstimmung anlegen</button>{{end}}

{{define "ballots"}}
{{template "appOpen" .}}
    <script src="/assets/announcements.js?v={{.AssetVersion}}" defer></script>
    <script src="/assets/attachments.js?v={{.AssetVersion}}" defer></script>
    <style>
      /* Abstimmungen: ohne laufende Abstimmung erklärt die Seite den Ablauf,
         statt eine leere Fläche unter einem Hinweisstreifen zu lassen. */
      .vote-page { gap: 20px; }
      .vote-blank { display: grid; grid-template-columns: minmax(0,1.42fr) minmax(272px,.88fr); gap: 16px; }
      .vote-blank-main { display: grid; align-content: center; gap: 20px; border: 1px dashed rgba(200,153,63,.45); border-radius: var(--radius-sm); background: rgba(255,254,251,.7); padding: clamp(22px,3.2vw,36px); }
      .vote-blank-lead { display: grid; justify-items: start; gap: 13px; }
      .vote-blank.ok .vote-blank-main { border-color: rgba(47,107,74,.3); }
      .vote-blank-icon { width: 52px; height: 52px; display: grid; place-items: center; border: 1px solid rgba(200,153,63,.3); border-radius: 12px; background: rgba(200,153,63,.1); color: var(--gold-ink); }
      .vote-blank.ok .vote-blank-icon { border-color: rgba(47,107,74,.24); background: rgba(47,107,74,.1); color: var(--leaf); }
      .vote-blank-icon svg { width: 26px; height: 26px; fill: none; stroke: currentColor; stroke-width: 1.6; stroke-linecap: round; stroke-linejoin: round; }
      .vote-blank-main h2 { font-size: clamp(25px,3vw,31px); }
      .vote-blank-main p { max-width: 54ch; color: var(--muted); font-size: 15px; line-height: 1.55; }
      .vote-blank-note { border-left: 3px solid var(--line); padding-left: 12px; font-size: 13.5px; }
      .vote-blank-actions { display: flex; flex-wrap: wrap; gap: 9px; margin-top: 3px; }
      .vote-blank-actions .button { min-height: 44px; }
      .vote-blank-side { display: grid; align-content: start; gap: 12px; border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel); padding: 20px; box-shadow: var(--shadow-panel); }
      .vote-blank-side h2 { font-size: 12px; font-weight: 800; letter-spacing: .12em; text-transform: uppercase; color: var(--gold-ink); font-family: var(--font-sans); }
      .vote-blank-steps { display: grid; gap: 12px; margin: 0; padding: 0; list-style: none; counter-reset: vote-step; }
      .vote-blank-steps li { display: grid; grid-template-columns: 26px minmax(0,1fr); gap: 2px 11px; align-items: start; counter-increment: vote-step; }
      .vote-blank-steps li::before { content: counter(vote-step); grid-row: 1 / span 2; width: 26px; height: 26px; display: grid; place-items: center; border-radius: 50%; background: var(--ink); color: #fff; font-size: 12.5px; font-weight: 850; }
      .vote-blank-steps strong { grid-column: 2; font-size: 13.5px; }
      .vote-blank-steps span { grid-column: 2; color: var(--muted); font-size: 12.5px; line-height: 1.4; }
      .vote-blank-facts { grid-column: 1 / -1; display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 14px; margin: 0; padding: 0; list-style: none; }
      .vote-blank-facts li { display: grid; gap: 5px; border-top: 2px solid var(--ink); padding-top: 11px; }
      .vote-blank-facts strong { font-size: 13.5px; }
      .vote-blank-facts span { color: var(--muted); font-size: 12.5px; line-height: 1.5; }
      .vote-page .vote-details > summary, .vote-page .vote-live-result > summary { min-height: 44px; display: flex; align-items: center; }
      .vote-readonly-note { margin: 0; border-left: 3px solid var(--line); padding: 10px 12px; color: var(--muted); background: var(--panel-soft); font-size: 13.5px; line-height: 1.45; }
      @media (min-width: 901px) {
        .vote-blank { min-height: max(420px, calc(100vh - 348px)); grid-template-rows: minmax(0,1fr) auto; }
      }
      @media (max-width: 900px) {
        .vote-page .vote-page-head .lede { display: block; margin-top: 10px; font-size: 14.5px; }
        .vote-blank { grid-template-columns: minmax(0,1fr); gap: 14px; }
        .vote-blank-main { padding: 22px 18px; }
        .vote-blank-facts { grid-template-columns: minmax(0,1fr); gap: 12px; }
      }
      @media (max-width: 560px) {
        .vote-blank-actions { display: grid; }
        .vote-blank-actions .button { width: 100%; justify-content: center; }
      }
    </style>
    <main id="main-content" tabindex="-1" class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M5 19V9M12 19V5M19 19v-7"/><path d="M3.5 19h17"/></svg><span>/</span><span>Abstimmungen</span></span>
        {{if and .CanManageVotes .HasBallots}}<div class="page-actions">{{template "ballotCreateTrigger" .}}</div>{{end}}
      </div>
      <section class="page vote-page">
        <div class="vote-page-head">
          <div>
            <h1>Abstimmungen</h1>
            <p class="lede">Entscheidungen der Eigentümergemeinschaft – abstimmen und Ergebnisse nachvollziehen.</p>
          </div>
          {{if .HasBallots}}<span class="pill">{{.BallotCountLabel}}</span>{{end}}
        </div>
        {{if .VoteMsg}}<p class="flash {{if .VoteOK}}ok{{end}}">{{.VoteMsg}}</p>{{end}}
        {{if not .HasBallots}}
        <section class="vote-blank {{.VoteOverviewClass}}" aria-labelledby="vote-blank-title">
          <div class="vote-blank-main">
            <div class="vote-blank-lead">
              <span class="vote-blank-icon" aria-hidden="true"><svg viewBox="0 0 24 24"><path d="M5 8h14v12H5z"/><path d="M9 4h6v4H9z"/><path d="m9 14 2 2 4-4"/></svg></span>
              <h2 id="vote-blank-title">{{.VoteOverviewTitle}}</h2>
              <p>{{.VoteOverviewText}}</p>
              {{if and (not .CanVote) (not .CanManageVotes)}}<p class="vote-blank-note">Ihr Zugang hat kein Stimmrecht. Ergebnisse und Protokolle bleiben einsehbar.</p>{{end}}
            </div>
            <div class="vote-blank-actions">
              {{if .CanManageVotes}}
                {{template "ballotCreateTrigger" .}}
              {{else}}
                <a class="button primary" href="/app/anliegen?new=1">Thema vorschlagen</a>
                <a class="button ghost" href="/app/kontakte">Verwaltung im Kontaktverzeichnis</a>
              {{end}}
            </div>
          </div>
          <aside class="vote-blank-side">
            <h2>So läuft eine Abstimmung</h2>
            <ol class="vote-blank-steps">
              <li><strong>Entwurf</strong><span>Die Verwaltung legt Frage, Antwortmöglichkeiten und Frist an.</span></li>
              <li><strong>Offen</strong><span>Stimmberechtigte Eigentümer stimmen ab und können ihre Stimme bis zur Frist ändern.</span></li>
              <li><strong>Ergebnis</strong><span>Nach dem Schließen sind Ergebnis, Beteiligung und Protokoll abrufbar.</span></li>
            </ol>
          </aside>
          <ul class="vote-blank-facts">
            <li><strong>Gewichtung</strong><span>Je nach Abstimmung zählt der Miteigentumsanteil oder eine Stimme pro Kopf.</span></li>
            <li><strong>Frist</strong><span>Ist eine Frist gesetzt, erinnert das Portal vor Ablauf alle, die noch nicht abgestimmt haben.</span></li>
            <li><strong>Nachvollziehbar</strong><span>Abgeschlossene Abstimmungen bleiben mit Ergebnis und Protokoll dauerhaft erreichbar.</span></li>
          </ul>
        </section>
        {{end}}
        {{if .HasBallots}}<div class="vote-overview {{.VoteOverviewClass}}">
          <span class="vote-overview-icon" aria-hidden="true">{{if eq .VoteOverviewClass "action"}}!{{else}}✓{{end}}</span>
          <div><strong>{{.VoteOverviewTitle}}</strong><p>{{.VoteOverviewText}}</p></div>
        </div>
        <section class="panel vote-library">
          <div class="kicker">{{if .CanManageVotes}}Abstimmungsübersicht{{else}}Ihre Abstimmungen{{end}}</div>
          <div class="vote-list">
            {{range .Ballots}}
                <article class="vote-card{{if .NeedsVote}} needs-action{{end}}" id="ballot-{{.ID}}">
                  <div class="vote-card-head">
                    <div>
                      {{if .NeedsVote}}<div class="vote-eyebrow">Ihre Stimme ist gefragt</div>
                      {{else if .IsDraft}}<div class="vote-eyebrow">Vor Veröffentlichung prüfen</div>
                      {{else if .IsClosed}}
                      {{else if .HasVote}}<div class="vote-eyebrow ok">Stimme gespeichert</div>
                      {{else}}<div class="vote-eyebrow">Offene Abstimmung</div>{{end}}
                      <h3>{{.Title}}</h3>
                    </div>
                    <span class="pill {{.StatusClass}}">{{.Status}}</span>
                  </div>

                  {{if .IsDraft}}
                    <span class="vote-deadline">Entwurf · noch nicht sichtbar</span>
                  {{else if and .IsOpen .HasClosesAt}}
                    <span class="vote-deadline"><svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="8"/><path d="M12 7v5l3 2"/></svg>Offen bis {{.ClosesAt}}</span>
                  {{else if .IsOpen}}
                    <span class="vote-deadline">Jetzt offen</span>
                  {{end}}

                  {{if .HasDescription}}<p class="vote-question">{{.Description}}</p>{{end}}
                  {{template "attachmentStrip" .}}

                  {{if .CanVote}}
                    <form method="post" action="/app/abstimmungen">
                      <input type="hidden" name="ballot_id" value="{{.ID}}">
                      <p class="vote-weight"><svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="8" r="3"/><path d="M6 20v-2a6 6 0 0 1 12 0v2"/></svg>Ihre Stimme zählt: {{.VoteWeight}}</p>
                      <div class="vote-options">
                        {{range .Options}}
                          <label class="vote-option"><input type="radio" name="option" value="{{.Value}}" required{{if .Selected}} checked{{end}}> <span>{{.Label}}</span></label>
                        {{end}}
                      </div>
                      <div class="vote-actions">
                        <span class="vote-current">{{if .HasVote}}Aktuell gewählt: {{.VoteOption}} · gespeichert {{.VotedAt}}{{else}}Bitte eine Option auswählen.{{end}}</span>
                        <button class="button primary" type="submit">{{if .HasVote}}Stimme ändern{{else}}Stimme speichern{{end}}</button>
                      </div>
                    </form>
                  {{else if and .IsOpen (not .HasResults)}}
                    <div class="vote-options">
                      {{range .Options}}<div class="vote-option vote-option-static"><span></span><span>{{.Label}}</span></div>{{end}}
                    </div>
                    {{if .ReadOnlyMessage}}<p class="vote-readonly-note">{{.ReadOnlyMessage}}</p>{{end}}
                  {{else if and .IsOpen .ReadOnlyMessage (not .CanManage)}}
                    <p class="mini">{{.ReadOnlyMessage}}</p>
                  {{end}}

                  {{if .CanManage}}
                    <div class="vote-management">
                      <div class="vote-management-copy">
                        {{if .CanOpen}}<strong>Nächster Schritt: Abstimmung öffnen</strong><span>Danach können stimmberechtigte Eigentümer teilnehmen.</span>
                        {{else if .CanClose}}<strong>Abstimmung läuft</strong><span>Schließen beendet die Stimmabgabe und veröffentlicht das Ergebnis.</span>
                        {{else}}<strong>Abstimmung abgeschlossen</strong><span>Ergebnis und Protokoll bleiben dauerhaft erreichbar.</span>{{end}}
                      </div>
                      <div class="vote-management-actions">
                        {{if .CanOpen}}<form method="post" action="/app/abstimmungen/open"><input type="hidden" name="id" value="{{.ID}}"><button class="button primary" type="submit">Abstimmung öffnen</button></form>{{end}}
                        {{if .CanClose}}<form method="post" action="/app/abstimmungen/close"><input type="hidden" name="id" value="{{.ID}}"><button class="button" type="submit">Abstimmung schließen</button></form>{{end}}
                      </div>
                    </div>
                  {{end}}

                  {{if and .HasResults .IsClosed}}
                    {{template "ballotResult" .}}
                  {{else if and .HasResults .IsOpen}}
                    <details class="vote-live-result">
                      <summary>Zwischenstand anzeigen</summary>
                      {{template "ballotResult" .}}
                    </details>
                  {{end}}

                  <details class="vote-details">
                    <summary>Details zur Abstimmung</summary>
                    <div class="vote-detail-body">
                      <span>{{.Type}}</span>
                      <span>Gewichtung: {{.Weighting}}</span>
                      {{if .HasQuorum}}<span>Quorum: {{.Quorum}}</span>{{end}}
                      {{if .HasOpensAt}}<span>Start: {{.OpensAt}}</span>{{end}}
                      {{if .HasClosesAt}}<span>Frist: {{.ClosesAt}}</span>{{end}}
                      {{if .HasClosesAt}}<span>Erinnerung: {{.ReminderLabel}}</span>{{end}}
                      {{if .IsDraft}}<div class="vote-detail-options">{{range .Options}}<span class="pill">{{.Label}}</span>{{end}}</div>{{end}}
                    </div>
                  </details>
                </article>
            {{end}}
          </div>
        </section>{{end}}
      </section>

      {{if .CanManageVotes}}
      <dialog id="ballot-create" class="dialog ballot-dialog" aria-labelledby="ballot-create-title">
        <form method="post" action="/app/abstimmungen" enctype="multipart/form-data">
          <div class="dialog-head">
            <h2 id="ballot-create-title">Abstimmung anlegen</h2>
            <button class="dialog-close" type="button" data-close-dialog aria-label="Schließen">&times;</button>
          </div>
          <div class="dialog-body">
            <p class="document-dialog-intro">Titel, Frage, Optionen und Frist genügen für einen Entwurf. Regeln und Unterlagen sind optional.</p>
            <div class="dialog-grid">
              <label class="full" for="ballot-title">Kurzer Titel<input id="ballot-title" name="title" required maxlength="160" autocomplete="off" placeholder="Zum Beispiel Fassadensanierung 2026"></label>
              <label class="full" for="ballot-description">Frage oder Erklärung<textarea id="ballot-description" name="description" placeholder="Was sollen die Eigentümer entscheiden?"></textarea></label>
              <label class="full" for="ballot-options">Antwortmöglichkeiten<textarea id="ballot-options" name="options_text" required placeholder="Ja&#10;Nein&#10;Enthaltung"></textarea></label>
              <label class="full" for="ballot-closes">Abstimmungsfrist<input id="ballot-closes" type="datetime-local" name="closes_at"></label>
              <details class="dialog-optional full">
                <summary>Abstimmungsregeln</summary>
                <div class="dialog-optional-grid">
                  <label for="ballot-type">Typ<select id="ballot-type" name="type" required>
                    <option value="Umlaufbeschluss">Umlaufbeschluss</option>
                    <option value="Versammlung">Versammlung</option>
                  </select></label>
                  <label for="ballot-weighting">Gewichtung<select id="ballot-weighting" name="weighting" required>
                    <option value="per-share">nach Miteigentumsanteil</option>
                    <option value="per-head">pro Kopf</option>
                  </select></label>
                  <label for="ballot-opens">Geplanter Start<input id="ballot-opens" type="datetime-local" name="opens_at" value="{{.NowInput}}"></label>
                  <label for="ballot-quorum">Quorum in %<input id="ballot-quorum" name="quorum_percent" inputmode="decimal" placeholder="50"></label>
                  <label class="full" for="ballot-reminder">Erinnerung vor Frist (h)<input id="ballot-reminder" name="reminder_before_hours" inputmode="decimal" value="24"></label>
                </div>
              </details>
              <details class="dialog-optional full">
                <summary>Unterlagen hinzufügen</summary>
                <div class="dialog-optional-grid">
                  <label class="full" for="ballot-attachments">Anhänge<span class="file-control"><input id="ballot-attachments" type="file" name="attachments" accept="image/jpeg,image/png,image/webp,image/gif,application/pdf" multiple><span>Bis zu 10 Dateien auswählen</span></span></label>
                </div>
              </details>
            </div>
            <button class="button primary" type="submit">Entwurf anlegen</button>
          </div>
        </form>
      </dialog>
      {{end}}
    </main>
{{template "appClose" .}}
{{end}}

{{define "handoverCard"}}
  <article class="panel handover-card" id="handover-{{.ID}}">
    <div class="handover-head">
      <div class="handover-card-copy">
        <div class="handover-meta"><span class="pill {{.StatusClass}}">{{.Status}}</span><span>{{.Type}}</span></div>
        <h3>{{.Title}}</h3>
        <p class="handover-place">{{if .HasUnit}}{{.UnitLabel}}{{end}}{{if .HasScheduledAt}}<span>{{.ScheduledAt}}</span>{{end}}</p>
      </div>
      {{if .HasConfirmations}}<div class="handover-confirm-progress" aria-label="{{.ConfirmedCount}} von {{.ConfirmationCount}} Bestätigungen"><strong>{{.ConfirmedCount}}/{{.ConfirmationCount}}</strong><span>bestätigt</span></div>{{end}}
    </div>
    <div class="handover-next {{if .IsFiled}}done{{else if .IsReady}}ready{{end}}">
      <div><span>Als Nächstes</span><strong>{{.NextStep}}</strong><small>{{.NextStepDetail}}</small></div>
      {{if .HasFiledDocument}}
        <a class="button primary small" href="{{.FiledDocumentURL}}">Dokument öffnen</a>
      {{else if and .CanFile .CanManageDocuments}}
        <form method="post" action="{{.FileURL}}">
          <input type="hidden" name="id" value="{{.ID}}">
          <button class="button primary small" type="submit">Jetzt ablegen</button>
        </form>
      {{end}}
    </div>
    <details class="handover-details">
      <summary>Protokolldetails anzeigen{{if .HasRooms}} · {{len .Rooms}} {{if eq (len .Rooms) 1}}Raum{{else}}Räume{{end}}{{end}}{{if .HasMeters}} · {{len .Meters}} {{if eq (len .Meters) 1}}Zählerstand{{else}}Zählerstände{{end}}{{end}}{{if .HasKeys}} · {{len .Keys}} {{if eq (len .Keys) 1}}Schlüsselposition{{else}}Schlüsselpositionen{{end}}{{end}}{{if .HasAttachments}} · {{len .Attachments}} {{if eq (len .Attachments) 1}}Datei{{else}}Dateien{{end}}{{end}}</summary>
      <div class="handover-parties">
        <div><span>Ausziehend</span><strong>{{.Outgoing}}</strong></div>
        <div><span>Einziehend</span><strong>{{.Incoming}}</strong></div>
      </div>
      <div class="handover-detail-grid">
        <section class="handover-detail">
          <h4>Räume</h4>
          {{if .HasRooms}}<ul>{{range .Rooms}}<li><strong>{{.Name}}</strong>{{if .Condition}}<span>{{.Condition}}</span>{{end}}{{if .Defects}}<em>{{.Defects}}</em>{{end}}</li>{{end}}</ul>{{else}}<p>Keine Räume erfasst.</p>{{end}}
        </section>
        <section class="handover-detail">
          <h4>Zählerstände</h4>
          {{if .HasMeters}}<ul>{{range .Meters}}<li><strong>{{.Label}}</strong><span>{{.Value}}{{if .Unit}} {{.Unit}}{{end}}</span></li>{{end}}</ul>{{else}}<p>Keine Zählerstände erfasst.</p>{{end}}
        </section>
        <section class="handover-detail">
          <h4>Schlüssel</h4>
          {{if .HasKeys}}<ul>{{range .Keys}}<li><strong>{{.Label}}</strong><span>{{.Count}} Stk.</span></li>{{end}}</ul>{{else}}<p>Keine Schlüssel erfasst.</p>{{end}}
        </section>
        <section class="handover-detail">
          <h4>Bestätigungen</h4>
          {{if .HasConfirmations}}<ul>{{range .Confirmations}}<li><strong>{{.Role}}</strong><span>{{if .Name}}{{.Name}}{{else}}{{.Email}}{{end}}</span><span class="pill {{.StatusClass}}">{{.Status}}</span></li>{{end}}</ul>{{else}}<p>Keine externen Bestätigungen vorgesehen.</p>{{end}}
        </section>
      </div>
      {{if .HasNotes}}<div class="handover-note"><strong>Notiz</strong><p>{{.Notes}}</p></div>{{end}}
      {{template "attachmentStrip" .AttachmentGroup}}
      {{if .CanChangeFiles}}
        <form class="handover-add-files" method="post" action="/app/uebergaben/attachments" enctype="multipart/form-data">
          <input type="hidden" name="id" value="{{.ID}}">
          <label for="handover-files-{{.ID}}">Fotos oder PDF ergänzen<span class="file-control"><input id="handover-files-{{.ID}}" type="file" name="attachments" accept="image/jpeg,image/png,image/webp,image/gif,application/pdf" multiple required><span>Dateien auswählen</span></span></label>
          <button class="button small" type="submit">Dateien ergänzen</button>
        </form>
      {{end}}
      <div class="handover-detail-actions">
        <a class="button small" href="{{.ProtocolURL}}">PDF exportieren</a>
        <span>Angelegt {{.CreatedAt}} · Aktualisiert {{.UpdatedAt}}</span>
      </div>
    </details>
  </article>
{{end}}

{{define "handovers"}}
{{template "appOpen" .}}
    <script src="/assets/attachments.js?v={{.AssetVersion}}" defer></script>
    <main id="main-content" tabindex="-1" class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M7 4h10v16H7z"/><path d="M9.5 8h5M9.5 12h4"/><path d="m9.5 16 1.5 1.5 3.5-4"/></svg><span>/</span><span>Übergaben</span></span>
        <div class="page-actions">
          <button class="button primary" type="button" data-dialog="handover-create" aria-haspopup="dialog" aria-controls="handover-create">Übergabe anlegen</button>
        </div>
      </div>
      <section class="page wide handover-page">
        <div class="handover-page-head">
          <div>
            <h1>Übergaben</h1>
            <p class="lede">Überblick und nächste Schritte.</p>
          </div>
          <p class="handover-scope">Zustand dokumentieren – Kaution, Schadenabrechnung und Buchhaltung bleiben bewusst außerhalb.</p>
        </div>
        {{if .HandoverMsg}}<div class="flash {{if .HandoverOK}}ok{{end}}">{{.HandoverMsg}}</div>{{end}}
        {{if .HasHandovers}}
          <div class="handover-metrics" aria-label="Übergabeübersicht">
            <div><strong>{{.HandoverOpenCount}}</strong><span>Jetzt offen</span></div>
            <div class="ready"><strong>{{.HandoverReadyCount}}</strong><span>Bereit zur Ablage</span></div>
            <div class="done"><strong>{{.HandoverFiledCount}}</strong><span>Abgeschlossen</span></div>
          </div>
          <div class="handover-sections">
            {{range .HandoverSections}}
              {{if .HasItems}}
                <details class="handover-section" {{if .Open}}open{{end}}>
                  <summary><span><strong>{{.Title}}</strong><small>{{.Description}}</small></span><span class="handover-section-count">{{.Count}}</span></summary>
                  <div class="handover-list">{{range .Items}}{{template "handoverCard" .}}{{end}}</div>
                </details>
              {{else}}
                <div class="handover-section is-empty">
                  <span><strong>{{.Title}}</strong><small>{{.Description}}</small></span>
                  <span class="handover-section-count">0</span>
                </div>
              {{end}}
            {{end}}
          </div>
        {{else}}
          <section class="handover-blank" aria-labelledby="handover-blank-title">
            <div class="handover-blank-main">
              <div class="handover-blank-lead">
                <span class="handover-blank-icon" aria-hidden="true"><svg viewBox="0 0 24 24"><path d="M7 4h10v16H7z"/><path d="M9.5 8h5M9.5 12h4"/><path d="m9.5 16 1.5 1.5 3.5-4"/></svg></span>
                <h2 id="handover-blank-title">{{.HandoversEmpty.Title}}</h2>
                <p>Ein Übergabeprotokoll hält fest, in welchem Zustand eine Einheit übergeben wurde: Räume, Zählerstände, Schlüssel, Fotos und die Bestätigung von ausziehender und einziehender Partei. Genau das ist später der Nachweis, wenn jemand nachfragt.</p>
              </div>
              <div class="handover-blank-actions">
                <button class="button primary" type="button" data-dialog="handover-create" aria-haspopup="dialog" aria-controls="handover-create">Erste Übergabe anlegen</button>
                {{if .CanManageBuilding}}<a class="button" href="/app/settings/building#units">Einheiten prüfen</a>{{end}}
              </div>
            </div>
            <aside class="handover-blank-side">
              <h2>So entsteht ein Protokoll</h2>
              <ol class="handover-blank-steps">
                <li><strong>Anlegen</strong><span>Einheit, Anlass und Termin. Für den Start genügt eine Zeile je Raum.</span></li>
                <li><strong>Ergänzen</strong><span>Zählerstände, Schlüssel, Notiz sowie Fotos und PDF kommen danach dazu.</span></li>
                <li><strong>Bestätigen</strong><span>Beide Parteien bestätigen über einen persönlichen Link. Ab der ersten Bestätigung bleibt das Protokoll unverändert.</span></li>
                <li><strong>Ablegen</strong><span>Vollständig bestätigt wandert es als Dokument in die Ablage des Hauses.</span></li>
              </ol>
              <p class="handover-blank-note">Ohne Einheit lässt sich kein Protokoll anlegen. Einheiten pflegen Sie in den Gebäude-Einstellungen.</p>
            </aside>
            <ul class="handover-blank-facts">
              <li><strong>Jetzt offen</strong><span>Angelegte Protokolle, die noch auf eine Bestätigung warten. Diese Liste steht hier zuerst.</span></li>
              <li><strong>Bereit zur Ablage</strong><span>Vollständig bestätigt. Ein Klick legt daraus das Dokument im Dokumentenbereich an.</span></li>
              <li><strong>Abgeschlossen</strong><span>Abgelegte Protokolle bleiben mit allen Anhängen als PDF abrufbar.</span></li>
            </ul>
          </section>
        {{end}}
      </section>

      <dialog id="handover-create" class="dialog handover-dialog" aria-labelledby="handover-create-title">
        <form method="post" action="/app/uebergaben" enctype="multipart/form-data">
          <div class="dialog-head">
            <h2 id="handover-create-title">Übergabe anlegen</h2>
            <button class="dialog-close" type="button" data-close-dialog aria-label="Schließen">&times;</button>
          </div>
          <div class="dialog-body">
            <p class="document-dialog-intro">Die drei Pflichtangaben zuerst; Personen, Messwerte und Fotos lassen sich danach gezielt ergänzen.</p>
            <div class="handover-form-step"><span>1</span><div><strong>Übergabe</strong><small>Einheit, Anlass und Termin</small></div></div>
            <div class="dialog-grid">
              <label class="full" for="handover-title">Kurzer Titel<input id="handover-title" name="title" required maxlength="160" placeholder="Nutzerwechsel Top 11"></label>
              <label for="handover-unit">Einheit<select id="handover-unit" name="unit_id" required>{{range .UnitOptions}}<option value="{{.Value}}" {{if .Selected}}selected{{end}}>{{.Label}}</option>{{end}}</select></label>
              <label for="handover-type">Anlass<select id="handover-type" name="handover_type"><option>Nutzerwechsel</option><option>Einzug</option><option>Auszug</option></select></label>
              <label class="full" for="handover-time">Termin<input id="handover-time" type="datetime-local" name="scheduled_at" value="{{.NowInput}}"></label>
            </div>
            <div class="handover-form-step"><span>2</span><div><strong>Zustand</strong><small>Eine Zeile je Raum: Raum | Zustand | Mangel</small></div></div>
            <label for="handover-rooms">Räume<input id="handover-rooms" name="rooms_text" required placeholder="Wohnzimmer | gut | keine Mängel"></label>
            <details class="dialog-optional">
              <summary>Personen für Bestätigung</summary>
              <p class="dialog-optional-copy">Nur mit E-Mail wird ein persönlicher Bestätigungslink vorbereitet.</p>
              <div class="dialog-optional-grid">
                <label for="handover-out-name">Ausziehend – Name<input id="handover-out-name" name="outgoing_name" autocomplete="name"></label>
                <label for="handover-out-email">Ausziehend – E-Mail<input id="handover-out-email" type="email" name="outgoing_email" autocomplete="email"></label>
                <label for="handover-in-name">Einziehend – Name<input id="handover-in-name" name="incoming_name" autocomplete="name"></label>
                <label for="handover-in-email">Einziehend – E-Mail<input id="handover-in-email" type="email" name="incoming_email" autocomplete="email"></label>
              </div>
            </details>
            <details class="dialog-optional">
              <summary>Zähler, Schlüssel und Notiz</summary>
              <div class="dialog-optional-grid">
                <label class="full" for="handover-meters">Zählerstände <small>Bezeichnung | Wert | Einheit</small><textarea id="handover-meters" name="meters_text" placeholder="Strom | 12345,6 | kWh"></textarea></label>
                <label class="full" for="handover-keys">Schlüssel <small>Bezeichnung | Anzahl</small><textarea id="handover-keys" name="keys_text" placeholder="Wohnungsschlüssel | 3"></textarea></label>
                <label class="full" for="handover-notes">Notiz<textarea id="handover-notes" name="notes" placeholder="Zusätzliche Beobachtungen"></textarea></label>
              </div>
            </details>
            <details class="dialog-optional">
              <summary>Fotos und PDF</summary>
              <div class="dialog-optional-grid"><label class="full" for="handover-attachments">Dateien<span class="file-control"><input id="handover-attachments" type="file" name="attachments" accept="image/jpeg,image/png,image/webp,image/gif,application/pdf" multiple><span>Bis zu 10 Dateien auswählen</span></span></label></div>
            </details>
          </div>
          <div class="dialog-footer handover-dialog-submit"><span>Links werden nach dem Speichern versendet.</span><button class="button primary" type="submit">Übergabe anlegen</button></div>
        </form>
      </dialog>
    </main>
{{template "appClose" .}}
{{end}}

{{define "handoverConfirm"}}
<!doctype html>
<html lang="de">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  <link rel="icon" type="image/svg+xml" href="/favicon.svg">
  {{template "appStyles" .}}
</head>
<body class="handover-confirm-body">
  <main class="handover-confirm-page">
    <section class="panel handover-confirm-card">
      <header class="handover-confirm-head">
        <div class="handover-meta"><span class="pill unread">{{.Confirmation.Role}}</span>{{if .Handover.HasUnit}}<span>{{.Handover.UnitLabel}}</span>{{end}}</div>
        <h1>{{.Handover.Title}}</h1>
        <p>{{.Tenant.Address}}{{if .Handover.HasScheduledAt}}<span>{{.Handover.ScheduledAt}}</span>{{end}}</p>
      </header>
      {{if .Msg}}<div class="flash {{if .MsgOK}}ok{{end}}">{{.Msg}}</div>{{end}}
      <section class="handover-review" aria-labelledby="handover-review-title">
        <div class="handover-review-head"><div><span>Vor der Bestätigung</span><h2 id="handover-review-title">Protokoll prüfen</h2></div><span class="pill {{.Confirmation.StatusClass}}">{{.Confirmation.Status}}</span></div>
        {{if .Handover.HasRooms}}<details open><summary><span><strong>Räume</strong><small>{{len .Handover.Rooms}} Einträge</small></span></summary><ul>{{range .Handover.Rooms}}<li><strong>{{.Name}}</strong><span>{{if .Condition}}{{.Condition}}{{else}}–{{end}}{{if .Defects}} · {{.Defects}}{{end}}</span></li>{{end}}</ul></details>{{end}}
        {{if .Handover.HasMeters}}<details><summary><span><strong>Zählerstände</strong><small>{{len .Handover.Meters}} Einträge</small></span></summary><ul>{{range .Handover.Meters}}<li><strong>{{.Label}}</strong><span>{{.Value}}{{if .Unit}} {{.Unit}}{{end}}</span></li>{{end}}</ul></details>{{end}}
        {{if .Handover.HasKeys}}<details><summary><span><strong>Schlüssel</strong><small>{{len .Handover.Keys}} Positionen</small></span></summary><ul>{{range .Handover.Keys}}<li><strong>{{.Label}}</strong><span>{{.Count}} Stk.</span></li>{{end}}</ul></details>{{end}}
        {{if .Handover.HasNotes}}<details><summary><span><strong>Notiz</strong><small>1 Eintrag</small></span></summary><p>{{.Handover.Notes}}</p></details>{{end}}
        {{if .Handover.HasAttachments}}<div class="handover-review-files"><strong>Fotos &amp; Dateien</strong><span>{{len .Handover.Attachments}} zum Protokoll gespeichert</span></div>{{end}}
      </section>
      <p class="handover-scope">Dieses Protokoll dokumentiert den Zustand bei der Übergabe. Es ist keine Kautions-, Schaden- oder sonstige Abrechnung.</p>
      {{if .Confirmation.HasConfirmed}}
        <p class="empty">Bestätigt{{if .Confirmation.HasConfirmed}} am {{.Confirmation.ConfirmedAt}}{{end}}. Es ist nichts mehr zu tun.</p>
      {{else}}
        <form method="post" action="/handover/{{.Token}}" class="handover-confirm-form">
          <label>Name für die Bestätigung<input name="name" value="{{.Confirmation.Name}}" autocomplete="name"></label>
          <details class="handover-confirm-note"><summary>Notiz ergänzen (optional)</summary><label>Notiz<textarea name="note" placeholder="Falls etwas ergänzt werden soll"></textarea></label></details>
          <label class="handover-confirm-consent"><input type="checkbox" name="confirm" value="yes" required><span>Ich habe das Protokoll vollständig geprüft und bestätige den dokumentierten Stand.</span></label>
          <div class="handover-confirm-submit"><button class="button primary" type="submit">Verbindlich bestätigen</button><small>Die Bestätigung wird mit Zeitpunkt und Rolle protokolliert.</small></div>
        </form>
      {{end}}
    </section>
  </main>
</body>
</html>
{{end}}

{{define "parking"}}
{{template "appOpen" .}}
    <script src="/assets/attachments.js?v={{.AssetVersion}}" defer></script>
    <main id="main-content" tabindex="-1" class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg><span>/</span><span>Parkplatznutzung</span></span>
        <div class="page-actions"></div>
      </div>
      <section class="page wide parking-page">
        <div class="parking-page-head">
          <div>
            <h1>Parkplatz</h1>
            <p class="lede">Laden und Monatskosten auf einen Blick.</p>
          </div>
          <div class="parking-primary-actions">
            <details class="parking-more">
              <summary class="button">Mehr</summary>
              <div class="parking-more-menu">
                <a class="button" href="/app/parking"><svg viewBox="0 0 24 24" width="17" height="17" aria-hidden="true"><path d="M4 4v6h6" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"/><path d="M20 20v-6h-6" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"/><path d="M5 10a7 7 0 0 1 12-3M19 14a7 7 0 0 1-12 3" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round"/></svg>Aktualisieren</a>
                <a class="button" href="/app/parking/export/{{.StatementYear}}">CSV exportieren</a>
                {{if .CanManageParkingPayments}}<form method="post" action="/app/parking/reminders"><button class="button ghost" type="submit" title="Nur fällige, noch nicht erinnerte Monate benachrichtigen">Erinnerungen senden</button></form>{{end}}
                {{if .IsAdmin}}<a class="button ghost" href="/app/parking/settings"><svg viewBox="0 0 24 24" width="17" height="17" aria-hidden="true"><path d="M12 8.5a3.5 3.5 0 1 0 0 7 3.5 3.5 0 0 0 0-7z" fill="none" stroke="currentColor" stroke-width="1.9"/><path d="M19 12a7 7 0 0 0-.1-1l2-1.5-2-3.5-2.4 1a7 7 0 0 0-1.8-1L14.4 3h-4.8L9.3 6a7 7 0 0 0-1.8 1l-2.4-1-2 3.5 2 1.5A7 7 0 0 0 5 12a7 7 0 0 0 .1 1l-2 1.5 2 3.5 2.4-1a7 7 0 0 0 1.8 1l.3 3h4.8l.3-3a7 7 0 0 0 1.8-1l2.4 1 2-3.5-2-1.5a7 7 0 0 0 .1-1z" fill="none" stroke="currentColor" stroke-width="1.9"/></svg>Abrechnung konfigurieren</a>{{end}}
              </div>
            </details>
          </div>
        </div>
        {{if and .Live.Available .Accounting.HasMonths}}
          <nav class="parking-switcher" aria-label="Parkplatzbereiche">
            <a href="#parking-live">Jetzt</a>
            <a href="#parking-months">Monate</a>
          </nav>
        {{end}}

        {{if .ParkingMsg}}<p class="flash {{if .ParkingOK}}ok{{end}}">{{.ParkingMsg}}</p>{{end}}

        {{template "parkingLiveCard" .}}

        {{if .Accounting.HasMonths}}
          <section class="panel parking-months-panel" id="parking-months" aria-label="Monatsabrechnungen">
            <div class="parking-months-head">
              <div>
                <h2>Monate</h2>
                <p class="muted">Aktueller Betrag und Zahlungsstatus.</p>
              </div>
              <div class="parking-months-status">
                {{if .Accounting.HasOutstanding}}<span class="pill">{{.Accounting.Outstanding}} offen</span>{{end}}
                {{if .Accounting.HasOverdue}}<span class="pill dringend">{{.Accounting.Overdue}} überfällig</span>{{end}}
              </div>
            </div>

            {{with .CurrentMonth}}
              <a class="parking-current-month" id="parking-month-{{.Month}}" href="{{.DetailPath}}">
                <span>
                  <span class="metric-label">{{$.CurrentMonthHeading}}</span>
                  <strong>{{.MonthLabel}}</strong>
                  <b>{{.TotalCost}}</b>
                </span>
                <span class="parking-current-facts">
                  <span><strong>{{.KWh}}</strong><small>Verbrauch</small></span>
                  <span><strong>{{.GridCost}}</strong><small>Netzgebühr</small></span>
                </span>
                <span class="pill {{if .Paid}}ok{{else if .Overdue}}dringend{{end}}">{{.PaidLabel}}</span>
                <span class="parking-row-arrow" aria-hidden="true">›</span>
              </a>
            {{end}}

            {{if .HasOlderMonths}}
              <div class="parking-month-list" aria-label="Ältere Monate">
                {{range .OlderMonths}}
                  <a class="parking-compact-month" id="parking-month-{{.Month}}" href="{{.DetailPath}}">
                    <span><strong>{{.MonthLabel}}</strong><small>{{if .Partial}}Teilmonat · {{end}}{{.HourCount}} Stunden</small></span>
                    <span class="amount">{{.TotalCost}}</span>
                    <span class="pill {{if .Paid}}ok{{else if .Overdue}}dringend{{end}}">{{.PaidLabel}}</span>
                    <span class="parking-row-arrow" aria-hidden="true">›</span>
                  </a>
                {{end}}
              </div>
            {{end}}

            <details class="parking-utility">
              <summary>Wie wird gerechnet?</summary>
              <div class="parking-utility-body">
                <p class="muted">{{.Accounting.Message}}</p>
                <p class="mini">Letzter Messpunkt: {{if .Accounting.LastSampleLabel}}{{.Accounting.LastSampleLabel}}{{else}}–{{end}}</p>
              </div>
            </details>
          </section>
        {{else}}
          <section class="panel parking-empty" aria-label="Parkplatznutzung einrichten">
            <div class="parking-empty-art" aria-hidden="true">
              <svg viewBox="0 0 240 190">
                <circle class="soft-fill" cx="118" cy="94" r="74"/>
                <path d="M55 146h138"/>
                <path d="M64 126h30"/>
                <path d="M146 126h35"/>
                <path d="M91 126h14l13-30h46l16 30h13"/>
                <path d="M118 96h24M146 96h18"/>
                <circle cx="113" cy="132" r="11"/>
                <circle cx="171" cy="132" r="11"/>
                <path d="M91 126v-16M194 126v-12"/>
                <path d="M70 146V68"/>
                <path d="M52 68h36v38H52z"/>
                <path d="M63 98V76h10.5a7 7 0 0 1 0 14H63"/>
                <path d="M102 64h42l20 18"/>
                <path d="M117 64v31"/>
                <path d="M159 82h25v24"/>
                <path d="M36 129c-10 0-18-7-18-17 0-8 6-15 14-16 3-10 12-17 23-17 12 0 22 8 25 19"/>
              </svg>
            </div>
            <div class="parking-empty-copy">
              <div class="kicker">Bereit für die erste Abrechnung</div>
              <h2>Noch keine Monatswerte</h2>
              <p class="muted">Tarif, zwei Zählerstände und ein Preis genügen. Danach rechnet das Portal automatisch.</p>
            </div>
            <div class="parking-empty-actions">
              {{if .IsAdmin}}<a class="button primary" href="/app/parking/settings"><svg viewBox="0 0 24 24" width="17" height="17" aria-hidden="true"><path d="M12 8.5a3.5 3.5 0 1 0 0 7 3.5 3.5 0 0 0 0-7z" fill="none" stroke="currentColor" stroke-width="1.9"/><path d="M19 12a7 7 0 0 0-.1-1l2-1.5-2-3.5-2.4 1a7 7 0 0 0-1.8-1L14.4 3h-4.8L9.3 6a7 7 0 0 0-1.8 1l-2.4-1-2 3.5 2 1.5A7 7 0 0 0 5 12a7 7 0 0 0 .1 1l-2 1.5 2 3.5 2.4-1a7 7 0 0 0 1.8 1l.3 3h4.8l.3-3a7 7 0 0 0 1.8-1l2.4 1 2-3.5-2-1.5a7 7 0 0 0 .1-1z" fill="none" stroke="currentColor" stroke-width="1.9"/></svg>Abrechnung konfigurieren</a>{{end}}
              {{if .CanManageParkingPayments}}<a class="button ghost" href="/app/settings/parking-access" title="Festlegen, wer den Parkplatz nutzen darf"><svg viewBox="0 0 24 24" width="17" height="17" aria-hidden="true"><path d="M8.5 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6z" fill="none" stroke="currentColor" stroke-width="1.9"/><path d="M3.5 20a5 5 0 0 1 10 0" fill="none" stroke="currentColor" stroke-width="1.9"/><path d="M17 8v8M13 12h8" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round"/></svg>Zugriff verwalten</a>{{end}}
              {{if and (not .IsAdmin) (not .CanManageParkingPayments)}}<a class="button ghost" href="/app">Hausüberblick öffnen</a>{{end}}
            </div>
            <div class="parking-empty-steps">
              <div class="parking-empty-step">
                <span class="parking-empty-step-number">1</span>
                <strong>Tarif</strong>
              </div>
              <div class="parking-empty-step">
                <span class="parking-empty-step-number">2</span>
                <strong>Messwerte</strong>
              </div>
              <div class="parking-empty-step">
                <span class="parking-empty-step-number">3</span>
                <strong>Monat</strong>
              </div>
            </div>
            <div class="parking-empty-note">
              <span class="parking-empty-note-icon"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 8v5"/><path d="M12 17h.01"/><circle cx="12" cy="12" r="9"/></svg></span>
              <span><strong>Nur Nachweis.</strong> Keine Buchung, kein Mahnwesen, kein Zahlungsauftrag.</span>
            </div>
          </section>
        {{end}}
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "parkingLiveCard"}}
{{with .Live}}{{if .Available}}
  <section class="panel parking-live" id="parking-live" aria-label="Aktueller Ladezustand Parkplatz 20">
    <div class="parking-live-head">
      <div>
        <span class="metric-label">Jetzt</span>
        <h2>{{.ModeLabel}}</h2>
        <p class="muted">{{.ModeDetail}}</p>
      </div>
      <span class="pill {{.ModeClass}}">{{if .AutoPaused}}Automatik pausiert{{else if .ToggleOn}}Lädt{{else}}Bereit{{end}}</span>
    </div>
    <div class="parking-live-facts">
      {{if .PowerLabel}}<span><small>Leistung</small><strong>{{.PowerLabel}}</strong></span>{{end}}
      {{if .BatterySOCLabel}}<span><small>Hausakku</small><strong class="battery-{{.BatteryClass}}">{{.BatterySOCLabel}}</strong></span>{{end}}
      {{if .FeedInLabel}}<span><small>Einspeisung</small><strong>{{.FeedInLabel}}</strong></span>{{end}}
    </div>
    {{if .CanToggle}}
      <div class="parking-live-actions">
        {{if .AutoPaused}}
          <form method="post" action="/app/parking/charging/auto"><button class="button primary" type="submit">Automatik aktivieren</button></form>
          <details class="parking-manual">
            <summary class="button">Manuell steuern</summary>
            <form method="post" action="/app/parking/charging/on"><button class="button" type="submit">Jetzt laden · Normaltarif</button></form>
          </details>
        {{else if .ToggleOn}}
          <form method="post" action="/app/parking/charging/off"><button class="button" type="submit">Ladung ausschalten</button></form>
        {{else}}
          <form method="post" action="/app/parking/charging/on"><button class="button primary" type="submit">Jetzt laden · Normaltarif</button></form>
        {{end}}
      </div>
    {{end}}
    <details class="parking-live-details">
      <summary>Ladeverbrauch und technische Details</summary>
      <div class="parking-live-details-body">
        {{if .SessionSince}}<p class="mini">Seit {{.SessionSince}}{{if .SessionKWh}} · {{.SessionKWh}}{{if .SessionCost}} · ca. {{.SessionCost}}{{end}}{{end}}</p>{{end}}
        {{if .RateLabel}}<p class="mini">Aktueller Tarif: {{.RateLabel}}</p>{{end}}
        {{if .BatteryHint}}<p class="mini">{{.BatteryHint}}</p>{{end}}
        {{if or .TodaySplit.HasAny .MonthSplit.HasAny}}
          <div class="parking-split-list">
            {{if .TodaySplit.HasAny}}<p><strong>Heute</strong><span>Überschuss {{.TodaySplit.SurplusKWh}} · {{.TodaySplit.SurplusCost}}</span><span>Normal {{.TodaySplit.NormalKWh}} · {{.TodaySplit.NormalCost}}</span></p>{{end}}
            {{if .MonthSplit.HasAny}}<p><strong>Monat</strong><span>Überschuss {{.MonthSplit.SurplusKWh}} · {{.MonthSplit.SurplusCost}}</span><span>Normal {{.MonthSplit.NormalKWh}} · {{.MonthSplit.NormalCost}}</span></p>{{end}}
          </div>
        {{end}}
        {{if .Admin.Show}}
          <div class="parking-admin-state">
            <strong>Regler: {{.Admin.PhaseLabel}}</strong>
            {{if .Admin.SinceLabel}}<span>seit {{.Admin.SinceLabel}}</span>{{end}}
            {{if .Admin.PollLabel}}<span>HA-Poll {{.Admin.PollLabel}}</span>{{end}}
            {{if .Admin.LastReason}}<span>{{.Admin.LastReason}}</span>{{end}}
            {{if .Admin.ErrorDetail}}<span class="pill dringend">{{.Admin.ErrorDetail}}</span>{{end}}
            <a href="/app/parking/settings#laderegelung">Laderegelung öffnen</a>
          </div>
        {{end}}
        {{if .HasSessions}}
          <div>
            <h3>Letzte Ladevorgänge</h3>
            {{template "parkingSessionList" .Sessions}}
          </div>
        {{end}}
        {{if .ShadowMode}}<span class="pill">Testbetrieb</span>{{end}}
        {{if .StaleData}}<span class="pill dringend">Daten veraltet</span>{{end}}
      </div>
      </details>
  </section>
{{end}}{{end}}
{{end}}

{{define "parkingSessionList"}}
  <div class="parking-session-list">
    {{range .}}
      <div class="parking-session-row">
        <span>{{.StartLabel}}</span>
        <span class="mini">{{.DurationLabel}}</span>
        {{if .KWh}}<span>{{.KWh}}</span>{{end}}
        <span class="pill {{.ModeClass}}">{{if eq .ModeClass "mode-surplus"}}☀️ {{end}}{{.ModeLabel}}</span>
        {{if .Active}}<span class="pill ok">läuft</span>{{end}}
        {{if .Cost}}<span class="amount">{{.Cost}}</span>{{end}}
      </div>
    {{end}}
  </div>
{{end}}

{{define "parkingMonth"}}
{{template "appOpen" .}}
    <script src="/assets/attachments.js?v={{.AssetVersion}}" defer></script>
    <style>
      @media print {
        .sidebar, .content-top .page-actions, .parking-month-hero-actions, [data-print], .attachment-strip, .release-dialog { display: none !important; }
        .app-shell, .app-main { display: block !important; margin: 0 !important; padding: 0 !important; }
        .panel { box-shadow: none !important; border-color: #ccc !important; break-inside: avoid; }
        body, .app-shell { background: #fff !important; }
        a[href] { color: inherit !important; text-decoration: none !important; }
        @page { margin: 1.4cm; }
      }
    </style>
    <main id="main-content" tabindex="-1" class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg><span>/</span><a href="/app/parking">Parkplatznutzung</a><span>/</span><span>{{.Detail.MonthLabel}}</span></span>
        <div class="page-actions"><a class="button" href="{{.Detail.BackPath}}#parking-months">Zurück zu Monaten</a></div>
      </div>
      <section class="page wide parking-month-page">
        <div class="parking-month-hero">
          <div>
            <span class="metric-label">Monatsabrechnung</span>
            <h1>{{.Detail.MonthLabel}}</h1>
            <p class="lede">Verbrauch, Kosten und Zahlungsstatus.</p>
          </div>
          {{if .Detail.Summary.Month}}
            <div class="parking-month-hero-actions">
              <details class="parking-more">
                <summary class="button">Exportieren</summary>
                <div class="parking-more-menu">
                  <a class="button" href="/app/parking/month/{{.Detail.Month}}/export">CSV herunterladen</a>
                  <button class="button" type="button" data-print>Drucken / PDF</button>
                </div>
              </details>
            </div>
          {{end}}
        </div>
        {{if .ParkingMsg}}<p class="flash {{if .ParkingOK}}ok{{end}}">{{.ParkingMsg}}</p>{{end}}
        {{if .Detail.Summary.Partial}}<div class="notice warn"><strong>Messdaten unvollständig · Teilmonat:</strong> Werte beginnen nicht am Monatsanfang; bitte vor dem Teilen prüfen.</div>{{end}}
        {{if .Detail.Summary.Month}}
          <section class="panel parking-month-summary">
            <div class="parking-month-total">
              <div>
                <span class="metric-label">Gesamt</span>
                <strong>{{.Detail.Summary.TotalCost}}</strong>
              </div>
              <span class="pill {{if .Detail.Summary.Paid}}ok{{else if .Detail.Summary.Overdue}}dringend{{end}}">{{.Detail.Summary.PaidLabel}}</span>
            </div>

            <dl class="parking-month-essential">
              <div><dt>Verbrauch</dt><dd>{{.Detail.Summary.KWh}}</dd></div>
              <div><dt>Netzgebühr</dt><dd>{{.Detail.Summary.GridCost}}</dd></div>
              <div><dt>Stromkosten</dt><dd>{{.Detail.Summary.EnergyCost}}</dd></div>
            </dl>

            <div class="parking-month-payment">
              {{if .Detail.Summary.Paid}}
                <div class="pay-settled">
                  <span class="pay-check" aria-hidden="true"><svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="10" fill="none" stroke="currentColor" stroke-width="1.8"/><path d="M8 12.5l2.6 2.6L16.5 9" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/></svg></span>
                  <div class="pay-settled-meta">
                    <strong>Bezahlt{{if .Detail.Summary.PaidAtLabel}} am {{.Detail.Summary.PaidAtLabel}}{{end}}</strong>
                    {{if .CanManageParkingPayments}}{{if .Detail.Summary.PaymentMethod}}<span>{{.Detail.Summary.PaymentMethod}}{{if .Detail.Summary.PaymentReference}} · {{.Detail.Summary.PaymentReference}}{{end}}</span>{{end}}{{else}}<span>Zahlung ist markiert.</span>{{end}}
                  </div>
                  {{if .CanManageParkingPayments}}
                    <form class="pay-unmark" method="post" action="/app/parking/month">
                      <input type="hidden" name="month" value="{{.Detail.Summary.Month}}">
                      <input type="hidden" name="paid" value="false">
                      <button class="button pay-ghost" type="submit">Als offen markieren</button>
                    </form>
                  {{end}}
                </div>
              {{else if .CanMarkParkingPayment}}
                <form class="pay-form" method="post" action="/app/parking/month">
                  <input type="hidden" name="month" value="{{.Detail.Summary.Month}}">
                  <input type="hidden" name="paid" value="true">
                  <div class="pay-row">
                    <p class="muted">{{if .CanManageParkingPayments}}Zahlungseingang für diesen Monat bestätigen.{{else}}Nach der Zahlung können Sie den Monat hier abschließen.{{end}}</p>
                    <button class="button primary" type="submit">{{if .CanManageParkingPayments}}Bezahlung erhalten{{else}}Als bezahlt markieren{{end}}</button>
                  </div>
                  <details class="pay-details">
                    <summary>Details (optional)</summary>
                    <div class="pay-fields">
                      <label>Datum<input type="date" name="paid_at" value="{{.Detail.Summary.PaidAtInput}}" autocomplete="off"></label>
                      <label>Zahlungsart<input type="text" name="payment_method" value="{{.Detail.Summary.PaymentMethod}}" placeholder="Überweisung" autocomplete="off"></label>
                      <label class="pay-ref">Referenz<input type="text" name="payment_reference" value="{{.Detail.Summary.PaymentReference}}" placeholder="z. B. Abrechnung Juli" autocomplete="off"></label>
                    </div>
                  </details>
                </form>
              {{end}}
            </div>

            <details class="parking-month-disclosure">
              <summary>Kosten aufschlüsseln</summary>
              <dl class="parking-cost-list">
                {{if .Detail.Summary.HasSurplus}}<div><dt>PV-Überschuss · {{.Detail.Summary.SurplusKWh}}</dt><dd>{{.Detail.Summary.SurplusCost}}</dd></div>{{end}}
                <div><dt>Normalladen · {{.Detail.Summary.NormalKWh}}</dt><dd>{{.Detail.Summary.EnergyCost}}</dd></div>
                <div><dt>Netzgebühr · {{.Detail.GridFeeLabel}}</dt><dd>{{.Detail.Summary.GridCost}}</dd></div>
                <div><dt>Basis</dt><dd>{{.Detail.Summary.BaseFee}}</dd></div>
                <div><dt>Ø aWATTar</dt><dd>{{.Detail.Summary.AverageAwattar}}</dd></div>
                <div><dt>Ø effektiv</dt><dd>{{.Detail.Summary.EffectivePrice}}</dd></div>
              </dl>
            </details>

            <details class="parking-month-disclosure">
              <summary>Stundenwerte · {{.Detail.Summary.HourCount}} Stunden</summary>
              {{if .Detail.HasHours}}
                <div class="parking-hour-list">
                  {{range .Detail.Hours}}
                    <details class="parking-hour-row">
                      <summary><span>{{.AtLabel}}</span><span>{{.KWh}}</span><strong>{{.TotalCost}}</strong></summary>
                      <dl class="parking-hour-details">
                        <div><dt>Überschuss</dt><dd>{{if .HasSurplus}}{{.SurplusKWh}}{{else}}–{{end}}</dd></div>
                        <div><dt>aWATTar</dt><dd>{{.AverageAwattar}}</dd></div>
                        <div><dt>Strom</dt><dd>{{.EnergyCost}}</dd></div>
                        <div><dt>Netzgebühr</dt><dd>{{.GridCost}}</dd></div>
                      </dl>
                    </details>
                  {{end}}
                </div>
              {{else}}
                <p class="empty">Noch keine Stundenwerte gespeichert.</p>
              {{end}}
            </details>

            {{if .Detail.HasSessions}}
              <details class="parking-month-disclosure">
                <summary>Ladevorgänge</summary>
                {{template "parkingSessionList" .Detail.Sessions}}
              </details>
            {{end}}

            {{if and .CanManageParkingPayments .Detail.Summary.HasAttachments}}{{template "attachmentStrip" .Detail.Summary}}{{end}}
            <p class="mini">Messwerte aus Zählerdifferenz und aWATTar-Preis. {{if .Detail.LastSampleLabel}}Letzter Zählerwert: {{.Detail.LastSampleLabel}}.{{end}}</p>
          </section>
        {{end}}
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "settingsHub"}}
{{template "appOpen" .}}
    <style>
      /* Die Bereiche bleiben sofort scannbar; die bereits sichtbaren Marker
         werden nur auf Wunsch in einer kurzen Legende erklärt. */
      .settings-hub { display: grid; gap: 18px; }
      .settings-hub-head { display: grid; gap: 5px; }
      .settings-hub-head .lede { max-width: 620px; }
      .settings-account { display: grid; grid-template-columns: 58px minmax(0,1fr) auto; gap: 14px; align-items: center; padding: 18px; }
      .settings-account-icon { width: 58px; height: 58px; display: grid; place-items: center; border-radius: 50%; background: var(--ink); color: #fff; }
      .settings-account-icon svg { width: 27px; height: 27px; fill: none; stroke: currentColor; stroke-width: 1.7; }
      .settings-account-copy { min-width: 0; }
      .settings-account-copy h2 { font-size: 23px; overflow-wrap: anywhere; }
      .settings-account-copy p { margin-top: 3px; color: var(--muted); font-size: 13px; overflow-wrap: anywhere; }
      .settings-account .pill { justify-self: end; }
      .settings-layout { display: grid; gap: 16px; align-items: start; }
      .settings-sections { min-width: 0; display: grid; grid-template-columns: minmax(0,1fr) minmax(0,1fr); gap: 16px; align-items: stretch; }
      .settings-aside { min-width: 0; }
      .settings-section { padding: 0; overflow: hidden; }
      .settings-section-head { display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 6px 12px; align-items: baseline; padding: 15px 18px 11px; }
      .settings-section-head h2 { font-size: 21px; }
      .settings-section-head p { grid-column: 1 / -1; color: var(--muted); font-size: 12.5px; line-height: 1.4; }
      .settings-tag { justify-self: end; display: inline-flex; align-items: center; gap: 6px; border-radius: 999px; padding: 3px 9px; background: rgba(47,107,74,.1); color: var(--leaf); font-size: 10.5px; font-weight: 850; letter-spacing: .07em; text-transform: uppercase; white-space: nowrap; }
      .settings-tag::before { content: ""; width: 6px; height: 6px; border-radius: 50%; background: currentColor; }
      .settings-tag.house { background: rgba(200,153,63,.16); color: var(--gold-ink); }
      .settings-tag.read { background: #ece8de; color: var(--muted); }
      .settings-links { display: grid; }
      .settings-link { min-height: 76px; display: grid; grid-template-columns: 42px minmax(0,1fr) auto; gap: 12px; align-items: center; border-top: 1px solid var(--line); padding: 12px 17px; color: var(--ink); text-decoration: none; }
      .settings-link:hover { background: var(--panel-soft); }
      .settings-link-icon { width: 42px; height: 42px; display: grid; place-items: center; border-radius: var(--radius-xs); background: rgba(200,153,63,.12); color: var(--gold-ink); }
      .settings-link-icon svg { width: 21px; height: 21px; fill: none; stroke: currentColor; stroke-width: 1.8; }
      .settings-link-copy { min-width: 0; }
      .settings-link-copy strong { display: block; font-family: var(--font-serif); font-size: 18px; }
      .settings-link-copy span { display: block; margin-top: 3px; color: var(--muted); font-size: 12.5px; line-height: 1.35; overflow-wrap: anywhere; }
      .settings-link-copy .settings-home-unit { margin-top: 2px; color: var(--muted); font-size: 12.5px; font-weight: 650; }
      .settings-link-copy .settings-home-purpose { margin-top: 6px; color: var(--soft); font-size: 11.5px; font-weight: 550; }
      .settings-link-arrow { color: var(--gold-ink); font-size: 24px; }
      .settings-link.secondary { min-height: 64px; }
      .settings-link.secondary .settings-link-icon { width: 36px; height: 36px; background: var(--panel-soft); }
      .settings-management { grid-column: 1 / -1; }
      .settings-home-profile { grid-column: 1 / -1; }
      .settings-management .settings-links { grid-template-columns: repeat(2,minmax(0,1fr)); }
      .settings-management .settings-link:nth-child(even) { border-left: 1px solid var(--line); }
      .settings-guide-details { padding: 0; overflow: hidden; }
      .settings-guide-details > summary { min-height: 54px; display: flex; align-items: center; gap: 12px; padding: 12px 16px; color: var(--ink); cursor: pointer; list-style: none; font-weight: 800; }
      .settings-guide-details > summary::-webkit-details-marker { display: none; }
      .settings-guide-details > summary span { margin-left: auto; color: var(--muted); font-size: 12px; font-weight: 650; }
      .settings-guide-details > summary::after { content: "›"; color: var(--gold-ink); font-size: 22px; line-height: 1; transform: rotate(90deg); }
      .settings-guide-details[open] > summary::after { transform: rotate(-90deg); }
      .settings-guide-details[open] > summary { border-bottom: 1px solid var(--line); }
      .settings-guide-body { padding: 15px 16px 16px; }
      .settings-guide { display: grid; gap: 13px; margin: 0; padding: 0; list-style: none; }
      .settings-guide li { display: grid; gap: 5px; }
      .settings-guide .settings-tag { justify-self: start; }
      .settings-guide span { color: var(--muted); font-size: 12.6px; line-height: 1.45; }
      @media (max-width: 760px) {
        .settings-hub { gap: 14px; }
        .settings-hub-head .lede { font-size: 15px; }
        .settings-account { grid-template-columns: 48px minmax(0,1fr); padding: 15px; }
        .settings-account-icon { width: 48px; height: 48px; }
        .settings-account .pill { grid-column: 2; justify-self: start; }
        .settings-layout, .settings-sections { grid-template-columns: 1fr; gap: 13px; }
        .settings-management { grid-column: 1; }
        .settings-management .settings-links { grid-template-columns: 1fr; }
        .settings-management .settings-link:nth-child(even) { border-left: 0; }
      }
    </style>
    <main id="main-content" tabindex="-1" class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg><span>/</span><span>Einstellungen</span></span>
      </div>
      <section class="page settings-hub">
        <div class="settings-hub-head">
          <h1>Einstellungen</h1>
          <p class="lede">Konto und Kommunikation für {{.Tenant.Address}}.</p>
        </div>
        <section class="panel settings-account">
          <span class="settings-account-icon"><svg viewBox="0 0 24 24"><path d="M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8z"/><path d="M4.5 21a7.5 7.5 0 0 1 15 0"/></svg></span>
          <div class="settings-account-copy"><h2>{{.SettingsDisplayName}}</h2><p>{{.Email}}</p></div>
          <span class="pill">{{.Role}}</span>
        </section>
        {{$mgmt := ""}}{{if .CanManageBuilding}}{{$mgmt = print $mgmt "bb"}}{{end}}{{if .CanManageUsers}}{{$mgmt = print $mgmt "uu"}}{{end}}{{if .CanManageDocuments}}{{$mgmt = print $mgmt "d"}}{{end}}{{if .CanManageHandovers}}{{$mgmt = print $mgmt "h"}}{{end}}{{if .CanViewAudit}}{{$mgmt = print $mgmt "a"}}{{end}}{{if .IsAdmin}}{{$mgmt = print $mgmt "p"}}{{end}}
        <div class="settings-layout">
        <div class="settings-sections">
          {{if .CanManageHomeIdentity}}<section class="panel settings-section settings-home-profile">
            <div class="settings-section-head"><h2>Mein Zuhause</h2><span class="settings-tag">Nur Sie</span><p>Anzeigename, Art und zugeordnete Wohnung.</p></div>
            <div class="settings-links">
              <a class="settings-link" href="{{.SettingsHomeURL}}"{{if .HomeIdentity.HasDisplayName}} data-home-identity="settings" aria-label="{{.HomeIdentity.AriaLabel}} bearbeiten"{{end}}>
                <span class="settings-link-icon"><svg viewBox="0 0 24 24"><path d="M4 21V9l8-6 8 6v12"/><path d="M9 21v-7h6v7"/></svg></span>
                <span class="settings-link-copy">{{if .HomeIdentity.HasDisplayName}}<strong data-home-display-name>{{.HomeIdentity.DisplayName}}</strong>{{if .HomeIdentity.HasUnit}}<span class="settings-home-unit" data-home-unit-label>{{.HomeIdentity.UnitLabel}}</span>{{end}}<span class="settings-home-purpose">Anzeigename und Zuordnung bearbeiten</span>{{else}}<strong>Mein Zuhause einrichten</strong><span>Name, Art und Zuordnung festlegen</span>{{end}}</span><span class="settings-link-arrow">›</span>
              </a>
              {{if .SettingsCanManageEnergyData}}<a class="settings-link secondary" href="/app/settings/energy-data">
                <span class="settings-link-icon"><svg viewBox="0 0 24 24"><path d="M12 3v18"/><path d="M7 7h8.5a3 3 0 0 1 0 6H9a3 3 0 0 0 0 6h8"/></svg></span>
                <span class="settings-link-copy"><strong>Energiedaten &amp; Datenschutz</strong><span>Gespeicherte Daten ansehen, exportieren oder löschen</span></span><span class="settings-link-arrow">›</span>
              </a>{{end}}
            </div>
          </section>{{end}}
          <section class="panel settings-section">
            <div class="settings-section-head"><h2>Mein Konto</h2><span class="settings-tag">Nur Sie</span><p>Persönliche Angaben und Erreichbarkeit. Sichtbar wird davon nur, was Sie freigeben.</p></div>
            <div class="settings-links">
              <a class="settings-link" href="/app/settings/profile">
                <span class="settings-link-icon"><svg viewBox="0 0 24 24"><path d="M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8z"/><path d="M4.5 21a7.5 7.5 0 0 1 15 0"/></svg></span>
                <span class="settings-link-copy"><strong>Profil</strong><span>Name, Telefon und Sichtbarkeit verwalten</span></span><span class="settings-link-arrow">›</span>
              </a>
            </div>
          </section>
          <section class="panel settings-section">
            <div class="settings-section-head"><h2>Kommunikation</h2><span class="settings-tag">Nur Sie</span><p>Was automatisch bei Ihnen ankommt. Gilt allein für Ihr Postfach.</p></div>
            <div class="settings-links">
              <a class="settings-link" href="/app/settings/notifications">
                <span class="settings-link-icon"><svg viewBox="0 0 24 24"><path d="M18 8a6 6 0 1 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9"/><path d="M10 21h4"/></svg></span>
                <span class="settings-link-copy"><strong>Benachrichtigungen</strong><span>{{.SettingsNotificationSummary}}</span></span><span class="settings-link-arrow">›</span>
              </a>
              {{if .HasCalendarFeedURL}}<a class="settings-link secondary" href="{{.CalendarFeedURL}}">
                <span class="settings-link-icon"><svg viewBox="0 0 24 24"><path d="M7 3v4M17 3v4"/><path d="M4.5 6h15v14h-15z"/><path d="M4.5 10h15"/><path d="M8 14h8M8 17h5"/></svg></span>
                <span class="settings-link-copy"><strong>Kalender-Abo</strong><span>Haustermine im eigenen Kalender</span></span><span class="settings-link-arrow">›</span>
              </a>{{end}}
            </div>
          </section>
          {{if or .CanManageUsers .CanManageBuilding .CanManageDocuments .CanManageHandovers .CanViewAudit .IsAdmin}}
          <section class="panel settings-section{{if gt (len $mgmt) 1}} settings-management{{end}}">
            <div class="settings-section-head">{{if or .CanManageUsers .CanManageBuilding .CanManageDocuments .CanManageHandovers .IsAdmin}}<h2>Verwaltung</h2><span class="settings-tag house">Verwaltungsrechte</span><p>Nur Bereiche, für die Sie berechtigt sind. Änderungen wirken für das ganze Haus.</p>{{else}}<h2>Verlauf</h2><span class="settings-tag read">Nur lesen</span><p>Eigene Änderungen nachvollziehen. Es wird nichts verändert.</p>{{end}}</div>
            <div class="settings-links">
              {{if .CanManageBuilding}}<a class="settings-link" href="/app/settings/building"><span class="settings-link-icon"><svg viewBox="0 0 24 24"><path d="M4 21V8l8-5 8 5v13"/><path d="M9 21v-7h6v7"/></svg></span><span class="settings-link-copy"><strong>Gebäude &amp; Einheiten</strong><span>Hausdaten und Einheiten pflegen</span></span><span class="settings-link-arrow">›</span></a>{{end}}
              {{if .CanManageUsers}}<a class="settings-link" href="/app/settings/users"><span class="settings-link-icon"><svg viewBox="0 0 24 24"><path d="M8.5 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6z"/><path d="M3.5 20a5 5 0 0 1 10 0"/><path d="M16 11.5a2.5 2.5 0 1 0 0-5"/></svg></span><span class="settings-link-copy"><strong>Benutzer &amp; Rechte</strong><span>Einladungen und Rollen verwalten</span></span><span class="settings-link-arrow">›</span></a>
              <a class="settings-link" href="/app/settings/parking-access"><span class="settings-link-icon"><svg viewBox="0 0 24 24"><path d="M5 16h14"/><path d="m7 16 1.5-5h7L17 16"/><path d="M7 16v3M17 16v3"/></svg></span><span class="settings-link-copy"><strong>Parkplatz-Zugriff</strong><span>Nutzung freigeben oder entziehen</span></span><span class="settings-link-arrow">›</span></a>{{end}}
              {{if .CanManageDocuments}}<a class="settings-link" href="/app/dokumente"><span class="settings-link-icon"><svg viewBox="0 0 24 24"><path d="M7 3h7l3 3v15H7z"/><path d="M14 3v4h4"/></svg></span><span class="settings-link-copy"><strong>Dokumente</strong><span>Unterlagen verwalten</span></span><span class="settings-link-arrow">›</span></a>{{end}}
              {{if .CanManageBuilding}}<a class="settings-link" href="/app/settings/data-export"><span class="settings-link-icon"><svg viewBox="0 0 24 24"><path d="M12 3v12"/><path d="m8 11 4 4 4-4"/><path d="M5 19h14"/></svg></span><span class="settings-link-copy"><strong>Datenübergabe</strong><span>Ausgewählte Rohdaten sicher weitergeben</span></span><span class="settings-link-arrow">›</span></a>{{end}}
              {{if .CanManageHandovers}}<a class="settings-link" href="/app/uebergaben"><span class="settings-link-icon"><svg viewBox="0 0 24 24"><path d="M7 4h10v16H7z"/><path d="M9.5 8h5M9.5 12h4"/></svg></span><span class="settings-link-copy"><strong>Übergaben</strong><span>Protokolle vorbereiten und ablegen</span></span><span class="settings-link-arrow">›</span></a>{{end}}
              {{if .CanViewAudit}}<a class="settings-link" href="/app/audit"><span class="settings-link-icon"><svg viewBox="0 0 24 24"><path d="M5 4h14v16H5z"/><path d="M8 8h8M8 12h8M8 16h5"/></svg></span><span class="settings-link-copy"><strong>Aktivitätsverlauf</strong><span>Änderungen nachvollziehen</span></span><span class="settings-link-arrow">›</span></a>{{end}}
              {{if .IsAdmin}}<a class="settings-link" href="/app/parking/settings"><span class="settings-link-icon"><svg viewBox="0 0 24 24"><path d="M5 16h14"/><path d="m7 16 1.5-5h7L17 16"/></svg></span><span class="settings-link-copy"><strong>Parkplatz-Abrechnung</strong><span>Tarife und Abrechnungswerte</span></span><span class="settings-link-arrow">›</span></a>{{end}}
            </div>
          </section>
          {{end}}
        </div>
        <aside class="settings-aside" aria-label="Hinweise zu den Einstellungen">
          <details class="panel compact settings-guide-details">
            <summary id="settings-guide-title">Berechtigungen erklärt <span>Bei Bedarf</span></summary>
            <div class="settings-guide-body">
            <ul class="settings-guide">
              <li><span class="settings-tag">Nur Sie</span><span>Wirkt allein auf Ihren Zugang. Andere im Haus merken davon nichts, solange Sie nichts freigeben.</span></li>
              {{if or .CanManageUsers .CanManageBuilding .CanManageDocuments .CanManageHandovers .IsAdmin}}<li><span class="settings-tag house">Verwaltungsrechte</span><span>Gilt für die ganze Liegenschaft. Jede Änderung wird im Verlauf festgehalten.</span></li>{{end}}
              {{if and .CanViewAudit (not (or .CanManageUsers .CanManageBuilding .CanManageDocuments .CanManageHandovers .IsAdmin))}}<li><span class="settings-tag read">Nur lesen</span><span>Reine Ansicht zum Nachvollziehen. Hier lässt sich nichts verändern.</span></li>{{end}}
            </ul>
            </div>
          </details>
        </aside>
        </div>
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "homeIdentitySettings"}}
{{template "appOpen" .}}
    <style>
      .home-identity-page { width: min(920px,100%); gap: 18px; }
      .home-identity-head { display: grid; gap: 6px; max-width: 760px; }
      .home-identity-head-unit { margin-top: -1px; color: var(--muted); font-size: 13px; font-weight: 650; line-height: 1.35; }
      .home-identity-head .lede { max-width: 680px; }
      .home-identity-map { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); overflow: hidden; padding: 0; }
      .home-identity-scope { min-height: 116px; display: grid; align-content: center; gap: 5px; padding: 18px 20px; border-right: 1px solid var(--line); }
      .home-identity-scope:last-child { border-right: 0; }
      .home-identity-scope.current { background: var(--panel-soft); }
      .home-identity-scope span { color: var(--gold-ink); font-size: 10.5px; font-weight: 850; letter-spacing: .08em; text-transform: uppercase; }
      .home-identity-scope strong { font-family: var(--font-serif); font-size: 21px; line-height: 1.2; overflow-wrap: anywhere; }
      .home-identity-scope small { color: var(--muted); font-size: 12px; line-height: 1.4; }
      .home-identity-scope .home-identity-unit-label { font-weight: 650; }
      .home-identity-scope .home-identity-scope-meta { margin-top: 1px; color: var(--soft); font-size: 11.5px; }
      .home-identity-form-card { display: grid; gap: 18px; padding: clamp(20px,3vw,30px); }
      .home-identity-form-head { display: grid; gap: 5px; }
      .home-identity-form-head h2 { font-size: 28px; }
      .home-identity-form-head p { color: var(--muted); font-size: 13.5px; line-height: 1.5; }
      .home-identity-form { display: grid; gap: 17px; }
      .home-identity-form > label > span { display: block; margin-bottom: 7px; color: #75652d; font-size: 11px; font-weight: 850; letter-spacing: .09em; text-transform: uppercase; }
      .home-identity-field > span, .home-identity-field > label > span { display: block; margin-bottom: 7px; color: #75652d; font-size: 11px; font-weight: 850; letter-spacing: .09em; text-transform: uppercase; }
      .home-identity-form input, .home-identity-form select { width: 100%; min-height: 50px; }
      .home-identity-readonly { min-height: 50px; display: flex; align-items: center; justify-content: space-between; gap: 12px; border: 1px solid var(--line); border-radius: var(--radius-xs); padding: 12px 14px; background: var(--panel-soft); }
      .home-identity-readonly strong { font-size: 15px; }
      .home-identity-readonly small, .home-identity-field-help { display: block; margin-top: 6px; color: var(--muted); font-size: 12px; font-weight: 600; line-height: 1.45; letter-spacing: normal; text-transform: none; }
      .home-identity-note { display: grid; grid-template-columns: 34px minmax(0,1fr); gap: 11px; align-items: start; border-top: 1px solid var(--line); padding-top: 17px; }
      .home-identity-note svg { width: 32px; height: 32px; padding: 7px; border-radius: 50%; color: var(--leaf); background: rgba(47,107,74,.1); fill: none; stroke: currentColor; stroke-width: 1.8; }
      .home-identity-note strong { display: block; font-size: 13.5px; }
      .home-identity-note p { margin-top: 3px; color: var(--muted); font-size: 12.5px; line-height: 1.45; }
      .home-identity-actions { display: flex; justify-content: flex-end; gap: 9px; }
      @media (max-width: 720px) {
        .home-identity-map { grid-template-columns: 1fr; }
        .home-identity-scope { min-height: 0; border-right: 0; border-bottom: 1px solid var(--line); padding: 15px 17px; }
        .home-identity-scope:last-child { border-bottom: 0; }
        .home-identity-actions { display: grid; grid-template-columns: 1fr; }
        .home-identity-actions .button { width: 100%; }
      }
    </style>
    <main id="main-content" tabindex="-1" class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg><span>/</span><a href="/app/settings">Einstellungen</a><span>/</span><span>Mein Zuhause</span></span>
        <div class="page-actions"><a class="button" href="{{.BackURL}}">{{.BackLabel}}</a></div>
      </div>
      <section class="page home-identity-page">
        <header class="home-identity-head" data-home-identity="editor-heading" aria-label="{{.HomeIdentity.AriaLabel}}">
          <span class="eyebrow">Mein Zuhause · Einstellungen</span>
          <h1 data-home-display-name>{{.Profile.HouseholdName}}</h1>
          {{if .HasHomeUnit}}<p class="home-identity-head-unit" data-home-unit-label>{{.HomeUnitLabel}}</p>{{end}}
          <p class="lede">Anzeigename, Art und Zuordnung an einem Ort.</p>
        </header>
        {{if .Saved}}<div class="message success">Der Anzeigename von „Mein Zuhause“ wurde gespeichert.</div>{{end}}
        {{if .Invalid}}<div class="message error">Bitte einen Namen und eine gültige Zuhause-Art auswählen.</div>{{end}}
        <section class="panel home-identity-map" aria-label="Namensbereiche">
          <div class="home-identity-scope"><span>Liegenschaft</span><strong>{{.HouseName}}</strong><small>{{if eq .HouseName .Tenant.Address}}Name und Adresse der Liegenschaft{{else}}{{.Tenant.Address}}{{end}}</small></div>
          <div class="home-identity-scope current" data-home-identity="editor-summary" aria-label="{{.HomeIdentity.AriaLabel}}"><span>Mein Zuhause</span><strong data-home-display-name>{{.Profile.HouseholdName}}</strong>{{if .HasHomeUnit}}<small class="home-identity-unit-label" data-home-unit-label>{{.HomeUnitLabel}}</small>{{end}}<small class="home-identity-scope-meta">{{.HomeTypeLabel}}{{if not .HasHomeUnit}} · Hausprofil{{end}}</small></div>
          <div class="home-identity-scope"><span>{{.OfficialUnitTitle}}</span><strong>{{.OfficialUnitSummary}}</strong><small>{{if .HasHomeUnit}}Diesem Hausprofil zugeordnet.{{else}}Legt fest, wem dieses Hausprofil gehört.{{end}}</small></div>
        </section>
        <section class="panel home-identity-form-card">
          <header class="home-identity-form-head"><h2>Anzeigename festlegen</h2><p>{{if .HasHomeUnit}}„{{.Profile.HouseholdName}}“ ist der freundliche Name. „{{.HomeUnitLabel}}“ bleibt die offizielle Einheit und wird hier eindeutig zugeordnet.{{else}}Der freundliche Name erscheint im Energie- und Wartungsbereich; die Stammdaten der Liegenschaft bleiben unverändert.{{end}}</p></header>
          <form class="home-identity-form" method="post" action="/app/settings/home">
            <input type="hidden" name="from" value="{{.From}}">
            <label><span>Anzeigename für „Mein Zuhause“</span><input type="text" name="household_name" value="{{.Profile.HouseholdName}}" placeholder="z. B. Penthouse" required maxlength="100"></label>
            {{if .HomeTypeLocked}}
              <div class="home-identity-field"><span>Art des Zuhauses</span><input type="hidden" name="home_type" value="{{.Profile.HomeType}}"><div class="home-identity-readonly"><strong>{{.HomeTypeLabel}}</strong><small>Durch die zugeordnete Wohneinheit festgelegt</small></div></div>
            {{else}}
              <label><span>Art des Zuhauses</span><select name="home_type" data-home-type-select aria-describedby="home-settings-type-explanation">
              <option value="apartment" data-label="Wohnung" data-description="Ein einzelner Haushalt in einem Mehrparteienhaus. Der Überblick konzentriert sich auf die Wohnung und ihre eigenen Geräte."{{if eq .Profile.HomeType "apartment"}} selected{{end}}>Wohnung</option>
              <option value="house" data-label="Einfamilienhaus" data-description="Ein Haushalt mit eigenem Gebäude. Haus-, Heiz- und Energietechnik können gemeinsam betrachtet werden."{{if eq .Profile.HomeType "house"}} selected{{end}}>Einfamilienhaus</option>
              <option value="community" data-label="Hausgemeinschaft" data-description="Mehrere Parteien und gemeinsam genutzte Anlagen. Der Überblick richtet sich an Eigentümergemeinschaft oder Hausverwaltung."{{if eq .Profile.HomeType "community"}} selected{{end}}>Hausgemeinschaft</option>
              </select></label>
            {{end}}
            {{if .HasHomeUnit}}
              <div class="home-identity-field"><span>Zugeordnete offizielle Wohnung</span><input type="hidden" name="unit_id" value="{{.HomeUnitID}}"><div class="home-identity-readonly"><strong>{{.HomeUnitLabel}}</strong><small>Diesem Hausprofil zugeordnet</small></div><small class="home-identity-field-help">Die Sichtbarkeit folgt dieser Wohnung; ausdrücklich freigeschaltete technische Betreuung bleibt möglich.</small></div>
            {{else if .HasUnitOptions}}<div class="home-identity-field" data-home-unit-field><label><span>Zugeordnete offizielle Wohnung</span><select name="unit_id" required aria-describedby="home-settings-unit-help">
              {{range .UnitOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}
            </select></label><small class="home-identity-field-help" id="home-settings-unit-help">Die Auswahl legt fest, welche Wohnung den Überblick sieht. Technische Freigaben bleiben separat.</small></div>{{else}}<input type="hidden" name="unit_id" value="">{{end}}
            <div class="home-type-explanation" id="home-settings-type-explanation" data-home-type-explanation role="status" aria-live="polite">
              <span class="home-type-explanation-icon" aria-hidden="true">⌂</span>
              <div>
                <strong data-home-type-label>{{.HomeTypeLabel}}</strong>
                <p data-home-type-copy>{{.HomeTypeDescription}}</p>
                <small>{{if .HomeTypeLocked}}Die Art ist mit der Wohnung verbunden.{{else}}Die Auswahl kann Geltungsbereich und Sichtbarkeit ändern.{{end}} „Nur beobachten“ bleibt unverändert.</small>
              </div>
            </div>
            <div class="home-identity-note">
              <svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="9"/><path d="M12 10v6M12 7h.01"/></svg>
              <div><strong>Was bleibt gleich?</strong><p>Name und Adresse der Liegenschaft{{if .HasHomeUnit}} sowie die offizielle Bezeichnung „{{.HomeUnitLabel}}“{{end}} werden hier nicht umbenannt.</p></div>
            </div>
            <div class="home-identity-actions"><a class="button quiet" href="{{.BackURL}}">Abbrechen</a><button class="button primary" type="submit">Änderungen speichern</button></div>
          </form>
        </section>
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "structuredExport"}}
{{template "appOpen" .}}
    <style>
      .raw-export { display: grid; gap: 18px; }
      .raw-export-head { display: grid; gap: 6px; max-width: 760px; }
      .raw-export-head .lede { max-width: 690px; }
      .raw-export-steps { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 10px; }
      .raw-export-step { display: grid; grid-template-columns: 30px minmax(0,1fr); gap: 9px; align-items: center; padding: 11px 12px; border: 1px solid var(--line); border-radius: var(--radius-xs); background: rgba(255,255,255,.55); color: var(--muted); font-size: 12.5px; font-weight: 700; }
      .raw-export-step b { width: 30px; height: 30px; display: grid; place-items: center; border-radius: 50%; background: var(--panel-soft); color: var(--gold-ink); font-family: var(--font-serif); font-size: 17px; }
      .raw-export-grid { display: grid; grid-template-columns: minmax(0,1.15fr) minmax(280px,.85fr); gap: 16px; align-items: start; }
      .raw-export-card { padding: 20px; }
      .raw-export-card-head { display: flex; align-items: start; justify-content: space-between; gap: 12px; margin-bottom: 16px; }
      .raw-export-card-head h2 { font-size: 23px; }
      .raw-export-card-head p { margin-top: 4px; color: var(--muted); font-size: 13px; }
      .raw-export-kicker { display: inline-flex; align-items: center; min-height: 28px; padding: 4px 9px; border-radius: 999px; background: rgba(47,107,74,.1); color: var(--leaf); font-size: 11px; font-weight: 800; letter-spacing: .04em; text-transform: uppercase; white-space: nowrap; }
      .raw-export-options { display: grid; gap: 10px; }
      .raw-export-option { position: relative; display: grid; grid-template-columns: 26px minmax(0,1fr) auto; gap: 12px; align-items: center; min-height: 78px; padding: 14px; border: 1px solid var(--line); border-radius: var(--radius-xs); background: #fff; cursor: pointer; text-transform: none; letter-spacing: normal; font-weight: 400; }
      .raw-export-option:hover { border-color: rgba(200,153,63,.55); background: rgba(200,153,63,.04); }
      .raw-export-option input { width: 20px; height: 20px; accent-color: var(--leaf); }
      .raw-export-option strong { display: block; color: var(--ink); font-family: var(--font-serif); font-size: 18px; line-height: 1.15; overflow-wrap: anywhere; }
      .raw-export-option small { display: block; margin-top: 3px; color: var(--muted); line-height: 1.35; overflow-wrap: anywhere; }
      .raw-export-count { min-width: 34px; height: 28px; display: grid; place-items: center; padding: 0 8px; border-radius: 999px; background: var(--panel-soft); color: var(--gold-ink); font-size: 12px; font-weight: 800; }
      .raw-export-actions { display: flex; justify-content: flex-end; margin-top: 16px; }
      .raw-export-guard { display: grid; gap: 14px; }
      .raw-export-guard-row { display: grid; grid-template-columns: 34px minmax(0,1fr); gap: 11px; align-items: start; }
      .raw-export-guard-row svg { width: 34px; height: 34px; padding: 7px; border-radius: 50%; background: rgba(200,153,63,.11); color: var(--gold-ink); fill: none; stroke: currentColor; stroke-width: 1.7; }
      .raw-export-guard-row strong { display: block; font-size: 14px; }
      .raw-export-guard-row span { display: block; margin-top: 2px; color: var(--muted); font-size: 12.5px; line-height: 1.45; }
      .raw-export-preview { grid-column: 1 / -1; padding: 20px; border-color: rgba(47,107,74,.25); }
      .raw-export-preview-top { display: flex; justify-content: space-between; gap: 16px; align-items: start; }
      .raw-export-preview h2 { font-size: 24px; }
      .raw-export-preview p { color: var(--muted); font-size: 13px; }
      .raw-export-summary { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 10px; margin: 16px 0; }
      .raw-export-metric { padding: 13px; border-radius: var(--radius-xs); background: var(--panel-soft); }
      .raw-export-metric strong { display: block; font-family: var(--font-serif); font-size: 24px; }
      .raw-export-metric span { color: var(--muted); font-size: 11.5px; }
      .raw-export-selection { display: grid; gap: 8px; margin-bottom: 16px; }
      .raw-export-selection-row { display: flex; justify-content: space-between; gap: 12px; padding: 10px 12px; border: 1px solid var(--line); border-radius: var(--radius-xs); font-size: 13px; }
      .raw-export-checksum { overflow-wrap: anywhere; font-family: ui-monospace,SFMono-Regular,Menlo,monospace; font-size: 11px !important; }
      .raw-export-download { display: flex; align-items: center; justify-content: space-between; gap: 15px; padding-top: 16px; border-top: 1px solid var(--line); }
      .raw-export-download strong { display: block; font-size: 14px; overflow-wrap: anywhere; }
      @media (max-width: 760px) {
        .raw-export { gap: 14px; }
        .raw-export-steps { grid-template-columns: 1fr; gap: 7px; }
        .raw-export-step { min-height: 48px; }
        .raw-export-grid { grid-template-columns: 1fr; }
        .raw-export-card { padding: 16px; }
        .raw-export-option { grid-template-columns: 24px minmax(0,1fr) auto; padding: 12px; }
        .raw-export-preview { grid-column: 1; }
        .raw-export-preview-top, .raw-export-download { display: grid; }
        .raw-export-summary { grid-template-columns: 1fr 1fr; }
        .raw-export-metric:last-child { grid-column: 1 / -1; }
        .raw-export-download .button { width: 100%; }
      }
    </style>
    <main id="main-content" tabindex="-1" class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg><span>/</span><a href="/app/settings">Einstellungen</a><span>/</span><span>Datenübergabe</span></span>
      </div>
      <section class="page raw-export">
        <header class="raw-export-head">
          <h1>Daten sicher weitergeben</h1>
          <p class="lede">Sie wählen bewusst aus. hausv.org bündelt nur diese Rohdaten mit Prüfdatei — ohne Buchungen zu erzeugen.</p>
        </header>
        <div class="raw-export-steps" aria-label="Ablauf">
          <div class="raw-export-step"><b>1</b><span>Datenbereiche wählen</span></div>
          <div class="raw-export-step"><b>2</b><span>Umfang prüfen</span></div>
          <div class="raw-export-step"><b>3</b><span>Paket herunterladen</span></div>
        </div>
        {{if .StructuredExportMsg}}<p class="flash {{if .StructuredExportOK}}ok{{end}}">{{.StructuredExportMsg}}</p>{{end}}
        <div class="raw-export-grid">
          <section class="panel raw-export-card">
            <div class="raw-export-card-head"><div><h2>Was soll ins Paket?</h2><p>Nichts ist vorausgewählt.</p></div><span class="raw-export-kicker">raw-v0</span></div>
            <form method="post" action="/app/settings/data-export/preview">
              <div class="raw-export-options">
                {{range .StructuredExportSources}}
                <label class="raw-export-option">
                  <input type="checkbox" name="source" value="{{.Value}}">
                  <span><strong>{{.Title}}</strong><small>{{.Description}}</small></span>
                  <span class="raw-export-count">{{.Count}}</span>
                </label>
                {{end}}
              </div>
              <div class="raw-export-actions"><button class="button primary" type="submit">Auswahl prüfen</button></div>
            </form>
          </section>
          <aside class="panel raw-export-card raw-export-guard">
            <div class="raw-export-card-head"><div><h2>Was Sie bekommen</h2><p>Ein prüfbares ZIP-Paket.</p></div></div>
            <div class="raw-export-guard-row"><svg viewBox="0 0 24 24"><path d="M7 3h7l3 3v15H7z"/><path d="M14 3v4h4"/></svg><div><strong>Lesbare CSV-Datei</strong><span>UTF-8, Semikolon, stabile Spalten.</span></div></div>
            <div class="raw-export-guard-row"><svg viewBox="0 0 24 24"><path d="m5 12 4 4L19 6"/></svg><div><strong>Manifest mit Prüfsumme</strong><span>Version, Zeitpunkt, Anzahl und SHA-256.</span></div></div>
            <div class="raw-export-guard-row"><svg viewBox="0 0 24 24"><path d="M12 3 5 6v5c0 4.5 2.8 8 7 10 4.2-2 7-5.5 7-10V6z"/><path d="M9 12h6"/></svg><div><strong>Keine Zielsystem-Zusage</strong><span>Kein BMD-/RZL-Format, keine Steuer- oder Buchungslogik.</span></div></div>
          </aside>
          {{with .StructuredExportPreview}}
          <section class="panel raw-export-preview">
            <div class="raw-export-preview-top"><div><span class="raw-export-kicker">Bereit</span><h2>Auswahl geprüft</h2><p>Erstellt {{.CreatedAt}} · nur für diesen Zugang · 15 Minuten verfügbar</p></div></div>
            <div class="raw-export-summary">
              <div class="raw-export-metric"><strong>{{.Accepted}}</strong><span>Datensätze im CSV</span></div>
              <div class="raw-export-metric"><strong>{{len .Sources}}</strong><span>gewählte Bereiche</span></div>
              <div class="raw-export-metric"><strong>{{.Rejected}}</strong><span>nicht übernommen</span></div>
            </div>
            <div class="raw-export-selection">
              {{range .Sources}}<div class="raw-export-selection-row"><strong>{{.Title}}</strong><span>{{.Count}} {{if eq .Count 1}}Datensatz{{else}}Datensätze{{end}}</span></div>{{end}}
            </div>
            <p class="raw-export-checksum">CSV SHA-256: {{.CSVChecksum}}</p>
            <div class="raw-export-download">
              <div><strong>{{.Filename}}</strong><p>Der Download wird im Aktivitätsverlauf protokolliert und ist einmalig.</p></div>
              <form method="post" action="/app/settings/data-export/download"><input type="hidden" name="preview_token" value="{{.Token}}"><button class="button primary" type="submit">ZIP herunterladen</button></form>
            </div>
          </section>
          {{end}}
        </div>
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "auditLog"}}
{{template "appOpen" .}}
    <style>
      /* Der Verlauf ist eine Nachweisfläche: Dichte und Lesbarkeit gehen vor
         Gestaltung. Auf breiten Schirmen stehen Zeit, Vorgang, Person und Objekt
         in eigenen Spalten; darunter fällt die Zeile auf eine Kompaktform mit
         einer Kontextzeile zurück. Der leere Zustand ist der Normalfall eines
         neuen Hauses und deshalb ein eigener Bereich statt eines Kastens. */
      .audit-screen .audit-help-disclosure { border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel); box-shadow: var(--shadow-panel); padding: 0 16px; }
      .audit-screen .audit-help-disclosure > summary { color: var(--ink); font-size: 14px; font-weight: 850; }
      .audit-screen .audit-help-disclosure .audit-notes { padding-bottom: 16px; }
      .audit-screen .audit-notes { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 16px 26px; margin: 4px 0 0; padding: 0; list-style: none; }
      .audit-screen .audit-notes > li { display: grid; gap: 9px; align-content: start; border-top: 2px solid var(--ink); padding-top: 12px; }
      .audit-screen .audit-notes h2 { font-family: var(--font-sans); font-size: 11.5px; font-weight: 800; letter-spacing: .11em; text-transform: uppercase; color: var(--gold-ink); }
      .audit-screen .audit-notes p { color: var(--muted); font-size: 12.6px; line-height: 1.5; }
      .audit-screen .audit-legend { display: grid; gap: 11px; margin: 0; padding: 0; list-style: none; }
      .audit-screen .audit-legend li { display: grid; grid-template-columns: 10px minmax(0,1fr); gap: 11px; align-items: start; }
      .audit-screen .audit-legend i { margin-top: 5px; width: 10px; height: 10px; border-radius: 50%; background: var(--gold); }
      .audit-screen .audit-legend i.add { background: var(--leaf); }
      .audit-screen .audit-legend i.danger { background: #9e2a2b; }
      .audit-screen .audit-legend strong { display: block; font-size: 13.5px; line-height: 1.25; }
      .audit-screen .audit-legend span { display: block; color: var(--muted); font-size: 12.5px; line-height: 1.42; }
      .audit-screen .audit-facts { display: grid; gap: 9px; margin: 0; padding: 0; list-style: none; }
      .audit-screen .audit-facts li { display: grid; gap: 2px; border-top: 1px solid var(--line); padding-top: 9px; }
      .audit-screen .audit-facts li:first-child { border-top: 0; padding-top: 0; }
      .audit-screen .audit-facts strong { font-size: 13.5px; }
      .audit-screen .audit-facts span { color: var(--muted); font-size: 12.5px; line-height: 1.42; }
      .audit-screen .audit-links { display: grid; }
      .audit-screen .audit-link { min-height: 44px; display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 10px; align-items: center; border-top: 1px solid var(--line); padding: 9px 0; color: inherit; text-decoration: none; }
      .audit-screen .audit-link:first-child { border-top: 0; padding-top: 0; }
      .audit-screen .audit-link strong { display: block; font-size: 13.5px; }
      .audit-screen .audit-link small { display: block; margin-top: 2px; color: var(--muted); font-size: 12.5px; line-height: 1.35; }
      .audit-screen .audit-link::after { content: "\203A"; color: var(--gold-ink); font-size: 21px; line-height: 1; }
      .audit-screen .audit-link:hover strong { color: var(--gold-ink); }
      .audit-screen .audit-blank { display: grid; grid-template-columns: minmax(0,1.42fr) minmax(272px,.88fr); gap: 16px; }
      .audit-screen .audit-blank-main { display: grid; align-content: center; gap: 20px; border: 1px dashed rgba(200,153,63,.45); border-radius: var(--radius-sm); background: rgba(255,254,251,.7); padding: clamp(22px,3.2vw,36px); }
      .audit-screen .audit-blank-lead { display: grid; justify-items: start; gap: 13px; }
      .audit-screen .audit-blank-icon { width: 52px; height: 52px; display: grid; place-items: center; border: 1px solid rgba(200,153,63,.3); border-radius: 12px; background: rgba(200,153,63,.1); color: var(--gold-ink); }
      .audit-screen .audit-blank-icon svg { width: 26px; height: 26px; fill: none; stroke: currentColor; stroke-width: 1.5; stroke-linecap: round; stroke-linejoin: round; }
      .audit-screen .audit-blank-main h2 { font-size: clamp(25px,3vw,31px); }
      .audit-screen .audit-blank-main p { max-width: 54ch; color: var(--muted); font-size: 15px; line-height: 1.55; }
      .audit-screen .audit-blank-actions { display: flex; flex-wrap: wrap; gap: 9px; margin-top: 3px; }
      .audit-screen .audit-blank-actions .button { min-height: 44px; }
      .audit-screen .audit-blank-side { display: grid; align-content: start; gap: 13px; border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel); padding: 20px; box-shadow: var(--shadow-panel); }
      .audit-screen .audit-blank-side h2 { font-family: var(--font-sans); font-size: 12px; font-weight: 800; letter-spacing: .12em; text-transform: uppercase; color: var(--gold-ink); }
      .audit-screen .audit-blank-notes { grid-column: 1 / -1; display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 14px; margin: 0; padding: 0; list-style: none; }
      .audit-screen .audit-blank-notes li { display: grid; gap: 5px; border-top: 2px solid var(--ink); padding-top: 11px; }
      .audit-screen .audit-blank-notes strong { font-size: 13.5px; }
      .audit-screen .audit-blank-notes span { color: var(--muted); font-size: 12.5px; line-height: 1.5; }
      .audit-screen .audit-no-result { display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 16px; align-items: center; border: 1px dashed rgba(200,153,63,.45); border-radius: var(--radius-sm); background: rgba(255,254,251,.7); padding: 26px 22px; }
      .audit-screen .audit-no-result h3 { font-size: 19px; }
      .audit-screen .audit-no-result .button { min-height: 44px; }
      .audit-screen .audit-result-note { margin-top: 5px; color: var(--muted); font-size: 13.5px; line-height: 1.5; }
      @media (min-width: 901px) {
        .audit-screen .audit-blank { min-height: max(420px, calc(100vh - 348px)); grid-template-rows: minmax(0,1fr) auto; }
      }
      /* Tablet: die Seitenleiste ist eingeklappt, die volle Breite steht der
         Tabelle zur Verfügung – Spalten bleiben deshalb bis 620px erhalten. */
      @media (min-width: 621px) and (max-width: 900px) {
        .audit-screen .audit-timeline { --audit-cols: 56px 12px minmax(0,1.3fr) minmax(0,.95fr) minmax(0,1fr) 16px; }
        .audit-screen .audit-columns { display: grid; }
        .audit-screen .audit-actor, .audit-screen .audit-object { display: block; }
        .audit-screen .audit-context { display: none; }
      }
      @media (max-width: 900px) {
        .audit-screen .audit-notes, .audit-screen .audit-blank { grid-template-columns: minmax(0,1fr); gap: 13px; }
        .audit-screen .audit-notes > li { padding-top: 11px; }
        .audit-screen .audit-blank-main { padding: 24px 18px; }
        .audit-screen .audit-blank-notes { grid-template-columns: minmax(0,1fr); gap: 12px; }
        .audit-screen .audit-blank-side { padding: 16px; }
        .audit-screen .audit-no-result { grid-template-columns: minmax(0,1fr); padding: 22px 16px; }
        .audit-screen .audit-no-result .button { width: 100%; justify-content: center; }
        .audit-screen .content-top .page-actions .button { min-height: 44px; }
      }
      @media (max-width: 560px) {
        .audit-screen .audit-blank-actions { display: grid; }
        .audit-screen .audit-blank-actions .button { width: 100%; justify-content: center; }
      }
    </style>
    <main id="main-content" tabindex="-1" class="app-main audit-screen">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M5 4h14v16H5z"/><path d="M8 8h8M8 12h8M8 16h5"/></svg><span>/</span><span>Verlauf</span></span>
        {{if .CanUseResidentAreas}}<div class="page-actions"><a class="button ghost" href="/app/settings">Einstellungen</a></div>{{end}}
      </div>
      <section class="page wide audit-page">
        <div class="audit-page-head">
          <h1>{{.AuditPageTitle}}</h1>
          <p class="lede">{{.AuditLede}}</p>
        </div>
        {{if .HasAnyEvents}}
        <div class="audit-stream">
          <details class="audit-filter-panel"{{if .AuditStats.HasActiveFilters}} open{{end}}>
            <summary>
              <span class="audit-overview"><svg viewBox="0 0 24 24"><path d="M5 7h14M5 12h14M5 17h14"/><path d="M2.5 7h.01M2.5 12h.01M2.5 17h.01"/></svg><strong>{{.AuditStats.TotalEvents}} {{if eq .AuditStats.TotalEvents 1}}Eintrag{{else}}Einträge{{end}}</strong><span>· {{.AuditStats.TodayCount}} heute</span></span>
              <span class="audit-filter-trigger"><svg viewBox="0 0 24 24"><path d="M4 5h16l-6 7v6l-4 2v-8z"/></svg>Filtern{{if .AuditStats.HasActiveFilters}} <span class="chip">{{len .AuditStats.ActiveFilters}}</span>{{end}}</span>
            </summary>
            <div class="audit-filter-content">
              <div class="audit-filter-content-head"><strong>Filter</strong>{{if .AuditIsFull}}<span>{{.AuditStats.ActorCount}} sichtbare Personen</span>{{end}}</div>
              <form class="filter-form audit-filter" method="get" action="/app/audit">
                <label for="audit-action">Art der Änderung
                  <select id="audit-action" name="action">
                    {{range .ActionOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}
                  </select>
                </label>
                <label for="audit-search">Suche
                  <input id="audit-search" type="search" name="q" value="{{.SearchQuery}}" placeholder="Vorgang, Dokument oder Person">
                </label>
                <button class="button primary" type="submit">Ergebnisse zeigen</button>
              </form>
              {{if .AuditStats.HasActiveFilters}}
                <div class="audit-active-filters" aria-label="Aktive Filter">
                  {{range .AuditStats.ActiveFilters}}<span class="chip"><strong>{{.Label}}:</strong> {{.Value}}</span>{{end}}
                </div>
                <div class="audit-filter-actions"><a class="button ghost" href="/app/audit">Filter zurücksetzen</a></div>
              {{end}}
            </div>
          </details>
          {{if .HasEvents}}
            <div class="audit-timeline" aria-label="Aktivitätsverlauf">
              <div class="audit-columns" aria-hidden="true"><span>Zeit</span><span></span><span>Vorgang</span><span>Person</span><span>Objekt</span><span></span></div>
              {{range .Events}}
                {{if .ShowDateHeader}}<h2 class="audit-day">{{.DateHeader}}</h2>{{end}}
                {{if .HasDetails}}
                  <details class="audit-event audit-{{.ActionTone}}">
                    <summary class="audit-row">
                      <time class="audit-time" datetime="{{.AtISO}}" aria-label="{{.At}}"><strong>{{.AtTime}}</strong></time>
                      <span class="audit-marker" aria-label="{{.ToneLabel}}"></span>
                      <span class="audit-main"><strong class="audit-title">{{.DisplayTitle}}</strong>{{if .HasContext}}<span class="audit-context">{{.Context}}</span>{{end}}</span>
                      <span class="audit-actor">{{if .ActorLabel}}{{.ActorLabel}}{{else}}<span class="audit-empty-cell">–</span>{{end}}</span>
                      <span class="audit-object">{{if .HasObject}}{{.ObjectLabel}}{{else}}<span class="audit-empty-cell">–</span>{{end}}</span>
                      <span class="audit-row-chevron" aria-hidden="true">›</span><span class="sr-only">Details zu {{.DisplayTitle}}</span>
                    </summary>
                    <dl class="audit-detail-list">{{range .Details}}<div class="audit-detail-row"><dt>{{.Key}}</dt><dd>{{.Value}}</dd></div>{{end}}</dl>
                  </details>
                {{else}}
                  <article class="audit-event audit-row audit-{{.ActionTone}}">
                    <time class="audit-time" datetime="{{.AtISO}}" aria-label="{{.At}}"><strong>{{.AtTime}}</strong></time>
                    <span class="audit-marker" aria-label="{{.ToneLabel}}"></span>
                    <span class="audit-main"><strong class="audit-title">{{.DisplayTitle}}</strong>{{if .HasContext}}<span class="audit-context">{{.Context}}</span>{{end}}</span>
                    <span class="audit-actor">{{if .ActorLabel}}{{.ActorLabel}}{{else}}<span class="audit-empty-cell">–</span>{{end}}</span>
                    <span class="audit-object">{{if .HasObject}}{{.ObjectLabel}}{{else}}<span class="audit-empty-cell">–</span>{{end}}</span>
                  </article>
                {{end}}
              {{end}}
            </div>
          {{else}}
            <div class="audit-no-result">
              <div>
                <h3>Kein Eintrag passt zu dieser Auswahl</h3>
                <p class="audit-result-note">Wählen Sie eine andere Art der Änderung oder einen anderen Suchbegriff.</p>
              </div>
              <a class="button" href="/app/audit">Filter zurücksetzen</a>
            </div>
          {{end}}
        </div>
        <details class="audit-help-disclosure guide-disclosure">
          <summary><strong>Einträge verstehen</strong></summary>
          <ul class="audit-notes">
          <li>
            <h2>Lesehilfe</h2>
            <ul class="audit-legend">
              <li><i class="add" aria-hidden="true"></i><div><strong>Angelegt</strong><span>Ein Eintrag, eine Freigabe oder ein Zugang ist neu entstanden.</span></div></li>
              <li><i aria-hidden="true"></i><div><strong>Geändert oder angesehen</strong><span>Bestehendes wurde bearbeitet, geöffnet oder heruntergeladen.</span></div></li>
              <li><i class="danger" aria-hidden="true"></i><div><strong>Entfernt</strong><span>Etwas wurde gelöscht oder ein Zugriff wurde entzogen.</span></div></li>
            </ul>
            <p>Zeitangaben in Ortszeit. Zeilen mit Pfeil lassen sich für die technischen Details aufklappen.</p>
          </li>
          <li>
            <h2>Umfang</h2>
            <ul class="audit-facts">
              {{if .AuditIsFull}}
                <li><strong>Ganzes Haus</strong><span>Änderungen und Zugriffe aller Zugänge dieser Liegenschaft.</span></li>
                <li><strong>Bis zu 500 Einträge</strong><span>Angezeigt werden die jüngsten Vorgänge, neueste zuerst.</span></li>
              {{else}}
                <li><strong>Ihr Ausschnitt</strong><span>Eigene Vorgänge und alles, worauf Sie aktuell Zugriff haben.</span></li>
                <li><strong>Folgt der Berechtigung</strong><span>Endet ein Zugriff, verschwindet der zugehörige Verlauf hier ebenfalls.</span></li>
              {{end}}
              <li><strong>Unveränderlich</strong><span>Einträge lassen sich hier weder bearbeiten noch löschen.</span></li>
            </ul>
          </li>
          {{if .CanUseResidentAreas}}<li>
            <h2>Weiter im Portal</h2>
            <div class="audit-links">
              {{if .CanManageUsers}}<a class="audit-link" href="/app/settings/users"><span><strong>Benutzer &amp; Rechte</strong><small>Rollen prüfen und Zugänge deaktivieren.</small></span></a>{{end}}
              <a class="audit-link" href="/app/settings"><span><strong>Einstellungen</strong><small>Konto, Kommunikation und Verwaltungsbereiche.</small></span></a>
            </div>
          </li>{{end}}
          </ul>
        </details>
        {{else}}
        <section class="audit-blank" aria-labelledby="audit-blank-title">
          <div class="audit-blank-main">
            <div class="audit-blank-lead">
              <span class="audit-blank-icon" aria-hidden="true"><svg viewBox="0 0 24 24"><path d="M5 4h14v16H5z"/><path d="M8 8h8M8 12h5"/><circle cx="14.5" cy="15.5" r="3"/><path d="m16.8 17.8 2.4 2.4"/></svg></span>
              <h2 id="audit-blank-title">{{.EventsEmpty.Title}}</h2>
              {{if .AuditIsFull}}<p>Sobald jemand sich anmeldet, eine Unterlage öffnet, eine Rolle ändert oder ein Anliegen bearbeitet, steht der Vorgang hier mit Zeitpunkt, Person und Objekt. Der Verlauf ist die Nachweisspur des Hauses und lässt sich nicht nachträglich ändern.</p>{{else}}<p>Sobald Sie sich anmelden, eine Unterlage öffnen oder ein Anliegen bearbeiten, steht der Vorgang hier mit Zeitpunkt und Objekt. Sie sehen ausschließlich Ihren eigenen Ausschnitt.</p>{{end}}
            </div>
            {{if .CanUseResidentAreas}}<div class="audit-blank-actions">
              {{if .CanManageUsers}}<a class="button primary" href="/app/settings/users">Benutzer &amp; Rechte</a>{{end}}
              <a class="button" href="/app/settings">Zu den Einstellungen</a>
            </div>{{end}}
          </div>
          <aside class="audit-blank-side" aria-labelledby="audit-blank-side-title">
            <h2 id="audit-blank-side-title">Was festgehalten wird</h2>
            <ul class="audit-facts">
              {{if .AuditIsFull}}
                <li><strong>Anmeldungen</strong><span>Wer sich wann und mit welchem Verfahren angemeldet hat.</span></li>
                <li><strong>Zugänge &amp; Rollen</strong><span>Einladungen, Rollenwechsel und entzogene Berechtigungen.</span></li>
                <li><strong>Unterlagen</strong><span>Hochladen, Ersetzen, Ansehen und Herunterladen von Dokumenten.</span></li>
                <li><strong>Entscheidungen</strong><span>Abstimmungen, Anliegen, Termine und Übergaben.</span></li>
                <li><strong>Hausdaten</strong><span>Änderungen an Gebäude, Einheiten und Zahlungsstatus.</span></li>
              {{else}}
                <li><strong>Ihre Anmeldungen</strong><span>Zeitpunkt und Verfahren jeder Anmeldung mit Ihrem Zugang.</span></li>
                <li><strong>Ihre Vorgänge</strong><span>Anliegen, Nachrichten und Unterlagen, die Sie betreffen.</span></li>
                <li><strong>Freigaben</strong><span>Was für Sie freigegeben oder wieder entzogen wurde.</span></li>
              {{end}}
            </ul>
          </aside>
          <ul class="audit-blank-notes">
            <li><strong>Unveränderlich</strong><span>Einträge werden angehängt, nie überschrieben. Auch die Verwaltung kann sie hier nicht entfernen.</span></li>
            <li><strong>Sparsam</strong><span>Festgehalten wird der Vorgang selbst – Zeitpunkt, Person, Objekt –, nicht der Inhalt.</span></li>
            <li><strong>Durchsuchbar</strong><span>Ab dem ersten Eintrag stehen Filter nach Art der Änderung und die Suche bereit.</span></li>
          </ul>
        </section>
        {{end}}
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "ebInterfaceImport"}}
{{template "appOpen" .}}
    <style>
      .invoice-import .page { gap: 18px; }
      .invoice-import .content-top .button { min-height: 44px; }
      .invoice-import .page-intro { display: grid; gap: 6px; }
      .invoice-import .page-intro .lede { max-width: 720px; }
      .invoice-import .flow-strip { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 9px; }
      .invoice-import .flow-step { display: grid; grid-template-columns: 34px minmax(0,1fr); gap: 10px; align-items: center; min-height: 64px; padding: 11px 13px; border: 1px solid var(--line); border-radius: 11px; background: var(--panel-soft); }
      .invoice-import .flow-step > span { width: 32px; height: 32px; display: grid; place-items: center; border-radius: 50%; background: var(--ink); color: #fff; font-weight: 850; }
      .invoice-import .flow-step strong, .invoice-import .flow-step small { display: block; }
      .invoice-import .flow-step small { margin-top: 2px; color: var(--muted); line-height: 1.35; }
      .invoice-import .import-panel { display: grid; gap: 15px; }
      .invoice-import .import-panel-head { display: flex; align-items: start; justify-content: space-between; gap: 14px; }
      .invoice-import .import-panel-head h2 { margin: 0 0 4px; }
      .invoice-import .upload-form { display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 12px; align-items: end; padding: 15px; border: 1px dashed var(--gold); border-radius: 11px; background: #fffefb; }
      .invoice-import .upload-copy { display: grid; gap: 7px; }
      .invoice-import .upload-copy label { font-weight: 850; }
      .invoice-import .upload-copy input[type=file] { width: 100%; min-height: 46px; padding: 8px; border: 1px solid var(--line); border-radius: 8px; background: #fff; }
      .invoice-import .privacy-note { display: flex; gap: 8px; align-items: start; margin: 0; color: var(--muted); font-size: 13px; line-height: 1.45; }
      .invoice-import .privacy-note::before { content: "✓"; flex: 0 0 auto; color: var(--gold-ink); font-weight: 900; }
      .invoice-import .profile-chip { display: inline-flex; min-height: 34px; align-items: center; padding: 0 10px; border: 1px solid var(--line); border-radius: 999px; background: var(--panel-soft); color: var(--gold-ink); font-size: 12px; font-weight: 850; white-space: nowrap; }
      .invoice-import .preview-actions { display: flex; align-items: center; justify-content: flex-end; gap: 8px; }
      .invoice-import .preview-actions .button { min-height: 44px; }
      .invoice-import .invoice-hero { display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 16px; align-items: end; padding: 18px; border: 1px solid var(--line); border-radius: 12px; background: linear-gradient(135deg,#fffefb,var(--panel-soft)); }
      .invoice-import .invoice-hero .kicker { margin-bottom: 5px; }
      .invoice-import .invoice-number { margin: 0; font-family: var(--font-serif); font-size: clamp(25px,4vw,38px); line-height: 1.05; overflow-wrap: anywhere; }
      .invoice-import .invoice-amount { font-family: var(--font-serif); font-size: clamp(25px,4vw,38px); font-weight: 800; line-height: 1; white-space: nowrap; }
      .invoice-import .invoice-parties { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 9px; }
      .invoice-import .party-card, .invoice-import .date-card { display: grid; gap: 3px; padding: 12px 13px; border: 1px solid var(--line); border-radius: 10px; background: #fff; }
      .invoice-import .party-card span, .invoice-import .date-card span { color: var(--muted); font-size: 12px; font-weight: 750; }
      .invoice-import .party-card strong { overflow-wrap: anywhere; }
      .invoice-import .invoice-dates { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 9px; }
      .invoice-import .error-list { display: grid; gap: 7px; margin: 0; padding: 0; list-style: none; }
      .invoice-import .error-list li { min-height: 44px; display: flex; align-items: center; gap: 9px; padding: 9px 11px; border: 1px solid #e5c6bc; border-radius: 9px; background: #fff6f2; color: #743e30; font-weight: 700; }
      .invoice-import .error-list li::before { content: "!"; width: 24px; height: 24px; flex: 0 0 auto; display: grid; place-items: center; border-radius: 50%; background: #874331; color: #fff; font-size: 12px; font-weight: 900; }
      .invoice-import .store-bar { display: flex; align-items: center; justify-content: space-between; gap: 14px; padding-top: 2px; }
      .invoice-import .store-bar .mini { max-width: 650px; }
      @media (max-width: 720px) {
        .invoice-import .flow-strip, .invoice-import .invoice-parties { grid-template-columns: 1fr; }
        .invoice-import .upload-form { grid-template-columns: 1fr; }
        .invoice-import .upload-form .button { width: 100%; min-height: 48px; }
        .invoice-import .import-panel-head, .invoice-import .store-bar { align-items: stretch; flex-direction: column; }
        .invoice-import .invoice-hero { grid-template-columns: 1fr; align-items: start; }
        .invoice-import .invoice-amount { white-space: normal; }
        .invoice-import .invoice-dates { grid-template-columns: 1fr; }
        .invoice-import .store-bar .button { width: 100%; min-height: 48px; }
      }
    </style>
    <main id="main-content" tabindex="-1" class="app-main invoice-import">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M7 3h7l3 3v15H7z"/><path d="M14 3v4h4"/><path d="M9 13h6M9 17h6"/></svg><span>/</span><span>Dokumente</span><span>/</span><span>E-Rechnung</span></span>
        <div class="page-actions"><a class="button" href="/app/dokumente">Zur Ablage</a></div>
      </div>
      <section class="page">
        <div class="page-intro">
          <h1>E-Rechnung einlesen</h1>
          <p class="lede">Rechnung zuerst prüfen, dann unverändert und nur für die Verwaltung ablegen.</p>
        </div>
        <div class="flow-strip" aria-label="Ablauf">
          <div class="flow-step"><span>1</span><div><strong>Prüfen</strong><small>Profil und wichtigste Rechnungsdaten ansehen</small></div></div>
          <div class="flow-step"><span>2</span><div><strong>Ablegen</strong><small>Original-XML geschützt in Dokumente speichern</small></div></div>
        </div>
        {{if .EBInterfaceImportMsg}}<p class="flash {{if .EBInterfaceImportOK}}ok{{end}}">{{.EBInterfaceImportMsg}}</p>{{end}}
        {{if not .EBInterfacePreview}}<section class="panel import-panel">
          <div class="import-panel-head">
            <div><div class="kicker">Neue Rechnung</div><h2>XML auswählen</h2><p class="muted">Für ebInterface 5.0 und 6.0.</p></div>
          </div>
          <form class="upload-form" method="post" action="/app/dokumente/rechnungen/import/preview" enctype="multipart/form-data">
            <div class="upload-copy">
              <label for="invoice-file">ebInterface-Datei</label>
              <input id="invoice-file" type="file" name="invoice_file" accept=".xml,application/xml,text/xml" required>
              <span class="mini">XML bis {{.MaxEBInterfaceImportSize}}</span>
            </div>
            <button class="button primary" type="submit">Vorschau erstellen</button>
          </form>
          <p class="privacy-note">Die Vorschau bleibt höchstens 15 Minuten im Arbeitsspeicher und wird nicht an externe Prüfdienste gesendet.</p>
        </section>{{end}}

        {{with .EBInterfacePreview}}
          <section class="panel import-panel" id="preview">
            <div class="import-panel-head">
              <div><div class="kicker">Vorschau</div><h2>{{.Filename}}</h2><p class="muted">Geprüft {{.CreatedAt}} · noch nicht abgelegt</p></div>
              <div class="preview-actions"><span class="profile-chip">ebInterface {{.SourceVersion}}</span><a class="button small" href="/app/dokumente/rechnungen/import">Andere Datei</a></div>
            </div>
            {{if .AlreadyStored}}<p class="flash ok">Diese Datei wurde bereits abgelegt. Ein zweites Dokument ist gesperrt.</p>{{end}}
            {{if .ErrorLabels}}
              <ul class="error-list">
                {{range .ErrorLabels}}<li>{{.}}</li>{{end}}
              </ul>
            {{else}}
              <div class="invoice-hero">
                <div><div class="kicker">Rechnung</div><p class="invoice-number">{{.InvoiceNumber}}</p></div>
                <strong class="invoice-amount">{{.Amount}}</strong>
              </div>
              <div class="invoice-parties">
                <div class="party-card"><span>Von</span><strong>{{.IssuerName}}</strong></div>
                <div class="party-card"><span>An</span><strong>{{.RecipientName}}</strong></div>
              </div>
              <div class="invoice-dates">
                <div class="date-card"><span>Rechnungsdatum</span><strong>{{.IssueDate}}</strong></div>
                <div class="date-card"><span>Fällig</span><strong>{{.DueDate}}</strong></div>
                <div class="date-card"><span>Leistungszeitraum</span><strong>{{.ServicePeriod}}</strong></div>
              </div>
            {{end}}
            <form class="store-bar" method="post" action="/app/dokumente/rechnungen/import/store">
              <input type="hidden" name="preview_token" value="{{.Token}}">
              <span class="mini">Gespeichert wird die unveränderte XML als „Abrechnung“. Sie bleibt ausschließlich für die Verwaltung sichtbar; es wird nichts gebucht oder bezahlt.</span>
              {{if .CanStore}}<button class="button primary" type="submit">Geschützt ablegen</button>{{else}}<button class="button" type="button" disabled>Ablage nicht möglich</button>{{end}}
            </form>
          </section>
        {{end}}
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "paymentImport"}}
{{template "appOpen" .}}
    <style>
      .payment-import .page { gap: 18px; }
      .payment-import .page-intro { display: grid; gap: 6px; }
      .payment-import .page-intro .lede { max-width: 720px; }
      .payment-import .import-steps { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 9px; }
      .payment-import .import-step { display: grid; grid-template-columns: 34px minmax(0,1fr); gap: 10px; align-items: center; padding: 12px; border: 1px solid var(--line); border-radius: 10px; background: var(--panel-soft); }
      .payment-import .import-step > span { width: 32px; height: 32px; display: grid; place-items: center; border-radius: 50%; background: var(--ink); color: #fff; font-weight: 850; }
      .payment-import .import-step strong, .payment-import .import-step small { display: block; }
      .payment-import .import-step small { margin-top: 2px; color: var(--muted); line-height: 1.35; }
      .payment-import .import-panel { display: grid; gap: 16px; }
      .payment-import .import-panel-head { display: flex; gap: 14px; align-items: start; justify-content: space-between; }
      .payment-import .import-panel-head h2 { margin: 0 0 4px; }
      .payment-import .period-form { display: flex; gap: 8px; align-items: end; }
      .payment-import .period-form label { min-width: 165px; }
      .payment-import .reference-disclosure { border: 1px solid var(--line); border-radius: 10px; background: var(--panel-soft); overflow: clip; }
      .payment-import .reference-disclosure > summary { min-height: 48px; display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 0 13px; cursor: pointer; font-weight: 800; }
      .payment-import .reference-disclosure > summary span { color: var(--muted); font-size: 13px; }
      .payment-import .reference-list { display: grid; border-top: 1px solid var(--line); }
      .payment-import .reference-row { display: grid; grid-template-columns: minmax(120px,.7fr) minmax(220px,1.3fr); gap: 12px; align-items: center; padding: 10px 13px; }
      .payment-import .reference-row + .reference-row { border-top: 1px solid var(--line); }
      .payment-import .reference-row code { overflow-wrap: anywhere; color: var(--gold-ink); font-size: 12.5px; }
      .payment-import .upload-form { display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 12px; align-items: end; padding: 15px; border: 1px dashed var(--gold); border-radius: 11px; background: #fffefb; }
      .payment-import .upload-copy { display: grid; gap: 7px; }
      .payment-import .upload-copy label { font-weight: 850; }
      .payment-import .upload-copy .file-control { min-height: 46px; border-style: dashed; }
      .payment-import .privacy-note { display: flex; gap: 8px; align-items: start; margin: 0; color: var(--muted); font-size: 13px; line-height: 1.45; }
      .payment-import .privacy-note::before { content: "✓"; flex: 0 0 auto; color: var(--gold-ink); font-weight: 900; }
      .payment-import .preview-head { display: flex; gap: 14px; justify-content: space-between; align-items: start; }
      .payment-import .preview-head h2 { margin: 0 0 4px; }
      .payment-import .profile-chip { display: inline-flex; min-height: 34px; align-items: center; padding: 0 10px; border-radius: 999px; background: var(--panel-soft); border: 1px solid var(--line); color: var(--gold-ink); font-size: 12px; font-weight: 850; }
      .payment-import .preview-metrics { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 8px; }
      .payment-import .preview-metric { display: grid; gap: 2px; padding: 11px 12px; border: 1px solid var(--line); border-radius: 9px; background: var(--panel-soft); }
      .payment-import .preview-metric strong { font-family: var(--font-serif); font-size: 24px; }
      .payment-import .preview-metric span { color: var(--muted); font-size: 12.5px; font-weight: 750; }
      .payment-import .import-rows { display: grid; gap: 7px; }
      .payment-import .import-row { display: grid; grid-template-columns: 96px minmax(150px,1.2fr) minmax(120px,.8fr) 100px minmax(180px,1.2fr); gap: 10px; align-items: center; padding: 11px 12px; border: 1px solid var(--line); border-radius: 9px; background: #fffefb; }
      .payment-import .decision { width: fit-content; display: inline-flex; min-height: 30px; align-items: center; padding: 0 9px; border-radius: 999px; font-size: 11.5px; font-weight: 900; text-transform: uppercase; letter-spacing: .025em; }
      .payment-import .decision.ok { background: #e8f1e6; color: #315c32; }
      .payment-import .decision.warn { background: #fff1ce; color: #745511; }
      .payment-import .decision.danger { background: #f8e5df; color: #874331; }
      .payment-import .row-reference { overflow-wrap: anywhere; font-size: 12px; font-weight: 800; color: var(--gold-ink); }
      .payment-import .row-unit strong, .payment-import .row-unit span { display: block; }
      .payment-import .row-unit span, .payment-import .row-reason { color: var(--muted); font-size: 12.5px; line-height: 1.4; }
      .payment-import .row-amount { font-weight: 850; text-align: right; white-space: nowrap; }
      .payment-import .apply-bar { display: flex; gap: 13px; align-items: center; justify-content: flex-end; padding-top: 2px; }
      .payment-import .apply-bar .mini { margin-right: auto; max-width: 620px; }
      @media (max-width: 900px) {
        .payment-import .import-row { grid-template-columns: 90px minmax(150px,1fr) minmax(120px,.8fr) 90px; }
        .payment-import .row-reason { grid-column: 2 / -1; }
      }
      @media (max-width: 700px) {
        .payment-import .page-intro h1 { font-size: clamp(38px,10vw,46px); line-height: .98; }
        .payment-import .import-steps { grid-template-columns: 1fr; }
        .payment-import .import-step { min-height: 64px; }
        .payment-import .import-panel-head, .payment-import .preview-head { display: grid; }
        .payment-import .period-form { width: 100%; display: grid; grid-template-columns: minmax(0,1fr) auto; }
        .payment-import .period-form label { min-width: 0; }
        .payment-import .period-form input, .payment-import .period-form .button, .payment-import .content-top .page-actions .button { min-height: 44px; }
        .payment-import .reference-row { grid-template-columns: 1fr; gap: 3px; }
        .payment-import .upload-form { grid-template-columns: 1fr; }
        .payment-import .upload-form .button { width: 100%; min-height: 46px; }
        .payment-import .preview-metrics { grid-template-columns: repeat(3,minmax(0,1fr)); }
        .payment-import .preview-metric { padding: 9px; }
        .payment-import .preview-metric strong { font-size: 21px; }
        .payment-import .import-row { grid-template-columns: minmax(0,1fr) auto; gap: 7px 10px; }
        .payment-import .decision { grid-column: 1; }
        .payment-import .row-amount { grid-column: 2; grid-row: 1; }
        .payment-import .row-reference, .payment-import .row-unit, .payment-import .row-reason { grid-column: 1 / -1; }
        .payment-import .row-unit { display: flex; gap: 8px; align-items: baseline; }
        .payment-import .apply-bar { position: static; display: grid; padding: 9px; border: 1px solid var(--line); border-radius: 10px; background: var(--panel-soft); }
        .payment-import .apply-bar .mini { margin: 0; }
        .payment-import .apply-bar .button { width: 100%; min-height: 46px; }
      }
    </style>
    <script src="/assets/attachments.js?v={{.AssetVersion}}" defer></script>
    <main id="main-content" tabindex="-1" class="app-main payment-import">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg><span>/</span><a href="/app/settings">Einstellungen</a><span>/</span><span>Bankdatei</span></span>
        <div class="page-actions"><a class="button ghost" href="/app/settings/building#units">Zu den Einheiten</a></div>
      </div>
      <section class="page wide">
        <div class="page-intro">
          <div class="kicker">Zahlungsstatus</div>
          <h1>Zahlungen aus Bankdatei</h1>
          <p class="lede">Erst prüfen, dann übernehmen. Nur eindeutige Zahlungsreferenzen ändern einen Status.</p>
        </div>
        <div class="import-steps" aria-label="Ablauf">
          <div class="import-step"><span>1</span><div><strong>Monat wählen</strong><small>Referenzen gelten für diesen Zeitraum.</small></div></div>
          <div class="import-step"><span>2</span><div><strong>Datei prüfen</strong><small>Treffer erscheinen zuerst als Vorschau.</small></div></div>
          <div class="import-step"><span>3</span><div><strong>Sicher übernehmen</strong><small>Nur eindeutige Treffer werden gespeichert.</small></div></div>
        </div>
        {{if .PaymentImportMsg}}<p class="flash {{if .PaymentImportOK}}ok{{end}}">{{.PaymentImportMsg}}</p>{{end}}

        <section class="panel import-panel">
          <div class="import-panel-head">
            <div><h2>1. Zeitraum und Referenzen</h2><p class="muted">Der Monat bestimmt, welche Referenz zu welcher Einheit gehört.</p></div>
            <form class="period-form" method="get" action="/app/settings/payments/import">
              <label>Monat<input type="month" name="period" value="{{.PaymentImportPeriod}}" min="2000-01" max="2100-12" required></label>
              <button class="button" type="submit">Anzeigen</button>
            </form>
          </div>
          {{if .HasPaymentImportUnits}}
            <details class="reference-disclosure">
              <summary><strong>{{len .PaymentImportReferences}} Zahlungsreferenzen</strong><span>Bei Bedarf anzeigen</span></summary>
              <div class="reference-list">
                {{range .PaymentImportReferences}}<div class="reference-row"><strong>{{.Label}}</strong><code>{{.Reference}}</code></div>{{end}}
              </div>
            </details>
          {{else}}
            <p class="muted">Für dieses Haus sind noch keine Einheiten angelegt.</p>
          {{end}}
          <form class="upload-form" method="post" action="/app/settings/payments/import/preview" enctype="multipart/form-data">
            <input type="hidden" name="period" value="{{.PaymentImportPeriod}}">
            <div class="upload-copy">
              <label for="camt-file">2. camt.053-Datei auswählen</label>
              <span class="file-control"><span>Datei auswählen</span><input id="camt-file" type="file" name="camt_file" accept=".xml,application/xml,text/xml" required></span>
              <span class="mini">XML bis {{.MaxCAMTImportSize}} · unterstützt: camt.053.001.02 und .001.08</span>
            </div>
            <button class="button primary" type="submit"{{if not .HasPaymentImportUnits}} disabled{{end}}>Vorschau erstellen</button>
          </form>
          <p class="privacy-note">Die Datei wird nicht gespeichert. 15 Minuten lang bleiben nur Referenz, Betrag und Prüfdaten im Arbeitsspeicher – nie IBAN oder Namen.</p>
        </section>

        {{with .PaymentImportPreview}}
          <section class="panel import-panel" id="preview">
            <div class="preview-head">
              <div><div class="kicker">Vorschau</div><h2>{{.Filename}}</h2><p class="muted">Geprüft {{.CreatedAt}} · noch nichts übernommen</p></div>
              <span class="profile-chip">{{.SourceVersion}}</span>
            </div>
            <div class="preview-metrics" aria-label="Prüfergebnis">
              <div class="preview-metric"><strong>{{.Assigned}}</strong><span>eindeutig</span></div>
              <div class="preview-metric"><strong>{{.Unclear}}</strong><span>zu prüfen</span></div>
              <div class="preview-metric"><strong>{{.Rejected}}</strong><span>abgelehnt</span></div>
            </div>
            {{if .AlreadyApplied}}<p class="flash ok">Diese Datei wurde bereits übernommen. Eine zweite Übernahme ist gesperrt.</p>{{end}}
            {{if .Changed}}<p class="flash">Einheiten oder Referenzen haben sich seit der Vorschau geändert. Bitte eine neue Vorschau erstellen.</p>{{end}}
            <div class="import-rows">
              {{range .Rows}}
                <article class="import-row">
                  <span class="decision {{.DecisionClass}}">{{.Decision}}</span>
                  <code class="row-reference">{{.Reference}}</code>
                  <span class="row-unit"><strong>{{.UnitLabel}}</strong>{{if .Status}}<span>→ {{.Status}}</span>{{end}}</span>
                  <strong class="row-amount">{{.Amount}}</strong>
                  <span class="row-reason">{{.Reason}}</span>
                </article>
              {{end}}
            </div>
            <form class="apply-bar" method="post" action="/app/settings/payments/import/apply">
              <input type="hidden" name="preview_token" value="{{.Token}}">
                <span class="mini">Unklare Zeilen bleiben unverändert. Die Übernahme wird protokolliert und lässt sich nicht doppelt ausführen.</span>
              {{if .CanApply}}<button class="button primary" type="submit">Eindeutige Treffer übernehmen</button>{{else}}<button class="button" type="button" disabled>Keine Übernahme möglich</button>{{end}}
            </form>
          </section>
        {{end}}
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "buildingSettings"}}
{{template "appOpen" .}}
    <style>
      .building .page { gap: 18px; }
      .building .page-intro { display: grid; gap: 6px; }
      .building .page-intro .lede { max-width: 720px; }
      .building .section-nav { position: sticky; top: 10px; z-index: 8; display: flex; gap: 8px; overflow-x: auto; padding: 8px; border: 1px solid var(--line); border-radius: 12px; background: rgba(255,254,251,.96); box-shadow: 0 8px 22px rgba(37,45,38,.08); scrollbar-width: none; }
      .building .section-nav::-webkit-scrollbar { display: none; }
      .building .section-nav a { flex: 0 0 auto; min-height: 42px; display: inline-flex; align-items: center; justify-content: center; padding: 0 15px; border-radius: 8px; color: var(--ink); font-size: 13.5px; font-weight: 800; text-decoration: none; }
      .building .section-nav a:first-child { background: var(--ink); color: #fff; }
      .building .section-nav a:hover, .building .section-nav a:focus-visible { background: var(--panel-soft); color: var(--ink); outline: 2px solid var(--gold); outline-offset: 1px; }
      .building .section-nav a:first-child:hover, .building .section-nav a:first-child:focus-visible { background: var(--ink); color: #fff; }
      .building #overview, .building #contacts, .building #units, .building #appearance, .building #unit-add { scroll-margin-top: 86px; }
      .building .settings-disclosure { padding: 0; overflow: clip; }
      .building .settings-disclosure > summary { min-height: 92px; display: grid; grid-template-columns: 52px minmax(0,1fr) auto; gap: 15px; align-items: center; padding: 18px 20px; cursor: pointer; list-style: none; }
      .building .settings-disclosure > summary::-webkit-details-marker, .building .unit-editor > summary::-webkit-details-marker, .building .unit-add > summary::-webkit-details-marker { display: none; }
      .building .settings-disclosure[open] > summary { border-bottom: 1px solid var(--line); }
      .building .section-icon { width: 48px; height: 48px; display: grid; place-items: center; border: 1px solid var(--line); border-radius: 50%; background: var(--panel-soft); color: var(--gold-ink); font-family: var(--font-serif); font-size: 23px; font-weight: 800; }
      .building .summary-copy { min-width: 0; }
      .building .summary-copy h2 { margin: 0; }
      .building .summary-copy p { margin: 3px 0 0; color: var(--muted); font-size: 13.5px; line-height: 1.4; overflow-wrap: anywhere; }
      .building .disclosure-action { display: inline-flex; align-items: center; gap: 8px; color: var(--ink); font-size: 13px; font-weight: 800; }
      .building .disclosure-action::after { content: "›"; font-family: var(--font-serif); font-size: 27px; line-height: 1; transform: rotate(90deg); transition: transform .15s ease; }
      .building .settings-disclosure[open] .disclosure-action::after { transform: rotate(-90deg); }
      .building .disclosure-body { display: grid; gap: 18px; padding: 20px; }
      .building .meta-form { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 12px; }
      .building .meta-form .full, .building .unit-form .full { grid-column: 1 / -1; }
      .building textarea { min-height: 88px; }
      .building .contact-groups { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 12px; }
      .building .contact-group { display: grid; align-content: start; gap: 11px; padding: 15px; border: 1px solid var(--line); border-radius: 10px; background: var(--panel-soft); }
      .building .contact-group h3 { margin: 0; font-family: var(--font-serif); font-size: 20px; }
      .building .section-save { display: flex; justify-content: flex-end; align-items: center; gap: 12px; padding-top: 2px; }
      .building .section-save .mini { margin-right: auto; }
      .building .brand-grid { display: grid; grid-template-columns: minmax(0,.9fr) minmax(320px,1.1fr); gap: 18px; align-items: start; }
      .building .brand-settings, .building .hero-settings { display: grid; gap: 13px; }
      .building .brand-preview { display: grid; grid-template-columns: 58px minmax(0,1fr); gap: 13px; align-items: center; border: 1px solid var(--line); border-radius: 9px; padding: 12px; background: var(--panel-soft); }
      .building .brand-preview-mark { width: 52px; height: 52px; border-radius: 8px; display: grid; place-items: center; color: var(--gold-ink); background: #fffefb; border: 1px solid var(--line); }
      .building .brand-preview-mark svg { width: 39px; height: 34px; display: block; stroke: currentColor; stroke-width: 2.2; fill: none; stroke-linecap: round; stroke-linejoin: round; }
      .building .brand-preview strong { display: block; font-family: var(--font-serif); font-size: 18px; }
      .building .brand-preview span { display: block; margin-top: 3px; color: var(--muted); font-size: 13px; line-height: 1.35; }
      .building .hero-preview { width: 100%; aspect-ratio: 16 / 7; object-fit: cover; border: 1px solid var(--line); border-radius: 9px; background: var(--panel-soft); }
      .building .hero-form { display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 10px; align-items: end; }
      .building .hero-actions { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
      .building .hero-delete { margin: 0; }
      .building .unit-panel { display: grid; gap: 16px; }
      .building .unit-head { display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 18px; align-items: start; }
      .building .unit-head-actions { display: flex; gap: 8px; align-items: start; flex-wrap: wrap; justify-content: flex-end; }
      .building .unit-head h2 { margin-bottom: 4px; }
      .building .unit-metrics { display: flex; gap: 8px; flex-wrap: wrap; margin-top: 10px; }
      .building .unit-metric { display: inline-flex; gap: 6px; align-items: baseline; border: 1px solid var(--line); border-radius: 999px; padding: 6px 10px; background: var(--panel-soft); color: #6f6a5c; font-size: 12.5px; font-weight: 750; }
      .building .unit-metric strong { color: var(--ink); font-size: 15px; }
      .building .home-profile-context { display: grid; grid-template-columns: 48px minmax(150px,.7fr) minmax(260px,1.25fr) auto; gap: 15px; align-items: center; border: 1px solid var(--line); border-radius: 10px; padding: 14px 16px; background: var(--panel-soft); }
      .building .home-profile-context-icon { width: 46px; height: 46px; display: grid; place-items: center; border-radius: 50%; color: var(--gold-ink); background: #fffefb; }
      .building .home-profile-context-icon svg { width: 24px; height: 24px; fill: none; stroke: currentColor; stroke-width: 1.8; stroke-linecap: round; stroke-linejoin: round; }
      .building .home-profile-context-name { min-width: 0; display: grid; gap: 2px; }
      .building .home-profile-context-name span { color: var(--gold-ink); font-size: 10.5px; font-weight: 850; letter-spacing: .08em; text-transform: uppercase; }
      .building .home-profile-context-name strong { font-family: var(--font-serif); font-size: 21px; line-height: 1.15; overflow-wrap: anywhere; }
      .building .home-profile-context-name small, .building .home-profile-context > p { color: var(--muted); font-size: 12.5px; line-height: 1.4; }
      .building .home-profile-context-name .home-profile-unit { font-weight: 650; }
      .building .home-profile-context-name .home-profile-meta { margin-top: 1px; color: var(--soft); font-size: 11.5px; }
      .building .home-profile-context > p { margin: 0; }
      .building .unit-add { border: 1px solid var(--ink); border-radius: 9px; background: var(--ink); color: #fff; }
      .building .unit-add > summary { min-height: 44px; display: flex; align-items: center; justify-content: center; gap: 8px; padding: 0 15px; cursor: pointer; list-style: none; font-weight: 800; }
      .building .unit-add > summary::before { content: "+"; width: 22px; height: 22px; display: grid; place-items: center; border: 1px solid rgba(255,255,255,.55); border-radius: 50%; font-size: 18px; line-height: 1; }
      .building .unit-add[open] { grid-column: 1 / -1; background: var(--panel-soft); color: var(--ink); border-color: var(--line); }
      .building .unit-add[open] > summary { justify-content: flex-start; border-bottom: 1px solid var(--line); }
      .building .unit-add[open] > summary::before { border-color: var(--gold); color: var(--ink); transform: rotate(45deg); }
      .building .unit-add .unit-form { padding: 16px; }
      .building .unit-list { display: grid; gap: 8px; }
      .building .unit-editor { border: 1px solid var(--line); border-radius: 10px; background: #fffefb; overflow: clip; }
      .building .unit-editor > summary { min-height: 76px; display: grid; grid-template-columns: minmax(145px,1.15fr) minmax(120px,.9fr) minmax(150px,1fr) auto; gap: 14px; align-items: center; padding: 13px 15px; cursor: pointer; list-style: none; }
      .building .unit-editor[open] > summary { border-bottom: 1px solid var(--line); background: var(--panel-soft); }
      .building .unit-name { display: grid; gap: 2px; min-width: 0; }
      .building .unit-name strong { font-family: var(--font-serif); font-size: 20px; line-height: 1.15; overflow-wrap: anywhere; }
      .building .unit-name span, .building .unit-share span, .building .payment-meta { color: var(--muted); font-size: 12.5px; font-weight: 700; line-height: 1.35; }
      .building .unit-name .unit-official { font-weight: 650; }
      .building .unit-name .unit-meta { color: var(--soft); font-size: 11.5px; font-weight: 600; line-height: 1.3; }
      .building .unit-share { display: grid; gap: 2px; min-width: 0; }
      .building .unit-share strong { font-size: 13.5px; overflow-wrap: anywhere; }
      .building .unit-open { min-height: 42px; display: inline-flex; align-items: center; justify-content: center; gap: 7px; padding: 0 12px; border: 1px solid var(--line); border-radius: 8px; color: var(--ink); font-size: 13px; font-weight: 800; }
      .building .unit-editor[open] .unit-open { border-color: var(--gold); }
      .building .unit-editor-body { display: grid; gap: 16px; padding: 16px; }
      .building .unit-form { display: grid; grid-template-columns: repeat(12,minmax(0,1fr)); gap: 10px; align-items: end; }
      .building .unit-form .f-label { grid-column: span 4; }
      .building .unit-form .f-type { grid-column: span 3; }
      .building .unit-form .f-share { grid-column: span 3; }
      .building .unit-form .f-owners, .building .unit-form .f-renters { grid-column: span 6; }
      .building .unit-form .f-actions { grid-column: span 2; display: flex; justify-content: flex-end; }
      .building .unit-tools { display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 14px; align-items: end; padding-top: 15px; border-top: 1px solid var(--line); }
      .building .payment-block { display: grid; gap: 8px; }
      .building .payment-block h3 { margin: 0; font-family: var(--font-serif); font-size: 18px; }
      .building .payment-status-form { display: grid; grid-template-columns: minmax(150px,230px) auto minmax(0,1fr); gap: 8px; align-items: center; }
      .building .payment-status-form .sr-only { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip: rect(0,0,0,0); white-space: nowrap; border: 0; }
      .building .payment-status-form select { min-height: 42px; }
      .building .unit-delete { margin: 0; }
      @media (max-width: 960px) {
        .building .contact-groups, .building .brand-grid { grid-template-columns: 1fr; }
        .building .unit-editor > summary { grid-template-columns: minmax(140px,1fr) minmax(120px,.8fr) auto; }
        .building .unit-editor > summary .unit-share { display: none; }
      }
      @media (min-width: 761px) and (max-width: 1120px) {
        .building .home-profile-context { grid-template-columns: 48px minmax(0,1fr); }
        .building .home-profile-context > p, .building .home-profile-context > .button { grid-column: 2; }
        .building .home-profile-context > .button { justify-self: start; }
      }
      @media (max-width: 760px) {
        .building .page { gap: 14px; }
        .building .page-intro h1 { font-size: clamp(38px,12vw,52px); }
        .building .page-intro .lede { font-size: 15px; line-height: 1.45; }
        .building .section-nav { top: 6px; margin-inline: -2px; padding: 6px; }
        .building .section-nav a { min-height: 44px; padding-inline: 13px; }
        .building .settings-disclosure > summary { min-height: 82px; grid-template-columns: 44px minmax(0,1fr) auto; gap: 11px; padding: 15px; }
        .building .section-icon { width: 42px; height: 42px; font-size: 20px; }
        .building .summary-copy h2 { font-size: 24px; }
        .building .summary-copy p { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
        .building .disclosure-action { font-size: 0; }
        .building .disclosure-body { padding: 15px; }
        .building .meta-form, .building .unit-form { grid-template-columns: 1fr; }
        .building .meta-form > *, .building .unit-form .f-label, .building .unit-form .f-type, .building .unit-form .f-share, .building .unit-form .f-owners, .building .unit-form .f-renters, .building .unit-form .f-actions { grid-column: 1 / -1; }
        .building .contact-groups { gap: 10px; }
        .building .section-save { position: sticky; bottom: 8px; z-index: 5; margin: 2px -5px -5px; padding: 9px; border: 1px solid var(--line); border-radius: 10px; background: rgba(255,254,251,.97); box-shadow: 0 8px 24px rgba(37,45,38,.16); }
        .building .section-save .mini { display: none; }
        .building .section-save .button { width: 100%; min-height: 46px; }
        .building .hero-form { grid-template-columns: 1fr; }
        .building .hero-actions .button { width: 100%; }
        .building .unit-head { grid-template-columns: 1fr; gap: 13px; }
        .building .unit-head-actions { display: grid; grid-template-columns: 1fr; justify-content: stretch; }
        .building .unit-head-actions > .button { width: 100%; min-height: 44px; }
        .building .home-profile-context { grid-template-columns: 46px minmax(0,1fr); gap: 11px; padding: 14px; }
        .building .home-profile-context > p, .building .home-profile-context > .button { grid-column: 1 / -1; }
        .building .home-profile-context > .button { width: 100%; }
        .building .unit-add { width: 100%; }
        .building .unit-editor > summary { min-height: 96px; grid-template-columns: minmax(0,1fr) auto; gap: 9px; padding: 13px; }
        .building .unit-editor > summary .unit-share { display: grid; grid-column: 1 / -1; grid-row: 2; }
        .building .unit-editor > summary .pill { grid-column: 2; grid-row: 1; }
        .building .unit-open { grid-column: 2; grid-row: 2; min-height: 38px; }
        .building .unit-tools { grid-template-columns: 1fr; }
        .building .payment-status-form { grid-template-columns: 1fr; align-items: stretch; }
        .building .unit-delete .button { width: 100%; min-height: 44px; }
      }
    </style>
    <script src="/assets/attachments.js?v={{.AssetVersion}}" defer></script>
    <main id="main-content" tabindex="-1" class="app-main building">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg><span>/</span><a href="/app/settings">Einstellungen</a><span>/</span><span>Gebäude &amp; Einheiten</span></span>
        <div class="page-actions"><a class="button" href="/app/settings">Zurück zu Einstellungen</a></div>
      </div>
      <section class="page wide">
        <div class="page-intro">
          <h1>Gebäude &amp; Einheiten</h1>
          <p class="lede">Hausdaten, Kontakte und Einheiten an einem Ort.</p>
        </div>
        <nav class="section-nav" aria-label="Bereiche">
          <a href="#overview">Stammdaten</a>
          <a href="#contacts">Kontakte</a>
          <a href="#units">Einheiten</a>
          <a href="#appearance">Erscheinungsbild</a>
        </nav>

        <form id="building-meta-form" method="post" action="/app/settings/building"></form>

        <details class="panel settings-disclosure" id="overview" open>
          <summary>
            <span class="section-icon" aria-hidden="true">⌂</span>
            <span class="summary-copy"><h2>Stammdaten</h2><p>{{.Tenant.Name}} · {{.Tenant.Address}}</p></span>
            <span class="disclosure-action">Bearbeiten</span>
          </summary>
          <div class="disclosure-body">
            {{if .BuildingMsg}}<p class="flash {{if .BuildingOK}}ok{{end}}">{{.BuildingMsg}}</p>{{end}}
            <div class="meta-form">
              <label class="full" for="building-name">Name
                <input id="building-name" form="building-meta-form" type="text" name="name" value="{{.Tenant.Name}}" maxlength="160" required>
              </label>
              <label class="full" for="building-address">Adresse
                <textarea id="building-address" form="building-meta-form" name="address" maxlength="500" required>{{.Tenant.Address}}</textarea>
              </label>
            </div>
            <div class="section-save"><span class="mini">Gilt für dieses Hausportal.</span><button class="button primary" type="submit" form="building-meta-form">Änderungen speichern</button></div>
          </div>
        </details>

        <details class="panel settings-disclosure" id="contacts">
          <summary>
            <span class="section-icon" aria-hidden="true">☎</span>
            <span class="summary-copy"><h2>Hauskontakte</h2><p>Verwaltung, Notdienst und Hausmeister</p></span>
            <span class="disclosure-action">Bearbeiten</span>
          </summary>
          <div class="disclosure-body">
            <div class="contact-groups">
              <section class="contact-group">
                <h3>Verwaltung</h3>
                <label for="contact-name">Name oder Firma
                  <input id="contact-name" form="building-meta-form" type="text" name="contact_name" value="{{.Tenant.ContactName}}" maxlength="160">
                </label>
                <label for="contact-address">Anschrift
                  <textarea id="contact-address" form="building-meta-form" name="contact_address" maxlength="500" placeholder="Straße, PLZ Ort">{{.Tenant.ContactAddress}}</textarea>
                </label>
                <label for="contact-email">E-Mail
                  <input id="contact-email" form="building-meta-form" type="email" name="contact_email" value="{{.Tenant.ContactEmail}}" maxlength="160" autocomplete="email">
                </label>
                <label for="contact-phone">Telefon
                  <input id="contact-phone" form="building-meta-form" type="tel" name="contact_phone" value="{{.Tenant.ContactPhone}}" maxlength="80" autocomplete="tel">
                </label>
              </section>
              <section class="contact-group">
                <h3>Notdienst</h3>
                <label for="emergency-name">Bezeichnung
                  <input id="emergency-name" form="building-meta-form" type="text" name="emergency_name" value="{{.Tenant.EmergencyName}}" maxlength="160" placeholder="Notdienst">
                </label>
                <label for="emergency-phone">Telefon
                  <input id="emergency-phone" form="building-meta-form" type="tel" name="emergency_phone" value="{{.Tenant.EmergencyPhone}}" maxlength="80" autocomplete="tel">
                </label>
              </section>
              <section class="contact-group">
                <h3>Hausmeister</h3>
                <label for="caretaker-name">Name
                  <input id="caretaker-name" form="building-meta-form" type="text" name="caretaker_name" value="{{.Tenant.CaretakerName}}" maxlength="160">
                </label>
                <label for="caretaker-email">E-Mail
                  <input id="caretaker-email" form="building-meta-form" type="email" name="caretaker_email" value="{{.Tenant.CaretakerEmail}}" maxlength="160" autocomplete="email">
                </label>
                <label for="caretaker-phone">Telefon
                  <input id="caretaker-phone" form="building-meta-form" type="tel" name="caretaker_phone" value="{{.Tenant.CaretakerPhone}}" maxlength="80" autocomplete="tel">
                </label>
              </section>
            </div>
            <div class="section-save"><span class="mini">Diese Angaben erscheinen bei den Hauskontakten.</span><button class="button primary" type="submit" form="building-meta-form">Änderungen speichern</button></div>
          </div>
        </details>

        <section id="units" class="panel unit-panel">
          <div class="unit-head">
            <div>
              <h2>Einheiten</h2>
              <p class="muted">Stammdaten und manueller Zahlungsstatus direkt je Einheit.</p>
              <div class="unit-metrics" aria-label="Einheiten Übersicht">
                <span class="unit-metric"><strong>{{.UnitTotal}}</strong> Einträge</span>
                <span class="unit-metric"><strong>{{.BillableUnits}}</strong> von {{.FairUseFreeUnits}} {{.BillableLabel}} (Fair Use)</span>
              </div>
              {{if .FairUseExceeded}}<p class="muted">Über dem kostenlosen Rahmen von {{.FairUseFreeUnits}} Wohneinheiten — Richtwert 1 € pro Einheit und Monat.</p>{{end}}
            </div>
            <div class="unit-head-actions">
              <a class="button" href="/app/settings/payments/import">Bankdatei einlesen</a>
              <details class="unit-add" id="unit-add">
                <summary>Einheit hinzufügen</summary>
                <form class="unit-form" method="post" action="/app/settings/building/units">
                <label class="f-label">Offizielle Bezeichnung
                  <input type="text" name="label" maxlength="120" required placeholder="Top 1">
                </label>
                <label class="f-type">Typ
                  <select name="unit_type">
                    <option value="residential" selected>Wohnung</option>
                    <option value="commercial">Geschäftslokal</option>
                    <option value="parking">Stellplatz</option>
                    <option value="storage">Keller / Lager</option>
                    <option value="other">Sonstiges</option>
                  </select>
                </label>
                <label class="f-share">Miteigentumsanteil
                  <input type="number" name="miteigentumsanteil" min="0" max="1000000" step="1" value="0" inputmode="numeric">
                </label>
                <label class="f-owners">Eigentümer E-Mails
                  <input type="text" name="owner_emails" placeholder="name@example.com, zweite@example.com">
                </label>
                <label class="f-renters">Mieter E-Mails
                  <input type="text" name="renter_emails" placeholder="name@example.com">
                </label>
                  <div class="f-actions"><button class="button primary" type="submit">Einheit anlegen</button></div>
                </form>
              </details>
            </div>
          </div>
          {{if .HasHomeProfile}}
            {{if .HomeProfileSaved}}<p class="flash ok">Der Anzeigename von „Mein Zuhause“ wurde gespeichert.</p>{{end}}
            <aside class="home-profile-context" data-home-identity="building-context" aria-label="{{.HomeIdentity.AriaLabel}}">
              <span class="home-profile-context-icon" aria-hidden="true"><svg viewBox="0 0 24 24"><path d="m4 11 8-7 8 7"/><path d="M6 10v10h12V10"/><path d="M10 20v-6h4v6"/></svg></span>
              <span class="home-profile-context-name"><span>Mein Zuhause</span><strong data-home-display-name>{{.HomeProfile.HouseholdName}}</strong>{{if .HasHomeProfileUnit}}<small class="home-profile-unit" data-home-unit-label>{{.HomeProfileUnitLabel}}</small>{{end}}<small class="home-profile-meta">{{.HomeTypeLabel}}{{if not .HasHomeProfileUnit}} · {{.HomeProfileScopeLabel}}{{end}}</small></span>
              <p>{{if .HasHomeProfileUnit}}„{{.HomeProfileUnitLabel}}“ bleibt die offizielle Stammdatenbezeichnung.{{else if eq .HomeProfile.HomeType "apartment"}}Dieses Wohnungsprofil ist noch keiner offiziellen Einheit zugeordnet.{{else}}Anzeigename für Energie, Wartung und Empfehlungen der Liegenschaft.{{end}}</p>
              <a class="button" href="/app/settings/home?from=building">Zuhause bearbeiten</a>
            </aside>
          {{end}}
          {{if .UnitMsg}}<p class="flash {{if .UnitOK}}ok{{end}}">{{.UnitMsg}}</p>{{end}}
          {{if .PaymentMsg}}<p class="flash {{if .PaymentOK}}ok{{end}}">{{.PaymentMsg}}</p>{{end}}
          {{if .Units}}
            <div class="unit-list">
              {{range .Units}}
                <details class="unit-editor" id="unit-{{.ID}}">
                  <summary>
                    <span class="unit-name"{{if .HasHomeDisplayName}} data-home-identity="building-unit" aria-label="{{.HomeDisplayName}}, offizielle Einheit {{.Label}}"{{end}}>{{if .HasHomeDisplayName}}<strong data-home-display-name>{{.HomeDisplayName}}</strong><span class="unit-official" data-home-unit-label>{{.Label}}</span><small class="unit-meta">{{.UnitTypeLabel}} · {{.BillableLabel}}</small>{{else}}<strong>{{.Label}}</strong><span>{{.UnitTypeLabel}} · {{.BillableLabel}}</span>{{end}}</span>
                    <span class="unit-share"><strong>{{.Share}}</strong><span>{{.MembersLabel}}</span></span>
                    <span class="pill {{.PaymentStatusClass}}">{{.PaymentStatus}}</span>
                    <span class="unit-open">Bearbeiten</span>
                  </summary>
                  <div class="unit-editor-body">
                    <form class="unit-form" method="post" action="/app/settings/building/units">
                      <input type="hidden" name="orig_id" value="{{.ID}}">
                      <input type="hidden" name="id" value="{{.ID}}">
                      <label class="f-label">Offizielle Bezeichnung
                        <input type="text" name="label" value="{{.Label}}" maxlength="120" required>
                      </label>
                      <label class="f-type">Typ
                        <select name="unit_type">
                          {{range .TypeOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}
                        </select>
                      </label>
                      <label class="f-share">Miteigentumsanteil
                        <input type="number" name="miteigentumsanteil" min="0" max="1000000" step="1" value="{{.ShareValue}}" inputmode="numeric">
                      </label>
                      <label class="f-owners">Eigentümer E-Mails
                        <input type="text" name="owner_emails" value="{{.OwnerEmails}}">
                      </label>
                      <label class="f-renters">Mieter E-Mails
                        <input type="text" name="renter_emails" value="{{.RenterEmails}}">
                      </label>
                      <div class="f-actions"><button class="button primary" type="submit">Einheit speichern</button></div>
                    </form>
                    <div class="unit-tools">
                      <div class="payment-block">
                        <h3>Zahlungsstatus</h3>
                        <form class="payment-status-form" method="post" action="/app/settings/building/payment-status">
                          <input type="hidden" name="unit_id" value="{{.ID}}">
                          <label class="sr-only" for="payment-status-{{.ID}}">Status für {{.Label}}</label>
                          <select id="payment-status-{{.ID}}" name="status">{{range .PaymentOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}</select>
                          <button class="button small" type="submit">Status speichern</button>
                          <span class="payment-meta">{{if .PaymentHasUpdated}}{{.PaymentUpdatedAt}}{{else}}{{.PaymentDetail}}{{end}}</span>
                        </form>
                      </div>
                      <form class="unit-delete" method="post" action="/app/settings/building/units/delete" data-confirm="{{.DeleteConfirmLabel}}?">
                        <input type="hidden" name="id" value="{{.ID}}">
                        <button class="button small ghost" type="submit">Einheit entfernen</button>
                      </form>
                    </div>
                  </div>
                </details>
              {{end}}
            </div>
          {{else}}
            {{template "emptyState" .UnitsEmpty}}
          {{end}}
        </section>

        <details class="panel settings-disclosure" id="appearance">
          <summary>
            <span class="section-icon" aria-hidden="true">◇</span>
            <span class="summary-copy"><h2>Erscheinungsbild</h2><p>Portal-Symbol, Kurzkennung und Titelbild</p></span>
            <span class="disclosure-action">Bearbeiten</span>
          </summary>
          <div class="disclosure-body">
            {{if .HeroMsg}}<p class="flash {{if .HeroOK}}ok{{end}}">{{.HeroMsg}}</p>{{end}}
            <div class="brand-grid">
              <section class="brand-settings">
                <div class="brand-preview">
                  <span class="brand-preview-mark">{{template "tenantBrandMark" .}}</span>
                  <div><strong>{{.BrandIconLabel}}</strong><span>{{.Tenant.BrandAbbreviation}} erscheint als kurze Kennung in der Seitenleiste.</span></div>
                </div>
                <label for="brand-icon">Portal-Symbol
                  <select id="brand-icon" form="building-meta-form" name="brand_icon">
                    {{range .BrandIconOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}
                  </select>
                </label>
                <label for="brand-abbreviation">Kurzkennung
                  <input id="brand-abbreviation" form="building-meta-form" type="text" name="brand_abbreviation" value="{{.Tenant.BrandAbbreviation}}" maxlength="12" placeholder="JHW22">
                </label>
                <div class="section-save"><button class="button primary" type="submit" form="building-meta-form">Änderungen speichern</button></div>
              </section>
              <section class="hero-settings">
                <img class="hero-preview" src="{{.Tenant.HeroImageURL}}" alt="">
                <form class="hero-form" method="post" action="/app/settings/building/hero" enctype="multipart/form-data">
                  <label for="hero-image">Titelbild
                    <span class="file-control"><input id="hero-image" type="file" name="hero_image" accept="image/jpeg,image/png,image/webp" required><span>Bild auswählen</span></span>
                  </label>
                  <div class="hero-actions"><button class="button" type="submit">Titelbild speichern</button></div>
                </form>
                <span class="mini">JPG, PNG oder WebP bis 5 MB.</span>
                {{if .HasCustomHero}}
                  <form class="hero-delete" method="post" action="/app/settings/building/hero/delete" data-confirm="Titelbild entfernen und Standardbild verwenden?">
                    <button class="button small ghost" type="submit">Standardbild verwenden</button>
                  </form>
                {{end}}
              </section>
            </div>
          </div>
        </details>
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "profileSettings"}}
{{template "appOpen" .}}
    <style>
      /* Das Formular bleibt primär. Erklärungen und unveränderliche Kontodaten
         sind als kurze, zugängliche Details verfügbar, ohne den Weg zu bremsen. */
      .profile { display: grid; gap: 18px; }
      .profile .profile-head { display: flex; justify-content: space-between; gap: 18px; align-items: end; }
      .profile .profile-head .lede { margin-top: 4px; max-width: 640px; }
      .profile .profile-layout { display: grid; gap: 16px; align-items: start; }
      .profile .profile-col, .profile .profile-aside { min-width: 0; display: grid; gap: 14px; align-content: start; }
      .profile .settings-card { display: grid; gap: 0; padding: 0; overflow: hidden; }
      .profile .profile-flash { margin: 0; padding: 10px 13px; border-radius: 9px; font-size: 13.5px; font-weight: 600; border: 1px solid transparent; }
      .profile .profile-flash.ok { background: rgba(47,107,74,.12); color: var(--leaf); border-color: rgba(47,107,74,.25); }
      .profile .profile-flash.warn { background: rgba(150,40,40,.08); color: #9a2b2b; border-color: rgba(150,40,40,.22); }
      .profile .profile-flash { margin: 18px 18px 0; }
      .profile .profile-form { display: grid; gap: 0; }
      .profile .profile-section { display: grid; gap: 12px; padding: 18px; border-bottom: 1px solid var(--line); }
      .profile .profile-section-head { display: grid; gap: 3px; }
      .profile .profile-section-head h2 { font-size: 20px; }
      .profile .profile-section-head p { color: var(--muted); font-size: 12.5px; }
      .profile .profile-fields { display: grid; grid-template-columns: 140px repeat(2,minmax(0,1fr)); gap: 12px; }
      .profile .profile-contact-fields { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 12px; }
      .profile .profile-readonly { display: grid; gap: 7px; }
      .profile .profile-readonly span { color: var(--gold-ink); font-size: 11px; font-weight: 800; letter-spacing: .06em; text-transform: uppercase; }
      .profile .profile-readonly strong { min-height: 42px; display: flex; align-items: center; border: 1px solid var(--line); border-radius: var(--radius-xs); padding: 9px 12px; background: var(--panel-soft); font-size: 13px; overflow-wrap: anywhere; }
      .profile .directory-check { min-height: 72px; display: grid; grid-template-columns: 22px minmax(0,1fr); gap: 11px; align-items: start; border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 13px; background: var(--panel-soft); color: var(--ink); font-size: 14px; font-weight: 700; letter-spacing: 0; text-transform: none; cursor: pointer; }
      .profile .directory-check input { width: 20px; height: 20px; min-height: 0; margin: 1px 0 0; accent-color: var(--leaf); }
      .profile .directory-check span { display: grid; gap: 3px; }
      .profile .directory-check small { color: var(--muted); font-size: 12.5px; font-weight: 500; line-height: 1.4; }
      .profile .profile-actions { display: flex; align-items: center; justify-content: flex-end; gap: 10px; padding: 15px 18px; background: rgba(247,243,234,.55); }
      .profile .profile-cancel { color: var(--muted); font-size: 13px; font-weight: 750; }
      .profile .account-details { border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel); box-shadow: var(--shadow-panel); overflow: hidden; }
      .profile .account-details > summary { min-height: 56px; display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 4px 12px; align-items: baseline; padding: 15px 18px 11px; cursor: pointer; color: var(--ink); list-style: none; }
      .profile .account-details > summary::-webkit-details-marker { display: none; }
      .profile .account-details > summary strong { font-family: var(--font-serif); font-size: 19px; font-weight: 700; }
      .profile .account-details > summary::after { content: "\203A"; grid-column: 2; grid-row: 1; justify-self: end; color: var(--gold-ink); font-size: 21px; line-height: 1; transform: rotate(90deg); }
      .profile .account-details:not([open]) > summary::after { transform: none; }
      .profile .account-details > summary span { grid-column: 1 / -1; color: var(--muted); font-size: 12.5px; font-weight: 500; line-height: 1.4; }
      .profile .profile-visibility-body { display: grid; gap: 13px; border-top: 1px solid var(--line); padding: 15px 18px 18px; }
      .profile .readonly-grid { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 10px 18px; padding: 0 18px 18px; }
      .profile .readonly-box { display: grid; align-content: start; gap: 6px; min-width: 0; border-top: 1px solid var(--line); padding-top: 11px; }
      .profile .readonly-box strong { font-family: var(--font-serif); font-size: 17px; overflow-wrap: anywhere; }
      .profile .readonly-box .muted { font-size: 13px; }
      .profile .chips { display: flex; flex-wrap: wrap; gap: 6px; }
      .profile .chip { display: inline-flex; align-items: center; border: 1px solid var(--line); background: var(--panel); color: #6f6a5c; border-radius: 8px; padding: 4px 10px; font-size: 12.5px; font-weight: 700; }
      .profile .unit-list { display: grid; gap: 8px; }
      .profile .unit-row { display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 10px; align-items: center; border-top: 1px solid var(--line); padding-top: 8px; }
      .profile .unit-row:first-child { border-top: 0; padding-top: 0; }
      .profile .profile-visibility { display: grid; gap: 11px; margin: 0; padding: 0; list-style: none; }
      .profile .profile-visibility li { display: grid; gap: 2px; border-top: 1px solid var(--line); padding-top: 10px; }
      .profile .profile-visibility li:first-child { border-top: 0; padding-top: 0; }
      .profile .profile-visibility strong { font-size: 13.5px; }
      .profile .profile-visibility span { color: var(--muted); font-size: 12.6px; line-height: 1.45; }
      .profile .profile-note-link { min-height: 44px; display: inline-flex; align-items: center; gap: 7px; color: var(--gold-ink); font-size: 13.5px; font-weight: 800; }
      .profile .profile-note-link::after { content: "\203A"; font-size: 19px; line-height: 1; }
      @media (min-width: 1181px) {
        .profile .profile-layout { grid-template-columns: minmax(0,1fr) 336px; }
        .profile .readonly-grid { grid-template-columns: minmax(0,1fr); }
      }
      @media (max-width: 1180px) and (min-width: 861px) {
        .profile .profile-aside { grid-template-columns: repeat(2,minmax(0,1fr)); align-items: stretch; }
      }
      @media (max-width: 860px) {
        .profile .readonly-grid { grid-template-columns: repeat(2,minmax(0,1fr)); }
      }
      @media (max-width: 680px) {
        .profile .profile-head { display: grid; gap: 8px; }
        .profile .profile-fields, .profile .profile-contact-fields, .profile .readonly-grid { grid-template-columns: 1fr; }
        .profile .profile-actions { display: grid; grid-template-columns: 1fr; gap: 6px; padding: 12px 18px; }
        .profile .profile-actions .button { width: 100%; min-height: 46px; }
        .profile .profile-cancel { min-height: 44px; display: grid; place-items: center; text-align: center; }
        .app-main .content-top .page-actions .button { min-height: 44px; }
      }
    </style>
    <main id="main-content" tabindex="-1" class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg><span>/</span><a href="/app/settings">Einstellungen</a><span>/</span><span>Profil</span></span>
      </div>
      <section class="page profile">
        <div class="profile-head">
          <div><h1>Profil</h1><p class="lede">Ihre Angaben und Sichtbarkeit. Änderungen gelten nur für Ihren Zugang.</p></div>
        </div>
        <div class="profile-layout">
        <div class="profile-col">
        <section class="panel settings-card">
          {{if .ProfileMsg}}<p class="profile-flash{{if .ProfileOK}} ok{{else}} warn{{end}}">{{.ProfileMsg}}</p>{{end}}
          <form class="profile-form" method="post" action="/app/settings/profile">
            <section class="profile-section">
              <div class="profile-section-head"><h2>Name</h2><p>So werden Sie im Portal angesprochen.</p></div>
              <div class="profile-fields">
                <label for="profile-title">Titel optional<input id="profile-title" name="title" value="{{.Profile.Title}}" maxlength="40" autocomplete="honorific-prefix" placeholder="z. B. Dr."></label>
                <label for="profile-first">Vorname<input id="profile-first" name="first_name" value="{{.Profile.FirstName}}" maxlength="120" autocomplete="given-name"></label>
                <label for="profile-last">Nachname<input id="profile-last" name="last_name" value="{{.Profile.LastName}}" maxlength="120" autocomplete="family-name"></label>
              </div>
            </section>
            <section class="profile-section">
              <div class="profile-section-head"><h2>Kontakt &amp; Sichtbarkeit</h2><p>Ihre Anmeldung bleibt unverändert.</p></div>
              <div class="profile-contact-fields">
                <div class="profile-readonly"><span>E-Mail</span><strong>{{.Email}}</strong></div>
                <label for="profile-phone">Telefon optional<input id="profile-phone" name="phone" value="{{.Profile.Phone}}" maxlength="80" autocomplete="tel"></label>
              </div>
              <label class="directory-check" for="profile-directory"><input id="profile-directory" type="checkbox" name="directory_opt_in"{{if .Profile.DirectoryOptIn}} checked{{end}}><span>Im Kontakte-Verzeichnis anzeigen<small>Name, E-Mail und – falls angegeben – Telefon werden für die Hausgemeinschaft sichtbar. Die Freigabe ist freiwillig.</small></span></label>
            </section>
            <div class="profile-actions">
              <a class="profile-cancel" href="/app/settings">Abbrechen</a>
              <button class="button primary" type="submit">Profil speichern</button>
            </div>
          </form>
        </section>
        </div>
        <aside class="profile-aside" aria-label="Wirkung Ihrer Angaben">
          <details class="account-details profile-visibility-details">
            <summary><strong id="profile-visibility-title">Was andere sehen</strong><span>Name ist sichtbar; Kontaktangaben nur nach Ihrer Freigabe.</span></summary>
            <div class="profile-visibility-body">
            <ul class="profile-visibility">
              <li><strong>Name</strong><span>Steht an Ihren Beiträgen, Anliegen und Stimmabgaben. Für die Hausgemeinschaft immer sichtbar.</span></li>
              <li><strong>E-Mail-Adresse</strong><span>Ihre Anmeldung. Sie bleibt unverändert und ist außerhalb der Verwaltung nur sichtbar, wenn Sie das Verzeichnis freigeben.</span></li>
              <li><strong>Telefon</strong><span>Freiwillig. Wird ausschließlich mit der Freigabe für das Kontakte-Verzeichnis sichtbar.</span></li>
              <li><strong>Verzeichniseintrag</strong><span>{{if .Profile.DirectoryOptIn}}Derzeit freigegeben – Ihr Eintrag steht unter Kontakte.{{else}}Derzeit nicht freigegeben – Ihr Eintrag fehlt unter Kontakte.{{end}} Sie können das jederzeit zurücknehmen.</span></li>
            </ul>
            <a class="profile-note-link" href="/app/kontakte">Kontakte ansehen</a>
            </div>
          </details>
          <details class="account-details">
            <summary><strong>Konto &amp; Berechtigungen</strong><span>Rolle, Anmeldung und Einheiten. Diese Angaben vergibt die Hausverwaltung.</span></summary>
            <div class="readonly-grid">
              <div class="readonly-box">
                <span class="field-label">Rolle</span>
                <strong>{{.Role}}</strong>
                <div class="chips">{{range .PermissionList}}<span class="chip">{{.}}</span>{{end}}</div>
              </div>
              <div class="readonly-box">
                <span class="field-label">Anmeldung</span>
                <div class="chips">{{range .AuthList}}<span class="chip">{{.}}</span>{{end}}</div>
              </div>
              <div class="readonly-box">
                <span class="field-label">Einheiten</span>
                {{if .HasUnits}}
                  <div class="unit-list">
                    {{range .Units}}
                      <div class="unit-row"><span><strong>{{.Label}}</strong><span class="mini">{{.Relation}}</span></span><span class="chip">{{.Share}}</span></div>
                    {{end}}
                  </div>
                {{else}}
                  <p class="muted">Keine Einheit verknüpft.</p>
                {{end}}
              </div>
            </div>
          </details>
        </aside>
        </div>
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "notificationSettings"}}
{{template "appOpen" .}}
    <style>
      .notifications { display: grid; gap: 18px; }
      .notifications .notification-head { display: grid; gap: 5px; }
      .notifications .notification-head .lede { max-width: 640px; }
      .notifications .notification-layout { display: grid; gap: 16px; align-items: start; }
      .notifications .notification-col, .notifications .notification-aside { min-width: 0; display: grid; gap: 14px; align-content: start; }
      .notifications .settings-card { display: grid; gap: 14px; }
      .notifications .notify-flash { margin: 0; padding: 10px 13px; border-radius: 9px; font-size: 13.5px; font-weight: 600; border: 1px solid transparent; }
      .notifications .notify-flash.ok { background: rgba(47,107,74,.12); color: var(--leaf); border-color: rgba(47,107,74,.25); }
      .notifications .notify-flash.warn { background: rgba(150,40,40,.08); color: #9a2b2b; border-color: rgba(150,40,40,.22); }
      .notifications .notification-form { display: grid; gap: 16px; }
      .notifications .notification-master { min-height: 88px; display: grid; grid-template-columns: 48px minmax(0,1fr) auto; gap: 13px; align-items: center; border: 1px solid rgba(47,107,74,.22); border-radius: var(--radius-sm); padding: 14px; background: linear-gradient(120deg,rgba(47,107,74,.08),rgba(255,255,255,.7)); color: var(--ink); letter-spacing: 0; text-transform: none; cursor: pointer; }
      .notifications .notification-master-icon { width: 48px; height: 48px; display: grid; place-items: center; border-radius: 50%; background: rgba(200,153,63,.14); color: var(--gold-ink); }
      .notifications .notification-master-icon svg { width: 23px; height: 23px; fill: none; stroke: currentColor; stroke-width: 1.8; }
      .notifications .notification-master-copy { display: grid; gap: 3px; }
      .notifications .notification-master-copy strong { font-family: var(--font-serif); font-size: 19px; }
      .notifications .notification-master-copy span { color: var(--muted); font-size: 12.5px; font-weight: 500; line-height: 1.35; }
      .notifications .notification-switch { position: relative; width: 48px; height: 28px; display: inline-block; flex: 0 0 auto; }
      .notifications .notification-switch input { position: absolute; opacity: 0; pointer-events: none; }
      .notifications .notification-switch-track { position: absolute; inset: 0; border-radius: 20px; background: #d7d5cf; box-shadow: inset 0 0 0 1px rgba(23,32,25,.1); transition: .16s ease; }
      .notifications .notification-switch-track::after { content: ""; position: absolute; width: 22px; height: 22px; left: 3px; top: 3px; border-radius: 50%; background: #fff; box-shadow: 0 2px 6px rgba(23,32,25,.2); transition: .16s ease; }
      .notifications .notification-switch input:checked + .notification-switch-track { background: var(--leaf); }
      .notifications .notification-switch input:checked + .notification-switch-track::after { transform: translateX(20px); }
      .notifications .notification-switch input:focus-visible + .notification-switch-track { outline: 3px solid var(--gold); outline-offset: 3px; }
      .notifications .notification-topics { display: grid; gap: 12px; transition: opacity .16s ease; }
      .notifications .notification-topics-head { display: flex; justify-content: space-between; gap: 14px; align-items: end; }
      .notifications .notification-topics-head h2 { font-size: 22px; }
      .notifications .notification-topics-head p { margin-top: 3px; color: var(--muted); font-size: 12.5px; }
      .notifications .notification-count { color: var(--gold-ink); font-size: 12px; font-weight: 850; white-space: nowrap; }
      .notifications .notification-group { overflow: hidden; border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel-soft); }
      .notifications .notification-group-title { padding: 10px 14px 7px; color: var(--gold-ink); font-size: 10.5px; font-weight: 900; letter-spacing: .08em; text-transform: uppercase; }
      .notifications .notification-row { min-height: 64px; display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 14px; align-items: center; border-top: 1px solid var(--line); padding: 11px 14px; color: var(--ink); letter-spacing: 0; text-transform: none; cursor: pointer; }
      .notifications .notification-row-copy { display: grid; gap: 3px; }
      .notifications .notification-row-copy strong { font-size: 14px; }
      .notifications .notification-row-copy span { color: var(--muted); font-size: 12px; font-weight: 500; line-height: 1.35; }
      .notifications .email-paused .notification-topics { opacity: .72; }
      .notifications .notification-paused-note { display: none; border-left: 3px solid var(--gold); padding: 9px 12px; background: rgba(200,153,63,.09); color: var(--muted); font-size: 12.5px; line-height: 1.4; }
      .notifications .email-paused .notification-paused-note { display: block; }
      .notifications .actions { display: flex; justify-content: flex-end; gap: 12px; align-items: center; border-top: 1px solid var(--line); padding-top: 14px; }
      .notifications .actions a { color: var(--muted); font-size: 13px; font-weight: 750; }
      .notifications .notification-details { border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel); box-shadow: var(--shadow-panel); overflow: hidden; }
      .notifications .notification-details > summary { min-height: 56px; display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 4px 12px; align-items: baseline; padding: 15px 18px 11px; color: var(--ink); cursor: pointer; list-style: none; }
      .notifications .notification-details > summary::-webkit-details-marker { display: none; }
      .notifications .notification-details > summary strong { font-family: var(--font-serif); font-size: 19px; }
      .notifications .notification-details > summary span { grid-column: 1 / -1; color: var(--muted); font-size: 12.5px; font-weight: 500; line-height: 1.4; }
      .notifications .notification-details > summary::after { content: "›"; grid-column: 2; grid-row: 1; color: var(--gold-ink); font-size: 21px; line-height: 1; }
      .notifications .notification-details[open] > summary::after { transform: rotate(90deg); }
      .notifications .notification-details-body { display: grid; gap: 13px; border-top: 1px solid var(--line); padding: 15px 18px 18px; }
      .notifications .notification-details-body p { color: var(--muted); font-size: 13px; line-height: 1.5; }
      .notifications .notification-address { display: grid; gap: 3px; border: 1px solid var(--line); border-radius: var(--radius-xs); padding: 11px 13px; background: var(--panel-soft); }
      .notifications .notification-address span { color: var(--gold-ink); font-size: 10.5px; font-weight: 850; letter-spacing: .08em; text-transform: uppercase; }
      .notifications .notification-address strong { font-size: 14px; overflow-wrap: anywhere; }
      .notifications .notification-rules { display: grid; gap: 11px; margin: 0; padding: 0; list-style: none; }
      .notifications .notification-rules li { display: grid; gap: 2px; border-top: 1px solid var(--line); padding-top: 10px; }
      .notifications .notification-rules li:first-child { border-top: 0; padding-top: 0; }
      .notifications .notification-rules strong { font-size: 13.5px; }
      .notifications .notification-rules span { color: var(--muted); font-size: 12.6px; line-height: 1.45; }
      .notifications .notification-note-link { min-height: 44px; display: inline-flex; align-items: center; gap: 7px; color: var(--gold-ink); font-size: 13.5px; font-weight: 800; }
      .notifications .notification-note-link::after { content: "\203A"; font-size: 19px; line-height: 1; }
      @media (min-width: 1181px) {
        .notifications .notification-layout { grid-template-columns: minmax(0,1fr) 336px; }
      }
      @media (max-width: 1180px) and (min-width: 861px) {
        .notifications .notification-aside { grid-template-columns: repeat(2,minmax(0,1fr)); align-items: stretch; }
      }
      @media (max-width: 680px) {
        .notifications .settings-card { padding: 14px; }
        .notifications .notification-master { grid-template-columns: 42px minmax(0,1fr) auto; padding: 12px; }
        .notifications .notification-master-icon { width: 42px; height: 42px; }
        .notifications .notification-master-copy strong { font-size: 17px; }
        .notifications .notification-topics-head { align-items: start; }
        .notifications .actions { display: grid; grid-template-columns: 1fr; gap: 6px; padding: 12px 0 0; }
        .notifications .actions .button { width: 100%; min-height: 46px; }
        .notifications .actions a { min-height: 44px; display: grid; place-items: center; text-align: center; }
        .app-main .content-top .page-actions .button { min-height: 44px; }
      }
    </style>
    <main id="main-content" tabindex="-1" class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg><span>/</span><a href="/app/settings">Einstellungen</a><span>/</span><span>Benachrichtigungen</span></span>
      </div>
      <section class="page notifications">
        <div class="notification-head">
          <h1>Benachrichtigungen</h1>
          <p class="lede">Festlegen, welche E-Mails Sie erhalten möchten. Die Auswahl gilt nur für Ihr Postfach.</p>
        </div>
        <div class="notification-layout">
        <div class="notification-col">
        <section class="panel settings-card">
          {{if .NotifyMsg}}<p class="notify-flash{{if .NotifyOK}} ok{{else}} warn{{end}}">{{.NotifyMsg}}</p>{{end}}
          <form class="notification-form{{if not .EmailNotificationsEnabled}} email-paused{{end}}" method="post" action="/app/settings/notifications" data-notification-form>
            <label class="notification-master">
              <span class="notification-master-icon"><svg viewBox="0 0 24 24"><path d="M3 6h18v12H3z"/><path d="m3 7 9 6 9-6"/></svg></span>
              <span class="notification-master-copy"><strong>E-Mails erhalten</strong><span data-notification-status>{{if .EmailNotificationsEnabled}}Aktuell zu {{.NotificationEnabledCount}} von {{.NotificationEventCount}} Themen.{{else}}Der Versand ist derzeit pausiert.{{end}}</span></span>
              <span class="notification-switch"><input type="checkbox" name="email_enabled" value="on" data-notification-master{{if .EmailNotificationsEnabled}} checked{{end}}><span class="notification-switch-track"></span></span>
            </label>
            <div class="notification-topics">
              <div class="notification-topics-head"><div><h2>Wofür?</h2><p>Ihre Auswahl für einzelne Themen.</p></div><span class="notification-count" data-notification-count>{{.NotificationEnabledCount}} von {{.NotificationEventCount}} aktiv</span></div>
              <p class="notification-paused-note">Die Themenauswahl bleibt gespeichert und gilt wieder, sobald Sie E-Mails aktivieren.</p>
              <section class="notification-group">
                <h3 class="notification-group-title">Haus &amp; Kommunikation</h3>
                {{range .NotificationEvents}}{{if or (eq .Key "announcement") (eq .Key "issue")}}<label class="notification-row"><span class="notification-row-copy"><strong>{{.Label}}</strong><span>{{.Description}}</span></span><span class="notification-switch"><input type="checkbox" name="events" value="{{.Key}}"{{if .Checked}} checked{{end}} data-notification-topic><span class="notification-switch-track"></span></span></label>{{end}}{{end}}
              </section>
              <section class="notification-group">
                <h3 class="notification-group-title">Entscheidungen &amp; Unterlagen</h3>
                {{range .NotificationEvents}}{{if or (eq .Key "vote") (eq .Key "document")}}<label class="notification-row"><span class="notification-row-copy"><strong>{{.Label}}</strong><span>{{.Description}}</span></span><span class="notification-switch"><input type="checkbox" name="events" value="{{.Key}}"{{if .Checked}} checked{{end}} data-notification-topic><span class="notification-switch-track"></span></span></label>{{end}}{{end}}
              </section>
              <section class="notification-group">
                <h3 class="notification-group-title">Zahlung &amp; Nutzung</h3>
                {{range .NotificationEvents}}{{if or (eq .Key "payment") (eq .Key "charging")}}<label class="notification-row"><span class="notification-row-copy"><strong>{{.Label}}</strong><span>{{.Description}}</span></span><span class="notification-switch"><input type="checkbox" name="events" value="{{.Key}}"{{if .Checked}} checked{{end}} data-notification-topic><span class="notification-switch-track"></span></span></label>{{end}}{{end}}
              </section>
            </div>
            <div class="actions">
              <a href="/app/settings">Abbrechen</a>
              <button class="button primary" type="submit">Benachrichtigungen speichern</button>
            </div>
          </form>
        </section>
        </div>
        <aside class="notification-aside" aria-label="Hinweise zur Zustellung">
          <details class="notification-details">
            <summary><strong id="notification-delivery-title">Empfängeradresse</strong><span>Ihre unveränderliche Anmeldeadresse</span></summary>
            <div class="notification-details-body">
            <div class="notification-address"><span>Empfängeradresse</span><strong>{{.Email}}</strong></div>
            <p>Die Adresse ist zugleich Ihre Anmeldung und lässt sich hier nicht ändern. Eine Änderung veranlasst die Hausverwaltung.</p>
            <a class="notification-note-link" href="/app/settings/profile">Profil ansehen</a>
            </div>
          </details>
          <details class="notification-details">
            <summary><strong id="notification-rules-title">Grundregeln</strong><span>Was unabhängig von Ihrer Auswahl gilt</span></summary>
            <div class="notification-details-body">
            <ul class="notification-rules">
              <li><strong>Anmeldelinks</strong><span>Einen Link, den Sie selbst anfordern, erhalten Sie immer – auch bei pausiertem Versand.</span></li>
              <li><strong>Keine Werbung</strong><span>Versendet wird ausschließlich, was dieses Haus betrifft.</span></li>
              <li><strong>Jederzeit änderbar</strong><span>Ihre Themenauswahl bleibt gespeichert und gilt wieder, sobald Sie den Versand aktivieren.</span></li>
              <li><strong>Im Portal vollständig</strong><span>Alles bleibt im Portal sichtbar, unabhängig davon, was per E-Mail hinausgeht.</span></li>
            </ul>
            </div>
          </details>
        </aside>
        </div>
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "parkingAdminNav"}}
  <nav class="pk-nav{{if not .CanManageParkingConfig}} pk-nav-one{{else if not .CanManageUsers}} pk-nav-three{{end}}" aria-label="Parkplatz-Verwaltung">
    {{if .CanManageUsers}}<a class="pk-nav-item{{if eq .ParkingSection "access"}} active{{end}}" href="/app/settings/parking-access"><span class="pk-nav-icon"><svg viewBox="0 0 24 24"><circle cx="8" cy="8" r="3"/><path d="M3 20a5 5 0 0 1 10 0M15 12h6M18 9v6"/></svg></span><strong>Zugriff</strong><span>Personen</span></a>{{end}}
    {{if .CanManageParkingConfig}}
    <a class="pk-nav-item{{if eq .ParkingSection "accounting"}} active{{end}}" href="/app/parking/settings?section=accounting"><span class="pk-nav-icon"><svg viewBox="0 0 24 24"><path d="M6 3h12v18H6zM9 8h6M9 12h6M9 16h4"/></svg></span><strong>Abrechnung</strong><span>Tarif &amp; Zahlung</span></a>
    <a class="pk-nav-item{{if eq .ParkingSection "charging"}} active{{end}}" href="/app/parking/settings?section=charging"><span class="pk-nav-icon"><svg viewBox="0 0 24 24"><path d="m13 2-7 12h6l-1 8 7-12h-6z"/></svg></span><strong>Laderegeln</strong><span>Automatik</span></a>
    <a class="pk-nav-item{{if eq .ParkingSection "telegram"}} active{{end}}" href="/app/parking/settings?section=telegram"><span class="pk-nav-icon"><svg viewBox="0 0 24 24"><path d="m3 11 18-8-6 18-3-7zM12 14l5-6"/></svg></span><strong>Telegram</strong><span>Verknüpfung</span></a>
    {{end}}
  </nav>
{{end}}

{{define "parkingAccessSettings"}}
{{template "appOpen" .}}
    <style>
      .parking-access { display: grid; gap: 22px; max-width: 920px; }
      .pk-nav { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 9px; }
      .pk-nav-three { grid-template-columns: repeat(3, minmax(0, 1fr)); }
      .pk-nav-one { grid-template-columns: minmax(180px, 260px); }
      .pk-nav-item { min-width: 0; min-height: 92px; padding: 13px 12px 11px; border: 1px solid var(--line); border-radius: 12px; background: var(--panel); color: var(--ink); text-decoration: none; display: grid; justify-items: start; align-content: center; gap: 2px; position: relative; transition: border-color .15s ease, background .15s ease; }
      .pk-nav-item:hover { border-color: rgba(200,153,63,.52); background: var(--panel-soft); }
      .pk-nav-item.active { border-color: rgba(200,153,63,.76); background: rgba(200,153,63,.07); box-shadow: inset 0 -3px var(--gold); }
      .pk-nav-item strong { font-size: 14px; }
      .pk-nav-item > span:last-child { color: var(--muted); font-size: 11.5px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 100%; }
      .pk-nav-icon { width: 22px; height: 22px; margin-bottom: 4px; color: var(--gold-ink); }
      .pk-nav-icon svg { width: 100%; height: 100%; fill: none; stroke: currentColor; stroke-width: 1.8; stroke-linecap: round; stroke-linejoin: round; }
      .parking-access .panel { padding: 0; overflow: visible; }
      .access-head { padding: 18px 20px 14px; border-bottom: 1px solid var(--line); display: flex; align-items: baseline; justify-content: space-between; gap: 16px; }
      .access-head h2 { margin: 0; font-size: 20px; }
      .access-head span { color: var(--muted); font-size: 12.5px; }
      .access-list { display: grid; }
      .access-row { display: grid; grid-template-columns: minmax(220px, 1fr) 126px 132px minmax(106px, auto); align-items: center; gap: 14px; padding: 15px 20px; border-top: 1px solid var(--line); }
      .access-row:first-child { border-top: 0; }
      .access-row:hover { background: rgba(250,247,239,.6); }
      .access-person { display: flex; align-items: center; gap: 11px; min-width: 0; }
      .access-avatar { flex: 0 0 auto; width: 38px; height: 38px; border-radius: 50%; background: var(--ink); color: #fff; display: grid; place-items: center; font-family: var(--font-serif); font-size: 15px; }
      .access-person-copy { display: grid; min-width: 0; }
      .access-person-copy strong { font-family: var(--font-serif); font-size: 16.5px; overflow-wrap: anywhere; }
      .access-person-copy span { color: var(--muted); font-size: 12.5px; overflow-wrap: anywhere; }
      .access-action { justify-self: end; margin: 0; }
      .access-action .button { min-width: 102px; min-height: 42px; margin: 0; }
      .access-fixed { position: relative; min-height: 44px; cursor: help; }
      .access-fixed svg { width: 13px; height: 13px; fill: none; stroke: currentColor; stroke-width: 1.8; stroke-linecap: round; stroke-linejoin: round; }
      .access-fixed::after { content: attr(data-help); position: absolute; right: 0; bottom: calc(100% + 7px); z-index: 10; width: max-content; max-width: 230px; border-radius: 8px; padding: 8px 10px; color: #fff; background: var(--ink); box-shadow: var(--shadow-dialog); font-size: 11.5px; font-weight: 650; line-height: 1.35; text-align: left; opacity: 0; visibility: hidden; transform: translateY(4px); transition: opacity .14s ease, transform .14s ease; pointer-events: none; }
      .access-fixed:hover::after, .access-fixed:focus-visible::after { opacity: 1; visibility: visible; transform: none; }
      .access-flash { margin: 0; padding: 11px 15px; border-radius: 9px; font-size: 13.5px; font-weight: 650; border: 1px solid transparent; }
      .access-flash.ok { background: rgba(47,107,74,.12); color: var(--leaf); border-color: rgba(47,107,74,.25); }
      .access-flash.warn { background: rgba(150,40,40,.08); color: #9a2b2b; border-color: rgba(150,40,40,.22); }
      @media (max-width: 680px) {
        .parking-access { gap: 17px; }
        .pk-nav { grid-template-columns: repeat(4, minmax(72px, 1fr)); gap: 6px; overflow-x: auto; padding: 0 1px 5px; scrollbar-width: none; }
        .pk-nav-three { grid-template-columns: repeat(3, minmax(92px, 1fr)); }
        .pk-nav-one { grid-template-columns: minmax(0, 1fr); }
        .pk-nav::-webkit-scrollbar { display: none; }
        .pk-nav-item { min-height: 78px; padding: 10px 8px 9px; }
        .pk-nav-item strong { font-size: 12.5px; }
        .pk-nav-item > span:last-child { display: none; }
        .pk-nav-icon { width: 19px; height: 19px; }
        .access-head { padding: 15px 16px 12px; }
        .access-row { grid-template-columns: minmax(0, 1fr) auto; gap: 8px 10px; padding: 14px 16px; }
        .access-person { grid-column: 1 / -1; }
        .access-row > .role-pill { justify-self: start; }
        .access-row > .pill { justify-self: end; }
        .access-action { grid-column: 1 / -1; justify-self: stretch; text-align: left; }
        .access-fixed::after { right: 0; }
        .access-action .button { width: 100%; min-height: 44px; }
      }
    </style>
    <main id="main-content" tabindex="-1" class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg><span>/</span><a href="/app/settings">Einstellungen</a><span>/</span><span>Parkplatz</span></span>
        <div class="page-actions"><a class="button ghost" href="/app/parking">Parkplatz öffnen</a></div>
      </div>
      <section class="page parking-access">
        <div>
          <h1>Parkplatz verwalten</h1>
          <p class="lede">Zugriff, Abrechnung und Laden – jeweils dort, wo es hingehört.</p>
        </div>
        {{template "parkingAdminNav" .}}
        {{if .AccessMsg}}<p class="access-flash{{if .AccessOK}} ok{{else}} warn{{end}}" role="status">{{.AccessMsg}}</p>{{end}}
        <section class="panel" aria-labelledby="access-title">
          <div class="access-head"><h2 id="access-title">Wer darf den Parkplatz nutzen?</h2><span>{{len .AccessRows}} Personen</span></div>
          {{if .HasAccessRows}}
            <div class="access-list">
              {{range .AccessRows}}
                <article class="access-row">
                  <span class="access-person"><span class="access-avatar">{{.Initials}}</span><span class="access-person-copy"><strong>{{.DisplayName}}</strong><span>{{.Email}}</span></span></span>
                  <span class="role-pill {{.RoleClass}}">{{.Role}}</span>
                  {{if and .Editable (or $.ServiceProviderAccessEnabled (ne .Role "Dienstleister")) (or $.IsAdmin (ne .Role "Admin"))}}
                    {{if .ParkingChecked}}<span class="pill ok">Freigegeben</span>{{else}}<span class="pill">Kein Zugriff</span>{{end}}
                    <form class="access-action" method="post" action="/app/settings/parking-access">
                      <input type="hidden" name="email" value="{{.Email}}">
                      <input type="hidden" name="parking" value="{{if .ParkingChecked}}0{{else}}1{{end}}">
                      <button class="button small{{if not .ParkingChecked}} primary{{end}}" type="submit">{{if .ParkingChecked}}Entziehen{{else}}Freigeben{{end}}</button>
                    </form>
                  {{else}}
                    <span class="pill{{if .ParkingChecked}} ok{{end}} access-fixed" tabindex="0" role="note" aria-label="{{if .ParkingChecked}}Freigegeben{{else}}Kein Zugriff{{end}}. Fest vergeben und hier nicht änderbar." data-help="Fest vergeben · hier nicht änderbar"><svg viewBox="0 0 24 24" aria-hidden="true"><rect x="5" y="10" width="14" height="10" rx="2"/><path d="M8 10V7a4 4 0 0 1 8 0v3"/></svg>{{if .ParkingChecked}}Freigegeben{{else}}Kein Zugriff{{end}}<span class="sr-only">. Aus Konfiguration. Schreibgeschützt.</span></span>
                  {{end}}
                </article>
              {{end}}
            </div>
          {{else}}
            {{template "emptyState" .AccessRowsEmpty}}
          {{end}}
        </section>
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "parkingSettings"}}
{{template "appOpen" .}}
    <style>
      .pk-set { display: grid; gap: 22px; max-width: 920px; }
      .pk-set .panel { display: grid; gap: 16px; }
      .pk-set .settings-card { max-width: none; }
      .pk-nav { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 9px; }
      .pk-nav-three { grid-template-columns: repeat(3, minmax(0, 1fr)); }
      .pk-nav-one { grid-template-columns: minmax(180px, 260px); }
      .pk-nav-item { min-width: 0; min-height: 92px; padding: 13px 12px 11px; border: 1px solid var(--line); border-radius: 12px; background: var(--panel); color: var(--ink); text-decoration: none; display: grid; justify-items: start; align-content: center; gap: 2px; position: relative; transition: border-color .15s ease, background .15s ease; }
      .pk-nav-item:hover { border-color: rgba(200,153,63,.52); background: var(--panel-soft); }
      .pk-nav-item.active { border-color: rgba(200,153,63,.76); background: rgba(200,153,63,.07); box-shadow: inset 0 -3px var(--gold); }
      .pk-nav-item strong { font-size: 14px; }
      .pk-nav-item > span:last-child { color: var(--muted); font-size: 11.5px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 100%; }
      .pk-nav-icon { width: 22px; height: 22px; margin-bottom: 4px; color: var(--gold-ink); }
      .pk-nav-icon svg { width: 100%; height: 100%; fill: none; stroke: currentColor; stroke-width: 1.8; stroke-linecap: round; stroke-linejoin: round; }
      .pk-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 14px; flex-wrap: wrap; }
      .pk-head h2 { margin: 0; }
      .pk-head .muted { margin: 5px 0 0; max-width: 52ch; }
      .pk-form { display: grid; gap: 14px; }
      .pk-fields { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px 14px; }
      .pk-fields.cols-3 { grid-template-columns: repeat(3, minmax(0, 1fr)); }
      .pk-group { border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel-soft); padding: 14px 14px 16px; display: grid; gap: 11px; }
      .pk-group-head { display: flex; align-items: baseline; gap: 10px; }
      .pk-group-head h3 { margin: 0; font-family: var(--font-serif); font-size: 15.5px; }
      .pk-group-head span { color: var(--muted); font-size: 12.5px; }
      .pk-unit { position: relative; }
      .pk-unit input { padding-right: 88px; text-align: right; font-variant-numeric: tabular-nums; }
      .pk-unit .unit { position: absolute; right: 12px; top: 50%; transform: translateY(-50%); font-size: 12px; font-weight: 600; letter-spacing: 0; text-transform: none; color: var(--soft); pointer-events: none; }
      .pk-toggles { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; }
      label.pk-toggle { display: grid; grid-template-columns: auto minmax(0, 1fr); align-items: start; gap: 4px 11px; border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel); padding: 12px 14px; cursor: pointer; text-transform: none; letter-spacing: 0; transition: border-color .15s ease, background .15s ease; }
      label.pk-toggle:hover { border-color: rgba(200,153,63,.45); }
      label.pk-toggle:has(input:checked) { border-color: rgba(200,153,63,.6); background: rgba(200,153,63,.08); }
      .pk-toggle input { width: 17px; height: 17px; min-height: 0; margin: 2px 0 0; accent-color: var(--gold); grid-row: 1 / span 2; }
      .pk-toggle strong { color: var(--ink); font-size: 14px; font-weight: 700; }
      .pk-toggle small { grid-column: 2; color: var(--muted); font-size: 12.5px; font-weight: 500; line-height: 1.4; }
      .pk-actions { display: flex; align-items: center; justify-content: flex-end; gap: 14px; flex-wrap: wrap; }
      .pk-actions .mini { margin-right: auto; }
      .pk-set .button { width: auto; margin: 0; padding: 11px 24px; }
      .pk-overview { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 10px; }
      .pk-overview-item { min-height: 78px; padding: 13px 14px; border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel-soft); display: grid; align-content: center; gap: 4px; }
      .pk-overview-item span { color: var(--muted); font-size: 12px; }
      .pk-overview-item strong { font-family: var(--font-serif); font-size: 17px; }
      .pk-flow { display: grid; gap: 0; border: 1px solid var(--line); border-radius: var(--radius-sm); overflow: hidden; }
      .pk-flow-row { min-height: 58px; padding: 11px 14px; display: flex; align-items: center; gap: 12px; border-top: 1px solid var(--line); color: var(--ink); text-decoration: none; background: var(--panel-soft); }
      .pk-flow-row:first-child { border-top: 0; }
      .pk-flow-step { flex: 0 0 auto; width: 26px; height: 26px; border-radius: 50%; display: grid; place-items: center; background: var(--ink); color: #fff; font-size: 12px; font-weight: 750; }
      .pk-flow-copy { display: grid; min-width: 0; }
      .pk-flow-copy strong { font-size: 14px; }
      .pk-flow-copy span { color: var(--muted); font-size: 12.5px; }
      .pk-flow-arrow { margin-left: auto; color: var(--gold-ink); font-size: 21px; }
      details.pk-disclosure { border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel-soft); overflow: hidden; }
      details.pk-disclosure > summary { min-height: 54px; padding: 12px 15px; cursor: pointer; list-style: none; display: flex; align-items: center; gap: 11px; font-weight: 700; color: var(--ink); }
      details.pk-disclosure > summary::-webkit-details-marker { display: none; }
      details.pk-disclosure > summary::after { content: "›"; margin-left: auto; color: var(--gold-ink); font-size: 23px; transform: rotate(90deg); transition: transform .15s ease; }
      details.pk-disclosure[open] > summary::after { transform: rotate(-90deg); }
      details.pk-disclosure[open] > summary { border-bottom: 1px solid var(--line); }
      .pk-disclosure-note { margin-left: auto; color: var(--muted); font-size: 12px; font-weight: 600; }
      details.pk-disclosure > summary::after { margin-left: 0; }
      .pk-disclosure-body { padding: 14px; display: grid; gap: 13px; }
      .pk-rule-summary { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 10px; }
      .pk-rule { min-height: 68px; padding: 12px 14px; border: 1px solid var(--line); border-radius: var(--radius-sm); background: rgba(255,255,255,.55); display: grid; gap: 3px; }
      .pk-rule strong { font-size: 13.5px; }
      .pk-rule span { color: var(--muted); font-size: 12.5px; line-height: 1.45; }
      .pk-safety { border-color: rgba(47,107,74,.28); background: rgba(47,107,74,.06); }
      .pk-strip { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; }
      .pk-strip .mini { display: inline-flex; align-items: center; min-height: 26px; padding: 2px 11px; border: 1px solid var(--line); border-radius: var(--radius-pill); background: var(--panel-soft); color: var(--muted); font-weight: 600; }
      .pk-events { display: grid; gap: 6px; max-height: 360px; overflow-y: auto; }
      .pk-event { display: grid; grid-template-columns: 96px auto minmax(0, 1fr); gap: 10px; align-items: baseline; border: 1px solid var(--line); border-radius: var(--radius-xs); background: var(--panel-soft); padding: 7px 11px; font-size: 13px; }
      .pk-event time { color: var(--soft); font-variant-numeric: tabular-nums; font-size: 12px; }
      .pk-event .pill { min-height: 22px; padding: 1px 9px; font-size: 11.5px; }
      .pk-inline { display: grid; grid-template-columns: minmax(0, 1fr) auto; gap: 10px; align-items: end; }
      .pk-set select { width: 100%; border: 1px solid #e2dac9; border-radius: var(--radius-xs); min-height: 42px; padding: 9px 12px; font: inherit; color: var(--ink); background: #fffefb; }
      .pk-code { border: 1px dashed rgba(200,153,63,.55); border-radius: var(--radius-sm); background: rgba(200,153,63,.07); padding: 14px 16px; display: grid; gap: 6px; text-align: center; justify-items: center; }
      .pk-code code { font-size: 22px; font-weight: 800; letter-spacing: .14em; color: var(--gold-ink); }
      .pk-code .mini { color: var(--muted); }
      .pk-chats { display: grid; gap: 8px; }
      .pk-chat { display: flex; align-items: center; gap: 12px; border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel-soft); padding: 10px 13px; }
      .pk-chat .pk-avatar { flex: 0 0 auto; width: 36px; height: 36px; border-radius: 50%; display: grid; place-items: center; font-size: 14px; font-weight: 700; font-family: var(--font-serif); color: #fff; background: var(--gold); border: 1px solid rgba(138,123,63,.5); }
      .pk-chat-meta { min-width: 0; }
      .pk-chat-meta strong { display: block; font-size: 14px; }
      .pk-chat-meta span { color: var(--muted); font-size: 12.5px; }
      .pk-chat form { margin-left: auto; }
      .pk-chat .button { min-height: 42px; padding: 8px 14px; font-size: 13px; }
      .pk-status.panel { padding: 0; gap: 0; }
      .pk-status > summary { background: var(--panel); }
      .pk-status .pk-disclosure-body { background: var(--panel-soft); }
      .pk-telegram-state { padding: 13px 15px; border-radius: var(--radius-sm); display: flex; align-items: center; gap: 11px; background: rgba(47,107,74,.07); border: 1px solid rgba(47,107,74,.22); }
      .pk-telegram-state.warn { background: rgba(150,40,40,.055); border-color: rgba(150,40,40,.18); }
      .pk-telegram-state-copy { display: grid; gap: 2px; }
      .pk-telegram-state-copy strong { font-size: 14px; }
      .pk-telegram-state-copy span { color: var(--muted); font-size: 12.5px; }
      @media (max-width: 620px) {
        .pk-set { gap: 17px; }
        .pk-nav { grid-template-columns: repeat(4, minmax(72px, 1fr)); gap: 6px; overflow-x: auto; padding: 0 1px 5px; scrollbar-width: none; }
        .pk-nav-three { grid-template-columns: repeat(3, minmax(92px, 1fr)); }
        .pk-nav-one { grid-template-columns: minmax(0, 1fr); }
        .pk-nav::-webkit-scrollbar { display: none; }
        .pk-nav-item { min-height: 78px; padding: 10px 8px 9px; }
        .pk-nav-item strong { font-size: 12.5px; }
        .pk-nav-item > span:last-child { display: none; }
        .pk-nav-icon { width: 19px; height: 19px; }
        .pk-fields, .pk-fields.cols-3, .pk-toggles { grid-template-columns: 1fr; }
        .pk-overview, .pk-rule-summary { grid-template-columns: 1fr; }
        .pk-inline { grid-template-columns: 1fr; }
        .pk-actions { justify-content: stretch; }
        .pk-set .button { width: 100%; }
        .pk-actions .mini { margin-right: 0; }
        .pk-disclosure-note { display: none; }
        .pk-event { grid-template-columns: 80px auto; }
        .pk-event span:last-child { grid-column: 1 / -1; }
        .pk-chat { align-items: flex-start; flex-wrap: wrap; }
        .pk-chat form { width: 100%; margin-left: 48px; }
      }
    </style>
    <main id="main-content" tabindex="-1" class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg><span>/</span><a href="/app/settings">Einstellungen</a><span>/</span><span>Parkplatz</span></span>
        <div class="page-actions"><a class="button ghost" href="/app/parking">Parkplatz öffnen</a></div>
      </div>
      <section class="page pk-set">
        <div>
          <h1>Parkplatz verwalten</h1>
          <p class="lede">Zugriff, Abrechnung und Laden – jeweils dort, wo es hingehört.</p>
        </div>
        {{template "parkingAdminNav" .}}

        {{if .SectionAccounting}}
        <section class="panel settings-card" id="abrechnung">
          <div class="pk-head">
            <div>
              <h2>Tarif &amp; Gültigkeit</h2>
              <p class="muted">Ein neuer Tarif gilt erst ab dem gewählten Datum. Bereits abgerechnete Monate bleiben unverändert.</p>
            </div>
            <span class="pill ok">Aktiv ab {{.Accounting.EffectiveFromLabel}}</span>
          </div>
          {{if .SettingsMsg}}<p class="flash {{if .SettingsOK}}ok{{end}}">{{.SettingsMsg}}</p>{{end}}
          <form class="pk-form" method="post" action="/app/parking/settings">
            <div class="pk-fields">
              <label for="effective_from">Gültig ab
                <input id="effective_from" type="date" name="effective_from" value="{{.Accounting.EffectiveFrom}}" autocomplete="off">
              </label>
              <label for="grid_fee_eur_per_kwh">Netzgebühr
                <span class="pk-unit"><input id="grid_fee_eur_per_kwh" type="text" inputmode="decimal" name="grid_fee_eur_per_kwh" value="{{.Accounting.GridFeeValue}}" autocomplete="off"><span class="unit">€/kWh</span></span>
              </label>
              <label for="base_fee_eur">Basisgebühr
                <span class="pk-unit"><input id="base_fee_eur" type="text" inputmode="decimal" name="base_fee_eur" value="{{.Accounting.BaseFeeValue}}" autocomplete="off"><span class="unit">€/Monat</span></span>
              </label>
              <label for="surplus_rate_eur_per_kwh">Überschusstarif
                <span class="pk-unit"><input id="surplus_rate_eur_per_kwh" type="text" inputmode="decimal" name="surplus_rate_eur_per_kwh" value="{{.Charging.SurplusRateValue}}" autocomplete="off"><span class="unit">€/kWh</span></span>
              </label>
            </div>
            <div class="pk-actions">
              {{if .Accounting.LastSampleLabel}}<span class="mini">Letzter Zählerwert: {{.Accounting.LastSampleLabel}}</span>{{end}}
              <button class="button primary" type="submit">Tarif speichern</button>
            </div>
          </form>
          {{if .Accounting.HasTariffs}}
            <details class="pk-disclosure">
              <summary><span>Frühere Tarife</span><span class="pk-disclosure-note">{{len .Accounting.Tariffs}} Einträge</span></summary>
              <div class="pk-disclosure-body"><div class="legend" aria-label="Tarifhistorie">
                {{range .Accounting.Tariffs}}<div><strong>{{.EffectiveFrom}}</strong><span>{{.GridFee}} · Basis {{.BaseFee}}</span></div>{{end}}
              </div></div>
            </details>
          {{end}}
        </section>

        <section class="panel settings-card" aria-labelledby="payment-flow-title">
          <div class="pk-head">
            <div>
              <h2 id="payment-flow-title">Monate &amp; Zahlungen</h2>
              <p class="muted">Vom Monatsbetrag bis zur Erinnerung in einer nachvollziehbaren Reihenfolge.</p>
            </div>
          </div>
          <div class="pk-flow">
            <div class="pk-flow-row"><span class="pk-flow-step">1</span><span class="pk-flow-copy"><strong>Tarif und Zeitraum</strong><span>Oben festlegen, ab wann neue Preise gelten.</span></span></div>
            <a class="pk-flow-row" href="/app/parking?view=months#monate"><span class="pk-flow-step">2</span><span class="pk-flow-copy"><strong>Zahlungsstände prüfen</strong><span>Monate öffnen, Zahlungseingang bestätigen oder Beleg ansehen.</span></span><span class="pk-flow-arrow">›</span></a>
          </div>
          <details class="pk-disclosure">
            <summary><span>Offene Zahlungen erinnern</span><span class="pk-disclosure-note">nur überfällige Monate</span></summary>
            <div class="pk-disclosure-body">
              <p class="muted">Es erhält nur eine Person eine Nachricht, wenn ein überfälliger Betrag offen ist und noch keine Erinnerung für diesen Monat versandt wurde.</p>
              <form method="post" action="/app/parking/reminders"><button class="button ghost" type="submit">Erinnerungen jetzt senden</button></form>
            </div>
          </details>
        </section>
        {{end}}

        {{if .SectionCharging}}
        <section class="panel settings-card" id="laderegelung">
          <div class="pk-head">
            <div>
              <h2>Laderegeln</h2>
              <p class="muted">Die Automatik nutzt Sonnenstrom und schützt Relais, Hausakku und Ladeelektronik.</p>
            </div>
            {{if .Charging.Enabled}}<span class="pill ok">Automatik aktiv</span>{{else}}<span class="pill">Automatik aus</span>{{end}}
          </div>
          {{if .ChargingMsg}}<p class="flash {{if .ChargingOK}}ok{{end}}">{{.ChargingMsg}}</p>{{end}}
          <form class="pk-form" method="post" action="/app/parking/charging/settings">
            <div class="pk-toggles">
              <label class="pk-toggle" for="controller_enabled">
                <input id="controller_enabled" type="checkbox" name="controller_enabled" value="1" {{if .Charging.Enabled}}checked{{end}}>
                <strong>Regler aktiv</strong>
                <small>Überwacht Akku und Einspeisung und startet Ladevorgänge.</small>
              </label>
              <label class="pk-toggle" for="shadow_mode">
                <input id="shadow_mode" type="checkbox" name="shadow_mode" value="1" {{if or .Charging.ShadowMode (not .Charging.Enabled)}}checked{{end}}>
                <strong>Testbetrieb</strong>
                <small>Nur beobachten und protokollieren — die Steckdose bleibt unberührt.</small>
              </label>
            </div>
            <div class="pk-rule-summary" aria-label="Zusammenfassung der Laderegeln">
              <div class="pk-rule"><strong>Start</strong><span>Bei mindestens {{.Charging.StartSocValue}} % Akku und {{.Charging.StartFeedInValue}} W Einspeisung.</span></div>
              <div class="pk-rule pk-safety"><strong>Stopp &amp; Schutz</strong><span>Unter {{.Charging.StopSocValue}} % Akku oder {{.Charging.StopFeedInValue}} W; Schaltpausen bleiben aktiv.</span></div>
            </div>
            <details class="pk-disclosure">
              <summary><span>Erweiterte Grenzwerte</span><span class="pk-disclosure-note">7 Werte</span></summary>
              <div class="pk-disclosure-body">
                <div class="pk-group">
                  <div class="pk-group-head"><h3>Starten</h3><span>beide Bedingungen müssen erfüllt sein</span></div>
                  <div class="pk-fields">
                    <label for="start_soc_percent">Akkustand mindestens
                      <span class="pk-unit"><input id="start_soc_percent" type="text" inputmode="decimal" name="start_soc_percent" value="{{.Charging.StartSocValue}}" autocomplete="off"><span class="unit">%</span></span>
                    </label>
                    <label for="start_feed_in_w">Einspeisung mindestens
                      <span class="pk-unit"><input id="start_feed_in_w" type="text" inputmode="numeric" name="start_feed_in_w" value="{{.Charging.StartFeedInValue}}" autocomplete="off"><span class="unit">W</span></span>
                    </label>
                  </div>
                </div>
                <div class="pk-group">
                  <div class="pk-group-head"><h3>Stoppen</h3><span>sobald eine Bedingung zutrifft</span></div>
                  <div class="pk-fields cols-3">
                    <label for="stop_soc_percent">Akkustand unter
                      <span class="pk-unit"><input id="stop_soc_percent" type="text" inputmode="decimal" name="stop_soc_percent" value="{{.Charging.StopSocValue}}" autocomplete="off"><span class="unit">%</span></span>
                    </label>
                    <label for="stop_feed_in_w">Einspeisung unter
                      <span class="pk-unit"><input id="stop_feed_in_w" type="text" inputmode="numeric" name="stop_feed_in_w" value="{{.Charging.StopFeedInValue}}" autocomplete="off"><span class="unit">W</span></span>
                    </label>
                    <label for="stop_delay_minutes">… und zwar durchgehend für
                      <span class="pk-unit"><input id="stop_delay_minutes" type="text" inputmode="numeric" name="stop_delay_minutes" value="{{.Charging.StopDelayValue}}" autocomplete="off"><span class="unit">Min.</span></span>
                    </label>
                  </div>
                </div>
                <div class="pk-group pk-safety">
                  <div class="pk-group-head"><h3>Schaltschutz</h3><span>schont Relais und Ladeelektronik</span></div>
                  <div class="pk-fields">
                    <label for="min_on_minutes">Mindest-Einschaltdauer
                      <span class="pk-unit"><input id="min_on_minutes" type="text" inputmode="numeric" name="min_on_minutes" value="{{.Charging.MinOnValue}}" autocomplete="off"><span class="unit">Min.</span></span>
                    </label>
                    <label for="min_off_minutes">Mindest-Pausendauer
                      <span class="pk-unit"><input id="min_off_minutes" type="text" inputmode="numeric" name="min_off_minutes" value="{{.Charging.MinOffValue}}" autocomplete="off"><span class="unit">Min.</span></span>
                    </label>
                  </div>
                </div>
              </div>
            </details>
            <div class="pk-actions">
              <button class="button primary" type="submit">Laderegeln speichern</button>
            </div>
          </form>
        </section>

        <details class="panel pk-disclosure pk-status" id="regler-status">
          <summary><span>Regler-Status &amp; Ereignisse</span><span class="pk-disclosure-note">{{.Charging.State.PhaseLabel}}</span></summary>
          <div class="pk-disclosure-body">
            <div class="pk-strip">
              {{if .Charging.State.SinceLabel}}<span class="mini">seit {{.Charging.State.SinceLabel}}</span>{{end}}
              {{if .Charging.State.PollLabel}}<span class="mini">HA-Poll {{.Charging.State.PollLabel}}</span>{{end}}
              {{if .Charging.State.ShadowPill}}<span class="pill">Testbetrieb</span>{{end}}
              {{if .Charging.State.ErrorDetail}}<span class="pill dringend">{{.Charging.State.ErrorDetail}}</span>{{end}}
            </div>
            {{if .Charging.HasEvents}}
              <div class="pk-events" aria-label="Ereignisprotokoll">
                {{range .Charging.Events}}<div class="pk-event"><time>{{.AtLabel}}</time><span class="pill {{.KindClass}}">{{.KindLabel}}</span><span>{{.Detail}}</span></div>{{end}}
              </div>
            {{else}}
              <p class="empty">Noch keine Ereignisse seit dem letzten Neustart. Die nächste Entscheidung erscheint automatisch hier.</p>
            {{end}}
          </div>
        </details>
        {{end}}

        {{if .SectionTelegram}}
        <section class="panel settings-card" id="telegram">
          <div class="pk-head">
            <div>
              <h2>Telegram verbinden</h2>
              <p class="muted">Eine Person auswählen, Einmal-Code erzeugen und direkt an den HAUSV-Bot senden.</p>
            </div>
          </div>
          {{if .Charging.Telegram.Configured}}
            <div class="pk-telegram-state"><span class="pill ok">Bot bereit</span><span class="pk-telegram-state-copy"><strong>Telegram ist eingerichtet</strong><span>Neue Verknüpfungen können sofort erstellt werden.</span></span></div>
          {{else}}
            <div class="pk-telegram-state warn"><span class="pill dringend">Nicht bereit</span><span class="pk-telegram-state-copy"><strong>Bot-Zugang fehlt am Server</strong><span>Bestehende Verknüpfungen bleiben gespeichert; neue Codes funktionieren erst nach der Einrichtung.</span></span></div>
          {{end}}
          {{if .ChargingMsg}}<p class="flash {{if .ChargingOK}}ok{{end}}" role="status">{{.ChargingMsg}}</p>{{end}}
          {{if .Charging.Telegram.PendingCode}}
            <div class="pk-code">
              <span class="mini">Code für {{.Charging.Telegram.CodeEmail}} — 24 h gültig, einmal verwendbar:</span>
              <code>/start {{.Charging.Telegram.PendingCode}}</code>
              <span class="mini">Diese Zeile per Telegram an den Bot senden.</span>
            </div>
          {{end}}
          <form class="pk-inline" method="post" action="/app/parking/charging/telegram/link">
            <label for="tg_email">Verknüpfungscode für
              <select id="tg_email" name="email">
                {{range .Charging.Telegram.LinkOptions}}<option value="{{.Email}}">{{.Label}} ({{.Email}})</option>{{end}}
              </select>
            </label>
            <button class="button primary" type="submit">Code erzeugen</button>
          </form>
          {{if .Charging.Telegram.HasChats}}
            <details class="pk-disclosure">
              <summary><span>Verknüpfte Chats</span><span class="pk-disclosure-note">{{len .Charging.Telegram.Chats}} aktiv</span></summary>
              <div class="pk-disclosure-body"><div class="pk-chats" aria-label="Verknüpfte Chats">
                {{range .Charging.Telegram.Chats}}
                  <div class="pk-chat">
                    <span class="pk-avatar">{{printf "%.1s" .DisplayName}}</span>
                    <span class="pk-chat-meta"><strong>{{.DisplayName}}</strong><span>{{.Email}} · seit {{.LinkedAt}}</span></span>
                    <form method="post" action="/app/parking/charging/telegram/unlink"><input type="hidden" name="chat_id" value="{{.ChatID}}"><button class="button" type="submit">Trennen</button></form>
                  </div>
                {{end}}
              </div></div>
            </details>
          {{else}}
            <p class="empty">Noch niemand verbunden. Der erste Schritt ist ein Einmal-Code.</p>
          {{end}}
        </section>
        {{end}}
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "userSettings"}}
{{template "appOpen" .}}
    <style>
      .users .panel { background: var(--panel); border: 1px solid var(--line); border-radius: 12px; padding: 22px; }
      .users .stack { display: grid; gap: 14px; }
      .users .panel-head { display: flex; align-items: baseline; justify-content: space-between; gap: 16px; border-bottom: 2px solid var(--ink); padding-bottom: 10px; margin-bottom: 14px; }
      .users .panel-head .kicker { border: 0; padding: 0; margin: 0; font-size: 12px; font-weight: 700; letter-spacing: .14em; text-transform: uppercase; color: var(--gold-ink); }
      .users .count { color: var(--soft); font-size: 12px; font-weight: 700; letter-spacing: .04em; font-variant-numeric: tabular-nums; }
      .users .muted { color: var(--muted); line-height: 1.55; }
      .users .roster-intro { margin: 2px 0 4px; }
      .users .disclosure { border: 1px solid var(--line); border-radius: 11px; background: var(--panel-soft); }
      .users .disclosure > summary { list-style: none; cursor: pointer; display: flex; align-items: center; gap: 11px; padding: 13px 16px; font-weight: 700; font-size: 14px; color: var(--ink); user-select: none; border-radius: 10px; }
      .users .disclosure > summary::-webkit-details-marker { display: none; }
      .users .disclosure > summary:hover { color: var(--gold-ink); }
      .users .disclosure > summary:focus-visible { outline: 2px solid var(--gold); outline-offset: -2px; }
      .users .disclosure[open] > summary { border-radius: 10px 10px 0 0; }
      .users .invite-plus { flex: 0 0 auto; width: 22px; height: 22px; border-radius: 6px; display: grid; place-items: center; background: var(--ink); color: #fff; }
      .users .summary-sub { margin-left: auto; font-weight: 600; font-size: 12.5px; color: var(--soft); }
      .users .disclosure-body { padding: 4px 16px 18px; }
      .users .invite-form { display: grid; grid-template-columns: repeat(12, 1fr); gap: 10px; }
      .users .invite-form .f-titel { grid-column: span 2; }
      .users .invite-form .f-vorname { grid-column: span 3; }
      .users .invite-form .f-nachname { grid-column: span 3; }
      .users .invite-form .f-email { grid-column: span 4; }
      .users .invite-form .f-role { grid-column: span 5; }
      .users .invite-form .f-submit { grid-column: span 7; }
      .users .invite-form .f-permissions, .users .dlg-form .f-permissions { grid-column: 1 / -1; }
      .users input, .users select { width: 100%; border: 1px solid #e2dac9; border-radius: 9px; padding: 12px; font: inherit; background: #fffefb; color: var(--ink); }
      .users .permission-fieldset { border: 1px solid var(--line); border-radius: 9px; padding: 12px; background: #fffefb; display: grid; gap: 10px; }
      .users .permission-fieldset legend { padding: 0 6px; color: var(--gold-ink); font-size: 11px; font-weight: 800; letter-spacing: .06em; text-transform: uppercase; }
      .users .preset-label { display: inline-flex; width: fit-content; min-height: 24px; align-items: center; border: 1px solid rgba(200,153,63,.28); border-radius: 999px; padding: 3px 9px; background: rgba(200,153,63,.12); color: #8a6a1f; font-size: 11.5px; font-weight: 800; }
      .users .permission-grid { display: grid; grid-template-columns: repeat(auto-fit,minmax(220px,1fr)); gap: 8px; }
      .users .permission-check { display: grid; grid-template-columns: auto minmax(0,1fr); gap: 9px; align-items: start; border: 1px solid var(--line); border-radius: 8px; padding: 10px; color: var(--ink); background: var(--panel-soft); text-transform: none; letter-spacing: 0; font-size: 13px; font-weight: 600; }
      .users .permission-check input, .users .dlg-form .permission-check input { width: auto; min-height: 0; margin: 2px 0 0; accent-color: var(--gold); grid-row: 1 / span 2; }
      .users .permission-check span { display: block; color: var(--muted); font-size: 12px; font-weight: 500; line-height: 1.35; margin-top: 3px; grid-column: 2; }
      .users .invite-form button { border: 1px solid var(--ink); background: var(--ink); border-radius: 10px; color: #fff; min-height: 44px; padding: 10px 13px; font: inherit; font-weight: 700; cursor: pointer; }
      .users .invite-form button:hover { background: #2c3329; }
      .users .invite-flash { margin: 0 0 12px; padding: 10px 13px; border-radius: 9px; font-size: 13.5px; font-weight: 600; border: 1px solid transparent; }
      .users .invite-flash.ok { background: rgba(47,107,74,.12); color: var(--leaf); border-color: rgba(47,107,74,.25); }
      .users .invite-flash.warn { background: rgba(200,153,63,.14); color: #93701d; border-color: rgba(200,153,63,.3); }
      .users .table-wrap { overflow: visible; }
      .users table { width: 100%; border-collapse: collapse; font-size: 15px; }
      .users thead th { color: var(--gold-ink); font-size: 11px; font-weight: 800; text-transform: uppercase; letter-spacing: .06em; text-align: left; padding: 4px 14px 12px; border-bottom: 2px solid var(--line); white-space: nowrap; }
      .users tbody td { padding: 15px 14px; border-bottom: 1px solid var(--line); vertical-align: middle; }
      .users tbody tr:last-child td { border-bottom: 0; }
      .users tbody tr { transition: background .12s ease; }
      .users tbody tr:hover { background: #faf6ec; }
      .users th:first-child, .users td:first-child { padding-left: 4px; }
      .users th:last-child, .users td:last-child { padding-right: 4px; }
      .users .person { display: flex; align-items: center; gap: 13px; min-width: 220px; }
      .users .avatar { flex: 0 0 auto; width: 40px; height: 40px; border-radius: 50%; display: grid; place-items: center; font-size: 14px; font-weight: 700; color: var(--gold-ink); background: rgba(200,153,63,.15); border: 1px solid rgba(200,153,63,.32); }
      .users .person-name { font-weight: 600; line-height: 1.25; }
      .users .person-mail { color: var(--muted); font-size: 13px; margin-top: 2px; word-break: break-word; }
      .users .chips { display: flex; flex-wrap: wrap; gap: 6px; }
      .users .chip { display: inline-flex; align-items: center; gap: 6px; border: 1px solid var(--line); background: var(--panel-soft); color: #6f6a5c; border-radius: 8px; padding: 4px 10px; font-size: 12.5px; font-weight: 600; white-space: nowrap; }
      .users .chip.plain { color: var(--soft); }
      .users .role-caps { display: flex; flex-wrap: wrap; gap: 5px; margin-top: 6px; }
      .users .role-cap { display: inline-flex; align-items: center; min-height: 22px; border: 1px solid var(--line); border-radius: 999px; padding: 2px 8px; color: var(--muted); background: var(--panel-soft); font-size: 11.5px; font-weight: 700; white-space: nowrap; }
      .users .pill { display: inline-flex; align-items: center; gap: 7px; border-radius: 999px; min-height: 28px; padding: 4px 12px; font-size: 13px; font-weight: 700; white-space: nowrap; border: 1px solid transparent; }
      .users .pill .dot { width: 7px; height: 7px; border-radius: 50%; background: currentColor; opacity: .9; }
      .users .pill.role-admin { background: rgba(200,153,63,.16); color: #8a6a1f; border-color: rgba(200,153,63,.28); }
      .users .pill.role-manager { background: rgba(32,37,31,.08); color: var(--ink); border-color: rgba(32,37,31,.15); }
      .users .pill.role-owner { background: rgba(47,107,74,.12); color: var(--leaf); border-color: rgba(47,107,74,.22); }
      .users .pill.role-renter { background: rgba(76,103,138,.11); color: #365475; border-color: rgba(76,103,138,.22); }
      .users .pill.role-resident { background: rgba(47,107,74,.11); color: var(--leaf); border-color: rgba(47,107,74,.2); }
      .users .pill.role-beirat { background: rgba(32,37,31,.06); color: #4b4f45; border-color: rgba(32,37,31,.12); }
      .users .pill.role-service { background: rgba(150,40,40,.08); color: #8c3434; border-color: rgba(150,40,40,.2); }
      .users .pill.status-active { background: rgba(47,107,74,.12); color: var(--leaf); }
      .users .pill.status-pending { background: rgba(200,153,63,.14); color: #93701d; }
      .users .pill.status-off { background: rgba(150,40,40,.10); color: #8c3434; }
      .users .last-seen { display: block; color: var(--soft); font-size: 11.5px; margin-top: 5px; white-space: nowrap; }
      .users th.col-role, .users td.col-role, .users th.col-status, .users td.col-status { white-space: nowrap; }
      .users .th-label { display: inline-flex; align-items: center; gap: 6px; }
      .users .info { position: relative; display: inline-flex; }
      .users .info-btn { width: 17px; height: 17px; border-radius: 50%; border: 1px solid var(--gold-ink); background: transparent; color: var(--gold-ink); display: grid; place-items: center; padding: 0; cursor: help; }
      .users .info-btn:hover, .users .info-btn:focus-visible { background: var(--gold-ink); color: #fff; outline: none; }
      .users .info-btn:focus-visible { box-shadow: 0 0 0 2px rgba(200,153,63,.4); }
      .users .popup { position: absolute; top: calc(100% + 11px); left: -72px; width: min(820px, calc(100vw - 48px)); background: var(--panel); border: 1px solid var(--line); border-radius: 14px; box-shadow: 0 20px 46px rgba(32,37,31,.17), 0 3px 9px rgba(32,37,31,.05); padding: 18px 22px 20px; z-index: 8; opacity: 0; visibility: hidden; transform: translateY(-6px); transition: opacity .16s ease, transform .16s ease; text-transform: none; letter-spacing: normal; }
      .users .popup::before { content: ""; position: absolute; top: -6px; left: 82px; width: 12px; height: 12px; background: var(--panel); border-left: 1px solid var(--line); border-top: 1px solid var(--line); border-radius: 3px 0 0 0; transform: rotate(45deg); }
      .users .popup::after { content: ""; position: absolute; top: -15px; left: 0; right: 0; height: 15px; }
      .users .info:hover .popup, .users .info:focus-within .popup { opacity: 1; visibility: visible; transform: translateY(0); }
      .users .popup-title { display: block; font-family: var(--font-serif); font-weight: 600; font-size: 18px; color: var(--ink); padding-bottom: 13px; border-bottom: 1px solid var(--line); }
      .users .popup-grid { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 0 24px; }
      .users .popup .permission { display: block; min-width: 0; padding: 14px 0; border-top: 1px solid var(--line); }
      .users .popup-grid .permission:nth-child(-n+3) { border-top: 0; }
      .users .popup .permission strong { display: block; font-family: var(--font-serif); font-weight: 600; font-size: 15px; color: var(--ink); margin-bottom: 4px; line-height: 1.2; }
      .users .popup .permission .muted { display: block; max-width: 100%; font-size: 13px; font-weight: 400; color: var(--muted); line-height: 1.42; white-space: normal; overflow-wrap: anywhere; }
      .users .rdot { display: inline-block; width: 8px; height: 8px; border-radius: 50%; margin-right: 9px; vertical-align: middle; }
      .users .rdot.admin { background: var(--gold); }
      .users .rdot.manager { background: var(--ink); }
      .users .rdot.owner { background: var(--leaf); }
      .users .rdot.renter { background: #365475; }
      .users .rdot.resident { background: var(--leaf); }
      .users .rdot.beirat { background: #8a8d80; }
      .users .rdot.service { background: #8c3434; }
      .users .rdot.right { background: var(--gold-light); box-shadow: inset 0 0 0 1px var(--gold); }
      .users .col-actions { width: 44px; }
      .users td.col-actions { text-align: right; }
      .users .row-edit { border: 1px solid transparent; background: transparent; border-radius: 8px; width: 44px; height: 44px; display: inline-grid; place-items: center; color: var(--soft); cursor: pointer; padding: 0; }
      .users .row-edit:hover { border-color: var(--line); background: var(--panel-soft); color: var(--gold-ink); }
      .users .row-edit svg { stroke: currentColor; fill: none; stroke-width: 1.8; stroke-linecap: round; stroke-linejoin: round; }
      .users .edit-dialog { position: relative; width: min(440px, 92vw); border: 1px solid var(--line); border-radius: 14px; padding: 22px; background: var(--panel); color: var(--ink); box-shadow: 0 30px 80px rgba(32,37,31,.32); }
      .users .edit-dialog::backdrop { background: rgba(32,37,31,.42); }
      .users .edit-dialog h2 { margin: 0 0 4px; font-family: var(--font-serif); font-weight: 600; font-size: 19px; }
      .users .edit-dialog .dlg-sub { color: var(--muted); font-size: 13px; margin: 0 0 16px; word-break: break-word; }
      .users .dlg-x { position: absolute; top: 12px; right: 12px; }
      .users .dlg-x button { border: 0; background: transparent; font-size: 22px; line-height: 1; color: var(--soft); cursor: pointer; padding: 2px 6px; }
      .users .dlg-x button:hover { color: var(--ink); }
      .users .dlg-form { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
      .users .dlg-form input, .users .dlg-form select, .users .dlg-form button { grid-column: 1 / -1; }
      .users .dlg-form .f-vorname, .users .dlg-form .f-nachname { grid-column: span 1; }
      .users .dlg-form button { border: 1px solid var(--ink); background: var(--ink); color: #fff; border-radius: 10px; min-height: 44px; padding: 10px 13px; font: inherit; font-weight: 700; cursor: pointer; }
      .users .dlg-form button:hover { background: #2c3329; }
      .users .dlg-delete { margin-top: 16px; padding-top: 14px; border-top: 1px solid var(--line); display: flex; justify-content: space-between; align-items: center; gap: 12px; }
      .users .dlg-delete span { color: var(--muted); font-size: 12.5px; }
      .users .dlg-delete .danger { border: 1px solid rgba(150,40,40,.32); background: rgba(150,40,40,.07); color: #9a2b2b; border-radius: 10px; min-height: 40px; padding: 8px 15px; font: inherit; font-weight: 700; cursor: pointer; }
	      .users .dlg-delete .danger:hover { background: rgba(150,40,40,.14); }
	      @media (max-width: 1340px) {
	        .users, .users .panel, .users .stack, .users .disclosure, .users .disclosure-body, .users .invite-form, .users .table-wrap { min-width: 0; max-width: 100%; }
	        .users .panel { padding: 18px; }
	        .users .panel-head { display: grid; grid-template-columns: minmax(0,1fr) auto; align-items: end; gap: 10px; }
	        .users .disclosure > summary { display: grid; grid-template-columns: 22px minmax(0,1fr); align-items: center; gap: 10px; }
	        .users .summary-sub { grid-column: 2; margin-left: 0; }
	        .users .invite-form { grid-template-columns: 1fr; }
	        .users .table-wrap { overflow: visible; border: 0; background: transparent; }
	        .users table { min-width: 0; }
	        .users table, .users thead, .users tbody, .users tr, .users td { display: block; width: 100%; }
	        .users thead, .users thead tr, .users thead th { display: none; }
	        .users tbody tr { position: relative; border: 1px solid var(--line); border-radius: 11px; padding: 16px; margin-bottom: 12px; background: var(--panel); box-shadow: 0 8px 20px rgba(32,37,31,.04); }
	        .users tbody tr:hover { background: var(--panel); }
	        .users tbody td { border: 0; padding: 0; }
	        .users .person { min-width: 0; padding-right: 42px; align-items: flex-start; }
	        .users tbody td.col-person { margin-bottom: 12px; }
	        .users tbody td[data-label]:not(.col-person):not(.col-actions) { display: block; padding: 10px 0 0; margin-top: 10px; border-top: 1px dashed var(--line); white-space: normal; }
	        .users tbody td[data-label]:not(.col-person):not(.col-actions)::before { content: attr(data-label); display: block; color: var(--gold-ink); font-size: 10.5px; font-weight: 800; letter-spacing: .06em; text-transform: uppercase; margin-bottom: 7px; }
	        .users .chips, .users .role-caps { min-width: 0; width: 100%; }
	        .users .chips { gap: 7px; }
	        .users .role-caps { margin-top: 8px; gap: 6px; }
	        .users .chip, .users .role-cap { white-space: nowrap; overflow-wrap: normal; word-break: normal; max-width: 100%; }
	        .users .role-cap { min-height: 25px; padding: 3px 9px; }
	        .users .last-seen { white-space: normal; }
	        .users td.col-actions { position: absolute; top: 12px; right: 12px; width: auto; text-align: right; }
	        .users td.col-actions::before { display: none; }
	        .users .row-edit { border-color: var(--line); background: var(--panel-soft); color: var(--gold-ink); }
	        .users .invite-form > * { grid-column: 1 / -1 !important; }
      }
      @media (max-width: 1340px) { .users .popup { left: -96px; width: min(620px, calc(100vw - 32px)); } .users .popup-grid { grid-template-columns: 1fr; } .users .popup-grid .permission { border-top: 1px solid var(--line); } .users .popup-grid .permission:first-child { border-top: 0; } }
      /* Ab Tabellenbreite hängt die Rollen-Erklärung an der Tabelle statt am
         Info-Knopf. Mit fester Breite ragte sie bei 1024px über den rechten
         Rand hinaus und erzeugte einen horizontalen Seitenlauf. */
      @media (min-width: 1121px) {
        .users .table-wrap { position: relative; }
        .users .info { position: static; }
        .users .popup { top: auto; left: 0; right: 0; width: auto; margin-top: 26px; }
        .users .popup::before { display: none; }
      }
      /* Zwischen Tabelle und Handy: die Zeile wird zur Karte, die Felder stehen
         aber weiter nebeneinander – sonst würde die Liste sechs Bildschirme lang. */
      @media (max-width: 1120px) {
        /* Ein beschriftetes, aber leeres Feld liest sich wie ein Fehler. Der
           Standard-Chip ist ausgeblendet, deshalb entfällt in der Karte auch
           seine Beschriftung. */
        .users tbody td.col-secondary:not(:has(.chip:not(.plain))) { display: none; }
        .users tbody td.col-auth:not(:has(.chip)) { display: none; }
      }
      @media (min-width: 761px) and (max-width: 1340px) {
        .users tbody tr { display: grid; grid-template-columns: minmax(0,2.1fr) minmax(0,1fr) minmax(0,1.1fr) auto; gap: 10px 16px; align-items: start; padding: 15px 16px; }
        .users tbody td.col-person { grid-column: 1; grid-row: 1 / span 2; margin-bottom: 0; }
        .users .person { padding-right: 0; }
        .users tbody td[data-label]:not(.col-person):not(.col-actions) { margin-top: 0; padding-top: 0; border-top: 0; }
        .users tbody td.col-role { grid-column: 2; grid-row: 1; }
        .users tbody td.col-status { grid-column: 3; grid-row: 1; }
        .users tbody td.col-secondary { grid-column: 2; grid-row: 2; }
        .users tbody td.col-auth { grid-column: 3; grid-row: 2; }
        .users td.col-actions { position: static; grid-column: 4; grid-row: 1; align-self: center; width: auto; }
      }
      .users .users-heading .lede { max-width: 680px; margin-bottom: 0; }
      .users .panel { border-radius: 16px; padding: 24px; box-shadow: var(--shadow-panel); }
      .users .panel-head { margin: 0; padding: 0 0 12px; border-bottom: 1px solid var(--line); }
      .users .access-metrics { display: grid; grid-template-columns: repeat(4,minmax(0,1fr)); border: 1px solid var(--line); border-radius: 12px; overflow: hidden; background: var(--panel-soft); }
      .users .access-metrics > div { position: relative; display: grid; gap: 2px; padding: 13px 16px; border-left: 1px solid var(--line); }
      .users .access-metrics > div:first-child { border-left: 0; }
      .users .access-metrics strong { font-family: var(--font-serif); font-size: 23px; font-weight: 600; line-height: 1; }
      .users .access-metrics span { color: var(--muted); font-size: 12px; font-weight: 700; }
      .users .access-metrics .metric-active strong { color: var(--leaf); }
      .users .access-metrics .metric-pending strong { color: #93701d; }
      .users .access-metrics .metric-disabled strong { color: #8c3434; }
      .users .invite-bar { border-color: var(--ink); background: var(--ink); }
      .users .invite-bar > summary { min-height: 52px; color: #fff; padding: 14px 18px; }
      .users .invite-bar > summary:hover { color: var(--gold-light); }
      .users .invite-bar .invite-plus { background: #fff; color: var(--ink); }
      .users .invite-bar .summary-sub { color: rgba(255,255,255,.68); }
      .users .invite-bar[open] { border-color: var(--line); background: var(--panel-soft); }
      .users .invite-bar[open] > summary { background: var(--ink); color: #fff; }
      .users .disclosure-body { padding: 18px; }
      .users .invite-intro { margin: 0 0 14px; font-size: 13.5px; }
      .users .invite-form { grid-template-columns: repeat(2,minmax(0,1fr)); gap: 12px; }
      .users .core-field, .users .dialog-core-field { display: grid; gap: 6px; color: var(--gold-ink); font-size: 11px; font-weight: 800; letter-spacing: .07em; text-transform: uppercase; }
      .users .service-note { grid-column: 1 / -1; margin: -2px 0 0; color: #8a6a1f; font-size: 12.5px; line-height: 1.4; }
      .users .form-disclosure { grid-column: 1 / -1; border: 1px solid var(--line); border-radius: 10px; background: #fffefb; }
      .users .form-disclosure > summary, .users .danger-zone > summary { list-style: none; cursor: pointer; display: flex; justify-content: space-between; align-items: center; min-height: 44px; gap: 12px; padding: 11px 13px; color: var(--ink); font-size: 13px; font-weight: 750; }
      .users .form-disclosure > summary::-webkit-details-marker, .users .danger-zone > summary::-webkit-details-marker { display: none; }
      .users .form-disclosure > summary::after, .users .danger-zone > summary::after { content: "+"; margin-left: auto; color: var(--gold-ink); font-size: 18px; font-weight: 500; }
      .users .form-disclosure[open] > summary::after, .users .danger-zone[open] > summary::after { content: "−"; }
      .users .form-disclosure > summary span { color: var(--soft); font-size: 11.5px; font-weight: 650; }
      .users .optional-grid { display: grid; grid-template-columns: .7fr 1fr 1fr; gap: 10px; padding: 0 12px 12px; }
      .users .optional-grid .f-email, .users .optional-grid .dlg-hint { grid-column: 1 / -1; }
      .users .optional-stack { display: grid; gap: 10px; padding: 0 12px 12px; }
      .users .invite-form .f-submit { grid-column: 1 / -1; justify-self: start; min-width: 220px; }
      .users .unit-link-note { grid-column: 1 / -1; margin: -2px 0 0; color: var(--muted); font-size: 12.5px; }
      .users .unit-link-note a, .users .unit-context a { min-height: 44px; display: inline-flex; align-items: center; color: var(--gold-ink); font-weight: 750; }
      .users .list-heading { display: flex; justify-content: space-between; align-items: baseline; gap: 16px; padding: 8px 2px 0; }
      .users .list-heading strong { font-family: var(--font-serif); font-size: 20px; font-weight: 600; }
      .users .list-heading span { color: var(--soft); font-size: 12px; font-weight: 650; }
      .users .role-caps { display: none; }
      .users .chip.plain { display: none; }
      .users .person-units { display: flex; flex-wrap: wrap; gap: 5px; margin-top: 6px; }
      .users .person-units span { color: var(--gold-ink); font-size: 11.5px; font-weight: 700; }
      .users tbody tr.needs-attention { background: rgba(200,153,63,.045); }
      .users tbody tr.needs-attention.disabled { background: rgba(150,40,40,.035); }
      .users .col-actions { width: 112px; }
      .users .row-edit { width: auto; min-width: 44px; min-height: 44px; gap: 7px; padding: 0 10px; border-color: var(--line); background: var(--panel-soft); color: var(--gold-ink); font-weight: 750; }
      .users .edit-label { font-size: 12px; }
      .users .edit-dialog { width: min(520px, calc(100vw - 32px)); max-height: calc(100dvh - 32px); overflow: auto; padding: 24px; }
      .users .edit-dialog h2 { font-size: 25px; }
      .users .edit-dialog .dlg-sub { margin-bottom: 14px; }
      .users .dlg-form { display: grid; grid-template-columns: 1fr; gap: 10px; }
      .users .dlg-form > * { grid-column: 1 / -1; }
      .users .dialog-core-field { order: 1; }
      .users .identity-section { order: 2; }
      .users .unit-section { order: 3; }
      .users .access-section { order: 4; }
      .users .status-field { order: 5; }
      .users .save-access { order: 6; }
      .users .dialog-disclosure .optional-grid { grid-template-columns: .65fr 1fr 1fr; }
      .users .unit-context { display: grid; gap: 10px; padding: 0 12px 12px; }
      .users .unit-context p { margin: 0; color: var(--muted); font-size: 12.5px; }
      .users .unit-context a { font-size: 12.5px; }
      .users .status-field { padding: 10px; }
      .users .status-field .permission-check { background: rgba(150,40,40,.035); }
      .users .save-access { margin-top: 2px; }
      .users .danger-zone { margin-top: 12px; border-top: 1px solid var(--line); }
      .users .danger-zone > summary { padding-inline: 2px; color: #8c3434; }
      .users .danger-zone > summary::after { color: #8c3434; }
      .users .dlg-delete { margin-top: 0; padding: 2px 0 0; border-top: 0; align-items: end; }
      .users .dlg-delete span { max-width: 270px; line-height: 1.45; }
      @media (max-width: 760px) {
        .users .users-heading h1 { font-size: clamp(38px,11vw,48px); }
        .users .users-heading .lede { display: none; }
        .users .panel { padding: 14px; border-radius: 14px; gap: 12px; }
        .users .panel-head { padding-bottom: 10px; }
        .users .access-metrics { grid-template-columns: repeat(3,minmax(0,1fr)); }
        .users .access-metrics > div { padding: 11px 8px; text-align: center; }
        .users .access-metrics > div:first-child { display: none; }
        .users .access-metrics > div:nth-child(2) { border-left: 0; }
        .users .access-metrics strong { font-size: 21px; }
        .users .invite-bar > summary { grid-template-columns: 24px minmax(0,1fr); min-height: 54px; }
        .users .invite-bar .summary-sub { display: none; }
        .users .disclosure-body { padding: 14px; }
        .users .invite-form { grid-template-columns: 1fr; }
        .users .invite-form .f-submit { width: 100%; min-width: 0; justify-self: stretch; }
        .users .optional-grid, .users .dialog-disclosure .optional-grid { grid-template-columns: 1fr; }
        .users .optional-grid > * { grid-column: 1 / -1; }
        .users .list-heading { padding-top: 6px; }
        .users .list-heading span { display: none; }
        .users table, .users tbody { display: grid; gap: 9px; }
        .users tbody tr { display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 9px 10px; margin: 0; padding: 13px; border-radius: 12px; box-shadow: none; }
        .users tbody td { width: auto; min-width: 0; }
        .users tbody td.col-person { grid-column: 1; grid-row: 1; margin: 0; }
        .users .person { min-width: 0; padding-right: 0; gap: 10px; }
        .users .avatar { width: 38px; height: 38px; }
        .users .person-name { font-size: 14px; }
        .users .person-mail { font-size: 12px; }
        .users .person-units { margin-top: 4px; }
        .users tbody td.col-role, .users tbody td.col-status { display: flex !important; align-items: center; padding: 0 !important; margin: 0 !important; border: 0 !important; }
        .users tbody td.col-role { grid-column: 1; grid-row: 2; }
        .users tbody td.col-status { grid-column: 2; grid-row: 2; justify-content: end; }
        .users tbody td.col-role::before, .users tbody td.col-status::before { display: none !important; }
        .users tbody td.col-secondary, .users tbody td.col-auth { display: none !important; }
        .users .pill { min-height: 26px; padding: 3px 9px; font-size: 12px; }
        .users .last-seen { display: none; }
        .users td.col-actions { position: static; grid-column: 2; grid-row: 1; align-self: center; width: auto; }
        .users .row-edit { min-height: 44px; padding-inline: 9px; }
        .users .edit-label { display: none; }
        .users .edit-dialog { width: calc(100vw - 24px); max-height: calc(100dvh - 24px); padding: 18px; border-radius: 16px; }
        .users .edit-dialog h2 { padding-right: 24px; font-size: 23px; }
        .users .dialog-disclosure .optional-grid, .users .optional-stack { padding-inline: 10px; }
        .users .dlg-delete { display: grid; align-items: start; }
        .users .dlg-delete span { max-width: none; }
      }
      @media (max-width: 350px) {
        .users tbody tr { position: relative; }
        .users tbody td.col-person { grid-column: 1 / -1; }
        .users .person { display: grid; grid-template-columns: 38px minmax(0,1fr); column-gap: 10px; row-gap: 4px; }
        .users .person > div { display: contents; }
        .users .avatar { grid-column: 1; grid-row: 1; }
        .users .person-name { grid-column: 2; grid-row: 1; align-self: center; padding-right: 48px; }
        .users .person-mail, .users .person-units { grid-column: 1 / -1; min-width: 0; margin-top: 0; word-break: normal; overflow-wrap: anywhere; }
        .users td.col-actions { position: absolute; top: 11px; right: 11px; }
      }
    </style>
    <script src="/assets/users.js?v={{.AssetVersion}}" defer></script>
    <main id="main-content" tabindex="-1" class="app-main">
      <div class="content-top"><span class="crumb"><svg viewBox="0 0 24 24"><path d="M8.5 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6z"/><path d="M3.5 20a5 5 0 0 1 10 0"/><path d="M16 11.5a2.5 2.5 0 1 0 0-5"/><path d="M17 15a4 4 0 0 1 3.5 4"/></svg>Benutzer &amp; Rechte</span></div>
      <section class="page users">
	        <div class="users-heading">
	          <h1>Benutzer &amp; Rechte</h1>
	          <p class="lede">Zugänge einladen, prüfen und sicher verwalten.</p>
	        </div>
    <section class="panel stack">
      <div class="panel-head">
        <span class="kicker">Zugänge</span>
        <span class="count">{{.UserCount}} {{if eq .UserCount 1}}Person{{else}}Personen{{end}}</span>
      </div>

      <div class="access-metrics" aria-label="Zugangsübersicht">
        <div><strong>{{.UserCount}}</strong><span>Gesamt</span></div>
        <div class="metric-active"><strong>{{.ActiveUserCount}}</strong><span>Aktiv</span></div>
        <div class="metric-pending"><strong>{{.InvitedUserCount}}</strong><span>Eingeladen</span></div>
        <div class="metric-disabled"><strong>{{.DisabledUserCount}}</strong><span>Deaktiviert</span></div>
      </div>

      {{if .InviteMsg}}<p class="invite-flash{{if .InviteOK}} ok{{else}} warn{{end}}" role="status">{{.InviteMsg}}</p>{{end}}

      <details id="invite" class="disclosure invite-bar">
        <summary>
          <span class="invite-plus"><svg viewBox="0 0 24 24" width="13" height="13" aria-hidden="true"><path d="M12 5.5v13M5.5 12h13" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round"/></svg></span>
          Person einladen
          <span class="summary-sub">Neuen Zugang anlegen</span>
        </summary>
        <div class="disclosure-body">
          <p class="muted invite-intro">E-Mail und Rolle genügen. Name und besondere Zugangsregeln sind optional.</p>
          <form class="invite-form" method="post" action="/app/settings/users">
            <label class="core-field f-email"><span>E-Mail-Adresse</span><input type="email" name="email" placeholder="name@example.com" aria-label="E-Mail-Adresse" autocomplete="email" required></label>
            <label class="core-field"><span>Rolle</span><select class="f-role" name="role" aria-label="Rolle">
                <option value="Mieter" data-preset-label="Standardzugriff" data-preset-permissions="">Mieter</option>
                <option value="Eigentümer" data-preset-label="Eigentümerzugriff" data-preset-permissions="">Eigentümer</option>
                <option value="Beirat" data-preset-label="Beiratszugriff" data-preset-permissions="">Beirat</option>
	                <option value="Verwalter" data-preset-label="Verwalterzugriff" data-preset-permissions="">Verwalter</option>
	                {{if .IsAdmin}}<option value="Admin" data-preset-label="Adminzugriff" data-preset-permissions="parking">Admin</option>{{end}}
	                {{if .ServiceProviderAccessEnabled}}<option value="Dienstleister" data-preset-label="Nur zugewiesene Anliegen" data-preset-permissions="">Dienstleister</option>{{end}}
	                <option value="Bewohner" data-preset-label="Bewohnerzugriff" data-preset-permissions="">Bewohner</option>
              </select></label>
            {{if not .ServiceProviderAccessEnabled}}<p class="service-note">Dienstleister-Zugänge können noch nicht angelegt oder geändert werden.</p>{{end}}
            <details class="form-disclosure f-person-details">
              <summary>Name ergänzen <span>optional</span></summary>
              <div class="optional-grid">
                <input class="f-titel" type="text" name="title" placeholder="Titel" aria-label="Titel">
                <input class="f-vorname" type="text" name="first_name" placeholder="Vorname" aria-label="Vorname">
                <input class="f-nachname" type="text" name="last_name" placeholder="Nachname" aria-label="Nachname">
              </div>
            </details>
            <details class="form-disclosure f-access-details">
              <summary>Zugang anpassen <span>optional</span></summary>
              <div class="optional-stack">
                <fieldset class="permission-fieldset f-permissions">
                  <legend>Sonderrechte</legend>
                  <span class="preset-label" data-preset-label>Standardzugriff</span>
                  <div class="permission-grid">
                    <label class="permission-check"><input type="checkbox" name="permissions" value="parking" data-permission="parking"><strong>Parkplatznutzung</strong><span>Privater Bereich für Stellplatz- und Ladeabrechnung.</span></label>
                    <label class="permission-check"><input type="checkbox" name="permissions" value="energy-caretaker" data-permission="energy-caretaker"><strong>Technische Vertrauensperson</strong><span>Darf Energiedaten ansehen und die Einrichtung unterstützen, aber keine Steuerung aktivieren.</span></label>
                  </div>
                </fieldset>
                <fieldset class="permission-fieldset f-permissions">
                  <legend>Anmeldung</legend>
                  <div class="permission-grid">
                    <label class="permission-check"><input type="checkbox" name="auth_methods" value="email" checked><strong>E-Mail-Link</strong><span>Anmeldung per Magic-Link an die E-Mail-Adresse.</span></label>
                    <label class="permission-check"><input type="checkbox" name="auth_methods" value="oidc" checked><strong>Sichere Anmeldung</strong><span>Anmeldung über den zentralen Hauszugang.</span></label>
                  </div>
                </fieldset>
              </div>
            </details>
            <button class="f-submit" type="submit">Einladung senden</button>
            <p class="unit-link-note">Danach bei Bedarf: <a href="/app/settings/building#units">Einheit verknüpfen</a></p>
          </form>
        </div>
      </details>

      <div id="access-list" class="list-heading"><strong>Personen</strong><span>Handlungsbedarf zuerst</span></div>

      {{if .HasUsers}}<div class="table-wrap">
        <table aria-label="Benutzerliste">
          <thead>
            <tr>
              <th class="col-person">Person</th>
              <th class="col-role">
                <span class="th-label">Rolle
                  <span class="info">
                    <button type="button" class="info-btn" aria-label="Rollen und Rechte erklärt" aria-describedby="role-help"><svg viewBox="0 0 16 16" width="10" height="10" aria-hidden="true"><circle cx="8" cy="3.5" r="1.15" fill="currentColor"/><rect x="6.9" y="6.3" width="2.2" height="6.3" rx="1.1" fill="currentColor"/></svg></button>
                    <span id="role-help" class="popup" role="tooltip">
                      <span class="popup-title">Rollen &amp; Rechte</span>
                      <span class="popup-grid">
                        <span class="permission"><strong><span class="rdot admin"></span>Admin</strong><span class="muted">Zugänge verwalten, Rollen setzen und Portalbereiche vorbereiten.</span></span>
                        <span class="permission"><strong><span class="rdot manager"></span>Verwalter</strong><span class="muted">Tenant-Verwaltung ohne Plattform- oder Parkplatzkonfiguration.</span></span>
                        <span class="permission"><strong><span class="rdot owner"></span>Eigentümer</strong><span class="muted">Bewohnerbereich plus Eigentümer-Dokumente und Abstimmungen.</span></span>
	                        <span class="permission"><strong><span class="rdot renter"></span>Mieter</strong><span class="muted">Bewohnerbereich ohne Eigentümer-Abstimmungen.</span></span>
	                        <span class="permission"><strong><span class="rdot beirat"></span>Beirat</strong><span class="muted">Bewohnerbereich plus lesende Übersicht.</span></span>
	                        {{if $.ServiceProviderAccessEnabled}}<span class="permission"><strong><span class="rdot service"></span>Dienstleister</strong><span class="muted">Nur zugewiesene Anliegen und deren Anhänge.</span></span>{{end}}
	                        <span class="permission"><strong><span class="rdot right"></span>Parkplatznutzung</strong><span class="muted">Separates Sonderrecht für den privaten Parkplatzbereich.</span></span>
                      </span>
                    </span>
                  </span>
                </span>
              </th>
              <th class="col-secondary">Rechte</th>
              <th class="col-auth">Anmeldung</th>
              <th class="col-status">Status</th>
              <th class="col-actions" aria-label="Aktionen"></th>
            </tr>
          </thead>
          <tbody>
            {{range .Users}}
            <tr class="{{if eq .Status "Deaktiviert"}}needs-attention disabled{{else if eq .Status "Eingeladen"}}needs-attention pending{{end}}">
              <td class="col-person" data-label="Person">
                <div class="person">
                  <span class="avatar">{{.Initials}}</span>
                  <div>
                    <div class="person-name">{{.DisplayName}}</div>
                    <div class="person-mail">{{.Email}}</div>
                    {{if .Phone}}<div class="person-mail">{{.Phone}}</div>{{end}}
                    {{if .HasUnits}}<div class="person-units">{{range .UnitList}}<span>{{.}}</span>{{end}}</div>{{end}}
                  </div>
                </div>
              </td>
              <td class="col-role" data-label="Rolle"><span class="pill {{.RoleClass}}"><span class="dot"></span>{{.Role}}</span><div class="role-caps">{{range .RoleCapabilities}}<span class="role-cap">{{.}}</span>{{end}}</div></td>
              <td class="col-secondary" data-label="Rechte"><div class="chips">{{range .PermissionList}}<span class="chip{{if eq . "Standard"}} plain{{end}}">{{.}}</span>{{end}}</div></td>
              <td class="col-auth" data-label="Anmeldung"><div class="chips">{{range .AuthList}}<span class="chip">{{.}}</span>{{end}}</div></td>
              <td class="col-status" data-label="Status"><span class="pill {{if eq .Status "Aktiv"}}status-active{{else if eq .Status "Deaktiviert"}}status-off{{else}}status-pending{{end}}"><span class="dot"></span>{{.Status}}</span>{{if .LastSeen}}<span class="last-seen">{{.LastSeen}}</span>{{end}}</td>
	              <td class="col-actions" data-label="">
	                {{if or .Editable (and .IsConfig (not .Protected))}}
	                {{if and (not $.ServiceProviderAccessEnabled) (eq .Role "Dienstleister")}}
	                <span class="mini">Schreibgeschützt</span>
	                {{else}}
	                <button type="button" class="row-edit" data-edit="{{.Email}}" aria-label="{{.DisplayName}} bearbeiten" aria-haspopup="dialog" aria-controls="edit-{{.Email}}"><svg viewBox="0 0 24 24" width="16" height="16" aria-hidden="true"><path d="M4 20h4L18.5 9.5a2 2 0 0 0-2.83-2.83L5 17.2z"/><path d="M13.5 6.5 17 10"/></svg><span class="edit-label">Bearbeiten</span></button>
                <dialog id="edit-{{.Email}}" class="edit-dialog" aria-labelledby="edit-title-{{.Email}}">
                  <div class="dlg-x"><form method="dialog"><button aria-label="Schließen">&times;</button></form></div>
                  <h2 id="edit-title-{{.Email}}">Zugang bearbeiten</h2>
                  <p class="dlg-sub">{{.Email}}{{if .IsConfig}} (aus Konfiguration){{end}}</p>
                  <form method="post" action="/app/settings/users/edit" class="dlg-form">
                    <input type="hidden" name="orig_email" value="{{.Email}}">
                    <label class="dialog-core-field"><span>Rolle in diesem Haus</span><select class="f-role" name="role" aria-label="Rolle" autofocus>
                      <option value="Mieter" data-preset-label="Standardzugriff" data-preset-permissions=""{{if eq .Role "Mieter"}} selected{{end}}>Mieter</option>
                      <option value="Eigentümer" data-preset-label="Eigentümerzugriff" data-preset-permissions=""{{if eq .Role "Eigentümer"}} selected{{end}}>Eigentümer</option>
                      <option value="Beirat" data-preset-label="Beiratszugriff" data-preset-permissions=""{{if eq .Role "Beirat"}} selected{{end}}>Beirat</option>
	                      <option value="Verwalter" data-preset-label="Verwalterzugriff" data-preset-permissions=""{{if eq .Role "Verwalter"}} selected{{end}}>Verwalter</option>
	                      {{if $.IsAdmin}}<option value="Admin" data-preset-label="Adminzugriff" data-preset-permissions="parking"{{if eq .Role "Admin"}} selected{{end}}>Admin</option>{{end}}
	                      {{if $.ServiceProviderAccessEnabled}}<option value="Dienstleister" data-preset-label="Nur zugewiesene Anliegen" data-preset-permissions=""{{if eq .Role "Dienstleister"}} selected{{end}}>Dienstleister</option>{{end}}
	                      <option value="Bewohner" data-preset-label="Bewohnerzugriff" data-preset-permissions=""{{if eq .Role "Bewohner"}} selected{{end}}>Bewohner</option>
                    </select></label>
                    <details class="form-disclosure dialog-disclosure identity-section">
                      <summary>Personendaten <span>{{if $.IsAdmin}}bearbeiten{{else}}zentral gepflegt{{end}}</span></summary>
                      <div class="optional-grid">
                        <input class="f-titel" type="text" name="title" value="{{.Title}}" placeholder="Titel" aria-label="Titel"{{if not $.IsAdmin}} readonly{{end}}>
                        <input class="f-vorname" type="text" name="first_name" value="{{.FirstName}}" placeholder="Vorname" aria-label="Vorname"{{if not $.IsAdmin}} readonly{{end}}>
                        <input class="f-nachname" type="text" name="last_name" value="{{.LastName}}" placeholder="Nachname" aria-label="Nachname"{{if not $.IsAdmin}} readonly{{end}}>
                        <input class="f-email" type="email" name="email" value="{{.Email}}" aria-label="E-Mail-Adresse" autocomplete="email"{{if or .IsConfig (not $.IsAdmin)}} readonly{{else}} required{{end}}>
                        {{if not $.IsAdmin}}<p class="dlg-hint">Name und E-Mail gelten für alle Häuser und werden zentral von der Plattform-Administration gepflegt. Die Einstellungen hier gelten nur für dieses Haus.</p>{{end}}
                      </div>
                    </details>
                    <details class="form-disclosure dialog-disclosure unit-section">
                      <summary>Einheiten <span>{{if .HasUnits}}{{len .UnitList}} verknüpft{{else}}keine verknüpft{{end}}</span></summary>
                      <div class="unit-context">
                        {{if .HasUnits}}<div class="chips">{{range .UnitList}}<span class="chip">{{.}}</span>{{end}}</div>{{else}}<p>Dieser Zugang ist noch keiner Einheit zugeordnet.</p>{{end}}
                        <a href="/app/settings/building#units">Zuordnung bei Gebäude &amp; Einheiten verwalten</a>
                      </div>
                    </details>
                    <details class="form-disclosure dialog-disclosure access-section">
                      <summary>Anmeldung &amp; Sonderrechte <span>anpassen</span></summary>
                      <div class="optional-stack">
                        <fieldset class="permission-fieldset f-permissions">
                          <legend>Sonderrechte</legend>
                          <span class="preset-label" data-preset-label>Gespeicherte Rechte</span>
                          <div class="permission-grid">
                            <label class="permission-check"><input type="checkbox" name="permissions" value="parking" data-permission="parking"{{if .ParkingChecked}} checked{{end}}><strong>Parkplatznutzung</strong><span>Privater Bereich für Stellplatz- und Ladeabrechnung.</span></label>
                            <label class="permission-check"><input type="checkbox" name="permissions" value="energy-caretaker" data-permission="energy-caretaker"{{if .EnergyCaretakerChecked}} checked{{end}}><strong>Technische Vertrauensperson</strong><span>Darf Energiedaten ansehen und die Einrichtung unterstützen, aber keine Steuerung aktivieren.</span></label>
                          </div>
                        </fieldset>
                        <fieldset class="permission-fieldset f-permissions">
                          <legend>Anmeldung</legend>
                          <div class="permission-grid">
                            <label class="permission-check"><input type="checkbox" name="auth_methods" value="email"{{if .EmailAuthChecked}} checked{{end}}><strong>E-Mail-Link</strong><span>Anmeldung per Magic-Link an die E-Mail-Adresse.</span></label>
                            <label class="permission-check"><input type="checkbox" name="auth_methods" value="oidc"{{if .OIDCAuthChecked}} checked{{end}}><strong>Sichere Anmeldung</strong><span>Anmeldung über den zentralen Hauszugang.</span></label>
                          </div>
                        </fieldset>
                      </div>
                    </details>
                    <fieldset class="permission-fieldset status-field">
                      <legend>Status</legend>
                      <div class="permission-grid">
                        <label class="permission-check"><input type="checkbox" name="deactivated" value="1"{{if .Deactivated}} checked{{end}}><strong>Zugang deaktivieren</strong><span>Die Anmeldung wird gesperrt; der Eintrag bleibt reaktivierbar.</span></label>
                      </div>
                    </fieldset>
                    <button class="save-access" type="submit">Änderungen speichern</button>
                  </form>
                  {{if not .IsConfig}}<details class="danger-zone">
                    <summary>Zugang dauerhaft entfernen</summary>
                    <div class="dlg-delete">
                      <span>Entfernt die Hauszuordnung dauerhaft. Sperren ist weiterhin rückgängig zu machen.</span>
                      <form method="post" action="/app/settings/users/delete" data-confirm="Diesen Zugang wirklich dauerhaft entfernen?">
                        <input type="hidden" name="email" value="{{.Email}}">
                        <button type="submit" class="danger">Dauerhaft entfernen</button>
                      </form>
                    </div>
                  </details>{{end}}
	                </dialog>
	                {{end}}
	                {{end}}
              </td>
            </tr>
            {{end}}
          </tbody>
        </table>
      </div>{{else}}
        {{template "emptyState" .UsersEmpty}}
      {{end}}
    </section>
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "energyModeStrip"}}
  <section class="energy-mode-strip {{if .IsActiveMode}}active{{end}}" aria-label="Energiemodus">
    <span class="energy-mode-icon" aria-hidden="true">{{if .IsActiveMode}}!{{else}}✓{{end}}</span>
    <div class="energy-mode-copy">
      <strong>{{if .IsActiveMode}}{{if .IsShadowMode}}Testlauf aktiv · keine Gerätewirkung{{else}}Aktive Steuerung{{end}}{{else}}Nur beobachten{{end}}</strong>
      <span>{{if .IsActiveMode}}{{if .IsShadowMode}}HAUSV protokolliert nur. Es schaltet kein Gerät.{{else}}Freigegebene Geräte folgen Ihren Regeln.{{end}}{{else}}Liest und empfiehlt. Keine Steuerung.{{end}}</span>
    </div>
    {{if .IsActiveMode}}
      {{if .CanControlEnergy}}<form method="post" action="/app/energie/mode">
        <input type="hidden" name="mode" value="observe">
        <button class="energy-mode-action" type="submit" aria-label="Sofort zurück zu Nur beobachten"><span class="energy-mode-action-full">Sofort zurück zu „Nur beobachten“</span><span class="energy-mode-action-compact" aria-hidden="true">Nur beobachten</span></button>
      </form>{{end}}
    {{else}}
      {{if .CanControlEnergy}}<details class="energy-mode-control">
        <summary class="energy-mode-action" aria-label="Wirkungslosen Testlauf bewusst starten"><span class="energy-mode-action-full">Testlauf bewusst starten</span><span class="energy-mode-action-compact" aria-hidden="true">Testlauf starten</span></summary>
        <form class="energy-mode-popover" method="post" action="/app/energie/mode">
          <h3>Wirkungslosen Testlauf starten?</h3>
          <p>HAUSV protokolliert nur, welche Entscheidungen seine Regeln treffen würden. Es wird kein Gerät geschaltet. Aktive Steuerung ist erst nach einer späteren, gerätespezifischen Freigabe möglich.</p>
          <label><input type="checkbox" name="confirm" value="yes" required><span>Ich bin Eigentümer oder Hausadministrator und möchte diesen wirkungslosen Testlauf für diese Liegenschaft bewusst starten.</span></label>
          <label><span>Zur Bestätigung <strong>TESTLAUF</strong> eingeben</span></label>
          <input type="text" name="confirmation_text" autocomplete="off" placeholder="TESTLAUF" required>
          <input type="hidden" name="mode" value="active">
          <button class="button primary" type="submit">Testlauf starten</button>
        </form>
      </details>{{else}}<span class="energy-mode-capability">Freigabe nur für Eigentümer oder Hausadministration</span>{{end}}
    {{end}}
  </section>
{{end}}

{{define "homeOnboarding"}}
{{template "appOpen" .}}
  <main id="main-content" tabindex="-1" class="app-main">
	    {{template "energyModeStrip" .}}
	    <div class="page onboarding-page">
	      {{if .ProfileReset}}<div class="message success"><strong>Das Energieprofil wurde gelöscht.</strong> Hausname, Anlagen, Zuordnungen, Messverlauf und Auswertungen sind entfernt. Unabhängige Anliegen und das Sicherheitsprotokoll bleiben nach ihren eigenen Fristen bestehen.</div>{{end}}
	      <div class="onboarding-progress" aria-label="Einrichtungsfortschritt">
        <div class="onboarding-progress-head"><span>Einrichtung Ihres Zuhauses</span><span>Schritt {{.Step}} von 5</span></div>
        <div class="onboarding-progress-track"><span style="width:{{.Progress}}%"></span></div>
      </div>
      <section class="onboarding-card">
        {{if eq .Step 1}}
          <header class="onboarding-card-head">
            <span class="eyebrow">Ganz ohne Technikstress</span>
            <h1>Womit möchten Sie beginnen? Mit Ihrem Zuhause.</h1>
            <p>Wir lernen zuerst kennen, was schon da ist. PV, Batterie oder Home Assistant sind keine Voraussetzung.</p>
          </header>
          <form class="onboarding-body" method="post" action="/app/zuhause/onboarding">
            <div class="onboarding-explain">
              <article><strong>1 · Verstehen</strong><p>Welche großen Verbraucher gibt es und was lässt sich zeitlich verschieben?</p></article>
              <article><strong>2 · Messen</strong><p>Bestehende Messwerte lesen. Home Assistant ist hilfreich, aber keine Voraussetzung.</p></article>
              <article><strong>3 · Verbessern</strong><p>HAUSV zeigt kleine, nachvollziehbare Schritte. Sie behalten die Kontrolle.</p></article>
            </div>
            <div class="onboarding-trust"><span aria-hidden="true">✓</span><div><strong>Standard bleibt „Nur beobachten“</strong><p>Während der gesamten Einrichtung wird nichts geschaltet. Eine spätere Freigabe ist separat, dauerhaft sichtbar und nur für Eigentümer oder Hausadministration möglich.</p></div></div>
            <div class="onboarding-actions"><a class="button" href="/app">Später</a><button class="button primary" type="submit" name="action" value="understand">Verstanden, weiter</button></div>
          </form>
        {{else if eq .Step 2}}
          <header class="onboarding-card-head"><span class="eyebrow">Ihr Zuhause</span><h1>Was richten wir gemeinsam ein?</h1><p>Ein Name und die Art des Zuhauses genügen. Technische Details kommen erst, wenn sie wirklich helfen.</p></header>
          <form class="onboarding-body onboarding-form" method="post" action="/app/zuhause/onboarding">
            <label><span>Anzeigename für „Mein Zuhause“</span><input type="text" name="household_name" value="{{.Profile.HouseholdName}}" placeholder="z. B. Penthouse oder Zuhause Barta" required maxlength="100"></label>
            {{if .HomeTypeLocked}}
              <div class="home-identity-field"><span>Art</span><input type="hidden" name="home_type" value="{{.Profile.HomeType}}"><div class="home-identity-readonly"><strong>{{.HomeTypeLabel}}</strong><small>Durch die zugeordnete Wohneinheit festgelegt</small></div></div>
            {{else}}
              <label><span>Art</span><select name="home_type" data-home-type-select aria-describedby="home-type-explanation">
              <option value="apartment" data-label="Wohnung" data-description="Ein einzelner Haushalt in einem Mehrparteienhaus. Der Überblick konzentriert sich auf Ihre Wohnung und Ihre eigenen Geräte."{{if eq .Profile.HomeType "apartment"}} selected{{end}}>Wohnung</option>
              <option value="house" data-label="Einfamilienhaus" data-description="Ein Haushalt mit eigenem Gebäude. Haus-, Heiz- und Energietechnik können gemeinsam betrachtet werden."{{if eq .Profile.HomeType "house"}} selected{{end}}>Einfamilienhaus</option>
              <option value="community" data-label="Hausgemeinschaft" data-description="Mehrere Parteien und gemeinsam genutzte Anlagen. Der Überblick richtet sich an Eigentümergemeinschaft oder Hausverwaltung."{{if eq .Profile.HomeType "community"}} selected{{end}}>Hausgemeinschaft</option>
              </select></label>
            {{end}}
            {{if .HasHomeUnit}}
              <div class="home-identity-field"><span>Zugeordnete offizielle Wohnung</span><input type="hidden" name="unit_id" value="{{.HomeUnitID}}"><div class="home-identity-readonly"><strong>{{.HomeUnitLabel}}</strong><small>Diesem Hausprofil zugeordnet</small></div><small class="home-identity-field-help">Die Sichtbarkeit folgt dieser Wohnung; technische Freigaben bleiben separat.</small></div>
            {{else if .HasUnitOptions}}<div class="home-identity-field" data-home-unit-field><label><span>Zugeordnete offizielle Wohnung</span><select name="unit_id" required aria-describedby="home-onboarding-unit-help">
              {{range .UnitOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}
            </select></label><small class="home-identity-field-help" id="home-onboarding-unit-help">Die Auswahl legt fest, welche Wohnung den Überblick sieht. Technische Freigaben bleiben separat.</small></div>{{else}}<input type="hidden" name="unit_id" value="">{{end}}
            <div class="home-type-explanation" id="home-type-explanation" data-home-type-explanation role="status" aria-live="polite">
              <span class="home-type-explanation-icon" aria-hidden="true">⌂</span>
              <div>
                <strong data-home-type-label>{{if eq .Profile.HomeType "house"}}Einfamilienhaus{{else if eq .Profile.HomeType "community"}}Hausgemeinschaft{{else}}Wohnung{{end}}</strong>
                <p data-home-type-copy>{{.HomeTypeDescription}}</p>
                <small>{{if .HomeTypeLocked}}Die Art ist mit der Wohnung verbunden.{{else}}Die Auswahl kann Geltungsbereich und Sichtbarkeit ändern.{{end}} „Nur beobachten“ bleibt unverändert.</small>
              </div>
            </div>
            <div class="onboarding-actions"><button class="button" type="submit" name="action" value="back">Zurück</button><button class="button primary" type="submit" name="action" value="profile">Weiter zu den Verbrauchern</button></div>
          </form>
        {{else if eq .Step 3}}
          <header class="onboarding-card-head"><span class="eyebrow">Bestandsaufnahme</span><h1>Was gibt es bereits?</h1><p>Wählen Sie nur, was Sie sicher wissen. Fehlende Details können jederzeit ergänzt werden.</p></header>
          <form class="onboarding-body onboarding-form" method="post" action="/app/zuhause/onboarding">
            <fieldset><legend class="onboarding-legend">Anlagen und größere Verbraucher</legend>
              <div class="onboarding-choice-grid">{{range .AssetOptions}}<label class="onboarding-choice"><input type="checkbox" name="assets" value="{{.Kind}}"{{if .Checked}} checked{{end}}><span><strong>{{.Label}}</strong><small>Vorhanden oder regelmäßig genutzt</small></span></label>{{end}}</div>
            </fieldset>
            <div class="onboarding-actions"><button class="button" type="submit" name="action" value="back">Zurück</button><button class="button primary" type="submit" name="action" value="assets">Weiter zu den Messwerten</button></div>
          </form>
        {{else if eq .Step 4}}
          <header class="onboarding-card-head"><span class="eyebrow">Datenquelle</span><h1>Wie kommen Messwerte herein?</h1><p>{{.ConnectorMessage}}</p></header>
          <form class="onboarding-body onboarding-form" method="post" action="/app/zuhause/onboarding">
            <section class="energy-mapping-guide" aria-labelledby="energy-mapping-guide-title">
              <header><div><span class="onboarding-legend">Messwert-Setup</span><h2 id="energy-mapping-guide-title">Was bedeutet welcher Wert?</h2></div><small>Alles bleibt nur gelesen.</small></header>
              <div class="energy-mapping-slots">{{range .MappingSlots}}<article class="energy-mapping-slot {{.Tone}}" data-mapping-slot="{{.Key}}">
                <div><strong>{{.Label}}</strong><span>{{.Purpose}}</span></div>
                <p><b>{{.Status}}</b><small>{{.Detail}}</small></p>
              </article>{{end}}</div>
            </section>
            {{if .HasCandidates}}<fieldset class="onboarding-recommended"><legend class="onboarding-legend">Empfohlen</legend><div class="onboarding-candidates">{{range .Candidates}}<label class="onboarding-candidate"><input type="checkbox" name="entities" value="{{.EntityID}}"{{if .Checked}} checked{{end}}><span><strong>{{.DisplayName}}</strong><small>{{.SourceName}}</small></span><span class="onboarding-readonly">Nur lesen</span></label>{{end}}</div></fieldset>
            {{else}}<div class="onboarding-trust"><span aria-hidden="true">i</span><div><strong>Ohne Verbindung fortfahren</strong><p>Sie können das Haus-Cockpit bereits nutzen und Home Assistant später ergänzen. Es werden keine Zugangsdaten im Portal angezeigt oder gespeichert.</p></div></div>{{end}}
            {{if .HasAdditional}}<details class="onboarding-disclosure onboarding-additional"><summary>Weitere technische Treffer anzeigen <span>{{len .AdditionalCandidates}} optional</span></summary><div class="onboarding-candidates">{{range .AdditionalCandidates}}<label class="onboarding-candidate"><input type="checkbox" name="entities" value="{{.EntityID}}"{{if .Checked}} checked{{end}}><span><strong>{{.MetricLabel}}</strong><small>{{.SourceName}}</small><code>{{.EntityID}}</code></span><span class="onboarding-readonly">Nur lesen</span></label>{{end}}</div></details>{{end}}
            <details class="onboarding-disclosure"><summary>Messwert selbst zuordnen <span>nur falls nötig</span></summary><div class="optional-grid">
              <label><span>Sensor-ID</span><input type="text" name="manual_entity_id" placeholder="sensor.netzbezug"></label>
              <label><span>Bedeutung</span><select name="manual_metric"><option value="grid-import-power">Netzbezug Leistung</option><option value="grid-import-energy">Netzbezug Energie</option><option value="grid-export-power">Netzeinspeisung</option><option value="pv-power">PV-Leistung</option><option value="battery-power">Batterie-Leistung</option><option value="battery-soc">Batterie-Ladestand</option><option value="load-power">Hausverbrauch</option></select></label>
              <label><span>Verständlicher Name</span><input type="text" name="manual_name" placeholder="Netzbezug gesamt"></label>
              <label><span>Einheit</span><input type="text" name="manual_unit" placeholder="W oder kW"></label>
              <label><span>Gehört zu</span><select name="manual_asset_id"><option value="">Gesamtes Haus</option>{{range .MappingAssetOptions}}<option value="{{.Value}}">{{.Label}}</option>{{end}}</select></label>
            </div></details>
            <div class="onboarding-actions"><button class="button" type="submit" name="action" value="back">Zurück</button>{{if .HasCandidates}}<button class="button ghost onboarding-skip" type="submit" name="action" value="skip-mappings">Ohne Verbindung starten</button><button class="button primary" type="submit" name="action" value="mappings">{{.RecommendedCount}} Messwerte übernehmen</button>{{else}}<button class="button primary" type="submit" name="action" value="skip-mappings">Ohne Verbindung starten</button>{{end}}</div>
          </form>
        {{else}}
          <header class="onboarding-card-head" data-home-identity="onboarding-summary"{{if .HasHomeUnit}} aria-label="{{.Profile.HouseholdName}}, offizielle Einheit {{.HomeUnitLabel}}"{{end}}><span class="eyebrow">Ihr Zuhause ist startklar</span><h1 data-home-display-name>{{.Profile.HouseholdName}}</h1>{{if .HasHomeUnit}}<span class="onboarding-home-unit" data-home-unit-label>{{.HomeUnitLabel}}</span>{{end}}<p>Alles bleibt im sicheren Beobachtungsmodus. Sie gehen in Ihrem Tempo weiter.</p></header>
          <form class="onboarding-body" method="post" action="/app/zuhause/onboarding">
            <div class="onboarding-trust"><span aria-hidden="true">→</span><div><strong>Als Nächstes: {{.FinishRecommendation.Title}}</strong><p>{{.FinishRecommendation.Reason}}</p></div></div>
            <div class="onboarding-trust"><span aria-hidden="true">✓</span><div><strong>Drei Jahre voller Produktumfang kostenlos</strong><p>Danach gilt nach heutigem Modell: 1 € pro Monat, jährlich als 12 € verrechnet. Noch gibt es keine Zahlung und keine versteckte Einschränkung.</p></div></div>
            <div class="onboarding-actions"><button class="button" type="submit" name="action" value="back">Noch einmal prüfen</button><button class="button primary" type="submit" name="action" value="finish">Mein Zuhause öffnen</button></div>
          </form>
        {{end}}
      </section>
    </div>
  </main>
{{template "appClose" .}}
{{end}}

{{define "energyMetricIcon"}}
  {{if eq .Metric "load-power"}}<span class="energy-metric-icon load" aria-hidden="true"><svg viewBox="0 0 32 32"><path d="M4.5 15.2 16 5.5l11.5 9.7V27h-23Z"/><path d="M8.5 20h4l2-5 3.1 9 2.1-5h3.8"/></svg></span>
  {{else if eq .Metric "pv-power"}}<span class="energy-metric-icon pv" aria-hidden="true"><svg viewBox="0 0 32 32"><circle cx="11" cy="8" r="3.5"/><path d="M11 1.5v2M11 12.5v2M4.5 8h2M15.5 8h2M6.4 3.4l1.4 1.4M14.2 11.2l1.4 1.4M15.6 3.4l-1.4 1.4M7.8 11.2l-1.4 1.4"/><path d="m8 17.5-2.2 10h20.4l-2.2-10Z"/><path d="M9.5 22.5h15M16 17.5l-1 10M21 17.5l1 10"/></svg></span>
  {{else if eq .Metric "grid-import-power"}}<span class="energy-metric-icon grid-import" aria-hidden="true"><svg viewBox="0 0 32 32"><path d="M10.5 28 16 4l5.5 24M12 20h8M13.2 14h5.6M14.3 9h3.4M8.5 14h15M7 20h18"/><path d="M22.5 11.5h6M26 8l3.5 3.5L26 15"/></svg></span>
  {{else if eq .Metric "grid-export-power"}}<span class="energy-metric-icon grid-export" aria-hidden="true"><svg viewBox="0 0 32 32"><path d="M10.5 28 16 4l5.5 24M12 20h8M13.2 14h5.6M14.3 9h3.4M8.5 14h15M7 20h18"/><path d="M9.5 11.5h-6M6 8l-3.5 3.5L6 15"/></svg></span>
  {{else if eq .Metric "battery-power"}}<span class="energy-metric-icon battery" aria-hidden="true"><svg viewBox="0 0 32 32"><rect x="4" y="8" width="23" height="16" rx="3"/><path d="M27 13h2.5v6H27M16.7 11.5 12 17h4l-1 4 5-6h-4Z"/></svg></span>
  {{end}}
{{end}}

{{define "energyChartLegend"}}
<div class="energy-chart-legend" aria-label="Diagrammlegende">{{range .Chart.Series}}<span class="{{.Key}}"><i aria-hidden="true"></i>{{.Label}} <strong>{{.Latest}}</strong></span>{{end}}{{if .Chart.HasThreshold}}<span class="threshold"><i aria-hidden="true"></i>{{.Chart.ThresholdLabel}} {{.Chart.ThresholdValue}} <small>(konfigurierbar)</small></span>{{end}}</div>
{{end}}

{{define "energyChartInteractive"}}
<div class="energy-chart-interactive" data-energy-chart-interactive tabindex="0" aria-label="{{.Chart.Title}}: Viertelstundenwerte. Mit der Maus erkunden oder mit den Pfeiltasten durchgehen.">
  <svg class="energy-chart-svg desktop" viewBox="0 0 800 236" role="img" aria-label="Leistungsverlauf: {{.Chart.Title}}">
    <title>Leistungsverlauf: {{.Chart.Title}}</title>
    <desc>Hausverbrauch, PV-Erzeugung, Netz und Speicher als Viertelstunden-Verlauf in Kilowatt. Die gestrichelte Linie ist eine konfigurierbare Planungsgrenze.</desc>
    {{range .Chart.YTicks}}<line class="energy-chart-grid{{if eq .Label "0 kW"}} zero{{end}}" x1="52" x2="788" y1="{{.Position}}" y2="{{.Position}}"></line><text class="energy-chart-axis-label" x="44" y="{{.Position}}" text-anchor="end" dominant-baseline="middle">{{.Label}}</text>{{end}}
    {{range .Chart.XTicks}}<line class="energy-chart-grid" x1="{{.Position}}" x2="{{.Position}}" y1="16" y2="204"></line><text class="energy-chart-axis-label" x="{{.Position}}" y="226" text-anchor="middle">{{.Label}}</text>{{end}}
    {{range .Chart.Series}}{{if .AreaPath}}<path class="energy-chart-area {{.Key}}" d="{{.AreaPath}}"></path>{{end}}{{end}}
    {{if .Chart.HasThreshold}}<line class="energy-chart-threshold" x1="52" x2="788" y1="{{.Chart.ThresholdPosition}}" y2="{{.Chart.ThresholdPosition}}"></line><text class="energy-chart-threshold-label" x="782" y="{{.Chart.ThresholdPosition}}" dy="-6" text-anchor="end">{{.Chart.ThresholdValue}}</text>{{end}}
    {{range .Chart.Series}}<path class="energy-chart-line {{.Key}}" d="{{.Path}}"></path>{{end}}
    <line class="energy-chart-guide" data-chart-guide x1="52" x2="52" y1="16" y2="204" hidden></line>
    {{range .Chart.Samples}}{{$sample := .}}{{range .Values}}<circle class="energy-chart-marker {{.Key}}" data-chart-marker-index="{{$sample.Index}}" cx="{{$sample.Position}}" cy="{{.Position}}" r="4" hidden></circle>{{end}}<rect class="energy-chart-hit" data-chart-hit data-index="{{.Index}}" data-x="{{.Position}}" x="{{.HitPosition}}" y="16" width="{{.HitWidth}}" height="188"></rect>{{end}}
  </svg>
  <svg class="energy-chart-svg mobile" viewBox="0 0 400 236" role="img" aria-label="Leistungsverlauf: {{.Chart.Title}}">
    <title>Leistungsverlauf: {{.Chart.Title}}</title>
    <desc>Hausverbrauch, PV-Erzeugung, Netz und Speicher als Viertelstunden-Verlauf in Kilowatt. Die gestrichelte Linie ist eine konfigurierbare Planungsgrenze.</desc>
    {{range .Chart.YTicks}}<line class="energy-chart-grid{{if eq .Label "0 kW"}} zero{{end}}" x1="44" x2="388" y1="{{.MobilePosition}}" y2="{{.MobilePosition}}"></line><text class="energy-chart-axis-label" x="37" y="{{.MobilePosition}}" text-anchor="end" dominant-baseline="middle">{{.Label}}</text>{{end}}
    {{range .Chart.XTicks}}<line class="energy-chart-grid" x1="{{.MobilePosition}}" x2="{{.MobilePosition}}" y1="16" y2="204"></line><text class="energy-chart-axis-label" x="{{.MobilePosition}}" y="226" text-anchor="middle">{{.Label}}</text>{{end}}
    {{range .Chart.Series}}{{if .MobileAreaPath}}<path class="energy-chart-area {{.Key}}" d="{{.MobileAreaPath}}"></path>{{end}}{{end}}
    {{if .Chart.HasThreshold}}<line class="energy-chart-threshold" x1="44" x2="388" y1="{{.Chart.ThresholdMobilePosition}}" y2="{{.Chart.ThresholdMobilePosition}}"></line><text class="energy-chart-threshold-label" x="382" y="{{.Chart.ThresholdMobilePosition}}" dy="-5" text-anchor="end">{{.Chart.ThresholdValue}}</text>{{end}}
    {{range .Chart.Series}}<path class="energy-chart-line {{.Key}}" d="{{.MobilePath}}"></path>{{end}}
    <line class="energy-chart-guide" data-chart-guide x1="44" x2="44" y1="16" y2="204" hidden></line>
    {{range .Chart.Samples}}{{$sample := .}}{{range .Values}}<circle class="energy-chart-marker {{.Key}}" data-chart-marker-index="{{$sample.Index}}" cx="{{$sample.MobilePosition}}" cy="{{.MobilePosition}}" r="4" hidden></circle>{{end}}<rect class="energy-chart-hit" data-chart-hit data-index="{{.Index}}" data-x="{{.MobilePosition}}" x="{{.MobileHit}}" y="16" width="{{.MobileHitWidth}}" height="188"></rect>{{end}}
  </svg>
  <div class="energy-chart-tooltip" data-chart-tooltip role="status" aria-live="polite" hidden><strong data-chart-tooltip-time></strong><div data-chart-tooltip-values></div></div>
  <div class="energy-chart-sample-bank" data-chart-sample-bank hidden>{{range .Chart.Samples}}<div data-chart-sample="{{.Index}}" data-time="{{.Time}}">{{range .Values}}<span data-key="{{.Key}}" data-label="{{.Label}}" data-value="{{.Value}}"></span>{{end}}</div>{{end}}</div>
</div>
{{end}}

{{define "energyLead"}}
  <div class="energy-lead-side">
    <aside class="energy-live" aria-label="Energie gerade jetzt" data-energy-reading-count="{{len .Metrics}}">
      <header class="energy-live-head"><div><h2>Energie gerade jetzt</h2><span>Live aus Home Assistant</span></div><div class="energy-live-tools"><small>Nur gelesen</small>{{if .CanManageEnergy}}<a href="/app/zuhause/onboarding?step=4">Messwerte zuordnen</a>{{end}}</div></header>
      {{if .HasMetrics}}
        {{if .Live.HasMain}}<div class="energy-live-main">{{template "energyMetricIcon" .Live.Main}}<div><span>{{.Live.Main.Label}}</span><strong>{{.Live.Main.Value}}</strong><span>{{.Live.Main.Detail}}</span></div></div>{{end}}
        {{if .Live.HasBatterySOC}}<div class="energy-storage-live" data-energy-metric="battery-power">
          <span class="energy-battery-visual {{.Live.Battery.Direction}}" data-energy-direction="{{.Live.Battery.Direction}}" role="img" aria-label="Speicher zu {{.Live.BatterySOC.Value}} gefüllt{{if .Live.HasBattery}}, {{.Live.Battery.Detail}} mit {{.Live.Battery.Value}}{{end}}">
            <span class="energy-battery-gauge" aria-hidden="true"><i style="width:{{.Live.BatteryFill}}%"></i>{{if or (eq .Live.Battery.Direction "charging") (eq .Live.Battery.Direction "discharging")}}<span class="energy-battery-flow"><svg class="energy-battery-chevron" viewBox="0 0 9 14"><path d="m2.5 2 4.5 5-4.5 5"/></svg><svg class="energy-battery-chevron" viewBox="0 0 9 14"><path d="m2.5 2 4.5 5-4.5 5"/></svg><svg class="energy-battery-chevron" viewBox="0 0 9 14"><path d="m2.5 2 4.5 5-4.5 5"/></svg></span>{{end}}</span>
          </span>
          <div><span>Speicher</span><strong>{{.Live.BatterySOC.Value}}</strong>{{if .Live.HasBattery}}<small>{{.Live.Battery.Detail}} · {{.Live.Battery.Value}}</small>{{else}}<small>Aktueller Ladestand</small>{{end}}</div>
        </div>{{end}}
        {{if or .Live.Flows (and .Live.HasBattery (not .Live.HasBatterySOC))}}<div class="energy-flow-grid">
          {{range .Live.Flows}}<div class="energy-flow-item {{.Tone}}" data-energy-metric="{{.Metric}}">{{template "energyMetricIcon" .}}<div class="energy-flow-copy"><span>{{.Label}}</span><strong>{{.Value}}</strong><small>{{.Detail}}</small></div></div>{{end}}
          {{if and .Live.HasBattery (not .Live.HasBatterySOC)}}<div class="energy-flow-item" data-energy-metric="battery-power">{{template "energyMetricIcon" .Live.Battery}}<div class="energy-flow-copy"><span>{{.Live.Battery.Label}}</span><strong>{{.Live.Battery.Value}}</strong><small>{{.Live.Battery.Detail}}</small></div></div>{{end}}
        </div>{{end}}
        {{if .Live.HasAdditional}}<details class="energy-live-more"><summary><span><strong>Weitere Messwerte ({{.Live.AdditionalCount}})</strong><small>{{.Live.AdditionalTopics}}</small></span></summary><div class="energy-live-more-list">{{range .Live.Additional}}<div class="energy-live-more-row"><span>{{.Label}}</span><strong>{{.Value}}</strong></div>{{end}}</div></details>{{end}}
      {{else}}<div class="energy-live-empty"><strong>Noch keine Live-Werte</strong><span>Home Assistant kann später verbunden werden.</span></div>{{end}}
    </aside>
    <section class="energy-card energy-nextstep" id="naechster-schritt">
      <div class="energy-nextstep-copy">
        <span class="eyebrow">Als Nächstes</span>
        <h2>{{.Recommendation.Title}}</h2>
        <p>{{.Recommendation.Reason}}</p>
        <div class="energy-next-meta"><span>{{.Recommendation.Benefit}}</span><span>Aufwand: {{.Recommendation.Effort}}</span><span>{{.Recommendation.ImpactRange}}</span></div>
      </div>
      {{if .MeasureCreated}}<div class="message success">Als nachvollziehbare Hausaufgabe angelegt. Erst dort wählen Sie später bewusst einen bekannten Dienstleister.</div>{{end}}
      {{if .RecommendationDeferred}}<div class="message">Für später gemerkt. HAUSV aktiviert dadurch nichts.</div>{{else if .RecommendationDismissed}}<div class="message">Abgelehnt. Die Entscheidung bleibt nachvollziehbar und löst nichts aus.</div>{{else}}
      <div class="energy-recommendation-actions">
        {{if .RecommendationURL}}<a class="button primary" href="{{.RecommendationURL}}">Diesen Schritt öffnen</a>{{end}}
        <details class="energy-measure-control">
          <summary class="button">Als Hausaufgabe übernehmen</summary>
          <form class="energy-measure-form" method="post" action="/app/energie/measure">
            <h3>Was darf in die Aufgabe?</h3>
            <p>Es entsteht ein normales Anliegen – keine Bestellung und keine Preiszusage.</p>
            <label><input type="checkbox" name="share" value="inventory"><span>Anlageninventar beilegen</span></label>
            <label><input type="checkbox" name="share" value="measurements"><span>Zusammengefasste Messwerte beilegen</span></label>
            <label><input type="checkbox" name="share" value="contact"><span>Kontaktdaten beilegen</span></label>
            <input type="hidden" name="recommendation_id" value="{{.Recommendation.ID}}">
            <button class="button primary" type="submit">Hausaufgabe anlegen</button>
          </form>
        </details>
        <form method="post" action="/app/energie/recommendation"><input type="hidden" name="recommendation_id" value="{{.Recommendation.ID}}"><button class="button quiet" type="submit" name="status" value="deferred">Später</button><button class="button quiet" type="submit" name="status" value="dismissed">Nicht für uns</button></form>
      </div>{{end}}
    </section>
  </div>
{{end}}

{{define "energyCockpit"}}
{{template "appOpen" .}}
  <main id="main-content" tabindex="-1" class="app-main">
    {{template "energyModeStrip" .}}
    <div class="page energy-page">
      <header class="energy-heading">
        <div class="energy-heading-copy" data-home-identity="energy-heading" aria-label="{{.HomeIdentity.AriaLabel}}">
          <span class="eyebrow">Mein Zuhause</span>
          <h1 data-home-display-name>{{.Profile.HouseholdName}}</h1>
          {{if .HasHomeUnit}}<p class="energy-heading-unit" data-home-unit-label>{{.HomeUnitLabel}}</p>{{end}}
          <p class="energy-heading-context">{{.HomeTypeLabel}} · {{.Tenant.Address}}</p>
        </div>
        {{if .CanManageHomeIdentity}}<a class="button energy-heading-action" href="/app/settings/home?from=energy"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m4 16-.7 4.7L8 20l10.8-10.8a2.3 2.3 0 0 0-3.2-3.2Z"/><path d="m14.5 7.1 3.2 3.2"/></svg>Zuhause bearbeiten</a>{{end}}
      </header>
      {{if .Welcome}}<div class="message success">Ihr Hausprofil ist bereit. Der sichere Beobachtungsmodus bleibt aktiv.</div>{{end}}
      {{if .ModeChanged}}<div class="message success">Der Energiemodus wurde nachvollziehbar geändert.</div>{{end}}
      {{if .ProfileChanged}}<div class="message success">Der Anzeigename von „Mein Zuhause“ wurde gespeichert.</div>{{end}}
      <section class="energy-health">
        {{template "energyLead" .}}
        <section class="energy-card energy-tariff" id="tarif" aria-labelledby="energy-tariff-title">
          <header class="energy-card-head">
            <div><span class="eyebrow">Leistungstarif ab 2027</span><h2 id="energy-tariff-title">Ihre Monatsspitze · {{.Tariff.MonthLabel}}</h2><p>Ab 01.01.2027 bemisst der Netzbetreiber die höchste Viertelstunde jedes Kalendermonats.</p></div>
            <span class="pill">{{.Tariff.Status}}</span>
          </header>
          {{if .TargetChanged}}<div class="message success">Ihr persönliches Peak-Ziel wurde gespeichert.</div>{{end}}
          {{if .AgreedPowerChanged}}<div class="message success">Die vereinbarte Anschlussleistung wurde gespeichert.</div>{{end}}
          {{if .Tariff.HasEstimate}}<div class="energy-billed" aria-label="Verrechnete Leistung nach dem Entwurf">
            <div><span>Höchste Viertelstunde</span><strong>{{.Tariff.PeakKW}}</strong>{{if .Tariff.HasPeakTime}}<small>gemessen am {{.Tariff.PeakTime}}</small>{{end}}</div>
            <div><span>Verrechnet</span><strong>{{.Tariff.BilledKW}}</strong><small>{{if .Tariff.MinimumReason}}{{.Tariff.MinimumReason}}{{else}}die gemessene Spitze ist maßgeblich{{end}}</small></div>
            {{if .Tariff.HasTier}}<div><span>Günstigere Stufe</span><strong>{{.Tariff.BelowKW}}</strong><small>{{.Tariff.BelowRateEUR}} je kW und Jahr</small></div>
            <div class="energy-billed-above"><span>Höhere Stufe</span><strong>{{.Tariff.AboveKW}}</strong><small>{{.Tariff.AboveRateEUR}} je kW und Jahr</small></div>{{end}}
          </div>
          {{if .Tariff.Basis}}<p class="energy-tariff-basis"><span aria-hidden="true">◎</span>Grundlage: {{.Tariff.Basis}}</p>{{end}}
          <div class="energy-tariff-cost">
            <div class="energy-tariff-cost-figure"><span>Leistungsanteil des Netztarifs</span><strong>{{.Tariff.AnnualPowerEUR}}</strong><small>pro Jahr · Modellrechnung</small></div>
            <div class="energy-tariff-cost-copy">
              <p><strong>Das ist nicht Ihre Stromrechnung.</strong> Der Betrag umfasst ausschließlich den Leistungsanteil des Netztarifs. Arbeitspreis, Energiekosten, Abgaben und Steuern sind darin nicht enthalten.</p>
              <p>Gerechnet mit {{.Tariff.BelowRateEUR}} je kW bis {{.Tariff.ThresholdKW}} und {{.Tariff.AboveRateEUR}} je kW darüber, auf die verrechnete Leistung von {{.Tariff.BilledKW}}. Die Sätze stammen aus einem Begutachtungsentwurf und können sich noch ändern.</p>
            </div>
          </div>
          <ul class="energy-tariff-notes">
            {{if .Tariff.HasTier}}<li>{{.Tariff.TierHint}}</li>{{end}}
            <li>Niedertarif-Fenster (SNAP, WiNAP) und Energiegemeinschaften senken den Arbeitspreis, nicht die verrechnete Leistung.</li>
            <li>{{.Tariff.Disclaimer}}</li>
          </ul>
          {{else}}<div class="energy-tariff-empty{{if .Tariff.MissingIsWaiting}} waiting{{end}}">
            <strong>{{if .Tariff.MissingIsWaiting}}Noch keine volle Viertelstunde{{else}}Netzbezug noch nicht zugeordnet{{end}}</strong>
            <p>{{.Tariff.MissingReason}}</p>
            {{if not .Tariff.MissingIsWaiting}}<a class="button" href="#messwerte">Netzbezug zuordnen</a>{{end}}
          </div>{{end}}
          {{if eq .TariffAssessmentStatus "saved"}}<div class="message success">Diese Modellbewertung wurde mit ihrer damaligen Regelversion festgehalten.</div>{{else if eq .TariffAssessmentStatus "no_data"}}<div class="message">Für eine historische Bewertung fehlen noch abgeschlossene Viertelstunden.</div>{{end}}
          {{if .CanManageEnergy}}<details class="energy-tariff-settings">
            <summary>Ihre Werte für diese Rechnung <span>Peak-Ziel, Anschlussleistung</span></summary>
            <div class="energy-tariff-forms">
              <form class="energy-target-form" method="post" action="/app/energie/target"><label><span class="onboarding-legend">Persönliches Peak-Ziel in kW</span><input type="text" name="target_peak_kw" inputmode="decimal" value="{{.TargetPeakValue}}" placeholder="z. B. 8,0" required></label><small class="muted">Ein Planungsziel, keine technische Anschlussgrenze.</small><button class="button" type="submit">Ziel speichern</button></form>
              <form class="energy-target-form" method="post" action="/app/energie/anschlussleistung"><label><span class="onboarding-legend">Vereinbarte Anschlussleistung in kW</span><input type="text" name="agreed_power_kw" inputmode="decimal" value="{{.AgreedPowerValue}}" placeholder="z. B. 14,0"></label><small class="muted">{{if .Tariff.HasAgreed}}Steht auf Ihrer Netzrechnung. Der Entwurf bemisst mindestens 20 % davon.{{else}}{{.Tariff.AgreedHint}}{{end}} Leer lassen, wenn unbekannt.</small><button class="button" type="submit">Anschlussleistung speichern</button></form>
            </div>
          </details>{{end}}
          <footer class="energy-tariff-foot">
            <span>Regelprofil {{.Tariff.ID}} · Stand {{.Tariff.Version}}</span>
            <a href="{{.Tariff.SourceURL}}" target="_blank" rel="noopener noreferrer">Quelle: {{.Tariff.SourceTitle}} →</a>
            {{if and .CanManageEnergy .Tariff.HasEstimate}}<form method="post" action="/app/energie/tariff/assessment"><button class="button quiet" type="submit">Diesen Stand festhalten</button></form>{{end}}
          </footer>
          {{if .HasTariffAssessments}}<div class="energy-history"><h4>Festgehaltene Bewertungen</h4><div aria-label="Historische Tarifbewertungen">{{range .TariffAssessments}}<article class="energy-history-row"><div><strong>{{.Month}} · {{.Peak}}</strong><span>{{.Profile}} · {{.Quality}}</span></div><strong>{{.Annual}}</strong><small>{{.Created}}</small></article>{{end}}</div></div>{{end}}
        </section>
      </section>
      <section class="energy-card energy-chart" id="energieverlauf" aria-labelledby="energy-chart-title">
        <header class="energy-chart-head"><div><h2 id="energy-chart-title">{{.Chart.Title}}</h2><p>Wann war viel los – und woher kam die Energie?</p></div><div class="energy-chart-head-actions"><small>{{.Chart.Status}}</small><div class="energy-chart-toolbar"><nav class="energy-chart-range" aria-label="Zeitraum auswählen"><a href="/app/energie?zeitraum=letzte-24h#energieverlauf"{{if not .Chart.IsToday}} aria-current="page"{{end}}>Letzte 24 h</a><a href="/app/energie?zeitraum=heute#energieverlauf"{{if .Chart.IsToday}} aria-current="page"{{end}}>Heute</a></nav>{{if .Chart.HasData}}<div class="energy-chart-size-actions"><button class="button small energy-chart-size-button" type="button" data-dialog="energy-chart-dialog" aria-haspopup="dialog" aria-controls="energy-chart-dialog"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M8 3H3v5M16 3h5v5M21 16v5h-5M8 21H3v-5" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/></svg>Vergrößern</button><button class="button small energy-chart-size-button" type="button" data-dialog="energy-chart-dialog" data-energy-fullscreen aria-haspopup="dialog" aria-controls="energy-chart-dialog" aria-pressed="false"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M9 3H3v6M15 3h6v6M21 15v6h-6M9 21H3v-6" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/></svg><span data-fullscreen-label>Vollbild</span></button></div>{{end}}</div></div></header>
        {{if .Chart.HasData}}<div class="energy-chart-layout">
          <div class="energy-chart-plot">
            {{template "energyChartLegend" .}}
            {{template "energyChartInteractive" .}}
            <small class="energy-chart-hint">Netz und Speicher unter null bedeuten Einspeisung beziehungsweise Laden. Die Planungsgrenze ist eine änderbare Modellannahme, kein geltender Tarif. {{.Chart.Range}}</small>
          </div>
          <aside class="energy-chart-note"><span>Auf einen Blick</span><strong>{{.Chart.Summary}}</strong><p>{{.Chart.Detail}}</p></aside>
        </div>{{else}}<div class="energy-chart-empty"><strong>Noch kein vollständiger Tagesverlauf</strong><p>{{.Chart.Status}}</p></div>{{end}}
      </section>
      {{if .Chart.HasData}}<dialog id="energy-chart-dialog" class="energy-chart-dialog" aria-labelledby="energy-chart-dialog-title"><div class="energy-chart-dialog-shell" data-energy-fullscreen-surface>
        <header><div><span class="eyebrow">Energieverlauf</span><h2 id="energy-chart-dialog-title">{{.Chart.DialogTitle}}</h2><p>Fahren Sie über die Kurve oder nutzen Sie die Pfeiltasten.</p></div><div class="energy-chart-dialog-actions"><button class="button" type="button" data-dialog="energy-chart-dialog" data-energy-fullscreen aria-controls="energy-chart-dialog" aria-pressed="false"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M9 3H3v6M15 3h6v6M21 15v6h-6M9 21H3v-6" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/></svg><span data-fullscreen-label>Vollbild</span></button><button class="button" type="button" data-close-dialog>Schließen</button></div></header>
        {{template "energyChartLegend" .}}
        <div class="energy-chart-dialog-plot">{{template "energyChartInteractive" .}}</div>
        <small class="energy-chart-hint">Netz und Speicher unter null bedeuten Einspeisung beziehungsweise Laden. {{.Chart.Range}}</small>
      </div></dialog>{{end}}
      <div class="energy-pair">
        {{if .HasScenarios}}<section class="energy-card" id="szenarien">
          <header class="energy-card-head"><div><h2>Was-wäre-wenn</h2><p>Bandbreite statt Einsparversprechen.</p></div></header>
          {{range .Scenarios}}<article class="energy-scenario">
            <div class="energy-scenario-copy"><strong>{{.Title}}</strong><p>{{.EffectBand}} · Unsicherheit: {{.Uncertainty}}</p><p class="energy-scenario-assumptions">{{.Assumptions}}</p>{{if .BaselineNote}}<p class="energy-scenario-assumptions">{{.BaselineNote}}</p>{{end}}</div>
            <div class="energy-scenario-result">
              <span>Spitze im Modell</span><strong>{{.PeakBand}}</strong>
              {{if .HasBilledBand}}<span class="energy-scenario-billed">davon verrechnet <b>{{.BilledBand}}</b></span>{{end}}
            </div>
          </article>{{if .FloorNote}}<p class="energy-scenario-floor"><span aria-hidden="true">!</span>{{.FloorNote}}</p>{{end}}{{end}}
          <p class="energy-scenario-caveat">Die Bandbreite tritt nur ein, wenn sich die genannten Verbraucher tatsächlich verschieben lassen und nicht ohnehin gleichzeitig laufen. HAUSV rechnet sie bewusst nicht in Euro um: dafür ist der Entwurf zu unsicher und die Messbasis zu jung.</p>
        </section>{{end}}
        <section class="energy-card">
          <header class="energy-card-head"><div><h2>Datenlage</h2><p>Keine scheinpräzisen Aussagen bei Lücken oder alten Werten.</p></div></header>
          <div class="energy-quality"><span aria-hidden="true">{{if eq .Quality.Status "measured"}}✓{{else}}!{{end}}</span><div><strong>{{.Quality.Label}}</strong><p>{{.Quality.Effect}}</p></div><small>{{.Quality.NextAction}}</small></div>
          <div class="energy-coverage" aria-label="Messabdeckung"><div class="energy-coverage-head"><strong>Messabdeckung</strong><span>{{.CoverageSummary}}</span></div><div class="energy-coverage-list">{{range .Coverage}}<div class="energy-coverage-row" title="{{.Detail}}"><strong>{{.Label}}</strong><span class="energy-coverage-state {{.Tone}}">{{if eq .Tone "good"}}✓{{else}}•{{end}} {{.Status}}</span></div>{{end}}</div></div>
        </section>
      </div>
      <section class="energy-card energy-roadmap-card" id="fahrplan">
        <header class="energy-card-head"><div><h2>Ihr Energie-Fahrplan</h2><p>Ein klarer Schritt nach dem anderen.</p></div></header>
        <div class="energy-roadmap">{{range .Roadmap}}<article class="energy-roadmap-step {{if .Current}}current{{end}}"><span class="energy-roadmap-number">{{.Number}}</span><span class="energy-roadmap-state">{{.State}}</span><strong>{{.Title}}</strong><p>{{.Detail}}</p>{{if .Action}}<a href="{{.URL}}">{{.Action}} →</a>{{end}}</article>{{end}}</div>
      </section>
      <section class="energy-card">
        <header class="energy-card-head"><div><h2>Was Ihr Zuhause mitbringt</h2><p>Die Grundlage für Empfehlungen – keine Einkaufsliste.</p></div>{{if .CanManageEnergy}}<a class="button" href="/app/zuhause/onboarding?step=3">Bearbeiten</a>{{end}}</header>
        {{if .HasAssets}}<div class="energy-assets">{{range .Assets}}<span class="energy-asset"><span aria-hidden="true">✓</span>{{.Name}}</span>{{end}}</div><nav class="energy-related-links" aria-label="Verknüpfte Hausbereiche"><a href="/app/dokumente">Unterlagen</a><a href="/app/events">Wartungstermine</a><a href="/app/anliegen?new=1">Aufgabe melden</a><a href="/app/kontakte">Fachkontakte</a></nav>{{else}}<p class="muted">Noch keine größeren Verbraucher erfasst.</p>{{end}}
        <div id="anlagen" class="energy-consumers">
          {{if eq .ConsumerNotice "1"}}<div class="message success">Der Verbraucher wurde angelegt.</div>{{end}}
          {{if eq .ConsumerNotice "weg"}}<div class="message success">Der Verbraucher wurde entfernt.</div>{{end}}
          {{if eq .ConsumerNotice "name"}}<div class="message">Bitte einen Namen angeben.</div>{{end}}
          {{if eq .ConsumerNotice "leistung"}}<div class="message">Die Leistung muss eine positive kW-Zahl sein.</div>{{end}}
          {{if eq .ConsumerNotice "vorlage"}}<div class="message">Vorlagen werden im Onboarding verwaltet, nicht hier.</div>{{end}}
          {{if .CustomConsumers}}<ul class="energy-consumer-list">{{range .CustomConsumers}}<li>
            <div><strong>{{.Name}}</strong><small>{{.KindLabel}}{{if .Power}} · {{.Power}}{{end}} · {{.Flexibility}}</small></div>
            {{if $.CanManageEnergy}}<form method="post" action="/app/energie/verbraucher/entfernen"><input type="hidden" name="asset_id" value="{{.ID}}"><button class="button" type="submit">Entfernen</button></form>{{end}}
          </li>{{end}}</ul>{{end}}
          {{if .CanManageEnergy}}<details class="onboarding-disclosure"><summary>Eigenen Verbraucher hinzufügen <span>Sauna, Werkstatt, Pool …</span></summary>
            <form class="energy-consumer-form" method="post" action="/app/energie/verbraucher">
              <label>Name<input type="text" name="name" maxlength="80" required placeholder="z. B. Sauna"></label>
              <label>Kategorie<select name="kind">{{range .ConsumerKindOptions}}<option value="{{.Value}}">{{.Label}}</option>{{end}}</select></label>
              <label>Leistung in kW<input type="text" name="rated_power_kw" inputmode="decimal" placeholder="z. B. 8,0"></label>
              <label>Flexibilität<select name="flexibility">
                <option value="unknown">noch offen</option>
                <option value="shift">zeitlich verschiebbar</option>
                <option value="throttle">kurz begrenzbar</option>
                <option value="fixed">fest</option>
              </select></label>
              <button class="button primary" type="submit">Verbraucher hinzufügen</button>
            </form>
            <small class="muted">Ohne gesetzte Flexibilität zählt ein Verbraucher nur im Verbrauch, nicht in der Peak-Wirkung. Fehlt die Leistung, rechnen wir mit einem Richtwert der Art (E-Auto, Wallbox und Speicher 3 kW, Wärmepumpe und Warmwasser 1 kW) — Ihre eigene Angabe ist genauer. Eine PV-Anlage zählt nie in die Peak-Wirkung: Erzeugung verschiebt die Bezugsspitze nicht.</small>
          </details>{{end}}
        </div>
        <div class="energy-business-note"><span aria-hidden="true">◎</span><p><strong>Kostenmodell:</strong> voller Produktumfang drei Jahre kostenlos{{if .FreeUntil}} bis {{.FreeUntil}}{{end}}, danach nach heutigem Modell 12 € pro Jahr. Kein Zahlungszwang während des Piloten.</p></div>
      </section>
      <div class="energy-admin">
      <div class="energy-admin-head"><h2>Verwalten und nachweisen</h2><span>Wartung, Messwerte, Zugriff und Fachhilfe – geöffnet, wenn Sie sie brauchen.</span></div>
      <section class="energy-card energy-card-quiet" id="wartung">
        <header class="energy-card-head"><div><h2>Wartung, ohne daran denken zu müssen</h2><p>Fälligkeit, Kontakt, Unterlage und Nachweis bleiben an der Anlage.</p></div><a href="/app/events">Termine</a></header>
        {{if eq .MaintenanceStatus "saved"}}<div class="message success">Wartungsplan gespeichert.</div>{{else if eq .MaintenanceStatus "completed"}}<div class="message success">Erledigt. Der nächste Termin wurde automatisch vorgemerkt.</div>{{else if eq .MaintenanceStatus "invalid"}}<div class="message error">Bitte Anlage, Intervall und Fälligkeit prüfen.</div>{{end}}
        {{if .HasMaintenance}}<div class="energy-maintenance-list">{{range .Maintenance}}{{$plan := .}}<details class="energy-maintenance-row">
          <summary class="energy-maintenance-summary"><span><strong>{{.Title}}</strong><small>{{.AssetName}} · alle {{.IntervalMonths}} Monate{{if .LastCompleted}} · zuletzt {{.LastCompleted}}{{end}}</small></span><span class="energy-status-{{.Tone}}">{{.DueLabel}}</span></summary>
          <div class="energy-maintenance-body">
            {{if or .ContactName .DocumentTitle .IssueTitle}}<nav class="energy-related-links" aria-label="Wartungsnachweise">{{if .ContactName}}<a href="/app/kontakte">{{.ContactName}}</a>{{end}}{{if .DocumentTitle}}<a href="/app/dokumente/{{.DocumentID}}/preview">{{.DocumentTitle}}</a>{{end}}{{if .IssueTitle}}<a href="/app/anliegen/{{.IssueID}}">{{.IssueTitle}}</a>{{end}}</nav>{{end}}
            {{if .EvidenceNote}}<p class="muted">{{.EvidenceNote}}</p>{{end}}
            {{if $.CanManageEnergy}}<form class="energy-inline-form" method="post" action="/app/energie/maintenance">
              <input type="hidden" name="id" value="{{.ID}}"><input type="hidden" name="asset_id" value="{{.AssetID}}"><input type="hidden" name="active" value="true">
              <label>Titel<input name="title" value="{{.Title}}" maxlength="140" required></label><label>Intervall in Monaten<input type="number" name="interval_months" min="1" max="120" value="{{.IntervalMonths}}" required></label>
              <label>Nächste Fälligkeit<input type="date" name="next_due" value="{{.NextDueValue}}" required></label>
              <label>Fachkontakt<select name="contact_id"><option value="">Noch offen</option>{{range $.ContactOptions}}<option value="{{.Value}}"{{if eq .Value $plan.ContactID}} selected{{end}}>{{.Label}}</option>{{end}}</select></label>
              <label>Unterlage<select name="document_id"><option value="">Keine verknüpft</option>{{range $.DocumentOptions}}<option value="{{.Value}}"{{if eq .Value $plan.DocumentID}} selected{{end}}>{{.Label}}</option>{{end}}</select></label>
              <label>Aufgabe / Nachweis<select name="issue_id"><option value="">Keine verknüpft</option>{{range $.IssueOptions}}<option value="{{.Value}}"{{if eq .Value $plan.IssueID}} selected{{end}}>{{.Label}}</option>{{end}}</select></label>
              <label class="wide">Hinweis<input name="evidence_note" value="{{.EvidenceNote}}" maxlength="500" placeholder="Was ist beim nächsten Mal wichtig?"></label>
              <div class="actions"><button class="button" type="submit">Plan speichern</button></div>
            </form>
            <form class="energy-inline-form" method="post" action="/app/energie/maintenance/complete">
              <input type="hidden" name="id" value="{{.ID}}">
              <label>Erledigt am<input type="date" name="completed_at" required></label><label>Nachweis-Aufgabe<select name="issue_id"><option value="">Keine</option>{{range $.IssueOptions}}<option value="{{.Value}}">{{.Label}}</option>{{end}}</select></label>
              <label class="wide">Was wurde gemacht?<input name="evidence_note" maxlength="500" placeholder="Kurz und nachvollziehbar"></label>
              <div class="actions"><button class="button primary" type="submit">Als erledigt eintragen</button></div>
            </form>{{end}}
          </div>
        </details>{{end}}</div>{{else}}<p class="muted">Noch kein wiederkehrender Wartungspunkt hinterlegt.</p>{{end}}
        {{if and .CanManageEnergy .HasAssets}}<details class="energy-compact-create"><summary>Wartung an einer Anlage vormerken</summary><form class="energy-inline-form" method="post" action="/app/energie/maintenance">
          <input type="hidden" name="active" value="true">
          <label>Anlage<select name="asset_id" required>{{range .Assets}}<option value="{{.ID}}">{{.Name}}</option>{{end}}</select></label>
          <label>Titel<input name="title" maxlength="140" placeholder="z. B. Wärmepumpe warten" required></label>
          <label>Alle wie viele Monate?<input type="number" name="interval_months" min="1" max="120" value="12" required></label>
          <label>Erstmals fällig<input type="date" name="next_due" required></label>
          <label>Fachkontakt<select name="contact_id"><option value="">Später wählen</option>{{range .ContactOptions}}<option value="{{.Value}}">{{.Label}}</option>{{end}}</select></label>
          <label>Unterlage<select name="document_id"><option value="">Später verknüpfen</option>{{range .DocumentOptions}}<option value="{{.Value}}">{{.Label}}</option>{{end}}</select></label>
          <div class="actions"><button class="button primary" type="submit">Wartung vormerken</button></div>
        </form></details>{{end}}
      </section>
      <details class="energy-card energy-card-quiet energy-collapsible" id="messwerte">
        <summary class="energy-card-head"><div><h2>Messwerte &amp; Referenz</h2><p>{{if .HasPeaks}}Referenz vorhanden – Details und Import öffnen.{{else}}Home Assistant oder Smart Meter später verbinden.{{end}}</p></div></summary>
        <div class="energy-collapsible-body">
        {{if eq .ImportStatus "added"}}<div class="message success">Smart-Meter-Datei übernommen. Ein erneuter Import derselben Datei erzeugt keine Duplikate.</div>{{else if eq .ImportStatus "duplicate"}}<div class="message">Diese Datei war bereits vorhanden; es wurde nichts doppelt gespeichert.</div>{{else if eq .ImportStatus "invalid"}}<div class="message error">Datei nicht erkannt. Erwartet werden 15-Minuten-Zeilen mit <strong>timestamp</strong> und <strong>import_kwh</strong>.</div>{{end}}
        {{if .HasPeaks}}<div class="energy-reference-grid">{{range .Peaks}}<article class="energy-reference-item"><span>Monatsspitze · {{.Source}}</span><strong>{{.Value}}</strong><span>höchstes abgeschlossenes 15-Minuten-Fenster</span></article>{{end}}</div>{{else}}<p class="muted">Noch keine abgeschlossenen Viertelstunden vorhanden.</p>{{end}}
        {{if .HasComparison}}<div class="energy-quality {{.Comparison.Tone}}"><span aria-hidden="true">{{if eq .Comparison.Tone "good"}}✓{{else}}!{{end}}</span><div><strong>{{.Comparison.Title}}</strong><p>{{.Comparison.Details}}</p></div></div>{{end}}
	        {{if .HasImports}}<div class="energy-assets energy-import-history">{{range .Imports}}<span class="energy-asset">{{.Filename}} · {{.Date}}</span>{{end}}</div>{{end}}
	        {{if .CanManageEnergy}}<form class="energy-import-form" method="post" action="/app/energie/smart-meter" enctype="multipart/form-data">
	          <label><span class="onboarding-legend">Smart-Meter-CSV auswählen</span><input type="file" name="smart_meter_file" accept=".csv,text/csv" required></label>
	          <small>Unterstütztes Profil: <strong>timestamp;import_kwh</strong>, exakt um Minute 00, 15, 30 oder 45. Originaldateien werden nach 30 Tagen, Viertelstundenwerte nach 13 Monaten zur Löschung fällig und spätestens sechs Stunden später entfernt.</small>
	          <button class="button" type="submit">Als Referenz importieren</button>
	        </form>{{end}}
	        {{if .CanManageEnergyData}}<nav class="energy-related-links"><a href="/app/settings/energy-data">Energiedaten ansehen, exportieren oder löschen →</a></nav>{{end}}
	        </div>
	      </details>
      <details class="energy-card energy-card-quiet energy-collapsible" id="betreuung">
        <summary class="energy-card-head"><div><h2>Technische Betreuung</h2><p>{{if .HasCaretakers}}Hausbezogene Hilfe ist eingerichtet.{{else}}Optional eine Vertrauensperson einladen.{{end}}</p></div></summary>
        <div class="energy-collapsible-body">
        <nav class="energy-related-links"><a href="/app/settings/users">Personen verwalten</a></nav>
        {{if .CaretakerChanged}}<div class="message success">Der technische Zugriff für dieses Haus wurde geändert und protokolliert.</div>{{end}}
        {{if eq .CaretakerInviteStatus "invited"}}<div class="message success">Einladung verschickt. Die Person meldet sich mit ihrem eigenen Konto an.</div>{{else if eq .CaretakerInviteStatus "saved_no_mail"}}<div class="message">Zugang gespeichert; die Einladung konnte nicht zugestellt werden.</div>{{else if eq .CaretakerInviteStatus "exists"}}<div class="message">Diese Person gehört bereits zum Haus. Rechte können unten geändert werden.</div>{{else if eq .CaretakerInviteStatus "invalid"}}<div class="message error">Bitte eine gültige E-Mail-Adresse angeben.</div>{{else if eq .CaretakerInviteStatus "error"}}<div class="message error">Der hausbezogene Zugang konnte nicht angelegt werden. Bitte in der Personenverwaltung prüfen.</div>{{end}}
        {{if .HasCaretakers}}<div class="energy-caretaker-list">{{range .Caretakers}}<form class="energy-caretaker" method="post" action="/app/energie/caretaker">
          <input type="hidden" name="email" value="{{.Email}}">
          <div><strong>{{.Name}}</strong><small>{{.Email}}</small></div>
          <label><input type="checkbox" name="scope" value="view"{{if .CanView}} checked{{end}}{{if not .Editable}} disabled{{end}}> ansehen</label>
          <label><input type="checkbox" name="scope" value="configure"{{if .CanConfigure}} checked{{end}}{{if not .Editable}} disabled{{end}}> einrichten</label>
          {{if and $.CanGrantEnergyAccess .Editable}}<button class="button" type="submit">Zugriff speichern</button>{{else if not .Editable}}<a href="/app/settings/users">Zuerst als Portalzugang übernehmen</a>{{end}}
        </form>{{end}}</div>{{else}}<p class="muted">Noch keine weitere Person ist diesem Haus zugeordnet. Laden Sie zuerst ein eigenes Benutzerkonto ein.</p>{{end}}
        {{if .CanInviteEnergyAccess}}<details class="energy-compact-create"><summary>Technische Vertrauensperson einladen</summary><form class="energy-inline-form" method="post" action="/app/energie/caretaker/invite">
          <label>Vorname<input name="first_name" maxlength="80"></label><label>Nachname<input name="last_name" maxlength="80"></label>
          <label class="wide">E-Mail<input type="email" name="email" required></label>
          <label><input type="checkbox" name="scope" value="configure"> darf Messwerte und Geräte einrichten</label>
          <div class="actions"><button class="button primary" type="submit">Hausbezogen einladen</button></div>
        </form></details>{{end}}
        <div class="energy-business-note"><span aria-hidden="true">i</span><p>Technische Vertrauenspersonen dürfen ansehen und einrichten. Den dauerhaft sichtbaren Haus-Schalter dürfen ausschließlich Eigentümer oder Hausadministration bewusst umlegen.</p></div>
        </div>
      </details>
      <details class="energy-card energy-card-quiet energy-collapsible" id="fachhilfe">
        <summary class="energy-card-head"><div><h2>Fachhilfe, wenn sie wirklich nötig ist</h2><p>{{if .HasMeasures}}Eine Hausaufgabe ist in Bearbeitung.{{else}}Bekannte Kontakte statt offener Marktplatz.{{end}}</p></div></summary>
        <div class="energy-collapsible-body">
        <nav class="energy-related-links"><a href="/app/kontakte">Bekannte Kontakte</a></nav>
        <p>Eine Empfehlung wird zuerst als Hausaufgabe angelegt. Dort bleiben Rückfragen, Angebot, Termin und Nachweis zusammen. Sie entscheiden ausdrücklich, welche Daten geteilt werden.</p>
        {{if eq .MeasureStatus "saved"}}<div class="message success">Maßnahmenstand gespeichert und protokolliert.</div>{{else if eq .MeasureStatus "appointment"}}<div class="message error">Für „Termin vereinbart“ bitte einen Termin eintragen.</div>{{else if eq .MeasureStatus "ranges"}}<div class="message error">Für den Abschluss brauchen wir zwei getrennte, gültige Vorher-/Nachher-Zeiträume.</div>{{end}}
        {{if .HasMeasures}}<div class="energy-measure-list">{{range .Measures}}{{$measure := .}}<details class="energy-measure-row">
          <summary class="energy-measure-summary"><span><strong>{{.Title}}</strong><small>{{.StatusLabel}}{{if .ContactName}} · {{.ContactName}}{{end}}{{if .Appointment}} · {{.Appointment}}{{end}}</small></span><span>Öffnen</span></summary>
          <div class="energy-measure-body">
            {{if .HasComparison}}<div class="energy-comparison"><div><small>Vorher · {{.BeforeQuality}}</small><strong>{{.BeforePeak}}</strong></div><span>→ Messbefund →</span><div><small>Nachher · {{.AfterQuality}}</small><strong>{{.AfterPeak}}</strong></div></div>{{end}}
            <nav class="energy-related-links"><a href="/app/anliegen/{{.IssueID}}">Anliegen, Rückfragen &amp; Anhänge öffnen</a></nav>
            {{if $.CanManageEnergy}}<form class="energy-inline-form" method="post" action="/app/energie/measure/update">
              <input type="hidden" name="id" value="{{.ID}}">
              <label>Status<select name="status"><option value="requested"{{if eq .Status "requested"}} selected{{end}}>Anfrage vorbereitet</option><option value="assigned"{{if eq .Status "assigned"}} selected{{end}}>Kontakt ausgewählt</option><option value="scheduled"{{if eq .Status "scheduled"}} selected{{end}}>Termin vereinbart</option><option value="completed"{{if eq .Status "completed"}} selected{{end}}>Abgeschlossen</option><option value="cancelled"{{if eq .Status "cancelled"}} selected{{end}}>Nicht weiterverfolgt</option></select></label>
              <label>Bekannter Fachkontakt<select name="contact_id"><option value="">Noch niemand</option>{{range $.ContactOptions}}<option value="{{.Value}}"{{if eq .Value $measure.ContactID}} selected{{end}}>{{.Label}}</option>{{end}}</select></label>
              <label>Termin<input type="datetime-local" name="appointment_at" value="{{.AppointmentValue}}"></label>
              <label>Angebot / Rückfrage<input name="offer_note" value="{{.OfferNote}}" maxlength="1000" placeholder="Noch keine Zusage oder Zahlung"></label>
              <label class="wide">Ausgeführte Arbeit<input name="work_note" value="{{.WorkNote}}" maxlength="1000"></label>
              <label class="wide">Nachweis / Fachnachweis<input name="evidence_note" value="{{.EvidenceNote}}" maxlength="1000" placeholder="Dateien im verknüpften Anliegen anhängen"></label>
              <label>Vorher von<input type="date" name="before_from" value="{{.BeforeFrom}}"></label><label>Vorher bis<input type="date" name="before_to" value="{{.BeforeTo}}"></label>
              <label>Nachher von<input type="date" name="after_from" value="{{.AfterFrom}}"></label><label>Nachher bis<input type="date" name="after_to" value="{{.AfterTo}}"></label>
              <div class="actions"><button class="button primary" type="submit">Maßnahmenstand speichern</button></div>
            </form>{{end}}
          </div>
        </details>{{end}}</div>{{else}}<p class="muted">Noch keine Energieempfehlung wurde als Hausaufgabe übernommen.</p>{{end}}
        <div class="energy-marketplace-gate"><strong>Marktplatz geschlossen</strong><span>Kein Zahlungsfluss, keine Provision und keine öffentliche Anbieterreihung vor belastbaren Pilotdaten.</span></div>
        </div>
      </details>
      </div>
    </div>
  </main>
{{template "appClose" .}}
{{end}}

{{define "energyData"}}
{{template "appOpen" .}}
  <main id="main-content" tabindex="-1" class="app-main">
    {{template "energyModeStrip" .}}
    <div class="page energy-data-page">
      <header class="energy-data-heading">
        <div>
          <p class="home-eyebrow">Mein Zuhause · Datenkontrolle</p>
          <h1>Energiedaten &amp; Datenschutz</h1>
          <p class="lede">Hier sehen Sie verständlich, was gespeichert ist. Sie können Ihre Energiedaten vollständig mitnehmen oder bewusst löschen.</p>
        </div>
        <div class="energy-data-trust" aria-label="Datenschutz-Grundsätze">
          <span>Home Assistant nur gelesen</span>
          <span>Keine Werbung oder Analyse</span>
          <span>Keine automatische Entscheidung</span>
        </div>
      </header>

      {{if .HistoryDeleted}}<div class="message success"><strong>Der Messverlauf wurde gelöscht.</strong> Smart-Meter-Originale, Viertelstundenwerte, Tarifstände und Messvergleiche sind entfernt. Anlagen und Zuordnungen bleiben erhalten.</div>{{end}}
      {{if .ExportUnavailable}}<div class="message error"><strong>Der direkte Export ist derzeit zu groß.</strong> Bitte wenden Sie sich an den technischen Kontakt; die Daten werden dann sicher bereitgestellt.</div>{{end}}

      <section class="energy-lifecycle" aria-label="Aufbewahrungsfristen">
        <div class="energy-lifecycle-step"><strong>30 Tage</strong><span>Smart-Meter-Originale; Löschung beim nächsten 6-Stunden-Lauf</span></div>
        <div class="energy-lifecycle-step"><strong>13 Monate</strong><span>Viertelstundenwerte; Löschung beim nächsten 6-Stunden-Lauf</span></div>
        <div class="energy-lifecycle-step"><strong>3 Jahre</strong><span>Tarifbewertungen und Energie-Audit; Löschung beim nächsten 6-Stunden-Lauf</span></div>
      </section>

      <section class="energy-data-section energy-data-export">
        <header class="energy-data-section-head">
          <div>
            <h2>Alles mitnehmen</h2>
            <p>Ein ZIP-Paket mit Profil, Anlagen, bestätigten Messwert-Zuordnungen, Viertelstundenwerten, noch vorhandenen Originaldateien, Auswertungen und Energieprotokoll. Home-Assistant-Adresse und Zugangstoken sind ausdrücklich nicht enthalten.</p>
          </div>
          <form method="post" action="/app/settings/energy-data/export" data-download-form>
            <button class="button primary" type="submit" data-busy-label="Export wird erstellt...">Energiedaten exportieren</button>
          </form>
        </header>
        <div class="energy-data-inventory" aria-label="Gespeicherte Energiedaten">
          <div><strong>{{.AssetCount}}</strong><span>Anlagen und Verbraucher</span></div>
          <div><strong>{{.MappingCount}}</strong><span>bestätigte Zuordnungen</span></div>
          <div><strong>{{.IntervalCount}}</strong><span>Viertelstundenwerte</span></div>
          <div><strong>{{.AssessmentCount}}</strong><span>Tarifbewertungen</span></div>
        </div>
      </section>

      <section class="energy-data-section">
        <header class="energy-data-section-head">
          <div>
            <h2>Smart-Meter-Originale</h2>
            <p>Diese Dateien werden nach 30 Tagen, die daraus berechneten Viertelstundenwerte nach 13 Monaten zur Löschung fällig. Der automatische Lauf entfernt sie spätestens sechs Stunden später. Ist ein Import falsch, löschen Sie den Messverlauf und importieren anschließend die korrigierte Datei erneut.</p>
          </div>
          <span class="pill">{{.ImportCount}} vorhanden</span>
        </header>
        {{if .HasImports}}<div class="energy-data-imports">
          {{range .Imports}}<article class="energy-data-import">
            <div><strong>{{.Filename}}</strong><span>{{.Format}} · importiert am {{.Date}}</span></div>
            <small>{{.Size}}</small>
          </article>{{end}}
        </div>{{else}}<p class="energy-data-empty">Keine Smart-Meter-Originaldatei gespeichert.</p>{{end}}
      </section>

      <section class="energy-data-section energy-data-danger">
        <header>
          <h2>Daten löschen</h2>
          <p>Die zwei Stufen trennen Messhistorie von der vollständigen Energie-Einrichtung. Beide Aktionen sind sofort wirksam. Vorgang, ausführende Person, Zeitpunkt und nur betroffene Anzahlen werden protokolliert; gelöschte Inhalte nicht.</p>
        </header>
        <details class="energy-delete-action">
          <summary><span><strong>Nur Messverlauf löschen</strong><small>Originaldateien, Viertelstundenwerte, Tarifstände und Vorher-/Nachher-Messvergleiche. Anlagen und Zuordnungen bleiben.</small></span></summary>
          <form class="energy-delete-form" method="post" action="/app/settings/energy-data/history/delete">
            <label>Zur Bestätigung <strong>MESSVERLAUF LÖSCHEN</strong> eingeben
              <input name="confirmation" autocomplete="off" spellcheck="false" required>
            </label>
            <button class="energy-delete-button" type="submit">Messverlauf löschen</button>
          </form>
        </details>
        <details class="energy-delete-action">
          <summary><span><strong>Ganzes Energieprofil löschen</strong><small>Zusätzlich Haus-Anzeigename, Wohnform, verknüpfte Einheit, Anlagen, Zuordnungen, Wartungspläne und Maßnahmen-Metadaten. Nur der Beginn des kostenlosen Anspruchs bleibt erhalten, damit eine Neueinrichtung die drei Jahre nicht neu startet.</small></span></summary>
          <form class="energy-delete-form" method="post" action="/app/settings/energy-data/profile/delete">
            <label>Zur Bestätigung <strong>ENERGIEPROFIL LÖSCHEN</strong> eingeben
              <input name="confirmation" autocomplete="off" spellcheck="false" required>
            </label>
            <button class="energy-delete-button" type="submit">Energieprofil löschen</button>
          </form>
        </details>
      </section>

      <p class="energy-data-note">Unabhängige Anliegen, Dokumente und Sicherheitsnachweise haben eigene Aufbewahrungsregeln und werden durch diese Energie-Löschung nicht entfernt. Der einmalige Beginn des kostenlosen Anspruchs ist Vertragsmetadatum des Zuhauses und bleibt bis zum Ende dieses Anspruchsverhältnisses erhalten; er enthält keine Messwerte. Verschlüsselte Sicherungskopien bleiben gesperrt und laufen nach dem betrieblichen Backup-Zyklus aus. Details, Empfänger und Ihre Rechte stehen in der <a href="/datenschutz">Datenschutzinformation</a>.</p>
    </div>
  </main>
{{template "appClose" .}}
{{end}}

{{define "errorPage"}}
<!doctype html>
<html lang="de">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  <link rel="icon" type="image/svg+xml" href="/favicon.svg">
  <link rel="shortcut icon" href="/favicon.svg">
  <style>
    /* Scoped to the error page: it is a standalone document (like the sign-in
       and handover-confirmation screens) and loads none of the app shell. */
    :root {
      color-scheme: light;
{{template "designTokens" .}}
    }
    * { box-sizing: border-box; }
    body { margin: 0; color: var(--ink); background: var(--paper); }
    :where(a):focus-visible { outline: 3px solid var(--gold-light); outline-offset: 3px; }
    .hv-error-shell { min-height: 100vh; display: grid; grid-template-rows: auto 1fr auto; }
    .hv-error-top { display: flex; align-items: center; gap: 12px; padding: 16px clamp(18px,5vw,64px); background: rgba(255,254,251,.97); border-bottom: 1px solid rgba(231,224,210,.78); }
    .hv-error-mark { flex: 0 0 auto; width: 50px; height: 38px; display: grid; place-items: center; color: var(--gold); }
    .hv-error-mark .hausv-mark { width: 50px; height: 38px; display: block; stroke: currentColor; stroke-width: 2.15; fill: none; stroke-linecap: round; stroke-linejoin: round; }
    .hv-error-place { min-width: 0; font-family: var(--font-serif); font-weight: 600; font-size: clamp(15px,3.6vw,18px); line-height: 1.25; overflow-wrap: anywhere; }
    .hv-error-main { display: grid; place-items: center; padding: clamp(26px,7vh,64px) clamp(16px,5vw,64px); }
    .hv-error-card { width: min(620px,100%); background: var(--panel); border: 1px solid var(--line); border-radius: var(--radius-xl); box-shadow: var(--shadow-md); padding: clamp(22px,5.5vw,38px); }
    .hv-error-code { display: inline-flex; align-items: center; gap: 9px; margin: 0; color: var(--gold-ink); font-size: 11.5px; font-weight: 800; letter-spacing: .16em; text-transform: uppercase; }
    .hv-error-code::before { content: ""; width: 24px; height: 2px; border-radius: 2px; background: var(--gold); }
    .hv-error-card h1 { margin: 15px 0 0; font-family: var(--font-serif); font-weight: 500; font-size: clamp(31px,7vw,44px); line-height: 1.08; letter-spacing: -.01em; overflow-wrap: anywhere; }
    .hv-error-message { margin: 16px 0 0; font-size: clamp(15.5px,3.8vw,17px); line-height: 1.55; color: var(--ink); overflow-wrap: anywhere; text-wrap: pretty; }
    .hv-error-advice { margin: 9px 0 0; font-size: 14px; line-height: 1.55; color: var(--muted); overflow-wrap: anywhere; text-wrap: pretty; }
    .hv-error-actions { margin-top: 24px; }
    .hv-error-primary { display: inline-flex; align-items: center; justify-content: center; gap: 9px; width: 100%; min-height: 50px; padding: 12px 22px; border-radius: var(--radius-md); background: var(--ink); color: #fff; font-weight: 700; text-decoration: none; }
    .hv-error-primary:hover { background: #000; }
    .hv-error-primary svg { width: 17px; height: 17px; flex: 0 0 auto; stroke: currentColor; stroke-width: 2; fill: none; stroke-linecap: round; stroke-linejoin: round; }
    .hv-error-onward { margin-top: 24px; padding-top: 20px; border-top: 1px solid var(--line); }
    .hv-error-onward-title { margin: 0 0 12px; color: var(--gold-ink); font-size: 11px; font-weight: 800; letter-spacing: .13em; text-transform: uppercase; }
    .hv-error-onward ul { margin: 0; padding: 0; list-style: none; display: grid; gap: 8px; }
    .hv-error-onward a { display: grid; gap: 2px; align-content: center; min-height: 50px; padding: 9px 14px; border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel-soft); color: var(--ink); text-decoration: none; }
    .hv-error-onward a:hover { border-color: var(--gold); }
    .hv-error-onward strong { font-size: 14.5px; font-weight: 750; }
    .hv-error-onward span { color: var(--muted); font-size: 12.5px; line-height: 1.35; }
    .hv-error-foot { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 4px 16px; padding: 15px clamp(18px,5vw,64px); background: rgba(255,254,251,.97); border-top: 1px solid rgba(231,224,210,.78); color: var(--muted); font-size: 12.5px; font-weight: 550; overflow-wrap: anywhere; }
    @media (min-width: 620px) {
      .hv-error-primary { width: auto; }
      .hv-error-onward ul { grid-template-columns: repeat(2,minmax(0,1fr)); }
    }
  </style>
</head>
<body class="hv-error-body">
  <div class="hv-error-shell">
    <header class="hv-error-top">
      <span class="hv-error-mark">{{template "tenantBrandMark" .}}</span>
      <span class="hv-error-place">{{.Tenant.Address}}</span>
    </header>
    <main class="hv-error-main">
      <section class="hv-error-card" aria-labelledby="hv-error-title">
        <p class="hv-error-code">Fehler {{.ErrorStatus}}</p>
        <h1 id="hv-error-title">{{.ErrorHeadline}}</h1>
        <p class="hv-error-message">{{.ErrorMessage}}</p>
        {{if .ErrorAdvice}}<p class="hv-error-advice">{{.ErrorAdvice}}</p>{{end}}
        <div class="hv-error-actions">
          <a class="hv-error-primary" href="{{.ErrorPrimaryURL}}"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M19 12H5"/><path d="m11 18-6-6 6-6"/></svg>{{.ErrorPrimaryLabel}}</a>
        </div>
        {{if .HasErrorLinks}}
        <nav class="hv-error-onward" aria-label="Weitere Bereiche">
          <p class="hv-error-onward-title">Für Sie freigegeben</p>
          <ul>
            {{range .ErrorLinks}}<li><a href="{{.URL}}"><strong>{{.Label}}</strong><span>{{.Hint}}</span></a></li>{{end}}
          </ul>
        </nav>
        {{end}}
      </section>
    </main>
    <footer class="hv-error-foot"><span>{{.Tenant.Address}}</span><span>Hausportal · hausv.org</span></footer>
  </div>
</body>
</html>
{{end}}
`
