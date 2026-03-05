package main

import (
	"fmt"
	"log"
	"os"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: seed <command> [args]")
		fmt.Println()
		fmt.Println("commands:")
		fmt.Println("  user <handle> <name>         — create a user")
		fmt.Println("  bot <handle> <name>          — create a bot user")
		fmt.Println("  room <name> [topic]          — create a room")
		fmt.Println("  defaults                     — create default rooms and claude bot")
		os.Exit(1)
	}

	app := pocketbase.New()

	if err := app.Bootstrap(); err != nil {
		log.Fatalf("bootstrap: %v", err)
	}

	cmd := os.Args[1]
	switch cmd {
	case "user":
		if len(os.Args) < 4 {
			log.Fatal("usage: seed user <handle> <name>")
		}
		createUser(app, os.Args[2], os.Args[3], false)
	case "bot":
		if len(os.Args) < 4 {
			log.Fatal("usage: seed bot <handle> <name>")
		}
		createUser(app, os.Args[2], os.Args[3], true)
	case "room":
		if len(os.Args) < 3 {
			log.Fatal("usage: seed room <name> [topic]")
		}
		topic := ""
		if len(os.Args) >= 4 {
			topic = os.Args[3]
		}
		createRoom(app, os.Args[2], topic)
	case "defaults":
		seedDefaults(app)
	default:
		log.Fatalf("unknown command: %s", cmd)
	}
}

func createUser(app *pocketbase.PocketBase, handle, name string, isBot bool) {
	col, err := app.FindCollectionByNameOrId("members")
	if err != nil {
		log.Fatalf("members collection not found (run the server first to create collections): %v", err)
	}

	existing, _ := app.FindFirstRecordByFilter(col, "handle = {:h}", map[string]any{"h": handle})
	if existing != nil {
		fmt.Printf("user %q already exists\n", handle)
		return
	}

	rec := core.NewRecord(col)
	rec.Set("handle", handle)
	rec.Set("name", name)
	rec.Set("is_bot", isBot)

	if err := app.Save(rec); err != nil {
		log.Fatalf("failed to create user: %v", err)
	}
	fmt.Printf("created user: @%s (%s)\n", handle, name)
}

func createRoom(app *pocketbase.PocketBase, name, topic string) {
	col, err := app.FindCollectionByNameOrId("rooms")
	if err != nil {
		log.Fatalf("rooms collection not found (run the server first to create collections): %v", err)
	}

	existing, _ := app.FindFirstRecordByFilter(col, "name = {:n}", map[string]any{"n": name})
	if existing != nil {
		fmt.Printf("room #%s already exists\n", name)
		return
	}

	rec := core.NewRecord(col)
	rec.Set("name", name)
	rec.Set("topic", topic)

	if err := app.Save(rec); err != nil {
		log.Fatalf("failed to create room: %v", err)
	}
	fmt.Printf("created room: #%s\n", name)
}

func seedDefaults(app *pocketbase.PocketBase) {
	createUser(app, "claude", "Claude", true)
	createUser(app, "github", "GitHub", true)
	createRoom(app, "general", "General discussion")
	createRoom(app, "claude", "Claude activity stream")
	createRoom(app, "errors", "Bridge errors and alerts")
	fmt.Println("defaults seeded")
}
