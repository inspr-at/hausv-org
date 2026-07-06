// Benutzer & Rechte: open the per-row edit dialog and confirm deletes.
// Served as a same-origin file so it satisfies the strict CSP (default-src 'self').
document.addEventListener("click", function (e) {
  var b = e.target.closest(".users .row-edit");
  if (b) {
    var d = document.getElementById("edit-" + b.dataset.edit);
    if (d && d.showModal) d.showModal();
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
