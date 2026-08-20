// Data-driven energy-flow renderer for the cockpit (HAUSV-439, HAUSV-441).
//
// Each [data-energy-flow] container carries its config as a child
// <script type="application/json"> (CSP-safe: no inline executable JS, no
// server HTML for user-named tiles — every string lands via textContent).
// The renderer builds the tiles into fixed slots (producers top, storage
// left, grid bottom, consumers right, home centre) and draws the flows as
// calm translucent SVG lanes. Lane width still encodes power; small coloured
// dots move from source to destination and show direction without arrowheads.
// Source ribbons carry proportional colour bands — each destination owns a
// share of the line length matching its kW share, softly blended. Stroke
// width scales with sqrt(kW). Values are pre-formatted by the server; the
// renderer only appends the small raised unit marker.
//
// Consumers with an id can be reordered by dragging their tile or pressing
// the arrow keys on the grip button; the order is persisted through
// POST /app/energie/verbraucher/reihenfolge and rolled back if that fails.
// Every visible consumer, including the configured parking charger, has a
// durable id and participates in the same edit/reorder contract.
(function () {
  "use strict";

  var ORDER_ENDPOINT = "/app/energie/verbraucher/reihenfolge";
  var LIVE_ENDPOINT = "/app/energie/live";
  var LIVE_REFRESH_MS = 10000;

  // The server emits the names of every SVG in the pinned, locally vendored
  // Lucide package. Both the picker and flow renderer use that exact whitelist;
  // configuration can therefore never inject markup or a remote URL.
  var iconManifestNode = document.getElementById("energy-lucide-icon-names");
  var LUCIDE_ICON_NAMES = [];
  try { LUCIDE_ICON_NAMES = JSON.parse(iconManifestNode ? iconManifestNode.textContent : "[]"); } catch (err) { LUCIDE_ICON_NAMES = []; }
  LUCIDE_ICON_NAMES = LUCIDE_ICON_NAMES.filter(function (name) { return /^[a-z0-9-]+$/.test(name); });
  var ICON_NAMES = LUCIDE_ICON_NAMES.reduce(function (names, name) { names[name] = true; return names; }, {});
  var HUB = "#a24b42", BATT = "#7893a1", GRID = "#b8891f", LOAD = "#3e704c", PV = "#4f7d49";

  function nodeColor(value, fallback) {
    return /^#[0-9a-f]{6}$/i.test(value || "") ? value : fallback;
  }

  function el(tag, className, parent) {
    var node = document.createElement(tag);
    if (className) node.className = className;
    if (parent) parent.appendChild(node);
    return node;
  }

  function glyph(name, className, parent) {
    var safeName = ICON_NAMES[name] ? name : "plug";
    var node = el("span", "energy-ui-icon energy-ui-icon-" + safeName + (className ? " " + className : ""), parent);
    var localURL = 'url("/assets/icons/lucide/' + safeName + '.svg")';
    node.style.webkitMaskImage = localURL;
    node.style.maskImage = localURL;
    node.setAttribute("aria-hidden", "true");
    return node;
  }

  function icon(name, holderClass, parent) {
    var holder = el("span", holderClass, parent);
    glyph(name, "", holder);
    holder.setAttribute("aria-hidden", "true");
    return holder;
  }

  // Number + quiet raised unit ("0,52" + "kW"); strings come pre-formatted.
  function appendValue(node, value, unit) {
    node.appendChild(document.createTextNode(value || "–"));
    if (unit) {
      var u = el("span", "u", node);
      u.textContent = "\u00a0" + unit;
    }
  }

  function valueLine(parent, value, unit, className) {
    var strong = el("strong", className || "", parent);
    appendValue(strong, value, unit);
    return strong;
  }

  function tile(kind, iconName, parent, color) {
    var node = el("div", "energy-flow-tile k-" + kind, parent);
    node.style.setProperty("--energy-node-color", color);
    icon(iconName, "ico", node);
    el("div", "", node);
    return node;
  }

  function metricFromLegacy(label, formatted) {
    if (!formatted || formatted === "–") return null;
    var match = String(formatted).match(/^(.*?)(?:\u00a0|\s)(kWh|MWh|kW|W|%|€)$/);
    return {
      label: label || "Weitere Kennzahl",
      value: match ? match[1] : formatted,
      unit: match ? match[2] : "",
    };
  }

  function metricList(metrics, fallbackLabel, fallbackValue) {
    if (Array.isArray(metrics) && metrics.length) return metrics.filter(function (metric) { return metric && metric.value; });
    var fallback = metricFromLegacy(fallbackLabel, fallbackValue);
    return fallback ? [fallback] : [];
  }

  function quietMetrics(parent, metrics, fallbackLabel, fallbackValue, className) {
    var values = metricList(metrics, fallbackLabel, fallbackValue);
    if (!values.length) return;
    var row = el("span", className || "energy-flow-quiet-row", parent);
    values.forEach(function (metric) {
      var item = el("span", "energy-flow-quiet-metric", row);
      var value = el("b", "", item);
      appendValue(value, metric.value, metric.unit);
      el("small", "", item).textContent = metric.label || "Weitere Kennzahl";
    });
  }

  function sinkBands(cfg) {
    var bands = [];
    if (cfg.storage && cfg.storage.flow > 0 && cfg.storage.mode !== "entlädt") bands.push({ c: nodeColor(cfg.storage.color, BATT), kw: cfg.storage.flow });
    (cfg.consumers || []).forEach(function (c) { if (c.kw > 0) bands.push({ c: nodeColor(c.color, LOAD), kw: c.kw }); });
    if (cfg.grid && cfg.grid.kw > 0 && cfg.grid.dir === "export") bands.push({ c: nodeColor(cfg.grid.color, GRID), kw: cfg.grid.kw });
    bands.push({ c: nodeColor(cfg.home && cfg.home.color, HUB), kw: Math.max(cfg.home && cfg.home.kw > 0 ? cfg.home.kw : 0, 0.01) });
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
  var consumerIconValue = consumerDialog && consumerDialog.querySelector("[data-consumer-icon-value]");
  var consumerIconPresets = consumerDialog && consumerDialog.querySelector("[data-consumer-icon-presets]");
  var consumerIconResults = consumerDialog && consumerDialog.querySelector("[data-consumer-icon-results]");
  var consumerIconResultNote = consumerDialog && consumerDialog.querySelector("[data-consumer-icon-result-note]");
  function chooseConsumerIcon(name) {
    var safeName = ICON_NAMES[name] ? name : "plug";
    if (consumerIconValue) consumerIconValue.value = safeName;
    [].forEach.call(consumerDialog.querySelectorAll('[name="icon_choice"]'), function (radio) {
      radio.checked = radio.value === safeName;
    });
  }
  function consumerIconChoice(name, label) {
    var choice = el("label", "energy-icon-choice", consumerIconResults);
    choice.dataset.consumerIconChoice = label + " " + name;
    var radio = el("input", "", choice);
    radio.type = "radio";
    radio.name = "icon_choice";
    radio.value = name;
    radio.checked = Boolean(consumerIconValue && consumerIconValue.value === name);
    radio.addEventListener("change", function () { if (radio.checked) chooseConsumerIcon(name); });
    glyph(name, "", choice);
    el("span", "", choice).textContent = label;
  }
  if (consumerDialog) {
    consumerDialog.addEventListener("change", function (event) {
      if (event.target && event.target.name === "icon_choice" && event.target.checked) chooseConsumerIcon(event.target.value);
    });
  }
  if (consumerIconSearch) {
    consumerIconSearch.addEventListener("input", function () {
      var query = consumerIconSearch.value.trim().toLocaleLowerCase("de");
      if (!query) {
        consumerIconPresets.hidden = false;
        consumerIconResults.hidden = true;
        consumerIconResults.replaceChildren();
        consumerIconResultNote.hidden = true;
        return;
      }
      var presetMatches = [];
      [].forEach.call(consumerIconPresets.querySelectorAll("[data-consumer-icon-choice]"), function (choice) {
        if (choice.dataset.consumerIconChoice.toLocaleLowerCase("de").indexOf(query) !== -1) {
          var radio = choice.querySelector("input");
          presetMatches.push({ name: radio.value, label: choice.querySelector("span:last-child").textContent });
        }
      });
      var seen = {};
      var matches = [];
      presetMatches.concat(LUCIDE_ICON_NAMES.filter(function (name) { return name.indexOf(query) !== -1; }).map(function (name) {
        return { name: name, label: name };
      })).forEach(function (item) {
        if (!seen[item.name]) { seen[item.name] = true; matches.push(item); }
      });
      consumerIconResults.replaceChildren();
      matches.slice(0, 100).forEach(function (item) { consumerIconChoice(item.name, item.label); });
      consumerIconPresets.hidden = true;
      consumerIconResults.hidden = false;
      consumerIconResultNote.hidden = false;
      consumerIconResultNote.textContent = matches.length > 100
        ? matches.length + " Treffer · die ersten 100 werden gezeigt, bitte Suche verfeinern."
        : matches.length + (matches.length === 1 ? " Symbol gefunden." : " Symbole gefunden.");
    });
  }

  var measurementRequest = null;
  function measurementSelect(name) {
    return consumerDialog && consumerDialog.querySelector('[name="' + name + '"]');
  }
  function seedMeasurementSelect(name, current) {
    var select = measurementSelect(name);
    if (!select) return;
    select.replaceChildren();
    var empty = document.createElement("option");
    empty.value = "";
    empty.textContent = "Nicht zugeordnet";
    select.appendChild(empty);
    if (current) {
      var existing = document.createElement("option");
      existing.value = current;
      existing.textContent = current + " · bisher zugeordnet";
      select.appendChild(existing);
      select.value = current;
    }
  }
  function populateMeasurementSelect(slot, current, assetID, entities) {
    var select = measurementSelect(slot.name);
    if (!select) return;
    seedMeasurementSelect(slot.name, "");
    entities.filter(function (entity) { return entity.kind === slot.kind; }).forEach(function (entity) {
      var option = document.createElement("option");
      option.value = entity.entityId;
      option.textContent = entity.name + (entity.unit ? " · " + entity.unit : "") + " · " + entity.entityId;
      var assignedElsewhere = Boolean(entity.entityId !== current && entity.assignedAssetName && (!entity.assignedAssetId || entity.assignedAssetId !== assetID));
      if (assignedElsewhere) {
        option.disabled = true;
        option.textContent += " · verwendet für " + entity.assignedAssetName;
      }
      select.appendChild(option);
    });
    if (current && ![].some.call(select.options, function (option) { return option.value === current; })) {
      var retained = document.createElement("option");
      retained.value = current;
      retained.textContent = current + " · bisher zugeordnet";
      select.appendChild(retained);
    }
    select.value = current || "";
  }
  function loadConsumerMeasurements(item) {
    var status = consumerDialog.querySelector("[data-consumer-measurement-status]");
    var fields = consumerDialog.querySelector("[data-consumer-measurement-fields]");
    var slots = item.measurements || [
      { name: "consumer_power_entity", label: "Aktuelle Leistung", kind: "power", entity: item.powerEntity || "" },
      { name: "consumer_energy_entity", label: "Energiezähler", kind: "energy", entity: item.energyEntity || "" },
      { name: "consumer_soc_entity", label: "Ladestand", kind: "percentage", entity: item.socEntity || "" },
    ];
    if (fields) {
      fields.replaceChildren();
      slots.forEach(function (slot) {
        var label = el("label", "energy-consumer-field", fields);
        el("span", "", label).textContent = slot.label;
        var select = el("select", "", label);
        select.name = slot.name;
        seedMeasurementSelect(slot.name, slot.entity || "");
      });
    }
    if (status) status.textContent = "Home-Assistant-Entities werden geladen …";
    if (!measurementRequest) {
      var firstPathSegment = window.location.pathname.split("/").filter(Boolean)[0] || "";
      var tenantPrefix = firstPathSegment === "app" ? "" : "/" + firstPathSegment;
      measurementRequest = fetch(tenantPrefix + "/app/energie/verbraucher/messwerte", { credentials: "same-origin" }).then(function (response) {
        if (!response.ok || response.redirected) throw new Error("measurement options unavailable");
        return response.json();
      });
    }
    measurementRequest.then(function (payload) {
      slots.forEach(function (slot) {
        populateMeasurementSelect(slot, slot.entity || "", item.id || "", payload.entities || []);
      });
      if (status) status.textContent = payload.message || "Home Assistant · nur gelesen";
    }).catch(function () {
      if (status) status.textContent = "Home Assistant ist gerade nicht erreichbar. Bestehende Zuordnungen bleiben erhalten.";
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
    setDialogValue("node_type", item.nodeType || "consumer");
    setDialogValue("name", item.title || "");
    setDialogValue("kind", item.kind || "other");
    setDialogValue("rated_power_kw", item.ratedPower || "");
    setDialogValue("flexibility", item.flexibility || "unknown");
    setDialogValue("color", nodeColor(item.color, LOAD));
    setDialogValue("secondary_label", item.secondaryLabel || "");
    setDialogValue("stale_after_minutes", item.staleAfterMinutes || "");

    var priorityField = consumerDialog.querySelector('[name="priority"]');
    var priorityWrap = consumerDialog.querySelector("[data-consumer-priority-field]");
    var staleField = consumerDialog.querySelector("[data-consumer-stale-field]");
    var recommendations = consumerDialog.querySelector("[data-consumer-recommendations]");
    var isRailItem = !item.nodeType || item.nodeType === "consumer" || item.nodeType === "parking";
    if (priorityWrap) priorityWrap.hidden = !isRailItem;
    if (staleField) staleField.hidden = !isRailItem;
    if (recommendations) recommendations.hidden = !isRailItem;
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
    chooseConsumerIcon(wantedIcon);

    var title = consumerDialog.querySelector("[data-consumer-dialog-title]");
    var context = consumerDialog.querySelector("[data-consumer-dialog-context]");
    var submit = consumerDialog.querySelector("[data-consumer-submit]");
    var isSystemNode = Boolean(item.nodeType && item.nodeType !== "consumer" && item.nodeType !== "parking");
    if (title) title.textContent = isSystemNode ? "Energiefluss bearbeiten" : (item.id ? "Verbraucher bearbeiten" : "Verbraucher hinzufügen");
    if (context) context.textContent = isSystemNode ? "Name, Farbe, Symbol und passende Home-Assistant-Messwerte direkt anpassen." : (item.id ? "Name, Priorität, Farbe, Symbol und Messwerte direkt anpassen." : "Neuen Verbraucher mit optionalen Messwerten anlegen.");
    if (submit) submit.textContent = item.id ? "Änderungen speichern" : "Verbraucher hinzufügen";

    var remove = consumerDialog.querySelector("[data-consumer-delete]");
    var removeConfirm = consumerDialog.querySelector("[data-consumer-delete-confirm]");
    if (remove) {
      remove.hidden = !item.id || !item.deletable;
      remove.disabled = remove.hidden;
    }
    if (removeConfirm) removeConfirm.hidden = true;
    if (consumerIconSearch) {
      consumerIconSearch.value = "";
      consumerIconSearch.dispatchEvent(new Event("input"));
    }
    loadConsumerMeasurements(item);
    consumerDialog._returnFocus = trigger || null;
    if (!consumerDialog.open) consumerDialog.showModal();
    window.setTimeout(function () {
      var nameField = consumerDialog.querySelector('[name="name"]');
      if (nameField) nameField.focus();
    }, 0);
  }

  if (consumerDialog) {
    var removeButton = consumerDialog.querySelector("[data-consumer-delete]");
    var removeConfirm = consumerDialog.querySelector("[data-consumer-delete-confirm]");
    var removeCancel = consumerDialog.querySelector("[data-consumer-delete-cancel]");
    if (removeButton && removeConfirm) removeButton.addEventListener("click", function () {
      removeButton.hidden = true;
      removeConfirm.hidden = false;
      var confirmButton = removeConfirm.querySelector('button[type="submit"]');
      if (confirmButton) confirmButton.focus();
    });
    if (removeCancel && removeButton && removeConfirm) removeCancel.addEventListener("click", function () {
      removeConfirm.hidden = true;
      removeButton.hidden = false;
      removeButton.focus();
    });
  }

  function makeFlowNodeEditable(node, item) {
    if (!item || !item.editable) return;
    node.classList.add("editable");
    node.tabIndex = 0;
    node.setAttribute("role", "button");
    node.setAttribute("aria-label", (item.label || "Energiefluss") + " bearbeiten");
    var holder = node.querySelector(".ico");
    if (holder && holder.firstChild) {
      holder.firstChild.classList.add("energy-flow-icon-default");
      glyph("pencil", "energy-flow-icon-edit", holder);
    }
    function open() {
      openConsumerDialog({
        id: item.id,
        title: item.label,
        icon: item.icon,
        color: item.color,
        secondaryLabel: item.secondaryLabel,
        nodeType: item.nodeType,
        measurements: item.measurements || [],
        deletable: false,
      }, 1, 1, node);
    }
    node.addEventListener("click", open);
    node.addEventListener("keydown", function (event) {
      if (event.key !== "Enter" && event.key !== " ") return;
      event.preventDefault();
      open();
    });
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

    // Swap data inside the fixed flow area; the page, scroll position and
    // surrounding cockpit stay untouched.
    wrap._energyFlowUpdate = function (nextConfig) {
      if (!nextConfig) return;
      cfg = nextConfig;
      configNode.textContent = JSON.stringify(nextConfig);
      // Keep the trigger node alive while its edit dialog is open so Escape
      // can return focus reliably. The newest data is already retained above.
      if (consumerDialog && consumerDialog.open) {
        wrap._energyFlowPending = true;
        return;
      }
      rebuild("");
    };
    if (consumerDialog) consumerDialog.addEventListener("close", function () {
      // openConsumerDialog stores the trigger on _returnFocus, but nothing ever read it, so
      // closing the dialog left focus on <body> and a keyboard user lost their place. Native
      // <dialog> restoration cannot cover for that here: this dialog deliberately moves focus
      // inside itself (the name field on open, the remove button when a delete is cancelled).
      // announcements.js and building-settings.js both restore their trigger explicitly; this
      // one only looked as though it did.
      var target = consumerDialog._returnFocus;
      consumerDialog._returnFocus = null;
      var restoreFocus = function () {
        var node = target;
        // A pending change rebuilds the flow list, which detaches the original button. Falling
        // back to the equivalent trigger keeps focus in the list instead of dropping it.
        if (!node || !node.isConnected) node = wrap.querySelector("button.energy-flow-main");
        if (node && typeof node.focus === "function") node.focus();
      };
      if (!wrap._energyFlowPending) {
        restoreFocus();
        return;
      }
      wrap._energyFlowPending = false;
      window.setTimeout(function () { rebuild(""); restoreFocus(); }, 0);
    });

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
        var node = tile("pv", p.icon || "solar-panel", top, nodeColor(p.color, PV));
        if (p.hover) node.title = p.hover;
        node.dataset.edge = "producer-" + i;
        valueLine(node.lastChild, p.value, p.unit);
        el("span", "", node.lastChild).textContent = p.label;
        quietMetrics(node.lastChild, p.metrics, p.secondaryLabel, p.secondary);
        makeFlowNodeEditable(node, p);
        if (p.kw > 0) edges.push({ from: "producer-" + i, to: "hub", kw: p.kw, banded: true });
      });

      if (cfg.storage) {
        var leftSlot = el("div", "energy-flow-slot-left", flow);
        var st = tile("batt", cfg.storage.icon || "battery", leftSlot, nodeColor(cfg.storage.color, BATT));
        if (cfg.storage.hover) st.title = cfg.storage.hover;
        st.dataset.edge = "storage";
        valueLine(st.lastChild, cfg.storage.value, cfg.storage.unit);
        el("span", "", st.lastChild).textContent = cfg.storage.sub;
        quietMetrics(st.lastChild, cfg.storage.metrics, cfg.storage.secondaryLabel, cfg.storage.secondary);
        makeFlowNodeEditable(st, cfg.storage);
        if (cfg.storage.flow > 0) {
          if (cfg.storage.mode === "entlädt") edges.push({ from: "storage", to: "hub", kw: cfg.storage.flow, banded: true });
          else edges.push({ from: "hub", to: "storage", kw: cfg.storage.flow, stops: [{ at: 0, c: nodeColor(cfg.storage.color, BATT) }, { at: 1, c: nodeColor(cfg.storage.color, BATT) }] });
        }
      }

      var hub = tile("hub", cfg.home.icon || "house", flow, nodeColor(cfg.home.color, HUB));
      hub.classList.add("energy-flow-hub2");
      if (cfg.home.hover) hub.title = cfg.home.hover;
      hub.dataset.edge = "hub";
      valueLine(hub.lastChild, cfg.home.value, cfg.home.unit);
      el("span", "", hub.lastChild).textContent = cfg.home.label || "Hausverbrauch";
      quietMetrics(hub.lastChild, cfg.home.metrics, cfg.home.secondaryLabel, cfg.home.secondary);
      makeFlowNodeEditable(hub, cfg.home);

      if (cfg.grid) {
        var bottom = el("div", "energy-flow-slot-bottom", flow);
        var gr = tile("grid", cfg.grid.icon || "utility-pole", bottom, nodeColor(cfg.grid.color, GRID));
        if (cfg.grid.hover) gr.title = cfg.grid.hover;
        gr.dataset.edge = "grid";
        valueLine(gr.lastChild, cfg.grid.value, cfg.grid.unit);
        el("span", "", gr.lastChild).textContent = cfg.grid.label;
        quietMetrics(gr.lastChild, cfg.grid.metrics, cfg.grid.secondaryLabel, cfg.grid.secondary);
        makeFlowNodeEditable(gr, cfg.grid);
        if (cfg.grid.kw > 0) {
          if (cfg.grid.dir === "import") edges.push({ from: "grid", to: "hub", kw: cfg.grid.kw, banded: true });
          else edges.push({ from: "hub", to: "grid", kw: cfg.grid.kw, stops: [{ at: 0, c: nodeColor(cfg.grid.color, GRID) }, { at: 1, c: nodeColor(cfg.grid.color, GRID) }] });
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
          var dataStatus = /^(stale|unavailable|unknown|sleep)$/.test(c.dataStatus || "") ? c.dataStatus : "";
          var t = el("div", "energy-flow-big" + (c.active ? " active" : "") + (canEdit ? " editable" : "") + (dataStatus ? " data-" + dataStatus : "") + (c.age ? " has-data-age" : ""), rail);
          t.style.setProperty("--energy-node-color", nodeColor(c.color, LOAD));
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
          var primary = c.primary && c.primary.value ? c.primary : null;
          var current = c.current && c.current.value ? c.current : null;
          var sameAsPrimary = Boolean(primary && current && primary.value === current.value && primary.unit === current.unit);
          var subtitles = el("span", "energy-flow-subtitles", text);
          if (current && !sameAsPrimary) {
            var currentCopy = el("span", "energy-flow-state-copy energy-flow-current", subtitles);
            appendValue(currentCopy, current.value, current.unit);
          } else if (!primary) {
            el("span", "energy-flow-state-copy", subtitles).textContent = c.state || "Bereit";
          }
          if (canEdit) el("span", "energy-flow-edit-copy", subtitles).textContent = "Klicken zum Bearbeiten";
          if (c.age) {
            var dataMeta = el("span", "energy-flow-data-meta", text);
            if (c.dataLabel) el("span", "energy-flow-data-label", dataMeta).textContent = c.dataLabel;
            el("span", "", dataMeta).textContent = c.age;
          }
          if (primary) {
            var primaryLine = el("span", "energy-flow-primary", text);
            valueLine(primaryLine, primary.value, primary.unit);
            if (primary.label) el("small", "energy-flow-primary-label", primaryLine).textContent = primary.label;
          }
          quietMetrics(text, c.metrics, c.secondaryLabel, c.secondary, "energy-flow-consumer-metrics");
          if (c.kw > 0) edges.push({ from: "hub", to: "consumer-" + i, kw: c.kw, stops: [{ at: 0, c: nodeColor(c.color, LOAD) }, { at: 1, c: nodeColor(c.color, LOAD) }] });
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
          var x1, y1, x2, y2;
          if (vertical) {
            var down = a.cy < b.cy;
            x1 = a.cx; y1 = down ? a.b : a.t; x2 = b.cx; y2 = down ? b.t : b.b;
          } else {
            var right = a.cx < b.cx;
            x1 = right ? a.r : a.l; y1 = a.cy; x2 = right ? b.l : b.r; y2 = b.cy;
          }
          var w = 20 + 22 * Math.sqrt(e.kw / maxKw);
          var d;
          if (vertical) {
            var dy = (y2 - y1) * 0.5;
            d = "M " + x1 + " " + y1 + " C " + x1 + " " + (y1 + dy) + ", " + x2 + " " + (y2 - dy) + ", " + x2 + " " + y2;
          } else {
            var dx = (x2 - x1) * 0.5;
            d = "M " + x1 + " " + y1 + " C " + (x1 + dx) + " " + y1 + ", " + (x2 - dx) + " " + y2 + ", " + x2 + " " + y2;
          }
          var gid = "efg-" + wrap.dataset.energyFlow + "-" + i;
          var stops = "";
          e.stops.forEach(function (st) { stops += '<stop offset="' + st.at.toFixed(3) + '" stop-color="' + st.c + '"/>'; });
          defs += '<linearGradient id="' + gid + '" gradientUnits="userSpaceOnUse" x1="' + x1 + '" y1="' + y1 + '" x2="' + x2 + '" y2="' + y2 + '">' + stops + "</linearGradient>";
          shapes += '<path class="energy-flow-band" d="' + d + '" fill="none" stroke="url(#' + gid + ')" stroke-width="' + w.toFixed(2) + '" stroke-linecap="round"/>';
          shapes += '<path class="energy-flow-dots" d="' + d + '" fill="none" stroke="url(#' + gid + ')" stroke-width="' + Math.min(4, Math.max(2.5, w * 0.09)).toFixed(2) + '" stroke-dasharray="0.1 10" stroke-linecap="round"/>';
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

    var diagram = document.querySelector("[data-energy-flow-diagram]");
    var flow = diagram && diagram.querySelector("[data-energy-flow]");
    var status = diagram && diagram.querySelector("[data-energy-live-status]");
    var label = status && status.querySelector("[data-energy-live-label]");
    var updated = status && status.querySelector("[data-energy-live-updated]");
    if (!diagram || !flow || !status || !updated) return;

    var lastSuccess = Date.now();
    var refreshing = false;
    function updateAge() {
      var seconds = Math.max(0, Math.floor((Date.now() - lastSuccess) / 1000));
      updated.textContent = "Portal abgerufen vor " + seconds + "\u00a0s";
    }
    function markLive() {
      status.classList.remove("is-stale");
      if (label) label.textContent = "Live";
    }
    function markStale() {
      status.classList.add("is-stale");
      if (label) label.textContent = "Abruf gestört";
    }
    function refreshLive() {
      if (refreshing) return;
      refreshing = true;
      fetch(LIVE_ENDPOINT, { credentials: "same-origin", cache: "no-store", headers: { "Accept": "application/json" } })
        .then(function (response) {
          if (!response.ok || response.redirected) throw new Error("live refresh unavailable");
          return response.json();
        })
        .then(function (payload) {
          if (!payload || !payload.flow || typeof flow._energyFlowUpdate !== "function") throw new Error("invalid live refresh");
          flow._energyFlowUpdate(payload.flow);
          var fetchedAt = Date.parse(payload.updatedAt || "");
          lastSuccess = Number.isFinite(fetchedAt) ? fetchedAt : Date.now();
          markLive();
          updateAge();
        })
        .catch(markStale)
        .finally(function () { refreshing = false; });
    }

    updateAge();
    window.setInterval(updateAge, 1000);
    window.setInterval(refreshLive, LIVE_REFRESH_MS);
    document.addEventListener("visibilitychange", function () {
      if (!document.hidden && Date.now() - lastSuccess >= LIVE_REFRESH_MS) refreshLive();
    });
  }

  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init, { once: true });
  else init();
})();
