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
// The fixture allows five logins per account and quarter hour; reuse the
// session cookies of an account after its first login.
const sessions=new Map();
async function login(context,email) {
  const page=await context.newPage();
  if (sessions.has(email)) {
    await context.addCookies(sessions.get(email));
    await page.goto(`${baseURL}/app`,{waitUntil:'networkidle'});
    if (new URL(page.url()).pathname.includes('/app')) return page;
  }
  await page.goto(`${baseURL}/`,{waitUntil:'networkidle'});
  await page.locator('details:has(form[action$="/auth/request"])').evaluateAll(nodes => nodes.forEach(node => {node.open=true;}));
  await page.locator('input[name="email"]').fill(email);
  await page.locator('form[action$="/auth/request"] button[type="submit"]').click();
  const link=page.locator('a.dev-link'); await link.waitFor({state:'visible'});
  const target=new URL(await link.getAttribute('href'),baseURL); const origin=new URL(baseURL);target.protocol=origin.protocol;target.port=origin.port;
  await page.goto(target.href,{waitUntil:'networkidle'});assert(new URL(page.url()).pathname.includes('/app'),'login failed');
  sessions.set(email,(await context.storageState()).cookies);return page;
}
const visibleBar=page => page.locator('[data-context-bar]:visible');
async function quietChevron(page,picker,label) {
  const summary=picker.locator(':scope > summary');
  const indicator=summary.locator(':scope > .disclosure-chevron');
  const opacity=async value=>{
    await page.waitForFunction(({id,value})=>getComputedStyle(document.getElementById(id).querySelector(':scope > summary > .disclosure-chevron')).opacity===value,{id:await picker.getAttribute('id'),value});
  };
  await page.mouse.move(0,899);
  await summary.evaluate(element=>element.blur());
  await opacity('0.4');
  const resting=await indicator.evaluate(element=>getComputedStyle(element).backgroundColor);
  await summary.hover();await opacity('1');
  await page.mouse.move(0,899);await page.keyboard.press('Tab');await summary.focus();await opacity('1');
  await summary.press('Enter');await opacity('1');
  await page.waitForFunction(id=>{
    const svg=document.getElementById(id).querySelector(':scope > summary > .disclosure-chevron svg');
    return Math.abs(new DOMMatrixReadOnly(getComputedStyle(svg).transform).a+1)<.001;
  },await picker.getAttribute('id'));
  assert.notEqual(await indicator.evaluate(element=>getComputedStyle(element).backgroundColor),resting,`${label}: open fill becomes lighter`);
  await page.keyboard.press('Escape');
  await summary.evaluate(element=>element.blur());await opacity('0.4');
  checks.push(`${label}: quiet/hover/focus/open chevron`);
}
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
    const names=[...document.querySelectorAll('[data-switcher] > summary .house-header-copy strong, [data-switcher][open] .switcher-row-copy strong, [data-switcher][open] .switcher-option strong')].filter(visible).map(e=>({summary:!!e.closest('summary'),width:e.getBoundingClientRect().width,lines:getComputedStyle(e).webkitLineClamp,white:getComputedStyle(e).whiteSpace,overflow:getComputedStyle(e).overflow,textOverflow:getComputedStyle(e).textOverflow,text:e.textContent}));
    const pills=[...document.querySelectorAll('.context-bar .context-account > summary,.context-bar .context-scope summary')].filter(visible).map(e=>{
      const box=e.getBoundingClientRect(),style=getComputedStyle(e),paint=getComputedStyle(e,'::before');
      const text=e.querySelector('.house-header-copy strong'),circle=e.querySelector('.disclosure-chevron');
      const center=element=>{const r=element.getBoundingClientRect();return r.top+r.height/2;};
      return {width:box.width,height:box.height,paintHeight:box.height-parseFloat(paint.top)-parseFloat(paint.bottom),gap:style.gap,paddingLeft:style.paddingLeft,scope:!!text,centered:!text||Math.abs(center(text)-center(circle))<1};
    });
    return {left,overflow:document.documentElement.scrollWidth>innerWidth+1,navs:[...document.querySelectorAll('nav[aria-label="Bereiche"]')].filter(visible).length,names,pills};
  });
  assert.deepEqual(probe.left,[],`${label}: element left of viewport`);assert(!probe.overflow,`${label}: horizontal overflow`);
  assert(probe.names.length,`${label}: scope text missing`);
  for(const pill of probe.pills){
    assert(pill.width>=44&&pill.height>=44,`${label}: context summary has a 44px tap target`);
    assert.equal(pill.paintHeight,32,`${label}: visible context pill is 32px high`);
    assert.equal(pill.gap,'10px',`${label}: context elements have 10px spacing`);
    if(pill.scope){assert.equal(pill.paddingLeft,'14px',`${label}: scope left inset`);assert(pill.centered,`${label}: text and chevron vertically centred`);}
  }
  for(const name of probe.names){
    assert(name.width>=Math.min(100, name.text.trim().length*5),`${label}: name reduced to a few characters (${JSON.stringify(name)})`);
    if(name.summary){
      assert.equal(name.white,'nowrap',`${label}: HAUSV-715 summary stays on one line`);
      assert.equal(name.overflow,'hidden',`${label}: long summary stays within its slot`);
      assert.equal(name.textOverflow,'ellipsis',`${label}: long summary uses ellipsis`);
    }else{
      assert.equal(name.lines,'2',`${label}: results allow two lines`);
      assert.notEqual(name.white,'nowrap',`${label}: result name must wrap`);
    }
  }
  if(width>760) {assert.equal(probe.navs,1,`${label}: exactly one desktop navigation`);await alignedNavigation(page,label);}
}
async function panelGeometry(page,picker,label) {
  const panel=picker.locator('.switcher-panel');
  await panel.waitFor({state:'visible'});
  const pickerID=await picker.getAttribute('id');
  await page.waitForFunction(id=>{
    const panel=document.getElementById(id)?.querySelector('.switcher-panel');
    return panel && panel.scrollTop===0 && (typeof panel.showPopover!=='function'||panel.matches(':popover-open'));
  },pickerID);
  const state=await panel.evaluate(element => {
    const box=element.getBoundingClientRect();
    const heading=element.querySelector('.switcher-heading').getBoundingClientRect();
    const current=element.querySelector('[data-switcher-current]')?.getBoundingClientRect();
    const bar=parseFloat(getComputedStyle(document.documentElement).getPropertyValue('--context-bar-h'));
    const card=element.parentElement.querySelector(':scope > summary').getBoundingClientRect();
    const topmost=document.elementFromPoint(heading.left+8,heading.top+8);
    return {top:box.top,bottom:box.bottom,left:box.left,right:box.right,height:innerHeight,width:innerWidth,bar,cardTop:card.top,sidebar:!!element.closest('.sidebar'),
      headingTop:heading.top,headingBottom:heading.bottom,currentTop:current?.top,currentBottom:current?.bottom,
      scrollTop:element.scrollTop,overflow:getComputedStyle(element).overflowY,headingExposed:element.contains(topmost),
      topLayer:typeof element.showPopover!=='function'||element.matches(':popover-open')};
  });
  assert(state.top>=state.bar+8-1,`${label}: panel clears the context bar (${JSON.stringify(state)})`);
  assert(state.bottom<=state.height-7,`${label}: panel ends inside viewport`);
  assert(state.left>=0 && state.right<=state.width,`${label}: panel fits horizontally`);
  assert(state.headingTop>=state.top && state.headingBottom<=state.bottom,`${label}: heading fully visible`);
  if(state.currentTop!==undefined) assert(state.currentTop>=state.top && state.currentBottom<=state.bottom,`${label}: current entry fully visible`);
  assert.equal(state.scrollTop,0,`${label}: opens at panel heading`);
  assert.equal(state.overflow,'auto',`${label}: internal scrolling`);
  assert(state.headingExposed && state.topLayer,`${label}: heading not clipped or covered by sidebar/drawer`);
  if(state.sidebar && state.cardTop>=state.bar+8 && state.cardTop<=state.height-328) assert(Math.abs(state.top-state.cardTop)<=1,`${label}: panel aligns with card`);
  const indicator=picker.locator(':scope > summary > .disclosure-chevron');
  assert.equal(await indicator.count(),1,`${label}: one shared disclosure indicator`);
  const circle=await indicator.boundingBox();assert.equal(circle.width,18,`${label}: 18px chevron`);assert.equal(circle.height,18,`${label}: round chevron`);
  const icon=await indicator.locator('svg').evaluate(svg=>{const cs=getComputedStyle(svg);return {width:parseFloat(cs.width),height:parseFloat(cs.height)};});assert.equal(icon.width,12,`${label}: 12px icon`);assert.equal(icon.height,12,`${label}: square icon`);
  const target=await picker.locator(':scope > summary').boundingBox();assert(target.height>=44 && target.width>=44,`${label}: summary tap target`);
  await page.waitForFunction(id=>{
    const svg=document.getElementById(id)?.querySelector(':scope > summary > .disclosure-chevron svg');
    return svg && Math.abs(new DOMMatrixReadOnly(getComputedStyle(svg).transform).a+1)<.001 && getComputedStyle(svg.closest('.disclosure-chevron')).opacity==='1';
  },pickerID);
  await panel.evaluate(element=>{element.scrollTop=element.scrollHeight;});
  await page.keyboard.press('Escape');assert.equal(await picker.getAttribute('open'),null,`${label}: Escape closes picker`);
  await picker.locator(':scope > summary').click();
  await panel.waitFor({state:'visible'});
  await page.waitForFunction(id=>document.getElementById(id)?.querySelector('.switcher-panel').scrollTop===0,pickerID);
  assert.equal(await panel.evaluate(element=>element.scrollTop),0,`${label}: reopening restores the heading after scrolling`);
}
async function checkRoute(page,width,path,label) {
  await page.goto(`${baseURL}${path}`,{waitUntil:'networkidle'});
  assert.equal((await page.locator('main').count())>0,true,`${label}: route body`);
  await geometry(page,width,label);
  if (path.includes('/app/verwaltung')) {
    const sidebar = page.locator('aside.sidebar');
    assert.match(await sidebar.locator('.nav-house-card .house-header-copy strong').textContent(), /^Alle Liegenschaften · \d+$/, `${label}: overview card keeps property count`);
    assert.equal(await sidebar.locator('.nav-house-items').count(), 1, `${label}: house navigation persists in organisation scope`);
    assert.equal(await sidebar.locator('.side-map-portfolio').count(), 1, `${label}: neutral portfolio map`);
  }
  await page.evaluate(()=>window.scrollTo(0,document.body.scrollHeight));await geometry(page,width,`${label} scrolled`);
  await page.evaluate(()=>window.scrollTo(0,0));
  if(path==='/app/verwaltung/posteingang') {
    for(const selector of ['.filter-panel','.phone-panel']) {
      const panel=page.locator(`.queue-tools ${selector}`);
      assert.equal(await panel.evaluate(element=>element.getClientRects().length),0,`${label}: closed ${selector} has no layout box`);
      const trigger=panel.locator('..').locator(':scope > summary');
      await trigger.click();
      await panel.waitFor({state:'visible'});
      const box=await panel.boundingBox();
      assert(box.x>=0 && box.x+box.width<=width,`${label}: open ${selector} fits horizontally (${JSON.stringify(box)})`);
      await geometry(page,width,`${label} ${selector}`);
      const field=panel.locator('select').first();
      assert(await field.evaluate(element=>{
        const box=element.getBoundingClientRect();
        return document.elementFromPoint(box.left+box.width/2,box.top+box.height/2)===element;
      }),`${label}: ${selector} controls are exposed`);
      await trigger.click();
    }
  }
  const segment=visibleBar(page).locator('.context-scope [data-switcher]').last();
  await segment.locator(':scope > summary').click();await segment.locator('.switcher-panel').waitFor({state:'visible'});
  assert.equal(await segment.locator('input[role="combobox"]').count(),1);
  await panelGeometry(page,segment,`${label} context panel`);
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
    const picker=menu.locator('[data-switcher]').first();await picker.locator(':scope > summary').click();
    await panelGeometry(page,picker,`${label} drawer panel`);
    await page.keyboard.press('Escape');
    await menu.locator(':scope > summary').click();
  }else{
    const picker=page.locator('aside.sidebar [data-switcher]').first();await picker.locator(':scope > summary').click();
    await panelGeometry(page,picker,`${label} sidebar panel`);
    await page.keyboard.press('Escape');
  }
  checks.push(label);
}
async function checkPreview(width,role='Bewohner') {
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
    const resident=chooser.locator('form').filter({has:page.locator(`input[name="role"][value="${role}"]`)});
    await resident.getByRole('button').click();
    await bar.locator(`[data-preview-role="${role}"]`).waitFor({state:'attached'});
    assert(await bar.evaluate(element=>element.classList.contains('context-preview')),'preview tints the context bar');
    assert.notEqual(await bar.evaluate(element=>getComputedStyle(element).backgroundColor),background,'preview background differs');
    if(width<=760) await bar.locator(':scope > details.menu > summary').click();
    const picker=page.locator(width<=760?'.menu-panel [data-switcher]':'aside.sidebar [data-switcher]').first();
    await picker.locator(':scope > summary').click();await panelGeometry(page,picker,`${role} preview ${width}`);await page.keyboard.press('Escape');
    if(width<=760) await bar.locator(':scope > details.menu > summary').click();
    if(width<=760) await bar.locator('[data-context-account] > summary').click();
    const preview=bar.locator(`[data-preview-role="${role}"]`);
    assert.match(await preview.innerText(),new RegExp(`Vorschau: ${role}-Sicht\\s*·\\s*beenden`),'active preview names the view and exit');
    assert.equal(await bar.locator('.context-view-chip').count(),0,'active preview does not offer another preview');
    await preview.getByRole('button',{name:'Vorschau beenden',exact:true}).click();
    await bar.locator('.context-view-chip').waitFor({state:'attached'});
    assert.equal(await bar.locator('[data-preview-role]').count(),0,'ending preview restores the normal view');
    assert.equal(await bar.evaluate(element=>getComputedStyle(element).backgroundColor),background,'ending preview restores the background');
    assert.match(await bar.locator('.context-role').textContent(),/Admin/,'ending preview restores Admin');
    checks.push(`${role} preview choice/panel/start/end ${width}px`);
  } finally {await context.close();}
}
try {
  for(const persona of [{email:'resident@example.com',paths:['/app','/app/announcements','/app/settings']},{email:'owner@example.com',paths:['/app','/app/dokumente']},{email:'verwalter@example.com',paths:['/app','/app/verwaltung','/app/verwaltung/posteingang']},{email:'admin@example.com',paths:['/app','/app/verwaltung','/app/verwaltung/rechte','/app/verwaltung/einstellungen']}]){
    const context=await browser.newContext({viewport:{width:1440,height:900},locale:'de-AT'});
    try{
      const page=await login(context,persona.email);
      page.on('pageerror',error=>failures.push(`${persona.email}: ${error.message}`));
      await quietChevron(page,page.locator('aside.sidebar [data-switcher]').first(),persona.email+' sidebar');
      await quietChevron(page,visibleBar(page).locator('.context-scope [data-switcher]').last(),persona.email+' context');
      for(const width of [1440,1024,768,760,390]){
        await page.setViewportSize({width,height:900});
        for(const path of persona.paths) await checkRoute(page,width,path,`${persona.email} ${width} ${path}`);
      }
      await page.screenshot({path:join(out,`${persona.email.split('@')[0]}-390.png`),fullPage:false});
    }finally{await context.close();}
  }
  for(const width of [1440,390]) await checkPreview(width);
  for(const width of [1440,1024,768,390]) await checkPreview(width,'Eigentümer');
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
    assert.equal((await context.cookies()).find(cookie=>cookie.name==='hausv-sidebar-w')?.value,'360','width persisted for server rendering');
    await page.reload({waitUntil:'networkidle'});assert(Math.abs((await page.locator('aside.sidebar').boundingBox()).width-360)<=1,'persist width');
    await handle.focus();await page.keyboard.press('ArrowLeft');assert.equal(await handle.getAttribute('aria-valuenow'),'350');
    await handle.dblclick();assert.equal(await handle.getAttribute('aria-valuenow'),'280');
    // Short windows and an already scrolled sidebar formerly clipped the head.
    for(const width of [1440,1024,768,390]){
      await page.setViewportSize({width,height:560});
      if(width<=760) await visibleBar(page).locator(':scope > details.menu > summary').click();
      const picker=page.locator(width<=760?'.menu-panel [data-switcher]':'aside.sidebar [data-switcher]').first();
      await picker.locator(':scope > summary').click();await panelGeometry(page,picker,`short viewport ${width}`);await page.keyboard.press('Escape');
      if(width<=760) await visibleBar(page).locator(':scope > details.menu > summary').click();
    }
    checks.push('typeahead keyboard, atomic context POST, resize/persistence/reset/keyboard');
  }finally{await context.close();}
  const noJS=await browser.newContext({javaScriptEnabled:false,viewport:{width:390,height:900}});
  try{
    const page=await login(noJS,'multi@example.com');
    const segment=visibleBar(page).locator('.context-scope [data-switcher]').last();await segment.locator(':scope > summary').click();
    await segment.getByRole('combobox').fill('Haus B');await segment.getByRole('button',{name:'Suchen',exact:true}).click();await page.waitForLoadState('networkidle');
    assert.match(await page.locator('main').innerText(),/Haus B/);checks.push('no-JS GET search');
  }finally{await noJS.close();}
  const reduced=await browser.newContext({reducedMotion:'reduce',viewport:{width:1440,height:900}});
  try{
    const page=await login(reduced,'owner@example.com');
    const picker=page.locator('aside.sidebar [data-switcher]').first();await picker.locator(':scope > summary').click();
    await panelGeometry(page,picker,'reduced motion');
    const durations=await picker.locator(':scope > summary .disclosure-chevron, :scope > summary .disclosure-chevron svg').evaluateAll(elements=>elements.map(element=>getComputedStyle(element).transitionDuration));
    assert(durations.length===2 && durations.every(value=>value==='0s'),'reduced motion disables circle and rotation transitions');
    checks.push('reduced-motion chevron');
  }finally{await reduced.close();}
  assert.deepEqual(failures,[],'browser errors');
}catch(error){failures.push(error.stack||String(error));process.exitCode=1;}
finally{await browser.close();writeFileSync(join(out,'qa-switcher.json'),JSON.stringify({checks,failures},null,2));}
console.log(`${checks.length} Prüfungen; ${failures.length} Fehler`);if(failures.length)console.error(failures.join('\n'));
