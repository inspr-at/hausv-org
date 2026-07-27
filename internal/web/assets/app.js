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
