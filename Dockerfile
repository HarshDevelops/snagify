# syntax=docker/dockerfile:1.7
#
# Snagify — multi-stage container image.
#
# Stage 1 (build): golang toolchain matching the project's go.mod. We build
# the binary here from source, with CGO disabled to produce a static binary
# the distroless final stage can run unmodified.
#
# Stage 2 (runtime): gcr.io/distroless/base-debian12:nonroot. We use `base`
# (not `static` / scratch) because Snagify's runtime-probe helpers shell out
# to other tools (`node --version`, `docker version`, etc.). `nonroot`
# matches the assumption that CI runners don't run as root.

ARG GO_VERSION=1.25

FROM golang:${GO_VERSION}-alpine AS build
WORKDIR /src

# Cache the module graph first so `go mod download` only re-runs when the
# go.mod / go.sum change. .dockerignore keeps the build context small.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ENV CGO_ENABLED=0 GOOS=linux
RUN --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" \
    -o /out/snagify .

FROM gcr.io/distroless/base-debian12:nonroot AS runtime

COPY --from=build /out/snagify /usr/local/bin/snagify

USER nonroot:nonroot
WORKDIR /workspace
ENTRYPOINT ["/usr/local/bin/snagify"]
CMD ["--help"]
