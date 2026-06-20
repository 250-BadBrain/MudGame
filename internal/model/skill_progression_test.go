package model

import "testing"

func TestSkillLevelCapForExperience(t *testing.T) {
	tests := []struct {
		experience int
		want       int
	}{
		{0, 20},
		{249, 20},
		{250, 30},
		{1999, 30},
		{2000, 40},
		{128000, 100},
		{5488000, 300},
		{13718000, 400},
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
