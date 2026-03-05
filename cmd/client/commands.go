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

func loadMessages(api *API, roomID string) tea.Cmd {
	return func() tea.Msg {
		msgs, err := api.ListMessages(roomID)
		if err != nil {
			return errMsg{err}
		}
		reactions, err := api.ListReactionsForRoom(roomID)
		if err != nil {
			return errMsg{err}
		}
		return messagesLoadedMsg{messages: msgs, reactions: reactions}
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
		if err := api.UpdateMessage(id, body); err != nil {
			return errMsg{err}
		}
		return messageEditedMsg{}
	}
}

func deleteMessage(api *API, id string) tea.Cmd {
	return func() tea.Msg {
		if err := api.DeleteMessage(id); err != nil {
			return errMsg{err}
		}
		return messageDeletedMsg{}
	}
}

func addReaction(api *API, msgID, userID, char string) tea.Cmd {
	return func() tea.Msg {
		if err := api.AddReaction(msgID, userID, char); err != nil {
			return errMsg{err}
		}
		return reactionAddedMsg{}
	}
}

func loadReadMarkers(api *API, memberID string, rooms []Room) tea.Cmd {
	return func() tea.Msg {
		markers, err := api.GetReadMarkers(memberID)
		if err != nil {
			return errMsg{err}
		}
		markerMap := make(map[string]string)
		for _, marker := range markers {
			markerMap[marker.Room] = marker.LastRead
		}
		latest, err := api.LatestMessagePerRoom(rooms)
		if err != nil {
			return errMsg{err}
		}
		return readMarkersLoadedMsg{markers: markerMap, latest: latest}
	}
}
