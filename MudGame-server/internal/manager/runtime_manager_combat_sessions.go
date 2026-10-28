package manager

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// 主动招式的战斗绑定与临时效果（实现规范§3）。
// 全部状态由 RuntimeManager.mu 保护；临时效果只在服务端内存，
// 战斗结束/实例销毁/重启一起清理正负效果，两种CD照常保留。

// ActorKey 标识一个战斗参与者：公共NPC用房间运行时实体，
// 副本NPC必须带实例ID和实例实体ID，不能把模板ID当运行时实体。
type ActorKey struct {
	Kind       string // player / static_npc / dungeon_npc
	InstanceID string
	EntityID   string
}

func (k ActorKey) String() string { return k.Kind + "|" + k.InstanceID + "|" + k.EntityID }

// IsZero 报告是否为空键。
func (k ActorKey) IsZero() bool { return k.EntityID == "" && k.InstanceID == "" && k.Kind == "" }

// RecentCombatEvent 最近4秒窗口的战斗事件，供踏浪穿隙/踏雪折影等条件判定。
type RecentCombatEvent struct {
	BattleID     string
	Source       ActorKey
	ActionKind   string // auto / move / counter / dot / self
	Dodged       bool
	ActualDamage int
	At           int64
}

// CombatEffect 一条已生效的临时效果（自用增益、敌向削弱、姿态反击、持续效果）。
type CombatEffect struct {
	ID         string
	BattleID   string
	MoveID     string
	Owner      ActorKey // 效果作用对象：自用=施放者，敌向=目标
	Source     ActorKey
	Op         string
	Stat       string
	Scope      string
	Group      string
	Value      float64 // 施放时按S成长后的幅度快照
	Cap        float64
	Power      float64
	Charges    int
	Ticks      int
	TickBudget []int // 持续效果按tick的整数预算（施放时快照拆分）
	DurationMs int
	CreatedAt  int64
	ExpiresAt  int64 // 0=不限时；Charges与时间取先到者
	Tags       []string
	PaidMP     int // paid_mp_refund 的实付费用账本
	RefundedMP int
}

// combatPendingHeal 镜湖倒影的待回疗：下一tick核存活后再恢复。
type combatPendingHeal struct {
	BattleID string
	Owner    ActorKey
	Amount   int
	At       int64
}

// combatBinding 一次合法交战的绑定：目标改变必须生成新的不可猜BattleID。
type combatBinding struct {
	ID          string
	Self        ActorKey
	Target      ActorKey
	RoomID      string
	Mode        string // KILL / SPAR
	StartedAt   int64
	AutoRounds  int
	Recent      []RecentCombatEvent
	PendingHeal *combatPendingHeal
}

// npcIntentState NPC公开预告的招式意图：到公开时刻仍按即时技能结算。
type npcIntentState struct {
	Kind        string // static_npc / dungeon_npc
	NPCID       string
	MoveID      string
	MoveName    string
	TargetID    string
	AnnouncedAt int64
	ReleaseAt   int64
}

// initCombatStateNoLock 初始化战斗会话相关字段（构造与测试注入共用）。
func (r *RuntimeManager) initCombatStateNoLock() {
	if r.combatBindings == nil {
		r.combatBindings = map[string]*combatBinding{}
	}
	if r.actorBattles == nil {
		r.actorBattles = map[string]string{}
	}
	if r.combatEffects == nil {
		r.combatEffects = map[string][]*CombatEffect{}
	}
	if r.npcIntents == nil {
		r.npcIntents = map[string]*npcIntentState{}
	}
	if r.combatRevision == nil {
		r.combatRevision = map[string]int64{}
	}
	if r.combatNow == nil {
		r.combatNow = time.Now
	}
	if r.combatEpoch == "" {
		r.combatEpoch = newBattleToken()
	}
}

// newBattleToken 生成不可猜的战斗令牌。
func newBattleToken() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("b%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func combatNowUnix(r *RuntimeManager) int64 { return r.combatNow().UnixMilli() }

// beginCombatBindingNoLock 建立或重建绑定；目标改变即换新BattleID并清旧目标相关状态（保留角色CD）。
func (r *RuntimeManager) beginCombatBindingNoLock(self, target ActorKey, roomID, mode string) *combatBinding {
	r.initCombatStateNoLock()
	now := combatNowUnix(r)
	oldID := r.actorBattles[self.String()]
	if oldID != "" {
		if old := r.combatBindings[oldID]; old != nil && old.Target.String() == target.String() {
			old.RoomID, old.Mode = roomID, mode
			return old
		}
		r.endCombatBindingNoLock(self)
	}
	b := &combatBinding{ID: newBattleToken(), Self: self, Target: target, RoomID: roomID, Mode: mode, StartedAt: now}
	r.combatBindings[b.ID] = b
	r.actorBattles[self.String()] = b.ID
	return b
}

// endCombatBindingNoLock 结束本人的当前绑定：清本场全部正负效果、最近事件与待回疗；CD保留。
func (r *RuntimeManager) endCombatBindingNoLock(self ActorKey) {
	r.initCombatStateNoLock()
	id := r.actorBattles[self.String()]
	if id == "" {
		return
	}
	delete(r.actorBattles, self.String())
	delete(r.combatBindings, id)
	r.clearCombatEffectsForBattleNoLock(id)
}

// clearCombatEffectsForBattleNoLock 按战斗清理效果：遍历所有owner列表移除该场的正负效果。
func (r *RuntimeManager) clearCombatEffectsForBattleNoLock(battleID string) {
	if battleID == "" {
		return
	}
	for owner, list := range r.combatEffects {
		kept := list[:0]
		for _, ef := range list {
			if ef.BattleID != battleID {
				kept = append(kept, ef)
			}
		}
		if len(kept) == 0 {
			delete(r.combatEffects, owner)
		} else {
			r.combatEffects[owner] = kept
		}
	}
}

// resetCombatStateAfterRestartNoLock 重启恢复：COMBAT归IDLE（DEAD保持）、
// 清除TargetID/CombatType，旧BattleID与临时效果作废；不回血回蓝、不复活。
func (r *RuntimeManager) resetCombatStateAfterRestartNoLock() {
	players, err := r.loadPlayers()
	if err != nil {
		return
	}
	changed := false
	for _, p := range players {
		if p.State == "COMBAT" {
			p.State = "IDLE"
			p.TargetID = ""
			p.CombatType = ""
			changed = true
		}
	}
	if changed {
		_ = r.savePlayers(players)
	}
	r.initCombatStateNoLock()
	r.clearAllCombatStateNoLock()
}

// clearAllCombatStateNoLock 重启恢复：旧BattleID全部作废，清空临时效果与NPC意图。
// 保留HP/MP、序号与真实时间CD（见实现规范§3/§4）。
func (r *RuntimeManager) clearAllCombatStateNoLock() {
	r.combatBindings = map[string]*combatBinding{}
	r.actorBattles = map[string]string{}
	r.combatEffects = map[string][]*CombatEffect{}
	r.npcIntents = map[string]*npcIntentState{}
	r.combatEpoch = newBattleToken()
}

// currentBindingNoLock 返回本人当前绑定；无绑定返回nil。
func (r *RuntimeManager) currentBindingNoLock(self ActorKey) *combatBinding {
	id := r.actorBattles[self.String()]
	if id == "" {
		return nil
	}
	return r.combatBindings[id]
}

// addCombatEffectNoLock 落一条效果：同名（同MoveID+Op+Stat，同一owner视角）不叠层只保留较强；
// 同组取强不叠层；等幅保留旧项不刷新期限/次数；较弱新效果不排队等待。
func (r *RuntimeManager) addCombatEffectNoLock(ef *CombatEffect) {
	r.initCombatStateNoLock()
	list := r.combatEffects[ef.Owner.String()]
	for _, old := range list {
		sameName := ef.MoveID != "" && old.MoveID == ef.MoveID && old.Op == ef.Op && old.Stat == ef.Stat
		sameGroup := ef.Group != "" && old.Group == ef.Group
		if !sameName && !sameGroup {
			continue
		}
		if magnitude(ef) > magnitude(old) {
			*old = *ef // 较强新效果覆盖同名/同组弱项
		}
		// 等幅或较弱：保留旧项，不刷新期限与次数
		return
	}
	r.combatEffects[ef.Owner.String()] = append(list, ef)
}

func magnitude(ef *CombatEffect) float64 {
	m := ef.Value
	if ef.Cap > m {
		m = ef.Cap
	}
	if ef.Power > m {
		m = ef.Power
	}
	if m < 0 {
		m = -m
	}
	return m
}

// activeEffectsNoLock 返回owner名下仍在期限内的效果：次数型需Charges>0，时限型看ExpiresAt。
func (r *RuntimeManager) activeEffectsNoLock(owner ActorKey) []*CombatEffect {
	list := r.combatEffects[owner.String()]
	if len(list) == 0 {
		return nil
	}
	now := combatNowUnix(r)
	out := make([]*CombatEffect, 0, len(list))
	for _, ef := range list {
		if ef.ExpiresAt > 0 && ef.ExpiresAt <= now {
			continue
		}
		if ef.Charges <= 0 && ef.DurationMs <= 0 {
			continue
		}
		out = append(out, ef)
	}
	return out
}

// consumeEffectChargeNoLock 消耗一次次数；耗尽即移除。
func (r *RuntimeManager) consumeEffectChargeNoLock(ef *CombatEffect) {
	ef.Charges--
	if ef.Charges <= 0 {
		r.removeCombatEffectNoLock(ef)
	}
}

// removeCombatEffectNoLock 按指针移除一条效果。
func (r *RuntimeManager) removeCombatEffectNoLock(ef *CombatEffect) {
	list := r.combatEffects[ef.Owner.String()]
	for i, item := range list {
		if item == ef {
			r.combatEffects[ef.Owner.String()] = append(list[:i], list[i+1:]...)
			return
		}
	}
}

// recordCombatEventNoLock 记录最近事件；只保留最近4秒与有伤害/落空意义的事件。
func (r *RuntimeManager) recordCombatEventNoLock(b *combatBinding, ev RecentCombatEvent) {
	if b == nil {
		return
	}
	b.Recent = append(b.Recent, ev)
	cutoff := combatNowUnix(r) - 4000
	kept := b.Recent[:0]
	for _, item := range b.Recent {
		if item.At >= cutoff {
			kept = append(kept, item)
		}
	}
	b.Recent = kept
}

// recentEventNoLock 查找满足条件窗口的最近一笔事件（0≤经过<4000，同一战斗同一来源）。
func (b *combatBinding) recentEventNoLock(now int64, from ActorKey, kind string, needDamage bool) *RecentCombatEvent {
	for i := len(b.Recent) - 1; i >= 0; i-- {
		ev := b.Recent[i]
		if ev.BattleID != b.ID || ev.Source.String() != from.String() {
			continue
		}
		if kind != "" && ev.ActionKind != kind {
			continue
		}
		if needDamage && ev.ActualDamage <= 0 {
			continue
		}
		if ev.At > now || now-ev.At >= 4000 {
			continue // 窗口：0≤经过时间<4000毫秒
		}
		return &b.Recent[i]
	}
	return nil
}

// consumeRecentEventNoLock 成功使用后消耗对应记录，防止同一事件重复兑换。
func (b *combatBinding) consumeRecentEventNoLock(ev *RecentCombatEvent) {
	for i := range b.Recent {
		if &b.Recent[i] == ev {
			b.Recent = append(b.Recent[:i], b.Recent[i+1:]...)
			return
		}
	}
}

// hasImmunityNoLock 报告owner是否免疫指定标签的新增效果。
func (r *RuntimeManager) hasImmunityNoLock(owner ActorKey, tag string) bool {
	for _, ef := range r.activeEffectsNoLock(owner) {
		if ef.Op != "immunity" {
			continue
		}
		for _, t := range ef.Tags {
			if t == tag {
				return true
			}
		}
	}
	return false
}

// bumpCombatRevisionNoLock 玩家招式视图变化时递增修订号。
func (r *RuntimeManager) bumpCombatRevisionNoLock(playerID string) {
	if playerID == "" {
		return
	}
	r.combatRevision[playerID]++
}
