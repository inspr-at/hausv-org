#!/usr/bin/env node

import { existsSync, mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { chromium } from 'playwright';

const baseURL = process.argv[2];
if (!baseURL) {
  console.error('usage: qa-public-auth.mjs <tenant-base-url>');
  process.exit(1);
}

const artifactDir = process.env.HV_QA_ARTIFACT_DIR?.trim() || '';
const screenshotDir = process.env.HV_QA_SCREENSHOT_DIR?.trim() || artifactDir;
const screenshotPrefix = process.env.HV_QA_SCREENSHOT_PREFIX?.trim() || 'public-auth';
if (artifactDir) mkdirSync(artifactDir, { recursive: true });
if (screenshotDir) mkdirSync(screenshotDir, { recursive: true });

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
  headless: true,
  args: ['--host-resolver-rules=MAP hausv.test 127.0.0.1, MAP *.hausv.test 127.0.0.1', '--no-proxy-server'],
};
if (executablePath) launchOptions.executablePath = executablePath;

const browser = await chromium.launch(launchOptions);
const requestedViewports = new Set((process.env.HV_QA_VIEWPORTS || '').split(',').map((item) => item.trim()).filter(Boolean));
const viewports = [
  { name: '320', width: 320, height: 568, email: 'resident@example.com' },
  { name: '390', width: 390, height: 844, email: 'owner@example.com' },
  { name: '768', width: 768, height: 900, email: 'admin@example.com' },
  { name: '1024', width: 1024, height: 900, email: 'verwalter@example.com' },
  { name: '1440', width: 1440, height: 900, email: 'resident@example.com' },
].filter((viewport) => requestedViewports.size === 0 || requestedViewports.has(viewport.name));
const report = [];
let mapTileResponseVerified = false;

function tenantOrigin() {
  const url = new URL(baseURL);
  url.hostname = 'localhost';
  return url.origin;
}

function publicOrigin() {
  const url = new URL(baseURL);
  url.hostname = 'hausv.test';
  return url.origin;
}

async function screenshot(page, name, fullPage = false) {
  if (!screenshotDir) return;
  await page.screenshot({ path: join(screenshotDir, `${name}.png`), fullPage });
}

async function screenshotElement(locator, name) {
  if (!screenshotDir) return;
  await locator.screenshot({ path: join(screenshotDir, `${name}.png`) });
}

async function metrics(page) {
  return page.evaluate(() => {
    const controls = [...document.querySelectorAll('a, button, input, summary')]
      .filter((element) => {
        const style = getComputedStyle(element);
        const box = element.getBoundingClientRect();
        return style.visibility !== 'hidden' && style.display !== 'none' && box.width > 0 && box.height > 0;
      })
      .map((element) => {
        const box = element.getBoundingClientRect();
        return {
          label: (element.getAttribute('aria-label') || element.textContent || element.getAttribute('name') || '').trim().replace(/\s+/g, ' ').slice(0, 80),
          tag: element.tagName.toLowerCase(),
          width: Math.round(box.width * 10) / 10,
          height: Math.round(box.height * 10) / 10,
          outside: box.left < -1 || box.right > innerWidth + 1,
        };
      });
    return {
      url: location.pathname + location.search + location.hash,
      title: document.title,
      width: innerWidth,
      scrollWidth: document.documentElement.scrollWidth,
      scrollHeight: document.documentElement.scrollHeight,
      overflow: document.documentElement.scrollWidth > innerWidth + 1,
      outside: controls.filter((item) => item.outside),
      shortPrimaryControls: controls.filter((item) => ['button', 'input', 'summary'].includes(item.tag) && item.height < 44),
      headings: [...document.querySelectorAll('h1, h2')].map((element) => element.textContent.trim()).slice(0, 10),
    };
  });
}

async function newContext(viewport, options = {}) {
  return browser.newContext({
    viewport: { width: viewport.width, height: viewport.height },
    locale: 'de-AT',
    timezoneId: 'Europe/Vienna',
    javaScriptEnabled: options.javaScriptEnabled ?? true,
    reducedMotion: options.reducedMotion ?? 'no-preference',
  });
}

async function verifyLoggedOutMap(page, viewport) {
  const card = page.locator('.location-card');
  const map = page.locator('.location-map');
  const tiles = page.locator('.location-map-tile');
  const pin = page.locator('.location-pin');
  await tiles.first().waitFor({ state: 'visible' });
  const tileSources = await tiles.evaluateAll((elements) => elements.map((element) => ({
    src: element.getAttribute('data-map-tile'),
    background: getComputedStyle(element).backgroundImage,
  })));
  if (tileSources.length < 2 || tileSources.some((tile) => !tile.src?.startsWith('/map-tiles/') || !tile.background.includes(tile.src))) {
    throw new Error(`Login ${viewport.name}: location card is not backed by local map tiles ${JSON.stringify(tileSources)}`);
  }

  const mapBox = await map.boundingBox();
  const cardBox = await card.boundingBox();
  const pinBox = await pin.boundingBox();
  const tileBoxes = await tiles.evaluateAll((images) => images.map((image) => {
    const box = image.getBoundingClientRect();
    return { left: box.left, right: box.right, top: box.top, bottom: box.bottom };
  }));
  if (!mapBox || !cardBox || !pinBox || cardBox.width > 421 || Math.abs(mapBox.height - 92) > 1) {
    throw new Error(`Login ${viewport.name}: invalid location-card dimensions`);
  }
  const coverage = {
    left: Math.min(...tileBoxes.map((box) => box.left)),
    right: Math.max(...tileBoxes.map((box) => box.right)),
    top: Math.min(...tileBoxes.map((box) => box.top)),
    bottom: Math.max(...tileBoxes.map((box) => box.bottom)),
  };
  if (coverage.left > mapBox.x + 1 || coverage.right < mapBox.x + mapBox.width - 1 ||
      coverage.top > mapBox.y + 1 || coverage.bottom < mapBox.y + mapBox.height - 1) {
    throw new Error(`Login ${viewport.name}: map tiles do not cover the crop ${JSON.stringify({ mapBox, coverage })}`);
  }
  const markerDelta = {
    x: Math.abs(pinBox.x + pinBox.width / 2 - (mapBox.x + mapBox.width / 2)),
    y: Math.abs(pinBox.y + pinBox.height / 2 - (mapBox.y + mapBox.height / 2)),
  };
  if (markerDelta.x > 1 || markerDelta.y > 1 || await map.evaluate((element) => element.classList.contains('map-tile-failed'))) {
    throw new Error(`Login ${viewport.name}: location marker is not centred ${JSON.stringify(markerDelta)}`);
  }
  const layers = await map.evaluate((element) => ({
    tiles: getComputedStyle(element.querySelector('.location-map-tiles')).zIndex,
    fallback: getComputedStyle(element.querySelector('.location-map-fallback')).zIndex,
  }));
  if (layers.tiles !== '1' || layers.fallback !== 'auto') throw new Error(`Login ${viewport.name}: successful map does not cover its fallback`);

  if (!mapTileResponseVerified) {
    const response = await page.context().request.get(new URL(tileSources[0].src, page.url()).href);
    const bytes = await response.body();
    const pngSignature = Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]);
    if (!response.ok() || !response.headers()['content-type']?.startsWith('image/png') || bytes.length < 1000 ||
        !bytes.subarray(0, pngSignature.length).equals(pngSignature)) {
      throw new Error('Logged-out map endpoint is not a representative local PNG tile');
    }
    mapTileResponseVerified = true;
  }
  await screenshotElement(card, `${screenshotPrefix}-login-map-card-${viewport.name}`);
  return { cardWidth: cardBox.width, mapWidth: mapBox.width, mapHeight: mapBox.height, markerDelta, tileCount: tileSources.length };
}

async function verifyMapFailureFallback(viewport) {
  if (viewport.name !== '390') return null;
  const context = await newContext(viewport);
  await context.route('**/map-tiles/**', (route) => route.fulfill({ status: 502, contentType: 'text/plain', body: 'fixture failure' }));
  const page = await context.newPage();
  await page.goto(tenantOrigin(), { waitUntil: 'networkidle' });
  await page.waitForFunction(() => document.querySelector('.location-map')?.classList.contains('map-tile-failed'));
  const fallback = {
    failed: await page.locator('.location-map').evaluate((element) => element.classList.contains('map-tile-failed')),
    opacity: await page.locator('.location-map-fallback').evaluate((element) => getComputedStyle(element).opacity),
    visibleTiles: await page.locator('.location-map-tile:visible').count(),
  };
  if (!fallback.failed || fallback.opacity !== '1' || fallback.visibleTiles !== 0) {
    throw new Error(`Login ${viewport.name}: map failure fallback is not clean ${JSON.stringify(fallback)}`);
  }
  await screenshotElement(page.locator('.location-card'), `${screenshotPrefix}-login-map-fallback-${viewport.name}`);
  await context.close();

  const noJSContext = await newContext(viewport, { javaScriptEnabled: false });
  await noJSContext.route('**/map-tiles/**', (route) => route.fulfill({ status: 502, contentType: 'text/plain', body: 'fixture failure' }));
  const noJSPage = await noJSContext.newPage();
  await noJSPage.goto(tenantOrigin(), { waitUntil: 'networkidle' });
  const noJS = {
    failedClass: await noJSPage.locator('.location-map').evaluate((element) => element.classList.contains('map-tile-failed')),
    fallbackDisplay: await noJSPage.locator('.location-map-fallback').evaluate((element) => getComputedStyle(element).display),
    tileCount: await noJSPage.locator('.location-map-tile').count(),
  };
  if (noJS.failedClass || noJS.fallbackDisplay !== 'grid' || noJS.tileCount < 2) {
    throw new Error(`Login ${viewport.name}: no-JS map fallback is not structurally available ${JSON.stringify(noJS)}`);
  }
  await screenshotElement(noJSPage.locator('.location-card'), `${screenshotPrefix}-login-map-fallback-no-js-${viewport.name}`);
  await noJSContext.close();
  return { scripted: fallback, noJS };
}

async function captureViewport(viewport) {
  const context = await newContext(viewport);
  const unexpectedNetwork = [];
  context.on('request', (request) => {
    const host = new URL(request.url()).hostname;
    if (host !== 'localhost' && host !== 'hausv.test') unexpectedNetwork.push(request.url());
  });
  const page = await context.newPage();

  await page.goto(publicOrigin(), { waitUntil: 'networkidle' });
  await screenshot(page, `${screenshotPrefix}-landing-top-${viewport.name}`);
  await screenshot(page, `${screenshotPrefix}-landing-full-${viewport.name}`, true);
  const landing = await metrics(page);
  const primaryCTA = await page.locator('.landing-hero .landing-button.primary').boundingBox();
  if (!primaryCTA || primaryCTA.height < 44 || primaryCTA.y + primaryCTA.height > viewport.height + 1) {
    throw new Error(`Landing ${viewport.name}: primary CTA is not fully reachable in the first viewport`);
  }
  if (landing.overflow || landing.outside.length || landing.shortPrimaryControls.length) {
    throw new Error(`Landing ${viewport.name}: invalid geometry ${JSON.stringify(landing)}`);
  }
  const contactHref = await page.locator('.landing-hero .landing-button.primary').getAttribute('href');
  if (!contactHref?.startsWith('mailto:')) throw new Error(`Landing ${viewport.name}: primary contact action is not ready`);
  for (const id of ['funktionen', 'sicherheit', 'preise', 'impressum', 'kontakt']) {
    if (!(await page.locator(`#${id}`).count())) throw new Error(`Landing ${viewport.name}: #${id} destination is missing`);
  }

  if (viewport.width <= 900) {
    const menu = page.locator('[data-landing-menu-toggle]');
    if ((await menu.getAttribute('aria-expanded')) !== 'false') throw new Error(`Landing ${viewport.name}: menu starts expanded`);
    await menu.click();
    if ((await menu.getAttribute('aria-expanded')) !== 'true' || !(await page.locator('#landing-navigation').isVisible())) {
      throw new Error(`Landing ${viewport.name}: menu does not expose its state`);
    }
    await screenshot(page, `${screenshotPrefix}-landing-menu-${viewport.name}`);
    await page.keyboard.press('Escape');
    if ((await menu.getAttribute('aria-expanded')) !== 'false' || !(await menu.evaluate((element) => element === document.activeElement))) {
      throw new Error(`Landing ${viewport.name}: Escape does not close and restore focus`);
    }
  }

  const productDetails = page.locator('details.landing-more');
  await productDetails.locator('summary').click();
  await screenshot(page, `${screenshotPrefix}-landing-product-details-${viewport.name}`);
  if (!(await productDetails.getByRole('heading', { name: 'Heute nutzbar' }).isVisible())) {
    throw new Error(`Landing ${viewport.name}: product disclosure does not open`);
  }
  await productDetails.locator('summary').click();
  const legalDetails = page.locator('details.legal-details');
  await legalDetails.locator('summary').click();
  await screenshot(page, `${screenshotPrefix}-landing-legal-details-${viewport.name}`);
  if (!(await legalDetails.getByText('Keine externe Zertifizierung', { exact: false }).isVisible())) {
    throw new Error(`Landing ${viewport.name}: legal disclosure does not open`);
  }

  await page.goto(tenantOrigin(), { waitUntil: 'networkidle' });
  await screenshot(page, `${screenshotPrefix}-login-${viewport.name}`);
  await screenshot(page, `${screenshotPrefix}-login-full-${viewport.name}`, true);
  const login = await metrics(page);
  const loginAction = await page.getByRole('button', { name: 'Anmeldelink senden' }).boundingBox();
  if (!loginAction || loginAction.height < 44 || loginAction.y + loginAction.height > viewport.height + 1) {
    throw new Error(`Login ${viewport.name}: primary action is not fully reachable in the first viewport`);
  }
  if (login.overflow || login.outside.length || login.shortPrimaryControls.length) {
    throw new Error(`Login ${viewport.name}: invalid geometry ${JSON.stringify(login)}`);
  }
  const locationMap = await verifyLoggedOutMap(page, viewport);

  await page.keyboard.press('Tab');
  await page.keyboard.press('Tab');
  const keyboard = await page.evaluate(() => {
    const active = document.activeElement;
    const style = active ? getComputedStyle(active) : null;
    return {
      tag: active?.tagName.toLowerCase() || '',
      visible: Boolean(active?.getClientRects().length),
      outlineWidth: Number.parseFloat(style?.outlineWidth || '0'),
    };
  });
  if (!keyboard.visible || keyboard.outlineWidth < 2) throw new Error(`Login ${viewport.name}: keyboard focus is not visible`);
  await screenshot(page, `${screenshotPrefix}-login-keyboard-${viewport.name}`);

  const email = page.locator('input[name="email"]');
  await email.fill(viewport.email);
  await page.locator('form[action="/auth/request"] button[type="submit"]').click();
  await page.locator('a.dev-link').waitFor({ state: 'visible' });
  const devHref = await page.locator('a.dev-link').getAttribute('href');
  await screenshot(page, `${screenshotPrefix}-login-link-ready-${viewport.name}`);
  await screenshot(page, `${screenshotPrefix}-login-link-ready-full-${viewport.name}`, true);
  const linkReady = await metrics(page);
  const readyAction = await page.getByRole('link', { name: 'Weiter zum Portal' }).boundingBox();
  if (!readyAction || readyAction.height < 44 || readyAction.y + readyAction.height > viewport.height + 1) {
    throw new Error(`Login ${viewport.name}: prepared-login action is not fully reachable`);
  }
  if ((await page.locator('form[action="/auth/request"]:visible').count()) !== 0) {
    throw new Error(`Login ${viewport.name}: resend form competes with prepared-login action`);
  }
  const readyText = await page.locator('#login').innerText();
  if (/Mailversand|Testzugang|Magic.Link|SSO|Zitadel/i.test(readyText)) {
    throw new Error(`Login ${viewport.name}: prepared-login state exposes provider/development jargon`);
  }

  await page.goBack({ waitUntil: 'networkidle' });
  const back = {
    url: new URL(page.url()).pathname + new URL(page.url()).search,
    email: await page.locator('input[name="email"]').inputValue(),
  };

  if (unexpectedNetwork.length) throw new Error(`Public/auth ${viewport.name}: external network request ${unexpectedNetwork[0]}`);
  await context.close();
  const mapFailureFallback = await verifyMapFailureFallback(viewport);

  const privacyContext = await newContext(viewport);
  const privacyPage = await privacyContext.newPage();
  await privacyPage.goto(tenantOrigin(), { waitUntil: 'networkidle' });
  await Promise.all([
    privacyPage.waitForURL((url) => url.pathname === '/datenschutz'),
    privacyPage.locator('footer a[href="/datenschutz"]').click(),
  ]);
  if (!(await privacyPage.getByRole('heading', { name: 'Datenschutz' }).count())) {
    throw new Error(`Privacy ${viewport.name}: heading is missing`);
  }
  await screenshot(privacyPage, `${screenshotPrefix}-privacy-${viewport.name}`);
  await screenshot(privacyPage, `${screenshotPrefix}-privacy-full-${viewport.name}`, true);
  const privacy = await metrics(privacyPage);
  if (privacy.overflow || privacy.outside.length) throw new Error(`Privacy ${viewport.name}: invalid geometry ${JSON.stringify(privacy)}`);
  await privacyPage.goBack({ waitUntil: 'networkidle' });
  if (new URL(privacyPage.url()).pathname !== '/') throw new Error(`Privacy ${viewport.name}: browser Back does not return to login`);
  await privacyContext.close();

  const consumedContext = await newContext(viewport);
  const consumedPage = await consumedContext.newPage();
  await consumedPage.goto(devHref, { waitUntil: 'networkidle' });
  const firstUse = new URL(consumedPage.url()).pathname;
  await consumedContext.close();

  const usedContext = await newContext(viewport);
  const usedPage = await usedContext.newPage();
  await usedPage.goto(devHref, { waitUntil: 'networkidle' });
  await screenshot(usedPage, `${screenshotPrefix}-login-link-used-${viewport.name}`);
  await screenshot(usedPage, `${screenshotPrefix}-login-link-used-full-${viewport.name}`, true);
  const used = await metrics(usedPage);
  if (!(await usedPage.getByRole('heading', { name: 'Neuen Link anfordern' }).count())) {
    throw new Error(`Login ${viewport.name}: used token has no clear recovery action`);
  }
  await usedContext.close();

  const invalidContext = await newContext(viewport);
  const invalidPage = await invalidContext.newPage();
  await invalidPage.goto(`${tenantOrigin()}/auth/verify?token=invalid-audit-token`, { waitUntil: 'networkidle' });
  await screenshot(invalidPage, `${screenshotPrefix}-login-link-invalid-${viewport.name}`);
  const invalid = await metrics(invalidPage);
  await invalidContext.close();

  const deniedContext = await newContext(viewport);
  const deniedPage = await deniedContext.newPage();
  await deniedPage.goto(`${tenantOrigin()}/?denied=1#login`, { waitUntil: 'networkidle' });
  await screenshot(deniedPage, `${screenshotPrefix}-login-access-denied-${viewport.name}`);
  const denied = await metrics(deniedPage);
  if (!(await deniedPage.getByRole('heading', { name: 'Zugang prüfen' }).count())) {
    throw new Error(`Login ${viewport.name}: denied state has no clear next step`);
  }
  await deniedContext.close();

  const reducedContext = await newContext(viewport, { reducedMotion: 'reduce' });
  const reducedPage = await reducedContext.newPage();
  await reducedPage.goto(publicOrigin(), { waitUntil: 'networkidle' });
  const reduced = await reducedPage.evaluate(() => ({
    reduced: matchMedia('(prefers-reduced-motion: reduce)').matches,
    behavior: getComputedStyle(document.documentElement).scrollBehavior,
    mark: document.querySelector('[data-hausv-mark-3d]')?.getAttribute('data-mark3d-state') || 'not-mounted',
  }));
  if (!reduced.reduced || reduced.behavior !== 'auto') throw new Error(`Landing ${viewport.name}: reduced motion is not respected`);
  await screenshot(reducedPage, `${screenshotPrefix}-landing-reduced-motion-${viewport.name}`);
  await reducedContext.close();

  const noJSContext = await newContext(viewport, { javaScriptEnabled: false });
  const noJSPage = await noJSContext.newPage();
  await noJSPage.goto(publicOrigin(), { waitUntil: 'networkidle' });
  if (viewport.width <= 900) {
    if ((await noJSPage.locator('#landing-navigation a:visible').count()) !== 4) {
      throw new Error(`Landing ${viewport.name}: no-JS navigation is not fully reachable`);
    }
    await screenshot(noJSPage, `${screenshotPrefix}-landing-no-js-menu-${viewport.name}`);
  }
  await noJSPage.goto(tenantOrigin(), { waitUntil: 'networkidle' });
  await noJSPage.locator('input[name="email"]').fill(viewport.email);
  await noJSPage.locator('form[action="/auth/request"] button[type="submit"]').click();
  const noJS = {
    url: new URL(noJSPage.url()).pathname + new URL(noJSPage.url()).search,
    devLinkVisible: await noJSPage.locator('a.dev-link').isVisible(),
    metrics: await metrics(noJSPage),
  };
  if (!noJS.devLinkVisible || noJS.metrics.overflow) throw new Error(`Login ${viewport.name}: no-JS flow failed`);
  await screenshot(noJSPage, `${screenshotPrefix}-login-no-js-link-ready-${viewport.name}`);
  await noJSContext.close();

  if (back.email !== viewport.email || firstUse !== '/app' || used.url !== '/?login=expired#login' || invalid.url !== '/?login=expired#login') {
    throw new Error(`Login ${viewport.name}: auth navigation contract failed`);
  }
  report.push({ viewport, landing, login, locationMap, mapFailureFallback, keyboard, linkReady, back, privacy, firstUse, used, invalid, denied, reduced, noJS });
  process.stdout.write(`  ✓ public/auth ${viewport.width}x${viewport.height}\n`);
}

try {
  for (const viewport of viewports) await captureViewport(viewport);
  if (artifactDir) writeFileSync(join(artifactDir, 'public-auth-audit.json'), JSON.stringify(report, null, 2));
} finally {
  await browser.close();
}
