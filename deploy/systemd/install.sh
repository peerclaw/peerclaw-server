#!/usr/bin/env bash
#
# PeerClaw systemd installation script.
# Usage: sudo ./install.sh [path-to-binary]
#
# This script:
#   1. Creates the peerclaw system user and group
#   2. Installs the binary to /usr/local/bin/
#   3. Sets up configuration and data directories
#   4. Installs and enables the systemd unit
#
set -euo pipefail

BINARY="${1:-../../bin/peerclawd}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC}  $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; exit 1; }

# Must run as root.
if [[ $EUID -ne 0 ]]; then
    error "This script must be run as root (use sudo)"
fi

# Check binary exists.
if [[ ! -f "$BINARY" ]]; then
    error "Binary not found: $BINARY\n       Build first with: make build"
fi

info "Installing PeerClaw..."

# 1. Create system user.
if ! id -u peerclaw &>/dev/null; then
    useradd --system --no-create-home --shell /usr/sbin/nologin --user-group peerclaw
    info "Created system user: peerclaw"
else
    info "System user peerclaw already exists"
fi

# 2. Install binary.
install -m 0755 "$BINARY" /usr/local/bin/peerclawd
info "Installed binary to /usr/local/bin/peerclawd"

# 3. Create directories.
install -d -m 0755 -o peerclaw -g peerclaw /var/lib/peerclaw
install -d -m 0755 -o peerclaw -g peerclaw /var/log/peerclaw
install -d -m 0750 -o root    -g peerclaw /etc/peerclaw
info "Created directories: /var/lib/peerclaw, /var/log/peerclaw, /etc/peerclaw"

# 4. Install config (don't overwrite existing).
if [[ ! -f /etc/peerclaw/config.yaml ]]; then
    install -m 0640 -o root -g peerclaw "$SCRIPT_DIR/../../configs/peerclaw.production.yaml" /etc/peerclaw/config.yaml
    info "Installed config to /etc/peerclaw/config.yaml"
else
    warn "Config /etc/peerclaw/config.yaml already exists, skipping"
fi

if [[ ! -f /etc/peerclaw/peerclaw.env ]]; then
    install -m 0640 -o root -g peerclaw "$SCRIPT_DIR/peerclaw.env" /etc/peerclaw/peerclaw.env
    info "Installed env file to /etc/peerclaw/peerclaw.env"
    warn "Edit /etc/peerclaw/peerclaw.env and set JWT_SECRET before starting"
else
    warn "Env file /etc/peerclaw/peerclaw.env already exists, skipping"
fi

# 5. Install systemd unit.
install -m 0644 "$SCRIPT_DIR/peerclawd.service" /etc/systemd/system/peerclawd.service
systemctl daemon-reload
info "Installed systemd unit: peerclawd.service"

# 6. Enable (but don't start yet).
systemctl enable peerclawd.service
info "Enabled peerclawd.service"

echo ""
info "Installation complete!"
echo ""
echo "  Next steps:"
echo "    1. Edit /etc/peerclaw/peerclaw.env and set JWT_SECRET:"
echo "       sudo nano /etc/peerclaw/peerclaw.env"
echo ""
echo "    2. Review /etc/peerclaw/config.yaml for your environment:"
echo "       sudo nano /etc/peerclaw/config.yaml"
echo ""
echo "    3. Start the service:"
echo "       sudo systemctl start peerclawd"
echo ""
echo "    4. Check status:"
echo "       sudo systemctl status peerclawd"
echo "       sudo journalctl -u peerclawd -f"
echo ""
