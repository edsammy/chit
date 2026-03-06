package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type sseEvent struct{}

func subscribeSSE(base, token string, p *tea.Program) {
	go func() {
		backoff := time.Second
		for {
			err := listenSSE(base, token, p)
			_ = err
			time.Sleep(backoff)
			if backoff < 30*time.Second {
				backoff *= 2
			}
			p.Send(sseEvent{})
		}
	}()
}

func listenSSE(base, token string, p *tea.Program) error {
	req, err := http.NewRequest("GET", base+"/api/realtime", nil)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 256*1024), 256*1024)
	var clientID string
	var lastSend time.Time

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
				go subscribe(base, token, clientID)
				continue
			}
		}

		if clientID != "" && time.Since(lastSend) >= 500*time.Millisecond {
			lastSend = time.Now()
			p.Send(sseEvent{})
		}
	}
	return scanner.Err()
}

func subscribe(base, token, clientID string) {
	body := fmt.Sprintf(`{"clientId":"%s","subscriptions":["messages"]}`, clientID)
	req, err := http.NewRequest("POST", base+"/api/realtime", strings.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	http.DefaultClient.Do(req)
}
