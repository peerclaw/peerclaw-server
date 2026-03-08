#!/usr/bin/env bash
#
# PeerClaw systemd uninstall script.
# Usage: sudo ./uninstall.sh [--purge]
#
# --purge  Also removes config, data, logs, and the system user.
#
set -euo pipefail

PURGE=false
if [[ "${1:-}" == "--purge" ]]; then
    PURGE=true
fi

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC}  $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; exit 1; }

if [[ $EUID -ne 0 ]]; then
    error "This script must be run as root (use sudo)"
fi

info "Uninstalling PeerClaw..."

# Stop and disable service.
if systemctl is-active --quiet peerclawd 2>/dev/null; then
    systemctl stop peerclawd
    info "Stopped peerclawd service"
fi

if systemctl is-enabled --quiet peerclawd 2>/dev/null; then
    systemctl disable peerclawd
    info "Disabled peerclawd service"
fi

# Remove unit file.
rm -f /etc/systemd/system/peerclawd.service
systemctl daemon-reload
info "Removed systemd unit"

# Remove binary.
rm -f /usr/local/bin/peerclawd
info "Removed /usr/local/bin/peerclawd"

if $PURGE; then
    warn "Purging all data, config, and logs..."
    rm -rf /var/lib/peerclaw
    rm -rf /var/log/peerclaw
    rm -rf /etc/peerclaw
    info "Removed /var/lib/peerclaw, /var/log/peerclaw, /etc/peerclaw"

    if id -u peerclaw &>/dev/null; then
        userdel peerclaw 2>/dev/null || true
        info "Removed system user: peerclaw"
    fi
fi

echo ""
info "Uninstall complete."
if ! $PURGE; then
    echo "  Config, data, and logs were preserved."
    echo "  Run with --purge to remove everything."
fi
