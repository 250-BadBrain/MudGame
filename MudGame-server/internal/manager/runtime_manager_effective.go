package manager

import (
	"math"
	"time"

	"mudgame/server/internal/model"
)

func effectiveAttributesForPlayerNoLock(p *model.Player) model.EffectiveAttributes {
	if p == nil {
		return model.EffectiveAttributes{}
	}
	attrs := model.NormalizeAttributes(p.Attributes)
	eff := model.BuildEffectiveAttributes(attrs)
	// 玩家气血上限 = 60 + 根骨×4（根骨生效），再叠加内功转化与技能/装备加成
	eff.MaxHP = 60 + eff.Constitution*4
	eff.Attack = 0
	eff.Defense = 0
	eff.HitRate = 0
	eff.Dodge = 0
	eff.Parry = 0
	applySkillBonusesNoLock(&eff, p.Skills)
	applyEquipmentEffectsNoLock(&eff, p.Equipment)
	directAttack := eff.Attack
	directDefense := eff.Defense
	directHitRate := eff.HitRate
	directDodge := eff.Dodge
	directParry := eff.Parry
	derived := model.CalculateCombatDerived(eff)
	eff.Attack = derived.Attack + directAttack
	eff.Defense = derived.Defense + directDefense
	eff.HitRate = derived.HitRate + directHitRate
	eff.Dodge = derived.Dodge + directDodge
	eff.Parry = derived.Parry + directParry
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

func applySkillBonusesNoLock(eff *model.EffectiveAttributes, skills map[string]int) {
	if eff == nil {
		return
	}
	trainedMPToHPPercent := 0.0
	for skillID, level := range skills {
		if level <= 0 {
			continue
		}
		def, ok := fileSkills[skillID]
		if !ok {
			continue
		}
		for key, value := range def.Bonuses {
			if key == "trained_mp_to_max_hp_percent" {
				trainedMPToHPPercent += value
				continue
			}
			applyLeveledBonus(eff, key, value, level)
		}
	}
	if trainedMPToHPPercent > 0 {
		eff.MaxHP += int(math.Ceil(float64(eff.TrainedMaxMP) * trainedMPToHPPercent))
	}
	clampEffectiveResources(eff)
}

func applyEquipmentEffectsNoLock(eff *model.EffectiveAttributes, equipment map[string]string) {
	if eff == nil {
		return
	}
	for _, itemID := range equipment {
		def, ok := fileItems[itemID]
		if !ok {
			continue
		}
		for key, value := range def.Effects {
			applyFlatBonus(eff, key, toIntEffect(value))
		}
	}
	clampEffectiveResources(eff)
}

func applyLeveledBonus(eff *model.EffectiveAttributes, key string, value float64, level int) {
	switch key {
	case "post_strength_per_10_levels":
		eff.Strength += int(value) * (level / 10)
	case "post_constitution_per_10_levels":
		eff.Constitution += int(value) * (level / 10)
	case "post_agility_per_10_levels":
		eff.Agility += int(value) * (level / 10)
	case "post_perception_per_10_levels", "intelligence_check_per_10_levels":
		eff.Perception += int(value) * (level / 10)
	case "post_willpower_per_10_levels":
		eff.Willpower += int(value) * (level / 10)
	case "attack_per_10_levels":
		eff.Attack += int(value) * (level / 10)
	case "defense_per_10_levels":
		eff.Defense += int(value) * (level / 10)
	case "hit_rate_per_10_levels":
		eff.HitRate += int(value) * (level / 10)
	case "dodge_per_10_levels":
		eff.Dodge += int(value) * (level / 10)
	case "parry_per_10_levels":
		eff.Parry += int(value) * (level / 10)
	case "crit_rate_per_10_levels":
		eff.CritRate += int(value) * (level / 10)
	case "max_mp_per_level":
		eff.MaxMP += int(value) * level
	case "max_hp_per_level":
		eff.MaxHP += int(value) * level
		eff.HP += int(value) * level
	}
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
	case "critRate":
		eff.CritRate += value
	case "maxHp":
		eff.MaxHP += value
		eff.HP += value
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

type jsonNumber interface {
	Int() int
}
