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

// PageTemplates holds the remaining public and ancillary {{define}} blocks,
// parsed once at startup. Converted portal pages live in templ components.
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
{{if .TenantBrandLucideSVG}}
{{.TenantBrandLucideSVG}}
{{else if eq .Tenant.BrandIcon "single-home"}}
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
              {{if .DemoLoginEnabled}}<label for="access-code">Zugangscode</label><input id="access-code" name="access_code" type="password" autocomplete="off" required placeholder="Zugangscode der Demo">{{end}}
              <button type="submit">Anmeldelink senden</button>
            </form>
            <p class="foot-note">15 Minuten gültig · nur für eingeladene Personen</p>
          {{else if and (not .Sent) (not .OIDCConfigured)}}
            <div class="notice">Die Anmeldung ist gerade nicht verfügbar.</div>
          {{end}}
          {{if and .EmailLoginAvailable .OIDCConfigured (not .Sent)}}</details>{{end}}
          {{if and .Sent .EmailLoginAvailable}}
            {{if .DemoCodeWrong}}<div class="notice">Der Zugangscode war falsch. Bitte noch einmal versuchen.</div>{{end}}
            <details class="login-retry">
              <summary>Andere Adresse verwenden</summary>
              <form method="post" action="/auth/request">
                <label for="email-retry">E-Mail-Adresse</label>
                <input id="email-retry" name="email" type="email" inputmode="email" autocomplete="email" required placeholder="name@example.com">
                {{if .DemoLoginEnabled}}<label for="access-code-retry">Zugangscode</label><input id="access-code-retry" name="access_code" type="password" autocomplete="off" required placeholder="Zugangscode der Demo">{{end}}
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
  <meta name="description" content="HAUSV verbindet Hauskommunikation, Verwaltung und Live-Energie in einem sicheren Portal – frei self-hosted mit HAUSV Free, bequem mit HAUSV Home oder professionell mit HAUSV Professional.">
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
    /* Hidden on desktop while the hero shows the rotating mark; shown below
       900px, where the 3D mark is not mounted and the bar would be empty. */
    .landing-mark .hausv-mark { width: 70px; height: 42px; display: none; stroke: currentColor; stroke-width: 2.2; fill: none; stroke-linecap: round; stroke-linejoin: round; }
    .landing-menu-toggle { display: none; }
    /* Sticky chrome, layered above the page. */
    .landing-navbar { position: fixed; z-index: 5; top: 0; left: 0; right: 0; }
    .mark3d-veil { position: fixed; z-index: 2; top: 0; left: 0; right: 0; height: 84px; opacity: 0; transition: opacity .35s ease; pointer-events: none; background: linear-gradient(180deg, rgba(12,18,13,.82) 0%, rgba(12,18,13,.66) 34%, rgba(12,18,13,.34) 66%, rgba(12,18,13,.12) 85%, rgba(12,18,13,0) 100%); }
    body[data-hero-mark="out"] .mark3d-veil { opacity: 1; }
    /* The rotating mark is a static block above the hero eyebrow. The inline
       fallback SVG shows until the renderer is ready, then cross-fades away. */
    .mark3d-stage { position: relative; width: min(380px,100%); aspect-ratio: 1008 / 616; margin: 0 0 8px; filter: drop-shadow(0 0 .8px rgba(24,28,22,.06)) drop-shadow(0 3px 9px rgba(24,28,22,.1)); }
    .mark3d-stage canvas, .mark3d-fallback { transition: opacity .45s ease; }
    .mark3d-fallback { position: absolute; inset: 12% 18%; width: 64%; height: 76%; fill: none; stroke: var(--gold-light); stroke-width: 2.2; stroke-linecap: round; stroke-linejoin: round; }
    .mark3d-stage[data-mark3d-state="ready"] .mark3d-fallback { opacity: 0; }
    @media (min-width: 901px) {
      body[data-hero-mark="out"] .landing-mark .hausv-mark { display: block; color: #fff; }
    }
    @media (max-width: 900px) { .mark3d-veil, .mark3d-stage { display: none; } }
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
    .features-section { background: var(--paper); border-top: 1px solid var(--line); border-bottom: 1px solid var(--line); }
    .products-section, .trust-section { background: var(--panel); }
    /* The pricing band used to close the trust section with a hairline; imprint
       now follows trust directly and carries that seam itself. */
    .imprint-section { background: var(--panel); border-top: 1px solid var(--line); }

    /* ---- Panels ---------------------------------------------------------
       One panel language for the whole page: hairline border, 12px radius,
       hairline dividers instead of gaps, so every row shares an edge. */
    .price-panel { border: 1px solid var(--line); border-radius: var(--radius-lg); background: var(--panel); box-shadow: var(--shadow-panel); overflow: hidden; }
    .product-paths { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); align-items: stretch; gap: 18px; }
    .product-path { min-width: 0; display: grid; grid-template-rows: 78px 134px 150px 68px 48px; border: 2px solid var(--line); border-radius: var(--radius-lg); padding: 30px 24px; background: var(--panel); box-shadow: var(--shadow-panel); }
    .product-path.free { border-color: rgba(79,91,80,.32); }
    .product-path.home { border-color: rgba(47,107,74,.42); }
    .product-path.professional { border-color: rgba(200,153,63,.5); }
    .product-path-head { min-width: 0; display: grid; grid-template-columns: 48px minmax(0,1fr); gap: 14px; align-items: center; }
    .product-path-icon { width: 48px; height: 48px; display: grid; place-items: center; border-radius: 50%; background: rgba(47,107,74,.1); color: var(--leaf); }
    .product-path.free .product-path-icon { background: rgba(79,91,80,.09); color: var(--muted); }
    .product-path.professional .product-path-icon { background: rgba(200,153,63,.13); color: var(--gold-ink); }
    .product-path-icon svg { width: 24px; height: 24px; fill: none; stroke: currentColor; stroke-width: 1.8; stroke-linecap: round; stroke-linejoin: round; }
    .product-path-kicker { display: block; margin-bottom: 5px; color: var(--muted); font-size: 10px; font-weight: 900; letter-spacing: .1em; text-transform: uppercase; }
    .product-path h3 { margin: 0; font-family: var(--font-serif); font-size: clamp(24px,1.75vw,25px); font-weight: 600; line-height: 1.05; }
    .product-path > p { margin: 0; padding-top: 18px; color: var(--muted); font-size: 15px; line-height: 1.52; text-wrap: pretty; }
    .product-capabilities { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); grid-template-rows: repeat(2,66px); column-gap: 16px; margin: 0; padding: 0; list-style: none; }
    .product-capabilities li { min-width: 0; display: grid; grid-template-columns: 18px minmax(0,1fr); align-items: start; gap: 8px; border-top: 1px solid var(--line); padding: 11px 0; font-size: 13px; font-weight: 780; line-height: 1.4; }
    .product-capabilities li::before { content: "✓"; width: 18px; height: 1.4em; display: grid; place-items: center; align-self: start; color: var(--leaf); font-weight: 900; }
    .product-path.free .product-capabilities li::before { color: var(--muted); }
    .product-path.professional .product-capabilities li::before { color: var(--gold-ink); }
    .product-path-price { min-width: 0; display: grid; align-content: center; border-top: 1px solid var(--line); }
    .product-path-price strong { font-family: var(--font-serif); font-size: 18px; font-weight: 650; line-height: 1.15; }
    .product-path-price span { margin-top: 4px; color: var(--soft); font-size: 12px; font-weight: 780; line-height: 1.3; }
    .product-path-start { min-height: 46px; display: inline-flex; align-items: center; justify-content: center; border-radius: var(--radius-sm); padding: 11px 16px; background: var(--leaf); color: #fff; text-decoration: none; font-size: 14px; font-weight: 900; }
    .product-path-start:hover { background: #24563b; }
    .product-path.free .product-path-start { border: 1px solid rgba(79,91,80,.38); background: transparent; color: var(--ink); }
    .product-path.free .product-path-start:hover { border-color: var(--muted); background: rgba(79,91,80,.08); }
    .product-path.professional .product-path-start { background: var(--gold-ink); }
    .product-path.professional .product-path-start:hover { background: #72551e; }
    .feature-grid { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 16px; }
    /* Every feature card carries the SAME hairline, radius and shadow. The
       category accent lives in the kicker only — a tinted border on two of ten
       cards read as an unfinished grid rather than as an accent (HAUSV-668). */
    .feature-card { min-width: 0; display: grid; grid-template-columns: 200px minmax(0,1fr); min-height: 210px; border: 1px solid var(--line); border-radius: var(--radius-lg); background: var(--panel); box-shadow: var(--shadow-panel); overflow: hidden; }
    /* The illustrations are square and bring their own paper ground (#fbf4e8,
       sampled from the asset edges). object-fit:contain letterboxed them
       against a colder card ground, so the corner of every non-matching image
       leaked a different tone; cover fills the box to the card's hairline, the
       column widths keep the box near-square so the crop stays inside the
       artwork's margin, and the matching background covers the moment before a
       lazily loaded image has decoded. .feature-card's overflow:hidden clips it
       to the radius, and the grid has no gap, so the media meets the border. */
    .feature-visual { position: relative; min-height: 210px; background: #fbf4e8; overflow: hidden; }
    .feature-visual img { width: 100%; height: 100%; display: block; object-fit: cover; object-position: center; }
    .feature-copy { min-width: 0; display: grid; align-content: center; padding: 24px 24px 25px; }
    .feature-number { margin-bottom: 10px; color: var(--gold-ink); font-size: 11px; font-weight: 900; letter-spacing: .14em; text-transform: uppercase; }
    .feature-card h3 { margin: 0; font-family: var(--font-serif); font-size: clamp(21px,1.8vw,26px); font-weight: 600; line-height: 1.08; text-wrap: balance; }
    .feature-card p { margin: 11px 0 0; color: var(--muted); font-size: 14px; line-height: 1.55; text-wrap: pretty; }
    .shared-core { position: relative; display: grid; grid-template-columns: minmax(190px,.9fr) repeat(4,minmax(0,1fr)); border: 1px solid rgba(47,107,74,.3); border-radius: var(--radius-lg); background: #162018; color: #fff; overflow: hidden; }
    .shared-core::before { content: ""; position: absolute; inset: 0; pointer-events: none; background: linear-gradient(110deg,rgba(200,153,63,.13),transparent 34%,rgba(47,107,74,.12)); }
    .shared-core > * { position: relative; min-width: 0; padding: 22px 20px; }
    .shared-core-head { display: grid; align-content: center; border-right: 1px solid rgba(255,255,255,.16); }
    .shared-core-head span { color: var(--gold-light); font-size: 11px; font-weight: 900; letter-spacing: .12em; text-transform: uppercase; }
    .shared-core-head strong { margin-top: 5px; font-family: var(--font-serif); font-size: 23px; font-weight: 600; line-height: 1.1; }
    .shared-core-item { display: grid; align-content: center; gap: 5px; border-right: 1px solid rgba(255,255,255,.12); }
    .shared-core-item:last-child { border-right: 0; }
    .shared-core-item strong { font-size: 14px; }
    .shared-core-item span { color: rgba(255,255,255,.68); font-size: 12px; line-height: 1.4; }

    /* ---- Disclosures ----------------------------------------------------- */
    .landing-more { border-top: 1px solid var(--line); border-bottom: 1px solid var(--line); }
    .landing-more summary { min-height: 56px; display: flex; align-items: center; justify-content: space-between; gap: 16px; cursor: pointer; color: var(--ink); font-weight: 850; list-style: none; }
    .landing-more summary::-webkit-details-marker { display: none; }
    .landing-more summary::after { content: "+"; color: var(--gold-ink); font-size: 24px; font-weight: 500; }
    .landing-more[open] summary::after { content: "−"; }
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

    /* ---- Imprint and closing --------------------------------------------- */
    .imprint-grid { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 14px; }
    .imprint-card { border: 1px solid var(--line); border-radius: var(--radius-md); padding: 18px 20px; background: var(--panel-soft); }
    .imprint-card strong { display: block; font-size: 15px; }
    .imprint-card p { margin: 8px 0 0; color: var(--muted); line-height: 1.55; }
    .imprint-card .landing-button.imprint-more { min-height: 42px; margin-top: 2px; border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 9px 16px; background: var(--paper); color: var(--ink); }
    .landing-contact { display: flex; align-items: center; justify-content: space-between; gap: 24px; padding: 26px 28px; border-radius: var(--radius-lg); background: var(--nav); color: #fff; }
    .landing-contact strong { display: block; font-family: var(--font-serif); font-weight: 600; font-size: 27px; line-height: 1.15; }
    .landing-contact p { margin: 7px 0 0; color: rgba(255,255,255,.74); }
    .landing-contact .landing-button { flex: 0 0 auto; background: #fff; color: var(--ink); }
    footer { padding: 26px clamp(20px,4vw,42px); color: var(--muted); background: var(--paper); border-top: 1px solid var(--line); }
    footer div { width: min(1180px,100%); margin: 0 auto; display: flex; justify-content: space-between; gap: 16px; flex-wrap: wrap; font-size: 14px; }

    /* ---- Responsive -------------------------------------------------------
       900: the nav collapses. 640: everything stacks. */
    @media (max-width: 1100px) {
      .section-head { grid-template-columns: minmax(0,1fr) minmax(0,320px); }
      .product-paths { grid-template-columns: minmax(0,1fr); }
      .product-path { grid-template-rows: auto; }
      .product-path-head { min-height: 72px; }
      .product-path > p { padding-top: 16px; }
      .product-capabilities { grid-template-rows: repeat(2,minmax(58px,auto)); margin-top: 20px; }
      .product-path-price { min-height: 64px; margin-top: 4px; }
      .product-path-start { margin-top: 4px; }
      .feature-card { grid-template-columns: 180px minmax(0,1fr); }
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
      .landing-menu-toggle svg { width: 18px; height: 18px; flex: 0 0 auto; fill: none; stroke: currentColor; stroke-width: 2; stroke-linecap: round; stroke-linejoin: round; }
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
      .feature-grid { grid-template-columns: minmax(0,1fr); }
      .feature-card { grid-template-columns: 210px minmax(0,1fr); }
      .shared-core { grid-template-columns: repeat(2,minmax(0,1fr)); }
      .shared-core-head { grid-column: 1 / -1; border-right: 0; border-bottom: 1px solid rgba(255,255,255,.16); }
      .shared-core-item:nth-child(3) { border-right: 0; }
      .shared-core-item:nth-child(-n+3) { border-bottom: 1px solid rgba(255,255,255,.12); }
    }
    @media (max-width: 640px) {
      .imprint-grid, .landing-more-grid { grid-template-columns: minmax(0,1fr); }
      /* Stacked, the number belongs beside the line rather than above it —
         four full-width blocks otherwise cost a screen of scrolling. */
      .trust-summary { grid-template-columns: minmax(0,1fr); }
      .trust-line { grid-template-columns: 30px minmax(0,1fr); column-gap: 14px; padding-top: 20px; }
      .trust-line:nth-child(-n+3) { padding-bottom: 20px; border-bottom: 1px solid var(--line); }
      .trust-number { grid-row: 1 / span 2; margin-bottom: 0; }
      .trust-line strong, .trust-line p { grid-column: 2; }
      .landing-contact { align-items: flex-start; flex-direction: column; }
      .product-paths { grid-template-columns: minmax(0,1fr); }
      .shared-core { grid-template-columns: minmax(0,1fr); }
      .shared-core-head { grid-column: auto; }
      .shared-core-item { border-right: 0; border-bottom: 1px solid rgba(255,255,255,.12); }
      .shared-core-item:last-child { border-bottom: 0; }
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
      .feature-card { grid-template-columns: minmax(0,1fr); }
      /* Stacked, the media stays as close to the artwork's own square as the
         page length allows: at 16/9 a cover crop would cut away a third of
         every illustration. */
      .feature-visual { min-height: 0; aspect-ratio: 5 / 4; } .feature-visual img { object-position: center 35%; }
      .feature-copy { padding: 21px 20px 23px; }
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
  <!-- Darkens the top strip once the hero mark has scrolled away, so the header
       keeps contrast over the cream sections. landing.js drives it via
       body[data-hero-mark]. -->
  <div class="mark3d-veil" aria-hidden="true"></div>

  <header class="landing-navbar">
    <div class="landing-nav">
      <!-- Empty on desktop while the hero shows the rotating mark; landing.js
           reveals the flat white mark once the hero mark scrolls out of view. -->
      <a class="landing-brand" href="/" aria-label="hausv.org"><span class="landing-mark">{{template "hausvLandingMark" .}}</span></a>
      <button class="landing-menu-toggle" type="button" data-landing-menu-toggle aria-expanded="false" aria-controls="landing-navigation"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 5h16"/><path d="M4 12h16"/><path d="M4 19h16"/></svg><span>Menü</span></button>
      <nav id="landing-navigation" class="landing-links" aria-label="Navigation">
        <a href="#produkte">Produkte</a>
        <a href="#leistungen">Leistungen</a>
        <a href="#sicherheit">Vertrauen</a>
        <a href="#impressum">Impressum</a>
        <a href="#kontakt" class="js-mail-link" data-mail-local="{{.ContactLocal}}" data-mail-domain="{{.ContactDomain}}">{{.ContactDisplay}}</a>
      </nav>
    </div>
  </header>

  <section class="landing-hero">
    <div class="landing-copy">
      <!-- The rotating 3D mark sits statically above the eyebrow — no scroll
           animation. data-mark3d-mode="solid" switches back to the gold glass.
           The inline SVG is the visible fallback until WebGL has rendered. -->
      <div class="mark3d-stage" data-hausv-mark-3d data-src="/assets/hausv-mark.svg?v={{.AssetVersion}}" data-glass-src="/assets/hausv-mark.glb?v={{.AssetVersion}}" data-min-width="901" aria-hidden="true">
        <svg class="mark3d-fallback" viewBox="0 0 72 42" focusable="false" aria-hidden="true">
          <path d="M9 35h54"/><path d="M11 35V23l8-6 8 6v12"/>
          <path d="M45 35V23l8-6 8 6v12"/><path d="M25 35V17.5L36 9l11 8.5V35"/>
          <path d="M31.5 35v-9h9v9"/><path d="M15.5 27h5"/>
          <path d="M51.5 27h5"/><path d="M31 21h10"/>
        </svg>
      </div>
      <div class="landing-eyebrow">Selbstverwaltung &amp; professionelle Hausverwaltung</div>
      <h1>Ein Hausportal. Alles, was Menschen und Gebäude verbindet.</h1>
      <p class="landing-lead">Kommunikation, Aufgaben, Dokumente, Entscheidungen und Energie an einem Ort – einfach für Bewohner, verlässlich für Hausverwaltungen.</p>
      <div class="landing-actions">
        <a class="landing-button primary" href="#produkte">Produkte ansehen</a>
        <a class="landing-button secondary" href="#leistungen">10 Top-Features entdecken</a>
      </div>
      <p class="landing-access"><svg viewBox="0 0 24 24" aria-hidden="true"><rect x="5" y="10" width="14" height="11" rx="2"/><path d="M8 10V7a4 4 0 0 1 8 0v3"/></svg>Sicher · datensparsam · wahlweise hosted oder selbst betrieben</p>
    </div>
  </section>

  <section id="produkte" class="section products-section">
    <div class="section-inner">
      <div class="section-head">
        <div>
          <p class="section-kicker">Drei Produkte</p>
          <h2>Ein Portal. Drei Wege zum besseren Hausalltag.</h2>
        </div>
        <p class="section-lead">Rollen, Dokumente, Anliegen, Aushänge, Energie und Messwerte mit nachvollziehbarem Verlauf: Sie wählen nur noch Betrieb und Betreuung.</p>
      </div>
      <div class="product-paths">
        <article class="product-path free" id="free">
          <header class="product-path-head"><span class="product-path-icon"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m18 16 4-4-4-4"/><path d="m6 8-4 4 4 4"/><path d="m14.5 4-5 16"/></svg></span><div><span class="product-path-kicker">Quelloffen · selbst betrieben</span><h3>HAUSV Free</h3></div></header>
          <p>Das volle Hausportal für technisch versierte WEGs – frei, anpassbar und vollständig auf der eigenen Infrastruktur betrieben.</p>
          <ul class="product-capabilities"><li>Volle Rollen &amp; Rechte</li><li>Dokumente &amp; Aushänge</li><li>Anliegen mit Verlauf</li><li>Energie &amp; Messwerte</li></ul>
          <div class="product-path-price"><strong>0&nbsp;€ für immer</strong><span>GNU AGPL-3.0 · Quellcode ab Version 1.0</span></div>
          <a class="product-path-start js-mail-link" href="#kontakt" data-mail-local="{{.ContactLocal}}" data-mail-domain="{{.ContactDomain}}" data-mail-subject="HAUSV Free im Eigenbetrieb" data-mail-reveal="false">Eigenbetrieb vormerken</a>
        </article>
        <article class="product-path home" id="home">
          <header class="product-path-head"><span class="product-path-icon"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m3 11 9-8 9 8"/><path d="M5 10v11h14V10"/><path d="M9 21v-6h6v6"/></svg></span><div><span class="product-path-kicker">Für selbstverwaltete WEGs</span><h3>HAUSV Home</h3></div></header>
          <p>Das volle Hausportal für Eigentümer und selbstverwaltete WEGs – sicher gehostet und ohne technischen Aufwand startklar.</p>
          <ul class="product-capabilities"><li>Rollen für Ihre WEG</li><li>Dokumente &amp; Aushänge</li><li>Anliegen mit Verlauf</li><li>Energie &amp; Messwerte</li></ul>
          <div class="product-path-price"><strong>12 Monate kostenlos</strong><span>danach 12&nbsp;€ pro Jahr</span></div>
          <a class="product-path-start" href="/start">HAUSV Home starten</a>
        </article>
        <article class="product-path professional" id="professional">
          <header class="product-path-head"><span class="product-path-icon"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M3 21h18"/><path d="M6 21V5l6-3 6 3v16"/><path d="M9 9h1M14 9h1M9 13h1M14 13h1M9 17h1M14 17h1"/></svg></span><div><span class="product-path-kicker">Für Hausverwaltungen</span><h3>HAUSV Professional</h3></div></header>
          <p>Das volle Hausportal für professionelle Verwaltungen mit mehreren WEGs – gehostet oder auf der eigenen Infrastruktur betrieben.</p>
          <ul class="product-capabilities"><li>Mehrere WEGs &amp; Rollen</li><li>Dokumente &amp; Aushänge</li><li>Anliegen mit Verlauf</li><li>Energie &amp; Messwerte</li></ul>
          <div class="product-path-price"><strong>0&nbsp;€ Grundgebühr</strong><span>25 WE kostenlos · danach Verrechnung je&nbsp;WE&nbsp;/&nbsp;Monat</span></div>
          <a class="product-path-start js-mail-link" href="#kontakt" data-mail-local="{{.ContactLocal}}" data-mail-domain="{{.ContactDomain}}" data-mail-subject="HAUSV Professional kennenlernen" data-mail-reveal="false">Professional anfragen</a>
        </article>
      </div>
      <div class="shared-core" aria-label="Gemeinsamer Vertrauenskern">
        <div class="shared-core-head"><span>Ein gemeinsamer Kern</span><strong>Vertrauen verbindet alle drei Wege.</strong></div>
        <div class="shared-core-item"><strong>Rollen &amp; Rechte</strong><span>Nur sehen, was zur eigenen Aufgabe gehört.</span></div>
        <div class="shared-core-item"><strong>Dokumente</strong><span>Geschützt und passend freigegeben.</span></div>
        <div class="shared-core-item"><strong>Messwerte</strong><span>Lesend, bestätigt und nachvollziehbar.</span></div>
        <div class="shared-core-item"><strong>Verlauf</strong><span>Änderungen bleiben überprüfbar.</span></div>
      </div>
    </div>
  </section>

  <section id="leistungen" class="section features-section">
    <div class="section-inner">
      <div class="section-head">
        <div>
          <p class="section-kicker">Top 10 Features</p>
          <h2>Ein Portal für den gesamten Hausalltag.</h2>
        </div>
        <p class="section-lead">Von der ersten Mitteilung bis zur versendeten Jahresabrechnung: Jede Funktion ist so gestaltet, dass Menschen schnell verstehen, was als Nächstes zu tun ist.</p>
      </div>
      <div class="feature-grid">
        <article class="feature-card"><div class="feature-visual"><img src="/assets/feature-overview.webp" width="560" height="560" alt="" loading="lazy" decoding="async"></div><div class="feature-copy"><span class="feature-number">01 · Orientierung</span><h3>Zentraler Hausüberblick</h3><p>Aufgaben, Termine, Aushänge, offene Anliegen und der Energiezustand des Hauses auf einen Blick.</p></div></article>
        <article class="feature-card"><div class="feature-visual"><img src="/assets/feature-communication.webp" width="560" height="560" alt="" loading="lazy" decoding="async"></div><div class="feature-copy"><span class="feature-number">02 · Kommunikation</span><h3>Aushänge, die ankommen</h3><p>Mitteilungen zentral veröffentlichen, bearbeiten und zielgerichtet für Bewohner sichtbar machen.</p></div></article>
        <article class="feature-card"><div class="feature-visual"><img src="/assets/feature-calendar.webp" width="560" height="560" alt="" loading="lazy" decoding="async"></div><div class="feature-copy"><span class="feature-number">03 · Termine</span><h3>Kalender, der mitgeht</h3><p>Wartungen, Versammlungen und Ablesungen verwalten und per persönlichem Kalender-Feed abonnieren.</p></div></article>
        <article class="feature-card"><div class="feature-visual"><img src="/assets/feature-issues.webp" width="560" height="560" alt="" loading="lazy" decoding="async"></div><div class="feature-copy"><span class="feature-number">04 · Anliegen</span><h3>Vom Mail-Eingang zur Lösung</h3><p>Mit Foto melden oder der Verwaltung schreiben; die KI schlägt Kategorie, Dringlichkeit und Antwort vor – zuweisen und freigeben bleibt bei Ihnen.</p></div></article>
        <article class="feature-card"><div class="feature-visual"><img src="/assets/feature-documents.webp" width="560" height="560" alt="" loading="lazy" decoding="async"></div><div class="feature-copy"><span class="feature-number">05 · Dokumente</span><h3>Geschützt und auffindbar</h3><p>Dokumente nach Haus, Eigentümer und Einheit ablegen – mit Sichtbarkeit, Vorschau, Download und Versionen.</p></div></article>
        <article class="feature-card"><div class="feature-visual"><img src="/assets/feature-handover.webp" width="560" height="560" alt="" loading="lazy" decoding="async"></div><div class="feature-copy"><span class="feature-number">06 · Übergaben</span><h3>Wohnungen digital übergeben</h3><p>Räume, Zustand, Zählerstände, Schlüssel und Anhänge erfassen und dauerhaft digital bestätigen.</p></div></article>
        <article class="feature-card"><div class="feature-visual"><img src="/assets/feature-voting.webp" width="560" height="560" alt="" loading="lazy" decoding="async"></div><div class="feature-copy"><span class="feature-number">07 · Entscheidungen</span><h3>Abstimmungen mit Verlauf</h3><p>Berechtigte Personen stimmen sicher ab; Ergebnis, Abschluss und Protokoll bleiben transparent nachvollziehbar.</p></div></article>
        <article class="feature-card"><div class="feature-visual"><img src="/assets/feature-roles.webp" width="560" height="560" alt="" loading="lazy" decoding="async"></div><div class="feature-copy"><span class="feature-number">08 · Zugriff</span><h3>Kontakte, Rollen und Rechte</h3><p>Verwaltung, Beirat, Bewohner, Eigentümer und Dienstleister erhalten genau die Zugriffe, die sie brauchen; Mitarbeiter betreuen das ganze Portfolio.</p></div></article>
        <article class="feature-card"><div class="feature-visual"><img src="/assets/feature-energy.webp" width="560" height="560" alt="" loading="lazy" decoding="async"></div><div class="feature-copy"><span class="feature-number">09 · Energie</span><h3>Live-Energie verständlich</h3><p>PV, Netz, Speicher, Haus und Verbraucher über Home Assistant verbinden; Stellplätze und Ladepunkte zeigen Ladezustand und Monatswerte.</p></div></article>
        <article class="feature-card"><div class="feature-visual"><img src="/assets/feature-annual.webp" width="560" height="560" alt="" loading="lazy" decoding="async"></div><div class="feature-copy"><span class="feature-number">10 · Abrechnung</span><h3>Jahresabrechnung bis zum Versand</h3><p>Abrechnungslauf berechnen, PDF je Partei erzeugen, unveränderlich archivieren und per E‑Mail zustellen – Belege und Zahlungen lesen Sie per ebInterface und CAMT ein.</p></div></article>
      </div>
      <details id="ausblick" class="landing-more">
        <summary>Was gerade entsteht</summary>
        <div class="landing-more-grid">
          <div><h3>Verrechnung: gemeinsam mit Friendly Customers</h3><ul><li>Die Jahresabrechnung ist da: Lauf, PDF je Partei, Archiv und Versand.</li><li>Buchhaltung, Mahnwesen und Zahlungsläufe folgen – Start später in diesem Jahr.</li><li>Wir bauen sie mit ausgewählten Verwaltungen statt am grünen Tisch.</li></ul></div>
          <div><h3>Energie: kontrolliert statt unbedacht</h3><ul><li>Aktive Steuerung nach bewusster Freigabe – in Entwicklung, Start später in diesem Jahr.</li><li>Dienstleister-Zugänge bleiben rollenbasiert.</li><li>Kein öffentlicher Marktplatz, kein Handel im Hintergrund.</li></ul></div>
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
        <p class="section-lead">Einfach für Gemeinschaft und Zuhause, nachvollziehbar für den jeweiligen Betrieb.</p>
      </div>
      <div class="trust-summary" aria-label="Sicherheitsprinzipien">
        <div class="trust-line"><span class="trust-number">01</span><strong>Getrennte Häuser</strong><p>Eigener Portalbereich, eigene Rollen, eigene Sichtbarkeit.</p></div>
        <div class="trust-line"><span class="trust-number">02</span><strong>Geschützte Dateien</strong><p>Downloads nur über geprüfte App-Wege.</p></div>
        <div class="trust-line"><span class="trust-number">03</span><strong>Datensparsam</strong><p>Nur Angaben, die der Betrieb wirklich braucht.</p></div>
        <div class="trust-line"><span class="trust-number">04</span><strong>KI nur mit Opt-in</strong><p>Keine automatische Auswertung ohne Zustimmung.</p></div>
      </div>
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
        <div class="imprint-card"><strong>Medieninhaber / Betreiber</strong><p><a href="/impressum">{{.OperatorName}}</a> · natürliche Person</p></div>
        <div class="imprint-card"><strong>Kontakt</strong><p><a id="kontakt" class="js-mail-link" href="#kontakt" data-mail-local="{{.ContactLocal}}" data-mail-domain="{{.ContactDomain}}">{{.ContactDisplay}}</a></p></div>
        <div class="imprint-card"><strong>Rechtliches im Detail</strong><p><a class="landing-button imprint-more" href="/impressum">Impressum &amp; Infos</a></p></div>
      </div>
      <div class="landing-contact">
        <div><strong>Home oder Professional?</strong><p>Wir zeigen beide Produkte persönlich und klären gemeinsam, welcher Betrieb zu Ihrem Haus passt.</p></div>
        <a class="landing-button js-mail-link" href="#kontakt" data-mail-local="{{.ContactLocal}}" data-mail-domain="{{.ContactDomain}}" data-mail-subject="HAUSV kennenlernen" data-mail-reveal="false">Gespräch anfragen</a>
      </div>
    </div>
  </section>

  <footer>
    <div><span>hausv.org · sicher, fair und datensparsam</span><span><a href="/datenschutz">Datenschutz</a> · <a href="#impressum">Impressum</a> · <a class="js-mail-link" href="#kontakt" data-mail-local="{{.ContactLocal}}" data-mail-domain="{{.ContactDomain}}">{{.ContactDisplay}}</a> · {{.AppVersion}}</span></div>
  </footer>
</body>
</html>
{{end}}

{{define "homeStartStyles"}}
    :root { color-scheme: light; {{template "designTokens" .}} }
    * { box-sizing: border-box; }
    body { margin: 0; min-height: 100vh; background: var(--paper); color: var(--ink); font-family: var(--font-sans); }
    a { color: var(--leaf); }
    :where(a,button,input):focus-visible { outline: 3px solid var(--gold-light); outline-offset: 3px; }
    .home-start-shell { min-height: 100vh; display: grid; grid-template-rows: auto 1fr auto; }
    .home-start-head, .home-start-main, .home-start-foot { width: min(980px,calc(100% - 36px)); margin: 0 auto; }
    .home-start-head { min-height: 84px; display: flex; align-items: center; justify-content: space-between; gap: 18px; }
    .home-start-brand { display: inline-flex; align-items: center; gap: 12px; color: var(--ink); text-decoration: none; font-weight: 900; }
    .home-start-brand .hausv-mark { width: 58px; height: 40px; stroke: var(--gold-ink); stroke-width: 2.2; fill: none; stroke-linecap: round; stroke-linejoin: round; }
    .home-start-back { font-size: 14px; font-weight: 800; text-decoration: none; }
    .home-start-main { display: grid; grid-template-columns: minmax(0,.9fr) minmax(360px,1.1fr); gap: clamp(28px,6vw,72px); align-items: center; padding: 42px 0 68px; }
    .home-start-kicker { margin: 0 0 14px; color: var(--gold-ink); font-size: 12px; font-weight: 900; letter-spacing: .15em; text-transform: uppercase; }
    .home-start-copy h1 { margin: 0; max-width: 12ch; font-family: var(--font-serif); font-size: clamp(40px,6vw,64px); font-weight: 500; line-height: 1.02; }
    .home-start-lead { margin: 22px 0 0; max-width: 46ch; color: var(--muted); font-size: 18px; line-height: 1.58; }
    .home-start-trust { margin: 28px 0 0; padding: 0; list-style: none; display: grid; gap: 12px; }
    .home-start-trust li { display: flex; gap: 10px; align-items: flex-start; color: var(--muted); line-height: 1.45; }
    .home-start-trust li::before { content: "✓"; color: var(--leaf); font-weight: 900; }
    .home-start-card { border: 1px solid var(--line); border-radius: var(--radius-xl); padding: clamp(24px,4vw,38px); background: var(--panel); box-shadow: var(--shadow-md); }
    .home-start-card h2 { margin: 0; font-family: var(--font-serif); font-size: 30px; font-weight: 600; }
    .home-start-card > p { color: var(--muted); line-height: 1.5; }
    .home-start-form { display: grid; gap: 18px; margin-top: 24px; }
    .home-start-field { display: grid; gap: 8px; }
    .home-start-field label, .home-start-label { font-size: 13px; font-weight: 900; letter-spacing: .06em; text-transform: uppercase; }
    .home-start-field input { width: 100%; min-height: 52px; border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 12px 14px; background: #fff; color: var(--ink); font: inherit; }
    .home-start-field small { color: var(--muted); line-height: 1.4; }
    .home-start-path { display: grid; grid-template-columns: auto minmax(0,1fr); align-items: center; border: 1px solid var(--line); border-radius: var(--radius-sm); background: #fff; overflow: hidden; }
    .home-start-path span { padding-left: 14px; color: var(--muted); white-space: nowrap; }
    .home-start-path input { border: 0; padding-left: 2px; }
    .home-start-check { display: grid; grid-template-columns: 22px minmax(0,1fr); gap: 10px; align-items: start; color: var(--muted); font-size: 14px; line-height: 1.45; }
    .home-start-check input { width: 20px; height: 20px; margin: 0; accent-color: var(--leaf); }
    .home-start-submit { min-height: 52px; border: 0; border-radius: var(--radius-sm); padding: 12px 18px; background: var(--ink); color: #fff; font: inherit; font-weight: 900; cursor: pointer; }
    .home-start-submit:hover { background: #000; }
    .home-start-notice { border: 1px solid rgba(47,107,74,.28); border-radius: var(--radius-md); padding: 18px; background: rgba(47,107,74,.07); color: var(--ink); line-height: 1.5; }
    .home-start-notice strong { display: block; margin-bottom: 5px; }
    .home-start-error { border-color: rgba(160,70,50,.28); background: rgba(160,70,50,.06); }
    .home-start-steps { display: grid; gap: 12px; margin: 24px 0 0; padding: 0; list-style: none; counter-reset: setup; }
    .home-start-steps li { counter-increment: setup; display: grid; grid-template-columns: 34px minmax(0,1fr); gap: 12px; align-items: start; color: var(--muted); line-height: 1.5; }
    .home-start-steps li::before { content: counter(setup); width: 32px; height: 32px; display: grid; place-items: center; border-radius: 50%; background: rgba(47,107,74,.1); color: var(--leaf); font-weight: 900; }
    .home-start-path-result { margin-top: 22px; border: 1px solid var(--line); border-radius: var(--radius-md); padding: 16px 18px; background: var(--panel-soft); }
    .home-start-path-result span { display: block; color: var(--muted); font-size: 12px; font-weight: 800; text-transform: uppercase; letter-spacing: .08em; }
    .home-start-path-result strong { display: block; margin-top: 5px; font-size: 20px; overflow-wrap: anywhere; }
    .home-setup-progress { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 10px; margin: 0 0 22px; }
    .home-setup-step { border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 12px 14px; color: var(--muted); font-size: 13px; line-height: 1.35; }
    .home-setup-step strong { display: block; margin-bottom: 2px; color: var(--ink); font-size: 15px; }
    .home-setup-step.active { border-color: var(--leaf); background: rgba(47,107,74,.07); }
    .home-portal-activation { margin: 0 0 24px; border: 2px solid var(--leaf); border-radius: var(--radius-md); padding: 18px; background: rgba(47,107,74,.06); }
    .home-portal-activation strong { display: block; font-size: 19px; }
    .home-portal-activation p { margin: 7px 0 14px; color: var(--muted); line-height: 1.5; }
    .home-portal-open { display: inline-flex; min-height: 46px; align-items: center; border-radius: var(--radius-sm); padding: 10px 15px; background: var(--ink); color: #fff; font-weight: 900; text-decoration: none; }
    .home-connector-status { margin: 22px 0; border: 2px solid var(--leaf); border-radius: var(--radius-md); padding: 18px; background: rgba(47,107,74,.06); }
    .home-connector-status span { display: block; color: var(--muted); font-size: 12px; font-weight: 900; letter-spacing: .08em; text-transform: uppercase; }
    .home-connector-status strong { display: block; margin-top: 5px; font-size: 20px; }
    .home-connector-status p { margin: 7px 0 0; color: var(--muted); line-height: 1.5; }
    .home-connector-facts { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 10px; margin: 14px 0 22px; }
    .home-connector-facts div { border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 12px; background: #fff; }
    .home-connector-facts span { display: block; color: var(--muted); font-size: 11px; font-weight: 800; letter-spacing: .05em; text-transform: uppercase; }
    .home-connector-facts strong { display: block; margin-top: 4px; overflow-wrap: anywhere; }
    .home-pairing { margin-top: 22px; border: 1px solid var(--gold-light); border-radius: var(--radius-md); padding: 18px; background: var(--panel-soft); }
    .home-pairing h3 { margin: 0; font-size: 18px; }
    .home-pairing-code { display: block; margin: 12px 0; border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 13px; background: #fff; font: 700 14px/1.4 ui-monospace,SFMono-Regular,Consolas,monospace; overflow-wrap: anywhere; user-select: all; }
    .home-command { display: block; margin: 10px 0; border-radius: var(--radius-sm); padding: 13px; background: var(--ink); color: #fff; font: 12px/1.55 ui-monospace,SFMono-Regular,Consolas,monospace; overflow-wrap: anywhere; user-select: all; }
    .home-downloads { display: flex; flex-wrap: wrap; gap: 10px; margin: 16px 0; }
    .home-downloads a { border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 10px 12px; background: #fff; font-size: 13px; font-weight: 800; text-decoration: none; }
    .home-technical { margin-top: 18px; border-top: 1px solid var(--line); padding-top: 16px; }
    .home-technical summary { min-height: 44px; display: flex; align-items: center; color: var(--leaf); font-weight: 900; cursor: pointer; }
    .home-help { margin-top: 22px; border-radius: var(--radius-sm); padding: 14px 16px; background: var(--panel-soft); color: var(--muted); font-size: 14px; line-height: 1.5; }
    .home-help strong { color: var(--ink); }
    .home-connector-actions { display: flex; flex-wrap: wrap; gap: 10px; margin-top: 20px; }
    .home-connector-actions form { margin: 0; }
    .home-connector-action { min-height: 46px; border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 10px 15px; background: #fff; color: var(--ink); font: inherit; font-weight: 900; cursor: pointer; }
    .home-connector-action.primary { border-color: var(--ink); background: var(--ink); color: #fff; }
    .home-connector-action.danger { color: #8f352b; }
    .home-connector-note { color: var(--muted); font-size: 13px; line-height: 1.5; }
    .home-start-foot { padding: 20px 0 28px; border-top: 1px solid var(--line); color: var(--muted); font-size: 13px; }
    @media (max-width: 760px) { .home-start-main { grid-template-columns: minmax(0,1fr); padding-top: 24px; } .home-start-copy h1 { max-width: none; } .home-connector-facts { grid-template-columns: minmax(0,1fr); } }
{{end}}

{{define "homeStart"}}
<!doctype html>
<html lang="de">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}} · hausv.org</title>
  <meta name="description" content="HAUSV Home sicher und in wenigen Schritten vorbereiten.">
  <link rel="icon" type="image/svg+xml" href="/favicon.svg">
  <style>{{template "homeStartStyles" .}}</style>
</head>
<body>
  <div class="home-start-shell">
    <header class="home-start-head"><a class="home-start-brand" href="/">{{template "hausvLandingMark" .}}<span>HAUSV Home</span></a><a class="home-start-back" href="/">Zur Übersicht</a></header>
    <main class="home-start-main">
      <section class="home-start-copy">
        <p class="home-start-kicker">Ihr eigener Bereich</p>
        <h1>Zuhause zuerst sicher anlegen.</h1>
        <p class="home-start-lead">Reservieren Sie Ihre persönliche Adresse und bestätigen Sie Ihre E-Mail. Danach können Sie Ihr Portal sofort öffnen.</p>
        <ul class="home-start-trust"><li>Keine Zahlungsdaten erforderlich</li><li>In diesem Schritt ist kein Zugriff auf Geräte nötig</li><li>Unbestätigte Reservierungen verfallen nach 24 Stunden</li><li>Keine Gerätesteuerung ohne Ihre ausdrückliche Freigabe</li></ul>
      </section>
      <section class="home-start-card" aria-labelledby="home-start-title">
        {{if .Sent}}
          <h2 id="home-start-title">Bitte E-Mail prüfen</h2>
          <div class="home-start-notice"><strong>Wenn die Angaben reservierbar sind, ist der Bestätigungslink unterwegs.</strong>Er gilt 15 Minuten. Diese neutrale Antwort schützt bestehende Reservierungen und Konten.</div>
        {{else}}
          <h2 id="home-start-title">Pfad reservieren</h2>
          <p>Drei Angaben genügen. Der Bereich wird erst nach Ihrer Bestätigung vorbereitet.</p>
          {{if .Expired}}<div class="home-start-notice home-start-error"><strong>Der Link ist nicht mehr gültig.</strong>Starten Sie die Reservierung erneut, um einen neuen Einmal-Link zu erhalten.</div>{{end}}
          <form class="home-start-form" method="post" action="/start">
            <div class="home-start-field"><label for="household-name">Name des Zuhauses</label><input id="household-name" name="household_name" autocomplete="organization" maxlength="80" required placeholder="Zum Beispiel: Zuhause am Stadtpark"></div>
            <div class="home-start-field"><label for="home-path">Gewünschter Pfad</label><div class="home-start-path"><span>hausv.org/</span><input id="home-path" name="slug" inputmode="url" autocomplete="off" minlength="3" maxlength="32" pattern="[a-z0-9](?:[a-z0-9-]{1,30}[a-z0-9])?" required placeholder="mein-zuhause"></div><small>Kleinbuchstaben, Zahlen und Bindestriche; 3 bis 32 Zeichen.</small></div>
            <div class="home-start-field"><label for="owner-email">Eigentümer-E-Mail</label><input id="owner-email" type="email" name="email" autocomplete="email" maxlength="254" required placeholder="name@beispiel.at"></div>
            <label class="home-start-check"><input type="checkbox" name="authority" value="1" required><span>Ich bin Eigentümer oder ausdrücklich berechtigt, dieses Zuhause in HAUSV anzulegen.</span></label>
            <button class="home-start-submit" type="submit">Bestätigungslink anfordern</button>
          </form>
        {{end}}
      </section>
    </main>
    <footer class="home-start-foot">HAUSV Home · datensparsam · zunächst nur beobachten</footer>
  </div>
</body>
</html>
{{end}}

{{define "homeConnectorStart"}}
<!doctype html>
<html lang="de">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}} · hausv.org</title>
  <link rel="icon" type="image/svg+xml" href="/favicon.svg">
  <style>{{template "homeStartStyles" .}}</style>
</head>
<body>
  <div class="home-start-shell">
    <header class="home-start-head"><a class="home-start-brand" href="/">{{template "hausvLandingMark" .}}<span>HAUSV Home</span></a><a class="home-start-back" href="/">Zur Übersicht</a></header>
    <main class="home-start-main">
      <section class="home-start-copy"><p class="home-start-kicker">E-Mail bestätigt</p><h1>{{.HouseholdName}} {{if .PortalActive}}ist bereit.{{else}}ist reserviert.{{end}}</h1><p class="home-start-lead">{{if .PortalActive}}Ihr persönlicher Bereich ist aktiv. Die optionale Energieverbindung können Sie jetzt oder später einrichten.{{else}}Aktivieren Sie jetzt Ihr privates Portal. Die Energieverbindung ist danach ein eigener, optionaler Schritt.{{end}}</p><div class="home-start-path-result"><span>{{if .PortalActive}}Portaladresse{{else}}Reservierte Adresse{{end}}</span><strong>hausv.org{{.PublicPath}}</strong></div></section>
      <section class="home-start-card">
        <div class="home-setup-progress" aria-label="Fortschritt der Einrichtung">
          <div class="home-setup-step {{if not .PortalActive}}active{{end}}"><strong>Schritt 1 von 2</strong>Privates Portal öffnen</div>
          <div class="home-setup-step {{if .PortalActive}}active{{end}}"><strong>Schritt 2 von 2</strong>Energie verbinden, optional</div>
        </div>
        <div class="home-portal-activation">
          {{if .PortalActive}}
          <strong>Ihr privates Portal ist aktiv.</strong><p>Öffnen Sie es jederzeit über Ihren persönlichen Pfad. Die lokale Energieverbindung können Sie unabhängig davon unten verwalten.</p><a class="home-portal-open" href="{{.PublicPath}}/app">Privates Portal öffnen</a>
          {{else}}
          <strong>Privates Portal aktivieren</strong><p>HAUSV legt Ihren Bereich und Ihren Eigentümerzugang gemeinsam an. Danach werden Sie direkt angemeldet.</p><form method="post" action="/start/activate"><button class="home-connector-action primary" type="submit">Portal jetzt aktivieren</button></form>
          {{end}}
        </div>
        <h2>Energieverbindung</h2>
        <p>Optional verbindet ein kleiner Helfer Ihr Energiesystem zu Hause mit HAUSV. Er darf nur lesen und kann keine Geräte steuern.</p>
        <div class="home-connector-status" aria-live="polite"><span>Status</span><strong>{{.ConnectorState}}</strong><p>{{.ConnectorDetail}}</p></div>
        {{if .Connected}}
        <div class="home-connector-facts" aria-label="Verbindungsdetails"><div><span>Zuletzt gemeldet</span><strong>{{.LastSeen}}</strong></div><div><span>Energiesystem</span><strong>{{.HomeAssistantVersion}}</strong></div><div><span>Erkannte Messwerte</span><strong>{{.EntityCount}}</strong></div></div>
        {{end}}
        {{if .PairingCreated}}
        <div class="home-pairing">
          <h3>Verbindung vorbereiten</h3>
          <p class="home-connector-note">Geben Sie diesen Einmal-Code beim lokalen Helfer ein. Er funktioniert bis {{.PairingExpires}} genau einmal.</p>
          <code class="home-pairing-code">{{.PairingCode}}</code>
          <ol class="home-start-steps"><li><span>Laden Sie den Helfer auf den Computer, auf dem Ihr Energiesystem läuft.</span></li><li><span>Starten Sie ihn dort mit dem Einmal-Code. Zugangsdaten bleiben ausschließlich bei Ihnen zu Hause.</span></li><li><span>Kehren Sie zu dieser Seite zurück. Der Status wechselt automatisch nach der ersten erfolgreichen Meldung.</span></li></ol>
          <details class="home-technical"><summary>Technische Anleitung für die Installation</summary><p class="home-connector-note">Home Assistant ist die lokale Software, aus der der Helfer später ausgewählte Messwerte liest. Er benötigt dort einen eigenen Nur-Lese-Zugang.</p><div class="home-downloads"><a href="/downloads/hausv-connector-linux-amd64">Für Intel/AMD Linux laden</a><a href="/downloads/hausv-connector-linux-arm64">Für ARM oder Raspberry Pi laden</a></div><code class="home-command">chmod 700 ./hausv-connector-linux-amd64<br>./hausv-connector-linux-amd64 connector --pairing-code {{.PairingCode}} --home-assistant-url http://homeassistant.local:8123 --home-assistant-token-file ./home-assistant.token</code><p class="home-connector-note">Speichern Sie den in Home Assistant erzeugten Leseschlüssel lokal in <code>home-assistant.token</code>. Für ARM ersetzen Sie den Dateinamen im Befehl.</p></details>
        </div>
        {{else if .PairingPending}}
        <div class="home-start-notice"><strong>Die Verbindung wartet auf den Helfer zu Hause.</strong>Der Einmal-Code wird nur einmal angezeigt. Falls er verloren ging oder ablief, bereiten Sie die Verbindung einfach erneut vor.</div>
        {{end}}
        <div class="home-connector-actions">
          <form method="post" action="/start/connector/pairing"><button class="home-connector-action primary" type="submit">{{if .Connected}}Verbindung erneut vorbereiten{{else}}Energieverbindung vorbereiten{{end}}</button></form>
          {{if .Connected}}<form method="post" action="/start/connector/revoke"><button class="home-connector-action danger" type="submit">Verbindung widerrufen</button></form>{{end}}
        </div>
        <div class="home-help"><strong>Brauchen Sie Hilfe?</strong> Sie können diesen Schritt überspringen und Ihr Portal bereits verwenden. Für die Verbindung braucht die unterstützende Person nur Zugriff auf den Computer Ihres Energiesystems, niemals Ihr HAUSV-Passwort.</div>
      </section>
    </main>
    <footer class="home-start-foot">HAUSV Home · keine Geheimnisse im Portal · keine Steuerung ohne Freigabe</footer>
  </div>
</body>
</html>
{{end}}

{{define "imprint"}}
<!doctype html>
<html lang="de">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  <meta name="description" content="Impressum und Informationen zu hausv.org">
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
    p, dd { line-height: 1.62; }
    .lead { color: var(--muted); font-size: 18px; }
    dl { display: grid; grid-template-columns: 220px minmax(0,1fr); gap: 10px 18px; margin: 26px 0 0; }
    dt { font-weight: 800; }
    dd { margin: 0; color: var(--muted); }
    .mini { margin-top: 34px; max-width: 88ch; color: var(--soft); font-size: 13px; line-height: 1.6; }
    footer { border-top: 1px solid var(--line); padding: 22px 0 36px; color: var(--muted); font-size: 14px; }
    @media (max-width: 620px) { dl { grid-template-columns: 1fr; } dt { margin-top: 8px; } }
  </style>
</head>
<body>
  <header><a href="/">← Zurück zur Startseite</a><span>{{.AppVersion}}</span></header>
  <main>
    <h1>Impressum &amp; Infos</h1>
    <p class="lead">Direkter Kontakt statt anonymer Hotline – und die rechtlichen Angaben zu hausv.org an einem Ort.</p>

    <dl>
      <dt>Medieninhaber / Betreiber</dt><dd>{{.OperatorName}}</dd>
      <dt>Ladungsfähige Anschrift</dt><dd>{{.OperatorAddress}}</dd>
      <dt>Kontakt</dt><dd><a class="js-mail-link" href="/#kontakt" data-mail-local="{{.ContactLocal}}" data-mail-domain="{{.ContactDomain}}">{{.ContactDisplay}}</a></dd>
      <dt>Zweck des Angebots</dt><dd>Information und technischer Pilot für HAUSV Gemeinschaft als Kommunikations- und Transparenzportal sowie HAUSV Zuhause für Hauszustand, Wartung und lesende Energieeinblicke.</dd>
      <dt>Blattlinie</dt><dd>Information über hausv.org, digitale Selbstverwaltung für Mehrparteienhäuser und verständliche, datensparsame Unterstützung für das persönliche Zuhause.</dd>
    </dl>

    <h2>Professionelle Services</h2>
    <p>{{.ProfessionalServicesNotice}}</p>

    <h2>Open Source</h2>
    <p>Der Kern von hausv.org ist quelloffen und steht unter der <a href="https://www.gnu.org/licenses/agpl-3.0.html" rel="noopener noreferrer">GNU AGPL-3.0</a>.</p>

    <p class="mini">Betreiber-Selbstprüfung vom {{.LegalReviewDate}} anhand von <a href="https://www.ris.bka.gv.at/NormDokument.wxe?Abfrage=Bundesnormen&Gesetzesnummer=20001703&Paragraf=5" rel="noopener noreferrer">§ 5 ECG (RIS)</a>, <a href="https://www.usp.gv.at/themen/brancheninformationen/information-und-kommunikation/impressumspflicht-gemaess-para-24-mediengesetz.html" rel="noopener noreferrer">§ 24 MedienG (USP)</a> und der <a href="https://www.dsb.gv.at/" rel="noopener noreferrer">Österreichischen Datenschutzbehörde</a>. Keine externe Zertifizierung oder Rechtsberatung. Details zum Datenschutz in der <a href="/datenschutz">Datenschutzinformation</a>.</p>
  </main>
  <footer>hausv.org · <a href="/datenschutz">Datenschutz</a> · <a href="/">Startseite</a></footer>
  <script src="/assets/landing.js?v={{.AssetVersion}}" defer></script>
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
    <p class="lead">Diese Information beschreibt den tatsächlichen Pilotbetrieb von HAUSV Gemeinschaft und HAUSV Zuhause. Sie ist eine dokumentierte Betreiber-Selbstprüfung nach den Grundsätzen der DSGVO, kein Zertifikat und keine unabhängige Rechtsberatung.</p>

    <div class="status">
      <strong>Dienstleister-Zugang: {{if .ServiceProviderEnabled}}für den begrenzten Pilot freigegeben{{else}}geschlossen{{end}}</strong>
      {{if .ServiceProviderEnabled}}Die Betreiberprüfung {{.ServiceProviderAssessment}} wurde ausdrücklich aktiviert. Dienstleister sehen ausschließlich offene, ihnen zugewiesene Anliegen.{{else}}Ohne ausdrücklich versionierte Betreiberfreigabe können keine Dienstleister eingeladen, zugeordnet oder angemeldet werden.{{end}}
    </div>

    <h2>Wer entscheidet worüber?</h2>
    {{if .PortalClassified}}
      {{if .IsPrivateHome}}
      <p>Bei einem privaten Zuhause entscheidet die Eigentümerin oder der Eigentümer über die eigenen Inhalte der Liegenschaft, Energie-Zuordnungen und ausdrücklich freigegebene Vertrauenspersonen. hausv.org verantwortet den sicheren technischen Portalbetrieb, Konten, Zugriffsschutz und die im Produkt angeforderten Auswertungen. Eine technische Vertrauensperson erhält nur den widerrufbaren, Zugriff auf die Liegenschaft, der im Portal sichtbar freigegeben wurde.</p>
      {{else}}
      <p>Bei einer Hausgemeinschaft entscheidet die Eigentümergemeinschaft beziehungsweise die beauftragte Hausverwaltung über gemeinschaftliche Inhalte und Zwecke. hausv.org verantwortet den sicheren technischen Portalbetrieb, Konten und Zugriffsschutz und verarbeitet Inhalte der Liegenschaft weisungsgebunden, soweit dies für den konkreten Zweck vereinbart ist. Die datenschutzrechtliche Rolle wird deshalb je Zweck bestimmt und nicht pauschal aus einer Produktbezeichnung abgeleitet.</p>
      {{end}}
    {{else}}
    <p>Solange noch keine Wohnform gewählt wurde oder das Energieprofil zurückgesetzt ist, weist hausv.org hier keine Eigentümer- oder Hausgemeinschaftsrolle pauschal zu. Über konkrete Inhalte der Liegenschaft entscheidet die tatsächlich zuständige Eigentümerin, Eigentümergemeinschaft oder beauftragte Verwaltung; hausv.org verantwortet Konten, Zugriffsschutz und den technischen Portalbetrieb jeweils für den konkreten Zweck.</p>
    {{end}}
    <dl>
      <dt>{{if .PortalClassified}}{{if .IsPrivateHome}}Kontakt der Liegenschaft{{else}}Verantwortlicher Hausbetrieb{{end}}{{else}}Kontakt der Liegenschaft{{end}}</dt><dd>{{.HouseContactName}}{{if .HouseContactAddress}}, {{.HouseContactAddress}}{{end}}</dd>
      <dt>{{if .PortalClassified}}{{if .IsPrivateHome}}Betroffenes Zuhause{{else}}Betroffene Liegenschaft{{end}}{{else}}Betroffene Adresse{{end}}</dt><dd>{{.Tenant.Address}}</dd>
      <dt>Kontakt</dt><dd><a href="mailto:{{.HouseContactEmail}}">{{.HouseContactEmail}}</a>{{if .HouseContactPhone}} · {{.HouseContactPhone}}{{end}}</dd>
      <dt>Technischer Betrieb</dt><dd>{{.TechnicalOperatorName}}, {{.TechnicalOperatorAddress}} · <a href="mailto:{{.TechnicalContactEmail}}">{{.TechnicalContactEmail}}</a></dd>
    </dl>

    <h2>Welche Daten und wofür?</h2>
    <ul>
      <li>Identität, Liegenschaftszugehörigkeit, Rollen und Rechte für Anmeldung und Zugriffsschutz.</li>
      <li>Für eine HAUSV-Home-Reservierung werden der gewünschte Pfad, der Name des Zuhauses, die Eigentümer-E-Mail und die ausdrückliche Berechtigungsbestätigung gespeichert. In diesem Schritt werden keine Home-Assistant-Adresse und kein Zugangstoken angenommen.</li>
      <li>Bei der Aktivierung werden der Portalpfad, der Name des Zuhauses, der Aktivierungszeitpunkt und die Eigentümer-Mitgliedschaft der Liegenschaft dauerhaft gespeichert. Portal und Mitgliedschaft entstehen gemeinsam, damit kein Bereich ohne berechtigten Eigentümerzugang veröffentlicht wird.</li>
      <li>Bei der lokalen Connector-Kopplung speichert HAUSV abgeleitete Hashes des Einmal-Codes und Connector-Zugangs sowie Connector-Version, lokale Home-Assistant-Version, Anzahl der Entitäten und Zeitpunkt der letzten Meldung. Für spätere Änderungen der Auswahl bleibt ein auf höchstens 64 Leistung-, Energie- und Ladestandssensoren begrenzter Katalog gespeichert; nur die bestätigten Sensoren werden laufend aktualisiert. Home-Assistant-Adresse, Home-Assistant-Token und alle übrigen Gerätezustände bleiben lokal.</li>
      <li>Aushänge, Termine, Dokumente, Anliegen, Kommentare, Anhänge und Abstimmungen für Kommunikation und Verwaltung der Liegenschaft.</li>
      <li>Anmelde- und Auditdaten für Sicherheit, Fehlerklärung und nachvollziehbare Änderungen.</li>
      <li>Parkplatz- und Ladedaten nur für berechtigte Personen der jeweiligen Liegenschaft.</li>
      <li>Beginn und fixes Ende des kostenlosen Nutzungszeitraums bleiben als Vertrags- und Anspruchsmerkmale des Zuhauses erhalten, damit eine Neueinrichtung oder ein Eigentümerwechsel den Zeitraum nicht neu startet. Neue HAUSV-Home-Portale erhalten zwölf Monate; bestehende Pilot-Enddaten werden nicht verkürzt. Diese Angaben enthalten keine Messwerte.</li>
      {{if .EnergyProfileExists}}<li>Energieprofil mit Wohnform, Anzeigename, verknüpfter Einheit, Anlagen, Wartungsplänen und bestätigten Messwert-Zuordnungen.</li>
      <li>Bei einer direkt betriebenen Home-Assistant-Verbindung werden verfügbare Entitäten zur Auswahl gelesen; dauerhaft gespeichert werden nur bestätigte Zuordnungen. Beim lokalen Self-Service-Connector wird zusätzlich der jeweils letzte ausgewählte Messwert gespeichert und im Energieexport ausgewiesen. Vollständige Home-Assistant-Verläufe werden nicht als eigene Kopie gespeichert.</li>
      <li>Ist der Netzbezug bestätigt, wird er laufend gelesen und je abgeschlossener Viertelstunde ein Mittelwert aufgezeichnet. Gespeichert wird nur dieser Viertelstundenwert mit seiner Güte, nicht der einzelne Messwert.</li>
      <li>Hochgeladene Smart-Meter-Originaldateien, daraus normalisierte Viertelstundenwerte sowie daraus abgeleitete Spitzen, Tarifstände, Empfehlungen und Vorher-/Nachher-Vergleiche.</li>{{end}}
    </ul>
    <p>Die Rechtsgrundlage wird je Zweck gewählt: objektiv notwendige Kernfunktionen auf Grundlage des angeforderten Portalvertrags (Art. 6 Abs. 1 lit. b DSGVO), Zugriffsschutz und eng begrenzte Sicherheitsnachweise auf Grundlage berechtigter Interessen (Art. 6 Abs. 1 lit. f DSGVO) und gesetzliche Pflichten nur, wenn sie im Einzelfall tatsächlich anwendbar und dokumentiert sind (Art. 6 Abs. 1 lit. c DSGVO). Freiwillige Zusatzfreigaben können widerrufen werden. Nicht erforderliche Zweitnutzungen für Werbung, Training oder allgemeine Produktanalyse finden ohne eigene Rechtsgrundlage und ausdrückliche Aktivierung nicht statt. Freitext und Fotos sollen keine Gesundheitsdaten, Ausweiskopien oder andere besonders geschützte Angaben enthalten.</p>
    {{if .EnergyProfileExists}}<p>Die Energieansicht ist zunächst <strong>nur lesend</strong>. Empfehlungen sind nachvollziehbare Hinweise; HAUSV trifft keine ausschließlich automatisierte Entscheidung mit rechtlicher oder ähnlich erheblicher Wirkung. Eine spätere aktive Steuerung bleibt gesondert geschlossen, bis sie bewusst freigegeben und datenschutzrechtlich neu geprüft wurde.</p>{{end}}

    <h2>Quellen, Empfänger und Speicherorte</h2>
    <ul>
      <li>Daten stammen von eingeladenen Personen, der Hausadministration, ausdrücklich verbundenen Home-Assistant-Instanzen und bewusst hochgeladenen Smart-Meter-Dateien.</li>
      <li>Angaben zur HAUSV-Home-Reservierung stammen ausschließlich von der Person, die den Pfad anfordert und ihre E-Mail über den Einmal-Link bestätigt.</li>
      <li>Der lokale Connector übermittelt beim ersten Verbinden einen auf höchstens 64 sichere Energiesensoren begrenzten Katalog. Nach der Bestätigung übermittelt er ausschließlich die ausgewählten Entity-IDs, Werte, Einheiten und Aktualisierungszeiten. Home-Assistant-Adresse, Zugangstoken und andere Gerätezustände werden nicht übertragen.</li>
      <li>Innerhalb einer Liegenschaft sehen nur die jeweils berechtigten Rollen die für ihre Aufgabe notwendigen Bereiche. Technische Vertrauenspersonen sehen oder konfigurieren Energie nur im sichtbar erteilten Umfang und dürfen den Haus-Schalter nicht umlegen. Der Zugriff ist widerrufbar.</li>
      <li>{{.IdentityStorageNotice}}</li>
      <li>{{.WebAccessNotice}}</li>
      <li>{{.BackupStorageNotice}}</li>
      <li>{{.MailDeliveryNotice}}</li>
      <li>Es gibt keine Werbung, keine Analyse-Skripte und keine extern geladenen Web-Schriften.</li>
      <li>Die festen Kartenausschnitte auf der Anmeldeseite und in der Portalnavigation nutzen OpenStreetMap-Kartenkacheln. hausv.org ruft ausschließlich die für das konfigurierte Haus benötigten Kacheln serverseitig ab und speichert sie mindestens sieben Tage zwischen; OpenStreetMap erhält dabei weder die IP-Adresse noch Anmelde- oder Kontodaten der Portalbesuchenden. In den Gebäudeeinstellungen wird eine eingegebene Hausadresse nur nach einem bewussten Klick serverseitig an den OpenStreetMap-Suchdienst Nominatim übermittelt. Erst beim bewussten Öffnen des Kartenlinks baut der Browser eine direkte Verbindung zu OpenStreetMap auf.</li>
      {{if .EnergyProfileExists}}<li>Home-Assistant-Endpunkt und Zugangstoken bleiben in der verschlüsselten Host-Konfiguration. Sie werden weder in der Fachdatenbank noch im Energieexport gespeichert oder angezeigt.</li>{{end}}
    </ul>

    <h2>Aufbewahrung</h2>
    <ul>
      <li>Einmalige E-Mail-Anmelde- und Reservierungslinks: 15 Minuten; OIDC-Anmeldevorgänge: 10 Minuten; alle nur einmal nutzbar.</li>
      <li>Unbestätigte HAUSV-Home-Reservierungen: nach 24 Stunden zur Löschung fällig und spätestens im nächsten stündlichen Bereinigungslauf entfernt. Bestätigte Reservierungen: bis zur Aktivierung des angeforderten Bereichs oder bis zum Widerruf beziehungsweise Löschverlangen. Aktivierter Portalpfad, Zuhause-Name und Eigentümer-Mitgliedschaft: bis zur Beendigung beziehungsweise Löschung des privaten Portals.</li>
      <li>Connector-Einmal-Codes: zehn Minuten gültig und nach erfolgreicher Nutzung verworfen. Der abgeleitete Connector-Zugang, seine Statusdaten, der begrenzte Sensorkatalog und die letzten ausgewählten Energiewerte bleiben bis zum Widerruf, zur Löschung des Energieprofils oder zur Löschung des Zuhause-Bereichs gespeichert. Ein Widerruf beendet den Zugang sofort und entfernt Status und Messwertkopie.</li>
      <li>Sitzungscookie: regulär höchstens 30 Tage oder bis zur Abmeldung beziehungsweise Sperre.</li>
      <li>Liegenschaftszugehörigkeit und Dienstleister-Zugriff: bis zum Entzug; der Zugriff endet sofort.</li>
      <li>Gelöschte Anhangdateien: sofort entfernt; leere Löschmarkierung nach einem Jahr.</li>
      <li>Geschlossene Anliegen samt Kommentaren und Anhängen: jährliche Prüfung, regulär Löschung nach {{.ServiceProviderRetentionYears}} Jahren, sofern keine offene Gewährleistungs-, Rechts- oder Dokumentationspflicht entgegensteht.</li>
      {{if .EnergyProfileExists}}<li>Smart-Meter-Originaldateien werden nach 30 Tagen, normalisierte Viertelstundenwerte nach 13 Monaten und festgehaltene Tarifbewertungen nach drei Jahren zur Löschung fällig. Die technische Löschung erfolgt beim Start und danach alle sechs Stunden, also spätestens innerhalb weiterer sechs Stunden.</li>
      <li>Energieprofil, Anlagen und bestätigte Zuordnungen: bis zur Korrektur, Trennung oder ausdrücklichen Löschung des Energieprofils. Beim lokalen Connector bleiben der begrenzte Sensorkatalog und der letzte Wert je ausgewähltem Sensor bis zum Widerruf oder zur Profillöschung gespeichert; vollständige Home-Assistant-Historien werden nicht kopiert. Aufgezeichnete Viertelstundenmittelwerte unterliegen der Frist von 13 Monaten.</li>{{end}}
      <li>Beginn und fixes Ende des kostenlosen Anspruchs bleiben bis zum Ende des Anspruchs- beziehungsweise Portalverhältnisses erhalten, auch wenn das übrige Energieprofil gelöscht wird.</li>
      <li>Auditdaten werden nach drei Jahren zur Löschung fällig und spätestens beim nächsten sechsstündlichen Bereinigungslauf entfernt; das laufende Protokoll rotiert zusätzlich nach Größe oder Alter.</li>
      <li>Gelöschte Daten können bis zum Ablauf des dokumentierten betrieblichen Backup-Zyklus noch in verschlüsselten Sicherungskopien enthalten sein. Diese Kopien bleiben gesperrt und werden ausschließlich für eine kontrollierte Wiederherstellung verwendet.</li>
    </ul>

    <h2>Ihre Kontrolle und Rechte</h2>
    <p>Betroffene Personen können Information, Auskunft, Berichtigung, Löschung, Einschränkung, Datenübertragbarkeit oder Widerspruch verlangen. Eigentümer und Hausadministration können unter <a href="/app/settings/energy-data">Energiedaten &amp; Datenschutz</a> ein maschinenlesbares ZIP-Paket anfordern, den gesamten Messverlauf löschen oder das Energieprofil zurücksetzen. Unabhängige Anliegen, Dokumente und Sicherheitsnachweise folgen ihren eigenen Fristen und werden dort klar getrennt ausgewiesen.</p>
    <p>Anfragen gehen an den oben genannten Kontakt der Liegenschaft; technisch notwendige Unterstützung leistet hausv.org. Eine erteilte Vertrauenspersonen-Freigabe kann in der Personenverwaltung jederzeit entzogen werden. Beschwerden können an die <a href="https://dsb.gv.at/" rel="noopener noreferrer">Österreichische Datenschutzbehörde</a> gerichtet werden.</p>

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
    .side-brand { flex: 0 0 auto; display: grid; gap: 0; margin: -22px -16px 0; padding: 0 0 8px; }
    .side-map-card { position: relative; min-width: 0; height: 276px; }
    .side-map { position: absolute; inset: 0; width: 100%; height: 100%; overflow: hidden; border: 0; border-radius: 0; background: #d8d2c4; color: inherit; isolation: isolate; }
    .side-map::after { content: ""; position: absolute; inset: 0; z-index: 2; pointer-events: none; background: linear-gradient(180deg, rgba(247,243,234,.08) 0%, rgba(23,32,25,.06) 48%, rgba(23,32,25,.48) 73%, rgba(23,32,25,.93) 94%, var(--nav) 100%); box-shadow: inset 0 -18px 28px rgba(23,32,25,.42); }
    .side-map-tiles { position: absolute; inset: 0; z-index: 1; pointer-events: none; filter: saturate(.54) sepia(.1) contrast(.86) brightness(.97); }
    .side-map-tile { position: absolute; width: 256px; height: 256px; max-width: none; display: block; user-select: none; pointer-events: none; }
    .side-map-fallback { position: absolute; inset: 0; z-index: 1; display: grid; place-items: center; padding-bottom: 72px; color: rgba(23,32,25,.68); background: linear-gradient(135deg,rgba(255,255,255,.3),transparent 55%), #d9d5c9; font-size: 10.5px; font-weight: 750; letter-spacing: .03em; }
    .side-map-pin { position: absolute; left: 50%; top: 50%; z-index: 4; width: 44px; height: 56px; color: var(--gold-light); filter: drop-shadow(0 6px 7px rgba(23,32,25,.38)); transform: translate(-50%,-100%); }
    .side-map-pin-shape { position: absolute; inset: 0; width: 100%; height: 100%; display: block; overflow: visible; fill: var(--nav); stroke: rgba(231,216,177,.88); stroke-width: 1.25; stroke-linejoin: round; }
    .side-map-pin-mark { position: absolute; left: 50%; top: 9px; width: 25px; height: 22px; display: grid; place-items: center; transform: translateX(-50%); }
    .side-map-pin-mark svg { width: 25px; height: 21px; display: block; stroke: currentColor; stroke-width: 2.2; fill: none; stroke-linecap: round; stroke-linejoin: round; }
    .side-map-top { position: absolute; left: 0; right: 0; top: 0; z-index: 7; min-width: 0; padding: 14px 14px 48px; background: linear-gradient(180deg, rgba(23,32,25,.91) 0%, rgba(23,32,25,.67) 54%, rgba(23,32,25,0) 100%); pointer-events: none; }
    .side-map-top > * { pointer-events: auto; }
    .side-place-copy { position: absolute; left: 0; right: 0; bottom: 0; z-index: 6; min-width: 0; display: grid; gap: 8px; padding: 0 18px 14px; color: inherit; }
    .side-address-label { min-width: 0; display: grid; gap: 2px; justify-self: start; color: inherit; font-family: var(--font-sans); line-height: 1.2; text-decoration: none; }
    .side-address-label strong { overflow: hidden; color: rgba(255,255,255,.96); font-size: 20px; font-weight: 750; text-overflow: ellipsis; white-space: nowrap; }
    .side-address-label small { overflow: hidden; color: rgba(255,255,255,.6); font-size: 10.5px; font-weight: 550; text-overflow: ellipsis; white-space: nowrap; }
    .side-address-label:hover small { color: rgba(255,255,255,.86); }
    .side-ruler { height: 12px; border-top: 1px solid rgba(207,171,83,.72); background: linear-gradient(rgba(207,171,83,.72),rgba(207,171,83,.72)) 25% 0/1px 6px no-repeat, linear-gradient(rgba(207,171,83,.82),rgba(207,171,83,.82)) 50% 0/1px 11px no-repeat, linear-gradient(rgba(207,171,83,.72),rgba(207,171,83,.72)) 75% 0/1px 6px no-repeat; }
    .side-portal { min-width: 0; display: flex; align-items: baseline; gap: 4px; justify-self: start; color: inherit; }
    .side-portal strong { font-family: var(--font-sans); color: rgba(255,255,255,.9); font-size: 15px; font-weight: 600; line-height: 1; }
    .side-portal span { color: rgba(255,255,255,.48); font-family: var(--font-sans); font-size: 12px; font-weight: 450; }
    .side-portal { text-decoration: none; }
    .side-portal:hover span { color: rgba(255,255,255,.82); }
    .side-nav { flex: 1 1 auto; min-height: 0; display: grid; align-content: start; gap: 5px; overflow-y: auto; overflow-x: hidden; padding-right: 3px; }
    .side-nav::-webkit-scrollbar { width: 7px; }
    .side-nav::-webkit-scrollbar-thumb { border-radius: var(--radius-pill); background: rgba(255,255,255,.16); }
    .mobile-menu-toggle { display: none; }
    .skip-link { position: fixed; left: 12px; top: 10px; z-index: 1000; min-height: 44px; display: flex; align-items: center; transform: translateY(calc(-100% - 16px)); border: 2px solid var(--gold); border-radius: var(--radius-xs); padding: 10px 14px; color: var(--ink); background: var(--panel); box-shadow: var(--shadow-dialog); font-weight: 850; text-decoration: none; transition: transform .16s ease; }
    .skip-link:focus { transform: none; }
    .nav-item { position: relative; min-height: 44px; display: flex; align-items: center; gap: 12px; padding: 9px 12px; border-radius: var(--radius-xs); color: rgba(255,255,255,.78); text-decoration: none; font-size: 15px; font-weight: 600; }
    .nav-item:hover { color: #fff; background: rgba(255,255,255,.06); }
    .nav-item.active { color: var(--panel); background: var(--nav-2); }
    .nav-item.active .nav-icon { color: var(--gold); }
    .nav-item.active .nav-label { color: var(--panel); font-weight: 700; }
    .nav-item.disabled { color: rgba(255,255,255,.38); cursor: default; }
    .nav-item.disabled:hover { background: transparent; }
    .nav-icon { width: 23px; height: 23px; display: grid; place-items: center; flex: 0 0 auto; color: currentColor; }
    .nav-icon svg { width: 22px; height: 22px; stroke: currentColor; stroke-width: 1.9; fill: none; stroke-linecap: round; stroke-linejoin: round; }
    .nav-icon .energy-ui-icon { width: 100%; height: 100%; }
    .nav-label { min-width: 0; }
    .nav-home-identity { min-width: 0; display: grid; gap: 1px; line-height: 1.08; }
    .nav-home-identity strong { min-width: 0; overflow: hidden; color: inherit; font-size: 15px; font-weight: 700; text-overflow: ellipsis; white-space: nowrap; }
    .nav-home-identity small { min-width: 0; overflow: hidden; color: rgba(255,255,255,.48); font-size: 11px; font-weight: 550; line-height: 1.2; text-overflow: ellipsis; white-space: nowrap; }
    .nav-badge { margin-left: auto; min-width: 25px; height: 22px; display: inline-flex; align-items: center; justify-content: center; border-radius: var(--radius-pill); padding: 0 7px; background: var(--gold); color: #172019; font-size: 11px; font-weight: 900; line-height: 1; }
    .nav-group-label { margin: 10px 12px 2px; color: rgba(255,255,255,.42); font-size: 10px; font-weight: 750; letter-spacing: .1em; text-transform: uppercase; }
    .side-foot { flex: 0 0 auto; margin-top: 0; border-top: 1px solid rgba(255,255,255,.16); padding: 14px 8px 0; display: grid; gap: 8px; }
    .portal-context-switch { position: relative; min-width: 0; }
    .portal-context-switch > summary { min-height: 58px; display: grid; grid-template-columns: 32px minmax(0,1fr) 16px; gap: 10px; align-items: center; border: 1px solid rgba(255,255,255,.28); border-radius: var(--radius-xs); padding: 8px 10px; color: rgba(255,255,255,.95); background: rgba(23,32,25,.5); box-shadow: 0 8px 22px rgba(23,32,25,.14); backdrop-filter: blur(9px); cursor: pointer; list-style: none; }
    .portal-context-switch > summary::-webkit-details-marker { display: none; }
    .portal-context-switch > summary:hover, .portal-context-switch[open] > summary { border-color: rgba(231,197,116,.62); background: rgba(231,197,116,.1); }
    .portal-context-icon { width: 28px; height: 28px; display: grid; place-items: center; border-radius: 50%; color: var(--gold-light); background: rgba(255,255,255,.08); }
    .portal-context-icon svg { width: 17px; height: 17px; fill: none; stroke: currentColor; stroke-width: 1.9; stroke-linecap: round; stroke-linejoin: round; }
    .portal-context-current { min-width: 0; display: grid; gap: 1px; }
    .portal-context-current small { color: rgba(255,255,255,.58); font-size: 9px; font-weight: 750; letter-spacing: .06em; text-transform: uppercase; }
    .portal-context-current strong { font-size: 12.5px; line-height: 1.22; overflow-wrap: anywhere; }
    .portal-context-chevron { width: 8px; height: 8px; justify-self: center; border-right: 1.5px solid currentColor; border-bottom: 1.5px solid currentColor; transform: rotate(45deg) translateY(-2px); transition: transform .15s ease; }
    .portal-context-switch[open] .portal-context-chevron { transform: rotate(225deg) translate(-1px,-1px); }
    .portal-context-menu { position: absolute; left: 0; top: calc(100% + 7px); z-index: 80; width: min(360px,calc(100vw - 28px)); max-height: min(420px,70vh); display: grid; gap: 5px; overflow: auto; border: 1px solid rgba(231,197,116,.38); border-radius: var(--radius-sm); padding: 7px; color: var(--ink); background: var(--surface); box-shadow: var(--shadow-lg); }
    .portal-context-option, .portal-context-option button { width: 100%; min-width: 0; }
    .portal-context-option { margin: 0; }
    .portal-context-option button, .portal-context-option-current { min-height: 48px; display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 8px; align-items: center; border: 0; border-radius: var(--radius-xs); padding: 8px 10px; color: var(--ink); background: transparent; text-align: left; }
    .portal-context-option button { cursor: pointer; }
    .portal-context-option button:hover { background: var(--panel-soft); }
    .portal-context-option-current { background: #f3efe4; }
    .portal-context-option-copy { min-width: 0; display: grid; gap: 2px; }
    .portal-context-option-copy strong, .portal-context-option-copy small { overflow-wrap: anywhere; }
    .portal-context-option-copy strong { font-size: 12px; line-height: 1.25; }
    .portal-context-option-copy small { color: var(--muted); font-size: 10.5px; line-height: 1.3; }
    .portal-context-role { color: var(--gold-ink); font-size: 10px; font-weight: 850; }
    .portal-context-active { color: var(--leaf); font-size: 9px; font-weight: 850; letter-spacing: .06em; text-transform: uppercase; }
    .side-map-attribution { min-width: 0; justify-self: start; margin-left: 54px; color: rgba(255,255,255,.38); font-size: 9px; line-height: 1.2; text-decoration: none; }
    .side-map-attribution:hover { color: rgba(255,255,255,.68); }
    .side-user { min-width: 0; display: grid; grid-template-columns: 42px minmax(0,1fr) auto; gap: 12px; align-items: center; }
    .side-user-copy { min-width: 0; display: grid; gap: 2px; }
    .side-user-meta { min-width: 0; display: flex; align-items: center; gap: 6px; }
    .avatar { width: 42px; height: 42px; border-radius: 50%; display: grid; place-items: center; background: var(--gold); color: #fff; font-weight: 800; border: 1px solid rgba(255,255,255,.25); }
    .side-user strong { display: -webkit-box; max-height: 2.5em; color: #fff; font-size: 14px; line-height: 1.22; overflow: hidden; overflow-wrap: anywhere; -webkit-line-clamp: 2; -webkit-box-orient: vertical; }
    .side-user span, .side-version { color: rgba(255,255,255,.64); font-size: 13px; }
    .version-button { width: auto; min-height: 28px; border: 0; padding: 2px 4px; background: transparent; color: rgba(255,255,255,.48); font: inherit; font-size: 10.5px; font-weight: 650; cursor: pointer; }
    .version-button:hover { color: rgba(255,255,255,.84); }
    .logout-form { margin: 0; }
    .logout-button { width: 44px; height: 44px; display: inline-flex; align-items: center; justify-content: center; border: 1px solid rgba(255,255,255,.2); border-radius: var(--radius-xs); color: rgba(255,255,255,.78); background: transparent; cursor: pointer; }
    .logout-button .energy-ui-icon { width: 19px; height: 19px; }
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
    .energy-nextstep { display: grid; align-content: start; gap: 13px; }
    .energy-nextstep h3 { margin: 0; font-size: 23px; }
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
    .energy-recommendation-trigger { min-height: 44px; display: inline-flex; gap: 7px; align-items: center; border: 1px solid var(--line); border-radius: var(--radius-xs); padding: 8px 12px; color: var(--ink); background: transparent; font: inherit; font-size: 12px; font-weight: 750; cursor: pointer; }
    .energy-recommendation-trigger:hover { border-color: #b9aa80; background: #faf8f1; }
    .energy-recommendation-trigger .energy-ui-icon { width: 17px; height: 17px; color: #496f47; }
    .energy-recommendation-dialog { width: min(760px,calc(100vw - 28px)); }
    .energy-recommendation-dialog .dialog-head { align-items: flex-start; }
    .energy-recommendation-dialog .dialog-body { gap: 22px; }
    .energy-recommendation-summary { display: grid; grid-template-columns: 68px minmax(0,1fr); gap: 18px; align-items: center; }
    .energy-recommendation-summary .energy-nextstep-icon { width: 68px; height: 68px; }
    .energy-recommendation-summary .energy-nextstep-icon .energy-ui-icon { width: 34px; height: 34px; }
    .energy-recommendation-dialog .energy-nextstep-copy { display: grid; gap: 7px; }
    .energy-recommendation-dialog .energy-next-progress-row { margin-top: 2px; }
    .energy-recommendation-dialog .energy-next-why > div { margin-top: 4px; }
    .energy-recommendation-dialog .energy-recommendation-actions { padding-top: 2px; }
    .energy-recommendation-dialog .energy-measure-form { border: 0; padding: 14px; background: transparent; }
    .energy-recommendation-scenarios { display: grid; gap: 12px; border-top: 1px solid var(--line); padding-top: 20px; }
    .energy-recommendation-scenarios > header { display: grid; gap: 3px; }
    .energy-recommendation-scenarios > header p { color: var(--muted); font-size: 13px; }
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
    /* Mittig durch Konstruktion, nicht durch Rechnung: das Pseudoelement erbt
       kein border-box, ein fester Randabstand hinge also an der Randbreite. */
    /* Die Richtung gehört in den Speicher, nicht daneben: die Pfeile laufen auf
       derselben Achse wie der Füllstand. Laden zeigt nach rechts, Entladen nach
       links — gespiegelt wird der ganze Streifen, damit Form und Bewegung nie
       auseinanderlaufen. Das gilt auch, wenn Bewegung abgeschaltet ist. */
    .energy-battery-flow { position: absolute; inset: 2px 4px; display: flex; align-items: center; justify-content: center; gap: 2px; pointer-events: none; }
    @keyframes energy-flow-forward { 0% { opacity: 0; transform: translateX(-5px); } 38% { opacity: .85; } 100% { opacity: 0; transform: translateX(5px); } }
    @media (prefers-reduced-motion: reduce) {
    }
    @media (max-width: 560px) {
    }
    @media (max-width: 340px) {
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
    .energy-system-assets { display: flex; flex-wrap: wrap; gap: 10px; }
    .energy-system-asset { min-width: 190px; display: inline-flex; align-items: center; gap: 11px; border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 13px 15px; background: #fff; font-size: 14px; font-weight: 800; }
    .energy-system-asset .energy-ui-icon { width: 24px; height: 24px; color: #4a5a4e; }
    .energy-system-helper { margin-top: 13px; color: var(--muted); font-size: 12.5px; }
    .energy-service-links { margin-top: 18px; border-top: 1px solid var(--line); padding-top: 16px; }
    .energy-service-links h3 { font-family: var(--font-sans); font-size: 14px; }
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
    .energy-billed { display: grid; grid-template-columns: repeat(auto-fit,minmax(min(146px,100%),1fr)); gap: 1px; margin: 0; border: 1px solid var(--line); border-radius: var(--radius-sm); overflow: hidden; background: var(--line); }
    .energy-billed > div { display: grid; align-content: start; gap: 3px; padding: 13px 14px; background: #fff; }
    .energy-billed span { min-height: 26px; color: var(--muted); font-size: 11px; font-weight: 800; letter-spacing: .05em; line-height: 1.2; text-transform: uppercase; }
    .energy-billed strong { font-family: var(--font-serif); font-size: 28px; font-weight: 600; line-height: 1.05; }
    .energy-billed small { color: var(--muted); font-size: 11.5px; line-height: 1.35; }
    .energy-billed-above { background: rgba(200,153,63,.09); }

    /* Approved energy-cockpit top. This is deliberately scoped to the
       mode/header/live/tariff/next-step viewport; the chart and every section
       after #energieverlauf keep their established 1160px geometry. */
    .energy-ui-icon { width: 1em; height: 1em; display: block; flex: 0 0 auto; background: currentColor; -webkit-mask-position: center; mask-position: center; -webkit-mask-repeat: no-repeat; mask-repeat: no-repeat; -webkit-mask-size: contain; mask-size: contain; }
    .energy-ui-icon-house-plug { -webkit-mask-image: url("/assets/icons/lucide/house-plug.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/house-plug.svg?v={{.AssetVersion}}"); }
    .energy-ui-icon-house { -webkit-mask-image: url("/assets/icons/lucide/house.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/house.svg?v={{.AssetVersion}}"); }
    .energy-ui-icon-solar-panel { -webkit-mask-image: url("/assets/icons/lucide/solar-panel.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/solar-panel.svg?v={{.AssetVersion}}"); }
    .energy-ui-icon-utility-pole { -webkit-mask-image: url("/assets/icons/lucide/utility-pole.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/utility-pole.svg?v={{.AssetVersion}}"); }
    .energy-ui-icon-battery { -webkit-mask-image: url("/assets/icons/lucide/battery.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/battery.svg?v={{.AssetVersion}}"); }
    .energy-ui-icon-binoculars { -webkit-mask-image: url("/assets/icons/lucide/binoculars.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/binoculars.svg?v={{.AssetVersion}}"); }
    .energy-ui-icon-check { -webkit-mask-image: url("/assets/icons/lucide/check.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/check.svg?v={{.AssetVersion}}"); }
    .energy-ui-icon-alert { -webkit-mask-image: url("/assets/icons/lucide/triangle-alert.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/triangle-alert.svg?v={{.AssetVersion}}"); }
    .energy-ui-icon-info { -webkit-mask-image: url("/assets/icons/lucide/info.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/info.svg?v={{.AssetVersion}}"); }
    .energy-ui-icon-play { -webkit-mask-image: url("/assets/icons/lucide/play.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/play.svg?v={{.AssetVersion}}"); }
    .energy-ui-icon-pencil { -webkit-mask-image: url("/assets/icons/lucide/pencil.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/pencil.svg?v={{.AssetVersion}}"); }
    .energy-ui-icon-log-out { -webkit-mask-image: url("/assets/icons/lucide/log-out.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/log-out.svg?v={{.AssetVersion}}"); }
    .energy-ui-icon-circle-help { -webkit-mask-image: url("/assets/icons/lucide/circle-help.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/circle-help.svg?v={{.AssetVersion}}"); }
    .energy-ui-icon-shield-check { -webkit-mask-image: url("/assets/icons/lucide/shield-check.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/shield-check.svg?v={{.AssetVersion}}"); }
    .energy-ui-icon-workflow { -webkit-mask-image: url("/assets/icons/lucide/workflow.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/workflow.svg?v={{.AssetVersion}}"); }
    .energy-ui-icon-server-cog { -webkit-mask-image: url("/assets/icons/lucide/server-cog.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/server-cog.svg?v={{.AssetVersion}}"); }
    .energy-ui-icon-circle-check { -webkit-mask-image: url("/assets/icons/lucide/circle-check.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/circle-check.svg?v={{.AssetVersion}}"); }
    .energy-ui-icon-chevron-right { -webkit-mask-image: url("/assets/icons/lucide/chevron-right.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/chevron-right.svg?v={{.AssetVersion}}"); }
    .energy-ui-icon-chevron-down { -webkit-mask-image: url("/assets/icons/lucide/chevron-down.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/chevron-down.svg?v={{.AssetVersion}}"); }
    .energy-ui-icon-ellipsis { -webkit-mask-image: url("/assets/icons/lucide/ellipsis.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/ellipsis.svg?v={{.AssetVersion}}"); }
    .energy-ui-icon-clock { -webkit-mask-image: url("/assets/icons/lucide/clock-3.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/clock-3.svg?v={{.AssetVersion}}"); }
    .energy-ui-icon-car-front { -webkit-mask-image: url("/assets/icons/lucide/car-front.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/car-front.svg?v={{.AssetVersion}}"); }
    .energy-ui-icon-plug-zap { -webkit-mask-image: url("/assets/icons/lucide/plug-zap.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/plug-zap.svg?v={{.AssetVersion}}"); }
    .energy-ui-icon-heater { -webkit-mask-image: url("/assets/icons/lucide/heater.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/heater.svg?v={{.AssetVersion}}"); }
    .energy-ui-icon-fan { -webkit-mask-image: url("/assets/icons/lucide/fan.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/fan.svg?v={{.AssetVersion}}"); }
    .energy-ui-icon-washing-machine { -webkit-mask-image: url("/assets/icons/lucide/washing-machine.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/washing-machine.svg?v={{.AssetVersion}}"); }
    .energy-ui-icon-flame { -webkit-mask-image: url("/assets/icons/lucide/flame.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/flame.svg?v={{.AssetVersion}}"); }
    .energy-ui-icon-drill { -webkit-mask-image: url("/assets/icons/lucide/drill.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/drill.svg?v={{.AssetVersion}}"); }
    .energy-ui-icon-waves-ladder { -webkit-mask-image: url("/assets/icons/lucide/waves-ladder.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/waves-ladder.svg?v={{.AssetVersion}}"); }
    .energy-ui-icon-square-parking { -webkit-mask-image: url("/assets/icons/lucide/square-parking.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/square-parking.svg?v={{.AssetVersion}}"); }
    .energy-ui-icon-plug { -webkit-mask-image: url("/assets/icons/lucide/plug.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/plug.svg?v={{.AssetVersion}}"); }
    .energy-ui-icon-grip-vertical { -webkit-mask-image: url("/assets/icons/lucide/grip-vertical.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/grip-vertical.svg?v={{.AssetVersion}}"); }
    .energy-ui-icon-plus { -webkit-mask-image: url("/assets/icons/lucide/plus.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/plus.svg?v={{.AssetVersion}}"); }
    .energy-ui-icon-search { -webkit-mask-image: url("/assets/icons/lucide/search.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/search.svg?v={{.AssetVersion}}"); }
    .energy-ui-icon-x { -webkit-mask-image: url("/assets/icons/lucide/x.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/x.svg?v={{.AssetVersion}}"); }
    .energy-ui-icon-snowflake { -webkit-mask-image: url("/assets/icons/lucide/snowflake.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/snowflake.svg?v={{.AssetVersion}}"); }
    .energy-ui-icon-shower-head { -webkit-mask-image: url("/assets/icons/lucide/shower-head.svg?v={{.AssetVersion}}"); mask-image: url("/assets/icons/lucide/shower-head.svg?v={{.AssetVersion}}"); }

    @media (min-width: 901px) {
      .app-shell:has(.energy-cockpit-main) { display: block; padding-left: 264px; }
      .app-shell:has(.energy-cockpit-main) > .sidebar { position: fixed; inset: 0 auto 0 0; width: 264px; height: auto; }
    }
    .energy-cockpit-main .energy-mode-strip { min-height: 68px; display: flex; gap: 16px; align-items: center; padding-block: 10px; padding-inline: max(18px,calc(50% - 686px)); color: var(--ink); background: #fffefa; border-bottom: 1px solid var(--line); box-shadow: 0 5px 18px rgba(31,39,32,.04); }
    .energy-cockpit-main .energy-mode-strip.active { color: #5f2f28; background: #fffaf7; border-bottom-color: rgba(145,73,61,.24); }
    .energy-mode-state { min-height: 46px; display: inline-flex; gap: 10px; align-items: center; border: 1px solid var(--line); border-radius: 10px; padding: 7px 26px 7px 10px; background: rgba(255,255,255,.56); }
    .energy-cockpit-main .energy-mode-icon { width: 28px; height: 28px; flex: 0 0 28px; border: 0; border-radius: 50%; color: #fff; background: #4d7c4b; }
    .energy-cockpit-main .energy-mode-icon .energy-ui-icon { width: 16px; height: 16px; }
    .energy-cockpit-main .energy-mode-strip.active .energy-mode-icon { color: #fff; background: #8f4f45; }
    .energy-cockpit-main .energy-mode-copy { min-width: 0; display: flex; gap: 10px; align-items: center; border: 0; padding: 0; }
    .energy-cockpit-main .energy-mode-copy strong { font-family: var(--font-sans); font-size: 14px; font-weight: 800; white-space: nowrap; }
    .energy-cockpit-main .energy-mode-copy span { display: flex; gap: 10px; align-items: center; color: var(--muted); font-size: 12.5px; white-space: nowrap; }
    .energy-cockpit-main .energy-mode-copy span::before { content: ""; width: 4px; height: 4px; flex: 0 0 auto; border-radius: 50%; background: #b9b2a4; }
    .energy-cockpit-main .energy-mode-control, .energy-cockpit-main .energy-mode-strip > form, .energy-cockpit-main .energy-mode-capability { margin-left: auto; }
    .energy-cockpit-main .energy-mode-action { min-height: 46px; display: inline-flex; gap: 9px; align-items: center; border-color: var(--line); border-radius: 10px; padding: 8px 20px; color: var(--ink); background: transparent; font-size: 13px; }
    .energy-cockpit-main .energy-mode-action .energy-ui-icon { width: 16px; height: 16px; }
    .energy-cockpit-main .energy-mode-action:hover { border-color: #a99e89; background: #fff; }
    .energy-cockpit-main .energy-mode-capability { color: var(--muted); }
    .energy-cockpit-main .energy-mode-popover { color: var(--ink); background: var(--surface); }

    .energy-cockpit-top { width: 100%; display: grid; gap: 18px; }
    .energy-cockpit-top .energy-heading { width: 100%; min-width: 0; display: flex; align-items: baseline; gap: 14px; padding: 2px 2px 0; }
    .energy-cockpit-top .energy-heading-copy { display: flex; align-items: baseline; gap: 12px; min-width: 0; }
    .energy-cockpit-top .energy-heading-side { margin-left: auto; display: flex; align-items: center; gap: 12px; }
    .energy-cockpit-top .energy-heading-breadcrumb { display: flex; flex-wrap: wrap; gap: 5px 7px; color: var(--muted); font-size: 12.5px; line-height: 1.35; }
    .energy-cockpit-main .energy-page { padding-top: 26px; }
    .energy-cockpit-top .energy-heading h1 { margin: 0; font-size: clamp(26px,2.4vw,32px); font-weight: 550; letter-spacing: -.02em; line-height: 1.05; }
    .energy-cockpit-top .energy-heading-unit-row { display: flex; gap: 5px; align-items: center; margin: 0; color: var(--muted); font-size: 12.5px; }
    .energy-cockpit-top .energy-heading-unit { margin: 0; color: inherit; font-size: inherit; }
    .energy-cockpit-top .energy-heading-context { position: absolute; width: 1px; height: 1px; margin: -1px; overflow: hidden; clip: rect(0,0,0,0); white-space: nowrap; }
    .energy-cockpit-top .energy-heading-action { width: 50px; height: 50px; min-height: 50px; justify-content: center; border-color: var(--line); border-radius: 10px; padding: 0; background: rgba(255,255,255,.5); }
    .energy-cockpit-top .energy-heading-action > span:last-child { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0,0,0,0); white-space: nowrap; }
    .energy-cockpit-top .energy-heading-action .energy-ui-icon { width: 20px; height: 20px; }
    .energy-cockpit-top .energy-health { display: grid; grid-template-columns: minmax(0,1fr); row-gap: 22px; }
    .energy-cockpit-top .energy-lead-side { display: contents; }
    .energy-cockpit-top .energy-live { grid-column: 1; grid-row: 1; min-width: 0; overflow: visible; border-color: #ded9ce; border-radius: 14px; box-shadow: 0 12px 32px rgba(32,40,33,.06); }
    /* ---- Data-driven energy flow (HAUSV-439) ------------------------- */
    .energy-flow-area { position: relative; display: grid; grid-template-columns: minmax(0,1fr); gap: 0 26px; align-items: start; padding: 12px 26px 36px; }
    /* The 258px rail column exists only while the renderer mounted a rail. */
    @media (min-width: 1180px) { .energy-flow-area.has-rail { grid-template-columns: minmax(0,1fr) 258px; } }
    .energy-flow-area .energy-flow-fallback { padding: 6px 0 12px; }
    .energy-flow-area.is-enhanced .energy-flow-fallback { display: none; }
    svg.energy-flow-ribbons { position: absolute; inset: 0; width: 100%; height: 100%; pointer-events: none; }
    .energy-flow-band { opacity: .2; }
    .energy-flow-dots { opacity: .86; animation: energy-flow-dots 1.15s linear infinite; }
    @keyframes energy-flow-dots { to { stroke-dashoffset: -10.1; } }
    @media (prefers-reduced-motion: reduce) { .energy-flow-dots { animation: none; } }
    .energy-flow-grid2 { --energy-flow-node-width:258px; --energy-flow-node-height:80px; position: relative; display: grid; grid-template-columns: var(--energy-flow-node-width) minmax(26px,1fr) var(--energy-flow-node-width) minmax(26px,1fr); grid-template-rows: auto 64px auto 64px auto; align-items: center; justify-items: center; }
    .energy-flow-slot-top { grid-column: 3; grid-row: 1; display: flex; gap: 14px; justify-content: center; width: max-content; justify-self: center; }
    .energy-flow-slot-left { grid-column: 1; grid-row: 3; width: 100%; }
    .energy-flow-slot-bottom { grid-column: 3; grid-row: 5; display: flex; justify-content: center; width: max-content; justify-self: center; }
    .energy-flow-slot-top .energy-flow-tile, .energy-flow-slot-bottom .energy-flow-tile { width: var(--energy-flow-node-width); }
    .energy-flow-tile { position: relative; z-index: 1; width: var(--energy-flow-node-width); min-height: var(--energy-flow-node-height); box-sizing: border-box; border: 2px solid var(--energy-node-color,#d8d2c7); border-radius: 12px; background: #fffefa; padding: 11px 14px; display: grid; grid-template-columns: 38px minmax(0,1fr); align-items: center; gap: 10px; }
    .energy-flow-tile .ico { width: 38px; height: 38px; border-radius: 50%; display: grid; place-items: center; color: var(--energy-node-color,#4a5a4e); background: #f0f2ec; }
    .energy-flow-tile .ico .energy-ui-icon { width: 21px; height: 21px; }
    .energy-flow-tile strong { font-family: var(--font-serif); font-weight: 550; font-size: 17px; line-height: 1; white-space: nowrap; font-variant-numeric: tabular-nums; }
    .energy-flow-tile > div { min-width: 0; }
    .energy-flow-tile > div > span { display: block; margin-top: 4px; overflow: hidden; font-size: 11px; line-height: 1.25; color: var(--muted); text-overflow: ellipsis; white-space: nowrap; }
    .energy-flow-tile > div > .energy-flow-secondary { margin-top: 2px; font-size: 10px; }
    .energy-flow-secondary b, .energy-flow-secondary-copy b { display: inline; color: inherit; font-family: var(--font-sans); font-size: inherit; font-weight: 750; white-space: nowrap; }
    .energy-flow-tile span.u, .energy-flow-big span.u { display: inline; margin: 0; font-family: var(--font-sans); font-size: 10px; font-weight: 800; letter-spacing: .04em; color: #8a8e80; vertical-align: 6px; }
    .energy-flow-tile.k-pv { background: #f7faf3; }
    .energy-flow-tile.k-pv .ico { background: #edf3ea; }
    .energy-flow-tile.k-grid { background: #fffaf0; }
    .energy-flow-tile.k-grid .ico { background: #faf4e5; }
    .energy-flow-tile.k-batt { background: #f3f7f8; }
    .energy-flow-tile.k-batt .ico { background: #eaf1f3; }
    .energy-flow-tile.editable { cursor: pointer; transition: border-color .15s ease, background-color .15s ease; }
    .energy-flow-tile.editable .ico > .energy-ui-icon { grid-area: 1 / 1; transition: opacity .15s ease, transform .15s ease; }
    .energy-flow-tile.editable .energy-flow-icon-edit { opacity: 0; transform: scale(.86); }
    .energy-flow-tile.editable:hover, .energy-flow-tile.editable:focus-visible { background: #fffaf0; box-shadow: 0 0 0 2px color-mix(in srgb,var(--energy-node-color) 22%,transparent); }
    .energy-flow-tile.editable:hover .energy-flow-icon-default, .energy-flow-tile.editable:focus-visible .energy-flow-icon-default { opacity: 0; }
    .energy-flow-tile.editable:hover .energy-flow-icon-edit, .energy-flow-tile.editable:focus-visible .energy-flow-icon-edit { opacity: 1; transform: scale(1); }
    .energy-flow-hub2 { grid-column: 3; grid-row: 3; width: var(--energy-flow-node-width); z-index: 2; }
    .energy-flow-hub2 .ico { background: #fbf1ed; }
    .energy-flow-rail { position: relative; z-index: 1; align-self: start; display: grid; gap: 8px; align-content: start; }
    .energy-flow-rail > .microlabel { position: absolute; top: -22px; left: 2px; font-size: 11px; font-weight: 800; letter-spacing: .06em; text-transform: uppercase; color: var(--muted); }
    .energy-flow-big { width: 258px; height: 80px; box-sizing: border-box; border: 2px solid var(--energy-node-color,#d8d2c7); border-radius: 11px; background: #fffefa; padding: 8px 10px; display: grid; grid-template-columns: minmax(0,1fr) 16px; align-items: center; gap: 9px; transition: background-color .15s ease, box-shadow .15s ease; }
    .energy-flow-main { min-width: 0; width: 100%; display: grid; grid-template-columns: 34px minmax(0,1fr); align-items: center; gap: 9px; border: 0; padding: 0; color: inherit; background: transparent; text-align: left; font: inherit; }
    button.energy-flow-main { cursor: pointer; }
    .energy-flow-big .ico { position: relative; width: 34px; height: 34px; border-radius: 50%; display: grid; place-items: center; color: var(--energy-node-color,#4a5a4e); background: #f0f2ec; }
    .energy-flow-big .ico .energy-ui-icon { width: 19px; height: 19px; grid-area: 1 / 1; transition: opacity .15s ease, transform .15s ease; }
    .energy-flow-big .energy-flow-icon-edit { opacity: 0; transform: scale(.86); }
    .energy-flow-copy { min-width: 0; display: grid; gap: 3px; }
    .energy-flow-big b { display: block; overflow: hidden; font-size: 12.5px; line-height: 1.35; text-overflow: ellipsis; white-space: nowrap; }
    .energy-flow-subtitles { display: grid; min-width: 0; }
    .energy-flow-subtitles > span { grid-area: 1 / 1; display: block; overflow: hidden; color: var(--muted); font-size: 11px; line-height: 1.4; text-overflow: ellipsis; white-space: nowrap; transition: opacity .15s ease; }
    .energy-flow-secondary-copy { display: block; overflow: hidden; color: var(--muted); font-size: 10px; line-height: 1.3; text-overflow: ellipsis; white-space: nowrap; }
    .energy-flow-big .energy-flow-secondary-copy b { display: inline; color: inherit; font-family: var(--font-sans); font-size: inherit; font-weight: 750; white-space: nowrap; }
    .energy-flow-subtitles .energy-flow-edit-copy { color: #765f1d; font-weight: 750; opacity: 0; }
    .energy-flow-big.editable:hover, .energy-flow-big.editable:focus-within { background: #fffaf0; box-shadow: 0 0 0 2px color-mix(in srgb,var(--energy-node-color) 22%,transparent); }
    .energy-flow-big.editable:hover .energy-flow-icon-default, .energy-flow-big.editable:focus-within .energy-flow-icon-default,
    .energy-flow-big.editable:hover .energy-flow-state-copy, .energy-flow-big.editable:focus-within .energy-flow-state-copy { opacity: 0; }
    .energy-flow-big.editable:hover .energy-flow-icon-edit, .energy-flow-big.editable:focus-within .energy-flow-icon-edit { opacity: 1; transform: scale(1); }
    .energy-flow-big.editable:hover .energy-flow-edit-copy, .energy-flow-big.editable:focus-within .energy-flow-edit-copy { opacity: 1; }
    .energy-flow-big.active { background: #f7faf3; }
    .energy-flow-big.active .state { color: var(--energy-node-color,#3e704c); font-weight: 700; }
    .energy-flow-big .rcol { display: grid; justify-items: center; gap: 5px; align-self: center; }
    .energy-flow-big .prio { width: 13px; height: 13px; border-radius: 50%; border: 1px solid #8a7b3f; color: #705c22; font-size: 8.5px; font-weight: 800; line-height: 1; display: grid; place-items: center; font-variant-numeric: tabular-nums; margin-top: 0; }
    .energy-flow-big .drag { color: #b5b0a2; cursor: grab; margin-top: 0; }
    .energy-flow-big button.drag { border: 0; background: none; padding: 5px; margin: -5px; display: grid; place-items: center; }
    .energy-flow-big .drag .energy-ui-icon { width: 10px; height: 16px; }
    .energy-flow-big .drag-static { cursor: default; opacity: .55; }
    .energy-flow-big.dragging { opacity: .55; }
    .energy-flow-big.ghost { border: 1.5px dashed #c9c2b2; background: transparent; color: var(--muted); grid-template-columns: minmax(0,1fr); cursor: pointer; }
    .energy-flow-big.ghost .energy-flow-main { cursor: pointer; }
    .energy-flow-big.ghost .plus { width: 21px; height: 21px; border-radius: 50%; border: 1.5px dashed #a89f8a; display: grid; place-items: center; color: #8a7b3f; margin: 0 auto; }
    .energy-flow-big.ghost .plus .energy-ui-icon { width: 11px; height: 11px; }
    .energy-flow-big.ghost b { font-size: 12px; font-weight: 700; color: var(--muted); }
    .energy-flow-hint { position: absolute; left: 26px; bottom: 10px; }
    .energy-peak-chip { display: inline-flex; align-items: baseline; gap: 9px; margin-inline: auto; justify-self: center; font-size: 12.5px; color: var(--muted); white-space: nowrap; text-decoration: none; }
    .energy-peak-chip b { font-family: var(--font-serif); font-weight: 600; font-size: 17px; color: var(--ink); font-variant-numeric: tabular-nums; }
    .energy-peak-chip .sep { color: #cfc7b0; }
    .energy-peak-chip-label { font-size: 11px; font-weight: 800; letter-spacing: .06em; text-transform: uppercase; color: var(--gold-ink); }
    /* The coverage note is the first thing to yield: below 1420px it would
       push the Testlauf control out of the strip. */
    @media (max-width: 1419px) { .energy-peak-chip-coverage, .energy-peak-chip-coverage-sep { display: none; } }
    @media (max-width: 1239px) { .energy-peak-chip { display: none; } }
    /* Below 1180px the fixed flow grid (452px) plus the 258px rail no longer
       share the card: the rail stacks under the flow. */
    @media (max-width: 1179px) {
      .energy-flow-rail { margin-top: 34px; }
      .energy-flow-hint { position: static; margin-top: 12px; }
    }
    @media (max-width: 760px) {
      .energy-flow-area { padding: 28px 14px 30px; }
      .energy-flow-grid2 { grid-template-columns: minmax(0,1fr); grid-template-rows: auto 40px auto 40px auto 40px auto; }
      .energy-flow-slot-top { grid-column: 1; grid-row: 1; width: 100%; flex-wrap: wrap; justify-self: stretch; }
      .energy-flow-slot-left { grid-column: 1; grid-row: 3; display: flex; justify-content: center; }
      .energy-flow-hub2 { grid-column: 1; grid-row: 5; }
      .energy-flow-slot-bottom { grid-column: 1; grid-row: 7; width: 100%; justify-self: stretch; }
      .energy-flow-slot-top .energy-flow-tile, .energy-flow-slot-bottom .energy-flow-tile, .energy-flow-slot-left .energy-flow-tile, .energy-flow-hub2 { width: min(258px,100%); }
      .energy-flow-rail { justify-items: center; }
      .energy-flow-rail > .microlabel { left: max(2px,calc((100% - 258px) / 2)); }
    }
    .energy-cockpit-top .energy-live-head { min-height: 86px; align-items: center; border: 0; padding: 24px 28px 8px; }
    .energy-cockpit-top .energy-live-head > div { display: grid; gap: 6px; }
    .energy-cockpit-top .energy-live-title { display: flex; gap: 9px; align-items: center; }
    .energy-cockpit-top .energy-live-title h2 { font-size: 27px; font-weight: 550; letter-spacing: -.02em; }
    .energy-live-meta { display: flex; gap: 8px; align-items: center; color: var(--muted); font-size: 12px; }
    .energy-live-meta strong { color: #3e704c; font-size: 12px; font-weight: 750; }
    .energy-live-meta > span:last-child { margin-left: 6px; }
    .energy-live-meta .energy-live-updated { margin-left: 4px; color: var(--muted); font-variant-numeric: tabular-nums; }
    .energy-live-meta .energy-live-updated::before { content: "·"; margin-right: 8px; }
    .energy-live-meta.is-stale strong { color: #8a681c; }
    .energy-live-meta.is-stale .energy-live-dot { background: #b8891f; box-shadow: 0 0 0 3px rgba(184,137,31,.12); }
    .energy-cockpit-top .energy-live-dot { width: 9px; height: 9px; border-radius: 50%; background: #5d965c; box-shadow: 0 0 0 3px rgba(93,150,92,.11); }
    .energy-live-state-offline { color: var(--muted); font-size: 11.5px; font-weight: 650; }
    .energy-cockpit-top .energy-live-head > div > span { color: var(--muted); font-size: 12px; }
    .energy-live-footer { min-height: 58px; display: flex; flex-wrap: wrap; justify-content: space-between; gap: 0 16px; align-items: center; border-top: 1px solid var(--line); padding: 0 22px 0 28px; }
    .energy-live-footer .energy-recommendation-trigger { margin-left: auto; border-color: transparent; }
    .energy-live-footer:not(:has(> details[open])) { height: 58px; }
    .energy-cockpit-top .energy-live-more { margin: 0; }
    .energy-cockpit-top .energy-live-more > summary, .energy-info-disclosure > summary { min-height: 44px; display: flex; gap: 8px; align-items: center; border: 0; padding: 0; color: var(--ink); background: transparent; font-size: 12px; cursor: pointer; list-style: none; }
    .energy-cockpit-top .energy-live-more > summary > .energy-ui-icon { width: 16px; height: 16px; color: #705c22; transition: transform .16s ease; }
    .energy-cockpit-top .energy-live-more > summary::after { content: none; }
    .energy-cockpit-top .energy-live-more[open] > summary > .energy-ui-icon { transform: rotate(90deg); }
    .energy-cockpit-top .energy-live-more > summary span { display: flex; gap: 5px; align-items: baseline; }
    .energy-cockpit-top .energy-live-more > summary small { display: none; }
    .energy-cockpit-top .energy-live-more > summary strong { font-size: 12px; font-weight: 650; }
    .energy-info-disclosure { margin-left: auto; }
    .energy-info-disclosure > summary { color: var(--muted); }
    .energy-info-mark { width: 22px; height: 22px; display: grid; place-items: center; color: #7b7b73; }
    .energy-info-mark .energy-ui-icon { width: 20px; height: 20px; }
    .energy-live-footer > details[open] { flex: 1 0 100%; }
    /* The relocated flow explainer opens as a panel card inside the flow area. */
    .energy-flow-hint[open] > div { display: grid; gap: 10px; margin-top: 10px; border: 1px solid var(--line); border-radius: var(--radius-md); background: var(--panel); box-shadow: var(--shadow-md); padding: 16px 18px; max-width: 460px; position: relative; z-index: 3; color: var(--muted); font-size: 12.5px; line-height: 1.45; }
    .energy-flow-hint[open] > div > strong { color: var(--ink); font-size: 14px; }
    .energy-flow-hint[open] > div > a { min-height: 44px; display: inline-flex; align-items: center; justify-self: start; color: #705c22; font-weight: 800; }
    .energy-live-footer > details[open] > div { display: grid; gap: 10px; border-top: 1px solid var(--line); padding: 16px 4px 10px; color: var(--muted); font-size: 12.5px; line-height: 1.45; }
    .energy-live-footer > details[open] > div > strong { color: var(--ink); font-size: 14px; }
    .energy-live-footer > details[open] > div > a { min-height: 44px; display: inline-flex; align-items: center; justify-self: start; color: #705c22; font-weight: 800; }
    .energy-help-readings { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 5px 12px; }
    .energy-help-readings > span { display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 2px 8px; }
    .energy-help-readings small { grid-column: 1 / -1; color: var(--soft); }
    .energy-cockpit-top .energy-live-more-list { grid-template-columns: repeat(2,minmax(0,1fr)); border: 0; padding: 8px 0 0; }
    .energy-cockpit-top .energy-live-more-row { padding: 8px 4px; }

    .energy-metric-info { position: relative; display: inline-grid; flex: 0 0 auto; }
    .energy-metric-info > summary { width: 28px; height: 28px; display: grid; place-items: center; border: 0; border-radius: 50%; color: #777870; background: transparent; cursor: pointer; list-style: none; }
    .energy-metric-info > summary .energy-ui-icon { width: 20px; height: 20px; }
    .energy-metric-info > summary::-webkit-details-marker, .energy-info-disclosure > summary::-webkit-details-marker, .energy-tariff-disclosure > summary::-webkit-details-marker, .energy-next-why > summary::-webkit-details-marker, .energy-next-overflow::-webkit-details-marker { display: none; }
    .energy-metric-info > div { position: absolute; z-index: 30; right: 0; top: calc(100% + 7px); width: min(280px,calc(100vw - 42px)); display: grid; gap: 5px; border: 1px solid var(--line); border-radius: 10px; padding: 13px 14px; color: var(--muted); background: #fffefa; box-shadow: var(--shadow-lg); font-family: var(--font-sans); font-size: 12px; font-weight: 500; line-height: 1.4; text-transform: none; letter-spacing: 0; }
    .energy-metric-info > div strong { color: var(--ink); font-family: var(--font-sans) !important; font-size: 13px !important; font-weight: 800 !important; white-space: normal !important; }
    .energy-metric-info > div b { color: var(--ink); font-weight: 750; }
    .energy-metric-info > div small { color: var(--soft); }

    .energy-tariff { display: flex; flex-direction: column; gap: 12px; overflow: visible; border-color: #ded9ce; border-radius: 14px; padding: 26px 30px 5px; box-shadow: 0 12px 32px rgba(32,40,33,.06); }
    .energy-tariff > .energy-card-head { min-height: 58px; align-items: start; margin: 0; }
    .energy-tariff-heading-copy { min-width: 0; display: grid; gap: 12px; }
    .energy-tariff-title-row { position: relative; display: flex; gap: 7px; align-items: center; }
    .energy-tariff > .energy-card-head p { margin-top: 0; font-size: 13px; }
    .energy-tariff > .energy-card-head .pill { border: 0; padding: 6px 11px; color: #816528; background: #f2ebd9; font-size: 11px; }
    .energy-tariff-overview-info { position: static; }
    .energy-tariff-overview-info > div { left: 0; right: auto; top: calc(100% + 8px); }
    .energy-billed { grid-template-columns: repeat(2,minmax(0,1fr)); gap: 12px; margin-top: 8px; border: 0; border-radius: 0; overflow: visible; background: transparent; }
    .energy-billed > div { min-height: 184px; gap: 9px; border: 1px solid var(--line); border-radius: 11px; padding: 20px 19px; background: #fffefa; }
    .energy-metric-label { display: flex; gap: 5px; align-items: center; justify-content: space-between; }
    .energy-billed .energy-metric-label > span, .energy-tariff-cost .energy-metric-label > span { min-height: 0; color: var(--muted); font-size: 10px; font-weight: 800; letter-spacing: .07em; text-transform: uppercase; }
    .energy-billed > div > strong { font-size: clamp(38px,3vw,44px); white-space: nowrap; }
    .energy-tariff-meter { width: 100%; height: 8px; display: block; overflow: hidden; border: 0; border-radius: 99px; appearance: none; background: #e8e5df; }
    .energy-tariff-meter::-webkit-progress-bar { border-radius: inherit; background: #e8e5df; }
    .energy-tariff-meter::-webkit-progress-value { border-radius: inherit; background: #234332; }
    .energy-tariff-meter::-moz-progress-bar { border-radius: inherit; background: #234332; }
    .energy-tariff-meter.peak::-webkit-progress-value { min-width: 2px; background: #c6a24b; }
    .energy-tariff-meter.peak::-moz-progress-bar { min-width: 2px; background: #c6a24b; }
    .energy-tariff-basis { min-height: 28px; align-items: center; margin: 4px 0 0; overflow: hidden; font-size: 11.5px; white-space: nowrap; }
    .energy-tariff-basis > .energy-ui-icon { width: 16px; height: 16px; flex: 0 0 16px; color: var(--muted); }
    .energy-tariff-basis > span:last-child { min-width: 0; overflow: hidden; text-overflow: ellipsis; }
    .energy-tariff-cost { min-height: 106px; display: block; margin-top: 2px; border: 1px solid var(--line); border-radius: 11px; padding: 11px 18px; background: #fffefa; }
    .energy-tariff-cost-figure { gap: 5px; }
    .energy-tariff-cost-figure > p { display: flex; gap: 7px; align-items: baseline; }
    .energy-tariff-cost-figure > p strong { font-family: var(--font-serif); font-size: 36px; font-weight: 550; line-height: 1; white-space: nowrap; }
    .energy-tariff-cost-figure > p small { color: var(--muted); font-size: 13px; }
    .energy-tariff-disclosure { margin-top: auto; border-top: 1px solid var(--line); }
    .energy-tariff-disclosure > summary { min-height: 48px; display: flex; justify-content: space-between; gap: 12px; align-items: center; color: var(--ink); font-size: 12.5px; font-weight: 700; cursor: pointer; list-style: none; }
    .energy-tariff-disclosure > summary > .energy-ui-icon { width: 18px; height: 18px; transition: transform .16s ease; }
    .energy-tariff-disclosure[open] > summary > .energy-ui-icon { transform: rotate(90deg); }
    .energy-tariff-disclosure > div { min-width: 0; display: grid; grid-template-columns: minmax(0,1fr); gap: 14px; border-top: 1px solid var(--line); padding: 18px 0 8px; }
    .energy-tariff-disclosure > div > * { min-width: 0; }
    .energy-tariff-method, .energy-tariff-detail-basis { color: var(--muted); font-size: 12.5px; line-height: 1.5; }
    .energy-tariff-detail-basis { display: grid; gap: 3px; }
    .energy-tariff-detail-basis strong { color: var(--ink); }
    .energy-tariff-tiers { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 10px; }
    .energy-tariff-tiers > div { display: grid; gap: 3px; border: 1px solid var(--line); border-radius: 9px; padding: 12px; }
    .energy-tariff-tiers span, .energy-tariff-tiers small { color: var(--muted); font-size: 11px; }
    .energy-tariff-tiers strong { font-family: var(--font-serif); font-size: 21px; }
    .energy-tariff-disclosure .energy-tariff-foot { margin-top: 2px; }

    .energy-nextstep-icon { width: 84px; height: 84px; display: grid; place-items: center; border-radius: 50%; color: #436a3e; background: #edf3e7; }
    .energy-nextstep-icon .energy-ui-icon { width: 43px; height: 43px; }
    .energy-next-progress-row { min-width: 0; display: flex; gap: 10px; align-items: center; }
    .energy-next-progress-row > small { color: var(--muted); font-size: 11px; white-space: nowrap; }
    .energy-next-progress { width: min(300px,100%); height: 7px; overflow: hidden; border-radius: 99px; background: #e6e4de; }
    .energy-next-progress > span { height: 100%; display: block; border-radius: inherit; background: #4e824c; }
    .energy-next-status { display: flex; gap: 6px; align-items: center; color: var(--muted); font-size: 11.5px; }
    .energy-next-status > span { width: 18px; height: 18px; display: grid; place-items: center; border-radius: 50%; color: #fff; background: #579057; }
    .energy-next-status > span .energy-ui-icon { width: 11px; height: 11px; }
    .energy-next-why { justify-self: start; }
    .energy-next-why > summary { min-height: 44px; display: inline-flex; align-items: center; color: #6e684f; font-size: 11.5px; font-weight: 750; text-decoration: underline; text-underline-offset: 3px; cursor: pointer; list-style: none; }
    .energy-next-why > div { grid-column: 1 / -1; display: grid; gap: 7px; border-top: 1px solid var(--line); padding-top: 10px; }
    .energy-next-why[open] { grid-column: 1 / -1; width: 100%; }
    .energy-next-why[open] > summary { margin-bottom: 5px; }
    .energy-recommendation-dialog .energy-next-why p { margin: 0; }
    .energy-observation-running { min-height: 52px; display: inline-flex; gap: 9px; align-items: center; justify-content: center; border-radius: 8px; padding: 10px 22px; color: #fff; background: #173526; font-size: 13px; font-weight: 750; white-space: nowrap; }
    .energy-observation-running > span { color: #79a579; font-size: 9px; }
    .energy-next-menu { display: grid; gap: 8px; margin-top: 10px; }
    .energy-dismiss-form { justify-self: end; }

    @media (max-width: 1439px) {
      .energy-cockpit-top { width: 100%; margin-left: 0; }
      .energy-cockpit-top .energy-health { grid-template-columns: minmax(0,1fr); }
      .energy-cockpit-top .energy-live { grid-column: 1; grid-row: 1; }
      .energy-tariff-disclosure { margin-top: 0; }
    }
    @media (min-width: 901px) and (max-width: 1439px) {
    }
    @media (min-width: 901px) and (max-width: 1100px) {
    }
    @media (max-width: 1023px) {
      .energy-cockpit-main .energy-mode-copy span { display: none; }
    }
    @media (max-width: 900px) {
      .energy-cockpit-main .energy-mode-strip { top: var(--mobile-nav-height); min-height: 58px; padding: 6px 12px; }
      .energy-mode-state { min-height: 44px; padding: 6px 10px 6px 8px; }
      .energy-cockpit-main .energy-mode-icon { width: 28px; height: 28px; flex-basis: 28px; }
      .energy-cockpit-main .energy-mode-copy { min-height: 0; padding: 0; }
      .energy-cockpit-main .energy-mode-copy strong { font-size: 12px; }
      .energy-cockpit-main .energy-mode-action { width: auto; max-width: 126px; min-height: 44px; padding: 6px 11px; font-size: 11px; }
      .energy-cockpit-main .energy-mode-control, .energy-cockpit-main .energy-mode-strip > form { grid-column: auto; }
      .energy-cockpit-main .energy-mode-popover { top: calc(var(--mobile-nav-height) + 64px); }
      .energy-cockpit-top .energy-heading { grid-template-columns: minmax(0,1fr) auto; }
      .energy-cockpit-top .energy-heading-action { justify-self: end; }
      .energy-cockpit-top .energy-card { padding: 18px; }
      .energy-cockpit-top .energy-live { padding: 0; }
      .energy-tariff { padding: 20px 18px 10px; }
      .energy-tariff > .energy-card-head { flex-wrap: wrap; gap: 10px; }
      .energy-tariff > .energy-card-head > div { flex: 1 1 200px; order: 1; }
      .energy-tariff > .energy-card-head > .pill { order: 2; }
      .energy-observation-running { flex: 1 1 auto; }
      .energy-metric-info > summary { width: 44px; height: 44px; }
    }
    @media (max-width: 620px) {
      .energy-cockpit-top { gap: 16px; }
      .energy-cockpit-top .energy-live-head > div { min-width: 0; }
      .energy-live-meta { flex-wrap: wrap; }
      .energy-live-meta .energy-live-updated { flex-basis: 100%; margin-left: 17px; white-space: nowrap; }
      .energy-live-meta .energy-live-updated::before { content: none; }
      .energy-recommendation-trigger { width: 44px; min-width: 44px; padding: 0; justify-content: center; }
      .energy-recommendation-trigger > span:last-child { position: absolute; width: 1px; height: 1px; margin: -1px; overflow: hidden; clip: rect(0,0,0,0); white-space: nowrap; }
      .energy-cockpit-top .energy-heading { position: relative; min-height: 0; display: block; padding: 0 56px 0 2px; }
      .energy-cockpit-top .energy-heading h1 { overflow: hidden; font-size: 36px; text-overflow: ellipsis; white-space: nowrap; }
      .energy-cockpit-top .energy-heading-breadcrumb { min-width: 0; flex-wrap: nowrap; overflow: hidden; white-space: nowrap; }
      .energy-cockpit-top .energy-heading-breadcrumb span:last-child { min-width: 0; max-width: none; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
      .energy-cockpit-top .energy-heading-action { position: absolute; right: 0; top: 28px; }
      .energy-live-footer { padding-left: 16px; padding-right: 16px; }
      .energy-info-disclosure > summary > span:first-child { display: none; }
      .energy-info-mark { width: 44px; height: 44px; }
      .energy-info-mark .energy-ui-icon { width: 20px; height: 20px; }
      .energy-metric-info > div { width: min(280px,calc(100vw - 74px)); }
      .energy-help-readings, .energy-cockpit-top .energy-live-more-list { grid-template-columns: 1fr; }
      .energy-billed > div { min-height: 154px; padding: 17px 15px; }
      .energy-billed > div > strong { font-size: clamp(32px,10vw,40px); }
      .energy-tariff-tiers { grid-template-columns: 1fr; }
      .energy-tariff-disclosure :where(p,li,a,span,small) { overflow-wrap: anywhere; }
      .energy-tariff-settings > summary { flex-wrap: wrap; }
      .energy-tariff-settings > summary span { min-width: 0; }
      .energy-recommendation-summary { grid-template-columns: 52px minmax(0,1fr); gap: 13px; }
      .energy-recommendation-summary .energy-nextstep-icon { width: 52px; height: 52px; }
      .energy-recommendation-summary .energy-nextstep-icon .energy-ui-icon { width: 27px; height: 27px; }
      .energy-nextstep h3 { font-size: 20px; }
      .energy-recommendation-dialog .energy-recommendation-actions { flex-wrap: wrap; }
      .energy-observation-running { width: 100%; }
      .energy-recommendation-dialog .energy-recommendation-actions > form { flex: 1 1 auto; }
    }
    @media (max-width: 560px) {
    }
    @media (max-width: 379px) {
      .energy-cockpit-main .energy-mode-strip { gap: 8px; }
      .energy-mode-state { padding-right: 8px; }
      .energy-cockpit-main .energy-mode-action { padding-inline: 9px; }
      .energy-billed { grid-template-columns: 1fr; }
      .energy-billed > div { min-height: 138px; }
    }
    @media (forced-colors: active) {
      .energy-ui-icon { forced-color-adjust: none; background: currentColor; }
    }
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
    .issue-safety-note { display: block; border: 1px solid var(--line); border-radius: var(--radius-xs); padding: 10px 12px; background: rgba(200,153,63,.08); color: var(--muted); font-size: 12.5px; font-weight: 650; line-height: 1.45; }
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
    .issue-next-step { border: 1px solid var(--line); border-radius: var(--radius-xs); padding: 8px 11px; color: var(--ink); background: rgba(200,153,63,.08); font-size: 13.5px; font-weight: 750; line-height: 1.4; }
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
    .attachment-delete { position: absolute; top: -3px; right: -3px; margin: 0; }
    .attachment-delete button { position: relative; width: 44px; height: 44px; display: grid; place-items: center; border: 1px solid transparent; border-radius: var(--radius-xs); background: transparent; color: transparent; font-size: 18px; line-height: 1; cursor: pointer; }
    .attachment-delete button::before { content: "\00d7"; position: absolute; top: 50%; left: 50%; width: 28px; height: 28px; display: grid; place-items: center; border: 1px solid rgba(32,37,31,.18); border-radius: var(--radius-xs); background: rgba(255,254,251,.94); color: var(--ink); transform: translate(-50%,-50%); }
    .attachment-delete button:hover, .attachment-delete button:focus-visible { border-color: transparent; background: transparent; color: transparent; }
    .attachment-delete button:hover::before, .attachment-delete button:focus-visible::before { border-color: #9e2a2b; color: #9e2a2b; }
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
    .energy-consumer-dialog { width: min(720px,calc(100vw - 28px)); }
    .energy-consumer-dialog[open] { height: min(860px,calc(100dvh - 28px)); }
    .energy-consumer-dialog > form { height: 100%; }
    .energy-consumer-dialog .dialog-body { overflow-x: hidden; overflow-y: auto; touch-action: pan-y; -webkit-overflow-scrolling: touch; scrollbar-gutter: stable; }
    .energy-consumer-dialog .dialog-head { align-items: flex-start; }
    .energy-consumer-dialog-heading { display: grid; gap: 3px; }
    .energy-consumer-dialog-heading p { margin: 0; color: var(--muted); font-size: 13px; }
    .energy-consumer-dialog .dialog-close { display: grid; place-items: center; }
    .energy-consumer-dialog .dialog-close .energy-ui-icon { width: 17px; height: 17px; }
    .energy-consumer-primary { display: grid; grid-template-columns: minmax(0,1fr) 150px; gap: 12px; }
    .energy-consumer-display-fields { display: grid; grid-template-columns: 112px minmax(0,1fr); gap: 12px; }
    .energy-consumer-field { display: grid; gap: 6px; color: var(--ink); font-size: 13px; font-weight: 800; }
    .energy-consumer-field[hidden], .energy-consumer-recommendations[hidden] { display: none; }
    .energy-consumer-field input, .energy-consumer-field select { width: 100%; min-height: 44px; box-sizing: border-box; border: 1px solid var(--line); border-radius: var(--radius-xs); padding: 9px 11px; color: var(--ink); background: #fff; font: 600 14px/1.3 var(--font-sans); }
    .energy-consumer-field input[type="color"] { padding: 5px; cursor: pointer; }
    .energy-icon-picker { display: grid; gap: 10px; border-top: 1px solid var(--line); padding-top: 16px; }
    .energy-icon-picker-head { display: flex; align-items: end; justify-content: space-between; gap: 14px; }
    .energy-icon-picker-head > div { display: grid; gap: 2px; }
    .energy-icon-picker-head strong { font-size: 14px; }
    .energy-icon-picker-head small { color: var(--muted); }
    .energy-icon-search { position: relative; width: min(235px,44%); }
    .energy-icon-search .energy-ui-icon { position: absolute; left: 11px; top: 50%; width: 15px; height: 15px; color: var(--muted); transform: translateY(-50%); pointer-events: none; }
    .energy-icon-search input { width: 100%; min-height: 40px; box-sizing: border-box; border: 1px solid var(--line); border-radius: var(--radius-xs); padding: 8px 10px 8px 34px; background: #fff; font: 600 13px/1.2 var(--font-sans); }
    .energy-icon-options { display: grid; grid-template-columns: repeat(5,minmax(0,1fr)); gap: 8px; }
    .energy-icon-search-results[hidden], .energy-icon-presets[hidden] { display: none; }
    .energy-icon-result-note { margin: 0; color: var(--muted); font-size: 12px; }
    .energy-icon-choice { position: relative; min-width: 0; min-height: 70px; display: grid; place-items: center; align-content: center; gap: 6px; border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 8px 5px; color: var(--muted); background: #fff; cursor: pointer; }
    .energy-icon-choice[hidden] { display: none; }
    .energy-icon-choice:hover { border-color: #aa9d82; color: var(--ink); }
    .energy-icon-choice:has(input:checked) { border-color: rgba(62,112,76,.65); color: #3e704c; background: #f4f8f1; box-shadow: inset 0 0 0 1px rgba(62,112,76,.18); }
    .energy-icon-choice:has(input:focus-visible) { outline: 2px solid var(--focus); outline-offset: 2px; }
    .energy-icon-choice input { position: absolute; opacity: 0; pointer-events: none; }
    .energy-icon-choice .energy-ui-icon { width: 24px; height: 24px; }
    .energy-icon-choice span:last-child { max-width: 100%; overflow: hidden; font-size: 10.5px; font-weight: 750; text-overflow: ellipsis; white-space: nowrap; }
    .energy-consumer-recommendations { border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel-soft); overflow: hidden; }
    .energy-consumer-recommendations > summary { cursor: pointer; list-style: none; padding: 12px 13px; font-size: 13px; font-weight: 800; }
    .energy-consumer-recommendations > summary::-webkit-details-marker { display: none; }
    .energy-consumer-recommendations > summary::after { content: "›"; float: right; color: var(--gold-ink); font-size: 20px; line-height: .8; }
    .energy-consumer-recommendations[open] > summary { border-bottom: 1px solid var(--line); }
    .energy-consumer-recommendations[open] > summary::after { transform: rotate(90deg); }
    .energy-consumer-recommendation-fields { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 12px; padding: 12px 13px 14px; }
    .energy-consumer-measurements { display: grid; gap: 10px; border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 13px; background: #fff; }
    .energy-consumer-measurements-head { display: flex; align-items: start; justify-content: space-between; gap: 12px; }
    .energy-consumer-measurements-head > div { display: grid; gap: 3px; }
    .energy-consumer-measurements-head strong { font-size: 14px; }
    .energy-consumer-measurements-head small, .energy-consumer-measurement-status { color: var(--muted); font-size: 12px; }
    .energy-consumer-measurement-status { margin: 0; }
    .energy-consumer-measurement-fields { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 12px; }
    .energy-consumer-measurement-fields option:disabled { color: #999; }
    .energy-consumer-delete { margin-right: auto; }
    .energy-consumer-delete[hidden] { display: none; }
    .energy-consumer-delete-confirm { display: flex; align-items: center; gap: 8px; margin-right: auto; color: #812c2d; font-size: 12px; font-weight: 800; }
    .energy-consumer-delete-confirm[hidden] { display: none; }
    @media (max-width: 600px) {
      .dialog-head { padding: 14px; }
      .dialog-close { width: 44px; height: 44px; flex: 0 0 auto; }
      .dialog-body { padding: 16px 14px 18px; }
      .dialog-footer { padding: 12px 14px max(12px,env(safe-area-inset-bottom)); }
      .dialog-footer .button { width: 100%; min-height: 44px; }
      .energy-consumer-primary, .energy-consumer-display-fields, .energy-consumer-recommendation-fields, .energy-consumer-measurement-fields { grid-template-columns: 1fr; }
      .energy-icon-picker-head { align-items: stretch; flex-direction: column; }
      .energy-icon-search { width: 100%; }
      .energy-icon-options { grid-template-columns: repeat(3,minmax(0,1fr)); }
      .energy-consumer-dialog .dialog-footer { flex-wrap: wrap; }
      .energy-consumer-dialog .dialog-footer .energy-consumer-delete { width: auto; margin-right: auto; }
      .energy-consumer-delete-confirm { width: 100%; flex-wrap: wrap; }
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
    .documents-screen .document-admin-tools-body .button,
    .documents-screen .version-row .button { min-height: 44px; }
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
    .handover-note { border: 1px solid var(--line); border-radius: var(--radius-xs); padding: 9px 12px; background: rgba(200,153,63,.08); color: var(--muted); }
    .handover-note strong { color: var(--ink); }
    .handover-note p { margin-top: 4px; line-height: 1.45; overflow-wrap: anywhere; }
    .handover-scope { border-left: 3px solid rgba(47,107,74,.35); padding: 9px 12px; background: rgba(47,107,74,.06); color: var(--muted); font-size: 13px; line-height: 1.45; }
    .handover-foot { display: flex; flex-wrap: wrap; justify-content: space-between; gap: 10px; padding-top: 4px; color: var(--soft); font-size: 12px; }
    .handover-dialog textarea { min-height: 94px; }
    .handover-dialog #handover-rooms { min-height: 82px; }
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
    @media (max-width: 1120px) { .handover-page-head { grid-template-columns: minmax(0,1fr); } }
    .handover-blank { display: grid; grid-template-columns: minmax(0,1.25fr) minmax(286px,.75fr); gap: 16px; }
    .handover-blank-main { display: grid; align-content: center; gap: 18px; border: 1px dashed rgba(200,153,63,.45); border-radius: var(--radius-sm); background: rgba(255,254,251,.7); padding: clamp(22px,3.2vw,36px); }
    .handover-blank-lead { display: grid; justify-items: start; gap: 13px; }
    .handover-blank-icon { width: 52px; height: 52px; display: grid; place-items: center; border: 1px solid rgba(200,153,63,.3); border-radius: 12px; background: rgba(200,153,63,.1); color: var(--gold-ink); }
    .handover-blank-icon svg { width: 26px; height: 26px; fill: none; stroke: currentColor; stroke-width: 1.5; stroke-linecap: round; stroke-linejoin: round; }
    .handover-blank-main h2 { font-size: clamp(25px,3vw,31px); }
    .handover-blank-main p { max-width: 50ch; color: var(--muted); font-size: 15px; line-height: 1.5; }
    .handover-blank-actions { display: flex; flex-wrap: wrap; gap: 9px; margin-top: 3px; }
    .handover-blank-actions .button { min-height: 44px; }
    .handover-blank-side { align-self: start; display: grid; align-content: start; gap: 13px; border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel); padding: 20px; box-shadow: var(--shadow-panel); }
    .handover-blank-side h2 { font-size: 12px; font-weight: 800; letter-spacing: .12em; text-transform: uppercase; color: var(--gold-ink); font-family: var(--font-sans); }
    .handover-blank-steps { display: grid; gap: 11px; margin: 0; padding: 0; list-style: none; counter-reset: handover-step; }
    .handover-blank-steps li { display: grid; grid-template-columns: 24px minmax(0,1fr); gap: 2px 11px; align-items: start; counter-increment: handover-step; }
    .handover-blank-steps li::before { content: counter(handover-step); grid-row: 1 / span 2; width: 24px; height: 24px; display: grid; place-items: center; border-radius: 50%; background: var(--ink); color: #fff; font-size: 12px; font-weight: 850; }
    .handover-blank-steps strong { grid-column: 2; font-size: 13.5px; }
    .handover-blank-steps span { grid-column: 2; color: var(--muted); font-size: 12.5px; line-height: 1.4; }
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
    .handover-next .button { min-height: 44px; }
    .handover-details { border-top: 1px solid var(--line); padding-top: 10px; }
    .handover-details > summary { min-height: 44px; display: flex; align-items: center; gap: 8px; cursor: pointer; color: var(--gold-ink); font-size: 13px; font-weight: 850; }
    .handover-details > summary small { color: var(--muted); font-size: 11.5px; font-weight: 650; }
    .handover-details[open] > summary { margin-bottom: 13px; }
    .handover-parties { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 8px; margin-bottom: 10px; }
    .handover-parties div { border: 1px solid var(--line); border-radius: var(--radius-xs); padding: 10px 12px; background: var(--panel-soft); min-width: 0; }
    .handover-parties span { display: block; color: var(--soft); font-size: 10.5px; font-weight: 850; text-transform: uppercase; letter-spacing: .06em; }
    .handover-parties strong { display: block; margin-top: 3px; font-size: 12.5px; overflow-wrap: anywhere; }
    .handover-detail h4 { font-size: 15px; }
    .handover-add-files { display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 10px; align-items: end; border-top: 1px solid var(--line); margin-top: 12px; padding-top: 12px; }
    .handover-add-files label { min-width: 0; }
    .handover-detail-actions { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 10px; margin-top: 12px; color: var(--soft); font-size: 11.5px; }
    .handover-add-files .button, .handover-detail-actions .button { min-height: 44px; justify-content: center; }
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
    .handover-public-files { display: grid; gap: 7px; padding: 0 14px 14px; }
    .handover-public-file { min-height: 58px; display: grid; grid-template-columns: 48px minmax(0,1fr) auto; gap: 10px; align-items: center; border: 1px solid var(--line); border-radius: var(--radius-xs); padding: 7px; background: var(--panel); color: var(--ink); text-decoration: none; }
    .handover-public-file:hover, .handover-public-file:focus-visible { border-color: var(--gold); }
    .handover-public-file img, .handover-public-file-icon { width: 48px; height: 42px; display: grid; place-items: center; border-radius: 6px; object-fit: cover; background: rgba(200,153,63,.14); color: var(--gold-ink); font-size: 10px; font-weight: 900; }
    .handover-public-file-copy { min-width: 0; display: grid; gap: 2px; }
    .handover-public-file-copy strong { overflow-wrap: anywhere; }
    .handover-public-file-copy small { color: var(--muted); font-size: 11.5px; }
    .handover-public-file-arrow { color: var(--gold-ink); font-size: 20px; }
    .handover-confirm-note { border: 1px solid var(--line); border-radius: var(--radius-xs); padding: 11px 12px; background: var(--panel-soft); }
    .handover-confirm-note > summary { min-height: 44px; display: flex; align-items: center; margin: -11px -12px; padding: 11px 12px; cursor: pointer; color: var(--gold-ink); font-size: 12px; font-weight: 850; }
    .handover-confirm-note label { margin-top: 11px; }
    .handover-confirm-consent { grid-template-columns: 24px minmax(0,1fr); align-items: start; border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 13px; background: #fffefb; color: var(--ink); font-size: 13px; line-height: 1.45; letter-spacing: 0; text-transform: none; cursor: pointer; }
    .handover-confirm-consent input { width: 22px; height: 22px; min-height: 0; margin: 0; accent-color: var(--leaf); }
    .handover-confirm-submit { display: grid; gap: 7px; }
    .handover-confirm-submit .button { width: 100%; min-height: 50px; }
    .handover-confirm-submit small { color: var(--muted); font-size: 11.5px; text-align: center; }
    .handover-confirmed { min-height: 76px; display: grid; grid-template-columns: 38px minmax(0,1fr); gap: 12px; align-items: center; border: 1px solid rgba(47,107,74,.24); border-radius: var(--radius-sm); padding: 14px; background: rgba(47,107,74,.08); }
    .handover-confirmed-icon { width: 38px; height: 38px; display: grid; place-items: center; border-radius: 50%; background: var(--leaf); color: #fff; font-size: 18px; font-weight: 900; }
    .handover-confirmed strong { font-size: 15px; }
    .handover-confirmed p { margin-top: 2px; color: var(--muted); font-size: 12.5px; }
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
      .side-map-card { height: 176px !important; }
      .side-map-pin { width: 35px; height: 45px; }
      .side-map-pin-mark { top: 7px; width: 20px; height: 18px; }
      .side-map-pin-mark svg { width: 20px; height: 17px; }
      .side-address-label small, .side-ruler, .side-portal { display: none; }
      .side-map-top { padding-top: 9px; }
      .portal-context-switch > summary { min-height: 50px; }
    }
    @media (min-width: 901px) and (max-height: 1040px) {
      .sidebar { gap: 10px; padding-top: 14px; padding-bottom: 12px; }
      .side-brand { margin-top: -14px; padding-bottom: 3px; }
      .side-map-card { height: 236px; }
      .side-place-copy { gap: 5px; padding-bottom: 8px; }
      .side-address-label strong { font-size: 17px; }
      .side-address-label small { font-size: 9.5px; }
      .side-ruler { height: 8px; background-size: 1px 4px,1px 7px,1px 4px; }
      .side-portal strong { font-size: 14.5px; }
      .side-portal span { font-size: 10px; }
      .side-nav { gap: 2px; scrollbar-gutter: stable; }
      .nav-item { min-height: 36px; padding-top: 6px; padding-bottom: 6px; font-size: 13.5px; }
      .nav-icon { width: 19px; height: 19px; }
      .nav-icon svg { width: 18px; height: 18px; }
      .side-foot { gap: 7px; padding-top: 9px; }
      .side-user { grid-template-columns: 32px minmax(0,1fr) auto; gap: 8px; }
      .side-map-attribution { margin-left: 40px; }
      .avatar { width: 32px; height: 32px; font-size: 11px; }
      .side-user strong { font-size: 12.5px; }
      .side-user span { font-size: 11.5px; }
      .version-button { font-size: 10px; }
      .logout-button { min-height: 44px; }
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
	      .side-brand { display: block; margin: 0; padding: 0; min-width: 0; }
	      .side-map-card { height: auto; display: grid; grid-template-columns: 52px minmax(0,1fr); gap: 7px; align-items: center; }
	      .side-map { position: relative; inset: auto; width: 52px; height: 48px; border-radius: var(--radius-xs); }
	      .side-map-pin { width: 24px; height: 31px; }
	      .side-map-pin-mark { top: 5px; width: 14px; height: 13px; }
	      .side-map-pin-mark svg { width: 14px; height: 12px; stroke-width: 2.5; }
	      .side-map-top { position: static; min-width: 0; padding: 0; background: transparent; }
	      .side-place-copy { position: static; min-width: 0; min-height: 44px; align-content: center; gap: 3px; padding: 0; }
	      .side-place-copy.has-context-switch { display: none; }
	      .side-address-label { max-width: 100%; }
	      .side-address-label strong { font-size: 13px; }
	      .side-address-label small { font-size: 9px; }
	      .side-ruler { display: none; }
	      .side-portal { display: none; }
	      .portal-context-switch > summary { min-height: 44px; grid-template-columns: minmax(0,1fr) 14px; gap: 5px; border: 0; padding: 2px 3px; background: transparent; backdrop-filter: none; }
	      .portal-context-icon { display: none; }
	      .portal-context-current small { font-size: 8px; }
	      .portal-context-current strong { font-size: 11px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	      .portal-context-menu { position: fixed; left: 14px; right: 14px; top: 70px; max-height: min(420px,70vh); }
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
	      .nav-icon { width: 18px; height: 18px; }
	      .nav-icon svg { width: 17px; height: 17px; }
	      .nav-badge { min-width: 20px; height: 18px; padding: 0 6px; font-size: 10px; }
	      .nav-group-label { grid-column: 1 / -1; margin: 7px 4px 0; }
	      .side-foot { margin-top: 2px; gap: 8px; padding: 10px 0 0; }
	      .side-user { grid-template-columns: 34px minmax(0,1fr) auto; gap: 9px; min-width: 0; }
	      .side-map-attribution { margin-left: 43px; }
	      .avatar { width: 34px; height: 34px; font-size: 12px; }
	      .side-user strong { font-size: 13px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
	      .side-user span { font-size: 12px; }
	      .side-version { min-width: 44px; min-height: 44px; display: grid; place-items: center; }
	      .side-map-attribution { min-height: 44px; display: inline-flex; align-items: center; }
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
      .side-address-label { font-size: 9px; }
      .content-top .crumb { max-width: 100%; gap: 6px; flex-wrap: wrap; font-size: 13px; }
      .mobile-menu-prefix { display: none; }
      .settings-hub .settings-section-head { grid-template-columns: minmax(0,1fr); align-items: start; }
      .settings-hub .settings-section-head .settings-tag { justify-self: start; }
      .parking-month-essential > div { padding-inline: 6px; }
      .parking-month-essential dt { font-size: 9px; letter-spacing: 0; }
    }
    @media (max-width: 900px) {
      .side-map-fallback { padding: 0; font-size: 0; }
    }
  </style>
{{end}}

{{define "sidebar"}}
  <aside class="sidebar" aria-label="Portalnavigation">
    <div class="side-brand">
	    <div class="side-map-card">
	      <a class="side-map side-address" href="{{.MapURL}}" target="_blank" rel="noopener noreferrer" aria-label="{{.Tenant.Address}} in OpenStreetMap öffnen" title="In OpenStreetMap öffnen">
	        {{if .SidebarMap.Configured}}
	        <div class="side-map-tiles" aria-hidden="true">
	          {{range .SidebarMap.Tiles}}<img class="side-map-tile" src="{{.URL}}" style="{{.Style}}" width="256" height="256" alt="" draggable="false">{{end}}
	        </div>
	        <span class="side-map-pin" aria-hidden="true">
	          <svg class="side-map-pin-shape" viewBox="0 0 44 56" focusable="false"><path d="M22 55C18.7 49.2 4.5 36.7 4.5 22.2A17.5 17.5 0 1 1 39.5 22.2C39.5 36.7 25.3 49.2 22 55Z"/></svg>
	          <span class="side-map-pin-mark">{{template "tenantBrandMark" .}}</span>
	        </span>
	        {{else}}<span class="side-map-fallback">Standort nicht hinterlegt</span>{{end}}
	      </a>
	      {{if .CanSwitchPortalContext}}
	      <div class="side-map-top">
	        <details class="portal-context-switch">
	          <summary aria-label="Liegenschaft oder Ansicht wechseln">
	            <span class="portal-context-icon" aria-hidden="true"><svg viewBox="0 0 24 24"><path d="M3 10.5 12 4l9 6.5"/><path d="M5.5 9.5V20h13V9.5"/><path d="M9 20v-6h6v6"/></svg></span>
	            <span class="portal-context-current"><small>Portal wechseln · {{.Role}}</small><strong>{{.HouseName}}</strong></span>
	            <span class="portal-context-chevron" aria-hidden="true"></span>
	          </summary>
	          <div class="portal-context-menu" role="list" aria-label="Freigegebene Portal-Kontexte">
	            {{range .PortalContexts}}
	              {{if .Current}}
	              <div class="portal-context-option-current" role="listitem" aria-current="true"><span class="portal-context-option-copy"><strong>{{.HouseName}}</strong><small>{{.Address}}</small></span><span><span class="portal-context-role">{{.Role}}</span><span class="portal-context-active">Aktiv</span></span></div>
	              {{else}}
	              <form class="portal-context-option" method="post" action="/app/context" role="listitem"><input type="hidden" name="tenant" value="{{.TenantSlug}}"><input type="hidden" name="role" value="{{.Role}}"><button type="submit"><span class="portal-context-option-copy"><strong>{{.HouseName}}</strong><small>{{.Address}}</small></span><span class="portal-context-role">{{.Role}}</span></button></form>
	              {{end}}
	            {{end}}
	          </div>
	        </details>
	      </div>
	      {{end}}
	      <div class="side-place-copy{{if .CanSwitchPortalContext}} has-context-switch{{end}}">
	        {{if not .CanSwitchPortalContext}}<a class="side-address-label" href="/app" aria-label="Hausportal {{.HouseName}} öffnen"><strong>{{.HouseName}}</strong><small>{{.SidebarAddress.Full}}</small></a>{{end}}
	        <span class="side-ruler" aria-hidden="true"></span>
	        <a class="side-portal" href="/app"><strong>Hausportal</strong><span>· hausv.org</span></a>
	      </div>
	    </div>
	    </div>
	    <button class="mobile-menu-toggle" type="button" data-mobile-menu-toggle aria-controls="portal-navigation portal-account" aria-expanded="false" aria-label="Navigation öffnen"><span class="mobile-menu-prefix">Menü</span>{{if eq .ActivePage "energy"}}<span class="mobile-home-identity" data-home-identity="mobile-menu" aria-label="{{.HomeIdentity.AriaLabel}}"><strong data-home-display-name>{{.HomeIdentity.DisplayName}}</strong>{{if .HomeIdentity.HasUnit}}<small data-home-unit-label>{{.HomeIdentity.UnitLabel}}</small>{{end}}</span>{{else}}<span>{{if eq .ActivePage "home"}}Überblick{{else if eq .ActivePage "announcements"}}Aushang{{else if eq .ActivePage "events"}}Termine{{else if eq .ActivePage "contacts"}}Kontakte{{else if eq .ActivePage "parking"}}Parkplatz{{else if eq .ActivePage "documents"}}Dokumente{{else if eq .ActivePage "handovers"}}Übergaben{{else if eq .ActivePage "issues"}}Anliegen{{else if eq .ActivePage "abstimmungen"}}Abstimmung{{else if eq .ActivePage "users"}}Benutzer{{else if eq .ActivePage "audit"}}Audit{{else if eq .ActivePage "help"}}Hilfe{{else}}Einstellungen{{end}}</span>{{end}}</button>
	    <nav id="portal-navigation" class="side-nav">
	      {{if .CanUseResidentAreas}}
	      <a class="nav-item {{if eq .ActivePage "home"}}active{{end}}" href="/app"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg></span><span class="nav-label">Hausüberblick</span></a>
	      {{if .CanViewEnergy}}<a class="nav-item {{if eq .ActivePage "energy"}}active{{end}}" href="/app/energie" data-home-identity="nav" aria-label="{{.HomeIdentity.AriaLabel}}"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M4 13h5l2-7 3 12 2-5h4"/><path d="M5 20h14"/></svg></span><span class="nav-label"><span class="nav-home-identity"><strong data-home-display-name>{{.HomeIdentity.DisplayName}}</strong>{{if .HomeIdentity.HasUnit}}<small data-home-unit-label>{{.HomeIdentity.UnitLabel}}</small>{{end}}</span></span></a>{{end}}
	      {{if .PortalModules.Announcements}}<a class="nav-item {{if eq .ActivePage "announcements"}}active{{end}}" href="/app/announcements"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M4 5h16v13H7l-3 3z"/><path d="M8 9h8M8 13h6"/></svg></span><span class="nav-label">Aushang</span>{{if .HasUnreadAnnouncements}}<span class="nav-badge">{{.UnreadAnnouncements}}</span>{{end}}</a>{{end}}
	      {{if .PortalModules.Events}}<a class="nav-item {{if eq .ActivePage "events"}}active{{end}}" href="/app/events"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M7 3v4M17 3v4"/><path d="M4.5 6h15v14h-15z"/><path d="M4.5 10h15"/><path d="M8 14h.01M12 14h.01M16 14h.01"/></svg></span><span class="nav-label">Termine</span></a>{{end}}
	      {{if .PortalModules.Contacts}}<a class="nav-item {{if eq .ActivePage "contacts"}}active{{end}}" href="/app/kontakte"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M16 4h2.5A1.5 1.5 0 0 1 20 5.5v13A1.5 1.5 0 0 1 18.5 20h-13A1.5 1.5 0 0 1 4 18.5v-13A1.5 1.5 0 0 1 5.5 4H8"/><path d="M8.5 3.5h7v4h-7z"/><path d="M9 13a3 3 0 1 0 6 0"/><path d="M7.5 18a4.5 4.5 0 0 1 9 0"/></svg></span><span class="nav-label">Kontakte</span></a>{{end}}
	      {{if .PortalModules.Documents}}<a class="nav-item {{if eq .ActivePage "documents"}}active{{end}}" href="/app/dokumente"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M7 3h7l3 3v15H7z"/><path d="M14 3v4h4"/><path d="M9 13h6M9 17h6"/></svg></span><span class="nav-label">Dokumente</span></a>{{end}}
	      {{end}}
	      {{if .PortalModules.Issues}}<a class="nav-item {{if eq .ActivePage "issues"}}active{{end}}" href="{{if .CanManageIssues}}/app/anliegen/board{{else}}/app/anliegen{{end}}"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M5 18.5V6.5A2.5 2.5 0 0 1 7.5 4h9A2.5 2.5 0 0 1 19 6.5v6A2.5 2.5 0 0 1 16.5 15H10l-5 3.5z"/></svg></span><span class="nav-label">Anliegen</span>{{if .HasOpenIssues}}<span class="nav-badge">{{.OpenIssues}}</span>{{end}}</a>{{end}}
	      {{if .CanUseResidentAreas}}
	      {{if .PortalModules.Votes}}<a class="nav-item {{if eq .ActivePage "abstimmungen"}}active{{end}}" href="/app/abstimmungen"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M5 19V9M12 19V5M19 19v-7"/><path d="M3.5 19h17"/></svg></span><span class="nav-label">Abstimmungen</span></a>{{end}}
	      {{if or (and .PortalModules.Parking .CanSeeParking) (and .PortalModules.Handovers .CanManageHandovers) (and .PortalModules.Users .CanManageUsers)}}<span class="nav-group-label">{{if or (and .PortalModules.Handovers .CanManageHandovers) (and .PortalModules.Users .CanManageUsers)}}Verwaltung{{else}}Weitere Bereiche{{end}}</span>{{end}}
	      {{if and .PortalModules.Parking .CanSeeParking}}<a class="nav-item {{if eq .ActivePage "parking"}}active{{end}}" href="/app/parking"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M5 16h14"/><path d="m7 16 1.5-5h7L17 16"/><path d="M7 16v3M17 16v3"/><path d="M7 19h1M16 19h1"/></svg></span><span class="nav-label">Parkplatznutzung</span></a>{{end}}
	      {{if and .PortalModules.Handovers .CanManageHandovers}}<a class="nav-item {{if eq .ActivePage "handovers"}}active{{end}}" href="/app/uebergaben"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M7 4h10v16H7z"/><path d="M9.5 8h5M9.5 12h4"/><path d="m9.5 16 1.5 1.5 3.5-4"/></svg></span><span class="nav-label">Übergaben</span></a>{{end}}
	      {{if and .PortalModules.Users .CanManageUsers}}<a class="nav-item {{if eq .ActivePage "users"}}active{{end}}" href="/app/settings/users"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M8.5 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6z"/><path d="M3.5 20a5 5 0 0 1 10 0"/><path d="M16 11.5a2.5 2.5 0 1 0 0-5"/><path d="M17 15a4 4 0 0 1 3.5 4"/></svg></span><span class="nav-label">Benutzer &amp; Rechte</span></a>{{end}}
	      {{if and .PortalModules.Audit .CanViewAudit}}<a class="nav-item {{if eq .ActivePage "audit"}}active{{end}}" href="/app/audit"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M5 4h14v16H5z"/><path d="M8 8h8M8 12h8M8 16h5"/></svg></span><span class="nav-label">Verlauf</span></a>{{end}}
	      <a class="nav-item {{if eq .ActivePage "settings"}}active{{end}}" href="/app/settings"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M12 8.5a3.5 3.5 0 1 0 0 7 3.5 3.5 0 0 0 0-7z"/><path d="M19 12a7 7 0 0 0-.1-1l2-1.5-2-3.5-2.4 1a7 7 0 0 0-1.8-1L14.4 3h-4.8L9.3 6a7 7 0 0 0-1.8 1l-2.4-1-2 3.5 2 1.5A7 7 0 0 0 5 12a7 7 0 0 0 .1 1l-2 1.5 2 3.5 2.4-1a7 7 0 0 0 1.8 1l.3 3h4.8l.3-3a7 7 0 0 0 1.8-1l2.4 1 2-3.5-2-1.5a7 7 0 0 0 .1-1z"/></svg></span><span class="nav-label">Einstellungen</span></a>
	      {{end}}
	      {{if .PortalModules.Help}}<a class="nav-item {{if eq .ActivePage "help"}}active{{end}}" href="/app/hilfe"><span class="nav-icon"><span class="energy-ui-icon energy-ui-icon-circle-help" aria-hidden="true"></span></span><span class="nav-label">Hilfe</span></a>{{end}}
    </nav>
    <div id="portal-account" class="side-foot">
      <div class="side-user">
        <span class="avatar">{{.Initials}}</span>
        <div class="side-user-copy"><strong>{{.DisplayName}}</strong><span class="side-user-meta"><span>{{.Role}}</span><button class="side-version version-button" type="button" data-dialog="release-history" aria-haspopup="dialog" aria-controls="release-history" aria-label="Version {{.DisplayVersion}} – Versionsverlauf öffnen">v{{.DisplayVersion}}</button></span></div>
        <form class="logout-form" method="post" action="/auth/logout"><button class="logout-button" type="submit" aria-label="Abmelden" title="Abmelden"><span class="energy-ui-icon energy-ui-icon-log-out" aria-hidden="true"></span><span class="sr-only">Abmelden</span></button></form>
      </div>
      <a class="side-map-attribution" href="https://www.openstreetmap.org/copyright" target="_blank" rel="noopener noreferrer">Kartendaten © OpenStreetMap</a>
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
      {{if and .Msg (not .Confirmation.HasConfirmed)}}<div class="flash {{if .MsgOK}}ok{{end}}">{{.Msg}}</div>{{end}}
      <section class="handover-review" aria-labelledby="handover-review-title">
        <div class="handover-review-head"><div><span>{{if .Confirmation.HasConfirmed}}Bestätigung abgeschlossen{{else}}Vor der Bestätigung{{end}}</span><h2 id="handover-review-title">Protokoll prüfen</h2></div><span class="pill {{.Confirmation.StatusClass}}">{{.Confirmation.Status}}</span></div>
        {{if .Handover.HasRooms}}<details open><summary><span><strong>Räume</strong><small>{{len .Handover.Rooms}} Einträge</small></span></summary><ul>{{range .Handover.Rooms}}<li><strong>{{.Name}}</strong><span>{{if .Condition}}{{.Condition}}{{else}}–{{end}}{{if .Defects}} · {{.Defects}}{{end}}</span></li>{{end}}</ul></details>{{end}}
        {{if .Handover.HasMeters}}<details open><summary><span><strong>Zählerstände</strong><small>{{len .Handover.Meters}} Einträge</small></span></summary><ul>{{range .Handover.Meters}}<li><strong>{{.Label}}</strong><span>{{.Value}}{{if .Unit}} {{.Unit}}{{end}}</span></li>{{end}}</ul></details>{{end}}
        {{if .Handover.HasKeys}}<details open><summary><span><strong>Schlüssel</strong><small>{{len .Handover.Keys}} Positionen</small></span></summary><ul>{{range .Handover.Keys}}<li><strong>{{.Label}}</strong><span>{{.Count}} Stk.</span></li>{{end}}</ul></details>{{end}}
        {{if .Handover.HasNotes}}<details open><summary><span><strong>Notiz</strong><small>1 Eintrag</small></span></summary><p>{{.Handover.Notes}}</p></details>{{end}}
        {{if .Handover.HasAttachments}}<details open><summary><span><strong>Fotos &amp; Dateien</strong><small>{{len .Handover.Attachments}} {{if eq (len .Handover.Attachments) 1}}Datei{{else}}Dateien{{end}}</small></span></summary><div class="handover-public-files">{{range .Handover.Attachments}}<a class="handover-public-file" href="{{.URL}}" target="_blank" rel="noopener">{{if .IsImage}}<img src="{{.ThumbURL}}" alt="">{{else}}<span class="handover-public-file-icon">{{if .IsPDF}}PDF{{else}}DATEI{{end}}</span>{{end}}<span class="handover-public-file-copy"><strong>{{.Filename}}</strong><small>{{.Size}} · öffnen</small></span><span class="handover-public-file-arrow" aria-hidden="true">↗</span></a>{{end}}</div></details>{{end}}
      </section>
      <p class="handover-scope">Dieses Protokoll dokumentiert den Zustand bei der Übergabe. Es ist keine Kautions-, Schaden- oder sonstige Abrechnung.</p>
      {{if .Confirmation.HasConfirmed}}
        <section class="handover-confirmed" aria-label="Bestätigung abgeschlossen"><span class="handover-confirmed-icon" aria-hidden="true">✓</span><div><strong>Protokoll bestätigt</strong><p>{{if .Confirmation.HasConfirmed}}Am {{.Confirmation.ConfirmedAt}} · {{end}}nichts weiter zu tun.</p></div></section>
      {{else}}
        <form method="post" action="/handover/{{.Token}}" class="handover-confirm-form">
          <label>Name für die Bestätigung<input name="name" value="{{.Confirmation.Name}}" autocomplete="name"></label>
          <details class="handover-confirm-note"><summary>Notiz ergänzen (optional)</summary><label>Notiz<textarea name="note" placeholder="Falls etwas ergänzt werden soll"></textarea></label></details>
          <label class="handover-confirm-consent"><input type="checkbox" name="confirm" value="yes" required><span>Die oben angezeigten Angaben entsprechen dem gemeinsam festgehaltenen Stand.</span></label>
          <div class="handover-confirm-submit"><button class="button primary" type="submit">Protokoll bestätigen</button><small>Zeitpunkt und Rolle werden protokolliert.</small></div>
        </form>
      {{end}}
    </section>
  </main>
</body>
</html>
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

{{define "portalModuleSettings"}}
{{template "appOpen" .}}
    <style>
      .module-settings-page { width: min(980px,100%); gap: 18px; }
      .module-settings-head { display: grid; gap: 6px; max-width: 760px; }
      .module-core { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 12px; }
      .module-core-card { min-height: 90px; display: grid; grid-template-columns: 42px minmax(0,1fr) auto; gap: 12px; align-items: center; padding: 16px; }
      .module-core-icon, .module-option-icon { width: 42px; height: 42px; display: grid; place-items: center; border-radius: 12px; background: rgba(47,107,74,.1); color: var(--leaf); }
      .module-core-icon svg, .module-option-icon svg { width: 22px; height: 22px; fill: none; stroke: currentColor; stroke-width: 1.8; }
      .module-core-copy strong, .module-option-copy strong { display: block; font-family: var(--font-serif); font-size: 18px; }
      .module-core-copy span, .module-option-copy span { display: block; margin-top: 3px; color: var(--muted); font-size: 12.5px; line-height: 1.4; }
      .module-fixed { border-radius: 999px; padding: 5px 9px; background: rgba(47,107,74,.1); color: var(--leaf); font-size: 10px; font-weight: 850; letter-spacing: .06em; text-transform: uppercase; }
      .module-form { display: grid; gap: 14px; }
      .module-form-head { display: flex; justify-content: space-between; gap: 16px; align-items: end; }
      .module-form-head p { max-width: 620px; color: var(--muted); font-size: 13px; line-height: 1.45; }
      .module-grid { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 10px; }
      .module-option { position: relative; min-height: 90px; display: grid; grid-template-columns: 42px minmax(0,1fr) auto; gap: 12px; align-items: center; border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 14px 15px; background: var(--panel); cursor: pointer; transition: border-color .16s ease, background .16s ease, box-shadow .16s ease; }
      .module-option:hover { border-color: rgba(47,107,74,.4); }
      .module-option:has(input:checked) { border-color: rgba(47,107,74,.48); background: rgba(47,107,74,.035); box-shadow: inset 3px 0 0 var(--leaf); }
      .module-option input { position: absolute; opacity: 0; pointer-events: none; }
      .module-switch { width: 42px; height: 24px; position: relative; border-radius: 999px; background: #d6d2c8; transition: background .16s ease; }
      .module-switch::after { content: ""; position: absolute; width: 18px; height: 18px; top: 3px; left: 3px; border-radius: 50%; background: #fff; box-shadow: 0 1px 3px rgba(0,0,0,.18); transition: transform .16s ease; }
      .module-option input:focus-visible + .module-switch { outline: 3px solid rgba(47,107,74,.24); outline-offset: 3px; }
      .module-option input:checked + .module-switch { background: var(--leaf); }
      .module-option input:checked + .module-switch::after { transform: translateX(18px); }
      .module-actions { position: sticky; bottom: 12px; display: flex; justify-content: space-between; align-items: center; gap: 12px; border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 12px 14px; background: rgba(255,253,248,.96); box-shadow: 0 10px 28px rgba(36,42,36,.12); backdrop-filter: blur(10px); }
      .module-actions span { color: var(--muted); font-size: 12.5px; }
      @media (max-width: 720px) { .module-core, .module-grid { grid-template-columns: 1fr; } .module-form-head { display: grid; } .module-actions { align-items: stretch; flex-direction: column; } .module-actions .button { width: 100%; } }
    </style>
    <main id="main-content" tabindex="-1" class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/></svg><span>/</span><a href="/app/settings">Einstellungen</a><span>/</span><span>Portalbereiche</span></span>
      </div>
      <section class="page module-settings-page">
        <header class="module-settings-head"><span class="eyebrow">Liegenschaft · Navigation</span><h1>Portalbereiche</h1><p class="lede">Zeigen Sie nur Funktionen, die in {{.HouseName}} tatsächlich gebraucht werden.</p></header>
        {{if .PortalModulesSaved}}<div class="message success">Die sichtbaren Portalbereiche wurden gespeichert.</div>{{end}}
        {{if .PortalModulesError}}<div class="message error">Die Portalbereiche konnten nicht gespeichert werden.</div>{{end}}
        <section class="module-core" aria-label="Immer sichtbare Bereiche">
          <article class="panel module-core-card"><span class="module-core-icon"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/></svg></span><span class="module-core-copy"><strong>Hausüberblick</strong><span>Der zentrale Einstieg bleibt immer sichtbar.</span></span><span class="module-fixed">Immer aktiv</span></article>
          <article class="panel module-core-card"><span class="module-core-icon"><svg viewBox="0 0 24 24"><path d="M12 8.5a3.5 3.5 0 1 0 0 7 3.5 3.5 0 0 0 0-7z"/><path d="M19 12a7 7 0 0 0-.1-1l2-1.5-2-3.5-2.4 1M5.1 11l-2-1.5 2-3.5 2.4 1"/></svg></span><span class="module-core-copy"><strong>Einstellungen</strong><span>Die Verwaltung bleibt jederzeit erreichbar.</span></span><span class="module-fixed">Immer aktiv</span></article>
        </section>
        <form class="module-form" method="post" action="/app/settings/modules">
          <div class="module-form-head"><div><h2>Sichtbare Bereiche</h2><p>Ausgeschaltete Bereiche verschwinden aus Navigation, Hausüberblick und Einstellungen. Direkte Links sind anschließend ebenfalls gesperrt.</p></div></div>
          <div class="module-grid">
            {{range .PortalModuleOptions}}<label class="module-option"><span class="module-option-icon" aria-hidden="true"><svg viewBox="0 0 24 24"><path d="M5 5h14v14H5z"/><path d="M9 9h6M9 13h6"/></svg></span><span class="module-option-copy"><strong>{{.Label}}</strong><span>{{.Description}}</span></span><input type="checkbox" name="modules" value="{{.ID}}"{{if .Enabled}} checked{{end}}><span class="module-switch" aria-hidden="true"></span></label>{{end}}
          </div>
          <div class="module-actions"><span>Die Änderung gilt für alle Personen in dieser Liegenschaft.</span><button class="button primary" type="submit">Portalbereiche speichern</button></div>
        </form>
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

{{define "ebInterfaceImport"}}
{{template "appOpen" .}}
    <script src="/assets/attachments.js?v={{.AssetVersion}}" defer></script>
    <style>
      .invoice-import .page { gap: 18px; }
      .invoice-import .content-top .button { min-height: 44px; }
      .invoice-import .page-intro { display: grid; gap: 6px; }
      .invoice-import .page-intro .lede { max-width: 720px; }
      .invoice-import .flow-strip { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 9px; }
      .invoice-import .flow-step { display: grid; grid-template-columns: 34px minmax(0,1fr); gap: 10px; align-items: center; min-height: 64px; padding: 11px 13px; border: 1px solid var(--line); border-radius: 11px; background: var(--panel-soft); }
      .invoice-import .flow-step > span { width: 32px; height: 32px; display: grid; place-items: center; border-radius: 50%; background: var(--ink); color: #fff; font-weight: 850; }
      .invoice-import .flow-step.done > span { background: var(--leaf); }
      .invoice-import .flow-step.active { border-color: rgba(200,153,63,.52); background: rgba(200,153,63,.08); }
      .invoice-import .flow-step strong, .invoice-import .flow-step small { display: block; }
      .invoice-import .flow-step small { margin-top: 2px; color: var(--muted); line-height: 1.35; }
      .invoice-import .import-panel { display: grid; gap: 15px; }
      .invoice-import .import-panel-head { display: flex; align-items: start; justify-content: space-between; gap: 14px; }
      .invoice-import .import-panel-head h2 { margin: 0 0 4px; }
      .invoice-import .upload-form { display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 12px; align-items: end; padding: 15px; border: 1px dashed var(--gold); border-radius: 11px; background: #fffefb; }
      .invoice-import .upload-copy { display: grid; gap: 7px; }
      .invoice-import .upload-copy label { font-weight: 850; }
      .invoice-import .upload-copy input[type=file] { width: 100%; min-height: 46px; padding: 8px; border: 1px solid var(--line); border-radius: 8px; background: #fff; }
      .invoice-import .invoice-file-control { min-height: 48px; }
      .invoice-import .invoice-file-control > span { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
      .invoice-import .upload-form .button, .invoice-import .store-bar .button { min-height: 44px; }
      .invoice-import .privacy-note { display: flex; gap: 8px; align-items: start; margin: 0; color: var(--muted); font-size: 13px; line-height: 1.45; }
      .invoice-import .privacy-note::before { content: "✓"; flex: 0 0 auto; color: var(--gold-ink); font-weight: 900; }
      .invoice-import .profile-chip { display: inline-flex; min-height: 34px; align-items: center; padding: 0 10px; border: 1px solid var(--line); border-radius: 999px; background: var(--panel-soft); color: var(--gold-ink); font-size: 12px; font-weight: 850; white-space: nowrap; }
      .invoice-import .preview-actions { display: flex; align-items: center; justify-content: flex-end; gap: 8px; }
      .invoice-import .preview-actions .button { min-height: 44px; }
      .invoice-import .preview-actions .ghost { border-color: rgba(32,37,31,.14); color: var(--muted); background: transparent; }
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
        .invoice-import .flow-strip { gap: 6px; }
        .invoice-import .flow-step { grid-template-columns: 28px minmax(0,1fr); gap: 8px; min-height: 52px; padding: 8px; }
        .invoice-import .flow-step > span { width: 28px; height: 28px; }
        .invoice-import .flow-step small { display: none; }
        .invoice-import .upload-form { grid-template-columns: 1fr; }
        .invoice-import .upload-form .button { width: 100%; min-height: 48px; }
        .invoice-import .import-panel-head, .invoice-import .store-bar { align-items: stretch; flex-direction: column; }
        .invoice-import .invoice-hero { grid-template-columns: 1fr; align-items: start; }
        .invoice-import .invoice-amount { white-space: normal; }
        .invoice-import .invoice-parties, .invoice-import .invoice-dates { display: grid; grid-template-columns: 1fr; gap: 0; border: 1px solid var(--line); border-radius: 10px; background: #fff; overflow: hidden; }
        .invoice-import .party-card { grid-template-columns: minmax(82px,.45fr) minmax(0,1fr); }
        .invoice-import .date-card { grid-template-columns: minmax(116px,.55fr) minmax(0,1fr); }
        .invoice-import .party-card, .invoice-import .date-card { align-items: baseline; gap: 10px; border: 0; border-radius: 0; padding: 10px 12px; }
        .invoice-import .party-card + .party-card, .invoice-import .date-card + .date-card { border-top: 1px solid var(--line); }
        .invoice-import .store-bar .button { width: 100%; min-height: 48px; }
      }
      @media (max-width: 420px) {
        .invoice-import .page { gap: 14px; }
        .invoice-import .import-panel { gap: 12px; padding: 16px; }
        .invoice-import .page-intro .lede { font-size: 14px; }
        .invoice-import .preview-actions { justify-content: flex-start; flex-wrap: wrap; }
      }
      @media (max-width: 360px) {
        .invoice-import .date-card:last-child { grid-template-columns: minmax(0,1fr); gap: 2px; }
      }
    </style>
    <main id="main-content" tabindex="-1" class="app-main invoice-import">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M7 3h7l3 3v15H7z"/><path d="M14 3v4h4"/><path d="M9 13h6M9 17h6"/></svg><span>/</span><span>Dokumente</span><span>/</span><span>E-Rechnung</span></span>
        <div class="page-actions"><a class="button ghost" href="/app/dokumente">Dokumente</a></div>
      </div>
      <section class="page">
        <div class="page-intro">
          <h1>E-Rechnung ablegen</h1>
          <p class="lede">XML prüfen und geschützt für die Verwaltung speichern.</p>
        </div>
        <div class="flow-strip" aria-label="Ablauf">
          <div class="flow-step {{if .EBInterfacePreview}}done{{else}}active{{end}}"{{if not .EBInterfacePreview}} aria-current="step"{{end}}><span>1</span><div><strong>Prüfen</strong><small>XML &amp; Rechnungsdaten</small></div></div>
          <div class="flow-step {{if .EBInterfacePreview}}active{{end}}"{{if .EBInterfacePreview}} aria-current="step"{{end}}><span>2</span><div><strong>Ablegen</strong><small>Original geschützt speichern</small></div></div>
        </div>
        {{if .EBInterfaceImportMsg}}<p class="flash {{if .EBInterfaceImportOK}}ok{{end}}">{{.EBInterfaceImportMsg}}</p>{{end}}
        {{if not .EBInterfacePreview}}<section class="panel import-panel">
          <div class="import-panel-head">
            <div><div class="kicker">Neue Rechnung</div><h2>XML auswählen</h2><p class="muted">Für ebInterface 5.0 und 6.0.</p></div>
          </div>
          <form class="upload-form" method="post" action="/app/dokumente/rechnungen/import/preview" enctype="multipart/form-data">
            <div class="upload-copy">
              <label for="invoice-file">ebInterface-Datei</label>
              <span class="file-control invoice-file-control"><input id="invoice-file" type="file" name="invoice_file" accept=".xml,application/xml,text/xml" data-simple-file data-default-file-label="XML auswählen" required><span>XML auswählen</span></span>
              <span class="mini">XML bis {{.MaxEBInterfaceImportSize}}</span>
            </div>
            <button class="button primary" type="submit">Vorschau erstellen</button>
          </form>
          <p class="privacy-note">Lokal geprüft · nicht extern gesendet · nach 15 Minuten gelöscht.</p>
        </section>{{end}}

        {{with .EBInterfacePreview}}
          <section class="panel import-panel" id="preview">
            <div class="import-panel-head">
              <div><div class="kicker">Vorschau</div><h2>{{.Filename}}</h2><p class="muted">Geprüft {{.CreatedAt}} · noch nicht abgelegt</p></div>
              <div class="preview-actions"><span class="profile-chip">ebInterface {{.SourceVersion}}</span><a class="button small ghost" href="/app/dokumente/rechnungen/import">Andere Datei</a></div>
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
              <span class="mini">Nur Verwaltung · keine Buchung oder Zahlung.</span>
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
        .payment-import .import-panel { min-width: 0; grid-template-columns: minmax(0,1fr); }
        .payment-import .import-panel > * { min-width: 0; max-width: 100%; }
        .payment-import .import-panel-head, .payment-import .preview-head { display: grid; }
        .payment-import .period-form { width: 100%; display: grid; grid-template-columns: minmax(0,1fr) auto; }
        .payment-import .period-form label { min-width: 0; }
        .payment-import .period-form input, .payment-import .period-form .button, .payment-import .content-top .page-actions .button { min-height: 44px; }
        .payment-import .reference-row { grid-template-columns: 1fr; gap: 3px; }
        .payment-import .upload-form { min-width: 0; width: 100%; max-width: 100%; grid-template-columns: 1fr; }
        .payment-import .upload-copy, .payment-import .upload-copy > * { min-width: 0; max-width: 100%; }
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
        <div class="page-actions"><a class="button ghost" href="/app/settings/building?section=units">Zu den Einheiten</a></div>
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
            <p class="muted">Für diese Liegenschaft sind noch keine Einheiten angelegt.</p>
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

{{define "energyModeStrip"}}
  <section class="energy-mode-strip {{if .IsActiveMode}}active{{end}}" aria-label="Energiemodus">
    <div class="energy-mode-state">
      <span class="energy-mode-icon" aria-hidden="true"><span class="energy-ui-icon {{if .IsActiveMode}}energy-ui-icon-alert{{else}}energy-ui-icon-check{{end}}"></span></span>
      <div class="energy-mode-copy">
        <strong>{{if .IsActiveMode}}{{if .IsShadowMode}}Testlauf aktiv{{else}}Aktive Steuerung{{end}}{{else}}Nur beobachten{{end}}</strong>
        <span title="{{if .IsActiveMode}}{{if .IsShadowMode}}HAUSV protokolliert nur und schaltet kein Gerät.{{else}}Freigegebene Geräte folgen Ihren Regeln.{{end}}{{else}}HAUSV liest und empfiehlt, steuert aber kein Gerät.{{end}}">{{if .IsActiveMode}}{{if .IsShadowMode}}Keine Gerätewirkung{{else}}Geräte freigegeben{{end}}{{else}}Keine Steuerung{{end}}</span>
      </div>
    </div>
    {{if .Tariff}}{{if .Tariff.HasEstimate}}<a class="energy-peak-chip" href="#tarif" aria-label="Monatsspitze {{.Tariff.PeakKW}}, verrechnet {{.Tariff.BilledKW}}, {{.Tariff.AnnualPowerEUR}} pro Jahr — Details öffnen"><span class="energy-peak-chip-label">Monatsspitze</span><b>{{.Tariff.PeakKW}}</b><span class="sep" aria-hidden="true">·</span><span>verrechnet</span><b>{{.Tariff.BilledKW}}</b><span class="sep" aria-hidden="true">·</span><b>{{.Tariff.AnnualPowerEUR}}</b><span>/ Jahr</span>{{if .Tariff.CoverageLabel}}<span class="sep energy-peak-chip-coverage-sep" aria-hidden="true">·</span><span class="energy-peak-chip-coverage">{{.Tariff.CoverageLabel}}</span>{{end}}</a>{{end}}{{end}}
    {{if .IsActiveMode}}
      {{if .CanControlEnergy}}<form method="post" action="/app/energie/mode">
        <input type="hidden" name="mode" value="observe">
        <button class="energy-mode-action" type="submit" aria-label="Sofort zurück zu Nur beobachten"><span class="energy-mode-action-full">Sofort zurück zu „Nur beobachten“</span><span class="energy-mode-action-compact" aria-hidden="true">Nur beobachten</span></button>
      </form>{{end}}
    {{else}}
      {{if .CanControlEnergy}}<details class="energy-mode-control">
        <summary class="energy-mode-action" aria-label="Wirkungslosen Testlauf bewusst starten" title="Wirkungslosen Testlauf bewusst starten"><span class="energy-ui-icon energy-ui-icon-play" aria-hidden="true"></span><span class="energy-mode-action-full">Testlauf</span><span class="energy-mode-action-compact" aria-hidden="true">Testlauf</span></summary>
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
          <summary><span><strong>Ganzes Energieprofil löschen</strong><small>Zusätzlich Haus-Anzeigename, Wohnform, verknüpfte Einheit, Anlagen, Zuordnungen, Wartungspläne und Maßnahmen-Metadaten. Beginn und Ende des kostenlosen Anspruchs bleiben erhalten, damit eine Neueinrichtung den Zeitraum nicht neu startet.</small></span></summary>
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
