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

	if m.mode == modeDelete {
		if key == "y" {
			return m, deleteMessage(m.api, m.editID)
		}
		m.mode = modeNone
		m.editID = ""
		return m, nil
	}

	if m.mode == modeReact {
		if strings.ContainsAny(key, "*+!?~") && m.msgIdx >= 0 && m.msgIdx < len(m.display) {
			msgID := m.display[m.msgIdx].msg.ID
			return m, addReaction(m.api, msgID, m.me.ID, key)
		}
		m.mode = modeNone
		return m, nil
	}

	if m.focusRooms {
		return m.handleRoomNav(key)
	}

	if m.msgIdx >= 0 && m.mode == modeNone {
		return m.handleMsgSelection(key)
	}

	return m.handleTextInput(msg, key)
}

func (m model) handleEsc() (tea.Model, tea.Cmd) {
	if m.mode != modeNone {
		m.mode = modeNone
		m.editID = ""
		m.replyID = ""
		m.clearInput()
		return m, nil
	}
	if m.msgIdx >= 0 {
		m.msgIdx = -1
		return m, nil
	}
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
		if m.roomIdx < len(m.rooms)-1 {
			m.roomIdx++
			return m, loadMessages(m.api, m.rooms[m.roomIdx].ID)
		}
	case "up":
		if m.roomIdx > 0 {
			m.roomIdx--
			return m, loadMessages(m.api, m.rooms[m.roomIdx].ID)
		}
	case "enter":
		m.focusRooms = false
	}
	return m, nil
}

func (m model) handleMsgSelection(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "down":
		if m.msgIdx < len(m.display)-1 {
			m.msgIdx++
			m.refreshViewport()
		}
	case "up":
		if m.msgIdx > 0 {
			m.msgIdx--
			m.refreshViewport()
		}
	case "e":
		if m.msgIdx < len(m.display) && m.display[m.msgIdx].msg.Author == m.me.ID {
			dm := m.display[m.msgIdx]
			m.mode = modeEdit
			m.editID = dm.msg.ID
			m.input = dm.msg.Body
			m.cursor = len(m.input)
			m.msgIdx = -1
		}
	case "d":
		if m.msgIdx < len(m.display) && m.display[m.msgIdx].msg.Author == m.me.ID {
			m.mode = modeDelete
			m.editID = m.display[m.msgIdx].msg.ID
		}
	case "r":
		if m.msgIdx < len(m.display) {
			dm := m.display[m.msgIdx]
			parentID := dm.msg.ID
			if dm.msg.Parent != "" {
				parentID = dm.msg.Parent
			}
			m.mode = modeReply
			m.replyID = parentID
			m.clearInput()
			m.msgIdx = -1
		}
	case "s":
		if m.msgIdx < len(m.display) {
			m.mode = modeReact
		}
	case "t", "enter":
		if m.msgIdx < len(m.display) {
			dm := m.display[m.msgIdx]
			threadID := dm.msg.ID
			if dm.msg.Parent != "" {
				threadID = dm.msg.Parent
			}
			if m.threadViewID != "" {
				m.msgIdx = -1
			} else if dm.replyCount > 0 || dm.isThread {
				m.threadViewID = threadID
				m.msgIdx = -1
				m.buildDisplay()
				m.refreshViewport()
			} else {
				m.msgIdx = -1
			}
		}
	}
	return m, nil
}

func (m model) handleTextInput(msg tea.KeyMsg, key string) (tea.Model, tea.Cmd) {
	switch key {
	case "enter":
		if strings.TrimSpace(m.input) == "" || len(m.rooms) == 0 {
			return m, nil
		}
		if m.mode == modeEdit {
			return m, editMessage(m.api, m.editID, m.input)
		}
		if m.mode == modeReply {
			return m, sendReply(m.api, m.rooms[m.roomIdx].ID, m.me.ID, m.input, m.replyID)
		}
		if m.threadViewID != "" {
			return m, sendReply(m.api, m.rooms[m.roomIdx].ID, m.me.ID, m.input, m.threadViewID)
		}
		return m, sendMessage(m.api, m.rooms[m.roomIdx].ID, m.me.ID, m.input)

	case "ctrl+p":
		if len(m.display) > 0 {
			m.msgIdx = len(m.display) - 1
			m.refreshViewport()
			m.viewport.GotoBottom()
		}
		return m, nil

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
		if m.mode != modeNone {
			m.mode = modeNone
			m.editID = ""
			m.replyID = ""
		}
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
