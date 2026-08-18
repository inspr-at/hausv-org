// The interactive contract of one rendered page.
//
// Snapshot the interactive DOM contract so evidence captures lost scripts, body
// attributes, POST targets, confirmations, required fields and enhancement hooks.
// Reachable href checks alone cannot see a lost <script>, a dropped body attribute, a form that no longer
// asks before deleting, or a DOM hook a script still looks for. Four real
// regressions went through that gap in Phase 3 — two of them security-relevant.
//
// Extract the things a user can act on in DOM terms rather than by regex.
//
// Runs inside page.evaluate(), so it must be a self-contained function body with no
// imports and no closure over module scope.
export const EXTRACT_CONTRACT = () => {
  const norm = (value) => String(value || '')
    .replace(/\?v=[^"'&\s]*/g, '')
    .replace(/^https?:\/\/[^/]+/, '')
    .replace(/token=[A-Za-z0-9_\-.]+/g, 'token=TOKEN')
    .replace(/\/calendar\/[^/"'\s]+\.ics/g, '/calendar/FEEDTOKEN.ics');

  const uniq = (values) => Array.from(new Set(values)).sort();

  const forms = Array.from(document.querySelectorAll('form'));
  const formKey = (form) => `${(form.getAttribute('method') || 'get').toLowerCase()} ${norm(form.getAttribute('action') || '')}`;

  return {
    // A missing script silently disables every enhancement it carries. app.js
    // alone owns bfcache invalidation, data-confirm and the double-submit lock.
    scripts: uniq(Array.from(document.querySelectorAll('script[src]')).map((s) => norm(s.getAttribute('src')))),

    // app.js keys its post-logout bfcache reload on data-authenticated-app.
    bodyAttrs: uniq(Array.from(document.body.attributes).map((a) => a.name)),

    // A POST endpoint the page no longer offers is a lost action, and it is not
    // an href, so the reachability check cannot see it.
    forms: uniq(forms.map(formKey)),

    // data-confirm has no native browser behaviour. A form that carries it on one
    // rendering and not the other deletes without asking on the other.
    confirms: uniq(forms.filter((f) => f.hasAttribute('data-confirm')).map(formKey)),

    // Field names are the server contract; ids and labels move freely in a redesign.
    fields: uniq(forms.flatMap((form) => Array.from(form.querySelectorAll('input,select,textarea'))
      .filter((el) => el.name)
      .map((el) => `${formKey(form)} | ${el.name}`))),

    required: uniq(forms.flatMap((form) => Array.from(form.querySelectorAll('input,select,textarea'))
      .filter((el) => el.name && el.hasAttribute('required'))
      .map((el) => `${formKey(form)} | ${el.name}`))),

    // Progressive enhancement is a handshake: the script looks for a hook, the
    // markup provides it. Dropping the hook leaves the script loaded and inert.
    //
    // <body> is excluded because bodyAttrs above already owns it; counting it
    // twice inflates a single defect into two and makes the totals lie.
    hooks: uniq(Array.from(document.querySelectorAll('*')).filter((el) => el !== document.body)
      .flatMap((el) => Array.from(el.attributes)
        .filter((a) => a.name.startsWith('data-'))
        .map((a) => a.name))),
  };
};
