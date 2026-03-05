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

type Room struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Topic string `json:"topic"`
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

func (a *API) GetMe() (*Member, error) {
	var member Member
	if err := a.get("/api/auth/me", &member); err != nil {
		return nil, err
	}
	return &member, nil
}

func (a *API) ClaimInvite(code, handle, name string) (string, *Member, error) {
	payload := map[string]string{"code": code, "handle": handle, "name": name}
	var resp struct {
		Token  string `json:"token"`
		Member Member `json:"member"`
	}
	if err := a.post("/api/auth/claim", payload, &resp); err != nil {
		return "", nil, err
	}
	return resp.Token, &resp.Member, nil
}

func (a *API) ListRooms() ([]Room, error) {
	var resp listResponse[Room]
	if err := a.get("/api/collections/rooms/records?sort=created", &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func (a *API) ListMessages(roomID string) ([]Message, error) {
	var resp listResponse[Message]
	v := url.Values{}
	v.Set("filter", fmt.Sprintf("room='%s'", roomID))
	v.Set("sort", "created")
	v.Set("expand", "author")
	v.Set("perPage", "200")
	if err := a.get("/api/collections/messages/records?"+v.Encode(), &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
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

type ReadMarker struct {
	ID       string `json:"id"`
	Member   string `json:"member"`
	Room     string `json:"room"`
	LastRead string `json:"last_read"`
}

func (a *API) GetReadMarkers(memberID string) ([]ReadMarker, error) {
	var resp listResponse[ReadMarker]
	v := url.Values{}
	v.Set("filter", fmt.Sprintf("member='%s'", memberID))
	v.Set("perPage", "100")
	if err := a.get("/api/collections/read_markers/records?"+v.Encode(), &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func (a *API) SetReadMarker(memberID, roomID, lastMsgID string) error {
	var resp listResponse[ReadMarker]
	v := url.Values{}
	v.Set("filter", fmt.Sprintf("member='%s' && room='%s'", memberID, roomID))
	if err := a.get("/api/collections/read_markers/records?"+v.Encode(), &resp); err != nil {
		return err
	}
	if len(resp.Items) > 0 {
		return a.patch("/api/collections/read_markers/records/"+resp.Items[0].ID,
			map[string]string{"last_read": lastMsgID})
	}
	return a.post("/api/collections/read_markers/records", map[string]string{
		"member":    memberID,
		"room":      roomID,
		"last_read": lastMsgID,
	}, nil)
}

func (a *API) LatestMessagePerRoom(rooms []Room) (map[string]string, error) {
	result := make(map[string]string)
	for _, room := range rooms {
		var resp listResponse[Message]
		v := url.Values{}
		v.Set("filter", fmt.Sprintf("room='%s'", room.ID))
		v.Set("sort", "-created")
		v.Set("perPage", "1")
		if err := a.get("/api/collections/messages/records?"+v.Encode(), &resp); err != nil {
			continue
		}
		if len(resp.Items) > 0 {
			result[room.ID] = resp.Items[0].ID
		}
	}
	return result, nil
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
