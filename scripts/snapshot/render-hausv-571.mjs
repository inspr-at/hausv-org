#!/usr/bin/env node
// Render the real HAUSV-571 templ compositions from a running fixture portal.
//
// Usage: node render-hausv-571.mjs <baseURL> [outputDir]

// The reference images in HAUSV-521 are static HTML. This proof goes through
// login, real handlers, role selection, templ rendering, and the shared portal
// chrome so the screenshots cover the integration rather than a second mockup.

import { existsSync } from 'node:fs';
import { mkdir } from 'node:fs/promises';
import path from 'node:path';
import { chromium } from 'playwright';

const baseURL = process.argv[2]?.replace(/\/$/, '');
const outputDir = path.resolve(process.argv[3] || 'docs/snapshots/HAUSV-571');
if (!baseURL) {
  console.error('usage: node render-hausv-571.mjs <baseURL> [outputDir]');
  process.exit(1);
}

const tenantURL = `${baseURL}/demo`;
const executableCandidates = [
  process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH,
  '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
  '/Applications/Chromium.app/Contents/MacOS/Chromium',
  '/usr/bin/chromium',
  '/usr/bin/chromium-browser',
  '/usr/bin/google-chrome',
].filter(Boolean);
const executablePath = executableCandidates.find(existsSync);
const browser = await chromium.launch({
  headless: process.env.CI === 'true' || process.env.HV_QA_HEADLESS !== 'false',
  ...(executablePath ? { executablePath } : {}),
});

function futureLocalInput(daysAhead, hour) {
  const date = new Date();
  date.setDate(date.getDate() + daysAhead);
  date.setHours(hour, 0, 0, 0);
  const offset = date.getTimezoneOffset() * 60_000;
  return new Date(date.getTime() - offset).toISOString().slice(0, 16);
}

async function contextFor(viewport) {
  return browser.newContext({
    viewport,
    deviceScaleFactor: 2,
    locale: 'de-AT',
    timezoneId: 'Europe/Vienna',
    reducedMotion: 'reduce',
  });
}

async function login(context, email) {
  const page = await context.newPage();
  await page.goto(`${tenantURL}/`, { waitUntil: 'networkidle' });
  const emailDetails = page.locator('details:has(form[action$="/auth/request"])');
  if (await emailDetails.count()) {
    await emailDetails.evaluate((element) => { element.open = true; });
  }
  await page.locator('input[name="email"]').fill(email);
  await page.locator('form[action$="/auth/request"] button[type="submit"]').click();
  const devLink = page.locator('a.dev-link');
  await devLink.waitFor({ state: 'visible' });
  const href = await devLink.getAttribute('href');
  if (!href) throw new Error(`local login did not return a link for ${email}`);
  const target = new URL(href, `${tenantURL}/`);
  const localOrigin = new URL(tenantURL);
  target.protocol = localOrigin.protocol;
  target.port = localOrigin.port;
  await page.goto(target.href, { waitUntil: 'networkidle' });
  return page;
}

async function createIssue(email, title, body, location) {
  const context = await contextFor({ width: 1440, height: 900 });
  const page = await login(context, email);
  await page.goto(`${tenantURL}/app/anliegen?new=1#issue-new`, { waitUntil: 'networkidle' });
  if (await page.getByText(title, { exact: true }).count()) {
    await context.close();
    return;
  }
  const panel = page.locator('#issue-new');
  if (await panel.evaluate((element) => element.tagName === 'DETAILS' && !element.open)) {
    await panel.locator(':scope > summary').click();
  }
  const form = panel.locator('form[data-issue-wizard]');
  await form.locator('textarea[name="body"]').fill(body);
  await form.locator('input[name="location_detail"]').fill(location);
  await form.locator('[data-issue-next]').click();
  const titleDetails = form.locator('details.issue-title-option');
  await titleDetails.evaluate((element) => { element.open = true; });
  await form.locator('input[name="title"]').fill(title);
  await form.locator('.wizard-submit').click();
  await page.waitForLoadState('networkidle');
  if (!(await page.getByText(title, { exact: true }).count())) {
    throw new Error(`created issue is not visible: ${title}`);
  }
  await context.close();
}

async function seedManagedContent() {
  const context = await contextFor({ width: 1440, height: 900 });
  const page = await login(context, 'admin@example.com');

  const announcements = [
    ['Reinigung Dachrinnen: Termin verschoben', 'Die Reinigung findet am kommenden Montag statt.', 'Wartung'],
    ['Neue Regelung für Müllraum-Nutzung', 'Bitte halten Sie die gekennzeichneten Stellflächen frei.', 'Info'],
    ['Einladung: Sommerfest im Innenhof', 'Der Beirat lädt die Hausgemeinschaft zum Sommerfest ein.', 'Termin'],
  ];
  for (const [title, body, category] of announcements) {
    await page.goto(`${tenantURL}/app/announcements`, { waitUntil: 'networkidle' });
    if (await page.getByText(title, { exact: true }).count()) continue;
    await page.getByRole('button', { name: 'Aushang erstellen' }).first().click();
    const form = page.locator('#announcement-create form');
    await form.locator('input[name="title"]').fill(title);
    await form.locator('textarea[name="body"]').fill(body);
    await form.locator('select[name="category"]').selectOption(category);
    await form.locator('button[type="submit"]').click();
    await page.waitForURL(/\/app\/announcements/);
  }

  const events = [
    ['Wartung Heizungsanlage', 1, 9, 'Heizraum'],
    ['Beiratssitzung', 7, 18, 'Gemeinschaftsraum'],
  ];
  for (const [title, daysAhead, hour, location] of events) {
    await page.goto(`${tenantURL}/app/events`, { waitUntil: 'networkidle' });
    if (await page.getByText(title, { exact: true }).count()) continue;
    await page.getByRole('button', { name: 'Termin erstellen' }).first().click();
    const form = page.locator('#event-create form');
    await form.locator('input[name="title"]').fill(title);
    await form.locator('input[name="starts_at"]').fill(futureLocalInput(daysAhead, hour));
    await form.locator('input[name="location"]').fill(location);
    await form.locator('button[type="submit"]').click();
    await page.waitForURL(/\/app\/events/);
  }
  await context.close();
}

await mkdir(outputDir, { recursive: true });
await createIssue(
  'admin@example.com',
  'Fahrradständer im Innenhof wackelt',
  'Der linke Fahrradständer ist lose und muss befestigt werden.',
  'Innenhof',
);
await createIssue(
  'owner@example.com',
  'Briefkastenanlage: Beschriftung erneuern',
  'Mehrere Namensschilder sind nicht mehr lesbar.',
  'Eingangsbereich',
);
await createIssue(
  'resident@example.com',
  'Kellerlicht im Stiegenhaus defekt',
  'Die Leuchte im ersten Kellergeschoß fällt immer wieder aus.',
  'Stiegenhaus · 1. Kellergeschoß',
);
await seedManagedContent();

const shots = [
  {
    name: 'hausueberblick-dense-templ',
    email: 'verwalter@example.com',
    viewport: { width: 1440, height: 900 },
    composition: 'dense',
  },
  {
    name: 'hausueberblick-quiet-templ',
    email: 'owner@example.com',
    viewport: { width: 1440, height: 900 },
    composition: 'calm',
  },
  {
    name: 'hausueberblick-mobile-templ',
    email: 'verwalter@example.com',
    viewport: { width: 390, height: 844 },
    composition: 'mobile',
  },
];

for (const shot of shots) {
  const context = await contextFor(shot.viewport);
  const page = await login(context, shot.email);
  await page.goto(`${tenantURL}/app`, { waitUntil: 'networkidle' });
  if (shot.composition === 'mobile') {
    if (!(await page.locator('.mobile-content').isVisible()) || await page.locator('.portal-home-desktop').isVisible()) {
      throw new Error('phone viewport did not select the shared mobile composition');
    }
  } else if (!(await page.locator(`body.${shot.composition}`).count())) {
    throw new Error(`${shot.email} did not select the ${shot.composition} composition`);
  }
  for (const heading of ['Anliegen', 'Termine', 'Aushang']) {
    if (!(await page.getByText(heading, { exact: true }).count())) {
      throw new Error(`${shot.name} is missing the shared ${heading} block`);
    }
  }
  await page.screenshot({ path: path.join(outputDir, `${shot.name}.png`), fullPage: true });
  console.log(`rendered ${shot.name}.png`);
  await context.close();
}

await browser.close();
