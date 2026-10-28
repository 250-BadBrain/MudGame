package model

// EnhancementStep defines the additive gain per level in each stat's own units.
// critRate is measured in percentage points, not a probability fraction.
func EnhancementStep(key string) (float64, bool) {
	switch key {
	case "attack", "defense", "hitRate", "dodge", "parry", "initiative", "charm":
		return 1, true
	case "maxHp", "maxMp":
		return 5, true
	case "critRate":
		return .1, true
	default:
		return 0, false
	}
}
