# Style Guide

## Principles

- Simple over clever
- Readable over compact
- Delete code rather than comment it out
- No premature abstraction

## Go Conventions

**Files**: One file per concern. Keep files around 300 lines.

**Functions**: Do one thing. If you need "and" to describe it, split it.

**Errors**: Return errors, don't panic. Wrap with context: `fmt.Errorf("connecting to server: %w", err)`

**Naming**:
- `getRoomByID` not `fetchRoomFromDatabaseByID`
- `room` not `r` (except in tight loops)
- `err` is fine for errors

## Project Patterns

**Server**: PocketBase handles collections, API, and realtime. Custom logic goes in hooks.

**Client**: Bubble Tea model/update/view. Keep rendering in `View()`, logic in `Update()`.

**Bridge**: Thin daemon. SSE in, claude -p/-r out, POST response back. No business logic.

**No raw SQL**: All data flows through the PocketBase API. Never touch `pb_data/` directly.

## Comments

Only when the "why" isn't obvious. Never explain "what" the code does.

```go
// Good: explains why
// PocketBase SSE sends a newline keep-alive every 30s, reset timer on any data
timer.Reset(45 * time.Second)

// Bad: explains what
// Reset the timer to 45 seconds
timer.Reset(45 * time.Second)
```

## Dependencies

Add dependencies reluctantly. Justify in commit message.

Current deps:
- `pocketbase` - Server framework (SQLite + API + SSE in one binary)
- `bubbletea` - TUI framework
- `bubbles` - TUI components (viewport)
- `lipgloss` - TUI styling
