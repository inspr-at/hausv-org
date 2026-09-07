// Focused two-step resident issue creation with a complete no-JS fallback.
(function () {
  function radioValue(form, name) {
    var checked = form.querySelector('input[name="' + name + '"]:checked');
    return checked ? checked.value : "";
  }

  function prefersReducedMotion() {
    return window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  }

  function scrollBehavior() {
    return prefersReducedMotion() ? "auto" : "smooth";
  }

  function normaliseWords(value) {
    return String(value || "").replace(/\s+/g, " ").trim();
  }

  function suggestedTitle(body) {
    var clean = normaliseWords(body);
    if (!clean) return "";
    var cleanChars = Array.from(clean);
    var sentenceEnd = -1;
    cleanChars.some(function (char, index) {
      if (char !== "." && char !== "!" && char !== "?") return false;
      if (index < cleanChars.length - 1 && cleanChars[index + 1] !== " ") return false;
      var prefix = cleanChars.slice(0, index).join("").trim();
      // Keep early punctuation that belongs to short German abbreviations,
      // such as "z. B." and "z.B.", instead of suggesting only "z".
      if (index < cleanChars.length - 1 && Array.from(prefix).length < 8) return false;
      sentenceEnd = index;
      return true;
    });
    var title = sentenceEnd >= 0 ? cleanChars.slice(0, sentenceEnd).join("").trim() : clean;
    if (!title) title = clean;
    var chars = Array.from(title);
    if (chars.length <= 76) return title;
    var first = chars.slice(0, 75);
    var lastSpace = first.lastIndexOf(" ");
    var cut = lastSpace >= 38 ? lastSpace : 75;
    return first.slice(0, cut).join("").trim() + "…";
  }

  function locationSummary(form) {
    var type = radioValue(form, "location_type");
    var base = type === "own-unit" ? "Eigene Einheit" : "Gemeinschaftsbereich";
    var detail = form.elements.location_detail.value.trim();
    return detail ? base + " · " + detail : base;
  }

  function clearFieldError(field, error) {
    if (field) field.removeAttribute("aria-invalid");
    if (!error) return;
    error.textContent = "";
    error.hidden = true;
  }

  function showFieldError(field, error, message) {
    if (field) field.setAttribute("aria-invalid", "true");
    if (error) {
      error.textContent = message;
      error.hidden = false;
    }
  }

  function focusInvalid(field) {
    if (!field) return;
    field.focus({ preventScroll: true });
    field.scrollIntoView({ block: "center", behavior: scrollBehavior() });
  }

  function describeIsValid(form, announce) {
    var body = form.elements.body;
    var bodyError = form.querySelector("#issue-body-error");
    var categoryGroup = form.querySelector(".issue-category-options");
    var categoryError = form.querySelector("#issue-category-error");
    var locationGroup = form.querySelector(".issue-location-options");
    var locationError = form.querySelector("#issue-location-error");

    clearFieldError(body, bodyError);
    clearFieldError(categoryGroup, categoryError);
    clearFieldError(locationGroup, locationError);

    if (!body.value.trim()) {
      if (announce) {
        showFieldError(body, bodyError, "Bitte beschreiben Sie kurz, worum es geht.");
        focusInvalid(body);
      }
      return false;
    }
    if (!radioValue(form, "category")) {
      if (announce) {
        showFieldError(categoryGroup, categoryError, "Bitte wählen Sie die Art des Anliegens.");
        focusInvalid(form.querySelector('input[name="category"]'));
      }
      return false;
    }
    if (!radioValue(form, "location_type")) {
      if (announce) {
        showFieldError(locationGroup, locationError, "Bitte wählen Sie den Bereich.");
        focusInvalid(form.querySelector('input[name="location_type"]'));
      }
      return false;
    }
    return true;
  }

  function updateReview(form) {
    var body = form.elements.body.value.trim();
    var files = form.elements.attachments.files || [];
    var bodySummary = form.querySelector('[data-issue-summary="body"]');
    var expand = form.querySelector("[data-issue-review-expand]");
    form.querySelector('[data-issue-summary="category"]').textContent = radioValue(form, "category") || "—";
    bodySummary.textContent = body || "—";
    form.querySelector('[data-issue-summary="location"]').textContent = locationSummary(form);
    form.querySelector('[data-issue-summary="files"]').textContent =
      files.length === 0 ? "Keine" : files.length === 1 ? files[0].name : files.length + " Dateien ausgewählt";

    bodySummary.classList.add("is-collapsed");
    expand.hidden = body.length <= 320;
    expand.setAttribute("aria-expanded", "false");
    expand.textContent = "Vollständig lesen";

    if (!form.elements.title.value.trim()) {
      form.elements.title.placeholder = suggestedTitle(body) || "Kurzer, passender Titel";
    }
  }

  function historyState(form, step) {
    var state = Object.assign({}, window.history.state || {});
    state.issueWizardForm = form.id;
    state.issueWizardStep = step;
    return state;
  }

  function historyURL(step) {
    var url = new URL(window.location.href);
    url.searchParams.set("new", "1");
    url.searchParams.set("step", step);
    url.hash = "issue-new";
    return url.pathname + url.search + url.hash;
  }

  function enhance(form) {
    var steps = Array.prototype.slice.call(form.querySelectorAll("[data-issue-step]"));
    var panel = form.closest(".issue-create-panel");
    var current = "describe";
    form.classList.add("is-enhanced");
    form.noValidate = true;

    function stepElement(name) {
      return form.querySelector('[data-issue-step="' + name + '"]');
    }

    function setHistory(step, mode) {
      window.history[mode + "State"](historyState(form, step), "", historyURL(step));
    }

    function show(step, options) {
      options = options || {};
      current = step === "review" ? "review" : "describe";
      steps.forEach(function (item) {
        item.hidden = item.dataset.issueStep !== current;
      });
      if (current === "review") updateReview(form);

      var active = stepElement(current);
      var heading = active ? active.querySelector("h3") : null;
      if (heading) heading.setAttribute("tabindex", "-1");
      if (options.scroll !== false && panel) {
        panel.scrollIntoView({ block: "start", behavior: scrollBehavior() });
      }
      if (options.focus !== false && heading) heading.focus({ preventScroll: true });
    }

    function openReview() {
      if (!describeIsValid(form, true)) return;
      setHistory("review", "push");
      show("review");
    }

    form.addEventListener("input", function (event) {
      if (event.target === form.elements.body && event.target.value.trim()) {
        clearFieldError(event.target, form.querySelector("#issue-body-error"));
      }
    });
    form.addEventListener("change", function (event) {
      if (event.target.name === "category") {
        clearFieldError(form.querySelector(".issue-category-options"), form.querySelector("#issue-category-error"));
      }
      if (event.target.name === "location_type") {
        clearFieldError(form.querySelector(".issue-location-options"), form.querySelector("#issue-location-error"));
      }
    });

    form.addEventListener("click", function (event) {
      var next = event.target.closest("[data-issue-next]");
      var back = event.target.closest("[data-issue-back]");
      var expand = event.target.closest("[data-issue-review-expand]");
      var cancel = event.target.closest(".wizard-cancel");
      if (next) {
        openReview();
        return;
      }
      if (back) {
        if (window.history.state && window.history.state.issueWizardForm === form.id && window.history.state.issueWizardStep === "review") {
          window.history.back();
        } else {
          setHistory("describe", "replace");
          show("describe");
        }
        return;
      }
      if (expand) {
        var summary = form.querySelector('[data-issue-summary="body"]');
        var expanded = expand.getAttribute("aria-expanded") === "true";
        summary.classList.toggle("is-collapsed", expanded);
        expand.setAttribute("aria-expanded", expanded ? "false" : "true");
        expand.textContent = expanded ? "Vollständig lesen" : "Weniger anzeigen";
        return;
      }
      if (cancel && panel) {
        panel.removeAttribute("open");
        var url = new URL(window.location.href);
        url.searchParams.delete("new");
        url.searchParams.delete("step");
        url.hash = "issue-own";
        window.history.replaceState({}, "", url.pathname + url.search + url.hash);
        show("describe", { scroll: false, focus: false });
        var summaryControl = panel.querySelector(":scope > summary");
        if (summaryControl) summaryControl.focus();
      }
    });

    form.addEventListener("submit", function (event) {
      if (current !== "review") {
        event.preventDefault();
        event.stopPropagation();
        openReview();
        return;
      }
      if (!describeIsValid(form, false)) {
        event.preventDefault();
        event.stopPropagation();
        setHistory("describe", "replace");
        show("describe", { focus: false });
        describeIsValid(form, true);
      }
    });

    window.addEventListener("popstate", function (event) {
      var state = event.state || {};
      if (state.issueWizardForm !== form.id) return;
      var target = state.issueWizardStep === "review" && describeIsValid(form, false) ? "review" : "describe";
      if (target !== state.issueWizardStep) setHistory(target, "replace");
      show(target);
    });

    // Start focusing only after a deliberate entry action. A native toggle also
    // fires for server-rendered open details, so it must not drive focus/history.
    function enterWizard() {
      if (panel && panel.tagName === "DETAILS") panel.open = true;
      setHistory("describe", "replace");
      show("describe");
    }

    if (panel && panel.tagName === "DETAILS") {
      panel.querySelector(":scope > summary").addEventListener("click", function (event) {
        if (panel.open) return;
        event.preventDefault();
        enterWizard();
      });
    }
    Array.prototype.forEach.call(document.querySelectorAll("[data-issue-open]"), function (link) {
      link.addEventListener("click", function (event) {
        event.preventDefault();
        enterWizard();
      });
    });

    var requested = new URL(window.location.href).searchParams.get("step");
    var initial = requested === "review" && describeIsValid(form, false) ? "review" : "describe";
    // A deliberate entry (?new=1 or #issue-new) records its step in the URL so
    // Back from the review lands on ?step=describe again; an ordinary page load
    // keeps its URL and only gets a history baseline, so Reload does not jump
    // down to the wizard.
    var location = new URL(window.location.href);
    if (location.searchParams.get("new") === "1" || location.hash === "#issue-new") {
      setHistory(initial, "replace");
    } else {
      window.history.replaceState(historyState(form, initial), "", window.location.href);
    }
    show(initial, { scroll: false, focus: false });
  }

  Array.prototype.forEach.call(document.querySelectorAll("[data-issue-wizard]"), enhance);
}());
