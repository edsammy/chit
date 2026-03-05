#!/bin/bash
set -e

HOST=${1:?usage: deploy.sh <host>}
REMOTE_DIR=/opt/chit

echo "==> Cross-compiling for linux/amd64"
cd "$(dirname "$0")/.."
GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$(git rev-parse --short HEAD)" -o dist/chit-server ./cmd/server/
GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$(git rev-parse --short HEAD)" -o dist/chit-bridge ./cmd/bridge/
GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$(git rev-parse --short HEAD)" -o dist/seed ./cmd/seed/

echo "==> Uploading binaries"
ssh "$HOST" "mkdir -p $REMOTE_DIR/bin $REMOTE_DIR/pb_hooks"
scp dist/chit-server dist/chit-bridge dist/seed "$HOST:$REMOTE_DIR/bin/"
scp pb_hooks/claude_system_prompt.md "$HOST:$REMOTE_DIR/pb_hooks/"
scp .bridge.env.example "$HOST:$REMOTE_DIR/"

echo "==> Installing systemd units"
scp deploy/chit-server.service deploy/chit-bridge.service "$HOST:/tmp/"
ssh "$HOST" "sudo mv /tmp/chit-server.service /tmp/chit-bridge.service /etc/systemd/system/ && sudo systemctl daemon-reload"

echo "==> Restarting services"
ssh "$HOST" "sudo systemctl restart chit-server && sudo systemctl restart chit-bridge"

echo "==> Done"
ssh "$HOST" "sudo systemctl status chit-server chit-bridge --no-pager"
