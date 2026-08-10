# Changelog

All notable changes to Snagify are documented here. This project adheres to
semantic versioning.

## v0.4.0

LAN compare, doctor, and fix workflow.

- `snagify share current --lan` — capture the current machine as a sanitized
  baseline and advertise it on the LAN/VPN via mDNS (_snagify._tcp). Supports
  pairing-code mode (default) and open mode (--open).
- `snagify compare --lan` — discover a LAN share via mDNS, download the
  sanitized baseline over TLS-pinned HTTPS (certificate fingerprint from mDNS
  TXT), capture local snapshot, compare locally. Teammate snapshot never leaves
  their machine.
- `snagify doctor` — run check logic and emit a structured fix plan (safe /
  guided / unfixable). Never mutates files.
- `snagify fix --safe` — apply only safe fixes: create .env from .env.example,
  append missing keys as blank placeholders, pull required Docker images (with
  --yes). Supports --dry-run, --yes. Never invents secret values, never
  installs runtimes.
- `snagify fix --guided` — print version-manager commands for runtime
  mismatches (nvm/sdkman/pyenv/brew). Prints only, does not execute.
- GitHub SEO: H1 updated, docs/ guides added covering works-on-my-machine,
  environment-drift, compare-dev-environments, local-vs-ci, dotenv-missing-keys.

## v0.3.2

Reduce `.env.example` noise from `snagify init` and `snagify check`.

- `snagify init` now classifies `.env.example` keys as `required`,
  `recommended`, or `optional` using conservative heuristics:
  - Blank keys with sensitive names (API_KEY, TOKEN, SECRET, DATABASE_URL, etc.)
    → `required`.
  - Keys with concrete defaults, or tuning names (MAX, RETRY, TIMEOUT, etc.)
    → `recommended`.
  - Keys belonging to disabled feature groups (e.g. `FALLBACK_1_ENABLED=false`
    causes all `FALLBACK_1_*` keys) → `optional`.
  - Comment hints in `.env.example` (`# required`, `# optional`, etc.)
    override heuristics.
- `snagify check` maps the three classes to severities:
  - `required` missing → critical blocker.
  - `recommended` missing → warning.
  - `optional` missing → silently ignored.
- Grouped env output: instead of N separate rows for N missing keys, a single
  grouped row shows the count and lists all keys in the blocker message.
- Backwards compatible: existing configs with only `env.required` keep working.

## v0.3.1

Docs/packaging patch.

- Added a README demo GIF (generated with VHS; see `assets/demo.tape`).
- Rewrote the README as a value-first landing page; moved version history here.
- Bumped the CLI version string to 0.3.1. No product behavior changes.

## v0.3.0

Diagnostics expansion — majority coverage of common "works on my machine"
failure classes.

- Git drift: branch, commit, dirty/untracked, ahead/behind.
- PATH/executable resolution capture with home-directory redaction.
- System drift: timezone, locale, OS/arch, case-sensitive filesystem.
- Docker drift: runtime + compose detection, compose files, images, containers
  (read-only; no containers started).
- Active probes: TCP service reachability, DNS, HTTP status, TLS handshake/
  hostname/expiry, proxy presence.
- New `snagify probe` command; `check` gains `--timeout`, `--no-network`,
  `--no-tls`, `--no-docker`, `--insecure-probe`; `snapshot` gains `--probes`.
- Expanded `.snagify.yaml` schema (all new sections optional; v0.2 configs keep
  working).

## v0.2.0

Team workflow.

- `.snagify.yaml` repo requirements with `snagify init` and `snagify check`.
- Sanitized baselines: `baseline create` and `check --against`.
- `diff-many` team drift summary.
- Secure local baseline sharing over TLS with certificate pinning
  (`share baseline`, `check --from`).

## v0.1.0

Initial release.

- `snagify snapshot` and `snagify diff` with terminal, markdown, and JSON
  output and ranked blockers.

## v0.4.1

- Add GoReleaser release packaging.
- Prepare clean npm package `snagify`.
- Prepare clean PyPI package `snagify`.
- Add Homebrew formula template.
- No product behavior changes.

## v0.5.0

Six feature sprint — Closes the v0.4.1 roadmap (GitHub Action, plugin
system) and ships four additional capabilities that round Snagify out
against peer tooling.

- **GitHub Action** (`.github/actions/snagify-check`): composite action
  that downloads the GoReleaser binary for the runner OS/arch, verifies
  it against `dist/checksums.txt`, and runs `snagify check`. Inputs
  `version`, `config`, `fail_on_blocker`, `args`, `workdir`. Example
  workflow in `examples/workflows/snagify-ci.yml`. The project itself
  bootstraps the missing CI/releaser infrastructure at
  `.github/workflows/ci.yml` and `.github/workflows/release.yml`.
- **Protocol-level probes** (`internal/probe/db.go`): three new
  credential-free probes — `redis.ping`, `postgres.startup`,
  `mysql.handshake`. Activated via the new `databases:` section of
  `.snagify.yaml`. Failures become `report.Item{Category: "Database"}`,
  `Required` upgrades severity to Critical.
- **Custom shell checks** (`internal/customcheck`): new `checks.custom`
  block lets `.snagify.yaml` declare project-specific invariants executed
  under a per-check timeout with stdout/stderr captured and truncated.
  Unknown severity defaults to warning.
- **Docker image** (`Dockerfile` + GoReleaser `dockers:` block): published
  as `ghcr.io/HarshDevelops/snagify` for `linux/amd64` and `linux/arm64`.
  Final stage on `gcr.io/distroless/base-debian12:nonroot` so runtime
  shell-out for `node --version` etc. keeps working. See `docs/docker.md`.
- **Secret / `.env` safety scan** (`internal/capture/secretscan.go`):
  four checks that always run — `.env` tracked in git (critical),
  missing `.gitignore` coverage (warning), group/other-readable `.env`
  mode on Unix (warning), and Shannon-entropy > 4.5 over a 24+ char
  value (warning). A value-reading scan for known secret shapes
  (AWS/GitHub/Stripe/OpenAI/Slack/Google/JWT) is opt-in via
  `--scan-secrets` on `snapshot`, `check`, and `init`. Values are
  never persisted — only key names + reasons surface.
- **Plugin system** (`internal/plugin/` + companion `cmd/snagify-plugin-shellcheck/`):
  subprocess + JSON-RPC contract `snagify/plugin/v1`. Plugins are
  external binaries; the loader resolves `binary` (PATH) or `command`
  (executed directly), enforces a timeout, parses one JSON document from
  stdout, and translates per-check results into `report.Item{Category:
  "Plugin"}`. The bundled `snagify-plugin-shellcheck` companion reuses
  the same `checks.custom` schema so users get feature-3 ergonomics
  with feature-6 isolation.

Upgrade notes:
- All additions are backward-compatible additive sections; v0.4.x
  `.snagify.yaml` files keep parsing unchanged.
- `--scan-secrets` is required to enable value-reading patterns; the
  other three safety items run unconditionally.
- The Go module path stays `github.com/harshdevelops/snagify`; the
  companion binary `snagify-plugin-shellcheck` is independent (no
  Go module needed by users — it ships prebuilt in
  `dist/`).

Verified:
- `go build ./...` clean.
- `go vet ./...` clean.
- `go test ./...` — all packages green (capture, check, config,
  customcheck, plugin, probe, share, lan, initcfg, fixplan, model,
  share, team, verreq).
