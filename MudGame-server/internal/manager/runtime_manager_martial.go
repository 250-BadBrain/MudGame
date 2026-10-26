package manager

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"mudgame/server/internal/exception"
	"mudgame/server/internal/model"
)

func cloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func currentWeaponType(p *model.Player) string {
	id := p.Equipment["WEAPON"]
	if id == "" {
		return "unarmed"
	}
	if def, ok := fileItems[id]; ok && def.WeaponType != "" {
		return def.WeaponType
	}
	return "unknown"
}

func skillRequirementError(p *model.Player, def model.SkillDef) error {
	if def.Type == "BASIC" {
		return nil
	}
	if base := skillPrerequisiteID(def); base != "" && p.Skills[base] < skillPrerequisiteLevel(def) {
		return fmt.Errorf("需要%s达到%d级", fileSkills[base].Name, skillPrerequisiteLevel(def))
	}
	if level := toIntEffect(def.Requirements["literacy"]); p.Skills["basic_literacy"] < level {
		return fmt.Errorf("需要读书识文达到%d级", level)
	}
	if def.SourceDungeon != "" && !p.Flags["dungeon_cleared_"+def.SourceDungeon] {
		return fmt.Errorf("尚未取得指定传承：%s", def.Source)
	}
	if def.TeacherID != "" && !p.Flags["martial_taught_"+def.ID] {
		return fmt.Errorf("请先向%s请教入门", fileNpcs[def.TeacherID].Name)
	}
	return nil
}

func skillTrainingCap(p *model.Player, def model.SkillDef) int {
	cap := model.SkillLevelCapForExperience(p.Attributes.Experience)
	if def.Type == "ADVANCED" {
		cap = min(cap, p.Skills[skillPrerequisiteID(def)])
	}
	return cap
}

func skillActiveReason(p *model.Player, def model.SkillDef) string {
	if p.Skills[def.ID] < 1 {
		return "尚未学会"
	}
	if err := skillRequirementError(p, def); err != nil {
		return err.Error()
	}
	if def.Type == "ADVANCED" && p.ActiveArts[def.Slot] != def.ID {
		return "尚未装备"
	}
	if def.Slot == "combat" && currentWeaponType(p) != def.WeaponType {
		return "当前兵器与该流派不符"
	}
	return ""
}

func (r *RuntimeManager) PracticeSkill(playerID, skillID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	players, err := r.loadPlayers()
	if err != nil {
		return err
	}
	p, ok := players[playerID]
	if !ok {
		return exception.ErrPlayerNotFound
	}
	if err := startPracticeNoLock(p, skillID); err != nil {
		return err
	}
	return r.savePlayers(players)
}

func startPracticeNoLock(p *model.Player, skillID string) error {
	if p.State != "IDLE" {
		return errors.New("busy")
	}
	def, ok := fileSkills[skillID]
	if !ok || p.Skills[skillID] < 1 {
		return errors.New("请先取得这门武功的入门传承")
	}
	if room, exists := fileRooms[p.RoomID]; !exists || !room.SafeZone || strings.HasPrefix(p.RoomID, "dg_") {
		return errors.New("请在安全的公共房间修炼")
	}
	if err := skillRequirementError(p, def); err != nil {
		return err
	}
	if p.Skills[skillID] >= skillTrainingCap(p, def) {
		return errors.New("已达到当前经验或基本功允许的修炼上限")
	}
	if p.Attributes.Potential <= 0 {
		return errors.New("potential_not_enough")
	}
	p.State, p.LearningSkillID = "LEARNING", skillID
	p.LearningTeacherID, p.LearningTeacherLevel = "", 0
	p.StateTickCountdown = stateSettlementSeconds
	if p.SkillLearningProgress == nil {
		p.SkillLearningProgress = map[string]int{}
	}
	return nil
}

// Settle only the actual potential used; unused per-tick budget never becomes currency.
func settlePracticeNoLock(p *model.Player) string {
	id := p.LearningSkillID
	def, ok := fileSkills[id]
	if !ok {
		stopLearningNoLock(p)
		return "修炼目标已失效，停止了修炼。"
	}
	if err := skillRequirementError(p, def); err != nil {
		stopLearningNoLock(p)
		return err.Error() + "，停止了修炼。"
	}
	room, safe := fileRooms[p.RoomID]
	if !safe || !room.SafeZone {
		stopLearningNoLock(p)
		return "离开了安全房间，停止修炼。"
	}
	cap := skillTrainingCap(p, def)
	if p.Skills[id] >= cap {
		stopLearningNoLock(p)
		return "已达到当前经验或基本功上限，停止修炼。"
	}
	if p.SkillLearningProgress == nil {
		p.SkillLearningProgress = map[string]int{}
	}
	budget := model.LearningPotentialPerTick + effectiveAttributesForPlayerNoLock(p).LearningInvestBonus
	before := p.Skills[id]
	for budget > 0 && p.Attributes.Potential > 0 && p.Skills[id] < cap {
		cost := model.SkillPotentialForTargetLevel(p.Skills[id] + 1)
		progress := p.SkillLearningProgress[id]
		if progress < 0 || progress >= cost {
			stopLearningNoLock(p)
			return "修炼进度异常，请联系维护者。"
		}
		spent := min(budget, p.Attributes.Potential, cost-progress)
		p.Attributes.Potential -= spent
		budget -= spent
		p.SkillLearningProgress[id] += spent
		if p.SkillLearningProgress[id] == cost {
			p.Skills[id]++
			p.SkillLearningProgress[id] = 0
		}
	}
	message := "你参悟了" + def.Name + "。"
	if p.Skills[id] > before {
		message = fmt.Sprintf("你的%s从%d级修炼至%d级。", def.Name, before, p.Skills[id])
	}
	if p.Skills[id] >= cap || p.Attributes.Potential == 0 {
		stopLearningNoLock(p)
		message += "已达到修炼上限或潜能耗尽，停止修炼。"
	}
	return message
}

func (r *RuntimeManager) UnequipSkill(playerID, slot string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	players, err := r.loadPlayers()
	if err != nil {
		return "", err
	}
	p, ok := players[playerID]
	if !ok {
		return "", exception.ErrPlayerNotFound
	}
	if p.State != "IDLE" {
		return "", errors.New("busy")
	}
	if p.ActiveArts[slot] == "" {
		return "", errors.New("该位置没有装备武学")
	}
	delete(p.ActiveArts, slot)
	clampPlayerCurrentResources(p)
	return "已卸下该位置的武学。", r.savePlayers(players)
}

func clampPlayerCurrentResources(p *model.Player) {
	eff := effectiveAttributesForPlayerNoLock(p)
	p.Attributes.Resources.HP = max(0, min(p.Attributes.Resources.HP, eff.MaxHP))
	p.Attributes.Resources.MP = max(0, min(p.Attributes.Resources.MP, eff.TrainedMaxMP))
}

func leveledValue(value float64, level int) int { return int(math.Floor(value * float64(level) / 100)) }

func requirementHint(p *model.Player, def model.SkillDef) string {
	if err := skillRequirementError(p, def); err != nil {
		return err.Error()
	}
	return "已满足学习条件"
}

func skillRequirementText(def model.SkillDef) string {
	if def.Type == "BASIC" {
		return "向世界老师免费请教入门"
	}
	parts := []string{}
	if base := skillPrerequisiteID(def); base != "" {
		parts = append(parts, fmt.Sprintf("%s≥%d级", fileSkills[base].Name, skillPrerequisiteLevel(def)))
	}
	if lv := toIntEffect(def.Requirements["literacy"]); lv > 0 {
		parts = append(parts, fmt.Sprintf("读书识文≥%d级", lv))
	}
	if def.SourceDungeon != "" {
		parts = append(parts, "指定副本100%通关并参悟完整秘本")
	} else {
		parts = append(parts, "指定老师入门")
	}
	return strings.Join(parts, "；")
}
