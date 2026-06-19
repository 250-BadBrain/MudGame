package model

type User struct {
	Username           string
	PasswordHash       string
	CharacterIDs       []string
	ReconnectTokenHash string
}

type Attributes struct {
	Experience       int `json:"experience"`
	Potential        int `json:"potential"`
	CurrentHP        int `json:"currentHp"`
	MaxHP            int `json:"maxHp"`
	CurrentMP        int `json:"currentMp"`
	TrainedMaxMP     int `json:"trainedMaxMp"`
	MaxMP            int `json:"maxMp"`
	Strength         int `json:"strength"`
	Constitution     int `json:"constitution"`
	Agility          int `json:"agility"`
	Intelligence     int `json:"intelligence"`
	Appearance       int `json:"appearance"`
	Luck             int `json:"luck"`
	Attack           int `json:"attack"`
	Defense          int `json:"defense"`
	HitRate          int `json:"hitRate"`
	Dodge            int `json:"dodge"`
	Parry            int `json:"parry"`
	CritRate         int `json:"critRate"`
	AttackSpeed      int `json:"attackSpeed"`
	PostStrength     int `json:"postStrength"`
	PostConstitution int `json:"postConstitution"`
	PostAgility      int `json:"postAgility"`
	PostIntelligence int `json:"postIntelligence"`
}

type Player struct {
	ID                 string
	UserID             string
	CharacterName      string
	RoomID             string
	State              string
	StateTickCountdown int
	LearningSkillID    string
	TargetID           string
	CombatType         string
	Money              int
	Attributes         Attributes
	Skills             map[string]int
	Backpack           map[string]int
	Equipment          map[string]string
	Online             bool
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
