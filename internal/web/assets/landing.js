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

  // The disclosure is a real button so keyboard and assistive-technology users
  // get the same open/closed state as pointer users. The noscript rule keeps
  // every destination visible when JavaScript is unavailable.
  var nav = document.querySelector(".landing-nav");
  var navToggle = document.querySelector("[data-landing-menu-toggle]");
  if (nav && navToggle) {
    var closeMenu = function (restoreFocus) {
      nav.removeAttribute("data-menu-open");
      navToggle.setAttribute("aria-expanded", "false");
      if (restoreFocus) navToggle.focus();
    };
    navToggle.addEventListener("click", function () {
      var open = navToggle.getAttribute("aria-expanded") !== "true";
      if (open) {
        nav.setAttribute("data-menu-open", "true");
      } else {
        nav.removeAttribute("data-menu-open");
      }
      navToggle.setAttribute("aria-expanded", String(open));
    });
    document.querySelectorAll(".landing-links a").forEach(function (link) {
      link.addEventListener("click", function () {
        closeMenu(false);
      });
    });
    document.addEventListener("keydown", function (event) {
      if (event.key === "Escape" && navToggle.getAttribute("aria-expanded") === "true") {
        closeMenu(true);
      }
    });
    document.addEventListener("click", function (event) {
      if (navToggle.getAttribute("aria-expanded") === "true" && !nav.contains(event.target)) {
        closeMenu(false);
      }
    });
    var desktop = window.matchMedia("(min-width: 901px)");
    desktop.addEventListener("change", function (event) {
      if (event.matches) closeMenu(false);
    });
  }

  // Reveal the flat white nav mark and the dark top veil once the hero's
  // rotating mark has scrolled out of view. CSS keys off body[data-hero-mark];
  // the 3D module itself never touches scroll state.
  var heroMark = document.querySelector("[data-hausv-mark-3d]");
  if (heroMark && "IntersectionObserver" in window) {
    new IntersectionObserver(function (entries) {
      document.body.setAttribute("data-hero-mark", entries[entries.length - 1].isIntersecting ? "in" : "out");
    }).observe(heroMark);
  }
});
