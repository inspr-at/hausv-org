#!/usr/bin/env node
// Isolated fixture-only support lifecycle and responsive contract.
import assert from 'node:assert/strict';
import { existsSync, mkdirSync } from 'node:fs';
import { chromium } from 'playwright';
const base = process.argv[2];
if (!base || !['localhost','127.0.0.1','hausv.test'].includes(new URL(base).hostname)) throw new Error('Local QA fixture URL required');
const executablePath = [process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH,...(process.env.CI==='true'?[]:['/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'])].filter(Boolean).find(existsSync);
const browser = await chromium.launch({headless:true,...(executablePath?{executablePath}:{}),args:['--host-resolver-rules=MAP hausv.test 127.0.0.1','--no-proxy-server']});
const context = await browser.newContext();
const page = await context.newPage();
const artifactDir = process.env.HV_QA_ARTIFACT_DIR;
if (artifactDir) mkdirSync(artifactDir,{recursive:true});
try {
 await page.goto(base+'/');
 const disclosure=page.locator('details:has(form[action$="/auth/request"])');
 if (await disclosure.count()) await disclosure.evaluate(el=>el.open=true);
 await page.locator('input[name="email"]').fill('admin@example.com');
 await page.locator('form[action$="/auth/request"] button[type="submit"]').click();
 const dev=page.locator('a.dev-link');await dev.waitFor();
 const target = new URL(await dev.getAttribute('href'),base);target.protocol=new URL(base).protocol;target.port=new URL(base).port;
 await page.goto(target.href);
 const origin=new URL(page.url()).origin;
 const appBase=origin+'/demo';
 await page.goto(appBase+'/app');
 await page.locator('.desktop-context-bar [popovertarget]').filter({hasText:'Ansicht als'}).click();
 await page.locator('[data-support-view-link]:visible').click();
 assert.match(page.url(),/\/app\/support-view$/);
 const choice=page.locator('[data-support-view-start]').filter({has:page.locator('input[value="resident@example.com"]')});
 await choice.getByRole('button').click();
 await page.locator('[data-support-view-banner]').waitFor();
 assert.equal(await page.locator('[data-support-view-link]').count(),0);
 for (const width of [320,390,768,1440]) {
  await page.setViewportSize({width,height:900});
  for (const path of ['/app','/app/events','/app/settings']) {
   await page.goto(appBase+path);
   const banner=page.locator('[data-support-view-banner]');
   assert.equal(await banner.count(),1);
   assert.match(await banner.innerText(),/Portal anzeigen als Rita Bewohnerin · Bewohner/);
   await page.evaluate(()=>window.scrollTo(0,1200));
   await page.waitForTimeout(80);
   const box=await banner.boundingBox();const bar=await page.locator('[data-context-bar]:visible').boundingBox();
   assert.ok(box && Math.abs(box.y)<=1 && box.width<=width+1,`banner ${width} ${path}`);
   assert.ok(bar && Math.abs(bar.y-(box.y+box.height))<=1,`bar overlaps banner ${width} ${path}`);
   assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth+1),false,`overflow ${width} ${path}`);
   assert.ok((await banner.getByRole('button').boundingBox()).height>=44);
   if (path==='/app' && artifactDir) await page.screenshot({path:`${artifactDir}/support-${width}.png`});
  }
 }
 const blocked=await context.request.post(appBase+'/app/settings/profile',{form:{first_name:'Changed'},headers:{Origin:origin},maxRedirects:0});assert.equal(blocked.status(),403);
 const foreign=await context.request.get(origin+'/haus-b/app',{maxRedirects:0});assert.equal(foreign.status(),303);
 assert.equal(await page.locator('a[href*="/calendar/"]').count(),0);
 await page.locator('[data-support-view-banner]').getByRole('button').click();
 assert.equal(await page.locator('[data-support-view-banner]').count(),0);
 console.log('PASS support view: explicit grant, chooser, target identity, 4 widths × 3 routes, sticky banner, writes denied, tenant isolation, exit');
} finally { await browser.close(); }
