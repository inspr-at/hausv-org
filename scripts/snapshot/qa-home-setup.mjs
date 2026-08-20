#!/usr/bin/env node

import { existsSync, mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { chromium } from 'playwright';

const [baseURL, phase, statePath] = process.argv.slice(2);
if (!baseURL || !['create', 'verify'].includes(phase) || !statePath) {
  console.error('usage: qa-home-setup.mjs <base-url> <create|verify> <storage-state-path>');
  process.exit(1);
}

const smtpAPI = process.env.HV_QA_SMTP_API;
if (!smtpAPI) throw new Error('HV_QA_SMTP_API is required');
const artifactDir = process.env.HV_QA_ARTIFACT_DIR?.trim() || '';
if (artifactDir) mkdirSync(artifactDir, { recursive: true });
const slug = 'qa-home-e2e';
const ownerEmail = 'qa-home-owner@example.com';
const executableCandidates = [
  process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH,
  ...(process.env.CI === 'true' ? [] : [
    '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
    '/Applications/Chromium.app/Contents/MacOS/Chromium',
    '/usr/bin/chromium',
    '/usr/bin/google-chrome',
  ]),
].filter(Boolean);
const executablePath = executableCandidates.find(existsSync);
const launchOptions = {
  headless: true,
  args: ['--host-resolver-rules=MAP hausv.test 127.0.0.1, MAP *.hausv.test 127.0.0.1', '--no-proxy-server'],
};
if (executablePath) launchOptions.executablePath = executablePath;

function publicOrigin() {
  const url = new URL(baseURL);
  url.hostname = 'hausv.test';
  return url.origin;
}

async function capturedConfirmationPath() {
  const deadline = Date.now() + 10_000;
  while (Date.now() < deadline) {
    const response = await fetch(new URL('/messages', smtpAPI));
    const messages = await response.json();
    const message = messages.find((item) => item.body.includes(ownerEmail));
    const match = message?.body.match(/https?:\/\/[^\s]+(\/start\/verify\?token=[^\s]+)/);
    if (match) return match[1];
    await new Promise((resolve) => setTimeout(resolve, 100));
  }
  throw new Error('confirmation email was not captured');
}

const browser = await chromium.launch(launchOptions);
let page;
try {
  const context = await browser.newContext({
    viewport: { width: 1280, height: 900 },
    locale: 'de-AT',
    timezoneId: 'Europe/Vienna',
    ...(phase === 'verify' ? { storageState: statePath } : {}),
  });
  page = await context.newPage();
  const origin = publicOrigin();

  if (phase === 'create') {
    await page.goto(new URL('/start', origin).href, { waitUntil: 'networkidle' });
    await page.getByLabel('Name des Zuhauses').fill('QA Zuhause Browserlauf');
    await page.getByLabel('Gewünschter Pfad').fill(slug);
    await page.getByLabel('Eigentümer-E-Mail').fill(ownerEmail);
    await page.getByRole('checkbox').check();
    await Promise.all([
      page.waitForURL('**/start?sent=1'),
      page.getByRole('button', { name: 'Bestätigungslink anfordern' }).click(),
    ]);
    await page.getByRole('heading', { name: 'Bitte E-Mail prüfen' }).waitFor();

    const confirmationPath = await capturedConfirmationPath();
    await page.goto(new URL(confirmationPath, origin).href, { waitUntil: 'networkidle' });
    await page.getByRole('heading', { name: /QA Zuhause Browserlauf/ }).waitFor();
    await page.getByText('Schritt 1 von 2').waitFor();
    await Promise.all([
      page.waitForURL(`**/${slug}/app?activated=1`),
      page.getByRole('button', { name: 'Portal jetzt aktivieren' }).click(),
    ]);
    await page.locator('[data-portal-shell]').waitFor();

    await page.goto(new URL('/start/connector', origin).href, { waitUntil: 'networkidle' });
    await page.getByText('Schritt 2 von 2').waitFor();
    await page.getByRole('button', { name: 'Energieverbindung vorbereiten' }).click();
    const pairingCode = (await page.locator('.home-pairing-code').textContent())?.trim();
    if (!pairingCode || pairingCode.length < 40) throw new Error('pairing code missing');

    const heartbeat = {
      connector_version: '0.86.0', home_assistant_version: '2026.8.1', entity_count: 27,
      readings: [{
        entity_id: 'sensor.grid_import_power', state: '1250', display_name: 'Netzbezug',
        unit: 'W', device_class: 'power', state_class: 'measurement', last_updated: new Date().toISOString(),
      }],
    };
    const pairResult = await page.evaluate(async ({ pairingCode, heartbeat }) => {
      const response = await fetch('/api/home-connectors/pair', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ pairing_code: pairingCode, ...heartbeat }),
      });
      return { status: response.status, body: await response.json() };
    }, { pairingCode, heartbeat });
    if (pairResult.status !== 201) throw new Error(`pairing failed with ${pairResult.status}`);
    const { credential } = pairResult.body;
    const beatStatus = await page.evaluate(async ({ credential, heartbeat }) => {
      const response = await fetch('/api/home-connectors/heartbeat', {
        method: 'POST', headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${credential}` },
        body: JSON.stringify(heartbeat),
      });
      return response.status;
    }, { credential, heartbeat });
    if (beatStatus !== 200) throw new Error(`heartbeat failed with ${beatStatus}`);
    await page.reload({ waitUntil: 'networkidle' });
    await page.getByText('Verbunden und bereit').waitFor();
    await context.storageState({ path: statePath });
    process.stdout.write('  ✓ Reservierung, echte E-Mail, Aktivierung und lokale Kopplung im Browser\n');
  } else {
    await page.goto(new URL(`/${slug}/app`, origin).href, { waitUntil: 'networkidle' });
    await page.locator('[data-portal-shell]').waitFor();
    await page.goto(new URL('/start/connector', origin).href, { waitUntil: 'networkidle' });
    await page.getByText('Verbunden und bereit').waitFor();

    const foreignStatus = await page.evaluate(async () => {
      const response = await fetch('/demo/app', { redirect: 'manual' });
      return response.status;
    });
    if (![0, 302, 303, 401, 403].includes(foreignStatus)) {
      throw new Error(`owner session crossed tenant boundary with ${foreignStatus}`);
    }
    process.stdout.write('  ✓ Portal, Verbindung und Mandantentrennung nach Neustart\n');
  }
  await context.close();
} catch (error) {
  if (artifactDir) {
    writeFileSync(join(artifactDir, `home-setup-${phase}-failure.txt`), `${error?.stack || error}\n`);
    if (page) {
      await page.locator('.home-pairing-code,.home-command').evaluateAll((elements) => {
        for (const element of elements) element.textContent = '[vertraulicher Inhalt ausgeblendet]';
      }).catch(() => {});
      await page.screenshot({ path: join(artifactDir, `home-setup-${phase}-failure.png`), fullPage: true }).catch(() => {});
    }
  }
  throw error;
} finally {
  await browser.close();
}
