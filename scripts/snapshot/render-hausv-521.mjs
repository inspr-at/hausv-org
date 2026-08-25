#!/usr/bin/env node
// Render static HTML files to PNG for HAUSV-521 Hausüberblick rebuild snapshots.
//
// Usage: node render-hausv-521.mjs

import { chromium } from 'playwright';
import { readFile, writeFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import path from 'node:path';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const snapshotDir = path.resolve(__dirname, '../../docs/snapshots/HAUSV-521');

const executablePath = process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH;
const headless = process.env.CI === 'true' || process.env.HV_QA_HEADLESS !== 'false';

const browser = await chromium.launch({
  headless,
  ...(executablePath ? { executablePath } : {}),
});

const snapshots = [
  {
    name: 'hausueberblick-dense',
    file: 'hausueberblick-dense.html',
    viewport: { width: 1440, height: 900 },
    description: 'Dense view for Verwalter/Admin'
  },
  {
    name: 'hausueberblick-quiet',
    file: 'hausueberblick-quiet.html',
    viewport: { width: 1440, height: 900 },
    description: 'Quiet view for Eigentümer/Beirat/Bewohner'
  },
  {
    name: 'hausueberblick-mobile',
    file: 'hausueberblick-mobile.html',
    viewport: { width: 390, height: 844 },
    description: 'Mobile composition'
  },
];

console.log('Rendering HAUSV-521 snapshots...\n');

for (const snapshot of snapshots) {
  const htmlPath = path.join(snapshotDir, snapshot.file);
  const pngPath = path.join(snapshotDir, `${snapshot.name}.png`);
  
  console.log(`Rendering ${snapshot.description}...`);
  console.log(`  Input:  ${htmlPath}`);
  console.log(`  Output: ${pngPath}`);
  
  const ctx = await browser.newContext({
    viewport: snapshot.viewport,
    deviceScaleFactor: 2,
    locale: 'de-AT',
    timezoneId: 'Europe/Vienna',
  });
  
  const page = await ctx.newPage();
  
  // Read the HTML file and navigate to it as a data URL
  const html = await readFile(htmlPath, 'utf-8');
  await page.goto(`data:text/html;charset=utf-8,${encodeURIComponent(html)}`, {
    waitUntil: 'networkidle',
  });
  
  // Take full page screenshot
  await page.screenshot({
    path: pngPath,
    fullPage: true,
  });
  
  console.log(`  ✓ Rendered to ${snapshot.name}.png\n`);
  
  await ctx.close();
}

await browser.close();
console.log('Done! All snapshots rendered.');
