# syntax=docker/dockerfile:1.7
#
# Snagify — multi-arch container image.
#
# Stage 1 (build): golang toolchain identical to go.mod's go directive, CGO
# disabled to produce a statically linked binary that the distroless final
# stage can run unmodified.
#
# Stage 2 (runtime): gcr.io/distroless/base-debian12:nonroot. We use `base`
# rather than `static` because the snagify binary shells out to other tools
# (`node --version`, `docker version`, etc.) for runtime detection; pure
# scratch would lack the dynamic loader infrastructure those rely on, and
# `nonroot` matches the assumption that CI runners do not run as root.

ARG GO_VERSION=1.25

FROM golang:${GO_VERSION}-alpine AS build
WORKDIR /src

# Cache the module graph first so `go mod download` only re-runs when go.mod
# or go.sum change.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Single static binary; ldflags strip the symbol table and inject version.
ARG VERSION=dev
ENV CGO_ENABLED=0 GOOS=linux
RUN --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" \
    -o /out/snagify .

FROM gcr.io/distroless/base-debian12:nonroot AS runtime

# Copy CA bundle + tzdata via the distroless `static` package contents.
# `base` already ships /etc/ssl/certs/ca-certificates.crt and basic /etc/passwd.
COPY --from=build /out/snagify /usr/local/bin/snagify

USER nonroot:nonroot
WORKDIR /workspace
ENTRYPOINT ["/usr/local/bin/snagify"]
CMD ["--help"]
