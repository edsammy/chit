#!/bin/bash
set -e

# Full setup script for a fresh VPS. Run as root from /opt/chit.
# Usage: bash deploy/setup.sh

CHIT_DIR=$(pwd)

if [ "$(id -u)" -ne 0 ]; then
    echo "Run as root"
    exit 1
fi

echo "==> Creating chit user"
if ! id chit &>/dev/null; then
    useradd -r -m -s /bin/bash chit
    echo "Created chit user"
else
    echo "chit user already exists"
    # Ensure shell and home are correct
    usermod -s /bin/bash -d /home/chit chit
    mkdir -p /home/chit
    chown chit:chit /home/chit
fi

echo "==> Installing Go"
if ! command -v go &>/dev/null; then
    curl -fsSL https://go.dev/dl/go1.24.1.linux-amd64.tar.gz -o /tmp/go.tar.gz
    tar -C /usr/local -xzf /tmp/go.tar.gz
    rm /tmp/go.tar.gz
    ln -sf /usr/local/go/bin/go /usr/local/bin/go
    echo "Installed Go $(go version)"
else
    echo "Go already installed: $(go version)"
fi

echo "==> Installing Caddy"
if ! command -v caddy &>/dev/null; then
    apt-get install -y debian-keyring debian-archive-keyring apt-transport-https > /dev/null 2>&1
    curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
    curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | tee /etc/apt/sources.list.d/caddy-stable.list > /dev/null
    apt-get update > /dev/null 2>&1 && apt-get install -y caddy > /dev/null 2>&1
    echo "Installed Caddy"
else
    echo "Caddy already installed"
fi

echo "==> Checking Claude CLI"
if command -v claude &>/dev/null; then
    echo "Claude CLI installed: $(claude --version 2>&1 | head -1)"
else
    echo "Claude CLI not found — install it: curl -fsSL https://claude.ai/install.sh | bash"
fi

echo "==> Configuring git for chit user"
sudo -u chit git config --global user.name "Claude (chit)"
sudo -u chit git config --global user.email "claude@chit"
sudo -u chit git config --global --add safe.directory /opt/chit

echo "==> Building"
mkdir -p "$CHIT_DIR/.cache"
chown chit:chit "$CHIT_DIR/.cache"
GOPATH="$CHIT_DIR/.cache/go" GOCACHE="$CHIT_DIR/.cache/go-build" make build

echo "==> Cross-compiling client binaries for download"
mkdir -p "$CHIT_DIR/dist"
VERSION=$(git rev-parse --short HEAD 2>/dev/null || echo "dev")
LDFLAGS="-X main.version=$VERSION"
GOPATH="$CHIT_DIR/.cache/go" GOCACHE="$CHIT_DIR/.cache/go-build" \
    GOOS=darwin GOARCH=arm64 go build -ldflags "$LDFLAGS" -o dist/chit-darwin-arm64 ./cmd/client/
GOPATH="$CHIT_DIR/.cache/go" GOCACHE="$CHIT_DIR/.cache/go-build" \
    GOOS=darwin GOARCH=amd64 go build -ldflags "$LDFLAGS" -o dist/chit-darwin-amd64 ./cmd/client/
GOPATH="$CHIT_DIR/.cache/go" GOCACHE="$CHIT_DIR/.cache/go-build" \
    GOOS=linux GOARCH=amd64 go build -ldflags "$LDFLAGS" -o dist/chit-linux-amd64 ./cmd/client/

echo "==> Installing systemd units"
cp deploy/chit-server.service deploy/chit-bridge.service /etc/systemd/system/
systemctl daemon-reload

echo "==> Setting permissions"
chown -R chit:chit "$CHIT_DIR"

echo ""
echo "========================================="
echo "  Build complete. Next steps:"
echo "========================================="
echo ""

if [ ! -f "$CHIT_DIR/pb_data/data.db" ]; then
    echo "  1. Seed the database:"
    echo "     sudo -u chit bin/seed defaults"
    echo ""
    echo "  2. Create .bridge.env with the bot token from seed output:"
    echo "     sudo -u chit cp .bridge.env.example .bridge.env"
    echo "     vi .bridge.env   # paste the claude bot token"
    echo ""
    echo "  3. Generate invite codes:"
    echo "     sudo -u chit bin/seed invite 2"
    echo ""
fi

if [ ! -f /etc/caddy/Caddyfile ] || grep -q "yourteam" /etc/caddy/Caddyfile 2>/dev/null; then
    echo "  4. Set up Caddy (edit domain first):"
    echo "     vi deploy/Caddyfile"
    echo "     cp deploy/Caddyfile /etc/caddy/Caddyfile"
    echo "     systemctl restart caddy"
    echo ""
fi

echo "  5. Start services:"
echo "     systemctl enable --now chit-server chit-bridge"
echo ""
