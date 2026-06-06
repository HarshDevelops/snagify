# Environment Drift: Why Code Works Locally but Fails Elsewhere

Environment drift is the gradual divergence between developer machines, CI environments, and staging/production setups. It is the single largest source of "it works on my machine" problems.

## Types of drift

### Runtime drift

Different versions of Node, Java, Python, Go, Rust, Maven, Gradle, or Docker between machines. Even a minor version change can break code.

```sh
snagify diff working.json broken.json
```

Example:

```text
Java       working: 17.0.9    broken: 21.0.2   ← likely blocker
```

### Env drift

`.env.example` lists all expected keys. `.env` is often incomplete because:
- team members add keys without telling everyone
- new services require new secrets
- optional fallback providers get added

Snagify classifies keys as `required`, `recommended`, or `optional` to reduce noise.

### Docker drift

- Docker not installed or at a different version
- Required Compose files missing
- Required images not pulled
- Required containers not running

### Git drift

- Wrong branch checked out
- Dirty working tree (uncommitted changes that affect behavior)
- Missing commits that added required config

### Network and TLS drift

- DNS resolution differs (corporate VPN, split-horizon DNS)
- Internal services unreachable
- Expired or self-signed certificates

### System drift

- Different timezone causes time-sensitive test failures
- Case-insensitive filesystem (macOS) vs case-sensitive (Linux) causes file import failures
- Different locale affects date/number parsing

## Detecting drift with Snagify

Snagify captures a snapshot of the full environment and compares:

```sh
# Capture known-good machine
snagify snapshot good.json

# Capture broken machine
snagify snapshot broken.json

# Compare
snagify diff good.json broken.json --format markdown
```

Or configure expected requirements in `.snagify.yaml` and check locally:

```sh
snagify init
snagify check
```

## Privacy

Snagify never reads `.env` values — only key names. Baselines strip hostnames and absolute paths. Teammate snapshots never leave the teammate's machine in LAN compare mode.
