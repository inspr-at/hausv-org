// Benutzer & Rechte: open the per-row edit dialog.
// Served as a same-origin file so it satisfies the strict CSP (default-src 'self').
var dialogTriggers = new WeakMap();

function focusFirstDialogField(dialog) {
  var field = dialog.querySelector(
    "[autofocus], input:not([type='hidden']), select, textarea, button:not([aria-label='Schließen'])"
  );
  if (field && field.focus) field.focus();
}

function openEditDialog(trigger) {
  var dialog = document.getElementById("edit-" + (trigger.dataset.edit || trigger.dataset.accessEdit));
  if (!dialog || !dialog.showModal) return;
  dialogTriggers.set(dialog, trigger);
  trigger.setAttribute("aria-expanded", "true");
  dialog.showModal();
  focusFirstDialogField(dialog);
}

document.addEventListener("click", function (e) {
  var b = e.target.closest(".users .row-edit, .users [data-access-edit]");
  if (b) {
    openEditDialog(b);
  }
});
function applyRolePreset(select) {
  var option = select.options[select.selectedIndex];
  if (!option) return;
  var form = select.closest("form");
  if (!form) return;
  var preset = (option.dataset.presetPermissions || "")
    .split(",")
    .map(function (item) {
      return item.trim();
    })
    .filter(Boolean);
  form.querySelectorAll("[data-permission]").forEach(function (input) {
    input.checked = preset.indexOf(input.dataset.permission) !== -1;
  });
  var label = form.querySelector("[data-preset-label]");
  if (label) label.textContent = option.dataset.presetLabel || "Standardzugriff";
}

document.addEventListener("change", function (e) {
  if (e.target.matches(".users .f-role")) {
    applyRolePreset(e.target);
  }
});

document.addEventListener(
  "close",
  function (e) {
    if (!e.target || e.target.nodeName !== "DIALOG") return;
    var trigger = dialogTriggers.get(e.target);
    if (!trigger) return;
    trigger.setAttribute("aria-expanded", "false");
    if (trigger.focus) trigger.focus();
  },
  true
);

// Selection and filters only switch the already authorized server-rendered views.
(function () {
  var root = document.querySelector('.users');
  if (!root) return;
  var rows = Array.from(root.querySelectorAll('[data-user-row]'));
  var panels = Array.from(root.querySelectorAll('[data-user-access]'));
  var search = root.querySelector('[data-user-search]');
  var role = root.querySelector('[data-user-role]');
  var selected = '';
  var invite = root.querySelector('#invite');
  root.classList.add('users-enhanced');
  root.querySelector('[data-user-filters]').hidden = false;

  function selectPerson(email) {
    selected = email;
    rows.forEach(function (row) {
      var active = row.dataset.userRow === email;
      row.classList.toggle('is-selected', active);
      var link = row.querySelector('[data-user-select]');
      if (active) link.setAttribute('aria-current', 'true');
      else link.removeAttribute('aria-current');
    });
    panels.forEach(function (panel) { panel.hidden = panel.dataset.userAccess !== email; });
  }

  function filterPeople() {
    var query = search.value.trim().toLocaleLowerCase('de-AT');
    rows.forEach(function (row) {
      row.hidden = !row.dataset.userSearchText.toLocaleLowerCase('de-AT').includes(query) ||
        (role.value !== '' && 'filter:' + row.dataset.userRoleValue !== role.value);
    });
    var visible = rows.filter(function (row) { return !row.hidden; });
    if (!visible.some(function (row) { return row.dataset.userRow === selected; })) {
      selectPerson(visible.length ? visible[0].dataset.userRow : '');
    }
    var empty = root.querySelector('[data-user-empty]');
    if (empty) empty.hidden = visible.length !== 0;
    var count = root.querySelector('[data-user-count]');
    if (count) count.textContent = visible.length + ' von ' + rows.length + ' Personen';
  }

  function followHash() {
    if (location.hash === '#invite') {
      invite.open = true;
      invite.scrollIntoView({ block: 'start' });
      return;
    }
    var id;
    try { id = decodeURIComponent(location.hash.slice(1)); } catch (_) { return; }
    var panel = document.getElementById(id);
    if (panel && panel.matches('[data-user-access]')) {
      search.value = ''; role.value = ''; filterPeople();
      selectPerson(panel.dataset.userAccess);
      panel.scrollIntoView({ block: 'nearest' });
    }
  }

  root.addEventListener('click', function (event) {
    var link = event.target.closest('[data-user-select]');
    if (link) {
      event.preventDefault();
      selectPerson(link.dataset.userSelect);
      if (window.matchMedia('(max-width:1000px)').matches) {
        var panel = panels.find(function (item) { return !item.hidden; });
        if (panel) { panel.focus({ preventScroll: true }); panel.scrollIntoView({ block: 'start' }); }
      }
    }
    if (event.target.closest('[data-user-invite]')) {
      invite.open = true;
      invite.querySelector('input[type="email"]').focus();
    }
  });
  search.addEventListener('input', filterPeople);
  role.addEventListener('change', filterPeople);
  window.addEventListener('hashchange', followHash);
  selectPerson(rows.length ? rows[0].dataset.userRow : '');
  followHash();
})();
