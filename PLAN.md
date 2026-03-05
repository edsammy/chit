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
Your cofounder's laptop     │    │                              │
┌──────────┐                │    └─▶ (static assets, later)     │
│ chit TUI │───── HTTPS ───▶│                                   │
│ (local)  │◀───────────────│  chit-bridge (Go daemon)          │
└──────────┘                │    ├─ subscribes to SSE           │
                            │    ├─ spawns claude -p / -r       │
                            │    ├─ posts responses to chat     │
                            │    └─ triggers deploys            │
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
- **Hosting**: Single VPS, Caddy for reverse proxy + TLS.


## Access Model

TUI client runs on your local machine. Points at the remote server.

```
CHIT_SERVER=https://chat.yourteam.com chit
```

No SSH into the VPS needed. The server exposes the PocketBase API over HTTPS
behind Caddy. Each team member runs their own `chit` binary locally.


## Auth

No auth for MVP. Username from environment variable. Small trusted team.

```sh
export CHIT_SERVER="https://chat.yourteam.com"
export CHIT_USER="jake"
```

Add API keys later when the team grows.


## Collections

### members

- handle (text, unique) — @handle for mentions
- name (text) — display name
- is_bot (bool) — true for Claude, GitHub bot

### rooms

- name (text, unique) — #general, #backend, #chit-dev
- topic (text) — current room topic

### messages

- room (relation → rooms)
- author (relation → members)
- body (text) — markdown
- parent (relation → messages, optional) — thread parent

### reactions

- message (relation → messages)
- user (relation → members)
- char (text) — one of: * + ! ? ~

### read_markers

- member (relation → members)
- room (relation → rooms)
- last_read (text) — last read message ID


## API

PocketBase gives you CRUD + realtime for free:

```
GET    /api/collections/rooms/records       — list rooms
GET    /api/collections/messages/records     — list messages (filter by room)
POST   /api/collections/messages/records     — send a message
PATCH  /api/collections/messages/records/:id — edit a message
DELETE /api/collections/messages/records/:id — delete a message
POST   /api/collections/reactions/records    — add a reaction
GET    /api/realtime                         — SSE stream
```


## Claude Integration

### How it works

Claude is not an API endpoint — it's a full Claude Code agent running on the VPS.

The **bridge daemon** (`chit-bridge`) is the nervous system:

1. Subscribes to the chat SSE stream
2. When a message contains `@claude`, assembles context:
   - Recent channel history
   - CLAUDE.md (company context, ethos, conventions)
   - Relevant thread history
3. Spawns `claude -p` (new conversation) or `claude -r` (resume thread)
4. Claude Code has full access: files, shell, git, build tools
5. Bridge posts Claude's response back to the chat

### Context & memory

Claude's memory lives in the repo, not a database table:

- **`CLAUDE.md`** — Company ethos, architecture decisions, team conventions.
  Claude reads this on every invocation. Claude can update it as decisions
  are made in chat. This is the living company doc.
- **Claude's auto-memory** (`~/.claude/memory/`) — Things Claude learns over time.
- **Chat history** — The bridge feeds recent messages as context per invocation.
- **Thread continuity** — `claude -r` resumes a prior session, preserving full
  conversation context within a thread.

### In-channel

```
sarah:   @claude what's the best way to handle webhook auth in Go?
claude:  Compute HMAC-SHA256 of the request body using your webhook
         secret and compare to the X-Hub-Signature-256 header...
```

### Planning

Multi-turn conversations with Claude in threads or dedicated rooms.
Claude remembers the full thread context via `-r` (resume).

```
jake:    @claude let's plan the billing integration
claude:  Sure. A few questions first — are you thinking Stripe or...
jake:    Stripe. Keep it simple, subscriptions only.
claude:  Got it. Here's what I'd propose: [detailed plan]
```


## Company Docs

The team's shared knowledge lives in `docs/` in the repo — not a wiki, not Notion,
just markdown files that Claude can read and edit like any other code.

### How it works

- Markdown files in `docs/` at the root of the repo
- Version controlled alongside the code. Full git history.
- Claude can read them for context and edit them when decisions are made
- Team members can read and update them from chat via slash commands

### Slash commands

- **`/doc <name>`** — reads a doc and renders it in chat. `/doc roadmap` shows `docs/roadmap.md`.
- **`/ethos`** — shortcut for `/doc ethos`. The company ethos doc gets its own command
  because it's referenced constantly.
- **`/decide`** — Claude summarizes the current discussion into a decision doc. Saved to
  `docs/decisions/` with a date prefix (e.g. `2026-03-04-pricing.md`). Captures the
  what, why, and alternatives considered.
- **`/update-doc <name>`** — tells Claude to update a doc based on what was just discussed.
  Claude reads the existing doc, reads recent chat context, and makes edits.

### Directory structure

```
docs/
├── ethos.md
├── roadmap.md
├── architecture.md
└── decisions/
    ├── 2026-03-04-pricing.md
    └── 2026-03-05-auth-approach.md
```

### Key insight

These aren't a separate wiki. They're files in the repo that Claude sees and edits
the same way it sees and edits code. When someone says "let's update the roadmap"
in chat, Claude opens `docs/roadmap.md`, makes the change, commits, and pushes.
No context switching, no stale wikis.


## Self-Update (Meta-Coding)

Claude can modify chit itself, rebuild, and redeploy.

### Flow

```
jake:    @claude add a /status command that shows who's online
claude:  On it.
         [reads codebase, edits files, runs tests]
claude:  Done — committed to main, rebuilding now.
         [go build, restarts services]
claude:  Live. Try /status.
```

### Mechanics

- Claude edits code in the chit repo on the VPS
- `git commit && git push` to GitHub (backup + history)
- Runs `go build` for affected binaries
- Restarts services via the deploy script
- Bridge daemon handles its own restart gracefully

### Guardrails

- **GitHub is the safety net.** Every change is committed and pushed.
  Worst case: `git revert`.
- **Claude does not touch `pb_data/` directly.** No raw SQL, no manual
  DB edits. All data flows through the PocketBase API.
- **No destructive git operations.** No force push, no reset --hard,
  no deleting remote branches.
- **Deploy script is simple and auditable.** Build, restart, done.

### Client updates

Server serves the latest client binaries. Clients know when they're outdated.

**Build side:**
- Version is the git commit short hash, baked in at compile time:
  `go build -ldflags "-X main.version=$(git rev-parse --short HEAD)"`
- Claude cross-compiles for darwin-arm64, darwin-amd64, linux-amd64
- Binaries placed in `dist/` on the server

**Server side:**
```
GET /api/version          → {"version": "a1b2c3d"}
GET /download/:os-:arch   → binary (e.g. /download/darwin-arm64)
```

**Client side:**
- On startup, client calls `/api/version` and compares to its own
- If outdated, shows indicator in the status bar: `chit a1b2c3d *`
- `chit update` downloads the new binary and replaces itself

**TUI status bar:**
```
┌─ rooms ──────┬─ #backend ──────────────── chit a1b2c3d * ┐
```
The `*` appears when a newer version is available on the server.
No auto-update — user runs `chit update` when ready.

### #chit-dev channel

Dedicated room for meta-coding discussions. Claude posts updates about
self-modifications here. Keeps the noise out of #general.


## Reactions (removed for MVP, add back later)

Stripped for simplicity. Server still has the `reactions` collection.
Characters when re-added: `*` (star), `+` (agree), `!` (important), `?` (confused), `~` (unsure)


## Threads

Reply to any message to start a thread. Threads show nested under the parent.

```
jake 10:32am
pushed the webhook handler, needs review
  ├─ sarah: looks good, one nit on line 42
  └─ jake: fixed
```


## GitHub Integration

Webhook receiver posts to rooms as a github bot user.

Events:
- push — "jake pushed 3 commits to feature/auth"
- pull_request — opened, merged, closed with title and diff stats
- check_run — CI pass/fail


## TUI

```
┌─ rooms ──────┬─ #backend ──────────────── chit a1b2c3d ┐
│              │                                          │
│ #general     │ jake 10:32am                             │
│ #backend  *  │ pushed the webhook handler               │
│ #frontend    │   [* 2] [+ 1]                            │
│ #chit-dev    │                                          │
│              │ sarah 10:35am                             │
│              │ @claude how do we validate webhook        │
│              │ signatures in go?                         │
│              │                                          │
│              │ claude 10:35am                            │
│              │ Compute HMAC-SHA256 of the request body   │
│              │ using your webhook secret and compare     │
│              │ to the X-Hub-Signature-256 header.        │
│              │                                          │
├──────────────┴──────────────────────────────────────────┤
│ > _                                                     │
└─────────────────────────────────────────────────────────┘
```

Rooms panel on the left. Messages on the right. Input at the bottom.
Markdown rendered via Glamour.


## Deployment

```yaml
# docker-compose.yml
services:
  chit-server:
    build: .
    volumes:
      - ./pb_data:/app/pb_data
    ports:
      - "8090:8090"

  caddy:
    image: caddy:2
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile
      - caddy_data:/data

volumes:
  caddy_data:
```

Or just systemd units on a bare VPS:

```
chit-server.service  — PocketBase
chit-bridge.service  — Bridge daemon (watches chat, spawns Claude)
```

Claude CLI is installed on the VPS, authed with Max subscription.
Bridge spawns it as needed — no long-running Claude process.

Backups: `pb_data/data.db` is the entire platform. Push to GitHub + periodic copy.


## Build Phases

### Phase 1 — Server skeleton ✓
- PocketBase with collections defined
- Basic message CRUD
- SSE subscriptions working
- Seed script to create users and rooms

### Phase 2 — TUI client ✓
- Connect to server, pick a room, see messages, send messages
- Rooms panel with unread indicators
- Markdown rendering (tables, headings, bold, code, HR)
- Threads (view and reply)
- Shift+Tab quick channel switching
- #errors hidden from sidebar, ⚠ indicator when unread

### Phase 3 — Bridge + Claude ✓
- Bridge daemon: SSE listener → context assembly → claude -p → post response
- Streaming output (stream-json --verbose) with live status updates
- #claude dedicated channel for all Claude interaction
- Session continuity via claude --resume
- CLAUDE.md as shared company context
- DIY markdown rendering (no glamour)
- Client-side dot animation for pending responses
- Model tag display ([opus 4-6]) in message headers
- CHIT_PROJECT_DIR for pointing Claude at target codebase
- #errors channel for bridge error visibility

### Phase 4 — Auth ✓
- Invite code claim flow (admin generates codes, users claim with handle + name)
- Per-member API tokens (Bearer auth on all requests)
- Token saved to ~/.config/chit/token
- First-run TUI prompts for invite code
- Auth middleware on server (401 for missing/invalid tokens)
- Bot tokens generated at seed time
- .bridge.env for bridge config

### Phase 5 — Deploy (next)
- Deploy: Caddy (TLS) + systemd on VPS, or Tailscale on Mac
- Cross-compile + deploy script

### Phase 6 — Self-update + client distribution
- Cross-compile client for darwin-arm64, darwin-amd64, linux-amd64
- Server serves client binaries at /download/:os-:arch
- `curl | sh` install script hosted at /install.sh
- Git commit hash baked into binaries as version
- Version check on client startup, `*` indicator when outdated
- `chit update` to self-replace with latest binary
- Claude can edit code, commit, push, rebuild, redeploy
- Guardrails: no touching pb_data, no destructive git ops

### Phase 7 — GitHub integration
- Webhook receiver for push, PR, CI
- GitHub bot user posts events to rooms

### Phase 8 — Polish
- Avatars (ASCII identicons)
- Search
- /summarize (room-level summaries)
- File sharing
- Mobile HTML client
