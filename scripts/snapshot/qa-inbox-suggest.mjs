#!/usr/bin/env node
// Run through run.sh: it supplies isolated data, a non-demo default tenant and
// AI_BASE_URL/AI_MODEL/AI_API_KEY for the stub owned by this oracle.
// HV_CAPTURE=qa-inbox-suggest.mjs scripts/snapshot/run.sh WORKTREE /tmp/hausv-695 8099
import assert from 'node:assert/strict';
import { existsSync, mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { chromium } from 'playwright';
import { startFakeAI } from './fake-ai.mjs';

const [baseURL, reportDir] = process.argv.slice(2);
assert(baseURL && reportDir, 'usage: qa-inbox-suggest.mjs <baseURL> <report-directory>');
assert.equal(process.env.DEFAULT_TENANT, 'haus-b', 'Use run.sh: default tenant must differ from /demo');
const aiPort = Number(process.env.HV_QA_AI_PORT);
assert(Number.isInteger(aiPort) && aiPort > 0, 'HV_QA_AI_PORT must be supplied by run.sh');
assert.equal(process.env.AI_BASE_URL, `http://127.0.0.1:${aiPort}/v1`);
assert.equal(process.env.AI_MODEL, 'qa-inbox-suggest');
mkdirSync(reportDir, { recursive: true });
const report = { ticket: 'HAUSV-695', tenant: 'demo', defaultTenant: 'haus-b', passed: false, scenarios: [], pollPaths: [], pageErrors: [] };
const fixture = await startFakeAI(aiPort);
let browser, page;

function statusText(text) {
  return page.locator('#vorschlag').getByText(text, { exact: false }).waitFor({ state: 'visible', timeout: 15_000 });
}

async function assertIntact() {
  assert.equal(await page.locator('#vorschlag').count(), 1, '#vorschlag must always exist exactly once');
  assert.deepEqual(await page.evaluate(() => window.__inboxQA.missing), [], '#vorschlag disappeared after a poll');
  for (const path of report.pollPaths) assert.match(path, /^\/demo\/app\/verwaltung\/posteingang\/[^/]+\/vorschlag(?:\?|$)/);
}

async function snapshot(name) {
  await assertIntact();
  await page.screenshot({ path: join(reportDir, `${name}.png`), fullPage: true });
}

async function login() {
  await page.goto(`${baseURL}/demo/`, { waitUntil: 'networkidle' });
  const details = page.locator('details:has(form[action$="/auth/request"])');
  if (await details.count()) await details.evaluate(element => { element.open = true; });
  await page.locator('input[name="email"]').fill('verwalter@example.com');
  await page.locator('form[action$="/auth/request"] button[type="submit"]').click();
  const link = page.locator('a.dev-link');
  await link.waitFor({ state: 'visible' });
  const target = new URL(await link.getAttribute('href'), baseURL);
  target.protocol = new URL(baseURL).protocol;
  target.port = new URL(baseURL).port;
  await page.goto(target.href, { waitUntil: 'networkidle' });
  assert.match(new URL(page.url()).pathname, /^\/demo\/app(?:\/|$)/);
}

async function createOpenCase(name) {
  // Phone-note creation also attempts triage synchronously. Let that attempt
  // fail locally so the real UI offers "Vorschlag anfordern" on an open case.
  fixture.setMode('Fehler', 0);
  await page.goto(`${baseURL}/demo/app/verwaltung/posteingang`, { waitUntil: 'networkidle' });
  await page.locator('details:has(form.phone-panel) > summary').click();
  const form = page.locator('form.phone-panel');
  await form.locator('[name="house"]').selectOption('demo');
  await form.locator('[name="unit"]').fill('Top 1');
  await form.locator('[name="from_name"]').fill('QA Hausverwaltung');
  await form.locator('[name="subject"]').fill(`HAUSV-695 ${name}`);
  await form.locator('[name="body"]').fill('Wasser im Keller. Bitte die Reparatur prüfen.');
  await Promise.all([
    page.waitForURL(/\/demo\/app\/verwaltung\/posteingang\/[^/?]+(?:\?|$)/),
    form.getByRole('button', { name: 'Notiz aufnehmen', exact: true }).click(),
  ]);
  await page.getByRole('button', { name: 'Vorschlag anfordern', exact: true }).waitFor({ state: 'visible' });
  return new URL(page.url()).pathname;
}

async function requestSuggestion(mode, delay) {
  fixture.setMode(mode, delay);
  const requestsBefore = fixture.stats.requests;
  await page.getByRole('button', { name: 'Vorschlag anfordern', exact: true }).click();
  await statusText('Vorschlag wird erstellt');
  const pollURL = await page.locator('#vorschlag').getAttribute('hx-get');
  assert.match(pollURL, /^\/demo\/app\/verwaltung\/posteingang\/[^/]+\/vorschlag/);
  await page.evaluate(() => { window.__inboxQA.armed = true; });
  return requestsBefore;
}

async function arrived() {
  await statusText('Vorschlag eingetroffen');
  await page.getByRole('heading', { name: 'Einordnung', exact: true }).waitFor({ state: 'visible' });
  assert.match(await page.locator('#case-workflow').innerText(), /Reparatur/);
  assert.equal(await page.locator('#vorschlag').getAttribute('hx-trigger'), null, 'completed job still polls');
  await page.locator('[data-shortcut="edit"]').first().waitFor({ state: 'visible' });
  assert(await page.getByRole('button', { name: 'Manuell', exact: true }).isEnabled());
  await assertIntact();
}

async function assertPollingStopped() {
  const count = report.pollPaths.length;
  // More than two 1-second ticks: removing the attribute without disposing the
  // htmx timer must fail this check.
  await page.waitForTimeout(2300);
  assert.equal(report.pollPaths.length, count, 'polling continued after error');
  await assertIntact();
}

try {
  const executablePath = [
    process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH,
    ...(process.env.CI === 'true' ? [] : [
      '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
      '/Applications/Chromium.app/Contents/MacOS/Chromium',
      '/usr/bin/chromium', '/usr/bin/chromium-browser', '/usr/bin/google-chrome',
    ]),
  ].filter(Boolean).find(existsSync);
  browser = await chromium.launch({ headless: true, ...(executablePath ? { executablePath } : {}) });
  const context = await browser.newContext({ viewport: { width: 1440, height: 1000 } });
  await context.addInitScript(() => {
    window.__inboxQA = { armed: false, missing: [] };
    const check = () => {
      if (window.__inboxQA.armed && document.querySelector('#case-body') && !document.querySelector('#vorschlag')) {
        window.__inboxQA.missing.push('missing after DOM mutation or htmx event');
      }
    };
    new MutationObserver(check).observe(document, { childList: true, subtree: true });
    for (const name of ['htmx:afterSwap', 'htmx:afterRequest', 'htmx:responseError', 'htmx:sendError']) {
      document.addEventListener(name, check);
    }
  });
  page = await context.newPage();
  page.setDefaultTimeout(15_000);
  page.on('pageerror', error => report.pageErrors.push(error.message));
  page.on('request', request => {
    const url = new URL(request.url());
    if (request.method() === 'GET' && /\/posteingang\/[^/]+\/vorschlag$/.test(url.pathname)) {
      report.pollPaths.push(url.pathname + url.search);
    }
  });
  await login();

  for (const mode of ['ok', 'langsam']) {
    await createOpenCase(mode);
    const pollsBefore = report.pollPaths.length;
    const requestsBefore = await requestSuggestion(mode);
    const runningState = await page.locator('#vorschlag').evaluate(element => element.outerHTML);
    await snapshot(`${mode}-running`);
    await arrived();
    assert.equal(fixture.stats.requests - requestsBefore, 1, 'one click must start exactly one provider request');
    assert(report.pollPaths.length - pollsBefore >= (mode === 'langsam' ? 2 : 1));
    await snapshot(`${mode}-arrived`);
    report.scenarios.push({ name: mode, passed: true });
    if (mode === 'ok') {
      // Re-arm this case's genuine running markup alongside its now populated
      // workflow. This isolates the preservation contract for Bearbeiten and
      // Manuell (a newly requested, empty case has neither control yet).
      const casePath = new URL(page.url()).pathname;
      const matches = url => url.pathname === `${casePath}/vorschlag`;
      await page.route(matches, route => route.fulfill({ status: 503, body: 'QA status unavailable' }));
      const workflow = await page.locator('#case-workflow').innerHTML();
      await page.evaluate(html => {
        htmx.swap(document.querySelector('#vorschlag'), html, { swapStyle: 'outerHTML', settleDelay: 0 });
      }, runningState);
      await statusText('Vorschlagsstatus konnte nicht geladen werden');
      assert.equal(await page.locator('#case-workflow').innerHTML(), workflow);
      await assertPollingStopped();
      await snapshot('existing-workflow-controls');
      await page.locator('[data-shortcut="edit"]').first().click();
      await page.locator('#case-workflow textarea[name="reply"]').waitFor({ state: 'visible' });
      await page.goto(`${baseURL}${casePath}`, { waitUntil: 'networkidle' });
      await page.getByRole('button', { name: 'Manuell', exact: true }).click();
      await page.locator('.flash').getByText('Manuell übernommen', { exact: true }).waitFor({ state: 'visible' });
      await page.unroute(matches);
      report.scenarios.push({ name: 'existing-workflow-controls', passed: true });
    }
  }

  await createOpenCase('provider-error');
  await requestSuggestion('Fehler');
  await statusText('Vorschlag fehlgeschlagen');
  await statusText('Anbieter nicht erreichbar');
  await page.getByRole('button', { name: 'Erneut versuchen', exact: true }).waitFor({ state: 'visible' });
  await assertPollingStopped();
  await snapshot('provider-error');
  fixture.setMode('ok');
  await page.getByRole('button', { name: 'Erneut versuchen', exact: true }).click();
  await statusText('Vorschlag wird erstellt');
  await page.evaluate(() => { window.__inboxQA.armed = true; });
  await arrived();
  report.scenarios.push({ name: 'provider-error-and-retry', passed: true });

  const faults = ['missing-record', 'http-503', 'http-401', 'http-403', 'foreign-redirect', 'missing-select', 'empty-204', 'network'];
  for (const fault of faults) {
    const casePath = await createOpenCase(fault);
    const pollPath = `${casePath}/vorschlag`;
    const matches = url => url.pathname === pollPath;
    let fail = true;
    await page.route(matches, async route => {
      if (!fail) return route.continue();
      if (fault === 'network') return route.abort('connectionfailed');
      if (fault === 'foreign-redirect') return route.fulfill({ status: 303, headers: { location: `${baseURL}/haus-b/` }, body: '' });
      if (fault === 'missing-record') {
        const response = await route.fetch({ url: `${baseURL}/demo/app/verwaltung/posteingang/missing-hausv695/vorschlag` });
        assert.equal(response.status(), 404);
        assert.match(await response.text(), /Vorschlagsstatus konnte nicht geladen werden/);
        return route.fulfill({ response });
      }
      const status = fault.startsWith('http-') ? Number(fault.slice(5)) : fault === 'empty-204' ? 204 : 200;
      return route.fulfill({ status, contentType: 'text/html', body: status === 204 ? '' : '<main>Andere Seite ohne Vorschlagsstatus</main>' });
    });
    const requestsBefore = await requestSuggestion('langsam');
    const workflow = await page.locator('#case-workflow').innerHTML();
    await statusText('Vorschlagsstatus konnte nicht geladen werden');
    await statusText('im Hintergrund weiter');
    assert.equal(await page.locator('#case-workflow').innerHTML(), workflow, 'poll error changed the case workflow');
    assert(await page.getByRole('button', { name: 'Abbrechen', exact: true }).isEnabled());
    await page.getByRole('button', { name: 'Erneut versuchen', exact: true }).waitFor({ state: 'visible' });
    assert.equal(await page.locator('#vorschlag').getAttribute('hx-trigger'), 'inbox:suggestion-retry');
    await assertPollingStopped();
    await snapshot(fault);
    if (fault === 'http-503') {
      await Promise.all([
        page.waitForResponse(response => new URL(response.url()).pathname === pollPath),
        page.getByRole('button', { name: 'Erneut versuchen', exact: true }).click(),
      ]);
      await statusText('Vorschlagsstatus konnte nicht geladen werden');
      await assertPollingStopped();
    }
    fail = false;
    if (fault === 'network') {
      await page.getByRole('button', { name: 'Abbrechen', exact: true }).click();
      await statusText('Abgebrochen');
      await assertIntact();
    } else {
      await page.getByRole('button', { name: 'Erneut versuchen', exact: true }).click();
      await arrived();
    }
    assert.equal(fixture.stats.requests - requestsBefore, 1, 'status retry started another provider job');
    await page.unroute(matches);
    report.scenarios.push({ name: fault, passed: true });
  }
  assert.deepEqual(report.pageErrors, [], 'unexpected browser JavaScript errors');
  report.passed = true;
  process.stdout.write(`HAUSV-695: ${report.scenarios.length} scenarios passed; ${report.pollPaths.length} tenant-scoped polls\n`);
} catch (error) {
  report.error = error.stack || String(error);
  if (page) await page.screenshot({ path: join(reportDir, 'failure.png'), fullPage: true }).catch(() => {});
  throw error;
} finally {
  writeFileSync(join(reportDir, 'report.json'), JSON.stringify(report, null, 2) + '\n');
  try {
    await browser?.close();
  } finally {
    await fixture.close();
  }
}
