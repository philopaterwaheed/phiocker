#!/usr/bin/env bash

set -euo pipefail

SERVICE_NAME="phiocker"
SERVICE_FILE="${SERVICE_NAME}.service"
SYSTEMD_DIR="/etc/systemd/system"
BINARY_SRC="$(go env GOPATH)/bin/${SERVICE_NAME}"
BINARY_DST="/usr/local/bin/${SERVICE_NAME}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

need_root() {
    if [[ $EUID -ne 0 ]]; then
        echo "Error: this script must be run as root (use sudo)." >&2
        exit 1
    fi
}

cmd_install() {
    need_root

    echo "==> Building phiocker..."
    pushd "${SCRIPT_DIR}" >/dev/null
    go build -o "${BINARY_DST}" ./cmd/phiocker
    popd >/dev/null
    chmod +x "${BINARY_DST}"
    echo "    Binary installed at ${BINARY_DST}"

    echo "==> Installing systemd unit..."
    cp "${SCRIPT_DIR}/${SERVICE_FILE}" "${SYSTEMD_DIR}/${SERVICE_FILE}"
    systemctl daemon-reload
    systemctl enable "${SERVICE_NAME}"
    systemctl start  "${SERVICE_NAME}"

    echo ""
    echo "Service '${SERVICE_NAME}' is now enabled and started."
    echo "Check status with:  sudo systemctl status ${SERVICE_NAME}"
}

cmd_uninstall() {
    need_root

    echo "==> Stopping and disabling service..."
    systemctl stop    "${SERVICE_NAME}" 2>/dev/null || true
    systemctl disable "${SERVICE_NAME}" 2>/dev/null || true
    rm -f "${SYSTEMD_DIR}/${SERVICE_FILE}"
    systemctl daemon-reload
    echo "    Service removed."

    read -rp "Also remove the binary at ${BINARY_DST}? [y/N] " ans
    if [[ "${ans,,}" == "y" ]]; then
        rm -f "${BINARY_DST}"
        echo "    Binary removed."
    fi
}

cmd_status() {
    systemctl status "${SERVICE_NAME}" || true
}

usage() {
    echo "Usage: $0 {install|uninstall|status}"
}

case "${1:-}" in
    install)   cmd_install   ;;
    uninstall) cmd_uninstall ;;
    status)    cmd_status    ;;
    *)         usage; exit 1 ;;
esac
