# How to Debug "It Works on My Machine" Issues

> "It works on my machine" is the most common phrase in software development,
> and usually the most expensive.

## What it means

When code works on one developer's machine but fails on another, the root cause is almost always environment drift — invisible differences between two setups that took hours or days to detect manually.

## Common causes

| Category | Examples |
|---|---|
| Runtime drift | Node 18 vs 22, Java 17 vs 21, Python 3.9 vs 3.11 |
| Missing env keys | `.env.example` exists but `.env` is incomplete |
| Wrong Git state | wrong branch, dirty working tree, missing commit |
| Docker drift | Docker missing, Compose files missing, required images not pulled |
| Port/service drift | Postgres/Redis not running locally |
| PATH drift | command resolves to different binary via nvm/pyenv/asdf |
| System drift | different timezone, locale, or case-sensitive filesystem |

## Manual checklist

Before using a tool, you can check manually:

1. `node --version` / `java -version` / `python --version` on both machines
2. `diff .env.example .env` — look for missing keys
3. `git status` — check branch and dirty state
4. `docker ps` — check which containers are running
5. `echo $PATH` — check if version managers are active

This takes 15–30 minutes. Snagify does it in seconds.

## Snagify workflow

**Step 1 — define your project's requirements once:**

```sh
snagify init
```

This generates a `.snagify.yaml` in your repo root, classifying env keys as `required`, `recommended`, or `optional` based on conservative heuristics.

**Step 2 — any teammate runs a local check:**

```sh
snagify check
```

Example output:

```text
Project: checkout-service
Your setup is not ready

Critical blockers
─────────────────
Maven      found: missing     expected: required
.env       3 required keys missing

Likely blockers:
  • Maven is missing
  • Required env keys are absent: DATABASE_URL, JWT_SECRET, REDIS_URL
```

**Step 3 — known-good machine shares their setup over LAN:**

```sh
# Known-good machine:
snagify share current --lan --repo checkout-service

# Teammate (same LAN/VPN):
snagify compare --lan checkout-service
```

The teammate's snapshot never leaves their machine.

## Doctor and fix

```sh
snagify doctor         # detect blockers + suggest fixes
snagify fix --safe     # apply safe fixes (create .env, append missing keys)
snagify fix --guided   # print version-manager commands (does not execute)
```

## Honest limits

Snagify detects likely setup blockers and deterministic config violations. It does not prove a universal root cause and does not replace real debugging.
