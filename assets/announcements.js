// Aushang/Termine dialogs and destructive-action confirmation.
// Served same-origin to satisfy the strict CSP.
var dialogTriggers = new WeakMap();

function focusFirstDialogField(dialog) {
  var field = dialog.querySelector(
    "[autofocus], input:not([type='hidden']), select, textarea, button:not([data-close-dialog])"
  );
  if (field && field.focus) field.focus();
}

function openManagedDialog(trigger) {
  var dialog = document.getElementById(trigger.dataset.dialog);
  if (!dialog || !dialog.showModal) return;
  dialogTriggers.set(dialog, trigger);
  trigger.setAttribute("aria-expanded", "true");
  dialog.showModal();
  focusFirstDialogField(dialog);
}

document.addEventListener("click", function (e) {
  var openButton = e.target.closest("[data-dialog]");
  if (openButton) {
    openManagedDialog(openButton);
  }

  var closeButton = e.target.closest("[data-close-dialog]");
  if (closeButton) {
    var openDialog = closeButton.closest("dialog");
    if (openDialog) openDialog.close();
  }
});

document.addEventListener("submit", function (e) {
  var form = e.target.closest("form[data-confirm]");
  if (form && !confirm(form.dataset.confirm)) {
    e.preventDefault();
  }
});

document.addEventListener(
  "close",
  function (e) {
    if (!e.target || e.target.nodeName !== "DIALOG") return;
    var trigger = dialogTriggers.get(e.target);
    if (!trigger) return;
    trigger.setAttribute("aria-expanded", "false");
    if (trigger.focus) trigger.focus();
  },
  true
);
