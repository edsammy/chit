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
	p := tea.NewProgram(m, tea.WithAltScreen())

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
type messagesLoadedMsg struct {
	messages []Message
}
type messageSentMsg struct{}
type readMarkersLoadedMsg struct {
	markers map[string]string
	latest  map[string]string
}
type errMsg struct{ err error }
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

	threadViewID string

	readMarkers map[string]string
	latestMsgs  map[string]string

	dotCount  int
	dotActive bool
}

func (m *model) clearInput() {
	m.input = ""
	m.cursor = 0
}

func initialModel(api *API, me *Member) model {
	return model{
		api:         api,
		me:          me,
		readMarkers: make(map[string]string),
		latestMsgs:  make(map[string]string),
		dotCount:    1,
		viewport:    viewport.New(0, 0),
		ready:       true,
	}
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
	if atBottom {
		m.viewport.GotoBottom()
	}
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
			return m, tea.Batch(
				loadMessages(m.api, m.rooms[m.roomIdx].ID),
				loadReadMarkers(m.api, m.me.ID, m.rooms),
			)
		}
		return m, nil

	case readMarkersLoadedMsg:
		for roomID, serverRead := range msg.markers {
			if local, ok := m.readMarkers[roomID]; !ok || local < serverRead {
				m.readMarkers[roomID] = serverRead
			}
		}
		for roomID, serverLatest := range msg.latest {
			if local, ok := m.latestMsgs[roomID]; !ok || local < serverLatest {
				m.latestMsgs[roomID] = serverLatest
			}
		}
		return m, nil

	case messagesLoadedMsg:
		m.messages = msg.messages
		m.buildDisplay()
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
		if m.dotActive {
			cmd = dotTick()
		}
		return m, cmd

	case dotTickMsg:
		if !m.dotActive {
			return m, nil
		}
		m.dotCount = m.dotCount%3 + 1
		m.refreshViewport()
		return m, dotTick()

	case messageSentMsg:
		m.clearInput()
		if len(m.rooms) > 0 {
			roomID := m.rooms[m.roomIdx].ID
			return m, loadMessages(m.api, roomID)
		}
		return m, nil

	case sseEvent:
		if len(m.rooms) > 0 {
			return m, tea.Batch(
				loadMessages(m.api, m.rooms[m.roomIdx].ID),
				loadReadMarkers(m.api, m.me.ID, m.rooms),
			)
		}
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil
	}

	return m, nil
}

