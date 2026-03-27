package model

type RoomMessage struct {
	RoomID  string
	Message string
}

type PlayerMessage struct {
	PlayerID string
	Message  string
}

type CombatTickResult struct {
	Messages       []RoomMessage
	PlayerMessages []PlayerMessage
	UpdatedRoomIDs []string
}
