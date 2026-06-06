# Snagify

> Snag the differences. Kill "works on my machine".

Snagify is a single-binary CLI that diagnoses why a project runs on one machine
but fails on another. It captures a focused snapshot of your environment,
runtimes, services, and env files, then produces a ranked diff that puts the
likely blockers right at the top.

No AI. No bloat. A 10–30 second workflow.

## Quickstart

```sh
# On machine A
snagify snapshot harsh.json

# On machine B (your teammate / CI)
snagify snapshot teammate.json

# Compare them
snagify diff harsh.json teammate.json
```

Example output:

```
Project: checkout-service
A = harsh   B = teammate

Critical differences
────────────────────
Java         harsh: 17.0.9   teammate: 21.0.2
Maven        harsh: 3.9.6    teammate: missing       ← likely blocker
Docker       harsh: 25.0     teammate: missing       ← likely blocker

Differences
────────────────────
5432 (Postgres)  harsh: running   teammate: not running

Likely blockers for teammate:
  • Maven is required by this project but missing on teammate
  • Docker major version mismatch ...
  • required environment variables absent on B: DATABASE_URL, JWT_SECRET
```

## Commands

### `snagify snapshot [output.json]`

Captures the current machine and project state. If no path is given, writes
`snagify-snapshot-<hostname>-<date>.json` in the current directory.

```sh
snagify snapshot                 # auto-named file
snagify snapshot mymachine.json  # explicit name
snagify snapshot -q              # quiet
```

### `snagify diff <A.json> <B.json>`

Compares two snapshots and prints ranked differences.

```sh
snagify diff a.json b.json                    # colored terminal output
snagify diff a.json b.json --format markdown  # GitHub-friendly tables
snagify diff a.json b.json --format json      # machine-readable
```

Exit code is `0` when the environments match and `1` when differences are
found, so it composes cleanly in scripts and CI.

## Team workflow (v0.2)

For teams, pairwise snapshot sharing doesn't scale. v0.2 adds repo-defined
requirements, sanitized baselines, a team drift summary, and secure
same-LAN/VPN baseline sharing.

### `snagify init [--force]`

Generates a conservative starter `.snagify.yaml` from your detected manifests
(package.json → node/npm, pom.xml → java/maven, docker-compose → docker,
`.env.example` → env keys). Never writes secret values or absolute paths.
Refuses to overwrite an existing file unless `--force` is passed.

### `snagify check`

Checks this machine against requirements and prints a ranked report. Exit code
`0` = passed, `1` = failed, `2` = config/runtime error.

```sh
snagify check                                  # against .snagify.yaml
snagify check --config path/to/.snagify.yaml   # explicit config
snagify check --against .snagify/baseline.json # against a baseline file
snagify check --format markdown|json           # other formats
```

`.snagify.yaml` supports runtime version requirements (`"17"`, `">=20 <23"`,
`">=3.8"`, `"required"`), required env keys (names only — values are never
read), and ports that must be free or listening.

### `snagify baseline create --out <path>`

Captures the current machine as a sanitized, shareable baseline: hostname and
absolute paths removed, runtime/port/env-key info kept, secret values never
included. Safe to commit.

```sh
snagify baseline create --out .snagify/baseline.json
# teammates:
snagify check --against .snagify/baseline.json
```

### `snagify diff-many --against <baseline.json> <snapshots-dir>`

Compares every snapshot in a folder against a baseline and prints a team drift
table plus the most common blockers.

### `snagify share baseline --file <baseline.json>`

Serves a baseline over a temporary, TLS-encrypted, fingerprint-pinned server on
your LAN/VPN. A one-time random token guards the path, the server expires
(default 10m) and stops after a max number of downloads (default 20), and it
never accepts uploads. The teammate's machine snapshot never leaves their
machine.

```sh
# known-good machine:
snagify share baseline --file .snagify/baseline.json
# prints a command including --pin <fingerprint>

# teammate (same LAN/VPN):
snagify check --from https://<host>:<port>/baseline/<token> --pin sha256:<fingerprint>
```

Flags: `--host`, `--port`, `--ttl`, `--max-downloads`, and `--unsafe-http`
(plain HTTP, prints a warning). Direct sharing requires the machines to reach
each other (same Wi-Fi, office LAN, or VPN) — there is no NAT traversal or
cloud relay.

## Majority-coverage diagnostics (v0.3)

v0.3 widens coverage of common "works on my machine" failure classes. All new
config sections are optional; existing v0.2 configs keep working unchanged.

`snagify snapshot` now also captures (passively, fast, no network by default):

- **Git** — branch, short commit, dirty/untracked state, ahead/behind.
- **PATH/executables** — resolved locations of common tools, with the home
  directory redacted to `~` (no usernames or absolute home paths stored).
- **System** — timezone, locale (`LC_ALL`/`LC_CTYPE`/`LANG`), and a
  case-sensitivity probe of a temp file (always cleaned up).
- **Docker** — runtime + compose version, compose files in the root, local
  images and running containers (read-only; no containers are started).

Add `--probes` to also run the active probes declared in config.

`snagify check` additionally validates git (`require_branch`, `require_clean`,
`warn_if_dirty`), required PATH commands, Docker (`required`, `compose_files`,
`required_images`, `required_containers`), and system (`timezone`, `locale`,
`case_sensitive_fs`, `allowed_arch`). It runs active probes only when the
config declares them.

```sh
snagify check --no-network   # skip DNS/HTTP probes
snagify check --no-tls       # skip TLS probes
snagify check --no-docker    # skip Docker checks
snagify check --timeout 2s   # per-probe timeout
```

### `snagify probe`

Runs only the active connectivity probes from config — handy for a quick
connectivity check without the full setup check:

- **TCP services** — reachability dial (no credentials, no DB protocols).
- **DNS** — name resolution.
- **HTTP** — status-code check only; response bodies are never read.
- **TLS** — handshake, hostname verification, issuer, and expiry. Verification
  is never disabled unless you pass `--insecure-probe` (which prints a loud
  warning).
- **Proxy** — records only whether `HTTP(S)_PROXY`/`NO_PROXY` are set, never
  their values.

Active probes only run when `services`, `network`, or `tls` sections exist in
`.snagify.yaml`. See [docs/TRACE.md](docs/TRACE.md) for the future `trace` mode
design note.

### Honest promise

Snagify detects *likely* setup blockers and environment drift. It does not
claim a proven universal root cause. Output uses "likely blocker" and "may
affect runtime behavior" rather than asserting an exact cause, except for
deterministic config violations (e.g. a required command missing from PATH).

### Global flags

- `--project-root <path>` — override project auto-detection.
- `--verbose` / `-v` — extra detail on stderr.

## What it captures

- **Project**: root detection by walking up for `package.json`, `go.mod`,
  `pom.xml`, `build.gradle`, `requirements.txt`, `Cargo.toml`, and more; plus a
  project name.
- **Environment**: OS, arch, OS version, git version.
- **Runtimes**: Node, npm, pnpm, Yarn, Python, pip, uv, Java, Maven, Gradle,
  Go, Rust, Cargo, Docker — version and presence.
- **Services**: common dev ports (3000, 5432, 6379, 8080, 9200, 27017, …) and
  whether they're listening.

  > Port detection limitation (v0.1): Snagify checks listening status by
  > attempting a short TCP connection to `127.0.0.1`. It reports only whether a
  > port is accepting connections on localhost — it does not identify the owning
  > process and does not detect services bound to other interfaces. This is an
  > intentional MVP tradeoff favoring cross-platform portability (no `ss`,
  > `lsof`, or `netstat` dependency, no elevated privileges).
- **Env files**: presence of `.env.example` / `.env` and which declared keys
  are missing. Only key names are read — values are never inspected.

## How ranking works

- **Critical** — likely blockers: a required runtime missing for the detected
  project type, a major version mismatch, or missing required env keys.
- **Differences** — real mismatches that may or may not matter (minor version
  drift, optional tools, port state).
- **Informational** — OS/arch context.

## Install

```sh
go install github.com/harshdevelops/snagify@latest
```

Or build from source:

```sh
git clone https://github.com/harshdevelops/snagify.git
cd snagify
go mod tidy
go build -o snagify .
```

## Development

```sh
go mod tidy
go run . snapshot
go build -o snagify .
go test ./...
```

## License

MIT
