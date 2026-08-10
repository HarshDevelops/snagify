# syntax=docker/dockerfile:1.7
#
# Snagify — runtime-only container image.
#
# Stage 1 (build): bash + curl to download the v0.5.0+ snagify binary tarball
# from the GitHub release we are publishing in this same workflow. We do not
# compile from source here — that's goreleaser's job, and it already produced
# the archive in dist/ before this image is built.
#
# Stage 2 (runtime): gcr.io/distroless/base-debian12:nonroot. We use `base`
# rather than `static` because the snagify binary shells out to other tools
# (`node --version`, `docker version`, etc.) for runtime detection. `nonroot`
# because CI runners don't run as root.

FROM alpine:3.20 AS fetch
ARG VERSION=dev
ARG PLATFORM=amd64
# Map PLATFORM (amd64 | arm64) onto the GoReleaser archive suffix.
# GoReleaser emits `snagify_Linux_x86_64.tar.gz` for amd64 and
# `snagify_Linux_arm64.tar.gz` for arm64. We use busybox `wget`
# (always present in alpine:3.20) instead of `curl`, which isn't
# installed by default on slim alpine images.
RUN set -eux; \
    case "$PLATFORM" in \
        amd64) ARCH_SUFFIX=x86_64 ;; \
        arm64) ARCH_SUFFIX=arm64  ;; \
        *)     echo "unknown PLATFORM $PLATFORM" >&2; exit 1 ;; \
    esac; \
    url="https://github.com/Harshdevelops/snagify/releases/download/v${VERSION}/snagify_Linux_${ARCH_SUFFIX}.tar.gz"; \
    wget -q -O /tmp/snagify.tar.gz "$url"; \
    tar -xzf /tmp/snagify.tar.gz -C /tmp; \
    install -m 0755 /tmp/snagify /tmp/snagify-bin

FROM gcr.io/distroless/base-debian12:nonroot AS runtime

COPY --from=fetch /tmp/snagify-bin /usr/local/bin/snagify

USER nonroot:nonroot
WORKDIR /workspace
ENTRYPOINT ["/usr/local/bin/snagify"]
CMD ["--help"]
