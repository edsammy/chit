package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func main() {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: "pb_data"})

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		if err := ensureCollections(app); err != nil {
			log.Fatalf("failed to ensure collections: %v", err)
		}

		registerHooks(se, app)

		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

func ensureCollections(app *pocketbase.PocketBase) error {
	type colDef struct {
		name   string
		fields []map[string]any
	}

	defs := []colDef{
		{
			name: "members",
			fields: []map[string]any{
				{"name": "handle", "type": "text", "required": true},
				{"name": "name", "type": "text", "required": true},
				{"name": "is_bot", "type": "bool"},
				{"name": "token", "type": "text"},
			},
		},
		{
			name: "invite_codes",
			fields: []map[string]any{
				{"name": "code", "type": "text", "required": true},
				{"name": "used", "type": "bool"},
			},
		},
		{
			name: "rooms",
			fields: []map[string]any{
				{"name": "name", "type": "text", "required": true},
				{"name": "topic", "type": "text"},
			},
		},
		{
			name: "messages",
			fields: []map[string]any{
				{"name": "body", "type": "text", "required": true},
			},
		},
		{
			name: "reactions",
			fields: []map[string]any{
				{"name": "char", "type": "text", "required": true},
			},
		},
		{
			name: "read_markers",
			fields: []map[string]any{
				{"name": "last_read", "type": "text"},
			},
		},
		{
			name: "claude_threads",
			fields: []map[string]any{
				{"name": "title", "type": "text", "required": true},
				{"name": "messages", "type": "json"},
				{"name": "shared_at", "type": "date"},
			},
		},
	}

	for _, d := range defs {
		existing, _ := app.FindCollectionByNameOrId(d.name)
		if existing != nil {
			continue
		}

		col := core.NewBaseCollection(d.name)
		for _, f := range d.fields {
			if field := simpleField(f); field != nil {
				col.Fields.Add(field)
			}
		}

		col.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		col.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})

		switch d.name {
		case "members":
			col.AddIndex("idx_members_handle", true, "handle", "")
			col.AddIndex("idx_members_token", true, "token", "token != ''")
		case "rooms":
			col.AddIndex("idx_rooms_name", true, "name", "")
		case "invite_codes":
			col.AddIndex("idx_invite_codes_code", true, "code", "")
		}

		col.ListRule = strPtr("")
		col.ViewRule = strPtr("")
		col.CreateRule = strPtr("")
		col.UpdateRule = strPtr("")
		col.DeleteRule = strPtr("")

		if err := app.Save(col); err != nil {
			return fmt.Errorf("create %s: %w", d.name, err)
		}
		log.Printf("created collection: %s", d.name)
	}

	type relDef struct {
		collection string
		field      string
		target     string
		required   bool
	}

	rels := []relDef{
		{"messages", "room", "rooms", true},
		{"messages", "author", "members", true},
		{"messages", "parent", "messages", false},
		{"reactions", "message", "messages", true},
		{"reactions", "user", "members", true},
		{"read_markers", "member", "members", true},
		{"read_markers", "room", "rooms", true},
		{"invite_codes", "claimed_by", "members", false},
		{"claude_threads", "started_by", "members", true},
		{"claude_threads", "shared_to", "rooms", false},
	}

	for _, r := range rels {
		col, err := app.FindCollectionByNameOrId(r.collection)
		if err != nil {
			return fmt.Errorf("find %s: %w", r.collection, err)
		}

		if col.Fields.GetByName(r.field) != nil {
			continue
		}

		target, err := app.FindCollectionByNameOrId(r.target)
		if err != nil {
			return fmt.Errorf("find target %s: %w", r.target, err)
		}

		col.Fields.Add(&core.RelationField{
			Name:         r.field,
			Required:     r.required,
			CollectionId: target.Id,
			MaxSelect:    1,
		})

		if err := app.Save(col); err != nil {
			return fmt.Errorf("add relation %s.%s: %w", r.collection, r.field, err)
		}
		log.Printf("added relation: %s.%s -> %s", r.collection, r.field, r.target)
	}

	if err := migrateTokenField(app); err != nil {
		return fmt.Errorf("migrate token field: %w", err)
	}

	return nil
}

func migrateTokenField(app *pocketbase.PocketBase) error {
	col, err := app.FindCollectionByNameOrId("members")
	if err != nil {
		return nil
	}

	if col.Fields.GetByName("token") == nil {
		col.Fields.Add(&core.TextField{Name: "token"})
		col.AddIndex("idx_members_token", true, "token", "token != ''")
		if err := app.Save(col); err != nil {
			return fmt.Errorf("add token field: %w", err)
		}
		log.Printf("migrated: added token field to members")
	}

	records, err := app.FindAllRecords(col)
	if err != nil {
		return nil
	}
	for _, rec := range records {
		if rec.GetString("token") == "" {
			token, err := generateToken()
			if err != nil {
				return fmt.Errorf("generate token: %w", err)
			}
			rec.Set("token", token)
			if err := app.Save(rec); err != nil {
				return fmt.Errorf("backfill token for %s: %w", rec.GetString("handle"), err)
			}
			log.Printf("backfilled token for @%s", rec.GetString("handle"))
		}
	}
	return nil
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func simpleField(m map[string]any) core.Field {
	name := m["name"].(string)
	typ := m["type"].(string)
	required, _ := m["required"].(bool)

	switch typ {
	case "text":
		return &core.TextField{Name: name, Required: required}
	case "bool":
		return &core.BoolField{Name: name}
	case "json":
		return &core.JSONField{Name: name}
	case "date":
		return &core.DateField{Name: name}
	}
	return nil
}

func strPtr(s string) *string { return &s }

func registerHooks(se *core.ServeEvent, app *pocketbase.PocketBase) {
	registerAuth(se, app)
}
