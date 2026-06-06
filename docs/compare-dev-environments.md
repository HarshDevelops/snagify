# Compare Two Developer Environments from the Terminal

When something works on one machine but fails on another, the fastest path to an answer is a direct comparison.

## Snapshot + diff

Capture both machines and compare:

```sh
# Machine A
snagify snapshot alice.json

# Machine B  
snagify snapshot bob.json

# Compare
snagify diff alice.json bob.json
```

Output:

```text
Project: checkout-service
A = alice   B = bob

Critical differences
────────────────────
Maven       alice: 3.9.6     bob: missing       ← likely blocker

Differences
───────────
Java        alice: 17.0.9    bob: 21.0.2
Postgres    alice: running   bob: not running
```

Export to markdown for a GitHub issue:

```sh
snagify diff alice.json bob.json --format markdown
```

## LAN compare (no file sharing needed)

When both machines are on the same Wi-Fi, office network, or VPN:

```sh
# Known-good machine:
snagify share current --lan --repo checkout-service

# Teammate:
snagify compare --lan checkout-service
```

No JSON files to export. No Slack/Teams links. The teammate's snapshot never leaves their machine — it is compared locally.

## Baseline compare

Commit a sanitized known-good baseline to the repo. Every teammate checks against it:

```sh
# Known-good machine:
snagify baseline create --out .snagify/baseline.json
git add .snagify/baseline.json && git commit -m "Add setup baseline"

# Teammates:
snagify check --against .snagify/baseline.json
```

## Privacy model

| Data | Shared? |
|---|---|
| `.env` values/secrets | Never |
| Absolute file paths | Redacted to project name |
| Hostnames | Removed from baselines |
| Runtime versions | Yes (names and versions only) |
| Teammate snapshot | Never leaves teammate's machine |

## Output formats

```sh
snagify diff a.json b.json                    # colored terminal
snagify diff a.json b.json --format markdown  # GitHub tables
snagify diff a.json b.json --format json      # machine-readable
```
