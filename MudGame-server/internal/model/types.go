package model

type User struct {
	Username           string
	PasswordHash       string
	CharacterIDs       []string
	ReconnectTokenHash string
}

type Attributes struct {
	Level       int                  `json:"level"`
	Realm       string               `json:"realm"`
	Experience  int                  `json:"experience"`
	Potential   int                  `json:"potential"`
	Base        BaseAttributes       `json:"base"`
	Growth      GrowthAttributes     `json:"growth"`
	Resources   ResourceAttributes   `json:"resources"`
	Derived     DerivedAttributes    `json:"derived"`
	Social      SocialAttributes     `json:"social"`
	Resistances ResistanceAttributes `json:"resistances"`
}

type BaseAttributes struct {
	Strength     int `json:"strength"`
	Constitution int `json:"constitution"`
	Agility      int `json:"agility"`
	Perception   int `json:"perception"`
	Willpower    int `json:"willpower"`
	Charm        int `json:"charm"`
	Luck         int `json:"luck"`
}

type GrowthAttributes struct {
	PostStrength     int `json:"postStrength"`
	PostConstitution int `json:"postConstitution"`
	PostAgility      int `json:"postAgility"`
	PostPerception   int `json:"postPerception"`
	PostWillpower    int `json:"postWillpower"`
}

type ResourceAttributes struct {
	HP           int `json:"hp"`
	MaxHP        int `json:"maxHp"`
	MP           int `json:"mp"`
	TrainedMaxMP int `json:"trainedMaxMp"`
	MaxMP        int `json:"maxMp"`
	Spirit       int `json:"spirit"`
	MaxSpirit    int `json:"maxSpirit"`
}

type DerivedAttributes struct {
	Attack          int `json:"attack"`
	Defense         int `json:"defense"`
	HitRate         int `json:"hitRate"`
	Dodge           int `json:"dodge"`
	Parry           int `json:"parry"`
	CritRate        int `json:"critRate"`
	AttackSpeed     int `json:"attackSpeed"`
	Initiative      int `json:"initiative"`
	GuardBreak      int `json:"guardBreak"`
	DamageReduction int `json:"damageReduction"`
}

type SocialAttributes struct {
	Fame      int `json:"fame"`
	Notoriety int `json:"notoriety"`
	Authority int `json:"authority"`
	Credit    int `json:"credit"`
}

type ResistanceAttributes struct {
	Poison int `json:"poison"`
	Bleed  int `json:"bleed"`
	Stun   int `json:"stun"`
	Fear   int `json:"fear"`
	Charm  int `json:"charm"`
	Seal   int `json:"seal"`
}

type Player struct {
	ID                    string
	UserID                string
	CharacterName         string
	RoomID                string
	State                 string
	StateTickCountdown    int
	LearningSkillID       string
	LearningTeacherID     string
	LearningTeacherLevel  int
	SkillLearningProgress map[string]int
	TargetID              string
	CombatType            string
	Money                 int
	Attributes            Attributes
	Skills                map[string]int
	// EquippedSkills 秘籍装备槽：键=前置基本武功 ID，值=装备中的秘籍 ID；仅装备中的 ADVANCED 技能参与属性加成
	EquippedSkills        map[string]string
	// EnhancedItems 装备强化等级：键=装备物品 ID，值=强化等级（0-5）；每级 +2 主属性
	EnhancedItems         map[string]int
	Backpack              map[string]int
	Equipment             map[string]string
	QuestStates           map[string]QuestState
	Flags                 map[string]bool
	DialogueHistory       map[string]bool
	Reputations           map[string]int
	Online                bool
	// WeakUntil 为死亡复活后虚弱的到期时间（Unix 秒），0 表示无虚弱
	WeakUntil int64
}

// IsWeak 报告在给定的 Unix 秒时刻玩家是否处于虚弱期。
func (p *Player) IsWeak(now int64) bool {
	return p != nil && p.WeakUntil > now
}

// WeakRemainingSeconds 返回虚弱剩余秒数（不足 1 秒按 1 秒计），无虚弱返回 0。
func (p *Player) WeakRemainingSeconds(now int64) int {
	if !p.IsWeak(now) {
		return 0
	}
	remaining := p.WeakUntil - now
	if remaining < 1 {
		remaining = 1
	}
	return int(remaining)
}

type QuestState struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	Progress    int    `json:"progress"`
	CompletedAt int64  `json:"completedAt,omitempty"`
}

type ItemDef struct {
	ID            string
	Name          string
	Description   string
	Type          string
	Subtype       string
	Rarity        int
	Value         int
	Stackable     bool
	Unit          string
	EquipmentSlot string
	Sellable      bool
	Bind          string
	QuestItem     bool
	Usable        bool
	Effects       map[string]interface{}
	Source        string
}

type SkillDef struct {
	ID           string
	Name         string
	Description  string
	Rarity       int
	Type         string
	Category     string
	School       string
	Bonuses      map[string]float64
	Requirements map[string]interface{}
}

type NPC struct {
	ID                string
	Name              string
	Title             string
	Description       string
	Faction           string
	Role              string
	Importance        string
	DialogueID        string
	QuestIDs          []string
	InstanceOnly      bool
	SpawnRule         string
	Aggressive        bool
	NonCombat         bool
	IsShop            bool
	ShopInventory     map[string]int
	Skills            map[string]int
	Attributes        Attributes
	WeaknessFlag      string
	WeaknessModifiers map[string]int
	CombatMode        string
}

type DialogueDef struct {
	ID      string           `json:"id"`
	NPCID   string           `json:"npcId"`
	Title   string           `json:"title"`
	Opening string           `json:"opening"`
	Options []DialogueOption `json:"options"`
}

type DialogueOption struct {
	ID              string          `json:"id"`
	Text            string          `json:"text"`
	Response        string          `json:"response"`
	StartQuestID    string          `json:"startQuestId,omitempty"`
	CompleteQuestID string          `json:"completeQuestId,omitempty"`
	SetFlags        map[string]bool `json:"setFlags,omitempty"`
	GiveItems       map[string]int  `json:"giveItems,omitempty"`
	TakeItems       map[string]int  `json:"takeItems,omitempty"`
	TakeMoney       int             `json:"takeMoney,omitempty"`
	GiveMoney       int             `json:"giveMoney,omitempty"`
	Requires        DialogueRequire `json:"requires,omitempty"`
}

type DialogueRequire struct {
	QuestID       string         `json:"questId,omitempty"`
	QuestStatus   string         `json:"questStatus,omitempty"`
	Flag          string         `json:"flag,omitempty"`
	FlagValue     *bool          `json:"flagValue,omitempty"`
	ReputationID  string         `json:"reputationId,omitempty"`
	ReputationMin int            `json:"reputationMin,omitempty"`
	Items         map[string]int `json:"items,omitempty"`
}

type QuestDef struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	GiverNPCID  string      `json:"giverNpcId"`
	TurnInNPCID string      `json:"turnInNpcId"`
	Objectives  []Objective `json:"objectives"`
	Rewards     Rewards     `json:"rewards"`
	Repeatable  bool        `json:"repeatable"`
}

type Objective struct {
	Type        string `json:"type"`
	TargetID    string `json:"targetId"`
	Description string `json:"description"`
	Required    int    `json:"required"`
}

type Rewards struct {
	Experience  int             `json:"experience"`
	Potential   int             `json:"potential"`
	Money       int             `json:"money"`
	Items       map[string]int  `json:"items"`
	Flags       map[string]bool `json:"flags"`
	Reputations map[string]int  `json:"reputations"`
}

type FactionDef struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Visible     bool   `json:"visible"`
	Order       int    `json:"order"`
}

type Exit struct {
	Direction      string
	TargetRoomID   string
	TargetRoomName string
}

type ExitRequirement struct {
	DefeatNPCs           []string
	RequiredItemID       string
	RequiredAmount       int
	ConsumeItem          bool
	RequiredPuzzleStates map[string]string
	Message              string
}

type Room struct {
	ID               string
	WorldID          string
	WorldName        string
	Name             string
	Description      string
	Area             string
	Tags             []string
	SafeZone         bool
	Noise            int
	Function         string
	Actions          []string
	ActionLabels     map[string]string
	Exits            map[string]Exit
	ExitRequirements map[string]ExitRequirement
	NPCIDs           []string
}

type World struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Order       int    `json:"order"`
	Category    string `json:"category"`
	Visible     bool   `json:"visible"`
	SpawnRoomID string
}
