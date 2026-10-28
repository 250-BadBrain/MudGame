package manager

import (
	"math/rand"
	engruntime "mudgame/server/internal/engine/runtime"
	"mudgame/server/internal/model"
)

const dungeonItemDropChance = .15

// Only a successful delivery consumes a lifetime claim. Both normal and quick
// completions pass through this helper, including fixed process rewards.
func grantDungeonItem(p *model.Player, itemID string, amount int) bool {
	def, exists := fileItems[itemID]
	if !exists || amount <= 0 {
		return false
	}
	if sid, ok := def.Effects["learnSkill"].(string); ok && (p.Skills[sid] > 0 || p.Backpack[itemID] > 0) {
		return false
	}
	if def.FirstClaim && p.Flags["item_claimed_"+itemID] {
		return false
	}
	if p.Backpack == nil {
		p.Backpack = map[string]int{}
	}
	p.Backpack[itemID] += amount
	if def.FirstClaim {
		if p.Flags == nil {
			p.Flags = map[string]bool{}
		}
		p.Flags["item_claimed_"+itemID] = true
	}
	return true
}

func applyDungeonSettlementReward(p *model.Player, s engruntime.Settlement) engruntime.Settlement {
	return applyDungeonSettlementRewardWithRoll(p, s, rand.Float64)
}

func applyDungeonSettlementRewardWithRoll(p *model.Player, s engruntime.Settlement, roll func() float64) engruntime.Settlement {
	if p == nil {
		return s
	}
	s.IsFirstClear = s.Progress >= 100 && !p.Flags["dungeon_cleared_"+s.DungeonID]
	if p.Flags == nil {
		p.Flags = map[string]bool{}
	}
	if s.Progress >= 100 {
		p.Flags["dungeon_cleared_"+s.DungeonID] = true
	}
	p.Attributes.Experience += s.RewardExperience
	p.Attributes.Potential += s.RewardPotential
	p.Money += s.RewardMoney
	received := map[string]int{}
	for itemID, amount := range s.RewardItems {
		def, exists := fileItems[itemID]
		if !exists || amount <= 0 {
			continue
		}
		if !def.QuestItem && def.Effects["learnSkill"] == nil && roll() >= dungeonItemDropChance {
			continue
		}
		if grantDungeonItem(p, itemID, amount) {
			received[itemID] += amount
		}
	}
	if !s.IsFirstClear && s.Progress >= 100 {
		if grantDungeonItem(p, "bronze_fragment", 2) {
			received["bronze_fragment"] += 2
		}
	}
	s.RewardItems = received
	for flag, value := range s.RewardFlags {
		p.Flags[flag] = value
	}
	if p.Reputations == nil {
		p.Reputations = map[string]int{}
	}
	for faction, value := range s.RewardReputations {
		p.Reputations[faction] += value
	}
	return s
}

func repairDungeonDropClaims(p *model.Player) bool {
	if p == nil || p.Flags["dungeon_drop_claim_repair_v1"] {
		return false
	}
	if p.Flags == nil {
		p.Flags = map[string]bool{}
	}
	for _, id := range []string{"liangdao_spear", "youlong_whip", "lieshi_mace", "suolang_hook", "jinggang_staff"} {
		owned := p.Backpack[id] > 0
		for _, equipped := range p.Equipment {
			if equipped == id {
				owned = true
			}
		}
		if !owned {
			delete(p.Flags, "item_claimed_"+id)
		}
	}
	p.Flags["dungeon_drop_claim_repair_v1"] = true
	return true
}
