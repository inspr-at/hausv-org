#!/usr/bin/env node
// Valorisation demo: preview, approval, archived PDF and owner denial. Headless, against one local rig.
//   node qa-valorisation.mjs http://localhost:8309 /path/to/artifacts

import { existsSync, mkdirSync, writeFileSync } from 'node:fs';
import { execFileSync } from 'node:child_process';
import { chromium } from 'playwright';

const baseURL = process.argv[2];
const artifactDir = process.argv[3] || '';
if (!baseURL) {
  console.error('usage: qa-valorisation.mjs <baseURL> [artifact-dir]');
  process.exit(1);
}
const parsedBaseURL = new URL(baseURL);
if (!['localhost', '127.0.0.1', '::1'].includes(parsedBaseURL.hostname)) {
  throw new Error('qa-valorisation only runs against localhost');
}
if (artifactDir) mkdirSync(artifactDir, { recursive: true });

const executableCandidates = [
  process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH,
  '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
  '/Applications/Chromium.app/Contents/MacOS/Chromium',
].filter(Boolean);
const executablePath = executableCandidates.find((path) => existsSync(path));
const browser = await chromium.launch({
  headless: true,
  args: ['--no-proxy-server'],
  ...(executablePath ? { executablePath } : {}),
});

function fail(message) {
  throw new Error(message);
}

async function login(email) {
  const context = await browser.newContext({ locale: 'de-AT', timezoneId: 'Europe/Vienna' });
  const page = await context.newPage();
  await page.goto(`${baseURL}/`, { waitUntil: 'networkidle' });
  const details = page.locator('details:has(form[action$="/auth/request"])');
  if (await details.count()) await details.evaluate((node) => { node.open = true; });
  await page.locator('input[name="email"]').fill(email);
  await page.locator('form[action$="/auth/request"] button[type="submit"]').click();
  const devLink = page.locator('a.dev-link');
  await devLink.waitFor({ state: 'visible', timeout: 15000 });
  const href = await devLink.getAttribute('href');
  if (!href) fail(`Kein Anmeldelink für ${email}`);
  const target = new URL(href, baseURL);
  target.protocol = parsedBaseURL.protocol;
  target.hostname = parsedBaseURL.hostname;
  target.port = parsedBaseURL.port;
  await page.goto(target.href, { waitUntil: 'networkidle' });
  if (!page.url().includes('/app')) fail(`Anmeldung für ${email} endete auf ${page.url()}`);
  await page.close();
  return context;
}

async function shot(page, name) {
  if (!artifactDir) return;
  await page.evaluate(() => window.scrollTo(0, 0));
  await page.screenshot({ path: `${artifactDir}/${name}.png`, fullPage: true, animations: 'disabled' });
  process.stdout.write(`  screenshot ${artifactDir}/${name}.png\n`);
}

async function switchHouse(page, name) {
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.locator('#context-property > summary').click();
  await page.locator('#context-property-search').fill(name);
  await page.getByRole('option').filter({ hasText: name }).first().click();
  await page.waitForLoadState('networkidle');
}

async function checkLetter(context, href, runID, name, requiresMRGNotice = false) {
  const response = await context.request.get(new URL(href, baseURL).href, { headers: { Connection: 'close' } });
  const raw = await response.body();
  if (response.status() !== 200 || !raw.subarray(0, 5).equals(Buffer.from('%PDF-'))) fail(`${name}: PDF fehlt`);
  if (!/^inline;/.test(response.headers()['content-disposition'] || '')) fail(`${name}: Schreiben öffnet nicht inline`);
  // Requires Poppler; check extracted text, including every page footer.
  const text = execFileSync('pdftotext', ['-layout', '-', '-'], { input: raw, encoding: 'utf8' });
  for (const forbidden of ['Lauf', 'Referenz', 'SHA-256', runID.slice(0, 16), '260924101853.0.0', 'Sehr geehrte Damen und Herren,']) {
    if (text.includes(forbidden)) fail(`${name}: interne Prüfdaten im Schreiben (${forbidden})`);
  }
  if (!text.includes('Anpassung des Hauptmietzinses')) fail(`${name}: Brieftext fehlt`);
  const letterText = text.replace(/\s+/g, ' ');
  const noBackPayment = 'Der höhere Hauptmietzins ist erst ab diesem Zinstermin zu bezahlen; für die davorliegenden Monate wird keine Nachzahlung verlangt.';
  for (const expected of ['VPI 2020, Basis September 2024: 123,6', 'Auslösemonat Dezember 2025', 'Anker: September 2024', '5,02 %', '+2,91 %', 'Indexwerte laut Statistik Austria, Stand 24.09.2026', `Guten Tag ${requiresMRGNotice ? 'Lukas Steiner' : 'Eva Huber'},`, 'Für Rückfragen:', 'Mit freundlichen Grüßen', ...(requiresMRGNotice ? ['§ 16 Abs 9 MRG', 'zugeht', '14 Tage', noBackPayment] : ['+3,28 %'])]) {
    if (!letterText.includes(expected)) fail(`${name}: ${expected} fehlt`);
  }
  if (!requiresMRGNotice && letterText.includes(noBackPayment)) fail(`${name}: MRG-Nachzahlungshinweis in Teilanwendung`);
  if (requiresMRGNotice && !text.split('\f')[1]?.replace(/\s+/g, ' ').includes(noBackPayment)) fail(`${name}: Nachzahlungshinweis fehlt auf Seite 2`);
  if (/\d{4}-\d{2}|\d+,\d{3,} %|Wertsicherung · Janusbergweg 123 · Janusbergweg 123/.test(text)) fail(`${name}: technische oder doppelte Briefangaben`);
  if (text.split('\f').filter(page => page.trim()).length !== 2) fail(`${name}: erwartet zwei Briefseiten`);
  if (artifactDir) {
    writeFileSync(`${artifactDir}/${name}.pdf`, raw);
    writeFileSync(`${artifactDir}/${name}.txt`, text);
    writeFileSync(`${artifactDir}/${name}-headers.json`, JSON.stringify({ status: response.status(), contentDisposition: response.headers()['content-disposition'], contentType: response.headers()['content-type'] }, null, 2));
  }
}

try {
  const manager = await login('vera.verwalter@musterstadt.example');
  const page = await manager.newPage();
  const route = `${baseURL}/app/settings/valorisation`;
  // Runtime reference-data controls: use the same signed-in admin session.
  // Fetching is covered with injected fixture clients in Go; this oracle never
  // contacts the live OGD service.
  await page.goto(`${baseURL}/app/verwaltung/wertsicherung`, { waitUntil: 'networkidle' });
  const indexCard = page.locator('[aria-labelledby="index-refresh-title"]');
  if (!(await indexCard.getByRole('button', { name: 'VPI aktualisieren', exact: true }).isVisible())) fail('VPI-Adminaktion fehlt');
  if (!(await indexCard.innerText()).includes('Datenquelle: Statistik Austria, CC BY 4.0')) fail('VPI-Datenquelle fehlt');
  for (const width of [1440, 390]) {
    await page.setViewportSize({ width, height: 900 });
    const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
    if (overflow > 1) fail(`VPI ${width}px: Überlauf ${overflow}px`);
    if (artifactDir) await indexCard.screenshot({ path: `${artifactDir}/index-refresh-${width}.png` });
  }
  await page.setViewportSize({ width: 1440, height: 900 });
  let response = await page.goto(`${route}?new=1&preview=1&effective_on=2026-04-01&house=janusbergweg-123`, { waitUntil: 'networkidle' });
  if (response?.status() !== 200) fail(`Vorschau HTTP ${response?.status()}`);
  const preview = page.locator('article[data-run-id=""]');
  for (const group of ['ready', 'unchanged', 'exception']) {
    if (await preview.locator(`[data-group="${group}"]`).count() !== 1) fail(`Gruppe ${group} fehlt`);
  }
  if (await preview.locator('[data-group="ready"] > .vr-item').count() < 6) fail('Weniger als sechs bereite Verträge');
  const top1 = preview.locator('[data-lease-id="lease-top-1"]');
  const staffel = preview.locator('[data-group="ready"] > [data-lease-id="lease-top-9"]');
  if (await staffel.count() !== 1) fail('Top 9: Staffelmietzins ist nicht bereit');
  await staffel.locator(':scope > summary').click();
  for (const value of ['Staffelmietzins laut Vertrag', '1.150,00', '1.081,54', '68,46']) {
    if (!(await staffel.innerText()).includes(value)) fail(`Staffelvorschau: ${value} fehlt`);
  }
  await top1.locator(':scope > summary').click();
  const used = top1.getByRole('table', { name: 'Verwendete Indexwerte', exact: true });
  if (await used.locator('tbody tr').count() !== 5) fail('E1: erwartete Basis, Auslöser und drei Jahresmittel fehlen');
  if (await top1.getByRole('table', { name: 'Geprüfte Monatsreihe und Jahresmittel' }).isVisible()) fail('Volle Indexreihe ist nicht eingeklappt');
  await top1.locator('summary').filter({ hasText: 'Alle Indexwerte anzeigen' }).click();
  if (await top1.getByRole('table', { name: 'Geprüfte Monatsreihe und Jahresmittel' }).locator('tbody tr').count() <= 5) fail('Volle Indexreihe fehlt');
  const body = await preview.innerText();
  for (const value of ['1.040,28', '1.017,35', '21.04.2026', '05.05.2026', 'Top\u00a01 · Eva Huber', 'Schwelle 5 % überschritten: +5,02 %', '+3,55412 %']) {
    if (!body.includes(value)) fail(`Vorschau: ${value} fehlt`);
  }
  if (/\btop-\d|\d+\/\d+ %|\d+\.\d+ %|percent|Kurve exakt/.test(body)) fail('Technische Zahlen oder IDs in der Vorschau');
  await switchHouse(page, 'Musterstraße 12');
  await page.goto(`${baseURL}/musterstrasse-12/app/settings/valorisation`, { waitUntil: 'networkidle' });
  const mrgStaffel = page.locator('[data-group="ready"] > [data-lease-id="m12-lease-top-8"]').first();
  if (await mrgStaffel.count() !== 1) fail('Musterstraße Top 8: feste Staffel ist nicht bereit');
  await mrgStaffel.locator(':scope > summary').click();
  for (const value of ['Julian Eder', '910,00', '928,20', 'Staffelmietzins laut Vertrag']) {
    if (!(await mrgStaffel.innerText()).includes(value)) fail(`Musterstraße Staffel: ${value} fehlt`);
  }
  if ((await mrgStaffel.innerText()).includes('Veröffentlichungsnachweis fehlt')) fail('Feste Staffel verlangt einen monatlichen Index');
  for (const width of [1440, 390]) {
    await page.setViewportSize({ width, height: 900 });
    if (await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth) > 1) fail(`Musterstraße Staffel: Überlauf bei ${width}px`);
    if (artifactDir) await mrgStaffel.screenshot({ path: `${artifactDir}/mrg-staffel-top8-${width}.png` });
  }
  // The seeded April draft is reviewable even when the system clock is later.
  await switchHouse(page, 'Janusbergweg 123');
  await page.goto(`${baseURL}/janusbergweg-123/app/settings/valorisation`, { waitUntil: 'networkidle' });
  const first = page.locator('article[data-run-id]').first();
  const runID = await first.getAttribute('data-run-id');
  if (!runID) fail('Demo-Lauf fehlt');
  const draftPDF = first.locator('[data-group="ready"] a[href$="/pdf"]').first();
  await checkLetter(manager, await draftPDF.getAttribute('href'), runID, 'valorisation-letter-draft');
  await checkLetter(manager, await first.locator('[data-lease-id="lease-top-2"] a[href$="/pdf"]').first().getAttribute('href'), runID, 'valorisation-letter-draft-mrg', true);
  const approveButton = first.getByRole('button', { name: 'Freigeben und archivieren' });
  if (!(await approveButton.isDisabled()) || !(await approveButton.getAttribute('class')).includes('ghost')) fail('Freigabe bei offenen Ausnahmen nicht deaktiviert');
  const issueID = await approveButton.getAttribute('aria-describedby');
  if (!issueID || !(await page.locator(`#${issueID}`).innerText()).includes('3 Ausnahmen sind noch offen.')) fail('Anzahl offener Ausnahmen fehlt');
  const approveAction = await approveButton.locator('..').getAttribute('action');
  const blocked = await manager.request.post(new URL(approveAction, baseURL).href, { headers: { Origin: baseURL }, maxRedirects: 0 });
  if (blocked.status() !== 303) fail(`Freigabe mit offenen Ausnahmen: HTTP ${blocked.status()}`);
  await page.goto(new URL(blocked.headers().location, baseURL).href, { waitUntil: 'networkidle' });
  const notice = await page.locator(`#run-${runID} [role="status"]`).innerText();
  if (notice !== '3 Ausnahmen sind noch offen. Bitte zuerst prüfen oder mit Begründung ausschließen.') fail(`Freigabehinweis: ${notice}`);
  if ((await first.innerText()).split('3 Ausnahmen sind noch offen.').length !== 2) fail('Freigabehinweis doppelt');
  if (!(await first.locator('.vr-actions').evaluate(node => node.previousElementSibling?.getAttribute('role') === 'status'))) fail('Freigabehinweis nicht bei der Aktion');
  if (!(await first.innerText()).includes('Erstellt von Vera Verwalter')) fail('Anzeigename im Fuß fehlt');
  if (await first.getByText('Prüfsumme', { exact: true }).count()) fail('Leere Prüfsummenbeschriftung');
  const audit = first.locator('details.vr-meta');
  if (await audit.count() !== 1 || await audit.evaluate(node => node.open)) fail('Prüfdaten fehlen oder sind offen');
  if (!(await audit.textContent()).includes('Index-Datenversion 260924101853.0.0')) fail('Exakte Index-Datenversion fehlt in Prüfdaten');
  const deadline = await first.locator(':scope > .vr-deadline').first().innerText();
  if (!deadline.includes('spätestens am 21.04.2026 zugehen') || !deadline.includes('Postlaufzeit einplanen.')) fail(`Zugangsfrist: ${deadline}`);
  for (const width of [1440, 390]) {
    await page.setViewportSize({ width, height: 900 });
    await first.locator('.vr-item').first().evaluate(node => { node.open = true; });
    const chevron = first.locator('.vr-item').first().locator(':scope > summary .vr-item-chevron');
    if (await chevron.locator('svg').count() !== 1) fail('Produktchevron fehlt');
    await page.waitForTimeout(200);
    if (await chevron.evaluate(node => getComputedStyle(node).transform) !== 'matrix(0, 1, -1, 0, 0, 0)') fail('Offener Chevron nicht gedreht');
    await shot(page, `valorisation-${width}`);
    const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
    if (overflow > 1) fail(`${width}px: Überlauf ${overflow}px`);
    await first.locator('[data-group="exception"] > .vr-item').evaluateAll(nodes => nodes.forEach(node => { node.open = true; }));
    await shot(page, `valorisation-open-exceptions-${width}`);
  }
  const itemIDs = await first.locator('[data-group="exception"] > .vr-item').evaluateAll(nodes => nodes.map(node => node.id.slice(5)));
  for (const itemID of itemIDs) {
    const detail = page.locator(`#item-${itemID}`);
    await detail.evaluate(node => { node.open = true; });
    await detail.locator('.vr-manual > summary').click();
    await detail.getByLabel('Begründung für Ausschluss').fill('Browserprüfung: gesonderte Vertragsprüfung');
    await detail.getByRole('button', { name: 'Ausschließen', exact: true }).click();
    await page.waitForLoadState('networkidle');
  }
  if (!(await approveButton.isEnabled())) fail('Freigabe nach begründetem Ausschluss noch gesperrt');
  await page.locator(`#run-${runID}`).getByRole('button', { name: 'Freigeben und archivieren' }).click();
  await page.waitForLoadState('networkidle');
  const approved = page.locator(`#run-${runID}`);
  if (!(await approved.innerText()).includes('Freigegeben')) fail('Lauf nicht freigegeben');
  await approved.locator('.vr-item').evaluateAll(nodes => nodes.forEach(node => { node.open = false; }));
  const pdfLink = approved.getByRole('link', { name: 'Schreiben (PDF)', exact: true }).first();
  if (!(await pdfLink.isVisible())) fail('Schreiben ist bei eingeklappter Zeile nicht erreichbar');
  if (await approved.locator('input[name="reason"]').isVisible()) fail('Stornogrund wird ohne Bestätigungsschritt angezeigt');
  await approved.locator('.vr-cancel > summary').click();
  const cancelReason = approved.getByLabel('Stornogrund');
  if (!(await cancelReason.isVisible()) || !(await cancelReason.evaluate(node => node.required && node.validity.valueMissing))) fail('Stornierung benötigt keinen sichtbaren Pflichtgrund');
  await approved.locator('.vr-cancel > summary').click();
  const href = await pdfLink.getAttribute('href');
  if (!href) fail('PDF-Link fehlt');
  await checkLetter(manager, href, runID, 'valorisation-letter-approved');
  await checkLetter(manager, await approved.locator('[data-lease-id="lease-top-2"] a[href$="/pdf"]').first().getAttribute('href'), runID, 'valorisation-letter-approved-mrg', true);
  for (const width of [1440, 390]) {
    await page.setViewportSize({ width, height: 900 });
    await shot(page, `valorisation-approved-${width}`);
  }
  // Structured schedule editing must survive a save, without interpreting prose.
  await page.goto(`${baseURL}/app/settings/building/units/top-9/lease?bearbeiten=1`, { waitUntil: 'networkidle' });
  const editor = page.locator('[data-staffel-editor]');
  const rows = editor.locator('[data-staffel-row]');
  if (!(await editor.isVisible()) || await rows.count() !== 2) fail('Staffel-Editor fehlt');
  await editor.getByRole('button', { name: 'Stufe hinzufügen' }).click();
  if (await rows.count() !== 3) fail('Staffelstufe nicht ergänzt');
  await rows.last().getByRole('button', { name: 'Stufe entfernen' }).click();
  await rows.nth(1).locator('[name="staffel_value"]').fill('2,5');
  for (const width of [1440, 390]) {
    await page.setViewportSize({ width, height: 900 });
    await shot(page, `staffel-editor-${width}`);
    if (await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth) > 1) fail(`Staffel-Editor: Überlauf bei ${width}px`);
  }
  await page.getByRole('button', { name: 'Speichern', exact: true }).click();
  await page.waitForLoadState('networkidle');
  if (!(await page.locator('body').innerText()).includes('2,5 % auf den vorigen Vertragsbetrag')) fail('Staffelstufe wurde nicht gespeichert');
  await shot(page, 'staffel-summary-390');
  await manager.close();
  const owner = await login('alina.eigentuemer@musterstadt.example');
  const ownerPage = await owner.newPage();
  response = await ownerPage.goto(route, { waitUntil: 'networkidle' });
  if (response?.status() !== 403) fail(`Eigentümer HTTP ${response?.status()}, erwartet 403`);
  await shot(ownerPage, 'valorisation-owner-denied');
  const deniedRefresh = await owner.request.post(`${baseURL}/app/verwaltung/wertsicherung/refresh`, { headers: { Origin: baseURL, Referer: `${baseURL}/app/verwaltung/wertsicherung` }, maxRedirects: 0 });
  if (deniedRefresh.status() !== 403) fail(`VPI Eigentümer POST ${deniedRefresh.status()}, erwartet 403`);
  await owner.close();
  process.stdout.write('qa-valorisation ok\n');
} finally {
  await browser.close();
}
