package model

import "testing"

func TestSkillLevelCapForExperience(t *testing.T) {
	// 曲线: 所需经验 = 40 × root³；上限 = 20 + 10×root
	tests := []struct {
		experience int
		want       int
	}{
		{0, 20},
		{39, 20},
		{40, 30},
		{319, 30},
		{320, 40},
		{2560, 60},
		{2720, 60},     // 主线七副本通关一遍的经验总量
		{20480, 100},   // 40×8³
		{878080, 300},  // 40×28³
		{2195200, 400}, // 40×38³
	}
	for _, tt := range tests {
		if got := SkillLevelCapForExperience(tt.experience); got != tt.want {
			t.Fatalf("experience %d: expected cap %d, got %d", tt.experience, tt.want, got)
		}
	}
}

func TestSkillPotentialForTargetLevel(t *testing.T) {
	tests := []struct {
		level int
		want  int
	}{
		{1, 1},
		{20, 1},
		{21, 2},
		{100, 11},
		{200, 60},
		{300, 210},
		{301, 214},
		{400, 500},
	}
	for _, tt := range tests {
		if got := SkillPotentialForTargetLevel(tt.level); got != tt.want {
			t.Fatalf("target level %d: expected cost %d, got %d", tt.level, tt.want, got)
		}
	}
}
