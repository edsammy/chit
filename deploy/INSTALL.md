# Chit VPS Install — Claude Instructions

Follow these steps exactly, in order. Run all commands as root.

## 1. Create chit user

```bash
useradd -r -m -s /bin/bash chit
mkdir -p /opt/chit
chown -R chit:chit /opt/chit
```

If the user already exists, make sure it has a bash shell and home dir:
```bash
usermod -s /bin/bash -d /home/chit chit
mkdir -p /home/chit
chown chit:chit /home/chit
```

## 2. Install Go

```bash
curl -fsSL https://go.dev/dl/go1.24.1.linux-amd64.tar.gz -o /tmp/go.tar.gz
tar -C /usr/local -xzf /tmp/go.tar.gz
rm /tmp/go.tar.gz
ln -sf /usr/local/go/bin/go /usr/local/bin/go
```

Skip if `go version` already works.

## 3. Install Caddy

```bash
apt-get install -y debian-keyring debian-archive-keyring apt-transport-https
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | tee /etc/apt/sources.list.d/caddy-stable.list
apt-get update && apt-get install -y caddy
```

Skip if `caddy version` already works.

## 4. Make Claude CLI available to chit user

```bash
ln -sf /root/.local/bin/claude /usr/local/bin/claude
```

## 5. Configure git for chit user

```bash
sudo -u chit git config --global user.name "Claude (chit)"
sudo -u chit git config --global user.email "claude@chit"
sudo -u chit git config --global --add safe.directory /opt/chit
```

## 6. Build

```bash
cd /opt/chit
mkdir -p .cache
chown chit:chit .cache
GOPATH=/opt/chit/.cache/go GOCACHE=/opt/chit/.cache/go-build make build
```

## 7. Cross-compile client binaries

```bash
VERSION=$(git rev-parse --short HEAD)
LDFLAGS="-X main.version=$VERSION"
mkdir -p dist
GOPATH=/opt/chit/.cache/go GOCACHE=/opt/chit/.cache/go-build GOOS=darwin GOARCH=arm64 go build -ldflags "$LDFLAGS" -o dist/chit-darwin-arm64 ./cmd/client/
GOPATH=/opt/chit/.cache/go GOCACHE=/opt/chit/.cache/go-build GOOS=darwin GOARCH=amd64 go build -ldflags "$LDFLAGS" -o dist/chit-darwin-amd64 ./cmd/client/
GOPATH=/opt/chit/.cache/go GOCACHE=/opt/chit/.cache/go-build GOOS=linux GOARCH=amd64 go build -ldflags "$LDFLAGS" -o dist/chit-linux-amd64 ./cmd/client/
```

## 8. Seed the database

```bash
sudo -u chit bin/seed defaults
```

Save the claude bot token from the output — you need it for the next step.

## 9. Create .bridge.env

```bash
sudo -u chit cp .bridge.env.example .bridge.env
```

Edit `.bridge.env` and set `CHIT_TOKEN` to the claude bot token from step 8.
Set `CHIT_PROJECT_DIR=.`

## 10. Generate invite codes

```bash
sudo -u chit bin/seed invite 2
```

Print the invite codes — the user needs them to connect.

## 11. Set up Caddy

Edit `deploy/Caddyfile` and replace `chat.yourteam.com` with the user's domain.
Then:

```bash
cp deploy/Caddyfile /etc/caddy/Caddyfile
systemctl restart caddy
```

## 12. Install systemd units and start

```bash
cp deploy/chit-server.service deploy/chit-bridge.service /etc/systemd/system/
systemctl daemon-reload
chown -R chit:chit /opt/chit
systemctl enable --now chit-server chit-bridge
```

## 13. Verify

```bash
systemctl status chit-server chit-bridge --no-pager
curl -s http://127.0.0.1:8090/api/version
```

Both services should be active. The version endpoint should return a JSON response.

Print a summary: services status, domain, and invite codes.
