// Shared app behavior: dialogs, one confirm path and POST submit locking.
(function () {
  var lockTimeoutMs = 15000;
  var dialogTriggers = Array.prototype.slice.call(document.querySelectorAll("[data-dialog]"));

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

  dialogTriggers.forEach(function (trigger) {
    var id = trigger.dataset.dialog;
    var dialog = id ? document.getElementById(id) : null;
    if (!dialog || typeof dialog.showModal !== "function") return;
    trigger.setAttribute("aria-expanded", "false");
    trigger.addEventListener("click", function () {
      if (!dialog.open) {
        dialog.showModal();
        setExpanded(id, true);
        window.setTimeout(function () {
          focusFirstDialogField(dialog);
        }, 0);
      }
    });
    dialog.addEventListener("close", function () {
      setExpanded(id, false);
      if (document.body.contains(trigger)) {
        trigger.focus({ preventScroll: true });
      }
    });
  });

  document.addEventListener("click", function (event) {
    var close = event.target.closest("[data-close-dialog]");
    if (!close) return;
    var dialog = close.closest("dialog");
    if (dialog && dialog.open) {
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
    var explanation = document.querySelector("[data-home-type-explanation]");
    if (!explanation) return;
    var label = explanation.querySelector("[data-home-type-label]");
    var copy = explanation.querySelector("[data-home-type-copy]");

    function updateHomeTypeExplanation() {
      var option = select.options[select.selectedIndex];
      if (!option) return;
      if (label) label.textContent = option.dataset.label || option.textContent || "";
      if (copy) copy.textContent = option.dataset.description || "";
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
    }, lockTimeoutMs);
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
