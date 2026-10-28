package manager

import (
	"fmt"
	"mudgame/server/internal/model"
)

func canonicalItemID(id string) string {
	if canonical := loadedStaticData.ItemAliases[id]; canonical != "" {
		return canonical
	}
	return id
}

// The old key is removed in the same saved transaction as its compensation.
// Thus reloading or rerunning normalization cannot issue the refund twice.
func normalizePlayerItemAliases(p *model.Player) (bool, error) {
	if p == nil {
		return false, nil
	}
	changed := false
	for old, target := range loadedStaticData.ItemAliases {
		if n, ok := p.Backpack[old]; ok {
			total, err := model.CheckedAdd(p.Backpack[target], n)
			if err != nil {
				return false, err
			}
			p.Backpack[target] = total
			delete(p.Backpack, old)
			changed = true
		}
		for slot, id := range p.Equipment {
			if id == old {
				p.Equipment[slot] = target
				changed = true
			}
		}
		if oldLevel, ok := p.EnhancedItems[old]; ok {
			level := p.EnhancedItems[target]
			if oldLevel < 0 || level < 0 || oldLevel > maxEnhanceLevel || level > maxEnhanceLevel {
				return false, fmt.Errorf("invalid legacy enhancement: %s", old)
			}
			duplicate := min(oldLevel, level)
			money, err := model.CheckedAdd(p.Money, 100*duplicate*(duplicate+1))
			if err != nil {
				return false, err
			}
			if p.Backpack == nil {
				p.Backpack = map[string]int{}
			}
			bronze, err := model.CheckedAdd(p.Backpack["bronze_fragment"], duplicate*(duplicate+1)/2)
			if err != nil {
				return false, err
			}
			p.Money = money
			if bronze > 0 {
				p.Backpack["bronze_fragment"] = bronze
			}
			p.EnhancedItems[target] = max(oldLevel, level)
			delete(p.EnhancedItems, old)
			changed = true
		}
		for _, prefix := range []string{"item_claimed_", "enhancement_reset_available_"} {
			if value, ok := p.Flags[prefix+old]; ok {
				p.Flags[prefix+target] = p.Flags[prefix+target] || value
				delete(p.Flags, prefix+old)
				changed = true
			}
		}
	}
	return changed, nil
}
