package manager

import (
	"errors"
	"fmt"
	"mudgame/server/internal/exception"
	"mudgame/server/internal/model"
	"strconv"
)

const enhancementRulesFlag = "enhancement_coefficients_v2"

func enhancementResetFlag(itemID string) string { return "enhancement_reset_available_" + itemID }

// Mark legacy investments only once; never refund until the player opts in.
func prepareEnhancementReset(p *model.Player) (bool, error) {
	if p.Flags[enhancementRulesFlag] {
		return false, nil
	}
	if p.Flags == nil {
		p.Flags = map[string]bool{}
	}
	for id, level := range p.EnhancedItems {
		if level < 0 || level > maxEnhanceLevel || fileItems[id].EquipmentSlot == "" {
			return false, fmt.Errorf("invalid legacy enhancement: %s/%d", id, level)
		}
		if level > 0 {
			p.Flags[enhancementResetFlag(id)] = true
		}
	}
	p.Flags[enhancementRulesFlag] = true
	return true, nil
}

func (r *RuntimeManager) ResetEnhancement(playerID, itemID string) (string, error) {
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
	level := p.EnhancedItems[itemID]
	if !p.Flags[enhancementResetFlag(itemID)] || level < 1 || level > maxEnhanceLevel {
		return "", errors.New("这件装备没有可用的免费重置机会")
	}
	moneyRefund, bronzeRefund := 100*level*(level+1), level*(level+1)/2
	money, err := model.CheckedAdd(p.Money, moneyRefund)
	if err != nil {
		return "", err
	}
	bronze, err := model.CheckedAdd(p.Backpack["bronze_fragment"], bronzeRefund)
	if err != nil {
		return "", err
	}
	if p.Backpack == nil {
		p.Backpack = map[string]int{}
	}
	p.Money = money
	p.Backpack["bronze_fragment"] = bronze
	delete(p.EnhancedItems, itemID)
	delete(p.Flags, enhancementResetFlag(itemID))
	clampPlayerCurrentResources(p)
	if err := r.savePlayers(players); err != nil {
		return "", err
	}
	return def.Name + "强化已重置，返还" + strconv.Itoa(moneyRefund) + "铜钱、古铜残片×" + strconv.Itoa(bronzeRefund) + "。", nil
}
