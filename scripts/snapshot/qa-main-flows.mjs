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
  { path: '/app/kontakte', heading: 'Kontakte', content: 'QA Energiehilfe' },
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
  args: ['--host-resolver-rules=MAP hausv.test 127.0.0.1, MAP *.hausv.test 127.0.0.1', '--no-proxy-server'],
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

function tenantOrigin(hostname) {
  const url = new URL(baseURL);
  url.hostname = hostname;
  return url.origin;
}

async function localLogin(context, email, origin = baseURL) {
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
  await contact.locator('select[name="kind"]').selectOption({ label: 'Energie-Fachbetrieb' });
  await contact.locator('input[name="name"]').fill('QA Energiehilfe');
  await contact.locator('input[name="phone"]').fill('+43 316 000000');
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

  await context.close();
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
  await context.close();

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
  if (!(await page.getByText('Rechte und „Nur beobachten“ bleiben unverändert.', { exact: false }).count())) {
    fail('Onboarding: Auswahlwirkung grenzt Rechte und Sicherheitsmodus nicht ehrlich ab');
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
  await mobileContext.close();

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
  await context.close();

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
  await context.close();
  process.stdout.write('  ✓ Energie-Onboarding · Tastatur · Fortsetzen · Mobil\n');
}

async function importPilotReference(page) {
  const measurementPanel = page.locator('details.energy-collapsible').filter({ hasText: 'Messwerte & Referenz' });
  await measurementPanel.locator(':scope > summary').click();
  await page.locator('input[name="smart_meter_file"]').setInputFiles({
    name: 'smart-meter-pilot.csv',
    mimeType: 'text/csv',
    buffer: Buffer.from('timestamp;import_kwh\n2026-07-01T00:00:00+02:00;0,42\n2026-07-01T00:15:00+02:00;0,38\n'),
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
    if (await helper.getByText('Steuerung bewusst freigeben', { exact: true }).count()) {
      fail(`${householdName}: technische Hilfe sieht den Eigentümer-Schalter`);
    }
    await helperContext.close();
  }

  await context.close();
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
    const mark = await page.locator('.side-mark svg').boundingBox();
    if (!mark) fail(`${persona.name} ${viewportName}: Hauszeichen fehlt`);
    if (viewportName === 'Desktop' && mark.width < 64) {
      fail(`${persona.name} Desktop: Hauszeichen ist mit ${mark.width}px nicht deutlich größer`);
    }
    if (viewportName === 'Mobil' && mark.width > 42) {
      fail(`${persona.name} Mobil: Hauszeichen verdrängt mit ${mark.width}px die Navigation`);
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
  if (await page.getByText('Home Current Consumption', { exact: true }).isVisible().catch(() => false)) {
    fail(`Energie ${viewport.name}: technische Home-Assistant-Rohbezeichnung konkurriert mit der Übersicht`);
  }
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
    const measurementPanel = page.locator('details.energy-collapsible').filter({ hasText: 'Messwerte & Referenz' });
    await measurementPanel.locator(':scope > summary').click();
    const csv = Buffer.from('timestamp;import_kwh\n2026-07-01T00:00:00+02:00;0,42\n2026-07-01T00:15:00+02:00;0,38\n');
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
    if (await helper.getByText('Steuerung bewusst freigeben', { exact: true }).count()) {
      fail('Energie Desktop: technische Vertrauensperson sieht die Eigentümer-/Admin-Freigabe');
    }
    await helperContext.close();

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
    }
  }
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
    await assertHomeOnboarding();
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
