package manager

import (
	"errors"
	"strconv"

	"mudgame/server/internal/exception"
	"mudgame/server/internal/model"
)

// maxEnhanceLevel 装备强化最大等级
const maxEnhanceLevel = 5

// EnhanceItem 强化装备：消耗银两 + 古铜残片，永久提升装备主属性。
// 每级强化 +2 主属性，最高 5 级。费用递增：200/400/600/800/1000 两。
func (r *RuntimeManager) EnhanceItem(playerID, itemID string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return "", err
	}
	p, ok := players[playerID]
	if !ok {
		return "", exception.ErrPlayerNotFound
	}
	if p.State == "DEAD" {
		return "", errors.New("dead_state")
	}

	def, ok := fileItems[itemID]
	if !ok || def.EquipmentSlot == "" {
		return "", errors.New("item_not_equipment")
	}

	if p.EnhancedItems == nil {
		p.EnhancedItems = map[string]int{}
	}
	currentLevel := p.EnhancedItems[itemID]
	if currentLevel >= maxEnhanceLevel {
		return "", errors.New(def.Name + "已强化至最高等级（" + strconv.Itoa(maxEnhanceLevel) + "级）。")
	}

	nextLevel := currentLevel + 1
	costSilver := nextLevel * 200
costBronze := nextLevel

	if p.Money < costSilver {
		return "", errors.New("银两不足——强化到" + strconv.Itoa(nextLevel) + "级需要" + strconv.Itoa(costSilver) + "两。")
	}
	if p.Backpack["bronze_fragment"] < costBronze {
		return "", errors.New("古铜残片不足——需要" + strconv.Itoa(costBronze) + "枚。")
	}

	p.Money -= costSilver
	p.Backpack["bronze_fragment"] -= costBronze
	if p.Backpack["bronze_fragment"] <= 0 {
		delete(p.Backpack, "bronze_fragment")
	}
	p.EnhancedItems[itemID] = nextLevel

	if err := r.savePlayers(players); err != nil {
		return "", err
	}
	return def.Name + "强化成功！当前等级：" + strconv.Itoa(nextLevel) + "/" + strconv.Itoa(maxEnhanceLevel) +
		"（消耗" + strconv.Itoa(costSilver) + "两 + " + strconv.Itoa(costBronze) + "枚古铜残片）", nil
}

// getEnhanceBonus 计算装备强化附加属性（主属性 +2/级）。
func getEnhanceBonus(itemID string, level int) map[string]int {
	def, ok := fileItems[itemID]
	if !ok || level <= 0 {
		return nil
	}
	bonus := map[string]int{}
	for key := range def.Effects {
		bonus[key] = level * 2
	}
	return bonus
}

// applyEnhancementNoLock 将装备强化加成叠加到生效属性中。
func applyEnhancementNoLock(eff *model.EffectiveAttributes, p *model.Player) {
	if p == nil || len(p.EnhancedItems) == 0 {
		return
	}
	for itemID, level := range p.EnhancedItems {
		if level <= 0 {
			continue
		}
		bonus := getEnhanceBonus(itemID, level)
		for key, value := range bonus {
			applyFlatBonus(eff, key, value)
		}
	}
}

