package gamedata

import (
	"fmt"
	"os"
	"path/filepath"

	"mudgame/server/internal/model"
)

// Count is intentionally unconstrained: expansion adds definitions, not code branches.
func validateMartialData(dir string, skills map[string]model.SkillDef, items map[string]model.ItemDef, npcs map[string]model.NPC) error {
	slots := map[string]bool{"internal": true, "movement": true, "guard": true, "combat": true, "study": true}
	weapons := map[string]bool{"blade": true, "sword": true, "staff": true, "hidden_weapon": true, "spear": true, "whip": true, "mace": true, "hook": true}
	sources := map[string]map[string]int{}
	entries, err := os.ReadDir(filepath.Join(dir, "dungeons"))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var d struct {
			ID      string `json:"id"`
			Rewards struct {
				Items map[string]int `json:"items"`
			} `json:"rewards"`
		}
		if err := readJSON(filepath.Join(dir, "dungeons", entry.Name()), &d); err != nil {
			return err
		}
		sources[d.ID] = d.Rewards.Items
	}
	names := map[string]bool{}
	roots := map[string]bool{}
	validBonuses := map[string]bool{"attack_per_100_levels": true, "defense_per_100_levels": true, "hit_rate_per_100_levels": true, "dodge_per_100_levels": true, "parry_per_100_levels": true, "max_mp_per_100_levels": true, "max_hp_per_100_levels": true, "initiative_per_100_levels": true, "post_strength_per_100_levels": true, "post_constitution_per_100_levels": true, "post_agility_per_100_levels": true, "post_perception_per_100_levels": true, "post_willpower_per_100_levels": true, "meditate_recovery_per_100_levels": true, "learning_invest_bonus_per_100_levels": true, "crit_rate_per_100_levels": true, "trained_mp_to_max_hp_percent": true}
	for id, s := range skills {
		if !slots[s.Slot] || s.Rarity < 1 || s.Rarity > 5 || names[s.Name] {
			return fmt.Errorf("invalid martial slot, grade or duplicate name: %s", id)
		}
		names[s.Name] = true
		for key, value := range s.Bonuses {
			if !validBonuses[key] || value < 0 {
				return fmt.Errorf("unknown or negative bonus %s/%s", id, key)
			}
		}
		if s.Slot != "combat" && s.WeaponType != "" {
			return fmt.Errorf("noncombat art has weapon type: %s", id)
		}
		if s.Slot == "combat" && s.WeaponType != "unarmed" && !weapons[s.WeaponType] {
			return fmt.Errorf("skill %s has invalid weapon type", id)
		}
		if s.Type == "BASIC" {
			rootKey := s.Slot + "/" + s.WeaponType
			if s.Rarity != 1 || roots[rootKey] {
				return fmt.Errorf("duplicate or graded basic root: %s", id)
			}
			roots[rootKey] = true
			taught := false
			for _, npc := range npcs {
				if !npc.InstanceOnly && npc.Skills[id] > 0 {
					taught = true
					break
				}
			}
			if !taught {
				return fmt.Errorf("basic root has no public teacher: %s", id)
			}
			continue
		}
		if s.Type != "ADVANCED" || s.Rarity < 2 {
			return fmt.Errorf("invalid skill type: %s", id)
		}
		pre, ok := s.Requirements["prerequisite"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("missing root: %s", id)
		}
		baseID, _ := pre["skill"].(string)
		base, ok := skills[baseID]
		level, valid := pre["level"].(float64)
		if !ok || !valid || level < 1 || base.Type != "BASIC" || base.Slot != s.Slot || base.WeaponType != s.WeaponType {
			return fmt.Errorf("incompatible root: %s", id)
		}
		book, ok := items[s.ManualID]
		if !ok || book.Effects["learnSkill"] != id || book.Sellable || book.Value != 0 || book.Bind != "character" {
			return fmt.Errorf("invalid complete manual: %s", id)
		}
		if s.TeacherID != "" {
			if s.Rarity != 2 || npcs[s.TeacherID].Skills[id] < 1 || s.SourceDungeon != "" {
				return fmt.Errorf("invalid teacher: %s", id)
			}
		} else if s.SourceDungeon == "" || sources[s.SourceDungeon][s.ManualID] < 1 {
			return fmt.Errorf("missing complete transmission reward: %s", id)
		}
	}
	for id, item := range items {
		if item.EquipmentSlot == "HIDDEN_WEAPON" || (item.EquipmentSlot == "WEAPON" && !weapons[item.WeaponType]) {
			return fmt.Errorf("invalid primary weapon: %s", id)
		}
		if sid, ok := item.Effects["learnSkill"].(string); ok && (skills[sid].ID == "" || skills[sid].ManualID != id) {
			return fmt.Errorf("orphan manual: %s", id)
		}
	}
	for id, npc := range npcs {
		for sid := range npc.Skills {
			s, ok := skills[sid]
			if !ok || (s.Type == "ADVANCED" && s.TeacherID != id) {
				return fmt.Errorf("invalid teaching link %s/%s", id, sid)
			}
		}
	}
	return nil
}
