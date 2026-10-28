package manager

import (
	"errors"
	"math"
	"strconv"

	"mudgame/server/internal/exception"
	"mudgame/server/internal/model"
)

// maxEnhanceLevel 装备强化最大等级
const maxEnhanceLevel = 5

// EnhanceItem 按物品ID记录强化，使用分属性系数，最高5级。
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
	if currentLevel < 0 {
		return "", errors.New("强化等级异常")
	}
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

// getEnhanceBonus is shared by effective stats and the next-level preview.
func getEnhanceBonus(itemID string, level int) map[string]float64 {
	def, ok := fileItems[itemID]
	if !ok || level <= 0 {
		return nil
	}
	bonus := map[string]float64{}
	for key, value := range def.Effects {
		if step, supported := model.EnhancementStep(key); supported && toFloatEffect(value) > 0 {
			bonus[key] = math.Round(step*float64(min(level, maxEnhanceLevel))*1000) / 1000
		}
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
			applyNumericBonus(eff, key, value)
		}
	}
}

func enhancementInfo(p *model.Player, itemID string) map[string]interface{} {
	if fileItems[itemID].EquipmentSlot == "" {
		return nil
	}
	level := p.EnhancedItems[itemID]
	next := level + 1
	return map[string]interface{}{"level": level, "maxLevel": maxEnhanceLevel, "nextMoney": next * 200, "nextBronze": next, "ownedMoney": p.Money, "ownedBronze": p.Backpack["bronze_fragment"], "atMax": level >= maxEnhanceLevel, "nextBonus": getEnhanceBonus(itemID, 1), "canReset": level > 0 && p.Flags[enhancementResetFlag(itemID)], "resetMoney": 100 * level * (level + 1), "resetBronze": level * (level + 1) / 2}
}
