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
	// CombatMoves 主动招式的私有视图集合：每项只定向发给对应玩家本人，
	// 不得混入公共房间payload（实现规范§5）
	CombatMoves []CombatMovesUpdate
}
