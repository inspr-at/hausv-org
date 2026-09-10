// Google Ads tag bootstrap. Loaded only when GOOGLE_ADS_TAG_ID is set; the
// tag id and the optional lead-conversion id come from data attributes on
// this script tag so no inline script is needed under the CSP.
(function () {
  var script = document.currentScript;
  if (!script || !script.dataset.tagId) return;
  window.dataLayer = window.dataLayer || [];
  function gtag() { window.dataLayer.push(arguments); }
  window.gtag = gtag;
  gtag("js", new Date());
  // accept_incoming: the company site (augmentoring.com/hausv) links here
  // with the click id in the URL, so a lead submitted on /start is credited
  // to the ad that landed there (HAUSV-734).
  gtag("config", script.dataset.tagId, { linker: { accept_incoming: true } });
  if (script.dataset.leadConversion) {
    gtag("event", "conversion", {
      send_to: script.dataset.leadConversion,
      value: 1.0,
      currency: "EUR",
    });
  }
})();
