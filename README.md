# chit

Team chat for terminal people. PocketBase server, Bubble Tea TUI, Claude as a teammate.

One VPS. One SQLite file. One AI cofounder.

```
┌─ rooms ──────┬─ #general ──────────────────────────┐
│              │                                      │
│  #general    │ eddie 10:32am                        │
│  #claude     │ pushed the webhook handler            │
│              │                                      │
│              │ milind 10:35am                        │
│              │ nice, tests passing?                  │
│              │                                      │
├──────────────┴──────────────────────────────────────┤
│ > _                                                  │
└──────────────────────────────────────────────────────┘
```

## Quick start

```sh
# Build everything
make build

# Start the server
make run-server

# Seed default data (rooms, bot users)
make seed-defaults

# Generate invite codes
make invite

# Run the TUI
bin/chit

# Run the Claude bridge
make run-bridge
```

## Architecture

```
Your laptop                          VPS
┌──────────┐                ┌───────────────────────┐
│ chit TUI │── HTTPS/SSE ──▶│ chit-server            │
└──────────┘                │   (PocketBase+SQLite)  │
                            │                        │
                            │ chit-bridge             │
                            │   (SSE → Claude CLI)   │
                            └───────────────────────┘
```

- **chit-server** — PocketBase. API, SSE realtime, SQLite. One binary.
- **chit** — Bubble Tea TUI. Rooms, markdown.
- **chit-bridge** — Watches #claude, streams Claude Code responses back to chat.

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `CHIT_SERVER` | `http://127.0.0.1:8090` | Server URL |
| `CHIT_TOKEN` | *(from ~/.config/chit/token)* | API auth token |
| `CHIT_PROJECT_DIR` | `.` | Working directory for Claude |
| `CHIT_SYSTEM_PROMPT` | `pb_hooks/claude_system_prompt.md` | Claude system prompt file |
| `CHIT_MAX_TURNS` | `25` | Max Claude tool-use turns |

## TUI keybindings

| Key | Action |
|---|---|
| `Enter` | Send message / enter room |
| `Shift+Tab` | Next room |
| `Up/Down` | Navigate rooms (sidebar) / scroll messages (chat) |
| `Esc` | Switch to sidebar |
| `Ctrl+C` | Clear input |
| `Ctrl+Q` | Quit |
| `Ctrl+A/E` | Jump to start/end of input |
| `Ctrl+W` | Delete word |
| `Ctrl+U` | Delete to start |
| `Ctrl+K` | Delete to end |

## Deploy

See [deploy/README.md](deploy/README.md).

## Install (from a running server)

```sh
curl -fsSL https://chat.yourteam.com/install.sh | sh
CHIT_SERVER=https://chat.yourteam.com chit
```

## License

MIT
