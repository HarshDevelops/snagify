# Changelog

All notable changes to Snagify are documented here. This project adheres to
semantic versioning.

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
