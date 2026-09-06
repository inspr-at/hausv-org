// Einstellungen → Demodaten initialisieren (HAUSV-636): the card's link opens the
// confirmation <dialog> in place instead of leaving for the single-page fallback
// it points at. Served as a same-origin file because the portal CSP forbids
// inline scripts; without it the link simply navigates and the page confirms.
(function () {
  var trigger = document.querySelector("[data-demo-init-open]");
  var dialog = document.getElementById("demo-init-dialog");
  if (!trigger || !dialog || typeof dialog.showModal !== "function") return;
  trigger.setAttribute("aria-expanded", "false");
  trigger.addEventListener("click", function (event) {
    event.preventDefault();
    if (dialog.open) return;
    // app.js returns focus here when the dialog closes (Abbrechen, Escape).
    dialog._returnFocus = trigger;
    dialog.showModal();
    trigger.setAttribute("aria-expanded", "true");
  });
  dialog.addEventListener("close", function () {
    trigger.setAttribute("aria-expanded", "false");
  });
})();
