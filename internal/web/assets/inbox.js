(function () {
  function ready() {
    const page = document.querySelector('[data-inbox-shortcuts]');
    if (!page) return;
    const queue = page.querySelector('.queue');
    const selected = page.querySelector('.queue-row.selected');
    if (queue && selected && queue.clientHeight) {
      const toolbar = queue.querySelector('.queue-toolbar');
      const top = selected.getBoundingClientRect().top - queue.getBoundingClientRect().top;
      const inset = toolbar ? toolbar.offsetHeight : 0;
      if (top < inset || top + selected.offsetHeight > queue.clientHeight) {
        queue.scrollTop += top - inset;
      }
    }
    document.addEventListener('keydown', function (event) {
      if (event.defaultPrevented || event.altKey || event.ctrlKey || event.metaKey || event.repeat) return;
      if (event.target.closest('input,textarea,select,button,a,summary,[contenteditable="true"]')) return;
      const selectors = {
        j: '[data-nav="next"]', k: '[data-nav="prev"]',
        a: '[data-shortcut="approve"]', e: '[data-shortcut="edit"]',
        m: '[data-shortcut="manual"]'
      };
      const selector = selectors[event.key.toLowerCase()];
      const control = selector && page.querySelector(selector);
      if (!control) return;
      event.preventDefault();
      if (control.tagName === 'A') control.click();
      else if (control.form) control.form.requestSubmit(control);
    });
  }
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', ready);
  else ready();
})();
