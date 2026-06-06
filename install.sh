#!/usr/bin/env bash
# Snagify installer. Builds from source if Go is available, otherwise points
# to the releases page. Usage: ./install.sh [install-dir]
set -euo pipefail

REPO="github.com/HarshDevelops/snagify"
INSTALL_DIR="${1:-${HOME}/.local/bin}"
BIN="snagify"

main() {
	if command -v go >/dev/null 2>&1; then
		echo "Go found — installing ${BIN} via go install..."
		GOBIN="${INSTALL_DIR}" go install "${REPO}@latest"
		echo "Installed to ${INSTALL_DIR}/${BIN}"
	else
		echo "Go is not installed."
		echo "Install Go (https://go.dev/dl/) and re-run, or download a prebuilt"
		echo "binary from https://${REPO}/releases"
		exit 1
	fi

	case ":${PATH}:" in
		*":${INSTALL_DIR}:"*) ;;
		*) echo "Note: add ${INSTALL_DIR} to your PATH to run ${BIN} directly." ;;
	esac
}

main "$@"
