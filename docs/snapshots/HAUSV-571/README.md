# HAUSV-571 templ proof

These screenshots render the real authenticated portal from `PortalPage`, not
standalone HTML:

| Proof | Role | Viewport |
| --- | --- | --- |
| `hausueberblick-dense-templ.png` | Verwalter | 1440 × 900 |
| `hausueberblick-quiet-templ.png` | Eigentümer | 1440 × 900 |
| `hausueberblick-mobile-templ.png` | Verwalter, shared phone composition | 390 × 844 |

The fixture uses the Austrian WEG copy and the same three Anliegen, two Termine,
and three Aushänge in all renders. Role-based permissions still determine which
records an actor may open; the composition itself does not change the data.

The HAUSV-521 approved static references live in the sibling directory. The
templ proof deliberately retains the subsequently approved application chrome:
the full-width hero and hairline, flush sidebar map, quiet portal switcher, and
role in the account footer.

Reproduce against a fixture portal started with `scripts/dev.sh`:

```sh
nix develop -c node scripts/snapshot/render-hausv-571.mjs \
  http://localhost:8098 docs/snapshots/HAUSV-571
```
