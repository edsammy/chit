# Deploying Chit

## Prerequisites

- A VPS with Ubuntu/Debian
- A domain pointing at the VPS IP (e.g. `chat.yourteam.com`)
- SSH access to the VPS
- Go installed locally (for cross-compiling)

## 1. VPS setup

SSH into the VPS as root:

```bash
# Create chit user and directory
useradd -r -s /bin/false chit
mkdir -p /opt/chit
chown chit:chit /opt/chit

# Install Caddy
apt install -y debian-keyring debian-archive-keyring apt-transport-https
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | tee /etc/apt/sources.list.d/caddy-stable.list
apt update && apt install caddy

# Install Claude CLI (for the bridge)
curl -fsSL https://claude.ai/install.sh | bash
```

## 2. Deploy

From your local machine:

```bash
./deploy/deploy.sh root@your-vps-ip
```

This cross-compiles, uploads binaries, installs systemd units, and restarts services.
The first run will fail on restart (nothing seeded yet) — that's fine.

## 3. First run

SSH into the VPS as root:

```bash
cd /opt/chit

# Seed the database
sudo -u chit bin/seed defaults

# Create .bridge.env with the claude bot token from seed output
sudo -u chit cp .bridge.env.example .bridge.env
sudo -u chit vi .bridge.env  # paste the token

# Generate invite codes for your team
sudo -u chit bin/seed invite 5

# Set up Caddy (edit domain in Caddyfile first)
vi /opt/chit/deploy/Caddyfile
cp /opt/chit/deploy/Caddyfile /etc/caddy/Caddyfile
systemctl restart caddy

# Start everything
systemctl enable --now chit-server chit-bridge
```

## Connect

On your local machine:

```bash
CHIT_SERVER=https://chat.yourteam.com bin/chit
```

Enter your invite code when prompted. Token is saved to `~/.config/chit/token`.

Or install from the server:

```bash
curl -fsSL https://chat.yourteam.com/install.sh | sh
CHIT_SERVER=https://chat.yourteam.com chit
```

## Updating

From your local machine:

```bash
./deploy/deploy.sh root@your-vps-ip
```

## Logs

```bash
journalctl -u chit-server -f
journalctl -u chit-bridge -f
```
