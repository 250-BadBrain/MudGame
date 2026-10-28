package engine

import "mudgame/server/internal/model"

// 主动招式工作流包装：身份之后的服务端权威入口（实现规范§5）。
// 客户端只能传moveId/battleId/expectedSequence，其余一律由服务端判定。

func GetCombatMovesView(repo RuntimeRepository, playerID string) (model.CombatMovesView, error) {
	return repo.GetCombatMoves(playerID)
}

func CastCombatMoveWithEffects(repo RuntimeRepository, playerID, moveID, battleID string, expectedSequence int64) (model.CombatMoveResult, error) {
	return repo.CastCombatMove(playerID, moveID, battleID, expectedSequence)
}

func ProcessCombatMovesTickWithEffects(repo RuntimeRepository) (model.CombatTickResult, error) {
	return repo.ProcessCombatMovesTick()
}
