package manager

import (
	"mudgame/server/internal/model"
)

// NPC意图调度与释放（实现规范§8/提案§7）：
// 100ms ticker仅处理合法NPC意图，不执行玩家排队招式、普攻、dot/hot。
// AI在开战至少一个普攻tick后按配置优先级选招并提前2秒公开意图；
// 到公开时刻按同一结算引擎即时施放并开启双CD；预告与出招CD期间普攻继续。

// npcCombatState NPC的内存双CD状态：实体共享，不持久化（重启作废）。
type npcCombatState struct {
	Action model.CooldownWindow
	Skills map[string]model.CooldownWindow
}

// initNPCCombatStateNoLock 确保NPC战斗状态容器就绪。
func (r *RuntimeManager) initNPCCombatStateNoLock() {
	if r.npcCombatStates == nil {
		r.npcCombatStates = map[string]*npcCombatState{}
	}
}

// npcCombatStateNoLock 取或建NPC的战斗状态。
func (r *RuntimeManager) npcCombatStateNoLock(key ActorKey) *npcCombatState {
	r.initNPCCombatStateNoLock()
	state, ok := r.npcCombatStates[key.String()]
	if !ok {
		state = &npcCombatState{Skills: map[string]model.CooldownWindow{}}
		r.npcCombatStates[key.String()] = state
	}
	return state
}

// npcMoveDefNoLock 按运行时实体解析模板combatArts中的招式定义与等级。
// 教学Skills完全不变；level只算招式数值，不叠加武功被动。
func (r *RuntimeManager) npcMoveDefNoLock(entityID, moveID string) (model.CombatMoveDef, int, bool) {
	var arts []model.NPCCombatArt
	if npc, ok := r.dungeonNPCByIDNoLock(entityID); ok && npc != nil {
		if tpl, ok := fileNpcs[npc.TemplateID]; ok {
			arts = tpl.CombatArts
		}
	} else if tpl, ok := fileNpcs[entityID]; ok {
		arts = tpl.CombatArts
	}
	for _, art := range arts {
		for _, mid := range art.MoveIDs {
			if mid != moveID {
				continue
			}
			if skill, ok := fileSkills[art.ArtID]; ok {
				for _, m := range skill.Moves {
					if m.ID == moveID {
						return m, art.Level, true
					}
				}
			}
		}
	}
	return model.CombatMoveDef{}, 0, false
}

// scheduleNPCIntentsNoLock 战斗tick后调用：就绪且有MP时为试点NPC公开下一招意图。
// 开战AutoRounds≥1才出手；意图公开后不换招不换目标；一人一场只有一个意图。
func (r *RuntimeManager) scheduleNPCIntentsNoLock(players map[string]*model.Player) {
	r.initNPCCombatStateNoLock()
	now := combatNowUnix(r)
	for key := range r.npcCombatStates {
		actorKey := parseActorKey(key)
		if _, busy := r.npcIntents[key]; busy {
			continue
		}
		entity := r.entityByKeyNoLock(actorKey, players)
		if entity == nil || !entity.canAct() {
			continue
		}
		binding := r.currentBindingNoLock(actorKey)
		if binding == nil || binding.AutoRounds < 1 {
			continue
		}
		state := r.npcCombatStateNoLock(actorKey)
		if now < state.Action.ReadyAt {
			continue
		}
		art := r.npcCombatArtNoLock(actorKey)
		if art == nil {
			continue
		}
		ready := func(moveID string) bool {
			skillReady, ok := state.Skills[moveID]
			return !(ok && now < skillReady.ReadyAt)
		}
		hpRatio := float64(*entity.hp) / float64(entity.maxHP)
		preferDefense := art.LowHPPercent > 0 && hpRatio < art.LowHPPercent
		pick := ""
		if preferDefense { // 低血优先非攻击招式（宋缺归鞘/庞斑幻身）
			for _, moveID := range art.MoveIDs {
				def, _, ok := r.npcMoveDefNoLock(actorKey.EntityID, moveID)
				if ok && def.Kind != "attack" && ready(moveID) && (entity.mp == nil || *entity.mp >= def.MPCost) {
					pick = moveID
					break
				}
			}
		}
		if pick == "" {
			for _, moveID := range art.MoveIDs {
				def, _, ok := r.npcMoveDefNoLock(actorKey.EntityID, moveID)
				if ok && ready(moveID) && (entity.mp == nil || *entity.mp >= def.MPCost) {
					pick = moveID
					break
				}
			}
		}
		if pick == "" {
			continue
		}
		def, _, _ := r.npcMoveDefNoLock(actorKey.EntityID, pick)
		r.npcIntents[key] = &npcIntentState{
			Kind: actorKey.Kind, NPCID: actorKey.EntityID,
			MoveID: pick, MoveName: def.Name,
			TargetID:    binding.Target.EntityID,
			AnnouncedAt: now, ReleaseAt: now + 2000,
		}
	}
}

// npcCombatArtNoLock 取NPC模板的战斗招式配置。
func (r *RuntimeManager) npcCombatArtNoLock(key ActorKey) *model.NPCCombatArt {
	var tplID string
	if npc, ok := r.dungeonNPCByIDNoLock(key.EntityID); ok && npc != nil {
		tplID = npc.TemplateID
	} else if _, ok := fileNpcs[key.EntityID]; ok {
		tplID = key.EntityID
	} else {
		return nil
	}
	tpl, ok := fileNpcs[tplID]
	if !ok || len(tpl.CombatArts) == 0 {
		return nil
	}
	return &tpl.CombatArts[0]
}

// ProcessCombatMovesTick 由100ms调度触发：仅释放到期的NPC公开意图。
func (r *RuntimeManager) ProcessCombatMovesTick() (model.CombatTickResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := model.CombatTickResult{}
	players, err := r.loadPlayers()
	if err != nil {
		return result, err
	}
	now := combatNowUnix(r)
	changed := false
	for key, intent := range r.npcIntents {
		if intent == nil || now < intent.ReleaseAt {
			continue
		}
		if r.releaseNPCIntentNoLock(parseActorKey(key), intent, players, &result, now) {
			changed = true
		}
	}
	if changed {
		if err := r.savePlayers(players); err != nil {
			return result, err
		}
	}
	return result, nil
}

// releaseNPCIntentNoLock 到公开时刻即时结算：MP实扣、双CD开启，普攻照常。
// 目标失效、认负/死亡、MP不足或CD不满足时取消，不扣费不启动CD。
func (r *RuntimeManager) releaseNPCIntentNoLock(key ActorKey, intent *npcIntentState, players map[string]*model.Player, result *model.CombatTickResult, now int64) bool {
	defer delete(r.npcIntents, key.String())
	entity := r.entityByKeyNoLock(key, players)
	if entity == nil || !entity.canAct() || entity.mp == nil {
		return false
	}
	def, level, ok := r.npcMoveDefNoLock(key.EntityID, intent.MoveID)
	if !ok {
		return false
	}
	if *entity.mp < def.MPCost {
		return false
	}
	state := r.npcCombatStateNoLock(key)
	if now < state.Action.ReadyAt {
		return false
	}
	if skillReady, ok := state.Skills[intent.MoveID]; ok && now < skillReady.ReadyAt {
		return false
	}
	target := r.entityByKeyNoLock(ActorKey{Kind: "player", EntityID: intent.TargetID}, players)
	if target == nil || !target.canAct() {
		return false
	}
	binding := r.currentBindingNoLock(key)
	if binding == nil || binding.Target.EntityID != intent.TargetID {
		return false
	}
	roomID := binding.RoomID

	*entity.mp -= def.MPCost
	r.applyMoveCastEffectsNoLock(def, level, key, target.key, binding.ID, def.MPCost)
	result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: entity.name + "使出了" + def.Name + "！"})
	if def.Kind == "attack" {
		out := r.resolveCombatActionNoLock(combatAction{
			Kind: actionMove, MoveID: def.ID, MoveName: def.Name,
			MoveDef: def, MoveLevel: level, BattleID: binding.ID,
			Actor: entity, Target: target,
			HitCount: def.HitCount, Power: def.EffectivePower(level),
			HitBonus: def.EffectiveHitBonus(level), ArmorIgnore: def.EffectiveArmorIgnore(level),
			ParryMult: def.ParryMultiplier(level),
			Now:       now, RoomID: roomID,
		})
		result.Messages = append(result.Messages, out.Messages...)
		// NPC招式击败玩家：与tick终止规则一致
		if *target.hp <= target.floor() {
			if p, ok := players[intent.TargetID]; ok && p.State == "COMBAT" {
				p.State = "DEAD"
				p.TargetID = ""
				p.CombatType = ""
				result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: p.CharacterName + "大叫一声，倒在地上，死了！"})
				r.endCombatBindingNoLock(key)
				r.endCombatBindingNoLock(target.key)
				r.bumpCombatRevisionNoLock(p.ID)
			}
		}
	}
	// NPC双CD：实体共享；不重置他人CD
	speed := r.effectiveCastSpeedNoLock(key, entity.attrs.EffectiveActionSpeed)
	actionDur := model.ActionDurationMs(def.ActionMs, speed)
	state.Action = model.CooldownWindow{StartedAt: now, ReadyAt: now + actionDur}
	state.Skills[intent.MoveID] = model.CooldownWindow{StartedAt: now, ReadyAt: now + int64(def.CooldownSec)*1000}
	return true
}

// cancelNPCIntentNoLock 取消意图（目标失效/条件失效等），不扣费不启动CD。
func (r *RuntimeManager) cancelNPCIntentNoLock(key ActorKey) {
	delete(r.npcIntents, key.String())
}
