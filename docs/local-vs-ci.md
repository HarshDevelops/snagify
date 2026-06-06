# Debug Local vs CI Environment Differences

CI failures that pass locally are one of the most frustrating developer experiences. Snagify helps narrow down the differences.

## Common causes of local vs CI drift

### Runtime mismatch

CI uses an exact pinned version. Local machines use whatever is installed.

```text
Node       local: 22.3.0    CI: 20.11.1   ← likely blocker
```

Snagify `check` against a committed baseline catches this before you push.

### Missing secrets or env keys

CI injects secrets from a vault or secrets manager. Local machines use `.env`. Keys added to CI are often forgotten in `.env.example`.

```text
.env       3 required keys missing
```

Snagify's env key classification:
- `required` — critical, blocks check
- `recommended` — warning, worth setting
- `optional` — silent, disabled feature providers etc.

### Services not running

CI starts services via Docker Compose. Locally, developers often forget to `docker compose up`.

```text
Postgres    local: not running   expected: listening:5432
```

### Docker / Compose drift

CI uses the Compose file directly. Local machines may have a different Docker version or a missing image.

### Case-sensitive filesystem

CI (Linux) is case-sensitive. macOS is case-insensitive by default. A `require('./Routes')` that imports `routes.js` works on Mac but fails in CI.

Snagify captures case-sensitivity and can warn on mismatch.

## Workflow

1. Configure `.snagify.yaml` with project requirements
2. Developers run `snagify check` before pushing
3. Optional: add to pre-push git hook

```sh
snagify init
snagify check || { echo "Fix setup blockers before pushing"; exit 1; }
```

4. Commit a baseline from a known-good CI-like environment:

```sh
snagify baseline create --out .snagify/baseline.json
```

5. Check against baseline before pushing:

```sh
snagify check --against .snagify/baseline.json
```

## Note on CI integration

Snagify v0.4 does not install CI-specific integrations. Running `snagify check` as a CI step is straightforward, but automating it into GitHub Actions / GitLab CI / CircleCI is planned for a future version.
