package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type sseRecord struct {
	Action string  `json:"action"`
	Record Message `json:"record"`
}

func watchMessages(base, botID, claudeRoomID string, handler *ClaudeHandler) {
	seen := struct {
		sync.Mutex
		ids map[string]bool
	}{ids: make(map[string]bool)}

	backoff := time.Second

	for {
		err := listenSSE(base, func(record sseRecord) {

			msg := record.Record
			if record.Action != "create" {
				return
			}
			if msg.Author == botID {
				return
			}
			if msg.Room != claudeRoomID {
				return
			}

			seen.Lock()
			if seen.ids[msg.ID] {
				seen.Unlock()
				return
			}
			seen.ids[msg.ID] = true
			seen.Unlock()

			log.Printf("message %s in #claude", msg.ID)
			go handler.Handle(msg)
		})

		if err != nil {
			log.Printf("SSE error: %v, reconnecting in %v", err, backoff)
			time.Sleep(backoff)
			if backoff < 30*time.Second {
				backoff *= 2
			}
		} else {
			backoff = time.Second
			log.Printf("SSE disconnected, reconnecting")
		}
	}
}

func listenSSE(base string, onMessage func(sseRecord)) error {
	resp, err := http.Get(base + "/api/realtime")
	if err != nil {
		return fmt.Errorf("connecting to SSE: %w", err)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 256*1024), 256*1024)
	var clientID string

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimPrefix(line, "data:")

		if clientID == "" {
			var connect struct {
				ClientID string `json:"clientId"`
			}
			if json.Unmarshal([]byte(data), &connect) == nil && connect.ClientID != "" {
				clientID = connect.ClientID
				log.Printf("SSE connected, subscribing (clientID=%s)", clientID)
				if err := subscribeSSE(base, clientID); err != nil {
					return fmt.Errorf("subscribing: %w", err)
				}
				continue
			}
		}

		if clientID != "" {
			var record sseRecord
			if json.Unmarshal([]byte(data), &record) == nil && record.Record.ID != "" {
				onMessage(record)
			}
		}
	}
	return scanner.Err()
}

func subscribeSSE(base, clientID string) error {
	body := fmt.Sprintf(`{"clientId":"%s","subscriptions":["messages"]}`, clientID)
	resp, err := http.Post(base+"/api/realtime", "application/json", strings.NewReader(body))
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("subscribe returned %d", resp.StatusCode)
	}
	return nil
}
