package model

import "testing"

func TestCalculateCombatDerived(t *testing.T) {
	derived := CalculateCombatDerived(EffectiveAttributes{
		Strength: 16, Constitution: 16, Agility: 16, Perception: 16, Willpower: 16,
	})
	if derived.Attack != 50 || derived.HitRate != 98 || derived.Defense != 53 || derived.Parry != 70 || derived.Dodge != 70 {
		t.Fatalf("unexpected derived attributes: %+v", derived)
	}
}

func TestCalculateCombatDerivedUsesPostAttributeTotals(t *testing.T) {
	base := CalculateCombatDerived(EffectiveAttributes{
		Strength: 16, Constitution: 16, Agility: 16, Perception: 16, Willpower: 16,
	})
	boosted := CalculateCombatDerived(EffectiveAttributes{
		Strength: 20, Constitution: 18, Agility: 22, Perception: 19, Willpower: 17,
	})
	if boosted.Attack <= base.Attack || boosted.HitRate <= base.HitRate || boosted.Defense <= base.Defense || boosted.Parry <= base.Parry || boosted.Dodge <= base.Dodge {
		t.Fatalf("post attributes should dynamically raise all derived values: base=%+v boosted=%+v", base, boosted)
	}
}
