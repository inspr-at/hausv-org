// Consent gate for the optional Google Ads tag on the public pages (HAUSV-742).
//
// Nothing optional loads before an affirmative choice: the Google tag is
// injected only after "marketing" was granted, with Consent Mode v2 in basic
// mode (no tag, no ping before consent). The decision lives in a host-only
// cookie without any visitor identifier. An affirmative Global Privacy Control
// or Do-Not-Track signal counts as refusal and suppresses the bar. Withdrawal
// is as easy as consent: every public page carries a "Privatsphäre" control.
//
// The policy part has no DOM dependency so scripts/test-consent-policy.mjs
// can exercise it under Node; the DOM part runs only in a browser.
(function () {
  "use strict";

  var COOKIE = "hausv_consent";
  // Bump only for a material change of purposes (new category, new
  // recipient). Copy edits keep the stored decision.
  var PURPOSE_REVISION = 1;
  var MAX_AGE = 15552000; // 180 days, for a grant and for a refusal alike
  var LEAD_KEY = "hausv_ads_lead_fired";
  var GOOGLE_COOKIE = /^(_gcl_|_gac_|_ga|_gid)/;
  var GOOGLE_LOCAL = ["_gcl_ls"];
  var BOT = /bot|crawl|spider|slurp|headless|lighthouse|pagespeed|preview/i;

  function parse(raw) {
    if (typeof raw !== "string" || !raw) return null;
    var pairs = raw.split(";");
    if (pairs[0] !== "v1") return null;
    var out = { revision: 0, marketing: false, at: 0 };
    for (var i = 1; i < pairs.length; i++) {
      var kv = pairs[i].split("=");
      if (kv[0] === "r") out.revision = parseInt(kv[1], 10) || 0;
      if (kv[0] === "m") out.marketing = kv[1] === "1";
      if (kv[0] === "t") out.at = parseInt(kv[1], 10) || 0;
    }
    return out;
  }

  function serialize(marketing, now) {
    return "v1;r=" + PURPOSE_REVISION + ";m=" + (marketing ? "1" : "0") + ";t=" + Math.floor(now);
  }

  // decide returns what the page may do right now.
  //   marketing: whether the Google tag may load
  //   prompt:    whether the bar has to be shown
  //   persist:   "refuse" when a signal-driven refusal should be stored
  function decide(input) {
    var stored = input.stored || null;
    var now = input.now;
    if (input.bot) return { marketing: false, prompt: false, persist: "none" };
    if (input.signal) return { marketing: false, prompt: false, persist: "refuse" };
    if (stored && stored.revision === PURPOSE_REVISION && now - stored.at < MAX_AGE) {
      return { marketing: stored.marketing === true, prompt: false, persist: "none" };
    }
    return { marketing: false, prompt: true, persist: "none" };
  }

  var policy = { parse: parse, serialize: serialize, decide: decide, cookieName: COOKIE, revision: PURPOSE_REVISION, maxAge: MAX_AGE };
  var root = typeof globalThis !== "undefined" ? globalThis : this;
  root.hausvConsentPolicy = policy;
  if (typeof document === "undefined") return;

  // ---- DOM part -----------------------------------------------------------

  var script = document.currentScript;
  var config = script ? script.dataset : {};
  var tagId = config.tagId || "";
  var leadConversion = config.leadConversion || "";
  var googleLoaded = false;
  var bar = null;
  var sheet = null;

  function readCookie() {
    var parts = document.cookie ? document.cookie.split("; ") : [];
    for (var i = 0; i < parts.length; i++) {
      if (parts[i].indexOf(COOKIE + "=") === 0) return parse(decodeURIComponent(parts[i].slice(COOKIE.length + 1)));
    }
    return null;
  }

  // Returns true only when the decision can be read back: without a
  // persisted decision nothing optional may load (fail closed).
  function writeCookie(marketing) {
    var secure = location.protocol === "https:" ? "; Secure" : "";
    try {
      document.cookie = COOKIE + "=" + encodeURIComponent(serialize(marketing, Date.now() / 1000)) + "; Max-Age=" + MAX_AGE + "; Path=/; SameSite=Lax" + secure;
    } catch (_) { return false; }
    var stored = readCookie();
    return !!stored && stored.marketing === marketing && stored.revision === PURPOSE_REVISION;
  }

  function signalActive() {
    return navigator.globalPrivacyControl === true || navigator.doNotTrack === "1" || window.doNotTrack === "1";
  }

  function isBot() { return BOT.test(navigator.userAgent || ""); }

  // Every load re-evaluates the full policy: a stale grant, an active privacy
  // signal, a bot or a failed cookie write never lets the tag through.
  function authorized() {
    var decision = decide({ stored: readCookie(), now: Date.now() / 1000, signal: signalActive(), bot: isBot() });
    return decision.marketing === true;
  }

  function clearMarketingStorage() {
    clearGoogleCookies();
    GOOGLE_LOCAL.forEach(function (key) { try { localStorage.removeItem(key); } catch (_) { /* unavailable */ } });
    try { sessionStorage.removeItem(LEAD_KEY); } catch (_) { /* unavailable */ }
  }

  function clearGoogleCookies() {
    var names = (document.cookie ? document.cookie.split("; ") : []).map(function (c) { return c.split("=")[0]; });
    var host = location.hostname;
    var domains = [""];
    var labels = host.split(".");
    for (var i = 0; i < labels.length - 1; i++) domains.push(labels.slice(i).join("."));
    names.forEach(function (name) {
      if (!GOOGLE_COOKIE.test(name)) return;
      domains.forEach(function (domain) {
        document.cookie = name + "=; Max-Age=0; Path=/" + (domain ? "; Domain=" + domain : "");
      });
    });
  }

  function gtag() { window.dataLayer.push(arguments); }

  function loadGoogle() {
    if (googleLoaded || !tagId || !authorized()) return;
    googleLoaded = true;
    window.dataLayer = window.dataLayer || [];
    window.gtag = gtag;
    // Consent Mode v2, basic: defaults are denied and the tag is only injected
    // after the update, so no cookieless ping ever leaves before consent.
    gtag("consent", "default", { ad_storage: "denied", ad_user_data: "denied", ad_personalization: "denied", analytics_storage: "denied" });
    // Only what the bar and the privacy notice disclose: conversion
    // measurement. Personalised advertising stays denied.
    gtag("consent", "update", { ad_storage: "granted", ad_user_data: "granted", ad_personalization: "denied", analytics_storage: "denied" });
    gtag("js", new Date());
    // accept_incoming: the company site links here with the click id in the
    // URL so a lead submitted on /start is credited to the ad (HAUSV-734).
    // cookie_domain keeps Google's cookies on this host only.
    gtag("config", tagId, { linker: { accept_incoming: true }, cookie_domain: location.hostname, cookie_flags: "SameSite=Lax;Secure" });
    if (leadConversion) fireLeadConversion();
    var el = document.createElement("script");
    el.async = true;
    el.src = "https://www.googletagmanager.com/gtag/js?id=" + encodeURIComponent(tagId);
    document.head.appendChild(el);
  }

  // The lead conversion is only configured on the page a visitor reaches after
  // submitting the landing form; this guard keeps a reload from counting twice.
  function fireLeadConversion() {
    try {
      if (sessionStorage.getItem(LEAD_KEY) === "1") return;
      sessionStorage.setItem(LEAD_KEY, "1");
    } catch (_) { /* storage unavailable: fire once per page load */ }
    gtag("event", "conversion", { send_to: leadConversion, value: 1.0, currency: "EUR" });
  }

  function withdraw() {
    writeCookie(false);
    if (!googleLoaded) { clearMarketingStorage(); return; }
    gtag("consent", "update", { ad_storage: "denied", ad_user_data: "denied", ad_personalization: "denied", analytics_storage: "denied" });
    clearMarketingStorage();
    // An already-loaded tag has no reliable teardown; a reload starts clean.
    location.reload();
  }

  function el(tag, attrs, children) {
    var node = document.createElement(tag);
    Object.keys(attrs || {}).forEach(function (k) {
      if (k === "text") node.textContent = attrs[k];
      else if (k === "html") node.innerHTML = attrs[k];
      else node.setAttribute(k, attrs[k]);
    });
    (children || []).forEach(function (c) { node.appendChild(c); });
    return node;
  }

  function removeBar() {
    if (bar && bar.parentNode) bar.parentNode.removeChild(bar);
    bar = null;
    document.removeEventListener("keydown", onEscape);
  }

  function onEscape(event) {
    if (event.key === "Escape" && bar) refuse();
  }

  function accept() {
    removeBar();
    // A grant that cannot be persisted is no grant; a refusal signal wins.
    if (signalActive() || !writeCookie(true)) { withdraw(); return; }
    loadGoogle();
  }
  function refuse() { removeBar(); withdraw(); }

  function renderBar() {
    if (bar) return;
    var reject = el("button", { type: "button", class: "hv-consent-btn", "data-consent": "reject", text: "Ablehnen" });
    var acceptBtn = el("button", { type: "button", class: "hv-consent-btn", "data-consent": "accept", text: "Akzeptieren" });
    var settings = el("button", { type: "button", class: "hv-consent-link", "data-consent-open": "", text: "Einstellungen" });
    reject.addEventListener("click", refuse);
    acceptBtn.addEventListener("click", accept);
    settings.addEventListener("click", openSheet);
    bar = el("section", { class: "hv-consent", role: "region", "aria-label": "Cookie-Einwilligung" }, [
      el("p", { class: "hv-consent-text", html: "Wir verwenden notwendige Cookies. Mit Ihrer Einwilligung nutzen wir zusätzlich Google Ads, um zu messen, welche Anzeige zu einer Anfrage geführt hat. <a href=\"/datenschutz\">Datenschutz</a>" }),
      el("div", { class: "hv-consent-actions" }, [reject, acceptBtn, settings])
    ]);
    document.body.appendChild(bar);
    document.addEventListener("keydown", onEscape);
  }

  var marketingBox = null;

  function openSheet() {
    if (!sheet) {
      var marketing = el("input", { type: "checkbox", id: "hv-consent-marketing" });
      marketingBox = marketing;
      var necessary = el("input", { type: "checkbox", id: "hv-consent-necessary", checked: "", disabled: "" });
      var save = el("button", { type: "button", class: "hv-consent-btn", text: "Speichern" });
      var cancel = el("button", { type: "button", class: "hv-consent-btn", text: "Abbrechen" });
      sheet = el("dialog", { class: "hv-consent-sheet", "aria-labelledby": "hv-consent-title" }, [
        el("h2", { id: "hv-consent-title", text: "Privatsphäre" }),
        el("p", { text: "Notwendige Cookies halten Sie angemeldet und schützen Formulare. Alles Weitere bleibt aus, bis Sie es hier einschalten." }),
        el("div", { class: "hv-consent-row" }, [necessary, el("label", { for: "hv-consent-necessary", html: "<strong>Notwendig</strong> — Anmeldung, Formulare, diese Einstellung. Immer aktiv." })]),
        el("div", { class: "hv-consent-row" }, [marketing, el("label", { for: "hv-consent-marketing", html: "<strong>Marketing (Google Ads)</strong> — Google Ireland Ltd. misst, ob eine Anzeige zu einer Anfrage geführt hat; Daten können in die USA übertragen werden. Cookies bis zu 90 Tage, Ihre Wahl 6 Monate." })]),
        el("div", { class: "hv-consent-actions" }, [cancel, save]),
        el("p", { class: "hv-consent-mini", html: "Details und Widerruf: <a href=\"/datenschutz\">Datenschutz</a>" })
      ]);
      save.addEventListener("click", function () {
        sheet.close();
        if (marketing.checked) accept(); else refuse();
      });
      // Cancel keeps an existing decision; while the first-layer bar is still
      // open there is none yet, and dismissing counts as refusal.
      cancel.addEventListener("click", function () { sheet.close(); if (bar) refuse(); });
      sheet.addEventListener("cancel", function () { if (bar) refuse(); });
      document.body.appendChild(sheet);
    }
    // Reflect the validated current state on every open, never a stale draft.
    marketingBox.checked = authorized();
    marketingBox.disabled = signalActive();
    if (typeof sheet.showModal === "function") sheet.showModal(); else sheet.setAttribute("open", "");
  }

  function wireControls() {
    var controls = document.querySelectorAll("[data-consent-open]");
    for (var i = 0; i < controls.length; i++) {
      if (controls[i].hasAttribute("data-consent-wired")) continue;
      controls[i].setAttribute("data-consent-wired", "");
      controls[i].addEventListener("click", openSheet);
    }
  }

  function boot() {
    wireControls();
    if (!tagId) return;
    var decision = decide({ stored: readCookie(), now: Date.now() / 1000, signal: signalActive(), bot: isBot() });
    if (decision.persist === "refuse") { writeCookie(false); clearMarketingStorage(); }
    if (decision.marketing) loadGoogle();
    if (decision.prompt) renderBar();
  }

  root.hausvConsent = { open: openSheet, withdraw: withdraw };
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", boot);
  else boot();
})();
