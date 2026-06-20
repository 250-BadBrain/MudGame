package manager

import "testing"

func TestInternalSkillConversionBonusIsFirstAndFixed(t *testing.T) {
	bonuses := map[string]float64{
		"max_mp_per_level":             10,
		"trained_mp_to_max_hp_percent": 0.3,
	}

	levelOne := skillBonusText(bonuses, 1)
	levelHundred := skillBonusText(bonuses, 100)
	if len(levelOne) < 2 || len(levelHundred) < 2 {
		t.Fatalf("expected conversion and level bonuses, got level1=%+v level100=%+v", levelOne, levelHundred)
	}
	if levelOne[0] != "转血 +30%" || levelHundred[0] != "转血 +30%" {
		t.Fatalf("conversion bonus must remain first and fixed, got level1=%+v level100=%+v", levelOne, levelHundred)
	}
}
