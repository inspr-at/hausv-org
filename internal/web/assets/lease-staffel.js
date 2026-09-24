(() => {
  const editor = document.querySelector('[data-staffel-editor]');
  if (!editor) return;
  const form = editor.closest('form');
  const type = form.querySelector('[name="clause_type"]');
  const rows = editor.querySelector('[data-staffel-rows]');
  const add = editor.querySelector('[data-staffel-add]');
  function update() {
    const active = type.value === 'staffel';
    editor.hidden = !active;
    for (const field of form.querySelectorAll('[data-clause-index]')) {
      field.hidden = active;
      for (const input of field.querySelectorAll('input, select')) input.disabled = active;
    }
    for (const input of rows.querySelectorAll('input, select')) {
      input.disabled = !active;
      input.required = active;
    }
    for (const button of rows.querySelectorAll('[data-staffel-remove]')) {
      button.disabled = rows.children.length <= 1;
    }
    add.disabled = rows.children.length >= 120;
  }
  type.addEventListener('change', update);
  add.addEventListener('click', () => {
    if (rows.children.length >= 120) return;
    rows.append(editor.querySelector('template').content.cloneNode(true));
    update();
    rows.lastElementChild.querySelector('input').focus();
  });
  rows.addEventListener('click', (event) => {
    const button = event.target.closest('[data-staffel-remove]');
    if (!button || rows.children.length <= 1) return;
    button.closest('[data-staffel-row]').remove();
    update();
    add.focus();
  });
  update();
})();
