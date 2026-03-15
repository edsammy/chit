package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

var version = "dev"

func main() {
	server := envOr("CHIT_SERVER", loadConfig("server"))
	token := envOr("CHIT_TOKEN", loadToken())

	if token == "" {
		var err error
		server, token, err = claimFlow(server)
		if err != nil {
			log.Fatalf("claim failed: %v", err)
		}
		saveConfig("server", server)
		saveToken(token)
	}

	if server == "" {
		server = "http://127.0.0.1:8090"
	}

	api := NewAPI(server, token)
	me, err := api.GetMe()
	if err != nil {
		log.Fatalf("auth failed (bad token?): %v", err)
	}

	m := initialModel(api, me)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithoutBracketedPaste())

	subscribeSSE(server, token, p)

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

func claimFlow(server string) (string, string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Welcome to chit! Enter your invite code to get started.")
	fmt.Println()

	if server == "" {
		fmt.Print("Server: ")
		s, _ := reader.ReadString('\n')
		server = strings.TrimSpace(s)
	}

	fmt.Print("Invite code: ")
	code, _ := reader.ReadString('\n')
	code = strings.TrimSpace(code)

	fmt.Print("Handle: ")
	handle, _ := reader.ReadString('\n')
	handle = strings.TrimSpace(handle)

	fmt.Print("Display name: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	api := NewAPI(server, "")
	token, member, err := api.ClaimInvite(code, handle, name)
	if err != nil {
		return "", "", err
	}

	fmt.Printf("\nWelcome, %s! You're in.\n", member.Handle)
	return server, token, nil
}

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "chit")
}

func loadToken() string {
	return loadConfig("token")
}

func saveToken(token string) {
	saveConfig("token", token)
}

func loadConfig(name string) string {
	data, err := os.ReadFile(filepath.Join(configDir(), name))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func saveConfig(name, value string) {
	dir := configDir()
	os.MkdirAll(dir, 0700)
	os.WriteFile(filepath.Join(dir, name), []byte(value+"\n"), 0600)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}


type roomsLoadedMsg struct{ rooms []Room }
type messagesPageLoadedMsg struct {
	messages   []Message
	totalPages int
	page       int
}
type olderMessagesLoadedMsg struct {
	messages   []Message
	totalPages int
	page       int
}
type messageSentMsg struct{}
type readMarkersLoadedMsg struct {
	markers map[string]string
}
type latestMsgsLoadedMsg struct {
	latest map[string]string
}
type errMsg struct{ err error }
type errClearMsg struct{}
type dotTickMsg struct{}


type displayMsg struct {
	msg        Message
	isThread   bool
	replyCount int
}

type model struct {
	api *API
	me  *Member

	rooms     []Room
	roomIdx   int
	messages  []Message
	display   []displayMsg
	input     string
	cursor    int
	err       error
	width     int
	height    int

	focusRooms bool
	viewport   viewport.Model
	ready bool

	msgIdx        int // -1 = no selection
	msgLines      []int // line offset of each display message
	confirmDelete bool
	replyToID     string
	replyToHandle string
	editID        string
	threadViewID  string

	readMarkers map[string]string
	latestMsgs  map[string]string

	snapToBottom   bool
	dotCount       int
	dotActive      bool
	currentPage    int
	totalPages     int
	loadingHistory bool
	allLoaded      bool
}

func (m *model) clearInput() {
	m.input = ""
	m.cursor = 0
}

func initialModel(api *API, me *Member) model {
	return model{
		api:         api,
		me:          me,
		msgIdx:      -1,
		readMarkers: make(map[string]string),
		latestMsgs:  make(map[string]string),
		dotCount:    1,
		viewport:    viewport.New(0, 0),
		ready:       true,
		currentPage: 1,
	}
}

func (m *model) resetPagination() {
	m.currentPage = 1
	m.totalPages = 0
	m.loadingHistory = false
	m.allLoaded = false
}

func (m model) Init() tea.Cmd {
	return loadRooms(m.api)
}

const inputAreaH = 3

func (m *model) resizeViewport() {
	roomW := 20
	msgW := m.width - roomW - 4
	if msgW < 20 {
		msgW = 20
	}
	vpH := m.height - inputAreaH - 4 // borders + title
	if vpH < 1 {
		vpH = 1
	}
	contentW := msgW - 4
	m.viewport.Width = contentW
	m.viewport.Height = vpH

}

func (m *model) refreshViewport() {
	atBottom := m.viewport.AtBottom()
	content := m.renderMessages()
	m.viewport.SetContent(content)
	if m.msgIdx >= 0 && m.msgIdx < len(m.msgLines) {
		m.scrollToMsg(m.msgIdx)
	} else if atBottom || m.snapToBottom {
		m.viewport.GotoBottom()
		m.snapToBottom = false
	}
}

func (m *model) scrollToMsg(idx int) {
	if idx < 0 || idx >= len(m.msgLines) {
		return
	}
	line := m.msgLines[idx]
	// put selected message in the top third of viewport
	target := line - m.viewport.Height/3
	if target < 0 {
		target = 0
	}
	m.viewport.SetYOffset(target)
}

func dotTick() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
		return dotTickMsg{}
	})
}

func (m *model) hasPendingDots() bool {
	if len(m.display) == 0 {
		return false
	}
	last := m.display[len(m.display)-1]
	if last.msg.Expand.Author != nil && last.msg.Expand.Author.IsBot {
		return isPendingDots(last.msg.Body)
	}
	return false
}

func (m *model) handleSSEMessage(msg sseMessageEvent) tea.Cmd {
	roomID := ""
	if len(m.rooms) > 0 {
		roomID = m.rooms[m.roomIdx].ID
	}
	switch msg.Action {
	case "create":
		m.latestMsgs[msg.Record.Room] = msg.Record.ID
		if msg.Record.Room == roomID {
			dup := false
			for _, existing := range m.messages {
				if existing.ID == msg.Record.ID {
					dup = true
					break
				}
			}
			if !dup {
				m.messages = append(m.messages, msg.Record)
				m.buildDisplay()
				m.refreshViewport()
				m.readMarkers[roomID] = msg.Record.ID
				m.latestMsgs[roomID] = msg.Record.ID
				go m.api.SetReadMarker(m.me.ID, roomID, msg.Record.ID)
			}
		}
	case "update":
		if msg.Record.Room == roomID {
			for i, existing := range m.messages {
				if existing.ID == msg.Record.ID {
					m.messages[i] = msg.Record
					break
				}
			}
			m.buildDisplay()
			m.refreshViewport()
		}
	case "delete":
		if msg.Record.Room == roomID {
			for i, existing := range m.messages {
				if existing.ID == msg.Record.ID {
					m.messages = append(m.messages[:i], m.messages[i+1:]...)
					break
				}
			}
			m.buildDisplay()
			m.refreshViewport()
		}
	}
	wasDotActive := m.dotActive
	m.dotActive = m.hasPendingDots()
	if m.dotActive && !wasDotActive {
		return dotTick()
	}
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeViewport()
		m.refreshViewport()
		return m, nil

	case roomsLoadedMsg:
		m.err = nil
		m.rooms = msg.rooms
		if len(m.rooms) > 0 {
			m.resetPagination()
			m.snapToBottom = true
			return m, tea.Batch(
				loadNewestMessages(m.api, m.rooms[m.roomIdx].ID),
				loadReadMarkers(m.api, m.me.ID),
				loadLatestMsgs(m.api, m.rooms),
			)
		}
		return m, nil

	case readMarkersLoadedMsg:
		for roomID, lastRead := range msg.markers {
			m.readMarkers[roomID] = lastRead
		}
		return m, nil

	case latestMsgsLoadedMsg:
		for roomID, latest := range msg.latest {
			m.latestMsgs[roomID] = latest
		}
		return m, nil

	case messagesPageLoadedMsg:
		m.messages = msg.messages
		m.totalPages = msg.totalPages
		m.currentPage = 2
		m.allLoaded = msg.totalPages <= 1
		m.loadingHistory = false
		m.buildDisplay()
		wasDotActive := m.dotActive
		m.dotActive = m.hasPendingDots()
		m.refreshViewport()
		if len(m.rooms) > 0 && len(m.messages) > 0 {
			roomID := m.rooms[m.roomIdx].ID
			lastID := m.messages[len(m.messages)-1].ID
			if m.readMarkers[roomID] != lastID {
				m.readMarkers[roomID] = lastID
				m.latestMsgs[roomID] = lastID
				go m.api.SetReadMarker(m.me.ID, roomID, lastID)
			}
		}
		var cmd tea.Cmd
		if m.dotActive && !wasDotActive {
			cmd = dotTick()
		}
		return m, cmd

	case olderMessagesLoadedMsg:
		oldLines := m.viewport.TotalLineCount()
		m.messages = append(msg.messages, m.messages...)
		m.currentPage = msg.page + 1
		m.allLoaded = msg.page >= msg.totalPages
		m.loadingHistory = false
		m.buildDisplay()
		content := m.renderMessages()
		m.viewport.SetContent(content)
		delta := m.viewport.TotalLineCount() - oldLines
		m.viewport.SetYOffset(m.viewport.YOffset + delta)
		return m, nil

	case dotTickMsg:
		if !m.dotActive {
			return m, nil
		}
		m.dotCount = m.dotCount%3 + 1
		m.refreshViewport()
		return m, dotTick()

	case messageSentMsg:
		m.clearInput()
		m.snapToBottom = true
		return m, nil

	case sseMessageEvent:
		return m, m.handleSSEMessage(msg)

	case sseEvent:
		if len(m.rooms) > 0 {
			return m, loadLatestMsgs(m.api, m.rooms)
		}
		return m, nil

	case errClearMsg:
		m.err = nil
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
			return errClearMsg{}
		})
	}

	return m, nil
}

