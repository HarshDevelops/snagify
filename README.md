# Snagify

**Snag the differences. Kill "works on my machine".**

<p align="center"><img src="assets/demo.gif" alt="Snagify demo" width="820"></p>

Snagify is a single-binary CLI that surfaces likely setup blockers when a
project works on one machine but fails on another.

It checks the boring things that usually waste hours:

- wrong Node / Java / Python / Go version
- missing `.env` keys
- wrong Git branch or dirty working tree
- Docker / Compose drift
- ports and local services
- DNS / HTTP / TLS connectivity
- PATH and executable resolution
- timezone, locale, OS/arch, case-sensitive filesystem drift

No AI. No daemon. No cloud. No secret values read.

## Install

```sh
go install github.com/harshdevelops/snagify@latest
```

Or build from source:

```sh
git clone https://github.com/harshdevelops/snagify.git
cd snagify
go build -o snagify .
```

## 30-second demo

Inside a repo:

```sh
snagify init     # generate a starter .snagify.yaml from your project
snagify check    # check this machine against it
```

Example:

```text
Project: checkout-service
Your setup is not ready

Critical blockers
─────────────────
Java       found: 21.0.2      expected: 17
Maven      found: missing     expected: >=3.8
.env       missing: JWT_SECRET
Port 8080  occupied           expected: free

Likely blockers:
  • Maven is missing
  • Java version does not match the project requirement
  • JWT_SECRET is absent from .env
  • Port 8080 is already occupied
```

Exit code is `0` when the setup is ready, `1` when there are likely blockers,
and `2` on a config or runtime error — so it drops straight into CI or a
pre-flight script.

## What Snagify catches

| Class | Examples |
|---|---|
| Runtime drift | Node, Java, Python, Go, Rust, Maven, Gradle, Docker |
| Git drift | wrong branch, dirty tree, untracked files, commit mismatch |
| Env drift | missing keys from `.env` based on `.env.example` |
| Docker drift | Docker missing, Compose files missing, required images/containers |
| Port/service drift | ports that must be free/listening, TCP service reachability |
| Network drift | DNS, HTTP status checks, proxy presence |
| TLS drift | handshake failure, hostname mismatch, expired/expiring certs |
| System drift | OS/arch, timezone, locale, case-sensitive filesystem |
| PATH drift | command missing, executable resolves to a different path |

## Compare two machines

When you don't have a config yet, compare a working machine against a broken one
directly.

On the working machine:

```sh
snagify snapshot working.json
```

On the broken machine:

```sh
snagify snapshot broken.json
```

Compare them:

```sh
snagify diff working.json broken.json
```

Example:

```text
Project: checkout-service
A = working   B = broken

Critical differences
────────────────────
Maven       working: 3.9.6     broken: missing       ← likely blocker
Docker      working: present   broken: missing       ← likely blocker
.env        broken missing: DATABASE_URL, JWT_SECRET

Differences
───────────
Java        working: 17.0.9    broken: 21.0.2
Postgres    working: running   broken: not running

Likely blockers for broken:
  • Maven is missing
  • Docker is not installed
  • Required environment keys are absent: DATABASE_URL, JWT_SECRET
```

`--format markdown` produces GitHub-friendly tables; `--format json` is
machine-readable.

## Workflows

### Repo requirements

Define expected setup in `.snagify.yaml` and let every teammate check locally.

```sh
snagify init          # generate a conservative starter config
snagify check         # check this machine against it
```

`.snagify.yaml` supports runtime version requirements (`"17"`, `">=20 <23"`,
`">=3.8"`, `"required"`), required `.env` keys (names only — values are never
read), ports that must be free or listening, git branch/clean rules, required
PATH commands, Docker/compose requirements, system expectations, and active
service/network/TLS probes.

### Baselines for teams

Capture a known-good machine as a sanitized baseline, commit it, and let
teammates compare against it — no one has to export and email snapshots.

```sh
# known-good machine:
snagify baseline create --out .snagify/baseline.json

# teammates:
snagify check --against .snagify/baseline.json
```

The baseline strips hostname and absolute paths and never contains secret
values. Safe to commit.

### Secure LAN/VPN baseline sharing

Share a baseline over a temporary, TLS-encrypted, fingerprint-pinned server.
The teammate's machine snapshot never leaves their machine.

```sh
# known-good machine:
snagify share baseline --file .snagify/baseline.json
# prints a command including --pin <fingerprint>

# teammate (same LAN/VPN):
snagify check --from https://<host>:<port>/baseline/<token> --pin sha256:<fingerprint>
```

A one-time random token guards the path, the server expires (default 10m) and
stops after a max number of downloads (default 20), and it never accepts
uploads. Requires the machines to reach each other directly (same Wi-Fi, office
LAN, or VPN) — there is no NAT traversal or cloud relay.

### Active probes

Check connectivity without a full setup check:

```sh
snagify probe
```

Runs only the `services` (TCP), `network` (DNS/HTTP/proxy), and `tls` probes
declared in your config. HTTP probes check status codes only — response bodies
are never read. TLS verification is never disabled unless you pass
`--insecure-probe` (which prints a loud warning).

### Team drift summary

Compare many snapshots against a baseline and see where the team diverges.

```sh
snagify diff-many --against baseline.json snapshots/
```

Prints a per-machine drift table plus the most common blockers across the team.

## Commands

| Command | Purpose |
|---|---|
| `snagify snapshot [out.json]` | Capture this machine's state |
| `snagify diff <a> <b>` | Compare two snapshots |
| `snagify init` | Generate a starter `.snagify.yaml` |
| `snagify check` | Check this machine against config or a baseline |
| `snagify probe` | Run only the active connectivity probes |
| `snagify baseline create` | Capture a sanitized, shareable baseline |
| `snagify share baseline` | Serve a baseline over secure local TLS |
| `snagify diff-many` | Summarize team drift against a baseline |

Useful flags: `--format terminal|markdown|json`, `--project-root <path>`,
`--timeout 2s`, `--no-network`, `--no-tls`, `--no-docker`, `snapshot --probes`.

## Security & privacy

- `.env` values are never read — only key names are parsed.
- Baselines never contain hostnames, absolute paths, or secret values.
- Proxy variables are recorded as set/unset, never their values.
- Captured paths redact the home directory to `~`.
- Peer sharing is TLS-encrypted with mandatory certificate pinning; plain HTTP
  requires `--unsafe-http` and prints a warning.
- Sharing servers are short-lived (expiry + max downloads), accept no uploads,
  and there is no telemetry, daemon, or cloud component.

## Honest limits

Snagify does **not** prove universal root cause.

It detects likely setup blockers and deterministic config violations. It does
not replace Docker, Nix, Devbox, tracing tools, or actual debugging.

A future `trace` mode is documented in [`docs/TRACE.md`](docs/TRACE.md), but
this release does not run `strace`, ProcMon, or dtruss.

## Development

```sh
go mod tidy
go test ./...
go vet ./...
go build -o snagify .
```

## License

MIT
