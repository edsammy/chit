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
	client *http.Client
}

func NewAPI(base string) *API {
	return &API{base: base, client: &http.Client{}}
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

type Reaction struct {
	ID      string `json:"id"`
	Message string `json:"message"`
	User    string `json:"user"`
	Char    string `json:"char"`

	Expand struct {
		User *Member `json:"user"`
	} `json:"expand"`
}

type listResponse[T any] struct {
	Items      []T `json:"items"`
	TotalItems int `json:"totalItems"`
}

func (a *API) ListRooms() ([]Room, error) {
	var resp listResponse[Room]
	if err := a.get("/api/collections/rooms/records?sort=name", &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
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

func (a *API) ListReactionsForRoom(roomID string) ([]Reaction, error) {
	var resp listResponse[Reaction]
	v := url.Values{}
	v.Set("expand", "user")
	v.Set("perPage", "500")
	if err := a.get("/api/collections/reactions/records?"+v.Encode(), &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func (a *API) AddReaction(messageID, userID, char string) error {
	payload := map[string]string{
		"message": messageID,
		"user":    userID,
		"char":    char,
	}
	return a.post("/api/collections/reactions/records", payload, nil)
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
	for _, r := range rooms {
		var resp listResponse[Message]
		v := url.Values{}
		v.Set("filter", fmt.Sprintf("room='%s'", r.ID))
		v.Set("sort", "-created")
		v.Set("perPage", "1")
		if err := a.get("/api/collections/messages/records?"+v.Encode(), &resp); err != nil {
			continue
		}
		if len(resp.Items) > 0 {
			result[r.ID] = resp.Items[0].ID
		}
	}
	return result, nil
}

func (a *API) UpdateMessage(id, body string) error {
	payload := map[string]string{"body": body}
	return a.patch("/api/collections/messages/records/"+id, payload)
}

func (a *API) DeleteMessage(id string) error {
	return a.del("/api/collections/messages/records/" + id)
}

func (a *API) get(path string, out any) error {
	resp, err := a.client.Get(a.base + path)
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
	resp, err := a.client.Post(a.base+path, "application/json", bytes.NewReader(data))
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

func (a *API) del(path string) error {
	req, err := http.NewRequest("DELETE", a.base+path, nil)
	if err != nil {
		return err
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("DELETE %s: %d %s", path, resp.StatusCode, string(body))
	}
	return nil
}
