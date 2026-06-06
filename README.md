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
go install github.com/HarshDevelops/snagify@latest
```

Or build from source:

```sh
git clone https://github.com/HarshDevelops/snagify.git
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
