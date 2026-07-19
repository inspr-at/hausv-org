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
      --font-sans: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      --font-serif: Spectral, serif;
      --space-1:4px; --space-2:8px; --space-3:12px; --space-4:16px; --space-5:20px; --space-6:24px;
      --radius-xs:7px; --radius-sm:8px; --radius-md:10px; --radius-lg:12px; --radius-xl:14px; --radius-pill:999px;
      --shadow-panel:0 12px 30px rgba(32,37,31,.04);
      --shadow-dialog:0 28px 70px rgba(0,0,0,.34);
      --shadow-login:0 28px 70px rgba(0,0,0,.42);
      font-family: var(--font-sans);
{{end}}
{{define "hausvLandingMark"}}
<svg class="hausv-mark hausv-mark-with-text" viewBox="0 0 72 56" aria-hidden="true" focusable="false">
  <path class="mark-frame" d="M11 7h50c2.8 0 5 2.2 5 5v32c0 2.8-2.2 5-5 5H11c-2.8 0-5-2.2-5-5V12c0-2.8 2.2-5 5-5z"/>
  <path d="M15 34h42"/>
  <path d="M16 34v-8.5l7-5.5 7 5.5V34"/>
  <path d="M42 34v-8.5l7-5.5 7 5.5V34"/>
  <path d="M28 34V20.5L36 14l8 6.5V34"/>
  <path d="M32.5 34v-7h7v7"/>
  <path d="M19.5 28h3"/>
  <path d="M49.5 28h3"/>
  <path d="M32.5 23.5h7"/>
  <text class="mark-word" x="36" y="44.2" text-anchor="middle">hausv.org</text>
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
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=Spectral:wght@400;500;600;700&display=swap" rel="stylesheet">
  <style>
    :root {
      color-scheme: light;
{{template "designTokens" .}}
    }
    * { box-sizing: border-box; }
    body { margin: 0; color: var(--ink); background: #10160f; }
    /* Accessibility convention: all keyboard-reachable controls keep a visible focus ring. */
    :where(a, button, input, select, textarea, summary, [tabindex]):focus-visible { outline: 3px solid var(--gold-light); outline-offset: 3px; }
    .hero { position: relative; min-height: 100vh; overflow: hidden; display: grid; grid-template-rows: auto 1fr auto; }
    .hero::before {
      content: ""; position: absolute; inset: -16px;
      background: url('{{.Tenant.HeroImageURL}}') center 42% / cover no-repeat;
      filter: blur(3px) brightness(.74) saturate(.95); transform: scale(1.05); z-index: -2;
    }
    .hero::after {
      content: ""; position: absolute; inset: 0;
      background: linear-gradient(180deg, rgba(16,22,16,.52) 0%, rgba(16,22,16,.3) 34%, rgba(16,22,16,.6) 76%, rgba(16,22,16,.9) 100%);
      z-index: -1;
    }
    header { display: flex; justify-content: space-between; align-items: center; gap: 24px; padding: 28px clamp(20px,5vw,72px); color: #fff; }
    .brand { display: inline-flex; align-items: center; gap: 12px; text-decoration: none; color: #fff; }
    .mark { width: 58px; height: 46px; border-radius: var(--radius-sm); background: rgba(255,255,255,.16); border: 1px solid rgba(255,255,255,.4); backdrop-filter: blur(6px); display: grid; place-items: center; color: var(--gold-light); }
    .mark .hausv-mark { width: 52px; height: 40px; display: block; stroke: currentColor; stroke-width: 2.15; fill: none; stroke-linecap: round; stroke-linejoin: round; }
    .mark .mark-word { fill: currentColor; stroke: none; font-family: var(--font-sans); font-size: 6.2px; font-weight: 900; letter-spacing: .02em; }
    .brand .name { font-family: var(--font-serif); font-weight: 600; font-size: 17px; }
    nav { display: flex; gap: 24px; color: rgba(255,255,255,.92); font-size: 14px; font-weight: 600; }
    nav a { color: inherit; text-decoration: none; padding-bottom: 4px; border-bottom: 1px solid rgba(231,197,116,.65); }
    nav a:hover { color: #fff; border-bottom-color: var(--gold-light); }
    main { display: grid; grid-template-columns: minmax(0,1.1fr) minmax(320px,420px); gap: clamp(28px,6vw,64px); align-items: end; padding: 0 clamp(20px,5vw,72px) clamp(40px,8vh,72px); }
    .copy { max-width: 760px; color: #fff; }
    .eyebrow { font-size: 12px; font-weight: 700; text-transform: uppercase; letter-spacing: .2em; color: var(--gold-light); margin-bottom: 18px; }
    h1 { margin: 0; font-family: var(--font-serif); font-weight: 500; font-size: clamp(46px,7vw,72px); line-height: 1.0; letter-spacing: -.01em; text-shadow: 0 2px 30px rgba(0,0,0,.3); }
    .lead { max-width: 440px; margin: 24px 0 0; font-size: clamp(17px,2vw,19px); line-height: 1.55; color: rgba(255,255,255,.9); }
    .meta { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 18px; margin-top: 34px; max-width: 620px; }
    .meta div { border-left: 1px solid rgba(231,197,116,.58); padding-left: 16px; min-width: 0; }
    .meta strong { display: block; font-weight: 700; font-size: 14px; color: #fff; margin-bottom: 6px; }
    .meta span { color: rgba(255,255,255,.78); font-size: 13px; line-height: 1.4; }
    .login { background: var(--paper); border-radius: var(--radius-xl); padding: 30px; box-shadow: var(--shadow-login); }
    .login h2 { margin: 0; font-family: var(--font-serif); font-weight: 600; font-size: 26px; }
    .login p { color: var(--muted); line-height: 1.5; margin: 11px 0 22px; font-size: 14.5px; }
    label { display: block; font-size: 11.5px; font-weight: 700; text-transform: uppercase; letter-spacing: .07em; color: var(--gold-ink); margin-bottom: 8px; }
    input { width: 100%; border: 1px solid #e2dac9; border-radius: 10px; padding: 14px 15px; font: inherit; background: #fffefb; color: var(--ink); }
    button { width: 100%; border: 0; border-radius: 10px; padding: 15px 16px; margin-top: 13px; font: inherit; font-weight: 700; color: #fff; background: var(--ink); cursor: pointer; }
    button:hover { background: #000; }
    .notice { border: 1px solid rgba(32,37,31,.16); background: rgba(200,153,63,.1); color: #6a5320; border-radius: 10px; padding: 12px 14px; font-size: 14px; line-height: 1.4; margin-bottom: 16px; }
    .notice.warn { border-color: rgba(173,92,27,.22); background: rgba(231,197,116,.2); color: #6c491a; }
    .dev-link { display: block; border: 1px solid var(--line); background: #fffefb; color: var(--ink); border-radius: 10px; padding: 12px 14px; margin: -4px 0 16px; text-align: center; text-decoration: none; font-size: 14px; font-weight: 700; }
    .dev-link:hover { border-color: var(--gold); }
    .sso-button { display: flex; align-items: center; justify-content: center; min-height: 48px; border-radius: 10px; background: var(--ink); color: #fff; text-decoration: none; font-weight: 700; margin-bottom: 14px; }
    .sso-button:hover { background: #000; }
    .divider { display: grid; grid-template-columns: 1fr auto 1fr; align-items: center; gap: 10px; color: var(--muted); font-size: 13px; margin: 12px 0; }
    .divider::before, .divider::after { content: ""; height: 1px; background: var(--line); }
    .foot-note { margin: 16px 0 0; font-size: 13px; line-height: 1.4; color: var(--soft); }
    footer { padding: 20px clamp(20px,5vw,72px) 26px; color: rgba(255,255,255,.85); font-weight: 500; font-size: 14px; }
    footer .version { margin-left: 8px; color: rgba(255,255,255,.54); font-size: 12px; }
    @media (max-width: 860px) {
      nav { display: none; }
      main { grid-template-columns: 1fr; align-items: start; gap: 28px; }
      .meta { grid-template-columns: 1fr; gap: 12px; max-width: 320px; margin-top: 22px; }
      .meta div:not(:first-child) { display: none; }
      h1 { font-size: clamp(40px,12vw,56px); }
    }
  </style>
</head>
<body>
  <section class="hero">
    <header>
      <a class="brand" href="/" aria-label="WEG Portal Startseite"><span class="mark">{{template "hausvLandingMark" .}}</span><span class="name">{{.Tenant.Name}}</span></a>
      <nav aria-label="Seitennavigation">
        <a href="#login">Anmelden</a>
      </nav>
    </header>
    <main>
      <div class="copy">
        <div class="eyebrow">WEG Portal</div>
        <h1>Alles rund um unser gemeinsames Haus.</h1>
        <p class="lead">Der private digitale Eingang für die Hausgemeinschaft, erreichbar per persönlichem E-Mail-Zugang oder SSO.</p>
        <div class="meta" aria-label="Portalüberblick">
          {{if .HasUnitCount}}<div><strong>{{.UnitCount}}</strong><span>{{.UnitCountLabel}} im Haus, direkt aus den hinterlegten Einheiten.</span></div>{{else}}<div><strong>Eingeladen</strong><span>Zugang nur für freigegebene E-Mail-Adressen der Hausgemeinschaft.</span></div>{{end}}
          <div><strong>Einmalig</strong><span>Anmeldung per SSO oder zeitlich begrenztem E-Mail-Link.</span></div>
          <div><strong>Parkplatz</strong><span>Verbrauch und Abrechnung bleiben im geschützten Portal.</span></div>
        </div>
      </div>
      <section id="login" class="login" aria-label="Anmeldung">
        <h2>Anmelden</h2>
        <p>{{if .OIDCConfigured}}Melden Sie sich per SSO an oder verwenden Sie einen einmaligen E-Mail-Link.{{else}}Geben Sie Ihre E-Mail-Adresse ein. Wenn sie eingeladen ist, schicken wir einen einmaligen Anmeldelink.{{end}}</p>
        {{if .OIDCConfigured}}<a class="sso-button" href="/auth/oidc/start">Mit {{.OIDCProviderName}} anmelden</a>{{end}}
        {{if and .OIDCConfigured .EmailLoginAvailable}}<div class="divider"><span>oder</span></div>{{end}}
        {{if .Sent}}
          <div class="notice">Wenn die Adresse eingeladen ist, wurde ein Link verschickt. Bitte Posteingang prüfen.</div>
          {{if not .MailConfigured}}<div class="notice warn">Mailversand ist lokal noch nicht konfiguriert. In Produktion kommt SMTP aus agenix.</div>{{end}}
          {{if .DevLoginLink}}<a class="dev-link" href="{{.DevLoginLink}}">Lokalen Dev-Login öffnen</a>{{end}}
        {{end}}
        {{if .Denied}}<div class="notice warn">Diese Adresse ist noch nicht eingeladen.</div>{{end}}
        {{if .EmailLoginAvailable}}
          <form method="post" action="/auth/request">
            <label for="email">E-Mail-Adresse</label>
            <input id="email" name="email" type="email" inputmode="email" autocomplete="email" required placeholder="name@example.com">
            <button type="submit">Anmeldelink senden</button>
          </form>
          <p class="foot-note">Link 15 Minuten gültig · privat für die Hausgemeinschaft</p>
        {{else}}
          <div class="notice">E-Mail-Anmeldelinks sind nicht aktiv. Bitte SSO verwenden.</div>
        {{end}}
      </section>
    </main>
    <footer>{{.Tenant.Address}} · Privat für die Hausgemeinschaft <span class="version">{{.AppVersion}}</span></footer>
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
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800;900&family=Spectral:wght@400;500;600;700&display=swap" rel="stylesheet">
  <script src="/assets/landing.js?v={{.AssetVersion}}" defer></script>
  <style>
    :root {
      color-scheme: light;
{{template "designTokens" .}}
      --sky:#6f9ab3; --mint:#dfeee5; --rose:#f0d7d0; --cream:#faf6ed;
    }
    * { box-sizing: border-box; }
    html { scroll-behavior: smooth; }
    body { margin: 0; color: var(--ink); background: var(--cream); font-family: var(--font-sans); }
    a { color: inherit; }
    :where(a, button):focus-visible { outline: 3px solid var(--gold-light); outline-offset: 3px; }
    .landing-hero { position: relative; min-height: 86svh; display: grid; grid-template-rows: auto minmax(0,1fr); overflow: hidden; color: #fff; background: #162018; }
    .landing-hero::before { content: ""; position: absolute; inset: 0; background: url('{{.LandingHeroURL}}') center 48% / cover no-repeat; transform: scale(1.01); }
    .landing-hero::after { content: ""; position: absolute; inset: 0; background: linear-gradient(90deg, rgba(12,18,13,.86) 0%, rgba(12,18,13,.74) 34%, rgba(12,18,13,.32) 66%, rgba(12,18,13,.12) 100%); }
    .landing-nav, .landing-copy { position: relative; z-index: 1; width: min(1180px,100%); margin: 0 auto; padding-left: clamp(20px,4vw,42px); padding-right: clamp(20px,4vw,42px); }
    .landing-nav { display: flex; justify-content: space-between; align-items: center; gap: 18px; padding-top: 26px; padding-bottom: 20px; }
    .landing-brand { display: inline-flex; align-items: center; text-decoration: none; color: #fff; font-weight: 800; }
    .landing-mark { width: 74px; height: 54px; border-radius: 8px; display: grid; place-items: center; border: 1px solid rgba(255,255,255,.42); background: rgba(255,255,255,.14); backdrop-filter: blur(8px); color: var(--gold-light); }
    .landing-mark .hausv-mark { width: 64px; height: 48px; display: block; stroke: currentColor; stroke-width: 2.2; fill: none; stroke-linecap: round; stroke-linejoin: round; }
    .landing-mark .mark-word { fill: currentColor; stroke: none; font-family: var(--font-sans); font-size: 6.2px; font-weight: 900; letter-spacing: .02em; }
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
    .landing-proof { display: flex; flex-wrap: wrap; gap: 10px; margin-top: 28px; max-width: 760px; }
    .landing-proof span { border: 1px solid rgba(255,255,255,.28); border-radius: 999px; padding: 8px 12px; color: rgba(255,255,255,.86); background: rgba(255,255,255,.08); font-size: 13px; font-weight: 800; }
    .section { padding: clamp(48px,8vw,86px) clamp(20px,4vw,42px); }
    .section-inner { position: relative; z-index: 1; width: min(1180px,100%); margin: 0 auto; }
    .section h2 { margin: 0; font-family: var(--font-serif); font-size: clamp(34px,4.6vw,56px); line-height: 1.02; font-weight: 500; max-width: 820px; }
    .section-kicker { color: var(--gold-ink); font-size: 12px; font-weight: 900; letter-spacing: .14em; text-transform: uppercase; margin-bottom: 14px; }
    .section-lead { max-width: 760px; margin-top: 18px; color: var(--muted); font-size: 18px; line-height: 1.55; }
    .feature-grid { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 18px; margin-top: 34px; }
    .feature { border: 1px solid var(--line); border-radius: 8px; padding: 18px; background: var(--panel); display: grid; gap: 14px; align-content: start; box-shadow: 0 12px 26px rgba(32,37,31,.035); }
    .feature-icon { width: 48px; height: 48px; border-radius: 8px; display: grid; place-items: center; background: rgba(200,153,63,.12); color: var(--gold-ink); }
    .feature-icon svg { width: 25px; height: 25px; display: block; stroke: currentColor; stroke-width: 1.9; fill: none; stroke-linecap: round; stroke-linejoin: round; }
    .feature strong { display: block; font-family: var(--font-serif); font-size: 23px; line-height: 1.15; }
    .feature p { margin: 10px 0 0; color: var(--muted); line-height: 1.5; }
    .positioning-strip { margin-top: 30px; display: grid; grid-template-columns: 68px minmax(0,1fr) auto; gap: 18px; align-items: center; border: 1px solid rgba(47,107,74,.2); border-radius: 10px; background: rgba(47,107,74,.06); padding: 20px 22px; }
    .positioning-mark { width: 52px; height: 52px; border-radius: 50%; display: grid; place-items: center; background: rgba(47,107,74,.12); color: var(--leaf); }
    .positioning-mark svg { width: 26px; height: 26px; stroke: currentColor; stroke-width: 1.9; fill: none; stroke-linecap: round; stroke-linejoin: round; }
    .positioning-strip strong { display: block; font-family: var(--font-serif); font-size: clamp(24px,2.4vw,32px); line-height: 1.08; }
    .positioning-strip p { margin: 6px 0 0; color: var(--muted); line-height: 1.5; }
    .positioning-tag { justify-self: end; border: 1px solid rgba(200,153,63,.28); border-radius: 999px; padding: 8px 12px; background: rgba(200,153,63,.12); color: var(--gold-ink); font-size: 12px; font-weight: 900; letter-spacing: .08em; text-transform: uppercase; white-space: nowrap; }
    .roadmap-section { background: #fffefb; }
    .roadmap-grid { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 16px; margin-top: 30px; }
    .roadmap-card { min-height: 216px; border: 1px solid var(--line); border-radius: 10px; background: var(--panel); padding: 20px; display: grid; grid-template-rows: auto auto 1fr; gap: 12px; box-shadow: 0 16px 34px rgba(32,37,31,.035); }
    .roadmap-card .feature-icon { background: rgba(47,107,74,.09); color: var(--leaf); }
    .roadmap-card strong { display: block; font-family: var(--font-serif); font-size: 25px; line-height: 1.12; }
    .roadmap-card p { margin: 0; color: var(--muted); line-height: 1.5; }
    .roadmap-card small { align-self: end; color: var(--gold-ink); font-size: 12px; font-weight: 900; letter-spacing: .08em; text-transform: uppercase; }
    .band { background: #fffefb; border-top: 1px solid var(--line); border-bottom: 1px solid var(--line); }
    .visual-section { position: relative; overflow: hidden; }
    .visual-section::before { content: ""; position: absolute; inset: 0; pointer-events: none; background-repeat: no-repeat; background-size: cover; background-position: center; filter: saturate(.86); }
    .features-section::before { background-image: url('/assets/landing-features.jpg'); opacity: .12; }
    .features-section .feature { background: rgba(255,254,251,.92); backdrop-filter: blur(2px); }
    .roles-section { background: #f7f3ea; }
    .roles-section::before { background-image: linear-gradient(90deg, rgba(247,243,234,.96) 0%, rgba(247,243,234,.86) 52%, rgba(247,243,234,.76) 100%), url('/assets/landing-roles.jpg'); opacity: 1; background-position: center; }
    .roles-section .section-inner { display: grid; gap: 28px; }
    .use-grid { counter-reset: role-card; display: grid; grid-template-columns: repeat(4,minmax(0,1fr)); gap: 14px; margin-top: 4px; }
    .use { counter-increment: role-card; min-height: 258px; padding: 22px; border: 1px solid rgba(231,224,210,.92); border-radius: 8px; background: rgba(255,254,251,.9); backdrop-filter: blur(3px); box-shadow: 0 18px 44px rgba(32,37,31,.045); display: grid; grid-template-columns: minmax(0,1fr) 42px; grid-template-rows: 42px minmax(74px,auto) minmax(0,1fr); gap: 16px 18px; align-items: start; }
    .use::before { content: "0" counter(role-card); grid-column: 2; grid-row: 1; width: 42px; height: 42px; display: grid; place-items: center; border-radius: 50%; background: rgba(200,153,63,.12); color: var(--gold-ink); font-size: 12px; font-weight: 900; letter-spacing: .04em; }
    .use span { grid-column: 1; grid-row: 1; align-self: center; color: var(--gold-ink); font-size: 12px; font-weight: 900; letter-spacing: .1em; text-transform: uppercase; }
    .use strong { grid-column: 1 / -1; grid-row: 2; align-self: start; max-width: 260px; font-family: var(--font-serif); font-size: clamp(25px,2.2vw,31px); line-height: 1.12; }
    .use p { grid-column: 1 / -1; grid-row: 3; align-self: start; max-width: 280px; color: var(--muted); line-height: 1.45; }
    .trust-section { background: #fffefb; }
    .trust-layout { display: grid; grid-template-columns: minmax(300px,.78fr) minmax(560px,1.22fr); gap: clamp(32px,5vw,72px); align-items: start; }
    .trust-copy { display: grid; gap: 24px; align-content: start; }
    .trust-copy h2 { max-width: 560px; font-size: clamp(42px,5vw,66px); }
    .trust-copy .section-lead { margin-top: 0; max-width: 520px; }
    .trust-summary { display: grid; border-top: 1px solid var(--line); }
    .trust-line { display: grid; grid-template-columns: 44px minmax(0,1fr); gap: 16px; padding: 18px 0; border-bottom: 1px solid var(--line); }
    .trust-number { width: 32px; height: 32px; display: grid; place-items: center; border-radius: 50%; background: rgba(200,153,63,.11); color: var(--gold-ink); font-size: 11px; font-weight: 900; letter-spacing: .08em; }
    .trust-line strong { display: block; font-size: 17px; line-height: 1.25; }
    .trust-line p { margin: 5px 0 0; color: var(--muted); line-height: 1.48; }
    .trust-board { border: 1px solid rgba(47,107,74,.22); border-radius: 10px; background: var(--panel); box-shadow: 0 22px 54px rgba(32,37,31,.055); overflow: hidden; }
    .trust-board-head { display: grid; grid-template-columns: 72px minmax(0,1fr); gap: 18px; align-items: center; padding: 26px; border-bottom: 1px solid var(--line); background: rgba(47,107,74,.045); }
    .trust-seal { width: 72px; height: 72px; border-radius: 50%; display: grid; place-items: center; background: rgba(47,107,74,.11); color: var(--leaf); }
    .trust-seal svg, .trust-proof svg, .cost-note svg { width: 28px; height: 28px; stroke: currentColor; stroke-width: 1.9; fill: none; stroke-linecap: round; stroke-linejoin: round; }
    .trust-board-head strong { display: block; font-family: var(--font-serif); font-size: clamp(28px,3vw,40px); line-height: 1.05; }
    .trust-board-head p { margin: 8px 0 0; max-width: 520px; color: var(--muted); line-height: 1.45; }
    .trust-proof-grid { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); padding: 6px 26px 10px; }
    .trust-proof { min-height: 138px; display: grid; grid-template-columns: 34px minmax(0,1fr); gap: 14px; align-content: start; border-bottom: 1px solid var(--line); padding: 20px 0; }
    .trust-proof:nth-child(odd) { padding-right: 24px; border-right: 1px solid var(--line); }
    .trust-proof:nth-child(even) { padding-left: 24px; }
    .trust-proof:nth-last-child(-n+2) { border-bottom: 0; }
    .trust-proof svg { width: 23px; height: 23px; margin-top: 1px; color: var(--leaf); }
    .trust-proof strong { display: block; font-size: 16px; line-height: 1.25; }
    .trust-proof p { margin: 6px 0 0; color: var(--muted); font-size: 14px; line-height: 1.45; }
    .cost-section { background: #f7f3ea; }
    .cost-layout { display: grid; grid-template-columns: minmax(300px,.72fr) minmax(560px,1.28fr); gap: clamp(30px,5vw,68px); align-items: start; }
    .cost-copy h2 { max-width: 520px; font-size: clamp(40px,4.7vw,62px); }
    .cost-copy .section-lead { max-width: 500px; }
    .cost-panel { border: 1px solid rgba(138,123,63,.26); border-radius: 10px; background: var(--panel); box-shadow: 0 22px 54px rgba(32,37,31,.055); padding: 8px 28px; }
    .cost-row { display: grid; grid-template-columns: 118px minmax(0,1fr); gap: 22px; align-items: center; border-bottom: 1px solid var(--line); padding: 24px 0; }
    .cost-row:last-child { border-bottom: 0; }
    .cost-value { min-height: 82px; display: grid; place-items: center; border: 1px solid rgba(47,107,74,.15); border-radius: 10px; background: #fffaf0; color: var(--leaf); font-family: var(--font-serif); font-size: 36px; font-weight: 700; line-height: 1; text-align: center; }
    .cost-row:nth-child(2) .cost-value { color: var(--gold-ink); border-color: rgba(200,153,63,.22); }
    .cost-row:nth-child(3) .cost-value { color: #8b5a52; border-color: rgba(139,90,82,.15); background: #fff6f2; font-size: 25px; }
    .cost-row strong { display: block; font-family: var(--font-serif); font-size: clamp(25px,2.5vw,36px); line-height: 1.05; }
    .cost-row p { margin: 8px 0 0; color: var(--muted); line-height: 1.48; }
    .cost-note { display: grid; grid-template-columns: 32px minmax(0,1fr); gap: 13px; align-items: start; border: 1px solid rgba(200,153,63,.28); border-radius: 10px; background: rgba(255,254,251,.76); margin-top: 16px; padding: 17px 18px; color: var(--muted); line-height: 1.5; }
    .cost-note svg { width: 25px; height: 25px; color: var(--gold-ink); }
    .cost-note strong { display: block; color: var(--ink); }
    .imprint-grid { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 14px; margin-top: 28px; }
    .imprint-card { border: 1px solid var(--line); border-radius: 8px; padding: 16px; background: var(--panel); }
    .imprint-card strong { display: block; font-size: 15px; }
    .imprint-card p { margin: 7px 0 0; color: var(--muted); line-height: 1.5; }
    .final-cta { position: relative; overflow: hidden; background: #172019; color: #fff; }
    .final-cta::before { content: ""; position: absolute; inset: 0; background: url('/assets/landing-closing.jpg') center center / cover no-repeat; opacity: .32; pointer-events: none; }
    .final-cta::after { content: ""; position: absolute; inset: 0; background: rgba(23,32,25,.72); pointer-events: none; }
    .final-cta .section-inner { position: relative; z-index: 1; }
    .final-cta .section-lead { color: rgba(255,255,255,.78); }
    .final-cta .landing-actions { margin-top: 28px; }
    .final-cta .landing-button.secondary { color: #fff; }
    footer { padding: 24px clamp(20px,4vw,42px); color: #6b6f63; background: #fffefb; border-top: 1px solid var(--line); }
    footer div { width: min(1180px,100%); margin: 0 auto; display: flex; justify-content: space-between; gap: 16px; flex-wrap: wrap; font-size: 14px; }
    @media (max-width: 900px) {
      .landing-links { display: none; }
      .landing-hero { min-height: 88svh; }
      .landing-hero::after { background: linear-gradient(180deg, rgba(12,18,13,.78) 0%, rgba(12,18,13,.5) 46%, rgba(12,18,13,.88) 100%); }
      .landing-copy { padding-top: 64px; }
      .feature-grid, .use-grid, .trust-layout, .cost-layout, .imprint-grid, .roadmap-grid, .positioning-strip { grid-template-columns: 1fr; }
      .positioning-tag { justify-self: start; }
      .trust-proof-grid { grid-template-columns: 1fr; }
      .trust-proof:nth-child(odd), .trust-proof:nth-child(even) { padding-left: 0; padding-right: 0; border-right: 0; }
      .trust-proof:nth-last-child(2) { border-bottom: 1px solid var(--line); }
      .cost-row { grid-template-columns: 92px minmax(0,1fr); gap: 16px; }
      .cost-value { min-height: 70px; font-size: 29px; }
      .cost-row strong { font-size: clamp(24px,7vw,30px); line-height: 1.08; }
      .cost-row p { font-size: 16px; line-height: 1.45; }
      .use { min-height: 0; grid-template-rows: 42px auto auto; }
      .use strong, .use p { max-width: none; }
    }
  </style>
</head>
<body>
  <section class="landing-hero">
    <header class="landing-nav">
      <a class="landing-brand" href="/" aria-label="hausv.org"><span class="landing-mark">{{template "hausvLandingMark" .}}</span></a>
      <nav class="landing-links" aria-label="Navigation">
        <a href="#funktionen">Funktionen</a>
        <a href="#ausblick">Ausblick</a>
        <a href="#sicherheit">Sicherheit</a>
        <a href="#preise">Preise</a>
        <a href="#impressum">Impressum</a>
        <a href="#kontakt" class="js-mail-link" data-mail-local="{{.ContactLocal}}" data-mail-domain="{{.ContactDomain}}">{{.ContactDisplay}}</a>
      </nav>
    </header>
    <div class="landing-copy">
      <div class="landing-eyebrow">Hausverwaltung von und für Mehrparteien</div>
      <h1>Ein Portal für alle, die ein Haus gemeinsam verwalten.</h1>
      <p class="landing-lead">Kommunikation, Transparenz und Self-Service für WEGs, Wohnungen und Mehrparteienhäuser. Klar für Eigentümer, Mieter, Beiräte und kleine Verwaltungen.</p>
      <div class="landing-actions">
        <a class="landing-button primary js-mail-link" href="#kontakt" data-mail-local="{{.ContactLocal}}" data-mail-domain="{{.ContactDomain}}" data-mail-subject="hausv.org anfragen" data-mail-reveal="false">Kostenlos starten</a>
        <a class="landing-button secondary" href="#funktionen">Funktionen ansehen</a>
      </div>
      <div class="landing-proof" aria-label="Kurzversprechen">
        <span>Bis 25 Wohneinheiten kostenlos</span>
        <span>Datenschutz mitgedacht</span>
        <span>Kommunikation statt Buchhaltung</span>
        <span>KI nur mit Opt-in</span>
      </div>
    </div>
  </section>

  <section id="funktionen" class="section band visual-section features-section">
    <div class="section-inner">
      <div class="section-kicker">Betrieb statt Bauchgefühl</div>
      <h2>Alles für den Alltag einer Hausgemeinschaft.</h2>
      <p class="section-lead">Ein ruhiger Arbeitsbereich für wiederkehrende Abläufe: informieren, entscheiden, dokumentieren, nachverfolgen. Finanzdaten bleiben Status und Nachweis, nicht Buchhaltung.</p>
      <div class="feature-grid">
        <div class="feature"><span class="feature-icon"><svg viewBox="0 0 24 24"><path d="M4 11.5 12 5l8 6.5"/><path d="M6 10.5V20h12v-9.5"/><path d="M9 20v-5h6v5"/></svg></span><div><strong>Hausüberblick</strong><p>Offene Punkte, Termine, Dokumente, Zahlungsstatus und nächste Schritte direkt auf der Startseite.</p></div></div>
        <div class="feature"><span class="feature-icon"><svg viewBox="0 0 24 24"><path d="M4 5h16v12H8l-4 3z"/><path d="M8 9h8M8 13h6"/></svg></span><div><strong>Aushang & Termine</strong><p>Mitteilungen, Wartungen, Fristen und Versammlungen mit Anhängen und Archiv.</p></div></div>
        <div class="feature"><span class="feature-icon"><svg viewBox="0 0 24 24"><path d="M5 18.5V7a3 3 0 0 1 3-3h8a3 3 0 0 1 3 3v5a3 3 0 0 1-3 3H10z"/><path d="M8.5 8.5h7"/></svg></span><div><strong>Anliegen</strong><p>Meldungen mit Fotos, Kommentaren, Zuständigkeit und nachvollziehbarem Status.</p></div></div>
        <div class="feature"><span class="feature-icon"><svg viewBox="0 0 24 24"><path d="M7 3h7l4 4v14H7z"/><path d="M14 3v5h5"/><path d="M9 13h6M9 17h6"/></svg></span><div><strong>Dokumente</strong><p>Protokolle, Abrechnungen und Unterlagen sicher abgelegt und passend freigegeben.</p></div></div>
        <div class="feature"><span class="feature-icon"><svg viewBox="0 0 24 24"><path d="M6 18V9M12 18V5M18 18v-6"/><path d="M4 18h16"/></svg></span><div><strong>Abstimmungen</strong><p>Beschlüsse vorbereiten, transparent abstimmen und Ergebnisse dokumentieren.</p></div></div>
        <div class="feature"><span class="feature-icon"><svg viewBox="0 0 24 24"><path d="M5 16h14"/><path d="m7 16 1.5-5h7L17 16"/><path d="M7 16v3M17 16v3"/><path d="M8 19h1M15 19h1"/></svg></span><div><strong>Parkplätze</strong><p>Stellplätze, Ladeverbrauch, Zahlungserinnerungen und CSV-Export im Blick.</p></div></div>
        <div class="feature"><span class="feature-icon"><svg viewBox="0 0 24 24"><path d="M8 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8z"/><path d="M3 21a5 5 0 0 1 10 0"/><path d="M16 12h5M18.5 9.5v5"/></svg></span><div><strong>Rollen & Rechte</strong><p>Eigentümer, Mieter, Beirat und Verwaltung sehen nur, was für sie gedacht ist.</p></div></div>
        <div class="feature"><span class="feature-icon"><svg viewBox="0 0 24 24"><path d="M18 8a6 6 0 1 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9"/><path d="M10 21h4"/></svg></span><div><strong>Benachrichtigungen</strong><p>Mails führen direkt zum betroffenen Aushang, Anliegen, Dokument oder Zahlungspunkt.</p></div></div>
        <div class="feature"><span class="feature-icon"><svg viewBox="0 0 24 24"><path d="M12 3v3M12 18v3M4.6 6.6l2.1 2.1M17.3 17.3l2.1 2.1M3 12h3M18 12h3M4.6 17.4l2.1-2.1M17.3 6.7l2.1-2.1"/><path d="M9 12a3 3 0 1 0 6 0 3 3 0 0 0-6 0z"/></svg></span><div><strong>KI-Assistenz</strong><p>Zusammenfassen und Formulieren auf Wunsch, nur tenantweise und mit Opt-in.</p></div></div>
      </div>
      <div class="positioning-strip" aria-label="Produktgrenze">
        <span class="positioning-mark"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 7h16"/><path d="M4 12h16"/><path d="M4 17h10"/><path d="M18 15l2 2 3-4"/></svg></span>
        <div><strong>Kommunikation statt Buchhaltung.</strong><p>hausv.org ist der Kommunikations- und Transparenz-Layer. Wir zeigen Status, Dokumente und Nachweise, integrieren mit bestehenden Systemen und vermeiden bewusst Buchführung, Nebenkostenabrechnung, Steuerlogik und Mahnwesen.</p></div>
        <span class="positioning-tag">Transparenz statt Buchung</span>
      </div>
    </div>
  </section>

  <section id="ausblick" class="section roadmap-section">
    <div class="section-inner">
      <div class="section-kicker">Roadmap</div>
      <h2>Ausblick ohne Nebel.</h2>
      <p class="section-lead">Einige Bausteine laufen bereits im Pilot, andere sind bewusst als nächste Schritte markiert. Die Linie bleibt gleich: besser koordinieren, sauber dokumentieren, offen integrieren. Keine eigene Buchhaltung.</p>
      <div class="roadmap-grid" aria-label="Geplante Produktbausteine">
        <div class="roadmap-card"><span class="feature-icon"><svg viewBox="0 0 24 24"><path d="M8 11a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7z"/><path d="M3 20a5 5 0 0 1 10 0"/><path d="M16 7h5M16 12h5M16 17h5"/></svg></span><strong>Dienstleister einbinden</strong><p>Handwerker sehen nur zugewiesene Anliegen, können Status, Fotos, Rückfragen und Termine ergänzen.</p><small>Datenschutzprüfung offen</small></div>
        <div class="roadmap-card"><span class="feature-icon"><svg viewBox="0 0 24 24"><path d="M5 4h14v16H5z"/><path d="M8 8h8M8 12h5M8 16h4"/><path d="m15 16 2 2 3-4"/></svg></span><strong>Übergaben dokumentieren</strong><p>Mobile Protokolle für Räume, Zählerstände, Schlüssel, Mängel, Fotos und Bestätigung.</p><small>Pilot verfügbar</small></div>
        <div class="roadmap-card"><span class="feature-icon"><svg viewBox="0 0 24 24"><path d="M4 7h16M4 12h16M4 17h10"/><path d="M17 15l3 3 3-5"/></svg></span><strong>Zahlungsstatus zeigen</strong><p>Offen, bezahlt oder überfällig als geschützte Statusinformation pro Einheit, ohne Sollstellung oder Mahnwesen.</p><small>Pilot verfügbar</small></div>
        <div class="roadmap-card"><span class="feature-icon"><svg viewBox="0 0 24 24"><path d="M6 4h12v16H6z"/><path d="M9 8h6M9 12h6M9 16h3"/></svg></span><strong>AT-Schnittstellen</strong><p>camt.053 und camt.054 lesen Zahlungsstatus. BMD/RZL und ebInterface bleiben Übergaben an bestehende Systeme.</p><small>Österreich-first</small></div>
        <div class="roadmap-card"><span class="feature-icon"><svg viewBox="0 0 24 24"><path d="M7 3v4M17 3v4"/><path d="M4.5 6h15v14h-15z"/><path d="M8 12h8M8 16h5"/></svg></span><strong>Kalender abonnieren</strong><p>Termine, Versammlungen und Dienstleister-Zeitfenster als geschützter ICS-Feed.</p><small>Pilot verfügbar</small></div>
        <div class="roadmap-card"><span class="feature-icon"><svg viewBox="0 0 24 24"><path d="M8 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6z"/><path d="M3.5 20a4.5 4.5 0 0 1 9 0"/><path d="M16 11a2.5 2.5 0 1 0 0-5"/><path d="M16.5 15a4 4 0 0 1 4 4"/></svg></span><strong>Kontakte pro Verwaltung</strong><p>Hausmeister, Notdienste und wiederkehrende Dienstleister zentral pflegen und gezielt verwenden.</p><small>Pilot verfügbar</small></div>
      </div>
    </div>
  </section>

  <section class="section visual-section roles-section">
    <div class="section-inner">
      <div class="section-kicker">Für alle Rollen</div>
      <h2>Ein Portal, mehrere Perspektiven.</h2>
      <div class="use-grid">
        <div class="use"><span>Eigentümer</span><strong>Entscheiden und prüfen</strong><p>Beschlüsse, Dokumente, Hausstatus und relevante Abrechnungen bleiben auffindbar.</p></div>
        <div class="use"><span>Mieter</span><strong>Melden ohne Umwege</strong><p>Anliegen erfassen, Aushänge lesen und informiert bleiben.</p></div>
        <div class="use"><span>Verwalter</span><strong>Steuern und dokumentieren</strong><p>Rollen, Status, Benachrichtigungen und Audit-Spuren für den Alltag.</p></div>
        <div class="use"><span>Beirat</span><strong>Mitsehen und begleiten</strong><p>Überblick dort, wo er hilft, ohne Rechte zu vermischen.</p></div>
      </div>
    </div>
  </section>

  <section id="sicherheit" class="section band trust-section">
    <div class="section-inner trust-layout">
      <div class="trust-copy">
        <div>
          <div class="section-kicker">Sicherheit & Datenschutz</div>
          <h2>Vertrauen zuerst.</h2>
        </div>
        <p class="section-lead">Einfach genug für die Hausgemeinschaft. Strukturiert genug für Verwaltung, Datenschutz und saubere Abläufe.</p>
        <div class="trust-summary" aria-label="Sicherheitsprinzipien">
          <div class="trust-line"><span class="trust-number">01</span><div><strong>Getrennte Häuser</strong><p>Eigene Domain, eigene Rollen, eigene Sichtbarkeit.</p></div></div>
          <div class="trust-line"><span class="trust-number">02</span><div><strong>Geschützte Dateiwege</strong><p>Anhänge und Dokumente laufen über App-Routen.</p></div></div>
          <div class="trust-line"><span class="trust-number">03</span><div><strong>Datensparsam</strong><p>Nur Profile, Rechte und Protokolle, die der Betrieb wirklich braucht.</p></div></div>
          <div class="trust-line"><span class="trust-number">04</span><div><strong>KI nur mit Opt-in</strong><p>Keine automatische Auswertung, Aktivierung pro Haus.</p></div></div>
        </div>
      </div>
      <aside class="trust-board" aria-label="Sicherheitsversprechen">
        <div class="trust-board-head">
          <span class="trust-seal"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 3 5 6v5c0 4.7 2.9 8.4 7 10 4.1-1.6 7-5.3 7-10V6z"/><path d="m8.8 12.2 2.1 2.1 4.3-4.6"/></svg></span>
          <div><strong>Bleibt privat.</strong><p>Ein Portal pro Hausgemeinschaft, klare Rollen und geschützte Wege für sensible Inhalte.</p></div>
        </div>
        <div class="trust-proof-grid">
          <div class="trust-proof"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m5 12 4 4L19 6"/></svg><div><strong>Keine öffentlichen Datei-Links</strong><p>Downloads laufen über geschützte App-Routen.</p></div></div>
          <div class="trust-proof"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m5 12 4 4L19 6"/></svg><div><strong>Keine KI ohne Zustimmung</strong><p>Funktionen werden bewusst pro Haus aktiviert.</p></div></div>
          <div class="trust-proof"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m5 12 4 4L19 6"/></svg><div><strong>Keine unkontrollierte Weitergabe</strong><p>Daten bleiben im vorgesehenen Haus-Kontext und werden nur gezielt zugänglich gemacht.</p></div></div>
          <div class="trust-proof"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m5 12 4 4L19 6"/></svg><div><strong>Rollen und Rechte pro Haus</strong><p>Eigentümer, Mieter, Beirat und Verwaltung sauber getrennt.</p></div></div>
          <div class="trust-proof"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m5 12 4 4L19 6"/></svg><div><strong>Audit-Spuren für wichtige Aktionen</strong><p>Änderungen bleiben nachvollziehbar.</p></div></div>
          <div class="trust-proof"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m5 12 4 4L19 6"/></svg><div><strong>Datenschutz mitgedacht</strong><p>Zuständigkeiten, Rechtsgrundlagen und sparsame Profile werden vor der Freigabe geklärt.</p></div></div>
        </div>
      </aside>
    </div>
  </section>

  <section id="preise" class="section cost-section">
    <div class="section-inner cost-layout">
      <div class="cost-copy">
        <div class="section-kicker">Fair geregelt</div>
        <h2>Kosten fair.</h2>
      <p class="section-lead">hausv.org soll kleinen Hausgemeinschaften helfen und für größere Verwaltungen trotzdem planbar bleiben: nach Einheit, nicht nach Bauchgefühl.</p>
      </div>
      <div>
        <div class="cost-panel" aria-label="Faire Nutzung und Preise">
          <div class="cost-row">
            <span class="cost-value">25</span>
            <div><strong>Kostenlos</strong><p>Bis 25 Wohneinheiten. Für kleine Hausgemeinschaften, private Betreuung und den fairen Einstieg.</p></div>
          </div>
          <div class="cost-row">
            <span class="cost-value">1€</span>
            <div><strong>1 € pro Monat</strong><p>Je Wohneinheit als Richtwert für größere Verwaltungen. Wohnungen und vergleichbare Nutzungseinheiten zählen; Zubehör wie Keller oder Stellplätze nicht automatisch.</p></div>
          </div>
          <div class="cost-row">
            <span class="cost-value">frei</span>
            <div><strong>Spenden</strong><p>Optional. Hilft bei Betrieb, Backups, Sicherheit und Weiterentwicklung.</p></div>
          </div>
        </div>
        <div class="cost-note">
          <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 3v18"/><path d="M6 7h12"/><path d="M7 7l-4 7h8z"/><path d="M17 7l-4 7h8z"/></svg>
          <div><strong>Fair bleibt fair.</strong> Klare Einheitspreise, keine versteckten Grundgebühren, kein Verkaufsdruck.</div>
        </div>
      </div>
    </div>
  </section>

  <section id="impressum" class="section">
    <div class="section-inner">
      <div class="section-kicker">Impressum</div>
      <h2>Impressum & Kontakt</h2>
      <p class="section-lead">Die folgenden Felder sind bewusst als Platzhalter angelegt und werden vor dem produktiven Start mit den konkreten Betreiberangaben ergänzt.</p>
      <div class="imprint-grid">
        <div class="imprint-card"><strong>Medieninhaber / Betreiber</strong><p>[Name oder Firma, Rechtsform]</p></div>
        <div class="imprint-card"><strong>Sitz / Anschrift</strong><p>[Straße und Hausnummer, PLZ Ort, Land]</p></div>
        <div class="imprint-card"><strong>Kontakt</strong><p><a id="kontakt" class="js-mail-link" href="#kontakt" data-mail-local="{{.ContactLocal}}" data-mail-domain="{{.ContactDomain}}">{{.ContactDisplay}}</a></p></div>
        <div class="imprint-card"><strong>Unternehmensgegenstand</strong><p>[Software-/IT-Dienstleistungen, digitale Hausverwaltungsplattform]</p></div>
        <div class="imprint-card"><strong>Firmenbuch / UID</strong><p>[Firmenbuchnummer, Firmenbuchgericht, UID-Nummer]</p></div>
        <div class="imprint-card"><strong>Gewerbebehörde / Kammer</strong><p>[Bezirkshauptmannschaft/Magistrat], Mitglied der WKO [Bundesland]</p></div>
        <div class="imprint-card"><strong>Anwendbare Vorschriften</strong><p>[Gewerbeordnung, abrufbar unter ris.bka.gv.at]</p></div>
        <div class="imprint-card"><strong>Blattlinie</strong><p>Informationen über hausv.org und digitale Selbstverwaltung für Mehrparteienhäuser.</p></div>
      </div>
      <p class="mini">Platzhalter auf Basis der WKO-Impressum-Orientierung final mit den echten Betreiberangaben ausfüllen.</p>
    </div>
  </section>

  <section class="section final-cta">
    <div class="section-inner">
      <div class="section-kicker">Starten</div>
      <h2>Aus einer Hausgemeinschaft heraus gebaut, für echte Hausgemeinschaften.</h2>
      <p class="section-lead">Entstanden aus dem eigenen Verwaltungsalltag: zu viel Papier, zu viele Tools, zu wenig Überblick. Deshalb ein Portal, das WEGs und Mehrparteienhäuser wirklich nutzen können.</p>
      <div class="landing-actions">
        <a class="landing-button primary js-mail-link" href="#kontakt" data-mail-local="{{.ContactLocal}}" data-mail-domain="{{.ContactDomain}}" data-mail-subject="hausv.org Pilotzugang" data-mail-reveal="false">Kontakt aufnehmen</a>
        <a class="landing-button secondary" href="{{.PrimaryAppURL}}">Beispielportal öffnen</a>
      </div>
    </div>
  </section>

  <footer>
    <div><span>hausv.org · sicher, fair und datensparsam</span><span><a href="#impressum">Impressum</a> · <a class="js-mail-link" href="#kontakt" data-mail-local="{{.ContactLocal}}" data-mail-domain="{{.ContactDomain}}">{{.ContactDisplay}}</a> · {{.AppVersion}}</span></div>
  </footer>
</body>
</html>
{{end}}

{{define "appStyles"}}
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=Spectral:wght@400;500;600;700&display=swap" rel="stylesheet">
  <style>
    :root {
      color-scheme: light;
{{template "designTokens" .}}
    }
    * { box-sizing: border-box; }
    body { margin: 0; background: var(--paper); color: var(--ink); }
    a { color: inherit; }
    button, input { font: inherit; }
    /* Accessibility convention: all keyboard-reachable controls keep a visible focus ring. */
    :where(a, button, input, select, textarea, summary, [tabindex]):focus-visible { outline: 3px solid var(--gold); outline-offset: 3px; }
    .app-shell { min-height: 100vh; display: grid; grid-template-columns: 264px minmax(0,1fr); background: var(--paper); }
    .sidebar { position: sticky; top: 0; height: 100vh; min-height: 0; display: flex; flex-direction: column; gap: 18px; padding: 22px 16px 18px; color: rgba(255,255,255,.86); background: radial-gradient(circle at 20% 0%, rgba(255,255,255,.08), transparent 28%), var(--nav); border-right: 1px solid rgba(255,255,255,.08); }
    .side-brand { flex: 0 0 auto; display: grid; grid-template-columns: 50px 1fr; gap: 14px; align-items: center; padding: 0 8px 12px; }
    .side-mark { width: 48px; height: 48px; border-radius: var(--radius-sm); display: grid; place-items: center; color: var(--gold-light); border: 1px solid rgba(255,255,255,.34); background: rgba(255,255,255,.07); text-decoration: none; }
	    .side-mark:hover { border-color: rgba(231,197,116,.72); background: rgba(255,255,255,.1); color: #f0d58c; }
	    .side-mark svg { width: 38px; height: 34px; display: block; stroke: currentColor; stroke-width: 2.3; fill: none; stroke-linecap: round; stroke-linejoin: round; }
	    .side-title { display: block; font-family: var(--font-serif); font-size: 18px; font-weight: 600; line-height: 1.1; color: #fff; text-decoration: none; }
	    .side-sub { display: block; margin-top: 5px; font-size: 14px; color: rgba(255,255,255,.72); }
	    .side-code { display: inline-flex; align-items: center; width: max-content; max-width: 100%; margin-top: 7px; border: 1px solid rgba(231,197,116,.28); border-radius: var(--radius-pill); padding: 2px 8px; color: var(--gold-light); background: rgba(231,197,116,.08); font-size: 11px; font-weight: 900; letter-spacing: .08em; text-transform: uppercase; }
    .side-nav { flex: 1 1 auto; min-height: 0; display: grid; align-content: start; gap: 5px; overflow-y: auto; overflow-x: hidden; padding-right: 3px; }
    .side-nav::-webkit-scrollbar { width: 7px; }
    .side-nav::-webkit-scrollbar-thumb { border-radius: var(--radius-pill); background: rgba(255,255,255,.16); }
    .nav-toggle, .mobile-menu-toggle { display: none; }
    .nav-item { position: relative; min-height: 44px; display: flex; align-items: center; gap: 12px; padding: 9px 12px; border-radius: var(--radius-xs); color: rgba(255,255,255,.78); text-decoration: none; font-size: 15px; font-weight: 600; }
    .nav-item:hover { color: #fff; background: rgba(255,255,255,.06); }
    .nav-item.active { color: #fff; background: rgba(255,255,255,.08); }
    .nav-item.active::before { content: ""; position: absolute; left: -16px; top: 0; bottom: 0; width: 4px; background: var(--gold); }
    .nav-item.disabled { color: rgba(255,255,255,.38); cursor: default; }
    .nav-item.disabled:hover { background: transparent; }
    .nav-icon { width: 23px; height: 23px; display: grid; place-items: center; flex: 0 0 auto; color: currentColor; }
    .nav-icon svg { width: 22px; height: 22px; stroke: currentColor; stroke-width: 1.9; fill: none; stroke-linecap: round; stroke-linejoin: round; }
    .nav-label { min-width: 0; }
    .nav-badge { margin-left: auto; min-width: 25px; height: 22px; display: inline-flex; align-items: center; justify-content: center; border-radius: var(--radius-pill); padding: 0 7px; background: var(--gold); color: #172019; font-size: 11px; font-weight: 900; line-height: 1; }
    .side-foot { flex: 0 0 auto; margin-top: 0; border-top: 1px solid rgba(255,255,255,.16); padding: 16px 8px 0; display: grid; gap: 12px; }
    .side-user { display: grid; grid-template-columns: 42px 1fr; gap: 12px; align-items: center; }
    .avatar { width: 42px; height: 42px; border-radius: 50%; display: grid; place-items: center; background: var(--gold); color: #fff; font-weight: 800; border: 1px solid rgba(255,255,255,.25); }
    .side-user strong { display: -webkit-box; max-height: 2.5em; color: #fff; font-size: 14px; line-height: 1.22; overflow: hidden; overflow-wrap: anywhere; -webkit-line-clamp: 2; -webkit-box-orient: vertical; }
    .side-user span, .side-version { color: rgba(255,255,255,.64); font-size: 13px; }
    .version-button { justify-self: start; width: auto; min-height: 30px; border: 1px solid rgba(255,255,255,.16); border-radius: var(--radius-pill); padding: 4px 10px; background: rgba(255,255,255,.05); color: rgba(255,255,255,.72); font: inherit; font-size: 12.5px; font-weight: 800; cursor: pointer; }
    .version-button:hover { border-color: rgba(231,197,116,.48); color: #fff; background: rgba(255,255,255,.09); }
    .logout-form { margin: 0; }
    .logout-button { width: 100%; min-height: 42px; display: inline-flex; align-items: center; justify-content: center; gap: 10px; border: 1px solid rgba(255,255,255,.24); border-radius: var(--radius-xs); color: rgba(255,255,255,.92); background: transparent; font-weight: 700; cursor: pointer; }
    .logout-button:hover { border-color: var(--gold); color: #fff; }
    .app-main { min-width: 0; padding-bottom: 58px; }
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
    .issue-card, .entry, .event-card, .document-row, .vote-card, tr[id^="parking-month-"] { scroll-margin-top: 82px; }
    .home-hero { position: relative; min-height: 178px; display: flex; align-items: center; overflow: hidden; border-bottom: 1px solid var(--line); background: #f7f3ea; padding: 38px clamp(28px,4vw,72px) 36px; }
    .home-hero::before { content: ""; position: absolute; inset: 0; background: url('{{.Tenant.HeroImageURL}}') center 47% / cover no-repeat; }
    .home-hero::after { content: ""; position: absolute; inset: 0; background: linear-gradient(90deg, rgba(247,243,234,.98) 0%, rgba(247,243,234,.93) 32%, rgba(247,243,234,.58) 55%, rgba(247,243,234,.18) 100%); }
    .home-hero-copy { position: relative; z-index: 1; width: min(720px,100%); }
    .home-hero h1 { font-size: clamp(44px,5.2vw,58px); }
    .home-hero p { margin-top: 15px; color: var(--muted); font-size: 16px; line-height: 1.5; }
    .home-page { padding-top: 24px; }
    .home-page .panel { padding: 24px 26px; }
    .home-grid { display: grid; grid-template-columns: minmax(0,1.08fr) minmax(360px,.92fr); gap: 22px 24px; align-items: start; }
    .home-stack { display: grid; gap: 22px; min-width: 0; }
    .home-grid .section-head { align-items: center; margin-bottom: 18px; }
    .home-grid .section-head .kicker { margin-bottom: 0; }
    .section-link { display: inline-flex; align-items: center; gap: 8px; color: var(--ink); text-decoration: none; font-size: 13px; font-weight: 700; white-space: nowrap; }
    .section-link::after { content: "›"; color: var(--gold-ink); font-size: 21px; line-height: 1; }
    .section-link:hover { color: var(--gold-ink); }
    .home-status-panel { grid-column: 1 / -1; padding: 24px 26px 26px; }
    .home-full { grid-column: 1 / -1; }
    .home-status-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 18px; margin-bottom: 20px; }
    .home-status-head .kicker { margin-bottom: 0; }
    .home-status-grid { display: grid; grid-template-columns: repeat(auto-fit,minmax(210px,1fr)); gap: 20px 32px; align-items: center; }
    .status-card { min-width: 0; display: grid; grid-template-columns: 64px minmax(0,1fr); gap: 14px; align-items: center; color: inherit; text-decoration: none; }
    .status-card:hover strong { color: var(--gold-ink); }
    .status-icon { width: 64px; height: 64px; border-radius: 50%; display: inline-grid; place-items: center; line-height: 0; background: rgba(200,153,63,.14); color: var(--gold-ink); }
    .status-icon svg { display: block; width: 24px; height: 24px; margin: 0; stroke: currentColor; stroke-width: 1.9; fill: none; stroke-linecap: round; stroke-linejoin: round; overflow: visible; }
    .status-card.status-announcements .status-icon { background: rgba(47,107,74,.12); color: var(--leaf); }
    .status-card.status-events .status-icon { background: rgba(200,153,63,.15); color: var(--gold-ink); }
    .status-card.status-issues .status-icon { background: rgba(158,42,43,.1); color: #9e2a2b; }
    .status-card.status-parking .status-icon { background: rgba(76,103,138,.12); color: #365475; }
    .status-card strong { display: block; font-family: var(--font-serif); font-size: 24px; line-height: 1.02; }
    .status-card > span:not(.status-icon) { display: block; min-width: 0; color: var(--muted); font-size: 13.5px; line-height: 1.35; overflow-wrap: anywhere; }
    .status-card > span:not(.status-icon) > span { display: block; }
    .status-card .status-label { color: var(--soft); font-size: 11px; font-weight: 800; letter-spacing: .06em; text-transform: uppercase; }
    .home-list { display: grid; gap: 10px; }
    .home-list-row { display: grid; grid-template-columns: 52px minmax(0,1fr) auto; gap: 14px; align-items: center; min-height: 74px; border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 12px 14px; background: var(--panel-soft); color: inherit; text-decoration: none; }
    .home-list-row:hover { border-color: var(--gold); }
    .home-list-row svg { display: block; width: 22px; height: 22px; margin: 0; stroke: currentColor; stroke-width: 1.8; fill: none; stroke-linecap: round; stroke-linejoin: round; overflow: visible; }
    .home-list-icon { width: 52px; height: 52px; border-radius: var(--radius-sm); display: inline-grid; place-items: center; line-height: 0; background: rgba(200,153,63,.12); color: var(--gold-ink); }
    .home-list-row strong { display: block; font-size: 15.5px; overflow-wrap: anywhere; }
    .home-list-row > span:not(.home-list-icon):not(.pill) { display: block; min-width: 0; color: var(--muted); font-size: 13px; line-height: 1.35; overflow-wrap: anywhere; }
    .home-list-row > span:not(.home-list-icon):not(.pill) > span { display: block; margin-top: 3px; }
    .home-card-actions { margin-top: 12px; display: flex; flex-wrap: wrap; gap: 10px; }
    .home-card-actions .section-link { min-height: 34px; padding: 0 2px; }
    .home-events-list, .document-dashboard-list { overflow: hidden; border: 1px solid var(--line); border-radius: var(--radius-sm); background: #fffefb; }
    .home-event-row { display: grid; grid-template-columns: 56px minmax(0,1fr) auto; gap: 14px; align-items: center; min-height: 74px; padding: 12px 14px; color: inherit; text-decoration: none; border-bottom: 1px solid var(--line); }
    .home-event-row:last-child { border-bottom: 0; }
    .home-event-row:hover, .document-dashboard-row:hover { background: rgba(200,153,63,.07); }
    .home-date { display: grid; justify-items: center; align-content: center; min-height: 48px; color: var(--ink); font-weight: 800; text-transform: uppercase; }
    .home-date strong { font-family: var(--font-serif); font-size: 24px; line-height: 1; }
    .home-date span { margin-top: 3px; font-size: 11px; letter-spacing: .06em; }
    .home-event-copy strong { display: block; font-size: 14px; line-height: 1.25; overflow-wrap: anywhere; }
    .home-event-copy span { display: block; margin-top: 5px; color: var(--muted); font-size: 12.5px; line-height: 1.35; overflow-wrap: anywhere; }
    .document-dashboard-row { display: grid; grid-template-columns: minmax(0,1fr) auto auto auto 24px; gap: 14px; align-items: center; min-height: 44px; padding: 10px 12px; color: inherit; text-decoration: none; border-bottom: 1px solid var(--line); font-size: 12.5px; }
    .document-dashboard-row:last-child { border-bottom: 0; }
    .document-file-title { min-width: 0; display: inline-flex; align-items: center; gap: 9px; }
    .document-file-title svg, .document-download svg { width: 17px; height: 17px; stroke: currentColor; stroke-width: 1.8; fill: none; stroke-linecap: round; stroke-linejoin: round; }
    .document-file-title svg { flex: 0 0 auto; color: #b65045; }
    .document-file-title strong { min-width: 0; overflow-wrap: anywhere; font-size: 13px; }
    .document-dashboard-row > span:not(.document-file-title):not(.document-download) { color: var(--muted); white-space: nowrap; }
    .document-download { color: var(--gold-ink); display: grid; place-items: center; }
    .parking-summary { display: grid; grid-template-columns: 64px minmax(0,1fr) auto; gap: 16px; align-items: center; border: 1px solid #d9e1ea; border-radius: var(--radius-sm); padding: 18px 20px; background: linear-gradient(90deg, rgba(76,103,138,.08), rgba(255,254,251,.96)); }
    .parking-summary .status-icon { width: 58px; height: 58px; background: rgba(76,103,138,.12); color: #365475; }
    .parking-summary strong { display: block; font-family: var(--font-serif); font-size: 24px; line-height: 1.08; }
    .parking-summary p { margin-top: 5px; color: var(--muted); font-size: 13.5px; line-height: 1.4; }
    .parking-check { width: 48px; height: 48px; border-radius: 50%; display: grid; place-items: center; justify-self: end; background: rgba(47,107,74,.12); color: var(--leaf); border: 1px solid rgba(47,107,74,.16); }
    .parking-check svg { width: 22px; height: 22px; stroke: currentColor; stroke-width: 2.2; fill: none; stroke-linecap: round; stroke-linejoin: round; }
    .parking-summary .pill, .home-list-row .pill { justify-self: end; }
    .entries { display: grid; gap: 22px; }
    .entry + .entry { border-top: 1px solid var(--line); padding-top: 22px; }
    .entry-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 14px; }
    .entry h3 a { color: inherit; text-decoration: none; }
    .entry h3 a:hover { color: var(--gold-ink); }
    .entry p, .entry-body { margin-top: 8px; color: #5c5f54; line-height: 1.6; text-wrap: pretty; }
    .entry-meta { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; margin-top: 10px; color: var(--soft); font-size: 12.5px; }
    .entry-actions { display: flex; gap: 8px; flex-wrap: wrap; justify-content: flex-end; }
    .entry-actions form { margin: 0; }
    .archive-tools { display: grid; gap: 12px; margin: -4px 0 20px; }
    .filter-form { display: grid; grid-template-columns: minmax(220px,1fr) auto; gap: 10px; align-items: end; }
    .filter-form.audit-filter { grid-template-columns: minmax(180px,.55fr) minmax(240px,1fr) auto auto; }
    .filter-form label { margin: 0; }
    .audit-panel { display: grid; gap: 16px; }
    .audit-summary-grid { margin: 0; }
    .audit-summary-grid .metric-card { background: #fffdf8; }
    .audit-filter-panel { border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel-soft); overflow: hidden; }
    .audit-filter-panel > summary { list-style: none; cursor: pointer; display: grid; grid-template-columns: auto minmax(0,1fr); gap: 10px; align-items: baseline; padding: 13px 16px; color: var(--muted); }
    .audit-filter-panel > summary::-webkit-details-marker { display: none; }
    .audit-filter-panel > summary::after { content: "›"; justify-self: end; color: var(--gold-ink); font-size: 22px; line-height: 1; transition: transform .18s ease; }
    .audit-filter-panel[open] > summary::after { transform: rotate(90deg); }
    .audit-filter-panel > summary span { color: var(--gold-ink); font-size: 11px; font-weight: 900; letter-spacing: .07em; text-transform: uppercase; }
    .audit-filter-panel > summary strong { min-width: 0; color: var(--ink); overflow-wrap: anywhere; }
    .audit-filter-panel .audit-filter { padding: 0 16px 14px; }
    .audit-active-filters { display: flex; gap: 8px; flex-wrap: wrap; padding: 0 16px 14px; }
    .audit-timeline { display: grid; gap: 0; border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel); overflow: hidden; }
    .audit-day { margin: 0; padding: 14px 18px 10px; color: var(--gold-ink); font-size: 12px; font-weight: 900; letter-spacing: .08em; text-transform: uppercase; background: rgba(251,248,240,.82); border-bottom: 1px solid var(--line); }
    .audit-row { display: grid; grid-template-columns: 82px 18px minmax(0,1fr); gap: 14px; align-items: start; padding: 16px 18px; border-bottom: 1px solid var(--line); }
    .audit-row:last-child { border-bottom: 0; }
    .audit-marker { width: 12px; height: 12px; border-radius: 50%; margin-top: 7px; background: var(--gold); box-shadow: 0 0 0 5px rgba(200,153,63,.14); }
    .audit-row.audit-add .audit-marker { background: var(--leaf); box-shadow: 0 0 0 5px rgba(47,107,74,.12); }
    .audit-row.audit-danger .audit-marker { background: #9e2a2b; box-shadow: 0 0 0 5px rgba(158,42,43,.11); }
    .audit-time { display: grid; gap: 3px; color: var(--ink); font-variant-numeric: tabular-nums; text-align: right; }
    .audit-time strong { font-family: var(--font-serif); font-size: 20px; line-height: 1.05; }
    .audit-time span { color: var(--soft); font-size: 11.5px; font-weight: 800; }
    .audit-main { min-width: 0; display: grid; gap: 10px; }
    .audit-row-head { display: grid; grid-template-columns: minmax(0,1fr); gap: 12px; align-items: start; }
    .audit-action { display: grid; gap: 7px; min-width: 0; }
    .audit-action strong { min-width: 0; font-size: 16px; line-height: 1.35; overflow-wrap: anywhere; }
    .audit-pill { justify-self: start; gap: 7px; }
    .audit-pill::before { content: ""; width: 8px; height: 8px; border-radius: 50%; background: currentColor; }
    .audit-pill.audit-add { background: rgba(47,107,74,.12); color: var(--leaf); }
    .audit-pill.audit-danger { background: rgba(158,42,43,.11); color: #9e2a2b; }
    .audit-pill.audit-change { background: rgba(200,153,63,.16); color: #8a6a1f; }
	.audit-meta { display: flex; gap: 10px 18px; flex-wrap: wrap; color: var(--muted); font-size: 13px; line-height: 1.35; }
    .audit-meta span { min-width: 0; overflow-wrap: anywhere; }
    .audit-meta strong { color: var(--gold-ink); font-size: 10.5px; font-weight: 900; letter-spacing: .06em; text-transform: uppercase; }
    .audit-meta em { color: var(--soft); font-style: normal; }
    .audit-details { min-width: 0; border-top: 1px dashed var(--line); padding-top: 9px; }
    .audit-details summary { cursor: pointer; color: var(--gold-ink); font-size: 12px; font-weight: 900; letter-spacing: .06em; text-transform: uppercase; }
    .audit-details .chips { gap: 7px; margin-top: 8px; }
    .audit-details .chip { max-width: 100%; white-space: normal; overflow-wrap: anywhere; }
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
    .agenda-list { display: grid; gap: 10px; margin-bottom: 22px; }
    .event-card { display: grid; grid-template-columns: 58px minmax(0,1fr); gap: 13px; align-items: start; border: 1px solid var(--line); border-radius: var(--radius-sm); padding: var(--space-3); color: inherit; background: var(--panel-soft); text-decoration: none; }
    .event-card:hover { border-color: var(--gold); }
    .event-card.past { opacity: .68; }
    .date-badge { min-height: 58px; display: grid; place-items: center; align-content: center; gap: 2px; border-radius: var(--radius-sm); background: var(--ink); color: #fff; font-weight: 800; text-align: center; }
    .date-badge strong { font-family: var(--font-serif); font-size: 24px; line-height: .95; }
    .date-badge span { font-size: 11px; text-transform: uppercase; letter-spacing: .08em; }
    .event-info { min-width: 0; display: grid; gap: 7px; }
    .event-info h3 { font-size: 18px; overflow-wrap: anywhere; }
    .event-info p { color: var(--muted); line-height: 1.45; font-size: 13.5px; }
    .event-meta { display: flex; flex-wrap: wrap; align-items: center; gap: 7px; color: var(--soft); font-size: 12.5px; font-weight: 700; }
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
    .issue-form { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 12px; }
    .issue-form label { display: grid; gap: 7px; color: var(--gold-ink); font-size: 11px; font-weight: 800; letter-spacing: .06em; text-transform: uppercase; }
    .issue-form .full { grid-column: 1 / -1; }
    .issue-form input, .issue-form select, .issue-form textarea { width: 100%; border: 1px solid #e2dac9; border-radius: var(--radius-sm); padding: 12px; font: inherit; background: #fffefb; color: var(--ink); }
    .issue-form textarea { min-height: 148px; resize: vertical; line-height: 1.45; }
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
    .attachment-picker-meta { color: var(--soft); font-size: 11.5px; font-weight: 700; }
    .attachment-picker-remove { width: 30px; height: 30px; display: grid; place-items: center; border: 1px solid rgba(32,37,31,.16); border-radius: var(--radius-xs); background: var(--panel); color: var(--ink); font: inherit; font-size: 18px; line-height: 1; cursor: pointer; }
    .attachment-picker-remove:hover { border-color: #9e2a2b; color: #9e2a2b; }
    .attachment-picker-progress { display: none; height: 5px; border-radius: var(--radius-pill); background: #ece5d6; overflow: hidden; }
    .attachment-picker-progress span { display: block; width: 40%; height: 100%; border-radius: inherit; background: var(--gold); animation: upload-progress 1.1s ease-in-out infinite; }
    .attachment-picker.is-uploading .attachment-picker-progress { display: block; }
    @keyframes upload-progress { 0% { transform: translateX(-120%); } 100% { transform: translateX(260%); } }
    .issue-form .hint { margin-top: 2px; color: var(--soft); font-size: 12px; font-weight: 600; letter-spacing: 0; text-transform: none; }
    .issue-form button { grid-column: 1 / -1; min-height: 46px; border: 1px solid var(--ink); border-radius: var(--radius-sm); background: var(--ink); color: #fff; font: inherit; font-weight: 800; cursor: pointer; }
    .issue-form button:hover { background: #2c3329; }
    .issue-flash { margin: 0 0 14px; padding: 11px 13px; border-radius: var(--radius-sm); font-size: 13.5px; font-weight: 700; border: 1px solid transparent; }
    .issue-flash.ok { background: rgba(47,107,74,.12); color: var(--leaf); border-color: rgba(47,107,74,.25); }
    .issue-flash.warn { background: rgba(200,153,63,.14); color: #93701d; border-color: rgba(200,153,63,.3); }
    .issue-list { display: grid; gap: 10px; }
    .issue-card { border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 13px; background: #fffefb; display: grid; gap: 10px; }
	.issue-card h3 { font-size: 18px; }
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
    .dialog::backdrop { background: rgba(23,32,25,.42); }
    .dialog form { margin: 0; display: grid; grid-template-rows: auto minmax(0,1fr); max-height: inherit; }
    .dialog-head { display: flex; justify-content: space-between; align-items: center; gap: 14px; padding: 20px 22px; border-bottom: 1px solid var(--line); }
    .dialog-head h2 { font-size: 25px; }
    .dialog-close { width: 34px; height: 34px; border: 1px solid var(--line); border-radius: var(--radius-xs); background: var(--panel-soft); color: var(--ink); font-size: 22px; line-height: 1; cursor: pointer; }
    .dialog-body { display: grid; gap: 14px; padding: 20px 22px 22px; overflow: auto; overscroll-behavior: contain; }
    .dialog-body > button:last-child { position: sticky; bottom: -1px; z-index: 2; box-shadow: 0 -12px 0 12px var(--panel), 0 -10px 18px rgba(255,254,251,.92); }
    .dialog-grid { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 12px; }
    .dialog-grid .full { grid-column: 1 / -1; }
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
    .parking-empty { display: grid; grid-template-columns: 220px minmax(0,1fr) auto; gap: 24px 30px; align-items: center; }
    .parking-empty-art { width: min(220px,100%); aspect-ratio: 1.25; justify-self: center; color: var(--gold); opacity: .92; }
    .parking-empty-art svg { width: 100%; height: 100%; display: block; stroke: currentColor; fill: none; stroke-width: 1.7; stroke-linecap: round; stroke-linejoin: round; }
    .parking-empty-art .soft-fill { fill: rgba(200,153,63,.11); stroke: none; }
    .parking-empty-copy { display: grid; gap: 8px; min-width: 0; }
    .parking-empty-copy h2 { font-size: clamp(28px,3.4vw,40px); }
    .parking-empty-actions { display: flex; flex-wrap: wrap; justify-content: flex-end; gap: 10px; align-self: center; }
    .parking-empty-steps { grid-column: 1 / -1; display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 0; border-top: 1px solid var(--line); padding-top: 20px; }
    .parking-empty-step { display: grid; grid-template-columns: 44px minmax(0,1fr); gap: 13px; align-items: start; min-width: 0; padding: 0 22px; border-left: 1px solid var(--line); }
    .parking-empty-step:first-child { border-left: 0; padding-left: 0; }
    .parking-empty-step:last-child { padding-right: 0; }
    .parking-empty-step-number { width: 36px; height: 36px; border-radius: 50%; display: grid; place-items: center; background: rgba(200,153,63,.13); color: var(--gold-ink); font-weight: 900; }
    .parking-empty-step strong { display: block; font-family: var(--font-serif); font-size: 18px; line-height: 1.15; }
    .parking-empty-step p { margin-top: 5px; color: var(--muted); font-size: 13px; line-height: 1.42; }
    .parking-empty-note { grid-column: 1 / -1; display: grid; grid-template-columns: 40px minmax(0,1fr) auto; gap: 14px; align-items: center; border: 1px solid rgba(47,107,74,.18); border-radius: var(--radius-sm); background: rgba(47,107,74,.06); padding: 13px 15px; color: var(--muted); }
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
    .document-sections { display: grid; gap: 14px; }
    .filter-form.document-filter { grid-template-columns: minmax(280px,1fr) minmax(150px,.28fr) auto; align-items: end; margin-bottom: 12px; }
    .filter-form.document-filter button { width: auto; margin-top: 0; min-height: 42px; }
    .document-section { border-top: 1px solid var(--line); padding-top: 14px; display: grid; gap: 10px; }
    .document-section:first-child { border-top: 0; padding-top: 0; }
    .document-section h3 { display: flex; align-items: center; justify-content: space-between; gap: 10px; font-size: 18px; }
    .document-section h3 .section-count { font-family: var(--font-sans); font-size: 11px; font-weight: 900; color: var(--gold-ink); background: rgba(200,153,63,.12); border: 1px solid rgba(200,153,63,.24); border-radius: var(--radius-pill); padding: 3px 8px; white-space: nowrap; }
    .document-section-empty { padding-top: 10px; gap: 6px; }
    .document-section-empty h3 { color: var(--muted); font-size: 15px; }
    .document-list { display: grid; gap: 8px; }
    .document-row { display: grid; grid-template-columns: 38px minmax(0,1fr); gap: 8px 12px; align-items: start; border: 1px solid var(--line); border-radius: var(--radius-xs); background: #fffefb; padding: 10px 12px; }
    .document-row::before { content: ""; width: 34px; height: 40px; border: 1px solid rgba(200,153,63,.28); border-radius: 8px; background: linear-gradient(135deg, rgba(200,153,63,.18), rgba(200,153,63,.18) 34%, transparent 35%), rgba(200,153,63,.08); box-shadow: inset 0 -10px 0 rgba(255,254,251,.48); }
    .document-row strong { display: block; font-size: 15px; line-height: 1.25; overflow-wrap: anywhere; }
    .document-meta { margin-top: 5px; min-width: 0; display: flex; flex-wrap: wrap; gap: 6px 7px; align-items: center; color: var(--muted); font-size: 12.5px; }
    .document-file { min-width: 0; max-width: min(260px,100%); color: var(--soft); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
    .document-side { grid-column: 2; display: flex; align-items: center; justify-content: flex-start; gap: 7px; flex-wrap: wrap; }
    .document-versions { grid-column: 2 / -1; border-top: 1px solid var(--line); padding-top: 10px; }
    .document-versions summary { cursor: pointer; font-weight: 800; color: var(--gold-ink); }
    .version-list { margin-top: 8px; display: grid; gap: 7px; }
    .version-row { display: flex; align-items: center; justify-content: space-between; gap: 10px; flex-wrap: wrap; color: var(--muted); font-size: 12.5px; }
    .vote-list { display: grid; gap: 12px; }
    .vote-card { border: 1px solid var(--line); border-radius: var(--radius-sm); background: #fffefb; padding: 16px; display: grid; gap: 13px; }
    .vote-card form { display: grid; gap: 14px; }
    .vote-card-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 14px; }
    .vote-card-head > .pill { justify-self: end; }
    .vote-card h3 { font-size: 20px; overflow-wrap: anywhere; }
    .vote-meta { display: flex; flex-wrap: wrap; align-items: center; gap: 7px; color: var(--soft); font-size: 12.5px; font-weight: 700; }
    .vote-options { display: grid; gap: 8px; }
    .vote-option { min-height: 42px; display: grid; grid-template-columns: auto minmax(0,1fr); gap: 10px; align-items: center; border: 1px solid var(--line); border-radius: var(--radius-xs); background: var(--panel-soft); padding: 10px 12px; color: var(--ink); font-size: 14px; font-weight: 700; }
    .vote-option input { width: auto; min-height: 0; }
    .vote-actions { display: flex; align-items: center; justify-content: space-between; gap: 12px; flex-wrap: wrap; margin-top: 4px; padding-top: 13px; border-top: 1px solid var(--line); }
    .vote-result { display: grid; gap: 6px; }
    .vote-result-row { display: grid; grid-template-columns: minmax(110px,.45fr) minmax(120px,1fr) auto; gap: 10px; align-items: center; color: var(--muted); font-size: 13px; }
    .vote-result-row strong { color: var(--ink); overflow-wrap: anywhere; }
    .vote-bar { height: 9px; border-radius: var(--radius-pill); background: #ece5d6; overflow: hidden; }
    .vote-bar span { display: block; height: 100%; min-width: 2px; border-radius: inherit; background: var(--gold); }
    .vote-manage-list { display: grid; gap: 8px; }
    .vote-manage-row { display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 12px; align-items: center; border-bottom: 1px solid var(--line); padding: 10px 0; }
    .vote-manage-row:last-child { border-bottom: 0; }
    .vote-manage-actions { display: flex; gap: 8px; flex-wrap: wrap; justify-content: flex-end; }
    .vote-manage-actions form { margin: 0; }
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
    @media (max-width: 1120px) {
      .app-shell { grid-template-columns: 1fr; }
      .sidebar { position: relative; height: auto; padding: 16px; }
      .side-nav { grid-template-columns: repeat(auto-fit,minmax(170px,1fr)); }
      .side-foot { margin-top: 4px; grid-template-columns: 1fr auto; align-items: center; }
      .logout-form { justify-self: end; min-width: 160px; }
      .home-grid, .metric-grid, .issue-layout { grid-template-columns: 1fr; }
      .handover-detail-grid { grid-template-columns: repeat(2,minmax(0,1fr)); }
      .handover-head { grid-template-columns: 1fr; }
      .handover-actions { justify-content: flex-start; }
      .parking-page-head, .parking-guide, .parking-workspace, .parking-detail-actions { grid-template-columns: 1fr; }
      .pay-fields { grid-template-columns: 1fr; }
      .parking-primary-actions { justify-content: flex-start; }
      .parking-guide-status { justify-content: flex-start; }
      .parking-empty { grid-template-columns: minmax(0,1fr); align-items: start; }
      .parking-empty-art { width: min(260px,72vw); justify-self: start; }
      .parking-empty-actions { justify-content: flex-start; }
      .parking-empty-steps { grid-template-columns: 1fr; gap: 14px; }
      .parking-empty-step, .parking-empty-step:first-child, .parking-empty-step:last-child { padding: 0; border-left: 0; }
      .parking-empty-note { grid-template-columns: 40px minmax(0,1fr); }
      .parking-empty-note .button { grid-column: 1 / -1; justify-self: start; }
      .parking-detail-stack { position: static; }
      .digest-panel .quick-list { grid-template-columns: 1fr; }
    }
	    @media (max-width: 680px) {
	      .app-shell { display: block; }
	      .sidebar { position: sticky; top: 0; z-index: 50; display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 10px 12px; height: auto; padding: 10px 14px; box-shadow: 0 10px 28px rgba(23,32,25,.18); }
	      .side-brand { grid-template-columns: 42px minmax(0,1fr); gap: 10px; align-items: center; padding: 0; min-width: 0; }
	      .side-mark { width: 42px; height: 42px; }
	      .side-mark svg { width: 34px; height: 30px; }
	      .side-title { font-size: 15.5px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
	      .side-sub { margin-top: 3px; font-size: 12px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
	      .side-code { margin-top: 4px; padding: 1px 7px; font-size: 9.5px; }
	      .nav-toggle { position: absolute; inline-size: 1px; block-size: 1px; opacity: 0; pointer-events: none; }
	      .mobile-menu-toggle { min-height: 38px; align-self: center; display: inline-flex; align-items: center; justify-content: center; gap: 7px; border: 1px solid rgba(255,255,255,.24); border-radius: var(--radius-xs); padding: 8px 10px; color: rgba(255,255,255,.92); background: rgba(255,255,255,.07); font-size: 12px; font-weight: 850; letter-spacing: .01em; cursor: pointer; }
	      .mobile-menu-toggle::before { content: ""; width: 14px; height: 10px; border-top: 2px solid currentColor; border-bottom: 2px solid currentColor; box-shadow: 0 4px 0 currentColor inset; }
	      .nav-toggle:focus-visible + .mobile-menu-toggle { outline: 3px solid var(--gold); outline-offset: 3px; }
	      .nav-toggle:checked + .mobile-menu-toggle { border-color: rgba(231,197,116,.62); color: #fff; background: rgba(231,197,116,.13); }
	      .side-nav, .side-foot { grid-column: 1 / -1; display: none; }
	      .nav-toggle:checked ~ .side-nav, .nav-toggle:checked ~ .side-foot { display: grid; }
	      .side-nav { grid-template-columns: repeat(2,minmax(0,1fr)); gap: 6px; padding-top: 8px; border-top: 1px solid rgba(255,255,255,.14); }
	      .nav-item { min-height: 40px; padding: 8px 9px; gap: 8px; font-size: 12.5px; }
	      .nav-item.active::before { left: -14px; width: 3px; }
	      .nav-icon { width: 18px; height: 18px; }
	      .nav-icon svg { width: 17px; height: 17px; }
	      .nav-badge { min-width: 20px; height: 18px; padding: 0 6px; font-size: 10px; }
	      .side-foot { margin-top: 2px; grid-template-columns: minmax(0,1fr) auto; align-items: center; gap: 9px 10px; padding: 10px 0 0; }
	      .side-user { grid-template-columns: 34px minmax(0,1fr); gap: 9px; min-width: 0; }
	      .avatar { width: 34px; height: 34px; font-size: 12px; }
	      .side-user strong { font-size: 13px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
	      .side-user span { font-size: 12px; }
	      .side-version { justify-self: end; }
	      .logout-form { grid-column: 1 / -1; justify-self: stretch; min-width: 0; }
	      .logout-button { min-height: 38px; }
      .content-top { height: auto; min-height: 58px; flex-direction: column; align-items: flex-start; padding-top: 12px; padding-bottom: 12px; }
      .home-hero { min-height: 224px; align-items: flex-end; padding: 96px 18px 30px; }
      .home-hero::before { inset: 0; background-position: center top; }
      .home-hero::after { background: linear-gradient(180deg, rgba(247,243,234,.28) 0%, rgba(247,243,234,.86) 50%, rgba(247,243,234,.98) 100%); }
      .home-hero h1 { font-size: clamp(40px,12vw,52px); }
      .home-hero p { font-size: 15px; }
	      .page, .page.wide { width: 100vw; max-width: 100vw; padding-left: 18px; padding-right: 18px; overflow-x: clip; }
	      .page > *, .panel, .audit-timeline, .filter-form.audit-filter { min-width: 0; max-width: 100%; }
	      h1 { font-size: clamp(34px,10.5vw,42px); }
	      .metric-grid { grid-template-columns: 1fr; }
	      .payment-fields, .parking-breakdown { grid-template-columns: 1fr; }
	      .handover-detail-grid, .handover-confirm-summary { grid-template-columns: 1fr; }
	      .parking-page-head, .parking-page-head > *, .parking-primary-actions, .parking-page .panel { min-width: 0; max-width: 100%; }
	      .parking-primary-actions { width: 100%; display: grid; grid-template-columns: 1fr; justify-content: stretch; }
	      .parking-primary-actions .button, .parking-primary-actions form, .parking-primary-actions form button { width: 100%; }
	      .parking-more { width: 100%; }
	      .parking-more-menu { left: 0; right: auto; width: min(270px, 100%); }
	      .parking-month-row { grid-template-columns: 40px minmax(0,1fr) auto; }
	      .parking-month-row .amount { grid-column: 2; }
	      .parking-month-row .pill, .parking-month-row .parking-queue-action { justify-self: start; }
      .parking-stepper { grid-template-columns: 1fr; }
      .parking-step { justify-items: start; grid-template-columns: 32px minmax(0,1fr); text-align: left; }
      .parking-step-number { width: 32px; height: 32px; }
      .parking-step::before { display: none; }
      .issue-stats { grid-template-columns: repeat(2,minmax(0,1fr)); }
      .issue-create-panel > summary { grid-template-columns: 1fr; }
      .issue-create-panel > summary::after { justify-self: start; }
      .issue-preview-card { grid-template-columns: 1fr; }
      .issue-preview-card .chips { justify-self: start; justify-content: flex-start; }
      .issue-form { grid-template-columns: 1fr; }
      .issue-board-filter { grid-template-columns: 1fr; }
      .issue-board-filter label, .issue-board-filter label.assignee, .issue-board-filter label.sort, .issue-board-filter .board-filter-actions { grid-column: 1 / -1; }
	      .quick-row { grid-template-columns: 28px minmax(0,1fr); }
	      .quick-row .entry-actions { grid-column: 2; justify-self: stretch; justify-content: flex-start; margin-top: 5px; }
	      .quick-row .entry-actions .button, .quick-row .entry-actions form { flex: 1 1 112px; min-width: 0; }
	      .quick-row .entry-actions form .button { width: 100%; }
	      .filter-form.audit-filter { grid-template-columns: 1fr; align-items: stretch; }
      .filter-form.audit-filter button, .filter-form.audit-filter .button { width: 100%; min-height: 42px; }
      .audit-filter-panel > summary { grid-template-columns: 1fr auto; align-items: center; }
      .audit-filter-panel > summary strong { grid-column: 1 / -1; font-size: 13px; }
      .audit-timeline { width: 100%; }
      .audit-day { padding: 12px 14px 9px; }
      .audit-row { grid-template-columns: 56px 14px minmax(0,1fr); gap: 10px; padding: 14px; }
      .audit-time { text-align: left; }
      .audit-time strong { font-size: 16px; }
      .audit-time span { font-size: 10.5px; overflow-wrap: anywhere; }
      .audit-marker { width: 10px; height: 10px; margin-top: 5px; box-shadow: 0 0 0 4px rgba(200,153,63,.13); }
      .audit-row-head { gap: 8px; }
      .audit-action strong { font-size: 15px; }
      .audit-meta { gap: 7px; }
	      .document-row { grid-template-columns: 34px minmax(0,1fr); gap: 8px 10px; padding: 12px; }
	      .document-row::before { width: 32px; height: 38px; }
	      .document-row > div:first-of-type { min-width: 0; }
	      .document-side { gap: 6px; margin-top: 4px; }
	      .document-side .pill { order: -1; }
	      .document-file { max-width: 100%; }
	      .document-versions { grid-column: 2; }
	      .vote-card-head, .vote-actions, .vote-manage-row { display: grid; grid-template-columns: 1fr; }
	      .vote-card-head > .pill { justify-self: start; }
	      .vote-result-row { grid-template-columns: 1fr; }
	      .vote-manage-actions { justify-content: flex-start; }
	      .quick-row .pill { grid-column: 2; justify-self: start; }
	      .filter-form { grid-template-columns: 1fr; }
	      .filter-form.document-filter { grid-template-columns: 1fr; }
      .dialog-grid { grid-template-columns: 1fr; }
      .release-history { grid-template-columns: 1fr; max-height: 76vh; }
      .release-rail { grid-template-columns: repeat(auto-fit,minmax(132px,1fr)); border-right: 0; border-bottom: 1px solid var(--line); padding-right: 0; padding-bottom: 12px; }
      .release-item { grid-template-columns: 1fr; gap: 4px; }
      .empty-state { grid-template-columns: 1fr; }
      .home-status-head { display: grid; gap: 12px; }
      .home-status-grid { grid-template-columns: 1fr; }
      .status-card { grid-template-columns: 54px minmax(0,1fr); }
      .home-list-row, .parking-summary { grid-template-columns: 1fr; }
      .home-event-row { grid-template-columns: 52px minmax(0,1fr); }
      .document-dashboard-row { grid-template-columns: 1fr; gap: 5px; align-items: start; }
      .document-dashboard-row > span:not(.document-file-title):not(.document-download) { white-space: normal; }
      .document-download { justify-self: start; }
      .parking-summary .pill, .home-list-row .pill { justify-self: start; }
      .parking-check { justify-self: start; }
      .home-list-row .quick-arrow { display: none; }
      .quick-arrow { display: none; }
    }
  </style>
{{end}}

{{define "sidebar"}}
  <aside class="sidebar" aria-label="Portalnavigation">
    <div class="side-brand">
	      <a class="side-mark" href="/app" aria-label="{{if .IsServiceProvider}}Anliegen{{else}}Hausüberblick{{end}}">
        {{template "tenantBrandMark" .}}
      </a>
      <div>
        <a class="side-title" href="/app">{{.Tenant.Name}}</a>
        <span class="side-sub">{{.Tenant.Address}}</span>
        {{if .Tenant.BrandAbbreviation}}<span class="side-code">{{.Tenant.BrandAbbreviation}}</span>{{end}}
      </div>
	    </div>
	    <input class="nav-toggle" id="portal-nav-toggle" type="checkbox" aria-label="Navigation anzeigen">
	    <label class="mobile-menu-toggle" for="portal-nav-toggle">Menü</label>
	    <nav class="side-nav">
	      {{if .CanUseResidentAreas}}
	      <a class="nav-item {{if eq .ActivePage "home"}}active{{end}}" href="/app"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg></span><span class="nav-label">Hausüberblick</span></a>
	      <a class="nav-item {{if eq .ActivePage "announcements"}}active{{end}}" href="/app/announcements"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M4 5h16v13H7l-3 3z"/><path d="M8 9h8M8 13h6"/></svg></span><span class="nav-label">Aushang</span>{{if .HasUnreadAnnouncements}}<span class="nav-badge">{{.UnreadAnnouncements}}</span>{{end}}</a>
	      <a class="nav-item {{if eq .ActivePage "events"}}active{{end}}" href="/app/events"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M7 3v4M17 3v4"/><path d="M4.5 6h15v14h-15z"/><path d="M4.5 10h15"/><path d="M8 14h.01M12 14h.01M16 14h.01"/></svg></span><span class="nav-label">Termine</span></a>
	      <a class="nav-item {{if eq .ActivePage "contacts"}}active{{end}}" href="/app/kontakte"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M16 4h2.5A1.5 1.5 0 0 1 20 5.5v13A1.5 1.5 0 0 1 18.5 20h-13A1.5 1.5 0 0 1 4 18.5v-13A1.5 1.5 0 0 1 5.5 4H8"/><path d="M8.5 3.5h7v4h-7z"/><path d="M9 13a3 3 0 1 0 6 0"/><path d="M7.5 18a4.5 4.5 0 0 1 9 0"/></svg></span><span class="nav-label">Kontakte</span></a>
	      {{if .CanSeeParking}}<a class="nav-item {{if eq .ActivePage "parking"}}active{{end}}" href="/app/parking"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M5 16h14"/><path d="m7 16 1.5-5h7L17 16"/><path d="M7 16v3M17 16v3"/><path d="M7 19h1M16 19h1"/></svg></span>Parkplatznutzung</a>{{end}}
	      <a class="nav-item {{if eq .ActivePage "documents"}}active{{end}}" href="/app/dokumente"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M7 3h7l3 3v15H7z"/><path d="M14 3v4h4"/><path d="M9 13h6M9 17h6"/></svg></span>Dokumente</a>
	      {{if .CanManageHandovers}}<a class="nav-item {{if eq .ActivePage "handovers"}}active{{end}}" href="/app/uebergaben"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M7 4h10v16H7z"/><path d="M9.5 8h5M9.5 12h4"/><path d="m9.5 16 1.5 1.5 3.5-4"/></svg></span>Übergaben</a>{{end}}
	      {{end}}
	      <a class="nav-item {{if eq .ActivePage "issues"}}active{{end}}" href="/app/anliegen"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M5 18.5V6.5A2.5 2.5 0 0 1 7.5 4h9A2.5 2.5 0 0 1 19 6.5v6A2.5 2.5 0 0 1 16.5 15H10l-5 3.5z"/></svg></span>Anliegen{{if .HasOpenIssues}}<span class="nav-badge">{{.OpenIssues}}</span>{{end}}</a>
	      {{if .CanUseResidentAreas}}
	      <a class="nav-item {{if eq .ActivePage "abstimmungen"}}active{{end}}" href="/app/abstimmungen"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M5 19V9M12 19V5M19 19v-7"/><path d="M3.5 19h17"/></svg></span>Abstimmungen</a>
	      {{if .CanManageUsers}}<a class="nav-item {{if eq .ActivePage "users"}}active{{end}}" href="/app/settings/users"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M8.5 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6z"/><path d="M3.5 20a5 5 0 0 1 10 0"/><path d="M16 11.5a2.5 2.5 0 1 0 0-5"/><path d="M17 15a4 4 0 0 1 3.5 4"/></svg></span>Benutzer &amp; Rechte</a>{{end}}
	      {{if .CanViewAudit}}<a class="nav-item {{if eq .ActivePage "audit"}}active{{end}}" href="/app/audit"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M5 4h14v16H5z"/><path d="M8 8h8M8 12h8M8 16h5"/></svg></span>Audit-Log</a>{{end}}
	      <a class="nav-item {{if eq .ActivePage "settings"}}active{{end}}" href="/app/settings"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M12 8.5a3.5 3.5 0 1 0 0 7 3.5 3.5 0 0 0 0-7z"/><path d="M19 12a7 7 0 0 0-.1-1l2-1.5-2-3.5-2.4 1a7 7 0 0 0-1.8-1L14.4 3h-4.8L9.3 6a7 7 0 0 0-1.8 1l-2.4-1-2 3.5 2 1.5A7 7 0 0 0 5 12a7 7 0 0 0 .1 1l-2 1.5 2 3.5 2.4-1a7 7 0 0 0 1.8 1l.3 3h4.8l.3-3a7 7 0 0 0 1.8-1l2.4 1 2-3.5-2-1.5a7 7 0 0 0 .1-1z"/></svg></span>Einstellungen</a>
	      {{end}}
	    </nav>
    <div class="side-foot">
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
</head>
<body>
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

{{define "portal"}}
{{template "appOpen" .}}
    <main class="app-main">
      <div class="content-top"><span class="crumb"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg>Hausüberblick</span></div>
      <section class="home-hero">
        <div class="home-hero-copy">
          <h1>Hausüberblick</h1>
          <p>Willkommen zurück, {{.GreetingName}}! Hier finden Sie einen schnellen Überblick über alles Wichtige.</p>
        </div>
      </section>
      <section class="page home-page">
        <div class="home-grid">
          <section class="panel home-status-panel">
            <div class="home-status-head">
              <div class="kicker">Aktuell</div>
              <a class="section-link" href="/app/announcements">Archiv öffnen</a>
            </div>
            <div class="home-status-grid">
              <a class="status-card status-announcements" href="/app/announcements">
                <span class="status-icon"><svg viewBox="0 0 24 24"><path d="M4 5h16v13H7l-3 3z"/><path d="M8 9h8M8 13h6"/></svg></span>
                <span><span class="status-label">Aushänge</span><strong>{{.UnreadAnnouncements}}</strong><span>neu</span></span>
              </a>
              <a class="status-card status-events" href="/app/events">
                <span class="status-icon"><svg viewBox="0 0 24 24"><path d="M7 3v4M17 3v4"/><path d="M4.5 6h15v14h-15z"/><path d="M4.5 10h15"/></svg></span>
                <span><span class="status-label">Termine</span><strong>{{.EventCount}}</strong><span>anstehend</span></span>
              </a>
              <a class="status-card status-issues" href="{{.IssueSummaryURL}}">
                <span class="status-icon"><svg viewBox="0 0 24 24"><path d="M5 18.5V6.5A2.5 2.5 0 0 1 7.5 4h9A2.5 2.5 0 0 1 19 6.5v6A2.5 2.5 0 0 1 16.5 15H10l-5 3.5z"/></svg></span>
                <span><span class="status-label">Anliegen</span><strong>{{.IssueCount}}</strong><span>offen</span></span>
              </a>
              {{if .CanSeeParking}}
                <a class="status-card status-parking" href="/app/parking">
                  <span class="status-icon"><svg viewBox="0 0 24 24"><path d="M5 16h14"/><path d="m7 16 1.5-5h7L17 16"/><path d="M7 16v3M17 16v3"/><path d="M7 19h1M16 19h1"/></svg></span>
                  <span><span class="status-label">Parkplatz</span><strong>{{.ParkingStatusValue}}</strong><span>{{.ParkingStatusDetail}}</span></span>
                </a>
              {{end}}
            </div>
          </section>

          <div class="home-stack">
            <section class="panel">
              <div class="section-head">
                <div class="kicker">Aushang</div>
                {{if .HasUnreadAnnouncements}}<span class="pill unread">{{.UnreadAnnouncements}} neu</span>{{end}}
                <a class="section-link" href="/app/announcements">Archiv öffnen</a>
              </div>
              {{if .HasAnnouncements}}
                <div class="entries">
                  {{range .Announcements}}
                    <article class="entry">
                      <div class="entry-head">
                        <div>
                          <h3><a href="/app/announcements">{{.Title}}</a></h3>
                          <div class="entry-meta">
                            <span class="pill {{.CategoryClass}}">{{.Category}}</span>
                            {{if .Unread}}<span class="pill unread">neu</span>{{end}}
                            {{if .Pinned}}<span class="pill">Fixiert</span>{{end}}
                            <span>{{.PublishedAt}}</span>
                          </div>
                        </div>
                      </div>
                      <div class="entry-body">{{.BodyHTML}}</div>
                    </article>
                  {{end}}
                </div>
              {{else}}
                {{template "emptyState" .AnnouncementsEmpty}}
              {{end}}
            </section>

            <section class="panel">
              <div class="section-head">
                <div class="kicker">Offene Anliegen</div>
                <a class="section-link" href="{{.IssueSummaryURL}}">Anliegen öffnen</a>
              </div>
              {{if .HasDashboardIssues}}
                <div class="home-list">
                  {{range .DashboardIssues}}
                    <a class="home-list-row" href="{{$.IssueSummaryURL}}">
                      <span class="home-list-icon"><svg viewBox="0 0 24 24"><path d="M5 18.5V6.5A2.5 2.5 0 0 1 7.5 4h9A2.5 2.5 0 0 1 19 6.5v6A2.5 2.5 0 0 1 16.5 15H10l-5 3.5z"/></svg></span>
                      <span><strong>{{.Title}}</strong><span>{{.Priority}} · {{.CreatedAt}}{{if .Location}} · {{.Location}}{{end}}</span></span>
                      <span class="pill {{.StatusClass}}">{{.Status}}</span>
                    </a>
                  {{end}}
                </div>
              {{else}}
                {{template "emptyState" .DashboardIssuesEmpty}}
              {{end}}
            </section>
	          </div>

	          <div class="home-stack">
	            <section class="panel">
              <div class="section-head">
                <div class="kicker">Nächste Termine</div>
                <a class="section-link" href="/app/events">Termine öffnen</a>
              </div>
              {{if .HasEvents}}
                <div class="home-events-list">
                  {{range .Events}}
                    <a class="home-event-row" href="/app/events">
                      <span class="home-date"><strong>{{.DateBadgeDay}}</strong><span>{{.DateBadgeMonth}}</span></span>
                      <span class="home-event-copy">
                        <strong>{{.Title}}</strong>
                        <span>{{.StartsAt}}{{if .HasLocation}} · {{.Location}}{{end}}</span>
                      </span>
                      <span class="quick-arrow">›</span>
                    </a>
                  {{end}}
                </div>
              {{else}}
	                {{template "emptyState" .EventsEmpty}}
	              {{end}}
	            </section>

	            {{if .HasUnitPaymentStatuses}}
	              <section class="panel">
	                <div class="section-head">
	                  <div class="kicker">Zahlungsstatus</div>
	                </div>
	                <div class="home-list">
	                  {{range .UnitPaymentStatuses}}
	                    <div class="home-list-row">
	                      <span class="home-list-icon"><svg viewBox="0 0 24 24"><path d="M4 7h16v10H4z"/><path d="M7 10h3"/><path d="M14 14h3"/></svg></span>
	                      <span><strong>{{.UnitLabel}}</strong><span>{{.UnitTypeLabel}}{{if .Relation}} · {{.Relation}}{{end}}{{if .HasUpdatedAt}} · {{.UpdatedAt}}{{end}}</span></span>
	                      <span class="pill {{.StatusClass}}">{{.Status}}</span>
	                    </div>
	                  {{end}}
	                </div>
	              </section>
	            {{end}}

	            {{if .CanSeeParking}}
	              <section class="panel">
                <div class="section-head">
                  <div class="kicker">Parkplatznutzung</div>
                  <a class="section-link" href="/app/parking">Parkplatz öffnen</a>
                </div>
                <div class="parking-summary">
                  <span class="status-icon"><svg viewBox="0 0 24 24"><path d="M5 16h14"/><path d="m7 16 1.5-5h7L17 16"/><path d="M7 16v3M17 16v3"/><path d="M7 19h1M16 19h1"/></svg></span>
                  <span><strong>{{.ParkingStatusTitle}}</strong><p>{{.ParkingSummaryDetail}}</p></span>
                  {{if eq .ParkingStatusClass "ok"}}
                    <span class="parking-check"><svg viewBox="0 0 24 24"><path d="m5 13 4 4L19 7"/></svg></span>
                  {{else}}
                    <span class="pill {{.ParkingStatusClass}}">{{.ParkingStatusTitle}}</span>
                  {{end}}
                </div>
                <div class="home-card-actions">
                  <a class="button small" href="/app/parking">Nutzungsübersicht</a>
                  {{if .IsAdmin}}<a class="button small" href="/app/parking/settings">Abrechnungen</a>{{end}}
                </div>
              </section>
            {{end}}
          </div>

          <section class="panel home-full">
	              <div class="section-head">
	                <div class="kicker">Dokumente</div>
	                <a class="section-link" href="/app/dokumente">Dokumente öffnen</a>
	              </div>
              {{if .HasDashboardDocuments}}
                <div class="document-dashboard-list">
                  {{range .DashboardDocuments}}
                    <a class="document-dashboard-row" href="{{.DownloadURL}}">
                      <span class="document-file-title"><svg viewBox="0 0 24 24"><path d="M7 3h7l3 3v15H7z"/><path d="M14 3v4h4"/><path d="M9 13h6M9 17h6"/></svg><strong>{{.Title}}</strong></span>
                      <span>{{.FileKind}}</span>
                      <span>{{.Size}}</span>
                      <span>{{.UploadedDate}}</span>
                      <span class="document-download"><svg viewBox="0 0 24 24"><path d="M12 3v12"/><path d="m7 10 5 5 5-5"/><path d="M5 20h14"/></svg></span>
                    </a>
                  {{end}}
                </div>
              {{else}}
	                {{template "emptyState" .DashboardDocumentsEmpty}}
	              {{end}}
	            </section>
        </div>
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "contacts"}}
{{template "appOpen" .}}
    <style>
      .contacts .contact-grid { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 22px; align-items: start; }
      .contacts .contact-section { display: grid; gap: 14px; }
      .contacts .contact-list { display: grid; gap: 10px; }
      .contacts .contact-card { border: 1px solid var(--line); border-radius: 8px; padding: 14px; background: var(--panel-soft); display: grid; gap: 10px; }
      .contacts .contact-head { display: flex; justify-content: space-between; gap: 12px; align-items: flex-start; }
      .contacts .contact-head strong { font-family: Spectral, serif; font-size: 20px; line-height: 1.12; overflow-wrap: anywhere; }
      .contacts .contact-lines { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; color: var(--muted); font-size: 13.5px; font-weight: 700; }
      .contacts .contact-lines a { color: inherit; text-decoration: none; border-bottom: 1px solid rgba(200,153,63,.5); }
      .contacts .contact-lines a:hover { color: var(--gold-ink); }
      .contacts .directory-panel, .contacts .managed-panel { grid-column: 1 / -1; }
      .contacts .contact-form { display: grid; grid-template-columns: repeat(12,minmax(0,1fr)); gap: 10px; align-items: end; }
      .contacts .contact-form .f-kind { grid-column: span 3; }
      .contacts .contact-form .f-name, .contacts .contact-form .f-company { grid-column: span 4; }
      .contacts .contact-form .f-email, .contacts .contact-form .f-phone { grid-column: span 4; }
      .contacts .contact-form .f-notes { grid-column: span 8; }
      .contacts .contact-form .f-actions { grid-column: span 4; }
      .contacts .contact-actions { display: flex; gap: 8px; flex-wrap: wrap; justify-content: flex-end; align-items: center; }
      .contacts .contact-card.inactive { opacity: .68; }
      .contacts .contact-card.inactive .pill { background: #f1ede3; color: #777166; }
      .contacts .dialog .contact-form { grid-template-columns: repeat(2,minmax(0,1fr)); }
      .contacts .dialog .contact-form > * { grid-column: auto; }
      .contacts .dialog .contact-form .f-notes, .contacts .dialog .contact-form .f-actions { grid-column: 1 / -1; }
      @media (max-width: 900px) { .contacts .contact-grid { grid-template-columns: 1fr; } .contacts .directory-panel, .contacts .managed-panel { grid-column: 1; } }
      @media (max-width: 720px) { .contacts .contact-form, .contacts .dialog .contact-form { grid-template-columns: 1fr; } .contacts .contact-form > *, .contacts .dialog .contact-form > * { grid-column: 1 / -1 !important; } .contacts .contact-actions { justify-content: flex-start; } }
    </style>
    <main class="app-main contacts">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M16 4h2.5A1.5 1.5 0 0 1 20 5.5v13A1.5 1.5 0 0 1 18.5 20h-13A1.5 1.5 0 0 1 4 18.5v-13A1.5 1.5 0 0 1 5.5 4H8"/><path d="M8.5 3.5h7v4h-7z"/><path d="M9 13a3 3 0 1 0 6 0"/><path d="M7.5 18a4.5 4.5 0 0 1 9 0"/></svg><span>/</span><span>Kontakte</span></span>
      </div>
      <section class="page wide">
        <div>
          <h1>Kontakte</h1>
          <p class="lede">Verwaltung, Notdienst, Hausmeister, Beirat und freigegebene Kontakte für {{.Tenant.Address}}.</p>
        </div>
        {{if .ContactMsg}}<p class="flash {{if .ContactOK}}ok{{end}}">{{.ContactMsg}}</p>{{end}}
        <div class="contact-grid">
          <section class="panel contact-section managed-panel">
            <div class="section-head">
              <div>
                <div class="kicker">Adressbuch</div>
	                <h2>{{if .ServiceProviderAccessEnabled}}Dienstleister &amp; wichtige Kontakte{{else}}Wichtige Kontakte{{end}}</h2>
	                <p class="muted">{{if .ServiceProviderAccessEnabled}}Wiederkehrende Kontakte pro Hausverwaltung. Dienstleister mit E-Mail können im Anliegen direkt ausgewählt werden.{{else}}Hausmeister, Notdienste und weitere wiederkehrende Kontakte. Datenschutzprüfung offen: Dienstleister bleiben gesperrt.{{end}}</p>
              </div>
            </div>
            {{if .CanManageContacts}}
              <form class="contact-form" method="post" action="/app/kontakte">
                <input type="hidden" name="active" value="true">
                <label class="f-kind">Art<select name="kind" required>{{range .ContactKindOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}</select></label>
                <label class="f-name">Name<input name="name" maxlength="120" placeholder="Ansprechperson"></label>
                <label class="f-company">Firma<input name="company" maxlength="140" placeholder="Firma oder Organisation"></label>
                <label class="f-email">E-Mail<input type="email" name="email" placeholder="kontakt@example.com"></label>
                <label class="f-phone">Telefon<input name="phone" maxlength="80" placeholder="+43 ..."></label>
                <label class="f-notes">Notiz<input name="notes" maxlength="300" placeholder="z. B. Lift, Elektrik, 24h"></label>
                <div class="f-actions"><button class="button primary" type="submit">Kontakt speichern</button></div>
              </form>
            {{end}}
            {{if .HasManagedContacts}}
              <div class="contact-list">
                {{range .ManagedContacts}}
                  <article class="contact-card {{if not .Active}}inactive{{end}}">
                    <div class="contact-head"><strong>{{.DisplayName}}</strong><span class="pill">{{.Kind}}</span></div>
                    {{if .Description}}<p class="muted">{{.Description}}</p>{{end}}
                    <div class="contact-lines">{{if .HasEmail}}<a href="mailto:{{.Email}}">{{.Email}}</a>{{end}}{{if .HasPhone}}<a href="tel:{{.Phone}}">{{.Phone}}</a>{{end}}<span>{{.StatusLabel}}</span></div>
                    {{if $.CanManageContacts}}
                      <div class="contact-actions">
	                        {{if or $.ServiceProviderAccessEnabled (ne .Kind "Dienstleister")}}<button class="button small" type="button" data-dialog="{{.EditDialogID}}" aria-haspopup="dialog" aria-controls="{{.EditDialogID}}">Bearbeiten</button>{{end}}
	                        {{if .Active}}<form method="post" action="/app/kontakte/delete" data-confirm="{{.DeleteConfirmLabel}}"><input type="hidden" name="id" value="{{.ID}}"><button class="button small" type="submit">Deaktivieren</button></form>{{end}}
	                      </div>
	                      {{if or $.ServiceProviderAccessEnabled (ne .Kind "Dienstleister")}}
	                      <dialog id="{{.EditDialogID}}" class="dialog" aria-labelledby="{{.EditDialogID}}-title">
                        <form method="post" action="/app/kontakte">
                          <input type="hidden" name="id" value="{{.ID}}">
                          <input type="hidden" name="active" value="{{if .Active}}true{{else}}false{{end}}">
                          <div class="dialog-head">
                            <h2 id="{{.EditDialogID}}-title">Kontakt bearbeiten</h2>
                            <button class="dialog-close" type="button" data-close-dialog aria-label="Schließen">&times;</button>
                          </div>
                          <div class="dialog-body">
                            <div class="contact-form">
                              <label>Art<select name="kind" required>{{range .KindOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}</select></label>
                              <label>Name<input name="name" value="{{.Name}}" maxlength="120"></label>
                              <label>Firma<input name="company" value="{{.Company}}" maxlength="140"></label>
                              <label>E-Mail<input type="email" name="email" value="{{.Email}}"></label>
                              <label>Telefon<input name="phone" value="{{.Phone}}" maxlength="80"></label>
                              <label class="f-notes">Notiz<input name="notes" value="{{.Notes}}" maxlength="300"></label>
                              <div class="f-actions"><button class="button primary" type="submit">Speichern</button></div>
                            </div>
                          </div>
	                        </form>
	                      </dialog>
	                      {{end}}
	                    {{end}}
                  </article>
                {{end}}
              </div>
            {{else}}
              {{template "emptyState" .ManagedEmpty}}
            {{end}}
          </section>

          <section class="panel contact-section">
            <div>
              <div class="kicker">Verwaltung</div>
              <h2>Hausverwaltung</h2>
            </div>
            {{if .HasManagerContacts}}
              <div class="contact-list">
                {{range .ManagerContacts}}
                  <article class="contact-card">
                    <div class="contact-head"><strong>{{.Name}}</strong><span class="pill">{{.Role}}</span></div>
                    <p class="muted">{{.Description}}</p>
                    <div class="contact-lines">{{if .HasEmail}}<a href="mailto:{{.Email}}">{{.Email}}</a>{{end}}{{if .HasPhone}}<a href="tel:{{.Phone}}">{{.Phone}}</a>{{end}}</div>
                  </article>
                {{end}}
              </div>
            {{else}}
              {{template "emptyState" .ManagerEmpty}}
            {{end}}
          </section>

          <section class="panel contact-section">
            <div>
              <div class="kicker">Notfall</div>
              <h2>Notdienst &amp; Hausmeister</h2>
            </div>
            {{if .HasEmergencyContacts}}
              <div class="contact-list">
                {{range .EmergencyContacts}}
                  <article class="contact-card">
                    <div class="contact-head"><strong>{{.Name}}</strong><span class="pill">{{.Role}}</span></div>
                    <p class="muted">{{.Description}}</p>
                    <div class="contact-lines">{{if .HasEmail}}<a href="mailto:{{.Email}}">{{.Email}}</a>{{end}}{{if .HasPhone}}<a href="tel:{{.Phone}}">{{.Phone}}</a>{{end}}</div>
                  </article>
                {{end}}
              </div>
            {{else}}
              {{template "emptyState" .EmergencyEmpty}}
            {{end}}
          </section>

          <section class="panel contact-section">
            <div>
              <div class="kicker">Beirat</div>
              <h2>Beirat</h2>
            </div>
            {{if .HasBoardContacts}}
              <div class="contact-list">
                {{range .BoardContacts}}
                  <article class="contact-card">
                    <div class="contact-head"><strong>{{.Name}}</strong><span class="pill">{{.Role}}</span></div>
                    <p class="muted">{{.Description}}</p>
                    <div class="contact-lines">{{if .HasEmail}}<a href="mailto:{{.Email}}">{{.Email}}</a>{{end}}{{if .HasPhone}}<a href="tel:{{.Phone}}">{{.Phone}}</a>{{end}}</div>
                  </article>
                {{end}}
              </div>
            {{else}}
              {{template "emptyState" .BoardEmpty}}
            {{end}}
          </section>

          <section class="panel contact-section directory-panel">
            <div>
              <div class="kicker">Hausgemeinschaft</div>
              <h2>Freigegebene Kontakte</h2>
            </div>
            {{if .HasResidentContacts}}
              <div class="contact-list">
                {{range .ResidentContacts}}
                  <article class="contact-card">
                    <div class="contact-head"><strong>{{.Name}}</strong><span class="pill">{{.Role}}</span></div>
                    <p class="muted">{{.Description}}</p>
                    <div class="contact-lines">{{if .HasEmail}}<a href="mailto:{{.Email}}">{{.Email}}</a>{{end}}{{if .HasPhone}}<a href="tel:{{.Phone}}">{{.Phone}}</a>{{end}}</div>
                  </article>
                {{end}}
              </div>
            {{else}}
              {{template "emptyState" .ResidentEmpty}}
            {{end}}
          </section>
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
      <span class="comment-meta">{{.Author}} · {{.CreatedAt}}</span>
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

{{define "issues"}}
{{template "appOpen" .}}
    <script src="/assets/attachments.js?v={{.AssetVersion}}" defer></script>
    <main class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M5 18.5V6.5A2.5 2.5 0 0 1 7.5 4h9A2.5 2.5 0 0 1 19 6.5v6A2.5 2.5 0 0 1 16.5 15H10l-5 3.5z"/></svg><span>/</span><span>Anliegen</span></span>
        {{if or .HasCalendarFeedURL .CanManageIssues}}<div class="page-actions">
          {{if .HasCalendarFeedURL}}<a class="button" href="{{.CalendarFeedURL}}"><svg viewBox="0 0 24 24" width="17" height="17" aria-hidden="true"><path d="M7 3v4M17 3v4"/><path d="M4.5 6h15v14h-15z"/><path d="M4.5 10h15"/><path d="M8 14h8M8 17h5"/></svg>Kalender abonnieren</a>{{end}}
          {{if .CanManageIssues}}{{if .BoardOnly}}<a class="button" href="/app/anliegen">Zurück zu Anliegen</a>{{else}}<a class="button" href="/app/anliegen/board">Triage-Board</a>{{end}}{{end}}
        </div>{{end}}
      </div>
      <section class="page">
        <div>
          <h1>Anliegen</h1>
          <p class="lede">Mängel, Fragen und Vorschläge direkt an die Verwaltung melden.</p>
        </div>
        {{if .HasServiceProviderContacts}}<datalist id="service-provider-contacts">{{range .ServiceProviderContacts}}<option value="{{.Email}}">{{.Label}}</option>{{end}}</datalist>{{end}}
        {{if not .BoardOnly}}
        <div class="issue-dashboard">
          <div class="issue-stats" aria-label="Anliegen-Überblick">
            <a class="issue-stat" href="#issue-new"><span>Melden</span><strong>+</strong><p>Neues Anliegen mit Fotos oder PDF erfassen.</p></a>
            <a class="issue-stat" href="#issue-own"><span>Meine Anliegen</span><strong>{{.IssueCount}}</strong><p>Status und Rückfragen auf einen Blick.</p></a>
            {{if .CanManageIssues}}<a class="issue-stat" href="/app/anliegen/board"><span>Verwalten</span><strong>{{.OpenIssueCount}}</strong><p>{{.TotalIssueCount}} gesamt, {{.UrgentIssueCount}} dringend.</p></a>{{end}}
          </div>
          <nav class="issue-tabs" aria-label="Anliegen-Bereiche">
            <a class="issue-tab active" href="#issue-new">Anliegen melden</a>
            <a class="issue-tab" href="#issue-own">Meine Anliegen</a>
            {{if .CanManageIssues}}<a class="issue-tab" href="/app/anliegen/board">Verwalten</a>{{end}}
          </nav>
          {{if .CanCreateIssue}}
          <details class="panel issue-create-panel" id="issue-new"{{if .IssueMsg}} open{{end}}>
            <summary>
              <div>
                <div class="kicker">Melden</div>
                <h2>Anliegen melden</h2>
                <p>Kurzer Titel, Ort und Beschreibung reichen für den ersten Schritt.</p>
              </div>
            </summary>
            <div class="issue-create-body">
              {{if .IssueMsg}}<p class="issue-flash{{if .IssueOK}} ok{{else}} warn{{end}}">{{.IssueMsg}}</p>{{end}}
              <form class="issue-form" method="post" action="/app/anliegen" enctype="multipart/form-data">
                <label>Kategorie
                  <select name="category" required>
                    <option value="Reparatur">Reparatur</option>
                    <option value="Frage">Frage</option>
                    <option value="Vorschlag">Vorschlag</option>
                    <option value="Sonstiges">Sonstiges</option>
                  </select>
                </label>
                <label>Ort
                  <select name="location_type" required>
                    <option value="common">Gemeinschaftsbereich</option>
                    <option value="own-unit">Eigene Einheit</option>
                  </select>
                </label>
                <label class="full">Titel
                  <input type="text" name="title" maxlength="140" required placeholder="Kurz zusammenfassen">
                </label>
                <label class="full">Beschreibung
                  <textarea name="body" maxlength="4000" required placeholder="Was ist passiert? Seit wann? Gibt es eine Dringlichkeit?"></textarea>
                </label>
                <label class="full">Details zum Ort
                  <input type="text" name="location_detail" maxlength="160" placeholder="z. B. Stiegenhaus, Garage, Top 3">
                </label>
                <label class="full">Anhänge
                  <span class="file-control"><input type="file" name="attachments" accept="image/jpeg,image/png,image/webp,image/gif,application/pdf" multiple><span>Bis zu 10 Dateien auswählen</span></span>
                  <span class="hint">Optional, Bilddateien oder PDF bis 10 MB je Datei.</span>
                </label>
                <button type="submit">Anliegen senden</button>
              </form>
            </div>
          </details>
          {{end}}
          <section class="panel" id="issue-own">
            <div class="section-head">
              <div>
                <div class="kicker">{{if and .CanManageIssues (not .BoardOnly)}}Meine eigenen Anliegen{{else if eq .Role "Beirat"}}Anliegen im Haus{{else}}Meine letzten Anliegen{{end}}</div>
                <p class="muted">Status und Rückfragen werden hier zusammengeführt.</p>
              </div>
            </div>
            {{if .HasIssues}}
              <div class="issue-list">
                {{range .Issues}}
                  <article class="issue-card" id="issue-{{.ID}}">
                    <div class="issue-meta">
                      <span class="pill {{.StatusClass}}">{{.Status}}</span>
                      <span class="pill">{{.Category}}</span>
                      <span class="pill">{{.Priority}}</span>
                      <span>{{.CreatedAt}}</span>
                    </div>
	                    <h3>{{.Title}}</h3>
		                    <p class="issue-location">{{.Location}}{{if .HasAssignee}} · Zuständig: {{.AssigneeEmail}}{{end}}{{if .HasPhotos}} · {{.PhotoCount}} Foto{{if ne .PhotoCount 1}}s{{end}}{{end}}</p>
	                    {{if .HasServiceAppointment}}<p class="issue-proposal"><strong>Termin:</strong> {{.ServiceAppointment}}</p>{{end}}{{if .HasServiceProposal}}<p class="issue-proposal"><strong>Hinweis:</strong> {{.ServiceProposal}}</p>{{end}}
	                    {{template "issueEstimate" .}}
                    {{template "attachmentStrip" .}}
                    <div class="comment-thread">
                      {{if .HasComments}}
                        {{range .Comments}}{{template "issueComment" .}}{{end}}
                      {{else}}
                        <p class="empty">Noch keine Kommentare.</p>
                      {{end}}
                    </div>
	                    {{if .CanComment}}<form class="comment-form" method="post" action="/app/anliegen/comment" enctype="multipart/form-data">
	                      <input type="hidden" name="id" value="{{.ID}}">
	                      <textarea name="body" maxlength="3000" placeholder="Kommentar oder Ergänzung schreiben" aria-label="Kommentar oder Ergänzung"></textarea>
                      <label class="comment-upload">
                        <span class="file-control"><input type="file" name="attachments" accept="image/jpeg,image/png,image/webp,image/gif,application/pdf" multiple><span>Anhang hinzufügen</span></span>
                      </label>
		                      <button type="submit">Kommentar senden</button>
		                    </form>{{end}}
	                    {{if .CanServiceUpdate}}
	                      <form class="issue-actions" method="post" action="/app/anliegen/workflow">
	                        <input type="hidden" name="id" value="{{.ID}}">
	                        <label>Status
	                          <select name="status">
	                            {{range .ServiceStatusOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}
	                          </select>
	                        </label>
	                        <label class="proposal">Termin (Start)
	                          <input type="datetime-local" name="service_start" value="{{.ServiceStartInput}}">
	                        </label>
	                        <label class="proposal">Termin (Ende, optional)
	                          <input type="datetime-local" name="service_end" value="{{.ServiceEndInput}}">
	                        </label>
	                        <label class="proposal">Hinweis (optional)
	                          <input type="text" name="service_proposal" value="{{.ServiceProposal}}" maxlength="180" placeholder="z. B. Zugang über Hinterhof">
	                        </label>
	                        <button type="submit">Status senden</button>
	                      </form>
	                    {{end}}
	                    {{if or .CanClose .CanReopen}}
                      <div class="issue-actions">
                        {{if .CanClose}}<form method="post" action="/app/anliegen/workflow"><input type="hidden" name="id" value="{{.ID}}"><input type="hidden" name="status" value="Erledigt"><button class="ghost" type="submit">Erledigt melden</button></form>{{end}}
                        {{if .CanReopen}}<form method="post" action="/app/anliegen/workflow"><input type="hidden" name="id" value="{{.ID}}"><input type="hidden" name="status" value="Neu"><button class="ghost" type="submit">Wieder öffnen</button></form>{{end}}
                      </div>
                    {{end}}
                  </article>
                {{end}}
              </div>
            {{else}}
              {{template "emptyState" .IssuesEmpty}}
            {{end}}
          </section>
          {{if .CanManageIssues}}
          <section class="panel issue-management-preview" id="issue-manage">
            <div class="section-head">
              <div>
                <div class="kicker">Verwalten</div>
                <h2>Anliegen verwalten</h2>
                <p class="muted">Offene Punkte priorisieren, Zuständigkeit setzen und Rückfragen bündeln.</p>
              </div>
              <span class="pill">{{.OpenIssueCount}} offen</span>
            </div>
            {{if .HasManageIssuePreview}}
              <div class="issue-preview-list">
                {{range .ManageIssuePreview}}
                  <a class="issue-preview-card" href="/app/anliegen/board#issue-{{.ID}}">
                    <span><strong>{{.Title}}</strong><span>{{.Category}} · {{.Location}}{{if .HasAssignee}} · {{.AssigneeEmail}}{{end}}</span></span>
                    <span class="chips"><span class="pill {{.StatusClass}}">{{.Status}}</span><span class="pill">{{.Priority}}</span></span>
                  </a>
                {{end}}
              </div>
            {{else}}
              {{template "emptyState" .ManageIssuesEmpty}}
            {{end}}
            <div class="issue-management-actions">
              <a class="button primary" href="/app/anliegen/board">Triage-Board öffnen</a>
              {{if .HasCalendarFeedURL}}<a class="button" href="{{.CalendarFeedURL}}">Kalender abonnieren</a>{{end}}
            </div>
          </section>
          {{end}}
        </div>{{end}}
        {{if and .CanManageIssues .BoardOnly}}
          <section class="panel" id="issue-manage">
            <div class="section-head">
              <div>
                <div class="kicker">Anliegen verwalten</div>
                <p class="muted">Status, Priorität und Zuständigkeit innerhalb der Hausverwaltung setzen.</p>
              </div>
            </div>
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
                <button type="submit">Filtern</button>
                {{if .BoardFilters.HasActive}}<a href="{{.BoardAction}}">Zurücksetzen</a>{{end}}
              </div>
            </form>
            {{if .HasManageIssues}}
              <div class="issue-list">
                {{range .ManageIssues}}
                  <article class="issue-card" id="issue-{{.ID}}">
                    <div class="issue-meta">
                      <span class="pill {{.StatusClass}}">{{.Status}}</span>
	                      <span class="pill">{{.Category}}</span>
	                      <span class="pill">{{.Priority}}</span>
	                      <span>{{.Author}}</span>
	                      <span>{{.CreatedAt}}</span>
	                    </div>
	                    <h3>{{.Title}}</h3>
	                    <p class="issue-location">{{.Location}}{{if .HasAssignee}} · Zuständig: {{.AssigneeEmail}}{{end}}{{if .HasPhotos}} · {{.PhotoCount}} Foto{{if ne .PhotoCount 1}}s{{end}}{{end}}</p>
	                    {{if .HasServiceAppointment}}<p class="issue-proposal"><strong>Termin:</strong> {{.ServiceAppointment}}</p>{{end}}{{if .HasServiceProposal}}<p class="issue-proposal"><strong>Hinweis:</strong> {{.ServiceProposal}}</p>{{end}}
	                    {{template "attachmentStrip" .}}
                    <details class="issue-card-tools">
                      <summary>Kommentar &amp; Status</summary>
                      <div class="issue-card-tools-body">
                        {{template "issueEstimate" .}}
                        <div class="comment-thread">
                          {{if .HasComments}}
                            {{range .Comments}}{{template "issueComment" .}}{{end}}
                          {{else}}
                            <p class="empty">Noch keine Kommentare.</p>
                          {{end}}
                        </div>
                        {{if .CanComment}}<form class="comment-form" method="post" action="/app/anliegen/comment" enctype="multipart/form-data">
                          <input type="hidden" name="id" value="{{.ID}}">
                          <textarea name="body" maxlength="3000" placeholder="Kommentar oder Rückfrage schreiben" aria-label="Kommentar oder Rückfrage"></textarea>
                          <label class="comment-upload">
                            <span class="file-control"><input type="file" name="attachments" accept="image/jpeg,image/png,image/webp,image/gif,application/pdf" multiple><span>Anhang hinzufügen</span></span>
                          </label>
                          <button type="submit">Kommentar senden</button>
                        </form>{{end}}
                        <form class="issue-actions" method="post" action="/app/anliegen/workflow">
                          <input type="hidden" name="id" value="{{.ID}}">
                          <label>Status
                            <select name="status">
                              {{range .StatusOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}
                            </select>
                          </label>
                          <label>Priorität
                            <select name="priority">
                              {{range .PriorityOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}
                            </select>
                          </label>
	                          <label class="assignee">Zuständig
	                            {{if $.ServiceProviderAccessEnabled}}
	                              <input type="email" name="assignee_email" value="{{.AssigneeEmail}}" placeholder="name@example.com"{{if $.HasServiceProviderContacts}} list="service-provider-contacts"{{end}}>
	                              <span class="hint">Neue E-Mail lädt als Dienstleister ein. Leeren entzieht den Zugriff.</span>
	                            {{else}}
	                              <input type="email" value="{{.AssigneeEmail}}" placeholder="Datenschutzprüfung offen" disabled>
	                              <input type="hidden" name="assignee_email" value="{{.AssigneeEmail}}">
	                              <span class="hint">Datenschutzprüfung offen: Neue Dienstleister können nicht zugeordnet werden.</span>
	                              {{if .HasAssignee}}<span class="hint assignee-removal"><input type="checkbox" name="remove_assignee" value="1"> Bestehende Zuordnung entfernen</span>{{end}}
	                            {{end}}
	                          </label>
                          <button type="submit">Aktualisieren</button>
                        </form>
                      </div>
                    </details>
                  </article>
                {{end}}
              </div>
            {{else}}
              {{template "emptyState" .ManageIssuesEmpty}}
            {{end}}
          </section>
        {{end}}
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "announcements"}}
{{template "appOpen" .}}
    <script src="/assets/announcements.js?v={{.AssetVersion}}" defer></script>
    <script src="/assets/attachments.js?v={{.AssetVersion}}" defer></script>
    <main class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M4 5h16v13H7l-3 3z"/><path d="M8 9h8M8 13h6"/></svg><span>/</span><span>Aushang</span></span>
        {{if .CanManageAnnouncements}}<div class="page-actions"><button class="button primary" type="button" data-dialog="announcement-create" aria-haspopup="dialog" aria-controls="announcement-create"><svg viewBox="0 0 24 24" width="17" height="17" aria-hidden="true"><path d="M12 5v14M5 12h14" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round"/></svg>Neu verfassen</button></div>{{end}}
      </div>
      <section class="page">
        <div>
          <h1>Aushang</h1>
          <p class="lede">Offizielle Informationen, Termine und Hinweise der Hausgemeinschaft.</p>
        </div>
        {{if .AnnounceMsg}}<p class="flash ok">{{.AnnounceMsg}}</p>{{end}}
        <div class="home-grid">
          <section class="panel">
            <div class="section-head">
              <div class="kicker">Archiv</div>
              {{if .HasAnnouncements}}<span class="pill">{{len .Announcements}} Treffer</span>{{end}}
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
              <div class="entries">
                {{range .Announcements}}
                  <article class="entry" id="announcement-{{.ID}}">
                    <div class="entry-head">
                      <div>
                        <h3>{{.Title}}</h3>
                        <div class="entry-meta">
                          <span class="pill {{.CategoryClass}}">{{.Category}}</span>
                          {{if .Unread}}<span class="pill unread">neu</span>{{end}}
                          {{if .Pinned}}<span class="pill">Fixiert</span>{{end}}
                          {{if .Status}}<span class="pill">{{.Status}}</span>{{end}}
                          <span>{{.PublishedAt}}</span>
                          {{if .HasExpiresAt}}<span>bis {{.ExpiresAt}}</span>{{end}}
                        </div>
                      </div>
                    </div>
                    <div class="entry-body">{{.BodyHTML}}</div>
                    {{template "attachmentStrip" .}}
                  </article>
                {{end}}
              </div>
            {{else}}
              {{if .HasAnyAnnouncements}}
                {{template "emptyState" .AnnouncementsEmpty}}
              {{else}}
                {{template "emptyState" .AnnouncementsBlank}}
              {{end}}
            {{end}}
          </section>

          <section class="panel">
            <div class="kicker">Aushang verwalten</div>
            {{if .CanManageAnnouncements}}
              <div class="quick-list">
                <button class="quick-row" type="button" data-dialog="announcement-create" aria-haspopup="dialog" aria-controls="announcement-create">
                  <svg viewBox="0 0 24 24"><path d="M12 5v14M5 12h14"/></svg>
                  <div><h3>Neu verfassen</h3><p>Kategorie, Fixierung, Veröffentlichung und Ablaufdatum setzen.</p></div>
                  <span class="quick-arrow">›</span>
                </button>
                {{if .HasAllAnnouncements}}
                  {{range .AllAnnouncements}}
                    <div class="quick-row">
                      <svg viewBox="0 0 24 24"><path d="M4 5h16v13H7l-3 3z"/><path d="M8 9h8M8 13h6"/></svg>
                      <div><h3>{{.Title}}</h3><p>{{.Category}} · {{.PublishedAt}}{{if .Status}} · {{.Status}}{{end}}</p></div>
                      <span class="entry-actions">
                        <button class="button small" type="button" data-dialog="{{.EditDialogID}}" aria-haspopup="dialog" aria-controls="{{.EditDialogID}}">Bearbeiten</button>
                        <form method="post" action="/app/announcements/delete" data-confirm="{{.DeleteConfirmLabel}}">
                          <input type="hidden" name="id" value="{{.ID}}">
                          <button class="button small" type="submit">Löschen</button>
                        </form>
                      </span>
                    </div>
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
                            <label for="category-{{.ID}}">Kategorie<select id="category-{{.ID}}" name="category">
                              <option value="Info"{{if eq .Category "Info"}} selected{{end}}>Info</option>
                              <option value="Termin"{{if eq .Category "Termin"}} selected{{end}}>Termin</option>
                              <option value="Wartung"{{if eq .Category "Wartung"}} selected{{end}}>Wartung</option>
                              <option value="Dringend"{{if eq .Category "Dringend"}} selected{{end}}>Dringend</option>
                            </select></label>
                            <label for="published-{{.ID}}">Veröffentlichen<input id="published-{{.ID}}" type="datetime-local" name="published_at" value="{{.PublishedAtInput}}"></label>
                            <label for="expires-{{.ID}}">Ablauf optional<input id="expires-{{.ID}}" type="datetime-local" name="expires_at" value="{{.ExpiresAtInput}}"></label>
                            <label class="check-row"><input type="checkbox" name="pinned" value="true"{{if .PinnedChecked}} checked{{end}}> oben fixieren</label>
                            <label class="full" for="body-{{.ID}}">Text<textarea id="body-{{.ID}}" name="body" required>{{.Body}}</textarea></label>
                            <label class="full" for="attachments-{{.ID}}">Anhänge ergänzen<span class="file-control"><input id="attachments-{{.ID}}" type="file" name="attachments" accept="image/jpeg,image/png,image/webp,image/gif,application/pdf" multiple><span>Bis zu 10 Dateien auswählen</span></span></label>
                          </div>
                          <button class="button primary" type="submit">Speichern</button>
                        </div>
                      </form>
                    </dialog>
                  {{end}}
            {{else}}
                  {{template "emptyState" .AllAnnouncementsEmpty}}
            {{end}}
              </div>
            {{else}}
              <p class="empty">Veröffentlichen und Bearbeiten ist der Verwaltung vorbehalten.</p>
            {{end}}
          </section>
        </div>
      </section>

      {{if .CanManageAnnouncements}}
      <dialog id="announcement-create" class="dialog" aria-labelledby="announcement-create-title">
        <form method="post" action="/app/announcements" enctype="multipart/form-data">
          <div class="dialog-head">
            <h2 id="announcement-create-title">Neu verfassen</h2>
            <button class="dialog-close" type="button" data-close-dialog aria-label="Schließen">&times;</button>
          </div>
          <div class="dialog-body">
            <div class="dialog-grid">
              <label class="full" for="announcement-title">Titel<input id="announcement-title" name="title" required maxlength="140"></label>
              <label for="announcement-category">Kategorie<select id="announcement-category" name="category">
                <option value="Info">Info</option>
                <option value="Termin">Termin</option>
                <option value="Wartung">Wartung</option>
                <option value="Dringend">Dringend</option>
              </select></label>
              <label for="announcement-published">Veröffentlichen<input id="announcement-published" type="datetime-local" name="published_at" value="{{.NowInput}}"></label>
              <label for="announcement-expires">Ablauf optional<input id="announcement-expires" type="datetime-local" name="expires_at"></label>
              <label class="check-row"><input type="checkbox" name="pinned" value="true"> oben fixieren</label>
              <label class="full" for="announcement-body">Text<textarea id="announcement-body" name="body" required></textarea></label>
              <label class="full" for="announcement-attachments">Anhänge<span class="file-control"><input id="announcement-attachments" type="file" name="attachments" accept="image/jpeg,image/png,image/webp,image/gif,application/pdf" multiple><span>Bis zu 10 Dateien auswählen</span></span></label>
            </div>
            <button class="button primary" type="submit">Veröffentlichen</button>
          </div>
        </form>
      </dialog>
      {{end}}
    </main>
{{template "appClose" .}}
{{end}}

{{define "events"}}
{{template "appOpen" .}}
    <script src="/assets/announcements.js?v={{.AssetVersion}}" defer></script>
    <script src="/assets/attachments.js?v={{.AssetVersion}}" defer></script>
    <main class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M7 3v4M17 3v4"/><path d="M4.5 6h15v14h-15z"/><path d="M4.5 10h15"/><path d="M8 14h.01M12 14h.01M16 14h.01"/></svg><span>/</span><span>Termine</span></span>
        {{if or .HasCalendarFeedURL .CanManageEvents}}<div class="page-actions">
          {{if .HasCalendarFeedURL}}<a class="button" href="{{.CalendarFeedURL}}"><svg viewBox="0 0 24 24" width="17" height="17" aria-hidden="true"><path d="M7 3v4M17 3v4"/><path d="M4.5 6h15v14h-15z"/><path d="M4.5 10h15"/><path d="M8 14h8M8 17h5"/></svg>Kalender abonnieren</a>{{end}}
          {{if .CanManageEvents}}<button class="button primary" type="button" data-dialog="event-create" aria-haspopup="dialog" aria-controls="event-create"><svg viewBox="0 0 24 24" width="17" height="17" aria-hidden="true"><path d="M12 5v14M5 12h14" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round"/></svg>Termin anlegen</button>{{end}}
        </div>{{end}}
      </div>
      <section class="page">
        <div>
          <h1>Termine</h1>
          <p class="lede">Versammlungen, Wartungen, Fristen und gemeinsame Ereignisse im Haus.</p>
        </div>
        {{if .EventMsg}}<p class="flash ok">{{.EventMsg}}</p>{{end}}
        <div class="home-grid">
          <section class="panel">
            <div class="section-head">
              <div class="kicker">Kommende Termine</div>
              {{if .HasEvents}}<span class="pill">{{len .Events}} geplant</span>{{end}}
            </div>
            {{if .HasEvents}}
              <div class="agenda-list">
                {{range .Events}}
                  <article class="event-card" id="event-{{.ID}}">
                    <span class="date-badge"><strong>{{.DateBadgeDay}}</strong><span>{{.DateBadgeMonth}}</span></span>
                    <div class="event-info">
                      <h3>{{.Title}}</h3>
                      <div class="event-meta">
                        <span class="pill {{.CategoryClass}}">{{.Category}}</span>
                        <span>{{.StartsAt}}</span>
                        <span>{{.TimeRange}}</span>
                        {{if .HasLocation}}<span>{{.Location}}</span>{{end}}
                        {{if .Status}}<span class="pill">{{.Status}}</span>{{end}}
                      </div>
                      {{if .HasBody}}<div class="entry-body">{{.BodyHTML}}</div>{{end}}
                      {{template "attachmentStrip" .}}
                    </div>
                  </article>
                {{end}}
              </div>
            {{else}}
              {{template "emptyState" .EventsEmpty}}
            {{end}}
          </section>

          <section class="panel">
            <div class="kicker">Termine verwalten</div>
            {{if .CanManageEvents}}
              <div class="quick-list">
                <button class="quick-row" type="button" data-dialog="event-create" aria-haspopup="dialog" aria-controls="event-create">
                  <svg viewBox="0 0 24 24"><path d="M12 5v14M5 12h14"/></svg>
                  <div><h3>Termin anlegen</h3><p>Kategorie, Zeitpunkt, Ort und Details speichern.</p></div>
                  <span class="quick-arrow">›</span>
                </button>
                {{if .HasAllEvents}}
                  {{range .AllEvents}}
                    <div class="quick-row">
                      <svg viewBox="0 0 24 24"><path d="M7 3v4M17 3v4"/><path d="M4.5 6h15v14h-15z"/><path d="M4.5 10h15"/><path d="M8 14h.01M12 14h.01M16 14h.01"/></svg>
                      <div><h3>{{.Title}}</h3><p>{{.Category}} · {{.StartsAt}}{{if .Past}} · vergangen{{end}}</p></div>
                      <span class="entry-actions">
                        <button class="button small" type="button" data-dialog="{{.EditDialogID}}" aria-haspopup="dialog" aria-controls="{{.EditDialogID}}">Bearbeiten</button>
                        <form method="post" action="/app/events/delete" data-confirm="{{.DeleteConfirmLabel}}">
                          <input type="hidden" name="id" value="{{.ID}}">
                          <button class="button small" type="submit">Löschen</button>
                        </form>
                      </span>
                    </div>
                    <dialog id="{{.EditDialogID}}" class="dialog" aria-labelledby="{{.EditDialogID}}-title">
                      <form method="post" action="/app/events/edit" enctype="multipart/form-data">
                        <input type="hidden" name="id" value="{{.ID}}">
                        <div class="dialog-head">
                          <h2 id="{{.EditDialogID}}-title">Termin bearbeiten</h2>
                          <button class="dialog-close" type="button" data-close-dialog aria-label="Schließen">&times;</button>
                        </div>
                        <div class="dialog-body">
                          <div class="dialog-grid">
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
                            <label for="event-end-{{.ID}}">Ende optional<input id="event-end-{{.ID}}" type="datetime-local" name="ends_at" value="{{.EndsAtInput}}"></label>
                            <label class="full" for="event-location-{{.ID}}">Ort<input id="event-location-{{.ID}}" name="location" value="{{.Location}}" maxlength="160"></label>
                            <label class="full" for="event-body-{{.ID}}">Details<textarea id="event-body-{{.ID}}" name="body">{{.Body}}</textarea></label>
                            <label class="full" for="event-attachments-{{.ID}}">Anhänge ergänzen<span class="file-control"><input id="event-attachments-{{.ID}}" type="file" name="attachments" accept="image/jpeg,image/png,image/webp,image/gif,application/pdf" multiple><span>Bis zu 10 Dateien auswählen</span></span></label>
                          </div>
                          <button class="button primary" type="submit">Speichern</button>
                        </div>
                      </form>
                    </dialog>
                  {{end}}
            {{else}}
                  {{template "emptyState" .AllEventsEmpty}}
            {{end}}
              </div>
            {{else}}
              <p class="empty">Anlegen und Bearbeiten ist der Verwaltung vorbehalten.</p>
            {{end}}
          </section>
        </div>
      </section>

      {{if .CanManageEvents}}
      <dialog id="event-create" class="dialog" aria-labelledby="event-create-title">
        <form method="post" action="/app/events" enctype="multipart/form-data">
          <div class="dialog-head">
            <h2 id="event-create-title">Termin anlegen</h2>
            <button class="dialog-close" type="button" data-close-dialog aria-label="Schließen">&times;</button>
          </div>
          <div class="dialog-body">
            <div class="dialog-grid">
              <label class="full" for="event-title">Titel<input id="event-title" name="title" required maxlength="140"></label>
              <label for="event-category">Kategorie<select id="event-category" name="category">
                <option value="Eigentümerversammlung">Eigentümerversammlung</option>
                <option value="Reinigung">Reinigung</option>
                <option value="Wartung">Wartung</option>
                <option value="Ablesung">Ablesung</option>
                <option value="Frist">Frist</option>
                <option value="Sonstiges">Sonstiges</option>
              </select></label>
              <label for="event-start">Beginn<input id="event-start" type="datetime-local" name="starts_at" value="{{.NowInput}}" required></label>
              <label for="event-end">Ende optional<input id="event-end" type="datetime-local" name="ends_at"></label>
              <label class="full" for="event-location">Ort<input id="event-location" name="location" maxlength="160"></label>
              <label class="full" for="event-body">Details<textarea id="event-body" name="body"></textarea></label>
              <label class="full" for="event-attachments">Anhänge<span class="file-control"><input id="event-attachments" type="file" name="attachments" accept="image/jpeg,image/png,image/webp,image/gif,application/pdf" multiple><span>Bis zu 10 Dateien auswählen</span></span></label>
            </div>
            <button class="button primary" type="submit">Speichern</button>
          </div>
        </form>
      </dialog>
      {{end}}
    </main>
{{template "appClose" .}}
{{end}}

{{define "documents"}}
{{template "appOpen" .}}
    <script src="/assets/announcements.js?v={{.AssetVersion}}" defer></script>
    <script src="/assets/attachments.js?v={{.AssetVersion}}" defer></script>
    <main class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M7 3h7l3 3v15H7z"/><path d="M14 3v4h4"/><path d="M9 13h6M9 17h6"/></svg><span>/</span><span>Dokumente</span></span>
        {{if .CanManageDocuments}}<div class="page-actions"><button class="button primary" type="button" data-dialog="document-upload" aria-haspopup="dialog" aria-controls="document-upload"><svg viewBox="0 0 24 24" width="17" height="17" aria-hidden="true"><path d="M12 5v14M5 12h14" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round"/></svg>Dokument hochladen</button></div>{{end}}
      </div>
      <section class="page">
        <div>
          <h1>Dokumente</h1>
          <p class="lede">Protokolle, Abrechnungen, Hausordnung und Unterlagen nach Berechtigung der jeweiligen Person.</p>
        </div>
        {{if .DocumentMsg}}<p class="flash {{if .DocumentOK}}ok{{end}}">{{.DocumentMsg}}</p>{{end}}
        <div class="home-grid">
          <section class="panel">
            <div class="section-head">
              <div class="kicker">Ablage</div>
              {{if .HasDocuments}}<span class="pill">{{len .Documents}} Treffer</span>{{else if .HasAnyDocuments}}<span class="pill">0 Treffer</span>{{else}}<span class="pill">Noch leer</span>{{end}}
            </div>
            <form class="filter-form document-filter" method="get" action="/app/dokumente">
              <label for="document-search">Suchen<input id="document-search" name="q" value="{{.SearchQuery}}" placeholder="Titel, Kategorie, Datei oder Person"></label>
              <label for="document-sort">Sortierung<select id="document-sort" name="sort">
                {{range .SortOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}
              </select></label>
              <button class="button" type="submit">Suchen</button>
            </form>
            <div class="document-sections">
              {{range .DocumentSections}}
                <section class="document-section{{if not .HasDocuments}} document-section-empty{{end}}">
                  <h3>{{.Category}}{{if .HasDocuments}}<span class="section-count">{{len .Documents}}</span>{{end}}</h3>
                  {{if .HasDocuments}}
                    <div class="document-list">
                      {{range .Documents}}
                        <article class="document-row" id="document-{{.ID}}">
                          <div>
                            <strong>{{.Title}}</strong>
                            <div class="document-meta">
                              <span class="pill {{.VisibilityClass}}">{{.Visibility}}</span>
                              {{if .HasUnit}}<span class="pill">Einheit {{.UnitLabel}}</span>{{end}}
                              <span>{{.VersionLabel}}</span>
                              <span>{{.UploadedAt}}</span>
                              <span>{{.Size}}</span>
                              <span class="document-file">{{.Filename}}</span>
                            </div>
                          </div>
                          <div class="document-side">
                            <span class="pill">{{.Category}}</span>
                            {{if .CanPreview}}{{if .IsImage}}<button class="button small" type="button" data-lightbox-src="{{.PreviewURL}}" data-lightbox-caption="{{.Filename}}">Vorschau</button>{{else}}<a class="button small" href="{{.PreviewURL}}" target="_blank" rel="noopener">Vorschau</a>{{end}}{{end}}
                            <a class="button small" href="{{.DownloadURL}}">Herunterladen</a>
                            {{if $.CanManageDocuments}}<button class="button small" type="button" data-dialog="{{.ReplaceDialogID}}" aria-haspopup="dialog" aria-controls="{{.ReplaceDialogID}}">Ersetzen</button>{{end}}
                          </div>
                          {{if .HasVersions}}
                            <details class="document-versions">
                              <summary>Ältere Versionen</summary>
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
                              <h2 id="{{.ReplaceDialogID}}-title">Neue Version hochladen</h2>
                              <button class="dialog-close" type="button" data-close-dialog aria-label="Schließen">&times;</button>
                            </div>
                            <div class="dialog-body">
                              <p class="mini">{{.Title}} · aktuell {{.VersionLabel}}</p>
                              <div class="dialog-grid">
                                <label class="full" for="{{.ReplaceDialogID}}-file">Datei<input id="{{.ReplaceDialogID}}-file" type="file" name="document" accept="application/pdf,image/jpeg,image/png,image/webp" required></label>
                              </div>
                              <p class="mini">Kategorie, Sichtbarkeit und Einheit bleiben unverändert; die bisherige Version bleibt im Verlauf abrufbar.</p>
                              <button class="button primary" type="submit">Version speichern</button>
                            </div>
                          </form>
                        </dialog>
                        {{end}}
                      {{end}}
                    </div>
                  {{else}}
                    <p class="empty">{{.EmptyMessage}}</p>
                  {{end}}
                </section>
              {{end}}
            </div>
          </section>

          <section class="panel">
            <div class="kicker">Dokumentenverwaltung</div>
            {{if .CanManageDocuments}}
              <div class="quick-list">
                <button class="quick-row" type="button" data-dialog="document-upload" aria-haspopup="dialog" aria-controls="document-upload">
                  <svg viewBox="0 0 24 24"><path d="M12 5v14M5 12h14"/></svg>
                  <div><h3>Dokument hochladen</h3><p>Kategorie, Sichtbarkeit und Datei bis {{.MaxDocumentSize}} speichern.</p></div>
                  <span class="quick-arrow">›</span>
                </button>
                <div class="legend" aria-label="Sichtbarkeiten für Dokumente">
                  <div><strong>Alle Bewohner</strong><span>Allgemeine Informationen wie Hausordnung oder Hinweise.</span></div>
                  <div><strong>Nur Eigentümer</strong><span>Unterlagen für Eigentümer, etwa Protokolle oder Abrechnungen.</span></div>
                  <div><strong>Nur Verwaltung</strong><span>Interne Arbeitsdokumente der Verwaltung.</span></div>
                </div>
              </div>
            {{else}}
              <p class="empty">Hochladen und Sichtbarkeit setzen ist der Verwaltung vorbehalten.</p>
            {{end}}
          </section>
        </div>
      </section>

      {{if .CanManageDocuments}}
      <dialog id="document-upload" class="dialog" aria-labelledby="document-upload-title">
        <form method="post" action="/app/dokumente" enctype="multipart/form-data">
          <div class="dialog-head">
            <h2 id="document-upload-title">Dokument hochladen</h2>
            <button class="dialog-close" type="button" data-close-dialog aria-label="Schließen">&times;</button>
          </div>
          <div class="dialog-body">
            <div class="dialog-grid">
              <label class="full" for="document-title">Titel<input id="document-title" name="title" required maxlength="160" autocomplete="off"></label>
              <label for="document-category">Kategorie<select id="document-category" name="category" required>
                {{range .CategoryOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}
              </select></label>
              <label for="document-visibility">Sichtbarkeit<select id="document-visibility" name="visibility" required>
                {{range .VisibilityOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}
              </select></label>
              <label for="document-unit">Einheit optional<select id="document-unit" name="unit_id">
                {{range .UnitOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}
              </select></label>
              <label class="full" for="document-file">Datei<input id="document-file" type="file" name="document" accept="application/pdf,image/jpeg,image/png,image/webp" required></label>
            </div>
            <p class="mini">Erlaubt sind PDF, JPG, PNG oder WebP bis {{.MaxDocumentSize}}. Dateien werden nicht öffentlich ausgeliefert.</p>
            <button class="button primary" type="submit">Hochladen</button>
          </div>
        </form>
      </dialog>
      {{end}}
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
    body { margin: 0; background: #f7f3ea; color: #20251f; font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }
    main { width: min(920px,100%); margin: 0 auto; padding: 42px 28px; }
    h1, h2, h3 { font-family: Spectral, serif; margin: 0; }
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
    .box strong { display: block; margin-top: 6px; font-family: Spectral, serif; font-size: 22px; }
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

{{define "ballots"}}
{{template "appOpen" .}}
    <script src="/assets/announcements.js?v={{.AssetVersion}}" defer></script>
    <script src="/assets/attachments.js?v={{.AssetVersion}}" defer></script>
    <main class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M5 19V9M12 19V5M19 19v-7"/><path d="M3.5 19h17"/></svg><span>/</span><span>Abstimmungen</span></span>
        {{if .CanManageVotes}}<div class="page-actions"><button class="button primary" type="button" data-dialog="ballot-create" aria-haspopup="dialog" aria-controls="ballot-create"><svg viewBox="0 0 24 24" width="17" height="17" aria-hidden="true"><path d="M12 5v14M5 12h14" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round"/></svg>Abstimmung anlegen</button></div>{{end}}
      </div>
      <section class="page">
        <div>
          <h1>Abstimmungen</h1>
          <p class="lede">Beschlüsse, Umlaufbeschlüsse und Stimmabgabe für Eigentümer der Gemeinschaft.</p>
        </div>
        {{if .VoteMsg}}<p class="flash {{if .VoteOK}}ok{{end}}">{{.VoteMsg}}</p>{{end}}
        <div class="home-grid">
          <section class="panel">
            <div class="section-head">
              <div class="kicker">Abstimmungen</div>
              {{if .HasBallots}}<span class="pill">{{.BallotCountLabel}}</span>{{else}}<span class="pill">Noch leer</span>{{end}}
            </div>
            {{if .HasBallots}}
              <div class="vote-list">
                {{range .Ballots}}
                  <article class="vote-card" id="ballot-{{.ID}}">
                    <div class="vote-card-head">
                      <div>
                        <h3>{{.Title}}</h3>
                        <div class="vote-meta">
                          <span class="pill {{.StatusClass}}">{{.Status}}</span>
                          <span>{{.Type}}</span>
                          <span>{{.Weighting}}</span>
                          {{if .HasQuorum}}<span>Quorum {{.Quorum}}</span>{{end}}
                          {{if .HasClosesAt}}<span>Erinnerung {{.ReminderLabel}}</span>{{end}}
                          {{if .HasOpensAt}}<span>ab {{.OpensAt}}</span>{{end}}
                          {{if .HasClosesAt}}<span>bis {{.ClosesAt}}</span>{{end}}
                        </div>
                      </div>
                      {{if .HasVote}}<span class="pill ok">Stimme gespeichert</span>{{end}}
                    </div>
                    {{if .HasDescription}}<p class="muted">{{.Description}}</p>{{end}}
                    {{template "attachmentStrip" .}}
                    {{if .CanVote}}
                      <form method="post" action="/app/abstimmungen">
                        <input type="hidden" name="ballot_id" value="{{.ID}}">
                        <div class="vote-options">
                          {{range .Options}}
                            <label class="vote-option"><input type="radio" name="option" value="{{.Value}}" required{{if .Selected}} checked{{end}}> <span>{{.Label}}</span></label>
                          {{end}}
                        </div>
                        <div class="vote-actions">
                          <span class="mini">{{if .HasVote}}Aktuell: {{.VoteOption}} · {{.VoteWeight}} · {{.VotedAt}}{{else}}Stimmgewicht: {{.VoteWeight}}{{end}}</span>
                          <button class="button primary" type="submit">{{if .HasVote}}Stimme ändern{{else}}Stimme speichern{{end}}</button>
                        </div>
                      </form>
                    {{else}}
                      <div class="vote-options">
                        {{range .Options}}<div class="vote-option"><span></span><span>{{.Label}}</span></div>{{end}}
                      </div>
                      {{if .ReadOnlyMessage}}<p class="empty">{{.ReadOnlyMessage}}</p>{{end}}
                    {{end}}
                    {{if .HasResults}}
                      <div class="vote-result" aria-label="Abstimmungsergebnis">
                        <div class="vote-meta">
                          <span>{{.TotalVotes}} Stimmen</span>
                          <span>{{.TotalWeightLabel}} von {{.EligibleWeightLabel}}</span>
                          <span>Teilnahme {{.Participation}}</span>
                          <span class="pill {{.QuorumClass}}">{{.QuorumStatus}}</span>
                          {{if .HasWinner}}<span>Ergebnis {{.WinnerLabel}}</span>{{end}}
                          {{if .HasProtocol}}<a class="button small" href="{{.ProtocolURL}}">Protokoll</a>{{end}}
                        </div>
                        {{range .Options}}
                          <div class="vote-result-row">
                            <strong>{{.Label}}</strong>
                            <span class="vote-bar"><span style="width: {{.PercentStyle}}%;"></span></span>
                            <span>{{.WeightLabel}} · {{.VoteCount}} Stimmen</span>
                          </div>
                        {{end}}
                      </div>
                    {{end}}
                  </article>
                {{end}}
              </div>
            {{else}}
              {{template "emptyState" .BallotsEmpty}}
            {{end}}
          </section>

          <section class="panel">
            <div class="kicker">Verwaltung</div>
            {{if .CanManageVotes}}
              <div class="quick-list">
                <button class="quick-row" type="button" data-dialog="ballot-create" aria-haspopup="dialog" aria-controls="ballot-create">
                  <svg viewBox="0 0 24 24"><path d="M12 5v14M5 12h14"/></svg>
                  <div><h3>Abstimmung anlegen</h3><p>Optionen, Frist, Quorum und Gewichtung festlegen.</p></div>
                  <span class="quick-arrow">›</span>
                </button>
              </div>
              {{if .HasManageBallots}}
                <div class="vote-manage-list">
                  {{range .ManageBallots}}
                    <div class="vote-manage-row">
                      <div>
                        <strong>{{.Title}}</strong>
                        <div class="vote-meta"><span class="pill {{.StatusClass}}">{{.Status}}</span><span>{{.Type}}</span><span>{{.Weighting}}</span>{{if .HasClosesAt}}<span>Erinnerung {{.ReminderLabel}}</span>{{end}}<span>{{.UpdatedAt}}</span></div>
                        {{template "attachmentStrip" .}}
                      </div>
                      <div class="vote-manage-actions">
                        {{if .CanOpen}}<form method="post" action="/app/abstimmungen/open"><input type="hidden" name="id" value="{{.ID}}"><button class="button small" type="submit">Öffnen</button></form>{{end}}
                        {{if .CanClose}}<form method="post" action="/app/abstimmungen/close"><input type="hidden" name="id" value="{{.ID}}"><button class="button small" type="submit">Schließen</button></form>{{end}}
                        {{if .HasProtocol}}<a class="button small" href="{{.ProtocolURL}}">Protokoll</a>{{end}}
                      </div>
                    </div>
                  {{end}}
                </div>
              {{else}}
                {{template "emptyState" .ManageBallotsEmpty}}
              {{end}}
            {{else if .CanOversightVotes}}
              <p class="empty">Beirat sieht offene Abstimmungen und Ergebnisse lesend.</p>
            {{else}}
              <p class="empty">Abstimmungen werden von der Verwaltung angelegt.</p>
            {{end}}
          </section>
        </div>
      </section>

      {{if .CanManageVotes}}
      <dialog id="ballot-create" class="dialog" aria-labelledby="ballot-create-title">
        <form method="post" action="/app/abstimmungen" enctype="multipart/form-data">
          <div class="dialog-head">
            <h2 id="ballot-create-title">Abstimmung anlegen</h2>
            <button class="dialog-close" type="button" data-close-dialog aria-label="Schließen">&times;</button>
          </div>
          <div class="dialog-body">
            <div class="dialog-grid">
              <label class="full" for="ballot-title">Titel<input id="ballot-title" name="title" required maxlength="160" autocomplete="off"></label>
              <label for="ballot-type">Typ<select id="ballot-type" name="type" required>
                <option value="Umlaufbeschluss">Umlaufbeschluss</option>
                <option value="Versammlung">Versammlung</option>
              </select></label>
              <label for="ballot-weighting">Gewichtung<select id="ballot-weighting" name="weighting" required>
                <option value="per-share">nach Miteigentumsanteil</option>
                <option value="per-head">pro Kopf</option>
              </select></label>
              <label for="ballot-opens">Öffnen optional<input id="ballot-opens" type="datetime-local" name="opens_at" value="{{.NowInput}}"></label>
              <label for="ballot-closes">Frist optional<input id="ballot-closes" type="datetime-local" name="closes_at"></label>
              <label for="ballot-quorum">Quorum in %<input id="ballot-quorum" name="quorum_percent" inputmode="decimal" placeholder="50"></label>
              <label for="ballot-reminder">Erinnerung vor Frist (h)<input id="ballot-reminder" name="reminder_before_hours" inputmode="decimal" value="24"></label>
              <label class="full" for="ballot-options">Optionen<textarea id="ballot-options" name="options_text" required placeholder="Ja&#10;Nein&#10;Enthaltung"></textarea></label>
              <label class="full" for="ballot-description">Beschreibung<textarea id="ballot-description" name="description"></textarea></label>
              <label class="full" for="ballot-attachments">Anhänge<span class="file-control"><input id="ballot-attachments" type="file" name="attachments" accept="image/jpeg,image/png,image/webp,image/gif,application/pdf" multiple><span>Bis zu 10 Dateien auswählen</span></span></label>
            </div>
            <button class="button primary" type="submit">Anlegen</button>
          </div>
        </form>
      </dialog>
      {{end}}
    </main>
{{template "appClose" .}}
{{end}}

{{define "handovers"}}
{{template "appOpen" .}}
    <script src="/assets/attachments.js?v={{.AssetVersion}}" defer></script>
    <main class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M7 4h10v16H7z"/><path d="M9.5 8h5M9.5 12h4"/><path d="m9.5 16 1.5 1.5 3.5-4"/></svg><span>/</span><span>Übergaben</span></span>
        <div class="page-actions">
          <button class="button primary" type="button" data-dialog="handover-create" aria-haspopup="dialog" aria-controls="handover-create">Übergabe anlegen</button>
        </div>
      </div>
      <section class="page wide handover-page">
        <div>
          <h1>Übergaben</h1>
          <p class="lede">Nutzerwechsel vor Ort erfassen, Fotos sichern, bestätigen lassen und als Protokoll im Dokumentenbereich ablegen.</p>
        </div>
        {{if .HandoverMsg}}<div class="flash {{if .HandoverOK}}ok{{end}}">{{.HandoverMsg}}</div>{{end}}
        {{if .HasHandovers}}
          <div class="handover-list">
            {{range .Handovers}}
              <article class="panel handover-card" id="handover-{{.ID}}">
                <div class="handover-head">
                  <div>
                    <div class="handover-meta"><span class="pill {{.StatusClass}}">{{.Status}}</span><span>{{.Type}}</span>{{if .HasUnit}}<span>{{.UnitLabel}}</span>{{end}}{{if .HasScheduledAt}}<span>{{.ScheduledAt}}</span>{{end}}</div>
                    <h2>{{.Title}}</h2>
                    <p class="muted">Ausziehend: {{.Outgoing}} · Einziehend: {{.Incoming}}</p>
                  </div>
                  <div class="handover-actions">
                    <a class="button small" href="{{.ProtocolURL}}">PDF exportieren</a>
                    {{if .HasFiledDocument}}
                      <a class="button small" href="{{.FiledDocumentURL}}">Dokument öffnen</a>
                    {{else if $.CanManageDocuments}}
                      <form method="post" action="{{.FileURL}}">
                        <input type="hidden" name="id" value="{{.ID}}">
                        <button class="button small" type="submit">Im Dokumentenbereich ablegen</button>
                      </form>
                    {{end}}
                  </div>
                </div>
                <div class="handover-detail-grid">
                  <section class="handover-detail">
                    <h3>Räume</h3>
                    {{if .HasRooms}}<ul>{{range .Rooms}}<li><strong>{{.Name}}</strong>{{if .Condition}}<span>{{.Condition}}</span>{{end}}{{if .Defects}}<em>{{.Defects}}</em>{{end}}</li>{{end}}</ul>{{else}}<p>Keine Räume erfasst.</p>{{end}}
                  </section>
                  <section class="handover-detail">
                    <h3>Zähler</h3>
                    {{if .HasMeters}}<ul>{{range .Meters}}<li><strong>{{.Label}}</strong><span>{{.Value}}{{if .Unit}} {{.Unit}}{{end}}</span></li>{{end}}</ul>{{else}}<p>Keine Zählerstände erfasst.</p>{{end}}
                  </section>
                  <section class="handover-detail">
                    <h3>Schlüssel</h3>
                    {{if .HasKeys}}<ul>{{range .Keys}}<li><strong>{{.Label}}</strong><span>{{.Count}} Stk.</span></li>{{end}}</ul>{{else}}<p>Keine Schlüssel erfasst.</p>{{end}}
                  </section>
                  <section class="handover-detail">
                    <h3>Bestätigung</h3>
                    {{if .HasConfirmations}}<ul>{{range .Confirmations}}<li><strong>{{.Role}}</strong><span>{{if .Name}}{{.Name}}{{else}}{{.Email}}{{end}}</span><span class="pill {{.StatusClass}}">{{.Status}}</span></li>{{end}}</ul>{{else}}<p>Keine externen Bestätigungen vorgesehen.</p>{{end}}
                  </section>
                </div>
                {{if .HasNotes}}<div class="handover-note"><strong>Notiz</strong><p>{{.Notes}}</p></div>{{end}}
                {{template "attachmentStrip" .AttachmentGroup}}
                <div class="handover-foot"><span>Angelegt {{.CreatedAt}}</span><span>Aktualisiert {{.UpdatedAt}}</span></div>
              </article>
            {{end}}
          </div>
        {{else}}
          {{template "emptyState" .HandoversEmpty}}
        {{end}}
      </section>

      <dialog id="handover-create" class="dialog handover-dialog" aria-labelledby="handover-create-title">
        <form method="post" action="/app/uebergaben" enctype="multipart/form-data">
          <div class="dialog-head">
            <h2 id="handover-create-title">Übergabe anlegen</h2>
            <button class="dialog-close" type="button" data-close-dialog aria-label="Schließen">&times;</button>
          </div>
          <div class="dialog-body">
            <div class="dialog-grid">
              <label class="full" for="handover-title">Titel<input id="handover-title" name="title" required maxlength="160" placeholder="Übergabe Top 11"></label>
              <label for="handover-unit">Einheit<select id="handover-unit" name="unit_id" required>{{range .UnitOptions}}<option value="{{.Value}}" {{if .Selected}}selected{{end}}>{{.Label}}</option>{{end}}</select></label>
              <label for="handover-type">Typ<select id="handover-type" name="handover_type"><option>Nutzerwechsel</option><option>Einzug</option><option>Auszug</option></select></label>
              <label for="handover-time">Termin<input id="handover-time" type="datetime-local" name="scheduled_at" value="{{.NowInput}}"></label>
              <label for="handover-out-name">Ausziehend Name<input id="handover-out-name" name="outgoing_name" autocomplete="name"></label>
              <label for="handover-out-email">Ausziehend E-Mail<input id="handover-out-email" type="email" name="outgoing_email" autocomplete="email"></label>
              <label for="handover-in-name">Einziehend Name<input id="handover-in-name" name="incoming_name" autocomplete="name"></label>
              <label for="handover-in-email">Einziehend E-Mail<input id="handover-in-email" type="email" name="incoming_email" autocomplete="email"></label>
              <label class="full" for="handover-rooms">Räume<textarea id="handover-rooms" name="rooms_text" required placeholder="Wohnzimmer | gut | keine Mängel&#10;Bad | sauber | Silikonfuge prüfen"></textarea></label>
              <label class="full" for="handover-meters">Zählerstände<textarea id="handover-meters" name="meters_text" placeholder="Strom | 12345,6 | kWh&#10;Wasser kalt | 81,2 | m³"></textarea></label>
              <label class="full" for="handover-keys">Schlüssel<textarea id="handover-keys" name="keys_text" placeholder="Wohnungsschlüssel | 3&#10;Postkasten | 1"></textarea></label>
              <label class="full" for="handover-notes">Notiz<textarea id="handover-notes" name="notes" placeholder="Zusätzliche Vereinbarungen oder offene Punkte"></textarea></label>
              <label class="full" for="handover-attachments">Fotos &amp; Anhänge<span class="file-control"><input id="handover-attachments" type="file" name="attachments" accept="image/jpeg,image/png,image/webp,image/gif,application/pdf" multiple><span>Bis zu 10 Dateien auswählen</span></span></label>
            </div>
            <button class="button primary" type="submit">Übergabe speichern</button>
          </div>
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
      <div class="kicker">Übergabe bestätigen</div>
      <h1>{{.Handover.Title}}</h1>
      <p class="lede">{{.Tenant.Address}} · {{.Handover.UnitLabel}}</p>
      {{if .Msg}}<div class="flash {{if .MsgOK}}ok{{end}}">{{.Msg}}</div>{{end}}
      <div class="handover-confirm-summary">
        <div><span>Rolle</span><strong>{{.Confirmation.Role}}</strong></div>
        <div><span>Status</span><strong>{{.Confirmation.Status}}</strong></div>
        {{if .Handover.HasScheduledAt}}<div><span>Termin</span><strong>{{.Handover.ScheduledAt}}</strong></div>{{end}}
      </div>
      {{if .Confirmation.HasConfirmed}}
        <p class="empty">Diese Übergabe wurde bereits bestätigt.</p>
      {{else}}
        <form method="post" action="/handover/{{.Token}}" class="handover-confirm-form">
          <input type="hidden" name="confirm" value="yes">
          <label>Name für die Bestätigung<input name="name" value="{{.Confirmation.Name}}" autocomplete="name"></label>
          <label>Notiz optional<textarea name="note" placeholder="Falls etwas ergänzt werden soll"></textarea></label>
          <button class="button primary" type="submit">Protokoll bestätigen</button>
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
    <main class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg><span>/</span><span>Parkplatznutzung</span></span>
        <div class="page-actions"></div>
      </div>
      <section class="page wide parking-page">
        <div class="parking-page-head">
          <div>
            <h1>Parkplatznutzung</h1>
            <p class="lede">Private Lade- und Stellplatzabrechnung für die persönlich abgestimmte Nutzung.</p>
            <p class="muted subtle-note">Sichtbar nur für berechtigte Personen. Die Seite führt Schritt für Schritt durch offene Monate.</p>
          </div>
          <div class="parking-primary-actions">
            {{if .Accounting.HasMonths}}{{with index .Accounting.Months 0}}<a class="button primary" href="#parking-detail-{{.Month}}">Nächsten offenen Monat prüfen</a>{{end}}{{end}}
            <details class="parking-more">
              <summary class="button">Mehr</summary>
              <div class="parking-more-menu">
                <a class="button" href="/app/parking"><svg viewBox="0 0 24 24" width="17" height="17" aria-hidden="true"><path d="M4 4v6h6" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"/><path d="M20 20v-6h-6" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"/><path d="M5 10a7 7 0 0 1 12-3M19 14a7 7 0 0 1-12 3" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round"/></svg>Aktualisieren</a>
                <a class="button" href="/app/parking/export/{{.StatementYear}}">CSV exportieren</a>
                {{if .CanManageParkingPayments}}<form method="post" action="/app/parking/reminders"><button class="button" type="submit">Erinnerungen senden</button></form>{{end}}
                {{if .IsAdmin}}<a class="button" href="/app/parking/settings"><svg viewBox="0 0 24 24" width="17" height="17" aria-hidden="true"><path d="M12 8.5a3.5 3.5 0 1 0 0 7 3.5 3.5 0 0 0 0-7z" fill="none" stroke="currentColor" stroke-width="1.9"/><path d="M19 12a7 7 0 0 0-.1-1l2-1.5-2-3.5-2.4 1a7 7 0 0 0-1.8-1L14.4 3h-4.8L9.3 6a7 7 0 0 0-1.8 1l-2.4-1-2 3.5 2 1.5A7 7 0 0 0 5 12a7 7 0 0 0 .1 1l-2 1.5 2 3.5 2.4-1a7 7 0 0 0 1.8 1l.3 3h4.8l.3-3a7 7 0 0 0 1.8-1l2.4 1 2-3.5-2-1.5a7 7 0 0 0 .1-1z" fill="none" stroke="currentColor" stroke-width="1.9"/></svg>Abrechnung konfigurieren</a>{{end}}
              </div>
            </details>
          </div>
        </div>

        {{if .ParkingMsg}}<p class="flash {{if .ParkingOK}}ok{{end}}">{{.ParkingMsg}}</p>{{end}}

        {{template "parkingLiveCard" .}}

        {{if .Accounting.HasMonths}}
          <section class="parking-guide" aria-label="Abrechnung in 2 Schritten. Offen {{.Accounting.Outstanding}}. Überfällig {{.Accounting.Overdue}}.">
            <div>
              <h2>Abrechnung in 2 Schritten</h2>
              <p class="muted">Monat prüfen und Zahlung markieren.</p>
            </div>
            <div class="parking-stepper" aria-label="Abrechnungsschritte">
              <div class="parking-step active">
                <span class="parking-step-number">1</span>
                <span><strong>Monat prüfen</strong><small>Verbrauch und Betrag kontrollieren</small></span>
              </div>
              <div class="parking-step">
                <span class="parking-step-number">2</span>
                <span><strong>Zahlung markieren</strong><small>Zahlung bestätigen und abschließen</small></span>
              </div>
            </div>
            <div class="parking-guide-status">
              {{if .Accounting.HasOutstanding}}<span class="pill">{{.Accounting.Outstanding}} offen</span>{{end}}
              {{if .Accounting.HasOverdue}}<span class="pill dringend">{{.Accounting.Overdue}} überfällig</span>{{end}}
              {{if .Telemetry.Configured}}<span class="pill ok">Home Assistant aktiv</span>{{end}}
            </div>
          </section>

          <div class="parking-workspace">
            <section class="panel parking-assistant" aria-label="Abrechnungsassistent">
              <div class="parking-queue-head">
                <div>
                  <h2>Monate in Bearbeitung</h2>
                  <p class="muted">Offene und überfällige Monate auf einen Blick.</p>
                </div>
                {{if .Accounting.HasOutstanding}}<span class="pill">{{.Accounting.Outstanding}} offen</span>{{end}}
              </div>
              <div>
                <div class="parking-month-queue">
                  {{range .Accounting.Months}}
                    <a class="parking-month-row" id="parking-month-{{.Month}}" href="#parking-detail-{{.Month}}">
                      <span class="parking-month-icon"><svg viewBox="0 0 24 24"><path d="M7 3v4M17 3v4"/><path d="M5 6h14v14H5z"/><path d="M5 10h14"/></svg></span>
                      <span><strong>{{.MonthLabel}}</strong><span class="mini">{{if .Partial}}Teilmonat · {{end}}{{.HourCount}} Stunden</span></span>
                      <span class="amount">{{.TotalCost}}</span>
                      <span class="pill {{if .Paid}}ok{{else if .Overdue}}dringend{{end}}">{{.PaidLabel}}</span>
                      <span class="parking-queue-action">{{if .Paid}}Ansehen{{else}}Prüfen{{end}}</span>
                    </a>
                  {{end}}
                </div>
              </div>
              <details class="parking-utility">
                <summary>Messwerte und Berechnungsgrundlage</summary>
                <div class="parking-utility-body">
                  <p class="muted">{{.Accounting.Message}}</p>
                  {{if .Telemetry.Connected}}
                    <div class="metric-grid">
                      {{range .Telemetry.Metrics}}
                        <div class="metric-card">
                          <span class="metric-label">{{.Label}}</span>
                          <strong class="metric-value">{{.Value}}</strong>
                          <code>{{.Detail}}</code>
                        </div>
                      {{end}}
                    </div>
                  {{else}}
                    <p class="empty">{{.Telemetry.Message}}</p>
                  {{end}}
                </div>
              </details>
            </section>

            <aside class="parking-detail-stack" aria-label="Zahlungsdetails">
              {{range .Accounting.Months}}
                <section class="parking-month-detail" id="parking-detail-{{.Month}}">
                  <div class="parking-detail-head">
                    <div>
                      <span class="metric-label">Ausgewählter Monat</span>
                      <h2>{{.MonthLabel}}</h2>
                      <strong>{{.TotalCost}}</strong>
                      {{if .Paid}}<p class="muted">Zahlung ist markiert.</p>{{else if .Overdue}}<p class="muted">Überfällig. Bitte Zahlung prüfen.</p>{{else if .Outstanding}}<p class="muted">Noch nicht als bezahlt markiert.</p>{{else}}<p class="muted">Keine offene Zahlung für diesen Monat.</p>{{end}}
                    </div>
                    <span class="pill {{if .Paid}}ok{{else if .Overdue}}dringend{{end}}">{{.PaidLabel}}</span>
                  </div>
                  <dl class="parking-breakdown">
                    <div><dt>Verbrauch</dt><dd>{{.KWh}}</dd></div>
                    {{if .HasSurplus}}
                      <div><dt>☀️ Überschuss</dt><dd>{{.SurplusKWh}} · {{.SurplusCost}}</dd></div>
                      <div><dt>Normal</dt><dd>{{.NormalKWh}}</dd></div>
                    {{end}}
                    <div><dt>Ø aWATTar</dt><dd>{{.AverageAwattar}}</dd></div>
                    <div><dt>Ø effektiv</dt><dd>{{.EffectivePrice}}</dd></div>
                    <div><dt>Strom</dt><dd>{{.EnergyCost}}</dd></div>
                    <div><dt>Netzgebühr</dt><dd>{{.GridCost}}</dd></div>
                    <div><dt>Basis</dt><dd>{{.BaseFee}}</dd></div>
                  </dl>
                  <div class="parking-tabs" aria-label="Ansichten für {{.MonthLabel}}">
                    <span>Übersicht</span>
                    <a href="{{.DetailPath}}">Stundenwerte</a>
                  </div>
                  <div class="parking-detail-actions">
                    <div class="pay-sec">
                      <div class="pay-head">
                        <h3>Zahlung</h3>
                        {{if .Paid}}<span class="pay-state paid">Bezahlt</span>{{else}}<span class="pay-state">Offen</span>{{end}}
                      </div>
                      {{if .Paid}}
                        <div class="pay-settled">
                          <span class="pay-check" aria-hidden="true"><svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="10" fill="none" stroke="currentColor" stroke-width="1.8"/><path d="M8 12.5l2.6 2.6L16.5 9" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/></svg></span>
                          <div class="pay-settled-meta">
                            {{if $.CanManageParkingPayments}}
                              <strong>Bezahlt{{if .PaidAtLabel}} am {{.PaidAtLabel}}{{end}}{{if .PaymentMethod}} · {{.PaymentMethod}}{{end}}</strong>
                              {{if .PaymentReference}}<span>Referenz: {{.PaymentReference}}</span>{{end}}
                            {{else}}
                              <strong>Bezahlt{{if .PaidAtLabel}} am {{.PaidAtLabel}}{{end}}</strong>
                              <span>Zahlung ist markiert.</span>
                            {{end}}
                          </div>
                          {{if $.CanManageParkingPayments}}
                            <form class="pay-unmark" method="post" action="/app/parking/month">
                              <input type="hidden" name="month" value="{{.Month}}">
                              <input type="hidden" name="paid" value="false">
                              <button class="button pay-ghost" type="submit">Als offen markieren</button>
                            </form>
                          {{end}}
                        </div>
                        {{if $.CanManageParkingPayments}}
                          <details class="pay-details">
                            <summary>Details bearbeiten</summary>
                            <form class="pay-form" method="post" action="/app/parking/month">
                              <input type="hidden" name="month" value="{{.Month}}">
                              <input type="hidden" name="paid" value="true">
                              <div class="pay-fields">
                                <label>Datum<input type="date" name="paid_at" value="{{.PaidAtInput}}" autocomplete="off"></label>
                                <label>Zahlungsart<input type="text" name="payment_method" value="{{.PaymentMethod}}" placeholder="Überweisung" autocomplete="off"></label>
                                <label class="pay-ref">Referenz<input type="text" name="payment_reference" value="{{.PaymentReference}}" placeholder="z. B. Telegram-Abrechnung 05.03.2026" autocomplete="off"></label>
                              </div>
                              <div class="pay-actions"><button class="button primary" type="submit">Details speichern</button></div>
                            </form>
                          </details>
                        {{end}}
                      {{else if $.CanMarkParkingPayment}}
                        <form class="pay-form" method="post" action="/app/parking/month">
                          <input type="hidden" name="month" value="{{.Month}}">
                          <input type="hidden" name="paid" value="true">
                          <div class="pay-row">
                            <p class="muted">{{if $.CanManageParkingPayments}}Wenn der Betrag eingegangen ist, als erhalten markieren.{{else}}Markieren Sie diesen Monat, wenn Sie die Zahlung erledigt haben.{{end}}</p>
                            <button class="button primary" type="submit">{{if $.CanManageParkingPayments}}Bezahlung erhalten{{else}}Als bezahlt markieren{{end}}</button>
                          </div>
                          <details class="pay-details">
                            <summary>Details (optional)</summary>
                            <div class="pay-fields">
                              <label>Datum<input type="date" name="paid_at" value="{{.PaidAtInput}}" autocomplete="off"></label>
                              <label>Zahlungsart<input type="text" name="payment_method" value="{{.PaymentMethod}}" placeholder="Überweisung" autocomplete="off"></label>
                              <label class="pay-ref">Referenz<input type="text" name="payment_reference" value="{{.PaymentReference}}" placeholder="z. B. Telegram-Abrechnung 05.03.2026" autocomplete="off"></label>
                            </div>
                          </details>
                        </form>
                      {{else}}
                        <p class="muted">Zahlungen können nur von berechtigten Personen markiert werden.</p>
                      {{end}}
                    </div>
                  </div>
                  <div class="parking-detail-foot">
                    <span>Historie wird ab 01.01.2026 aus Home Assistant nachgezogen.</span>
                    <span>Letzter Messpunkt: {{if $.Accounting.LastSampleLabel}}{{$.Accounting.LastSampleLabel}}{{else}}-{{end}}</span>
                  </div>
                </section>
              {{end}}
            </aside>
          </div>
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
              <p class="muted">Sobald zwei Zählerstände und mindestens ein aWATTar-Preis vorliegen, berechnet das Portal den ersten Monat automatisch. Bis dahin bleiben Konfiguration und Zugriff im Vordergrund.</p>
            </div>
            <div class="parking-empty-actions">
              {{if .IsAdmin}}<a class="button primary" href="/app/parking/settings"><svg viewBox="0 0 24 24" width="17" height="17" aria-hidden="true"><path d="M12 8.5a3.5 3.5 0 1 0 0 7 3.5 3.5 0 0 0 0-7z" fill="none" stroke="currentColor" stroke-width="1.9"/><path d="M19 12a7 7 0 0 0-.1-1l2-1.5-2-3.5-2.4 1a7 7 0 0 0-1.8-1L14.4 3h-4.8L9.3 6a7 7 0 0 0-1.8 1l-2.4-1-2 3.5 2 1.5A7 7 0 0 0 5 12a7 7 0 0 0 .1 1l-2 1.5 2 3.5 2.4-1a7 7 0 0 0 1.8 1l.3 3h4.8l.3-3a7 7 0 0 0 1.8-1l2.4 1 2-3.5-2-1.5a7 7 0 0 0 .1-1z" fill="none" stroke="currentColor" stroke-width="1.9"/></svg>Abrechnung konfigurieren</a>{{end}}
              {{if .CanManageParkingPayments}}<a class="button" href="/app/settings/parking-access"><svg viewBox="0 0 24 24" width="17" height="17" aria-hidden="true"><path d="M8.5 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6z" fill="none" stroke="currentColor" stroke-width="1.9"/><path d="M3.5 20a5 5 0 0 1 10 0" fill="none" stroke="currentColor" stroke-width="1.9"/><path d="M17 8v8M13 12h8" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round"/></svg>Zugriff verwalten</a>{{end}}
              {{if and (not .IsAdmin) (not .CanManageParkingPayments)}}<a class="button" href="/app">Hausüberblick öffnen</a>{{end}}
            </div>
            <div class="parking-empty-steps">
              <div class="parking-empty-step">
                <span class="parking-empty-step-number">1</span>
                <div><strong>Abrechnung konfigurieren</strong><p>Tarif, Netzgebühr und Basiswerte einmal sauber festlegen.</p></div>
              </div>
              <div class="parking-empty-step">
                <span class="parking-empty-step-number">2</span>
                <div><strong>Monatswerte prüfen</strong><p>Neue Zähler- und Preisdaten werden automatisch zu Monatswerten.</p></div>
              </div>
              <div class="parking-empty-step">
                <span class="parking-empty-step-number">3</span>
                <div><strong>Zahlung markieren</strong><p>Offene Monate werden nach Prüfung als erledigt markiert.</p></div>
              </div>
            </div>
            <div class="parking-empty-note">
              <span class="parking-empty-note-icon"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 8v5"/><path d="M12 17h.01"/><circle cx="12" cy="12" r="9"/></svg></span>
              <span><strong>Transparenz statt Buchhaltung.</strong> Keine Sollstellung, kein Mahnwesen und keine Zahlungsaufträge. Hier geht es nur um nachvollziehbare private Stellplatznutzung.</span>
              {{if .IsAdmin}}<a class="button" href="/app/parking/settings">Zeitraum konfigurieren</a>{{end}}
            </div>
          </section>
        {{end}}
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "parkingLiveCard"}}
{{with .Live}}{{if .Available}}
  <style>
    .parking-live { display: grid; gap: 14px; }
    .parking-live-head { display: flex; flex-wrap: wrap; align-items: center; gap: 10px; }
    .parking-live .pill.live-surplus { background: rgba(200,153,63,.16); color: #8a6a1f; border: 1px solid rgba(200,153,63,.32); }
    .parking-live .pill.live-manual { background: rgba(76,103,138,.12); color: #365475; border: 1px solid rgba(76,103,138,.24); }
    .parking-live .pill.live-idle { background: var(--panel-soft); color: var(--muted); border: 1px solid var(--line); }
    .parking-live .live-rate { font-weight: 700; }
    .parking-live .battery-full { color: var(--leaf); font-weight: 600; }
    .parking-live .battery-partial { color: #93701d; font-weight: 600; }
    .parking-live .battery-low { color: #a04545; font-weight: 600; }
    .parking-live .bar.split { position: relative; }
    .parking-live .bar.split .surplus { position: absolute; inset: 0 auto 0 0; background: var(--gold); border-radius: inherit; }
    .parking-live .split-row { display: grid; gap: 4px; }
    .parking-live .split-row .mini { display: flex; flex-wrap: wrap; gap: 10px; }
    .parking-live-actions { display: flex; flex-wrap: wrap; gap: 10px; align-items: center; }
    .parking-session-list { display: grid; gap: 6px; }
    .parking-session-row { display: flex; flex-wrap: wrap; align-items: center; gap: 10px; padding: 8px 10px; border: 1px solid var(--line); border-radius: 9px; background: var(--panel-soft); font-size: 13.5px; }
    .parking-session-row .pill.mode-surplus { background: rgba(200,153,63,.16); color: #8a6a1f; border: 1px solid rgba(200,153,63,.32); }
    .parking-session-row .pill.mode-normal { background: rgba(76,103,138,.12); color: #365475; border: 1px solid rgba(76,103,138,.24); }
    .parking-session-row .amount { margin-left: auto; font-weight: 700; }
  </style>
  <section class="panel parking-live" id="parking-live" aria-label="Aktueller Ladezustand Parkplatz 20">
    <div class="parking-live-head">
      <span class="pill {{.ModeClass}}">{{if eq .Mode "surplus"}}☀️ {{else if eq .Mode "manual"}}⚡ {{end}}{{.ModeLabel}}</span>
      {{if .RateLabel}}<span class="live-rate">{{.RateLabel}}</span>{{end}}
      {{if .SessionSince}}<span class="mini">seit {{.SessionSince}}{{if .SessionKWh}} · {{.SessionKWh}}{{if .SessionCost}} · ≈ {{.SessionCost}}{{end}}{{end}}</span>{{end}}
      {{if .ShadowMode}}<span class="pill">Testbetrieb</span>{{end}}
      {{if .StaleData}}<span class="pill dringend">Daten veraltet</span>{{end}}
    </div>
    <p class="muted">{{.ModeDetail}}</p>
    <div class="metric-grid">
      {{if .PowerLabel}}<div class="metric-card"><span class="metric-label">Leistung</span><strong class="metric-value">{{.PowerLabel}}</strong></div>{{end}}
      {{if .BatterySOCLabel}}<div class="metric-card"><span class="metric-label">Hausakku</span><strong class="metric-value">{{.BatterySOCLabel}}</strong><span class="battery-{{.BatteryClass}}">{{.BatteryHint}}</span></div>{{end}}
      {{if .FeedInLabel}}<div class="metric-card"><span class="metric-label">Einspeisung</span><strong class="metric-value">{{.FeedInLabel}}</strong></div>{{end}}
    </div>
    {{if or .TodaySplit.HasAny .MonthSplit.HasAny}}
      <div class="split-row">
        {{if .TodaySplit.HasAny}}
          <div>
            <span class="metric-label">{{.TodaySplit.Label}}</span>
            <div class="bar split"><span class="surplus" style="width: {{.TodaySplit.SurplusPct}}%;"></span></div>
            <div class="mini"><span>☀️ {{.TodaySplit.SurplusKWh}} · {{.TodaySplit.SurplusCost}}</span><span>⚡ {{.TodaySplit.NormalKWh}} · {{.TodaySplit.NormalCost}}</span></div>
          </div>
        {{end}}
        {{if .MonthSplit.HasAny}}
          <div>
            <span class="metric-label">{{.MonthSplit.Label}}</span>
            <div class="bar split"><span class="surplus" style="width: {{.MonthSplit.SurplusPct}}%;"></span></div>
            <div class="mini"><span>☀️ {{.MonthSplit.SurplusKWh}} · {{.MonthSplit.SurplusCost}}</span><span>⚡ {{.MonthSplit.NormalKWh}} · {{.MonthSplit.NormalCost}}</span></div>
          </div>
        {{end}}
      </div>
    {{end}}
    {{if .CanToggle}}
      <div class="parking-live-actions">
        {{if .ToggleOn}}
          <form method="post" action="/app/parking/charging/off"><button class="button" type="submit">Ladung ausschalten</button></form>
        {{else}}
          <form method="post" action="/app/parking/charging/on"><button class="button primary" type="submit">Jetzt laden (Normaltarif)</button></form>
        {{end}}
        {{if .AutoPaused}}
          <form method="post" action="/app/parking/charging/auto"><button class="button" type="submit">Automatik aktivieren</button></form>
        {{end}}
      </div>
    {{end}}
    {{if .Admin.Show}}
      <div class="rule">
        <span class="pill">Regler: {{.Admin.PhaseLabel}}</span>
        {{if .Admin.SinceLabel}}<span class="mini">seit {{.Admin.SinceLabel}}</span>{{end}}
        {{if .Admin.PollLabel}}<span class="mini{{if .Admin.StalePill}} pill dringend{{end}}">HA-Poll {{.Admin.PollLabel}}</span>{{end}}
        {{if .Admin.LastReason}}<span class="mini">{{.Admin.LastReason}}</span>{{end}}
        {{if .Admin.ErrorDetail}}<span class="pill dringend">{{.Admin.ErrorDetail}}</span>{{end}}
        <a class="button" href="/app/parking/settings#laderegelung">Laderegelung</a>
      </div>
    {{end}}
    {{if .HasSessions}}
      <details>
        <summary>Letzte Ladevorgänge</summary>
        {{template "parkingSessionList" .Sessions}}
      </details>
    {{end}}
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
    <main class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg><span>/</span><a href="/app/parking">Parkplatznutzung</a><span>/</span><span>{{.Detail.MonthLabel}}</span></span>
        <div class="page-actions"><a class="button" href="{{.Detail.BackPath}}">Monate</a></div>
      </div>
      <section class="page wide">
        <div>
          <h1>{{.Detail.MonthLabel}}</h1>
          <p class="lede">{{.Detail.Message}}</p>
        </div>
        <section class="panel status-strip">
          <div class="rule">
            <span class="pill">Netzgebühr {{.Detail.GridFeeLabel}}</span>
            {{if .Detail.LastSampleLabel}}<span class="mini">Letzter Zählerwert: {{.Detail.LastSampleLabel}}</span>{{end}}
          </div>
          {{if .Detail.Summary.Month}}
            <div class="metric-grid">
              <div class="metric-card"><span class="metric-label">Verbrauch</span><strong class="metric-value">{{.Detail.Summary.KWh}}</strong></div>
              {{if .Detail.Summary.HasSurplus}}
                <div class="metric-card"><span class="metric-label">☀️ Überschuss</span><strong class="metric-value">{{.Detail.Summary.SurplusKWh}}</strong><span class="mini">{{.Detail.Summary.SurplusCost}}</span></div>
                <div class="metric-card"><span class="metric-label">Normal</span><strong class="metric-value">{{.Detail.Summary.NormalKWh}}</strong></div>
              {{end}}
              <div class="metric-card"><span class="metric-label">Ø aWATTar</span><strong class="metric-value">{{.Detail.Summary.AverageAwattar}}</strong></div>
              <div class="metric-card"><span class="metric-label">Ø effektiv</span><strong class="metric-value">{{.Detail.Summary.EffectivePrice}}</strong></div>
              <div class="metric-card"><span class="metric-label">Strom</span><strong class="metric-value">{{.Detail.Summary.EnergyCost}}</strong></div>
              <div class="metric-card"><span class="metric-label">Netzgeb.</span><strong class="metric-value">{{.Detail.Summary.GridCost}}</strong></div>
              <div class="metric-card"><span class="metric-label">Basis</span><strong class="metric-value">{{.Detail.Summary.BaseFee}}</strong></div>
              <div class="metric-card"><span class="metric-label">Summe</span><strong class="metric-value">{{.Detail.Summary.TotalCost}}</strong></div>
            </div>
            {{template "attachmentStrip" .Detail.Summary}}
          {{end}}
        </section>
        <section class="panel accounting">
          <h2>Stundenwerte</h2>
          <div class="legend" aria-label="Legende für Stundenwerte">
            <div><strong>Stunde</strong><span>Beginn der Abrechnungsstunde; jede Zeile umfasst diese Stunde.</span></div>
            <div><strong>Verbrauch</strong><span>Geschätzte kWh aus der Differenz der Zählerstände innerhalb dieser Stunde.</span></div>
            <div><strong>Ø aWATTar</strong><span>Stündlicher aWATTar-Arbeitspreis ohne Netzgebühr.</span></div>
            <div><strong>Strom</strong><span>Verbrauch × aWATTar-Preis.</span></div>
            <div><strong>Netzgeb.</strong><span>Verbrauch × in dieser Stunde gültige Netzgebühr.</span></div>
            <div><strong>Summe</strong><span>Strom plus Netzgebühr für diese Stunde.</span></div>
            <div><strong>Gewichtung</strong><span>Relative Balkenlänge im Vergleich zur teuersten Stunde des Monats.</span></div>
          </div>
          {{if .Detail.HasHours}}
            <div class="table-wrap" tabindex="0" role="region" aria-label="Stundenwerte Parkplatznutzung">
              <table>
                <caption class="sr-only">Stundenwerte Parkplatznutzung mit Verbrauch, aWATTar-Preis, Stromkosten, Netzgebühr, Summe und Gewichtung.</caption>
                <thead>
                  <tr>
                    <th scope="col" title="Beginn der Abrechnungsstunde; jede Zeile umfasst diese Stunde.">Stunde</th>
                    <th scope="col" class="num" title="Geschätzte kWh aus der Differenz der Zählerstände innerhalb dieser Stunde.">Verbrauch</th>
                    <th scope="col" class="num" title="Anteil der Stunde, der als PV-Überschuss zum Fixpreis abgerechnet wird.">☀️ Überschuss</th>
                    <th scope="col" class="num" title="Stündlicher aWATTar-Arbeitspreis ohne Netzgebühr.">Ø aWATTar</th>
                    <th scope="col" class="num" title="Verbrauch × aWATTar-Preis.">Strom</th>
                    <th scope="col" class="num" title="Verbrauch × in dieser Stunde gültige Netzgebühr.">Netzgeb.</th>
                    <th scope="col" class="num" title="Strom plus Netzgebühr für diese Stunde.">Summe</th>
                    <th scope="col" title="Relative Balkenlänge im Vergleich zur teuersten Stunde des Monats.">Gewichtung</th>
                  </tr>
                </thead>
                <tbody>
                  {{range .Detail.Hours}}
                    <tr>
                      <th scope="row" title="{{.AtTitle}}">{{.AtLabel}}</th>
                      <td class="num" title="{{.KWhTitle}}">{{.KWh}}</td>
                      <td class="num" title="{{.SurplusKWhTitle}}">{{if .HasSurplus}}{{.SurplusKWh}}{{else}}—{{end}}</td>
                      <td class="num" title="{{.AverageAwattarTitle}}">{{.AverageAwattar}}</td>
                      <td class="num" title="{{.EnergyCostTitle}}">{{.EnergyCost}}</td>
                      <td class="num" title="{{.GridCostTitle}}">{{.GridCost}}</td>
                      <td class="num amount" title="{{.TotalCostTitle}}">{{.TotalCost}}</td>
                      <td class="bar-cell" title="{{.WeightTitle}}"><div class="bar"><span style="width: {{.ChartPercent}}%;"></span></div></td>
                    </tr>
                  {{end}}
                </tbody>
              </table>
            </div>
          {{else}}
            <p class="empty">Für diesen Monat sind noch keine Stundenwerte gespeichert.</p>
          {{end}}
        </section>
        {{if .Detail.HasSessions}}
          <section class="panel">
            <h2>Ladevorgänge</h2>
            {{template "parkingSessionList" .Detail.Sessions}}
          </section>
        {{end}}
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "settingsHub"}}
{{template "appOpen" .}}
    <main class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg><span>/</span><span>Einstellungen</span></span>
      </div>
      <section class="page">
        <div>
          <h1>Einstellungen</h1>
          <p class="lede">Persönliche Einstellungen und Verwaltungsbereiche für {{.Tenant.Address}}.</p>
        </div>
        <div class="home-grid">
          <section class="panel">
            <div class="kicker">Konto</div>
            <div class="quick-list">
              <a class="quick-row" href="/app/settings/profile">
                <svg viewBox="0 0 24 24"><path d="M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8z"/><path d="M4.5 21a7.5 7.5 0 0 1 15 0"/></svg>
                <div><h3>Profil</h3><p>{{.Email}} · {{.Role}}</p></div>
                <span class="quick-arrow">›</span>
              </a>
              <a class="quick-row" href="/app/settings/notifications">
                <svg viewBox="0 0 24 24"><path d="M18 8a6 6 0 1 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9"/><path d="M10 21h4"/></svg>
                <div><h3>Benachrichtigungen</h3><p>E-Mail-Ereignisse pro Bereich steuern.</p></div>
                <span class="quick-arrow">›</span>
              </a>
            </div>
          </section>
          <section class="panel">
            <div class="kicker">Verwaltung</div>
            {{if or .CanManageUsers .CanManageBuilding .CanManageDocuments .CanManageHandovers .IsAdmin}}
              <div class="quick-list">
                {{if .CanManageBuilding}}<a class="quick-row" href="/app/settings/building">
                  <svg viewBox="0 0 24 24"><path d="M4 21V8l8-5 8 5v13"/><path d="M9 21v-7h6v7"/><path d="M8 10h.01M16 10h.01"/></svg>
                  <div><h3>Gebäude</h3><p>Adresse, Kontakt, Hero-Bild und Einheiten verwalten.</p></div>
                  <span class="quick-arrow">›</span>
                </a>{{end}}
                {{if .CanManageUsers}}
                <a class="quick-row" href="/app/settings/users">
                  <svg viewBox="0 0 24 24"><path d="M8.5 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6z"/><path d="M3.5 20a5 5 0 0 1 10 0"/><path d="M16 11.5a2.5 2.5 0 1 0 0-5"/><path d="M17 15a4 4 0 0 1 3.5 4"/></svg>
                  <div><h3>Benutzer &amp; Rechte</h3><p>Einladungen, Rollen und Zugriff der Hausgemeinschaft verwalten.</p></div>
                  <span class="quick-arrow">›</span>
                </a>
                <a class="quick-row" href="/app/settings/parking-access">
                  <svg viewBox="0 0 24 24"><path d="M5 16h14"/><path d="m7 16 1.5-5h7L17 16"/><path d="M7 16v3M17 16v3"/><path d="M7 19h1M16 19h1"/></svg>
                  <div><h3>Parkplatz-Zugriff</h3><p>Parkplatznutzung für Bewohner freigeben oder entziehen.</p></div>
                  <span class="quick-arrow">›</span>
                </a>
                {{end}}
                {{if .CanManageDocuments}}
                <a class="quick-row" href="/app/dokumente">
                  <svg viewBox="0 0 24 24"><path d="M7 3h7l3 3v15H7z"/><path d="M14 3v4h4"/><path d="M9 13h6M9 17h6"/></svg>
                  <div><h3>Dokumente</h3><p>Unterlagen hochladen, kategorisieren und Sichtbarkeit setzen.</p></div>
                  <span class="quick-arrow">›</span>
                </a>
                {{end}}
                {{if .CanManageHandovers}}
                <a class="quick-row" href="/app/uebergaben">
                  <svg viewBox="0 0 24 24"><path d="M7 4h10v16H7z"/><path d="M9.5 8h5M9.5 12h4"/><path d="m9.5 16 1.5 1.5 3.5-4"/></svg>
                  <div><h3>Übergaben</h3><p>Nutzerwechsel mit Räumen, Zählern, Schlüsseln, Fotos und Bestätigung dokumentieren.</p></div>
                  <span class="quick-arrow">›</span>
                </a>
                {{end}}
                {{if .CanViewAudit}}
                <a class="quick-row" href="/app/audit">
                  <svg viewBox="0 0 24 24"><path d="M5 4h14v16H5z"/><path d="M8 8h8M8 12h8M8 16h5"/></svg>
                  <div><h3>Audit-Log</h3><p>Sensible Aktionen und Änderungen im Portal nachvollziehen.</p></div>
                  <span class="quick-arrow">›</span>
                </a>
                {{end}}
                {{if .IsAdmin}}<a class="quick-row" href="/app/parking/settings">
                  <svg viewBox="0 0 24 24"><path d="M5 16h14"/><path d="m7 16 1.5-5h7L17 16"/><path d="M7 16v3M17 16v3"/><path d="M7 19h1M16 19h1"/></svg>
                  <div><h3>Parkplatz-Abrechnung</h3><p>Netzgebühr und Abrechnungswerte für die private Parkplatznutzung.</p></div>
                  <span class="quick-arrow">›</span>
                </a>{{end}}
              </div>
            {{else}}
              <p class="empty">Verwaltungsbereiche sind nur für berechtigte Personen sichtbar.</p>
            {{end}}
          </section>
        </div>
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "auditLog"}}
{{template "appOpen" .}}
    <main class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M5 4h14v16H5z"/><path d="M8 8h8M8 12h8M8 16h5"/></svg><span>/</span><span>Audit-Log</span></span>
        <div class="page-actions"><a class="button" href="/app/settings">Einstellungen</a></div>
      </div>
      <section class="page wide">
        <div>
          <h1>Audit-Log</h1>
          <p class="lede">Sensible Aktionen im Portal, begrenzt auf {{.Tenant.Address}}.</p>
        </div>
        <section class="panel audit-panel">
          <div class="metric-grid audit-summary-grid" aria-label="Audit-Überblick">
            <div class="metric-card"><span class="metric-label">Ereignisse</span><strong class="metric-value">{{.AuditStats.TotalEvents}}</strong><span class="mini">aktuelle Auswahl</span></div>
            <div class="metric-card"><span class="metric-label">Personen</span><strong class="metric-value">{{.AuditStats.ActorCount}}</strong><span class="mini">sichtbare Akteure</span></div>
            <div class="metric-card"><span class="metric-label">Heute</span><strong class="metric-value">{{.AuditStats.TodayCount}}</strong><span class="mini">Aktionen im Portal</span></div>
          </div>
          <details class="audit-filter-panel"{{if .AuditStats.HasActiveFilters}} open{{end}}>
            <summary><span>Filter</span><strong>{{.AuditStats.FilterSummary}}</strong></summary>
            <form class="filter-form audit-filter" method="get" action="/app/audit">
              <label for="audit-action">Aktion
                <select id="audit-action" name="action">
                  {{range .ActionOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}
                </select>
              </label>
              <label for="audit-search">Suche
                <input id="audit-search" type="search" name="q" value="{{.SearchQuery}}" placeholder="Person, Ziel oder Aktion">
              </label>
              <button class="button" type="submit">Filtern</button>
              {{if .AuditStats.HasActiveFilters}}<a class="button ghost" href="/app/audit">Zurücksetzen</a>{{end}}
            </form>
            {{if .AuditStats.HasActiveFilters}}
              <div class="audit-active-filters" aria-label="Aktive Filter">
                {{range .AuditStats.ActiveFilters}}<span class="chip"><strong>{{.Label}}:</strong> {{.Value}}</span>{{end}}
              </div>
            {{end}}
          </details>
          {{if .HasEvents}}
            <div class="audit-timeline" aria-label="Audit-Log">
              {{range .Events}}
                {{if .ShowDateHeader}}<h2 class="audit-day">{{.DateHeader}}</h2>{{end}}
                <article class="audit-row audit-{{.ActionTone}}">
                  <time class="audit-time" datetime="{{.AtISO}}"><strong>{{.AtTime}}</strong><span>{{.AtDate}}</span></time>
                  <span class="audit-marker" aria-label="{{.ToneLabel}}"></span>
					<div class="audit-main">
						<div class="audit-row-head">
							<div class="audit-action"><span class="pill audit-pill audit-{{.ActionTone}}">{{.ActionText}}</span><strong>{{.Summary}}</strong></div>
						</div>
                    <div class="audit-meta">
                      <span><strong>Wer</strong> {{.Actor}}{{if .ActorRole}} <em>{{.ActorRole}}</em>{{end}}</span>
                      <span><strong>Ziel</strong> {{if .HasTarget}}{{.Target}}{{else}}-{{end}}</span>
                    </div>
                    {{if .HasDetails}}
                      <details class="audit-details">
                        <summary>Details</summary>
                        <div class="chips">{{range .Details}}<span class="chip"><strong>{{.Key}}:</strong> {{.Value}}</span>{{end}}</div>
                      </details>
                    {{end}}
                  </div>
                </article>
              {{end}}
            </div>
          {{else}}
            {{template "emptyState" .EventsEmpty}}
          {{end}}
        </section>
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "buildingSettings"}}
{{template "appOpen" .}}
    <style>
      .building .building-grid { display: grid; grid-template-columns: minmax(0,1.15fr) minmax(320px,.85fr); gap: 22px; align-items: start; }
      .building .settings-card { max-width: none; }
	      .building .meta-form { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 12px; }
	      .building .meta-form .full, .building .unit-form .full { grid-column: 1 / -1; }
	      .building .meta-form .f-actions { grid-column: 1 / -1; display: flex; justify-content: flex-end; }
	      .building .brand-preview { grid-column: 1 / -1; display: grid; grid-template-columns: 58px minmax(0,1fr); gap: 13px; align-items: center; border: 1px solid var(--line); border-radius: 8px; padding: 12px; background: var(--panel-soft); }
	      .building .brand-preview-mark { width: 52px; height: 52px; border-radius: 8px; display: grid; place-items: center; color: var(--gold-ink); background: #fffefb; border: 1px solid var(--line); }
	      .building .brand-preview-mark svg { width: 39px; height: 34px; display: block; stroke: currentColor; stroke-width: 2.2; fill: none; stroke-linecap: round; stroke-linejoin: round; }
	      .building .brand-preview strong { display: block; font-family: var(--font-serif); font-size: 18px; }
	      .building .brand-preview span { display: block; margin-top: 3px; color: var(--muted); font-size: 13px; line-height: 1.35; }
	      .building textarea { min-height: 92px; }
      .building .hero-preview { width: 100%; aspect-ratio: 16 / 9; object-fit: cover; border: 1px solid var(--line); border-radius: 8px; background: var(--panel-soft); }
      .building .hero-form { display: grid; gap: 12px; }
      .building .hero-actions { display: flex; gap: 10px; align-items: center; flex-wrap: wrap; }
      .building .hero-actions .button { flex: 1 1 auto; }
      .building .hero-delete { margin: 0; }
	      .building .unit-panel { display: grid; gap: 18px; }
	      .building .unit-add, .building .unit-editor { border: 1px solid var(--line); border-radius: 8px; padding: 16px; background: var(--panel-soft); }
	      .building .unit-list { display: grid; gap: 12px; }
	      .building .unit-metrics { display: flex; gap: 8px; flex-wrap: wrap; }
	      .building .unit-metric { display: inline-flex; gap: 7px; align-items: baseline; border: 1px solid var(--line); border-radius: 8px; padding: 7px 10px; background: var(--panel); color: #6f6a5c; font-size: 13px; font-weight: 700; }
	      .building .unit-metric strong { color: var(--ink); font-size: 16px; }
	      .building .unit-form { display: grid; grid-template-columns: repeat(12,minmax(0,1fr)); gap: 10px; align-items: end; }
	      .building .unit-form .f-label { grid-column: span 4; }
	      .building .unit-form .f-type { grid-column: span 3; }
	      .building .unit-form .f-share { grid-column: span 3; }
	      .building .unit-form .f-owners, .building .unit-form .f-renters { grid-column: span 6; }
	      .building .unit-form .f-actions { grid-column: span 2; display: flex; gap: 8px; align-items: center; justify-content: flex-end; flex-wrap: wrap; }
		      .building .unit-delete { display: inline; margin: 0; }
		      .building .unit-summary { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; margin-bottom: 12px; }
		      .building .unit-summary strong { font-family: Spectral, serif; font-size: 20px; }
		      .building .unit-summary .pill.soft { background: var(--panel); color: #6f6a5c; }
		      .building .payment-status-panel { display: grid; gap: 14px; }
		      .building .payment-status-list { display: grid; gap: 10px; }
		      .building .payment-status-row { display: grid; grid-template-columns: minmax(180px,1fr) auto minmax(180px,240px) auto; gap: 12px; align-items: center; border: 1px solid var(--line); border-radius: 8px; padding: 13px 14px; background: var(--panel-soft); }
		      .building .payment-status-unit { display: grid; gap: 3px; min-width: 0; }
		      .building .payment-status-unit strong { font-family: Spectral, serif; font-size: 19px; line-height: 1.12; overflow-wrap: anywhere; }
		      .building .payment-status-unit span, .building .payment-status-meta { color: var(--muted); font-size: 12.5px; font-weight: 700; line-height: 1.35; }
			      .building .payment-status-form { display: grid; grid-template-columns: minmax(130px,1fr) auto; gap: 8px; align-items: end; }
			      .building .payment-status-form .sr-only { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip: rect(0,0,0,0); white-space: nowrap; border: 0; }
			      .building .payment-status-form select { min-height: 42px; }
		      @media (max-width: 960px) { .building .building-grid { grid-template-columns: 1fr; } }
		      @media (max-width: 760px) {
		        .building .meta-form, .building .unit-form { grid-template-columns: 1fr; }
		        .building .unit-form .f-label, .building .unit-form .f-type, .building .unit-form .f-share, .building .unit-form .f-owners, .building .unit-form .f-renters, .building .unit-form .f-actions { grid-column: 1 / -1; }
	        .building .meta-form .f-actions, .building .unit-form .f-actions { justify-content: stretch; }
	        .building .unit-form .f-actions .button { flex: 1 1 auto; }
	        .building .payment-status-row { grid-template-columns: 1fr; align-items: stretch; }
	        .building .payment-status-form { grid-template-columns: 1fr; gap: 10px; }
	      }
    </style>
    <script src="/assets/attachments.js?v={{.AssetVersion}}" defer></script>
    <main class="app-main building">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg><span>/</span><a href="/app/settings">Einstellungen</a><span>/</span><span>Gebäude</span></span>
        <div class="page-actions"><a class="button" href="/app/settings">Zurück zu Einstellungen</a></div>
      </div>
      <section class="page wide">
        <div>
          <h1>Gebäude</h1>
          <p class="lede">Stammdaten, Kontaktblock, Titelbild und Einheiten für {{.Tenant.Address}}.</p>
        </div>
        <div class="building-grid">
          <section class="panel settings-card">
            <div>
              <h2>Stammdaten</h2>
              <p class="muted">Diese Angaben überschreiben die Umgebungswerte für diesen Tenant.</p>
            </div>
            {{if .BuildingMsg}}<p class="flash {{if .BuildingOK}}ok{{end}}">{{.BuildingMsg}}</p>{{end}}
            <form class="meta-form" method="post" action="/app/settings/building">
              <label class="full" for="building-name">Name
                <input id="building-name" type="text" name="name" value="{{.Tenant.Name}}" maxlength="160" required>
              </label>
	              <label class="full" for="building-address">Adresse
	                <textarea id="building-address" name="address" maxlength="500" required>{{.Tenant.Address}}</textarea>
	              </label>
	              <div class="brand-preview">
	                <span class="brand-preview-mark">{{template "tenantBrandMark" .}}</span>
	                <div><strong>{{.BrandIconLabel}}</strong><span>{{.Tenant.BrandAbbreviation}} erscheint als kurze Kennung in der Seitenleiste.</span></div>
	              </div>
	              <label for="brand-icon">Portal-Symbol
	                <select id="brand-icon" name="brand_icon">
	                  {{range .BrandIconOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}
	                </select>
	              </label>
	              <label for="brand-abbreviation">Kurzkennung
	                <input id="brand-abbreviation" type="text" name="brand_abbreviation" value="{{.Tenant.BrandAbbreviation}}" maxlength="12" placeholder="JHW22">
	              </label>
	              <label for="contact-name">Verwalter Kontakt
	                <input id="contact-name" type="text" name="contact_name" value="{{.Tenant.ContactName}}" maxlength="160" placeholder="Name oder Firma">
	              </label>
              <label for="contact-email">Kontakt-E-Mail
                <input id="contact-email" type="email" name="contact_email" value="{{.Tenant.ContactEmail}}" maxlength="160" autocomplete="email">
              </label>
              <label for="contact-phone">Kontakt-Telefon
                <input id="contact-phone" type="tel" name="contact_phone" value="{{.Tenant.ContactPhone}}" maxlength="80" autocomplete="tel">
              </label>
              <label for="emergency-name">Notdienst
                <input id="emergency-name" type="text" name="emergency_name" value="{{.Tenant.EmergencyName}}" maxlength="160" placeholder="Notdienst">
              </label>
              <label for="emergency-phone">Notdienst-Telefon
                <input id="emergency-phone" type="tel" name="emergency_phone" value="{{.Tenant.EmergencyPhone}}" maxlength="80" autocomplete="tel">
              </label>
              <label for="caretaker-name">Hausmeister
                <input id="caretaker-name" type="text" name="caretaker_name" value="{{.Tenant.CaretakerName}}" maxlength="160">
              </label>
              <label for="caretaker-email">Hausmeister-E-Mail
                <input id="caretaker-email" type="email" name="caretaker_email" value="{{.Tenant.CaretakerEmail}}" maxlength="160" autocomplete="email">
              </label>
              <label for="caretaker-phone">Hausmeister-Telefon
                <input id="caretaker-phone" type="tel" name="caretaker_phone" value="{{.Tenant.CaretakerPhone}}" maxlength="80" autocomplete="tel">
              </label>
              <div class="f-actions"><button class="button primary" type="submit">Stammdaten speichern</button></div>
            </form>
          </section>

          <aside class="panel settings-card">
            <div>
              <h2>Hero-Bild</h2>
              <p class="muted">Das Bild erscheint auf der Startseite und im App-Banner.</p>
            </div>
            <img class="hero-preview" src="{{.Tenant.HeroImageURL}}" alt="">
            {{if .HeroMsg}}<p class="flash {{if .HeroOK}}ok{{end}}">{{.HeroMsg}}</p>{{end}}
            <form class="hero-form" method="post" action="/app/settings/building/hero" enctype="multipart/form-data">
              <label for="hero-image">Bilddatei
                <span class="file-control"><input id="hero-image" type="file" name="hero_image" accept="image/jpeg,image/png,image/webp" required><span>Bild auswählen</span></span>
              </label>
              <div class="hero-actions">
                <button class="button primary" type="submit">Hero-Bild speichern</button>
              </div>
              <span class="mini">JPG, PNG oder WebP bis 5 MB.</span>
            </form>
            {{if .HasCustomHero}}
              <form class="hero-delete" method="post" action="/app/settings/building/hero/delete" data-confirm="Hero-Bild entfernen und Standardbild verwenden?">
                <button class="button ghost" type="submit">Standardbild verwenden</button>
              </form>
            {{end}}
          </aside>
        </div>

        <section class="panel unit-panel">
	          <div class="section-head">
	            <div>
	              <h2>Einheiten</h2>
	              <p class="muted">Wohnungen, Geschäftslokale und zugehörige Objekte. Nur abrechenbare Wohneinheiten zählen für Fair Use.</p>
	            </div>
	            <div class="unit-metrics" aria-label="Einheiten Übersicht">
	              <span class="unit-metric"><strong>{{.UnitTotal}}</strong> Einträge</span>
	              <span class="unit-metric"><strong>{{.BillableUnits}}</strong> von {{.FairUseFreeUnits}} {{.BillableLabel}} (Fair Use)</span>
	            </div>
	            {{if .FairUseExceeded}}<p class="muted">Über dem kostenlosen Rahmen von {{.FairUseFreeUnits}} Wohneinheiten — Richtwert 1 € pro Einheit und Monat.</p>{{end}}
	          </div>
	          {{if .UnitMsg}}<p class="flash {{if .UnitOK}}ok{{end}}">{{.UnitMsg}}</p>{{end}}
	          <div class="unit-add">
	            <div class="unit-summary"><strong>Neue Einheit</strong><span class="pill">Anlegen</span></div>
	            <form class="unit-form" method="post" action="/app/settings/building/units">
	              <label class="f-label">Einheit
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
          </div>
          {{if .Units}}
            <div class="unit-list">
	              {{range .Units}}
	                <article class="unit-editor">
	                  <div class="unit-summary"><strong>{{.Label}}</strong><span class="pill">{{.UnitTypeLabel}}</span><span class="pill soft">{{.BillableLabel}}</span><span class="pill soft">{{.Share}}</span></div>
	                  <form class="unit-form" method="post" action="/app/settings/building/units">
	                    <input type="hidden" name="orig_id" value="{{.ID}}">
	                    <input type="hidden" name="id" value="{{.ID}}">
	                    <label class="f-label">Einheit
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
                    <div class="f-actions">
                      <button class="button primary" type="submit">Speichern</button>
                    </div>
                  </form>
                  <form class="unit-delete" method="post" action="/app/settings/building/units/delete">
                    <input type="hidden" name="id" value="{{.ID}}">
                    <button class="button small ghost" type="submit" aria-label="{{.DeleteConfirmLabel}}">Entfernen</button>
                  </form>
                </article>
              {{end}}
            </div>
	          {{else}}
	            {{template "emptyState" .UnitsEmpty}}
	          {{end}}
	        </section>

	        <section class="panel payment-status-panel">
	          <div class="section-head">
	            <div>
	              <h2>Zahlungsstatus</h2>
	              <p class="muted">Manuelle Transparenz pro Einheit. Keine Sollstellung, keine Buchung, kein Mahnwesen.</p>
	            </div>
	          </div>
	          {{if .PaymentMsg}}<p class="flash {{if .PaymentOK}}ok{{end}}">{{.PaymentMsg}}</p>{{end}}
	          {{if .HasPaymentRows}}
	            <div class="payment-status-list">
	              {{range .PaymentRows}}
	                <article class="payment-status-row">
	                  <div class="payment-status-unit">
	                    <strong>{{.UnitLabel}}</strong>
	                    <span>{{.UnitTypeLabel}}</span>
	                  </div>
	                  <span class="pill {{.StatusClass}}">{{.Status}}</span>
	                  <form class="payment-status-form" method="post" action="/app/settings/building/payment-status">
	                    <input type="hidden" name="unit_id" value="{{.UnitID}}">
	                    <label class="sr-only" for="payment-status-{{.UnitID}}">Status für {{.UnitLabel}}</label>
	                    <select id="payment-status-{{.UnitID}}" name="status">{{range .StatusOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}</select>
	                    <button class="button small" type="submit">Status speichern</button>
	                  </form>
	                  <span class="payment-status-meta">{{if .HasUpdatedAt}}{{.UpdatedAt}}{{else}}{{.Detail}}{{end}}</span>
	                </article>
	              {{end}}
	            </div>
	          {{else}}
	            {{template "emptyState" .UnitsEmpty}}
	          {{end}}
	        </section>
	      </section>
	    </main>
{{template "appClose" .}}
{{end}}

{{define "profileSettings"}}
{{template "appOpen" .}}
    <style>
      .profile .settings-card { max-width: 820px; display: grid; gap: 18px; }
      .profile .profile-flash { margin: 0; padding: 10px 13px; border-radius: 9px; font-size: 13.5px; font-weight: 600; border: 1px solid transparent; }
      .profile .profile-flash.ok { background: rgba(47,107,74,.12); color: var(--leaf); border-color: rgba(47,107,74,.25); }
      .profile .profile-flash.warn { background: rgba(150,40,40,.08); color: #9a2b2b; border-color: rgba(150,40,40,.22); }
      .profile .profile-form { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 12px; }
      .profile .profile-form .short { grid-column: span 1; }
      .profile .profile-form .full { grid-column: 1 / -1; }
      .profile .directory-check { grid-column: 1 / -1; min-height: 42px; display: flex; align-items: center; gap: 10px; color: var(--ink); font-size: 14px; font-weight: 700; letter-spacing: 0; text-transform: none; }
      .profile .directory-check input { width: auto; min-height: 0; }
      .profile .readonly-grid { display: grid; grid-template-columns: repeat(auto-fit,minmax(220px,1fr)); gap: 10px; }
      .profile .readonly-box { border: 1px solid var(--line); border-radius: 8px; padding: 13px; background: var(--panel-soft); display: grid; gap: 8px; }
      .profile .readonly-box strong { font-family: Spectral, serif; font-size: 18px; }
      .profile .chips { display: flex; flex-wrap: wrap; gap: 6px; }
      .profile .chip { display: inline-flex; align-items: center; border: 1px solid var(--line); background: var(--panel); color: #6f6a5c; border-radius: 8px; padding: 4px 10px; font-size: 12.5px; font-weight: 700; }
      .profile .unit-list { display: grid; gap: 8px; }
      .profile .unit-row { display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 10px; align-items: center; border-top: 1px solid var(--line); padding-top: 8px; }
      .profile .unit-row:first-child { border-top: 0; padding-top: 0; }
      @media (max-width: 680px) { .profile .profile-form { grid-template-columns: 1fr; } .profile .profile-form .short { grid-column: 1 / -1; } }
    </style>
    <main class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg><span>/</span><a href="/app/settings">Einstellungen</a><span>/</span><span>Profil</span></span>
        <div class="page-actions"><a class="button" href="/app/settings">Zurück zu Einstellungen</a></div>
      </div>
      <section class="page profile">
        <div>
          <h1>Profil</h1>
          <p class="lede">{{.Email}}</p>
        </div>
        <section class="panel settings-card">
          {{if .ProfileMsg}}<p class="profile-flash{{if .ProfileOK}} ok{{else}} warn{{end}}">{{.ProfileMsg}}</p>{{end}}
          <form class="profile-form" method="post" action="/app/settings/profile">
            <label class="short" for="profile-title">Titel<input id="profile-title" name="title" value="{{.Profile.Title}}" maxlength="40" autocomplete="honorific-prefix"></label>
            <label class="short" for="profile-phone">Telefon optional<input id="profile-phone" name="phone" value="{{.Profile.Phone}}" maxlength="80" autocomplete="tel"></label>
            <label for="profile-first">Vorname<input id="profile-first" name="first_name" value="{{.Profile.FirstName}}" maxlength="120" autocomplete="given-name"></label>
            <label for="profile-last">Nachname<input id="profile-last" name="last_name" value="{{.Profile.LastName}}" maxlength="120" autocomplete="family-name"></label>
            <label class="directory-check" for="profile-directory"><input id="profile-directory" type="checkbox" name="directory_opt_in"{{if .Profile.DirectoryOptIn}} checked{{end}}>Im Kontakte-Verzeichnis anzeigen</label>
            <div class="full row-actions">
              <button class="button primary" type="submit">Speichern</button>
              <a class="button" href="/app/settings">Abbrechen</a>
            </div>
          </form>
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
        </section>
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "notificationSettings"}}
{{template "appOpen" .}}
    <style>
      .notifications .panel { background: var(--panel); border: 1px solid var(--line); border-radius: 12px; padding: 22px; }
      .notifications .settings-card { max-width: 760px; display: grid; gap: 16px; }
      .notifications .notify-flash { margin: 0; padding: 10px 13px; border-radius: 9px; font-size: 13.5px; font-weight: 600; border: 1px solid transparent; }
      .notifications .notify-flash.ok { background: rgba(47,107,74,.12); color: var(--leaf); border-color: rgba(47,107,74,.25); }
      .notifications .notify-flash.warn { background: rgba(150,40,40,.08); color: #9a2b2b; border-color: rgba(150,40,40,.22); }
      .notifications .toggle-list { display: grid; gap: 10px; }
      .notifications .toggle-row { display: grid; grid-template-columns: auto minmax(0,1fr); gap: 11px; align-items: start; border: 1px solid var(--line); border-radius: 9px; padding: 13px; background: var(--panel-soft); color: var(--ink); }
      .notifications .toggle-row input { width: 18px; height: 18px; margin-top: 2px; accent-color: var(--gold); }
      .notifications .toggle-row strong { display: block; font-size: 14px; }
      .notifications .toggle-row span { display: block; color: var(--muted); font-size: 13px; line-height: 1.45; margin-top: 2px; }
      .notifications .actions { display: flex; flex-wrap: wrap; gap: 10px; align-items: center; }
    </style>
    <main class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg><span>/</span><a href="/app/settings">Einstellungen</a><span>/</span><span>Benachrichtigungen</span></span>
        <div class="page-actions"><a class="button" href="/app/settings">Zurück zu Einstellungen</a></div>
      </div>
      <section class="page notifications">
        <div>
          <h1>Benachrichtigungen</h1>
          <p class="lede">{{.Email}}</p>
        </div>
        <section class="panel settings-card">
          {{if .NotifyMsg}}<p class="notify-flash{{if .NotifyOK}} ok{{else}} warn{{end}}">{{.NotifyMsg}}</p>{{end}}
          <form class="form-grid" method="post" action="/app/settings/notifications">
            <div class="toggle-list full">
              <label class="toggle-row">
                <input type="checkbox" name="email_enabled" value="on"{{if .EmailNotificationsEnabled}} checked{{end}}>
                <span><strong>E-Mail-Benachrichtigungen</strong><span>Globale Zustellung für dieses Konto.</span></span>
              </label>
              {{range .NotificationEvents}}
              <label class="toggle-row">
                <input type="checkbox" name="events" value="{{.Key}}"{{if .Checked}} checked{{end}}>
                <span><strong>{{.Label}}</strong><span>{{.Description}}</span></span>
              </label>
              {{end}}
            </div>
            <div class="actions full">
              <button class="button primary" type="submit">Speichern</button>
              <a class="button" href="/app/settings">Abbrechen</a>
            </div>
          </form>
        </section>
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "parkingAccessSettings"}}
{{template "appOpen" .}}
    <style>
      .parking-access .panel { background: var(--panel); border: 1px solid var(--line); border-radius: 12px; padding: 22px; }
      .parking-access .access-table { width: 100%; border-collapse: collapse; }
      .parking-access .access-table th, .parking-access .access-table td { padding: 12px 10px; border-bottom: 1px solid var(--line); text-align: left; vertical-align: middle; }
      .parking-access .access-table th { color: var(--gold-ink); font-size: 11px; text-transform: uppercase; letter-spacing: .06em; }
      .parking-access .person { display: grid; gap: 2px; min-width: 0; }
      .parking-access .person strong { font-family: Spectral, serif; font-size: 17px; overflow-wrap: anywhere; }
      .parking-access .person span { color: var(--muted); font-size: 12.5px; overflow-wrap: anywhere; }
      .parking-access .actions { display: flex; justify-content: flex-end; gap: 8px; flex-wrap: wrap; }
      .parking-access .access-flash { margin: 0 0 14px; padding: 10px 13px; border-radius: 9px; font-size: 13.5px; font-weight: 600; border: 1px solid transparent; }
      .parking-access .access-flash.ok { background: rgba(47,107,74,.12); color: var(--leaf); border-color: rgba(47,107,74,.25); }
      .parking-access .access-flash.warn { background: rgba(150,40,40,.08); color: #9a2b2b; border-color: rgba(150,40,40,.22); }
      @media (max-width: 680px) {
        .parking-access .access-table, .parking-access .access-table tbody, .parking-access .access-table tr, .parking-access .access-table td { display: block; width: 100%; min-width: 0; }
        .parking-access .access-table thead { display: none; }
        .parking-access .access-table tr { border: 1px solid var(--line); border-radius: 8px; padding: 10px; margin-bottom: 10px; background: var(--panel-soft); }
        .parking-access .access-table td { border-bottom: 0; padding: 7px 0; }
        .parking-access .access-table td::before { content: attr(data-label); display: block; color: var(--gold-ink); font-size: 11px; font-weight: 800; text-transform: uppercase; letter-spacing: .06em; margin-bottom: 3px; }
        .parking-access .actions { justify-content: flex-start; }
      }
    </style>
    <main class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg><span>/</span><a href="/app/settings">Einstellungen</a><span>/</span><span>Parkplatz-Zugriff</span></span>
        <div class="page-actions"><a class="button" href="/app/settings">Zurück zu Einstellungen</a><form method="post" action="/app/parking/reminders"><input type="hidden" name="return_to" value="parking_access"><button class="button" type="submit">Erinnerungen senden</button></form></div>
      </div>
      <section class="page parking-access">
        <div>
          <h1>Parkplatz-Zugriff</h1>
          <p class="lede">Berechtigungen für die private Stellplatz- und Ladeabrechnung.</p>
        </div>
        <section class="panel">
          {{if .AccessMsg}}<p class="access-flash{{if .AccessOK}} ok{{else}} warn{{end}}">{{.AccessMsg}}</p>{{end}}
          {{if .HasAccessRows}}
            <table class="access-table" aria-label="Parkplatz-Zugriff">
              <thead><tr><th>Person</th><th>Rolle</th><th>Status</th><th>Offen</th><th>Quelle</th><th></th></tr></thead>
              <tbody>
                {{range .AccessRows}}
                <tr>
                  <td data-label="Person"><span class="person"><strong>{{.DisplayName}}</strong><span>{{.Email}}</span></span></td>
                  <td data-label="Rolle"><span class="role-pill {{.RoleClass}}">{{.Role}}</span></td>
                  <td data-label="Status">{{if .ParkingChecked}}<span class="pill ok">Freigegeben</span>{{else}}<span class="pill">Kein Zugriff</span>{{end}}</td>
                  <td data-label="Offen">{{if .HasOutstanding}}<span class="pill">{{.OutstandingBalance}}</span>{{else}}<span class="mini">-</span>{{end}}</td>
                  <td data-label="Quelle">{{if .Editable}}<span class="mini">Portal</span>{{else}}<span class="mini">Konfiguration</span>{{end}}</td>
                  <td data-label="Aktion">
                    <div class="actions">
                      {{if .ParkingChecked}}<a class="button small" href="/app/parking/export/{{$.StatementYear}}?user={{.Email}}">CSV</a>{{end}}
                      {{if and .Editable (or $.ServiceProviderAccessEnabled (ne .Role "Dienstleister")) (or $.IsAdmin (ne .Role "Admin"))}}
                        <form method="post" action="/app/settings/parking-access">
                          <input type="hidden" name="email" value="{{.Email}}">
                          <input type="hidden" name="parking" value="{{if .ParkingChecked}}0{{else}}1{{end}}">
                          <button class="button small" type="submit">{{if .ParkingChecked}}Entziehen{{else}}Freigeben{{end}}</button>
                        </form>
                      {{else}}
                        <span class="mini">Schreibgeschützt</span>
                      {{end}}
                    </div>
                  </td>
                </tr>
                {{end}}
              </tbody>
            </table>
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
      .pk-set { display: grid; gap: 24px; max-width: 760px; }
      .pk-set .panel { display: grid; gap: 16px; }
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
      .pk-chat .button { padding: 8px 14px; font-size: 13px; }
      @media (max-width: 620px) {
        .pk-fields, .pk-fields.cols-3, .pk-toggles { grid-template-columns: 1fr; }
        .pk-inline { grid-template-columns: 1fr; }
        .pk-actions { justify-content: stretch; }
        .pk-set .button { width: 100%; }
        .pk-actions .mini { margin-right: 0; }
      }
    </style>
    <main class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg><span>/</span><a href="/app/settings">Einstellungen</a><span>/</span><span>Parkplatz-Abrechnung</span></span>
        <div class="page-actions"><a class="button" href="/app/settings">Zurück zu Einstellungen</a><a class="button" href="/app/parking">Zur Parkplatznutzung</a></div>
      </div>
      <section class="page pk-set">
        <div>
          <h1>Parkplatz-Abrechnung</h1>
          <p class="lede">Tarif, Laderegelung und Benachrichtigungen für Parkplatz 20.</p>
        </div>

        <section class="panel settings-card">
          <div class="pk-head">
            <div>
              <h2>Tarif</h2>
              <p class="muted">Preise gelten ab dem Gültigkeitsdatum; ältere Monate behalten ihren Tarif.</p>
            </div>
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
            <div class="legend" aria-label="Tarifhistorie">
              {{range .Accounting.Tariffs}}
                <div><strong>{{.EffectiveFrom}}</strong><span>{{.GridFee}} · Basis {{.BaseFee}}</span></div>
              {{end}}
            </div>
          {{end}}
        </section>

        <section class="panel settings-card" id="laderegelung">
          <div class="pk-head">
            <div>
              <h2>Laderegelung</h2>
              <p class="muted">Automatisches PV-Überschussladen. Im Testbetrieb entscheidet und protokolliert der Regler, schaltet aber nicht.</p>
            </div>
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
            <div class="pk-group">
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
            <div class="pk-actions">
              <button class="button primary" type="submit">Laderegelung speichern</button>
            </div>
          </form>
        </section>

        <section class="panel settings-card">
          <div class="pk-head">
            <div>
              <h2>Regler-Status</h2>
              <p class="muted">Aktueller Zustand und die letzten Entscheidungen.</p>
            </div>
            <span class="pill">{{.Charging.State.PhaseLabel}}</span>
          </div>
          <div class="pk-strip">
            {{if .Charging.State.SinceLabel}}<span class="mini">seit {{.Charging.State.SinceLabel}}</span>{{end}}
            {{if .Charging.State.PollLabel}}<span class="mini">HA-Poll {{.Charging.State.PollLabel}}</span>{{end}}
            {{if .Charging.State.ShadowPill}}<span class="pill">Testbetrieb</span>{{end}}
            {{if .Charging.State.ErrorDetail}}<span class="pill dringend">{{.Charging.State.ErrorDetail}}</span>{{end}}
          </div>
          {{if .Charging.HasEvents}}
            <div class="pk-events" aria-label="Ereignisprotokoll">
              {{range .Charging.Events}}
                <div class="pk-event"><time>{{.AtLabel}}</time><span class="pill {{.KindClass}}">{{.KindLabel}}</span><span>{{.Detail}}</span></div>
              {{end}}
            </div>
          {{else}}
            <p class="empty">Noch keine Ereignisse seit dem letzten Neustart — sobald der Regler entscheidet, erscheint hier jede Aktion.</p>
          {{end}}
        </section>

        <section class="panel settings-card" id="telegram">
          <div class="pk-head">
            <div>
              <h2>Telegram-Bot</h2>
              <p class="muted">Benachrichtigungen und Befehle laufen über den eigenen Bot. Chats werden per Einmal-Code verknüpft; Chat-Kennungen bleiben auf dem Server.</p>
            </div>
            {{if .Charging.Telegram.Configured}}<span class="pill ok">Verbunden</span>{{else}}<span class="pill dringend">Kein Token</span>{{end}}
          </div>
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
            <button class="button" type="submit">Code erzeugen</button>
          </form>
          {{if .Charging.Telegram.HasChats}}
            <div class="pk-chats" aria-label="Verknüpfte Chats">
              {{range .Charging.Telegram.Chats}}
                <div class="pk-chat">
                  <span class="pk-avatar">{{printf "%.1s" .DisplayName}}</span>
                  <span class="pk-chat-meta"><strong>{{.DisplayName}}</strong><span>{{.Email}} · verknüpft seit {{.LinkedAt}}</span></span>
                  <form method="post" action="/app/parking/charging/telegram/unlink"><input type="hidden" name="chat_id" value="{{.ChatID}}"><button class="button" type="submit">Trennen</button></form>
                </div>
              {{end}}
            </div>
          {{else}}
            <p class="empty">Noch keine Chats verknüpft.</p>
          {{end}}
        </section>
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
      .users .popup-title { display: block; font-family: Spectral, serif; font-weight: 600; font-size: 18px; color: var(--ink); padding-bottom: 13px; border-bottom: 1px solid var(--line); }
      .users .popup-grid { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 0 24px; }
      .users .popup .permission { display: block; min-width: 0; padding: 14px 0; border-top: 1px solid var(--line); }
      .users .popup-grid .permission:nth-child(-n+3) { border-top: 0; }
      .users .popup .permission strong { display: block; font-family: Spectral, serif; font-weight: 600; font-size: 15px; color: var(--ink); margin-bottom: 4px; line-height: 1.2; }
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
      .users .row-edit { border: 1px solid transparent; background: transparent; border-radius: 8px; width: 32px; height: 32px; display: inline-grid; place-items: center; color: var(--soft); cursor: pointer; padding: 0; }
      .users .row-edit:hover { border-color: var(--line); background: var(--panel-soft); color: var(--gold-ink); }
      .users .row-edit svg { stroke: currentColor; fill: none; stroke-width: 1.8; stroke-linecap: round; stroke-linejoin: round; }
      .users .edit-dialog { position: relative; width: min(440px, 92vw); border: 1px solid var(--line); border-radius: 14px; padding: 22px; background: var(--panel); color: var(--ink); box-shadow: 0 30px 80px rgba(32,37,31,.32); }
      .users .edit-dialog::backdrop { background: rgba(32,37,31,.42); }
      .users .edit-dialog h2 { margin: 0 0 4px; font-family: Spectral, serif; font-weight: 600; font-size: 19px; }
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
	      @media (max-width: 760px) {
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
      @media (max-width: 760px) { .users .popup { left: -96px; width: min(620px, calc(100vw - 32px)); } .users .popup-grid { grid-template-columns: 1fr; } .users .popup-grid .permission { border-top: 1px solid var(--line); } .users .popup-grid .permission:first-child { border-top: 0; } }
    </style>
    <script src="/assets/users.js?v={{.AssetVersion}}" defer></script>
    <main class="app-main">
      <div class="content-top"><span class="crumb"><svg viewBox="0 0 24 24"><path d="M8.5 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6z"/><path d="M3.5 20a5 5 0 0 1 10 0"/><path d="M16 11.5a2.5 2.5 0 1 0 0-5"/><path d="M17 15a4 4 0 0 1 3.5 4"/></svg>Benutzer &amp; Rechte</span></div>
      <section class="page users">
	        <div>
	          <h1>Benutzer &amp; Rechte</h1>
	          <p class="lede">Lokale Verwaltung der eingeladenen E-Mail-Adressen und ihrer Rollen.</p>
	          {{if not .ServiceProviderAccessEnabled}}<p class="invite-flash warn">Datenschutzprüfung offen: Dienstleister-Zugänge können noch nicht angelegt oder geändert werden.</p>{{end}}
	        </div>
    <section class="panel stack">
      <div class="panel-head">
        <span class="kicker">Zugänge</span>
        <span class="count">{{len .Users}} {{if eq (len .Users) 1}}Person{{else}}Personen{{end}}</span>
      </div>

      <details class="disclosure invite-bar"{{if .InviteMsg}} open{{end}}>
        <summary>
          <span class="invite-plus"><svg viewBox="0 0 24 24" width="13" height="13" aria-hidden="true"><path d="M12 5.5v13M5.5 12h13" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round"/></svg></span>
          Person einladen
          <span class="summary-sub">Speichert &amp; lädt per E-Mail ein</span>
        </summary>
        <div class="disclosure-body">
          {{if .InviteMsg}}<p class="invite-flash{{if .InviteOK}} ok{{else}} warn{{end}}">{{.InviteMsg}}</p>{{end}}
          <p class="muted" style="margin-bottom:12px">Die eingeladene Person wird gespeichert und erhält eine E-Mail mit dem Anmelde-Link. Sie kann sich danach mit dieser Adresse anmelden.</p>
          <form class="invite-form" method="post" action="/app/settings/users">
            <input class="f-titel" type="text" name="title" placeholder="Titel" aria-label="Titel">
            <input class="f-vorname" type="text" name="first_name" placeholder="Vorname" aria-label="Vorname">
            <input class="f-nachname" type="text" name="last_name" placeholder="Nachname" aria-label="Nachname">
            <input class="f-email" type="email" name="email" placeholder="name@example.com" aria-label="E-Mail-Adresse" autocomplete="email" required>
            <select class="f-role" name="role" aria-label="Rolle">
              <option value="Mieter" data-preset-label="Standardzugriff" data-preset-permissions="">Mieter</option>
              <option value="Eigentümer" data-preset-label="Eigentümerzugriff" data-preset-permissions="">Eigentümer</option>
              <option value="Beirat" data-preset-label="Beiratszugriff" data-preset-permissions="">Beirat</option>
	              <option value="Verwalter" data-preset-label="Verwalterzugriff" data-preset-permissions="">Verwalter</option>
	              {{if .IsAdmin}}<option value="Admin" data-preset-label="Adminzugriff" data-preset-permissions="parking">Admin</option>{{end}}
	              {{if .ServiceProviderAccessEnabled}}<option value="Dienstleister" data-preset-label="Nur zugewiesene Anliegen" data-preset-permissions="">Dienstleister</option>{{end}}
	              <option value="Bewohner" data-preset-label="Bewohnerzugriff" data-preset-permissions="">Bewohner</option>
            </select>
            <fieldset class="permission-fieldset f-permissions">
              <legend>Sonderrechte</legend>
              <span class="preset-label" data-preset-label>Standardzugriff</span>
              <div class="permission-grid">
                <label class="permission-check"><input type="checkbox" name="permissions" value="parking" data-permission="parking"><strong>Parkplatznutzung</strong><span>Privater Bereich für Stellplatz- und Ladeabrechnung.</span></label>
              </div>
            </fieldset>
            <button class="f-submit" type="submit">Einladung senden</button>
          </form>
        </div>
      </details>

      <p class="muted roster-intro">Diese Liste kommt aus der Umgebungskonfiguration und den hier gespeicherten Einladungen.</p>

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
              <th>Rechte</th>
              <th>Anmeldung</th>
              <th class="col-status">Status</th>
              <th class="col-actions" aria-label="Aktionen"></th>
            </tr>
          </thead>
          <tbody>
            {{range .Users}}
            <tr>
              <td class="col-person" data-label="Person">
                <div class="person">
                  <span class="avatar">{{.Initials}}</span>
                  <div>
                    <div class="person-name">{{.DisplayName}}</div>
                    <div class="person-mail">{{.Email}}</div>
                    {{if .Phone}}<div class="person-mail">{{.Phone}}</div>{{end}}
                  </div>
                </div>
              </td>
              <td class="col-role" data-label="Rolle"><span class="pill {{.RoleClass}}"><span class="dot"></span>{{.Role}}</span><div class="role-caps">{{range .RoleCapabilities}}<span class="role-cap">{{.}}</span>{{end}}</div></td>
              <td data-label="Rechte"><div class="chips">{{range .PermissionList}}<span class="chip{{if eq . "Standard"}} plain{{end}}">{{.}}</span>{{end}}</div></td>
              <td data-label="Anmeldung"><div class="chips">{{range .AuthList}}<span class="chip">{{.}}</span>{{end}}</div></td>
              <td class="col-status" data-label="Status"><span class="pill {{if eq .Status "Aktiv"}}status-active{{else}}status-pending{{end}}"><span class="dot"></span>{{.Status}}</span>{{if .LastSeen}}<span class="last-seen">{{.LastSeen}}</span>{{end}}</td>
	              <td class="col-actions" data-label="">
	                {{if .Editable}}
	                {{if and (not $.ServiceProviderAccessEnabled) (eq .Role "Dienstleister")}}
	                <span class="mini">Schreibgeschützt</span>
	                {{else}}
	                <button type="button" class="row-edit" data-edit="{{.Email}}" aria-label="Bearbeiten" aria-haspopup="dialog" aria-controls="edit-{{.Email}}"><svg viewBox="0 0 24 24" width="16" height="16" aria-hidden="true"><path d="M4 20h4L18.5 9.5a2 2 0 0 0-2.83-2.83L5 17.2z"/><path d="M13.5 6.5 17 10"/></svg></button>
                <dialog id="edit-{{.Email}}" class="edit-dialog" aria-labelledby="edit-title-{{.Email}}">
                  <div class="dlg-x"><form method="dialog"><button aria-label="Schließen">&times;</button></form></div>
                  <h2 id="edit-title-{{.Email}}">Zugang bearbeiten</h2>
                  <p class="dlg-sub">{{.Email}}</p>
                  <form method="post" action="/app/settings/users/edit" class="dlg-form">
                    <input type="hidden" name="orig_email" value="{{.Email}}">
                    <input class="f-titel" type="text" name="title" value="{{.Title}}" placeholder="Titel" aria-label="Titel">
                    <input class="f-vorname" type="text" name="first_name" value="{{.FirstName}}" placeholder="Vorname" aria-label="Vorname">
                    <input class="f-nachname" type="text" name="last_name" value="{{.LastName}}" placeholder="Nachname" aria-label="Nachname">
                    <input class="f-email" type="email" name="email" value="{{.Email}}" aria-label="E-Mail-Adresse" autocomplete="email" required>
                    <select class="f-role" name="role" aria-label="Rolle">
                      <option value="Mieter" data-preset-label="Standardzugriff" data-preset-permissions=""{{if eq .Role "Mieter"}} selected{{end}}>Mieter</option>
                      <option value="Eigentümer" data-preset-label="Eigentümerzugriff" data-preset-permissions=""{{if eq .Role "Eigentümer"}} selected{{end}}>Eigentümer</option>
                      <option value="Beirat" data-preset-label="Beiratszugriff" data-preset-permissions=""{{if eq .Role "Beirat"}} selected{{end}}>Beirat</option>
	                      <option value="Verwalter" data-preset-label="Verwalterzugriff" data-preset-permissions=""{{if eq .Role "Verwalter"}} selected{{end}}>Verwalter</option>
	                      {{if $.IsAdmin}}<option value="Admin" data-preset-label="Adminzugriff" data-preset-permissions="parking"{{if eq .Role "Admin"}} selected{{end}}>Admin</option>{{end}}
	                      {{if $.ServiceProviderAccessEnabled}}<option value="Dienstleister" data-preset-label="Nur zugewiesene Anliegen" data-preset-permissions=""{{if eq .Role "Dienstleister"}} selected{{end}}>Dienstleister</option>{{end}}
	                      <option value="Bewohner" data-preset-label="Bewohnerzugriff" data-preset-permissions=""{{if eq .Role "Bewohner"}} selected{{end}}>Bewohner</option>
                    </select>
                    <fieldset class="permission-fieldset f-permissions">
                      <legend>Sonderrechte</legend>
                      <span class="preset-label" data-preset-label>Gespeicherte Rechte</span>
                      <div class="permission-grid">
                        <label class="permission-check"><input type="checkbox" name="permissions" value="parking" data-permission="parking"{{if .ParkingChecked}} checked{{end}}><strong>Parkplatznutzung</strong><span>Privater Bereich für Stellplatz- und Ladeabrechnung.</span></label>
                      </div>
                    </fieldset>
                    <button type="submit">Speichern</button>
                  </form>
                  <div class="dlg-delete">
                    <span>Dauerhaft entfernen</span>
                    <form method="post" action="/app/settings/users/delete" data-confirm="Diesen Zugang wirklich löschen?">
                      <input type="hidden" name="email" value="{{.Email}}">
                      <button type="submit" class="danger">Löschen</button>
                    </form>
                  </div>
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
`
