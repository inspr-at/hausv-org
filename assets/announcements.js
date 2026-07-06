// Aushang dialogs and destructive-action confirmation.
// Served same-origin to satisfy the strict CSP.
document.addEventListener("click", function (e) {
  var openButton = e.target.closest("[data-dialog]");
  if (openButton) {
    var dialog = document.getElementById(openButton.dataset.dialog);
    if (dialog && dialog.showModal) dialog.showModal();
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
