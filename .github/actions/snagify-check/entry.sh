#!/usr/bin/env bash
# entry.sh — composite action step for snagify-check.
#
# Downloads the GoReleaser-published snagify binary for the runner OS/arch,
# verifies its SHA256 against dist/checksums.txt, and exports it to PATH.
#
# Environment (set by action.yml from inputs):
#   INPUT_VERSION        - "latest" or a specific tag like "v0.5.0"
#   INPUT_CONFIG         - path to .snagify.yaml (forwarded to the gate step)
#   INPUT_FAIL_ON_BLOCKER - "true" or "false"
#   INPUT_ARGS           - passthrough args (forwarded to the gate step)
#
# The setup step is intentionally separate from the gate step so that CI can
# reuse the downloaded binary via $GITHUB_ACTION_PATH/setup/... — see the
# README for an advanced usage.
set -euo pipefail

REPO="${REPO:-HarshDevelops/snagify}"
VERSION="${INPUT_VERSION:-latest}"
INSTALL_DIR="${RUNNER_TEMP:-/tmp}/snagify-bin"
mkdir -p "${INSTALL_DIR}"

# ---------------------------------------------------------------------------
# Resolve the version tag.
# ---------------------------------------------------------------------------
if [[ "${VERSION}" == "latest" ]]; then
  echo "Resolving latest snagify release via GitHub API..."
  if command -v gh >/dev/null 2>&1; then
    VERSION="$(gh release view --repo "${REPO}" --json tagName -q .tagName)"
  else
    VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
      | grep -oE '"tag_name"\s*:\s*"[^"]+"' | head -1 \
      | sed -E 's/.*"([^"]+)"$/\1/')"
  fi
  if [[ -z "${VERSION}" || "${VERSION}" == "null" ]]; then
    echo "::error::Could not resolve latest snagify release for ${REPO}"
    exit 1
  fi
fi

# Strip a leading v for version-constant handling, but keep it in URLs.
VERSION_TAG="${VERSION}"
VERSION_NUM="${VERSION_TAG#v}"

# ---------------------------------------------------------------------------
# Map runner OS/arch → GoReleaser archive name.
# ---------------------------------------------------------------------------
case "${RUNNER_OS:-}" in
  Linux)   OS_TITLE="Linux"   ;;
  macOS)   OS_TITLE="Darwin"  ;;
  Windows) OS_TITLE="Windows" ;;
  *)
    echo "::error::Unsupported runner OS: ${RUNNER_OS:-unknown}"
    exit 1
    ;;
esac

case "${RUNNER_ARCH:-}" in
  X64)   ARCH_TOKEN="x86_64" ;;
  ARM64) ARCH_TOKEN="arm64"  ;;
  *)
    echo "::error::Unsupported runner arch: ${RUNNER_ARCH:-unknown}"
    exit 1
    ;;
esac

case "${OS_TITLE}-${ARCH_TOKEN}" in
  Windows-x86_64) ARCHIVE="snagify_${OS_TITLE}_x86_64.zip" ;;
  *)               ARCHIVE="snagify_${OS_TITLE}_${ARCH_TOKEN}.tar.gz" ;;
esac

URL="https://github.com/${REPO}/releases/download/${VERSION_TAG}/${ARCHIVE}"
CHECKSUM_URL="https://github.com/${REPO}/releases/download/${VERSION_TAG}/checksums.txt"

echo "Downloading snagify ${VERSION_TAG} (${RUNNER_OS:-}/${RUNNER_ARCH:-})..."
echo "  ${URL}"

ARCHIVE_PATH="${INSTALL_DIR}/${ARCHIVE}"
CHECKSUMS_PATH="${INSTALL_DIR}/checksums.txt"

curl -fsSL -o "${ARCHIVE_PATH}"     "${URL}"
curl -fsSL -o "${CHECKSUMS_PATH}"   "${CHECKSUM_URL}"

# ---------------------------------------------------------------------------
# Verify SHA256.
# ---------------------------------------------------------------------------
EXPECTED_SHA="$(grep "  ${ARCHIVE}\$" "${CHECKSUMS_PATH}" | awk '{print $1}')"
if [[ -z "${EXPECTED_SHA}" ]]; then
  echo "::error::checksums.txt has no entry for ${ARCHIVE}"
  exit 1
fi

if [[ "${OS_TITLE}" == "Windows" ]]; then
  ACTUAL_SHA="$(sha256sum "${ARCHIVE_PATH}" | awk '{print $1}')"
else
  ACTUAL_SHA="$(shasum -a 256 "${ARCHIVE_PATH}" | awk '{print $1}')"
fi

if [[ "${ACTUAL_SHA}" != "${EXPECTED_SHA}" ]]; then
  echo "::error::Checksum mismatch for ${ARCHIVE}"
  echo "  expected: ${EXPECTED_SHA}"
  echo "  actual:   ${ACTUAL_SHA}"
  exit 1
fi

# ---------------------------------------------------------------------------
# Extract.
# ---------------------------------------------------------------------------
BIN_NAME="snagify"
if [[ "${OS_TITLE}" == "Windows" ]]; then BIN_NAME="snagify.exe"; fi

cd "${INSTALL_DIR}"
case "${ARCHIVE}" in
  *.tar.gz)
    tar -xzf "${ARCHIVE_PATH}"
    ;;
  *.zip)
    unzip -o "${ARCHIVE_PATH}"
    ;;
esac

if [[ ! -x "${INSTALL_DIR}/${BIN_NAME}" ]]; then
  echo "::error::Extracted binary not found at ${INSTALL_DIR}/${BIN_NAME}"
  ls -la "${INSTALL_DIR}"
  exit 1
fi

# ---------------------------------------------------------------------------
# Install onto PATH for subsequent composite steps.
# ---------------------------------------------------------------------------
BIN_DIR="${INSTALL_DIR}/bin"
mkdir -p "${BIN_DIR}"
mv "${INSTALL_DIR}/${BIN_NAME}" "${BIN_DIR}/${BIN_NAME}"
chmod +x "${BIN_DIR}/${BIN_NAME}"
echo "${BIN_DIR}" >> "${GITHUB_PATH}"

# Surface the resolved version + binary dir as step outputs for downstream.
{
  echo "version=${VERSION_TAG}"
  echo "binary=${BIN_DIR}/${BIN_NAME}"
} >> "${GITHUB_OUTPUT}"

# Sanity-check the binary runs.
"${BIN_DIR}/${BIN_NAME}" --version

# Echo the forwarded inputs so users can see what was captured.
{
  echo "## Snagify check setup"
  echo "version=${VERSION_TAG}"
  echo "config=${INPUT_CONFIG}"
  echo "fail_on_blocker=${INPUT_FAIL_ON_BLOCKER}"
  echo "args=${INPUT_ARGS}"
} >> "${GITHUB_STEP_SUMMARY}"
