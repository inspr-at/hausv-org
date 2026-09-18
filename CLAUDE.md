<!--
  Layered doctrine loader for Claude Code.

  Loads the INSPR kernel, then this repo's own notes. Other tools (Cursor,
  Aider, OpenCode, Codex CLI) read AGENTS.md directly — it is the repo delta
  and stands on its own.

  Doctrine source: github.com/inspr-at/inspr-modules, vendored as the
  ./doctrine submodule; bump with `git submodule update --remote doctrine`.

  NOTE ON BUILDS: the submodule is checkout-only and never reaches a built
  artefact. Production on csb1 pulls the CI image from GHCR; it does not
  `git archive` onto the production host. The demo path on agm1 does use
  `git archive HEAD`, which silently omits submodule contents. Neither CI
  nor the Dockerfile checks out submodules recursively. That is fine for
  docs — nothing in the build reads doctrine — but do not grow a runtime
  dependency on anything under ./doctrine.

  Studio-only private kernel lives in a local `doctrine-private/` checkout
  that is gitignored and must not be a submodule of this public tree.
-->

@./doctrine/docs/AGENTS-KERNEL.md
@./AGENTS.md

## Commands wired in this repo

This list is authoritative for what is wired here. Keep it in step with
`.claude/commands/` — a mismatch in either direction is a defect (INSPR-296):
a command listed but not wired is an advertisement without an implementation;
a command wired but not listed hides from review.

| Command | Loads |
| --- | --- |
| `/ppm` | Paimos tickets + planning. This repo's project is `HAUSV` (id 21) on the `ppm` instance — the CLI default, so no `--instance` flag needed here. |
| `/dev` | Code, tests, git workflow |
| `/ops` | Fleet, SSH, deploys |
| `/secrets` | agenix + 1Password + env pipeline |
| `/nix` | nix-darwin, Home Manager, NixOS modules |
| `/iac` | Terraform / Zitadel / Cloudflare |
| `/incident` | Leak protocol |
| `/style` | Full Markus profile |
| `/inspr` | Doctrine map |
| `/push` | Single-repo commit + push |

There is no `/pushall`; it was retired from the doctrine on 2026-08-15.
