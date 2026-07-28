#!/usr/bin/env node
// Role-aware, stateful QA for the portal's most common paths.

import { existsSync, mkdirSync } from 'node:fs';
import { join } from 'node:path';
import { chromium } from 'playwright';

const baseURL = process.argv[2];
if (!baseURL) {
  console.error('usage: qa-main-flows.mjs <baseURL>');
  process.exit(1);
}

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
  { path: '/app/kontakte', heading: 'Kontakte', content: 'QA Hausbetreuung' },
  { path: '/app/dokumente', heading: 'Dokumente', content: 'QA Hausordnung' },
  { path: '/app/anliegen', heading: 'Anliegen', content: 'Anliegen' },
  { path: '/app/energie', heading: 'QA Zuhause', content: 'Nur beobachten' },
];

const executableCandidates = [
  process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH,
  '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
  '/Applications/Chromium.app/Contents/MacOS/Chromium',
  '/usr/bin/chromium',
  '/usr/bin/chromium-browser',
  '/usr/bin/google-chrome',
].filter(Boolean);
const executablePath = executableCandidates.find(existsSync);
const launchOptions = {
  args: ['--host-resolver-rules=MAP hausv.test 127.0.0.1', '--no-proxy-server'],
};
if (executablePath) {
  launchOptions.executablePath = executablePath;
} else {
  launchOptions.channel = 'chrome';
}
const browser = await chromium.launch(launchOptions);

function fail(message) {
  throw new Error(message);
}

async function localLogin(context, email) {
  const page = await context.newPage();
  await page.goto(`${baseURL}/`, { waitUntil: 'networkidle' });
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
  await page.goto(new URL(href, baseURL).href, { waitUntil: 'networkidle' });
  if (!page.url().includes('/app')) fail(`Lokale Anmeldung für ${email} endete auf ${page.url()}`);
  return page;
}

async function newContext(viewport) {
  return browser.newContext({
    viewport,
    deviceScaleFactor: 1,
    locale: 'de-AT',
    timezoneId: 'Europe/Vienna',
  });
}

async function assertPublicLanding(viewport) {
  const context = await browser.newContext({
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
  if (!(await page.getByRole('heading', { name: 'Ein Portal für alle, die ein Haus gemeinsam verwalten.' }).count())) {
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
  await context.close();
  process.stdout.write(`  ✓ Öffentliche Startseite · ${viewport.name} · ${metrics.height}px\n`);
}

async function createIssue(email, title) {
  const context = await newContext({ width: 1440, height: 900 });
  const page = await localLogin(context, email);
  await page.goto(`${baseURL}/app/anliegen`, { waitUntil: 'networkidle' });
  const panel = page.locator('#issue-new');
  if (!(await panel.evaluate((element) => element.open))) {
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
  await context.close();
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
  await page.getByRole('button', { name: 'Aushang erstellen' }).click();
  const announcement = page.locator('#announcement-create form');
  await announcement.locator('input[name="title"]').fill('QA Hausinformation');
  await announcement.locator('textarea[name="body"]').fill('Der gemeinsame Playwright-Lauf prüft diesen Aushang.');
  await announcement.locator('button[type="submit"]').click();
  await page.waitForURL(/\/app\/announcements/);

  await page.goto(`${baseURL}/app/events`, { waitUntil: 'networkidle' });
  await page.getByRole('button', { name: 'Termin erstellen' }).click();
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
  await contact.locator('select[name="kind"]').selectOption({ index: 1 });
  await contact.locator('input[name="name"]').fill('QA Hausbetreuung');
  await contact.locator('input[name="phone"]').fill('+43 316 000000');
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

  await context.close();
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

async function assertEnergySafetyAndFlow(viewport) {
  const residentContext = await newContext(viewport.size);
  const resident = await localLogin(residentContext, 'resident@example.com');
  await resident.goto(`${baseURL}/app/energie`, { waitUntil: 'networkidle' });
  if (await resident.getByText('Steuerung bewusst freigeben', { exact: true }).count()) {
    fail(`Bewohner ${viewport.name}: Steuerungsfreigabe sichtbar`);
  }
  await residentContext.close();

  const ownerContext = await newContext(viewport.size);
  const page = await localLogin(ownerContext, 'owner@example.com');
  await page.goto(`${baseURL}/app/energie`, { waitUntil: 'networkidle' });
  const strip = page.locator('.energy-mode-strip');
  if ((await strip.locator('strong').first().innerText()).trim() !== 'Nur beobachten') {
    fail(`Energie ${viewport.name}: startet nicht in Nur beobachten`);
  }
  await page.getByText('Steuerung bewusst freigeben', { exact: true }).click();
  const modeForm = page.locator('.energy-mode-popover');
  await modeForm.locator('input[type="checkbox"]').check();
  await modeForm.locator('input[name="confirmation_text"]').fill('AKTIVIEREN');
  await modeForm.getByRole('button', { name: 'Aktive Steuerung freigeben' }).click();
  await page.waitForLoadState('networkidle');
  if ((await strip.locator('strong').first().innerText()).trim() !== 'Steuerung freigegeben · Testlauf') {
    fail(`Energie ${viewport.name}: Freigabe startet nicht im Testlauf`);
  }
  if (!(await page.getByText('schaltet aber noch kein Gerät', { exact: false }).count())) {
    fail(`Energie ${viewport.name}: Shadow-Mode-Erklärung fehlt`);
  }
  await page.getByRole('button', { name: /Sofort zurück/ }).click();
  await page.waitForLoadState('networkidle');
  if ((await strip.locator('strong').first().innerText()).trim() !== 'Nur beobachten') {
    fail(`Energie ${viewport.name}: Sofort-Rückkehr fehlgeschlagen`);
  }

  if (viewport.name === 'Desktop') {
    const csv = Buffer.from('timestamp;import_kwh\n2026-07-01T00:00:00+02:00;0,42\n2026-07-01T00:15:00+02:00;0,38\n');
    const file = { name: 'smart-meter.csv', mimeType: 'text/csv', buffer: csv };
    await page.locator('input[name="smart_meter_file"]').setInputFiles(file);
    await page.getByRole('button', { name: 'Als Referenz importieren' }).click();
    await page.waitForLoadState('networkidle');
    if (!(await page.getByText('Smart-Meter-Datei übernommen.', { exact: false }).count())) {
      fail('Energie Desktop: Smart-Meter-Import nicht bestätigt');
    }
    await page.locator('input[name="smart_meter_file"]').setInputFiles(file);
    await page.getByRole('button', { name: 'Als Referenz importieren' }).click();
    await page.waitForLoadState('networkidle');
    if (!(await page.getByText('bereits vorhanden', { exact: false }).count())) {
      fail('Energie Desktop: doppelter Import nicht erkannt');
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
  await ownerContext.close();
}

try {
  for (const viewport of [
    { name: 'Desktop', size: { width: 1440, height: 900 } },
    { name: 'Mobil', size: { width: 390, height: 844 } },
  ]) {
    await assertPublicLanding(viewport);
  }

  if (process.env.HV_QA_LANDING_ONLY !== 'true') {
    await createIssue('resident@example.com', 'QA Bewohneranliegen');
    await createIssue('owner@example.com', 'QA Eigentümeranliegen');
    await seedManagedContent();

    for (const viewport of [
      { name: 'Desktop', size: { width: 1440, height: 900 } },
      { name: 'Mobil', size: { width: 390, height: 844 } },
    ]) {
      await assertEnergySafetyAndFlow(viewport);
      for (const persona of personas) {
        const context = await newContext(viewport.size);
        const page = await localLogin(context, persona.email);
        for (const route of routes) {
          await assertPage(page, persona, route, viewport.name);
        }
        await assertRoleActions(page, persona);
        await context.close();
        process.stdout.write(`  ✓ ${persona.name} · ${viewport.name}\n`);
      }
    }
  }
} finally {
  await browser.close();
}
