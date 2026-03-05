package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	server := envOr("CHIT_SERVER", "http://127.0.0.1:8090")
	user := envOr("CHIT_USER", "")

	if user == "" {
		fmt.Fprintln(os.Stderr, "set CHIT_USER to your handle")
		os.Exit(1)
	}

	api := NewAPI(server)
	me, err := api.FindMemberByHandle(user)
	if err != nil {
		log.Fatalf("could not find user %q: %v", user, err)
	}

	m := initialModel(api, me)
	p := tea.NewProgram(m, tea.WithAltScreen())

	subscribeSSE(server, p)

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// --- Tea messages ---

type roomsLoadedMsg struct{ rooms []Room }
type messagesLoadedMsg struct {
	messages  []Message
	reactions []Reaction
}
type messageSentMsg struct{}
type messageEditedMsg struct{}
type messageDeletedMsg struct{}
type reactionAddedMsg struct{}
type readMarkersLoadedMsg struct {
	markers map[string]string
	latest  map[string]string
}
type errMsg struct{ err error }
type dotTickMsg struct{}

// --- Model ---

type editMode int

const (
	modeNone   editMode = iota
	modeEdit            // editing existing message
	modeDelete          // confirming delete
	modeReply           // replying to a message
	modeReact           // picking a reaction char
)

type displayMsg struct {
	msg        Message
	isThread   bool
	replyCount int
	reactions  map[string]int
}

type model struct {
	api *API
	me  *Member

	rooms     []Room
	roomIdx   int
	messages  []Message
	reactions []Reaction
	display   []displayMsg
	input     string
	cursor    int
	err       error
	width     int
	height    int

	focusRooms bool
	viewport   viewport.Model
	ready bool

	msgIdx  int // selected display index (-1 = none)
	mode    editMode
	editID  string
	replyID string

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
		msgIdx:      -1,
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

// Height reserved for input bar and chrome around it.
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

// hasPendingDots checks if the last bot message is "...".
func (m *model) hasPendingDots() bool {
	if len(m.display) == 0 {
		return false
	}
	last := m.display[len(m.display)-1]
	if last.msg.Expand.Author != nil && last.msg.Expand.Author.IsBot {
		body := last.msg.Body
		return body == "." || body == ".." || body == "..."
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
		m.rooms = msg.rooms
		if len(m.rooms) > 0 {
			return m, tea.Batch(
				loadMessages(m.api, m.rooms[m.roomIdx].ID),
				loadReadMarkers(m.api, m.me.ID, m.rooms),
			)
		}
		return m, nil

	case readMarkersLoadedMsg:
		m.readMarkers = msg.markers
		m.latestMsgs = msg.latest
		return m, nil

	case messagesLoadedMsg:
		m.messages = msg.messages
		m.reactions = msg.reactions
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

	case messageSentMsg, messageEditedMsg, messageDeletedMsg, reactionAddedMsg:
		m.clearInput()
		m.mode = modeNone
		m.editID = ""
		m.replyID = ""
		m.msgIdx = -1
		if len(m.rooms) > 0 {
			return m, loadMessages(m.api, m.rooms[m.roomIdx].ID)
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

