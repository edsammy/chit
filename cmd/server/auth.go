package main

import (
	"encoding/json"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func registerAuth(se *core.ServeEvent, app *pocketbase.PocketBase) {
	se.Router.POST("/api/auth/claim", func(e *core.RequestEvent) error {
		var body struct {
			Code   string `json:"code"`
			Handle string `json:"handle"`
			Name   string `json:"name"`
		}
		if err := json.NewDecoder(e.Request.Body).Decode(&body); err != nil {
			return e.JSON(400, map[string]string{"error": "invalid request body"})
		}
		if body.Code == "" || body.Handle == "" || body.Name == "" {
			return e.JSON(400, map[string]string{"error": "code, handle, and name are required"})
		}

		inviteCol, err := app.FindCollectionByNameOrId("invite_codes")
		if err != nil {
			return e.JSON(500, map[string]string{"error": "invite_codes collection not found"})
		}
		invite, err := app.FindFirstRecordByFilter(inviteCol, "code = {:code} && used = false", map[string]any{"code": body.Code})
		if err != nil || invite == nil {
			return e.JSON(404, map[string]string{"error": "invalid or already used invite code"})
		}

		memberCol, err := app.FindCollectionByNameOrId("members")
		if err != nil {
			return e.JSON(500, map[string]string{"error": "members collection not found"})
		}
		existing, _ := app.FindFirstRecordByFilter(memberCol, "handle = {:h}", map[string]any{"h": body.Handle})
		if existing != nil {
			return e.JSON(409, map[string]string{"error": "handle already taken"})
		}

		token, err := generateToken()
		if err != nil {
			return e.JSON(500, map[string]string{"error": "failed to generate token"})
		}

		member := core.NewRecord(memberCol)
		member.Set("handle", body.Handle)
		member.Set("name", body.Name)
		member.Set("is_bot", false)
		member.Set("token", token)
		if err := app.Save(member); err != nil {
			return e.JSON(500, map[string]string{"error": "failed to create member"})
		}

		invite.Set("used", true)
		invite.Set("claimed_by", member.Id)
		app.Save(invite)

		return e.JSON(200, map[string]any{
			"token": token,
			"member": map[string]any{
				"id":     member.Id,
				"handle": body.Handle,
				"name":   body.Name,
			},
		})
	})

	se.Router.GET("/api/auth/me", func(e *core.RequestEvent) error {
		member, err := memberFromRequest(app, e)
		if err != nil {
			return e.JSON(401, map[string]string{"error": "unauthorized"})
		}
		return e.JSON(200, map[string]any{
			"id":     member.Id,
			"handle": member.GetString("handle"),
			"name":   member.GetString("name"),
			"is_bot": member.GetBool("is_bot"),
		})
	})

	se.Router.BindFunc(func(e *core.RequestEvent) error {
		path := e.Request.URL.Path

		if !strings.HasPrefix(path, "/api/") {
			return e.Next()
		}
		if path == "/api/auth/claim" {
			return e.Next()
		}
		if strings.HasPrefix(path, "/api/realtime") {
			return e.Next()
		}

		_, err := memberFromRequest(app, e)
		if err != nil {
			return e.JSON(401, map[string]string{"error": "unauthorized"})
		}
		return e.Next()
	})
}

func memberFromRequest(app *pocketbase.PocketBase, e *core.RequestEvent) (*core.Record, error) {
	auth := e.Request.Header.Get("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")
	if token == "" || token == auth {
		return nil, errUnauthorized
	}

	col, err := app.FindCollectionByNameOrId("members")
	if err != nil {
		return nil, err
	}
	return app.FindFirstRecordByFilter(col, "token = {:t}", map[string]any{"t": token})
}

var errUnauthorized = &unauthorizedError{}

type unauthorizedError struct{}

func (e *unauthorizedError) Error() string { return "unauthorized" }
