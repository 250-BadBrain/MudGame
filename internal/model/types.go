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
	SkillLearningProgress map[string]int
	TargetID              string
	CombatType            string
	Money                 int
	Attributes            Attributes
	Skills                map[string]int
	Backpack              map[string]int
	Equipment             map[string]string
	QuestStates           map[string]QuestState
	Flags                 map[string]bool
	Reputations           map[string]int
	Online                bool
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
	ID            string
	Name          string
	Title         string
	Description   string
	Faction       string
	Role          string
	Importance    string
	DialogueID    string
	QuestIDs      []string
	InstanceOnly  bool
	SpawnRule     string
	Aggressive    bool
	IsShop        bool
	ShopInventory map[string]int
	Skills        map[string]int
	Attributes    Attributes
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
	Requires        DialogueRequire `json:"requires,omitempty"`
}

type DialogueRequire struct {
	QuestID       string `json:"questId,omitempty"`
	QuestStatus   string `json:"questStatus,omitempty"`
	Flag          string `json:"flag,omitempty"`
	FlagValue     *bool  `json:"flagValue,omitempty"`
	ReputationID  string `json:"reputationId,omitempty"`
	ReputationMin int    `json:"reputationMin,omitempty"`
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

type Room struct {
	ID          string
	WorldID     string
	WorldName   string
	Name        string
	Description string
	Area        string
	Tags        []string
	SafeZone    bool
	Function    string
	Actions     []string
	Exits       map[string]Exit
	NPCIDs      []string
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
