#!/usr/bin/env node
// Navigation geometry is sampled from parser frames, not only finished pages.
import assert from 'node:assert/strict';
import {existsSync,mkdirSync,writeFileSync} from 'node:fs';
import {chromium} from 'playwright';
const base=process.argv[2],out=process.env.HV_QA_ARTIFACT_DIR||'/tmp/hausv-unified-context';
assert(['localhost','127.0.0.1','hausv.test'].includes(new URL(base).hostname));mkdirSync(out,{recursive:true});
const executablePath=[process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH,...(process.env.CI==='true'?[]:['/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'])].filter(Boolean).find(existsSync);
const browser=await chromium.launch({headless:true,...(executablePath?{executablePath}:{}),args:['--host-resolver-rules=MAP hausv.test 127.0.0.1','--no-proxy-server']});
const context=await browser.newContext({viewport:{width:1440,height:900}}),page=await context.newPage(),report=[];
await context.addInitScript(()=>{
 localStorage.setItem('hausv:version-display','reduced');
 window.__chromeFrames=[];
 const frame=()=>{const bar=document.querySelector('[data-context-bar]'),content=document.querySelector('.portal-shell-content');
 if(bar&&content){const b=bar.getBoundingClientRect(),c=content.getBoundingClientRect();window.__chromeFrames.push([b.x,b.y,b.width,b.height,c.x,c.y]);}
 if(window.__chromeFrames.length<100)requestAnimationFrame(frame);};requestAnimationFrame(frame);
});
try{
 await page.goto(base+'/');await page.locator('details:has(form[action$="/auth/request"])').evaluateAll(es=>es.forEach(e=>e.open=true));
 await page.locator('input[name="email"]').fill('admin@example.com');await page.locator('form[action$="/auth/request"] button[type="submit"]').click();
 const dev=page.locator('a.dev-link');await dev.waitFor();const url=new URL(await dev.getAttribute('href'),base);url.protocol=new URL(base).protocol;url.port=new URL(base).port;await page.goto(url.href,{waitUntil:'networkidle'});
 const app=new URL(page.url()).origin+'/demo';
 for(const mode of ['normal','preview','support']){
  if(mode==='preview'){await page.goto(app+'/app');await page.locator('.role-preview-trigger').click();await page.locator('form[action$="/app/ansicht/start"]').filter({has:page.locator('input[value="Eigentümer"]')}).getByRole('button').click();}
  if(mode==='support'){await page.goto(app+'/app/support-view');await page.locator('[data-support-view-start]').filter({has:page.locator('input[value="resident@example.com"]')}).getByRole('button').click();}
  for(const width of [320,390,768,1440]){
   await page.setViewportSize({width,height:900});
   const expectedHeight=width<=760?148:width<=1100?116:72;
   for(const route of ['/app','/app/announcements','/app/events','/app/kontakte','/app/dokumente','/app/anliegen','/app/abstimmungen','/app/settings','/app/hilfe']){
    await page.goto(app+route,{waitUntil:'networkidle'});await page.waitForTimeout(180);
    const frames=await page.evaluate(()=>window.__chromeFrames);assert(frames.length>=2,`${mode}/${width}/${route}: frame samples`);
    for(const frame of frames){assert(Math.abs(frame[1])<=.5,`header top ${frame}`);assert(Math.abs(frame[3]-expectedHeight)<=.5,`height changed ${frame}`);assert(Math.abs(frame[5]-expectedHeight)<=.5,`content shifted ${frame}`);assert(Math.abs(frame[4]-(width<=760?0:280))<=.5,`content x changed ${frame}`);}
    if(width>760){
     const map=await page.locator('aside.sidebar .side-map-hero').boundingBox();
     assert(map && Math.abs(map.y-expectedHeight)<=.5,`${mode}/${width}/${route}: map touches header`);
    }
    assert.equal(await page.locator('[data-context-bar]').count(),1);assert.equal(await page.locator('[data-house-picker]').count(),1);
    assert.match(await page.locator('[data-context-account] summary').innerText(),/Ada Admin/);
    assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth+1),false);
    report.push({mode,width,route,frames:frames.length,geometry:frames.at(-1)});
    if(route==='/app/dokumente')await page.screenshot({path:`${out}/context-${mode}-${width}.png`});
   }
  }
  if(mode!=='normal')await page.getByRole('button',{name:'Ansicht verlassen',exact:true}).click();
 }
 await page.setViewportSize({width:1440,height:900});await page.goto(app+'/app',{waitUntil:'networkidle'});
 await context.grantPermissions(['clipboard-read','clipboard-write'],{origin:new URL(page.url()).origin});
 const footer=page.locator('aside.sidebar [data-product-version]');await footer.scrollIntoViewIfNeeded();await footer.waitFor();
 const canonical=await footer.getAttribute('data-product-version');assert.match(canonical,/^[1-9][0-9]{11}\.0\.0$/);
 const historyLink=page.locator('aside.sidebar .release-trigger');
 assert.match(await historyLink.innerText(),/^Version:/);
 assert.equal(await historyLink.locator('[role="button"]').count(),0,'one native link, no nested copy action');
 await historyLink.focus();await historyLink.press('Enter');assert(await page.locator('#release-history').isVisible());
 assert.equal(await page.locator('[data-version-display]').count(),0,'no display selector');
 assert((await footer.locator('.separator').count())>0,'Pretty ignores the old reduced preference');
 await page.emulateMedia({reducedMotion:'reduce'});const stamp=page.locator('#release-history [data-product-version]').first();await stamp.focus();await stamp.press('Enter');assert.equal(await page.evaluate(()=>navigator.clipboard.readText()),canonical);
 assert.equal(await page.locator('#release-history [data-version-scheme="legacy"]').first().innerText(),'1.11.0');
 const transitionErrors=[];page.on('pageerror',error=>transitionErrors.push(error.message));
 await context.addInitScript(()=>{
  window.__revealedTransition=false;
  window.addEventListener('pagereveal',event=>{if(event.viewTransition){window.__revealedTransition=true;event.viewTransition.skipTransition();}});
 });
 await page.goto(app+'/app',{waitUntil:'networkidle'});
 for(const [motion,route] of [['reduce','dokumente'],['no-preference','events']]){
  await page.emulateMedia({reducedMotion:motion});
  await page.locator(`aside.sidebar a[href$="/app/${route}"]`).click();await page.waitForLoadState('networkidle');
  assert.equal(await page.evaluate(()=>window.__revealedTransition),motion==='no-preference',`${motion}: transition opt-in`);
  assert(await page.locator('main').first().isVisible(),'navigation remains usable after skipped transition');
 }
 assert.deepEqual(transitionErrors,[],'optional transition cancellation must not reject unhandled');
 writeFileSync(`${out}/context-geometry.json`,JSON.stringify(report,null,2));console.log(`PASS unified context: ${report.length} route/mode/viewport cases, parser-frame geometry, real identity, one picker, Pretty only despite old preference, exact clipboard in history, version link opens history, reduced motion, real transition cancellation`);
}finally{await browser.close();}
