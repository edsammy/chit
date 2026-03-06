# Deploying Chit

## Prerequisites

- A VPS with Ubuntu/Debian
- A domain pointing at the VPS IP
- SSH access as root

## Setup

SSH into the VPS as root:

```bash
# Create chit user
useradd -r -s /bin/false chit
mkdir -p /opt/chit
chown chit:chit /opt/chit

# Clone the repo
sudo -u chit git clone https://github.com/edsammy/chit.git /opt/chit
cd /opt/chit

# Run setup (installs Go, Caddy, Claude CLI, builds, installs systemd units)
bash deploy/setup.sh
```

## First run

After setup prints next steps:

```bash
# Seed the database (prints bot token)
sudo -u chit bin/seed defaults

# Create .bridge.env with the bot token
sudo -u chit cp .bridge.env.example .bridge.env
sudo -u chit vi .bridge.env

# Generate invite codes
sudo -u chit bin/seed invite 2

# Set up Caddy with your domain
vi deploy/Caddyfile
cp deploy/Caddyfile /etc/caddy/Caddyfile
systemctl restart caddy

# Start services
systemctl enable --now chit-server chit-bridge
```

## Connect

Install the client:

```bash
curl -fsSL https://yourdomain.com/install.sh | sh
CHIT_SERVER=https://yourdomain.com chit
```

Or build locally from the repo:

```bash
make client
CHIT_SERVER=https://yourdomain.com bin/chit
```

## Updating

On the VPS:

```bash
cd /opt/chit
git pull
make build
systemctl restart chit-server chit-bridge
```

Or just tell Claude in #claude to pull and rebuild.

## Logs

```bash
journalctl -u chit-server -f
journalctl -u chit-bridge -f
```
