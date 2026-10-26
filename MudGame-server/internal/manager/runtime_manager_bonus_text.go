package manager

import (
	"fmt"
	"sort"
	"strings"
)

func skillBonusText(bonuses map[string]float64, level int) []string {
	labels := map[string]string{"post_strength": "臂力", "post_constitution": "根骨", "post_willpower": "心志", "attack": "攻击", "defense": "防御", "hit_rate": "命中", "dodge": "躲闪", "parry": "招架", "max_mp": "内力上限", "max_hp": "气血上限", "initiative": "先手", "post_agility": "身法", "post_perception": "悟性", "learning_invest_bonus": "每10秒潜能投入上限", "meditate_recovery": "每10秒打坐回复", "crit_rate": "暴击率"}
	out := []string{}
	keys := []string{}
	for key := range bonuses {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := bonuses[key]
		if key == "trained_mp_to_max_hp_percent" && level > 0 {
			percent := value * 100
			if value < .3 {
				percent *= float64(min(level, 100)) / 100
			}
			out = append(out, fmt.Sprintf("已修内力转气血 +%g个百分点", percent))
			continue
		}
		if !strings.HasSuffix(key, "_per_100_levels") {
			continue
		}
		stat := strings.TrimSuffix(key, "_per_100_levels")
		amount := float64(leveledValue(value, level))
		suffix := ""
		if stat == "crit_rate" {
			amount = value * float64(level) / 100
			suffix = "个百分点"
		}
		if amount > 0 {
			out = append(out, fmt.Sprintf("%s +%g%s", labels[stat], amount, suffix))
		}
	}
	return out
}
