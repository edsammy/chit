#!/bin/bash
set -e

HOST=${1:?usage: deploy.sh <host>}
REMOTE_DIR=/opt/chit
VERSION=$(git rev-parse --short HEAD)
LDFLAGS="-X main.version=$VERSION"

cd "$(dirname "$0")/.."

echo "==> Cross-compiling server + bridge + seed (linux/amd64)"
GOOS=linux GOARCH=amd64 go build -ldflags "$LDFLAGS" -o dist/chit-server ./cmd/server/
GOOS=linux GOARCH=amd64 go build -ldflags "$LDFLAGS" -o dist/chit-bridge ./cmd/bridge/
GOOS=linux GOARCH=amd64 go build -ldflags "$LDFLAGS" -o dist/seed ./cmd/seed/

echo "==> Cross-compiling client (darwin-arm64, darwin-amd64, linux-amd64)"
GOOS=darwin GOARCH=arm64 go build -ldflags "$LDFLAGS" -o dist/chit-darwin-arm64 ./cmd/client/
GOOS=darwin GOARCH=amd64 go build -ldflags "$LDFLAGS" -o dist/chit-darwin-amd64 ./cmd/client/
GOOS=linux  GOARCH=amd64 go build -ldflags "$LDFLAGS" -o dist/chit-linux-amd64  ./cmd/client/

echo "==> Uploading binaries"
ssh "$HOST" "mkdir -p $REMOTE_DIR/bin $REMOTE_DIR/dist $REMOTE_DIR/pb_hooks"
scp dist/chit-server dist/chit-bridge dist/seed "$HOST:$REMOTE_DIR/bin/"
scp dist/chit-darwin-arm64 dist/chit-darwin-amd64 dist/chit-linux-amd64 "$HOST:$REMOTE_DIR/dist/"
scp pb_hooks/claude_system_prompt.md "$HOST:$REMOTE_DIR/pb_hooks/"
scp .bridge.env.example "$HOST:$REMOTE_DIR/"

echo "==> Installing systemd units"
scp deploy/chit-server.service deploy/chit-bridge.service "$HOST:/tmp/"
ssh "$HOST" "sudo mv /tmp/chit-server.service /tmp/chit-bridge.service /etc/systemd/system/ && sudo systemctl daemon-reload"

echo "==> Restarting services"
ssh "$HOST" "sudo systemctl restart chit-server && sudo systemctl restart chit-bridge"

echo "==> Done"
ssh "$HOST" "sudo systemctl status chit-server chit-bridge --no-pager"
