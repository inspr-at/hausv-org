#!/usr/bin/env node
// Real local fixture records reproduce the long location that hid the subject.
import assert from 'node:assert/strict';
import {existsSync,mkdirSync,writeFileSync,readFileSync} from 'node:fs';
import {chromium} from 'playwright';
const [base,out='/tmp/hausv-calm-overview']=process.argv.slice(2);
assert(['localhost','127.0.0.1','hausv.test'].includes(new URL(base).hostname));mkdirSync(out,{recursive:true});
const executablePath=[process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH,...(process.env.CI==='true'?[]:['/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'])].filter(Boolean).find(existsSync);
const browser=await chromium.launch({headless:true,...(executablePath?{executablePath}:{}),args:['--host-resolver-rules=MAP hausv.test 127.0.0.1','--no-proxy-server']});
const context=await browser.newContext(),page=await context.newPage(),report=[];
try{
 await page.goto(base+'/');await page.locator('details:has(form[action$="/auth/request"])').evaluateAll(es=>es.forEach(e=>e.open=true));
 await page.locator('input[name="email"]').fill('owner@example.com');await page.locator('form[action$="/auth/request"] button[type="submit"]').click();
 const dev=page.locator('a.dev-link');await dev.waitFor();const url=new URL(await dev.getAttribute('href'),base);url.protocol=new URL(base).protocol;url.port=new URL(base).port;await page.goto(url.href,{waitUntil:'networkidle'});
 const app=new URL(page.url()).origin+'/demo';
 await page.goto(app+'/app/anliegen',{waitUntil:'networkidle'});
 const form=page.locator('form[data-issue-wizard]');
 const action=new URL(await form.getAttribute('action'),page.url()).href;
 const fields=await form.evaluate(f=>Object.fromEntries([...new FormData(f)].filter(([,v])=>typeof v==='string')));
 const locations=['Heizungsraum, Tiefenbohrung in der 5.000m2 Wiese unter der Liegenschaft','Garten · Spielplatz','Tiefgarage · Einfahrt','GebäudeteilOhneTrennzeichen'.repeat(5)];
 for(let i=0;i<4;i++){
  const response=await context.request.post(action,{headers:{Origin:new URL(app).origin},multipart:{...fields,category:'Reparatur',location_type:'common',title:`QA ${i+1}: Umstellung auf eine zentrale Wärmepumpe und gemeinsame Wärmeversorgung der gesamten Liegenschaft`,body:'Lokale QA-Fixture für den Zeilenumbruch.',location_detail:locations[i]},maxRedirects:0});
  assert.equal(response.status(),303,'local issue fixture saved');
  assert.match(response.headers().location,/\/app\/anliegen\/.+\?created=1$/, 'created issue detail redirect');
 }
 if(process.env.HV_QA_BASELINE_CSS)await page.route('**/assets/portal-shell.css*',route=>route.fulfill({contentType:'text/css',body:readFileSync(process.env.HV_QA_BASELINE_CSS,'utf8')}));
 for(const width of [320,390,600,768,1024,1280,1440,1920,2560]){
  await page.setViewportSize({width,height:1100});await page.goto(app+'/app',{waitUntil:'networkidle'});
  assert.equal(await page.locator('.calm-issues .row').count(),4,`four persisted issue rows at ${width}; page=${await page.title()}`);
  await page.locator('.calm-issues details.more').evaluate(el=>el.open=true);
  if(width<=760){assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth+1),false);report.push({width,mobile:true});if(width===390)await page.screenshot({path:`${out}/overview-${width}.png`,fullPage:true});continue;}
  const result=await page.locator('.calm-issues').evaluate(module=>{
   const box=e=>{const r=e.getBoundingClientRect();return {left:r.left,right:r.right,top:r.top,bottom:r.bottom,width:r.width,height:r.height};};
   const clip=e=>e.scrollWidth>e.clientWidth+1||e.scrollHeight>e.clientHeight+1;
   const title=document.querySelector('.portal-home-landing .portal-section-header-copy'),content=document.querySelector('.portal-home-landing .portal-section-content'),main=document.querySelector('.portal-shell-content');
   return {module:box(module),main:box(main),heading:box(title),content:box(content),inset:parseFloat(getComputedStyle(content).paddingLeft),overflow:document.documentElement.scrollWidth>innerWidth+1,rows:[...module.querySelectorAll('.row a')].map(a=>({link:box(a),title:box(a.querySelector('strong')),location:box(a.querySelector('.issue-location')),date:box(a.querySelector('time')),clipped:[...a.children].some(clip)}))};
  });
  assert.equal(result.rows.length,4,'initial and expanded rows included');assert(!result.overflow,`page overflow ${width}`);
  assert(Math.abs(result.heading.left-(result.main.left+result.inset))<1,`heading left inset ${width}`);
  assert(Math.abs(result.module.left-result.heading.left)<1,`cards align with greeting ${width}`);
  for(const row of result.rows){
   assert(row.title.width>=100,`title squeezed at ${width}: ${row.title.width}`);
   assert(!row.clipped,`clipped issue text at ${width}`);
   for(const cell of [row.title,row.location,row.date])assert(cell.left>=row.link.left-1&&cell.right<=row.link.right+1,`row overflow at ${width}`);
   const overlaps=(a,b)=>a.left<b.right-.5&&a.right>b.left+.5&&a.top<b.bottom-.5&&a.bottom>b.top+.5;
   assert(!overlaps(row.title,row.location)&&!overlaps(row.title,row.date)&&!overlaps(row.location,row.date),`overlap at ${width}`);
  }
  report.push({width,...result});
  if([390,1440,2560].includes(width))await page.screenshot({path:`${out}/overview-${width}.png`,fullPage:true});
 }
 writeFileSync(`${out}/geometry.json`,JSON.stringify(report,null,2));console.log('PASS long issue subjects/locations, expanded rows and left-aligned greeting/cards at 9 widths');
}finally{await browser.close();}
