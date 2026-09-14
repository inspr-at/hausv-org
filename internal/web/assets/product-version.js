// One thin adapter; all segment geometry, color weights, motion and copy
// behavior come from the reviewed offline INSPR presentation bundle.
import {renderVersion, disposeVersion} from './versioning/version.js';
import {attachVersionInteraction} from './versioning/version-interaction.js';
const controllers = new WeakMap();
let mode = 'pretty';
try { mode = localStorage.getItem('hausv:version-display') === 'reduced' ? 'reduced' : 'pretty'; } catch {}
async function init() {
 const response = await fetch(new URL('./versioning/display.json', import.meta.url), {cache:'no-cache'});
 if (!response.ok) throw new Error('Version presentation unavailable');
 const config = await response.json();
 const render = () => {
  for (const host of document.querySelectorAll('[data-product-version]')) {
   if (host.dataset.versionScheme !== 'inspr-calendar-v2') continue;
   controllers.get(host)?.dispose(); disposeVersion(host);
   const brand = getComputedStyle(host).getPropertyValue('--gold').trim() || '#c8993f';
   renderVersion(host,host.dataset.productVersion,host.dataset.versionScheme,{config,mode,brand,interactive:false});
   controllers.set(host,attachVersionInteraction(host,host.dataset.productVersion));
  }
 };
 render();
 for (const select of document.querySelectorAll('[data-version-display]')) {
  select.value=mode;
  select.addEventListener('change',()=>{mode=select.value==='reduced'?'reduced':'pretty';try{localStorage.setItem('hausv:version-display',mode);}catch{}render();});
 }
}
init().catch(()=>{/* The canonical server-rendered version remains readable. */});
