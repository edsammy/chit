package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type ClaudeHandler struct {
	ctx          context.Context
	api          *API
	bot          *Member
	claudeRoomID string
	errRoomID    string
	systemPrompt string
	projectDir   string
	maxTurns     int

	sem      chan struct{}
	sessions struct {
		sync.Mutex
		m map[string]string
	}
}

func NewClaudeHandler(ctx context.Context, api *API, bot *Member, claudeRoomID, errRoomID, systemPrompt, projectDir string, maxTurns int) *ClaudeHandler {
	h := &ClaudeHandler{
		ctx:          ctx,
		api:          api,
		bot:          bot,
		claudeRoomID: claudeRoomID,
		errRoomID:    errRoomID,
		systemPrompt: systemPrompt,
		projectDir:   projectDir,
		maxTurns:     maxTurns,
		sem:          make(chan struct{}, 2),
	}
	h.sessions.m = make(map[string]string)
	return h
}

func (h *ClaudeHandler) Handle(msg Message) {
	h.sem <- struct{}{}
	defer func() { <-h.sem }()

	prompt, err := h.buildPrompt(msg)
	if err != nil {
		log.Printf("building prompt for %s: %v", msg.ID, err)
		h.postError(err)
		return
	}

	statusMsg, err := h.api.SendMessage(h.claudeRoomID, h.bot.ID, "...", "")
	if err != nil {
		log.Printf("posting status message for %s: %v", msg.ID, err)
		return
	}

	var lastUpdate time.Time
	onUpdate := func(text, status string) {
		if time.Since(lastUpdate) < 500*time.Millisecond {
			return
		}
		lastUpdate = time.Now()
		body := text
		if status != "" {
			if body != "" {
				body += "\n\n"
			}
			body += "_" + status + "_"
		}
		if body == "" {
			body = "..."
		}
		if err := h.api.UpdateMessage(statusMsg.ID, body); err != nil {
			log.Printf("patching stream update: %v", err)
		}
	}

	var sessionID string
	threadKey := msg.Parent
	if threadKey == "" {
		threadKey = msg.ID
	}
	h.sessions.Lock()
	sessionID = h.sessions.m[threadKey]
	h.sessions.Unlock()

	result, newSessionID, model, err := h.invoke(prompt, sessionID, onUpdate)
	if err != nil && sessionID != "" {
		log.Printf("resume failed, retrying fresh: %v", err)
		result, newSessionID, model, err = h.invoke(prompt, "", onUpdate)
	}
	if err != nil {
		log.Printf("claude invocation failed for %s: %v", msg.ID, err)
		errBody := fmt.Sprintf("Sorry, I hit an error processing that request.\n\n```\n%s\n```", err)
		if updateErr := h.api.UpdateMessage(statusMsg.ID, errBody); updateErr != nil {
			log.Printf("patching error message: %v", updateErr)
		}
		h.postError(err)
		return
	}

	h.sessions.Lock()
	h.sessions.m[threadKey] = newSessionID
	h.sessions.Unlock()

	if model != "" {
		result = fmt.Sprintf("[%s]\n%s", shortModel(model), result)
	}

	if err := h.api.UpdateMessage(statusMsg.ID, result); err != nil {
		log.Printf("patching final response for %s: %v", msg.ID, err)
		return
	}
	log.Printf("responded to %s in #claude", msg.ID)
}

func (h *ClaudeHandler) buildPrompt(msg Message) (string, error) {
	msgs, err := h.api.ListRoomMessages(h.claudeRoomID, 30)
	if err != nil {
		return "", fmt.Errorf("fetching room messages: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("Chat history:\n\n")
	for _, m := range msgs {
		handle := m.Author
		if m.Expand.Author != nil {
			handle = m.Expand.Author.Handle
		}
		sb.WriteString(fmt.Sprintf("%s: %s\n", handle, m.Body))
	}
	sb.WriteString("\nRespond to the latest message.")
	return sb.String(), nil
}

type streamEvent struct {
	Type    string          `json:"type"`
	Subtype string          `json:"subtype"`
	Message json.RawMessage `json:"message"`

	Result    string `json:"result"`
	SessionID string `json:"session_id"`
	Model     string `json:"model"` // present on "system" init event
}

type streamMessage struct {
	Content []contentBlock `json:"content"`
}

type contentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

func (h *ClaudeHandler) invoke(prompt, sessionID string, onUpdate func(text, status string)) (string, string, string, error) {
	ctx, cancel := context.WithTimeout(h.ctx, 15*time.Minute)
	defer cancel()

	args := []string{
		"-p",
		"--output-format", "stream-json",
		"--verbose",
		"--max-turns", fmt.Sprintf("%d", h.maxTurns),
		"--append-system-prompt", h.systemPrompt,
	}
	if sessionID != "" {
		args = append(args, "--resume", sessionID)
	}
	args = append(args, prompt)

	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Dir = h.projectDir
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", "", "", fmt.Errorf("creating stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", "", "", fmt.Errorf("starting claude: %w", err)
	}

	var resultText string
	var resultSessionID string
	var resultModel string
	var streamedText string

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var event streamEvent
		if err := json.Unmarshal(line, &event); err != nil {
			continue
		}

		switch event.Type {
		case "system":
			if event.Model != "" {
				resultModel = event.Model
			}
		case "assistant":
			if event.Message != nil {
				var msg streamMessage
				if err := json.Unmarshal(event.Message, &msg); err != nil {
					continue
				}
				for _, block := range msg.Content {
					if block.Type == "text" && block.Text != "" {
						streamedText = strings.TrimSpace(block.Text)
						onUpdate(streamedText, "")
					} else if block.Type == "tool_use" {
						if status := formatToolActivity(block.Name, block.Input); status != "" {
							onUpdate(streamedText, status)
						}
					}
				}
			}
		case "result":
			resultText = strings.TrimSpace(event.Result)
			resultSessionID = event.SessionID
		}
	}

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", "", "", fmt.Errorf("claude exited %d: %s", exitErr.ExitCode(), string(exitErr.Stderr))
		}
		return "", "", "", fmt.Errorf("running claude: %w", err)
	}

	if resultText == "" {
		return "", "", "", fmt.Errorf("claude returned empty result")
	}
	return resultText, resultSessionID, resultModel, nil
}

func formatToolActivity(name string, input json.RawMessage) string {
	var params map[string]any
	json.Unmarshal(input, &params)

	switch name {
	case "Read":
		if path, ok := params["file_path"].(string); ok {
			return fmt.Sprintf("reading %s...", shortenPath(path))
		}
		return "reading file..."
	case "Grep":
		if pattern, ok := params["pattern"].(string); ok {
			return fmt.Sprintf("searching for %q...", pattern)
		}
		return "searching..."
	case "Glob":
		if pattern, ok := params["pattern"].(string); ok {
			return fmt.Sprintf("looking for %s...", pattern)
		}
		return "looking for files..."
	case "Edit":
		if path, ok := params["file_path"].(string); ok {
			return fmt.Sprintf("editing %s...", shortenPath(path))
		}
		return "editing file..."
	case "Write":
		if path, ok := params["file_path"].(string); ok {
			return fmt.Sprintf("writing %s...", shortenPath(path))
		}
		return "writing file..."
	case "Bash":
		if cmd, ok := params["command"].(string); ok {
			if len(cmd) > 60 {
				cmd = cmd[:60] + "..."
			}
			return fmt.Sprintf("running `%s`", cmd)
		}
		return "running command..."
	default:
		return ""
	}
}

func shortenPath(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) <= 3 {
		return path
	}
	return strings.Join(parts[len(parts)-3:], "/")
}

func shortModel(model string) string {
	model = strings.TrimPrefix(model, "claude-")
	if i := strings.LastIndex(model, "-20"); i != -1 {
		model = model[:i]
	}
	model = strings.Replace(model, "-", " ", 1)
	return model
}

func (h *ClaudeHandler) postError(origErr error) {
	body := fmt.Sprintf("```\n%s\n```", origErr)
	if h.errRoomID != "" {
		if _, err := h.api.SendMessage(h.errRoomID, h.bot.ID, body, ""); err != nil {
			log.Printf("failed to post to #errors: %v", err)
		}
	}
}
