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
});
