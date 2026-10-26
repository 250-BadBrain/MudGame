package manager

import (
	"strings"
	"testing"
)

func TestInternalSkillConversionIsFixedAndAbsoluteConversionGrows(t *testing.T) {
	base := fileSkills["basic_neigong"]
	if !strings.Contains(strings.Join(skillBonusText(base.Bonuses, 1), " "), "30个百分点") || !strings.Contains(strings.Join(skillBonusText(base.Bonuses, 100), " "), "30个百分点") {
		t.Fatal("base conversion should remain fixed")
	}
	art := fileSkills["csj_changshengjue"]
	if !strings.Contains(strings.Join(skillBonusText(art.Bonuses, 1), " "), "0.15个百分点") || !strings.Contains(strings.Join(skillBonusText(art.Bonuses, 100), " "), "15个百分点") {
		t.Fatal("absolute conversion must grow with its own level")
	}
}
