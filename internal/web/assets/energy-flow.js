// Data-driven energy-flow renderer for the cockpit (HAUSV-439).
//
// Each [data-energy-flow] container carries its config as a child
// <script type="application/json"> (CSP-safe: no inline executable JS, no
// server HTML for user-named tiles — every string lands via textContent).
// The renderer builds the tiles into fixed slots (producers top, storage
// left, grid bottom, consumers right, home centre) and draws the flows as
// single filled SVG polygons: sides offset by half the width, ending in a
// chisel tip exactly as wide as the line (angle >= 135°, no overhang).
// Source ribbons carry proportional colour bands — each destination owns a
// share of the line length matching its kW share, softly blended. Stroke
// width scales with sqrt(kW). Values are pre-formatted by the server; the
// renderer only appends the small raised unit marker.
(function () {
  "use strict";

  var ICONS = {
    pv: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><rect x="4" y="4" width="16" height="10" rx="1"/><path d="M4 9h16M9 4v10M15 4v10M8 18h8M12 14v4"/></svg>',
    battery: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><rect x="3" y="8" width="16" height="9" rx="2"/><path d="M21 11v3M6 11v3M10 11v3"/></svg>',
    grid: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M12 3v18M12 3l-7 5M12 3l7 5M5 12h14M7 21h10"/></svg>',
    house: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 11l8-7 8 7"/><path d="M6 9.5V20h12V9.5"/><path d="M10 20v-5h4v5"/></svg>',
    car: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M5 16l1.5-6.5A2 2 0 0 1 8.4 8h7.2a2 2 0 0 1 1.9 1.5L19 16"/><rect x="4" y="16" width="16" height="4" rx="1.5"/></svg>',
    boiler: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><rect x="7" y="3" width="10" height="18" rx="3"/><path d="M10 7h4M12 11v6"/></svg>',
    parking: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><rect x="4" y="4" width="16" height="16" rx="2"/><path d="M9 16V8h4a2.5 2.5 0 0 1 0 5H9"/></svg>',
    pump: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><rect x="3" y="6" width="18" height="12" rx="2"/><circle cx="12" cy="12" r="3.4"/><path d="M12 8.6v-1M12 16.4v1"/></svg>',
    device: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><rect x="5" y="4" width="14" height="16" rx="2"/><path d="M9 8h6M9 12h6M9 16h3"/></svg>',
    grip: '<svg viewBox="0 0 10 16" fill="currentColor"><circle cx="2.5" cy="2.5" r="1.4"/><circle cx="7.5" cy="2.5" r="1.4"/><circle cx="2.5" cy="8" r="1.4"/><circle cx="7.5" cy="8" r="1.4"/><circle cx="2.5" cy="13.5" r="1.4"/><circle cx="7.5" cy="13.5" r="1.4"/></svg>',
  };
  var HUB = "#a24b42", BATT = "#7893a1", GRID = "#b8891f", LOAD = "#3e704c", PV = "#4f7d49";

  function el(tag, className, parent) {
    var node = document.createElement(tag);
    if (className) node.className = className;
    if (parent) parent.appendChild(node);
    return node;
  }

  function icon(name, holderClass, parent) {
    var holder = el("span", holderClass, parent);
    holder.innerHTML = ICONS[name] || ICONS.device; // static dictionary only
    holder.setAttribute("aria-hidden", "true");
    return holder;
  }

  // value + tiny raised unit ("0,52" + "kW"); strings come pre-formatted.
  function valueLine(parent, value, unit) {
    var strong = el("strong", "", parent);
    strong.textContent = value;
    if (unit) {
      var u = el("span", "u", strong);
      u.textContent = unit;
    }
    return strong;
  }

  function tile(kind, iconName, parent) {
    var node = el("div", "energy-flow-tile k-" + kind, parent);
    icon(iconName, "ico", node);
    el("div", "", node);
    return node;
  }

  function sinkBands(cfg) {
    var bands = [];
    if (cfg.storage && cfg.storage.flow > 0 && cfg.storage.mode !== "entlädt") bands.push({ c: BATT, kw: cfg.storage.flow });
    (cfg.consumers || []).forEach(function (c) { if (c.kw > 0) bands.push({ c: LOAD, kw: c.kw }); });
    if (cfg.grid && cfg.grid.kw > 0 && cfg.grid.dir === "export") bands.push({ c: GRID, kw: cfg.grid.kw });
    bands.push({ c: HUB, kw: Math.max(cfg.home && cfg.home.kw > 0 ? cfg.home.kw : 0, 0.01) });
    return bands;
  }

  function bandStops(srcColor, bands) {
    var total = 0;
    bands.forEach(function (b) { total += b.kw; });
    var stops = [{ at: 0, c: srcColor }, { at: 0.14, c: srcColor }];
    var acc = 0;
    bands.forEach(function (b) {
      stops.push({ at: 0.2 + 0.8 * ((acc + b.kw / 2) / total), c: b.c });
      acc += b.kw;
    });
    stops.push({ at: 1, c: bands[bands.length - 1].c });
    return stops;
  }

  function render(wrap) {
    var configNode = wrap.querySelector('script[type="application/json"]');
    if (!configNode) return;
    var cfg;
    try { cfg = JSON.parse(configNode.textContent); } catch (err) { return; }
    if (!cfg) return;

    var svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
    svg.setAttribute("class", "energy-flow-ribbons");
    svg.setAttribute("aria-hidden", "true");
    wrap.appendChild(svg);

    var flow = el("div", "energy-flow-grid2", wrap);
    var edges = [];

    var top = el("div", "energy-flow-slot-top", flow);
    (cfg.producers || []).forEach(function (p, i) {
      var node = tile("pv", p.icon || "pv", top);
      node.dataset.edge = "producer-" + i;
      valueLine(node.lastChild, p.value, p.unit);
      el("span", "", node.lastChild).textContent = p.label;
      if (p.kw > 0) edges.push({ from: "producer-" + i, to: "hub", kw: p.kw, src: PV });
    });

    if (cfg.storage) {
      var leftSlot = el("div", "energy-flow-slot-left", flow);
      var st = tile("batt", "battery", leftSlot);
      st.dataset.edge = "storage";
      valueLine(st.lastChild, cfg.storage.value, cfg.storage.unit);
      el("span", "", st.lastChild).textContent = cfg.storage.sub;
      if (cfg.storage.flow > 0) {
        if (cfg.storage.mode === "entlädt") edges.push({ from: "storage", to: "hub", kw: cfg.storage.flow, src: BATT });
        else edges.push({ from: "hub", to: "storage", kw: cfg.storage.flow, stops: [{ at: 0, c: PV }, { at: 1, c: BATT }] });
      }
    }

    var hub = tile("hub", "house", flow);
    hub.classList.add("energy-flow-hub2");
    hub.dataset.edge = "hub";
    valueLine(hub.lastChild, cfg.home.value, cfg.home.unit);
    el("span", "", hub.lastChild).textContent = cfg.home.label || "Hausverbrauch";

    if (cfg.grid) {
      var bottom = el("div", "energy-flow-slot-bottom", flow);
      var gr = tile("grid", "grid", bottom);
      gr.dataset.edge = "grid";
      valueLine(gr.lastChild, cfg.grid.value, cfg.grid.unit);
      el("span", "", gr.lastChild).textContent = cfg.grid.label;
      if (cfg.grid.kw > 0) {
        if (cfg.grid.dir === "import") edges.push({ from: "grid", to: "hub", kw: cfg.grid.kw, src: GRID });
        else edges.push({ from: "hub", to: "grid", kw: cfg.grid.kw, stops: [{ at: 0, c: PV }, { at: 1, c: GRID }] });
      }
    }

    if ((cfg.consumers && cfg.consumers.length) || cfg.addHint) {
      wrap.classList.add("has-rail");
      var rail = el("div", "energy-flow-rail", wrap);
      el("span", "microlabel", rail).textContent = "Großverbraucher · Priorität";
      (cfg.consumers || []).forEach(function (c, i) {
        var t = el("div", "energy-flow-big" + (c.active ? " active" : ""), rail);
        t.dataset.edge = "consumer-" + i;
        icon(c.icon || "device", "ico", t);
        var text = el("div", "", t);
        el("b", "", text).textContent = c.title;
        if (c.sub) el("span", "sub", text).textContent = c.sub;
        if (c.state) el("span", "state", text).textContent = c.state;
        var rcol = el("span", "rcol", t);
        el("span", "prio", rcol).textContent = String(i + 1);
        icon("grip", "drag", rcol);
        if (c.kw > 0) edges.push({ from: "hub", to: "consumer-" + i, kw: c.kw, stops: [{ at: 0, c: PV }, { at: 1, c: LOAD }] });
      });
      if (cfg.addHint) {
        var ghost = el("div", "energy-flow-big ghost", rail);
        var plus = el("span", "plus", ghost);
        plus.textContent = "+";
        var gtext = el("div", "", ghost);
        el("b", "", gtext).textContent = "Großverbraucher hinzufügen";
        el("span", "", gtext).textContent = "steuern oder nur beobachten";
        ghost.setAttribute("role", "link");
        ghost.tabIndex = 0;
        var go = function () { location.hash = "anlagen"; };
        ghost.addEventListener("click", go);
        ghost.addEventListener("keydown", function (evt) { if (evt.key === "Enter" || evt.key === " ") { evt.preventDefault(); go(); } });
      }
    }

    var bands = sinkBands(cfg);
    edges.forEach(function (e) {
      if (!e.stops) e.stops = bandStops(e.src, bands);
    });

    function draw() {
      var wr = wrap.getBoundingClientRect();
      if (wr.width < 10) return;
      svg.setAttribute("viewBox", "0 0 " + wr.width + " " + wr.height);
      var maxKw = 0.01;
      edges.forEach(function (e) { if (e.kw > maxKw) maxKw = e.kw; });
      var defs = "<defs>", shapes = "";
      edges.forEach(function (e, i) {
        var fromEl = wrap.querySelector('[data-edge="' + e.from + '"]');
        var toEl = wrap.querySelector('[data-edge="' + e.to + '"]');
        if (!fromEl || !toEl) return;
        var ra = fromEl.getBoundingClientRect(), rb = toEl.getBoundingClientRect();
        var a = { l: ra.left - wr.left, r: ra.right - wr.left, t: ra.top - wr.top, b: ra.bottom - wr.top, cx: ra.left - wr.left + ra.width / 2, cy: ra.top - wr.top + ra.height / 2 };
        var b = { l: rb.left - wr.left, r: rb.right - wr.left, t: rb.top - wr.top, b: rb.bottom - wr.top, cx: rb.left - wr.left + rb.width / 2, cy: rb.top - wr.top + rb.height / 2 };
        var vertical = Math.abs(a.cx - b.cx) < 60 || Math.abs(a.cy - b.cy) > Math.abs(a.cx - b.cx);
        var x1, y1, x2, y2, dir;
        if (vertical) {
          var down = a.cy < b.cy;
          x1 = a.cx; y1 = down ? a.b : a.t; x2 = b.cx; y2 = down ? b.t : b.b; dir = [0, down ? 1 : -1];
        } else {
          var right = a.cx < b.cx;
          x1 = right ? a.r : a.l; y1 = a.cy; x2 = right ? b.l : b.r; y2 = b.cy; dir = [right ? 1 : -1, 0];
        }
        var w = 3 + 11 * Math.sqrt(e.kw / maxKw);
        var h = w / 2;
        var tip = Math.max(2.5, 0.2 * w);
        var ex = x2 - dir[0] * tip, ey = y2 - dir[1] * tip;
        var d;
        if (vertical) {
          var dy = (ey - y1) * 0.5;
          d = "M " + (x1 - h) + " " + y1 + " C " + (x1 - h) + " " + (y1 + dy) + ", " + (x2 - h) + " " + (ey - dy) + ", " + (x2 - h) + " " + ey +
              " L " + x2 + " " + y2 + " L " + (x2 + h) + " " + ey + " C " + (x2 + h) + " " + (ey - dy) + ", " + (x1 + h) + " " + (y1 + dy) + ", " + (x1 + h) + " " + y1 + " Z";
        } else {
          var dx = (ex - x1) * 0.5;
          d = "M " + x1 + " " + (y1 - h) + " C " + (x1 + dx) + " " + (y1 - h) + ", " + (ex - dx) + " " + (ey - h) + ", " + ex + " " + (ey - h) +
              " L " + x2 + " " + y2 + " L " + ex + " " + (ey + h) + " C " + (ex - dx) + " " + (ey + h) + ", " + (x1 + dx) + " " + (y1 + h) + ", " + x1 + " " + (y1 + h) + " Z";
        }
        var gid = "efg-" + wrap.dataset.energyFlow + "-" + i;
        var stops = "";
        e.stops.forEach(function (st) { stops += '<stop offset="' + st.at.toFixed(3) + '" stop-color="' + st.c + '"/>'; });
        defs += '<linearGradient id="' + gid + '" gradientUnits="userSpaceOnUse" x1="' + x1 + '" y1="' + y1 + '" x2="' + x2 + '" y2="' + y2 + '">' + stops + "</linearGradient>";
        shapes += '<path d="' + d + '" fill="url(#' + gid + ')"/>';
      });
      svg.innerHTML = defs + "</defs>" + shapes;
    }

    wrap.classList.add("is-enhanced");
    requestAnimationFrame(draw);
    if ("ResizeObserver" in window) {
      var pending = 0;
      new ResizeObserver(function () {
        cancelAnimationFrame(pending);
        pending = requestAnimationFrame(draw);
      }).observe(wrap);
    }
  }

  function init() {
    var seq = 0;
    document.querySelectorAll("[data-energy-flow]").forEach(function (wrap) {
      if (!wrap.dataset.energyFlow) wrap.dataset.energyFlow = String(++seq);
      render(wrap);
    });
  }

  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init, { once: true });
  else init();
})();
