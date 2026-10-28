package manager

import (
	"errors"
	"strconv"

	"mudgame/server/internal/exception"
	"mudgame/server/internal/model"
)

// maxEnhanceLevel 装备强化最大等级
const maxEnhanceLevel = 5

// EnhanceItem 消耗钱币与古铜，按物品ID记录强化等级；当前每个效果均+2/级。
// 最高5级，费用递增200/400/600/800/1000铜。预算调整等待用户选择。
func (r *RuntimeManager) EnhanceItem(playerID, itemID string) (string, error) {
	itemID = canonicalItemID(itemID)
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
	if p.State != "IDLE" {
		return "", errors.New("busy_state")
	}

	def, ok := fileItems[itemID]
	if !ok || def.EquipmentSlot == "" {
		return "", errors.New("item_not_equipment")
	}
	if p.Backpack[itemID] < 1 && p.Equipment[def.EquipmentSlot] != itemID {
		return "", errors.New("item_not_in_inventory")
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
		return "", errors.New("钱币不足——强化到" + strconv.Itoa(nextLevel) + "级需要" + strconv.Itoa(costSilver) + "文钱。")
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
		"（消耗" + strconv.Itoa(costSilver) + "文钱 + " + strconv.Itoa(costBronze) + "枚古铜残片）", nil
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
	for slot, itemID := range p.Equipment {
		if fileItems[itemID].EquipmentSlot != slot {
			continue
		}
		level := p.EnhancedItems[itemID]
		if level <= 0 {
			continue
		}
		bonus := getEnhanceBonus(itemID, level)
		for key, value := range bonus {
			applyFlatBonus(eff, key, value)
		}
	}
}

func enhancementInfo(p *model.Player, itemID string) map[string]interface{} {
	if fileItems[itemID].EquipmentSlot == "" {
		return nil
	}
	level := p.EnhancedItems[itemID]
	next := level + 1
	return map[string]interface{}{"level": level, "maxLevel": maxEnhanceLevel, "nextMoney": next * 200, "nextBronze": next, "ownedMoney": p.Money, "ownedBronze": p.Backpack["bronze_fragment"], "atMax": level >= maxEnhanceLevel, "nextBonus": getEnhanceBonus(itemID, 1)}
}
