# Deploying Chit

## Prerequisites

- A VPS with Ubuntu/Debian
- A domain pointing at the VPS IP
- SSH access as root
- Claude CLI installed (`curl -fsSL https://claude.ai/install.sh | bash`)

## Deploy

SSH into the VPS as root:

```bash
# Set up Claude CLI permissions
mkdir -p ~/.claude
cat > ~/.claude/settings.json << 'EOF'
{"permissions":{"defaultMode":"bypassPermissions","deny":[]},"skipDangerousModePermissionPrompt":true}
EOF

# Clone the repo
git clone https://github.com/edsammy/chit.git /opt/chit
cd /opt/chit

# Let Claude do everything
claude -p "Read deploy/INSTALL.md and follow every step. My domain is YOURDOMAIN.COM. Print invite codes at the end."
```

That's it.

## Connect

On your local machine:

```bash
curl -fsSL https://yourdomain.com/install.sh | sh
chit
```

## Updating

Tell Claude in #claude to pull and rebuild, or on the VPS:

```bash
cd /opt/chit && claude -p "git pull, rebuild, restart services"
```

## Logs

```bash
journalctl -u chit-server -u chit-bridge -f
```
