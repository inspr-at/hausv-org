# Agent Notes

## Release And Deployment

- Every production deployment must bump `VERSION` first.
- Use patch bumps for small fixes, minor bumps for product-facing improvements, and major bumps for breaking or strategically large changes.
- Keep `docs/CHANGELOG.md` in German, newest entry first, with customer-facing release language. Prefer positive wording such as "Stabilität verbessert" over raw bug wording.
- Build and deploy with `VERSION` as `APP_VERSION`; do not deploy a changed product with an unchanged visible version.

## Secrets

- Never print, commit, or summarize secret values.
- Secret material belongs in agenix/host configuration, not in this repository.
- When checking declarative secret wiring, reference secret names and paths only.
