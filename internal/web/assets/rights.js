// Progressive enhancement: each switch remains an ordinary POST form.
(() => {
  const forms = [...document.querySelectorAll('[data-rights-switch]')];
  forms.forEach(form => {
    const input = form.querySelector('[name="allowed"]');
    const save = form.querySelector('[data-rights-save]');
    save.hidden = true;
    input.addEventListener('change', async () => {
      const group = forms.filter(other => other.dataset.rightsFamily === form.dataset.rightsFamily && other.dataset.rightsCapability === form.dataset.rightsCapability);
      const previous = group.map(other => other.querySelector('[name="allowed"]').defaultChecked);
      const desired = input.checked;
      const body = new URLSearchParams(new FormData(form));
      group.forEach(other => {
        const checkbox = other.querySelector('[name="allowed"]');
        checkbox.checked = desired;
        checkbox.disabled = true;
        other.querySelector('[data-rights-status]').textContent = 'Wird gespeichert …';
      });
      try {
        const response = await fetch(form.action, {method: 'POST', body, headers: {'Accept': 'application/json'}, credentials: 'same-origin'});
        if (!response.ok) throw new Error('save failed');
        const result = await response.json();
        if (!result.saved) throw new Error('save failed');
        group.forEach(other => {
          other.querySelector('[name="allowed"]').defaultChecked = desired;
          other.querySelector('[data-rights-status]').textContent = 'Rechte gespeichert.';
          let changed = other.querySelector('.rights-changed');
          if (result.changed && !changed) {
            changed = document.createElement('small');
            changed.className = 'rights-changed';
            changed.textContent = 'Abweichend vom Standard';
            other.querySelector('[data-rights-status]').before(changed);
          } else if (!result.changed && changed) changed.remove();
        });
      } catch (_) {
        group.forEach((other, index) => {
          other.querySelector('[name="allowed"]').checked = previous[index];
          other.querySelector('[data-rights-status]').textContent = 'Nicht gespeichert. Bitte erneut versuchen oder die Seite neu laden.';
          other.querySelector('[data-rights-save]').hidden = false;
        });
      } finally {
        group.forEach(other => { other.querySelector('[name="allowed"]').disabled = false; });
      }
    });
  });
})();
