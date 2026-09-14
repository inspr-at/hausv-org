(() => {
  const banner = document.querySelector('[data-support-view-banner]');
  if (!banner) return;
  // The server rejects every write. Disable the same controls in the page so
  // a support reader does not accidentally start an editing flow.
  for (const form of document.querySelectorAll('form[method="post"]')) {
    const path = new URL(form.action, location.href).pathname;
    if (path.endsWith('/app/support-view/end') || path.endsWith('/auth/logout')) continue;
    for (const control of form.querySelectorAll('button, input:not([type="hidden"]), select, textarea')) control.disabled = true;
  }
})();
