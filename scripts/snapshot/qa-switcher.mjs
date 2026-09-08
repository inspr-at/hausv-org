#!/usr/bin/env node
// HV_CAPTURE=qa-switcher.mjs scripts/snapshot/run.sh WORKTREE <out-dir> [port]
// Uses the same deterministic personas and email-link login as qa-main-flows.
import assert from 'node:assert/strict';
import { existsSync, mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { chromium } from 'playwright';
const [baseURL, out = '/private/tmp/hausv-switcher-qa'] = process.argv.slice(2);
if (!baseURL) throw new Error('usage: qa-switcher.mjs <baseURL> [out-dir]');
mkdirSync(out, {recursive:true});
const executablePath = [process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH, ...(process.env.CI === 'true' ? [] : ['/Applications/Google Chrome.app/Contents/MacOS/Google Chrome','/Applications/Chromium.app/Contents/MacOS/Chromium','/usr/bin/chromium'])].filter(Boolean).find(existsSync);
const browser = await chromium.launch({headless:true, ...(executablePath ? {executablePath} : {}), args:['--host-resolver-rules=MAP hausv.test 127.0.0.1, MAP *.hausv.test 127.0.0.1','--no-proxy-server']});
const failures=[], checks=[];
async function login(context,email) {
  const page=await context.newPage();
  await page.goto(`${baseURL}/`,{waitUntil:'networkidle'});
  await page.locator('details:has(form[action$="/auth/request"])').evaluateAll(nodes => nodes.forEach(node => {node.open=true;}));
  await page.locator('input[name="email"]').fill(email);
  await page.locator('form[action$="/auth/request"] button[type="submit"]').click();
  const link=page.locator('a.dev-link'); await link.waitFor({state:'visible'});
  const target=new URL(await link.getAttribute('href'),baseURL); const origin=new URL(baseURL);target.protocol=origin.protocol;target.port=origin.port;
  await page.goto(target.href,{waitUntil:'networkidle'});assert(new URL(page.url()).pathname.includes('/app'),'login failed');return page;
}
const visibleBar=page => page.locator('[data-context-bar]:visible');
async function alignedNavigation(page,label) {
  const positions=await page.locator('nav[aria-label="Bereiche"]:visible').evaluateAll(navs => navs.flatMap(nav => {
    const organisation=nav.querySelector('.nav-organisation');
    const house=nav.querySelector('.nav-house-items');
    if(!organisation || !house) return [];
    return [...organisation.querySelectorAll(':scope > a'),...house.querySelectorAll(':scope > a')].map(link => {
      const content=link.querySelector('.nav-icon') || link.querySelector('.nav-label') || link;
      return {label:link.textContent.trim(),x:content.getBoundingClientRect().x};
    });
  }));
  for(const position of positions) assert(Math.abs(position.x-positions[0].x)<=1,`${label}: navigation must align (${JSON.stringify(positions)})`);
}
async function geometry(page,width,label) {
  assert.equal(await visibleBar(page).count(),1,`${label}: one visible context bar`);
  const box=await visibleBar(page).boundingBox();assert(Math.abs(box.y)<=1,`${label}: sticky bar top`);assert.equal(Math.round(box.height),width<=760?74:40,`${label}: context height`);
  const probe=await page.evaluate(() => {
    const visible=e=>!!e.getClientRects().length && getComputedStyle(e).visibility!=='hidden';
    // OSM tile images deliberately extend inside a clipped map; all controls,
    // panels, text and their containers must stay inside the viewport.
    const left=[...document.querySelectorAll('body *')].filter(e=>visible(e)&& !e.closest('.sr-only,.skip-link,.side-map-tiles') && e.getBoundingClientRect().left < -1).map(e=>`${e.tagName}.${e.className}`);
    const names=[...document.querySelectorAll('[data-switcher] > summary .house-header-copy strong, [data-switcher][open] .switcher-row-copy strong, [data-switcher][open] .switcher-option strong')].filter(visible).map(e=>({width:e.getBoundingClientRect().width,lines:getComputedStyle(e).webkitLineClamp,white:getComputedStyle(e).whiteSpace,text:e.textContent}));
    return {left,overflow:document.documentElement.scrollWidth>innerWidth+1,navs:[...document.querySelectorAll('nav[aria-label="Bereiche"]')].filter(visible).length,names};
  });
  assert.deepEqual(probe.left,[],`${label}: element left of viewport`);assert(!probe.overflow,`${label}: horizontal overflow`);
  assert(probe.names.length,`${label}: scope text missing`);
  for(const name of probe.names){assert(name.width>=Math.min(100, name.text.trim().length*5),`${label}: name reduced to a few characters (${JSON.stringify(name)})`);assert.equal(name.lines,'2',`${label}: allow two lines`);assert.notEqual(name.white,'nowrap',`${label}: name must wrap`);}
  if(width>760) {assert.equal(probe.navs,1,`${label}: exactly one desktop navigation`);await alignedNavigation(page,label);}
}
async function checkRoute(page,width,path,label) {
  await page.goto(`${baseURL}${path}`,{waitUntil:'networkidle'});
  assert.equal((await page.locator('main').count())>0,true,`${label}: route body`);
  await geometry(page,width,label);
  await page.evaluate(()=>window.scrollTo(0,document.body.scrollHeight));await geometry(page,width,`${label} scrolled`);
  await page.evaluate(()=>window.scrollTo(0,0));
  const segment=visibleBar(page).locator('.context-scope [data-switcher]').last();
  await segment.locator(':scope > summary').click();await segment.locator('.switcher-panel').waitFor({state:'visible'});
  assert.equal(await segment.locator('input[role="combobox"]').count(),1);
  await geometry(page,width,`${label} panel`);
  await page.keyboard.press('Escape');assert.equal(await segment.getAttribute('open'),null);
  const account=visibleBar(page).locator(':scope > [data-context-account]');await account.locator(':scope > summary').click();
  const logout=account.getByRole('button',{name:'Abmelden',exact:true});assert(await logout.isVisible(),`${label}: explicit logout text`);
  assert(await account.getByRole('link',{name:'Profil',exact:true}).isVisible(),`${label}: profile link`);
  await account.locator(':scope > summary').click();
  if(width<=760){
    const menu=visibleBar(page).locator(':scope > details.menu');await menu.locator(':scope > summary').click();
    assert.equal(await page.locator('nav[aria-label="Bereiche"]:visible').count(),1,`${label}: exactly one mobile navigation`);
    await alignedNavigation(page,`${label} drawer`);
    await menu.locator(':scope > summary').click();
  }
  checks.push(label);
}
async function checkPreview(width) {
  const context=await browser.newContext({viewport:{width,height:900},locale:'de-AT'});
  try {
    const page=await login(context,'admin@example.com');
    await page.goto(`${baseURL}/app`,{waitUntil:'networkidle'});
    const bar=visibleBar(page);
    const background=await bar.evaluate(element=>getComputedStyle(element).backgroundColor);
    assert.equal(await bar.locator('[data-preview-role]').count(),0,'normal view has no active preview');
    if(width<=760) await bar.locator('[data-context-account] > summary').click();
    assert.equal((await bar.locator('.context-view-chip').textContent()).trim(),'Ansicht als …','normal view offers a role choice'); // textContent: the chip is uppercased by CSS
    await bar.locator('.role-preview-trigger').click();
    const chooser=page.locator('.role-preview-menu:popover-open');
    assert(await chooser.isVisible(),'role choice opens');
    const resident=chooser.locator('form').filter({has:page.locator('input[name="role"][value="Bewohner"]')});
    await resident.getByRole('button').click();
    await bar.locator('[data-preview-role="Bewohner"]').waitFor({state:'attached'});
    assert(await bar.evaluate(element=>element.classList.contains('context-preview')),'preview tints the context bar');
    assert.notEqual(await bar.evaluate(element=>getComputedStyle(element).backgroundColor),background,'preview background differs');
    if(width<=760) await bar.locator('[data-context-account] > summary').click();
    const preview=bar.locator('[data-preview-role="Bewohner"]');
    assert.match(await preview.innerText(),/Vorschau: Bewohner-Sicht\s*·\s*beenden/,'active preview names the view and exit');
    assert.equal(await bar.locator('.context-view-chip').count(),0,'active preview does not offer another preview');
    await preview.getByRole('button',{name:'Vorschau beenden',exact:true}).click();
    await bar.locator('.context-view-chip').waitFor({state:'attached'});
    assert.equal(await bar.locator('[data-preview-role]').count(),0,'ending preview restores the normal view');
    assert.equal(await bar.evaluate(element=>getComputedStyle(element).backgroundColor),background,'ending preview restores the background');
    assert.match(await bar.locator('.context-role').textContent(),/Admin/,'ending preview restores Admin');
    checks.push(`preview choice/start/end ${width}px`);
  } finally {await context.close();}
}
try {
  for(const persona of [{email:'resident@example.com',paths:['/app','/app/announcements','/app/settings']},{email:'owner@example.com',paths:['/app','/app/dokumente']},{email:'verwalter@example.com',paths:['/app','/app/verwaltung','/app/verwaltung/posteingang']},{email:'admin@example.com',paths:['/app','/app/verwaltung','/app/verwaltung/rechte','/app/verwaltung/einstellungen']}]){
    const context=await browser.newContext({viewport:{width:1440,height:900},locale:'de-AT'});
    try{
      const page=await login(context,persona.email);
      page.on('pageerror',error=>failures.push(`${persona.email}: ${error.message}`));
      for(const width of [1440,1024,760,390]){
        await page.setViewportSize({width,height:900});
        for(const path of persona.paths) await checkRoute(page,width,path,`${persona.email} ${width} ${path}`);
      }
      await page.screenshot({path:join(out,`${persona.email.split('@')[0]}-390.png`),fullPage:false});
    }finally{await context.close();}
  }
  for(const width of [1440,390]) await checkPreview(width);
  const context=await browser.newContext({viewport:{width:1440,height:900},locale:'de-AT'});
  try{
    const page=await login(context,'multi@example.com');await page.goto(`${baseURL}/demo/app`,{waitUntil:'networkidle'});
    const picker=page.locator('#portal-house-picker');await picker.locator(':scope > summary').click();
    const input=picker.getByRole('combobox');await input.fill('Haus B');
    await picker.locator('[role="option"]').first().waitFor();assert((await picker.locator('[role="option"]').count())<=8);
    await input.press('ArrowDown');const active=await input.getAttribute('aria-activedescendant');assert(active,'combobox active descendant');
    await input.press('ArrowUp');assert(await input.getAttribute('aria-activedescendant'),'up navigation');await input.press('ArrowDown');assert.equal(await input.getAttribute('aria-activedescendant'),active,'return to first option');
    await input.press('Enter');await page.waitForURL('**/haus-b/app');await page.waitForLoadState('networkidle');
    assert.match(await visibleBar(page).locator('.context-scope').innerText(),/Haus B/);
    assert.match(await visibleBar(page).locator('.context-role').innerText(),/Admin/);
    // The group row is the same POST context switch, independent of typeahead.
    await page.locator('#portal-house-picker > summary').click();
    const back=page.locator('#portal-house-picker [data-switcher-initial] form').filter({has:page.locator('input[name="tenant"][value="demo"]')}).first();
    assert(await back.count(),'demo portal group entry');await back.getByRole('button').click();await page.waitForURL('**/demo/app');
    const handle=page.locator('[data-sidebar-resize]');const drag=await handle.boundingBox();
    await page.mouse.move(drag.x+drag.width/2,drag.y+100);await page.mouse.down();await page.mouse.move(360,drag.y+100,{steps:12});await page.mouse.up();
    assert(Math.abs((await page.locator('aside.sidebar').boundingBox()).width-360)<=1,'drag width');
    await page.reload({waitUntil:'networkidle'});assert(Math.abs((await page.locator('aside.sidebar').boundingBox()).width-360)<=1,'persist width');
    await handle.focus();await page.keyboard.press('ArrowLeft');assert.equal(await handle.getAttribute('aria-valuenow'),'350');
    await handle.dblclick();assert.equal(await handle.getAttribute('aria-valuenow'),'280');
    checks.push('typeahead keyboard, atomic context POST, resize/persistence/reset/keyboard');
  }finally{await context.close();}
  const noJS=await browser.newContext({javaScriptEnabled:false,viewport:{width:390,height:900}});
  try{
    const page=await login(noJS,'multi@example.com');
    const segment=visibleBar(page).locator('.context-scope [data-switcher]').last();await segment.locator(':scope > summary').click();
    await segment.getByRole('combobox').fill('Haus B');await segment.getByRole('button',{name:'Suchen',exact:true}).click();await page.waitForLoadState('networkidle');
    assert.match(await page.locator('main').innerText(),/Haus B/);checks.push('no-JS GET search');
  }finally{await noJS.close();}
  assert.deepEqual(failures,[],'browser errors');
}catch(error){failures.push(error.stack||String(error));process.exitCode=1;}
finally{await browser.close();writeFileSync(join(out,'qa-switcher.json'),JSON.stringify({checks,failures},null,2));}
console.log(`${checks.length} Prüfungen; ${failures.length} Fehler`);if(failures.length)console.error(failures.join('\n'));
