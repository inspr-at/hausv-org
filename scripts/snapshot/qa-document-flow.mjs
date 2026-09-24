#!/usr/bin/env node
// Deterministic browser QA for the complete document and e-invoice lifecycle.

import { mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { chromium } from 'playwright';

const baseURL = process.argv[2];
if (!baseURL) {
  console.error('usage: qa-document-flow.mjs <baseURL>');
  process.exit(1);
}

const artifactDir = process.env.HV_QA_ARTIFACT_DIR?.trim();
const screenshotDir = process.env.HV_QA_SCREENSHOT_DIR?.trim()
  || (artifactDir ? join(artifactDir, 'document-flow-screenshots') : '');
const screenshotPrefix = process.env.HV_QA_SCREENSHOT_PREFIX?.trim() || 'document-flow';
if (artifactDir) mkdirSync(artifactDir, { recursive: true });
if (screenshotDir) mkdirSync(screenshotDir, { recursive: true });

const sizes = [
  { width: 320, height: 568 },
  { width: 390, height: 844 },
  { width: 768, height: 1024 },
  { width: 1024, height: 768 },
  { width: 1440, height: 900 },
];
const publicTitle = 'QA Dokumentenfluss – Hausordnung';
const ownerTitle = 'QA Eigentümerprotokoll mit einem bewusst sehr langen und aussagekräftigen Titel';
const publicFilename = 'qa-dokumentenfluss-hausordnung.png';
const ownerFilenameV1 = 'qa-eigentuemerprotokoll-v1.png';
const ownerFilenameV2 = 'qa-eigentuemerprotokoll-v2.png';
const invoiceFilename = 'qa-ebinterface-rechnung.xml';
const pngV1 = Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=', 'base64');
const pngV2 = Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAIAAACQd1PeAAAADElEQVR42mP4z8AAAAMBAQDJ/pLvAAAAAElFTkSuQmCC', 'base64');
const invoiceXML = readFileSync(new URL('../../internal/integrations/testdata/ebinterface-6p0.xml', import.meta.url));
const contexts = new Set();
const browserEvents = [];

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
    const url = new URL(request.url());
    const detail = request.failure()?.errorText || 'unknown';
    // Chromium hands Content-Disposition downloads from the renderer to the
    // download manager and reports the renderer request itself as aborted.
    if (url.pathname.endsWith('/download') && detail === 'net::ERR_ABORTED') return;
    browserEvents.push(`requestfailed ${request.method()} ${url.pathname}: ${detail}`);
  });
}

const browser = await chromium.launch({
  headless: true,
  args: ['--no-proxy-server'],
});

async function newContext(size = sizes.at(-1)) {
  const context = await browser.newContext({
    viewport: size,
    deviceScaleFactor: 1,
    locale: 'de-AT',
    timezoneId: 'Europe/Vienna',
    reducedMotion: 'reduce',
    acceptDownloads: true,
  });
  contexts.add(context);
  context.on('page', watchPage);
  return context;
}

async function localLogin(context, email) {
  const page = await context.newPage();
  await page.goto(`${baseURL}/`, { waitUntil: 'networkidle' });
  const emailDetails = page.locator('details:has(form[action$="/auth/request"])');
  if (await emailDetails.count()) await emailDetails.evaluate((node) => { node.open = true; });
  await page.locator('input[name="email"]').fill(email);
  await page.locator('form[action$="/auth/request"] button[type="submit"]').click();
  const devLink = page.locator('a.dev-link');
  await devLink.waitFor({ state: 'visible', timeout: 10_000 });
  const href = await devLink.getAttribute('href');
  if (!href) fail(`Kein lokaler Anmeldelink für ${email}`);
  const target = new URL(href, baseURL);
  const localOrigin = new URL(baseURL);
  target.protocol = localOrigin.protocol;
  target.hostname = localOrigin.hostname;
  target.port = localOrigin.port;
  await page.goto(target.href, { waitUntil: 'networkidle' });
  if (!new URL(page.url()).pathname.endsWith('/app')) fail(`Anmeldung für ${email} endete auf ${page.url()}`);
  return page;
}

async function closeContext(context) {
  contexts.delete(context);
  await context.close();
}

async function screenshot(page, name, fullPage = true) {
  if (!screenshotDir) return;
  await page.evaluate(() => {
    if (document.activeElement instanceof HTMLElement) document.activeElement.blur();
    window.scrollTo(0, 0);
  });
  await page.screenshot({ path: join(screenshotDir, `${screenshotPrefix}-${name}.png`), fullPage });
}

async function assertGeometry(page, label, rootSelector = 'main') {
  const result = await page.evaluate((selector) => {
    const root = document.querySelector(selector);
    const viewport = document.documentElement.clientWidth;
    const visible = (node) => Boolean(node.getClientRects().length);
    const controls = [...(root?.querySelectorAll('button, a.button, summary, input, select') || [])]
      .filter(visible)
      .map((node) => {
        const rect = node.getBoundingClientRect();
        return {
          name: (node.getAttribute('aria-label') || node.textContent || node.getAttribute('name') || '').trim().replace(/\s+/g, ' ').slice(0, 90),
          width: Math.round(rect.width * 10) / 10,
          height: Math.round(rect.height * 10) / 10,
          primary: node.matches('button, a.button, input, select'),
        };
      });
    return {
      viewport,
      bodyWidth: document.body.scrollWidth,
      rootWidth: document.documentElement.scrollWidth,
      undersized: controls.filter((control) => control.width < 23.5 || control.height < 23.5),
      shortPrimary: viewport <= 760
        ? controls.filter((control) => control.primary && control.height < 43.5)
        : [],
    };
  }, rootSelector);
  if (result.bodyWidth > result.viewport + 1 || result.rootWidth > result.viewport + 1) {
    fail(`${label}: horizontaler Überlauf ${JSON.stringify(result)}`);
  }
  if (result.undersized.length) fail(`${label}: controls below 24 px ${JSON.stringify(result.undersized)}`);
  if (result.shortPrimary.length) fail(`${label}: primary controls below 44 px ${JSON.stringify(result.shortPrimary)}`);
}

async function assertDialogFooterClear(page, label) {
  const result = await page.locator('#document-upload').evaluate((dialog) => {
    const body = dialog.querySelector('.dialog-body');
    const footer = dialog.querySelector('.dialog-footer');
    const optional = dialog.querySelector('.dialog-optional > summary');
    if (!body || !footer || !optional) return { complete: false };
    body.scrollTop = body.scrollHeight;
    const bodyRect = body.getBoundingClientRect();
    const footerRect = footer.getBoundingClientRect();
    const optionalRect = optional.getBoundingClientRect();
    const overlap = Math.max(0, Math.min(optionalRect.bottom, footerRect.bottom) - Math.max(optionalRect.top, footerRect.top));
    return {
      complete: true,
      bodyEndsBeforeFooter: bodyRect.bottom <= footerRect.top + 1,
      overlap,
      footerHeight: footerRect.height,
      optionalHeight: optionalRect.height,
    };
  });
  if (!result.complete || !result.bodyEndsBeforeFooter || result.overlap > 0 || result.footerHeight < 44 || result.optionalHeight < 44) {
    fail(`${label}: Dialog-Fuß überlagert optionale Felder ${JSON.stringify(result)}`);
  }
}

async function assertCompactDocumentSearch(page) {
  const result = await page.evaluate(() => {
    const input = document.querySelector('#document-search');
    const select = document.querySelector('#document-sort');
    const submit = document.querySelector('.document-toolbar button[type="submit"]');
    return {
      placeholder: input?.getAttribute('placeholder') || '',
      selected: select?.selectedOptions[0]?.textContent?.trim() || '',
      selectWidth: select?.getBoundingClientRect().width || 0,
      submit: submit?.textContent?.trim() || '',
    };
  });
  if (result.placeholder !== 'Dokument suchen' || result.selected !== 'Neueste' || result.selectWidth < 130 || result.submit !== 'Anzeigen') {
    fail(`Dokumentsuche ist bei 320 px gekürzt oder unklar ${JSON.stringify(result)}`);
  }
}

function documentRow(page, title) {
  return page.locator('.document-row').filter({ hasText: title }).first();
}

async function uploadDocument(page, { title, category, visibility, filename, buffer }) {
  await page.goto(`${baseURL}/app/dokumente`, { waitUntil: 'networkidle' });
  await page.getByRole('button', { name: 'Dokument hochladen' }).first().click();
  const form = page.locator('#document-upload form');
  await form.locator('input[name="title"]').fill(title);
  await form.locator('select[name="category"]').selectOption({ label: category });
  await form.locator('select[name="visibility"]').selectOption({ label: visibility });
  await form.locator('input[name="document"]').setInputFiles({ name: filename, mimeType: 'image/png', buffer });
  await Promise.all([
    page.waitForURL((url) => url.pathname.endsWith('/app/dokumente') && url.searchParams.get('doc') === 'uploaded'),
    form.getByRole('button', { name: 'Hochladen', exact: true }).click(),
  ]);
  const row = documentRow(page, title);
  await row.waitFor({ state: 'visible' });
  return row;
}

async function readDownload(download) {
  const stream = await download.createReadStream();
  const chunks = [];
  for await (const chunk of stream) chunks.push(chunk);
  return Buffer.concat(chunks);
}

async function downloadFrom(row, expectedFilename, expectedBytes) {
  const [download] = await Promise.all([
    row.page().waitForEvent('download'),
    row.getByRole('link', { name: 'Herunterladen' }).first().click(),
  ]);
  if (download.suggestedFilename() !== expectedFilename) {
    fail(`Downloadname ${download.suggestedFilename()}, erwartet ${expectedFilename}`);
  }
  const data = await readDownload(download);
  if (!data.equals(expectedBytes)) fail(`Download ${expectedFilename} enthält nicht die unveränderte Fixture`);
}

async function directStatus(context, path) {
  const response = await context.request.get(`${baseURL}${path}`, { maxRedirects: 0 });
  return response.status();
}

async function run() {
  const managerContext = await newContext();
  const manager = await localLogin(managerContext, 'verwalter@example.com');

  let publicRow = await uploadDocument(manager, {
    title: publicTitle,
    category: 'Hausordnung',
    visibility: 'Alle Bewohner',
    filename: publicFilename,
    buffer: pngV1,
  });
  if (!(await publicRow.getByText('Alle Bewohner', { exact: true }).count())) fail('Freigabe „Alle Bewohner“ fehlt');
  const publicPreviewPath = await publicRow.locator('a.document-title').getAttribute('data-lightbox-src');
  const publicDownloadPath = await publicRow.getByRole('link', { name: 'Herunterladen' }).first().getAttribute('href');
  if (!publicPreviewPath || !publicDownloadPath) fail('Vorschau- oder Downloadpfad fehlt');

  await manager.setViewportSize(sizes[1]);
  const previewResponse = manager.waitForResponse((response) => new URL(response.url()).pathname === publicPreviewPath);
  await publicRow.locator('a.document-title').click();
  if ((await previewResponse).status() !== 200) fail('Bildvorschau antwortet nicht mit 200');
  await manager.locator('.attachment-lightbox.open').waitFor({ state: 'visible' });
  await screenshot(manager, 'image-preview-390', false);
  const closePreview = manager.locator('.attachment-lightbox.open').getByRole('button', { name: 'Schließen' });
  if ((await closePreview.boundingBox())?.height < 43.5) fail('Schließen der Bildvorschau ist kleiner als 44 px');
  await closePreview.click();
  await manager.setViewportSize(sizes.at(-1));
  await downloadFrom(publicRow, publicFilename, pngV1);

  let ownerRow = await uploadDocument(manager, {
    title: ownerTitle,
    category: 'Protokoll',
    visibility: 'Nur Eigentümer',
    filename: ownerFilenameV1,
    buffer: pngV1,
  });
  const ownerPreviewPathV1 = await ownerRow.locator('a.document-title').getAttribute('data-lightbox-src');
  const ownerDownloadPathV1 = await ownerRow.getByRole('link', { name: 'Herunterladen' }).first().getAttribute('href');
  if (!ownerPreviewPathV1 || !ownerDownloadPathV1) fail('Eigentümer-Dokument hat keine Dateiaktionen');

  const residentContext = await newContext();
  const resident = await localLogin(residentContext, 'resident@example.com');
  await resident.goto(`${baseURL}/app/dokumente`, { waitUntil: 'networkidle' });
  if (!(await resident.getByText(publicTitle, { exact: true }).count())) fail('Bewohner sieht die allgemeine Unterlage nicht');
  if (await resident.getByText(ownerTitle, { exact: true }).count()) fail('Bewohner sieht Eigentümer-Metadaten');
  if (await directStatus(residentContext, ownerPreviewPathV1) !== 403 || await directStatus(residentContext, ownerDownloadPathV1) !== 403) {
    fail('Bewohner kann Eigentümer-Datei direkt abrufen');
  }

  const ownerContext = await newContext();
  const owner = await localLogin(ownerContext, 'owner@example.com');
  await owner.goto(`${baseURL}/app/dokumente`, { waitUntil: 'networkidle' });
  if (!(await owner.getByText(publicTitle, { exact: true }).count()) || !(await owner.getByText(ownerTitle, { exact: true }).count())) {
    fail('Eigentümer sieht nicht beide freigegebenen Unterlagen');
  }
  await downloadFrom(documentRow(owner, ownerTitle), ownerFilenameV1, pngV1);

  await manager.goto(`${baseURL}/app/dokumente`, { waitUntil: 'networkidle' });
  ownerRow = documentRow(manager, ownerTitle);
  const more = ownerRow.locator('.document-more');
  const moreSummary = more.locator(':scope > summary');
  await moreSummary.focus();
  await moreSummary.press('Enter');
  if (!(await more.evaluate((node) => node.open))) fail('Mehr öffnet nicht per Tastatur');
  await more.getByRole('button', { name: 'Neue Version hochladen' }).click();
  const replaceDialog = manager.locator('dialog[open][id^="document-replace-"]');
  await replaceDialog.locator('input[name="document"]').setInputFiles({ name: ownerFilenameV2, mimeType: 'image/png', buffer: pngV2 });
  await Promise.all([
    manager.waitForURL((url) => url.pathname.endsWith('/app/dokumente') && url.searchParams.get('doc') === 'replaced'),
    replaceDialog.getByRole('button', { name: 'Neue Version speichern' }).click(),
  ]);
  ownerRow = documentRow(manager, ownerTitle);
  if (!(await ownerRow.getByText('Version 2', { exact: true }).count())) fail('Neue Dokumentversion fehlt');
  await ownerRow.locator('.document-more > summary').click();
  const versionHistory = ownerRow.locator('.versions');
  if (!(await versionHistory.getByText(/Version 1/).count()) || !(await versionHistory.getByText(ownerFilenameV1, { exact: false }).count())) {
    fail('Frühere Fassung fehlt im Versionsverlauf');
  }
  const [oldVersionDownload] = await Promise.all([
    manager.waitForEvent('download'),
    versionHistory.getByRole('link', { name: 'Herunterladen' }).click(),
  ]);
  if (!(await readDownload(oldVersionDownload)).equals(pngV1)) fail('Versionsverlauf liefert nicht Version 1');
  await downloadFrom(ownerRow, ownerFilenameV2, pngV2);

  await manager.goto(`${baseURL}/app/dokumente/rechnungen/import`, { waitUntil: 'networkidle' });
  await manager.locator('input[name="invoice_file"]').setInputFiles({ name: invoiceFilename, mimeType: 'application/xml', buffer: invoiceXML });
  if ((await manager.locator('.invoice-file-control > span').textContent())?.trim() !== invoiceFilename) {
    fail('E-Rechnungs-Dateiauswahl bestätigt den gewählten Dateinamen nicht');
  }
  await Promise.all([
    manager.waitForURL((url) => url.pathname.endsWith('/app/dokumente/rechnungen/import') && url.searchParams.has('preview')),
    manager.getByRole('button', { name: 'Vorschau erstellen' }).click(),
  ]);
  const invoicePreviewURL = manager.url();
  for (const text of ['RE-2026-0006', '99,90 €', 'Hausservice Beispiel e.U.', 'demo', '09.07.2026', '23.07.2026']) {
    if (!(await manager.getByText(text, { exact: true }).count())) fail(`Rechnungsvorschau fehlt: ${text}`);
  }

  for (const size of sizes) {
    for (const [role, page] of [['manager', manager], ['resident', resident], ['owner', owner]]) {
      await page.setViewportSize(size);
      await page.goto(`${baseURL}/app/dokumente`, { waitUntil: 'networkidle' });
      await assertGeometry(page, `Dokumente ${role} ${size.width}px`);
      if (role === 'manager' && size.width === 320) await assertCompactDocumentSearch(page);
      await screenshot(page, `library-${role}-${size.width}`);
    }

    await manager.setViewportSize(size);
    await manager.goto(invoicePreviewURL, { waitUntil: 'networkidle' });
    await assertGeometry(manager, `E-Rechnung Vorschau ${size.width}px`);
    await screenshot(manager, `invoice-preview-${size.width}`);

    await manager.goto(`${baseURL}/app/dokumente/rechnungen/import`, { waitUntil: 'networkidle' });
    await assertGeometry(manager, `E-Rechnung Auswahl ${size.width}px`);
    await screenshot(manager, `invoice-upload-${size.width}`);

    await manager.goto(`${baseURL}/app/dokumente`, { waitUntil: 'networkidle' });
    await manager.getByRole('button', { name: 'Dokument hochladen' }).first().click();
    await assertGeometry(manager, `Dokument-Dialog ${size.width}px`, '#document-upload');
    await assertDialogFooterClear(manager, `Dokument-Dialog ${size.width}px`);
    await screenshot(manager, `upload-dialog-${size.width}`);
  }

  await manager.setViewportSize(sizes.at(-1));
  await manager.goto(invoicePreviewURL, { waitUntil: 'networkidle' });
  await Promise.all([
    manager.waitForURL((url) => url.pathname.endsWith('/app/dokumente') && url.searchParams.get('doc') === 'invoice-imported'),
    manager.getByRole('button', { name: 'Geschützt ablegen' }).click(),
  ]);
  if (!(await manager.getByText('E-Rechnung geprüft und geschützt abgelegt.', { exact: true }).count())) fail('Ablagebestätigung fehlt');
  const invoiceRow = manager.locator('.document-row').filter({ hasText: 'RE-2026-0006' }).first();
  await invoiceRow.waitFor({ state: 'visible' });
  if (!(await invoiceRow.getByText('Nur Verwaltung', { exact: true }).count())) fail('E-Rechnung ist nicht verwaltungsintern');
  const invoiceDownloadPath = await invoiceRow.getByRole('link', { name: 'Herunterladen' }).first().getAttribute('href');
  if (!invoiceDownloadPath) fail('E-Rechnung hat keinen Downloadpfad');
  await downloadFrom(invoiceRow, 'ebinterface-re20260006.xml', invoiceXML);

  await resident.goto(`${baseURL}/app/dokumente`, { waitUntil: 'networkidle' });
  if (await resident.getByText(/RE-2026-0006/).count()) fail('Bewohner sieht E-Rechnungs-Metadaten');
  if (await directStatus(residentContext, invoiceDownloadPath) !== 403) fail('Bewohner kann E-Rechnung direkt herunterladen');

  await owner.goto(`${baseURL}/app/dokumente`, { waitUntil: 'networkidle' });
  if (await owner.getByText(/RE-2026-0006/).count()) fail('Eigentümer sieht verwaltungsinterne E-Rechnung');
  if (await directStatus(ownerContext, invoiceDownloadPath) !== 403) fail('Eigentümer kann E-Rechnung direkt herunterladen');

  if (browserEvents.length) fail(`Browserfehler:\n${browserEvents.join('\n')}`);
  process.stdout.write(`  ✓ Dokumente · Upload/Freigabe/Vorschau/Download/Version · Bewohner/Eigentümer/Verwaltung · ${sizes.map((size) => size.width).join('/')}px\n`);
  process.stdout.write('  ✓ E-Rechnung · Vorschau → geschützte Ablage → verwaltungsinternes Original\n');

  await closeContext(ownerContext);
  await closeContext(residentContext);
  await closeContext(managerContext);
}

try {
  await run();
} catch (error) {
  if (artifactDir) writeFileSync(join(artifactDir, 'document-flow-failure.txt'), `${error?.stack || error}\n`);
  console.error(error?.stack || error);
  process.exitCode = 1;
} finally {
  for (const context of [...contexts]) {
    try { await context.close(); } catch {}
  }
  await browser.close();
}
