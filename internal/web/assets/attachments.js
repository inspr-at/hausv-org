// Attachment previews, selected-file picker and image lightbox.
(function () {
  var triggers = [];
  var currentIndex = -1;
  var overlay;
  var image;
  var caption;
  var prevButton;
  var nextButton;
  var pickerUrls = new WeakMap();

  function fileArray(input) {
    return Array.prototype.slice.call(input.files || []);
  }

  function formatBytes(bytes) {
    if (!bytes) return "0 KB";
    if (bytes < 1024 * 1024) return Math.max(1, Math.round(bytes / 1024)) + " KB";
    return (bytes / (1024 * 1024)).toLocaleString("de-AT", { maximumFractionDigits: 1 }) + " MB";
  }

  function selectedSummary(files) {
    var total = files.reduce(function (sum, file) { return sum + (file.size || 0); }, 0);
    if (files.length === 1) return "1 Datei ausgewählt · " + formatBytes(total);
    return files.length + " Dateien ausgewählt · " + formatBytes(total);
  }

  function shortFileType(file) {
    var type = String(file.type || "").toLowerCase();
    if (type.indexOf("pdf") !== -1) return "PDF";
    if (type.indexOf("image/") === 0) return "Bild";
    var name = String(file.name || "");
    var ext = name.indexOf(".") !== -1 ? name.split(".").pop() : "";
    return ext ? ext.slice(0, 4).toUpperCase() : "Datei";
  }

  function revokePickerUrls(input) {
    var urls = pickerUrls.get(input) || [];
    urls.forEach(function (url) { URL.revokeObjectURL(url); });
    pickerUrls.set(input, []);
  }

  function setInputFiles(input, files) {
    try {
      var transfer = new DataTransfer();
      files.forEach(function (file) { transfer.items.add(file); });
      input.files = transfer.files;
      input.dispatchEvent(new Event("change", { bubbles: true }));
      return true;
    } catch (err) {
      input.value = "";
      input.dispatchEvent(new Event("change", { bubbles: true }));
      return false;
    }
  }

  function renderPicker(input) {
    var picker = input._attachmentPicker;
    if (!picker) return;
    var files = fileArray(input);
    var control = input.closest(".file-control");
    var label = control && control.querySelector("span");
    var status = picker.querySelector(".attachment-picker-status");
    var list = picker.querySelector(".attachment-picker-list");
    revokePickerUrls(input);
    list.textContent = "";
    picker.classList.remove("is-uploading");

    if (!files.length) {
      picker.classList.remove("has-files");
      if (control) control.classList.remove("is-filled");
      if (label) label.textContent = input.dataset.defaultFileLabel || "Datei auswählen";
      status.textContent = "Keine Datei ausgewählt.";
      return;
    }

    picker.classList.add("has-files");
    if (control) control.classList.add("is-filled");
    if (label) label.textContent = files.length === 1 ? "1 Datei ausgewählt" : files.length + " Dateien ausgewählt";
    status.textContent = selectedSummary(files);

    var urls = [];
    files.forEach(function (file, index) {
      var item = document.createElement("div");
      item.className = "attachment-picker-item";

      var thumb = document.createElement("span");
      thumb.className = "attachment-picker-thumb";
      if (String(file.type || "").indexOf("image/") === 0) {
        var url = URL.createObjectURL(file);
        urls.push(url);
        var img = document.createElement("img");
        img.src = url;
        img.alt = "";
        thumb.appendChild(img);
      } else {
        thumb.textContent = shortFileType(file);
      }

      var copy = document.createElement("span");
      copy.className = "attachment-picker-copy";
      var name = document.createElement("span");
      name.className = "attachment-picker-name";
      name.textContent = file.name || "Datei";
      name.title = file.name || "Datei";
      var meta = document.createElement("span");
      meta.className = "attachment-picker-meta";
      meta.textContent = shortFileType(file) + " · " + formatBytes(file.size || 0);
      copy.appendChild(name);
      copy.appendChild(meta);

      var remove = document.createElement("button");
      remove.className = "attachment-picker-remove";
      remove.type = "button";
      remove.setAttribute("aria-label", "Datei entfernen");
      remove.textContent = "\u00d7";
      remove.addEventListener("click", function () {
        var next = fileArray(input).filter(function (_, current) { return current !== index; });
        setInputFiles(input, next);
      });

      item.appendChild(thumb);
      item.appendChild(copy);
      item.appendChild(remove);
      list.appendChild(item);
    });
    pickerUrls.set(input, urls);
  }

  function ensurePicker(input) {
    if (input._attachmentPicker) return;
    input.dataset.attachmentPicker = "true";
    var control = input.closest(".file-control");
    var label = control && control.querySelector("span");
    if (label) input.dataset.defaultFileLabel = label.textContent.trim() || "Datei auswählen";

    var picker = document.createElement("div");
    picker.className = "attachment-picker";
    picker.innerHTML =
      '<div class="attachment-picker-head"><span class="attachment-picker-status">Keine Datei ausgewählt.</span><span>Vor dem Speichern prüfbar</span></div>' +
      '<div class="attachment-picker-list"></div>' +
      '<div class="attachment-picker-progress" aria-hidden="true"><span></span></div>';

    var anchor = input.closest("label") || control || input;
    anchor.insertAdjacentElement("afterend", picker);
    input._attachmentPicker = picker;

    if (control) {
      control.addEventListener("dragover", function (event) {
        event.preventDefault();
        control.classList.add("is-dragover");
      });
      control.addEventListener("dragleave", function () {
        control.classList.remove("is-dragover");
      });
      control.addEventListener("drop", function () {
        control.classList.remove("is-dragover");
      });
    }

    input.addEventListener("change", function () { renderPicker(input); });
    renderPicker(input);
  }

  function enhanceFileInputs() {
    Array.prototype.forEach.call(document.querySelectorAll('input[type="file"]'), ensurePicker);
  }

  function collectTriggers() {
    triggers = Array.prototype.slice.call(document.querySelectorAll("[data-lightbox-src]"));
  }

  function ensureOverlay() {
    if (overlay) return overlay;
    overlay = document.createElement("div");
    overlay.className = "attachment-lightbox";
    overlay.setAttribute("role", "dialog");
    overlay.setAttribute("aria-modal", "true");
    overlay.setAttribute("aria-label", "Bildvorschau");
    overlay.innerHTML =
      '<button class="lightbox-close" type="button" aria-label="Schließen">&times;</button>' +
      '<button class="lightbox-prev" type="button" aria-label="Vorheriges Bild">‹</button>' +
      '<figure><img alt=""><figcaption></figcaption></figure>' +
      '<button class="lightbox-next" type="button" aria-label="Nächstes Bild">›</button>';
    document.body.appendChild(overlay);
    image = overlay.querySelector("img");
    caption = overlay.querySelector("figcaption");
    prevButton = overlay.querySelector(".lightbox-prev");
    nextButton = overlay.querySelector(".lightbox-next");
    overlay.querySelector(".lightbox-close").addEventListener("click", closeLightbox);
    prevButton.addEventListener("click", function () { move(-1); });
    nextButton.addEventListener("click", function () { move(1); });
    overlay.addEventListener("click", function (event) {
      if (event.target === overlay) closeLightbox();
    });
    return overlay;
  }

  function openLightbox(index) {
    collectTriggers();
    if (index < 0 || index >= triggers.length) return;
    currentIndex = index;
    ensureOverlay();
    updateLightbox();
    overlay.classList.add("open");
    document.body.style.overflow = "hidden";
    overlay.querySelector(".lightbox-close").focus();
  }

  function updateLightbox() {
    var trigger = triggers[currentIndex];
    if (!trigger) return;
    image.src = trigger.dataset.lightboxSrc || trigger.dataset.lightboxFull || "";
    image.alt = trigger.dataset.lightboxCaption || "";
    caption.textContent = trigger.dataset.lightboxCaption || "";
    var multiple = triggers.length > 1;
    prevButton.style.visibility = multiple ? "visible" : "hidden";
    nextButton.style.visibility = multiple ? "visible" : "hidden";
  }

  function move(delta) {
    if (triggers.length < 2) return;
    currentIndex = (currentIndex + delta + triggers.length) % triggers.length;
    updateLightbox();
  }

  function closeLightbox() {
    if (!overlay) return;
    overlay.classList.remove("open");
    document.body.style.overflow = "";
    image.removeAttribute("src");
  }

  document.addEventListener("click", function (event) {
    var trigger = event.target.closest("[data-lightbox-src]");
    if (trigger) {
      event.preventDefault();
      collectTriggers();
      openLightbox(triggers.indexOf(trigger));
    }
  });

  document.addEventListener("submit", function (event) {
    var form = event.target.closest("form");
    if (!form) return;
    Array.prototype.forEach.call(form.querySelectorAll('input[type="file"]'), function (input) {
      var picker = input._attachmentPicker;
      if (!picker || !fileArray(input).length) return;
      picker.classList.add("is-uploading");
      var status = picker.querySelector(".attachment-picker-status");
      if (status) status.textContent = "Wird beim Speichern hochgeladen...";
    });
  });

  document.addEventListener("keydown", function (event) {
    if (!overlay || !overlay.classList.contains("open")) return;
    if (event.key === "Escape") closeLightbox();
    if (event.key === "ArrowLeft") move(-1);
    if (event.key === "ArrowRight") move(1);
  });

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", enhanceFileInputs);
  } else {
    enhanceFileInputs();
  }
})();
