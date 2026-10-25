package model

const DefaultRealm = "ordinary"

type EffectiveAttributes struct {
	Level           int
	Realm           string
	Experience      int
	Potential       int
	HP              int
	MaxHP           int
	MP              int
	TrainedMaxMP    int
	MaxMP           int
	// MeditateRecoveryBonus 打坐内力恢复加成（仅秘籍装备规则下的 meditate_recovery_per_10_levels 来源）
	MeditateRecoveryBonus int
	Spirit                int
	MaxSpirit             int
	Strength        int
	Constitution    int
	Agility         int
	Perception      int
	Willpower       int
	Charm           int
	Luck            int
	Attack          int
	Defense         int
	HitRate         int
	Dodge           int
	Parry           int
	CritRate        int
	AttackSpeed     int
	Initiative      int
	GuardBreak      int
	DamageReduction int
	Fame            int
	Notoriety       int
	Authority       int
	Credit          int
	PoisonResist    int
	BleedResist     int
	StunResist      int
	FearResist      int
	CharmResist     int
	SealResist      int
}

func NewPlayerAttributes(strength, constitution, agility, perception, willpower int) Attributes {
	return NormalizeAttributes(Attributes{
		Level:      1,
		Realm:      DefaultRealm,
		Experience: 0,
		Potential:  500,
		Base: BaseAttributes{
			Strength:     strength,
			Constitution: constitution,
			Agility:      agility,
			Perception:   perception,
			Willpower:    willpower,
			Charm:        20,
			Luck:         20,
		},
		Resources: ResourceAttributes{
			HP:           80,
			MaxHP:        80,
			MP:           80,
			TrainedMaxMP: 80,
			MaxMP:        200,
			Spirit:       100,
			MaxSpirit:    100,
		},
	})
}

func NormalizeAttributes(a Attributes) Attributes {
	if a.Level <= 0 {
		a.Level = 1
	}
	if a.Realm == "" {
		a.Realm = DefaultRealm
	}
	maxHPMissing := a.Resources.MaxHP <= 0
	if maxHPMissing {
		a.Resources.MaxHP = max(80, a.Base.Constitution*5)
	}
	if maxHPMissing && a.Resources.HP == 0 {
		a.Resources.HP = a.Resources.MaxHP
	} else if a.Resources.HP < 0 {
		a.Resources.HP = 0
	}
	if a.Resources.MaxMP <= 0 {
		a.Resources.MaxMP = max(50, (a.Base.Perception+a.Base.Willpower)*5)
	}
	// 与 HP 的兜底策略一致：只有当上限字段本身缺失（旧存档）时才回满当前值，
	// 正常游戏中耗尽的 MP 不能在存/读档时被免费回满
	trainedMaxMPMissing := a.Resources.TrainedMaxMP <= 0
	if trainedMaxMPMissing {
		a.Resources.TrainedMaxMP = min(a.Resources.MaxMP, max(30, a.Resources.MaxMP/2))
	}
	if trainedMaxMPMissing && a.Resources.MP <= 0 {
		a.Resources.MP = a.Resources.TrainedMaxMP
	} else if a.Resources.MP < 0 {
		a.Resources.MP = 0
	} else if a.Resources.MP > a.Resources.TrainedMaxMP {
		a.Resources.MP = a.Resources.TrainedMaxMP
	}
	maxSpiritMissing := a.Resources.MaxSpirit <= 0
	if maxSpiritMissing {
		a.Resources.MaxSpirit = max(60, (a.Base.Perception+a.Base.Willpower)*3)
	}
	if maxSpiritMissing && a.Resources.Spirit <= 0 {
		a.Resources.Spirit = a.Resources.MaxSpirit
	} else if a.Resources.Spirit < 0 {
		a.Resources.Spirit = 0
	} else if a.Resources.Spirit > a.Resources.MaxSpirit {
		a.Resources.Spirit = a.Resources.MaxSpirit
	}
	eff := BuildEffectiveAttributes(a)
	combatDerived := CalculateCombatDerived(eff)
	if a.Derived.Attack <= 0 {
		a.Derived.Attack = combatDerived.Attack
	}
	if a.Derived.Defense <= 0 {
		a.Derived.Defense = combatDerived.Defense
	}
	if a.Derived.HitRate <= 0 {
		a.Derived.HitRate = combatDerived.HitRate
	}
	if a.Derived.Dodge <= 0 {
		a.Derived.Dodge = combatDerived.Dodge
	}
	if a.Derived.Parry <= 0 {
		a.Derived.Parry = combatDerived.Parry
	}
	if a.Derived.CritRate <= 0 {
		a.Derived.CritRate = 1
	}
	if a.Derived.AttackSpeed <= 0 {
		a.Derived.AttackSpeed = 4
	}
	if a.Derived.Initiative <= 0 {
		a.Derived.Initiative = 60 + eff.Agility + eff.Perception/2
	}
	return a
}

func CalculateCombatDerived(eff EffectiveAttributes) DerivedAttributes {
	return DerivedAttributes{
		Attack:  max(1, 10+eff.Strength*2+eff.Agility/2),
		HitRate: max(1, 50+eff.Perception*2+eff.Agility),
		Defense: max(1, 5+eff.Constitution*2+eff.Strength/2+eff.Willpower/2),
		Parry:   max(1, 30+eff.Strength+eff.Perception+eff.Agility/2),
		Dodge:   max(1, 30+eff.Agility*2+eff.Perception/2),
	}
}

func BuildEffectiveAttributes(a Attributes) EffectiveAttributes {
	base := a.Base
	growth := a.Growth
	return EffectiveAttributes{
		Level:           max(1, a.Level),
		Realm:           firstNonEmpty(a.Realm, DefaultRealm),
		Experience:      a.Experience,
		Potential:       a.Potential,
		HP:              a.Resources.HP,
		MaxHP:           a.Resources.MaxHP,
		MP:              a.Resources.MP,
		TrainedMaxMP:    a.Resources.TrainedMaxMP,
		MaxMP:           a.Resources.MaxMP,
		Spirit:          a.Resources.Spirit,
		MaxSpirit:       a.Resources.MaxSpirit,
		Strength:        base.Strength + growth.PostStrength,
		Constitution:    base.Constitution + growth.PostConstitution,
		Agility:         base.Agility + growth.PostAgility,
		Perception:      base.Perception + growth.PostPerception,
		Willpower:       base.Willpower + growth.PostWillpower,
		Charm:           base.Charm,
		Luck:            base.Luck,
		Attack:          a.Derived.Attack,
		Defense:         a.Derived.Defense,
		HitRate:         a.Derived.HitRate,
		Dodge:           a.Derived.Dodge,
		Parry:           a.Derived.Parry,
		CritRate:        a.Derived.CritRate,
		AttackSpeed:     a.Derived.AttackSpeed,
		Initiative:      a.Derived.Initiative,
		GuardBreak:      a.Derived.GuardBreak,
		DamageReduction: a.Derived.DamageReduction,
		Fame:            a.Social.Fame,
		Notoriety:       a.Social.Notoriety,
		Authority:       a.Social.Authority,
		Credit:          a.Social.Credit,
		PoisonResist:    a.Resistances.Poison,
		BleedResist:     a.Resistances.Bleed,
		StunResist:      a.Resistances.Stun,
		FearResist:      a.Resistances.Fear,
		CharmResist:     a.Resistances.Charm,
		SealResist:      a.Resistances.Seal,
	}
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
