# Agent Notes

## INSPR And PPM

- The default PPM project is `HAUSV` (project id `21`) in the configured PPM instance.
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
- Deployment happens by itself. CI builds the image on every push and, on `main`, pushes it to GHCR as `:<sha>`, `:<short sha>` and `:release-<version>-<short sha>`. After that green CI run, `.github/workflows/deploy.yml` runs on the self-hosted runner on csb1, pulls exactly that image and swaps the production container; when `internal/db/migrations` changed it takes the pre-deploy snapshot first. So a merge to `main` with a bumped `VERSION` releases to production without anyone typing a command.
- `scripts/deploy.sh` (`--dry-run` to check first) is the attended path for verification or break-glass, run from a Mac against the configured host. It pulls the same CI image from GHCR, swaps the compose service, verifies the live version and prints a rollback command. It refuses to run on a dirty tree, when `VERSION` is already live, when `HEAD` differs from `origin/main`, or when that exact commit has no completed, green CI run on Blacksmith runners. Details in `docs/production-deploy.md`.
- A merge to `main` **without** a `VERSION` bump therefore produces a red `Deploy` run with `release refused: VERSION <x> is already live`. That is the fail-closed guard doing its job, not a defect: production keeps running the previous image. Bump `VERSION` when you want the merge released, and expect that red run when you deliberately do not.

## Secrets

- Never print, commit, or summarize secret values.
- Secret material belongs in agenix/host configuration, not in this repository.
- When checking declarative secret wiring, reference secret names and paths only.
