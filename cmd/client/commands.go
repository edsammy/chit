package main

import tea "github.com/charmbracelet/bubbletea"

func loadRooms(api *API) tea.Cmd {
	return func() tea.Msg {
		rooms, err := api.ListRooms()
		if err != nil {
			return errMsg{err}
		}
		return roomsLoadedMsg{rooms}
	}
}

func loadNewestMessages(api *API, roomID string) tea.Cmd {
	return func() tea.Msg {
		msgs, totalPages, page, err := api.ListMessagesPaginated(roomID, 1, 200)
		if err != nil {
			return errMsg{err}
		}
		reverseMessages(msgs)
		return messagesPageLoadedMsg{messages: msgs, totalPages: totalPages, page: page}
	}
}

func loadOlderMessages(api *API, roomID string, page int) tea.Cmd {
	return func() tea.Msg {
		msgs, totalPages, pg, err := api.ListMessagesPaginated(roomID, page, 200)
		if err != nil {
			return errMsg{err}
		}
		reverseMessages(msgs)
		return olderMessagesLoadedMsg{messages: msgs, totalPages: totalPages, page: pg}
	}
}

func reverseMessages(msgs []Message) {
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
}

func sendMessage(api *API, roomID, authorID, body string) tea.Cmd {
	return func() tea.Msg {
		_, err := api.SendMessage(roomID, authorID, body, "")
		if err != nil {
			return errMsg{err}
		}
		return messageSentMsg{}
	}
}

func sendReply(api *API, roomID, authorID, body, parentID string) tea.Cmd {
	return func() tea.Msg {
		_, err := api.SendMessage(roomID, authorID, body, parentID)
		if err != nil {
			return errMsg{err}
		}
		return messageSentMsg{}
	}
}


func editMessage(api *API, id, body string) tea.Cmd {
	return func() tea.Msg {
		if err := api.EditMessage(id, body); err != nil {
			return errMsg{err}
		}
		return messageSentMsg{}
	}
}

func deleteMessage(api *API, id string) tea.Cmd {
	return func() tea.Msg {
		if err := api.DeleteMessage(id); err != nil {
			return errMsg{err}
		}
		return messageSentMsg{}
	}
}

func loadReadMarkers(api *API, memberID string) tea.Cmd {
	return func() tea.Msg {
		markers, err := api.GetReadMarkers(memberID)
		if err != nil {
			return errMsg{err}
		}
		markerMap := make(map[string]string)
		for _, marker := range markers {
			markerMap[marker.Room] = marker.LastRead
		}
		return readMarkersLoadedMsg{markers: markerMap}
	}
}

func loadLatestMsgs(api *API, rooms []Room) tea.Cmd {
	return func() tea.Msg {
		latest, err := api.LatestMessagePerRoom(rooms)
		if err != nil {
			return errMsg{err}
		}
		return latestMsgsLoadedMsg{latest: latest}
	}
}
