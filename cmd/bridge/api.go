package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type API struct {
	base   string
	token  string
	client *http.Client
}

func NewAPI(base, token string) *API {
	return &API{base: base, token: token, client: &http.Client{}}
}

func (a *API) GetMe() (*Member, error) {
	var member Member
	if err := a.get("/api/auth/me", &member); err != nil {
		return nil, err
	}
	return &member, nil
}

type Member struct {
	ID     string `json:"id"`
	Handle string `json:"handle"`
	Name   string `json:"name"`
	IsBot  bool   `json:"is_bot"`
}

type Message struct {
	ID      string `json:"id"`
	Room    string `json:"room"`
	Author  string `json:"author"`
	Body    string `json:"body"`
	Parent  string `json:"parent"`
	Created string `json:"created"`

	Expand struct {
		Author *Member `json:"author"`
	} `json:"expand"`
}

type listResponse[T any] struct {
	Items      []T `json:"items"`
	TotalItems int `json:"totalItems"`
}

type Room struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (a *API) FindRoomByName(name string) (*Room, error) {
	var resp listResponse[Room]
	v := url.Values{}
	v.Set("filter", fmt.Sprintf("name='%s'", name))
	if err := a.get("/api/collections/rooms/records?"+v.Encode(), &resp); err != nil {
		return nil, err
	}
	if len(resp.Items) == 0 {
		return nil, fmt.Errorf("room %q not found", name)
	}
	return &resp.Items[0], nil
}

func (a *API) FindMemberByHandle(handle string) (*Member, error) {
	var resp listResponse[Member]
	v := url.Values{}
	v.Set("filter", fmt.Sprintf("handle='%s'", handle))
	if err := a.get("/api/collections/members/records?"+v.Encode(), &resp); err != nil {
		return nil, err
	}
	if len(resp.Items) == 0 {
		return nil, fmt.Errorf("member %q not found", handle)
	}
	return &resp.Items[0], nil
}

func (a *API) ListRoomMessages(roomID string, limit int) ([]Message, error) {
	var resp listResponse[Message]
	v := url.Values{}
	v.Set("filter", fmt.Sprintf("room='%s' && parent=''", roomID))
	v.Set("sort", "-created")
	v.Set("expand", "author")
	v.Set("perPage", fmt.Sprintf("%d", limit))
	if err := a.get("/api/collections/messages/records?"+v.Encode(), &resp); err != nil {
		return nil, err
	}
	// Reverse so oldest is first.
	msgs := resp.Items
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

func (a *API) ListThreadMessages(parentID string) ([]Message, error) {
	var resp listResponse[Message]
	v := url.Values{}
	v.Set("filter", fmt.Sprintf("parent='%s'", parentID))
	v.Set("sort", "created")
	v.Set("expand", "author")
	v.Set("perPage", "100")
	if err := a.get("/api/collections/messages/records?"+v.Encode(), &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func (a *API) GetMessage(id string) (*Message, error) {
	var msg Message
	v := url.Values{}
	v.Set("expand", "author")
	if err := a.get("/api/collections/messages/records/"+id+"?"+v.Encode(), &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func (a *API) SendMessage(roomID, authorID, body, parent string) (*Message, error) {
	payload := map[string]string{
		"room":   roomID,
		"author": authorID,
		"body":   body,
	}
	if parent != "" {
		payload["parent"] = parent
	}
	var msg Message
	if err := a.post("/api/collections/messages/records", payload, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func (a *API) UpdateMessage(id, body string) error {
	payload := map[string]string{"body": body}
	return a.patch("/api/collections/messages/records/"+id, payload)
}

func (a *API) get(path string, out any) error {
	req, err := http.NewRequest("GET", a.base+path, nil)
	if err != nil {
		return err
	}
	if a.token != "" {
		req.Header.Set("Authorization", "Bearer "+a.token)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GET %s: %d %s", path, resp.StatusCode, string(body))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (a *API) post(path string, payload any, out any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", a.base+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if a.token != "" {
		req.Header.Set("Authorization", "Bearer "+a.token)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("POST %s: %d %s", path, resp.StatusCode, string(body))
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func (a *API) patch(path string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("PATCH", a.base+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if a.token != "" {
		req.Header.Set("Authorization", "Bearer "+a.token)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("PATCH %s: %d %s", path, resp.StatusCode, string(body))
	}
	return nil
}
