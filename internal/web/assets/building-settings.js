(() => {
  const legacySection = window.location.hash.replace(/^#/, "");
  if (["overview", "contacts", "units", "appearance"].includes(legacySection) && !new URL(window.location.href).searchParams.has("section")) {
    const target = new URL(window.location.href);
    target.searchParams.set("section", legacySection);
    target.hash = "";
    window.location.replace(target.toString());
    return;
  }

  let activeDialog = null;
  let dialogTrigger = null;
  const dialogSelector = ".unit-dialog-shell";
  const syncUnitDialog = () => {
    const target = window.location.hash ? document.getElementById(decodeURIComponent(window.location.hash.slice(1))) : null;
    const nextDialog = target?.matches(dialogSelector) ? target : null;
    activeDialog = nextDialog;
    document.querySelectorAll(dialogSelector).forEach((dialog) => dialog.classList.toggle("is-open", dialog === activeDialog));
    document.body.classList.add("unit-dialog-enhanced");
    document.body.classList.toggle("unit-dialog-open", Boolean(activeDialog));
    if (activeDialog) {
      window.setTimeout(() => (activeDialog.querySelector("input:not([type=hidden]), select, button, a") || activeDialog).focus(), 0);
    } else if (dialogTrigger) {
      dialogTrigger.focus();
      dialogTrigger = null;
    }
  };

  document.addEventListener("click", (event) => {
    const opener = event.target.closest('a[href^="#unit-"]');
    if (opener) dialogTrigger = opener;
    const closer = event.target.closest("[data-unit-dialog-close]");
    if (!closer) return;
    event.preventDefault();
    const target = new URL(window.location.href);
    target.hash = "";
    window.history.replaceState(null, "", target.toString());
    syncUnitDialog();
  });
  window.addEventListener("hashchange", syncUnitDialog);
  document.addEventListener("keydown", (event) => {
    if (event.key !== "Escape" || !activeDialog) return;
    event.preventDefault();
    const target = new URL(window.location.href);
    target.hash = "";
    window.history.replaceState(null, "", target.toString());
    syncUnitDialog();
  });
  syncUnitDialog();

  const appearance = document.querySelector("[data-appearance-workspace]");
  if (appearance) {
    const abbreviation = appearance.querySelector("#brand-abbreviation");
    const abbreviationPreviews = appearance.querySelectorAll("[data-brand-preview-abbreviation]");
    const count = appearance.querySelector("[data-abbreviation-count]");
    const markPreviews = appearance.querySelectorAll("[data-brand-preview-mark]");
    const iconInputs = appearance.querySelectorAll('input[name="brand_icon"]');
    const heroInput = appearance.querySelector("#hero-image");
    const heroPreviews = appearance.querySelectorAll("[data-hero-preview], [data-hero-stage]");
    const heroFileName = appearance.querySelector("[data-hero-file-name]");
    let heroObjectURL = "";

    const updateAbbreviation = () => {
      const value = abbreviation.value.trim() || "HAUS";
      abbreviationPreviews.forEach((node) => { node.textContent = value; });
      if (count) count.textContent = `${abbreviation.value.length} / 12`;
    };
    abbreviation?.addEventListener("input", updateAbbreviation);

    iconInputs.forEach((input) => input.addEventListener("change", () => {
      if (!input.checked) return;
      const source = input.closest(".brand-icon-choice")?.querySelector("svg");
      if (!source) return;
      markPreviews.forEach((node) => node.replaceChildren(source.cloneNode(true)));
    }));

    heroInput?.addEventListener("change", () => {
      const file = heroInput.files?.[0];
      if (!file) return;
      if (heroObjectURL) URL.revokeObjectURL(heroObjectURL);
      heroObjectURL = URL.createObjectURL(file);
      heroPreviews.forEach((image) => { image.src = heroObjectURL; });
      if (heroFileName) heroFileName.textContent = file.name;
    });
    window.addEventListener("pagehide", () => { if (heroObjectURL) URL.revokeObjectURL(heroObjectURL); });

    appearance.querySelectorAll("[data-preview-tab]").forEach((tab) => tab.addEventListener("click", () => {
      const target = tab.dataset.previewTab;
      appearance.querySelectorAll("[data-preview-tab]").forEach((item) => {
        const active = item === tab;
        item.classList.toggle("is-active", active);
        item.setAttribute("aria-pressed", String(active));
      });
      appearance.querySelectorAll("[data-preview-panel]").forEach((panel) => { panel.hidden = panel.dataset.previewPanel !== target; });
    }));
  }

  const button = document.querySelector("[data-geocode-address]");
  if (!button) return;

  const form = button.closest("form");
  const address = form?.querySelector("#building-address");
  const position = form?.querySelector("#building-map-position");
  const status = form?.querySelector("[data-geocode-status]");
  const results = form?.querySelector("[data-geocode-results]");
  const preview = form?.querySelector("[data-map-preview]");
  const label = form?.querySelector("[data-map-label]");
  const coordinates = form?.querySelector("[data-map-coordinates]");
  if (!form || !address || !position || !status || !results || !preview || !label || !coordinates) return;

  const icon = () => {
    const pin = document.createElement("span");
    pin.className = "map-pin";
    pin.setAttribute("aria-hidden", "true");
    pin.innerHTML = '<svg viewBox="0 0 24 24"><path d="m3 11 9-7 9 7"/><path d="M5 10v10h14V10"/></svg>';
    return pin;
  };

  const selectResult = (item) => {
    position.value = item.position;
    label.textContent = "Adresse gefunden";
    coordinates.textContent = `${item.label} · ${item.position}`;
    status.textContent = "Standort ausgewählt. Mit Speichern übernehmen.";
    status.classList.add("ok");
    preview.replaceChildren();
    item.tiles.forEach((tile) => {
      const image = document.createElement("img");
      image.src = tile.url;
      image.alt = "";
      image.style.cssText = tile.style;
      preview.append(image);
    });
    preview.append(icon());
    results.hidden = true;
  };

  button.addEventListener("click", async () => {
    if (!address.value.trim()) {
      address.focus();
      status.textContent = "Bitte zuerst eine Adresse eingeben.";
      status.classList.remove("ok");
      return;
    }
    button.disabled = true;
    status.textContent = "Adresse wird gesucht …";
    status.classList.remove("ok");
    results.hidden = true;
    try {
      const endpoint = `${form.action.replace(/\/$/, "")}/geocode`;
      const response = await fetch(endpoint, {
        method: "POST",
        headers: { "Content-Type": "application/x-www-form-urlencoded;charset=UTF-8" },
        body: new URLSearchParams({ address: address.value.trim() }),
      });
      if (!response.ok) throw new Error((await response.text()).trim());
      const payload = await response.json();
      if (!Array.isArray(payload.results) || payload.results.length === 0) throw new Error("Keine passende Adresse gefunden.");

      results.replaceChildren();
      payload.results.forEach((item, index) => {
        const option = document.createElement("button");
        option.type = "button";
        option.className = "geocode-result";
        const title = document.createElement("strong");
        title.textContent = index === 0 ? "Bester Treffer" : `Weiterer Treffer ${index}`;
        const detail = document.createElement("span");
        detail.textContent = item.label;
        option.append(title, detail);
        option.addEventListener("click", () => selectResult(item));
        results.append(option);
      });
      results.hidden = false;
      status.textContent = payload.results.length === 1 ? "Treffer prüfen und auswählen." : `${payload.results.length} Treffer prüfen und auswählen.`;
      results.querySelector("button")?.focus();
    } catch (error) {
      status.textContent = error instanceof Error && error.message ? error.message : "Die Adresse konnte nicht gesucht werden.";
    } finally {
      button.disabled = false;
    }
  });
})();
