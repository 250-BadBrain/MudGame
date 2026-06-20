package model

import "math"

const (
	InitialSkillLevelCap       = 20
	SkillLevelCapStep          = 10
	SkillExperienceCurveFactor = 250
	LearningPotentialPerTick   = 10
)

var skillPotentialCostsByTwentyLevels = []int{
	1, 2, 4, 7, 11,
	16, 23, 32, 44, 60,
	80, 105, 135, 170, 210,
}

func SkillLevelCapForExperience(experience int) int {
	if experience <= 0 {
		return InitialSkillLevelCap
	}
	units := float64(experience) / SkillExperienceCurveFactor
	root := int64(math.Cbrt(units))
	exp := int64(experience)
	for skillExperienceThreshold(root+1) <= exp {
		root++
	}
	for root > 0 && skillExperienceThreshold(root) > exp {
		root--
	}
	return InitialSkillLevelCap + SkillLevelCapStep*int(root)
}

func SkillPotentialForTargetLevel(targetLevel int) int {
	if targetLevel <= 0 {
		return 0
	}
	band := (targetLevel - 1) / 20
	if band < len(skillPotentialCostsByTwentyLevels) {
		return skillPotentialCostsByTwentyLevels[band]
	}
	cost := math.Ceil(math.Pow(float64(targetLevel), 3) / 128000)
	maxInt := int(^uint(0) >> 1)
	if cost >= float64(maxInt) {
		return maxInt
	}
	return max(1, int(cost))
}

func skillExperienceThreshold(root int64) int64 {
	if root <= 0 {
		return 0
	}
	return SkillExperienceCurveFactor * root * root * root
}
