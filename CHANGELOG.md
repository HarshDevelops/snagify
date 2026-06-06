# Changelog

All notable changes to Snagify are documented here. This project adheres to
semantic versioning.

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
