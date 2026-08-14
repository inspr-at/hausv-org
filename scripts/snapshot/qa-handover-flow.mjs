#!/usr/bin/env node
// Deterministic end-to-end QA for the complete handover lifecycle.
//
// The handover itself is created through the real manager UI. Confirmation
// links are deliberately not exposed by the product in local development, so
// this isolated harness replaces only the two token hashes in the fixture DB
// with known values before exercising the public review screens.

import { existsSync, mkdirSync, writeFileSync } from 'node:fs';
import { join, resolve } from 'node:path';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright';

const baseURL = process.argv[2];
if (!baseURL) {
  console.error('usage: qa-handover-flow.mjs <baseURL>');
  process.exit(1);
}

const artifactRoot = resolve(process.env.HV_QA_ARTIFACT_DIR?.trim() || '/tmp/hausv-handover-flow');
const screenshotDir = join(artifactRoot, 'screenshots');
mkdirSync(screenshotDir, { recursive: true });

const widths = [320, 390, 768, 1024, 1440];
const viewportFor = (width) => ({ width, height: width <= 390 ? 740 : width <= 768 ? 900 : 920 });
const browserEvents = [];
const activeContexts = new Set();
let adminStorageState;

const launchOptions = {
  headless: true,
  args: ['--host-resolver-rules=MAP hausv.test 127.0.0.1, MAP *.hausv.test 127.0.0.1', '--no-proxy-server'],
};
if (process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH && existsSync(process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH)) {
  launchOptions.executablePath = process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH;
}
const browser = await chromium.launch(launchOptions);

function fail(message) {
  throw new Error(message);
}

function watchPage(page) {
  page.on('console', (message) => {
    if (message.type() === 'error' || message.type() === 'warning') {
      browserEvents.push(`${message.type()} ${new URL(page.url()).pathname}: ${message.text()}`);
    }
  });
  page.on('pageerror', (error) => browserEvents.push(`pageerror ${new URL(page.url()).pathname}: ${error.message}`));
  page.on('requestfailed', (request) => {
    const path = new URL(request.url()).pathname;
    const error = request.failure()?.errorText || 'unknown';
    // Chromium reports a successful navigation-to-download handoff as an
    // aborted page request. The paired Playwright download event is verified
    // separately, so this one specific abort is expected.
    if (/^\/app\/dokumente\/[^/]+\/download$/.test(path) && error.includes('ERR_ABORTED')) return;
    browserEvents.push(`requestfailed ${path}: ${error}`);
  });
}

async function newContext(width, options = {}) {
  const context = await browser.newContext({
    viewport: viewportFor(width),
    reducedMotion: options.reducedMotion,
    javaScriptEnabled: options.javaScriptEnabled,
  });
  activeContexts.add(context);
  context.on('page', watchPage);
  return context;
}

async function closeContext(context) {
  activeContexts.delete(context);
  await context.close();
}

async function localLogin(context, email = 'admin@example.com') {
  if (email === 'admin@example.com' && adminStorageState) {
    await context.addCookies(adminStorageState.cookies);
    const cachedPage = await context.newPage();
    await cachedPage.goto(`${baseURL}/app`, { waitUntil: 'networkidle' });
    if (cachedPage.url().includes('/app')) return cachedPage;
    await cachedPage.close();
    adminStorageState = undefined;
  }
  const page = await context.newPage();
  await page.goto(`${baseURL}/`, { waitUntil: 'networkidle' });
  const details = page.locator('details:has(form[action="/auth/request"])');
  if (await details.count()) await details.evaluate((element) => { element.open = true; });
  await page.locator('input[name="email"]').fill(email);
  await page.locator('form[action="/auth/request"] button[type="submit"]').click();
  const link = page.locator('a.dev-link');
  await link.waitFor({ state: 'visible' });
  const href = await link.getAttribute('href');
  if (!href) fail(`Kein lokaler Anmeldelink für ${email}`);
  const target = new URL(href, baseURL);
  const local = new URL(baseURL);
  target.protocol = local.protocol;
  target.hostname = local.hostname;
  target.port = local.port;
  await page.goto(target.href, { waitUntil: 'networkidle' });
  if (!page.url().includes('/app')) fail(`Anmeldung endete auf ${page.url()}`);
  if (email === 'admin@example.com') adminStorageState = await context.storageState();
  return page;
}

async function assertNoOverflow(page, label) {
  const geometry = await page.evaluate(() => ({
    viewport: document.documentElement.clientWidth,
    document: document.documentElement.scrollWidth,
    body: document.body.scrollWidth,
  }));
  if (geometry.document > geometry.viewport + 1 || geometry.body > geometry.viewport + 1) {
    fail(`${label}: horizontaler Überlauf ${JSON.stringify(geometry)}`);
  }
}

async function assertMobileTargets(page, selectors, label) {
  const undersized = await page.locator(selectors.join(',')).evaluateAll((elements) => elements
    .filter((element) => {
      const style = getComputedStyle(element);
      const box = element.getBoundingClientRect();
      return style.display !== 'none' && style.visibility !== 'hidden' && box.width > 0 && box.height > 0 &&
        (box.width < 43.5 || box.height < 43.5);
    })
    .map((element) => ({
      tag: element.tagName,
      text: element.textContent?.trim().replace(/\s+/g, ' ').slice(0, 80),
      width: element.getBoundingClientRect().width,
      height: element.getBoundingClientRect().height,
    })));
  if (undersized.length) fail(`${label}: zu kleine Ziele ${JSON.stringify(undersized)}`);
}

async function capture(page, name, fullPage = true) {
  await page.screenshot({ path: join(screenshotDir, `${name}.png`), fullPage });
}

async function ensureUnit() {
  const context = await newContext(1024);
  const page = await localLogin(context);
  await page.goto(`${baseURL}/app/settings/building#units`, { waitUntil: 'networkidle' });
  if (!(await page.getByText('Einheit 12', { exact: true }).count())) {
    const panel = page.locator('#unit-add');
    if (!(await panel.evaluate((element) => element.open))) await panel.locator('summary').click();
    const form = panel.locator('form');
    await form.locator('input[name="label"]').fill('Einheit 12');
    await form.locator('input[name="owner_emails"]').fill('owner@example.com');
    await form.locator('input[name="renter_emails"]').fill('resident@example.com');
    await form.getByRole('button', { name: 'Einheit anlegen' }).click();
    await page.waitForURL(/unit=saved/);
  }
  await closeContext(context);
}

async function captureEmptyAndDialogBaseline() {
  for (const width of widths) {
    const context = await newContext(width);
    const page = await localLogin(context);
    await page.goto(`${baseURL}/app/uebergaben`, { waitUntil: 'networkidle' });
    await assertNoOverflow(page, `Leere Übergaben ${width}`);
    await capture(page, `baseline-empty-${width}`);

    await page.getByRole('button', { name: /Übergabe anlegen/ }).first().click();
    const dialog = page.locator('#handover-create[open]');
    await dialog.waitFor({ state: 'visible' });
    await assertNoOverflow(page, `Dialog ${width}`);
    if (width <= 390) {
      await assertMobileTargets(page, ['#handover-create .dialog-close', '#handover-create button[type="submit"]'], `Dialog ${width}`);
    }
    await capture(page, `baseline-dialog-${width}`, false);
    await page.keyboard.press('Escape');
    await closeContext(context);
  }
}

const fixturePNG = Buffer.from(
  'iVBORw0KGgoAAAANSUhEUgAAAAIAAAACCAYAAABytg0kAAAAFElEQVR42mP8z8Dwn4GBgYGJAQoAHgQCAf+srE0AAAAASUVORK5CYII=',
  'base64',
);

async function createHandover() {
  const context = await newContext(1024);
  const page = await localLogin(context);
  await page.goto(`${baseURL}/app/uebergaben`, { waitUntil: 'networkidle' });
  await page.getByRole('button', { name: /Übergabe anlegen/ }).first().click();
  const dialog = page.locator('#handover-create[open]');
  await dialog.locator('input[name="title"]').fill('Nutzerwechsel Einheit 12');
  await dialog.locator('select[name="unit_id"]').selectOption({ label: 'Einheit 12' });
  await dialog.locator('[name="rooms_text"]').fill('Wohnzimmer | sehr gut | keine Mängel');

  const people = dialog.locator('details.dialog-optional').filter({ hasText: 'Personen für Bestätigung' });
  await people.locator('summary').click();
  await dialog.locator('[name="outgoing_name"]').fill('Alex Ausziehend');
  await dialog.locator('[name="outgoing_email"]').fill('alex@example.invalid');
  await dialog.locator('[name="incoming_name"]').fill('Ina Einziehend');
  await dialog.locator('[name="incoming_email"]').fill('ina@example.invalid');

  const extras = dialog.locator('details.dialog-optional').filter({ hasText: 'Zähler, Schlüssel und Notiz' });
  await extras.locator('summary').click();
  await dialog.locator('[name="meters_text"]').fill('Strom | 12345,6 | kWh\nWasser | 87,2 | m³');
  await dialog.locator('[name="keys_text"]').fill('Wohnungsschlüssel | 3\nBriefkasten | 2');
  await dialog.locator('[name="notes"]').fill('Alle Räume gemeinsam begangen.');

  const files = dialog.locator('details.dialog-optional').filter({ hasText: 'Fotos und PDF' });
  await files.locator('summary').click();
  await dialog.locator('[name="attachments"]').setInputFiles({
    name: 'wohnzimmer-qa.png',
    mimeType: 'image/png',
    buffer: fixturePNG,
  });
  await capture(page, 'baseline-dialog-complete-1024', false);
  await dialog.getByRole('button', { name: 'Übergabe anlegen' }).click();
  await page.waitForURL(/handover=created/);
  const card = page.locator('article.handover-card').filter({ hasText: 'Nutzerwechsel Einheit 12' }).first();
  await card.waitFor({ state: 'visible' });
  const id = (await card.getAttribute('id'))?.replace(/^handover-/, '');
  if (!id) fail('ID der angelegten Übergabe fehlt');
  await closeContext(context);
  return id;
}

function installFixtureTokens(id, tokens) {
  const dbPath = process.env.DB_PATH?.trim() || join(process.env.HV_DATA || '', 'hausv.db');
  if (!dbPath || !existsSync(dbPath)) fail(`Fixture-Datenbank fehlt: ${dbPath}`);
  const helper = fileURLToPath(new URL('./set-handover-fixture-tokens.py', import.meta.url));
  const result = spawnSync('python3', [helper, dbPath, id], {
    encoding: 'utf8',
    input: `${JSON.stringify({ tokens })}\n`,
  });
  if (result.status !== 0 || result.stdout.trim() !== '1') {
    fail(`Bestätigungstoken konnten nicht installiert werden: ${result.stderr || result.stdout}`);
  }
}

async function captureManagerState(state) {
  for (const width of widths) {
    const context = await newContext(width);
    const page = await localLogin(context);
    await page.goto(`${baseURL}/app/uebergaben`, { waitUntil: 'networkidle' });
    if (state === 'filed') {
      const filedSection = page.locator('details.handover-section').filter({ hasText: 'Abgeschlossen' });
      if (!(await filedSection.evaluate((element) => element.open))) await filedSection.locator(':scope > summary').click();
    }
    const card = page.locator('article.handover-card').filter({ hasText: 'Nutzerwechsel Einheit 12' }).first();
    await card.waitFor({ state: 'visible' });
    await assertNoOverflow(page, `${state} Verwaltung ${width}`);
    if (width <= 390) {
      await assertMobileTargets(page, ['.handover-section > summary', '.handover-details > summary', '.handover-next .button'], `${state} Verwaltung ${width}`);
    }
    await capture(page, `${state}-manager-${width}`);
    await closeContext(context);
  }
}

async function capturePublicReview(token, state) {
  for (const width of widths) {
    const context = await newContext(width);
    const page = await context.newPage();
    await page.goto(`${baseURL}/handover/${token}`, { waitUntil: 'networkidle' });
    await page.getByRole('heading', { name: 'Nutzerwechsel Einheit 12' }).waitFor();
    const publicAttachment = page.getByRole('link', { name: /wohnzimmer-qa\.png/ });
    await publicAttachment.waitFor();
    const attachmentHref = await publicAttachment.getAttribute('href');
    if (!attachmentHref?.startsWith(`/handover/${token}/attachments/`)) {
      fail(`${state} öffentlich ${width}: Anhang ist nicht an den Bestätigungslink gebunden`);
    }
    const attachmentResponse = await page.request.get(new URL(attachmentHref, baseURL).href);
    if (!attachmentResponse.ok() || !attachmentResponse.headers()['cache-control']?.includes('no-store')) {
      fail(`${state} öffentlich ${width}: tokengebundener Anhang ist nicht sicher erreichbar`);
    }
    await assertNoOverflow(page, `${state} öffentlich ${width}`);
    if (width <= 390) {
      await assertMobileTargets(page, ['.handover-review summary', '.handover-confirm-note > summary', '.handover-confirm-consent', '.handover-confirm-submit button'], `${state} öffentlich ${width}`);
    }
    await capture(page, `${state}-public-${width}`);
    await closeContext(context);
  }
}

async function confirmToken(token, note) {
  const context = await newContext(390);
  const page = await context.newPage();
  await page.goto(`${baseURL}/handover/${token}`, { waitUntil: 'networkidle' });
  const noteDetails = page.locator('.handover-confirm-note');
  await noteDetails.locator('summary').click();
  await noteDetails.locator('textarea[name="note"]').fill(note);
  await page.locator('input[name="confirm"]').check();
  await capture(page, `confirm-ready-${token}`, false);
  await page.getByRole('button', { name: 'Protokoll bestätigen' }).click();
  await page.waitForURL(/handover=confirmed/);
  await page.getByText(/nichts weiter zu tun/).waitFor();
  await capture(page, `confirmed-${token}-390`);
  await closeContext(context);
}

async function assertConfirmedTokenReadOnly(token) {
  const context = await newContext(390);
  const page = await context.newPage();
  await page.goto(`${baseURL}/handover/${token}`, { waitUntil: 'networkidle' });
  await page.getByText('Protokoll bestätigt', { exact: true }).waitFor();
  if (await page.locator('form.handover-confirm-form').count()) fail('Bestätigter Link zeigt erneut ein Bestätigungsformular');
  if ((await context.cookies()).length) fail('Öffentlicher Bestätigungslink hat eine Portal-Sitzung erzeugt');

  const attachmentHref = await page.locator('.handover-public-file').first().getAttribute('href');
  if (!attachmentHref) fail('Bestätigter Link verliert den tokengebundenen Anhang');
  const attachmentID = attachmentHref.split('/').at(-1);
  const directGuess = await context.request.get(`${baseURL}/app/attachments/${attachmentID}`, { maxRedirects: 0 });
  if (directGuess.status() === 200) fail('Öffentlicher Nutzer konnte den Anhang ohne Bestätigungstoken direkt öffnen');
  const unrelatedGuess = await context.request.get(`${baseURL}/handover/${token}/attachments/not-this-handover`, { maxRedirects: 0 });
  if (unrelatedGuess.status() !== 404) fail(`Bestätigungstoken akzeptiert fremde Anhang-ID (${unrelatedGuess.status()})`);

  const repeated = await context.request.post(`${baseURL}/handover/${token}`, {
    form: { confirm: 'yes', name: 'Andere Person', note: 'darf nichts ändern' },
    headers: { Origin: new URL(baseURL).origin },
    maxRedirects: 0,
  });
  if (repeated.status() !== 303) fail(`Wiederholte Bestätigung antwortet ${repeated.status()} statt read-only Redirect`);
  await page.reload({ waitUntil: 'networkidle' });
  if (await page.locator('form.handover-confirm-form').count()) fail('Wiederholte Bestätigung hat das Formular reaktiviert');
  await capture(page, `confirmed-public-read-only-${token}-390`);
  await closeContext(context);
}

async function assertLockedAfterFirstConfirmation() {
  const context = await newContext(390);
  const page = await localLogin(context);
  await page.goto(`${baseURL}/app/uebergaben`, { waitUntil: 'networkidle' });
  const card = page.locator('article.handover-card').filter({ hasText: 'Nutzerwechsel Einheit 12' }).first();
  if (await card.locator('form.handover-add-files').count()) fail('Dateien bleiben nach erster Bestätigung veränderbar');
  if (await card.locator('.attachment-delete').count()) fail('Anhang bleibt nach erster Bestätigung löschbar');
  if (await card.locator('details.handover-details').evaluate((element) => element.open)) fail('Halb bestätigte Manager-Karte öffnet Details ungefragt');
  await capture(page, 'half-confirmed-manager-locked-390');
  await closeContext(context);
}

async function fileAndVerify() {
  const context = await newContext(390);
  const page = await localLogin(context);
  await page.goto(`${baseURL}/app/uebergaben`, { waitUntil: 'networkidle' });
  const card = page.locator('article.handover-card').filter({ hasText: 'Nutzerwechsel Einheit 12' }).first();
  await card.getByRole('button', { name: 'Jetzt ablegen' }).click();
  await page.waitForURL(/handover=filed/);
  await page.getByRole('link', { name: 'Dokument öffnen' }).waitFor();
  await capture(page, 'filed-manager-390', false);

  const [filedDownload] = await Promise.all([
    page.waitForEvent('download'),
    page.getByRole('link', { name: 'Dokument öffnen' }).click(),
  ]);
  if (!filedDownload.suggestedFilename().endsWith('.pdf')) fail('Abgelegtes Protokoll ist kein PDF');

  await page.goto(`${baseURL}/app/uebergaben`, { waitUntil: 'networkidle' });
  const filedSection = page.locator('details.handover-section').filter({ hasText: 'Abgeschlossen' });
  if (!(await filedSection.evaluate((element) => element.open))) await filedSection.locator(':scope > summary').click();
  const filedCard = page.locator('article.handover-card').filter({ hasText: 'Nutzerwechsel Einheit 12' }).first();
  await filedCard.locator('details.handover-details summary').click();
  const image = filedCard.locator('.attachment-open').first();
  if (!(await image.count())) fail('Anhang fehlt im abgelegten Protokoll');
  await image.click();
  await page.locator('.attachment-lightbox.open').waitFor({ state: 'visible' });
  await capture(page, 'filed-attachment-lightbox-390', false);
  await page.keyboard.press('Escape');
  await closeContext(context);
}

async function assertKeyboardAndReducedMotion(token) {
  const reducedContext = await newContext(390, { reducedMotion: 'reduce' });
  const reducedPage = await reducedContext.newPage();
  await reducedPage.goto(`${baseURL}/handover/${token}`, { waitUntil: 'networkidle' });
  await assertNoOverflow(reducedPage, 'Bestätigt, reduzierte Bewegung');
  await closeContext(reducedContext);

  const context = await newContext(390);
  const page = await localLogin(context);
  await page.goto(`${baseURL}/app/uebergaben`, { waitUntil: 'networkidle' });
  const trigger = page.getByRole('button', { name: /Übergabe anlegen/ }).first();
  await trigger.focus();
  await page.keyboard.press('Enter');
  const dialog = page.locator('#handover-create[open]');
  await dialog.waitFor({ state: 'visible' });
  if (!(await dialog.locator('.dialog-close').evaluate((element) => element === document.activeElement))) {
    fail('Dialog setzt den Fokus nicht auf das erste Feld');
  }
  await page.keyboard.press('Escape');
  if (!(await trigger.evaluate((element) => element === document.activeElement))) {
    fail('Dialog gibt Fokus nicht an den Auslöser zurück');
  }
  await closeContext(context);
}

async function run() {
  await ensureUnit();
  await captureEmptyAndDialogBaseline();
  const id = await createHandover();
  const tokens = ['qa-handover-outgoing', 'qa-handover-incoming'];
  installFixtureTokens(id, tokens);
  await captureManagerState('pending');
  await capturePublicReview(tokens[0], 'review');
  await confirmToken(tokens[0], 'Gemeinsam geprüft.');
  await assertConfirmedTokenReadOnly(tokens[0]);
  await assertLockedAfterFirstConfirmation();
  await captureManagerState('half-confirmed');
  await confirmToken(tokens[1], 'Übernahme bestätigt.');
  await captureManagerState('ready');
  await fileAndVerify();
  await captureManagerState('filed');
  await assertKeyboardAndReducedMotion(tokens[0]);
  if (browserEvents.length) fail(`Browserfehler:\n${browserEvents.join('\n')}`);
  writeFileSync(join(artifactRoot, 'report.json'), `${JSON.stringify({
    baseURL,
    handoverID: id,
    widths,
    screenshots: 'screenshots/',
    result: 'pass',
  }, null, 2)}\n`);
  process.stdout.write(`  ✓ Übergabe · Anlegen → 2× bestätigen → sperren → PDF ablegen · ${widths.join('/')} px\n`);
  process.stdout.write(`  ✓ Evidenz: ${artifactRoot}\n`);
}

try {
  await run();
} catch (error) {
  writeFileSync(join(artifactRoot, 'failure.txt'), `${error instanceof Error ? error.stack : error}\n`);
  console.error(error);
  process.exitCode = 1;
} finally {
  for (const context of [...activeContexts]) await context.close().catch(() => {});
  await browser.close();
}
