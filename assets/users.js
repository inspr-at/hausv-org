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
