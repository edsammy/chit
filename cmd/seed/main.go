package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: seed <command> [args]")
		fmt.Println()
		fmt.Println("commands:")
		fmt.Println("  user <handle> <name>         — create a human user")
		fmt.Println("  bot <handle> <name>          — create a bot user")
		fmt.Println("  room <name> [topic]          — create a room")
		fmt.Println("  invite [count]               — generate invite codes")
		fmt.Println("  defaults                     — create default rooms and users")
		os.Exit(1)
	}

	app := pocketbase.New()
	if err := app.Bootstrap(); err != nil {
		log.Fatalf("bootstrap: %v", err)
	}

	switch os.Args[1] {
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
	case "invite":
		count := 1
		if len(os.Args) >= 3 {
			n, err := strconv.Atoi(os.Args[2])
			if err != nil || n < 1 {
				log.Fatal("usage: seed invite [count]")
			}
			count = n
		}
		createInvites(app, count)
	case "defaults":
		seedDefaults(app)
	default:
		log.Fatalf("unknown command: %s", os.Args[1])
	}
}

func createUser(app *pocketbase.PocketBase, handle, name string, isBot bool) {
	col, err := app.FindCollectionByNameOrId("members")
	if err != nil {
		log.Fatalf("members collection not found (run the server first): %v", err)
	}

	existing, _ := app.FindFirstRecordByFilter(col, "handle = {:h}", map[string]any{"h": handle})
	if existing != nil {
		token := existing.GetString("token")
		if token == "" {
			token, err = generateToken()
			if err != nil {
				log.Fatalf("failed to generate token: %v", err)
			}
			existing.Set("token", token)
			if err := app.Save(existing); err != nil {
				log.Fatalf("failed to backfill token: %v", err)
			}
		}
		fmt.Printf("user @%s already exists", handle)
		if isBot {
			fmt.Printf("  token: %s", token)
		}
		fmt.Println()
		return
	}

	token, err := generateToken()
	if err != nil {
		log.Fatalf("failed to generate token: %v", err)
	}

	rec := core.NewRecord(col)
	rec.Set("handle", handle)
	rec.Set("name", name)
	rec.Set("is_bot", isBot)
	rec.Set("token", token)

	if err := app.Save(rec); err != nil {
		log.Fatalf("failed to create user: %v", err)
	}
	fmt.Printf("created user: @%s (%s)", handle, name)
	if isBot {
		fmt.Printf("  token: %s", token)
	}
	fmt.Println()
}

func createRoom(app *pocketbase.PocketBase, name, topic string) {
	col, err := app.FindCollectionByNameOrId("rooms")
	if err != nil {
		log.Fatalf("rooms collection not found (run the server first): %v", err)
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

const inviteAlphabet = "23456789abcdefghjkmnpqrstuvwxyz"

func createInvites(app *pocketbase.PocketBase, count int) {
	col, err := app.FindCollectionByNameOrId("invite_codes")
	if err != nil {
		log.Fatalf("invite_codes collection not found (run the server first): %v", err)
	}

	for range count {
		code := randomCode(6)
		rec := core.NewRecord(col)
		rec.Set("code", code)
		rec.Set("used", false)
		if err := app.Save(rec); err != nil {
			log.Fatalf("failed to create invite: %v", err)
		}
		fmt.Println(code)
	}
}

func randomCode(length int) string {
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(inviteAlphabet))))
		if err != nil {
			log.Fatalf("failed to generate random: %v", err)
		}
		b[i] = inviteAlphabet[n.Int64()]
	}
	return string(b)
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func seedDefaults(app *pocketbase.PocketBase) {
	createUser(app, "claude", "Claude", true)
	createUser(app, "github", "GitHub", true)
	createUser(app, "eddie", "Eddie", false)
	createUser(app, "milind", "Milind", false)
	createRoom(app, "general", "General discussion")
	createRoom(app, "claude", "Claude activity stream")
	createRoom(app, "errors", "Bridge errors and alerts")
	fmt.Println("defaults seeded")
}
