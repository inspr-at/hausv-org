// Shared app behavior: dialogs, one confirm path and POST submit locking.
(function () {
  var lockTimeoutMs = 15000;
  var dialogTriggers = Array.prototype.slice.call(document.querySelectorAll("[data-dialog]"));

  // A browser may restore a protected page from its back/forward cache after
  // logout without contacting the server. Force one network/session check
  // whenever such an authenticated page is restored.
  window.addEventListener("pageshow", function (event) {
    if (!document.body || !document.body.hasAttribute("data-authenticated-app")) return;
    var entries = window.performance && typeof window.performance.getEntriesByType === "function"
      ? window.performance.getEntriesByType("navigation")
      : [];
    var restored = Boolean(event.persisted || (entries[0] && entries[0].type === "back_forward"));
    if (restored) window.location.reload();
  });

  function focusFirstDialogField(dialog) {
    var target = dialog.querySelector(
      "button[data-close-dialog], input:not([type='hidden']), select, textarea, button, a[href], [tabindex]:not([tabindex='-1'])"
    );
    if (target && typeof target.focus === "function") {
      target.focus({ preventScroll: true });
    }
  }

  function setExpanded(id, expanded) {
    dialogTriggers.forEach(function (trigger) {
      if (trigger.dataset.dialog === id) {
        trigger.setAttribute("aria-expanded", expanded ? "true" : "false");
      }
    });
  }

  function currentFullscreenElement() {
    return document.fullscreenElement || document.webkitFullscreenElement || null;
  }

  function fullscreenSurface(dialog) {
    return dialog ? dialog.querySelector("[data-energy-fullscreen-surface]") : null;
  }

  function isEnergyFullscreen(dialog) {
    var active = currentFullscreenElement();
    return Boolean(dialog && ((active && dialog.contains(active)) || dialog.classList.contains("is-fullscreen-fallback")));
  }

  function setFullscreenState(dialog, active) {
    if (!dialog || !dialog.id) return;
    dialogTriggers.forEach(function (trigger) {
      if (trigger.dataset.dialog !== dialog.id || !trigger.hasAttribute("data-energy-fullscreen")) return;
      trigger.setAttribute("aria-pressed", active ? "true" : "false");
      var label = trigger.querySelector("[data-fullscreen-label]");
      if (label) label.textContent = active ? "Vollbild beenden" : "Vollbild";
    });
  }

  function enterFullscreenFallback(dialog) {
    dialog.classList.add("is-fullscreen-fallback");
    dialog._energyWasFullscreen = true;
    setFullscreenState(dialog, true);
  }

  function requestEnergyFullscreen(dialog) {
    var surface = fullscreenSurface(dialog);
    if (!surface) return;
    var request = surface.requestFullscreen || surface.webkitRequestFullscreen;
    if (typeof request !== "function") {
      enterFullscreenFallback(dialog);
      return;
    }
    try {
      var pending = request.call(surface);
      if (pending && typeof pending.catch === "function") {
        pending.catch(function () {
          enterFullscreenFallback(dialog);
        });
      }
    } catch (_) {
      enterFullscreenFallback(dialog);
    }
  }

  function exitEnergyFullscreen(dialog) {
    if (dialog && dialog.classList.contains("is-fullscreen-fallback")) {
      dialog.classList.remove("is-fullscreen-fallback");
      dialog._energyWasFullscreen = false;
      setFullscreenState(dialog, false);
      return null;
    }
    var active = currentFullscreenElement();
    if (!active || !dialog || !dialog.contains(active)) return null;
    var exit = document.exitFullscreen || document.webkitExitFullscreen;
    if (typeof exit !== "function") return null;
    try {
      return exit.call(document);
    } catch (_) {
      return null;
    }
  }

  function toggleEnergyFullscreen(dialog) {
    if (isEnergyFullscreen(dialog)) {
      return exitEnergyFullscreen(dialog);
    }
    requestEnergyFullscreen(dialog);
    return null;
  }

  dialogTriggers.forEach(function (trigger) {
    var id = trigger.dataset.dialog;
    var dialog = id ? document.getElementById(id) : null;
    if (!dialog || typeof dialog.showModal !== "function") return;
    trigger.setAttribute("aria-expanded", "false");
    trigger.addEventListener("click", function () {
      if (!dialog.open) {
        dialog._returnFocus = trigger;
        dialog.showModal();
        setExpanded(id, true);
        window.setTimeout(function () {
          focusFirstDialogField(dialog);
        }, 0);
      }
      if (trigger.hasAttribute("data-energy-fullscreen")) {
        toggleEnergyFullscreen(dialog);
      }
    });
  });

  Array.prototype.forEach.call(document.querySelectorAll("dialog"), function (dialog) {
    dialog.addEventListener("cancel", function (event) {
      if (isEnergyFullscreen(dialog)) {
        event.preventDefault();
        exitEnergyFullscreen(dialog);
        var button = dialog.querySelector("[data-energy-fullscreen]");
        if (button) button.focus({ preventScroll: true });
      }
    });
    dialog.addEventListener("close", function () {
      exitEnergyFullscreen(dialog);
      dialog.classList.remove("is-fullscreen-fallback");
      dialog._energyWasFullscreen = false;
      setExpanded(dialog.id, false);
      setFullscreenState(dialog, false);
      var trigger = dialog._returnFocus;
      if (trigger && document.body.contains(trigger)) {
        trigger.focus({ preventScroll: true });
      }
      dialog._returnFocus = null;
    });
  });

  function syncFullscreenState() {
    var active = currentFullscreenElement();
    Array.prototype.forEach.call(document.querySelectorAll("dialog"), function (dialog) {
      var surface = fullscreenSurface(dialog);
      var isActive = Boolean(surface && active === surface) || dialog.classList.contains("is-fullscreen-fallback");
      setFullscreenState(dialog, isActive);
      if (isActive) {
        dialog._energyWasFullscreen = true;
      } else if (dialog._energyWasFullscreen && dialog.open) {
        dialog._energyWasFullscreen = false;
        var button = dialog.querySelector("[data-energy-fullscreen]");
        if (button) button.focus({ preventScroll: true });
      }
    });
  }
  document.addEventListener("fullscreenchange", syncFullscreenState);
  document.addEventListener("webkitfullscreenchange", syncFullscreenState);

  document.addEventListener("click", function (event) {
    var close = event.target.closest("[data-close-dialog]");
    if (!close) return;
    var dialog = close.closest("dialog");
    if (!dialog || !dialog.open) return;
    var pending = exitEnergyFullscreen(dialog);
    if (pending && typeof pending.finally === "function") {
      pending.finally(function () { dialog.close(); });
    } else {
      dialog.close();
    }
  });

  Array.prototype.forEach.call(document.querySelectorAll("[data-energy-chart-interactive]"), function (chart) {
    var tooltip = chart.querySelector("[data-chart-tooltip]");
    var tooltipTime = chart.querySelector("[data-chart-tooltip-time]");
    var tooltipValues = chart.querySelector("[data-chart-tooltip-values]");
    var samples = Array.prototype.slice.call(chart.querySelectorAll("[data-chart-sample]"));
    var indices = samples.map(function (sample) { return Number(sample.dataset.chartSample); });
    var currentIndex = indices.length ? indices[indices.length - 1] : -1;

    function visibleSVG() {
      return Array.prototype.find.call(chart.querySelectorAll("svg"), function (svg) {
        return svg.getBoundingClientRect().width > 0;
      }) || null;
    }

    function sampleFor(index) {
      return chart.querySelector('[data-chart-sample="' + index + '"]');
    }

    function hitFor(svg, index) {
      return svg ? svg.querySelector('[data-chart-hit][data-index="' + index + '"]') : null;
    }

    function hideSample() {
      if (tooltip) {
        tooltip.hidden = true;
        tooltip.removeAttribute("data-index");
      }
      Array.prototype.forEach.call(chart.querySelectorAll("[data-chart-guide], [data-chart-marker-index]"), function (node) {
        node.setAttribute("hidden", "");
      });
    }

    function showSample(index, hit) {
      var sample = sampleFor(index);
      var svg = hit ? hit.closest("svg") : visibleSVG();
      hit = hit || hitFor(svg, index);
      if (!sample || !svg || !hit || !tooltip || !tooltipTime || !tooltipValues) return;
      currentIndex = index;

      Array.prototype.forEach.call(chart.querySelectorAll("[data-chart-guide], [data-chart-marker-index]"), function (node) {
        node.setAttribute("hidden", "");
      });
      var guide = svg.querySelector("[data-chart-guide]");
      if (guide) {
        guide.setAttribute("x1", hit.dataset.x);
        guide.setAttribute("x2", hit.dataset.x);
        guide.removeAttribute("hidden");
      }
      Array.prototype.forEach.call(svg.querySelectorAll('[data-chart-marker-index="' + index + '"]'), function (marker) {
        marker.removeAttribute("hidden");
      });

      tooltipTime.textContent = sample.dataset.time + " Uhr";
      tooltipValues.textContent = "";
      Array.prototype.forEach.call(sample.querySelectorAll("[data-label]"), function (value) {
        var row = document.createElement("span");
        var swatch = document.createElement("i");
        var label = document.createElement("span");
        var reading = document.createElement("b");
        row.className = "energy-chart-tooltip-row";
        swatch.className = value.dataset.key || "";
        label.textContent = value.dataset.label || "";
        reading.textContent = value.dataset.value || "";
        row.appendChild(swatch);
        row.appendChild(label);
        row.appendChild(reading);
        tooltipValues.appendChild(row);
      });
      tooltip.hidden = false;
      tooltip.dataset.index = String(index);

      var chartBox = chart.getBoundingClientRect();
      var hitBox = hit.getBoundingClientRect();
      var centered = hitBox.left + hitBox.width / 2 - chartBox.left - tooltip.offsetWidth / 2;
      var maximum = Math.max(8, chartBox.width - tooltip.offsetWidth - 8);
      tooltip.style.left = Math.max(8, Math.min(maximum, centered)) + "px";
    }

    Array.prototype.forEach.call(chart.querySelectorAll("[data-chart-hit]"), function (hit) {
      hit.addEventListener("pointermove", function () {
        showSample(Number(hit.dataset.index), hit);
      });
      hit.addEventListener("pointerdown", function () {
        showSample(Number(hit.dataset.index), hit);
      });
    });
    chart.addEventListener("pointerleave", function (event) {
      if (!event.pointerType || event.pointerType === "mouse") hideSample();
    });
    chart.addEventListener("focus", function () {
      if (currentIndex >= 0) showSample(currentIndex);
    });
    chart.addEventListener("blur", hideSample);
    chart.addEventListener("keydown", function (event) {
      if (!indices.length) return;
      var position = Math.max(0, indices.indexOf(currentIndex));
      if (event.key === "ArrowLeft") position = Math.max(0, position - 1);
      else if (event.key === "ArrowRight") position = Math.min(indices.length - 1, position + 1);
      else if (event.key === "Home") position = 0;
      else if (event.key === "End") position = indices.length - 1;
      else return;
      event.preventDefault();
      showSample(indices[position]);
    });
  });

  Array.prototype.forEach.call(document.querySelectorAll("[data-notification-form]"), function (form) {
    var master = form.querySelector("[data-notification-master]");
    var topics = Array.prototype.slice.call(form.querySelectorAll("[data-notification-topic]"));
    var status = form.querySelector("[data-notification-status]");
    var count = form.querySelector("[data-notification-count]");
    if (!master) return;

    function updateNotificationState() {
      var active = topics.filter(function (input) { return input.checked; }).length;
      form.classList.toggle("email-paused", !master.checked);
      if (count) count.textContent = active + " von " + topics.length + " aktiv";
      if (status) {
        status.textContent = master.checked
          ? "Aktuell zu " + active + " von " + topics.length + " Themen."
          : "Der Versand ist derzeit pausiert.";
      }
    }

    master.addEventListener("change", updateNotificationState);
    topics.forEach(function (input) {
      input.addEventListener("change", updateNotificationState);
    });
    updateNotificationState();
  });

  Array.prototype.forEach.call(document.querySelectorAll("[data-home-type-select]"), function (select) {
    var scope = select.closest("form") || document;
    var explanation = scope.querySelector("[data-home-type-explanation]");
    if (!explanation) return;
    var label = explanation.querySelector("[data-home-type-label]");
    var copy = explanation.querySelector("[data-home-type-copy]");
    var unitField = scope.querySelector("[data-home-unit-field]");
    var unitSelect = unitField ? unitField.querySelector("select[name='unit_id']") : null;

    function updateHomeTypeExplanation() {
      var option = select.options[select.selectedIndex];
      if (!option) return;
      if (label) label.textContent = option.dataset.label || option.textContent || "";
      if (copy) copy.textContent = option.dataset.description || "";
      if (unitField) unitField.hidden = select.value !== "apartment";
      if (unitSelect) unitSelect.disabled = select.value !== "apartment";
    }

    select.addEventListener("change", updateHomeTypeExplanation);
    updateHomeTypeExplanation();
  });

  function submitButtons(form, submitter) {
    var buttons = Array.prototype.slice.call(
      form.querySelectorAll("button[type='submit'], input[type='submit']")
    );
    if (submitter && buttons.indexOf(submitter) === -1) {
      buttons.unshift(submitter);
    }
    return buttons;
  }

  function lockForm(form, submitter) {
    if (form.dataset.submitting === "true") return false;
    form.dataset.submitting = "true";
    form.setAttribute("aria-busy", "true");

    // Disabled submit buttons are excluded from the browser's form payload.
    // Preserve the clicked button's name/value before locking it so forms with
    // multiple explicit actions (Back, Continue, Finish) keep their intent.
    var submitterValue = null;
    if (submitter && submitter.name) {
      submitterValue = document.createElement("input");
      submitterValue.type = "hidden";
      submitterValue.name = submitter.name;
      submitterValue.value = submitter.value;
      submitterValue.dataset.submitterValue = "true";
      form.appendChild(submitterValue);
    }

    submitButtons(form, submitter).forEach(function (button) {
      if (!button || button.disabled) return;
      button.dataset.submitText = button.value || button.textContent || "";
      button.disabled = true;
      button.setAttribute("aria-disabled", "true");
      if (button.tagName === "BUTTON" && button.type === "submit") {
        button.textContent = button.dataset.busyLabel || "Bitte warten...";
      }
    });

    window.setTimeout(function () {
      if (!document.body.contains(form)) return;
      form.dataset.submitting = "false";
      form.removeAttribute("aria-busy");
      if (submitterValue && submitterValue.parentNode === form) {
        form.removeChild(submitterValue);
      }
      submitButtons(form, submitter).forEach(function (button) {
        if (!button) return;
        button.disabled = false;
        button.removeAttribute("aria-disabled");
        if (button.tagName === "BUTTON" && button.dataset.submitText) {
          button.textContent = button.dataset.submitText;
        }
      });
    }, form.hasAttribute("data-download-form") ? 2500 : lockTimeoutMs);
    return true;
  }

  document.addEventListener("submit", function (event) {
    var form = event.target.closest("form");
    if (!form || String(form.method || "").toLowerCase() !== "post") return;

    if (form.dataset.confirm && !window.confirm(form.dataset.confirm)) {
      event.preventDefault();
      return;
    }

    if (!lockForm(form, event.submitter)) {
      event.preventDefault();
    }
  });

  // Print / save-as-PDF trigger (CSP-safe: no inline handler).
  document.addEventListener("click", function (event) {
    var trigger = event.target.closest("[data-print]");
    if (!trigger) return;
    event.preventDefault();
    window.print();
  });
})();
