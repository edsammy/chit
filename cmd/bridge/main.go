package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
)

func main() {
	// Prevent nested Claude Code session when bridge is launched from a Claude Code terminal.
	os.Unsetenv("CLAUDECODE")

	server := envOr("CHIT_SERVER", "http://127.0.0.1:8090")
	handle := envOr("CHIT_BRIDGE_USER", "claude")
	promptFile := envOr("CHIT_SYSTEM_PROMPT", "pb_hooks/claude_system_prompt.md")
	projectDir := envOr("CHIT_PROJECT_DIR", ".")
	maxTurns := envOrInt("CHIT_MAX_TURNS", 25)

	api := NewAPI(server)

	bot, err := api.FindMemberByHandle(handle)
	if err != nil {
		log.Fatalf("finding bot user %q: %v", handle, err)
	}
	log.Printf("bridge started as @%s (%s)", bot.Handle, bot.ID)

	claudeRoom, err := api.FindRoomByName("claude")
	if err != nil {
		log.Fatalf("finding #claude room: %v", err)
	}
	log.Printf("streaming to #claude (%s)", claudeRoom.ID)

	var errRoomID string
	errRoom, err := api.FindRoomByName("errors")
	if err != nil {
		log.Printf("warning: #errors room not found, errors will only be logged")
	} else {
		errRoomID = errRoom.ID
		log.Printf("errors will post to #errors (%s)", errRoomID)
	}

	systemPrompt, err := os.ReadFile(promptFile)
	if err != nil {
		log.Fatalf("reading system prompt %q: %v", promptFile, err)
	}

	handler := NewClaudeHandler(api, bot, claudeRoom.ID, errRoomID, string(systemPrompt), projectDir, maxTurns)
	log.Printf("claude working directory: %s", projectDir)

	log.Printf("connecting to SSE at %s", server)
	watchMessages(server, bot.ID, claudeRoom.ID, handler)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envOrInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %s=%q is not a number, using default %d\n", key, v, def)
		return def
	}
	return n
}
