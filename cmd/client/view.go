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

	botStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5"))
	titleStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")).Padding(0, 1)
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


	top := lipgloss.JoinHorizontal(lipgloss.Top, roomPanel, msgPanel)
	return top + "\n" + status + inputPanel
}

func (m model) viewInput() string {
	inputW := m.width - 4
	prompt := "> "
	if m.threadViewID != "" {
		prompt = editBarStyle.Render("[thread] ") + "> "
	}
	before := m.input[:m.cursor]
	after := m.input[m.cursor:]
	content := prompt + before + "\u2588" + after
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

// 12 distinct colors that look good on dark and light terminals
var nameColors = []string{
	"1", "2", "3", "4", "6", "9", "10", "11", "12", "13", "14", "208",
}

func handleColor(handle string) lipgloss.Style {
	var h uint
	for _, c := range handle {
		h = h*31 + uint(c)
	}
	color := nameColors[h%uint(len(nameColors))]
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(color))
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

	for _, dm := range m.display {
		msg := dm.msg
		handle := msg.Author
		isBot := false
		if msg.Expand.Author != nil {
			handle = msg.Expand.Author.Handle
			isBot = msg.Expand.Author.IsBot
		}

		ts := formatTimestamp(msg.Created)

		nameRendered := handleColor(handle).Render(handle)
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

		lines = append(lines, header, body)

		if !dm.isThread && dm.replyCount > 0 && m.threadViewID == "" {
			badge := fmt.Sprintf("[%d replies]", dm.replyCount)
			if dm.replyCount == 1 {
				badge = "[1 reply]"
			}
			lines = append(lines, "    "+threadBadgeStyle.Render(badge))
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
			m.display = append(m.display, displayMsg{msg: parent})
			for _, reply := range threads[m.threadViewID] {
				m.display = append(m.display, displayMsg{
					msg:      reply,
					isThread: true,
				})
			}
		}
		return
	}

	for _, msg := range topLevel {
		m.display = append(m.display, displayMsg{
			msg:        msg,
			replyCount: len(threads[msg.ID]),
		})
	}
}
