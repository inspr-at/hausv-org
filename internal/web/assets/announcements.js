// Aushang/Termine dialogs.
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

// Bulk controls enhance the native, individually keyboard-operable disclosures.
(function () {
  var toolbar = document.querySelector('[data-announcement-bulk]');
  if (!toolbar) return;
  var entries = Array.from(document.querySelectorAll('.announcement-card > .announcement-body'));
  if (!entries.length) return;
  toolbar.hidden = false;
  toolbar.addEventListener('click', function (event) {
    var button = event.target.closest('[data-announcements-expand]');
    if (!button) return;
    entries.forEach(function (entry) { entry.open = button.dataset.announcementsExpand === 'true'; });
  });
  var legend = document.querySelector('.announcement-legend');
  document.addEventListener('keydown', function (event) {
    if (event.key === 'Escape' && legend && legend.open) {
      legend.open = false;
      legend.querySelector('summary').focus();
    }
  });
  document.addEventListener('click', function (event) {
    if (legend && !legend.contains(event.target)) legend.open = false;
  });
})();
