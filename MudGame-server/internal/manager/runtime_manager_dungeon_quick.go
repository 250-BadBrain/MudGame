package manager

import (
	"errors"
	engruntime "mudgame/server/internal/engine/runtime"
	"mudgame/server/internal/exception"
	"mudgame/server/internal/model"
	"strings"
)

func quickCompleteHint(p *model.Player, dungeonID string) string {
	if p == nil || !p.Flags["dungeon_cleared_"+dungeonID] {
		return "需先手动100%通关并结算本副本"
	}
	if p.State != "IDLE" {
		return "请先结束当前行为，恢复空闲后快速完成"
	}
	if _, exists := fileRooms[p.RoomID]; !exists || strings.HasPrefix(p.RoomID, "dg_") {
		return "请先返回公共世界"
	}
	return ""
}

func (r *RuntimeManager) QuickCompleteDungeon(playerID, dungeonID string, expectedSequence int) (map[string]interface{}, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	players, err := r.loadPlayers()
	if err != nil {
		return nil, err
	}
	p, ok := players[playerID]
	if !ok {
		return nil, exception.ErrPlayerNotFound
	}
	if r.dungeonMgr == nil {
		return nil, errors.New("dungeon_unavailable")
	}
	tpl, ok := r.dungeonMgr.DungeonTemplateNoLock(dungeonID)
	if !ok {
		return nil, errors.New("dungeon_not_found")
	}
	if hint := quickCompleteHint(p, dungeonID); hint != "" {
		return nil, errors.New(hint)
	}
	if expectedSequence < 0 || expectedSequence != p.QuickCompleteSequence || p.QuickCompleteSequence >= 9007199254740991 {
		return nil, errors.New("快速完成请求已过期，请刷新副本目录后重试")
	}
	// Build the same full-clear settlement without allocating rooms, NPCs or combat.
	inst := &engruntime.Instance{DungeonID: tpl.ID, Name: tpl.Name, Sequence: max(1, tpl.Order), ChainID: tpl.ChainID, Chapter: tpl.Chapter, Progress: 100, Rewards: tpl.Rewards}
	settlement := r.buildDungeonSettlementNoLock(inst)
	settlement.RewardMoney += tpl.QuickRewards.Money
	settlement = applyDungeonSettlementReward(p, settlement)
	for item, amount := range tpl.QuickRewards.Items {
		if grantDungeonItem(p, item, amount) {
			settlement.RewardItems[item] += amount
		}
	}
	p.QuickCompleteSequence++
	if err := r.savePlayers(players); err != nil {
		return nil, err
	}
	return dungeonSettlementPayload(settlement), nil
}

func dungeonSettlementPayload(s engruntime.Settlement) map[string]interface{} {
	names := map[string]string{}
	for id := range s.RewardItems {
		names[id] = fileItems[id].Name
	}
	return map[string]interface{}{"dungeonId": s.DungeonID, "dungeonName": s.DungeonName, "progress": s.Progress, "isFirstClear": s.IsFirstClear, "rewardExperience": s.RewardExperience, "rewardPotential": s.RewardPotential, "rewardMoney": s.RewardMoney, "rewardItems": s.RewardItems, "rewardItemNames": names, "rewardReputations": s.RewardReputations}
}
