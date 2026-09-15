// One product adapter around the reviewed offline INSPR bundle.
// HAUSV always shows Pretty; historical display preferences are ignored.
import {renderVersion} from './versioning/version.js';
import {attachVersionInteraction} from './versioning/version-interaction.js';
async function init() {
 const response = await fetch(new URL('./versioning/display.json', import.meta.url), {cache:'no-cache'});
 if (!response.ok) throw new Error('Version presentation unavailable');
 const config = await response.json();
 for (const host of document.querySelectorAll('[data-product-version]')) {
  if (host.dataset.versionScheme !== 'inspr-calendar-v2') continue;
  const brand = getComputedStyle(host).getPropertyValue('--gold').trim() || '#c8993f';
  renderVersion(host,host.dataset.productVersion,host.dataset.versionScheme,{config,mode:'pretty',brand,interactive:false});
  // A version inside a link keeps the link’s native navigation and keyboard behavior.
  if (!host.closest('a')) attachVersionInteraction(host,host.dataset.productVersion);
 }
}
init().catch(()=>{/* The canonical server-rendered version remains readable. */});
