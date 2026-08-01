(() => {
  for (const map of document.querySelectorAll('.location-map-configured')) {
    const fail = () => map.classList.add('map-tile-failed');
    for (const tile of map.querySelectorAll('[data-map-tile]')) {
      const source = tile.dataset.mapTile;
      if (!source) {
        fail();
        continue;
      }
      const image = new Image();
      image.addEventListener('error', fail, { once: true });
      image.src = source;
    }
  }
})();
