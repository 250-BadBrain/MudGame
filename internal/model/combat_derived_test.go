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

func TestNormalizeAttributesPreservesZeroHPForDeadCharacter(t *testing.T) {
	attrs := NewPlayerAttributes(16, 16, 16, 16, 16)
	attrs.Resources.HP = 0

	normalized := NormalizeAttributes(attrs)
	if normalized.Resources.HP != 0 {
		t.Fatalf("dead character hp should remain zero, got %d", normalized.Resources.HP)
	}
}

func TestNormalizeAttributesKeepsDepletedMPAndSpirit(t *testing.T) {
	attrs := NewPlayerAttributes(16, 16, 16, 16, 16)
	attrs.Resources.MP = 0
	attrs.Resources.Spirit = 0

	normalized := NormalizeAttributes(attrs)
	if normalized.Resources.MP != 0 {
		t.Fatalf("depleted mp should remain zero, got %d", normalized.Resources.MP)
	}
	if normalized.Resources.Spirit != 0 {
		t.Fatalf("depleted spirit should remain zero, got %d", normalized.Resources.Spirit)
	}
}

func TestNormalizeAttributesFillsResourcesForLegacySave(t *testing.T) {
	// 旧存档缺少 TrainedMaxMP/MaxSpirit 字段时应回满，避免出现 0 上限
	attrs := NewPlayerAttributes(16, 16, 16, 16, 16)
	attrs.Resources.TrainedMaxMP = 0
	attrs.Resources.MaxSpirit = 0
	attrs.Resources.MP = 0
	attrs.Resources.Spirit = 0

	normalized := NormalizeAttributes(attrs)
	if normalized.Resources.TrainedMaxMP <= 0 {
		t.Fatalf("legacy save should get trained max mp, got %d", normalized.Resources.TrainedMaxMP)
	}
	if normalized.Resources.MP != normalized.Resources.TrainedMaxMP {
		t.Fatalf("legacy save mp should refill to trained max, got %d", normalized.Resources.MP)
	}
	if normalized.Resources.MaxSpirit <= 0 {
		t.Fatalf("legacy save should get max spirit, got %d", normalized.Resources.MaxSpirit)
	}
	if normalized.Resources.Spirit != normalized.Resources.MaxSpirit {
		t.Fatalf("legacy save spirit should refill to max, got %d", normalized.Resources.Spirit)
	}
}
