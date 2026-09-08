// Local filters retain all server-authorized cards and native forms.
(function () {
  var toolbar = document.querySelector('[data-ballot-filters]');
  if (!toolbar) return;
  var cards = Array.from(document.querySelectorAll('[data-ballot-status]'));
  var empty = document.querySelector('[data-ballot-empty]');
  toolbar.hidden = false;
  toolbar.addEventListener('click', function (event) {
    var button = event.target.closest('[data-ballot-filter]');
    if (!button) return;
    var filter = button.dataset.ballotFilter;
    toolbar.querySelectorAll('button').forEach(function (item) {
      item.setAttribute('aria-pressed', String(item === button));
    });
    cards.forEach(function (card) { card.hidden = filter !== 'all' && card.dataset.ballotStatus !== filter; });
    empty.hidden = cards.some(function (card) { return !card.hidden; });
  });
  window.addEventListener('hashchange', function () {
    var card = document.getElementById(location.hash.slice(1));
    if (card && card.matches('[data-ballot-status]') && card.hidden) {
      toolbar.querySelector('[data-ballot-filter="all"]').click();
      card.scrollIntoView();
    }
  });
})();
