// Benutzer & Rechte: open the per-row edit dialog and confirm deletes.
// Served as a same-origin file so it satisfies the strict CSP (default-src 'self').
var dialogTriggers = new WeakMap();

function focusFirstDialogField(dialog) {
  var field = dialog.querySelector(
    "[autofocus], input:not([type='hidden']), select, textarea, button:not([aria-label='Schließen'])"
  );
  if (field && field.focus) field.focus();
}

function openEditDialog(trigger) {
  var dialog = document.getElementById("edit-" + trigger.dataset.edit);
  if (!dialog || !dialog.showModal) return;
  dialogTriggers.set(dialog, trigger);
  trigger.setAttribute("aria-expanded", "true");
  dialog.showModal();
  focusFirstDialogField(dialog);
}

document.addEventListener("click", function (e) {
  var b = e.target.closest(".users .row-edit");
  if (b) {
    openEditDialog(b);
  }
});
document.addEventListener("submit", function (e) {
  if (e.target.closest(".users .dlg-delete") && !confirm("Diesen Zugang wirklich löschen?")) {
    e.preventDefault();
  }
});

function applyRolePreset(select) {
  var option = select.options[select.selectedIndex];
  if (!option) return;
  var form = select.closest("form");
  if (!form) return;
  var preset = (option.dataset.presetPermissions || "")
    .split(",")
    .map(function (item) {
      return item.trim();
    })
    .filter(Boolean);
  form.querySelectorAll("[data-permission]").forEach(function (input) {
    input.checked = preset.indexOf(input.dataset.permission) !== -1;
  });
  var label = form.querySelector("[data-preset-label]");
  if (label) label.textContent = option.dataset.presetLabel || "Standardzugriff";
}

document.addEventListener("change", function (e) {
  if (e.target.matches(".users .f-role")) {
    applyRolePreset(e.target);
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
