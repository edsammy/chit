package main

import (
	"fmt"
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func main() {
	app := pocketbase.New()

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		if err := ensureCollections(app); err != nil {
			log.Fatalf("failed to ensure collections: %v", err)
		}

		registerHooks(se)

		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

func ensureCollections(app *pocketbase.PocketBase) error {
	// Pass 1: create collections with only non-relation fields.
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

		// Add created/updated autodate fields.
		col.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		col.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})

		switch d.name {
		case "members":
			col.AddIndex("idx_members_handle", true, "handle", "")
		case "rooms":
			col.AddIndex("idx_rooms_name", true, "name", "")
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

	// Pass 2: add relation fields to existing collections.
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
		{"claude_threads", "started_by", "members", true},
		{"claude_threads", "shared_to", "rooms", false},
	}

	for _, r := range rels {
		col, err := app.FindCollectionByNameOrId(r.collection)
		if err != nil {
			return fmt.Errorf("find %s: %w", r.collection, err)
		}

		// Skip if field already exists.
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

	return nil
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

func registerHooks(se *core.ServeEvent) {
	se.Router.POST("/api/hooks/claude", func(e *core.RequestEvent) error {
		return e.JSON(200, map[string]string{"status": "ok", "message": "claude hook placeholder"})
	})

	se.Router.POST("/api/hooks/claude/thread", func(e *core.RequestEvent) error {
		return e.JSON(200, map[string]string{"status": "ok", "message": "claude thread placeholder"})
	})

	se.Router.POST("/api/hooks/claude/share", func(e *core.RequestEvent) error {
		return e.JSON(200, map[string]string{"status": "ok", "message": "claude share placeholder"})
	})

	se.Router.POST("/api/hooks/github", func(e *core.RequestEvent) error {
		return e.JSON(200, map[string]string{"status": "ok", "message": "github hook placeholder"})
	})

}
