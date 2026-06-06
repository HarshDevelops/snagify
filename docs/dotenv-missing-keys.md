# Find Missing .env Keys Before Your App Breaks

`.env` files are the most common source of silent setup failures. An app that crashes with `undefined is not a function` or `connection refused` is often missing an env key that was added by a teammate and never propagated.

## The problem

1. A teammate adds `REDIS_URL` to the app and to CI secrets.
2. They add it to `.env.example` with a blank value.
3. Other teammates pull the branch, run the app, and get a cryptic error.
4. Twenty minutes later: "Oh, you need to add REDIS_URL to your .env."

## Snagify's approach

Snagify compares `.env.example` against `.env` and reports missing keys — without ever reading actual values.

```sh
snagify check
```

```text
.env  3 required keys missing

Likely blockers:
  • Required env keys are absent: DATABASE_URL, JWT_SECRET, REDIS_URL
```

## Required, recommended, and optional

`snagify init` classifies keys from `.env.example` automatically:

| Classification | Trigger | Missing = |
|---|---|---|
| `required` | blank value + sensitive name (API_KEY, TOKEN, SECRET, etc.) | Critical blocker |
| `recommended` | concrete default (MAX_RETRIES=3) or tuning name | Warning |
| `optional` | disabled feature group (FALLBACK_1_ENABLED=false) | Silent |

Example `.env.example`:

```env
DATABASE_URL=
JWT_SECRET=

MAX_RETRIES=3
TIMEOUT_MS=5000

FALLBACK_1_ENABLED=false
FALLBACK_1_API_KEY=
FALLBACK_1_MODEL=
```

Result after `snagify init`:

```yaml
env:
  required:
    - DATABASE_URL
    - JWT_SECRET
  recommended:
    - MAX_RETRIES
    - TIMEOUT_MS
  optional:
    - FALLBACK_1_ENABLED
    - FALLBACK_1_API_KEY
    - FALLBACK_1_MODEL
```

The fallback provider's API key is `optional` because `FALLBACK_1_ENABLED=false` in `.env.example`.

## Comment hints

Override heuristics with a comment on the line before:

```env
# optional
ANALYTICS_API_KEY=

# required
PAYMENT_SECRET=
```

## What Snagify never does

- Never reads actual `.env` values
- Never prints secret values
- Never stores or transmits credentials

## Fix missing keys

```sh
snagify fix --safe              # create .env from .env.example / append missing keys
snagify fix --safe --dry-run   # preview without applying
```

Safe fix creates `.env` from `.env.example` and appends blank placeholders for missing required keys. You still need to set real values.
