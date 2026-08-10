# Running Snagify in Docker

The Snagify container ships a single static binary on a slim distroless base
(no shell, no package manager, no daemons). It is published as
`ghcr.io/HarshDevelops/snagify` for amd64 and arm64.

## Quickstart

```sh
docker run --rm ghcr.io/HarshDevelops/snagify --version
docker run --rm -v "$PWD:/workspace" -w /workspace \
  ghcr.io/HarshDevelops/snagify check
```

The image runs as the `nonroot` user (uid 65532) and uses `/workspace` as
its working directory. Mount your project there.

## Tags

| Tag | What it is |
|---|---|
| `latest` | Most recent stable release on the default branch. |
| `X.Y.Z` | An exact SemVer release (e.g. `0.5.0`). |
| `X.Y` | The latest patch for a minor line. |
| `X.Y.Z-next` | The next release preview (matches `goreleaser --snapshot`). |

## Multi-arch

Manifests for `linux/amd64` and `linux/arm64` are emitted automatically
from the project's `.github/workflows/release.yml` on every `v*.*.*` tag
push. Pull whichever your host wants:

```sh
docker pull --platform=linux/arm64 ghcr.io/HarshDevelops/snagify
```

## Why distroless/base and not `scratch`?

Snagify's runtime detection (the `runtimes` snapshot) shells out to whatever
is installed on the *host* (`node --version`, `docker version`, etc.). The
container itself doesn't need a package manager to do this — but a pure
`scratch` image lacks the dynamic loader and CA bundle that the Go runtime
expects. `distroless/base-debian12:nonroot` is the smallest image that
still satisfies `go` runtime requirements *and* runs as a non-root user.

If you don't need the runtime-detect helpers (e.g. you only want
`snagify check --against some-baseline.json`), `distroless/static` would
work — but the savings are minor (~10MB) and the UX cost (mysterious
`env: 'node': not found` style failures when users run `--probes`) is real.

## Building locally

```sh
docker build -t snagify:dev .
docker run --rm snagify:dev --version
# Multi-arch (requires buildx):
docker buildx build --platform linux/amd64,linux/arm64 -t snagify:dev .
```

## Security notes

- The image runs as `nonroot` (uid 65532). Mount `/workspace` read-write
  if you need snagify to write a baseline or update `.env`.
- No `tini` or init — the binary does not fork or daemonize. If you run
  snagify in a container with the `--lan` share, the share lives as long
  as the process.
- No secrets are read by the binary. Mounting `~/.aws`, `~/.docker`, or
  any other credential store does not put them at risk.
