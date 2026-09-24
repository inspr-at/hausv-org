#!/usr/bin/env node
// Role-aware, stateful QA for the portal's most common paths.

import { existsSync, mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { chromium } from 'playwright';
import {
  assertEnergyMetricDisclosures,
  assertEnergyRedesignViewport,
  energyRedesignWidths,
} from './energy-redesign-contract.mjs';

const baseURL = process.argv[2];
if (!baseURL) {
  console.error('usage: qa-main-flows.mjs <baseURL>');
  process.exit(1);
}
const artifactDir = process.env.HV_QA_ARTIFACT_DIR?.trim();
const ciCore = process.env.HV_QA_CI_CORE === 'true';
const energyOnly = process.env.HV_QA_ENERGY_ONLY === 'true';
const fastQA = process.env.HV_QA_FAST === 'true';
const activeContexts = new Set();
const browserEvents = [];
const loginStorageStates = new Map();
if (artifactDir) mkdirSync(artifactDir, { recursive: true });

const personas = [
  { name: 'Bewohner', email: 'resident@example.com', manages: false },
  { name: 'Eigentümer', email: 'owner@example.com', manages: false },
  { name: 'Verwalter', email: 'verwalter@example.com', manages: true },
  { name: 'Admin', email: 'admin@example.com', manages: true },
];

const routes = [
  { path: '/app', heading: /Hallo /, content: 'Jetzt zu erledigen' },
  { path: '/app/announcements', heading: 'Aushang', content: 'QA Hausinformation' },
  { path: '/app/events', heading: 'Termine', content: 'QA Hausbegehung' },
  { path: '/app/kontakte', heading: 'Kontakte', content: 'QA Energiehilfe' },
  { path: '/app/dokumente', heading: 'Dokumente', content: 'QA Hausordnung' },
  { path: '/app/anliegen', heading: 'Anliegen', content: 'Anliegen' },
  { path: '/app/energie', heading: 'QA Zuhause', content: 'Nur beobachten' },
];

const portalChromeRoutes = [
  // HAUSV-704: organisation landings now use the same kit. Only the map's
  // portfolio presentation changes; geometry and action checks stay identical.
  { path: '/app/verwaltung', hero: false, overview: true },
  { path: '/app/verwaltung/posteingang', hero: false, overview: true, action: 'Telefonnotiz' },
  { path: '/app/verwaltung/textbausteine', hero: false, overview: true, action: 'Neuer Textbaustein' },
  { path: '/app/verwaltung/rechte', hero: false, overview: true },
  { path: '/app/verwaltung/einstellungen', hero: false, overview: true },
  { path: '/app', hero: true, action: 'Anliegen melden' },
  { path: '/app/energie', hero: false, action: 'Zuhause bearbeiten', energy: true },
  { path: '/app/announcements', hero: false, action: 'Aushang erstellen' },
  { path: '/app/events', hero: true, action: 'Termin erstellen' },
  { path: '/app/kontakte', hero: false, action: 'Kontakt hinzufügen' },
  { path: '/app/dokumente', hero: false, action: 'Hochladen' },
  { path: '/app/anliegen', hero: false, action: 'Triage-Board' },
  { path: '/app/anliegen/board', hero: false, boardContext: true },
  { path: '/app/abstimmungen', hero: false, action: 'Abstimmung anlegen' },
  { path: '/app/parking', hero: false, action: 'Mehr' },
  { path: '/app/uebergaben', hero: false },
  { path: '/app/settings/users', hero: false, action: 'Person einladen' },
  { path: '/app/audit', hero: false, action: 'Einstellungen' },
  { path: '/app/settings', hero: false },
  { path: '/app/hilfe', hero: false },
  { path: '/app/hilfe/energie', hero: false },
];

const executableCandidates = [
  process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH,
  ...(process.env.CI === 'true' ? [] : [
    '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
    '/Applications/Chromium.app/Contents/MacOS/Chromium',
    '/usr/bin/chromium',
    '/usr/bin/chromium-browser',
    '/usr/bin/google-chrome',
  ]),
].filter(Boolean);
const executablePath = executableCandidates.find(existsSync);
// Headless is the test contract, not merely Playwright's current default.
// Local debugging may opt into a visible browser, but CI stays headless even
// if a surrounding environment happens to set HV_QA_HEADLESS=false.
const headless = process.env.CI === 'true' || process.env.HV_QA_HEADLESS !== 'false';
const launchOptions = {
  headless,
  args: ['--host-resolver-rules=MAP hausv.test 127.0.0.1, MAP *.hausv.test 127.0.0.1', '--no-proxy-server'],
};
if (executablePath) {
  launchOptions.executablePath = executablePath;
}
const browser = await chromium.launch(launchOptions);

function fail(message) {
  throw new Error(message);
}

// HAUSV-697/701: the context bar reserves space above shell content at every
// width. On phones it also IS the navigation header; never add both heights.
async function contextBarBox(page) {
  const box = await page.locator('[data-context-bar]:visible').boundingBox();
  if (!box || box.height <= 0 || Math.abs(box.y) > 1) {
    fail(`Kontextleiste fehlt oder klebt nicht am Viewportrand (${JSON.stringify(box)})`);
  }
  return box;
}

async function toggleMobileMenu(page) {
  const summary = page.locator('[data-context-bar] > details.menu > summary');
  if (!(await summary.count())) fail('Mobiles Menü: kein Bedienelement gefunden');
  await summary.click();
}

async function assertHomeIdentityPair(page, scope, displayName, unitLabel, label) {
  const identity = page.locator(`[data-home-identity="${scope}"]`).first();
  if (!(await identity.count())) fail(`${label}: Zuhause-Identität fehlt`);
  const result = await identity.evaluate((root, expected) => {
    const primary = root.querySelector('[data-home-display-name]');
    const secondary = root.querySelector('[data-home-unit-label]');
    const primaryStyle = primary ? getComputedStyle(primary) : null;
    const secondaryStyle = secondary ? getComputedStyle(secondary) : null;
    const primaryRect = primary?.getBoundingClientRect();
    const secondaryRect = secondary?.getBoundingClientRect();
    return {
      primary: primary?.textContent?.trim() || '',
      secondary: secondary?.textContent?.trim() || '',
      inOrder: Boolean(primary && secondary && (primary.compareDocumentPosition(secondary) & Node.DOCUMENT_POSITION_FOLLOWING)),
      primaryFont: Number.parseFloat(primaryStyle?.fontSize || '0'),
      secondaryFont: Number.parseFloat(secondaryStyle?.fontSize || '0'),
      visible: Boolean(root.getClientRects().length),
      below: !primaryRect || !secondaryRect || !root.getClientRects().length || secondaryRect.top >= primaryRect.bottom - 1,
      // Since 0.69.0 the energy heading sets the unit beside the name on one
      // baseline; the sidebar keeps the stacked arrangement.
      beside: Boolean(primaryRect && secondaryRect && secondaryRect.left >= primaryRect.right - 1 &&
        secondaryRect.top < primaryRect.bottom && secondaryRect.bottom > primaryRect.top),
      aria: root.getAttribute('aria-label') || '',
      expected,
    };
  }, { displayName, unitLabel });
  if (result.primary !== displayName ||
      result.secondary !== unitLabel ||
      !result.inOrder ||
      result.secondaryFont >= result.primaryFont ||
      !result.visible ||
      !(result.below || result.beside) ||
      !result.aria.includes(displayName) ||
      !result.aria.includes(unitLabel)) {
    fail(`${label}: Anzeigename und offizielle Einheit sind nicht sauber hierarchisiert (${JSON.stringify(result)})`);
  }
}

async function localLogin(context, email, origin = baseURL) {
  const loginKey = `${origin}|${email}`;
  const cachedState = loginStorageStates.get(loginKey);
  if (cachedState) {
    await context.addCookies(cachedState.cookies);
    const cachedPage = await context.newPage();
    await cachedPage.goto(`${origin}/app`, { waitUntil: 'networkidle' });
    if (cachedPage.url().includes('/app')) return cachedPage;
    await cachedPage.close();
    loginStorageStates.delete(loginKey);
  }

  const page = await context.newPage();
  await page.goto(`${origin}/`, { waitUntil: 'networkidle' });
  const emailDetails = page.locator('details:has(form[action$="/auth/request"])');
  if (await emailDetails.count()) {
    await emailDetails.evaluate((element) => {
      element.open = true;
    });
  }
  await page.locator('input[name="email"]').fill(email);
  await page.locator('form[action$="/auth/request"] button[type="submit"]').click();
  const devLink = page.locator('a.dev-link');
  await devLink.waitFor({ state: 'visible', timeout: 10_000 });
  const href = await devLink.getAttribute('href');
  if (!href) fail(`Kein lokaler Anmeldelink für ${email}`);
  const target = new URL(href, origin);
  const localOrigin = new URL(origin);
  target.protocol = localOrigin.protocol;
  target.port = localOrigin.port;
  await page.goto(target.href, { waitUntil: 'networkidle' });
  if (!page.url().includes('/app')) fail(`Lokale Anmeldung für ${email} endete auf ${page.url()}`);
  loginStorageStates.set(loginKey, await context.storageState());
  return page;
}

function safePagePath(page) {
  try {
    const url = new URL(page.url());
    return url.pathname;
  } catch {
    return '/unknown';
  }
}

function recordBrowserEvent(kind, page, detail) {
  browserEvents.push(JSON.stringify({
    kind,
    page: safePagePath(page),
    detail: String(detail),
  }));
}

function watchPage(page) {
  page.on('console', (message) => {
    if (message.type() === 'error' || message.type() === 'warning') {
      recordBrowserEvent(`console.${message.type()}`, page, message.text());
    }
  });
  page.on('pageerror', (error) => recordBrowserEvent('pageerror', page, error.message));
  page.on('requestfailed', (request) => {
    const url = new URL(request.url());
    recordBrowserEvent('requestfailed', page, `${request.method()} ${url.pathname}: ${request.failure()?.errorText || 'unknown'}`);
  });
}

async function trackedContext(options) {
  const context = await browser.newContext(options);
  activeContexts.add(context);
  context.on('page', watchPage);
  if (artifactDir) {
    await context.tracing.start({ screenshots: true, snapshots: true, sources: true });
  }
  return context;
}

async function closeContext(context) {
  activeContexts.delete(context);
  await context.close();
}

function artifactLabel(value) {
  return value.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '') || 'page';
}

async function captureFailureArtifacts(error) {
  if (!artifactDir) return;

  const screenshotsDir = join(artifactDir, 'screenshots');
  const tracesDir = join(artifactDir, 'traces');
  mkdirSync(screenshotsDir, { recursive: true });
  mkdirSync(tracesDir, { recursive: true });
  writeFileSync(
    join(artifactDir, 'failure.txt'),
    `${error instanceof Error ? error.stack || error.message : String(error)}\n`,
  );

  let contextIndex = 0;
  for (const context of [...activeContexts]) {
    contextIndex += 1;
    let pageIndex = 0;
    for (const page of context.pages()) {
      pageIndex += 1;
      try {
        await page.screenshot({
          path: join(
            screenshotsDir,
            `${contextIndex}-${pageIndex}-${artifactLabel(safePagePath(page))}.png`,
          ),
          fullPage: true,
        });
      } catch (artifactError) {
        recordBrowserEvent('artifact.screenshot', page, artifactError);
      }
    }
    try {
      await context.tracing.stop({
        path: join(tracesDir, `context-${contextIndex}.zip`),
      });
    } catch (artifactError) {
      const page = context.pages()[0];
      if (page) recordBrowserEvent('artifact.trace', page, artifactError);
    }
  }

  writeFileSync(
    join(artifactDir, 'browser-console.log'),
    `${browserEvents.length ? browserEvents.join('\n') : 'Keine Browserfehler oder -warnungen aufgezeichnet.'}\n`,
  );
}

async function newContext(viewport) {
  return trackedContext({
    viewport,
    deviceScaleFactor: 1,
    locale: 'de-AT',
    timezoneId: 'Europe/Vienna',
  });
}

async function assertPortalChromeKit() {
  const context = await newContext({ width: 1440, height: 900 });
  const page = await localLogin(context, 'admin@example.com');

  for (const route of portalChromeRoutes) {
    const response = await page.goto(`${baseURL}${route.path}`, { waitUntil: 'networkidle' });
    if (!response || response.status() !== 200) {
      fail(`Portal-Chrome ${route.path}: Status ${response?.status() ?? 0}`);
    }
    const result = await page.evaluate((expected) => {
      const shell = document.querySelector('[data-portal-shell]');
      const landing = document.querySelector('[data-portal-section-landing]');
      const header = landing?.querySelector('[data-portal-section-header]');
      const hero = landing?.querySelector(':scope > [data-portal-section-hero]');
      const landingRect = landing?.getBoundingClientRect();
      const heroRect = hero?.getBoundingClientRect();
      const headerButtons = [...(header?.querySelectorAll('.button') || [])];
      const sidebar = shell?.querySelector('aside.sidebar');
      const mobileIdentity = document.querySelector('[data-context-bar] .context-scope .house-header-copy strong')?.textContent?.trim() || '';
      const switchRows = [...document.querySelectorAll('.switcher-row-copy small:last-child')]
        .map((node) => node.textContent?.trim() || '');
      const strip = landing?.querySelector(':scope > .energy-mode-strip');
      const stripStyle = strip ? getComputedStyle(strip) : null;
      const filledActions = headerButtons.filter((button) => {
        const style = getComputedStyle(button);
        return button.classList.contains('primary') ||
          !['transparent', 'rgba(0, 0, 0, 0)'].includes(style.backgroundColor);
      });
      const borderlessActions = headerButtons.filter((button) => {
        const style = getComputedStyle(button);
        return style.borderTopStyle === 'none' || Number.parseFloat(style.borderTopWidth) < 1;
      });
      return {
        shellCount: document.querySelectorAll('[data-portal-shell]').length,
        landingCount: document.querySelectorAll('[data-portal-section-landing]').length,
        headerCount: landing?.querySelectorAll('[data-portal-section-header]').length || 0,
        headerPlacement: expected.hero ? Boolean(header && hero?.contains(header)) : header?.parentElement === landing,
        declaredHero: landing?.getAttribute('data-portal-hero'),
        heroCount: landing?.querySelectorAll('[data-portal-section-hero]').length || 0,
        heroGap: heroRect && landingRect ? Math.round(heroRect.top - landingRect.top) : null,
        actionTexts: headerButtons.map((button) => button.textContent?.trim() || ''),
        filledActions: filledActions.map((button) => button.textContent?.trim() || ''),
        borderlessActions: borderlessActions.map((button) => button.textContent?.trim() || ''),
        wrappedActions: headerButtons.filter((button) => getComputedStyle(button).whiteSpace !== 'nowrap').length,
        sidebarMap: Boolean(sidebar?.querySelector(expected.overview ? '.side-map-portfolio' : '.side-map')),
        sidebarAddressCount: sidebar?.querySelectorAll('.side-address-label small').length || 0,
        headerAddressCount: document.querySelectorAll('[data-context-bar] .house-header-copy small').length,
        sidebarAccountRole: document.querySelector('[data-context-bar] .context-role')?.textContent?.trim() || '',
        mobileIdentity,
        switchRows,
        boardContext: Boolean(landing?.querySelector('.portal-section-context a[href$="/app/anliegen"]')),
        energyStrip: Boolean(strip),
        energyFlexWrap: stripStyle?.flexWrap || '',
        energyContainerType: stripStyle?.containerType || '',
        expected,
      };
    }, route);

    if (result.shellCount !== 1 || result.landingCount !== 1 ||
        result.headerCount !== 1 || !result.headerPlacement) {
      fail(`Portal-Chrome ${route.path}: Shared kit fehlt (${JSON.stringify(result)})`);
    }
    if (result.declaredHero !== String(route.hero) || result.heroCount !== (route.hero ? 1 : 0)) {
      fail(`Portal-Chrome ${route.path}: Hero-Vertrag verletzt (${JSON.stringify(result)})`);
    }
    if (route.hero && result.heroGap !== 0) {
      fail(`Portal-Chrome ${route.path}: Hero beginnt mit ${result.heroGap}px Abstand`);
    }
    if (route.action && !result.actionTexts.some((text) => text.includes(route.action))) {
      fail(`Portal-Chrome ${route.path}: Header-Aktion „${route.action}“ fehlt`);
    }
    // HAUSV-765: a header may carry exactly ONE filled main action; all others stay outline/ghost.
    if (result.filledActions.length > 1 || result.borderlessActions.length || result.wrappedActions) {
      fail(`Portal-Chrome ${route.path}: Header-Aktion ist gefüllt oder bricht um (${JSON.stringify(result)})`);
    }
    if (!result.sidebarMap || result.sidebarAddressCount !== 0 || result.headerAddressCount !== 1 || !result.sidebarAccountRole.includes('Admin')) {
      fail(`Portal-Chrome ${route.path}: Sidebar-Invarianten verletzt (${JSON.stringify(result)})`);
    }
    if (!result.mobileIdentity || !result.switchRows.length ||
        result.switchRows.some((row) => !/ · (Bewohner|Mieter|Eigentümer|Verwalter|Admin|Verwaltung)/.test(row))) {
      fail(`Portal-Chrome ${route.path}: Scope oder Rolle im gemeinsamen Wechsler fehlt (${JSON.stringify(result)})`);
    }
    if (route.boardContext && !result.boardContext) {
      fail(`Portal-Chrome ${route.path}: Board-Zurücklink fehlt im Kontext-Slot`);
    }
    if (route.energy && (!result.energyStrip || result.energyFlexWrap !== 'wrap' ||
        !result.energyContainerType.includes('inline-size'))) {
      fail(`Portal-Chrome ${route.path}: Energie-Strip-Wrap verloren (${JSON.stringify(result)})`);
    }
  }

  await closeContext(context);
  process.stdout.write(`  ✓ Gemeinsames Portal-Chrome · ${portalChromeRoutes.length - 1} Navigationseinträge + Anliegen-Board\n`);
}

// Eine überlaufende Seitenleiste ist auf einem Screenshot unsichtbar: die Seite
// sieht richtig aus, der Eintrag fehlt einfach. Genau das war der Fall — bei
// 900 Pixel Fensterhöhe, der verbreitetsten Notebook-Größe, waren "Verlauf" und
// "Einstellungen" für Eigentümer nicht erreichbar, ohne jeden Hinweis darauf.
async function assertSidebarNavReachable() {
  for (const height of [720, 800, 900, 1000, 1080]) {
    const context = await trackedContext({ viewport: { width: 1440, height }, locale: 'de-AT' });
    const page = await localLogin(context, 'owner@example.com');
    await page.goto(`${baseURL}/app`, { waitUntil: 'networkidle' });
    const result = await page.evaluate(() => {
      // The templ shell renders the sidebar as #portal-navigation; .side-nav was the
      // legacy renderer's class and matched nothing once the switch went live. Nobody
      // noticed because CI runs the energy-only subset of these flows, so this probe had
      // been failing silently for every full run since 0.96.0.
      const nav = document.querySelector('#portal-navigation');
      if (!nav) return { missing: true };
      const box = nav.getBoundingClientRect();
      const hidden = [...nav.querySelectorAll('.nav-item')]
        .filter((el) => {
          const r = el.getBoundingClientRect();
          return r.bottom > box.bottom + 1 || r.top < box.top - 1;
        })
        .map((el) => (el.textContent || '').trim());
      return { missing: false, hidden };
    });
    if (result.missing) fail(`Seitenleiste bei ${height}px: #portal-navigation fehlt`);
    if (result.hidden.length) {
      fail(`Seitenleiste bei 1440x${height}: nicht erreichbar — ${result.hidden.join(', ')}`);
    }
    await closeContext(context);
  }
  process.stdout.write('  ✓ Seitenleiste · alle Einträge erreichbar · 720–1080px\n');
}

// The legacy shell placed the portal switcher inside a sidebar map card, and this
// assertion used to encode that placement (.side-map-card, .side-foot, .side-map-top).
// The templ shell renders it as details.context-switch inside aside.sidebar (HAUSV-621 moved it
// below the house header, out of <header>; the exact nesting is not part of the contract) and a mobile twin,
// so those selectors matched nothing — and because CI runs only the energy subset of
// these flows, this had been failing silently since the switch went live. What is worth
// keeping is the BEHAVIOUR: switching portals updates URL, house name and the
// context-bar role atomically; switch rows explicitly name the target role.
// The switcher also remains usable at phone width.
async function assertPortalSwitcherAtomic() {
  const context = await trackedContext({ viewport: { width: 1440, height: 900 }, locale: 'de-AT' });
  const page = await localLogin(context, 'multi@example.com');
  await page.goto(`${baseURL}/demo/app`, { waitUntil: 'networkidle' });

  const picker = page.locator('[data-context-bar] details.house-picker[data-house-picker-shell="scope"]');
  if ((await picker.count()) !== 1) {
    fail(`Hauswechsler fehlt in der Seitenleiste oder ist mehrfach vorhanden (${await picker.count()})`);
  }
  if ((await picker.locator(':scope > summary .house-header-copy strong').textContent())?.trim() !== 'Demohaus') {
    fail('Hauswechsler zeigt den aktiven Hausnamen nicht eindeutig');
  }
  await picker.locator(':scope > summary').click();
  await picker.locator('form').filter({ has: page.locator('input[name="tenant"][value="haus-b"]') })
    .filter({ has: page.locator('input[name="role"][value="Admin"]') }).getByRole('button').click();
  await page.waitForLoadState('networkidle');
  const after = page.locator('[data-context-bar] details.house-picker[data-house-picker-shell="scope"] > summary .house-header-copy strong');
  const accountRole = page.locator('[data-context-bar] .context-role').first();
  if (new URL(page.url()).pathname !== '/haus-b/app' ||
      (await after.textContent())?.trim() !== 'Haus B' ||
      !(await accountRole.textContent())?.includes('Admin')) {
    fail(`Portalwechsel aktualisiert URL, Name oder Account-Rolle nicht atomar (${page.url()})`);
  }

  for (const width of [390, 320]) {
    await page.setViewportSize({ width, height: 844 });
    const menu = page.locator('[data-context-bar] > details.menu');
    if ((await menu.getAttribute('open')) === null) await menu.locator(':scope > summary').click();
    const mobileGeometry = await page.evaluate(() => {
      const summary = document.querySelector('[data-context-bar] details.house-picker[data-house-picker-shell="scope"] > summary');
      const box = summary?.getBoundingClientRect();
      return {
        overflow: document.documentElement.scrollWidth > window.innerWidth + 1,
        pickerVisible: Boolean(summary?.getClientRects().length),
        pickerContained: Boolean(box && box.left >= -1 && box.right <= window.innerWidth + 1),
      };
    });
    if (mobileGeometry.overflow || !mobileGeometry.pickerVisible || !mobileGeometry.pickerContained) {
      fail(`Hauswechsler ist bei ${width}px nicht stabil (${JSON.stringify(mobileGeometry)})`);
    }
  }
  await closeContext(context);
  process.stdout.write('  ✓ Hauswechsler · URL, Name und Account-Rolle atomar · Desktop und Mobil\n');
}

async function assertSharedAppShellNavigation() {
  let adminStorageState;
  for (const width of [320, 390, 430]) {
    const context = await trackedContext({ viewport: { width, height: 844 }, locale: 'de-AT' });
    const page = await localLogin(context, 'admin@example.com');
    await page.goto(`${baseURL}/app`, { waitUntil: 'networkidle' });
    adminStorageState = await context.storageState();

    await page.evaluate(() => document.activeElement instanceof HTMLElement && document.activeElement.blur());
    await page.keyboard.press('Tab');
    if (!(await page.locator('.skip-link').evaluate((link) => link === document.activeElement))) {
      fail(`App-Shell ${width}px: Sprunglink ist nicht das erste Tastaturziel`);
    }
    await page.keyboard.press('Enter');
    // The shared landing is the one focus target at every width. Wait for focus
    // to land instead of sampling the hash during the browser's focus update.
    await page.waitForFunction(() => document.activeElement && document.activeElement.hasAttribute('data-skip-target'));
    const skipResult = await page.evaluate(() => {
      const m = document.getElementById('main-content');
      return {
        hash: window.location.hash,
        active: document.activeElement?.id || document.activeElement?.tagName?.toLowerCase() || '',
        // Diagnostic context, so a failure says WHY focus did not land rather than only that
        // it did not: whether the target exists, is focusable, and is actually rendered.
        target: m ? (() => {
          const cs = getComputedStyle(m); const r = m.getBoundingClientRect();
          return { tabindex: m.getAttribute('tabindex'), display: cs.display, visibility: cs.visibility, w: Math.round(r.width), h: Math.round(r.height), inDoc: document.contains(m), count: document.querySelectorAll('#main-content').length, parent: m.parentElement?.className || m.parentElement?.tagName };
        })() : null,
      };
    });
    if (skipResult.active !== 'main-content') {
      fail(`App-Shell ${width}px: Sprunglink fokussiert den Inhalt nicht (${JSON.stringify(skipResult)})`);
    }

    await page.goto(`${baseURL}/app`, { waitUntil: 'networkidle' });
    const menu = page.locator('[data-context-bar] > details.menu');
    const summary = menu.locator(':scope > summary');
    const panel = menu.locator(':scope > .menu-panel');
    const navigation = panel.locator('nav[aria-label="Bereiche"]');
    // HAUSV-697/701 puts the house identity in the scope switcher and account
    // actions in their own native disclosure. Keep testing both header actions
    // and the closed navigation; the old direct identity/avatar links are gone.
    const identity = page.locator('[data-context-bar] > .context-scope > [data-switcher]').last().locator(':scope > summary');
    const account = page.locator('[data-context-bar] > [data-context-account]');
    const accountSummary = account.locator(':scope > summary');
    const accountPanel = account.locator(':scope > .context-account-menu');
    const avatar = accountSummary.locator('.avatar');
    const logout = accountPanel.getByRole('button', { name: 'Abmelden', exact: true });
    if ((await menu.count()) !== 1 || !(await summary.isVisible()) ||
        await menu.getAttribute('open') !== null || await panel.isVisible() ||
        await summary.getAttribute('aria-label') !== 'Navigation öffnen' ||
        !(await identity.isVisible()) || !(await avatar.isVisible()) ||
        await account.getAttribute('open') !== null || await accountPanel.isVisible()) {
      fail(`App-Shell ${width}px: natives mobiles Menü startet nicht geschlossen oder Kopfaktionen fehlen`);
    }

    await accountSummary.focus();
    await page.keyboard.press('Enter');
    if (await account.getAttribute('open') === null || !(await logout.isVisible()) ||
        !(await accountPanel.getByRole('link', { name: 'Profil', exact: true }).isVisible()) ||
        !(await accountPanel.getByRole('link', { name: 'Einstellungen', exact: true }).isVisible())) {
      fail(`App-Shell ${width}px: Enter öffnet die Kontoaktionen nicht`);
    }
    await page.keyboard.press('Escape');
    if (await account.getAttribute('open') !== null || await accountPanel.isVisible() ||
        !(await accountSummary.evaluate((element) => element === document.activeElement))) {
      fail(`App-Shell ${width}px: Escape schließt/fokussiert das Kontomenü nicht sauber`);
    }

    await summary.focus();
    await page.keyboard.press('Enter');
    if (await menu.getAttribute('open') === null || !(await panel.isVisible()) ||
        !(await navigation.isVisible())) {
      fail(`App-Shell ${width}px: Enter öffnet die Navigation nicht`);
    }

    const openGeometry = await page.evaluate(() => {
      const panelElement = document.querySelector('[data-context-bar] > details.menu > .menu-panel');
      return {
        pageOverflow: document.documentElement.scrollWidth > window.innerWidth + 1,
        panelOverflow: panelElement ? panelElement.scrollWidth > panelElement.clientWidth + 1 : null,
      };
    });
    if (openGeometry.pageOverflow || openGeometry.panelOverflow !== false) {
      fail(`App-Shell ${width}px: horizontales Überlaufen im offenen Menü (${JSON.stringify(openGeometry)})`);
    }

    await summary.focus();
    await page.keyboard.press('Enter');
    if (await menu.getAttribute('open') !== null || await panel.isVisible()) {
      fail(`App-Shell ${width}px: Enter schließt das native Menü nicht`);
    }
    await page.keyboard.press('Enter');
    if (await menu.getAttribute('open') === null || !(await panel.isVisible())) {
      fail(`App-Shell ${width}px: Enter öffnet das native Menü nicht erneut`);
    }
    await page.keyboard.press('Tab');
    // The location map leads the drawer and remains keyboard-accessible.
    const mapLink = panel.locator('[data-house-map-link]');
    if (await mapLink.count()) {
      if (!(await mapLink.evaluate((link) => link === document.activeElement))) {
        fail(`App-Shell ${width}px: Tab erreicht die vorangestellte Standortkarte nicht`);
      }
      await page.keyboard.press('Tab');
    }
    const tabReachedNavigation = await navigation.evaluate((nav) => nav.contains(document.activeElement));
    if (!tabReachedNavigation) fail(`App-Shell ${width}px: Tab erreicht nach der Standortkarte nicht die Navigation`);

    const navEntries = navigation.locator('a');
    const navEntryCount = await navEntries.count();
    if (!navEntryCount) fail(`App-Shell ${width}px: offenes Menü enthält keine Navigationseinträge`);
    for (let index = 0; index < navEntryCount; index += 1) {
      const entry = navEntries.nth(index);
      const inCollapsed = await entry.evaluate((element) => {
        for (let node = element.parentElement; node; node = node.parentElement) {
          if (node.classList && node.classList.contains('menu-panel')) return false;
          if (node.tagName === 'DETAILS' && !node.open) return true;
        }
        return false;
      });
      if (inCollapsed) continue;
      await entry.focus();
      await entry.evaluate((element) => element.scrollIntoView({ block: 'nearest', inline: 'nearest' }));
      const reachability = await entry.evaluate((element) => {
        const panelElement = element.closest('.menu-panel');
        if (!panelElement) return { missingPanel: true };
        // Content of a collapsed disclosure (the Liegenschaft picker) is not part
        // of the visible navigation; the summary that opens it is checked instead.
        for (let node = element.parentElement; node && node !== panelElement; node = node.parentElement) {
          if (node.tagName === 'DETAILS' && !node.open) {
            return { skipped: true, focused: true, visible: true, verticallyReachable: true, horizontallyContained: true };
          }
        }
        const item = element.getBoundingClientRect();
        const viewport = panelElement.getBoundingClientRect();
        return {
          focused: document.activeElement === element,
          visible: item.width > 0 && item.height > 0,
          verticallyReachable: item.top >= viewport.top - 1 && item.bottom <= viewport.bottom + 1,
          horizontallyContained: item.left >= -1 && item.right <= window.innerWidth + 1,
          label: (element.textContent || '').trim(),
        };
      });
      if (!reachability.focused || !reachability.visible || !reachability.verticallyReachable ||
          !reachability.horizontallyContained) {
        fail(`App-Shell ${width}px: Navigationseintrag nicht erreichbar (${JSON.stringify(reachability)})`);
      }
    }

    await page.keyboard.press('Escape');
    // Closing a <details> and the panel losing its rendered box are not synchronous with
    // the keypress; sampling getClientRects() the same instant reads the previous frame.
    // Wait for the panel to actually be gone, then assert — the same fix the energy Escape
    // probe needed, for the same reason. If it never goes, the wait fails and says so.
    await page.waitForFunction(() => {
      const d = document.querySelector('[data-context-bar] > details.menu');
      return d && !d.open && !d.querySelector(':scope > .menu-panel')?.getClientRects().length;
    }, undefined, { timeout: 3000 }).catch(() => {});
    const escapeResult = await page.evaluate(() => {
      const details = document.querySelector('[data-context-bar] > details.menu');
      const summaryElement = details?.querySelector(':scope > summary');
      return {
        open: details?.open,
        focused: summaryElement === document.activeElement,
        // "Visible" means a user can see or reach it. A closed <details> keeps its panel out
        // of hit-testing and paint even though Chromium still lays out a box for it (measured:
        // 40x516, hitTest null, no open attribute), so getClientRects() alone over-reports.
        // Ask whether anything in the panel can actually be hit at its own position.
        panelVisible: (() => { const pnl = details?.querySelector(':scope > .menu-panel'); if (!pnl) return false;
          if (details.open) return true;
          const r = pnl.getBoundingClientRect(); if (!r.width || !r.height) return false;
          const hit = document.elementFromPoint(Math.min(r.left + 5, innerWidth - 1), Math.min(r.top + 5, innerHeight - 1));
          return Boolean(hit && pnl.contains(hit)); })(),
      };
    });
    if (escapeResult.open || !escapeResult.focused || escapeResult.panelVisible) {
      fail(`App-Shell ${width}px: Escape schließt/fokussiert nicht sauber (${JSON.stringify(escapeResult)})`);
    }

    await summary.click();
    if (await menu.getAttribute('open') === null || !(await panel.isVisible())) {
      fail(`App-Shell ${width}px: Zeiger öffnet das native Menü nicht`);
    }
    await summary.click();
    if (await menu.getAttribute('open') !== null || await panel.isVisible()) {
      fail(`App-Shell ${width}px: Zeiger schließt das native Menü nicht`);
    }

    // Closing an overlay from outside it is a shell behaviour, not a selector or
    // placement choice. Keep this strict even though the surviving <details>
    // implementation may need an explicit enhancement to provide it.
    await summary.click();
    const outsideResult = await page.evaluate(() => {
      const main = [...document.querySelectorAll('[data-skip-target]')]
        .find((element) => element.getClientRects().length);
      main?.focus();
      main?.click();
      return {
        open: document.querySelector('[data-context-bar] > details.menu')?.open,
        active: document.activeElement?.id || '',
      };
    });
    if (outsideResult.open || outsideResult.active !== 'main-content') {
      fail(`App-Shell ${width}px: Außenklick schließt nicht ohne Fokusdiebstahl (${JSON.stringify(outsideResult)})`);
    }

    if (width === 390 && process.env.HV_QA_SCREENSHOT_DIR) {
      mkdirSync(process.env.HV_QA_SCREENSHOT_DIR, { recursive: true });
      await summary.click();
      await page.screenshot({
        path: join(process.env.HV_QA_SCREENSHOT_DIR, 'app-shell-mobile-semantic-menu.png'),
        fullPage: false,
      });
    }
    await closeContext(context);
  }

  for (const width of [901, 1024, 1440]) {
    const context = await trackedContext({ viewport: { width, height: 844 }, locale: 'de-AT' });
    const page = await localLogin(context, 'admin@example.com');
    await page.goto(`${baseURL}/app`, { waitUntil: 'networkidle' });
    const desktopState = await page.evaluate(() => ({
      mobileHeaderVisible: Boolean(document.querySelector('[data-context-bar] .context-navigation')?.getClientRects().length),
      navigationVisible: Boolean(document.querySelector('aside.sidebar > nav[aria-label="Bereiche"]')?.getClientRects().length),
      accountVisible: Boolean(document.querySelector('[data-context-bar] [data-context-account]')?.getClientRects().length),
    }));
    if (desktopState.mobileHeaderVisible || !desktopState.navigationVisible || !desktopState.accountVisible) {
      fail(`App-Shell ${width}px: Desktop-Navigation ist nicht vollständig sichtbar (${JSON.stringify(desktopState)})`);
    }
    await closeContext(context);
  }

  const noScriptContext = await trackedContext({
    viewport: { width: 390, height: 844 },
    locale: 'de-AT',
    javaScriptEnabled: false,
    storageState: adminStorageState,
  });
  const noScriptPage = await noScriptContext.newPage();
  const noScriptResponse = await noScriptPage.goto(`${baseURL}/app`, { waitUntil: 'domcontentloaded' });
  if (!noScriptResponse || noScriptResponse.status() !== 200) {
    fail(`App-Shell ohne JavaScript: Status ${noScriptResponse?.status() ?? 0}`);
  }
  const noScriptState = await noScriptPage.evaluate(() => ({
    menuOpen: document.querySelector('[data-context-bar] > details.menu')?.open,
    summaryVisible: Boolean(document.querySelector('[data-context-bar] > details.menu > summary')?.getClientRects().length),
    panelVisible: (() => { const d = document.querySelector('[data-context-bar] > details.menu'); const pnl = d?.querySelector(':scope > .menu-panel'); if (!pnl) return false;
      if (d.open) return true; const r = pnl.getBoundingClientRect(); if (!r.width || !r.height) return false;
      const hit = document.elementFromPoint(Math.min(r.left + 5, innerWidth - 1), Math.min(r.top + 5, innerHeight - 1)); return Boolean(hit && pnl.contains(hit)); })(),
  }));
  if (noScriptState.menuOpen || !noScriptState.summaryVisible || noScriptState.panelVisible) {
    fail(`App-Shell ohne JavaScript: natives Menü startet nicht bedienbar geschlossen (${JSON.stringify(noScriptState)})`);
  }
  const noScriptMenu = noScriptPage.locator('[data-context-bar] > details.menu');
  await noScriptMenu.locator(':scope > summary').click();
  if (await noScriptMenu.getAttribute('open') === null ||
      !(await noScriptMenu.locator(':scope > .menu-panel nav[aria-label="Bereiche"]').isVisible())) {
    fail('App-Shell ohne JavaScript: natives Menü lässt sich nicht öffnen');
  }
  await closeContext(noScriptContext);
  process.stdout.write('  ✓ App-Shell · Sprunglink, Tastaturmenü, Escape & No-JS · 320–1440px\n');
}

async function ensureDialogContact() {
  const context = await newContext({ width: 1440, height: 900 });
  const page = await localLogin(context, 'admin@example.com');
  await page.goto(`${baseURL}/app/kontakte`, { waitUntil: 'networkidle' });
  if (!(await page.getByText('QA Dialogkontakt', { exact: true }).count())) {
    const contactPanel = page.locator('#contact-add');
    if (!(await contactPanel.evaluate((element) => element.open))) {
      await contactPanel.locator(':scope > summary').click();
    }
    const form = contactPanel.locator('form');
    await form.locator('select[name="kind"]').selectOption({ index: 1 });
    await form.locator('input[name="name"]').fill('QA Dialogkontakt');
    await form.locator('input[name="phone"]').fill('+43 316 111111');
    await form.getByRole('button', { name: 'Kontakt hinzufügen', exact: true }).click();
    await page.waitForURL(/\/app\/kontakte/);
  }
  await closeContext(context);
}

async function assertBoundedAdminDialogs() {
  await ensureDialogContact();
  const viewports = [
    { width: 320, height: 568 },
    { width: 390, height: 667 },
    { width: 430, height: 844 },
    { width: 1024, height: 600 },
  ];

  for (const viewport of viewports) {
    const label = `${viewport.width}x${viewport.height}`;
    const context = await newContext(viewport);
    const page = await localLogin(context, 'admin@example.com');

    await page.goto(`${baseURL}/app/kontakte`, { waitUntil: 'networkidle' });
    const contactRow = page.locator('.contact-row').filter({ hasText: 'QA Dialogkontakt' }).first();
    const contactTrigger = contactRow.getByRole('button', { name: 'Bearbeiten' });
    // Diagnostic: if the click cannot happen, say what is in the way rather than time out.
    const clickable = await contactTrigger.evaluate((btn) => {
      btn.scrollIntoView({ block: 'center' });
      const r = btn.getBoundingClientRect();
      const cs = getComputedStyle(btn);
      const hit = document.elementFromPoint(r.left + r.width / 2, r.top + r.height / 2);
      return { w: Math.round(r.width), h: Math.round(r.height), top: Math.round(r.top), display: cs.display, vis: cs.visibility, op: cs.opacity,
        covered: hit === btn || btn.contains(hit) ? null : (() => { const hr = hit.getBoundingClientRect();
          // walk up to name the select's context: its label text, its form, and its own box
          const lbl = hit.closest('label')?.textContent?.trim().slice(0, 30); const frm = hit.closest('form')?.className || hit.closest('form')?.id || 'no-form';
          const dlg = hit.closest('dialog'); return { tag: hit.tagName.toLowerCase(), name: hit.getAttribute('name'), label: lbl, form: frm,
            inDialog: dlg ? (dlg.id + ' open=' + dlg.open) : null, box: [Math.round(hr.left), Math.round(hr.top), Math.round(hr.width), Math.round(hr.height)],
            zi: getComputedStyle(hit).zIndex, pos: getComputedStyle(hit).position }; })(),
        inViewport: r.top >= 0 && r.bottom <= innerHeight };
    }).catch((e) => ({ error: String(e).slice(0, 120) }));
    if (clickable.error || clickable.covered || !clickable.w) {
      fail(`Kontaktliste ${label}: Bearbeiten ist nicht klickbar (${JSON.stringify(clickable)})`);
    }
    await contactTrigger.click();
    // templ renders the contact editor as dialog.dialog with a per-contact id
    // (contacts.templ:228); .contact-edit-dialog was the legacy renderer's class.
    const contactDialog = page.locator('dialog.dialog[open]');
    await contactDialog.waitFor({ state: 'visible' });
    // Exercise the full editor, including the now-collapsed note/profile fields,
    // so every viewport still verifies a scrolling body and a stationary footer.
    await contactDialog.locator('details.contact-add-optional').evaluate((element) => { element.open = true; });
    const contactGeometry = await contactDialog.evaluate(async (dialog, mobile) => {
      const head = dialog.querySelector(':scope > .dialog-head');
      const body = dialog.querySelector(':scope > .dialog-body');
      const footer = dialog.querySelector(':scope > .dialog-footer');
      const form = body?.querySelector('form[id$="-form"]');
      const submit = footer?.querySelector('button[type="submit"]');
      const close = head?.querySelector('.dialog-close');
      const before = footer?.getBoundingClientRect();
      if (body) body.scrollTop = body.scrollHeight;
      await new Promise((resolve) => requestAnimationFrame(resolve));
      const outer = dialog.getBoundingClientRect();
      const headBox = head?.getBoundingClientRect();
      const bodyBox = body?.getBoundingClientRect();
      const footerBox = footer?.getBoundingClientRect();
      const submitBox = submit?.getBoundingClientRect();
      const closeBox = close?.getBoundingClientRect();
      return {
        outerClient: dialog.clientHeight,
        outerScroll: dialog.scrollHeight,
        bodyClient: body?.clientHeight || 0,
        bodyScroll: body?.scrollHeight || 0,
        direct: head?.parentElement === dialog && body?.parentElement === dialog && footer?.parentElement === dialog,
        rowsMeet: Boolean(headBox && bodyBox && footerBox &&
          headBox.top >= outer.top - 1 && bodyBox.top >= headBox.bottom - 1 && footerBox.top >= bodyBox.bottom - 1 &&
          footerBox.bottom <= outer.bottom + 1),
        footerShift: before && footerBox ? Math.abs(before.top - footerBox.top) : 999,
        submitHeight: submitBox?.height || 0,
        submitInside: Boolean(submitBox && footerBox && submitBox.bottom <= footerBox.bottom + 1),
        closeHeight: closeBox?.height || 0,
        formAssociation: Boolean(form && submit && submit.form === form &&
          submit.getAttribute('form') === form.getAttribute('id')),
        mobile,
      };
    }, viewport.width <= 600);
    if (contactGeometry.outerScroll > contactGeometry.outerClient + 1 ||
        contactGeometry.bodyScroll <= contactGeometry.bodyClient + 1 ||
        !contactGeometry.direct || !contactGeometry.rowsMeet || contactGeometry.footerShift > 1 ||
        !contactGeometry.submitInside || !contactGeometry.formAssociation ||
        (contactGeometry.mobile && (contactGeometry.submitHeight < 43.5 || contactGeometry.closeHeight < 43.5))) {
      fail(`Kontaktdialog ${label}: Kopf, Scrollkörper oder Aktion ist nicht stabil (${JSON.stringify(contactGeometry)})`);
    }
    if (viewport.width === 390 && process.env.HV_QA_SCREENSHOT_DIR) {
      mkdirSync(process.env.HV_QA_SCREENSHOT_DIR, { recursive: true });
      await page.screenshot({
        path: join(process.env.HV_QA_SCREENSHOT_DIR, 'contact-dialog-bounded-mobile.png'),
        fullPage: false,
      });
    }
    await page.keyboard.press('Escape');
    await contactDialog.waitFor({ state: 'hidden' });
    if (!(await contactTrigger.evaluate((trigger) => trigger === document.activeElement))) {
      fail(`Kontaktdialog ${label}: Escape gibt den Fokus nicht an den Auslöser zurück`);
    }

    await page.goto(`${baseURL}/app/uebergaben`, { waitUntil: 'networkidle' });
    const handoverTrigger = page.getByRole('button', { name: 'Übergabe anlegen' }).first();
    await handoverTrigger.click();
    const handoverDialog = page.locator('#handover-create[open]');
    await handoverDialog.waitFor({ state: 'visible' });
    const handoverGeometry = await handoverDialog.evaluate(async (dialog, mobile) => {
      const shell = dialog.querySelector(':scope > form');
      const head = shell?.querySelector(':scope > .dialog-head');
      const body = shell?.querySelector(':scope > .dialog-body');
      const footer = shell?.querySelector(':scope > .dialog-footer');
      const submit = footer?.querySelector('button[type="submit"]');
      const close = head?.querySelector('.dialog-close');
      const before = footer?.getBoundingClientRect();
      if (body) body.scrollTop = body.scrollHeight;
      await new Promise((resolve) => requestAnimationFrame(resolve));
      const outer = dialog.getBoundingClientRect();
      const headBox = head?.getBoundingClientRect();
      const bodyBox = body?.getBoundingClientRect();
      const footerBox = footer?.getBoundingClientRect();
      const submitBox = submit?.getBoundingClientRect();
      const closeBox = close?.getBoundingClientRect();
      return {
        outerClient: dialog.clientHeight,
        outerScroll: dialog.scrollHeight,
        bodyClient: body?.clientHeight || 0,
        bodyScroll: body?.scrollHeight || 0,
        direct: shell?.parentElement === dialog && head?.parentElement === shell && body?.parentElement === shell && footer?.parentElement === shell,
        rowsMeet: Boolean(headBox && bodyBox && footerBox &&
          headBox.top >= outer.top - 1 && bodyBox.top >= headBox.bottom - 1 && footerBox.top >= bodyBox.bottom - 1 &&
          footerBox.bottom <= outer.bottom + 1),
        footerShift: before && footerBox ? Math.abs(before.top - footerBox.top) : 999,
        submitHeight: submitBox?.height || 0,
        submitInside: Boolean(submitBox && footerBox && submitBox.bottom <= footerBox.bottom + 1),
        closeHeight: closeBox?.height || 0,
        mobile,
      };
    }, viewport.width <= 600);
    if (handoverGeometry.outerScroll > handoverGeometry.outerClient + 1 ||
        handoverGeometry.bodyScroll <= handoverGeometry.bodyClient + 1 ||
        !handoverGeometry.direct || !handoverGeometry.rowsMeet || handoverGeometry.footerShift > 1 ||
        !handoverGeometry.submitInside ||
        (handoverGeometry.mobile && (handoverGeometry.submitHeight < 43.5 || handoverGeometry.closeHeight < 43.5))) {
      fail(`Übergabedialog ${label}: Kopf, Scrollkörper oder Aktion ist nicht stabil (${JSON.stringify(handoverGeometry)})`);
    }

    const files = handoverDialog.locator('details.dialog-optional').filter({ hasText: 'Fotos und PDF' });
    await files.evaluate((details) => { details.open = true; });
    await handoverDialog.locator('#handover-attachments').focus();
    const fileFocus = await handoverDialog.evaluate((dialog) => {
      const body = dialog.querySelector(':scope > form > .dialog-body');
      const footer = dialog.querySelector(':scope > form > .dialog-footer');
      const input = dialog.querySelector('#handover-attachments');
      const control = input?.closest('.file-control');
      const bodyBox = body?.getBoundingClientRect();
      const footerBox = footer?.getBoundingClientRect();
      const controlBox = control?.getBoundingClientRect();
      return {
        active: input === document.activeElement,
        visible: Boolean(bodyBox && footerBox && controlBox && controlBox.top >= bodyBox.top - 1 &&
          controlBox.bottom <= bodyBox.bottom + 1 && controlBox.bottom <= footerBox.top + 1),
      };
    });
    if (!fileFocus.active || !fileFocus.visible) {
      fail(`Übergabedialog ${label}: fokussierte Dateiauswahl liegt unter der festen Aktion (${JSON.stringify(fileFocus)})`);
    }
    if (viewport.width === 390 && process.env.HV_QA_SCREENSHOT_DIR) {
      await page.screenshot({
        path: join(process.env.HV_QA_SCREENSHOT_DIR, 'handover-dialog-bounded-mobile.png'),
        fullPage: false,
      });
    }
    await page.keyboard.press('Escape');
    await handoverDialog.waitFor({ state: 'hidden' });
    if (!(await handoverTrigger.evaluate((trigger) => trigger === document.activeElement))) {
      fail(`Übergabedialog ${label}: Escape gibt den Fokus nicht an den Auslöser zurück`);
    }
    await closeContext(context);
  }
  process.stdout.write('  ✓ Admin-Dialoge · feste Aktionen, Scrollkörper & Fokus · 320x568–1024x600\n');
}

async function ensureResponsiveAnnouncement() {
  const context = await newContext({ width: 1440, height: 900 });
  const page = await localLogin(context, 'admin@example.com');
  await page.goto(`${baseURL}/app/announcements`, { waitUntil: 'networkidle' });
  const title = 'QA Responsive Aushang mit einem absichtlich sehr langen Titel für die gesamte Hausgemeinschaft';
  if (!(await page.getByText(title, { exact: true }).count())) {
    await page.getByRole('button', { name: 'Aushang erstellen' }).first().click();
    const form = page.locator('#announcement-create form');
    await form.locator('input[name="title"]').fill(title);
    await form.locator('textarea[name="body"]').fill('Dieser Aushang prüft Aktionszeile und Inhaltsbreite ohne abgeschnittene Bedienelemente. Er ist bewusst länger als die Vorschau, damit das Aufklappen, das Einklappen und die Fokusrückgabe weiterhin geprüft werden (HAUSV-662).');
    await form.getByRole('button', { name: 'Aushang veröffentlichen' }).click();
    await page.waitForURL(/\/app\/announcements/);
  }
  const entry = page.locator('.announcement-card').filter({ hasText: title });
  const details = entry.locator('.announcement-body');
  if (!(await details.evaluate((node) => node.open))) await details.locator('summary').click();
  await entry.getByRole('button', { name: 'Einklappen', exact: true }).click();
  if (await details.evaluate((node) => node.open) ||
      !(await details.locator('summary').evaluate((node) => node === document.activeElement))) {
    fail('Aushang: Einklappen schließt den Eintrag nicht mit Fokus zurück auf der Zeile');
  }
  await page.keyboard.press('Enter');
  await entry.getByRole('button', { name: 'Bearbeiten', exact: true }).click();
  const edit = page.getByRole('dialog', { name: 'Aushang bearbeiten' });
  if (await edit.getByLabel('Titel', { exact: true }).inputValue() !== title) {
    fail('Aushang: Bearbeiten öffnet nicht den gewählten Beitrag');
  }
  await page.keyboard.press('Escape');
  await closeContext(context);
}

async function assertResponsiveAdminWidths() {
  await ensureResponsiveAnnouncement();
  const context = await newContext({ width: 1440, height: 900 });
  const page = await localLogin(context, 'admin@example.com');
  await page.goto(`${baseURL}/app/announcements`, { waitUntil: 'networkidle' });

  for (const width of [320, 390, 430, 760, 768, 899, 900, 1024, 1440]) {
    await page.setViewportSize({ width, height: 900 });
    const result = await page.evaluate(async (phone) => {
      await new Promise((resolve) => requestAnimationFrame(resolve));
      // The feature body stays page-owned: .feed contains announcement cards,
      // while shell/header geometry comes from the shared portal kit.
      const feed = document.querySelector('.feed');
      if (!feed) return { missing: true };
      const style = getComputedStyle(feed);
      const feedBox = feed.getBoundingClientRect();
      const contentLeft = feedBox.left + Number.parseFloat(style.borderLeftWidth) + Number.parseFloat(style.paddingLeft);
      const contentRight = feedBox.right - Number.parseFloat(style.borderRightWidth) - Number.parseFloat(style.paddingRight);
      const clipped = [...feed.querySelectorAll(':scope > .section-head, :scope > .archive-tools, .announcement-card')]
        .filter((element) => element.getClientRects().length)
        .filter((element) => {
          const box = element.getBoundingClientRect();
          return box.left < contentLeft - 1 || box.right > contentRight + 1;
        })
        .map((element) => `${element.tagName.toLowerCase()}.${element.className}`);
      const badHeads = [...feed.querySelectorAll('.announcement-card .announcement-card-header')]
        .filter((head) => {
          const entry = head.closest('.announcement-card');
          const entryStyle = getComputedStyle(entry);
          const entryBox = entry.getBoundingClientRect();
          const headBox = head.getBoundingClientRect();
          const left = entryBox.left + Number.parseFloat(entryStyle.borderLeftWidth) + Number.parseFloat(entryStyle.paddingLeft);
          const right = entryBox.right - Number.parseFloat(entryStyle.borderRightWidth) - Number.parseFloat(entryStyle.paddingRight);
          return headBox.left < left - 1 || headBox.right > right + 1;
        }).length;
      const offscreenControls = [...document.querySelectorAll('a, button, input, select, textarea, summary')]
        .filter((element) => element.getClientRects().length)
        .filter((element) => {
          const box = element.getBoundingClientRect();
          return box.left < -1 || box.right > window.innerWidth + 1;
        })
        .map((element) => (element.textContent || element.getAttribute('aria-label') || element.tagName).trim().slice(0, 50));
      const headDisplays = [...feed.querySelectorAll('.announcement-card .announcement-card-header')]
        .map((head) => getComputedStyle(head).display);
      const tools = [...feed.querySelectorAll('.archive-tools > form, .archive-tools > nav, .announcement-sort')]
        .map((node) => node.getBoundingClientRect());
      const toolsInOneRow = tools.length === 3 && tools.every((box) => Math.abs(box.top - tools[0].top) <= 2);
      const shortTargets = [...feed.querySelectorAll('button, summary, .filter-tab, select')]
        .filter((node) => node.getClientRects().length && node.getBoundingClientRect().height < 43.5)
        .map((node) => (node.textContent || node.getAttribute('aria-label') || node.tagName).trim().slice(0, 50));
      return {
        missing: false,
        documentWidth: document.documentElement.scrollWidth,
        viewportWidth: window.innerWidth,
        feedColumns: style.gridTemplateColumns.trim().split(/\s+/).length,
        clipped,
        badHeads,
        offscreenControls,
        toolsInOneRow,
        shortTargets,
        listPanels: feed.querySelectorAll('.announcement-list').length,
        phone,
        headsStructured: headDisplays.length > 0 && headDisplays.every((display) => ['flex', 'grid'].includes(display)),
      };
    // "Phone" is where the shared shell collapses to a single column:
    // max-width 760px (portal.templ). The legacy renderer collapsed at 1180px and
    // this probe still carried that number, so it demanded phone layout at 768 and 1024 —
    // widths the current design deliberately keeps as tablet/desktop.
    }, width <= 760);
    if (result.missing || result.documentWidth > result.viewportWidth + 1 ||
        result.clipped.length || result.badHeads || result.offscreenControls.length ||
        result.shortTargets.length || result.listPanels !== 1 ||
        (width >= 900 && !result.toolsInOneRow) ||
        (result.phone && (result.feedColumns !== 1 || !result.headsStructured))) {
      fail(`Aushang ${width}px: Inhalt oder Aktionen werden abgeschnitten (${JSON.stringify(result)})`);
    }
    if (width === 390 && process.env.HV_QA_SCREENSHOT_DIR) {
      mkdirSync(process.env.HV_QA_SCREENSHOT_DIR, { recursive: true });
      await page.screenshot({
        path: join(process.env.HV_QA_SCREENSHOT_DIR, 'announcements-mobile-contained.png'),
        fullPage: true,
      });
    }
  }

  await page.goto(`${baseURL}/app/settings/building?section=units`, { waitUntil: 'networkidle' });
  for (const width of [901, 920, 959, 1024, 1050, 1075, 1100, 1120, 1280, 1440]) {
    await page.setViewportSize({ width, height: 900 });
    const result = await page.evaluate(async () => {
      await new Promise((resolve) => requestAnimationFrame(resolve));
      const pageBox = document.querySelector('main.building')?.getBoundingClientRect();
      const unitsBox = document.querySelector('.building .workspace')?.getBoundingClientRect();
      return {
        missing: !pageBox || !unitsBox,
        documentWidth: document.documentElement.scrollWidth,
        viewportWidth: window.innerWidth,
        rightEdges: [pageBox?.right || 0, unitsBox?.right || 0],
      };
    });
    if (result.missing || result.documentWidth > result.viewportWidth + 1 ||
        result.rightEdges.some((right) => right > result.viewportWidth + 1)) {
      fail(`Gebäude-Kontext ${width}px: Desktop-Shell-Naht läuft über (${JSON.stringify(result)})`);
    }
    if ((width === 1024 || width === 1440) && process.env.HV_QA_SCREENSHOT_DIR) {
      await page.locator('.building .workspace').screenshot({
        path: join(process.env.HV_QA_SCREENSHOT_DIR, `building-context-${width}.png`),
      });
    }
  }
  await closeContext(context);
  process.stdout.write('  ✓ Responsive Verwaltung · Aushang 320–1440px · Gebäude 901–1440px\n');
}

async function assertPublicLanding(viewport) {
  const context = await trackedContext({
    viewport: viewport.size,
    deviceScaleFactor: 1,
    locale: 'de-AT',
    timezoneId: 'Europe/Vienna',
  });
  const page = await context.newPage();
  const publicURL = new URL(baseURL);
  publicURL.hostname = 'hausv.test';
  const response = await page.goto(publicURL.href, { waitUntil: 'networkidle' });
  if (!response || response.status() !== 200) {
    fail(`Öffentliche Startseite ${viewport.name}: Status ${response?.status() ?? 0}`);
  }
  if (!(await page.getByRole('heading', { name: 'Ein Hausportal. Alles, was Menschen und Gebäude verbindet.' }).count())) {
    fail(`Öffentliche Startseite ${viewport.name}: Hauptaussage fehlt`);
  }
  const paths = await page.locator('.product-path').count();
  if (paths !== 3) fail(`Öffentliche Startseite ${viewport.name}: ${paths} statt 3 Produktwege`);
  for (const text of [
    'HAUSV Free',
    'HAUSV Home',
    'HAUSV Professional',
    '12 Monate kostenlos',
  ]) {
    if (!(await page.getByText(text, { exact: true }).count())) {
      fail(`Öffentliche Startseite ${viewport.name}: „${text}“ fehlt`);
    }
  }
  // HAUSV-664: the Professional path teases the fee model, and the former
  // pricing section is gone for good. Non-breaking spaces are normalised so the
  // oracle compares the words a reader sees.
  const feeTeaser = await page.locator('.product-path.professional .product-path-price').evaluate((element) => ({
    headline: (element.querySelector('strong')?.textContent ?? '').replace(/\u00a0/g, ' ').trim(),
    note: (element.querySelector('span')?.textContent ?? '').replace(/\u00a0/g, ' ').trim(),
  }));
  if (feeTeaser.headline !== '0 € Grundgebühr' ||
      feeTeaser.note !== '25 WE kostenlos · danach Verrechnung je WE / Monat') {
    fail(`Öffentliche Startseite ${viewport.name}: Preis-Teaser lautet ${JSON.stringify(feeTeaser)}`);
  }
  if (await page.locator('#preise, .cost-section, .offer-card, a[href="#preise"]').count()) {
    fail(`Öffentliche Startseite ${viewport.name}: der entfernte Preisabschnitt ist zurück`);
  }
  if (await page.getByRole('heading', { name: 'Klein starten. Erst mit dem Nutzen wachsen.' }).count()) {
    fail(`Öffentliche Startseite ${viewport.name}: die entfernte Preis-Überschrift ist zurück`);
  }
  if (process.env.HV_QA_SCREENSHOT_DIR) {
    mkdirSync(process.env.HV_QA_SCREENSHOT_DIR, { recursive: true });
    await page.screenshot({
      path: join(process.env.HV_QA_SCREENSHOT_DIR, `landing-${viewport.name.toLowerCase()}.png`),
      fullPage: true,
    });
  }
  if (viewport.name === 'Desktop') {
    const cardRows = await page.locator('.product-path').evaluateAll((cards) => cards.map((card) => {
      const box = (selector) => {
        const element = card.querySelector(selector);
        const rect = element?.getBoundingClientRect();
        return rect ? { top: rect.top, height: rect.height, center: rect.top + rect.height / 2 } : null;
      };
      return {
        name: card.querySelector('h3')?.textContent?.trim() || 'Unbenannte Produktkarte',
        head: box('.product-path-head'),
        headCopy: box('.product-path-head > div'),
        description: box(':scope > p'),
        capabilities: box('.product-capabilities'),
        price: box('.product-path-price'),
        action: box('.product-path-start'),
        icon: box('.product-path-icon'),
      };
    }));
    for (const row of ['head', 'description', 'capabilities', 'price', 'action']) {
      const missing = cardRows.filter((card) => !card[row]).map((card) => card.name);
      if (missing.length) {
        fail(`Öffentliche Startseite Desktop: Produktkarten-Zeile „${row}“ fehlt bei ${missing.join(', ')}`);
      }
      const tops = new Set(cardRows.map((card) => card[row].top.toFixed(2)));
      const heights = new Set(cardRows.map((card) => card[row].height.toFixed(2)));
      if (tops.size !== 1 || heights.size !== 1) {
        const positions = cardRows.map((card) => `${card.name}: ${card[row].top.toFixed(2)}px / ${card[row].height.toFixed(2)}px`).join(', ');
        fail(`Öffentliche Startseite Desktop: Produktkarten-Zeile „${row}“ ist nicht ausgerichtet (${positions})`);
      }
    }
    const uncentered = cardRows.filter((card) => !card.icon || !card.headCopy || Math.abs(card.icon.center - card.headCopy.center) > 0.5);
    if (uncentered.length) {
      fail(`Öffentliche Startseite Desktop: Produkt-Icons sind nicht mittig zum Titelblock (${uncentered.map((card) => card.name).join(', ')})`);
    }
  }
  // HAUSV-668: every feature card carries the same hairline, and its image
  // fills the media box to that hairline — no card ground leaking in a corner.
  const featureCards = await page.$$eval('.feature-card', (cards) => cards.map((card) => {
    const visual = card.querySelector('.feature-visual');
    const image = visual.querySelector('img');
    const cardBox = card.getBoundingClientRect();
    const visualBox = visual.getBoundingClientRect();
    const imageBox = image.getBoundingClientRect();
    const cardStyle = getComputedStyle(card);
    return {
      hairline: `${cardStyle.borderTopWidth} ${cardStyle.borderTopColor}`,
      radius: cardStyle.borderTopLeftRadius,
      fit: getComputedStyle(image).objectFit,
      inset: Math.max(Math.abs(visualBox.left - cardBox.left), Math.abs(visualBox.top - cardBox.top)),
      letterbox: Math.max(
        Math.abs(imageBox.left - visualBox.left), Math.abs(imageBox.top - visualBox.top),
        Math.abs(imageBox.right - visualBox.right), Math.abs(imageBox.bottom - visualBox.bottom),
      ),
    };
  }));
  if (featureCards.length !== 10) {
    fail(`Öffentliche Startseite ${viewport.name}: ${featureCards.length} Feature-Karten statt 10`);
  }
  const hairlines = new Set(featureCards.map((card) => `${card.hairline} ${card.radius}`));
  if (hairlines.size !== 1) {
    fail(`Öffentliche Startseite ${viewport.name}: uneinheitliche Kartenränder ${JSON.stringify([...hairlines])}`);
  }
  const leaking = featureCards.filter((card) => card.fit !== 'cover' || card.letterbox > 0.5 || card.inset > 1.5);
  if (leaking.length) {
    fail(`Öffentliche Startseite ${viewport.name}: Bildfläche reicht nicht bis zum Kartenrand ${JSON.stringify(leaking)}`);
  }
  const productDetails = page.locator('details.landing-more').first();
  if (await productDetails.evaluate((element) => element.open)) {
    fail(`Öffentliche Startseite ${viewport.name}: Ausblick ist ungefragt offen`);
  }
  await productDetails.locator('summary').click();
  for (const heading of ['Verrechnung: gemeinsam mit Friendly Customers', 'Energie: kontrolliert statt unbedacht']) {
    if (!(await page.getByRole('heading', { name: heading }).count())) {
      fail(`Öffentliche Startseite ${viewport.name}: „${heading}“ fehlt im Ausblick`);
    }
  }
  for (const stale of ['Kein Verrechnungssystem', 'Keine Jahresabrechnung oder Buchhaltung', 'Was bewusst nicht Teil des Portals ist']) {
    if (await page.getByText(stale, { exact: false }).count()) {
      fail(`Öffentliche Startseite ${viewport.name}: „${stale}“ ist zurück`);
    }
  }
  const metrics = await page.evaluate(() => ({
    overflow: document.documentElement.scrollWidth > window.innerWidth + 1,
    height: document.documentElement.scrollHeight,
  }));
  if (metrics.overflow) fail(`Öffentliche Startseite ${viewport.name}: horizontaler Überlauf`);
  const maxHeight = viewport.name === 'Mobil' ? 12_000 : 8_000;
  if (metrics.height > maxHeight) {
    fail(`Öffentliche Startseite ${viewport.name}: mit ${metrics.height}px unnötig lang (maximal ${maxHeight}px)`);
  }
  // The legal details moved to their own /impressum page (0.68.0); the landing
  // links there instead of holding an inline disclosure.
  const imprintResponse = await page.goto(new URL('/impressum', publicURL).href, { waitUntil: 'networkidle' });
  if (!imprintResponse || imprintResponse.status() !== 200) {
    fail(`Impressum ${viewport.name}: Status ${imprintResponse?.status() ?? 0}`);
  }
  for (const text of ['Ladungsfähige Anschrift', 'HAUSV Professional']) {
    if (!(await page.getByText(text).count())) {
      fail(`Impressum ${viewport.name}: „${text}“ fehlt`);
    }
  }
  if (!(await page.locator('a[href="/datenschutz"]').count())) {
    fail(`Impressum ${viewport.name}: Datenschutz ist nicht erreichbar`);
  }

  const startResponse = await page.goto(new URL('/start', publicURL).href, { waitUntil: 'networkidle' });
  if (!startResponse || startResponse.status() !== 200 ||
      !(await page.getByRole('heading', { name: 'Zuhause zuerst sicher anlegen.' }).count()) ||
      !(await page.getByRole('button', { name: 'Bestätigungslink anfordern' }).count())) {
    fail(`HAUSV Home Start ${viewport.name}: sicherer Reservierungsweg fehlt`);
  }
  const startMetrics = await page.evaluate(() => ({
    overflow: document.documentElement.scrollWidth > window.innerWidth + 1,
    fields: [...document.querySelectorAll('.home-start-form input, .home-start-form button')].map((element) => {
      const rect = element.getBoundingClientRect();
      return { width: rect.width, height: rect.height, right: rect.right };
    }),
    tokenFields: document.querySelectorAll('[name="token"], [name="base_url"], [name^="ha_"]').length,
  }));
  if (startMetrics.overflow || startMetrics.tokenFields ||
      startMetrics.fields.some((field) => field.right > viewport.size.width + 1 || field.width < 20 || field.height < 20)) {
    fail(`HAUSV Home Start ${viewport.name}: Formular läuft über oder fragt Geheimnisse ab (${JSON.stringify(startMetrics)})`);
  }
  await closeContext(context);
  process.stdout.write(`  ✓ Öffentliche Startseite + HAUSV Home Start · ${viewport.name} · ${metrics.height}px\n`);
}

async function createIssue(email, title, { verifyResidentAttachmentTarget = false } = {}) {
  const context = await newContext({ width: 1440, height: 900 });
  const page = await localLogin(context, email);
  await page.goto(`${baseURL}/app/anliegen`, { waitUntil: 'networkidle' });
  // Auf einer leeren Anliegen-Seite ist das Formular eine feste Karte, damit die
  // Hauptaktion nicht hinter einem Aufklapper liegt; sobald Anliegen bestehen,
  // ist es ein details-Element. Nur letzteres muss geöffnet werden — element.open
  // ist auf einer section undefined und liefe sonst in einen Klick ins Leere.
  const panel = page.locator('#issue-new');
  if (await panel.evaluate((element) => element.tagName === 'DETAILS' && !element.open)) {
    await panel.locator(':scope > summary').click();
  }
  const form = page.locator('form[data-issue-wizard]');
  await form.locator('textarea[name="body"]').fill(`${title}. Bitte im Haus prüfen.`);
  await form.locator('input[name="location_detail"]').fill('Keller, neben dem Fahrradraum');
  if (verifyResidentAttachmentTarget) {
    await form.locator('input[type="file"][name="attachments"]').setInputFiles({
      name: 'qa-anliegen.pdf',
      mimeType: 'application/pdf',
      buffer: Buffer.from('%PDF-1.4\n% headless resident attachment target\n'),
    });
  }
  await form.locator('[data-issue-step="describe"] [data-issue-next]').click();
  await form.locator('button[type="submit"]').click();
  await page.waitForURL(/\/app\/anliegen\/.+\?created=1$/);
  if (!(await page.getByText(title, { exact: true }).count())) fail(`Anliegen „${title}“ wurde nicht sichtbar gespeichert`);
  if (verifyResidentAttachmentTarget) {
    const deleteTargets = page.locator('.issue-resident-report .attachment-delete button');
    if (await deleteTargets.count() !== 1) {
      fail(`Bewohner-Anhang: genau ein Entfernen-Ziel erwartet, gefunden ${await deleteTargets.count()}`);
    }
    const target = await deleteTargets.first().boundingBox();
    if (!target || target.width < 44 || target.height < 44) {
      fail(`Bewohner-Anhang: Entfernen-Ziel ist ${target ? `${target.width}×${target.height}px` : 'nicht sichtbar'} statt mindestens 44×44px`);
    }
  }
  await closeContext(context);
}

async function assertResidentIssueProgressiveEnhancement() {
  const authContext = await newContext({ width: 390, height: 844 });
  await localLogin(authContext, 'resident@example.com');
  const storageState = await authContext.storageState();
  await closeContext(authContext);

  const noJSContext = await trackedContext({
    storageState,
    viewport: { width: 390, height: 844 },
    deviceScaleFactor: 1,
    locale: 'de-AT',
    timezoneId: 'Europe/Vienna',
    javaScriptEnabled: false,
  });
  const noJS = await noJSContext.newPage();
  await noJS.goto(`${baseURL}/app/anliegen?new=1#issue-new`, { waitUntil: 'domcontentloaded' });
  const fallback = await noJS.locator('form[data-issue-wizard]').evaluate((form) => ({
    noValidate: form.noValidate,
    visibleSteps: [...form.querySelectorAll('[data-issue-step]')].filter((step) => step.getClientRects().length).length,
    reviewVisible: Boolean(form.querySelector('.issue-review-enhanced')?.getClientRects().length),
    nextVisible: Boolean(form.querySelector('[data-issue-next]')?.getClientRects().length),
    submitVisible: Boolean(form.querySelector('.wizard-submit')?.getClientRects().length),
    exitVisible: Boolean(form.querySelector('.wizard-exit')?.getClientRects().length),
    titleRequired: form.elements.title.required,
  }));
  if (fallback.noValidate || fallback.visibleSteps !== 2 || fallback.reviewVisible || fallback.nextVisible ||
      !fallback.submitVisible || !fallback.exitVisible || fallback.titleRequired) {
    fail(`Anliegen ohne JavaScript ist nicht der vollständige Formular-Fallback: ${JSON.stringify(fallback)}`);
  }
  await noJS.locator('textarea[name="body"]').fill('JS-freier Meldeweg funktioniert ohne Skript. Bitte prüfen.');
  await Promise.all([
    noJS.waitForURL(/\/app\/anliegen\/.+\?created=1$/),
    noJS.locator('.wizard-submit').click(),
  ]);
  if (!(await noJS.getByRole('heading', { name: 'JS-freier Meldeweg funktioniert ohne Skript' }).count()) ||
      !(await noJS.getByText('Anliegen gemeldet', { exact: true }).count())) {
    fail('Anliegen ohne JavaScript wurde nicht mit Server-Titel direkt bestätigt');
  }
  await closeContext(noJSContext);

  const enhancedContext = await trackedContext({
    storageState,
    viewport: { width: 390, height: 844 },
    deviceScaleFactor: 1,
    locale: 'de-AT',
    timezoneId: 'Europe/Vienna',
    reducedMotion: 'reduce',
  });
  await enhancedContext.addInitScript(() => {
    window.__issueScrollCalls = [];
    const original = Element.prototype.scrollIntoView;
    Element.prototype.scrollIntoView = function (options) {
      window.__issueScrollCalls.push(options || null);
      return original.call(this, options);
    };
  });
  const enhanced = await enhancedContext.newPage();
  await enhanced.goto(`${baseURL}/app/anliegen?new=1#issue-new`, { waitUntil: 'networkidle' });
  const form = enhanced.locator('form[data-issue-wizard]');
  await form.locator('[data-issue-next]').click();
  const error = form.locator('#issue-body-error');
  if (!(await error.isVisible()) || await error.textContent() !== 'Bitte beschreiben Sie kurz, worum es geht.' ||
      await form.locator('textarea[name="body"]').getAttribute('aria-invalid') !== 'true') {
    fail('Anliegen zeigt die deutsche Inline-Validierung nicht eindeutig');
  }
  await form.locator('textarea[name="body"]').fill('z. B. flackert das Licht im Keller. Bitte prüfen.');
  await form.locator('[data-issue-next]').click();
  await enhanced.waitForURL((url) => url.searchParams.get('step') === 'review');
  const suggestedTitle = await form.locator('input[name="title"]').getAttribute('placeholder');
  if (suggestedTitle !== 'z. B. flackert das Licht im Keller') {
    fail(`Anliegen kürzt eine deutsche Abkürzung falsch: ${JSON.stringify(suggestedTitle)}`);
  }
  await form.locator('[data-issue-back]').click();
  await enhanced.waitForURL((url) => url.searchParams.get('step') === 'describe');
  await form.locator('textarea[name="body"]').fill('Browser-Verlauf bewahrt diesen Entwurf. Bitte prüfen.');
  await form.locator('input[name="location_detail"]').fill('Keller');
  await form.locator('[data-issue-next]').click();
  await enhanced.waitForURL((url) => url.searchParams.get('step') === 'review');
  if (!(await form.locator('[data-issue-step="review"]').isVisible())) fail('Anliegen-Prüfschritt fehlt');
  await enhanced.goBack();
  if (!(await form.locator('[data-issue-step="describe"]').isVisible()) ||
      await form.locator('textarea[name="body"]').inputValue() !== 'Browser-Verlauf bewahrt diesen Entwurf. Bitte prüfen.' ||
      await form.locator('input[name="location_detail"]').inputValue() !== 'Keller') {
    fail('Browser-Zurück verliert den Anliegen-Entwurf');
  }
  await enhanced.goForward();
  if (!(await form.locator('[data-issue-step="review"]').isVisible()) ||
      await form.locator('textarea[name="body"]').inputValue() !== 'Browser-Verlauf bewahrt diesen Entwurf. Bitte prüfen.') {
    fail('Browser-Vorwärts stellt den Anliegen-Prüfschritt nicht wieder her');
  }
  const motionCalls = await enhanced.evaluate(() => window.__issueScrollCalls || []);
  if (!motionCalls.length || motionCalls.some((call) => call && call.behavior === 'smooth')) {
    fail(`Anliegen ignoriert reduzierte Bewegung: ${JSON.stringify(motionCalls)}`);
  }
  await closeContext(enhancedContext);
  process.stdout.write('  ✓ Anliegen · zwei Schritte, Browser-Verlauf und vollständiger No-JS-Fallback\n');
}

function futureLocalInput(daysAhead, hour) {
  const date = new Date();
  date.setDate(date.getDate() + daysAhead);
  date.setHours(hour, 0, 0, 0);
  const offset = date.getTimezoneOffset() * 60_000;
  return new Date(date.getTime() - offset).toISOString().slice(0, 16);
}

function nextYearLocalInput(month, day, hour) {
  const date = new Date(new Date().getFullYear() + 1, month - 1, day, hour, 0, 0, 0);
  const offset = date.getTimezoneOffset() * 60_000;
  return new Date(date.getTime() - offset).toISOString().slice(0, 16);
}

async function createManagedEvent(page, title, startsAt) {
  await page.goto(`${baseURL}/app/events`, { waitUntil: 'networkidle' });
  await page.getByRole('button', { name: 'Termin erstellen' }).first().click();
  const form = page.locator('#event-create form');
  await form.locator('input[name="title"]').fill(title);
  await form.locator('input[name="starts_at"]').fill(startsAt);
  await form.locator('input[name="location"]').fill('Gemeinschaftsraum');
  await form.locator('details.dialog-optional').evaluate((node) => { node.open = true; });
  await form.locator('textarea[name="body"]').fill('Kurzer, klarer Hinweis für die Hausgemeinschaft.');
  await form.locator('button[type="submit"]').click();
  await page.waitForURL(/\/app\/events/);
}

async function seedManagedContent() {
  const context = await newContext({ width: 1440, height: 900 });
  const page = await localLogin(context, 'admin@example.com');

  await page.goto(`${baseURL}/app/announcements`, { waitUntil: 'networkidle' });
  // Die Hauptaktion steht in der Werkzeugleiste UND im Leerzustand, damit sie
  // auf einer leeren Seite erreichbar ist. Beide öffnen denselben Dialog.
  await page.getByRole('button', { name: 'Aushang erstellen' }).first().click();
  const announcement = page.locator('#announcement-create form');
  await announcement.locator('input[name="title"]').fill('QA Hausinformation zur Trinkwasserwartung');
  await announcement.locator('textarea[name="body"]').fill('Der gemeinsame Playwright-Lauf prüft diesen Aushang. Der Text ist bewusst länger als die Vorschau, damit Lesen, Einklappen und Ausklappen weiterhin über die native Aufklappfläche geprüft werden (HAUSV-662).');
  await announcement.locator('button[type="submit"]').click();
  await page.waitForURL(/\/app\/announcements/);

  await page.goto(`${baseURL}/app/events`, { waitUntil: 'networkidle' });
  await page.getByRole('button', { name: 'Termin erstellen' }).first().click();
  const event = page.locator('#event-create form');
  await event.locator('input[name="title"]').fill('QA Hausbegehung');
  await event.locator('input[name="starts_at"]').fill(futureLocalInput(14, 18));
  await event.locator('input[name="location"]').fill('Innenhof');
  await event.locator('details.dialog-optional').evaluate((node) => { node.open = true; });
  await event.locator('textarea[name="body"]').fill('Der Treffpunkt und die wichtigsten Hinweise stehen direkt beim Termin.');
  await event.locator('button[type="submit"]').click();
  await page.waitForURL(/\/app\/events/);

  // Zwei Folgemonate desselben künftigen Jahres machen die reduzierte
  // Jahresbeschriftung im echten Browser deterministisch prüfbar.
  await createManagedEvent(page, 'QA Jännertermin', nextYearLocalInput(1, 12, 18));
  await createManagedEvent(page, 'QA Februartermin', nextYearLocalInput(2, 9, 9));

  await page.goto(`${baseURL}/app/kontakte`, { waitUntil: 'networkidle' });
  const contactPanel = page.locator('#contact-add');
  if (!(await contactPanel.evaluate((element) => element.open))) {
    await contactPanel.locator(':scope > summary').click();
  }
  const contact = contactPanel.locator('form');
  await contact.locator('select[name="kind"]').selectOption({ label: 'Energie-Fachbetrieb' });
  await contact.locator('input[name="name"]').fill('QA Energiehilfe');
  await contact.locator('input[name="phone"]').fill('+43 316 000000');
  // Region, Qualifikation und Energie-Fähigkeiten liegen bewusst hinter einer
  // Zusatzangaben-Klappe, damit das Pflichtfeld-Formular kurz bleibt.
  const optional = contact.locator('details.contact-add-optional');
  if (await optional.count()) {
    await optional.evaluate((element) => {
      element.open = true;
    });
  }
  await contact.locator('input[name="service_region"]').fill('Wien und Umgebung');
  await contact.locator('input[name="qualification"]').fill('Elektrotechnik');
  await contact.locator('input[name="energy_capabilities"][value="metering"]').check();
  await contact.locator('input[name="energy_capabilities"][value="home-assistant"]').check();
  await contact.locator('button[type="submit"]').click();
  await page.waitForURL(/\/app\/kontakte/);

  await page.goto(`${baseURL}/app/dokumente`, { waitUntil: 'networkidle' });
  await page.getByRole('button', { name: 'Dokument hochladen' }).click();
  const documentForm = page.locator('#document-upload form');
  await documentForm.locator('input[name="title"]').fill('QA Hausordnung');
  await documentForm.locator('select[name="category"]').selectOption({ index: 1 });
  await documentForm.locator('select[name="visibility"]').selectOption({ label: 'Alle Bewohner' });
  await documentForm.locator('input[name="document"]').setInputFiles({
    name: 'qa-hausordnung.pdf',
    mimeType: 'application/pdf',
    buffer: Buffer.from('%PDF-1.4\n% isolated QA fixture\n'),
  });
  await documentForm.locator('button[type="submit"]').click();
  await page.waitForURL(/\/app\/dokumente/);

  await page.goto(`${baseURL}/app/settings/building?section=units`, { waitUntil: 'networkidle' });
  if (!(await page.getByText('Einheit 12', { exact: true }).count())) {
    await page.getByRole('link', { name: 'Einheit hinzufügen' }).click();
    const unitPanel = page.locator('#unit-add');
    await unitPanel.waitFor({ state: 'visible' });
    const unitForm = unitPanel.locator('form');
    await unitForm.locator('input[name="label"]').fill('Einheit 12');
    await unitForm.locator('input[name="owner_emails"]').fill('owner@example.com');
    await unitForm.locator('input[name="renter_emails"]').fill('resident@example.com');
    await unitPanel.getByRole('button', { name: 'Einheit anlegen' }).click();
    await page.waitForURL(/\/app\/settings\/building/);
  }
  await page.goto(`${baseURL}/app/settings/home?from=building`, { waitUntil: 'networkidle' });
  const officialUnit = page.locator('select[name="unit_id"]');
  if (await officialUnit.count()) {
    await officialUnit.selectOption('einheit-12');
    await page.getByRole('button', { name: 'Änderungen speichern' }).click();
    await page.waitForURL(/\/app\/settings\/building\?section=units&home=saved/);
  }
  await page.goto(`${baseURL}/app/settings/building?section=units`, { waitUntil: 'networkidle' });
  const linkedUnit = page.locator('.unit-card').filter({ hasText: 'Einheit 12' });
  if (!(await linkedUnit.getByText('Mein Zuhause · QA Zuhause', { exact: true }).count()) ||
      !(await linkedUnit.getByText('Einheit 12', { exact: true }).count())) {
    fail('Gebäude-Einstellungen: „Mein Zuhause“ und offizielle Einheit werden nicht klar getrennt');
  }
  await linkedUnit.getByRole('link', { name: 'Einheit 12 bearbeiten' }).click();
  if (!(await page.locator('#unit-einheit-12').getByLabel('Offizielle Bezeichnung').count())) {
    fail('Gebäude-Einstellungen: offizielle Bezeichnung fehlt im Einheiten-Dialog');
  }
  await page.keyboard.press('Escape');
  if (process.env.HV_QA_SCREENSHOT_DIR) {
    mkdirSync(process.env.HV_QA_SCREENSHOT_DIR, { recursive: true });
    await page.locator('.building .workspace').screenshot({
      path: join(process.env.HV_QA_SCREENSHOT_DIR, 'home-identity-building-desktop.png'),
    });
  }

  await page.goto(`${baseURL}/app/abstimmungen`, { waitUntil: 'networkidle' });
  await page.getByRole('button', { name: 'Abstimmung anlegen' }).first().click();
  const ballot = page.locator('#ballot-create form');
  await ballot.locator('input[name="title"]').fill('QA Bewohnerabstimmung Innenhof');
  await ballot.locator('textarea[name="description"]').fill('Soll der Innenhof mit heimischen Pflanzen begrünt werden?');
  await ballot.locator('textarea[name="options_text"]').fill('Ja, begrünen\nNein, unverändert lassen\nEnthaltung');
  await ballot.locator('input[name="closes_at"]').fill(futureLocalInput(60, 18));
  await ballot.locator('details.dialog-optional').filter({ hasText: 'Abstimmungsregeln' }).evaluate((node) => { node.open = true; });
  await ballot.locator('select[name="weighting"]').selectOption('per-head');
  await ballot.getByRole('button', { name: 'Entwurf anlegen' }).click();
  await page.waitForURL(/\/app\/abstimmungen/);
  const ballotCard = page.locator('.vote-card').filter({ hasText: 'QA Bewohnerabstimmung Innenhof' });
  await ballotCard.getByRole('button', { name: 'Abstimmung öffnen' }).click();
  await page.waitForURL(/vote=opened/);

  await closeContext(context);
}

async function toggleNativeDisclosure(details, label) {
  if (!(await details.count())) fail(`${label}: native Aufklappfläche fehlt`);
  const summary = details.locator(':scope > summary');
  const before = await details.evaluate((node) => node.open);
  await summary.focus();
  if (!(await summary.evaluate((node) => node === document.activeElement))) {
    fail(`${label}: Aufklappfläche erhält keinen Tastaturfokus`);
  }
  await summary.press('Enter');
  const after = await details.evaluate((node) => node.open);
  if (after === before) fail(`${label}: Enter ändert den nativen Zustand nicht`);
  await summary.press('Enter');
  if (await details.evaluate((node) => node.open) !== before) {
    fail(`${label}: Ausgangszustand lässt sich nicht wiederherstellen`);
  }
}

async function assertResidentContentResponsiveMatrix(sizes = [
    { width: 320, height: 568 },
    { width: 390, height: 844 },
    { width: 430, height: 932 },
    { width: 768, height: 1024 },
    { width: 1024, height: 768 },
    { width: 1440, height: 900 },
  ]) {
  const routeChecks = [
    { name: 'aushang', path: '/app/announcements', email: 'resident@example.com', details: '.announcement-body', guide: '.announcement-legend' },
    { name: 'termine', path: '/app/events', email: 'resident@example.com', details: '.event-details', guide: '.events-aside > details.guide' },
    { name: 'kontakte', path: '/app/kontakte', email: 'resident@example.com', details: '.contacts-aside > details.aside-panel', guide: '.contacts-aside > details.aside-panel' },
    { name: 'dokumente', path: '/app/dokumente', email: 'resident@example.com', details: 'details.document-more' },
    { name: 'abstimmungen', path: '/app/abstimmungen', email: 'owner@example.com', details: '.vote-details' },
    { name: 'verlauf', path: '/app/audit', email: 'resident@example.com', details: '.filter-panel', guide: '.help-disclosure' },
  ];
  const nextYear = new Date().getFullYear() + 1;

  for (const size of sizes) {
    const contexts = new Map();
    for (const route of routeChecks) {
      let state = contexts.get(route.email);
      if (!state) {
        const context = await trackedContext({
          viewport: size,
          deviceScaleFactor: 1,
          locale: 'de-AT',
          timezoneId: 'Europe/Vienna',
          reducedMotion: 'reduce',
        });
        const page = await localLogin(context, route.email);
        state = { context, page };
        contexts.set(route.email, state);
      }
      const { page } = state;
      const response = await page.goto(`${baseURL}${route.path}`, { waitUntil: 'networkidle' });
      if (!response || response.status() !== 200) {
        fail(`${route.name} ${size.width}px: Status ${response?.status() ?? 0}`);
      }
      const geometry = await page.evaluate((checkTargets) => {
        const main = document.querySelector('main');
        const visible = (node) => Boolean(node.getClientRects().length);
        const controls = [...(main?.querySelectorAll('button, a.button, summary, .contact-route') || [])].filter(visible);
        const offscreen = controls.filter((node) => {
          const box = node.getBoundingClientRect();
          const visibleWidth = Math.max(0, Math.min(box.right, window.innerWidth) - Math.max(box.left, 0));
          // Test reachability rather than font-dependent border alignment. Real page
          // overflow remains fail-closed, and every control keeps at least a 40px-wide
          // usable target when a fitted edge lands outside the integer viewport.
          return visibleWidth < Math.min(box.width, 39.5);
        }).map((node) => (node.getAttribute('aria-label') || node.textContent || node.tagName).trim().slice(0, 60));
        const short = checkTargets ? controls.filter((node) => {
          const box = node.getBoundingClientRect();
          return box.width > 0 && box.height > 0 && box.height < 39.5;
        }).map((node) => ({
          label: (node.getAttribute('aria-label') || node.textContent || node.tagName).trim().replace(/\s+/g, ' ').slice(0, 60),
          height: Math.round(node.getBoundingClientRect().height * 10) / 10,
        })) : [];
        return {
          overflow: document.documentElement.scrollWidth > window.innerWidth + 1,
          mainRight: main?.getBoundingClientRect().right || 0,
          offscreen,
          short,
        };
      }, size.width <= 430);
      if (geometry.overflow || geometry.mainRight > size.width + 1 || geometry.offscreen.length || geometry.short.length) {
        fail(`${route.name} ${size.width}px: Geometrie oder mobile Ziele sind instabil (${JSON.stringify(geometry)})`);
      }

      if (route.guide) {
        const guide = page.locator(route.guide).first();
        const guideOpen = await guide.evaluate((node) => node.open);
        if (guideOpen !== Boolean(route.guideOpen)) {
          fail(`${route.name} ${size.width}px: Ausgangszustand der Lesehilfe ist falsch`);
        }
        // In the 2-column aside band a CLOSED guide must not be stretched to its neighbour's
        // height. The old check compared against 110px — the legacy panel's own height — so a
        // templ panel that is legitimately 118px tall when closed (padding + a 46px summary
        // with eyebrow and heading) read as "stretched". Ask the real question instead: is
        // the box taller than its own content wants to be? A grid with align-items:start
        // never stretches, so rendered height must equal scrollHeight (±1).
        // getBoundingClientRect is the border box; scrollHeight excludes the borders. Compare
        // like with like — the first version of this check tolerated 1px and flagged a
        // 1px-bordered panel as "stretched by 2px". Measure the borders and subtract them.
        const stretch = await guide.evaluate((node) => {
          const cs = getComputedStyle(node);
          const borders = Number.parseFloat(cs.borderTopWidth) + Number.parseFloat(cs.borderBottomWidth);
          return { rendered: Math.round(node.getBoundingClientRect().height - borders), intrinsic: node.scrollHeight };
        });
        if (size.width >= 761 && size.width <= 1050 && stretch.rendered > stretch.intrinsic + 1) {
          fail(`${route.name} ${size.width}px: geschlossene Lesehilfe wird auf ${stretch.rendered}px gestreckt (eigener Inhalt: ${stretch.intrinsic}px)`);
        }
        await toggleNativeDisclosure(guide, `${route.name} ${size.width}px Lesehilfe`);
      }
      await toggleNativeDisclosure(page.locator(route.details).first(), `${route.name} ${size.width}px Inhalt`);

      if (route.name === 'aushang' && size.width === 768) {
        const search = await page.locator('.feed .filter-form').evaluate((form) => {
          const input = form.querySelector('input[type="search"]')?.getBoundingClientRect();
          const button = form.querySelector('button')?.getBoundingClientRect();
          return {
            sideBySide: Boolean(input && button && button.left >= input.right - 1),
            aligned: Boolean(input && button && Math.abs(input.bottom - button.bottom) <= 2),
            inputHeight: input?.height || 0,
            buttonHeight: button?.height || 0,
          };
        });
        if (!search.sideBySide || !search.aligned || search.inputHeight < 40 || Math.abs(search.inputHeight - search.buttonHeight) > 2) {
          fail(`Aushang 768px: Suche und Aktion bilden keine ruhige Zeile (${JSON.stringify(search)})`);
        }
      }
      if (route.name === 'termine') {
        const headings = await page.locator('.month-head h3').allTextContents();
        if (!headings.includes(`Jänner ${nextYear}`) || !headings.includes('Februar')) {
          fail(`Termine ${size.width}px: Jahr wird in Folgemonaten nicht reduziert (${JSON.stringify(headings)})`);
        }
      }
      if (route.name === 'aushang' && size.width === 390) {
        await page.evaluate(() => window.scrollTo(0, Math.min(500, document.documentElement.scrollHeight - window.innerHeight)));
        const sticky = await page.locator('[data-context-bar]').evaluate((node) => {
          const box = node.getBoundingClientRect();
          const declared = parseFloat(getComputedStyle(document.documentElement).getPropertyValue('--context-bar-h'));
          return { top: box.top, bottom: box.bottom, height: box.height, declared, activeView: Boolean(node.querySelector('.context-view')) };
        });
        // The unified phone header keeps its declared height while scrolling. HAUSV-765: it has
        // three rows (account, property, view) only while a role/support view is active (148px);
        // otherwise the view chooser shares the account row and the header is shorter.
        const maxHeight = sticky.activeView ? 148 : 112;
        if (Math.abs(sticky.top) > 1 || Math.abs(sticky.height - sticky.declared) > 1 || sticky.height > maxHeight + 1 || sticky.bottom > maxHeight + 1) {
          fail(`Mobile Navigation überdeckt beim Scrollen zu viel Inhalt (${JSON.stringify(sticky)})`);
        }
        await page.evaluate(() => window.scrollTo(0, 0));
        if (process.env.HV_QA_SCREENSHOT_DIR) {
          mkdirSync(process.env.HV_QA_SCREENSHOT_DIR, { recursive: true });
          await page.locator('#announcement-search').focus();
          await page.screenshot({
            path: join(process.env.HV_QA_SCREENSHOT_DIR, 'resident-content-keyboard-focus-aushang-390.png'),
            fullPage: false,
          });
        }
      }
      if ((size.width === 390 || size.width === 768 || size.width === 1440) && process.env.HV_QA_SCREENSHOT_DIR) {
        mkdirSync(process.env.HV_QA_SCREENSHOT_DIR, { recursive: true });
        await page.evaluate(() => {
          if (document.activeElement instanceof HTMLElement) document.activeElement.blur();
          window.scrollTo(0, 0);
        });
        await page.screenshot({
          path: join(process.env.HV_QA_SCREENSHOT_DIR, `resident-content-${route.name}-${size.width}.png`),
          fullPage: true,
        });
      }
    }
    for (const { context } of contexts.values()) await closeContext(context);
  }
  process.stdout.write(`  ✓ Bewohner-Inhalte · 6 Wege · ${sizes.map((size) => size.width).join('/')}px · Tastatur · reduzierte Bewegung\n`);
}

async function assertResidentContentClickFlows() {
  const context = await newContext({ width: 768, height: 1024 });
  const page = await localLogin(context, 'resident@example.com');

  await page.goto(`${baseURL}/app/announcements`, { waitUntil: 'networkidle' });
  await page.locator('#announcement-search').fill('QA');
  await Promise.all([
    page.waitForURL((url) => url.searchParams.get('sort') === 'oldest' && url.searchParams.get('q') === 'QA'),
    page.getByLabel('Sortierung', { exact: true }).selectOption('oldest'),
  ]);
  await Promise.all([
    page.waitForURL((url) => url.searchParams.get('category') === 'Info'),
    page.getByRole('navigation', { name: 'Aushang-Kategorien' }).getByRole('link', { name: 'Info', exact: true }).click(),
  ]);
  if (await page.locator('#announcement-search').inputValue() !== 'QA' ||
      await page.getByLabel('Sortierung', { exact: true }).inputValue() !== 'oldest') {
    fail('Aushang: Sortierung, Suche und Kategorie bleiben beim Filtern nicht erhalten');
  }
  await page.locator('#announcement-search').fill('nicht vorhandener QA Aushang');
  await Promise.all([
    page.waitForURL((url) => url.pathname.endsWith('/app/announcements') && url.searchParams.get('q') === 'nicht vorhandener QA Aushang'),
    page.getByRole('button', { name: 'Suchen' }).click(),
  ]);
  if (!(await page.locator('.empty-filter').isVisible())) fail('Aushang-Suche: verständlicher Kein-Treffer-Zustand fehlt');
  await page.getByRole('link', { name: 'Filter zurücksetzen' }).click();
  await page.waitForURL((url) => url.pathname.endsWith('/app/announcements') && !url.search);
  const announcement = page.locator('.announcement-card').filter({ hasText: 'QA Hausinformation' }).first();
  const announcementBody = announcement.locator('.announcement-body');
  if (!(await announcementBody.evaluate((node) => node.open))) await announcementBody.locator('summary').click();
  if (!(await announcementBody.locator('.announcement-body-content').isVisible())) fail('Aushang lesen: Inhalt bleibt verborgen');
  await page.getByRole('button', { name: 'Alle einklappen', exact: true }).click();
  if (await page.locator('.announcement-card > details[open]').count()) fail('Aushang: Alle einklappen lässt Beiträge offen');
  const collapsedHeight = await announcement.evaluate(node => node.getBoundingClientRect().height);
  if (await announcementBody.locator('.announcement-body-content').isVisible()) fail('Aushang: geschlossener Volltext ist sichtbar');
  await page.getByRole('button', { name: 'Alle ausklappen', exact: true }).click();
  if (await page.locator('.announcement-card > details:not([open])').count()) fail('Aushang: Alle ausklappen lässt Beiträge geschlossen');
  const expandedHeight = await announcement.evaluate(node => node.getBoundingClientRect().height);
  if (expandedHeight < collapsedHeight + 24) fail('Aushang: eingeklappte Karte spart nicht sichtbar Platz');
  if (await announcement.locator('.announcement-card-preview').isVisible()) fail('Aushang: Auszug wiederholt den offenen Volltext');
  const legend = page.locator('.announcement-legend');
  await legend.locator('summary').focus();
  await page.keyboard.press('Enter');
  if (!(await legend.locator('.announcement-legend-popover').isVisible())) fail('Aushang: Farblegende ist nicht per Tastatur erreichbar');
  await page.keyboard.press('Escape');
  if (await legend.evaluate(node => node.open)) fail('Aushang: Escape schließt die Farblegende nicht');


  await page.goto(`${baseURL}/app/events`, { waitUntil: 'networkidle' });
  const calendar = page.getByRole('link', { name: 'Kalender abonnieren' });
  const calendarHref = await calendar.getAttribute('href');
  if (!calendarHref) fail('Termine: persönlicher Kalenderlink fehlt');
  const calendarResponse = await context.request.get(new URL(calendarHref, baseURL).href);
  if (!calendarResponse.ok() || !calendarResponse.headers()['content-type']?.includes('text/calendar')) {
    fail(`Termine: Kalenderfeed antwortet nicht korrekt (${calendarResponse.status()})`);
  }

  await page.goto(`${baseURL}/app/kontakte`, { waitUntil: 'networkidle' });
  const contact = page.locator('.contact-row, .quick-card').filter({ hasText: 'QA Energiehilfe' }).first();
  const phoneHref = await contact.locator('a[href^="tel:"]').getAttribute('href');
  if (!phoneHref || decodeURIComponent(phoneHref).replace(/\D/g, '') !== '43316000000') {
    fail(`Kontakte: direkte Telefonroute fehlt (${phoneHref || 'kein Link'})`);
  }

  await page.goto(`${baseURL}/app/dokumente`, { waitUntil: 'networkidle' });
  await page.locator('#document-search').fill('QA Hausordnung');
  await Promise.all([
    page.waitForURL((url) => url.pathname.endsWith('/app/dokumente') && url.searchParams.get('q') === 'QA Hausordnung'),
    page.getByRole('button', { name: 'Anzeigen' }).click(),
  ]);
  const document = page.locator('.document-row').filter({ hasText: 'QA Hausordnung' }).first();
  if (!(await document.isVisible())) fail('Dokumente: Suche findet die freigegebene Unterlage nicht');
  const downloadPromise = page.waitForEvent('download');
  await document.getByRole('link', { name: 'Herunterladen' }).first().click();
  const download = await downloadPromise;
  if (download.suggestedFilename() !== 'qa-hausordnung.pdf') {
    fail(`Dokumente: unerwarteter Downloadname ${download.suggestedFilename()}`);
  }

  await page.goto(`${baseURL}/app/audit`, { waitUntil: 'networkidle' });
  const auditFilter = page.locator('.filter-panel');
  if (!(await auditFilter.evaluate((node) => node.open))) await auditFilter.locator(':scope > summary').click();
  await page.locator('#audit-search').fill('nicht vorhandener QA Vorgang');
  await Promise.all([
    page.waitForURL((url) => url.pathname.endsWith('/app/audit') && url.searchParams.has('q')),
    page.getByRole('button', { name: 'Ergebnisse zeigen' }).click(),
  ]);
  if (!(await page.getByRole('heading', { name: 'Kein Eintrag passt zu dieser Auswahl' }).isVisible())) {
    fail('Verlauf: Kein-Treffer-Zustand fehlt');
  }
  await page.getByRole('link', { name: 'Filter zurücksetzen' }).first().click();
  await page.waitForURL((url) => url.pathname.endsWith('/app/audit') && !url.search);
  const auditHelp = page.locator('.help-disclosure');
  const auditHelpSummary = await auditHelp.locator(':scope > summary').boundingBox();
  if (await auditHelp.evaluate((node) => node.open) || !auditHelpSummary || auditHelpSummary.height < 43.5) {
    fail(`Verlauf: Erklärungen sind nicht kompakt hinter einem 44px-Auslöser (${JSON.stringify(auditHelpSummary)})`);
  }
  await closeContext(context);

  const noJS = await trackedContext({
    viewport: { width: 390, height: 844 },
    deviceScaleFactor: 1,
    locale: 'de-AT',
    timezoneId: 'Europe/Vienna',
    javaScriptEnabled: false,
    reducedMotion: 'reduce',
  });
  const noJSPage = await localLogin(noJS, 'resident@example.com');
  for (const route of [
    { path: '/app/announcements', details: '.announcement-body' },
    { path: '/app/events', details: '.event-details' },
    { path: '/app/kontakte', details: '.contacts-aside > details.aside-panel' },
    { path: '/app/dokumente', details: 'details.document-more' },
    { path: '/app/audit', details: '.help-disclosure' },
  ]) {
    const response = await noJSPage.goto(`${baseURL}${route.path}`, { waitUntil: 'networkidle' });
    if (!response || response.status() !== 200) fail(`No-JS ${route.path}: Status ${response?.status() ?? 0}`);
    const details = noJSPage.locator(route.details).first();
    const before = await details.evaluate((node) => node.open);
    await details.locator(':scope > summary').click();
    if (await details.evaluate((node) => node.open) === before) fail(`No-JS ${route.path}: native Aufklappfläche reagiert nicht`);
    if (await noJSPage.evaluate(() => document.documentElement.scrollWidth > window.innerWidth + 1)) {
      fail(`No-JS ${route.path}: horizontaler Überlauf`);
    }
  }
  await closeContext(noJS);
  process.stdout.write('  ✓ Bewohner-Klickwege · Suche · Kalender · Kontakt · Download · Verlauf · No-JS\n');
}

async function assertResidentBallotFlow() {
  const title = 'QA Bewohnerabstimmung Innenhof';
  const ownerContext = await trackedContext({
    viewport: { width: 390, height: 844 },
    deviceScaleFactor: 1,
    locale: 'de-AT',
    timezoneId: 'Europe/Vienna',
    reducedMotion: 'reduce',
  });
  const owner = await localLogin(ownerContext, 'owner@example.com');
  await owner.goto(`${baseURL}/app/abstimmungen`, { waitUntil: 'networkidle' });
  let card = owner.locator('.vote-card').filter({ hasText: title });
  await owner.locator('[data-ballot-filter="closed"]').click();
  if (await card.isVisible()) fail('Abstimmungsfilter: laufende Abstimmung unter abgeschlossen sichtbar');
  await owner.locator('[data-ballot-filter="open"]').click();
  if (!(await card.isVisible())) fail('Abstimmungsfilter: laufende Abstimmung fehlt');
  const option = card.locator('input[name="option"]').first();
  if (!(await option.count())) fail('Abstimmung: stimmberechtigter Eigentümer erhält keine Auswahl');
  await option.check();
  await Promise.all([
    owner.waitForURL(/vote=cast/),
    card.getByRole('button', { name: 'Stimme speichern' }).click(),
  ]);
  card = owner.locator('.vote-card').filter({ hasText: title });
  if (!(await card.getByText('Stimme gespeichert', { exact: true }).count()) ||
      !(await card.getByRole('button', { name: 'Stimme ändern' }).count())) {
    fail('Abstimmung: gespeicherte Stimme oder Änderungsweg fehlt');
  }
  if (process.env.HV_QA_SCREENSHOT_DIR) {
    await owner.evaluate(() => window.scrollTo(0, 0));
    await owner.screenshot({
      path: join(process.env.HV_QA_SCREENSHOT_DIR, 'resident-content-abstimmung-stimme-390.png'),
      fullPage: true,
    });
  }

  const adminContext = await newContext({ width: 768, height: 1024 });
  const admin = await localLogin(adminContext, 'admin@example.com');
  await admin.goto(`${baseURL}/app/abstimmungen`, { waitUntil: 'networkidle' });
  const adminCard = admin.locator('.vote-card').filter({ hasText: title });
  await Promise.all([
    admin.waitForURL(/vote=closed/),
    adminCard.getByRole('button', { name: 'Abstimmung schließen' }).click(),
  ]);
  await closeContext(adminContext);

  await owner.goto(`${baseURL}/app/abstimmungen`, { waitUntil: 'networkidle' });
  card = owner.locator('.vote-card').filter({ hasText: title });
  if (!(await card.getByLabel('Abstimmungsergebnis').count())) fail('Abstimmung: Ergebnis fehlt nach dem Schließen');
  const protocolPromise = owner.waitForEvent('download');
  await card.getByRole('link', { name: 'Protokoll herunterladen' }).click();
  const protocol = await protocolPromise;
  if (!protocol.suggestedFilename().endsWith('-protokoll.html')) {
    fail(`Abstimmung: unerwarteter Protokollname ${protocol.suggestedFilename()}`);
  }
  if (process.env.HV_QA_SCREENSHOT_DIR) {
    await owner.evaluate(() => window.scrollTo(0, 0));
    await owner.screenshot({
      path: join(process.env.HV_QA_SCREENSHOT_DIR, 'resident-content-abstimmung-ergebnis-390.png'),
      fullPage: true,
    });
  }
  await closeContext(ownerContext);
  process.stdout.write('  ✓ Abstimmung · Eigentümerstimme · Änderung · Ergebnis · Protokoll\n');
}

async function assertHomeOnboarding() {
  let context = await newContext({ width: 1440, height: 900 });
  let page = await localLogin(context, 'owner@example.com');
  await page.goto(`${baseURL}/app/zuhause/onboarding`, { waitUntil: 'networkidle' });
  if (!(await page.getByRole('heading', { name: 'Womit möchten Sie beginnen? Mit Ihrem Zuhause.' }).count())) {
    fail('Onboarding: verständlicher Einstieg fehlt');
  }
  const strip = page.locator('.energy-mode-strip');
  if ((await strip.locator('strong').first().innerText()).trim() !== 'Nur beobachten') {
    fail('Onboarding: sichtbarer Beobachtungsmodus fehlt');
  }
  const firstNext = page.getByRole('button', { name: 'Verstanden, weiter' });
  await firstNext.focus();
  if (!(await firstNext.evaluate((node) => node === document.activeElement))) {
    fail('Onboarding: Tastaturfokus ist am ersten Weiter-Schritt nicht sichtbar erreichbar');
  }
  await firstNext.press('Enter');
  await page.waitForURL(/step=2/);
  await closeContext(context);

  // A fresh browser context proves that the saved step survives an interruption.
  context = await newContext({ width: 1440, height: 900 });
  page = await localLogin(context, 'owner@example.com');
  await page.goto(`${baseURL}/app/zuhause/onboarding`, { waitUntil: 'networkidle' });
  if (!(await page.getByRole('heading', { name: 'Was richten wir gemeinsam ein?' }).count())) {
    fail('Onboarding: unterbrochener Schritt wurde nicht fortgesetzt');
  }
  await page.locator('input[name="household_name"]').fill('');
  await page.getByRole('button', { name: 'Weiter zu den Verbrauchern' }).press('Enter');
  const householdName = page.locator('input[name="household_name"]');
  const householdValidation = await householdName.evaluate((node) => ({
    valid: node.checkValidity(),
    message: node.validationMessage,
  }));
  if (
    await page.getByRole('heading', { name: 'Was richten wir gemeinsam ein?' }).count() !== 1
    || householdValidation.valid
    || !householdValidation.message.trim()
  ) {
    fail('Onboarding: verständliche Pflichtfeldprüfung greift nicht');
  }
  const homeType = page.locator('select[name="home_type"]');
  const homeTypeLabel = page.locator('[data-home-type-label]');
  const homeTypeCopy = page.locator('[data-home-type-copy]');
  const homeTypeCases = [
    ['apartment', 'Wohnung', 'Ein einzelner Haushalt in einem Mehrparteienhaus.'],
    ['house', 'Einfamilienhaus', 'Ein Haushalt mit eigenem Gebäude.'],
    ['community', 'Hausgemeinschaft', 'Mehrere Parteien und gemeinsam genutzte Anlagen.'],
  ];
  for (const [value, label, copy] of homeTypeCases) {
    await homeType.selectOption(value);
    if ((await homeTypeLabel.innerText()).trim() !== label ||
        !(await homeTypeCopy.innerText()).includes(copy)) {
      fail(`Onboarding: Wirkung der Zuhause-Art „${label}“ wird nicht direkt erklärt`);
    }
  }
  if (!(await page.getByText('Die Auswahl kann Geltungsbereich und Sichtbarkeit ändern.', { exact: false }).count()) ||
      !(await page.getByText('„Nur beobachten“ bleibt unverändert.', { exact: false }).count())) {
    fail('Onboarding: Auswahlwirkung grenzt Sichtbarkeit und Sicherheitsmodus nicht ehrlich ab');
  }
  await homeType.selectOption('apartment');
  await page.locator('input[name="household_name"]').fill('QA Zuhause');

  const mobileContext = await newContext({ width: 390, height: 844 });
  const mobilePage = await localLogin(mobileContext, 'owner@example.com');
  await mobilePage.goto(`${baseURL}/app/zuhause/onboarding`, { waitUntil: 'networkidle' });
  if (!(await mobilePage.locator('[data-home-type-explanation]').count()) ||
      await mobilePage.evaluate(() => document.documentElement.scrollWidth > window.innerWidth + 1)) {
    fail('Onboarding Mobil: Erklärung der Zuhause-Art fehlt oder läuft horizontal über');
  }
  if (process.env.HV_QA_SCREENSHOT_DIR) {
    mkdirSync(process.env.HV_QA_SCREENSHOT_DIR, { recursive: true });
    await mobilePage.screenshot({
      path: join(process.env.HV_QA_SCREENSHOT_DIR, 'energy-home-type-mobile.png'),
      fullPage: true,
    });
    await page.screenshot({
      path: join(process.env.HV_QA_SCREENSHOT_DIR, 'energy-home-type-desktop.png'),
      fullPage: true,
    });
  }
  await closeContext(mobileContext);

  await page.getByRole('button', { name: 'Weiter zu den Verbrauchern' }).press('Enter');
  await page.waitForURL(/step=3/);
  await page.locator('input[name="assets"][value="pv"]').check();
  await page.locator('input[name="assets"][value="ev"]').check();
  await page.locator('input[name="assets"][value="wallbox"]').check();
  await page.getByRole('button', { name: 'Weiter zu den Messwerten' }).press('Enter');
  await page.waitForURL(/step=4/);
  if (!(await page.getByRole('button', { name: 'Ohne Verbindung starten' }).count())) {
    fail('Onboarding: klarer Offline-Weg fehlt');
  }
  if ((await page.locator('.onboarding-recommended .onboarding-candidate').count()) !== 5) {
    fail('Onboarding: es werden nicht genau fünf priorisierte Messwerte gezeigt');
  }
  if ((await page.getByText('Nur lesen', { exact: true }).count()) < 5) {
    fail('Onboarding: Read-only-Charakter ist nicht pro Empfehlung sichtbar');
  }
  for (const noisyEntity of ['iphone_battery', 'robot_battery', 'pv_forecast_power', 'kettle_power', 'battery_force_charge', 'lock_battery']) {
    if (await page.getByText(noisyEntity, { exact: false }).count()) {
      fail(`Onboarding: irrelevanter Home-Assistant-Treffer sichtbar: ${noisyEntity}`);
    }
  }
  const disclosures = page.locator('details.onboarding-disclosure');
  if (await disclosures.evaluateAll((elements) => elements.some((element) => element.open))) {
    fail('Onboarding: optionale technische Treffer oder Zuordnung sind ungefragt offen');
  }
  if (process.env.HV_QA_SCREENSHOT_DIR) {
    mkdirSync(process.env.HV_QA_SCREENSHOT_DIR, { recursive: true });
    await page.screenshot({
      path: join(process.env.HV_QA_SCREENSHOT_DIR, 'energy-onboarding-desktop.png'),
      fullPage: true,
    });
  }
  const technicalHits = page.getByText('Weitere technische Treffer anzeigen', { exact: false });
  if (!(await technicalHits.count())) {
    fail('Onboarding: zusätzliche plausible Treffer sind nicht kontrolliert einklappbar');
  }
  await technicalHits.click();
  if (!(await page.getByText('sensor.home_consumption', { exact: true }).count())) {
    fail('Onboarding: zusätzliche plausible Messwerte fehlen in den Technikdetails');
  }
  const additionalReadings = page.locator('.onboarding-additional input[name="entities"]');
  if ((await additionalReadings.count()) < 3) {
    fail('Onboarding: Fixture liefert nicht mindestens acht prüfbare Messwerte');
  }
  for (const checkbox of await additionalReadings.all()) {
    await checkbox.check();
  }
  await technicalHits.click();
  await page.getByRole('button', { name: '5 Messwerte übernehmen' }).press('Enter');
  await page.waitForURL(/step=5/);
  if ((await page.locator('.onboarding-trust').count()) !== 2 ||
      !(await page.getByText('Als Nächstes:', { exact: false }).count())) {
    fail('Onboarding: Abschluss zeigt nicht genau einen nächsten Schritt plus Kostenhinweis');
  }
  await page.getByRole('button', { name: 'Mein Zuhause öffnen' }).press('Enter');
  await page.waitForURL(/\/app\/energie/);
  await closeContext(context);
  if (fastQA) {
    process.stdout.write('  ✓ Energie-Onboarding · schneller Einrichtungsweg\n');
    return;
  }

  context = await newContext({ width: 390, height: 844 });
  page = await localLogin(context, 'owner@example.com');
  await page.goto(`${baseURL}/app/zuhause/onboarding?step=4`, { waitUntil: 'networkidle' });
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth + 1);
  if (overflow) fail('Onboarding Mobil: horizontaler Überlauf');
  const smallTargets = await page.locator('.onboarding-page .button, .energy-mode-action').evaluateAll((nodes) =>
    nodes.filter((node) => {
      const rect = node.getBoundingClientRect();
      return rect.width > 0 && rect.height > 0 && rect.height < 44;
    }).map((node) => (node.textContent || '').trim())
  );
  if (smallTargets.length) fail(`Onboarding Mobil: Touch-Ziele unter 44px: ${smallTargets.join(', ')}`);
  const box = await page.locator('.energy-mode-strip').boundingBox();
  const mobileNav = await contextBarBox(page);
  const stripInOnboarding = await page.locator('.onboarding-main .energy-mode-strip').count();
  if (!box || !mobileNav || stripInOnboarding !== 1 ||
      Math.abs(box.y - (mobileNav.y + mobileNav.height)) > 1) {
    fail(`Onboarding Mobil: Beobachtungsmodus nicht sauber unter der Navigation (${JSON.stringify({ box, mobileNav })})`);
  }
  await closeContext(context);
  process.stdout.write('  ✓ Energie-Onboarding · Tastatur · Fortsetzen · Mobil\n');
}

async function ensureFocusedEnergyUnit() {
  const context = await newContext({ width: 1440, height: 900 });
  const page = await localLogin(context, 'admin@example.com');
  await page.goto(`${baseURL}/app/settings/building?section=units`, { waitUntil: 'networkidle' });
  if (!(await page.getByText('Einheit 12', { exact: true }).count())) {
    await page.getByRole('link', { name: 'Einheit hinzufügen' }).click();
    const unitPanel = page.locator('#unit-add');
    await unitPanel.waitFor({ state: 'visible' });
    const unitForm = unitPanel.locator('form');
    await unitForm.locator('input[name="label"]').fill('Einheit 12');
    await unitForm.locator('input[name="owner_emails"]').fill('owner@example.com');
    await unitForm.locator('input[name="renter_emails"]').fill('resident@example.com');
    await unitPanel.getByRole('button', { name: 'Einheit anlegen' }).click();
    await page.waitForURL(/\/app\/settings\/building/);
  }
  await page.goto(`${baseURL}/app/settings/home?from=building`, { waitUntil: 'networkidle' });
  const officialUnit = page.locator('select[name="unit_id"]');
  if (await officialUnit.count()) {
    await officialUnit.selectOption('einheit-12');
    await page.getByRole('button', { name: 'Änderungen speichern' }).click();
    await page.waitForURL(/\/app\/settings\/building\?section=units&home=saved/);
  }
  await page.goto(`${baseURL}/app/kontakte`, { waitUntil: 'networkidle' });
  if (!(await page.getByText('QA Energiehilfe', { exact: true }).count())) {
    const contactPanel = page.locator('#contact-add');
    if (!(await contactPanel.evaluate((element) => element.open))) {
      await contactPanel.locator(':scope > summary').click();
    }
    const contact = contactPanel.locator('form');
    await contact.locator('select[name="kind"]').selectOption({ label: 'Energie-Fachbetrieb' });
    await contact.locator('input[name="name"]').fill('QA Energiehilfe');
    await contact.locator('input[name="phone"]').fill('+43 316 000000');
    const optional = contact.locator('details.contact-add-optional');
    if (await optional.count()) await optional.evaluate((element) => { element.open = true; });
    await contact.locator('input[name="service_region"]').fill('Wien und Umgebung');
    await contact.locator('input[name="qualification"]').fill('Elektrotechnik');
    await contact.locator('input[name="energy_capabilities"][value="metering"]').check();
    await contact.locator('input[name="energy_capabilities"][value="home-assistant"]').check();
    await contact.locator('button[type="submit"]').click();
    await page.waitForURL(/\/app\/kontakte/);
  }
  await closeContext(context);
}

// Viertelstunden im laufenden Kalendermonat. Ein fest verdrahtetes Datum
// funktioniert genau so lange, bis der Monat wechselt: die Tarifkarte liest über
// PeakForMonth und findet dann nichts mehr, worauf der Prüflauf mit einem
// scheinbar unzusammenhängenden Timeout stirbt.
function currentMonthQuarterHourCSV() {
  const now = new Date();
  const day = String(Math.min(now.getDate(), 28)).padStart(2, '0');
  const month = String(now.getMonth() + 1).padStart(2, '0');
  const date = `${now.getFullYear()}-${month}-${day}`;
  return Buffer.from(
    `timestamp;import_kwh\n${date} 00:00;0,42\n${date} 00:15;0,38\n`,
  );
}

async function importPilotReference(page) {
  const measurementPanel = page.locator('details.energy-collapsible').filter({ hasText: 'Messwerte & Referenz' });
  await measurementPanel.locator(':scope > summary').click();
  await page.locator('input[name="smart_meter_file"]').setInputFiles({
    name: 'smart-meter-pilot.csv',
    mimeType: 'text/csv',
    buffer: currentMonthQuarterHourCSV(),
  });
  await page.getByRole('button', { name: 'Als Referenz importieren' }).click();
  await page.waitForLoadState('networkidle');
  if (!(await page.getByText('Smart-Meter-Datei übernommen.', { exact: false }).count())) {
    fail('Pilot: Smart-Meter-Referenz wurde nicht übernommen');
  }
}

async function assertPilotHome({
  slug,
  email,
  householdName,
  expectedAssets,
  absentAssets = [],
  expectedMeasured,
  expectedCaptured,
  inviteHelper = false,
}) {
  const origin = `${baseURL}/${slug}`;
  const context = await newContext({ width: 1440, height: 900 });
  const page = await localLogin(context, email, origin);
  await page.goto(`${origin}/app/zuhause/onboarding`, { waitUntil: 'networkidle' });
  await page.getByRole('button', { name: 'Verstanden, weiter' }).click();
  await page.waitForURL(/step=2/);
  await page.locator('input[name="household_name"]').fill(householdName);
  await page.locator('select[name="home_type"]').selectOption('house');
  await page.getByRole('button', { name: 'Weiter zu den Verbrauchern' }).click();
  await page.waitForURL(/step=3/);
  for (const kind of expectedAssets) {
    if (!(await page.locator(`input[name="assets"][value="${kind}"]`).isChecked())) {
      fail(`${householdName}: vorhandene Anlage ${kind} ist im sanften Onboarding nicht vorausgewählt`);
    }
  }
  for (const kind of absentAssets) {
    if (await page.locator(`input[name="assets"][value="${kind}"]`).isChecked()) {
      fail(`${householdName}: nicht vorhandene Anlage ${kind} wurde vorausgewählt`);
    }
  }
  await page.getByRole('button', { name: 'Weiter zu den Messwerten' }).click();
  await page.waitForURL(/step=4/);
  const candidateCount = await page.locator('.onboarding-recommended .onboarding-candidate').count();
  if (candidateCount < 1 || candidateCount > 5) {
    fail(`${householdName}: ${candidateCount} statt höchstens fünf ruhiger Messvorschläge`);
  }
  if (await page.locator('.onboarding-recommended code').count()) {
    fail(`${householdName}: technische Sensor-IDs stehen in der normalen Empfehlung`);
  }
  if ((await page.getByText('Nur lesen', { exact: true }).count()) < candidateCount) {
    fail(`${householdName}: Read-only-Zusage fehlt an Messvorschlägen`);
  }
  await page.getByRole('button', { name: /Messwerte übernehmen/ }).click();
  await page.waitForURL(/step=5/);
  if (!(await page.getByText('Zwölf Monate voller Produktumfang kostenlos', { exact: true }).count()) ||
      !(await page.getByText('1 € pro Monat', { exact: false }).count())) {
    fail(`${householdName}: transparentes Zwölf-Monats-/12-Euro-Modell fehlt`);
  }
  await page.getByRole('button', { name: 'Mein Zuhause öffnen' }).click();
  await page.waitForURL(/\/app\/energie/);
  if ((await page.locator('.energy-mode-strip strong').first().innerText()).trim() !== 'Nur beobachten') {
    fail(`${householdName}: startet nicht sicher im Beobachtungsmodus`);
  }
  const coverage = page.locator('.energy-coverage');
  for (const label of expectedMeasured) {
    const row = coverage.locator('.energy-coverage-row').filter({ hasText: label });
    if (!(await row.locator('.energy-coverage-state.good').getByText('Gemessen', { exact: false }).count())) {
      fail(`${householdName}: ${label} wird trotz Messwert nicht als gemessen erklärt`);
    }
  }
  for (const label of expectedCaptured) {
    const row = coverage.locator('.energy-coverage-row').filter({ hasText: label });
    if (!(await row.locator('.energy-coverage-state.open').getByText('Nur erfasst', { exact: false }).count())) {
      fail(`${householdName}: Messlücke bei ${label} ist nicht verständlich sichtbar`);
    }
  }
  for (const label of absentAssets.map((kind) => ({
    battery: 'Batteriespeicher',
    pv: 'PV-Anlage',
    ev: 'E-Auto',
  })[kind]).filter(Boolean)) {
    if (await coverage.locator('.energy-coverage-row').filter({ hasText: label }).count()) {
      fail(`${householdName}: nicht vorhandene Anlage ${label} steht in der Messabdeckung`);
    }
  }

  await importPilotReference(page);
  if (process.env.HV_QA_SCREENSHOT_DIR) {
    mkdirSync(process.env.HV_QA_SCREENSHOT_DIR, { recursive: true });
    await page.screenshot({
      path: join(process.env.HV_QA_SCREENSHOT_DIR, `energy-pilot-${slug}.png`),
      fullPage: true,
    });
  }

  if (inviteHelper) {
    const caretakerPanel = page.locator('details.energy-collapsible').filter({ hasText: 'Technische Betreuung' });
    await caretakerPanel.locator(':scope > summary').click();
    const invitation = caretakerPanel.locator('details.energy-compact-create').filter({ hasText: 'Technische Vertrauensperson einladen' });
    await invitation.locator('summary').click();
    await invitation.locator('input[name="first_name"]').fill('Sanfte');
    await invitation.locator('input[name="last_name"]').fill('Hilfe');
    await invitation.locator('input[name="email"]').fill('house_b-helper@example.com');
    await invitation.locator('input[value="configure"]').check();
    await invitation.getByRole('button', { name: 'Zur Liegenschaft einladen' }).click();
    await page.waitForLoadState('networkidle');
    const helperContext = await newContext({ width: 390, height: 844 });
    const helper = await localLogin(helperContext, 'house_b-helper@example.com', origin);
    const response = await helper.goto(`${origin}/app/energie`, { waitUntil: 'networkidle' });
    if (!response || response.status() !== 200 ||
        !(await helper.getByText('Nur beobachten', { exact: true }).count())) {
      fail(`${householdName}: technische Hilfe erreicht das Haus nicht`);
    }
    if (await helper.locator('summary.energy-mode-action').count()) {
      fail(`${householdName}: technische Hilfe sieht den Eigentümer-Schalter`);
    }
    await closeContext(helperContext);
  }

  await closeContext(context);
  process.stdout.write(`  ✓ Pilot ${householdName} · getrennt · read-only · Messlücken\n`);
}


async function assertSidebarAccountFits() {
  for (const size of [{ name: '1440x900', width: 1440, height: 900 }, { name: '1280x800', width: 1280, height: 800 }]) {
    const context = await newContext({ width: size.width, height: size.height });
    const page = await localLogin(context, 'admin@example.com');
    const response = await page.goto(`${baseURL}/app`, { waitUntil: 'networkidle' });
    if (!response || response.status() !== 200) {
      fail(`Seitenleiste ${size.name}: /app nicht erreichbar`);
    }
    const account = await page.locator('aside.sidebar footer.sidebar-release').boundingBox();
    if (!account || account.bottom > size.height + 1) {
      fail(`Seitenleiste ${size.name}: footer.sidebar-release ragt aus dem Viewport (${JSON.stringify(account)})`);
    }
    await closeContext(context);
    process.stdout.write(`  ✓ Seitenleiste · ${size.name} · Konto bleibt im Viewport\n`);
  }
}

async function assertPage(page, persona, route, viewportName) {
  const response = await page.goto(`${baseURL}${route.path}`, { waitUntil: 'networkidle' });
  if (!response || response.status() !== 200) {
    fail(`${persona.name} ${viewportName} ${route.path}: Status ${response?.status() ?? 0}`);
  }
  const heading = page.getByRole('heading', { name: route.heading }).first();
  if (!(await heading.count())) {
    // HAUSV-620: the overview greets residents by name, but for people who
    // administer several houses it names the current house instead. Both are
    // valid; the house name must then match the sidebar's header card.
    const houseHeading = route.path === '/app'
      ? await page.evaluate(() => {
          const h1 = document.querySelector('main h1');
          const card = document.querySelector('[data-context-bar] .house-header-copy strong');
          return h1 && card && h1.textContent.trim() === card.textContent.trim() ? h1.textContent.trim() : '';
        })
      : '';
    if (!houseHeading) fail(`${persona.name} ${viewportName} ${route.path}: Überschrift fehlt`);
  }
  if (!(await page.getByText(route.content, { exact: false }).count())) {
    fail(`${persona.name} ${viewportName} ${route.path}: Inhalt „${route.content}“ fehlt`);
  }
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth + 1);
  if (overflow) fail(`${persona.name} ${viewportName} ${route.path}: horizontaler Überlauf`);

  if (route.path === '/app') {
    // Exactly one main state: the overview is EITHER the calm resident view OR the dense
    // manager view, never both and never neither. Legacy named them .home-calm /
    // .home-primary-task; templ renders main.calm-main or main.dense-main.
    const focusCount = await page.locator('main.calm-main, main.dense-main').count();
    if (focusCount !== 1) fail(`${persona.name} ${viewportName}: Hausüberblick hat ${focusCount} Hauptzustände`);
    // The sidebar map is again a raster tile (HAUSV-560): GET /map-tiles/…, pin overlay,
    // height tiers. The SVG street sketch is only the unconfigured fallback, not a design
    // decision. Asserted here: the OSM link, the tile URL in the painted card, and that
    // footer.sidebar-release stays inside the viewport on the hire-walk notebooks.
    if (viewportName === 'Desktop') {
      const map = page.locator('aside.sidebar a.map');
      if ((await map.count()) !== 1) fail(`${persona.name} Desktop: Kartenlink in der Seitenleiste fehlt`);
      const href = await map.getAttribute('href');
      const label = await map.getAttribute('aria-label');
      if (!href || !/openstreetmap\.org/.test(href) || !label || !/OpenStreetMap/.test(label)) {
        fail(`${persona.name} Desktop: Kartenlink zeigt nicht auf OpenStreetMap oder hat kein Label (${href} / ${label})`);
      }
      // HAUSV-620/621 made the map a thumbnail inside the house header card; the
      // contract is that it stays a visible, clickable tile, not that it is wide.
      const box = await map.boundingBox();
      if (!box || box.width < 40 || box.height < 40) {
        fail(`${persona.name} Desktop: Kartenlink ist nicht sichtbar gerendert (${JSON.stringify(box)})`);
      }
      const tileCount = await map.locator('img.side-map-tile').count();
      const tileSrc = tileCount ? await map.locator('img.side-map-tile').first().getAttribute('src') : '';
      if (!tileCount || !/map-tiles\//.test(tileSrc || '')) {
        fail(`${persona.name} Desktop: Kartenkachel /map-tiles/ fehlt (${tileCount} / ${tileSrc})`);
      }
      const account = await page.locator('aside.sidebar footer.sidebar-release').boundingBox();
      if (!account || account.bottom > 900 + 1) {
        fail(`${persona.name} Desktop: footer.sidebar-release ragt aus dem Viewport (${JSON.stringify(account)})`);
      }
    }
  }
}

async function assertRoleActions(page, persona) {
  if (persona.manages) {
    const expected = [
      ['/app/announcements', 'Aushang erstellen'],
      ['/app/events', 'Termin erstellen'],
      ['/app/kontakte', 'Kontakt hinzufügen'],
      ['/app/dokumente', 'Dokument hochladen', 'button'],
    ];
    for (const [path, label, role] of expected) {
      await page.goto(`${baseURL}${path}`, { waitUntil: 'networkidle' });
      const action = role ? page.getByRole(role, { name: label }) : page.getByText(label, { exact: true });
      if (!(await action.count())) fail(`${persona.name}: Aktion „${label}“ fehlt`);
    }
    const board = await page.goto(`${baseURL}/app/anliegen/board`, { waitUntil: 'networkidle' });
    // HAUSV-717: „Bearbeiten“ lives in the detail panel that opens from a card.
    if (board && board.status() === 200 && (await page.locator('[data-board-card]').count())) {
      await page.locator('[data-board-card]').first().click();
      await page.locator('[data-board-panel] a, [data-board-panel] button').filter({ hasText: /^Bearbeiten$/ }).first().waitFor({ state: 'visible', timeout: 10000 }).catch(() => {});
    }
    if (!board || board.status() !== 200 || !(await page.getByText('Bearbeiten', { exact: true }).count())) {
      fail(`${persona.name}: Anliegen-Bearbeitung fehlt`);
    }
    return;
  }

  for (const path of ['/app/anliegen/board', '/app/settings/users', '/app/settings/building']) {
    const response = await page.goto(`${baseURL}${path}`, { waitUntil: 'networkidle' });
    if (!response || response.status() !== 403) fail(`${persona.name}: ${path} ist nicht mit 403 geschützt`);
  }
  await page.goto(`${baseURL}/app/anliegen`, { waitUntil: 'networkidle' });
  if (!(await page.getByText(/Anliegen melden|Neues Anliegen|Erstes Anliegen melden/).count())) {
    fail(`${persona.name}: Bewohner-Aktion für Anliegen fehlt`);
  }
}

async function assertLogoutBackNavigation() {
  const context = await newContext({ width: 1440, height: 900 });
  const page = await localLogin(context, 'resident@example.com');
  await page.goto(`${baseURL}/app`, { waitUntil: 'networkidle' });
  const protectedHeading = page.getByRole('heading', { name: /Hallo Rita/ }).first();
  if (!(await protectedHeading.isVisible())) {
    fail('Abmelden/Zurück: geschützte Ausgangsseite fehlt');
  }

  await page.locator('[data-context-account] > summary').click();
  await Promise.all([
    page.waitForURL((url) => url.pathname.endsWith('/')),
    page.getByRole('button', { name: 'Abmelden' }).click(),
  ]);
  const loggedOutPath = new URL(page.url()).pathname;
  loginStorageStates.delete(`${baseURL}|resident@example.com`);
  if (!(await page.getByRole('heading', { name: 'Anmelden' }).isVisible())) {
    fail('Abmelden/Zurück: Loginseite nach Abmeldung fehlt');
  }

  await page.goBack({ waitUntil: 'domcontentloaded' });
  if (new URL(page.url()).pathname !== loggedOutPath) {
    await page.reload({ waitUntil: 'domcontentloaded' });
  }
  await page.waitForURL((url) => url.pathname.endsWith('/'), { timeout: 10_000 });
  await page.waitForLoadState('networkidle');
  const authenticatedBody = await page.locator('body[data-authenticated-app]').count();
  if (authenticatedBody || await protectedHeading.isVisible().catch(() => false)) {
    fail('Abmelden/Zurück: geschützter Inhalt wurde aus dem Browsercache wieder sichtbar');
  }
  if (!(await page.getByRole('heading', { name: 'Anmelden' }).isVisible())) {
    fail('Abmelden/Zurück: unauthentifizierter Zustand fehlt nach Browser-Zurück');
  }

  await closeContext(context);
  process.stdout.write('  ✓ Abmelden → Browser-Zurück bleibt unauthentifiziert\n');
}

async function assertEnergySafetyAndFlow(viewport) {
  const residentContext = await newContext(viewport.size);
  const resident = await localLogin(residentContext, 'resident@example.com');
  await resident.goto(`${baseURL}/app/energie`, { waitUntil: 'networkidle' });
  if (await resident.locator('summary.energy-mode-action').count()) {
    fail(`Bewohner ${viewport.name}: Steuerungsfreigabe sichtbar`);
  }
  await closeContext(residentContext);

  const ownerContext = await newContext(viewport.size);
  const page = await localLogin(ownerContext, 'owner@example.com');
  await page.goto(`${baseURL}/app/energie`, { waitUntil: 'networkidle' });
  await assertHomeIdentityPair(page, 'energy-heading', 'QA Zuhause', 'Einheit 12', `Energie ${viewport.name}`);
  if (viewport.name === 'Mobil') {
    await toggleMobileMenu(page);
    // At mobile width the "nav" scope IS the desktop sidebar, and the responsive
    // shell hides it on purpose — shell-widths.mjs asserts exactly one navigation
    // per width, so requiring the sidebar to be visible here would contradict it.
    // The mobile equivalent is the menu, asserted next.
    await assertHomeIdentityPair(page, 'mobile-menu', 'QA Zuhause', 'Einheit 12', `Mobiler Menükopf ${viewport.name}`);
    if (process.env.HV_QA_SCREENSHOT_DIR) {
      mkdirSync(process.env.HV_QA_SCREENSHOT_DIR, { recursive: true });
      await page.screenshot({
        path: join(process.env.HV_QA_SCREENSHOT_DIR, 'home-name-nav-mobile-open.png'),
      });
    }
    await toggleMobileMenu(page);
  } else {
    await assertHomeIdentityPair(page, 'nav', 'QA Zuhause', 'Einheit 12', `Navigation ${viewport.name}`);
  }
  if (!(await page.locator('.portal-section-lede').getByText(/Wohnung.*Musterweg 1, 1010 Wien/, { exact: false }).count()) ||
      !(await page.locator('.portal-section-identity').getByText('Mein Zuhause', { exact: true }).count()) ||
      !(await page.getByRole('link', { name: 'Zuhause bearbeiten' }).count())) {
    fail(`Energie ${viewport.name}: Name, offizielle Wohnung oder sichtbarer Bearbeitungsweg fehlt`);
  }
  await page.getByRole('link', { name: 'Zuhause bearbeiten' }).click();
  await page.waitForURL(/\/app\/settings\/home/);
  await assertHomeIdentityPair(page, 'editor-heading', 'QA Zuhause', 'Einheit 12', `Zuhause-Einstellungen ${viewport.name}`);
  await assertHomeIdentityPair(page, 'editor-summary', 'QA Zuhause', 'Einheit 12', `Zuhause-Zusammenfassung ${viewport.name}`);
  if (!(await page.getByRole('heading', { name: 'QA Zuhause', exact: true }).count()) ||
      (await page.locator('input[name="household_name"]').inputValue()) !== 'QA Zuhause' ||
      !(await page.getByText('Einheit 12', { exact: true }).count()) ||
      (await page.locator('[name="unit_id"]').inputValue()) !== 'einheit-12' ||
      !(await page.getByText('Diesem Hausprofil zugeordnet.', { exact: true }).count()) ||
      !(await page.locator('[data-home-type-explanation]').count())) {
    fail(`Energie ${viewport.name}: Hausname ist nicht verständlich mit „Einheit 12“ verknüpft`);
  }
  if (await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth + 1)) {
    fail(`Energie ${viewport.name}: Hausname-Einstellungen laufen horizontal über`);
  }
  if (process.env.HV_QA_SCREENSHOT_DIR) {
    mkdirSync(process.env.HV_QA_SCREENSHOT_DIR, { recursive: true });
    await page.screenshot({
      path: join(process.env.HV_QA_SCREENSHOT_DIR, `home-identity-${viewport.name.toLowerCase()}.png`),
      fullPage: true,
    });
  }
  if (viewport.name === 'Desktop') {
    await page.locator('input[name="household_name"]').fill('Sonnendeck QA');
    await page.getByRole('button', { name: 'Änderungen speichern' }).click();
    await page.waitForURL(/\/app\/energie/);
    await assertHomeIdentityPair(page, 'energy-heading', 'Sonnendeck QA', 'Einheit 12', 'Umbenennung Energie Desktop');
    await assertHomeIdentityPair(page, 'nav', 'Sonnendeck QA', 'Einheit 12', 'Umbenennung Navigation Desktop');
    await page.getByRole('link', { name: 'Zuhause bearbeiten' }).click();
    await page.locator('input[name="household_name"]').fill('QA Zuhause');
    await page.getByRole('button', { name: 'Änderungen speichern' }).click();
    await page.waitForURL(/\/app\/energie/);
  } else {
    const longDisplayName = 'DachterrassenwohnungMitAußergewöhnlichLangemAnzeigenamenFürDieMobileDarstellung';
    await page.locator('input[name="household_name"]').fill(longDisplayName);
    await page.getByRole('button', { name: 'Änderungen speichern' }).click();
    await page.waitForURL(/\/app\/energie/);
    await assertHomeIdentityPair(page, 'energy-heading', longDisplayName, 'Einheit 12', 'Langer Anzeigename Mobil');
    await toggleMobileMenu(page);
    await assertHomeIdentityPair(page, 'mobile-menu', longDisplayName, 'Einheit 12', 'Langer Navigationsname Mobil');
    const longNameOverflow = await page.evaluate(() => ({
      viewport: window.innerWidth,
      documentWidth: document.documentElement.scrollWidth,
      offenders: [...document.querySelectorAll('body *')].filter((node) => {
        if (!node.getClientRects().length) return false;
        const rect = node.getBoundingClientRect();
        return rect.right > window.innerWidth + 1 || rect.left < -1 || node.scrollWidth > node.clientWidth + 1;
      }).slice(0, 16).map((node) => {
        const rect = node.getBoundingClientRect();
        return {
          node: `${node.tagName.toLowerCase()}.${String(node.className || '').trim().replace(/\s+/g, '.')}`,
          left: rect.left,
          right: rect.right,
          client: node.clientWidth,
          scroll: node.scrollWidth,
        };
      }),
    }));
    if (longNameOverflow.documentWidth > longNameOverflow.viewport + 1) {
      fail(`Langer Anzeigename Mobil: Darstellung läuft horizontal über (${JSON.stringify(longNameOverflow)})`);
    }
    await toggleMobileMenu(page);
    await page.getByRole('link', { name: 'Zuhause bearbeiten' }).click();
    await page.locator('input[name="household_name"]').fill('QA Zuhause');
    await page.getByRole('button', { name: 'Änderungen speichern' }).click();
    await page.waitForURL(/\/app\/energie/);
  }
  await page.goto(`${baseURL}/app/settings`, { waitUntil: 'networkidle' });
  await assertHomeIdentityPair(page, 'settings', 'QA Zuhause', 'Einheit 12', `Einstellungen ${viewport.name}`);
  if (process.env.HV_QA_SCREENSHOT_DIR) {
    await page.screenshot({
      path: join(process.env.HV_QA_SCREENSHOT_DIR, `home-name-settings-${viewport.name.toLowerCase()}.png`),
      fullPage: true,
    });
  }
  await page.goto(`${baseURL}/app/energie`, { waitUntil: 'networkidle' });
  await assertEnergyRedesignViewport(page, {
    label: `Energie ${viewport.name}`,
    width: viewport.size.width,
  });
  const live = page.locator('[data-energy-flow-diagram]');
  // The flow diagram must actually MOVE, not merely carry an animation rule. Sample the
  // animated property twice: if the dots' stroke-dashoffset does not change over 400ms the
  // animation is not running — a stylesheet regression, a dropped keyframe, or a JS path
  // that stopped emitting .energy-flow-dots would all read as "present but frozen" here.
  // Only asserted when the diagram has flow edges at all (it does on this fixture).
  //
  // The real contract is "not frozen". When prefers-reduced-motion is on, the CSS correctly
  // disables the animation, so the test must not require motion in that case. When reduced
  // motion is off, pass if the diagram moved OR a running animation is present; fail only
  // when both signals are absent (frozen). When reduced motion is on, fail only if an
  // animation is still running (HAUSV-561).
  const motion = await live.evaluate(async (diagram) => {
    const dots = diagram.querySelector('.energy-flow-dots');
    if (!dots) return { present: false };
    const prefersReducedMotion = matchMedia('(prefers-reduced-motion: reduce)').matches;
    const read = () => getComputedStyle(dots).strokeDashoffset;
    const a = read(); await new Promise((r) => setTimeout(r, 400)); const b = read();
    return { present: true, prefersReducedMotion, animationName: getComputedStyle(dots).animationName, running: (dots.getAnimations?.() || []).length, moved: a !== b, a, b };
  });
  if (motion.present) {
    const hasRunningAnimation = motion.animationName === 'energy-flow-dots' && motion.running > 0;
    if (motion.prefersReducedMotion) {
      // When reduced motion is on, the animation should be disabled. Fail if it's still running.
      if (hasRunningAnimation) {
        fail(`Energie ${viewport.name}: Energiefluss should not animate with prefers-reduced-motion (${JSON.stringify(motion)})`);
      }
    } else {
      // When reduced motion is off, pass if moved OR animation is running. Fail if frozen (neither).
      if (!motion.moved && !hasRunningAnimation) {
        fail(`Energie ${viewport.name}: Energiefluss animiert nicht (${JSON.stringify(motion)})`);
      }
    }
  }
  const chart = page.locator('.energy-chart');
  if (!(await chart.getByRole('heading', { name: 'Letzte 24 Stunden' }).count()) ||
      (await chart.locator('path.energy-chart-line').count()) < 3 ||
      !(await chart.getByText('Die höchste Last lag um', { exact: false }).count())) {
    fail(`Energie ${viewport.name}: verständlicher 24-Stunden-Verlauf fehlt`);
  }
  if ((await chart.getByRole('link', { name: 'Letzte 24 h' }).getAttribute('aria-current')) !== 'page' ||
      !(await chart.getByRole('button', { name: 'Vollbild', exact: true }).count())) {
    fail(`Energie ${viewport.name}: Zeitraum- oder Vollbildsteuerung fehlt`);
  }
  await chart.getByRole('link', { name: 'Heute' }).click();
  await page.waitForLoadState('networkidle');
  if (!(await chart.getByRole('heading', { name: 'Heute · 00 bis 24 Uhr' }).count()) ||
      (await chart.getByRole('link', { name: 'Heute' }).getAttribute('aria-current')) !== 'page') {
    fail(`Energie ${viewport.name}: feste Heute-Ansicht lässt sich nicht auswählen`);
  }
  const todayHeadingBox = await chart.getByRole('heading', { name: 'Heute · 00 bis 24 Uhr' }).boundingBox();
  const stickyModeBox = await page.locator('.energy-mode-strip').boundingBox();
  if (!todayHeadingBox || !stickyModeBox || todayHeadingBox.y < stickyModeBox.y + stickyModeBox.height - 1) {
    fail(`Energie ${viewport.name}: Zeitraumwechsel verdeckt den Diagrammkopf unter der Sicherheitsleiste (${JSON.stringify({ todayHeadingBox, stickyModeBox })})`);
  }
  const todayAxis = await chart.locator('svg.energy-chart-svg:visible text.energy-chart-axis-label').allTextContents();
  for (const label of ['00:00', '06:00', '12:00', '18:00', '24:00']) {
    if (!todayAxis.includes(label)) fail(`Energie ${viewport.name}: Heute-Achse enthält ${label} nicht`);
  }
  const todayHitCount = await chart.locator('svg.energy-chart-svg:visible [data-chart-hit]').count();
  // "Heute" spans 00:00-24:00, so the number of slots that CAN carry a value depends on how
  // much of the day has actually elapsed. The old bound was a flat `< 2`, which is unsatisfiable
  // between 00:00 and 00:30 — the day is one quarter-hour old and one point is all there is.
  // That made this check fail deterministically just after midnight while passing all day, which
  // read as flakiness and blocked four merges. Ask the browser for the day it is rendering
  // rather than the runner's clock, so a timezone difference cannot reintroduce the same gap.
  const elapsedSlots = await page.evaluate(() => {
    const now = new Date();
    return Math.floor((now.getHours() * 60 + now.getMinutes()) / 15);
  });
  // Two separate properties, so a failure says which one broke. The old message claimed
  // "future values are not empty" for BOTH branches, including the one that fires when there is
  // hardly any data at all — the opposite problem, and it sent the reader hunting the wrong way.
  if (todayHitCount < 1) {
    fail(`Energie ${viewport.name}: Heute-Diagramm zeigt keinen einzigen Messwert (${todayHitCount}, ${elapsedSlots} Viertelstunden vergangen)`);
  }
  // +1 absorbs the slot in progress and a tick of the clock between render and assertion.
  const maxPlausible = Math.min(97, elapsedSlots + 2);
  if (todayHitCount > maxPlausible) {
    fail(`Energie ${viewport.name}: zukünftige Heute-Werte sind nicht leer (${todayHitCount} von 97 belegt, höchstens ${maxPlausible} plausibel nach ${elapsedSlots} Viertelstunden)`);
  }
  if (process.env.HV_QA_SCREENSHOT_DIR) {
    mkdirSync(process.env.HV_QA_SCREENSHOT_DIR, { recursive: true });
    await page.screenshot({
      path: join(process.env.HV_QA_SCREENSHOT_DIR, `energy-today-${viewport.name.toLowerCase()}.png`),
    });
  }
  await chart.getByRole('link', { name: 'Letzte 24 h' }).click();
  await page.waitForLoadState('networkidle');
  const visibleChart = chart.locator('svg.energy-chart-svg:visible');
  const thresholdLegend = chart.locator('.energy-chart-legend .threshold').first();
  if (!(await visibleChart.locator('path.energy-chart-area.load').count()) ||
      !(await visibleChart.locator('line.energy-chart-threshold').count()) ||
      !(await thresholdLegend.getByText('Planungsgrenze', { exact: false }).count()) ||
      !(await thresholdLegend.locator('.energy-value[aria-label="10\u00a0kW"] .u').count())) {
    fail(`Energie ${viewport.name}: Verbrauchsfläche oder konfigurierbare 10-kW-Planungsgrenze fehlt`);
  }
  const scaleLabels = await visibleChart.locator('text.energy-chart-axis-label').allTextContents();
  if (!scaleLabels.includes('15\u00a0kW') || !scaleLabels.includes('-15\u00a0kW')) {
    fail(`Energie ${viewport.name}: symmetrische ±15-kW-Skala mit Headroom fehlt (${scaleLabels.join(', ')})`);
  }
  const loadStyle = await visibleChart.locator('path.energy-chart-line.load').evaluate((node) => getComputedStyle(node).stroke);
  if (!loadStyle || loadStyle === 'none' || loadStyle === 'rgb(27, 32, 26)') {
    fail(`Energie ${viewport.name}: Hausverbrauch ist nicht eigenständig rot ausgezeichnet`);
  }
  const chartInteraction = chart.locator('[data-energy-chart-interactive]').first();
  const chartHit = visibleChart.locator('[data-chart-hit]').nth(20);
  await chartHit.hover();
  const chartTooltip = chartInteraction.locator('[data-chart-tooltip]');
  await chartTooltip.waitFor({ state: 'visible' });
  if (!(await chartTooltip.getByText('Hausverbrauch', { exact: true }).count()) ||
      !(await chartTooltip.getByText('PV-Erzeugung', { exact: true }).count()) ||
      !(await chartTooltip.getByText(/Netzbezug|Einspeisung/, { exact: true }).count()) ||
      !(await chartTooltip.getByText(/Speicher lädt|Speicher entlädt/, { exact: true }).count()) ||
      !(await visibleChart.locator('[data-chart-guide]:not([hidden])').count()) ||
      !(await visibleChart.locator('[data-chart-marker-index]:not([hidden])').count())) {
    fail(`Energie ${viewport.name}: Hover erklärt den Viertelstundenwert nicht vollständig`);
  }
  await chartInteraction.focus();
  await page.keyboard.press('End');
  if (!(await chartTooltip.isVisible()) || (await chartTooltip.getAttribute('data-index')) === null) {
    fail(`Energie ${viewport.name}: Diagrammwerte sind nicht mit der Tastatur erreichbar`);
  }

  const mainChartBox = await visibleChart.boundingBox();
  const zoomButton = chart.getByRole('button', { name: 'Vergrößern' });
  await zoomButton.click();
  const chartDialog = page.locator('#energy-chart-dialog');
  await chartDialog.waitFor({ state: 'visible' });
  const fullscreenButton = chartDialog.getByRole('button', { name: 'Vollbild', exact: true });
  await fullscreenButton.click();
  await page.waitForFunction(() => {
    const dialog = document.querySelector('#energy-chart-dialog');
    const surface = dialog?.querySelector('[data-energy-fullscreen-surface]');
    return Boolean(surface && document.fullscreenElement === surface) || Boolean(dialog?.classList.contains('is-fullscreen-fallback'));
  });
  const fullscreenSurface = chartDialog.locator('[data-energy-fullscreen-surface]');
  const fullscreenBox = await fullscreenSurface.boundingBox();
  if (!fullscreenBox || fullscreenBox.width < viewport.size.width - 2 || fullscreenBox.height < viewport.size.height - 2) {
    fail(`Energie ${viewport.name}: Vollbild nutzt nicht den ganzen Bildschirm`);
  }
  if (process.env.HV_QA_SCREENSHOT_DIR) {
    mkdirSync(process.env.HV_QA_SCREENSHOT_DIR, { recursive: true });
    await page.screenshot({
      path: join(process.env.HV_QA_SCREENSHOT_DIR, `energy-fullscreen-${viewport.name.toLowerCase()}.png`),
    });
  }
  await chartDialog.getByRole('button', { name: 'Vollbild beenden', exact: true }).click();
  await page.waitForFunction(() => {
    const dialog = document.querySelector('#energy-chart-dialog');
    return !document.fullscreenElement && !dialog?.classList.contains('is-fullscreen-fallback');
  });
  if (!(await chartDialog.isVisible()) ||
      (await fullscreenButton.getAttribute('aria-pressed')) !== 'false') {
    fail(`Energie ${viewport.name}: „Vollbild beenden“ beendet den Vollbildmodus nicht sauber`);
  }
  await fullscreenButton.click();
  await page.waitForFunction(() => {
    const dialog = document.querySelector('#energy-chart-dialog');
    const surface = dialog?.querySelector('[data-energy-fullscreen-surface]');
    return Boolean(surface && document.fullscreenElement === surface) || Boolean(dialog?.classList.contains('is-fullscreen-fallback'));
  });
  await page.keyboard.press('Escape');
  await page.waitForFunction(() => {
    const dialog = document.querySelector('#energy-chart-dialog');
    return !document.fullscreenElement && !dialog?.classList.contains('is-fullscreen-fallback');
  });
  if (!(await chartDialog.isVisible()) || !(await fullscreenButton.evaluate((node) => document.activeElement === node))) {
    fail(`Energie ${viewport.name}: erstes Escape beendet nicht nur das Vollbild mit sauberem Fokus`);
  }
  const zoomChart = chartDialog.locator('[data-energy-chart-interactive]');
  const zoomSVG = zoomChart.locator('svg:visible');
  const zoomChartBox = await zoomSVG.boundingBox();
  if (!zoomChartBox || (viewport.name === 'Desktop' && mainChartBox && zoomChartBox.width <= mainChartBox.width)) {
    fail(`Energie ${viewport.name}: vergrößerter Verlauf ist nicht größer als die Übersicht`);
  }
  await zoomChart.focus();
  await page.keyboard.press('ArrowLeft');
  if (!(await zoomChart.locator('[data-chart-tooltip]').isVisible())) {
    fail(`Energie ${viewport.name}: vergrößerter Verlauf zeigt keine Tastaturwerte`);
  }
  if (process.env.HV_QA_SCREENSHOT_DIR) {
    mkdirSync(process.env.HV_QA_SCREENSHOT_DIR, { recursive: true });
    await page.screenshot({
      path: join(process.env.HV_QA_SCREENSHOT_DIR, `energy-zoom-${viewport.name.toLowerCase()}.png`),
    });
  }
  await page.keyboard.press('Escape');
  await chartDialog.waitFor({ state: 'hidden' });
  if (!(await zoomButton.evaluate((node) => document.activeElement === node))) {
    fail(`Energie ${viewport.name}: Fokus kehrt nach dem Schließen nicht zum Vergrößern-Knopf zurück`);
  }
  // Der Desktop-Durchlauf läuft zuerst und importiert erst weiter unten eine
  // Viertelstunden-Datei; hier ist der Monat also noch ohne Grundlage. Eine
  // frische Home-Assistant-Verbindung darf dann NICHT beruhigen — der
  // Leistungstarif bemisst die Monatsspitze (HAUSV-430). Im Mobil-Durchlauf
  // liegt der Import des Desktop-Laufs bereits vor, dort gilt das Gegenteil.
  if (viewport.name === 'Desktop') {
    if (await page.getByText('Keine Aktion nötig.', { exact: true }).count()) {
      fail(`Energie ${viewport.name}: Datenlage beruhigt, obwohl für den Monat nichts gemessen ist`);
    }
    const tariffWaiting = await page.locator('.energy-tariff-empty')
      .getByText('Noch keine volle Viertelstunde', { exact: true }).count();
    const monthlyWarning = (await page.getByText('Keine Viertelstunde in diesem Monat', { exact: true }).count()) ||
      (await page.getByText('Messlücke erkannt', { exact: true }).count());
    if (!tariffWaiting || !monthlyWarning) {
      fail(`Energie ${viewport.name}: fehlende Monatsgrundlage wird nicht benannt`);
    }
  } else if (!(await page.getByText('Messwerte aktuell', { exact: true }).count()) &&
      // Der Sampler darf für sein angebrochenes Start-Viertel bewusst eine
      // Gap-Zeile schreiben (HAUSV-428); ein veralteter Live-Zeitpunkt bleibt
      // dagegen ein Fehler.
      !(await page.getByText('Messlücke erkannt', { exact: true }).count())) {
    fail(`Energie ${viewport.name}: Live-Aktualität verwendet nicht den aktuellen Home-Assistant-Zeitpunkt`);
  } else if (await page.getByText('Messwert nicht aktuell', { exact: true }).count()) {
    fail(`Energie ${viewport.name}: frische Live-Werte werden als veraltet ausgewiesen`);
  }
  const chartBox = await visibleChart.boundingBox();
  if (!chartBox || chartBox.width > viewport.size.width + 1) {
    fail(`Energie ${viewport.name}: 24-Stunden-Diagramm läuft aus dem sichtbaren Bereich`);
  }
  const strip = page.locator('.energy-mode-strip');
  if ((await strip.locator('strong').first().innerText()).trim() !== 'Nur beobachten') {
    fail(`Energie ${viewport.name}: startet nicht in Nur beobachten`);
  }
  // <summary> is the native disclosure control for <details>; user agents do
  // not expose one uniform ARIA role for it. Target the native control and use
  // the keyboard so this assertion follows the real interaction contract.
  const modeDisclosure = page.locator('summary.energy-mode-action');
  if ((await modeDisclosure.count()) !== 1 ||
      (await modeDisclosure.getAttribute('aria-label')) !== 'Wirkungslosen Testlauf bewusst starten' ||
      (await modeDisclosure.innerText()).trim() !== 'Testlauf') {
    fail(`Energie ${viewport.name}: Testlauf-Offenlegung ist nicht eindeutig beschriftet`);
  }
  await modeDisclosure.focus();
  await page.keyboard.press('Enter');
  const modeForm = page.locator('.energy-mode-popover');
  await modeForm.waitFor({ state: 'visible' });
  await modeForm.locator('input[type="checkbox"]').check();
  await modeForm.locator('input[name="confirmation_text"]').fill('TESTLAUF');
  await modeForm.getByRole('button', { name: 'Testlauf starten' }).click();
  await page.waitForLoadState('networkidle');
  if ((await strip.locator('.energy-mode-copy strong').innerText()).trim() !== 'Testlauf aktiv' ||
      (await strip.locator('.energy-mode-copy span').innerText()).trim() !== 'Keine Gerätewirkung') {
    fail(`Energie ${viewport.name}: Freigabe startet nicht im Testlauf`);
  }
  if (!((await strip.locator('.energy-mode-copy span').getAttribute('title')) || '').includes('schaltet kein Gerät')) {
    fail(`Energie ${viewport.name}: Shadow-Mode-Erklärung fehlt`);
  }
  await page.getByRole('button', { name: /Sofort zurück/ }).click();
  await page.waitForLoadState('networkidle');
  if ((await strip.locator('strong').first().innerText()).trim() !== 'Nur beobachten') {
    fail(`Energie ${viewport.name}: Sofort-Rückkehr fehlgeschlagen`);
  }

  if (viewport.name === 'Desktop') {
    const measurementPanel = page.locator('details.energy-collapsible').filter({ hasText: 'Messwerte & Referenz' });
    await measurementPanel.locator(':scope > summary').click();
    const csv = currentMonthQuarterHourCSV();
    const file = { name: 'smart-meter.csv', mimeType: 'text/csv', buffer: csv };
    await page.locator('input[name="smart_meter_file"]').setInputFiles(file);
    await page.getByRole('button', { name: 'Als Referenz importieren' }).click();
    await page.waitForLoadState('networkidle');
    if (!(await page.getByText('Smart-Meter-Datei übernommen.', { exact: false }).count())) {
      fail('Energie Desktop: Smart-Meter-Import nicht bestätigt');
    }
    await measurementPanel.locator(':scope > summary').click();
    await page.locator('input[name="smart_meter_file"]').setInputFiles(file);
    await page.getByRole('button', { name: 'Als Referenz importieren' }).click();
    await page.waitForLoadState('networkidle');
    if (!(await page.getByText('bereits vorhanden', { exact: false }).count())) {
      fail('Energie Desktop: doppelter Import nicht erkannt');
    }

    // Jetzt gibt es Viertelstunden im laufenden Monat UND frische Live-Werte.
    // Der Sampler darf für sein angebrochenes Start-Viertel bewusst eine
    // Gap-Zeile schreiben (HAUSV-428) — je nach Laufzeit dieses QA-Laufs ist
    // die ehrliche Aussage deshalb „aktuell" oder „Messlücke erkannt".
    // Niemals zulässig sind hier Stale-, Konflikt- oder Leerzustände.
    const quietOK = await page.getByText('Messwerte aktuell', { exact: true }).count();
    const restartGapOK = await page.getByText('Messlücke erkannt', { exact: true }).count();
    if (!quietOK && !restartGapOK) {
      fail('Energie Desktop: mit Monatsgrundlage und frischen Werten fehlt die ruhige Datenlage');
    }
    if (await page.getByText('Messwert nicht aktuell', { exact: true }).count()) {
      fail('Energie Desktop: frische Live-Werte werden als veraltet ausgewiesen');
    }

    const tariffDetails = page.locator('details.energy-tariff-disclosure');
    await tariffDetails.locator(':scope > summary').click();
    await tariffDetails.getByRole('button', { name: 'Diesen Stand festhalten' }).click();
    await page.waitForLoadState('networkidle');
    if (!(await page.getByText('Festgehaltene Bewertungen', { exact: true }).count())) {
      fail('Energie Desktop: Tarifstand wurde nicht historisch sichtbar');
    }
    // 0.69.0: with an estimate the Monatsspitze lives as a chip in the mode
    // strip on roomy desktops and links to the detail card.
    if (viewport.size.width >= 1240) {
      const chip = page.locator('.energy-peak-chip');
      if (!(await chip.isVisible()) ||
          !/monatsspitze/i.test(await chip.innerText()) ||
          (await chip.getAttribute('href')) !== '#tarif') {
        fail('Energie Desktop: Monatsspitzen-Chip fehlt oder verlinkt nicht auf die Detailkarte');
      }
    }

    // 0.70.0: Prioritäten sind bedienbar — Pfeiltaste ordnet um, die
    // Reihenfolge überlebt das Neuladen und wird danach zurückgestellt.
    const railTiles = page.locator('.energy-flow-rail .energy-flow-big:not(.ghost)');
    if ((await railTiles.count()) >= 2) {
      const titlesBefore = await railTiles.locator('.energy-flow-copy > b').allTextContents();
      await railTiles.first().locator('button.drag').focus();
      const orderSaved = page.waitForResponse((response) =>
        response.url().includes('/app/energie/verbraucher/reihenfolge') && response.status() === 204);
      await page.keyboard.press('ArrowDown');
      await orderSaved;
      const titlesAfter = await page.locator('.energy-flow-rail .energy-flow-big:not(.ghost) .energy-flow-copy > b').allTextContents();
      if (titlesAfter[0] !== titlesBefore[1] || titlesAfter[1] !== titlesBefore[0]) {
        fail(`Energie Desktop: Prioritäten-Umsortierung greift nicht (${titlesBefore} -> ${titlesAfter})`);
      }
      await page.reload({ waitUntil: 'networkidle' });
      const titlesReloaded = await page.locator('.energy-flow-rail .energy-flow-big:not(.ghost) .energy-flow-copy > b').allTextContents();
      if (titlesReloaded[0] !== titlesAfter[0]) {
        fail(`Energie Desktop: Prioritäten-Reihenfolge überlebt das Neuladen nicht (${titlesReloaded})`);
      }
      await page.locator('.energy-flow-rail .energy-flow-big:not(.ghost)').nth(1).locator('button.drag').focus();
      const restoreSaved = page.waitForResponse((response) =>
        response.url().includes('/app/energie/verbraucher/reihenfolge') && response.status() === 204);
      await page.keyboard.press('ArrowUp');
      await restoreSaved;
    }
    await assertEnergyMetricDisclosures(page, {
      label: 'Energie Desktop nach Messimport',
      width: viewport.size.width,
    });

    const maintenance = page.locator('details.energy-compact-create').filter({ hasText: 'Wartung an einer Anlage vormerken' });
    await maintenance.locator('summary').click();
    await maintenance.locator('input[name="title"]').fill('QA PV-Sichtprüfung');
    await maintenance.locator('input[name="next_due"]').fill(new Date(Date.now() + 30 * 86400000).toISOString().slice(0, 10));
    await maintenance.locator('button[type="submit"]').click();
    await page.waitForLoadState('networkidle');
    if (!(await page.getByText('QA PV-Sichtprüfung', { exact: true }).count())) {
      fail('Energie Desktop: wiederkehrende Wartung wurde nicht sichtbar');
    }

    const caretakerPanel = page.locator('details.energy-collapsible').filter({ hasText: 'Technische Betreuung' });
    await caretakerPanel.locator(':scope > summary').click();
    const invitation = caretakerPanel.locator('details.energy-compact-create').filter({ hasText: 'Technische Vertrauensperson einladen' });
    await invitation.locator('summary').click();
    await invitation.locator('input[name="first_name"]').fill('QA');
    await invitation.locator('input[name="last_name"]').fill('Hilfe');
    await invitation.locator('input[name="email"]').fill('qa-helper@example.com');
    await invitation.locator('input[value="configure"]').check();
    await invitation.getByRole('button', { name: 'Zur Liegenschaft einladen' }).click();
    await page.waitForLoadState('networkidle');
    if (!(await page.getByText(/Einladung verschickt|Zugang gespeichert/).count())) {
      fail('Energie Desktop: hausbezogene Betreuungseinladung ohne Rückmeldung');
    }
    const helperContext = await newContext({ width: 390, height: 844 });
    const helper = await localLogin(helperContext, 'qa-helper@example.com');
    const helperEnergy = await helper.goto(`${baseURL}/app/energie`, { waitUntil: 'networkidle' });
    if (!helperEnergy || helperEnergy.status() !== 200 ||
        !(await helper.getByText('Nur beobachten', { exact: true }).count())) {
      fail('Energie Desktop: eingeladene Vertrauensperson kann den hausbezogenen Zugang nicht annehmen');
    }
    if (await helper.locator('summary.energy-mode-action').count()) {
      fail('Energie Desktop: technische Vertrauensperson sieht die Eigentümer-/Admin-Freigabe');
    }
    if (await helper.getByRole('link', { name: 'Zuhause bearbeiten' }).count()) {
      fail('Energie Desktop: technische Vertrauensperson kann die gemeinsame Hausidentität bearbeiten');
    }
    const helperIdentity = await helper.goto(`${baseURL}/app/settings/home`, { waitUntil: 'networkidle' });
    if (!helperIdentity || helperIdentity.status() !== 403) {
      fail('Energie Desktop: Hausidentität ist für technische Vertrauensperson nicht mit 403 geschützt');
    }
    await closeContext(helperContext);

    const recommendationDialog = page.locator('#energy-recommendation-dialog');
    await page.locator('[data-dialog="energy-recommendation-dialog"]').click();
    await recommendationDialog.waitFor({ state: 'visible' });
    const measureControl = recommendationDialog.locator('details.energy-measure-control');
    await measureControl.locator('summary').click();
    const measureBox = await measureControl.locator('.energy-measure-form').boundingBox();
    const dialogBox = await recommendationDialog.boundingBox();
    if (!measureBox || !dialogBox || measureBox.x < dialogBox.x - 1 ||
        measureBox.x + measureBox.width > dialogBox.x + dialogBox.width + 1) {
      fail('Energie Desktop: Hausaufgaben-Formular bleibt nicht im Empfehlungsdialog');
    }
    if (process.env.HV_QA_SCREENSHOT_DIR) {
      mkdirSync(process.env.HV_QA_SCREENSHOT_DIR, { recursive: true });
      await page.screenshot({
        path: join(process.env.HV_QA_SCREENSHOT_DIR, 'energy-many-metrics-desktop.png'),
        fullPage: true,
      });
      await chart.screenshot({
        path: join(process.env.HV_QA_SCREENSHOT_DIR, 'energy-24h-desktop.png'),
      });
    }
    await measureControl.locator('input[value="inventory"]').check();
    await measureControl.getByRole('button', { name: 'Hausaufgabe anlegen' }).click();
    await page.waitForLoadState('networkidle');
    if (await page.locator('#energy-recommendation-dialog').isVisible().catch(() => false)) {
      await page.locator('#energy-recommendation-dialog [data-close-dialog]').click();
    }
    const specialistPanel = page.locator('details.energy-collapsible').filter({ hasText: 'Fachhilfe, wenn sie wirklich nötig ist' });
    await specialistPanel.locator(':scope > summary').click();
    const measure = specialistPanel.locator('details.energy-measure-row').first();
    await measure.locator('summary').click();
    await measure.locator('select[name="contact_id"]').selectOption({ label: 'QA Energiehilfe · Wien und Umgebung' });
    await measure.locator('input[name="offer_note"]').fill('Messkonzept angefragt');
    await measure.getByRole('button', { name: 'Maßnahmenstand speichern' }).click();
    await page.waitForLoadState('networkidle');
    if (!(await page.getByText('QA Energiehilfe', { exact: false }).count())) {
      fail('Energie Desktop: kuratierter Fachkontakt fehlt an der Maßnahme');
    }
  }

  if (viewport.name === 'Mobil') {
    await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight / 2));
    const box = await strip.boundingBox();
    const mobileNav = await contextBarBox(page);
    if (!box || !mobileNav || Math.abs(box.y - (mobileNav.y + mobileNav.height)) > 1) {
      fail(`Energie Mobil: Modus nicht sauber unter der Navigation (${JSON.stringify({ box, mobileNav })})`);
    }
    const primaryTargets = await page.locator('.energy-page .button, .energy-mode-action').evaluateAll((nodes) =>
      nodes.filter((node) => {
        const rect = node.getBoundingClientRect();
        return rect.width > 0 && rect.height > 0 && rect.height < 44;
      }).map((node) => (node.textContent || '').trim())
    );
    if (primaryTargets.length) fail(`Energie Mobil: Touch-Ziele unter 44px: ${primaryTargets.join(', ')}`);
  }
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth + 1);
  if (overflow) fail(`Energie ${viewport.name}: horizontaler Überlauf`);
  if (process.env.HV_QA_SCREENSHOT_DIR) {
    mkdirSync(process.env.HV_QA_SCREENSHOT_DIR, { recursive: true });
    await page.screenshot({
      path: join(process.env.HV_QA_SCREENSHOT_DIR, `energy-${viewport.name.toLowerCase()}.png`),
      fullPage: true,
    });
    if (viewport.name === 'Mobil') {
      await page.screenshot({
        path: join(process.env.HV_QA_SCREENSHOT_DIR, 'energy-many-metrics-mobile.png'),
        fullPage: true,
      });
      await chart.screenshot({
        path: join(process.env.HV_QA_SCREENSHOT_DIR, 'energy-24h-mobile.png'),
      });
    }
  }
  const liveInfo = live.locator('.energy-info-disclosure');
  await liveInfo.locator(':scope > summary').click();
  await liveInfo.getByRole('link', { name: 'Messwerte zuordnen' }).click();
  await page.waitForLoadState('networkidle');
  const setup = page.locator('.energy-mapping-guide');
  if ((await setup.locator('[data-mapping-slot]').count()) !== 6 ||
      !(await setup.getByText('Viertelstunden-Spitzen', { exact: true }).count()) ||
      !(await setup.getByText('Speicherfüllstand', { exact: true }).count()) ||
      !(await setup.getByText('Alles bleibt nur gelesen.', { exact: true }).count())) {
    fail(`Energie ${viewport.name}: verständliches Messwert-Rollen-Setup fehlt`);
  }
  if (await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth + 1)) {
    fail(`Energie ${viewport.name}: Messwert-Setup läuft horizontal über`);
  }
  if (process.env.HV_QA_SCREENSHOT_DIR) {
    await page.screenshot({
      path: join(process.env.HV_QA_SCREENSHOT_DIR, `energy-mapping-${viewport.name.toLowerCase()}.png`),
      fullPage: true,
    });
  }
  await closeContext(ownerContext);
}

// Canonical viewport matrix for the energy lead. Horizontal document overflow
// alone did not catch the former tariff/live breakpoint collision, so this also asserts
// DOM order, sibling geometry, card scroll width and sticky-stack alignment.
async function assertEnergyGeometryMatrix() {
  const sizes = [
    { name: '320x568', width: 320, height: 568 },
    { name: '359x780', width: 359, height: 780 },
    { name: '360x800', width: 360, height: 800 },
    { name: '361x800', width: 361, height: 800 },
    { name: '390x844', width: 390, height: 844 },
    { name: '430x932', width: 430, height: 932 },
    { name: '559x900', width: 559, height: 900 },
    { name: '560x900', width: 560, height: 900 },
    { name: '561x900', width: 561, height: 900 },
    { name: '619x900', width: 619, height: 900 },
    { name: '620x900', width: 620, height: 900 },
    { name: '621x900', width: 621, height: 900 },
    { name: '768x1024', width: 768, height: 1024 },
    { name: '899x900', width: 899, height: 900 },
    { name: '900x900', width: 900, height: 900 },
    { name: '901x900', width: 901, height: 900 },
    { name: '1050x900', width: 1050, height: 900 },
    { name: '1051x900', width: 1051, height: 900 },
    { name: '1023x768', width: 1023, height: 768 },
    { name: '1024x768', width: 1024, height: 768 },
    { name: '1280x800', width: 1280, height: 800 },
    { name: '1366x768', width: 1366, height: 768 },
    { name: '1439x900', width: 1439, height: 900 },
    { name: '1440x900', width: 1440, height: 900 },
    { name: '1441x900', width: 1441, height: 900 },
    { name: '1600x900', width: 1600, height: 900 },
    { name: '1672x941', width: 1672, height: 941 },
    { name: '1920x1080', width: 1920, height: 1080 },
    { name: '2048x1152', width: 2048, height: 1152 },
  ];
  const selectedSizes = process.env.HV_QA_ENERGY_PAGE_END_ONLY === 'true'
    ? sizes.filter(({ width }) => width >= 1920)
    : sizes;

  for (const size of selectedSizes) {
    const context = await newContext({ width: size.width, height: size.height });
    const page = await localLogin(context, 'owner@example.com');
    const response = await page.goto(`${baseURL}/app/energie`, { waitUntil: 'networkidle' });
    if (!response || response.status() !== 200) {
      fail(`Energie-Geometrie ${size.name}: Cockpit nicht erreichbar`);
    }
    await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight));
    // The context bar occupies the viewport's top edge. The mode strip meets its
    // bottom at every width; the desktop/tablet sidebar fills the remaining height.
    const phone = size.width <= 760;
    const contextBar = await contextBarBox(page);
    const sidebarAtPageEnd = await page.evaluate(({ mobile, barHeight }) => {
      const strip = document.querySelector('.energy-mode-strip')?.getBoundingClientRect();
      const anchor = mobile ? null : document.querySelector('.sidebar')?.getBoundingClientRect();
      return {
        barHeight,
        stripTop: strip?.top ?? -1,
        anchorTop: anchor?.top ?? -1,
        anchorBottom: anchor?.bottom ?? -1,
        viewportHeight: window.innerHeight,
      };
    }, { mobile: phone, barHeight: contextBar.height });
    if (Math.abs(sidebarAtPageEnd.stripTop - sidebarAtPageEnd.barHeight) > 1 ||
        (!phone && (Math.abs(sidebarAtPageEnd.anchorTop - sidebarAtPageEnd.barHeight) > 1 ||
          Math.abs(sidebarAtPageEnd.anchorBottom - sidebarAtPageEnd.viewportHeight) > 1))) {
      fail(`Energie-Geometrie ${size.name}: Seitenleiste schließt am Seitenende nicht mit dem Viewport ab (${JSON.stringify(sidebarAtPageEnd)})`);
    }
    await page.evaluate(() => window.scrollTo(0, 0));
    if (energyRedesignWidths.has(size.width)) {
      await assertEnergyRedesignViewport(page, {
        label: `Energie-Redesign ${size.name}`,
        width: size.width,
      });
    }

    const result = await page.evaluate(({ width, height }) => {
      const rectOf = (node) => {
        const rect = node?.getBoundingClientRect();
        return rect ? {
          top: rect.top, right: rect.right, bottom: rect.bottom, left: rect.left,
          width: rect.width, height: rect.height,
        } : null;
      };
      const intersects = (left, right) => Boolean(left && right &&
        Math.min(left.right, right.right) - Math.max(left.left, right.left) > 1 &&
        Math.min(left.bottom, right.bottom) - Math.max(left.top, right.top) > 1);
      const health = document.querySelector('.energy-health');
      const lead = health?.querySelector(':scope > .energy-lead-side');
      // Since 0.69.0 the tariff detail card lives below the chart, page-level.
      const tariff = document.querySelector('.energy-tariff');
      const live = lead?.querySelector('.energy-live');
      const recommendationTrigger = live?.querySelector('[data-dialog="energy-recommendation-dialog"]');
      const strip = document.querySelector('.energy-mode-strip');
      const sidebar = document.querySelector('.sidebar');
      const heading = document.querySelector('.energy-main > .portal-section-header');
      const action = strip?.querySelector('.energy-mode-action');
      const modeState = strip?.querySelector('.energy-mode-state');
      const liveRect = rectOf(live);
      const tariffRect = rectOf(tariff);
      const headingRect = rectOf(heading);
      const healthRect = rectOf(health);
      const stripRect = rectOf(strip);
      const sidebarRect = rectOf(sidebar);
      const actionRect = rectOf(action);
      const modeStateRect = rectOf(modeState);
      const tariffMetricRects = [...(tariff?.querySelectorAll('.energy-billed > div') || [])]
        .filter((node) => node.getClientRects().length)
        .map(rectOf);
      const flowArea = live?.querySelector('.energy-flow-area');
      const overflow = [health, lead, tariff, live, flowArea]
        .filter(Boolean)
        .filter((node) => node.scrollWidth > node.clientWidth + 1)
        .map((node) => node.className);
      const overflowDetails = [...(health?.querySelectorAll('*') || [])]
        .filter((node) => node.scrollWidth > node.clientWidth + 1)
        .slice(0, 12)
        .map((node) => ({
          node: `${node.tagName.toLowerCase()}.${String(node.className || '').trim().replace(/\s+/g, '.')}`,
          client: node.clientWidth,
          scroll: node.scrollWidth,
          text: (node.textContent || '').trim().replace(/\s+/g, ' ').slice(0, 80),
        }));
      const stripStyle = strip ? getComputedStyle(strip) : null;
      return {
        width,
        height,
        missing: [health, lead, tariff, live, recommendationTrigger, strip].some((node) => !node),
        documentOverflow: document.documentElement.scrollWidth > window.innerWidth + 1,
        overflow,
        overflowDetails,
        siblingOverlap: intersects(liveRect, tariffRect),
        headingBeforeHealth: Boolean(headingRect && healthRect && headingRect.bottom <= healthRect.top + 1),
        modeControlOverlap: intersects(modeStateRect, actionRect),
        sourceLeadFirst: Boolean(lead && tariff &&
          (lead.compareDocumentPosition(tariff) & Node.DOCUMENT_POSITION_FOLLOWING)),
        oneColumn: Boolean(liveRect && tariffRect && Math.abs(liveRect.left - tariffRect.left) <= 1),
        leadBeforeTariff: Boolean(liveRect && tariffRect && liveRect.bottom <= tariffRect.top + 1),
        liveStartsInViewport: Boolean(live && live.getBoundingClientRect().top < height),
        recommendationInLive: Boolean(recommendationTrigger && live.contains(recommendationTrigger)),
        tariffMetricCount: tariffMetricRects.length,
        tariffMetricsOverlap: tariffMetricRects.length === 2 && intersects(tariffMetricRects[0], tariffMetricRects[1]),
        tariffMetricsOneColumn: tariffMetricRects.length !== 2 || Math.abs(tariffMetricRects[0].left - tariffMetricRects[1].left) <= 1,
        stripRect,
        sidebarRect,
        actionHeight: action?.getBoundingClientRect().height || 0,
        actionLabel: action?.innerText.trim() || '',
        safetyTitle: strip?.querySelector('.energy-mode-copy strong')?.textContent?.trim() || '',
        safetyCopyVisible: Boolean(strip?.querySelector('.energy-mode-copy span')?.getClientRects().length),
        capabilityVisible: Boolean(strip?.querySelector('.energy-mode-control, .energy-mode-capability, form')?.getClientRects().length),
        stripDisplay: stripStyle?.display || '',
        stripFlexWrap: stripStyle?.flexWrap || '',
        softToken: stripStyle?.getPropertyValue('--soft').trim() || '',
        accentToken: stripStyle?.getPropertyValue('--gold-ink').trim() || '',
        focusToken: stripStyle?.getPropertyValue('--energy-focus-ring').trim() || '',
      };
    }, size);

    if (result.missing || result.documentOverflow || result.overflow.length ||
        result.siblingOverlap || result.modeControlOverlap || !result.headingBeforeHealth ||
        !result.sourceLeadFirst || !result.recommendationInLive ||
        result.tariffMetricsOverlap ||
        !result.liveStartsInViewport) {
      fail(`Energie-Geometrie ${size.name}: Grundlayout verletzt (${JSON.stringify(result)})`);
    }
    // One shared column at every width since 0.69.0: live first, tariff detail below the chart.
    if (!result.oneColumn || !result.leadBeforeTariff) {
      fail(`Energie-Geometrie ${size.name}: Cockpit ist nicht live-zuerst in einer Spalte (${JSON.stringify(result)})`);
    }
    // The legacy cockpit switched the two tariff metrics from one column to two at exactly
    // 380px, and this probe pinned that seam. The templ cockpit stacks them at every phone
    // width and has no 380px rule — a design decision, not a regression. What still holds and
    // is asserted above: no overlap (tariffMetricsOverlap) and no overflow.
    // The legacy strip hid its safety copy under 1024px. The templ strip keeps it visible and
    // deliberately wraps when its container narrows (HAUSV-558). Pin containment, ordering,
    // and touch size without reintroducing the old fixed one-row height.
    // 44px is the TOUCH floor and applies through the tablet range (energy.templ's ≤900px
    // rule); on a mouse-driven desktop the action is the shell's standard 40px button.
    const actionFloor = size.width <= 900 ? 44 : 40;
    if (result.safetyTitle !== 'Nur beobachten' || !result.safetyCopyVisible ||
        !result.capabilityVisible || result.actionHeight < actionFloor || result.actionLabel !== 'Testlauf') {
      fail(`Energie-Geometrie ${size.name}: Sicherheitszustand oder Freigabe fehlt (${JSON.stringify(result)})`);
    }
    if (result.softToken !== '#716d62' || result.accentToken !== '#705c22' ||
        result.focusToken !== '#ad862c') {
      fail(`Energie-Geometrie ${size.name}: scoped AA-Tokens fehlen (${JSON.stringify(result)})`);
    }
    const stripMisaligned = !result.stripRect || Math.abs(result.stripRect.top - contextBar.height) > 1;
    const stripWrapPreserved = size.width <= 900
      ? result.stripDisplay === 'grid'
      : result.stripFlexWrap === 'wrap';
    if (!result.stripRect || !stripWrapPreserved || stripMisaligned) {
      fail(`Energie-Geometrie ${size.name}: Navigation/Sicherheitsleiste kollidiert (${JSON.stringify(result)})`);
    }
    if (size.width <= 560 && result.actionHeight > 46) {
      fail(`Energie-Geometrie ${size.name}: mobile Freigabe ist nicht kompakt (${JSON.stringify(result)})`);
    }
    if (size.name === '901x900') {
      const control = page.locator('summary.energy-mode-action');
      if (!(await control.count())) {
        fail(`Energie-Geometrie ${size.name}: Testlauf-Schalter fehlt für die active-mode Zeile`);
      }
      await control.click();
      await page.locator('.energy-mode-popover input[name="confirm"]').check();
      await page.locator('.energy-mode-popover input[name="confirmation_text"]').fill('TESTLAUF');
      await Promise.all([
        page.waitForNavigation({ waitUntil: 'networkidle' }),
        page.getByRole('button', { name: 'Testlauf starten' }).click(),
      ]);
      const active = await page.evaluate(() => {
        const strip = document.querySelector('.energy-mode-strip');
        const rect = strip?.getBoundingClientRect();
        return {
          active: Boolean(strip?.classList.contains('active')),
          top: rect?.top ?? -1,
          height: rect?.height ?? 0,
          label: strip?.querySelector('.energy-mode-action')?.innerText.trim() || '',
        };
      });
      const activeContextBar = await contextBarBox(page);
      if (!active.active || Math.abs(active.top - activeContextBar.height) > 1) {
        fail(`Energie-Geometrie ${size.name} active: Sicherheitsleiste verliert ihre Position (${JSON.stringify(active)})`);
      }
      await Promise.all([
        page.waitForNavigation({ waitUntil: 'networkidle' }),
        page.locator('button.energy-mode-action').click(),
      ]);
    }

    if (size.name === '390x844') {
      const stripHeight = result.stripRect.height;
      const control = page.locator('.energy-mode-control > summary');
      await control.focus();
      await control.press('Enter');
      const openState = await page.evaluate(() => {
        const strip = document.querySelector('.energy-mode-strip');
        const popover = document.querySelector('.energy-mode-popover');
        const trigger = document.querySelector('.energy-mode-control > summary');
        const stripRect = strip?.getBoundingClientRect();
        const popoverRect = popover?.getBoundingClientRect();
        const triggerRect = trigger?.getBoundingClientRect();
        return {
          open: Boolean(document.querySelector('.energy-mode-control[open]')),
          stripHeight: stripRect?.height || 0,
          triggerBottom: triggerRect?.bottom || 0,
          popoverTop: popoverRect?.top || 0,
          popoverLeft: popoverRect?.left || 0,
          popoverRight: popoverRect?.right || 0,
          popoverBottom: popoverRect?.bottom || 0,
        };
      });
      if (!openState.open || Math.abs(openState.stripHeight - stripHeight) > 1 ||
          openState.popoverTop < openState.triggerBottom || openState.popoverLeft < -1 ||
          openState.popoverRight > size.width + 1 ||
          openState.popoverBottom > size.height + 1) {
        fail(`Energie-Geometrie ${size.name}: Testlauf-Erklärung verdrängt oder verlässt den Viewport (${JSON.stringify(openState)})`);
      }
      await control.press('Enter');
    }

    if (process.env.HV_QA_SCREENSHOT_DIR) {
      mkdirSync(process.env.HV_QA_SCREENSHOT_DIR, { recursive: true });
      await page.screenshot({
        path: join(process.env.HV_QA_SCREENSHOT_DIR, `energy-geometry-${size.name}.png`),
      });
    }

    await page.evaluate(() => window.scrollTo(0, Math.min(1200, document.documentElement.scrollHeight - window.innerHeight)));
    // Scrolling must preserve the same single context-bar offset at every width.
    const stickyContextBar = await contextBarBox(page);
    const stickyStrip = await page.locator('.energy-mode-strip').boundingBox();
    if (!stickyStrip || Math.abs(stickyStrip.y - stickyContextBar.height) > 1) {
      const sticky = { stripTop: stickyStrip?.y ?? -1, barHeight: stickyContextBar.height };
      fail(`Energie-Geometrie ${size.name}: Sicherheitsleiste ist beim Scrollen nicht sauber gestapelt (${JSON.stringify(sticky)})`);
    }
    await closeContext(context);
    process.stdout.write(`  ✓ Energie-Geometrie · ${size.name} · ${result.oneColumn ? 'gestapelt' : 'zweispaltig'} · ${Math.round(result.stripRect.height)}px Sicherheit\n`);
  }
}

async function assertEnergyDataControl(viewport) {
  const ownerContext = await newContext(viewport.size);
  const page = await localLogin(ownerContext, 'owner@example.com');
  const response = await page.goto(`${baseURL}/app/settings/energy-data`, { waitUntil: 'networkidle' });
  if (!response || response.status() !== 200 ||
      !(await page.getByRole('heading', { name: 'Energiedaten & Datenschutz' }).count())) {
    fail(`Energiedaten ${viewport.name}: Eigentümerseite nicht erreichbar`);
  }
  for (const text of ['30 Tage', '13 Monate', '3 Jahre', 'Smart-Meter-Originale', 'Ganzes Energieprofil löschen']) {
    if (!(await page.getByText(text, { exact: true }).count())) {
      fail(`Energiedaten ${viewport.name}: „${text}“ fehlt`);
    }
  }
  const deletionDetails = page.locator('details.energy-delete-action');
  if ((await deletionDetails.count()) !== 2 ||
      await deletionDetails.evaluateAll((nodes) => nodes.some((node) => node.open))) {
    fail(`Energiedaten ${viewport.name}: Löschwege sind nicht sicher eingeklappt`);
  }
  const undersized = await page.locator(
    '.energy-data-page button, .energy-data-page details.energy-delete-action > summary'
  ).evaluateAll((nodes) => nodes.filter((node) => {
    const rect = node.getBoundingClientRect();
    return rect.width > 0 && rect.height > 0 && rect.height < 44;
  }).map((node) => (node.textContent || '').trim()));
  if (undersized.length) {
    fail(`Energiedaten ${viewport.name}: Touch-Ziele unter 44px: ${undersized.join(', ')}`);
  }
  if (await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth + 1)) {
    fail(`Energiedaten ${viewport.name}: horizontaler Überlauf`);
  }
  if (process.env.HV_QA_SCREENSHOT_DIR) {
    mkdirSync(process.env.HV_QA_SCREENSHOT_DIR, { recursive: true });
    await page.screenshot({
      path: join(process.env.HV_QA_SCREENSHOT_DIR, `energy-data-${viewport.name.toLowerCase()}.png`),
      fullPage: true,
    });
  }
  const [download] = await Promise.all([
    page.waitForEvent('download'),
    page.getByRole('button', { name: 'Energiedaten exportieren' }).click(),
  ]);
  if (!/^hausv-energiedaten-demo-\d{8}\.zip$/.test(download.suggestedFilename())) {
    fail(`Energiedaten ${viewport.name}: unerwarteter Exportname ${download.suggestedFilename()}`);
  }
  await page.waitForFunction(() => { const b = [...document.querySelectorAll('.energy-data-export button')][0]; return b && !b.disabled && b.textContent.trim() === 'Energiedaten exportieren'; }, null, { timeout: 5000 });
  const exportButton = page.getByRole('button', { name: 'Energiedaten exportieren' });
  if (!(await exportButton.isEnabled())) {
    fail(`Energiedaten ${viewport.name}: Exportknopf bleibt nach dem Download gesperrt`);
  }
  await closeContext(ownerContext);

  const deniedContext = await newContext(viewport.size);
  const deniedPage = await localLogin(deniedContext, 'resident@example.com');
  const denied = await deniedPage.goto(`${baseURL}/app/settings/energy-data`, { waitUntil: 'networkidle' });
  if (!denied || denied.status() !== 403) {
    fail(`Energiedaten ${viewport.name}: Bewohnerzugriff ist nicht mit 403 geschützt`);
  }
  await closeContext(deniedContext);
  process.stdout.write(`  ✓ Energiedaten & Datenschutz · ${viewport.name}\n`);
}

const viewports = [
  { name: 'Desktop', size: { width: 1440, height: 900 } },
  { name: 'Mobil', size: { width: 390, height: 844 } },
];
const activePersonas = ciCore
  ? personas.filter((persona) => persona.name === 'Bewohner' || persona.name === 'Admin')
  : personas;
const activeRoutes = ciCore
  ? routes.filter((route) => route.path === '/app' || route.path === '/app/anliegen')
  : routes;

try {
  if (process.env.HV_QA_FAILURE_PROBE === 'true') {
    const failureContext = await newContext({ width: 390, height: 844 });
    await localLogin(failureContext, 'resident@example.com');
    fail('Erwarteter QA-Artefakt-Testfehler');
  }

  if (energyOnly) {
    await assertHomeOnboarding();
    await ensureFocusedEnergyUnit();
    for (const viewport of viewports) {
      await assertEnergySafetyAndFlow(viewport);
      if (!fastQA) await assertEnergyDataControl(viewport);
    }
    if (!fastQA) {
      await assertEnergyGeometryMatrix();
    }
  } else {
    if (!ciCore) {
      for (const viewport of viewports) {
        await assertPublicLanding(viewport);
      }
    }

    if (process.env.HV_QA_LANDING_ONLY !== 'true') {
      await assertSidebarNavReachable();
      await assertSidebarAccountFits();
      await assertPortalSwitcherAtomic();
      await assertSharedAppShellNavigation();
      await assertHomeOnboarding();
      await assertBoundedAdminDialogs();
      await assertResponsiveAdminWidths();
      if (!ciCore) {
        await assertPilotHome({
          slug: 'haus-a',
          email: 'house-a-owner@example.com',
          householdName: 'Haus A',
          expectedAssets: ['pv', 'ev', 'hot-water', 'heat-pump'],
          absentAssets: ['battery'],
          expectedMeasured: ['Hausanschluss', 'PV-Anlage'],
          expectedCaptured: ['E-Auto', 'Warmwasser', 'Wärmepumpe'],
        });
        await assertPilotHome({
          slug: 'haus-b',
          email: 'house-b-owner@example.com',
          householdName: 'Haus B',
          expectedAssets: ['pv', 'battery', 'ev'],
          expectedMeasured: ['Hausanschluss', 'PV-Anlage', 'Batteriespeicher'],
          expectedCaptured: ['E-Auto'],
          inviteHelper: true,
        });
      }
      await assertResidentIssueProgressiveEnhancement();
      await createIssue('resident@example.com', 'QA Bewohneranliegen', { verifyResidentAttachmentTarget: true });
      if (!ciCore) {
        await createIssue('owner@example.com', 'QA Eigentümeranliegen');
      }
      await seedManagedContent();
      await assertPortalChromeKit();
      await assertResidentContentResponsiveMatrix(ciCore ? [
        { width: 390, height: 844 },
        { width: 768, height: 1024 },
      ] : undefined);
      await assertResidentContentClickFlows();
      await assertResidentBallotFlow();

      for (const viewport of viewports) {
        if (!ciCore) {
          await assertEnergySafetyAndFlow(viewport);
          await assertEnergyDataControl(viewport);
        }
        for (const persona of activePersonas) {
          const context = await newContext(viewport.size);
          const page = await localLogin(context, persona.email);
          for (const route of activeRoutes) {
            await assertPage(page, persona, route, viewport.name);
          }
          await assertRoleActions(page, persona);
          await closeContext(context);
          process.stdout.write(`  ✓ ${persona.name} · ${viewport.name}\n`);
        }
      }
      if (!fastQA) await assertEnergyGeometryMatrix();
      await assertLogoutBackNavigation();
    }
  }
} catch (error) {
  await captureFailureArtifacts(error);
  console.error(error instanceof Error ? error.stack || error.message : String(error));
  process.exitCode = 1;
} finally {
  await browser.close();
}
