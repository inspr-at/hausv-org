// Keep the native disclosure usable without JavaScript.
(function () {
  var panel = document.getElementById("contact-add");
  if (!panel) return;
  var cancel = panel.querySelector("[data-contact-cancel]");
  if (cancel) {
    cancel.hidden = false;
    cancel.addEventListener("click", function () {
      panel.open = false;
      panel.querySelector("summary").focus();
    });
  }
  document.querySelectorAll('a[href="#contact-add"]').forEach(function (link) {
    link.addEventListener("click", function () {
      panel.open = true;
      panel.querySelector("select, input:not([type=hidden])").focus();
    });
  });
  if (window.location.hash === "#contact-add") panel.open = true;
})();
