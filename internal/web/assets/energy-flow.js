// Data-driven energy-flow renderer for the cockpit (HAUSV-439, HAUSV-441).
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
//
// Consumers with an id can be reordered by dragging their tile or pressing
// the arrow keys on the grip button; the order is persisted through
// POST /app/energie/verbraucher/reihenfolge and rolled back if that fails.
// Entries without an id (Parkplatz 20) keep their anchored place at the end.
(function () {
  "use strict";

  var ORDER_ENDPOINT = "/app/energie/verbraucher/reihenfolge";

  // Only names from the locally vendored Lucide library are accepted here.
  // Configuration data can therefore select an icon, but never inject markup
  // or point the browser at a remote resource.
  var ICON_NAMES = [
    "solar-panel", "battery", "utility-pole", "house", "car-front",
    "plug-zap", "heater", "fan", "washing-machine", "flame", "drill",
    "waves-ladder", "square-parking", "snowflake", "shower-head", "plug",
    "plus", "grip-vertical", "pencil"
  ].reduce(function (names, name) { names[name] = true; return names; }, {});
  var HUB = "#a24b42", BATT = "#7893a1", GRID = "#b8891f", LOAD = "#3e704c", PV = "#4f7d49";

  function el(tag, className, parent) {
    var node = document.createElement(tag);
    if (className) node.className = className;
    if (parent) parent.appendChild(node);
    return node;
  }

  function glyph(name, className, parent) {
    var safeName = ICON_NAMES[name] ? name : "plug";
    var node = el("span", "energy-ui-icon energy-ui-icon-" + safeName + (className ? " " + className : ""), parent);
    node.setAttribute("aria-hidden", "true");
    return node;
  }

  function icon(name, holderClass, parent) {
    var holder = el("span", holderClass, parent);
    glyph(name, "", holder);
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

  // The ribbon shows only DESTINATION shares: soft bands sized by kW, no
  // source colour (the tile itself already names the source).
  function bandStops(bands) {
    var total = 0;
    bands.forEach(function (b) { total += b.kw; });
    var stops = [{ at: 0, c: bands[0].c }];
    var acc = 0;
    bands.forEach(function (b) {
      stops.push({ at: (acc + b.kw / 2) / total, c: b.c });
      acc += b.kw;
    });
    stops.push({ at: 1, c: bands[bands.length - 1].c });
    return stops;
  }

  // Saves run strictly one after another; a failed save only rolls back when
  // no newer order has been submitted since. An auth redirect (302/303 to the
  // login page) counts as failure even though the followed GET returns 200.
  var saveChain = Promise.resolve();
  var saveSeq = 0;
  function saveOrder(cfg, onFail) {
    var ids = (cfg.consumers || []).filter(function (c) { return c.id; }).map(function (c) { return c.id; });
    if (!ids.length) return;
    var body = ids.map(function (id) { return "order=" + encodeURIComponent(id); }).join("&");
    saveSeq += 1;
    var mySeq = saveSeq;
    saveChain = saveChain.then(function () {
      return fetch(ORDER_ENDPOINT, {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body: body,
      }).then(function (response) {
        if ((!response.ok || response.redirected) && mySeq === saveSeq) onFail();
      }).catch(function () {
        if (mySeq === saveSeq) onFail();
      });
    });
  }

  // Moves the id-bearing consumer at `from` to `to` (indices among the
  // movable entries); anchored entries (no id) always stay behind them.
  function reorderConsumers(cfg, from, to) {
    var movable = cfg.consumers.filter(function (c) { return c.id; });
    var anchored = cfg.consumers.filter(function (c) { return !c.id; });
    if (from < 0 || from >= movable.length) return false;
    var target = Math.max(0, Math.min(movable.length - 1, to));
    if (target === from) return false;
    var moved = movable.splice(from, 1)[0];
    movable.splice(target, 0, moved);
    cfg.consumers = movable.concat(anchored);
    return true;
  }

  var consumerDialog = document.getElementById("energy-consumer-dialog");
  var consumerIconSearch = consumerDialog && consumerDialog.querySelector("[data-consumer-icon-search]");
  if (consumerIconSearch) {
    consumerIconSearch.addEventListener("input", function () {
      var query = consumerIconSearch.value.trim().toLocaleLowerCase("de");
      [].forEach.call(consumerDialog.querySelectorAll("[data-consumer-icon-choice]"), function (choice) {
        choice.hidden = Boolean(query && choice.dataset.consumerIconChoice.toLocaleLowerCase("de").indexOf(query) === -1);
      });
    });
  }

  function setDialogValue(name, value) {
    if (!consumerDialog) return;
    var field = consumerDialog.querySelector('[name="' + name + '"]');
    if (field) field.value = value == null ? "" : String(value);
  }

  function openConsumerDialog(consumer, priority, total, trigger) {
    if (!consumerDialog || typeof consumerDialog.showModal !== "function") return;
    var item = consumer || {};
    setDialogValue("asset_id", item.id || "");
    setDialogValue("name", item.title || "");
    setDialogValue("kind", item.kind || "other");
    setDialogValue("rated_power_kw", item.ratedPower || "");
    setDialogValue("flexibility", item.flexibility || "unknown");

    var priorityField = consumerDialog.querySelector('[name="priority"]');
    if (priorityField) {
      priorityField.replaceChildren();
      var count = Math.max(1, total + (item.id ? 0 : 1));
      for (var i = 1; i <= count; i += 1) {
        var option = document.createElement("option");
        option.value = String(i);
        option.textContent = String(i);
        priorityField.appendChild(option);
      }
      priorityField.value = String(priority || count);
    }

    var wantedIcon = ICON_NAMES[item.icon] ? item.icon : "plug";
    var iconFound = false;
    [].forEach.call(consumerDialog.querySelectorAll('[name="icon"]'), function (radio) {
      radio.checked = radio.value === wantedIcon;
      if (radio.checked) iconFound = true;
    });
    if (!iconFound) {
      var fallback = consumerDialog.querySelector('[name="icon"][value="plug"]');
      if (fallback) fallback.checked = true;
    }

    var title = consumerDialog.querySelector("[data-consumer-dialog-title]");
    var context = consumerDialog.querySelector("[data-consumer-dialog-context]");
    var submit = consumerDialog.querySelector("[data-consumer-submit]");
    if (title) title.textContent = item.id ? "Verbraucher bearbeiten" : "Verbraucher hinzufügen";
    if (context) context.textContent = item.id ? "Name, Priorität und Symbol direkt anpassen." : "Neuen Verbraucher im Energiefluss anlegen.";
    if (submit) submit.textContent = item.id ? "Änderungen speichern" : "Verbraucher hinzufügen";

    var remove = consumerDialog.querySelector("[data-consumer-delete]");
    if (remove) {
      remove.hidden = !(item.id && item.custom);
      remove.disabled = remove.hidden;
    }
    if (consumerIconSearch) {
      consumerIconSearch.value = "";
      consumerIconSearch.dispatchEvent(new Event("input"));
    }
    consumerDialog._returnFocus = trigger || null;
    if (!consumerDialog.open) consumerDialog.showModal();
    window.setTimeout(function () {
      var nameField = consumerDialog.querySelector('[name="name"]');
      if (nameField) nameField.focus();
    }, 0);
  }

  function render(wrap) {
    var configNode = wrap.querySelector('script[type="application/json"]');
    if (!configNode) return;
    var cfg;
    try { cfg = JSON.parse(configNode.textContent); } catch (err) { return; }
    if (!cfg) return;

    var built = [];
    var currentDraw = null;
    if ("ResizeObserver" in window) {
      var pendingDraw = 0;
      new ResizeObserver(function () {
        cancelAnimationFrame(pendingDraw);
        pendingDraw = requestAnimationFrame(function () { if (currentDraw) currentDraw(); });
      }).observe(wrap);
    }

    function rebuild(focusConsumerID) {
      built.forEach(function (node) { node.remove(); });
      built = [];
      build();
      if (focusConsumerID) {
        var grip = wrap.querySelector('[data-consumer-id="' + focusConsumerID + '"] button.drag');
        if (grip) grip.focus();
      }
    }

    function persist(previousOrder, focusConsumerID) {
      saveOrder(cfg, function () {
        cfg.consumers = previousOrder;
        rebuild(focusConsumerID);
        announce("Reihenfolge konnte nicht gespeichert werden und wurde zurückgesetzt.");
      });
    }

    // Screen-reader feedback for keyboard reordering (polite live region).
    var live = el("span", "sr-only", wrap);
    live.setAttribute("aria-live", "polite");
    function announce(message) {
      live.textContent = "";
      requestAnimationFrame(function () { live.textContent = message; });
    }

    function build() {
      var svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
      svg.setAttribute("class", "energy-flow-ribbons");
      svg.setAttribute("aria-hidden", "true");
      wrap.appendChild(svg);
      built.push(svg);

      var flow = el("div", "energy-flow-grid2", wrap);
      built.push(flow);
      var edges = [];

      var top = el("div", "energy-flow-slot-top", flow);
      (cfg.producers || []).forEach(function (p, i) {
        var node = tile("pv", p.icon || "solar-panel", top);
        if (p.hover) node.title = p.hover;
        node.dataset.edge = "producer-" + i;
        valueLine(node.lastChild, p.value, p.unit);
        el("span", "", node.lastChild).textContent = p.label;
        if (p.kw > 0) edges.push({ from: "producer-" + i, to: "hub", kw: p.kw, banded: true });
      });

      if (cfg.storage) {
        var leftSlot = el("div", "energy-flow-slot-left", flow);
        var st = tile("batt", "battery", leftSlot);
        if (cfg.storage.hover) st.title = cfg.storage.hover;
        st.dataset.edge = "storage";
        valueLine(st.lastChild, cfg.storage.value, cfg.storage.unit);
        el("span", "", st.lastChild).textContent = cfg.storage.sub;
        if (cfg.storage.flow > 0) {
          if (cfg.storage.mode === "entlädt") edges.push({ from: "storage", to: "hub", kw: cfg.storage.flow, banded: true });
          else edges.push({ from: "hub", to: "storage", kw: cfg.storage.flow, stops: [{ at: 0, c: BATT }, { at: 1, c: BATT }] });
        }
      }

      var hub = tile("hub", "house", flow);
      hub.classList.add("energy-flow-hub2");
      if (cfg.home.hover) hub.title = cfg.home.hover;
      hub.dataset.edge = "hub";
      valueLine(hub.lastChild, cfg.home.value, cfg.home.unit);
      el("span", "", hub.lastChild).textContent = cfg.home.label || "Hausverbrauch";

      if (cfg.grid) {
        var bottom = el("div", "energy-flow-slot-bottom", flow);
        var gr = tile("grid", "utility-pole", bottom);
        if (cfg.grid.hover) gr.title = cfg.grid.hover;
        gr.dataset.edge = "grid";
        valueLine(gr.lastChild, cfg.grid.value, cfg.grid.unit);
        el("span", "", gr.lastChild).textContent = cfg.grid.label;
        if (cfg.grid.kw > 0) {
          if (cfg.grid.dir === "import") edges.push({ from: "grid", to: "hub", kw: cfg.grid.kw, banded: true });
          else edges.push({ from: "hub", to: "grid", kw: cfg.grid.kw, stops: [{ at: 0, c: GRID }, { at: 1, c: GRID }] });
        }
      }

      if ((cfg.consumers && cfg.consumers.length) || cfg.addHint) {
        wrap.classList.add("has-rail");
        var rail = el("div", "energy-flow-rail", wrap);
        built.push(rail);
        el("span", "microlabel", rail).textContent = "Verbraucher · Priorität";
        var draggingTile = null;
        function renumberRail() {
          [].forEach.call(rail.querySelectorAll("[data-consumer-id]"), function (node, index) {
            var prio = node.querySelector(".prio");
            if (prio) prio.textContent = String(index + 1);
          });
        }
        function commitDomOrder(focusID) {
          var domIDs = [].map.call(rail.querySelectorAll("[data-consumer-id]"), function (node) {
            return node.dataset.consumerId;
          });
          var current = cfg.consumers.filter(function (item) { return item.id; }).map(function (item) { return item.id; });
          if (domIDs.join(",") === current.join(",")) { rebuild(focusID); return; }
          var previous = cfg.consumers.slice();
          var byID = {};
          cfg.consumers.forEach(function (item) { if (item.id) byID[item.id] = item; });
          cfg.consumers = domIDs.map(function (id) { return byID[id]; })
            .concat(cfg.consumers.filter(function (item) { return !item.id; }));
          rebuild(focusID || domIDs[0]);
          persist(previous, focusID || domIDs[0]);
        }
        rail.addEventListener("dragover", function (event) {
          if (!draggingTile) return;
          event.preventDefault();
          var tiles = [].filter.call(rail.querySelectorAll("[data-consumer-id]"), function (node) {
            return node !== draggingTile;
          });
          var after = null;
          for (var index = 0; index < tiles.length; index += 1) {
            var box = tiles[index].getBoundingClientRect();
            if (event.clientY < box.top + box.height / 2) { after = tiles[index]; break; }
          }
          if (after) rail.insertBefore(draggingTile, after);
          else {
            // Ans Ende der verschiebbaren Kacheln — vor verankerte und Ghost.
            var anchor = rail.querySelector(".energy-flow-big:not([data-consumer-id])");
            rail.insertBefore(draggingTile, anchor);
          }
          renumberRail();
        });
        rail.addEventListener("drop", function (event) {
          if (!draggingTile) return;
          event.preventDefault();
          var droppedID = draggingTile.dataset.consumerId;
          draggingTile = null;
          commitDomOrder(droppedID);
        });
        var movableIndex = -1;
        (cfg.consumers || []).forEach(function (c, i) {
          var canEdit = Boolean(c.id && cfg.addHint);
          var t = el("div", "energy-flow-big" + (c.active ? " active" : "") + (canEdit ? " editable" : ""), rail);
          t.dataset.edge = "consumer-" + i;
          if (c.id) t.dataset.consumerId = c.id;
          var main = el(canEdit ? "button" : "div", "energy-flow-main", t);
          if (canEdit) {
            main.type = "button";
            main.setAttribute("aria-label", c.title + " bearbeiten");
            main.addEventListener("click", function () {
              var total = cfg.consumers.filter(function (item) { return item.id; }).length;
              openConsumerDialog(c, i + 1, total, main);
            });
          }
          var holder = icon(c.icon || "plug", "ico", main);
          holder.firstChild.classList.add("energy-flow-icon-default");
          glyph("pencil", "energy-flow-icon-edit", holder);
          var text = el("span", "energy-flow-copy", main);
          el("b", "", text).textContent = c.title;
          var subtitles = el("span", "energy-flow-subtitles", text);
          el("span", "energy-flow-state-copy", subtitles).textContent = c.state || "Bereit";
          if (canEdit) el("span", "energy-flow-edit-copy", subtitles).textContent = "Klicken zum Bearbeiten";
          if (c.kw > 0) edges.push({ from: "hub", to: "consumer-" + i, kw: c.kw, stops: [{ at: 0, c: LOAD }, { at: 1, c: LOAD }] });
          var rcol = el("span", "rcol", t);
          el("span", "prio", rcol).textContent = String(i + 1);
          var canReorder = Boolean(c.id && cfg.addHint);
          if (canReorder) {
            movableIndex += 1;
            (function (myIndex) {
              var grip = el("button", "drag", rcol);
              grip.type = "button";
              glyph("grip-vertical", "", grip);
              grip.setAttribute("aria-label",
                "Priorität von " + c.title + ", Position " + (i + 1) + " von " + cfg.consumers.length + ": Pfeiltasten oder ziehen");
              grip.addEventListener("keydown", function (event) {
                var delta = event.key === "ArrowUp" ? -1 : event.key === "ArrowDown" ? 1 : 0;
                if (!delta) return;
                event.preventDefault();
                var previous = cfg.consumers.slice();
                if (reorderConsumers(cfg, myIndex, myIndex + delta)) {
                  rebuild(c.id);
                  persist(previous, c.id);
                  announce(c.title + " ist jetzt Position " + (myIndex + delta + 1) + " von " + cfg.consumers.length + ".");
                }
              });
              // Dragging arms only from the grip, so tile text stays selectable.
              grip.addEventListener("mousedown", function () { t.draggable = true; });
              grip.addEventListener("mouseup", function () { t.draggable = false; });
              t.addEventListener("dragstart", function (event) {
                event.dataTransfer.effectAllowed = "move";
                event.dataTransfer.setData("text/plain", c.id);
                t.classList.add("dragging");
                draggingTile = t;
              });
              t.addEventListener("dragend", function () {
                t.classList.remove("dragging");
                t.draggable = false;
                if (!draggingTile) return; // Drop auf der Rail hat bereits übernommen
                draggingTile = null;
                rebuild(c.id); // kein Rail-Drop: Vorschau verwerfen
              });
            })(movableIndex);
          } else {
            var staticGrip = el("span", "drag drag-static", rcol);
            glyph("grip-vertical", "", staticGrip);
          }
        });
        if (cfg.addHint) {
          var ghost = el("div", "energy-flow-big ghost", rail);
          var add = el("button", "energy-flow-main", ghost);
          add.type = "button";
          add.setAttribute("aria-label", "Verbraucher hinzufügen");
          icon("plus", "plus", add);
          var gtext = el("span", "energy-flow-copy", add);
          el("b", "", gtext).textContent = "Verbraucher hinzufügen";
          var gsub = el("span", "energy-flow-subtitles", gtext);
          el("span", "energy-flow-state-copy", gsub).textContent = "steuern oder nur beobachten";
          add.addEventListener("click", function () {
            var total = cfg.consumers.filter(function (item) { return item.id; }).length;
            openConsumerDialog(null, total + 1, total, add);
          });
        }
      }

      var bands = sinkBands(cfg);
      edges.forEach(function (e) {
        if (!e.stops) e.stops = bandStops(bands);
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
          var w = 20 + 22 * Math.sqrt(e.kw / maxKw);
          var h = w / 2;
          // One closed polygon per ribbon: sides offset by half the width,
          // ending in a chisel tip exactly as wide as the line (no lateral
          // overhang); tip length 0.2*w keeps the point angle >= 135deg.
          var tipLen = Math.max(2.5, 0.2 * w);
          var ex = x2 - dir[0] * tipLen, ey = y2 - dir[1] * tipLen;
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

      currentDraw = draw;
      requestAnimationFrame(draw);
    }

    wrap.classList.add("is-enhanced");
    build();
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
