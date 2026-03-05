package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if key == "ctrl+q" {
		return m, tea.Quit
	}

	if key == "esc" {
		return m.handleEsc()
	}

	if key == "shift+tab" && len(m.rooms) > 1 {
		for range m.rooms {
			m.roomIdx = (m.roomIdx + 1) % len(m.rooms)
			if m.rooms[m.roomIdx].Name != "errors" {
				break
			}
		}
		return m, loadMessages(m.api, m.rooms[m.roomIdx].ID)
	}

	if m.focusRooms {
		return m.handleRoomNav(key)
	}

	return m.handleTextInput(msg, key)
}

func (m model) handleEsc() (tea.Model, tea.Cmd) {
	if m.threadViewID != "" {
		m.threadViewID = ""
		m.buildDisplay()
		m.refreshViewport()
		return m, nil
	}
	if !m.focusRooms {
		m.focusRooms = true
		return m, nil
	}
	return m, nil
}

func (m model) handleRoomNav(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "down":
		for i := m.roomIdx + 1; i < len(m.rooms); i++ {
			if m.rooms[i].Name != "errors" {
				m.roomIdx = i
				return m, loadMessages(m.api, m.rooms[m.roomIdx].ID)
			}
		}
	case "up":
		for i := m.roomIdx - 1; i >= 0; i-- {
			if m.rooms[i].Name != "errors" {
				m.roomIdx = i
				return m, loadMessages(m.api, m.rooms[m.roomIdx].ID)
			}
		}
	case "enter":
		m.focusRooms = false
	}
	return m, nil
}

func (m model) handleTextInput(msg tea.KeyMsg, key string) (tea.Model, tea.Cmd) {
	switch key {
	case "enter":
		if strings.TrimSpace(m.input) == "" || len(m.rooms) == 0 {
			return m, nil
		}
		if m.threadViewID != "" {
			return m, sendReply(m.api, m.rooms[m.roomIdx].ID, m.me.ID, m.input, m.threadViewID)
		}
		return m, sendMessage(m.api, m.rooms[m.roomIdx].ID, m.me.ID, m.input)

	case "ctrl+a", "home":
		m.cursor = 0
	case "ctrl+e", "end":
		m.cursor = len(m.input)
	case "ctrl+f", "right":
		if m.cursor < len(m.input) {
			m.cursor++
		}
	case "ctrl+b", "left":
		if m.cursor > 0 {
			m.cursor--
		}
	case "backspace", "ctrl+h":
		if m.cursor > 0 {
			m.input = m.input[:m.cursor-1] + m.input[m.cursor:]
			m.cursor--
		}
	case "ctrl+d", "delete":
		if m.cursor < len(m.input) {
			m.input = m.input[:m.cursor] + m.input[m.cursor+1:]
		}
	case "ctrl+u":
		m.input = m.input[m.cursor:]
		m.cursor = 0
	case "ctrl+k":
		m.input = m.input[:m.cursor]
	case "ctrl+w":
		i := m.cursor - 1
		for i >= 0 && m.input[i] == ' ' {
			i--
		}
		for i >= 0 && m.input[i] != ' ' {
			i--
		}
		i++
		m.input = m.input[:i] + m.input[m.cursor:]
		m.cursor = i
	case "ctrl+c":
		m.clearInput()
	case "up", "down":
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	default:
		if msg.Type == tea.KeyRunes || key == " " {
			s := msg.String()
			m.input = m.input[:m.cursor] + s + m.input[m.cursor:]
			m.cursor += len(s)
		}
	}

	return m, nil
}
