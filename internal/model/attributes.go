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
	Spirit          int
	MaxSpirit       int
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
			HP:           100,
			MaxHP:        100,
			MP:           80,
			TrainedMaxMP: 80,
			MaxMP:        200,
			Spirit:       100,
			MaxSpirit:    100,
		},
		Derived: DerivedAttributes{
			Attack:      15,
			Defense:     15,
			HitRate:     100,
			Dodge:       100,
			Parry:       100,
			CritRate:    1,
			AttackSpeed: 4,
			Initiative:  100,
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
	if a.Resources.MaxHP <= 0 {
		a.Resources.MaxHP = max(80, a.Base.Constitution*5)
	}
	if a.Resources.HP <= 0 || a.Resources.HP > a.Resources.MaxHP {
		a.Resources.HP = a.Resources.MaxHP
	}
	if a.Resources.MaxMP <= 0 {
		a.Resources.MaxMP = max(50, (a.Base.Perception+a.Base.Willpower)*5)
	}
	if a.Resources.TrainedMaxMP <= 0 || a.Resources.TrainedMaxMP > a.Resources.MaxMP {
		a.Resources.TrainedMaxMP = min(a.Resources.MaxMP, max(30, a.Resources.MaxMP/2))
	}
	if a.Resources.MP <= 0 || a.Resources.MP > a.Resources.TrainedMaxMP {
		a.Resources.MP = a.Resources.TrainedMaxMP
	}
	if a.Resources.MaxSpirit <= 0 {
		a.Resources.MaxSpirit = max(60, (a.Base.Perception+a.Base.Willpower)*3)
	}
	if a.Resources.Spirit <= 0 || a.Resources.Spirit > a.Resources.MaxSpirit {
		a.Resources.Spirit = a.Resources.MaxSpirit
	}
	eff := BuildEffectiveAttributes(a)
	if a.Derived.Attack <= 0 {
		a.Derived.Attack = max(1, eff.Strength/2+eff.Agility/4)
	}
	if a.Derived.Defense <= 0 {
		a.Derived.Defense = max(1, eff.Constitution/2+eff.Willpower/4)
	}
	if a.Derived.HitRate <= 0 {
		a.Derived.HitRate = 60 + eff.Perception + eff.Agility/2
	}
	if a.Derived.Dodge <= 0 {
		a.Derived.Dodge = 60 + eff.Agility
	}
	if a.Derived.Parry <= 0 {
		a.Derived.Parry = 50 + eff.Strength/2 + eff.Perception/2
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

func ApplyEffectiveResources(a *Attributes, eff EffectiveAttributes) {
	a.Resources.HP = eff.HP
	a.Resources.MaxHP = eff.MaxHP
	a.Resources.MP = eff.MP
	a.Resources.TrainedMaxMP = eff.TrainedMaxMP
	a.Resources.MaxMP = eff.MaxMP
	a.Resources.Spirit = eff.Spirit
	a.Resources.MaxSpirit = eff.MaxSpirit
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
