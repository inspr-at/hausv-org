# Agent Notes

## INSPR And PPM

- The default PPM project is `HAUSV` (project id `21`) on `pm.barta.cm`.
- Keep backlog and ticket state in PPM; do not create local backlog or TODO files.
- PPM Knowledge is canonical for durable architecture, design rationale,
  positioning, playbooks, research, delivery/approval status, and cross-project
  context.
- Keep versioned, code-adjacent specifications in this repository when they
  describe concrete formats, routes, fixtures, or operational commands. Link
  them from PPM Knowledge instead of copying status or backlog into the repo.
- Treat PPM as read-only unless the user explicitly asks for project-management
  changes. Mark tickets done only after every acceptance criterion is met.

## Release And Deployment

- Every production deployment must bump `VERSION` first.
- Use patch bumps for small fixes, minor bumps for product-facing improvements, and major bumps for breaking or strategically large changes.
- Keep `docs/CHANGELOG.md` in German, newest entry first, with customer-facing release language. Prefer positive wording such as "Stabilität verbessert" over raw bug wording.
- Add a matching entry to `Notes()` in `internal/version/version.go`. Its newest entry must equal `VERSION`; a test enforces this, so bump the version *before* the final test run — it has been forgotten once already.
- Build and deploy with `VERSION` as `APP_VERSION`; do not deploy a changed product with an unchanged visible version.
- Deploy with `scripts/deploy.fish` (`--dry-run` to check first). It ships `HEAD` via `git archive`, builds the image on the host with `VERSION`/commit baked in, recreates the compose service, verifies the live version, and prints a rollback command. It refuses to run on a dirty tree, when `VERSION` is already live, when `HEAD` differs from `origin/main`, or when that exact commit has no completed, green CI run on Blacksmith runners. Details in `docs/csb1-deploy.md`.
- Deployment is manual: CI builds the image to prove the Dockerfile works but never pushes it.

## Secrets

- Never print, commit, or summarize secret values.
- Secret material belongs in agenix/host configuration, not in this repository.
- When checking declarative secret wiring, reference secret names and paths only.
