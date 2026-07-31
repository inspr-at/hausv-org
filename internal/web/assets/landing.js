document.addEventListener("DOMContentLoaded", function () {
  document.querySelectorAll("[data-mail-local][data-mail-domain]").forEach(function (link) {
    var address = link.dataset.mailLocal + "@" + link.dataset.mailDomain;
    var subject = link.dataset.mailSubject ? "?subject=" + encodeURIComponent(link.dataset.mailSubject) : "";
    link.href = "mailto:" + address + subject;
    if (link.dataset.mailReveal !== "false") {
      link.textContent = address;
    }
    link.setAttribute("aria-label", "E-Mail an " + address);
  });

  // The mobile menu is a CSS-only checkbox, so following a link would leave it
  // open over the section it just jumped to. Close it on selection, and on
  // Escape while it has focus.
  var navToggle = document.getElementById("landing-nav-toggle");
  if (navToggle) {
    document.querySelectorAll(".landing-links a").forEach(function (link) {
      link.addEventListener("click", function () {
        navToggle.checked = false;
      });
    });
    document.addEventListener("keydown", function (event) {
      if (event.key === "Escape" && navToggle.checked) {
        navToggle.checked = false;
      }
    });
  }
});
