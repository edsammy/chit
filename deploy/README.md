# Deploying Chit

## Prerequisites

- A VPS with Ubuntu/Debian
- A domain pointing at the VPS IP
- SSH access as root

## Quick start

SSH into the VPS as root:

```bash
# Clone the repo
mkdir -p /opt/chit
git clone https://github.com/edsammy/chit.git /opt/chit
cd /opt/chit

# Set up Claude CLI permissions (so it can run everything)
mkdir -p ~/.claude
cp deploy/claude-settings.json ~/.claude/settings.json

# Run setup (creates chit user, installs Go/Caddy/Claude CLI, builds)
bash deploy/setup.sh
```

Then follow the printed next steps, or let Claude handle it:

```bash
claude -p "Read deploy/README.md. Run the 'First run' steps. \
  My domain is YOURDOMAIN.COM. \
  After seeding, put the claude bot token in .bridge.env. \
  Generate 2 invite codes and print them."
```

## First run

After setup.sh completes:

```bash
# Seed the database (prints bot token)
sudo -u chit bin/seed defaults

# Create .bridge.env with the bot token
sudo -u chit cp .bridge.env.example .bridge.env
vi .bridge.env   # paste the claude bot token

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
chit
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
bash deploy/setup.sh
systemctl restart chit-server chit-bridge
```

Or just tell Claude in #claude to pull and rebuild.

## Logs

```bash
journalctl -u chit-server -f
journalctl -u chit-bridge -f
```
