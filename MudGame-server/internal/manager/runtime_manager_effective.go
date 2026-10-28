package manager

import (
	"math"
	"strings"
	"time"

	"mudgame/server/internal/model"
)

func effectiveAttributesForPlayerNoLock(p *model.Player) model.EffectiveAttributes {
	if p == nil {
		return model.EffectiveAttributes{}
	}
	eff := model.BuildEffectiveAttributes(model.NormalizeAttributes(p.Attributes))
	eff.MaxHP, eff.MaxMP, eff.MaxSpirit = 0, 0, 0
	eff.Attack, eff.Defense, eff.HitRate, eff.Dodge, eff.Parry, eff.Initiative = 0, 0, 0, 0, 0, 0
	eff.CritRate = 1
	conversion := applyMartialBonusesNoLock(&eff, p)
	applyEquipmentEffectsNoLock(&eff, p.Equipment)
	applyEnhancementNoLock(&eff, p)
	eff.MaxMP += max(50, (eff.Perception+eff.Willpower)*5)
	eff.TrainedMaxMP = min(eff.TrainedMaxMP, eff.MaxMP)
	eff.MaxHP += 60 + eff.Constitution*4 + int(math.Ceil(float64(eff.TrainedMaxMP)*conversion))
	eff.MaxSpirit += max(60, (eff.Perception+eff.Willpower)*3)
	derived := model.CalculateCombatDerived(eff)
	eff.Attack += derived.Attack
	eff.Defense += derived.Defense
	eff.HitRate += derived.HitRate
	eff.Dodge += derived.Dodge
	eff.Parry += derived.Parry
	eff.Initiative += 60 + eff.Agility + eff.Perception/2
	eff.CritRate = min(25, max(0, math.Round(eff.CritRate*1e6)/1e6))
	clampEffectiveResources(&eff)
	applyWeaknessNoLock(&eff, p)
	return eff
}

// applyWeaknessNoLock 死亡复活后的虚弱期内战斗属性减半（见 docs/design/formulas.md 1.5）
func applyWeaknessNoLock(eff *model.EffectiveAttributes, p *model.Player) {
	if p == nil || !p.IsWeak(time.Now().Unix()) {
		return
	}
	eff.Attack /= 2
	eff.Defense /= 2
	eff.HitRate /= 2
	eff.Dodge /= 2
	eff.Parry /= 2
	eff.Initiative /= 2
}

func applyMartialBonusesNoLock(eff *model.EffectiveAttributes, p *model.Player) float64 {
	conversion := 0.0
	for id, level := range p.Skills {
		def, exists := fileSkills[id]
		if !exists || skillActiveReason(p, def) != "" {
			continue
		}
		for key, value := range def.Bonuses {
			if key == "trained_mp_to_max_hp_percent" {
				if def.Type == "BASIC" {
					conversion += value
				} else {
					conversion += value * float64(min(level, 100)) / 100
				}
			} else {
				applyLeveledBonus(eff, key, value, level)
			}
		}
	}
	return min(.45, conversion)
}

// skillPrerequisiteID 返回技能声明的根基 ID，无声明返回空串。
func skillPrerequisiteID(def model.SkillDef) string {
	prereq, ok := def.Requirements["prerequisite"].(map[string]interface{})
	if !ok {
		return ""
	}
	id, _ := prereq["skill"].(string)
	return id
}

// skillPrerequisiteLevel 返回技能声明的前置基本武功最低等级，未声明返回 0。
func skillPrerequisiteLevel(def model.SkillDef) int {
	prereq, ok := def.Requirements["prerequisite"].(map[string]interface{})
	if !ok {
		return 0
	}
	return toIntEffect(prereq["level"])
}

func applyEquipmentEffectsNoLock(eff *model.EffectiveAttributes, equipment map[string]string) {
	if eff == nil {
		return
	}
	for slot, itemID := range equipment {
		def, ok := fileItems[itemID]
		if !ok || def.EquipmentSlot != slot {
			continue
		}
		for key, value := range def.Effects {
			applyNumericBonus(eff, key, toFloatEffect(value))
		}
	}
}

func applyLeveledBonus(eff *model.EffectiveAttributes, key string, value float64, level int) {
	if strings.HasSuffix(key, "_per_100_levels") {
		stat := strings.TrimSuffix(key, "_per_100_levels")
		if stat == "crit_rate" {
			eff.CritRate += float64(value) * float64(level) / 100
			return
		}
		n := leveledValue(value, level)
		switch stat {
		case "post_strength":
			eff.Strength += n
		case "post_constitution":
			eff.Constitution += n
		case "post_willpower":
			eff.Willpower += n
		case "post_agility":
			eff.Agility += n
		case "post_perception":
			eff.Perception += n
		case "meditate_recovery":
			eff.MeditateRecoveryBonus += n
		case "learning_invest_bonus":
			eff.LearningInvestBonus += n
		case "initiative":
			eff.Initiative += n
		default:
			alias := map[string]string{"max_mp": "maxMp", "max_hp": "maxHp", "hit_rate": "hitRate"}
			if key, ok := alias[stat]; ok {
				stat = key
			}
			applyFlatBonus(eff, stat, n)
		}
		return
	}
}

func applyNumericBonus(eff *model.EffectiveAttributes, key string, value float64) {
	if key == "critRate" {
		eff.CritRate += value
		return
	}
	applyFlatBonus(eff, key, int(value))
}

func applyFlatBonus(eff *model.EffectiveAttributes, key string, value int) {
	switch key {
	case "strength", "postStrength":
		eff.Strength += value
	case "constitution", "postConstitution":
		eff.Constitution += value
	case "agility", "postAgility":
		eff.Agility += value
	case "perception", "postPerception":
		eff.Perception += value
	case "willpower", "postWillpower":
		eff.Willpower += value
	case "charm":
		eff.Charm += value
	case "luck":
		eff.Luck += value
	case "attack":
		eff.Attack += value
	case "defense":
		eff.Defense += value
	case "hitRate":
		eff.HitRate += value
	case "dodge":
		eff.Dodge += value
	case "parry":
		eff.Parry += value
	case "initiative":
		eff.Initiative += value
	case "critRate":
		eff.CritRate += float64(value)
	case "maxHp":
		eff.MaxHP += value
	case "maxMp":
		eff.MaxMP += value
	case "spirit", "maxSpirit":
		eff.MaxSpirit += value
		eff.Spirit += value
	}
}

func clampEffectiveResources(eff *model.EffectiveAttributes) {
	if eff == nil {
		return
	}
	if eff.TrainedMaxMP > eff.MaxMP {
		eff.TrainedMaxMP = eff.MaxMP
	}
	if eff.HP > eff.MaxHP {
		eff.HP = eff.MaxHP
	}
	if eff.MP > eff.TrainedMaxMP {
		eff.MP = eff.TrainedMaxMP
	}
	if eff.Spirit > eff.MaxSpirit {
		eff.Spirit = eff.MaxSpirit
	}
}

func toIntEffect(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	case jsonNumber:
		return v.Int()
	default:
		return 0
	}
}

func toFloatEffect(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	default:
		return float64(toIntEffect(value))
	}
}

type jsonNumber interface {
	Int() int
}
