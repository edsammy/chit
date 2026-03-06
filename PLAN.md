# Chit

Team chat for terminal people. Shared AI teammate. Self-improving.

One VPS. One SQLite file. Claude as a cofounder.


## Why

Working in siloed Claude instances means no shared context. Decisions evaporate.
Chit gives the whole team — humans and Claude — a single persistent workspace.
Claude sees every conversation, remembers every decision, and can act on them.


## Architecture

```
Your laptop                          VPS (single box)
┌──────────┐                ┌──────────────────────────────────┐
│ chit TUI │───── HTTPS ───▶│  Caddy (TLS termination)         │
│ (local)  │◀───────────────│    │                              │
└──────────┘                │    ├─▶ chit-server (PocketBase)   │
                            │    │     SQLite, SSE, API         │
Your cofounder's laptop     │    │     serves client binaries   │
┌──────────┐                │    │                              │
│ chit TUI │───── HTTPS ───▶│                                   │
│ (local)  │◀───────────────│  chit-bridge (Go daemon)          │
└──────────┘                │    ├─ subscribes to SSE           │
                            │    ├─ spawns claude -p / -r       │
                            │    └─ posts responses to chat     │
                            │                                   │
                            │  claude CLI (Max account auth)    │
                            │    ├─ reads CLAUDE.md (context)   │
                            │    ├─ full repo access            │
                            │    └─ shell, git, build, deploy   │
                            └──────────────────────────────────┘
```

Clients run locally, connect to the VPS over HTTPS. No SSH required.


## Stack

- **Server**: PocketBase (Go + SQLite). Realtime SSE, hooks, API. One binary.
- **Client**: Go + Bubble Tea. Single binary, cross-platform.
- **Bridge**: Go daemon. Connects chat to Claude Code CLI.
- **AI**: Claude Code CLI (`claude -p` / `claude -r`), authed with Max subscription.
- **Hosting**: Single VPS (RackNerd), Caddy for reverse proxy + TLS.


## Access Model

TUI client runs on your local machine. Points at the remote server.

First run prompts for server URL, invite code, handle, and display name.
Config saved to `~/.config/chit/` (server, token). Subsequent runs skip straight to chat.

```
curl -fsSL https://chit.nance.app/install.sh | sh
chit
```


## Auth

Invite code claim flow. Admin generates codes (`make invite`), new user enters
code + handle + name on first run, gets a permanent Bearer token.

- Tokens stored in `~/.config/chit/token`
- All API requests carry `Authorization: Bearer <token>`
- Bot tokens generated at seed time, stored in `.bridge.env`
- PocketBase admin dashboard uses separate superuser auth (not Bearer tokens)


## Collections

### members

- handle (text, unique) — @handle for mentions
- name (text) — display name
- is_bot (bool) — true for Claude, GitHub bot
- token (text, unique) — API auth token

### rooms

- name (text, unique) — #general, #claude, #chit-dev
- topic (text) — current room topic

### messages

- room (relation -> rooms)
- author (relation -> members)
- body (text) — markdown
- parent (relation -> messages, optional) — thread parent

### invite_codes

- code (text, unique) — short random string
- used (bool) — flipped to true on claim
- claimed_by (relation -> members, optional)

### read_markers

- member (relation -> members)
- room (relation -> rooms)
- last_read (text) — last read message ID


## API

PocketBase CRUD + custom endpoints:

```
GET    /api/collections/rooms/records       — list rooms
GET    /api/collections/messages/records     — list messages (filter by room)
POST   /api/collections/messages/records     — send a message
GET    /api/realtime                         — SSE stream

POST   /api/auth/claim                      — exchange invite code for token
GET    /api/auth/me                         — current member info
GET    /api/version                         — server version
GET    /download/{platform}                 — client binary (e.g. darwin-arm64)
GET    /install.sh                          — curl-pipe-sh installer
```


## Claude Integration

### How it works

Claude is not an API endpoint — it's a full Claude Code agent running on the VPS.

The **bridge daemon** (`chit-bridge`) is the nervous system:

1. Subscribes to the chat SSE stream
2. When a message contains `@claude`, assembles context:
   - Recent channel history
   - System prompt (`pb_hooks/claude_system_prompt.md`)
   - Relevant thread history
3. Spawns `claude -p` (new conversation) or `claude -r` (resume thread)
4. Claude Code has full access: files, shell, git, build tools
5. Bridge streams Claude's response back to the chat with live dot animation

### Context & memory

Claude's memory lives in the repo, not a database table:

- **`CLAUDE.md`** — Company ethos, architecture decisions, team conventions.
  Claude reads this on every invocation and can update it.
- **Claude's auto-memory** (`~/.claude/memory/`) — Things Claude learns over time.
- **Chat history** — The bridge feeds recent messages as context per invocation.
- **Thread continuity** — `claude -r` resumes a prior session, preserving full
  conversation context within a thread.


## Company Docs

The team's shared knowledge lives in `docs/` in the repo and on GitHub — not a wiki,
not Notion, just markdown files version-controlled alongside the code.

- Markdown files in `docs/` at the root of the repo
- Full git history. GitHub is the reading UI and the backup.
- Claude reads them for context and edits them when decisions are made in chat
- No slash commands needed — just ask Claude in #claude ("update the roadmap",
  "write up that decision we just made") and it edits the file, commits, and pushes


## Self-Update (Meta-Coding)

Claude can modify chit itself, rebuild, and redeploy.

### Mechanics

- Claude edits code in the chit repo on the VPS
- `git commit && git push` to GitHub (backup + history)
- `make deploy` builds all binaries + cross-compiles client + restarts services
- Bridge daemon handles its own restart gracefully

### Guardrails

- **GitHub is the safety net.** Every change is committed and pushed.
  Worst case: `git revert`.
- **Claude does not touch `pb_data/` directly.** No raw SQL, no manual
  DB edits. All data flows through the PocketBase API.
- **No destructive git operations.** No force push, no reset --hard,
  no deleting remote branches.

### Client distribution

Server serves cross-compiled client binaries.

- Version is git commit short hash, baked in at compile time
- `make cross` builds darwin-arm64, darwin-amd64, linux-amd64 to `dist/`
- `curl -fsSL https://chit.nance.app/install.sh | sh` installs to `~/.local/bin`


## TUI

```
┌─ rooms ──────┬─ #claude ─────────────────────────────────┐
│              │                                            │
│ #general     │ eddie 10:32am                              │
│ #claude   *  │ @claude what's the best way to handle      │
│ #chit-dev    │ webhook auth in Go?                        │
│              │                                            │
│              │ claude 10:33am                [opus 4-6]   │
│              │ Compute HMAC-SHA256 of the request body    │
│              │ using your webhook secret and compare      │
│              │ to the X-Hub-Signature-256 header.         │
│              │                                            │
├──────────────┴────────────────────────────────────────────┤
│ > _                                                       │
└───────────────────────────────────────────────────────────┘
```

- Rooms panel on left, messages on right, input at bottom
- `*` unread indicator on rooms
- Esc to focus rooms, arrow keys to navigate, Enter to select
- Shift+Tab to quick-switch rooms
- DIY markdown rendering (bold, code, headings, tables, HR)
- Dot animation while Claude is thinking
- Model tag display in bot message headers


## Deployment

Bare VPS with systemd + Caddy. Clone repo, build on machine, Claude-driven install.

```
chit-server.service  — PocketBase on 127.0.0.1:8090
chit-bridge.service  — Bridge daemon (watches chat, spawns Claude)
Caddy                — TLS termination + reverse proxy
```

- Runs as `chit` user with sudo, bash, home dir
- Claude CLI installed as chit user (not root)
- `make deploy` = build + cross-compile + restart services
- `pb_data/data.db` is the entire platform. SQLite file, easy to back up.

See `deploy/INSTALL.md` for full setup instructions (designed for Claude to follow).


## Build Phases

### Phase 1 — Server ✓
- PocketBase with collections (members, rooms, messages, read_markers, invite_codes)
- Message CRUD, SSE subscriptions
- Seed script for users and rooms
- Auth middleware (Bearer tokens, invite code claim)
- Client binary download + install script

### Phase 2 — TUI client ✓
- Connect to server, pick a room, see messages, send messages
- Rooms panel with unread indicators
- DIY markdown rendering (tables, headings, bold, code, HR)
- Threads (view and reply via bridge)
- Shift+Tab quick channel switching
- First-run claim flow (invite code, handle, name)
- Server URL + token saved to ~/.config/chit/

### Phase 3 — Bridge + Claude ✓
- Bridge daemon: SSE listener -> context assembly -> claude -p -> post response
- Streaming output with live message updates and tool activity status
- #claude dedicated channel
- Session continuity via claude --resume
- Model tag as dedicated field, displayed in message headers
- CHIT_PROJECT_DIR for pointing Claude at target codebase
- #errors channel for bridge error logging (hidden from client)

### Phase 4 — Deploy ✓
- Caddy (TLS) + systemd on VPS
- Build on VPS, Claude-driven install via INSTALL.md
- Cross-compile client binaries for download
- `make deploy` for hot-swap rebuild + restart (sub-second downtime)

### Phase 5 — Client distribution ✓
- Server serves client binaries at /download/{platform} and /install.sh
- `curl -fsSL https://chit.nance.app/install.sh | sh` installs to ~/.local/bin

### Phase 6 — Client polish ✓
- Message selection mode (shift+up/down) with reply, edit, delete
- Deterministic handle-based username colors
- Unread indicators with read markers
- Thinking animation for bot responses
- Version display in room panel
- Markdown rendering (bold, italic, code, tables, headings)

### Phase 7 — GitHub integration
- Webhook receiver for push, PR, CI events
- GitHub bot user posts events to rooms

### Phase 8 — Future
- Search
- File sharing
- Reactions
- Room-level summaries (ask Claude)
