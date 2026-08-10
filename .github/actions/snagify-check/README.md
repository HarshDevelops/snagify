# snagify-check

A composite GitHub Action that runs [`snagify check`](https://github.com/HarshDevelops/snagify)
against the repository on every push and pull request, blocking merges when
critical environment-drift blockers are detected.

## Why use it?

A `works-on-my-machine` failure is a CI failure that wasn't caught locally.
Snagify surfaces likely blockers (wrong Node/Java/Python/Go version, missing
`.env` keys, port conflicts, missing Docker images, untrusted TLS certs,
database outages) before the merge that introduces them.

This action calls `snagify check` against the repo's `.snagify.yaml` and:
- **exits non-zero** (fails the job) when critical blockers are present.
- **exits zero** when the local environment is ready.

The binary is downloaded from the official Snagify release on GitHub Releases
and verified against `dist/checksums.txt`. No Go toolchain required on the
runner.

## Inputs

| Name | Default | Description |
|---|---|---|
| `version` | `latest` | Snagify release tag (e.g. `v0.5.0`) or `latest`. |
| `config` | `.snagify.yaml` | Path to the config file (relative to repo root). |
| `fail_on_blocker` | `true` | Whether to fail the job when blockers are detected. Set `false` for advisory mode. |
| `args` | `""` | Extra args to forward to `snagify check` (e.g. `--no-tls --timeout 5s`). |
| `workdir` | `.` | Working directory where snagify runs. |

## Outputs

None. The action's effect is on the job exit code, surfaced via the step's
`outcome` property in dependent steps.

## Example usage

### Drop-in replacement for `run: snagify check`

```yaml
# .github/workflows/ci.yml
name: Environment drift check
on:
  pull_request:
  push:
    branches: [main]

jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: HarshDevelops/snagify/.github/actions/snagify-check@main
        with:
          args: --no-tls --timeout 10s
```

### Pinning a specific version

```yaml
      - uses: HarshDevelops/snagify/.github/actions/snagify-check@v0.5.0
```

### Advisory mode (don't fail the job)

```yaml
      - uses: HarshDevelops/snagify/.github/actions/snagify-check@main
        with:
          fail_on_blocker: 'false'
```

### Running `snagify probe` against the same checkout

`probe` shares the binary installation; wire it as a second composite step:

```yaml
      - uses: HarshDevelops/snagify/.github/actions/snagify-check@main
        with:
          fail_on_blocker: 'false'
          args: ''
      - run: snagify probe --format json > probe-report.json
      - uses: actions/upload-artifact@v4
        with:
          name: snagify-probe
          path: probe-report.json
```

## Security

- Binary is verified against the SHA256 listed in
  `dist/checksums.txt` of the chosen release.
- The action makes HTTPS GETs to `github.com` only (downloads) and
  optionally calls `gh release view` (also `api.github.com`). No telemetry,
  no analytics, no other network.
- The composite action does **not** read or transmit `.env` values.

## License

MIT — same as Snagify itself.
