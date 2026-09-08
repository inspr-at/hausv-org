(function () {
  function ready() {
    const page = document.querySelector('[data-inbox-shortcuts]');
    if (!page) return;

    function suggestionPoll(event) {
      const state = event.detail?.requestConfig?.elt || event.detail?.elt;
      return state?.matches?.('#vorschlag[hx-get]') && state.isConnected ? state : null;
    }

    function failSuggestionPoll(state) {
      if (state.getAttribute('hx-trigger') === 'inbox:suggestion-retry') return;
      const failed = state.cloneNode(true);
      failed.setAttribute('hx-trigger', 'inbox:suggestion-retry');
      failed.setAttribute('role', 'status');
      failed.setAttribute('aria-live', 'polite');
      failed.classList.remove('htmx-request');
      const content = failed.querySelector('[data-suggestion-poll-error]').content.cloneNode(true);
      failed.querySelector('.suggest-state-main').replaceWith(content);
      // Changing hx-trigger alone leaves htmx's existing timer alive. A swap
      // runs htmx's element cleanup (including the timer) and processes the new
      // click-only reader, preserving the cancel form and the case workflow.
      htmx.swap(state, failed.outerHTML, { swapStyle: 'outerHTML', settleDelay: 0 });
    }

    document.addEventListener('htmx:beforeSwap', function (event) {
      const state = suggestionPoll(event);
      if (!state) return;
      const detail = event.detail;
      const response = new DOMParser().parseFromString(detail.serverResponse || '', 'text/html');
      if (detail.xhr.status < 200 || detail.xhr.status >= 300 || !response.querySelector('#vorschlag')) {
        // Includes a followed login/wrong-tenant redirect and an empty 204.
        // Cancel before htmx processes either hx-select or out-of-band swaps.
        detail.shouldSwap = false;
        event.preventDefault();
        failSuggestionPoll(state);
      }
    });
    for (const name of ['htmx:responseError', 'htmx:sendError', 'htmx:timeout']) {
      document.addEventListener(name, function (event) {
        const state = suggestionPoll(event);
        if (state) failSuggestionPoll(state);
      });
    }
    page.addEventListener('click', function (event) {
      const retry = event.target.closest('[data-suggestion-retry]');
      if (!retry) return;
      const state = retry.closest('#vorschlag');
      if (state) htmx.trigger(state, 'inbox:suggestion-retry');
    });

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
