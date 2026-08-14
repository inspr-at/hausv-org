#!/usr/bin/env node
// Deterministic headless QA for the settings and parking journeys.
//
// Normal use (against the isolated snapshot server):
//   HV_QA_PARKING_STATE=empty node qa-settings-parking.mjs http://localhost:8121
//   HV_QA_PARKING_STATE=populated node qa-settings-parking.mjs http://localhost:8121
//
// Before starting the populated server, create its rolling two-month fixture:
//   node qa-settings-parking.mjs --write-populated-fixture /tmp/data/parking.json

import { dirname, join } from 'node:path';
import { existsSync, mkdirSync, writeFileSync } from 'node:fs';
import { chromium } from 'playwright';

const fixtureCommand = process.argv[2] === '--write-populated-fixture';
if (fixtureCommand) {
  const target = process.argv[3];
  if (!target) throw new Error('usage: qa-settings-parking.mjs --write-populated-fixture <parking.json>');
  writePopulatedFixture(target);
  process.stdout.write(`  ✓ Parkplatz-Fixture geschrieben: ${target}\n`);
  process.exit(0);
}

const baseURL = process.argv[2];
if (!baseURL) {
  console.error('usage: qa-settings-parking.mjs <baseURL> [artifact-dir]');
  process.exit(1);
}
const parsedBaseURL = new URL(baseURL);
if (!['localhost', '127.0.0.1', '::1'].includes(parsedBaseURL.hostname) && process.env.HV_QA_ALLOW_REMOTE !== 'true') {
  throw new Error('Settings/Parking-QA mutates fixture data and therefore only runs against localhost');
}

const expectedParkingState = (process.env.HV_QA_PARKING_STATE || 'empty').trim();
if (!['empty', 'populated'].includes(expectedParkingState)) {
  throw new Error('HV_QA_PARKING_STATE must be empty or populated');
}
const artifactDir = process.argv[3] || (process.env.HV_QA_ARTIFACT_DIR
  ? join(process.env.HV_QA_ARTIFACT_DIR, `settings-parking-${expectedParkingState}`)
  : '');
if (artifactDir) mkdirSync(artifactDir, { recursive: true });

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
const browser = await chromium.launch({
  headless: true,
  args: ['--no-proxy-server'],
  ...(executablePath ? { executablePath } : {}),
});

const viewports = [
  { label: '320x568', width: 320, height: 568, touch: true },
  { label: '390x844', width: 390, height: 844, touch: true },
  { label: '1024x600', width: 1024, height: 600, touch: false },
  { label: '1440x720', width: 1440, height: 720, touch: false },
];
const report = [];
const storageStates = new Map();
let activePage;

function fail(message) {
  throw new Error(message);
}

function artifactName(value) {
  return value.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '') || 'page';
}

async function login(context, email) {
  const cached = storageStates.get(email);
  if (cached) {
    await context.addCookies(cached.cookies);
    return;
  }
  const page = await context.newPage();
  await page.goto(`${baseURL}/`, { waitUntil: 'networkidle' });
  const details = page.locator('details:has(form[action="/auth/request"])');
  if (await details.count()) await details.evaluate((node) => { node.open = true; });
  await page.locator('input[name="email"]').fill(email);
  await page.locator('form[action="/auth/request"] button[type="submit"]').click();
  const devLink = page.locator('a.dev-link');
  await devLink.waitFor({ state: 'visible', timeout: 10_000 });
  const href = await devLink.getAttribute('href');
  if (!href) fail(`Kein lokaler Anmeldelink für ${email}`);
  const target = new URL(href, baseURL);
  target.protocol = parsedBaseURL.protocol;
  target.hostname = parsedBaseURL.hostname;
  target.port = parsedBaseURL.port;
  await page.goto(target.href, { waitUntil: 'networkidle' });
  if (!page.url().includes('/app')) fail(`Anmeldung für ${email} endete auf ${page.url()}`);
  storageStates.set(email, await context.storageState());
  await page.close();
}

async function loggedInPage(viewport, email = 'admin@example.com') {
  const context = await browser.newContext({
    viewport: { width: viewport.width, height: viewport.height },
    locale: 'de-AT',
    timezoneId: 'Europe/Vienna',
  });
  await login(context, email);
  const page = await context.newPage();
  const events = [];
  page.on('console', (message) => {
    if (message.type() === 'error' || message.type() === 'warning') {
      events.push(`console.${message.type()}: ${message.text()}`);
    }
  });
  page.on('pageerror', (error) => events.push(`pageerror: ${error.message}`));
  page.on('requestfailed', (request) => {
    const url = new URL(request.url());
    events.push(`requestfailed: ${request.method()} ${url.pathname}: ${request.failure()?.errorText || 'unknown'}`);
  });
  return { context, page, events };
}

async function open(page, events, path, label) {
  events.length = 0;
  const response = await page.goto(`${baseURL}${path}`, { waitUntil: 'networkidle' });
  const expected = new URL(path, baseURL);
  const current = new URL(page.url());
  const sameDocument = !response && current.pathname === expected.pathname && current.search === expected.search;
  if ((!response && !sameDocument) || (response && response.status() !== 200)) {
    fail(`${label}: HTTP ${response?.status() || 'ohne Antwort'}`);
  }
  await page.evaluate(() => document.fonts.ready);
  if (events.length) fail(`${label}: Browserfehler: ${events.join(' | ')}`);
}

async function geometry(page, viewport, label) {
  const result = await page.evaluate(({ touch }) => {
    const visible = (node) => {
      const style = getComputedStyle(node);
      const rect = node.getBoundingClientRect();
      return style.visibility !== 'hidden' && style.display !== 'none' && rect.width > 0 && rect.height > 0;
    };
    const targetSelector = 'a[href],button,input:not([type="hidden"]):not([type="checkbox"]):not([type="radio"]),select,textarea,summary,[tabindex]:not([tabindex="-1"])';
    const smallTargets = touch ? [...document.querySelectorAll(targetSelector)]
      .filter(visible)
      .map((node) => {
        const rect = node.getBoundingClientRect();
        const labelNode = node.closest('label');
        const effective = labelNode && visible(labelNode) ? labelNode.getBoundingClientRect() : rect;
        return {
          text: (node.getAttribute('aria-label') || node.textContent || node.getAttribute('name') || '').trim().replace(/\s+/g, ' ').slice(0, 80),
          width: Math.round(effective.width),
          height: Math.round(effective.height),
        };
      })
      .filter((target) => target.width < 44 || target.height < 44) : [];
    const clipped = [...document.querySelectorAll('.panel,.settings-section,.settings-link,.access-row,.parking-empty-step,.parking-month-summary,.upload-form')]
      .filter(visible)
      .map((node) => {
        const rect = node.getBoundingClientRect();
        return { className: node.className, left: Math.round(rect.left), right: Math.round(rect.right) };
      })
      .filter((item) => item.left < -1 || item.right > innerWidth + 1);
    const labelNode = document.querySelector('.side-address-label');
    const mapLink = document.querySelector('a.side-address');
    const homeLink = document.querySelector('a.side-place-copy');
    const brand = document.querySelector('.side-brand');
    const menu = document.querySelector('.mobile-menu-toggle');
    const brandRect = brand?.getBoundingClientRect();
    const menuRect = menu && visible(menu) ? menu.getBoundingClientRect() : null;
    return {
      overflowX: Math.max(0, document.documentElement.scrollWidth - innerWidth),
      smallTargets,
      clipped,
      address: labelNode ? labelNode.innerText.trim().replace(/\s+/g, ' ') : '',
      addressWidth: labelNode ? Math.round(labelNode.getBoundingClientRect().width) : 0,
      addressScrollWidth: labelNode?.scrollWidth || 0,
      mapLabel: mapLink?.getAttribute('aria-label') || '',
      homeLabel: homeLink?.getAttribute('aria-label') || '',
      shellOverlap: Boolean(menuRect && brandRect && brandRect.right > menuRect.left + 1),
      bodyWidth: Math.round(document.body.getBoundingClientRect().width),
    };
  }, { touch: viewport.touch });
  if (result.overflowX > 1) fail(`${label}: ${result.overflowX}px horizontaler Überlauf`);
  if (result.smallTargets.length) fail(`${label}: Touch-Ziele unter 44px: ${JSON.stringify(result.smallTargets)}`);
  if (result.clipped.length) fail(`${label}: seitlich abgeschnitten: ${JSON.stringify(result.clipped)}`);
  if (result.shellOverlap) fail(`${label}: Ortskopf und Menü überlappen`);
  if (!result.address.startsWith('Musterweg 1') || /\bDEMO\b/i.test(result.address)) {
    fail(`${label}: sichtbare Adresse ist nicht sinnvoll ausgeschrieben (${result.address})`);
  }
  if (!result.mapLabel.includes('Musterweg 1, 1010 Wien') ||
      !result.homeLabel.includes('Musterweg 1, 1010 Wien')) {
    fail(`${label}: vollständige Adresse fehlt in den zugänglichen Linknamen`);
  }
  if (viewport.width === 320 && result.addressScrollWidth > result.addressWidth + 1) {
    fail(`${label}: ausgeschriebene Straße wird bei 320px abgeschnitten`);
  }
  return result;
}

async function capture(page, viewport, slug) {
  if (!artifactDir) return;
  await page.evaluate(() => {
    if (document.activeElement instanceof HTMLElement) document.activeElement.blur();
    window.scrollTo(0, 0);
  });
  await page.waitForTimeout(50);
  await page.screenshot({
    path: join(artifactDir, `${artifactName(slug)}-${viewport.label}.png`),
    fullPage: true,
  });
}

async function recordPage(page, events, viewport, path, slug, verify) {
  const label = `${slug} ${viewport.label}`;
  await open(page, events, path, label);
  await verify(page, label);
  const layout = await geometry(page, viewport, label);
  report.push({ slug, path, viewport: viewport.label, ...layout });
  await capture(page, viewport, slug);
}

async function recordCurrentPage(page, events, viewport, slug) {
  if (events.length) fail(`${slug}: Browserfehler: ${events.join(' | ')}`);
  const layout = await geometry(page, viewport, slug);
  report.push({ slug, path: new URL(page.url()).pathname, viewport: viewport.label, ...layout });
  await capture(page, viewport, slug);
}

async function assertSettingsHub(page, label) {
  if (!(await page.getByRole('heading', { name: 'Einstellungen', exact: true }).count())) fail(`${label}: Überschrift fehlt`);
  for (const heading of ['Mein Zuhause', 'Mein Konto', 'Kommunikation', 'Verwaltung']) {
    if (!(await page.getByRole('heading', { name: heading, exact: true }).count())) fail(`${label}: Bereich ${heading} fehlt`);
  }
  const guide = page.locator('details.settings-guide-details');
  if ((await guide.count()) !== 1 || await guide.evaluate((node) => node.open)) fail(`${label}: Rechtehilfe ist nicht kompakt eingeklappt`);
  if (await page.locator('.settings-note').count()) fail(`${label}: redundante Rollenkarte ist wieder vorhanden`);
}

async function assertProgressiveSettings(page, label, heading, detailsSelector) {
  if (!(await page.getByRole('heading', { name: heading, exact: true }).count())) fail(`${label}: Überschrift ${heading} fehlt`);
  if (await page.locator('.content-top .page-actions a').count()) fail(`${label}: redundante Kopfaktion ist wieder vorhanden`);
  const details = page.locator(detailsSelector);
  if (!(await details.count()) || await details.evaluateAll((nodes) => nodes.some((node) => node.open))) {
    fail(`${label}: Zusatzinformationen sind nicht progressiv eingeklappt`);
  }
}

async function assertAccess(page, label, viewport) {
  if (!(await page.getByRole('heading', { name: 'Parkplatz verwalten', exact: true }).count())) fail(`${label}: Überschrift fehlt`);
  const rows = await page.locator('.access-row').evaluateAll((nodes) => nodes.map((node) => ({
    name: node.querySelector('.access-person-copy strong')?.textContent?.trim() || '',
    email: node.querySelector('.access-person-copy span')?.textContent?.trim() || '',
  })));
  const requiredEmails = ['admin@example.com', 'owner@example.com', 'resident@example.com', 'verwalter@example.com'];
  // The unified browser gate deliberately keeps its fake data between
  // lifecycles. The base flow may therefore have created this valid DEMO
  // technical helper before the settings flow starts. Prove the stable
  // principals and tenant boundary without treating legitimate prior state as
  // a leak from another house.
  const allowedEmails = new Set([...requiredEmails, 'qa-helper@example.com']);
  const emails = rows.map((row) => row.email).sort();
  if (requiredEmails.some((email) => !emails.includes(email)) || emails.some((email) => !allowedEmails.has(email))) {
    fail(`${label}: Zugriffsliste verletzt den erwarteten Hausumfang (${JSON.stringify(rows)})`);
  }
  if (rows.some((row) => !row.name || row.name === row.email)) fail(`${label}: E-Mail wird trotz Name als Primärlabel gezeigt`);
  if (rows.some((row) => /house_a|house_b|cockpit/i.test(row.email))) fail(`${label}: fremder Mandant in Zugriffsliste`);
  const fixed = page.locator('.access-fixed').first();
  if (!(await fixed.count()) || !(await fixed.getAttribute('aria-label'))?.includes('hier nicht änderbar')) {
    fail(`${label}: Erklärung für festen Zugriff fehlt`);
  }
  if (viewport.width >= 1024) {
    await fixed.hover();
    await page.waitForTimeout(180);
    const tooltipVisible = await fixed.evaluate((node) => Number.parseFloat(getComputedStyle(node, '::after').opacity) > 0.9);
    if (!tooltipVisible) fail(`${label}: Hover-Hilfe wird nicht sichtbar`);
  }
}

async function assertParking(page, label) {
  if (!(await page.getByRole('heading', { name: 'Parkplatz', exact: true }).count())) fail(`${label}: Überschrift fehlt`);
  if (expectedParkingState === 'empty') {
    if (!(await page.locator('.parking-empty').count()) || !(await page.getByText('Noch keine Monatswerte', { exact: true }).count())) {
      fail(`${label}: deterministischer Leerzustand fehlt`);
    }
    for (const step of ['Tarif', 'Messwerte', 'Monat']) {
      if (!(await page.locator('.parking-empty-step').getByText(step, { exact: true }).count())) fail(`${label}: Schritt ${step} fehlt`);
    }
    if (!(await page.getByText('Nur Nachweis.', { exact: true }).count())) fail(`${label}: Sicherheitsrahmen fehlt`);
    return;
  }
  if (!(await page.locator('.parking-current-month').count()) || !(await page.locator('.parking-compact-month').count())) {
    fail(`${label}: befüllte Monatsübersicht fehlt`);
  }
  if (!(await page.locator('.parking-current-month .pill.ok').count())) fail(`${label}: bezahlter aktueller Monat fehlt`);
  if (!(await page.locator('.parking-compact-month .pill').filter({ hasText: 'Offen' }).count())) fail(`${label}: offener Vormonat fehlt`);
}

async function assertMonth(page, label, paid) {
  if (!(await page.locator('.parking-month-summary').count())) fail(`${label}: Monatszusammenfassung fehlt`);
  if ((await page.locator('.parking-month-essential > div').count()) !== 3) fail(`${label}: drei Kernwerte fehlen`);
  if (paid) {
    if (!(await page.getByText(/Bezahlt am/).count()) || !(await page.getByText('Überweisung · QA-ZAHLUNG', { exact: true }).count())) {
      fail(`${label}: bezahlter Zustand ist nicht klar`);
    }
  } else if (!(await page.getByRole('button', { name: 'Bezahlung erhalten', exact: true }).count())) {
    fail(`${label}: offener Zahlungsweg fehlt`);
  }
}

async function assertPaymentImport(page, label) {
  if (!(await page.getByRole('heading', { name: 'Zahlungen aus Bankdatei', exact: true }).count())) fail(`${label}: Überschrift fehlt`);
  const picker = page.locator('.upload-form .file-control');
  if (!(await picker.count()) || !(await page.locator('#camt-file').count())) fail(`${label}: polierter Dateiwähler fehlt`);
  if ((await picker.boundingBox())?.width > page.viewportSize().width + 1) fail(`${label}: Dateiwähler ist breiter als der Viewport`);
}

async function createPaymentPreview() {
  const viewport = viewports.find((item) => item.label === '390x844');
  const { context, page, events } = await loggedInPage(viewport);
  activePage = page;
  await open(page, events, '/app/settings/building#units', 'Zahlungs-Fixture: Einheit');
  if (!(await page.getByText('QA Top 42', { exact: true }).count())) {
    const add = page.locator('#unit-add');
    await add.evaluate((node) => { node.open = true; });
    const form = add.locator('form');
    await form.locator('input[name="label"]').fill('QA Top 42');
    await form.locator('input[name="owner_emails"]').fill('owner@example.com');
    await Promise.all([
      page.waitForURL(/\/app\/settings\/building/),
      form.getByRole('button', { name: 'Einheit anlegen' }).click(),
    ]);
  }
  await open(page, events, '/app/settings/building#units', 'Zahlungs-Fixture: Status');
  const unit = page.locator('details.unit-editor').filter({ hasText: 'QA Top 42' });
  if (!(await unit.count())) fail('Zahlungs-Fixture: angelegte Einheit fehlt');
  await unit.evaluate((node) => { node.open = true; });
  const statusForm = unit.locator('.payment-status-form');
  await statusForm.locator('select[name="status"]').selectOption('offen');
  await Promise.all([
    page.waitForURL(/payment=saved/),
    statusForm.getByRole('button', { name: 'Status speichern' }).click(),
  ]);
  if (!(await page.getByText('Zahlungsstatus gespeichert.', { exact: true }).count())) {
    fail('Zahlungs-Fixture: gespeicherter Einheitenstatus wird nicht bestätigt');
  }
  const period = rollingMonths().recent;
  await open(page, events, `/app/settings/payments/import?period=${period}`, 'Zahlungs-Fixture: Import');
  const references = page.locator('details.reference-disclosure');
  if (!(await references.count())) fail('Zahlungs-Fixture: Zahlungsreferenzen fehlen');
  await references.evaluate((node) => { node.open = true; });
  const row = references.locator('.reference-row').filter({ hasText: 'QA Top 42' });
  const reference = (await row.locator('code').innerText()).trim();
  if (!reference) fail('Zahlungs-Fixture: Referenz ist leer');
  const xml = camtXML(reference);
  await page.locator('#camt-file').setInputFiles({ name: 'qa-kontoauszug.xml', mimeType: 'application/xml', buffer: Buffer.from(xml) });
  if (!(await page.locator('.attachment-picker-name').getByText('qa-kontoauszug.xml', { exact: true }).count())) {
    fail('Zahlungs-Fixture: gewählte Datei wird nicht sichtbar bestätigt');
  }
  await Promise.all([
    page.waitForURL(/preview=/),
    page.getByRole('button', { name: 'Vorschau erstellen' }).click(),
  ]);
  if (!(await page.locator('#preview').count()) || !(await page.getByText('Zuordnen', { exact: true }).count())) {
    fail('Zahlungs-Fixture: sichere Vorschau fehlt');
  }
  const applyPosition = await page.locator('.apply-bar').evaluate((node) => getComputedStyle(node).position);
  if (applyPosition !== 'static') fail('Zahlungs-Fixture: Übernahmeaktion überlagert die mobile Vorschau');
  const body = await page.locator('body').innerText();
  for (const secret of ['Max Geheim', 'AT483200000012345864', 'NTR-SECRET']) {
    if (body.includes(secret)) fail(`Zahlungs-Fixture: Vorschau zeigt privaten Bankwert ${secret}`);
  }
  const layout = await geometry(page, viewport, 'Zahlungsvorschau 390x844');
  report.push({ slug: 'payment-preview', path: new URL(page.url()).pathname, viewport: viewport.label, ...layout });
  await capture(page, viewport, 'payment-preview');
  await context.close();
  activePage = undefined;
}

async function exerciseProfileAndNotifications() {
  const viewport = viewports.find((item) => item.label === '390x844');
  const { context, page, events } = await loggedInPage(viewport);
  activePage = page;

  await open(page, events, '/app/settings/profile', 'Profil speichern');
  const profile = page.locator('form.profile-form');
  await profile.locator('#profile-phone').fill('+43 316 555 0101');
  await profile.locator('#profile-directory').check();
  await Promise.all([
    page.waitForURL(/profile=saved/),
    profile.getByRole('button', { name: 'Profil speichern' }).click(),
  ]);
  if (!(await page.locator('.profile-flash.ok').getByText('Profil gespeichert.', { exact: true }).count()) ||
      (await page.locator('#profile-phone').inputValue()) !== '+43 316 555 0101') {
    fail('Profil speichern: Bestätigung oder persistierter Wert fehlt');
  }
  await recordCurrentPage(page, events, viewport, 'profile-save-success');

  await open(page, events, '/app/settings/notifications', 'Benachrichtigungen speichern');
  const notifications = page.locator('form.notification-form');
  await notifications.locator('[data-notification-master]').check();
  const announcement = notifications.locator('input[name="events"][value="announcement"]');
  if (await announcement.isChecked()) await announcement.locator('xpath=following-sibling::*[1]').click();
  await Promise.all([
    page.waitForURL(/notify=saved/),
    notifications.getByRole('button', { name: 'Benachrichtigungen speichern' }).click(),
  ]);
  if (!(await page.locator('.notify-flash.ok').getByText('Benachrichtigungen gespeichert.', { exact: true }).count()) ||
      await page.locator('input[name="events"][value="announcement"]').isChecked()) {
    fail('Benachrichtigungen speichern: Bestätigung oder persistierte Auswahl fehlt');
  }
  await recordCurrentPage(page, events, viewport, 'notifications-save-success');

  await context.close();
  activePage = undefined;
}

async function exerciseInviteLifecycle() {
  const viewport = viewports.find((item) => item.label === '390x844');
  const email = 'qa-flow-user@example.com';
  const { context, page, events } = await loggedInPage(viewport);
  activePage = page;
  await open(page, events, '/app/settings/users', 'Einladung anlegen');

  // A prior interrupted local run may have left the disposable fixture behind.
  const existing = page.locator(`[data-edit="${email}"]`);
  if (await existing.count()) await removeInvite(page, email);

  const invite = page.locator('#invite');
  await invite.evaluate((node) => { node.open = true; });
  const form = invite.locator('form.invite-form');
  await form.locator('input[name="email"]').fill(email);
  await form.locator('select[name="role"]').selectOption({ label: 'Mieter' });
  const personDetails = form.locator('details.f-person-details');
  await personDetails.evaluate((node) => { node.open = true; });
  await form.locator('input[name="first_name"]').fill('Quirin');
  await form.locator('input[name="last_name"]').fill('Prüfer');
  await Promise.all([
    page.waitForURL(/invite=(?:invited|saved_no_mail)/),
    form.getByRole('button', { name: 'Einladung senden' }).click(),
  ]);
  if (!(await page.getByText(email, { exact: true }).count()) || !(await page.getByText('Quirin Prüfer', { exact: true }).count())) {
    fail('Einladung anlegen: angelegter Testzugang fehlt');
  }

  await page.locator(`[data-edit="${email}"]`).click();
  let dialog = page.locator('dialog.edit-dialog').filter({ hasText: email });
  await dialog.waitFor({ state: 'visible' });
  await dialog.locator('select[name="role"]').selectOption({ label: 'Bewohner' });
  await Promise.all([
    page.waitForURL(/invite=updated/),
    dialog.getByRole('button', { name: 'Änderungen speichern' }).click(),
  ]);
  const editedRow = page.locator('tr').filter({ hasText: email });
  if (!(await page.locator('.invite-flash.ok').getByText('Änderungen gespeichert.', { exact: true }).count()) ||
      !(await editedRow.getByText('Bewohner', { exact: true }).count())) {
    fail('Einladung bearbeiten: Rollenänderung oder Bestätigung fehlt');
  }
  await recordCurrentPage(page, events, viewport, 'invite-edit-success');

  await exerciseParkingAccessLifecycle(page, events, viewport, email);
  await open(page, events, '/app/settings/users', 'Einladung nach Parkplatz-Zugriff entfernen');

  await page.locator(`[data-edit="${email}"]`).click();
  dialog = page.locator('dialog.edit-dialog').filter({ hasText: email });
  await dialog.waitFor({ state: 'visible' });
  const danger = dialog.locator('details.danger-zone');
  await danger.evaluate((node) => { node.open = true; });
  page.once('dialog', (confirmation) => confirmation.accept());
  await Promise.all([
    page.waitForURL(/invite=deleted/),
    danger.getByRole('button', { name: 'Dauerhaft entfernen' }).click(),
  ]);
  if ((await page.getByText(email, { exact: true }).count()) ||
      !(await page.locator('.invite-flash.ok').getByText('Zugang gelöscht.', { exact: true }).count())) {
    fail('Einladung entfernen: Zugang blieb sichtbar oder Bestätigung fehlt');
  }
  await recordCurrentPage(page, events, viewport, 'invite-delete-success');

  await context.close();
  activePage = undefined;
}

async function exerciseParkingAccessLifecycle(page, events, viewport, email) {
  await open(page, events, '/app/settings/parking-access', 'Parkplatz-Zugriff freigeben');
  let row = page.locator('.access-row').filter({ hasText: email });
  if (!(await row.count()) || !(await row.getByText('Quirin Prüfer', { exact: true }).count())) {
    fail('Parkplatz-Zugriff: der mandantenscharfe Testzugang fehlt');
  }
  await Promise.all([
    page.waitForURL(/parking_access=granted/),
    row.getByRole('button', { name: 'Freigeben', exact: true }).click(),
  ]);
  row = page.locator('.access-row').filter({ hasText: email });
  if (!(await page.locator('.access-flash.ok').getByText('Parkplatz-Zugriff freigegeben.', { exact: true }).count()) ||
      !(await row.getByText('Freigegeben', { exact: true }).count()) ||
      !(await row.getByRole('button', { name: 'Entziehen', exact: true }).count())) {
    fail('Parkplatz-Zugriff: Freigabe oder Bestätigung fehlt');
  }
  await recordCurrentPage(page, events, viewport, 'parking-access-grant-success');

  await Promise.all([
    page.waitForURL(/parking_access=revoked/),
    row.getByRole('button', { name: 'Entziehen', exact: true }).click(),
  ]);
  row = page.locator('.access-row').filter({ hasText: email });
  if (!(await page.locator('.access-flash.ok').getByText('Parkplatz-Zugriff entzogen.', { exact: true }).count()) ||
      !(await row.getByText('Kein Zugriff', { exact: true }).count()) ||
      !(await row.getByRole('button', { name: 'Freigeben', exact: true }).count())) {
    fail('Parkplatz-Zugriff: Entzug oder Bestätigung fehlt');
  }
  await recordCurrentPage(page, events, viewport, 'parking-access-revoke-success');
}

async function removeInvite(page, email) {
  await page.locator(`[data-edit="${email}"]`).click();
  const dialog = page.locator('dialog.edit-dialog').filter({ hasText: email });
  await dialog.waitFor({ state: 'visible' });
  const danger = dialog.locator('details.danger-zone');
  await danger.evaluate((node) => { node.open = true; });
  page.once('dialog', (confirmation) => confirmation.accept());
  await Promise.all([
    page.waitForURL(/invite=deleted/),
    danger.getByRole('button', { name: 'Dauerhaft entfernen' }).click(),
  ]);
}

async function exerciseStructuredExport() {
  const viewport = viewports.find((item) => item.label === '390x844');
  const { context, page, events } = await loggedInPage(viewport);
  activePage = page;
  await open(page, events, '/app/settings/data-export', 'Datenübergabe auswählen');
  await page.locator('input[name="source"][value="unit-payment-status"]').check();
  await Promise.all([
    page.waitForURL(/preview=/),
    page.getByRole('button', { name: 'Auswahl prüfen' }).click(),
  ]);
  if (!(await page.getByRole('heading', { name: 'Auswahl geprüft', exact: true }).count()) ||
      !(await page.getByText('CSV SHA-256:', { exact: false }).count())) {
    fail('Datenübergabe: prüfbare Vorschau fehlt');
  }
  await recordCurrentPage(page, events, viewport, 'data-export-preview');
  const [download] = await Promise.all([
    page.waitForEvent('download'),
    page.getByRole('button', { name: 'ZIP herunterladen' }).click(),
  ]);
  if (!/^hausv-rohdaten-demo-\d{8}-\d{6}\.zip$/.test(download.suggestedFilename())) {
    fail(`Datenübergabe: unerwarteter Dateiname ${download.suggestedFilename()}`);
  }
  const stream = await download.createReadStream();
  const chunks = [];
  for await (const chunk of stream) chunks.push(chunk);
  const archive = Buffer.concat(chunks);
  if (archive.length < 4 || archive.subarray(0, 2).toString('binary') !== 'PK') {
    fail('Datenübergabe: Download ist kein ZIP-Paket');
  }
  await context.close();
  activePage = undefined;
}

try {
  await createPaymentPreview();
  await exerciseProfileAndNotifications();
  await exerciseInviteLifecycle();
  await exerciseStructuredExport();
  const months = rollingMonths();
  for (const viewport of viewports) {
    const { context, page, events } = await loggedInPage(viewport);
    activePage = page;
    await recordPage(page, events, viewport, '/app/settings', 'settings-hub', assertSettingsHub);
    await recordPage(page, events, viewport, '/app/settings/profile', 'settings-profile', (candidate, label) => assertProgressiveSettings(candidate, label, 'Profil', '.profile details'));
    await recordPage(page, events, viewport, '/app/settings/notifications', 'settings-notifications', (candidate, label) => assertProgressiveSettings(candidate, label, 'Benachrichtigungen', '.notifications details'));
    await recordPage(page, events, viewport, '/app/settings/building', 'settings-building', async (candidate, label) => {
      if (!(await candidate.getByRole('heading', { name: 'Gebäude & Einheiten', exact: true }).count())) fail(`${label}: Überschrift fehlt`);
    });
    await recordPage(page, events, viewport, '/app/settings/users', 'settings-users', async (candidate, label) => {
      if (!(await candidate.getByRole('heading', { name: 'Benutzer & Rechte', exact: true }).count())) fail(`${label}: Überschrift fehlt`);
      if (viewport.width === 320) {
        const wrappedEmails = await candidate.locator('.person-mail').evaluateAll((nodes) => nodes
          .filter((node) => node.textContent.includes('@'))
          .map((node) => {
            const style = getComputedStyle(node);
            return { email: node.textContent.trim(), height: node.getBoundingClientRect().height, lineHeight: Number.parseFloat(style.lineHeight) };
          })
          .filter((item) => item.height > item.lineHeight * 1.35));
        if (wrappedEmails.length) fail(`${label}: E-Mail-Adressen brechen unsauber um: ${JSON.stringify(wrappedEmails)}`);
      }
    });
    await recordPage(page, events, viewport, '/app/settings/data-export', 'settings-data-export', async (candidate, label) => {
      if (!(await candidate.getByRole('heading', { name: 'Daten sicher weitergeben', exact: true }).count())) fail(`${label}: Überschrift fehlt`);
    });
    await recordPage(page, events, viewport, '/app/settings/parking-access', 'parking-access', (candidate, label) => assertAccess(candidate, label, viewport));
    await recordPage(page, events, viewport, '/app/parking/settings?section=accounting', 'parking-settings', async (candidate, label) => {
      if (!(await candidate.getByRole('heading', { name: 'Parkplatz verwalten', exact: true }).count())) fail(`${label}: Überschrift fehlt`);
    });
    await recordPage(page, events, viewport, `/app/settings/payments/import?period=${months.recent}`, 'payment-import', assertPaymentImport);
    await recordPage(page, events, viewport, '/app/parking', `parking-${expectedParkingState}`, assertParking);
    if (expectedParkingState === 'populated') {
      await recordPage(page, events, viewport, `/app/parking/month/${months.recent}`, 'parking-month-paid', (candidate, label) => assertMonth(candidate, label, true));
      await recordPage(page, events, viewport, `/app/parking/month/${months.older}`, 'parking-month-open', (candidate, label) => assertMonth(candidate, label, false));
    }
    await context.close();
    activePage = undefined;
  }
  if (artifactDir) writeFileSync(join(artifactDir, 'report.json'), `${JSON.stringify(report, null, 2)}\n`);
  process.stdout.write(`  ✓ Einstellungen/Parkplatz · ${expectedParkingState} · ${report.length} Headless-Prüfungen\n`);
} catch (error) {
  if (artifactDir) {
    writeFileSync(join(artifactDir, 'failure.txt'), `${error instanceof Error ? error.stack || error.message : String(error)}\n`);
    if (activePage) {
      try { await activePage.screenshot({ path: join(artifactDir, 'failure.png'), fullPage: true }); } catch {}
    }
    writeFileSync(join(artifactDir, 'report.json'), `${JSON.stringify(report, null, 2)}\n`);
  }
  console.error(error instanceof Error ? error.stack || error.message : String(error));
  process.exitCode = 1;
} finally {
  await browser.close();
}

function rollingMonths(now = new Date()) {
  const parts = new Intl.DateTimeFormat('en-CA', { timeZone: 'Europe/Vienna', year: 'numeric', month: '2-digit' })
    .formatToParts(now)
    .reduce((out, part) => ({ ...out, [part.type]: part.value }), {});
  const anchor = new Date(Date.UTC(Number(parts.year), Number(parts.month) - 1, 15));
  const key = (offset) => {
    const date = new Date(Date.UTC(anchor.getUTCFullYear(), anchor.getUTCMonth() + offset, 15));
    return `${date.getUTCFullYear()}-${String(date.getUTCMonth() + 1).padStart(2, '0')}`;
  };
  return { recent: key(-1), older: key(-2) };
}

function sampleAt(month, hour) {
  return `${month}-25T${String(hour).padStart(2, '0')}:00:00Z`;
}

function writePopulatedFixture(target) {
  const { recent, older } = rollingMonths();
  const oldestYear = Number(older.slice(0, 4)) - 1;
  const fixture = {
    tenants: {
      demo: {
        settings: {
          grid_fee_eur_per_kwh: 0.1,
          base_fee_eur: 3,
          tariffs: [{
            effective_from: `${oldestYear}-01-01`,
            grid_fee_eur_per_kwh: 0.1,
            base_fee_eur: 3,
            surplus_rate_eur_per_kwh: 0.08,
          }],
          charging: { enabled: false, shadow_mode: true },
        },
        months: {
          [older]: { paid: false },
          [recent]: {
            paid: true,
            paid_at: new Date().toISOString(),
            paid_by: 'admin@example.com',
            payment_method: 'Überweisung',
            payment_reference: 'QA-ZAHLUNG',
          },
        },
        energy_samples: [
          { at: sampleAt(older, 10), value: 100 },
          { at: sampleAt(older, 12), value: 102 },
          { at: sampleAt(recent, 10), value: 102 },
          { at: sampleAt(recent, 12), value: 105 },
        ],
        price_samples: [
          { at: sampleAt(older, 10), value: 0.2 },
          { at: sampleAt(older, 11), value: 0.4 },
          { at: sampleAt(recent, 10), value: 0.18 },
          { at: sampleAt(recent, 11), value: 0.32 },
        ],
      },
    },
  };
  mkdirSync(dirname(target), { recursive: true });
  writeFileSync(target, `${JSON.stringify(fixture, null, 2)}\n`);
}

function camtXML(reference) {
  const escaped = reference.replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;');
  return `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.08">
  <BkToCstmrStmt><Stmt><Id>OTHER</Id><Acct><Ownr><Nm>Fremder Kontoinhaber</Nm></Ownr></Acct>
    <Ntry><NtryRef>NTR-SECRET</NtryRef><Amt Ccy="EUR">42.00</Amt><CdtDbtInd>CRDT</CdtDbtInd><BookgDt><Dt>${rollingMonths().recent}-08</Dt></BookgDt>
      <NtryDtls><TxDtls><Refs><EndToEndId>${escaped}</EndToEndId></Refs>
        <RltdPties><Dbtr><Nm>Max Geheim</Nm></Dbtr><DbtrAcct><Id><IBAN>AT483200000012345864</IBAN></Id></DbtrAcct></RltdPties>
        <RmtInf><Ustrd>Zahlung ${escaped}</Ustrd></RmtInf>
      </TxDtls></NtryDtls>
    </Ntry>
  </Stmt></BkToCstmrStmt>
</Document>`;
}
