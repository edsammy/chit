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

func subscribeSSE(base string, p *tea.Program) {
	go func() {
		backoff := time.Second
		for {
			err := listenSSE(base, p)
			_ = err
			time.Sleep(backoff)
			if backoff < 30*time.Second {
				backoff *= 2
			}
			p.Send(sseEvent{})
		}
	}()
}

func listenSSE(base string, p *tea.Program) error {
	resp, err := http.Get(base + "/api/realtime")
	if err != nil {
		return err
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
				go subscribe(base, clientID)
				continue
			}
		}

		if clientID != "" {
			p.Send(sseEvent{})
		}
	}
	return scanner.Err()
}

func subscribe(base, clientID string) {
	body := fmt.Sprintf(`{"clientId":"%s","subscriptions":["messages","reactions"]}`, clientID)
	http.Post(base+"/api/realtime", "application/json", strings.NewReader(body))
}
