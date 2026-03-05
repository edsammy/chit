# chit

Team chat for terminal people. PocketBase server, Bubble Tea TUI, Claude as a teammate.

One VPS. One SQLite file. One AI cofounder.

```
┌─ rooms ──────┬─ #general ──────────────────────────┐
│              │                                      │
│  #general    │ jake 10:32am                         │
│  #claude     │ pushed the webhook handler            │
│              │   [* 2] [+ 1]                        │
│              │                                      │
│              │ sarah 10:35am                         │
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
./chit-server serve --http 0.0.0.0:8090

# Seed default data (rooms, bot users)
./seed defaults

# Run the TUI
CHIT_USER=jake ./chit

# Run the Claude bridge (optional)
CHIT_PROJECT_DIR=~/your-project ./chit-bridge
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
- **chit** — Bubble Tea TUI. Rooms, threads, reactions, markdown.
- **chit-bridge** — Watches #claude, streams Claude Code responses back to chat.

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `CHIT_SERVER` | `http://127.0.0.1:8090` | Server URL |
| `CHIT_USER` | *(required)* | Your member handle |
| `CHIT_PROJECT_DIR` | `.` | Working directory for Claude |
| `CHIT_BRIDGE_USER` | `claude` | Bot handle for bridge |
| `CHIT_SYSTEM_PROMPT` | `pb_hooks/claude_system_prompt.md` | Claude system prompt file |
| `CHIT_MAX_TURNS` | `10` | Max Claude tool-use turns |

## TUI keybindings

| Key | Action |
|---|---|
| `Enter` | Send message |
| `Tab` | Toggle room/chat focus |
| `Up/Down` | Scroll messages |
| `Ctrl+P` | Select message (for edit/delete/reply/react) |
| `Esc` | Deselect / exit thread |
| `t` | Open thread (when message selected) |
| `e` | Edit message |
| `d` | Delete message |
| `r` | Reply to message |
| `s` | React to message |

## License

MIT
