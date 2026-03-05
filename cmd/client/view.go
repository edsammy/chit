package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	roomListStyle = lipgloss.NewStyle().
			Padding(1, 1).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8"))

	roomActiveStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	roomInactiveStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	unreadStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)

	inputStyle = lipgloss.NewStyle().
			Padding(0, 1).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("12"))

	authorStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	botStyle         = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5"))
	titleStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")).Padding(0, 1)
	selectStyle      = lipgloss.NewStyle().Background(lipgloss.Color("4")).Foreground(lipgloss.Color("15")).Bold(true)
	reactionStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	editBarStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)
	threadBadgeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Italic(true)
	hintStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	timeStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	modelStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))

	msgBorderStyle = lipgloss.NewStyle().
			Padding(0, 1).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8"))
)

func (m model) View() string {
	if m.width == 0 || !m.ready {
		return "loading..."
	}

	roomW := 20
	msgW := m.width - roomW - 4
	if msgW < 20 {
		msgW = 20
	}
	panelH := m.height - inputAreaH - 1

	roomPanel := m.viewRooms()
	roomPanel = roomListStyle.Width(roomW - 2).Height(panelH - 2).MaxHeight(panelH - 2).Render(roomPanel)

	var roomTitle string
	if len(m.rooms) > 0 {
		roomTitle = "#" + m.rooms[m.roomIdx].Name
		if topic := m.rooms[m.roomIdx].Topic; topic != "" {
			roomTitle += " — " + topic
		}
	}
	msgPanel := msgBorderStyle.
		Width(msgW - 2).
		Height(panelH - 2).
		MaxHeight(panelH - 2).
		Render(titleStyle.Render(roomTitle) + "\n" + m.viewport.View())

	inputPanel := m.viewInput()

	status := ""
	if m.err != nil {
		status = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("err: "+m.err.Error()) + "\n"
	}
	if m.msgIdx >= 0 && m.mode == modeNone {
		if m.threadViewID != "" {
			status = hintStyle.Render(" e:edit  d:delete  r:reply  s:react  esc:back") + "\n"
		} else {
			status = hintStyle.Render(" t:thread  e:edit  d:delete  r:reply  s:react  esc:back") + "\n"
		}
	}

	top := lipgloss.JoinHorizontal(lipgloss.Top, roomPanel, msgPanel)
	return top + "\n" + status + inputPanel
}

func (m model) viewInput() string {
	inputW := m.width - 4
	var content string

	switch m.mode {
	case modeDelete:
		content = editBarStyle.Render("delete this message? (y/n)")
	case modeReact:
		content = editBarStyle.Render("react: * (star)  + (agree)  ! (important)  ? (confused)  ~ (unsure)")
	default:
		prompt := "> "
		switch m.mode {
		case modeEdit:
			prompt = editBarStyle.Render("[edit] ") + "> "
		case modeReply:
			prompt = editBarStyle.Render("[reply] ") + "> "
		default:
			if m.threadViewID != "" {
				prompt = editBarStyle.Render("[thread] ") + "> "
			}
		}
		if m.focusRooms || m.msgIdx >= 0 {
			content = prompt + m.input
		} else {
			before := m.input[:m.cursor]
			after := m.input[m.cursor:]
			content = prompt + before + "\u2588" + after
		}
	}

	return inputStyle.Width(inputW).Render(content)
}

func (m model) viewRooms() string {
	var lines []string
	header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Render("rooms")
	lines = append(lines, header, "")

	for i, room := range m.rooms {
		name := "#" + room.Name
		marker := "  "

		lastRead := m.readMarkers[room.ID]
		latest := m.latestMsgs[room.ID]
		if latest != "" && latest != lastRead {
			marker = unreadStyle.Render("* ")
		}

		if i == m.roomIdx {
			if m.focusRooms {
				name = roomActiveStyle.Render("> " + name)
			} else {
				name = roomActiveStyle.Render("  " + name)
			}
		} else {
			name = roomInactiveStyle.Render(marker + name)
		}
		lines = append(lines, name)
	}

	return strings.Join(lines, "\n")
}

func isPendingDots(body string) bool {
	return body == "." || body == ".." || body == "..."
}

func (m model) renderMessages() string {
	if len(m.display) == 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("no messages yet")
	}

	var lines []string

	if m.threadViewID != "" {
		lines = append(lines, hintStyle.Render("── thread (esc to go back) ──"), "")
	}

	for i, dm := range m.display {
		msg := dm.msg
		handle := msg.Author
		isBot := false
		if msg.Expand.Author != nil {
			handle = msg.Expand.Author.Handle
			isBot = msg.Expand.Author.IsBot
		}

		ts := formatTimestamp(msg.Created)

		nameRendered := authorStyle.Render(handle)
		if isBot {
			nameRendered = botStyle.Render(handle)
		}

		body := msg.Body
		var modelTag string
		if isBot && strings.HasPrefix(body, "[") {
			if nl := strings.Index(body, "\n"); nl != -1 {
				tag := body[:nl]
				if strings.HasPrefix(tag, "[") && strings.HasSuffix(tag, "]") {
					modelTag = tag[1 : len(tag)-1]
					body = body[nl+1:]
				}
			}
		}

		header := nameRendered + " " + timeStyle.Render(ts)
		if modelTag != "" {
			header += " " + modelStyle.Render("["+modelTag+"]")
		}

		wrapW := m.viewport.Width - 4
		if wrapW > 0 {
			body = wordWrap(body, wrapW)
		}
		if isBot && isPendingDots(msg.Body) {
			body = strings.Repeat(".", m.dotCount)
		} else if isBot {
			body = renderMarkdown(body)
		}

		bodyLines := strings.Split(body, "\n")
		for j := range bodyLines {
			bodyLines[j] = "  " + bodyLines[j]
		}
		body = strings.Join(bodyLines, "\n")

		if i == m.msgIdx {
			header = selectStyle.Render(header)
			body = selectStyle.Render(body)
		}

		lines = append(lines, header, body)

		if !dm.isThread && dm.replyCount > 0 && m.threadViewID == "" {
			badge := fmt.Sprintf("[%d replies]", dm.replyCount)
			if dm.replyCount == 1 {
				badge = "[1 reply]"
			}
			lines = append(lines, "    "+threadBadgeStyle.Render(badge))
		}

		if len(dm.reactions) > 0 {
			var parts []string
			for _, ch := range []string{"*", "+", "!", "?", "~"} {
				if n, ok := dm.reactions[ch]; ok {
					parts = append(parts, reactionStyle.Render(fmt.Sprintf("[%s %d]", ch, n)))
				}
			}
			if len(parts) > 0 {
				lines = append(lines, "    "+strings.Join(parts, " "))
			}
		}

		lines = append(lines, "")
	}

	lines = append(lines, "", "", "")

	return strings.Join(lines, "\n")
}


func formatTimestamp(created string) string {
	if created == "" {
		return ""
	}
	t, err := time.Parse("2006-01-02 15:04:05.000Z", created)
	if err != nil {
		return ""
	}
	local := t.Local()
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterday := today.AddDate(0, 0, -1)
	msgDay := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location())

	switch {
	case msgDay.Equal(today):
		return local.Format("3:04pm")
	case msgDay.Equal(yesterday):
		return "yesterday " + local.Format("3:04pm")
	case now.Sub(local) < 7*24*time.Hour:
		return local.Format("Mon 3:04pm")
	default:
		return local.Format("Jan 2 3:04pm")
	}
}

func (m *model) buildDisplay() {
	reactionMap := make(map[string]map[string]int)
	for _, r := range m.reactions {
		if reactionMap[r.Message] == nil {
			reactionMap[r.Message] = make(map[string]int)
		}
		reactionMap[r.Message][r.Char]++
	}

	var topLevel []Message
	threads := make(map[string][]Message)
	msgByID := make(map[string]Message)
	for _, msg := range m.messages {
		msgByID[msg.ID] = msg
		if msg.Parent == "" {
			topLevel = append(topLevel, msg)
		} else {
			threads[msg.Parent] = append(threads[msg.Parent], msg)
		}
	}

	m.display = nil

	if m.threadViewID != "" {
		if parent, ok := msgByID[m.threadViewID]; ok {
			m.display = append(m.display, displayMsg{
				msg:       parent,
				reactions: reactionMap[parent.ID],
			})
			for _, reply := range threads[m.threadViewID] {
				m.display = append(m.display, displayMsg{
					msg:       reply,
					isThread:  true,
					reactions: reactionMap[reply.ID],
				})
			}
		}
		return
	}

	for _, msg := range topLevel {
		m.display = append(m.display, displayMsg{
			msg:        msg,
			replyCount: len(threads[msg.ID]),
			reactions:  reactionMap[msg.ID],
		})
	}
}
