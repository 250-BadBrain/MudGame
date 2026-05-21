package engine

func Reconnect(repo RuntimeRepository, userIDHint, sessionUserID, token, playerID string) (ReconnectResult, error) {
	userID := firstNonEmpty(userIDHint, sessionUserID)
	if !repo.IsReconnectTokenValid(userID, token) {
		return ReconnectResult{}, errReconnectTokenInvalid
	}
	res := ReconnectResult{UserID: userID}
	if playerID == "" {
		return res, nil
	}
	if !repo.CharacterBelongsToUser(userID, playerID) {
		return ReconnectResult{}, errReconnectPlayerInvalid
	}
	res.PlayerID = playerID
	return res, nil
}
