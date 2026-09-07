// Profilbild: Datei wählen, quadratisch zuschneiden, als 512×512-JPEG hochladen.
//
// Served as a same-origin file so it satisfies the strict CSP (default-src
// 'self'); the portal forbids inline scripts.
//
// Progressive enhancement, deliberately: without this file the form next to the
// dialog still uploads the chosen file and the server centre-crops it. This
// script only replaces the file the form carries with the cropped one, so the
// upload path — and its CSRF protection, which is the same-origin form POST —
// stays exactly the same.
(function () {
  "use strict";

  var MAX_BYTES = 8 * 1024 * 1024;
  var OUTPUT = 512;
  var QUALITY = 0.86;

  var state = {
    form: null,
    input: null,
    image: null,
    scale: 1,
    minScale: 1,
    offsetX: 0,
    offsetY: 0,
    dragging: false,
    lastX: 0,
    lastY: 0,
    objectURL: "",
  };

  function el(selector, root) {
    return (root || document).querySelector(selector);
  }

  function showError(node, message) {
    if (!node) return;
    node.textContent = message;
    node.hidden = false;
  }

  function clearError(node) {
    if (!node) return;
    node.textContent = "";
    node.hidden = true;
  }

  function formatMegabytes(bytes) {
    return (bytes / (1024 * 1024)).toLocaleString("de-AT", { maximumFractionDigits: 1 }) + " MB";
  }

  function releaseObjectURL() {
    if (state.objectURL) {
      URL.revokeObjectURL(state.objectURL);
      state.objectURL = "";
    }
  }

  function stageSize(canvas) {
    return canvas.width;
  }

  // fitScale is the smallest zoom at which the picture still covers the whole
  // square. Panning is clamped to it, so the crop can never contain a gap.
  function fitScale(canvas) {
    var side = stageSize(canvas);
    return Math.max(side / state.image.naturalWidth, side / state.image.naturalHeight);
  }

  function clampOffsets(canvas) {
    var side = stageSize(canvas);
    var width = state.image.naturalWidth * state.scale;
    var height = state.image.naturalHeight * state.scale;
    var maxX = Math.max(0, (width - side) / 2);
    var maxY = Math.max(0, (height - side) / 2);
    state.offsetX = Math.min(maxX, Math.max(-maxX, state.offsetX));
    state.offsetY = Math.min(maxY, Math.max(-maxY, state.offsetY));
  }

  function draw(canvas) {
    if (!canvas || !state.image) return;
    var context = canvas.getContext("2d");
    if (!context) return;
    var side = stageSize(canvas);
    clampOffsets(canvas);
    var width = state.image.naturalWidth * state.scale;
    var height = state.image.naturalHeight * state.scale;
    context.clearRect(0, 0, side, side);
    context.fillStyle = "#f7f3ea";
    context.fillRect(0, 0, side, side);
    context.drawImage(
      state.image,
      (side - width) / 2 + state.offsetX,
      (side - height) / 2 + state.offsetY,
      width,
      height
    );
  }

  function setZoomFromSlider(canvas, slider) {
    if (!slider) return;
    var factor = Number(slider.value) / Number(slider.min || 100);
    if (!isFinite(factor) || factor <= 0) factor = 1;
    state.scale = state.minScale * factor;
    draw(canvas);
  }

  function syncSlider(slider) {
    if (!slider) return;
    var base = Number(slider.min || 100);
    slider.value = String(Math.round((state.scale / state.minScale) * base));
  }

  function openDialog(dialog, canvas, slider) {
    state.minScale = fitScale(canvas);
    state.scale = state.minScale;
    state.offsetX = 0;
    state.offsetY = 0;
    syncSlider(slider);
    draw(canvas);
    if (dialog.showModal) {
      dialog.showModal();
    } else {
      dialog.setAttribute("open", "open");
    }
  }

  function closeDialog(dialog) {
    if (dialog.close) {
      dialog.close();
    } else {
      dialog.removeAttribute("open");
    }
    releaseObjectURL();
    state.image = null;
  }

  // renderOutput paints the chosen crop onto a 512×512 canvas. The dialog canvas
  // is only the viewport; the mapping from it to the output is one factor.
  function renderOutput(canvas) {
    var out = document.createElement("canvas");
    out.width = OUTPUT;
    out.height = OUTPUT;
    var context = out.getContext("2d");
    if (!context) return null;
    var side = stageSize(canvas);
    var factor = OUTPUT / side;
    var width = state.image.naturalWidth * state.scale * factor;
    var height = state.image.naturalHeight * state.scale * factor;
    context.fillStyle = "#ffffff";
    context.fillRect(0, 0, OUTPUT, OUTPUT);
    context.drawImage(
      state.image,
      (OUTPUT - width) / 2 + state.offsetX * factor,
      (OUTPUT - height) / 2 + state.offsetY * factor,
      width,
      height
    );
    return out;
  }

  function submitCropped(blob, fileError, dialogError, dialog) {
    if (!blob) {
      showError(dialogError, "Der Ausschnitt konnte nicht erzeugt werden. Bitte erneut versuchen.");
      return;
    }
    var file;
    try {
      file = new File([blob], "profilbild.jpg", { type: "image/jpeg" });
      var transfer = new DataTransfer();
      transfer.items.add(file);
      state.input.files = transfer.files;
    } catch (err) {
      showError(dialogError, "Dieser Browser kann den Zuschnitt nicht übernehmen. Bitte ein bereits quadratisches Bild hochladen.");
      return;
    }
    clearError(fileError);
    closeDialog(dialog);
    state.form.submit();
  }

  function start() {
    var card = el("[data-templ-profile-picture]");
    var dialog = el("[data-profile-picture-dialog]");
    if (!card || !dialog) return;
    var form = el("[data-profile-picture-form]", card);
    var input = el("[data-profile-picture-input]", card);
    var fileError = el("[data-profile-picture-error]", card);
    var dialogError = el("[data-profile-picture-dialog-error]", dialog);
    var canvas = el("[data-profile-picture-canvas]", dialog);
    var slider = el("[data-profile-picture-zoom]", dialog);
    var stage = el(".crop-stage", dialog);
    var saveButton = el("[data-profile-picture-save]", dialog);
    var cancelButton = el("[data-profile-picture-cancel]", dialog);
    if (!form || !input || !canvas || !stage) return;

    state.form = form;
    state.input = input;
    document.body.classList.add("profile-picture-enhanced");

    input.addEventListener("change", function () {
      clearError(fileError);
      clearError(dialogError);
      var file = (input.files || [])[0];
      if (!file) return;
      if (String(file.type || "").indexOf("image/") !== 0) {
        input.value = "";
        showError(fileError, "Bitte ein Bild wählen (JPEG, PNG oder WebP).");
        return;
      }
      if (file.size > MAX_BYTES) {
        input.value = "";
        showError(fileError, "Das Bild ist zu gross (" + formatMegabytes(file.size) + "). Erlaubt sind bis zu 8 MB.");
        return;
      }
      releaseObjectURL();
      state.objectURL = URL.createObjectURL(file);
      var image = new Image();
      image.onload = function () {
        state.image = image;
        openDialog(dialog, canvas, slider);
      };
      image.onerror = function () {
        releaseObjectURL();
        input.value = "";
        showError(fileError, "Diese Datei konnte nicht als Bild geöffnet werden.");
      };
      image.src = state.objectURL;
    });

    if (slider) {
      slider.addEventListener("input", function () {
        setZoomFromSlider(canvas, slider);
      });
    }

    stage.addEventListener("pointerdown", function (event) {
      if (!state.image) return;
      state.dragging = true;
      state.lastX = event.clientX;
      state.lastY = event.clientY;
      if (stage.setPointerCapture) stage.setPointerCapture(event.pointerId);
    });
    stage.addEventListener("pointermove", function (event) {
      if (!state.dragging || !state.image) return;
      state.offsetX += event.clientX - state.lastX;
      state.offsetY += event.clientY - state.lastY;
      state.lastX = event.clientX;
      state.lastY = event.clientY;
      draw(canvas);
    });
    ["pointerup", "pointercancel", "pointerleave"].forEach(function (name) {
      stage.addEventListener(name, function () {
        state.dragging = false;
      });
    });
    stage.addEventListener(
      "wheel",
      function (event) {
        if (!state.image) return;
        event.preventDefault();
        var step = event.deltaY < 0 ? 1.06 : 1 / 1.06;
        state.scale = Math.min(state.minScale * 3.2, Math.max(state.minScale, state.scale * step));
        syncSlider(slider);
        draw(canvas);
      },
      { passive: false }
    );

    if (cancelButton) {
      cancelButton.addEventListener("click", function () {
        input.value = "";
        closeDialog(dialog);
      });
    }
    dialog.addEventListener("cancel", function () {
      input.value = "";
      releaseObjectURL();
      state.image = null;
    });

    if (saveButton) {
      saveButton.addEventListener("click", function () {
        if (!state.image) return;
        clearError(dialogError);
        var out = renderOutput(canvas);
        if (!out) {
          showError(dialogError, "Der Ausschnitt konnte nicht erzeugt werden. Bitte erneut versuchen.");
          return;
        }
        saveButton.disabled = true;
        out.toBlob(
          function (blob) {
            saveButton.disabled = false;
            submitCropped(blob, fileError, dialogError, dialog);
          },
          "image/jpeg",
          QUALITY
        );
      });
    }
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", start);
  } else {
    start();
  }
})();
