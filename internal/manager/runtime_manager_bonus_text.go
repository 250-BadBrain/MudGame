package manager

import "strconv"

func skillBonusText(bonuses map[string]float64, level int) []string {
	out := []string{}
	if value := bonuses["trained_mp_to_max_hp_percent"]; level > 0 && value > 0 {
		out = append(out, "转血 +"+strconv.FormatFloat(value*100, 'f', -1, 64)+"%")
	}
	for key, value := range bonuses {
		amount := 0
		switch key {
		case "post_strength_per_10_levels":
			amount = int(value) * (level / 10)
			out = appendPositiveBonus(out, "后天臂力", amount)
		case "post_constitution_per_10_levels":
			amount = int(value) * (level / 10)
			out = appendPositiveBonus(out, "后天根骨", amount)
		case "post_agility_per_10_levels":
			amount = int(value) * (level / 10)
			out = appendPositiveBonus(out, "后天身法", amount)
		case "post_perception_per_10_levels", "intelligence_check_per_10_levels":
			amount = int(value) * (level / 10)
			out = appendPositiveBonus(out, "后天悟性", amount)
		case "post_willpower_per_10_levels":
			amount = int(value) * (level / 10)
			out = appendPositiveBonus(out, "后天心志", amount)
		case "attack_per_10_levels":
			amount = int(value) * (level / 10)
			out = appendPositiveBonus(out, "攻击", amount)
		case "defense_per_10_levels":
			amount = int(value) * (level / 10)
			out = appendPositiveBonus(out, "防御", amount)
		case "hit_rate_per_10_levels":
			amount = int(value) * (level / 10)
			out = appendPositiveBonus(out, "命中", amount)
		case "dodge_per_10_levels":
			amount = int(value) * (level / 10)
			out = appendPositiveBonus(out, "躲闪", amount)
		case "parry_per_10_levels":
			amount = int(value) * (level / 10)
			out = appendPositiveBonus(out, "招架", amount)
		case "crit_rate_per_10_levels":
			amount = int(value) * (level / 10)
			out = appendPositiveBonus(out, "暴击率", amount)
		case "max_mp_per_level":
			amount = int(value) * level
			out = appendPositiveBonus(out, "内力上限", amount)
		case "max_hp_per_level":
			amount = int(value) * level
			out = appendPositiveBonus(out, "气血上限", amount)
		case "trained_mp_to_max_hp_percent":
			continue
		default:
			out = appendPositiveBonus(out, key, int(value))
		}
	}
	return out
}

func appendPositiveBonus(out []string, label string, amount int) []string {
	if amount <= 0 {
		return out
	}
	return append(out, label+" +"+strconv.Itoa(amount))
}
