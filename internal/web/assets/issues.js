// Focused three-step resident issue creation.
(function () {
  function radioValue(form, name) {
    var checked = form.querySelector('input[name="' + name + '"]:checked');
    return checked ? checked.value : "";
  }

  function visibleStepFields(step) {
    return Array.prototype.slice.call(step.querySelectorAll("input, select, textarea"))
      .filter(function (field) { return !field.disabled && field.type !== "hidden"; });
  }

  function stepIsValid(step) {
    var fields = visibleStepFields(step);
    for (var index = 0; index < fields.length; index += 1) {
      if (!fields[index].checkValidity()) {
        fields[index].reportValidity();
        return false;
      }
    }
    return true;
  }

  function suggestedTitle(body) {
    var clean = String(body || "").replace(/\s+/g, " ").trim();
    if (!clean) return "";
    var firstSentence = clean.split(/[.!?](?:\s|$)/)[0] || clean;
    var title = firstSentence.slice(0, 76).trim();
    if (title.length < firstSentence.length) title = title.replace(/\s+\S*$/, "") + "…";
    return title;
  }

  function locationSummary(form) {
    var type = radioValue(form, "location_type");
    var base = type === "own-unit" ? "Eigene Einheit" : "Gemeinschaftsbereich";
    var detail = form.elements.location_detail.value.trim();
    return detail ? base + " · " + detail : base;
  }

  function updateReview(form) {
    var body = form.elements.body.value.trim();
    var files = form.elements.attachments.files || [];
    form.querySelector('[data-issue-summary="category"]').textContent = radioValue(form, "category") || "—";
    form.querySelector('[data-issue-summary="body"]').textContent = body || "—";
    form.querySelector('[data-issue-summary="location"]').textContent = locationSummary(form);
    form.querySelector('[data-issue-summary="files"]').textContent =
      files.length === 0 ? "Keine" : files.length === 1 ? files[0].name : files.length + " Dateien";
    if (!form.elements.title.dataset.edited) form.elements.title.value = suggestedTitle(body);
  }

  function enhance(form) {
    var steps = Array.prototype.slice.call(form.querySelectorAll("[data-issue-step]"));
    var current = 0;
    var panel = form.closest(".issue-create-panel");
    form.classList.add("is-enhanced");

    function show(index) {
      current = Math.max(0, Math.min(index, steps.length - 1));
      steps.forEach(function (step, stepIndex) { step.hidden = stepIndex !== current; });
      if (current === steps.length - 1) updateReview(form);
      var heading = steps[current].querySelector("h3");
      if (heading) heading.setAttribute("tabindex", "-1");
      if (heading) heading.focus({ preventScroll: true });
      if (panel) panel.scrollIntoView({ block: "start", behavior: "smooth" });
    }

    form.addEventListener("click", function (event) {
      var next = event.target.closest("[data-issue-next]");
      var back = event.target.closest("[data-issue-back]");
      if (next) {
        if (stepIsValid(steps[current])) show(current + 1);
        return;
      }
      if (back) show(current - 1);
      if (event.target.closest(".wizard-cancel") && panel) panel.removeAttribute("open");
    });
    form.elements.title.addEventListener("input", function () {
      form.elements.title.dataset.edited = "true";
    });
    show(0);
  }

  Array.prototype.forEach.call(document.querySelectorAll("[data-issue-wizard]"), enhance);
}());
