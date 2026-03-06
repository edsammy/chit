# Deploying Chit

## Prerequisites

- A VPS with Ubuntu/Debian
- A domain pointing at the VPS IP
- SSH access as root

## Deploy

SSH into the VPS as root:

```bash
# Create chit user with sudo
useradd -m -s /bin/bash chit
echo "chit ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/chit
echo "source ~/.bashrc" > /home/chit/.profile && chown chit:chit /home/chit/.profile
mkdir -p /opt/chit && chown chit:chit /opt/chit

# Switch to chit user
su - chit

# Install Claude CLI
curl -fsSL https://claude.ai/install.sh | bash

# Clone the repo
git clone https://github.com/edsammy/chit.git /opt/chit
cd /opt/chit

# Give Claude full permissions
mkdir -p ~/.claude
cp deploy/claude-settings.json ~/.claude/settings.json

# Let Claude do everything (use interactive mode so you can watch)
claude

# Then paste:
# Read deploy/INSTALL.md and follow every step. My domain is YOURDOMAIN.COM. Print invite codes at the end.
```

## Connect

On your local machine:

```bash
curl -fsSL https://yourdomain.com/install.sh | sh
chit
```

## Updating

Tell Claude in #claude to pull and rebuild, or:

```bash
su - chit
cd /opt/chit && claude -p "git pull, rebuild, restart services"
```

## Logs

```bash
journalctl -u chit-server -u chit-bridge -f
```
