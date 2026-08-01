#!/usr/bin/env node
// Role-aware, stateful QA for the portal's most common paths.

import { existsSync, mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { chromium } from 'playwright';

const baseURL = process.argv[2];
if (!baseURL) {
  console.error('usage: qa-main-flows.mjs <baseURL>');
  process.exit(1);
}
const artifactDir = process.env.HV_QA_ARTIFACT_DIR?.trim();
const ciCore = process.env.HV_QA_CI_CORE === 'true';
const activeContexts = new Set();
const browserEvents = [];
const loginStorageStates = new Map();
let mapTileResponseVerified = false;
if (artifactDir) mkdirSync(artifactDir, { recursive: true });

const personas = [
  { name: 'Bewohner', email: 'resident@example.com', manages: false },
  { name: 'Eigentümer', email: 'owner@example.com', manages: false },
  { name: 'Verwalter', email: 'verwalter@example.com', manages: true },
  { name: 'Admin', email: 'admin@example.com', manages: true },
];

const routes = [
  { path: '/app', heading: /Hallo /, content: 'Was ist als Nächstes zu tun?' },
  { path: '/app/announcements', heading: 'Aushang', content: 'QA Hausinformation' },
  { path: '/app/events', heading: 'Termine', content: 'QA Hausbegehung' },
  { path: '/app/kontakte', heading: 'Kontakte', content: 'QA Energiehilfe' },
  { path: '/app/dokumente', heading: 'Dokumente', content: 'QA Hausordnung' },
  { path: '/app/anliegen', heading: 'Anliegen', content: 'Anliegen' },
  { path: '/app/energie', heading: 'QA Zuhause', content: 'Nur beobachten' },
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
const launchOptions = {
  args: ['--host-resolver-rules=MAP hausv.test 127.0.0.1, MAP *.hausv.test 127.0.0.1', '--no-proxy-server'],
};
if (executablePath) {
  launchOptions.executablePath = executablePath;
}
const browser = await chromium.launch(launchOptions);

function fail(message) {
  throw new Error(message);
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
      aria: root.getAttribute('aria-label') || '',
      expected,
    };
  }, { displayName, unitLabel });
  if (result.primary !== displayName ||
      result.secondary !== unitLabel ||
      !result.inOrder ||
      result.secondaryFont >= result.primaryFont ||
      !result.visible ||
      !result.below ||
      !result.aria.includes(displayName) ||
      !result.aria.includes(unitLabel)) {
    fail(`${label}: Anzeigename und offizielle Einheit sind nicht sauber hierarchisiert (${JSON.stringify(result)})`);
  }
}

function tenantOrigin(hostname) {
  const url = new URL(baseURL);
  url.hostname = hostname;
  return url.origin;
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
  const emailDetails = page.locator('details:has(form[action="/auth/request"])');
  if (await emailDetails.count()) {
    await emailDetails.evaluate((element) => {
      element.open = true;
    });
  }
  await page.locator('input[name="email"]').fill(email);
  await page.locator('form[action="/auth/request"] button[type="submit"]').click();
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
      const nav = document.querySelector('.side-nav');
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
    if (result.missing) fail(`Seitenleiste bei ${height}px: .side-nav fehlt`);
    if (result.hidden.length) {
      fail(`Seitenleiste bei 1440x${height}: nicht erreichbar — ${result.hidden.join(', ')}`);
    }
    await closeContext(context);
  }
  process.stdout.write('  ✓ Seitenleiste · alle Einträge erreichbar · 720–1080px\n');
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
  if (!(await page.getByRole('heading', { name: 'Ein Portal für alles was Zuhause anfällt.' }).count())) {
    fail(`Öffentliche Startseite ${viewport.name}: Hauptaussage fehlt`);
  }
  const features = await page.locator('.feature').count();
  if (features !== 5) fail(`Öffentliche Startseite ${viewport.name}: ${features} statt 5 Kernaufgaben`);
  for (const text of [
    'Heute im privaten Pilot',
    'Nächste Ausbaustufe',
    'Bis 25 Einheiten im Pilot kostenlos',
    '1 € je Einheit und Monat',
  ]) {
    if (!(await page.getByText(text, { exact: true }).count())) {
      fail(`Öffentliche Startseite ${viewport.name}: „${text}“ fehlt`);
    }
  }
  if (process.env.HV_QA_SCREENSHOT_DIR) {
    mkdirSync(process.env.HV_QA_SCREENSHOT_DIR, { recursive: true });
    await page.screenshot({
      path: join(process.env.HV_QA_SCREENSHOT_DIR, `landing-${viewport.name.toLowerCase()}.png`),
      fullPage: true,
    });
  }
  const productDetails = page.locator('details.landing-more');
  if (await productDetails.evaluate((element) => element.open)) {
    fail(`Öffentliche Startseite ${viewport.name}: Produktdetails sind ungefragt offen`);
  }
  await productDetails.locator('summary').click();
  if (!(await page.getByRole('heading', { name: 'Heute nutzbar' }).count())) {
    fail(`Öffentliche Startseite ${viewport.name}: Produktdetails lassen sich nicht öffnen`);
  }
  const legalDetails = page.locator('details.legal-details');
  await legalDetails.locator('summary').click();
  if (!(await page.getByText('Datenschutz', { exact: true }).count())) {
    fail(`Öffentliche Startseite ${viewport.name}: Datenschutz ist nicht erreichbar`);
  }
  const metrics = await page.evaluate(() => ({
    overflow: document.documentElement.scrollWidth > window.innerWidth + 1,
    height: document.documentElement.scrollHeight,
  }));
  if (metrics.overflow) fail(`Öffentliche Startseite ${viewport.name}: horizontaler Überlauf`);
  const maxHeight = viewport.name === 'Mobil' ? 7_200 : 5_200;
  if (metrics.height > maxHeight) {
    fail(`Öffentliche Startseite ${viewport.name}: mit ${metrics.height}px unnötig lang (maximal ${maxHeight}px)`);
  }
  await closeContext(context);
  process.stdout.write(`  ✓ Öffentliche Startseite · ${viewport.name} · ${metrics.height}px\n`);
}

async function createIssue(email, title) {
  const context = await newContext({ width: 1440, height: 900 });
  const page = await localLogin(context, email);
  await page.goto(`${baseURL}/app/anliegen`, { waitUntil: 'networkidle' });
  // Auf einer leeren Anliegen-Seite ist das Formular eine feste Karte, damit die
  // Hauptaktion nicht hinter einem Aufklapper liegt; sobald Anliegen bestehen,
  // ist es ein details-Element. Nur letzteres muss geöffnet werden — element.open
  // ist auf einer section undefined und liefe sonst in einen Klick ins Leere.
  const panel = page.locator('#issue-new');
  if (await panel.evaluate((element) => element.tagName === 'DETAILS' && !element.open)) {
    await panel.locator('summary').click();
  }
  const form = page.locator('form[data-issue-wizard]');
  await form.locator('textarea[name="body"]').fill(`${title}. Bitte im Haus prüfen.`);
  await form.locator('[data-issue-step="1"] [data-issue-next]').click();
  await form.locator('input[name="location_detail"]').fill('Keller, neben dem Fahrradraum');
  await form.locator('[data-issue-step="2"] [data-issue-next]').click();
  await form.locator('input[name="title"]').fill(title);
  await form.locator('button[type="submit"]').click();
  await page.waitForURL(/\/app\/anliegen/);
  if (!(await page.getByText(title, { exact: true }).count())) fail(`Anliegen „${title}“ wurde nicht sichtbar gespeichert`);
  await closeContext(context);
}

function futureLocalInput(daysAhead, hour) {
  const date = new Date();
  date.setDate(date.getDate() + daysAhead);
  date.setHours(hour, 0, 0, 0);
  const offset = date.getTimezoneOffset() * 60_000;
  return new Date(date.getTime() - offset).toISOString().slice(0, 16);
}

async function seedManagedContent() {
  const context = await newContext({ width: 1440, height: 900 });
  const page = await localLogin(context, 'admin@example.com');

  await page.goto(`${baseURL}/app/announcements`, { waitUntil: 'networkidle' });
  // Die Hauptaktion steht in der Werkzeugleiste UND im Leerzustand, damit sie
  // auf einer leeren Seite erreichbar ist. Beide öffnen denselben Dialog.
  await page.getByRole('button', { name: 'Aushang erstellen' }).first().click();
  const announcement = page.locator('#announcement-create form');
  await announcement.locator('input[name="title"]').fill('QA Hausinformation');
  await announcement.locator('textarea[name="body"]').fill('Der gemeinsame Playwright-Lauf prüft diesen Aushang.');
  await announcement.locator('button[type="submit"]').click();
  await page.waitForURL(/\/app\/announcements/);

  await page.goto(`${baseURL}/app/events`, { waitUntil: 'networkidle' });
  await page.getByRole('button', { name: 'Termin erstellen' }).first().click();
  const event = page.locator('#event-create form');
  await event.locator('input[name="title"]').fill('QA Hausbegehung');
  await event.locator('input[name="starts_at"]').fill(futureLocalInput(14, 18));
  await event.locator('input[name="location"]').fill('Innenhof');
  await event.locator('button[type="submit"]').click();
  await page.waitForURL(/\/app\/events/);

  await page.goto(`${baseURL}/app/kontakte`, { waitUntil: 'networkidle' });
  const contactPanel = page.locator('#contact-add');
  if (!(await contactPanel.evaluate((element) => element.open))) {
    await contactPanel.locator('summary').click();
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
  await contact.locator('input[name="service_region"]').fill('Graz und Umgebung');
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

  await page.goto(`${baseURL}/app/settings/building#units`, { waitUntil: 'networkidle' });
  if (!(await page.getByText('Top 11', { exact: true }).count())) {
    const unitPanel = page.locator('#unit-add');
    if (!(await unitPanel.evaluate((element) => element.open))) {
      await unitPanel.locator('summary').click();
    }
    const unitForm = unitPanel.locator('form');
    await unitForm.locator('input[name="label"]').fill('Top 11');
    await unitForm.locator('input[name="owner_emails"]').fill('owner@example.com');
    await unitForm.locator('input[name="renter_emails"]').fill('resident@example.com');
    await unitForm.getByRole('button', { name: 'Einheit anlegen' }).click();
    await page.waitForURL(/\/app\/settings\/building/);
  }
  await page.goto(`${baseURL}/app/settings/home?from=building`, { waitUntil: 'networkidle' });
  const officialUnit = page.locator('select[name="unit_id"]');
  if (await officialUnit.count()) {
    await officialUnit.selectOption('top-11');
    await page.getByRole('button', { name: 'Änderungen speichern' }).click();
    await page.waitForURL(/\/app\/settings\/building\?home=saved/);
  }
  await page.goto(`${baseURL}/app/settings/building#units`, { waitUntil: 'networkidle' });
  await assertHomeIdentityPair(page, 'building-context', 'QA Zuhause', 'Top 11', 'Gebäude-Einstellungen');
  await assertHomeIdentityPair(page, 'building-unit', 'QA Zuhause', 'Top 11', 'Verknüpfte Einheit');
  if (!(await page.locator('[data-home-identity="building-context"]').getByText('QA Zuhause', { exact: true }).count()) ||
      !(await page.getByText('Offizielle Bezeichnung', { exact: true }).count())) {
    fail('Gebäude-Einstellungen: „Mein Zuhause“ und offizielle Einheit werden nicht klar getrennt');
  }
  if (process.env.HV_QA_SCREENSHOT_DIR) {
    mkdirSync(process.env.HV_QA_SCREENSHOT_DIR, { recursive: true });
    await page.locator('#units').screenshot({
      path: join(process.env.HV_QA_SCREENSHOT_DIR, 'home-identity-building-desktop.png'),
    });
  }

  await closeContext(context);
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
  await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight));
  const box = await page.locator('.energy-mode-strip').boundingBox();
  if (!box || box.y > 65) fail(`Onboarding Mobil: Beobachtungsmodus nicht permanent sichtbar (${box?.y ?? 'fehlt'})`);
  await closeContext(context);
  process.stdout.write('  ✓ Energie-Onboarding · Tastatur · Fortsetzen · Mobil\n');
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
  const origin = tenantOrigin(`${slug}.hausv.test`);
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
  if (!(await page.getByText('Drei Jahre voller Produktumfang kostenlos', { exact: true }).count()) ||
      !(await page.getByText('1 € pro Monat', { exact: false }).count())) {
    fail(`${householdName}: transparentes Drei-Jahres-/12-Euro-Modell fehlt`);
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
    await invitation.locator('input[name="email"]').fill('inlaws-helper@example.com');
    await invitation.locator('input[value="configure"]').check();
    await invitation.getByRole('button', { name: 'Hausbezogen einladen' }).click();
    await page.waitForLoadState('networkidle');
    const helperContext = await newContext({ width: 390, height: 844 });
    const helper = await localLogin(helperContext, 'inlaws-helper@example.com', origin);
    const response = await helper.goto(`${origin}/app/energie`, { waitUntil: 'networkidle' });
    if (!response || response.status() !== 200 ||
        !(await helper.getByText('Nur beobachten', { exact: true }).count())) {
      fail(`${householdName}: technische Hilfe erreicht das Haus nicht`);
    }
    if (await helper.getByText('Testlauf bewusst starten', { exact: true }).count()) {
      fail(`${householdName}: technische Hilfe sieht den Eigentümer-Schalter`);
    }
    await closeContext(helperContext);
  }

  await closeContext(context);
  process.stdout.write(`  ✓ Pilot ${householdName} · getrennt · read-only · Messlücken\n`);
}

async function assertPage(page, persona, route, viewportName) {
  const response = await page.goto(`${baseURL}${route.path}`, { waitUntil: 'networkidle' });
  if (!response || response.status() !== 200) {
    fail(`${persona.name} ${viewportName} ${route.path}: Status ${response?.status() ?? 0}`);
  }
  const heading = page.getByRole('heading', { name: route.heading }).first();
  if (!(await heading.count())) fail(`${persona.name} ${viewportName} ${route.path}: Überschrift fehlt`);
  if (!(await page.getByText(route.content, { exact: false }).count())) {
    fail(`${persona.name} ${viewportName} ${route.path}: Inhalt „${route.content}“ fehlt`);
  }
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth + 1);
  if (overflow) fail(`${persona.name} ${viewportName} ${route.path}: horizontaler Überlauf`);

  if (route.path === '/app') {
    const focusCount = await page.locator('.home-primary-task, .home-calm').count();
    if (focusCount !== 1) fail(`${persona.name} ${viewportName}: Hausüberblick hat ${focusCount} Hauptzustände`);
    const map = await page.locator('.side-map').boundingBox();
    const pin = await page.locator('.side-map-pin-mark svg').boundingBox();
    if (!map || !pin) fail(`${persona.name} ${viewportName}: Karte oder Haus-Pin fehlt`);
    const tile = page.locator('img.side-map-tile').first();
    await tile.waitFor({ state: 'visible' });
    const decodedTile = await tile.evaluate((image) => ({
      complete: image.complete,
      width: image.naturalWidth,
      height: image.naturalHeight,
    }));
    if (!decodedTile.complete || decodedTile.width < 1 || decodedTile.height < 1) {
      fail(`${persona.name} ${viewportName}: Kartenkachel ist keine dekodierbare Grafik`);
    }
    if (!mapTileResponseVerified) {
      const tilePath = await tile.getAttribute('src');
      if (!tilePath) fail(`${persona.name} ${viewportName}: Kartenkachel hat keine Quelle`);
      const tileResponse = await page.context().request.get(new URL(tilePath, page.url()).href);
      const tileBytes = await tileResponse.body();
      const pngSignature = Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]);
      if (!tileResponse.ok() ||
          !tileResponse.headers()['content-type']?.startsWith('image/png') ||
          tileBytes.length < pngSignature.length ||
          !tileBytes.subarray(0, pngSignature.length).equals(pngSignature)) {
        fail(`${persona.name} ${viewportName}: Kartenendpoint liefert keine gültige PNG-Kachel`);
      }
      mapTileResponseVerified = true;
    }
    if (viewportName === 'Desktop' && (map.width < 250 || map.height < 190 || pin.width < 22)) {
      fail(`${persona.name} Desktop: Ortskopf ist mit ${map.width}×${map.height}px / Pin ${pin.width}px zu klein`);
    }
    if (viewportName === 'Mobil' && (map.width > 64 || map.height > 54 || pin.width > 24)) {
      fail(`${persona.name} Mobil: Ortskopf verdrängt mit ${map.width}×${map.height}px / Pin ${pin.width}px die Navigation`);
    }
    if (!(await page.locator('.side-address[href*="openstreetmap.org"]').count()) ||
        !(await page.locator('.side-portal span', { hasText: 'hausv.org' }).count())) {
      fail(`${persona.name} ${viewportName}: Adresse oder Portal-Signatur fehlt`);
    }
    if (persona.name === 'Eigentümer' && process.env.HV_QA_SCREENSHOT_DIR) {
      mkdirSync(process.env.HV_QA_SCREENSHOT_DIR, { recursive: true });
      await page.evaluate(() => window.scrollTo(0, 0));
      await page.screenshot({
        path: join(process.env.HV_QA_SCREENSHOT_DIR, `sidebar-${viewportName.toLowerCase()}.png`),
        fullPage: false,
      });
    }
  }
}

async function assertRoleActions(page, persona) {
  if (persona.manages) {
    const expected = [
      ['/app/announcements', 'Aushang erstellen'],
      ['/app/events', 'Termin erstellen'],
      ['/app/kontakte', 'Kontakt hinzufügen'],
      ['/app/dokumente', 'Dokument hochladen'],
    ];
    for (const [path, label] of expected) {
      await page.goto(`${baseURL}${path}`, { waitUntil: 'networkidle' });
      if (!(await page.getByText(label, { exact: true }).count())) fail(`${persona.name}: Aktion „${label}“ fehlt`);
    }
    const board = await page.goto(`${baseURL}/app/anliegen/board`, { waitUntil: 'networkidle' });
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

  await Promise.all([
    page.waitForURL((url) => url.pathname === '/'),
    page.getByRole('button', { name: 'Abmelden' }).click(),
  ]);
  loginStorageStates.delete(`${baseURL}|resident@example.com`);
  if (!(await page.getByRole('heading', { name: 'Willkommen zurück' }).isVisible())) {
    fail('Abmelden/Zurück: Loginseite nach Abmeldung fehlt');
  }

  await page.goBack({ waitUntil: 'domcontentloaded' });
  await page.waitForURL((url) => url.pathname === '/', { timeout: 10_000 });
  await page.waitForLoadState('networkidle');
  const authenticatedBody = await page.locator('body[data-authenticated-app]').count();
  if (authenticatedBody || await protectedHeading.isVisible().catch(() => false)) {
    fail('Abmelden/Zurück: geschützter Inhalt wurde aus dem Browsercache wieder sichtbar');
  }
  if (!(await page.getByRole('heading', { name: 'Willkommen zurück' }).isVisible())) {
    fail('Abmelden/Zurück: unauthentifizierter Zustand fehlt nach Browser-Zurück');
  }

  await closeContext(context);
  process.stdout.write('  ✓ Abmelden → Browser-Zurück bleibt unauthentifiziert\n');
}

async function assertEnergySafetyAndFlow(viewport) {
  const residentContext = await newContext(viewport.size);
  const resident = await localLogin(residentContext, 'resident@example.com');
  await resident.goto(`${baseURL}/app/energie`, { waitUntil: 'networkidle' });
  if (await resident.getByText('Testlauf bewusst starten', { exact: true }).count()) {
    fail(`Bewohner ${viewport.name}: Steuerungsfreigabe sichtbar`);
  }
  await closeContext(residentContext);

  const ownerContext = await newContext(viewport.size);
  const page = await localLogin(ownerContext, 'owner@example.com');
  await page.goto(`${baseURL}/app/energie`, { waitUntil: 'networkidle' });
  await assertHomeIdentityPair(page, 'energy-heading', 'QA Zuhause', 'Top 11', `Energie ${viewport.name}`);
  if (viewport.name === 'Mobil') {
    await page.locator('.mobile-menu-toggle').click();
    await assertHomeIdentityPair(page, 'nav', 'QA Zuhause', 'Top 11', `Navigation ${viewport.name}`);
    await assertHomeIdentityPair(page, 'mobile-menu', 'QA Zuhause', 'Top 11', `Mobiler Menükopf ${viewport.name}`);
    if (process.env.HV_QA_SCREENSHOT_DIR) {
      mkdirSync(process.env.HV_QA_SCREENSHOT_DIR, { recursive: true });
      await page.screenshot({
        path: join(process.env.HV_QA_SCREENSHOT_DIR, 'home-name-nav-mobile-open.png'),
      });
    }
    await page.locator('.mobile-menu-toggle').click();
  } else {
    await assertHomeIdentityPair(page, 'nav', 'QA Zuhause', 'Top 11', `Navigation ${viewport.name}`);
  }
  if (!(await page.locator('.energy-heading-context').getByText('Wohnung ·', { exact: false }).count()) ||
      !(await page.getByRole('link', { name: 'Zuhause bearbeiten' }).count())) {
    fail(`Energie ${viewport.name}: Name, offizielle Wohnung oder sichtbarer Bearbeitungsweg fehlt`);
  }
  await page.getByRole('link', { name: 'Zuhause bearbeiten' }).click();
  await page.waitForURL(/\/app\/settings\/home/);
  await assertHomeIdentityPair(page, 'editor-heading', 'QA Zuhause', 'Top 11', `Zuhause-Einstellungen ${viewport.name}`);
  await assertHomeIdentityPair(page, 'editor-summary', 'QA Zuhause', 'Top 11', `Zuhause-Zusammenfassung ${viewport.name}`);
  if (!(await page.getByRole('heading', { name: 'QA Zuhause', exact: true }).count()) ||
      (await page.locator('input[name="household_name"]').inputValue()) !== 'QA Zuhause' ||
      !(await page.getByText('Top 11', { exact: true }).count()) ||
      (await page.locator('[name="unit_id"]').inputValue()) !== 'top-11' ||
      !(await page.getByText('Diesem Hausprofil zugeordnet.', { exact: true }).count()) ||
      !(await page.locator('[data-home-type-explanation]').count())) {
    fail(`Energie ${viewport.name}: Hausname ist nicht verständlich mit „Top 11“ verknüpft`);
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
    await assertHomeIdentityPair(page, 'energy-heading', 'Sonnendeck QA', 'Top 11', 'Umbenennung Energie Desktop');
    await assertHomeIdentityPair(page, 'nav', 'Sonnendeck QA', 'Top 11', 'Umbenennung Navigation Desktop');
    await page.getByRole('link', { name: 'Zuhause bearbeiten' }).click();
    await page.locator('input[name="household_name"]').fill('QA Zuhause');
    await page.getByRole('button', { name: 'Änderungen speichern' }).click();
    await page.waitForURL(/\/app\/energie/);
  } else {
    const longDisplayName = 'DachterrassenwohnungMitAußergewöhnlichLangemAnzeigenamenFürDieMobileDarstellung';
    await page.locator('input[name="household_name"]').fill(longDisplayName);
    await page.getByRole('button', { name: 'Änderungen speichern' }).click();
    await page.waitForURL(/\/app\/energie/);
    await assertHomeIdentityPair(page, 'energy-heading', longDisplayName, 'Top 11', 'Langer Anzeigename Mobil');
    await page.locator('.mobile-menu-toggle').click();
    await assertHomeIdentityPair(page, 'nav', longDisplayName, 'Top 11', 'Langer Navigationsname Mobil');
    if (await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth + 1)) {
      fail('Langer Anzeigename Mobil: Darstellung läuft horizontal über');
    }
    await page.locator('.mobile-menu-toggle').click();
    await page.getByRole('link', { name: 'Zuhause bearbeiten' }).click();
    await page.locator('input[name="household_name"]').fill('QA Zuhause');
    await page.getByRole('button', { name: 'Änderungen speichern' }).click();
    await page.waitForURL(/\/app\/energie/);
  }
  await page.goto(`${baseURL}/app/settings`, { waitUntil: 'networkidle' });
  await assertHomeIdentityPair(page, 'settings', 'QA Zuhause', 'Top 11', `Einstellungen ${viewport.name}`);
  if (process.env.HV_QA_SCREENSHOT_DIR) {
    await page.screenshot({
      path: join(process.env.HV_QA_SCREENSHOT_DIR, `home-name-settings-${viewport.name.toLowerCase()}.png`),
      fullPage: true,
    });
  }
  await page.goto(`${baseURL}/app/energie`, { waitUntil: 'networkidle' });
  const live = page.locator('.energy-live');
  const readingCount = Number(await live.getAttribute('data-energy-reading-count'));
  if (readingCount < 8) {
    fail(`Energie ${viewport.name}: Mehr-Messwerte-QA hat nur ${readingCount || 0} Live-Werte`);
  }
  if (!(await live.getByText('Hausverbrauch', { exact: true }).count()) ||
      !(await live.locator('[data-energy-metric="pv-power"]').count()) ||
      !(await live.locator('[data-energy-metric="grid-import-power"]').count()) ||
      !(await live.locator('[data-energy-metric="battery-power"]').count())) {
    fail(`Energie ${viewport.name}: verständliche Energiefluss-Zusammenfassung fehlt`);
  }
  if (!(await live.locator('details.energy-live-more').count())) {
    fail(`Energie ${viewport.name}: weitere Messwerte sind nicht progressiv erreichbar`);
  }
  const batteryGauge = live.locator('.energy-battery-gauge');
  const batteryVisual = live.locator('.energy-battery-visual');
  if (!(await batteryGauge.count()) ||
      !((await batteryVisual.getAttribute('aria-label')) || '').includes('78 %') ||
      !(await live.getByText('lädt · 600 W', { exact: true }).count())) {
    fail(`Energie ${viewport.name}: Batterie-Füllstand und aktuelle Speicherleistung fehlen`);
  }
  const batteryFill = batteryGauge.locator('i');
  const gaugeBox = await batteryGauge.boundingBox();
  const fillBox = await batteryFill.boundingBox();
  if (!gaugeBox || !fillBox || fillBox.width / gaugeBox.width < 0.62 || fillBox.width / gaugeBox.width > 0.75 ||
      (await batteryVisual.getAttribute('data-energy-direction')) !== 'charging' ||
      (await batteryVisual.locator('.energy-battery-chevron').count()) !== 3 ||
      (await live.locator('.energy-metric-icon').count()) < 4) {
    fail(`Energie ${viewport.name}: proportionaler Batteriestand, Laderichtung oder Energie-Icons fehlen`);
  }
  const flowGrid = live.locator('.energy-flow-grid');
  const lastFlow = flowGrid.locator('.energy-flow-item').last();
  const flowGridBox = await flowGrid.boundingBox();
  const lastFlowBox = await lastFlow.boundingBox();
  if (!flowGridBox || !lastFlowBox ||
      (viewport.name === 'Desktop' && lastFlowBox.width < flowGridBox.width - 2)) {
    fail(`Energie ${viewport.name}: ungerader Energiefluss lässt eine unbeabsichtigte Leerzelle zurück`);
  }
  if (!(await live.getByRole('link', { name: 'Messwerte zuordnen' }).count())) {
    fail(`Energie ${viewport.name}: dauerhafter Einstieg ins Messwert-Setup fehlt`);
  }
  if (await page.getByText('Home Current Consumption', { exact: true }).isVisible().catch(() => false)) {
    fail(`Energie ${viewport.name}: technische Home-Assistant-Rohbezeichnung konkurriert mit der Übersicht`);
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
  if (todayHitCount < 2 || todayHitCount >= 97) {
    fail(`Energie ${viewport.name}: zukünftige Heute-Werte sind nicht leer (${todayHitCount} von 97 belegt)`);
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
  if (!(await visibleChart.locator('path.energy-chart-area.load').count()) ||
      !(await visibleChart.locator('line.energy-chart-threshold').count()) ||
      !(await chart.getByText('Planungsgrenze 10 kW', { exact: false }).count())) {
    fail(`Energie ${viewport.name}: Verbrauchsfläche oder konfigurierbare 10-kW-Planungsgrenze fehlt`);
  }
  const scaleLabels = await visibleChart.locator('text.energy-chart-axis-label').allTextContents();
  if (!scaleLabels.includes('15 kW') || !scaleLabels.includes('-15 kW')) {
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
  if (!(await page.getByText('Messwerte aktuell', { exact: true }).count())) {
    fail(`Energie ${viewport.name}: Live-Aktualität verwendet nicht den aktuellen Home-Assistant-Zeitpunkt`);
  }
  const chartBox = await visibleChart.boundingBox();
  if (!chartBox || chartBox.width > viewport.size.width + 1) {
    fail(`Energie ${viewport.name}: 24-Stunden-Diagramm läuft aus dem sichtbaren Bereich`);
  }
  const strip = page.locator('.energy-mode-strip');
  if ((await strip.locator('strong').first().innerText()).trim() !== 'Nur beobachten') {
    fail(`Energie ${viewport.name}: startet nicht in Nur beobachten`);
  }
  await page.getByText('Testlauf bewusst starten', { exact: true }).click();
  const modeForm = page.locator('.energy-mode-popover');
  await modeForm.locator('input[type="checkbox"]').check();
  await modeForm.locator('input[name="confirmation_text"]').fill('TESTLAUF');
  await modeForm.getByRole('button', { name: 'Testlauf starten' }).click();
  await page.waitForLoadState('networkidle');
  if ((await strip.locator('strong').first().innerText()).trim() !== 'Testlauf aktiv · keine Gerätewirkung') {
    fail(`Energie ${viewport.name}: Freigabe startet nicht im Testlauf`);
  }
  if (!(await page.getByText('Es schaltet kein Gerät.', { exact: false }).count())) {
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

    await page.getByRole('button', { name: 'Diesen Stand festhalten' }).click();
    await page.waitForLoadState('networkidle');
    if (!(await page.getByText('Festgehaltene Bewertungen', { exact: true }).count())) {
      fail('Energie Desktop: Tarifstand wurde nicht historisch sichtbar');
    }

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
    await invitation.getByRole('button', { name: 'Hausbezogen einladen' }).click();
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
    if (await helper.getByText('Testlauf bewusst starten', { exact: true }).count()) {
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

    const measureControl = page.locator('details.energy-measure-control');
    await measureControl.locator('summary').click();
    const measureBox = await measureControl.locator('.energy-measure-form').boundingBox();
    const roadmapBox = await page.locator('#fahrplan').boundingBox();
    if (!measureBox || !roadmapBox || measureBox.y + measureBox.height > roadmapBox.y + 1) {
      fail('Energie Desktop: geöffnetes Hausaufgaben-Formular überlagert den Fahrplan');
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
    const specialistPanel = page.locator('details.energy-collapsible').filter({ hasText: 'Fachhilfe, wenn sie wirklich nötig ist' });
    await specialistPanel.locator(':scope > summary').click();
    const measure = specialistPanel.locator('details.energy-measure-row').first();
    await measure.locator('summary').click();
    await measure.locator('select[name="contact_id"]').selectOption({ label: 'QA Energiehilfe · Graz und Umgebung' });
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
    if (!box || box.y > 65) fail(`Energie Mobil: Modus nicht permanent sichtbar (${box?.y ?? 'fehlt'})`);
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
  await live.getByRole('link', { name: 'Messwerte zuordnen' }).click();
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
  if (!/^hausv-energiedaten-jhw22-\d{8}\.zip$/.test(download.suggestedFilename())) {
    fail(`Energiedaten ${viewport.name}: unerwarteter Exportname ${download.suggestedFilename()}`);
  }
  await page.waitForTimeout(2700);
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

  if (!ciCore) {
    for (const viewport of viewports) {
      await assertPublicLanding(viewport);
    }
  }

  if (process.env.HV_QA_LANDING_ONLY !== 'true') {
    await assertSidebarNavReachable();
    await assertHomeOnboarding();
    if (!ciCore) {
      await assertPilotHome({
        slug: 'eltern',
        email: 'parents-owner@example.com',
        householdName: 'Haus Eltern',
        expectedAssets: ['pv', 'ev', 'hot-water', 'heat-pump'],
        absentAssets: ['battery'],
        expectedMeasured: ['Hausanschluss', 'PV-Anlage'],
        expectedCaptured: ['E-Auto', 'Warmwasser', 'Wärmepumpe'],
      });
      await assertPilotHome({
        slug: 'schwiegereltern',
        email: 'inlaws-owner@example.com',
        householdName: 'Haus Schwiegereltern',
        expectedAssets: ['pv', 'battery', 'ev'],
        expectedMeasured: ['Hausanschluss', 'PV-Anlage', 'Batteriespeicher'],
        expectedCaptured: ['E-Auto'],
        inviteHelper: true,
      });
    }
    await createIssue('resident@example.com', 'QA Bewohneranliegen');
    if (!ciCore) {
      await createIssue('owner@example.com', 'QA Eigentümeranliegen');
      await seedManagedContent();
    }

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
    await assertLogoutBackNavigation();
  }
} catch (error) {
  await captureFailureArtifacts(error);
  console.error(error instanceof Error ? error.stack || error.message : String(error));
  process.exitCode = 1;
} finally {
  await browser.close();
}
