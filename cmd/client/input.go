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
		m.roomIdx = (m.roomIdx + 1) % len(m.rooms)
		m.msgIdx = -1
		m.confirmDelete = false
		return m, loadMessages(m.api, m.rooms[m.roomIdx].ID)
	}

	if key == "shift+left" {
		m.focusRooms = true
		return m, nil
	}

	if m.focusRooms {
		return m.handleRoomNav(key)
	}

	if m.msgIdx >= 0 {
		return m.handleMsgSelection(msg, key)
	}

	return m.handleTextInput(msg, key)
}

func (m model) handleMsgSelection(msg tea.KeyMsg, key string) (tea.Model, tea.Cmd) {
	if m.confirmDelete {
		if key == "y" {
			id := m.display[m.msgIdx].msg.ID
			m.confirmDelete = false
			m.msgIdx = -1
			return m, deleteMessage(m.api, id)
		}
		m.confirmDelete = false
		m.refreshViewport()
		return m, nil
	}
	switch key {
	case "shift+up":
		if m.msgIdx > 0 {
			m.msgIdx--
			m.refreshViewport()
		}
	case "shift+down":
		if m.msgIdx < len(m.display)-1 {
			m.msgIdx++
			m.refreshViewport()
		} else {
			m.msgIdx = -1
			m.refreshViewport()
		}
	case "up", "down":
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	case "enter", "t":
		if m.msgIdx < len(m.display) {
			dm := m.display[m.msgIdx]
			threadID := dm.msg.ID
			if dm.msg.Parent != "" {
				threadID = dm.msg.Parent
			}
			m.threadViewID = threadID
			m.msgIdx = -1
			m.buildDisplay()
			m.refreshViewport()
		}
	case "r":
		if m.msgIdx < len(m.display) {
			dm := m.display[m.msgIdx]
			parentID := dm.msg.ID
			if dm.msg.Parent != "" {
				parentID = dm.msg.Parent
			}
			m.replyToID = parentID
			m.replyToHandle = ""
			if dm.msg.Expand.Author != nil {
				m.replyToHandle = dm.msg.Expand.Author.Handle
			}
			m.msgIdx = -1
			m.refreshViewport()
		}
	case "e":
		if m.msgIdx < len(m.display) && m.display[m.msgIdx].msg.Author == m.me.ID {
			dm := m.display[m.msgIdx]
			m.editID = dm.msg.ID
			m.input = dm.msg.Body
			m.cursor = len(m.input)
			m.msgIdx = -1
			m.refreshViewport()
		}
	case "d":
		if m.msgIdx < len(m.display) && m.display[m.msgIdx].msg.Author == m.me.ID {
			m.confirmDelete = true
			m.refreshViewport()
		}
	case "esc":
		m.msgIdx = -1
		m.viewport.GotoBottom()
		m.refreshViewport()
	default:
		// any other key exits selection and goes back to input
		m.msgIdx = -1
		m.viewport.GotoBottom()
		m.refreshViewport()
		return m.handleTextInput(msg, key)
	}
	return m, nil
}

func (m model) handleEsc() (tea.Model, tea.Cmd) {
	if m.msgIdx >= 0 {
		m.msgIdx = -1
		m.refreshViewport()
		return m, nil
	}
	if m.editID != "" {
		m.editID = ""
		m.clearInput()
		return m, nil
	}
	if m.replyToID != "" {
		m.replyToID = ""
		m.replyToHandle = ""
		return m, nil
	}
	if m.threadViewID != "" {
		m.threadViewID = ""
		m.buildDisplay()
		m.refreshViewport()
		return m, nil
	}
	// nothing to cancel — jump to bottom
	m.viewport.GotoBottom()
	return m, nil
}

func (m model) handleRoomNav(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "down":
		if m.roomIdx < len(m.rooms)-1 {
			m.roomIdx++
			m.msgIdx = -1
			m.confirmDelete = false
			return m, loadMessages(m.api, m.rooms[m.roomIdx].ID)
		}
	case "up":
		if m.roomIdx > 0 {
			m.roomIdx--
			m.msgIdx = -1
			m.confirmDelete = false
			return m, loadMessages(m.api, m.rooms[m.roomIdx].ID)
		}
	case "enter", "shift+right":
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
		if m.editID != "" {
			id := m.editID
			m.editID = ""
			return m, editMessage(m.api, id, m.input)
		}
		parentID := m.replyToID
		if m.threadViewID != "" {
			parentID = m.threadViewID
		}
		if parentID != "" {
			m.replyToID = ""
			m.replyToHandle = ""
			return m, sendReply(m.api, m.rooms[m.roomIdx].ID, m.me.ID, m.input, parentID)
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
	case "shift+up":
		if len(m.display) > 0 {
			m.msgIdx = len(m.display) - 1
			m.refreshViewport()
		}
		return m, nil
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
